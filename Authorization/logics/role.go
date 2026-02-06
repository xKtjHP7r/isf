// Package logics group AnyShare 角色业务逻辑层
package logics

import (
	"context"
	_ "embed"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"sync"
	"unicode/utf8"

	gerrors "github.com/kweaver-ai/go-lib/error"
	"github.com/kweaver-ai/proton-rds-sdk-go/sqlx"
	"github.com/satori/uuid"

	"Authorization/common"
	errors "Authorization/error"
	"Authorization/interfaces"
)

type role struct {
	roleDB                interfaces.DBRole
	roleMemberDB          interfaces.DBRoleMember
	userMgnt              interfaces.DrivenUserMgnt
	pool                  *sqlx.DB
	logger                common.Logger
	event                 interfaces.LogicsEvent
	resourceType          interfaces.LogicsResourceType
	resourceTypeHierarchy interfaces.LogicsResourceTypeHierarchy
	roleSortOrder         map[string]int // 角色排序
	i18n                  *common.I18n
}

var (
	//go:embed init_data/role_order.json
	roleOrderStr string
)

const (
	simplifiedChinese  = "zh-cn" // 简体中文
	traditionalChinese = "zh-tw" // 繁体中文
	americanEnglish    = "en-us" // 英文
)

const (
	_ int = iota
	i18nRoleNotFound
)

var (
	roleOnce   sync.Once
	roleLogics *role
)

// NewLogicsRole 创建新的role对象
//
//nolint:errcheck
func NewLogicsRole() *role {
	roleOnce.Do(func() {
		roleLogics = &role{
			roleDB:                dbRole,
			roleMemberDB:          dbRoleMember,
			userMgnt:              dnUserMgnt,
			pool:                  dbPool,
			logger:                common.NewLogger(),
			event:                 NewEvent(),
			resourceType:          NewResourceType(),
			resourceTypeHierarchy: NewResourceTypeHierarchy(),
			i18n: common.NewI18n(common.I18nMap{
				i18nRoleNotFound: {
					simplifiedChinese:  "角色不存在",
					traditionalChinese: "角色不存在",
					americanEnglish:    "The Role does not exist.",
				},
			}),
		}
		// 注册用户删除处理函数
		roleLogics.event.RegisterUserDeleted(roleLogics.deleteMemberByMemberID)
		// 注册用户组删除处理函数
		roleLogics.event.RegisterUserGroupDeleted(roleLogics.deleteMemberByMemberID)
		// 注册部门删除处理函数
		roleLogics.event.RegisterDepartmentDeleted(roleLogics.deleteMemberByMemberID)
		// 应用账户 删除处理函数
		roleLogics.event.RegisterAppDeleted(roleLogics.deleteMemberByMemberID)
		// 组织名称修改处理函数
		roleLogics.event.RegisterOrgNameModified(roleLogics.updateMemberName)
		// 应用账户改名
		roleLogics.event.RegisterAppNameModified(roleLogics.updateAppName)
		// 初始化角色数据
		roleLogics.initRoleOrder()
	})

	return roleLogics
}

func (r *role) checkVisitorType(ctx context.Context, visitor *interfaces.Visitor) (err error) {
	// 获取访问者角色
	var roleTypes []interfaces.SystemRoleType

	// 实名用户获取对应角色信息
	if visitor.Type == interfaces.RealName {
		// 获取用户角色信息
		roleTypes, err = r.userMgnt.GetUserRolesByUserID(ctx, visitor.ID)
		if err != nil {
			return err
		}
	}

	return checkVisitorType(
		visitor,
		roleTypes,
		[]interfaces.VisitorType{interfaces.RealName},
		[]interfaces.SystemRoleType{interfaces.SuperAdmin, interfaces.SystemAdmin, interfaces.SecurityAdmin},
	)
}

func (r *role) checkAndFilterResourceTypeScopeInfo(ctx context.Context, info *interfaces.ResourceTypeScopeInfo) (newInfo interfaces.ResourceTypeScopeInfo, err error) {
	if info.Unlimited {
		newInfo.Unlimited = true
		return newInfo, nil
	}

	checkDupMap := make(map[string]bool)
	resourceTypeIDs := make([]string, 0, len(info.Types))
	// 有重复或者空直接抛错
	for _, v := range info.Types {
		if v.ResourceTypeID == "" {
			err = gerrors.NewError(gerrors.PublicBadRequest, "resource_type_scope is empty")
			return newInfo, err
		}
		if _, ok := checkDupMap[v.ResourceTypeID]; ok {
			err = gerrors.NewError(gerrors.PublicBadRequest, "resource_type_scope is duplicate")
			return newInfo, err
		}
		checkDupMap[v.ResourceTypeID] = true
		resourceTypeIDs = append(resourceTypeIDs, v.ResourceTypeID)
	}

	// resourceTypeInfosMap 获取资源类型信息
	if len(resourceTypeIDs) == 0 {
		return newInfo, nil
	}

	resourceTypeInfosMap, err := r.resourceType.GetByIDsInternal(ctx, resourceTypeIDs)
	if err != nil {
		return newInfo, err
	}

	// 过滤不存在的资源类型
	for i := range info.Types {
		if _, ok := resourceTypeInfosMap[info.Types[i].ResourceTypeID]; ok {
			newInfo.Types = append(newInfo.Types, info.Types[i])
		}
	}

	return newInfo, nil
}

// AddRole 角色创建
func (r *role) AddRole(ctx context.Context, visitor *interfaces.Visitor, role *interfaces.RoleInfo) (id string, err error) {
	// 权限检查 visitor
	err = r.checkVisitorType(ctx, visitor)
	if err != nil {
		return
	}
	// 参数检查
	err = r.checkName(role.Name)
	if err != nil {
		return
	}

	// 根据名称获取指定的角色
	info, err := r.roleDB.GetRoleByName(ctx, role.Name)
	if err != nil {
		return id, err
	}
	// 如果角色名称存在，则返回错误
	if info.ID != "" {
		return id, gerrors.NewError(errors.RoleNameConflict, "role name already exists")
	}

	//  检查资源类型范围信息
	role.ResourceTypeScopeInfo, err = r.checkAndFilterResourceTypeScopeInfo(ctx, &role.ResourceTypeScopeInfo)
	if err != nil {
		return id, err
	}

	// 创建角色ID
	id = uuid.Must(uuid.NewV4(), err).String()
	if err != nil {
		return id, err
	}
	role.ID = id
	// 接口创建角色，都是用户自定义角色
	role.RoleSource = interfaces.RoleSourceUser
	// 创建角色
	err = r.roleDB.AddRoles(ctx, []interfaces.RoleInfo{*role})
	if err != nil {
		return id, err
	}

	return id, err
}

// DeleteRole 角色删除
func (r *role) DeleteRole(ctx context.Context, visitor *interfaces.Visitor, roleID string) (err error) {
	// 权限检查 visitor
	err = r.checkVisitorType(ctx, visitor)
	if err != nil {
		return
	}
	// 判断角色是否存在,如果存在 获取角色信息
	var info interfaces.RoleInfo
	info, err = r.roleDB.GetRoleByID(ctx, roleID)
	if err != nil {
		return err
	}

	// 如果角色不存在， 则不处理
	if info.ID != "" {
		// 删除角色
		err = r.roleDB.DeleteRole(ctx, info.ID)
		if err != nil {
			return err
		}
		// 删除角色成员
		err = r.roleMemberDB.DeleteByRoleID(ctx, roleID)
		if err != nil {
			return err
		}
		err = r.event.RoleDeleted(roleID)
		if err != nil {
			return err
		}
	}

	return err
}

// ModifyRole 角色修改
func (r *role) ModifyRole(ctx context.Context, visitor *interfaces.Visitor, roleID, name string,
	nameChanged bool, decryption string, decryptionhanged bool, resourceTypeScopes interfaces.ResourceTypeScopeInfo, resourceTypeScopesChanged bool,
) (err error) {
	// 权限检查 visitor
	err = r.checkVisitorType(ctx, visitor)
	if err != nil {
		return
	}
	// 判断角色是否存在
	var info interfaces.RoleInfo
	info, err = r.roleDB.GetRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if info.ID == "" {
		return gerrors.NewError(errors.RoleNotFound, r.i18n.Load(i18nRoleNotFound, visitor.Language))
	}

	if nameChanged {
		// 根据名称获取指定的角色
		var info interfaces.RoleInfo
		info, err = r.roleDB.GetRoleByName(ctx, name)
		if err != nil {
			return err
		}
		// 如果角色名称存在，且不是当前角色，则返回错误
		if info.ID != "" && info.ID != roleID {
			return gerrors.NewError(errors.RoleNameConflict, "role name already exists")
		}
	}

	if resourceTypeScopesChanged {
		resourceTypeScopes, err = r.checkAndFilterResourceTypeScopeInfo(ctx, &resourceTypeScopes)
		if err != nil {
			return err
		}
	}

	// 修改角色信息
	err = r.roleDB.ModifyRole(ctx, roleID, name, nameChanged, decryption, decryptionhanged, resourceTypeScopes, resourceTypeScopesChanged)
	if err != nil {
		return err
	}

	if nameChanged {
		err = r.event.RoleNameModified(roleID, name)
		if err != nil {
			return err
		}
	}
	return err
}

/*
填充资源类型范围信息
1. 如果资源类型范围信息为不限，则返回所有资源类型
2. 如果资源类型范围信息为限制，则返回限制的资源类型
3. 返回的信息有资源类型名称，资源类型描述，资源类型实例URL，资源类型数据结构，资源类型上的操作，资源类型实例上的操作
*/
//nolint:staticcheck
func (r *role) getResourceTypeScopeInfo(ctx context.Context, visitor *interfaces.Visitor, info interfaces.ResourceTypeScopeInfo,
	param interfaces.RoleInfoParam) (respInfo interfaces.ResourceTypeScopeInfoWithOperation, err error) {
	respInfo.Unlimited = info.Unlimited
	var resourceTypeInfoMap map[string]interfaces.ResourceType
	var resourceTypeIDs []string
	// 如果不限制，则返回所有资源类型
	if info.Unlimited {
		var resourceTypeInfos []interfaces.ResourceType
		resourceTypeInfos, err = r.resourceType.GetAllInternal(ctx)
		if err != nil {
			return respInfo, err
		}
		resourceTypeIDs = make([]string, 0, len(resourceTypeInfos))
		resourceTypeInfoMap = make(map[string]interfaces.ResourceType, len(resourceTypeInfos))
		for _, resourceTypeInfo := range resourceTypeInfos {
			resourceTypeIDs = append(resourceTypeIDs, resourceTypeInfo.ID)
			resourceTypeInfoMap[resourceTypeInfo.ID] = resourceTypeInfo
		}
	} else {
		// 获取角色上面的所有资源类型ID
		resourceTypeIDs = make([]string, 0, len(info.Types))
		for _, resourceTypeScope := range info.Types {
			resourceTypeIDs = append(resourceTypeIDs, resourceTypeScope.ResourceTypeID)
		}
		// 获取资源类型信息
		resourceTypeInfoMap, err = r.resourceType.GetByIDsInternal(ctx, resourceTypeIDs)
		if err != nil {
			return respInfo, err
		}
	}

	if param.ResourceTypeViewMode == interfaces.ResourceTypeViewModeFlat {
		respInfo.Types = r.getResourceTypeScopeInfoFlat(ctx, visitor, resourceTypeIDs, resourceTypeInfoMap)
	} else if param.ResourceTypeViewMode == interfaces.ResourceTypeViewModeHierarchy {
		respInfo.Types, err = r.getResourceTypeScopeInfoHierarchy(ctx, visitor, resourceTypeIDs, resourceTypeInfoMap)
		if err != nil {
			return respInfo, err
		}
	}
	return
}

// getResourceTypeScopeInfoFlat 获取资源类型范围信息扁平化
func (r *role) getResourceTypeScopeInfoFlat(ctx context.Context, visitor *interfaces.Visitor, resourceTypeIDs []string,
	resourceTypeInfoMap map[string]interfaces.ResourceType) (types []interfaces.ResourceTypeScopeWithOperation) {
	// 填充资源类型和资源实例上的操作
	for _, resourceTypeID := range resourceTypeIDs {
		tmpResourceTypeScope, ok := r.getResourceTypeInfo(ctx, visitor, resourceTypeID, resourceTypeInfoMap)
		// 如果资源类型不存在，则跳过
		if !ok {
			continue
		}
		types = append(types, tmpResourceTypeScope)
	}
	return
}

// getResourceTypeScopeInfoHierarchy 获取资源类型范围信息层级化
func (r *role) getResourceTypeScopeInfoHierarchy(ctx context.Context, visitor *interfaces.Visitor, resourceTypeIDs []string,
	resourceTypeInfoMap map[string]interfaces.ResourceType) (types []interfaces.ResourceTypeScopeWithOperation, err error) {
	// 有层级关系的，按照层级展示
	resourceTypeHierarchyMap, err := r.resourceTypeHierarchy.GetAll(ctx)
	if err != nil {
		return types, err
	}
	// 下层层级的资源类型都放到这个childrenMap中，递归检查Children
	childrenMap := make(map[string]bool)
	topMap := make(map[string]bool)
	var collectChildrenIDs func(children []interfaces.ResourceTypeHierarchy)
	collectChildrenIDs = func(children []interfaces.ResourceTypeHierarchy) {
		for _, child := range children {
			childrenMap[child.ResourceTypeID] = true
			collectChildrenIDs(child.Children)
		}
	}
	for _, resourceTypeHierarchy := range resourceTypeHierarchyMap {
		topMap[resourceTypeHierarchy.ResourceTypeID] = true
		collectChildrenIDs(resourceTypeHierarchy.Children)
	}

	for _, resourceTypeID := range resourceTypeIDs {
		// 如果不在顶层，在子层级中，则跳过
		if !topMap[resourceTypeID] && childrenMap[resourceTypeID] {
			continue
		}
		tmpResourceTypeScope, ok := r.getResourceTypeInfo(ctx, visitor, resourceTypeID, resourceTypeInfoMap)

		// 如果资源类型不存在，则跳过
		if !ok {
			continue
		}
		// 如果有层级关系, 递归填充子资源类型（hierarchy.Children 下可能还有嵌套的 Children）
		if hierarchy, ok := resourceTypeHierarchyMap[resourceTypeID]; ok && len(hierarchy.Children) > 0 {
			resourceTypeIDSet := make(map[string]bool, len(resourceTypeIDs))
			for _, id := range resourceTypeIDs {
				resourceTypeIDSet[id] = true
			}
			var fillChildrenFromHierarchy func(children []interfaces.ResourceTypeHierarchy) ([]interfaces.ResourceTypeScopeWithOperation, error)
			fillChildrenFromHierarchy = func(children []interfaces.ResourceTypeHierarchy) ([]interfaces.ResourceTypeScopeWithOperation, error) {
				result := make([]interfaces.ResourceTypeScopeWithOperation, 0, len(children))
				for _, child := range children {
					if !resourceTypeIDSet[child.ResourceTypeID] {
						continue
					}
					scope, ok := r.getResourceTypeInfo(ctx, visitor, child.ResourceTypeID, resourceTypeInfoMap)
					if !ok {
						continue
					}
					if len(child.Children) > 0 {
						scope.Children, err = fillChildrenFromHierarchy(child.Children)
						if err != nil {
							return nil, err
						}
					}
					result = append(result, scope)
				}
				return result, nil
			}
			tmpResourceTypeScope.Children, err = fillChildrenFromHierarchy(hierarchy.Children)
			if err != nil {
				return types, err
			}
		}
		types = append(types, tmpResourceTypeScope)
	}
	return
}

func (r *role) getResourceTypeInfo(_ context.Context, visitor *interfaces.Visitor, resourceTypeID string,
	resourceTypeInfoMap map[string]interfaces.ResourceType) (tmpResourceTypeInfo interfaces.ResourceTypeScopeWithOperation, ok bool) {
	resourceTypeInfo, ok := resourceTypeInfoMap[resourceTypeID]
	if !ok {
		return tmpResourceTypeInfo, false
	}
	tmpResourceTypeInfo = interfaces.ResourceTypeScopeWithOperation{
		ID:          resourceTypeInfo.ID,
		Name:        resourceTypeInfo.Name,
		Description: resourceTypeInfo.Description,
		InstanceURL: resourceTypeInfo.InstanceURL,
		DataStruct:  resourceTypeInfo.DataStruct,
	}
	typeOperations := make([]interfaces.ResourceTypeOperationResponse, 0, len(resourceTypeInfo.Operation))
	instanceOperations := make([]interfaces.ResourceTypeOperationResponse, 0, len(resourceTypeInfo.Operation))
	for _, operation := range resourceTypeInfo.Operation {
		opeInfo := interfaces.ResourceTypeOperationResponse{
			ID:          operation.ID,
			Description: operation.Description,
		}
		// 根据国际化获取名称
		opeInfo.Name = r.getOperationNameByLanguage(visitor.Language, operation.Name)
		// 填充类型上的操作
		if slices.Contains(operation.Scope, interfaces.ScopeType) {
			typeOperations = append(typeOperations, opeInfo)
		}
		// 填充实例上的操作
		if slices.Contains(operation.Scope, interfaces.ScopeInstance) {
			instanceOperations = append(instanceOperations, opeInfo)
		}
	}
	tmpResourceTypeInfo.TypeOperation = typeOperations
	tmpResourceTypeInfo.InstanceOperation = instanceOperations
	return tmpResourceTypeInfo, true
}

// GetRoleByID 获取指定的角色
func (r *role) GetRoleByID(ctx context.Context, visitor *interfaces.Visitor, roleID string, param interfaces.RoleInfoParam) (info interfaces.RoleInfoWithResourceTypeOperation, err error) {
	// 权限检查 visitor
	err = r.checkVisitorType(ctx, visitor)
	if err != nil {
		return
	}
	// 获取符合条件的角色
	tmpInfo, err := r.roleDB.GetRoleByID(ctx, roleID)
	if err != nil {
		return info, err
	}
	if tmpInfo.ID == "" {
		return info, gerrors.NewError(errors.RoleNotFound, r.i18n.Load(i18nRoleNotFound, visitor.Language))
	}

	info.ID = tmpInfo.ID
	info.Name = tmpInfo.Name
	info.Description = tmpInfo.Description
	info.RoleSource = tmpInfo.RoleSource
	// 填充资源类型范围信息
	info.ResourceTypeScopesInfo, err = r.getResourceTypeScopeInfo(ctx, visitor, tmpInfo.ResourceTypeScopeInfo, param)
	if err != nil {
		return info, err
	}

	return info, err
}

func (r *role) getOperationNameByLanguage(language string, operationName []interfaces.OperationName) (name string) {
	for _, name := range operationName {
		if strings.EqualFold(name.Language, language) {
			return name.Value
		}
	}
	return
}

// GetRoles 列举符合条件的角色
//
//nolint:staticcheck
func (r *role) GetRoles(ctx context.Context, visitor *interfaces.Visitor, info interfaces.RoleSearchInfo) (num int, outInfo []interfaces.RoleInfo, err error) {
	// 权限检查 visitor
	err = r.checkVisitorType(ctx, visitor)
	if err != nil {
		return
	}
	// 去掉keyword 的前后空格
	info.Keyword = strings.TrimSpace(info.Keyword)
	// 获取所有符合条件的角色数量
	num, err = r.roleDB.GetRolesSum(ctx, info)
	if err != nil {
		return num, outInfo, err
	}

	var infos []interfaces.RoleInfo
	if num > 0 {
		// 获取符合条件并且分页的角色
		infos, err = r.roleDB.GetRoles(ctx, info)
		if err != nil {
			return num, outInfo, err
		}

		resourceTypeMap := make(map[string]bool)
		resourceTypeIDs := make([]string, 0, len(infos))
		for _, info := range infos {
			// 不限制 直接跳过
			if info.ResourceTypeScopeInfo.Unlimited {
				continue
			}
			for _, resourceType := range info.ResourceTypeScopeInfo.Types {
				if _, ok := resourceTypeMap[resourceType.ResourceTypeID]; !ok {
					resourceTypeMap[resourceType.ResourceTypeID] = true
					resourceTypeIDs = append(resourceTypeIDs, resourceType.ResourceTypeID)
				}
			}
		}
		var resourceTypeInfosMap map[string]interfaces.ResourceType
		// 填充资源类型范围信息
		if len(resourceTypeIDs) > 0 {
			resourceTypeInfosMap, err = r.resourceType.GetByIDsInternal(ctx, resourceTypeIDs)
			if err != nil {
				return num, outInfo, err
			}
		}
		for i := range infos {
			if infos[i].ResourceTypeScopeInfo.Unlimited {
				continue
			}
			for j := range infos[i].ResourceTypeScopeInfo.Types {
				if _, ok := resourceTypeInfosMap[infos[i].ResourceTypeScopeInfo.Types[j].ResourceTypeID]; ok {
					id := infos[i].ResourceTypeScopeInfo.Types[j].ResourceTypeID
					infos[i].ResourceTypeScopeInfo.Types[j].ResourceTypeName = resourceTypeInfosMap[id].Name
				}
			}
		}
	}

	outInfo = append(outInfo, infos...)
	return num, outInfo, err
}

// AddOrDeleteRoleMemebers 批量删除或者添加角色成员
func (r *role) AddOrDeleteRoleMemebers(ctx context.Context, visitor *interfaces.Visitor, method, roleID string, infos map[string]interfaces.RoleMemberInfo) (err error) {
	switch method {
	case "POST":
		err = r.AddRoleMembers(ctx, visitor, roleID, infos)
	case "DELETE":
		err = r.DeleteRoleMembers(ctx, visitor, roleID, infos)
	default:
		return gerrors.NewError(gerrors.PublicBadRequest, "unsupported method: "+method)
	}
	return err
}

// AddRoleMembers 角色成员添加
func (r *role) AddRoleMembers(ctx context.Context, visitor *interfaces.Visitor, roleID string, infos map[string]interfaces.RoleMemberInfo) (err error) {
	// 权限检查 visitor
	err = r.checkVisitorType(ctx, visitor)
	if err != nil {
		return
	}
	// 判断角色是否存在
	var info interfaces.RoleInfo
	info, err = r.roleDB.GetRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if info.ID == "" {
		err = gerrors.NewError(errors.RoleNotFound, r.i18n.Load(i18nRoleNotFound, visitor.Language))
		return err
	}

	// 获取角色已存在成员
	oldMembers, err := r.roleMemberDB.GetRoleMembersByRoleID(ctx, roleID)
	if err != nil {
		r.logger.Errorf("AddRoleMembers GetRoleMembers: %v", err)
		return
	}
	oldMembersMap := make(map[string]bool, len(oldMembers))
	for _, member := range oldMembers {
		oldMembersMap[member.ID] = true
	}

	// 过滤已存在的成员
	newMembers := make([]interfaces.RoleMemberInfo, 0, len(infos))
	// 用于批量检查访问者ID与Type是否对应
	idTypeMap := make(map[string]interfaces.AccessorType)
	for _, temp := range infos {
		if _, ok := oldMembersMap[temp.ID]; !ok {
			newMembers = append(newMembers, temp)
			idTypeMap[temp.ID] = temp.MemberType
		}
	}

	if len(newMembers) == 0 {
		return
	}

	// 获取用户信息
	idNameMap, err := r.userMgnt.GetNameByAccessorIDs(ctx, idTypeMap)
	if err != nil {
		r.logger.Errorf("AddRoleMembers GetNameByAccessorIDs: %v", err)
		return
	}
	// 添加角色成员
	for i, temp := range newMembers {
		newMembers[i].Name = idNameMap[temp.ID]
	}

	// 添加角色
	err = r.roleMemberDB.AddRoleMembers(ctx, roleID, newMembers)
	if err != nil {
		return err
	}
	return err
}

// DeleteRoleMembers 角色成员删除
func (r *role) DeleteRoleMembers(ctx context.Context, visitor *interfaces.Visitor, roleID string, infos map[string]interfaces.RoleMemberInfo) (err error) {
	// 权限检查 visitor
	err = r.checkVisitorType(ctx, visitor)
	if err != nil {
		return
	}
	// 判断角色是否存在
	var info interfaces.RoleInfo
	info, err = r.roleDB.GetRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if info.ID == "" {
		err = gerrors.NewError(errors.RoleNotFound, r.i18n.Load(i18nRoleNotFound, visitor.Language))
		return err
	}

	// 删除用户
	membersIDs := make([]string, 0, len(infos))
	for k := range infos {
		membersIDs = append(membersIDs, infos[k].ID)
	}

	err = r.roleMemberDB.DeleteRoleMembers(ctx, roleID, membersIDs)
	if err != nil {
		r.logger.Errorf("DeleteRoleMembers DeleteRoleMembers err:%v", err)
		return err
	}
	return err
}

// GetRoleMembers 角色成员列举
//
//nolint:staticcheck
func (r *role) GetRoleMembers(ctx context.Context, visitor *interfaces.Visitor, roleID string, info interfaces.RoleMemberSearchInfo) (num int, outInfo []interfaces.RoleMemberInfo, err error) {
	// 权限检查 visitor
	err = r.checkVisitorType(ctx, visitor)
	if err != nil {
		return
	}
	// 判断角色是否存在
	var roleInfo interfaces.RoleInfo
	roleInfo, err = r.roleDB.GetRoleByID(ctx, roleID)
	if err != nil {
		return num, outInfo, err
	}
	if roleInfo.ID == "" {
		err = gerrors.NewError(errors.RoleNotFound, r.i18n.Load(i18nRoleNotFound, visitor.Language))
		return num, outInfo, err
	}

	// 去除keyword 的前后空格
	info.Keyword = strings.TrimSpace(info.Keyword)
	// 获取符合条件的角色成员数量
	num, err = r.roleMemberDB.GetRoleMembersNum(ctx, roleID, info)
	if err != nil {
		return num, outInfo, err
	}

	// 获取符合条件的分页角色成员信息
	var outInfos []interfaces.RoleMemberInfo
	if num > 0 {
		outInfos, err = r.roleMemberDB.GetPaginationByRoleID(ctx, roleID, info)
		if err != nil {
			return num, outInfo, err
		}

		// 获取用户的部门信息
		userIDs := []string{}
		// 获取部门的父部门信息
		for i, outInfo := range outInfos {
			if outInfo.MemberType == interfaces.AccessorUser {
				userIDs = append(userIDs, outInfo.ID)
			} else if outInfo.MemberType == interfaces.AccessorDepartment {
				var dep []interfaces.Department
				dep, err = r.userMgnt.GetParentDepartmentsByDepartmentID(ctx, outInfo.ID)
				if err != nil {
					return num, outInfos, err
				}
				outInfos[i].ParentDeps = append(outInfos[i].ParentDeps, dep)
			} else {
				outInfos[i].ParentDeps = [][]interfaces.Department{}
			}
		}

		// 批量获取用户的父部门信息
		var userInfoMaps map[string]interfaces.UserInfo
		userInfoMaps, err = r.userMgnt.BatchGetUserInfoByID(ctx, userIDs)
		if err != nil {
			return num, outInfos, err
		}
		for i, outInfo := range outInfos {
			if outInfo.MemberType == interfaces.AccessorUser {
				outInfos[i].ParentDeps = userInfoMaps[outInfo.ID].ParentDeps
			}
		}
	}

	return num, outInfos, err
}

// 检查有效性
func (r *role) checkName(name string) (err error) {
	// TODO: 中文的 ？ 也被允许了
	illegalChars := "|\\/:*?\"<>"
	length := utf8.RuneCountInString(name)

	if strings.ContainsAny(name, illegalChars) || length < 1 || length > 128 {
		return gerrors.NewError(gerrors.PublicBadRequest, "role name is illegal")
	}

	return nil
}

// GetRoleByMembers 通过成员获取角色
func (r *role) GetRoleByMembers(ctx context.Context, memberIDs []string) (outInfo []interfaces.RoleInfo, err error) {
	roles, err := r.roleMemberDB.GetRoleByMembers(ctx, memberIDs)
	if err != nil {
		return nil, err
	}
	return roles, nil
}

// GetRolesByIDs 批量获取角色
func (r *role) GetRolesByIDs(ctx context.Context, roleIDs []string) (infoMap map[string]interfaces.RoleInfo, err error) {
	infoMap, err = r.roleDB.GetRoleByIDs(ctx, roleIDs)
	if err != nil {
		return nil, err
	}
	return infoMap, nil
}

// checkResourceTypeChange 检查资源类型是否发生变化， 有变化返回true
func (r *role) checkRoleChange(old, newInfo *interfaces.RoleInfo) (changed bool) {
	if old.Name != newInfo.Name {
		return true
	}
	if old.Description != newInfo.Description {
		return true
	}
	if !reflect.DeepEqual(old.ResourceTypeScopeInfo, newInfo.ResourceTypeScopeInfo) {
		return true
	}
	return false
}

// InitRoles 初始化角色
func (r *role) InitRoles(ctx context.Context, roles []interfaces.RoleInfo) (err error) {
	if len(roles) == 0 {
		return nil
	}
	tmpRoles := make([]interfaces.RoleInfo, 0, len(roles))
	for _, role := range roles {
		var oldInfo interfaces.RoleInfo
		oldInfo, err = r.roleDB.GetRoleByName(ctx, role.Name)
		if err != nil {
			return err
		}
		// 如果角色名称存在
		if oldInfo.ID != "" {
			// 如果有变化，则更新
			if r.checkRoleChange(&oldInfo, &role) {
				err = r.roleDB.SetRoleByID(ctx, oldInfo.ID, &role)
				if err != nil {
					return err
				}
			}
			continue
		}
		tmpRoles = append(tmpRoles, role)
	}

	// 创建角色
	err = r.roleDB.AddRoles(ctx, tmpRoles)
	if err != nil {
		return err
	}
	return err
}

// InitRoleMemebers 批量初始化角色成员
func (r *role) InitRoleMemebers(ctx context.Context, infoMap map[string][]interfaces.RoleMemberInfo) (err error) {
	for roleID, infos := range infoMap {
		var oldMembers []interfaces.RoleMemberInfo
		// 获取角色已存在成员
		oldMembers, err = r.roleMemberDB.GetRoleMembersByRoleID(ctx, roleID)
		if err != nil {
			r.logger.Errorf("InitRoleMemebers GetRoleMembers: %v", err)
			return
		}
		oldMembersMap := make(map[string]bool, len(oldMembers))
		for _, member := range oldMembers {
			oldMembersMap[member.ID] = true
		}

		newMembers := make([]interfaces.RoleMemberInfo, 0, len(infos))
		// 过滤已存在的角色成员
		for k := range infos {
			if _, ok := oldMembersMap[infos[k].ID]; !ok {
				newMembers = append(newMembers, infos[k])
			}
		}

		// 添加角色成员
		err = r.roleMemberDB.AddRoleMembers(ctx, roleID, newMembers)
		if err != nil {
			r.logger.Errorf("InitRoleMemebers AddRoleMembers err:%v", err)
			return err
		}
	}
	return
}

func (r *role) deleteMemberByMemberID(memberID string) (err error) {
	return r.roleMemberDB.DeleteByMemberIDs([]string{memberID})
}

func (r *role) updateMemberName(memberID, name string) (err error) {
	return r.roleMemberDB.UpdateMemberName(memberID, name)
}

func (r *role) updateAppName(info *interfaces.AppInfo) (err error) {
	return r.roleMemberDB.UpdateMemberName(info.ID, info.Name)
}

/*
GetResourceTypeRoles 资源类型角色列举
内存分页
*/

//nolint:staticcheck
func (r *role) GetResourceTypeRoles(ctx context.Context, visitor *interfaces.Visitor, info interfaces.ResourceTypeRoleSearchInfo) (count int, outInfo []interfaces.RoleInfo, err error) {
	// 检查资源类型是否合法
	resourceTypeMap, err := r.resourceType.GetByIDsInternal(ctx, []string{info.ResourceTypeID})
	if err != nil {
		return 0, nil, err
	}
	if _, ok := resourceTypeMap[info.ResourceTypeID]; !ok {
		return 0, []interfaces.RoleInfo{}, gerrors.NewError(gerrors.PublicBadRequest, "resource_type_id is not exist")
	}

	// 去掉keyword 的前后空格
	info.Keyword = strings.TrimSpace(info.Keyword)
	// 获取所有角色，DB层按名称过滤
	roles, err := r.roleDB.GetAllUserRolesInternal(ctx, info.Keyword)
	if err != nil {
		return 0, nil, err
	}

	resultRoles := make([]interfaces.RoleInfo, 0, len(roles))
	for _, role := range roles {
		// 资源类型不限，则直接添加
		if role.ResourceTypeScopeInfo.Unlimited {
			resultRoles = append(resultRoles, role)
			continue
		} else {
			for _, v := range role.ResourceTypeScopeInfo.Types {
				if v.ResourceTypeID == info.ResourceTypeID {
					resultRoles = append(resultRoles, role)
					break
				}
			}
		}
	}

	if len(resultRoles) == 0 {
		return 0, []interfaces.RoleInfo{}, nil
	}

	// resultRoles 按照修改时间 降序排序，然后再去分页，
	slices.SortFunc(resultRoles, func(a, b interfaces.RoleInfo) int {
		if a.ModifyTime > b.ModifyTime {
			return -1
		}
		if a.ModifyTime < b.ModifyTime {
			return 1
		}
		return 0
	})

	// 计算总数
	count = len(resultRoles)

	// 分页处理
	start := info.Offset
	end := start + info.Limit
	if start >= count {
		// 如果起始位置超出总数，返回空结果
		return count, []interfaces.RoleInfo{}, nil
	}
	if end > count {
		end = count
	}

	outInfo = resultRoles[start:end]
	return count, outInfo, nil
}

// initRoleOrder 初始化角色排序
func (r *role) initRoleOrder() (err error) {
	var rolesJson []any
	err = json.Unmarshal([]byte(roleOrderStr), &rolesJson)
	if err != nil {
		r.logger.Errorf("unmarshal roleDataStr failed, err: %v", err)
		return
	}
	r.roleSortOrder = make(map[string]int)
	for i, roleJson := range rolesJson {
		roleDr := roleJson.(map[string]any)
		var roleID string
		roleIDJson, ok := roleDr["id"]
		if ok {
			roleID = roleIDJson.(string)
		}
		r.roleSortOrder[roleID] = i + 1
	}
	return nil
}

// GetAccessorRoles根据访问者列举角色
//
//nolint:gocyclo
func (r *role) GetAccessorRoles(ctx context.Context, param interfaces.AccessorRoleSearchInfo) (count int, outInfo []interfaces.RoleInfo, err error) {
	accessTokens, err := r.userMgnt.GetAccessorIDsByUserID(ctx, param.AccessorID)
	if err != nil {
		r.logger.Errorf("GetAccessorRoles GetAccessorIDsByUserID userID:%s  err:%v", param.AccessorID, err)
		return
	}
	// 添加根部门ID， 表示 所有用户/部门/组织
	accessTokens = append(accessTokens, rootDepID)
	roles, err := r.roleMemberDB.GetRoleByMembers(ctx, accessTokens)
	if err != nil {
		r.logger.Errorf("GetAccessorRoles GetRoleByMembers accessTokens:%v  err:%v", accessTokens, err)
		return 0, nil, err
	}
	r.logger.Debugf("GetAccessorRoles GetRoleByMembers length of roles:%v", len(roles))
	var pagedAccessTokens []string
	for _, role := range roles {
		pagedAccessTokens = append(pagedAccessTokens, role.ID)
	}
	var hasRoleSourceSystem bool
	if slices.Contains(param.RoleSources, interfaces.RoleSourceSystem) {
		hasRoleSourceSystem = true
	}
	// 如果有 系统角色的参数， 则获取用户系统角色
	if hasRoleSourceSystem {
		var roleTypes []interfaces.SystemRoleType
		roleTypes, err = r.userMgnt.GetUserRolesByUserID(ctx, param.AccessorID)
		if err != nil {
			r.logger.Errorf("GetAccessorRoles GetUserRolesByUserID userID:%s  err:%v", param.AccessorID, err)
			return
		}

		for _, roleType := range roleTypes {
			switch roleType {
			case interfaces.SuperAdmin:
				pagedAccessTokens = append(pagedAccessTokens, superAdminRoleID)
			case interfaces.SystemAdmin:
				pagedAccessTokens = append(pagedAccessTokens, systemAdminRoleID)
			case interfaces.SecurityAdmin:
				pagedAccessTokens = append(pagedAccessTokens, securityAdminRoleID)
			case interfaces.AuditAdmin:
				pagedAccessTokens = append(pagedAccessTokens, auditAdminRoleID)
			case interfaces.OrganizationAdmin:
				pagedAccessTokens = append(pagedAccessTokens, organizationAdminRoleID)
			case interfaces.OrganizationAudit:
				pagedAccessTokens = append(pagedAccessTokens, organizationAuditRoleID)
			case interfaces.NormalUser:
			}
		}
	}
	count = len(pagedAccessTokens)
	r.logger.Debugf("GetAccessorRoles length of pagedAccessTokens:%v", count)
	if count == 0 {
		return 0, []interfaces.RoleInfo{}, nil
	}

	// 根据 id 获取角色信息
	roleInfosMap, err := r.roleDB.GetRoleByIDs(ctx, pagedAccessTokens)
	if err != nil {
		r.logger.Errorf("GetAccessorRoles GetRoleByIDs pagedAccessTokens:%v  err:%v", pagedAccessTokens, err)
		return 0, nil, err
	}
	roles = make([]interfaces.RoleInfo, 0, len(pagedAccessTokens))
	for _, roleID := range pagedAccessTokens {
		roles = append(roles, roleInfosMap[roleID])
	}

	// 按照角色来源 过滤
	if len(param.RoleSources) == 0 {
		param.RoleSources = []interfaces.RoleSource{interfaces.RoleSourceBusiness, interfaces.RoleSourceUser}
	}

	var hasRoleSourceBusiness bool
	if slices.Contains(param.RoleSources, interfaces.RoleSourceBusiness) {
		hasRoleSourceBusiness = true
	}

	var hasRoleUser bool
	if slices.Contains(param.RoleSources, interfaces.RoleSourceUser) {
		hasRoleUser = true
	}

	tmpRoles := make([]interfaces.RoleInfo, 0, len(roles))
	// 去重
	dupMap := make(map[string]bool)
	for _, role := range roles {
		if _, ok := dupMap[role.ID]; ok {
			continue
		}
		if role.RoleSource == interfaces.RoleSourceSystem && hasRoleSourceSystem {
			tmpRoles = append(tmpRoles, role)
			dupMap[role.ID] = true
		}
		if role.RoleSource == interfaces.RoleSourceBusiness && hasRoleSourceBusiness {
			tmpRoles = append(tmpRoles, role)
			dupMap[role.ID] = true
		}
		if role.RoleSource == interfaces.RoleSourceUser && hasRoleUser {
			tmpRoles = append(tmpRoles, role)
			dupMap[role.ID] = true
		}
	}
	r.logger.Debugf("GetAccessorRoles length of tmpRoles:%v", len(tmpRoles))
	// 按照 r.roleSortOrder 的顺序排序 tmpRoles
	slices.SortFunc(tmpRoles, func(a, b interfaces.RoleInfo) int {
		orderA, existsA := r.roleSortOrder[a.ID]
		orderB, existsB := r.roleSortOrder[b.ID]

		// 如果都在 roleSortOrder 中，按照排序值排序
		if existsA && existsB {
			if orderA < orderB {
				return -1
			}
			if orderA > orderB {
				return 1
			}
			return 0
		}
		// 如果只有 a 在 roleSortOrder 中，a 排在前面
		if existsA {
			return -1
		}
		// 如果只有 b 在 roleSortOrder 中，b 排在前面
		if existsB {
			return 1
		}
		// 如果都不在 roleSortOrder 中，按修改时间降序排序（最新的在前）
		if a.ModifyTime > b.ModifyTime {
			return -1
		}
		if a.ModifyTime < b.ModifyTime {
			return 1
		}
		return 0
	})

	count = len(tmpRoles)
	if count == 0 {
		return 0, []interfaces.RoleInfo{}, nil
	}

	// 应用分页（limit == -1 时返回所有）
	if param.Limit == -1 {
		outInfo = tmpRoles
	} else {
		start := param.Offset
		end := param.Offset + param.Limit
		if start > len(tmpRoles) {
			start = len(tmpRoles)
		}
		if end > len(tmpRoles) {
			end = len(tmpRoles)
		}
		if start < 0 {
			start = 0
		}
		if end < start {
			end = start
		}
		outInfo = tmpRoles[start:end]
	}

	return count, outInfo, nil
}
