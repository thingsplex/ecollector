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
	return ResolveRetentionByDuration(timeSinceNow),nil
}
// Converts duration into retention policy .
func ResolveRetentionByDuration(timeSinceNow time.Duration) string {
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

// Relative time must be in format Xh,Xd,Xw
func GetDurationFromRelativeTime(rTime string) time.Duration {
	var d int
	if strings.Contains(rTime, "h") {
		d,_ = strconv.Atoi(strings.ReplaceAll(rTime,"h",""))
		return time.Duration(d)*time.Hour
	}else if strings.Contains(rTime, "d") {
		d,_ = strconv.Atoi(strings.ReplaceAll(rTime,"d",""))
		return time.Duration(d)*DayDuration
	}else if strings.Contains(rTime, "m") {
		d,_ = strconv.Atoi(strings.ReplaceAll(rTime,"m",""))
		return time.Duration(d)*MonthDuration
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

func ResolveWriteRetentionPolicyName(mName string ) string {
	if IsHighFrequencyData(mName){
		return "gen_raw"
	}
	return "gen_default"
}

func IsHighFrequencyData(measurementName string) bool {
	if measurementName == "electricity_meter_power" ||
		measurementName == "electricity_meter_energy" ||
		measurementName == "electricity_meter_ext" ||
		strings.Contains(measurementName,"sensor_") {
			if strings.Contains(measurementName,"sensor_presence") || strings.Contains(measurementName,"sensor_contact") {
				return false
			}
			return true
	}
	return false
}