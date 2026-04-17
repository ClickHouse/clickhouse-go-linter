package chclose

import (
	"go/ast"
	"go/token"
	"os"
	"strconv"

	"github.com/ClickHouse/clickhouse-go-linter/internal/util"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

type analyzer struct {
	// if true, report valid usages and log spurious but valid cases.
	debug bool
}

func NewAnalyzer() *analysis.Analyzer {
	debug, _ := strconv.ParseBool(os.Getenv("CH_GO_LINTER_DEBUG"))
	a := analyzer{
		debug: debug,
	}
	return &analysis.Analyzer{
		Name:     "chclosecheck",
		Doc:      "chclosecheck checks whether defer xxx.Close() is called on ClickHouse driver Batch/Rows variables",
		Run:      a.run,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func (a *analyzer) run(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.FuncLit)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		var body *ast.BlockStmt
		switch fn := n.(type) {
		case *ast.FuncDecl:
			body = fn.Body
		case *ast.FuncLit:
			body = fn.Body
		}

		if body == nil {
			return
		}
		a.checkFunc(pass, body)
	})

	return nil, nil
}

// closableTypes lists the ClickHouse driver types that must be closed defensively.
var closableTypes = []string{"Batch", "Rows"}

// closableUsage tracks whether a driver.Batch / driver.Rows variable has a defer Close() or is returned.
type closableUsage struct {
	typeName      string // "Batch" or "Rows"
	assignPos     token.Pos
	deferredClose bool
	returned      bool
}

func (u *closableUsage) report(varName string, pass *analysis.Pass, debug bool) {
	if u.assignPos == token.NoPos {
		// no usage of Batch
		return
	}
	if !u.deferredClose && !u.returned {
		pass.Reportf(u.assignPos,
			"clickhouse %s %s must be closed defensively with defer %s.Close() after successful instantiation",
			u.typeName, varName, varName)
	} else if debug {
		if u.deferredClose {
			pass.Reportf(u.assignPos,
				"clickhouse %s %s is properly closed defensively after successful instantiation [valid]",
				u.typeName, varName)
		} else {
			pass.Reportf(u.assignPos,
				"clickhouse %s %s is returned by the function [valid]",
				u.typeName, varName)
		}
	}
}

// checkFunc analyzes a single function/closure body.
// It does a single-pass collection of Batch assignments, defer Close/Abort calls, and return statements.
// It does not descend into nested closures (they are handled as separate units by the Preorder visitor above).
func (a *analyzer) checkFunc(pass *analysis.Pass, body *ast.BlockStmt) {
	usages := map[string]*closableUsage{}

	ast.Inspect(body, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		// don't descend into nested closures
		if n != body {
			if _, ok := n.(*ast.FuncLit); ok {
				return false
			}
		}

		switch node := n.(type) {
		case *ast.AssignStmt:
			a.handleAssign(pass, node, usages)
		case *ast.DeferStmt:
			handleDefer(node, usages)
		case *ast.ReturnStmt:
			handleReturn(node, usages)
		}

		return true
	})

	// remaining usages that were not flushed
	for varName, u := range usages {
		u.report(varName, pass, a.debug)
	}
}

// handleAssign checks if any LHS variable in the assignment is a closable ClickHouse driver type.
// If a tracked variable is reassigned, it flushes/reports the previous tracking first.
func (a *analyzer) handleAssign(pass *analysis.Pass, assign *ast.AssignStmt, usages map[string]*closableUsage) {
	for _, lhs := range assign.Lhs {
		name := util.IdentName(lhs)
		if name == "" {
			continue
		}

		for _, typeName := range closableTypes {
			if !util.IsChObj(pass, lhs, typeName) {
				continue
			}

			if name == "_" {
				pass.Reportf(assign.Pos(), "clickhouse %s assigned to blank identifier. Connection leak. clickhouse %s must be instantiated and closed defensively with defer %s.Close() after successful instantiation",
					typeName, typeName, typeName)
				break
			}

			// if this var was already tracked, flush previous usage before re-tracking
			if u, ok := usages[name]; ok {
				u.report(name, pass, a.debug)
				delete(usages, name)
			}

			usages[name] = &closableUsage{typeName: typeName, assignPos: assign.Pos()}
			break
		}
	}
}

// handleDefer checks if a defer statement calls Close() or Abort() on a tracked Batch variable.
func handleDefer(deferStmt *ast.DeferStmt, usages map[string]*closableUsage) {
	call := deferStmt.Call
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	varName := util.IdentName(sel.X)
	if varName == "" {
		return
	}
	u, exists := usages[varName]
	if !exists {
		return
	}

	switch sel.Sel.Name {
	case "Close":
		u.deferredClose = true
	}
}

// handleReturn checks if any return value is a tracked Batch variable.
func handleReturn(ret *ast.ReturnStmt, usages map[string]*closableUsage) {
	for _, result := range ret.Results {
		name := util.IdentName(result)
		if name == "" {
			continue
		}
		if u, exists := usages[name]; exists {
			u.returned = true
		}
	}
}
