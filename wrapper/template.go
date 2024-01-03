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
	htmlTemplate "html/template"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	textTemplate "text/template"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
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
	t.textTmpl = textTemplate.New("root").Funcs(funcMaps)
	t.textTmpl, err = t.textTmpl.Parse(raw)
	return err
}

func (t *Template) parseTextFileTemplate(filename string) (err error) {
	t.htmlTmpl = nil
	t.textTmpl = textTemplate.New(filepath.Base(filename)).Funcs(funcMaps)
	t.textTmpl, err = t.textTmpl.ParseFiles(filename)
	return err
}

func (t *Template) parseHTMLFileTemplate(filename string) (err error) {
	t.textTmpl = nil
	t.htmlTmpl = htmlTemplate.New(filepath.Base(filename)).Funcs(funcMaps)
	t.htmlTmpl, err = t.htmlTmpl.ParseFiles(filename)
	return err
}

func (t *Template) parseHTMLTemplate(raw string) (err error) {
	t.textTmpl = nil
	t.htmlTmpl = htmlTemplate.New("root").Funcs(funcMaps)
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
		err = t.parseHTMLFileTemplate(value.Value)
	case "!html":
		switch value.Kind {
		case yaml.ScalarNode:
			err = t.parseHTMLTemplate(value.Value)
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

var defaultFuncs = textTemplate.FuncMap{
	"toUpper": strings.ToUpper,
	"toLower": strings.ToLower,
	"add":     func(a, b int) int { return a + b },
	"sub":     func(a, b int) int { return a - b },
	"title":   cases.Title(language.AmericanEnglish).String,
	// join is equal to strings.Join but inverts the argument order
	// for easier pipelining in templates.
	"join": func(sep string, s []string) string {
		return strings.Join(s, sep)
	},
	"match": regexp.MatchString,
	"safeHtml": func(text string) htmlTemplate.HTML {
		return htmlTemplate.HTML(text)
	},
	"reReplaceAll": func(pattern, repl, text string) string {
		re := regexp.MustCompile(pattern)
		return re.ReplaceAllString(text, repl)
	},
	"stringSlice": func(s ...string) []string {
		return s
	},
	"toMap": func(s string) map[string]string {
		m := make(map[string]string)
		if err := json.Unmarshal([]byte(s), &m); err != nil {
			panic(err)
		} else {
			return m
		}
	},
	"toJson": func(o interface{}) string {
		data, _ := json.Marshal(o)
		return string(data)
	},
	"safeJson": func(o interface{}) string {
		data, _ := json.Marshal(o)
		return strings.Trim(string(data), "\"")
	},
}

var funcMaps = textTemplate.FuncMap{}

func AddFuncMaps(funcName string, f interface{}) {
	funcMaps[funcName] = f
}

func init() {
	for name, f := range defaultFuncs {
		AddFuncMaps(name, f)
	}
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
			err = t.parseHTMLTemplate(val)
		case "fn::htmlFile":
			err = t.parseHTMLFileTemplate(val)
		default:
			return fmt.Errorf("unknown function: %s", name)
		}
		break
	}
	return err
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
