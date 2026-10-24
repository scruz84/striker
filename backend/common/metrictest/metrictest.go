// Package metrictest lets tests read the OpenTelemetry instruments declared in common/telemetry.go.
//
// Those instruments are package vars created from otel's global Meter, which delegates to the
// first MeterProvider installed in a process and ignores later ones. So a test binary must install
// exactly one provider, once, before it records anything: Install does that and hands every caller
// the same ManualReader. The reader is cumulative and shared by every test in the binary, so read a
// value before and after the code under test and compare the difference (see Sum).
package metrictest

import (
	"context"
	"sync"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

var (
	once   sync.Once
	reader *sdkmetric.ManualReader
)

// Install makes the global MeterProvider export to a ManualReader (once per test binary) and
// returns that reader.
func Install() *sdkmetric.ManualReader {
	once.Do(func() {
		reader = sdkmetric.NewManualReader()
		otel.SetMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
	})
	return reader
}

// Collect returns every metric the provider currently holds, by instrument name.
func Collect(t testing.TB) map[string]metricdata.Metrics {
	t.Helper()
	var rm metricdata.ResourceMetrics
	// A failing gauge callback (one registered by another test against a stub database) is
	// reported by Collect but does not stop the other instruments from being gathered.
	if err := Install().Collect(context.Background(), &rm); err != nil {
		t.Logf("collecting metrics: %s", err)
	}
	byName := map[string]metricdata.Metrics{}
	for _, scope := range rm.ScopeMetrics {
		for _, m := range scope.Metrics {
			byName[m.Name] = m
		}
	}
	return byName
}

func matches(set attribute.Set, want []attribute.KeyValue) bool {
	for _, kv := range want {
		v, ok := set.Value(kv.Key)
		if !ok || v != kv.Value {
			return false
		}
	}
	return true
}

// Sum adds up the value of every point of the named counter, up/down counter or gauge whose
// attributes include all of attrs (none: every point). A metric that has not recorded anything yet
// sums to 0.
func Sum(t testing.TB, name string, attrs ...attribute.KeyValue) float64 {
	t.Helper()
	m, ok := Collect(t)[name]
	if !ok {
		return 0
	}
	var total float64
	switch data := m.Data.(type) {
	case metricdata.Sum[int64]:
		for _, p := range data.DataPoints {
			if matches(p.Attributes, attrs) {
				total += float64(p.Value)
			}
		}
	case metricdata.Sum[float64]:
		for _, p := range data.DataPoints {
			if matches(p.Attributes, attrs) {
				total += p.Value
			}
		}
	case metricdata.Gauge[int64]:
		for _, p := range data.DataPoints {
			if matches(p.Attributes, attrs) {
				total += float64(p.Value)
			}
		}
	case metricdata.Gauge[float64]:
		for _, p := range data.DataPoints {
			if matches(p.Attributes, attrs) {
				total += p.Value
			}
		}
	default:
		t.Fatalf("metric %q has data of type %T, not a sum or gauge", name, m.Data)
	}
	return total
}

// Points counts the data points of the named metric whose attributes include all of attrs: for a
// histogram, the number of attribute sets that recorded at least one observation.
func Points(t testing.TB, name string, attrs ...attribute.KeyValue) int {
	t.Helper()
	m, ok := Collect(t)[name]
	if !ok {
		return 0
	}
	count := 0
	switch data := m.Data.(type) {
	case metricdata.Histogram[float64]:
		for _, p := range data.DataPoints {
			if matches(p.Attributes, attrs) {
				count++
			}
		}
	case metricdata.Sum[int64]:
		for _, p := range data.DataPoints {
			if matches(p.Attributes, attrs) {
				count++
			}
		}
	case metricdata.Gauge[int64]:
		for _, p := range data.DataPoints {
			if matches(p.Attributes, attrs) {
				count++
			}
		}
	case metricdata.Gauge[float64]:
		for _, p := range data.DataPoints {
			if matches(p.Attributes, attrs) {
				count++
			}
		}
	default:
		t.Fatalf("metric %q has data of type %T", name, m.Data)
	}
	return count
}

// HistogramCount adds up the observation counts of the named histogram's points that include all
// of attrs.
func HistogramCount(t testing.TB, name string, attrs ...attribute.KeyValue) uint64 {
	t.Helper()
	m, ok := Collect(t)[name]
	if !ok {
		return 0
	}
	data, ok := m.Data.(metricdata.Histogram[float64])
	if !ok {
		t.Fatalf("metric %q has data of type %T, not a float histogram", name, m.Data)
	}
	var total uint64
	for _, p := range data.DataPoints {
		if matches(p.Attributes, attrs) {
			total += p.Count
		}
	}
	return total
}
