package aqi2otel

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/syncfloat64"
	"go.opentelemetry.io/otel/metric/instrument/syncint64"
)

func newInt64Counter(meter metric.Meter, name string) syncint64.Counter {
	counter, err := meter.SyncInt64().Counter(name)
	if err != nil {
		panic(err)
	}
	return counter
}

func newFloat64Counter(meter metric.Meter, name string) syncfloat64.Counter {
	counter, err := meter.SyncFloat64().Counter(name)
	if err != nil {
		panic(err)
	}
	return counter
}

func setNewInt64Gauge(ctx context.Context, meter metric.Meter, sensorName, name string, val int64) {
	gauge, err := meter.AsyncInt64().Gauge(name)
	if err != nil {
		panic(err)
	}
	err = meter.RegisterCallback(
		[]instrument.Asynchronous{gauge},
		func(ctx context.Context) {
			//log.Println("Got callback for " + name)
			gauge.Observe(ctx, val, attribute.String("name", sensorName))
		},
	)
	if err != nil {
		panic(err)
	}
}

func setNewFloat64Gauge(ctx context.Context, meter metric.Meter, sensorName, name string, val float64) {
	gauge, err := meter.AsyncFloat64().Gauge(name)
	if err != nil {
		panic(err)
	}
	err = meter.RegisterCallback(
		[]instrument.Asynchronous{gauge},
		func(ctx context.Context) {
			//log.Println("Got callback for " + name)
			gauge.Observe(ctx, val, attribute.String("name", sensorName))
		},
	)
	if err != nil {
		panic(err)
	}
}
