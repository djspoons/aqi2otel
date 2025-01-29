package aqi2otel

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

func GetPurpleAirSensorDataFromAPI(apiKey, sensorID string) (*Sample, error) {
	url := "https://api.purpleair.com/v1/sensors/" + sensorID

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", apiKey)

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

	sample.SensorName = data["sensor"].(map[string]interface{})["name"].(string)
	log.Println("sensorName is " + sample.SensorName)

	sample.Uptime = data["sensor"].(map[string]interface{})["uptime"].(float64)
	sample.Lag =
		float64(time.Now().Unix()) -
			data["sensor"].(map[string]interface{})["stats"].(map[string]interface{})["time_stamp"].(float64)
	// TODO update corrections with new methods
	sample.Temperature =
		data["sensor"].(map[string]interface{})["temperature"].(float64) - temperature_offset
	sample.Pressure =
		data["sensor"].(map[string]interface{})["pressure"].(float64)
	sample.Humidity =
		data["sensor"].(map[string]interface{})["humidity"].(float64)

	// Sometimes Channel A is broken, for example:
	//sample.PM25 = data["sensor"].(map[string]interface{})["pm2.5_atm_b"].(float64)
	sample.PM25 = (data["sensor"].(map[string]interface{})["pm2.5_atm_a"].(float64) + data["sensor"].(map[string]interface{})["pm2.5_atm_b"].(float64)) / 2

	sample.AQI = float64(PM25ToAQI(sample.PM25))

	return &sample, nil
}
