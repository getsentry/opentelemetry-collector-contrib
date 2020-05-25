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
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// A testHTTPServer counts events sent to it. It requires a call to Unblock
// before incrementing its internal counter and sending a response to the HTTP
// client. This allows for coordinating the execution flow when needed.
type testHTTPServer struct {
	*httptest.Server
	// transactionCounter counts the number of events processed by the server.
	transactionCounter *uint64
}

func (ts *testHTTPServer) TransactionCount() uint64 {
	return atomic.LoadUint64(ts.transactionCounter)
}

func newTestSentryServer(t *testing.T) *testHTTPServer {
	transactionCounter := new(uint64)
	handler := func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(transactionCounter, 1)
	}

	return &testHTTPServer{
		Server:             httptest.NewTLSServer(http.HandlerFunc(handler)),
		transactionCounter: transactionCounter,
	}
}

func TestTransactionEnvelope(t *testing.T) {
	b, err := transactionToEnvelope(transaction1)
	if err != nil {
		t.Fatal(err)
	}

	env := string(b)

	envParts := strings.Split(env, "\n")
	assert.Len(t, envParts, 4)
	assert.Empty(t, envParts[3])

	// Header
	var header map[string]interface{}
	err = json.Unmarshal([]byte(envParts[0]), &header)
	if err != nil {
		t.Fatal(err)
	}

	sentAtStr := header["sent_at"].(string)

	_, err = time.Parse(time.RFC3339Nano, sentAtStr)
	if err != nil {
		t.Fatal(err)
	}

	// Item Header
	assert.Equal(t, `{"type":"transaction"}`, envParts[1])

	// Item Payload
	payload := envParts[2]
	assert.NotEmpty(t, payload)
}

func TestNewSentryTransport(t *testing.T) {
	transport := NewSentryTransport()
	assert.Equal(t, defaultBufferSize, transport.BufferSize)
	assert.Equal(t, defaultTimeout, transport.Timeout)
}

func TestConfigure(t *testing.T) {
	cfg := &Config{
		DSN: "https://key@host/path/42",
	}

	transport := NewSentryTransport()
	transport.Configure(cfg)

	assert.Equal(t, cfg.DSN, transport.DSN.String())
	assert.Equal(t, cap(transport.buffer), transport.BufferSize)

	assert.Equal(t, transport.Timeout, transport.client.Timeout)
	assert.Equal(t, transport.transport, transport.client.Transport)
}

func TestFlush(t *testing.T) {
	server := newTestSentryServer(t)
	defer server.Close()

	cfg := &Config{
		DSN: fmt.Sprintf("https://test@%s/1", server.Listener.Addr()),
	}

	t.Run("with flushed events", func(t *testing.T) {
		transport := NewSentryTransport()
		transport.Configure(cfg)
		transport.client = server.Client()

		ok := transport.Flush(100 * time.Millisecond)
		if !ok {
			t.Fatalf("Flush() timed out")
		}
	})

	t.Run("with timeout", func(t *testing.T) {
		transport := NewSentryTransport()
		transport.Configure(cfg)
		transport.client = server.Client()
		transport.wg.Add(10000)

		ok := transport.Flush(1)
		if ok {
			t.Fatalf("Flush() did not timeout")
		}
	})
}

func TestRetryAfter(t *testing.T) {
	testCases := []struct {
		testName string
		// input
		now      time.Time
		response *http.Response
		// output
		duration time.Duration
	}{
		{
			testName: "No Header",
			now:      time.Now(),
			response: &http.Response{},
			duration: time.Second * 60,
		},
		{
			testName: "Incorrect Header",
			now:      time.Now(),
			response: &http.Response{
				Header: map[string][]string{
					"Return-After": {"x"},
				},
			},
			duration: time.Second * 60,
		},
		{
			testName: "Delay Header",
			now: func() time.Time {
				now, _ := time.Parse(time.RFC1123, "Wed, 21 Oct 2015 07:28:00 GMT")
				return now
			}(),
			response: &http.Response{
				Header: map[string][]string{
					"Retry-After": {"Wed, 21 Oct 2015 07:28:13 GMT"},
				},
			},
			duration: time.Second * 13,
		},
	}

	for _, test := range testCases {
		t.Run(test.testName, func(t *testing.T) {
			assert.Equal(t, retryAfter(test.now, test.response), test.duration)
		})
	}
}
