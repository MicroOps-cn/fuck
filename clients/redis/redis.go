/*
 Copyright © 2022 MicroOps-cn.

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

package redis

import (
	"bytes"
	"context"
	"encoding"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/go-kit/log/level"
	"github.com/go-redis/redis"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/pkg/errors"
	"github.com/prometheus/common/model"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/MicroOps-cn/fuck/clients/tls"
	"github.com/MicroOps-cn/fuck/log"
	"github.com/MicroOps-cn/fuck/safe"
	"github.com/MicroOps-cn/fuck/signals"
	w "github.com/MicroOps-cn/fuck/wrapper"
)

type Client struct {
	client  *redis.Client
	options *Options
}

func (r Client) MarshalJSONPB(_ *jsonpb.Marshaler) ([]byte, error) {
	return r.MarshalJSON()
}

func (r Client) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.options)
}

func (r *Client) UnmarshalJSON(data []byte) (err error) {
	if r.options == nil {
		r.options = &Options{}
	}
	if err = json.Unmarshal(data, r.options); err != nil {
		return err
	}
	if r.client, err = NewRedisClient(context.Background(), r.options); err != nil {
		return err
	}
	return
}

type Options struct {
	o        *redis.Options
	URL      string      `json:"url,omitempty"`
	Password safe.String `json:"password"`
	// Database to be selected after connecting to the server.
	DB *int `json:"db"`

	// Maximum number of retries before giving up.
	// Default is to not retry failed commands.
	MaxRetries *int `json:"max_retries"`
	// Minimum backoff between each retry.
	// Default is 8 milliseconds; -1 disables backoff.
	MinRetryBackoff *model.Duration `json:"min_retry_backoff"`
	// Maximum backoff between each retry.
	// Default is 512 milliseconds; -1 disables backoff.
	MaxRetryBackoff *model.Duration `json:"max_retry_backoff"`

	// Dial timeout for establishing new connections.
	// Default is 5 seconds.
	DialTimeout *model.Duration `json:"dial_timeout"`
	// Timeout for socket reads. If reached, commands will fail
	// with a timeout instead of blocking. Use value -1 for no timeout and 0 for default.
	// Default is 3 seconds.
	ReadTimeout *model.Duration `json:"read_timeout"`
	// Timeout for socket writes. If reached, commands will fail
	// with a timeout instead of blocking.
	// Default is ReadTimeout.
	WriteTimeout *model.Duration `json:"write_timeout"`

	// Maximum number of socket connections.
	// Default is 10 connections per every CPU as reported by runtime.NumCPU.
	PoolSize *int `json:"pool_size"`
	// Minimum number of idle connections which is useful when establishing
	// new connection is slow.
	MinIdleConns *int `json:"min_idle_conns"`
	// Connection age at which client retires (closes) the connection.
	// Default is to not close aged connections.
	MaxConnAge *time.Duration `json:"max_conn_age"`
	// Amount of time client waits for connection if all connections
	// are busy before returning an error.
	// Default is ReadTimeout + 1 second.
	PoolTimeout *time.Duration `json:"pool_timeout"`
	// Amount of time after which client closes idle connections.
	// Should be less than server's timeout.
	// Default is 5 minutes. -1 disables idle timeout check.
	IdleTimeout *time.Duration `json:"idle_timeout"`
	// Frequency of idle checks made by idle connections reaper.
	// Default is 1 minute. -1 disables idle connections reaper,
	// but idle connections are still discarded by the client
	// if IdleTimeout is set.
	IdleCheckFrequency *time.Duration  `json:"idle_check_frequency"`
	TLSConfig          *tls.TLSOptions `json:"tls_config" yaml:"tls_config" mapstructure:"tls_config"`
}

func (o *Options) UnmarshalJSON(data []byte) (err error) {
	if len(data) > 0 && data[0] == '"' {
		if err = json.Unmarshal(data, &o.URL); err != nil {
			return err
		}
		o.o, err = redis.ParseURL(o.URL)
		if err != nil {
			return fmt.Errorf("failed to parse redis url: %s", err)
		}
		return o.Password.SetValue(w.DefaultString(o.Password.String(), o.o.Password))
	}
	type plain Options
	err = json.Unmarshal(data, (*plain)(o))
	if err != nil {
		return err
	}
	o.o, err = redis.ParseURL(o.URL)
	if err != nil {
		return fmt.Errorf("failed to parse redis url: %s", err)
	}
	if err = o.Password.SetValue(w.DefaultString(o.Password.String(), o.o.Password)); err != nil {
		return err
	}
	o.o.DB = *w.DefaultPointer(o.DB, &o.o.DB)
	o.o.MaxRetries = *w.DefaultPointer(o.MaxRetries, &o.o.MaxRetries)
	o.o.MinRetryBackoff = *w.DefaultPointer[time.Duration]((*time.Duration)(o.MinRetryBackoff), &o.o.MinRetryBackoff)
	o.o.MaxRetryBackoff = *w.DefaultPointer[time.Duration]((*time.Duration)(o.MaxRetryBackoff), &o.o.MaxRetryBackoff)
	o.o.DialTimeout = *w.DefaultPointer[time.Duration]((*time.Duration)(o.DialTimeout), &o.o.DialTimeout)
	o.o.ReadTimeout = *w.DefaultPointer[time.Duration]((*time.Duration)(o.ReadTimeout), &o.o.ReadTimeout)
	o.o.WriteTimeout = *w.DefaultPointer[time.Duration]((*time.Duration)(o.WriteTimeout), &o.o.WriteTimeout)
	o.o.PoolSize = *w.DefaultPointer(o.PoolSize, &o.o.PoolSize)
	o.o.MinIdleConns = *w.DefaultPointer(o.MinIdleConns, &o.o.MinIdleConns)
	o.o.MaxConnAge = *w.DefaultPointer(o.MaxConnAge, &o.o.MaxConnAge)
	o.o.PoolTimeout = *w.DefaultPointer(o.PoolTimeout, &o.o.PoolTimeout)
	o.o.IdleTimeout = *w.DefaultPointer(o.IdleTimeout, &o.o.IdleTimeout)
	o.o.IdleCheckFrequency = *w.DefaultPointer(o.IdleCheckFrequency, &o.o.IdleCheckFrequency)
	if o.TLSConfig != nil {
		tlsConfig, err := tls.NewTLSConfig(o.TLSConfig)
		if err != nil {
			return err
		}
		o.o.TLSConfig = tlsConfig
	}
	return nil
}

func NewRedisClient(ctx context.Context, option *Options) (*redis.Client, error) {
	var err error
	logger := log.GetContextLogger(ctx)
	o := *option.o
	o.Password, err = option.Password.UnsafeString()
	if err != nil {
		return nil, err
	}
	level.Debug(logger).Log("msg", "connect to redis server", "host", o.Addr, "db", o.DB)
	client := redis.NewClient(&o)
	if err = client.Ping().Err(); err != nil {
		level.Error(logger).Log("msg", "Redis connection failed", "err", err)
		_ = client.Close()
		return nil, err
	}

	level.Info(logger).Log("msg", "connected to redis server", "host", option.o.Addr, "db", option.o.DB)
	stopCh := signals.SetupSignalHandler(logger)
	stopCh.PreStop(signals.LevelDB, func() {
		if err = client.Close(); err != nil {
			level.Error(logger).Log("msg", "Redis client shutdown error", "err", err)
			time.Sleep(1 * time.Second)
		}
		level.Debug(logger).Log("msg", "Closed Redis connection")
	})
	return client, nil
}

func (o *Options) GetPeer() (string, int) {
	u, err := url.Parse(o.URL)
	if err != nil {
		return "", 0
	}
	port, _ := strconv.Atoi(u.Port())
	return u.Hostname(), port
}

func parseArg(v interface{}) (string, error) {
	switch v := v.(type) {
	case nil:
		return "", nil
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	case int:
		return strconv.Itoa(v), nil
	case int8:
		return strconv.Itoa(int(v)), nil
	case int16:
		return strconv.Itoa(int(v)), nil
	case int32:
		return strconv.Itoa(int(v)), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case uint:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint64:
		return strconv.FormatUint(v, 10), nil
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 64), nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(v), nil
	case encoding.BinaryMarshaler:
		b, err := v.MarshalBinary()
		if err != nil {
			return "", err
		}
		return string(b), nil
	default:
		return "", fmt.Errorf(
			"redis: can't marshal %T (implement encoding.BinaryMarshaler)", v)
	}
}

type Cmder struct {
	redis.Cmder
}

func (c Cmder) String() string {
	var buf bytes.Buffer
	for idx, arg := range c.Args() {
		if idx != 0 {
			buf.WriteString(" ")
		}
		if s, err := parseArg(arg); err != nil {
			buf.Write(w.M(json.Marshal(fmt.Sprintf("<%s>", err))))
		} else {
			buf.WriteString(s)
		}
	}
	return buf.String()
}

const instrumentationName = "github.com/MicroOps-cn/fuck/clients/redis"

func (r *Client) Redis(ctx context.Context) *redis.Client {
	tracer := otel.GetTracerProvider().Tracer(instrumentationName)
	logger := log.GetContextLogger(ctx, log.WithCaller(7))
	session := r.client.WithContext(ctx)
	host, port := r.options.GetPeer()
	session.WrapProcess(func(oldProcess func(cmd redis.Cmder) error) func(cmd redis.Cmder) error {
		return func(cmd redis.Cmder) (err error) {
			_, span := tracer.Start(ctx, "ExecuteRedisCommand."+cmd.Name(),
				trace.WithAttributes(
					attribute.String("db.info", r.client.String()),
					attribute.String("net.peer.name", host),
					attribute.Int("net.peer.port", port),
					attribute.String("db.statement", (&Cmder{Cmder: cmd}).String()),
					attribute.String("db.system", "redis"),
				),
			)
			defer func() {
				span.End()
				if err != nil {
					if err != redis.Nil {
						span.SetStatus(codes.Error, err.Error())
						level.Error(logger).Log("msg", "failed to exec Redis Command", "cmd", Cmder{Cmder: cmd}, "err", err)
					} else {
						span.SetStatus(codes.Error, err.Error())
						level.Debug(logger).Log("msg", "failed to exec Redis Command", "cmd", Cmder{Cmder: cmd}, "err", err)
					}
				} else {
					span.SetStatus(codes.Ok, "")
					level.Debug(logger).Log("msg", "exec Redis Command", "cmd", Cmder{Cmder: cmd})
				}
			}()
			return oldProcess(cmd)
		}
	})
	return session
}

var ErrStopLoop = errors.New("stop")

func ForeachSet(ctx context.Context, c *redis.Client, key string, cursor uint64, pageSize int64, f func(key, val string) error) (err error) {
	var listLength int64
	if ret, err := c.SCard(key).Result(); err != nil {
		return err
	} else if listLength = ret; listLength == 0 {
		return nil
	}
	if pageSize == 0 {
		pageSize = 100
	}
	var ret []string
	for {
		select {
		case <-ctx.Done():
		default:
			ret, cursor, err = c.SScan(key, cursor, "*", pageSize).Result()
			if err != nil {
				return err
			}
			for _, member := range ret {
				if err = f(key, member); err != nil {
					if err == ErrStopLoop {
						break
					}
					return err
				}
			}
			if int64(len(ret)) < pageSize {
				return nil
			}
		}
	}
}
