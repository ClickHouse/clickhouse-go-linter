package testcases

import (
	"context"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

var conn driver.Conn
var ctx = context.Background()

// valid: defer Close() after PrepareBatch
func validDeferClose() {
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO t")
	if err != nil {
		return
	}
	defer batch.Close()
	_ = batch.Append(1)
	_ = batch.Send()
}

// invalid: defer Abort() after PrepareBatch
// defer Abort() can return an error that is ignored by defer. batch.Close should be used
func invalidDeferAbort() {
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO t") //want `clickhouse Batch batch must be closed defensively with defer batch\.Close\(\) after successful instantiation`
	if err != nil {
		return
	}
	defer batch.Abort()
	_ = batch.Append(1)
	_ = batch.Send()
}

// valid: batch is returned to caller
// a common use case is a helper method that prepares a batch and check for errors with custom logging logic (see below)
func validReturn() (driver.Batch, error) {
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO t")
	if err != nil {
		return nil, err
	}
	return batch, nil
}

// valid: batch is returned to caller "without instantiation"
func helperReturningBatch() (driver.Batch, error) {
	return conn.PrepareBatch(ctx, "INSERT INTO t")
}

// valid: defer Close() after Batch is instantiated from helper method
func validFromHelper() {
	batch, err := helperReturningBatch()
	if err != nil {
		return
	}
	defer batch.Close()
	_ = batch.Append(1)
	_ = batch.Send()
}

// invalid assignment to blank identifier.
// unlikely to exist in real code base but results in a connection leak.
func toBlankIdentifier() {
	_, err := conn.PrepareBatch(ctx, "INSERT INTO t") // want `clickhouse Batch assigned to blank identifier. Connection leak. clickhouse Batch must be instantiated and closed defensively with defer batch\.Close\(\) after successful instantiation`
	if err != nil {
		return
	}
}

// invalid: no defer, no return
func invalidNoDeferNoReturn() {
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO t") //want `clickhouse Batch batch must be closed defensively with defer batch\.Close\(\) after successful instantiation`
	if err != nil {
		return
	}
	_ = batch.Append(1)
	_ = batch.Send()
}

// invalid: Send() called but no defer — Send can fail, leaking the batch
func invalidOnlySend() {
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO t") //want `clickhouse Batch batch must be closed defensively with defer batch\.Close\(\) after successful instantiation`
	if err != nil {
		return
	}
	if err := batch.Append(1); err != nil {
		return
	}
	_ = batch.Send()
}

// invalid: batch passed to another function but no defer at call site
func sendHelper(b driver.Batch) error {
	return b.Send()
}

func invalidPassedToFunc() {
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO t") //want `clickhouse Batch batch must be closed defensively with defer batch\.Close\(\) after successful instantiation`
	if err != nil {
		return
	}
	_ = sendHelper(batch)
}

// invalid: batch from helper function, no defer, no return
func invalidFromHelper() {
	batch, err := helperReturningBatch() //want `clickhouse Batch batch must be closed defensively with defer batch\.Close\(\) after successful instantiation`
	if err != nil {
		return
	}
	_ = batch.Append(1)
	_ = batch.Send()
}

// reassignment: first has no defer (invalid), second has defer (valid)
func reassignInvalidValid() {
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO t1") //want `clickhouse Batch batch must be closed defensively with defer batch\.Close\(\) after successful instantiation`
	if err != nil {
		return
	}
	_ = batch.Send()

	batch, err = conn.PrepareBatch(ctx, "INSERT INTO t2")
	if err != nil {
		return
	}
	defer batch.Close()
	_ = batch.Send()
}

// reassignment: first has defer (valid), second has no defer (invalid)
func reassignValidInvalid() {
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO t1")
	if err != nil {
		return
	}
	defer batch.Close()
	_ = batch.Send()

	batch, err = conn.PrepareBatch(ctx, "INSERT INTO t2") //want `clickhouse Batch batch must be closed defensively with defer batch\.Close\(\) after successful instantiation`
	if err != nil {
		return
	}
	_ = batch.Send()
}

// reassignment: first has no defer (invalid), second has no defer (invalid)
func reassignInvalidInvalid() {
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO t1") //want `clickhouse Batch batch must be closed defensively with defer batch\.Close\(\) after successful instantiation`
	if err != nil {
		return
	}
	_ = batch.Send()

	batch, err = conn.PrepareBatch(ctx, "INSERT INTO t2") //want `clickhouse Batch batch must be closed defensively with defer batch\.Close\(\) after successful instantiation`
	if err != nil {
		return
	}
	_ = batch.Send()
}

// reassignment: first has defer (valid), second has defer (valid)
func reassignValidValid() {
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO t1")
	if err != nil {
		return
	}
	defer batch.Close()
	_ = batch.Send()

	batch, err = conn.PrepareBatch(ctx, "INSERT INTO t2")
	if err != nil {
		return
	}
	defer batch.Close()
	_ = batch.Send()
}

// closures are not supported
// while the code below is in theory correct, it is very likely to be a bad pattern and should be flagged by the linter
func deferCloseIsInClosure() {
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO t") //want `clickhouse Batch batch must be closed defensively with defer batch\.Close\(\) after successful instantiation`
	if err != nil {
		return
	}
	defer func() { batch.Close() }()
}

// the code below is in theory correct as all error cases are handled and result in a batch.Abort().
// we still mark this as an error as it's not defensive. A defer batch.Close() would not change the code correctness and is easy to add.
func invalidNotDefensive() error {
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO t") //want `clickhouse Batch batch must be closed defensively with defer batch\.Close\(\) after successful instantiation`
	if err != nil {
		return err
	}

	for i, v := range []string{"a", "b", "c"} {
		err := batch.Append(i, v)
		if err != nil {
			_ = batch.Abort()
			return err
		}
	}

	err = batch.Send()
	if err != nil {
		_ = batch.Abort()
		return err
	}
	return nil
}
