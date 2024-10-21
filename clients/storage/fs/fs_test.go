package fs

import (
	"context"
	"github.com/MicroOps-cn/fuck/clients/storage"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"io/fs"
	"os"
	"sort"
	"testing"
)

func touchFile(fs afero.Fs, path string) error {
	file, err := fs.OpenFile(path, os.O_CREATE, 0o777)
	if err != nil {
		return err
	}
	return file.Close()
}

func TestClient_ListObject(t *testing.T) {
	temppath, err := os.MkdirTemp("", "test-fs")
	defer os.RemoveAll(temppath)
	t.Logf("make temp fs: %s", temppath)
	tempFS := afero.NewBasePathFs(afero.NewOsFs(), temppath)
	require.NoError(t, touchFile(tempFS, "00000000000000.txt"))
	require.NoError(t, tempFS.MkdirAll("A/B/C/D", 0o777))
	require.NoError(t, touchFile(tempFS, "A/B/C/D/1"))
	require.NoError(t, tempFS.MkdirAll("A/B/D/E", 0o777))
	require.NoError(t, touchFile(tempFS, "A/B/D/E/2"))
	require.NoError(t, tempFS.MkdirAll("C/A", 0o777))
	require.NoError(t, touchFile(tempFS, "C/A/3"))
	require.NoError(t, tempFS.MkdirAll("C0000/A", 0o777))
	require.NoError(t, touchFile(tempFS, "C0000/A/4"))
	require.NoError(t, err)

	type fields struct {
		fs             fs.FS
		configProvider storage.ConfigProvider
	}
	type args struct {
		ctx          context.Context
		objectPrefix string
		recursion    bool
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantErr  bool
		wantList []string
	}{{
		name: "test-recursion",
		fields: fields{
			fs: nil,
			configProvider: storage.NewMapConfigProvider(map[string]interface{}{
				"type": "local",
				"base": temppath,
			}),
		},
		args: args{
			ctx:          context.Background(),
			objectPrefix: ".",
			recursion:    true,
		},
		wantList: []string{"00000000000000.txt", "A/B/C/D/1", "A/B/D/E/2", "C/A/3", "C0000/A/4"},
	}, {
		name: "test-fs",
		fields: fields{
			fs: afero.NewIOFS(tempFS),
			configProvider: storage.NewMapConfigProvider(map[string]interface{}{
				"type": "local",
			}),
		},
		args: args{
			ctx:          context.Background(),
			objectPrefix: ".",
		},
		wantList: []string{"00000000000000.txt"},
	}, {
		name: "test-fs-recursion",
		fields: fields{
			fs: afero.NewIOFS(tempFS),
			configProvider: storage.NewMapConfigProvider(map[string]interface{}{
				"type": "local",
			}),
		},
		args: args{
			ctx:          context.Background(),
			objectPrefix: ".",
			recursion:    true,
		},
		wantList: []string{"00000000000000.txt", "A/B/C/D/1", "A/B/D/E/2", "C/A/3", "C0000/A/4"},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fileList []string
			var callback = func(key storage.Object) {
				fileList = append(fileList, key.Key)
			}
			c, err := NewClient(nil, tt.fields.fs, tt.fields.configProvider)
			require.NoError(t, err)
			if err := c.ListObject(tt.args.ctx, tt.args.objectPrefix, tt.args.recursion, callback); (err != nil) != tt.wantErr {
				t.Errorf("ListObject() error = %v, wantErr %v", err, tt.wantErr)
			}
			sort.Strings(fileList)
			sort.Strings(tt.wantList)
			require.Equal(t, tt.wantList, fileList)
		})
	}
}
