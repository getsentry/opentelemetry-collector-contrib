# Opentelemetry to Sentry Transformation

This document aims to define the transformations between an OpenTelemetry (otel) span and a Sentry Span. It will also describe how a Sentry transaction is created from a set of Sentry spans.

## Spans

The interface for a Sentry Span can be found [here](https://develop.sentry.dev/sdk/event-payloads/span/)

| Sentry | OpenTelemetry | Notes |
|---------------------|---------------------------------------|--------------------------------------------------------------------------|
| Span.TraceID | Span.TraceID |  |
| Span.SpanID | Span.SpanID |  |
| Span.ParentSpanID | Span.ParentSpanID | If a span does not have a parent span ID, it is a root span (considered the start of a new transaction in Sentry) |
| Span.Description | Span.Name, Span.Attributes, Span.Kind | The span description is decided using OpenTelemetry Semantic Conventions |
| Span.Op | Span.Name, Span.Attributes, Span.Kind | The span op is decided using OpenTelemetry Semantic Conventions |
| Span.Tags | Span.Attributes, Span.Kind, Span.Status | The otel span status message and span kind are stored as tags on the Sentry span |
| Span.StartTimestamp | span.StartTime |  |
| Span.EndTimestamp | span.EndTime |  |
| Span.Status | Span.Status |  |

As can be seen by the table above, the Otel span and Sentry spans map one to one fairly reasonably. Currently the OpenTelemtry `Span.Link` and `Span.TraceState` are not used when constructing a `SentrySpan`

## Transactions

In order to injest spans into Sentry, they must be sorted into transactions, which is made up of a root span and it's corresponding child spans, along with useful metadata.

A couple of definitions to lay out to understand the implementation here:

1. **Root Span** A span with no parent
2. **Root Span Tree** A data structure that contains a root span and all it's children
3. **Orphan Span** A span which cannot be associated to any root span tree

### Implementation

We first iterate through all spans in a trace to figure out which spans are root spans. As with our definition, this can be easily done by just checking for an empty parent span id. If a root span is found, we can assign it a new root span tree. Along the way, if we find any children that belong to a root span tree, we can assign them to a root span tree.

After this first iteration, we are left with two structures, an array of root span trees, and an array of orphan spans, which we could not classify the first pass.

We can then try again to classify these orphan spans, but if not possible, we can assume these orphan spans to be a root span (as we could not find their parent in the trace). Those root spans generated from orphan spans can be also be then used to create their own respective root span tree.

We can then generate a transactions from each root span tree.

The interface for a Sentry Transction can be found [here](https://develop.sentry.dev/sdk/event-payloads/transaction/)

| Sentry | Used to generate |
|-------------------------------|----------------------------------------------------------------------|
| Transaction.Contexts["trace"] | RootSpan.TraceID, RootSpan.SpanID, RootSpan.Op, RootSpan.Description |
| Transaction.Spans | ChildSpans |
| Transaction.Sdk.Name | InstrumentationLibrary.Name |
| Transaction.Sdk.Version | InstrumentationLibrary.Version |
| Transaction.Tags | Resource.Attributes, RootSpan.Tags |
| Transaction.StartTimestamp | RootSpan.StartTimestamp |
| Transaction.Timestamp | RootSpan.EndTimestamp |
| Transaction.Transaction | RootSpan.Description |


