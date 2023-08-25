package controllerv1

import (
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"
	uuid "github.com/satori/go.uuid"
	"github.com/sipcapture/homer-app/data/service"
	"github.com/sipcapture/homer-app/model"
	httpresponse "github.com/sipcapture/homer-app/network/response"
	"github.com/sipcapture/homer-app/system/webmessages"
	"github.com/sirupsen/logrus"
)

type MappingController struct {
	Controller
	MappingService *service.MappingService
}

// swagger:route GET /mapping/protocol mapping mappingGetMapping
//
// Get all mappings
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
//   '201': body:UserCreateSuccessfulResponse
//   '400': body:UserCreateSuccessfulResponse
func (mpc *MappingController) GetMapping(c echo.Context) error {
	reply, err := mpc.MappingService.GetMapping()
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}
	return httpresponse.CreateSuccessResponseWithJson(&c, http.StatusOK, []byte(reply))
}

// swagger:operation GET /mapping/protocol/{id}/{transaction} mapping mappingGetMappingFields
//
// Get mapping using id and transaction
// ---
// consumes:
// - application/json
// produces:
// - application/json
// parameters:
// - name: id
//   in: path
//   example: 1
//   description: hepid
//   required: true
//   type: int
// parameters:
// - name: transaction
//   in: path
//   example: call
//   description: profile
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
func (mpc *MappingController) GetMappingFields(c echo.Context) error {

	id := url.QueryEscape(c.Param("id"))
	transaction := url.QueryEscape(c.Param("transaction"))
	reply, err := mpc.MappingService.GetMappingFields(id, transaction)
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, webmessages.MappingSchemaFailed)
	}
	return httpresponse.CreateSuccessResponseWithJson(&c, http.StatusOK, []byte(reply))

}

// swagger:operation GET /mapping/protocol/{guid} mapping mappingGetMappingAgainstGUID
//
// Get mapping using guid
// ---
// consumes:
// - application/json
// produces:
// - application/json
// parameters:
// - name: guid
//   in: path
//   example: 11111111-1111-1111-1111-111111111111
//   description: guid of mapping
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
func (mpc *MappingController) GetMappingAgainstGUID(c echo.Context) error {

	guid := url.QueryEscape(c.Param("guid"))
	reply, err := mpc.MappingService.GetMappingAgainstGUID(guid)
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, webmessages.MappingSchemaByUUIDFailed)
	}
	return httpresponse.CreateSuccessResponseWithJson(&c, http.StatusOK, []byte(reply))
}

// swagger:operation POST /mapping/protocol mapping mappingAddMapping
//
// Add mapping
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
//   '201': body:UserCreateSuccessfulResponse
//   '400': body:UserCreateSuccessfulResponse
func (mpc *MappingController) AddMapping(c echo.Context) error {
	// Stub an user to be populated from the body
	u := model.TableMappingSchema{}
	err := c.Bind(&u)
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}
	// validate input request body
	if err := c.Validate(u); err != nil {
		logrus.Error(err.Error())
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}

	uid := uuid.NewV4()
	u.GUID = uid.String()
	reply, err := mpc.MappingService.AddMapping(u)
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}
	return httpresponse.CreateSuccessResponseWithJson(&c, http.StatusCreated, []byte(reply))
}

// swagger:operation PUT /mapping/protocol/{guid} mapping mappingUpdateMappingAgainstGUID
//
// Update mapping using guid
// ---
// consumes:
// - application/json
// produces:
// - application/json
// parameters:
// - name: guid
//   in: path
//   example: 11111111-1111-1111-1111-111111111111
//   description: guid of mapping
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
func (mpc *MappingController) UpdateMappingAgainstGUID(c echo.Context) error {
	guid := url.QueryEscape(c.Param("guid"))
	reply, err := mpc.MappingService.GetMappingAgainstGUID(guid)
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}
	// Stub an user to be populated from the body
	u := model.TableMappingSchema{}
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
	reply, err = mpc.MappingService.UpdateMappingAgainstGUID(guid, u)
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}
	return httpresponse.CreateSuccessResponseWithJson(&c, http.StatusOK, []byte(reply))
}

// swagger:operation DELETE /mapping/protocol/{guid} mapping mappingDeleteMappingAgainstGUID
//
// Delete mapping using guid
// ---
// consumes:
// - application/json
// produces:
// - application/json
// parameters:
// - name: guid
//   in: path
//   example: 11111111-1111-1111-1111-111111111111
//   description: guid of mapping
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
func (mpc *MappingController) DeleteMappingAgainstGUID(c echo.Context) error {
	guid := url.QueryEscape(c.Param("guid"))
	reply, err := mpc.MappingService.GetMappingAgainstGUID(guid)
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}
	reply, err = mpc.MappingService.DeleteMappingAgainstGUID(guid)
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, webmessages.DeleteMappingSchemaFailed)
	}
	return httpresponse.CreateSuccessResponseWithJson(&c, http.StatusOK, []byte(reply))
}

// swagger:operation GET /smart/search/tag/{hepid}/{profile} mapping mappingGetSmartHepProfile
//
// Get smart hep profile using hepid and profile
// ---
// consumes:
// - application/json
// produces:
// - application/json
// parameters:
// - name: hepid
//   in: path
//   example: 1
//   description: hepid
//   required: true
//   type: string
// - name: profile
//   in: path
//   example:
//   description: profile
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
func (mpc *MappingController) GetSmartHepProfile(c echo.Context) error {

	hepid := url.QueryEscape(c.Param("hepid"))
	profile := url.QueryEscape(c.Param("profile"))

	reply, err := mpc.MappingService.GetSmartSuggestionAginstProfile(hepid, profile)
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, webmessages.SmartHepProfileFailed)
	}
	return httpresponse.CreateSuccessResponseWithJson(&c, http.StatusOK, []byte(reply))
}
