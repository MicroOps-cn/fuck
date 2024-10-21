package fs

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/MicroOps-cn/fuck/clients/storage"
	"github.com/MicroOps-cn/fuck/log"
	"github.com/aws/smithy-go/time"
	"github.com/go-kit/log/level"
	"github.com/spf13/afero"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type Options struct {
	Base string `json:"base,omitempty" yaml:"base,omitempty"`
	Type string `json:"type,omitempty" yaml:"type,omitempty"`
}

type Client struct {
	fs     fs.FS
	fsType string
	o      Options
}

func (c Client) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.o)
}

func (c Client) Name() string {
	return fmt.Sprintf("%s://%s", c.Type(), c.o.Base)
}
func (c Client) GetObject(_ context.Context, objectPath string) (*storage.ObjectReader, error) {
	if strings.HasPrefix(objectPath, "file://") {
		objectPath = strings.TrimPrefix(objectPath, "file://")
	} else if strings.HasPrefix(objectPath, "local://") {
		objectPath = strings.TrimPrefix(objectPath, "local://")
	}
	f, err := os.Open(objectPath)
	if err != nil {
		return nil, err
	}
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	return &storage.ObjectReader{
		Headers: map[string][]string{
			"Content-Length": {strconv.FormatInt(stat.Size(), 64)},
			"Last-Modified":  {time.FormatDateTime(stat.ModTime())},
		},
		LastModified: stat.ModTime(),
		Metadata:     map[string]string{},
		ReadCloser:   f,
		Size:         stat.Size(),
	}, nil
}

func (c Client) PutObject(ctx context.Context, objectPath string, obj io.Reader, headers http.Header, metadata map[string]string) error {
	if strings.HasPrefix(objectPath, "file://") {
		objectPath = strings.TrimPrefix(objectPath, "file://")
	} else if strings.HasPrefix(objectPath, "local://") {
		objectPath = strings.TrimPrefix(objectPath, "local://")
	}
	w, err := os.OpenFile(objectPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, obj)
	return err
}

func (c Client) HeadObject(ctx context.Context, objectPath string) (obj *storage.Object, err error) {
	if strings.HasPrefix(objectPath, "file://") {
		objectPath = strings.TrimPrefix(objectPath, "file://")
	} else if strings.HasPrefix(objectPath, "local://") {
		objectPath = strings.TrimPrefix(objectPath, "local://")
	}
	stat, err := os.Stat(objectPath)
	if err != nil {
		return nil, err
	}
	return &storage.Object{
		Key:          objectPath,
		LastModified: stat.ModTime(),
		Size:         stat.Size(),
		StorageClass: c.Type(),
	}, nil
}

func (c Client) ListObject(ctx context.Context, objectPrefix string, recursion bool, callback func(key storage.Object)) error {
	logger := log.GetContextLogger(ctx)
	if strings.HasPrefix(objectPrefix, "file://") {
		objectPrefix = strings.TrimPrefix(objectPrefix, "file://")
	} else if strings.HasPrefix(objectPrefix, "local://") {
		objectPrefix = strings.TrimPrefix(objectPrefix, "local://")
	}
	return fs.WalkDir(c.fs, objectPrefix, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if path == "." {
				return nil
			}
			if !recursion {
				return fs.SkipDir
			}
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			o := storage.Object{
				Key:          path,
				StorageClass: c.Type(),
			}
			fileInfo, err := info.Info()
			if err != nil {
				level.Warn(logger).Log("msg", "failed to get file info", "path", path, "err", err)
			} else {
				o.LastModified = fileInfo.ModTime()
				o.Size = fileInfo.Size()
			}
			callback(o)
		}
		return nil
	})
}

func (c Client) Type() string {
	return c.fsType
}

func NewClient(_ context.Context, fs fs.FS, v storage.ConfigProvider) (*Client, error) {
	var storageType string
	var base string
	if v != nil {
		storageType = v.GetString("type")
		base = v.GetString("base")
	}
	var err error
	if fs == nil {
		switch storageType {
		case "zip":
			if fs, err = zip.OpenReader(base); err != nil {
				return nil, err
			}
		case "in-memory", "inmemory":
			fs = afero.NewIOFS(afero.NewMemMapFs())
		case "local", "file":
			fs = afero.NewIOFS(afero.NewBasePathFs(afero.NewOsFs(), base))
		}
	}
	if storageType == "" {
		switch vfs := fs.(type) {
		case afero.IOFS:
			storageType = vfs.Name()
		default:
			storageType = "unknown"
		}
	}
	return &Client{
		fs:     fs,
		fsType: storageType,
		o:      Options{Base: base, Type: storageType},
	}, nil
}

func init() {
	storage.RegisterStorage("local", func(ctx context.Context, v storage.ConfigProvider) (storage.Storage, error) {
		return NewClient(ctx, nil, v)
	})
	storage.RegisterStorage("file", func(ctx context.Context, v storage.ConfigProvider) (storage.Storage, error) {
		return NewClient(ctx, nil, v)
	})
	storage.RegisterStorage("zip", func(ctx context.Context, v storage.ConfigProvider) (storage.Storage, error) {
		return NewClient(ctx, nil, v)
	})
	storage.RegisterStorage("in-memory", func(ctx context.Context, v storage.ConfigProvider) (storage.Storage, error) {
		return NewClient(ctx, nil, v)
	})
	storage.RegisterStorage("inmemory", func(ctx context.Context, v storage.ConfigProvider) (storage.Storage, error) {
		return NewClient(ctx, nil, v)
	})
}
