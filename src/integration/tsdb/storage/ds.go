package storage

import "github.com/influxdata/influxdb1-client/v2"

type DataStorage interface {
	InitDefaultBuckets() error
	InitSimpleBuckets()
	RunQuery(query string) (*client.Response, error)
	GetDataPoints(fieldName, measurement, relativeTime, fromTime, toTime, groupByTime, fillType, dataFunction, transformFunction, groupByTag string, filter DataPointsFilter) *client.Response
	GetEnergyDataPoints(relativeTime, fromTime, toTime, groupByTime, groupByTag string, filter DataPointsFilter) *client.Response
	WriteDataPoints(bp client.BatchPoints) error
	InitDB(name string) error
	DropDB(name string) error
	UpdateRetentionPolicy(name, duration string) error
	AddRetentionPolicy(name, duration string) error
	DeleteRetentionPolicy(name string) error
	AddCQ(name, srcRetentionPolicy, targetRetentionPolicy, time string) error
	DeleteCQ(name string) error
	DeleteMeasurement(name string) error
	GetDbMeasurements() ([]string, error)
	GetDbRetentionPolicies() ([]string, error)
	Close()
}
