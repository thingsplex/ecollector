package api

import (
	"fmt"
	"github.com/futurehomeno/fimpgo"
	influx "github.com/influxdata/influxdb1-client/v2"
	log "github.com/sirupsen/logrus"
	"github.com/thingsplex/ecollector/integration/tsdb"
	"github.com/thingsplex/ecollector/integration/tsdb/storage"
	"github.com/thingsplex/ecollector/model"
	"github.com/thingsplex/ecollector/utils"
	"os"
	"runtime/debug"
	"strconv"
	"time"
)

type AdminApi struct {
	integr  *tsdb.Integration
	mqt     *fimpgo.MqttTransport
	configs *model.Configs
}

func NewAdminApi(integr *tsdb.Integration, configs *model.Configs) *AdminApi {
	return &AdminApi{integr: integr, configs: configs}
}

func (api *AdminApi) Start() {
	mqttClientId := fmt.Sprintf("ec_api_%d", utils.GenerateRandomNumber())
	api.mqt = fimpgo.NewMqttTransport(api.configs.MqttServerURI, mqttClientId, api.configs.MqttUsername, api.configs.MqttPassword, true, 1, 1)
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

func (api *AdminApi) getProcID(val map[string]string) tsdb.IDt {
	procIdStr, ok := val["proc_id"]
	if !ok {
		return -1
	}
	procId, err := strconv.Atoi(procIdStr)
	if err == nil {
		return tsdb.IDt(procId)
	}
	return -1
}

func (api *AdminApi) getStorageByProcId() {

}

func (api *AdminApi) onCommand(topic string, addr *fimpgo.Address, iotMsg *fimpgo.FimpMessage, rawMessage []byte) {
	if iotMsg.Service != "ecollector" {
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error("---API PANIC----")
				trace := debug.Stack()
				log.Errorf("%s", string(trace))
			}
		}()

		var msg *fimpgo.FimpMessage
		var adr fimpgo.Address
		switch iotMsg.Type {
		case "cmd.ecprocess.get_list":
			response := api.integr.Processes()
			msg = fimpgo.NewMessage("evt.ecprocess.proc_list_report", "ecollector", fimpgo.VTypeObject, response, nil, nil, iotMsg)

		case "cmd.ecprocess.update_config":
			conf := tsdb.ProcessConfig{}
			err := iotMsg.GetObjectValue(&conf)
			if err != nil {
				log.Error("Wrong configuration format")
				return
			}
			err = api.integr.UpdateProcConfig(conf.ID, conf, true)
			if err != nil {
				log.Error("Err while updating proc config.Err :", err.Error())
			}

			errStr := ""
			status := "ok"
			if err != nil {
				status = "error"
				errStr = err.Error()
			}
			response := map[string]string{"op": "update_config", "status": status, "error": errStr}
			msg = fimpgo.NewStrMapMessage("evt.ecprocess.ctrl_report", "ecollector", response, nil, nil, iotMsg)

		case "cmd.ecprocess.add":
			_, err := api.integr.AddProcess(nil)
			if err != nil {
				log.Error("Err while adding new proc.Err :", err.Error())
			}
			errStr := ""
			status := "ok"
			if err != nil {
				status = "error"
				errStr = err.Error()
			}
			response := map[string]string{"op": "add", "status": status, "error": errStr}
			msg = fimpgo.NewStrMapMessage("evt.ecprocess.ctrl_report", "ecollector", response, nil, nil, iotMsg)

		case "cmd.ecprocess.ctrl":
			val, err := iotMsg.GetStrMapValue()
			if err != nil {
				log.Debug(" Wrong value format for cmd.ecprocess.ctrl")
				return
			}
			op, _ := val["operation"]
			status := "error"
			procId := api.getProcID(val)
			errStr := ""
			if procId > 0 && op != "" {
				proc := api.integr.GetProcessByID(procId)
				if proc != nil {
					switch op {
					case "start":
						err = proc.Start()
					case "stop":
						err = proc.Stop()
					case "delete":
						err = api.integr.RemoveProcess(procId)
					}
					if err == nil {
						status = "ok"
					}
				} else {
					status = "error"
					errStr = "unknown process id"
				}

			}

			if err != nil {
				errStr = err.Error()
			}
			response := map[string]string{"op": op, "status": status, "error": errStr}
			msg = fimpgo.NewStrMapMessage("evt.ecprocess.ctrl_report", "ecollector", response, nil, nil, iotMsg)

		case "cmd.ecprocess.reset_to_default":
			api.configs.LoadDefaults()
			api.integr.ResetConfigsToDefault()
			os.Exit(0)

		case "cmd.tsdb.query":
			val, err := iotMsg.GetStrMapValue()
			if err != nil {
				log.Debug(" Wrong value format for cmd.influxdb.query")
				return
			}
			procId := api.getProcID(val)
			if procId < 0 {
				log.Error(" Wrong process ID")
				return
			}
			query, _ := val["query"]
			_, storage, err := api.getProcAndStorageByProcId(procId)
			if err != nil {
				return
			}

			response, err := storage.RunQuery(query)
			if err != nil {
				log.Error("<api> Error while querying data . Err:", err.Error())
			}
			msg = fimpgo.NewMessage("evt.tsdb.query_report", "ecollector", fimpgo.VTypeObject, response, nil, nil, iotMsg)

		case "cmd.tsdb.write_data_points":
			req := WriteDataPointsRequest{}
			err := iotMsg.GetObjectValue(&req)
			if err != nil {
				log.Debug(" Wrong request value format for cmd.influxdb.write_data_points")
				return
			}
			if req.ProcID <= 0 {
				log.Error(" Wrong process ID")
				return
			}
			proc := api.integr.GetProcessByID(req.ProcID)
			if proc == nil {
				log.Error(" Can't fine process with ID = ", req.ProcID)
				return
			}

			for i := range req.DataPoints {
				ts := time.Now()
				idp, err := influx.NewPoint(req.DataPoints[i].Name, req.DataPoints[i].Tags, req.DataPoints[i].Fields, ts)
				if err != nil {
					log.Error("Can't create data point.Err:", err.Error())
				} else {
					proc.WriteDirect(req.Bucket, idp)
				}
			}

		case "cmd.tsdb.get_data_points":
			req := GetDataPointsRequest{}
			err := iotMsg.GetObjectValue(&req)
			if err != nil {
				log.Debug(" Wrong request value format for cmd.influxdb.get_data_points")
				return
			}

			if req.ProcID <= 0 {
				log.Error(" Wrong process ID")
				return
			}
			// fieldName,measurement,relativeTime,fromTime,toTime,groupByTime,fillType, dataFunction,groupByField

			_, storage, err := api.getProcAndStorageByProcId(req.ProcID)
			if err != nil {
				return
			}

			response := storage.GetDataPoints(req.FieldName, req.MeasurementName, req.RelativeTime, req.FromTime, req.ToTime, req.GroupByTime, req.FillType, req.DataFunction, req.TransformFunction, req.GroupByTag, req.Filters)
			msg = fimpgo.NewMessage("evt.tsdb.data_points_report", "ecollector", fimpgo.VTypeObject, response, nil, nil, iotMsg)

		case "cmd.tsdb.get_energy_data_points":
			req := GetDataPointsRequest{}
			err := iotMsg.GetObjectValue(&req)
			if err != nil {
				log.Debug(" Wrong request value format for cmd.influxdb.get_data_points")
				return
			}

			if req.ProcID <= 0 {
				log.Error(" Wrong process ID")
				return
			}
			// fieldName,measurement,relativeTime,fromTime,toTime,groupByTime,fillType, dataFunction,groupByField

			_, storage, err := api.getProcAndStorageByProcId(req.ProcID)
			if err != nil {
				return
			}
			response := storage.GetEnergyDataPoints(req.RelativeTime, req.FromTime, req.ToTime, req.GroupByTime, req.GroupByTag, req.Filters)
			msg = fimpgo.NewMessage("evt.tsdb.data_points_report", "ecollector", fimpgo.VTypeObject, response, nil, nil, iotMsg)

		case "cmd.tsdb.get_measurements":
			val, err := iotMsg.GetStrMapValue()
			if err != nil {
				log.Debug(" Wrong value format for cmd.influxdb.query")
				return
			}
			procId := api.getProcID(val)
			if procId < 0 {
				log.Error(" Wrong process ID")
				return
			}
			_, storage, err := api.getProcAndStorageByProcId(procId)
			if err != nil {
				return
			}
			response, err := storage.GetDbMeasurements()
			if err != nil {
				log.Error("<api> Error while getting measurements . Err:", err.Error())
			}
			msg = fimpgo.NewMessage("evt.tsdb.measurements_report", "ecollector", fimpgo.VTypeStrArray, response, nil, nil, iotMsg)

		case "cmd.tsdb.get_retention_policies":
			val, err := iotMsg.GetStrMapValue()
			if err != nil {
				log.Debug(" Wrong value format for cmd.ecprocess.ctrl")
				return
			}
			procId := api.getProcID(val)
			if procId < 0 {
				log.Error(" Wrong process ID")
				return
			}
			var response []string
			_, storage, err := api.getProcAndStorageByProcId(procId)
			if err != nil {
				return
			}
			response, err = storage.GetDbRetentionPolicies()
			if err != nil {
				log.Error("<api> Error while getting measurements . Err:", err.Error())
			}
			msg = fimpgo.NewMessage("evt.tsdb.retention_policies", "ecollector", fimpgo.VTypeStrArray, response, nil, nil, iotMsg)

		case "cmd.tsdb.add_retention_policy":
			// configure retentions
			val, err := iotMsg.GetStrMapValue()
			if err != nil {
				log.Debug(" Wrong value format for cmd.influxdb.query")
				return
			}
			procId := api.getProcID(val)
			if procId < 0 {
				log.Error(" Wrong process ID")
				return
			}
			name, _ := val["name"]
			duration, _ := val["duration"]

			_, storage, err := api.getProcAndStorageByProcId(procId)
			if err != nil {
				return
			}
			storage.AddRetentionPolicy(name, duration)

		case "cmd.tsdb.update_retention_policy":
			// configure retentions
			val, err := iotMsg.GetStrMapValue()
			if err != nil {
				log.Debug(" Wrong value format for cmd.influxdb.query")
				return
			}
			name, _ := val["name"]
			duration, _ := val["duration"]
			procId := api.getProcID(val)
			if procId < 0 {
				log.Error(" Wrong process ID")
				return
			}

			_, storage, err := api.getProcAndStorageByProcId(procId)
			if err != nil {
				return
			}
			storage.UpdateRetentionPolicy(name, duration)

		case "cmd.tsdb.delete_object":
			// configure retentions
			val, err := iotMsg.GetStrMapValue()
			if err != nil {
				log.Debug(" Wrong value format for cmd.influxdb.query")
				return
			}
			name, _ := val["name"]
			otype, _ := val["object_type"]
			procId := api.getProcID(val)
			if procId < 0 {
				log.Error(" Wrong process ID")
				return
			}

			proc, storage, err := api.getProcAndStorageByProcId(procId)
			if err != nil {
				return
			}

			switch otype {
			case "retention_policy":
				proc.Stop()
				storage.DeleteRetentionPolicy(name)
				proc.Start()
			case "database":
				proc.Stop()
				storage.DropDB(name)
				proc.Start()
			case "cq":
				storage.DeleteCQ(name)
			case "measurement":
				storage.DeleteMeasurement(name)
			}
			response := map[string]string{"status": "ok", "error": ""}
			msg = fimpgo.NewStrMapMessage("evt.tsdb.delete_object_report", "ecollector", response, nil, nil, iotMsg)
			// set default retention policy

		case "cmd.tsdb.get_configs":
			//
		case "cmd.log.set_level":
			// Configure log level
			level, err := iotMsg.GetStringValue()
			if err != nil {
				return
			}
			logLevel, err := log.ParseLevel(level)
			if err == nil {
				log.SetLevel(logLevel)
				api.configs.LogLevel = level
				api.configs.SaveToFile()
				log.Info("Log level updated to = ", logLevel)
			} else {
				log.Error("Failed to update log level.Err: ", err.Error())
			}

		}
		if msg == nil {
			return
		}
		if iotMsg.ResponseToTopic != "" {
			api.mqt.RespondToRequest(iotMsg, msg)
		} else {
			adr = fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeApp, ResourceName: "ecollector", ResourceAddress: "1"}
			api.mqt.Publish(&adr, msg)
		}
	}()

}

func (api *AdminApi) getProcAndStorageByProcId(procId tsdb.IDt) (*tsdb.Process, storage.DataStorage, error) {
	proc := api.integr.GetProcessByID(procId)
	if proc == nil {
		log.Error("Can't find process with id = %d ", procId)
		return nil, nil, fmt.Errorf("unknown process")
	}
	storage := proc.Storage()
	if storage == nil {
		log.Error("Storage is not initialized ")
		return nil, nil, fmt.Errorf("nil storage")
	}
	return proc, storage, nil
}
