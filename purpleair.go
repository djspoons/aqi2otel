package aqi2otel

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

// TODO: could return a struct instead of a map

func GetPurpleAirSensorData(apiKey, sensorID string) (map[string]interface{}, error) {
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

	return data, nil
}
