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
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/MicroOps-cn/fuck/log"
	"github.com/MicroOps-cn/fuck/signals"
	w "github.com/MicroOps-cn/fuck/wrapper"
)

type Client struct {
	client  *redis.Client
	options *RedisOptions
}

// Merge implement proto.Merger
func (r *Client) Merge(src proto.Message) {
	if s, ok := src.(*Client); ok {
		r.options = s.options
		r.client = s.client
	}
}

// String implement proto.Message
func (r Client) String() string {
	return r.options.String()
}

// ProtoMessage implement proto.Message
func (r *Client) ProtoMessage() {
	r.options.ProtoMessage()
}

// Reset *implement proto.Message*
func (r *Client) Reset() {
	r.options.Reset()
}

func (r Client) Marshal() ([]byte, error) {
	return proto.Marshal(r.options)
}

func (r *Client) Unmarshal(data []byte) (err error) {
	if r.options == nil {
		r.options = &RedisOptions{}
	}
	if err = proto.Unmarshal(data, r.options); err != nil {
		return err
	}
	if r.client, err = NewRedisClient(context.Background(), r.options); err != nil {
		return err
	}
	return
}

func (r Client) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.options)
}

func (r *Client) UnmarshalJSON(data []byte) (err error) {
	if r.options == nil {
		r.options = &RedisOptions{}
	}
	if err = json.Unmarshal(data, r.options); err != nil {
		return err
	}
	if r.client, err = NewRedisClient(context.Background(), r.options); err != nil {
		return err
	}
	return
}

func (o *RedisOptions) UnmarshalJSON(data []byte) (err error) {
	if len(data) > 0 && data[0] == '"' {
		return json.Unmarshal(data, &o.Url)
	}
	type plain RedisOptions
	return json.Unmarshal(data, (*plain)(o))
}

func NewRedisClient(ctx context.Context, option *RedisOptions) (*redis.Client, error) {
	logger := log.GetContextLogger(ctx)
	options, err := redis.ParseURL(option.Url)
	if err != nil {
		level.Error(logger).Log("msg", "Failed to parse Redis URL", "err", err, "url", option.Url)
		return nil, err
	}

	client := redis.NewClient(options)
	if err = client.Ping().Err(); err != nil {
		level.Error(logger).Log("msg", "Redis connection failed", "err", err)
		_ = client.Close()
		return nil, err
	}

	stopCh := signals.SetupSignalHandler(logger)
	if stopCh != nil {
		stopCh.Add(1)
		go func() {
			<-stopCh.Channel()
			stopCh.WaitRequest()
			if err = client.Close(); err != nil {
				level.Error(logger).Log("msg", "Redis client shutdown error", "err", err)
				time.Sleep(1 * time.Second)
			}
			level.Debug(logger).Log("msg", "Closed Redis connection", "err", err)
			stopCh.Done()
		}()
	}
	return client, nil
}

func (o *RedisOptions) GetPeer() (string, int) {
	u, err := url.Parse(o.Url)
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
