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

func (pr *InfluxV1Storage) InitDefaultBuckets() {
	// CQ buckets
	pr.AddRetentionPolicy("gen_year","240w")  // 1-5 years from last 5 years
	pr.AddRetentionPolicy("gen_month","48w") // 1-12 month from last year
	pr.AddRetentionPolicy("gen_week","12w") // 1-4 weeks from last 3 month
	pr.AddRetentionPolicy("gen_day","2w")   // 1-3 days from last month
	// Default bucket for high frequency measurements
	pr.AddRetentionPolicy("gen_raw","2w")   // 1-2 days from last week
	// Default bucket for slow measurements
	pr.AddRetentionPolicy("gen_default","12w") //

	pr.DeleteRetentionPolicy("default_20w")
	pr.DeleteRetentionPolicy("default_8w")

	log.Info("Setting up CQ ")

	//pr.DeleteCQ("raw_to_day")
	//pr.DeleteCQ("day_to_week")
	//pr.DeleteCQ("week_to_month")
	//pr.DeleteCQ("month_to_year")

	pr.AddCQ("raw_to_day","gen_raw","gen_day","1m")
	pr.AddCQ("day_to_week","gen_day","gen_week","10m")
	pr.AddCQ("week_to_month","gen_week","gen_month","1h")
	pr.AddCQ("month_to_year","gen_month","gen_year","1d")
}

func (pr *InfluxV1Storage) InitSimpleBuckets() {
	pr.AddRetentionPolicy("gen_raw","240w")
	pr.AddRetentionPolicy("gen_default","240w")
}

func (pr *InfluxV1Storage) RunQuery(query string) *influx.Response {
	q := influx.NewQuery(query, pr.dbName, "s")
	if response, err := pr.influxC.Query(q); err == nil && response.Error() == nil {
		log.Trace(response.Results)
		return response
	}else {
		log.Error(response.Error())
		return response
	}
}
// GetDataPoints - relative must be in format 1m,1h,1d,1w . If groupByTime is empty , aggregate function will be skipped
func (pr *InfluxV1Storage) GetDataPoints(fieldName,measurement,relativeTime,fromTime,toTime,groupByTime,fillType, dataFunction,transformFunction, groupByTag string ) *influx.Response {
	// TODO : Add filters
	var retentionPolicyName,timeQuery,query string
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
		timeQuery = fmt.Sprintf("time >= %s AND time <= %s ",fromTime,toTime)
		//timeInterval,err = CalculateDuration(fromTime,toTime)
		//if err != nil {
		//	log.Error("<ifv1> Can't calculate duration.Err:",err.Error())
		//	return nil
		//}
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
	selector := ""
	if groupByTime == "" && groupByTag !="" {
		query = fmt.Sprintf("AS \"value\" FROM \"%s\".\"%s\" WHERE %s GROUP BY %s FILL(%s)",
			retentionPolicyName,measurement,timeQuery, groupByTag,fillType)
		selector = fmt.Sprintf("\"%s\"",fieldName)
	}else if groupByTime != "" && groupByTag =="" {
		query = fmt.Sprintf(" AS \"value\" FROM \"%s\".\"%s\" WHERE %s GROUP BY time(%s) FILL(%s)",
			retentionPolicyName,measurement,timeQuery,groupByTime,fillType)
		selector = fmt.Sprintf("%s(\"%s\")",dataFunction,fieldName)
	}else if groupByTime != "" && groupByTag !="" {
		query = fmt.Sprintf(" AS \"value\" FROM \"%s\".\"%s\" WHERE %s GROUP BY time(%s), %s FILL(%s)",
			retentionPolicyName,measurement,timeQuery,groupByTime, groupByTag,fillType)
		selector = fmt.Sprintf("%s(\"%s\")",dataFunction,fieldName)
	}else {
		if dataFunction != "" {
			query = fmt.Sprintf(" AS \"value\" FROM \"%s\".\"%s\" WHERE %s FILL(%s)",
				retentionPolicyName,measurement,timeQuery,fillType)
			selector = fmt.Sprintf("%s(\"%s\")",dataFunction,fieldName)
		}else {
			query = fmt.Sprintf(" AS \"value\" FROM \"%s\".\"%s\" WHERE %s FILL(%s)",
				retentionPolicyName,measurement,timeQuery,fillType)
			selector = fmt.Sprintf("\"%s\"",fieldName)
		}

	}
	if transformFunction != "" {
		selector = fmt.Sprintf("%s(%s)",transformFunction,selector)
	}

	query = fmt.Sprintf("SELECT %s %s",selector,query)
	log.Debug("<ifv1> --- Final query :",query)

	q := influx.NewQuery(query, pr.dbName, "s")
	if response, err := pr.influxC.Query(q); err == nil && response.Error() == nil {
		log.Trace(response.Results)
		return response
	}else {
		log.Error(response.Error())
		return response
	}

}

func (pr *InfluxV1Storage) WriteDataPoints(bp influx.BatchPoints) error {
	return pr.influxC.Write(bp)
}

func (pr *InfluxV1Storage) InitDB(name string) error {
	log.Info("<ifv1> Setting up database")
	q := influx.NewQuery(fmt.Sprintf("CREATE DATABASE %s", name), "", "")
	if response, err := pr.influxC.Query(q); err == nil && response.Error() == nil {
		log.Infof("<tsdb> Database %s was created with status :%s",name, response.Results)
		return nil
	}else {
		return err
	}
}

func (pr *InfluxV1Storage) UpdateRetentionPolicy(name,duration string) {
	log.Info("<ifv1> Altering retention policy")
	var query = fmt.Sprintf("ALTER RETENTION POLICY %s ON %s DURATION %s", name, pr.dbName, duration)
	q := influx.NewQuery(query, pr.dbName, "s")
	if response, err := pr.influxC.Query(q); err == nil && response.Error() == nil {
		log.Debug(response.Results)
	}else {
		log.Error(response.Error())
	}
}

func (pr *InfluxV1Storage) AddRetentionPolicy(name,duration string) {
	log.Info("<ifv1> Adding retention policy")
	var query = fmt.Sprintf("CREATE RETENTION POLICY %s ON %s DURATION %s REPLICATION 1", name, pr.dbName, duration)
	q := influx.NewQuery(query, pr.dbName, "s")
	if response, err := pr.influxC.Query(q); err == nil && response.Error() == nil {
		log.Debug(response.Results)
	}else {
		log.Error(response.Error())
	}

}

func (pr *InfluxV1Storage) DeleteRetentionPolicy(name string) {
	log.Infof("Deleting retention policy %s",name)
	var query = fmt.Sprintf("DROP RETENTION POLICY %s ON %s", name, pr.dbName)
	q := influx.NewQuery(query, pr.dbName, "s")
	if response, err := pr.influxC.Query(q); err == nil && response.Error() == nil {
		log.Debug(response.Results)
	}else {
		log.Error(response.Error())
	}

}

func (pr *InfluxV1Storage) AddCQ(name,srcRetentionPolicy,targetRetentionPolicy,time string) {
	log.Info("Adding retention policy")
	var query = fmt.Sprintf("CREATE CONTINUOUS QUERY \"%s\" ON \"%s\"\n" +
		"BEGIN\n " +
		"SELECT mean(*) INTO \"%s\".\"%s\".:MEASUREMENT FROM \"%s\".\"%s\"./.*/ GROUP BY time(%s),* \n" +
		"END", name, pr.dbName,pr.dbName,targetRetentionPolicy,pr.dbName,srcRetentionPolicy,time)
	log.Debugf("CQ query",query)
	q := influx.NewQuery(query, pr.dbName, "s")
	if response, err := pr.influxC.Query(q); err == nil && response.Error() == nil {
		log.Debug(response.Results)
	}else {
		log.Error(response.Error())
	}
}

func (pr *InfluxV1Storage) DeleteCQ(name string) {
	log.Infof("Deleting CQ  %s",name)
	var query = fmt.Sprintf("DROP CONTINUOUS QUERY %s ON %s", name, pr.dbName)
	q := influx.NewQuery(query, pr.dbName, "s")
	if response, err := pr.influxC.Query(q); err == nil && response.Error() == nil {
		log.Debug(response.Results)
	}else {
		log.Error(response.Error())
	}
}

func (pr *InfluxV1Storage) DeleteMeasurement(name string) {
	log.Infof("Deleting measurement %s",name)
	var query = fmt.Sprintf("DROP MEASUREMENT \"%s\" ", name)
	q := influx.NewQuery(query, pr.dbName, "s")
	if response, err := pr.influxC.Query(q); err == nil && response.Error() == nil {
		log.Debug(response.Results)
	}else {
		log.Error(response.Error())
	}

}

// Return list of measurements from db
func (pr *InfluxV1Storage) GetDbMeasurements() []string {
	q := influx.NewQuery("SHOW MEASUREMENTS", pr.dbName, "ms")
	if response, err := pr.influxC.Query(q); err == nil && response.Error() == nil {
		//log.Debug(response.Results)
		if len(response.Results) > 0 {
			if len(response.Results[0].Series)>0 {
				var result []string
				for i := range response.Results[0].Series[0].Values {
					result = append(result,response.Results[0].Series[0].Values[i][0].(string))
				}
				return result
			}
		}
		return nil
	}else {
		log.Error(err)
		log.Error(response.Error())
	}
	return nil
}

// Return list of measurements from db
func (pr *InfluxV1Storage) GetDbRetentionPolicies() []string {
	q := influx.NewQuery("SHOW RETENTION POLICIES", pr.dbName, "ms")
	if response, err := pr.influxC.Query(q); err == nil && response.Error() == nil {
		//log.Debug(response.Results)
		if len(response.Results) > 0 {
			if len(response.Results[0].Series)>0 {
				var result []string
				for i := range response.Results[0].Series[0].Values {
					result = append(result,response.Results[0].Series[0].Values[i][0].(string))
				}
				return result
			}
		}
		return nil
	}else {
		log.Error(err)
		log.Error(response.Error())
	}
	return nil
}

func (pr *InfluxV1Storage)Close() {
	pr.influxC.Close()
}