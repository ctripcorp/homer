package service

import (
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Jeffail/gabs/v2"
	uuid "github.com/satori/go.uuid"
	"github.com/sipcapture/homer-app/model"
)

type UserSettingsService struct {
	ServiceConfig
}

func (ss *UserSettingsService) GetCorrelationMap(data *model.SearchObject) ([]model.TableMappingSchema, error) {

	mappingSchema := []model.TableMappingSchema{}
	Data, _ := json.Marshal(data.Param.Search)
	sData, _ := gabs.ParseJSON(Data)
	hepid := 1
	//profile := "call"

	for key := range sData.ChildrenMap() {
		dataParse := strings.Split(key, "_")
		hepid, _ = strconv.Atoi(dataParse[0])
		//profile = dataParse[1]
	}

	ss.Session.Debug().
		Table("mapping_schema").
		Where("hepid = ? and profile in ('call', 'default', 'registration')", hepid).
		Find(&mappingSchema)

	return mappingSchema, nil
}

// get Category by param
func (as *UserSettingsService) GetScriptByParam(category string, scriptName string) (string, error) {

	var userGlobalSettings = model.TableGlobalSettings{}
	if err := as.Session.Debug().
		Table("global_settings").
		Where("category = ? AND param = ? AND partid = 10", category, scriptName).
		First(&userGlobalSettings).Error; err != nil {
		return "", errors.New("no users settings found")
	}

	return string(userGlobalSettings.Data), nil
}

/* get all */
func (ss *UserSettingsService) GetAll(UserName string, isAdmin bool) (string, error) {
	var userSettings = []model.TableUserSettings{}

	var sqlWhere = make(map[string]interface{})

	if !isAdmin {
		sqlWhere = map[string]interface{}{"username": UserName}
	}

	if err := ss.Session.Debug().
		Table("user_settings").
		Where(sqlWhere).
		Find(&userSettings).Error; err != nil {
		return "", errors.New("no users settings found")
	}
	data, _ := json.Marshal(userSettings)
	rows, _ := gabs.ParseJSON(data)
	// sort the array
	vals := rows.Data().([]interface{})
	sort.Slice(vals, func(i, j int) bool {
		return gabs.Wrap(vals[i]).S("username").Data().(string) < gabs.Wrap(vals[j]).S("username").Data().(string)
	})
	count, _ := rows.ArrayCount()
	reply := gabs.New()
	reply.Set(count, "count")
	reply.Set(vals, "data")
	return reply.String(), nil
}

/* get only for this user and category */
func (ss *UserSettingsService) GetCategory(UserName string, UserCategory string) (string, error) {

	var userSettings = []model.TableUserSettings{}

	var sqlWhere = map[string]interface{}{"username": UserName, "category": UserCategory}

	if err := ss.Session.Debug().
		Table("user_settings").
		Where(sqlWhere).
		Find(&userSettings).Error; err != nil {
		return "", errors.New("no users settings found")
	}
	data, _ := json.Marshal(userSettings)
	rows, _ := gabs.ParseJSON(data)
	// sort the array
	vals := rows.Data().([]interface{})
	sort.Slice(vals, func(i, j int) bool {
		return gabs.Wrap(vals[i]).S("username").Data().(string) < gabs.Wrap(vals[j]).S("username").Data().(string)
	})
	count, _ := rows.ArrayCount()
	reply := gabs.New()
	reply.Set(count, "count")
	reply.Set(vals, "data")
	return reply.String(), nil
}

// this method create new userSetting in the database
// it doesn't check internally whether all the validation are applied or not
func (ss *UserSettingsService) Add(userObject *model.TableUserSettings) (string, error) {
	u1 := uuid.NewV4()
	userObject.GUID = u1.String()
	userObject.CreateDate = time.Now()
	if err := ss.Session.Debug().
		Table("user_settings").
		Create(&userObject).Error; err != nil {
		return "", err
	}
	reply := gabs.New()
	reply.Set(u1.String(), "data")
	reply.Set("successfully created userObject", "message")
	return reply.String(), nil
}

// this method create new user in the database
// it doesn't check internally whether all the validation are applied or not
func (ss *UserSettingsService) Get(userObject *model.TableUserSettings, UserName string, isAdmin bool) (model.TableUserSettings, error) {
	data := model.TableUserSettings{}

	var sqlWhere = make(map[string]interface{})

	if !isAdmin {
		sqlWhere = map[string]interface{}{"guid": userObject.GUID, "username": UserName}
	} else {
		sqlWhere = map[string]interface{}{"guid": userObject.GUID}
	}

	if err := ss.Session.Debug().
		Table("user_settings").
		Where(sqlWhere).Find(&data).Error; err != nil {
		return data, err
	}
	return data, nil
}

// this method create new user in the database
// it doesn't check internally whether all the validation are applied or not
func (ss *UserSettingsService) Delete(userObject *model.TableUserSettings, UserName string, isAdmin bool) error {

	var sqlWhere = make(map[string]interface{})

	if !isAdmin {
		sqlWhere = map[string]interface{}{"guid": userObject.GUID, "username": UserName}
	} else {
		sqlWhere = map[string]interface{}{"guid": userObject.GUID}
	}

	if err := ss.Session.Debug().
		Table("user_settings").
		Where(sqlWhere).
		Delete(model.TableUserSettings{}).Error; err != nil {
		return err
	}
	return nil
}

// this method create new user in the database
// it doesn't check internally whether all the validation are applied or not
func (ss *UserSettingsService) Update(userObject *model.TableUserSettings, UserName string, isAdmin bool) error {

	var sqlWhere = make(map[string]interface{})

	if !isAdmin {
		sqlWhere = map[string]interface{}{"guid": userObject.GUID, "username": UserName}
	} else {
		sqlWhere = map[string]interface{}{"guid": userObject.GUID}
	}

	if err := ss.Session.Debug().
		Table("user_settings").
		Debug().
		Model(&model.TableUserSettings{}).
		Where(sqlWhere).Update(userObject).Error; err != nil {
		return err
	}
	return nil
}
