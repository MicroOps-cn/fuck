package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	stdtime "time"

	"github.com/spf13/cast"
)

type Storage interface {
	fs.ReadDirFS
	json.Marshaler
	Type() string
	Name() string
	ListObject(ctx context.Context, objectPrefix string, recursion bool, callback func(key Object)) error
	HeadObject(ctx context.Context, objectPath string) (obj *Object, err error)
	GetObject(ctx context.Context, objectPath string) (*ObjectReader, error)
	PutObject(ctx context.Context, objectPath string, obj io.Reader, headers http.Header, metadata map[string]string) error
}

type Object struct {
	Key          string
	LastModified stdtime.Time
	Size         int64
	ETag         string
	StorageClass string
	Mode         fs.FileMode
	Headers      http.Header
	Metadata     map[string]string
}

func (o Object) Name() string {
	return o.Key
}

func (o Object) IsDir() bool {
	return o.Mode.IsDir()
}

func (o Object) Type() fs.FileMode {
	return o.Mode
}

func (o Object) Info() (fs.FileInfo, error) {
	return &FileInfo{
		name:    o.Key,
		size:    o.Size,
		mode:    o.Mode,
		modTime: o.LastModified,
	}, nil
}

type ObjectReader struct {
	io.ReadCloser
	Object
}

type FileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime stdtime.Time
}

func (f FileInfo) Name() string {
	return f.name
}

func (f FileInfo) Size() int64 {
	return f.size
}

func (f FileInfo) Mode() fs.FileMode {
	return f.mode
}

func (f FileInfo) ModTime() stdtime.Time {
	return f.modTime
}

func (f FileInfo) IsDir() bool {
	return f.mode.IsDir()
}

func (f FileInfo) Sys() any {
	return nil
}

func (o ObjectReader) Stat() (fs.FileInfo, error) {
	return o.Info()
}

func RawPermissionsToMode(raw string) fs.FileMode {
	const str = "dalTLDpSugct?"
	if len(raw) == 9 {
		raw = "-" + raw
	}
	if len(raw) != 10 {
		return 0o777
	}
	permissions := []rune(raw)
	var mode fs.FileMode
	if raw[0] != '-' {
		for i, c := range str {
			if permissions[0] == c {
				mode |= 1 << uint(32-1-i)
			}
		}
	}
	rwx := []rune("rwxrwxrwx")
	for i, ch := range permissions[1:] {
		if rwx[i] == ch {
			mode |= 1 << uint(8-i)
		}
	}

	return mode
}

type ConfigProvider interface {
	GetString(name string) string
	GetInt(name string) int
}

type mapConfigProvider map[string]interface{}

func (v mapConfigProvider) GetString(key string) string {
	return cast.ToString(v.Get(key))
}

func (v mapConfigProvider) Get(key string) interface{} {
	val, _ := v[key]
	return val
}

func (v mapConfigProvider) GetInt(key string) int {
	return cast.ToInt(v.Get(key))
}

func NewMapConfigProvider(m map[string]interface{}) ConfigProvider {
	return mapConfigProvider(m)
}

var registeredStorage = make(map[string]func(ctx context.Context, v ConfigProvider) (Storage, error))

func RegisterStorage(storageType string, newFunc func(ctx context.Context, v ConfigProvider) (Storage, error)) {
	if _, ok := registeredStorage[storageType]; ok {
		panic("Duplicate registration of storage types")
	}
	registeredStorage[storageType] = newFunc
}

func NewClient(ctx context.Context, v ConfigProvider) (Storage, error) {
	storageType := v.GetString("type")
	if storageType == "" {
		return nil, fmt.Errorf("storage type is empty")
	}
	if newFunc, ok := registeredStorage[storageType]; ok {
		return newFunc(ctx, v)
	}
	return nil, fmt.Errorf("unknown backend type %s", storageType)
}
