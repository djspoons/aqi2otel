module github.com/djspoons/aqi2otel

go 1.16

require (
	github.com/go-logr/stdr v1.2.2
	go.opentelemetry.io/otel v1.8.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v0.31.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v0.31.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v0.31.0
	go.opentelemetry.io/otel/metric v0.31.0
	go.opentelemetry.io/otel/sdk v1.8.0
	go.opentelemetry.io/otel/sdk/metric v0.31.0
)
