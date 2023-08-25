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

type HepsubController struct {
	Controller
	HepsubService *service.HepsubService
}

// swagger:route GET /hepsub/protocol hep hepSubGetHepSub
//
// Get all hepsub
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
func (hsc *HepsubController) GetHepSub(c echo.Context) error {

	reply, err := hsc.HepsubService.GetHepSub()
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, webmessages.HepSubRequestFailed)
	}
	return httpresponse.CreateSuccessResponseWithJson(&c, http.StatusOK, []byte(reply))

}

// swagger:operation GET /hepsub/protocol/{guid} hep hepSubGetHepSubAgainstGUID
//
// Get hepsub by guid
// ---
// consumes:
// - application/json
// produces:
// - application/json
// parameters:
// - name: guid
//   in: path
//   example: 11111111-1111-1111-1111-111111111111
//   description: guid of item
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
func (hsc *HepsubController) GetHepSubAgainstGUID(c echo.Context) error {
	guid := url.QueryEscape(c.Param("guid"))
	reply, err := hsc.HepsubService.GetHepSubAgainstGUID(guid)
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}
	return httpresponse.CreateSuccessResponseWithJson(&c, http.StatusOK, []byte(reply))

}

// swagger:operation GET /hepsub/protocol/{id}/{transaction} hep hepSubGetHepSubFields
//
// Get hepsub by id and transaction
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
//   type: string
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
func (hsc *HepsubController) GetHepSubFields(c echo.Context) error {
	id := url.QueryEscape(c.Param("id"))
	transaction := url.QueryEscape(c.Param("transaction"))
	reply, err := hsc.HepsubService.GetHepSubFields(id, transaction)
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}
	return httpresponse.CreateSuccessResponseWithJson(&c, http.StatusOK, []byte(reply))

}

// swagger:operation POST /hepsub/protocol hep hepSubAddHepSub
//
// Add new Hepsub information
// ---
// consumes:
// - application/json
// produces:
// - application/json
// parameters:
// - name: hepSubStruct
//   in: body
//   description: hepSub parameters
//   schema:
//     "$ref": "#/definitions/HepsubSchema"
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
func (hsc *HepsubController) AddHepSub(c echo.Context) error {
	// Stub an user to be populated from the body
	u := model.TableHepsubSchema{}
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
	reply, err := hsc.HepsubService.AddHepSub(u)
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}
	return httpresponse.CreateSuccessResponseWithJson(&c, http.StatusCreated, []byte(reply))
}

// swagger:operation PUT /hepsub/protocol/{guid} hep hepSubUpdateHepSubAgainstGUID
//
// Update hepsub by guid
// ---
// consumes:
// - application/json
// produces:
// - application/json
// parameters:
// - name: guid
//   in: path
//   example: eacdae5b-4203-40a2-b388-969312ffcffe
//   description: guid of hepsub item
//   required: true
//   type: string
// - name: hepSubStruct
//   in: body
//   description: hepSub parameters
//   schema:
//     "$ref": "#/definitions/HepsubSchema"
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
func (hsc *HepsubController) UpdateHepSubAgainstGUID(c echo.Context) error {
	guid := url.QueryEscape(c.Param("guid"))
	reply, err := hsc.HepsubService.GetHepSubAgainstGUID(guid)
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}
	// Stub an user to be populated from the body
	u := model.TableHepsubSchema{}
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
	reply, err = hsc.HepsubService.UpdateHepSubAgainstGUID(guid, u)
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}
	return httpresponse.CreateSuccessResponseWithJson(&c, http.StatusOK, []byte(reply))
}

// swagger:operation DELETE /hepsub/protocol/{guid} hep hepSubDeleteHepSubAgainstGUID
//
// Delete hepsub by guid
// ---
// consumes:
// - application/json
// produces:
// - application/json
// parameters:
// - name: guid
//   in: path
//   example: eacdae5b-4203-40a2-b388-969312ffcffe
//   description: guid of hepsub item
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
func (hsc *HepsubController) DeleteHepSubAgainstGUID(c echo.Context) error {
	guid := url.QueryEscape(c.Param("guid"))

	reply, err := hsc.HepsubService.GetHepSubAgainstGUID(guid)
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}
	reply, err = hsc.HepsubService.DeleteHepSubAgainstGUID(guid)
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}
	reply, err = hsc.HepsubService.DeleteHepSubAgainstGUID(guid)
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}
	return httpresponse.CreateSuccessResponseWithJson(&c, http.StatusOK, []byte(reply))
}
