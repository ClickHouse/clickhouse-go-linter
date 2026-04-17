package chclose_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/ClickHouse/clickhouse-go-linter/passes/chclose"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), chclose.NewAnalyzer(), "testcases")
}
