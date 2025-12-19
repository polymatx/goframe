package render

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
)

// JSON renders JSON response
func JSON(w http.ResponseWriter, code int, obj interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	return json.NewEncoder(w).Encode(obj)
}

// JSONIndent renders indented JSON response
func JSONIndent(w http.ResponseWriter, code int, obj interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(obj)
}

// XML renders XML response
func XML(w http.ResponseWriter, code int, obj interface{}) error {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(code)
	return xml.NewEncoder(w).Encode(obj)
}

// String renders plain text response
func String(w http.ResponseWriter, code int, format string, values ...interface{}) error {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(code)
	_, err := fmt.Fprintf(w, format, values...)
	return err
}

// HTML renders HTML template
func HTML(w http.ResponseWriter, code int, name string, data interface{}) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)

	tmpl, err := template.ParseFiles(name)
	if err != nil {
		return err
	}

	return tmpl.Execute(w, data)
}

// HTMLString renders HTML from string
func HTMLString(w http.ResponseWriter, code int, html string) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	_, err := w.Write([]byte(html))
	return err
}

// File sends file for download
func File(w http.ResponseWriter, filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileInfo.Name()))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	_, err = io.Copy(w, file)
	return err
}

// FileAttachment sends file with custom filename
func FileAttachment(w http.ResponseWriter, filepath, filename string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	_, err = io.Copy(w, file)
	return err
}

// Data renders raw bytes
func Data(w http.ResponseWriter, code int, contentType string, data []byte) error {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(code)
	_, err := w.Write(data)
	return err
}

// NoContent sends 204 No Content
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Redirect redirects to URL
func Redirect(w http.ResponseWriter, r *http.Request, code int, location string) {
	http.Redirect(w, r, location, code)
}

// TemplateRenderer holds templates
type TemplateRenderer struct {
	templates *template.Template
}

// NewTemplateRenderer creates a new template renderer
func NewTemplateRenderer(pattern string) (*TemplateRenderer, error) {
	tmpl, err := template.ParseGlob(pattern)
	if err != nil {
		return nil, err
	}

	return &TemplateRenderer{
		templates: tmpl,
	}, nil
}

// Render renders a template by name
func (tr *TemplateRenderer) Render(w http.ResponseWriter, code int, name string, data interface{}) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	return tr.templates.ExecuteTemplate(w, name, data)
}

// AddTemplate adds a template file
func (tr *TemplateRenderer) AddTemplate(files ...string) error {
	tmpl, err := tr.templates.ParseFiles(files...)
	if err != nil {
		return err
	}
	tr.templates = tmpl
	return nil
}

// AddTemplateGlob adds templates by glob pattern
func (tr *TemplateRenderer) AddTemplateGlob(pattern string) error {
	tmpl, err := tr.templates.ParseGlob(pattern)
	if err != nil {
		return err
	}
	tr.templates = tmpl
	return nil
}
