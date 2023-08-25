package apirouterv1

import (
	"github.com/labstack/echo/v4"
	controllerv1 "github.com/sipcapture/homer-app/controller/v1"
	"github.com/sipcapture/homer-app/data/service"
)

// RouteStatisticApis : here you tell us what RouteStatisticApis is
func RouteLokiApis(acc *echo.Group, serviceLoki service.ServiceLoki) {

	// initialize service of user
	remoteService := service.RemoteService{ServiceLoki: serviceLoki}

	// initialize user controller
	src := controllerv1.RemoteController{
		RemoteService: &remoteService,
	}

	// create new stats
	acc.GET("/search/remote/label", src.RemoteLabel)

	// create new stats
	acc.GET("/search/remote/values", src.RemoteValues)

	// create new stats
	acc.POST("/search/remote/data", src.RemoteData)

}
