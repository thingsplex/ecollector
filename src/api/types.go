package api

import (
	"github.com/thingsplex/ecollector/integration/tsdb"
	"github.com/thingsplex/ecollector/integration/tsdb/storage"
)

type GetDataPointsRequest struct {
	ProcID            tsdb.IDt                 `json:"proc_id"`
	FieldName         string                   `json:"field_name"`
	DataFunction      string                   `json:"data_function"`
	TransformFunction string                   `json:"transform_function"`
	MeasurementName   string                   `json:"measurement_name"`
	RelativeTime      string                   `json:"relative_time"`
	FromTime          string                   `json:"from_time"`
	ToTime            string                   `json:"to_time"`
	GroupByTime       string                   `json:"group_by_time"`
	GroupByTag        string                   `json:"group_by_tag"`
	FillType          string                   `json:"fill_type"`
	Filters           storage.DataPointsFilter `json:"filters"`
}

type MDataPoint struct {
	Name      string                 `json:"name"` // name of the measurement
	Tags      map[string]string      `json:"tags"`
	Fields    map[string]interface{} `json:"fields"`
	TimeStamp int64                  `json:"ts"` // if 0 , ecollector will set local time
}

type WriteDataPointsRequest struct {
	ProcID     tsdb.IDt     `json:"proc_id"`
	Bucket     string       `json:"bucket"` // data storage with retention policy , if not set system will try to auto calculate based on measurement name
	DataPoints []MDataPoint `json:"dp"`
}
