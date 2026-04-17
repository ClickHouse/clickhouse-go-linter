package main

import (
	"github.com/ClickHouse/clickhouse-go-linter/passes/chclose"
	"github.com/ClickHouse/clickhouse-go-linter/passes/chrowserr"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	multichecker.Main(chrowserr.NewAnalyzer(), chclose.NewAnalyzer())
}
