// Package interfaces 接口
package interfaces

import (
	"context"
)

//go:generate mockgen -package mock -source ../interfaces/logics.go -destination ../interfaces/mock/mock_logics.go

// CheckResult 权限检查结果
type CheckResult struct {
	Result           bool
	OperatrionOblist map[string][]PolicyObligationItem
}

// AccessorType 访问者类型
type AccessorType int

// 访问者类型
const (
	_                     AccessorType = iota
	AccessorUser                       // 用户
	AccessorDepartment                 // 部门
	AccessorContactor                  // 联系人
	AccessorAnonymous                  // 匿名用户
	AccessorGroup                      // 用户组
	AccessorApp                        // 应用账户
	AccessorGroupInternal              // 内部组
	AccessorRole                       // 角色
)

// Visitor 访问者信息
type Visitor struct {
	ID string

	// TokenID 在 JSON 序列化和反序列化时会被忽略，用于防止令牌在持久化过程中泄露
	// 如需在反序列化时获取 TokenID，请通过代码手动处理
	TokenID   string `json:"-"`
	IP        string
	Mac       string
	UserAgent string
	ClientID  string
	Type      VisitorType
	ClientType
	Language string
}

// LogicsEvent  处理来自其他服务的事件
// TODO 1.可变参数函数 2.自动生成
type LogicsEvent interface {
	/*
		Org
	*/
	// OrgNameModified 更新组织架构显示名, 不区分类型使用 RegisterOrgNameModified。区分类型使用具体类型如 RegisterUserNameModified
	OrgNameModified(id, name string, orgType AccessorType) error
	// RegisterOrgNameModified 注册组织架构显示名更新
	RegisterOrgNameModified(f func(string, string) error)
	// RegisterUserNameModified 注册用户名称更新
	RegisterUserNameModified(f func(string, string) error)
	// RegisterDepartmentNameModified 注册部门名称更新
	RegisterDepartmentNameModified(f func(string, string) error)
	// RegisterUserGroupNameModified 注册用户组名称更新
	RegisterUserGroupNameModified(f func(string, string) error)

	/*
		User
	*/
	// UserDeleted 用户被删除
	UserDeleted(userID string) (err error)
	// RegisterUserDeleted 注册用户被删除
	RegisterUserDeleted(f func(string) error)

	/*
		Department
	*/
	// DepartmentDeleted 部门被删除
	DepartmentDeleted(depID string) (err error)
	// RegisterDepartmentDeleted 注册部门被删除
	RegisterDepartmentDeleted(f func(string) error)

	/*
		UserGroup
	*/
	// UserGroupDeleted 用户组被删除
	UserGroupDeleted(groupID string) (err error)
	// RegisterUserGroupDeleted 注册用户组被删除
	RegisterUserGroupDeleted(f func(string) error)

	/*
		App
	*/
	// AppDeleted 删除应用账户权限(文档权限和文档库权限)和所有者
	AppDeleted(appID string) (err error)
	// RegisterAppDeleted 注册应用账户删除
	RegisterAppDeleted(f func(string) error)

	// AppNameModified 更新应用账户名称(文档和文档库权限表、所有者表)
	AppNameModified(info *AppInfo) (err error)
	// RegisterAppNameModified 注册应用账户名称更新
	RegisterAppNameModified(f func(*AppInfo) error)

	/*
		Role
	*/
	// RoleDeleted 角色被删除
	RoleDeleted(roleID string) (err error)
	// RegisterRoleDeleted 注册角色被删除
	RegisterRoleDeleted(f func(string) error)

	// RoleNameModified 更新角色名称
	RoleNameModified(roleID, name string) (err error)
	// RegisterRoleNameModified 注册角色名称更新
	RegisterRoleNameModified(f func(string, string) error)
}

// OperationName 操作名称结构体
type OperationName struct {
	Language string `json:"language"`
	Value    string `json:"value"`
}

// OperationScopeType 操作范围类型
type OperationScopeType string

// 操作范围类型定义
const (
	ScopeType     OperationScopeType = "type"
	ScopeInstance OperationScopeType = "instance"
)

// ResourceTypeOperationResponse 操作结构体
// Name是string, 用于指定语言后的操作名称
type ResourceTypeOperationResponse struct {
	ID          string
	Name        string
	Description string
}

// ResourceTypeOperation 操作结构体
// Name是一个数组，用于支持多语言
type ResourceTypeOperation struct {
	ID          string               `json:"id"`
	Name        []OperationName      `json:"name"`
	Description string               `json:"description"`
	Scope       []OperationScopeType `json:"scope"`
}

type ChildrenResourceTypeOperation struct {
	ResourceTypeID string
	Name           string
	Operations     []ResourceTypeOperationResponse
	Children       []ChildrenResourceTypeOperation
}

// ResourceType 资源类型结构体
type ResourceType struct {
	ID          string
	Name        string
	Description string
	InstanceURL string
	DataStruct  string
	CreateTime  int64
	Operation   []ResourceTypeOperation
	Hidden      bool
	ModifyTime  int64
}

type ResourceTypePagination struct {
	Offset int
	Limit  int
}

// LogicsResourceType 资源类型逻辑层接口
type LogicsResourceType interface {
	// 资源类型获取
	GetPagination(ctx context.Context, visitor *Visitor, params ResourceTypePagination) (count int, resourceTypes []ResourceType, err error)
	// 资源类型设置
	Set(ctx context.Context, visitor *Visitor, resourceType *ResourceType) error
	// 资源类型删除
	Delete(ctx context.Context, visitor *Visitor, resourceTypeID string) error
	// 获取资源
	GetByID(ctx context.Context, visitor *Visitor, resourceTypeID string) (resource ResourceType, err error)

	// 获取资源类型所有操作
	GetAllOperation(ctx context.Context, visitor *Visitor, resourceTypeID string, scope OperationScopeType) (operations []ResourceTypeOperationResponse,
		childrenOperations []ChildrenResourceTypeOperation, err error)

	// 获取资源
	GetByIDsInternal(ctx context.Context, resourceTypeIDs []string) (resourceMap map[string]ResourceType, err error)

	// 获取所有资源类型, 不包含隐藏的资源类型
	GetAllInternal(ctx context.Context) (resourceTypes []ResourceType, err error)

	// 资源类型批量添加
	InitResourceTypes(ctx context.Context, resourceTypes []ResourceType) error

	// 资源类型设置-私有接口
	SetPrivate(ctx context.Context, resourceType *ResourceType) error
}

// LogicsResourceTypeHierarchy 资源类型层级关系逻辑层接口
type LogicsResourceTypeHierarchy interface {
	// SetPrivate 设置资源类型层级关系
	SetPrivate(ctx context.Context, visitor *Visitor, hierarchie *ResourceTypeHierarchy) error
	// Get 获取资源类型层级关系
	Get(ctx context.Context, visitor *Visitor, resourceTypeID string) (hierarchy ResourceTypeHierarchy, err error)

	// GetAll 获取所有资源类型层级关系
	GetAll(ctx context.Context) (resourceTypeHierarchyMap map[string]ResourceTypeHierarchy, err error)

	// 层级关系查询接口
	// HasHierarchy 判断一个资源类型是否有设置过层级关系
	HasHierarchy(ctx context.Context, visitor *Visitor, resourceTypeID string) (hasHierarchy bool, err error)
}

// PolicyIncludeType 策略包含类型
type PolicyIncludeType int

// 策略包含类型定义
const (
	_                           PolicyIncludeType = iota
	PolicyIncludeObligationType                   // 义务类型
	PolicyIncludeObligation                       // 义务
)

type AccessorPolicyParam struct {
	AccessorID   string
	AccessorType AccessorType
	Offset       int
	Limit        int
	ResourceType string
	ResourceID   string
	Include      []PolicyIncludeType // 返回信息包含类型 可选值: obligation_type,obligation
}

type ResourcePolicyPagination struct {
	ResourceID         string
	ResourceType       string
	Offset             int
	Limit              int
	Include            []PolicyIncludeType // 返回信息包含类型 可选值: obligation_type,obligation
	SubResourceTypeIDs []string            // 子资源类型ID列表, 目前由层级关系确定该参数
}

type PolicyIncludeResp struct {
	ObligationTypes []ObligationTypeInfo
	Obligations     []ObligationInfo
}

// LogicsPolicy 策略逻辑层接口
type LogicsPolicy interface {
	// 获取策略
	GetPagination(ctx context.Context, visitor *Visitor, params PolicyPagination) (count int, policies []PolicyInfo, err error)
	// 新增策略
	Create(ctx context.Context, visitor *Visitor, policys []PolicyInfo) (policyIDs []string, err error)

	// 新增策略
	CreatePrivate(ctx context.Context, policys []PolicyInfo) error

	// 更新策略
	Update(ctx context.Context, visitor *Visitor, policys []PolicyInfo) error
	// 删除策略
	Delete(ctx context.Context, visitor *Visitor, id []string) error

	// 删除策略 根据资源id删除策略
	DeleteByResourceIDs(ctx context.Context, resources []PolicyDeleteResourceInfo) error

	// 更新资源实例名称
	UpdateResourceName(ctx context.Context, resourceID, resourceType, name string) error

	// 更新资源实例祖先信息
	UpdateResourceAncestors(ctx context.Context, resourceID, resourceType string, ancestors []Ancestor) error

	// 获取策略
	GetResourcePolicy(ctx context.Context, visitor *Visitor, params ResourcePolicyPagination) (count int, policies []PolicyInfo, includeResp PolicyIncludeResp, err error)

	// 获取访问者策略
	GetAccessorPolicy(ctx context.Context, visitor *Visitor, param AccessorPolicyParam) (count int, policies []PolicyInfo, includeResp PolicyIncludeResp, err error)

	// 删除过期策略
	DeleteByEndTime(curTime int64) error

	// 初始化策略
	InitPolicy(ctx context.Context, policys []PolicyInfo) error
}

// ResourceInfo 资源对象信息, 用于策略计算
// 其他字段为系统字段，id,type为必传字段
type ResourceInfo struct {
	ID        string
	Type      string
	Name      string
	Ancestors []Ancestor
}

// AccessorInfo 访问者对象信息, 用于策略计算
// 其他字段为系统属性，id,type为必传字段
type AccessorInfo struct {
	ID   string
	Type VisitorType
	Name string
}

// PolicCalcyIncludeType 策略计算包含类型
type PolicCalcyIncludeType int

// 策略计算包含类型定义
const (
	_                                     PolicCalcyIncludeType = iota
	PolicCalcyIncludeOperationObligations                       // 操作和义务
)

// LogicsPolicyCalc 策略计算逻辑层接口
type LogicsPolicyCalc interface {
	// 检查接口
	Check(ctx context.Context, resource *ResourceInfo, accessor *AccessorInfo, operation []string, include []PolicCalcyIncludeType) (checkResult CheckResult, err error)
	// 获取指定资源类型上的资源列表
	GetResourceList(ctx context.Context, resourceTypeID string, accessor *AccessorInfo, operation []string, include []PolicCalcyIncludeType) (resources []ResourceInfo,
		resourceOperationObligationMap map[string]map[string][]PolicyObligationItem, err error)
	// 过滤资源列表
	ResourceFilter(ctx context.Context, resources []ResourceInfo, accessor *AccessorInfo, operation []string, include []PolicCalcyIncludeType) (result []ResourceInfo,
		resourceOperationMap map[string][]string, resourceOperationObligationMap map[string]map[string][]PolicyObligationItem, err error)
	// 资源操作
	GetResourceOperation(ctx context.Context, resources []ResourceInfo, accessor *AccessorInfo) (resourceOperationMap map[string][]string,
		resourceOperationObligationMap map[string]map[string][]PolicyObligationItem, err error)
	// 获取资源类型操作
	GetResourceTypeOperation(ctx context.Context, resourceTypes []string, accessor *AccessorInfo) (resourceTypeOperationMap map[string][]string, err error)
}

type ResourceTypeScope struct {
	ResourceTypeID   string
	ResourceTypeName string
}

type ResourceTypeScopeInfo struct {
	Unlimited bool
	Types     []ResourceTypeScope
}

type RoleSource int

const (
	_ RoleSource = iota
	RoleSourceSystem
	RoleSourceBusiness
	RoleSourceUser
)

// RoleInfo 角色信息
type RoleInfo struct {
	ID          string
	Name        string
	Description string
	RoleSource
	ResourceTypeScopeInfo
	CreateTime int64
	ModifyTime int64
}

type ResourceTypeScopeWithOperation struct {
	ID                string
	Name              string
	Description       string
	InstanceURL       string
	DataStruct        string
	TypeOperation     []ResourceTypeOperationResponse
	InstanceOperation []ResourceTypeOperationResponse
	Children          []ResourceTypeScopeWithOperation
}

type ResourceTypeScopeInfoWithOperation struct {
	Unlimited bool
	Types     []ResourceTypeScopeWithOperation
}

// RoleInfoWithResourceTypeOperation 带有资源类型操作的角色信息
type RoleInfoWithResourceTypeOperation struct {
	ID          string
	Name        string
	Description string
	RoleSource
	ResourceTypeScopesInfo ResourceTypeScopeInfoWithOperation
	CreateTime             int64
	ModifyTime             int64
}

// SortFiled 排序字段
type SortFiled int

const (
	_ SortFiled = iota

	// DateCreated 创建时间
	DateCreated

	// Name 名称
	Name
)

// RoleSearchInfo 列举信息
type RoleSearchInfo struct {
	Offset      int
	Limit       int
	Keyword     string
	RoleSources []RoleSource
}

type AccessorRoleSearchInfo struct {
	AccessorID   string
	AccessorType AccessorType
	Offset       int
	Limit        int
	RoleSources  []RoleSource
}

// ResourceTypeRoleSearchInfo 资源类型角色列举信息
type ResourceTypeRoleSearchInfo struct {
	Offset         int
	Limit          int
	Keyword        string
	ResourceTypeID string
}

// RoleMemberInfo 角色成员信息
type RoleMemberInfo struct {
	ID         string
	MemberType AccessorType
	Name       string
	ParentDeps [][]Department
	CreateTime int64
	ModifyTime int64
}

// NameInfo 名称信息
type NameInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// RoleMemberSearchInfo 列举信息
type RoleMemberSearchInfo struct {
	Offset      int
	Limit       int
	MemberTypes []AccessorType
	Keyword     string
}

type RoleInfoParam struct {
	ResourceTypeViewMode ResourceTypeViewMode
}

type ResourceTypeViewMode string

const (
	ResourceTypeViewModeFlat      ResourceTypeViewMode = "flat"
	ResourceTypeViewModeHierarchy ResourceTypeViewMode = "hierarchy"
)

// LogicsRole 角色
type LogicsRole interface {
	// InitRoles 初始化角色
	InitRoles(ctx context.Context, roles []RoleInfo) (err error)

	// AddRole 角色创建
	AddRole(ctx context.Context, visitor *Visitor, role *RoleInfo) (id string, err error)

	// DeleteRole 角色删除
	DeleteRole(ctx context.Context, visitor *Visitor, groupID string) (err error)

	// ModifyRole 角色修改
	ModifyRole(ctx context.Context, visitor *Visitor, groupID, name string, nameChanged bool,
		description string, descriptionChanged bool, resourceTypeScopes ResourceTypeScopeInfo, resourceTypeScopesChanged bool) (err error)

	// GetRoles 角色列举
	GetRoles(ctx context.Context, visitor *Visitor, info RoleSearchInfo) (count int, outInfo []RoleInfo, err error)

	// GetRoleByID 获取指定的角色
	GetRoleByID(ctx context.Context, visitor *Visitor, roleID string, param RoleInfoParam) (info RoleInfoWithResourceTypeOperation, err error)

	// GetRolesByIDs 批量获取角色
	GetRolesByIDs(ctx context.Context, roleIDs []string) (infoMap map[string]RoleInfo, err error)

	// AddOrDeleteRoleMemebers 批量删除或者添加角色成员
	AddOrDeleteRoleMemebers(ctx context.Context, visitor *Visitor, method, roleID string, infos map[string]RoleMemberInfo) (err error)

	// InitRoleMemebers 批量初始化角色成员
	InitRoleMemebers(ctx context.Context, infos map[string][]RoleMemberInfo) (err error)

	// GetRoleMembers 角色成员列举
	GetRoleMembers(ctx context.Context, visitor *Visitor, roleID string, info RoleMemberSearchInfo) (count int, outInfo []RoleMemberInfo, err error)

	// GetRoleByMembers 通过成员获取角色
	GetRoleByMembers(ctx context.Context, memberIDs []string) (outInfo []RoleInfo, err error)

	// 角色查询接口
	// GetResourceTypeRoles 根据资源类型列举角色
	GetResourceTypeRoles(ctx context.Context, visitor *Visitor, info ResourceTypeRoleSearchInfo) (count int, outInfo []RoleInfo, err error)

	// GetAccessorRoles根据访问者列举角色
	GetAccessorRoles(ctx context.Context, param AccessorRoleSearchInfo) (count int, outInfo []RoleInfo, err error)
}

type ObligationType interface {
	// 添加义务类型
	Set(ctx context.Context, visitor *Visitor, info *ObligationTypeInfo) (err error)
	// SetPrivate 设置义务类型-私有接口
	SetPrivate(ctx context.Context, visitor *Visitor, info *ObligationTypeInfo) (err error)
	// 删除义务类型
	Delete(ctx context.Context, visitor *Visitor, obligationTypeID string) (err error)
	// 指定ID获取义务类型
	GetByID(ctx context.Context, visitor *Visitor, obligationTypeID string) (info ObligationTypeInfo, err error)

	// 列举义务类型
	Get(ctx context.Context, visitor *Visitor, info *ObligationTypeSearchInfo) (count int, resultInfos []ObligationTypeInfo, err error)

	// 指定ID批量获取义务类型
	GetByIDSInternal(ctx context.Context, obligationTypeIDs map[string]bool) (infos []ObligationTypeInfo, err error)

	// 义务类型批量初始化
	InitObligationTypes(ctx context.Context, obligationTypes []ObligationTypeInfo) error

	// 查询接口
	// 查询义务类型
	Query(ctx context.Context, visitor *Visitor, queryInfo *QueryObligationTypeInfo) (resultInfos map[string][]ObligationTypeInfo, err error)
	// 查询义务类型V2
	QueryV2(ctx context.Context, visitor *Visitor, queryInfo *QueryObligationTypeInfoV2) (resultInfos map[string]map[string][]ObligationTypeInfo, err error)
}

type LogicsObligation interface {
	// 添加义务
	Add(ctx context.Context, visitor *Visitor, info *ObligationInfo) (id string, err error)
	// 更新义务
	Update(ctx context.Context, visitor *Visitor, obligationID string, name string, nameChanged bool, description string, descriptionChanged bool, value any, valueChanged bool) (err error)
	// 删除义务
	Delete(ctx context.Context, visitor *Visitor, obligationID string) (err error)
	// 指定ID获取义务
	GetByID(ctx context.Context, visitor *Visitor, obligationID string) (info ObligationInfo, err error)
	// 获取义务
	Get(ctx context.Context, visitor *Visitor, info *ObligationSearchInfo) (count int, resultInfos []ObligationInfo, err error)

	// 指定ID批量获取义务
	GetByIDSInternal(ctx context.Context, obligationIDs map[string]bool) (infos []ObligationInfo, err error)

	// 查询接口
	// 查询义务
	Query(ctx context.Context, visitor *Visitor, queryInfo *QueryObligationInfo) (resultInfos map[string][]ObligationInfo, err error)
}
