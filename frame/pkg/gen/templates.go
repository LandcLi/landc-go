package gen

// ============ API 层模板 ============

var tplAPIInterface = `package {{.NameLower}}

import (
	"context"
	"net/http"

	"{{.Module}}/api/{{.NameLower}}/v1"
	"github.com/gin-gonic/gin"
)

// {{.NameLower}}Api {{.NameCamel}} API 接口
type {{.NameLower}}Api interface {
	Create{{.NameCamel}}(ctx context.Context, req *v1.Create{{.NameCamel}}Request) (*v1.Create{{.NameCamel}}Response, error)
	Get{{.NameCamel}}(ctx context.Context, req *v1.Get{{.NameCamel}}Request) (*v1.Get{{.NameCamel}}Response, error)
	Update{{.NameCamel}}(ctx context.Context, req *v1.Update{{.NameCamel}}Request) (*v1.Update{{.NameCamel}}Response, error)
	Delete{{.NameCamel}}(ctx context.Context, req *v1.Delete{{.NameCamel}}Request) (*v1.Delete{{.NameCamel}}Response, error)
	List{{.NameCamel}}(ctx context.Context, req *v1.List{{.NameCamel}}Request) (*v1.List{{.NameCamel}}Response, error)
}

var global{{.NameCamel}}Api {{.NameLower}}Api

func Register{{.NameCamel}}Controller(controller {{.NameLower}}Api) {
	global{{.NameCamel}}Api = controller
}

func Get{{.NameCamel}}Controller() {{.NameLower}}Api {
	return global{{.NameCamel}}Api
}

// ============ Gin Handler ============

func Create{{.NameCamel}}(c *gin.Context) {
	var req v1.Create{{.NameCamel}}Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	resp, err := Get{{.NameCamel}}Controller().Create{{.NameCamel}}(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func Get{{.NameCamel}}(c *gin.Context) {
	var req v1.Get{{.NameCamel}}Request
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	resp, err := Get{{.NameCamel}}Controller().Get{{.NameCamel}}(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func Update{{.NameCamel}}(c *gin.Context) {
	var req v1.Update{{.NameCamel}}Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	resp, err := Get{{.NameCamel}}Controller().Update{{.NameCamel}}(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func Delete{{.NameCamel}}(c *gin.Context) {
	var req v1.Delete{{.NameCamel}}Request
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	resp, err := Get{{.NameCamel}}Controller().Delete{{.NameCamel}}(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func List{{.NameCamel}}(c *gin.Context) {
	var req v1.List{{.NameCamel}}Request
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	resp, err := Get{{.NameCamel}}Controller().List{{.NameCamel}}(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}
`

var tplAPIRequest = `package v1

// Create{{.NameCamel}}Request 创建{{.NameCamel}}请求
type Create{{.NameCamel}}Request struct {
	Name string ` + "`" + `json:"name" binding:"required"` + "`" + `
}

// Get{{.NameCamel}}Request 获取{{.NameCamel}}请求
type Get{{.NameCamel}}Request struct {
	ID uint ` + "`" + `form:"id" binding:"required"` + "`" + `
}

// Update{{.NameCamel}}Request 更新{{.NameCamel}}请求
type Update{{.NameCamel}}Request struct {
	ID   uint   ` + "`" + `json:"id" binding:"required"` + "`" + `
	Name string ` + "`" + `json:"name"` + "`" + `
}

// Delete{{.NameCamel}}Request 删除{{.NameCamel}}请求
type Delete{{.NameCamel}}Request struct {
	ID uint ` + "`" + `form:"id" binding:"required"` + "`" + `
}

// List{{.NameCamel}}Request 列出{{.NameCamel}}请求
type List{{.NameCamel}}Request struct {
	Page     int    ` + "`" + `form:"page"` + "`" + `
	PageSize int    ` + "`" + `form:"page_size"` + "`" + `
	Keyword  string ` + "`" + `form:"keyword"` + "`" + `
}
`

var tplAPIResponse = `package v1

// Create{{.NameCamel}}Response 创建{{.NameCamel}}响应
type Create{{.NameCamel}}Response struct {
	ID   uint   ` + "`" + `json:"id"` + "`" + `
	Name string ` + "`" + `json:"name"` + "`" + `
}

// Get{{.NameCamel}}Response 获取{{.NameCamel}}响应
type Get{{.NameCamel}}Response struct {
	ID   uint   ` + "`" + `json:"id"` + "`" + `
	Name string ` + "`" + `json:"name"` + "`" + `
}

// Update{{.NameCamel}}Response 更新{{.NameCamel}}响应
type Update{{.NameCamel}}Response struct {
	ID   uint   ` + "`" + `json:"id"` + "`" + `
	Name string ` + "`" + `json:"name"` + "`" + `
}

// Delete{{.NameCamel}}Response 删除{{.NameCamel}}响应
type Delete{{.NameCamel}}Response struct {
	ID uint ` + "`" + `json:"id"` + "`" + `
}

// List{{.NameCamel}}Response 列出{{.NameCamel}}响应
type List{{.NameCamel}}Response struct {
	List  interface{} ` + "`" + `json:"list"` + "`" + `
	Total int64       ` + "`" + `json:"total"` + "`" + `
	Page  int         ` + "`" + `json:"page"` + "`" + `
	Size  int         ` + "`" + `json:"size"` + "`" + `
}
`

// ============ Service 层模板 ============

var tplServiceInterface = `package service

import (
	"context"
	"{{.Module}}/model"
)

// {{.NameCamel}}Service {{.NameCamel}}服务接口
type {{.NameCamel}}Service interface {
	Create{{.NameCamel}}(ctx context.Context, input *model.Create{{.NameCamel}}Input) (*model.Create{{.NameCamel}}Output, error)
	Get{{.NameCamel}}(ctx context.Context, input *model.Get{{.NameCamel}}Input) (*model.Get{{.NameCamel}}Output, error)
	Update{{.NameCamel}}(ctx context.Context, input *model.Update{{.NameCamel}}Input) (*model.Update{{.NameCamel}}Output, error)
	Delete{{.NameCamel}}(ctx context.Context, input *model.Delete{{.NameCamel}}Input) error
	List{{.NameCamel}}(ctx context.Context, input *model.List{{.NameCamel}}Input) (*model.List{{.NameCamel}}Output, error)
}

var global{{.NameCamel}}Service {{.NameCamel}}Service

func Register{{.NameCamel}}Service(s {{.NameCamel}}Service) {
	global{{.NameCamel}}Service = s
}

func Get{{.NameCamel}}Service() {{.NameCamel}}Service {
	return global{{.NameCamel}}Service
}
`

var tplServiceImpl = `package service

import (
	"context"
	"{{.Module}}/dao"
	"{{.Module}}/model"
)

type {{.NameLower}}ServiceImpl struct {
	dao dao.{{.NameCamel}}Dao
}

func New{{.NameCamel}}Service() *{{.NameLower}}ServiceImpl {
	return &{{.NameLower}}ServiceImpl{
		dao: dao.Get{{.NameCamel}}Dao(),
	}
}

func (s *{{.NameLower}}ServiceImpl) Create{{.NameCamel}}(ctx context.Context, input *model.Create{{.NameCamel}}Input) (*model.Create{{.NameCamel}}Output, error) {
	// TODO: 实现业务逻辑
	return nil, nil
}

func (s *{{.NameLower}}ServiceImpl) Get{{.NameCamel}}(ctx context.Context, input *model.Get{{.NameCamel}}Input) (*model.Get{{.NameCamel}}Output, error) {
	// TODO: 实现业务逻辑
	return nil, nil
}

func (s *{{.NameLower}}ServiceImpl) Update{{.NameCamel}}(ctx context.Context, input *model.Update{{.NameCamel}}Input) (*model.Update{{.NameCamel}}Output, error) {
	// TODO: 实现业务逻辑
	return nil, nil
}

func (s *{{.NameLower}}ServiceImpl) Delete{{.NameCamel}}(ctx context.Context, input *model.Delete{{.NameCamel}}Input) error {
	// TODO: 实现业务逻辑
	return nil
}

func (s *{{.NameLower}}ServiceImpl) List{{.NameCamel}}(ctx context.Context, input *model.List{{.NameCamel}}Input) (*model.List{{.NameCamel}}Output, error) {
	// TODO: 实现业务逻辑
	return nil, nil
}
`

// ============ DAO 层模板 ============

var tplDAOInterface = `package dao

import (
	"context"
	"{{.Module}}/model"
)

// {{.NameCamel}}Dao {{.NameCamel}} DAO 接口
type {{.NameCamel}}Dao interface {
	Create(ctx context.Context, input *model.Create{{.NameCamel}}Input) (*model.{{.NameCamel}}, error)
	GetByID(ctx context.Context, id uint) (*model.{{.NameCamel}}, error)
	Update(ctx context.Context, input *model.Update{{.NameCamel}}Input) (*model.{{.NameCamel}}, error)
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, input *model.List{{.NameCamel}}Input) ([]*model.{{.NameCamel}}, int64, error)
}

var global{{.NameCamel}}Dao {{.NameCamel}}Dao

func Register{{.NameCamel}}Dao(d {{.NameCamel}}Dao) {
	global{{.NameCamel}}Dao = d
}

func Get{{.NameCamel}}Dao() {{.NameCamel}}Dao {
	return global{{.NameCamel}}Dao
}
`

var tplDAOImpl = `package dao

import (
	"context"
	"{{.Module}}/model"
	"github.com/LandcLi/landc-go/frame/pkg/db"
)

type {{.NameLower}}DaoImpl struct{}

func New{{.NameCamel}}Dao() *{{.NameLower}}DaoImpl {
	return &{{.NameLower}}DaoImpl{}
}

func (d *{{.NameLower}}DaoImpl) Create(ctx context.Context, input *model.Create{{.NameCamel}}Input) (*model.{{.NameCamel}}, error) {
	entity := &model.{{.NameCamel}}{
		Name: input.Name,
	}
	if err := db.GetTx(ctx).Create(entity).Error; err != nil {
		return nil, err
	}
	return entity, nil
}

func (d *{{.NameLower}}DaoImpl) GetByID(ctx context.Context, id uint) (*model.{{.NameCamel}}, error) {
	var entity model.{{.NameCamel}}
	if err := db.GetTx(ctx).First(&entity, id).Error; err != nil {
		return nil, err
	}
	return &entity, nil
}

func (d *{{.NameLower}}DaoImpl) Update(ctx context.Context, input *model.Update{{.NameCamel}}Input) (*model.{{.NameCamel}}, error) {
	entity := &model.{{.NameCamel}}{ID: input.ID}
	if err := db.GetTx(ctx).Model(entity).Updates(map[string]interface{}{
		"name": input.Name,
	}).Error; err != nil {
		return nil, err
	}
	return d.GetByID(ctx, input.ID)
}

func (d *{{.NameLower}}DaoImpl) Delete(ctx context.Context, id uint) error {
	return db.GetTx(ctx).Delete(&model.{{.NameCamel}}{}, id).Error
}

func (d *{{.NameLower}}DaoImpl) List(ctx context.Context, input *model.List{{.NameCamel}}Input) ([]*model.{{.NameCamel}}, int64, error) {
	var list []*model.{{.NameCamel}}
	var total int64

	query := db.GetTx(ctx).Model(&model.{{.NameCamel}}{})
	if input.Keyword != "" {
		query = query.Where("name LIKE ?", "%"+input.Keyword+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Scopes(db.Paginate(input.Page, input.PageSize)).Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}
`

// ============ Model 层模板 ============

var tplModel = `package model

import "time"

// {{.NameCamel}} {{.NameCamel}}数据模型
type {{.NameCamel}} struct {
	ID        uint      ` + "`" + `gorm:"primarykey" json:"id"` + "`" + `
	Name      string    ` + "`" + `gorm:"type:varchar(100);not null" json:"name"` + "`" + `
	CreatedAt time.Time ` + "`" + `json:"created_at"` + "`" + `
	UpdatedAt time.Time ` + "`" + `json:"updated_at"` + "`" + `
}

func ({{.NameCamel}}) TableName() string {
	return "{{.NameSnake}}s"
}

// Create{{.NameCamel}}Input 创建输入
type Create{{.NameCamel}}Input struct {
	Name string
}

// Get{{.NameCamel}}Input 获取输入
type Get{{.NameCamel}}Input struct {
	ID uint
}

// Update{{.NameCamel}}Input 更新输入
type Update{{.NameCamel}}Input struct {
	ID   uint
	Name string
}

// Delete{{.NameCamel}}Input 删除输入
type Delete{{.NameCamel}}Input struct {
	ID uint
}

// List{{.NameCamel}}Input 列表输入
type List{{.NameCamel}}Input struct {
	Page     int
	PageSize int
	Keyword  string
}

// Create{{.NameCamel}}Output 创建输出
type Create{{.NameCamel}}Output struct {
	ID   uint
	Name string
}

// Get{{.NameCamel}}Output 获取输出
type Get{{.NameCamel}}Output struct {
	ID   uint
	Name string
}

// Update{{.NameCamel}}Output 更新输出
type Update{{.NameCamel}}Output struct {
	ID   uint
	Name string
}

// List{{.NameCamel}}Output 列表输出
type List{{.NameCamel}}Output struct {
	List  []*{{.NameCamel}}
	Total int64
	Page  int
}
`
