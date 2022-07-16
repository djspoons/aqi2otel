package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/go-logr/stdr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/syncfloat64"
	"go.opentelemetry.io/otel/metric/instrument/syncint64"
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

func main() {
	log.Println("starting main()...")
	ctx := context.Background()

	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		log.Println("Got error: " + err.Error())
	}))
	otel.SetLogger(stdr.New(log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)))

	var exporter export.Exporter
	var err error
	if true {
		log.Println("Using stdout exporter")
		exporter, err = stdoutmetric.New(stdoutmetric.WithPrettyPrint())
	} else {
		if true {
			log.Println("Using OTLP gRPC exporter")
			exporter, err = otlpmetricgrpc.New(ctx,
				otlpmetricgrpc.WithEndpoint("ingest.lightstep.com:443"),
			)
		} else {
			log.Println("Using OTLP HTTP exporter")
			exporter, err = otlpmetrichttp.New(ctx,
				otlpmetrichttp.WithEndpoint("ingest.lightstep.com"),
				otlpmetrichttp.WithURLPath("/metrics/otlp/v0.6"),
			)
		}
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

	data, err := callPurpleAirAPI()
	if err != nil {
		errorCounter.Add(ctx, 1)
		err = provider.Stop(ctx)
		panic(err)
	}
	sensorName := data["sensor"].(map[string]interface{})["name"].(string)
	pm25 := data["sensor"].(map[string]interface{})["pm2.5_atm"].(float64)
	aqi := pm25ToAQI(pm25)
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

func callPurpleAirAPI() (map[string]interface{}, error) {
	url := "https://api.purpleair.com/v1/sensors/" + os.Getenv("PURPLE_AIR_SENSOR_ID")

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", os.Getenv("PURPLE_AIR_API_KEY"))

	res, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	data := make(map[string]interface{})
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

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
			log.Println("Got callback for " + name)
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
			log.Println("Got callback for " + name)
			gauge.Observe(ctx, val, attribute.String("name", sensorName))
		},
	)
	if err != nil {
		panic(err)
	}
}

func pm25ToAQI(pm25 float64) int {
	// See table 5, etc. from:
	// https://www.airnow.gov/sites/default/files/2020-05/aqi-technical-assistance-document-sept2018.pdf
	table := []struct {
		pm25 float64
		aqi  int
	}{
		{0, 0},
		{12.0, 50},
		{12.1, 51},
		{35.4, 100},
		{35.5, 101},
		{55.4, 150},
		{55.5, 151},
		{150.4, 200},
		{150.5, 201},
		{250.4, 300},
		{250.5, 301},
		{350.4, 400},
		{350.5, 401},
		{500.4, 500},
	}
	for i := 0; i < len(table)-1; i++ {
		if pm25 < table[i].pm25 {
			return table[i].aqi
		}
		if pm25 < table[i+1].pm25 {
			x := float64(table[i+1].aqi-table[i].aqi) / (table[i+1].pm25 - table[i].pm25)
			return int(math.Round((pm25-table[i].pm25)*x)) + table[i].aqi
		}
	}
	// No official definition above pm2.5 > 500, but one common convention is
	// to just use pm2.5 concentration.
	return int(math.Round(pm25))
}

// PubSubMessage is the payload of a Pub/Sub event. Please refer to the docs for
// additional information regarding Pub/Sub events.
type PubSubMessage struct {
	Data []byte `json:"data"`
}

// Tick is called by the function invoker
func Tick(ctx context.Context, m PubSubMessage) error {
	log.Println("In Tick: " + string(m.Data))
	main()
	return nil
}
