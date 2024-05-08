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
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/gogo/protobuf/jsonpb"
	"gopkg.in/yaml.v3"
)

type String struct {
	value  string
	secret string
}

func (e *String) Reset() { *e = String{} }

func (e *String) getSecret() string {
	if e.secret != "" {
		return e.secret
	}
	return os.Getenv(SecretEnvName)
}

func (e *String) String() string {
	if e.value == "" {
		return ""
	}
	if !strings.HasPrefix(e.value, ciphertextPrefix) {
		if secret := e.getSecret(); secret != "" {
			if value, err := Encrypt([]byte(e.value), e.secret, nil); err == nil {
				e.value = value
			}
		}
	}
	return e.value
}

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
	if err = json.Unmarshal(bytes, &e.value); err != nil {
		return err
	}
	if !strings.HasPrefix(e.value, ciphertextPrefix) {
		if secret := e.getSecret(); secret != "" {
			if e.value, err = Encrypt([]byte(e.value), secret, nil); err != nil {
				return err
			}
			e.secret = secret
		}
	}
	return nil
}

func (e *String) UnmarshalYAML(value *yaml.Node) (err error) {
	if err := value.Decode(&e.value); err != nil {
		return err
	}
	if !strings.HasPrefix(e.value, ciphertextPrefix) {
		if secret := e.getSecret(); secret != "" {
			if e.value, err = Encrypt([]byte(e.value), secret, nil); err != nil {
				return err
			}
			e.secret = secret
		}
	}
	return nil
}

func (e String) UnsafeString() (string, error) {
	if e.value == "" {
		return "", nil
	}
	if strings.HasPrefix(e.value, ciphertextPrefix) {
		if secret := e.getSecret(); secret != "" {
			decrypt, err := Decrypt(e.value, secret)
			return string(decrypt), err
		}
	}
	return e.value, nil
}

func (e String) Size() int {
	return len(e.value)
}

func (*String) GormDataType() string {
	return "string"
}

func (e *String) SetValue(value string) (err error) {
	if !strings.HasPrefix(value, ciphertextPrefix) {
		if secret := e.getSecret(); secret != "" {
			e.value, err = Encrypt([]byte(value), e.secret, nil)
			return err
		}
	}
	e.value = value
	return nil
}

func (e *String) UpdateSecret(secret string) {
	plain, err := e.UnsafeString()
	if err != nil {
		return
	}
	if len(secret) != 0 {
		if safeString, err := Encrypt([]byte(plain), secret, nil); err == nil {
			e.value = safeString
		}
	}
	e.secret = secret
}

func (e *String) SetSecret(secret string) {
	e.secret = secret
}

// Scan implements the Scanner interface.
func (e *String) Scan(value any) error {
	switch vt := value.(type) {
	case []uint8:
		e.value = string(vt)
	case string:
		e.value = vt
	default:
		return fmt.Errorf("failed to resolve field, type exception: %T", value)
	}
	return nil
}

// Value implements the driver Valuer interface.
func (e String) Value() (driver.Value, error) {
	return e.value, nil
}

func NewEncryptedString(plain, secret string) *String {
	if !strings.HasPrefix(plain, ciphertextPrefix) {
		if len(secret) > 0 {
			if safeString, err := Encrypt([]byte(plain), secret, nil); err == nil {
				return &String{value: safeString, secret: secret}
			}
		}
	}
	return &String{value: plain, secret: secret}
}
