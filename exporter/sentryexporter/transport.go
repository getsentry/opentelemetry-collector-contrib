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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
)

const defaultBufferSize = 30
const defaultRetryAfter = time.Second * 60
const defaultTimeout = time.Second * 30

func transactionToEnvelope(t *SentryTransaction) (envelope *bytes.Buffer, err error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)

	fmt.Fprintf(&b, `{"sent_at":"%s"}`, time.Now().UTC().Format(time.RFC3339Nano))
	fmt.Fprint(&b, "\n", `{"type":"transaction"}`, "\n")
	err = enc.Encode(t)
	return &b, err
}

// A SentryTransport is used to deliver events to a remote server
type SentryTransport struct {
	DSN       *sentry.Dsn
	client    *http.Client
	transport http.RoundTripper

	buffer        chan *http.Request
	disabledUntil time.Time
	mu            sync.RWMutex

	wg    sync.WaitGroup
	start sync.Once

	// Size of the transport buffer. Defaults to 30.
	BufferSize int
	// HTTP Client request timeout. Defaults to 30 seconds.
	Timeout time.Duration
}

// NewSentryTransport returns a new pre-configured instance of SentryTransport
func NewSentryTransport() *SentryTransport {
	return &SentryTransport{
		BufferSize: defaultBufferSize,
		Timeout:    defaultTimeout,
	}
}

// Configure configures the SentryTransport based on provided config
func (t *SentryTransport) Configure(config *Config) {
	DSN, err := sentry.NewDsn(config.DSN)
	if err != nil {
		log.Printf("%v\n", err)
		return
	}

	t.DSN = DSN
	t.buffer = make(chan *http.Request, t.BufferSize)

	t.client = &http.Client{
		Transport: t.transport,
		Timeout:   t.Timeout,
	}

	t.start.Do(func() {
		go t.worker()
	})
}

// SendTransaction send a transaction to a remote server
func (t *SentryTransport) SendTransaction(transaction *SentryTransaction) error {
	if t.DSN == nil {
		return errors.New("Invalid DSN. Not sending Transaction")
	}

	t.mu.RLock()
	disabled := time.Now().Before(t.disabledUntil)
	t.mu.RUnlock()
	if disabled {
		return errors.New("Transport is disabled, cannot send transactions")
	}

	request, err := getRequest(transaction, t.DSN)
	if err != nil {
		return err
	}

	for headerKey, headerValue := range t.DSN.RequestHeaders() {
		request.Header.Set(headerKey, headerValue)
	}

	t.wg.Add(1)

	select {
	case t.buffer <- request:
		return nil
	default:
		t.wg.Done()
		return errors.New("Event dropped due to transport buffer being full")
	}
}

func (t *SentryTransport) worker() {
	for request := range t.buffer {
		t.mu.RLock()
		disabled := time.Now().Before(t.disabledUntil)
		t.mu.RUnlock()
		if disabled {
			t.wg.Done()
			continue
		}

		response, _ := t.client.Do(request)

		if response != nil && response.StatusCode == http.StatusTooManyRequests {
			deadline := time.Now().Add(retryAfter(time.Now(), response))
			t.mu.Lock()
			t.disabledUntil = deadline
			t.mu.Unlock()
		}

		t.wg.Done()
	}
}

// Flush waits until any buffered events are sent to the Sentry server, blocking
// for at most the given timeout. It returns false if the timeout was reached.
func (t *SentryTransport) Flush(timeout time.Duration) bool {
	toolate := time.After(timeout)
	c := make(chan struct{})

	go func() {
		t.wg.Wait()
		close(c)
	}()

	select {
	case <-c:
		return true
	case <-toolate:
		return false
	}
}

func retryAfter(now time.Time, r *http.Response) time.Duration {
	retryAfterHeader := r.Header["Retry-After"]

	if retryAfterHeader == nil {
		return defaultRetryAfter
	}

	if date, err := time.Parse(time.RFC1123, retryAfterHeader[0]); err == nil {
		return date.Sub(now)
	}

	if seconds, err := strconv.Atoi(retryAfterHeader[0]); err == nil {
		return time.Second * time.Duration(seconds)
	}

	return defaultRetryAfter
}

func getRequest(transaction *SentryTransaction, DSN *sentry.Dsn) (request *http.Request, err error) {
	var body *bytes.Buffer
	URL := ""
	envURL, err := envelopeAPIURL(DSN)
	if err == nil {
		URL = envURL

		envelope, err := transactionToEnvelope(transaction)
		if err != nil {
			return nil, err
		}

		body = envelope
	} else {
		URL = DSN.StoreAPIURL().String()

		b, err := json.Marshal(transaction)
		if err != nil {
			return nil, err
		}

		body = bytes.NewBuffer(b)
	}

	request, _ = http.NewRequest(
		http.MethodPost,
		URL,
		body,
	)

	return request, nil
}

func envelopeAPIURL(DSN *sentry.Dsn) (string, error) {
	url := DSN.StoreAPIURL()

	if strings.HasSuffix(url.Path, "/store/") {
		url.Path = strings.Replace(url.Path, "/store/", "/envelope/", -1)

		return url.String(), nil
	}

	return "", errors.New("Envelope URL cannot be generated")
}
