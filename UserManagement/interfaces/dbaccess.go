// Package interfaces AnyShare 接口
package interfaces

import (
	"context"
	"database/sql"
)

//go:generate mockgen -package mock -source ../interfaces/dbaccess.go -destination ../interfaces/mock/mock_dbaccess.go

// NameInfo 名称信息
type NameInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ContactorInfo 联系人组信息
type ContactorInfo struct {
	UserID      string
	ContactorID string
}

// DBContactor contactor 数据访问层处理接口
type DBContactor interface {
	// GetContactorName 批量获取联系人组名
	GetContactorName(groupIDs []string) ([]NameInfo, []string, error)

	// GetUserAllBelongContactorIDs 获取包含用户的所有联系人组
	GetUserAllBelongContactorIDs(userID string) (contactorIDs []string, err error)

	// GetContactorInfo  获取联系人组信息
	GetContactorInfo(contactorID string) (result bool, info ContactorInfo, err error)

	// DeleteContactors 删除联系人组
	DeleteContactors(contactorIDs []string, tx *sql.Tx) (err error)

	// DeleteContactorMembers 删除联系人组中的成员
	DeleteContactorMembers(contactorIDs []string, tx *sql.Tx) (err error)

	// GetAllContactorInfos 获取用户所有的联系人组信息
	GetAllContactorInfos(userID string) (infos []ContactorInfo, err error)

	// DeleteUserInContactors 在联系人组内删除用户
	DeleteUserInContactors(userID string, tx *sql.Tx) (err error)

	// UpdateContactorCount 更新联系人组的联系人数量信息
	UpdateContactorCount() (err error)

	// GetContactorMemberIDs 批量获取联系人组成员ID
	GetContactorMemberIDs(ctx context.Context, contactorIDs []string) (infos map[string][]string, err error)
}

// DepartmentDBInfo 数据库信息
type DepartmentDBInfo struct {
	ID        string
	Name      string
	IsRoot    int
	Email     string
	Path      string
	ManagerID string
	Code      string
	Status    bool
	Remark    string
	ThirdID   string
}

// AuthType 用户认证类型
type AuthType int

const (
	_ AuthType = iota

	// Local 本地认证
	Local

	// Domain 域认证
	Domain

	// Third 第三方认证
	Third
)

// DisableType 用户禁用状态类型
type DisableType int

const (
	_ DisableType = iota

	// Enabled 未禁用
	Enabled

	// Disabled 禁用
	Disabled

	// Deleted 用户被第三方删除
	Deleted
)

// AutoDisableType 自动禁用状态类型
type AutoDisableType int

const (
	_ AutoDisableType = iota

	// AEnabled 未自动禁用
	AEnabled

	// ADisabled 长时间未登录自动禁用
	ADisabled

	// ExpireDisabled 用户过期自动禁用
	ExpireDisabled
)

// LDAPServerType LDAP Server类型
type LDAPServerType int

const (
	_ LDAPServerType = iota

	// WindowAD Window AD
	WindowAD

	// OtherLDAP Other LDAP
	OtherLDAP
)

// UserDBInfo 用户数据库信息
type UserDBInfo struct {
	ID                 string
	Name               string
	Account            string
	CSFLevel           int
	CSFLevel2          int
	Priority           int
	DisableStatus      DisableType     // 禁用状态
	AutoDisableStatus  AutoDisableType // 自动禁用状态
	Email              string
	AuthType           AuthType // 用户认证类型
	Password           string
	DesPassword        string
	NtlmPassword       string
	Sha2Password       string
	Frozen             bool           // 冻结状态
	Authenticated      bool           // 实名认证状态
	TelNumber          string         // 电话号码
	ThirdAttr          string         // 第三方插件应用属性，目前只有第三方消息插件用到
	ThirdID            string         // 第三方系统ID
	PWDControl         bool           // 用户密码管控状态
	PWDTimeStamp       int64          // 上次密码修改时间
	PWDErrCnt          int            // 密码错误次数
	PWDErrLatestTime   int64          // 上次密码错误的时间
	LDAPType           LDAPServerType // ldap服务器类型
	DomainPath         string         // 域路径
	OssID              string         // 存储id
	ManagerID          string         // 管理员id
	CreatedAtTimeStamp int64
	Remark             string
	Code               string
	Position           string
}

// DepartmentManagerInfo 部门组织管理员信息
type DepartmentManagerInfo struct {
	DepartmentID string     `json:"department_id"`
	Managers     []NameInfo `json:"managers"`
}

// DBDepartment department 数据访问层处理接口
type DBDepartment interface {
	// GetDepartmentName 批量获取部门名
	GetDepartmentName(deptIDs []string) ([]NameInfo, []string, error)

	// GetDepartmentInfo 批量获取部门信息
	GetDepartmentInfo(deptIDs []string, bLimit bool, offset, limit int) ([]DepartmentDBInfo, error)

	// GetDepartmentInfo2 批量获取部门信息v2版本，添加trace
	GetDepartmentInfo2(ctx context.Context, deptIDs []string, bLimit bool, offset, limit int) ([]DepartmentDBInfo, error)

	// GetDepartmentInfoByIDs 根据ID批量获取部门信息
	GetDepartmentInfoByIDs(ctx context.Context, deptIDs []string) (out []DepartmentDBInfo, err error)

	// GetParentDepartmentID 批量获取父部门id
	GetParentDepartmentID(deptIDs []string) ([]string, error)

	// GetChildDepartmentIDs 批量获取子部门id
	GetChildDepartmentIDs(deptIDs []string) ([]string, map[string][]string, error)

	// GetChildDepartmentIDs2 批量获取子部门idv2版本，添加trace
	GetChildDepartmentIDs2(ctx context.Context, deptIDs []string) ([]string, map[string][]string, error)

	// GetChildUserIDs 获取部门子用户id
	GetChildUserIDs(deptIDs []string) ([]string, map[string][]string, error)

	// GetRootDeps 获取根部门信息
	GetRootDeps(bCount, bNoScope bool, scope []string, offset, limit int) ([]DepartmentDBInfo, error)

	// GetSubDepartmentInfos 获取部门子部门信息(需排序)
	GetSubDepartmentInfos(deptID string, bCount bool, offset, limit int) (out []DepartmentDBInfo, err error)

	// GetSubUserInfos 获取部门子用户信息(需排序)
	GetSubUserInfos(deptID string, bCount bool, offset, limit int) (out []UserDBInfo, err error)

	// SearchDepartsByKey 搜索部门
	SearchDepartsByKey(ctx context.Context, bCount, bNoScope bool, scope []string, keyword string, offset, limit int) (out []DepartmentDBInfo, err error)

	// GetManagersOfDepartment 批量获取部门组织管理员信息
	GetManagersOfDepartment(departmentIDs []string) (infoList []DepartmentManagerInfo, err error)

	// GetDepartmentByPathLength 根据path长度获取部门信息
	GetDepartmentByPathLength(nLen int) (infoList []DepartmentDBInfo, err error)

	// GetAllSubUserIDsByDepartPath 根据path获取部门所有子成员ID
	GetAllSubUserIDsByDepartPath(path string) (ids []string, err error)

	// GetAllSubUserInfosByDepartPath 根据path获取部门所有子成员基本信息
	GetAllSubUserInfosByDepartPath(path string) (infos []UserBaseInfo, err error)

	// DeleteOrgManagerRelationByDepartID 根据部门ID删除组织管理员管辖信息
	DeleteOrgManagerRelationByDepartID(id string) (err error)

	// DeleteOrgAuditRelationByDepartID 根据部门ID删除组织审计员管辖信息
	DeleteOrgAuditRelationByDepartID(id string) (err error)

	// DeleteUserDepartRelationByPath 根据部门路径删除部门下用户的用户/部门关系
	DeleteUserDepartRelationByPath(path string, tx *sql.Tx) (err error)

	// AddUserToDepart 添加用户到部门
	AddUserToDepart(userIDs []string, departID string, departPath string, tx *sql.Tx) (err error)

	// DeleteUserOURelation 删除用户/组织关系
	DeleteUserOURelation(userIDs []string, orgID string, tx *sql.Tx) (err error)

	// DeleteDepartByPath 根据路径删除部门下所有部门信息
	DeleteDepartByPath(path string, tx *sql.Tx) (err error)

	// DeleteDepartRelations 删除部门的部门关系
	DeleteDepartRelations(departIDs []string, tx *sql.Tx) (err error)

	// DeleteDepartRelations 删除部门的组织关系
	DeleteDepartOURelations(departIDs []string, tx *sql.Tx) (err error)

	// GetAllSubDepartInfosByPath 根据路径获取部门下所有子部门信息，包含自己
	GetAllSubDepartInfosByPath(path string) (ids []DepartmentDBInfo, err error)

	// GetAllOrgManagerIDsByDepartIDs 根据部门ID获取所有的组织管理员
	GetAllOrgManagerIDsByDepartIDs(departIds []string) (orgManagerIDs []string, err error)

	// GetAllOrgManagerIDs 获取所有的组织管理员ID
	GetAllOrgManagerIDs() (ids []string, err error)

	// DeleteDocAutoCleanStrategy 删除文档自动清理策略
	DeleteDocAutoCleanStrategy(obj string) (err error)

	// DeleteDepartManager 清理部门负责人数据
	DeleteDepartManager(userID string) (err error)

	// SearchDeparts 内置管理员按照关键字搜索部门
	SearchDeparts(ctx context.Context, ks *DepartSearchKeyScope, k *DepartSearchKey, limitDepartIDs []string) (out []DepartmentDBInfo, err error)

	// SearchDepartsCount 内置管理员按照关键字搜索部门
	SearchDepartsCount(ctx context.Context, ks *DepartSearchKeyScope, k *DepartSearchKey, limitDepartIDs []string) (count int, err error)
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

// Direction 排序方向
type Direction int

const (
	_ Direction = iota

	// Desc 降序
	Desc

	// Asc 升序
	Asc
)

// SearchInfo 列举信息
type SearchInfo struct {
	Direction           Direction
	Sort                SortFiled
	Offset              int
	Limit               int
	Keyword             string
	HasKeyWord          bool
	NotShowDisabledUser bool
}

// GroupInfo 用户组信息
type GroupInfo struct {
	ID    string
	Name  string
	Notes string
}

// DBGroup group 数据访问层处理接口
type DBGroup interface {
	// GetGroupIDByName 根据用户名获取用户ID
	GetGroupIDByName(name string) (id string, err error)

	// GetGroupIDByName2 根据用户名获取用户ID,支持trace
	GetGroupIDByName2(ctx context.Context, name string) (id string, err error)

	// AddGroup 添加用户组
	AddGroup(ctx context.Context, id, name, notes string, tx *sql.Tx) (err error)

	// DeleteGroup 删除用户组
	DeleteGroup(id string) (err error)

	// ModifyGroup 修改用户组
	ModifyGroup(id, name string, nameChanged bool, notes string, notesChanged bool) (err error)

	// GetGroups 列举符合条件的用户组
	GetGroups(info SearchInfo) (groups []GroupInfo, err error)

	// GetGroupsNum 列举符合条件的用户组数量
	GetGroupsNum(info SearchInfo) (num int, err error)

	// GetGroupByID 获取指定的用户组
	GetGroupByID(id string) (info GroupInfo, err error)

	// GetGroupByID2 获取指定的用户组， 支持trace
	GetGroupByID2(ctx context.Context, id string) (info GroupInfo, err error)

	// GetExistGroupIDs 获取存在的用户组id
	GetExistGroupIDs(groupIDs []string) (existGroupIDs []string, err error)

	// SearchGroupByKeyword 用户组关键字搜索
	SearchGroupByKeyword(keyword string, start, limit int) (out []NameInfo, err error)

	// SearchGroupNumByKeyword 用户组关键字搜索符合条件的用户组数量
	SearchGroupNumByKeyword(keyword string) (num int, err error)

	// GetGroupName 根据用户组ID获取用户组名
	GetGroupName(ids []string) (nameInfo []NameInfo, exsitIDs []string, err error)

	// GetGroupName2 根据用户组ID获取用户组名,支持trace
	GetGroupName2(ctx context.Context, ids []string) (nameInfo []NameInfo, exsitIDs []string, err error)
}

// GroupMemberInfo 组成员数据库信息
type GroupMemberInfo struct {
	ID              string
	MemberType      int
	Name            string
	DepartmentNames []string
	ParentDeps      [][]NameInfo
}

// MemberInfo 用户组成员信息
type MemberInfo struct {
	ID         string
	Name       string
	NType      int
	GroupNames []string
}

// MemberSimpleInfo 用户组成员信息
type MemberSimpleInfo struct {
	ID    string
	Name  string
	NType int
}

// GroupMemberIDs 组成员ID数据
type GroupMemberIDs struct {
	UserIDs       []string
	DepartmentIDs []string
}

// DBGroupMember group member 数据访问层处理接口
type DBGroupMember interface {
	// DeleteGroupMemberByID 按照用户组id删除用户组成员
	DeleteGroupMemberByID(id string) (err error)

	// AddGroupMember 添加用户组成员
	AddGroupMember(id string, info *GroupMemberInfo) (err error)

	// AddGroupMembers 批量添加用户组成员
	AddGroupMembers(ctx context.Context, id string, infos []GroupMemberInfo, tx *sql.Tx) (err error)

	// DeleteGroupMember 删除用户组成员
	DeleteGroupMember(id string, info *GroupMemberInfo) (err error)

	// DeleteGroupMemberByMemberID 根据成员ID删除用户组成员关系
	DeleteGroupMemberByMemberID(id string) (err error)

	// GetGroupMembers 批量获取用户组成员的id和类型
	GetGroupMembersByGroupIDs(groupIDs []string) (outInfo []GroupMemberInfo, err error)

	// GetGroupMembersByGroupIDs2 批量获取用户组成员的id和类型，支持trace
	GetGroupMembersByGroupIDs2(ctx context.Context, groupIDs []string) (outInfo map[string][]GroupMemberInfo, err error)

	// GetGroupMembers 列举用户组成员（分页）
	GetGroupMembers(ctx context.Context, id string, info SearchInfo) (outInfo []GroupMemberInfo, err error)

	// GetGroupMembersNum 列举用户组成员数量
	GetGroupMembersNum(id string, info SearchInfo) (num int, err error)

	// GetGroupMembersNum 2列举用户组成员数量,支持trace
	GetGroupMembersNum2(ctx context.Context, id string, info SearchInfo) (num int, err error)

	// CheckGroupMembersExist 用户组成员是否存在
	CheckGroupMembersExist(id string, info *GroupMemberInfo) (ret bool, err error)

	// GetMembersBelongGroupIDs 获取包含成员的用户组ID
	GetMembersBelongGroupIDs(memberIDs []string) (groupIDs []string, groups []GroupInfo, err error)

	// SearchMembersByKeyword 用户组成员关键字搜索
	SearchMembersByKeyword(keyword string, start, limit int) (out []MemberInfo, err error)

	// SearchMemberNumByKeyword 用户组成员关键字搜索符合条件的用户组总数目
	SearchMemberNumByKeyword(keyword string) (num int, err error)

	// GetMemberOnClient 客户端列举组成员
	GetMemberOnClient(id string, offset, limit int) (info []MemberSimpleInfo, err error)
}

// DisableStatus 用户禁用状态类型
type DisableStatus int

const (
	// NotLoginDisabled 长时间未登录自动禁用
	NotLoginDisabled DisableStatus = 0x00000001
	// ExpiredDisabled 用户过期自动禁用
	ExpiredDisabled DisableStatus = 0x00000002
)

type DistributedLockType int

const (
	_ DistributedLockType = iota

	// UserExpiredDisableLock 用户过期自动禁用锁
	UserExpiredDisableLock

	// UserNotLoginDisableLock 用户长时间未登录自动禁用锁
	UserNotLoginDisableLock
)

// DBUser user 数据访问层处理接口
type DBUser interface {
	// GetUserName 批量获取用户显示名
	GetUserName(userIDs []string) ([]UserDBInfo, []string, error)

	// GetDirectBelongDepartmentIDs 获取用户直属部门id
	GetDirectBelongDepartmentIDs(userID string) ([]string, []DepartmentDBInfo, error)

	// GetDirectBelongDepartmentIDs2 获取用户直属部门idv2 添加trace
	GetDirectBelongDepartmentIDs2(ctx context.Context, userID string) ([]string, []DepartmentDBInfo, error)

	// GetUsersInDepartments 获取在部门内部的用户ID
	GetUsersInDepartments(userIDs, departmentIDs []string) ([]string, error)

	// GetUserDBInfo 获取用户基本数据库信息
	GetUserDBInfo(userID []string) (out []UserDBInfo, err error)

	// GetUserDBInfo2 获取用户基本数据库信息，增加trace 其他一样
	GetUserDBInfo2(ctx context.Context, userID []string) (out []UserDBInfo, err error)

	// GetUserDBInfoByTels 根据手机号获取用户基本数据库信息
	GetUserDBInfoByTels(ctx context.Context, tels []string) (out []UserDBInfo, err error)

	// GetOrgAduitDepartInfo 获取组织审计员审计范围内部门
	GetOrgAduitDepartInfo(userID string) (out []string, err error)

	// GetOrgAduitDepartInfo2 获取组织审计员审计范围内部门v2版本，增加trace 其他一样
	GetOrgAduitDepartInfo2(ctx context.Context, userID string) (out []string, err error)

	// GetOrgManagerDepartInfo 获取组织管理员管理范围部门
	GetOrgManagerDepartInfo(userID string) (out []string, err error)

	// GetOrgManagerDepartInfo2 获取组织管理员管理范围部门v2版本，增加trace 其他一样
	GetOrgManagerDepartInfo2(ctx context.Context, userID string) (out []string, err error)

	// GetOrgManagersDepartInfo 获取组织管理员管理范围部门
	GetOrgManagersDepartInfo(userIDs []string) (out map[string][]string, err error)

	// SearchOrgUsersByKey 在组织架构内的用户中进行搜索
	SearchOrgUsersByKey(ctx context.Context, bScope, bCount bool, keyword string, offset, limit int, onlyEnableUser, onlyAssignedUser bool, scope []string) (out []UserDBInfo, err error)

	// CheckNameExist 检查名字存在
	CheckNameExist(name string) (exist bool, err error)

	// ModifyUserInfo 修改用户信息
	ModifyUserInfo(bRange UserUpdateRange, info *UserDBInfo, tx *sql.Tx) (err error)

	// GetUsersPath 获取用户所属路径
	GetUsersPath(ids []string) (paths map[string][]string, err error)

	// GetUsersPath2 获取用户所属路径 支持trace
	GetUsersPath2(ctx context.Context, ids []string) (paths map[string][]string, err error)

	// GetUserInfoByAccount 根据登录名获取用户信息
	GetUserInfoByAccount(account string) (info UserDBInfo, err error)

	// GetUserInfoByName 根据显示名获取用户信息
	GetUserInfoByName(ctx context.Context, name string) (info UserDBInfo, err error)

	// SearchUserInfoByName 根据显示名搜索用户信息
	SearchUserInfoByName(ctx context.Context, name string) (infos []UserDBInfo, err error)

	// GetDomainUserInfoByAccount 根据登录名获取域用户信息
	GetDomainUserInfoByAccount(account string) (info UserDBInfo, err error)

	// GetUserInfoByIDCard 根据身份号获取用户信息
	GetUserInfoByIDCard(id string) (info UserDBInfo, err error)

	// UpdatePwdErrInfo 更新账户密码错误信息
	UpdatePwdErrInfo(id string, pwdErrCnt int, pwdErrLastTime int64) (err error)

	// GetUserCustomAttr 获取用户自定义属性
	GetUserCustomAttr(id string) (customAttr map[string]interface{}, err error)

	// UpdateUserCustomAttr 更新用户自定义属性
	UpdateUserCustomAttr(ctx context.Context, id string, customAttr map[string]interface{}, tx *sql.Tx) (err error)

	// AddUserCustomAttr 添加用户自定义属性
	AddUserCustomAttr(ctx context.Context, id string, customAttr map[string]interface{}, tx *sql.Tx) (err error)

	// DeleteUserManagerID 删除用户上级信息
	DeleteUserManagerID(id string) (err error)

	// 用户搜索
	SearchUsers(ctx context.Context, ks *UserSearchInDepartKeyScope, k *UserSearchInDepartKey) (out []UserDBInfo, err error)

	// SearchUsersCount 搜索用户数量
	SearchUsersCount(ctx context.Context, ks *UserSearchInDepartKeyScope, k *UserSearchInDepartKey) (num int, err error)

	// GetUserList 获取用户列表
	GetUserList(ctx context.Context, direction Direction, bHasMarker bool, createdStamp int64, userID string, limit int) (out []UserDBInfo, err error)

	// GetAllUserCount 获取所有用户数量
	GetAllUserCount(ctx context.Context) (num int, err error)

	// GetExpiredNeedDisableUserInfos 获取过期需要禁用的用户信息
	GetExpiredNeedDisableUserInfos(ctx context.Context, tx *sql.Tx) (userInfos []UserDBInfo, err error)

	// GetLock 获取锁
	GetLock(ctx context.Context, lockType DistributedLockType, tx *sql.Tx) (err error)

	// SetAutoDisableStatus 设置用户自动禁用状态
	SetAutoDisableStatus(ctx context.Context, userIDs []string, disableStatus DisableStatus, tx *sql.Tx) error

	// GetNotLoginNeedDisableUserInfos 获取长时间未登录要禁用的用户信息
	GetNotLoginNeedDisableUserInfos(ctx context.Context, allowDays int64, tx *sql.Tx) (userInfos []UserDBInfo, err error)
}

// DBRole 角色信息
type DBRole interface {
	// GetRolesByUserIDs 根据用户ID数组批量获取用户角色
	GetRolesByUserIDs(userIDs []string) (map[string]map[Role]bool, error)

	// GetRolesByUserIDs2 根据用户ID数组批量获取用户角色，支持trace
	GetRolesByUserIDs2(ctx context.Context, userIDs []string) (map[string]map[Role]bool, error)

	// GetUserIDsByRoleIDs 根据角色ID数组批量获取角色成员
	GetUserIDsByRoleIDs(ctx context.Context, roles []Role) (map[Role][]string, error)
}

// DBAnonymous 匿名账户
type DBAnonymous interface {
	// Create 创建匿名账户
	Create(info *AnonymousInfo) error

	// GetAccount 获取匿名账户
	GetAccount(ID string) (info AnonymousInfo, err error)

	// AddAccessTimes 访问计数+1
	AddAccessTimes(ID string, tx *sql.Tx) error

	// DeleteByID 删除匿名账户
	DeleteByID(ID string) error

	// DeleteByTime 删除过期匿名账户
	DeleteByTime(curTime int64) error
}

// DBOutbox 发件箱数据库接口
type DBOutbox interface {
	// 批量添加outbox消息
	AddOutboxInfos(businessType int, messages []string, tx *sql.Tx) error
	// 获取推送消息
	GetPushMessage(businessType int, tx *sql.Tx) (messageID int64, message string, err error)
	// 根据ID删除outbox消息
	DeleteOutboxInfoByID(messageID int64, tx *sql.Tx) error
}

// CredentialType 凭证类型
type CredentialType int

const (
	// 使用iota，增加新的枚举时，必须按照顺序添加，不得在中间插入
	_ CredentialType = iota

	// CredentialTypePassword 密码
	CredentialTypePassword

	// CredentialTypeToken 令牌
	CredentialTypeToken
)

// AppInfo 应用账户信息
type AppInfo struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	CredentialType CredentialType `json:"-"`
}

// AppCompleteInfo 应用账户完整信息
type AppCompleteInfo struct {
	AppInfo
	Password string
	Type     AppType
}

// DBApp 应用账户管理接口
type DBApp interface {
	// RegisterApp 注册应用账户
	RegisterApp(appInfo *AppCompleteInfo, tx *sql.Tx) (err error)

	// DeleteApp 删除应用账户
	DeleteApp(clientID string, tx *sql.Tx) (err error)

	// UpdateApp 更新应用账户
	UpdateApp(id string, bName bool, name string, bPwd bool, pwd string, tx *sql.Tx) (err error)

	// AppList 获取应用账户列表
	AppList(searchInfo *SearchInfo) (info *[]AppInfo, err error)

	// AppListCount 获取应用账户总数
	AppListCount(searchInfo *SearchInfo) (num int, err error)

	// GetAppByName 获取应用账户信息
	GetAppByName(name string) (appinfo *AppInfo, err error)

	// GetAppByID 获取应用账户信息
	GetAppByID(id string) (appinfo *AppInfo, err error)

	// GetAppName 获取应用账户名
	GetAppName(ids []string) (nameInfo []NameInfo, exsitIDs []string, err error)
}

// ConfigKey 配置参数枚举
type ConfigKey int

const (
	_ ConfigKey = iota

	// UserDefaultNTLMPWD 用户默认ntlm密码
	UserDefaultNTLMPWD

	// UserDefaultDESPWD 用户默认DES密码
	UserDefaultDESPWD

	// UserDefaultSha2PWD 用户默认sha2密码
	UserDefaultSha2PWD

	// UserDefaultPWD 用户默认明文密码
	UserDefaultPWD

	// UserDefaultMd5PWD 用户默认md5密码
	UserDefaultMd5PWD

	// IDCardLogin 身份证登录
	IDCardLogin

	// EmailPwdRetrieval 通过邮箱找回密码功能开启状态
	EmailPwdRetrieval

	// TelPwdRetrieval 通过手机号找回密码功能开启状态
	TelPwdRetrieval

	// PWDExpireTime 密码过期时间
	PWDExpireTime

	// StrongPWDStatus 强密码标识
	StrongPWDStatus

	// StrongPWDLength 强密码长度
	StrongPWDLength

	// EnablePWDLock 密码错误锁定
	EnablePWDLock

	// PWDErrCnt 密码错误锁定次数
	PWDErrCnt

	// PWDLockTime 密码锁定时间
	PWDLockTime

	// EnableDesPassWord 是否记录des密码
	EnableDesPassWord

	// EnableThirdPwdLock 是否允许第三方密码锁定
	EnableThirdPwdLock

	// strCSFLevelEnum 密级枚举
	CSFLevelEnum

	// strCSFLevel2Enum 密级2枚举
	CSFLevel2Enum

	// ShowCSFLevel2 是否显示密级2
	ShowCSFLevel2

	// AutoDisable 自动禁用功能是否开启
	AutoDisable
)

// Config 配置信息
type Config struct {
	UserDefaultNTLMPWD string // 用户默认ntlm密码
	UserDefaultDESPWD  string // 用户默认DES密码
	UserDefaultSha2PWD string // 用户默认sha2密码
	UserDefaultMd5PWD  string // 用户默认md5密码
	UserDefaultPWD     string // 用户默认明文密码
	IDCardLogin        bool   // 身份证登录是否开启
	EmailPwdRetrieval  bool   // 通过邮箱找回密码是否开启
	TelPwdRetrieval    bool   // 通过手机号找回密码是否开启
	PwdExpireTime      int64  // 密码过期时间，单位为 天
	StrongPwdStatus    bool
	StrongPwdLength    int
	EnablePwdLock      bool
	PwdErrCnt          int
	PwdLockTime        int64 // 密码锁定时间，单位为 分钟
	EnableDesPwd       bool
	EnableThirdPwdLock bool
	CSFLevelEnum       map[string]int
	CSFLevel2Enum      map[string]int
	ShowCSFLevel2      bool
	AutoDisableEnabled bool
	AutoDisableTime    int64
}

// DBConfig 配置信息接口
type DBConfig interface {
	// SetConfig 设置认证配置
	SetConfig(configKeys map[ConfigKey]bool, cfg *Config, tx *sql.Tx) (err error)

	// SetShareMgntConfig 设置认证配置 sharemgnt_db 数据库操作
	SetShareMgntConfig(key ConfigKey, cfg *Config, tx *sql.Tx) (err error)

	// GetConfig 获取配置信息
	GetConfig(configKeys map[ConfigKey]bool) (cfg Config, err error)

	// GetConfigFromOption 获取配置信息
	GetConfigFromOption(configKeys map[ConfigKey]bool) (cfg Config, err error)
}

// OrgType 组织架构对象类型
type OrgType int

// 组织架构对象值定义
const (
	_ OrgType = iota

	// User 用户
	User

	// Department 部门
	Department

	// Group 用户组
	Group
)

// AppOrgPermValue 应用账户组织架构管理权限类型
type AppOrgPermValue int32

const (
	// 使用iota，增加新的枚举时，必须按照顺序添加，不得在中间插入
	// 否则会改变枚举的值，造成应用账户组织管理权限与db中记录的type对应不上

	// Modify 修改
	Modify AppOrgPermValue = 1 << iota

	// Read 读取
	Read
)

// OrgPermValue 账户组织架构管理权限类型
type OrgPermValue int32

const (
	// 使用iota，增加新的枚举时，必须按照顺序添加，不得在中间插入
	// 否则会改变枚举的值，造成账户组织管理权限与db中记录的type对应不上

	// OPRead 读取
	OPRead OrgPermValue = 1 << iota
)

// DBOrgPermApp 应用账户对组织架构权限信息
type DBOrgPermApp interface {
	// UpdateAppName 更新应用账户组织架构权限表 名称
	UpdateAppName(info *AppInfo) (err error)

	// GetAppPermByID 根据应用账户ID获取应用账户对组织架构的权限信息
	GetAppPermByID(id string) (out map[OrgType]AppOrgPerm, err error)

	// GetAppPermByID2 根据应用账户ID获取应用账户对组织架构的权限信息，支持trace
	GetAppPermByID2(ctx context.Context, id string) (out map[OrgType]AppOrgPerm, err error)

	// UpdateAppOrgPerm 更新应用账户权限信息
	UpdateAppOrgPerm(info AppOrgPerm, tx *sql.Tx) (err error)

	// AddAppOrgPerm 增加应用账户权限信息
	AddAppOrgPerm(info AppOrgPerm, tx *sql.Tx) (err error)

	// DeleteAppOrgPerm 删除应用账户权限信息
	DeleteAppOrgPerm(id string, types []OrgType) (err error)
}

// AvatarOSSInfo 头像文件在OSS存储信息
type AvatarOSSInfo struct {
	UserID  string
	OSSID   string
	Key     string
	Type    string
	BUseful bool
	Time    int64
}

// DBAvatar 头像信息接口
type DBAvatar interface {
	// Get 获取用户当前头像信息
	Get(userID string) (info AvatarOSSInfo, err error)

	// Add 新增头像信息
	Add(info *AvatarOSSInfo) (err error)

	// UpdateStatusByKey 根据key更新头像状态
	UpdateStatusByKey(key string, status bool, tx *sql.Tx) (err error)

	// SetAvatarUnableByID 根据用户ID禁用用户当前头像
	SetAvatarUnableByID(userID string, tx *sql.Tx) (err error)

	// GetUselessAvatar 获取超时未用到的头像信息
	GetUselessAvatar(time int64) (data []AvatarOSSInfo, err error)

	// Delete 删除用户头像信息
	Delete(key string) (err error)
}

// InternelGroup  内部组信息
type InternelGroup struct {
	ID string
}

// DBInternalGroup 内部组接口
type DBInternalGroup interface {
	// Add 新增内部组
	Add(id string) (err error)

	// Delete 删除内部组
	Delete(ids []string, tx *sql.Tx) (err error)

	// Get 获取内部组信息
	Get(ids []string) (infos map[string]InternelGroup, err error)
}

// InternalGroupMember 内部组成员信息
type InternalGroupMember struct {
	ID   string
	Type OrgType
}

// DBInternalGroupMember 内部组成员接口
type DBInternalGroupMember interface {
	// Add 增加内部组成员
	Add(groupID string, infos []InternalGroupMember, tx *sql.Tx) (err error)

	// DeleteAll 删除内部组内所有成员
	DeleteAll(ids []string, tx *sql.Tx) (err error)

	// Get 获取内部组成员
	Get(groupID string) (outInfo []InternalGroupMember, err error)

	// GetBelongGroups 获取用户成员所属内部组
	GetBelongGroups(info InternalGroupMember) (ids []string, err error)
}

// DBOrgPerm 账户对组织架构权限信息
type DBOrgPerm interface {
	// UpdateName 更新账户组织架构权限表 名称
	UpdateName(id string, newName string) (err error)

	// GetPermByID 根据账户ID获取账户对组织架构的权限信息，支持trace
	GetPermByID(ctx context.Context, id string) (out map[OrgType]OrgPerm, err error)

	// UpdateOrgPerm 更新应用账户权限信息
	UpdateOrgPerm(ctx context.Context, info OrgPerm, tx *sql.Tx) (err error)

	// AddOrgPerm 增加应用账户权限信息
	AddOrgPerm(ctx context.Context, info OrgPerm, tx *sql.Tx) (err error)

	// DeleteOrgPerm 删除应用账户权限信息
	DeleteOrgPerm(ctx context.Context, id string, types []OrgType) (err error)

	// DeleteOrgPermByID 删除账户权限信息
	DeleteOrgPermByID(id string) (err error)
}

// DBReservedName 保留名称数据库操作接口
type DBReservedName interface {
	// AddReservedName 新增保留名称
	AddReservedName(name ReservedNameInfo, tx *sql.Tx) error

	// UpdateReservedName 更新保留名称
	UpdateReservedName(name ReservedNameInfo, tx *sql.Tx) error

	// GetReservedNameByID 根据ID获取保留名称
	GetReservedNameByID(id string, tx *sql.Tx) (ReservedNameInfo, bool, error)

	// GetReservedNameByName 根据name获取保留名称
	GetReservedNameByName(id string, tx *sql.Tx) (ReservedNameInfo, bool, error)

	// DeleteReservedName 删除保留名称
	DeleteReservedName(id string) error

	// GetLock 获取锁
	GetLock(tx *sql.Tx) error
}
