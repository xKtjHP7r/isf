// Package logics department AnyShare 部门业务逻辑层
package logics

import (
	"context"
	"database/sql"
	"strings"
	"sync"

	"github.com/kweaver-ai/go-lib/observable"
	"github.com/kweaver-ai/go-lib/rest"
	"github.com/kweaver-ai/proton-rds-sdk-go/sqlx"
	"github.com/mitchellh/mapstructure"

	"UserManagement/common"
	"UserManagement/errors"
	"UserManagement/interfaces"
)

type department struct {
	db            interfaces.DBDepartment
	groupMemberDB interfaces.DBGroupMember
	userDB        interfaces.DBUser
	role          interfaces.LogicsRole
	orgPerm       interfaces.LogicsOrgPerm
	pool          *sqlx.DB
	logger        common.Logger
	eacpLog       interfaces.DrivenEacpLog
	ob            interfaces.LogicsOutbox
	messageBroker interfaces.DrivenMessageBroker
	event         interfaces.LogicsEvent
	i18n          *common.I18n
	trace         observable.Tracer
}

var (
	dOnce   sync.Once
	dLogics *department
)

// NewDepartment 创建部门对象
func NewDepartment() *department {
	dOnce.Do(func() {
		dLogics = &department{
			db:            dbDepartment,
			userDB:        dbUser,
			groupMemberDB: dbGroupMember,
			role:          NewRole(),
			pool:          dbPool,
			logger:        common.NewLogger(),
			eacpLog:       dnEacpLog,
			ob:            NewOutbox(OutboxBusinessDepart),
			messageBroker: dnMessageBroker,
			event:         NewEvent(),
			i18n: common.NewI18n(common.I18nMap{
				i18nIDObjectsInDepartDeleteNotContain: {
					interfaces.SimplifiedChinese:  "用户无法删除自己所在的部门",
					interfaces.TraditionalChinese: "使用者無法刪除自己所在的部門",
					interfaces.AmericanEnglish:    "The user can't delete his department.",
				},
				i18nIDObjectsInDepartNotFound: {
					interfaces.SimplifiedChinese:  "部门不存在",
					interfaces.TraditionalChinese: "部門不存在",
					interfaces.AmericanEnglish:    "This department does not exist",
				},
			}),
			trace:   common.SvcARTrace,
			orgPerm: NewOrgPerm(),
		}

		dLogics.ob.RegisterHandlers(outboxDepartDeleted, dLogics.sendDepartDeleted)
		dLogics.ob.RegisterHandlers(outboxOrgManagerChange, dLogics.sendOrgManagerChanged)
		dLogics.ob.RegisterHandlers(outboxTopDepartDeletedLog, dLogics.sendDepartDeletedAuditLog)

		dLogics.event.RegisterDeptDeleted(dLogics.DeleteOrgManagerRelationByDepartID)
		dLogics.event.RegisterDeptDeleted(dLogics.DeleteOrgAuditRelationByDepartID)

		dLogics.event.RegisterUserDeleted(dLogics.DeleteDepartManager)

		// efast逻辑，文档自动清理策略
		dLogics.event.RegisterDeptDeleted(dLogics.DeleteDocAutoCleanStrategy)
	})

	return dLogics
}

func (d *department) convertDepartmentName(deptIDs []string, bStrict bool) ([]interfaces.NameInfo, []string, error) {
	infoArray := make([]interfaces.NameInfo, 0)
	if len(deptIDs) == 0 {
		return infoArray, nil, nil
	}

	// 去掉重复id
	copyIDs := make([]string, len(deptIDs))
	copy(copyIDs, deptIDs)
	RemoveDuplicatStrs(&copyIDs)

	// 获取部门显示名
	tempInfo, tmpIDs, err := d.db.GetDepartmentName(copyIDs)
	if err != nil {
		return nil, nil, err
	}

	// 有部门不存在
	if bStrict && len(copyIDs) != len(tmpIDs) {
		// 获取不存在的部门id
		notExistIDs := Difference(copyIDs, tmpIDs)
		return nil, notExistIDs, nil
	}

	infoArray = append(infoArray, tempInfo...)
	return infoArray, nil, nil
}

// ConvertDepartmentName 根据departmentid批量获取部门名称
func (d *department) ConvertDepartmentName(visitor *interfaces.Visitor, deptIDs []string, bStrict bool) ([]interfaces.NameInfo, error) {
	infoArray, notExistIDs, err := d.convertDepartmentName(deptIDs, bStrict)
	if err != nil {
		return nil, err
	}

	// 如果严格模式， 且有部门不存在，则返回错误
	if bStrict && notExistIDs != nil {
		err = rest.NewHTTPErrorV2(errors.DepartmentNotFound,
			d.i18n.Load(i18nIDObjectsInDepartNotFound, visitor.Language),
			rest.SetDetail(map[string]interface{}{"ids": notExistIDs}),
			rest.SetCodeStr(errors.StrBadRequestDepartmentNotFound))
		return nil, err
	}

	return infoArray, nil
}

// GetDepartEmails 根据departmentid批量获取部门邮箱
func (d *department) GetDepartEmails(visitor *interfaces.Visitor, deptIDs []string) ([]interfaces.EmailInfo, error) {
	infoArray := make([]interfaces.EmailInfo, 0)
	if len(deptIDs) == 0 {
		return infoArray, nil
	}

	// 去掉重复id
	copyIDs := make([]string, len(deptIDs))
	copy(copyIDs, deptIDs)
	RemoveDuplicatStrs(&copyIDs)

	// 获取部门显示名
	tempInfo, err := d.db.GetDepartmentInfo(copyIDs, false, 0, 0)
	if err != nil {
		return nil, err
	}

	// 有部门不存在
	IDLen := len(tempInfo)
	if len(copyIDs) != IDLen {
		tmpIDs := make([]string, 0, IDLen)
		for k := range tempInfo {
			tmpIDs = append(tmpIDs, tempInfo[k].ID)
		}

		// 获取不存在的部门id
		notExistIDs := Difference(copyIDs, tmpIDs)
		err = rest.NewHTTPErrorV2(errors.DepartmentNotFound,
			d.i18n.Load(i18nIDObjectsInDepartNotFound, visitor.Language),
			rest.SetDetail(map[string]interface{}{"ids": notExistIDs}),
			rest.SetCodeStr(errors.StrBadRequestDepartmentNotFound))
		return nil, err
	}

	// 类型转换
	for k := range tempInfo {
		infoArray = append(infoArray, interfaces.EmailInfo{ID: tempInfo[k].ID, Email: tempInfo[k].Email})
	}

	return infoArray, nil
}

// GetAllChildDeparmentIDs 根据departmentid批量获取子部门ID
func (d *department) GetAllChildDeparmentIDs(deptIDs []string) (outInfo []string, err error) {
	return getAllChildDeparmentIDs(d.db, deptIDs, nil)
}

// GetAccessorIDsOfDepartment 获取指定部门的访问令牌
func (d *department) GetAccessorIDsOfDepartment(depID string) (accessorIDs []string, err error) {
	accessorIDs = make([]string, 0)
	// 判断部门是否存在
	info, err := d.db.GetDepartmentInfo([]string{depID}, false, -1, -1)
	if err != nil {
		return accessorIDs, err
	}
	if len(info) != 1 {
		err = rest.NewHTTPErrorV2(errors.NotFound, "department does not exist")
		return accessorIDs, err
	}

	// 获取部门所有的父部门
	deptIDs := strings.Split(info[0].Path, "/")

	// 获取包含用户和部门的所有用户组
	accessorIDs = append(accessorIDs, deptIDs...)
	groupIDs, _, err := d.groupMemberDB.GetMembersBelongGroupIDs(accessorIDs)
	if err != nil {
		return nil, err
	}

	// 整理令牌
	accessorIDs = append(accessorIDs, groupIDs...)

	return
}

// GetDepartMemberIDs 获取指定部门的成员ID（直属成员）
func (d *department) GetDepartMemberIDs(depID string) (info interfaces.DepartMemberID, err error) {
	// 检测部门是否存在
	_, outInfo, err := d.db.GetDepartmentName([]string{depID})
	if err != nil {
		return info, err
	}
	if len(outInfo) != 1 {
		err = rest.NewHTTPErrorV2(errors.NotFound, "department does not exist")
		return info, err
	}

	// 获取指定部门的子部门（直属部门）
	outDepIDs, _, err := d.db.GetChildDepartmentIDs([]string{depID})
	if err != nil {
		return info, err
	}
	info.DepartIDs = outDepIDs

	// 获取指定部门的子用户（直属用户）
	outUserIds, _, err := d.db.GetChildUserIDs([]string{depID})
	if err != nil {
		return info, err
	}
	info.UserIDs = outUserIds
	return info, nil
}

// GetAllDepartUserIDs 获取指定部门的用户ID（所有用户）
func (d *department) GetAllDepartUserIDs(depID string, bShowAllUser bool) (info []string, err error) {
	// 检测部门是否存在
	outInfo, err := d.db.GetDepartmentInfo([]string{depID}, false, 0, 0)
	if err != nil {
		return info, err
	}
	if len(outInfo) != 1 {
		err = rest.NewHTTPErrorV2(errors.NotFound, "department does not exist")
		return info, err
	}

	// 根据部门path获取部门下所有的子用户
	info, err = d.db.GetAllSubUserIDsByDepartPath(outInfo[0].Path)
	if err != nil {
		return info, err
	}

	// 去重
	RemoveDuplicatStrs(&info)

	// 如果不显示被禁用用户，则筛选
	if !bShowAllUser {
		// 获取用户信息，判断用户是否被禁用
		userInfos, err := d.userDB.GetUserDBInfo(info)
		if err != nil {
			return info, err
		}

		userIDs := make([]string, 0)
		for i := 0; i < len(userInfos); i++ {
			if userInfos[i].DisableStatus == interfaces.Enabled && userInfos[i].AutoDisableStatus == interfaces.AEnabled {
				userIDs = append(userIDs, userInfos[i].ID)
			}
		}

		info = userIDs
	}

	return
}

// GetAllDepartUserInfos 根据depID获取部门所有子成员基本信息
func (d *department) GetAllDepartUserInfos(depID string) (infos []interfaces.UserBaseInfo, err error) {
	// 检测部门是否存在
	outInfo, err := d.db.GetDepartmentInfo([]string{depID}, false, 0, 0)
	if err != nil {
		return infos, err
	}
	if len(outInfo) != 1 {
		err = rest.NewHTTPErrorV2(errors.NotFound, "department does not exist")
		return infos, err
	}

	// 根据部门path获取部门下所有的子用户
	var outInfos []interfaces.UserBaseInfo
	outInfos, err = d.db.GetAllSubUserInfosByDepartPath(outInfo[0].Path)
	if err != nil {
		return infos, err
	}

	// 去重
	infos = make([]interfaces.UserBaseInfo, 0, len(outInfos))
	allUserIDs := make(map[string]bool, len(outInfos))
	for k := range outInfos {
		if _, ok := allUserIDs[outInfos[k].ID]; ok {
			continue
		}

		infos = append(infos, outInfos[k])
		allUserIDs[outInfos[k].ID] = true
	}

	return
}

// GetDepartMemberInfo 获取指定部门的成员信息（直属成员）
func (d *department) GetDepartMemberInfo(visitor *interfaces.Visitor, depID string, info interfaces.OrgShowPageInfo) (depInfo []interfaces.DepartInfo,
	depNum int, userInfo []interfaces.ObjectBaseInfo, userNum int, err error) {
	depInfo = make([]interfaces.DepartInfo, 0)
	userInfo = make([]interfaces.ObjectBaseInfo, 0)
	// 检测调用者是否拥有指定角色
	var mapRoles map[interfaces.Role]bool
	mapRoles, err = getRolesByUserID(d.role, visitor.ID)
	if err != nil {
		return
	}
	if !mapRoles[info.Role] {
		err = rest.NewHTTPError("this user do not has this role", rest.BadRequest, nil)
		return
	}

	// 是否是显示根部门
	if depID == interfaces.RootDepartmentParentID {
		depInfo, depNum, err = d.getSupervisoryRootDeps(visitor.ID, info)
	} else {
		depInfo, depNum, userInfo, userNum, err = d.getDepartMemberInfoByID(visitor.ID, depID, info)
	}
	return
}

// getSupervisoryRootDeps 获取管理的所有根部门
func (d *department) getSupervisoryRootDeps(managerID string, info interfaces.OrgShowPageInfo) (outData []interfaces.DepartInfo, num int, err error) {
	if !info.BShowDeparts {
		return
	}

	// 获取限制的范围
	var depIDs []string
	var bNoScope, result bool
	switch info.Role {
	case interfaces.SystemRoleSuperAdmin, interfaces.SystemRoleSysAdmin, interfaces.SystemRoleAuditAdmin, interfaces.SystemRoleSecAdmin:
		bNoScope = true
	case interfaces.SystemRoleOrgAudit:
		depIDs, err = d.userDB.GetOrgAduitDepartInfo(managerID)
	case interfaces.SystemRoleOrgManager:
		depIDs, err = d.userDB.GetOrgManagerDepartInfo(managerID)
	case interfaces.SystemRoleNormalUser:
		ctx := context.Background()
		result, err = d.orgPerm.CheckPerms(ctx, managerID, interfaces.Department, interfaces.OPRead)
		if !result {
			return
		}
		bNoScope = true
	default:
		return
	}

	if err != nil {
		return
	}

	// 部门的信息获取和排序
	var allDeps []interfaces.DepartmentDBInfo
	var allDepIDs []string
	allDeps, allDepIDs, num, err = d.getAllRootDepart(bNoScope, depIDs, info)
	if err != nil {
		return
	}

	// 获取部门信息的子部门和子用户是否存在信息
	var subUsers map[string][]string
	var subDeps map[string][]string
	subUsers, subDeps, err = d.getSubUserAndDepart(allDepIDs, info)
	if err != nil {
		return
	}

	// 获取所有管理员信息
	mapManagerInfos := make(map[string]interfaces.NameInfo)
	if info.BShowDepartManager {
		managerIDs := make([]string, 0, len(allDeps))
		for k := range allDeps {
			managerIDs = append(managerIDs, allDeps[k].ManagerID)
		}

		var managerInfos []interfaces.UserDBInfo
		managerInfos, err = d.userDB.GetUserDBInfo(managerIDs)
		if err != nil {
			return
		}

		for k := range managerInfos {
			mapManagerInfos[managerInfos[k].ID] = interfaces.NameInfo{
				ID:   managerInfos[k].ID,
				Name: managerInfos[k].Name,
			}
		}
	}

	// 用户信息整理
	for k := range allDeps {
		tempDep := interfaces.DepartInfo{
			ID:            allDeps[k].ID,
			Name:          allDeps[k].Name,
			IsRoot:        allDeps[k].IsRoot > 0,
			BUserExistd:   len(subUsers[allDeps[k].ID]) > 0,
			BDepartExistd: len(subDeps[allDeps[k].ID]) > 0,
			Code:          allDeps[k].Code,
			Enabled:       allDeps[k].Status,
			Remark:        allDeps[k].Remark,
			Email:         allDeps[k].Email,
			ParentDeps:    []interfaces.ObjectBaseInfo{},
		}

		if _, ok := mapManagerInfos[allDeps[k].ManagerID]; ok {
			tempDep.Manager = mapManagerInfos[allDeps[k].ManagerID]
		}

		outData = append(outData, tempDep)
	}
	return
}

func (d *department) getSubUserAndDepart(depIDs []string, info interfaces.OrgShowPageInfo) (subUsers, subDeps map[string][]string, err error) {
	// 获取部门信息的子部门和子用户是否存在信息
	if info.BShowSubUser {
		_, subUsers, err = d.db.GetChildUserIDs(depIDs)
		if err != nil {
			return
		}
	}

	if info.BShowSubDepart {
		_, subDeps, err = d.db.GetChildDepartmentIDs(depIDs)
		if err != nil {
			return
		}
	}

	return
}

// getTopDepartments 获取部门数组内所有最上层的部门
func (d *department) filterTopDepartments(depIDs []string) (out []string, err error) {
	// 获取所有部门的path
	infos, err := d.db.GetDepartmentInfo(depIDs, false, -1, -1)
	if err != nil {
		return
	}

	// 判断其他部门是否在自己路径上
	for k := range infos {
		var bIsSub bool
		for k1 := range infos {
			if strings.Contains(infos[k].Path, infos[k1].Path) && infos[k1].ID != infos[k].ID {
				bIsSub = true
				break
			}
		}

		if !bIsSub {
			out = append(out, infos[k].ID)
		}
	}
	return
}

// getDepartMemberInfoByID 获取指定部门的成员信息（直属成员）
func (d *department) getDepartMemberInfoByID(managerID, depID string, info interfaces.OrgShowPageInfo) (depInfo []interfaces.DepartInfo,
	depNum int, userInfo []interfaces.ObjectBaseInfo, userNum int, err error) {
	// 判断部门是否存在
	outInfo, err := d.db.GetDepartmentInfo([]string{depID}, false, 0, 0)
	if err != nil {
		return
	}
	if len(outInfo) != 1 {
		err = rest.NewHTTPErrorV2(errors.NotFound, "department does not exist")
		return depInfo, depNum, userInfo, userNum, err
	}

	// 检测部门是否在用户的管辖范围之内
	bCanShowUser, bCanShowDepart, err := d.checkNormalUserAuth(managerID, info.Role, depID)
	if err != nil {
		return
	}

	// 获取指定部门的子部门信息（直属部门）
	if info.BShowDeparts && bCanShowDepart {
		depInfo, depNum, err = d.getSubDepartmentInfo(depID, info, outInfo[0].Path)
		if err != nil {
			return
		}
	}

	// 获取指定部门的子用户（直属用户）
	if info.BShowUsers && bCanShowUser {
		userInfo, userNum, err = d.getSubUsersInfo(depID, info.Offset, info.Limit)
		if err != nil {
			return
		}
	}
	return
}

// getSubDepartmentInfo 获取部门子部门信息
func (d *department) getSubDepartmentInfo(depID string, info interfaces.OrgShowPageInfo, path string) (out []interfaces.DepartInfo, depNum int, err error) {
	// 获取子部门信息
	var outDepInfos []interfaces.DepartmentDBInfo
	outDepInfos, err = d.db.GetSubDepartmentInfos(depID, true, info.Offset, info.Limit)
	if err != nil {
		return
	}
	allDepIDs := make([]string, 0)
	var allManagerIDs []string
	for k1 := range outDepInfos {
		allDepIDs = append(allDepIDs, outDepInfos[k1].ID)
		allManagerIDs = append(allManagerIDs, outDepInfos[k1].ManagerID)
	}

	// 获取子部门数量
	var tempData []interfaces.DepartmentDBInfo
	tempData, err = d.db.GetSubDepartmentInfos(depID, false, 0, 0)
	if err != nil {
		return
	}
	depNum = len(tempData)

	// 如果子部门为空，则直接返回
	if len(outDepInfos) == 0 {
		return nil, depNum, nil
	}

	// 更新子用户和子部门信息
	var subDeparts map[string][]string
	var subUsers map[string][]string
	if info.BShowSubDepart {
		_, subDeparts, err = d.db.GetChildDepartmentIDs(allDepIDs)
		if err != nil {
			return
		}
	}

	if info.BShowSubUser {
		_, subUsers, err = d.db.GetChildUserIDs(allDepIDs)
		if err != nil {
			return
		}
	}

	// 获取父部门信息
	parentDepInfos := make([]interfaces.ObjectBaseInfo, 0)
	if info.BShowDepartParentDeps {
		// 获取父部门信息
		parentIDs := strings.Split(path, "/")

		// 获取父部门信息
		var parentDepNameInfos []interfaces.DepartmentDBInfo
		parentDepNameInfos, err = d.db.GetDepartmentInfo(parentIDs, false, 0, 0)
		if err != nil {
			return
		}
		mapParentDepNameInfos := make(map[string]interfaces.DepartmentDBInfo)
		for k := range parentDepNameInfos {
			mapParentDepNameInfos[parentDepNameInfos[k].ID] = parentDepNameInfos[k]
		}

		for k := range parentIDs {
			tempDep := interfaces.ObjectBaseInfo{
				ID:   parentIDs[k],
				Name: mapParentDepNameInfos[parentIDs[k]].Name,
				Type: "department",
				Code: mapParentDepNameInfos[parentIDs[k]].Code,
			}
			parentDepInfos = append(parentDepInfos, tempDep)
		}
	}

	// 获取部门管理员信息
	mapManagerInfos := make(map[string]interfaces.NameInfo)
	if info.BShowDepartManager {
		var userInfos []interfaces.UserDBInfo
		userInfos, err = d.userDB.GetUserDBInfo(allManagerIDs)
		if err != nil {
			return
		}

		for k := range userInfos {
			mapManagerInfos[userInfos[k].ID] = interfaces.NameInfo{ID: userInfos[k].ID, Name: userInfos[k].Name}
		}
	}

	// 数据整理
	for k := range outDepInfos {
		tempDep := interfaces.DepartInfo{
			ID:            outDepInfos[k].ID,
			Name:          outDepInfos[k].Name,
			IsRoot:        outDepInfos[k].IsRoot > 0,
			BUserExistd:   len(subUsers[outDepInfos[k].ID]) > 0,
			BDepartExistd: len(subDeparts[outDepInfos[k].ID]) > 0,
			Code:          outDepInfos[k].Code,
			Enabled:       outDepInfos[k].Status,
			Remark:        outDepInfos[k].Remark,
			Email:         outDepInfos[k].Email,
		}

		tempDep.ParentDeps = parentDepInfos
		if _, ok := mapManagerInfos[outDepInfos[k].ManagerID]; ok {
			tempDep.Manager = mapManagerInfos[outDepInfos[k].ManagerID]
		}

		out = append(out, tempDep)
	}

	return out, depNum, nil
}

// getSubUsersInfo 获取部门子用户信息
func (d *department) getSubUsersInfo(depID string, offset, limit int) (out []interfaces.ObjectBaseInfo, userNum int, err error) {
	var userInfos []interfaces.UserDBInfo
	out = make([]interfaces.ObjectBaseInfo, 0)
	// 获取子用户信息
	userInfos, err = d.db.GetSubUserInfos(depID, true, offset, limit)
	if err != nil {
		return
	}

	// 获取子用户长度
	var tempData []interfaces.UserDBInfo
	tempData, err = d.db.GetSubUserInfos(depID, false, 0, 0)
	userNum = len(tempData)
	if err != nil {
		return
	}

	// 数据整理
	for k := range userInfos {
		out = append(out, interfaces.ObjectBaseInfo{ID: userInfos[k].ID, Name: userInfos[k].Name, Type: "user"})
	}
	return
}

// SearchDepartsByKey 按照关键字搜索部门
func (d *department) SearchDepartsByKey(ctx context.Context, visitor *interfaces.Visitor, info interfaces.OrgShowPageInfo) (out []interfaces.SearchDepartInfo, num int, err error) {
	// trace
	d.trace.SetInternalSpanName("业务逻辑-在管辖范围内搜索部门")
	newCtx, span := d.trace.AddInternalTrace(ctx)
	defer func() { d.trace.TelemetrySpanEnd(span, err) }()

	out = make([]interfaces.SearchDepartInfo, 0)
	var departInfos []interfaces.DepartmentDBInfo
	var depIDs []string

	// 获取限制范围
	var bNoScope, result bool
	switch info.Role {
	case interfaces.SystemRoleSuperAdmin, interfaces.SystemRoleSysAdmin, interfaces.SystemRoleSecAdmin, interfaces.SystemRoleAuditAdmin:
		bNoScope = true
	case interfaces.SystemRoleOrgAudit:
		depIDs, err = d.userDB.GetOrgAduitDepartInfo(visitor.ID)
	case interfaces.SystemRoleOrgManager:
		depIDs, err = d.userDB.GetOrgManagerDepartInfo(visitor.ID)
	case interfaces.SystemRoleNormalUser:
		ctx := context.Background()
		result, err = d.orgPerm.CheckPerms(ctx, visitor.ID, interfaces.Department, interfaces.OPRead)
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

	// 根据部门ID获取所有限制范围内部门ID
	var allDepIDs []string
	allDepIDs, err = getAllChildDeparmentIDsByIDs(d.db, depIDs, newCtx)
	allDepIDs = append(allDepIDs, depIDs...)
	if err != nil {
		return
	}

	departInfos, err = d.db.SearchDepartsByKey(newCtx, false, bNoScope, allDepIDs, info.Keyword, info.Offset, info.Limit)
	if err != nil {
		return
	}

	var tempuserInfo []interfaces.DepartmentDBInfo
	tempuserInfo, err = d.db.SearchDepartsByKey(newCtx, true, bNoScope, allDepIDs, info.Keyword, info.Offset, info.Limit)
	if err != nil {
		return
	}
	num = len(tempuserInfo)

	// 获取部门父部门信息
	for i := range departInfos {
		var temp interfaces.SearchDepartInfo
		temp.ID = departInfos[i].ID
		temp.Name = departInfos[i].Name
		temp.Type = "department"
		_, temp.Path, err = getParentDep(d.db, &departInfos[i], false, nil)
		if err != nil {
			return
		}
		out = append(out, temp)
	}
	return
}

// GetDepartsInfo 获取部门信息
func (d *department) GetDepartsInfo(departmentIDs []string, scope interfaces.DepartInfoScope, bCheckID bool) (infos []interfaces.DepartInfo, err error) {
	// 检查id数量
	if len(departmentIDs) == 0 {
		return
	}

	// 检查id是否重复
	if bCheckID {
		nDepsNum := len(departmentIDs)
		RemoveDuplicatStrs(&departmentIDs)
		if nDepsNum != len(departmentIDs) {
			err = rest.NewHTTPError("departmentID is not unique", rest.BadRequest, nil)
			return
		}
	}

	// 获取部门信息
	outInfos, err := d.db.GetDepartmentInfo(departmentIDs, false, 0, -1)
	if err != nil {
		return
	}

	// 检查部门是否存在，并且获取部门路径上所有部门ID
	existDepartIDs := make([]string, 0)
	allParentDepIDs := make([]string, 0)
	for k := range outInfos {
		existDepartIDs = append(existDepartIDs, outInfos[k].ID)
		tempIDs := strings.Split(outInfos[k].Path, "/")
		allParentDepIDs = append(allParentDepIDs, tempIDs...)
	}

	if bCheckID {
		notExistIDs := Difference(departmentIDs, existDepartIDs)
		if len(notExistIDs) != 0 {
			err = rest.NewHTTPErrorV2(errors.NotFound, "department does not exist",
				rest.SetDetail(map[string]interface{}{"ids": notExistIDs}))
			return nil, err
		}
	}

	// 如果需要获取父部门路径，获取所有父部门名称和ID
	depNameInfos := make(map[string]string)
	if scope.BParentDeps {
		RemoveDuplicatStrs(&allParentDepIDs)
		nameInfos, _, err := d.db.GetDepartmentName(allParentDepIDs)
		if err != nil {
			return nil, err
		}
		for _, v := range nameInfos {
			depNameInfos[v.ID] = v.Name
		}
	}

	// 如果需要获取部门管理员，则获取信息
	managers := make(map[string][]interfaces.NameInfo)
	if scope.BManagers {
		outInfo, err := d.db.GetManagersOfDepartment(departmentIDs)
		if err != nil {
			return nil, err
		}
		for _, v := range outInfo {
			managers[v.DepartmentID] = v.Managers
		}
	}

	// 获取部门信息
	infos = make([]interfaces.DepartInfo, 0)
	for i := range outInfos {
		data := d.handleDepartInfo(&outInfos[i], scope, depNameInfos, managers)
		infos = append(infos, data)
	}

	return
}

// handleDepartInfo 处理部门信息
func (d *department) handleDepartInfo(dbInfo *interfaces.DepartmentDBInfo, scope interfaces.DepartInfoScope,
	depNameInfos map[string]string, managers map[string][]interfaces.NameInfo) (data interfaces.DepartInfo) {
	data.ID = dbInfo.ID
	if scope.BShowName {
		data.Name = dbInfo.Name
	}

	if scope.BParentDeps {
		data.ParentDeps = make([]interfaces.ObjectBaseInfo, 0)
		parentIDs := strings.Split(dbInfo.Path, "/")
		parentIDs = parentIDs[:len(parentIDs)-1]
		for _, v1 := range parentIDs {
			object := interfaces.ObjectBaseInfo{ID: v1, Name: depNameInfos[v1], Type: "department"}
			data.ParentDeps = append(data.ParentDeps, object)
		}
	}

	if scope.BManagers {
		data.Managers = managers[dbInfo.ID]
	}

	if scope.BManager {
		data.Manager.ID = dbInfo.ManagerID
	}

	if scope.BCode {
		data.Code = dbInfo.Code
	}

	if scope.BEnabled {
		data.Enabled = dbInfo.Status
	}

	if scope.BThirdID {
		data.ThirdID = dbInfo.ThirdID
	}

	return data
}

// DeleteOrgManagerRelationByDepartID 根据部门ID删除组织管理员管辖信息
func (d *department) DeleteOrgManagerRelationByDepartID(id string) (err error) {
	return d.db.DeleteOrgManagerRelationByDepartID(id)
}

// DeleteOrgAuditRelationByDepartID 根据部门ID删除组织审计员管辖信息
func (d *department) DeleteOrgAuditRelationByDepartID(id string) (err error) {
	return d.db.DeleteOrgAuditRelationByDepartID(id)
}

// DeleteDocAutoCleanStrategy 删除文档自动清理策略
func (d *department) DeleteDocAutoCleanStrategy(obj string) (err error) {
	return d.db.DeleteDocAutoCleanStrategy(obj)
}

// DeleteDepartManager 清理部门负责人数据
func (d *department) DeleteDepartManager(userID string) (err error) {
	return d.db.DeleteDepartManager(userID)
}

// GetDepartsInfoByLevel 根据层级获取部门信息
func (d *department) GetDepartsInfoByLevel(level int) (infos []interfaces.ObjectBaseInfo, err error) {
	// 检查level
	if level < 0 {
		err = rest.NewHTTPError("level is illegal", rest.BadRequest, nil)
		return
	}

	// 获取信息  根据f_path长度获取
	// 部门的f_path信息保存部门的完整ID路径，包括部门本身ID，比如部门A属于根部门B，则f_path:B.ID/A.ID
	// 部门的ID为长度为36的uuid，则根部门f_path为36，下属部门依次增加36+1（多出来的1为ID分隔符号/）
	nLen := 36 + level*37
	out, err := d.db.GetDepartmentByPathLength(nLen)
	if err != nil {
		return
	}

	infos = make([]interfaces.ObjectBaseInfo, 0)
	for k := range out {
		infos = append(infos, interfaces.ObjectBaseInfo{ID: out[k].ID, Name: out[k].Name, Type: "department", ThirdID: out[k].ThirdID})
	}
	return
}

// DeleteDepart 根据部门ID删除部门
func (d *department) DeleteDepart(visitor *interfaces.Visitor, id string) (err error) {
	// 判断部门是否存在
	departInfos, err := d.db.GetDepartmentInfo([]string{id}, false, 0, -1)
	if err != nil {
		return err
	}

	if len(departInfos) != 1 {
		return rest.NewHTTPError("this department do not exist", rest.URINotExist, nil)
	}

	// 获取判断用户是否有权限操作此部门，私有接口不判断
	if visitor != nil && visitor.ID != "" {
		err = d.checkDepartInScope(visitor, &departInfos[0])
		if err != nil {
			return err
		}
	}

	// 获取删除部门需要的所有信息
	// 需要被移动到未分配组的用户
	// 被移动到其他组织的用户
	// 所有的子部门信息（包括自己）
	// 所有相关的组织管理员
	needAddToUnDistributeIDs, needDeleteOUIDs, allDepIDs, allOrgManagerIDs, err := d.handleDeletedDepartInfo(departInfos[0].Path)
	if err != nil {
		return err
	}

	// 获取事务处理器
	tx, err := d.pool.Begin()
	if err != nil {
		return
	}

	// 异常时Rollback
	defer func() {
		switch err {
		case nil:
			// 提交事务
			if err = tx.Commit(); err != nil {
				d.logger.Errorf("DeleteDepart Transaction Commit Error:%v", err)
				return
			}

			// notify outbox推送线程
			d.ob.NotifyPushOutboxThread()
		default:
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				d.logger.Errorf("DeleteDepart Rollback err:%v", rollbackErr)
			}
		}
	}()

	// 删除部门的数据库信息，包含用户/部门关系，用户/组织关系，部门/部门关系，部门/组织关系，部门相关信息
	err = d.deleteDepartInfo(departInfos[0].Path, needAddToUnDistributeIDs, needDeleteOUIDs, allDepIDs, tx)
	if err != nil {
		return
	}

	// 发送部门被删除消息
	for _, v := range allDepIDs {
		contentJSON := make(map[string]interface{})
		contentJSON["id"] = v

		err = d.ob.AddOutboxInfo(outboxDepartDeleted, contentJSON, tx)
		if err != nil {
			d.logger.Errorf("outboxDepartDeleted Add Outbox Info err:%v", err)
			return
		}
	}

	// 发送组织管理员变更事件
	contentJSON := make(map[string]interface{})
	contentJSON["ids"] = allOrgManagerIDs
	err = d.ob.AddOutboxInfo(outboxOrgManagerChange, contentJSON, tx)
	if err != nil {
		d.logger.Errorf("outboxOrgManagerChange Add Outbox Info err:%v", err)
		return
	}

	// 发送审计日志
	if visitor != nil && visitor.ID != "" {
		logJSON := make(map[string]interface{})
		logJSON["visitor"] = *visitor
		logJSON["name"] = departInfos[0].Name
		logJSON["root"] = departInfos[0].IsRoot > 0
		err = d.ob.AddOutboxInfo(outboxTopDepartDeletedLog, logJSON, tx)
		if err != nil {
			d.logger.Errorf("outboxTopDepartDeletedLog Add Outbox Info err:%v", err)
			return
		}
	}
	return
}

// 记录审计日志
func (d *department) sendDepartDeletedAuditLog(content interface{}) (err error) {
	info := content.(map[string]interface{})
	v := interfaces.Visitor{}
	err = mapstructure.Decode(info["visitor"], &v)
	if err != nil {
		d.logger.Errorf("sendDepartDeletedAuditLog mapstructure.Decode err:%v", err)
		return
	}

	err = d.eacpLog.OpDeleteDepart(&v, info["name"].(string), info["root"].(bool))
	if err != nil {
		d.logger.Errorf("OpDeleteDepart err:%v", err)
	}
	return err
}

/*
**
deleteDepartInfo 删除部门相关信息
1、从部门下移出所有的用户：根据部门路径，删除t_user_department_relation表内的部门下所有的用户/部门关系
2、如果用户只属于被删除部门，则用户需要被移动到未分配组：添加特定用户到未分配组的用户/部门关系
3、如果部门被移动之后不属于此组织了，则需要删除用户的用户/组织关系：删除t_ou_user内的用户/组织关系
4、删除被删除部门信息：根据路径在t_department内删除被删除部门下所有部门记录
5、删除被删除部门/部门关系：根据部门ID，删除t_department_relation表内部门记录
6、删除被删除部门/组织关系：根据部门ID，删除t_ou_department表内部门信息
**
*/
func (d *department) deleteDepartInfo(path string, needAddToUnDistributeIDs, needDeleteOUIDs, allDepIDs []string, tx *sql.Tx) (err error) {
	// 从部门下移出所有的用户
	err = d.db.DeleteUserDepartRelationByPath(path, tx)
	if err != nil {
		return
	}

	// 如果用户只属于被删除部门，则用户需要被移动到未分配组
	err = d.db.AddUserToDepart(needAddToUnDistributeIDs, "-1", "-1", tx)
	if err != nil {
		return
	}

	// 如果部门被移动之后不属于此组织了，则需要删除用户的用户/组织关系
	err = d.db.DeleteUserOURelation(needDeleteOUIDs, d.getRootByPath(path), tx)
	if err != nil {
		return
	}

	// 删除被删除部门信息
	err = d.db.DeleteDepartByPath(path, tx)
	if err != nil {
		return
	}

	// 删除部门下所有部门的关系
	err = d.db.DeleteDepartRelations(allDepIDs, tx)
	if err != nil {
		return
	}

	// 删除部门下所有部门的组织关系
	err = d.db.DeleteDepartOURelations(allDepIDs, tx)
	return
}

// getRootByPath 获取部门组织id
func (d *department) getRootByPath(path string) (root string) {
	parts := strings.Split(path, "/")
	return parts[0]
}

/*
**
handleDeletedDepartInfo 处理被删除部门内用户的信息
1、获取部门下所有的用户ID
2、获取需要被移动到未分配组的用户ID：如果用户只属于被删除部门，则需要被移动到未分配组
3、获取部门被删除之后，不属于此组织的用户：如果用户存在其他部门中，并且这些部门不属于被删除部门的组织，那么这些用户与被删除部门的组织关系需要被删除
4、获取所有部门ID
**
*/
func (d *department) handleDeletedDepartInfo(path string) (needAddToUnDistributeIDs, needDeleteOUIDs, allSubDepIDs, allOrgManagerIDs []string, err error) {
	// 获取部门下所有用户
	var allUserIDs []string
	allUserIDs, err = d.db.GetAllSubUserIDsByDepartPath(path)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	allUserPaths, err := d.userDB.GetUsersPath(allUserIDs)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// 如果用户的部门路径不包含被删除的部门，则用户不用被移动到未分配组
	// 如果用户不用被移动到未分配组，并且所在的其他部门和被删除部门是同一个组织，则不用删除用户的组织关系
	mapNotNeedMove := make(map[string]bool)
	mapNotNeedDeteleOU := make(map[string]bool)
	for k, v := range allUserPaths {
		mapNotNeedMove[k] = false
		mapNotNeedDeteleOU[k] = false
		for _, v1 := range v {
			if !strings.Contains(v1, path) {
				mapNotNeedMove[k] = true

				if d.getRootByPath(v1) == d.getRootByPath(path) {
					mapNotNeedDeteleOU[k] = true
					break
				}
			}
		}
	}

	// 获取需要被加到为分配组的用户
	needAddToUnDistributeIDs = make([]string, 0)
	for k, v := range mapNotNeedMove {
		if !v {
			needAddToUnDistributeIDs = append(needAddToUnDistributeIDs, k)
		}
	}

	// 获取需要删除用户/组织关系的用户
	needDeleteOUIDs = make([]string, 0)
	for k, v := range mapNotNeedDeteleOU {
		if !v {
			needDeleteOUIDs = append(needDeleteOUIDs, k)
		}
	}

	// 获取部门下所有子部门,包含部门本身的ID和Path信息
	var subDeparts []interfaces.DepartmentDBInfo
	allDepartIDsMap := make(map[string]bool)
	subDeparts, err = d.db.GetAllSubDepartInfosByPath(path)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// 获取所有收到影响的部门，包括被删除部门的子部门和父部门ID
	allSubDepIDs = make([]string, 0, len(subDeparts))
	for k := range subDeparts {
		allSubDepIDs = append(allSubDepIDs, subDeparts[k].ID)

		// 获取受到影响的所有部门ID
		depIDs := strings.Split(subDeparts[k].Path, "/")
		for k1 := range depIDs {
			allDepartIDsMap[depIDs[k1]] = true
		}
	}

	// 获取所有受到影响的组织管理员
	allDepartsIDsSlice := make([]string, 0, len(allDepartIDsMap))
	for k := range allDepartIDsMap {
		allDepartsIDsSlice = append(allDepartsIDsSlice, k)
	}

	allOrgManagerIDs, err = d.db.GetAllOrgManagerIDsByDepartIDs(allDepartsIDsSlice)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return
}

// 判断用户是否有权限操作此部门checkDepartInScope
func (d *department) checkDepartInScope(visitor *interfaces.Visitor, departInfo *interfaces.DepartmentDBInfo) (err error) {
	// 获取用户角色
	var mapRoles map[interfaces.Role]bool
	mapRoles, err = getRolesByUserID(d.role, visitor.ID)
	if err != nil {
		return err
	}

	// 获取判断用户对此部门是否有权限
	if mapRoles[interfaces.SystemRoleSuperAdmin] || mapRoles[interfaces.SystemRoleSysAdmin] {
		// 如果是超级管理员或者系统管理员，直接删除
		return nil
	} else if mapRoles[interfaces.SystemRoleOrgManager] {
		// 如果此部门包含此组织管理员，则不允许删除
		var paths map[string][]string
		paths, err = d.userDB.GetUsersPath([]string{visitor.ID})
		if err != nil {
			return
		}
		for _, v := range paths[visitor.ID] {
			if v == departInfo.Path {
				err = rest.NewHTTPErrorV2(rest.Forbidden, d.i18n.Load(i18nIDObjectsInDepartDeleteNotContain, visitor.Language))
				return
			}
		}

		// 如果是组织管理员，则判断是否在组织管理员范围内
		// 获取管辖范围
		var depIDs []string
		depIDs, err = d.userDB.GetOrgManagerDepartInfo(visitor.ID)
		if err != nil {
			return
		}

		// 获取所有部门信息
		var rangeDepartInfos []interfaces.DepartmentDBInfo
		rangeDepartInfos, err = d.db.GetDepartmentInfo(depIDs, false, 0, -1)
		if err != nil {
			return err
		}

		// 如果当前部门的路径在管辖范围的路径内，并且当前部门并不是组织管理员被分配的部门，则可以删除
		var bContains bool
		for i := range rangeDepartInfos {
			if strings.Contains(departInfo.Path, rangeDepartInfos[i].Path) {
				bContains = true
				break
			}
		}

		if !bContains {
			err = rest.NewHTTPError("this user do not has the authority", errors.Forbidden, nil)
		}
	} else {
		err = rest.NewHTTPError("this user do not has the authority", errors.Forbidden, nil)
	}
	return
}

// sendDepartDeleted 发送部门被删除消息
func (d *department) sendDepartDeleted(content interface{}) error {
	info := content.(map[string]interface{})
	err := d.messageBroker.DepartDeleted(info["id"].(string))
	return err
}

// sendOrgManagerChanged 发送组织管理员变更事件
func (d *department) sendOrgManagerChanged(content interface{}) error {
	info := content.(map[string]interface{})
	data := info["ids"].([]interface{})

	ids := make([]string, 0)
	for _, v := range data {
		ids = append(ids, v.(string))
	}
	err := d.messageBroker.OrgManagerChanged(ids)
	return err
}

// getParentDep  获取部门父部门路径
func getParentDep(db interfaces.DBDepartment, deptInfo *interfaces.DepartmentDBInfo, bHasOwn bool, ctx context.Context) (out []interfaces.ObjectBaseInfo, path string, err error) {
	pathIDs := strings.Split(deptInfo.Path, "/")
	if !bHasOwn {
		pathIDs = pathIDs[:len(pathIDs)-1]
	}

	var pathDeparts []interfaces.DepartmentDBInfo
	if ctx != nil {
		pathDeparts, err = db.GetDepartmentInfo2(ctx, pathIDs, false, 0, 0)
	} else {
		pathDeparts, err = db.GetDepartmentInfo(pathIDs, false, 0, 0)
	}

	if err != nil {
		return
	}

	tempName := make(map[string]string)
	tempCode := make(map[string]string)
	for k := range pathDeparts {
		tempName[pathDeparts[k].ID] = pathDeparts[k].Name
		tempCode[pathDeparts[k].ID] = pathDeparts[k].Code
	}

	nLen := len(pathIDs)
	out = make([]interfaces.ObjectBaseInfo, nLen)
	for i, v := range pathIDs {
		if i == 0 {
			path = tempName[v]
		} else {
			path = path + "/" + tempName[v]
		}
		out[i] = interfaces.ObjectBaseInfo{ID: v, Name: tempName[v], Type: "department", Code: tempCode[v]}
	}
	return
}

// checkDepInRange 判断部门是否调用者特定角色的管辖范围
//
//nolint:exhaustive
func checkDepInRange(userDB interfaces.DBUser, departDB interfaces.DBDepartment, managerID string, roleID interfaces.Role, departIDs [][]string) (ret []bool, err error) {
	var depIDs []string
	ret = make([]bool, len(departIDs))

	var bNoScope bool
	switch roleID {
	case interfaces.SystemRoleSuperAdmin, interfaces.SystemRoleSysAdmin, interfaces.SystemRoleSecAdmin, interfaces.SystemRoleAuditAdmin:
		bNoScope = true
	case interfaces.SystemRoleOrgAudit:
		depIDs, err = userDB.GetOrgAduitDepartInfo(managerID)
	case interfaces.SystemRoleOrgManager:
		depIDs, err = userDB.GetOrgManagerDepartInfo(managerID)
	default:
		return
	}

	if bNoScope {
		for i := 0; i < len(ret); i++ {
			ret[i] = true
		}
		return ret, nil
	}

	if err != nil {
		return
	}

	// 根据部门ID获取所有子部门ID
	allDepIDs := depIDs
	tempIDs := depIDs
	for {
		if len(tempIDs) == 0 {
			break
		}

		var childIDs []string
		childIDs, _, err = departDB.GetChildDepartmentIDs(tempIDs)
		if err != nil {
			return
		}
		allDepIDs = append(allDepIDs, childIDs...)
		tempIDs = childIDs
	}

	// 去重
	RemoveDuplicatStrs(&allDepIDs)

	// 判断是否在范围内
	for index, v := range departIDs {
		out := Intersection(v, allDepIDs)
		if len(out) > 0 {
			ret[index] = true
		}
	}

	return
}

// getAllChildDeparmentIDs 根据departmentid批量获取子部门ID
func getAllChildDeparmentIDs(db interfaces.DBDepartment, deptIDs []string, ctx context.Context) (outInfo []string, err error) {
	// 获取所有的子部门
	tmpOutInfo := make(map[string]bool)

	// 获取所有部门的路径
	for {
		if len(deptIDs) == 0 {
			break
		}

		var err error
		var outDepIDs []string
		if ctx != nil {
			outDepIDs, _, err = db.GetChildDepartmentIDs2(ctx, deptIDs)
		} else {
			outDepIDs, _, err = db.GetChildDepartmentIDs(deptIDs)
		}

		if err != nil {
			return nil, err
		}

		// 过滤已经存在的部门
		var tmpDepIDs []string
		for _, v := range outDepIDs {
			if !tmpOutInfo[v] {
				tmpOutInfo[v] = true
				tmpDepIDs = append(tmpDepIDs, v)
			}
		}

		deptIDs = tmpDepIDs
	}

	// 获取最终的部门数组
	for key := range tmpOutInfo {
		outInfo = append(outInfo, key)
	}

	return
}

// getAllChildDeparmentIDsByIDs 根据departmentid批量获取子部门ID,支持context
func getAllChildDeparmentIDsByIDs(db interfaces.DBDepartment, deptIDs []string, ctx context.Context) (outInfo []string, err error) {
	// 获取所有的子部门
	tmpOutInfo := make(map[string]bool)

	// 获取所有部门的路径
	departInfos, err := db.GetDepartmentInfo2(ctx, deptIDs, false, 0, 0)
	if err != nil {
		return nil, err
	}

	// 获取所有部门的子部门
	for k := range departInfos {
		tempDeparts, err := db.GetAllSubDepartInfosByPath(departInfos[k].Path)
		if err != nil {
			return nil, err
		}

		for k1 := range tempDeparts {
			tmpOutInfo[tempDeparts[k1].ID] = true
		}
	}

	//
	outInfo = make([]string, 0, len(tmpOutInfo))
	for k := range tmpOutInfo {
		outInfo = append(outInfo, k)
	}

	return
}

func (d *department) getAllRootDepart(bNoScope bool, depIDs []string, info interfaces.OrgShowPageInfo) (allDeps []interfaces.DepartmentDBInfo, allDepIDs []string, num int, err error) {
	if bNoScope {
		// 获取所有的根部门
		allDeps, err = d.db.GetRootDeps(false, true, nil, info.Offset, info.Limit)
		if err != nil {
			return
		}

		for k := range allDeps {
			allDepIDs = append(allDepIDs, allDeps[k].ID)
		}

		var temp []interfaces.DepartmentDBInfo
		temp, err = d.db.GetRootDeps(true, true, nil, info.Offset, info.Limit)
		if err != nil {
			return
		}
		num = len(temp)
	} else {
		// 筛选部门范围内所有的最上层的部门
		allDepIDs, err = d.filterTopDepartments(depIDs)
		if err != nil {
			return
		}

		allDeps, err = d.db.GetDepartmentInfo(allDepIDs, true, info.Offset, info.Limit)
		if err != nil {
			return
		}
		num = len(allDepIDs)
	}
	return
}

// checkNormalUserAuth 检查特定用户在特定角色下对用户和部门是否有权限
func (d *department) checkNormalUserAuth(userID string, role interfaces.Role, departID string) (bShowUser, bShowDepart bool, err error) {
	ctx := context.Background()
	if role == interfaces.SystemRoleNormalUser {
		bShowUser, err = d.orgPerm.CheckPerms(ctx, userID, interfaces.User, interfaces.OPRead)
		if err != nil {
			return
		}

		bShowDepart, err = d.orgPerm.CheckPerms(ctx, userID, interfaces.Department, interfaces.OPRead)
		if err != nil {
			return
		}
	} else {
		var ret []bool
		temp := []string{departID}

		ret, err = checkDepInRange(d.userDB, d.db, userID, role, [][]string{temp})
		if err != nil {
			return
		}

		if ret[0] {
			bShowUser = true
			bShowDepart = true
		}
	}
	return
}

func (d *department) SearchDeparts(ctx context.Context, visitor *interfaces.Visitor, scope *interfaces.DepartInfoScope, ks *interfaces.DepartSearchKeyScope,
	k *interfaces.DepartSearchKey, r interfaces.Role) (out []interfaces.DepartInfo, num int, err error) {
	// trace
	d.trace.SetInternalSpanName("业务逻辑-搜索部门信息")
	newCtx, span := d.trace.AddInternalTrace(ctx)
	defer func() { d.trace.TelemetrySpanEnd(span, err) }()

	// 检测调用者是否拥有指定角色
	var mapRoles map[interfaces.Role]bool
	mapRoles, err = getRolesByUserID2(newCtx, d.role, visitor.ID)
	if err != nil {
		return
	}

	if !mapRoles[r] {
		return nil, 0, rest.NewHTTPErrorV2(rest.BadRequest, "this user do not has this role")
	}

	// 获取部门信息
	var infos []interfaces.DepartmentDBInfo
	infos, num, err = d.getDepartInfo(ctx, visitor, ks, k, r)
	if err != nil {
		return nil, 0, err
	}

	// 获取基本信息
	managerIDs := make([]string, 0)
	parentDepIDs := make([]string, 0)
	mapParentDepPath := make(map[string]string)
	for k := range infos {
		if infos[k].ManagerID != "" {
			managerIDs = append(managerIDs, infos[k].ManagerID)
		}
		mapParentDepPath[infos[k].ID] = infos[k].Path

		temp := strings.Split(infos[k].Path, "/")
		if len(temp) > 1 {
			parentDepIDs = append(parentDepIDs, temp[:len(temp)-1]...)
		}
	}

	// 获取管理员信息
	mapManagerName := make(map[string]string)
	if scope.BManager && len(managerIDs) > 0 {
		managerInfos, err := d.userDB.GetUserDBInfo(managerIDs)
		if err != nil {
			return nil, 0, err
		}

		for k := range managerInfos {
			mapManagerName[managerInfos[k].ID] = managerInfos[k].Name
		}
	}

	// 获取父部门信息
	mapParentDep := make(map[string]interfaces.ObjectBaseInfo)
	if scope.BParentDeps {
		parentDepInfos, err := d.db.GetDepartmentInfo(parentDepIDs, false, 0, -1)
		if err != nil {
			return nil, 0, err
		}

		for k := range parentDepInfos {
			mapParentDep[parentDepInfos[k].ID] = interfaces.ObjectBaseInfo{
				ID:   parentDepInfos[k].ID,
				Name: parentDepInfos[k].Name,
				Type: "department",
				Code: parentDepInfos[k].Code,
			}
		}
	}

	// 获取部门信息
	for k := range infos {
		temp := d.handleSearchDepartOutInfo(&infos[k], scope, mapManagerName, mapParentDep)
		out = append(out, temp)
	}

	return
}

// 获取部门信息
func (d *department) handleSearchDepartOutInfo(info *interfaces.DepartmentDBInfo, scope *interfaces.DepartInfoScope, mapManagerName map[string]string,
	mapParentDep map[string]interfaces.ObjectBaseInfo) (temp interfaces.DepartInfo) {
	temp.ID = info.ID
	temp.Name = info.Name
	temp.Code = info.Code
	temp.Enabled = info.Status
	temp.Remark = info.Remark
	temp.Email = info.Email

	if scope.BManager {
		_, ok := mapManagerName[info.ManagerID]
		if info.ManagerID != "" && ok {
			temp.Manager = interfaces.NameInfo{ID: info.ManagerID, Name: mapManagerName[info.ManagerID]}
		}
	}

	if scope.BParentDeps {
		temp.ParentDeps = make([]interfaces.ObjectBaseInfo, 0)
		allParentDepIDs := strings.Split(info.Path, "/")

		if len(allParentDepIDs) > 1 {
			tempDepIDs := allParentDepIDs[:len(allParentDepIDs)-1]
			for k1 := range tempDepIDs {
				if _, ok := mapParentDep[tempDepIDs[k1]]; ok {
					temp.ParentDeps = append(temp.ParentDeps, mapParentDep[tempDepIDs[k1]])
				}
			}
		}
	}

	return temp
}

// getDepartInfo 获取部门信息
func (d *department) getDepartInfo(ctx context.Context, visitor *interfaces.Visitor, ks *interfaces.DepartSearchKeyScope,
	k *interfaces.DepartSearchKey, r interfaces.Role) (infos []interfaces.DepartmentDBInfo, num int, err error) {
	if r == interfaces.SystemRoleSuperAdmin || r == interfaces.SystemRoleSysAdmin || r == interfaces.SystemRoleSecAdmin {
		infos, err = d.db.SearchDeparts(ctx, ks, k, nil)
		if err != nil {
			return nil, 0, err
		}

		num, err = d.db.SearchDepartsCount(ctx, ks, k, nil)
		if err != nil {
			return nil, 0, err
		}
	} else if r == interfaces.SystemRoleOrgManager {
		infos, num, err = d.orgManagerSearchDeparts(ctx, visitor, ks, k)
		if err != nil {
			return nil, 0, err
		}
	} else {
		return nil, 0, rest.NewHTTPErrorV2(rest.Forbidden, "this user has no authority")
	}

	return
}

func (d *department) orgManagerSearchDeparts(ctx context.Context, visitor *interfaces.Visitor, ks *interfaces.DepartSearchKeyScope,
	k *interfaces.DepartSearchKey) (out []interfaces.DepartmentDBInfo, num int, err error) {
	// 获取用户管理的部门
	depIDs, err := d.userDB.GetOrgManagerDepartInfo(visitor.ID)
	if err != nil {
		return nil, 0, err
	}

	// 如果没有负责的部门，直接返回
	if len(depIDs) == 0 {
		return nil, 0, nil
	}

	// 获取部门的信息，路径
	depInfos, err := d.db.GetDepartmentInfo2(ctx, depIDs, false, 0, -1)
	if err != nil {
		return nil, 0, err
	}

	// 如果有部门不存在，则记录错误信息
	if len(depInfos) != len(depIDs) {
		d.logger.Errorln("orgManagerSearchDeparts: some departments not found %s", depIDs)
	}

	// 获取管辖部门的下属所有部门
	limitDepartIDs := make([]string, 0)
	for k := range depInfos {
		var temp []interfaces.DepartmentDBInfo
		temp, err = d.db.GetAllSubDepartInfosByPath(depInfos[k].Path)
		if err != nil {
			return nil, 0, err
		}

		for k1 := range temp {
			limitDepartIDs = append(limitDepartIDs, temp[k1].ID)
		}
	}

	// 搜索
	out, err = d.db.SearchDeparts(ctx, ks, k, limitDepartIDs)
	if err != nil {
		return nil, 0, err
	}

	num, err = d.db.SearchDepartsCount(ctx, ks, k, limitDepartIDs)
	if err != nil {
		return nil, 0, err
	}

	return
}
