package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
)

// Context wraps http.Request and http.ResponseWriter with additional functionality
type Context struct {
	Request  *http.Request
	Response http.ResponseWriter
	params   map[string]string
	query    url.Values
}

// NewContext creates a new Context
func NewContext(w http.ResponseWriter, r *http.Request) *Context {
	return &Context{
		Request:  r,
		Response: w,
		params:   mux.Vars(r),
		query:    r.URL.Query(),
	}
}

// Param returns URL parameter by name
func (c *Context) Param(name string) string {
	return c.params[name]
}

// Query returns query parameter by name
func (c *Context) Query(name string) string {
	return c.query.Get(name)
}

// QueryDefault returns query parameter with default value
func (c *Context) QueryDefault(name, defaultValue string) string {
	value := c.query.Get(name)
	if value == "" {
		return defaultValue
	}
	return value
}

// Header returns request header by name
func (c *Context) Header(name string) string {
	return c.Request.Header.Get(name)
}

// SetHeader sets response header
func (c *Context) SetHeader(name, value string) {
	c.Response.Header().Set(name, value)
}

// JSON sends JSON response
func (c *Context) JSON(code int, data interface{}) error {
	c.SetHeader("Content-Type", "application/json;charset=UTF-8")
	c.Response.WriteHeader(code)
	return json.NewEncoder(c.Response).Encode(data)
}

// JSONError sends JSON error response
func (c *Context) JSONError(code int, err error) error {
	return c.JSON(code, map[string]string{"error": err.Error()})
}

// String sends string response
func (c *Context) String(code int, format string, values ...interface{}) error {
	c.SetHeader("Content-Type", "text/plain;charset=UTF-8")
	c.Response.WriteHeader(code)
	_, err := fmt.Fprintf(c.Response, format, values...)
	return err
}

// Bind decodes request body into provided struct
func (c *Context) Bind(v interface{}) error {
	defer c.Request.Body.Close()
	return json.NewDecoder(c.Request.Body).Decode(v)
}

// BindJSON is alias for Bind
func (c *Context) BindJSON(v interface{}) error {
	return c.Bind(v)
}

// Body returns raw request body
func (c *Context) Body() ([]byte, error) {
	return io.ReadAll(c.Request.Body)
}

// Status sends status code only
func (c *Context) Status(code int) {
	c.Response.WriteHeader(code)
}

// NoContent sends 204 No Content
func (c *Context) NoContent() {
	c.Status(http.StatusNoContent)
}

// Redirect redirects to URL
func (c *Context) Redirect(code int, url string) error {
	if code < 300 || code > 308 {
		return fmt.Errorf("invalid redirect code")
	}
	c.SetHeader("Location", url)
	c.Status(code)
	return nil
}

// ClientIP returns client IP address
func (c *Context) ClientIP() string {
	// Check CF-Connecting-IP
	if ip := c.Header("CF-Connecting-IP"); ip != "" {
		return ip
	}

	// Check X-Forwarded-For
	if ip := c.Header("X-Forwarded-For"); ip != "" {
		return ip
	}

	// Check X-Real-IP
	if ip := c.Header("X-Real-IP"); ip != "" {
		return ip
	}

	// Use RemoteAddr
	return c.Request.RemoteAddr
}

// Method returns HTTP method
func (c *Context) Method() string {
	return c.Request.Method
}

// Path returns request path
func (c *Context) Path() string {
	return c.Request.URL.Path
}

// UserAgent returns User-Agent header
func (c *Context) UserAgent() string {
	return c.Header("User-Agent")
}
