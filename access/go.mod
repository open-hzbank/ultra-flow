module hzbank.com.cn/ultra-flow-access

go 1.22

require (
	github.com/emicklei/go-restful/v3 v3.13.0
	hzbank.com.cn/ultra-flow v0.0.0
	hzbank.com.cn/ultra-flow-scenes v0.0.0
)

replace (
	hzbank.com.cn/ultra-flow => ../
	hzbank.com.cn/ultra-flow-scenes => ../scenes
)
