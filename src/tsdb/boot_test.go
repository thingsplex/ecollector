package tsdb

import (
	"testing"
	"time"

	"github.com/spf13/viper"
)

// func TestIntegration_SaveConfigs(t *testing.T) {

// 	it := &Integration{
// 		processConfigs: ProcessConfig{}{
// 			ProcessConfig{
// 			Filters:   []Filter{Filter{}},
// 			Selectors: []Selector{Selector{}},
// 		},
// 			ProcessConfig{}},
// 		StoreLocation: "./",
// 	}
// 	it.boot()
// 	err := it.SaveConfigs()
// 	if err != nil {
// 		t.Error(err)
// 	}

// }

func SetupIntegrationTest() {
	viper.SetDefault("mqtt_broker_addr", "localhost:1883")
	viper.SetDefault("mqtt_username", "")
	viper.SetDefault("mqtt_password", "")
	viper.SetDefault("mqtt_clientid", "bfint-influxdb")
}

func TestIntegration(t *testing.T) {
	SetupIntegrationTest()
	selector := []Selector{
		Selector{ID: 1, Topic: "j1/evt/+"},
	}
	filters := []Filter{
		Filter{
			ID:       1,
			MsgType: "evt.sensor.report",
			IsAtomic: true,
		},
		Filter{
			ID:       2,
			MsgType: "evt.binary.report",
			IsAtomic: true,
		},
		Filter{
			ID:       3,
			MsgType: "evt.meter.report",
			IsAtomic: true,
		},Filter{
			ID:       4,
			MsgType: "evt.open.report",
			IsAtomic: true,
		},Filter{
			ID:       5,
			MsgType: "evt.presence.report",
			IsAtomic: true,
		},
	}

	measurements := []Measurement{
		Measurement{
			ID:                      "sensor_temp",
			RetentionPolicyDuration: "8w",
			RetentionPolicyName:     "bf_sensor_temp",
		},
		Measurement{
			ID:                      "sensor_lumin",
			RetentionPolicyDuration: "8w",
			RetentionPolicyName:     "bf_sensor_lumin",
		},
		Measurement{
			ID:                      "sensor_presence",
			RetentionPolicyDuration: "8w",
			RetentionPolicyName:     "bf_sensor_presence",
		},
		Measurement{
			ID:                      "sensor_contact",
			RetentionPolicyDuration: "8w",
			RetentionPolicyName:     "sensor_contact",
		},
		Measurement{
			ID:                      "meter_elec",
			RetentionPolicyDuration: "8w",
			RetentionPolicyName:     "bf_meter_elec",
		},
		Measurement{
			ID:                      "default",
			RetentionPolicyDuration: "8w",
			RetentionPolicyName:     "bf_default",
		},
	}

	config1 := ProcessConfig{
		ID:                 1,
		MqttBrokerAddr:     "tcp://" + viper.GetString("mqtt_broker_addr"),
		MqttBrokerUsername: viper.GetString("mqtt_username"),
		MqttBrokerPassword: viper.GetString("mqtt_password"),
		MqttClientID:       viper.GetString("mqtt_clientid") + "-test-1",
		InfluxAddr:         "http://localhost:8086",
		InfluxUsername:     "",
		InfluxPassword:     "",
		InfluxDB:           "iotmsg_test",
		BatchMaxSize:       1000,
		SaveInterval:       1000,
		Filters:            filters,
		Selectors:          selector,
		Measurements:       measurements,
	}
	config2 := ProcessConfig{
		ID:                 2,
		MqttBrokerAddr:     "tcp://" + viper.GetString("mqtt_broker_addr"),
		MqttBrokerUsername: viper.GetString("mqtt_username"),
		MqttBrokerPassword: viper.GetString("mqtt_password"),
		MqttClientID:       viper.GetString("mqtt_clientid") + "-test-2",
		InfluxAddr:         "http://localhost:8086",
		InfluxUsername:     "",
		InfluxPassword:     "",
		InfluxDB:           "iotmsg_test",
		BatchMaxSize:       1000,
		SaveInterval:       1000,
		Filters:            filters,
		Selectors:          selector,
		Measurements:       measurements,
	}

	integr := Integration{Name: "influxdb", StoreLocation: ""}
	integr.SetConfig([]ProcessConfig{config1})
	if err := integr.InitProcesses(); err != nil {
		t.Error("Process Init failed .")
	}

	if integr.GetProcessByID(1).State != "RUNNING" {
		t.Error("Process is not running")
	}

	// Adding new process

	if _, err := integr.AddProcess(config2); err != nil {
		t.Error("Failed to add new process")
	}

	if integr.GetProcessByID(1).State != "RUNNING" {
		t.Error("Process is not running")
	}
	if integr.GetProcessByID(2).State != "RUNNING" {
		t.Error("Process is not running")
	}
	t.Log("Process was added")
	time.Sleep(time.Second * 10)
	t.Log("Removing process ")
	if err := integr.RemoveProcess(2); err != nil {
		t.Error("Failed to remove process")
	}

	if len(integr.processes) != 1 || len(integr.processConfigs) != 1 {
		t.Error("Process deleted process still exists")
	}

	if err := integr.GetProcessByID(1).Stop(); err != nil {
		t.Fatal("Process was started but failed to stop.")
	}
	time.Sleep(time.Second * 1)

}
