package nacos

import (
	"net/url"
	"strconv"

	"golang.org/x/net/http/httpproxy"

	"github.com/MicroOps-cn/fuck/clients/tls"
)

type ClientConfig struct {
	ServerAddr  string
	ServerPort  uint64
	Username    string
	Password    string
	NamespaceId string
	Schema      string
	ContextPath string
	TLSOption   *tls.TLSOptions
	Proxy       func(reqURL *url.URL) (*url.URL, error)
}

type ClientOption func(*ClientConfig)

// WithUsername ...
func WithUsername(username string) ClientOption {
	return func(config *ClientConfig) {
		config.Username = username
	}
}

// WithPassword ...
func WithPassword(password string) ClientOption {
	return func(config *ClientConfig) {
		config.Password = password
	}
}

// WithNamespaceId ...
func WithNamespaceId(namespaceId string) ClientOption {
	return func(config *ClientConfig) {
		config.NamespaceId = namespaceId
	}
}

func WithContextPath(path string) ClientOption {
	return func(config *ClientConfig) {
		config.ContextPath = path
	}
}

func WithScheme(scheme string) ClientOption {
	return func(config *ClientConfig) {
		config.Schema = scheme
	}
}

func WithServerPort(num uint64) ClientOption {
	return func(config *ClientConfig) {
		config.ServerPort = num
	}
}

func WithServerAddr(addr string) ClientOption {
	return func(config *ClientConfig) {
		config.ServerAddr = addr
	}
}

func WithServerURL(u string) (ClientOption, error) {
	nacosServerURL, err := url.Parse(u)
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
	return func(config *ClientConfig) {
		config.ServerPort = nacosServerPortNum
		config.ServerAddr = nacosServerURL.Hostname()
		config.Schema = nacosServerURL.Scheme
		config.ContextPath = nacosServerURL.Path
	}, nil
}

func WithTLSOption(tlsOptions tls.TLSOptions) ClientOption {
	return func(config *ClientConfig) {
		config.TLSOption = &tlsOptions
	}
}

func WithProxy(addr string) ClientOption {
	return func(config *ClientConfig) {
		if addr != "" {
			config.Proxy = (&httpproxy.Config{
				HTTPProxy:  addr,
				HTTPSProxy: addr,
			}).ProxyFunc()
		}
	}
}
