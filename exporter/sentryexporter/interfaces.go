// Copyright 2020 OpenTelemetry Authors
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
)

// Tags describes a Sentry Tag
type Tags map[string]string

// SentrySpan describes a Span following the Sentry format
type SentrySpan struct {
	TraceID        string    `json:"trace_id"`
	SpanID         string    `json:"span_id"`
	ParentSpanID   string    `json:"parent_span_id,omitempty"`
	Description    string    `json:"description,omitempty"`
	Op             string    `json:"op,omitempty"`
	Tags           Tags      `json:"tags,omitempty"`
	StartTimestamp time.Time `json:"start_timestamp,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
	Status         string    `json:"status"`
}
