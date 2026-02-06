// Package logics perm Anyshare 业务逻辑层 -文档权限
package logics

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"sync"

	gerrors "github.com/kweaver-ai/go-lib/error"
	"github.com/kweaver-ai/proton-rds-sdk-go/sqlx"
	"github.com/satori/uuid"

	"Authorization/common"
	errors "Authorization/error"
	"Authorization/interfaces"
)

var (
	policyOnce      sync.Once
	policySingleton *policy
)

const (
	rootDepID          = "00000000-0000-0000-0000-000000000000"
	authorizeOperation = "authorize"
)

const (
	_ int = iota
	i18nAccessorRoleNotFound
)

type policy struct {
	db                    interfaces.DBPolicy
	logger                common.Logger
	role                  interfaces.LogicsRole
	resourceType          interfaces.LogicsResourceType
	resourceTypeHierarchy interfaces.LogicsResourceTypeHierarchy
	userMgmt              interfaces.DrivenUserMgnt
	pool                  *sqlx.DB
	event                 interfaces.LogicsEvent
	policyCalc            interfaces.LogicsPolicyCalc
	obligationType        interfaces.ObligationType
	obligation            interfaces.LogicsObligation
	i18n                  *common.I18n
}

// NewPolicy 创建新的NewPolicy对象
func NewPolicy() *policy {
	policyOnce.Do(func() {
		policySingleton = &policy{
			db:                    dbPolicy,
			pool:                  dbPool,
			role:                  NewLogicsRole(),
			resourceType:          NewResourceType(),
			resourceTypeHierarchy: NewResourceTypeHierarchy(),
			obligationType:        NewObligationType(),
			obligation:            NewObligation(),
			logger:                common.NewLogger(),
			userMgmt:              dnUserMgnt,
			event:                 NewEvent(),
			policyCalc:            NewPolicyCalc(),
			i18n: common.NewI18n(common.I18nMap{
				i18nAccessorRoleNotFound: {
					simplifiedChinese:  "角色不存在",
					traditionalChinese: "角色不存在",
					americanEnglish:    "The Role does not exist.",
				},
			}),
		}
		// 注册用户删除处理函数
		policySingleton.event.RegisterUserDeleted(policySingleton.deletePolicyByAccessorID)
		// 注册用户组删除处理函数
		policySingleton.event.RegisterUserGroupDeleted(policySingleton.deletePolicyByAccessorID)
		// 注册部门删除处理函数
		policySingleton.event.RegisterDepartmentDeleted(policySingleton.deletePolicyByAccessorID)
		// 应用账户 删除处理函数
		policySingleton.event.RegisterAppDeleted(policySingleton.deletePolicyByAccessorID)
		// 组织名称修改处理函数
		policySingleton.event.RegisterOrgNameModified(policySingleton.updatePolicyAccessorName)
		// 应用账户改名
		policySingleton.event.RegisterAppNameModified(policySingleton.updateAppName)

		// 注册角色删除处理函数
		policySingleton.event.RegisterRoleDeleted(policySingleton.deletePolicyByAccessorID)

		// 注册角色名称修改处理函数
		policySingleton.event.RegisterRoleNameModified(policySingleton.updatePolicyAccessorName)
	})
	return policySingleton
}

func (r *policy) getOperationNameByLanguage(operationName []interfaces.OperationName, language string) (name string) {
	for _, name := range operationName {
		if strings.EqualFold(name.Language, language) {
			return name.Value
		}
	}
	return
}

type operationInfo struct {
	id   string
	name string
}

// GetPagination 获取策略
func (r *policy) GetPagination(ctx context.Context, visitor *interfaces.Visitor, params interfaces.PolicyPagination) (count int, policies []interfaces.PolicyInfo, err error) {
	// 权限检查 visitor
	tmpMap := make(map[string]map[string]bool)
	tmpMap[params.ResourceType] = make(map[string]bool)
	tmpMap[params.ResourceType][params.ResourceID] = true
	err = r.checkVisitorAuthorize(ctx, visitor, tmpMap)
	if err != nil {
		r.logger.Errorf("GetPagination: checkVisitorAuthorize %v", err)
		return
	}

	count, policies, err = r.db.GetPagination(ctx, params)
	if err != nil {
		r.logger.Errorf("GetPagination db: %v", err)
		return
	}

	if len(policies) == 0 {
		return
	}

	// 获取操作名称
	operationInfoMap, err := r.getResourceTypesOperation(ctx, visitor, []string{policies[0].ResourceType})
	if err != nil {
		r.logger.Errorf("GetPagination getResourceTypesOperation: %v", err)
		return
	}
	opeInfo := operationInfoMap[policies[0].ResourceType]
	r.logger.Debugf("GetPagination opeInfo: %v", opeInfo)
	// 获取用户的部门信息
	parentDepsMap, err := r.getParentDeps(ctx, policies)
	if err != nil {
		return 0, nil, err
	}
	for i := range policies {
		// 填充父部门信息
		if policies[i].AccessorType == interfaces.AccessorUser || policies[i].AccessorType == interfaces.AccessorDepartment {
			if policies[i].AccessorID == rootDepID {
				policies[i].ParentDeps = [][]interfaces.Department{}
			} else {
				policies[i].ParentDeps = parentDepsMap[policies[i].AccessorID]
			}
		} else {
			policies[i].ParentDeps = [][]interfaces.Department{}
		}

		// 填充操作名称和矫正权限顺序
		policies[i].Operation.Allow,
			policies[i].Operation.Deny = r.sortOperationAndFillName(policies[i].Operation.Allow, policies[i].Operation.Deny, opeInfo)
	}

	return count, policies, nil
}

/*
getResourceTypesOperation 获取资源类型操作
返回参数:

	opeNamesMap: 资源类型操作信息
		key: 资源类型
		value: 操作信息数组，按照注册资源时的顺序，数组元素为 operationName 结构体，结构体包含操作ID和操作名称
	err: 错误信息
*/
func (r *policy) getResourceTypesOperation(ctx context.Context, visitor *interfaces.Visitor, resourceTypes []string) (
	operationInfoMap map[string][]operationInfo, err error,
) {
	// 获取资源类型
	resourceTypeInfoMap, err := r.resourceType.GetByIDsInternal(ctx, resourceTypes)
	if err != nil {
		r.logger.Errorf("getResourceTypesOperation GetByIDsInternal err: %v", err)
		return
	}

	operationInfoMap = make(map[string][]operationInfo, len(resourceTypes))
	for _, resourceType := range resourceTypes {
		resourceTypeInfo, ok := resourceTypeInfoMap[resourceType]
		if !ok {
			err = gerrors.NewError(gerrors.PublicBadRequest, fmt.Sprintf("resource type %s not found", resourceType))
			return
		}

		opeNames := make([]operationInfo, 0, len(resourceTypeInfo.Operation))
		for _, operation := range resourceTypeInfo.Operation {
			// 根据国际化获取名称
			name := r.getOperationNameByLanguage(operation.Name, visitor.Language)
			opeNames = append(opeNames, operationInfo{
				id:   operation.ID,
				name: name,
			})
		}
		operationInfoMap[resourceType] = opeNames
	}
	return
}

/*
getParentDeps 获取父部门信息
返回参数:

	parentDepsMap: 父部门信息
		key: 用户ID或部门ID
		value: 父部门信息，数组元素为 Department 结构体
	err: 错误信息
*/
func (r *policy) getParentDeps(ctx context.Context, policies []interfaces.PolicyInfo) (parentDepsMap map[string][][]interfaces.Department, err error) {
	parentDepsMap = make(map[string][][]interfaces.Department)
	userIDs := []string{}
	for i := range policies {
		if policies[i].AccessorType == interfaces.AccessorUser {
			userIDs = append(userIDs, policies[i].AccessorID)
		} else if policies[i].AccessorType == interfaces.AccessorDepartment {
			if policies[i].AccessorID == rootDepID {
				continue
			}
			// 获取部门父部门信息
			var dep []interfaces.Department
			dep, err = r.userMgmt.GetParentDepartmentsByDepartmentID(ctx, policies[i].AccessorID)
			if err != nil {
				return parentDepsMap, err
			}
			parentDepsMap[policies[i].AccessorID] = [][]interfaces.Department{dep}
		}
	}
	// 批量获取用户父部门信息
	userIDs = common.Distinct(userIDs)
	if len(userIDs) == 0 {
		return parentDepsMap, nil
	}
	userInfoMaps, err := r.userMgmt.BatchGetUserInfoByID(ctx, userIDs)
	if err != nil {
		return
	}
	for userID := range userInfoMaps {
		parentDepsMap[userID] = userInfoMaps[userID].ParentDeps
	}
	return parentDepsMap, nil
}

// checkPolicyOperationValid 检查策略操作是否合法
/*
    一条策略， 检查操作枚举是否定义
	一条策略， 拒绝和允许不能同时为空
	一条策略， 拒绝和允许不能存在相同的操作
*/
func (d *policy) checkPolicyOperationValid(operationIDMap map[string]bool, policy *interfaces.PolicyInfo) (err error) {
	// 允许和拒绝不能同时为空
	if len(policy.Operation.Allow) == 0 && len(policy.Operation.Deny) == 0 {
		err = gerrors.NewError(gerrors.PublicBadRequest, "allow and deny cannot be empty at the same time")
		return
	}

	denyOperationMap := make(map[string]bool)
	for _, deny := range policy.Operation.Deny {
		if !operationIDMap[deny.ID] {
			err = gerrors.NewError(gerrors.PublicBadRequest, fmt.Sprintf("operation %s not found", deny.ID))
			return
		}
		denyOperationMap[deny.ID] = true
	}
	for _, allow := range policy.Operation.Allow {
		if !operationIDMap[allow.ID] {
			err = gerrors.NewError(gerrors.PublicBadRequest, fmt.Sprintf("operation %s not found", allow.ID))
			return
		}
		if denyOperationMap[allow.ID] {
			err = gerrors.NewError(gerrors.PublicBadRequest, fmt.Sprintf("allow and deny cannot have the same operation: %s", allow.ID))
			return
		}
	}
	return
}

// cmpPolicy 检查新策略是否有修改
// 以下条件认为无修改，结果返回true
// 1. 过期时间是否相等
// 2. 新策略的操作是否是旧策略的子集, 且操作的义务是否相等
func (d *policy) cmpPolicy(old, newInfo *interfaces.PolicyInfo) (isSame bool) {
	// 过期时间是否相等
	if old.EndTime != newInfo.EndTime {
		isSame = false
		return
	}

	oldAllowMap := make(map[string]interfaces.PolicyOperationItem, len(old.Operation.Allow))
	oldDenyMap := make(map[string]interfaces.PolicyOperationItem, len(old.Operation.Deny))
	for _, deny := range old.Operation.Deny {
		oldDenyMap[deny.ID] = deny
	}
	for _, allow := range old.Operation.Allow {
		oldAllowMap[allow.ID] = allow
	}

	// 新权限 如果是 旧的子集，则无变化
	isSame = true
	for _, allow := range newInfo.Operation.Allow {
		// 旧权限如果不存在 ， 则有变化
		oldAllow, ok := oldAllowMap[allow.ID]
		if !ok {
			isSame = false
			return
		}
		// 旧权限存在， 但是义务有变化， 则有变化
		if !reflect.DeepEqual(allow.Obligations, oldAllow.Obligations) {
			isSame = false
			return
		}
	}

	for _, deny := range newInfo.Operation.Deny {
		// 旧权限如果不存在 ， 则有变化
		oldDeny, ok := oldDenyMap[deny.ID]
		if !ok {
			isSame = false
			return
		}
		// 旧权限存在， 但是义务有变化， 则有变化
		if !reflect.DeepEqual(deny.Obligations, oldDeny.Obligations) {
			isSame = false
			return
		}
	}
	return
}

// mergeNewPolicy 合并策略 , 操作、到期时间
// 为了方便，新的的义务会覆盖旧的义务
func (d *policy) mergeNewPolicy(old, newInfo *interfaces.PolicyInfo) (newPolicy interfaces.PolicyInfo) {
	allowMap := make(map[string]interfaces.PolicyOperationItem)
	denyMap := make(map[string]interfaces.PolicyOperationItem)
	// 合并权限时 拒绝优先
	for _, deny := range newInfo.Operation.Deny {
		denyMap[deny.ID] = deny
	}

	for _, deny := range old.Operation.Deny {
		// 已存在使用newInfo的义务
		if _, ok := denyMap[deny.ID]; ok {
			continue
		}
		// 不存在使用旧的义务
		denyMap[deny.ID] = deny
	}

	for _, allow := range newInfo.Operation.Allow {
		if _, ok := denyMap[allow.ID]; ok {
			continue
		}
		allowMap[allow.ID] = allow
	}

	for _, allow := range old.Operation.Allow {
		if _, ok := denyMap[allow.ID]; ok {
			continue
		}
		if _, ok := allowMap[allow.ID]; ok {
			continue
		}
		allowMap[allow.ID] = allow
	}

	allow := []interfaces.PolicyOperationItem{}
	deny := []interfaces.PolicyOperationItem{}
	for key := range allowMap {
		allow = append(allow, allowMap[key])
	}
	for key := range denyMap {
		deny = append(deny, denyMap[key])
	}

	newPolicy.ID = old.ID
	newPolicy.Operation.Allow = allow
	newPolicy.Operation.Deny = deny
	newPolicy.ResourceID = old.ResourceID
	newPolicy.ResourceType = old.ResourceType
	newPolicy.ResourceName = old.ResourceName
	newPolicy.AccessorID = old.AccessorID
	newPolicy.AccessorType = old.AccessorType
	newPolicy.AccessorName = old.AccessorName
	newPolicy.EndTime = d.calcMinEndTime(old.EndTime, newInfo.EndTime)
	return
}

// calcMinEndTime 计算最小过期时间
func (d *policy) calcMinEndTime(endTime1, endTime2 int64) int64 {
	// -1 表示永久有效
	if endTime1 == -1 {
		return endTime2
	}
	if endTime2 == -1 {
		return endTime1
	}

	if endTime1 < endTime2 {
		return endTime1
	}
	return endTime2
}

// getResourceTypeOperations 获取资源类型上的操作
func (d *policy) getResourceTypeOperations(ctx context.Context, resourceTypeMap map[string][]string) (resourceTypeOperationIDMap map[string]map[string]bool, err error) {
	resourceTypeIDs := make([]string, 0, len(resourceTypeMap))
	for resourceTypeID := range resourceTypeMap {
		resourceTypeIDs = append(resourceTypeIDs, resourceTypeID)
	}
	resourceTypeInfoMap, err := d.resourceType.GetByIDsInternal(ctx, resourceTypeIDs)
	if err != nil {
		d.logger.Errorf("getResourceTypeOperations: %v", err)
		return nil, err
	}

	// 保存 资源类型上面的操作集合
	resourceTypeOperationIDMap = make(map[string]map[string]bool, len(resourceTypeIDs))
	for _, resourceTypeID := range resourceTypeIDs {
		resourceTypeInfo, ok := resourceTypeInfoMap[resourceTypeID]
		if !ok {
			des := fmt.Sprintf("resource type %s not found", resourceTypeID)
			err = gerrors.NewError(gerrors.PublicBadRequest, des)
			d.logger.Errorf("getResourceTypeOperations: %v", err)
			return
		}

		operationIDMap := make(map[string]bool, len(resourceTypeInfo.Operation))
		for _, operation := range resourceTypeInfo.Operation {
			operationIDMap[operation.ID] = true
		}

		resourceTypeOperationIDMap[resourceTypeID] = operationIDMap
	}
	return resourceTypeOperationIDMap, nil
}

// Create 新增策略
//
//nolint:gocyclo,gocritic
func (d *policy) Create(ctx context.Context, visitor *interfaces.Visitor, policys []interfaces.PolicyInfo) (policyIDs []string, err error) {
	if len(policys) == 0 {
		return
	}

	// 保存资源类型和对应的资源实例ID
	resourceTypeMap := make(map[string][]string)
	tmpMap := make(map[string]map[string]bool)
	for _, policy := range policys {
		resourceTypeMap[policy.ResourceType] = append(resourceTypeMap[policy.ResourceType], policy.ResourceID)
		if _, ok := tmpMap[policy.ResourceType]; !ok {
			tmpMap[policy.ResourceType] = make(map[string]bool)
		}
		tmpMap[policy.ResourceType][policy.ResourceID] = true
	}

	// 检查是否有授权的权限
	err = d.checkVisitorAuthorize(ctx, visitor, tmpMap)
	if err != nil {
		d.logger.Errorf("Create: checkVisitorAuthorize %v", err)
		return
	}

	resourceTypeOperationIDMap, err := d.getResourceTypeOperations(ctx, resourceTypeMap)
	if err != nil {
		d.logger.Errorf("Create: %v", err)
		return
	}

	// 同一资源类型下，同一资源实例下，同一访问者不能重复
	// resourceIDpolicysDup [资源类型][资源实例ID][访问者ID]bool
	resourceIDpolicysDup := make(map[string]map[string]map[string]bool)
	accessorTypeMap := make(map[string]interfaces.AccessorType)
	accessorRoleMap := make(map[string]string)

	for _, policy := range policys {
		// 检查 到期时间
		err = checkEndTime(policy.EndTime)
		if err != nil {
			return
		}

		// 检查资源id和祖先
		err = d.checkResourceIDAndAncestors(ctx, visitor, &policy)
		if err != nil {
			return
		}

		if _, ok := resourceIDpolicysDup[policy.ResourceType]; !ok {
			resourceIDpolicysDup[policy.ResourceType] = make(map[string]map[string]bool)
		}
		if _, ok := resourceIDpolicysDup[policy.ResourceType][policy.ResourceID]; !ok {
			resourceIDpolicysDup[policy.ResourceType][policy.ResourceID] = make(map[string]bool)
		}
		// 同一资源类型下，同一资源实例下，同一访问者不能重复
		if resourceIDpolicysDup[policy.ResourceType][policy.ResourceID][policy.AccessorID] {
			des := fmt.Sprintf(" resource type %s, resource %s, accessor %s duplicate。resource.id %s, accessor.id %s",
				policy.ResourceType, policy.ResourceName, policy.AccessorName, policy.ResourceID, policy.AccessorID)
			err = gerrors.NewError(gerrors.PublicBadRequest, des)
			return
		}
		resourceIDpolicysDup[policy.ResourceType][policy.ResourceID][policy.AccessorID] = true

		if policy.AccessorType == interfaces.AccessorRole {
			accessorRoleMap[policy.AccessorID] = policy.AccessorName
		} else {
			if policy.AccessorID != rootDepID {
				accessorTypeMap[policy.AccessorID] = policy.AccessorType
			}
		}

		err = d.checkPolicyOperationValid(resourceTypeOperationIDMap[policy.ResourceType], &policy)
		if err != nil {
			return
		}
	}

	idNameMap, err := d.getAccessorName(ctx, visitor, accessorTypeMap, accessorRoleMap)
	if err != nil {
		d.logger.Errorf("Create: getAccessorName: %v", err)
		return
	}

	newPolicy, updatePolicy, policyIDsTmp, err := d.getCreateAndUpdatePolicy(ctx, policys, resourceTypeMap, idNameMap)
	if err != nil {
		return
	}

	// 如果创建和更新的策略都为空，则直接返回
	if len(newPolicy) == 0 && len(updatePolicy) == 0 {
		return policyIDsTmp, nil
	}

	var tx *sql.Tx
	tx, err = d.pool.Begin()
	if err != nil {
		return
	}
	// 异常时Rollback
	defer func() {
		switch err {
		case nil:
			// 提交事务
			err = tx.Commit()
			if err != nil {
				d.logger.Errorf("Create Transaction Commit Error:%v", err)
				return
			}
		default:
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				d.logger.Errorf("Create Rollback Error:%v", rollbackErr)
			}
		}
	}()
	// 写入数据库
	if len(newPolicy) > 0 {
		err = d.db.Create(ctx, newPolicy, tx)
		if err != nil {
			d.logger.Errorf("Create: %v", err)
			return
		}
	}
	if len(updatePolicy) > 0 {
		err = d.db.Update(ctx, updatePolicy, tx)
		if err != nil {
			d.logger.Errorf("Update: %v", err)
			return
		}
	}
	return policyIDsTmp, nil
}

// CreatePrivate 新增策略
//
//nolint:gocyclo,gocritic
func (d *policy) CreatePrivate(ctx context.Context, policys []interfaces.PolicyInfo) (err error) {
	d.logger.Debugf("CreatePrivate: len policys: %d", len(policys))
	if len(policys) == 0 {
		return
	}
	// 保存资源类型和对应的资源实例ID
	resourceTypeMap := make(map[string][]string)
	for _, policy := range policys {
		resourceTypeMap[policy.ResourceType] = append(resourceTypeMap[policy.ResourceType], policy.ResourceID)
	}

	resourceTypeOperationIDMap, err := d.getResourceTypeOperations(ctx, resourceTypeMap)
	if err != nil {
		d.logger.Errorf("CreatePrivate: getResourceTypeOperations: %v", err)
		return
	}

	// 同一资源类型下，同一资源实例下，同一访问者不能重复
	// resourceIDpolicysDup [资源类型][资源实例ID][访问者ID]bool
	resourceIDpolicysDup := make(map[string]map[string]map[string]bool)
	accessorTypeMap := make(map[string]interfaces.AccessorType)
	accessorRoleMap := make(map[string]string)

	for _, policy := range policys {
		// 检查 到期时间
		err = checkEndTime(policy.EndTime)
		if err != nil {
			d.logger.Errorf("CreatePrivate: checkEndTime: %v", err)
			return
		}

		// 检查资源id和祖先
		err = d.checkResourceIDAndAncestors(ctx, nil, &policy)
		if err != nil {
			return
		}

		if _, ok := resourceIDpolicysDup[policy.ResourceType]; !ok {
			resourceIDpolicysDup[policy.ResourceType] = make(map[string]map[string]bool)
		}
		if _, ok := resourceIDpolicysDup[policy.ResourceType][policy.ResourceID]; !ok {
			resourceIDpolicysDup[policy.ResourceType][policy.ResourceID] = make(map[string]bool)
		}
		// 同一资源类型下，同一资源实例下，同一访问者不能重复
		if resourceIDpolicysDup[policy.ResourceType][policy.ResourceID][policy.AccessorID] {
			des := fmt.Sprintf(" resource type %s, resource %s, accessor %s duplicate。resource.id %s, accessor.id %s",
				policy.ResourceType, policy.ResourceName, policy.AccessorName, policy.ResourceID, policy.AccessorID)
			err = gerrors.NewError(gerrors.PublicBadRequest, des)
			return
		}
		resourceIDpolicysDup[policy.ResourceType][policy.ResourceID][policy.AccessorID] = true

		if policy.AccessorType == interfaces.AccessorRole {
			accessorRoleMap[policy.AccessorID] = policy.AccessorName
		} else {
			if policy.AccessorID != rootDepID {
				accessorTypeMap[policy.AccessorID] = policy.AccessorType
			}
		}

		err = d.checkPolicyOperationValid(resourceTypeOperationIDMap[policy.ResourceType], &policy)
		if err != nil {
			d.logger.Errorf("CreatePrivate checkPolicyOperationValid : %v", err)
			return
		}
	}

	// 内部接口默认中文
	visitor := &interfaces.Visitor{
		Language: simplifiedChinese,
	}
	idNameMap, err := d.getAccessorName(ctx, visitor, accessorTypeMap, accessorRoleMap)
	if err != nil {
		d.logger.Errorf("CreatePrivate: getAccessorName: %v", err)
		return
	}

	newPolicy, updatePolicy, _, err := d.getCreateAndUpdatePolicy(ctx, policys, resourceTypeMap, idNameMap)
	if err != nil {
		return
	}

	// 如果创建和更新的策略都为空，则直接返回
	if len(newPolicy) == 0 && len(updatePolicy) == 0 {
		return
	}

	var tx *sql.Tx
	tx, err = d.pool.Begin()
	if err != nil {
		return
	}
	// 异常时Rollback
	defer func() {
		switch err {
		case nil:
			// 提交事务
			err = tx.Commit()
			if err != nil {
				d.logger.Errorf("CreatePrivate Transaction Commit Error:%v", err)
				return
			}
		default:
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				d.logger.Errorf("CreatePrivate Rollback Error:%v", rollbackErr)
			}
		}
	}()
	// 写入数据库
	if len(newPolicy) > 0 {
		err = d.db.Create(ctx, newPolicy, tx)
		if err != nil {
			d.logger.Errorf("Create: %v", err)
			return
		}
	}

	if len(updatePolicy) > 0 {
		err = d.db.Update(ctx, updatePolicy, tx)
		if err != nil {
			d.logger.Errorf("Update: %v", err)
			return
		}
	}
	return nil
}

// Update 更新策略
//
//nolint:gocyclo
func (d *policy) Update(ctx context.Context, visitor *interfaces.Visitor, policys []interfaces.PolicyInfo) (err error) {
	if len(policys) == 0 {
		return
	}

	policyIDs := []string{}
	idDupCheck := make(map[string]bool)
	for i := range policys {
		if idDupCheck[policys[i].ID] {
			err = gerrors.NewError(gerrors.PublicBadRequest, fmt.Sprintf("policy id duplicate, policy.id %s", policys[i].ID))
			return
		}
		idDupCheck[policys[i].ID] = true
		policyIDs = append(policyIDs, policys[i].ID)
		// 检查有效期
		err = checkEndTime(policys[i].EndTime)
		if err != nil {
			return
		}
	}

	oldPoliciesMap, err := d.db.GetByPolicyIDs(ctx, policyIDs)
	if err != nil {
		d.logger.Errorf("Update: %v", err)
		return
	}

	if len(oldPoliciesMap) == 0 {
		return
	}

	resourceTypeMap := make(map[string][]string)
	tmpMap := make(map[string]map[string]bool)
	for i := range oldPoliciesMap {
		resourceTypeMap[oldPoliciesMap[i].ResourceType] = append(resourceTypeMap[oldPoliciesMap[i].ResourceType], oldPoliciesMap[i].ResourceID)
		if _, ok := tmpMap[oldPoliciesMap[i].ResourceType]; !ok {
			tmpMap[oldPoliciesMap[i].ResourceType] = make(map[string]bool)
		}
		tmpMap[oldPoliciesMap[i].ResourceType][oldPoliciesMap[i].ResourceID] = true
	}

	// 检查是否有授权的权限
	err = d.checkVisitorAuthorize(ctx, visitor, tmpMap)
	if err != nil {
		d.logger.Errorf("Update: checkVisitorAuthorize %v", err)
		return
	}

	resourceTypeOperationIDMap, err := d.getResourceTypeOperations(ctx, resourceTypeMap)
	if err != nil {
		d.logger.Errorf("Update: %v", err)
		return
	}

	for i := range policys {
		// 策略不存在直接跳过
		if _, ok := oldPoliciesMap[policys[i].ID]; !ok {
			continue
		}
		err = d.checkPolicyOperationValid(resourceTypeOperationIDMap[oldPoliciesMap[policys[i].ID].ResourceType], &policys[i])
		if err != nil {
			d.logger.Errorf("Update: %v", err)
			return
		}

		// 如果策略存在，则更新
		oldPoliciesMap[policys[i].ID] = policys[i]
	}

	var tx *sql.Tx
	tx, err = d.pool.Begin()
	if err != nil {
		return
	}
	// 异常时Rollback
	defer func() {
		switch err {
		case nil:
			// 提交事务
			err = tx.Commit()
			if err != nil {
				d.logger.Errorf("Update Transaction Commit Error:%v", err)
				return
			}
		default:
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				d.logger.Errorf("Update Rollback Error:%v", rollbackErr)
			}
		}
	}()

	err = d.db.Update(ctx, policys, tx)
	if err != nil {
		d.logger.Errorf("Update: %v", err)
		return
	}
	return
}

// Delete 删除策略
func (d *policy) Delete(ctx context.Context, visitor *interfaces.Visitor, ids []string) (err error) {
	if len(ids) == 0 {
		return nil
	}

	oldPoliciesMap, err := d.db.GetByPolicyIDs(ctx, ids)
	if err != nil {
		d.logger.Errorf("Update: %v", err)
		return
	}

	tmpMap := make(map[string]map[string]bool)
	for i := range oldPoliciesMap {
		if _, ok := tmpMap[oldPoliciesMap[i].ResourceType]; !ok {
			tmpMap[oldPoliciesMap[i].ResourceType] = make(map[string]bool)
		}
		tmpMap[oldPoliciesMap[i].ResourceType][oldPoliciesMap[i].ResourceID] = true
	}

	// 检查是否有授权的权限
	err = d.checkVisitorAuthorize(ctx, visitor, tmpMap)
	if err != nil {
		d.logger.Errorf("Delete: checkVisitorAuthorize %v", err)
		return
	}

	return d.db.Delete(ctx, ids)
}

// DeleteByResourceIDs 删除策略 根据资源id删除策略
func (d *policy) DeleteByResourceIDs(ctx context.Context, resources []interfaces.PolicyDeleteResourceInfo) error {
	if len(resources) == 0 {
		return nil
	}

	return d.db.DeleteByResourceIDs(ctx, resources)
}

func (d *policy) deletePolicyByAccessorID(accessorID string) error {
	return d.db.DeleteByAccessorIDs([]string{accessorID})
}

// updatePolicyAccessorName 更新策略访问者名称
func (d *policy) updatePolicyAccessorName(id, name string) error {
	return d.db.UpdateAccessorName(id, name)
}

// updateAppName 更新应用账户名称
func (d *policy) updateAppName(info *interfaces.AppInfo) error {
	return d.db.UpdateAccessorName(info.ID, info.Name)
}

// UpdateResourceName 更新资源实例名称
func (d *policy) UpdateResourceName(ctx context.Context, resourceID, resourceType, name string) error {
	return d.db.UpdateResourceName(ctx, resourceID, resourceType, name)
}

// UpdateResourceAncestors 更新资源实例祖先信息
func (d *policy) UpdateResourceAncestors(ctx context.Context, resourceID, resourceType string, ancestors []interfaces.Ancestor) error {
	return d.db.UpdateResourceAncestors(ctx, resourceID, resourceType, ancestors)
}

// DeleteByEndTime 删除过期策略
func (d *policy) DeleteByEndTime(curTime int64) error {
	return d.db.DeleteByEndTime(curTime)
}

// InitPolicy 初始化策略
func (d *policy) InitPolicy(ctx context.Context, policys []interfaces.PolicyInfo) (err error) {
	if len(policys) == 0 {
		return
	}
	d.logger.Infof("InitPolicy policys start len: %d", len(policys))
	// 保存资源类型和对应的资源实例ID
	resourceTypeMap := make(map[string][]string)
	idNameMap := make(map[string]string)
	for i := range policys {
		resourceTypeMap[policys[i].ResourceType] = append(resourceTypeMap[policys[i].ResourceType], policys[i].ResourceID)
		if policys[i].AccessorID != rootDepID {
			idNameMap[policys[i].AccessorID] = policys[i].AccessorName
		}
	}

	newPolicy, updatePolicy, _, err := d.getCreateAndUpdatePolicy(ctx, policys, resourceTypeMap, idNameMap)
	if err != nil {
		return
	}

	if len(newPolicy) == 0 && len(updatePolicy) == 0 {
		return
	}

	var tx *sql.Tx
	tx, err = d.pool.Begin()
	if err != nil {
		return
	}
	// 异常时Rollback
	defer func() {
		switch err {
		case nil:
			// 提交事务
			err = tx.Commit()
			if err != nil {
				d.logger.Errorf("InitPolicy Transaction Commit Error:%v", err)
				return
			}
		default:
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				d.logger.Errorf("InitPolicy Rollback Error:%v", rollbackErr)
			}
		}
	}()
	// 写入数据库
	if len(newPolicy) > 0 {
		err = d.db.Create(ctx, newPolicy, tx)
		if err != nil {
			d.logger.Errorf("InitPolicy Create: %v", err)
			return
		}
	}

	if len(updatePolicy) > 0 {
		err = d.db.Update(ctx, updatePolicy, tx)
		if err != nil {
			d.logger.Errorf("InitPolicy Update: %v", err)
			return
		}
	}
	return nil
}

// 检查是否可以授权
// 1. 检查系统角色 超级管理员、系统管理员、安全管理员
// 2. 检查是否有授权的权限
func (d *policy) checkVisitorAuthorize(ctx context.Context, visitor *interfaces.Visitor, resourceTypeMap map[string]map[string]bool) (err error) {
	// 实名用户获取对应系统角色信息
	var roleTypes []interfaces.SystemRoleType
	if visitor.Type == interfaces.RealName {
		// 获取用户角色信息
		roleTypes, err = d.userMgmt.GetUserRolesByUserID(ctx, visitor.ID)
		if err != nil {
			return err
		}

		acceptRoleTypeMap := map[interfaces.SystemRoleType]bool{
			interfaces.SuperAdmin:    true,
			interfaces.SystemAdmin:   true,
			interfaces.SecurityAdmin: true,
		}
		// 访问者角色是否在检测范围内
		for _, roleType := range roleTypes {
			if acceptRoleTypeMap[roleType] {
				// 有超级管理员、系统管理员、安全管理员权限，则直接返回
				return
			}
		}
	}

	// 检查授权的权限
	operations := []string{authorizeOperation}
	for resourceType, resourceMap := range resourceTypeMap {
		for resourceID := range resourceMap {
			resource := interfaces.ResourceInfo{
				ID:   resourceID,
				Type: resourceType,
			}
			checkResult, err := d.policyCalc.Check(ctx, &resource, &interfaces.AccessorInfo{
				ID:   visitor.ID,
				Type: visitor.Type,
			}, operations, []interfaces.PolicCalcyIncludeType{})
			if err != nil {
				return err
			}
			// 有一个没有权限，则返回错误
			if !checkResult.Result {
				return gerrors.NewError(gerrors.PublicForbidden, "no permission to authorize")
			}
		}
	}
	return
}

// 获取访问者名称
// 1. 获取访问者类型名称
// 2. 获取访问者角色名称
func (d *policy) getAccessorName(ctx context.Context, visitor *interfaces.Visitor, accessorTypeMap map[string]interfaces.AccessorType, accessorRoleMap map[string]string) (
	idNameMap map[string]string, err error,
) {
	d.logger.Debugf("getAccessorName  len accessorRoleMap: %d, len accessorTypeMap: %d", len(accessorRoleMap), len(accessorTypeMap))
	idNameMap = make(map[string]string)
	if len(accessorTypeMap) > 0 {
		idNameMap, err = d.userMgmt.GetNameByAccessorIDs(ctx, accessorTypeMap)
		if err != nil {
			d.logger.Errorf("getAccessorName GetNameByAccessorIDs:%v", err)
			return
		}
	}

	if len(accessorRoleMap) > 0 {
		accessorRoleIDs := make([]string, 0, len(accessorRoleMap))
		for accessorRoleID := range accessorRoleMap {
			accessorRoleIDs = append(accessorRoleIDs, accessorRoleID)
		}

		// 批量获取角色信息
		var roleInfoMap map[string]interfaces.RoleInfo
		roleInfoMap, err = d.role.GetRolesByIDs(ctx, accessorRoleIDs)
		if err != nil {
			d.logger.Errorf("Create: %v", err)
			return
		}
		// 检查角色是否存在，不存在角色放到一个数组里，然后抛错返回
		notExistRoleIDs := make([]string, 0)
		for accessorRoleID := range accessorRoleMap {
			if _, ok := roleInfoMap[accessorRoleID]; !ok {
				notExistRoleIDs = append(notExistRoleIDs, accessorRoleID)
			}
		}

		if len(notExistRoleIDs) > 0 {
			err = gerrors.NewError(errors.RoleNotFound, d.i18n.Load(i18nAccessorRoleNotFound, visitor.Language),
				gerrors.SetDetail(map[string]any{"ids": notExistRoleIDs}))
			return
		}

		for _, roleInfo := range roleInfoMap {
			idNameMap[roleInfo.ID] = roleInfo.Name
		}
	}
	d.logger.Debugf("getAccessorName end len (idNameMap): %d", len(idNameMap))
	return
}

// 获取创建和更新的策略
// 1. 获取资源类型和对应的资源实例ID
// 2. 获取访问者名称
// 3. 获取创建和更新的策略, 过滤掉无变化的策略
//
//nolint:gocritic
func (d *policy) getCreateAndUpdatePolicy(ctx context.Context, policys []interfaces.PolicyInfo,
	resourceTypeMap map[string][]string, idNameMap map[string]string) (
	newPolicy []interfaces.PolicyInfo, updatePolicy []interfaces.PolicyInfo, policyIDs []string, err error,
) {
	resourceTypeOldPoliciesMap := make(map[string]map[string][]interfaces.PolicyInfo)
	for resourceTypeID, resourceIDs := range resourceTypeMap {
		var oldPoliciesMap map[string][]interfaces.PolicyInfo
		oldPoliciesMap, err = d.db.GetByResourceIDs(ctx, resourceTypeID, resourceIDs)
		if err != nil {
			d.logger.Errorf("getCreateAndUpdatePolicy  GetByResourceIDs error: %v", err)
			return
		}
		// 如果资源实例下没有策略，则跳过
		if len(oldPoliciesMap) == 0 {
			continue
		}
		resourceTypeOldPoliciesMap[resourceTypeID] = oldPoliciesMap
	}

	// oldResourceAccessorsPoliciesMap [资源类型][资源实例ID][访问者ID]策略
	oldResourceAccessorsPoliciesMap := make(map[string]map[string]map[string]interfaces.PolicyInfo)
	for resourceTypeID, resourceIDPolicieMap := range resourceTypeOldPoliciesMap {
		if _, ok := oldResourceAccessorsPoliciesMap[resourceTypeID]; !ok {
			oldResourceAccessorsPoliciesMap[resourceTypeID] = make(map[string]map[string]interfaces.PolicyInfo)
		}
		for resourceID, policys := range resourceIDPolicieMap {
			if _, ok := oldResourceAccessorsPoliciesMap[resourceTypeID][resourceID]; !ok {
				oldResourceAccessorsPoliciesMap[resourceTypeID][resourceID] = make(map[string]interfaces.PolicyInfo)
			}
			for _, policy := range policys {
				oldResourceAccessorsPoliciesMap[resourceTypeID][resourceID][policy.AccessorID] = policy
			}
		}
	}

	// 如果访问者之前策略存在，且策略条件为空，则合并
	newPolicy = []interfaces.PolicyInfo{}
	updatePolicy = []interfaces.PolicyInfo{}
	policyIDs = make([]string, 0, len(policys))
	for _, policy := range policys {
		var oldPolicy interfaces.PolicyInfo
		var exist bool
		if resourceIDPolicieMap, ok := oldResourceAccessorsPoliciesMap[policy.ResourceType]; ok {
			if policysMap, ok := resourceIDPolicieMap[policy.ResourceID]; ok {
				if _, ok := policysMap[policy.AccessorID]; ok {
					exist = true
					oldPolicy = policysMap[policy.AccessorID]
				}
			}
		}
		if exist {
			// 检查是否无变化
			policyIDs = append(policyIDs, oldPolicy.ID)
			isSame := d.cmpPolicy(&oldPolicy, &policy)
			if isSame {
				continue
			}
			tmpPolicy := d.mergeNewPolicy(&oldPolicy, &policy)
			updatePolicy = append(updatePolicy, tmpPolicy)
		} else {
			// 新加的策略， 生成唯一标识
			policy.ID = uuid.NewV4().String()
			policyIDs = append(policyIDs, policy.ID)
			// 名称找不到, 使用访问者id
			if _, ok := idNameMap[policy.AccessorID]; !ok {
				policy.AccessorName = policy.AccessorID
			} else {
				policy.AccessorName = idNameMap[policy.AccessorID]
			}
			newPolicy = append(newPolicy, policy)
		}
	}
	return
}

// GetAccessorPolicy 获取访问者策略
//
//nolint:gocritic,gocyclo
func (d *policy) GetAccessorPolicy(ctx context.Context, visitor *interfaces.Visitor, param interfaces.AccessorPolicyParam) (count int, policies []interfaces.PolicyInfo,
	includeResp interfaces.PolicyIncludeResp, err error,
) {
	// 必传参数校验
	if len(param.AccessorID) == 0 {
		err = gerrors.NewError(gerrors.PublicBadRequest, "accessor_id is required")
		return
	}

	if len(param.ResourceID) > 0 && len(param.ResourceType) == 0 {
		err = gerrors.NewError(gerrors.PublicBadRequest, "resource_type is required when resource_id is not empty")
		return
	}

	count, policies, err = d.db.GetAccessorPolicy(ctx, param)
	if err != nil {
		d.logger.Errorf("GetAccessorPolicy: %v", err)
		return
	}
	if len(policies) == 0 {
		return
	}
	d.logger.Debugf("GetAccessorPolicy policies len: %d", len(policies))
	resourceTypeMap := make(map[string]bool)
	resourceTypeIDs := make([]string, 0, len(policies))
	for i := range policies {
		if _, ok := resourceTypeMap[policies[i].ResourceType]; ok {
			continue
		}
		resourceTypeMap[policies[i].ResourceType] = true
		resourceTypeIDs = append(resourceTypeIDs, policies[i].ResourceType)
	}
	operationInfoMap, err := d.getResourceTypesOperation(ctx, visitor, resourceTypeIDs)
	if err != nil {
		d.logger.Errorf("GetAccessorPolicy getResourceTypesOperation: %v", err)
		return
	}

	// 收集义务类型和义务
	obligationTypeMap := make(map[string]bool)
	obligationMap := make(map[string]bool)

	for i := range policies {
		opeNames := operationInfoMap[policies[i].ResourceType]
		// 填充操作名称和矫正权限顺序
		policies[i].Operation.Allow,
			policies[i].Operation.Deny = d.sortOperationAndFillName(policies[i].Operation.Allow, policies[i].Operation.Deny, opeNames)

		// 收集义务类型和义务
		for _, ope := range policies[i].Operation.Allow {
			for _, obligation := range ope.Obligations {
				obligationTypeMap[obligation.TypeID] = true
				if obligation.ID != "" {
					obligationMap[obligation.ID] = true
				}
			}
		}
	}

	if len(obligationTypeMap) == 0 && len(obligationMap) == 0 {
		return
	}

	// 获取义务类型信息
	obligationTypes, err := d.obligationType.GetByIDSInternal(ctx, obligationTypeMap)
	if err != nil {
		d.logger.Errorf("GetResourcePolicy GetByIDSInternal: %v", err)
		return
	}
	validObligationTypesMap := make(map[string]bool, len(obligationTypes))
	for i := range obligationTypes {
		validObligationTypesMap[obligationTypes[i].ID] = true
	}

	// 获取义务信息
	obligations, err := d.obligation.GetByIDSInternal(ctx, obligationMap)
	if err != nil {
		d.logger.Errorf("GetResourcePolicy GetByIDSInternal: %v", err)
		return
	}

	// 义务类型不存在，该义务也无效
	validObligationMap := make(map[string]bool, len(obligations))
	for _, obligation := range obligations {
		if validObligationTypesMap[obligation.TypeID] {
			validObligationMap[obligation.ID] = true
		}
	}

	// 去掉策略中无效的义务类型和义务
	for i := range policies {
		for j := range policies[i].Operation.Allow {
			obligationsTmp := policies[i].Operation.Allow[j].Obligations
			var obligationsValid []interfaces.PolicyObligationItem
			for _, obligation := range obligationsTmp {
				// 义务类型不存在 直接过滤
				if !validObligationTypesMap[obligation.TypeID] {
					continue
				}

				// 使用了义务，但是义务不存在
				if obligation.ID != "" && !validObligationMap[obligation.ID] {
					continue
				}
				obligationsValid = append(obligationsValid, obligation)
			}
			policies[i].Operation.Allow[j].Obligations = obligationsValid
		}
	}

	// 查询include信息
	for _, includeType := range param.Include {
		switch includeType {
		case interfaces.PolicyIncludeObligationType:
			includeResp.ObligationTypes = append(includeResp.ObligationTypes, obligationTypes...)
		case interfaces.PolicyIncludeObligation:
			includeResp.Obligations = append(includeResp.Obligations, obligations...)
		}
	}

	return
}

/*
sortOperationAndFillName  填充操作名称和矫正操作顺序
*/
func (d *policy) sortOperationAndFillName(allow, deny []interfaces.PolicyOperationItem, operationInfos []operationInfo) (
	sortedAllow, sortedDeny []interfaces.PolicyOperationItem,
) {
	allIDMap := make(map[string]interfaces.PolicyOperationItem, len(allow))
	denyIDMap := make(map[string]interfaces.PolicyOperationItem, len(deny))
	for _, operation := range allow {
		allIDMap[operation.ID] = operation
	}
	for _, operation := range deny {
		denyIDMap[operation.ID] = operation
	}

	sortedAllow = make([]interfaces.PolicyOperationItem, 0, len(allow)+len(deny))
	sortedDeny = make([]interfaces.PolicyOperationItem, 0, len(allow)+len(deny))

	for _, opeName := range operationInfos {
		if _, ok := allIDMap[opeName.id]; ok {
			sortedAllow = append(sortedAllow, interfaces.PolicyOperationItem{
				ID:          opeName.id,
				Name:        opeName.name,
				Obligations: allIDMap[opeName.id].Obligations,
			})
		}
		if _, ok := denyIDMap[opeName.id]; ok {
			sortedDeny = append(sortedDeny, interfaces.PolicyOperationItem{
				ID:          opeName.id,
				Name:        opeName.name,
				Obligations: denyIDMap[opeName.id].Obligations,
			})
		}
	}
	return
}

// GetResourcePolicy 获取资源策略分页
//
//nolint:gocyclo
func (p *policy) GetResourcePolicy(ctx context.Context, visitor *interfaces.Visitor, params interfaces.ResourcePolicyPagination) (count int,
	policies []interfaces.PolicyInfo, includeResp interfaces.PolicyIncludeResp, err error,
) {
	// 权限检查 visitor
	tmpMap := make(map[string]map[string]bool)
	tmpMap[params.ResourceType] = make(map[string]bool)
	tmpMap[params.ResourceType][params.ResourceID] = true
	err = p.checkVisitorAuthorize(ctx, visitor, tmpMap)
	if err != nil {
		p.logger.Errorf("GetResourcePolicy: checkVisitorAuthorize %v", err)
		return
	}

	paramsTmp := interfaces.PolicyPagination{
		ResourceID:   params.ResourceID,
		ResourceType: params.ResourceType,
		Offset:       params.Offset,
		Limit:        params.Limit,
	}
	count, policies, err = p.db.GetPagination(ctx, paramsTmp)
	if err != nil {
		p.logger.Errorf("GetResourcePolicy db GetPagination: %v", err)
		return
	}

	if len(policies) == 0 {
		return
	}

	// 获取操作名称
	operationInfoMap, err := p.getResourceTypesOperation(ctx, visitor, []string{policies[0].ResourceType})
	if err != nil {
		p.logger.Errorf("GetResourcePolicy getResourceTypesOperation: %v", err)
		return
	}
	opeInfo := operationInfoMap[policies[0].ResourceType]
	p.logger.Debugf("GetResourcePolicy opeInfo: %v", opeInfo)
	// 获取用户的部门信息
	parentDepsMap, err := p.getParentDeps(ctx, policies)
	if err != nil {
		return 0, nil, includeResp, err
	}

	// 收集义务类型和义务
	obligationTypeMap := make(map[string]bool)
	obligationMap := make(map[string]bool)

	for i := range policies {
		// 填充父部门信息
		if policies[i].AccessorType == interfaces.AccessorUser || policies[i].AccessorType == interfaces.AccessorDepartment {
			if policies[i].AccessorID == rootDepID {
				policies[i].ParentDeps = [][]interfaces.Department{}
			} else {
				policies[i].ParentDeps = parentDepsMap[policies[i].AccessorID]
			}
		} else {
			policies[i].ParentDeps = [][]interfaces.Department{}
		}

		// 填充操作名称和矫正权限顺序
		policies[i].Operation.Allow,
			policies[i].Operation.Deny = p.sortOperationAndFillName(policies[i].Operation.Allow, policies[i].Operation.Deny, opeInfo)

		// 收集义务类型和义务
		for _, ope := range policies[i].Operation.Allow {
			for _, obligation := range ope.Obligations {
				obligationTypeMap[obligation.TypeID] = true
				if obligation.ID != "" {
					obligationMap[obligation.ID] = true
				}
			}
		}
	}

	if len(obligationTypeMap) == 0 && len(obligationMap) == 0 {
		return
	}

	// 获取义务类型信息
	obligationTypes, err := p.obligationType.GetByIDSInternal(ctx, obligationTypeMap)
	if err != nil {
		p.logger.Errorf("GetResourcePolicy GetByIDSInternal: %v", err)
		return
	}
	validObligationTypesMap := make(map[string]bool, len(obligationTypes))
	for i := range obligationTypes {
		validObligationTypesMap[obligationTypes[i].ID] = true
	}

	// 获取义务信息
	obligations, err := p.obligation.GetByIDSInternal(ctx, obligationMap)
	if err != nil {
		p.logger.Errorf("GetResourcePolicy GetByIDSInternal: %v", err)
		return
	}

	// 义务类型不存在，该义务也无效
	validObligationMap := make(map[string]bool, len(obligations))
	for _, obligation := range obligations {
		if validObligationTypesMap[obligation.TypeID] {
			validObligationMap[obligation.ID] = true
		}
	}

	// 去掉策略中无效的义务类型和义务
	for i := range policies {
		for j := range policies[i].Operation.Allow {
			obligationsTmp := policies[i].Operation.Allow[j].Obligations
			var obligationsValid []interfaces.PolicyObligationItem
			for _, obligation := range obligationsTmp {
				// 义务类型不存在 直接过滤
				if !validObligationTypesMap[obligation.TypeID] {
					continue
				}

				// 使用了义务，但是义务不存在
				if obligation.ID != "" && !validObligationMap[obligation.ID] {
					continue
				}
				obligationsValid = append(obligationsValid, obligation)
			}
			policies[i].Operation.Allow[j].Obligations = obligationsValid
		}
	}

	// 补全include信息
	for _, includeType := range params.Include {
		switch includeType {
		case interfaces.PolicyIncludeObligationType:
			includeResp.ObligationTypes = append(includeResp.ObligationTypes, obligationTypes...)
		case interfaces.PolicyIncludeObligation:
			includeResp.Obligations = append(includeResp.Obligations, obligations...)
		}
	}

	return count, policies, includeResp, nil
}

func (p *policy) checkResourceIDAndAncestors(ctx context.Context, visitor *interfaces.Visitor, policy *interfaces.PolicyInfo) (err error) {
	// 是否设置过层级关系
	hasHierarchy, err := p.resourceTypeHierarchy.HasHierarchy(ctx, visitor, policy.ResourceType)
	if err != nil {
		p.logger.Errorf("checkResourceIDAndAncestors HasHierarchy: %v", err)
		return err
	}

	// 设置过层级关系，但祖先未设置，抛错
	if hasHierarchy && !policy.HasAncestors {
		des := fmt.Sprintf(" resource %s  ancestors must set ", policy.ResourceID)
		err = gerrors.NewError(gerrors.PublicBadRequest, des)
		return
	}

	// 检查资源类型 id 是否 包含 /, 表示 父节点下层配置
	if strings.Contains(policy.ResourceID, "/") {
		// 没有祖先 抛错
		if len(policy.Ancestors) == 0 {
			des := fmt.Sprintf(" resource %s  ancestors must set ", policy.ResourceID)
			err = gerrors.NewError(gerrors.PublicBadRequest, des)
			return
		}
		resourceIDParts := strings.Split(policy.ResourceID, "/")
		// 资源id 格式错误
		numTmp := 2
		if len(resourceIDParts) != numTmp {
			des := fmt.Sprintf(" resource id %s is invalid", policy.ResourceID)
			err = gerrors.NewError(gerrors.PublicBadRequest, des)
			return
		}
		// 资源id 格式错误
		if resourceIDParts[1] != "*" {
			des := fmt.Sprintf(" resource id %s is invalid", policy.ResourceID)
			err = gerrors.NewError(gerrors.PublicBadRequest, des)
			return
		}

		parentResourceID := policy.Ancestors[len(policy.Ancestors)-1].ID
		// 直接父级资源id 不一致
		if resourceIDParts[0] != parentResourceID {
			des := fmt.Sprintf(" resource id:%s  ancestors  set error, parentResourceID %s", policy.ResourceID, parentResourceID)
			err = gerrors.NewError(gerrors.PublicBadRequest, des)
			return
		}
	}

	// 祖先id 不可为*, 只能为具体实例ID
	for _, ancestor := range policy.Ancestors {
		if ancestor.ID == "*" {
			des := fmt.Sprintf(" resource id:%s  ancestors set error, ancestor id can't be *", policy.ResourceID)
			err = gerrors.NewError(gerrors.PublicBadRequest, des)
			return
		}
	}
	return nil
}
