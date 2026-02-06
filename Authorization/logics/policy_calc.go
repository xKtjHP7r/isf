// Package logics perm Anyshare 业务逻辑层 -文档权限
package logics

import (
	"context"
	_ "embed" // embed
	"sort"
	"strings"
	"sync"

	"Authorization/common"
	"Authorization/interfaces"
)

var (
	policyCalcOnce      sync.Once
	policyCalcSingleton interfaces.LogicsPolicyCalc
)

const (
	superAdminRoleID        = "7dcfcc9c-ad02-11e8-aa06-000c29358ad6"
	systemAdminRoleID       = "d2bd2082-ad03-11e8-aa06-000c29358ad6"
	securityAdminRoleID     = "d8998f72-ad03-11e8-aa06-000c29358ad6"
	auditAdminRoleID        = "def246f2-ad03-11e8-aa06-000c29358ad6"
	organizationAdminRoleID = "e63e1c88-ad03-11e8-aa06-000c29358ad6"
	organizationAuditRoleID = "f06ac18e-ad03-11e8-aa06-000c29358ad6"
)

const (
	rootDepObligationPriority = 5
)

type resourcePerm struct {
	allow map[string][]policyObligationCalcItem
	deny  map[string]bool
}

type policyCalc struct {
	db           interfaces.DBPolicyCalc
	logger       common.Logger
	userMgnt     interfaces.DrivenUserMgnt
	role         interfaces.LogicsRole
	event        interfaces.LogicsEvent
	resourceType interfaces.LogicsResourceType
	obligation   interfaces.LogicsObligation
	// 义务优先级
	obligationPriority map[interfaces.AccessorType]int
}

// NewPolicyCalc 创建新的LogicsPolicyCalc对象
func NewPolicyCalc() interfaces.LogicsPolicyCalc {
	policyCalcOnce.Do(func() {
		policyCalcSingleton = &policyCalc{
			db:           dbPolicyCalc,
			logger:       common.NewLogger(),
			role:         NewLogicsRole(),
			userMgnt:     dnUserMgnt,
			event:        NewEvent(),
			resourceType: NewResourceType(),
			obligation:   NewObligation(),
			obligationPriority: map[interfaces.AccessorType]int{
				interfaces.AccessorUser:       1,
				interfaces.AccessorApp:        1,
				interfaces.AccessorRole:       2,
				interfaces.AccessorGroup:      3,
				interfaces.AccessorDepartment: 4,
			},
		}
	})
	return policyCalcSingleton
}

// check 检查
func (d *policyCalc) Check(ctx context.Context, resource *interfaces.ResourceInfo, accessor *interfaces.AccessorInfo, operation []string, include []interfaces.PolicCalcyIncludeType) (
	checkResult interfaces.CheckResult, err error,
) {
	d.logger.Debugf("Check start, resource:%+v, accessor:%+v, operation: %v", *resource, *accessor, operation)
	// 查找当前用户的组织架构，角色
	accessTokens, err := d.getAccessorIDs(ctx, accessor)
	if err != nil {
		return
	}

	resources := d.createAncestorsResource(ctx, resource)

	// 获取资源策略
	policies, err := d.db.GetPoliciesByResourcesAndAccessToken(ctx, resources, accessTokens)
	if err != nil {
		d.logger.Errorf("Check GetPoliciesByResourcesAndAccessToken  err:%v", err)
		return checkResult, err
	}
	policyMap := make(map[string][]interfaces.PolicyInfo)
	time0 := common.GetCurrentMicrosecondTimestamp()
	d.logger.Debugf("Check policyMap, start, time: %d", time0)
	for i := range policies {
		policyMap[policies[i].ResourceID] = append(policyMap[policies[i].ResourceID], policies[i])
	}
	time1 := common.GetCurrentMicrosecondTimestamp()
	d.logger.Debugf("CheckpolicyMap, end, cost: %d", time1-time0)

	// 计算每个单个资源ID的权限
	// resourcePermMap[单个资源ID]操作权限
	resourcePermMap := make(map[string]resourcePerm)
	for resourceID, policies := range policyMap {
		if len(policies) == 0 {
			continue
		}
		resourcePermMap[resourceID] = d.calcOneResourcePerm(resourceID, policies)
	}

	// 计算资源继承的权限
	allowMap, denyMap, allowMapWithObligation := d.calcResourceInheritedOperation(resources, resourcePermMap)
	checkResult.Result = true
	// 所有的操作 都被允许 返回true，否则返回false
	for _, v := range operation {
		if denyMap[v] || !allowMap[v] {
			// 如果没有允许或者被拒绝，则返回false
			checkResult.Result = false
			d.logger.Debugf("Check end result:%v, resource:%+v, accessor:%+v, operation: %v", checkResult.Result, *resource, *accessor, v)
			return
		}
	}

	// 判断是否包含 操作义务信息
	includeMap := make(map[interfaces.PolicCalcyIncludeType]bool)
	for _, v := range include {
		includeMap[v] = true
	}
	if includeMap[interfaces.PolicCalcyIncludeOperationObligations] {
		checkResult.OperatrionOblist = d.calcObligationWithPriority(ctx, allowMapWithObligation)
	}
	d.logger.Debugf("Check end result:%v, resource:%+v, accessor:%+v, operation: %v", checkResult.Result, *resource, *accessor, operation)
	return checkResult, nil
}

func (p *policyCalc) calcObligationWithPriority(ctx context.Context, allowMapWithObligation map[string][]policyObligationCalcItem) (result map[string][]interfaces.PolicyObligationItem) {
	result = make(map[string][]interfaces.PolicyObligationItem)
	obligationMap := make(map[string]bool)
	for v, obligations := range allowMapWithObligation {
		// 按照优先级排序
		sort.Slice(obligations, func(i, j int) bool {
			return obligations[i].priority < obligations[j].priority
		})
		// 相同义务类型 只返回优先级最高的
		rTmp := map[string]interfaces.PolicyObligationItem{}
		for _, obligation := range obligations {
			if _, ok := rTmp[obligation.TypeID]; ok {
				continue
			}
			rTmp[obligation.TypeID] = interfaces.PolicyObligationItem{
				TypeID: obligation.TypeID,
				ID:     obligation.ID,
				Value:  obligation.Value,
			}
			if obligation.ID != "" {
				obligationMap[obligation.ID] = true
			}
		}
		arrayTmp := make([]interfaces.PolicyObligationItem, 0, len(rTmp))
		for _, v := range rTmp {
			arrayTmp = append(arrayTmp, v)
		}
		result[v] = arrayTmp
	}
	if len(obligationMap) == 0 {
		return
	}
	// 收集义务ID然后赋值，过滤不存在的义务
	// 获取义务信息
	obligations, err := p.obligation.GetByIDSInternal(ctx, obligationMap)
	if err != nil {
		p.logger.Errorf("calcObligationWithPriority GetByIDSInternal: %v", err)
		return
	}
	obligationInfMap := make(map[string]interfaces.ObligationInfo)
	for _, v := range obligations {
		obligationInfMap[v.ID] = v
	}

	tmp := make(map[string][]interfaces.PolicyObligationItem)
	for opreation, oblist := range result {
		for _, ob := range oblist {
			if ob.ID == "" {
				// 没有义务ID,表示设置的是value, 添加到结果中
				tmp[opreation] = append(tmp[opreation], ob)
				continue
			}
			// 义务ID不为空时, 如果义务存在,直接返回
			if info, ok := obligationInfMap[ob.ID]; ok {
				ob.Value = info.Value
				tmp[opreation] = append(tmp[opreation], ob)
			}
		}
	}
	p.logger.Debugf("calcObligationWithPriority end result: %+v", tmp)
	return tmp
}

// 获取指定资源类型上的资源列表
func (d *policyCalc) GetResourceList(ctx context.Context, resourceTypeID string, accessor *interfaces.AccessorInfo, operation []string,
	include []interfaces.PolicCalcyIncludeType) (resources []interfaces.ResourceInfo,
	resourceOperationObligationMap map[string]map[string][]interfaces.PolicyObligationItem, err error,
) {
	d.logger.Debugf("GetResourceList start, resourceTypeID: %s, accessor: %+v, operation: %v", resourceTypeID, *accessor, operation)
	// 查找当前用户的组织架构，角色
	accessTokens, err := d.getAccessorIDs(ctx, accessor)
	if err != nil {
		return
	}

	// 获取资源类型策略
	policies, err := d.db.GetPoliciesByResourceTypeAndAccessToken(ctx, resourceTypeID, accessTokens)
	if err != nil {
		return
	}

	policyMap := make(map[string][]interfaces.PolicyInfo)
	resourcesAncestorsMap := make(map[string][]interfaces.Ancestor)
	for i := range policies {
		policyMap[policies[i].ResourceID] = append(policyMap[policies[i].ResourceID], policies[i])
		resourcesAncestorsMap[policies[i].ResourceID] = policies[i].Ancestors
	}

	// 计算每个单个资源ID的权限
	// resourcePermMap[单个资源ID]操作权限
	resourcePermMap := make(map[string]resourcePerm)
	for resourceID, policies := range policyMap {
		if len(policies) == 0 {
			continue
		}
		resourcePermMap[resourceID] = d.calcOneResourcePerm(resourceID, policies)
	}

	allResources := make([]interfaces.ResourceInfo, 0, len(resourcePermMap))
	for resourceID := range resourcePermMap {
		allResources = append(allResources, interfaces.ResourceInfo{ID: resourceID})
	}
	resources = make([]interfaces.ResourceInfo, 0, len(allResources))
	resourceOperationObligationMap = make(map[string]map[string][]interfaces.PolicyObligationItem)
	for _, resource := range allResources {
		resourceSingle := interfaces.ResourceInfo{ID: resource.ID, Type: resourceTypeID, Ancestors: resourcesAncestorsMap[resource.ID]}
		resourcesTmp := d.createAncestorsResource(ctx, &resourceSingle)
		allowMap, denyMap, allowMapWithObligation := d.calcResourceInheritedOperation(resourcesTmp, resourcePermMap)
		checkResult := true
		// 所有的操作 都被允许 返回true，否则false
		for _, v := range operation {
			if denyMap[v] || !allowMap[v] {
				// 如果没有允许或者被拒绝，则false
				checkResult = false
				d.logger.Debugf("GetResourceList result:%v, resourceID:%v, operation: %v", checkResult, resource.ID, v)
				break
			}
		}
		if checkResult {
			// 判断是否包含 操作义务信息
			includeMap := make(map[interfaces.PolicCalcyIncludeType]bool)
			for _, v := range include {
				includeMap[v] = true
			}
			if includeMap[interfaces.PolicCalcyIncludeOperationObligations] {
				resourceOperationObligationMap[resource.ID] = d.calcObligationWithPriority(ctx, allowMapWithObligation)
			}
			d.logger.Debugf("GetResourceList result:%v, resourceID:%v", checkResult, resource.ID)
			resources = append(resources, resource)
		}
	}
	d.logger.Debugf("GetResourceList end, resources length: %v", len(resources))
	return resources, resourceOperationObligationMap, nil
}

// 过滤资源列表
func (d *policyCalc) ResourceFilter(ctx context.Context, resources []interfaces.ResourceInfo, accessor *interfaces.AccessorInfo,
	operation []string, include []interfaces.PolicCalcyIncludeType,
) (result []interfaces.ResourceInfo, resourceOperationMap map[string][]string, resourceOperationObligationMap map[string]map[string][]interfaces.PolicyObligationItem, err error) {
	if len(resources) == 0 {
		return
	}
	d.logger.Debugf("ResourceFilter start, resources length: %v, accessor: %+v, operation: %v", len(resources), *accessor, operation)
	accessTokens, err := d.getAccessorIDs(ctx, accessor)
	if err != nil {
		return nil, nil, nil, err
	}

	allResources := make([]interfaces.ResourceInfo, 0, len(resources))
	resourceTmpMap := make(map[string][]interfaces.ResourceInfo)
	for _, resource := range resources {
		resourceTmp := d.createAncestorsResource(ctx, &resource)
		resourceTmpMap[resource.ID] = resourceTmp
		allResources = append(allResources, resourceTmp...)
	}
	policies, err := d.db.GetPoliciesByResourcesAndAccessToken(ctx, allResources, accessTokens)
	if err != nil {
		return nil, nil, nil, err
	}

	policyMap := make(map[string][]interfaces.PolicyInfo)
	for i := range policies {
		policyMap[policies[i].ResourceID] = append(policyMap[policies[i].ResourceID], policies[i])
	}

	// 计算每个单个资源ID的权限
	// resourcePermMap[单个资源ID]操作权限
	resourcePermMap := make(map[string]resourcePerm)
	for resourceID, policies := range policyMap {
		if len(policies) == 0 {
			continue
		}
		resourcePermMap[resourceID] = d.calcOneResourcePerm(resourceID, policies)
	}

	resourceOperationMap = make(map[string][]string, len(resources))
	// 过滤有权限的列表
	resourceOperationObligationMap = make(map[string]map[string][]interfaces.PolicyObligationItem)
	for _, resource := range resources {
		resourceTmp := resourceTmpMap[resource.ID]
		allowMap, denyMap, allowMapWithObligation := d.calcResourceInheritedOperation(resourceTmp, resourcePermMap)
		hasOperation := true
		// 所有的操作 都被允许 返回true，否则返回false
		for _, v := range operation {
			if denyMap[v] || !allowMap[v] {
				// 如果没有允许或者被拒绝，则返回false
				hasOperation = false
				break
			}
		}

		if !hasOperation {
			continue
		}
		// 添加到结果列表
		result = append(result, resource)
		resourceOperationMap[resource.ID] = make([]string, 0, len(allowMap))
		// 判断是否包含 操作义务信息
		includeMap := make(map[interfaces.PolicCalcyIncludeType]bool)
		for _, v := range include {
			includeMap[v] = true
		}
		if includeMap[interfaces.PolicCalcyIncludeOperationObligations] {
			resourceOperationObligationMap[resource.ID] = d.calcObligationWithPriority(ctx, allowMapWithObligation)
		}

		for key := range allowMap {
			resourceOperationMap[resource.ID] = append(resourceOperationMap[resource.ID], key)
		}
	}
	d.logger.Debugf("ResourceFilter end, resources result length: %v, accessor: %+v, operation: %v", len(result), *accessor, operation)
	return
}

// 获取资源类型操作
func (d *policyCalc) GetResourceTypeOperation(ctx context.Context, resourceTypes []string, accessor *interfaces.AccessorInfo) (resourceTypeOperationMap map[string][]string, err error) {
	if len(resourceTypes) == 0 {
		return
	}
	d.logger.Debugf("GetResourceTypeOperation start, resourceTypes:%v, accessor:%+v", resourceTypes, *accessor)
	accessTokens, err := d.getAccessorIDs(ctx, accessor)
	if err != nil {
		return nil, err
	}

	policies, err := d.db.GetPoliciesByResourceTypes(ctx, resourceTypes, accessTokens)
	if err != nil {
		return nil, err
	}

	d.logger.Debugf("GetResourceTypeOperation, policies length: %v", len(policies))
	policyMap := make(map[string][]interfaces.PolicyInfo)
	for i := range policies {
		policyMap[policies[i].ResourceType] = append(policyMap[policies[i].ResourceType], policies[i])
	}

	resourcePermMap := make(map[string]resourcePerm)
	for resourceType, policies := range policyMap {
		if len(policies) == 0 {
			continue
		}
		resourcePermMap[resourceType] = d.calcOneResourcePerm(resourceType, policies)
	}

	resourceTypeMap, err := d.resourceType.GetByIDsInternal(ctx, resourceTypes)
	if err != nil {
		return
	}

	// 获取资源操作
	resourceTypeOperationMap = make(map[string][]string)
	for resourceType, resourceTypeInfo := range resourceTypeMap {
		typeOperationMap, _ := d.getTypeAndInstanceOperation(&resourceTypeInfo)
		d.logger.Debugf("resourceType: %s, typeOperationMap: %v\n", resourceType, typeOperationMap)
		allowMap := resourcePermMap[resourceType].allow
		resourceTypeOperationMap[resourceType] = make([]string, 0, len(allowMap))
		for key := range allowMap {
			if typeOperationMap[key] {
				resourceTypeOperationMap[resourceType] = append(resourceTypeOperationMap[resourceType], key)
			}
		}
	}

	return
}

// 资源操作
func (d *policyCalc) GetResourceOperation(ctx context.Context, resources []interfaces.ResourceInfo, accessor *interfaces.AccessorInfo) (resourceOperationMap map[string][]string,
	resourceOperationObligationMap map[string]map[string][]interfaces.PolicyObligationItem, err error,
) {
	if len(resources) == 0 {
		return
	}
	d.logger.Debugf("GetResourceOperation start, resources length: %v, accessor: %+v", len(resources), *accessor)
	accessTokens, err := d.getAccessorIDs(ctx, accessor)
	if err != nil {
		return nil, nil, err
	}

	allResources := make([]interfaces.ResourceInfo, 0, len(resources))
	resourceTmpMap := make(map[string][]interfaces.ResourceInfo)
	for _, resource := range resources {
		resourceTmp := d.createAncestorsResource(ctx, &resource)
		resourceTmpMap[resource.ID] = resourceTmp
		allResources = append(allResources, resourceTmp...)
	}
	policies, err := d.db.GetPoliciesByResourcesAndAccessToken(ctx, allResources, accessTokens)
	if err != nil {
		d.logger.Errorf("GetResourceOperation GetPoliciesByResourcesAndAccessToken resources:%v accessTokens:%v err:%v", resources, accessTokens, err)
		return nil, nil, err
	}

	policyMap := make(map[string][]interfaces.PolicyInfo)
	for i := range policies {
		policyMap[policies[i].ResourceID] = append(policyMap[policies[i].ResourceID], policies[i])
	}

	resourcePermMap := make(map[string]resourcePerm)
	for resourceID, policies := range policyMap {
		if len(policies) == 0 {
			continue
		}
		resourcePermMap[resourceID] = d.calcOneResourcePerm(resourceID, policies)
	}

	resourceTypeMap, err := d.resourceType.GetByIDsInternal(ctx, []string{resources[0].Type})
	if err != nil {
		return
	}
	resourceType := resourceTypeMap[resources[0].Type]
	typeOperationMap, instanceOperationMap := d.getTypeAndInstanceOperation(&resourceType)
	d.logger.Debugf("typeOperationMap: %v, instanceOperationMap: %v\n", typeOperationMap, instanceOperationMap)

	// 获取资源操作
	resourceOperationMap = make(map[string][]string)
	resourceOperationObligationMap = make(map[string]map[string][]interfaces.PolicyObligationItem)
	for _, resource := range resources {
		resourceTmp := resourceTmpMap[resource.ID]
		allowMap, _, allowMapWithObligation := d.calcResourceInheritedOperation(resourceTmp, resourcePermMap)
		resourceOperationMap[resource.ID] = make([]string, 0, len(allowMap))
		resourceOperationObligationMap[resource.ID] = d.calcObligationWithPriority(ctx, allowMapWithObligation)
		for key := range allowMap {
			if d.checkOperationScope(resource.ID, key, typeOperationMap, instanceOperationMap) {
				resourceOperationMap[resource.ID] = append(resourceOperationMap[resource.ID], key)
			}
		}
	}
	d.logger.Debugf("GetResourceOperation end\n")
	return
}

/*
获取本层和上层继承的权限, 就近原则，下层权限已配置，上层无效
入参 resources 资源ID列表，元素顺序为 根节点 -> 父节点 -> 本层
*/
func (d *policyCalc) calcResourceInheritedOperation(resources []interfaces.ResourceInfo, resourcePermMap map[string]resourcePerm) (
	allowMap map[string]bool, denyMap map[string]bool, allowMapWithObligation map[string][]policyObligationCalcItem,
) {
	// resourceID IDPath 用于打印日志
	resourceID := ""
	IDPath := ""
	if len(resources) > 0 { // 防止程序崩溃
		resourceID = resources[len(resources)-1].ID
	}
	for _, resource := range resources {
		tmp := strings.Split(resource.ID, "/")
		if len(tmp) > 0 { // 防止程序崩溃
			IDPath += "/" + tmp[0]
		}
	}

	// resourceIDs 用于控制决策策略的顺序， 本层 -> 父节点 -> 根节点
	resourceIDs := make([]string, len(resources))
	for i := len(resources) - 1; i >= 0; i-- {
		resourceIDs = append(resourceIDs, resources[i].ID)
	}
	// 如果资源ID不是*，则添加*，*表示所有实例
	if resourceID != "*" {
		resourceIDs = append(resourceIDs, "*")
	}

	d.logger.Debugf("calcResourceInheritedOperation start, resourceID: %s, IDPath: %s", resourceID, IDPath)
	allowMap = make(map[string]bool)
	allowMapWithObligation = make(map[string][]policyObligationCalcItem)
	denyMap = make(map[string]bool)
	// 权限配置 就近原则，下层权限已配置，上层无效
	for _, id := range resourceIDs {
		for v := range resourcePermMap[id].deny {
			if !denyMap[v] && !allowMap[v] {
				denyMap[v] = true
			}
		}

		for v, value := range resourcePermMap[id].allow {
			// 如果之前没有被拒绝，添加上层的义务
			if !denyMap[v] {
				allowMapWithObligation[v] = append(allowMapWithObligation[v], value...)
			}
			if !denyMap[v] && !allowMap[v] {
				allowMap[v] = true
				if len(value) == 0 {
					continue
				}
			}
		}
	}
	d.logger.Debugf("calcResourceInheritedOperation end, resourceID: %s, IDPath: %s, allowMap: %v, denyMap: %v", resourceID, IDPath, allowMap, denyMap)
	return
}

type policyObligationCalcItem struct {
	interfaces.PolicyObligationItem
	priority int // 优先级
}

/*
计算一个资源ID的权限, 只包含同一层级策略配置, 拒绝优先
resourceID 为了记录日志，方便调试
*/
func (d *policyCalc) calcOneResourcePerm(resourceID string, policies []interfaces.PolicyInfo) (result resourcePerm) {
	d.logger.Debugf("calcOneResourcePerm start, resourceID: %s", resourceID)
	allowMap := make(map[string][]policyObligationCalcItem)
	denyMap := make(map[string]bool)
	if len(policies) == 0 {
		return
	}

	for i := range policies {
		for _, v := range policies[i].Operation.Deny {
			denyMap[v.ID] = true
		}
	}

	// 拒绝优先，如果被拒绝，则不添加到allowMap
	for i := range policies {
		for _, v := range policies[i].Operation.Allow {
			if denyMap[v.ID] {
				continue
			}

			// 收集一个操作上配置的所有义务
			caclArray := make([]policyObligationCalcItem, 0, len(v.Obligations))
			for _, obl := range v.Obligations {
				tmp := policyObligationCalcItem{
					PolicyObligationItem: obl,
					priority:             d.obligationPriority[policies[i].AccessorType],
				}
				// 如果是所有用户 ，则优先级最低
				if policies[i].AccessorID == rootDepID {
					tmp.priority = rootDepObligationPriority
				}
				caclArray = append(caclArray, tmp)
			}
			allowMap[v.ID] = append(allowMap[v.ID], caclArray...)
		}
	}
	result.allow = allowMap
	result.deny = denyMap
	d.logger.Debugf("calcOneResourcePerm end, resourceID: %s, allowMap: %v, denyMap: %v", resourceID, allowMap, denyMap)
	return
}

/*
获取访问者的访问令牌, 访问者可以是用户或者应用账户
访问令牌包含以下3类:
1. 访问者自身ID
2. 访问者所属的组织架构，应用账户目前没有组织架构
3. 访问者和所属的组织架构关联的角色
*/
//nolint:staticcheck
func (d *policyCalc) getAccessorIDs(ctx context.Context, accessor *interfaces.AccessorInfo) (accessTokens []string, err error) {
	d.logger.Debugf("getAccessorIDs start, accessor.ID: %s, accessor.Type: %v", accessor.ID, accessor.Type)
	// 查找当前用户的组织架构
	if accessor.Type == interfaces.RealName {
		accessTokens, err = d.userMgnt.GetAccessorIDsByUserID(ctx, accessor.ID)
		if err != nil {
			d.logger.Errorf("getAccessorIDs userID:%s  err:%v", accessor.ID, err)
			return
		}
		// 添加根部门ID， 表示 所有用户/部门/组织
		accessTokens = append(accessTokens, rootDepID)
	} else if accessor.Type == interfaces.App {
		// 应用账户是自身ID
		accessTokens = []string{accessor.ID}
	}
	d.logger.Debugf("accessor.ID: %s, accessor.Type: %v, before GetRoleByMembers length: %v, accessTokens: %v", accessor.ID, accessor.Type, len(accessTokens), accessTokens)
	// 获取角色
	roles, err := d.role.GetRoleByMembers(ctx, accessTokens)
	if err != nil {
		d.logger.Errorf("getRoleByMembers  userID:%s err:%v", accessor.ID, err)
		return nil, err
	}

	for i := range roles {
		accessTokens = append(accessTokens, roles[i].ID)
	}

	if accessor.Type == interfaces.RealName {
		// 获取系统角色，系统角色ID都是固定ID
		// 现在系统角色还在账户管理服务，所以通过接口调用先获取用户的系统角色类型,然后将这些角色ID加入访问令牌
		var roleTypes []interfaces.SystemRoleType
		roleTypes, err = d.userMgnt.GetUserRolesByUserID(ctx, accessor.ID)
		if err != nil {
			d.logger.Errorf("getAccessorIDs GetUserRolesByUserID userID:%s  err:%v", accessor.ID, err)
			return
		}

		for _, roleType := range roleTypes {
			if roleType == interfaces.SuperAdmin {
				accessTokens = append(accessTokens, superAdminRoleID)
			} else if roleType == interfaces.SystemAdmin {
				accessTokens = append(accessTokens, systemAdminRoleID)
			} else if roleType == interfaces.SecurityAdmin {
				accessTokens = append(accessTokens, securityAdminRoleID)
			} else if roleType == interfaces.AuditAdmin {
				accessTokens = append(accessTokens, auditAdminRoleID)
			} else if roleType == interfaces.OrganizationAdmin {
				accessTokens = append(accessTokens, organizationAdminRoleID)
			} else if roleType == interfaces.OrganizationAudit {
				accessTokens = append(accessTokens, organizationAuditRoleID)
			}
		}
	}

	accessTokens = common.Distinct(accessTokens)
	d.logger.Debugf("getAccessorIDs end, accessor.ID: %s, accessTokens length: %v, accessTokens: %v", accessor.ID, len(accessTokens), accessTokens)
	return
}

func (d *policyCalc) checkOperationScope(resourceID, operationID string, typeOperationMap, instanceOperationMap map[string]bool) (result bool) {
	if resourceID == "*" {
		if typeOperationMap[operationID] {
			return true
		} else {
			return false
		}
	}

	if instanceOperationMap[operationID] {
		return true
	}
	return false
}

//nolint:staticcheck
func (d *policyCalc) getTypeAndInstanceOperation(resourceType *interfaces.ResourceType) (typeOperationMap, instanceOperationMap map[string]bool) {
	typeOperationMap = make(map[string]bool)
	instanceOperationMap = make(map[string]bool)
	for _, operation := range resourceType.Operation {
		for _, scope := range operation.Scope {
			if scope == interfaces.ScopeType {
				typeOperationMap[operation.ID] = true
			} else if scope == interfaces.ScopeInstance {
				instanceOperationMap[operation.ID] = true
			}
		}
	}
	return
}

/*
createAncestorsResource 创建祖先和本层资源信息数组，返回的数组顺序 根节点 -> 父节点 -> 本层
1. 有层级关系的资源类型，父节点本身的配置无法继承到下级
2. 策略配置为 { 资源实例ID:"父节点/*", 资源类型:"子节点资源类型"} 表示下层资源类型配置
*/
func (d *policyCalc) createAncestorsResource(_ context.Context, resource *interfaces.ResourceInfo) (resources []interfaces.ResourceInfo) {
	for _, ancestor := range resource.Ancestors {
		resources = append(resources, interfaces.ResourceInfo{
			ID:   ancestor.ID + "/*",
			Type: resource.Type,
		})
	}
	resources = append(resources, *resource)
	return resources
}
