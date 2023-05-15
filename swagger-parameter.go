package hd

import (
	"fmt"
	"reflect"
)

func (p *Parameter) SetFromType(rt reflect.Type, spec *APISpec) {
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	inType := rt.Kind()
	switch inType {
	case reflect.String:
		p.Schema = Schema{
			Type: "string",
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p.Schema = Schema{
			Type:   "integer",
			Format: "int64",
		}

	case reflect.Float32, reflect.Float64:
		p.Schema = Schema{
			Type:   "number",
			Format: "float64",
		}

	case reflect.Struct, reflect.Map:
		p.Schema = Schema{
			Type: "object",
			Ref:  fmt.Sprintf("#/definitions/%s", rt.Name()),
		}
		addDefinition(rt, spec)

	case reflect.Slice:
		p.Schema = Schema{
			Type: "array",
			Ref:  fmt.Sprintf("#/definitions/%s", rt.Name()),
		}
		addDefinition(rt.Elem(), spec)
	}
}
