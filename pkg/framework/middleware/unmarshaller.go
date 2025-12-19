package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"

	"github.com/polymatx/goframe/pkg/array"
	"github.com/polymatx/goframe/pkg/assert"
	"github.com/polymatx/goframe/pkg/framework"
)

type contextKey string

const (
	// ContextBody is the context key for the body unmarshalled object
	ContextBody contextKey = "_body"
)

// PayloadUnMarshallerGenerator create a middleware base on the pattern for the request body
func PayloadUnMarshallerGenerator(pattern interface{}) func(handlerFunc http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Make sure the request is POST or PUT since DELETE and GET must not have payloads
			method := strings.ToUpper(r.Method)
			ok := array.StringInArray(method, "GET", "DELETE")
			assert.True(
				!ok,
				"[BUG] Get and Delete must not have request body",
			)
			// Create a copy
			cp := reflect.New(reflect.TypeOf(pattern)).Elem().Addr().Interface()
			decoder := json.NewDecoder(r.Body)
			err := decoder.Decode(cp)
			if err != nil {
				w.Header().Set("error", "invalid request body")
				e := struct {
					Error string `json:"error"`
				}{
					Error: "invalid request body",
				}
				_ = framework.JSON(w, http.StatusBadRequest, e)
				return
			}

			// Just add it, no validation
			ctx := context.WithValue(r.Context(), ContextBody, cp)
			next(w, r.WithContext(ctx))
		}
	}
}

// GetPayload from the request
func GetPayload(c context.Context) (interface{}, bool) {
	t := c.Value(ContextBody)
	return t, t != nil
}
