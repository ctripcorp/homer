package model

import (
	"container/list"
	"encoding/json"
	"time"
)

// swagger:model SearchCallData
type SearchObject struct {
	// required: true
	Param struct {
		Transaction struct {
		} `json:"transaction"`
		// this controls the number of records to display
		// example: 200
		// required: true
		Limit int `json:"limit"`
		// this control the type of search one can perform
		// type: string
		// format: binary
		// example: `{"1_call":[{"name":"limit","value":"10","type":"string","hepid":1}]}`
		Search   json.RawMessage `json:"search"`
		Location struct {
			Node []string `json:node`
		} `json:"location"`
		// timezone settings
		// type: object
		// default: null
		Timezone struct {
			Value int    `json:"value"`
			Name  string `json:"name"`
		} `json:"timezone"`
	} `json:"param"`
	// this control the time range for used for search
	Timestamp struct {
		// current timestamp in miliseconds
		// required :true
		// example: 1581793200000
		From int64 `json:"from"`
		// current timestamp in miliseconds
		// required :true
		// example: 1581879599000
		To int64 `json:"to"`
	} `json:"timestamp"`
}

type HepTable struct {
	Id             int64           `json:"id"`
	Sid            string          `json:"sid"`
	CreatedDate    time.Time       `gorm:"column:create_date" json:"create_date"`
	ProtocolHeader json.RawMessage `gorm:"column:protocol_header" json:"protocol_header"`
	DataHeader     json.RawMessage `gorm:"column:data_header" json:"data_header"`
	Raw            string          `gorm:"column:raw" json:"raw"`
	DBNode         string          `gorm:"column:-" json:"dbnode"`
	Node           string          `gorm:"column:-" json:"node"`
	Profile        string          `gorm:"column:-" json:"profile"`
}

type TransactionTable struct {
	ViaBranch string               `json:"via_branch"`
	Name      string               `json:"name"`
	ErrorName string               `json:"error_name"`
	BeginDate time.Time            `json:"begin_date"`
	FromUser  string               `json:"from_user"`
	ToUser    string               `json:"to_user"`
	TMethods  []*TransactionMethod `json:"t_methods"`
}

type TransactionMethod struct {
	Name      string     `json:"name"`
	CSeq      string     `json:"cseq"`
	BeginDate time.Time  `json:"begin_date"`
	BodyList  *list.List `json:"body_list"`
}

type Message struct {
	Id     int            `json:"id"`
	Sid    string         `json:"sid"`
	ProtoH ProtocolHeader `json:"protocol_header"`
	DataH  DataHeader     `json:"data_header"`
	Raw    string         `json:"raw"`
}

type ProtocolHeader struct {
	DstIP          string `json:"dstIp"`
	SrcIP          string `json:"srcIp"`
	DstPort        int    `json:"dstPort"`
	SrcPort        int    `json:"srcPort"`
	Protocol       int    `json:"protocol"`
	CaptureID      string `json:"captureId"`
	CapturePass    string `json:"capturePass"`
	PayloadType    int    `json:"payloadType"`
	TimeSeconds    int    `json:"timeSeconds"`
	TimeUseconds   int    `json:"timeUseconds"`
	correlationId  string `json:"correlation_id"`
	ProtocolFamily int    `json:"protocolFamily"`
}

type DataHeader struct {
	CallID     string `json:"callid"`
	CSeq       string `json:"cseq"`
	Method     string `json:"method"`
	FromUser   string `json:"from_user"`
	ToUser     string `json:"to_user"`
	FromTag    string `json:"from_tag"`
	ToTag      string `json:"to_tag"`
	PidUser    string `json:"pid_user"`
	AuthUser   string `json:"auth_user"`
	RuriUser   string `json:"ruri_user"`
	UserAgent  string `json:"user_agent"`
	RuriDomain string `json:"ruri_domain"`
	ViaBranch  string `json:"via_branch"`
}

type TransactionResponse struct {
	Total int        `json:"total"`
	D     SecondData `json:"data"`
}

type SecondData struct {
	Messages []Message `json:"messages"`
}

type CallElement struct {
	ID          int64  `json:"id"`
	Sid         string `json:"sid"`
	DstHost     string `json:"dstHost"`
	SrcHost     string `json:"srcHost"`
	DstID       string `json:"dstId"`
	SrcID       string `json:"srcId"`
	SrcIP       string `json:"srcIp"`
	DstIP       string `json:"dstIp"`
	SrcPort     int    `json:"srcPort"`
	DstPort     int    `json:"dstPort"`
	Method      string `json:"method"`
	MethodText  string `json:"method_text"`
	CreateDate  int64  `json:"create_date"`
	Protocol    int    `json:"protocol"`
	MsgColor    string `json:"msg_color"`
	Table       string `json:"table"`
	RuriUser    string `json:"ruri_user"`
	Destination int    `json:"destination"`
	MicroTs     int64  `json:"micro_ts"`
}

type TransactionElement struct {
	ViaBranch string        `json:"via_branch"`
	Name      string        `json:"name"`
	ErrorName string        `json:"error_name"`
	BeginDate int64         `json:"begin_date"`
	FromUser  string        `json:"from_user"`
	ToUser    string        `json:"to_user"`
	Host      []string      `json:"host"`
	CallData  []CallElement `json:"call_data"`
}

type SbcOptionData struct {
	SbcIp        string `json:"sbc_ip"`
	DelayMS      int64  `json:"delay_ms"`
	TimeoutCount int    `json:"timeout_count"`
	TotalCount   int    `json:"total_count"`
	Method       string `json:"method"`
}

type DIRECTION int32

const (
	E_SBC_2_REG   = DIRECTION(101)
	E_SBC_2_PHONE = DIRECTION(102)
	E_REG_2_SBC   = DIRECTION(103)
)

type PHONE_TYPE int32

const (
	E_PHONE_TYPE_INVALID = PHONE_TYPE(-1)
	E_PHONE_TYPE_HARD    = PHONE_TYPE(101)
	E_PHONE_TYPE_SOFT    = PHONE_TYPE(102)
)

const (
	E_NOTIFY_INVALID   = int32(-1)
	E_NOTIFY_MAKE_CALL = int32(10)
	E_NOTIFY_ANSWER    = int32(11)
)

type SbcInvite100Data struct {
	SbcIp        string    `json:"sbc_ip"`
	DelayMS      int64     `json:"delay_ms"`
	TimeoutCount int       `json:"timeout_count"`
	TotalCount   int       `json:"total_count"`
	Direction    DIRECTION `json:"direction"`
}

type RegInvite100Data struct {
	RegIp        string     `json:"reg_ip"`
	PhoneType    PHONE_TYPE `json:"phone_type"`
	DelayMS      int64      `json:"delay_ms"`
	TimeoutCount int        `json:"timeout_count"`
	TotalCount   int        `json:"total_count"`
}

type CMNotifyData struct {
	CmIp         string `json:"cm_ip"`
	DelayMS      int64  `json:"delay_ms"`
	TimeoutCount int    `json:"timeout_count"`
	TotalCount   int    `json:"total_count"`
	Method       string `json:"method"`
	NotifyType   int32  `json:"notify_type"`
}

type SbcNotifyData struct {
	SbcIp        string `json:"sbc_ip"`
	DelayMS      int64  `json:"delay_ms"`
	TimeoutCount int    `json:"timeout_count"`
	TotalCount   int    `json:"total_count"`
	Method       string `json:"method"`
	NotifyType   int32  `json:"notify_type"`
}
