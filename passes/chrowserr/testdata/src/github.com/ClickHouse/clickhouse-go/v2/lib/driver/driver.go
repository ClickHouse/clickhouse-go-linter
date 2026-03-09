// mock CH driver necessary for tests
package driver

type Rows interface {
	Next() bool
	Scan(dest ...any) error
	ScanStruct(dest any) error
	Columns() []string
	Close() error
	Err() error
}

type Conn interface {
	Query(ctx any, query string, args ...any) (Rows, error)
}
