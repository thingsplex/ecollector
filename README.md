### ECollector service 

The service collects mqtt event from local/remote broker and saves them into timeseries DB . 

Supported timeseries DB : 

* influxDb 1.x

# Data management #  

## High frequency data ## 

Measurements : sensors , electricity_meter_power , electricity_meter_energy   


| Name      | Expected query range        | Aggregation | Retention |
|-----------|-----------------------------|-------------|-----------|
| gen_year  | 1-5 years for last 5 years  | 1 d avg     | 5 years   |
| gen_month | 1-12 month for last year    | 1 hour avg  | 1 year    |
| gen_week  | 1-4 weeks for  last 3 month | 10 min avg  | 3 month   |
| gen_day   | 2-14 days for last 2 weeks  | 1 min avg   | 2 weeks   |
| gen_raw   | 1-2 days for last 2 weeks   | none        | 2 weeks   |  

ecollector -> `gen_raw` -> CQ -> `gen_day` -> CQ -> `gen_week` -> CQ -> `gen_month` -> CQ -> `gen_year` 

if  (`to`-`from`) < `7days` AND (now() - `from`)< `month` use `gen_day` 

## Low frequency data ##

Measurements : all others 

Retention : 3 months , name - `default_w20`  

## Query API 

query params : 

1. measurement name
2. relative time . optional .
3. from timestamp . optional.
4. to timestamp . optional.
5. resolution , optional , default =`auto` 
6. missing data fill type, optional , default = `previous`   
7. aggregation function , optional,default = `mean` 
8. group field , optional , default = `device`

calculate retention policy using `end timestamp` - `start timestamp` and values from `expected query range` column 

**Message processing pipeline** : 
````


                        ------------------
                        |                |
       -----SUB---------|   Selectors    |
       |                | (subsriptions) |
       |                ------------------
      \|/ 
---------------             ------------                    -------------                      ---------                 ---------------       
|             |             |          |                    |           |                      |       |                 |   DataPoint |        
| MQTT broker | ---MSG----> |  Filters | --Context+IotMsg-->| Transform |--Context+DataPoint-->| Write | ---DataPoint--> |     Batch   |
|             |             |          |                    |           |                      |       |                 |             |       
---------------             ------------                    -------------                      ---------                 ---------------       
																															  |	
                           ------------------------Batch size trigger----------------------------------------------------------
                           |
                          \|/
                      --------------           ------------
   ---------          |   Write    |           |          |
   | Timer |--------->|   Batch    |---BATCH-->| InfluxDB |
   ---------          |            |           |          |
                      --------------           ------------





````

**Process configuration**

````                      

                          -----------------------------------
                          | ProcessConfig,Selectors,Filters | 
                          -----------------------------------
                               |
                              \|/ 
----------------         ------------------------          -------------
| MQTT broker  |-------->| Process A instance 1 |--------->| InfluxDB  | 
|              |         |                      |          |           |
----------------         ------------------------          -------------

                          -----------------------------------
                          | ProcessConfig,Selectors,Filters | 
                          -----------------------------------
                               |
                              \|/ 
----------------         ------------------------          -------------
| MQTT broker  |-------->| Process A instance 2 |--------->| InfluxDB  | 
|              |         |                      |          |           |
----------------         ------------------------          -------------

````
Configuration data model :

````

   --------------------------                                               
   |     ProcessConfig      |
   |________________________|
     |                 |
    /|\               /|\
-------------    ------------ 
| Selectors |    |  Filters | 
-------------    ------------ 
                    |    /|\
                    |     |
                    -------

````  
Process lifecycle : 

```
INIT_FAILED -> INITIALIZED -> RUNNING ->STOPPED

```


Filter structure : 

```
type Filter struct {
	ID          int
	Name        string
	Topic       string
	Domain      string
	MsgType     string
	MsgClass    string
	MsgSubClass string
	// If true then returns everythin except matching value
	Negation bool
	// Boolean operation between 2 filters , supported values : and , or
	LinkedFilterBooleanOperation string
	LinkedFilterID               int
	IsAtomic                     bool
	// Optional field , all tags defined here will be converted into influxDb tags
	Tags map[string]string
	// If set , then the value will overrride default measurement name defined in transformation
	MeasurementName string
	// definies if filter is temporary and can be stored in memory or persisted to disk
	InMemory bool
}
```

Selector structure :

``` 
type Selector struct {
	ID    IDt
	Topic string
	// definies if filter is temporary and can be stored in memory or persisted to disk
	InMemory bool
}
```

Tags : 
 TODO 

Measurement structure : 

```
// Measurement stores measurement specific configs like retention policy
type Measurement struct {
	Name string
	// Normally should be composed of Name + RetentionPolicy , for instance : sensor_1d
	RetentionPolicyName string
	// Have to be in format 1d , 1w , 1h .
	RetentionPolicyDuration string
}

```

Process config :

```
type ProcessConfig struct {
	ID                 string
	MqttBrokerAddr     string
	MqttClientID       string
	MqttBrokerUsername string
	MqttBrokerPassword string
	InfluxAddr         string
	InfluxUsername     string
	InfluxPassword     string
	InfluxDB           string
	// DataPoints are saved in batches .
	// Batch is sent to DB once it reaches BatchMaxSize or SaveInterval .
	// depends on what comes first .
	BatchMaxSize int
	// Interval in miliseconds
	SaveInterval time.Duration
	Filters      []Filter
	Selectors    []Selector
	Measurements []Measurement
}

```

Message context :
```
type MsgContext struct {
	FilterID        IDt
	MeasurementName string
}

```


