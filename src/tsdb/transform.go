package tsdb

import (
	"errors"
	"github.com/futurehomeno/fimpgo"
	influx "github.com/influxdata/influxdb1-client/v2"
	"strconv"
)

// DefaultTransform - transforms IotMsg into InfluxDb datapoint
func DefaultTransform(context *MsgContext, topic string,addr *fimpgo.Address, iotMsg *fimpgo.FimpMessage, domain string) (*influx.Point, error) {
	tags := map[string]string{
		"topic":  topic,
		"location_id":"",
		"service_id":"",
		"dev_id":"",
		"dev_type":"",
	}
	if domain != "-"{
		tags["domain"] =domain
	}

	//log.Debugf("<trans> Tags %+v",tags)
	if context.metadata !=nil {
		tags["location_id"] = strconv.Itoa(context.metadata.LocationID)
		tags["dev_id"] = strconv.Itoa(context.metadata.DeviceID)
		tags["dev_type"] = context.metadata.DeviceType
	}

	var vInt int64
	var err error
	fields := map[string]interface{}{}
	fields["src"] = iotMsg.Source // src can change as several services can generate commands as result the field can't be stored as tag
	valueType := iotMsg.ValueType
	switch iotMsg.Service {
	case "meter_elec" , "sensor_power":
		var mName string
		if iotMsg.Type == "evt.meter.report" {
			val ,err := iotMsg.GetFloatValue()
			unit , _ := iotMsg.Properties["unit"]
			if err == nil {
				if unit == "W" {
					mName = "electricity_meter_power"
				}else if unit == "kWh" {
					mName = "electricity_meter_energy"
				}
				fields["value"] = val
				fields["unit"] = unit
				fields["consumption"] = true
				context.measurementName = mName
				valueType = "_skip_"
			}else {
				return nil,err
			}

		}else if iotMsg.Type == "cmd.meter_ext.get_report" {
			mName = "electricity_meter_extended"
			val , err := iotMsg.GetFloatMapValue()
			// https://github.com/futurehomeno/fimp-api#extended-report-object
			if err != nil {
				return nil,err
			}
			fields["e_import"],_ = val["e_import"]
			fields["e_export"],_ = val["e_export"]
			fields["last_e_export"],_ = val["last_e_export"]
			fields["last_e_import"],_ = val["last_e_import"]
			fields["p_import"],_ = val["p_import"]
			fields["p_import_avg"],_ = val["p_import_avg"]
			fields["p_import_min"],_ = val["p_import_min"]
			fields["p_import_max"],_ = val["p_import_max"]
			fields["p_export"],_ = val["p_export"]
			fields["p_export_min"],_ = val["p_export_min"]
			fields["p_export_max"],_ = val["p_export_max"]
			fields["u1"],_ = val["u1"]
			fields["u2"],_ = val["u2"]
			fields["u3"],_ = val["u3"]
			fields["i1"],_ = val["i1"]
			fields["i2"],_ = val["i2"]
			fields["i3"],_ = val["i3"]
			context.measurementName = mName
			valueType = "_skip_"
		}


	case "thermostat":
		if iotMsg.Type == "cmd.setpoint.set" || iotMsg.Type == "cmd.setpoint.report" {
			tmap ,err := iotMsg.GetStrMapValue()
			if err != nil {
				return nil,err
			}
			unit := "C"
			ttype := "heat"
			tempStr , tok := tmap["temp"]
			unit , _ = tmap["unit"]
			ttype, _ = tmap["type"]
			if !tok {
				if err != nil {
					return nil,errors.New("temp is in wrong format")
				}
			}
			temp ,err := strconv.ParseFloat(tempStr,64)
			if err != nil {
				return nil,err
			}
			fields["value"] = temp
			fields["type"] = ttype
			fields["unit"] = unit
			valueType = "_skip_"
		}
	}

	switch valueType {
	case "float":
		val ,err := iotMsg.GetFloatValue()
		unit , _ := iotMsg.Properties["unit"]
		if err == nil {
			fields["value"] = val
			fields["unit"] = unit
		}

	case "bool":
		val ,err := iotMsg.GetBoolValue()
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
		return nil,errors.New("value type is not defined")
	case "_skip_":

	default:

		fields["value"] = iotMsg.Value
	}
	if fields != nil {
		point, err := influx.NewPoint(context.measurementName, tags, fields, context.time)
		return point, err
	}

	return nil, err

}


