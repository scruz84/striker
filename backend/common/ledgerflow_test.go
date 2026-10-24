package common

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/attribute"

	"github.com/scruz84/striker/common/metrictest"
)

func flowSum(t *testing.T, reason string, direction string) int64 {
	t.Helper()
	return int64(metrictest.Sum(t, "striker.economy.flow",
		attribute.String("reason", reason), attribute.String("direction", direction)))
}

// TestLedgerFlows_ExportsOnlyOnCommit: buffered flows are invisible until Commit, then export
// |amount| under the reason and direction, and a second Commit adds nothing.
func TestLedgerFlows_ExportsOnlyOnCommit(t *testing.T) {
	metrictest.Install()
	wagesOut, sponsorIn := flowSum(t, "wages", "out"), flowSum(t, "sponsorship", "in")

	var flows LedgerFlows
	flows.Add(TRANSACTION_WAGES, -120000)
	flows.Add(TRANSACTION_WAGES, -30000)
	flows.Add(TRANSACTION_SPONSORSHIP, 9000)
	flows.Add(TRANSACTION_SPONSORSHIP, 0) // skipped
	if got := flowSum(t, "wages", "out") - wagesOut; got != 0 {
		t.Fatalf("wages exported before Commit: %d", got)
	}

	flows.Commit(t.Context())
	if got := flowSum(t, "wages", "out") - wagesOut; got != 150000 {
		t.Errorf("wages out = %d, want 150000", got)
	}
	if got := flowSum(t, "sponsorship", "in") - sponsorIn; got != 9000 {
		t.Errorf("sponsorship in = %d, want 9000", got)
	}

	flows.Commit(t.Context())
	if got := flowSum(t, "wages", "out") - wagesOut; got != 150000 {
		t.Errorf("a second Commit exported again: wages out = %d", got)
	}
}

// TestLedgerFlows_DroppedWithoutCommit: a transaction that rolls back never calls Commit, so
// nothing it buffered is ever counted.
func TestLedgerFlows_DroppedWithoutCommit(t *testing.T) {
	metrictest.Install()
	before := flowSum(t, "transfer_out", "out")
	var flows LedgerFlows
	flows.Add(TRANSACTION_TRANSFER_OUT, -2000000)
	if got := flowSum(t, "transfer_out", "out") - before; got != 0 {
		t.Errorf("rolled-back flow was counted: %d", got)
	}
}

// TestLedgerFlows_MassInjection: +50 000 into 512 teams is 25 600 000 under one label.
func TestLedgerFlows_MassInjection(t *testing.T) {
	metrictest.Install()
	before := flowSum(t, "admin_mass_injection", "in")
	var flows LedgerFlows
	flows.AddTimes(TRANSACTION_ADMIN_MASS_INJECTION, 50000, 512)
	flows.Commit(t.Context())
	if got := flowSum(t, "admin_mass_injection", "in") - before; got != 25600000 {
		t.Errorf("admin_mass_injection in = %d, want 25600000", got)
	}
}

// TestLedgerFlows_TransferNetsToZero: both legs of a transfer export the same amount.
func TestLedgerFlows_TransferNetsToZero(t *testing.T) {
	metrictest.Install()
	in, out := flowSum(t, "transfer_in", "in"), flowSum(t, "transfer_out", "out")
	var flows LedgerFlows
	flows.Add(TRANSACTION_TRANSFER_OUT, -2000000)
	flows.Add(TRANSACTION_TRANSFER_IN, 2000000)
	flows.Commit(t.Context())
	if gotIn, gotOut := flowSum(t, "transfer_in", "in")-in, flowSum(t, "transfer_out", "out")-out; gotIn != 2000000 || gotOut != 2000000 {
		t.Errorf("transfer legs = in %d / out %d, want 2000000 each", gotIn, gotOut)
	}
}

// TestTransactionReasonLabel_EveryConstantHasOne parses fmconstants.go so a new TRANSACTION_*
// constant fails here until transactionReasonLabel names it.
func TestTransactionReasonLabel_EveryConstantHasOne(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "fmconstants.go", nil, 0)
	if err != nil {
		t.Fatalf("parsing fmconstants.go: %s", err)
	}
	seen := map[string]string{}
	found := 0
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			value := spec.(*ast.ValueSpec)
			for i, name := range value.Names {
				if !strings.HasPrefix(name.Name, "TRANSACTION_") {
					continue
				}
				found++
				lit, ok := value.Values[i].(*ast.BasicLit)
				if !ok {
					t.Fatalf("%s is not an integer literal; extend this test", name.Name)
				}
				reason, err := strconv.Atoi(lit.Value)
				if err != nil {
					t.Fatalf("%s: %s", name.Name, err)
				}
				label := transactionReasonLabel(reason)
				if label == "other" {
					t.Errorf("%s has no label in transactionReasonLabel", name.Name)
				}
				if previous, dup := seen[label]; dup {
					t.Errorf("%s and %s share the label %q", previous, name.Name, label)
				}
				seen[label] = name.Name
			}
		}
	}
	if found == 0 {
		t.Fatal("found no TRANSACTION_* constant in fmconstants.go")
	}
}

// TestLedgerRowsOnlyWrittenThroughInsertLedgerRow keeps striker.economy.flow complete: both
// ledger INSERT statements may be used only inside insertLedgerRow / insertLedgerRowForAllTeams,
// and no other string in the backend may insert into team_transactions.
func TestLedgerRowsOnlyWrittenThroughInsertLedgerRow(t *testing.T) {
	statements := map[string]bool{"InsertTeamTransactionStmtSql": true, "InsertTeamTransactionForAllTeamsStmtSql": true}
	helpers := map[string]bool{"insertLedgerRow": true, "insertLedgerRowForAllTeams": true}

	fset := token.NewFileSet()
	err := filepath.WalkDir("..", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		for _, decl := range file.Decls {
			switch decl := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range decl.Specs {
					value, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					declares := false
					for _, name := range value.Names {
						declares = declares || statements[name.Name]
					}
					if declares {
						continue // the constant's own declaration
					}
					checkLedgerUse(t, fset, value, "package-level declaration", statements)
				}
			case *ast.FuncDecl:
				if decl.Recv == nil && helpers[decl.Name.Name] {
					continue
				}
				checkLedgerUse(t, fset, decl, decl.Name.Name, statements)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking the backend sources: %s", err)
	}
}

func checkLedgerUse(t *testing.T, fset *token.FileSet, node ast.Node, where string, statements map[string]bool) {
	t.Helper()
	ast.Inspect(node, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Ident:
			if statements[n.Name] {
				t.Errorf("%s uses %s in %s; write ledger rows through insertLedgerRow", fset.Position(n.Pos()), n.Name, where)
			}
		case *ast.BasicLit:
			if n.Kind == token.STRING && strings.Contains(strings.ToLower(n.Value), "insert into team_transactions") {
				t.Errorf("%s inserts into team_transactions in %s; write ledger rows through insertLedgerRow", fset.Position(n.Pos()), where)
			}
		}
		return true
	})
}
