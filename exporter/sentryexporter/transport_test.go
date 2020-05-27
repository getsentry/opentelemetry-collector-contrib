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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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
