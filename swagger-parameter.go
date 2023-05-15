package hd

import (
	"fmt"
	"reflect"
	"strings"
)

func (p *Parameter) SetFromTypeStr(typeStr string, spec *APISpec) {
	switch typeStr {
	case "string":
		if p.InputType == "body" {
			p.Schema = &Schema{Type: "string", Format: ""}
			p.DataType = ""
			p.Format = ""
			return
		}

		p.DataType = "string"
		p.Format = ""
		p.Schema = nil

	case "int", "int8", "int16", "int32", "int64":
		if p.InputType == "body" {
			p.Schema = &Schema{Type: "integer", Format: "int64"}
			p.DataType = ""
			p.Format = ""
			return
		}

		p.DataType = "integer"
		p.Format = "int64"
		p.Schema = nil

	case "float32", "float64":
		if p.InputType == "body" {
			p.Schema = &Schema{Type: "number", Format: "float64"}
			p.DataType = ""
			p.Format = ""
			return
		}

		p.DataType = "number"
		p.Format = "float"
		p.Schema = nil

	default:
		p.DataType = ""
		p.Format = ""
		if strings.HasPrefix(typeStr, "[]") {
			p.Schema = &Schema{
				Type: "array",
				Ref:  fmt.Sprintf("#/definitions/%s", typeStr[3:]),
			}
			return
		}
		p.Schema = &Schema{
			Type: "object",
			Ref:  fmt.Sprintf("#/definitions/%s", typeStr),
		}
	}
}

func (p *Parameter) SetFromType(rt reflect.Type, spec *APISpec) {
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	inType := rt.Kind()
	switch inType {
	case reflect.String:
		if p.InputType == "body" {
			p.Schema = &Schema{Type: "string"}
			p.DataType = ""
			p.Format = ""
			return
		}

		p.DataType = "string"
		p.Format = ""
		p.Schema = nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if p.InputType == "body" {
			p.Schema = &Schema{Type: "integer", Format: "int64"}
			p.DataType = ""
			p.Format = ""
			return
		}

		p.DataType = "integer"
		p.Format = "int64"
		p.Schema = nil

	case reflect.Float32, reflect.Float64:
		if p.InputType == "body" {
			p.Schema = &Schema{Type: "number", Format: "float"}
			p.DataType = ""
			p.Format = ""
			return
		}

		p.DataType = "number"
		p.Format = "float"
		p.Schema = nil

	case reflect.Struct, reflect.Map:
		p.DataType = ""
		p.Format = ""
		p.Schema = &Schema{
			Type: "object",
			Ref:  fmt.Sprintf("#/definitions/%s", rt.Name()),
		}
		addDefinition(rt, spec)

	case reflect.Slice:
		p.DataType = ""
		p.Format = ""
		p.Schema = &Schema{
			Type: "array",
			Ref:  fmt.Sprintf("#/definitions/%s", rt.Name()),
		}
		addDefinition(rt.Elem(), spec)
	}
}
