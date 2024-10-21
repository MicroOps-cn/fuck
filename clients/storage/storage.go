package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/spf13/cast"
	"io"
	"net/http"
	stdtime "time"
)

type Storage interface {
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
}

type ObjectReader struct {
	io.ReadCloser
	Headers      http.Header
	Metadata     map[string]string
	LastModified stdtime.Time
	Size         int64
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
