package main

import (
	"fmt"

	"github.com/LandcLi/landc-go/mvc/pkg/meta"
	"github.com/LandcLi/landc-go/mvc/pkg/web"
)

type (
	AuthController struct{}

	LoginRequest struct {
		meta.Meta `path:"/login" method:"POST" description:"User login"`
		Username  string `source:"form" name:"username"`
		Password  string `source:"form" name:"password"`
	}

	LoginResponse struct {
		Token string `json:"token"`
		User  string `json:"user"`
	}

	GetProfileRequest struct {
		meta.Meta `path:"/profile" method:"GET" description:"Get user profile"`
	}

	GetProfileResponse struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	}
)

func (c *AuthController) Login(ctx *web.LandcContext, req *LoginRequest) (*LoginResponse, error) {
	fmt.Printf("Login request from %s\n", ctx.ClientIP())

	ctx.SetHeader("X-Request-ID", ctx.GetHeader("X-Request-ID"))

	if req.Username == "admin" && req.Password == "admin" {
		return &LoginResponse{
			Token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			User:  "admin",
		}, nil
	}

	return nil, fmt.Errorf("invalid credentials")
}

func (c *AuthController) GetProfile(ctx *web.LandcContext, req *GetProfileRequest) (*GetProfileResponse, error) {
	authHeader := ctx.GetHeader("Authorization")

	if authHeader == "" {
		return nil, fmt.Errorf("unauthorized")
	}

	fmt.Printf("Profile request with token: %s\n", authHeader)

	return &GetProfileResponse{
		Username: "admin",
		Email:    "admin@example.com",
	}, nil
}

func RunContextExample() {
	server := web.NewServer(&web.ServerConfig{
		Addr: ":8080",
	})

	authController := &AuthController{}
	if err := server.RegisterHandler(authController); err != nil {
		fmt.Printf("Failed to register auth handler: %v\n", err)
		return
	}

	fmt.Println("Server starting on :8080")
	fmt.Println("Available routes:")
	fmt.Println("  POST /login")
	fmt.Println("  GET  /profile")
	if err := server.Run(); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
