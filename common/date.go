package common

import (
	"errors"
	"time"
)

var ErrDateInvalid = errors.New("date is invalid")

type Date struct {
	value string
}

func ParseDate(value string) (Date, error) {
	if len(value) != len(time.DateOnly) {
		return Date{}, ErrDateInvalid
	}

	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil || parsed.Format(time.DateOnly) != value {
		return Date{}, ErrDateInvalid
	}

	return Date{value: value}, nil
}

func DateFromTime(value time.Time) Date {
	return Date{
		value: value.Format(time.DateOnly),
	}
}

func (d Date) IsZero() bool {
	return d.value == ""
}

func (d Date) IsAfter(other Date) bool {
	return d.value > other.value
}

func (d Date) IsBefore(other Date) bool {
	return d.value < other.value
}

func (d Date) String() string {
	return d.value
}
