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
	"time"

	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"

	"github.com/MicroOps-cn/fuck/log"
)

type logContext struct {
	logger        kitlog.Logger
	slowThreshold time.Duration
	tracer        trace.Tracer
}

func (l *logContext) LogMode(lvl logger.LogLevel) logger.Interface {
	var filter kitlog.Logger
	switch lvl {
	case logger.Silent:
		filter = level.NewFilter(l.logger, level.AllowNone())
	case logger.Info:
		filter = level.NewFilter(l.logger, level.AllowInfo())
	case logger.Warn:
		filter = level.NewFilter(l.logger, level.AllowWarn())
	case logger.Error:
		filter = level.NewFilter(l.logger, level.AllowError())
	default:
		filter = l.logger
	}

	return NewLogAdapter(filter, l.slowThreshold, l.tracer)
}

func (l logContext) Info(_ context.Context, msg string, data ...interface{}) {
	level.Info(l.logger).Log(log.CallerName, utils.FileWithLineNum(), "msg", fmt.Sprintf(msg, data...))
}

func (l logContext) Warn(_ context.Context, msg string, data ...interface{}) {
	level.Warn(l.logger).Log(log.CallerName, utils.FileWithLineNum(), "msg", fmt.Sprintf(msg, data...))
}

func (l logContext) Error(_ context.Context, msg string, data ...interface{}) {
	level.Error(l.logger).Log(log.CallerName, utils.FileWithLineNum(), "msg", fmt.Sprintf(msg, data...))
}

func (l logContext) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	var span trace.Span
	if l.tracer != nil {
		_, span = l.tracer.Start(ctx, "ExecuteSQL", trace.WithTimestamp(begin), trace.WithSpanKind(trace.SpanKindClient))
	} else {
		_, span = otel.GetTracerProvider().Tracer(instrumentationName).Start(ctx, "ExecuteSQL", trace.WithTimestamp(begin), trace.WithSpanKind(trace.SpanKindClient))
	}
	defer span.End()
	sql, rows := fc()
	span.SetAttributes(attribute.String("db.statement", sql), attribute.Int64("db.row_return_count", rows))
	switch {
	case err != nil && err != gorm.ErrRecordNotFound:
		span.SetStatus(codes.Error, err.Error())
		level.Error(l.logger).Log(log.CallerName, utils.FileWithLineNum(), "msg", "SQL execution exception", log.WrapKeyName("errorMsg"), err, log.WrapKeyName("sql"), sql, log.WrapKeyName("execTime"), float64(elapsed.Nanoseconds())/1e6, log.WrapKeyName("rowReturnCount"), rows)
	case elapsed > l.slowThreshold && l.slowThreshold != 0:
		span.SetStatus(codes.Ok, "")
		level.Warn(l.logger).Log(log.CallerName, utils.FileWithLineNum(), "msg", "exec SQL query", log.WrapKeyName("sql"), sql, log.WrapKeyName("execTime"), float64(elapsed.Nanoseconds())/1e6, log.WrapKeyName("rowReturnCount"), rows)
	default:
		span.SetStatus(codes.Ok, "")
		level.Debug(l.logger).Log(log.CallerName, utils.FileWithLineNum(), "msg", "exec SQL query", log.WrapKeyName("sql"), sql, log.WrapKeyName("execTime"), float64(elapsed.Nanoseconds())/1e6, log.WrapKeyName("rowReturnCount"), rows)
	}
}

func NewLogAdapter(l kitlog.Logger, slowThreshold time.Duration, tracer trace.Tracer) logger.Interface {
	return &logContext{logger: l, slowThreshold: slowThreshold, tracer: tracer}
}

var _ logger.Interface = new(logContext)
