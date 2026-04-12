package validator

import (
	"github.com/go-playground/validator/v10"
)

// Global validator instance
var Validate *validator.Validate

func init() {
	Validate = validator.New()
}

// ValidateStruct validates a struct and returns a map of field errors
func ValidateStruct(s interface{}) map[string]string {
	err := Validate.Struct(s)
	if err == nil {
		return nil
	}

	errors := make(map[string]string)
	for _, err := range err.(validator.ValidationErrors) {
		errors[err.Field()] = formatError(err)
	}

	return errors
}

// formatError creates a human-readable error message
func formatError(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return err.Field() + " is required"
	case "email":
		return "Invalid email format"
	case "min":
		return err.Field() + " must be at least " + err.Param() + " characters"
	case "max":
		return err.Field() + " must be at most " + err.Param() + " characters"
	case "gt":
		return err.Field() + " must be greater than " + err.Param()
	case "gte":
		return err.Field() + " must be greater than or equal to " + err.Param()
	default:
		return "Invalid " + err.Field()
	}
}
