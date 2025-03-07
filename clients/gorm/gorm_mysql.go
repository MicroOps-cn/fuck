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
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/log/level"
	"github.com/go-sql-driver/mysql"
	"github.com/prometheus/common/model"
	mysqldriver "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"github.com/MicroOps-cn/fuck/clients/tls"
	g "github.com/MicroOps-cn/fuck/generator"
	logs "github.com/MicroOps-cn/fuck/log"
	"github.com/MicroOps-cn/fuck/safe"
	"github.com/MicroOps-cn/fuck/signals"
	w "github.com/MicroOps-cn/fuck/wrapper"
)

type TLSOptions struct {
	options *tls.TLSOptions
	name    string
}

func (o TLSOptions) MarshalJSON() ([]byte, error) {
	if o.options == nil {
		return []byte(o.name), nil
	}
	return json.Marshal(o.options)
}

func (o *TLSOptions) UnmarshalJSON(data []byte) (err error) {
	if err = json.Unmarshal(data, &o.name); err == nil {
		return nil
	}
	if o.options == nil {
		o.options = &tls.TLSOptions{}
	}
	if err = json.Unmarshal(data, o.options); err != nil {
		return err
	}
	o.name = g.NewId()
	tlsConfig, err := tls.NewTLSConfig(o.options)
	if err != nil {
		return err
	}
	return mysql.RegisterTLSConfig(o.name, tlsConfig)
}

func (o *TLSOptions) Equal(o2 *TLSOptions) bool {
	if (o == nil && o2 == nil) || (o.options == nil && o2.options == nil) {
		return true
	}
	return o != nil && o2 != nil && reflect.DeepEqual(o.options, o2.options)
}

type MySQLOptions struct {
	Host                  string          `json:"host,omitempty"`
	Username              string          `json:"username,omitempty"`
	Password              safe.String     `json:"password,omitempty"`
	Schema                string          `json:"schema,omitempty"`
	MaxIdleConnections    int32           `json:"max_idle_connections,omitempty" yaml:"max_idle_connections" mapstructure:"max_idle_connections"`
	MaxOpenConnections    int32           `json:"max_open_connections,omitempty" yaml:"max_open_connections" mapstructure:"max_open_connections"`
	MaxConnectionLifeTime *model.Duration `json:"max_connection_life_time,omitempty" yaml:"max_connection_life_time" mapstructure:"max_connection_life_time"`
	Charset               string          `json:"charset,omitempty"`
	Collation             string          `json:"collation,omitempty"`
	TablePrefix           string          `json:"table_prefix,omitempty" yaml:"table_prefix" mapstructure:"table_prefix"`
	SlowThreshold         *model.Duration `json:"slow_threshold,omitempty" yaml:"slow_threshold" mapstructure:"slow_threshold"`
	TLSConfig             *TLSOptions     `json:"tls_config" yaml:"tls_config" mapstructure:"tls_config"`
}

func durationEqual(dur1, dur2 *model.Duration) bool {
	if dur1 == nil && dur2 == nil {
		return true
	}
	if dur1 != nil && dur2 != nil && dur1.String() == dur2.String() {
		return true
	}
	return false
}

func (x *MySQLOptions) Equal(options MySQLOptions) bool {
	return !(x.Host != options.Host ||
		x.Username != options.Username ||
		!x.Password.Equal(options.Password) ||
		x.Schema != options.Schema ||
		x.MaxIdleConnections != options.MaxIdleConnections ||
		x.MaxOpenConnections != options.MaxOpenConnections ||
		!durationEqual(x.MaxConnectionLifeTime, options.MaxConnectionLifeTime) ||
		x.Charset != options.Charset ||
		x.Collation != options.Collation ||
		x.TablePrefix != options.TablePrefix ||
		!durationEqual(x.SlowThreshold, options.SlowThreshold) ||
		!x.TLSConfig.Equal(x.TLSConfig))
}

func (x *MySQLOptions) String() string {
	return fmt.Sprintf("%s://%s@%s/%s", x.GetType(), x.Username, x.Host, x.Schema)
}

func (x *MySQLOptions) GetPeer() (string, int) {
	host, port, found := strings.Cut(x.Host, ":")
	if found && len(port) > 0 {
		portNum, err := strconv.Atoi(port)
		if err == nil {
			return host, portNum
		}
	}
	return x.Host, 3306
}

func (x *MySQLOptions) GetConnectionString() string {
	return fmt.Sprintf("mysql://%s", x.Host)
}

func (x *MySQLOptions) GetDBName() string {
	return x.Schema
}

func (x *MySQLOptions) GetUsername() string {
	return x.Username
}

func (x *MySQLOptions) GetType() string {
	return "mysql"
}

func openMysqlConn(ctx context.Context, slowThreshold time.Duration, options *MySQLOptions, autoCreateSchema bool) (*gorm.DB, error) {
	logger := logs.GetContextLogger(ctx)
	passwd, err := options.Password.UnsafeString()
	if err != nil {
		return nil, err
	}
	if options.Charset == "" {
		options.Charset = "utf8mb4"
	}
	if options.Collation == "" {
		options.Collation = "utf8mb4_general_ci"
	}
	if options.MaxOpenConnections == 0 {
		options.MaxOpenConnections = 100
	}

	var tlsConfigName string
	if options.TLSConfig != nil {
		tlsConfigName = options.TLSConfig.name
	}
	db, err := gorm.Open(
		mysqldriver.New(mysqldriver.Config{
			DSNConfig: &mysql.Config{
				User:                 options.Username,
				Passwd:               passwd,
				Net:                  "tcp",
				Addr:                 options.Host,
				DBName:               options.Schema,
				Params:               map[string]string{"charset": options.Charset},
				Collation:            options.Collation,
				AllowNativePasswords: true,
				CheckConnLiveness:    true,
				ParseTime:            true,
				TLSConfig:            tlsConfigName,
			},
	cfg := &mysql.Config{
		User:                 options.Username,
		Passwd:               passwd,
		Net:                  "tcp",
		Addr:                 options.Host,
		DBName:               options.Schema,
		Params:               map[string]string{"charset": options.Charset},
		Collation:            options.Collation,
		AllowNativePasswords: true,
		CheckConnLiveness:    true,
		ParseTime:            true,
		TLSConfig:            tlsConfigName,
	}
	db, err := gorm.Open(
		mysqldriver.New(mysqldriver.Config{
			DSNConfig: cfg,
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

func NewMySQLClient(ctx context.Context, name string, options MySQLOptions) (clt *Client, err error) {
	clt = new(Client)
	clt.options = &options
	logger := logs.GetContextLogger(ctx)
	if options.SlowThreshold != nil {
		clt.slowThreshold = time.Duration(*options.SlowThreshold)
	}
	clt.name = name
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
	stopCh.PreStop(signals.LevelDB, func() {
		if sqlDB, err := db.DB(); err == nil {
			if err = sqlDB.Close(); err != nil {
				level.Warn(logger).Log("msg", fmt.Errorf("failed to close mysql connect: [%s@%s]", options.Username, options.Host), "err", err)
			}
		} else {
			level.Warn(logger).Log("msg", fmt.Errorf("failed to close mysql connect: [%s@%s]", options.Username, options.Host), "err", err)
		}
		level.Debug(logger).Log("msg", "MySQL connect closed")
	})
	level.Info(logger).Log("msg", "connected to mysql server",
		"host", options.Host, "username", options.Username,
		"schema", options.Schema,
		"charset", options.Charset,
		"collation", options.Collation)
	clt.database = db
	clt.statsCollector = collector.Register(clt)
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
	if c.Client, err = NewMySQLClient(context.Background(), "", *c.options); err != nil {
		return err
	}
	return
}
