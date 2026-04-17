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

type Rows interface {
	Next() bool
	Scan(dest ...any) error
	ScanStruct(dest any) error
	// ColumnTypes() []ColumnType
	Totals(dest ...any) error
	Columns() []string
	Close() error
	Err() error
	HasData() bool
}

type Conn interface {
	PrepareBatch(ctx any, query string, opts ...any) (Batch, error)
	Query(ctx any, query string, args ...any) (Rows, error)
}
