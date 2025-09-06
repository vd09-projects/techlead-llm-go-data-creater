package stream

// EncoderFunc converts a value of type T to JSON bytes.
type EncoderFunc[T any] func(T) ([]byte, error)

// Emitter is a generic interface for emitting records of type T.
type Emitter[T any] interface {
	Emit(records []T) error
	EmitOne(record T) error
}
