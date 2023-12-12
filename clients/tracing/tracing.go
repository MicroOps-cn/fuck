/*
 Copyright © 2023 MicroOps-cn.

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

package tracing

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"

	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	uuid "github.com/satori/go.uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/MicroOps-cn/fuck/log"
	"github.com/MicroOps-cn/fuck/signals"
)

type RetryOptions struct {
	Enabled         bool
	InitialInterval time.Duration
	MaxInterval     time.Duration
	MaxElapsedTime  time.Duration
}

func (c *RetryOptions) UnmarshalJSON(data []byte) (err error) {
	type plain RetryOptions
	*c = RetryOptions{
		Enabled:         true,
		InitialInterval: 5 * time.Second,
		MaxInterval:     30 * time.Second,
		MaxElapsedTime:  time.Minute,
	}
	if string(data) == "false" || string(data) == `"false"` {
		c.Enabled = false
		return
	}
	return json.Unmarshal(data, (*plain)(c))
}

var DefaultRetryConfig = RetryOptions{
	Enabled:         true,
	InitialInterval: 5 * time.Second,
	MaxInterval:     30 * time.Second,
	MaxElapsedTime:  time.Minute,
}

type Compression int

func (c *Compression) UnmarshalJSON(data []byte) (err error) {
	switch string(data) {
	case `"true"`, `true`, `1`, `"1"`, `"gzip"`:
		*c = Compression(otlptracehttp.GzipCompression)
	case `"false"`, `false`, `0`, `"0"`, ``:
		*c = Compression(otlptracehttp.NoCompression)
	}
	return fmt.Errorf("the value can only be one of true, false, or gzip")
}

type TraceOptions struct {
	HTTP        *HTTPClientOptions   `json:"http" yaml:"http" mapstructure:"http"`
	GRPC        *GRPCClientOptions   `json:"grpc" yaml:"grpc" mapstructure:"grpc"`
	Jaeger      *JaegerClientOptions `json:"jaeger" yaml:"jaeger" mapstructure:"jaeger"`
	Zipkin      *ZipkinClientOptions `json:"zipkin" yaml:"zipkin" mapstructure:"zipkin"`
	File        *FileTracingOptions  `json:"file" yaml:"file" mapstructure:"file"`
	ServiceName string               `json:"service_name" yaml:"service_name" mapstructure:"service_name"`
}

func (o *TraceOptions) UnmarshalJSON(data []byte) (err error) {
	type plain TraceOptions
	return json.Unmarshal(data, (*plain)(o))
}

// String implement proto.Message
func (o TraceOptions) String() string {
	return proto.CompactTextString(&o)
}

// ProtoMessage implement proto.Message
func (o *TraceOptions) ProtoMessage() {}

// Reset *implement proto.Message*
func (o *TraceOptions) Reset() {
	*o = TraceOptions{}
}

func (o *TraceOptions) MarshalJSONPB(marshaller *jsonpb.Marshaler) ([]byte, error) {
	buf := &bytes.Buffer{}
	buf.WriteString(marshaller.Indent + "{")
	enc := json.NewEncoder(buf)
	enc.SetIndent(marshaller.Indent, marshaller.Indent)
	var err error
	if o.HTTP != nil {
		buf.WriteString(marshaller.Indent + marshaller.Indent + `"http":`)
		err = enc.Encode(o.HTTP)
		buf.WriteRune('\n')
	} else if o.GRPC != nil {
		buf.WriteString(marshaller.Indent + marshaller.Indent + `"grpc":`)
		err = enc.Encode(o.GRPC)
		buf.WriteRune('\n')
	} else if o.Jaeger != nil {
		buf.WriteString(marshaller.Indent + marshaller.Indent + `"jaeger":`)
		err = enc.Encode(o.Jaeger)
		buf.WriteRune('\n')
	} else if o.Zipkin != nil {
		buf.WriteString(marshaller.Indent + marshaller.Indent + `"zipkin":`)
		err = enc.Encode(o.Zipkin)
		buf.WriteRune('\n')
	} else if o.File != nil {
		buf.WriteString(marshaller.Indent + marshaller.Indent + `"file":`)
		err = enc.Encode(o.File)
		buf.WriteRune('\n')
	}
	buf.WriteString(marshaller.Indent + "}")
	return buf.Bytes(), err
}

type idGenerator struct{}

func (i idGenerator) NewIDs(ctx context.Context) (trace.TraceID, trace.SpanID) {
	tid, err := uuid.FromString(log.GetTraceId(ctx))
	if err != nil {
		tid = uuid.Must(uuid.NewV4())
	}

	sid := trace.SpanID{}
	_, _ = rand.Read(sid[:])
	return trace.TraceID(tid), sid
}

func (i idGenerator) NewSpanID(ctx context.Context, traceID trace.TraceID) trace.SpanID {
	sid := trace.SpanID{}
	_, _ = rand.Read(sid[:])
	return sid
}

var DefaultOptions *TraceOptions

func SetTraceOptions(o *TraceOptions) {
	DefaultOptions = o
}

func NewTraceProvider(ctx context.Context, o *TraceOptions) (p *sdktrace.TracerProvider, err error) {
	var logger kitlog.Logger
	ctx, logger = log.NewContextLogger(ctx)
	var exp sdktrace.SpanExporter
	if o.HTTP != nil {
		exp, err = NewHTTPTraceExporter(ctx, o.HTTP)
	} else if o.GRPC != nil {
		exp, err = NewGRPCTraceExporter(ctx, o.GRPC)
	} else if o.Jaeger != nil {
		exp, err = NewJaegerTraceExporter(ctx, o.Jaeger)
	} else if o.Zipkin != nil {
		exp, err = NewZipkinTraceExporter(ctx, o.Zipkin)
	} else if o.File != nil {
		exp, err = NewFileTraceExporter(ctx, o.File)
	} else {
		exp, err = NewNopTraceExporter(ctx)
	}
	if err != nil {
		return nil, err
	}
	attrs := []attribute.KeyValue{
		semconv.HostArchKey.String(runtime.GOARCH),
	}
	if len(o.ServiceName) != 0 {
		attrs = append(attrs, semconv.ServiceName(o.ServiceName))
	}
	if ns := os.Getenv("OTEL_SERVICE_NAMESPACE"); len(ns) != 0 {
		attrs = append(attrs, semconv.ServiceNamespace(ns))
	}

	r, err := resource.New(ctx,
		resource.WithContainer(),
		resource.WithOS(),
		resource.WithHost(),
		resource.WithProcess(),
		resource.WithAttributes(attrs...),
	)
	if err != nil {
		return nil, err
	}

	r, err = resource.Merge(
		resource.Default(),
		r,
	)
	if err != nil {
		return nil, err
	}

	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		level.Error(logger).Log("msg", "trace exception", "err", err)
	}))
	tracer := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(r),
		sdktrace.WithIDGenerator(&idGenerator{}),
	)
	stopCh := signals.SignalHandler()
	stopCh.PreStop(signals.LevelTrace, func() {
		if tracer != nil {
			timeoutCtx, closeCh := context.WithTimeout(context.Background(), time.Second*5)
			defer closeCh()
			if err = tracer.ForceFlush(timeoutCtx); err != nil {
				level.Error(logger).Log("msg", "failed to force flush trace", "err", err)
			}
			if err = tracer.Shutdown(timeoutCtx); err != nil {
				level.Error(logger).Log("msg", "failed to close trace", "err", err)
			}
		}
	})
	return tracer, nil
}
