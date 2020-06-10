package tsdb

import (
	"fmt"
	influx "github.com/influxdata/influxdb1-client/v2"
	log "github.com/sirupsen/logrus"
)

func (pr *Process) RunQuery(query string) *influx.Response {
	q := influx.NewQuery(query, pr.Config.InfluxDB, "s")
	if response, err := pr.influxC.Query(q); err == nil && response.Error() == nil {
		log.Trace(response.Results)
		return response
	}else {
		log.Error(response.Error())
		return response
	}
}

func (pr *Process) UpdateRetentionPolicy(name,duration string) {
	log.Info("Altering retention policy")
	var query = fmt.Sprintf("ALTER RETENTION POLICY %s ON %s DURATION %s", name, pr.Config.InfluxDB, duration)
	q := influx.NewQuery(query, pr.Config.InfluxDB, "s")
	if response, err := pr.influxC.Query(q); err == nil && response.Error() == nil {
		log.Debug(response.Results)
	}else {
		log.Error(response.Error())
	}
}

func (pr *Process) AddRetentionPolicy(name,duration string) {
	log.Info("Adding retention policy")
	var query = fmt.Sprintf("CREATE RETENTION POLICY %s ON %s DURATION %s REPLICATION 1", name, pr.Config.InfluxDB, duration)
	q := influx.NewQuery(query, pr.Config.InfluxDB, "s")
	if response, err := pr.influxC.Query(q); err == nil && response.Error() == nil {
		log.Debug(response.Results)
	}else {
		log.Error(response.Error())
	}

}

func (pr *Process) DeleteRetentionPolicy(name string) {
	log.Infof("Deleting retention policy %s",name)
	var query = fmt.Sprintf("DROP RETENTION POLICY %s ON %s", name, pr.Config.InfluxDB)
	q := influx.NewQuery(query, pr.Config.InfluxDB, "s")
	if response, err := pr.influxC.Query(q); err == nil && response.Error() == nil {
		log.Debug(response.Results)
	}else {
		log.Error(response.Error())
	}

}

func (pr *Process) AddCQ(name,srcRetentionPolicy,targetRetentionPolicy,time string) {
	log.Info("Adding retention policy")
	var query = fmt.Sprintf("CREATE CONTINUOUS QUERY \"%s\" ON \"%s\"\n" +
		"BEGIN\n " +
		"SELECT mean(*) INTO \"%s\".\"%s\".:MEASUREMENT FROM \"%s\".\"%s\"./.*/ GROUP BY time(%s),* \n" +
		"END", name, pr.Config.InfluxDB,pr.Config.InfluxDB,targetRetentionPolicy,pr.Config.InfluxDB,srcRetentionPolicy,time)
	log.Debugf("CQ query",query)
	q := influx.NewQuery(query, pr.Config.InfluxDB, "s")
	if response, err := pr.influxC.Query(q); err == nil && response.Error() == nil {
		log.Debug(response.Results)
	}else {
		log.Error(response.Error())
	}
}

func (pr *Process) DeleteCQ(name string) {
	log.Infof("Deleting CQ  %s",name)
	var query = fmt.Sprintf("DROP CONTINUOUS QUERY %s ON %s", name, pr.Config.InfluxDB)
	q := influx.NewQuery(query, pr.Config.InfluxDB, "s")
	if response, err := pr.influxC.Query(q); err == nil && response.Error() == nil {
		log.Debug(response.Results)
	}else {
		log.Error(response.Error())
	}
}

func (pr *Process) DeleteMeasurement(name string) {
	log.Infof("Deleting measurement %s",name)
	var query = fmt.Sprintf("DROP MEASUREMENT \"%s\" ", name)
	q := influx.NewQuery(query, pr.Config.InfluxDB, "s")
	if response, err := pr.influxC.Query(q); err == nil && response.Error() == nil {
		log.Debug(response.Results)
	}else {
		log.Error(response.Error())
	}

}

// Return list of measurements from db
func (pr *Process) GetDbMeasurements() []string {
	q := influx.NewQuery("SHOW MEASUREMENTS", pr.Config.InfluxDB, "ms")
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
func (pr *Process) GetDbRetentionPolicies() []string {
	q := influx.NewQuery("SHOW RETENTION POLICIES", pr.Config.InfluxDB, "ms")
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