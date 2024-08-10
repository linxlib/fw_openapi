package attribute

import (
	"github.com/linxlib/astp"
	"strings"
)

var innerAttrNames = map[string]AttributeType{
	"GET":        TypeHttpMethod,
	"POST":       TypeHttpMethod,
	"PUT":        TypeHttpMethod,
	"DELETE":     TypeHttpMethod,
	"HEAD":       TypeHttpMethod,
	"OPTIONS":    TypeHttpMethod,
	"TRACE":      TypeHttpMethod,
	"CONNECT":    TypeHttpMethod,
	"ANY":        TypeHttpMethod,
	"WS":         TypeHttpMethod,
	"Route":      TypeMiddleware,
	"Controller": TypeTagger,
	"Ctl":        TypeTagger,
	"Base":       TypeTagger,
	"Ignore":     TypeTagger,

	"Tag":        TypeDoc,
	"Deprecated": TypeDoc,

	"Body":           TypeParam,
	"Json":           TypeParam,
	"Path":           TypeParam,
	"Form":           TypeParam,
	"Header":         TypeParam,
	"Query":          TypeParam,
	"Cookie":         TypeParam,
	"XML":            TypeParam,
	"Multipart":      TypeParam,
	"Service":        TypeParam,
	"Plain":          TypeParam,
	"License":        TypeDoc,
	"Version":        TypeDoc,
	"Title":          TypeDoc,
	"Contact":        TypeDoc,
	"Description":    TypeDoc,
	"Summary":        TypeDoc,
	"TermsOfService": TypeDoc,
}

type AttributeType int

const (
	TypeHttpMethod AttributeType = iota //http 请求方法
	TypeOther                           //其他
	TypeDoc                             //注释内容
	TypeMiddleware                      //中间件类
	TypeParam                           //方法的参数和返回值专用
	TypeTagger                          //这种类型仅用于标记一些元素
	TypeInner
)

// Attribute 注解命令
type Attribute struct {
	Name  string
	Value string
	Type  AttributeType
	Index int
}

// ParseDoc 解析注解
func ParseDoc(doc []string, name string) []*Attribute {
	docs := make([]*Attribute, len(doc))
	if doc == nil {
		return docs
	}
	for j, s := range doc {
		if strings.HasPrefix(s, "@") {
			ps := strings.SplitN(s, " ", 2)
			value := ""
			if len(ps) == 2 {
				value = strings.TrimSpace(ps[1])
			}
			docName := strings.TrimLeft(ps[0], "@")
			docs[j] = &Attribute{
				Name:  docName,
				Value: value,
				Type:  innerAttrNames[docName],
			}
		} else if strings.HasPrefix(s, name) {
			ps := strings.SplitN(s, " ", 2)
			value := ""
			if len(ps) == 2 {
				value = strings.TrimSpace(ps[1])
			}
			docs[j] = &Attribute{
				Name:  name,
				Value: value,
				Type:  TypeDoc,
			}
		} else {
			docs[j] = &Attribute{
				Name:  name,
				Value: s,
				Type:  TypeDoc,
			}
		}

	}
	return docs
}

func GetFieldAttributeAsParamType(f *astp.Element) []*Attribute {
	results := make([]*Attribute, 0)
	if f.Item != nil {
		attrs := GetStructAttrs(f.Item)
		for _, attr := range attrs {
			if attr.Type == TypeParam {
				results = append(results, attr)
			}
		}
	} else {
		attr := new(Attribute)
		attr.Type = TypeInner
		attr.Name = f.Name
		attr.Value = ""
		attr.Index = 0
		results = append(results, attr)
	}
	return results
}

var cmdStructCaches = make(map[*astp.Element][]*Attribute)

func GetStructAttrs(s *astp.Element) []*Attribute {
	if cmdCache, ok := cmdStructCaches[s]; ok {
		return cmdCache
	}
	cmdCache := ParseDoc(s.Docs, s.Name)
	cmdStructCaches[s] = cmdCache
	return cmdCache
}

func HasAttribute(s *astp.Element, name string) bool {
	if s == nil {
		return false
	}
	if cmdCache, ok := cmdStructCaches[s]; ok {
		for _, cmd := range cmdCache {
			if cmd.Name == name {
				return true
			}
		}
	} else {
		cmdCache = GetStructAttrs(s)
		if cmdCache == nil || len(cmdCache) <= 0 {
			return false
		}
		return GetStructAttrByName(s, name) != nil
	}
	return false
}
func GetStructAttrByName(s *astp.Element, name string) *Attribute {
	if cmdCache, ok := cmdStructCaches[s]; ok {
		for _, cmd := range cmdCache {
			if cmd.Name == name {
				return cmd
			}
		}
	} else {
		cmdCache = GetStructAttrs(s)
		if cmdCache == nil || len(cmdCache) <= 0 {
			return nil
		}
		return GetStructAttrByName(s, name)
	}
	return nil
}

func GetStructAttrAsMiddleware(s *astp.Element) []*Attribute {
	results := make([]*Attribute, 0)
	attrs := GetStructAttrs(s)
	for _, attr := range attrs {
		if attr.Type == TypeMiddleware {
			results = append(results, attr)
		}
	}
	return results
}

var attrMethodCaches = make(map[*astp.Element][]*Attribute)

func GetMethodAttributes(m *astp.Element) []*Attribute {
	if cmdCache, ok := attrMethodCaches[m]; ok {
		return cmdCache
	}
	cmdCache := ParseDoc(m.Docs, m.Name)
	attrMethodCaches[m] = cmdCache
	return cmdCache
}

func GetMethodAttributesAsMiddleware(m *astp.Element) []*Attribute {
	results := make([]*Attribute, 0)
	attrs := GetMethodAttributes(m)
	for _, attr := range attrs {
		if attr.Type == TypeMiddleware {
			results = append(results, attr)
		}
	}
	return results
}

func GetLastAttr(f *astp.Element) *Attribute {
	as := GetFieldAttributeAsParamType(f)
	return as[len(as)-1]
}
