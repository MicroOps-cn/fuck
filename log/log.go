/*
 * Copyright © 2022 MicroOps-cn.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package log

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"

	"github.com/MicroOps-cn/fuck/capacity"
	g "github.com/MicroOps-cn/fuck/generator"
	w "github.com/MicroOps-cn/fuck/wrapper"
)

// TimestampFormat This timestamp format differs from RFC3339Nano by using .000 instead
// of .999999999 which changes the timestamp from 9 variable to 3 fixed
// decimals (.130 instead of .130987456).
var TimestampFormat = log.TimestampFormat(
	func() time.Time { return time.Now().UTC() },
	"2006-01-02T15:04:05.000Z07:00",
)

type callerName struct{}

func (callerName) String() string {
	return "caller"
}

var CallerName = callerName{}

const (
	TraceIdName = "traceId"
)

var DefaultCaller = Caller(5)

var sourceDir = GetSourceCodeDir("log/log.go")

func GetSourceCodeDir(codeRelativePath string, depth ...int) string {
	var currentFile string
	if len(depth) == 0 {
		_, currentFile, _, _ = runtime.Caller(1)
	} else {
		_, currentFile, _, _ = runtime.Caller(depth[0])
	}
	return strings.TrimSuffix(currentFile, codeRelativePath)
}

func SetSourceCodeDir(d string) {
	sourceDir = d
}

func Caller(depth int) log.Valuer {
	return func() interface{} {
		_, file, line, _ := runtime.Caller(depth)
		return strings.TrimPrefix(file, sourceDir) + ":" + strconv.Itoa(line)
	}
}

func newWriterFromConfig(c Config) io.Writer {
	var writer io.Writer
	if len(c.FilePath) != 0 {
		switch c.FilePath {
		case "/dev/stdin":
			writer = os.Stdin
		case "/dev/stdout":
			writer = os.Stdout
		case "/dev/stderr":
			writer = os.Stderr
		default:
			stat, err := os.Stat(c.FilePath)
			if err == nil && stat.Mode()&os.ModeType != 0 {
				_, _ = fmt.Fprintf(os.Stderr, "unknown file mode: %s", stat.Mode())
				//nolint:gofumpt
				if writer, err = os.OpenFile(c.FilePath, os.O_APPEND|os.O_RDWR, 0600); err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "[WARN]failed to open file %s, use /dev/stdout (console stdout)", err)
					return os.Stdout
				}
			} else if (err != nil && os.IsNotExist(err)) || err == nil {
				fileRotationSize := capacity.Capacities(0)
				if c.FileRotationSize == "" || c.FileRotationSize == "0" {
					fileRotationSize, err = capacity.ParseCapacities(c.FileRotationSize)
					if err != nil {
						_, _ = fmt.Fprintf(os.Stderr, "[WARN] Failed to parse parameter file-rotation-size, do not rotate logs by size: %s", err)
						fileRotationSize = 0
					}
				}
				if writer, err = rotatelogs.New(
					c.FilePath+"-%Y%m%d%H%M",
					rotatelogs.WithLinkName(c.FilePath),
					rotatelogs.WithMaxAge(c.FileMaxAge),
					rotatelogs.WithRotationSize(int64(fileRotationSize)),
					rotatelogs.WithRotationTime(c.FileRotationTime),
				); err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "[WARN]failed to create log rotate %s, use /dev/stdout (console stdout)", err)
					return os.Stdout
				}
			} else {
				_, _ = fmt.Fprintf(os.Stderr, "unknown err: %s, ", err)
				return os.Stdout
			}
		}
	} else {
		writer = os.Stdout
	}
	return writer
}

// New returns a new leveled oklog logger. Each logged line will be annotated
// with a timestamp. The output always goes to stderr.
func New(opts ...NewLoggerOption) log.Logger {
	o := newLoggerOptions{}
	for _, opt := range opts {
		opt(&o)
	}
	config := o.config
	var l log.Logger
	if config == nil {
		config = DefaultLoggerConfig
	}
	if config.Format == nil {
		config.Format = w.P[AllowedFormat](FormatLogfmt)
	}
	if config.Level == nil {
		config.Level = w.P[AllowedLevel](LevelInfo)
	}
	if o.w == nil {
		o.w = log.NewSyncWriter(newWriterFromConfig(*config))
	}

	if val, ok := registeredLogFormat.Load(*config.Format); ok {
		if createFunc, ok := val.(LoggerCreateFunc); ok {
			l = createFunc(o.w)
		} else {
			l = log.NewLogfmtLogger(o.w)
			_ = level.Warn(l).Log("msg", fmt.Errorf("log output format %s is not registered, using logfmt", *config.Format))
		}
	} else {
		l = log.NewLogfmtLogger(o.w)
		_ = level.Warn(l).Log("msg", fmt.Errorf("log output format %s is not registered, using logfmt", *config.Format))
	}
	l = level.NewFilter(log.With(&extLogger{logger: l, w: o.w}, "ts", TimestampFormat, CallerName, DefaultCaller), config.Level.getOption())
	return l
}

var (
	rootLoggerOnce sync.Once
	rootLogger     log.Logger
)

// SetDefaultLogger
//
//	@Description[en-US]: Set the default Logger
//	@Description[zh-CN]: 设置默认的Logger
//	@param logger 	log.Logger
func SetDefaultLogger(logger log.Logger) {
	rootLogger = logger
}

// GetDefaultLogger
//
//	@Description[en-US]: Get the default Logger
//	@Description[zh-CN]: 获取默认的Logger
//	@return log.Logger
func GetDefaultLogger() log.Logger {
	rootLoggerOnce.Do(func() {
		if rootLogger == nil {
			rootLogger = New()
		}
	})
	return rootLogger
}

type NewLoggerOption func(*newLoggerOptions)

// WithTraceId
//
//	@Description[en-US]: Specify TraceId when creating Logger
//	@Description[zh-CN]: 创建Logger时指定TraceId
//	@param traceId 	string
//	@return NewLoggerOption
func WithTraceId(traceId string) NewLoggerOption {
	return func(o *newLoggerOptions) {
		o.traceId = traceId
	}
}

// WithLogger
//
//	@Description[en-US]: Specify parent Logger when creating Logger
//	@Description[zh-CN]: 创建Logger时指定父Logger
//	@param l 	log.Logger
//	@return NewLoggerOption
func WithLogger(l log.Logger) NewLoggerOption {
	return func(o *newLoggerOptions) {
		o.l = l
	}
}

// WithWriter
//
//	@Description[en-US]: Specify parent Logger when creating Logger
//	@Description[zh-CN]: 创建Logger时指定父Logger
//	@param l 	log.Logger
//	@return NewLoggerOption
func WithWriter(w io.Writer) NewLoggerOption {
	return func(o *newLoggerOptions) {
		o.w = w
	}
}

func WithConfig(c *Config) NewLoggerOption {
	return func(o *newLoggerOptions) {
		o.config = c
	}
}

func WithKeyValues(keyvals ...interface{}) NewLoggerOption {
	return func(o *newLoggerOptions) {
		o.kvs = keyvals
	}
}

type newLoggerOptions struct {
	traceId string
	l       log.Logger
	w       io.Writer
	kvs     []interface{}
	config  *Config
}

// NewTraceLogger
//
//	@Description[en-US]: Create a new traceable log object
//	@Description[zh-CN]: 创建新的可跟踪的日志对象
//	@param options 	...NewLoggerOption
//	@return log.Logger
func NewTraceLogger(options ...NewLoggerOption) log.Logger {
	o := newLoggerOptions{traceId: NewTraceId(), l: rootLogger}
	for _, f := range options {
		f(&o)
	}
	if o.l == nil {
		return log.With(GetDefaultLogger(), TraceIdName, o.traceId)
	}
	if len(o.kvs) != 0 {
		o.l = log.With(o.l, o.kvs...)
		o.kvs = nil
	}
	return log.With(o.l, TraceIdName, o.traceId)
}

// NewTraceId
//
//	@Description[en-US]: Generate a trace ID
//	@Description[zh-CN]: 生成一个跟踪ID
//	@return string
func NewTraceId() string {
	return g.NewId("logging")
}

// GetTraceId
//
//	@Description[en-US]: Get traceId in Context.
//	@Description[zh-CN]: 获取Context中的traceId
//	@param ctx 	context.Context
//	@return string
func GetTraceId(ctx context.Context) string {
	s, _ := ctx.Value(contextTraceId{}).(string)
	if len(s) == 0 {
		return NewTraceId()
	}
	return s
}

type contextTraceId struct{}

// NewContextLogger
//
//	@Description[en-US]: Create a traceable context log object
//	@Description[zh-CN]: 创建可跟踪的上下文日志对象
//	@param parent 	context.Context
//	@param opts 	...NewLoggerOption
//	@return ${ret_name}	context.Context
//	@return ${ret_name}	log.Logger
func NewContextLogger(parent context.Context, opts ...NewLoggerOption) (context.Context, log.Logger) {
	o := newLoggerOptions{}
	for _, opt := range opts {
		opt(&o)
	}
	if len(o.traceId) == 0 {
		o.traceId = NewTraceId()
	}
	if o.l == nil {
		o.l = NewTraceLogger(WithTraceId(o.traceId))
	}
	if len(o.kvs) != 0 {
		o.l = log.With(o.l, o.kvs...)
		o.kvs = nil
	}
	return context.WithValue(context.WithValue(parent, contextTraceId{}, o.traceId), contextLogger{}, o.l), o.l
}

type contextLogger struct{}

// GetContextLogger
//
//	@Description[en-US]: Get the log.Logger object in Context. If it does not exist, a new Logger with traceId will be returned
//	@Description[zh-CN]: 获取Context中的log.Logger对象，如果不存在，则返回一个新的带traceId的Logger
//	@param ctx 	context.Context
//	@param options 	...Option
//	@return log.Logger
func GetContextLogger(ctx context.Context, options ...Option) log.Logger {
	l, ok := ctx.Value(contextLogger{}).(log.Logger)
	if !ok {
		l = NewTraceLogger()
	}
	for _, option := range options {
		l = option(l)
	}
	return l
}

type Option func(l log.Logger) log.Logger

// WithCaller
//
//	@Description[en-US]: Get the file name path of the specified location of the call chain.
//	@Description[zh-CN]: 获取调用链的指定位置的文件名路径
//	@param layer 	int
//	@return Option
func WithCaller(layer int) Option {
	return func(l log.Logger) log.Logger {
		return log.With(l, CallerName, Caller(layer))
	}
}

// WithMethod
//
//	@Description[en-US]: Get the method name of the specified location of the call chain (if not specified, the default is 2).
//	@Description[zh-CN]: 获取调用链的指定位置（如果不指定，默认为2）的方法名称
//	@param skip 	...int
//	@return Option
func WithMethod(skip ...int) Option {
	pc := make([]uintptr, 1)
	if len(skip) > 0 {
		runtime.Callers(skip[0], pc)
	} else {
		runtime.Callers(2, pc)
	}
	funcName := strings.SplitAfterN(runtime.FuncForPC(pc[0]).Name(), ".", 2)
	return func(l log.Logger) log.Logger {
		return log.With(l, "method", funcName[len(funcName)-1])
	}
}

type KeyName string

func (n KeyName) String() string {
	return string(n)
}

func WrapKeyName(name string) fmt.Stringer {
	return KeyName(name)
}

type nopLogger struct{}

func (n *nopLogger) Log(keyvals ...interface{}) error {
	return nil
}

func NewNopLogger() log.Logger {
	return &nopLogger{}
}
