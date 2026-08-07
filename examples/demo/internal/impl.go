package internal

// 引入所有实现子包，触发各 init() 完成 DI 注册
import (
	_ "github.com/LandcLi/landc-go/examples/demo/internal/controller/auth"
	_ "github.com/LandcLi/landc-go/examples/demo/internal/controller/hello"
	_ "github.com/LandcLi/landc-go/examples/demo/internal/dao_impl/hello"
	_ "github.com/LandcLi/landc-go/examples/demo/internal/service_impl/auth"
	_ "github.com/LandcLi/landc-go/examples/demo/internal/service_impl/hello"
)
