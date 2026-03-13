// Package logics user AnyShare 用户业务逻辑层
package logics

import (
	"context"
	"math/rand"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kweaver-ai/go-lib/observable"
	"github.com/kweaver-ai/go-lib/rest"
	"github.com/kweaver-ai/proton-rds-sdk-go/sqlx"

	"UserManagement/common"
	"UserManagement/errors"
	"UserManagement/interfaces"
)

type user struct {
	userDB        interfaces.DBUser
	departmentDB  interfaces.DBDepartment
	contactorDB   interfaces.DBContactor
	groupMemberDB interfaces.DBGroupMember
	configDB      interfaces.DBConfig
	pool          *sqlx.DB
	tracePool     *sqlx.DB
	logger        common.Logger
	ob            interfaces.LogicsOutbox
	role          interfaces.LogicsRole
	hydra         interfaces.DrivenHydra
	orgPermAppDB  interfaces.DBOrgPermApp
	avatar        interfaces.LogicsAvatar
	internalGroup interfaces.LogicsInternalGroup
	config        interfaces.LogicsConfig
	trace         observable.Tracer
	orgPerm       interfaces.LogicsOrgPerm
	event         interfaces.LogicsEvent
	i18n          *common.I18n
	reservedName  interfaces.LogicsReservedName
	eacpLog       interfaces.DrivenEacpLog
	messageBroker interfaces.DrivenMessageBroker
}

var (
	uOnce   sync.Once
	uLogics *user

	nOffsetTime     = 5
	randomTimeRange = 300
)

// NewUser 创建新的user对象
func NewUser() *user {
	uOnce.Do(func() {
		uLogics = &user{
			userDB:        dbUser,
			departmentDB:  dbDepartment,
			contactorDB:   dbContactor,
			groupMemberDB: dbGroupMember,
			configDB:      dbConfig,
			pool:          dbPool,
			tracePool:     dbTracePool,
			logger:        common.NewLogger(),
			ob:            NewOutbox(OutboxBusinessUser),
			hydra:         dnHydra,
			orgPermAppDB:  dbOrgPermApp,
			role:          NewRole(),
			avatar:        NewAvatar(),
			internalGroup: NewInternalGroup(),
			config:        NewConfig(),
			trace:         common.SvcARTrace,
			orgPerm:       NewOrgPerm(),
			event:         NewEvent(),
			i18n: common.NewI18n(common.I18nMap{
				i18nIDObjectsInUnDistributeUserGroup: {
					interfaces.SimplifiedChinese:  "未分配组",
					interfaces.TraditionalChinese: "未分配組",
					interfaces.AmericanEnglish:    "Unassigned Group",
				},
				i18nIDObjectsInUserNotFound: {
					interfaces.SimplifiedChinese:  "用户不存在",
					interfaces.TraditionalChinese: "用戶不存在",
					interfaces.AmericanEnglish:    "This user does not exist",
				},
			}),
			reservedName:  NewReservedName(),
			eacpLog:       dnEacpLog,
			messageBroker: dnMessageBroker,
		}

		uLogics.ob.RegisterHandlers(outboxUserExpiredAutoDisabled, uLogics.userExpiredAutoDisabled)
		uLogics.ob.RegisterHandlers(outboxUserNotLoginAutoDisabled, uLogics.userNotLoginAutoDisabled)

		uLogics.ob.RegisterHandlers(outboxUserPWDModified, func(content interface{}) error {
			contentJSON := content.(map[string]interface{})
			userID := contentJSON["id"].(string)

			// 删除认证与授权会话
			err := uLogics.hydra.DeleteConsentAndLogin("", userID)
			return err
		})

		uLogics.event.RegisterUserDeleted(uLogics.onUserDeleted)

		go uLogics.userAutoDisableThread()
	})

	return uLogics
}

// userNotLoginAutoDisabled 用户长时间未登录自动禁用
func (u *user) userNotLoginAutoDisabled(content interface{}) (err error) {
	contentJSON := content.(map[string]interface{})
	userID := contentJSON["id"].(string)
	displayName := contentJSON["display_name"].(string)
	loginName := contentJSON["login_name"].(string)

	// 发送mq消息
	err = u.messageBroker.UserStatusChanged(userID, false)
	if err != nil {
		u.logger.Errorf("userNotLoginAutoDisabled messageBroker UserStatusChanged error:%v", err)
		return err
	}

	// 记录日志
	err = u.eacpLog.OpUserNotLoginDisabled(displayName, loginName)
	if err != nil {
		u.logger.Errorf("userNotLoginAutoDisabled err:%v", err)
	}
	return err
}

// userExpiredAutoDisabled 用户过期自动禁用
func (u *user) userExpiredAutoDisabled(content interface{}) (err error) {
	contentJSON := content.(map[string]interface{})
	userID := contentJSON["id"].(string)
	displayName := contentJSON["display_name"].(string)
	loginName := contentJSON["login_name"].(string)

	// 发送mq消息
	err = u.messageBroker.UserStatusChanged(userID, false)
	if err != nil {
		u.logger.Errorf("userExpiredAutoDisabled messageBroker UserStatusChanged error:%v", err)
		return err
	}

	// 记录日志
	err = u.eacpLog.OpUserExpiredDisabled(displayName, loginName)
	if err != nil {
		u.logger.Errorf("userExpiredAutoDisabled err:%v", err)
	}
	return err
}

// 权限检查
func (u *user) checkSearchUserAuthority(ctx context.Context, visitor *interfaces.Visitor, ks *interfaces.UserSearchInDepartKeyScope,
	k *interfaces.UserSearchInDepartKey, r interfaces.Role) (err error) {
	// 权限检查
	if ks.BDepartmentID && k.DepartmentID != "-1" {
		// 检查部门是否存在
		var departInfos []interfaces.DepartmentDBInfo
		departInfos, err = u.departmentDB.GetDepartmentInfoByIDs(ctx, []string{k.DepartmentID})
		if err != nil {
			return err
		}

		if len(departInfos) == 0 {
			return rest.NewHTTPErrorV2(rest.BadRequest, "this department not exist")
		}
		// 检查角色范围
		var result bool
		result, err = u.checkDepartmentInUserScope(ctx, visitor, r, departInfos[0].ID)
		if err != nil {
			return err
		}
		if !result {
			return rest.NewHTTPErrorV2(rest.Forbidden, "this user do not has this authority")
		}
	} else if r != interfaces.SystemRoleAuditAdmin && r != interfaces.SystemRoleSecAdmin &&
		r != interfaces.SystemRoleSysAdmin && r != interfaces.SystemRoleSuperAdmin {
		// 未分配组和所有用户下搜索 只支持 超级管理员，安全管理员，审计管理员和系统管理员
		return rest.NewHTTPErrorV2(rest.Forbidden, "this user has no authority")
	}

	return nil
}

// SearchUsers 搜索用户信息
func (u *user) SearchUsers(ctx context.Context, visitor *interfaces.Visitor, ks *interfaces.UserSearchInDepartKeyScope, k *interfaces.UserSearchInDepartKey,
	f interfaces.UserBaseInfoRange, r interfaces.Role) (out []interfaces.UserBaseInfo, num int, err error) {
	// trace
	u.trace.SetInternalSpanName("业务逻辑-搜索用户信息")
	newCtx, span := u.trace.AddInternalTrace(ctx)
	defer func() { u.trace.TelemetrySpanEnd(span, err) }()

	// 检测调用者是否拥有指定角色
	var mapRoles map[interfaces.Role]bool
	mapRoles, err = getRolesByUserID2(newCtx, u.role, visitor.ID)
	if err != nil {
		return
	}

	if !mapRoles[r] {
		return nil, 0, rest.NewHTTPErrorV2(rest.BadRequest, "this user do not has this role")
	}

	// 权限检查
	err = u.checkSearchUserAuthority(newCtx, visitor, ks, k, r)
	if err != nil {
		return
	}

	// 搜索用户信息
	userDBInfos, err := u.userDB.SearchUsers(newCtx, ks, k)
	if err != nil {
		return nil, 0, err
	}

	// 搜索用户长度
	num, err = u.userDB.SearchUsersCount(newCtx, ks, k)
	if err != nil {
		return nil, 0, err
	}

	userIDs := make([]string, 0, len(userDBInfos))
	for k := range userDBInfos {
		userIDs = append(userIDs, userDBInfos[k].ID)
	}

	// 获取用户直属部门信息
	depInfos := make(map[string][][]interfaces.ObjectBaseInfo)
	if f.ShowParentDeps {
		for _, v := range userIDs {
			depInfos[v], _, err = u.getUserParentDepsInfo(v, newCtx)
			if err != nil {
				return nil, 0, err
			}
		}
	}

	// 获取用户的角色role信息
	var userRoleInfo map[string]map[interfaces.Role]bool
	if f.ShowRoles {
		userRoleInfo, err = u.role.GetRolesByUserIDs(userIDs)
		if err != nil {
			return nil, 0, err
		}
	}

	// 获取管理员信息
	managerMaps := make(map[string]interfaces.NameInfo)
	if f.ShowManager {
		managerMaps, err = u.getManagerInfos(userDBInfos)
		if err != nil {
			return nil, 0, err
		}
	}

	out = make([]interfaces.UserBaseInfo, 0)
	for k := range userDBInfos {
		temp := u.handleUserInfo(f, &userDBInfos[k], userRoleInfo[userDBInfos[k].ID], depInfos[userDBInfos[k].ID], nil, "", nil, nil, managerMaps)

		// 单独处理未分配组
		if len(temp.ParentDeps) == 0 {
			temp.ParentDeps = [][]interfaces.ObjectBaseInfo{
				{
					{
						ID:   "-1",
						Name: u.i18n.Load(i18nIDObjectsInUnDistributeUserGroup, visitor.Language),
						Type: "department",
						Code: "",
					},
				},
			}
		}
		out = append(out, temp)
	}

	return out, num, nil
}

// getManagerInfos 获取负责人信息
func (u *user) getManagerInfos(userDBInfos []interfaces.UserDBInfo) (managerMaps map[string]interfaces.NameInfo, err error) {
	managerMaps = make(map[string]interfaces.NameInfo)
	managerIDs := make([]string, 0)
	for k := range userDBInfos {
		if userDBInfos[k].ManagerID != "" {
			managerIDs = append(managerIDs, userDBInfos[k].ManagerID)
		}
	}

	managerInfos, existManagerIDs, err := u.userDB.GetUserName(managerIDs)
	if err != nil {
		return nil, err
	}

	// 如果存在管理员不存在则报错
	if len(existManagerIDs) != len(managerIDs) {
		temp := Difference(managerIDs, existManagerIDs)
		u.logger.Warnf("SearchUsers get managers ids warnning: exist managers not exist:%v", temp)
	}

	for k := range managerInfos {
		managerMaps[managerInfos[k].ID] = interfaces.NameInfo{
			ID:   managerInfos[k].ID,
			Name: managerInfos[k].Name,
		}
	}
	return
}

// 检查部门是否权限范围内
func (u *user) checkDepartmentInUserScope(ctx context.Context, v *interfaces.Visitor, roles interfaces.Role, deptID string) (result bool, err error) {
	if roles == interfaces.SystemRoleSuperAdmin || roles == interfaces.SystemRoleSysAdmin ||
		roles == interfaces.SystemRoleSecAdmin || roles == interfaces.SystemRoleAuditAdmin {
		return true, nil
	}

	var depIDs []string
	if roles == interfaces.SystemRoleOrgAudit {
		depIDs, err = u.userDB.GetOrgAduitDepartInfo2(ctx, v.ID)
		if err != nil {
			return false, err
		}
	}

	if roles == interfaces.SystemRoleOrgManager {
		depIDs, err = u.userDB.GetOrgManagerDepartInfo2(ctx, v.ID)
		if err != nil {
			return false, err
		}
	}

	for _, v := range depIDs {
		if v == deptID {
			result = true
			break
		}
	}

	return
}

// onUserDeleted 用户删除事件-联系人组相关
func (u *user) onUserDeleted(userID string) (err error) {
	return u.userDB.DeleteUserManagerID(userID)
}

// ConvertUserName 根据userid批量获取用户显示名
func (u *user) ConvertUserName(visitor *interfaces.Visitor, userIDs []string, bStrict bool) (infoArray []interfaces.NameInfo, err error) {
	infoArray = make([]interfaces.NameInfo, 0)
	if len(userIDs) == 0 {
		return infoArray, nil
	}

	// 去掉重复id
	copyIDs := make([]string, len(userIDs))
	copy(copyIDs, userIDs)
	RemoveDuplicatStrs(&copyIDs)

	// 获取用户显示名
	infoTmp, existIDs, err := u.userDB.GetUserName(copyIDs)
	if err != nil {
		return nil, err
	}

	// 如果严格模式， 且有用户不存在，则返回错误
	if bStrict && len(copyIDs) != len(existIDs) {
		// 获取不存在的用户id
		notExistIDs := Difference(copyIDs, existIDs)
		err = rest.NewHTTPErrorV2(errors.UserNotFound, u.i18n.Load(i18nIDObjectsInUserNotFound, visitor.Language),
			rest.SetCodeStr(errors.StrBadRequestUserNotFound),
			rest.SetDetail(map[string]interface{}{"ids": notExistIDs}))
		return nil, err
	}

	// 类型转换
	for k := range infoTmp {
		tmp := interfaces.NameInfo{
			Name: infoTmp[k].Name,
			ID:   infoTmp[k].ID,
		}
		infoArray = append(infoArray, tmp)
	}

	return infoArray, nil
}

// GetUserEmails 根据userid批量获取用户邮箱
func (u *user) GetUserEmails(visitor *interfaces.Visitor, userIDs []string) (infoArray []interfaces.EmailInfo, err error) {
	infoArray = make([]interfaces.EmailInfo, 0)
	if len(userIDs) == 0 {
		return infoArray, nil
	}

	// 去掉重复id
	copyIDs := make([]string, len(userIDs))
	copy(copyIDs, userIDs)
	RemoveDuplicatStrs(&copyIDs)

	// 获取用户邮箱
	infoTmp, err := u.userDB.GetUserDBInfo(copyIDs)
	if err != nil {
		return nil, err
	}

	// 有用户不存在
	IDLen := len(infoTmp)
	if len(copyIDs) != IDLen {
		existIDs := make([]string, 0, IDLen)
		for k := range infoTmp {
			existIDs = append(existIDs, infoTmp[k].ID)
		}
		// 获取不存在的用户id
		notExistIDs := Difference(copyIDs, existIDs)
		err = rest.NewHTTPErrorV2(errors.UserNotFound, u.i18n.Load(i18nIDObjectsInUserNotFound, visitor.Language),
			rest.SetCodeStr(errors.StrBadRequestUserNotFound),
			rest.SetDetail(map[string]interface{}{"ids": notExistIDs}))
		return nil, err
	}

	// 类型转换
	for k := range infoTmp {
		tmp := interfaces.EmailInfo{
			Email: infoTmp[k].Email,
			ID:    infoTmp[k].ID,
		}
		infoArray = append(infoArray, tmp)
	}

	return infoArray, nil
}

// GetAllBelongDepartmentIDs 获取用户所属部门id(直属部门+父部门)
func (u *user) GetAllBelongDepartmentIDs(userID string) (deptIDs []string, err error) {
	deptIDs = make([]string, 0)

	// 获取用户所有的路径
	paths, err := u.userDB.GetUsersPath([]string{userID})
	if err != nil {
		return nil, err
	}

	// 获取用户所属部门ID
	for _, v := range paths[userID] {
		// 如果是未分配组，则跳过
		if v == "-1" {
			continue
		}
		tempIDs := strings.Split(v, "/")
		deptIDs = append(deptIDs, tempIDs...)
	}

	// 去掉重复id
	RemoveDuplicatStrs(&deptIDs)

	return deptIDs, nil
}

// GetUsersInDepartments 检测用户是否在部门内部， 返回存在的用户ID
func (u *user) GetUsersInDepartments(userIDs, departmentIDs []string) (outUserIDs []string, err error) {
	outUserIDs = make([]string, 0)
	if len(userIDs) == 0 || len(departmentIDs) == 0 {
		return
	}

	// 获取范围内的用户ID
	outUserIDs, err = u.userDB.GetUsersInDepartments(userIDs, departmentIDs)
	RemoveDuplicatStrs(&outUserIDs)

	return
}

// GetAccessorIDsOfUser 获取指定用户的访问令牌
func (u *user) GetAccessorIDsOfUser(userID string) (accessorIDs []string, err error) {
	accessorIDs = make([]string, 0)
	// 判断用户是否存在
	_, info, err := u.userDB.GetUserName([]string{userID})
	if err != nil {
		return accessorIDs, err
	}
	if len(info) != 1 {
		err = rest.NewHTTPErrorV2(errors.NotFound, "user does not exist")
		return accessorIDs, err
	}

	// 获取用户所有的父部门
	deptIDs, err := u.GetAllBelongDepartmentIDs(userID)
	if err != nil {
		return nil, err
	}

	// 获取包含用户的联系人组
	contactorIDs, err := u.contactorDB.GetUserAllBelongContactorIDs(userID)
	if err != nil {
		return nil, err
	}

	// 获取包含用户和部门的所有用户组
	accessorIDs = append(accessorIDs, userID)
	accessorIDs = append(accessorIDs, deptIDs...)
	groupIDs, _, err := u.groupMemberDB.GetMembersBelongGroupIDs(accessorIDs)
	if err != nil {
		return nil, err
	}

	// 获取包含用户的所有内部组
	memberInfo := interfaces.InternalGroupMember{
		ID:   userID,
		Type: interfaces.User,
	}
	internalGroupIds, err := u.internalGroup.GetBelongGroups(memberInfo)
	if err != nil {
		return nil, err
	}

	// 整理令牌
	accessorIDs = append(accessorIDs, contactorIDs...)
	accessorIDs = append(accessorIDs, groupIDs...)
	accessorIDs = append(accessorIDs, internalGroupIds...)

	return
}

// GetNorlmalUserInfo 普通用户获取自己信息
func (u *user) GetNorlmalUserInfo(ctx context.Context, visitor *interfaces.Visitor, oGetRange interfaces.UserBaseInfoRange) (out interfaces.UserBaseInfo, err error) {
	u.trace.SetInternalSpanName("业务逻辑-获取普通用户信息")
	newCtx, span := u.trace.AddInternalTrace(ctx)
	defer func() { u.trace.TelemetrySpanEnd(span, err) }()

	// 判断用户是否为普通用户
	if visitor.Type != interfaces.RealName || AdminIDMap[visitor.ID] {
		err = rest.NewHTTPError("only support normal user", rest.BadRequest, nil)
		return
	}

	// 获取用户信息
	infos, err := u.GetUsersBaseInfo(newCtx, visitor, []string{visitor.ID}, oGetRange, true)
	if err != nil {
		return
	}

	return infos[0], nil
}

// GetUsersBaseInfo 获取多个用户基本信息
//
//nolint:gocyclo
func (u *user) GetUsersBaseInfo(ctx context.Context, visitor *interfaces.Visitor, userIDs []string,
	info interfaces.UserBaseInfoRange, bStrict bool) (out []interfaces.UserBaseInfo, err error) {
	u.trace.SetInternalSpanName("业务逻辑-获取用户基本信息")
	newCtx, span := u.trace.AddInternalTrace(ctx)
	defer func() { u.trace.TelemetrySpanEnd(span, err) }()

	var userDBInfo []interfaces.UserDBInfo
	var userRoleInfo map[string]map[interfaces.Role]bool
	var accessorIDs []string

	// 重复检测
	nOriginLen := len(userIDs)
	RemoveDuplicatStrs(&userIDs)
	if bStrict && nOriginLen != len(userIDs) {
		err = rest.NewHTTPError("there are same users", rest.BadRequest, nil)
		return
	}

	// 检查用户是否存在并获取用户数据库信息
	userDBInfo, err = u.userDB.GetUserDBInfo(userIDs)
	if err != nil {
		return out, err
	}

	// 有用户不存在
	IDLen := len(userIDs)
	if bStrict && len(userDBInfo) != IDLen {
		existIDs := make([]string, 0, IDLen)
		for k := range userDBInfo {
			existIDs = append(existIDs, userDBInfo[k].ID)
		}
		// 获取不存在的用户id
		notExistIDs := Difference(userIDs, existIDs)
		err = rest.NewHTTPErrorV2(errors.NotFound, "those users are not existing",
			rest.SetDetail(map[string]interface{}{"ids": notExistIDs}))
		return nil, err
	}

	userIDs = make([]string, 0, len(userDBInfo))
	for k := range userDBInfo {
		userIDs = append(userIDs, userDBInfo[k].ID)
	}

	// 获取用户的角色role信息
	if info.ShowRoles {
		userRoleInfo, err = u.role.GetRolesByUserIDs(userIDs)
		if err != nil {
			return out, err
		}
	}

	// 获取用户直属部门信息
	depInfos := make(map[string][][]interfaces.ObjectBaseInfo)
	if info.ShowParentDeps {
		for _, v := range userIDs {
			depInfos[v], _, err = u.getUserParentDepsInfo(v, nil)
			if err != nil {
				return out, err
			}
		}
	}

	// 获取用户头像URL
	avatars := make(map[string]string)
	if info.ShowAvatar {
		for _, v := range userIDs {
			avatars[v], err = u.avatar.Get(newCtx, visitor, v)
			if err != nil {
				return out, err
			}
		}
	}

	// 获取用户自定义属性
	customAttrs := make(map[string]map[string]interface{})
	if info.ShowCustomAttr {
		for _, v := range userIDs {
			customAttrs[v], err = u.userDB.GetUserCustomAttr(v)
			if err != nil {
				return nil, err
			}
		}
	}

	// 获取用户及其部门所属用户组
	groups := make(map[string][]interfaces.GroupInfo)
	if info.ShowGroups {
		for _, v := range userIDs {
			// 获取用户所有的父部门
			var deptIDs []string
			deptIDs, err = u.GetAllBelongDepartmentIDs(v)
			if err != nil {
				return nil, err
			}
			// 获取包含用户和部门的所有用户组
			accessorIDs = append(accessorIDs, v)
			accessorIDs = append(accessorIDs, deptIDs...)
			_, groups[v], err = u.groupMemberDB.GetMembersBelongGroupIDs(accessorIDs)
			if err != nil {
				return nil, err
			}
		}
	}

	// 获取管理员信息
	var managerMaps map[string]interfaces.NameInfo
	if info.ShowManager {
		managerMaps, err = u.getManagerInfos(userDBInfo)
		if err != nil {
			return nil, err
		}
	}

	// 数据整理
	for k := range userDBInfo {
		curUser := &userDBInfo[k]
		temp := u.handleUserInfo(info, curUser, userRoleInfo[curUser.ID], depInfos[curUser.ID], []string{}, avatars[curUser.ID], groups[curUser.ID], customAttrs[curUser.ID], managerMaps)
		out = append(out, temp)
	}

	return
}

// GetUsersBaseInfoByTelephones 获取用户基本信息
func (u *user) GetUserBaseInfoByTelephone(ctx context.Context, visitor *interfaces.Visitor, tel string, rg interfaces.UserBaseInfoRange) (result bool, out interfaces.UserBaseInfo, err error) {
	// trace
	u.trace.SetInternalSpanName("业务逻辑-根据手机号获取用户信息")
	newCtx, span := u.trace.AddInternalTrace(ctx)
	defer func() { u.trace.TelemetrySpanEnd(span, err) }()

	if visitor.Type != interfaces.App {
		err = rest.NewHTTPErrorV2(rest.Forbidden, "this user has no authority")
		return
	}
	err = checkAppPerm(u.orgPermAppDB, visitor.ID, interfaces.User, interfaces.Read)
	if err != nil {
		return
	}

	// 手机号解密
	telephone, err := decodeRSA(tel, RSA2048)
	if err != nil {
		err = rest.NewHTTPErrorV2(rest.BadRequest, err.Error())
		return false, out, err
	}

	var userDBInfo []interfaces.UserDBInfo
	// 检查用户是否存在并获取用户数据库信息
	userDBInfo, err = u.userDB.GetUserDBInfoByTels(newCtx, []string{telephone})
	if err != nil {
		return false, out, err
	}

	if len(userDBInfo) == 1 {
		result = true
		out = u.handleUserBaseInfo(rg, &userDBInfo[0])
	}

	return
}

// GetUserInfoByAccount 通过账户名匹配账户信息
func (u *user) GetUserInfoByAccount(account string, enableIDCardLogin, enablePrefixMatch bool) (result bool, userInfo interfaces.UserBaseInfo, err error) {
	var userDBInfo interfaces.UserDBInfo
	// (1) 精确匹配: 根据用户名匹配账户信息
	userDBInfo, err = u.userDB.GetUserInfoByAccount(account)
	if err != nil {
		return
	}

	// (2) 精确匹配: 根据身份证号匹配账户信息
	if userDBInfo.ID == "" && enableIDCardLogin {
		var desAccount string
		desAccount, err = encodeDes(account, PadNormal)
		if err != nil {
			return
		}

		userDBInfo, err = u.userDB.GetUserInfoByIDCard(desAccount)
		if err != nil {
			return
		}
	}

	// (3) 前缀匹配
	if userDBInfo.ID == "" && enablePrefixMatch {
		userDBInfo, err = u.userDB.GetDomainUserInfoByAccount(account)
		if err != nil {
			return
		}

		// 匹配到域账户
		// 处理：域用户如果登录名为zhangying@qq.com@test2.develop.cn，而zhangying, zhangying@qq.com zhangying@qq.com@test2.develop.cn 都可以登录的问题
		if userDBInfo.ID != "" {
			index := strings.LastIndex(userDBInfo.Account, "@")
			if !strings.EqualFold(account, userDBInfo.Account[:index]) {
				return
			}
		}
	}

	// 账户不存在
	if userDBInfo.ID == "" {
		return
	}

	// 检查账户是否处于锁定状态
	if err = u.checkAccountLocked(&userDBInfo); err != nil {
		return
	}

	infoRange := interfaces.UserBaseInfoRange{
		ShowAccount:        true,
		ShowAuthType:       true,
		ShowPwdErrCnt:      true,
		ShowPwdErrLastTime: true,
		ShowEnable:         true,
		ShowLDAPType:       true,
		ShowDomanPath:      true,
	}
	return true, u.handleUserBaseInfo(infoRange, &userDBInfo), nil
}

func (u *user) checkAccountLocked(user *interfaces.UserDBInfo) (err error) {
	// 获取配置信息
	rg := make(map[interfaces.ConfigKey]bool)
	rg[interfaces.EnablePWDLock] = true
	rg[interfaces.PWDErrCnt] = true
	rg[interfaces.PWDLockTime] = true
	rg[interfaces.EnableThirdPwdLock] = true
	curConfig, err := u.config.GetConfig(rg)
	if err != nil {
		return
	}

	// 本地账户锁定前置条件：EnablePwdLock=true
	// 域用户/第三方账户锁定前置条件：EnablePwdLock=true && EnableThirdPwdLock=true，可参考方案：https://confluence.aishu.cn/pages/viewpage.action?pageId=100083263
	if curConfig.EnablePwdLock && (user.AuthType == interfaces.Local || curConfig.EnableThirdPwdLock) {
		// 账户处于锁定状态，但锁定时间已到，需要解锁该账户
		// 为了方便测试，这里引入业务时间
		curTimeStamp := common.Now().Unix()
		if user.PWDErrLatestTime+curConfig.PwdLockTime*60 <= curTimeStamp {
			err = u.userDB.UpdatePwdErrInfo(user.ID, 0, curTimeStamp)
			if err != nil {
				return
			}

			// 解锁账户后，更新内存值，不再重新读取数据库。
			user.PWDErrCnt = 0
			user.PWDErrLatestTime = curTimeStamp
		}
	}

	return
}

// GetUserBaseInfoInScope 获取范围内用户基本信息
func (u *user) GetUserBaseInfoInScope(visitor *interfaces.Visitor, role interfaces.Role, userIDs []string, info interfaces.UserBaseInfoRange) (out []interfaces.UserBaseInfo, err error) {
	var userDBInfo []interfaces.UserDBInfo

	// 检测调用者是否拥有指定角色
	var mapRoles map[interfaces.Role]bool
	mapRoles, err = getRolesByUserID(u.role, visitor.ID)
	if err != nil {
		return
	}
	if !mapRoles[role] {
		return out, rest.NewHTTPError("this user do not has this role", rest.BadRequest, nil)
	}

	// 检查用户是否存在并获取用户数据库信息
	RemoveDuplicatStrs(&userIDs)
	userDBInfo, err = u.userDB.GetUserDBInfo(userIDs)
	if err != nil {
		return out, err
	}
	if len(userDBInfo) != len(userIDs) {
		err = rest.NewHTTPErrorV2(errors.NotFound, "user does not exist")
		return out, err
	}

	// 检查用户是否在调用者管辖范围内
	ret, err := u.checkUserInRange(visitor.ID, role, userIDs)
	if err != nil {
		return out, err
	}
	for _, v := range ret {
		if !v {
			err = rest.NewHTTPErrorV2(errors.NotFound, "user does not exist")
			return out, err
		}
	}

	// 数据整理
	for index := 0; index < len(userIDs); index++ {
		// 获取用户直属部门信息
		paths := make([]string, 0)
		if info.ShowParentDepPaths {
			_, paths, err = u.getUserParentDepsInfo(userDBInfo[index].ID, nil)
			if err != nil {
				return out, err
			}
		}

		// 数据整理
		temp := u.handleUserInfo(info, &userDBInfo[index], mapRoles, [][]interfaces.ObjectBaseInfo{}, paths, "", nil, nil, nil)
		out = append(out, temp)
	}
	return
}

// SearchUsersByKeyInScope 在范围内按照关键字搜索用户信息
func (u *user) SearchUsersByKeyInScope(ctx context.Context, visitor *interfaces.Visitor, info interfaces.OrgShowPageInfo) (out []interfaces.SearchUserInfo, num int, err error) {
	// trace
	u.trace.SetInternalSpanName("业务逻辑-在管辖范围内搜索用户")
	newCtx, span := u.trace.AddInternalTrace(ctx)
	defer func() { u.trace.TelemetrySpanEnd(span, err) }()

	out = make([]interfaces.SearchUserInfo, 0)
	var userInfo []interfaces.UserDBInfo
	var depIDs []string

	// 获取搜索限制范围
	var bNoScope, result bool
	switch info.Role {
	case interfaces.SystemRoleSuperAdmin, interfaces.SystemRoleSysAdmin, interfaces.SystemRoleSecAdmin, interfaces.SystemRoleAuditAdmin:
		bNoScope = true
	case interfaces.SystemRoleOrgAudit:
		depIDs, err = u.userDB.GetOrgAduitDepartInfo2(newCtx, visitor.ID)
	case interfaces.SystemRoleOrgManager:
		depIDs, err = u.userDB.GetOrgManagerDepartInfo2(newCtx, visitor.ID)
	case interfaces.SystemRoleNormalUser:
		ctx := context.Background()
		result, err = u.orgPerm.CheckPerms(ctx, visitor.ID, interfaces.User, interfaces.OPRead)
		if err != nil || !result {
			return
		}
		bNoScope = true
	default:
		return
	}

	if err != nil {
		return
	}

	// 获取所有限制范围内部门
	var allDepIDs []string
	allDepIDs, err = getAllChildDeparmentIDsByIDs(u.departmentDB, depIDs, newCtx)
	allDepIDs = append(allDepIDs, depIDs...)
	if err != nil {
		return
	}

	userInfo, err = u.userDB.SearchOrgUsersByKey(newCtx, bNoScope, false, info.Keyword, info.Offset, info.Limit,
		info.BOnlyShowEnabledUser, info.BOnlyShowAssignedUser, allDepIDs)
	if err != nil {
		return
	}

	var tempUserInfo []interfaces.UserDBInfo
	tempUserInfo, err = u.userDB.SearchOrgUsersByKey(newCtx, bNoScope, true, info.Keyword, info.Offset, info.Limit,
		info.BOnlyShowEnabledUser, info.BOnlyShowAssignedUser, allDepIDs)
	if err != nil {
		return
	}
	num = len(tempUserInfo)

	// 获取用户的直属部门
	for k := range userInfo {
		var temp interfaces.SearchUserInfo

		_, temp.ParentDepPaths, err = u.getUserParentDepsInfo(userInfo[k].ID, newCtx)
		if err != nil {
			return
		}

		temp.ID = userInfo[k].ID
		temp.Name = userInfo[k].Name
		temp.Account = userInfo[k].Account
		temp.Type = "user"
		out = append(out, temp)
	}
	return
}

// ModifyUserInfo 修改用户信息
func (u *user) ModifyUserInfo(visitor *interfaces.Visitor, bRange interfaces.UserUpdateRange, info *interfaces.UserBaseInfo) (err error) {
	// 检查visitor是否有权限
	if bRange.UpdatePWD {
		err = u.checkModifyUserPWDAuthority(visitor)
		if err != nil {
			return
		}
	}

	// 获取配置信息
	var pwdConfig interfaces.Config
	configKeys := make(map[interfaces.ConfigKey]bool)
	configKeys[interfaces.PWDErrCnt] = true
	configKeys[interfaces.PWDExpireTime] = true
	configKeys[interfaces.PWDLockTime] = true
	configKeys[interfaces.EnablePWDLock] = true
	configKeys[interfaces.StrongPWDLength] = true
	configKeys[interfaces.StrongPWDStatus] = true
	configKeys[interfaces.EnableDesPassWord] = true
	if bRange.UpdatePWD {
		// 获取密码配置信息
		pwdConfig, err = u.configDB.GetConfig(configKeys)
		if err != nil {
			return
		}
	}

	// 参数检查
	var strPwd string
	if bRange.UpdatePWD {
		strPwd, err = u.checkModifyPwdParam(info, &pwdConfig)
		if err != nil {
			return
		}
	}

	// 数据处理
	dbInfo := interfaces.UserDBInfo{
		ID: info.ID,
	}
	if bRange.UpdatePWD {
		dbInfo.Password, dbInfo.DesPassword, dbInfo.NtlmPassword, err = u.generatePWD(strPwd, &pwdConfig)
		if err != nil {
			return
		}
	}

	// 获取事务处理器
	tx, err := u.pool.Begin()
	if err != nil {
		return
	}

	// 异常时Rollback
	defer func() {
		switch err {
		case nil:
			// 提交事务
			if err = tx.Commit(); err != nil {
				u.logger.Errorf("ModifyUserInfo Transaction Commit Error:%v", err)
				return
			}

			// notify outbox推送线程
			u.ob.NotifyPushOutboxThread()
		default:
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				u.logger.Errorf("ModifyUserInfo Rollback err:%v", rollbackErr)
			}
		}
	}()

	// 修改用户信息
	err = u.userDB.ModifyUserInfo(bRange, &dbInfo, tx)
	if err != nil {
		return
	}

	// 处理outbox信息
	if bRange.UpdatePWD {
		// 插入outbox 信息
		contentJSON := make(map[string]interface{})
		contentJSON["id"] = info.ID

		err = u.ob.AddOutboxInfo(outboxUserPWDModified, contentJSON, tx)
		if err != nil {
			u.logger.Errorf("Add Outbox Info err:%v", err)
			return
		}
	}
	return
}

// IncrementModifyUserInfo 增量修改用户信息
func (u *user) IncrementModifyUserInfo(ctx context.Context, visitor *interfaces.Visitor, bRange interfaces.UserUpdateRange, info *interfaces.UserBaseInfo) (err error) {
	// trace
	u.trace.SetInternalSpanName("业务逻辑-增量修改用户信息")
	newCtx, span := u.trace.AddInternalTrace(ctx)
	defer func() { u.trace.TelemetrySpanEnd(span, err) }()
	// 检查visitor是否有权限
	if bRange.CustomAttr {
		if visitor != nil && visitor.ID != "" {
			err = u.checkIncrementModifyUserInfoAuthority(newCtx, visitor)
			if err != nil {
				return
			}
		}
	}

	// 检查用户是否存在
	userInfo, err := u.userDB.GetUserDBInfo2(ctx, []string{info.ID})
	if err != nil {
		return
	}
	if len(userInfo) != 1 {
		err = rest.NewHTTPErrorV2(errors.NotFound, "this user is not existing")
		return
	}

	// 获取事务处理器
	tx, err := u.tracePool.Begin()
	if err != nil {
		return
	}

	// 异常时Rollback
	defer func() {
		switch err {
		case nil:
			// 提交事务
			if err = tx.Commit(); err != nil {
				u.logger.Errorf("ModifyUserInfo Transaction Commit Error:%v", err)
				return
			}
		default:
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				u.logger.Errorf("ModifyUserInfo Rollback err:%v", rollbackErr)
			}
		}
	}()

	// 修改用户信息自定义属性

	if bRange.CustomAttr {
		// 获取用户自定义属性
		customAttrs, err := u.userDB.GetUserCustomAttr(info.ID)
		if err != nil {
			return err
		}
		if len(customAttrs) == 0 {
			// 添加用户自定义属性
			err = u.userDB.AddUserCustomAttr(newCtx, info.ID, info.CustomAttr, tx)
			if err != nil {
				return err
			}
		} else {
			// 合并用户自定义属性
			merged := u.deepMergeMap(customAttrs, info.CustomAttr)

			// 更新用户自定义属性
			err = u.userDB.UpdateUserCustomAttr(newCtx, info.ID, merged, tx)
			if err != nil {
				return err
			}
		}
	}
	return
}

// 递归合并map中的值
//
//nolint:gocritic
func (u *user) deepMergeMap(m1, m2 map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// 遍历 m1 的所有键值对
	for k, v1 := range m1 {
		// 如果 m2 中也存在该键
		if v2, ok := m2[k]; ok {
			switch v1.(type) {
			case map[string]interface{}:
				switch v2.(type) {
				case map[string]interface{}:
					// 如果 v1 和 v2 都是映射类型,则递归合并
					result[k] = u.deepMergeMap(v1.(map[string]interface{}), v2.(map[string]interface{}))
				default:
					// 如果 v1 是映射类型,但 v2 不是,则使用 v2 的值
					result[k] = v2
				}
			default:
				switch v2.(type) {
				case map[string]interface{}:
					// 如果 v2 是映射类型,但 v1 不是,则使用 v2 的值
					result[k] = v2
				default:
					// 如果 v1 和 v2 都不是映射类型,则使用 v2 的值
					result[k] = v2
				}
			}
		} else {
			// 如果 m2 中不存在该键,则使用 m1 中的值
			result[k] = v1
		}
	}

	// 遍历 m2 中剩余的键值对
	for k, v2 := range m2 {
		if _, ok := m1[k]; !ok {
			// 如果 m1 中不存在该键,则将 m2 中的键值对添加到结果中
			result[k] = v2
		}
	}

	return result
}

// GetPWDRetrievalMethodByAccount 根据用户账户获取密码找回信息
func (u *user) GetPWDRetrievalMethodByAccount(account string) (info interfaces.PwdRetrievalInfo, err error) {
	// 根据账号名获取用户信息
	var userInfo interfaces.UserDBInfo
	userInfo, err = u.userDB.GetUserInfoByAccount(account)
	if err != nil {
		return info, err
	}

	// 获取配置信息
	rg := make(map[interfaces.ConfigKey]bool)
	rg[interfaces.IDCardLogin] = true
	rg[interfaces.TelPwdRetrieval] = true
	rg[interfaces.EmailPwdRetrieval] = true
	curConfig, err := u.config.GetConfig(rg)
	if err != nil {
		return info, err
	}

	// 如果账号名不存在，判断身份证登录是否开启，如果开启，则根据身份证再次检查
	if userInfo.ID == "" {
		if curConfig.IDCardLogin {
			var desAccount string
			desAccount, err = encodeDes(account, PadNormal)
			if err != nil {
				return info, err
			}
			userInfo, err = u.userDB.GetUserInfoByIDCard(desAccount)
			if err != nil {
				return info, err
			}

			if userInfo.ID == "" {
				info.Status = interfaces.PRSInvalidAccount
				return
			}
		} else {
			info.Status = interfaces.PRSInvalidAccount
			return
		}
	}

	// 判断用户是否被禁用
	if userInfo.DisableStatus != interfaces.Enabled {
		info.Status = interfaces.PRSDisableUser
		return
	}

	// 判断密码找回功能是否开启
	if !curConfig.EmailPwdRetrieval && !curConfig.TelPwdRetrieval {
		info.Status = interfaces.PRSUnablePWDRetrieval
		return
	}

	// 判断是否是本地用户
	if userInfo.AuthType != interfaces.Local {
		info.Status = interfaces.PRSNonLocalUser
		return
	}

	// 判断用户是否被管控密码
	if userInfo.PWDControl {
		info.Status = interfaces.PRSEnablePwdControl
		return
	}

	info.Status = interfaces.PRSAvaliable
	info.ID = userInfo.ID
	if curConfig.EmailPwdRetrieval {
		info.BEmail = true
		info.Email = userInfo.Email
	}

	if curConfig.TelPwdRetrieval {
		info.BTelephone = true
		info.Telephone = userInfo.TelNumber
	}
	return
}

// checkModifyUserPWDAuthority 检查用户权限修改
//
//nolint:exhaustive
func (u *user) checkModifyUserPWDAuthority(visitor *interfaces.Visitor) (err error) {
	switch visitor.Type {
	case interfaces.RealName:
		// 如果是实名用户，检查用户角色
		err = checkUserRole(u.role, visitor.ID, []interfaces.Role{interfaces.SystemRoleSuperAdmin, interfaces.SystemRoleSecAdmin})
	case interfaces.App:
		// 如果是应用账户，检查账户权限
		err = checkAppPerm(u.orgPermAppDB, visitor.ID, interfaces.User, interfaces.Modify)
	default:
		err = rest.NewHTTPError("Unsupported user type", rest.Forbidden, nil)
	}

	return
}

// checkIncrementModifyUserInfoAuthority 检查用户权限修改
//
//nolint:exhaustive
func (u *user) checkIncrementModifyUserInfoAuthority(ctx context.Context, visitor *interfaces.Visitor) (err error) {
	switch visitor.Type {
	case interfaces.RealName:
		// 如果是实名用户，检查用户角色
		err = checkUserRole2(ctx, u.role, visitor.ID, []interfaces.Role{interfaces.SystemRoleSuperAdmin, interfaces.SystemRoleSecAdmin, interfaces.SystemRoleSysAdmin})
	case interfaces.App:
		// 如果是应用账户，检查账户权限
		err = checkAppPerm2(ctx, u.orgPermAppDB, visitor.ID, interfaces.User, interfaces.Modify)
	default:
		err = rest.NewHTTPErrorV2(rest.Forbidden, "Unsupported user type")
	}

	return
}

// generatePWD 根据明文获取加密密码
func (u *user) generatePWD(pwd string, config *interfaces.Config) (sha2Pwd, desPwd, ntlmPwd string, err error) {
	// ntlm加密
	ntlmPwd = encodeNtlm(pwd)

	// des加密
	if config.EnableDesPwd {
		desPwd, err = encodeDes(pwd, PKCS5Padding)
		if err != nil {
			return "", "", "", err
		}
	}

	// sha2加密
	sha2Pwd = encodeSha2(pwd)
	return
}

// checkModifyPwdParam 密码参数检查，返回明文
func (u *user) checkModifyPwdParam(userInfo *interfaces.UserBaseInfo, config *interfaces.Config) (out string, err error) {
	// 检查用户是否存在
	info, err := u.userDB.GetUserDBInfo([]string{userInfo.ID})
	if err != nil {
		return
	}
	if len(info) != 1 {
		err = rest.NewHTTPErrorV2(errors.NotFound, "user does not exist")
		return
	}

	userDBInfo := info[0]
	// 是否为本地认证用户，本地用户才能修改密码
	if userDBInfo.AuthType != interfaces.Local {
		err = rest.NewHTTPError("can not modify non local user password", rest.BadRequest, nil)
		return
	}

	// rsa 密码解密
	out, err = decodeRSA(userInfo.Password, RSA1024)
	if err != nil {
		err = rest.NewHTTPError(err.Error(), rest.BadRequest, nil)
		return
	}

	// 检测密码合法性
	bIsValid := u.checkPasswordValid(out, config)
	if !bIsValid {
		err = rest.NewHTTPError("invalid password", rest.BadRequest, nil)
		return "", err
	}

	return
}

// GetUserList 获取用户列表
func (u *user) GetUserList(ctx context.Context, userInfoRange interfaces.UserBaseInfoRange, direction interfaces.Direction,
	bHasMarker bool, createdStamp int64, userID string, limit int) (out []interfaces.UserBaseInfo, num int, hasNext bool, err error) {
	// trace
	u.trace.SetInternalSpanName("业务逻辑-获取用户列表")
	newCtx, span := u.trace.AddInternalTrace(ctx)
	defer func() { u.trace.TelemetrySpanEnd(span, err) }()

	// 获取用户列表，多获取一条用户信息判断是否存在下一条用户
	tempLimit := limit + 1
	userDBInfos, err := u.userDB.GetUserList(newCtx, direction, bHasMarker, createdStamp, userID, tempLimit)
	if err != nil {
		return
	}

	hasNext = len(userDBInfos) > limit
	if hasNext {
		userDBInfos = userDBInfos[:limit]
	}

	// 整理用户信息
	out = make([]interfaces.UserBaseInfo, 0)
	for i := range userDBInfos {
		out = append(out, u.handleUserBaseInfo(userInfoRange, &userDBInfos[i]))
	}

	// 获取所有用户数量
	num, err = u.userDB.GetAllUserCount(newCtx)
	if err != nil {
		return
	}

	return out, num, hasNext, nil
}

// handleUserBaseInfo 用户基础信息整理
//
//nolint:gocyclo
func (u *user) handleUserBaseInfo(info interfaces.UserBaseInfoRange, userDBInfo *interfaces.UserDBInfo) (out interfaces.UserBaseInfo) {
	out.ID = userDBInfo.ID

	if info.ShowCSFLevel {
		out.CSFLevel = userDBInfo.CSFLevel
	}

	if info.ShowCSFLevel2 {
		out.CSFLevel2 = userDBInfo.CSFLevel2
	}

	if info.ShowEnable {
		out.Enabled = userDBInfo.DisableStatus == interfaces.Enabled && userDBInfo.AutoDisableStatus == interfaces.AEnabled
	}

	if info.ShowPriority {
		out.Priority = userDBInfo.Priority
	}

	if info.ShowName {
		out.Name = userDBInfo.Name
	}

	if info.ShowAccount {
		out.Account = userDBInfo.Account
	}

	if info.ShowFrozen {
		out.Frozen = userDBInfo.Frozen
	}

	if info.ShowAuthenticated {
		out.Authenticated = userDBInfo.Authenticated
	}

	if info.ShowEmail {
		out.Email = userDBInfo.Email
	}

	if info.ShowTelNumber {
		out.TelNumber = userDBInfo.TelNumber
	}

	if info.ShowThirdAttr {
		out.ThirdAttr = userDBInfo.ThirdAttr
	}

	if info.ShowThirdID {
		out.ThirdID = userDBInfo.ThirdID
	}

	if info.ShowAuthType {
		out.AuthType = userDBInfo.AuthType
	}

	if info.ShowPwdErrCnt {
		out.PwdErrCnt = userDBInfo.PWDErrCnt
	}

	if info.ShowPwdErrLastTime {
		out.PwdErrLastTime = userDBInfo.PWDErrLatestTime
	}

	if info.ShowLDAPType {
		out.LDAPType = userDBInfo.LDAPType
	}

	if info.ShowDomanPath {
		out.DomainPath = userDBInfo.DomainPath
	}

	if info.ShowOssID {
		out.OssID = userDBInfo.OssID
	}

	if info.ShowRemark {
		out.Remark = userDBInfo.Remark
	}

	if info.ShowCode {
		out.Code = userDBInfo.Code
	}

	if info.ShowPosition {
		out.Position = userDBInfo.Position
	}

	if info.ShowCreatedAt {
		out.CreatedAt = userDBInfo.CreatedAtTimeStamp
	}

	return out
}

// handleUserInfo 用户信息整理
func (u *user) handleUserInfo(info interfaces.UserBaseInfoRange, userDBInfo *interfaces.UserDBInfo, userRoleInfo map[interfaces.Role]bool,
	depInfos [][]interfaces.ObjectBaseInfo, paths []string, avatar string, groups []interfaces.GroupInfo, customAttr map[string]interface{},
	managers map[string]interfaces.NameInfo) (out interfaces.UserBaseInfo) {
	out.ParentDeps = make([][]interfaces.ObjectBaseInfo, 0)

	out = u.handleUserBaseInfo(info, userDBInfo)

	if info.ShowRoles {
		var roleArray []interfaces.Role
		for k := range userRoleInfo {
			roleArray = append(roleArray, k)
		}

		out.VecRoles = roleArray
	}

	if info.ShowParentDeps {
		out.ParentDeps = depInfos
	}

	if info.ShowParentDepPaths {
		out.ParentDepPaths = paths
	}

	if info.ShowAvatar {
		out.Avatar = avatar
	}

	if info.ShowGroups {
		out.Groups = groups
	}

	if info.ShowCustomAttr {
		out.CustomAttr = customAttr
	}

	if info.ShowManager {
		data, ok := managers[userDBInfo.ManagerID]
		if ok {
			out.Manager.ID = userDBInfo.ManagerID
			out.Manager.Name = data.Name
		} else {
			out.Manager.ID = ""
		}
	}

	return
}

// getUserParentDepsInfo 获取用户直属部门信息
func (u *user) getUserParentDepsInfo(userID string, ctx context.Context) (depInfos [][]interfaces.ObjectBaseInfo, paths []string, err error) {
	depInfos = make([][]interfaces.ObjectBaseInfo, 0)
	paths = make([]string, 0)
	// 获取用户直属部门信息
	var directDepInfos []interfaces.DepartmentDBInfo
	if ctx != nil {
		_, directDepInfos, err = u.userDB.GetDirectBelongDepartmentIDs2(ctx, userID)
	} else {
		_, directDepInfos, err = u.userDB.GetDirectBelongDepartmentIDs(userID)
	}
	if err != nil {
		return
	}

	// 获取每个直属部门的路径
	for i := range directDepInfos {
		var out []interfaces.ObjectBaseInfo
		var path string
		out, path, err = getParentDep(u.departmentDB, &directDepInfos[i], true, ctx)
		if err != nil {
			return
		}
		depInfos = append(depInfos, out)
		paths = append(paths, path)
	}

	return
}

// checkUserInRange
func (u *user) checkUserInRange(managerID string, role interfaces.Role, userIDs []string) (ret []bool, err error) {
	userDepIDs := make([][]string, len(userIDs))
	for index, v := range userIDs {
		// 获取用户的直属部门
		var directDepIDs []string
		directDepIDs, _, err = u.userDB.GetDirectBelongDepartmentIDs(v)
		if err != nil {
			return
		}
		userDepIDs[index] = directDepIDs
	}

	// 判断直属部门是否在特定角色管辖范围之内
	return checkDepInRange(u.userDB, u.departmentDB, managerID, role, userDepIDs)
}

// checkPasswordValid 检测密码有效性
func (u *user) checkPasswordValid(pwd string, config *interfaces.Config) (bIsValid bool) {
	// 根据是否是强密码进行检查
	if config.StrongPwdStatus {
		bIsValid = u.checkIsStrongPassword(pwd, config.StrongPwdLength)
	} else {
		bIsValid = u.checkIsValidPassword(pwd)
	}

	return
}

// checkIsValidPassword  非强密码检查
// 密码可以包含ASCII字符的任意组合，特殊字符为!@#$%\-_,. ， 且长度[6-100]
func (u *user) checkIsValidPassword(pwd string) bool {
	str := "^([\x20-\x7E]{6,100})$"
	reg := regexp.MustCompile(str)
	return reg.MatchString(pwd)
}

// checkIsStrongPassword 检测是否是强密码
// 密码长度由管理员控制, 最多100个字符，需同时包含大小写字母及数字， 特殊字符为!@#$%\-_,.
func (u *user) checkIsStrongPassword(pwd string, strongLen int) bool {
	// 弱密码检查
	result := u.checkIsValidPassword(pwd)
	if !result {
		return false
	}
	// 强密码长度判断
	if len(pwd) < strongLen {
		return false
	}

	// 是否同时包含大小写字母以及数字
	patternList := []string{`[0-9]+`, `[a-z]+`, `[A-Z]+`, "[\x20-\x2F\x3A-\x40\x5B-\x60\x7B-\x7E]+"}
	for _, pattern := range patternList {
		match, _ := regexp.MatchString(pattern, pwd)
		if !match {
			return false
		}
	}

	return true
}

// UserAuth 本地认证
func (u *user) UserAuth(id, plainPassword string) (result bool, reason interfaces.AuthFailedReason, err error) {
	// 根据 userId, 获取密码
	out, err := u.userDB.GetUserDBInfo([]string{id})
	if err != nil {
		return
	}

	// 用户不存在
	if len(out) == 0 {
		return
	}

	// 密码校验
	var encryptedPassword string
	if out[0].Password == "" {
		encryptedPassword = encodeSha2(plainPassword)
		result = strings.EqualFold(encryptedPassword, out[0].Sha2Password)
	} else {
		encryptedPassword, err = encodeMD5(plainPassword)
		if err != nil {
			return
		}
		result = strings.EqualFold(encryptedPassword, out[0].Password)
	}

	// 密码校验失败
	if !result {
		return false, interfaces.InvalidPassword, nil
	}

	// 密码策略检查
	rg := make(map[interfaces.ConfigKey]bool)
	rg[interfaces.StrongPWDStatus] = true
	rg[interfaces.StrongPWDLength] = true
	rg[interfaces.PWDExpireTime] = true
	curConfig, err := u.config.GetConfig(rg)
	if err != nil {
		return
	}
	rg2 := make(map[interfaces.ConfigKey]bool)
	rg2[interfaces.UserDefaultMd5PWD] = true
	rg2[interfaces.UserDefaultSha2PWD] = true
	curConfig2, err := u.config.GetConfigFromOption(rg2)
	if err != nil {
		return
	}

	// (1) 初始密码
	if encryptedPassword == curConfig2.UserDefaultSha2PWD || encryptedPassword == curConfig2.UserDefaultMd5PWD {
		return false, interfaces.InitialPassword, nil
	}

	// (2) 强密码
	if curConfig.StrongPwdStatus && !u.checkIsStrongPassword(plainPassword, curConfig.StrongPwdLength) {
		return false, interfaces.PasswordNotSafe, nil
	}

	// (3) 密码时效性
	if curConfig.PwdExpireTime != -1 {
		dayDelta := (common.Now().Unix() - out[0].PWDTimeStamp) / (60 * 60 * 24)
		if dayDelta >= curConfig.PwdExpireTime {
			if out[0].PWDControl {
				return false, interfaces.UnderControlPasswordExpire, nil
			}
			return false, interfaces.PasswordExpire, nil
		}
	}

	return
}

// UpdatePwdErrInfo 更新账户密码错误信息
func (u *user) UpdatePwdErrInfo(id string, pwdErrCnt int, pwdErrLastTime int64) (err error) {
	// 根据 userId, 获取账户信息
	out, err := u.userDB.GetUserDBInfo([]string{id})
	if err != nil {
		return
	}
	// 用户不存在
	if len(out) == 0 {
		err = rest.NewHTTPError("invalid type", rest.URINotExist, map[string]interface{}{"params": "user_id"})
		return
	}

	// 参数合法性校验
	if pwdErrCnt < 0 {
		err = rest.NewHTTPError("invalid type", rest.BadRequest, map[string]interface{}{"params": "pwd_err_cnt"})
		return
	}

	// 增加时间冗余5s，增加冗余原因如下
	// 正常场景：authentication登录时调用此接口，传入时间为authentication的pod时间，如果authentication和usermanagement服务时间一致，登录正常，如果authentication时间大于usermanagement的时间，则会登陆失败
	// 错误场景：通过NTP服务同步时间之后，可能会出现authentication服务时间大于usermanagement服务时间，authentication服务和usermanagement服务存在时间差在1s以内，增加冗余支持此场景避免登录异常
	if pwdErrLastTime < 0 || pwdErrLastTime > common.Now().Add(time.Second*time.Duration(nOffsetTime)).Unix() {
		err = rest.NewHTTPError("invalid type", rest.BadRequest, map[string]interface{}{"params": "pwd_err_last_time, time in the future?"})
		return
	}

	// 修改账户密码错误信息
	err = u.userDB.UpdatePwdErrInfo(id, pwdErrCnt, pwdErrLastTime)

	return
}

// CheckUserNameExistd 检查显示名是否存在
func (u *user) CheckUserNameExistd(ctx context.Context, name string) (result bool, err error) {
	// trace
	u.trace.SetInternalSpanName("业务逻辑-检查用户名是否存在")
	newCtx, span := u.trace.AddInternalTrace(ctx)
	defer func() { u.trace.TelemetrySpanEnd(span, err) }()

	if name == "" {
		return false, rest.NewHTTPErrorV2(rest.BadRequest, "invalid name")
	}

	userInfo, err := u.userDB.GetUserInfoByName(newCtx, name)
	if err != nil {
		return false, err
	}

	if userInfo.ID != "" {
		return true, nil
	}

	reservedName, err := u.reservedName.GetReservedName(name)
	if err != nil {
		return false, err
	}

	if reservedName.ID != "" {
		return true, nil
	}

	return false, nil
}

// 用户自动禁用
func (u *user) userExpiredAutoDisable(ctx context.Context) (err error) {
	// trace
	u.trace.SetInternalSpanName("业务逻辑-用户自动禁用")
	newCtx, span := u.trace.AddInternalTrace(ctx)
	defer func() { u.trace.TelemetrySpanEnd(span, err) }()

	// 获取事务处理器
	tx, err := u.pool.Begin()
	if err != nil {
		return
	}

	// 异常时Rollback
	defer func() {
		switch err {
		case nil:
			// 提交事务
			if err = tx.Commit(); err != nil {
				u.logger.Errorf("user auto disable transaction commit error: %v", err)
				return
			}

			// notify outbox推送线程
			u.ob.NotifyPushOutboxThread()
		default:
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				u.logger.Errorf("user auto disable transaction rollback error: %v", rollbackErr)
			}
		}
	}()

	// 获取锁
	err = u.userDB.GetLock(newCtx, interfaces.UserExpiredDisableLock, tx)
	if err != nil {
		return err
	}

	// 获取需要禁用的用户信息
	userInfos, err := u.userDB.GetExpiredNeedDisableUserInfos(newCtx, tx)
	if err != nil {
		return err
	}

	// 获取用户ids,并且排序，保证事务处理的顺序性
	userIDs := make([]string, 0)
	for k := range userInfos {
		userIDs = append(userIDs, userInfos[k].ID)
	}
	sort.Strings(userIDs)

	// 禁用用户
	err = u.userDB.SetAutoDisableStatus(newCtx, userIDs, interfaces.ExpiredDisabled, tx)
	if err != nil {
		return err
	}

	// 记录用户自动禁用事件
	for k := range userInfos {
		// 插入outbox 信息
		contentJSON := make(map[string]interface{})
		contentJSON["id"] = userInfos[k].ID
		contentJSON["display_name"] = userInfos[k].Name
		contentJSON["login_name"] = userInfos[k].Account

		err = u.ob.AddOutboxInfo(outboxUserExpiredAutoDisabled, contentJSON, tx)
		if err != nil {
			u.logger.Errorf("Add Outbox Info err:%v", err)
			return
		}
	}

	return nil
}

// userNotLoginAutoDisable
func (u *user) userNotLoginAutoDisable(ctx context.Context) (err error) {
	// trace
	u.trace.SetInternalSpanName("业务逻辑-用户长时间未登录自动禁用")
	newCtx, span := u.trace.AddInternalTrace(ctx)
	defer func() { u.trace.TelemetrySpanEnd(span, err) }()

	// 获取配置
	cfg, err := u.config.GetConfig(map[interfaces.ConfigKey]bool{
		interfaces.AutoDisable: true,
	})
	if err != nil {
		return err
	}

	// 如果配置未开启则跳过
	if !cfg.AutoDisableEnabled {
		return nil
	}

	// 获取事务处理器
	tx, err := u.pool.Begin()
	if err != nil {
		return
	}

	// 异常时Rollback
	defer func() {
		switch err {
		case nil:
			// 提交事务
			if err = tx.Commit(); err != nil {
				u.logger.Errorf("user not login auto disable transaction commit error: %v", err)
				return
			}

			// notify outbox推送线程
			u.ob.NotifyPushOutboxThread()
		default:
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				u.logger.Errorf("user not login auto disable transaction rollback error: %v", rollbackErr)
			}
		}
	}()

	// 获取锁
	err = u.userDB.GetLock(newCtx, interfaces.UserNotLoginDisableLock, tx)
	if err != nil {
		return err
	}

	// 获取需要禁用的用户
	userInfos, err := u.userDB.GetNotLoginNeedDisableUserInfos(newCtx, cfg.AutoDisableTime, tx)
	if err != nil {
		return err
	}

	// 获取用户ids,并且排序，保证事务处理的顺序性
	userIDs := make([]string, 0)
	for k := range userInfos {
		userIDs = append(userIDs, userInfos[k].ID)
	}
	sort.Strings(userIDs)

	// 禁用用户
	err = u.userDB.SetAutoDisableStatus(newCtx, userIDs, interfaces.NotLoginDisabled, tx)
	if err != nil {
		return err
	}

	// 记录用户自动禁用事件
	for k := range userInfos {
		// 插入outbox 信息
		contentJSON := make(map[string]interface{})
		contentJSON["id"] = userInfos[k].ID
		contentJSON["display_name"] = userInfos[k].Name
		contentJSON["login_name"] = userInfos[k].Account

		err = u.ob.AddOutboxInfo(outboxUserNotLoginAutoDisabled, contentJSON, tx)
		if err != nil {
			u.logger.Errorf("Add Outbox Info err:%v", err)
			return
		}
	}

	return nil
}

// 用户过期自动禁用
func (u *user) userAutoDisableThread() {
	for {
		u.logger.Infoln("**************** user long time not login disable start *****************")

		// 自动禁用过期用户
		ctx := context.Background()
		err := u.userNotLoginAutoDisable(ctx)
		if err != nil {
			u.logger.Errorf("user auto disable error: %v", err)
		}

		u.logger.Infoln("**************** user long time not login disable end *****************")

		u.logger.Infoln("**************** user expire disable start *****************")

		// 自动禁用过期用户
		err = u.userExpiredAutoDisable(ctx)
		if err != nil {
			u.logger.Errorf("user auto disable error: %v", err)
		}

		u.logger.Infoln("**************** user expire disable end *****************")

		// 每天的23：59：59之后5分钟内执行一次，增加抖动时间，减少锁竞争
		randomTime := time.Duration(rand.Intn(randomTimeRange)) * time.Second
		now := common.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.Local).Add(randomTime)
		time.Sleep(next.Sub(now))
	}
}
