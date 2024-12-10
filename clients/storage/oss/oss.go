package oss

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/transport"

	"github.com/MicroOps-cn/fuck/clients/storage"
	"github.com/MicroOps-cn/fuck/safe"
)

type Options struct {
	Type                   string      `json:"type,omitempty" yaml:"type,omitempty" mapstructure:"type"`
	Endpoint               string      `json:"endpoint,omitempty" yaml:"endpoint,omitempty" mapstructure:"endpoint"`
	AccessKeyId            string      `json:"access_key_id,omitempty" yaml:"access_key_id,omitempty" mapstructure:"access_key_id"`
	SecretAccessKey        safe.String `json:"secret_access_key,omitempty" yaml:"secret_access_key,omitempty" mapstructure:"secret_access_key"`
	Bucket                 string      `json:"bucket,omitempty" yaml:"bucket,omitempty" mapstructure:"bucket"`
	Region                 string      `json:"region,omitempty" yaml:"region,omitempty" mapstructure:"region"`
	Worker                 int         `json:"worker,omitempty" yaml:"worker,omitempty" mapstructure:"worker"`
	OSSMaxIdleConns        int         `json:"oss_max_idle_conns,omitempty" yaml:"oss_max_idle_conns,omitempty" mapstructure:"oss_max_idle_conns"`
	OSSMaxIdleConnsPerHost int         `json:"oss_max_idle_conns_per_host,omitempty" yaml:"oss_max_idle_conns_per_host,omitempty" mapstructure:"oss_max_idle_conns_per_host"`
}

func (c Client) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.o)
}

type Client struct {
	clt      *oss.Client
	uploader *oss.Uploader
	bucket   string
	o        Options
}

func (c Client) ReadDir(name string) ([]fs.DirEntry, error) {
	var entries []fs.DirEntry
	err := c.ListObject(context.Background(), name, false, func(object storage.Object) {
		entries = append(entries, object)
	})
	return entries, err
}

func (c Client) Open(name string) (fs.File, error) {
	return c.GetObject(context.Background(), name)
}

func (c Client) Name() string {
	return fmt.Sprintf("%s://%s", c.Type(), c.bucket)
}

func (c Client) GetObject(ctx context.Context, objectPath string) (*storage.ObjectReader, error) {
	var key, bucket string
	if !strings.HasPrefix(objectPath, "oss://") {
		key = objectPath
		bucket = c.bucket
	} else {
		var found bool
		bucket, key, found = strings.Cut(strings.TrimPrefix(objectPath, "oss://"), "/")
		if !found {
			return nil, fmt.Errorf("invalid object path: path is empty: %s", objectPath)
		}
	}
	if len(bucket) == 0 {
		return nil, errors.New("invalid object path: bucket is empty")
	}
	if len(objectPath) == 0 {
		return nil, errors.New("invalid object path: path is empty")
	}
	obj, err := c.clt.GetObject(ctx, &oss.GetObjectRequest{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, err
	}
	return &storage.ObjectReader{
		ReadCloser: obj.Body,
		Object: storage.Object{
			Key:          key,
			LastModified: oss.ToTime(obj.LastModified),
			Size:         obj.ContentLength,
			ETag:         strings.Trim(oss.ToString(obj.ETag), `"`),
			StorageClass: obj.Headers.Get("x-oss-storage-class"),
			Metadata:     obj.Metadata,
			Headers:      obj.Headers,
			Mode:         storage.RawPermissionsToMode(obj.Metadata["x-oss-storage-perms"]),
		},
	}, nil
}

func (c Client) PutObject(ctx context.Context, objectPath string, obj io.Reader, headers http.Header, metadata map[string]string) error {
	var key, bucket string
	if !strings.HasPrefix(objectPath, "oss://") {
		key = objectPath
		bucket = c.bucket
	} else {
		var found bool
		bucket, key, found = strings.Cut(strings.TrimPrefix(objectPath, "oss://"), "/")
		if !found {
			return fmt.Errorf("invalid object path: path is empty: %s", objectPath)
		}
	}
	if len(bucket) == 0 {
		return errors.New("invalid object path: bucket is empty")
	}
	if len(objectPath) == 0 {
		return errors.New("invalid object path: path is empty")
	}
	req := oss.PutObjectRequest{
		Key:      &key,
		Body:     obj,
		Bucket:   &bucket,
		Metadata: metadata,
	}
	for name := range headers {
		switch name {
		case "Content-Type":
			req.ContentType = oss.Ptr(headers.Get(name))
		case "Content-Encoding":
			req.ContentEncoding = oss.Ptr(headers.Get(name))
		case "Content-Disposition":
			req.ContentDisposition = oss.Ptr(headers.Get(name))
		case "Cache-Control":
			req.CacheControl = oss.Ptr(headers.Get(name))
		case "Expires":
			req.Expires = oss.Ptr(headers.Get(name))
		}
	}
	_, err := c.uploader.UploadFrom(ctx, &oss.PutObjectRequest{
		Bucket:   &bucket,
		Key:      &key,
		Metadata: metadata,
		ProgressFn: func(increment, transferred, total int64) {

		},
	}, obj)
	return err
}

func (c Client) HeadObject(ctx context.Context, objectPath string) (obj *storage.Object, err error) {
	var key, bucket string
	if !strings.HasPrefix(objectPath, "oss://") {
		key = objectPath
		bucket = c.bucket
	} else {
		var found bool
		bucket, key, found = strings.Cut(strings.TrimPrefix(objectPath, "oss://"), "/")
		if !found {
			return nil, fmt.Errorf("invalid object path: path is empty: %s", objectPath)
		}
	}
	if len(bucket) == 0 {
		return nil, errors.New("invalid object path: bucket is empty")
	}
	if len(objectPath) == 0 {
		return nil, errors.New("invalid object path: path is empty")
	}
	meta, err := c.clt.GetObjectMeta(ctx, &oss.GetObjectMetaRequest{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, err
	}
	return &storage.Object{
		Key:          key,
		LastModified: oss.ToTime(meta.LastModified),
		Size:         meta.ContentLength,
		ETag:         strings.Trim(oss.ToString(meta.ETag), `"`),
		StorageClass: meta.Headers.Get("x-oss-storage-class"),
		Metadata:     obj.Metadata,
		Headers:      obj.Headers,
		Mode:         storage.RawPermissionsToMode(obj.Metadata["x-oss-storage-perms"]),
	}, nil
}

func (c Client) ListObject(ctx context.Context, objectPrefix string, recursion bool, callback func(key storage.Object)) error {
	var prefix, bucket string
	if !strings.HasPrefix(objectPrefix, "oss://") {
		prefix = objectPrefix
		bucket = c.bucket
	} else {
		bucket, prefix, _ = strings.Cut(strings.TrimPrefix(objectPrefix, "oss://"), "/")
	}
	if len(bucket) == 0 {
		return errors.New("invalid object path: bucket is empty")
	}
	var delimiter *string

	if !recursion {
		delimiter = oss.Ptr("/")
	}
	var continueToken *string
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			result, err := c.clt.ListObjectsV2(ctx, &oss.ListObjectsV2Request{
				Bucket:            &bucket,
				Prefix:            &prefix,
				MaxKeys:           1000,
				ContinuationToken: continueToken,
				Delimiter:         delimiter,
			},
			)
			if err != nil {
				return err
			}
			for _, obj := range result.Contents {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					callback(storage.Object{
						Key:          oss.ToString(obj.Key),
						LastModified: oss.ToTime(obj.LastModified),
						Size:         obj.Size,
						ETag:         strings.Trim(oss.ToString(obj.ETag), `"`),
						StorageClass: oss.ToString(obj.StorageClass),
					})
				}
			}
			if result.NextContinuationToken == nil || len(oss.ToString(result.NextContinuationToken)) == 0 {
				return nil
			}
			continueToken = result.NextContinuationToken
		}
	}
}

func (c Client) Type() string {
	return "oss"
}

func NewClient(ctx context.Context, v storage.ConfigProvider) (*Client, error) {
	accessKeyId := v.GetString("access_key_id")
	var safeSecretKey safe.String
	if err := safeSecretKey.SetValue(v.GetString("secret_access_key")); err != nil {
		return nil, fmt.Errorf("failed to parse secret_access_key: %s", err)
	}
	accessKeySecret, err := safeSecretKey.UnsafeString()
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt secret_access_key: %s", err)
	}
	region := v.GetString("region")
	endpoint := v.GetString("endpoint")
	workerNum := v.GetInt("worker")
	if endpoint == "" && region != "" {
		endpoint = "https://oss-" + region + ".aliyuncs.com"
	}
	maxIdleConns := v.GetInt("oss_max_idle_conns")
	if maxIdleConns == 0 {
		maxIdleConns = (workerNum + 1) * 3 / 2
		if maxIdleConns < 100 {
			maxIdleConns = 100
		}
	}
	maxIdleConnsPerHost := v.GetInt("oss_max_idle_conns_per_host")
	if maxIdleConnsPerHost == 0 {
		maxIdleConnsPerHost = (workerNum + 1) * 3 / 2
		if maxIdleConnsPerHost < 100 {
			maxIdleConnsPerHost = 100
		}
	}
	clt := oss.NewClient(&oss.Config{
		LogLevel:            oss.Ptr(oss.LogWarn),
		Endpoint:            &endpoint,
		Region:              &region,
		CredentialsProvider: credentials.NewStaticCredentialsProvider(accessKeyId, accessKeySecret, ""),
		RetryMaxAttempts:    oss.Ptr(16),
		HttpClient: transport.NewHttpClient(&transport.Config{}, func(t *http.Transport) {
			t.MaxIdleConns = maxIdleConns
			t.MaxIdleConnsPerHost = maxIdleConnsPerHost
			t.MaxConnsPerHost = 0
		}),
	})

	return &Client{
		clt:      clt,
		uploader: clt.NewUploader(),
		bucket:   v.GetString("bucket"),
		o: Options{
			Worker:                 workerNum,
			SecretAccessKey:        safeSecretKey,
			AccessKeyId:            accessKeyId,
			Region:                 region,
			Endpoint:               endpoint,
			Bucket:                 v.GetString("bucket"),
			OSSMaxIdleConns:        maxIdleConns,
			OSSMaxIdleConnsPerHost: maxIdleConnsPerHost,
			Type:                   Client{}.Type(),
		},
	}, nil
}
func init() {
	storage.RegisterStorage(Client{}.Type(), func(ctx context.Context, v storage.ConfigProvider) (storage.Storage, error) {
		return NewClient(ctx, v)
	})
}
