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
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
)

// EnvelopeHeader represents the top level header of a Sentry envelope
type EnvelopeHeader struct {
	EventID string `json:"event_id"`
	DSN     string `json:"dsn"`
}

func generateEnvelope(transaction *SentryTransaction, DSN *sentry.Dsn) (envelope string, err error) {
	eventID, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	header := &EnvelopeHeader{
		EventID: eventID.String(),
		DSN:     DSN.String(),
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}

	var env strings.Builder

	// Header
	_, err = fmt.Fprintf(&env, "%s%s", headerJSON, "\n")
	if err != nil {
		return "", err
	}

	// Item Header
	_, err = fmt.Fprintf(&env, "%s%s", `{"type":"transaction"}`, "\n")
	if err != nil {
		return "", err
	}

	transactionJSON, err := json.Marshal(transaction)
	if err != nil {
		return "", err
	}

	// Item Payload
	_, err = fmt.Fprintf(&env, "%s%s", transactionJSON, "\n")
	if err != nil {
		return "", err
	}

	return env.String(), nil
}
