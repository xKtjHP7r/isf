// Package logics AnyShare
package logics

import (
	"github.com/kweaver-ai/proton-rds-sdk-go/sqlx"

	"UserManagement/interfaces"
)

var dbUser interfaces.DBUser

// SetDBUser 设置实例
func SetDBUser(i interfaces.DBUser) {
	dbUser = i
}

var dbGroup interfaces.DBGroup

// SetDBGroup 设置实例
func SetDBGroup(i interfaces.DBGroup) {
	dbGroup = i
}

var dbGroupMember interfaces.DBGroupMember

// SetDBGroupMembers 设置实例
func SetDBGroupMembers(i interfaces.DBGroupMember) {
	dbGroupMember = i
}

var dbDepartment interfaces.DBDepartment

// SetDBDepartment 设置实例
func SetDBDepartment(i interfaces.DBDepartment) {
	dbDepartment = i
}

var dbContactor interfaces.DBContactor

// SetDBContactor 设置实例
func SetDBContactor(i interfaces.DBContactor) {
	dbContactor = i
}

var dbAnonymous interfaces.DBAnonymous

// SetDBAnonymous 设置实例
func SetDBAnonymous(i interfaces.DBAnonymous) {
	dbAnonymous = i
}

// dbOutbox 实例
var dbOutbox interfaces.DBOutbox

// SetDBOutbox 设置实例
func SetDBOutbox(i interfaces.DBOutbox) {
	dbOutbox = i
}

// dbPool 实例
var dbPool *sqlx.DB

// SetDBPool  设置实例
func SetDBPool(i *sqlx.DB) {
	dbPool = i
}

// dbApp 实例
var dbApp interfaces.DBApp

// SetDBApp 设置实例
func SetDBApp(i interfaces.DBApp) {
	dbApp = i
}

// dbConfig 实例
var dbConfig interfaces.DBConfig

// SetDBConfig 设置实例
func SetDBConfig(i interfaces.DBConfig) {
	dbConfig = i
}

// dbOrgPermApp 实例
var dbOrgPermApp interfaces.DBOrgPermApp

// SetDBOrgPermApp 设置实例
func SetDBOrgPermApp(i interfaces.DBOrgPermApp) {
	dbOrgPermApp = i
}

// dbRole 实例
var dbRole interfaces.DBRole

// SetDBRole 设置实例
func SetDBRole(i interfaces.DBRole) {
	dbRole = i
}

// dbAvatar 实例
var dbAvatar interfaces.DBAvatar

// SetDBAvatar 设置实例
func SetDBAvatar(i interfaces.DBAvatar) {
	dbAvatar = i
}

// dbInternalGroup 实例
var dbInternalGroup interfaces.DBInternalGroup

// SetDBInternalGroup 设置实例
func SetDBInternalGroup(i interfaces.DBInternalGroup) {
	dbInternalGroup = i
}

// dbInternalGroupMember 实例
var dbInternalGroupMember interfaces.DBInternalGroupMember

// SetDBInternalGroupMember 设置实例
func SetDBInternalGroupMember(i interfaces.DBInternalGroupMember) {
	dbInternalGroupMember = i
}

var dbTracePool *sqlx.DB

// SetDBTracePool 设置带有trace数据库连接
func SetDBTracePool(i *sqlx.DB) {
	dbTracePool = i
}

// dbOrgPerm 实例
var dbOrgPerm interfaces.DBOrgPerm

// SetDBOrgPerm 设置实例
func SetDBOrgPerm(i interfaces.DBOrgPerm) {
	dbOrgPerm = i
}

var dbReservedName interfaces.DBReservedName

// SetDBReservedName 设置实例
func SetDBReservedName(i interfaces.DBReservedName) {
	dbReservedName = i
}

var dbActiveUser interfaces.DBActiveUser

// SetDBActiveUser 设置实例
func SetDBActiveUser(i interfaces.DBActiveUser) {
	dbActiveUser = i
}
