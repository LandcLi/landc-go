package v1

import "github.com/LandcLi/landc-go/frame/pkg/meta"

// LoginRequest 登录请求
type LoginRequest struct {
	meta.Meta `path:"/api/auth/login" method:"POST" description:"登录并签发 JWT"`
	Username  string `source:"body" name:"username" binding:"required"`
	Password  string `source:"body" name:"password" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

// ProfileRequest 获取当前登录用户（需要 Authorization: Bearer <token>）
type ProfileRequest struct {
	meta.Meta `path:"/api/auth/profile" method:"GET" description:"获取当前登录用户信息（需 JWT）"`
}

type ProfileResponse struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}
