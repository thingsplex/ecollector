package tsdb

import (
	"errors"
	"fmt"
	"github.com/thingsplex/ecollector/metadata"
	"runtime/debug"
	"strings"
	"sync"
	"time"
	"github.com/futurehomeno/fimpgo"
	influx "github.com/influxdata/influxdb1-client/v2"
	log "github.com/sirupsen/logrus"
)

// Process implements integration flow between messaging system and influxdb timeseries database.
// It inserts events into db
type Process struct {
	mqttTransport *fimpgo.MqttTransport
	influxC     influx.Client
	Config      *ProcessConfig
	batchPoints map[string]influx.BatchPoints
	ticker      *time.Ticker
	writeMutex  *sync.Mutex
	apiMutex    *sync.Mutex
	transform   Transform
	State       string
	LastError   string
	serviceMedataStore metadata.MetadataStore // metadata store is used for event enrichment
}

// NewProcess is a constructor
func NewProcess(config *ProcessConfig) *Process {
	proc := Process{Config: config, transform: DefaultTransform}
	proc.writeMutex = &sync.Mutex{}
	proc.apiMutex = &sync.Mutex{}
	proc.State = "LOADED"
	return &proc
}

func (pr *Process) SetServiceMedataStore(serviceMedataStore metadata.MetadataStore) {
	pr.serviceMedataStore = serviceMedataStore
}

// Init doing the process bootrstrap .
func (pr *Process) Init() error {
	var err error
	pr.State = "INIT_FAILED"
	log.Info("<tsdb>Initializing influx client.")
	pr.influxC, err = influx.NewHTTPClient(influx.HTTPConfig{
		Addr:     pr.Config.InfluxAddr, //"http://localhost:8086",
		Username: pr.Config.InfluxUsername,
		Password: pr.Config.InfluxPassword,
		Timeout:30*time.Second,
	})
	if err != nil {
		log.Fatalln("Error: ", err)
		return err
	}

	if pr.Config.InitDb {
		// Creating database
		log.Info("<tsdb> Setting up database")
		q := influx.NewQuery(fmt.Sprintf("CREATE DATABASE %s", pr.Config.InfluxDB), "", "")
		if response, err := pr.influxC.Query(q); err == nil && response.Error() == nil {
			log.Infof("<tsdb> Database %s was created with status :%s", pr.Config.InfluxDB, response.Results)
		} else {
			pr.LastError = "InfluxDB is not reachable .Check connection parameters."
			pr.State = "INITIALIZED_WITH_ERRORS"
		}
		// Setting up retention policies
		log.Info("Setting up retention policies")

		// CQ buckets
		pr.AddRetentionPolicy("gen_year","240w")  // 1-5 years from last 5 years
		pr.AddRetentionPolicy("gen_month","48w") // 1-12 month from last year
		pr.AddRetentionPolicy("gen_week","12w") // 1-4 weeks from last 3 month
		pr.AddRetentionPolicy("gen_day","2w")   // 1-3 days from last month
		// Default bucket for high frequency measurements
		pr.AddRetentionPolicy("gen_raw","2w")   // 1-2 days from last week
		// Default bucket for slow measurements
		pr.AddRetentionPolicy("default_20w","12w") //

		log.Info("Setting up CQ ")

		pr.DeleteCQ("raw_to_day")
		pr.DeleteCQ("day_to_week")
		pr.DeleteCQ("week_to_month")
		pr.DeleteCQ("month_to_year")

		pr.AddCQ("raw_to_day","gen_raw","gen_day","1m")
		pr.AddCQ("day_to_week","gen_day","gen_week","10m")
		pr.AddCQ("week_to_month","gen_week","gen_month","1h")
		pr.AddCQ("month_to_year","gen_month","gen_year","1d")


	}else {
		log.Info("<tsdb> Database initialization is skipped.(turned off in config)")

	}

	pr.batchPoints = make(map[string]influx.BatchPoints)
	err = pr.InitBatchPoint("gen_raw")
	if err != nil {
		log.Error("<tsdb> Can't init batch points . Error: ", err)
	}
	err = pr.InitBatchPoint("default_20w")
	if err != nil {
		log.Error("<tsdb> Can't init batch points . Error: ", err)
	}

	log.Info("<tsdb> DB initialization completed.")
	log.Info("<tsdb> Initializing MQTT adapter.")
	//"tcp://localhost:1883", "blackflowint", "", ""
	pr.mqttTransport = fimpgo.NewMqttTransport(pr.Config.MqttBrokerAddr,pr.Config.MqttClientID,pr.Config.MqttBrokerUsername, pr.Config.MqttBrokerPassword,true,1,1)
	pr.mqttTransport.SetMessageHandler(pr.OnMessage)
	log.Info("<tsdb> MQTT adapter initialization completed.")
	if pr.State == "INIT_FAILED" {
		pr.State = "INITIALIZED"
	}

	log.Info("<tsdb> the process init state =",pr.State )
	return nil
}

// OnMessage is invoked by an adapter on every new message
// The code is executed in callers goroutine
func (pr *Process) OnMessage(topic string, addr *fimpgo.Address , iotMsg *fimpgo.FimpMessage, rawMessage []byte) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("---PANIC----")
			log.Errorf("OnMessage Err:%v",r)
			debug.PrintStack()

		}
	}()
	// log.Debugf("New msg of class = %s", iotMsg.Class
	context := &MsgContext{time:time.Now()}

	if pr.Config.SiteId!="" {
		addr.GlobalPrefix = pr.Config.SiteId
	}
	if pr.filter(context, topic, iotMsg, addr.GlobalPrefix, 0) {
		meta ,err := pr.serviceMedataStore.GetMetadataByAddress(topic)
		if err == nil {
			context.metadata = &meta
		}else {
			log.Debug("No metadata found")
		}
		points, err := pr.transform(context, topic,addr, iotMsg, addr.GlobalPrefix)
		if err != nil {
			log.Errorf("<tsdb> Transformation error: %s", err)
		} else {
			if points != nil {
				for i := range points {
					pr.write(context, points[i])
				}
			} else {
				log.Debug("<tsdb> Message can't be mapped .Skipping .")
			}

		}
	} else {
		log.Tracef("<tsdb> Message from topic %s is skiped .", topic)
	}
}

// AddMessage is invoked by an adapter on every new message
// Is used by batch loader
// The code is executed in callers goroutine
func (pr *Process) AddMessage(topic string, addr *fimpgo.Address , iotMsg *fimpgo.FimpMessage, modTime time.Time) {
	// log.Debugf("New msg of class = %s", iotMsg.Class
	context := &MsgContext{time:modTime}
	if pr.filter(context, topic, iotMsg, addr.GlobalPrefix, 0) {
		points, err := pr.transform(context, topic,addr, iotMsg, addr.GlobalPrefix)

		if err != nil {
			log.Errorf("<tsdb> Transformation error: %s", err)
		} else {
			if points != nil {
				for i := range points {
					pr.write(context, points[i])
				}
			} else {
				log.Debug("<tsdb> Message can't be mapped .Skipping .")
			}
		}
	} else {
		log.Debugf("<tsdb> Message from topic %s is skiped .", topic)
	}
}


// Filter - transforms IotMsg into DB compatible struct
func (pr *Process) filter(context *MsgContext, topic string, iotMsg *fimpgo.FimpMessage, domain string, filterID IDt) bool {
	var result bool
	// no filters defines , everything is allowed

	for i := range pr.Config.Filters {
		if (pr.Config.Filters[i].IsAtomic && filterID == 0) || (pr.Config.Filters[i].ID == filterID) {
			result = true
			if pr.Config.Filters[i].Topic != "" {
				if topic != pr.Config.Filters[i].Topic {
					result = false
				}
			}
			if pr.Config.Filters[i].Domain != "" {
				if domain != pr.Config.Filters[i].Domain {
					result = false
				}
			}
			if pr.Config.Filters[i].MsgType != "" {
				if iotMsg.Type != pr.Config.Filters[i].MsgType {
					result = false
				}
			}
			if pr.Config.Filters[i].Service != "" {
				if iotMsg.Service != pr.Config.Filters[i].Service {
					result = false
				}
			}

			if pr.Config.Filters[i].Negation {
				result = !(result)
			}
			if pr.Config.Filters[i].LinkedFilterID != 0 {
				// filters chaining
				// log.Debug("Starting recursion. Current result = ", result)
				nextResult := pr.filter(context, topic, iotMsg, domain, pr.Config.Filters[i].LinkedFilterID)
				// log.Debug("Nested call returned ", nextResult)
				switch pr.Config.Filters[i].LinkedFilterBooleanOperation {
				case "or":
					result = result || nextResult
				case "and":
					result = result && nextResult

				}
			}
			return result
		}
	}

	return false
}


// write - writes data points into batch point
func (pr *Process) write(context *MsgContext, point *DataPoint) {
	// log.Debugf("Point: %+v", point)
	rpName := pr.getRetentionPolicyName(point.MeasurementName)
	log.Debugf("<tsdb> Writing measurement: %s into %s", point.Point.Name(),rpName)
	if context.measurementName != "" {
		pr.writeMutex.Lock()
		pr.batchPoints[rpName].AddPoint(point.Point)
		pr.writeMutex.Unlock()
		if len(pr.batchPoints[rpName].Points()) >= pr.Config.BatchMaxSize {
			pr.WriteIntoDb()
		}
	}
}

//func (pr *Process) writeMultiple(context *MsgContext, point []*influx.Point) {
//	rpName := pr.getRetentionPolicyName(context.measurementName)
//	log.Debugf("<tsdb> Writing measurements: %s into %s", context.measurementName,rpName)
//	// log.Debugf("Point: %+v", point)
//	if context.measurementName != "" {
//		pr.writeMutex.Lock()
//		pr.batchPoints[rpName].AddPoints(point)
//		pr.writeMutex.Unlock()
//		if len(pr.batchPoints[rpName].Points()) >= pr.Config.BatchMaxSize {
//			pr.WriteIntoDb()
//		}
//	}
//}

// Configure should be used to replace new set of filters and selectors with new set .
// Process should be restarted after Configure call
func (pr *Process) Configure(procConfig ProcessConfig, doRestart bool) error {
	// pr.Config.Selectors = selectors
	// pr.Config.Filters = filters
	*pr.Config = procConfig
	if doRestart {
		pr.Stop()
		return pr.Start()
	}
	return nil
}

func (pr *Process) getRetentionPolicyName(mName string ) string {
	if mName == "electricity_meter_power" ||
		mName == "electricity_meter_energy" ||
		mName == "electricity_meter_ext" ||
		strings.Contains(mName,"sensor_") {
		return "gen_raw"
	}
	return "default_20w"
}

// InitBatchPoint initializes new batch point or resets existing one .
func (pr *Process) InitBatchPoint(bpName string) error {
	var err error
	// Create a new point batch
	log.Debugf("Init new batch point %s",bpName)
	pr.batchPoints[bpName], err = influx.NewBatchPoints(influx.BatchPointsConfig{
				Database:        pr.Config.InfluxDB,
				Precision:       "ns",
				RetentionPolicy: bpName,
	})

	return err
}

// WriteIntoDb - inserts record into db
func (pr *Process) WriteIntoDb() error {
	// Mutex is needed to fix condition when the function is invoked by timer and batch size almost at the same time
	defer func() {
		pr.writeMutex.Unlock()
	}()
	pr.writeMutex.Lock()

	for bpKey := range pr.batchPoints {
		if len(pr.batchPoints[bpKey].Points()) == 0 {
			continue
		}
		log.Debugf("<tsdb> Writing batch of size = %d , using retention policy = %s into db = %s", len(pr.batchPoints[bpKey].Points()), pr.batchPoints[bpKey].RetentionPolicy(), pr.batchPoints[bpKey].Database())
		var err error

		for i:=0; i<5 ; i++  {
			err = pr.influxC.Write(pr.batchPoints[bpKey])
			if err == nil {
				break
			}else if strings.Contains(err.Error(),"field type conflict") {
				break
			}else if strings.Contains(err.Error(),"unable to parse") {
				break
			} else  {
				log.Error("Retrying error after 5 sec. Err:",err.Error())
				time.Sleep(time.Second*5)
			}
		}

		if err != nil {
			if strings.Contains(err.Error(),"unable to parse") {
				log.Error("<tsdb> Batch write error , unable to parse packet.Error: ", err)
			}else if strings.Contains(err.Error(),"field type conflict") {
				log.Error("<tsdb> Field type conflict.Error: ", err)
			} else  {
				pr.State = "LOST_CONNECTION"
				log.Error("<tsdb> Batch write error , batch is dropped.Changing state to LOST_CONNECTION ", err)
			}
			err = pr.InitBatchPoint(bpKey)

		}else {
			if pr.State != "RUNNING" {
				pr.State = "RUNNING"
			}
			err = pr.InitBatchPoint(bpKey)
			if err != nil {
				log.Error("<tsdb> Batch init error , batch is dropped: ", err)
			}

		}

		if len(pr.batchPoints[bpKey].Points()) >= (pr.Config.BatchMaxSize+2000) {
			log.Error("BATCH size is too big. Removing all records.")
			// protection against infinite grows
			err = pr.InitBatchPoint(bpKey)
			if err != nil {
				log.Error("<tsdb> Batch init error , batch is dropped: ", err)
			}
		}
	}
	return nil
}

// Start starts the process by starting MQTT adapter ,
// starting scheduler
func (pr *Process) Start() error {
	log.Info("<tsdb> Starting process...")
	// try to initialize process first if current state is not INITIALIZED
	if pr.State == "INIT_FAILED" || pr.State == "LOADED" || pr.State == "INITIALIZED_WITH_ERRORS" || pr.State == "STOPPED" {
		if err := pr.Init(); err != nil {
			return err
		}
	}
	pr.ticker = time.NewTicker(time.Millisecond * pr.Config.SaveInterval)
	go func() {
		for _ = range pr.ticker.C {
			pr.WriteIntoDb()
		}
	}()
	err := pr.mqttTransport.Start()
	if err != nil {
		log.Error("Error: ", err)
		return err
	}
	for _, selector := range pr.Config.Selectors {
		pr.mqttTransport.Subscribe(selector.Topic)
	}

	if pr.serviceMedataStore == nil {
		pr.serviceMedataStore = metadata.NewVincMetadataStore(pr.mqttTransport)
		pr.serviceMedataStore.Start()
	}
	if pr.State == "INITIALIZED"{
		pr.State = "RUNNING"
	}
	//pr.serviceMedataStore = metadata.NewTpMetadataStore(pr.mqttTransport)
	//pr.serviceMedataStore.LoadFromTpRegistry()

	log.Info("<tsdb> Process started. State = RUNNING ")
	return nil

}

// Stop stops the process by unsubscribing from all topics ,
// stops scheduler and stops adapter.
func (pr *Process) Stop() error {
	if pr.State != "RUNNING" {
		return errors.New("process isn't running, nothing to stop")
	}
	log.Info("<tsdb> Stopping process...")
	pr.ticker.Stop()

	for _, selector := range pr.Config.Selectors {
		pr.mqttTransport.Unsubscribe(selector.Topic)
	}
	pr.influxC.Close()
	pr.mqttTransport.Stop()
	pr.serviceMedataStore.Stop()
	pr.State = "STOPPED"
	log.Info("<tsdb> Process stopped")
	return nil
}





