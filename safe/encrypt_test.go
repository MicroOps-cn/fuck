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
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncrypt(t *testing.T) {
	type args struct {
		key string
		o   *EncryptOptions
	}
	tests := []struct {
		name           string
		args           args
		want           string
		encryptWantErr bool
		decryptWantErr bool
	}{{
		name: "default",
		args: args{
			key: "$L9+W9M!jbGMPjKln7Rn6Ge.",
			o:   NewEncryptOptions(),
		},
	}, {
		name: "des",
		args: args{
			key: "12345678",
			o:   NewEncryptOptions(WithAlgorithm(AlgorithmDES)),
		},
	}, {
		name: "3des",
		args: args{
			key: "$L9+W9M!jbGMPjKln7Rn6Ge.",
			o:   NewEncryptOptions(WithAlgorithm(Algorithm3DES)),
		},
	}, {
		name: "aes",
		args: args{
			key: "$L9+W9M!jbGMPjKln7Rn6Ge.",
			o:   NewEncryptOptions(WithAlgorithm(AlgorithmAES)),
		},
	}, {
		name: "tea",
		args: args{
			key: "1234567890123456",
			o:   NewEncryptOptions(WithAlgorithm(AlgorithmTEA)),
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for dataLength := 1; dataLength < 1024; dataLength++ {
				originalData := make([]byte, dataLength)
				_, err := rand.Read(originalData)
				require.NoError(t, err)
				encryptedStr, err := Encrypt(originalData, tt.args.key, tt.args.o)
				if (err != nil) != tt.encryptWantErr {
					t.Logf("%s => %s", hex.EncodeToString(originalData), encryptedStr)
					t.Errorf("Encrypt() error = %v, wantErr %v", err, tt.encryptWantErr)
					return
				}
				decryptBytes, err := Decrypt(encryptedStr, tt.args.key)
				if (err != nil) != tt.decryptWantErr {
					t.Logf("%s => %s", hex.EncodeToString(originalData), encryptedStr)
					t.Errorf("Decrypt() error = %v, wantErr %v", err, tt.decryptWantErr)
					return
				}
				if !bytes.Equal(decryptBytes, originalData) {
					t.Logf("%s => %s", hex.EncodeToString(originalData), encryptedStr)
					t.Errorf("Decrypt(Encrypt()) Do not want to wait with the original data, got = %v, want %v", decryptBytes, originalData)
					return
				}
				if dataLength == 10 {
					t.Logf("%s => %s", base64.StdEncoding.EncodeToString(originalData), encryptedStr)
				}
			}
		})
	}
}
