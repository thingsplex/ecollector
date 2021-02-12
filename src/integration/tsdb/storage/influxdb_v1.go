package storage

import (
	"fmt"
	influx "github.com/influxdata/influxdb1-client/v2"
	log "github.com/sirupsen/logrus"
	"time"
)

type  InfluxV1Storage struct {
	dbName string
	influxC    influx.Client
}

type DataPointsFilter struct {
	Tags              map[string]string      `json:"tags"`
}


func NewInfluxV1Storage(address,username,password,dbName string) (*InfluxV1Storage,error) {
	var err error
	ic := &InfluxV1Storage{dbName: dbName}
	ic.influxC, err = influx.NewHTTPClient(influx.HTTPConfig{
		Addr:     address, //"http://localhost:8086",
		Username: username,
		Password: password,
		Timeout:30*time.Second,
	})
	if err != nil {
		log.Fatalln("Error: ", err)
		return nil,err
	}
	return ic,nil
}

func (pr *InfluxV1Storage) InitDefaultBuckets() error {
	// CQ buckets
	err := pr.AddRetentionPolicy("gen_year","240w")  // 1-5 years from last 5 years
	if err != nil {
		return err
	}
	pr.AddRetentionPolicy("gen_month","48w") // 1-12 month from last year
	pr.AddRetentionPolicy("gen_week","12w") // 1-4 weeks from last 3 month
	pr.AddRetentionPolicy("gen_day","2w")   // 1-3 days from last month
	// Default bucket for high frequency measurements
	pr.AddRetentionPolicy("gen_raw","2w")   // 1-2 days from last week
	// Default bucket for slow measurements
	pr.AddRetentionPolicy("gen_default","12w") //

	err = pr.DeleteRetentionPolicy("default_20w")
	if err != nil {
		return err
	}
	pr.DeleteRetentionPolicy("default_8w")

	log.Info("Setting up CQ ")

	//pr.DeleteCQ("raw_to_day")
	//pr.DeleteCQ("day_to_week")
	//pr.DeleteCQ("week_to_month")
	//pr.DeleteCQ("month_to_year")

	err = pr.AddCQ("raw_to_day","gen_raw","gen_day","1m")
	if err != nil {
		return err
	}
	pr.AddCQ("day_to_week","gen_day","gen_week","10m")
	pr.AddCQ("week_to_month","gen_week","gen_month","1h")
	pr.AddCQ("month_to_year","gen_month","gen_year","1d")
	return nil
}

func (pr *InfluxV1Storage) InitSimpleBuckets() {
	pr.AddRetentionPolicy("gen_raw","240w")
	pr.AddRetentionPolicy("gen_default","240w")
}

func (pr *InfluxV1Storage) RunQuery(query string) (*influx.Response,error) {
	q := influx.NewQuery(query, pr.dbName, "s")
	if response, err := pr.influxC.Query(q); err == nil {
		log.Trace(response.Results)
		return response,response.Error()
	}else {
		return nil,err
	}
}
// GetDataPoints - relative must be in format 1m,1h,1d,1w . If groupByTime is empty , aggregate function will be skipped
func (pr *InfluxV1Storage) GetDataPoints(fieldName,measurement,relativeTime,fromTime,toTime,groupByTime,fillType, dataFunction,transformFunction, groupByTag string,filter DataPointsFilter ) *influx.Response {
	var retentionPolicyName,timeQuery,query,filterStr string
	var err error
	var timeInterval time.Duration

	if groupByTime == "auto" {
		groupByTime = ""
	}

	if fieldName == "" {
		fieldName = "value"
	}
	if (groupByTag != "" || groupByTime != "") && dataFunction == "" {
		dataFunction = "mean"
	}
	if fillType == "" {
		fillType = "null"
	}
	if !IsHighFrequencyData(measurement) {
		retentionPolicyName = "gen_default"
	}

	if fromTime != "" && toTime != "" {
		if retentionPolicyName == "" {
			retentionPolicyName,err  = ResolveRetentionName(fromTime,toTime)
		}
		if err != nil {
			log.Error("<ifv1> Can't resolve retention name.Err:",err.Error())
			return nil
		}
		timeQuery = fmt.Sprintf("time >= '%s' AND time <= '%s' ",fromTime,toTime)
	}else {
		timeInterval = ResolveDurationFromRelativeTime(relativeTime)
		userSetTimeGroupDuration := ResolveDurationFromRelativeTime(groupByTime)
		if retentionPolicyName == "" {
			retentionPolicyName = ResolveRetentionByElapsedTimeDuration(timeInterval)
			aggregationDuration := GetRetentionTimeGroupDuration(retentionPolicyName)
			if (userSetTimeGroupDuration >= aggregationDuration) && dataFunction == "mean" {
				retentionPolicyName = ResolveRetentionByTimeGroup(groupByTime)
			}
		}
		timeQuery = fmt.Sprintf("time > now()-%s",relativeTime)
	}
	fieldName = ResolveFieldFullName(fieldName,retentionPolicyName)
	//if groupByTime == "auto" {
	//	//groupByTime = CalculateGroupByTimeByInterval(timeInterval)
	//}

	for k,v := range filter.Tags {
		filterStr = fmt.Sprintf("%s AND %s = '%s'",filterStr,k,v)
	}

	selector := ""
	if groupByTime == "" && groupByTag !="" {
		query = fmt.Sprintf("AS \"value\" FROM \"%s\".\"%s\" WHERE %s %s GROUP BY %s FILL(%s)",
			retentionPolicyName,measurement,timeQuery,filterStr, groupByTag,fillType)
		selector = fmt.Sprintf("\"%s\"",fieldName)
	}else if groupByTime != "" && groupByTag =="" {
		query = fmt.Sprintf(" AS \"value\" FROM \"%s\".\"%s\" WHERE %s %s GROUP BY time(%s) FILL(%s)",
			retentionPolicyName,measurement,timeQuery,filterStr,groupByTime,fillType)
		selector = fmt.Sprintf("%s(\"%s\")",dataFunction,fieldName)
	}else if groupByTime != "" && groupByTag !="" {
		query = fmt.Sprintf(" AS \"value\" FROM \"%s\".\"%s\" WHERE %s %s GROUP BY time(%s), %s FILL(%s)",
			retentionPolicyName,measurement,timeQuery,filterStr,groupByTime, groupByTag,fillType)
		selector = fmt.Sprintf("%s(\"%s\")",dataFunction,fieldName)
	}else {
		if dataFunction != "" {
			query = fmt.Sprintf(" AS \"value\" FROM \"%s\".\"%s\" WHERE %s %s FILL(%s)",
				retentionPolicyName,measurement,timeQuery,filterStr,fillType)
			selector = fmt.Sprintf("%s(\"%s\")",dataFunction,fieldName)
		}else {
			query = fmt.Sprintf(" AS \"value\" FROM \"%s\".\"%s\" WHERE %s %s FILL(%s)",
				retentionPolicyName,measurement,timeQuery,filterStr,fillType)
			selector = fmt.Sprintf("\"%s\"",fieldName)
		}

	}
	if transformFunction != "" {
		selector = fmt.Sprintf("%s(%s)",transformFunction,selector)
	}

	query = fmt.Sprintf("SELECT %s %s",selector,query)
	log.Debug("<ifv1> --- Final query :",query)

	q := influx.NewQuery(query, pr.dbName, "s")
	if response, err := pr.influxC.Query(q); err == nil {
		log.Trace(response.Results)
		return response
	}else {
		log.Error("<ifv1> Get datapoint Error: ",err.Error())
		return nil
	}

}

// GetEnergyDataPoints - relative must be in format 1m,1h,1d,1w . If groupByTime is empty , aggregate function will be skipped
func (pr *InfluxV1Storage) GetEnergyDataPoints(relativeTime,fromTime,toTime,groupByTime,groupByTag string,filter DataPointsFilter ) *influx.Response {

	// OLD algo :  data is stored as accumulated data , this means data points always growing until reset
	// Calculations :
	// 1. group by - 1h/1d and by devices . We can only group by devices here.
	// 2. calculate difference , each datapoint in the result contains energy consumed for every hour/day. Use "mod" for ever result , to avoid negative values
	// 3. group by location/device-type and use sum aggregation function. That has to be done in code due to influx SQL limitations
	//    3.1 - get all groups (locations/device-types)
	//    3.2 - for each group , group by time with "sum" aggregation function

	var timeQuery,filterStr string

	if groupByTime != "1d" {
		groupByTime = "1h"
	}

	if fromTime != "" && toTime != "" {
		timeQuery = fmt.Sprintf("time >= '%s' AND time <= '%s' ",fromTime,toTime)
	}else {
		timeQuery = fmt.Sprintf("time > now()-%s",relativeTime)
	}

	for k,v := range filter.Tags {
		filterStr = fmt.Sprintf("%s AND %s = '%s'",filterStr,k,v)
	}

	// Query - SELECT abs(difference(max("value"))) AS "value" FROM "historian"."gen_raw"."electricity_meter_energy" WHERE time > :dashboardTime: GROUP BY time(1h), "dev_id" FILL(null)
    //query := fmt.Sprintf("SELECT abs(difference(max(\"value\"))) AS \"value\" FROM \"historian\".\"%s\".\"electricity_meter_energy\" WHERE %s GROUP BY time(%s), \"dev_id\" FILL(previous)",
    // 	retentionPolicyName,timeQuery,groupByTime)

	query := fmt.Sprintf("SELECT sum(\"value\") AS \"value\" FROM \"historian\".\"gen_year\".\"electricity_meter_energy_sampled\" WHERE %s %s GROUP BY time(%s), %s FILL(null)",
		timeQuery,filterStr,groupByTime,groupByTag)

	log.Debug("<ifv1> --- Final energy aggregation query :",query)
	q := influx.NewQuery(query, pr.dbName, "s")
	if response, err := pr.influxC.Query(q); err == nil {
		log.Trace(response.Results)
		//if len(response.Results) > 0 {
			//tFrame := processing.NewEcDataFrame()
			//tFrame.LoadFromInfluxResponse(response.Results[0],"dev_id",devicesInGroup)
			//err = tFrame.AggregateByGroupAndTime()
			//if err != nil {
			//	log.Error("<ifv1> Aggregation error 1 :",err.Error())
			//	response.Err = err.Error()
			//}else {
			//	r,err := tFrame.GetInfluxSeries()
			//	if err != nil {
			//		log.Error("<ifv1> Aggregation error 2 : ",err.Error())
			//		response.Err = err.Error()
			//	}
			//	response.Results[0] = *r
			//}
		//}

		return response
	}else {
		log.Error("<ifv1> Get datapoint Error: ",err.Error())
		return nil
	}

	 return nil
}


func (pr *InfluxV1Storage) WriteDataPoints(bp influx.BatchPoints) error {
	return pr.influxC.Write(bp)
}

func (pr *InfluxV1Storage) InitDB(name string) error {
	log.Info("<ifv1> Setting up database")
	q := influx.NewQuery(fmt.Sprintf("CREATE DATABASE %s", name), "", "")
	if response, err := pr.influxC.Query(q); err == nil  {
		log.Infof("<tsdb> Database %s was created with status :%v",name, response.Results)
		return response.Error()
	}else {
		return err
	}
}

func (pr *InfluxV1Storage) DropDB(name string) error {
	log.Infof("<ifv1> Dropping database %s",name)
	q := influx.NewQuery(fmt.Sprintf("DROP DATABASE %s", name), "", "")
	if response, err := pr.influxC.Query(q); err == nil {
		log.Infof("<tsdb> Database %s was dropped with status :%v",name, response.Results)
		return response.Error()
	}else {
		return err
	}
}


func (pr *InfluxV1Storage) UpdateRetentionPolicy(name,duration string) error {
	log.Info("<ifv1> Altering retention policy")
	var query = fmt.Sprintf("ALTER RETENTION POLICY %s ON %s DURATION %s", name, pr.dbName, duration)
	q := influx.NewQuery(query, pr.dbName, "s")
	if response, err := pr.influxC.Query(q); err == nil {
		log.Debug(response.Results)
		return  response.Error()
	}else {
		return err
	}
}

func (pr *InfluxV1Storage) AddRetentionPolicy(name,duration string) error {
	log.Info("<ifv1> Adding retention policy")
	var query = fmt.Sprintf("CREATE RETENTION POLICY %s ON %s DURATION %s REPLICATION 1", name, pr.dbName, duration)
	q := influx.NewQuery(query, pr.dbName, "s")
	if response, err := pr.influxC.Query(q); err == nil {
		log.Debug(response.Results)
		return  response.Error()
	}else {
		return err
	}

}

func (pr *InfluxV1Storage) DeleteRetentionPolicy(name string) error {
	log.Infof("Deleting retention policy %s",name)
	var query = fmt.Sprintf("DROP RETENTION POLICY %s ON %s", name, pr.dbName)
	q := influx.NewQuery(query, pr.dbName, "s")
	if response, err := pr.influxC.Query(q); err == nil  {
		log.Debug(response.Results)
		return response.Error()
	}else {
		log.Error("<ifv1> DeleteRetentionPolicy Error:",err.Error())
		return err
	}

}

func (pr *InfluxV1Storage) AddCQ(name,srcRetentionPolicy,targetRetentionPolicy,time string) error {
	log.Info("Adding retention policy")
	var query = fmt.Sprintf("CREATE CONTINUOUS QUERY \"%s\" ON \"%s\"\n" +
		"BEGIN\n " +
		"SELECT mean(*) INTO \"%s\".\"%s\".:MEASUREMENT FROM \"%s\".\"%s\"./.*/ GROUP BY time(%s),* \n" +
		"END", name, pr.dbName,pr.dbName,targetRetentionPolicy,pr.dbName,srcRetentionPolicy,time)
	log.Debugf("CQ query : %s",query)
	q := influx.NewQuery(query, pr.dbName, "s")
	if response, err := pr.influxC.Query(q); err == nil {
		log.Debug(response.Results)
		return response.Error()
	}else {
		return err

	}
}

func (pr *InfluxV1Storage) DeleteCQ(name string)error {
	log.Infof("Deleting CQ  %s",name)
	var query = fmt.Sprintf("DROP CONTINUOUS QUERY %s ON %s", name, pr.dbName)
	q := influx.NewQuery(query, pr.dbName, "s")
	if response, err := pr.influxC.Query(q); err == nil  {
		log.Debug(response.Results)
		return response.Error()
	}else {
		return err
	}
}

func (pr *InfluxV1Storage) DeleteMeasurement(name string)error {
	log.Infof("Deleting measurement %s",name)
	var query = fmt.Sprintf("DROP MEASUREMENT \"%s\" ", name)
	q := influx.NewQuery(query, pr.dbName, "s")
	if response, err := pr.influxC.Query(q); err == nil  {
		log.Debug(response.Results)
		return response.Error()
	}else {
		return err
	}

}

// Return list of measurements from db
func (pr *InfluxV1Storage) GetDbMeasurements() ([]string,error) {
	q := influx.NewQuery("SHOW MEASUREMENTS", pr.dbName, "ms")
	if response, err := pr.influxC.Query(q); err == nil  {
		//log.Debug(response.Results)
		if len(response.Results) > 0 {
			if len(response.Results[0].Series)>0 {
				var result []string
				for i := range response.Results[0].Series[0].Values {
					result = append(result,response.Results[0].Series[0].Values[i][0].(string))
				}
				return result,response.Error()
			}
		}
		return nil,response.Error()
	}else {
		return nil,err
	}
}

// Return list of measurements from db
func (pr *InfluxV1Storage) GetDbRetentionPolicies() ([]string,error) {
	q := influx.NewQuery("SHOW RETENTION POLICIES", pr.dbName, "ms")
	if response, err := pr.influxC.Query(q); err == nil {
		//log.Debug(response.Results)
		if len(response.Results) > 0 {
			if len(response.Results[0].Series)>0 {
				var result []string
				for i := range response.Results[0].Series[0].Values {
					result = append(result,response.Results[0].Series[0].Values[i][0].(string))
				}
				return result,response.Error()
			}
		}
		return nil,response.Error()
	}else {
		return nil, err
	}
}

func (pr *InfluxV1Storage)Close() {
	pr.influxC.Close()
}


