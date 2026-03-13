// Package interfaces AnyShare 接口
package interfaces

import "context"

//go:generate mockgen -package mock -source ../interfaces/drivenadapters.go -destination ../interfaces/mock/mock_drivenadapters.go

// Operation 操作类型
type Operation int

const (
	_ Operation = iota

	// OpAddGroup 用户组创建
	OpAddGroup

	// OpDeleteGroup 用户组删除
	OpDeleteGroup

	// OpModifyGroup 用户组修改
	OpModifyGroup

	// OpDeleteGroupMembers 用户组成员删除
	OpDeleteGroupMembers

	// OpAddGroupMembers 用户组成员添加
	OpAddGroupMembers

	// OpAppRegister 应用账户注册
	OpAppRegister

	// OpDeleteApp 删除应用账户
	OpDeleteApp

	// OpUpdateApp 更新应用账户
	OpUpdateApp

	// OpAppTokenGenerated 生成应用账户令牌
	OpAppTokenGenerated
)

// DrivenEacpLog 日志处理接口
type DrivenEacpLog interface {
	// 记录日志
	EacpLog(visitor *Visitor, op Operation, logInfo interface{}) error

	// OpAddOrgPermAppLog 增加应用账户组织架构管理权限
	OpAddOrgPermAppLog(visitor *Visitor, perm *AppOrgPerm) error

	// OpDeleteOrgPermAppLog 删除应用账户组织架构管理权限
	OpDeleteOrgPermAppLog(visitor *Visitor, perm *AppOrgPerm) error

	// OpUpdateOrgPermAppLog 更新应用账户组织架构管理权限
	OpUpdateOrgPermAppLog(visitor *Visitor, perm *AppOrgPerm) error

	// OpSetDefaultPWDLog 更新用户初始密码
	OpSetDefaultPWDLog(visitor *Visitor) error

	// OpSetCSFLevelEnumLog 更新密级枚举
	OpSetCSFLevelEnumLog(visitor *Visitor, csfLevelEnum []string) error

	// OpSetCSFLevel2EnumLog 更新密级2枚举
	OpSetCSFLevel2EnumLog(visitor *Visitor, csfLevel2Enum []string) error

	// OpDeleteDepart 删除部门
	OpDeleteDepart(visitor *Visitor, departName string, isRoot bool) error

	// OpUserExpiredDisabled 用户状态变更
	OpUserExpiredDisabled(displayName, loginName string) error

	// OpUserNotLoginDisabled 用户长时间未登录自动禁用
	OpUserNotLoginDisabled(displayName, loginName string) error
}

// DrivenHydra 授权服务接口
type DrivenHydra interface {
	// Register 注册客户端
	Register(name, password string, lifespan int) (id string, err error)

	// Delete 删除客户端
	Delete(id string) (err error)

	// Update 更新客户端
	Update(id, name, password string) (err error)

	// DeleteConsentAndLogin 删除认证与授权会话
	DeleteConsentAndLogin(clientID, userID string) (err error)

	// GenerateToken 生成令牌
	GenerateToken(clientID, clientSecret string) (token string, err error)

	// DeleteClientToken 删除客户端令牌
	DeleteClientToken(clientID string) (err error)
}

// NameChangeMsg 名称变更消息结构体
type NameChangeMsg struct {
	ID      string
	NewName string
	OType   string
}

// MsgType 消息类型
type MsgType int

const (
	_ MsgType = iota

	// DeleteGroup 删除用户组消息类型
	DeleteGroup

	// OrgNameChange 组织成员名称变更消息类型
	OrgNameChange

	// AppDeleted 删除应用账户消息类型
	AppDeleted

	// AppNameChanged 应用账户名变更消息类型
	AppNameChanged

	// UserStatusChanged 用户状态变更消息类型
	UserStatusChanged
)

// DrivenMessageBroker 消息发送对象
type DrivenMessageBroker interface {
	// Publish 消息发送
	Publish(msgType MsgType, msg interface{}) error

	// AnonymityAuth 匿名认证消息发送
	AnonymityAuth(msgType string, msg interface{}) error

	// ContactorDeleted 联系人组被删除
	ContactorDeleted(ids []string) (err error)

	// InternalGroupDeleted 内部组被删除
	InternalGroupDeleted(ids []string) (err error)

	// DepartDeleted 部门被删除
	DepartDeleted(id string) (err error)

	// OrgManagerChanged 更新配额
	OrgManagerChanged(ids []string) (err error)

	// UserStatusChanged 用户状态变更
	UserStatusChanged(id string, status bool) (err error)
}

// OSSInfo 对象存储信息
type OSSInfo struct {
	ID       string // 对象存储ID
	BDefault bool   // 是否为默认存储
}

// DnOSSGateWay OSS网关对象
type DnOSSGateWay interface {
	// UploadFile 上传文件
	UploadFile(ctx context.Context, visitor *Visitor, ossID, key string, data []byte) (err error)

	// DeleteFile 删除文件
	DeleteFile(ctx context.Context, visitor *Visitor, ossID, key string) (err error)

	// GetDownloadURL 获取下载文件URL
	GetDownloadURL(ctx context.Context, visitor *Visitor, ossID, key string) (url string, err error)

	// GetLocalEnabledOSSInfo 获取本地站点可用存储信息
	GetLocalEnabledOSSInfo(ctx context.Context, visitor *Visitor) (out []OSSInfo, err error)
}
