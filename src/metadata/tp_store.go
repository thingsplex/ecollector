package metadata

import (
	"errors"
	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
	"strings"
)

type TpRegService struct {
	ID                  int    `json:"id"`
	Address             string `json:"address"`
	IntegrationId       string `json:"integr_id" `
	ParentContainerId   int    `json:"container_id" `
	ParentContainerType string `json:"container_type" `
	LocationId          int    `json:"location_id" `
}

type TpMetadataStore struct {
	store []ServiceMetaRec
	mqt   *fimpgo.MqttTransport
}

func NewTpMetadataStore(mqt *fimpgo.MqttTransport) MetadataStore {
	return &TpMetadataStore{mqt: mqt}
}



func (sm *TpMetadataStore) Start() error {
	log.Info("Loading metadata from TpRegistry")
	respTopic := "pt:j1/mt:rsp/rt:app/rn:ecollector/ad:1"
	sClient := fimpgo.NewSyncClient(sm.mqt)
	val := map[string]string{"filter_without_alias": "", "location_id": "", "service_name": "", "thing_id": "*"}
	req := fimpgo.NewStrMapMessage("cmd.registry.get_services", "tpflow", val, nil, nil, nil)
	req.ResponseToTopic = respTopic
	sClient.AddSubscription(respTopic)
	resp, err := sClient.SendFimp("pt:j1/mt:cmd/rt:app/rn:tpflow/ad:1", req, 20)
	sClient.RemoveSubscription(respTopic)
	if err != nil {
		log.Error("Error while loading metadata from TpRegistry")
		return err
	}
	var serv []TpRegService
	resp.GetObjectValue(&serv)
	sm.store = []ServiceMetaRec{}
	var i int
	for i = range serv {
		rec := ServiceMetaRec{DeviceID: serv[i].ParentContainerId, LocationID: serv[i].LocationId,Address:serv[i].Address}
		sm.store = append(sm.store, rec)
	}
	log.Debug("Number of entries loaded to the store ", i)
	return nil
}

func (sm *TpMetadataStore) Stop() error {
	return nil
}

func (sm *TpMetadataStore) GetMetadataByAddress(address string) (ServiceMetaRec , error) {
	address = strings.Replace(address,"pt:j1/mt:evt","",1)
	address = strings.Replace(address,"pt:j1/mt:cmd","",1)
	for i := range sm.store {
		if address == sm.store[i].Address {
			return sm.store[i],nil
		}
	}
	return ServiceMetaRec{},errors.New("not found")
}

func (sm *TpMetadataStore) GetDevicesGroupedByType() (map[string][]string,error) {
	return nil, nil
}

func (sm *TpMetadataStore) GetDevicesGroupedByLocation() (map[string][]string, error) {
	return nil, nil
}