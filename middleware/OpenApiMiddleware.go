package middleware

import (
	"github.com/linxlib/fw"
	"github.com/valyala/fasthttp"
)

import "embed"

//go:embed doc/*
var FS embed.FS

func NewOpenApiMiddleware(path string, fileName string) *OpenApiMiddleware {
	return &OpenApiMiddleware{
		MiddlewareGlobal: fw.NewMiddlewareGlobal("OpenApiMiddleware"),
		path:             path,
		fileName:         fileName,
	}
}

type OpenApiMiddleware struct {
	*fw.MiddlewareGlobal
	path     string
	fileName string
}

func (o *OpenApiMiddleware) CloneAsMethod() fw.IMiddlewareMethod {
	return o.CloneAsCtl()
}

func (o *OpenApiMiddleware) HandlerMethod(next fw.HandlerFunc) fw.HandlerFunc {
	return next
}

func (o *OpenApiMiddleware) CloneAsCtl() fw.IMiddlewareCtl {
	return NewOpenApiMiddleware(o.path, o.fileName)
}

func (o *OpenApiMiddleware) HandlerController(base string) []*fw.RouteItem {
	return []*fw.RouteItem{
		//跳转
		{
			Method: "GET",
			Path:   base + "/doc/",
			H: func(context *fw.Context) {
				context.Redirect(302, "index.html")
			},
			Middleware: o,
		},
		// 文档的相关文件
		{
			Method: "GET",
			Path:   base + "/doc/{any:*}",
			H: func(context *fw.Context) {
				path := context.GetFastContext().UserValue("any").(string)
				fasthttp.ServeFS(context.GetFastContext(), FS, "/doc/"+path)
			},
			Middleware: o,
		},
		// 生成的json或yaml文件
		{
			Method: "GET",
			Path:   base + "/doc/openapi.yaml",
			H: func(context *fw.Context) {
				context.File(o.fileName)
			},
			Middleware: o,
		},
	}
}
