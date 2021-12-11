package main
//
//import (
//	"github.com/alivinco/thingsplex/model"
//	"github.com/alivinco/thingsplex/process/tsdb"
//	"os"
//	"bufio"
//	"strings"
//	"github.com/alivinco/fimpgo"
//	"io/ioutil"
//	"path/filepath"
//	"github.com/paulhammond/tai64"
//	log "github.com/Sirupsen/logrus"
//	"flag"
//	"fmt"
//)
//
//type TsdbBatchLoader struct{
//	proc *tsdb.Process
//}
//
//func NewTsdbBactchLoader(proc *tsdb.Process) *TsdbBatchLoader {
//	tl := TsdbBatchLoader{proc:proc}
//	return &tl
//}
//
//
//func (tl *TsdbBatchLoader)LoadAllFiles(path string) error {
//	dirFiles, err := ioutil.ReadDir(path)
//	if err != nil {
//		log.Error(err)
//		return err
//	}
//	for _, dirFile := range dirFiles {
//		fullPath := filepath.Join(path, dirFile.Name())
//		log.Info("Loading file : ",fullPath)
//		tl.loadMessagesFromFile(fullPath)
//	}
//	return nil
//}
//
//func (tl *TsdbBatchLoader)loadMessagesFromFile(path string)error{
//	file, err := os.Open(path)
//	if err != nil {
//		return err
//	}
//	defer file.Close()
//	bytesReader := bufio.NewReader(file)
//	scanner := bufio.NewScanner(bytesReader)
//	for scanner.Scan() {
//		tl.parseLineAndLoad(scanner.Text())
//	}
//	return nil
//}
//
//func (tl *TsdbBatchLoader)parseLineAndLoad(line string)error {
//	tString1 := strings.SplitN(line," pt:",2)
//	if len(tString1)>1 {
//		tString2 := strings.SplitN("pt:"+tString1[1]," {",2)
//		topic := tString2[0]
//		payload := "{"+tString2[1]
//		log.Info("Time : ",tString1[0])
//		log.Info("Topic : ",topic)
//		log.Info("Payload : ",payload)
//
//		rawPayload := []byte(payload)
//		fimpMsg,err := fimpgo.NewMessageFromBytes(rawPayload)
//
//		ctime , err := tai64.ParseTai64n(tString1[0])
//		if err != nil {
//			log.Error("Time is in wrong format")
//		}else {
//			fimpMsg.CreationTime = ctime
//		}
//
//
//		if err != nil {
//			return err
//		}
//		addr,err := fimpgo.NewAddressFromString(topic)
//		if err != nil {
//			return err
//		}
//		tl.proc.AddMessage(topic,addr,fimpMsg,fimpMsg.CreationTime)
//	}
//	return nil
//}
//
//func main() {
//	var mdir string
//	flag.StringVar(&mdir, "c", "", "Config file")
//	flag.Parse()
//	if mdir == "" {
//		mdir = "./"
//	} else {
//		fmt.Println("Loading mqtt log from ", mdir)
//	}
//	log.SetFormatter(&log.TextFormatter{FullTimestamp: true, ForceColors: true,TimestampFormat:"2006-01-02T15:04:05.999"})
//	log.SetLevel(log.DebugLevel)
//	configs := model.FimpUiConfigs{ProcConfigStorePath:"./"}
//	integr := tsdb.Boot(&configs,nil,nil)
//	proc := integr.GetProcessByID(1)
//	tsl := NewTsdbBactchLoader(proc)
//	//dir := "/Users/alivinco/DevProjects/APPS/Futurehome/zipgateway-debug/stianhub-mqtt/mqtt"
//	tsl.LoadAllFiles(mdir)
//	proc.Stop()
//
//}
