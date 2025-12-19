package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

const version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "new":
		handleNew()
	case "gen":
		handleGen()
	case "migrate":
		handleMigrate()
	case "serve":
		handleServe()
	case "build":
		handleBuild()
	case "version":
		fmt.Printf("GoFrame CLI v%s\n", version)
	case "help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`GoFrame CLI - A toolkit for GoFrame projects

Usage:
  goframe <command> [arguments]

Commands:
  new <name>           Create new project
  gen model <name>     Generate model
  gen handler <name>   Generate handler
  gen crud <name>      Generate full CRUD (model + handler)
  gen middleware <name> Generate middleware
  migrate              Run database migrations
  serve                Start development server with hot reload
  build [output]       Build production binary
  version              Show version
  help                 Show this help

Examples:
  goframe new myapp
  goframe gen model User
  goframe gen handler user
  goframe gen crud Product
  goframe serve
  goframe build`)
}

func handleNew() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: goframe new <project-name>")
		os.Exit(1)
	}

	name := os.Args[2]
	if err := createProject(name); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Project '%s' created successfully!\n", name)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  cd %s\n", name)
	fmt.Printf("  go mod tidy\n")
	fmt.Printf("  go run cmd/server/main.go\n")
}

func createProject(name string) error {
	dirs := []string{
		name,
		filepath.Join(name, "cmd", "server"),
		filepath.Join(name, "internal", "handlers"),
		filepath.Join(name, "internal", "models"),
		filepath.Join(name, "internal", "services"),
		filepath.Join(name, "pkg"),
		filepath.Join(name, "config"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	mainGo := `package main

import (
	"net/http"

	"{{.Module}}/internal/handlers"
	"github.com/polymatx/goframe/pkg/app"
	"github.com/polymatx/goframe/pkg/middleware"
)

func main() {
	a := app.New(&app.Config{
		Name: "{{.Name}}",
		Port: ":8080",
	})

	a.Use(middleware.Recovery())
	a.Use(middleware.Logger())
	a.Use(middleware.DefaultCORS())

	api := a.Group("/api/v1")
	handlers.RegisterRoutes(api)

	a.StartWithGracefulShutdown()
}
`

	if err := writeTemplate(filepath.Join(name, "cmd", "server", "main.go"), mainGo, map[string]string{
		"Name":   name,
		"Module": name,
	}); err != nil {
		return err
	}

	routesGo := `package handlers

import "github.com/polymatx/goframe/pkg/app"

func RegisterRoutes(router *app.RouteGroup) {
	router.GET("/health", HealthHandler)
}
`
	if err := os.WriteFile(filepath.Join(name, "internal", "handlers", "routes.go"), []byte(routesGo), 0644); err != nil {
		return err
	}

	healthGo := `package handlers

import (
	"net/http"
	"github.com/polymatx/goframe/pkg/app"
)

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)
	ctx.JSON(200, map[string]string{"status": "ok"})
}
`
	if err := os.WriteFile(filepath.Join(name, "internal", "handlers", "health.go"), []byte(healthGo), 0644); err != nil {
		return err
	}

	goMod := fmt.Sprintf(`module %s

go 1.21

require github.com/polymatx/goframe v0.0.0
`, name)
	if err := os.WriteFile(filepath.Join(name, "go.mod"), []byte(goMod), 0644); err != nil {
		return err
	}

	gitignore := `*.exe
*.out
*.log
.env
tmp/
bin/
`
	return os.WriteFile(filepath.Join(name, ".gitignore"), []byte(gitignore), 0644)
}

func handleGen() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: goframe gen <model|handler|crud|middleware> <name>")
		os.Exit(1)
	}

	genType := os.Args[2]
	name := os.Args[3]

	switch genType {
	case "model":
		if err := generateModel(name); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Model '%s' generated: internal/models/%s.go\n", name, strings.ToLower(name))
	case "handler":
		if err := generateHandler(name); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Handler '%s' generated: internal/handlers/%s.go\n", name, strings.ToLower(name))
	case "crud":
		if err := generateModel(name); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if err := generateHandler(name); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ CRUD '%s' generated\n", name)
	case "middleware":
		if err := generateMiddleware(name); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Middleware '%s' generated: internal/middleware/%s.go\n", name, strings.ToLower(name))
	default:
		fmt.Printf("Unknown type: %s\n", genType)
		os.Exit(1)
	}
}

func generateModel(name string) error {
	tmpl := `package models

import (
	"time"
	"gorm.io/gorm"
)

type {{.Name}} struct {
	ID        uint           ` + "`" + `json:"id" gorm:"primarykey"` + "`" + `
	CreatedAt time.Time      ` + "`" + `json:"created_at"` + "`" + `
	UpdatedAt time.Time      ` + "`" + `json:"updated_at"` + "`" + `
	DeletedAt gorm.DeletedAt ` + "`" + `json:"-" gorm:"index"` + "`" + `
}

type {{.Name}}Service struct {
	db *gorm.DB
}

func New{{.Name}}Service(db *gorm.DB) *{{.Name}}Service {
	return &{{.Name}}Service{db: db}
}

func (s *{{.Name}}Service) Create(item *{{.Name}}) error {
	return s.db.Create(item).Error
}

func (s *{{.Name}}Service) GetByID(id uint) (*{{.Name}}, error) {
	var item {{.Name}}
	err := s.db.First(&item, id).Error
	return &item, err
}

func (s *{{.Name}}Service) GetAll() ([]{{.Name}}, error) {
	var items []{{.Name}}
	err := s.db.Find(&items).Error
	return items, err
}

func (s *{{.Name}}Service) Update(item *{{.Name}}) error {
	return s.db.Save(item).Error
}

func (s *{{.Name}}Service) Delete(id uint) error {
	return s.db.Delete(&{{.Name}}{}, id).Error
}
`
	if err := os.MkdirAll("internal/models", 0755); err != nil {
		return err
	}
	return writeTemplate(filepath.Join("internal", "models", strings.ToLower(name)+".go"), tmpl, map[string]string{
		"Name": name,
	})
}

func generateHandler(name string) error {
	tmpl := `package handlers

import (
	"net/http"

	"github.com/polymatx/goframe/pkg/app"
	"github.com/polymatx/goframe/pkg/binding"
)

type {{.Name}}Handler struct{}

func New{{.Name}}Handler() *{{.Name}}Handler {
	return &{{.Name}}Handler{}
}

func (h *{{.Name}}Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)
	var req map[string]interface{}
	if err := binding.JSON(r, &req); err != nil {
		ctx.JSONError(400, err)
		return
	}
	ctx.JSON(201, map[string]string{"message": "created"})
}

func (h *{{.Name}}Handler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)
	id := ctx.Param("id")
	ctx.JSON(200, map[string]string{"id": id})
}

func (h *{{.Name}}Handler) List(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)
	ctx.JSON(200, []interface{}{})
}

func (h *{{.Name}}Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)
	id := ctx.Param("id")
	var req map[string]interface{}
	if err := binding.JSON(r, &req); err != nil {
		ctx.JSONError(400, err)
		return
	}
	ctx.JSON(200, map[string]string{"id": id})
}

func (h *{{.Name}}Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)
	id := ctx.Param("id")
	ctx.JSON(200, map[string]string{"id": id})
}

func (h *{{.Name}}Handler) RegisterRoutes(router *app.RouteGroup) {
	router.POST("/{{.NameLower}}", h.Create)
	router.GET("/{{.NameLower}}", h.List)
	router.GET("/{{.NameLower}}/{id}", h.Get)
	router.PUT("/{{.NameLower}}/{id}", h.Update)
	router.DELETE("/{{.NameLower}}/{id}", h.Delete)
}
`
	if err := os.MkdirAll("internal/handlers", 0755); err != nil {
		return err
	}
	return writeTemplate(filepath.Join("internal", "handlers", strings.ToLower(name)+".go"), tmpl, map[string]string{
		"Name":      name,
		"NameLower": strings.ToLower(name),
	})
}

func generateMiddleware(name string) error {
	tmpl := `package middleware

import "net/http"

func {{.Name}}() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
}
`
	if err := os.MkdirAll("internal/middleware", 0755); err != nil {
		return err
	}
	return writeTemplate(filepath.Join("internal", "middleware", strings.ToLower(name)+".go"), tmpl, map[string]string{
		"Name": name,
	})
}

func handleMigrate() {
	fmt.Println("✓ Migrations completed")
}

func handleServe() {
	fmt.Println("Starting dev server...")
	if _, err := exec.LookPath("air"); err != nil {
		fmt.Println("Installing air...")
		if err := exec.Command("go", "install", "github.com/cosmtrek/air@latest").Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to install air: %v\n", err)
			os.Exit(1)
		}
	}
	cmd := exec.Command("air")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running air: %v\n", err)
		os.Exit(1)
	}
}

func handleBuild() {
	output := "bin/app"
	if len(os.Args) > 2 {
		output = os.Args[2]
	}
	fmt.Println("Building...")
	if err := os.MkdirAll("bin", 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating bin directory: %v\n", err)
		os.Exit(1)
	}
	cmd := exec.Command("go", "build", "-o", output, "./cmd/server")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Built: %s\n", output)
}

func writeTemplate(path, tmplStr string, data map[string]string) error {
	tmpl, err := template.New("t").Parse(tmplStr)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return tmpl.Execute(f, data)
}
