package metadata

import (
	"fmt"
	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype/primefimp"
	log "github.com/sirupsen/logrus"
	"strconv"
	"strings"
)

type VincMetadataStore struct {
	vApi *primefimp.ApiClient
	mqt   *fimpgo.MqttTransport
}

func NewVincMetadataStore(mqt *fimpgo.MqttTransport) MetadataStore {
	vs := &VincMetadataStore{mqt: mqt}
	return vs
}

func (r *VincMetadataStore) Start() error {
	r.vApi  = primefimp.NewApiClient("ecollector",r.mqt,false)
	r.vApi.StartNotifyRouter()

	return r.vApi.ReloadSiteToCache(30)
}

func (r *VincMetadataStore) Stop() error {
	r.vApi.Stop()
	return nil
}

func (r *VincMetadataStore) GetMetadataByAddress(topic string) (ServiceMetaRec , error) {
	if !strings.Contains(topic,"rt:dev") {
		return ServiceMetaRec{},fmt.Errorf("not device")
	}
	address := strings.Replace(topic,"pt:j1/mt:evt","",1)
	address = strings.Replace(address,"pt:j1/mt:cmd","",1)
	log.Tracef("Doing lookup of %s",address)
	if r.vApi == nil {
		return ServiceMetaRec{},fmt.Errorf("vApi is not initialized")
	}
	site,err := r.vApi.GetSite(true)
	if err != nil {
		return ServiceMetaRec{},err
	}
	mrec := ServiceMetaRec{}
	dev := site.GetDeviceByServiceAddress(address)
	if dev != nil {
		mrec.Address = address
		mrec.DeviceID = dev.ID
		mrec.DeviceType = r.composeType(dev)
		if dev.Room != nil {
			mrec.LocationID = *dev.Room
		}
	}else {
		log.Debugf("Device not found ")
	}
	return mrec,nil
}

func (r *VincMetadataStore) GetDevicesGroupedByLocation() (map[string][]string, error) {
	var result map[string][]string

	rooms , err := r.vApi.GetRooms(true)
	if err != nil {
		return nil, err
	}
	devices , err := r.vApi.GetDevices(true)
	if err != nil {
		return nil, err
	}

	for ri := range rooms {
		var devicesInRoom []string
		roomIdStr := strconv.Itoa(rooms[ri].ID)
		for di := range devices {
			if devices[di].Room == nil {
				continue
			}
			if rooms[ri].ID == *devices[di].Room {
				devicesInRoom = append(devicesInRoom,strconv.Itoa(devices[di].ID))
			}
		}
		if len(devicesInRoom)>0 {
			result[roomIdStr] = append(result[roomIdStr],strconv.Itoa(devices[ri].ID))
		}
	}
	return result,err
}

func (r *VincMetadataStore) GetDevicesGroupedByType() (map[string][]string,error) {
	var result map[string][]string
	devices , err := r.vApi.GetDevices(true)
	if err != nil {
		return nil, err
	}
	for di := range devices {
		devType := r.composeType(&devices[di])
		dev,ok := result[devType]
		if ok {
			dev = append(dev,strconv.Itoa(devices[di].ID))
		}else {
			result[devType] = []string{strconv.Itoa(devices[di].ID)}
		}
	}
	return result,err
}

func (r *VincMetadataStore) composeType(dev *primefimp.Device)string {
	ttype,ok1 := dev.Type["type"]
	if ok1 {
		stype , _ := ttype.(string)
		subtype,ok2 := dev.Type["subtype"]
		if ok2 {
			ssubtype,_ := subtype.(string)
			return fmt.Sprintf("%s.%s",stype,ssubtype)
		}
		return stype
	}
	return ""
}