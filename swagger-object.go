package hd

type XValues map[string]string

type APISpec struct {
	Swagger     string                      `json:"swagger" yaml:"swagger"`
	Info        Info                        `json:"info" yaml:"info"`
	BasePath    string                      `json:"basePath" yaml:"basePath"`
	Host        string                      `json:"host" yaml:"host"`
	Schemes     []string                    `json:"schemes" yaml:"schemes"`
	Definitions map[string]*Schema          `json:"definitions" yaml:"definitions"`
	Paths       map[string]map[string]*Path `json:"paths" yaml:"paths"`
}

func NewSpec() *APISpec {
	spec := new(APISpec)
	spec.Paths = map[string]map[string]*Path{}
	spec.Definitions = make(map[string]*Schema)
	return spec
}

type Components struct {
	Schemas map[string]Schema `json:"schemas,omitempty"`
}

type Info struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version,omitempty"`
}

type Path struct {
	Description  string               `json:"description,omitempty" yaml:"description"`
	Produces     []string             `json:"produces,omitempty" yaml:"produces"`
	Consumes     []string             `json:"consumes,omitempty" yaml:"produces"`
	Parameters   []*Parameter         `json:"parameters,omitempty"`
	Responses    map[string]*Response `json:"responses,omitempty" yaml:"responses"`
	XGoReference string               `json:"x-unique-id,omitempty" yaml:"x-unique-id"`
}

type Parameter struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	InputType   string `json:"in,omitempty"`
	DataType    string `json:"type,omitempty"`
	Format      string `json:"format,omitempty"`
	Schema      Schema `json:"schema,omitempty"`
}

type RequestBody struct {
	Description string                 `json:"description,omitempty"`
	Content     map[string]BodyContent `json:"content,omitempty"`
}

type BodyContent struct {
	Schema Schema `json:"schema,omitempty"`
}

type Schema struct {
	Type       string                    `json:"type,omitempty"`
	Items      map[string]interface{}    `json:"items,omitempty"`
	Format     string                    `json:"format,omitempty"`
	Ref        string                    `json:"$ref,omitempty"`
	Properties map[string]SchemaProperty `json:"properties,omitempty"`
}

type SchemaProperty struct {
	Type   string `json:"type,omitempty"`
	Format string `json:"format,omitempty"`
	Ref    string `json:"$ref,omitempty"`
}

type Security struct {
}

type InputTypeEnum string

const (
	InPath InputTypeEnum = "path"
	InForm InputTypeEnum = "dataForm"
)

type Response struct {
	Description string `json:"description,omitempty" yaml:"description"`
	Schema      Schema `json:"schema,omitempty"`
}

type SecurityDefinition struct {
}

type Tag struct {
}

type Definition struct {
	DefinitionType string
	Required       []string
	Properties     []DefinitionProperty
	Ref            string
	XValues        XValues
}

type DefinitionProperty struct {
	Name        string
	Description string
	InputType   string
	Required    bool
	DataType    string
	Format      string
}
