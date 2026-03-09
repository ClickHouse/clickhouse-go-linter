package testcases

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

var conn driver.Conn
var ctx = context.Background()

// valid: Next() followed by Err()
func validSimple() {
	rows, _ := conn.Query(ctx, "SELECT 1")
	for rows.Next() {
	}
	_ = rows.Err()
}

// invalid: Next() called, Err() never called
func invalidSimple() {
	rows, _ := conn.Query(ctx, "SELECT 1")
	for rows.Next() { // want `clickhouse rows\.Err\(\) must be checked after rows\.Next\(\)`
	}
}

// reassignment: valid then valid
func reassignValidValid() {
	rows, _ := conn.Query(ctx, "SELECT 1")
	for rows.Next() {
	}
	_ = rows.Err()

	rows, _ = conn.Query(ctx, "SELECT 2")
	for rows.Next() {
	}
	_ = rows.Err()
}

// reassignment: invalid then invalid
func reassignInvalidInvalid() {
	rows, _ := conn.Query(ctx, "SELECT 1")
	for rows.Next() { // want `clickhouse rows\.Err\(\) must be checked after rows\.Next\(\)`
	}

	rows, _ = conn.Query(ctx, "SELECT 2")
	for rows.Next() { // want `clickhouse rows\.Err\(\) must be checked after rows\.Next\(\)`
	}
}

// reassignment: valid then invalid
func reassignValidInvalid() {
	rows, _ := conn.Query(ctx, "SELECT 1")
	for rows.Next() {
	}
	_ = rows.Err()

	rows, _ = conn.Query(ctx, "SELECT 2")
	for rows.Next() { // want `clickhouse rows\.Err\(\) must be checked after rows\.Next\(\)`
	}
}

// reassignment: invalid then valid
func reassignInvalidValid() {
	rows, _ := conn.Query(ctx, "SELECT 1")
	for rows.Next() { // want `clickhouse rows\.Err\(\) must be checked after rows\.Next\(\)`
	}

	rows, _ = conn.Query(ctx, "SELECT 2")
	for rows.Next() {
	}
	_ = rows.Err()
}

// no usage: rows is queried but Next() is never called — no violation expected
func noUsage() {
	_, _ = conn.Query(ctx, "SELECT 1")
}

// Next and Err calls are not in the same function block
// while the code below is in theory correct, it is very likely to be a bad pattern and should be flagged by the linter
func nextAndErrNotInSameFunctionBlock() {
	rows, _ := conn.Query(ctx, "SELECT 1")
	for rows.Next() { // want `clickhouse rows\.Err\(\) must be checked after rows\.Next\(\)`
	}
	err := func() error { return rows.Err() }()
	if err != nil {
		fmt.Printf("an error happened!")
	}
}
