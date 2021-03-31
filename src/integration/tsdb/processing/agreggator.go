package processing

import (
	"github.com/montanaflynn/stats"
	log "github.com/sirupsen/logrus"
	"math"
	"sync"
	"time"
)

const (
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
	Time            time.Time
	Profile         SProfile
}

type SProfile struct {
	HourlyAccumulatedValue bool // device reports value once a hour , at beginning of each hour +/- few minutes. It also means that time has to be adjusted , so it belongs to previous hour.
}

// SValues is a container that stores data point metadata and array of values
type SValues struct {
	values         []float64
	dataPointMeta  DataPoint // Series metadata
	lastReportedAt time.Time
}

func (sv *SValues) TimeSinceLastUpdate() time.Duration {
	return time.Since(sv.lastReportedAt)
}

//DataPointAggregator is in-memory data point aggregator, it stores data points in memory and generate aggregates every X seconds
type DataPointAggregator struct {
	inputChannel         chan DataPoint
	outputChannel        chan DataPoint
	series               map[string]*SValues
	mux                  sync.Mutex
	aggregationInterval  time.Duration
	ticker               *time.Ticker
	diffSamplingInterval int // interval in minutes , default value 10 minutes
	waitingNextInterval  bool
}

func (dpa *DataPointAggregator) OutputChannel() chan DataPoint {
	return dpa.outputChannel
}

// NewDataPointAggregator
//    aggregationInterval - interval interval as duration
//    samplingInterval - sampling interval in minutes
func NewDataPointAggregator(aggregationInterval time.Duration, samplingInterval int) *DataPointAggregator {
	dpa := &DataPointAggregator{aggregationInterval: aggregationInterval, diffSamplingInterval: samplingInterval}
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
			dpa.calculateAndPublishAggregates()
			if math.Mod(float64(time.Now().Minute()), float64(dpa.diffSamplingInterval)) == 0 { // executing this 3 times an hour at exactly the same time. 00 ,
				if dpa.waitingNextInterval {
					log.Debug("---------It's time to run difference calculation--------")
					dpa.waitingNextInterval = false
					dpa.calculateAndPublishAccumulatedAggregates()
				}
			} else {
				dpa.waitingNextInterval = true
			}
		}
	}()
	return dpa
}

//startInputStreamProcessor starts stream processor . The processor consumes messages from input channel into internal buffer/vector with values . Later values are used to calculate aggregates
func (dpa *DataPointAggregator) startInputStreamProcessor() {
	go func() {
		for m := range dpa.inputChannel {
			dpa.mux.Lock()
			ser, exist := dpa.series[m.SeriesID]
			val, ok := m.Value.(float64)
			if !ok {
				dpa.mux.Unlock()
				log.Debug("<aggr> Value is not float . Value = ", m.Value)
				continue
			}
			if exist {
				switch ser.dataPointMeta.AggregationFunc {
				case AggregationFuncMean, AggregationFuncMin, AggregationFuncMax, AggregationFuncDifference:
					ser.values = append(ser.values, val) // accumulating values
				case AggregationFuncLast: // saving last value
					if len(ser.values) == 0 && cap(ser.values) > 0 {
						ser.values = ser.values[:1]
						ser.values[0] = val
					} else {
						ser.values = []float64{val}
					}
				}
				ser.dataPointMeta.Tags = m.Tags
				ser.dataPointMeta.Fields = m.Fields
				ser.dataPointMeta.Profile = m.Profile
				ser.lastReportedAt = time.Now()

			} else {
				m.Value = 0
				dpa.series[m.SeriesID] = &SValues{
					values:         []float64{val},
					dataPointMeta:  m,
					lastReportedAt: time.Now(),
				}
			}
			dpa.mux.Unlock()
		}
	}()
}

func (dpa *DataPointAggregator) AddDataPoint(dp DataPoint) {
	dpa.inputChannel <- dp
}

//calculateAndPublishAggregates - calculates aggregates for all series and sends results over output channel
// for accumulating measurements the process samples data on regular interval and saves into separate table , for instance
// for energy calculations
func (dpa *DataPointAggregator) calculateAndPublishAggregates() {
	dpa.mux.Lock()
	defer dpa.mux.Unlock()
	var result float64
	var err error
	for _, v := range dpa.series {
		result = 0
		if len(v.values) == 0 {
			continue
		}
		switch v.dataPointMeta.AggregationFunc {
		case AggregationFuncMean:
			result, err = stats.Mean(v.values)
			log.Debugf("<aggr> Mean value = %f for series = %s from values %v", result, v.dataPointMeta.SeriesID, v.values)
		case AggregationFuncMin:
			result, err = stats.Min(v.values)
		case AggregationFuncMax:
			result, err = stats.Max(v.values)
		case AggregationFuncLast:
			result = v.values[0]
			log.Debugf("<aggr> LAST value = %f for series = %s from values %v", result, v.dataPointMeta.SeriesID, v.values)
		default:
			continue
		}

		v.dataPointMeta.Time = time.Now()
		v.values = v.values[:0] // making slice empty

		if err != nil {
			continue
		}

		if v.dataPointMeta.Value == result {
			// Publishing only value that have changed
			log.Trace("<aggr> Datapoint didn't change , value skipped. val = ", result)
			continue
		}

		v.dataPointMeta.Value = result
		v.dataPointMeta.Fields["value"] = result
		dpa.outputChannel <- v.dataPointMeta
	}
}

func (dpa *DataPointAggregator) calculateAndPublishAccumulatedAggregates() {
	dpa.mux.Lock()
	defer dpa.mux.Unlock()
	var result float64
	for _, v := range dpa.series {
		result = 0
		if len(v.values) == 0 || v.dataPointMeta.AggregationFunc != AggregationFuncDifference{
			continue
		}

		// Irregular reporting can accumulate too much data over multiple hours. For instance HAN meter was offline for very long time.
		if v.TimeSinceLastUpdate() > 120*time.Minute {
			// TODO : calculate average since last update instead of drooping measurements.
			log.Debug("<aggr> Previous value is too old.")
			v.values = v.values[:0] // empty slice
			continue
		}

		if v.dataPointMeta.Profile.HourlyAccumulatedValue {
			log.Debug("<aggr> Setting time to previous hour")
			v.dataPointMeta.Time = adjustTimeByOneHour(time.Now())
		} else {
			v.dataPointMeta.Time = time.Now()
			v.values = filterSeries(v.values)
		}

		result = dpa.calculateDifference(v.values)
		log.Debugf("<aggr> DIFF value = %f for series = %s from values %v", result, v.dataPointMeta.SeriesID, v.values)
		v.values = v.values[len(v.values)-1:] // making last element as first element in next array

		// TODO: filtering and protection against anomalies must be moved to another place
		if result > 100 { // 100kWh is unrealistic value , can be result of un-regular reports
			log.Info("<aggr> Value is too high , ignored.Value = ",result)
			continue
		}

		if v.dataPointMeta.Profile.HourlyAccumulatedValue {
			if result == 0 {
				continue // Skipping 0 values
			}
		}else {
			if v.dataPointMeta.Value == result || result ==0 {
				log.Debug("<aggr> Datapoint didn't change , value skipped. val = ", result)
				continue // Skipping values that haven't changed or 0 values
			}
		}

		v.dataPointMeta.Value = result
		v.dataPointMeta.Fields["value"] = result
		dpa.outputChannel <- v.dataPointMeta
	}
}

// calculateDifference takes array or values (accumulated and growin) and calculates difference between first and last element.
// at some point value can be reset
func (dpa *DataPointAggregator) calculateDifference(values []float64) float64 {
	var result float64
	vlen := len(values)
	if vlen <= 1 {
		return 0
	}
	if vlen >= 2 {
		// We need to verify that all values are growing . It might happen that meter has reset internal counter.
		// [ 1 , 2 , 3 ]

		for i := 0; i < (vlen - 1); i++ {
			if values[i+1] >= values[i] {
				result += values[i+1] - values[i]
			} else {
				log.Infof("<aggr> Meter reset detected. old = %f , new = %f ",values[i], values[i+1])
			}
		}

	}
	return result
}

func adjustTimeByOneHour(t time.Time) time.Time {
	t = t.Add(-1 * time.Hour)
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 59, 0, 0, t.Location())
}
// filterSeries filtering out all noise
func filterSeries(values []float64) []float64 {

	var result []float64

	outliers, _ := findOutliers(values)

	isOutlier := func(val float64)bool {
		for _,ov := range outliers.Extreme {
			if val == ov {
				return true
			}
		}
		return false
	}

	for i := range values {
		if values[i] == 0  || isOutlier(values[i]){  // removing all 0 and outliers
			log.Info("<aggr> Filtering out outlier val = ",values[i])
			continue
		} else {
			result = append(result,values[i])
		}
	}
	return result
}

func findOutliers(values []float64) (stats.Outliers,error) {
	return stats.QuartileOutliers(values)
}
