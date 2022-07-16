package main

import (
	"testing"
)

func TestPM25ToAQI(t *testing.T) {
	tests := []struct {
		pm25 float64
		aqi int
	}{
		// As per http://www.sparetheair.com/publications/AQI_Lookup_Table-PM25.pdf
		{-1.0, 0},
		{0.0, 0},
		{1.0, 4},
		{2.0, 8},
		{3.0, 13},
		{4.0, 17},

		{12.0, 50},
		{13.0, 53},

		{35.0, 99},
		{36.0, 102},

		{55.0, 149},
		{56.0, 151},

		{130.0, 189},

		{150.0, 200},
		{151.0, 201},

		{190.0, 240},

		{250.0, 300},
		{251.0, 301},

		{330.0, 380},
		{380.0, 420},
		
		{500.0, 500},

		// As per convention
		{505.0, 505},
	}
	for i := 0; i < len(tests); i++ {
		res := pm25ToAQI(tests[i].pm25)
		if res != tests[i].aqi {
			t.Errorf("When converting %f, expected %d but got %d",
				tests[i].pm25, tests[i].aqi, res)
		}
	}
}
