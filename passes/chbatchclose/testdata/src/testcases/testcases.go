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

// valid: defer with a closure that calls batch.Close() directly.
// This is the IDE/errcheck-friendly pattern (allows wrapping the Close error, see test below).
func validDeferCloseInClosure() {
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO t")
	if err != nil {
		return
	}
	defer func() { batch.Close() }()
}

// valid: defer with closure handling the Close() error (real-world pattern of above).
func validDeferCloseInClosureWithErrCheck() {
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO t")
	if err != nil {
		return
	}
	defer func() {
		if err = batch.Close(); err != nil {
			_ = err // log error
		}
	}()
}

// invalid: defer with a closure that takes the batch as an argument.
// this is correct in theory, but not supported for the moment (false positive)
// Supporting this would require mapping closure params back to outer args. Assuming this pattern is not used for the moment.
func invalidDeferCloseInClosureWithArg() {
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO t") //want `clickhouse Batch batch must be closed defensively with defer batch\.Close\(\) after successful instantiation`
	if err != nil {
		return
	}
	defer func(b driver.Batch) { _ = b.Close() }(batch)
}

// invalid: Close is called from a goroutine nested inside the deferred closure.
// The defer itself does not synchronously close the batch.
// in theory correct-ish but for the moment I think it's not a good pattern
func invalidDeferCloseInNestedGoroutine() {
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO t") //want `clickhouse Batch batch must be closed defensively with defer batch\.Close\(\) after successful instantiation`
	if err != nil {
		return
	}
	defer func() {
		go func() { _ = batch.Close() }()
	}()
}

// invalid: 2 level of closure
// correct in theory but not tracked for the moment - false positive
func invalidDeferCloseInNestedCallback() {
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO t") //want `clickhouse Batch batch must be closed defensively with defer batch\.Close\(\) after successful instantiation`
	if err != nil {
		return
	}
	cb := func(fn func()) { fn() }
	defer func() {
		cb(func() { _ = batch.Close() })
	}()
}

// invalid: deferred closure calls only Abort() (or other methods), not Close().
func invalidDeferAbortInClosure() {
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO t") //want `clickhouse Batch batch must be closed defensively with defer batch\.Close\(\) after successful instantiation`
	if err != nil {
		return
	}
	defer func() { _ = batch.Abort() }()
}

// known limitation / false negative: a deferred closure shadows the outer `batch` name
// with a non-Batch local that is .Close()'d. The outer Batch is never closed, but the
// linter currently credits the inner .Close() to the outer name because tracking is
// name-based and does not consult type info inside the deferred closure body.
// In practice this is a contrived pattern that any IDE / shadow linter would flag.
func deferCloseInClosureShadowed() {
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO t")
	if err != nil {
		return
	}
	// no defer batch.Close() on the outer Batch — should ideally be reported.
	defer func() {
		batch := fakeCloser{} // shadows outer `batch`
		_ = batch.Close()
	}()
	_ = batch.Send()
}

type fakeCloser struct{}

func (fakeCloser) Close() error { return nil }

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
