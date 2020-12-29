package tsdb

import (
	"errors"
	"github.com/futurehomeno/fimpgo"
	influx "github.com/influxdata/influxdb1-client/v2"
	log "github.com/sirupsen/logrus"
	"github.com/thingsplex/ecollector/integration/tsdb/processing"
	"github.com/thingsplex/ecollector/integration/tsdb/storage"
	"github.com/thingsplex/ecollector/metadata"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

// Process implements integration flow between messaging system and influxdb timeseries database.
// It inserts events into db
type Process struct {
	ID                 IDt
	mqttTransport      *fimpgo.MqttTransport
	Config             *ProcessConfig
	storage            *storage.InfluxV1Storage
	batchPoints        map[string]influx.BatchPoints
	ticker             *time.Ticker
	writeMutex         *sync.Mutex
	apiMutex           *sync.Mutex
	transform          Transform
	State              string
	LastError          string
	rawAggregator      *processing.DataPointAggregator
	serviceMedataStore metadata.MetadataStore // metadata store is used for event enrichment
}

func (pr *Process) Storage() *storage.InfluxV1Storage {
	return pr.storage
}

// NewProcess is a constructor
func NewProcess(config *ProcessConfig) *Process {
	proc := Process{Config: config, transform: DefaultTransform}
	proc.writeMutex = &sync.Mutex{}
	proc.apiMutex = &sync.Mutex{}
	proc.State = "LOADED"
	proc.ID = config.ID
	proc.rawAggregator = processing.NewDataPointAggregator(30*time.Second)
	return &proc
}

func (pr *Process) SetServiceMedataStore(serviceMedataStore metadata.MetadataStore) {
	pr.serviceMedataStore = serviceMedataStore
}

// Init - initializes the process - creates MQTT connection , initializes storage , initializes batch points
func (pr *Process) Init() error {
	var err error
	pr.State = "INIT_FAILED"
	log.Info("<tsdb>Initializing influx client.")

	pr.storage,err = storage.NewInfluxV1Storage(pr.Config.InfluxAddr,pr.Config.InfluxUsername,pr.Config.InfluxPassword,pr.Config.InfluxDB)

	if err != nil {
		return err
	}

	if pr.Config.InitDb {
		// Creating database
		log.Info("<tsdb> Setting up database")
		if err = pr.storage.InitDB(pr.Config.InfluxDB); err != nil {
			pr.LastError = "InfluxDB is not reachable .Check connection parameters."
			pr.State = "INITIALIZED_WITH_ERRORS"
		}
		// Setting up retention policies
		log.Info("Setting up retention policies")
		if pr.Config.Profile == "optimized" {
			pr.storage.InitDefaultBuckets()
		}else {
			pr.storage.InitSimpleBuckets()
		}
	}else {
		log.Info("<tsdb> Database initialization is skipped.(turned off in config)")

	}
	pr.batchPoints = make(map[string]influx.BatchPoints)
	err = pr.InitBatchPoint("gen_raw")
	if err != nil {
		log.Error("<tsdb> Can't init batch points . Error: ", err)
	}
	err = pr.InitBatchPoint("gen_default")
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
	pr.StartAggregatorWorker()
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
	context := &MsgContext{time: time.Now()}
	context.measurementName = iotMsg.Service+"."+iotMsg.Type

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
					if storage.IsHighFrequencyData(points[i].MeasurementName) {
						// writing into aggregation store
						fields, _ := points[i].Point.Fields()
						agDp := processing.DataPoint{
							MeasurementName: points[i].MeasurementName,
							SeriesID:        points[i].SeriesID,
							Tags:            points[i].Point.Tags(),
							Fields:          fields,
							AggregationFunc: points[i].AggregationFunc,
							Value:           points[i].AggregationValue,
						}
						pr.rawAggregator.AddDataPoint(agDp)

					}else {
						pr.write(points[i]) // Writing directly to DB (writing to batch)
					}

				}
			} else {
				log.Debug("<tsdb> Message can't be mapped .Skipping .")
			}

		}
	} else {
		log.Tracef("<tsdb> Message from topic %s is skiped .", topic)
	}
}

func (pr *Process) StartAggregatorWorker() {
	go func() {
		for dp := range pr.rawAggregator.OutputChannel() {
			log.Debug("<tsdb> New aggregator event")
			point, err := influx.NewPoint(dp.MeasurementName, dp.Tags, dp.Fields, time.Now())
			if err == nil {
				pr.write(&DataPoint{
					MeasurementName:  dp.MeasurementName,
					Point:            point,
				})
			}else {
				log.Error("<tsdb> Can't create DP. Error:",err.Error())
			}
		}
		log.Error("<tsdb> Aggregator worker has QUIT")
	}()
}

// AddMessage Is used by batch loader
// The code is executed in callers goroutine
func (pr *Process) AddMessage(topic string, addr *fimpgo.Address , iotMsg *fimpgo.FimpMessage, modTime time.Time) {
	// log.Debugf("New msg of class = %s", iotMsg.Class
	context := &MsgContext{time: modTime}
	if pr.filter(context, topic, iotMsg, addr.GlobalPrefix, 0) {
		points, err := pr.transform(context, topic,addr, iotMsg, addr.GlobalPrefix)

		if err != nil {
			log.Errorf("<tsdb> Transformation error: %s", err)
		} else {
			if points != nil {
				for i := range points {
					pr.write(points[i])
				}
			} else {
				log.Debug("<tsdb> Message can't be mapped .Skipping .")
			}
		}
	} else {
		log.Debugf("<tsdb> Message from topic %s is skiped .", topic)
	}
}


// filter - transforms IotMsg into DB compatible struct
func (pr *Process) filter(context *MsgContext, topic string, iotMsg *fimpgo.FimpMessage, domain string, filterID IDt) bool {
	var result bool

	if iotMsg.Service == "ecollector" {
		// ignoring all messages from and to self .
		return false
	}
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
func (pr *Process) write(point *DataPoint) {
	// log.Debugf("Point: %+v", point)
	rpName := storage.ResolveWriteRetentionPolicyName(point.MeasurementName)
	log.Debugf("<tsdb> pID = %d. Writing measurement: %s into %s",pr.ID, point.Point.Name(),rpName)
	pr.writeMutex.Lock()
	pr.batchPoints[rpName].AddPoint(point.Point)
	pr.writeMutex.Unlock()
	if len(pr.batchPoints[rpName].Points()) >= pr.Config.BatchMaxSize {
		pr.WriteIntoDb()
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
	log.Info("Configuring process. pID = ",procConfig.ID)
	*pr.Config = procConfig
	if doRestart {
		pr.Stop()
		return pr.Start()
	}
	return nil
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
		log.Debugf("<tsdb> Writing batch of size = %d , using retention policy = %s into db = %s , proc = %d", len(pr.batchPoints[bpKey].Points()), pr.batchPoints[bpKey].RetentionPolicy(), pr.batchPoints[bpKey].Database(),pr.ID)
		var err error

		for i:=0; i<5 ; i++  {
			err = pr.storage.WriteDataPoints(pr.batchPoints[bpKey])
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
	if pr.Config.SaveInterval < 1000 {
		pr.Config.SaveInterval = 1000
	}
	if pr.Config.BatchMaxSize == 0 {
		pr.Config.BatchMaxSize = 1000
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
	pr.storage.Close()
	pr.mqttTransport.Stop()
	pr.State = "STOPPED"
	log.Info("<tsdb> Process stopped")
	return nil
}





