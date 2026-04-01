package connectutil

import "fmt"

const (
	MaxNameLength        = 255
	MaxDescriptionLength = 2000
	MaxURLLength         = 2048
	MaxPageSize          = 100
)

// ValidateStringLength checks that a string does not exceed the maximum length.
func ValidateStringLength(field, value string, maxLen int) error {
	if len(value) > maxLen {
		return fmt.Errorf("%s exceeds maximum length of %d characters", field, maxLen)
	}

	return nil
}

// StringValidation defines a field-value-maxLen tuple for batch validation.
type StringValidation struct {
	Field  string
	Value  string
	MaxLen int
}

// ValidateStringLengths validates multiple field-value-maxLen tuples.
func ValidateStringLengths(validations ...StringValidation) error {
	for _, v := range validations {
		if err := ValidateStringLength(v.Field, v.Value, v.MaxLen); err != nil {
			return err
		}
	}

	return nil
}
