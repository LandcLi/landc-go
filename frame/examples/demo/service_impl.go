package demo

import (
	"context"
	"fmt"
)

// UserServiceImpl 用户服务实现
type UserServiceImpl struct {
	dao UserDAO
}

func NewUserServiceImpl() *UserServiceImpl {
	return &UserServiceImpl{dao: GetUserDAO()}
}

func (s *UserServiceImpl) CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error) {
	user := &User{
		Username: req.Username,
		Password: req.Password, // 生产环境需要 hash
		Email:    req.Email,
		Phone:    req.Phone,
		Age:      req.Age,
		Role:     "user",
	}

	if err := s.dao.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserServiceImpl) GetUser(ctx context.Context, id uint) (*User, error) {
	return s.dao.GetByID(ctx, id)
}

func (s *UserServiceImpl) UpdateUser(ctx context.Context, req *UpdateUserRequest) (*User, error) {
	user, err := s.dao.GetByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}

	if req.Username != "" {
		user.Username = req.Username
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	if req.Age > 0 {
		user.Age = req.Age
	}

	if err := s.dao.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserServiceImpl) DeleteUser(ctx context.Context, id uint) error {
	return s.dao.Delete(ctx, id)
}

func (s *UserServiceImpl) ListUsers(ctx context.Context, page, pageSize int, keyword string) ([]*User, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize
	return s.dao.List(ctx, offset, pageSize, keyword)
}

func (s *UserServiceImpl) Login(ctx context.Context, username, password string) (*User, error) {
	user, err := s.dao.GetByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("invalid username or password")
	}

	if user.Password != password {
		return nil, fmt.Errorf("invalid username or password")
	}

	return user, nil
}
