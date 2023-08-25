package apirouterv1

import (
	"github.com/jinzhu/gorm"
	"github.com/labstack/echo/v4"
	controllerv1 "github.com/sipcapture/homer-app/controller/v1"
	"github.com/sipcapture/homer-app/data/service"
)

func RouteUserSettingsApis(acc *echo.Group, session *gorm.DB) {
	// initialize service of user
	userSettingService := service.UserSettingsService{ServiceConfig: service.ServiceConfig{Session: session}}
	// initialize user controller
	urc := controllerv1.UserSettingsController{
		UserSettingsService: &userSettingService,
	}
	// get user settings
	acc.GET("/user/settings", urc.GetAll)
	acc.GET("/user/settings/:category", urc.GetCategory)

	acc.POST("/user/settings", urc.AddUserSettings)
	acc.DELETE("/user/settings/:category", urc.DeleteUserSettings)
	acc.PUT("/user/settings/:category", urc.UpdateUserSettings)

}
