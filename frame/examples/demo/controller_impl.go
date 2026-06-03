package demo

import (
	"context"

	"github.com/LandcLi/landc-go/frame/pkg/auth"
)

// UserController 用户控制器实现
type UserController struct {
	service UserService
}

func NewUserController() *UserController {
	return &UserController{service: GetUserService()}
}

func (c *UserController) CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error) {
	return c.service.CreateUser(ctx, req)
}

func (c *UserController) GetUser(ctx context.Context, id uint) (*User, error) {
	return c.service.GetUser(ctx, id)
}

func (c *UserController) UpdateUser(ctx context.Context, req *UpdateUserRequest) (*User, error) {
	return c.service.UpdateUser(ctx, req)
}

func (c *UserController) DeleteUser(ctx context.Context, id uint) error {
	return c.service.DeleteUser(ctx, id)
}

func (c *UserController) ListUsers(ctx context.Context, req *ListUsersRequest) (*ListUsersResponse, error) {
	list, total, err := c.service.ListUsers(ctx, req.Page, req.PageSize, req.Keyword)
	if err != nil {
		return nil, err
	}

	return &ListUsersResponse{
		List:  list,
		Total: total,
		Page:  req.Page,
		Size:  req.PageSize,
	}, nil
}

func (c *UserController) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	user, err := c.service.Login(ctx, req.Username, req.Password)
	if err != nil {
		return nil, err
	}

	token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Token: token,
		User:  user,
	}, nil
}
