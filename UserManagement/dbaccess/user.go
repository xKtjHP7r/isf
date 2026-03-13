// Package dbaccess user Anyshare 数据访问层 - 用户数据库操作
package dbaccess

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/kweaver-ai/go-lib/observable"
	"github.com/kweaver-ai/proton-rds-sdk-go/sqlx"
	jsoniter "github.com/json-iterator/go"
	"github.com/oklog/ulid/v2"

	"UserManagement/common"
	"UserManagement/interfaces"
)

type user struct {
	db         *sqlx.DB
	logger     common.Logger
	trace      observable.Tracer
	dbTrace    *sqlx.DB
	lockKeyMap map[interfaces.DistributedLockType]string
}

// userDBData 用户数据库数据
type userDBData struct {
	ID                 string
	Name               string
	Account            string
	CSFLevel           int
	Priority           int
	DisableStatus      int
	AutoDisableStatus  int
	Email              string
	AuthType           int
	Password           string
	DesPassword        string
	NtlmPassword       string
	Sha2Password       string
	Frozen             int
	Authenticated      int
	TelNumber          string
	ThirdAttr          string
	ThirdID            string
	PWDControl         bool
	PWDTimeStamp       string
	PWDErrCnt          int
	PWDErrLatestTime   string
	LDAPType           int
	DomainPath         string
	OssID              string
	ManagerID          string
	Remark             string
	CreatedAtTimeStamp string
	Code               string
	Position           string
	CSFLevel2          int
}

var (
	uOnce sync.Once
	uDB   *user

	secondsPerDay = 24 * 60 * 60
)

// NewUser 创建数据库操作对象--和用户相关
func NewUser() *user {
	uOnce.Do(func() {
		uDB = &user{
			db:      dbPool,
			dbTrace: dbTracePool,
			logger:  common.NewLogger(),
			trace:   common.SvcARTrace,
			lockKeyMap: map[interfaces.DistributedLockType]string{
				interfaces.UserExpiredDisableLock:  "user_expired_disable_lock",
				interfaces.UserNotLoginDisableLock: "user_not_login_disable_lock",
			},
		}
	})

	return uDB
}

// GetUserName 批量获取用户显示名
func (u *user) GetUserName(userIDs []string) (info []interfaces.UserDBInfo, existIDs []string, err error) {
	// 分割ID 防止sql过长
	splitedIDs := SplitArray(userIDs)

	info = make([]interfaces.UserDBInfo, 0)
	existIDs = make([]string, 0)
	for _, ids := range splitedIDs {
		infoTmp, idTmp, tmpErr := u.getUserNameSingle(ids)
		if tmpErr != nil {
			return nil, nil, tmpErr
		}

		info = append(info, infoTmp...)
		existIDs = append(existIDs, idTmp...)
	}
	return info, existIDs, err
}

// getUserNameSingle 批量获取用户显示名, 显示名长度小于500
func (u *user) getUserNameSingle(userIDs []string) (info []interfaces.UserDBInfo, existIDs []string, err error) {
	if len(userIDs) == 0 {
		return nil, nil, nil
	}

	set, argIDs := GetFindInSetSQL(userIDs)
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "select f_user_id, f_display_name from %s.t_user where f_user_id in ("
	strSQL += set
	strSQL += ")"
	strSQL = fmt.Sprintf(strSQL, dbName)

	rows, sqlErr := u.db.Query(strSQL, argIDs...)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				u.logger.Errorln(rowsErr)
			}

			// 1、判断是否为空再关闭，2、如果不关闭而数据行并没有被scan的话，连接一直会被占用直到超时断开
			if closeErr := rows.Close(); closeErr != nil {
				u.logger.Errorln(closeErr)
			}
		}
	}()

	if sqlErr != nil {
		u.logger.Errorln(sqlErr, strSQL, argIDs)
		return nil, nil, sqlErr
	}

	tmpInfo := interfaces.UserDBInfo{}
	existIDs = make([]string, 0)
	info = make([]interfaces.UserDBInfo, 0)
	for rows.Next() {
		if scanErr := rows.Scan(&tmpInfo.ID, &tmpInfo.Name); scanErr != nil {
			u.logger.Errorln(scanErr, strSQL)
			return nil, nil, scanErr
		}

		existIDs = append(existIDs, tmpInfo.ID)
		info = append(info, tmpInfo)
	}

	return info, existIDs, nil
}

// GetDirectBelongDepartmentIDs 获取用户直属部门id
func (u *user) GetDirectBelongDepartmentIDs(userID string) ([]string, []interfaces.DepartmentDBInfo, error) {
	if userID == "" {
		return nil, nil, nil
	}
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "select f_department_id, f_path from %s.t_user_department_relation where f_user_id = ?"
	strSQL = fmt.Sprintf(strSQL, dbName)

	rows, sqlErr := u.db.Query(strSQL, userID)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				u.logger.Errorln(rowsErr)
			}

			// 1、判断是否为空再关闭，2、如果不关闭而数据行并没有被scan的话，连接一直会被占用直到超时断开
			if closeErr := rows.Close(); closeErr != nil {
				u.logger.Errorln(closeErr)
			}
		}
	}()

	if sqlErr != nil {
		u.logger.Errorln(sqlErr, strSQL)
		return nil, nil, sqlErr
	}

	deptIDs := make([]string, 0)
	deptInfos := make([]interfaces.DepartmentDBInfo, 0)
	var deptInfo interfaces.DepartmentDBInfo
	for rows.Next() {
		if err := rows.Scan(&deptInfo.ID, &deptInfo.Path); err != nil {
			u.logger.Errorln(err, strSQL)
			return nil, nil, err
		}

		// "-1"代表未分配用户，需要过滤
		if deptInfo.ID != "-1" {
			deptIDs = append(deptIDs, deptInfo.ID)
			deptInfos = append(deptInfos, deptInfo)
		}
	}

	return deptIDs, deptInfos, nil
}

// GetDirectBelongDepartmentIDs2 获取用户直属部门id
func (u *user) GetDirectBelongDepartmentIDs2(ctx context.Context, userID string) ([]string, []interfaces.DepartmentDBInfo, error) {
	// trace
	var err error
	u.trace.SetClientSpanName("数据库操作-获取用户直属部门id和路径")
	newCtx, span := u.trace.AddClientTrace(ctx)
	defer func() { u.trace.TelemetrySpanEnd(span, err) }()

	if userID == "" {
		return nil, nil, nil
	}
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "select f_department_id, f_path from %s.t_user_department_relation where f_user_id = ?"
	strSQL = fmt.Sprintf(strSQL, dbName)

	rows, sqlErr := u.dbTrace.QueryContext(newCtx, strSQL, userID)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				u.logger.Errorln(rowsErr)
			}

			// 1、判断是否为空再关闭，2、如果不关闭而数据行并没有被scan的话，连接一直会被占用直到超时断开
			if closeErr := rows.Close(); closeErr != nil {
				u.logger.Errorln(closeErr)
			}
		}
	}()

	if sqlErr != nil {
		u.logger.Errorln(sqlErr, strSQL)
		return nil, nil, sqlErr
	}

	deptIDs := make([]string, 0)
	deptInfos := make([]interfaces.DepartmentDBInfo, 0)
	var deptInfo interfaces.DepartmentDBInfo
	for rows.Next() {
		if err := rows.Scan(&deptInfo.ID, &deptInfo.Path); err != nil {
			u.logger.Errorln(err, strSQL)
			return nil, nil, err
		}

		// "-1"代表未分配用户，需要过滤
		if deptInfo.ID != "-1" {
			deptIDs = append(deptIDs, deptInfo.ID)
			deptInfos = append(deptInfos, deptInfo)
		}
	}

	return deptIDs, deptInfos, nil
}

// GetUsersInDepartments 获取在部门内部的用户ID
func (u *user) GetUsersInDepartments(userIDs, departmentIDs []string) ([]string, error) {
	ids := make([]string, 0)
	if len(userIDs) == 0 || len(departmentIDs) == 0 {
		return ids, nil
	}

	userSet, userArgIDs := GetFindInSetSQL(userIDs)
	depSet, depArgIDs := GetFindInSetSQL(departmentIDs)
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "select f_user_id from %s.t_user_department_relation where f_user_id in ( "
	strSQL += userSet
	strSQL += ") and f_department_id in ( "
	strSQL += depSet
	strSQL += ") "
	strSQL = fmt.Sprintf(strSQL, dbName)

	argIDs := make([]interface{}, 0)
	argIDs = append(argIDs, userArgIDs...)
	argIDs = append(argIDs, depArgIDs...)

	rows, sqlErr := u.db.Query(strSQL, argIDs...)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				u.logger.Errorln(rowsErr)
			}

			// 1、判断是否为空再关闭，2、如果不关闭而数据行并没有被scan的话，连接一直会被占用直到超时断开
			if closeErr := rows.Close(); closeErr != nil {
				u.logger.Errorln(closeErr)
			}
		}
	}()

	if sqlErr != nil {
		u.logger.Errorln(sqlErr, strSQL, argIDs)
		return ids, sqlErr
	}

	var temp string
	for rows.Next() {
		if scanErr := rows.Scan(&temp); scanErr != nil {
			u.logger.Errorln(scanErr, strSQL)
			return make([]string, 0), scanErr
		}

		ids = append(ids, temp)
	}

	return ids, nil
}

func (u *user) GetUserList(ctx context.Context, direction interfaces.Direction, bHasMarker bool, createdStamp int64, userID string, limit int) (out []interfaces.UserDBInfo, err error) {
	// trace
	u.trace.SetClientSpanName("数据库操作-获取用户列表")
	newCtx, span := u.trace.AddClientTrace(ctx)
	defer func() { u.trace.TelemetrySpanEnd(span, err) }()

	dbName := common.GetDBName("sharemgnt_db")
	args := make([]interface{}, 0)
	tempTime := time.Unix(createdStamp, 0).Local().Format("2006-01-02 15:04:05")
	strSQL := `select f_user_id, f_login_name, f_display_name, f_status, f_auto_disable_status, f_mail_address, IFNULL(f_tel_number,''),
	           f_create_time, f_freeze_status
				from %s.t_user 
				where f_user_id <> '266c6a42-6131-4d62-8f39-853e7093701c'  
				and f_user_id <> '94752844-BDD0-4B9E-8927-1CA8D427E699' 
				and f_user_id <> '234562BE-88FF-4440-9BFF-447F139871A2'
				and f_user_id <> '4bb41612-a040-11e6-887d-005056920bea'  `

	if bHasMarker && direction == interfaces.Desc {
		strSQL += "and (f_create_time < ? or (f_create_time = ? and f_user_id < ?)) "
		args = append(args, tempTime, tempTime, userID)
	} else if bHasMarker && direction == interfaces.Asc {
		strSQL += "and (f_create_time > ? or (f_create_time = ? and f_user_id > ?)) "
		args = append(args, tempTime, tempTime, userID)
	}

	if direction == interfaces.Desc {
		strSQL += " order by f_create_time desc , f_user_id desc limit ? "
		args = append(args, limit)
	} else {
		strSQL += " order by f_create_time asc , f_user_id asc limit ? "
		args = append(args, limit)
	}

	strSQL = fmt.Sprintf(strSQL, dbName)
	rows, sqlErr := u.dbTrace.QueryContext(newCtx, strSQL, args...)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				u.logger.Errorln(rowsErr)
			}

			// 1、判断是否为空再关闭，2、如果不关闭而数据行并没有被scan的话，连接一直会被占用直到超时断开
			if closeErr := rows.Close(); closeErr != nil {
				u.logger.Errorln(closeErr)
			}
		}
	}()

	if sqlErr != nil {
		u.logger.Errorln(sqlErr, strSQL)
		return out, sqlErr
	}

	out = make([]interfaces.UserDBInfo, 0)
	for rows.Next() {
		var temp userDBData
		if err := rows.Scan(&temp.ID, &temp.Account, &temp.Name, &temp.DisableStatus,
			&temp.AutoDisableStatus, &temp.Email, &temp.TelNumber, &temp.CreatedAtTimeStamp, &temp.Frozen); err != nil {
			u.logger.Errorln(err, strSQL)
			return out, err
		}
		out = append(out, handlerUserDBData(&temp))
	}

	return out, nil
}

// GetAllUserCount 获取所有用户数量
func (u *user) GetAllUserCount(ctx context.Context) (num int, err error) {
	// trace
	u.trace.SetClientSpanName("数据库操作-获取所有用户数量")
	newCtx, span := u.trace.AddClientTrace(ctx)
	defer func() { u.trace.TelemetrySpanEnd(span, err) }()

	dbName := common.GetDBName("sharemgnt_db")
	strSQL := `select count(*) from %s.t_user 
				where f_user_id <> '266c6a42-6131-4d62-8f39-853e7093701c' 
				and f_user_id <> '94752844-BDD0-4B9E-8927-1CA8D427E699'
				and f_user_id <> '234562BE-88FF-4440-9BFF-447F139871A2' 
				and f_user_id <> '4bb41612-a040-11e6-887d-005056920bea'`
	strSQL = fmt.Sprintf(strSQL, dbName)

	rows, sqlErr := u.dbTrace.QueryContext(newCtx, strSQL)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				u.logger.Errorln(rowsErr)
			}

			// 1、判断是否为空再关闭，2、如果不关闭而数据行并没有被scan的话，连接一直会被占用直到超时断开
			if closeErr := rows.Close(); closeErr != nil {
				u.logger.Errorln(closeErr)
			}
		}
	}()

	if sqlErr != nil {
		u.logger.Errorln(sqlErr, strSQL)
		return 0, sqlErr
	}

	for rows.Next() {
		if err := rows.Scan(&num); err != nil {
			u.logger.Errorln(err, strSQL)
			return num, err
		}
	}

	return num, nil
}

// GetUserDBInfo 获取用户基本数据库信息
func (u *user) GetUserDBInfo(userIDs []string) (info []interfaces.UserDBInfo, err error) {
	// 分割ID 防止sql过长
	splitedIDs := SplitArray(userIDs)

	info = make([]interfaces.UserDBInfo, 0)
	for _, ids := range splitedIDs {
		infoTmp, tmpErr := u.getUserDBInfoSingle(ids)
		if tmpErr != nil {
			return nil, tmpErr
		}

		info = append(info, infoTmp...)
	}
	return info, err
}

// getUserDBInfoSingle 获取用户基本数据库信息
func (u *user) getUserDBInfoSingle(userID []string) (out []interfaces.UserDBInfo, err error) {
	if len(userID) == 0 {
		return
	}

	userSet, userArgIDs := GetFindInSetSQL(userID)
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := `select f_user_id, f_login_name, f_display_name, f_priority, f_csf_level, f_status, f_auto_disable_status,
				f_mail_address, f_auth_type, f_freeze_status, f_real_name_auth_status, IFNULL(f_tel_number,''), f_third_party_attr, IFNULL(f_third_party_id,''),
				f_pwd_control, f_pwd_timestamp, f_password, f_sha2_password, IFNULL(f_oss_id,''), f_manager_id, f_create_time, f_csf_level2
				from %s.t_user where f_user_id in ( `
	strSQL += userSet
	strSQL += ")"
	strSQL = fmt.Sprintf(strSQL, dbName)

	rows, sqlErr := u.db.Query(strSQL, userArgIDs...)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				u.logger.Errorln(rowsErr)
			}

			// 1、判断是否为空再关闭，2、如果不关闭而数据行并没有被scan的话，连接一直会被占用直到超时断开
			if closeErr := rows.Close(); closeErr != nil {
				u.logger.Errorln(closeErr)
			}
		}
	}()

	if sqlErr != nil {
		u.logger.Errorln(sqlErr, strSQL)
		return out, sqlErr
	}

	out = make([]interfaces.UserDBInfo, 0)
	for rows.Next() {
		var temp userDBData
		if err := rows.Scan(&temp.ID, &temp.Account, &temp.Name, &temp.Priority, &temp.CSFLevel, &temp.DisableStatus,
			&temp.AutoDisableStatus, &temp.Email, &temp.AuthType, &temp.Frozen, &temp.Authenticated, &temp.TelNumber, &temp.ThirdAttr, &temp.ThirdID,
			&temp.PWDControl, &temp.PWDTimeStamp, &temp.Password, &temp.Sha2Password, &temp.OssID, &temp.ManagerID, &temp.CreatedAtTimeStamp, &temp.CSFLevel2); err != nil {
			u.logger.Errorln(err, strSQL)
			return out, err
		}
		out = append(out, handlerUserDBData(&temp))
	}

	return out, nil
}

// GetUserDBInfo 获取用户基本数据库信息 增加trace
func (u *user) GetUserDBInfo2(ctx context.Context, userID []string) (out []interfaces.UserDBInfo, err error) {
	// trace
	u.trace.SetClientSpanName("数据库操作-获取用户直属部门id和路径")
	newCtx, span := u.trace.AddClientTrace(ctx)
	defer func() { u.trace.TelemetrySpanEnd(span, err) }()
	if len(userID) == 0 {
		return
	}

	userSet, userArgIDs := GetFindInSetSQL(userID)
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := `select f_user_id, f_login_name, f_display_name, f_priority, f_csf_level, f_status, f_auto_disable_status,
				f_mail_address, f_auth_type, f_freeze_status, f_real_name_auth_status, IFNULL(f_tel_number,''), f_third_party_attr, IFNULL(f_third_party_id,''),
				f_pwd_control, f_pwd_timestamp, f_password, f_sha2_password, IFNULL(f_oss_id,'')
				from %s.t_user where f_user_id in ( `
	strSQL += userSet
	strSQL += ")"
	strSQL = fmt.Sprintf(strSQL, dbName)

	rows, sqlErr := u.dbTrace.QueryContext(newCtx, strSQL, userArgIDs...)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				u.logger.Errorln(rowsErr)
			}

			// 1、判断是否为空再关闭，2、如果不关闭而数据行并没有被scan的话，连接一直会被占用直到超时断开
			if closeErr := rows.Close(); closeErr != nil {
				u.logger.Errorln(closeErr)
			}
		}
	}()

	if sqlErr != nil {
		u.logger.Errorln(sqlErr, strSQL)
		return out, sqlErr
	}

	out = make([]interfaces.UserDBInfo, 0)
	for rows.Next() {
		var temp userDBData
		if err := rows.Scan(&temp.ID, &temp.Account, &temp.Name, &temp.Priority, &temp.CSFLevel, &temp.DisableStatus,
			&temp.AutoDisableStatus, &temp.Email, &temp.AuthType, &temp.Frozen, &temp.Authenticated, &temp.TelNumber, &temp.ThirdAttr, &temp.ThirdID,
			&temp.PWDControl, &temp.PWDTimeStamp, &temp.Password, &temp.Sha2Password, &temp.OssID); err != nil {
			u.logger.Errorln(err, strSQL)
			return out, err
		}
		out = append(out, handlerUserDBData(&temp))
	}

	return out, nil
}

// GetUserInfoByAccount 根据登录名获取用户信息
func (u *user) GetUserInfoByAccount(account string) (info interfaces.UserDBInfo, err error) {
	if account == "" {
		return
	}

	dbName := common.GetDBName("sharemgnt_db")
	strSQL := `select f_user_id, f_status, f_auto_disable_status,
				f_mail_address, f_auth_type, IFNULL(f_tel_number,'') , f_pwd_control,
				f_login_name, f_pwd_error_latest_timestamp, f_pwd_error_cnt, f_ldap_server_type, f_domain_path
				from %s.t_user
				where f_login_name = ?`
	strSQL = fmt.Sprintf(strSQL, dbName)
	rows, sqlErr := u.db.Query(strSQL, account)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				u.logger.Errorln(rowsErr)
			}

			// 1、判断是否为空再关闭，2、如果不关闭而数据行并没有被scan的话，连接一直会被占用直到超时断开
			if closeErr := rows.Close(); closeErr != nil {
				u.logger.Errorln(closeErr)
			}
		}
	}()

	if sqlErr != nil {
		u.logger.Errorln(sqlErr, strSQL)
		return info, sqlErr
	}

	for rows.Next() {
		var temp userDBData
		if err := rows.Scan(&temp.ID, &temp.DisableStatus, &temp.AutoDisableStatus,
			&temp.Email, &temp.AuthType, &temp.TelNumber, &temp.PWDControl,
			&temp.Account, &temp.PWDErrLatestTime, &temp.PWDErrCnt, &temp.LDAPType, &temp.DomainPath); err != nil {
			u.logger.Errorln(err, strSQL)
			return info, err
		}
		info = handlerUserDBData(&temp)
	}

	return info, nil
}

// GetDomainUserInfoByAccount 根据登录名获取域用户信息
func (u *user) GetDomainUserInfoByAccount(account string) (info interfaces.UserDBInfo, err error) {
	if account == "" {
		return
	}

	dbName := common.GetDBName("sharemgnt_db")
	strSQL := `select f_user_id, f_status, f_auto_disable_status,
				f_mail_address, f_auth_type, IFNULL(f_tel_number,'') , f_pwd_control,
				f_login_name, f_pwd_error_latest_timestamp, f_pwd_error_cnt, f_ldap_server_type, f_domain_path
				from %s.t_user
				where f_login_name like ? and f_auth_type = 2 order by f_login_name asc limit 1`
	strSQL = fmt.Sprintf(strSQL, dbName)
	rows, sqlErr := u.db.Query(strSQL, account+"@%%")
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				u.logger.Errorln(rowsErr)
			}

			// 1、判断是否为空再关闭，2、如果不关闭而数据行并没有被scan的话，连接一直会被占用直到超时断开
			if closeErr := rows.Close(); closeErr != nil {
				u.logger.Errorln(closeErr)
			}
		}
	}()

	if sqlErr != nil {
		u.logger.Errorln(sqlErr, strSQL)
		return info, sqlErr
	}

	for rows.Next() {
		var temp userDBData
		if err := rows.Scan(&temp.ID, &temp.DisableStatus, &temp.AutoDisableStatus,
			&temp.Email, &temp.AuthType, &temp.TelNumber, &temp.PWDControl,
			&temp.Account, &temp.PWDErrLatestTime, &temp.PWDErrCnt, &temp.LDAPType, &temp.DomainPath); err != nil {
			u.logger.Errorln(err, strSQL)
			return info, err
		}
		info = handlerUserDBData(&temp)
	}

	return info, nil
}

// GetUserInfoByIDCard 根据身份号获取用户信息
func (u *user) GetUserInfoByIDCard(id string) (info interfaces.UserDBInfo, err error) {
	if id == "" {
		return
	}

	dbName := common.GetDBName("sharemgnt_db")
	strSQL := `select f_user_id, f_status, f_auto_disable_status,
				f_mail_address, f_auth_type, IFNULL(f_tel_number,'') , f_pwd_control,
				f_login_name, f_pwd_error_latest_timestamp, f_pwd_error_cnt, f_ldap_server_type, f_domain_path
				from %s.t_user
				where f_idcard_number = ? `
	strSQL = fmt.Sprintf(strSQL, dbName)

	rows, sqlErr := u.db.Query(strSQL, id)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				u.logger.Errorln(rowsErr)
			}

			// 1、判断是否为空再关闭，2、如果不关闭而数据行并没有被scan的话，连接一直会被占用直到超时断开
			if closeErr := rows.Close(); closeErr != nil {
				u.logger.Errorln(closeErr)
			}
		}
	}()

	if sqlErr != nil {
		u.logger.Errorln(sqlErr, strSQL)
		return info, sqlErr
	}

	var temp userDBData
	for rows.Next() {
		if err := rows.Scan(&temp.ID, &temp.DisableStatus, &temp.AutoDisableStatus,
			&temp.Email, &temp.AuthType, &temp.TelNumber, &temp.PWDControl,
			&temp.Account, &temp.PWDErrLatestTime, &temp.PWDErrCnt, &temp.LDAPType, &temp.DomainPath); err != nil {
			u.logger.Errorln(err, strSQL)
			return info, err
		}
		info = handlerUserDBData(&temp)
	}

	return info, nil
}

// GetOrgAduitDepartInfo 获取组织审计员谁范围内部门
func (u *user) GetOrgAduitDepartInfo(userID string) (out []string, err error) {
	if userID == "" {
		return
	}
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "SELECT f_department_id FROM %s.t_department_audit_person WHERE f_user_id =  ? "
	strSQL = fmt.Sprintf(strSQL, dbName)

	rows, sqlErr := u.db.Query(strSQL, userID)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				u.logger.Errorln(rowsErr)
			}

			// 1、判断是否为空再关闭，2、如果不关闭而数据行并没有被scan的话，连接一直会被占用直到超时断开
			if closeErr := rows.Close(); closeErr != nil {
				u.logger.Errorln(closeErr)
			}
		}
	}()

	if sqlErr != nil {
		u.logger.Errorln(sqlErr, strSQL)
		return out, sqlErr
	}

	var temp string
	out = make([]string, 0)
	for rows.Next() {
		if err := rows.Scan(&temp); err != nil {
			u.logger.Errorln(err, strSQL)
			return out, err
		}
		out = append(out, temp)
	}

	return out, nil
}

// GetOrgAduitDepartInfo2 获取组织审计员谁范围内部门
func (u *user) GetOrgAduitDepartInfo2(ctx context.Context, userID string) (out []string, err error) {
	// trace
	u.trace.SetClientSpanName("数据库操作-获取组织审计员范围内部门ID")
	newCtx, span := u.trace.AddClientTrace(ctx)
	defer func() { u.trace.TelemetrySpanEnd(span, err) }()

	if userID == "" {
		return
	}
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "SELECT f_department_id FROM %s.t_department_audit_person WHERE f_user_id =  ? "
	strSQL = fmt.Sprintf(strSQL, dbName)

	rows, sqlErr := u.dbTrace.QueryContext(newCtx, strSQL, userID)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				u.logger.Errorln(rowsErr)
			}

			// 1、判断是否为空再关闭，2、如果不关闭而数据行并没有被scan的话，连接一直会被占用直到超时断开
			if closeErr := rows.Close(); closeErr != nil {
				u.logger.Errorln(closeErr)
			}
		}
	}()

	if sqlErr != nil {
		u.logger.Errorln(sqlErr, strSQL)
		return out, sqlErr
	}

	var temp string
	out = make([]string, 0)
	for rows.Next() {
		if err := rows.Scan(&temp); err != nil {
			u.logger.Errorln(err, strSQL)
			return out, err
		}
		out = append(out, temp)
	}

	return out, nil
}

// GetOrgManagerDepartInfo 获取组织管理员管理范围部门
func (u *user) GetOrgManagerDepartInfo(userID string) (out []string, err error) {
	if userID == "" {
		return
	}

	out = make([]string, 0)
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "SELECT f_department_id FROM %s.t_department_responsible_person WHERE f_user_id = ? "
	strSQL = fmt.Sprintf(strSQL, dbName)

	rows, sqlErr := u.db.Query(strSQL, userID)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				u.logger.Errorln(rowsErr)
			}

			// 1、判断是否为空再关闭，2、如果不关闭而数据行并没有被scan的话，连接一直会被占用直到超时断开
			if closeErr := rows.Close(); closeErr != nil {
				u.logger.Errorln(closeErr)
			}
		}
	}()

	if sqlErr != nil {
		u.logger.Errorln(sqlErr, strSQL)
		return out, sqlErr
	}

	for rows.Next() {
		var temp string
		if err := rows.Scan(&temp); err != nil {
			u.logger.Errorln(err, strSQL)
			return out, err
		}
		out = append(out, temp)
	}

	return out, nil
}

// GetOrgManagerDepartInfo2 获取组织管理员管理范围部门
func (u *user) GetOrgManagerDepartInfo2(ctx context.Context, userID string) (out []string, err error) {
	// trace
	u.trace.SetClientSpanName("数据库操作- 获取组织管理员管理范围部门ID")
	newCtx, span := u.trace.AddClientTrace(ctx)
	defer func() { u.trace.TelemetrySpanEnd(span, err) }()

	if userID == "" {
		return
	}

	out = make([]string, 0)
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "SELECT f_department_id FROM %s.t_department_responsible_person WHERE f_user_id = ? "
	strSQL = fmt.Sprintf(strSQL, dbName)

	rows, sqlErr := u.dbTrace.QueryContext(newCtx, strSQL, userID)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				u.logger.Errorln(rowsErr)
			}

			// 1、判断是否为空再关闭，2、如果不关闭而数据行并没有被scan的话，连接一直会被占用直到超时断开
			if closeErr := rows.Close(); closeErr != nil {
				u.logger.Errorln(closeErr)
			}
		}
	}()

	if sqlErr != nil {
		u.logger.Errorln(sqlErr, strSQL)
		return out, sqlErr
	}

	for rows.Next() {
		var temp string
		if err := rows.Scan(&temp); err != nil {
			u.logger.Errorln(err, strSQL)
			return out, err
		}
		out = append(out, temp)
	}

	return out, nil
}

// SearchOrgUsersByKey 在所有的用户中进行搜索
func (u *user) SearchOrgUsersByKey(ctx context.Context, bScope, bCount bool, keyword string, offset, limit int, onlyEnableUser,
	onlyAssignedUser bool, scope []string) (out []interfaces.UserDBInfo, err error) {
	// trace
	u.trace.SetClientSpanName("数据库操作-在管辖范围中中进行搜索相关用户的数据或者符合条件用户数量")
	newCtx, span := u.trace.AddClientTrace(ctx)
	defer func() { u.trace.TelemetrySpanEnd(span, err) }()

	var argIDs []interface{}
	argIDs = append(argIDs, "%"+keyword+"%", "%"+keyword+"%")
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := `select distinct a.f_user_id, a.f_display_name, a.f_login_name, a.f_priority from %s.t_user as a
				inner join %s.t_user_department_relation as b
				on b.f_user_id = a.f_user_id
				where (a.f_login_name like ? or a.f_display_name like ?) `
	strSQL = fmt.Sprintf(strSQL, dbName, dbName)

	if onlyEnableUser {
		strSQL += `and a.f_status = 0 and a.f_auto_disable_status = 0 `
	}

	if onlyAssignedUser {
		strSQL += `and b.f_department_id != '-1' `
	} else {
		strSQL += `and a.f_user_id not in ('266c6a42-6131-4d62-8f39-853e7093701c', '94752844-BDD0-4B9E-8927-1CA8D427E699',
		'4bb41612-a040-11e6-887d-005056920bea', '234562BE-88FF-4440-9BFF-447F139871A2') `
	}

	if !bScope {
		if len(scope) == 0 {
			return
		}
		depSQL := GetUUIDStringBySlice(scope)
		strSQL += ` and b.f_department_id in ( `
		strSQL += depSQL
		strSQL += `) `
	}

	if !bCount {
		strSQL += `order by a.f_priority, upper(a.f_display_name) limit ?, ?`
		argIDs = append(argIDs, offset, limit)
	}

	rows, sqlErr := u.dbTrace.QueryContext(newCtx, strSQL, argIDs...)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				u.logger.Errorln(rowsErr)
			}

			// 1、判断是否为空再关闭，2、如果不关闭而数据行并没有被scan的话，连接一直会被占用直到超时断开
			if closeErr := rows.Close(); closeErr != nil {
				u.logger.Errorln(closeErr)
			}
		}
	}()

	if sqlErr != nil {
		u.logger.Errorln(sqlErr, strSQL)
		return out, sqlErr
	}

	var temp interfaces.UserDBInfo
	for rows.Next() {
		if err := rows.Scan(&temp.ID, &temp.Name, &temp.Account, &temp.Priority); err != nil {
			u.logger.Errorln(err, strSQL)
			return out, err
		}
		out = append(out, temp)
	}

	return out, nil
}

// CheckNameExist 检查名字存在
func (u *user) CheckNameExist(name string) (exist bool, err error) {
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "select 1 from %s.t_user where f_login_name = ? limit 1"
	strSQL = fmt.Sprintf(strSQL, dbName)

	var tmp string
	err = u.db.QueryRow(strSQL, name).Scan(&tmp)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return
	}

	return true, nil
}

// ModifyUserInfo 修改用户信息
func (u *user) ModifyUserInfo(bRange interfaces.UserUpdateRange, info *interfaces.UserDBInfo, tx *sql.Tx) (err error) {
	var bUpdate bool
	var strSets string
	var arrays []interface{}

	now := common.Now().Local()
	strNow := now.Format("2006-01-02 15:04:05")
	if bRange.UpdatePWD {
		bUpdate = true
		strSets += "f_password = ?, f_sha2_password = ?, f_des_password = ?, f_ntlm_password = ?, f_pwd_timestamp = ?, f_pwd_error_latest_timestamp = ?, f_pwd_error_cnt = ? "
		arrays = append(arrays, "", info.Password, info.DesPassword, info.NtlmPassword, strNow, strNow, 0)
	}

	if !bUpdate {
		return
	}

	dbName := common.GetDBName("sharemgnt_db")
	sqlStr := "update %s.t_user set "
	sqlStr += strSets
	sqlStr += "WHERE f_user_id = ?"
	sqlStr = fmt.Sprintf(sqlStr, dbName)

	arrays = append(arrays, info.ID)
	if _, err := tx.Exec(sqlStr, arrays...); err != nil {
		return err
	}
	return nil
}

// GetUserInfoByName 根据显示名获取用户信息
func (u *user) GetUserInfoByName(ctx context.Context, name string) (info interfaces.UserDBInfo, err error) {
	// trace
	u.trace.SetClientSpanName("数据库操作-根据显示名获取用户信息")
	newCtx, span := u.trace.AddClientTrace(ctx)
	defer func() { u.trace.TelemetrySpanEnd(span, err) }()

	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "select f_user_id from %s.t_user where f_display_name = ? "
	strSQL = fmt.Sprintf(strSQL, dbName)
	rows, sqlErr := u.dbTrace.QueryContext(newCtx, strSQL, name)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				u.logger.Errorln(rowsErr)
			}

			// 1、判断是否为空再关闭，2、如果不关闭而数据行并没有被scan的话，连接一直会被占用直到超时断开
			if closeErr := rows.Close(); closeErr != nil {
				u.logger.Errorln(closeErr)
			}
		}
	}()

	if sqlErr != nil {
		u.logger.Errorln(sqlErr, strSQL, name)
		return info, sqlErr
	}

	for rows.Next() {
		if scanErr := rows.Scan(&info.ID); scanErr != nil {
			u.logger.Errorln(scanErr, strSQL)
			return info, scanErr
		}
	}

	return info, nil
}

// SearchUserInfoByName 根据显示名搜索用户信息
func (u *user) SearchUserInfoByName(ctx context.Context, name string) (infos []interfaces.UserDBInfo, err error) {
	// trace
	u.trace.SetClientSpanName("数据库操作-根据显示名搜索用户信息")
	newCtx, span := u.trace.AddClientTrace(ctx)
	defer func() { u.trace.TelemetrySpanEnd(span, err) }()

	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "select f_user_id, f_display_name from %s.t_user where f_display_name like ?  order by case when f_display_name = ? then 0 else 1 end, upper(f_display_name) "
	strSQL = fmt.Sprintf(strSQL, dbName)

	rows, sqlErr := u.dbTrace.QueryContext(newCtx, strSQL, "%"+name+"%", name)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				u.logger.Errorln(rowsErr)
			}

			// 1、判断是否为空再关闭，2、如果不关闭而数据行并没有被scan的话，连接一直会被占用直到超时断开
			if closeErr := rows.Close(); closeErr != nil {
				u.logger.Errorln(closeErr)
			}
		}
	}()

	if sqlErr != nil {
		u.logger.Errorln(sqlErr, strSQL, name)
		return infos, sqlErr
	}

	infos = make([]interfaces.UserDBInfo, 0)
	for rows.Next() {
		var info interfaces.UserDBInfo
		if scanErr := rows.Scan(&info.ID, &info.Name); scanErr != nil {
			u.logger.Errorln(scanErr, strSQL)
			return infos, scanErr
		}

		infos = append(infos, info)
	}

	return infos, nil
}

// GetUsersPath2 获取用户所属路径 支持trace
func (u *user) GetUsersPath2(ctx context.Context, ids []string) (paths map[string][]string, err error) {
	// trace
	u.trace.SetClientSpanName("数据库操作-获取用户所属路径")
	newCtx, span := u.trace.AddClientTrace(ctx)
	defer func() { u.trace.TelemetrySpanEnd(span, err) }()

	// 分割ID 防止sql过长
	splitedIDs := SplitArray(ids)

	paths = make(map[string][]string)
	for _, v := range splitedIDs {
		infoTmp, tmpErr := u.getUsersPath2(newCtx, v)
		if tmpErr != nil {
			return nil, tmpErr
		}

		for k, v := range infoTmp {
			paths[k] = v
		}
	}
	return paths, err
}

// getUsersPath2 获取用户所属路径 支持trace
func (u *user) getUsersPath2(ctx context.Context, ids []string) (paths map[string][]string, err error) {
	// 检查是否为空
	paths = make(map[string][]string)
	if len(ids) == 0 {
		return paths, nil
	}

	set, argIDs := GetFindInSetSQL(ids)
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "select f_user_id, f_path from %s.t_user_department_relation where f_user_id in ("
	strSQL += set
	strSQL += ")"
	strSQL = fmt.Sprintf(strSQL, dbName)
	rows, sqlErr := u.dbTrace.QueryContext(ctx, strSQL, argIDs...)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				u.logger.Errorln(rowsErr)
			}

			// 1、判断是否为空再关闭，2、如果不关闭而数据行并没有被scan的话，连接一直会被占用直到超时断开
			if closeErr := rows.Close(); closeErr != nil {
				u.logger.Errorln(closeErr)
			}
		}
	}()

	if sqlErr != nil {
		u.logger.Errorln(sqlErr, strSQL, ids)
		return nil, sqlErr
	}

	var path string
	var userID string
	for rows.Next() {
		if scanErr := rows.Scan(&userID, &path); scanErr != nil {
			u.logger.Errorln(scanErr, strSQL)
			return nil, scanErr
		}

		if _, ok := paths[userID]; !ok {
			paths[userID] = make([]string, 0)
		}

		paths[userID] = append(paths[userID], path)
	}

	return paths, nil
}

// GetUsersPath 获取用户所属路径
func (u *user) GetUsersPath(ids []string) (paths map[string][]string, err error) {
	// 检查是否为空
	paths = make(map[string][]string)
	if len(ids) == 0 {
		return paths, nil
	}

	set, argIDs := GetFindInSetSQL(ids)
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "select f_user_id, f_path from %s.t_user_department_relation where f_user_id in ("
	strSQL += set
	strSQL += ")"
	strSQL = fmt.Sprintf(strSQL, dbName)

	rows, sqlErr := u.db.Query(strSQL, argIDs...)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				u.logger.Errorln(rowsErr)
			}

			// 1、判断是否为空再关闭，2、如果不关闭而数据行并没有被scan的话，连接一直会被占用直到超时断开
			if closeErr := rows.Close(); closeErr != nil {
				u.logger.Errorln(closeErr)
			}
		}
	}()

	if sqlErr != nil {
		u.logger.Errorln(sqlErr, strSQL, ids)
		return nil, sqlErr
	}

	var path string
	var userID string
	for rows.Next() {
		if scanErr := rows.Scan(&userID, &path); scanErr != nil {
			u.logger.Errorln(scanErr, strSQL)
			return nil, scanErr
		}

		if _, ok := paths[userID]; !ok {
			paths[userID] = make([]string, 0)
		}

		paths[userID] = append(paths[userID], path)
	}

	return paths, nil
}

// UpdatePwdErrInfo 更新账户密码错误信息
func (u *user) UpdatePwdErrInfo(id string, pwdErrCnt int, pwdErrLastTime int64) (err error) {
	var keySets string
	var valueSets []interface{}

	keySets += "f_pwd_error_latest_timestamp = ?, f_pwd_error_cnt = ? "
	valueSets = append(valueSets, time.Unix(pwdErrLastTime, 0).Format(time.DateTime), pwdErrCnt)

	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "update %s.t_user set "
	strSQL += keySets
	strSQL += "WHERE f_user_id = ?"
	strSQL = fmt.Sprintf(strSQL, dbName)

	valueSets = append(valueSets, id)
	if _, err := u.db.Exec(strSQL, valueSets...); err != nil {
		return err
	}

	return nil
}

// GetOrgManagersDepartInfo 获取组织管理员管理范围部门
func (u *user) GetOrgManagersDepartInfo(userIDs []string) (out map[string][]string, err error) {
	out = make(map[string][]string, 0)
	if len(userIDs) == 0 {
		return
	}

	set, argIDs := GetFindInSetSQL(userIDs)
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "SELECT f_user_id, f_department_id FROM %s.t_department_responsible_person WHERE f_user_id in ( "
	strSQL += set
	strSQL += ")"
	strSQL = fmt.Sprintf(strSQL, dbName)

	rows, sqlErr := u.db.Query(strSQL, argIDs...)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				u.logger.Errorln(rowsErr)
			}

			// 1、判断是否为空再关闭，2、如果不关闭而数据行并没有被scan的话，连接一直会被占用直到超时断开
			if closeErr := rows.Close(); closeErr != nil {
				u.logger.Errorln(closeErr)
			}
		}
	}()

	if sqlErr != nil {
		u.logger.Errorln(sqlErr, strSQL)
		return out, sqlErr
	}

	var tempUsreID string
	var tempDepartID string
	for rows.Next() {
		if err := rows.Scan(&tempUsreID, &tempDepartID); err != nil {
			u.logger.Errorln(err, strSQL)
			return out, err
		}

		if _, ok := out[tempUsreID]; !ok {
			out[tempUsreID] = make([]string, 0)
		}
		out[tempUsreID] = append(out[tempUsreID], tempDepartID)
	}

	return out, nil
}

// GetUserCustomAttr 获取用户自定义属性
func (u *user) GetUserCustomAttr(userID string) (out map[string]interface{}, err error) {
	if userID == "" {
		return
	}

	out = make(map[string]interface{}, 0)
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "SELECT f_custom_attr FROM %s.t_user_custom_attr WHERE f_user_id = ? "
	strSQL = fmt.Sprintf(strSQL, dbName)

	rows, sqlErr := u.db.Query(strSQL, userID)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				u.logger.Errorln(rowsErr)
			}

			// 1、判断是否为空再关闭，2、如果不关闭而数据行并没有被scan的话，连接一直会被占用直到超时断开
			if closeErr := rows.Close(); closeErr != nil {
				u.logger.Errorln(closeErr)
			}
		}
	}()

	if sqlErr != nil {
		u.logger.Errorln(sqlErr, strSQL)
		return out, sqlErr
	}

	for rows.Next() {
		var temp []byte
		if err := rows.Scan(&temp); err != nil {
			u.logger.Errorln(err, strSQL)
			return out, err
		}
		if temp == nil {
			return out, nil
		}
		err := jsoniter.Unmarshal(temp, &out)
		if err != nil {
			u.logger.Errorln(err, temp)
			return out, err
		}
	}

	return out, nil
}

// UpdateUserCustomAttr 更新用户自定义属性
func (u *user) UpdateUserCustomAttr(ctx context.Context, userID string, customAttr map[string]interface{}, tx *sql.Tx) (err error) {
	// trace
	u.trace.SetClientSpanName("数据库操作-更新用户自定义属性")
	newCtx, span := u.trace.AddClientTrace(ctx)
	defer func() { u.trace.TelemetrySpanEnd(span, err) }()
	if userID == "" {
		return
	}
	var args []any
	var customAttrStr string
	customAttrStr, err = jsoniter.MarshalToString(customAttr)
	if err != nil {
		u.logger.Errorln(err, customAttr)
		return err
	}
	args = append(args, customAttrStr, userID)
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "update %s.t_user_custom_attr set f_custom_attr = ? WHERE f_user_id = ? "
	strSQL = fmt.Sprintf(strSQL, dbName)

	if _, err := tx.ExecContext(newCtx, strSQL, args...); err != nil {
		return err
	}
	return nil
}

// AddUserCustomAttr 添加用户自定义属性
func (u *user) AddUserCustomAttr(ctx context.Context, userID string, customAttr map[string]interface{}, tx *sql.Tx) (err error) {
	// trace
	u.trace.SetClientSpanName("数据库操作-添加用户自定义属性")
	newCtx, span := u.trace.AddClientTrace(ctx)
	defer func() { u.trace.TelemetrySpanEnd(span, err) }()
	if userID == "" {
		return
	}
	var args []any
	var customAttrStr, ID string
	ID = ulid.Make().String()
	customAttrStr, err = jsoniter.MarshalToString(customAttr)
	if err != nil {
		u.logger.Errorln(err, customAttr)
		return err
	}
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "insert into %s.t_user_custom_attr (f_id, f_user_id, f_custom_attr) values (?, ?, ?)"
	strSQL = fmt.Sprintf(strSQL, dbName)
	args = append(args, ID, userID, customAttrStr)
	if _, err := tx.ExecContext(newCtx, strSQL, args...); err != nil {
		return err
	}
	return nil
}

// GetUserDBInfoByTels 根据手机号获取用户基本数据库信息
func (u *user) GetUserDBInfoByTels(ctx context.Context, tels []string) (out []interfaces.UserDBInfo, err error) {
	// trace
	u.trace.SetClientSpanName("数据库操作-根据用户手机号获取用户信息")
	newCtx, span := u.trace.AddClientTrace(ctx)
	defer func() { u.trace.TelemetrySpanEnd(span, err) }()

	// 分割ID 防止sql过长
	splitedIDs := SplitArray(tels)

	out = make([]interfaces.UserDBInfo, 0)
	for _, ids := range splitedIDs {
		infoTmp, tmpErr := u.getUserDBInfoByTelsSingle(newCtx, ids)
		if tmpErr != nil {
			return nil, tmpErr
		}

		out = append(out, infoTmp...)
	}
	return out, err
}

// getUserDBInfoByTelsSingle 根据手机号获取用户基本数据库信息
func (u *user) getUserDBInfoByTelsSingle(ctx context.Context, tels []string) (out []interfaces.UserDBInfo, err error) {
	if len(tels) == 0 {
		return
	}

	userSet, userArgIDs := GetFindInSetSQL(tels)
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := `select f_user_id, f_login_name, f_display_name, f_mail_address, f_tel_number, IFNULL(f_third_party_id,'')
			from %s.t_user where f_tel_number in ( `
	strSQL += userSet
	strSQL += ")"
	strSQL = fmt.Sprintf(strSQL, dbName)

	rows, sqlErr := u.db.QueryContext(ctx, strSQL, userArgIDs...)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				u.logger.Errorln(rowsErr)
			}

			// 1、判断是否为空再关闭，2、如果不关闭而数据行并没有被scan的话，连接一直会被占用直到超时断开
			if closeErr := rows.Close(); closeErr != nil {
				u.logger.Errorln(closeErr)
			}
		}
	}()

	if sqlErr != nil {
		u.logger.Errorln(sqlErr, strSQL)
		return out, sqlErr
	}

	out = make([]interfaces.UserDBInfo, 0)
	for rows.Next() {
		var temp userDBData
		if err := rows.Scan(&temp.ID, &temp.Account, &temp.Name, &temp.Email, &temp.TelNumber, &temp.ThirdID); err != nil {
			u.logger.Errorln(err, strSQL)
			return out, err
		}
		out = append(out, handlerUserDBData(&temp))
	}

	return out, nil
}

// DeleteUserManagerID 删除用户上级信息
func (u *user) DeleteUserManagerID(id string) (err error) {
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "update %s.t_user set f_manager_id = '' where f_manager_id = ?"
	strSQL = fmt.Sprintf(strSQL, dbName)
	_, err = u.db.Exec(strSQL, id)
	return
}

// SearchUsers 搜索用户
func (u *user) SearchUsers(ctx context.Context, ks *interfaces.UserSearchInDepartKeyScope, k *interfaces.UserSearchInDepartKey) (
	out []interfaces.UserDBInfo, err error) {
	// trace
	u.trace.SetClientSpanName("数据库操作-搜索用户")
	newCtx, span := u.trace.AddClientTrace(ctx)
	defer func() { u.trace.TelemetrySpanEnd(span, err) }()

	argParms := make([]interface{}, 0)
	strParams := make([]interface{}, 0)
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := `select distinct a.f_user_id, a.f_login_name, a.f_display_name, IFNULL(a.f_remark, ''), a.f_csf_level, a.f_auth_type, a.f_priority, a.f_create_time, 
	        a.f_status, a.f_auto_disable_status, a.f_code, a.f_manager_id, a.f_position, a.f_freeze_status, a.f_csf_level2
			from %s.t_user as a`

	strParams = append(strParams, dbName)
	if ks.BDepartmentID {
		if k.DepartmentID == "-1" {
			strSQL += ` inner join %s.t_user_department_relation as b on a.f_user_id = b.f_user_id and b.f_department_id = '-1' `
			strParams = append(strParams, dbName)
		} else {
			strSQL += ` inner join %s.t_user_department_relation as b on a.f_user_id = b.f_user_id and b.f_path like ? `
			argParms = append(argParms, "%"+k.DepartmentID+"%")
			strParams = append(strParams, dbName)
		}
	}

	bHasKey := false
	if ks.BCode {
		bHasKey = true
		strSQL += ` where a.f_code like ? `
		argParms = append(argParms, "%"+k.Code+"%")
	} else if ks.BName {
		bHasKey = true
		strSQL += ` where a.f_display_name like ? `
		argParms = append(argParms, "%"+k.Name+"%")
	} else if ks.BAccount {
		strSQL += ` where a.f_login_name like ? `
		argParms = append(argParms, "%"+k.Account+"%")
		bHasKey = true
	} else if ks.BPosition {
		strSQL += ` where a.f_position like ? `
		argParms = append(argParms, "%"+k.Position+"%")
		bHasKey = true
	} else if ks.BManagerName {
		strSQL += ` inner join %s.t_user as m on a.f_manager_id = m.f_user_id and m.f_display_name LIKE ? `
		argParms = append(argParms, "%"+k.ManagerName+"%")
		strParams = append(strParams, dbName)
	} else if ks.BDirectDepartCode {
		strSQL += ` where a.f_user_id in (select f_user_id from %s.t_user_department_relation where f_department_id in (select f_department_id from %s.t_department where f_code like ?))`
		argParms = append(argParms, "%"+k.DirectDepartCode+"%")
		strParams = append(strParams, dbName, dbName)
		bHasKey = true
	}

	if bHasKey {
		strSQL += sqlAnd
	} else {
		strSQL += sqlWhere
	}

	strSQL += ` a.f_user_id not in ('266c6a42-6131-4d62-8f39-853e7093701c', '94752844-BDD0-4B9E-8927-1CA8D427E699',
		'4bb41612-a040-11e6-887d-005056920bea', '234562BE-88FF-4440-9BFF-447F139871A2') `

	strSQL += " order by a.f_priority, a.f_display_name limit ?, ? "
	argParms = append(argParms, k.Offset, k.Limit)
	strSQL = fmt.Sprintf(strSQL, strParams...)

	rows, sqlErr := u.db.QueryContext(newCtx, strSQL, argParms...)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				u.logger.Errorln(rowsErr)
			}

			// 1、判断是否为空再关闭，2、如果不关闭而数据行并没有被scan的话，连接一直会被占用直到超时断开
			if closeErr := rows.Close(); closeErr != nil {
				u.logger.Errorln(closeErr)
			}
		}
	}()

	if sqlErr != nil {
		u.logger.Errorln(sqlErr, strSQL)
		return out, sqlErr
	}

	out = make([]interfaces.UserDBInfo, 0)
	for rows.Next() {
		var temp userDBData
		if err := rows.Scan(&temp.ID, &temp.Account, &temp.Name, &temp.Remark, &temp.CSFLevel, &temp.AuthType, &temp.Priority,
			&temp.CreatedAtTimeStamp, &temp.DisableStatus, &temp.AutoDisableStatus, &temp.Code, &temp.ManagerID, &temp.Position, &temp.Frozen, &temp.CSFLevel2); err != nil {
			u.logger.Errorln(err, strSQL)
			return out, err
		}
		out = append(out, handlerUserDBData(&temp))
	}

	return out, nil
}

// SearchUsersCount 搜索用户数量
func (u *user) SearchUsersCount(ctx context.Context, ks *interfaces.UserSearchInDepartKeyScope, k *interfaces.UserSearchInDepartKey) (
	num int, err error) {
	// trace
	u.trace.SetClientSpanName("数据库操作-根据用户手机号获取用户信息")
	newCtx, span := u.trace.AddClientTrace(ctx)
	defer func() { u.trace.TelemetrySpanEnd(span, err) }()

	argParms := make([]interface{}, 0)
	strParams := make([]interface{}, 0)
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := `select count(*) from %s.t_user as a`

	strParams = append(strParams, dbName)
	if ks.BDepartmentID {
		if k.DepartmentID == "-1" {
			strSQL += ` inner join %s.t_user_department_relation as b on a.f_user_id = b.f_user_id and b.f_department_id = '-1' `
			strParams = append(strParams, dbName)
		} else {
			strSQL += ` inner join %s.t_user_department_relation as b on a.f_user_id = b.f_user_id and b.f_path like ? `
			argParms = append(argParms, "%"+k.DepartmentID+"%")
			strParams = append(strParams, dbName)
		}
	}

	bHasKey := false
	if ks.BCode {
		strSQL += ` where a.f_code like ? `
		argParms = append(argParms, "%"+k.Code+"%")
		bHasKey = true
	} else if ks.BName {
		strSQL += ` where a.f_display_name like ? `
		argParms = append(argParms, "%"+k.Name+"%")
		bHasKey = true
	} else if ks.BAccount {
		strSQL += ` where a.f_login_name like ? `
		argParms = append(argParms, "%"+k.Account+"%")
		bHasKey = true
	} else if ks.BPosition {
		strSQL += ` where a.f_position like ? `
		argParms = append(argParms, "%"+k.Position+"%")
		bHasKey = true
	} else if ks.BManagerName {
		strSQL += ` inner join %s.t_user as m on a.f_manager_id = m.f_user_id and m.f_display_name LIKE ? `
		argParms = append(argParms, "%"+k.ManagerName+"%")
		strParams = append(strParams, dbName)
	} else if ks.BDirectDepartCode {
		strSQL += ` where a.f_user_id in (select f_user_id from %s.t_user_department_relation where f_department_id in (select f_department_id from %s.t_department where f_code like ?))`
		argParms = append(argParms, "%"+k.DirectDepartCode+"%")
		strParams = append(strParams, dbName, dbName)
		bHasKey = true
	}

	if bHasKey {
		strSQL += sqlAnd
	} else {
		strSQL += sqlWhere
	}

	strSQL += ` a.f_user_id not in ('266c6a42-6131-4d62-8f39-853e7093701c', '94752844-BDD0-4B9E-8927-1CA8D427E699',
		'4bb41612-a040-11e6-887d-005056920bea', '234562BE-88FF-4440-9BFF-447F139871A2') `

	strSQL = fmt.Sprintf(strSQL, strParams...)
	rows, sqlErr := u.db.QueryContext(newCtx, strSQL, argParms...)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				u.logger.Errorln(rowsErr)
			}

			// 1、判断是否为空再关闭，2、如果不关闭而数据行并没有被scan的话，连接一直会被占用直到超时断开
			if closeErr := rows.Close(); closeErr != nil {
				u.logger.Errorln(closeErr)
			}
		}
	}()

	if sqlErr != nil {
		u.logger.Errorln(sqlErr, strSQL)
		return num, sqlErr
	}

	for rows.Next() {
		if err := rows.Scan(&num); err != nil {
			u.logger.Errorln(err, strSQL)
			return num, err
		}
	}

	return num, nil
}

// GetLock 获取锁
func (u *user) GetLock(ctx context.Context, lockType interfaces.DistributedLockType, tx *sql.Tx) (err error) {
	// trace
	u.trace.SetClientSpanName("数据库操作-获取锁")
	newCtx, span := u.trace.AddClientTrace(ctx)
	defer func() { u.trace.TelemetrySpanEnd(span, err) }()

	dbName := common.GetDBName("sharemgnt_db")
	sqlStr := "select f_value from %s.t_sharemgnt_config where f_key = '%s' for update"
	sqlStr = fmt.Sprintf(sqlStr, dbName, u.lockKeyMap[lockType])

	rows, sqlErr := tx.QueryContext(newCtx, sqlStr)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				u.logger.Errorln(rowsErr)
			}

			// 1、判断是否为空再关闭，2、如果不关闭而数据行并没有被scan的话，连接一直会被占用直到超时断开
			if closeErr := rows.Close(); closeErr != nil {
				u.logger.Errorln(closeErr)
			}
		}
	}()

	if sqlErr != nil {
		u.logger.Errorln(sqlErr, sqlStr)
		return sqlErr
	}

	return err
}

// GetExpiredNeedDisableUserInfos 获取需过期要禁用的用户信息
func (u *user) GetExpiredNeedDisableUserInfos(ctx context.Context, tx *sql.Tx) (userInfos []interfaces.UserDBInfo, err error) {
	// trace
	u.trace.SetClientSpanName("数据库操作-获取过期需要禁用的用户信息")
	newCtx, span := u.trace.AddClientTrace(ctx)
	defer func() { u.trace.TelemetrySpanEnd(span, err) }()

	dbName := common.GetDBName("sharemgnt_db")
	sqlStr := `select f_user_id, f_display_name, f_login_name from %s.t_user 
	    		where f_expire_time <= ? 
				and f_expire_time != -1 
				and f_auto_disable_status & %d = 0
				and f_user_id != ?
            	and f_user_id != ?
            	and f_user_id != ?
            	and f_user_id != ?`
	sqlStr = fmt.Sprintf(sqlStr, dbName, interfaces.ExpiredDisabled)

	// f_expire_time 单位为秒
	now := common.Now()
	rows, sqlErr := tx.QueryContext(newCtx, sqlStr, now.Unix(), interfaces.SystemSysAdmin, interfaces.SystemAuditAdmin,
		interfaces.SystemSecAdmin, interfaces.SystemOriginSysAdmin)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				u.logger.Errorln(rowsErr)
			}

			// 1、判断是否为空再关闭，2、如果不关闭而数据行并没有被scan的话，连接一直会被占用直到超时断开
			if closeErr := rows.Close(); closeErr != nil {
				u.logger.Errorln(closeErr)
			}
		}
	}()

	if sqlErr != nil {
		u.logger.Errorln(sqlErr, sqlStr)
		return userInfos, sqlErr
	}

	userInfos = make([]interfaces.UserDBInfo, 0)
	for rows.Next() {
		var temp userDBData
		if err = rows.Scan(&temp.ID, &temp.Name, &temp.Account); err != nil {
			u.logger.Errorln(err, sqlStr)
			return userInfos, err
		}
		userInfos = append(userInfos, handlerUserDBData(&temp))
	}
	return userInfos, err
}

// SetAutoDisableStatus 设置用户自动禁用状态
func (u *user) SetAutoDisableStatus(ctx context.Context, userIDs []string, disableStatus interfaces.DisableStatus, tx *sql.Tx) (err error) {
	// trace
	u.trace.SetClientSpanName("数据库操作-设置用户自动禁用状态")
	newCtx, span := u.trace.AddClientTrace(ctx)
	defer func() { u.trace.TelemetrySpanEnd(span, err) }()

	// 批量设置用户自动禁用状态， 每500个用户设置一次
	for i := 0; i < len(userIDs); i += 500 {
		end := i + 500
		if end > len(userIDs) {
			end = len(userIDs)
		}
		err = u.setAutoDisableStatusSingle(newCtx, userIDs[i:end], disableStatus, tx)
		if err != nil {
			return err
		}
	}
	return nil
}

// SetAutoDisableStatus 设置用户自动禁用状态
func (u *user) setAutoDisableStatusSingle(ctx context.Context, userIDs []string, disableStatus interfaces.DisableStatus, tx *sql.Tx) (err error) {
	if len(userIDs) == 0 {
		return nil
	}

	userSet, userArgIDs := GetFindInSetSQL(userIDs)

	dbName := common.GetDBName("sharemgnt_db")
	sqlStr := "UPDATE %s.t_user SET f_auto_disable_status = f_auto_disable_status | %d WHERE f_user_id in ( "
	sqlStr += userSet
	sqlStr += " )"
	sqlStr = fmt.Sprintf(sqlStr, dbName, disableStatus)
	_, err = tx.ExecContext(ctx, sqlStr, userArgIDs...)
	if err != nil {
		u.logger.Errorln(err, sqlStr)
		return err
	}
	return nil
}

// GetNotLoginNeedDisableUserInfos 获取长时间未登录要禁用的用户信息
func (u *user) GetNotLoginNeedDisableUserInfos(ctx context.Context, allowDays int64, tx *sql.Tx) (userInfos []interfaces.UserDBInfo, err error) {
	// trace
	u.trace.SetClientSpanName("数据库操作-获取长时间未登录要禁用的用户信息")
	newCtx, span := u.trace.AddClientTrace(ctx)
	defer func() { u.trace.TelemetrySpanEnd(span, err) }()

	dbName := common.GetDBName("sharemgnt_db")
	sqlStr := `select f_user_id, f_display_name, f_login_name from %s.t_user 
	    		where f_last_request_time <= ? 
				and f_auto_disable_status & %d = 0
				and f_user_id != ?
            	and f_user_id != ?
            	and f_user_id != ?
            	and f_user_id != ?`
	sqlStr = fmt.Sprintf(sqlStr, dbName, interfaces.NotLoginDisabled)

	// f_expire_time 单位为秒
	delayTime := common.Now().Add(-time.Duration(allowDays) * time.Second * time.Duration(secondsPerDay))
	rows, sqlErr := tx.QueryContext(newCtx, sqlStr, delayTime.Format("2006-01-02 15:04:05"), interfaces.SystemSysAdmin, interfaces.SystemAuditAdmin,
		interfaces.SystemSecAdmin, interfaces.SystemOriginSysAdmin)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				u.logger.Errorln(rowsErr)
			}

			// 1、判断是否为空再关闭，2、如果不关闭而数据行并没有被scan的话，连接一直会被占用直到超时断开
			if closeErr := rows.Close(); closeErr != nil {
				u.logger.Errorln(closeErr)
			}
		}
	}()

	if sqlErr != nil {
		u.logger.Errorln(sqlErr, sqlStr)
		return userInfos, sqlErr
	}

	userInfos = make([]interfaces.UserDBInfo, 0)
	for rows.Next() {
		var temp userDBData
		if err = rows.Scan(&temp.ID, &temp.Name, &temp.Account); err != nil {
			u.logger.Errorln(err, sqlStr)
			return userInfos, err
		}
		userInfos = append(userInfos, handlerUserDBData(&temp))
	}
	return userInfos, err
}
