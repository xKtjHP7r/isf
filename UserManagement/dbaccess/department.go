// Package dbaccess department Anyshare 数据访问层 - 部门数据库操作
package dbaccess

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"

	"github.com/kweaver-ai/go-lib/observable"
	"github.com/kweaver-ai/proton-rds-sdk-go/sqlx"

	"UserManagement/common"
	"UserManagement/interfaces"
)

type department struct {
	db      *sqlx.DB
	logger  common.Logger
	trace   observable.Tracer
	dbTrace *sqlx.DB
}

var (
	dOnce sync.Once
	dDB   *department
)

const (
	sqlWhere = " where "
	sqlAnd   = " and "
)

// NewDepartment 创建数据库操作对象
func NewDepartment() *department {
	dOnce.Do(func() {
		dDB = &department{
			db:      dbPool,
			logger:  common.NewLogger(),
			trace:   common.SvcARTrace,
			dbTrace: dbTracePool,
		}
	})

	return dDB
}

// GetDepartmentName
func (d *department) GetDepartmentName(deptIDs []string) (info []interfaces.NameInfo, existIDs []string, err error) {
	// 分割ID 防止sql过长
	splitedIDs := SplitArray(deptIDs)

	info = make([]interfaces.NameInfo, 0)
	existIDs = make([]string, 0)
	for _, ids := range splitedIDs {
		infoTmp, idTmp, tmpErr := d.getDepartmentNameSingle(ids)
		if tmpErr != nil {
			return nil, nil, tmpErr
		}

		info = append(info, infoTmp...)
		existIDs = append(existIDs, idTmp...)
	}
	return info, existIDs, err
}

// GetDepartmentName
func (d *department) getDepartmentNameSingle(deptIDs []string) (info []interfaces.NameInfo, existIDs []string, err error) {
	if len(deptIDs) == 0 {
		return nil, nil, nil
	}

	set, argIDs := GetFindInSetSQL(deptIDs)
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "select f_department_id, f_name from %s.t_department where f_department_id in ("
	strSQL += set
	strSQL += ")"
	strSQL = fmt.Sprintf(strSQL, dbName)

	rows, sqlErr := d.db.Query(strSQL, argIDs...)
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

	if sqlErr != nil {
		d.logger.Errorln(sqlErr, strSQL, argIDs)
		return nil, nil, sqlErr
	}

	var tmpInfo interfaces.NameInfo
	existIDs = make([]string, 0)
	info = make([]interfaces.NameInfo, 0)
	for rows.Next() {
		if scanErr := rows.Scan(&tmpInfo.ID, &tmpInfo.Name); scanErr != nil {
			d.logger.Errorln(scanErr, strSQL)
			return nil, nil, scanErr
		}

		existIDs = append(existIDs, tmpInfo.ID)
		info = append(info, tmpInfo)
	}

	return info, existIDs, err
}

// GetParentDepartmentID 批量获取父部门id
func (d *department) GetParentDepartmentID(deptIDs []string) ([]string, error) {
	if len(deptIDs) == 0 {
		return nil, nil
	}

	set, argIDs := GetFindInSetSQL(deptIDs)
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "select f_parent_department_id from %s.t_department_relation where f_department_id in ("
	strSQL += set
	strSQL += ")"
	strSQL = fmt.Sprintf(strSQL, dbName)

	rows, sqlErr := d.db.Query(strSQL, argIDs...)
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

	if sqlErr != nil {
		return nil, sqlErr
	}

	parentIDs := make([]string, 0)
	var parentDeptID string
	for rows.Next() {
		if err := rows.Scan(&parentDeptID); err != nil {
			d.logger.Errorln(err, strSQL)
			return nil, err
		}

		parentIDs = append(parentIDs, parentDeptID)
	}
	return parentIDs, nil
}

// GetChildDepartmentIDs 批量获取子部门id
func (d *department) GetChildDepartmentIDs(deptIDs []string) (childIDs []string, childDepMap map[string][]string, err error) {
	childIDs = make([]string, 0)
	childDepMap = make(map[string][]string)
	if len(deptIDs) == 0 {
		return childIDs, childDepMap, nil
	}

	set, argIDs := GetFindInSetSQL(deptIDs)
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "select f_department_id, f_parent_department_id from %s.t_department_relation where f_parent_department_id in ("
	strSQL += set
	strSQL += ")"
	strSQL = fmt.Sprintf(strSQL, dbName)

	rows, sqlErr := d.db.Query(strSQL, argIDs...)
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

	if sqlErr != nil {
		return childIDs, childDepMap, sqlErr
	}

	var childDeptID string
	var parentDepID string
	for rows.Next() {
		if err := rows.Scan(&childDeptID, &parentDepID); err != nil {
			d.logger.Errorln(err, strSQL)
			return make([]string, 0), make(map[string][]string), err
		}

		childIDs = append(childIDs, childDeptID)

		if _, ok := childDepMap[parentDepID]; !ok {
			childDepMap[parentDepID] = make([]string, 0)
		}
		childDepMap[parentDepID] = append(childDepMap[parentDepID], childDeptID)
	}
	return childIDs, childDepMap, nil
}

// GetChildDepartmentIDs 批量获取子部门id
func (d *department) GetChildDepartmentIDs2(ctx context.Context, deptIDs []string) (childIDs []string, childDepMap map[string][]string, err error) {
	// trace
	d.trace.SetClientSpanName("数据库操作-获取部门下所有的子部门ID")
	newCtx, span := d.trace.AddClientTrace(ctx)
	defer func() { d.trace.TelemetrySpanEnd(span, err) }()

	childIDs = make([]string, 0)
	childDepMap = make(map[string][]string)
	if len(deptIDs) == 0 {
		return childIDs, childDepMap, nil
	}

	set, argIDs := GetFindInSetSQL(deptIDs)
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "select f_department_id, f_parent_department_id from %s.t_department_relation where f_parent_department_id in ("
	strSQL += set
	strSQL += ")"
	strSQL = fmt.Sprintf(strSQL, dbName)

	rows, sqlErr := d.dbTrace.QueryContext(newCtx, strSQL, argIDs...)
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

	if sqlErr != nil {
		return childIDs, childDepMap, sqlErr
	}

	var childDeptID string
	var parentDepID string
	for rows.Next() {
		if err := rows.Scan(&childDeptID, &parentDepID); err != nil {
			d.logger.Errorln(err, strSQL)
			return make([]string, 0), make(map[string][]string), err
		}

		childIDs = append(childIDs, childDeptID)

		if _, ok := childDepMap[parentDepID]; !ok {
			childDepMap[parentDepID] = make([]string, 0)
		}
		childDepMap[parentDepID] = append(childDepMap[parentDepID], childDeptID)
	}
	return childIDs, childDepMap, nil
}

// GetChildUserIDs 获取部门子用户id
func (d *department) GetChildUserIDs(deptIDs []string) (childUserIDs []string, childUsersMap map[string][]string, err error) {
	childUsersMap = make(map[string][]string)
	childUserIDs = make([]string, 0)
	if len(deptIDs) == 0 {
		return childUserIDs, childUsersMap, nil
	}

	set, argIDs := GetFindInSetSQL(deptIDs)
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "select f_user_id, f_department_id from %s.t_user_department_relation where f_department_id in ("
	strSQL += set
	strSQL += ")"
	strSQL = fmt.Sprintf(strSQL, dbName)

	rows, sqlErr := d.db.Query(strSQL, argIDs...)
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

	if sqlErr != nil {
		return childUserIDs, childUsersMap, sqlErr
	}

	var childUserID string
	var parentDepID string
	for rows.Next() {
		if err := rows.Scan(&childUserID, &parentDepID); err != nil {
			d.logger.Errorln(err, strSQL)
			return make([]string, 0), make(map[string][]string), err
		}

		childUserIDs = append(childUserIDs, childUserID)

		if _, ok := childUsersMap[parentDepID]; !ok {
			childUsersMap[parentDepID] = make([]string, 0)
		}
		childUsersMap[parentDepID] = append(childUsersMap[parentDepID], childUserID)
	}
	return childUserIDs, childUsersMap, nil
}

// GetRootDeps 获取根部门信息
func (d *department) GetRootDeps(bCount, bNoScope bool, scope []string, offset, limit int) (out []interfaces.DepartmentDBInfo, err error) {
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "select f_department_id, f_name, f_is_enterprise, f_code, f_remark, f_manager_id, f_status, f_mail_address from %s.t_department where f_is_enterprise = 1 "
	strSQL = fmt.Sprintf(strSQL, dbName)

	var argIDs []interface{}
	if !bNoScope {
		if len(scope) == 0 {
			return
		}
		set, args := GetFindInSetSQL(scope)
		strSQL += "and f_department_id in ("
		strSQL += set
		strSQL += ")"

		argIDs = append(argIDs, args...)
	}

	if !bCount {
		strSQL += "order by f_priority, upper(f_name) limit ?,? "
		argIDs = append(argIDs, offset, limit)
	}

	rows, sqlErr := d.db.Query(strSQL, argIDs...)
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

	if sqlErr != nil {
		d.logger.Errorln(sqlErr, strSQL, argIDs)
		return out, sqlErr
	}

	var tmpInfo interfaces.DepartmentDBInfo
	for rows.Next() {
		tempStatus := 0
		if scanErr := rows.Scan(&tmpInfo.ID, &tmpInfo.Name, &tmpInfo.IsRoot, &tmpInfo.Code, &tmpInfo.Remark, &tmpInfo.ManagerID, &tempStatus, &tmpInfo.Email); scanErr != nil {
			d.logger.Errorln(scanErr, strSQL)
			return out, scanErr
		}

		tmpInfo.Status = false
		if tempStatus == 1 {
			tmpInfo.Status = true
		}

		out = append(out, tmpInfo)
	}

	return out, err
}

// GetSubDepartmentInfos 获取部门子部门信息
func (d *department) GetSubDepartmentInfos(deptID string, bCount bool, offset, limit int) (out []interfaces.DepartmentDBInfo, err error) {
	argIDs := []interface{}{deptID}
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := `select f_department_id, f_name, f_is_enterprise, f_code, f_remark, f_manager_id, f_status, f_mail_address from %s.t_department
				where f_department_id in
				(select f_department_id from %s.t_department_relation
	 			where f_parent_department_id = ?)  `
	strSQL = fmt.Sprintf(strSQL, dbName, dbName)

	if bCount {
		strSQL += " order by f_priority, upper(f_name) limit ? , ? "
		argIDs = append(argIDs, offset, limit)
	}

	rows, sqlErr := d.db.Query(strSQL, argIDs...)
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

	if sqlErr != nil {
		return out, sqlErr
	}

	var info interfaces.DepartmentDBInfo
	for rows.Next() {
		tempStatus := 0
		if err := rows.Scan(&info.ID, &info.Name, &info.IsRoot, &info.Code, &info.Remark, &info.ManagerID, &tempStatus, &info.Email); err != nil {
			d.logger.Errorln(err, strSQL)
			return out, err
		}

		info.Status = false
		if tempStatus == 1 {
			info.Status = true
		}

		out = append(out, info)
	}
	return out, nil
}

// GetSubUserInfos 获取部门子用户信息（需排序）
func (d *department) GetSubUserInfos(deptID string, bCount bool, offset, limit int) (out []interfaces.UserDBInfo, err error) {
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := `SELECT c_user.f_user_id, c_user.f_display_name, c_user.f_status, c_user.f_priority, c_user.f_csf_level, c_user.f_auto_disable_status
			FROM %s.t_user AS c_user
			JOIN %s.t_user_department_relation AS relation
				ON relation.f_user_id =c_user.f_user_id
				WHERE relation.f_department_id = ? and c_user.f_status = 0 and c_user.f_auto_disable_status = 0 `
	strSQL = fmt.Sprintf(strSQL, dbName, dbName)

	argID := []interface{}{deptID}
	if bCount {
		strSQL += "ORDER BY f_priority, upper(f_display_name) limit ? , ? "
		argID = append(argID, offset, limit)
	}

	rows, sqlErr := d.db.Query(strSQL, argID...)
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

	if sqlErr != nil {
		return out, sqlErr
	}

	var info userDBData
	for rows.Next() {
		if err := rows.Scan(&info.ID, &info.Name, &info.DisableStatus, &info.AutoDisableStatus, &info.Priority, &info.CSFLevel); err != nil {
			d.logger.Errorln(err, strSQL)
			return out, err
		}

		out = append(out, handlerUserDBData(&info))
	}
	return out, nil
}

// GetDepartmentInfo 批量获取部门信息
func (d *department) GetDepartmentInfo(deptIDs []string, bLimit bool, offset, limit int) (out []interfaces.DepartmentDBInfo, err error) {
	if len(deptIDs) == 0 {
		return
	}

	set, argIDs := GetFindInSetSQL(deptIDs)
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "select f_department_id, f_name, f_is_enterprise, f_mail_address, f_path, f_manager_id, f_code, f_status, f_third_party_id from %s.t_department where f_department_id in ("
	strSQL += set
	strSQL += ") "
	strSQL = fmt.Sprintf(strSQL, dbName)

	if bLimit {
		strSQL += "order by f_priority, upper(f_name) limit ?, ? "
		argIDs = append(argIDs, offset, limit)
	}

	rows, sqlErr := d.db.Query(strSQL, argIDs...)
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

	if sqlErr != nil {
		d.logger.Errorln(sqlErr, strSQL, argIDs)
		return out, sqlErr
	}

	var tmpInfo interfaces.DepartmentDBInfo
	for rows.Next() {
		tempStatus := 0
		if scanErr := rows.Scan(&tmpInfo.ID, &tmpInfo.Name, &tmpInfo.IsRoot, &tmpInfo.Email, &tmpInfo.Path, &tmpInfo.ManagerID, &tmpInfo.Code, &tempStatus, &tmpInfo.ThirdID); scanErr != nil {
			d.logger.Errorln(scanErr, strSQL)
			return out, scanErr
		}

		tmpInfo.Status = false
		if tempStatus == 1 {
			tmpInfo.Status = true
		}

		out = append(out, tmpInfo)
	}

	return out, err
}

// GetDepartmentInfoByIDs 根据ID批量获取部门信息
func (d *department) GetDepartmentInfoByIDs(ctx context.Context, deptIDs []string) (out []interfaces.DepartmentDBInfo, err error) {
	// trace
	d.trace.SetClientSpanName("数据库操作-根据部门ID批量获取部门信息")
	newCtx, span := d.trace.AddClientTrace(ctx)
	defer func() { d.trace.TelemetrySpanEnd(span, err) }()

	// 分割ID 防止sql过长
	splitedIDs := SplitArray(deptIDs)

	out = make([]interfaces.DepartmentDBInfo, 0)
	for _, ids := range splitedIDs {
		infoTmp, tmpErr := d.getDepartmentInfo2(newCtx, ids, false, 0, 0)
		if tmpErr != nil {
			return nil, tmpErr
		}

		out = append(out, infoTmp...)
	}
	return out, err
}

// GetDepartmentInfo2 批量获取部门信息
func (d *department) GetDepartmentInfo2(ctx context.Context, deptIDs []string, bLimit bool, offset, limit int) (out []interfaces.DepartmentDBInfo, err error) {
	// trace
	d.trace.SetClientSpanName("数据库操作-根据部门ID批量获取部门信息")
	newCtx, span := d.trace.AddClientTrace(ctx)
	defer func() { d.trace.TelemetrySpanEnd(span, err) }()

	return d.getDepartmentInfo2(newCtx, deptIDs, bLimit, offset, limit)
}

// getDepartmentInfo2 批量获取部门信息
func (d *department) getDepartmentInfo2(ctx context.Context, deptIDs []string, bLimit bool, offset, limit int) (out []interfaces.DepartmentDBInfo, err error) {
	if len(deptIDs) == 0 {
		return
	}

	set, argIDs := GetFindInSetSQL(deptIDs)
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "select f_department_id, f_name, f_is_enterprise, f_mail_address, f_path, f_code from %s.t_department where f_department_id in ("
	strSQL += set
	strSQL += ") "
	strSQL = fmt.Sprintf(strSQL, dbName)

	if bLimit {
		strSQL += "order by f_priority, upper(f_name) limit ?, ? "
		argIDs = append(argIDs, offset, limit)
	}

	rows, sqlErr := d.dbTrace.QueryContext(ctx, strSQL, argIDs...)
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

	if sqlErr != nil {
		d.logger.Errorln(sqlErr, strSQL, argIDs)
		return out, sqlErr
	}

	var tmpInfo interfaces.DepartmentDBInfo
	for rows.Next() {
		if scanErr := rows.Scan(&tmpInfo.ID, &tmpInfo.Name, &tmpInfo.IsRoot, &tmpInfo.Email, &tmpInfo.Path, &tmpInfo.Code); scanErr != nil {
			d.logger.Errorln(scanErr, strSQL)
			return out, scanErr
		}

		out = append(out, tmpInfo)
	}

	return out, err
}

// SearchDepartsByKey 搜索部门
func (d *department) SearchDepartsByKey(ctx context.Context, bCount, bNoScope bool, scope []string, keyword string, offset, limit int) (out []interfaces.DepartmentDBInfo, err error) {
	// trace
	d.trace.SetClientSpanName("数据库操作-在管辖范围中搜索相关部门的数据或者符合条件部门数量")
	newCtx, span := d.trace.AddClientTrace(ctx)
	defer func() { d.trace.TelemetrySpanEnd(span, err) }()

	var argIDs []interface{}
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := ` SELECT DISTINCT f_department_id, f_name, f_path FROM %s.t_department WHERE f_name LIKE ? `
	strSQL = fmt.Sprintf(strSQL, dbName)
	argIDs = append(argIDs, "%"+keyword+"%")
	if !bNoScope {
		if len(scope) == 0 {
			return
		}
		depSQL := GetUUIDStringBySlice(scope)
		strSQL += ` and f_department_id in (`
		strSQL += depSQL
		strSQL += `) `
	}

	if !bCount {
		strSQL += "ORDER BY upper(f_name) limit ?, ? "
		argIDs = append(argIDs, offset, limit)
	}

	rows, sqlErr := d.dbTrace.QueryContext(newCtx, strSQL, argIDs...)
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

	if sqlErr != nil {
		d.logger.Errorln(sqlErr, strSQL, argIDs)
		return out, sqlErr
	}

	var tmpInfo interfaces.DepartmentDBInfo
	for rows.Next() {
		if scanErr := rows.Scan(&tmpInfo.ID, &tmpInfo.Name, &tmpInfo.Path); scanErr != nil {
			d.logger.Errorln(scanErr, strSQL)
			return out, scanErr
		}

		out = append(out, tmpInfo)
	}

	return out, err
}

func (d *department) GetManagersOfDepartment(departmentIDs []string) (infoList []interfaces.DepartmentManagerInfo, err error) {
	set, argIDs := GetFindInSetSQL(departmentIDs)
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := `SELECT a.f_department_id, a.f_user_id, b.f_display_name
		FROM %s.t_department_responsible_person AS a
		LEFT JOIN %s.t_user AS b
			ON a.f_user_id = b.f_user_id
			WHERE a.f_department_id in (`
	strSQL += set
	strSQL += ")"
	strSQL = fmt.Sprintf(strSQL, dbName, dbName)

	rows, sqlErr := d.db.Query(strSQL, argIDs...)
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

	if sqlErr != nil {
		d.logger.Errorln(sqlErr, strSQL, argIDs)
		return nil, sqlErr
	}

	var tmpDepartmentID, tmpUserID, tmpName string
	var nameInfo interfaces.NameInfo
	infoMap := make(map[string][]interfaces.NameInfo)
	for rows.Next() {
		if err = rows.Scan(
			&tmpDepartmentID,
			&tmpUserID,
			&tmpName,
		); err != nil {
			d.logger.Errorln(err, strSQL)
			return
		}

		nameInfo = interfaces.NameInfo{
			ID:   tmpUserID,
			Name: tmpName,
		}
		infoMap[tmpDepartmentID] = append(infoMap[tmpDepartmentID], nameInfo)
	}

	var tmpInfo interfaces.DepartmentManagerInfo
	for k, v := range infoMap {
		tmpInfo.DepartmentID = k
		tmpInfo.Managers = v
		infoList = append(infoList, tmpInfo)
	}

	return
}

// GetDepartmentByPathLength 根据path长度获取部门信息
func (d *department) GetDepartmentByPathLength(nLen int) (infoList []interfaces.DepartmentDBInfo, err error) {
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := `SELECT f_department_id, f_name, f_third_party_id FROM %s.t_department WHERE LENGTH(f_path) = ?`
	strSQL = fmt.Sprintf(strSQL, dbName)

	rows, sqlErr := d.db.Query(strSQL, nLen)
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

	if sqlErr != nil {
		d.logger.Errorln(sqlErr, strSQL, nLen)
		return nil, sqlErr
	}

	infoList = make([]interfaces.DepartmentDBInfo, 0)
	for rows.Next() {
		nameInfo := interfaces.DepartmentDBInfo{}
		if err = rows.Scan(&nameInfo.ID, &nameInfo.Name, &nameInfo.ThirdID); err != nil {
			d.logger.Errorln(err, strSQL)
			return
		}
		infoList = append(infoList, nameInfo)
	}

	return
}

// GetAllSubUserIDsByDepartPath 根据path获取部门所有子成员
func (d *department) GetAllSubUserIDsByDepartPath(path string) (ids []string, err error) {
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := `SELECT f_user_id FROM %s.t_user_department_relation WHERE f_path like ?`
	strSQL = fmt.Sprintf(strSQL, dbName)

	rows, sqlErr := d.db.Query(strSQL, path+"%")
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

	if sqlErr != nil {
		d.logger.Errorln(sqlErr, strSQL, path)
		return nil, sqlErr
	}

	ids = make([]string, 0)
	for rows.Next() {
		useID := ""
		if err = rows.Scan(&useID); err != nil {
			d.logger.Errorln(err, strSQL)
			return
		}
		ids = append(ids, useID)
	}

	return
}

// GetAllSubUserInfosByDepartPath 根据path获取部门所有子成员基本信息
func (d *department) GetAllSubUserInfosByDepartPath(path string) (infos []interfaces.UserBaseInfo, err error) {
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := `SELECT a.f_user_id ,b.f_login_name, b.f_display_name,  b.f_mail_address, IFNULL(b.f_tel_number, ''),
		b.f_third_party_attr, IFNULL(b.f_third_party_id, '')  FROM %s.t_user_department_relation as a
		left join %s.t_user as b on a.f_user_id = b.f_user_id
		WHERE a.f_path like ?`
	strSQL = fmt.Sprintf(strSQL, dbName, dbName)

	rows, sqlErr := d.db.Query(strSQL, path+"%")
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

	if sqlErr != nil {
		d.logger.Errorln(sqlErr, strSQL, path)
		return nil, sqlErr
	}

	infos = make([]interfaces.UserBaseInfo, 0)
	for rows.Next() {
		userInfo := interfaces.UserBaseInfo{}
		if err = rows.Scan(&userInfo.ID, &userInfo.Account, &userInfo.Name, &userInfo.Email,
			&userInfo.TelNumber, &userInfo.ThirdAttr, &userInfo.ThirdID); err != nil {
			d.logger.Errorln(err, strSQL)
			return
		}
		infos = append(infos, userInfo)
	}

	return
}

// DeleteOrgManagerRelationByDepartID 根据部门ID删除组织管理员管辖信息
func (d *department) DeleteOrgManagerRelationByDepartID(id string) (err error) {
	dbName := common.GetDBName("sharemgnt_db")
	sqlStr := "delete from %s.t_department_responsible_person where f_department_id = ?"
	sqlStr = fmt.Sprintf(sqlStr, dbName)
	_, err = d.db.Exec(sqlStr, id)
	if err != nil {
		d.logger.Errorln(err, sqlStr, id)
	}
	return err
}

// DeleteOrgAuditRelationByDepartID 根据部门ID删除组织审计员管辖信息
func (d *department) DeleteOrgAuditRelationByDepartID(id string) (err error) {
	dbName := common.GetDBName("sharemgnt_db")
	sqlStr := "delete from %s.t_department_audit_person where f_department_id = ?"
	sqlStr = fmt.Sprintf(sqlStr, dbName)
	_, err = d.db.Exec(sqlStr, id)
	if err != nil {
		d.logger.Errorln(err, sqlStr, id)
	}
	return err
}

// DeleteUserDepartRelationByPath 根据部门路径删除部门下用户的用户/部门关系
func (d *department) DeleteUserDepartRelationByPath(path string, tx *sql.Tx) (err error) {
	dbName := common.GetDBName("sharemgnt_db")
	sqlStr := "delete from %s.t_user_department_relation where f_path like ?"
	sqlStr = fmt.Sprintf(sqlStr, dbName)
	_, err = tx.Exec(sqlStr, path+"%")
	if err != nil {
		d.logger.Errorln(err, sqlStr, path)
	}
	return err
}

// AddUserToDepart 添加用户到部门,每1w执行一次
func (d *department) AddUserToDepart(userIDs []string, departID, departPath string, tx *sql.Tx) (err error) {
	if len(userIDs) == 0 {
		return nil
	}

	nCount := len(userIDs)
	nStep := 10000
	nStart := 0
	nEnd := 0
	for {
		nEnd = nStart + nStep
		if nEnd > nCount {
			nEnd = nCount
		}

		err := d.addUserToDepartSingle(userIDs[nStart:nEnd], departID, departPath, tx)
		if err != nil {
			return err
		}

		nStart = nEnd
		if nStart >= nCount {
			break
		}
	}

	return nil
}

// AddUserToDepart 添加用户到部门
func (d *department) addUserToDepartSingle(userIDs []string, departID, departPath string, tx *sql.Tx) (err error) {
	nUserSize := len(userIDs)
	if nUserSize == 0 {
		return nil
	}

	nFieldSize := 3
	extraSQL := make([]string, 0, nUserSize)
	values := make([]interface{}, 0, nUserSize*nFieldSize)
	for _, v := range userIDs {
		extraSQL = append(extraSQL, "(?, ?, ?)")
		values = append(values, v, departID, departPath)
	}

	dbName := common.GetDBName("sharemgnt_db")
	sqlStr := fmt.Sprintf("insert into %s.t_user_department_relation (f_user_id, f_department_id, f_path) values %s", dbName, strings.Join(extraSQL, ","))
	_, err = tx.Exec(sqlStr, values...)
	if err != nil {
		d.logger.Errorln(err, sqlStr, values)
	}
	return err
}

// DeleteUserOURelation 删除用户/组织关系
func (d *department) DeleteUserOURelation(userIDs []string, orgID string, tx *sql.Tx) (err error) {
	if len(userIDs) == 0 {
		return nil
	}

	nCount := len(userIDs)
	nStep := 10000
	nStart := 0
	nEnd := 0
	for {
		nEnd = nStart + nStep
		if nEnd > nCount {
			nEnd = nCount
		}

		err := d.deleteUserOURelationSingle(userIDs[nStart:nEnd], orgID, tx)
		if err != nil {
			return err
		}

		nStart = nEnd
		if nStart >= nCount {
			break
		}
	}

	return nil
}

// DeleteUserOURelation 删除用户/组织关系
func (d *department) deleteUserOURelationSingle(userIDs []string, orgID string, tx *sql.Tx) (err error) {
	nUserSize := len(userIDs)
	if nUserSize == 0 {
		return nil
	}

	extraSQL := make([]string, 0, nUserSize)
	values := make([]interface{}, 0, nUserSize)
	for _, v := range userIDs {
		extraSQL = append(extraSQL, "?")
		values = append(values, v)
	}
	values = append(values, orgID)

	dbName := common.GetDBName("sharemgnt_db")
	sqlStr := fmt.Sprintf("delete from %s.t_ou_user where f_user_id in (%s) and f_ou_id = ?", dbName, strings.Join(extraSQL, ","))
	_, err = tx.Exec(sqlStr, values...)
	if err != nil {
		d.logger.Errorln(err, sqlStr, values)
	}
	return err
}

// DeleteDepartByPath 根据路径删除部门下所有部门信息
func (d *department) DeleteDepartByPath(path string, tx *sql.Tx) (err error) {
	dbName := common.GetDBName("sharemgnt_db")
	sqlStr := "delete from %s.t_department where f_path like ?"
	sqlStr = fmt.Sprintf(sqlStr, dbName)
	_, err = tx.Exec(sqlStr, path+"%")
	if err != nil {
		d.logger.Errorln(err, sqlStr, path)
	}
	return err
}

// GetAllSubDepartInfosByPath 根据路径获取部门下所有子部门
func (d *department) GetAllSubDepartInfosByPath(path string) (infos []interfaces.DepartmentDBInfo, err error) {
	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "select f_department_id, f_path from %s.t_department where f_path like ?"
	strSQL = fmt.Sprintf(strSQL, dbName)

	rows, sqlErr := d.db.Query(strSQL, path+"%")
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

	if sqlErr != nil {
		d.logger.Errorln(sqlErr, strSQL, path)
		return nil, sqlErr
	}

	infos = make([]interfaces.DepartmentDBInfo, 0)
	tempID := ""
	tempPath := ""
	for rows.Next() {
		if err = rows.Scan(&tempID, &tempPath); err != nil {
			d.logger.Errorln(err, strSQL)
			return
		}
		infos = append(infos, interfaces.DepartmentDBInfo{ID: tempID, Path: tempPath})
	}

	return
}

// DeleteDepartRelations 删除部门的组织关系
func (d *department) DeleteDepartRelations(departIDs []string, tx *sql.Tx) (err error) {
	if len(departIDs) == 0 {
		return nil
	}

	set, argIDs := GetFindInSetSQL(departIDs)
	dbName := common.GetDBName("sharemgnt_db")
	sqlStr := "delete from %s.t_department_relation where f_department_id in ("
	sqlStr += set
	sqlStr += ")"
	sqlStr = fmt.Sprintf(sqlStr, dbName)
	_, err = tx.Exec(sqlStr, argIDs...)
	if err != nil {
		d.logger.Errorln(err, sqlStr, argIDs)
	}
	return err
}

// DeleteDepartRelations 删除部门的组织关系
func (d *department) DeleteDepartOURelations(departIDs []string, tx *sql.Tx) (err error) {
	if len(departIDs) == 0 {
		return nil
	}

	set, argIDs := GetFindInSetSQL(departIDs)
	dbName := common.GetDBName("sharemgnt_db")
	sqlStr := "delete from %s.t_ou_department where f_department_id in ("
	sqlStr += set
	sqlStr += ")"
	sqlStr = fmt.Sprintf(sqlStr, dbName)

	_, err = tx.Exec(sqlStr, argIDs...)
	if err != nil {
		d.logger.Errorln(err, sqlStr, argIDs)
	}
	return err
}

// GetAllOrgManagerIDsByDepartIDs 根据部门ID获取所有的组织管理员
func (d *department) GetAllOrgManagerIDsByDepartIDs(departIds []string) (orgManagerIDs []string, err error) {
	if len(departIds) == 0 {
		return nil, nil
	}

	set, argIDs := GetFindInSetSQL(departIds)
	dbName := common.GetDBName("sharemgnt_db")
	sqlStr := "select f_user_id from %s.t_department_responsible_person where f_department_id in ("
	sqlStr += set
	sqlStr += ")"
	sqlStr = fmt.Sprintf(sqlStr, dbName)

	rows, sqlErr := d.db.Query(sqlStr, argIDs...)
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

	if sqlErr != nil {
		d.logger.Errorln(sqlErr, sqlStr, argIDs)
		return nil, sqlErr
	}

	orgManagerIDs = make([]string, 0)
	tempID := ""
	for rows.Next() {
		if err = rows.Scan(&tempID); err != nil {
			d.logger.Errorln(err, sqlStr, argIDs)
			return
		}
		orgManagerIDs = append(orgManagerIDs, tempID)
	}
	return
}

// GetAllOrgManagerIDs 获取所有的组织管理员ID
func (d *department) GetAllOrgManagerIDs() (ids []string, err error) {
	dbName := common.GetDBName("sharemgnt_db")
	sqlStr := "select f_user_id from %s.t_department_responsible_person"
	sqlStr = fmt.Sprintf(sqlStr, dbName)

	rows, sqlErr := d.db.Query(sqlStr)
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

	if sqlErr != nil {
		d.logger.Errorln(sqlErr, sqlStr)
		return nil, sqlErr
	}

	ids = make([]string, 0)
	tempID := ""
	for rows.Next() {
		if err = rows.Scan(&tempID); err != nil {
			d.logger.Errorln(err, sqlStr)
			return
		}
		ids = append(ids, tempID)
	}
	return
}

// DeleteDocAutoCleanStrategy 删除文档自动清理策略
func (d *department) DeleteDocAutoCleanStrategy(obj string) (err error) {
	dbName := common.GetDBName("sharemgnt_db")
	sqlStr := `delete from %s.t_doc_auto_clean_strategy where f_obj_id = ?`
	sqlStr = fmt.Sprintf(sqlStr, dbName)

	_, err = d.db.Exec(sqlStr, obj)
	if err != nil {
		d.logger.Errorln(err, sqlStr, obj)
	}
	return err
}

// DeleteDepartManager 清理部门负责人数据
func (d *department) DeleteDepartManager(userID string) (err error) {
	dbName := common.GetDBName("sharemgnt_db")
	sqlStr := `update %s.t_department set f_manager_id = '' where f_manager_id = ?`
	sqlStr = fmt.Sprintf(sqlStr, dbName)

	_, err = d.db.Exec(sqlStr, userID)
	if err != nil {
		d.logger.Errorln(err, sqlStr, userID)
	}
	return err
}

// SearchDeparts 内置管理员按照关键字搜索部门
func (d *department) SearchDeparts(ctx context.Context, ks *interfaces.DepartSearchKeyScope, k *interfaces.DepartSearchKey, limitDepartIDs []string) (out []interfaces.DepartmentDBInfo, err error) {
	// trace
	d.trace.SetClientSpanName("数据库操作-按照关键字进行搜索")
	newCtx, span := d.trace.AddClientTrace(ctx)
	defer func() { d.trace.TelemetrySpanEnd(span, err) }()

	args := make([]interface{}, 0)
	strSQLParam := make([]interface{}, 0)

	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "select f_department_id, f_path, f_name, f_code, f_manager_id, f_remark, f_status, f_mail_address from %s.t_department "
	strSQLParam = append(strSQLParam, dbName)

	bHasK := true
	if ks.BCode {
		strSQL += " where f_code like ? "
		args = append(args, "%"+k.Code+"%")
	} else if ks.BName {
		strSQL += " where f_name like ? "
		args = append(args, "%"+k.Name+"%")
	} else if ks.BRemark {
		strSQL += " where f_remark like ? "
		args = append(args, "%"+k.Remark+"%")
	} else if ks.BEnabled {
		status := 1
		if !k.Enabled {
			status = 2
		}
		strSQL += " where f_status = ? "
		args = append(args, status)
	} else if ks.BManagerName {
		strSQL += " where f_manager_id in (select f_user_id from %s.t_user where f_display_name like ?) "
		args = append(args, "%"+k.ManagerName+"%")
		strSQLParam = append(strSQLParam, dbName)
	} else if ks.BDirectDepartCode {
		strSQL += " where f_department_id in (select f_department_id from %s.t_department_relation where f_parent_department_id in (select f_department_id from %s.t_department where f_code like ?)) "
		args = append(args, "%"+k.DirectDepartCode+"%")
		strSQLParam = append(strSQLParam, dbName, dbName)
	} else {
		bHasK = false
	}

	if len(limitDepartIDs) > 0 {
		if bHasK {
			strSQL += sqlAnd
		} else {
			strSQL += sqlWhere
		}

		set, limitArgs := GetFindInSetSQL(limitDepartIDs)
		strSQL += " f_department_id in ("
		strSQL += set
		strSQL += ") "
		args = append(args, limitArgs...)
	}

	strSQL += " limit ?, ?"
	args = append(args, k.Offset, k.Limit)
	strSQL = fmt.Sprintf(strSQL, strSQLParam...)

	rows, sqlErr := d.dbTrace.QueryContext(newCtx, strSQL, args...)
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

	if sqlErr != nil {
		d.logger.Errorln(sqlErr, strSQL)
		return nil, sqlErr
	}

	out = make([]interfaces.DepartmentDBInfo, 0)
	for rows.Next() {
		temp := interfaces.DepartmentDBInfo{}
		status := 0
		if err = rows.Scan(&temp.ID, &temp.Path, &temp.Name, &temp.Code, &temp.ManagerID, &temp.Remark, &status, &temp.Email); err != nil {
			d.logger.Errorln(err, strSQL)
			return
		}
		temp.Status = status == 1
		out = append(out, temp)
	}

	return
}

// SearchDepartsCount 内置管理员按照关键字搜索部门
func (d *department) SearchDepartsCount(ctx context.Context, ks *interfaces.DepartSearchKeyScope, k *interfaces.DepartSearchKey, limitDepartIDs []string) (count int, err error) {
	// trace
	d.trace.SetClientSpanName("数据库操作-按照关键字进行搜索返回数量")
	newCtx, span := d.trace.AddClientTrace(ctx)
	defer func() { d.trace.TelemetrySpanEnd(span, err) }()

	args := make([]interface{}, 0)
	strSQLParam := make([]interface{}, 0)

	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "select count(*) from %s.t_department "
	strSQLParam = append(strSQLParam, dbName)

	bHasK := true
	if ks.BCode {
		strSQL += " where f_code like ? "
		args = append(args, "%"+k.Code+"%")
	} else if ks.BName {
		strSQL += " where f_name like ? "
		args = append(args, "%"+k.Name+"%")
	} else if ks.BRemark {
		strSQL += " where f_remark like ? "
		args = append(args, "%"+k.Remark+"%")
	} else if ks.BEnabled {
		status := 1
		if !k.Enabled {
			status = 2
		}
		strSQL += " where f_status = ? "
		args = append(args, status)
	} else if ks.BManagerName {
		strSQL += " where f_manager_id in (select f_user_id from %s.t_user where f_display_name like ?)"
		args = append(args, "%"+k.ManagerName+"%")
		strSQLParam = append(strSQLParam, dbName)
	} else if ks.BDirectDepartCode {
		strSQL += " where f_department_id in (select f_department_id from %s.t_department_relation where f_parent_department_id in (select f_department_id from %s.t_department where f_code like ?))"
		args = append(args, "%"+k.DirectDepartCode+"%")
		strSQLParam = append(strSQLParam, dbName, dbName)
	} else {
		bHasK = false
	}

	if len(limitDepartIDs) > 0 {
		if bHasK {
			strSQL += sqlAnd
		} else {
			strSQL += sqlWhere
		}

		set, limitArgs := GetFindInSetSQL(limitDepartIDs)
		strSQL += " f_department_id in ("
		strSQL += set
		strSQL += ")"
		args = append(args, limitArgs...)
	}

	strSQL = fmt.Sprintf(strSQL, strSQLParam...)

	rows, sqlErr := d.dbTrace.QueryContext(newCtx, strSQL, args...)
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

	if sqlErr != nil {
		d.logger.Errorln(sqlErr, strSQL)
		return 0, sqlErr
	}

	for rows.Next() {
		if err = rows.Scan(&count); err != nil {
			d.logger.Errorln(err, strSQL)
			return
		}
	}

	return
}
