package storage

import (
	"encoding/csv"
	"fmt"
	influx "github.com/influxdata/influxdb1-client/v2"
	log "github.com/sirupsen/logrus"
	"os"
	"path"
	"time"
)

type CsvStorage struct {
	storagePath string
	writer      *csv.Writer
	headers     []string
	bucketFile  *os.File
}

func NewCsvStorage(storagePath string) (DataStorage, error) {
	st := &CsvStorage{storagePath: storagePath}
	st.headers = []string{"name", "time", "dev_id", "dev_type", "dir", "location_id", "service","src", "topic","value","unit"}
	return st, nil
}

func (c *CsvStorage) InitDefaultBuckets() error {
	destFile := fmt.Sprintf("bucket_%s.csv", time.Now().Format("2006_01_02T15_04_05"))
	file := path.Join(c.storagePath, destFile)
	var err error
	c.bucketFile, err = os.Create(file)
	if err != nil {
		log.Error("<csv-store>. Can't create bucket file ", err)
		return err
	}
	c.writer = csv.NewWriter(c.bucketFile)
	return c.writer.Write(c.headers)
}

func (c *CsvStorage) InitSimpleBuckets() {
	c.InitDefaultBuckets()
}

func (c *CsvStorage) RunQuery(query string) (*influx.Response, error) {
	return nil, nil
}

func (c *CsvStorage) GetDataPoints(fieldName, measurement, relativeTime, fromTime, toTime, groupByTime, fillType, dataFunction, transformFunction, groupByTag string, filter DataPointsFilter) *influx.Response {
	return nil
}

func (c *CsvStorage) GetEnergyDataPoints(relativeTime, fromTime, toTime, groupByTime, groupByTag string, filter DataPointsFilter) *influx.Response {
	return nil
}

func (c *CsvStorage) WriteDataPoints(bp influx.BatchPoints) error {
	for _, point := range bp.Points() {
		rec, err := c.dataPointToRecord(point)
		if err != nil {
			continue
		}
		c.writer.Write(rec)
	}
	log.Debugf("<csv-store> writing to csv")
	c.writer.Flush()
	return nil
}

func (c *CsvStorage) dataPointToRecord(dp *influx.Point) ([]string, error) {
	// "name","time","dev_id","dev_type","dir","location_id","service","src","topic","value","unit"

	/*
		{
				"topic":       topic,
				"location_id": "",
				"service_id":  "",
				"dev_id":      "",
				"dev_type":    "",
		}
	*/

	tags := dp.Tags()
	fields, err := dp.Fields()
	if err != nil {
		return nil, err
	}

	var unit, val string

	unitI, ok := fields["unit"]
	if ok {
		unit, _ = unitI.(string)
	}
	valI, ok := fields["value"]
	if ok {
		val = fmt.Sprintf("%v", valI)
	}

	rec := []string{
		dp.Name(),
		fmt.Sprintf("%d",dp.Time().Unix()),
		tags["dev_id"],
		tags["dev_type"],
		tags["dir"],
		tags["location_id"],
		tags["service"],
		tags["src"],
		tags["topic"],
		val,
		unit,

	}
	return rec, nil
}

func (c *CsvStorage) InitDB(name string) error {
	return nil
}

func (c *CsvStorage) DropDB(name string) error {
	return nil
}

func (c *CsvStorage) UpdateRetentionPolicy(name, duration string) error {
	return nil
}

func (c *CsvStorage) AddRetentionPolicy(name, duration string) error {
	return nil
}

func (c *CsvStorage) DeleteRetentionPolicy(name string) error {
	return nil
}

func (c *CsvStorage) AddCQ(name, srcRetentionPolicy, targetRetentionPolicy, time string) error {
	return nil
}

func (c *CsvStorage) DeleteCQ(name string) error {
	return nil
}

func (c *CsvStorage) DeleteMeasurement(name string) error {
	return nil
}

func (c *CsvStorage) GetDbMeasurements() ([]string, error) {
	var result []string
	return result, nil
}

func (c *CsvStorage) GetDbRetentionPolicies() ([]string, error) {
	var result []string
	return result, nil
}

func (c *CsvStorage) Close() {
	c.writer.Flush()
}
