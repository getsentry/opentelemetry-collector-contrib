module github.com/open-telemetry/opentelemetry-collector-contrib/exporter/sentryexporter

go 1.14

require (
	github.com/getsentry/sentry-go v0.6.1
	github.com/google/go-cmp v0.4.0
	github.com/open-telemetry/opentelemetry-collector v0.3.1-0.20200511154150-871119061598
	github.com/open-telemetry/opentelemetry-proto v0.3.0
	github.com/stretchr/testify v1.5.1
	go.uber.org/zap v1.15.0
)

replace github.com/getsentry/sentry-go => /Users/abhijeetprasad/workspace/sentry-go
