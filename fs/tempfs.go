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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"

	"github.com/spf13/afero"
)

type TempFS struct {
	afero.Fs
	tempDir string
	files   []string
}

func NewTempFS(dir, pattern string) (*TempFS, error) {
	tempDir, err := os.MkdirTemp(dir, pattern)
	if err != nil {
		return nil, err
	}
	return &TempFS{tempDir: tempDir, Fs: afero.NewBasePathFs(afero.NewOsFs(), tempDir)}, nil
}

func (t *TempFS) OpenFile(filename string, flag int, perm os.FileMode) (afero.File, error) {
	dir := path.Dir(filename)
	_, err := t.Stat(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		if err = t.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	f, err := t.Fs.OpenFile(filename, flag, perm)
	if err != nil {
		return nil, err
	}
	t.files = append(t.files, filename)
	return f, nil
}

func (t *TempFS) Open(filename string) (fs.File, error) {
	return t.Fs.Open(filename)
}

func (t TempFS) List() []string {
	files := make([]string, len(t.files))
	copy(files, t.files)
	return files
}

func (t *TempFS) Close() error {
	if t.tempDir == "" {
		return errors.New("os: DirFS with empty root")
	}
	if t.tempDir == "/" || t.tempDir == "./" {
		return fmt.Errorf("The template dir is invalid: %s", t.tempDir)
	}
	return os.RemoveAll(t.tempDir)
}
