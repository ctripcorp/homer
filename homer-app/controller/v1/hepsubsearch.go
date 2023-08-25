package controllerv1

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sipcapture/homer-app/data/service"
	"github.com/sipcapture/homer-app/model"
	httpresponse "github.com/sipcapture/homer-app/network/response"
	"github.com/sirupsen/logrus"
)

type HepsubsearchController struct {
	Controller
	HepsubsearchService *service.HepsubsearchService
}

// swagger:operation POST /hepsub/search hep hepSubSearchDoHepsubsearch
//
// Add hepsubsearch item
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
func (hss *HepsubsearchController) DoHepsubsearch(c echo.Context) error {
	// Stub an user to be populated from the body
	searchObject := model.SearchObject{}
	err := c.Bind(&searchObject)
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}
	// validate input request body
	if err := c.Validate(searchObject); err != nil {
		logrus.Error(err.Error())
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}

	reply, err := hss.HepsubsearchService.DoHepSubSearch(searchObject)
	if err != nil {
		return httpresponse.CreateBadResponse(&c, http.StatusBadRequest, err.Error())
	}
	return httpresponse.CreateSuccessResponseWithJson(&c, http.StatusCreated, []byte(reply))
}
