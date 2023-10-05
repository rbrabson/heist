package timer

import "time"

type number interface {
	int | int32 | int64 | float32 | float64 | time.Duration
}

func min[N number](v1 N, v2 N) N {
	if v1 < v2 {
		return v1
	}
	return v2
}
