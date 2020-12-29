package processing

import (
	"github.com/montanaflynn/stats"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

const (
	AggregationFuncNone = 0
	AggregationFuncMean = 1
	AggregationFuncLast = 2
	AggregationFuncMin  = 3
	AggregationFuncMax  = 4
)

// DataPoint represents measurement data point
type DataPoint struct {
	MeasurementName string
	SeriesID        string
	Tags            map[string]string
	Fields          map[string]interface{}
	AggregationFunc byte
	Value           interface{}
}

// SValues is a container that stores data point metadata and array of values
type SValues struct {
	values    []float64
	dataPoint DataPoint
}

//DataPointAggregator stores data points in memory and generate aggregates every X seconds
type DataPointAggregator struct {
	inputChannel        chan DataPoint
	outputChannel       chan DataPoint
	series              map[string]*SValues
	mux                 sync.Mutex
	aggregationInterval time.Duration
	ticker              *time.Ticker
}

func (dpa *DataPointAggregator) OutputChannel() chan DataPoint {
	return dpa.outputChannel
}

func NewDataPointAggregator(aggregationInterval time.Duration) *DataPointAggregator {
	dpa := &DataPointAggregator{aggregationInterval: aggregationInterval}
	dpa.mux = sync.Mutex{}
	dpa.inputChannel = make(chan DataPoint, 20)
	dpa.outputChannel = make(chan DataPoint, 20)
	dpa.series = map[string]*SValues{}
	dpa.startInputStreamProcessor()

	dpa.ticker = time.NewTicker(dpa.aggregationInterval)
	go func() {
		for _ = range dpa.ticker.C {
			dpa.calculateAggregates()
		}
	}()

	return dpa
}

//startInputStreamProcessor loads data points from input stream into memory store
func (dpa *DataPointAggregator) startInputStreamProcessor() {
	go func() {
		for m := range dpa.inputChannel {
			dpa.mux.Lock()
			ser, exist := dpa.series[m.SeriesID]
			val, ok := m.Value.(float64)
			if !ok {
				dpa.mux.Unlock()
				log.Debug("<aggr> Value is not float")
				continue
			}
			if exist {
				switch ser.dataPoint.AggregationFunc {
				case AggregationFuncMean, AggregationFuncMin, AggregationFuncMax:
					ser.values = append(ser.values, val)
				case AggregationFuncLast:
					if len(ser.values) == 0 && cap(ser.values)>0 {
						ser.values = ser.values[:1]
						ser.values[0] = val
					}else {
						ser.values = []float64{val}
					}
				}
			} else {
				m.Value = 0
				dpa.series[m.SeriesID] = &SValues{
					values:    []float64{val},
					dataPoint: m,
				}
			}
			dpa.mux.Unlock()
		}
	}()
}

func (dpa *DataPointAggregator) AddDataPoint(dp DataPoint) {
	dpa.inputChannel <- dp
}

//calculateAggregates - calculates aggregates for all series and sends results over output channel
func (dpa *DataPointAggregator) calculateAggregates() {
	dpa.mux.Lock()
	defer dpa.mux.Unlock()
	var result float64
	var err error
	for _, v := range dpa.series {
		result = 0
		if len(v.values) == 0 {
			continue
		}
		switch v.dataPoint.AggregationFunc {
		case AggregationFuncMean:
			result, err = stats.Mean(v.values)
			log.Debugf("<aggr> Mean value = %d for series = %s from values %v",result,v.dataPoint.SeriesID,v.values)
		case AggregationFuncMin:
			result, err = stats.Min(v.values)
		case AggregationFuncMax:
			result, err = stats.Min(v.values)
		case AggregationFuncLast:
			result = v.values[0]
			log.Debugf("<aggr> LAST value = %d for series = %s from values %v",result,v.dataPoint.SeriesID,v.values)
		}
		if err != nil {
			continue
		}
		v.values = v.values[:0]
		if v.dataPoint.Value == result {
			// Publishing only value that have changed
			log.Debug("<aggr> Datapoint didn't change , value skipped. val = ",result )
			continue
		}
		v.dataPoint.Value = result
		v.dataPoint.Fields["value"] = result
		dpa.outputChannel <- v.dataPoint
	}
}
