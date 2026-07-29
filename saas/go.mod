module github.com/LandcLi/landc-go/saas

go 1.24.0

toolchain go1.24.9

require gorm.io/gorm v1.31.1

require (
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	golang.org/x/text v0.32.0 // indirect
)

replace (
	github.com/LandcLi/landc-go/api v0.1.0 => ../api
	github.com/LandcLi/landc-go/frame v0.1.0 => ../frame
	github.com/LandcLi/landc-go/log v0.1.0 => ../log
	github.com/LandcLi/landc-go/tools v0.1.0 => ../tools
)
