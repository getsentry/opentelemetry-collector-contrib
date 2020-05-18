// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sentryexporter

import (
	"time"

	"github.com/getsentry/sentry-go"
)

// Tags describes a Sentry Tag.
type Tags map[string]string

// SentrySpan describes a Span following the Sentry format.
type SentrySpan struct {
	TraceID        string    `json:"trace_id"`
	SpanID         string    `json:"span_id"`
	ParentSpanID   string    `json:"parent_span_id,omitempty"`
	Description    string    `json:"description,omitempty"`
	Op             string    `json:"op,omitempty"`
	Tags           Tags      `json:"tags,omitempty"`
	StartTimestamp time.Time `json:"start_timestamp,omitempty"`
	EndTimestamp   time.Time `json:"timestamp"`
	Status         string    `json:"status"`
}

// IsRootSpan determines if a span is a root span.
// If parent span id is empty, then the span is a root span.
// See: https://github.com/open-telemetry/opentelemetry-proto/blob/master/opentelemetry/proto/trace/v1/trace.proto#L82
func (s SentrySpan) IsRootSpan() bool {
	return s.ParentSpanID == ""
}

// TraceContext describes the context of the trace.
type TraceContext struct {
	TraceID     string `json:"trace_id"`
	SpanID      string `json:"span_id"`
	Op          string `json:"op,omitempty"`
	Description string `json:"description,omitempty"`
}

// SentryTransaction describes a Sentry Transaction.
// TODO: generate extra fields when creating envelope EventID, Type, User, Platform, SDK
type SentryTransaction struct {
	*sentry.Event
	StartTimestamp time.Time     `json:"start_timestamp,omitempty"`
	TraceContext   TraceContext  `json:"contexts,omitempty"`
	Spans          []*SentrySpan `json:"spans,omitempty"`
}
