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
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestEnvelopeSnapshot(t *testing.T) {
	DSN, err := sentry.NewDsn("https://publicKey:secretKey@host/path/42")
	if err != nil {
		t.Fatal(err)
	}

	env, err := generateEnvelope(transaction1, DSN)
	got := []byte(env)

	golden := filepath.Join(".", "testdata", fmt.Sprintf("%s.golden", "sentry_envelope"))
	if *update {
		err := ioutil.WriteFile(golden, got, 0644)
		if err != nil {
			t.Error(err)
		}
	}

	want, err := ioutil.ReadFile(golden)
	if err != nil {
		t.Error(err)
	}

	if !bytes.Equal(got, want) {
		t.Errorf("struct %s\n\tgot:\n%s\n\twant:\n%s", "sentry_envelope", got, want)
	}
}

func TestGenerateEnvelope(t *testing.T) {
	DSN, err := sentry.NewDsn("https://publicKey:secretKey@host/path/42")
	if err != nil {
		t.Fatal(err)
	}

	env, err := generateEnvelope(transaction1, DSN)

	envParts := strings.Split(env, "\n")
	assert.Len(t, envParts, 4)
	assert.Empty(t, envParts[3])

	// Header
	header := &EnvelopeHeader{}
	json.Unmarshal([]byte(envParts[0]), header)
	if err != nil {
		t.Error(err)
	} else {
		assert.Equal(t, DSN.String(), header.DSN)

		_, err = uuid.Parse(header.EventID)
		if err != nil {
			t.Error("Event ID not a valid UUID")
		}
	}

	// Item Header
	assert.Equal(t, `{"type":"transaction"}`, envParts[1])

	// Item Payload
	payload := envParts[2]
	assert.NotEmpty(t, payload)
}
