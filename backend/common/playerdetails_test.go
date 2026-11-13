package common

import (
	"reflect"
	"strings"
	"testing"
)

// player-negotiation "Private negotiation tier": the player details a manager gets have no field
// that could carry a negotiation tier, whoever asks.
func TestPlayerDetailsCarryNoNegotiationTier(t *testing.T) {
	var check func(typ reflect.Type)
	check = func(typ reflect.Type) {
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			if f.Anonymous && f.Type.Kind() == reflect.Struct {
				check(f.Type)
				continue
			}
			tag := strings.ToLower(strings.Split(f.Tag.Get("json"), ",")[0])
			if tag == "-" || !f.IsExported() {
				continue
			}
			name := strings.ToLower(f.Name + " " + tag)
			if strings.Contains(name, "negotiation") || strings.Contains(name, "intelligence") {
				t.Errorf("%s.%s (json %q) exposes the negotiation tier", typ.Name(), f.Name, tag)
			}
		}
	}
	check(reflect.TypeOf(PlayerDetails{}))
}
