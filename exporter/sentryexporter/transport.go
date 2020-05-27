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
	"time"

	"github.com/getsentry/sentry-go"
)

const defaultTimeout = time.Second * 30

func transactionToEnvelope(t *SentryTransaction) (envelope []byte, err error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)

	fmt.Fprintf(&b, `{"sent_at":"%s"}`, time.Now().UTC().Format(time.RFC3339Nano))
	fmt.Fprint(&b, "\n", `{"type":"transaction"}`, "\n")
	err = enc.Encode(t)
	return b.Bytes(), err
}

// A SentryTransport is used to deliver events to a remote server
type SentryTransport struct {
	DSN       *sentry.Dsn
	client    *http.Client
	transport http.RoundTripper

	// HTTP Client request timeout. Defaults to 30 seconds.
	Timeout time.Duration
}

// NewSentryTransport returns a new pre-configured instance of SentryTransport
func NewSentryTransport() *SentryTransport {
	return &SentryTransport{
		Timeout: defaultTimeout,
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

	t.client = &http.Client{
		Transport: t.transport,
		Timeout:   t.Timeout,
	}
}

// SendTransaction send a transaction to a remote server
func (t *SentryTransport) SendTransaction(transaction *SentryTransaction) error {
	if t.DSN == nil {
		return errors.New("Invalid DSN. Not sending Transaction")
	}

	body, err := json.Marshal(transaction)
	if err != nil {
		return err
	}

	request, _ := http.NewRequest(
		http.MethodPost,
		t.DSN.StoreAPIURL().String(),
		bytes.NewBuffer(body),
	)

	for headerKey, headerValue := range t.DSN.RequestHeaders() {
		request.Header.Set(headerKey, headerValue)
	}

	_, err = t.client.Do(request)
	if err != nil {
		return fmt.Errorf("There was an issue with sending an event: %v", err)
	}

	return nil
}
