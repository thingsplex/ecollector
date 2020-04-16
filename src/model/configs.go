package model

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"github.com/thingsplex/ecollector/utils"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Configs struct {
	path                  string
	InstanceAddress       string `json:"instance_address"`
	MqttServerURI         string `json:"mqtt_server_uri"`
	MqttUsername          string `json:"mqtt_server_username"`
	MqttPassword          string `json:"mqtt_server_password"`
	MqttClientIdPrefix    string `json:"mqtt_client_id_prefix"`
	LogFile               string `json:"log_file"`
	LogLevel              string `json:"log_level"`
	LogFormat             string `json:"log_format"`
	InitDb                bool   `json:"init_db"`
	DisableDiskMonitor    bool   `json:"disable_disk_monitor"`
	DiskMonitorShutdownLimit float64 `json:"disk_monitor_shutdown_limit"`
	WorkDirectory         string `json:"-"`
}

func NewConfigs(path string,workDir string) *Configs {
	return &Configs{path:path,WorkDirectory: workDir}
}

func (cf *Configs) SaveToFile() error {
	bpayload, err := json.Marshal(cf)
	err = ioutil.WriteFile(cf.path, bpayload, 0664)
	if err != nil {
		return err
	}
	return err
}

func (cf * Configs) LoadFromFile() error {
	configFileBody, err := ioutil.ReadFile(cf.path)
	if err != nil {
		return cf.SaveToFile()
	}
	err = json.Unmarshal(configFileBody, cf)
	if err != nil {
		return err
	}
	return nil
}

func (cf * Configs) LoadDefaults()error {
	configFile := filepath.Join(cf.WorkDirectory,"data","config.json")
	os.Remove(configFile)
	log.Info("Config file doesn't exist.Loading default config")
	defaultConfigFile := filepath.Join(cf.WorkDirectory,"defaults","config.json")
	return utils.CopyFile(defaultConfigFile,configFile)
}