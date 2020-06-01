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

import "github.com/getsentry/sentry-go"

// IsRootSpan determines if a span is a root span.
// If parent span id is empty, then the span is a root span.
func IsRootSpan(s *sentry.Span) bool {
	return s.ParentSpanID == ""
}

func transactionFromSpans(rootSpan *sentry.Span, childSpans []*sentry.Span, libName string, libVersion string, resourceTags map[string]string) *sentry.Event {
	transaction := sentry.NewEvent()

	transaction.Contexts["trace"] = sentry.TraceContext{
		TraceID:     rootSpan.TraceID,
		SpanID:      rootSpan.SpanID,
		Op:          rootSpan.Op,
		Description: rootSpan.Description,
	}

	transaction.Sdk.Name = libName
	transaction.Sdk.Version = libVersion

	transaction.Tags = rootSpan.Tags
	transaction.Timestamp = rootSpan.EndTimestamp
	transaction.Transaction = rootSpan.Description

	// Transactions should store resource tags
	for k, v := range resourceTags {
		transaction.Tags[k] = v
	}

	return transaction
}
