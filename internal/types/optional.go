package types

type Optional[T any] struct {
	Valid bool
	Value T
}

func NewOptional[T any]() *Optional[T] {
	return &Optional[T]{
		Valid: false,
	}
}

func (o *Optional[T]) SetValue(value T) {
	o.Valid = true
	o.Value = value
}
