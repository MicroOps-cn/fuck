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
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/go-kit/log/level"
	"github.com/prometheus/common/model"
	clickhousedriver "gorm.io/driver/clickhouse"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	logs "github.com/MicroOps-cn/fuck/log"
	"github.com/MicroOps-cn/fuck/safe"
	"github.com/MicroOps-cn/fuck/signals"
	w "github.com/MicroOps-cn/fuck/wrapper"
)

type ClickhouseOptions struct {
	Host                  string          `json:"host,omitempty"`
	Username              string          `json:"username,omitempty"`
	Password              safe.String     `json:"password,omitempty"`
	Schema                string          `json:"schema,omitempty"`
	MaxIdleConnections    int32           `json:"max_idle_connections,omitempty"`
	MaxOpenConnections    int32           `json:"max_open_connections,omitempty"`
	MaxConnectionLifeTime *model.Duration `json:"max_connection_life_time,omitempty"`
	TablePrefix           string          `json:"table_prefix,omitempty"`
	SlowThreshold         *model.Duration `json:"slow_threshold,omitempty"`
	DialTimeout           *model.Duration `json:"dial_timeout,omitempty"`
	ReadTimeout           *model.Duration `json:"read_timeout,omitempty"`
	MaxExecutionTime      *model.Duration `json:"max_execution_time,omitempty"`
}

func (x *ClickhouseOptions) Equal(options ClickhouseOptions) bool {
	return !(x.Host != options.Host ||
		x.Username != options.Username ||
		!x.Password.Equal(options.Password) ||
		x.Schema != options.Schema ||
		x.MaxIdleConnections != options.MaxIdleConnections ||
		x.MaxOpenConnections != options.MaxOpenConnections ||
		!durationEqual(x.MaxConnectionLifeTime, options.MaxConnectionLifeTime) ||
		x.TablePrefix != options.TablePrefix ||
		!durationEqual(x.SlowThreshold, options.SlowThreshold) ||
		!durationEqual(x.DialTimeout, options.DialTimeout) ||
		!durationEqual(x.ReadTimeout, options.ReadTimeout) ||
		!durationEqual(x.MaxExecutionTime, options.MaxExecutionTime))
}

func (x *ClickhouseOptions) GetPeer() (string, int) {
	host, port, found := strings.Cut(x.Host, ":")
	if found && len(port) > 0 {
		portNum, err := strconv.Atoi(port)
		if err == nil {
			return host, portNum
		}
	}
	return x.Host, 9000
}

func (x *ClickhouseOptions) String() string {
	return fmt.Sprintf("%s://%s@%s/%s", x.GetType(), x.Username, x.Host, x.Schema)
}

func (x *ClickhouseOptions) GetConnectionString() string {
	return fmt.Sprintf("clickhouse://%s", x.Host)
}

func (x *ClickhouseOptions) GetDBName() string {
	return x.Schema
}

func (x *ClickhouseOptions) GetUsername() string {
	return x.Username
}

func (x *ClickhouseOptions) GetType() string {
	return "clickhouse"
}

func openClickhouseConn(ctx context.Context, slowThreshold time.Duration, options *ClickhouseOptions, autoCreateSchema bool) (*gorm.DB, error) {
	logger := logs.GetContextLogger(ctx)
	dsn, err := options.GetDSN()
	if err != nil {
		return nil, err
	}

	db, err := gorm.Open(
		clickhousedriver.New(clickhousedriver.Config{
			DSN:                dsn,
			DefaultCompression: "LZ4",
		}), &gorm.Config{
			NamingStrategy: schema.NamingStrategy{
				TablePrefix:   options.TablePrefix,
				SingularTable: options.TablePrefix != "",
			},
			Logger:                                   NewLogAdapter(logger, slowThreshold, nil),
			DisableForeignKeyConstraintWhenMigrating: true,
		},
	)
	if err != nil && autoCreateSchema {
		if clickhouseErr, ok := err.(*clickhouse.Exception); ok {
			if clickhouseErr.Code == 1049 {
				level.Info(logger).Log("msg", fmt.Sprintf("auto create schema: %s", options.Schema))
				tmpOpts := *options
				tmpOpts.Schema = "clickhouse"
				db, err = openClickhouseConn(ctx, slowThreshold, &tmpOpts, false)
				if err != nil {
					return nil, err
				}
				err = db.Exec(fmt.Sprintf("CREATE DATABASE `%s`", options.Schema)).Error
				if err != nil {
					return nil, err
				}
				if sqlDB, err := db.DB(); err == nil {
					defer sqlDB.Close()
				}

				return openClickhouseConn(ctx, slowThreshold, options, false)
			}
		}
	}
	return db, err
}

func NewClickhouseClient(ctx context.Context, name string, options ClickhouseOptions) (clt *Client, err error) {
	clt = new(Client)
	clt.options = &options
	logger := logs.GetContextLogger(ctx)
	if options.SlowThreshold != nil {
		clt.slowThreshold = time.Duration(*options.SlowThreshold)
		if err != nil {
			level.Error(logger).Log("msg", fmt.Errorf("failed to connect to clickhouse server: [%s@%s]", options.Username, options.Host), "err", fmt.Errorf("`slow_threshold` option is invalid: %s", err))
			return nil, err
		}
	}
	clt.name = name
	level.Debug(logger).Log("msg", "connect to clickhouse server",
		"host", options.Host, "username", options.Username,
		"schema", options.Schema)

	db, err := openClickhouseConn(ctx, clt.slowThreshold, &options, true)
	if err != nil {
		level.Error(logger).Log("msg", fmt.Errorf("failed to connect to clickhouse server: [%s@%s]", options.Username, options.Host), "err", err)
		return nil, err
	}

	{
		sqlDB, err := db.DB()
		if err != nil {
			level.Error(logger).Log("msg", fmt.Errorf("failed to connect to clickhouse server: [%s@%s]", options.Username, options.Host), "err", err)
			return nil, err
		}
		sqlDB.SetMaxIdleConns(int(options.MaxIdleConnections))
		sqlDB.SetConnMaxLifetime(options.GetStdMaxConnectionLifeTime())
		sqlDB.SetMaxOpenConns(int(options.MaxOpenConnections))
	}

	stopCh := signals.SetupSignalHandler(logger)
	stopCh.PreStop(signals.LevelDB, func() {
		if sqlDB, err := db.DB(); err == nil {
			if err = sqlDB.Close(); err != nil {
				level.Warn(logger).Log("msg", fmt.Errorf("failed to close clickhouse connect: [%s@%s]", options.Username, options.Host), "err", err)
			}
		} else {
			level.Warn(logger).Log("msg", fmt.Errorf("failed to close clickhouse connect: [%s@%s]", options.Username, options.Host), "err", err)
		}
		level.Debug(logger).Log("msg", "Clickhouse connect closed")
	})

	level.Info(logger).Log("msg", "connected to clickhouse server",
		"host", options.Host, "username", options.Username,
		"schema", options.Schema)
	clt.database = db
	clt.statsCollector = collector.Register(clt)
	return clt, nil
}

func (x *ClickhouseOptions) GetStdMaxConnectionLifeTime() time.Duration {
	if x != nil && x.MaxConnectionLifeTime != nil {
		return time.Duration(*x.MaxConnectionLifeTime)
	}
	return time.Second * 30
}

func (x *ClickhouseOptions) GetDSN() (string, error) {
	var u url.URL
	u.Host = x.Host
	passwd, err := x.Password.UnsafeString()
	if err != nil {
		return "", err
	}
	u.User = url.UserPassword(x.Username, passwd)
	u.Scheme = "clickhouse"
	u.Path = fmt.Sprintf("/%s", x.Schema)
	var q url.Values
	if x.DialTimeout != nil {
		q.Set("dial_timeout", x.DialTimeout.String())
	}
	if x.ReadTimeout != nil {
		q.Set("read_timeout", x.ReadTimeout.String())
	}
	if x.MaxExecutionTime != nil {
		q.Set("max_execution_time", x.MaxExecutionTime.String())
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func NewClickhouseOptions() *ClickhouseOptions {
	return &ClickhouseOptions{
		MaxIdleConnections:    2,
		MaxOpenConnections:    100,
		MaxConnectionLifeTime: (*model.Duration)(w.P(30 * time.Second)),
		TablePrefix:           "t_",
		Host:                  "localhost",
		Schema:                "idas",
		Username:              "idas",
	}
}

type ClickhouseClient struct {
	*Client
	options *ClickhouseOptions
}

func (c ClickhouseClient) Options() ClickhouseOptions {
	return *c.options
}

func (c *ClickhouseClient) SetOptions(o *ClickhouseOptions) {
	c.options = o
}

func (c ClickhouseClient) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.options)
}

func (c *ClickhouseClient) UnmarshalJSON(data []byte) (err error) {
	if c.options == nil {
		c.options = NewClickhouseOptions()
	}
	if err = json.Unmarshal(data, c.options); err != nil {
		return err
	}
	if c.Client, err = NewClickhouseClient(context.Background(), "", *c.options); err != nil {
		return err
	}
	return
}
