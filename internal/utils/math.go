package utils

import (
	"cmp"
	"math"
)

// Generic type constraint for numbers (ints + floats)
type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64
}

// Round to N decimal places (floats only)
func RoundN[T ~float32 | ~float64](f T, decimals int) T {
	pow := math.Pow(10, float64(decimals))
	return T(math.Round(float64(f)*pow) / pow)
}

// Clamp between min and max (ints + floats)
func Clamp[T Number](v, min, max T) T {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// Special case: Clamp between 0 and 1
func Clamp01[T Number](v T) T {
	return Clamp(v, 0, 1)
}

// Max returns the larger of two values (works for numbers and strings)
func Max[T cmp.Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// Min returns the smaller of two values
func Min[T cmp.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}
