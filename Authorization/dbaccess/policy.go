// Package dbaccess 数据访问层
package dbaccess

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"sync"

	"github.com/kweaver-ai/proton-rds-sdk-go/sqlx"

	"Authorization/common"
	"Authorization/interfaces"
)

type policy struct {
	db     *sqlx.DB
	logger common.Logger
}

var (
	policyOnce    sync.Once
	policyService *policy
)

// NewPolicy 创建数据库对象
func NewPolicy() *policy {
	policyOnce.Do(func() {
		policyService = &policy{
			db:     dbPool,
			logger: common.NewLogger(),
		}
	})
	return policyService
}

//nolint:lll,dupl
func (d *policy) GetPagination(ctx context.Context, params interfaces.PolicyPagination) (count int, policies []interfaces.PolicyInfo, err error) {
	var countRows *sql.Rows
	countRows, err = d.db.Query("select count(1) from "+common.GetDBName(databaseName)+".t_policy where f_resource_id = ? and f_resource_type = ?", params.ResourceID, params.ResourceType)
	if err != nil {
		d.logger.Errorln(err)
		return 0, nil, err
	}

	for countRows.Next() {
		err = countRows.Scan(&count)
		if err != nil {
			d.logger.Errorln(err)
			return 0, nil, err
		}
	}

	if countRows != nil {
		if countRowsErr := countRows.Err(); countRowsErr != nil {
			d.logger.Errorln(countRowsErr)
		}
		if closeErr := countRows.Close(); closeErr != nil {
			d.logger.Errorln(closeErr)
		}
	}

	var rows *sql.Rows
	strSQL := "select f_id, f_resource_id, f_resource_type, f_resource_name, f_accessor_id, f_accessor_type, f_accessor_name, f_operation, f_condition, f_end_time, f_create_time, f_modify_time from " + common.GetDBName(databaseName) +
		".t_policy where f_resource_id = ? and f_resource_type = ? order by f_modify_time desc, f_primary_id desc limit ? offset ?"
	rows, err = d.db.Query(strSQL, params.ResourceID, params.ResourceType, params.Limit, params.Offset)
	if err != nil {
		d.logger.Errorln(err)
		return 0, nil, err
	}

	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				d.logger.Errorln(rowsErr)
			}

			if closeErr := rows.Close(); closeErr != nil {
				d.logger.Errorln(closeErr)
			}
		}
	}()

	for rows.Next() {
		var policy interfaces.PolicyInfo
		var operationStr string
		err := rows.Scan(&policy.ID, &policy.ResourceID, &policy.ResourceType,
			&policy.ResourceName, &policy.AccessorID, &policy.AccessorType, &policy.AccessorName,
			&operationStr, &policy.Condition, &policy.EndTime, &policy.CreateTime, &policy.ModifyTime)
		if err != nil {
			d.logger.Errorf("sql: %s, err: %v", strSQL, err)
			return 0, nil, err
		}
		policy.Operation, err = d.operationStrToInfo(operationStr)
		if err != nil {
			d.logger.Errorf("sql: %s, err: %v", strSQL, err)
			return 0, nil, err
		}
		policies = append(policies, policy)
	}
	return count, policies, nil
}

// Create 新增策略
//
//nolint:lll
func (d *policy) Create(ctx context.Context, policys []interfaces.PolicyInfo, tx *sql.Tx) (err error) {
	curTime := common.GetCurrentMicrosecondTimestamp()
	type tmpPolicyInfo struct {
		ID           string
		ResourceID   string
		ResourceType string
		ResourceName string
		Ancestors    string
		AccessorID   string
		AccessorType interfaces.AccessorType
		AccessorName string
		Operation    string
		Condition    string
		EndTime      int64
	}

	createPolicys := []tmpPolicyInfo{}
	for i := range policys {
		var operationStr string
		operationStr, err = d.operationInfoToString(policys[i].Operation)
		if err != nil {
			d.logger.Errorln(err)
			return err
		}
		var ancestorsStr string
		ancestorsStr, err = d.ancestorsInfoToString(policys[i].Ancestors)
		if err != nil {
			return err
		}
		tmpPolicy := tmpPolicyInfo{
			ID:           policys[i].ID,
			ResourceID:   policys[i].ResourceID,
			ResourceType: policys[i].ResourceType,
			ResourceName: policys[i].ResourceName,
			Ancestors:    ancestorsStr,
			AccessorID:   policys[i].AccessorID,
			AccessorType: policys[i].AccessorType,
			AccessorName: policys[i].AccessorName,
			Operation:    operationStr,
			Condition:    policys[i].Condition,
			EndTime:      policys[i].EndTime,
		}
		createPolicys = append(createPolicys, tmpPolicy)
	}

	var valuesStr []string
	var inserts []any
	// 批量插入
	for i := range createPolicys {
		valuesStr = append(valuesStr, "(?,?,?,?,?,?,?,?,?,?,?,?,?)")
		inserts = append(inserts, createPolicys[i].ID, createPolicys[i].ResourceID, createPolicys[i].ResourceType, createPolicys[i].ResourceName, createPolicys[i].Ancestors, createPolicys[i].AccessorID, createPolicys[i].AccessorType,
			createPolicys[i].AccessorName, createPolicys[i].Operation, createPolicys[i].Condition, createPolicys[i].EndTime, curTime, curTime)
	}
	if len(valuesStr) == 0 {
		return
	}
	valueStr := strings.Join(valuesStr, ",")

	strSQL := "insert into " + common.GetDBName(databaseName) +
		".t_policy(f_id, f_resource_id, f_resource_type, f_resource_name, f_ancestors, f_accessor_id, f_accessor_type, f_accessor_name, f_operation, f_condition, f_end_time, f_create_time, f_modify_time) values " + valueStr

	_, err = tx.Exec(strSQL, inserts...)
	if err != nil {
		d.logger.Errorf("sql: %s, err: %v", strSQL, err)
		return err
	}
	return
}

// Update 更新策略
func (d *policy) Update(ctx context.Context, policys []interfaces.PolicyInfo, tx *sql.Tx) (err error) {
	curTime := common.GetCurrentMicrosecondTimestamp()
	var operationStr string
	for i := range policys {
		operationStr, err = d.operationInfoToString(policys[i].Operation)
		if err != nil {
			d.logger.Errorln(err)
			return err
		}
		strSQL := "update " + common.GetDBName(databaseName) + ".t_policy set  f_operation = ?, f_end_time = ?, f_condition = ?, f_modify_time = ? where f_id = ?"
		_, err = tx.Exec(strSQL, operationStr, policys[i].EndTime, policys[i].Condition, curTime, policys[i].ID)
		if err != nil {
			d.logger.Errorf("Update sql: %s, err: %v", strSQL, err)
			return err
		}
	}
	return
}

// Delete 删除策略
func (d *policy) Delete(ctx context.Context, ids []string) (err error) {
	if len(ids) == 0 {
		return
	}
	IDsSet, IDsGroup := getFindInSetSQL(ids)
	strSQL := "delete from " + common.GetDBName(databaseName) + ".t_policy where f_id in (" + IDsSet + ")"
	_, err = d.db.Exec(strSQL, IDsGroup...)
	if err != nil {
		d.logger.Errorf("sql: %s, err: %v", strSQL, err)
		return err
	}
	return
}

// 获取资源策略
func (d *policy) GetByResourceIDs(ctx context.Context, resourceType string, resourceIDs []string) (policiesMap map[string][]interfaces.PolicyInfo, err error) {
	policiesMap = make(map[string][]interfaces.PolicyInfo)
	IDsSet, IDsGroup := getFindInSetSQL(resourceIDs)
	strSQL := "select f_id, f_resource_id, f_resource_type, f_resource_name, f_ancestors, f_accessor_id, f_accessor_type, f_accessor_name, f_operation, f_condition, f_end_time, f_create_time from " +
		common.GetDBName(databaseName) + ".t_policy where f_resource_id in (" + IDsSet + ") and f_resource_type = ?"
	var inserts []any
	inserts = append(inserts, IDsGroup...)
	inserts = append(inserts, resourceType)

	rows, err := d.db.Query(strSQL, inserts...)
	if err != nil {
		d.logger.Errorf("GetByResourceIDs sql: %s, err: %v", strSQL, err)
		return nil, err
	}
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				d.logger.Errorln(rowsErr)
			}
		}
	}()

	for rows.Next() {
		var policy interfaces.PolicyInfo
		var operationStr string
		var ancestorsStr string
		err := rows.Scan(&policy.ID, &policy.ResourceID, &policy.ResourceType,
			&policy.ResourceName, &ancestorsStr, &policy.AccessorID, &policy.AccessorType, &policy.AccessorName,
			&operationStr, &policy.Condition, &policy.EndTime, &policy.CreateTime)
		if err != nil {
			d.logger.Errorf("GetByResourceIDs sql: %s, err: %v", strSQL, err)
			return nil, err
		}
		policy.Operation, err = d.operationStrToInfo(operationStr)
		if err != nil {
			d.logger.Errorf("GetByResourceIDs sql: %s, err: %v", strSQL, err)
			return nil, err
		}
		policy.Ancestors, err = d.ancestorsStrToInfo(ancestorsStr)
		if err != nil {
			d.logger.Errorf("GetByResourceIDs sql: %s, err: %v", strSQL, err)
			return nil, err
		}
		policiesMap[policy.ResourceID] = append(policiesMap[policy.ResourceID], policy)
	}
	return policiesMap, nil
}

// 获取资源策略
func (d *policy) GetByPolicyIDs(ctx context.Context, policyIDs []string) (policies map[string]interfaces.PolicyInfo, err error) {
	policies = make(map[string]interfaces.PolicyInfo)
	if len(policyIDs) == 0 {
		return policies, nil
	}

	IDsSet, IDsGroup := getFindInSetSQL(policyIDs)
	strSQL := "select f_id, f_resource_id, f_resource_type, f_resource_name, f_ancestors, f_accessor_id, f_accessor_type, f_accessor_name, f_operation, f_condition, f_end_time, f_create_time from " +
		common.GetDBName(databaseName) + ".t_policy where f_id in (" + IDsSet + ")"
	rows, err := d.db.Query(strSQL, IDsGroup...)
	if err != nil {
		d.logger.Errorf("GetByPolicyIDs sql: %s, err: %v", strSQL, err)
		return nil, err
	}
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				d.logger.Errorln(rowsErr)
			}
		}
		if closeErr := rows.Close(); closeErr != nil {
			d.logger.Errorln(closeErr)
		}
	}()

	for rows.Next() {
		var policy interfaces.PolicyInfo
		var operationStr string
		var ancestorsStr string
		err := rows.Scan(&policy.ID, &policy.ResourceID, &policy.ResourceType,
			&policy.ResourceName, &ancestorsStr, &policy.AccessorID, &policy.AccessorType, &policy.AccessorName,
			&operationStr, &policy.Condition, &policy.EndTime, &policy.CreateTime)
		if err != nil {
			d.logger.Errorf("GetByPolicyIDs sql: %s, err: %v", strSQL, err)
			return nil, err
		}
		policy.Operation, err = d.operationStrToInfo(operationStr)
		if err != nil {
			d.logger.Errorf("GetByPolicyIDs sql: %s, err: %v", strSQL, err)
			return nil, err
		}
		policy.Ancestors, err = d.ancestorsStrToInfo(ancestorsStr)
		if err != nil {
			d.logger.Errorf("GetByPolicyIDs sql: %s, err: %v", strSQL, err)
			return nil, err
		}
		policies[policy.ID] = policy
	}
	return policies, nil
}

// UpdateResourceAncestors 更新资源实例祖先信息
func (d *policy) UpdateResourceAncestors(ctx context.Context, resourceID, resourceType string, ancestors []interfaces.Ancestor) error {
	d.logger.Debugf("UpdateResourceAncestors resourceID: %s, resourceType: %s, ancestorsLen: %d", resourceID, resourceType, len(ancestors))
	ancestorsStr, err := d.ancestorsInfoToString(ancestors)
	if err != nil {
		return err
	}
	strSQL := "update " + common.GetDBName(databaseName) + ".t_policy set f_ancestors = ? where f_resource_id = ? and f_resource_type = ?"
	_, err = d.db.Exec(strSQL, ancestorsStr, resourceID, resourceType)
	if err != nil {
		d.logger.Errorf("UpdateResourceAncestors sql: %s, err: %v", strSQL, err)
		return err
	}
	return nil
}

// DeleteByResourceIDs 删除策略 根据资源id删除策略
func (d *policy) DeleteByResourceIDs(ctx context.Context, resources []interfaces.PolicyDeleteResourceInfo) error {
	if len(resources) == 0 {
		return nil
	}

	var params []any
	var whereSQL string
	for i, resource := range resources {
		whereSQL += " (f_resource_type = ? and f_resource_id = ?) "
		params = append(params, resource.Type, resource.ID)
		if i != len(resources)-1 {
			whereSQL += " or "
		}
	}

	strSQL := "delete from " + common.GetDBName(databaseName) + ".t_policy where " + whereSQL
	_, err := d.db.Exec(strSQL, params...)
	if err != nil {
		d.logger.Errorf("DeleteByResourceIDs sql: %s, err: %v", strSQL, err)
		return err
	}
	return nil
}

// DeleteByAccessorIDs 删除策略 根据访问者id删除策略
func (d *policy) DeleteByAccessorIDs(accessorIDs []string) error {
	if len(accessorIDs) == 0 {
		return nil
	}
	IDsSet, IDsGroup := getFindInSetSQL(accessorIDs)
	strSQL := "delete from " + common.GetDBName(databaseName) + ".t_policy where f_accessor_id in (" + IDsSet + ")"
	_, err := d.db.Exec(strSQL, IDsGroup...)
	if err != nil {
		d.logger.Errorf("DeleteByAccessorIDs sql: %s, err: %v", strSQL, err)
		return err
	}
	return nil
}

func (d *policy) UpdateAccessorName(accessorID, name string) error {
	strSQL := "update " + common.GetDBName(databaseName) + ".t_policy set f_accessor_name = ? where f_accessor_id = ?"
	_, err := d.db.Exec(strSQL, name, accessorID)
	if err != nil {
		d.logger.Errorf("UpdateAccessorName sql: %s, err: %v", strSQL, err)
		return err
	}
	return nil
}

// UpdateResourceName 更新资源实例名称
func (d *policy) UpdateResourceName(ctx context.Context, resourceID, resourceType, name string) error {
	d.logger.Debugf("UpdateResourceName resourceID: %s, resourceType: %s, name: %s", resourceID, resourceType, name)
	strSQL := "update " + common.GetDBName(databaseName) + ".t_policy set f_resource_name = ? where f_resource_id = ? and f_resource_type = ?"
	_, err := d.db.Exec(strSQL, name, resourceID, resourceType)
	if err != nil {
		d.logger.Errorf("UpdateResourceName sql: %s, err: %v", strSQL, err)
		return err
	}
	return nil
}

// DeleteByEndTime 删除过期策略
func (d *policy) DeleteByEndTime(curTime int64) (err error) {
	dbName := common.GetDBName(databaseName)
	d.logger.Infof("DeleteByEndTime dbName: %s, curTime: %d", dbName, curTime)
	sqlStr := "delete from " + dbName + ".t_policy where f_end_time < ? and f_end_time != -1"
	if _, err := d.db.Exec(sqlStr, curTime); err != nil {
		d.logger.Errorln(err, sqlStr, curTime)
		return err
	}
	return nil
}

// 获取访问者策略
//
//nolint:gocritic,gocyclo
func (d *policy) GetAccessorPolicy(ctx context.Context, param interfaces.AccessorPolicyParam) (count int, policies []interfaces.PolicyInfo, err error) {
	dbName := common.GetDBName(databaseName)
	var countRows *sql.Rows

	var countArgs []any
	countStr := "select count(1) from " + dbName + ".t_policy where f_accessor_id = ? and f_accessor_type = ?"
	countArgs = append(countArgs, param.AccessorID, param.AccessorType)
	if len(param.ResourceType) > 0 {
		countStr += " and f_resource_type = ?"
		countArgs = append(countArgs, param.ResourceType)
	}
	if len(param.ResourceID) > 0 {
		countStr += " and f_resource_id = ?"
		countArgs = append(countArgs, param.ResourceID)
	}

	countRows, err = d.db.Query(countStr, countArgs...)
	if err != nil {
		d.logger.Errorln(err)
		return 0, nil, err
	}

	for countRows.Next() {
		err = countRows.Scan(&count)
		if err != nil {
			d.logger.Errorln(err)
			return 0, nil, err
		}
	}

	if countRows != nil {
		if countRowsErr := countRows.Err(); countRowsErr != nil {
			d.logger.Errorln(countRowsErr)
		}
		if closeErr := countRows.Close(); closeErr != nil {
			d.logger.Errorln(closeErr)
		}
	}

	args := []any{param.AccessorID, param.AccessorType}
	sqlStr := "select f_id, f_resource_id, f_resource_type, f_resource_name, f_operation, f_condition, f_ancestors, f_end_time, f_create_time from " + dbName +
		".t_policy where f_accessor_id = ? and f_accessor_type = ?"

	if len(param.ResourceType) > 0 {
		sqlStr += " and f_resource_type = ?"
		args = append(args, param.ResourceType)
	}
	if len(param.ResourceID) > 0 {
		sqlStr += " and f_resource_id = ?"
		args = append(args, param.ResourceID)
	}

	if param.Limit == -1 {
		sqlStr += " order by f_modify_time desc, f_primary_id desc"
	} else {
		sqlStr += " order by f_modify_time desc, f_primary_id desc limit ? offset ?"
		args = append(args, param.Limit, param.Offset)
	}

	rows, err := d.db.Query(sqlStr, args...)
	if err != nil {
		d.logger.Errorln(err, sqlStr, args)
		return 0, nil, err
	}

	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				d.logger.Errorln(rowsErr)
			}

			if closeErr := rows.Close(); closeErr != nil {
				d.logger.Errorln(closeErr)
			}
		}
	}()

	for rows.Next() {
		var policy interfaces.PolicyInfo
		var operationStr string
		var ancestorsStr string
		err = rows.Scan(&policy.ID, &policy.ResourceID, &policy.ResourceType,
			&policy.ResourceName, &operationStr, &policy.Condition, &ancestorsStr, &policy.EndTime, &policy.CreateTime)
		if err != nil {
			d.logger.Errorf("sql: %s, err: %v", sqlStr, err)
			return
		}
		policy.Operation, err = d.operationStrToInfo(operationStr)
		if err != nil {
			d.logger.Errorf("sql: %s, err: %v", sqlStr, err)
			return
		}
		policy.Ancestors, err = d.ancestorsStrToInfo(ancestorsStr)
		if err != nil {
			d.logger.Errorf("sql: %s, err: %v", sqlStr, err)
			return
		}
		policies = append(policies, policy)
	}
	return count, policies, nil
}

func (d *policy) operationInfoToString(operationInfo interfaces.PolicyOperation) (resp string, err error) {
	result := make(map[string]any)
	allowResult := make([]any, 0, len(operationInfo.Allow))
	for i := range operationInfo.Allow {
		allowItem := make(map[string]any)
		allowItem["id"] = operationInfo.Allow[i].ID
		obligations := make([]map[string]any, 0, len(operationInfo.Allow[i].Obligations))
		for j := range operationInfo.Allow[i].Obligations {
			obligationItem := make(map[string]any)
			obligationItem["type_id"] = operationInfo.Allow[i].Obligations[j].TypeID
			obligationItem["id"] = operationInfo.Allow[i].Obligations[j].ID
			obligationItem["value"] = operationInfo.Allow[i].Obligations[j].Value
			obligations = append(obligations, obligationItem)
		}
		allowItem["obligations"] = obligations
		allowResult = append(allowResult, allowItem)
	}

	denyResult := make([]any, 0, len(operationInfo.Deny))
	for i := range operationInfo.Deny {
		denyItem := make(map[string]any)
		denyItem["id"] = operationInfo.Deny[i].ID
		denyResult = append(denyResult, denyItem)
	}

	result["allow"] = allowResult
	result["deny"] = denyResult
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		d.logger.Errorf("json.Marshal: %v", err)
		return "", err
	}
	resp = string(jsonBytes)
	return
}

// ancestorsInfoToString 将祖先信息转换为字符串
func (d *policy) ancestorsInfoToString(ancestors []interfaces.Ancestor) (string, error) {
	ancestorsJson := []any{}
	if len(ancestors) != 0 {
		for i := range ancestors {
			ancestors := map[string]string{
				"id":   ancestors[i].ID,
				"type": ancestors[i].Type,
				"name": ancestors[i].Name,
			}
			ancestorsJson = append(ancestorsJson, ancestors)
		}
	}

	d.logger.Infof("ancestorsJson: %v", ancestorsJson)
	ancestorsJsonBytes, err := json.Marshal(ancestorsJson)
	if err != nil {
		return "", err
	}
	return string(ancestorsJsonBytes), nil
}

// ancestorsStrToInfo 将祖先信息字符串转换为结构体
func (d *policy) ancestorsStrToInfo(ancestorsStr string) (resp []interfaces.Ancestor, err error) {
	if ancestorsStr == "" {
		return []interfaces.Ancestor{}, nil
	}
	var ancestorsJson []any
	err = json.Unmarshal([]byte(ancestorsStr), &ancestorsJson)
	if err != nil {
		d.logger.Errorf("json.Unmarshal: %v", err)
		return
	}
	ancestors := []interfaces.Ancestor{}
	for _, v := range ancestorsJson {
		item := v.(map[string]any)
		ancestor := interfaces.Ancestor{
			ID:   item["id"].(string),
			Type: item["type"].(string),
			Name: item["name"].(string),
		}
		ancestors = append(ancestors, ancestor)
	}
	return ancestors, nil
}

//nolint:dupl
func (d *policy) operationStrToInfo(operationStr string) (resp interfaces.PolicyOperation, err error) {
	var jsonReq map[string]any
	err = json.Unmarshal([]byte(operationStr), &jsonReq)
	if err != nil {
		d.logger.Errorf("json.Unmarshal: %v", err)
		return
	}
	allowJson := jsonReq["allow"].([]any)
	denyJson := jsonReq["deny"].([]any)
	allow := []interfaces.PolicyOperationItem{}
	deny := []interfaces.PolicyOperationItem{}
	for _, v := range allowJson {
		item := v.(map[string]any)
		allowItem := interfaces.PolicyOperationItem{
			ID: item["id"].(string),
		}
		// 历史数据没有obligations,代码里直接兼容
		obligationsJson, ok := item["obligations"]
		if ok {
			allowItem.Obligations = d.getObligations(obligationsJson)
		}
		allow = append(allow, allowItem)
	}

	for _, v := range denyJson {
		item := v.(map[string]any)
		denyItem := interfaces.PolicyOperationItem{
			ID: item["id"].(string),
		}
		deny = append(deny, denyItem)
	}
	return interfaces.PolicyOperation{
		Allow: allow,
		Deny:  deny,
	}, nil
}

func (d *policy) getObligations(obligationsJson any) (result []interfaces.PolicyObligationItem) {
	obligations := obligationsJson.([]any)
	for _, v := range obligations {
		obligationMap := v.(map[string]any)
		obligationID := obligationMap["id"].(string)
		obligationValue := obligationMap["value"]
		obligation := interfaces.PolicyObligationItem{
			TypeID: obligationMap["type_id"].(string),
			ID:     obligationID,
			Value:  obligationValue,
		}
		result = append(result, obligation)
	}
	return
}

//nolint:lll,dupl
func (d *policy) GetResourcePolicies(ctx context.Context, params interfaces.ResourcePolicyPagination) (count int, policies []interfaces.PolicyInfo, err error) {
	var countRows *sql.Rows
	countRows, err = d.db.Query("select count(1) from "+common.GetDBName(databaseName)+".t_policy where f_resource_id = ? and f_resource_type = ?", params.ResourceID, params.ResourceType)
	if err != nil {
		d.logger.Errorln(err)
		return 0, nil, err
	}

	for countRows.Next() {
		err = countRows.Scan(&count)
		if err != nil {
			d.logger.Errorln(err)
			return 0, nil, err
		}
	}

	if countRows != nil {
		if countRowsErr := countRows.Err(); countRowsErr != nil {
			d.logger.Errorln(countRowsErr)
		}
		if closeErr := countRows.Close(); closeErr != nil {
			d.logger.Errorln(closeErr)
		}
	}

	var rows *sql.Rows
	strSQL := "select f_id, f_resource_id, f_resource_type, f_resource_name, f_accessor_id, f_accessor_type, f_accessor_name, f_operation, f_condition, f_end_time, f_create_time, f_modify_time from " + common.GetDBName(databaseName) +
		".t_policy where f_resource_id = ? and f_resource_type = ? order by f_modify_time desc, f_primary_id desc limit ? offset ?"
	rows, err = d.db.Query(strSQL, params.ResourceID, params.ResourceType, params.Limit, params.Offset)
	if err != nil {
		d.logger.Errorln(err)
		return 0, nil, err
	}

	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				d.logger.Errorln(rowsErr)
			}

			if closeErr := rows.Close(); closeErr != nil {
				d.logger.Errorln(closeErr)
			}
		}
	}()

	for rows.Next() {
		var policy interfaces.PolicyInfo
		var operationStr string
		err := rows.Scan(&policy.ID, &policy.ResourceID, &policy.ResourceType,
			&policy.ResourceName, &policy.AccessorID, &policy.AccessorType, &policy.AccessorName,
			&operationStr, &policy.Condition, &policy.EndTime, &policy.CreateTime, &policy.ModifyTime)
		if err != nil {
			d.logger.Errorf("sql: %s, err: %v", strSQL, err)
			return 0, nil, err
		}
		policy.Operation, err = d.operationStrToInfo(operationStr)
		if err != nil {
			d.logger.Errorf("sql: %s, err: %v", strSQL, err)
			return 0, nil, err
		}
		policies = append(policies, policy)
	}
	return count, policies, nil
}
