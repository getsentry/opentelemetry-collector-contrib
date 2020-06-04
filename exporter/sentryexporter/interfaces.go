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

func transactionFromTree(rtree *rootSpanTree) *sentry.Event {
	transaction := sentry.NewEvent()

	transaction.Contexts["trace"] = sentry.TraceContext{
		TraceID:     rtree.rootSpan.TraceID,
		SpanID:      rtree.rootSpan.SpanID,
		Op:          rtree.rootSpan.Op,
		Description: rtree.rootSpan.Description,
		Status:      rtree.rootSpan.Status,
	}

	transaction.Type = "transaction"

	transaction.Sdk.Name = rtree.libraryName
	transaction.Sdk.Version = rtree.libraryVersion

	transaction.Spans = rtree.childSpans
	transaction.StartTimestamp = rtree.rootSpan.StartTimestamp
	transaction.Tags = rtree.rootSpan.Tags
	transaction.Timestamp = rtree.rootSpan.EndTimestamp
	transaction.Transaction = rtree.rootSpan.Description

	// Transactions should store resource tags
	for k, v := range rtree.resourceTags {
		transaction.Tags[k] = v
	}

	return transaction
}
