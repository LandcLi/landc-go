package examples

import (
	"fmt"

	"github.com/LandcLi/landc-go/frame/pkg/meta"
	"github.com/LandcLi/landc-go/frame/pkg/web"
)

type (
	AdvancedController struct{}

	QueryParamsRequest struct {
		meta.Meta `path:"/query" method:"GET" description:"Query parameters example"`
		Page      int    `source:"query" name:"page"`
		PageSize  int    `source:"query" name:"pageSize"`
		Keyword   string `source:"query" name:"keyword"`
		Tags      string `source:"query" name:"tags"`
	}

	PathParamsRequest struct {
		meta.Meta `path:"/path/:category/:id" method:"GET" description:"Path parameters example"`
		Category  string `source:"path" name:"category"`
		ID        int64  `source:"path" name:"id"`
	}

	HeaderParamsRequest struct {
		meta.Meta     `path:"/header" method:"GET" description:"Header parameters example"`
		Authorization string `source:"header" name:"Authorization"`
		ContentType   string `source:"header" name:"Content-Type"`
	}

	FormParamsRequest struct {
		meta.Meta `path:"/form" method:"POST" description:"Form parameters example"`
		Username  string `source:"form" name:"username"`
		Password  string `source:"form" name:"password"`
	}

	BodyParamsRequest struct {
		meta.Meta `path:"/body" method:"POST" description:"Body parameters example"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		Age       int    `json:"age"`
	}

	MixedParamsRequest struct {
		meta.Meta `path:"/mixed/:id" method:"POST" description:"Mixed parameters example"`
		ID        int64  `source:"path" name:"id"`
		Token     string `source:"header" name:"Authorization"`
		Username  string `source:"form" name:"username"`
	}
)

func (c *AdvancedController) QueryParams(req *QueryParamsRequest) (map[string]interface{}, error) {
	return map[string]interface{}{
		"page":     req.Page,
		"pageSize": req.PageSize,
		"keyword":  req.Keyword,
		"tags":     req.Tags,
	}, nil
}

func (c *AdvancedController) PathParams(req *PathParamsRequest) (map[string]interface{}, error) {
	return map[string]interface{}{
		"category": req.Category,
		"id":       req.ID,
	}, nil
}

func (c *AdvancedController) HeaderParams(req *HeaderParamsRequest) (map[string]interface{}, error) {
	return map[string]interface{}{
		"authorization": req.Authorization,
		"contentType":   req.ContentType,
	}, nil
}

func (c *AdvancedController) FormParams(req *FormParamsRequest) (map[string]interface{}, error) {
	return map[string]interface{}{
		"username": req.Username,
		"password": req.Password,
	}, nil
}

func (c *AdvancedController) BodyParams(req *BodyParamsRequest) (map[string]interface{}, error) {
	return map[string]interface{}{
		"name":  req.Name,
		"email": req.Email,
		"age":   req.Age,
	}, nil
}

func (c *AdvancedController) MixedParams(req *MixedParamsRequest) (map[string]interface{}, error) {
	return map[string]interface{}{
		"id":       req.ID,
		"token":    req.Token,
		"username": req.Username,
	}, nil
}

func RunAdvancedExample() {
	server := web.NewServer(&web.ServerConfig{
		Addr: ":8080",
	})

	controller := &AdvancedController{}

	if err := server.RegisterHandler(controller); err != nil {
		fmt.Printf("Failed to register handler: %v\n", err)
		return
	}

	fmt.Println("Advanced server starting on :8080")
	fmt.Println("Available endpoints:")
	fmt.Println("  GET    /query?page=1&pageSize=10&keyword=test&tags=tag1,tag2")
	fmt.Println("  GET    /path/books/123")
	fmt.Println("  GET    /header")
	fmt.Println("  POST   /form")
	fmt.Println("  POST   /body")
	fmt.Println("  POST   /mixed/123")

	if err := server.Run(); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
