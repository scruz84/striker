package common

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const dashboardPath = "../resources/grafana/striker-dashboard.json"

// The rows the observability spec requires, in order.
var dashboardRows = []string{
	"Service health", "Engine health", "Simulated world", "Economy", "Economy flows",
	"Transfer & loan market", "Community", "Live & delivery", "Per node",
}

type dashboardPanel struct {
	Id      int    `json:"id"`
	Type    string `json:"type"`
	Title   string `json:"title"`
	GridPos struct {
		X, Y, W, H int
	} `json:"gridPos"`
	Targets []struct {
		Expr string `json:"expr"`
	} `json:"targets"`
}

type dashboardFile struct {
	Inputs []struct {
		Name string `json:"name"`
	} `json:"__inputs"`
	Panels []dashboardPanel `json:"panels"`
}

func loadDashboard(t *testing.T) dashboardFile {
	t.Helper()
	raw, err := os.ReadFile(dashboardPath)
	if err != nil {
		t.Fatalf("reading the dashboard: %s", err)
	}
	var d dashboardFile
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatalf("the dashboard is not valid JSON: %s", err)
	}
	return d
}

func TestDashboard_ImportableWithOnlyThePrometheusInput(t *testing.T) {
	d := loadDashboard(t)
	if len(d.Inputs) != 1 || d.Inputs[0].Name != "DS_PROMETHEUS" {
		t.Errorf("__inputs = %+v, want only DS_PROMETHEUS", d.Inputs)
	}
}

func TestDashboard_RowsInSpecOrder(t *testing.T) {
	var rows []string
	for _, p := range loadDashboard(t).Panels {
		if p.Type == "row" {
			rows = append(rows, p.Title)
		}
	}
	if strings.Join(rows, "|") != strings.Join(dashboardRows, "|") {
		t.Errorf("rows = %v, want %v", rows, dashboardRows)
	}
}

func TestDashboard_PanelIdsUniqueAndNothingOverlaps(t *testing.T) {
	ids := map[int]string{}
	var group []dashboardPanel // the panels of the current row
	var groupTitle string
	checkGroup := func() {
		for i, a := range group {
			if a.GridPos.X < 0 || a.GridPos.W <= 0 || a.GridPos.X+a.GridPos.W > 24 {
				t.Errorf("row %q: panel %q does not fit the 24-column grid: %+v", groupTitle, a.Title, a.GridPos)
			}
			for _, b := range group[i+1:] {
				overlapX := a.GridPos.X < b.GridPos.X+b.GridPos.W && b.GridPos.X < a.GridPos.X+a.GridPos.W
				overlapY := a.GridPos.Y < b.GridPos.Y+b.GridPos.H && b.GridPos.Y < a.GridPos.Y+a.GridPos.H
				if overlapX && overlapY {
					t.Errorf("row %q: panels %q and %q overlap", groupTitle, a.Title, b.Title)
				}
			}
		}
	}
	for _, p := range loadDashboard(t).Panels {
		if other, dup := ids[p.Id]; dup {
			t.Errorf("panel id %d is used by %q and %q", p.Id, other, p.Title)
		}
		ids[p.Id] = p.Title
		if p.Type == "row" {
			checkGroup()
			group, groupTitle = nil, p.Title
			continue
		}
		group = append(group, p)
	}
	checkGroup()
}

// prometheusName converts an OpenTelemetry instrument to the series name the OTLP-to-Prometheus
// translation gives it: dots become underscores, the unit is appended (s -> _seconds, ms ->
// _milliseconds, By -> _bytes), and a monotonic counter gets _total. An up/down counter becomes
// a plain gauge; a histogram is queried through its _bucket series, so only its base name is
// returned.
func prometheusName(name string, unit string, kind string) string {
	out := strings.ReplaceAll(name, ".", "_")
	switch unit {
	case "s":
		out += "_seconds"
	case "ms":
		out += "_milliseconds"
	case "By":
		out += "_bytes"
	}
	if kind == "Int64Counter" || kind == "Float64Counter" || kind == "Int64ObservableCounter" || kind == "Float64ObservableCounter" {
		out += "_total"
	}
	return out
}

// telemetryInstruments lists every instrument created from Meter in telemetry.go, with its
// Prometheus series name.
func telemetryInstruments(t *testing.T) map[string]string {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "telemetry.go", nil, 0)
	if err != nil {
		t.Fatalf("parsing telemetry.go: %s", err)
	}
	instruments := map[string]string{} // OTel name -> Prometheus name
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if recv, ok := sel.X.(*ast.Ident); !ok || recv.Name != "Meter" || !strings.HasPrefix(sel.Sel.Name, "Int64") && !strings.HasPrefix(sel.Sel.Name, "Float64") {
			return true
		}
		if sel.Sel.Name == "Int64ObservableUpDownCounter" || sel.Sel.Name == "Float64ObservableUpDownCounter" {
			t.Fatalf("telemetry.go creates a %s; extend prometheusName", sel.Sel.Name)
		}
		if len(call.Args) == 0 {
			return true
		}
		nameLit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || nameLit.Kind != token.STRING {
			return true
		}
		name, _ := strconv.Unquote(nameLit.Value)
		unit := ""
		for _, arg := range call.Args[1:] {
			if opt, ok := arg.(*ast.CallExpr); ok {
				if fn, ok := opt.Fun.(*ast.SelectorExpr); ok && fn.Sel.Name == "WithUnit" && len(opt.Args) == 1 {
					if lit, ok := opt.Args[0].(*ast.BasicLit); ok {
						unit, _ = strconv.Unquote(lit.Value)
					}
				}
			}
		}
		instruments[name] = prometheusName(name, unit, sel.Sel.Name)
		return true
	})
	if len(instruments) < 30 {
		t.Fatalf("found only %d instruments in telemetry.go (%v); the parser lost track of them", len(instruments), instruments)
	}
	return instruments
}

// TestDashboard_EveryMetricHasAPanel: a metric added to telemetry.go without a panel fails here.
func TestDashboard_EveryMetricHasAPanel(t *testing.T) {
	var exprs []string
	for _, p := range loadDashboard(t).Panels {
		for _, target := range p.Targets {
			exprs = append(exprs, target.Expr)
		}
	}
	all := strings.Join(exprs, "\n")
	for otelName, promName := range telemetryInstruments(t) {
		if !regexp.MustCompile(`\b` + regexp.QuoteMeta(promName) + `(_bucket|_sum|_count)?\b`).MatchString(all) {
			t.Errorf("%s (%s) is in telemetry.go but in no panel expression of the dashboard", otelName, promName)
		}
	}
}
