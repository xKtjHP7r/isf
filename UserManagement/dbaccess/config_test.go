package dbaccess

import (
	"testing"

	"github.com/kweaver-ai/go-lib/rest"
	"github.com/kweaver-ai/proton-rds-sdk-go/sqlx"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-playground/assert"
	. "github.com/smartystreets/goconvey/convey"

	"UserManagement/common"
	"UserManagement/interfaces"
)

func newConfigDB(ptrDB *sqlx.DB) *config {
	return &config{
		db:     ptrDB,
		logger: common.NewLogger(),
		confKeyMap: map[interfaces.ConfigKey]string{
			interfaces.UserDefaultSha2PWD: "user_defalut_sha2_password",
			interfaces.UserDefaultNTLMPWD: "user_defalut_ntlm_password",
			interfaces.UserDefaultDESPWD:  "user_defalut_des_password",
			interfaces.UserDefaultMd5PWD:  "user_defalut_md5_password",
			interfaces.IDCardLogin:        "id_card_login_status",
			interfaces.TelPwdRetrieval:    "vcode_server_status",
			interfaces.EmailPwdRetrieval:  "vcode_server_status",
			interfaces.PWDExpireTime:      "pwd_expire_time",
			interfaces.StrongPWDStatus:    "strong_pwd_status",
			interfaces.StrongPWDLength:    "strong_pwd_length",
			interfaces.EnablePWDLock:      "enable_pwd_lock",
			interfaces.PWDErrCnt:          "pwd_err_cnt",
			interfaces.PWDLockTime:        "pwd_lock_time",
			interfaces.EnableDesPassWord:  "enable_des_password",
			interfaces.ShowCSFLevel2:      "show_csf_level2",
			interfaces.CSFLevelEnum:       "csf_level_enum",
			interfaces.CSFLevel2Enum:      "csf_level2_enum",
			interfaces.AutoDisable:        "auto_disable_config",
		},
	}
}

func TestNewConfig(t *testing.T) {
	Convey("NewConfig", t, func() {
		data := NewConfig()
		assert.NotEqual(t, data, nil)
	})
}

//nolint:dupl
func TestGetConfigs(t *testing.T) {
	Convey("getConfigs, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		config := newConfigDB(db)

		ck := make(map[interfaces.ConfigKey]bool)
		Convey("未传入key，直接返回", func() {
			fields := []string{
				"f_key",
				"f_value",
			}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields))
			_, httpErr := config.GetConfig(ck)
			assert.Equal(t, httpErr, nil)
		})

		ck[interfaces.IDCardLogin] = true
		ck[interfaces.EmailPwdRetrieval] = true
		ck[interfaces.TelPwdRetrieval] = true
		ck[interfaces.PWDExpireTime] = true
		ck[interfaces.StrongPWDStatus] = true
		ck[interfaces.StrongPWDLength] = true
		ck[interfaces.EnablePWDLock] = true
		ck[interfaces.PWDErrCnt] = true
		ck[interfaces.PWDLockTime] = true
		ck[interfaces.EnableDesPassWord] = true
		ck[interfaces.ShowCSFLevel2] = true
		ck[interfaces.CSFLevelEnum] = true
		ck[interfaces.CSFLevel2Enum] = true
		ck[interfaces.AutoDisable] = true
		Convey("success", func() {
			fields := []string{
				"f_key",
				"f_value",
			}

			rows := sqlmock.NewRows(fields).AddRow(config.confKeyMap[interfaces.IDCardLogin], "1").
				AddRow(config.confKeyMap[interfaces.TelPwdRetrieval], "{\"send_vcode_by_sms\":true,\"send_vcode_by_email\":true}").
				AddRow(config.confKeyMap[interfaces.EmailPwdRetrieval], "{\"send_vcode_by_sms\":true,\"send_vcode_by_email\":true}").
				AddRow(config.confKeyMap[interfaces.PWDExpireTime], "200").
				AddRow(config.confKeyMap[interfaces.StrongPWDStatus], "1").
				AddRow(config.confKeyMap[interfaces.StrongPWDLength], "8").
				AddRow(config.confKeyMap[interfaces.EnablePWDLock], "1").
				AddRow(config.confKeyMap[interfaces.PWDErrCnt], "5").
				AddRow(config.confKeyMap[interfaces.PWDLockTime], "1").
				AddRow(config.confKeyMap[interfaces.EnableDesPassWord], "1").
				AddRow(config.confKeyMap[interfaces.ShowCSFLevel2], "0").
				AddRow(config.confKeyMap[interfaces.CSFLevelEnum], "{\"公开\":5,\"秘密\":6,\"机密\":7,\"绝密\":8}").
				AddRow(config.confKeyMap[interfaces.CSFLevel2Enum], "{\"公开1\":51,\"秘密1\":52,\"机密1\":53,\"绝密1\":54}").
				AddRow(config.confKeyMap[interfaces.AutoDisable], "{\"isEnabled\":0, \"days\":90}")
			mock.ExpectQuery("").WillReturnRows(rows)
			data, httpErr := config.GetConfig(ck)
			assert.Equal(t, httpErr, nil)
			assert.Equal(t, data.IDCardLogin, true)
			assert.Equal(t, data.TelPwdRetrieval, true)
			assert.Equal(t, data.EmailPwdRetrieval, true)
			assert.Equal(t, data.PwdExpireTime, int64(200))
			assert.Equal(t, data.StrongPwdStatus, true)
			assert.Equal(t, data.StrongPwdLength, 8)
			assert.Equal(t, data.EnablePwdLock, true)
			assert.Equal(t, data.PwdErrCnt, 5)
			assert.Equal(t, data.PwdLockTime, int64(1))
			assert.Equal(t, data.EnableDesPwd, true)
			assert.Equal(t, data.ShowCSFLevel2, false)
			assert.Equal(t, data.CSFLevelEnum["公开"], 5)
			assert.Equal(t, data.CSFLevelEnum["秘密"], 6)
			assert.Equal(t, data.CSFLevelEnum["机密"], 7)
			assert.Equal(t, data.CSFLevelEnum["绝密"], 8)
			assert.Equal(t, data.CSFLevel2Enum["公开1"], 51)
			assert.Equal(t, data.CSFLevel2Enum["秘密1"], 52)
			assert.Equal(t, data.CSFLevel2Enum["机密1"], 53)
			assert.Equal(t, data.CSFLevel2Enum["绝密1"], 54)
			assert.Equal(t, data.AutoDisableEnabled, false)
			assert.Equal(t, data.AutoDisableTime, int64(90))
			Convey("success1", func() {
				fields := []string{
					"f_key",
					"f_value",
				}

				ck1 := make(map[interfaces.ConfigKey]bool)
				ck1[interfaces.ShowCSFLevel2] = true
				rows := sqlmock.NewRows(fields)
				mock.ExpectQuery("").WillReturnRows(rows)
				data, httpErr := config.GetConfig(ck1)
				assert.Equal(t, httpErr, nil)
				assert.Equal(t, data.ShowCSFLevel2, false)
			})
		})

		Convey("success1", func() {
			fields := []string{
				"f_key",
				"f_value",
			}

			rows := sqlmock.NewRows(fields).AddRow(config.confKeyMap[interfaces.IDCardLogin], "1").
				AddRow(config.confKeyMap[interfaces.TelPwdRetrieval], "{\"send_vcode_by_sms\":true,\"send_vcode_by_email\":true}").
				AddRow(config.confKeyMap[interfaces.EmailPwdRetrieval], "{\"send_vcode_by_sms\":true,\"send_vcode_by_email\":true}").
				AddRow(config.confKeyMap[interfaces.PWDExpireTime], "200").
				AddRow(config.confKeyMap[interfaces.StrongPWDStatus], "1").
				AddRow(config.confKeyMap[interfaces.StrongPWDLength], "8").
				AddRow(config.confKeyMap[interfaces.EnablePWDLock], "1").
				AddRow(config.confKeyMap[interfaces.PWDErrCnt], "5").
				AddRow(config.confKeyMap[interfaces.PWDLockTime], "1").
				AddRow(config.confKeyMap[interfaces.EnableDesPassWord], "1").
				AddRow(config.confKeyMap[interfaces.ShowCSFLevel2], "0").
				AddRow(config.confKeyMap[interfaces.CSFLevelEnum], "{\"公开\":5,\"秘密\":6,\"机密\":7,\"绝密\":8}").
				AddRow(config.confKeyMap[interfaces.CSFLevel2Enum], "{\"公开1\":51,\"秘密1\":52,\"机密1\":53,\"绝密1\":54}").
				AddRow(config.confKeyMap[interfaces.AutoDisable], "{\"isEnabled\":true, \"days\":90}")
			mock.ExpectQuery("").WillReturnRows(rows)
			data, httpErr := config.GetConfig(ck)
			assert.Equal(t, httpErr, nil)
			assert.Equal(t, data.IDCardLogin, true)
			assert.Equal(t, data.TelPwdRetrieval, true)
			assert.Equal(t, data.EmailPwdRetrieval, true)
			assert.Equal(t, data.PwdExpireTime, int64(200))
			assert.Equal(t, data.StrongPwdStatus, true)
			assert.Equal(t, data.StrongPwdLength, 8)
			assert.Equal(t, data.EnablePwdLock, true)
			assert.Equal(t, data.PwdErrCnt, 5)
			assert.Equal(t, data.PwdLockTime, int64(1))
			assert.Equal(t, data.EnableDesPwd, true)
			assert.Equal(t, data.ShowCSFLevel2, false)
			assert.Equal(t, data.CSFLevelEnum["公开"], 5)
			assert.Equal(t, data.CSFLevelEnum["秘密"], 6)
			assert.Equal(t, data.CSFLevelEnum["机密"], 7)
			assert.Equal(t, data.CSFLevelEnum["绝密"], 8)
			assert.Equal(t, data.CSFLevel2Enum["公开1"], 51)
			assert.Equal(t, data.CSFLevel2Enum["秘密1"], 52)
			assert.Equal(t, data.CSFLevel2Enum["机密1"], 53)
			assert.Equal(t, data.CSFLevel2Enum["绝密1"], 54)
			assert.Equal(t, data.AutoDisableEnabled, true)
			assert.Equal(t, data.AutoDisableTime, int64(90))
			Convey("success1", func() {
				fields := []string{
					"f_key",
					"f_value",
				}

				ck1 := make(map[interfaces.ConfigKey]bool)
				ck1[interfaces.ShowCSFLevel2] = true
				rows := sqlmock.NewRows(fields)
				mock.ExpectQuery("").WillReturnRows(rows)
				data, httpErr := config.GetConfig(ck1)
				assert.Equal(t, httpErr, nil)
				assert.Equal(t, data.ShowCSFLevel2, false)
			})
		})
	})
}

func TestSetConfig(t *testing.T) {
	Convey("更新配置", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		config := newConfigDB(db)

		key := make(map[interfaces.ConfigKey]bool)
		var cfg interfaces.Config
		testErr := rest.NewHTTPError("error", 503000000, nil)

		key[interfaces.UserDefaultNTLMPWD] = true
		key[interfaces.UserDefaultMd5PWD] = true
		key[interfaces.UserDefaultSha2PWD] = true
		key[interfaces.UserDefaultDESPWD] = true
		Convey("更新配置信息报错", func() {
			mock.ExpectBegin()
			mock.ExpectExec("").WillReturnError(testErr)

			tx, err := db.Begin()
			assert.Equal(t, err, nil)

			err = config.SetConfig(key, &cfg, tx)
			assert.Equal(t, err, testErr)
		})

		Convey("成功", func() {
			mock.ExpectBegin()
			mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(-1, -1))
			mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(-1, -1))
			mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(-1, -1))
			mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(-1, -1))

			tx, err := db.Begin()
			assert.Equal(t, err, nil)

			err = config.SetConfig(key, &cfg, tx)
			assert.Equal(t, err, nil)
		})
	})
}

func TestSetShareMgntConfig(t *testing.T) {
	Convey("更新密级枚举成功", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		config := newConfigDB(db)
		testErr := rest.NewHTTPError("error", 503000000, nil)

		Convey("获取字段失败", func() {
			var cfg interfaces.Config
			cfg.CSFLevel2Enum = map[string]int{
				"test2": 2,
			}

			mock.ExpectBegin()
			mock.ExpectQuery("select").WillReturnError(testErr)
			tx, err := db.Begin()
			assert.Equal(t, err, nil)
			err = config.SetShareMgntConfig(interfaces.CSFLevel2Enum, &cfg, tx)
			assert.Equal(t, err, testErr)
		})

		Convey("获取字段成功，但不存在，插入密级枚举失败", func() {
			var cfg interfaces.Config
			cfg.CSFLevel2Enum = map[string]int{
				"test2": 2,
			}

			mock.ExpectBegin()
			mock.ExpectQuery("select").WillReturnRows(sqlmock.NewRows([]string{"f_value"}))
			mock.ExpectExec("insert").WillReturnError(testErr)
			tx, err := db.Begin()
			assert.Equal(t, err, nil)
			err = config.SetShareMgntConfig(interfaces.CSFLevel2Enum, &cfg, tx)
			assert.Equal(t, err, testErr)
		})

		Convey("获取字段成功，存在，更新密级枚举失败", func() {
			var cfg interfaces.Config
			cfg.CSFLevel2Enum = map[string]int{
				"test2": 2,
			}

			mock.ExpectBegin()
			mock.ExpectQuery("select").WillReturnRows(sqlmock.NewRows([]string{"f_value"}).AddRow("{\"test\":1,\"test2\":2}"))
			mock.ExpectExec("update").WillReturnError(testErr)
			tx, err := db.Begin()
			assert.Equal(t, err, nil)
			err = config.SetShareMgntConfig(interfaces.CSFLevel2Enum, &cfg, tx)
			assert.Equal(t, err, testErr)
		})

		Convey("更新密级枚举成功", func() {
			testConfig := interfaces.Config{
				CSFLevel2Enum: map[string]int{
					"test2": 2,
				}}
			mock.ExpectBegin()
			mock.ExpectQuery("select").WillReturnRows(sqlmock.NewRows([]string{"f_value"}).AddRow("{\"test\":1,\"test2\":2}"))
			mock.ExpectExec("update").WillReturnResult(sqlmock.NewResult(-1, -1))
			tx, err := db.Begin()
			assert.Equal(t, err, nil)
			err = config.SetShareMgntConfig(interfaces.CSFLevel2Enum, &testConfig, tx)
			assert.Equal(t, err, nil)
		})

		Convey("插入密级枚举成功", func() {
			testConfig := interfaces.Config{
				CSFLevel2Enum: map[string]int{
					"test2": 2,
				},
			}
			mock.ExpectBegin()
			mock.ExpectQuery("select").WillReturnRows(sqlmock.NewRows([]string{"f_value"}))
			mock.ExpectExec("insert").WillReturnResult(sqlmock.NewResult(-1, -1))
			tx, err := db.Begin()
			assert.Equal(t, err, nil)
			err = config.SetShareMgntConfig(interfaces.CSFLevel2Enum, &testConfig, tx)
			assert.Equal(t, err, nil)
		})
	})
}
