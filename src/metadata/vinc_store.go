package metadata

import (
	"fmt"
	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype/primefimp"
	log "github.com/sirupsen/logrus"
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

	return r.vApi.ReloadSiteToCache(5)
}


func (r *VincMetadataStore) GetMetadataByAddress(topic string) (ServiceMetaRec , error) {
	address := strings.Replace(topic,"pt:j1/mt:evt","",1)
	address = strings.Replace(address,"pt:j1/mt:cmd","",1)
	log.Debugf("Doing lookup of %s",address)
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