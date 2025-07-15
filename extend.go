package fw_openapi

import (
	"fmt"
	"github.com/linxlib/astp/constants"
	"github.com/linxlib/astp/types"
	"github.com/linxlib/conv"
	spec "github.com/sv-tools/openapi"
	"reflect"
	"strings"
	"time"
)

func (oa *OpenAPI) getTagByName(tag reflect.StructTag, name string, tagName string) string {
	if tagContent := tag.Get(tagName); tagContent != "" {
		tmp := strings.ReplaceAll(tagContent, ",omitempty", "")
		return tmp
	} else {
		return name
	}
}
func (oa *OpenAPI) getComment(comments []*types.Comment) string {
	if len(comments) > 0 {
		return comments[0].Content
	} else {
		return ""
	}
}

func (oa *OpenAPI) NewParentFieldProp(f *types.Struct, tagName string) map[string]*spec.RefOrSpec[spec.Schema] {
	fields := make(map[string]*spec.RefOrSpec[spec.Schema])
	f.VisitFields(func(element *types.Field) bool {
		return !element.Private
	}, func(field *types.Field) {
		fieldName := field.Name
		fieldName = oa.getTagByName(field.GetTag(), fieldName, tagName)
		//TODO: 考虑多层嵌套的情况
		// 处理隐式导入的字段
		if fieldName == "-" || field.Name == constants.EmptyName {
			if field.Name == constants.EmptyName {
				fields1 := oa.NewParentFieldProp(field.Struct, tagName)
				for s, s2 := range fields1 {
					fields[s] = s2
				}
				return
			}
			return
		}

		defaultValue := ""
		defaultValue = oa.getTagByName(field.GetTag(), defaultValue, "default")
		comment := oa.getComment(field.Comment)
		example := ""
		example = oa.getTagByName(field.GetTag(), example, "example")

		fieldSchema := oa.NewFieldProp(field, tagName, defaultValue, comment, example)
		fields[fieldName] = fieldSchema
	})

	return fields
}
func getFormat(typeString string) string {
	switch typeString {
	case "int64", "uint64":
		return "int64"
	case "int32", "uint32", "int":
		return "int32"
	case "float64":
		return "double"
	case "float32":
		return "float"
	default:
		return ""
	}
}

func (oa *OpenAPI) NewFieldProp(f *types.Field,
	tagName string, defaultValue string, comment string, exampleValue string) *spec.RefOrSpec[spec.Schema] {

	typeString := f.Type
	var defVal, exampleVal any = defaultValue, exampleValue
	isObject := false
	isBinary := false
	format := ""

	if strings.Contains(typeString, "Decimal") {
		typeString = "number"
		format = "double"
		defVal = 0.0
		exampleVal = 0.0
	} else {
		switch typeString {
		case "string":
			typeString = "string"
			defVal = defaultValue
			exampleVal = exampleValue
		case "int", "int64", "uint", "uint64", "uint32", "int32":
			format = getFormat(typeString)
			typeString = "integer"
			if defaultValue == "" {
				defVal = 0
			} else {
				defVal = conv.Int(defaultValue)
			}
			if exampleValue == "" {
				exampleVal = 0
			} else {
				exampleVal = conv.Int(exampleValue)
			}
		case "bool":
			typeString = "boolean"
			if defaultValue == "" {
				defVal = false
			} else {
				defVal = conv.Bool(defaultValue)
			}
			if exampleValue == "" {
				exampleVal = false
			} else {
				exampleVal = conv.Bool(exampleValue)
			}
		case "float32", "float64":
			format = getFormat(typeString)
			typeString = "number"
			if defaultValue == "" {
				defVal = 0.00
			} else {
				defVal = conv.Float64(defaultValue)
			}
			if exampleValue == "" {
				exampleVal = 0.00
			} else {
				exampleVal = conv.Float64(exampleValue)
			}
		case "Time":
			if v, ok := f.GetTag().Lookup("time_format"); ok {
				switch v {
				case "unix":
					typeString = "integer"
					format = "int64"
					if defaultValue == "" {
						defVal = time.Now().Unix()
					}
					if exampleValue == "" {
						exampleVal = time.Now().Unix()
					}
				case "unixnano":
					typeString = "integer"
					format = "int64"
					if defaultValue == "" {
						defVal = time.Now().UnixNano()
					}
					if exampleValue == "" {
						exampleVal = time.Now().UnixNano()
					}
				}
			} else {
				typeString = "string"
				format = "date-time"
				if defaultValue == "" {
					defVal = time.Now().Format("2006-01-02 15:04:05")
				}
				if exampleValue == "" {
					exampleVal = time.Now().Format("2006-01-02 15:04:05")
				}
			}

		case "FileHeader":
			typeString = "string"
			isBinary = true
			format = "binary"
			isObject = true
			defVal = ""
			exampleVal = ""
		default:
			isObject = true
		}
	}
	builder := spec.NewSchemaBuilder()
	if f.Slice {
		var schema *spec.RefOrSpec[spec.Schema]
		if isObject {
			schema = oa.NewObjectProp(f.Struct, tagName)
		} else {
			builder1 := spec.NewSchemaBuilder()
			builder1.Type(typeString)
			if comment != "" {
				builder1.Comment(comment)
			}
			if defVal != "" {
				builder1.Default(defVal)
			}
			if exampleVal != "" {
				builder1.Example(exampleVal)
			}
			schema = builder1.Build()

		}
		builder.Type("array").Items(spec.NewBoolOrSchema(schema))

	} else {
		if isObject {
			var schema *spec.RefOrSpec[spec.Schema]
			if isBinary {
				schema = spec.NewSchemaBuilder().Type("string").Format(format).Build()
			} else {
				schema = oa.NewObjectProp(f.Struct, tagName)
			}

			return schema
		} else {
			builder.Type(typeString).Format(format).Description(comment)
			if defVal != "" {
				builder.Default(defVal)
			}
			if exampleVal != "" {
				builder.Example(exampleVal)
			}
		}

	}
	return builder.Build()
}

func (oa *OpenAPI) NewObjectProp(f *types.Struct, tagName string) *spec.RefOrSpec[spec.Schema] {
	if f.IsEnum() {
		doc := oa.getComment(f.Doc)
		builder := spec.NewSchemaBuilder().Type(f.Type).Comment(doc)
		description := ""
		var enumDesc = make([]*types.Comment, 0)
		for _, enum := range f.Enum.Enums {
			builder.AddEnum(enum.Value)
			comment := oa.getComment(enum.Comment)
			if comment != "" {
				comment = fmt.Sprintf("(%s)", comment)
			}
			description += fmt.Sprintf("- %s: %s%s\n", conv.String(enum.Value), enum.Name, comment)
			enumDesc = append(enumDesc, &types.Comment{
				Content: fmt.Sprintf("- %s: %s%s\n", conv.String(enum.Value), enum.Name, comment),
			})
		}
		f.Enum.Comment = enumDesc
		builder.Description(description)
		return builder.Build()
	} else {
		builder := spec.NewSchemaBuilder().Type("object")
		f.VisitFields(func(element *types.Field) bool {
			return !element.Private
		}, func(field *types.Field) {
			fieldName := field.Name
			fieldName = oa.getTagByName(field.GetTag(), fieldName, tagName)
			// 处理隐式导入的字段
			if fieldName == "-" || field.Name == constants.EmptyName {
				if field.Name == constants.EmptyName {
					fields := oa.NewParentFieldProp(field.Struct, tagName)
					for key, v := range fields {
						builder.AddProperty(key, v)
					}
					return
				}
				return
			}

			defaultValue := ""
			defaultValue = oa.getTagByName(field.GetTag(), defaultValue, "default")
			comment := oa.getComment(field.Comment)
			example := ""
			example = oa.getTagByName(field.GetTag(), example, "example")
			//
			if field.Parent && field.Struct != nil {
				if field.Generic {
					schema := oa.NewFieldProp(field, tagName, defaultValue, comment, example)
					builder.AddProperty(fieldName, schema)
				} else {
					fields := oa.NewParentFieldProp(field.Struct, tagName)
					builder.Properties(fields)
				}

			} else {
				schema := oa.NewFieldProp(field, tagName, defaultValue, comment, example)

				builder.AddProperty(fieldName, schema)
			}

		})
		return builder.Build()
	}

}
func (oa *OpenAPI) NewTag(tagName string, desc string) *spec.Extendable[spec.Tag] {
	return spec.NewTagBuilder().Name(tagName).Description(desc).Build()
}

func (oa *OpenAPI) NewObjectParameters(f *types.Struct, tagName string) []*spec.RefOrSpec[spec.Extendable[spec.Parameter]] {
	var parameters = make([]*spec.RefOrSpec[spec.Extendable[spec.Parameter]], 0)
	f.VisitFields(func(element *types.Field) bool {
		return !element.Private
	}, func(field *types.Field) {
		builder := spec.NewParameterBuilder()
		fieldName := field.Name
		fieldName = oa.getTagByName(field.GetTag(), fieldName, tagName)
		if fieldName == "-" || field.Name == constants.EmptyName {
			return
		}
		defaultValue := ""
		defaultValue = oa.getTagByName(field.GetTag(), defaultValue, "default")
		comment := oa.getComment(field.Comment)
		example := ""
		example = oa.getTagByName(field.GetTag(), example, "example")
		schema := oa.NewFieldProp(field, tagName, defaultValue, comment, example)
		if field.Struct.IsEnum() {
			comment += "\n"
			for _, c := range field.Struct.Enum.Comment {
				comment += c.Content + "\n"
			}

		}

		builder.Name(fieldName)
		builder.Description(comment)
		builder.In(tagName)
		builder.Schema(schema)
		parameters = append(parameters, builder.Build())
	})
	return parameters
}
