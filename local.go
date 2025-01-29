package aqi2otel

// See https://community.purpleair.com/t/local-json-documentation/6917

import (
	"encoding/json"
	"fmt"
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
	fmt.Println("Raw temperature is ", sample.Temperature)
	fmt.Println("PA suggested corrected temperature is ", sample.Temperature-temperature_offset)
	// Atmosphere 2024, 15(4), 415; https://doi.org/10.3390/atmos15040415
	temperature_in_C := (sample.Temperature - 32) * 5 / 9
	corrected_temperature_in_C := temperature_in_C/1.07 - 1.6
	sample.Temperature = corrected_temperature_in_C*9/5 + 32
	fmt.Println("Computed corrected temperature is ", sample.Temperature)

	// Correct the humidity
	fmt.Println("Raw humidity is ", sample.Humidity)
	// Atmosphere 2024, 15(4), 415; https://doi.org/10.3390/atmos15040415
	sample.Humidity = sample.Humidity/0.75 - 0.12
	fmt.Println("Computed corrected humidity is ", sample.Humidity)

	cf1 := data["pm2_5_cf_1"].(float64)
	if data["pm2_5_cf_1_b"].(float64) > cf1 {
		cf1 = data["pm2_5_cf_1_b"].(float64)
	}
	if cf1 < 343 {
		sample.PM25 = 0.52*cf1 - 0.086*sample.Humidity + 5.75
	} else {
		sample.PM25 = 0.46*cf1 + 3.83e-4*cf1*cf1 + 2.97
	}

	//sample.AQI = float64(PM25ToAQI(sample.PM25))
	fmt.Println("Computed corrected AQI is ", PM25ToAQI(sample.PM25))
	fmt.Println("Using device reported AQI = ", data["pm2.5_aqi"].(float64))
	sample.AQI = data["pm2.5_aqi"].(float64)

	return &sample, nil
}
