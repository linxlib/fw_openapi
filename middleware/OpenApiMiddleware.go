package middleware

import (
	"github.com/linxlib/fw"
	"github.com/savsgio/gotils/strings"
)

import "embed"

//go:embed docs/*
var FS embed.FS

func NewOpenApiMiddleware(hasLicenseFile bool, licenseFileContent []byte) *OpenApiMiddleware {
	return &OpenApiMiddleware{
		MiddlewareGlobal:   fw.NewMiddlewareGlobal("OpenApiMiddleware"),
		options:            new(OpenApiOptions),
		hasLicenseFile:     hasLicenseFile,
		licenseFileContent: licenseFileContent,
	}
}

type OpenApiOptions struct {
	Open bool   `yaml:"open" default:"false"`   // open browser
	Type string `yaml:"type" default:"swagger"` //ui type. swagger\rapi\openapi-ui
}

type OpenApiMiddleware struct {
	*fw.MiddlewareGlobal
	options            *OpenApiOptions
	isProd             bool
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
	if !o.isProd {
		ris = append(ris, &fw.RouteItem{
			Method: "GET",
			Path:   "/",
			H: func(context *fw.Context) {
				context.Redirect(302, "docs")
			},
			Middleware: o,
		})
	}
	if o.hasLicenseFile {
		ris = append(ris, &fw.RouteItem{
			Method: "GET",
			Path:   "/docs/LICENSE",
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
	if strings.Include([]string{"swagger", "rapi", "openapi-ui"}, o.options.Type) {
		ri.H = func(context *fw.Context) {
			context.ServeFS(FS, "/docs/"+o.options.Type+".html")
		}
	} else {
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
	
	return ris
}

func (o *OpenApiMiddleware) GetDocType() string {
	return o.options.Type
}
func (o *OpenApiMiddleware) SetMode(isProd bool) {
	o.isProd = isProd
}
