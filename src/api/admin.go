package api

import (
	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
	"github.com/thingsplex/ecollector/model"
	"github.com/thingsplex/ecollector/tsdb"
)

type AdminApi struct {
	integr *tsdb.Integration
	mqt *fimpgo.MqttTransport
	configs model.Configs
}

func NewAdminApi(integr *tsdb.Integration,configs model.Configs) *AdminApi {
	return &AdminApi{integr: integr,configs:configs}
}

func(api *AdminApi) Start() {
	api.mqt = fimpgo.NewMqttTransport(api.configs.MqttServerURI,api.configs.MqttClientIdPrefix,api.configs.MqttUsername,api.configs.MqttPassword,true,1,1)
	err := api.mqt.Start()
	if err != nil {
		log.Error("Can't connect AdminAPI to broker. Error:", err.Error())
	} else {
		log.Info("Admin api Connected")
	}
	api.mqt.SetMessageHandler(api.onCommand)
	adr := fimpgo.Address{MsgType: fimpgo.MsgTypeCmd, ResourceType: fimpgo.ResourceTypeApp, ResourceName: "ecollector", ResourceAddress: "1"}
	api.mqt.Subscribe(adr.Serialize())

}

func(api *AdminApi) onCommand(topic string, addr *fimpgo.Address, iotMsg *fimpgo.FimpMessage,rawMessage []byte){
	//TODO : Run in it's own goroutine
	if iotMsg.Service != "ecollector" {
		return
	}
	var msg *fimpgo.FimpMessage
	var adr fimpgo.Address
	switch iotMsg.Type {
	case "cmd.ecprocess.get_list":

	case "cmd.ecprocess.get":

	case "cmd.ecprocess.ctrl":

	case "cmd.ecprocess.reset_to_default":

	case "cmd.ecprocess.save_selector":

	case "cmd.ecprocess.save_filter":

	case "cmd.ecprocess.save_measurement":

	case "cmd.ecprocess.delete_measurement":

	case "cmd.ecprocess.delete_selector":

	case "cmd.ecprocess.delete_filter":

	case "cmd.tsdb.query":
		val,err := iotMsg.GetStrMapValue()
		if err != nil {
			log.Debug(" Wrong value format for cmd.influxdb.query")
			return
		}
		//procId , ok1 := val["proc_id"]
		query  , _ := val["query"]
		proc := api.integr.GetProcessByID(1)
		response := proc.RunQuery(query)
		msg = fimpgo.NewMessage("evt.tsdb.query_report", "ecollector", fimpgo.VTypeObject, response, nil, nil,iotMsg)

	case "cmd.tsdb.get_measurements":
		//val,err := iotMsg.GetStrMapValue()
		//if err != nil {
		//	log.Debug(" Wrong value format for cmd.influxdb.query")
		//	return
		//}
		//procId , ok1 := val["proc_id"]
		proc := api.integr.GetProcessByID(1)
		response := proc.GetDbMeasurements()

		msg = fimpgo.NewMessage("evt.tsdb.measurements_report", "ecollector", fimpgo.VTypeStrArray, response, nil, nil,iotMsg)
	}
	if msg == nil {
		return
	}
	if iotMsg.ResponseToTopic != "" {
		fimpBin , _ := msg.SerializeToJson()
		if fimpBin != nil {
			api.mqt.PublishRaw(iotMsg.ResponseToTopic,fimpBin)
		}
	}else {
		adr = fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeApp, ResourceName: "ecollector", ResourceAddress: "1"}
		api.mqt.Publish(&adr,msg)
	}

}


