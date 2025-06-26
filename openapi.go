package fw_openapi

import (
	"bufio"
	"fmt"

	"github.com/gookit/goutil/fsutil"
	"github.com/linxlib/astp/constants"
	"github.com/linxlib/astp/types"

	"github.com/linxlib/fw"
	"github.com/linxlib/fw/attribute"
	"github.com/linxlib/fw_openapi/middleware"
	"github.com/pterm/pterm"
	spec "github.com/sv-tools/openapi"
	"os"
	"strings"
)

var innerAttrNames = map[string]attribute.AttributeType{
	"Tag":            attribute.TypeDoc,
	"Deprecated":     attribute.TypeTagger,
	"License":        attribute.TypeDoc,
	"Version":        attribute.TypeDoc,
	"Title":          attribute.TypeDoc,
	"Contact":        attribute.TypeDoc,
	"Description":    attribute.TypeDoc,
	"Summary":        attribute.TypeDoc,
	"TermsOfService": attribute.TypeDoc,
}
var openApiMiddleware *middleware.OpenApiMiddleware

func init() {
	for s, attributeType := range innerAttrNames {
		attribute.RegAttributeType(s, attributeType)
	}
}

func NewOpenAPIPlugin() *OpenAPI {
	oa := &OpenAPI{}
	return oa
}

type OpenAPI struct {
	*spec.OpenAPIBuilder
	s *fw.Server
	//fileName string
	so *fw.ServerOption
}

func (oa *OpenAPI) InitPlugin(s *fw.Server) {
	oa.s = s
	hasLicenseFile := fsutil.FileExist("LICENSE")
	oa.OpenAPIBuilder = spec.NewOpenAPIBuilder()
	oa.OpenAPIBuilder.OpenAPI("3.1.0")
	info := spec.NewInfoBuilder()
	info.Title("FW - OpenAPI 3.0")
	info.Description("")
	info.TermsOfService("https://github.com/linxlib/fw")
	contact := spec.NewContactBuilder()
	contact.Email("email@example.com")
	contact.Name("fw")
	contact.URL("https://github.com/linxlib/fw")
	info.Contact(contact.Build())
	license := spec.NewLicenseBuilder()
	license.Name("MIT License")
	license.URL("https://opensource.org/license/MIT")

	var licenseFileContent []byte
	if hasLicenseFile {
		licenseFileContent, _ = os.ReadFile("LICENSE")
		f, _ := os.Open("LICENSE")
		scanner := bufio.NewScanner(f)
		if scanner.Scan() {
			license.Name(scanner.Text())
		}
		license.URL("./LICENSE")
	} else {
		//info.Spec.License.Spec.Identifier = "MIT"
		license.Name("MIT License")
		license.URL("https://opensource.org/license/MIT")
	}

	info.License(license.Build())
	info.Version("1.0.0@beta")
	oa.OpenAPIBuilder.Info(info.Build())

	sec := spec.NewSecuritySchemeBuilder()
	sec.Name("Authorization")
	sec.Type("apiKey")
	sec.In("header")
	oa.OpenAPIBuilder.AddComponent("ApiKeyAuth", sec.Build())
	//oa.Spec.Components.Spec.SecuritySchemes["ApiKeyAuth"] = sec
	oa.OpenAPIBuilder.JsonSchemaDialect("")
	serverBuilder := spec.NewServerBuilder()
	serverBuilder.URL(fmt.Sprintf("%s://%s:%d%s", s.Schema(), s.ListenAddr(), s.Port(), s.BasePath()))
	serverBuilder.Description("FW Server")
	oa.OpenAPIBuilder.Servers(serverBuilder.Build())
	oa.so = new(fw.ServerOption)
	oa.s.Provide(oa.so)
	openApiMiddleware = middleware.NewOpenApiMiddleware(hasLicenseFile, licenseFileContent)
	s.Use(openApiMiddleware)
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

func (oa *OpenAPI) HandleStructs(ctl *types.Struct) {
	//oa.Log("controller", "start "+ctl.Name)
	//控制器
	allAttrs := ctl.Doc
	tagName := ""
	r := ""
	desc := ctl.Name
	isDeprecated := false
	for _, attr := range allAttrs {
		if attr.AttrType == constants.AT_CUSTOM {
			if strings.ToUpper(attr.CustomAttr) == "DEPRECATED" {
				isDeprecated = true
			} else if strings.ToUpper(attr.CustomAttr) == "TAG" {
				tagName = ctl.Name
				desc = attr.AttrValue
			}
		} else if attr.AttrType == constants.AT_ROUTE {
			r = attr.AttrValue
		} else if attr.IsSelf {
			if attr.AttrValue == "" {
				desc = ctl.Name
			} else {
				desc = attr.AttrValue
			}
		}
	}

	if tagName == "" {
		tagName = ctl.Name
	}

	oa.OpenAPIBuilder.AddTags(oa.NewTag(tagName, quoted(desc)))

	ctl.VisitMethods(func(method *types.Function) bool {
		return !method.Private && method.HasAttrs()
	}, func(method *types.Function) {

		route := oa.so.BasePath
		route = joinRoute(route, r)
		m := ""
		summary := ""
		desc := ""

		isMethodDeprecated := false
		attrs1 := method.Doc
		for _, a := range attrs1 {
			if a.IsHttpMethod() {
				m = constants.AttrNames[a.AttrType]
				route = joinRoute(route, a.AttrValue)
			} else if a.IsSelf {
				if a.AttrValue != "" {
					summary = a.AttrValue
				} else {
					summary = a.Content
				}
			} else if a.AttrType == constants.AT_DEPRECATED {
				isMethodDeprecated = true
			} else {
				desc += "\n" + a.Content
			}

		}
		if m == "" {
			return
		}
		if route == "" {
			route = "/"
		}
		//oa.Log("route", route)
		//fmt.Println(route)
		path := spec.NewPathItemBuilder()

		op := spec.NewOperationBuilder()

		op.OperationID(ctl.Name + "." + method.Name)
		op.Summary(summary)
		op.Description(quoted(desc))
		op.Deprecated(isDeprecated || isMethodDeprecated)

		sr := spec.NewSecurityRequirementBuilder().Add("ApiKeyAuth", "write:"+tagName, "read:"+tagName).Build()
		op.Security(*sr)

		op.Tags(tagName)

		//params
		method.VisitParams(func(element *types.Param) {
			//oa.Log("params", element.TypeName)

			if element.Struct == nil {
				return
			}

			oa.handleParam(element)

			attr := element.Struct.GetAttr()
			switch attr {
			case constants.AT_BODY, constants.AT_JSON:

				body := spec.NewRequestBodyBuilder()
				body.Required(true)

				schema := spec.NewSchemaBuilder().Type("object").Ref("#/components/schemas/" + element.Struct.TypeName).Build()
				mediaType := spec.NewMediaTypeBuilder().Schema(schema).Build()
				body.Description("请求body").AddContent("application/json", mediaType)
				op.RequestBody(body.Build())

			case constants.AT_PATH:
				ps := oa.NewObjectParameters(element.Struct, "path")
				op.AddParameters(ps...)
			case constants.AT_QUERY:
				ps := oa.NewObjectParameters(element.Struct, "query")
				op.AddParameters(ps...)

			case constants.AT_MULTIPART:

				body := spec.NewRequestBodyBuilder()
				body.Required(true)
				schema := oa.NewObjectProp(element.Struct, "multipart")

				//schema := spec.NewSchemaBuilder().Type("object").Ref("#/components/schemas/" + element.Struct.TypeName).Build()

				mediaType := spec.NewMediaTypeBuilder().Schema(schema).Build()
				body.Description("请求body").AddContent("multipart/form-data", mediaType)
				op.RequestBody(body.Build())

			case constants.AT_FORM:
				body := spec.NewRequestBodyBuilder()
				body.Required(true)
				schema := oa.NewObjectProp(element.Struct, "form")
				//schema := spec.NewSchemaBuilder().Type("object").Ref("#/components/schemas/" + element.Struct.TypeName).Build()
				mediaType := spec.NewMediaTypeBuilder().Schema(schema).Build()
				body.Description("请求body").AddContent("application/x-www-form-urlencoded", mediaType)
				op.RequestBody(body.Build())

			case constants.AT_HEADER:
				ps := oa.NewObjectParameters(element.Struct, "header")
				op.Parameters(ps...)
			case constants.AT_COOKIE:
				break
				//ps := oa.NewObjectParameters(element.Struct, "cookie")
				//op.AddParameters(ps...)
			case constants.AT_XML:
				body := spec.NewRequestBodyBuilder()
				body.Required(true)
				schema := spec.NewSchemaBuilder().Type("object").Ref("#/components/schemas/" + element.Struct.TypeName).Build()
				mediaType := spec.NewMediaTypeBuilder().Schema(schema).Build()
				body.Description("请求body").AddContent("application/xml", mediaType)
				op.RequestBody(body.Build())

			case constants.AT_PLAIN:
				body := spec.NewRequestBodyBuilder()
				body.Required(true)
				schema := spec.NewSchemaBuilder().Type("string").Build()
				mediaType := spec.NewMediaTypeBuilder().Schema(schema).Build()
				body.Description("请求body").AddContent("text/plain", mediaType)
				op.RequestBody(body.Build())
			default:

			}
		})

		response := spec.NewResponseBuilder()
		errResponse := spec.NewResponseBuilder()
		method.VisitResults(func(element *types.Param) {
			//oa.Log("results", element.TypeName)
			oa.handleResults(element)
			if element.Struct != nil {
				mediaType := spec.NewMediaTypeBuilder()
				schema := spec.NewSchemaBuilder().Type("object").Ref("#/components/schemas/" + element.TypeName).Build()
				mediaType.Schema(schema)
				response.Description("success").AddContent("application/json", mediaType.Build())
			} else {
				if element.Type == "error" {
					//oa.Log("results", "add 500")
					mediaType := spec.NewMediaTypeBuilder()
					errSchema := spec.NewSchemaBuilder().
						Type("object").
						AddProperty("code", spec.NewSchemaBuilder().Type("integer").Format("int").Example(0).Build()).
						AddProperty("message", spec.NewSchemaBuilder().Type("string").Example("错误信息").Build()).
						Build()
					mediaType.Schema(errSchema)
					errResponse.Description("fail").AddContent("application/json", mediaType.Build())
					return
				}
			}

		})

		//oa.OpenAPIBuilder.AddComponent("success", response.Build())
		op1 := op.Build()
		op1.Spec.Responses = new(spec.Extendable[spec.Responses])
		op1.Spec.Responses.Spec = new(spec.Responses)
		op1.Spec.Responses.Spec.Response = map[string]*spec.RefOrSpec[spec.Extendable[spec.Response]]{
			"200":     response.Build(),
			"not 200": errResponse.Build(),
		}

		switch m {
		case "GET":
			path.Get(op1)
		case "POST":
			path.Post(op1)
		case "PUT":
			path.Put(op1)
		case "DELETE":
			path.Delete(op1)
		case "OPTIONS":
			path.Options(op1)
		default:
			path.Get(op1)
		}

		oa.OpenAPIBuilder.AddPath(route, path.Build())
	})
}

// handleParam
// 将参数对应的类型注册到components.schemas中
func (oa *OpenAPI) handleParam(pf *types.Param) bool {
	if pf.Struct == nil {
		return false
	}

	attr := pf.Struct.GetAttr()
	switch attr {
	case constants.AT_BODY, constants.AT_JSON:
		// 参数的类型
		name := pf.Struct.TypeName
		op := oa.NewObjectProp(pf.Struct, "json")
		oa.OpenAPIBuilder.AddComponent(name, op)
	case constants.AT_XML:
		name := pf.Struct.TypeName
		op := oa.NewObjectProp(pf.Struct, "xml")
		oa.OpenAPIBuilder.AddComponent(name, op)
	case constants.AT_YAML:
		name := pf.Struct.TypeName
		op := oa.NewObjectProp(pf.Struct, "yaml")
		oa.OpenAPIBuilder.AddComponent(name, op)
	case constants.AT_FORM:
		name := pf.Struct.Type
		op := oa.NewObjectProp(pf.Struct, "form")
		oa.OpenAPIBuilder.AddComponent(name, op)
	case constants.AT_MULTIPART:
		name := pf.Struct.Type
		op := oa.NewObjectProp(pf.Struct, "multipart")
		oa.OpenAPIBuilder.AddComponent(name, op)
	default:
		return false
	}
	return true
}

func (oa *OpenAPI) handleResults(pf *types.Param) {
	if pf.Struct == nil {
		return
	}
	schema := oa.NewObjectProp(pf.Struct, "json")
	if pf.Slice {
		schema1 := spec.NewSchemaBuilder().Type("array").Items(spec.NewBoolOrSchema(schema)).Build()
		oa.OpenAPIBuilder.AddComponent(pf.TypeName, schema1)
	} else {
		oa.OpenAPIBuilder.AddComponent(pf.TypeName, schema)
	}

}

func (oa *OpenAPI) Print(slot string) {

	switch slot {
	case fw.AfterListen:
		oa.WriteOut()
		var so = new(fw.ServerOption)
		oa.s.Provide(so)
		style := pterm.NewStyle(pterm.FgLightGreen, pterm.Bold)
		style3 := pterm.NewStyle(pterm.FgLightWhite, pterm.Bold)
		style4 := pterm.NewStyle(pterm.FgWhite)
		style.Print("  ➜ ")
		style3.Printf("%10s", "ApiDoc: ")
		r := joinRoute(so.BasePath, "/docs")
		if oa.s.CanAccessByLan() {
			style4.Printf("http://%s:%d%s\n", so.IntranetIP, so.Port, r)
		} else {
			style4.Printf("http://%s:%d%s\n", oa.s.ListenAddr(), oa.s.Port(), r)
		}

	}
}

func quoted(s string) string {
	return "" + s + ""
}

func (oa *OpenAPI) HandleServerInfo(si []*types.Comment) {
	//for _, attr := range si {
	//	if attr.AttrType == constants.AT_CUSTOM {
	//		switch strings.ToLower(attr.CustomAttr) {
	//		case "title":
	//			oa.OpenAPIBuilder.In
	//		case "license":
	//			strs := strings.SplitN(attr.AttrValue, " ", 3)
	//			oa.Spec.Info.Spec.License.Spec.Name = strs[0]
	//			oa.Spec.Info.Spec.License.Spec.URL = strs[1]
	//			oa.Spec.Info.Spec.License.Spec.Identifier = strs[2]
	//		case "description":
	//			oa.Spec.Info.Spec.Description = quoted(attr.AttrValue)
	//		case "contact":
	//			strs := strings.SplitN(attr.AttrValue, " ", 3)
	//			oa.Spec.Info.Spec.Contact.Spec.Name = strs[0]
	//			oa.Spec.Info.Spec.Contact.Spec.URL = strs[1]
	//			oa.Spec.Info.Spec.Contact.Spec.Email = strs[2]
	//		case "version":
	//			oa.Spec.Info.Spec.Version = attr.AttrValue
	//		case "summary":
	//			oa.Spec.Info.Spec.Summary = attr.AttrValue
	//		case "termsofservice":
	//			oa.Spec.Info.Spec.TermsOfService = attr.AttrValue
	//		}
	//
	//	}
	//}
}

func (oa *OpenAPI) WriteOut() error {
	bs, err := oa.Build().MarshalJSON()
	if err != nil {
		return err
	}
	openApiMiddleware.SetDocContent(bs, "application/json")
	return nil
}
