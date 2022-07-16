package aqi2otel

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/go-logr/stdr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	"go.opentelemetry.io/otel/sdk/metric/export"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	selector "go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/semconv/v1.10.0"
)

// Per https://api.purpleair.com/#api-sensors-get-sensor-data, temperature
// inside the housing is "8F higher than ambient conditions."
const temperature_offset = 8

func Run(ctx context.Context, useStdoutExporter bool) {
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		log.Println("Got error: " + err.Error())
	}))
	otel.SetLogger(stdr.New(log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)))

	var exporter export.Exporter
	var err error
	if useStdoutExporter {
		log.Println("Using stdout exporter")
		exporter, err = stdoutmetric.New(stdoutmetric.WithPrettyPrint())
	} else {
		log.Println("Using OTLP gRPC exporter")
		exporter, err = otlpmetricgrpc.New(ctx,
			otlpmetricgrpc.WithEndpoint("ingest.lightstep.com:443"),
		)
	}
	if err != nil {
		panic(err)
	}

	factory := processor.NewFactory(
		selector.NewWithHistogramDistribution(),
		exporter,
	)

	provider := controller.New(factory,
		controller.WithExporter(exporter),
		controller.WithCollectPeriod(5*time.Second),
		controller.WithResource(
			resource.NewSchemaless(
				semconv.ServiceNameKey.String("AQI"),
				attribute.String("sensor_id", os.Getenv("PURPLE_AIR_SENSOR_ID")),
			),
		),
	)
	provider.Start(ctx)
	meter := provider.Meter("PurpleAir")

	// FIXME maybe scrapes should be an int?
	scrapeCounter := newFloat64Counter(meter, "scrapes")
	scrapeCounter.Add(ctx, 1)

	// FIXME maybe scrapes should be an int?
	errorCounter := newFloat64Counter(meter, "errors")

	data, err := GetPurpleAirSensorData(
		os.Getenv("PURPLE_AIR_API_KEY"),
		os.Getenv("PURPLE_AIR_SENSOR_ID"),
	)
	if err != nil {
		errorCounter.Add(ctx, 1)
		err = provider.Stop(ctx)
		panic(err)
	}
	sensorName := data["sensor"].(map[string]interface{})["name"].(string)
	pm25 := data["sensor"].(map[string]interface{})["pm2.5_atm"].(float64)
	aqi := PM25ToAQI(pm25)
	log.Println("sensorName is " + sensorName)

	// FIXME maybe uptime should be an int?
	setNewFloat64Gauge(ctx, meter, sensorName, "uptime",
		data["sensor"].(map[string]interface{})["uptime"].(float64))
	// FIXME maybe uptime should be an int?
	setNewFloat64Gauge(ctx, meter, sensorName, "lag",
		float64(time.Now().Unix())-
			data["sensor"].(map[string]interface{})["stats"].(map[string]interface{})["time_stamp"].(float64))
	setNewFloat64Gauge(ctx, meter, sensorName, "temperature",
		data["sensor"].(map[string]interface{})["temperature"].(float64)-temperature_offset)
	setNewFloat64Gauge(ctx, meter, sensorName, "pressure",
		data["sensor"].(map[string]interface{})["pressure"].(float64))
	setNewFloat64Gauge(ctx, meter, sensorName, "humidity",
		data["sensor"].(map[string]interface{})["humidity"].(float64))
	setNewFloat64Gauge(ctx, meter, sensorName, "pm25",
		data["sensor"].(map[string]interface{})["pm2.5_atm"].(float64))
	// FIXME maybe aqi should be an int?
	setNewFloat64Gauge(ctx, meter, sensorName, "aqi", float64(aqi))

	log.Println("calling Stop()")
	err = provider.Stop(ctx)
	if err != nil {
		panic(err)
	}
}
