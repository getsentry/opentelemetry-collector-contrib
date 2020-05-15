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
	"testing"

	"github.com/google/go-cmp/cmp"
	otlptrace "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"

	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
	"github.com/stretchr/testify/assert"
)

func TestSpanToSentrySpan(t *testing.T) {
	t.Run("with nil span", func(t *testing.T) {
		testSpan := pdata.NewSpan()

		sentrySpan := convertToSentrySpan(testSpan)
		assert.Nil(t, sentrySpan)
	})

	t.Run("with root span and nil parent span_id", func(t *testing.T) {
		testSpan := pdata.NewSpan()
		testSpan.InitEmpty()

		var parentSpanID []byte
		testSpan.SetParentSpanID(parentSpanID)

		sentrySpan := convertToSentrySpan(testSpan)
		assert.NotNil(t, sentrySpan)
		assert.True(t, sentrySpan.IsRootSpan())
	})

	t.Run("with root span and 0 byte slice", func(t *testing.T) {
		testSpan := pdata.NewSpan()
		testSpan.InitEmpty()

		parentSpanID := []byte{0, 0, 0, 0, 0, 0, 0, 0}
		testSpan.SetParentSpanID(parentSpanID)

		sentrySpan := convertToSentrySpan(testSpan)
		assert.NotNil(t, sentrySpan)
		assert.True(t, sentrySpan.IsRootSpan())
	})

	t.Run("with full span", func(t *testing.T) {
		testSpan := pdata.NewSpan()
		testSpan.InitEmpty()

		traceID := []byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1}
		spanID := []byte{1, 2, 3, 4, 5, 6, 7, 8}
		parentSpanID := []byte{8, 7, 6, 5, 4, 3, 2, 1}
		name := "span_name"
		var startTime pdata.TimestampUnixNano = 123
		var endTime pdata.TimestampUnixNano = 1234567890
		kind := pdata.SpanKindCLIENT
		statusMessage := "message"

		testSpan.Attributes().InsertString("key", "value")

		testSpan.SetTraceID(traceID)
		testSpan.SetSpanID(spanID)
		testSpan.SetParentSpanID(parentSpanID)
		testSpan.SetName(name)
		testSpan.SetStartTime(startTime)
		testSpan.SetEndTime(endTime)
		testSpan.SetKind(kind)

		testSpan.Status().InitEmpty()
		testSpan.Status().SetMessage(statusMessage)
		testSpan.Status().SetCode(pdata.StatusCode(otlptrace.Status_Ok))

		actual := convertToSentrySpan(testSpan)

		assert.NotNil(t, actual)
		assert.False(t, actual.IsRootSpan())

		expected := &SentrySpan{
			TraceID:      "01020304050607080807060504030201",
			SpanID:       "0102030405060708",
			ParentSpanID: "0807060504030201",
			Description:  name,
			Op:           "",
			Tags: map[string]string{
				"key":            "value",
				"span_kind":      pdata.SpanKindCLIENT.String(),
				"status_message": statusMessage,
			},
			StartTimestamp: unixNanoToTime(startTime),
			EndTimestamp:   unixNanoToTime(endTime),
			Status:         "ok",
		}

		if diff := cmp.Diff(expected, actual); diff != "" {
			t.Errorf("Span mismatch (-expected +actual):\n%s", diff)
		}
	})
}

type SpanDescriptorsCase struct {
	testName string
	// input
	name     string
	attrs    pdata.AttributeMap
	spanKind pdata.SpanKind
	// output
	op          string
	description string
}

func TestGenerateSpanDescriptors(t *testing.T) {
	testCases := []SpanDescriptorsCase{
		{
			testName: "http-client",
			name:     "/api/users/{user_id}",
			attrs: pdata.NewAttributeMap().InitFromMap(map[string]pdata.AttributeValue{
				conventions.AttributeHTTPMethod: pdata.NewAttributeValueString("GET"),
			}),
			spanKind:    pdata.SpanKindCLIENT,
			op:          "http.client",
			description: "GET /api/users/{user_id}",
		},
		{
			testName: "http-server",
			name:     "/api/users/{user_id}",
			attrs: pdata.NewAttributeMap().InitFromMap(map[string]pdata.AttributeValue{
				conventions.AttributeHTTPMethod: pdata.NewAttributeValueString("POST"),
			}),
			spanKind:    pdata.SpanKindSERVER,
			op:          "http.server",
			description: "POST /api/users/{user_id}",
		},
		{
			testName: "db-call-without-statement",
			name:     "SET mykey 'Val'",
			attrs: pdata.NewAttributeMap().InitFromMap(map[string]pdata.AttributeValue{
				conventions.AttributeDBType: pdata.NewAttributeValueString("redis"),
			}),
			spanKind:    pdata.SpanKindCLIENT,
			op:          "db",
			description: "SET mykey 'Val'",
		},
		{
			testName: "db-call-with-statement",
			name:     "mysql call",
			attrs: pdata.NewAttributeMap().InitFromMap(map[string]pdata.AttributeValue{
				conventions.AttributeDBType:      pdata.NewAttributeValueString("sql"),
				conventions.AttributeDBStatement: pdata.NewAttributeValueString("SELECT * FROM table"),
			}),
			spanKind:    pdata.SpanKindCLIENT,
			op:          "db",
			description: "SELECT * FROM table",
		},
		{
			testName: "rpc",
			name:     "grpc.test.EchoService/Echo",
			attrs: pdata.NewAttributeMap().InitFromMap(map[string]pdata.AttributeValue{
				conventions.AttributeRPCService: pdata.NewAttributeValueString("EchoService"),
			}),
			spanKind:    pdata.SpanKindCLIENT,
			op:          "rpc",
			description: "grpc.test.EchoService/Echo",
		},
		{
			testName: "message-system",
			name:     "message-destination",
			attrs: pdata.NewAttributeMap().InitFromMap(map[string]pdata.AttributeValue{
				"messaging.system": pdata.NewAttributeValueString("kafka"),
			}),
			spanKind:    pdata.SpanKindPRODUCER,
			op:          "message",
			description: "message-destination",
		},
		{
			testName: "faas",
			name:     "message-destination",
			attrs: pdata.NewAttributeMap().InitFromMap(map[string]pdata.AttributeValue{
				"faas.trigger": pdata.NewAttributeValueString("pubsub"),
			}),
			spanKind:    pdata.SpanKindSERVER,
			op:          "pubsub",
			description: "message-destination",
		},
	}

	for _, test := range testCases {
		t.Run(test.testName, func(t *testing.T) {
			op, description := generateSpanDescriptors(test.name, test.attrs, test.spanKind)
			assert.Equal(t, test.op, op)
			assert.Equal(t, test.description, description)
		})
	}
}

func TestGenerateTagsFromAttributes(t *testing.T) {
	attrs := pdata.NewAttributeMap()

	attrs.InsertString("string-key", "string-value")
	attrs.InsertBool("bool-key", true)
	attrs.InsertDouble("double-key", 123.123)
	attrs.InsertInt("int-key", 321)

	tags := generateTagsFromAttributes(attrs)

	stringVal, _ := tags["string-key"]
	assert.Equal(t, stringVal, "string-value")
	boolVal, _ := tags["bool-key"]
	assert.Equal(t, boolVal, "true")
	doubleVal, _ := tags["double-key"]
	assert.Equal(t, doubleVal, "123.123")
	intVal, _ := tags["int-key"]
	assert.Equal(t, intVal, "321")
}

type SpanStatusCase struct {
	testName string
	// input
	spanStatus pdata.SpanStatus
	// output
	status  string
	message string
}

func TestStatusFromSpanStatus(t *testing.T) {
	testCases := []SpanStatusCase{
		{
			testName:   "with nil status",
			spanStatus: pdata.NewSpanStatus(),
			status:     "",
			message:    "",
		},
		{
			testName: "with status code",
			spanStatus: func() pdata.SpanStatus {
				spanStatus := pdata.NewSpanStatus()
				spanStatus.InitEmpty()
				spanStatus.SetMessage("message")
				spanStatus.SetCode(pdata.StatusCode(otlptrace.Status_ResourceExhausted))

				return spanStatus
			}(),
			status:  "resource_exhausted",
			message: "message",
		},
		{
			testName: "with unimplemented status code",
			spanStatus: func() pdata.SpanStatus {
				spanStatus := pdata.NewSpanStatus()
				spanStatus.InitEmpty()
				spanStatus.SetMessage("message")
				spanStatus.SetCode(pdata.StatusCode(1337))

				return spanStatus
			}(),
			status:  "unknown",
			message: "error code 1337",
		},
	}

	for _, test := range testCases {
		t.Run(test.testName, func(t *testing.T) {
			status, message := statusFromSpanStatus(test.spanStatus)
			assert.Equal(t, test.status, status)
			assert.Equal(t, test.message, message)
		})
	}
}

type ClassifyOrphanSpanTestCase struct {
	testName string
	// input
	idMap IDMap
	ssMap SSMap
	spans []*SentrySpan
	// output
	assertion func(t *testing.T, orphanSpans []*SentrySpan)
}

func TestClassifyOrphanSpans(t *testing.T) {
	rootSpan1 := &SentrySpan{
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
		StartTimestamp: UnixNanoToTime(5),
		EndTimestamp:   UnixNanoToTime(10),
		Status:         "ok",
	}

	childSpan1 := &SentrySpan{
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
		StartTimestamp: UnixNanoToTime(5),
		EndTimestamp:   UnixNanoToTime(7),
		Status:         "ok",
	}

	childChildSpan1 := &SentrySpan{
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
		StartTimestamp: UnixNanoToTime(6),
		EndTimestamp:   UnixNanoToTime(7),
		Status:         "ok",
	}

	childSpan2 := &SentrySpan{
		TraceID:      "d6c4f03650bd47699ec65c84352b6208",
		SpanID:       "34efcde268684bb0",
		ParentSpanID: "1cc4b26ab9094ef0",
		Description:  "Serialize stuff",
		Op:           "",
		Tags: map[string]string{
			"span_kind": "server",
		},
		StartTimestamp: UnixNanoToTime(7),
		EndTimestamp:   UnixNanoToTime(10),
		Status:         "ok",
	}

	orphanSpan := &SentrySpan{
		TraceID:        "d6c4f03650bd47699ec65c84352b6208",
		SpanID:         "6241111811384fae",
		ParentSpanID:   "1930bb5cc45c4003",
		Description:    "A random span",
		Op:             "",
		Tags:           nil,
		StartTimestamp: UnixNanoToTime(3),
		EndTimestamp:   UnixNanoToTime(6),
		Status:         "ok",
	}

	rootSpan2 := &SentrySpan{
		TraceID:      "d6c4f03650bd47699ec65c84352b6208",
		SpanID:       "4c7f56556ffe4e4a",
		ParentSpanID: "",
		Description:  "Navigating to fancy website",
		Op:           "pageload",
		Tags: map[string]string{
			"status_message": "HTTP OK",
			"span_kind":      "client",
		},
		StartTimestamp: UnixNanoToTime(0),
		EndTimestamp:   UnixNanoToTime(5),
		Status:         "ok",
	}

	root2childSpan := &SentrySpan{
		TraceID:      "d6c4f03650bd47699ec65c84352b6208",
		SpanID:       "7ff3c8daf8184fee",
		ParentSpanID: "4c7f56556ffe4e4a",
		Description:  "<FancyReactComponent />",
		Op:           "react",
		Tags: map[string]string{
			"span_kind": "server",
		},
		StartTimestamp: UnixNanoToTime(4),
		EndTimestamp:   UnixNanoToTime(5),
		Status:         "ok",
	}

	testCases := []ClassifyOrphanSpanTestCase{
		{
			testName: "with no root spans",
			idMap:    make(IDMap),
			ssMap:    make(SSMap),
			spans:    []*SentrySpan{childSpan1, childSpan2},
			assertion: func(t *testing.T, orphanSpans []*SentrySpan) {
				assert.Len(t, orphanSpans, 2)
			},
		},
		{
			testName: "with no remaining orphans",
			idMap: func() IDMap {
				idMap := make(IDMap)
				idMap[rootSpan1.SpanID] = rootSpan1.SpanID
				return idMap
			}(),
			ssMap: func() SSMap {
				ssMap := make(SSMap)
				ssMap[rootSpan1.SpanID] = &SpanStore{
					rootSpan:   rootSpan1,
					childSpans: make([]*SentrySpan, 0),
				}
				return ssMap
			}(),
			spans: []*SentrySpan{childChildSpan1, childSpan1, childSpan2},
			assertion: func(t *testing.T, orphanSpans []*SentrySpan) {
				assert.Len(t, orphanSpans, 0)
			},
		},
		{
			testName: "with some remaining orphans",
			idMap: func() IDMap {
				idMap := make(IDMap)
				idMap[rootSpan1.SpanID] = rootSpan1.SpanID
				return idMap
			}(),
			ssMap: func() SSMap {
				ssMap := make(SSMap)
				ssMap[rootSpan1.SpanID] = &SpanStore{
					rootSpan:   rootSpan1,
					childSpans: make([]*SentrySpan, 0),
				}
				return ssMap
			}(),
			spans: []*SentrySpan{childChildSpan1, childSpan1, childSpan2, orphanSpan},
			assertion: func(t *testing.T, orphanSpans []*SentrySpan) {
				assert.Len(t, orphanSpans, 1)
				assert.Equal(t, orphanSpan, orphanSpans[0])
			},
		},
		{
			testName: "with some remaining orphans",
			idMap: func() IDMap {
				idMap := make(IDMap)
				idMap[rootSpan1.SpanID] = rootSpan1.SpanID
				idMap[rootSpan2.SpanID] = rootSpan2.SpanID
				return idMap
			}(),
			ssMap: func() SSMap {
				ssMap := make(SSMap)
				ssMap[rootSpan1.SpanID] = &SpanStore{
					rootSpan:   rootSpan1,
					childSpans: make([]*SentrySpan, 0),
				}
				ssMap[rootSpan2.SpanID] = &SpanStore{
					rootSpan:   rootSpan2,
					childSpans: make([]*SentrySpan, 0),
				}
				return ssMap
			}(),
			spans: []*SentrySpan{childChildSpan1, childSpan1, root2childSpan, childSpan2},
			assertion: func(t *testing.T, orphanSpans []*SentrySpan) {
				assert.Len(t, orphanSpans, 0)
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.testName, func(t *testing.T) {
			orphanSpans := classifyOrphanSpans(test.spans, len(test.spans)+1, test.idMap, test.ssMap)
			test.assertion(t, orphanSpans)
		})
	}
}
