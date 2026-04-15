package chbatchclose

import (
	"go/ast"
	"go/token"

	"github.com/ClickHouse/clickhouse-go-linter/internal/util"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var reportValid bool
var isDebug bool

func init() {
	Analyzer.Flags.BoolVar(&reportValid, "report-valid", false, "Set to true to report valid usages. Used for development purpose to ensure all usages are found.")
	Analyzer.Flags.BoolVar(&isDebug, "log-spurious-cases", false, "Set to true to log spurious but valid cases. Used for development purpose.")
}

var Analyzer = &analysis.Analyzer{
	Name:     "chbatchclosecheck",
	Doc:      "chbatchclosecheck checks whether defer batch.Close() is called on ClickHouse driver Batch variables",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

func run(pass *analysis.Pass) (interface{}, error) {
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
		checkFunc(pass, body)
	})

	return nil, nil
}

// batchUsage tracks whether a driver.Batch variable has a defer Close/Abort or is returned.
type batchUsage struct {
	assignPos     token.Pos
	deferredClose bool
	returned      bool
}

func (b *batchUsage) report(varName string, pass *analysis.Pass) {
	if b.assignPos == token.NoPos {
		// no usage of rows.Next()
		return
	}
	if !b.deferredClose && !b.returned {
		pass.Reportf(b.assignPos,
			"clickhouse Batch %s must be closed defensively with defer %s.Close() after successful instantiation",
			varName, varName)
	} else if reportValid {
		if b.deferredClose {
			pass.Reportf(b.assignPos,
				"clickhouse Batch %s is properly closed defensively after successful instantiation [valid]",
				varName)
		} else {
			pass.Reportf(b.assignPos,
				"clickhouse Batch %s is returned by the function [valid]",
				varName)
		}
	}
}

// checkFunc analyzes a single function/closure body.
// It does a single-pass collection of Batch assignments, defer Close/Abort calls, and return statements.
// It does not descend into nested closures (they are handled as separate units by the Preorder visitor above).
func checkFunc(pass *analysis.Pass, body *ast.BlockStmt) {
	usages := map[string]*batchUsage{}

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
			handleAssign(pass, node, usages)
		case *ast.DeferStmt:
			handleDefer(node, usages)
		case *ast.ReturnStmt:
			handleReturn(node, usages)
		}

		return true
	})

	// remaining usages that were not flushed
	for varName, u := range usages {
		u.report(varName, pass)
	}
}

// handleAssign checks if any LHS variable in the assignment is of type driver.Batch.
// If a tracked variable is reassigned, it flushes/reports the previous tracking first.
func handleAssign(pass *analysis.Pass, assign *ast.AssignStmt, usages map[string]*batchUsage) {
	for _, lhs := range assign.Lhs {
		name := util.IdentName(lhs)
		if name == "" {
			continue
		}
		if name == "_" && util.IsChObj(pass, lhs, "Batch") {
			pass.Reportf(assign.Pos(), "clickhouse Batch assigned to blank identifier. Connection leak. clickhouse Batch must be instantiated and closed defensively with defer batch.Close() after successful instantiation")
			continue
		}

		// if this var was already tracked, flush previous usage before re-tracking
		if u, ok := usages[name]; ok {
			u.report(name, pass)
			delete(usages, name)
		}

		if util.IsChObj(pass, lhs, "Batch") {
			usages[name] = &batchUsage{assignPos: assign.Pos()}
		}
	}
}

// handleDefer checks if a defer statement calls Close() or Abort() on a tracked Batch variable.
func handleDefer(deferStmt *ast.DeferStmt, usages map[string]*batchUsage) {
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
func handleReturn(ret *ast.ReturnStmt, usages map[string]*batchUsage) {
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
