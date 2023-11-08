/*
 * Copyright © 2023 MicroOps-cn.
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

package tracing

import (
	"context"
	"os"

	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type FileTracingOptions struct {
	Path string `json:"path" yaml:"path" mapstructure:"path"`
}

type FileTracing struct {
	*stdouttrace.Exporter
	f *os.File
}

func (t *FileTracing) Shutdown(ctx context.Context) error {
	if err := t.f.Sync(); err != nil {
		return err
	}
	return t.f.Close()
}

func NewFileTraceExporter(_ context.Context, o *FileTracingOptions) (sdktrace.SpanExporter, error) {
	f, err := os.Create(o.Path)
	if err != nil {
		return nil, err
	}
	exporter, err := stdouttrace.New(
		stdouttrace.WithWriter(f),
		stdouttrace.WithPrettyPrint(),
	)
	if err != nil {
		return nil, err
	}

	return &FileTracing{
		Exporter: exporter,
		f:        f,
	}, nil
}
