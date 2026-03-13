// Package dbaccess 数据访问层
package dbaccess

import (
	"context"
	"strings"
	"sync"

	jsoniter "github.com/json-iterator/go"

	"github.com/kweaver-ai/proton-rds-sdk-go/sqlx"

	"Authorization/common"
	"Authorization/interfaces"
)

type policyCalc struct {
	db     *sqlx.DB
	logger common.Logger
}

var (
	policyCalcOnce    sync.Once
	policyCalcService *policyCalc
)

// NewPolicyCalc 创建数据库对象
func NewPolicyCalc() *policyCalc {
	policyCalcOnce.Do(func() {
		policyCalcService = &policyCalc{
			db:     dbPool,
			logger: common.NewLogger(),
		}
	})
	return policyCalcService
}

// 通过资源类型和访问令牌获取配置
func (d *policyCalc) GetPoliciesByResourceTypeAndAccessToken(ctx context.Context, resourceTypeID string, accessToken []string) (policies []interfaces.PolicyInfo, err error) {
	accessorSet, accessorIDGroup := getFindInSetSQL(accessToken)
	var paramList []any
	paramList = append(paramList, resourceTypeID)
	paramList = append(paramList, accessorIDGroup...)

	// 查询策略
	strSQL := "select f_id, f_resource_id, f_resource_type, f_resource_name, f_accessor_id, f_accessor_type, f_accessor_name, f_operation, f_condition, f_end_time, f_create_time, f_modify_time from " +
		common.GetDBName(databaseName) + ".t_policy where f_resource_type = ? and f_accessor_id in (" + accessorSet + ") "
	rows, err := d.db.Query(strSQL, paramList...)
	if err != nil {
		d.logger.Errorf("sql: %s, err: %v", strSQL, err)
		return nil, err
	}

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

	curTime := common.GetCurrentMicrosecondTimestamp()
	for rows.Next() {
		var policy interfaces.PolicyInfo
		var operationStr string
		err = rows.Scan(&policy.ID, &policy.ResourceID,
			&policy.ResourceType, &policy.ResourceName,
			&policy.AccessorID, &policy.AccessorType, &policy.AccessorName, &operationStr,
			&policy.Condition, &policy.EndTime, &policy.CreateTime, &policy.ModifyTime)
		if err != nil {
			d.logger.Errorf("sql: %s, err: %v", strSQL, err)
			return nil, err
		}
		// 过滤已过期的策略：f_end_time != -1 且 f_end_time < 当前时间
		if policy.EndTime != -1 && policy.EndTime < curTime {
			continue
		}
		policy.Rules, err = d.rulesStrToInfo(operationStr)
		if err != nil {
			d.logger.Errorf("sql: %s, err: %v", strSQL, err)
			return nil, err
		}
		policies = append(policies, policy)
	}
	return
}

// 批量资源和访问令牌获取配置
func (d *policyCalc) GetPoliciesByResourcesAndAccessToken(ctx context.Context, resourceInfo []interfaces.ResourceInfo, accessToken []string) (policies []interfaces.PolicyInfo, err error) {
	if len(resourceInfo) == 0 {
		return nil, nil
	}
	var paramList []any
	resourceIDMap := make(map[string]bool)
	resourceIDMap["*"] = true
	for _, resource := range resourceInfo {
		resourceIDMap[resource.ID] = true
	}

	resourceType := resourceInfo[0].Type

	resourceIDs := make([]string, 0, len(resourceIDMap))
	for resourceID := range resourceIDMap {
		resourceIDs = append(resourceIDs, resourceID)
	}

	resourceIDSet, resourceIDGroup := getFindInSetSQL(resourceIDs)
	accessorSet, accessorIDGroup := getFindInSetSQL(accessToken)

	paramList = append(paramList, resourceIDGroup...)
	paramList = append(paramList, resourceType)
	paramList = append(paramList, accessorIDGroup...)
	time0 := common.GetCurrentMicrosecondTimestamp()
	d.logger.Debugf("GetPoliciesByResourcesAndAccessToken, start, time: %d", time0)
	strSQL := "select f_id, f_resource_id, f_resource_type, f_resource_name, f_accessor_id, f_accessor_type, f_accessor_name, f_operation, f_condition, f_end_time, f_create_time, f_modify_time from " +
		common.GetDBName(databaseName) + ".t_policy where f_resource_id in (" + resourceIDSet + ") and f_resource_type = ?  and f_accessor_id in (" + accessorSet + ") "
	rows, err := d.db.Query(strSQL, paramList...)
	if err != nil {
		d.logger.Errorf("sql: %s, err: %v", strSQL, err)
		return nil, err
	}
	time1 := common.GetCurrentMicrosecondTimestamp()
	d.logger.Debugf("GetPoliciesByResourcesAndAccessToken, end, cost: %d", time1-time0)

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

	policies = make([]interfaces.PolicyInfo, 0)
	curTime := common.GetCurrentMicrosecondTimestamp()
	for rows.Next() {
		var policy interfaces.PolicyInfo
		var operationStr string
		err = rows.Scan(&policy.ID, &policy.ResourceID, &policy.ResourceType,
			&policy.ResourceName, &policy.AccessorID, &policy.AccessorType, &policy.AccessorName,
			&operationStr, &policy.Condition, &policy.EndTime, &policy.CreateTime, &policy.ModifyTime)
		if err != nil {
			d.logger.Errorf("sql: %s, err: %v", strSQL, err)
			return nil, err
		}
		// 过滤已过期的策略：f_end_time != -1 且 f_end_time < 当前时间
		if policy.EndTime != -1 && policy.EndTime < curTime {
			continue
		}
		policy.Rules, err = d.rulesStrToInfo(operationStr)
		if err != nil {
			d.logger.Errorf("sql: %s, err: %v", strSQL, err)
			return nil, err
		}
		policies = append(policies, policy)
	}
	time2 := common.GetCurrentMicrosecondTimestamp()
	d.logger.Debugf("GetPoliciesByResourcesAndAccessToken, fill data, count: %d, cost: %d", len(policies), time2-time1)
	return
}

func getFindInSetSQL(value []string) (setStr string, args []any) {
	set := make([]string, 0)
	for _, v := range value {
		set = append(set, "?")
		args = append(args, v)
	}
	setStr = strings.Join(set, ",")

	return
}

// 通过访问令牌，批量获取资源类型的配置(不包含具体资源实例的配置，只包含资源实例id为*的配置)
func (d *policyCalc) GetPoliciesByResourceTypes(ctx context.Context, resourceTypes, accessToken []string) (policies []interfaces.PolicyInfo, err error) {
	var paramList []any
	accessorSet, accessorIDGroup := getFindInSetSQL(accessToken)
	resourceTypeSet, resourceTypeGroup := getFindInSetSQL(resourceTypes)
	paramList = append(paramList, resourceTypeGroup...)
	paramList = append(paramList, accessorIDGroup...)

	strSQL := "select f_id, f_resource_id, f_resource_type, f_resource_name, f_accessor_id, f_accessor_type, f_accessor_name, f_operation, f_condition, f_end_time, f_create_time, f_modify_time from " +
		common.GetDBName(databaseName) + ".t_policy where f_resource_id = '*' and f_resource_type in (" + resourceTypeSet + ") and f_accessor_id in (" + accessorSet + ") "
	rows, err := d.db.Query(strSQL, paramList...)
	if err != nil {
		d.logger.Errorf("sql: %s, err: %v", strSQL, err)
		return nil, err
	}
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

	curTime := common.GetCurrentMicrosecondTimestamp()
	for rows.Next() {
		var policy interfaces.PolicyInfo
		var operationStr string
		err = rows.Scan(&policy.ID, &policy.ResourceID, &policy.ResourceType, &policy.ResourceName, &policy.AccessorID,
			&policy.AccessorType, &policy.AccessorName, &operationStr, &policy.Condition,
			&policy.EndTime, &policy.CreateTime, &policy.ModifyTime)
		if err != nil {
			d.logger.Errorf("sql: %s, err: %v", strSQL, err)
			return nil, err
		}
		// 过滤已过期的策略：f_end_time != -1 且 f_end_time < 当前时间
		if policy.EndTime != -1 && policy.EndTime < curTime {
			continue
		}
		policy.Rules, err = d.rulesStrToInfo(operationStr)
		if err != nil {
			d.logger.Errorf("sql: %s, err: %v", strSQL, err)
			return nil, err
		}
		policies = append(policies, policy)
	}
	return
}

func (d *policyCalc) rulesStrToInfo(rulesStr string) (resp interfaces.PolicyRules, err error) {
	var jsonReq map[string]any
	err = jsoniter.Unmarshal([]byte(rulesStr), &jsonReq)
	if err != nil {
		d.logger.Errorf("json.Unmarshal: %v", err)
		return
	}
	allowJson := jsonReq["allow"].([]any)
	denyJson := jsonReq["deny"].([]any)
	allow := make([]interfaces.PolicyRuleItem, 0, len(allowJson))
	deny := make([]interfaces.PolicyRuleItem, 0, len(denyJson))

	for _, v := range allowJson {
		allow = append(allow, d.parseConditionItem(v))
	}
	for _, v := range denyJson {
		deny = append(deny, d.parseConditionItem(v))
	}
	return interfaces.PolicyRules{
		Allow: allow,
		Deny:  deny,
	}, nil
}

// emptyCondition 复用以避免每次 parseConditionItem 分配空 map
var emptyCondition = map[string]any{}

// parseConditionItem 解析单个 { condition, operations }
func (d *policyCalc) parseConditionItem(v any) interfaces.PolicyRuleItem {
	item := v.(map[string]any)
	// 解析 operations
	opsList := item["operations"].([]any)
	operations := make([]interfaces.PolicyOperationItem, 0, len(opsList))
	for _, opV := range opsList {
		opMap := opV.(map[string]any)
		opItem := interfaces.PolicyOperationItem{ID: opMap["id"].(string)}
		// 历史数据没有obligations,代码里直接兼容
		if obl, ok := opMap["obligations"]; ok {
			opItem.Obligations = d.getObligations(obl)
		}
		operations = append(operations, opItem)
	}
	condition := item["condition"]
	if condition == nil {
		condition = emptyCondition
	}
	return interfaces.PolicyRuleItem{Condition: condition, Operations: operations}
}

func (d *policyCalc) getObligations(obligationsJson any) (result []interfaces.PolicyObligationItem) {
	obligations := obligationsJson.([]any)
	result = make([]interfaces.PolicyObligationItem, 0, len(obligations))
	for _, v := range obligations {
		obligationMap := v.(map[string]any)
		result = append(result, interfaces.PolicyObligationItem{
			TypeID: obligationMap["type_id"].(string),
			ID:     obligationMap["id"].(string),
			Value:  obligationMap["value"],
		})
	}
	return
}
