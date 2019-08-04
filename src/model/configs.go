package model


type Configs struct {
	InstanceAddress       string `json:"instance_address"`
	MqttServerURI         string `json:"mqtt_server_uri"`
	MqttUsername          string `json:"mqtt_server_username"`
	MqttPassword          string `json:"mqtt_server_password"`
	MqttClientIdPrefix    string `json:"mqtt_client_id_prefix"`
	LogFile               string `json:"log_file"`
	LogLevel              string `json:"log_level"`
	LogFormat             string `json:"log_format"`
	ProcConfigStorePath   string `json:"proc_config_store_path"`
	InitDb                bool   `json:"init_db"`
	VincHost              string `json:"vinc_host"`
}
