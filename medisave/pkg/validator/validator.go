package validator

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()

	// Register custom tag names from json struct tags
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
}

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func Validate(s interface{}) []ValidationError {
	var errs []ValidationError

	if err := validate.Struct(s); err != nil {
		for _, e := range err.(validator.ValidationErrors) {
			errs = append(errs, ValidationError{
				Field:   e.Field(),
				Message: messageFor(e),
			})
		}
	}

	return errs
}

func messageFor(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", e.Field())
	case "email":
		return fmt.Sprintf("%s must be a valid email", e.Field())
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", e.Field(), e.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", e.Field(), e.Param())
	case "len":
		return fmt.Sprintf("%s must be exactly %s characters", e.Field(), e.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", e.Field(), e.Param())
	case "numeric":
		return fmt.Sprintf("%s must be numeric", e.Field())
	case "e164":
		return fmt.Sprintf("%s must be a valid phone number", e.Field())
	}
	return fmt.Sprintf("%s is invalid", e.Field())
}
