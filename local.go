package aqi2otel

// See https://community.purpleair.com/t/local-json-documentation/6917

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

func GetPurpleAirSensorDataFromSensor(host, sensorName string) (*Sample, error) {
	url := "http://" + host + "/json?live=false"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

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

	sample := Sample{}

	sample.SensorName = sensorName
	log.Println("sensorName is " + sample.SensorName)

	hardware := make(map[string]bool)
	hardwareDiscovered := data["hardwarediscovered"].(string)
	log.Println("hardware is " + hardwareDiscovered)
	for _, s := range strings.Split(hardwareDiscovered, "+") {
		hardware[s] = true
	}

	sample.Uptime = data["uptime"].(float64)
	timestamp, err :=
		time.Parse("2006/01/02T15:04:05Z07", strings.ToUpper(data["DateTime"].(string)))
	if err != nil {
		panic(err)
	}
	sample.Lag = float64(time.Now().Unix() - timestamp.Unix())

	sample.Temperature = data["current_temp_f"].(float64)
	sample.Pressure = data["pressure"].(float64)
	sample.Humidity = data["current_humidity"].(float64)

	if hardware["BME68X"] {
		// If we have a 680 unit, then average in these values
		sample.Temperature = (sample.Temperature + data["current_temp_f_680"].(float64)) / 2
		sample.Pressure = (sample.Pressure + data["pressure_680"].(float64)) / 2
		sample.Humidity = (sample.Humidity + data["current_humidity_680"].(float64)) / 2
	}

	// Correct the temperature
	sample.Temperature = sample.Temperature - temperature_offset

	// If, for example, Channel A is broken:
	// sample.PM25 := data["p_2_5_um_b"].(float64)
	sample.PM25 = (data["p_2_5_um"].(float64) + data["p_2_5_um_b"].(float64)) / 2

	sample.AQI = float64(PM25ToAQI(sample.PM25))

	return &sample, nil
}
