// Package interfaces AnyShare 接口
package interfaces

import (
	"context"
	"database/sql"
)

//go:generate mockgen -package mock -source ../interfaces/logics.go -destination ../interfaces/mock/mock_logics.go

// OrgNameInfo 组织架构名称信息
type OrgNameInfo struct {
	UserNames      []NameInfo
	DepartNames    []NameInfo
	ContactorNames []NameInfo
	GroupNames     []NameInfo
	AppNames       []NameInfo
}

// OrgIDInfo 组织架构ID信息
type OrgIDInfo struct {
	UserIDs      []string
	DepartIDs    []string
	ContactorIDs []string
	GroupIDs     []string
	AppIDs       []string
}

// SearchClientInfo 客户端搜索信息
type SearchClientInfo struct {
	Keyword     string
	Limit       int
	Offset      int
	BShowMember bool
	BShowGroup  bool
}

// EmailInfo 邮箱信息
type EmailInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// GMSearchOutInfo 组和组成员关键字搜索返回信息
type GMSearchOutInfo struct {
	MemberInfos []MemberInfo
	MemberNum   int
	GroupInfos  []NameInfo
	GroupNum    int
}

// OrgEmailInfo 组织邮箱信息
type OrgEmailInfo struct {
	UserEmails   []EmailInfo
	DepartEmails []EmailInfo
}

// OrgShowPageInfo 组织架构展示信息
type OrgShowPageInfo struct {
	BShowUsers            bool
	BShowDeparts          bool
	Keyword               string
	Role                  Role
	Offset                int
	Limit                 int
	BShowSubDepart        bool
	BShowSubUser          bool
	BOnlyShowEnabledUser  bool
	BOnlyShowAssignedUser bool
	BShowDepartManager    bool
	BShowDepartRemark     bool
	BShowDepartCode       bool
	BShowDepartEmail      bool
	BShowDepartEnabled    bool
	BShowDepartParentDeps bool
}

// SearchUserInfo 搜索返回用户信息
type SearchUserInfo struct {
	ID             string   `json:"id" binding:"required"`
	Name           string   `json:"name" binding:"required"`
	ParentDepPaths []string `json:"parent_dep_paths" binding:"required"`
	Account        string   `json:"account" binding:"required"`
	Type           string   `json:"type" binding:"required"`
}

// SearchDepartInfo 搜索返回部门信息
type SearchDepartInfo struct {
	ID   string `json:"id" binding:"required"`
	Name string `json:"name" binding:"required"`
	Path string `json:"parent_dep_path" binding:"required"`
	Type string `json:"type" binding:"required"`
}

// LogicsCombine 组合 逻辑处理接口
type LogicsCombine interface {
	// ConvertIDToName  根据用户/部门/联系人组/用户组/应用账户id 获取用户/部门/联系人组/用户组/应用账户显示名
	ConvertIDToName(visitor *Visitor, info *OrgIDInfo, bv2 bool, bStrict bool) (nameInfo OrgNameInfo, err error)

	// GetEmails  根据用户/部门id 获取用户/部门emails
	GetEmails(visitor *Visitor, info *OrgIDInfo) (emailInfo OrgEmailInfo, err error)

	// GetUserAndDepartmentInScope 获取在范围内的用户和部门
	GetUserAndDepartmentInScope(userIDs, deptIDs, rangeIDs []string) (outUserIDs, outDepIDs []string, err error)

	// SearchGroupAndMemberInfoByKey 客户端搜索组和组成员信息
	SearchGroupAndMemberInfoByKey(info *SearchClientInfo) (out GMSearchOutInfo, err error)

	// SearchInOrgTree 在组织架构内搜索用户和信息
	SearchInOrgTree(ctx context.Context, visitor *Visitor, info OrgShowPageInfo) (users []SearchUserInfo, departs []SearchDepartInfo, userNum, departNum int, err error)
}

// ContactorMemberInfo 联系人组成员信息
type ContactorMemberInfo struct {
	ContactorID string
	MemberIDs   []string
}

// LogicsContactor contactor逻辑处理接口
type LogicsContactor interface {
	// ConvertContactorName 根据Contactorid批量获取联系人组名称
	ConvertContactorName(contactorIDs []string, bStrict bool) ([]NameInfo, error)

	// DeleteContactor 删除联系人组
	DeleteContactor(visitor *Visitor, groupID string) error

	// GetContactorMembers 获取联系人组成员
	GetContactorMembers(ctx context.Context, contactorIDs []string) ([]ContactorMemberInfo, error)
}

// DepartMemberID 指定部门的成员ID（直属成员）
type DepartMemberID struct {
	UserIDs   []string `json:"user_ids" binding:"required"`
	DepartIDs []string `json:"department_ids" binding:"required"`
}

// ObjectBaseInfo  对象基本信息
type ObjectBaseInfo struct {
	ID      string `json:"id" binding:"required"`
	Name    string `json:"name" binding:"required"`
	Type    string `json:"type" binding:"required"`
	Code    string `json:"-"`
	ThirdID string `json:"-"`
}

// DepartInfoScope 获取部门信息范围
type DepartInfoScope struct {
	BShowName   bool
	BParentDeps bool
	BManagers   bool
	BManager    bool
	BCode       bool
	BEnabled    bool
	BRemark     bool
	BEmail      bool
	BThirdID    bool
}

// DepartSearchKeyScope 部门搜索关键字范围
type DepartSearchKeyScope struct {
	BName             bool
	BCode             bool
	BManagerName      bool
	BRemark           bool
	BDirectDepartCode bool
	BEnabled          bool
}

// DepartSearchKey 部门搜索关键字
type DepartSearchKey struct {
	Name             string
	Code             string
	ManagerName      string
	DirectDepartCode string
	Remark           string
	Enabled          bool
	Offset           int
	Limit            int
}

// DepartInfo 部门信息
type DepartInfo struct {
	ID            string
	Name          string
	ParentDeps    []ObjectBaseInfo
	Managers      []NameInfo
	IsRoot        bool
	BDepartExistd bool
	BUserExistd   bool
	Manager       NameInfo
	Code          string
	Enabled       bool
	Remark        string
	Email         string
	ThirdID       string
}

// UserSearchInDepartKeyScope 部门下搜索关键信息
type UserSearchInDepartKeyScope struct {
	BDepartmentID     bool
	BCode             bool
	BName             bool
	BAccount          bool
	BManagerName      bool
	BDirectDepartCode bool
	BPosition         bool
}

// UserSearchInDepartKey 部门下用户搜索关键字
type UserSearchInDepartKey struct {
	DepartmentID     string
	Code             string
	Name             string
	Account          string
	ManagerName      string
	DirectDepartCode string
	Position         string
	Offset           int
	Limit            int
}

// LogicsDepartment department逻辑处理接口
type LogicsDepartment interface {
	// ConvertDepartmentName 根据departmentid批量获取部门名称
	ConvertDepartmentName(visitor *Visitor, deptIDs []string, bStrict bool) ([]NameInfo, error)

	// GetDepartEmails 根据departmentid批量获取部门邮箱
	GetDepartEmails(visitor *Visitor, deptIDs []string) ([]EmailInfo, error)

	// GetAllChildDeparmentIDs 根据departmentid批量获取子部门ID
	GetAllChildDeparmentIDs(deptIDs []string) (outInfo []string, err error)

	// GetAccessorIDsOfDepartment 获取指定部门的访问令牌
	GetAccessorIDsOfDepartment(depID string) (accessorIDs []string, err error)

	// GetDepartMemberIDs 获取指定部门的成员ID（直属成员）
	GetDepartMemberIDs(depID string) (info DepartMemberID, err error)

	// GetAllDepartUserIDs 获取指定部门的用户ID（所有用户）
	GetAllDepartUserIDs(depID string, bShowAllUser bool) (info []string, err error)

	// GetDepartMemberInfo 获取指定部门的成员信息（直属成员）
	GetDepartMemberInfo(visitor *Visitor, depID string, info OrgShowPageInfo) (depInfo []DepartInfo, depNum int, userInfo []ObjectBaseInfo, userNum int, err error)

	// SearchDepartsByKey 按照关键字搜索部门
	SearchDepartsByKey(ctx context.Context, visitor *Visitor, info OrgShowPageInfo) ([]SearchDepartInfo, int, error)

	// GetDepartsInfo 获取部门信息
	GetDepartsInfo(depIDs []string, scope DepartInfoScope, bCheckID bool) (infos []DepartInfo, err error)

	// GetDepartsInfoByLevel 根据层级获取部门信息
	GetDepartsInfoByLevel(level int) (infos []ObjectBaseInfo, err error)

	// GetAllDepartUserInfos 根据depID获取部门所有子成员基本信息
	GetAllDepartUserInfos(depID string) (infos []UserBaseInfo, err error)

	// DeleteDepart 根据部门ID删除部门
	DeleteDepart(visitor *Visitor, id string) (err error)

	// SearchDeparts 按照关键字搜索部门
	SearchDeparts(ctx context.Context, visitor *Visitor, scope *DepartInfoScope, ks *DepartSearchKeyScope, k *DepartSearchKey, f Role) (out []DepartInfo, num int, err error)
}

// ErrorCodeType 错误码类型
type ErrorCodeType int

const (
	_ ErrorCodeType = iota
	// Number 整数类型错误码
	Number
	// Str 字符串类型错误码
	// NOTE: 编目属性值类型已存在枚举值String，所以这里使用Str
	Str
)

// Visitor 请求信息
type Visitor struct {
	ID string

	// TokenID 在 JSON 序列化和反序列化时会被忽略，用于防止令牌在持久化过程中泄露
	// 如需在反序列化时获取 TokenID，请通过代码手动处理
	TokenID   string `json:"-"`
	IP        string
	Mac       string
	UserAgent string
	Type      VisitorType
	LangType  LangType
	Language
	ErrorCodeType
}

// Language 语言类型
type Language int

// 语言类型
const (
	_                  Language = iota
	SimplifiedChinese           // 简体中文
	TraditionalChinese          // 繁体中文
	AmericanEnglish             // 美国英语
)

// GroupMemberNames 用户组及成员名
type GroupMemberNames struct {
	GroupName   string
	MemberNames []string
}

// LogicsGroup 用户组
type LogicsGroup interface {
	// AddGroup 用户组创建
	AddGroup(ctx context.Context, visitor *Visitor, name, notes string, initalGroupIDs []string) (id string, err error)

	// DeleteGroup 用户组删除
	DeleteGroup(visitor *Visitor, groupID string) (err error)

	// ModifyGroup 用户组修改
	ModifyGroup(visitor *Visitor, groupID, name string, nameChanged bool, notes string, notesChanged bool) (err error)

	// GetGroup 用户组列举
	GetGroup(visitor *Visitor, info SearchInfo) (count int, outInfo []GroupInfo, err error)

	// GetGroupByID 获取指定的用户组
	GetGroupByID(visitor *Visitor, groupID string) (info GroupInfo, err error)

	// AddOrDeleteGroupMemebers 批量删除或者添加用户组成员
	AddOrDeleteGroupMemebers(visitor *Visitor, method, groupID string, infos map[string]GroupMemberInfo) (err error)

	// GetGroupMembersID 批量获取用户组成员id
	GetGroupMembersID(visitor *Visitor, groupIDs []string, bShowAllUser bool) (userIDs, departmentIDs []string, err error)

	// GetGroupMembers 用户组成员列举
	GetGroupMembers(ctx context.Context, visitor *Visitor, groupID string, info SearchInfo) (count int, outInfo []GroupMemberInfo, err error)

	// AddGroupMembers 用户组成员添加
	AddGroupMembers(visitor *Visitor, groupID string, infos map[string]GroupMemberInfo) (err error)

	// DeleteGroupMembers 用户组成员删除
	DeleteGroupMembers(visitor *Visitor, groupID string, infos map[string]GroupMemberInfo) (err error)

	// GetMemberOnClient 客户端列举组成员
	DeleteGroupMemberByMemberID(id string) (err error)

	// SearchGroupByKeyword 用户组关键字搜索
	SearchGroupByKeyword(keyword string, start, limit int) (out []NameInfo, err error)

	// SearchGroupNumByKeyword 用户组关键字搜索符合条件的用户组总数目
	SearchGroupNumByKeyword(keyword string) (num int, err error)

	// SearchMembersByKeyword 用户组成员关键字搜索
	SearchMembersByKeyword(keyword string, start, limit int) (out []MemberInfo, err error)

	// SearchMemberNumByKeyword 用户组成员关键字搜索符合条件的用户组总数目
	SearchMemberNumByKeyword(keyword string) (num int, err error)

	// ConvertGroupName 根据用户组ID获取用户组名
	ConvertGroupName(visitor *Visitor, ids []string, bStrict bool) (nameInfo []NameInfo, err error)

	// GetGroupOnClient 客户端列举组
	GetGroupOnClient(offset, limit int) (info []NameInfo, num int, err error)

	// GetMemberOnClient 客户端列举组成员
	GetMemberOnClient(id string, offset, limit int) (info []MemberSimpleInfo, num int, err error)

	// UserMatch 组内用户匹配
	UserMatch(ctx context.Context, visitor *Visitor, groupID, userName string) (exist bool, uInfo GroupMemberInfo, mInfo []GroupMemberInfo, err error)

	// searchInAllGroupOrg 组内用户搜索
	SearchInAllGroupOrg(ctx context.Context, visitor *Visitor, groupID, userName string, offset, limt int) (
		num int, userIDs []string, uInfos map[string]GroupMemberInfo, mInfos map[string][]GroupMemberInfo, err error)
}

// UserBaseInfoRange 获取的用户信息范围
type UserBaseInfoRange struct {
	ShowRoles          bool
	ShowEnable         bool
	ShowPriority       bool
	ShowCSFLevel       bool
	ShowName           bool
	ShowParentDeps     bool
	ShowParentDepPaths bool
	ShowAccount        bool
	ShowFrozen         bool
	ShowAuthenticated  bool
	ShowEmail          bool
	ShowTelNumber      bool
	ShowThirdAttr      bool
	ShowThirdID        bool
	ShowAvatar         bool
	ShowAuthType       bool
	ShowGroups         bool
	ShowPwdErrCnt      bool
	ShowPwdErrLastTime bool
	ShowLDAPType       bool
	ShowDomanPath      bool
	ShowOssID          bool
	ShowCustomAttr     bool
	ShowManager        bool
	ShowRemark         bool
	ShowCreatedAt      bool
	ShowCode           bool
	ShowPosition       bool
	ShowCSFLevel2      bool
}

// UserBaseInfo 用户基本信息
type UserBaseInfo struct {
	VecRoles       []Role
	Priority       int
	CSFLevel       int
	Enabled        bool
	Name           string
	ID             string
	ParentDeps     [][]ObjectBaseInfo
	ParentDepPaths []string
	Account        string
	Password       string
	Frozen         bool
	Authenticated  bool
	Email          string
	TelNumber      string
	ThirdAttr      string
	ThirdID        string
	Avatar         string
	AuthType       AuthType
	Groups         []GroupInfo
	PwdErrCnt      int
	PwdErrLastTime int64
	LDAPType       LDAPServerType
	DomainPath     string
	OssID          string
	CustomAttr     map[string]interface{}
	Manager        NameInfo
	Code           string
	Remark         string
	CreatedAt      int64
	Position       string
	CSFLevel2      int
}

// UserUpdateRange 修改用户信息范围
type UserUpdateRange struct {
	UpdatePWD  bool
	CustomAttr bool
}

// PwdRetrievalStatus 用户密码找回状态
type PwdRetrievalStatus int

const (
	// 使用iota，增加新的枚举时，必须按照顺序添加，不得在中间插入
	// 否则会改变枚举的值，造成应用账户注册类型与db中记录的type对应不上
	_ PwdRetrievalStatus = iota

	// PRSAvaliable 密码找回功能可用
	PRSAvaliable

	// PRSInvalidAccount 无效账户
	PRSInvalidAccount

	// PRSDisableUser 用户被禁用
	PRSDisableUser

	// PRSUnablePWDRetrieval 密码找回功能未开启
	PRSUnablePWDRetrieval

	// PRSNonLocalUser 非本地用户
	PRSNonLocalUser

	// PRSEnablePwdControl 密码管控开启
	PRSEnablePwdControl
)

// PwdRetrievalInfo 用户密码找回功能信息
type PwdRetrievalInfo struct {
	Status     PwdRetrievalStatus // 密码找回功能状态
	ID         string             // 用户ID
	BTelephone bool               // 是否开启手机号找回密码功能
	Telephone  string             // 绑定手机号
	BEmail     bool               // 是否开启邮箱找回密码功能
	Email      string             // 绑定邮箱
}

// AuthFailedReason 本地认证失败的原因
type AuthFailedReason int

const (
	_ AuthFailedReason = iota

	// InvalidPassword 账户/密码错误
	InvalidPassword

	// InitialPassword 初始密码
	InitialPassword

	// PasswordNotSafe 密码不合符强密码要求
	PasswordNotSafe

	// UnderControlPasswordExpire 管控状态下密码过期
	UnderControlPasswordExpire

	// PasswordExpire 密码过期
	PasswordExpire
)

// LogicsUser user逻辑处理接口
type LogicsUser interface {
	// ConvertUserName 根据userid批量获取用户显示名
	ConvertUserName(visitor *Visitor, userIDs []string, bStrict bool) ([]NameInfo, error)

	// GetUserEmails 根据userid批量获取用户邮箱
	GetUserEmails(visitor *Visitor, userIDs []string) ([]EmailInfo, error)

	// GetAllBelongDepartmentIDs 获取用户所属部门id(直属部门+父部门)
	GetAllBelongDepartmentIDs(userID string) ([]string, error)

	// GetUsersInDepartments 检测用户是否在部门内部， 返回存在的用户ID
	GetUsersInDepartments(userIDs, departmentIDs []string) (outUserIDs []string, err error)

	// GetAccessorIDsOfUser 获取指定用户的访问令牌
	GetAccessorIDsOfUser(userID string) (accessorIDs []string, err error)

	// GetUsersBaseInfo 获取多个用户基本信息
	GetUsersBaseInfo(ctx context.Context, visitor *Visitor, userIDs []string, info UserBaseInfoRange, bStrict bool) ([]UserBaseInfo, error)

	// GetUserBaseInfoByTelephone 根据用户手机号获取用户基本信息
	GetUserBaseInfoByTelephone(ctx context.Context, visitor *Visitor, telephone string, rg UserBaseInfoRange) (result bool, info UserBaseInfo, err error)

	// GetUserInfoByAccount 通过账户名匹配账户信息
	GetUserInfoByAccount(account string, enableIDCardLogin bool, enablePrefixMatch bool) (result bool, user UserBaseInfo, err error)

	// GetUserBaseInfoInScope 获取范围内用户基本信息
	GetUserBaseInfoInScope(visitor *Visitor, role Role, userIDs []string, info UserBaseInfoRange) ([]UserBaseInfo, error)

	// SearchUsersByKeyInScope 在范围内按照关键字搜索用户信息
	SearchUsersByKeyInScope(ctx context.Context, visitor *Visitor, info OrgShowPageInfo) ([]SearchUserInfo, int, error)

	// ModifyUserInfo 修改用户信息
	ModifyUserInfo(visitor *Visitor, bRange UserUpdateRange, info *UserBaseInfo) (err error)

	// GetNorlmalUserInfo 普通用户获取自己信息
	GetNorlmalUserInfo(ctx context.Context, visitor *Visitor, oGetRange UserBaseInfoRange) (out UserBaseInfo, err error)

	// GetPWDRetrievalMethodByAccount 根据用户账户获取密码找回信息
	GetPWDRetrievalMethodByAccount(account string) (info PwdRetrievalInfo, err error)

	// UserAuth 本地认证
	UserAuth(id, plainPassword string) (result bool, reason AuthFailedReason, err error)

	// 更新账户密码错误信息
	UpdatePwdErrInfo(id string, pwdErrCnt int, pwdErrLastTime int64) (err error)

	// IncrementModifyUserInfo 修改用户信息
	IncrementModifyUserInfo(ctx context.Context, visitor *Visitor, bRange UserUpdateRange, info *UserBaseInfo) (err error)

	// SearchUsers 搜索用户信息
	SearchUsers(ctx context.Context, visitor *Visitor, ks *UserSearchInDepartKeyScope, k *UserSearchInDepartKey, f UserBaseInfoRange, r Role) (out []UserBaseInfo, num int, err error)

	// CheckUserNameExistd 检查用户名称是否存在
	CheckUserNameExistd(ctx context.Context, name string) (result bool, err error)

	// GetUserList 获取用户列表
	GetUserList(ctx context.Context, userInfoRange UserBaseInfoRange, direction Direction,
		bShowCreatedAt bool, createdStamp int64, userID string, limit int) (out []UserBaseInfo, num int, hasNext bool, err error)
}

// Role 角色枚举
type Role string

// 系统内置各角色枚举值
const (
	SystemRoleSuperAdmin Role = "7dcfcc9c-ad02-11e8-aa06-000c29358ad6"
	SystemRoleSysAdmin   Role = "d2bd2082-ad03-11e8-aa06-000c29358ad6"
	SystemRoleSecAdmin   Role = "d8998f72-ad03-11e8-aa06-000c29358ad6"
	SystemRoleAuditAdmin Role = "def246f2-ad03-11e8-aa06-000c29358ad6"
	SystemRoleOrgManager Role = "e63e1c88-ad03-11e8-aa06-000c29358ad6"
	SystemRoleOrgAudit   Role = "f06ac18e-ad03-11e8-aa06-000c29358ad6"
	SystemRoleNormalUser Role = "d7242388-7fe5-4e03-8b1e-f2f8f0437848"
)

// 系统内置管理员账户ID 超级管理员ID和系统管理员ID一致
const (
	SystemSysAdmin   string = "266c6a42-6131-4d62-8f39-853e7093701c"
	SystemAuditAdmin string = "94752844-BDD0-4B9E-8927-1CA8D427E699"
	SystemSecAdmin   string = "4bb41612-a040-11e6-887d-005056920bea"

	// 原有sys管理员ID
	SystemOriginSysAdmin string = "234562BE-88FF-4440-9BFF-447F139871A2"

	// 组织架构根ID，用于获取根部门
	RootDepartmentParentID string = "00000000-0000-0000-0000-000000000000"
)

// AnonymousInfo 匿名账户信息
type AnonymousInfo struct {
	ID             string
	ExpiresAtStamp int64
	Password       string
	LimitedTimes   int32
	AccessedTimes  int32
	Type           string
	VerifyMobile   bool // 是否需要手机验证码校验
}

// LogicsAnonymous 逻辑层 匿名账户处理
type LogicsAnonymous interface {
	// Create 创建匿名账户
	Create(info AnonymousInfo) error

	// DeleteByID 删除匿名账户
	DeleteByID(ID string) error

	// Authentication 认证匿名账户
	Authentication(ID, password, referrer string) error

	// DeleteByTime 删除过期匿名账户
	DeleteByTime(curTime int64) error

	// GetByID 获取匿名账户信息
	GetByID(ID string) (*AnonymousInfo, error)
}

// OutboxMsg outbox消息结构体
type OutboxMsg struct {
	Type    int         `json:"type"`
	Content interface{} `json:"content"`
}

// LogicsOutbox 逻辑层 发件箱处理接口
type LogicsOutbox interface {
	// 添加outbox消息
	AddOutboxInfo(opType int, content interface{}, tx *sql.Tx) error

	// 批量添加outbox消息
	AddOutboxInfos(msgs []OutboxMsg, tx *sql.Tx) error

	// RegisterHandlers 注册异步处理函数
	RegisterHandlers(opType int, op func(interface{}) error)

	// notify推送线程
	NotifyPushOutboxThread()
}

// OrgManagerInfo 组织管理员信息
type OrgManagerInfo struct {
	ID         string
	SubUserIDs []string
}

// OrgManagerInfoRange 组织管理员信息范围
type OrgManagerInfoRange struct {
	ShowSubUserIDs bool
}

// LogicsRole 逻辑层 角色接口
type LogicsRole interface {
	// GetRolesByUserIDs 根据用户id数组获取用户角色id,包含普通用户
	GetRolesByUserIDs(userIDs []string) (out map[string]map[Role]bool, err error)

	// GetRolesByUserIDs2 根据用户id数组获取用户角色id,包含普通用户，支持trace
	GetRolesByUserIDs2(ctx context.Context, userIDs []string) (out map[string]map[Role]bool, err error)

	// GetOrgManagersInfo 获取组织管理员信息
	GetOrgManagersInfo(orgIDs []string, rangeInfo OrgManagerInfoRange) (out []OrgManagerInfo, err error)

	// GetUserIDsByRoleIDs 根据角色ID获取角色成员
	GetUserIDsByRoleIDs(ctx context.Context, roles []Role) (out map[Role][]string, err error)
}

// AppType 应用账户类型
type AppType int

const (
	// 使用iota，增加新的枚举时，必须按照顺序添加，不得在中间插入
	// 否则会改变枚举的值，造成应用账户注册类型与db中记录的type对应不上
	_ AppType = iota

	// General 通用类型
	General

	// Specified 专用类型
	Specified

	// Internal 内部类型
	Internal
)

// LogicsApp 应用账户管理接口
type LogicsApp interface {
	// RegisterApp 通用账户注册
	RegisterApp(visitor *Visitor, name, password string, appType AppType) (id string, err error)

	// DeleteApp 删除应用账户
	DeleteApp(visitor *Visitor, id string) (err error)

	// UpdateApp 更新应用账户
	UpdateApp(visitor *Visitor, id string, bName bool, name string, bPwd bool, pwd string) (err error)

	// AppList 获取应用账户列表
	AppList(visitor *Visitor, searchInfo *SearchInfo) (info *[]AppInfo, num int, err error)

	// GetApp 获取应用账户
	GetApp(id string) (appInfo *AppInfo, err error)

	// ConvertAppName 根据应用账户ID获取应用账户名
	// 如果是v2，则返回400019005
	ConvertAppName(ids []string, bV2 bool, bStrict bool) (nameInfo []NameInfo, err error)

	// GenerateAppToken 生成应用账户令牌
	GenerateAppToken(ctx context.Context, visitor *Visitor, appID string) (token string, err error)
}

// AppOrgPerm 应用账户组织架构管理权限信息
type AppOrgPerm struct {
	Subject string          // 应用账户ID
	Object  OrgType         // 应用账户对组织架构权限类型
	Name    string          // 应用账户名称
	EndTime int64           // 权限有效期时间戳
	Value   AppOrgPermValue // 应用账户对组织架构权限值
}

// LogicsOrgPermApp 应用账户组织架构管理权限接口
type LogicsOrgPermApp interface {
	// UpdateAppName 更新应用账户组织架构权限表内应用账户名称
	UpdateAppName(info *AppInfo) error

	// GetAppOrgPerm 获取应用账户组织架构管理权限
	GetAppOrgPerm(visitor *Visitor, id string, types []OrgType) (perms []AppOrgPerm, err error)

	// SetAppOrgPerm 设置应用账户组织架构管理权限
	SetAppOrgPerm(visitor *Visitor, id string, perms []AppOrgPerm) (err error)

	// DeleteAppOrgPerm 删除应用账户组织架构管理权限
	DeleteAppOrgPerm(visitor *Visitor, id string, types []OrgType) (err error)
}

// LogicsAvatar 头像相关接口
type LogicsAvatar interface {
	// Get 根据用户ID 获取用户URL
	Get(ctx context.Context, visitor *Visitor, userID string) (url string, err error)

	// Update 更新用户头像
	Update(ctx context.Context, visitor *Visitor, typ string, data []byte) (err error)
}

// LogicsEvent  处理来自其他服务的事件
type LogicsEvent interface {
	// 部门删除
	DeptDeleted(deptID string) (err error)

	// RegisterDeptDeleted
	RegisterDeptDeleted(f func(string) error)

	// OrgManagerChanged 部门管理员变更事件
	OrgManagerChanged(userIDs []string) (err error)

	// RegisterDepartResponserChanged
	RegisterDepartResponserChanged(f func([]string) error)

	// UserDeleted 部门被删除
	UserDeleted(deptID string) error

	// RegisterUserDeleted
	RegisterUserDeleted(f func(string) error)

	// UserNameChanged 用户名变更
	UserNameChanged(userID string, newName string) error

	// RegisterUserNameChanged
	RegisterUserNameChanged(f func(string, string) error)
}

// LogicsInternalGroup 内部组相关接口
type LogicsInternalGroup interface {
	// AddGroup 增加内部组
	AddGroup() (id string, err error)

	// DeleteGroup 删除内部组
	DeleteGroup(ids []string) (err error)

	// GetGroupMemberByID 根据内部组ID获取成员ID
	GetGroupMemberByID(id string) (outInfos []InternalGroupMember, err error)

	// UpdateMembers 更新成员
	UpdateMembers(id string, infos []InternalGroupMember) (err error)

	// GetBelongGroups 获取用户成员所属内部组
	GetBelongGroups(info InternalGroupMember) (ids []string, err error)
}

// LangType 语言类型
type LangType int

const (
	// 使用iota，增加新的枚举时，必须按照顺序添加，不得在中间插入
	_ LangType = iota

	// LTZHCN 中文简体
	LTZHCN

	// LTZHTW 中文繁体
	LTZHTW

	// LTENUS 英文
	LTENUS
)

// LogicsConfig 配置信息
type LogicsConfig interface {
	// UpdateConfig 设置配置信息
	UpdateConfig(visitor *Visitor, rg map[ConfigKey]bool, config *Config) (err error)

	// CheckDefaultPWD 检查用户初始密码格式
	CheckDefaultPWD(visitor *Visitor, pwd string) (result bool, msg string, err error)

	// GetPWDRetrievalConfig 获取密码找回配置信息
	GetConfig(rg map[ConfigKey]bool) (config Config, err error)

	// GetConfigFromOption 获取配置信息
	GetConfigFromOption(rg map[ConfigKey]bool) (config Config, err error)
}

// OrgPerm 组织架构管理权限信息
type OrgPerm struct {
	SubjectID   string       // 账户ID
	SubjectType VisitorType  // 账户类型
	Object      OrgType      // 账户对组织架构权限类型
	Name        string       // 账户名称
	EndTime     int64        // 权限有效期时间戳
	Value       OrgPermValue // 账户对组织架构权限值
}

// LogicsOrgPerm 账户组织架构管理权限接口
type LogicsOrgPerm interface {
	// SetOrgPerm 设置账户组织架构管理权限
	SetOrgPerm(ctx context.Context, subjectID string, subjectType VisitorType, infos []OrgPerm) (err error)

	// DeleteOrgPerm 删除账户组织架构管理权限
	DeleteOrgPerm(ctx context.Context, subjectID string, subjectType VisitorType, objects []OrgType) (err error)

	// CheckPerms 账户组织架构管理权限检查
	CheckPerms(ctx context.Context, subjectID string, orgTyp OrgType, checkPerm OrgPermValue) (result bool, err error)
}

// ReservedNameInfo 保留名称信息
type ReservedNameInfo struct {
	ID         string
	Name       string
	CreateTime int64
	UpdateTime int64
}

// LogicsReservedName 保留名称接口
type LogicsReservedName interface {
	// UpdateReservedName 更新保留名称
	UpdateReservedName(name ReservedNameInfo) error

	// DeleteReservedName 删除保留名称
	DeleteReservedName(id string) error

	// GetReservedName 获取保留名称
	GetReservedName(name string) (info ReservedNameInfo, err error)
}

// LogicsActiveUser 活跃用户相关接口
type LogicsActiveUser interface {
	// GetActiveUserInfo 获取活跃用户信息
	GetActiveUserInfo(ctx context.Context, visitor *Visitor, bYear bool, year, month int) (out []byte, err error)
}
