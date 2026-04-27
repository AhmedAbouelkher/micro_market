# custom-collector

Custom OpenTelemetry Collector for `micro_market`.

It builds `otelcol-dev` with one custom logs processor from `./cprocessor`.

## What it does

- Receives OTLP logs on `0.0.0.0:4317` and `0.0.0.0:4318`
- Adds `deployment.environment` to resource logs
- Optionally adds `collector.processed=true` to each log record
- Optionally redacts `user.email` values
- Exports logs with `otlphttp` to `http://grafana-otel:3100/otlp`

## Build

```bash
go run go.opentelemetry.io/collector/cmd/builder@v0.148.0 --config builder-config.yaml
```

## Run

```bash
./otelcol-dev --config collector-config.yaml
```

## Custom processor config

```yaml
processors:
  cprocessor:
    environment: micro_market
    add_log_attribute: true
    redact_user_email: true
```

- `environment`: required
- `add_log_attribute`: adds `collector.processed=true`
- `redact_user_email`: replaces `user.email` with `[REDACTED]`

## Generated files

- `otelcol-dev/` is generated output
- `otelcol-dev` binary is ignored
