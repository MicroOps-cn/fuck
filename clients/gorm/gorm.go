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

package gorm

import (
	"context"
	"fmt"
	"sync"
	"time"

	logs "github.com/MicroOps-cn/fuck/log"
	"github.com/go-kit/log/level"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

type Database struct {
	*gorm.DB
}

type DBOptions interface {
	GetConnectionString() string
	GetPeer() (string, int)
	GetDBName() string
	GetUsername() string
	GetType() string
}

type Client struct {
	name          string
	database      *Database
	slowThreshold time.Duration
	tracer        trace.Tracer
	tracerInitial sync.Once
	options       DBOptions
}

type Handler func(*gorm.DB)

type Interceptor func(name string, next Handler) Handler

type Processor interface {
	Get(name string) func(*gorm.DB)
	Replace(name string, handler func(*gorm.DB)) error
}

const instrumentationName = "github.com/MicroOps-cn/fuck/clients/gorm"

func (c *Client) Session(ctx context.Context) *Database {
	host, port := c.options.GetPeer()
	c.tracerInitial.Do(func() {
		c.tracer = otel.GetTracerProvider().Tracer(instrumentationName, trace.WithInstrumentationAttributes(
			attribute.String("db.connection_string", c.options.GetConnectionString()),
			attribute.String("db.name", c.options.GetDBName()),
			attribute.String("db.system", c.options.GetType()),
			attribute.String("db.user", c.options.GetUsername()),
			attribute.String("net.peer.name", host),
			attribute.Int("net.peer.port", port),
		))
	})
	logger := logs.GetContextLogger(ctx)
	session := &gorm.Session{Logger: NewLogAdapter(logger, c.slowThreshold, c.tracer)}
	if conn := ctx.Value(gormConn{}); conn != nil {
		switch db := conn.(type) {
		case *Database:
			return &Database{DB: db.Session(session)}
		case *gorm.DB:
			return &Database{DB: db.Session(session)}
		default:
			level.Warn(logger).Log("msg", "Unknown context value type.", "name", fmt.Sprintf("%T", gormConn{}), "value", fmt.Sprintf("%T", conn))
		}
	}
	return &Database{DB: c.database.Session(session).WithContext(ctx)}
}

type ConnType interface {
	*Database | *gorm.DB
}

func WithConnContext[T ConnType](ctx context.Context, client T) context.Context {
	return context.WithValue(ctx, gormConn{}, client)
}

type gormConn struct{}
