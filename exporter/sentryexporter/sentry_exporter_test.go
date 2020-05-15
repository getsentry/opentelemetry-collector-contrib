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

		sentrySpan, isRootSpan := spanToSentrySpan(testSpan)
		assert.Nil(t, sentrySpan)
		assert.False(t, isRootSpan)
	})

	t.Run("with root span", func(t *testing.T) {
		testSpan := pdata.NewSpan()
		traceID := []byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1}
		spanID := []byte{1, 2, 3, 4, 5, 6, 7, 8}
		var parentSpanID []byte
		testSpan.InitEmpty()

		testSpan.SetTraceID(traceID)
		testSpan.SetSpanID(spanID)
		testSpan.SetParentSpanID(parentSpanID)

		sentrySpan, isRootSpan := spanToSentrySpan(testSpan)
		assert.NotNil(t, sentrySpan)
		assert.True(t, isRootSpan)
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

		actual, isRootSpan := spanToSentrySpan(testSpan)

		assert.NotNil(t, actual)
		assert.False(t, isRootSpan)

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
			StartTimestamp: UnixNanoToTime(startTime),
			Timestamp:      UnixNanoToTime(endTime),
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
