package fw_openapi

import (
	"github.com/linxlib/astp"
	"github.com/linxlib/fw"
	"github.com/linxlib/fw/attribute"
	"github.com/linxlib/fw_openapi/middleware"
	"github.com/pterm/pterm"
	"github.com/sv-tools/openapi/spec"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
)

var innerAttrNames = map[string]attribute.AttributeType{
	"Tag":            attribute.TypeDoc,
	"Deprecated":     attribute.TypeDoc,
	"License":        attribute.TypeDoc,
	"Version":        attribute.TypeDoc,
	"Title":          attribute.TypeDoc,
	"Contact":        attribute.TypeDoc,
	"Description":    attribute.TypeDoc,
	"Summary":        attribute.TypeDoc,
	"TermsOfService": attribute.TypeDoc,
}

func init() {
	for s, attributeType := range innerAttrNames {
		attribute.RegAttributeType(s, attributeType)
	}
}

//TODO: 1. Responses
//TODO: 2. 字段上的Doc是否也考虑作为description
//TODO: 3. tag上是否增加 example 的tag用于生成样例值
//TODO: 4. components 里结构体本身的Doc需要增加
//TODO: 5. 隐藏Schemas

func NewOpenAPIFromFWServer(s *fw.Server, fileName string) *OpenAPI {
	oa := &OpenAPI{
		Extendable: spec.NewOpenAPI(),
		s:          s,
		fileName:   fileName,
	}
	oa.Spec.OpenAPI = "3.0.3"
	info := spec.NewInfo()
	info.Spec.Title = "FW - OpenAPI 3.0"
	info.Spec.Description = ""
	info.Spec.TermsOfService = "https://github.com/linxlib/fw"
	info.Spec.Contact = spec.NewContact()
	info.Spec.Contact.Spec.Email = "email@example.com"
	info.Spec.Contact.Spec.URL = "https://github.com/linxlib/fw"
	info.Spec.Contact.Spec.Name = "fw"
	info.Spec.License = spec.NewLicense()
	info.Spec.License.Spec.Name = "MIT License"
	info.Spec.License.Spec.URL = "https://opensource.org/license/MIT"
	info.Spec.Version = "1.0.0@beta"
	oa.Spec.Info = info
	oa.Spec.Paths = spec.NewPaths()
	oa.Spec.Components = spec.NewComponents()
	oa.Spec.Components.Spec.Schemas = make(map[string]*spec.RefOrSpec[spec.Schema])
	s.RegisterHooks(oa)
	oa.so = new(fw.ServerOption)
	oa.s.Provide(oa.so)
	s.Use(middleware.NewOpenApiMiddleware())

	return oa
}

type OpenAPI struct {
	*spec.Extendable[spec.OpenAPI]
	s        *fw.Server
	fileName string
	so       *fw.ServerOption
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

func (oa *OpenAPI) HandleStructs(ctl *astp.Element) {
	//控制器
	attrs := attribute.ParseDoc(ctl.Docs, ctl.Name)
	tagName := ""
	r := ""
	desc := ctl.Name
	for _, attr := range attrs {
		if attr.Type == attribute.TypeDoc {
			desc = attr.Name
		}
		if attr.Name == "TAG" {
			tagName = attr.Value
			desc = attr.Value
		}
		if attr.Name == "ROUTE" {
			r = attr.Value
		}
	}
	if tagName == "" {
		tagName = ctl.Name
	}
	tag := spec.NewTag()
	tag.Spec.Name = tagName
	tag.Spec.Description = desc
	oa.Spec.Tags = append(oa.Spec.Tags, tag)

	ctl.VisitElements(astp.ElementMethod, func(method *astp.Element) bool {
		return !method.Private()
	}, func(method *astp.Element) {
		route := oa.so.BasePath
		route = joinRoute(route, r)
		m := ""
		desc := method.Name
		attrs1 := attribute.GetMethodAttributes(method)
		for _, a := range attrs1 {
			if a.Type == attribute.TypeHttpMethod {
				m = a.Name
				route = joinRoute(route, a.Value)
			}
			if a.Type == attribute.TypeDoc {
				if a.Value != "" {
					desc = a.Value
				}

			}
		}
		if m == "" {
			return
		}
		if route == "" {
			route = "/"
		}
		//fmt.Println(route)
		path := spec.NewPathItemSpec()

		op := spec.NewOperation()
		op.Spec.OperationID = ctl.Name + "." + method.Name
		op.Spec.Summary = desc
		op.Spec.Tags = []string{tagName}
		//params
		method.VisitElements(astp.ElementParam, func(element *astp.Element) bool {
			if attribute.HasAttribute(element, "Body") || attribute.HasAttribute(element.Item, "Body") {
				return true
			}
			if attribute.HasAttribute(element, "Query") || attribute.HasAttribute(element.Item, "Query") {
				return true
			}
			if attribute.HasAttribute(element, "Path") || attribute.HasAttribute(element.Item, "Path") {
				return true
			}
			if attribute.HasAttribute(element, "Multipart") || attribute.HasAttribute(element.Item, "Multipart") {
				return true
			}
			if attribute.HasAttribute(element, "Form") || attribute.HasAttribute(element.Item, "Form") {
				return true
			}
			if attribute.HasAttribute(element, "Json") || attribute.HasAttribute(element.Item, "Json") {
				return true
			}
			if attribute.HasAttribute(element, "Header") || attribute.HasAttribute(element.Item, "Header") {
				return true
			}
			if attribute.HasAttribute(element, "XML") || attribute.HasAttribute(element.Item, "XML") {
				return true
			}
			if attribute.HasAttribute(element, "Plain") || attribute.HasAttribute(element.Item, "Plain") {
				return true
			}
			return false

		}, func(element *astp.Element) {
			oa.handleParam(element)
			attr := attribute.GetLastAttr(element)
			switch attr.Name {
			case "BODY", "JSON":
				body := spec.NewRequestBodySpec()
				body.Spec.Spec.Required = true
				body.Spec.Spec.Content = make(map[string]*spec.Extendable[spec.MediaType])
				md := spec.NewMediaType()
				sche := spec.NewRefOrSpec[spec.Schema](spec.NewRef("#/components/schemas/"+element.Item.TypeString), nil)
				md.Spec.Schema = sche
				body.Spec.Spec.Content["application/json"] = md

				op.Spec.RequestBody = body

			case "PATH":
				element.Item.VisitElements(astp.ElementField, func(element *astp.Element) bool {
					return !element.Private()
				}, func(element *astp.Element) {
					t := element.GetTag()
					name := t.Get("path")
					ty := element.TypeString
					param := spec.NewParameterSpec()

					param.Spec.Spec.Name = name
					param.Spec.Spec.Description = element.Comment
					param.Spec.Spec.In = "path"
					param.Spec.Spec.Required = true
					schema := spec.NewSchemaSpec()
					v := spec.NewSingleOrArray[string]("integer")
					schema.Spec.Type = &v
					schema.Spec.Format = ty
					param.Spec.Spec.Schema = schema

					op.Spec.Parameters = append(op.Spec.Parameters, param)

				})

			case "QUERY":

				element.Item.VisitElements(astp.ElementField, func(element *astp.Element) bool {
					return !element.Private()
				}, func(element *astp.Element) {
					t := element.GetTag()
					name := t.Get("query")
					var isRequired bool
					valid := t.Get("validate")
					if strings.Contains(valid, "required") {
						isRequired = true
					}
					var def string
					defStr := t.Get("default")
					if defStr != "" {
						def = defStr
					}

					ty := element.TypeString
					param := spec.NewParameterSpec()

					param.Spec.Spec.Name = name
					param.Spec.Spec.Description = element.Comment
					param.Spec.Spec.In = "query"
					param.Spec.Spec.Required = isRequired
					// TODO: 加入一个类型判断的方法 例如 int64 -> integer
					// TODO: 其他几个也需要处理默认值和必选的情况
					schema := spec.NewSchemaSpec()
					v := spec.NewSingleOrArray[string]("integer")
					schema.Spec.Type = &v
					schema.Spec.Format = ty
					param.Spec.Spec.Schema = schema
					if def != "" {
						schema.Spec.Default = def
					}

					op.Spec.Parameters = append(op.Spec.Parameters, param)

				})
			case "MULTIPART":
				body := spec.NewRequestBodySpec()
				body.Spec.Spec.Required = true
				body.Spec.Spec.Content = make(map[string]*spec.Extendable[spec.MediaType])
				md := spec.NewMediaType()
				sche := spec.NewSchemaSpec()
				sche.Spec.Properties = make(map[string]*spec.RefOrSpec[spec.Schema])

				v := spec.NewSingleOrArray[string]("object")
				sche.Spec.Type = &v

				element.Item.VisitElements(astp.ElementField, func(element *astp.Element) bool {
					return !element.Private()
				}, func(element *astp.Element) {
					t := element.GetTag()
					name := t.Get("multipart")
					ty := element.TypeString
					prop := spec.NewSchemaSpec()
					if element.Item != nil || element.TypeString == "FileHeader" {
						v1 := spec.NewSingleOrArray[string]("string")
						prop.Spec.Format = "binary"
						prop.Spec.Type = &v1
						prop.Spec.Description = element.Comment
					} else {
						v1 := spec.NewSingleOrArray[string]("string")
						prop.Spec.Format = ty
						prop.Spec.Type = &v1
						prop.Spec.Description = element.Comment
					}

					sche.Spec.Properties[name] = prop
					md.Spec.Schema = sche
					body.Spec.Spec.Content["multipart/form-data"] = md

					op.Spec.RequestBody = body

				})
			case "FORM":
				//TODO
			case "HEADER":
				element.Item.VisitElements(astp.ElementField, func(element *astp.Element) bool {
					return !element.Private()
				}, func(element *astp.Element) {
					t := element.GetTag()
					name := t.Get("header")
					ty := element.TypeString
					param := spec.NewParameterSpec()

					param.Spec.Spec.Name = name
					param.Spec.Spec.Description = element.Comment
					param.Spec.Spec.In = "header"
					param.Spec.Spec.Required = true
					schema := spec.NewSchemaSpec()
					v := spec.NewSingleOrArray[string]("integer")
					schema.Spec.Type = &v
					schema.Spec.Format = ty
					param.Spec.Spec.Schema = schema

					op.Spec.Parameters = append(op.Spec.Parameters, param)

				})
			case "XML":
				//TODO
			case "PLAIN":
			}
		})

		op.Spec.Responses = spec.NewResponses()
		resp := spec.NewResponseSpec()
		resp.Spec.Spec.Description = "success"
		op.Spec.Responses.Spec.Response = make(map[string]*spec.RefOrSpec[spec.Extendable[spec.Response]])
		op.Spec.Responses.Spec.Response["200"] = resp

		if p, ok := oa.Spec.Paths.Spec.Paths[route]; ok {
			path = p
		}
		switch m {
		case "GET":
			path.Spec.Spec.Get = op
		case "POST":
			path.Spec.Spec.Post = op
		case "PUT":
			path.Spec.Spec.Put = op
		case "DELETE":
			path.Spec.Spec.Delete = op
		case "OPTIONS":
			path.Spec.Spec.Options = op
		default:
			path.Spec.Spec.Get = op
		}

		oa.Spec.Paths.Spec.Paths[route] = path

	})

}
func (oa *OpenAPI) handleParam(pf *astp.Element) {

	attr := attribute.GetLastAttr(pf)
	if attr.Name == "BODY" || attr.Name == "JSON" {
		name := pf.Item.TypeString
		sch := spec.NewSchemaSpec()
		v1 := spec.NewSingleOrArray[string]("object")
		sch.Spec.Type = &v1
		sch.Spec.Properties = make(map[string]*spec.RefOrSpec[spec.Schema])
		pf.Item.VisitElements(astp.ElementField, func(element *astp.Element) bool {
			return !element.Private()
		}, func(field *astp.Element) {
			prop := spec.NewSchemaSpec()
			v1 := spec.NewSingleOrArray[string]("string")
			prop.Spec.Type = &v1
			prop.Spec.Format = field.TypeString
			prop.Spec.Description = field.Comment
			t := field.GetTag()
			fname := t.Get("json")
			if fname == "" {
				fname = field.Name
			}
			sch.Spec.Properties[fname] = prop
		})
		oa.Spec.Components.Spec.Schemas[name] = sch

	}
}

func (oa *OpenAPI) Print(slot string) {

	switch slot {
	case fw.AfterListen:
		oa.WriteOut(oa.fileName)
		var so = new(fw.ServerOption)
		oa.s.Provide(so)
		style := pterm.NewStyle(pterm.FgLightGreen, pterm.Bold)
		style3 := pterm.NewStyle(pterm.FgLightWhite, pterm.Bold)
		style4 := pterm.NewStyle(pterm.FgWhite)
		style.Print("  ➜ ")
		style3.Printf("%10s", "ApiDoc: ")
		style4.Printf("http://%s:%d%s\n", so.IntranetIP, so.Port, so.BasePath+"/doc"+"/index.html")

	}
}

func (oa *OpenAPI) HandleServerInfo(si []string) {
	attrs := attribute.ParseDoc(si, "xxx")
	for _, attr := range attrs {
		if attr.Type == attribute.TypeDoc {
			switch strings.ToLower(attr.Name) {
			case "title":
				oa.Spec.Info.Spec.Title = attr.Value
			case "license":
				strs := strings.SplitN(attr.Value, " ", 3)
				oa.Spec.Info.Spec.License.Spec.Name = strs[0]
				oa.Spec.Info.Spec.License.Spec.URL = strs[1]
				oa.Spec.Info.Spec.License.Spec.Identifier = strs[2]
			case "description":
				oa.Spec.Info.Spec.Description = attr.Value
			case "contact":
				strs := strings.SplitN(attr.Value, " ", 3)
				oa.Spec.Info.Spec.Contact.Spec.Name = strs[0]
				oa.Spec.Info.Spec.Contact.Spec.URL = strs[1]
				oa.Spec.Info.Spec.Contact.Spec.Email = strs[2]
			case "version":
				oa.Spec.Info.Spec.Version = attr.Value
			case "summary":
				oa.Spec.Info.Spec.Summary = attr.Value
			case "termsofservice":
				oa.Spec.Info.Spec.TermsOfService = attr.Value
			}

		}
	}
}

func (oa *OpenAPI) WriteOut(file string) error {
	if filepath.Ext(file) == ".yaml" {
		bs, err := yaml.Marshal(oa)
		if err != nil {
			return err
		}
		err = os.WriteFile(file, bs, os.ModePerm)
		if err != nil {
			return err
		}
	} else {
		bs, err := oa.MarshalJSON()
		if err != nil {
			return err
		}
		err = os.WriteFile(file, bs, os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}
