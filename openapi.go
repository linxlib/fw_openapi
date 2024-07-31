package fw_openapi

import (
	"github.com/linxlib/astp"
	"github.com/linxlib/fw"
	"github.com/linxlib/fw_openapi/attribute"
	"github.com/sv-tools/openapi/spec"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
)

func NewOpenAPIFromFWServer(s *fw.Server) *OpenAPI {
	oa := &OpenAPI{
		Extendable: spec.NewOpenAPI(),
	}
	s.RegisterHooks(oa)
	return oa
}

type OpenAPI struct {
	*spec.Extendable[spec.OpenAPI]
}

func (oa *OpenAPI) HandleServerInfo(si []string) {
	attrs := attribute.ParseDoc(si, "xxx")
	for _, attr := range attrs {
		if attr.Type == attribute.TypeDoc {
			switch attr.Name {
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

func (oa *OpenAPI) HandleStructs(ctl *astp.Struct) {
	//TODO implement me
	panic("implement me")
}

func (oa *OpenAPI) HandleParams(pf *astp.ParamField) {
	//TODO implement me
	panic("implement me")
}

func (oa *OpenAPI) WriteOut(file string) error {
	if filepath.Ext(file) != ".yaml" {
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
