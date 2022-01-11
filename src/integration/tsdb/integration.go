package tsdb

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/futurehomeno/fimpgo"
	"github.com/shirou/gopsutil/disk"
	log "github.com/sirupsen/logrus"
	"github.com/thingsplex/ecollector/metadata"
	"github.com/thingsplex/ecollector/model"
	"github.com/thingsplex/ecollector/utils"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Integration is root level container
type Integration struct {
	processes                []*Process
	processConfigs           []ProcessConfig // TODO: Remove.There is no need to store configuration file in memory.
	workDir                  string
	storeFullPath            string
	Name                     string
	diskMonitorTicker        *time.Ticker
	configSaveMutex          *sync.Mutex
	serviceMedataStore       metadata.MetadataStore // metadata store is used for event enrichment
	DisableDiskMonitor       bool
	DiskMonitorShutdownLimit float64 // used disk space limit in % , if disk used space goes above the limit , ecollector stops all processes
	ControlMqttUri           string
	ControlMqttUsername      string
	ControlMqttPassword      string
}

func (it *Integration) Processes() []*Process {
	return it.processes
}

// GetProcessByID returns process by it's ID
func (it *Integration) GetProcessByID(ID IDt) *Process {
	for i := range it.processes {
		if it.processes[i].Config.ID == ID {
			return it.processes[i]
		}
	}
	return nil
}

// GetDefaultIntegrConfig returns default config .
func (it *Integration) GetDefaultIntegrConfig() ([]ProcessConfig, error) {
	defaultConfigFile := filepath.Join(it.workDir, "defaults", it.Name+".json")
	configFileBody, err := ioutil.ReadFile(defaultConfigFile)
	var conf []ProcessConfig
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(configFileBody, &conf)
	if err != nil {
		return nil, err
	}
	for i := range conf {
		conf[i].MqttClientID = fmt.Sprintf("eclr_%d", utils.GenerateRandomNumber())
	}
	return conf, nil

}

// Init initializes integration app
func (it *Integration) Init() {
	it.storeFullPath = filepath.Join(it.workDir, "data", "proc", it.Name+".json")
	//if !utils.FileExists(it.storeFullPath) {
	//	err := utils.CopyFile(filepath.Join(it.workDir, "defaults", it.Name+".json"), it.storeFullPath)
	//	if err != nil {
	//		log.Error("Failed to load integration default template , Err:", err.Error())
	//	}
	//}
}

// SetConfig config setter
func (it *Integration) SetConfig(processConfigs []ProcessConfig) {
	it.processConfigs = processConfigs
}

// UpdateProcConfig update process configurations
func (it *Integration) UpdateProcConfig(ID IDt, procConfig ProcessConfig, doRestart bool) error {
	proc := it.GetProcessByID(ID)
	// Updating process instance
	// TODO : Stop existing process regarding of state
	if doRestart {
		proc.Stop()
	}
	err := proc.Configure(procConfig, false)
	if err != nil {
		return err
	}
	// Updating configuration files
	for ic := range it.processConfigs {
		if it.processConfigs[ic].ID == ID {
			it.processConfigs[ic] = procConfig
			break
		}
	}
	err = it.SaveConfigs()
	if err != nil {
		return err
	}

	if doRestart {
		return proc.Start(false)
	}
	return err
}

// LoadConfig loads integration configs from json file and saves it into ProcessConfigs
func (it *Integration) LoadConfig() error {

	if it.configSaveMutex == nil {
		it.configSaveMutex = &sync.Mutex{}
	}
	if _, err := os.Stat(it.storeFullPath); os.IsNotExist(err) {
		it.processConfigs, err = it.GetDefaultIntegrConfig()
		if err != nil {
			log.Error("Can't load default configurations.Err: ", err.Error())
		}
		log.Info("Integration configuration is loaded from default.")
		return it.SaveConfigs()
	}
	payload, err := ioutil.ReadFile(it.storeFullPath)
	if err != nil {
		log.Errorf("Integration can't load configuration file from %s, Errro:%s", it.storeFullPath, err)
		return err
	}
	err = json.Unmarshal(payload, &it.processConfigs)
	if err != nil {
		log.Error("Can't load the integration cofig.Unmarshall error :", err)
	}
	return err

}

func (it *Integration) ResetConfigsToDefault() error {
	log.Info("Reseting configs to default")

	for i := range it.processes {
		it.processes[i].Stop()
	}
	var err error
	it.processes = it.processes[:0]
	it.processConfigs, err = it.GetDefaultIntegrConfig()
	if err != nil {
		return err
	}
	err = it.SaveConfigs()
	if err != nil {
		log.Error("<vmeta> Error while saving the config:", err)
	} else {
		it.LoadConfig()
		it.InitProcesses()
	}

	return err
}

// SaveConfigs saves configs to json file
func (it *Integration) SaveConfigs() error {
	if it.storeFullPath != "" {

		it.configSaveMutex.Lock()
		defer func() {
			it.configSaveMutex.Unlock()
		}()
		payload, err := json.Marshal(it.processConfigs)
		if err != nil {
			return err
		}
		return ioutil.WriteFile(it.storeFullPath, payload, 0777)

	}
	log.Info("<vmeta> Save to disk was skipped , storeFullPath is empty")
	return nil
}

// InitProcesses loads and starts ALL processes based on ProcessConfigs
func (it *Integration) InitProcesses() error {
	var err error
	if it.processConfigs == nil {
		return errors.New("process not configured")
	}
	//Initializing shared metadata store.The store is shared between processes.
	log.Info("Loading metadata from ", it.processConfigs[0].MetadataStore)
	if it.processConfigs[0].MetadataStore == "" || it.processConfigs[0].MetadataStore == "vinculum" {
		mqttClientId := fmt.Sprintf("vinc_mstore_%d", utils.GenerateRandomNumber())
		mqt := fimpgo.NewMqttTransport(it.ControlMqttUri, mqttClientId, it.ControlMqttUsername, it.ControlMqttPassword, true, 1, 1)
		mqt.Start()
		it.serviceMedataStore = metadata.NewVincMetadataStore(mqt)
		err = it.serviceMedataStore.Start()
	} else if it.processConfigs[0].MetadataStore == "tpflow" {

	} else if it.processConfigs[0].MetadataStore == "file" {
		it.serviceMedataStore = metadata.NewFileMetadataStore(it.processConfigs[0].MetadataStoreConfig)
		err = it.serviceMedataStore.Start()
	}

	if err != nil {
		log.Error("<boot> The process failed to connect to Meta Store . Err : ", err.Error())
	} else {
		log.Info("<boot> Meta store successfully initialized")
	}

	for i := range it.processConfigs {
		if it.processConfigs[i].SiteId == "" {
			it.processConfigs[i].SiteId = utils.GetFhSiteId("")
		}
		log.Info("<vmeta> Site id = ", it.processConfigs[i].SiteId)
		it.InitNewProcess(&it.processConfigs[i])
	}

	return nil
}

// InitNewProcess initialize and start single process
func (it *Integration) InitNewProcess(procConfig *ProcessConfig) error {
	if procConfig.Profile == "" {
		procConfig.Profile = "optimized"
	}
	proc := NewProcess(procConfig)
	proc.SetServiceMedataStore(it.serviceMedataStore)
	it.processes = append(it.processes, proc)
	if procConfig.Autostart {
		err := proc.Init()
		if err == nil {
			log.Infof("<boot> Process ID=%d was initialized.", procConfig.ID)
			err := proc.Start(true)
			log.Infof("<boot> Process ID=%d is started.", procConfig.ID)
			if err != nil {
				log.Errorf("<boot> Process ID=%d failed to start . Error : %s", procConfig, err)
			}

		} else {
			log.Errorf("<boot> Initialization of Process ID=%d FAILED . Error : %s", procConfig.ID, err)
			return err
		}
	}
	return nil
}

// AddProcess adds new process .
func (it *Integration) AddProcess(procConfig *ProcessConfig) (IDt, error) {
	if procConfig == nil {
		defaultProc, err := it.GetDefaultIntegrConfig()
		if err != nil {
			if err != nil {
				log.Error("Can't load default configurations.Err: ", err.Error())
			}
			return 0, err
		}
		procConfig = &ProcessConfig{}
		*procConfig = defaultProc[0]
		newID := GetNewID(it.processConfigs)
		log.Info("<tsdb> Adding new process.pID = ", newID)
		procConfig.ID = newID
		procConfig.Autostart = false
	}
	it.processConfigs = append(it.processConfigs, *procConfig)
	it.SaveConfigs()
	return procConfig.ID, it.InitNewProcess(procConfig)
}

// RemoveProcess stops process , removes it from config file and removes instance .
func (it *Integration) RemoveProcess(ID IDt) error {
	var err error
	// removing process instance
	for i := range it.processes {
		if it.processes[i].Config.ID == ID {
			err = it.processes[i].Stop()
			it.processes = append(it.processes[:i], it.processes[i+1:]...)
			break
		}
	}
	// removing from config file
	for ic := range it.processConfigs {
		if it.processConfigs[ic].ID == ID {
			it.processConfigs = append(it.processConfigs[:ic], it.processConfigs[ic+1:]...)
			break
		}
	}

	it.SaveConfigs()
	return err
}

func (it *Integration) StartDiskMonitor() {
	if it.diskMonitorTicker != nil {
		it.diskMonitorTicker.Stop()
	}
	it.diskMonitorTicker = time.NewTicker(1 * time.Minute)
	go func() {
		for {
			<-it.diskMonitorTicker.C
			info, err := disk.Usage("/")
			if err != nil {
				log.Error("<vmeta> Disk monitor failed to obtain disk usage .Error :", err.Error())
				continue
			}
			if info.UsedPercent > it.DiskMonitorShutdownLimit {
				log.Errorf("!!!!! DISK LOW SPACE !!!! Stopping all processes . Disk usage = %f", info.UsedPercent)
				for i := range it.processes {
					it.processes[i].Stop()
				}
				it.serviceMedataStore.Stop()
			}
		}
		log.Error("!!!!! DISK MONITOR HAS STOPPED !!!! ")
	}()

}

// Boot initializes integration
func Boot(mainConfig *model.Configs) *Integration {
	log.Info("<tsdb> Booting InfluxDB integration ")
	if mainConfig.WorkDirectory == "" {
		log.Info("<tsdb> Config path path is not defined  ")
		return nil
	}
	if mainConfig.DiskMonitorShutdownLimit == 0 {
		mainConfig.DiskMonitorShutdownLimit = 85
	}
	integr := Integration{Name: "influxdb",
		workDir:                  mainConfig.WorkDirectory,
		DiskMonitorShutdownLimit: mainConfig.DiskMonitorShutdownLimit,
		DisableDiskMonitor:       mainConfig.DisableDiskMonitor,
		ControlMqttUri:           mainConfig.MqttServerURI,
		ControlMqttUsername:      mainConfig.MqttUsername,
		ControlMqttPassword:      mainConfig.MqttPassword,
	}
	log.Info("<tsdb> Initializing integration  ")
	integr.Init()
	log.Info("<tsdb> Loading configs  ")
	integr.LoadConfig()
	log.Info("<tsdb> Initializing processes async ")
	go integr.InitProcesses() // Runs in its own goroutine so that boot process doesn't block in case of incorrect configs
	if !mainConfig.DisableDiskMonitor {
		log.Info("<tsdb> Starting disk monitor ")
		integr.StartDiskMonitor()
	}
	log.Info("<tsdb> All good . Running ... ")
	return &integr
}
