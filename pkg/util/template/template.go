package template

import (
	"bytes"
	"text/template"
)

// TemplateArgs represents the full set of arguments required to render the resources
type TemplateArgs struct {
	Name                 string
	LatestResourceSchema string
}

// RenderTemplate renders the resources rendered
func RenderTemplate(raw []byte, input TemplateArgs) ([]byte, error) {
	tmpl, err := template.New("template").Parse(string(raw))
	if err != nil {
		return nil, err
	}
	buffer := bytes.NewBuffer([]byte{})
	err = tmpl.Execute(buffer, input)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
