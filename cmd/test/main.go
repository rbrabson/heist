package main

import (
	"fmt"
	"math"
)

func main() {
	pct := 20.65

	calc1 := int(pct * float64(float64(5)/float64(8)))
	calc2 := (pct * 100 * 5 / 8) / 100
	calc3 := math.Round(calc2)

	fmt.Println(calc1, calc2, calc3)
}
