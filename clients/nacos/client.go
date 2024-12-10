package nacos

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/log/level"
	uuid "github.com/satori/go.uuid"

	"github.com/MicroOps-cn/fuck/buffer"
	"github.com/MicroOps-cn/fuck/errors"
	logs "github.com/MicroOps-cn/fuck/log"
)

type IConfigClient interface {
	// GetConfig use to get config from nacos server
	// dataId  require
	// group   require
	// tenant ==>nacos.namespace optional
	GetConfig(param ConfigParam) (string, error)

	// ListenConfig use to listen a config from nacos server
	// dataId  require
	// group   require
	// onChange require
	// tenant ==>nacos.namespace optional
	ListenConfig(param ConfigParam) (listenId string, data *ConfigQueryResponse)

	// PublishConfig use to publish config to nacos server
	// dataId  require
	// group   require
	// content require
	// tenant ==>nacos.namespace optional
	PublishConfig(param ConfigParam) (bool, error)

	// DeleteConfig use to delete config
	// dataId  require
	// group   require
	// tenant ==>nacos.namespace optional
	DeleteConfig(param ConfigParam) (bool, error)

	// SearchConfig use to search nacos config
	// search  require search=accurate--精确搜索  search=blur--模糊搜索
	// group   option
	// dataId  option
	// tenant ==>nacos.namespace optional
	// pageNo  option,default is 1
	// pageSize option,default is 10
	SearchConfig(param SearchConfigParam) (*ConfigPage, error)

	QueryConfig(param ConfigParam) (content *ConfigQueryResponse, err error)
}

type watchItem struct {
	dataId       string
	group        string
	lastHash     *string
	backoff      time.Duration
	backoffStart time.Time
	onChanges    map[string]OnChangeFunc
}

type watchdog struct {
	targets       []*watchItem
	once          sync.Once
	locker        sync.RWMutex
	client        *configClient
	ticker        *time.Ticker
	backoffTicker *time.Ticker
}

func (c *watchdog) Register(dataId string, group string, onChange OnChangeFunc) (id string) {
	id = uuid.Must(uuid.NewV4()).String()
	if !func() bool {
		c.locker.RLock()
		defer c.locker.RUnlock()
		for _, target := range c.targets {
			if target.dataId == dataId && target.group == group {
				target.onChanges[id] = onChange
				return true
			}
		}
		return false
	}() {
		c.locker.Lock()
		defer c.locker.Unlock()
		for _, target := range c.targets {
			if target.dataId == dataId && target.group == group {
				target.onChanges[id] = onChange
				return id
			}
		}
		c.targets = append(c.targets, &watchItem{
			dataId: dataId,
			group:  group,
			onChanges: map[string]OnChangeFunc{
				id: onChange,
			},
		})
	}
	return id
}

func (c *watchdog) getWatchItems() []*watchItem {
	c.locker.RLock()
	defer c.locker.RUnlock()
	items := make([]*watchItem, len(c.targets))
	copy(items, c.targets)
	return items
}

func (c *watchdog) runItemWatch(ctx context.Context, dataId string, group string) *ConfigQueryResponse {
	item, data := c.queryItem(ctx, dataId, group)
	if item != nil && data != nil {
		if item.lastHash == nil {
			item.lastHash = new(string)
			return data
		} else if *item.lastHash == data.ContentMd5 {
			return data
		}
		item.lastHash = &data.ContentMd5
		if item.onChanges != nil {
			for _, onChangeFunc := range item.onChanges {
				onChangeFunc(data)
			}
		}
	}
	return data
}

func (c *watchdog) queryItem(ctx context.Context, dataId string, group string) (*watchItem, *ConfigQueryResponse) {
	data, err := c.client.QueryConfig(ConfigParam{DataId: dataId, Group: group})
	c.locker.RLock()
	defer c.locker.RUnlock()
	for idx, target := range c.targets {
		if target.dataId == dataId && target.group == group {
			if err != nil || data.Code != 0 {
				c.targets[idx].backoffStart = time.Now()
				logger := logs.GetContextLogger(ctx)
				if target.backoff <= 0 {
					if err != nil {
						level.Error(logger).Log("msg", "watch config failed", "err", err, "group", group, "dataId", dataId)
					} else if data.Code != 0 {
						level.Error(logger).Log("msg", "watch config failed", "err", data.Error, "code", data.Code, "message", data.Message, "code", data.Code, "group", group, "dataId", dataId)
					}
					c.targets[idx].backoff = time.Second
				} else if target.backoff < 2^8*time.Second {
					c.targets[idx].backoff *= 2
					if err != nil {
						if !errors.IsNotFount(err) {
							level.Debug(logger).Log("msg", "watch config failed", "err", err, "group", group, "dataId", dataId, "backoff", c.targets[idx].backoff)
						}
					} else if data.Code != 0 {
						level.Debug(logger).Log("msg", "watch config failed", "err", data.Error, "code", data.Code, "message", data.Message, "code", data.Code, "group", group, "dataId", dataId, "backoff", c.targets[idx].backoff)
					}
				}
			} else {
				if target.backoff > 0 {
					c.targets[idx].backoff = 0
				}
				return target, data
			}
		}
	}
	return nil, nil
}

func (c *watchdog) loopWatch(ctx context.Context) {
	c.ticker = time.NewTicker(5 * time.Second)
	c.backoffTicker = time.NewTicker(time.Second)
	for {
		select {
		case <-c.ticker.C:
			items := c.getWatchItems()
			for _, item := range items {
				if item.backoff <= 0 {
					c.runItemWatch(ctx, item.dataId, item.group)
				}
			}
		case <-c.backoffTicker.C:
			items := c.getWatchItems()
			for _, item := range items {
				if item.backoff > 0 && time.Now().After(item.backoffStart.Add(item.backoff)) {
					c.runItemWatch(ctx, item.dataId, item.group)
				}
			}
		}
	}
}

type configClient struct {
	cfg           ClientConfig
	token         string
	client        *http.Client
	tokenExpireAt time.Time
	watchdog      watchdog
}

func (c *configClient) ListenConfig(param ConfigParam) (listenId string, data *ConfigQueryResponse) {
	ctx, _ := logs.NewContextLogger(context.Background())
	listenId = c.watchdog.Register(param.DataId, param.Group, param.OnChange)
	c.watchdog.once.Do(func() {
		c.watchdog.client = c
		go c.watchdog.loopWatch(ctx)
	})
	data = c.watchdog.runItemWatch(ctx, param.DataId, param.Group)
	return listenId, data
}

type GetConfigResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

func (c *configClient) GetConfig(param ConfigParam) (string, error) {
	content, err := c.QueryConfig(param)
	if err != nil {
		return "", err
	}
	return content.Data, nil
}

type PublishConfigResponse struct {
	Timestamp time.Time `json:"timestamp"`
	Error     string    `json:"error"`
	Message   string    `json:"message"`
	Code      int       `json:"code"`
	Data      bool      `json:"data"`
}

func (c *configClient) doRequest(method string, api string, data url.Values) (*http.Response, error) {
	if time.Until(c.tokenExpireAt) < time.Minute {
		if err := c.Authorization(); err != nil {
			return nil, err
		}
	}
	for name := range data {
		if data.Get(name) == "" {
			data.Del(name)
		}
	}
	var err error
	var r *http.Request
	switch method {
	case "POST", "PUT":
		r, err = http.NewRequest(method, c.url(api), strings.NewReader(data.Encode()))
		if err != nil {
			return nil, err
		}
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	default:
		u, err := url.Parse(c.url(api))
		if err != nil {
			return nil, err
		}
		u.RawQuery = data.Encode()
		r, err = http.NewRequest(method, u.String(), nil)
		if err != nil {
			return nil, err
		}
	}
	r.Header.Set("Authorization", "Bearer "+c.token)
	return c.client.Do(r)
}

func (c configClient) PublishConfig(param ConfigParam) (bool, error) {
	resp, err := c.doRequest("POST", "/v2/cs/config", url.Values{
		"namespaceId": {c.cfg.NamespaceId},
		"tenant":      {c.cfg.NamespaceId},
		"group":       {param.Group},
		"dataId":      {param.DataId},
		"content":     {param.Content},
		"tag":         {param.Tag},
		"appName":     {param.AppName},
		"srcUser":     {param.SrcUser},
		"configTags":  {param.ConfigTags},
		"desc":        {param.Desc},
		"effect":      {param.Effect},
		"type":        {param.Type},
		"schema":      {param.Schema},
	})
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	var respBody PublishConfigResponse

	buf, err := buffer.NewPreReader(resp.Body, 1024)
	if err != nil {
		return false, fmt.Errorf("publish failed: failed to read data: %s", err)
	}
	decoder := json.NewDecoder(buf)
	if err = decoder.Decode(&respBody); err != nil {
		if resp.StatusCode == http.StatusOK {
			return false, fmt.Errorf("publish failed: failed to decode response body: %s", err)
		}
		return false, fmt.Errorf("publish failed: body=%s,err=%s", buf.Buffer(), err)
	}

	if respBody.Code != 0 {
		if respBody.Error != "" {
			return false, fmt.Errorf("publish failed: %s", respBody.Error)
		} else if respBody.Message != "" {
			return false, fmt.Errorf("publish failed: %s", respBody.Message)
		}
		return false, fmt.Errorf("publish failed: code=%d", respBody.Code)
	}
	return respBody.Data, nil
}

type DeleteConfigResponse struct {
	Timestamp time.Time `json:"timestamp"`
	Error     string    `json:"error"`
	Message   string    `json:"message"`
	Code      int       `json:"code"`
	Data      bool      `json:"data"`
}

func (c configClient) DeleteConfig(param ConfigParam) (bool, error) {
	resp, err := c.doRequest("DELETE", "/v2/cs/config", url.Values{
		"namespaceId": {c.cfg.NamespaceId},
		"tenant":      {c.cfg.NamespaceId},
		"group":       {param.Group},
		"dataId":      {param.DataId},
		"tag":         {param.Tag},
	})
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	var respBody DeleteConfigResponse

	buf, err := buffer.NewPreReader(resp.Body, 1024)
	if err != nil {
		return false, fmt.Errorf("delete failed: failed to read data: %s", err)
	}
	decoder := json.NewDecoder(buf)
	if err = decoder.Decode(&respBody); err != nil {
		if resp.StatusCode == http.StatusOK {
			return false, fmt.Errorf("delete failed: failed to decode response body: %s", err)
		}
		return false, fmt.Errorf("delete failed: body=%s,err=%s", buf.Buffer(), err)
	}
	if respBody.Code != 0 {
		if respBody.Error != "" {
			return false, fmt.Errorf("delete failed: %s", respBody.Error)
		} else if respBody.Message != "" {
			return false, fmt.Errorf("publish failed: %s", respBody.Message)
		}
		return false, fmt.Errorf("delete failed: code=%d", respBody.Code)
	}
	return respBody.Data, nil
}

func (c configClient) SearchConfig(param SearchConfigParam) (*ConfigPage, error) {
	//TODO implement me
	panic("implement me")
}

type ConfigQueryResponse struct {
	Error      string `json:"error"`
	Message    string `json:"message"`
	Code       int    `json:"code"`
	Data       string `json:"data"`
	ContentMd5 string `json:"-"`
	ConfigType string `json:"-"`
}

func (c configClient) QueryConfig(param ConfigParam) (content *ConfigQueryResponse, err error) {
	resp, err := c.doRequest("GET", "/v2/cs/config", url.Values{
		"namespaceId": {c.cfg.NamespaceId},
		"tenant":      {c.cfg.NamespaceId},
		"group":       {param.Group},
		"dataId":      {param.DataId},
		"tag":         {param.Tag},
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var respBody ConfigQueryResponse
	respBody.ContentMd5 = resp.Header.Get("Content-MD5")
	buf, err := buffer.NewPreReader(resp.Body, 1024)
	if err != nil {
		return nil, fmt.Errorf("query failed: failed to read data: %s", err)
	}
	decoder := json.NewDecoder(buf)
	if err = decoder.Decode(&respBody); err != nil {
		if resp.StatusCode == http.StatusOK {
			return nil, fmt.Errorf("query failed: failed to decode response body: %s", err)
		}
		return nil, fmt.Errorf("query failed: body=%s,err=%s", buf.Buffer(), err)
	}
	if respBody.Code != 0 {
		if respBody.Code == 20004 {
			return nil, errors.NotFoundError
		}
		if respBody.Error != "" {
			return nil, errors.NewError(respBody.Code, fmt.Sprintf("get config failed: %s", respBody.Error))
		} else if respBody.Message != "" {
			return nil, errors.NewError(respBody.Code, fmt.Sprintf("get config failed: %s", respBody.Message))
		}
		return nil, errors.NewError(respBody.Code, fmt.Sprintf("get config failed: code=%s", respBody.Code))
	}

	return &respBody, nil
}

type LoginResponse struct {
	AccessToken string    `json:"accessToken"`
	TokenTTL    int       `json:"tokenTtl"`
	Timestamp   time.Time `json:"timestamp"`
	Error       string    `json:"error"`
	Message     string    `json:"message"`
	Path        string    `json:"path"`
}

func (c *configClient) url(api string) string {
	contextPath := c.cfg.ContextPath
	if contextPath == "" {
		contextPath = "/nacos"
	} else if contextPath[0] != '/' {
		contextPath = "/" + contextPath
	}

	return fmt.Sprintf("%s://%s:%d%s", c.cfg.Schema, c.cfg.ServerAddr, c.cfg.ServerPort, path.Join(contextPath, api))
}

func (c *configClient) Authorization() (err error) {
	resp, err := c.client.PostForm(c.url("v1/auth/login"), url.Values{"username": {c.cfg.Username}, "password": {c.cfg.Password}})
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	var respBody LoginResponse
	buf, err := buffer.NewPreReader(resp.Body, 1024)
	if err != nil {
		return fmt.Errorf("authentication failed: failed to read data: %s", err)
	}
	if strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		decoder := json.NewDecoder(buf)
		if err = decoder.Decode(&respBody); err != nil {
			if resp.StatusCode == http.StatusOK {
				return fmt.Errorf("authentication failed: failed to decode response body: %s", err)
			}
			return fmt.Errorf("authentication failed: body=%s,err=%s", buf.Buffer(), err)
		}
		if respBody.Error != "" {
			return fmt.Errorf("authentication failed: %s", respBody.Error)
		} else if respBody.Message != "" {
			return fmt.Errorf("authentication failed: %s", respBody.Message)
		}
		if len(respBody.AccessToken) == 0 {
			return fmt.Errorf("authentication failed: accessToken is null")
		}
		c.token = respBody.AccessToken
		c.tokenExpireAt = time.Now().Add(time.Duration(respBody.TokenTTL) * time.Second)
		return nil
	}
	return fmt.Errorf("authentication failed, invalid response Content-Type: Content-Type=%s, body=%s, status_code=%d", resp.Header.Get("Content-Type"), buf.Buffer(), resp.StatusCode)
}

func NewConfigClient(opts ...ClientOption) (IConfigClient, error) {
	cfg := ClientConfig{
		Schema:      "http",
		ContextPath: "/nacos",
		ServerAddr:  "127.0.0.1",
		ServerPort:  8848,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	cc := &configClient{cfg: cfg, client: &http.Client{
		Transport: &http.Transport{
			Proxy: func(r *http.Request) (*url.URL, error) {
				if cfg.Proxy != nil {
					return cfg.Proxy(r.URL)
				}
				return http.ProxyFromEnvironment(r)
			},
			DialContext: func(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
				return dialer.DialContext
			}(&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}),
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}}
	if err := cc.Authorization(); err != nil {
		return nil, err
	}
	return cc, nil
}

func init() {
}
