// mock CH driver necessary for tests
package driver

type Batch interface {
	Append(v ...any) error
	AppendStruct(v any) error
	Send() error
	Abort() error
	Close() error
	Rows() int
}

type Conn interface {
	PrepareBatch(ctx any, query string, opts ...any) (Batch, error)
}
