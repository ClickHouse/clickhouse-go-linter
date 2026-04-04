package chrowserr_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/ClickHouse/clickhouse-go-linter/passes/chrowserr"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), chrowserr.NewAnalyzer(), "testcases")
}
