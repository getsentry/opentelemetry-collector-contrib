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
	"strconv"
	"strings"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exporterhelper"
	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
)

var (
	sentryStatusUnknown = "unknown"
	// canonicalCodes maps OpenTelemetry span codes to Sentry's span status.
	// See numeric codes in https://godoc.org/github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1#Status_StatusCode.
	canonicalCodes = [...]string{
		"ok",
		"cancelled",
		sentryStatusUnknown,
		"invalid_argument",
		"deadline_exceeded",
		"not_found",
		"already_exists",
		"permission_denied",
		"resource_exhausted",
		"failed_precondition",
		"aborted",
		"out_of_range",
		"unimplemented",
		"internal",
		"unavailable",
		"data_loss",
		"unauthenticated",
	}

	otelSentryExporterVersion = "0.0.1"
)

// SentryExporter defines the Sentry Exporter.
type SentryExporter struct {
	transport *SentryTransport
}

// SpanStore stores a root span and it's child spans.
type SpanStore struct {
	rootSpan   *SentrySpan
	childSpans []*SentrySpan
}

// rootSpanTree stores a root span and it's child spans.
type rootSpanTree struct {
	rootSpan   *SentrySpan
	childSpans []*SentrySpan
}

// TODO: Add to function
func (s *SentryExporter) pushTraceData(ctx context.Context, td pdata.Traces) (droppedSpans int, err error) {
	// For a ResourceSpan, InstrumentationLibrarySpan and Span struct if IsNil() is "true", all other methods will cause a runtime error.
	resourceSpans := td.ResourceSpans()
	if resourceSpans.Len() == 0 {
		return 0, nil
	}

	orphanSpans := make([]*SentrySpan, 0, td.SpanCount())

	// Maps all child span ids to their root span.
	idMap := make(map[string]string)
	// Maps root span id to a root span tree.
	rootSpanTreeMap := make(map[string]*rootSpanTree)

	for i := 0; i < resourceSpans.Len(); i++ {
		rs := resourceSpans.At(i)
		if rs.IsNil() {
			continue
		}

		resourceTags := generateTagsFromAttributes(rs.Resource().Attributes())

		ilss := rs.InstrumentationLibrarySpans()
		for j := 0; j < ilss.Len(); j++ {
			ils := ilss.At(j)
			if ils.IsNil() {
				continue
			}

			library := ils.InstrumentationLibrary()

			spans := ils.Spans()
			for k := 0; k < spans.Len(); k++ {
				otelSpan := spans.At(k)
				if otelSpan.IsNil() {
					continue
				}

				sentrySpan := convertToSentrySpan(otelSpan, resourceTags, library)

				if sentrySpan.IsRootSpan() {
					// Add root span to span store map
					rootSpanTreeMap[sentrySpan.SpanID] = &rootSpanTree{
						rootSpan:   sentrySpan,
						childSpans: make([]*SentrySpan, 0),
					}

					idMap[sentrySpan.SpanID] = sentrySpan.SpanID
				} else {
					if rootSpanID, ok := idMap[sentrySpan.ParentSpanID]; ok {
						idMap[sentrySpan.SpanID] = rootSpanID
						rootSpanTreeMap[rootSpanID].childSpans = append(rootSpanTreeMap[rootSpanID].childSpans, sentrySpan)
					} else {
						orphanSpans = append(orphanSpans, sentrySpan)
					}
				}
			}
		}
	}

	orphanSpans = classifyOrphanSpans(orphanSpans, len(orphanSpans)+1, idMap, rootSpanTreeMap)

	transactions := generateTransactions(rootSpanTreeMap, orphanSpans)

	droppedSpans = 0
	for _, t := range transactions {
		err := s.transport.SendTransaction(t)
		if err != nil {
			droppedSpans += 1 + len(t.Spans)
		}
	}

	return droppedSpans, nil
}

func generateTransactions(rootSpanTreeMap map[string]*rootSpanTree, orphanSpans []*SentrySpan) []*SentryTransaction {
	transactions := make([]*SentryTransaction, 0, len(rootSpanTreeMap)+len(orphanSpans))

	for _, spanStore := range rootSpanTreeMap {
		transaction := transactionFromSpans(spanStore.rootSpan, spanStore.childSpans)
		transactions = append(transactions, transaction)
	}

	for _, span := range orphanSpans {
		transaction := transactionFromSpans(span, nil)
		transactions = append(transactions, transaction)
	}

	return transactions
}

func classifyOrphanSpans(orphanSpans []*SentrySpan, prevLength int, idMap map[string]string, rootSpanTreeMap map[string]*rootSpanTree) []*SentrySpan {
	if len(orphanSpans) == 0 || len(orphanSpans) == prevLength {
		return orphanSpans
	}

	newOrphanSpans := make([]*SentrySpan, 0, prevLength)

	for _, span := range orphanSpans {
		if rootSpanID, ok := idMap[span.ParentSpanID]; ok {
			idMap[span.SpanID] = rootSpanID
			rootSpanTreeMap[rootSpanID].childSpans = append(rootSpanTreeMap[rootSpanID].childSpans, span)
		} else {
			newOrphanSpans = append(newOrphanSpans, span)
		}
	}

	return classifyOrphanSpans(newOrphanSpans, len(orphanSpans), idMap, rootSpanTreeMap)
}

// TODO: Span.Link
// TODO; Span.Event -> create breadcrumbs
// TODO: Span.TraceState()
func convertToSentrySpan(span pdata.Span, resourceTags Tags, library pdata.InstrumentationLibrary) (sentrySpan *SentrySpan) {
	if span.IsNil() {
		return nil
	}

	parentSpanID := ""
	if psID := span.ParentSpanID(); !isAllZero(psID) {
		parentSpanID = psID.String()
	}

	attributes := span.Attributes()
	name := span.Name()
	spanKind := span.Kind()

	op, description := generateSpanDescriptors(name, attributes, spanKind)
	tags := generateTagsFromAttributes(attributes)

	status, message := statusFromSpanStatus(span.Status())

	if message != "" {
		tags["status_message"] = message
	}

	if spanKind != pdata.SpanKindUNSPECIFIED {
		tags["span_kind"] = spanKind.String()
	}

	sentrySpan = &SentrySpan{
		TraceID:        span.TraceID().String(),
		SpanID:         span.SpanID().String(),
		ParentSpanID:   parentSpanID,
		Description:    description,
		Op:             op,
		Tags:           tags,
		StartTimestamp: unixNanoToTime(span.StartTime()),
		EndTimestamp:   unixNanoToTime(span.EndTime()),
		Status:         status,
		ResourceTags:   resourceTags,
	}

	if !library.IsNil() {
		name := library.Name()
		version := library.Version()

		if name != "" && version != "" {
			sentrySpan.LibName = name
			sentrySpan.LibVersion = version
		} else {
			sentrySpan.LibName = "sentry.opentelemetry.collector"
			sentrySpan.LibVersion = otelSentryExporterVersion
		}
	} else {
		sentrySpan.LibName = "sentry.opentelemetry.collector"
		sentrySpan.LibVersion = otelSentryExporterVersion
	}

	return sentrySpan
}

// To generate span descriptors (op and description) for a particular span we use
// Semantic Conventions described by the open telemetry specification.
func generateSpanDescriptors(name string, attrs pdata.AttributeMap, spanKind pdata.SpanKind) (op string, description string) {
	// See https://github.com/open-telemetry/opentelemetry-specification/tree/5b78ee1/specification/trace/semantic_conventions
	// for more details about the semantic conventions.
	var opBuilder strings.Builder
	var dBuilder strings.Builder

	// Generating span descriptors operates under the assumption that only one of the conventions are present.
	// In the possible case that multiple convention attributes are available, conventions are selected based
	// on what is most likely and what is most useful (ex. http is prioritized over FaaS)

	// If http.method exists, this is an http request span.
	if httpMethod, ok := attrs.Get(conventions.AttributeHTTPMethod); ok {
		opBuilder.WriteString("http")

		switch spanKind {
		case pdata.SpanKindCLIENT:
			opBuilder.WriteString(".client")
		case pdata.SpanKindSERVER:
			opBuilder.WriteString(".server")
		}

		// Ex. description="GET /api/users/{user_id}".
		fmt.Fprintf(&dBuilder, "%s %s", httpMethod.StringVal(), name)

		return opBuilder.String(), dBuilder.String()
	}

	// If db.type exists then this is a database call span.
	if _, ok := attrs.Get(conventions.AttributeDBType); ok {
		// TODO: Use more detailed op code?
		opBuilder.WriteString("db")

		// Use DB statement (Ex "SELECT * FROM table") if possible as description.
		if statement, okInst := attrs.Get(conventions.AttributeDBStatement); okInst {
			dBuilder.WriteString(statement.StringVal())
		} else {
			dBuilder.WriteString(name)
		}

		return opBuilder.String(), dBuilder.String()
	}

	// If rpc.service exists then this is a rpc call span.
	if _, ok := attrs.Get(conventions.AttributeRPCService); ok {
		opBuilder.WriteString("rpc")

		return opBuilder.String(), name
	}

	// If messaging.system exists then this is a messaging system span.
	if _, ok := attrs.Get("messaging.system"); ok {
		opBuilder.WriteString("message")

		return opBuilder.String(), name
	}

	// If faas.trigger exists then this is a function as a service span.
	if trigger, ok := attrs.Get("faas.trigger"); ok {
		opBuilder.WriteString(trigger.StringVal())

		return opBuilder.String(), name
	}

	// Default just use span.name.
	return "", name
}

func generateTagsFromAttributes(attrs pdata.AttributeMap) Tags {
	tags := make(map[string]string)

	attrs.ForEach(func(key string, attr pdata.AttributeValue) {
		switch attr.Type() {
		case pdata.AttributeValueSTRING:
			tags[key] = attr.StringVal()
		case pdata.AttributeValueBOOL:
			tags[key] = strconv.FormatBool(attr.BoolVal())
		case pdata.AttributeValueDOUBLE:
			tags[key] = strconv.FormatFloat(attr.DoubleVal(), 'g', -1, 64)
		case pdata.AttributeValueINT:
			tags[key] = strconv.FormatInt(attr.IntVal(), 10)
		}
	})

	return tags
}

func statusFromSpanStatus(spanStatus pdata.SpanStatus) (status string, message string) {
	if spanStatus.IsNil() {
		return "", ""
	}

	code := spanStatus.Code()
	if code < 0 || int(code) >= len(canonicalCodes) {
		return sentryStatusUnknown, fmt.Sprintf("error code %d", code)
	}

	return canonicalCodes[code], spanStatus.Message()
}

// CreateSentryExporter returns a new Sentry Exporter.
func CreateSentryExporter(config *Config) (component.TraceExporter, error) {
	transport := NewSentryTransport()
	transport.Configure(config)

	s := &SentryExporter{
		transport: transport,
	}

	return exporterhelper.NewTraceExporter(config, s.pushTraceData)
}
