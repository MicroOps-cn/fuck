/*
 Copyright © 2023 MicroOps-cn.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package w

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	htmlTemplate "html/template"
	"io"
	"path/filepath"
	textTemplate "text/template"
)

type TemplateHandler interface {
	Name() string
	ExecuteTemplate(wr io.Writer, name string, data any) error
	Execute(wr io.Writer, data any) error
	DefinedTemplates() string
}

type Template struct {
	raw      string
	htmlTmpl *htmlTemplate.Template
	textTmpl *textTemplate.Template
}

func (t Template) MarshalYAML() (interface{}, error) {
	return t.raw, nil
}

func (t *Template) parseTextTemplate(raw string) (err error) {
	t.htmlTmpl = nil
	if t.textTmpl == nil {
		t.textTmpl = textTemplate.New("root")
	}
	t.textTmpl, err = t.textTmpl.Parse(raw)
	return err
}

func (t *Template) parseTextFileTemplate(filename string) (err error) {
	t.htmlTmpl = nil
	t.textTmpl = textTemplate.New(filepath.Base(filename))
	t.textTmpl, err = t.textTmpl.ParseFiles(filename)
	return err
}

func (t *Template) parseHtmlFileTemplate(filename string) (err error) {
	t.textTmpl = nil
	t.htmlTmpl = htmlTemplate.New(filepath.Base(filename))

	t.htmlTmpl, err = t.htmlTmpl.ParseFiles(filename)
	return err
}

func (t *Template) parseHtmlTemplate(raw string) (err error) {
	t.textTmpl = nil
	if t.htmlTmpl == nil {
		t.htmlTmpl = htmlTemplate.New("root")
	}
	t.htmlTmpl, err = t.htmlTmpl.Parse(raw)
	return err
}

func (t *Template) UnmarshalYAML(value *yaml.Node) (err error) {
	t.raw = value.Value
	switch value.Tag {
	case "!!str":
		switch value.Kind {
		case yaml.ScalarNode:
			err = t.parseTextTemplate(value.Value)
		default:
			return fmt.Errorf("unknown type: %s", value.Tag)
		}
	case "!text":
		switch value.Kind {
		case yaml.ScalarNode:
			err = t.parseTextTemplate(value.Value)
		default:
			return fmt.Errorf("unknown type: %s", value.Tag)
		}
	case "!file":
		err = t.parseTextFileTemplate(value.Value)
	case "!htmlFile":
		err = t.parseHtmlFileTemplate(value.Value)
	case "!html":
		switch value.Kind {
		case yaml.ScalarNode:
			err = t.parseHtmlTemplate(value.Value)
		default:
			return fmt.Errorf("unknown type: %s", value.Tag)
		}
	default:
		return fmt.Errorf("unknown type: %s", value.Tag)
	}

	return err
}

func (t Template) MarshalJSON() ([]byte, error) {
	return []byte(t.raw), nil
}

func (t *Template) UnmarshalJSON(data []byte) (err error) {
	if len(data) >= 2 && data[0] == '"' && data[len(data)-1] == '"' {
		var text string
		if err = json.Unmarshal(data, &text); err != nil {
			return err
		}
		err = t.parseTextTemplate(text)
		return err
	}
	var tmplFuncs map[string]string
	if err = json.Unmarshal(data, &tmplFuncs); err != nil {
		return err
	}
	if len(tmplFuncs) != 1 {
		return fmt.Errorf("invalid format")
	}
	for name, val := range tmplFuncs {
		switch name {
		case "fn::text":
			err = t.parseTextTemplate(val)
		case "fn::file":
			err = t.parseTextFileTemplate(val)
		case "fn::html":
			err = t.parseHtmlTemplate(val)
		case "fn::htmlFile":
			err = t.parseHtmlFileTemplate(val)
		default:
			return fmt.Errorf("unknown function: %s", name)
		}
		break
	}
	return nil
}

func (t *Template) Execute(wr io.Writer, data any) error {
	return t.handler().Execute(wr, data)
}

func (t *Template) DefinedTemplates() string {
	return t.handler().DefinedTemplates()
}

func (t *Template) Name() string {
	return t.handler().Name()
}

func (t *Template) ExecuteTemplate(wr io.Writer, name string, data any) error {
	return t.handler().ExecuteTemplate(wr, name, data)
}

func (t *Template) handler() TemplateHandler {
	if t.textTmpl != nil {
		return t.textTmpl
	}
	return t.htmlTmpl
}
