package controller

import (
	"net/http"
	"reflect"

	"github.com/go-playground/validator/v10"
)

// ValidationErrors represents validation errors as a map
type ValidationErrors map[string]string

// Validate validates a struct using validator tags
// Returns a map of field errors and a boolean indicating if validation failed
func Validate(i interface{}) (ValidationErrors, bool) {
	validationErr := make(ValidationErrors)
	validate := validator.New()

	err := validate.Struct(i)
	if err == nil {
		return validationErr, false
	}

	for _, err := range err.(validator.ValidationErrors) {
		field, ok := reflect.TypeOf(i).Elem().FieldByName(err.Field())
		if !ok {
			validationErr[err.Field()] = err.Error()
			continue
		}

		// Try to get custom message from tag
		message := field.Tag.Get("message")
		if message == "" {
			message = err.Error()
		}

		// Use json tag name if available, otherwise use field name
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" {
			validationErr[jsonTag] = message
		} else {
			validationErr[err.Field()] = message
		}
	}

	return validationErr, true
}

// Mix chains multiple middleware functions with a final handler
// Middleware are applied in the order they are passed
func Mix(final http.HandlerFunc, middlewares ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	handler := final

	// Apply middlewares in reverse order so they execute in the order specified
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}

	return handler
}
