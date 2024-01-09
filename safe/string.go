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

package safe

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/gogo/protobuf/jsonpb"
	"gopkg.in/yaml.v3"
)

type String struct {
	Value  string
	secret string
}

func (e *String) Reset() { e.Value = "" }

func (e *String) String() string { return e.Value }

func (e *String) XXX_WellKnownType() string { return "StringValue" } //nolint:revive

func (e *String) ProtoMessage() {}

func (e *String) UnmarshalJSONPB(_ *jsonpb.Unmarshaler, bytes []byte) error {
	return e.UnmarshalJSON(bytes)
}

func (e String) MarshalJSONPB(_ *jsonpb.Marshaler) ([]byte, error) {
	return e.MarshalJSON()
}

func (e String) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

func (e *String) UnmarshalJSON(bytes []byte) (err error) {
	if err = json.Unmarshal(bytes, &e.Value); err != nil {
		return err
	}
	if !strings.HasPrefix(e.Value, ciphertextPrefix) {
		if secret := os.Getenv(SecretEnvName); secret != "" {
			if e.Value, err = Encrypt([]byte(e.Value), secret, nil); err != nil {
				return err
			} else {
				e.secret = secret
			}
		}
	}
	return nil
}

func (e *String) UnmarshalYAML(value *yaml.Node) (err error) {
	if err := value.Decode(&e.Value); err != nil {
		return err
	}
	if !strings.HasPrefix(e.Value, ciphertextPrefix) {
		if secret := os.Getenv(SecretEnvName); secret != "" {
			if e.Value, err = Encrypt([]byte(e.Value), secret, nil); err != nil {
				return err
			} else {
				e.secret = secret
			}
		}
	}
	return nil
}

func (e String) UnsafeString() (string, error) {
	if strings.HasPrefix(e.Value, ciphertextPrefix) {
		secret := e.secret
		if secret == "" {
			secret = os.Getenv(SecretEnvName)
		}
		if secret != "" {
			decrypt, err := Decrypt(e.Value, secret)
			return string(decrypt), err
		}
	}
	return e.Value, nil
}

func (e String) Size() int {
	return len(e.Value)
}

func NewEncryptedString(plain, secret string) *String {
	return &String{Value: plain, secret: secret}
}
