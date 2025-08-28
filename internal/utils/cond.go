package utils

type CondChain[T any] struct {
	matched bool
	value   T
}

func If[T any](cond bool, v T) CondChain[T] {
	if cond {
		return CondChain[T]{matched: true, value: v}
	}
	return CondChain[T]{matched: false}
}

func (c CondChain[T]) ElseIf(cond bool, v T) CondChain[T] {
	if !c.matched && cond {
		return CondChain[T]{matched: true, value: v}
	}
	return c
}

func (c CondChain[T]) Else(v T) T {
	if c.matched {
		return c.value
	}
	return v
}
