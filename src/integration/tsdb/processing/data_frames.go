package processing


/*
{
  "serv": "ecollector",
  "type": "evt.tsdb.data_points_report",
  "val_t": "object",
  "val": {
    "Results": [
      {
        "Series": [
          {
            "name": "electricity_meter_energy",
            "tags": {
              "dev_id": "100"
            },
            "columns": [
              "time",
              "value"
            ],
            "values": [
              [
                1611846000,
                1.8800000000010186
              ],
              [
                1611849600,
                1.4599999999991269
              ],
              [
                1611853200,
                1.2099999999991269
              ],
              [
                1611856800,
                1.0999999999985448
              ],
              [
                1611860400,
                0.9099999999998545
              ],
              [
                1611864000,
                0.8900000000030559
              ],
              [
                1611867600,
                1.0399999999972351
              ],
              [
                1611871200,
                1.0300000000024738
              ],
              [
                1611874800,
                0.8400000000001455
              ],
              [
                1611878400,
                0.6899999999986903
              ],
              [
                1611882000,
                0.5800000000017462
              ],
              [
                1611885600,
                0.5499999999992724
              ],
              [
                1611889200,
                0.5599999999976717
              ],
              [
                1611892800,
                0.5500000000029104
              ],
              [
                1611896400,
                0.5499999999992724
              ],
              [
                1611900000,
                0.5999999999985448
              ],
              [
                1611903600,
                0.9599999999991269
              ],
              [
                1611907200,
                0.9000000000014552
              ],
              [
                1611910800,
                0.9000000000014552
              ],
              [
                1611914400,
                0.8899999999994179
              ],
              [
                1611918000,
                1.1599999999998545
              ],
              [
                1611921600,
                1.319999999999709
              ],
              [
                1611925200,
                1.5
              ],
              [
                1611928800,
                1.6699999999982538
              ]
            ]
          },
          {
            "name": "electricity_meter_energy",
            "tags": {
              "dev_id": "112"
            },
            "columns": [
              "time",
              "value"
            ],
            "values": [
              [
                1611846000,
                0.006000000000000005
              ],
              [
                1611849600,
                0.006000000000000005
              ],
              [
                1611853200,
                0.006000000000000005
              ],
              [
                1611856800,
                0.006000000000000005
              ],
              [
                1611860400,
                0.007000000000000117
              ],
              [
                1611864000,
                0.006000000000000005
              ],
              [
                1611867600,
                0.006000000000000005
              ],
              [
                1611871200,
                0.005999999999999783
              ],
              [
                1611874800,
                0.0030000000000001137
              ],
              [
                1611878400,
                0
              ],
              [
                1611882000,
                0
              ],
              [
                1611885600,
                0
              ],
              [
                1611889200,
                0
              ],
              [
                1611892800,
                0
              ],
              [
                1611896400,
                0
              ],
              [
                1611900000,
                0.004999999999999893
              ],
              [
                1611903600,
                0.007000000000000117
              ],
              [
                1611907200,
                0.006000000000000005
              ],
              [
                1611910800,
                0.006000000000000005
              ],
              [
                1611914400,
                0.006000000000000005
              ],
              [
                1611918000,
                0.006000000000000005
              ],
              [
                1611921600,
                0.0020000000000000018
              ],
              [
                1611925200,
                0
              ],
              [
                1611928800,
                6.661338147750939e-16
              ]
            ]
          },
          {
            "name": "electricity_meter_energy",
            "tags": {
              "dev_id": "30"
            },
            "columns": [
              "time",
              "value"
            ],
            "values": [
              [
                1611849600,
                0
              ],
              [
                1611853200,
                0.10700552804098606
              ],
              [
                1611856800,
                0.0020032610219971048
              ],
              [
                1611860400,
                0
              ],
              [
                1611864000,
                0.041000366209999584
              ],
              [
                1611867600,
                0.03999328613301145
              ],
              [
                1611871200,
                0
              ],
              [
                1611874800,
                0
              ],
              [
                1611878400,
                0
              ],
              [
                1611882000,
                0
              ],
              [
                1611885600,
                0
              ],
              [
                1611889200,
                0.0010070800780113132
              ],
              [
                1611892800,
                0
              ],
              [
                1611896400,
                0
              ],
              [
                1611900000,
                0.08699035644599462
              ],
              [
                1611903600,
                0.03401184081999986
              ],
              [
                1611907200,
                0
              ],
              [
                1611910800,
                0
              ],
              [
                1611914400,
                0
              ],
              [
                1611918000,
                0
              ],
              [
                1611921600,
                0
              ],
              [
                1611925200,
                0.0009918212890056566
              ],
              [
                1611928800,
                0
              ]
            ]
          }
        ],
        "Messages": null
      }
    ]
  },
  "props": null,
  "tags": null,
  "src": "tplex-ui",
  "ver": "1",
  "uid": "51b1d8e6-45d8-4ecc-8f5e-41137eb815bd",
  "topic": "pt:j1/mt:rsp/rt:app/rn:tplex-ui/ad:1"
}
 */

//type MergedRow struct {
//	time int
//	val  float64
//	//devId string
//	groupName string
//}
//
//
//type EcDataFrame struct {
//	qFrame qframe.QFrame
//	devicesInGroup map[string][]string
//}
//
//func (df *EcDataFrame) Frame() qframe.QFrame {
//	return df.qFrame
//}
//
//func (df *EcDataFrame) SetFrame(qFrame qframe.QFrame) {
//	df.qFrame = qFrame
//}
//
//func NewEcDataFrame() *EcDataFrame {
//	return &EcDataFrame{}
//}
//
//func (df *EcDataFrame) LoadFromInfluxResponse(result client.Result,tagName string, devicesInGroup map[string][]string)  {
//	// merge all series into flat table
//	var mergedResult []MergedRow
//	var rowCounter uint32
//	df.devicesInGroup = devicesInGroup
//	for si := range result.Series {
//		for groupName,devsInGroup := range devicesInGroup { // regrouping records
//			for _, devInGroup := range devsInGroup {
//				if result.Series[si].Tags[tagName] == devInGroup {
//					for vi := range result.Series[si].Values {
//						time := int(result.Series[si].Values[vi][0].(float64))
//						value := result.Series[si].Values[vi][1].(float64)
//						mergedResult = append(mergedResult,MergedRow{
//							time:  time,
//							val:   value,
//							//devId: result.Series[si].Tags[tagName],
//							groupName: groupName,
//						})
//						rowCounter++
//					}
//				}
//			}
//		}
//	}
//
//	//  Creating data frame
//	timeCol := make([]int,0,rowCounter)
//	//devIdCol := make([]string,0,rowCounter)
//	valCol := make([]float64,0,rowCounter)
//	groupNameCol := make([]string,0,rowCounter)
//
//	for mi := range mergedResult {
//		timeCol = append(timeCol,mergedResult[mi].time)
//		//devIdCol = append(devIdCol,mergedResult[mi].devId)
//		valCol = append(valCol,mergedResult[mi].val)
//		groupNameCol = append(groupNameCol,mergedResult[mi].groupName)
//	}
//
//	df.qFrame = qframe.New( map[string]interface{}{"time": timeCol,"val":valCol,"group":groupNameCol})
//}
//
//// AggregateByGroupAndTime aggregates data by group and time using "sum" function
//func (df *EcDataFrame) AggregateByGroupAndTime() error {
//	intSum := func(xx []float64) float64 {
//		var result float64
//		for _, x := range xx {
//			result += x
//		}
//		return result
//	}
//	// Creating new frame that contains grouped and ordered data
//	df.qFrame = df.qFrame.GroupBy(groupby.Columns("time","group")).Aggregate(qframe.Aggregation{Fn: intSum, Column: "val"}).Sort(qframe.Order{Column: "time"})
//	return nil
//}
//
//// AggregateByGroup aggregates data by group using "sum" function
//func (df *EcDataFrame) AggregateByGroup() error {
//	intSum := func(xx []float64) float64 {
//		var result float64
//		for _, x := range xx {
//			result += x
//		}
//		return result
//	}
//	// Creating new frame that contains grouped and ordered data
//	df.qFrame = df.qFrame.GroupBy(groupby.Columns("group")).Aggregate(qframe.Aggregation{Fn: intSum, Column: "val"})
//	return nil
//}
//
//func (df *EcDataFrame) GetInfluxSeries() (*client.Result,error) {
//	var series []models.Row
//
//	for group,_ := range df.devicesInGroup {
//		newF := df.qFrame.Filter(qframe.Filter{Column: "group", Comparator: "=", Arg: group})
//		timeView,err := newF.IntView("time")
//		if err != nil {
//			return nil, err
//		}
//		valView,err := newF.FloatView("val")
//		if err != nil {
//			return nil, err
//		}
//
//		var values [][]interface{}
//		for iv:=0;iv<newF.Len();iv++ {
//			value := []interface{}{timeView.ItemAt(iv),valView.ItemAt(iv)}
//			values = append(values,value)
//		}
//
//		row := models.Row{
//			Name:    "",
//			Tags: map[string]string{"group":group},
//			Columns: []string{"time","value"},
//			Values:  values,
//			Partial: false,
//		}
//		series = append(series,row)
//
//	}
//
//	result := &client.Result{
//		Series:   series,
//		Messages: nil,
//		Err:      "",
//	}
//	return result,nil
//}
//
//func (df *EcDataFrame) SaveToCSVFile(fileName string) error {
//	f, err := os.Create(fileName)
//	if err != nil {
//		return err
//	}
//	return df.qFrame.ToCSV(f)
//}