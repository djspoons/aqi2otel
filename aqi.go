package aqi2otel

import (
	"math"
)

func PM25ToAQI(pm25 float64) int {
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
	// to just use pm2.5 concentration as the AQI.
	return int(math.Round(pm25))
}
