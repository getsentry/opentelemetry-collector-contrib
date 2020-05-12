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
	"testing"
	"time"

	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
	otlptrace "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"
	"github.com/stretchr/testify/assert"
)

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

func TestGenerateStatusFromSpanStatus(t *testing.T) {
	t.Run("with unknown status code", func(t *testing.T) {
		spanStatus := pdata.NewSpanStatus()

		status, message := generateStatusFromSpanStatus(spanStatus)

		assert.Equal(t, "unknown", status)
		assert.Equal(t, "", message)
	})

	t.Run("with status code", func(t *testing.T) {
		spanStatus := pdata.NewSpanStatus()
		spanStatus.InitEmpty()
		spanStatus.SetMessage("message")
		spanStatus.SetCode(pdata.StatusCode(otlptrace.Status_ResourceExhausted))

		status, message := generateStatusFromSpanStatus(spanStatus)

		assert.Equal(t, "resource_exhausted", status)
		assert.Equal(t, "message", message)
	})
}

func TestSpanToSentrySpan(t *testing.T) {
	t.Run("with nil span", func(t *testing.T) {
		testSpan := pdata.NewSpan()

		sentrySpan, isRootSpan := spanToSentrySpan(testSpan)
		assert.Nil(t, sentrySpan)
		assert.False(t, isRootSpan)
	})

	t.Run("with root span", func(t *testing.T) {
		testSpan := pdata.NewSpan()
		testSpan.InitEmpty()

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
		attributes := pdata.NewAttributeMap()
		attributes.InsertString("key", "value")
		var startTime pdata.TimestampUnixNano = 123
		var endTime pdata.TimestampUnixNano = 1234567890
		kind := pdata.SpanKindCLIENT
		statusMessage := "message"

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

		sentrySpan, isRootSpan := spanToSentrySpan(testSpan)

		assert.Equal(t, "01020304050607080807060504030201", sentrySpan.TraceID)
		assert.Equal(t, "0102030405060708", sentrySpan.SpanID)
		assert.Equal(t, "0807060504030201", sentrySpan.ParentSpanID)
		assert.Equal(t, name, sentrySpan.Description)
		assert.Equal(t, "", sentrySpan.Op)

		// TODO: Finish the rest of the cases

		assert.NotNil(t, sentrySpan)
		assert.False(t, isRootSpan)
	})
}

type SpanDescriptorsCase struct {
	testName    string
	name        string
	attrs       pdata.AttributeMap
	spanKind    pdata.SpanKind
	op          string
	description string
}

func generateSpanDescriptorsTestCases() []SpanDescriptorsCase {
	return []SpanDescriptorsCase{
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
}

func TestGenerateSpanDescriptors(t *testing.T) {
	allTestCases := generateSpanDescriptorsTestCases()

	for _, test := range allTestCases {
		t.Run(test.testName, func(t *testing.T) {
			op, description := generateSpanDescriptors(test.name, test.attrs, test.spanKind)
			assert.Equal(t, test.op, op)
			assert.Equal(t, test.description, description)
		})
	}
}

func TestSpanStore(t *testing.T) {
	StartSpan := &SentrySpan{
		StartTimestamp: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		Timestamp:      time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC),
	}

	EndSpan := &SentrySpan{
		StartTimestamp: time.Date(2020, 1, 3, 0, 0, 0, 0, time.UTC),
		Timestamp:      time.Date(2020, 1, 4, 0, 0, 0, 0, time.UTC),
	}

	BigSpan := &SentrySpan{
		StartTimestamp: time.Date(2019, 12, 20, 0, 0, 0, 0, time.UTC),
		Timestamp:      time.Date(2020, 1, 6, 0, 0, 0, 0, time.UTC),
	}

	store := &SpanStore{}
	store.init(5)

	assert.Equal(t, cap(store.spans), 5)

	store.updateStore(StartSpan)
	assert.Equal(t, store.earliestSpan, StartSpan)
	assert.Equal(t, store.latestSpan, StartSpan)

	store.updateStore(EndSpan)
	assert.Equal(t, store.earliestSpan, StartSpan)
	assert.Equal(t, store.latestSpan, EndSpan)

	store.updateStore(StartSpan)
	assert.Equal(t, store.earliestSpan, StartSpan)
	assert.Equal(t, store.latestSpan, EndSpan)

	store.updateStore(BigSpan)
	assert.Equal(t, store.earliestSpan, BigSpan)
	assert.Equal(t, store.latestSpan, BigSpan)
}
