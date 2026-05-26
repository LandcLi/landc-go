package demo

import (
	"github.com/LandcLi/landc-go/mvc/pkg/response"
	"github.com/gin-gonic/gin"
)

// ============ Gin 路由 Handler（API 层 -> Controller 层的桥梁）============

// Login 登录
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	ctx := c.Request.Context()
	resp, err := GetUserController().Login(ctx, &req)
	if err != nil {
		response.Error(c, response.CodeUnauthorized, err.Error())
		return
	}

	response.Success(c, resp)
}

// CreateUser 创建用户
func CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	ctx := c.Request.Context()
	user, err := GetUserController().CreateUser(ctx, &req)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, user)
}

// GetUser 获取用户
func GetUser(c *gin.Context) {
	var req GetUserRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	ctx := c.Request.Context()
	user, err := GetUserController().GetUser(ctx, req.ID)
	if err != nil {
		response.Error(c, response.CodeNotFound, err.Error())
		return
	}

	response.Success(c, user)
}

// UpdateUser 更新用户
func UpdateUser(c *gin.Context) {
	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	ctx := c.Request.Context()
	user, err := GetUserController().UpdateUser(ctx, &req)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, user)
}

// DeleteUser 删除用户
func DeleteUser(c *gin.Context) {
	var req DeleteUserRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	ctx := c.Request.Context()
	if err := GetUserController().DeleteUser(ctx, req.ID); err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.SuccessWithMessage(c, "user deleted", nil)
}

// ListUsers 用户列表
func ListUsers(c *gin.Context) {
	var req ListUsersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	ctx := c.Request.Context()
	resp, err := GetUserController().ListUsers(ctx, &req)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, resp)
}
