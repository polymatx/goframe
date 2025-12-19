package binding

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// Bind binds request data to struct based on Content-Type
func Bind(r *http.Request, obj interface{}) error {
	contentType := r.Header.Get("Content-Type")

	switch {
	case strings.Contains(contentType, "application/json"):
		return JSON(r, obj)
	case strings.Contains(contentType, "application/xml"):
		return XML(r, obj)
	case strings.Contains(contentType, "application/x-www-form-urlencoded"):
		return Form(r, obj)
	case strings.Contains(contentType, "multipart/form-data"):
		return Form(r, obj)
	default:
		return JSON(r, obj)
	}
}

// JSON binds JSON request body to struct
func JSON(r *http.Request, obj interface{}) error {
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(obj); err != nil {
		if err == io.EOF {
			return fmt.Errorf("request body is empty")
		}
		return fmt.Errorf("invalid JSON: %w", err)
	}

	return Validate(obj)
}

// XML binds XML request body to struct
func XML(r *http.Request, obj interface{}) error {
	defer r.Body.Close()

	decoder := xml.NewDecoder(r.Body)
	if err := decoder.Decode(obj); err != nil {
		return fmt.Errorf("invalid XML: %w", err)
	}

	return Validate(obj)
}

// Form binds form data to struct
func Form(r *http.Request, obj interface{}) error {
	if err := r.ParseForm(); err != nil {
		return err
	}

	return mapForm(obj, r.Form)
}

// Query binds query parameters to struct
func Query(r *http.Request, obj interface{}) error {
	return mapForm(obj, r.URL.Query())
}

// Validate validates struct using validator tags
func Validate(obj interface{}) error {
	if err := validate.Struct(obj); err != nil {
		return err
	}
	return nil
}

// mapForm maps form values to struct fields
func mapForm(ptr interface{}, form map[string][]string) error {
	typ := reflect.TypeOf(ptr).Elem()
	val := reflect.ValueOf(ptr).Elem()

	for i := 0; i < typ.NumField(); i++ {
		typeField := typ.Field(i)
		structField := val.Field(i)

		if !structField.CanSet() {
			continue
		}

		inputFieldName := typeField.Tag.Get("form")
		if inputFieldName == "" {
			inputFieldName = strings.ToLower(typeField.Name)
		}

		inputValue, exists := form[inputFieldName]
		if !exists {
			continue
		}

		numElems := len(inputValue)
		if structField.Kind() == reflect.Slice && numElems > 0 {
			sliceOf := structField.Type().Elem().Kind()
			slice := reflect.MakeSlice(structField.Type(), numElems, numElems)
			for i := 0; i < numElems; i++ {
				if err := setField(sliceOf, inputValue[i], slice.Index(i)); err != nil {
					return err
				}
			}
			val.Field(i).Set(slice)
		} else {
			if err := setField(typeField.Type.Kind(), inputValue[0], structField); err != nil {
				return err
			}
		}
	}
	return nil
}

func setField(valueKind reflect.Kind, val string, field reflect.Value) error {
	switch valueKind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(intVal)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintVal, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(uintVal)
	case reflect.Bool:
		boolVal, err := strconv.ParseBool(val)
		if err != nil {
			return err
		}
		field.SetBool(boolVal)
	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return err
		}
		field.SetFloat(floatVal)
	case reflect.String:
		field.SetString(val)
	default:
		return fmt.Errorf("unknown type: %s", valueKind)
	}
	return nil
}
