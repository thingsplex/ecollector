package tsdb

import (
	"fmt"
	"github.com/futurehomeno/fimpgo"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"encoding/json"

	influx "github.com/influxdata/influxdb1-client/v2"
	log "github.com/sirupsen/logrus"
)


func Setup() {
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	log.SetLevel(log.DebugLevel)

}

func MsgGenerator(config ProcessConfig, numberOfMsg int) {
	r := rand.New(rand.NewSource(99))
	topics := []string{
		"15",
		"16",
		"17",
	}
	config.MqttClientID = "blackflowint_pub_test"
	mqttTransport := fimpgo.NewMqttTransport(config.MqttBrokerAddr,config.MqttClientID, config.MqttBrokerUsername, config.MqttBrokerPassword,true,1,1)
	mqttTransport.Start()
	for i := 0; i < numberOfMsg; i++ {
		msg := fimpgo.NewFloatMessage("evt.sensor.report","sensor_temp",r.Float64(),fimpgo.Props{"unit":"C"},nil,nil)
		adr := fimpgo.Address{MsgType:fimpgo.MsgTypeEvt,ResourceType:fimpgo.ResourceTypeDevice,ResourceName:"zw",ResourceAddress:"1",ServiceName:"sensor_temp",ServiceAddress:topics[r.Intn(len(topics))]}
		mqttTransport.Publish(&adr,msg)
	}
	time.Sleep(time.Second * 3)
	mqttTransport.Stop()

}

func CleanUpDB(influxC influx.Client, config *ProcessConfig) {
	// Delete measurments
	q := influx.NewQuery(fmt.Sprintf("DROP MEASUREMENT \"sensor_temp.evt.sensor.report\""), config.InfluxDB, "")
	if response, err := influxC.Query(q); err == nil && response.Error() == nil {
		log.Info("Datebase was deleted with status :", response.Results)

	}

}

func Count(influxC influx.Client, config *ProcessConfig) int {
	q := influx.NewQuery(fmt.Sprintf("select count(value) from \"bf_default\".\"sensor_temp.evt.sensor.report\""), config.InfluxDB, "")
	if response, err := influxC.Query(q); err == nil && response.Error() == nil {
		if len(response.Results[0].Series) > 0 {
			countN, ok := response.Results[0].Series[0].Values[0][1].(json.Number)
			count, _ := countN.Int64()
			if !ok {
				log.Errorf("Type assertion failed , type is = %s", reflect.TypeOf(response.Results[0].Series[0].Values[0][1]))
			}
			log.Info("Number of received messages = ", count)
			return int(count)
		}
		log.Error("No Results")
		return 0

	}else {
		log.Error(err)
		log.Error(response.Error())
	}
	return 0
}

func TestProcess(t *testing.T) {
	Setup()
	//Start container : docker run --name influxdb -d -p 8084:8083 -p 8086:8086 -v influxdb:/var/lib/influxdb influxdb:1.1.0-rc1-alpine
	//Start mqtt broker
	NumberOfMessagesToSend := 100
	selector := []Selector{
		Selector{Topic: "pt:j1/mt:evt/#"},
	}
	//filters := []Filter{
	//	Filter{
	//		ID:       1,
	//	    Service: "sensor_temp",
	//		IsAtomic: true,
	//		MeasurementID:"default",
	//	},
	//}
	filters := []Filter{}
	config := ProcessConfig{
		MqttBrokerAddr:     "tcp://localhost:1883",
		MqttBrokerUsername: "",
		MqttBrokerPassword: "",
		MqttClientID:       "blackflowint_sub_test",
		InfluxAddr:         "http://localhost:8086",
		InfluxUsername:     "",
		InfluxPassword:     "",
		InfluxDB:           "iotmsg_test",
		BatchMaxSize:       1000,
		SaveInterval:       1000,
		Filters:            filters,
		Selectors:          selector,
	}

	influxC, err := influx.NewHTTPClient(influx.HTTPConfig{
		Addr:     config.InfluxAddr, //"http://localhost:8086",
		Username: config.InfluxUsername,
		Password: config.InfluxPassword,
	})
	if err != nil {
		t.Fatal("Error: ", err)
	}

	CleanUpDB(influxC, &config)
	proc := NewProcess(&config)
	proc.Init()
	err = proc.Start()
	if err != nil {
		t.Fatal(err)
	}
	MsgGenerator(config, NumberOfMessagesToSend)

	time.Sleep(time.Second * 2)
	CountOfSavedEvents := Count(influxC, &config)
	if NumberOfMessagesToSend != CountOfSavedEvents {
		t.Errorf("Number of sent messages doesn't match number of saved messages. Number of sent messages = %d , number of saved events = %d", NumberOfMessagesToSend, CountOfSavedEvents)
	}
	proc.Stop()
	influxC.Close()

}

//func TestFilter(t *testing.T) {
//	Setup()
//	filters := []Filter{
//		Filter{
//			ID:       1,
//			Topic:    "jim1/cmd/test/1",
//			IsAtomic: true,
//		},
//		Filter{
//			ID:          2,
//			MsgClass:    "binary",
//			MsgSubClass: "test",
//			IsAtomic:    true,
//		},
//		Filter{
//			ID:                           4,
//			MsgClass:                     "binary",
//			LinkedFilterID:               3,
//			LinkedFilterBooleanOperation: "and",
//			IsAtomic:                     true,
//		},
//		Filter{
//			ID:          3,
//			MsgSubClass: "lock",
//			IsAtomic:    false,
//		},
//	}
//
//	proc := NewProcess(&ProcessConfig{Filters: filters})
//	msg := iotmsg.NewIotMsg(iotmsg.MsgTypeEvt, "sensor", "temperature", nil)
//	context := &MsgContext{}
//	log.Info("Test #1")
//	if !proc.filter(context, "jim1/cmd/test/1", msg, "", 0) {
//		t.Error("Topic check has to return true.")
//	}
//	log.Info("Test #2")
//	if proc.filter(context, "jim1/cmd/test/2", msg, "", 0) {
//		t.Error("Topic check has to return false.")
//	}
//	log.Info("Test #3")
//	msg = iotmsg.NewIotMsg(iotmsg.MsgTypeEvt, "binary", "test", nil)
//	if !proc.filter(context, "jim1/cmd/test/3", msg, "", 0) {
//		t.Error("Topic check has to return true.")
//	}
//	log.Info("Test #4")
//	msg = iotmsg.NewIotMsg(iotmsg.MsgTypeEvt, "binary", "lock", nil)
//	if !proc.filter(context, "jim1/cmd/test/3", msg, "", 0) {
//		t.Error("Topic check has to return true.")
//	}
//	log.Info("Test #5")
//	filters = []Filter{
//		Filter{
//			ID:       1,
//			Topic:    "jim1/cmd/test/1",
//			Negation: true,
//			IsAtomic: true,
//		},
//		Filter{
//			ID:          2,
//			MsgClass:    "binary",
//			MsgSubClass: "test",
//			IsAtomic:    true,
//		},
//	}
//	proc = NewProcess(&ProcessConfig{Filters: filters})
//	msg = iotmsg.NewIotMsg(iotmsg.MsgTypeEvt, "binary", "switch", nil)
//	if !proc.filter(context, "jim1/cmd/test/3", msg, "", 0) {
//		t.Error("Topic check has to return true.")
//	}
//	log.Info("Test #6")
//	if proc.filter(context, "jim1/cmd/test/1", msg, "", 0) {
//		t.Error("Topic check has to return false.")
//	}
//	log.Info("Test #7 Add filter")
//
//	filters = []Filter{
//		Filter{
//			ID:       1,
//			Topic:    "jim1/cmd/test/1",
//			IsAtomic: true,
//		},
//		Filter{
//			ID:          2,
//			MsgClass:    "binary",
//			MsgSubClass: "test",
//			IsAtomic:    true,
//		},
//	}
//	proc = NewProcess(&ProcessConfig{Filters: filters})
//	msg = iotmsg.NewIotMsg(iotmsg.MsgTypeEvt, "test", "filter", nil)
//	if proc.filter(context, "jim/cmd/test/addfilter", msg, "", 0) {
//		t.Error("Topic check has to return False.")
//	}
//	newID := proc.AddFilter(Filter{IsAtomic: true, Topic: "jim/cmd/test/addfilter"})
//	t.Logf("New filter ID = %d", newID)
//	if !proc.filter(context, "jim/cmd/test/addfilter", msg, "", 0) {
//		t.Error("Topic check has to return true.")
//	}
//	proc.RemoveFilter(newID)
//	if proc.filter(context, "jim/cmd/test/addfilter", msg, "", 0) {
//		t.Error("Topic check has to return False.")
//	}
//	// proc.RemoveFilter
//}
