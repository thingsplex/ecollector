package main

import (
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	api2 "github.com/thingsplex/ecollector/api"
	"github.com/thingsplex/ecollector/integration/tsdb"
	"github.com/thingsplex/ecollector/model"
	"github.com/thingsplex/ecollector/utils"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"path/filepath"
)

func SetupLog(logfile string, level string, logFormat string) {
	if logFormat == "json" {
		log.SetFormatter(&log.JSONFormatter{TimestampFormat: "2006-01-02 15:04:05.999"})
	} else {
		log.SetFormatter(&log.TextFormatter{FullTimestamp: true, ForceColors: true, TimestampFormat: "2006-01-02T15:04:05.999"})
	}

	logLevel, err := log.ParseLevel(level)
	if err == nil {
		log.SetLevel(logLevel)
	} else {
		log.SetLevel(log.DebugLevel)
	}

	if logfile != "" {
		l := lumberjack.Logger{
			Filename:   logfile,
			MaxSize:    5, // megabytes
			MaxBackups: 2,
		}
		log.SetOutput(&l)
	}
}

// overrides configs
func processEnvVars(configs *model.Configs) {
	if mqttUrl := os.Getenv("MQTT_URI"); mqttUrl != "" {
		configs.MqttServerURI = mqttUrl
	}
	if mqttUsername := os.Getenv("MQTT_USERNAME"); mqttUsername != "" {
		configs.MqttUsername = mqttUsername
	}
	if mqttPass := os.Getenv("MQTT_PASSWORD"); mqttPass != "" {
		configs.MqttPassword = mqttPass
	}
}

var Version string

func main() {
	var workDir string
	flag.StringVar(&workDir, "c", "", "Work dir")
	flag.Parse()
	if workDir == "" {
		workDir = "./"
	} else {
		fmt.Println("Work dir ", workDir)
	}

	configFile := filepath.Join(workDir, "data", "config.json")
	if !utils.FileExists(configFile) {
		defaultConfigFile := filepath.Join(workDir, "defaults", "config.json")
		log.Info("Config file doesn't exist.Loading default config from ", defaultConfigFile)
		err := utils.CopyFile(defaultConfigFile, configFile)
		if err != nil {
			fmt.Print(err)
			panic("Can't copy config file.")
		}
	}

	configs := model.NewConfigs(configFile, workDir)
	err := configs.LoadFromFile()
	if err != nil {
		fmt.Print(err)
		panic("Can't parse config file.")
	}
	processEnvVars(configs)
	SetupLog(configs.LogFile, configs.LogLevel, configs.LogFormat)
	log.Info("Loading main config file from ", configFile)
	log.Info("Control plane broker url :", configs.MqttServerURI)
	log.Infof("--------------Starting ECollector v.%s -----------------", Version)

	integr := tsdb.Boot(configs)

	api := api2.NewAdminApi(integr, configs)
	api.Start()
	//	proc := integr.GetProcessByID(1)

	select {}
}
