package s3

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	http2 "net/http"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/aws/smithy-go/time"
	"github.com/spf13/afero"

	"github.com/MicroOps-cn/fuck/clients/storage"
	"github.com/MicroOps-cn/fuck/safe"
)

type Options struct {
	Type            string      `json:"type,omitempty" yaml:"type,omitempty" mapstructure:"type"`
	Endpoint        string      `json:"endpoint,omitempty" yaml:"endpoint,omitempty" mapstructure:"endpoint"`
	AccessKeyId     string      `json:"access_key_id,omitempty" yaml:"access_key_id,omitempty" mapstructure:"access_key_id"`
	SecretAccessKey safe.String `json:"secret_access_key,omitempty" yaml:"secret_access_key,omitempty" mapstructure:"secret_access_key"`
	Bucket          string      `json:"bucket,omitempty" yaml:"bucket,omitempty" mapstructure:"bucket"`
	Region          string      `json:"region,omitempty" yaml:"region,omitempty" mapstructure:"region"`
	Worker          int         `json:"worker,omitempty" yaml:"worker,omitempty" mapstructure:"worker"`
}

func (c Client) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.o)
}

type Client struct {
	clt      *s3.Client
	uploader *manager.Uploader
	bucket   string
	o        Options
	cacheFS  afero.Fs
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
func (c Client) resolveObjectPath(objectPath string) (key, bucket string, err error) {
	if !strings.HasPrefix(objectPath, "s3://") {
		key = objectPath
		bucket = c.bucket
	} else {
		var found bool
		bucket, key, found = strings.Cut(strings.TrimPrefix(objectPath, "s3://"), "/")
		if !found {
			return "", "", fmt.Errorf("invalid object path: path is empty: %s", objectPath)
		}
	}
	if len(bucket) == 0 {
		return "", "", errors.New("invalid object path: bucket is empty")
	}
	return key, bucket, nil
}

func (c Client) GetObject(ctx context.Context, objectPath string) (*storage.ObjectReader, error) {
	key, bucket, err := c.resolveObjectPath(objectPath)
	if err != nil {
		return nil, err
	}
	if len(key) == 0 {
		return nil, errors.New("invalid object path: path is empty")
	}
	ret, err := c.clt.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, err
	}
	return &storage.ObjectReader{
		ReadCloser: ret.Body,
		Object: storage.Object{
			LastModified: aws.ToTime(ret.LastModified),
			Size:         aws.ToInt64(ret.ContentLength),
			Headers: map[string][]string{
				"Content-Type":        {aws.ToString(ret.ContentType)},
				"Content-Encoding":    {aws.ToString(ret.ContentEncoding)},
				"Content-Language":    {aws.ToString(ret.ContentLanguage)},
				"Content-Disposition": {aws.ToString(ret.ContentDisposition)},
				"Cache-Control":       {aws.ToString(ret.CacheControl)},
				"Expires":             {aws.ToString(ret.ExpiresString)},
				"Content-Length":      {strconv.FormatInt(aws.ToInt64(ret.ContentLength), 64)},
				"Last-Modified":       {time.FormatHTTPDate(aws.ToTime(ret.LastModified))},
				"ETag":                {aws.ToString(ret.ETag)},
			},
			Mode:     storage.RawPermissionsToMode(ret.Metadata["x-amz-meta-file-permissions"]),
			Metadata: ret.Metadata,
		},
	}, nil
}

func (c Client) PutObject(ctx context.Context, objectPath string, obj io.Reader, headers http2.Header, metadata map[string]string) error {
	key, bucket, err := c.resolveObjectPath(objectPath)
	if err != nil {
		return err
	}
	if len(key) == 0 {
		return errors.New("invalid object path: path is empty")
	}
	s3PutParams := s3.PutObjectInput{
		Key:      &key,
		Body:     obj,
		Bucket:   &bucket,
		Metadata: metadata,
	}
	for name := range headers {
		switch name {
		case "Content-Type":
			s3PutParams.ContentType = aws.String(headers.Get(name))
		case "Content-Encoding":
			s3PutParams.ContentEncoding = aws.String(headers.Get(name))
		case "Content-Language":
			s3PutParams.ContentLanguage = aws.String(headers.Get(name))
		case "Content-Disposition":
			s3PutParams.ContentDisposition = aws.String(headers.Get(name))
		case "Cache-Control":
			s3PutParams.CacheControl = aws.String(headers.Get(name))
		case "Expires":
			date, err := time.ParseHTTPDate(headers.Get(name))
			if err == nil {
				s3PutParams.Expires = aws.Time(date)
			}
		}
	}
	_, err = c.uploader.Upload(ctx, &s3PutParams)
	return err
}

func (c Client) HeadObject(ctx context.Context, objectPath string) (obj *storage.Object, err error) {
	key, bucket, err := c.resolveObjectPath(objectPath)
	if err != nil {
		return nil, err
	}
	if len(key) == 0 {
		return nil, errors.New("invalid object path: path is empty")
	}
	s3ObjectInfo, err := c.clt.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})

	if err == nil {
		return &storage.Object{
			Key:          key,
			LastModified: *s3ObjectInfo.LastModified,
			Size:         *s3ObjectInfo.ContentLength,
			ETag:         strings.Trim(aws.ToString(s3ObjectInfo.ETag), `"`),
			StorageClass: string(s3ObjectInfo.StorageClass),
			Headers: map[string][]string{
				"Content-Type":        {aws.ToString(s3ObjectInfo.ContentType)},
				"Content-Encoding":    {aws.ToString(s3ObjectInfo.ContentEncoding)},
				"Content-Language":    {aws.ToString(s3ObjectInfo.ContentLanguage)},
				"Content-Disposition": {aws.ToString(s3ObjectInfo.ContentDisposition)},
				"Cache-Control":       {aws.ToString(s3ObjectInfo.CacheControl)},
				"Expires":             {aws.ToString(s3ObjectInfo.ExpiresString)},
				"Content-Length":      {strconv.FormatInt(aws.ToInt64(s3ObjectInfo.ContentLength), 64)},
				"Last-Modified":       {time.FormatHTTPDate(aws.ToTime(s3ObjectInfo.LastModified))},
				"ETag":                {aws.ToString(s3ObjectInfo.ETag)},
			},
			Mode:     storage.RawPermissionsToMode(s3ObjectInfo.Metadata["x-amz-meta-file-permissions"]),
			Metadata: s3ObjectInfo.Metadata,
		}, nil
	} else {
		var oe *smithy.OperationError
		var re *http.ResponseError
		if errors.As(err, &oe) && errors.As(oe, &re) && re.HTTPStatusCode() == 404 {
			return nil, os.ErrNotExist
		} else {
			return nil, err
		}
	}
}

func (c Client) ListObject(ctx context.Context, objectPrefix string, recursion bool, callback func(key storage.Object)) error {
	prefix, bucket, err := c.resolveObjectPath(objectPrefix)
	if err != nil {
		return err
	}
	var delimiter *string

	if !recursion {
		delimiter = aws.String("/")
	}
	var continueToken *string
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			result, err := c.clt.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
				Bucket:            &bucket,
				Prefix:            &prefix,
				MaxKeys:           aws.Int32(1000),
				ContinuationToken: continueToken,
				Delimiter:         delimiter,
			})
			if err != nil {
				return err
			}
			for _, obj := range result.Contents {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					callback(storage.Object{
						Key:          aws.ToString(obj.Key),
						LastModified: aws.ToTime(obj.LastModified),
						Size:         aws.ToInt64(obj.Size),
						ETag:         strings.Trim(aws.ToString(obj.ETag), `"`),
						StorageClass: string(obj.StorageClass),
					})
				}
			}
			if result.NextContinuationToken == nil || len(aws.ToString(result.NextContinuationToken)) == 0 {
				return nil
			}
			continueToken = result.NextContinuationToken
		}
	}
}

func (c Client) Type() string {
	return "s3"
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
	if endpoint == "" && region != "" {
		endpoint = "s3." + region + ".amazonaws.com"
	}
	if len(endpoint) > 0 && !(strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://")) {
		endpoint = "http://" + endpoint
	}
	clt := s3.NewFromConfig(aws.Config{
		Region: region,
		Credentials: credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID:     accessKeyId,
				SecretAccessKey: accessKeySecret,
			},
		}},
		func(o *s3.Options) {
			o.BaseEndpoint = &endpoint
			o.RetryMaxAttempts = 16
		},
	)
	return &Client{
		clt: clt,
		uploader: manager.NewUploader(clt, func(uploader *manager.Uploader) {
			uploader.BufferProvider = manager.NewBufferedReadSeekerWriteToPool(10 * 1024 * 1024)
		}),
		bucket: v.GetString("bucket"),
		o: Options{
			AccessKeyId:     accessKeyId,
			SecretAccessKey: safeSecretKey,
			Region:          region,
			Endpoint:        endpoint,
			Bucket:          v.GetString("bucket"),
			Type:            Client{}.Type(),
		},
	}, nil
}

func init() {
	storage.RegisterStorage(Client{}.Type(), func(ctx context.Context, v storage.ConfigProvider) (storage.Storage, error) {
		return NewClient(ctx, v)
	})
}
