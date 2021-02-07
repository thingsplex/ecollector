package metadata

type ServiceMetaRec struct {
	Address    string
	DeviceID   int
	LocationID int
	DeviceType string // device type can be in format type.subtype
}

type MetadataStore interface {
	GetMetadataByAddress(address string) (ServiceMetaRec , error)
	Start() error
	Stop() error
	GetDevicesGroupedByType() (map[string][]string,error)
	GetDevicesGroupedByLocation() (map[string][]string, error)
}
