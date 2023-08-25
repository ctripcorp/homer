package service

import (
	"bytes"
	"container/list"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/dop251/goja"
	"github.com/shomali11/util/xconditions"
	"github.com/sipcapture/homer-app/config"
	"github.com/sipcapture/homer-app/model"
	"github.com/sipcapture/homer-app/utils/exportwriter"
	"github.com/sipcapture/homer-app/utils/heputils"
	"github.com/sipcapture/homer-app/utils/logger/function"
	"github.com/sirupsen/logrus"
)

//search Service
type SearchService struct {
	ServiceData
}

//external decoder
type ExternalDecoder struct {
	Binary    string   `json:"binary"`
	Param     string   `json:"param"`
	Protocols []string `json:"protocols"`
	UID       uint32   `json:"uid"`
	GID       uint32   `json:"gid"`
	Active    bool     `json:"active"`
}

func executeJSInputFunction(jsString string, callIds []interface{}) []interface{} {

	vm := goja.New()

	//jsString := "var returnData=[]; for (var i = 0; i < data.length; i++) { returnData.push(data[i]+'_b2b-1'); }; returnData;"
	// "input_function_js": "var returnData=[]; for (var i = 0; i < data.length; i++) { returnData.push(data[i]+'_b2b-1'); }; returnData;"

	logrus.Debug("Inside JS script: Callids: ", callIds)
	logrus.Debug("Script: ", jsString)

	vm.Set("scriptPrintf", ScriptPrintf)
	vm.Set("data", callIds)

	v, err := vm.RunString(jsString)
	if err != nil {
		logrus.Errorln("Javascript Script error:", err)
		return nil
	}

	data := v.Export().([]interface{})

	logrus.Debug("Inside JS output data: ", data)

	return data
}

func ScriptPrintf(val interface{}) {

	logrus.Debug("script:", val)
}

func executeJSOutputFunction(jsString string, dataRow []model.HepTable) []model.HepTable {

	vm := goja.New()

	//logrus.Debug("Inside JS script: Callids: ", dataRow)
	logrus.Debug("Script: ", jsString)
	vm.Set("scriptPrintf", ScriptPrintf)
	marshalData, _ := json.Marshal(dataRow)
	sData, _ := gabs.ParseJSON(marshalData)
	vm.Set("data", sData.Data())

	v, err := vm.RunString(jsString)
	if err != nil {
		logrus.Errorln("Javascript Script error:", err)
		return nil
	}

	returnData := []model.HepTable{}

	data := v.Export().([]interface{})

	marshalData, _ = json.Marshal(data)

	err = json.Unmarshal(marshalData, &returnData)
	if err != nil {
		logrus.Errorln("Couldnt unmarshal:", err)
		return nil
	}

	return returnData
}

const (
	// enum
	FIRST  = 1
	VALUE  = 2
	EQUALS = 3
	END    = 4
	RESET  = 10
)

func buildQuery(elems []interface{}) (sql string, sLimit int) {
	sLimit = 200
	for k, v := range elems {
		mapData := v.(map[string]interface{})
		if formVal, ok := mapData["value"]; ok {
			formValue := formVal.(string)
			formValue = strings.TrimSpace(formValue)
			formName := mapData["name"].(string)
			formType := mapData["type"].(string)

			if formName == "smartinput" {

				upperCaseValue := strings.ToUpper(formValue)
				totalLen := len(formValue)
				index := 0
				key := ""
				value := ""
				typeValue := "string"
				operator := ""
				endEl := " AND ("

				hdr := FIRST

				for i := 0; i < totalLen; i++ {

					switch {
					case hdr == FIRST:
						/* = */
						if upperCaseValue[i] == '=' {
							hdr = VALUE
							operator = " = "
							key = strings.Replace(formValue[index:i], " ", "", -1)
							value = ""
							index = i
							/* formValue != */
						} else if upperCaseValue[i] == '!' && upperCaseValue[(i+1)] == '=' {
							hdr = VALUE
							operator = " != "
							key = strings.Replace(formValue[index:i], " ", "", -1)
							value = ""
							i++
							index = i
						} else if upperCaseValue[i] == 'L' && i < (totalLen-4) && upperCaseValue[i+1] == 'I' && upperCaseValue[i+3] == 'E' {
							operator = " LIKE "
							hdr = VALUE
							key = strings.Replace(formValue[index:i], " ", "", -1)
							i += 3
							index = i
						}
					case hdr == VALUE:
						/* = */
						if formValue[i] == ' ' {
							index = i
						} else if upperCaseValue[i] == '"' {
							typeValue = "string"
							i++
							index = i
							hdr = END
							/* formValue != */
						} else {
							typeValue = "integer"
							index = i
							hdr = END
						}
					case hdr == END:
						if upperCaseValue[i] == '"' || upperCaseValue[i] == ' ' || i == (totalLen-1) {
							value = formValue[index:i]
							hdr = RESET
							i++
							index = i
							if strings.Contains(key, ".") {
								elemArray := strings.Split(key, ".")
								if typeValue == "integer" {
									sql = sql + endEl + fmt.Sprintf("(%s->>'%s')::int%s%d ", elemArray[0], elemArray[1], operator, heputils.CheckIntValue(value))
								} else {
									sql = sql + endEl + fmt.Sprintf("%s->>'%s'%s'%s' ", elemArray[0], elemArray[1], operator, value)
								}
							} else if formType == "integer" {
								sql = sql + endEl + fmt.Sprintf("%s%s%d ", key, operator, heputils.CheckIntValue(value))
							} else {
								if value == "isEmpty" {
									sql = sql + endEl + fmt.Sprintf("%s%s''", key, operator)
								} else if value == "isNull" && key == "=" {
									sql = sql + endEl + fmt.Sprintf("%s is NULL", key)
								} else {
									sql = sql + endEl + fmt.Sprintf("%s%s'%s'", key, operator, value)
								}
							}
							continue
						}
					case hdr == RESET:
						if i < (totalLen-2) && upperCaseValue[i] == 'O' && upperCaseValue[i+1] == 'R' {
							endEl = " OR "
							hdr = FIRST
							i += 2
							index = i
						} else if i < (totalLen-3) && upperCaseValue[i] == 'A' && upperCaseValue[i+1] == 'N' && upperCaseValue[i+2] == 'D' {
							endEl = " AND "
							hdr = FIRST
							i += 3
							index = i
						}
					}
				}
				sql += ")"
				continue
			}

			notStr := ""
			equalStr := "="
			operator := " AND "
			logrus.Debug(k, ". formName: ", formName)
			logrus.Debug(k, ". formValue: ", formValue)
			logrus.Debug(k, ". formType: ", formType)

			if strings.HasPrefix(formValue, "||") {
				formValue = strings.TrimPrefix(formValue, "||")
				if k > 0 {
					operator = " OR "
				}
			}
			if strings.HasPrefix(formValue, "!") {
				notStr = " NOT "
				equalStr = " <> "
			}
			if formName == "limit" {
				sLimit = heputils.CheckIntValue(formValue)
				continue
			} else if formName == "raw" {
				sql = sql + operator + formName + notStr + " ILIKE '%" + heputils.Sanitize(formValue) + "%'"
				continue
			}

			var valueArray []string
			if strings.Contains(formValue, ";") {
				valueArray = strings.Split(formValue, ";")
			} else {
				valueArray = []string{formValue}
			}
			valueArray = heputils.SanitizeTextArray(valueArray)

			// data_header or protocal_header values
			if strings.Contains(formName, ".") {
				elemArray := strings.Split(formName, ".")
				if formType == "integer" {
					sql = sql + operator + fmt.Sprintf("(%s->>'%s')::int%s%d", elemArray[0], elemArray[1], equalStr, heputils.CheckIntValue(formValue))
					continue
				}
				if strings.Contains(formValue, "%") && strings.Contains(formValue, ";") {
					sql = sql + operator + fmt.Sprintf("%s->>'%s' %sLIKE ANY ('{%s}')", elemArray[0], elemArray[1], notStr, heputils.Sanitize(formValue))
					sql = strings.Replace(sql, ";", ",", -1)
				} else if strings.Contains(formValue, "%") {
					sql = sql + operator + fmt.Sprintf("%s->>'%s' %sLIKE '%s'", elemArray[0], elemArray[1], notStr, heputils.Sanitize(formValue))
				} else if strings.Contains(formValue, ",") || len(valueArray) > 1 {
					sql = sql + operator + fmt.Sprintf("%s->>'%s' %sIN ('%s')", elemArray[0], elemArray[1], notStr, strings.Join(valueArray[:], "','"))
				} else {

					if len(valueArray) == 1 {
						if valueArray[0] == "isEmpty" {
							sql = sql + operator + fmt.Sprintf("%s->>'%s' %s ''", elemArray[0], elemArray[1], equalStr)
						} else if valueArray[0] == "isNull" && equalStr == "=" {
							sql = sql + operator + fmt.Sprintf("%s->>'%s' is NULL", elemArray[0], elemArray[1])
						} else {
							sql = sql + operator + fmt.Sprintf("%s->>'%s'%s'%s'", elemArray[0], elemArray[1], equalStr, strings.Join(valueArray[:], "','"))
						}
					} else {
						sql = sql + operator + fmt.Sprintf("%s->>'%s'%s'%s'", elemArray[0], elemArray[1], equalStr, strings.Join(valueArray[:], "','"))
					}
				}
				continue
			}

			if formType == "integer" {
				sql = sql + operator + fmt.Sprintf("%s%s%d", formName, equalStr, heputils.CheckIntValue(formValue))
				continue
			}

			if strings.Contains(formValue, "%") {
				sql = sql + operator + formName + notStr + " LIKE '" + heputils.Sanitize(formValue) + "'"
			} else {
				sql = sql + operator + fmt.Sprintf("%s %sIN ('%s')", formName, notStr, strings.Join(valueArray[:], "','"))
			}
		}
	}
	return sql, sLimit
}

// this method create new user in the database
// it doesn't check internally whether all the validation are applied or not
func (ss *SearchService) SearchData(searchObject *model.SearchObject, aliasData map[string]string, userGroup string) (string, error) {
	//table := "hep_proto_1_default"
	searchData := []model.HepTable{}
	searchFromTime := time.Unix(searchObject.Timestamp.From/int64(time.Microsecond), 0)
	searchToTime := time.Unix(searchObject.Timestamp.To/int64(time.Microsecond), 0)
	//if searchToTime.Sub(searchFromTime).Hours() > 7*24 {
	//	return "", errors.New("time span must in 7 days")
	//}

	Data, _ := json.Marshal(searchObject.Param.Search)
	sData, _ := gabs.ParseJSON(Data)
	sql := "create_date between ? AND ?"

	logrus.Debug("ISOLATEGROUP ", config.Setting.IsolateGroup)
	logrus.Debug("USERGROUP ", userGroup)

	if config.Setting.IsolateGroup != "" && config.Setting.IsolateGroup == userGroup {
		sql = sql + " AND " + config.Setting.IsolateQuery
	}

	var sLimit int
	sLimit = 200

	/**
	 * case "INVITE", "ACK", "BYE", "CANCEL", "UPDATE", "PRACK", "REFER", "INFO":
	 *		h.SIP.Profile = "call"
	 * case "REGISTER":
	 * 	h.SIP.Profile = "registration"
	 * default:
	 *		h.SIP.Profile = "default"
	 */
	for key, _ := range sData.ChildrenMap() {
		//table = "hep_proto_" + key
		if sData.Exists(key) {
			elems := sData.Search(key).Data().([]interface{})
			s, l := buildQuery(elems)
			sql += s
			if len(s) > 0 && l < 1000 {
				sLimit = 1000
			}
		}
	}

	var tableArray []string
	if strings.Contains(sql, "token") ||
		strings.Contains(sql, "NOTIFY") ||
		strings.Contains(sql, "OPTIONS") {
		tableArray = []string{"hep_proto_1_default"}
	} else if strings.Contains(sql, "REGISTER") {
		tableArray = []string{"hep_proto_1_registration"}
	} else if strings.Contains(sql, "INVITE") ||
		strings.Contains(sql, "ACK") ||
		strings.Contains(sql, "BYE") ||
		strings.Contains(sql, "CANCEL") ||
		strings.Contains(sql, "UPDATE") ||
		strings.Contains(sql, "PRACK") ||
		strings.Contains(sql, "REFER") ||
		strings.Contains(sql, "INFO") {
		tableArray = []string{"hep_proto_1_call"}
	} else if strings.Contains(sql, "X-CID") {
		tableArray = []string{"hep_proto_1_call", "hep_proto_1_default"}
	} else {
		tableArray = []string{"hep_proto_1_call", "hep_proto_1_registration", "hep_proto_1_default"}
	}

	for index := 0; index < len(tableArray); index++ {
		table := tableArray[index]
		//var searchData
		for session := range ss.Session {
			searchTmp := []model.HepTable{}
			/* if node doesnt exists - continue */
			if !heputils.ElementExists(searchObject.Param.Location.Node, session) {
				continue
			}

			ss.Session[session].Debug().
				Table(table).
				Where(sql, searchFromTime, searchToTime).
				Order("create_date DESC").
				Limit(sLimit).
				Find(&searchTmp)

			if len(searchTmp) > 0 {
				for val := range searchTmp {
					searchTmp[val].Node = session
					searchTmp[val].DBNode = session
				}

				searchData = append(searchData, searchTmp...)
			}
		}
	}

	/* lets sort it */
	//sort.Slice(searchData, func(i, j int) bool {
	//	return searchData[i].CreatedDate.Before(searchData[j].CreatedDate)
	//})

	searchData = uniqueHepTable(searchData)

	rows, _ := json.Marshal(searchData)
	data, _ := gabs.ParseJSON(rows)
	dataReply := gabs.Wrap([]interface{}{})
	for _, value := range data.Children() {
		alias := gabs.New()
		dataElement := gabs.New()
		for k, v := range value.ChildrenMap() {
			switch k {
			case "data_header", "protocol_header":
				dataElement.Merge(v)
			case "id", "sid", "node", "dbnode":
				newData := gabs.New()
				newData.Set(v.Data().(interface{}), k)
				dataElement.Merge(newData)
			}
		}

		srcPort, dstPort := "0", "0"

		if dataElement.Exists("srcPort") {
			srcPort = strconv.FormatFloat(dataElement.S("srcPort").Data().(float64), 'f', 0, 64)
		}

		if dataElement.Exists("dstPort") {
			dstPort = strconv.FormatFloat(dataElement.S("dstPort").Data().(float64), 'f', 0, 64)
		}

		srcIP := dataElement.S("srcIp").Data().(string)
		dstIP := dataElement.S("dstIp").Data().(string)

		srcIPPort := srcIP + ":" + srcPort
		dstIPPort := dstIP + ":" + dstPort

		testInput := net.ParseIP(srcIP)
		if testInput.To4() == nil && testInput.To16() != nil {
			srcIPPort = "[" + srcIP + "]:" + srcPort
		}

		testInput = net.ParseIP(dstIP)
		if testInput.To4() == nil && testInput.To16() != nil {
			dstIPPort = "[" + dstIP + "]:" + dstPort
		}

		if _, ok := aliasData[srcIP]; ok {
			alias.Set(aliasData[srcIP], srcIPPort)
		} else {
			alias.Set(srcIPPort, srcIPPort)
		}

		if _, ok := aliasData[dstIP]; ok {
			alias.Set(aliasData[dstIP], dstIPPort)
		} else {
			alias.Set(dstIPPort, dstIPPort)
		}

		//dataElement.Set(table, "table")

		createDate := int64(dataElement.S("timeSeconds").Data().(float64)*1000000 + dataElement.S("timeUseconds").Data().(float64))

		dataElement.Set(createDate/1000, "create_date")
		if err := dataReply.ArrayAppend(dataElement.Data()); err != nil {
			logrus.Errorln("Bad assigned array")
		}
	}

	dataKeys := gabs.Wrap([]interface{}{})
	for _, v := range dataReply.Children() {
		for key := range v.ChildrenMap() {
			if !function.ArrayKeyExits(key, dataKeys) {
				dataKeys.ArrayAppend(key)
			}
		}
	}

	total, _ := dataReply.ArrayCount()

	reply := gabs.New()
	reply.Set(total, "total")
	reply.Set(dataReply.Data(), "data")
	reply.Set(dataKeys.Data(), "keys")

	return reply.String(), nil
}

func (ss *SearchService) processSbcOption(processData []model.HepTable, totalMap map[string]int, tryAgain bool) ([]model.SbcOptionData, []string) {
	sbcOptionDataTable := []model.SbcOptionData{}
	invalidSids := []string{}

	if len(processData) <= 0 {
		return sbcOptionDataTable, invalidSids
	}

	sort.Slice(processData, func(i, j int) bool {
		return processData[i].CreatedDate.Before(processData[j].CreatedDate)
	})

	sipHepDataMap := make(map[string][]model.HepTable)
	for _, entry := range processData {
		var protocolHeader model.ProtocolHeader
		json.Unmarshal(entry.ProtocolHeader, &protocolHeader)
		if !strings.Contains(protocolHeader.CaptureID, "SBC") {
			continue
		}
		sipHepDataMap[entry.Sid] = append(sipHepDataMap[entry.Sid], entry)
	}

	timeoutMap := make(map[string]int)
	for sid, hepTables := range sipHepDataMap {
		var sbcOptionData model.SbcOptionData
		var optionTime time.Time
		var optionResponseTime time.Time
		var retryOption bool = false
		for _, entryHep := range hepTables {
			var protocolHeader model.ProtocolHeader
			json.Unmarshal(entryHep.ProtocolHeader, &protocolHeader)
			if _, ok := config.Setting.OriginSbcIpMap[protocolHeader.SrcIP]; ok {
				sbcOptionData.SbcIp = protocolHeader.SrcIP
			}
			if _, ok := config.Setting.OriginSbcIpMap[protocolHeader.DstIP]; ok {
				sbcOptionData.SbcIp = protocolHeader.DstIP
			}

			var dataHeader model.DataHeader
			json.Unmarshal(entryHep.DataHeader, &dataHeader)
			upperMethod := strings.ToUpper(dataHeader.Method)
			if strings.EqualFold(upperMethod, "OPTIONS") {
				if optionTime.IsZero() {
					optionTime = entryHep.CreatedDate
				} else {
					retryOption = true
				}
			}

			if !strings.EqualFold(upperMethod, "OPTIONS") && optionResponseTime.IsZero() {
				optionResponseTime = entryHep.CreatedDate
				sbcOptionData.Method = dataHeader.Method
			}
		}

		if _, ok := config.Setting.OriginSbcIpMap[sbcOptionData.SbcIp]; !ok {
			logrus.Errorf("processSbcOption invalid sbc_ip = %s, sid = %s, option_time = %v, option_response_time = %v\n",
				sbcOptionData.SbcIp, sid, optionTime, optionResponseTime)
			continue
		}

		if optionTime.IsZero() || optionResponseTime.IsZero() {
			invalidSids = append(invalidSids, sid)
			if tryAgain {
				break
			}
		}

		if _, ok := totalMap[sbcOptionData.SbcIp]; ok {
			totalMap[sbcOptionData.SbcIp]++
		} else {
			totalMap[sbcOptionData.SbcIp] = 1
		}

		if optionTime.IsZero() || optionResponseTime.IsZero() {
			if _, ok := timeoutMap[sbcOptionData.SbcIp]; ok {
				timeoutMap[sbcOptionData.SbcIp]++
			} else {
				timeoutMap[sbcOptionData.SbcIp] = 1
			}
			logrus.Errorf("processSbcOption timeout sbc_ip = %s, sid = %s, option_time = %v, option_response_time = %v, retry_option = %v\n",
				sbcOptionData.SbcIp, sid, optionTime, optionResponseTime, retryOption)
			continue
		}

		delayTime := optionResponseTime.Sub(optionTime)
		sbcOptionData.DelayMS = delayTime.Microseconds()
		sbcOptionData.TimeoutCount = 0
		sbcOptionDataTable = append(sbcOptionDataTable, sbcOptionData)
		if sbcOptionData.DelayMS > 5000000 {
			logrus.Errorf("processSbcOption high delay, sbc_ip ip = %s, sid = %s, option_time = %v, option_response_time = %v, delay_ms = %d\n",
				sbcOptionData.SbcIp, sid, optionTime, optionResponseTime, sbcOptionData.DelayMS)
		}
	}

	for sbcIp, timeoutCount := range timeoutMap {
		var sbcOptionData model.SbcOptionData
		sbcOptionData.SbcIp = sbcIp
		sbcOptionData.DelayMS = 0
		sbcOptionData.TimeoutCount = timeoutCount
		if totalCount, ok := totalMap[sbcOptionData.SbcIp]; ok {
			sbcOptionData.TotalCount = totalCount
		}
		sbcOptionDataTable = append(sbcOptionDataTable, sbcOptionData)
	}

	return sbcOptionDataTable, invalidSids
}

func (ss *SearchService) SearchSbcOption() []model.SbcOptionData {
	timeFrom := time.Now().Add(time.Duration(-2) * time.Minute)
	timeTo := time.Now().Add(time.Duration(-1) * time.Minute)
	logrus.Infof("SearchSbcOption search first timeFrom=%v, timeTo=%v\n", timeFrom, timeTo)
	searchData := []model.HepTable{}
	needCheckIps := ""
	for key, _ := range config.Setting.OriginSbcIpMap {
		if len(needCheckIps) > 0 {
			needCheckIps += ","
		}
		needCheckIps += "'" + key + "'"
	}
	query := "create_date between ? AND ?"
	query += " AND (protocol_header->>'dstIp' in (" + needCheckIps + ") OR protocol_header->>'srcIp' in (" + needCheckIps + "))"
	query += " AND data_header->>'method' in ('OPTIONS', '200', '480', '500')"
	table := "hep_proto_1_default"
	limitCount := 5000
	for session := range ss.Session {
		searchTmp := []model.HepTable{}
		if err := ss.Session[session].Debug().
			Table(table).
			Where(query, timeFrom.Format(time.RFC3339), timeTo.Format(time.RFC3339)).
			Limit(limitCount).
			Find(&searchTmp).Error; err != nil {
			logrus.Errorln("SearchSbcOption: We have got error: ", err)
		}

		if len(searchTmp) > 0 {
			searchData = append(searchData, searchTmp...)
		}
	}

	var sbcOptionDataTable []model.SbcOptionData
	if len(searchData) <= 0 {
		return sbcOptionDataTable
	}

	if len(searchData) >= limitCount {
		logrus.Errorf("SearchSbcOption search count is reached limit time_from = %v, time_to = %v\n",
			timeFrom, timeTo)
	}

	sbcTotalMap := make(map[string]int)
	sbcOptionDataTable, invalidSids := ss.processSbcOption(searchData, sbcTotalMap, true)
	if len(invalidSids) <= 0 {
		return sbcOptionDataTable
	}

	needSearchSids := ""
	for _, sid := range invalidSids {
		if len(needSearchSids) > 0 {
			needSearchSids += ","
		}
		needSearchSids += "'" + sid + "'"
	}

	timeFrom = time.Now().Add(time.Duration(-30) * time.Minute)
	timeTo = time.Now()
	logrus.Infof("SearchSbcOption search again timeFrom=%v, timeTo=%v\n", timeFrom, timeTo)
	query = "create_date between ? AND ?"
	query += " AND sid in (" + needSearchSids + ")"
	query += " AND data_header->>'method' in ('OPTIONS', '200', '480', '500')"
	table = "hep_proto_1_default"
	searchData = []model.HepTable{}
	for session := range ss.Session {
		searchTmp := []model.HepTable{}
		if err := ss.Session[session].Debug().
			Table(table).
			Where(query, timeFrom.Format(time.RFC3339), timeTo.Format(time.RFC3339)).
			Limit(limitCount).
			Find(&searchTmp).Error; err != nil {
			logrus.Errorln("SearchSbcOption: We have got error: ", err)
		}

		if len(searchTmp) > 0 {
			searchData = append(searchData, searchTmp...)
		}
	}

	processTable, invalidSids := ss.processSbcOption(searchData, sbcTotalMap, false)
	sbcOptionDataTable = append(sbcOptionDataTable, processTable...)
	if len(invalidSids) > 0 {
		logrus.Errorf("SearchSbcOption search again really invalidSids=%v, timeFrom=%v, timeTo=%v\n", invalidSids, timeFrom, timeTo)
	}

	return sbcOptionDataTable
}

func (ss *SearchService) processSbcInvite100(processData []model.HepTable, regIpMap map[string]bool, totalMap map[model.DIRECTION]map[string]int, tryAgain bool) ([]model.SbcInvite100Data, []string, bool) {
	sbcInvite100DataTable := []model.SbcInvite100Data{}
	invalidSids := []string{}
	dbDelay := false
	if len(processData) <= 0 {
		return sbcInvite100DataTable, invalidSids, dbDelay
	}

	sort.Slice(processData, func(i, j int) bool {
		return processData[i].CreatedDate.Before(processData[j].CreatedDate)
	})

	sipHepDataMap := make(map[model.DIRECTION]map[string]map[string][]model.HepTable)
	sipHepDataMap[model.E_SBC_2_REG] = make(map[string]map[string][]model.HepTable)
	sipHepDataMap[model.E_SBC_2_PHONE] = make(map[string]map[string][]model.HepTable)
	sipHepDataMap[model.E_REG_2_SBC] = make(map[string]map[string][]model.HepTable)
	for _, entry := range processData {
		var protocolHeader model.ProtocolHeader
		json.Unmarshal(entry.ProtocolHeader, &protocolHeader)
		var dataHeader model.DataHeader
		json.Unmarshal(entry.DataHeader, &dataHeader)

		_, fromSbc := config.Setting.OriginSbcIpMap[protocolHeader.SrcIP]
		_, fromReg := regIpMap[protocolHeader.SrcIP]
		_, toSbc := config.Setting.OriginSbcIpMap[protocolHeader.DstIP]
		_, toReg := regIpMap[protocolHeader.DstIP]
		upperCaptureID := strings.ToUpper(protocolHeader.CaptureID)
		upperMethod := strings.ToUpper(dataHeader.Method)
		if strings.Contains(upperCaptureID, "SBC") {
			if strings.EqualFold(upperMethod, "INVITE") && fromSbc {
				if toReg {
					if _, ok := sipHepDataMap[model.E_SBC_2_REG][entry.Sid]; !ok {
						sipHepDataMap[model.E_SBC_2_REG][entry.Sid] = make(map[string][]model.HepTable)
					}
					sipHepDataMap[model.E_SBC_2_REG][entry.Sid][dataHeader.CSeq] = append(sipHepDataMap[model.E_SBC_2_REG][entry.Sid][dataHeader.CSeq], entry)
				} else {
					if _, ok := sipHepDataMap[model.E_SBC_2_PHONE][entry.Sid]; !ok {
						sipHepDataMap[model.E_SBC_2_PHONE][entry.Sid] = make(map[string][]model.HepTable)
					}
					sipHepDataMap[model.E_SBC_2_PHONE][entry.Sid][dataHeader.CSeq] = append(sipHepDataMap[model.E_SBC_2_PHONE][entry.Sid][dataHeader.CSeq], entry)
				}
			}
			if !strings.EqualFold(upperMethod, "INVITE") && toSbc {
				if fromReg {
					if _, ok := sipHepDataMap[model.E_SBC_2_REG][entry.Sid]; !ok {
						sipHepDataMap[model.E_SBC_2_REG][entry.Sid] = make(map[string][]model.HepTable)
					}
					sipHepDataMap[model.E_SBC_2_REG][entry.Sid][dataHeader.CSeq] = append(sipHepDataMap[model.E_SBC_2_REG][entry.Sid][dataHeader.CSeq], entry)
				} else {
					if _, ok := sipHepDataMap[model.E_SBC_2_PHONE][entry.Sid]; !ok {
						sipHepDataMap[model.E_SBC_2_PHONE][entry.Sid] = make(map[string][]model.HepTable)
					}
					sipHepDataMap[model.E_SBC_2_PHONE][entry.Sid][dataHeader.CSeq] = append(sipHepDataMap[model.E_SBC_2_PHONE][entry.Sid][dataHeader.CSeq], entry)
				}
			}
		}

		if strings.Contains(upperCaptureID, "REG") {
			if _, ok := sipHepDataMap[model.E_REG_2_SBC][entry.Sid]; !ok {
				sipHepDataMap[model.E_REG_2_SBC][entry.Sid] = make(map[string][]model.HepTable)
			}
			if strings.EqualFold(upperMethod, "INVITE") && fromReg && toSbc {
				sipHepDataMap[model.E_REG_2_SBC][entry.Sid][dataHeader.CSeq] = append(sipHepDataMap[model.E_REG_2_SBC][entry.Sid][dataHeader.CSeq], entry)
			}

			if !strings.EqualFold(upperMethod, "INVITE") && fromSbc && toReg {
				sipHepDataMap[model.E_REG_2_SBC][entry.Sid][dataHeader.CSeq] = append(sipHepDataMap[model.E_REG_2_SBC][entry.Sid][dataHeader.CSeq], entry)
			}
		}
	}

	for direction, sidMap := range sipHepDataMap {
		timeoutMap := make(map[string]int)
		for sid, sidChildMap := range sidMap {
			for cseq, hepTables := range sidChildMap {
				var sbcInvite100Data model.SbcInvite100Data
				sbcInvite100Data.Direction = direction
				var inviteTime time.Time
				var invite100Time time.Time
				retryInvite := false
				for _, entryHep := range hepTables {
					var protocolHeader model.ProtocolHeader
					json.Unmarshal(entryHep.ProtocolHeader, &protocolHeader)
					if _, ok := config.Setting.OriginSbcIpMap[protocolHeader.SrcIP]; ok {
						sbcInvite100Data.SbcIp = protocolHeader.SrcIP
					}
					if _, ok := config.Setting.OriginSbcIpMap[protocolHeader.DstIP]; ok {
						sbcInvite100Data.SbcIp = protocolHeader.DstIP
					}

					var dataHeader model.DataHeader
					json.Unmarshal(entryHep.DataHeader, &dataHeader)
					upperMethod := strings.ToUpper(dataHeader.Method)
					if strings.EqualFold(upperMethod, "INVITE") {
						if inviteTime.IsZero() {
							inviteTime = entryHep.CreatedDate
						} else {
							retryInvite = true
						}
					}

					if !strings.EqualFold(upperMethod, "INVITE") && invite100Time.IsZero() {
						if strings.EqualFold(upperMethod, "180") || strings.EqualFold(upperMethod, "183") {
							//miss 100
							break
						}
						invite100Time = entryHep.CreatedDate
					}
				}

				if _, ok := config.Setting.OriginSbcIpMap[sbcInvite100Data.SbcIp]; !ok {
					logrus.Errorf("processSbcInvite100 invalid sbc_ip ip = %s, sid = %s, cseq = %s, invite_time = %v, invite_100_time = %v\n",
						sbcInvite100Data.SbcIp, sid, cseq, inviteTime, invite100Time)
					continue
				}

				if inviteTime.IsZero() || invite100Time.IsZero() {
					invalidSids = append(invalidSids, sid)
					if tryAgain {
						break
					}
					//because of db master-slave delay
					if !retryInvite {
						dbDelay = true
						break
					}
				}

				if _, ok := totalMap[direction][sbcInvite100Data.SbcIp]; ok {
					totalMap[direction][sbcInvite100Data.SbcIp]++
				} else {
					totalMap[direction][sbcInvite100Data.SbcIp] = 1
				}

				if inviteTime.IsZero() || invite100Time.IsZero() {
					if _, ok := timeoutMap[sbcInvite100Data.SbcIp]; ok {
						timeoutMap[sbcInvite100Data.SbcIp]++
					} else {
						timeoutMap[sbcInvite100Data.SbcIp] = 1
					}
					logrus.Errorf("processSbcInvite100 timeout direction=%d, sbc_ip ip = %s, sid = %s, cseq = %s, invite_time = %v, invite_100_time = %v, retry_invite = %v\n",
						direction, sbcInvite100Data.SbcIp, sid, cseq, inviteTime, invite100Time, retryInvite)
					continue
				}

				delayTime := invite100Time.Sub(inviteTime)
				sbcInvite100Data.DelayMS = delayTime.Microseconds()
				sbcInvite100Data.TimeoutCount = 0
				sbcInvite100DataTable = append(sbcInvite100DataTable, sbcInvite100Data)
				if sbcInvite100Data.DelayMS > 5000000 {
					logrus.Errorf("processSbcInvite100 high delay direction=%d, sbc_ip ip = %s, sid = %s, cseq = %s, invite_time = %v, invite_100_time = %v, delay_ms = %d\n",
						direction, sbcInvite100Data.SbcIp, sid, cseq, inviteTime, invite100Time, sbcInvite100Data.DelayMS)
				}
			}
		}

		for sbcIp, timeoutCount := range timeoutMap {
			var sbcInvite100Data model.SbcInvite100Data
			sbcInvite100Data.Direction = direction
			sbcInvite100Data.SbcIp = sbcIp
			sbcInvite100Data.DelayMS = 0
			sbcInvite100Data.TimeoutCount = timeoutCount
			if totalCount, ok := totalMap[direction][sbcInvite100Data.SbcIp]; ok {
				sbcInvite100Data.TotalCount = totalCount
			}
			sbcInvite100DataTable = append(sbcInvite100DataTable, sbcInvite100Data)
		}
	}

	return sbcInvite100DataTable, invalidSids, dbDelay
}

func (ss *SearchService) SearchSbcInvite100() []model.SbcInvite100Data {
	timeFrom := time.Now().Add(time.Duration(-2) * time.Minute)
	timeTo := time.Now().Add(time.Duration(-1) * time.Minute)
	logrus.Infof("SearchSbcInvite100 search first timeFrom=%v, timeTo=%v\n", timeFrom, timeTo)
	searchData := []model.HepTable{}
	needCheckIps := ""
	regIpMap := make(map[string]bool)
	regIpMap["10.58.208.9"] = true
	regIpMap["10.109.132.15"] = true
	//voip_reg
	for key, _ := range config.Setting.OriginVoipRegIpMap {
		regIpMap[key] = true
	}

	for key, _ := range config.Setting.OriginSbcIpMap {
		if len(needCheckIps) > 0 {
			needCheckIps += ","
		}
		needCheckIps += "'" + key + "'"
	}
	for key, _ := range regIpMap {
		needCheckIps += ",'" + key + "'"
	}
	query := "create_date between ? AND ? "
	query += " AND (protocol_header->>'dstIp' in (" + needCheckIps + ") OR protocol_header->>'srcIp' in (" + needCheckIps + "))"
	query += " AND data_header->>'method' in ('INVITE', '100')"
	table := "hep_proto_1_call"
	limitCount := 10000
	for session := range ss.Session {
		searchTmp := []model.HepTable{}
		if err := ss.Session[session].Debug().
			Table(table).
			Where(query, timeFrom.Format(time.RFC3339), timeTo.Format(time.RFC3339)).
			Limit(limitCount).
			Find(&searchTmp).Error; err != nil {
			logrus.Errorln("SearchSbcInvite100: We have got error: ", err)
		}

		if len(searchTmp) > 0 {
			searchData = append(searchData, searchTmp...)
		}
	}

	var sbcInvite100DataTable []model.SbcInvite100Data

	if len(searchData) <= 0 {
		return sbcInvite100DataTable
	}

	if len(searchData) >= limitCount {
		logrus.Errorf("SearchSbcInvite100 search count is reached limit time_from = %v, time_to = %v\n",
			timeFrom, timeTo)
	}

	sbcTotalMap := make(map[model.DIRECTION]map[string]int)
	sbcTotalMap[model.E_SBC_2_REG] = make(map[string]int)
	sbcTotalMap[model.E_SBC_2_PHONE] = make(map[string]int)
	sbcTotalMap[model.E_REG_2_SBC] = make(map[string]int)
	sbcInvite100DataTable, invalidSids, _ := ss.processSbcInvite100(searchData, regIpMap, sbcTotalMap, true)
	logrus.Infof("SearchSbcInvite100 search first finished invalidSids=%v, timeFrom=%v, timeTo=%v\n", invalidSids, timeFrom, timeTo)
	if len(invalidSids) <= 0 {
		return sbcInvite100DataTable
	}

	needSearchSids := ""
	for _, sid := range invalidSids {
		if len(needSearchSids) > 0 {
			needSearchSids += ","
		}
		needSearchSids += "'" + sid + "'"
	}

	timeFrom = time.Now().Add(time.Duration(-30) * time.Minute)
	timeTo = time.Now()
	logrus.Infof("SearchSbcInvite100 search again timeFrom=%v, timeTo=%v\n", timeFrom, timeTo)
	loopSelectCount := 0
LoopSelect:
	query = "create_date between ? AND ?"
	query += " AND sid in (" + needSearchSids + ")"
	query += " AND data_header->>'method' in ('INVITE', '100', '180', '183', '200')"
	query += " AND data_header->>'cseq' ILIKE '%INVITE%'"
	table = "hep_proto_1_call"
	searchData = []model.HepTable{}
	for session := range ss.Session {
		searchTmp := []model.HepTable{}
		if err := ss.Session[session].Debug().
			Table(table).
			Where(query, timeFrom.Format(time.RFC3339), timeTo.Format(time.RFC3339)).
			Limit(limitCount).
			Find(&searchTmp).Error; err != nil {
			logrus.Errorln("SearchSbcInvite100: We have got error: ", err)
		}

		if len(searchTmp) > 0 {
			searchData = append(searchData, searchTmp...)
		}
	}

	processTable, invalidSids, dbDelay := ss.processSbcInvite100(searchData, regIpMap, sbcTotalMap, false)
	sbcInvite100DataTable = append(sbcInvite100DataTable, processTable...)
	if len(invalidSids) > 0 {
		loopSelectCount++
		if dbDelay && loopSelectCount <= 5 {
			time.Sleep(time.Second)
			logrus.Infof("SearchSbcInvite100 loop select again invalidSids=%v, timeFrom=%v, timeTo=%v\n", invalidSids, timeFrom, timeTo)
			goto LoopSelect
		}
		logrus.Errorf("SearchSbcInvite100 search again really invalidSids=%v, timeFrom=%v, timeTo=%v\n", invalidSids, timeFrom, timeTo)
	}

	return sbcInvite100DataTable
}

func (ss *SearchService) processRegInvite100(processData []model.HepTable, regIpMap map[string]bool, totalMap map[model.PHONE_TYPE]map[string]int, tryAgain bool) ([]model.RegInvite100Data, []string) {
	regInvite100DataTable := []model.RegInvite100Data{}
	invalidSids := []string{}
	regexPhoneNormal := regexp.MustCompile("^[6|7][7-9]\\d{4}$")
	regexPhoneVoip := regexp.MustCompile("^[6|7][4-6]\\d{4}$")

	if len(processData) <= 0 {
		return regInvite100DataTable, invalidSids
	}

	sort.Slice(processData, func(i, j int) bool {
		return processData[i].CreatedDate.Before(processData[j].CreatedDate)
	})

	sipHepDataMap := make(map[model.PHONE_TYPE]map[string]map[string][]model.HepTable)
	sipHepDataMap[model.E_PHONE_TYPE_INVALID] = make(map[string]map[string][]model.HepTable)
	sipHepDataMap[model.E_PHONE_TYPE_HARD] = make(map[string]map[string][]model.HepTable)
	sipHepDataMap[model.E_PHONE_TYPE_SOFT] = make(map[string]map[string][]model.HepTable)
	for _, entry := range processData {
		var protocolHeader model.ProtocolHeader
		json.Unmarshal(entry.ProtocolHeader, &protocolHeader)
		var dataHeader model.DataHeader
		json.Unmarshal(entry.DataHeader, &dataHeader)

		_, fromReg := regIpMap[protocolHeader.SrcIP]
		_, toReg := regIpMap[protocolHeader.DstIP]
		upperCaptureID := strings.ToUpper(protocolHeader.CaptureID)
		upperMethod := strings.ToUpper(dataHeader.Method)

		if !strings.Contains(upperCaptureID, "REG") {
			continue
		}

		if strings.EqualFold(upperMethod, "INVITE") && !fromReg {
			continue
		}

		if !strings.EqualFold(upperMethod, "INVITE") && !toReg {
			continue
		}

		if len(regexPhoneNormal.FindString(dataHeader.ToUser)) > 0 {
			if _, ok := sipHepDataMap[model.E_PHONE_TYPE_HARD][entry.Sid]; !ok {
				sipHepDataMap[model.E_PHONE_TYPE_HARD][entry.Sid] = make(map[string][]model.HepTable)
			}
			sipHepDataMap[model.E_PHONE_TYPE_HARD][entry.Sid][dataHeader.CSeq] = append(sipHepDataMap[model.E_PHONE_TYPE_HARD][entry.Sid][dataHeader.CSeq], entry)
		} else if len(regexPhoneVoip.FindString(dataHeader.ToUser)) > 0 {
			if _, ok := sipHepDataMap[model.E_PHONE_TYPE_SOFT][entry.Sid]; !ok {
				sipHepDataMap[model.E_PHONE_TYPE_SOFT][entry.Sid] = make(map[string][]model.HepTable)
			}
			sipHepDataMap[model.E_PHONE_TYPE_SOFT][entry.Sid][dataHeader.CSeq] = append(sipHepDataMap[model.E_PHONE_TYPE_SOFT][entry.Sid][dataHeader.CSeq], entry)
		} else {
			logrus.Errorf("processRegInvite100 invalid phone user = %s, sid = %s cseq = %s\n", dataHeader.ToUser, entry.Sid, dataHeader.CSeq)
			continue
		}

	}

	for phoneType, sidMap := range sipHepDataMap {
		timeoutMap := make(map[string]int)
		for sid, sidChildMap := range sidMap {
			for cseq, hepTables := range sidChildMap {
				var regInvite100Data model.RegInvite100Data
				regInvite100Data.PhoneType = phoneType
				var inviteTime time.Time
				var invite100Time time.Time
				retryInvite := false
				for _, entryHep := range hepTables {
					var protocolHeader model.ProtocolHeader
					json.Unmarshal(entryHep.ProtocolHeader, &protocolHeader)
					if _, ok := regIpMap[protocolHeader.SrcIP]; ok {
						regInvite100Data.RegIp = protocolHeader.SrcIP
					}
					if _, ok := regIpMap[protocolHeader.DstIP]; ok {
						regInvite100Data.RegIp = protocolHeader.DstIP
					}

					var dataHeader model.DataHeader
					json.Unmarshal(entryHep.DataHeader, &dataHeader)
					upperMethod := strings.ToUpper(dataHeader.Method)
					if strings.EqualFold(upperMethod, "INVITE") && inviteTime.IsZero() {
						if inviteTime.IsZero() {
							inviteTime = entryHep.CreatedDate
						} else {
							retryInvite = true
						}
					}

					if !strings.EqualFold(upperMethod, "INVITE") && invite100Time.IsZero() {
						if strings.EqualFold(upperMethod, "180") || strings.EqualFold(upperMethod, "183") {
							//miss 100
							break
						}
						invite100Time = entryHep.CreatedDate
					}
				}

				if _, ok := regIpMap[regInvite100Data.RegIp]; !ok {
					logrus.Errorf("processRegInvite100 invalid reg_ip ip = %s, sid = %s, cseq = %s, invite_time = %v, invite_100_time = %v\n",
						regInvite100Data.RegIp, sid, cseq, inviteTime, invite100Time)
					continue
				}

				if inviteTime.IsZero() || invite100Time.IsZero() {
					invalidSids = append(invalidSids, sid)
					if tryAgain {
						break
					}
				}

				if _, ok := totalMap[phoneType][regInvite100Data.RegIp]; ok {
					totalMap[phoneType][regInvite100Data.RegIp]++
				} else {
					totalMap[phoneType][regInvite100Data.RegIp] = 1
				}

				if inviteTime.IsZero() || invite100Time.IsZero() {
					if _, ok := timeoutMap[regInvite100Data.RegIp]; ok {
						timeoutMap[regInvite100Data.RegIp]++
					} else {
						timeoutMap[regInvite100Data.RegIp] = 1
					}
					logrus.Errorf("processRegInvite100 timeout phone_type=%d, reg_ip ip = %s, sid = %s, cseq = %s, invite_time = %v, invite_100_time = %v, retry_invite = %v\n",
						phoneType, regInvite100Data.RegIp, sid, cseq, inviteTime, invite100Time, retryInvite)
					continue
				}

				delayTime := invite100Time.Sub(inviteTime)
				regInvite100Data.DelayMS = delayTime.Microseconds()
				regInvite100Data.TimeoutCount = 0
				regInvite100DataTable = append(regInvite100DataTable, regInvite100Data)
				if regInvite100Data.DelayMS > 3000000 {
					logrus.Errorf("processRegInvite100 high delay phone_type=%d, reg_ip ip = %s, sid = %s, cseq = %s, invite_time = %v, invite_100_time = %v, delay_ms = %d\n",
						phoneType, regInvite100Data.RegIp, sid, cseq, inviteTime, invite100Time, regInvite100Data.DelayMS)
				}
			}
		}

		for regIp, timeoutCount := range timeoutMap {
			var regInvite100Data model.RegInvite100Data
			regInvite100Data.PhoneType = phoneType
			regInvite100Data.RegIp = regIp
			regInvite100Data.DelayMS = 0
			regInvite100Data.TimeoutCount = timeoutCount
			if totalCount, ok := totalMap[phoneType][regInvite100Data.RegIp]; ok {
				regInvite100Data.TotalCount = totalCount
			}
			regInvite100DataTable = append(regInvite100DataTable, regInvite100Data)
		}
	}

	return regInvite100DataTable, invalidSids
}

func (ss *SearchService) SearchRegInvite100() []model.RegInvite100Data {
	timeFrom := time.Now().Add(time.Duration(-2) * time.Minute)
	timeTo := time.Now().Add(time.Duration(-1) * time.Minute)
	logrus.Infof("SearchRegInvite100 search first timeFrom=%v, timeTo=%v\n", timeFrom, timeTo)
	searchData := []model.HepTable{}
	needCheckIps := ""
	needMissIps := ""
	regIpMap := make(map[string]bool)
	regIpMap["10.58.208.9"] = true
	regIpMap["10.109.132.15"] = true

	smIpMap := make(map[string]bool)
	smIpMap["10.58.209.9"] = true
	smIpMap["10.109.132.16"] = true

	for key, _ := range regIpMap {
		if len(needCheckIps) > 0 {
			needCheckIps += ","
		}
		needCheckIps += "'" + key + "'"
	}

	for key, _ := range config.Setting.OriginSbcIpMap {
		if len(needMissIps) > 0 {
			needMissIps += ","
		}
		needMissIps += "'" + key + "'"
	}

	for key, _ := range smIpMap {
		needMissIps += ",'" + key + "'"
	}

	query := "create_date between ? AND ? "
	query += " AND ("
	query += "(protocol_header->>'dstIp' not in (" + needMissIps + ") AND protocol_header->>'srcIp' in (" + needCheckIps + "))"
	query += " OR (protocol_header->>'dstIp' in (" + needCheckIps + ") AND protocol_header->>'srcIp' not in (" + needMissIps + "))"
	query += ")"
	query += " AND data_header->>'method' in ('INVITE', '100')"
	table := "hep_proto_1_call"
	limitCount := 10000
	for session := range ss.Session {
		searchTmp := []model.HepTable{}
		if err := ss.Session[session].Debug().
			Table(table).
			Where(query, timeFrom.Format(time.RFC3339), timeTo.Format(time.RFC3339)).
			Limit(limitCount).
			Find(&searchTmp).Error; err != nil {
			logrus.Errorln("SearchRegInvite100: We have got error: ", err)
		}

		if len(searchTmp) > 0 {
			searchData = append(searchData, searchTmp...)
		}
	}

	var regInvite100DataTable []model.RegInvite100Data

	if len(searchData) <= 0 {
		return regInvite100DataTable
	}

	if len(searchData) >= limitCount {
		logrus.Errorf("SearchRegInvite100 search count is reached limit time_from = %v, time_to = %v\n",
			timeFrom, timeTo)
	}

	regTotalMap := make(map[model.PHONE_TYPE]map[string]int)
	regTotalMap[model.E_PHONE_TYPE_HARD] = make(map[string]int)
	regTotalMap[model.E_PHONE_TYPE_SOFT] = make(map[string]int)
	regInvite100DataTable, invalidSids := ss.processRegInvite100(searchData, regIpMap, regTotalMap, true)
	logrus.Infof("SearchRegInvite100 search first finished dataCount=%d, invalidSids=%v, timeFrom=%v, timeTo=%v\n", len(regInvite100DataTable), invalidSids, timeFrom, timeTo)
	if len(invalidSids) <= 0 {
		return regInvite100DataTable
	}

	needSearchSids := ""
	for _, sid := range invalidSids {
		if len(needSearchSids) > 0 {
			needSearchSids += ","
		}
		needSearchSids += "'" + sid + "'"
	}

	timeFrom = time.Now().Add(time.Duration(-30) * time.Minute)
	timeTo = time.Now()
	logrus.Infof("SearchRegInvite100 search again timeFrom=%v, timeTo=%v\n", timeFrom, timeTo)
	query = "create_date between ? AND ?"
	query += " AND sid in (" + needSearchSids + ")"
	query += " AND protocol_header->>'dstIp' not in (" + needMissIps + ")"
	query += " AND protocol_header->>'srcIp' not in (" + needMissIps + ")"
	query += " AND data_header->>'method' in ('INVITE', '100', '180', '183', '200')"
	query += " AND data_header->>'cseq' ILIKE '%INVITE%'"
	table = "hep_proto_1_call"
	searchData = []model.HepTable{}
	for session := range ss.Session {
		searchTmp := []model.HepTable{}
		if err := ss.Session[session].Debug().
			Table(table).
			Where(query, timeFrom.Format(time.RFC3339), timeTo.Format(time.RFC3339)).
			Limit(limitCount).
			Find(&searchTmp).Error; err != nil {
			logrus.Errorln("SearchRegInvite100: We have got error: ", err)
		}

		if len(searchTmp) > 0 {
			searchData = append(searchData, searchTmp...)
		}
	}

	processTable, invalidSids := ss.processRegInvite100(searchData, regIpMap, regTotalMap, false)
	regInvite100DataTable = append(regInvite100DataTable, processTable...)
	if len(invalidSids) > 0 {
		logrus.Errorf("SearchRegInvite100 search again really invalidSids=%v, timeFrom=%v, timeTo=%v\n", invalidSids, timeFrom, timeTo)
	}

	return regInvite100DataTable
}

func (ss *SearchService) SearchCmNotify() []model.CMNotifyData {
	timeFrom := time.Now().Add(time.Duration(-1) * time.Minute)
	timeTo := time.Now()
	searchData := []model.HepTable{}
	needCheckIps := ""
	cmIpMap := make(map[string]bool)
	//cm
	cmIpMap["10.58.208.5"] = true
	cmIpMap["10.58.208.6"] = true
	cmIpMap["10.58.208.22"] = true
	cmIpMap["10.58.208.24"] = true
	cmIpMap["10.58.208.25"] = true
	cmIpMap["10.58.209.18"] = true
	cmIpMap["10.109.133.9"] = true
	cmIpMap["10.109.133.10"] = true
	cmIpMap["10.109.133.11"] = true
	cmIpMap["10.109.133.12"] = true
	cmIpMap["10.109.133.13"] = true
	cmIpMap["10.109.133.14"] = true

	//robot
	cmIpMap["10.58.208.36"] = true
	cmIpMap["10.58.208.38"] = true
	cmIpMap["10.58.208.39"] = true
	cmIpMap["10.58.208.42"] = true
	cmIpMap["10.58.208.43"] = true
	cmIpMap["10.58.209.33"] = true
	cmIpMap["10.58.209.34"] = true
	cmIpMap["10.58.209.35"] = true
	cmIpMap["10.109.133.4"] = true
	cmIpMap["10.109.133.7"] = true
	cmIpMap["10.109.133.8"] = true
	cmIpMap["10.109.132.23"] = true
	cmIpMap["10.109.132.24"] = true
	cmIpMap["10.109.133.27"] = true
	cmIpMap["10.109.133.28"] = true
	cmIpMap["10.109.133.29"] = true

	//voip
	cmIpMap["10.58.208.26"] = true
	cmIpMap["10.58.208.27"] = true
	cmIpMap["10.58.209.21"] = true
	cmIpMap["10.109.133.15"] = true
	cmIpMap["10.109.132.18"] = true
	cmIpMap["10.109.132.19"] = true

	for key, _ := range cmIpMap {
		if len(needCheckIps) > 0 {
			needCheckIps += ","
		}
		needCheckIps += "'" + key + "'"
	}

	query := "create_date between ? AND ?"
	query += " AND (protocol_header->>'dstIp' in (" + needCheckIps + ") OR protocol_header->>'srcIp' in (" + needCheckIps + "))"
	query += " AND data_header->>'method' in ('NOTIFY', '200', '408', '403', '415', '400', '503', '404', '477', '444')"
	table := "hep_proto_1_default"
	for session := range ss.Session {
		searchTmp := []model.HepTable{}
		if err := ss.Session[session].Debug().
			Table(table).
			Where(query, timeFrom.Format(time.RFC3339), timeTo.Format(time.RFC3339)).
			Limit(6000).
			Find(&searchTmp).Error; err != nil {
			logrus.Errorln("SearchCmNotify: We have got error: ", err)
		}

		if len(searchTmp) > 0 {
			searchData = append(searchData, searchTmp...)
		}
	}

	cmNotifyDataTable := []model.CMNotifyData{}
	if len(searchData) <= 0 {
		return cmNotifyDataTable
	}

	if len(searchData) >= 6000 {
		logrus.Errorf("SearchCmNotify search count is reached limit time_from = %v, time_to = %v\n",
			timeFrom, timeTo)
	}

	sort.Slice(searchData, func(i, j int) bool {
		return searchData[i].CreatedDate.Before(searchData[j].CreatedDate)
	})

	sipHepDataMap := make(map[string][]model.HepTable)
	for _, entry := range searchData {
		var protocolHeader model.ProtocolHeader
		json.Unmarshal(entry.ProtocolHeader, &protocolHeader)
		var dataHeader model.DataHeader
		json.Unmarshal(entry.DataHeader, &dataHeader)
		upperCaptureID := strings.ToUpper(protocolHeader.CaptureID)
		_, fromCM := cmIpMap[protocolHeader.SrcIP]
		_, toCM := cmIpMap[protocolHeader.DstIP]
		upperMethod := strings.ToUpper(dataHeader.Method)
		if !strings.Contains(upperCaptureID, "CM") {
			continue
		}

		if strings.EqualFold(upperMethod, "NOTIFY") && !fromCM {
			continue
		}

		if !strings.EqualFold(upperMethod, "NOTIFY") && !toCM {
			continue
		}

		sipHepDataMap[entry.Sid] = append(sipHepDataMap[entry.Sid], entry)
	}

	totalMap := make(map[int32]map[string]int)
	totalMap[model.E_NOTIFY_INVALID] = make(map[string]int)
	totalMap[model.E_NOTIFY_MAKE_CALL] = make(map[string]int)
	totalMap[model.E_NOTIFY_ANSWER] = make(map[string]int)
	timeoutMap := make(map[int32]map[string]int)
	timeoutMap[model.E_NOTIFY_INVALID] = make(map[string]int)
	timeoutMap[model.E_NOTIFY_MAKE_CALL] = make(map[string]int)
	timeoutMap[model.E_NOTIFY_ANSWER] = make(map[string]int)
	for sid, hepTables := range sipHepDataMap {
		var cmNotifyData model.CMNotifyData
		cmNotifyData.NotifyType = model.E_NOTIFY_INVALID
		var notifyTime time.Time
		var notifyResponseTime time.Time
		notifyTimeout := true
		for _, entryHep := range hepTables {
			var dataHeader model.DataHeader
			json.Unmarshal(entryHep.DataHeader, &dataHeader)
			upperMethod := strings.ToUpper(dataHeader.Method)
			if strings.EqualFold(upperMethod, "NOTIFY") && notifyTime.IsZero() {
				notifyTime = entryHep.CreatedDate
				if strings.Contains(entryHep.Raw, "MakeCall") {
					cmNotifyData.NotifyType = model.E_NOTIFY_MAKE_CALL
				} else if strings.Contains(entryHep.Raw, "Answer") {
					cmNotifyData.NotifyType = model.E_NOTIFY_ANSWER
				} else {
					logrus.Errorf("SearchCmNotify invalid notify type sid = %s\n", entryHep.Sid)
				}
			}

			if !strings.EqualFold(upperMethod, "NOTIFY") && notifyResponseTime.IsZero() {
				notifyResponseTime = entryHep.CreatedDate
				cmNotifyData.Method = dataHeader.Method
				notifyTimeout = false
			}

			var protocolHeader model.ProtocolHeader
			json.Unmarshal(entryHep.ProtocolHeader, &protocolHeader)

			if _, ok := cmIpMap[protocolHeader.SrcIP]; ok {
				cmNotifyData.CmIp = protocolHeader.SrcIP
			}

			if _, ok := cmIpMap[protocolHeader.DstIP]; ok {
				cmNotifyData.CmIp = protocolHeader.DstIP
			}
		}

		if _, ok := cmIpMap[cmNotifyData.CmIp]; !ok {
			logrus.Errorf("SearchCmNotify invalid cm_ip = %s, sid = %s, notify_time = %v, notify_response_time = %v, "+
				"time_from = %v, time_to = %v\n",
				cmNotifyData.CmIp, sid, notifyTime, notifyResponseTime, timeFrom, timeTo)
			continue
		}

		if notifyTime.IsZero() {
			logrus.Errorf("SearchCmNotify no notify data sid = %s, notify_time = %v, notify_response_time = %v, "+
				"time_from = %v, time_to = %v\n",
				sid, notifyTime, notifyResponseTime, timeFrom, timeTo)
			for _, entryHep := range hepTables {
				var dataHeader model.DataHeader
				json.Unmarshal(entryHep.DataHeader, &dataHeader)
				var protocolHeader model.ProtocolHeader
				json.Unmarshal(entryHep.ProtocolHeader, &protocolHeader)
				logrus.Errorf("SearchCmNotify sid = %s, create_date = %v, method = %s, src = %s, dst = %s, capture = %s\n",
					sid, entryHep.CreatedDate, dataHeader.Method, protocolHeader.SrcIP, protocolHeader.DstIP, protocolHeader.CaptureID)
			}
			continue
		}

		if _, ok := totalMap[cmNotifyData.NotifyType][cmNotifyData.CmIp]; ok {
			totalMap[cmNotifyData.NotifyType][cmNotifyData.CmIp]++
		} else {
			totalMap[cmNotifyData.NotifyType][cmNotifyData.CmIp] = 1
		}

		if notifyTimeout {
			if _, ok := timeoutMap[cmNotifyData.NotifyType][cmNotifyData.CmIp]; ok {
				timeoutMap[cmNotifyData.NotifyType][cmNotifyData.CmIp]++
			} else {
				timeoutMap[cmNotifyData.NotifyType][cmNotifyData.CmIp] = 1
			}
			logrus.Errorf("SearchCmNotify timeout cm_ip = %s, sid = %s, notify_time = %v, notify_response_time = %v, "+
				"time_from = %v, time_to = %v\n",
				cmNotifyData.CmIp, sid, notifyTime, notifyResponseTime, timeFrom, timeTo)
			continue
		}

		if notifyResponseTime.IsZero() {
			logrus.Errorf("SearchCmNotify invalid data sid = %s, notify_time = %v, notify_response_time = %v, "+
				"time_from = %v, time_to = %v\n",
				sid, notifyTime, notifyResponseTime, timeFrom, timeTo)
			continue
		}

		delayTime := notifyResponseTime.Sub(notifyTime)
		cmNotifyData.DelayMS = delayTime.Microseconds()
		cmNotifyData.TimeoutCount = 0
		cmNotifyDataTable = append(cmNotifyDataTable, cmNotifyData)
	}

	for notifyType, childMap := range timeoutMap {
		for cmIp, timeoutCount := range childMap {
			var cmNotifyData model.CMNotifyData
			cmNotifyData.CmIp = cmIp
			cmNotifyData.NotifyType = notifyType
			cmNotifyData.DelayMS = 0
			cmNotifyData.TimeoutCount = timeoutCount
			if totalCount, ok := totalMap[notifyType][cmNotifyData.CmIp]; ok {
				cmNotifyData.TotalCount = totalCount
			}
			cmNotifyDataTable = append(cmNotifyDataTable, cmNotifyData)
		}
	}

	return cmNotifyDataTable
}

func (ss *SearchService) processSbcNotify(processData []model.HepTable, totalMap map[int32]map[string]int, tryAgain bool) ([]model.SbcNotifyData, []string) {
	sbcNotifyDataTable := []model.SbcNotifyData{}
	invalidSids := []string{}
	if len(processData) <= 0 {
		return sbcNotifyDataTable, invalidSids
	}

	sort.Slice(processData, func(i, j int) bool {
		return processData[i].CreatedDate.Before(processData[j].CreatedDate)
	})

	sipHepDataMap := make(map[string][]model.HepTable)
	for _, entry := range processData {
		var protocolHeader model.ProtocolHeader
		json.Unmarshal(entry.ProtocolHeader, &protocolHeader)
		var dataHeader model.DataHeader
		json.Unmarshal(entry.DataHeader, &dataHeader)
		upperCaptureID := strings.ToUpper(protocolHeader.CaptureID)
		_, fromSbc := config.Setting.OriginSbcIpMap[protocolHeader.SrcIP]
		_, toSbc := config.Setting.OriginSbcIpMap[protocolHeader.DstIP]
		upperMethod := strings.ToUpper(dataHeader.Method)
		if !strings.Contains(upperCaptureID, "SBC") {
			continue
		}

		if strings.EqualFold(upperMethod, "NOTIFY") && !fromSbc {
			continue
		}

		if !strings.EqualFold(upperMethod, "NOTIFY") && !toSbc {
			continue
		}

		sipHepDataMap[entry.Sid] = append(sipHepDataMap[entry.Sid], entry)
	}

	timeoutMap := make(map[int32]map[string]int)
	timeoutMap[model.E_NOTIFY_INVALID] = make(map[string]int)
	timeoutMap[model.E_NOTIFY_MAKE_CALL] = make(map[string]int)
	timeoutMap[model.E_NOTIFY_ANSWER] = make(map[string]int)
	for sid, hepTables := range sipHepDataMap {
		var sbcNotifyData model.SbcNotifyData
		sbcNotifyData.NotifyType = model.E_NOTIFY_INVALID
		var notifyTime time.Time
		var notifyResponseTime time.Time
		retryNotify := false
		for _, entryHep := range hepTables {
			var protocolHeader model.ProtocolHeader
			json.Unmarshal(entryHep.ProtocolHeader, &protocolHeader)

			if _, ok := config.Setting.OriginSbcIpMap[protocolHeader.SrcIP]; ok {
				sbcNotifyData.SbcIp = protocolHeader.SrcIP
			}

			if _, ok := config.Setting.OriginSbcIpMap[protocolHeader.DstIP]; ok {
				sbcNotifyData.SbcIp = protocolHeader.DstIP
			}

			var dataHeader model.DataHeader
			json.Unmarshal(entryHep.DataHeader, &dataHeader)
			upperMethod := strings.ToUpper(dataHeader.Method)
			if strings.EqualFold(upperMethod, "NOTIFY") {
				if notifyTime.IsZero() {
					notifyTime = entryHep.CreatedDate
					if strings.Contains(entryHep.Raw, "MakeCall") {
						sbcNotifyData.NotifyType = model.E_NOTIFY_MAKE_CALL
					} else if strings.Contains(entryHep.Raw, "Answer") {
						sbcNotifyData.NotifyType = model.E_NOTIFY_ANSWER
					} else {
						logrus.Errorf("processSbcNotify invalid notify type sbc_ip = %s, sid = %s\n",
							sbcNotifyData.SbcIp, entryHep.Sid)
					}
				} else {
					retryNotify = true
				}
			}

			if !strings.EqualFold(upperMethod, "NOTIFY") && notifyResponseTime.IsZero() {
				notifyResponseTime = entryHep.CreatedDate
				sbcNotifyData.Method = dataHeader.Method
			}
		}

		if _, ok := config.Setting.OriginSbcIpMap[sbcNotifyData.SbcIp]; !ok {
			logrus.Errorf("processSbcNotify invalid sbc_ip = %s, sid = %s, notify_time = %v, notify_response_time = %v\n",
				sbcNotifyData.SbcIp, sid, notifyTime, notifyResponseTime)
			continue
		}

		if notifyTime.IsZero() || notifyResponseTime.IsZero() {
			invalidSids = append(invalidSids, sid)
			if tryAgain {
				break
			}
		}

		if _, ok := totalMap[sbcNotifyData.NotifyType][sbcNotifyData.SbcIp]; ok {
			totalMap[sbcNotifyData.NotifyType][sbcNotifyData.SbcIp]++
		} else {
			totalMap[sbcNotifyData.NotifyType][sbcNotifyData.SbcIp] = 1
		}

		if notifyTime.IsZero() || notifyResponseTime.IsZero() {
			//if retryNotify {
			if _, ok := timeoutMap[sbcNotifyData.NotifyType][sbcNotifyData.SbcIp]; ok {
				timeoutMap[sbcNotifyData.NotifyType][sbcNotifyData.SbcIp]++
			} else {
				timeoutMap[sbcNotifyData.NotifyType][sbcNotifyData.SbcIp] = 1
			}
			//}
			logrus.Errorf("processSbcNotify timeout sbc_ip = %s, sid = %s, notify_time = %v, notify_response_time = %v, retry_notify = %v\n",
				sbcNotifyData.SbcIp, sid, notifyTime, notifyResponseTime, retryNotify)
			continue
		}

		delayTime := notifyResponseTime.Sub(notifyTime)
		sbcNotifyData.DelayMS = delayTime.Microseconds()
		sbcNotifyData.TimeoutCount = 0
		sbcNotifyDataTable = append(sbcNotifyDataTable, sbcNotifyData)
		if sbcNotifyData.DelayMS > 5000000 {
			logrus.Errorf("processSbcNotify high delay notify_type = %d, sbc_ip ip = %s, sid = %s, notify_time = %v, notify_response_time = %v, delay_ms = %d\n",
				sbcNotifyData.NotifyType, sbcNotifyData.SbcIp, sid, notifyTime, notifyResponseTime, sbcNotifyData.DelayMS)
		}
	}

	for notifyType, childMap := range timeoutMap {
		for sbcIp, timeoutCount := range childMap {
			var sbcNotifyData model.SbcNotifyData
			sbcNotifyData.SbcIp = sbcIp
			sbcNotifyData.NotifyType = notifyType
			sbcNotifyData.DelayMS = 0
			sbcNotifyData.TimeoutCount = timeoutCount
			if totalCount, ok := totalMap[notifyType][sbcNotifyData.SbcIp]; ok {
				sbcNotifyData.TotalCount = totalCount
			}
			sbcNotifyDataTable = append(sbcNotifyDataTable, sbcNotifyData)
		}
	}

	return sbcNotifyDataTable, invalidSids
}

func (ss *SearchService) SearchSbcNotify() []model.SbcNotifyData {
	timeFrom := time.Now().Add(time.Duration(-2) * time.Minute)
	timeTo := time.Now().Add(time.Duration(-1) * time.Minute)
	logrus.Infof("SearchSbcNotify search first timeFrom=%v, timeTo=%v\n", timeFrom, timeTo)
	searchData := []model.HepTable{}
	needCheckIps := ""

	for key, _ := range config.Setting.OriginSbcIpMap {
		if len(needCheckIps) > 0 {
			needCheckIps += ","
		}
		needCheckIps += "'" + key + "'"
	}

	query := "create_date between ? AND ?"
	query += " AND (protocol_header->>'dstIp' in (" + needCheckIps + ") OR protocol_header->>'srcIp' in (" + needCheckIps + "))"
	query += " AND data_header->>'method' in ('NOTIFY', '200', '408', '403', '415', '400', '503', '404', '477', '444')"
	table := "hep_proto_1_default"
	limitCount := 10000
	for session := range ss.Session {
		searchTmp := []model.HepTable{}
		if err := ss.Session[session].Debug().
			Table(table).
			Where(query, timeFrom.Format(time.RFC3339), timeTo.Format(time.RFC3339)).
			Limit(limitCount).
			Find(&searchTmp).Error; err != nil {
			logrus.Errorln("SearchSbcNotify: We have got error: ", err)
		}

		if len(searchTmp) > 0 {
			searchData = append(searchData, searchTmp...)
		}
	}

	var sbcNotifyDataTable []model.SbcNotifyData
	if len(searchData) <= 0 {
		return sbcNotifyDataTable
	}

	if len(searchData) >= limitCount {
		logrus.Errorf("SearchSbcNotify search count is reached limit time_from = %v, time_to = %v\n",
			timeFrom, timeTo)
	}

	totalMap := make(map[int32]map[string]int)
	totalMap[model.E_NOTIFY_INVALID] = make(map[string]int)
	totalMap[model.E_NOTIFY_MAKE_CALL] = make(map[string]int)
	totalMap[model.E_NOTIFY_ANSWER] = make(map[string]int)
	sbcNotifyDataTable, invalidSids := ss.processSbcNotify(searchData, totalMap, true)
	logrus.Infof("SearchSbcNotify search first finished invalidSids=%v, timeFrom=%v, timeTo=%v\n", invalidSids, timeFrom, timeTo)
	if len(invalidSids) <= 0 {
		return sbcNotifyDataTable
	}

	needSearchSids := ""
	for _, sid := range invalidSids {
		if len(needSearchSids) > 0 {
			needSearchSids += ","
		}
		needSearchSids += "'" + sid + "'"
	}

	timeFrom = time.Now().Add(time.Duration(-30) * time.Minute)
	timeTo = time.Now()
	logrus.Infof("SearchSbcNotify search again timeFrom=%v, timeTo=%v\n", timeFrom, timeTo)
	query = "create_date between ? AND ?"
	query += " AND sid in (" + needSearchSids + ")"
	query += " AND data_header->>'method' in ('NOTIFY', '200', '408', '403', '415', '400', '503', '404', '477', '444')"
	table = "hep_proto_1_default"
	searchData = []model.HepTable{}
	for session := range ss.Session {
		searchTmp := []model.HepTable{}
		if err := ss.Session[session].Debug().
			Table(table).
			Where(query, timeFrom.Format(time.RFC3339), timeTo.Format(time.RFC3339)).
			Limit(limitCount).
			Find(&searchTmp).Error; err != nil {
			logrus.Errorln("SearchSbcNotify: We have got error: ", err)
		}

		if len(searchTmp) > 0 {
			searchData = append(searchData, searchTmp...)
		}
	}

	processTable, invalidSids := ss.processSbcNotify(searchData, totalMap, false)
	sbcNotifyDataTable = append(sbcNotifyDataTable, processTable...)
	if len(invalidSids) > 0 {
		logrus.Errorf("SearchSbcNotify search again really invalidSids=%v, timeFrom=%v, timeTo=%v\n", invalidSids, timeFrom, timeTo)
	}

	return sbcNotifyDataTable
}

// this method create new user in the database
// it doesn't check internally whether all the validation are applied or not
func (ss *SearchService) GetDBNodeList(searchObject *model.SearchObject) (string, error) {

	reply := gabs.New()
	reply.Set(1, "total")
	reply.Set("", "data")

	return reply.String(), nil
}

// this method create new user in the database
// it doesn't check internally whether all the validation are applied or not
func (ss *SearchService) GetDecodedMessageByID(searchObject *model.SearchObject) (string, error) {
	//table := "hep_proto_1_default"
	sLimit := searchObject.Param.Limit
	searchData := []model.HepTable{}
	searchFromTime := time.Unix(searchObject.Timestamp.From/int64(time.Microsecond), 0)
	searchToTime := time.Unix(searchObject.Timestamp.To/int64(time.Microsecond), 0)
	Data, _ := json.Marshal(searchObject.Param.Search)
	sData, _ := gabs.ParseJSON(Data)
	sql := "create_date between ? and ?"
	var doDecode = false

	for key := range sData.ChildrenMap() {
		//table = "hep_proto_" + key
		if sData.Exists(key) {
			elems := sData.Search(key, "id").Data().(float64)
			sql = sql + " and id = " + fmt.Sprintf("%d", int(elems))
			/* check if we have to decode */
			if ss.Decoder.Active && heputils.ItemExists(ss.Decoder.Protocols, key) {
				doDecode = true
			}
		}
	}

	tableArray := [3]string{"hep_proto_1_call", "hep_proto_1_default", "hep_proto_1_registration"}
SqlExecute:
	for index := 0; index < len(tableArray); index++ {
		table := tableArray[index]
		for session := range ss.Session {
			/* if node doesnt exists - continue */
			if !heputils.ElementExists(searchObject.Param.Location.Node, session) {

				continue
			}

			searchTmp := []model.HepTable{}
			ss.Session[session].
				Table(table).
				Where(sql, searchFromTime, searchToTime).
				Limit(sLimit).
				Find(&searchTmp)

			if len(searchTmp) > 0 {
				for val := range searchTmp {
					searchTmp[val].DBNode = session
					searchTmp[val].Node = session

				}
				searchData = append(searchData, searchTmp...)
				break SqlExecute
			}
		}
	}

	rows, _ := json.Marshal(searchData)
	data, _ := gabs.ParseJSON(rows)
	dataReply := gabs.Wrap([]interface{}{})
	for _, value := range data.Children() {
		dataElement := gabs.New()
		newData := gabs.New()
		for k := range value.ChildrenMap() {
			switch k {
			case "raw":
				/* doing sipIsup extraction */
				if doDecode {
					if decodedData, err := ss.excuteExternalDecoder(value); err == nil {
						newData.Set(decodedData, "decoded")
					}
				}
			}
		}
		dataElement.Merge(newData)
		dataReply.ArrayAppend(dataElement.Data())
	}
	dataKeys := gabs.Wrap([]interface{}{})
	for _, v := range dataReply.Children() {
		for key := range v.ChildrenMap() {
			if !function.ArrayKeyExits(key, dataKeys) {
				dataKeys.ArrayAppend(key)
			}
		}
	}

	total, _ := dataReply.ArrayCount()

	reply := gabs.New()
	reply.Set(total, "total")
	reply.Set(dataReply.Data(), "data")

	return reply.String(), nil
}

// this method create new user in the database
// it doesn't check internally whether all the validation are applied or not
func (ss *SearchService) GetMessageByID(searchObject *model.SearchObject) (string, error) {
	//table := "hep_proto_1_default"
	sLimit := searchObject.Param.Limit
	searchData := []model.HepTable{}
	searchFromTime := time.Unix(searchObject.Timestamp.From/int64(time.Microsecond), 0)
	searchToTime := time.Unix(searchObject.Timestamp.To/int64(time.Microsecond), 0)
	Data, _ := json.Marshal(searchObject.Param.Search)
	sData, _ := gabs.ParseJSON(Data)
	sql := "create_date between ? and ?"
	var sipExist = false

	for key := range sData.ChildrenMap() {
		//table = "hep_proto_" + key
		if sData.Exists(key) {
			if key == "1_call" {
				sipExist = true
			}
			elems := sData.Search(key, "id").Data().(float64)
			sql = sql + " and id = " + fmt.Sprintf("%d", int(elems))
		}
	}

	tableArray := [3]string{"hep_proto_1_call", "hep_proto_1_default", "hep_proto_1_registration"}
SqlExecute:
	for index := 0; index < len(tableArray); index++ {
		table := tableArray[index]
		for session := range ss.Session {
			/* if node doesnt exists - continue */
			if !heputils.ElementExists(searchObject.Param.Location.Node, session) {
				continue
			}

			searchTmp := []model.HepTable{}
			ss.Session[session].Debug().
				Table(table).
				Where(sql, searchFromTime, searchToTime).
				Limit(sLimit).
				Find(&searchTmp)

			if len(searchTmp) > 0 {
				for val := range searchTmp {
					searchTmp[val].Node = session
					searchTmp[val].DBNode = session

				}
				searchData = append(searchData, searchTmp...)
				break SqlExecute
			}
		}
	}

	rows, _ := json.Marshal(searchData)
	data, _ := gabs.ParseJSON(rows)
	dataReply := gabs.Wrap([]interface{}{})
	for _, value := range data.Children() {
		dataElement := gabs.New()
		for k, v := range value.ChildrenMap() {
			switch k {
			case "data_header", "protocol_header":
				dataElement.Merge(v)
			case "id", "sid":
				newData := gabs.New()
				newData.Set(v.Data().(interface{}), k)
				dataElement.Merge(newData)

			case "raw":
				newData := gabs.New()
				/* doing sipIsup extraction */
				if sipExist {
					rawElement := fmt.Sprintf("%v", v.Data().(interface{}))
					newData.Set(heputils.IsupToHex(rawElement), k)
				} else {
					newData.Set(v.Data().(interface{}), k)
				}

				dataElement.Merge(newData)
			}
		}
		dataReply.ArrayAppend(dataElement.Data())
	}
	dataKeys := gabs.Wrap([]interface{}{})
	for _, v := range dataReply.Children() {
		for key := range v.ChildrenMap() {
			if !function.ArrayKeyExits(key, dataKeys) {
				dataKeys.ArrayAppend(key)
			}
		}
	}

	total, _ := dataReply.ArrayCount()

	reply := gabs.New()
	reply.Set(total, "total")
	reply.Set(dataReply.Data(), "data")
	reply.Set(dataKeys.Data(), "keys")

	return reply.String(), nil
}

//this method create new user in the database
//it doesn't check internally whether all the validation are applied or not
func (ss *SearchService) excuteExternalDecoder(dataRecord *gabs.Container) (interface{}, error) {

	if ss.Decoder.Active {
		logrus.Debug("Trying to debug using external decoder")
		logrus.Debug(fmt.Sprintf("Decoder to [%s, %s, %v]\n", ss.Decoder.Binary, ss.Decoder.Param, ss.Decoder.Protocols))
		//cmd := exec.Command(ss.Decoder.Binary, ss.Decoder.Param)
		var buffer bytes.Buffer
		export := exportwriter.NewWriter(buffer)
		var rootExecute = false

		// pcap export
		if err := export.WritePcapHeader(65536, 1); err != nil {
			logrus.Errorln("write error to the pcap header", err)
			return nil, err
		}

		if err := export.WriteDataPcapBuffer(dataRecord); err != nil {
			logrus.Errorln("write error to the pcap buffer", err)
			return nil, err
		}

		cmd := exec.Command(ss.Decoder.Binary, "-Q", "-T", "json", "-l", "-i", "-", ss.Decoder.Param)

		/*check if we root under root - changing to an user */
		uid, gid := os.Getuid(), os.Getgid()

		if uid == 0 || gid == 0 {
			logrus.Info("running under root/wheel: UID: [%d], GID: [%d] - [%d] - [%d]. Changing to user...", uid, gid, ss.Decoder.UID, ss.Decoder.GID)
			if ss.Decoder.UID != 0 && ss.Decoder.GID != 0 {
				logrus.Info("Changing to: UID: [%d], GID: [%d]", uid, gid)
				cmd.SysProcAttr = &syscall.SysProcAttr{
					Credential: &syscall.Credential{
						Uid: ss.Decoder.UID, Gid: ss.Decoder.GID,
						NoSetGroups: true,
					},
				}
			} else {
				logrus.Error("You run external decoder under root! Please set UID/GID in the config")
				rootExecute = true
			}
		}

		stdin, err := cmd.StdinPipe()
		if err != nil {
			logrus.Error("Bad cmd stdin", err)
			return nil, err
		}
		go func() {
			defer stdin.Close()
			io.WriteString(stdin, export.Buffer.String())
			return
		}()

		out, err := cmd.CombinedOutput()
		if err != nil {
			logrus.Error("Bad combined output: ", err)
			return nil, err
		}

		var skipElement = 0

		/* this is fix if you run the webapp under root */
		if rootExecute {
			/* limit search String */
			maxEl := len(out)
			if maxEl > 100 {
				maxEl = 100
			}
			for i := 0; i < maxEl; i++ {
				if string(out[i]) == "[" || string(out[i]) == "{" {
					skipElement = i
					break
				}
			}
		}

		sData, err := gabs.ParseJSON(out[skipElement:])
		if err != nil {
			logrus.Error("bad json", err)
			return nil, err
		}

		return sData.Data(), nil
	}

	return nil, errors.New("decoder not active. You should not be here")
}

//this method create new user in the database
//it doesn't check internally whether all the validation are applied or not
func (ss *SearchService) GetTransaction(table string, data []byte, tableMappingSchema []model.TableMappingSchema, doexp bool,
	aliasData map[string]string, typeReport int, nodes []string, settingService *UserSettingsService, userGroup string) (string, error) {
	var dataWhere []interface{}
	requestData, _ := gabs.ParseJSON(data)
	for key, value := range requestData.Search("param", "search").ChildrenMap() {
		table = "hep_proto_" + key
		for _, v := range value.Search("callid").Data().([]interface{}) {
			dataWhere = append(dataWhere, v)
		}
	}

	timeWhereFrom := requestData.S("timestamp", "from").Data().(float64)
	timeWhereTo := requestData.S("timestamp", "to").Data().(float64)
	timeFrom := time.Unix(int64(timeWhereFrom/float64(time.Microsecond)), 0).UTC()
	timeTo := time.Unix(int64(timeWhereTo/float64(time.Microsecond)), 0).UTC()

	dataRow := ss.getCorrelationFromDB(dataWhere, tableMappingSchema, timeFrom, timeTo, nodes, settingService, userGroup, aliasData)

	transactionTables := convert2TransactionTables(dataRow)

	//debug log info
	logrus.Infof("[GetTransaction] sorted ttransactionTables = %#v\n", transactionTables)
	dataRow = []model.HepTable{}
	for _, entryT := range transactionTables {
		for _, entryM := range entryT.TMethods {
			for elem := entryM.BodyList.Front(); elem != nil; elem = elem.Next() {
				dataRow = append(dataRow, elem.Value.(model.HepTable))
			}
		}
	}

	marshalData, _ := json.Marshal(dataRow)
	jsonParsed, _ := gabs.ParseJSON(marshalData)

	if typeReport == 0 {
		reply := ss.getTransactionSummary(jsonParsed, aliasData)
		return reply, nil
	} else {

		var buffer bytes.Buffer
		export := exportwriter.NewWriter(buffer)

		// pcap export
		if typeReport == 1 {
			err := export.WritePcapHeader(65536, 1)
			if err != nil {
				logrus.Errorln("write error to the pcap header", err)
			}
		}
		for _, h := range jsonParsed.Children() {

			if typeReport == 2 {
				err := export.WriteDataToBuffer(h)
				if err != nil {
					logrus.Errorln("write error to the buffer", err)
				}
			} else if typeReport == 1 {
				err := export.WriteDataPcapBuffer(h)
				if err != nil {
					logrus.Errorln("write error to the pcap buffer", err)
				}
			}
		}

		if typeReport == 1 {
			return export.Buffer.String(), nil
		}
		return export.Buffer.String(), nil
	}
}

func (ss *SearchService) GetTransactionV2(data []byte, tableMappingSchema []model.TableMappingSchema,
	aliasData map[string]string, nodes []string, settingService *UserSettingsService, userGroup string) (string, error) {
	var dataWhere []interface{}
	requestData, _ := gabs.ParseJSON(data)
	for _, value := range requestData.Search("param", "search").ChildrenMap() {
		for _, v := range value.Search("callid").Data().([]interface{}) {
			dataWhere = append(dataWhere, v)
		}
	}

	timeWhereFrom := requestData.S("timestamp", "from").Data().(float64)
	timeWhereTo := requestData.S("timestamp", "to").Data().(float64)
	timeFrom := time.Unix(int64(timeWhereFrom/float64(time.Microsecond)), 0).UTC()
	timeTo := time.Unix(int64(timeWhereTo/float64(time.Microsecond)), 0).UTC()

	dataRow := ss.getCorrelationFromDB(dataWhere, tableMappingSchema, timeFrom, timeTo, nodes, settingService, userGroup, aliasData)

	transactionTables := convert2TransactionTables(dataRow)

	//debug log info
	logrus.Infof("[GetTransactionV2] sorted ttransactionTables = %#v\n", transactionTables)
	reply := ss.getTransactionSummaryV2(transactionTables, aliasData)
	return reply, nil
}

func (ss *SearchService) getCorrelationFromDB(dataWhere []interface{}, tableMappingSchema []model.TableMappingSchema,
	timeFrom, timeTo time.Time, nodes []string, settingService *UserSettingsService, userGroup string,
	aliasData map[string]string) []model.HepTable {
	var dataRow []model.HepTable
	duplicateIdFilter := make(map[int64]bool)
	duplicateRawFilter := make(map[string]string)
	tableArray := [3]string{"hep_proto_1_call", "hep_proto_1_default", "hep_proto_1_registration"}
	for index := 0; index < len(tableArray); index++ {
		table := tableArray[index]
		tempDataRow, _ := ss.GetTransactionData(table, "sid", dataWhere, timeFrom, timeTo, nodes, userGroup, false)
		if len(tempDataRow) <= 0 {
			continue
		}
		sort.Slice(tempDataRow, func(i, j int) bool {
			var protocolHeaderI model.ProtocolHeader
			json.Unmarshal(tempDataRow[i].ProtocolHeader, &protocolHeaderI)
			var protocolHeaderJ model.ProtocolHeader
			json.Unmarshal(tempDataRow[j].ProtocolHeader, &protocolHeaderJ)

			//return protocolHeaderI.CaptureID < protocolHeaderJ.CaptureID && tempDataRow[i].CreatedDate.Before(tempDataRow[j].CreatedDate)
			return tempDataRow[i].CreatedDate.Before(tempDataRow[j].CreatedDate)
		})

		for _, entry := range tempDataRow {
			if duplicateIdFilter[entry.Id] {
				continue
			}
			duplicateIdFilter[entry.Id] = true

			var protocolHeader model.ProtocolHeader
			json.Unmarshal(entry.ProtocolHeader, &protocolHeader)
			captureId, exists := duplicateRawFilter[entry.Raw]
			captureStrings := strings.Split(protocolHeader.CaptureID, "_")
			//REG-FAT[100026750]_10.2.38.203
			if len(captureStrings) == 2 {
				if _, ok := aliasData[captureStrings[1]]; !ok {
					aliasData[captureStrings[1]] = captureStrings[0]
				}
			}

			if exists && captureId != protocolHeader.CaptureID {
				continue
			}
			duplicateRawFilter[entry.Raw] = protocolHeader.CaptureID
			dataRow = append(dataRow, entry)
		}
	}

	if len(dataRow) <= 0 {
		return dataRow
	}

	marshalData, _ := json.Marshal(dataRow)
	jsonParsed, _ := gabs.ParseJSON(marshalData)

	for index := 0; index < len(tableMappingSchema); index++ {
		correlationJSON := tableMappingSchema[index].CorrelationMapping
		correlation, _ := gabs.ParseJSON(correlationJSON)
		var dataSrcField = make(map[string][]interface{})

		if len(correlationJSON) > 0 {
			// S is shorthand for Search
			for _, child := range jsonParsed.Search().Children() {
				for _, corrChild := range correlation.Search().Children() {
					sf := corrChild.Search("source_field").Data().(string)
					nKey := make(map[string][]interface{})
					if strings.Index(sf, ".") > -1 {
						elemArray := strings.Split(sf, ".")
						switch child.Search(elemArray[0], elemArray[1]).Data().(type) {
						case string:
							nKey[sf] = append(nKey[sf], child.Search(elemArray[0], elemArray[1]).Data().(string))
						case float64:
							nKey[sf] = append(nKey[sf], child.Search(elemArray[0], elemArray[1]).Data().(float64))
						}
					} else {
						nKey[sf] = append(nKey[sf], child.Search(sf).Data().(string))
					}
					if len(nKey) != 0 {
						for _, v := range nKey[sf] {
							if !function.KeyExits(v, dataSrcField[sf]) {
								dataSrcField[sf] = append(dataSrcField[sf], v)
							}
						}
					}
				}
			}
		}

		var foundCidData []string

		for _, corrs := range correlation.Children() {
			var from time.Time
			var to time.Time

			sourceField := corrs.Search("source_field").Data().(string)
			lookupID := corrs.Search("lookup_id").Data().(float64)
			lookupProfile := corrs.Search("lookup_profile").Data().(string)
			lookupField := corrs.Search("lookup_field").Data().(string)
			lookupRange := corrs.Search("lookup_range").Data().([]interface{})
			newWhereData := dataSrcField[sourceField]
			likeSearch := false

			if len(newWhereData) == 0 {
				continue
			}

			table := "hep_proto_" + strconv.FormatFloat(lookupID, 'f', 0, 64) + "_" + lookupProfile

			logrus.Printf("[getCorrelationFromDB] search table name = %#v \n", table)
			if len(lookupRange) > 0 {
				from = timeFrom.Add(time.Duration(lookupRange[0].(float64)) * time.Second).UTC()
				to = timeTo.Add(time.Duration(lookupRange[1].(float64)) * time.Second).UTC()
			}
			if lookupID == 0 {
				logrus.Error("We need to implement remote call here")
			} else {
				if sourceField == "data_header.callid" {

					logrus.Debug(lookupProfile)
					logrus.Debug(lookupField)
				}

				if corrs.Exists("input_function_js") {
					inputFunction := corrs.Search("input_function_js").Data().(string)
					logrus.Debug("Input function: ", inputFunction)
					newDataArray := executeJSInputFunction(inputFunction, newWhereData)
					logrus.Debug("sid array before JS:", newWhereData)
					if newDataArray != nil {
						newWhereData = append(newWhereData, newDataArray...)
					}
				}

				if corrs.Exists("input_script") {
					inputScript := corrs.Search("input_script").Data().(string)
					logrus.Debug("Input function: ", inputScript)
					dataScript, err := settingService.GetScriptByParam("scripts", inputScript)
					if err == nil {
						scriptNew, _ := strconv.Unquote(dataScript)
						logrus.Debug("OUR script:", scriptNew)
						newDataArray := executeJSInputFunction(scriptNew, newWhereData)
						logrus.Debug("sid array before JS:", newWhereData)
						if newDataArray != nil {
							newWhereData = append(newWhereData, newDataArray...)
							logrus.Debug("sid array after JS:", newWhereData)
						}
					}
				}

				if len(foundCidData) > 0 {
					for _, v := range foundCidData {
						newWhereData = append(newWhereData, v)
					}
				}
				if corrs.Exists("like_search") && corrs.Search("like_search").Data().(bool) {
					likeSearch = true
				}

				newDataRow, _ := ss.GetTransactionData(table, lookupField, newWhereData, from, to, nodes, userGroup, likeSearch)
				logrus.Printf("[getCorrelationFromDB] search newDataRow count = %d \n", len(newDataRow))
				if corrs.Exists("append_sid") && corrs.Search("append_sid").Data().(bool) {
					marshalData, _ = json.Marshal(newDataRow)
					jsonParsed, _ = gabs.ParseJSON(marshalData)
					for _, value := range jsonParsed.Children() {
						elems := value.Search("sid").Data().(string)
						if !heputils.ItemExists(foundCidData, elems) {
							foundCidData = append(foundCidData, elems)
						}
					}
				}

				for _, entry := range newDataRow {
					if duplicateIdFilter[entry.Id] {
						continue
					}
					duplicateIdFilter[entry.Id] = true

					var protocolHeader model.ProtocolHeader
					json.Unmarshal(entry.ProtocolHeader, &protocolHeader)
					captureId, exists := duplicateRawFilter[entry.Raw]
					if exists && captureId != protocolHeader.CaptureID {
						continue
					}
					duplicateRawFilter[entry.Raw] = protocolHeader.CaptureID
					dataRow = append(dataRow, entry)
				}

				logrus.Debug("Correlation data len:", len(dataRow))

				if corrs.Exists("output_script") {
					outputScript := corrs.Search("output_script").Data().(string)
					logrus.Debug("Output function: ", outputScript)
					dataScript, err := settingService.GetScriptByParam("scripts", outputScript)
					if err == nil {
						scriptNew, _ := strconv.Unquote(dataScript)
						logrus.Debug("OUR script:", scriptNew)
						newDataRaw := executeJSOutputFunction(scriptNew, dataRow)
						//logrus.Debug("sid array before JS:", newDataRaw)
						if newDataRaw != nil {
							dataRow = newDataRaw
							//logrus.Debug("sid array after JS:", dataRow)
						}
					}
				}
			}
		}
	}

	return dataRow
}

func convert2TransactionTables(dataRow []model.HepTable) []*model.TransactionTable {
	sort.Slice(dataRow, func(i, j int) bool {
		return dataRow[i].CreatedDate.Before(dataRow[j].CreatedDate)
	})

	var transactionTables []*model.TransactionTable
	for _, entry := range dataRow {
		var protocolHeader model.ProtocolHeader
		json.Unmarshal(entry.ProtocolHeader, &protocolHeader)
		var dataHeader model.DataHeader
		json.Unmarshal(entry.DataHeader, &dataHeader)

		var transactionTable *model.TransactionTable
		for _, entryT := range transactionTables {
			if strings.Contains(entryT.ViaBranch, dataHeader.ViaBranch) ||
				strings.Contains(dataHeader.ViaBranch, entryT.ViaBranch) {
				transactionTable = entryT
				break
			}
		}

		if transactionTable == nil {
			transactionTable = new(model.TransactionTable)
			transactionTable.BeginDate = entry.CreatedDate
			transactionTable.FromUser = dataHeader.FromUser
			transactionTable.ToUser = dataHeader.ToUser
			transactionTables = append(transactionTables, transactionTable)
		}

		if len(transactionTable.ViaBranch) < len(dataHeader.ViaBranch) {
			transactionTable.ViaBranch = dataHeader.ViaBranch
		}

		var transactionMethod *model.TransactionMethod
		for _, entryM := range transactionTable.TMethods {
			if strings.Contains(entryM.Name, dataHeader.Method) {
				transactionMethod = entryM
				break
			}
		}

		if transactionMethod == nil {
			transactionMethod = new(model.TransactionMethod)
			if dataHeader.Method == "INVITE" {
				if strings.Contains(entry.Raw, "a=sendonly") ||
					strings.Contains(entry.Raw, "a=recvonly") ||
					strings.Contains(entry.Raw, "a=inactive") {
					transactionMethod.Name = "INVITE[HOLD]"
				} else {
					transactionMethod.Name = "INVITE"
				}

			} else if dataHeader.Method == "NOTIFY" {
				if strings.Contains(entry.Raw, "MakeCall") {
					transactionMethod.Name = "NOTIFY[MAKECALL]"
				} else if strings.Contains(entry.Raw, "Answer") {
					transactionMethod.Name = "NOTIFY[ANSWER]"
				} else {
					transactionMethod.Name = dataHeader.Method
				}
			} else {
				transactionMethod.Name = dataHeader.Method
			}

			transactionMethod.CSeq = dataHeader.CSeq
			transactionMethod.BeginDate = entry.CreatedDate
			transactionMethod.BodyList = list.New()
			transactionTable.TMethods = append(transactionTable.TMethods, transactionMethod)
		}
		transactionMethod.BodyList.PushBack(entry)
	}

	for _, entryT := range transactionTables {
		for _, entryM := range entryT.TMethods {
			if !strings.Contains(entryT.Name, entryM.Name) {
				if len(entryT.Name) <= 0 {
					entryT.Name = entryM.Name
				} else {
					entryT.Name = entryT.Name + "|" + entryM.Name
				}

				if strings.HasPrefix(entryM.Name, "4") {
					if len(entryT.ErrorName) <= 0 {
						entryT.ErrorName = entryM.Name
					} else {
						entryT.ErrorName = entryT.ErrorName + "|" + entryM.Name
					}
				}
			}

			newBodyList := list.New()
			for rightElem := entryM.BodyList.Front(); rightElem != nil; rightElem = rightElem.Next() {
				rightHepTable := rightElem.Value.(model.HepTable)
				var rightDataHeader model.DataHeader
				json.Unmarshal(rightHepTable.DataHeader, &rightDataHeader)

				leftElem := newBodyList.Back()
				for ; leftElem != nil; leftElem = leftElem.Prev() {
					leftHepTable := leftElem.Value.(model.HepTable)
					var leftDataHeader model.DataHeader
					json.Unmarshal(leftHepTable.DataHeader, &leftDataHeader)
					if isInsertAfter(entryT.ViaBranch, &leftDataHeader, &rightDataHeader) {
						break
					}
				}

				if leftElem == nil {
					newBodyList.PushFront(rightHepTable)
				} else {
					newBodyList.InsertAfter(rightHepTable, leftElem)
				}
			}
			entryM.BodyList = newBodyList
		}
	}

	return transactionTables
}

func isInsertAfter(viaBranch string, left *model.DataHeader, right *model.DataHeader) bool {
	viaBranchList := strings.Split(viaBranch, ";")
	leftViaBranchList := strings.Split(left.ViaBranch, ";")
	leftViaCount := len(leftViaBranchList)
	if leftViaCount == 1 {
		leftViaCount = len(viaBranchList)
		for i := 0; i < len(viaBranchList); i++ {
			if viaBranchList[i] == left.ViaBranch {
				break
			}
			leftViaCount--
		}
	}
	rightViaBranchList := strings.Split(right.ViaBranch, ";")
	rightViaCount := len(rightViaBranchList)
	if rightViaCount == 1 {
		rightViaCount = len(viaBranchList)
		for i := 0; i < len(viaBranchList); i++ {
			if viaBranchList[i] == right.ViaBranch {
				break
			}
			rightViaCount--
		}
	}

	if left.Method == right.Method {
		if strings.HasSuffix(right.CSeq, right.Method) || left.Method == "100" {
			return leftViaCount <= rightViaCount
		}
		return leftViaCount >= rightViaCount
	}

	if left.Method == "INVITE" && right.Method == "100" {
		return leftViaCount <= rightViaCount
	}

	if left.Method == "100" && right.Method == "INVITE" {
		return leftViaCount < rightViaCount
	}

	//hop by hop
	if left.CSeq == right.CSeq && leftViaCount == rightViaCount {
		if strings.HasSuffix(left.CSeq, left.Method) {
			return true
		}

		if strings.HasSuffix(right.CSeq, right.Method) {
			return false
		}
	}

	return true
}

func uniqueHepTable(hepSlice []model.HepTable) []model.HepTable {
	keys := make(map[string]bool)
	keys2 := make(map[string]string)
	var list []model.HepTable
	for _, entry := range hepSlice {
		dataKey := strconv.FormatInt(entry.Id, 10) + ":" + entry.CreatedDate.String()
		if keys[dataKey] {
			continue
		}
		keys[dataKey] = true

		var protocolHeader model.ProtocolHeader
		json.Unmarshal(entry.ProtocolHeader, &protocolHeader)
		captureId, exists := keys2[entry.Raw]
		if exists && captureId != protocolHeader.CaptureID {
			continue
		}

		keys2[entry.Raw] = protocolHeader.CaptureID
		list = append(list, entry)
	}

	return list
}

// this method create new user in the database
// it doesn't check internally whether all the validation are applied or not
func (ss *SearchService) GetTransactionData(table string, fieldKey string, dataWhere []interface{}, timeFrom,
	timeTo time.Time, nodes []string, userGroup string, likeSearch bool) ([]model.HepTable, error) {

	searchData := []model.HepTable{}
	query := "create_date between ? AND ? "
	//transactionData := model.TransactionResponse{}
	if likeSearch {
		query += "AND " + fieldKey + " LIKE ANY(ARRAY[?])"
	} else {
		query += "AND " + fieldKey + " in (?)"
	}

	//query := fieldKey + " in (?) and create_date between ? and ?"

	if config.Setting.IsolateGroup != "" && config.Setting.IsolateGroup == userGroup {
		query = query + " AND " + config.Setting.IsolateQuery
	}

	for session := range ss.Session {
		/* if node doesnt exists - continue */
		if !heputils.ElementExists(nodes, session) {
			continue
		}

		searchTmp := []model.HepTable{}
		if err := ss.Session[session].Debug().
			Table(table).
			Where(query, timeFrom.Format(time.RFC3339), timeTo.Format(time.RFC3339), dataWhere).
			Find(&searchTmp).Error; err != nil {
			logrus.Errorln("GetTransactionData: We have got error: ", err)
		}

		if len(searchTmp) > 0 {

			profileName := strings.TrimPrefix(table, "hep_proto_")

			for val := range searchTmp {
				searchTmp[val].Node = session
				searchTmp[val].DBNode = session
				searchTmp[val].Profile = profileName
			}
			searchData = append(searchData, searchTmp...)
		}
	}

	//response, _ := json.Marshal(searchData)
	return searchData, nil
}

func (ss *SearchService) getTransactionSummary(data *gabs.Container, aliasData map[string]string) string {

	var position = 0
	sid := gabs.New()
	host := gabs.New()
	alias := gabs.New()
	dataKeys := gabs.Wrap([]interface{}{})

	callData := []model.CallElement{}
	dataReply := gabs.Wrap([]interface{}{})

	for _, value := range data.Children() {
		dataElement := gabs.New()

		for k, v := range value.ChildrenMap() {
			switch k {
			case "data_header", "protocol_header":
				dataElement.Merge(v)
			case "sid", "correlation_id":
				sidData := gabs.New()
				sidData.Set(v.Data().(interface{}), k)
				dataElement.Merge(sidData)
				sid.Set(v.Data().(interface{}), k)
			default:
				newData := gabs.New()
				newData.Set(v.Data().(interface{}), k)
				dataElement.Merge(newData)
			}
		}

		callElement := model.CallElement{
			ID:          0,
			Sid:         "12345",
			DstHost:     "127.0.0.1",
			SrcHost:     "127.0.0.1",
			DstID:       "127.0.0.1:5060",
			SrcID:       "127.0.0.1:5060",
			SrcIP:       "127.0.0.1",
			DstIP:       "127.0.0.2",
			SrcPort:     0,
			DstPort:     0,
			Method:      "Generic",
			MethodText:  "generic",
			CreateDate:  0,
			Protocol:    1,
			MsgColor:    "blue",
			RuriUser:    "",
			Destination: 0,
		}
		if dataElement.Exists("payloadType") {
			callElement.Method, callElement.MethodText = heputils.ConvertPayloadTypeToString(heputils.CheckFloatValue(dataElement.S("payloadType").Data()))
		}

		if !dataElement.Exists("srcIp") {
			dataElement.Set("127.0.0.1", "srcIp")
			dataElement.Set(0, "srcPort")
		}

		if !dataElement.Exists("dstIp") {
			dataElement.Set("127.0.0.2", "dstIp")
			dataElement.Set(0, "dstPort")
		}

		if dataElement.Exists("id") {
			callElement.ID = int64(dataElement.S("id").Data().(float64))
		}
		if dataElement.Exists("srcIp") {
			callElement.SrcIP = dataElement.S("srcIp").Data().(string)
			callElement.SrcHost = dataElement.S("srcIp").Data().(string)
		}

		if dataElement.Exists("dstIp") {
			callElement.DstIP = dataElement.S("dstIp").Data().(string)
			callElement.DstHost = dataElement.S("dstIp").Data().(string)
		}
		if dataElement.Exists("srcPort") {
			callElement.SrcPort = int(heputils.CheckFloatValue(dataElement.S("srcPort").Data()))
		}
		if dataElement.Exists("dstPort") {
			callElement.DstPort = int(heputils.CheckFloatValue(dataElement.S("dstPort").Data()))
		}

		if dataElement.Exists("method") {
			callElement.Method = dataElement.S("method").Data().(string)
			callElement.MethodText = dataElement.S("method").Data().(string)
		}
		if dataElement.Exists("msg_name") {
			callElement.Method = dataElement.S("msg_name").Data().(string)
			callElement.MethodText = dataElement.S("msg_name").Data().(string)
		}
		if dataElement.Exists("event") {
			callElement.Method = dataElement.S("event").Data().(string)
			callElement.MethodText = dataElement.S("event").Data().(string)
		}
		if dataElement.Exists("create_date") {
			date, _ := dataElement.S("create_date").Data().(string)
			t, _ := time.Parse(time.RFC3339, date)
			callElement.CreateDate = t.Unix()
			callElement.MicroTs = t.Unix()
		}

		if dataElement.Exists("timeSeconds") && dataElement.Exists("timeUseconds") {
			ts := int64(heputils.CheckFloatValue(dataElement.S("timeSeconds").Data())*1000000 + heputils.CheckFloatValue(dataElement.S("timeUseconds").Data()))
			callElement.CreateDate = ts / 1000
			callElement.MicroTs = callElement.CreateDate
			dataElement.Set(callElement.MicroTs, "create_date")
		}

		if dataElement.Exists("protocol") {
			callElement.Protocol = int(heputils.CheckFloatValue(dataElement.S("protocol").Data()))
		}
		if dataElement.Exists("sid") {
			callElement.Sid = dataElement.S("sid").Data().(string)
		}
		if dataElement.Exists("raw") {
			callElement.RuriUser = dataElement.S("raw").Data().(string)
			if len(callElement.RuriUser) > 50 {
				callElement.RuriUser = callElement.RuriUser[:50]
			}
		}

		callElement.SrcID = callElement.SrcHost + ":" + strconv.Itoa(callElement.SrcPort)
		callElement.DstID = callElement.DstHost + ":" + strconv.Itoa(callElement.DstPort)

		srcIPPort := callElement.SrcIP + ":" + strconv.Itoa(callElement.SrcPort)
		dstIPPort := callElement.DstIP + ":" + strconv.Itoa(callElement.DstPort)

		testInput := net.ParseIP(callElement.SrcHost)
		if testInput.To4() == nil && testInput.To16() != nil {
			srcIPPort = "[" + callElement.SrcIP + "]:" + strconv.Itoa(callElement.SrcPort)
			callElement.SrcID = "[" + callElement.SrcHost + "]:" + strconv.Itoa(callElement.SrcPort)

		}

		testInput = net.ParseIP(callElement.DstIP)
		if testInput.To4() == nil && testInput.To16() != nil {
			dstIPPort = "[" + callElement.DstIP + "]:" + strconv.Itoa(callElement.DstPort)
			callElement.DstID = "[" + callElement.DstHost + "]:" + strconv.Itoa(callElement.DstPort)
		}

		if value, ok := aliasData[callElement.SrcIP]; ok {
			alias.Set(value, srcIPPort)
		} else {
			alias.Set(srcIPPort, srcIPPort)
		}

		if value, ok := aliasData[callElement.DstIP]; ok {
			alias.Set(value, dstIPPort)
		} else {
			alias.Set(dstIPPort, dstIPPort)
		}

		if !host.Exists(callElement.SrcID) {
			jsonObj := gabs.New()
			jsonObj.Array(callElement.SrcID, "host")
			jsonObj.ArrayAppend(callElement.SrcID, callElement.SrcID, "host")
			jsonObj.S(callElement.SrcID).Set(position, "position")
			host.Merge(jsonObj)
			position++
		}

		if !host.Exists(callElement.DstID) {
			jsonObj := gabs.New()
			jsonObj.Array(callElement.DstID, "host")
			jsonObj.ArrayAppend(callElement.DstID, callElement.DstID, "host")
			jsonObj.S(callElement.DstID).Set(position, "position")
			host.Merge(jsonObj)
			position++
		}

		callElement.Destination = host.Search(callElement.DstID, "position").Data().(int)
		callData = append(callData, callElement)
		for key := range dataElement.ChildrenMap() {
			if !function.ArrayKeyExits(key, dataKeys) {
				dataKeys.ArrayAppend(key)
			}
		}
		dataReply.ArrayAppend(dataElement.Data())

	}

	total, _ := dataReply.ArrayCount()
	reply := gabs.New()
	reply.Set(total, "total")
	reply.Set(dataReply.Data(), "data", "messages")
	reply.Set(host.Data(), "data", "hosts")
	reply.Set(callData, "data", "calldata")
	reply.Set(alias.Data(), "data", "alias")
	reply.Set(dataKeys.Data(), "keys")
	return reply.String()
}

func (ss *SearchService) getTransactionSummaryV2(transactionTables []*model.TransactionTable, aliasData map[string]string) string {
	alias := gabs.New()
	dataReply := gabs.Wrap([]interface{}{})
	var transactionElements []model.TransactionElement
	var hosts []string
	for _, entryT := range transactionTables {
		transactionElement := model.TransactionElement{
			ViaBranch: entryT.ViaBranch,
			Name:      entryT.Name,
			ErrorName: entryT.ErrorName,
			BeginDate: entryT.BeginDate.UnixNano() / 1000000,
			FromUser:  entryT.FromUser,
			ToUser:    entryT.ToUser,
		}
		for _, entryM := range entryT.TMethods {
			for elem := entryM.BodyList.Front(); elem != nil; elem = elem.Next() {
				hepTable := elem.Value.(model.HepTable)
				var protocolHeader model.ProtocolHeader
				json.Unmarshal(hepTable.ProtocolHeader, &protocolHeader)
				var dataHeader model.DataHeader
				json.Unmarshal(hepTable.DataHeader, &dataHeader)
				callElement := model.CallElement{
					MsgColor:    "blue",
					Destination: 0,
				}
				callElement.ID = hepTable.Id
				callElement.Sid = hepTable.Sid
				callElement.SrcHost = protocolHeader.SrcIP
				callElement.SrcIP = protocolHeader.SrcIP
				callElement.SrcPort = protocolHeader.SrcPort
				callElement.SrcID = protocolHeader.SrcIP + ":" + strconv.Itoa(protocolHeader.SrcPort)
				callElement.DstIP = protocolHeader.DstIP
				callElement.DstHost = protocolHeader.DstIP
				callElement.DstPort = protocolHeader.DstPort
				callElement.DstID = protocolHeader.DstIP + ":" + strconv.Itoa(protocolHeader.DstPort)
				callElement.Method = dataHeader.Method
				callElement.MethodText = dataHeader.Method
				callElement.CreateDate = hepTable.CreatedDate.UnixNano() / 1000000
				callElement.MicroTs = hepTable.CreatedDate.UnixNano() / 1000000
				callElement.Protocol = protocolHeader.Protocol
				callElement.RuriUser = dataHeader.RuriUser

				if value, ok := config.Setting.SbcIdMap[callElement.SrcID]; ok {
					newSrcIds := strings.Split(value, ":")
					newSrcIP := newSrcIds[0]
					valueAlias, _ := aliasData[newSrcIP]
					valueAlias = valueAlias + "[" + callElement.SrcID + "]"
					callElement.SrcID = value
					callElement.SrcHost = newSrcIP
					callElement.SrcIP = newSrcIP
					//callElement.SrcPort, _ = strconv.Atoi(newSrcIds[1])
					alias.Set(valueAlias, callElement.SrcID)
				} else {
					if !alias.Exists(callElement.SrcID) {
						if valueAlias, okAlias := aliasData[callElement.SrcIP]; okAlias {
							alias.Set(valueAlias, callElement.SrcID)
						}
					}
				}

				if value, ok := config.Setting.SbcIdMap[callElement.DstID]; ok {
					newDstIds := strings.Split(value, ":")
					newDstIP := newDstIds[0]
					valueAlias, _ := aliasData[newDstIP]
					valueAlias = valueAlias + "[" + callElement.DstID + "]"

					callElement.DstID = value
					callElement.DstHost = newDstIP
					callElement.DstIP = newDstIP
					//callElement.DstPort, _ = strconv.Atoi(newDstIds[1])

					alias.Set(valueAlias, callElement.DstID)
				} else {
					if !alias.Exists(callElement.DstID) {
						if valueAlias, okAlias := aliasData[callElement.DstIP]; okAlias {
							alias.Set(valueAlias, callElement.DstID)
						}
					}
				}

				transactionElement.CallData = append(transactionElement.CallData, callElement)
				dataElement := gabs.New()
				marshalData, _ := json.Marshal(hepTable)
				jsonParsed, _ := gabs.ParseJSON(marshalData)
				for k, v := range jsonParsed.ChildrenMap() {
					switch k {
					case "data_header", "protocol_header":
						dataElement.Merge(v)
					case "sid", "correlation_id":
						sidData := gabs.New()
						sidData.Set(v.Data().(interface{}), k)
						dataElement.Merge(sidData)
					default:
						newData := gabs.New()
						newData.Set(v.Data().(interface{}), k)
						dataElement.Merge(newData)
					}
				}
				dataReply.ArrayAppend(dataElement.Data())

				transactionElement.Host = heputils.AppendIfNotExists(transactionElement.Host, callElement.SrcID)
				transactionElement.Host = heputils.AppendIfNotExists(transactionElement.Host, callElement.DstID)

			}
		}
		transactionElements = append(transactionElements, transactionElement)
		for indexI := 0; indexI < len(transactionElement.Host); indexI++ {
			indexJ := 0
			for ; indexJ < len(hosts); indexJ++ {
				if hosts[indexJ] == transactionElement.Host[indexI] {
					break
				}
			}

			if indexJ >= len(hosts) {
				hosts = append(hosts, transactionElement.Host[indexI])
			}
		}
	}

	reply := gabs.New()
	reply.Set(dataReply.Data(), "data", "messages")
	reply.Set(hosts, "data", "hosts")
	reply.Set(transactionElements, "data", "transaction_elements")
	reply.Set(alias.Data(), "data", "alias")
	return reply.String()
}

func (ss *SearchService) GetEmptyQos() string {
	reply := gabs.New()
	for i := 0; i < 2; i++ {
		dataReply := gabs.Wrap([]interface{}{})
		dataQos := gabs.New()
		totalEl, _ := dataReply.ArrayCount()
		dataQos.Set(totalEl, "total")
		dataQos.Set(dataReply.Data(), "data")
		reply.Set(dataQos.Data(), xconditions.IfThenElse(i == 0, "rtcp", "rtp").(string))
	}
	return reply.String()
}

// this method create new user in the database
// it doesn't check internally whether all the validation are applied or not
func (ss *SearchService) GetTransactionQos(tables [2]string, data []byte, nodes []string, aliasData map[string]string) (string, error) {

	var dataWhere []interface{}
	sid := gabs.New()
	reply := gabs.New()
	requestData, _ := gabs.ParseJSON(data)
	for _, value := range requestData.Search("param", "search").ChildrenMap() {
		//table = "hep_proto_" + key
		for _, v := range value.Search("callid").Data().([]interface{}) {
			dataWhere = append(dataWhere, v)
		}
	}
	timeWhereFrom := requestData.S("timestamp", "from").Data().(float64)
	timeWhereTo := requestData.S("timestamp", "to").Data().(float64)
	timeFrom := time.Unix(int64(timeWhereFrom/float64(time.Microsecond)), 0).UTC()
	timeTo := time.Unix(int64(timeWhereTo/float64(time.Microsecond)), 0).UTC()

	for i, table := range tables {
		searchData := []model.HepTable{}
		dataReply := gabs.Wrap([]interface{}{})

		query := "sid in (?) and create_date between ? and ?"

		for session := range ss.Session {
			/* if node doesnt exists - continue */
			if !heputils.ElementExists(nodes, session) {
				continue
			}

			searchTmp := []model.HepTable{}
			if err := ss.Session[session].Debug().
				Table(table).
				Where(query, dataWhere, timeFrom.Format(time.RFC3339), timeTo.Format(time.RFC3339)).
				Find(&searchTmp).Error; err != nil {
				logrus.Errorln("GetTransactionQos: We have got error: ", err)
				return "", err

			}

			if len(searchTmp) > 0 {
				for val := range searchTmp {
					searchTmp[val].Node = session
					searchTmp[val].DBNode = session
				}
				searchData = append(searchData, searchTmp...)
			}
		}

		for _, entry := range searchData {
			var protocolHeader model.ProtocolHeader
			json.Unmarshal(entry.ProtocolHeader, &protocolHeader)

			captureStrings := strings.Split(protocolHeader.CaptureID, "_")
			//REG-FAT[100026750]_10.2.38.203
			if len(captureStrings) == 2 {
				if _, ok := aliasData[captureStrings[1]]; !ok {
					aliasData[captureStrings[1]] = captureStrings[0]
				}
			}
		}

		for index, entry := range searchData {
			var protocolHeader model.ProtocolHeader
			json.Unmarshal(entry.ProtocolHeader, &protocolHeader)
			needMarshal := false
			if value, ok := aliasData[protocolHeader.SrcIP]; ok {
				protocolHeader.SrcIP = value + "[" + protocolHeader.SrcIP + "]"
				needMarshal = true
			}
			if value, ok := aliasData[protocolHeader.DstIP]; ok {
				protocolHeader.DstIP = value + "[" + protocolHeader.DstIP + "]"
				needMarshal = true
			}
			if needMarshal {
				marshalProtocolHeader, _ := json.Marshal(protocolHeader)
				searchData[index].ProtocolHeader = marshalProtocolHeader
			}
		}

		/* lets sort it */
		sort.Slice(searchData, func(i, j int) bool {
			return searchData[i].CreatedDate.Before(searchData[j].CreatedDate)
		})

		response, _ := json.Marshal(searchData)
		row, _ := gabs.ParseJSON(response)
		for _, value := range row.Children() {
			dataElement := gabs.New()
			for k, v := range value.ChildrenMap() {
				switch k {
				case "data_header", "protocol_header":
					dataElement.Merge(v)
				case "sid", "correlation_id":
					sidData := gabs.New()
					sidData.Set(v.Data().(interface{}), k)
					dataElement.Merge(sidData)
					sid.Set(v.Data().(interface{}), k)
				default:
					newData := gabs.New()
					newData.Set(v.Data().(interface{}), k)
					dataElement.Merge(newData)
				}
			}
			dataReply.ArrayAppend(dataElement.Data())
		}

		dataQos := gabs.New()
		totalEl, _ := dataReply.ArrayCount()
		dataQos.Set(totalEl, "total")
		dataQos.Set(dataReply.Data(), "data")
		reply.Set(dataQos.Data(), xconditions.IfThenElse(i == 0, "rtcp", "rtp").(string))

	}

	return reply.String(), nil
}

// this method create new user in the database
// it doesn't check internally whether all the validation are applied or not
func (ss *SearchService) GetTransactionLog(table string, data []byte, nodes []string) (string, error) {

	var dataWhere []interface{}
	sid := gabs.New()
	searchData := []model.HepTable{}
	dataReply := gabs.Wrap([]interface{}{})
	requestData, _ := gabs.ParseJSON(data)
	for _, value := range requestData.Search("param", "search").ChildrenMap() {
		//table = "hep_proto_" + key
		for _, v := range value.Search("callid").Data().([]interface{}) {
			dataWhere = append(dataWhere, v)
		}
	}
	timeWhereFrom := requestData.S("timestamp", "from").Data().(float64)
	timeWhereTo := requestData.S("timestamp", "to").Data().(float64)
	timeFrom := time.Unix(int64(timeWhereFrom/float64(time.Microsecond)), 0).UTC()
	timeTo := time.Unix(int64(timeWhereTo/float64(time.Microsecond)), 0).UTC()

	query := "sid in (?) and create_date between ? and ?"
	for session := range ss.Session {
		/* if node doesnt exists - continue */
		if !heputils.ElementExists(nodes, session) {
			continue
		}
		searchTmp := []model.HepTable{}
		if err := ss.Session[session].Debug().
			Table(table).
			Where(query, dataWhere, timeFrom.Format(time.RFC3339), timeTo.Format(time.RFC3339)).
			Find(&searchTmp).Error; err != nil {
			logrus.Errorln("GetTransactionLog: We have got error: ", err)
			return "", err

		}

		if len(searchTmp) > 0 {
			for val := range searchTmp {
				searchTmp[val].Node = session
				searchTmp[val].DBNode = session
			}
			searchData = append(searchData, searchTmp...)
		}
	}

	response, _ := json.Marshal(searchData)
	row, _ := gabs.ParseJSON(response)
	for _, value := range row.Children() {
		dataElement := gabs.New()
		for k, v := range value.ChildrenMap() {
			switch k {
			case "data_header", "protocol_header":
				dataElement.Merge(v)
			case "sid", "correlation_id":
				sidData := gabs.New()
				sidData.Set(v.Data().(interface{}), k)
				dataElement.Merge(sidData)
				sid.Set(v.Data().(interface{}), k)
			default:
				newData := gabs.New()
				newData.Set(v.Data().(interface{}), k)
				dataElement.Merge(newData)
			}
		}
		dataReply.ArrayAppend(dataElement.Data())
	}

	total, _ := dataReply.ArrayCount()
	reply := gabs.New()
	reply.Set(total, "total")
	reply.Set(dataReply.Data(), "data")
	return reply.String(), nil
}
