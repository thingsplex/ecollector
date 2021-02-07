package processing

import (
	"fmt"
	"github.com/montanaflynn/stats"
	log "github.com/sirupsen/logrus"
	"math"
	"sync"
	"time"
)

const (
	AggregationFuncNone       = 0
	AggregationFuncMean       = 1
	AggregationFuncLast       = 2
	AggregationFuncMin        = 3
	AggregationFuncMax        = 4
	AggregationFuncDifference = 5
)

// DataPoint represents data point metadata
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

//DataPointAggregator is in-memory data point aggregator, it stores data points in memory and generate aggregates every X seconds
type DataPointAggregator struct {
	inputChannel        chan DataPoint
	outputChannel       chan DataPoint
	series              map[string]*SValues
	mux                 sync.Mutex
	aggregationInterval time.Duration
	ticker              *time.Ticker
	diffSamplingInterval int  // interval in minutes
	waitingNextInterval bool
}

func (dpa *DataPointAggregator) OutputChannel() chan DataPoint {
	return dpa.outputChannel
}

// NewDataPointAggregator
//    aggregationInterval - interval interval as duration
//    samplingInterval - sampling interval in minutes
func NewDataPointAggregator(aggregationInterval time.Duration,samplingInterval int) *DataPointAggregator {
	dpa := &DataPointAggregator{aggregationInterval: aggregationInterval,diffSamplingInterval: samplingInterval}
	dpa.mux = sync.Mutex{}
	dpa.inputChannel = make(chan DataPoint, 20)
	dpa.outputChannel = make(chan DataPoint, 20)
	dpa.series = map[string]*SValues{}
	dpa.startInputStreamProcessor()
	dpa.ticker = time.NewTicker(dpa.aggregationInterval)
	if dpa.diffSamplingInterval == 0 {
		dpa.diffSamplingInterval = 20
	}
	dpa.waitingNextInterval = true
	go func() {
		for _ = range dpa.ticker.C {
			dpa.calculateAggregates(AggregationFuncNone)
			if math.Mod(float64(time.Now().Minute()),float64(dpa.diffSamplingInterval)) == 0  { // executing this 3 times an hour at exactly the same time. 00 ,
				if dpa.waitingNextInterval {
					log.Debug("---------It's time to run difference calculation--------")
					dpa.waitingNextInterval = false
					dpa.calculateAggregates(AggregationFuncDifference)
				}
			}else {
				dpa.waitingNextInterval = true
			}
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
				log.Debug("<aggr> Value is not float . Value = ",m.Value)
				continue
			}
			if exist {
				switch ser.dataPoint.AggregationFunc {
				case AggregationFuncMean, AggregationFuncMin, AggregationFuncMax, AggregationFuncDifference:
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
// for accumulating measurements the process samples data on regular interval and saves into separate table , for instance
// for energy calculation
func (dpa *DataPointAggregator) calculateAggregates(filterFunc int) {
	dpa.mux.Lock()
	defer dpa.mux.Unlock()
	var result float64
	var err error
	for _, v := range dpa.series {
		result = 0
		if len(v.values) == 0 {
			continue
		}
		deleteAllValues := true
		switch v.dataPoint.AggregationFunc {
		case AggregationFuncMean:
			result, err = stats.Mean(v.values)
			log.Debugf("<aggr> Mean value = %f for series = %s from values %v",result,v.dataPoint.SeriesID,v.values)
		case AggregationFuncMin:
			result, err = stats.Min(v.values)
		case AggregationFuncMax:
			result, err = stats.Max(v.values)
		case AggregationFuncLast:
			result = v.values[0]
			log.Debugf("<aggr> LAST value = %f for series = %s from values %v",result,v.dataPoint.SeriesID,v.values)
		case AggregationFuncDifference :
			if filterFunc != AggregationFuncDifference {
				continue
			}
			result = dpa.calculateDifference(v.values)
			log.Debugf("<aggr> DIFF value = %f for series = %s from values %v",result,v.dataPoint.SeriesID,v.values)

			// TODO: filtering and protection against anomalies must be moved to another place
			if result > 100  { // 100kWh is unrealistic value .
 				log.Debug("<aggr> Value is too high , ignored.")
				v.values = v.values[:0]
				err = fmt.Errorf("aggr value too high , skipped")
			}else {
				v.values = v.values[len(v.values)-1:] // making last element as first element in next array
				deleteAllValues = false
			}

		}

		if deleteAllValues {
			v.values = v.values[:0] // making slice empty
		}

		if err != nil {
			continue
		}

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

// calculateDifference takes array or values (accumulated and growin) and calculates difference between first and last element. 
// at some point value can be reset 
func (dpa *DataPointAggregator) calculateDifference(values []float64) float64 {
	var result float64
	vlen := len(values)
	log.Debug("<aggr> Calculating difference")
	if vlen <= 1 {
		return 0
	}
	if vlen > 2 {
		// We need to verify that all values are growing . It might happen that meter has reset internal counter.
		prevVal := values[0]
		for i := range values {
			if values[i]>=prevVal {
				result += values[i] - prevVal
			}
			prevVal = values[i]
		}
	}
	return result
}