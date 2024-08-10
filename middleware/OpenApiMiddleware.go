package middleware

import (
	"github.com/linxlib/config"
	"github.com/linxlib/fw"
	"github.com/valyala/fasthttp"
	"strings"
)

import "embed"

//go:embed doc/*
var FS embed.FS

func NewOpenApiMiddleware() *OpenApiMiddleware {
	return &OpenApiMiddleware{
		MiddlewareGlobal: fw.NewMiddlewareGlobal("OpenApiMiddleware"),
		options:          new(OpenApiOptions),
	}
}

func joinRoute(base string, r string) string {
	var result = base
	if r == "/" || r == "" {

		if strings.HasSuffix(result, "/") && result != "/" {
			result = strings.TrimSuffix(result, "/")
			r = ""
		} else {
			r = strings.TrimSuffix(r, "/")
			result += r
		}
	} else {
		if strings.HasSuffix(result, "/") {
			r = strings.TrimPrefix(r, "/")
			result += r
		} else {
			r = strings.TrimPrefix(r, "/")
			result += "/" + r
		}
	}
	return result
}

type OpenApiOptions struct {
	Redirect bool   `yaml:"redirect" default:"true"`
	Route    string `yaml:"route" default:"doc"`             // the page route of openapi document. e.g. if your want to serve document at /docA/index.html, just set route to docA
	FileName string `yaml:"fileName" default:"openapi.yaml"` //file path refer to openapi.yaml or openapi.json
}

type OpenApiMiddleware struct {
	*fw.MiddlewareGlobal
	Conf    *config.Config `inject:""`
	options *OpenApiOptions
}

func (o *OpenApiMiddleware) Constructor() {
	_ = o.Conf.LoadWithKey("openapi", o.options)
}

func (o *OpenApiMiddleware) CloneAsMethod() fw.IMiddlewareMethod {
	return o.CloneAsCtl()
}

func (o *OpenApiMiddleware) HandlerMethod(next fw.HandlerFunc) fw.HandlerFunc {
	return next
}

func (o *OpenApiMiddleware) CloneAsCtl() fw.IMiddlewareCtl {
	ctl := NewOpenApiMiddleware()
	ctl.Conf = o.Conf
	ctl.options = o.options
	return ctl
}

func (o *OpenApiMiddleware) HandlerController(base string) []*fw.RouteItem {
	return []*fw.RouteItem{
		//跳转
		{
			Method: "GET",
			Path:   joinRoute(base, o.options.Route) + "/",
			H: func(context *fw.Context) {
				context.Redirect(302, "index.html")
			},
			Middleware: o,
		},
		// 文档的相关文件
		{
			Method: "GET",
			Path:   joinRoute(base, o.options.Route) + "/{any:*}",
			H: func(context *fw.Context) {
				path := context.GetFastContext().UserValue("any").(string)
				fasthttp.ServeFS(context.GetFastContext(), FS, "/"+o.options.Route+"/"+path)
			},
			Middleware: o,
		},
		// 生成的json或yaml文件
		{
			//TODO: 这里需要判断一下
			Method: "GET",
			Path:   joinRoute(base, o.options.Route) + "/openapi.yaml",
			H: func(context *fw.Context) {
				context.File(o.options.FileName)
			},
			Middleware: o,
		},
	}
}
