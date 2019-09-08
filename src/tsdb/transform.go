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
		"thing_id":"",
	}

	//log.Debugf("<trans> Tags %+v",tags)
	if context.metadata !=nil {
		tags["location_id"] = strconv.Itoa(context.metadata.LocationID)
		tags["service_id"] = strconv.Itoa(context.metadata.ServiceID)
		tags["thing_id"] = strconv.Itoa(context.metadata.ThingID)
	}

	var fields map[string]interface{}
	var vInt int64
	var err error

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
			fields = map[string]interface{}{
				"value": val,
				"unit":  unit,
				"consumption":true,
			}
		}
		context.measurementName = mName
		valueType = "_skip_"
	}

	switch valueType {
	case "float":
		val ,err := iotMsg.GetFloatValue()
		unit , _ := iotMsg.Properties["unit"]
		if err == nil {
			fields = map[string]interface{}{
				"value": val,
				"unit":  unit,
			}
		}

	case "bool":
		val ,err := iotMsg.GetBoolValue()
		if err == nil {
			fields = map[string]interface{}{
				"value": val,
			}
		}

	case "int":
		vInt, err = iotMsg.GetIntValue()
		if err == nil {
			fields = map[string]interface{}{
				"value": vInt,
			}
		}

	case "string":
		vStr, err := iotMsg.GetStringValue()
		if err == nil {
			fields = map[string]interface{}{
				"value": vStr,
			}
		}

	case "null":
		fields = map[string]interface{}{
			"value": 0,
		}
	case "object":
		fields = map[string]interface{}{
			"value": "object",
		}
	case "":
		return nil,errors.New("value type is not defined")
	case "_skip_":

	default:
		fields = map[string]interface{}{
			"value": iotMsg.Value,
		}
	}
	if fields != nil {
		point, err := influx.NewPoint(context.measurementName, tags, fields, context.time)
		return point, err
	}

	return nil, err

}

//func GetVincDeviceByFimpAddress(vincDb *vincInfra.HubData, addr *fimpgo.Address) *vincModel.Device {
//	node,_ := utils.GetNodeAndEndpoint(addr.ServiceAddress)
//	adapter := addr.ResourceName
//	if adapter == "zw" {
//		adapter = "zwave-ad"
//	}
//	for i := range vincDb.Device {
//		if vincDb.Device[i].Fimp.Adapter == adapter && vincDb.Device[i].Fimp.Address == node {
//			return &vincDb.Device[i]
//		}
//	}
//	return nil
//}
//
//func GetVincDeviceByFimpDevAddress(vincDb *vincInfra.HubData, adapter,node string ) *vincModel.Device {
//	if adapter == "zw" {
//		adapter = "zwave-ad"
//	}
//	for i := range vincDb.Device {
//		if vincDb.Device[i].Fimp.Adapter == adapter && vincDb.Device[i].Fimp.Address == node {
//			return &vincDb.Device[i]
//		}
//	}
//	return nil
//}
//
//
//func GetVincRoomById(vincDb *vincInfra.HubData, roomId int) *vincModel.Room {
//	for i := range vincDb.Room {
//		if vincDb.Room[i].ID == roomId {
//			return &vincDb.Room[i]
//		}
//	}
//	return nil
//}
//
//func GetVincAreaById(vincDb *vincInfra.HubData, areaId int) *vincModel.Area {
//	for i := range vincDb.Area {
//		if vincDb.Area[i].ID == areaId {
//			return &vincDb.Area[i]
//		}
//	}
//	return nil
//}

