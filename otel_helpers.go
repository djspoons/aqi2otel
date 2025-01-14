package aqi2otel

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func newFloat64Counter(meter metric.Meter, name string) metric.Float64Counter {
	counter, err := meter.Float64Counter(name)
	if err != nil {
		panic(err)
	}
	return counter
}

func setNewFloat64Gauge(ctx context.Context, meter metric.Meter, sensorName, name string, val float64) {
	gauge, err := meter.Float64Gauge(name)
	if err != nil {
		panic(err)
	}
	gauge.Record(ctx, val, metric.WithAttributes(attribute.String("name", sensorName)))
}
