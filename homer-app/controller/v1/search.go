package controllerv1

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sipcapture/homer-app/auth"
	"github.com/sipcapture/homer-app/data/service"
	"github.com/sipcapture/homer-app/model"
	httpresponse "github.com/sipcapture/homer-app/network/response"
	"github.com/sipcapture/homer-app/system/webmessages"
	"github.com/sirupsen/logrus"
)

type SearchController struct {
	Controller
	SearchService  *service.SearchService
	SettingService *service.UserSettingsService
	AliasService   *service.AliasService
}

// swagger:operation POST /search/call/data search searchSearchData
//
// Returns data based upon filtered json
// ---
// consumes:
// - application/json
// produces:
// - application/json
// parameters:
// - name: SearchObject
//   in: body
//   type: object
//   description: SearchObject parameters
//   schema:
//     "$ref": "#/definitions/SearchCallData"
//   required: true
// SecurityDefinitions:
// bearer:
//      type: apiKey
//      name: Authorization
//      in: header
// responses:
//   '200': body:ListUsers
//   '400': body:UserLoginFailureResponse
func (sc *SearchController) SearchData(c echo.Context) error {

	searchObject := model.SearchObject{}

	if err := c.Bind(&searchObject); err != nil {
		logrus.Error(err.Error())
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, webmessages.UserRequestFormatIncorrect)
	}

	aliasRowData, _ := sc.AliasService.GetAllActive()
	aliasData := make(map[string]string)
	for _, row := range aliasRowData {
		cidr := row.IP + "/" + strconv.Itoa(*row.Mask)
		//Port := strconv.Itoa(*row.Port)
		ip, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			return err
		}

		for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
			aliasData[ip.String()] = row.Alias
		}
	}
	userGroup := auth.GetUserGroup(c)

	responseData, err := sc.SearchService.SearchData(&searchObject, aliasData, userGroup)
	if err != nil {
		logrus.Println(responseData)
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}
	return httpresponse.CreateSuccessResponse(&c, http.StatusCreated, responseData)
}

func (sc *SearchController) SearchSbcOption() []model.SbcOptionData {
	return sc.SearchService.SearchSbcOption()
}

func (sc *SearchController) SearchSbcInvite100() []model.SbcInvite100Data {
	return sc.SearchService.SearchSbcInvite100()
}

func (sc *SearchController) SearchRegInvite100() []model.RegInvite100Data {
	return sc.SearchService.SearchRegInvite100()
}

func (sc *SearchController) SearchCmNotify() []model.CMNotifyData {
	return sc.SearchService.SearchCmNotify()
}

func (sc *SearchController) SearchSbcNotify() []model.SbcNotifyData {
	return sc.SearchService.SearchSbcNotify()
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// swagger:operation POST /search/call/message search searchGetMessageById
//
// Returns message data based upon filtered json
// ---
// consumes:
// - application/json
// produces:
// - application/json
// parameters:
// - name: SearchObject
//   in: body
//   type: object
//   description: SearchObject parameters
//   schema:
//     "$ref": "#/definitions/SearchCallData"
//   required: true
// SecurityDefinitions:
// bearer:
//      type: apiKey
//      name: Authorization
//      in: header
// responses:
//   '200': body:ListUsers
//   '400': body:UserLoginFailureResponse
func (sc *SearchController) GetMessageById(c echo.Context) error {

	searchObject := model.SearchObject{}

	if err := c.Bind(&searchObject); err != nil {
		logrus.Error(err.Error())
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, webmessages.UserRequestFormatIncorrect)
	}

	responseData, err := sc.SearchService.GetMessageByID(&searchObject)
	if err != nil {
		logrus.Println(responseData)
	}
	return httpresponse.CreateSuccessResponse(&c, http.StatusCreated, responseData)
}

// swagger:operation POST /search/call/decode/message search searchGetDecodeMessageById
//
// Returns data based upon filtered json
// ---
// consumes:
// - application/json
// produces:
// - application/json
// parameters:
// - name: SearchObject
//   in: body
//   type: object
//   description: SearchObject parameters
//   schema:
//     "$ref": "#/definitions/SearchCallData"
//   required: true
// SecurityDefinitions:
// bearer:
//      type: apiKey
//      name: Authorization
//      in: header
// responses:
//   '200': body:ListUsers
//   '400': body:UserLoginFailureResponse
func (sc *SearchController) GetDecodeMessageById(c echo.Context) error {

	searchObject := model.SearchObject{}

	if err := c.Bind(&searchObject); err != nil {
		logrus.Error(err.Error())
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, webmessages.UserRequestFormatIncorrect)
	}

	responseData, err := sc.SearchService.GetDecodedMessageByID(&searchObject)
	if err != nil {
		logrus.Println(responseData)
	}
	return httpresponse.CreateSuccessResponse(&c, http.StatusCreated, responseData)
}

// swagger:operation POST /call/transaction search searchGetTransaction
//
// Returns transaction data based upon filtered json
// ---
// consumes:
// - application/json
// produces:
// - application/json
// parameters:
// - name: SearchObject
//   in: body
//   type: object
//   description: SearchObject parameters
//   schema:
//     "$ref": "#/definitions/SearchCallData"
//   required: true
// SecurityDefinitions:
// bearer:
//      type: apiKey
//      name: Authorization
//      in: header
// responses:
//   '200': body:ListUsers
//   '400': body:UserLoginFailureResponse
func (sc *SearchController) GetTransaction(c echo.Context) error {

	transactionObject := model.SearchObject{}
	if err := c.Bind(&transactionObject); err != nil {
		logrus.Error(err.Error())
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, webmessages.UserRequestFormatIncorrect)
	}

	transactionData, _ := json.Marshal(transactionObject)
	correlation, _ := sc.SettingService.GetCorrelationMap(&transactionObject)
	aliasRowData, _ := sc.AliasService.GetAllActive()

	aliasData := make(map[string]string)
	for _, row := range aliasRowData {
		cidr := row.IP + "/" + strconv.Itoa(*row.Mask)
		ip, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			return err
		}

		for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
			aliasData[ip.String()] = row.Alias
		}
	}

	searchTable := "hep_proto_1_default'"

	userGroup := auth.GetUserGroup(c)

	reply, _ := sc.SearchService.GetTransaction(searchTable, transactionData,
		correlation, false, aliasData, 0, transactionObject.Param.Location.Node,
		sc.SettingService, userGroup)

	return httpresponse.CreateSuccessResponse(&c, http.StatusCreated, reply)

}

// swagger:operation POST /call/transaction search searchGetTransaction
//
// Returns transaction data based upon filtered json
// ---
// consumes:
// - application/json
// produces:
// - application/json
// parameters:
// - name: SearchObject
//   in: body
//   type: object
//   description: SearchObject parameters
//   schema:
//     "$ref": "#/definitions/SearchCallData"
//   required: true
// SecurityDefinitions:
// bearer:
//      type: apiKey
//      name: Authorization
//      in: header
// responses:
//   '200': body:ListUsers
//   '400': body:UserLoginFailureResponse
func (sc *SearchController) GetTransactionV2(c echo.Context) error {

	transactionObject := model.SearchObject{}
	if err := c.Bind(&transactionObject); err != nil {
		logrus.Error(err.Error())
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, webmessages.UserRequestFormatIncorrect)
	}

	transactionData, _ := json.Marshal(transactionObject)
	correlation, _ := sc.SettingService.GetCorrelationMap(&transactionObject)
	aliasRowData, _ := sc.AliasService.GetAllActive()

	aliasData := make(map[string]string)
	for _, row := range aliasRowData {
		cidr := row.IP + "/" + strconv.Itoa(*row.Mask)
		ip, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			return err
		}

		for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
			aliasData[ip.String()] = row.Alias
		}
	}

	userGroup := auth.GetUserGroup(c)

	reply, _ := sc.SearchService.GetTransactionV2(transactionData,
		correlation, aliasData, transactionObject.Param.Location.Node,
		sc.SettingService, userGroup)

	return httpresponse.CreateSuccessResponse(&c, http.StatusCreated, reply)

}

// swagger:operation POST /call/report/qos search searchGetTransactionQos
//
// Returns qos data based upon filtered json
// ---
// consumes:
// - application/json
// produces:
// - application/json
// parameters:
// - name: SearchObject
//   in: body
//   type: object
//   description: SearchObject parameters
//   schema:
//     "$ref": "#/definitions/SearchCallData"
//   required: true
// SecurityDefinitions:
// bearer:
//      type: apiKey
//      name: Authorization
//      in: header
// responses:
//   '200': body:ListUsers
//   '400': body:UserLoginFailureResponse
func (sc *SearchController) GetTransactionQos(c echo.Context) error {

	searchObject := model.SearchObject{}
	if err := c.Bind(&searchObject); err != nil {
		logrus.Error(err.Error())
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, webmessages.UserRequestFormatIncorrect)
	}

	aliasRowData, _ := sc.AliasService.GetAllActive()

	aliasData := make(map[string]string)
	for _, row := range aliasRowData {
		cidr := row.IP + "/" + strconv.Itoa(*row.Mask)
		ip, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			return err
		}

		for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
			aliasData[ip.String()] = row.Alias
		}
	}

	transactionData, _ := json.Marshal(searchObject)
	searchTable := [...]string{"hep_proto_5_default", "hep_proto_35_default"}
	row, _ := sc.SearchService.GetTransactionQos(searchTable, transactionData, searchObject.Param.Location.Node, aliasData)

	//row := sc.SearchService.GetEmptyQos()
	return httpresponse.CreateSuccessResponse(&c, http.StatusCreated, row)

}

// swagger:operation POST /call/report/log search searchGetTransactionLog
//
// Returns log data based upon filtered json
// ---
// consumes:
// - application/json
// produces:
// - application/json
// parameters:
// - name: SearchObject
//   in: body
//   type: object
//   description: SearchObject parameters
//   schema:
//     "$ref": "#/definitions/SearchCallData"
//   required: true
// SecurityDefinitions:
// bearer:
//      type: apiKey
//      name: Authorization
//      in: header
// responses:
//   '200': body:ListUsers
//   '400': body:UserLoginFailureResponse
func (sc *SearchController) GetTransactionLog(c echo.Context) error {

	searchObject := model.SearchObject{}
	if err := c.Bind(&searchObject); err != nil {
		logrus.Error(err.Error())
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, webmessages.UserRequestFormatIncorrect)
	}
	transactionData, _ := json.Marshal(searchObject)
	searchTable := "hep_proto_100_default"
	row, _ := sc.SearchService.GetTransactionLog(searchTable, transactionData, searchObject.Param.Location.Node)

	return httpresponse.CreateSuccessResponse(&c, http.StatusCreated, row)
}

func (sc *SearchController) GetTransactionHepSub(c echo.Context) error {

	searchObject := model.SearchObject{}
	if err := c.Bind(&searchObject); err != nil {
		logrus.Error(err.Error())
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, webmessages.UserRequestFormatIncorrect)
	}
	transactionData, _ := json.Marshal(searchObject)

	searchTable := "hep_proto_100_default"
	row, _ := sc.SearchService.GetTransactionLog(searchTable, transactionData, searchObject.Param.Location.Node)

	return httpresponse.CreateSuccessResponse(&c, http.StatusCreated, row)
}

// swagger:operation POST /export/call/messages/pcap search searchGetMessagesAsPCap
//
// Returns pcap data based upon filtered json
// ---
// consumes:
// - application/json
// produces:
// - application/json
// parameters:
// - name: SearchObject
//   in: body
//   type: object
//   description: SearchObject parameters
//   schema:
//     "$ref": "#/definitions/SearchCallData"
//   required: true
// SecurityDefinitions:
// bearer:
//      type: apiKey
//      name: Authorization
//      in: header
// responses:
//   '200': body:ListUsers
//   '400': body:UserLoginFailureResponse
func (sc *SearchController) GetMessagesAsPCap(c echo.Context) error {

	searchObject := model.SearchObject{}
	if err := c.Bind(&searchObject); err != nil {
		logrus.Error(err.Error())
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, webmessages.UserRequestFormatIncorrect)
	}

	transactionData, _ := json.Marshal(searchObject)
	correlation, _ := sc.SettingService.GetCorrelationMap(&searchObject)
	aliasRowData, _ := sc.AliasService.GetAllActive()

	aliasData := make(map[string]string)
	for _, row := range aliasRowData {
		aliasData[row.IP] = row.Alias
	}

	searchTable := "hep_proto_1_default'"
	userGroup := auth.GetUserGroup(c)

	reply, _ := sc.SearchService.GetTransaction(searchTable, transactionData, correlation, false, aliasData, 1,
		searchObject.Param.Location.Node, sc.SettingService, userGroup)

	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=export-%s.pcap", time.Now().Format(time.RFC3339)))
	if err := c.Blob(http.StatusOK, "application/octet-stream", []byte(reply)); err != nil {
		logrus.Error(err.Error())
	}

	c.Response().Flush()
	return nil

}

// swagger:operation POST /export/call/messages/text search searchGetMessagesAsText
//
// Returns text data based upon filtered json
// ---
// consumes:
// - application/json
// produces:
// - application/json
// parameters:
// - name: SearchObject
//   in: body
//   type: object
//   description: SearchObject parameters
//   schema:
//     "$ref": "#/definitions/SearchCallData"
//   required: true
// SecurityDefinitions:
// bearer:
//      type: apiKey
//      name: Authorization
//      in: header
// responses:
//   '200': body:ListUsers
//   '400': body:UserLoginFailureResponse
func (sc *SearchController) GetMessagesAsText(c echo.Context) error {

	searchObject := model.SearchObject{}
	if err := c.Bind(&searchObject); err != nil {
		logrus.Error(err.Error())
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, webmessages.UserRequestFormatIncorrect)
	}

	transactionData, _ := json.Marshal(searchObject)
	correlation, _ := sc.SettingService.GetCorrelationMap(&searchObject)
	aliasData := make(map[string]string)

	searchTable := "hep_proto_1_default'"

	userGroup := auth.GetUserGroup(c)

	reply, _ := sc.SearchService.GetTransaction(searchTable, transactionData,
		correlation, false, aliasData, 2, searchObject.Param.Location.Node,
		sc.SettingService, userGroup)

	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=export-%s.txt", time.Now().Format(time.RFC3339)))
	if err := c.String(http.StatusOK, reply); err != nil {
		logrus.Error(err.Error())
	}

	c.Response().Flush()

	return nil

	//return httpresponse.CreateSuccessResponse(&c, http.StatusCreated, reply)

}
