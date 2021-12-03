package metadata

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"strings"
)

type FileMetadataStore struct {
	metadataFilePath string
	store            []ServiceMetaRec
}

func NewFileMetadataStore(metadataFilePath string) MetadataStore {
	vs := &FileMetadataStore{metadataFilePath: metadataFilePath}
	return vs
}

func (r *FileMetadataStore) Start() error {
	file, err := ioutil.ReadFile(r.metadataFilePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(file, &r.store)
}

func (r *FileMetadataStore) Stop() error {
	return nil
}

func (r *FileMetadataStore) GetMetadataByAddress(topic string) (ServiceMetaRec, error) {
	if !strings.Contains(topic, "rt:dev") {
		return ServiceMetaRec{}, fmt.Errorf("not device")
	}
	address := strings.Replace(topic, "pt:j1/mt:evt", "", 1)
	address = strings.Replace(address, "pt:j1/mt:cmd", "", 1)
	log.Tracef("Doing lookup of %s", address)

	for i := range r.store {
		if r.store[i].Address == address {
			return r.store[i], nil
		}
	}
	return ServiceMetaRec{}, nil
}

func (r *FileMetadataStore) GetDevicesGroupedByLocation() (map[string][]string, error) {
	var result map[string][]string
	return result, nil
}

func (r *FileMetadataStore) GetDevicesGroupedByType() (map[string][]string, error) {
	var result map[string][]string

	return result, nil
}
