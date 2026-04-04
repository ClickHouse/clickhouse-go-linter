package chbatchclose_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/ClickHouse/clickhouse-go-linter/passes/chbatchclose"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), chbatchclose.NewAnalyzer(), "testcases")
}
