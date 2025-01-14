package aqi2otel

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-logr/stdr"
	"go.opentelemetry.io/otel"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Per https://api.purpleair.com/#api-sensors-get-sensor-data, temperature
// inside the housing is "8F higher than ambient conditions."
const temperature_offset = 8

type Sample struct {
	SensorName string
	// FIXME maybe uptime should be an int?
	Uptime      float64
	Lag         float64
	Temperature float64
	Pressure    float64
	Humidity    float64
	PM25        float64
	// FIXME maybe aqi should be an int?
	AQI float64
}

func Run(ctx context.Context, useStdoutExporter bool) {
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		log.Println("Got error: " + err.Error())
	}))
	otel.SetLogger(stdr.New(log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)))

	var exporter metric.Exporter
	var err error
	if useStdoutExporter {
		log.Println("Using stdout exporter")
		exporter, err = stdoutmetric.New(stdoutmetric.WithPrettyPrint())
	} else {
		log.Println("Using OTLP gRPC exporter")
		exporter, err = otlpmetricgrpc.New(ctx,
			otlpmetricgrpc.WithEndpointURL("http://localhost:4317"),
			otlpmetricgrpc.WithInsecure(),
		)
	}
	if err != nil {
		panic(err)
	}

	var res *resource.Resource
	res, err = resource.Merge(resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL,
			semconv.ServiceNameKey.String("AQI"),
			attribute.String("sensor_id", os.Getenv("PURPLE_AIR_SENSOR_ID")),
		))
	if err != nil {
		panic(err)
	}

	provider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(exporter,
			metric.WithInterval(5*time.Second))),
	)
	otel.SetMeterProvider(provider)
	meter := provider.Meter("PurpleAir")

	// FIXME maybe scrapes should be an int?
	scrapeCounter := newFloat64Counter(meter, "scrapes")
	scrapeCounter.Add(ctx, 1)

	// FIXME maybe scrapes should be an int?
	errorCounter := newFloat64Counter(meter, "errors")

	var sample *Sample
	if len(os.Getenv("PURPLE_AIR_API_KEY")) > 0 {
		fmt.Println("Using PurpleAir API key " + os.Getenv("PURPLE_AIR_API_KEY"))
		sample, err = GetPurpleAirSensorDataFromAPI(
			os.Getenv("PURPLE_AIR_API_KEY"),
			os.Getenv("PURPLE_AIR_SENSOR_ID"),
		)
	} else {
		fmt.Println("Using local PurpleAir host " + os.Getenv("PURPLE_AIR_HOST_NAME"))
		sample, err = GetPurpleAirSensorDataFromSensor(
			os.Getenv("PURPLE_AIR_HOST_NAME"),
			os.Getenv("PURPLE_AIR_SENSOR_NAME"),
		)
	}
	if err != nil {
		errorCounter.Add(ctx, 1)
		_ = provider.ForceFlush(ctx)
		panic(err)
	}

	// FIXME maybe uptime should be an int?
	setNewFloat64Gauge(ctx, meter, sample.SensorName, "uptime", sample.Uptime)
	// FIXME maybe lag should be an int?
	setNewFloat64Gauge(ctx, meter, sample.SensorName, "lag", sample.Lag)
	setNewFloat64Gauge(ctx, meter, sample.SensorName, "temperature", sample.Temperature-temperature_offset)
	setNewFloat64Gauge(ctx, meter, sample.SensorName, "pressure", sample.Pressure)
	setNewFloat64Gauge(ctx, meter, sample.SensorName, "humidity", sample.Humidity)
	setNewFloat64Gauge(ctx, meter, sample.SensorName, "pm25", sample.PM25)
	// FIXME maybe aqi should be an int?
	setNewFloat64Gauge(ctx, meter, sample.SensorName, "aqi", sample.AQI)

	err = provider.ForceFlush(ctx)
	if err != nil {
		panic(err)
	}
}
