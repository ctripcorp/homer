package apirouterv1

import (
	"github.com/jinzhu/gorm"
	"github.com/labstack/echo/v4"
	"github.com/sipcapture/homer-app/auth"
	controllerv1 "github.com/sipcapture/homer-app/controller/v1"
	"github.com/sipcapture/homer-app/data/service"
)

func RouteHepsubApis(acc *echo.Group, session *gorm.DB) {
	// initialize service of user
	HepsubService := service.HepsubService{ServiceConfig: service.ServiceConfig{Session: session}}
	// initialize user controller
	hs := controllerv1.HepsubController{
		HepsubService: &HepsubService,
	}
	// get all dashboards

	acc.GET("/hepsub/protocol/:id/:transaction", hs.GetHepSubFields)
	acc.GET("/hepsub/protocol", hs.GetHepSub)
	acc.GET("/hepsub/protocol/:guid", hs.GetHepSubAgainstGUID)
	acc.POST("/hepsub/protocol", hs.AddHepSub, auth.IsAdmin)
	acc.PUT("/hepsub/protocol/:guid", hs.UpdateHepSubAgainstGUID, auth.IsAdmin)
	acc.DELETE("/hepsub/protocol/:guid", hs.DeleteHepSubAgainstGUID, auth.IsAdmin)
}
