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
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "update .golden files")

func TestMarshalStruct(t *testing.T) {

	testCases := []struct {
		testName     string
		sentryStruct interface{}
	}{
		{
			testName:     "sentry_span",
			sentryStruct: rootSpan1,
		},
		{
			testName:     "sentry_transaction",
			sentryStruct: transaction1,
		},
	}

	for _, test := range testCases {
		t.Run(test.testName, func(t *testing.T) {
			got, err := json.MarshalIndent(test.sentryStruct, "", "    ")
			if err != nil {
				t.Error(err)
			}

			golden := filepath.Join(".", "testdata", fmt.Sprintf("%s.golden", test.testName))
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
				t.Errorf("struct %s\n\tgot:\n%s\n\twant:\n%s", test.testName, got, want)
			}
		})
	}
}
