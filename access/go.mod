module github.com/open-hzbank/ultra-flow-access

go 1.22

require (
	github.com/emicklei/go-restful/v3 v3.13.0
	github.com/open-hzbank/ultra-flow v0.0.0
	github.com/open-hzbank/ultra-flow-scenes v0.0.0
)

replace (
	github.com/open-hzbank/ultra-flow => ../
	github.com/open-hzbank/ultra-flow-scenes => ../scenes
)
