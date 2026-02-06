// Package interfaces 接口
package interfaces

import (
	"context"
	"database/sql"
)

//go:generate mockgen -package mock -source ../interfaces/dbaccess.go -destination ../interfaces/mock/mock_dbaccess.go

type DBResourceType interface {
	// 资源类型获取
	GetPagination(ctx context.Context, params ResourceTypePagination) (count int, resourceTypes []ResourceType, err error)
	// 资源类型设置
	Set(ctx context.Context, resourceType *ResourceType) error
	// 资源类型删除
	Delete(ctx context.Context, resourceTypeID string) error
	// 获取资源
	GetByIDs(ctx context.Context, resourceTypeIDs []string) (resourceMap map[string]ResourceType, err error)

	// 获取所有资源类型, 不包含隐藏的资源类型
	GetAllInternal(ctx context.Context) (resourceTypes []ResourceType, err error)
	// 获取所有资源类型, 包含隐藏的资源类型
	GetAllInternalWithHidden(ctx context.Context) (resourceTypes []ResourceType, err error)
}

// ResourceTypeHierarchy 资源类型层级关系
type ResourceTypeHierarchy struct {
	ResourceTypeID string
	Children       []ResourceTypeHierarchy
}

// DBResourceTypeHierarchy 资源类型层级关系数据库接口
type DBResourceTypeHierarchy interface {
	// Set 设置资源类型层级关系
	Set(ctx context.Context, hierarchy *ResourceTypeHierarchy) error
	// GetAll 获取所有资源类型层级关系
	GetAll(ctx context.Context) (hierarchyMap map[string]ResourceTypeHierarchy, err error)
}

// PolicyPagination 策略分页查询参数
type PolicyPagination struct {
	ResourceID   string
	ResourceType string
	Offset       int
	Limit        int
}

type PolicyObligationItem struct {
	TypeID string // 义务类型ID
	ID     string // 义务ID
	Value  any    // 义务值
}

type PolicyOperationItem struct {
	ID          string
	Name        string
	Obligations []PolicyObligationItem // 义务列表
}

type PolicyOperation struct {
	Allow []PolicyOperationItem
	Deny  []PolicyOperationItem
}

// Ancestor 资源祖先信息
type Ancestor struct {
	ID   string
	Type string
	Name string
}

// PolicyInfo 策略信息
type PolicyInfo struct {
	ID           string
	ResourceID   string
	ResourceType string
	ResourceName string
	Ancestors    []Ancestor
	HasAncestors bool
	AccessorID   string
	AccessorType
	AccessorName string
	ParentDeps   [][]Department
	Operation    PolicyOperation
	Condition    string
	EndTime      int64
	CreateTime   int64
	ModifyTime   int64
}

type PolicyDeleteResourceInfo struct {
	ID   string
	Type string
}

// DBPolicy 策略数据库接口
type DBPolicy interface {
	// 获取策略
	GetPagination(ctx context.Context, params PolicyPagination) (count int, policies []PolicyInfo, err error)
	// 新增策略
	Create(ctx context.Context, policys []PolicyInfo, tx *sql.Tx) error
	// 更新策略
	Update(ctx context.Context, policys []PolicyInfo, tx *sql.Tx) error
	// 删除策略
	Delete(ctx context.Context, ids []string) error
	// 获取资源策略 policies[资源实例ID][]策略
	GetByResourceIDs(ctx context.Context, resourceType string, resourceIDs []string) (policies map[string][]PolicyInfo, err error)
	// 获取资源策略
	GetByPolicyIDs(ctx context.Context, policyIDs []string) (policies map[string]PolicyInfo, err error)

	// 删除策略 根据资源id删除策略
	DeleteByResourceIDs(ctx context.Context, resources []PolicyDeleteResourceInfo) error

	// 删除策略 根据访问者id删除策略
	DeleteByAccessorIDs(accessorIDs []string) error
	// UpdateAccessorName 更新访问者名称
	UpdateAccessorName(accessorID, name string) error

	// 更新资源实例名称
	UpdateResourceName(ctx context.Context, resourceID, resourceType, name string) error

	// 更新资源实例祖先信息
	UpdateResourceAncestors(ctx context.Context, resourceID, resourceType string, ancestors []Ancestor) error

	// 删除过期策略
	DeleteByEndTime(curTime int64) error

	// 获取访问者策略
	GetAccessorPolicy(ctx context.Context, param AccessorPolicyParam) (count int, policies []PolicyInfo, err error)

	GetResourcePolicies(ctx context.Context, params ResourcePolicyPagination) (count int, policies []PolicyInfo, err error)
}

// DBPolicyCalc 策略计算数据库接口
type DBPolicyCalc interface {
	// 批量资源和访问令牌获取配置
	GetPoliciesByResourcesAndAccessToken(ctx context.Context, resourceInfo []ResourceInfo, accessToken []string) (policies []PolicyInfo, err error)
	// 通过访问令牌，获取资源类型上的所有配置
	GetPoliciesByResourceTypeAndAccessToken(ctx context.Context, resourceTypeID string, accessToken []string) (policies []PolicyInfo, err error)
	// 通过访问令牌，批量获取资源类型的配置(不包含具体资源实例的配置，只包含资源实例id为*的配置)
	GetPoliciesByResourceTypes(ctx context.Context, resourceTypes, accessToken []string) (policies []PolicyInfo, err error)
}

// DBRole role 数据访问层处理接口
type DBRole interface {
	// AddRole 添加角色
	AddRoles(ctx context.Context, roles []RoleInfo) (err error)

	// DeleteRole 删除角色
	DeleteRole(ctx context.Context, id string) (err error)

	// ModifyRole 修改角色
	ModifyRole(ctx context.Context, id, name string, nameChanged bool, description string, descriptionChanged bool,
		resourceTypeScopes ResourceTypeScopeInfo, resourceTypeScopesChanged bool) (err error)

	// SetRoleByID 根据ID修改角色
	SetRoleByID(ctx context.Context, id string, role *RoleInfo) (err error)

	// GetRoles 列举符合条件的角色
	GetRoles(ctx context.Context, info RoleSearchInfo) (roles []RoleInfo, err error)

	// GetRolesSum 列举符合条件的角色数量
	GetRolesSum(ctx context.Context, info RoleSearchInfo) (num int, err error)

	// GetRoleByID 获取指定的角色
	GetRoleByID(ctx context.Context, id string) (info RoleInfo, err error)

	// GetRoleByName 根据名称获取指定的角色
	GetRoleByName(ctx context.Context, name string) (info RoleInfo, err error)

	// GetRoleByIDs 获取指定的角色
	GetRoleByIDs(ctx context.Context, ids []string) (infoMap map[string]RoleInfo, err error)

	// GetAllUserRolesInternal 列举所有用户创建的角色
	GetAllUserRolesInternal(ctx context.Context, keyword string) (roles []RoleInfo, err error)
}

// DBRoleMember role member 数据访问层处理接口
type DBRoleMember interface {
	// AddRoleMembers 批量添加角色成员
	AddRoleMembers(ctx context.Context, id string, infos []RoleMemberInfo) (err error)
	// DeleteRoleMembers 批量删除角色成员
	DeleteRoleMembers(ctx context.Context, id string, membersIDs []string) (err error)

	// 根据角色ID删除成员
	DeleteByRoleID(ctx context.Context, roleID string) error

	// GetRoleMembersNum 列举角色成员数量
	GetRoleMembersNum(ctx context.Context, id string, info RoleMemberSearchInfo) (num int, err error)

	// GetPaginationByRoleID 列举角色成员
	GetPaginationByRoleID(ctx context.Context, id string, info RoleMemberSearchInfo) (outInfo []RoleMemberInfo, err error)

	// GetRoleMembersByRoleID 根据角色ID获取角色成员
	GetRoleMembersByRoleID(ctx context.Context, id string) (outInfo []RoleMemberInfo, err error)

	// GetRoleByMembers 通过成员获取角色
	GetRoleByMembers(ctx context.Context, memberIDs []string) (outInfo []RoleInfo, err error)

	// 删除成员 根据成员id
	DeleteByMemberIDs(memberIDs []string) error
	// UpdateMemberName 更新成员名称
	UpdateMemberName(memberID, name string) error
}

type ObligationOperation struct {
	ID   string
	Name string
}

type ObligationOperationsScopeInfo struct {
	Unlimited  bool
	Operations []ObligationOperation
}

type ObligationResourceTypeScope struct {
	ResourceTypeID   string
	ResourceTypeName string
	OperationsScope  ObligationOperationsScopeInfo
}

type ObligationResourceTypeScopeInfo struct {
	Unlimited bool
	Types     []ObligationResourceTypeScope
}

type ObligationTypeInfo struct {
	ID                string
	Name              string
	Description       string
	ResourceTypeScope ObligationResourceTypeScopeInfo
	Schema            any
	DefaultValue      any
	UiSchema          any
	CreatedTime       int64
	ModifyTime        int64
}

type ObligationTypeSearchInfo struct {
	Offset int
	Limit  int
}

type QueryObligationTypeInfo struct {
	ResourceType string
	Operation    []string
}

type QueryObligationTypeInfoV2 struct {
	ResourceTypeIDs []string
}

type ObligationInfo struct {
	ID          string
	TypeID      string
	Name        string
	Description string
	Value       any
	CreatedTime int64
	ModifyTime  int64
}

type ObligationSearchInfo struct {
	Offset int
	Limit  int
}

type QueryObligationInfo struct {
	ResourceType      string
	Operation         []string
	ObligationTypeIDs []string
}

type DBObligationType interface {
	// 设置义务类型
	Set(ctx context.Context, info *ObligationTypeInfo) (err error)
	// 删除义务类型
	Delete(ctx context.Context, obligationTypeID string) (err error)
	// 获取义务类型
	Get(ctx context.Context, info *ObligationTypeSearchInfo) (count int, resultInfos []ObligationTypeInfo, err error)
	// 指定ID获取义务类型
	GetByID(ctx context.Context, obligationTypeID string) (info ObligationTypeInfo, err error)

	// 通过ID批量获取义务类型
	GetByIDs(ctx context.Context, obligationTypeIDs []string) (infos []ObligationTypeInfo, err error)

	// 获取所有义务类型
	GetAll(ctx context.Context) (resultInfos []ObligationTypeInfo, err error)
}

type DBObligation interface {
	// 设置义务
	Add(ctx context.Context, info *ObligationInfo) (err error)
	// 更新义务
	Update(ctx context.Context, obligationID, name string, nameChanged bool, description string, descriptionChanged bool, value any, valueChanged bool) (err error)
	// 删除义务
	Delete(ctx context.Context, obligationID string) (err error)
	// 指定ID获取义务
	GetByID(ctx context.Context, obligationID string) (info ObligationInfo, err error)
	// 批量获取义务
	GetByIDs(ctx context.Context, obligationIDs []string) (infos []ObligationInfo, err error)
	// 获取义务
	Get(ctx context.Context, info *ObligationSearchInfo) (count int, resultInfos []ObligationInfo, err error)

	GetByObligationTypeIDs(ctx context.Context, obligationTypeID map[string]bool) (resultInfos map[string][]ObligationInfo, err error)
}
