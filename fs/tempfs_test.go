/*
 Copyright © 2024 MicroOps-cn.

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

package fs

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTempFS_OpenFile(t1 *testing.T) {
	fs, err := NewTempFS("", "test-tempfs.")
	require.NoError(t1, err)
	type args struct {
		filename string
		flag     int
		perm     os.FileMode
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "simple", args: args{filename: "abc", flag: os.O_RDWR | os.O_CREATE, perm: 0o644}},
		{name: "simple", args: args{filename: "a/b/c", flag: os.O_RDWR | os.O_CREATE, perm: 0o644}},
		{name: "simple", args: args{filename: "../../../../../../def", flag: os.O_RDWR | os.O_CREATE, perm: 0o644}, wantErr: true},
		{name: "simple", args: args{filename: "../%00%00/../../../../ghi", flag: os.O_RDWR | os.O_CREATE, perm: 0o644}, wantErr: true},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			_, err := fs.OpenFile(tt.args.filename, tt.args.flag, tt.args.perm)
			if (err != nil) != tt.wantErr {
				t1.Errorf("OpenFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				fmt.Println(fs.tempDir)
				if _, err = os.Stat(path.Join(fs.tempDir, tt.args.filename)); err != nil {
					require.NoError(t1, err)
				}
			}
		})
	}
}
