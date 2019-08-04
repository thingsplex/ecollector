package tsdb

import (
	"github.com/futurehomeno/fimpgo"
	influx "github.com/influxdata/influxdb1-client/v2"
)

// DefaultTransform - transforms IotMsg into InfluxDb datapoint
func DefaultTransform(context *MsgContext, topic string,addr *fimpgo.Address, iotMsg *fimpgo.FimpMessage, domain string) (*influx.Point, error) {
	tags := map[string]string{
		"topic":  topic,
		"domain": domain,
		//"mtype":    iotMsg.Type,
		//"serv": iotMsg.Service,
		//"location_id":"",
		//"location_alias":"",
		//"service_id":"",
		//"service_alias":"",
		//"area_id":"",
		//"area_alias":"",
		//"area_type":"",
	}
	//if context.vincDb != nil {
	//	dev := GetVincDeviceByFimpAddress(context.vincDb,addr)
	//	if dev != nil {
	//		room := GetVincRoomById(context.vincDb,dev.Room)
	//		if room != nil {
	//			tags["location_id"] = strconv.Itoa(room.ID)
	//			tags["location_alias"] = room.Client.Name
	//			tags["service_id"] = strconv.Itoa(int(dev.ID))
	//			tags["service_alias"] = dev.Client.Name
	//			area := GetVincAreaById(context.vincDb,room.Area)
	//			if area != nil {
	//				tags["area_id"] = strconv.Itoa(int(area.ID))
	//				tags["area_alias"] = area.Name
	//				tags["area_type"] = area.Type
	//			}
	//		}
	//	}
	//}
	//log.Debugf("<trans> Tags %+v",tags)
	var fields map[string]interface{}
	var vInt int64
	var err error
	switch iotMsg.ValueType {
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

