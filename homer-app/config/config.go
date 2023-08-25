package config

var Setting HomerSettingServer

type HomerSettingServer struct {
	IsolateQuery string `default:""`
	IsolateGroup string `default:""`

	GRAFANA_SETTINGS struct {
		URL      string `default:"http://grafana/"`
		AuthKey  string `default:""`
		User     string `default:""`
		Password string `default:""`
		Path     string `default:"/grafana"`
		Enable   bool   `default:"false"`
	}

	TRANSACTION_SETTINGS struct {
		DedupModel        string `default:"message-ip-pair"`
		GlobalDeduplicate bool   `default:"false"`
	}

	SbcIdMap           map[string]string
	OriginSbcIpMap     map[string]string
	OriginVoipRegIpMap map[string]string
}
