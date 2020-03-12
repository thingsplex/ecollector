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
		"domain": domain,
		"location_id":"",
		"service_id":"",
		"dev_id":"",
		"dev_type":"",
	}

	//log.Debugf("<trans> Tags %+v",tags)
	if context.metadata !=nil {
		tags["location_id"] = strconv.Itoa(context.metadata.LocationID)
		tags["dev_id"] = strconv.Itoa(context.metadata.DeviceID)
	}

	var vInt int64
	var err error
	fields := map[string]interface{}{}
	fields["src"] = iotMsg.Source // src can change as several services can generate commands as result the field can't be stored as tag
	valueType := iotMsg.ValueType
	switch iotMsg.Service {
	case "meter_elec" , "sensor_power":
		val ,err := iotMsg.GetFloatValue()
		unit , _ := iotMsg.Properties["unit"]
		var mName string
		if err == nil {
			if unit == "W" {
				mName = "electricity_meter_power"
			}else if unit == "kWh" {
				mName = "electricity_meter_energy"
			}
			fields["value"] = val
			fields["unit"] = unit
			fields["consumption"] = true

		}
		context.measurementName = mName
		valueType = "_skip_"
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


