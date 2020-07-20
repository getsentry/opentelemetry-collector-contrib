# Sentry Exporter

The Sentry Exporter allows you to send traces to [Sentry](https://sentry.io/).

For more details about distributed tracing in Sentry, please view [our documentation](https://docs.sentry.io/performance-monitoring/distributed-tracing/).

The following configuration options are supported:

- `dsn`: The DSN tells the exporter where to send the events. If this value is not provided, the exporter will try to read it from the `SENTRY_DSN` environment variable. If that variable also does not exist, the exporter will not send any events.

Example:

```yaml
exporters:
  sentry:
    dsn: https://key@host/path/42
```

See the [docs](./docs/transformation.md) for more details on how this transformation is working.

### Known Limitations

Currently Sentry Tracing leverages a transaction based system, where a transaction contains one or more spans. The exporter will try to group spans from a trace under one or more transactions based on internal heuristics, but this may lead to certain transactions that contain only one or two spans. These transactions will still be viewable and associated under a single trace in the Sentry UI.
