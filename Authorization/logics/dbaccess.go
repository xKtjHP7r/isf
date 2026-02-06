// Package logics combine Anyshare 业务逻辑层 -组合
package logics

import (
	"github.com/kweaver-ai/proton-rds-sdk-go/sqlx"

	"Authorization/interfaces"
)

// dbPool 实例
var dbPool *sqlx.DB

// SetDBPool  设置实例
func SetDBPool(i *sqlx.DB) {
	dbPool = i
}

// var dbTracePool *sqlx.DB
var dbTracePool *sqlx.DB

// SetDBTracePool 设置实例
func SetDBTracePool(i *sqlx.DB) {
	dbTracePool = i
}

// dbResourceType 实例
var dbResourceType interfaces.DBResourceType

// SetDBResourceType 设置实例
func SetDBResourceType(i interfaces.DBResourceType) {
	dbResourceType = i
}

// dbPolicy 实例
var dbPolicy interfaces.DBPolicy

// SetDBPolicy 设置实例
func SetDBPolicy(i interfaces.DBPolicy) {
	dbPolicy = i
}

// dbPolicyCalc 实例
var dbPolicyCalc interfaces.DBPolicyCalc

// SetDBPolicyCalc 设置实例
func SetDBPolicyCalc(i interfaces.DBPolicyCalc) {
	dbPolicyCalc = i
}

// dbRole 实例
var dbRole interfaces.DBRole

// SetDBRole 设置实例
func SetDBRole(i interfaces.DBRole) {
	dbRole = i
}

var dbRoleMember interfaces.DBRoleMember

// SetDBRoleMember 设置实例
func SetDBRoleMember(i interfaces.DBRoleMember) {
	dbRoleMember = i
}

var dbObligationType interfaces.DBObligationType

// SetDBObligationType 设置实例
func SetDBObligationType(i interfaces.DBObligationType) {
	dbObligationType = i
}

var dbObligation interfaces.DBObligation

// SetDBObligation 设置实例
func SetDBObligation(i interfaces.DBObligation) {
	dbObligation = i
}

var dbResourceTypeHierarchy interfaces.DBResourceTypeHierarchy

// SetDBResourceTypeHierarchy 设置实例
func SetDBResourceTypeHierarchy(i interfaces.DBResourceTypeHierarchy) {
	dbResourceTypeHierarchy = i
}
