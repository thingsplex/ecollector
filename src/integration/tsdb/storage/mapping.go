package storage

import (
	"strconv"
	"strings"
	"time"
)

const (
	DayDuration = time.Hour * 24
	WeekDuration = DayDuration * 7
	MonthDuration = WeekDuration * 4
)


func ParseTime(val string)(time.Time,error) {
	return time.Parse(time.RFC3339, val)
}

func ResolveRetentionName(fromTime,toTime string) (string,error) {
	from , errFrom := ParseTime(fromTime)
	if errFrom != nil {
		return "", errFrom
	}
	timeSinceNow:= time.Now().Sub(from)
	return ResolveRetentionByElapsedTimeDuration(timeSinceNow),nil
}
// ResolveRetentionByElapsedTimeDuration converts duration into retention policy. Used for query operations.
func ResolveRetentionByElapsedTimeDuration(timeSinceNow time.Duration) string {
	switch  {
	case timeSinceNow > 12*MonthDuration:
		return "gen_year"
	case timeSinceNow > 1*MonthDuration:
		return "gen_month"
	case timeSinceNow > 1*WeekDuration:
		return "gen_week"
	case timeSinceNow > 1*DayDuration:
		return "gen_day"
	default:
		return "gen_raw"
	}
}

// ResolveFieldFullName converts field name into field name withing retention policy.These unusual field names are result of Influx down sampling function.
func ResolveFieldFullName(name,retentionPolicyName string) string  {
	switch retentionPolicyName {
	case "gen_day":
		return "mean_"+name
	case "gen_week":
		return "mean_mean_"+name
	case "gen_month":
		return "mean_mean_mean_"+name
	case "gen_year":
		return "mean_mean_mean_mean_"+name
	default:
		return name
	}
}

func GetRetentionTimeGroupDuration(name string ) time.Duration {
	switch name {
	case "gen_day":
		return 1*time.Minute
	case "gen_week":
		return 10*time.Minute
	case "gen_month":
		return 1 * time.Hour
	case "gen_year":
		return 1 * DayDuration

	}
	return 0
}

// 0    10min
// 1m
// if  retention duration > target
// return

func ResolveRetentionByTimeGroup(timeGroup string) string {
	//pr.AddCQ("raw_to_day","gen_raw","gen_day","1m")
	//pr.AddCQ("day_to_week","gen_day","gen_week","10m")
	//pr.AddCQ("week_to_month","gen_week","gen_month","1h")
	//pr.AddCQ("month_to_year","gen_month","gen_year","1d")
	d := ResolveDurationFromRelativeTime(timeGroup)
	switch  {
	case d >= 1*DayDuration:
		return "gen_year"
	case d >= 1*time.Hour:
		return "gen_month"
	case d >= 10 * time.Minute:
		return "gen_week"
	case d >= 1*time.Minute :
		return "gen_day"
	default:
		return "gen_raw"
	}
}

// ResolveDurationFromRelativeTime Relative time must be in format Xh,Xd,Xw
func ResolveDurationFromRelativeTime(rTime string) time.Duration {
	var d int
	if strings.Contains(rTime, "h") {
		d,_ = strconv.Atoi(strings.ReplaceAll(rTime,"h",""))
		return time.Duration(d)*time.Hour
	}else if strings.Contains(rTime, "d") {
		d,_ = strconv.Atoi(strings.ReplaceAll(rTime,"d",""))
		return time.Duration(d)*DayDuration
	}else if strings.Contains(rTime, "m") {
		d,_ = strconv.Atoi(strings.ReplaceAll(rTime,"m",""))
		return time.Duration(d)*time.Minute
	}else if strings.Contains(rTime, "w") {
		d,_ = strconv.Atoi(strings.ReplaceAll(rTime,"w",""))
		return time.Duration(d)*WeekDuration
	}
	return 0
}

func CalculateGroupByTimeByInterval(duration time.Duration) string {
	//switch {
	//case duration > 12 * MonthDuration:
	//	return "1h"
	//
	//}
	return ""
}

func CalculateDuration(fromTime,toTime string) (time.Duration,error) {
	from , errFrom := ParseTime(fromTime)
	if errFrom != nil {
		return 0, errFrom
	}
	to , errTo := ParseTime(toTime)
	if errFrom != nil {
		return 0, errTo
	}
	return from.Sub(to),nil
}

// ResolveWriteRetentionPolicyName converts measurement into retention policy
func ResolveWriteRetentionPolicyName(mName string ) string {
	if mName == "electricity_meter_energy_sampled" {
		return "gen_year"
	}
	if IsHighFrequencyData(mName){
		return "gen_raw"
	}
	return "gen_default"
}

func IsHighFrequencyData(measurementName string) bool {
	if measurementName == "electricity_meter_power" ||
		measurementName == "electricity_meter_energy" ||
		measurementName == "electricity_meter_ext" ||
		measurementName == "electricity_meter_energy_sampled" ||
		strings.Contains(measurementName,"sensor_") {
			if strings.Contains(measurementName,"sensor_presence") || strings.Contains(measurementName,"sensor_contact") {
				return false
			}
			return true
	}
	return false
}

