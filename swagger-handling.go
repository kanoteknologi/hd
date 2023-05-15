package hd

import (
	"fmt"
	"go/parser"
	"go/token"
	"reflect"
	"strings"

	"git.kanosolution.net/kano/kaos"
)

func (hd *HttpDeployer) GenerateSwaggerSpec(service *kaos.Service, files []string, sourceCode, host, schemes string) (*APISpec, error) {
	res := NewSpec()

	for _, fileLoc := range files {
		fs := token.NewFileSet()
		f, err := parser.ParseFile(fs, fileLoc, nil, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("open file %s: %s", fileLoc, err.Error())
		}

		for _, fileComment := range f.Comments {
			commentTxts := strings.Split(fileComment.Text(), "\n")
			for _, comment := range commentTxts {
				translateComment(res, comment)
			}
		}
	}

	if sourceCode != "" {
		fs := token.NewFileSet()
		f, err := parser.ParseFile(fs, "", sourceCode, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("open source code: %s", err.Error())
		}

		for _, fileComment := range f.Comments {
			commentTxts := strings.Split(fileComment.Text(), "\n")
			for _, comment := range commentTxts {
				translateComment(res, comment)
			}
		}
	}

	res.BasePath = service.BasePoint()
	if !strings.HasPrefix(res.BasePath, "/") {
		res.BasePath = "/" + res.BasePath
	}
	res.Host = host
	res.Schemes = strings.Split(schemes, ",")

	routes := hd.routes
	for _, route := range routes {
		pathSpecFromRoute(route, res)
	}

	return res, nil
}

func pathSpecFromRoute(route *kaos.ServiceRoute, spec *APISpec) *Path {
	apiPath := spec.Path(route.Path, "post")
	apiPath.Produces = []string{"application/json"}
	apiPath.Consumes = []string{"application/json"}
	apiPath.Responses = make(map[string]*Response)

	uid := route.Fn.Type().String()
	apiPath.XGoReference = uid

	tFn := route.Fn.Type()
	in := tFn.In(tFn.NumIn() - 1)
	if in.Kind() == reflect.Ptr {
		in = in.Elem()
	}

	parameter := apiPath.Parameter("body")
	parameter.Name = "body"
	parameter.InputType = "body"
	parameter.SetFromType(in, spec)

	// responses
	outType := tFn.Out(0)
	resOK := apiPath.Response("200")
	resOK.Description = "Success"
	resOK.SetFromType(outType, spec)

	resError := apiPath.Response("500")
	resError.Description = "Error"

	return apiPath
}

func addDefinition(in reflect.Type, spec *APISpec) {
	schemaName := in.Name()
	schema, has := spec.Definitions[schemaName]
	if !has {
		schema = &Schema{}
	}
	schema.Type = "object"

	fieldIndex := 0
	fieldNum := in.NumField()
	schema.Properties = map[string]SchemaProperty{}
	for {
		if fieldIndex >= fieldNum {
			break
		}
		field := in.Field(fieldIndex)
		jsonFieldName := strings.Split(field.Tag.Get("json"), ",")[0]
		if jsonFieldName == "" {
			jsonFieldName = field.Name
		}
		prop := SchemaProperty{}
		switch field.Type.String() {
		case "string":
			prop.Type = "string"

		case "int", "int8", "int16", "int32", "int64":
			prop.Type = "integer"
			prop.Format = "int64"

		case "float32", "float64":
			prop.Type = "number"
			prop.Format = "float"

		case "time.Time", "*time.Time":
			prop.Type = "date"
			prop.Format = "date"

		default:
			prop.Ref = fmt.Sprintf("#/definitions/%s", field.Name)
		}
		schema.Properties[jsonFieldName] = prop
		fieldIndex++
	}
	spec.Definitions[schemaName] = schema
}

func translateComment(spec *APISpec, comment string) error {
	if !strings.HasPrefix(comment, "@") {
		return nil
	}

	tag := strings.Split(comment, " ")[0]
	switch tag {
	case "@swagger":
		spec.Swagger = stripTag(comment, tag)

	case "@title":
		spec.Info.Title = stripTag(comment, tag)

	case "@description":
		spec.Info.Description = stripTag(comment, tag)

	case "@version":
		spec.Info.Version = stripTag(comment, tag)

	case "@route":
		routeTxts := strings.Split(comment, " ")
		if len(routeTxts) < 3 {
			return fmt.Errorf("route is not valid. %s", comment)
		}

		apiPath := spec.Path(routeTxts[1], "")
		apiPath.Description = stripTag(comment, fmt.Sprintf("%s %s", tag, routeTxts[1]))

	case "@method":
		routeTxts := strings.Split(comment, " ")
		if len(routeTxts) < 3 {
			return fmt.Errorf("method is not valid. %s", comment)
		}

		routePath := routeTxts[1]
		method := routeTxts[2]
		apiPath := spec.Path(routePath, method)
		spec.SetPath(routePath, method, apiPath)

	case "@param":
		texts := strings.Split(comment, " ")
		if len(texts) < 5 {
			return fmt.Errorf("param is not valid. %s", comment)
		}
		routePath := texts[1]
		paramId := texts[2]
		paramType := texts[3]
		paramSource := texts[4]
		paramDesc := ""
		if len(texts) > 5 {
			paramDesc = strings.Join(texts[5:], " ")
		}

		apiPath := spec.Path(routePath, "")
		param := apiPath.Parameter(paramId)
		param.Description = paramDesc
		param.InputType = paramSource
		param.SetFromTypeStr(paramType, spec)
	}

	return nil
}

func stripTag(comment, tag string) string {
	tag = tag + " "
	if len(comment) < len(tag) {
		return ""
	}
	return comment[len(tag):]
}
