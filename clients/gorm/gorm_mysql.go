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
	w "github.com/MicroOps-cn/fuck/wrapper"
	"github.com/prometheus/common/model"
	"time"

	logs "github.com/MicroOps-cn/fuck/log"
	"github.com/go-kit/log/level"
	mysqldriver "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"github.com/MicroOps-cn/fuck/signals"
	mysql "github.com/go-sql-driver/mysql"
)

type MySQLOptions struct {
	Host                  string          `json:"host,omitempty"`
	Username              string          `json:"username,omitempty"`
	Password              string          `json:"password,omitempty"`
	Schema                string          `json:"schema,omitempty"`
	MaxIdle               int32           `json:"max_idle,omitempty"`
	MaxIdleConnections    int32           `json:"max_idle_connections,omitempty"`
	MaxOpenConnections    int32           `json:"max_open_connections,omitempty"`
	MaxConnectionLifeTime *model.Duration `json:"max_connection_life_time,omitempty"`
	Charset               string          `json:"charset,omitempty"`
	Collation             string          `json:"collation,omitempty"`
	TablePrefix           string          `json:"table_prefix,omitempty"`
	SlowThreshold         *model.Duration `json:"slow_threshold,omitempty"`
}

func openMysqlConn(ctx context.Context, slowThreshold time.Duration, options *MySQLOptions, autoCreateSchema bool) (*gorm.DB, error) {
	logger := logs.GetContextLogger(ctx)
	db, err := gorm.Open(
		mysqldriver.New(mysqldriver.Config{
			DSN: fmt.Sprintf(
				"%s:%s@tcp(%s)/%s?parseTime=1&multiStatements=1&charset=%s&collation=%s",
				options.Username,
				options.Password,
				options.Host,
				options.Schema,
				options.Charset,
				options.Collation,
			),
		}), &gorm.Config{
			NamingStrategy: schema.NamingStrategy{
				TablePrefix:   options.TablePrefix,
				SingularTable: true,
			},
			Logger: NewLogAdapter(logger, slowThreshold, nil),
		},
	)

	if err != nil && autoCreateSchema {
		if mysqlErr, ok := err.(*mysql.MySQLError); ok {
			if mysqlErr.Number == 1049 && autoCreateSchema {
				level.Info(logger).Log("msg", fmt.Sprintf("auto create schema: %s", options.Schema))
				tmpOpts := *options
				tmpOpts.Schema = "mysql"
				db, err = openMysqlConn(ctx, slowThreshold, &tmpOpts, false)
				if err != nil {
					return nil, err
				}
				err = db.Exec(fmt.Sprintf("CREATE SCHEMA `%s` DEFAULT CHARACTER SET %s COLLATE %s", options.Schema, options.Charset, options.Collation)).Error
				if err != nil {
					return nil, err
				}
				if sqlDB, err := db.DB(); err == nil {
					defer sqlDB.Close()
				}

				return openMysqlConn(ctx, slowThreshold, options, false)
			}
		}
	}
	return db, err
}

func NewMySQLClient(ctx context.Context, options MySQLOptions) (clt *Client, err error) {
	clt = new(Client)
	logger := logs.GetContextLogger(ctx)
	if options.SlowThreshold != nil {
		clt.slowThreshold = time.Duration(*options.SlowThreshold)
		if err != nil {
			level.Error(logger).Log("msg", fmt.Errorf("failed to connect to mysql server: [%s@%s]", options.Username, options.Host), "err", fmt.Errorf("`slow_threshold` option is invalid: %s", err))
			return nil, err
		}
	}
	clt.name = fmt.Sprintf("[MySQL]%s", options.Schema)
	level.Debug(logger).Log("msg", "connect to mysql server",
		"host", options.Host, "username", options.Username,
		"schema", options.Schema,
		"charset", options.Charset,
		"collation", options.Collation)

	db, err := openMysqlConn(ctx, clt.slowThreshold, &options, true)
	if err != nil {
		level.Error(logger).Log("msg", fmt.Errorf("failed to connect to mysql server: [%s@%s]", options.Username, options.Host), "err", err)
		return nil, err
	}

	{
		sqlDB, err := db.DB()
		if err != nil {
			level.Error(logger).Log("msg", fmt.Errorf("failed to connect to mysql server: [%s@%s]", options.Username, options.Host), "err", err)
			return nil, err
		}
		sqlDB.SetMaxIdleConns(int(options.MaxIdleConnections))
		sqlDB.SetConnMaxLifetime(options.GetStdMaxConnectionLifeTime())
		sqlDB.SetMaxOpenConns(int(options.MaxOpenConnections))
	}

	stopCh := signals.SetupSignalHandler(logger)
	stopCh.Add(1)
	go func() {
		<-stopCh.Channel()
		stopCh.WaitRequest()
		if sqlDB, err := db.DB(); err == nil {
			if err = sqlDB.Close(); err != nil {
				level.Warn(logger).Log("msg", fmt.Errorf("failed to close mysql connect: [%s@%s]", options.Username, options.Host), "err", err)
			}
		}
		level.Debug(logger).Log("msg", "MySQL connect closed")
		stopCh.Done()
	}()

	level.Debug(logger).Log("msg", "connected to mysql server",
		"host", options.Host, "username", options.Username,
		"schema", options.Schema,
		"charset", options.Charset,
		"collation", options.Collation)
	clt.database = &Database{
		DB: db,
	}
	return clt, nil
}

func (x *MySQLOptions) GetStdMaxConnectionLifeTime() time.Duration {
	if x != nil && x.MaxConnectionLifeTime != nil {
		return time.Duration(*x.MaxConnectionLifeTime)
	}
	return time.Second * 30
}

func NewMySQLOptions() *MySQLOptions {
	return &MySQLOptions{
		Charset:               "utf8",
		Collation:             "utf8_general_ci",
		MaxIdleConnections:    2,
		MaxOpenConnections:    100,
		MaxConnectionLifeTime: (*model.Duration)(w.P(30 * time.Second)),
		TablePrefix:           "t_",
		Host:                  "localhost",
		Schema:                "idas",
		Username:              "idas",
	}
}

type MySQLClient struct {
	*Client
	options *MySQLOptions
}

func (c MySQLClient) Options() MySQLOptions {
	return *c.options
}

func (c *MySQLClient) SetOptions(o *MySQLOptions) {
	c.options = o
}

func (c MySQLClient) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.options)
}

func (c *MySQLClient) UnmarshalJSON(data []byte) (err error) {
	if c.options == nil {
		c.options = NewMySQLOptions()
	}
	if err = json.Unmarshal(data, c.options); err != nil {
		return err
	}
	if c.Client, err = NewMySQLClient(context.Background(), *c.options); err != nil {
		return err
	}
	return
}
