package fw_openapi

import (
	"github.com/sv-tools/openapi/spec"
	"strings"
)

func (oa *OpenAPI) NewStringProp(prop *spec.RefOrSpec[spec.Schema], typeString string) (isArray bool, sa spec.SingleOrArray[string]) {
	if strings.HasPrefix(typeString, "[]") {
		isArray = true
		prop.Spec.Items = spec.NewBoolOrSchema(true, nil)
		prop.Spec.Items.Schema = spec.NewSchemaSpec()
		tt := spec.NewSingleOrArray[string]("string")
		prop.Spec.Items.Schema.Spec.Type = &tt
		sa = spec.NewSingleOrArray[string]("array")
	} else {
		sa = spec.NewSingleOrArray[string]("string")
	}
	return
}
func (oa *OpenAPI) NewIntProp(prop *spec.RefOrSpec[spec.Schema], typeString string) (isArray bool, sa spec.SingleOrArray[string]) {

	switch typeString {
	case "int":
		sa = spec.NewSingleOrArray[string]("integer")
		prop.Spec.Format = "int32"
	case "int64":
		sa = spec.NewSingleOrArray[string]("integer")
		prop.Spec.Format = "int64"
	case "[]int":
		sa = spec.NewSingleOrArray[string]("array")
		prop.Spec.Items = spec.NewBoolOrSchema(true, nil)
		prop.Spec.Items.Schema = spec.NewSchemaSpec()
		tt := spec.NewSingleOrArray[string]("integer")
		prop.Spec.Items.Schema.Spec.Type = &tt
		prop.Spec.Items.Schema.Spec.Format = "int32"
		isArray = true
	case "[]int64":
		sa = spec.NewSingleOrArray[string]("array")
		prop.Spec.Items = spec.NewBoolOrSchema(true, nil)
		prop.Spec.Items.Schema = spec.NewSchemaSpec()
		tt := spec.NewSingleOrArray[string]("integer")
		prop.Spec.Items.Schema.Spec.Type = &tt
		prop.Spec.Items.Schema.Spec.Format = "int64"
		isArray = true
	}
	return

}

func (oa *OpenAPI) NewFloatProp(prop *spec.RefOrSpec[spec.Schema], typeString string) (isArray bool, sa spec.SingleOrArray[string]) {
	sa = spec.NewSingleOrArray[string]("number")
	switch typeString {
	case "float32", "float64":
		prop.Spec.Format = "float"
	case "[]float32", "[]float64":
		sa = spec.NewSingleOrArray[string]("array")
		prop.Spec.Items = spec.NewBoolOrSchema(true, nil)
		prop.Spec.Items.Schema = spec.NewSchemaSpec()
		tt := spec.NewSingleOrArray[string]("number")
		prop.Spec.Items.Schema.Spec.Type = &tt
		prop.Spec.Items.Schema.Spec.Format = "float"
		isArray = true
	}
	return
}
