// This file was automatically generated based on the contents of *.tmpl
// If you need to update this file, change the contents of those files
// (or add new ones) and run 'go generate'

package swaggering

import "golang.org/x/tools/godoc/vfs/mapfs"

var Templates = mapfs.New(map[string]string{
	`api.tmpl`: "package {{.BasePackageName}}\n\nimport \"{{.PackageImportName}}/dtos\"\n\n{{range .Operations}}\n  {{template \"operation\" .}}\n{{end}}\n",
	`model.tmpl`: "package dtos\n\nimport (\n  \"fmt\"\n  \"io\"\n\n  \"github.com/opentable/swaggering\"\n)\n\n{{range $enum := .Enums}}\ntype {{$enum.Name}} string\n\nconst (\n  {{- range $value := $enum.Values}}\n  {{$enum.Name}}{{$value}} {{$enum.Name}} = \"{{$value}}\"\n  {{- end}}\n)\n{{end}}\n\ntype {{.GoName}} struct {\n  present map[string]bool\n{{range $name, $prop := .Properties}}\n  {{if $prop.GoTypeInvalid}}// {{end -}}\n  {{.GoName}} {{.GoTypePrefix}}\n  {{- if ne $.GoPackage .GoPackage}}{{.GoPackage}}\n    {{- if ne .GoPackage \"\" }}.{{end -}}\n  {{end -}}\n  {{.GoBaseType}} `json:\"{{$prop.SwaggerName}}\n  {{- if eq $prop.GoBaseType \"string\" -}}\n  ,omitempty\n  {{- end -}}\n  \"`\n{{end}}\n}\n\nfunc (self *{{.GoName}}) Populate(jsonReader io.ReadCloser) (err error) {\n	return swaggering.ReadPopulate(jsonReader, self)\n}\n\nfunc (self *{{.GoName}}) Absorb(other swaggering.DTO) error {\n  if like, ok := other.(*{{.GoName}}); ok {\n    *self = *like\n    return nil\n  }\n  return fmt.Errorf(\"A {{.GoName}} cannot absorb the values from %v\", other)\n}\n\nfunc (self *{{.GoName}}) MarshalJSON() ([]byte, error) {\n  return swaggering.MarshalJSON(self)\n}\n\nfunc (self *{{.GoName}}) FormatText() string {\n	return swaggering.FormatText(self)\n}\n\nfunc (self *{{.GoName}}) FormatJSON() string {\n	return swaggering.FormatJSON(self)\n}\n\nfunc (self *{{.GoName}}) FieldsPresent() []string {\n  return swaggering.PresenceFromMap(self.present)\n}\n\nfunc (self *{{.GoName}}) SetField(name string, value interface{}) error {\n  if self.present == nil {\n    self.present = make(map[string]bool)\n  }\n  switch name {\n  default:\n    return fmt.Errorf(\"No such field %s on {{.GoName}}\", name)\n  {{range $name, $prop := .Properties}}\n    {{ if not $prop.GoTypeInvalid }}\n    case \"{{$prop.SwaggerName}}\", \"{{$prop.GoName}}\":\n    v, ok := value.(\n      {{- .GoTypePrefix}}\n      {{- if ne $.GoPackage .GoPackage}}{{.GoPackage}}\n        {{- if ne .GoPackage \"\" }}.{{end -}}\n      {{end }} {{.GoBaseType -}}\n      )\n      if ok {\n        self.{{$prop.GoName}} = v\n        self.present[\"{{$prop.SwaggerName}}\"] = true\n        return nil\n      } else {\n        return fmt.Errorf(\"Field {{$prop.SwaggerName}}/{{$prop.GoName}}: value %v(%T) couldn't be cast to type {{$prop.GoTypePrefix}}{{$prop.GoBaseType}}\", value, value)\n      }\n    {{end}}\n  {{end}}\n  }\n}\n\nfunc (self *{{.GoName}}) GetField(name string) (interface{}, error) {\n  switch name {\n  default:\n    return nil, fmt.Errorf(\"No such field %s on {{.GoName}}\", name)\n  {{range $name, $prop := .Properties}}\n    {{ if not $prop.GoTypeInvalid }}\n    case \"{{$prop.SwaggerName}}\", \"{{$prop.GoName}}\":\n    if self.present != nil {\n      if _, ok := self.present[\"{{$prop.SwaggerName}}\"]; ok {\n        return self.{{$prop.GoName}}, nil\n      }\n    }\n    return nil, fmt.Errorf(\"Field {{$prop.GoName}} no set on {{.GoName}} %+v\", self)\n    {{end}}\n  {{end}}\n  }\n}\n\nfunc (self *{{.GoName}}) ClearField(name string) error {\n  if self.present == nil {\n    self.present = make(map[string]bool)\n  }\n  switch name {\n  default:\n    return fmt.Errorf(\"No such field %s on {{.GoName}}\", name)\n  {{range $name, $prop := .Properties}}\n    {{ if not $prop.GoTypeInvalid }}\n  case \"{{$prop.SwaggerName}}\", \"{{$prop.GoName}}\":\n    self.present[\"{{$prop.SwaggerName}}\"] = false\n    {{end}}\n  {{end}}\n  }\n\n  return nil\n}\n\nfunc (self *{{.GoName}}) LoadMap(from map[string]interface{}) error {\n  return swaggering.LoadMapIntoDTO(from, self)\n}\n\ntype {{.GoName}}List []*{{.GoName}}\n\nfunc (self *{{.GoName}}List) Absorb(other swaggering.DTO) error {\n  if like, ok := other.(*{{.GoName}}List); ok {\n    *self = *like\n    return nil\n  }\n  return fmt.Errorf(\"A {{.GoName}} cannot absorb the values from %v\", other)\n}\n\n\nfunc (list *{{.GoName}}List) Populate(jsonReader io.ReadCloser) (err error) {\n	return swaggering.ReadPopulate(jsonReader, list)\n}\n\nfunc (list *{{.GoName}}List) FormatText() string {\n	text := []byte{}\n	for _, dto := range *list {\n		text = append(text, (*dto).FormatText()...)\n		text = append(text, \"\\n\"...)\n	}\n	return string(text)\n}\n\nfunc (list *{{.GoName}}List) FormatJSON() string {\n	return swaggering.FormatJSON(list)\n}\n",
	`operation.tmpl`: "{{- if not .GoTypeInvalid -}}\nfunc (client *Client) {{.GoMethodName}}(\n{{- range .Parameters -}}\n{{.Name}} {{template \"type\" . -}}\n, {{end -}}\n) ({{ if not (eq .GoBaseType \"\") -}}\nresponse {{template \"type\" . -}}\n, {{end}} err error) {\n	pathParamMap := map[string]interface{}{\n		{{range .Parameters -}}\n		{{if eq \"path\" .ParamType -}}\n		  \"{{.Name}}\": {{.Name}},\n	  {{- end }}\n		{{- end }}\n	}\n\n  queryParamMap := map[string]interface{}{\n		{{range .Parameters -}}\n		{{if eq \"query\" .ParamType -}}\n		  \"{{.Name}}\": {{.Name}},\n	  {{- end }}\n		{{- end }}\n	}\n\n	{{if .DTORequest -}}\n	{{if .MakesResult}}\n    response = make({{- template \"type\" . -}}, 0)\n		err = client.DTORequest(&response, \"{{.Method}}\", \"{{.Path}}\", pathParamMap, queryParamMap\n		{{- if .HasBody -}}\n		, body\n		{{- end -}})\n	{{else}}\n    response = new({{.GoPackage}}\n    {{- if ne .GoPackage \"\"}}.{{end -}}\n    {{.GoBaseType}})\n		err = client.DTORequest(response, \"{{.Method}}\", \"{{.Path}}\", pathParamMap, queryParamMap\n		{{- if .HasBody -}}\n		, body\n		{{- end -}})\n	{{end}}\n	{{else if (eq .GoBaseType \"\")}}\n	_, err = client.Request(\"{{.Method}}\", \"{{.Path}}\", pathParamMap, queryParamMap\n	{{- if .HasBody -}}\n	, body\n  {{- end -}})\n	{{else if eq .GoBaseType \"string\"}}\n	resBody, err := client.Request(\"{{.Method}}\", \"{{.Path}}\", pathParamMap, queryParamMap\n	{{- if .HasBody -}}\n	, body\n  {{- end -}})\n	readBuf := bytes.Buffer{}\n	readBuf.ReadFrom(resBody)\n	response = string(readBuf.Bytes())\n	{{- end}}\n	return\n}\n{{end}}\n",
	`type.tmpl`: "{{.GoTypePrefix -}}\n{{if ne .GoPackage \"\"}}{{.GoPackage}}.{{end -}}\n{{.GoBaseType -}}\n",
})
