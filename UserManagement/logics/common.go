// Package logics Anyshare 业务逻辑层 -通用
package logics

import (
	"context"
	"sort"

	"github.com/kweaver-ai/go-lib/rest"

	"UserManagement/errors"
	"UserManagement/interfaces"
)

const (
	// 使用iota，增加新的枚举时，必须按照顺序添加，不得在中间插入
	// 否则会改变枚举的值，造成outbox handler与db中记录的type对应不上
	_ = iota
	outboxAnonymityAuth
	_ // outboxRegisterApp
	outboxDeleteApp
	outboxUpdateApp
	oubtoxAppDeleted
	outboxAppNameChanged
	outboxUserPWDModified
	outboxContactorDeleted
	outboxInternalGroupDeleted
	outboxDepartDeleted
	outboxOrgManagerChange
	outboxTopDepartDeletedLog
	outboxOrgPermAppAddedLog
	outboxOrgPermAppDeletedLog
	outboxOrgPermAppUpdatedLog
	outboxDefaultPWDModifiedLog
	outboxGroupAddedLog
	outboxGroupDeletedLog
	outboxGroupModifiedLog
	outboxGroupMembersDeletedLog
	outboxGroupMembersAddedLog
	outboxAppRegisteredLog
	outboxAppDeletedLog
	outboxAppModifiedLog
	outboxAppTokenGeneratedLog
	outboxCSFLevelEnumInitedLog
	outboxCSFLevelEnum2InitedLog
	outboxUserExpiredAutoDisabled
	outboxUserNotLoginAutoDisabled
	outboxActiveUserInfoExportedLog
)

// outbox业务类型
const (
	// 默认业务类型
	_ = iota
	// OutboxBusinessAnonymous 匿名认证
	OutboxBusinessAnonymous

	// OutboxBusinessApp 应用账户
	OutboxBusinessApp

	// BusinessConfig 配置设置
	OutboxBusinessConfig

	// OutboxBusinessContactor 联系人
	OutboxBusinessContactor

	// OutboxBusinessDepart
	OutboxBusinessDepart

	// OutboxBusinessGroup 群组
	OutboxBusinessGroup

	// OutboxBusinessOrgPermApp 组织权限应用
	OutboxBusinessOrgPermApp

	// OutboxBusinessInternalGroup 内部群组
	OutboxBusinessInternalGroup

	// OutboxBusinessUser 用户
	OutboxBusinessUser

	// OutboxBusinessActiveUser 活跃用户信息导出
	OutboxBusinessActiveUser

	// 若新增新的业务类型，需在initdb中对anyshare.t_outbox_lock表进行数据初始化
)

var (
	// AdminIDMap 内置管理员账号
	AdminIDMap = map[string]bool{
		interfaces.SystemSysAdmin:   true,
		interfaces.SystemAuditAdmin: true,
		interfaces.SystemSecAdmin:   true,
	}
)

const (
	_ int = iota
	i18nIDObjectsInUnDistributeUserGroup
	i18nIDObjectsInUserNotFound
	i18nIDObjectsInDepartDeleteNotContain
	i18nIDObjectsInDepartNotFound
	i18nIDObjectsInGroupNotFound
	i18nIDObjectsInAppNotFound

	// 活跃日志导出相关
	i18nIDObjectsMonthlyOverallIndex
	i18nIDObjectsYearlyOverallIndex
	i18nIDObjectsIndex
	i18nIDObjectsValue
	i18nIDObjectsTotalUserCount
	i18nIDObjectsActivateCount
	i18nIDObjectsAverageActiveUser
	i18nIDObjectsAverageActiveDegree
	i18nIDObjectsLowestActiveUser
	i18nIDObjectsLowestActiveDegree
	i18nIDObjectsHighestActiveUser
	i18nIDObjectsHighestActiveDegree
	i18nIDObjectsMonthlyDetailedIndex
	i18nIDObjectsYearlyDetailedIndex
	i18nIDObjectsDate
	i18nIDObjectsDailyActiveUser
	i18nIDObjectsDailyActiveDegree
	i18nIDObjectsMonth
	i18nIDObjectsMonthlyActiveUser
	i18nIDObjectsMonthlyActiveDegree
)

// RemoveDuplicatStrs 删除掉相邻重复的
func RemoveDuplicatStrs(s *[]string) {
	// 先进行排序
	sort.Strings(*s)

	// 删除掉相邻重复的
	if len(*s) == 0 {
		return
	}

	left, right := 0, 1
	for ; right < len(*s); right++ {
		if (*s)[left] == (*s)[right] {
			continue
		}

		left++
		(*s)[left] = (*s)[right]
	}

	*s = (*s)[:left+1]
}

// SplitArray 拆分数组，保证in值列表限制在500以内
func SplitArray(arr []string) [][]string {
	length := 500
	total := len(arr)
	count := total / length
	if total%length != 0 {
		count++
	}

	resArray := make([][]string, 0)
	start := 0
	end := 0
	for i := 0; i < count; i++ {
		end = (i + 1) * length
		if i != (count - 1) {
			resArray = append(resArray, arr[start:end])
		} else {
			resArray = append(resArray, arr[start:])
		}
		start = end
	}

	return resArray
}

// Difference 计算差集(a - b)
func Difference(a, b []string) []string {
	m := make(map[string]bool)
	for _, v := range b {
		m[v] = true
	}

	arr := make([]string, 0)
	for _, v := range a {
		_, ok := m[v]
		if !ok {
			arr = append(arr, v)
		}
	}

	return arr
}

// Intersection 计算交集
func Intersection(a, b []string) []string {
	m := make(map[string]bool)
	for _, v := range b {
		m[v] = true
	}

	arr := make([]string, 0)
	for _, v := range a {
		_, ok := m[v]
		if ok {
			arr = append(arr, v)
		}
	}

	return arr
}

// getRolesByUserID 根据用户id获取用户角色id,包含普通用户
func getRolesByUserID(r interfaces.LogicsRole, userID string) (out map[interfaces.Role]bool, err error) {
	roleInfos, err := r.GetRolesByUserIDs([]string{userID})
	if err != nil {
		return
	}

	return roleInfos[userID], err
}

// getRolesByUserID2 根据用户id获取用户角色id,包含普通用户,支持trace
func getRolesByUserID2(ctx context.Context, r interfaces.LogicsRole, userID string) (out map[interfaces.Role]bool, err error) {
	roleInfos, err := r.GetRolesByUserIDs2(ctx, []string{userID})
	if err != nil {
		return
	}

	return roleInfos[userID], err
}

// checkManageAuthority 检测管理权限
func checkManageAuthority(role interfaces.LogicsRole, userID string) (err error) {
	// 获取用户角色信息
	roleIDs, err := getRolesByUserID(role, userID)
	if err != nil {
		return err
	}

	// 超级管理员或者系统管理员角色拥有管理权限
	if roleIDs[interfaces.SystemRoleSuperAdmin] ||
		roleIDs[interfaces.SystemRoleSysAdmin] {
		return nil
	}
	return rest.NewHTTPError("this user do not has the authority", errors.Forbidden, nil)
}

// checkManageAuthority2 检测管理权限,支持trace
func checkManageAuthority2(ctx context.Context, role interfaces.LogicsRole, userID string) (err error) {
	// 获取用户角色信息
	roleIDs, err := getRolesByUserID2(ctx, role, userID)
	if err != nil {
		return err
	}

	// 超级管理员或者系统管理员角色拥有管理权限
	if roleIDs[interfaces.SystemRoleSuperAdmin] ||
		roleIDs[interfaces.SystemRoleSysAdmin] {
		return nil
	}
	return rest.NewHTTPErrorV2(errors.Forbidden, "this user do not has the authority")
}

// checkGetInfoAuthority 检测列举权限
func checkGetInfoAuthority(role interfaces.LogicsRole, userID string) (err error) {
	// 获取用户角色信息
	roleIDs, err := getRolesByUserID(role, userID)
	if err != nil {
		return err
	}

	// 超级管理员、系统管理员角色、安全管理员角色、审计管理员角色拥有列举权限
	if roleIDs[interfaces.SystemRoleSuperAdmin] ||
		roleIDs[interfaces.SystemRoleSysAdmin] ||
		roleIDs[interfaces.SystemRoleSecAdmin] ||
		roleIDs[interfaces.SystemRoleAuditAdmin] {
		return nil
	}
	return rest.NewHTTPError("this user do not has the authority", errors.Forbidden, nil)
}

// checkGetInfoAuthority2 检测列举权限，支持trace
func checkGetInfoAuthority2(ctx context.Context, role interfaces.LogicsRole, userID string) (err error) {
	// 获取用户角色信息
	roleIDs, err := getRolesByUserID2(ctx, role, userID)
	if err != nil {
		return err
	}

	// 超级管理员、系统管理员角色、安全管理员角色、审计管理员角色拥有列举权限
	if roleIDs[interfaces.SystemRoleSuperAdmin] ||
		roleIDs[interfaces.SystemRoleSysAdmin] ||
		roleIDs[interfaces.SystemRoleSecAdmin] ||
		roleIDs[interfaces.SystemRoleAuditAdmin] {
		return nil
	}
	return rest.NewHTTPErrorV2(errors.Forbidden, "this user do not has the authority")
}

// checkAppPerm 检查app是否具有权限
func checkAppPerm(orgPermApp interfaces.DBOrgPermApp, id string, oType interfaces.OrgType, perm interfaces.AppOrgPermValue) (err error) {
	allPerm, err := orgPermApp.GetAppPermByID(id)
	if err != nil {
		return err
	}

	// 判断是否具有修改密码权限
	data, ok := allPerm[oType]
	if ok && data.Value&perm != 0 {
		return nil
	}

	return rest.NewHTTPError("this user do not has the authority", rest.Forbidden, nil)
}

// checkAppPerm2 检查app是否具有权限，支持trace
func checkAppPerm2(ctx context.Context, orgPermApp interfaces.DBOrgPermApp, id string, oType interfaces.OrgType, perm interfaces.AppOrgPermValue) (err error) {
	allPerm, err := orgPermApp.GetAppPermByID2(ctx, id)
	if err != nil {
		return err
	}

	// 判断是否具有修改密码权限
	data, ok := allPerm[oType]
	if ok && data.Value&perm != 0 {
		return nil
	}

	return rest.NewHTTPErrorV2(rest.Forbidden, "this user do not has the authority")
}

// checkUserRole 检查用户角色
func checkUserRole(role interfaces.LogicsRole, id string, allowRoles []interfaces.Role) (err error) {
	var mapRoles map[interfaces.Role]bool
	mapRoles, err = getRolesByUserID(role, id)
	if err != nil {
		return err
	}

	// 如果存在允许的角色，则返回nil
	for _, v := range allowRoles {
		if mapRoles[v] {
			return nil
		}
	}

	return rest.NewHTTPError("this user do not has the authority", rest.Forbidden, nil)
}

// checkUserRole2 检查用户角色，支持trace
func checkUserRole2(ctx context.Context, role interfaces.LogicsRole, id string, allowRoles []interfaces.Role) (err error) {
	var mapRoles map[interfaces.Role]bool
	mapRoles, err = getRolesByUserID2(ctx, role, id)
	if err != nil {
		return err
	}

	// 如果存在允许的角色，则返回nil
	for _, v := range allowRoles {
		if mapRoles[v] {
			return nil
		}
	}

	return rest.NewHTTPErrorV2(rest.Forbidden, "this user do not has the authority")
}
