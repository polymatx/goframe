package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/polymatx/goframe/pkg/app"
	"github.com/polymatx/goframe/pkg/binding"
	"github.com/polymatx/goframe/pkg/middleware"
)

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
}

var (
	users  = make(map[string]User)
	mu     sync.RWMutex
	nextID = 1
)

func main() {
	a := app.New(&app.Config{
		Name: "rest-api",
		Port: ":8080",
	})

	a.Use(middleware.Recovery())
	a.Use(middleware.Logger())
	a.Use(middleware.DefaultCORS())

	// API routes
	api := a.Group("/api/v1")
	api.GET("/users", getUsers)
	api.POST("/users", createUser)
	api.GET("/users/{id}", getUser)
	api.PUT("/users/{id}", updateUser)
	api.DELETE("/users/{id}", deleteUser)

	a.StartWithGracefulShutdown()
}

func getUsers(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)
	mu.RLock()
	defer mu.RUnlock()

	userList := make([]User, 0, len(users))
	for _, user := range users {
		userList = append(userList, user)
	}

	ctx.JSON(200, userList)
}

func createUser(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)

	var user User
	if err := binding.JSON(r, &user); err != nil {
		ctx.JSONError(400, err)
		return
	}

	mu.Lock()
	user.ID = fmt.Sprintf("%d", nextID)
	nextID++
	users[user.ID] = user
	mu.Unlock()

	ctx.JSON(201, user)
}

func getUser(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)
	id := ctx.Param("id")

	mu.RLock()
	user, exists := users[id]
	mu.RUnlock()

	if !exists {
		ctx.JSONError(404, fmt.Errorf("user not found"))
		return
	}

	ctx.JSON(200, user)
}

func updateUser(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)
	id := ctx.Param("id")

	mu.Lock()
	defer mu.Unlock()

	if _, exists := users[id]; !exists {
		ctx.JSONError(404, fmt.Errorf("user not found"))
		return
	}

	var user User
	if err := binding.JSON(r, &user); err != nil {
		ctx.JSONError(400, err)
		return
	}

	user.ID = id
	users[id] = user

	ctx.JSON(200, user)
}

func deleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)
	id := ctx.Param("id")

	mu.Lock()
	defer mu.Unlock()

	if _, exists := users[id]; !exists {
		ctx.JSONError(404, fmt.Errorf("user not found"))
		return
	}

	delete(users, id)
	ctx.NoContent()
}
