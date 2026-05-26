package main

import (
	"fmt"

	"github.com/LandcLi/landc-go/mvc/pkg/meta"
	"github.com/LandcLi/landc-go/mvc/pkg/web"
)

type (
	UserController struct{}

	GetUserRequest struct {
		meta.Meta `path:"/user/:id" method:"GET" description:"Get user by ID"`
		ID        int64  `source:"path" name:"id"`
		Name      string `source:"query" name:"name"`
	}

	GetUserResponse struct {
		ID    int64  `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	CreateUserRequest struct {
		meta.Meta `path:"/user" method:"POST" description:"Create new user"`
		Name      string `source:"form" name:"name"`
		Email     string `source:"form" name:"email"`
	}

	CreateUserResponse struct {
		ID    int64  `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	ApiController struct {
		meta.Meta `path:"/api/v1"`
	}

	GetItemRequest struct {
		meta.Meta `path:"/item/:id" method:"GET" description:"Get item by ID"`
		ID        int64  `source:"path" name:"id"`
		Name      string `source:"query" name:"name"`
	}

	GetItemResponse struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
)

func (c *UserController) GetUser(req *GetUserRequest) (*GetUserResponse, error) {
	fmt.Printf("GetUser called with ID: %d, Name: %s\n", req.ID, req.Name)

	return &GetUserResponse{
		ID:    req.ID,
		Name:  "John Doe",
		Email: "john@example.com",
	}, nil
}

func (c *UserController) CreateUser(req *CreateUserRequest) (*CreateUserResponse, error) {
	fmt.Printf("CreateUser called with Name: %s, Email: %s\n", req.Name, req.Email)

	return &CreateUserResponse{
		ID:    123,
		Name:  req.Name,
		Email: req.Email,
	}, nil
}

func (c *ApiController) GetItem(req *GetItemRequest) (*GetItemResponse, error) {
	fmt.Printf("GetItem called with ID: %d, Name: %s\n", req.ID, req.Name)

	return &GetItemResponse{
		ID:   req.ID,
		Name: "Sample Item",
	}, nil
}

func RunWebExample() {
	server := web.NewServer(&web.ServerConfig{
		Addr: ":8080",
	})

	userController := &UserController{}

	if err := server.RegisterHandler(userController); err != nil {
		fmt.Printf("Failed to register handler: %v\n", err)
		return
	}

	apiController := &ApiController{}

	if err := server.RegisterHandler(apiController); err != nil {
		fmt.Printf("Failed to register API handler: %v\n", err)
		return
	}

	fmt.Println("Server starting on :8080")
	fmt.Println("Available routes:")
	fmt.Println("  GET  /user/:id")
	fmt.Println("  POST /user")
	fmt.Println("  GET  /api/v1/item/:id")
	if err := server.Run(); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
