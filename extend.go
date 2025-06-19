package fw_openapi

import (
	"fmt"
	"github.com/linxlib/astp/constants"
	"github.com/linxlib/astp/types"
	"github.com/linxlib/conv"
	spec "github.com/sv-tools/openapi"
	"reflect"
	"strings"
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

func (oa *OpenAPI) NewFieldProp(f *types.Field,
	tagName string, defaultValue string, comment string, example string) *spec.RefOrSpec[spec.Schema] {

	typeString := f.Type
	isObject := false
	isBinary := false
	format := ""
	builder := spec.NewSchemaBuilder()

	if strings.Contains(typeString, "Decimal") {
		typeString = "number"
	} else {
		switch typeString {
		case "string":
			typeString = "string"
		case "int", "int64", "uint", "uint64", "uint32", "int32":
			typeString = "integer"
			format = typeString
		case "bool":
			typeString = "boolean"
		case "float32", "float64":
			typeString = "number"
		case "Time":
			typeString = "string"
			format = "date-time"
		case "FileHeader":
			typeString = "string"
			isBinary = true
			format = "binary"
			isObject = true
		default:
			isObject = true
		}
	}

	if f.Slice {
		var schema *spec.RefOrSpec[spec.Schema]
		if isObject {
			schema = oa.NewObjectProp(f.Struct, tagName)
		} else {
			schema = builder.Type(typeString).GoType(f.Type).Default(defaultValue).Example(example).Description(comment).Build()
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
			builder.Type(typeString).GoType(typeString).Default(defaultValue).Example(example).Description(comment)
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
				fields := oa.NewParentFieldProp(field.Struct, tagName)
				builder.Properties(fields)
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
