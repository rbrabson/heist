package main

import (
	"fmt"
	"time"
)

func main() {
	now := time.Now()
	month := now.Month()
	year := now.Year()

	month++
	if month > time.December {
		month = time.January
		year++
	}

	nextMonth := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)

	fmt.Println(nextMonth)

	sleepTime := time.Until(nextMonth)
	fmt.Println(sleepTime)
}
