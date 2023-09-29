package race

// number defines numerical values that operations may be calculated on.
type number interface {
	int | int32 | int64 | float32 | float64
}

// min returns the minimum value of the two numbers
func min[V number](n1 V, n2 V) V {
	if n1 > n2 {
		return n2
	}
	return n1
}
