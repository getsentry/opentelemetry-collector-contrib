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

/*
	for trace d6c4f03650bd47699ec65c84352b6208:
	rootSpan1 <- childSpan1 <- childChildSpan1
	rootSpan1 <- childSpan2
	rootSpan2 <- root2childSpan
	orphanSpan
*/

var (
	rootSpan1 = &sentry.Span{
		TraceID:      "d6c4f03650bd47699ec65c84352b6208",
		SpanID:       "1cc4b26ab9094ef0",
		ParentSpanID: "",
		Description:  "/api/users/{user_id}",
		Op:           "http.server",
		Tags: map[string]string{
			"organization":   "12345",
			"status_message": "HTTP OK",
			"span_kind":      "server",
		},
		StartTimestamp: unixNanoToTime(5),
		EndTimestamp:   unixNanoToTime(10),
		Status:         "ok",
	}

	childSpan1 = &sentry.Span{
		TraceID:      "d6c4f03650bd47699ec65c84352b6208",
		SpanID:       "93ba92db3fa24752",
		ParentSpanID: "1cc4b26ab9094ef0",
		Description:  `SELECT * FROM user WHERE "user"."id" = {id}`,
		Op:           "db",
		Tags: map[string]string{
			"function_name":  "get_users",
			"status_message": "MYSQL OK",
			"span_kind":      "server",
		},
		StartTimestamp: unixNanoToTime(5),
		EndTimestamp:   unixNanoToTime(7),
		Status:         "ok",
	}

	childChildSpan1 = &sentry.Span{
		TraceID:      "d6c4f03650bd47699ec65c84352b6208",
		SpanID:       "1fa8913ec3814d34",
		ParentSpanID: "93ba92db3fa24752",
		Description:  `DB locked`,
		Op:           "db",
		Tags: map[string]string{
			"db_status":      "oh no im locked rn",
			"status_message": "MYSQL OK",
			"span_kind":      "server",
		},
		StartTimestamp: unixNanoToTime(6),
		EndTimestamp:   unixNanoToTime(7),
		Status:         "ok",
	}

	childSpan2 = &sentry.Span{
		TraceID:      "d6c4f03650bd47699ec65c84352b6208",
		SpanID:       "34efcde268684bb0",
		ParentSpanID: "1cc4b26ab9094ef0",
		Description:  "Serialize stuff",
		Op:           "",
		Tags: map[string]string{
			"span_kind": "server",
		},
		StartTimestamp: unixNanoToTime(7),
		EndTimestamp:   unixNanoToTime(10),
		Status:         "ok",
	}

	orphanSpan1 = &sentry.Span{
		TraceID:        "d6c4f03650bd47699ec65c84352b6208",
		SpanID:         "6241111811384fae",
		ParentSpanID:   "1930bb5cc45c4003",
		Description:    "A random span",
		Op:             "",
		StartTimestamp: unixNanoToTime(3),
		EndTimestamp:   unixNanoToTime(6),
		Status:         "ok",
	}

	rootSpan2 = &sentry.Span{
		TraceID:      "d6c4f03650bd47699ec65c84352b6208",
		SpanID:       "4c7f56556ffe4e4a",
		ParentSpanID: "",
		Description:  "Navigating to fancy website",
		Op:           "pageload",
		Tags: map[string]string{
			"status_message": "HTTP OK",
			"span_kind":      "client",
		},
		StartTimestamp: unixNanoToTime(0),
		EndTimestamp:   unixNanoToTime(5),
		Status:         "ok",
	}

	root2childSpan = &sentry.Span{
		TraceID:      "d6c4f03650bd47699ec65c84352b6208",
		SpanID:       "7ff3c8daf8184fee",
		ParentSpanID: "4c7f56556ffe4e4a",
		Description:  "<FancyReactComponent />",
		Op:           "react",
		Tags: map[string]string{
			"span_kind": "server",
		},
		StartTimestamp: unixNanoToTime(4),
		EndTimestamp:   unixNanoToTime(5),
		Status:         "ok",
	}

	transaction1 = transactionFromTree(&rootSpanTree{
		rootSpan:   rootSpan1,
		childSpans: []*sentry.Span{childSpan1, childChildSpan1, childSpan2},
	})
	transaction2 = transactionFromTree(&rootSpanTree{
		rootSpan:   rootSpan2,
		childSpans: []*sentry.Span{root2childSpan},
	})
)
