package common

import (
	"errors"
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	date, err := ParseDate("2026-08-31")
	if err != nil {
		t.Fatalf("ParseDate() error = %v", err)
	}
	if date.String() != "2026-08-31" {
		t.Fatalf("date = %q, want %q", date.String(), "2026-08-31")
	}
}

func TestParseDateRejectsInvalidValue(t *testing.T) {
	for _, value := range []string{
		"",
		"2026-8-31",
		"2026-02-30",
		"not-a-date",
	} {
		t.Run(value, func(t *testing.T) {
			_, err := ParseDate(value)
			if !errors.Is(err, ErrDateInvalid) {
				t.Fatalf(
					"ParseDate(%q) error = %v, want ErrDateInvalid",
					value,
					err,
				)
			}
		})
	}
}

func TestDateFromTime(t *testing.T) {
	value := time.Date(2026, time.August, 31, 23, 45, 0, 0, time.UTC)

	date := DateFromTime(value)

	if date.String() != "2026-08-31" {
		t.Fatalf("date = %q, want %q", date.String(), "2026-08-31")
	}
}

func TestDateComparison(t *testing.T) {
	earlier, _ := ParseDate("2026-08-31")
	later, _ := ParseDate("2026-09-01")

	if !later.IsAfter(earlier) {
		t.Fatal("later date is not after earlier date")
	}
	if !earlier.IsBefore(later) {
		t.Fatal("earlier date is not before later date")
	}
}
