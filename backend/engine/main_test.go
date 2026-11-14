package engine

import (
	"os"
	"testing"

	"github.com/scruz84/striker/common"
)

// TestMain switches restructuring off for the package's tests: most rollover tests lay out their
// own tiny pyramids (a tier or two of one group), which the saved world structure would otherwise
// reshape into a full 310-team country at their first rollover. The restructure tests set
// restructureTarget themselves.
func TestMain(m *testing.M) {
	restructureTarget = func(*common.DatabaseContext) []common.TierConfig { return nil }
	os.Exit(m.Run())
}
