// Package dbaccess config Anyshare 数据访问层 - t_sharemgnt_config 数据库操作
//
//nolint:exhaustive
package dbaccess

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/kweaver-ai/proton-rds-sdk-go/sqlx"
	jsoniter "github.com/json-iterator/go"

	"UserManagement/common"
	"UserManagement/interfaces"
)

type config struct {
	db         *sqlx.DB
	logger     common.Logger
	confKeyMap map[interfaces.ConfigKey]string
}

var (
	configOnce sync.Once
	conDB      *config
)

// NewConfig 创建数据库操作对象
func NewConfig() *config {
	configOnce.Do(func() {
		conDB = &config{
			db:     dbPool,
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
				interfaces.EnableThirdPwdLock: "enable_third_pwd_lock",
				interfaces.CSFLevelEnum:       "csf_level_enum",
				interfaces.CSFLevel2Enum:      "csf_level2_enum",
				interfaces.ShowCSFLevel2:      "show_csf_level2",
				interfaces.AutoDisable:        "auto_disable_config",
			},
		}
	})

	return conDB
}

// GetConfig 获取配置信息
func (d *config) GetConfig(configKeys map[interfaces.ConfigKey]bool) (config interfaces.Config, err error) {
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "select `f_key`, `f_value` from `" + dbName + "`.`t_sharemgnt_config` where `f_key` in (%s)"

	return d.getConfig(configKeys, strSQL)
}

// GetConfigFromOption 获取配置信息
func (d *config) GetConfigFromOption(configKeys map[interfaces.ConfigKey]bool) (config interfaces.Config, err error) {
	dbName := common.GetDBName("user_management")
	strSQL := "select `key`, `value` from `" + dbName + "`.`option` where `key` in (%s)"

	return d.getConfig(configKeys, strSQL)
}

func (d *config) getConfig(configKeys map[interfaces.ConfigKey]bool, strSQL string) (config interfaces.Config, err error) {
	if len(configKeys) == 0 {
		return
	}

	groupsStr := make([]string, 0)
	configs := make([]interface{}, 0, len(configKeys))
	for k := range configKeys {
		configs = append(configs, d.confKeyMap[k])
		groupsStr = append(groupsStr, "?")
	}
	groupStr := strings.Join(groupsStr, ",")
	rows, err := d.db.Query(fmt.Sprintf(strSQL, groupStr), configs...)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				d.logger.Errorln(rowsErr)
			}

			// 1、判断是否为空再关闭，2、如果不关闭而数据行并没有被scan的话，连接一直会被占用直到超时断开
			if closeErr := rows.Close(); closeErr != nil {
				d.logger.Errorln(closeErr)
			}
		}
	}()

	if err != nil {
		d.logger.Errorln(err)
		return
	}

	tmpKVMap := make(map[string]string)
	for rows.Next() {
		var fKey, fValue string
		err = rows.Scan(&fKey, &fValue)
		if err != nil {
			return
		}
		tmpKVMap[fKey] = fValue
	}

	return d.convertToConfig(configKeys, tmpKVMap)
}

//nolint:gocyclo,funlen
func (d *config) convertToConfig(configKeys map[interfaces.ConfigKey]bool, kvMap map[string]string) (cfg interfaces.Config, err error) {
	for k := range configKeys {
		switch k {
		case interfaces.UserDefaultSha2PWD:
			cfg.UserDefaultSha2PWD = kvMap[d.confKeyMap[interfaces.UserDefaultSha2PWD]]
		case interfaces.UserDefaultMd5PWD:
			cfg.UserDefaultMd5PWD = kvMap[d.confKeyMap[interfaces.UserDefaultMd5PWD]]
		case interfaces.IDCardLogin:
			var boolV bool
			boolV, err = strconv.ParseBool(kvMap[d.confKeyMap[interfaces.IDCardLogin]])
			if err != nil {
				return
			}
			cfg.IDCardLogin = boolV
		case interfaces.TelPwdRetrieval, interfaces.EmailPwdRetrieval:
			var value string
			if k == interfaces.EmailPwdRetrieval {
				value = d.confKeyMap[interfaces.EmailPwdRetrieval]
			} else {
				value = d.confKeyMap[interfaces.TelPwdRetrieval]
			}
			vcodeConf := make(map[string]interface{})
			err = jsoniter.Unmarshal([]byte(kvMap[value]), &vcodeConf)
			if err != nil {
				return cfg, errors.New("invalid vcode config")
			}

			bEmail, ret1 := vcodeConf["send_vcode_by_email"].(bool)
			BTel, ret2 := vcodeConf["send_vcode_by_sms"].(bool)
			if !ret1 || !ret2 {
				return cfg, errors.New("invalid vcode config")
			}
			cfg.TelPwdRetrieval = BTel
			cfg.EmailPwdRetrieval = bEmail
		case interfaces.PWDExpireTime:
			var int64V int64
			int64V, err = strconv.ParseInt(kvMap[d.confKeyMap[interfaces.PWDExpireTime]], 10, 64)
			if err != nil {
				return
			}
			cfg.PwdExpireTime = int64V
		case interfaces.StrongPWDStatus:
			var boolV bool
			boolV, err = strconv.ParseBool(kvMap[d.confKeyMap[interfaces.StrongPWDStatus]])
			if err != nil {
				return
			}
			cfg.StrongPwdStatus = boolV
		case interfaces.StrongPWDLength:
			var intV int
			intV, err = strconv.Atoi(kvMap[d.confKeyMap[interfaces.StrongPWDLength]])
			if err != nil {
				return
			}
			cfg.StrongPwdLength = intV
		case interfaces.EnablePWDLock:
			var boolV bool
			boolV, err = strconv.ParseBool(kvMap[d.confKeyMap[interfaces.EnablePWDLock]])
			if err != nil {
				return
			}
			cfg.EnablePwdLock = boolV
		case interfaces.PWDErrCnt:
			var intV int
			intV, err = strconv.Atoi(kvMap[d.confKeyMap[interfaces.PWDErrCnt]])
			if err != nil {
				return
			}
			cfg.PwdErrCnt = intV
		case interfaces.PWDLockTime:
			var int64V int64
			int64V, err = strconv.ParseInt(kvMap[d.confKeyMap[interfaces.PWDLockTime]], 10, 64)
			if err != nil {
				return
			}
			cfg.PwdLockTime = int64V
		case interfaces.EnableDesPassWord:
			var boolV bool
			boolV, err = strconv.ParseBool(kvMap[d.confKeyMap[interfaces.EnableDesPassWord]])
			if err != nil {
				return
			}
			cfg.EnableDesPwd = boolV
		case interfaces.EnableThirdPwdLock:
			var boolV bool
			boolV, err = strconv.ParseBool(kvMap[d.confKeyMap[interfaces.EnableThirdPwdLock]])
			if err != nil {
				return
			}
			cfg.EnableThirdPwdLock = boolV
		case interfaces.CSFLevelEnum:
			if kvMap[d.confKeyMap[interfaces.CSFLevelEnum]] == "" {
				cfg.CSFLevelEnum = make(map[string]int)
				continue
			}

			err = jsoniter.Unmarshal([]byte(kvMap[d.confKeyMap[interfaces.CSFLevelEnum]]), &cfg.CSFLevelEnum)
			if err != nil {
				return cfg, fmt.Errorf("invalid csf level enum: %v", err)
			}
		case interfaces.CSFLevel2Enum:
			if kvMap[d.confKeyMap[interfaces.CSFLevel2Enum]] == "" {
				cfg.CSFLevel2Enum = make(map[string]int)
				continue
			}

			err = jsoniter.Unmarshal([]byte(kvMap[d.confKeyMap[interfaces.CSFLevel2Enum]]), &cfg.CSFLevel2Enum)
			if err != nil {
				return cfg, fmt.Errorf("invalid csf level2 enum: %v", err)
			}
		case interfaces.ShowCSFLevel2:
			var boolV bool
			value, ok := kvMap[d.confKeyMap[interfaces.ShowCSFLevel2]]
			if !ok {
				// 默认关闭显示密级2
				cfg.ShowCSFLevel2 = false
				continue
			}
			boolV, err = strconv.ParseBool(value)
			if err != nil {
				return cfg, fmt.Errorf("invalid show csf level2: %v", err)
			}
			cfg.ShowCSFLevel2 = boolV
		case interfaces.AutoDisable:
			// 此值可能是bool 也可能是int64
			var autoDisableConfig map[string]interface{}
			err = jsoniter.Unmarshal([]byte(kvMap[d.confKeyMap[interfaces.AutoDisable]]), &autoDisableConfig)
			if err != nil {
				return cfg, fmt.Errorf("invalid auto disable config: %v", err)
			}

			tempEnabled, ok := autoDisableConfig["isEnabled"].(float64)
			tempEnabledBool, okBool := autoDisableConfig["isEnabled"].(bool)
			tempTime, ok1 := autoDisableConfig["days"].(float64)
			if (!ok && !okBool) || !ok1 {
				return cfg, fmt.Errorf("invalid auto disable config: %v", kvMap[d.confKeyMap[interfaces.AutoDisable]])
			}

			if okBool {
				cfg.AutoDisableEnabled = tempEnabledBool
			} else {
				cfg.AutoDisableEnabled = tempEnabled != 0
			}
			cfg.AutoDisableTime = int64(tempTime)
		default:
			// 此项不应该被匹配，如果匹配到此项则代表遍历项存在杂项
			return cfg, errors.New("this error is unexpected")
		}
	}
	return
}

// SetConfig 设置认证配置
func (d *config) SetConfig(keys map[interfaces.ConfigKey]bool, cfg *interfaces.Config, tx *sql.Tx) (err error) {
	dbName := common.GetDBName("user_management")
	strSQL := fmt.Sprintf("update %s.option set `value` = ? where `key` = ? ", dbName)

	for key := range keys {
		var value interface{}
		switch key {
		case interfaces.UserDefaultNTLMPWD:
			value = cfg.UserDefaultNTLMPWD
		case interfaces.UserDefaultSha2PWD:
			value = cfg.UserDefaultSha2PWD
		case interfaces.UserDefaultDESPWD:
			value = cfg.UserDefaultDESPWD
		case interfaces.UserDefaultMd5PWD:
			value = cfg.UserDefaultMd5PWD
		case interfaces.UserDefaultPWD:
			continue
		default:
			// 此项不应该被匹配，如果匹配到此项则代表遍历项存在杂项
			return errors.New("this error is unexpected")
		}

		_, err = tx.Exec(strSQL, value, d.confKeyMap[key])
		if err != nil {
			d.logger.Errorln(err, strSQL)
			return
		}
	}

	return
}

// SetShareMgntConfig 设置认证配置 sharemgnt_db 数据库操作
func (d *config) SetShareMgntConfig(key interfaces.ConfigKey, cfg *interfaces.Config, tx *sql.Tx) (err error) {
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := fmt.Sprintf("update %s.t_sharemgnt_config set f_value = ? where f_key = ? ", dbName)
	strSQL2 := fmt.Sprintf("select f_value from %s.t_sharemgnt_config where f_key = ? ", dbName)
	strSQL3 := fmt.Sprintf("insert into %s.t_sharemgnt_config (f_key, f_value) values (?, ?) ", dbName)

	// 检查配置是否存在
	var value1 string
	var bHasKey bool
	err = tx.QueryRow(strSQL2, d.confKeyMap[key]).Scan(&value1)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// 记录不存在，后续执行插入操作
			bHasKey = false
		} else {
			d.logger.Errorln(err, strSQL2)
			return
		}
	} else {
		// 记录存在，后续执行更新操作
		bHasKey = true
	}

	// 更新配置
	var value interface{}
	switch key {
	case interfaces.CSFLevelEnum:
		value, err = jsoniter.MarshalToString(cfg.CSFLevelEnum)
		if err != nil {
			return
		}
	case interfaces.CSFLevel2Enum:
		value, err = jsoniter.MarshalToString(cfg.CSFLevel2Enum)
		if err != nil {
			return
		}
	default:
		// 此项不应该被匹配，如果匹配到此项则代表遍历项存在杂项
		return errors.New("this error is unexpected")
	}

	// 如果存在则更新，否则插入
	if bHasKey {
		_, err = tx.Exec(strSQL, value, d.confKeyMap[key])
	} else {
		_, err = tx.Exec(strSQL3, d.confKeyMap[key], value)
	}

	if err != nil {
		d.logger.Errorln(err, strSQL, strSQL3)
		return
	}

	return
}
