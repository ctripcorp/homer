package controllerv1

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sipcapture/homer-app/auth"
	"github.com/sipcapture/homer-app/data/service"
	"github.com/sipcapture/homer-app/model"
	httpresponse "github.com/sipcapture/homer-app/network/response"
	"github.com/sipcapture/homer-app/system/webmessages"
	"github.com/sirupsen/logrus"
)

type AgentsubController struct {
	Controller
	AgentsubService *service.AgentsubService
}

// swagger:route GET /agent/subscribe agent agentsSubGetAgentsub
//
// Get all agents
// ---
// consumes:
// - application/json
// produces:
// - application/json
// Security:
// - bearer: []
//
// SecurityDefinitions:
// bearer:
//      type: apiKey
//      name: Authorization
//      in: header
// responses:
//   '200': body:AgentsLocation
//   '400': body:FailureResponse
func (ass *AgentsubController) GetAgentsub(c echo.Context) error {

	reply, err := ass.AgentsubService.GetAgentsub()
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, webmessages.GetAgentSubFailed)
	}
	return httpresponse.CreateSuccessResponseWithJson(&c, http.StatusOK, []byte(reply))

}

// swagger:operation GET /agent/type/{type} agent agentsSubGetAgentsubByType
//
// Get agent by type
// ---
// consumes:
// - application/json
// produces:
// - application/json
// parameters:
// - name: type
//   in: path
//   example: home
//   description: type of agent
//   required: true
//   type: string
// Security:
// - bearer: []
//
// SecurityDefinitions:
// bearer:
//      type: apiKey
//      name: Authorization
//      in: header
// responses:
//   '200': body:AgentsLocation
//   '400': body:FailureResponse
func (ass *AgentsubController) GetAgentsubByType(c echo.Context) error {
	typeRequest := url.QueryEscape(c.Param("type"))
	reply, err := ass.AgentsubService.GetAgentsubAgainstType(typeRequest)
	if err != nil {
		return httpresponse.CreateSuccessResponseWithJson(&c, http.StatusOK, []byte(reply))
	}
	return httpresponse.CreateSuccessResponseWithJson(&c, http.StatusOK, []byte(reply))
}

// swagger:operation GET /agent/subscribe/{guid} agent agentsSubGetAgentsubAgainstGUID
//
// Get agent by guid
// ---
// consumes:
// - application/json
// produces:
// - application/json
// parameters:
// - name: guid
//   in: path
//   example: eacdae5b-4203-40a2-b388-969312ffcffe
//   description: guid of agent
//   required: true
//   type: string
// Security:
// - bearer: []
//
// SecurityDefinitions:
// bearer:
//      type: apiKey
//      name: Authorization
//      in: header
// responses:
//   '201': body:AgentsLocation
//   '400': body:FailureResponse
func (ass *AgentsubController) GetAgentsubAgainstGUID(c echo.Context) error {
	guid := url.QueryEscape(c.Param("guid"))
	reply, err := ass.AgentsubService.GetAgentsubAgainstGUID(guid)
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}
	return httpresponse.CreateSuccessResponseWithJson(&c, http.StatusOK, []byte(reply))

}

// swagger:operation POST /agent/subscribe agent agentsSubAddAgentsubWithKey
//
// Add agent
// ---
// consumes:
// - application/json
// produces:
// - application/json
// parameters:
// - name: AgentsLocation Struct
//   in: body
//   description: agent parameters
//   schema:
//     "$ref": "#/definitions/AgentsLocation"
//   required: true
// Security:
// - bearer: []
//
// SecurityDefinitions:
// bearer:
//      type: apiKey
//      name: Authorization
//      in: header
// responses:
//   '201': body:UserCreateSuccessfulResponse
//   '400': body:UserCreateSuccessfulResponse
func (ass *AgentsubController) AddAgentsubWithKey(c echo.Context) error {
	// Stub an user to be populated from the body
	agentSub := model.TableAgentLocationSession{}
	err := c.Bind(&agentSub)
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}
	// validate input request body
	if err := c.Validate(agentSub); err != nil {
		logrus.Error(err.Error())
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}

	authToken := c.Request().Header.Get(auth.TokenHeader)
	if authToken == "" {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, "Auth-Token header not present or has empty value")
	}

	reply, err := ass.AgentsubService.GetAuthKeyByHeaderToken(authToken)
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}

	reply, err = ass.AgentsubService.DeleteAgentsubAgainstGUID(agentSub.GUID)
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}

	agentSub.CreateDate = time.Now()
	agentSub.ExpireDate = time.Now().Add(time.Duration(agentSub.TTL) * time.Second)
	agentSub.Active = 1

	reply, err = ass.AgentsubService.AddAgentsub(agentSub)
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}
	return httpresponse.CreateSuccessResponseWithJson(&c, http.StatusCreated, []byte(reply))
}

// swagger:operation PUT /agent/subscribe/{guid} agent agentsSubUpdateAgentsubAgainstGUID
//
// Update agent by guid
// ---
// consumes:
// - application/json
// produces:
// - application/json
// parameters:
// - name: guid
//   in: path
//   example: eacdae5b-4203-40a2-b388-969312ffcffe
//   description: guid of agent
//   required: true
//   type: string
// Security:
// - bearer: []
//
// SecurityDefinitions:
// bearer:
//      type: apiKey
//      name: Authorization
//      in: header
// responses:
//   '201': body:AgentsLocation
//   '400': body:FailureResponse
func (ass *AgentsubController) UpdateAgentsubAgainstGUID(c echo.Context) error {
	guid := url.QueryEscape(c.Param("guid"))
	reply, err := ass.AgentsubService.GetAgentsubAgainstGUID(guid)
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}
	// Stub an user to be populated from the body
	u := model.TableAgentLocationSession{}
	err = c.Bind(&u)
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}
	// validate input request body
	if err := c.Validate(u); err != nil {
		logrus.Error(err.Error())
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}
	u.GUID = guid
	reply, err = ass.AgentsubService.UpdateAgentsubAgainstGUID(guid, u)
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}
	return httpresponse.CreateSuccessResponseWithJson(&c, http.StatusOK, []byte(reply))
}

// swagger:operation DELETE /agent/subscribe/{guid} agent agentsSubDeleteAgentsubAgainstGUID
//
// Delete agent by guid
// ---
// consumes:
// - application/json
// produces:
// - application/json
// parameters:
// - name: guid
//   in: path
//   example: eacdae5b-4203-40a2-b388-969312ffcffe
//   description: guid of agent
//   required: true
//   type: string
// Security:
// - bearer: []
//
// SecurityDefinitions:
// bearer:
//      type: apiKey
//      name: Authorization
//      in: header
// responses:
//   '201': body:UserCreateSuccessfulResponse
//   '400': body:UserCreateSuccessfulResponse
func (ass *AgentsubController) DeleteAgentsubAgainstGUID(c echo.Context) error {
	guid := url.QueryEscape(c.Param("guid"))

	reply, err := ass.AgentsubService.GetAgentsubAgainstGUID(guid)
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}
	reply, err = ass.AgentsubService.DeleteAgentsubAgainstGUID(guid)
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}
	reply, err = ass.AgentsubService.DeleteAgentsubAgainstGUID(guid)
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}
	return httpresponse.CreateSuccessResponseWithJson(&c, http.StatusOK, []byte(reply))
}

// swagger:operation POST /agent/search/{guid}/{type} agent agentsSubGetAgentSearchByTypeAndGUID
//
// Get agent by guid and type
// ---
// consumes:
// - application/json
// produces:
// - application/json
// parameters:
// - name: guid
//   in: path
//   example: eacdae5b-4203-40a2-b388-969312ffcffe
//   description: guid of agent
//   required: true
//   type: string
// - name: type
//   in: path
//   example: home
//   description: type of agent
//   required: true
//   type: string
// Security:
// - bearer: []
//
// SecurityDefinitions:
// bearer:
//      type: apiKey
//      name: Authorization
//      in: header
// responses:
//   '201': body:UserCreateSuccessfulResponse
//   '400': body:UserCreateSuccessfulResponse
func (ass *AgentsubController) GetAgentSearchByTypeAndGUID(c echo.Context) error {
	guid := url.QueryEscape(c.Param("guid"))
	typeRequest := url.QueryEscape(c.Param("type"))

	transactionObject := model.SearchObject{}
	if err := c.Bind(&transactionObject); err != nil {
		logrus.Error(err.Error())
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, webmessages.UserRequestFormatIncorrect)
	}

	agentObject, err := ass.AgentsubService.GetAgentsubAgainstGUIDAndType(guid, typeRequest)

	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}

	reply, err := ass.AgentsubService.DoSearchByPost(agentObject, transactionObject, typeRequest)
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}

	if typeRequest == "download" {
		c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=export-%s-%s.pcap", guid, time.Now().Format(time.RFC3339)))
		if err := c.Blob(http.StatusOK, "application/octet-stream", reply); err != nil {
			logrus.Error(err.Error())
		}
		c.Response().Flush()
		return nil
	} else {
		return httpresponse.CreateSuccessResponseWithJson(&c, http.StatusOK, reply)
	}
}
