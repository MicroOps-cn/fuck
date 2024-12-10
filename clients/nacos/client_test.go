package nacos

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"testing"

	uuid "github.com/satori/go.uuid"
)

type NacosModel struct {
	Namespace string `tfsdk:"namespace"`
	Password  string `tfsdk:"password"`
	Url       string `tfsdk:"url"`
	Username  string `tfsdk:"username"`
	Proxy     string `tfsdk:"proxy"`
}

func newNacosClientFromModel(data NacosModel) (IConfigClient, error) {
	nacosServerURL, err := url.Parse(data.Url)
	if err != nil {
		return nil, err
	}
	if nacosServerURL.Scheme == "" {
		nacosServerURL.Scheme = "http"
	}
	if nacosServerURL.Path == "" {
		nacosServerURL.Path = "/nacos"
	}
	nacosServerPort := nacosServerURL.Port()
	if nacosServerPort == "" {
		nacosServerPort = "80"
	}
	nacosServerPortNum, err := strconv.ParseUint(nacosServerPort, 10, 64)
	if err != nil {
		return nil, err
	}
	client, err := NewConfigClient(
		WithUsername(data.Username),
		WithPassword(data.Password),
		WithNamespaceId(data.Namespace),
		WithServerPort(nacosServerPortNum),
		WithServerAddr(nacosServerURL.Hostname()),
		WithScheme(nacosServerURL.Scheme),
		WithContextPath(nacosServerURL.Path),
		WithProxy(data.Proxy),
	)
	if err != nil {
		return nil, err
	}

	testDataId := fmt.Sprintf("test-%s.txt", uuid.Must(uuid.NewV4()))

	ok, err := client.PublishConfig(ConfigParam{DataId: testDataId, Group: "CONNECT_TEST_GROUP", Type: "text", Content: "TestContent"})
	if err != nil {
		return nil, err
	} else if !ok {
		return nil, fmt.Errorf("failed to publish test config, ret!=ok")
	}
	defer func() {
		_, _ = client.DeleteConfig(ConfigParam{DataId: testDataId, Group: "CONNECT_TEST_GROUP"})
	}()
	testContent, err := client.GetConfig(ConfigParam{DataId: testDataId, Group: "CONNECT_TEST_GROUP"})
	if err != nil {
		return nil, err
	}
	if len(testContent) == 0 {
		return nil, fmt.Errorf("failed to get test config file, content is empty")
	}

	return client, nil
}

func Test_newNacosClient(t *testing.T) {
	type args struct {
		data NacosModel
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{{
		name: "success",
		args: args{data: NacosModel{
			Url:       os.Getenv("NACOS_URL"),
			Username:  os.Getenv("NACOS_USERNAME"),
			Password:  os.Getenv("NACOS_PASSWORD"),
			Namespace: os.Getenv("NACOS_NAMESPACE"),
			Proxy:     os.Getenv("NACOS_PROXY"),
		}},
		wantErr: false,
	}, {
		name: "403",
		args: args{data: NacosModel{
			Url:       os.Getenv("NACOS_URL"),
			Username:  os.Getenv("NACOS_USERNAME"),
			Password:  os.Getenv("NACOS_PASSWORD"),
			Namespace: os.Getenv("NACOS_NAMESPACE") + "1",
			Proxy:     os.Getenv("NACOS_PROXY"),
		}},
		wantErr: true,
	}, {
		name: "invalid password",
		args: args{data: NacosModel{
			Url:       os.Getenv("NACOS_URL"),
			Username:  os.Getenv("NACOS_USERNAME"),
			Password:  os.Getenv("NACOS_PASSWORD") + "AA",
			Namespace: os.Getenv("NACOS_NAMESPACE"),
			Proxy:     os.Getenv("NACOS_PROXY"),
		}},
		wantErr: true,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newNacosClientFromModel(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("newNacosClientFromModel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
