package middleware

import (
	"github.com/linxlib/fw"
)

import "embed"

//go:embed swagger/*
var FS embed.FS

//go:embed rapi/*
var RAPIFS embed.FS

//go:embed openapi-ui/*
var UIFS embed.FS

func NewOpenApiMiddleware(hasLicenseFile bool, licenseFileContent []byte) *OpenApiMiddleware {
	return &OpenApiMiddleware{
		MiddlewareGlobal:   fw.NewMiddlewareGlobal("OpenApiMiddleware"),
		options:            new(OpenApiOptions),
		hasLicenseFile:     hasLicenseFile,
		licenseFileContent: licenseFileContent,
	}
}

type OpenApiOptions struct {
	Redirect bool `yaml:"redirect" default:"true"` //if redirect /doc to /doc/index.html
	//Route    string `yaml:"route" default:"doc"`             // the page route of openapi document. e.g. if your want to serve document at /docA/index.html, just set route to docA
	FileName string `yaml:"fileName" default:"openapi.json"` //file path refer to openapi.yaml or openapi.json
	Type     string `yaml:"type" default:"swagger"`             //ui type. swagger\rapi\openapi-ui
}

type OpenApiMiddleware struct {
	*fw.MiddlewareGlobal
	options            *OpenApiOptions
	hasLicenseFile     bool
	licenseFileContent []byte
	docContent         []byte
	contentType        string
}

func (o *OpenApiMiddleware) SetDocContent(docContent []byte, contentType string) {
	o.docContent = docContent
	o.contentType = contentType
}
func (o *OpenApiMiddleware) DoInitOnce() {
	o.LoadConfig("openapi", o.options)
}

func (o *OpenApiMiddleware) Router(ctx *fw.MiddlewareContext) []*fw.RouteItem {
	ris := make([]*fw.RouteItem, 0)
	if o.options.Redirect {
		ris = append(ris, &fw.RouteItem{
			Method: "GET",
			Path:   "/doc/",
			H: func(context *fw.Context) {
				context.Redirect(302, "index.html")
			},
			Middleware: o,
		})
	}
	if o.hasLicenseFile {
		ris = append(ris, &fw.RouteItem{
			Method: "GET",
			Path:   "/doc/LICENSE",
			H: func(context *fw.Context) {
				context.Data(200, "text/plain", o.licenseFileContent)
			},
			Middleware: o,
		})
	}

	ri := &fw.RouteItem{
		Method:     "GET",
		Path:       "/docs",
		Middleware: o,
	}
	switch o.options.Type {
	case "swagger":
		ri.H = func(context *fw.Context) {
			//path := context.GetFastContext().UserValue("any").(string)
			context.ServeFS(FS, "/swagger/index.html")
		}
	case "rapi":
		ri.H = func(context *fw.Context) {
			//path := context.GetFastContext().UserValue("any").(string)
			context.ServeFS(RAPIFS, "/rapi/index.html")
		}
	case "openapi-ui":
		ri.H = func(context *fw.Context) {
			//path := context.GetFastContext().UserValue("any").(string)
			context.ServeFS(UIFS, "/openapi-ui/index.html")
		}
	default:
		ri.H = func(context *fw.Context) {
			context.String(404, "Not Found")
		}
	}
	ris = append(ris, ri)
	ris = append(ris, &fw.RouteItem{
		Method: "GET",
		Path:   "/openapi.json",
		H: func(context *fw.Context) {
			context.Data(200, o.contentType, o.docContent)
		},
		Middleware: o,
	})
	ris = append(ris, &fw.RouteItem{
		Method: "GET",
		Path:   "/css/index.css",
		H: func(context *fw.Context) {
			context.ServeFS(FS, "/swagger/index.css")
		},
		Middleware: o,
	})
	ris = append(ris, &fw.RouteItem{
		Method: "GET",
		Path:   "/js/swagger-initializer.js",
		H: func(context *fw.Context) {
			context.ServeFS(FS, "/swagger/swagger-initializer.js")
		},
		Middleware: o,
	})
	return ris
}
