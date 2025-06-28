package middleware

import (
	"fmt"
	"github.com/linxlib/conv"
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
	Open           bool   `yaml:"open" default:"false"`   // open browser
	Type           string `yaml:"type" default:"swagger"` //ui type. swagger\rapi\openapi-ui
	Path           string `yaml:"path" default:"/docs"`
	GroupQueryName string `yaml:"groupQueryName" default:"urls.primaryName"`
	OpenApiPath    string `yaml:"openApiPath" default:"/openapi.json"`
}

type OpenApiMiddleware struct {
	*fw.MiddlewareGlobal
	options            *OpenApiOptions
	isProd             bool
	hasLicenseFile     bool
	licenseFileContent []byte
	docs               map[string]*doc
	docConfig          *DocConfig
}
type doc struct {
	docContent  []byte
	contentType string
}

func (o *OpenApiMiddleware) SetDocContent(groupName string, docContent []byte, contentType string) {
	if o.docs == nil {
		o.docs = make(map[string]*doc)
	}
	o.docs[groupName] = &doc{
		docContent:  docContent,
		contentType: contentType,
	}
	//  "/openapi.json?urls.primaryName=" + groupName
	o.docConfig.Urls = append(o.docConfig.Urls, DocConfigUrl{
		Name: groupName,
		URL:  fmt.Sprintf("%s?%s=%s", o.options.OpenApiPath, o.options.GroupQueryName, groupName),
	})
}
func (o *OpenApiMiddleware) DoInitOnce() {
	o.LoadConfig("openapi", o.options)
	o.docConfig = new(DocConfig)
	o.docConfig.Urls = make([]DocConfigUrl, 0)
}

type DocConfig struct {
	Urls                   []DocConfigUrl `json:"urls,omitempty"`
	ValidatorUrl           string         `json:"validatorUrl,omitempty"`
	DeepLinking            bool           `json:"deepLinking,omitempty"`
	DocExpansion           string         `json:"docExpansion,omitempty"`
	QueryConfigEnabled     bool           `json:"queryConfigEnabled,omitempty"`
	Url                    string         `json:"url,omitempty"`
	TryItOutEnabled        bool           `json:"tryItOutEnabled,omitempty"`
	SupportedSubmitMethods []string       `json:"supported_submit_methods,omitempty"`
	DisplayRequestDuration bool           `json:"displayRequestDuration,omitempty"`
}

type DocConfigUrl struct {
	URL  string `json:"url,omitempty"`
	Name string `json:"name,omitempty"`
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
	ris = append(ris, &fw.RouteItem{
		Method: "GET",
		Path:   "/docs/config",
		H: func(context *fw.Context) {
			context.JSON(200, o.docConfig)
		},
		Middleware: o,
	})
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
		Path:       o.options.Path,
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
		Path:   o.options.OpenApiPath,
		H: func(context *fw.Context) {
			//urls.primaryName
			var primaryName = context.QueryArgs().Peek(o.options.GroupQueryName)
			if primaryName != nil {
				context.Data(200, o.docs[conv.String(primaryName)].contentType, o.docs[conv.String(primaryName)].docContent)
			} else {
				context.Data(200, o.docs["default"].contentType, o.docs["default"].docContent)
			}
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
