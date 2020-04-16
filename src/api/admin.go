package api

import (
	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
	"github.com/thingsplex/ecollector/model"
	"github.com/thingsplex/ecollector/tsdb"
	"os"
	"strconv"
)

type AdminApi struct {
	integr *tsdb.Integration
	mqt *fimpgo.MqttTransport
	configs *model.Configs
}

func NewAdminApi(integr *tsdb.Integration,configs *model.Configs) *AdminApi {
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
		response := api.integr.Processes()
		msg = fimpgo.NewMessage("evt.ecprocess.proc_list_report", "ecollector", fimpgo.VTypeObject, response, nil, nil,iotMsg)
	case "cmd.ecprocess.get":

	case "cmd.ecprocess.ctrl":
		val,err := iotMsg.GetStrMapValue()
		if err != nil {
			log.Debug(" Wrong value format for cmd.ecprocess.ctrl")
			return
		}
		procIdStr , _ := val["proc_id"]
		op  , _ := val["operation"]
		status := "error"
		if procIdStr != "" && op != "" {
			procId,err  := strconv.Atoi(procIdStr)
			if err == nil {
				proc := api.integr.GetProcessByID(tsdb.IDt(procId))
				switch op {
				case "start":
					err = proc.Start()
				case "stop":
					err = proc.Stop()
				}
				if err == nil {
					status = "ok"
				}
			}
		}
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		response := map[string]string{"op":op,"status":status,"error":errStr}
		msg = fimpgo.NewStrMapMessage("evt.ecprocess.ctrl_report", "ecollector", response, nil, nil,iotMsg)

	case "cmd.ecprocess.save_selector":

	case "cmd.ecprocess.save_filter":

	case "cmd.ecprocess.save_measurement":

	case "cmd.ecprocess.delete_measurement":

	case "cmd.ecprocess.delete_selector":

	case "cmd.ecprocess.delete_filter":

	case "cmd.ecprocess.reset_to_default":
		api.configs.LoadDefaults()
		api.integr.ResetConfigsToDefault()
		os.Exit(0)

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

	case "cmd.tsdb.get_retention_policies":
		//val,err := iotMsg.GetStrMapValue()
		//if err != nil {
		//	log.Debug(" Wrong value format for cmd.influxdb.query")
		//	return
		//}
		//procId , ok1 := val["proc_id"]


		val,err := iotMsg.GetStrMapValue()
		if err != nil {
			log.Debug(" Wrong value format for cmd.ecprocess.ctrl")
			return
		}
		procIdStr , _ := val["proc_id"]
		var response []string
		if procIdStr != ""{
			procId,err  := strconv.Atoi(procIdStr)
			if err == nil {
				proc := api.integr.GetProcessByID(tsdb.IDt(procId))
				response = proc.GetDbRetentionPolicies()
				}
		}
		msg = fimpgo.NewMessage("evt.tsdb.retention_policies", "ecollector", fimpgo.VTypeStrArray, response, nil, nil,iotMsg)

	case "cmd.tsdb.add_retention_policy":
		// configure retentions
		val,err := iotMsg.GetStrMapValue()
		if err != nil {
			log.Debug(" Wrong value format for cmd.influxdb.query")
			return
		}
		name  , _ := val["name"]
		duration  , _ := val["duration"]
		proc := api.integr.GetProcessByID(1)
		proc.UpdateRetentionPolicy(name,duration)

	case "cmd.tsdb.update_retention_policy":
		// configure retentions
		val,err := iotMsg.GetStrMapValue()
		if err != nil {
			log.Debug(" Wrong value format for cmd.influxdb.query")
			return
		}
		name  , _ := val["name"]
		duration  , _ := val["duration"]
		proc := api.integr.GetProcessByID(1)
		proc.UpdateRetentionPolicy(name,duration)

	case "cmd.tsdb.delete_object":
		// configure retentions
		val,err := iotMsg.GetStrMapValue()
		if err != nil {
			log.Debug(" Wrong value format for cmd.influxdb.query")
			return
		}
		name  , _ := val["name"]
		otype  , _ := val["object_type"]

		proc := api.integr.GetProcessByID(1)
		switch otype {
		case "retention_policy":
			proc.Stop()
			proc.DeleteRetentionPolicy(name)
			proc.Start()
		case "measurement":
			proc.DeleteMeasurement(name)
		}
		response := map[string]string{"status":"ok","error":""}
		msg = fimpgo.NewStrMapMessage("evt.tsdb.delete_object_report", "ecollector", response, nil, nil,iotMsg)
		// set default retention policy

	case "cmd.tsdb.get_configs":
		//
	case "cmd.log.set_level":
		// Configure log level
		level , err :=iotMsg.GetStringValue()
		if err != nil {
			return
		}
		logLevel, err := log.ParseLevel(level)
		if err == nil {
			log.SetLevel(logLevel)
			api.configs.LogLevel = level
			api.configs.SaveToFile()
		}
		log.Info("Log level updated to = ",logLevel)
	}
	if msg == nil {
		return
	}
	if iotMsg.ResponseToTopic != "" {
		api.mqt.RespondToRequest(iotMsg,msg)
	}else {
		adr = fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeApp, ResourceName: "ecollector", ResourceAddress: "1"}
		api.mqt.Publish(&adr,msg)
	}

}


