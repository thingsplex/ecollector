package tsdb

import (
	"errors"
	"fmt"
	"github.com/futurehomeno/fimpgo"
	influx "github.com/influxdata/influxdb1-client/v2"
	"github.com/thingsplex/ecollector/integration/tsdb/processing"
	"strconv"
)

const (
	MeasurementElecMeterPower  = "electricity_meter_power"
	MeasurementElecMeterEnergy = "electricity_meter_energy"
	DirectionImport            = "import"
	DirectionExport            = "export"
)

// DefaultTransform - transforms IotMsg into InfluxDb datapoint
func DefaultTransform(context *MsgContext, topic string, addr *fimpgo.Address, iotMsg *fimpgo.FimpMessage, domain string) ([]*DataPoint, error) {
	var points []*DataPoint
	tags := getDefaultTags(context, topic, domain)
	var seriesID string
	devId, ok := tags["dev_id"]
	if ok {
		seriesID = devId
	} else {
		seriesID = topic
	}

	var vInt int64
	var err error
	fields := map[string]interface{}{}
	fields["src"] = iotMsg.Source // src can change as several services can generate commands as result the field can't be stored as tag
	valueType := iotMsg.ValueType
	switch iotMsg.Service {
	case "meter_elec", "sensor_power":
		var mName string
		if iotMsg.Type == "evt.meter.report" || iotMsg.Type == "evt.sensor.report" {
			val, err := iotMsg.GetFloatValue()
			unit, _ := iotMsg.Properties["unit"]
			if err == nil {
				if unit == "W" {
					mName = MeasurementElecMeterPower
				} else if unit == "kWh" {
					mName = MeasurementElecMeterEnergy
				} else {
					return nil, fmt.Errorf("unknown unit")
				}
				fields["value"] = val
				fields["unit"] = unit
				tags["dir"] = DirectionImport
				context.measurementName = mName
				valueType = "_skip_"
				seriesID = fmt.Sprintf("%s;import",seriesID)
			} else {
				return nil, err
			}

		} else if iotMsg.Type == "evt.meter_ext.report" {
			val, err := iotMsg.GetFloatMapValue()
			// https://github.com/futurehomeno/fimp-api#extended-report-object
			if err != nil {
				return nil, err
			}
			var eImportOk, eExportOk bool
			fields["e_import"], eImportOk = val["e_import"]
			if eImportOk {
				pTags := getDefaultTags(context, topic, domain)
				pTags["dir"] = DirectionImport
				pFields := map[string]interface{}{"value": fields["e_import"], "unit": "kWh"}
				point, err := influx.NewPoint(MeasurementElecMeterEnergy, pTags, pFields, context.time)
				if err == nil {
					points = append(points, &DataPoint{
						MeasurementName:  MeasurementElecMeterEnergy,
						AggregationValue: fields["e_import"],
						AggregationFunc:  processing.AggregationFuncLast,
						SeriesID:         fmt.Sprintf("%s;%s;import", MeasurementElecMeterEnergy, seriesID),
						Point:            point,
					})
				}
			}
			fields["e_export"], eExportOk = val["e_export"]
			if eExportOk {
				pTags := getDefaultTags(context, topic, domain)
				pTags["dir"] = DirectionExport
				pFields := map[string]interface{}{"value": fields["e_export"], "unit": "kWh"}
				point, err := influx.NewPoint(MeasurementElecMeterEnergy, pTags, pFields, context.time)
				if err == nil {
					points = append(points, &DataPoint{
						MeasurementName:  MeasurementElecMeterEnergy,
						AggregationValue: fields["e_export"],
						AggregationFunc:  processing.AggregationFuncLast,
						SeriesID:         fmt.Sprintf("%s;%s;export", MeasurementElecMeterEnergy, seriesID),
						Point:            point,
					})
				}
			}

			fields["last_e_export"], _ = val["last_e_export"]
			fields["last_e_import"], _ = val["last_e_import"]

			pImport, pImportOk := val["p_import"]
			if pImportOk {
				//fields["p_import"] = pImport
				//fields["p_import_react"], _ = val["p_import_react"]
				//fields["p_import_apparent"], _ = val["p_import_apparent"]
				//fields["p_import_avg"], _ = val["p_import_avg"]
				//fields["p_import_min"], _ = val["p_import_min"]
				//fields["p_import_max"], _ = val["p_import_max"]

				// Creating separate measurement
				pTags := getDefaultTags(context, topic, domain)
				pTags["dir"] = DirectionImport
				//log.Debug("Writing p_import , value = ",pImport)
				pFields := map[string]interface{}{"value": pImport, "unit": "W"}
				point, err := influx.NewPoint(MeasurementElecMeterPower, pTags, pFields, context.time)
				if err == nil {
					points = append(points, &DataPoint{
						MeasurementName:  MeasurementElecMeterPower,
						AggregationValue: pImport,
						AggregationFunc:  processing.AggregationFuncMean,
						SeriesID:         fmt.Sprintf("%s;%s;import", MeasurementElecMeterPower, seriesID),
						Point:            point,
					})
				}
			}

			pExport, pExportOk := val["p_export"]
			if pExportOk {
				//fields["p_export"] = pExport
				//fields["p_export_react"], _ = val["p_export_react"]
				//fields["p_export_min"], _ = val["p_export_min"]
				//fields["p_export_max"], _ = val["p_export_max"]

				// Creating separate measurement
				pTags := getDefaultTags(context, topic, domain)
				pTags["dir"] = DirectionExport
				pFields := map[string]interface{}{"value": pExport, "unit": "W"}
				point, err := influx.NewPoint(MeasurementElecMeterPower, pTags, pFields, context.time)
				if err == nil {
					points = append(points, &DataPoint{
						MeasurementName:  MeasurementElecMeterPower,
						AggregationValue: pExport,
						AggregationFunc:  processing.AggregationFuncMean,
						SeriesID:         fmt.Sprintf("%s;%s;export", MeasurementElecMeterPower, seriesID),
						Point:            point,
					})
				}
			}
			//freq, freqOk := val["freq"]
			//if freqOk {
			//	fields["freq"] = freq
			//	fields["freq_min"], _ = val["freq_min"]
			//	fields["freq_max"], _ = val["freq_max"]
			//}
			//
			//fields["u1"], _ = val["u1"]
			//fields["u2"], _ = val["u2"]
			//fields["u3"], _ = val["u3"]
			//fields["i1"], _ = val["i1"]
			//fields["i2"], _ = val["i2"]
			//fields["i3"], _ = val["i3"]
			//
			//dcp, dcpOk := val["dc_p"]
			//if dcpOk {
			//	fields["dc_p"] = dcp
			//	fields["dc_p_min"], _ = val["dc_p_min"]
			//	fields["dc_p_max"], _ = val["dc_p_max"]
			//}
			//
			//dcu, dcuOk := val["dc_u"]
			//if dcuOk {
			//	fields["dc_u"] = dcu
			//	fields["dc_u_min"], _ = val["dc_u_min"]
			//	fields["dc_u_max"], _ = val["dc_u_max"]
			//}
			//
			//dci, dciOk := val["dc_i"]
			//if dciOk {
			//	fields["dc_i"] = dci
			//	fields["dc_i_min"], _ = val["dc_i_min"]
			//	fields["dc_i_max"], _ = val["dc_i_max"]
			//}
			fields = nil
			//context.measurementName = "electricity_meter_ext"
			valueType = "_skip_"
		}

	case "thermostat":
		if iotMsg.Type == "cmd.setpoint.set" || iotMsg.Type == "cmd.setpoint.report" {
			tmap, err := iotMsg.GetStrMapValue()
			if err != nil {
				return nil, err
			}
			unit := "C"
			ttype := "heat"
			tempStr, tok := tmap["temp"]
			unit, _ = tmap["unit"]
			ttype, _ = tmap["type"]
			if !tok {
				if err != nil {
					return nil, errors.New("temp is in wrong format")
				}
			}
			temp, err := strconv.ParseFloat(tempStr, 64)
			if err != nil {
				return nil, err
			}
			fields["value"] = temp
			fields["type"] = ttype
			fields["unit"] = unit
			valueType = "_skip_"
		}
	}

	switch valueType {
	case "float":
		val, err := iotMsg.GetFloatValue()
		unit, _ := iotMsg.Properties["unit"]
		if err == nil {
			fields["value"] = val
			fields["unit"] = unit
		}

	case "bool":
		val, err := iotMsg.GetBoolValue()
		if err == nil {
			fields["value"] = val
		}

	case "int":
		vInt, err = iotMsg.GetIntValue()
		if err == nil {
			fields["value"] = vInt
		}

	case "string":
		vStr, err := iotMsg.GetStringValue()
		if err == nil {
			fields["value"] = vStr
		}

	case "null":
		fields["value"] = 0
	case "object":
		fields["value"] = "object"
	case "":
		return nil, errors.New("value type is not defined")
	case "_skip_":

	default:

		fields["value"] = iotMsg.Value
	}
	if fields != nil {
		point, err := influx.NewPoint(context.measurementName, tags, fields, context.time)
		if err == nil {
			return append(points, &DataPoint{
				MeasurementName:  context.measurementName,
				AggregationValue: fields["value"],
				AggregationFunc:  processing.AggregationFuncMean,
				SeriesID:         fmt.Sprintf("%s;%s", context.measurementName, seriesID),
				Point:            point,
			}), err
		}
	}

	return points, err

}

func getDefaultTags(context *MsgContext, topic, domain string) map[string]string {
	tags := map[string]string{
		"topic":       topic,
		"location_id": "",
		"service_id":  "",
		"dev_id":      "",
		"dev_type":    "",
	}
	if domain != "-" {
		tags["domain"] = domain
	}
	//log.Debugf("<trans> Tags %+v",tags)
	if context.metadata != nil {
		tags["location_id"] = strconv.Itoa(context.metadata.LocationID)
		tags["dev_id"] = strconv.Itoa(context.metadata.DeviceID)
		tags["dev_type"] = context.metadata.DeviceType
	}
	return tags
}
