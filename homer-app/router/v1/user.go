package apirouterv1

import (
	"github.com/jinzhu/gorm"
	"github.com/labstack/echo/v4"
	"github.com/sipcapture/homer-app/auth"
	controllerv1 "github.com/sipcapture/homer-app/controller/v1"
	"github.com/sipcapture/homer-app/data/service"
	"github.com/sipcapture/homer-app/utils/httpauth"
	"github.com/sipcapture/homer-app/utils/ldap"
)

func RouteUserDetailsApis(acc *echo.Group, session *gorm.DB) {
	// initialize service of user
	userService := service.UserService{ServiceConfig: service.ServiceConfig{Session: session}, LdapClient: nil}
	// initialize user controller
	urc := controllerv1.UserController{
		UserService: &userService,
	}
	// get all user
	acc.GET("/users", urc.GetUser)
	// create new user
	acc.POST("/users", urc.CreateUser, auth.IsAdmin)
	// update user
	acc.PUT("/users/:userGuid", urc.UpdateUser)
	// delete user
	acc.DELETE("/users/:userGuid", urc.DeleteUser, auth.IsAdmin)
}

func RouteUserApis(acc *echo.Group, session *gorm.DB, ldapClient *ldap.LDAPClient, httpAuth *httpauth.Client) {
	// initialize service of user
	userService := service.UserService{ServiceConfig: service.ServiceConfig{Session: session}, LdapClient: ldapClient, HttpAuth: httpAuth}
	// initialize user controller
	urc := controllerv1.UserController{
		UserService: &userService,
	}
	// user login
	acc.POST("/auth", urc.LoginUser)
}
