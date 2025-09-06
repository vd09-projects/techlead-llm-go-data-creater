package stream

// Reader is a generic interface for streaming structured data of type T.
type Reader[T any] interface {
	// Next returns the next record.
	// ok=false means EOF. error is non-nil only on failure.
	Next() (record T, ok bool, err error)

	// ReadAll loads all records into memory.
	ReadAll() ([]T, error)

	// Close releases underlying resources (file handles, sockets, etc).
	Close() error
}
