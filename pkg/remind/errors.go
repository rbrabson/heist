package remind

import "errors"

var (
	ErrNoReminders     = errors.New("you don't have any upcoming notifications")
	ErrInvalidDuration = errors.New("unable to parse duration")
)
