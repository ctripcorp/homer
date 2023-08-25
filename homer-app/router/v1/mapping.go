package apirouterv1

import (
	"github.com/jinzhu/gorm"
	"github.com/labstack/echo/v4"
	"github.com/sipcapture/homer-app/auth"
	controllerv1 "github.com/sipcapture/homer-app/controller/v1"
	"github.com/sipcapture/homer-app/data/service"
)

func RouteMappingdApis(acc *echo.Group, session *gorm.DB) {
	// initialize service of user
	mappingService := service.MappingService{ServiceConfig: service.ServiceConfig{Session: session}}
	// initialize user controller
	mpc := controllerv1.MappingController{
		MappingService: &mappingService,
	}
	// get all dashboards
	acc.GET("/mapping/protocol", mpc.GetMapping)
	acc.GET("/mapping/protocol/:id/:transaction", mpc.GetMappingFields)
	acc.GET("/mapping/protocol/:guid", mpc.GetMappingAgainstGUID)
	acc.POST("/mapping/protocol", mpc.AddMapping, auth.IsAdmin)
	acc.PUT("/mapping/protocol/:guid", mpc.UpdateMappingAgainstGUID, auth.IsAdmin)
	acc.DELETE("/mapping/protocol/:guid", mpc.DeleteMappingAgainstGUID, auth.IsAdmin)

	/* search smart */
	acc.GET("/smart/search/tag/:hepid/:profile", mpc.GetSmartHepProfile)

}
