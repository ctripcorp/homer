package apirouterv1

import (
	"github.com/jinzhu/gorm"
	"github.com/labstack/echo/v4"
	"github.com/sipcapture/homer-app/auth"
	controllerv1 "github.com/sipcapture/homer-app/controller/v1"
	"github.com/sipcapture/homer-app/data/service"
)

func RouteAdvancedApis(acc *echo.Group, configSession *gorm.DB) {
	// initialize service of user
	AdvancedService := service.AdvancedService{ServiceConfig: service.ServiceConfig{Session: configSession}}
	// initialize user controller
	ac := controllerv1.AdvancedController{
		AdvancedService: &AdvancedService,
	}
	acc.GET("/advanced", ac.GetAll)
	acc.GET("/advanced/:guid", ac.GetAdvancedAgainstGUID)
	acc.POST("/advanced", ac.AddAdvanced, auth.IsAdmin)
	acc.PUT("/advanced/:guid", ac.UpdateAdvancedAgainstGUID, auth.IsAdmin)
	acc.DELETE("/advanced/:guid", ac.DeleteAdvancedAgainstGUID, auth.IsAdmin)
}
