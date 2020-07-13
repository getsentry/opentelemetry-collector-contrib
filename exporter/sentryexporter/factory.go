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
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/config/configerror"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
)

const (
	typeStr = "sentry"
)

// Factory is the factory for the Sentry Exporter.
type Factory struct {
}

// Type gets the type of the Exporter config created by this factory.
func (f *Factory) Type() configmodels.Type {
	return typeStr
}

// CreateDefaultConfig creates the default configuration for the Sentry Exporter.
func (f *Factory) CreateDefaultConfig() configmodels.Exporter {
	return &Config{
		ExporterSettings: configmodels.ExporterSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
		DSN: "",
	}
}

// CreateTraceExporter creates a trace exporter based on the Sentry config.
func (f *Factory) CreateTraceExporter(ctx context.Context, params component.ExporterCreateParams, config configmodels.Exporter) (component.TraceExporter, error) {
	sentryConfig, ok := config.(*Config)
	if !ok {
		return nil, fmt.Errorf("Unexpected config type: %T", config)
	}

	// Create exporter based on sentry config.
	exp, err := CreateSentryExporter(sentryConfig)
	return exp, err
}

// CreateMetricsExporter creates a metrics exporter based on the Sentry config.
// This function is currently a no-op as Sentry does not accept metrics data
func (f *Factory) CreateMetricsExporter(ctx context.Context, params component.ExporterCreateParams,
	cfg configmodels.Exporter) (component.MetricsExporter, error) {
	return nil, configerror.ErrDataTypeIsNotSupported
}
