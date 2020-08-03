package api

import "github.com/thingsplex/ecollector/integration/tsdb"

type GetDataPointsRequest struct {
	ProcID            tsdb.IDt `json:"proc_id"`
	FieldName         string   `json:"field_name"`
	DataFunction      string   `json:"data_function"`
	TransformFunction string   `json:"transform_function"`
	MeasurementName   string   `json:"measurement_name"`
	RelativeTime      string   `json:"relative_time"`
	FromTime          string   `json:"from_time"`
	ToTime            string   `json:"to_time"`
	GroupByTime       string   `json:"group_by_time"`
	GroupByTag        string   `json:"group_by_tag"`
	FillType          string   `json:"fill_type"`
}
