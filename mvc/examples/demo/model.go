package demo

import "context"

// ============ Model 层 ============

// User 用户数据模型
type User struct {
	ID        uint   `json:"id"`
	Username  string `json:"username"`
	Password  string `json:"-"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Age       int    `json:"age"`
	Role      string `json:"role"`
	CreatedAt int64  `json:"created_at"`
}

// ============ API 请求/响应 ============

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  *User  `json:"user"`
}

type CreateUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=20"`
	Password string `json:"password" binding:"required,min=6"`
	Email    string `json:"email" binding:"required,email"`
	Phone    string `json:"phone"`
	Age      int    `json:"age"`
}

type UpdateUserRequest struct {
	ID       uint   `json:"id" binding:"required"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Age      int    `json:"age"`
}

type GetUserRequest struct {
	ID uint `form:"id" binding:"required"`
}

type DeleteUserRequest struct {
	ID uint `form:"id" binding:"required"`
}

type ListUsersRequest struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Keyword  string `form:"keyword"`
}

type ListUsersResponse struct {
	List  []*User `json:"list"`
	Total int64   `json:"total"`
	Page  int     `json:"page"`
	Size  int     `json:"size"`
}

// ============ Service 层接口 ============

type UserService interface {
	CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error)
	GetUser(ctx context.Context, id uint) (*User, error)
	UpdateUser(ctx context.Context, req *UpdateUserRequest) (*User, error)
	DeleteUser(ctx context.Context, id uint) error
	ListUsers(ctx context.Context, page, pageSize int, keyword string) ([]*User, int64, error)
	Login(ctx context.Context, username, password string) (*User, error)
}

var globalUserService UserService

func RegisterUserService(s UserService) { globalUserService = s }
func GetUserService() UserService       { return globalUserService }

// ============ DAO 层接口 ============

type UserDAO interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uint) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, offset, limit int, keyword string) ([]*User, int64, error)
}

var globalUserDAO UserDAO

func RegisterUserDAO(d UserDAO) { globalUserDAO = d }
func GetUserDAO() UserDAO       { return globalUserDAO }

// ============ Controller 层接口 ============

type UserControllerInterface interface {
	CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error)
	GetUser(ctx context.Context, id uint) (*User, error)
	UpdateUser(ctx context.Context, req *UpdateUserRequest) (*User, error)
	DeleteUser(ctx context.Context, id uint) error
	ListUsers(ctx context.Context, req *ListUsersRequest) (*ListUsersResponse, error)
	Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error)
}

var globalUserController UserControllerInterface

func RegisterUserController(c UserControllerInterface) { globalUserController = c }
func GetUserController() UserControllerInterface       { return globalUserController }
