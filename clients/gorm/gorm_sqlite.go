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
	"database/sql/driver"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/MicroOps-cn/fuck/log"
	gosqlite "github.com/glebarez/go-sqlite"
	"github.com/glebarez/sqlite"
	"github.com/go-kit/log/level"
	"github.com/gogo/protobuf/types"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"github.com/MicroOps-cn/fuck/signals"
)

type SQLiteOptions struct {
	Path                 string          `protobuf:"bytes,1,opt,name=path,proto3" json:"path,omitempty"`
	TablePrefix          string          `protobuf:"bytes,2,opt,name=table_prefix,json=tablePrefix,proto3" json:"table_prefix,omitempty"`
	SlowThreshold        *types.Duration `protobuf:"bytes,12,opt,name=slow_threshold,json=slowThreshold,proto3" json:"slow_threshold,omitempty"`
	XXX_NoUnkeyedLiteral struct{}        `json:"-"`
	XXX_unrecognized     []byte          `json:"-"`
	XXX_sizecache        int32           `json:"-"`
}

func init() {
	gosqlite.MustRegisterDeterministicScalarFunction("from_base64", 1, func(ctx *gosqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
		switch argTyped := args[0].(type) {
		case string:
			return base64.StdEncoding.DecodeString(argTyped)
		default:
			return nil, fmt.Errorf("unsupported type: %T", args[0])
		}
	})
}

func NewSQLiteClient(ctx context.Context, options *SQLiteOptions) (clt *SQLiteClient, err error) {
	client, err := NewGormSQLiteClient(ctx, options)
	if err != nil {
		return nil, err
	}
	return &SQLiteClient{Client: client, options: options}, nil
}

func NewGormSQLiteClient(ctx context.Context, options *SQLiteOptions) (clt *Client, err error) {
	clt = new(Client)
	logger := log.GetContextLogger(ctx)
	if options.SlowThreshold != nil {
		clt.slowThreshold, err = types.DurationFromProto(options.SlowThreshold)
		if err != nil {
			level.Error(logger).Log("msg", fmt.Sprintf("failed to connect to SQLite database: %s", options.Path), "err", fmt.Errorf("`slow_threshold` option is invalid: %s", err))
			return nil, err
		}
	}
	clt.name = fmt.Sprintf("[SQLite]%s", filepath.Base(options.Path))

	level.Debug(logger).Log("msg", "connect to sqlite", "dsn", options.Path)
	db, err := gorm.Open(sqlite.Open(options.Path), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "t_",
			SingularTable: true,
		},
		Logger: NewLogAdapter(logger, clt.slowThreshold, nil),
	})
	if err != nil {
		level.Error(logger).Log("msg", fmt.Sprintf("failed to connect to SQLite database: %s", options.Path), "err", err)
		return nil, fmt.Errorf("failed to connect to SQLite database: %s: %s", options.Path, err)
	}
	stopCh := signals.SetupSignalHandler(logger)
	stopCh.Add(1)
	go func() {
		<-stopCh.Channel()
		stopCh.WaitRequest()
		if sqlDB, err := db.DB(); err == nil {
			if err = sqlDB.Close(); err != nil {
				level.Warn(logger).Log("msg", "Failed to close SQLite database", "err", err)
			}
		}
		level.Debug(logger).Log("msg", "Sqlite connect closed")
		stopCh.Done()
	}()
	clt.database = &Database{DB: db}
	return clt, nil
}

func NewSQLiteOptions() *SQLiteOptions {
	return &SQLiteOptions{
		Path:        "idas.db",
		TablePrefix: "t_",
	}
}

type SQLiteClient struct {
	*Client
	options *SQLiteOptions
}

func (c SQLiteClient) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.options)
}

func (c *SQLiteClient) UnmarshalJSON(data []byte) (err error) {
	if c.options == nil {
		c.options = NewSQLiteOptions()
	}
	if err = json.Unmarshal(data, c.options); err != nil {
		return err
	}
	if c.Client, err = NewGormSQLiteClient(context.Background(), c.options); err != nil {
		return err
	}
	return
}
