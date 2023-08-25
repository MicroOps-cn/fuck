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
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"os"
	"testing"
)

func TestTemplate_UnmarshalYAML(t1 *testing.T) {
	temp, err := os.CreateTemp("/tmp/", "go-template-test")
	require.NoError(t1, err)
	_, _ = temp.WriteString(`hello {{.text}}`)
	_ = temp.Close()
	defer func(name string) {
		_ = os.Remove(name)
	}(temp.Name())
	type config struct {
		Template Template `yaml:"template"`
	}
	type args struct {
		value string
		data  any
	}
	tests := []struct {
		name    string
		config  config
		args    args
		wantErr bool
		want    string
	}{{
		name:   "simple text template",
		config: config{},
		args:   args{value: "template: 'hello {{.text}}'", data: map[string]string{"text": "world"}},
		want:   "hello world",
	}, {
		name:   "text template",
		config: config{},
		args:   args{value: `template: !text "hello {{.text}}"`, data: map[string]string{"text": "<script>alert('world')</script>"}},
		want:   "hello <script>alert('world')</script>",
	}, {
		name:   "html template",
		config: config{},
		args:   args{value: `template: !html "hello {{.text}}"`, data: map[string]string{"text": "<script>alert('world')</script>"}},
		want:   "hello &lt;script&gt;alert(&#39;world&#39;)&lt;/script&gt;",
	}, {
		name:   "block template",
		config: config{},
		args:   args{value: "template: !html |\n  hello {{.text}}", data: map[string]string{"text": "<script>alert('world')</script>"}},
		want:   "hello &lt;script&gt;alert(&#39;world&#39;)&lt;/script&gt;",
	}, {
		name:   "file template",
		config: config{},
		args:   args{value: `template: !file ` + temp.Name(), data: map[string]string{"text": "<script>alert('world')</script>"}},
		want:   "hello <script>alert('world')</script>",
	}, {
		name:   "text template",
		config: config{},
		args:   args{value: `template: !htmlFile ` + temp.Name(), data: map[string]string{"text": "<script>alert('world')</script>"}},
		want:   "hello &lt;script&gt;alert(&#39;world&#39;)&lt;/script&gt;",
	}}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			if err := yaml.Unmarshal([]byte(tt.args.value), &tt.config); (err != nil) != tt.wantErr {
				t1.Errorf("UnmarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				buf := bytes.Buffer{}
				err = tt.config.Template.Execute(&buf, tt.args.data)
				require.NoError(t1, err)
				require.Equal(t1, tt.want, buf.String())
			}
		})
	}
}

func TestTemplate_UnmarshalJSON(t1 *testing.T) {
	temp, err := os.CreateTemp("/tmp/", "go-template-test")
	require.NoError(t1, err)
	_, _ = temp.WriteString(`hello {{.text}}`)
	_ = temp.Close()
	defer func(name string) {
		_ = os.Remove(name)
	}(temp.Name())
	type config struct {
		Template Template `yaml:"template"`
	}
	type args struct {
		value string
		data  any
	}
	tests := []struct {
		name    string
		config  config
		args    args
		wantErr bool
		want    string
	}{{
		name:   "simple text template",
		config: config{},
		args:   args{value: `{"template": "hello {{.text}}"}`, data: map[string]string{"text": "world"}},
		want:   "hello world",
	}, {
		name:   "text template",
		config: config{},
		args:   args{value: `{"template": {"fn::text":"hello {{.text}}"}}`, data: map[string]string{"text": "<script>alert('world')</script>"}},
		want:   "hello <script>alert('world')</script>",
	}, {
		name:   "html template",
		config: config{},
		args:   args{value: `{"template": {"fn::html":"hello {{.text}}"}}`, data: map[string]string{"text": "<script>alert('world')</script>"}},
		want:   "hello &lt;script&gt;alert(&#39;world&#39;)&lt;/script&gt;",
	}, {
		name:   "file template",
		config: config{},
		args:   args{value: fmt.Sprintf(`{"template": {"fn::file":"%s"}}`, temp.Name()), data: map[string]string{"text": "<script>alert('world')</script>"}},
		want:   "hello <script>alert('world')</script>",
	}, {
		name:   "html file template",
		config: config{},
		args:   args{value: fmt.Sprintf(`{"template": {"fn::htmlFile":"%s"}}`, temp.Name()), data: map[string]string{"text": "<script>alert('world')</script>"}},
		want:   "hello &lt;script&gt;alert(&#39;world&#39;)&lt;/script&gt;",
	}}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			if err := json.Unmarshal([]byte(tt.args.value), &tt.config); (err != nil) != tt.wantErr {
				t1.Errorf("UnmarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				buf := bytes.Buffer{}
				err = tt.config.Template.Execute(&buf, tt.args.data)
				require.NoError(t1, err)
				require.Equal(t1, tt.want, buf.String())
			}
		})
	}
}
