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
	oa.Spec.OpenAPI = "3.1.0"
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
func (oa *OpenAPI) Log(t string, msg string) {
	return
	//f, _ := os.OpenFile("openapi.log", os.O_WRONLY|os.O_APPEND, 0666)
	//defer f.Close()
	//f.WriteString(fmt.Sprintf("%s %s: %s\n", time.Now().Format(time.DateTime), t, msg))
}
func (oa *OpenAPI) checkParam(element *astp.Element) bool {
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
	if attribute.HasAttribute(element, "Cookie") || attribute.HasAttribute(element.Item, "Cookie") {
		return true
	}
	return false
}
func (oa *OpenAPI) NewSimpleParam(element *astp.Element, tag string) *spec.RefOrSpec[spec.Extendable[spec.Parameter]] {
	t := element.GetTag()
	name := t.Get(tag)
	if name == "" {
		name = element.Name
	}
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

	param := spec.NewParameterSpec()
	param.Spec.Spec.Name = name
	param.Spec.Spec.Description = element.Comment
	param.Spec.Spec.In = tag
	param.Spec.Spec.Required = isRequired

	schema, _ := oa.NewProp(element.TypeString, element.Comment)
	schema.Spec.Default = def
	param.Spec.Spec.Schema = schema
	return param
}
func (oa *OpenAPI) HandleStructs(ctl *astp.Element) {
	oa.Log("controller", "start "+ctl.Name)
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
		oa.Log("route", route)
		//fmt.Println(route)
		path := spec.NewPathItemSpec()

		op := spec.NewOperation()
		op.Spec.OperationID = ctl.Name + "." + method.Name
		op.Spec.Summary = desc
		op.Spec.Tags = []string{tagName}
		//params
		method.VisitElements(astp.ElementParam, oa.checkParam, func(element *astp.Element) {
			oa.Log("params", element.TypeString)
			oa.handleParam(element)
			attr := attribute.GetLastAttr(element)
			switch attr.Name {
			case "BODY", "JSON":
				body := spec.NewRequestBodySpec()
				if op.Spec.RequestBody != nil {
					body = op.Spec.RequestBody
				}
				body.Spec.Spec.Required = true
				if body.Spec.Spec.Content == nil {
					body.Spec.Spec.Content = make(map[string]*spec.Extendable[spec.MediaType])
				}
				md := spec.NewMediaType()
				sche := spec.NewRefOrSpec[spec.Schema](spec.NewRef("#/components/schemas/"+element.Item.TypeString), nil)
				md.Spec.Schema = sche
				body.Spec.Spec.Content["application/json"] = md

				op.Spec.RequestBody = body

			case "PATH":
				element.Item.VisitElements(astp.ElementField, func(element *astp.Element) bool {
					return !element.Private()
				}, func(element *astp.Element) {
					param := oa.NewSimpleParam(element, "path")
					op.Spec.Parameters = append(op.Spec.Parameters, param)
				})

			case "QUERY":

				element.Item.VisitElements(astp.ElementField, func(element *astp.Element) bool {
					return !element.Private()
				}, func(element *astp.Element) {

					param := oa.NewSimpleParam(element, "query")
					op.Spec.Parameters = append(op.Spec.Parameters, param)

				})
			case "MULTIPART":
				body := spec.NewRequestBodySpec()
				if op.Spec.RequestBody != nil {
					body = op.Spec.RequestBody
				}

				body.Spec.Spec.Required = true
				if body.Spec.Spec.Content == nil {
					body.Spec.Spec.Content = make(map[string]*spec.Extendable[spec.MediaType])
				}
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
					if name == "" {
						name = element.Name
					}
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
				body := spec.NewRequestBodySpec()
				if op.Spec.RequestBody != nil {
					body = op.Spec.RequestBody
				}
				body.Spec.Spec.Required = true
				if body.Spec.Spec.Content == nil {
					body.Spec.Spec.Content = make(map[string]*spec.Extendable[spec.MediaType])
				}

				md := spec.NewMediaType()
				sche := spec.NewRefOrSpec[spec.Schema](spec.NewRef("#/components/schemas/"+element.Item.TypeString), nil)
				md.Spec.Schema = sche
				body.Spec.Spec.Content["application/x-www-form-urlencoded"] = md

				op.Spec.RequestBody = body
			case "HEADER":
				element.Item.VisitElements(astp.ElementField, func(element *astp.Element) bool {
					return !element.Private()
				}, func(element *astp.Element) {
					param := oa.NewSimpleParam(element, "header")
					op.Spec.Parameters = append(op.Spec.Parameters, param)

				})
			case "COOKIE":
				element.Item.VisitElements(astp.ElementField, func(element *astp.Element) bool {
					return !element.Private()
				}, func(element *astp.Element) {
					param := oa.NewSimpleParam(element, "cookie")
					op.Spec.Parameters = append(op.Spec.Parameters, param)
				})
			case "XML":
				body := spec.NewRequestBodySpec()
				if op.Spec.RequestBody != nil {
					body = op.Spec.RequestBody
				}
				body.Spec.Spec.Required = true
				if body.Spec.Spec.Content == nil {
					body.Spec.Spec.Content = make(map[string]*spec.Extendable[spec.MediaType])
				}
				md := spec.NewMediaType()
				sche := spec.NewRefOrSpec[spec.Schema](spec.NewRef("#/components/schemas/"+element.Item.TypeString), nil)
				md.Spec.Schema = sche
				body.Spec.Spec.Content["application/xml"] = md

				op.Spec.RequestBody = body
			case "PLAIN":
				body := spec.NewRequestBodySpec()
				if op.Spec.RequestBody != nil {
					body = op.Spec.RequestBody
				}
				body.Spec.Spec.Required = true
				if body.Spec.Spec.Content == nil {
					body.Spec.Spec.Content = make(map[string]*spec.Extendable[spec.MediaType])
				}
				md := spec.NewMediaType()
				sche := spec.NewSingleOrArray[string]("string")
				md.Spec.Schema.Spec.Type = &sche
				body.Spec.Spec.Content["text/plain"] = md

				op.Spec.RequestBody = body
			}
		})
		op.Spec.Responses = spec.NewResponses()
		op.Spec.Responses.Spec.Response = make(map[string]*spec.RefOrSpec[spec.Extendable[spec.Response]])
		method.VisitElements(astp.ElementResult, func(element *astp.Element) bool {
			return true
		}, func(element *astp.Element) {
			oa.Log("results", element.TypeString)
			if element.ElementType != astp.ElementStruct {
				if element.TypeString == "error" {
					oa.Log("results", "add 500")
					resp := oa.NewStringResponse("fail", "text/plain")
					op.Spec.Responses.Spec.Response["500"] = resp
				} else {
					oa.Log("results", "add 200 empty not struct")
					resp := oa.NewResponse("success", element.TypeString, "text/plain")
					op.Spec.Responses.Spec.Response["200"] = resp
					return
				}

			}
			if element.Item == nil {
				if op.Spec.Responses.Spec.Response["200"] == nil {
					oa.Log("results", "add 200 empty element.Item == nil")
					resp := oa.NewResponse("success", element.TypeString, "text/plain")
					op.Spec.Responses.Spec.Response["200"] = resp
				}

				return
			}
			oa.handleResults(element)
			//fmt.Println(element.String())
			oa.Log("results", "add 200 object ")
			resp := oa.NewObjectResponse(element.ElementString, "success", "application/json")

			op.Spec.Responses.Spec.Response["200"] = resp
		})
		if len(op.Spec.Responses.Spec.Response) == 0 || op.Spec.Responses.Spec.Response["200"] == nil {
			oa.Log("results", "add 200 object default")
			resp := oa.NewStringResponse("success", "text/plain")
			op.Spec.Responses.Spec.Response["200"] = resp
		}

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
func (oa *OpenAPI) NewObjectResponse(schemaName string, desc string, contentType string) *spec.RefOrSpec[spec.Extendable[spec.Response]] {
	resp := spec.NewResponseSpec()
	resp.Spec.Spec.Description = desc
	resp.Spec.Spec.Content = make(map[string]*spec.Extendable[spec.MediaType])
	md := spec.NewMediaType()
	sche := spec.NewRefOrSpec[spec.Schema](spec.NewRef("#/components/schemas/"+schemaName), nil)
	md.Spec.Schema = sche
	resp.Spec.Spec.Content[contentType] = md
	return resp

}

func (oa *OpenAPI) NewStringResponse(desc string, contentType string) *spec.RefOrSpec[spec.Extendable[spec.Response]] {
	resp := spec.NewResponseSpec()
	resp.Spec.Spec.Description = desc
	resp.Spec.Spec.Content = make(map[string]*spec.Extendable[spec.MediaType])
	md := spec.NewMediaType()
	sch := spec.NewSchemaSpec()
	v1 := spec.NewSingleOrArray[string]("string")
	sch.Spec.Type = &v1
	sch.Spec.Properties = make(map[string]*spec.RefOrSpec[spec.Schema])
	md.Spec.Schema = sch
	resp.Spec.Spec.Content[contentType] = md
	return resp
}
func (oa *OpenAPI) NewResponse(desc string, v string, contentType string) *spec.RefOrSpec[spec.Extendable[spec.Response]] {
	switch v {
	case "int":
		v = "integer"
	case "int64":
		v = "integer"
	case "float64":
		v = "number"
	case "float32":
		v = "number"
	case "error":
		v = "string"
	}

	resp := spec.NewResponseSpec()
	resp.Spec.Spec.Description = desc
	resp.Spec.Spec.Content = make(map[string]*spec.Extendable[spec.MediaType])
	md := spec.NewMediaType()
	sch := spec.NewSchemaSpec()
	v1 := spec.NewSingleOrArray[string](v)
	sch.Spec.Type = &v1
	sch.Spec.Properties = make(map[string]*spec.RefOrSpec[spec.Schema])
	md.Spec.Schema = sch
	resp.Spec.Spec.Content[contentType] = md
	return resp
}
func (oa *OpenAPI) AddResponse(op *spec.Extendable[spec.Operation], code string, resp *spec.RefOrSpec[spec.Extendable[spec.Response]]) {
	op.Spec.Responses.Spec.Response[code] = resp
}

func (oa *OpenAPI) handleParam(pf *astp.Element) {

	attr := attribute.GetLastAttr(pf)
	switch attr.Name {
	case "BODY", "JSON":
		name := pf.Item.TypeString
		sch := spec.NewSchemaSpec()
		v1 := spec.NewSingleOrArray[string]("object")
		sch.Spec.Type = &v1
		sch.Spec.Properties = make(map[string]*spec.RefOrSpec[spec.Schema])
		pf.Item.VisitElements(astp.ElementField, func(element *astp.Element) bool {
			return !element.Private()
		}, func(field *astp.Element) {
			prop, tmp := oa.NewProp(field.TypeString, field.Comment)
			if tmp {
				name := field.Name
				prop.Ref = spec.NewRef("#/components/schemas/" + name)
				attr := attribute.GetStructAttrByName(field, name)
				if attr == nil {
					attr = &attribute.Attribute{}
				}
				sch1 := oa.NewObjectSchema(attr.Value)
				field.VisitElements(astp.ElementField, func(element *astp.Element) bool {
					return !element.Private()
				}, func(field *astp.Element) {
					// TODO: 多层嵌套需要递归
					prop, _ := oa.NewProp(field.TypeString, field.Comment)
					t := field.GetTag()
					fname := t.Get("json")
					if fname == "-" {
						return
					}
					if fname == "" {
						fname = field.Name
					}
					sch1.Spec.Properties[fname] = prop
				})
				oa.Spec.Components.Spec.Schemas[name] = sch1

			}
			t := field.GetTag()
			fname := t.Get("json")
			if fname == "-" {
				return
			}
			if fname == "" {
				fname = field.Name
			}
			sch.Spec.Properties[fname] = prop
		})
		oa.Spec.Components.Spec.Schemas[name] = sch
	case "XML":
		name := pf.Item.TypeString
		sch := spec.NewSchemaSpec()
		v1 := spec.NewSingleOrArray[string]("object")
		sch.Spec.Type = &v1
		sch.Spec.Properties = make(map[string]*spec.RefOrSpec[spec.Schema])
		pf.Item.VisitElements(astp.ElementField, func(element *astp.Element) bool {
			return !element.Private()
		}, func(field *astp.Element) {
			prop, tmp := oa.NewProp(field.TypeString, field.Comment)
			if tmp {
				name := field.Name
				prop.Ref = spec.NewRef("#/components/schemas/" + name)
				attr := attribute.GetStructAttrByName(field, name)
				if attr == nil {
					attr = &attribute.Attribute{}
				}
				sch1 := oa.NewObjectSchema(attr.Value)
				field.VisitElements(astp.ElementField, func(element *astp.Element) bool {
					return !element.Private()
				}, func(field *astp.Element) {
					// TODO: 多层嵌套需要递归
					prop, _ := oa.NewProp(field.TypeString, field.Comment)
					t := field.GetTag()
					fname := t.Get("xml")
					if fname == "-" {
						return
					}
					if fname == "" {
						fname = field.Name
					}
					sch1.Spec.Properties[fname] = prop
				})
				oa.Spec.Components.Spec.Schemas[name] = sch1

			}
			t := field.GetTag()
			fname := t.Get("xml")
			if fname == "-" {
				return
			}
			if fname == "" {
				fname = field.Name
			}
			sch.Spec.Properties[fname] = prop
		})
		oa.Spec.Components.Spec.Schemas[name] = sch
	case "FORM":
		name := pf.Item.TypeString
		sch := spec.NewSchemaSpec()
		v1 := spec.NewSingleOrArray[string]("object")
		sch.Spec.Type = &v1
		sch.Spec.Properties = make(map[string]*spec.RefOrSpec[spec.Schema])
		pf.Item.VisitElements(astp.ElementField, func(element *astp.Element) bool {
			return !element.Private()
		}, func(field *astp.Element) {
			prop, tmp := oa.NewProp(field.TypeString, field.Comment)
			if tmp {
				name := field.Name
				prop.Ref = spec.NewRef("#/components/schemas/" + name)
				attr := attribute.GetStructAttrByName(field, name)
				if attr == nil {
					attr = &attribute.Attribute{}
				}
				sch1 := oa.NewObjectSchema(attr.Value)
				field.VisitElements(astp.ElementField, func(element *astp.Element) bool {
					return !element.Private()
				}, func(field *astp.Element) {
					// TODO: 多层嵌套需要递归
					prop, _ := oa.NewProp(field.TypeString, field.Comment)
					t := field.GetTag()
					fname := t.Get("form")
					if fname == "-" {
						return
					}
					if fname == "" {
						fname = field.Name
					}
					sch1.Spec.Properties[fname] = prop
				})
				oa.Spec.Components.Spec.Schemas[name] = sch1

			}
			t := field.GetTag()
			fname := t.Get("form")
			if fname == "-" {
				return
			}
			if fname == "" {
				fname = field.Name
			}
			sch.Spec.Properties[fname] = prop
		})
		oa.Spec.Components.Spec.Schemas[name] = sch
	}

}
func (oa *OpenAPI) NewObjectSchema(comment string) *spec.RefOrSpec[spec.Schema] {
	sch := spec.NewSchemaSpec()
	v1 := spec.NewSingleOrArray[string]("object")
	sch.Spec.Type = &v1
	sch.Spec.Description = comment
	sch.Spec.Properties = make(map[string]*spec.RefOrSpec[spec.Schema])
	return sch
}

func (oa *OpenAPI) NewProp(v string, desc string) (*spec.RefOrSpec[spec.Schema], bool) {
	prop := spec.NewSchemaSpec()
	var v1 spec.SingleOrArray[string]
	var tmp bool
	if strings.Contains(v, "string") {

		switch v {
		case "[]string":
			v1 = spec.NewSingleOrArray[string]("array")
			prop.Spec.Items.Schema = spec.NewSchemaSpec()
			tt := spec.NewSingleOrArray[string]("string")
			prop.Spec.Items.Schema.Spec.Type = &tt
		case "string":
			v1 = spec.NewSingleOrArray[string]("string")
		}
	} else if strings.Contains(v, "int") {
		v1 = spec.NewSingleOrArray[string]("integer")
		switch v {
		case "int":
			prop.Spec.Format = "int32"
		case "int64":
			prop.Spec.Format = "int64"
		case "[]int":
			v1 = spec.NewSingleOrArray[string]("array")
			prop.Spec.Items.Schema = spec.NewSchemaSpec()
			tt := spec.NewSingleOrArray[string]("integer")
			prop.Spec.Items.Schema.Spec.Type = &tt
			prop.Spec.Items.Schema.Spec.Format = "int32"
		case "[]int64":
			v1 = spec.NewSingleOrArray[string]("array")
			prop.Spec.Items.Schema = spec.NewSchemaSpec()
			tt := spec.NewSingleOrArray[string]("integer")
			prop.Spec.Items.Schema.Spec.Type = &tt
			prop.Spec.Items.Schema.Spec.Format = "int64"
		}
	} else if strings.Contains(v, "float") {
		v1 = spec.NewSingleOrArray[string]("number")
		switch v {
		case "float32", "float64":
			prop.Spec.Format = "float"
		case "[]float32", "[]float64":
			v1 = spec.NewSingleOrArray[string]("array")
			prop.Spec.Items.Schema = spec.NewSchemaSpec()
			tt := spec.NewSingleOrArray[string]("number")
			prop.Spec.Items.Schema.Spec.Type = &tt
			prop.Spec.Items.Schema.Spec.Format = "float"
		}
	} else if strings.Contains(v, "bool") {
		v1 = spec.NewSingleOrArray[string]("boolean")
	} else if strings.Contains(v, "Time") {
		v1 = spec.NewSingleOrArray[string]("string")
		prop.Spec.Format = "date" // or date-time
	} else {
		if strings.HasPrefix(v, "[]") {
			v1 = spec.NewSingleOrArray[string]("array")
			prop.Spec.Items.Schema = spec.NewSchemaSpec()
			tt := spec.NewSingleOrArray[string]("object")
			prop.Spec.Items.Schema.Spec.Type = &tt
		} else {
			v1 = spec.NewSingleOrArray[string]("object")
		}

		tmp = true
	}
	prop.Spec.Description = desc
	prop.Spec.Type = &v1
	return prop, tmp
}

func (oa *OpenAPI) handleResults(pf *astp.Element) {
	schemaName := pf.Item.TypeString
	attr := attribute.GetStructAttrByName(pf.Item, schemaName)
	schemaName = pf.ElementString
	sch := oa.NewObjectSchema(attr.Value)
	pf.Item.VisitElements(astp.ElementField, func(element *astp.Element) bool {
		return !element.Private()
	}, func(field *astp.Element) {
		prop, tmp := oa.NewProp(field.TypeString, field.Comment)
		if tmp {
			name := field.Name
			prop.Ref = spec.NewRef("#/components/schemas/" + name)
			attr := attribute.GetStructAttrByName(field, name)
			sch1 := oa.NewObjectSchema(attr.Value)
			field.VisitElements(astp.ElementField, func(element *astp.Element) bool {
				return !element.Private()
			}, func(field *astp.Element) {
				// TODO: 多层嵌套需要递归
				prop, _ := oa.NewProp(field.TypeString, field.Comment)
				t := field.GetTag()
				fname := t.Get("json")
				if fname == "-" {
					return
				}
				if fname == "" {
					fname = field.Name
				}
				sch1.Spec.Properties[fname] = prop
			})
			oa.Spec.Components.Spec.Schemas[name] = sch1

		}

		t := field.GetTag()
		fname := t.Get("json")
		if fname == "-" {
			return
		}
		if fname == "" {
			fname = field.Name
		}
		sch.Spec.Properties[fname] = prop
	})

	oa.Spec.Components.Spec.Schemas[schemaName] = sch
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
