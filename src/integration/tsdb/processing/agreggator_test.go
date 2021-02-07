package processing

import (
	log "github.com/sirupsen/logrus"
	"testing"
	"time"
)

func Setup() {
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	log.SetLevel(log.DebugLevel)

}

func TestDataPointAggregator_AddDataPoint(t *testing.T) {
	Setup()
	da := NewDataPointAggregator(5 * time.Second,1)

	go func() {
		var i float64
		for i=10;i<1000;i++ {
			dp := DataPoint{
				MeasurementName: "meter_elec",
				SeriesID:        "meter_1",
				Tags:            map[string]string{"device_id":"111"},
				Fields:          map[string]interface{}{"value":i},
				AggregationFunc: AggregationFuncMean,
				Value:           i,
			}
			time.Sleep(500*time.Millisecond)
			da.AddDataPoint(dp)
			t.Log("DP-1 added")
		}
	}()

	go func() {
		var i float64
		for i=10;i<1000;i++ {
			dp := DataPoint{
				MeasurementName: "meter_elec",
				SeriesID:        "meter_2",
				Tags:            map[string]string{"device_id":"111"},
				Fields:          map[string]interface{}{"value":i},
				AggregationFunc: AggregationFuncLast,
				Value:           i,
			}
			time.Sleep(500*time.Millisecond)
			da.AddDataPoint(dp)
			t.Log("DP-2 added")
		}
	}()

	outChan := da.OutputChannel()
	var i int
	for dp := range outChan {
		t.Logf("New Serias ID = %s , values = %d",dp.SeriesID,dp.Value)
		i++
		if i > 900 {
			break
		}
	}

}

func TestDataPointAggregator_calculateDifference(t *testing.T) {
	da := NewDataPointAggregator(5 * time.Second,1)
	vals := []float64{10,12,14,16,40}
	r := da.calculateDifference(vals)
	if r != 30 {
		t.Error("Error 1 , result = ",r)
	}

	vals2 := []float64{10,12,10,20,30}
	r2 := da.calculateDifference(vals2)
	if r2 != 22 {
		t.Error("Error 2 , result = ",r2)
	}
	vals3 := []float64{10}
	r3 := da.calculateDifference(vals3)
	if r3 != 0 {
		t.Error("Error 3 , result = ",r3)
	}

	vals4 := []float64{}
	r4 := da.calculateDifference(vals4)
	if r4 != 0 {
		t.Error("Error 3 , result = ",r4)
	}

}