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
	"encoding/json"
	"time"

	"github.com/getsentry/sentry-go"
)

// SentryEvent aliases the sentry Event type.
// Needed to Marshal the transactions into JSON properly.
type SentryEvent sentry.Event

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

// MarshalJSON converts the SentrySpan struct to JSON.
func (s SentrySpan) MarshalJSON() ([]byte, error) {
	type alias SentrySpan
	return json.Marshal(&struct {
		StartTimestamp string `json:"start_timestamp,omitempty"`
		EndTimestamp   string `json:"timestamp"`
		*alias
	}{
		StartTimestamp: s.StartTimestamp.UTC().Format(time.RFC3339),
		EndTimestamp:   s.EndTimestamp.UTC().Format(time.RFC3339),
		alias:          (*alias)(&s),
	})
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
	*SentryEvent
	StartTimestamp time.Time     `json:"start_timestamp,omitempty"`
	TraceContext   TraceContext  `json:"contexts,omitempty"`
	Spans          []*SentrySpan `json:"spans,omitempty"`
}

// MarshalJSON converts the SentryTransaction struct to JSON.
func (t SentryTransaction) MarshalJSON() ([]byte, error) {
	type alias SentryTransaction
	return json.Marshal(&struct {
		StartTimestamp string       `json:"start_timestamp,omitempty"`
		Timestamp      string       `json:"timestamp"`
		Type           string       `json:"type"`
		Contexts       TraceContext `json:"contexts,omitempty"`
		*alias
	}{
		StartTimestamp: t.StartTimestamp.UTC().Format(time.RFC3339),
		Timestamp:      t.Timestamp.UTC().Format(time.RFC3339),
		Type:           "transaction",
		Contexts:       t.TraceContext,
		alias:          (*alias)(&t),
	})
}

func transactionFromSpans(rootSpan *SentrySpan, childSpans []*SentrySpan) *SentryTransaction {
	transaction := &SentryTransaction{
		SentryEvent:    (*SentryEvent)(sentry.NewEvent()),
		StartTimestamp: rootSpan.StartTimestamp,
		TraceContext: TraceContext{
			TraceID:     rootSpan.TraceID,
			SpanID:      rootSpan.SpanID,
			Op:          rootSpan.Op,
			Description: rootSpan.Description,
		},
		Spans: childSpans,
	}

	transaction.Tags = rootSpan.Tags
	transaction.Timestamp = rootSpan.EndTimestamp
	transaction.Transaction = rootSpan.Description

	return transaction
}
