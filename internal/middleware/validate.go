package middleware

import (
	"fmt"
	"regexp"
	"time"
)

// Validation error messages — fixed strings only (OWASP A03).
const (
	ErrFieldTooLong     = "field too long"
	ErrFieldRequired    = "field required"
	ErrFieldOutOfRange  = "value out of range"
	ErrInvalidUUID      = "invalid id format"
	ErrDateRangeTooWide = "date range exceeds maximum"
	ErrInvalidDate      = "invalid date format"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// ValidateStringLength checks a string field does not exceed maxLen bytes.
func ValidateStringLength(field, value string, maxLen int) error {
	if len(value) > maxLen {
		return fmt.Errorf("%s: %s", field, ErrFieldTooLong)
	}
	return nil
}

// ValidateRequired checks a string field is non-empty.
func ValidateRequired(field, value string) error {
	if value == "" {
		return fmt.Errorf("%s: %s", field, ErrFieldRequired)
	}
	return nil
}

// ValidateNumericRange checks a float64 value is within [min, max].
func ValidateNumericRange(field string, value, min, max float64) error {
	if value < min || value > max {
		return fmt.Errorf("%s: %s", field, ErrFieldOutOfRange)
	}
	return nil
}

// ValidateUUID checks a string is a valid UUID v4 format.
func ValidateUUID(field, value string) error {
	if !uuidPattern.MatchString(value) {
		return fmt.Errorf("%s: %s", field, ErrInvalidUUID)
	}
	return nil
}

// ValidateDateRange checks that from <= to and the range does not exceed maxDays.
// Returns parsed times in the given timezone.
func ValidateDateRange(fromStr, toStr string, maxDays int, loc *time.Location) (time.Time, time.Time, error) {
	from, err := time.ParseInLocation("2006-01-02", fromStr, loc)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("from: %s", ErrInvalidDate)
	}
	to, err := time.ParseInLocation("2006-01-02", toStr, loc)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("to: %s", ErrInvalidDate)
	}
	if to.Before(from) {
		from, to = to, from
	}
	if to.Sub(from).Hours()/24 > float64(maxDays) {
		return time.Time{}, time.Time{}, fmt.Errorf("%s (%d days max)", ErrDateRangeTooWide, maxDays)
	}
	return from, to, nil
}
