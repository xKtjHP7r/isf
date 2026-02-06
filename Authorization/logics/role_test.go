package logics

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"Authorization/common"
	"Authorization/interfaces"
	"Authorization/interfaces/mock"
)

const (
	roleTmpID = "role1"
)

func newRole(
	roleDB interfaces.DBRole,
	roleMemberDB interfaces.DBRoleMember,
	userMgnt interfaces.DrivenUserMgnt,
	logger common.Logger,
	event interfaces.LogicsEvent,
) *role {
	return &role{
		roleDB:       roleDB,
		roleMemberDB: roleMemberDB,
		userMgnt:     userMgnt,
		logger:       logger, // 可mock logger
		event:        event,  // 可mock event
		i18n: common.NewI18n(common.I18nMap{
			i18nRoleNotFound: {
				simplifiedChinese:  "角色不存在",
				traditionalChinese: "角色不存在",
				americanEnglish:    "The Role does not exist.",
			},
		}),
	}
}

func TestRole_DeleteRole(t *testing.T) {
	Convey("测试DeleteRole方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		roleDB := mock.NewMockDBRole(ctrl)
		roleMemberDB := mock.NewMockDBRoleMember(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		logger := common.NewLogger()
		event := mock.NewMockLogicsEvent(ctrl)
		r := newRole(roleDB, roleMemberDB, userMgnt, logger, event)

		ctx := context.Background()
		visitor := &interfaces.Visitor{ID: "user1", Type: interfaces.RealName}
		roleID := roleTmpID
		roleInfo := interfaces.RoleInfo{ID: roleID}

		Convey("权限检查失败", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), "user1").Return(nil, errors.New("权限错误"))
			err := r.DeleteRole(ctx, visitor, roleID)
			assert.Error(t, err)
		})

		Convey("角色不存在", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), "user1").Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			roleDB.EXPECT().GetRoleByID(gomock.Any(), roleID).Return(interfaces.RoleInfo{}, nil)
			err := r.DeleteRole(ctx, visitor, roleID)
			assert.NoError(t, err)
		})

		Convey("删除角色和成员成功", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), "user1").Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			roleDB.EXPECT().GetRoleByID(gomock.Any(), roleID).Return(roleInfo, nil)
			roleDB.EXPECT().DeleteRole(gomock.Any(), roleID).Return(nil)
			roleMemberDB.EXPECT().DeleteByRoleID(gomock.Any(), roleID).Return(nil)
			event.EXPECT().RoleDeleted(roleID).Return(nil)
			err := r.DeleteRole(ctx, visitor, roleID)
			assert.NoError(t, err)
		})
	})
}

func TestRole_ModifyRole(t *testing.T) {
	Convey("测试ModifyRole方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		roleDB := mock.NewMockDBRole(ctrl)
		roleMemberDB := mock.NewMockDBRoleMember(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		logger := common.NewLogger()
		event := mock.NewMockLogicsEvent(ctrl)
		r := newRole(roleDB, roleMemberDB, userMgnt, logger, event)

		ctx := context.Background()
		visitor := &interfaces.Visitor{ID: "user1", Type: interfaces.RealName}
		roleID := roleTmpID
		roleInfo := interfaces.RoleInfo{ID: roleID}

		Convey("权限检查失败", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), "user1").Return(nil, errors.New("权限错误"))
			err := r.ModifyRole(ctx, visitor, roleID, "newName", true, "desc", true, interfaces.ResourceTypeScopeInfo{}, false)
			assert.Error(t, err)
		})

		Convey("角色不存在", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), "user1").Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			roleDB.EXPECT().GetRoleByID(gomock.Any(), roleID).Return(interfaces.RoleInfo{}, nil)
			err := r.ModifyRole(ctx, visitor, roleID, "newName", true, "desc", true, interfaces.ResourceTypeScopeInfo{}, false)
			assert.Error(t, err)
		})

		Convey("修改角色成功", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), "user1").Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			roleDB.EXPECT().GetRoleByID(gomock.Any(), roleID).Return(roleInfo, nil)
			roleDB.EXPECT().GetRoleByName(gomock.Any(), "newName").Return(interfaces.RoleInfo{ID: roleID}, nil)
			roleDB.EXPECT().ModifyRole(gomock.Any(), roleID, "newName", true, "desc", true, interfaces.ResourceTypeScopeInfo{}, false).Return(nil)
			event.EXPECT().RoleNameModified(roleID, "newName").Return(nil)
			err := r.ModifyRole(ctx, visitor, roleID, "newName", true, "desc", true, interfaces.ResourceTypeScopeInfo{}, false)
			assert.NoError(t, err)
		})
	})
}

func TestRole_GetRoleByID(t *testing.T) {
	Convey("测试GetRoleByID方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		roleDB := mock.NewMockDBRole(ctrl)
		roleMemberDB := mock.NewMockDBRoleMember(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		logger := common.NewLogger()
		event := mock.NewMockLogicsEvent(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		r := newRole(roleDB, roleMemberDB, userMgnt, logger, event)
		r.resourceTypeHierarchy = resourceTypeHierarchy
		r.resourceType = resourceType

		ctx := context.Background()
		visitor := &interfaces.Visitor{ID: "user1", Type: interfaces.RealName}
		roleID := roleTmpID
		roleInfo := interfaces.RoleInfo{ID: roleID}
		param := interfaces.RoleInfoParam{
			ResourceTypeViewMode: interfaces.ResourceTypeViewModeFlat,
		}
		Convey("权限检查失败", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), "user1").Return(nil, errors.New("权限错误"))
			_, err := r.GetRoleByID(ctx, visitor, roleID, param)
			assert.Error(t, err)
		})

		Convey("角色不存在", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), "user1").Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			roleDB.EXPECT().GetRoleByID(gomock.Any(), roleID).Return(interfaces.RoleInfo{}, nil)
			_, err := r.GetRoleByID(ctx, visitor, roleID, param)
			assert.Error(t, err)
		})

		Convey("获取角色成功", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), "user1").Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			roleDB.EXPECT().GetRoleByID(gomock.Any(), roleID).Return(roleInfo, nil)
			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), gomock.Any()).Return(map[string]interfaces.ResourceType{"test": {ID: "test", Name: "test"}}, nil)
			info, err := r.GetRoleByID(ctx, visitor, roleID, param)
			assert.NoError(t, err)
			assert.Equal(t, roleID, info.ID)
		})
	})
}

func TestRole_GetRoles(t *testing.T) {
	Convey("测试GetRoles方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		roleDB := mock.NewMockDBRole(ctrl)
		roleMemberDB := mock.NewMockDBRoleMember(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		logger := common.NewLogger()
		event := mock.NewMockLogicsEvent(ctrl)
		r := newRole(roleDB, roleMemberDB, userMgnt, logger, event)

		ctx := context.Background()
		visitor := &interfaces.Visitor{ID: "user1", Type: interfaces.RealName}
		searchInfo := interfaces.RoleSearchInfo{Offset: 0, Limit: 10}
		roleInfo := interfaces.RoleInfo{ID: roleTmpID}

		Convey("权限检查失败", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), "user1").Return(nil, errors.New("权限错误"))
			_, _, err := r.GetRoles(ctx, visitor, searchInfo)
			assert.Error(t, err)
		})

		Convey("获取角色数量失败", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), "user1").Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			roleDB.EXPECT().GetRolesSum(gomock.Any(), searchInfo).Return(0, errors.New("db error"))
			_, _, err := r.GetRoles(ctx, visitor, searchInfo)
			assert.Error(t, err)
		})

		Convey("获取角色成功", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), "user1").Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			roleDB.EXPECT().GetRolesSum(gomock.Any(), searchInfo).Return(1, nil)
			roleDB.EXPECT().GetRoles(gomock.Any(), searchInfo).Return([]interfaces.RoleInfo{roleInfo}, nil)
			num, infos, err := r.GetRoles(ctx, visitor, searchInfo)
			assert.NoError(t, err)
			assert.Equal(t, 1, num)
			assert.Equal(t, 1, len(infos))
		})
	})
}

func TestRole_AddRoleMembers(t *testing.T) {
	Convey("测试AddRoleMembers方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		roleDB := mock.NewMockDBRole(ctrl)
		roleMemberDB := mock.NewMockDBRoleMember(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		logger := common.NewLogger()
		event := mock.NewMockLogicsEvent(ctrl)
		r := newRole(roleDB, roleMemberDB, userMgnt, logger, event)

		ctx := context.Background()
		visitor := &interfaces.Visitor{ID: "user1", Type: interfaces.RealName}
		roleID := roleTmpID
		roleInfo := interfaces.RoleInfo{ID: roleID}
		memberInfo := interfaces.RoleMemberInfo{ID: "member1", MemberType: interfaces.AccessorUser}
		infos := map[string]interfaces.RoleMemberInfo{"member1": memberInfo}

		Convey("权限检查失败", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), "user1").Return(nil, errors.New("权限错误"))
			err := r.AddRoleMembers(ctx, visitor, roleID, infos)
			assert.Error(t, err)
		})

		Convey("角色不存在", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), "user1").Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			roleDB.EXPECT().GetRoleByID(gomock.Any(), roleID).Return(interfaces.RoleInfo{}, nil)
			err := r.AddRoleMembers(ctx, visitor, roleID, infos)
			assert.Error(t, err)
		})

		Convey("添加成员成功", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), "user1").Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			roleDB.EXPECT().GetRoleByID(gomock.Any(), roleID).Return(roleInfo, nil)
			roleMemberDB.EXPECT().GetRoleMembersByRoleID(gomock.Any(), roleID).Return([]interfaces.RoleMemberInfo{}, nil)
			userMgnt.EXPECT().GetNameByAccessorIDs(gomock.Any(), gomock.Any()).Return(map[string]string{"member1": "name1"}, nil)
			roleMemberDB.EXPECT().AddRoleMembers(gomock.Any(), roleID, gomock.Any()).Return(nil)
			err := r.AddRoleMembers(ctx, visitor, roleID, infos)
			assert.NoError(t, err)
		})
	})
}

func TestRole_DeleteRoleMembers(t *testing.T) {
	Convey("测试DeleteRoleMembers方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		roleDB := mock.NewMockDBRole(ctrl)
		roleMemberDB := mock.NewMockDBRoleMember(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		logger := common.NewLogger()
		event := mock.NewMockLogicsEvent(ctrl)
		r := newRole(roleDB, roleMemberDB, userMgnt, logger, event)

		ctx := context.Background()
		visitor := &interfaces.Visitor{ID: "user1", Type: interfaces.RealName}
		roleID := roleTmpID
		roleInfo := interfaces.RoleInfo{ID: roleID}
		memberInfo := interfaces.RoleMemberInfo{ID: "member1", MemberType: interfaces.AccessorUser}
		infos := map[string]interfaces.RoleMemberInfo{"member1": memberInfo}

		Convey("权限检查失败", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), "user1").Return(nil, errors.New("权限错误"))
			err := r.DeleteRoleMembers(ctx, visitor, roleID, infos)
			assert.Error(t, err)
		})

		Convey("角色不存在", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), "user1").Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			roleDB.EXPECT().GetRoleByID(gomock.Any(), roleID).Return(interfaces.RoleInfo{}, nil)
			err := r.DeleteRoleMembers(ctx, visitor, roleID, infos)
			assert.Error(t, err)
		})

		Convey("删除成员成功", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), "user1").Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			roleDB.EXPECT().GetRoleByID(gomock.Any(), roleID).Return(roleInfo, nil)
			roleMemberDB.EXPECT().DeleteRoleMembers(gomock.Any(), roleID, gomock.Any()).Return(nil)
			err := r.DeleteRoleMembers(ctx, visitor, roleID, infos)
			assert.NoError(t, err)
		})
	})
}

func TestRole_GetRoleMembers(t *testing.T) {
	Convey("测试GetRoleMembers方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		roleDB := mock.NewMockDBRole(ctrl)
		roleMemberDB := mock.NewMockDBRoleMember(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		logger := common.NewLogger()
		event := mock.NewMockLogicsEvent(ctrl)
		r := newRole(roleDB, roleMemberDB, userMgnt, logger, event)

		ctx := context.Background()
		visitor := &interfaces.Visitor{ID: "user1", Type: interfaces.RealName}
		roleID := roleTmpID
		roleInfo := interfaces.RoleInfo{ID: roleID}
		searchInfo := interfaces.RoleMemberSearchInfo{Offset: 0, Limit: 10}
		memberInfo := interfaces.RoleMemberInfo{ID: "member1", MemberType: interfaces.AccessorUser}

		Convey("权限检查失败", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), "user1").Return(nil, errors.New("权限错误"))
			_, _, err := r.GetRoleMembers(ctx, visitor, roleID, searchInfo)
			assert.Error(t, err)
		})

		Convey("角色不存在", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), "user1").Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			roleDB.EXPECT().GetRoleByID(gomock.Any(), roleID).Return(interfaces.RoleInfo{}, nil)
			_, _, err := r.GetRoleMembers(ctx, visitor, roleID, searchInfo)
			assert.Error(t, err)
		})

		Convey("获取成员数量失败", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), "user1").Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			roleDB.EXPECT().GetRoleByID(gomock.Any(), roleID).Return(roleInfo, nil)
			roleMemberDB.EXPECT().GetRoleMembersNum(gomock.Any(), roleID, searchInfo).Return(0, errors.New("db error"))
			_, _, err := r.GetRoleMembers(ctx, visitor, roleID, searchInfo)
			assert.Error(t, err)
		})

		Convey("获取成员成功", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), "user1").Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			roleDB.EXPECT().GetRoleByID(gomock.Any(), roleID).Return(roleInfo, nil)
			roleMemberDB.EXPECT().GetRoleMembersNum(gomock.Any(), roleID, searchInfo).Return(1, nil)
			roleMemberDB.EXPECT().GetPaginationByRoleID(gomock.Any(), roleID, searchInfo).Return([]interfaces.RoleMemberInfo{memberInfo}, nil)
			userMgnt.EXPECT().BatchGetUserInfoByID(gomock.Any(), gomock.Any()).Return(map[string]interfaces.UserInfo{"member1": {ParentDeps: [][]interfaces.Department{}}}, nil)
			num, infos, err := r.GetRoleMembers(ctx, visitor, roleID, searchInfo)
			assert.NoError(t, err)
			assert.Equal(t, 1, num)
			assert.Equal(t, 1, len(infos))
		})
	})
}

func TestRole_checkName(t *testing.T) {
	Convey("测试checkName方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		roleDB := mock.NewMockDBRole(ctrl)
		roleMemberDB := mock.NewMockDBRoleMember(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		logger := common.NewLogger()
		event := mock.NewMockLogicsEvent(ctrl)
		r := newRole(roleDB, roleMemberDB, userMgnt, logger, event)

		Convey("正常角色名称", func() {
			testCases := []string{
				"admin",
				"user_role",
				"角色名称",
				"test123",
				"a",                      // 最小长度
				strings.Repeat("a", 128), // 最大长度
			}

			for _, name := range testCases {
				err := r.checkName(name)
				assert.NoError(t, err, "角色名称 '%s' 应该通过验证", name)
			}
		})

		Convey("包含非法字符的角色名称", func() {
			illegalNames := []string{
				"admin\\role", // 反斜杠
				"admin/role",  // 正斜杠
				"admin:role",  // 冒号
				"admin*role",  // 星号
				"admin?role",  // 问号
				"admin\"role", // 双引号
				"admin<role",  // 小于号
				"admin>role",  // 大于号
				"admin|role",  // 竖线
			}

			for _, name := range illegalNames {
				err := r.checkName(name)
				assert.Error(t, err, "角色名称 '%s' 应该被拒绝", name)
			}
		})

		Convey("长度异常的角色名称", func() {
			// 空字符串
			err := r.checkName("")
			assert.Error(t, err, "空字符串应该被拒绝")

			// 超过最大长度
			longName := strings.Repeat("a", 129)
			err = r.checkName(longName)
			assert.Error(t, err, "超过128字符的名称应该被拒绝")
		})

		Convey("边界值测试", func() {
			// 1个字符 - 应该通过
			err := r.checkName("a")
			assert.NoError(t, err)

			// 128个字符 - 应该通过
			fiftyCharName := strings.Repeat("a", 128)
			err = r.checkName(fiftyCharName)
			assert.NoError(t, err)

			// 129个字符 - 应该失败
			fiftyOneCharName := strings.Repeat("a", 129)
			err = r.checkName(fiftyOneCharName)
			assert.Error(t, err)
		})
	})
}

func TestRole_AddRole(t *testing.T) {
	Convey("测试AddRole方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		roleDB := mock.NewMockDBRole(ctrl)
		roleMemberDB := mock.NewMockDBRoleMember(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		logger := common.NewLogger()
		event := mock.NewMockLogicsEvent(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		r := newRole(roleDB, roleMemberDB, userMgnt, logger, event)
		r.resourceType = resourceType

		ctx := context.Background()
		visitor := &interfaces.Visitor{ID: "user1", Type: interfaces.RealName}
		roleInfo := &interfaces.RoleInfo{Name: "test_role", ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
			Unlimited: true,
		}}

		Convey("权限检查失败", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), "user1").Return(nil, errors.New("权限错误"))
			_, err := r.AddRole(ctx, visitor, roleInfo)
			assert.Error(t, err)
		})

		Convey("角色名称非法", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), "user1").Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			roleInfo.Name = "invalid role |name"
			_, err := r.AddRole(ctx, visitor, roleInfo)
			assert.Error(t, err)
		})

		Convey("角色名称已存在", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), "user1").Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			roleInfo.Name = "existing_role"
			roleDB.EXPECT().GetRoleByName(gomock.Any(), "existing_role").Return(interfaces.RoleInfo{ID: "existing_id"}, nil)
			_, err := r.AddRole(ctx, visitor, roleInfo)
			assert.Error(t, err)
		})

		Convey("创建角色成功", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), "user1").Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			roleInfo.Name = "new_role"
			roleDB.EXPECT().GetRoleByName(gomock.Any(), "new_role").Return(interfaces.RoleInfo{}, nil)
			roleDB.EXPECT().AddRoles(gomock.Any(), gomock.Any()).Return(nil)
			id, err := r.AddRole(ctx, visitor, roleInfo)
			assert.NoError(t, err)
			assert.NotEmpty(t, id)
		})
	})
}

func TestRole_AddOrDeleteRoleMemebers(t *testing.T) {
	Convey("测试AddOrDeleteRoleMemebers方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		roleDB := mock.NewMockDBRole(ctrl)
		roleMemberDB := mock.NewMockDBRoleMember(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		logger := common.NewLogger()
		event := mock.NewMockLogicsEvent(ctrl)
		r := newRole(roleDB, roleMemberDB, userMgnt, logger, event)

		ctx := context.Background()
		visitor := &interfaces.Visitor{ID: "user1", Type: interfaces.RealName}
		roleID := roleTmpID
		infos := map[string]interfaces.RoleMemberInfo{"member1": {ID: "member1"}}

		Convey("POST方法 - 添加成员", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), "user1").Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			roleDB.EXPECT().GetRoleByID(gomock.Any(), roleID).Return(interfaces.RoleInfo{ID: roleID}, nil)
			roleMemberDB.EXPECT().GetRoleMembersByRoleID(gomock.Any(), roleID).Return([]interfaces.RoleMemberInfo{}, nil)
			userMgnt.EXPECT().GetNameByAccessorIDs(gomock.Any(), gomock.Any()).Return(map[string]string{"member1": "name1"}, nil)
			roleMemberDB.EXPECT().AddRoleMembers(gomock.Any(), roleID, gomock.Any()).Return(nil)
			err := r.AddOrDeleteRoleMemebers(ctx, visitor, "POST", roleID, infos)
			assert.NoError(t, err)
		})

		Convey("DELETE方法 - 删除成员", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), "user1").Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			roleDB.EXPECT().GetRoleByID(gomock.Any(), roleID).Return(interfaces.RoleInfo{ID: roleID}, nil)
			roleMemberDB.EXPECT().DeleteRoleMembers(gomock.Any(), roleID, gomock.Any()).Return(nil)
			err := r.AddOrDeleteRoleMemebers(ctx, visitor, "DELETE", roleID, infos)
			assert.NoError(t, err)
		})

		Convey("其他方法", func() {
			err := r.AddOrDeleteRoleMemebers(ctx, visitor, "PUT", roleID, infos)
			assert.Error(t, err)
		})
	})
}

func TestRole_GetRoleByMembers(t *testing.T) {
	Convey("测试GetRoleByMembers方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		roleDB := mock.NewMockDBRole(ctrl)
		roleMemberDB := mock.NewMockDBRoleMember(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		logger := common.NewLogger()
		event := mock.NewMockLogicsEvent(ctrl)
		r := newRole(roleDB, roleMemberDB, userMgnt, logger, event)

		ctx := context.Background()
		memberIDs := []string{"member1", "member2"}

		Convey("获取角色成功", func() {
			expectedRoles := []interfaces.RoleInfo{{ID: "role1"}, {ID: "role2"}}
			roleMemberDB.EXPECT().GetRoleByMembers(gomock.Any(), memberIDs).Return(expectedRoles, nil)
			roles, err := r.GetRoleByMembers(ctx, memberIDs)
			assert.NoError(t, err)
			assert.Equal(t, expectedRoles, roles)
		})

		Convey("数据库错误", func() {
			roleMemberDB.EXPECT().GetRoleByMembers(gomock.Any(), memberIDs).Return(nil, errors.New("db error"))
			roles, err := r.GetRoleByMembers(ctx, memberIDs)
			assert.Error(t, err)
			assert.Nil(t, roles)
		})
	})
}

func TestRole_GetRolesByIDs(t *testing.T) {
	Convey("测试GetRolesByIDs方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		roleDB := mock.NewMockDBRole(ctrl)
		roleMemberDB := mock.NewMockDBRoleMember(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		logger := common.NewLogger()
		event := mock.NewMockLogicsEvent(ctrl)
		r := newRole(roleDB, roleMemberDB, userMgnt, logger, event)

		ctx := context.Background()
		roleIDs := []string{"role1", "role2"}

		Convey("获取角色成功", func() {
			expectedRoles := map[string]interfaces.RoleInfo{
				"role1": {ID: "role1", Name: "Admin"},
				"role2": {ID: "role2", Name: "User"},
			}
			roleDB.EXPECT().GetRoleByIDs(gomock.Any(), roleIDs).Return(expectedRoles, nil)
			roles, err := r.GetRolesByIDs(ctx, roleIDs)
			assert.NoError(t, err)
			assert.Equal(t, expectedRoles, roles)
		})

		Convey("数据库错误", func() {
			roleDB.EXPECT().GetRoleByIDs(gomock.Any(), roleIDs).Return(nil, errors.New("db error"))
			roles, err := r.GetRolesByIDs(ctx, roleIDs)
			assert.Error(t, err)
			assert.Nil(t, roles)
		})
	})
}

func TestRole_InitRoles(t *testing.T) {
	Convey("测试InitRoles方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		roleDB := mock.NewMockDBRole(ctrl)
		roleMemberDB := mock.NewMockDBRoleMember(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		logger := common.NewLogger()
		event := mock.NewMockLogicsEvent(ctrl)
		r := newRole(roleDB, roleMemberDB, userMgnt, logger, event)

		ctx := context.Background()
		roles := []interfaces.RoleInfo{
			{Name: "existing_role", Description: "old description"},
			{Name: "new_role", Description: "new role"},
			{ID: "custom_id", Name: "custom_role", Description: "custom role"},
		}

		Convey("初始化角色成功 - 包含已存在角色和新增角色", func() {
			// 第一个角色已存在，但需要更新
			existingRole := interfaces.RoleInfo{
				ID:          "existing_id",
				Name:        "existing_role",
				Description: "old description",
			}
			roleDB.EXPECT().GetRoleByName(gomock.Any(), "existing_role").Return(existingRole, nil)

			// 第二个角色不存在
			roleDB.EXPECT().GetRoleByName(gomock.Any(), "new_role").Return(interfaces.RoleInfo{}, nil)

			// 第三个角色不存在但有自定义ID
			roleDB.EXPECT().GetRoleByName(gomock.Any(), "custom_role").Return(interfaces.RoleInfo{}, nil)

			// 添加新角色（第二个和第三个）
			roleDB.EXPECT().AddRoles(gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, roles []interfaces.RoleInfo) error {
					// 验证只添加了不存在的新角色
					assert.Equal(t, 2, len(roles))
					roleNames := make([]string, 0, len(roles))
					for _, role := range roles {
						roleNames = append(roleNames, role.Name)
					}
					assert.Contains(t, roleNames, "new_role")
					assert.Contains(t, roleNames, "custom_role")
					return nil
				})

			err := r.InitRoles(ctx, roles)
			assert.NoError(t, err)
		})

		Convey("初始化角色成功 - 已存在角色无需更新", func() {
			// 第一个角色已存在，且无需更新
			existingRole := interfaces.RoleInfo{
				ID:          "existing_id",
				Name:        "existing_role",
				Description: "old description",
			}
			roleDB.EXPECT().GetRoleByName(gomock.Any(), "existing_role").Return(existingRole, nil)
			// 由于角色信息相同，不会调用 SetRoleByID

			// 第二个角色不存在
			roleDB.EXPECT().GetRoleByName(gomock.Any(), "new_role").Return(interfaces.RoleInfo{}, nil)

			// 第三个角色不存在但有自定义ID
			roleDB.EXPECT().GetRoleByName(gomock.Any(), "custom_role").Return(interfaces.RoleInfo{}, nil)

			// 添加新角色
			roleDB.EXPECT().AddRoles(gomock.Any(), gomock.Any()).Return(nil)

			err := r.InitRoles(ctx, roles)
			assert.NoError(t, err)
		})

		Convey("获取角色信息失败", func() {
			roleDB.EXPECT().GetRoleByName(gomock.Any(), "existing_role").Return(interfaces.RoleInfo{}, errors.New("db error"))
			err := r.InitRoles(ctx, roles)
			assert.Error(t, err)
		})

		Convey("更新已存在角色失败", func() {
			// 第一个角色已存在，需要更新但更新失败
			existingRole := interfaces.RoleInfo{
				ID:          "existing_id",
				Name:        "existing_role",
				Description: "new description",
			}
			roleDB.EXPECT().GetRoleByName(gomock.Any(), "existing_role").Return(existingRole, nil)
			roleDB.EXPECT().SetRoleByID(gomock.Any(), "existing_id", gomock.Any()).Return(errors.New("update error"))

			err := r.InitRoles(ctx, roles)
			assert.Error(t, err)
		})

		Convey("添加新角色失败", func() {
			// 第一个角色已存在，无需更新
			existingRole := interfaces.RoleInfo{
				ID:          "existing_id",
				Name:        "existing_role",
				Description: "old description",
			}
			roleDB.EXPECT().GetRoleByName(gomock.Any(), "existing_role").Return(existingRole, nil)

			// 第二个角色不存在
			roleDB.EXPECT().GetRoleByName(gomock.Any(), "new_role").Return(interfaces.RoleInfo{}, nil)

			// 第三个角色不存在但有自定义ID
			roleDB.EXPECT().GetRoleByName(gomock.Any(), "custom_role").Return(interfaces.RoleInfo{}, nil)

			// 添加新角色失败
			roleDB.EXPECT().AddRoles(gomock.Any(), gomock.Any()).Return(errors.New("add error"))

			err := r.InitRoles(ctx, roles)
			assert.Error(t, err)
		})

		Convey("空角色列表", func() {
			err := r.InitRoles(ctx, []interfaces.RoleInfo{})
			assert.NoError(t, err)
		})
	})
}

func TestRole_InitRoleMemebers(t *testing.T) {
	Convey("测试InitRoleMemebers方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		roleDB := mock.NewMockDBRole(ctrl)
		roleMemberDB := mock.NewMockDBRoleMember(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		logger := common.NewLogger()
		event := mock.NewMockLogicsEvent(ctrl)
		r := newRole(roleDB, roleMemberDB, userMgnt, logger, event)

		ctx := context.Background()
		infoMap := map[string][]interfaces.RoleMemberInfo{
			roleTmpID: {
				{ID: "existing_member"},
				{ID: "new_member"},
			},
		}

		Convey("初始化角色成员成功", func() {
			// 获取现有成员
			roleMemberDB.EXPECT().GetRoleMembersByRoleID(gomock.Any(), roleTmpID).Return([]interfaces.RoleMemberInfo{{ID: "existing_member"}}, nil)
			// 添加新成员
			roleMemberDB.EXPECT().AddRoleMembers(gomock.Any(), roleTmpID, gomock.Any()).Return(nil)
			err := r.InitRoleMemebers(ctx, infoMap)
			assert.NoError(t, err)
		})

		Convey("获取现有成员失败", func() {
			roleMemberDB.EXPECT().GetRoleMembersByRoleID(gomock.Any(), "role1").Return(nil, errors.New("db error"))
			err := r.InitRoleMemebers(ctx, infoMap)
			assert.Error(t, err)
		})

		Convey("添加成员失败", func() {
			roleMemberDB.EXPECT().GetRoleMembersByRoleID(gomock.Any(), "role1").Return([]interfaces.RoleMemberInfo{}, nil)
			roleMemberDB.EXPECT().AddRoleMembers(gomock.Any(), "role1", gomock.Any()).Return(errors.New("add error"))
			err := r.InitRoleMemebers(ctx, infoMap)
			assert.Error(t, err)
		})
	})
}

func TestRole_deleteMemberByMemberID(t *testing.T) {
	Convey("测试deleteMemberByMemberID方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		roleDB := mock.NewMockDBRole(ctrl)
		roleMemberDB := mock.NewMockDBRoleMember(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		logger := common.NewLogger()
		event := mock.NewMockLogicsEvent(ctrl)
		r := newRole(roleDB, roleMemberDB, userMgnt, logger, event)

		memberID := "member1"

		Convey("删除成员成功", func() {
			roleMemberDB.EXPECT().DeleteByMemberIDs([]string{memberID}).Return(nil)
			err := r.deleteMemberByMemberID(memberID)
			assert.NoError(t, err)
		})

		Convey("删除成员失败", func() {
			roleMemberDB.EXPECT().DeleteByMemberIDs([]string{memberID}).Return(errors.New("delete error"))
			err := r.deleteMemberByMemberID(memberID)
			assert.Error(t, err)
		})
	})
}

func TestRole_updateMemberName(t *testing.T) {
	Convey("测试updateMemberName方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		roleDB := mock.NewMockDBRole(ctrl)
		roleMemberDB := mock.NewMockDBRoleMember(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		logger := common.NewLogger()
		event := mock.NewMockLogicsEvent(ctrl)
		r := newRole(roleDB, roleMemberDB, userMgnt, logger, event)

		memberID := "member1"
		name := "new_name"

		Convey("更新成员名称成功", func() {
			roleMemberDB.EXPECT().UpdateMemberName(memberID, name).Return(nil)
			err := r.updateMemberName(memberID, name)
			assert.NoError(t, err)
		})

		Convey("更新成员名称失败", func() {
			roleMemberDB.EXPECT().UpdateMemberName(memberID, name).Return(errors.New("update error"))
			err := r.updateMemberName(memberID, name)
			assert.Error(t, err)
		})
	})
}

func TestRole_updateAppName(t *testing.T) {
	Convey("测试updateAppName方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		roleDB := mock.NewMockDBRole(ctrl)
		roleMemberDB := mock.NewMockDBRoleMember(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		logger := common.NewLogger()
		event := mock.NewMockLogicsEvent(ctrl)
		r := newRole(roleDB, roleMemberDB, userMgnt, logger, event)

		appInfo := &interfaces.AppInfo{ID: "app1", Name: "new_app_name"}

		Convey("更新应用名称成功", func() {
			roleMemberDB.EXPECT().UpdateMemberName(appInfo.ID, appInfo.Name).Return(nil)
			err := r.updateAppName(appInfo)
			assert.NoError(t, err)
		})

		Convey("更新应用名称失败", func() {
			roleMemberDB.EXPECT().UpdateMemberName(appInfo.ID, appInfo.Name).Return(errors.New("update error"))
			err := r.updateAppName(appInfo)
			assert.Error(t, err)
		})
	})
}

func TestRole_checkAndFilterResourceTypeScopeInfo(t *testing.T) {
	Convey("测试checkAndFilterResourceTypeScopeInfo方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		roleDB := mock.NewMockDBRole(ctrl)
		roleMemberDB := mock.NewMockDBRoleMember(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		logger := common.NewLogger()
		event := mock.NewMockLogicsEvent(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		r := newRole(roleDB, roleMemberDB, userMgnt, logger, event)
		r.resourceType = resourceType

		ctx := context.Background()

		Convey("Unlimited为true时直接返回", func() {
			info := &interfaces.ResourceTypeScopeInfo{
				Unlimited: true,
				Types: []interfaces.ResourceTypeScope{
					{ResourceTypeID: "test1"},
					{ResourceTypeID: "test2"},
				},
			}

			result, err := r.checkAndFilterResourceTypeScopeInfo(ctx, info)
			assert.NoError(t, err)
			assert.True(t, result.Unlimited)
			assert.Empty(t, result.Types)
		})

		Convey("包含空的ResourceTypeID时返回错误", func() {
			info := &interfaces.ResourceTypeScopeInfo{
				Unlimited: false,
				Types: []interfaces.ResourceTypeScope{
					{ResourceTypeID: "test1"},
					{ResourceTypeID: ""}, // 空的ResourceTypeID
					{ResourceTypeID: "test2"},
				},
			}

			result, err := r.checkAndFilterResourceTypeScopeInfo(ctx, info)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "resource_type_scope is empty")
			assert.False(t, result.Unlimited)
			assert.Empty(t, result.Types)
		})

		Convey("包含重复的ResourceTypeID时返回错误", func() {
			info := &interfaces.ResourceTypeScopeInfo{
				Unlimited: false,
				Types: []interfaces.ResourceTypeScope{
					{ResourceTypeID: "test1"},
					{ResourceTypeID: "test2"},
					{ResourceTypeID: "test1"}, // 重复的ResourceTypeID
				},
			}

			result, err := r.checkAndFilterResourceTypeScopeInfo(ctx, info)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "resource_type_scope is duplicate")
			assert.False(t, result.Unlimited)
			assert.Empty(t, result.Types)
		})

		Convey("获取资源类型信息失败时返回错误", func() {
			info := &interfaces.ResourceTypeScopeInfo{
				Unlimited: false,
				Types: []interfaces.ResourceTypeScope{
					{ResourceTypeID: "test1"},
					{ResourceTypeID: "test2"},
				},
			}

			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), []string{"test1", "test2"}).Return(nil, errors.New("db error"))

			result, err := r.checkAndFilterResourceTypeScopeInfo(ctx, info)
			assert.Error(t, err)
			assert.Equal(t, "db error", err.Error())
			assert.False(t, result.Unlimited)
			assert.Empty(t, result.Types)
		})

		Convey("成功过滤存在的资源类型", func() {
			info := &interfaces.ResourceTypeScopeInfo{
				Unlimited: false,
				Types: []interfaces.ResourceTypeScope{
					{ResourceTypeID: "test1"},
					{ResourceTypeID: "test2"},
					{ResourceTypeID: "test3"},
				},
			}

			// 模拟只存在test1和test3，test2不存在
			resourceTypeMap := map[string]interfaces.ResourceType{
				"test1": {ID: "test1", Name: "Test1"},
				"test3": {ID: "test3", Name: "Test3"},
			}
			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), []string{"test1", "test2", "test3"}).Return(resourceTypeMap, nil)

			result, err := r.checkAndFilterResourceTypeScopeInfo(ctx, info)
			assert.NoError(t, err)
			assert.False(t, result.Unlimited)
			assert.Len(t, result.Types, 2)
			assert.Equal(t, "test1", result.Types[0].ResourceTypeID)
			assert.Equal(t, "test3", result.Types[1].ResourceTypeID)
		})

		Convey("所有资源类型都不存在时返回空列表", func() {
			info := &interfaces.ResourceTypeScopeInfo{
				Unlimited: false,
				Types: []interfaces.ResourceTypeScope{
					{ResourceTypeID: "test1"},
					{ResourceTypeID: "test2"},
				},
			}

			// 模拟所有资源类型都不存在
			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), []string{"test1", "test2"}).Return(map[string]interfaces.ResourceType{}, nil)

			result, err := r.checkAndFilterResourceTypeScopeInfo(ctx, info)
			assert.NoError(t, err)
			assert.False(t, result.Unlimited)
			assert.Empty(t, result.Types)
		})

		Convey("空Types列表时正常处理", func() {
			info := &interfaces.ResourceTypeScopeInfo{
				Unlimited: false,
				Types:     []interfaces.ResourceTypeScope{},
			}

			result, err := r.checkAndFilterResourceTypeScopeInfo(ctx, info)
			assert.NoError(t, err)
			assert.False(t, result.Unlimited)
			assert.Empty(t, result.Types)
		})

		Convey("所有资源类型都存在时保留所有", func() {
			info := &interfaces.ResourceTypeScopeInfo{
				Unlimited: false,
				Types: []interfaces.ResourceTypeScope{
					{ResourceTypeID: "test1"},
					{ResourceTypeID: "test2"},
				},
			}

			// 模拟所有资源类型都存在
			resourceTypeMap := map[string]interfaces.ResourceType{
				"test1": {ID: "test1", Name: "Test1"},
				"test2": {ID: "test2", Name: "Test2"},
			}
			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), []string{"test1", "test2"}).Return(resourceTypeMap, nil)

			result, err := r.checkAndFilterResourceTypeScopeInfo(ctx, info)
			assert.NoError(t, err)
			assert.False(t, result.Unlimited)
			assert.Len(t, result.Types, 2)
			assert.Equal(t, "test1", result.Types[0].ResourceTypeID)
			assert.Equal(t, "test2", result.Types[1].ResourceTypeID)
		})
	})
}

//nolint:funlen
func TestRole_checkRoleChange(t *testing.T) {
	Convey("测试checkRoleChange方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		roleDB := mock.NewMockDBRole(ctrl)
		roleMemberDB := mock.NewMockDBRoleMember(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		logger := common.NewLogger()
		event := mock.NewMockLogicsEvent(ctrl)
		r := newRole(roleDB, roleMemberDB, userMgnt, logger, event)

		Convey("角色信息完全相同 - 应该返回false", func() {
			oldInfo := &interfaces.RoleInfo{
				ID:          "role1",
				Name:        "admin",
				Description: "管理员角色",
				ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
					Unlimited: true,
					Types:     []interfaces.ResourceTypeScope{},
				},
			}
			newInfo := &interfaces.RoleInfo{
				ID:          "role1",
				Name:        "admin",
				Description: "管理员角色",
				ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
					Unlimited: true,
					Types:     []interfaces.ResourceTypeScope{},
				},
			}

			changed := r.checkRoleChange(oldInfo, newInfo)
			assert.False(t, changed, "角色信息完全相同，应该返回false")
		})

		Convey("角色名称不同 - 应该返回true", func() {
			oldInfo := &interfaces.RoleInfo{
				ID:          "role1",
				Name:        "admin",
				Description: "管理员角色",
				ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
					Unlimited: true,
					Types:     []interfaces.ResourceTypeScope{},
				},
			}
			newInfo := &interfaces.RoleInfo{
				ID:          "role1",
				Name:        "super_admin", // 名称不同
				Description: "管理员角色",
				ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
					Unlimited: true,
					Types:     []interfaces.ResourceTypeScope{},
				},
			}

			changed := r.checkRoleChange(oldInfo, newInfo)
			assert.True(t, changed, "角色名称不同，应该返回true")
		})

		Convey("角色描述不同 - 应该返回true", func() {
			oldInfo := &interfaces.RoleInfo{
				ID:          "role1",
				Name:        "admin",
				Description: "管理员角色",
				ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
					Unlimited: true,
					Types:     []interfaces.ResourceTypeScope{},
				},
			}
			newInfo := &interfaces.RoleInfo{
				ID:          "role1",
				Name:        "admin",
				Description: "超级管理员角色", // 描述不同
				ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
					Unlimited: true,
					Types:     []interfaces.ResourceTypeScope{},
				},
			}

			changed := r.checkRoleChange(oldInfo, newInfo)
			assert.True(t, changed, "角色描述不同，应该返回true")
		})

		Convey("资源类型范围信息不同 - Unlimited字段不同", func() {
			oldInfo := &interfaces.RoleInfo{
				ID:          "role1",
				Name:        "admin",
				Description: "管理员角色",
				ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
					Unlimited: true, // 不限制
					Types:     []interfaces.ResourceTypeScope{},
				},
			}
			newInfo := &interfaces.RoleInfo{
				ID:          "role1",
				Name:        "admin",
				Description: "管理员角色",
				ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
					Unlimited: false, // 限制
					Types:     []interfaces.ResourceTypeScope{},
				},
			}

			changed := r.checkRoleChange(oldInfo, newInfo)
			assert.True(t, changed, "资源类型范围Unlimited字段不同，应该返回true")
		})

		Convey("资源类型范围信息不同 - Types字段不同", func() {
			oldInfo := &interfaces.RoleInfo{
				ID:          "role1",
				Name:        "admin",
				Description: "管理员角色",
				ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
					Unlimited: false,
					Types: []interfaces.ResourceTypeScope{
						{ResourceTypeID: "type1", ResourceTypeName: "文档"},
					},
				},
			}
			newInfo := &interfaces.RoleInfo{
				ID:          "role1",
				Name:        "admin",
				Description: "管理员角色",
				ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
					Unlimited: false,
					Types: []interfaces.ResourceTypeScope{
						{ResourceTypeID: "type2", ResourceTypeName: "图片"}, // 不同的资源类型
					},
				},
			}

			changed := r.checkRoleChange(oldInfo, newInfo)
			assert.True(t, changed, "资源类型范围Types字段不同，应该返回true")
		})

		Convey("资源类型范围信息不同 - Types数量不同", func() {
			oldInfo := &interfaces.RoleInfo{
				ID:          "role1",
				Name:        "admin",
				Description: "管理员角色",
				ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
					Unlimited: false,
					Types: []interfaces.ResourceTypeScope{
						{ResourceTypeID: "type1", ResourceTypeName: "文档"},
					},
				},
			}
			newInfo := &interfaces.RoleInfo{
				ID:          "role1",
				Name:        "admin",
				Description: "管理员角色",
				ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
					Unlimited: false,
					Types: []interfaces.ResourceTypeScope{
						{ResourceTypeID: "type1", ResourceTypeName: "文档"},
						{ResourceTypeID: "type2", ResourceTypeName: "图片"}, // 多了一个资源类型
					},
				},
			}

			changed := r.checkRoleChange(oldInfo, newInfo)
			assert.True(t, changed, "资源类型范围Types数量不同，应该返回true")
		})

		Convey("资源类型范围信息不同 - Types为空vs非空", func() {
			oldInfo := &interfaces.RoleInfo{
				ID:          "role1",
				Name:        "admin",
				Description: "管理员角色",
				ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
					Unlimited: false,
					Types:     []interfaces.ResourceTypeScope{}, // 空
				},
			}
			newInfo := &interfaces.RoleInfo{
				ID:          "role1",
				Name:        "admin",
				Description: "管理员角色",
				ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
					Unlimited: false,
					Types: []interfaces.ResourceTypeScope{
						{ResourceTypeID: "type1", ResourceTypeName: "文档"}, // 非空
					},
				},
			}

			changed := r.checkRoleChange(oldInfo, newInfo)
			assert.True(t, changed, "资源类型范围Types从空变为非空，应该返回true")
		})

		Convey("多个字段同时不同 - 应该返回true", func() {
			oldInfo := &interfaces.RoleInfo{
				ID:          "role1",
				Name:        "admin",
				Description: "管理员角色",
				ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
					Unlimited: true,
					Types:     []interfaces.ResourceTypeScope{},
				},
			}
			newInfo := &interfaces.RoleInfo{
				ID:          "role1",
				Name:        "super_admin", // 名称不同
				Description: "超级管理员角色",     // 描述不同
				ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
					Unlimited: false, // Unlimited不同
					Types: []interfaces.ResourceTypeScope{
						{ResourceTypeID: "type1", ResourceTypeName: "文档"}, // Types不同
					},
				},
			}

			changed := r.checkRoleChange(oldInfo, newInfo)
			assert.True(t, changed, "多个字段同时不同，应该返回true")
		})

		Convey("边界情况 - 空字符串vs非空字符串", func() {
			oldInfo := &interfaces.RoleInfo{
				ID:          "role1",
				Name:        "", // 空字符串
				Description: "管理员角色",
				ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
					Unlimited: true,
					Types:     []interfaces.ResourceTypeScope{},
				},
			}
			newInfo := &interfaces.RoleInfo{
				ID:          "role1",
				Name:        "admin", // 非空字符串
				Description: "管理员角色",
				ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
					Unlimited: true,
					Types:     []interfaces.ResourceTypeScope{},
				},
			}

			changed := r.checkRoleChange(oldInfo, newInfo)
			assert.True(t, changed, "空字符串vs非空字符串，应该返回true")
		})

		Convey("边界情况 - 空描述vs非空描述", func() {
			oldInfo := &interfaces.RoleInfo{
				ID:          "role1",
				Name:        "admin",
				Description: "", // 空描述
				ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
					Unlimited: true,
					Types:     []interfaces.ResourceTypeScope{},
				},
			}
			newInfo := &interfaces.RoleInfo{
				ID:          "role1",
				Name:        "admin",
				Description: "管理员角色", // 非空描述
				ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
					Unlimited: true,
					Types:     []interfaces.ResourceTypeScope{},
				},
			}

			changed := r.checkRoleChange(oldInfo, newInfo)
			assert.True(t, changed, "空描述vs非空描述，应该返回true")
		})

		Convey("复杂资源类型范围比较 - 相同内容但顺序不同", func() {
			oldInfo := &interfaces.RoleInfo{
				ID:          "role1",
				Name:        "admin",
				Description: "管理员角色",
				ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
					Unlimited: false,
					Types: []interfaces.ResourceTypeScope{
						{ResourceTypeID: "type1", ResourceTypeName: "文档"},
						{ResourceTypeID: "type2", ResourceTypeName: "图片"},
					},
				},
			}
			newInfo := &interfaces.RoleInfo{
				ID:          "role1",
				Name:        "admin",
				Description: "管理员角色",
				ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
					Unlimited: false,
					Types: []interfaces.ResourceTypeScope{
						{ResourceTypeID: "type2", ResourceTypeName: "图片"}, // 顺序不同
						{ResourceTypeID: "type1", ResourceTypeName: "文档"},
					},
				},
			}

			changed := r.checkRoleChange(oldInfo, newInfo)
			assert.True(t, changed, "资源类型范围内容相同但顺序不同，应该返回true（因为reflect.DeepEqual会考虑顺序）")
		})

		Convey("复杂资源类型范围比较 - 相同内容且顺序相同", func() {
			oldInfo := &interfaces.RoleInfo{
				ID:          "role1",
				Name:        "admin",
				Description: "管理员角色",
				ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
					Unlimited: false,
					Types: []interfaces.ResourceTypeScope{
						{ResourceTypeID: "type1", ResourceTypeName: "文档"},
						{ResourceTypeID: "type2", ResourceTypeName: "图片"},
					},
				},
			}
			newInfo := &interfaces.RoleInfo{
				ID:          "role1",
				Name:        "admin",
				Description: "管理员角色",
				ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
					Unlimited: false,
					Types: []interfaces.ResourceTypeScope{
						{ResourceTypeID: "type1", ResourceTypeName: "文档"},
						{ResourceTypeID: "type2", ResourceTypeName: "图片"},
					},
				},
			}

			changed := r.checkRoleChange(oldInfo, newInfo)
			assert.False(t, changed, "资源类型范围内容相同且顺序相同，应该返回false")
		})
	})
}

func TestRole_getOperationNameByLanguage(t *testing.T) {
	Convey("测试getOperationNameByLanguage方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		roleDB := mock.NewMockDBRole(ctrl)
		roleMemberDB := mock.NewMockDBRoleMember(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		logger := common.NewLogger()
		event := mock.NewMockLogicsEvent(ctrl)
		r := newRole(roleDB, roleMemberDB, userMgnt, logger, event)

		Convey("找到匹配的语言时返回对应的名称", func() {
			operationNames := []interfaces.OperationName{
				{Language: "zh-cn", Value: "读取"},
				{Language: "en-us", Value: "read"},
				{Language: "zh-tw", Value: "讀取"},
			}

			result := r.getOperationNameByLanguage("zh-cn", operationNames)
			assert.Equal(t, "读取", result)
		})

		Convey("语言匹配不区分大小写", func() {
			operationNames := []interfaces.OperationName{
				{Language: "zh-cn", Value: "读取"},
				{Language: "en-us", Value: "read"},
				{Language: "zh-CN", Value: "读取"},
			}

			// 测试小写匹配
			result := r.getOperationNameByLanguage("zh-cn", operationNames)
			assert.Equal(t, "读取", result)

			// 测试大写匹配
			result = r.getOperationNameByLanguage("ZH-CN", operationNames)
			assert.Equal(t, "读取", result)

			// 测试混合大小写匹配
			result = r.getOperationNameByLanguage("Zh-Cn", operationNames)
			assert.Equal(t, "读取", result)
		})

		Convey("找不到匹配的语言时返回空字符串", func() {
			operationNames := []interfaces.OperationName{
				{Language: "zh-cn", Value: "读取"},
				{Language: "en-us", Value: "read"},
			}

			result := r.getOperationNameByLanguage("fr-fr", operationNames)
			assert.Equal(t, "", result)
		})

		Convey("空操作名称列表时返回空字符串", func() {
			result := r.getOperationNameByLanguage("zh-cn", []interfaces.OperationName{})
			assert.Equal(t, "", result)
		})

		Convey("空语言参数时返回空字符串", func() {
			operationNames := []interfaces.OperationName{
				{Language: "zh-cn", Value: "读取"},
				{Language: "en-us", Value: "read"},
			}

			result := r.getOperationNameByLanguage("", operationNames)
			assert.Equal(t, "", result)
		})

		Convey("多个相同语言时返回第一个匹配的", func() {
			operationNames := []interfaces.OperationName{
				{Language: "zh-cn", Value: "读取"},
				{Language: "zh-cn", Value: "查看"}, // 重复的语言
				{Language: "en-us", Value: "read"},
			}

			result := r.getOperationNameByLanguage("zh-cn", operationNames)
			assert.Equal(t, "读取", result) // 应该返回第一个匹配的
		})

		Convey("包含空语言的操作名称", func() {
			operationNames := []interfaces.OperationName{
				{Language: "", Value: "默认操作"},
				{Language: "zh-cn", Value: "读取"},
				{Language: "en-us", Value: "read"},
			}

			// 空语言应该能匹配
			result := r.getOperationNameByLanguage("", operationNames)
			assert.Equal(t, "默认操作", result)

			// 正常语言匹配
			result = r.getOperationNameByLanguage("zh-cn", operationNames)
			assert.Equal(t, "读取", result)
		})

		Convey("包含空值的操作名称", func() {
			operationNames := []interfaces.OperationName{
				{Language: "zh-cn", Value: ""}, // 空值
				{Language: "en-us", Value: "read"},
			}

			result := r.getOperationNameByLanguage("zh-cn", operationNames)
			assert.Equal(t, "", result) // 应该返回空值
		})

		Convey("特殊字符和空格的处理", func() {
			operationNames := []interfaces.OperationName{
				{Language: "zh-cn", Value: "读取 文件"},
				{Language: "en-us", Value: "read file"},
				{Language: "zh-CN", Value: "读取文件"},
			}

			result := r.getOperationNameByLanguage("zh-cn", operationNames)
			assert.Equal(t, "读取 文件", result)

			result = r.getOperationNameByLanguage("ZH-CN", operationNames)
			assert.Equal(t, "读取 文件", result)
		})

		Convey("边界情况测试", func() {
			// 单个操作名称
			operationNames := []interfaces.OperationName{
				{Language: "zh-cn", Value: "读取"},
			}

			result := r.getOperationNameByLanguage("zh-cn", operationNames)
			assert.Equal(t, "读取", result)

			// 不匹配的情况
			result = r.getOperationNameByLanguage("en-us", operationNames)
			assert.Equal(t, "", result)
		})

		Convey("大量操作名称的性能测试", func() {
			operationNames := make([]interfaces.OperationName, 1000)
			for i := 0; i < 1000; i++ {
				operationNames[i] = interfaces.OperationName{
					Language: fmt.Sprintf("lang-%d", i),
					Value:    fmt.Sprintf("value-%d", i),
				}
			}
			// 添加目标语言在中间位置
			operationNames[500] = interfaces.OperationName{
				Language: "target-lang",
				Value:    "target-value",
			}

			result := r.getOperationNameByLanguage("target-lang", operationNames)
			assert.Equal(t, "target-value", result)
		})
	})
}

//nolint:funlen
func TestRole_getResourceTypeScopeInfo(t *testing.T) {
	Convey("测试getResourceTypeScopeInfo方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		roleDB := mock.NewMockDBRole(ctrl)
		roleMemberDB := mock.NewMockDBRoleMember(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		logger := common.NewLogger()
		event := mock.NewMockLogicsEvent(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		r := newRole(roleDB, roleMemberDB, userMgnt, logger, event)
		r.resourceType = resourceType

		ctx := context.Background()
		visitor := &interfaces.Visitor{
			ID:       "user1",
			Type:     interfaces.RealName,
			Language: "zh-cn",
		}

		param := interfaces.RoleInfoParam{
			ResourceTypeViewMode: interfaces.ResourceTypeViewModeFlat,
		}

		Convey("Unlimited为true时获取所有资源类型", func() {
			info := interfaces.ResourceTypeScopeInfo{
				Unlimited: true,
				Types:     []interfaces.ResourceTypeScope{},
			}

			// 模拟获取所有资源类型
			allResourceTypes := []interfaces.ResourceType{
				{
					ID:          "doc",
					Name:        "文档",
					Description: "文档资源",
					InstanceURL: "/api/doc",
					DataStruct:  "tree",
					Operation: []interfaces.ResourceTypeOperation{
						{
							ID:          "read",
							Description: "读取操作",
							Scope:       []interfaces.OperationScopeType{interfaces.ScopeType, interfaces.ScopeInstance},
							Name: []interfaces.OperationName{
								{Language: "zh-cn", Value: "读取"},
								{Language: "en-us", Value: "Read"},
							},
						},
						{
							ID:          "write",
							Description: "写入操作",
							Scope:       []interfaces.OperationScopeType{interfaces.ScopeInstance},
							Name: []interfaces.OperationName{
								{Language: "zh-cn", Value: "写入"},
								{Language: "en-us", Value: "Write"},
							},
						},
					},
				},
				{
					ID:          "folder",
					Name:        "文件夹",
					Description: "文件夹资源",
					InstanceURL: "/api/folder",
					DataStruct:  "tree",
					Operation: []interfaces.ResourceTypeOperation{
						{
							ID:          "create",
							Description: "创建操作",
							Scope:       []interfaces.OperationScopeType{interfaces.ScopeType},
							Name: []interfaces.OperationName{
								{Language: "zh-cn", Value: "创建"},
								{Language: "en-us", Value: "Create"},
							},
						},
					},
				},
			}

			resourceType.EXPECT().GetAllInternal(ctx).Return(allResourceTypes, nil)

			result, err := r.getResourceTypeScopeInfo(ctx, visitor, info, param)

			assert.NoError(t, err)
			assert.True(t, result.Unlimited)
			assert.Len(t, result.Types, 2)

			// 验证第一个资源类型
			docType := result.Types[0]
			assert.Equal(t, "doc", docType.ID)
			assert.Equal(t, "文档", docType.Name)
			assert.Equal(t, "文档资源", docType.Description)
			assert.Equal(t, "/api/doc", docType.InstanceURL)
			assert.Equal(t, "tree", docType.DataStruct)

			// 验证第二个资源类型
			folderType := result.Types[1]
			assert.Equal(t, "folder", folderType.ID)
			assert.Equal(t, "文件夹", folderType.Name)
			assert.Len(t, folderType.TypeOperation, 1)
			assert.Len(t, folderType.InstanceOperation, 0)
		})

		Convey("Unlimited为false时获取指定资源类型", func() {
			info := interfaces.ResourceTypeScopeInfo{
				Unlimited: false,
				Types: []interfaces.ResourceTypeScope{
					{ResourceTypeID: "doc"},
					{ResourceTypeID: "image"},
				},
			}

			// 模拟获取指定资源类型
			resourceTypeMap := map[string]interfaces.ResourceType{
				"doc": {
					ID:          "doc",
					Name:        "文档",
					Description: "文档资源",
					InstanceURL: "/api/doc",
					DataStruct:  "tree",
					Operation: []interfaces.ResourceTypeOperation{
						{
							ID:          "read",
							Description: "读取操作",
							Scope:       []interfaces.OperationScopeType{interfaces.ScopeType},
							Name: []interfaces.OperationName{
								{Language: "zh-cn", Value: "读取"},
								{Language: "en-us", Value: "Read"},
							},
						},
					},
				},
				"image": {
					ID:          "image",
					Name:        "图片",
					Description: "图片资源",
					InstanceURL: "/api/image",
					DataStruct:  "string",
					Operation: []interfaces.ResourceTypeOperation{
						{
							ID:          "view",
							Description: "查看操作",
							Scope:       []interfaces.OperationScopeType{interfaces.ScopeInstance},
							Name: []interfaces.OperationName{
								{Language: "zh-cn", Value: "查看"},
								{Language: "en-us", Value: "View"},
							},
						},
					},
				},
			}

			resourceType.EXPECT().GetByIDsInternal(ctx, []string{"doc", "image"}).Return(resourceTypeMap, nil)

			result, err := r.getResourceTypeScopeInfo(ctx, visitor, info, param)

			assert.NoError(t, err)
			assert.False(t, result.Unlimited)
			assert.Len(t, result.Types, 2)

			// 验证文档资源类型
			docType := result.Types[0]
			assert.Equal(t, "doc", docType.ID)
			assert.Equal(t, "文档", docType.Name)
			assert.Len(t, docType.TypeOperation, 1)
			assert.Len(t, docType.InstanceOperation, 0)

			// 验证图片资源类型
			imageType := result.Types[1]
			assert.Equal(t, "image", imageType.ID)
			assert.Equal(t, "图片", imageType.Name)
			assert.Len(t, imageType.TypeOperation, 0)
			assert.Len(t, imageType.InstanceOperation, 1)
		})

		Convey("GetAllInternal返回错误时", func() {
			info := interfaces.ResourceTypeScopeInfo{
				Unlimited: true,
				Types:     []interfaces.ResourceTypeScope{},
			}

			resourceType.EXPECT().GetAllInternal(ctx).Return(nil, errors.New("database error"))

			result, err := r.getResourceTypeScopeInfo(ctx, visitor, info, param)

			assert.Error(t, err)
			assert.Equal(t, "database error", err.Error())
			assert.True(t, result.Unlimited)
			assert.Empty(t, result.Types)
		})

		Convey("GetByIDsInternal返回错误时", func() {
			info := interfaces.ResourceTypeScopeInfo{
				Unlimited: false,
				Types: []interfaces.ResourceTypeScope{
					{ResourceTypeID: "doc"},
				},
			}

			resourceType.EXPECT().GetByIDsInternal(ctx, []string{"doc"}).Return(nil, errors.New("database error"))

			result, err := r.getResourceTypeScopeInfo(ctx, visitor, info, param)

			assert.Error(t, err)
			assert.Equal(t, "database error", err.Error())
			assert.False(t, result.Unlimited)
			assert.Empty(t, result.Types)
		})

		Convey("资源类型不存在时跳过", func() {
			info := interfaces.ResourceTypeScopeInfo{
				Unlimited: false,
				Types: []interfaces.ResourceTypeScope{
					{ResourceTypeID: "doc"},
					{ResourceTypeID: "nonexistent"},
				},
			}

			// 只返回存在的资源类型
			resourceTypeMap := map[string]interfaces.ResourceType{
				"doc": {
					ID:          "doc",
					Name:        "文档",
					Description: "文档资源",
					InstanceURL: "/api/doc",
					DataStruct:  "tree",
					Operation:   []interfaces.ResourceTypeOperation{},
				},
			}

			resourceType.EXPECT().GetByIDsInternal(ctx, []string{"doc", "nonexistent"}).Return(resourceTypeMap, nil)

			result, err := r.getResourceTypeScopeInfo(ctx, visitor, info, param)

			assert.NoError(t, err)
			assert.False(t, result.Unlimited)
			assert.Len(t, result.Types, 1) // 只包含存在的资源类型
			assert.Equal(t, "doc", result.Types[0].ID)
		})

		Convey("空操作列表时", func() {
			info := interfaces.ResourceTypeScopeInfo{
				Unlimited: false,
				Types: []interfaces.ResourceTypeScope{
					{ResourceTypeID: "doc"},
				},
			}

			resourceTypeMap := map[string]interfaces.ResourceType{
				"doc": {
					ID:          "doc",
					Name:        "文档",
					Description: "文档资源",
					InstanceURL: "/api/doc",
					DataStruct:  "tree",
					Operation:   []interfaces.ResourceTypeOperation{}, // 空操作列表
				},
			}

			resourceType.EXPECT().GetByIDsInternal(ctx, []string{"doc"}).Return(resourceTypeMap, nil)

			result, err := r.getResourceTypeScopeInfo(ctx, visitor, info, param)

			assert.NoError(t, err)
			assert.False(t, result.Unlimited)
			assert.Len(t, result.Types, 1)
			assert.Empty(t, result.Types[0].TypeOperation)
			assert.Empty(t, result.Types[0].InstanceOperation)
		})

		Convey("操作范围分类正确", func() {
			info := interfaces.ResourceTypeScopeInfo{
				Unlimited: false,
				Types: []interfaces.ResourceTypeScope{
					{ResourceTypeID: "doc"},
				},
			}

			resourceTypeMap := map[string]interfaces.ResourceType{
				"doc": {
					ID:          "doc",
					Name:        "文档",
					Description: "文档资源",
					InstanceURL: "/api/doc",
					DataStruct:  "tree",
					Operation: []interfaces.ResourceTypeOperation{
						{
							ID:          "read",
							Description: "读取操作",
							Scope:       []interfaces.OperationScopeType{interfaces.ScopeType}, // 只有类型操作
							Name: []interfaces.OperationName{
								{Language: "zh-cn", Value: "读取"},
							},
						},
						{
							ID:          "write",
							Description: "写入操作",
							Scope:       []interfaces.OperationScopeType{interfaces.ScopeInstance}, // 只有实例操作
							Name: []interfaces.OperationName{
								{Language: "zh-cn", Value: "写入"},
							},
						},
						{
							ID:          "delete",
							Description: "删除操作",
							Scope:       []interfaces.OperationScopeType{interfaces.ScopeType, interfaces.ScopeInstance}, // 两种操作都有
							Name: []interfaces.OperationName{
								{Language: "zh-cn", Value: "删除"},
							},
						},
					},
				},
			}

			resourceType.EXPECT().GetByIDsInternal(ctx, []string{"doc"}).Return(resourceTypeMap, nil)

			result, err := r.getResourceTypeScopeInfo(ctx, visitor, info, param)

			assert.NoError(t, err)
			assert.Len(t, result.Types, 1)

			docType := result.Types[0]
			// 类型操作应该包含 read 和 delete
			assert.Len(t, docType.TypeOperation, 2)
			typeOpIDs := []string{docType.TypeOperation[0].ID, docType.TypeOperation[1].ID}
			assert.Contains(t, typeOpIDs, "read")
			assert.Contains(t, typeOpIDs, "delete")

			// 实例操作应该包含 write 和 delete
			assert.Len(t, docType.InstanceOperation, 2)
			instanceOpIDs := []string{docType.InstanceOperation[0].ID, docType.InstanceOperation[1].ID}
			assert.Contains(t, instanceOpIDs, "write")
			assert.Contains(t, instanceOpIDs, "delete")
		})

		Convey("不同语言的操作名称", func() {
			visitor.Language = "en-us"
			info := interfaces.ResourceTypeScopeInfo{
				Unlimited: false,
				Types: []interfaces.ResourceTypeScope{
					{ResourceTypeID: "doc"},
				},
			}

			resourceTypeMap := map[string]interfaces.ResourceType{
				"doc": {
					ID:          "doc",
					Name:        "文档",
					Description: "文档资源",
					InstanceURL: "/api/doc",
					DataStruct:  "tree",
					Operation: []interfaces.ResourceTypeOperation{
						{
							ID:          "read",
							Description: "读取操作",
							Scope:       []interfaces.OperationScopeType{interfaces.ScopeType},
							Name: []interfaces.OperationName{
								{Language: "zh-cn", Value: "读取"},
								{Language: "en-us", Value: "Read"},
							},
						},
					},
				},
			}

			resourceType.EXPECT().GetByIDsInternal(ctx, []string{"doc"}).Return(resourceTypeMap, nil)

			result, err := r.getResourceTypeScopeInfo(ctx, visitor, info, param)

			assert.NoError(t, err)
			assert.Len(t, result.Types, 1)
			assert.Len(t, result.Types[0].TypeOperation, 1)
			assert.Equal(t, "Read", result.Types[0].TypeOperation[0].Name) // 应该返回英文名称
		})

		Convey("空资源类型列表", func() {
			info := interfaces.ResourceTypeScopeInfo{
				Unlimited: false,
				Types:     []interfaces.ResourceTypeScope{}, // 空列表
			}

			resourceType.EXPECT().GetByIDsInternal(ctx, []string{}).Return(map[string]interfaces.ResourceType{}, nil)

			result, err := r.getResourceTypeScopeInfo(ctx, visitor, info, param)

			assert.NoError(t, err)
			assert.False(t, result.Unlimited)
			assert.Empty(t, result.Types)
		})

		Convey("GetAllInternal返回空列表", func() {
			info := interfaces.ResourceTypeScopeInfo{
				Unlimited: true,
				Types:     []interfaces.ResourceTypeScope{},
			}

			resourceType.EXPECT().GetAllInternal(ctx).Return([]interfaces.ResourceType{}, nil)

			result, err := r.getResourceTypeScopeInfo(ctx, visitor, info, param)

			assert.NoError(t, err)
			assert.True(t, result.Unlimited)
			assert.Empty(t, result.Types)
		})
	})
}

func TestRole_getResourceTypeScopeInfoHierarchy(t *testing.T) {
	Convey("测试getResourceTypeScopeInfoHierarchy方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		roleDB := mock.NewMockDBRole(ctrl)
		roleMemberDB := mock.NewMockDBRoleMember(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		logger := common.NewLogger()
		event := mock.NewMockLogicsEvent(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		r := newRole(roleDB, roleMemberDB, userMgnt, logger, event)
		r.resourceTypeHierarchy = resourceTypeHierarchy

		ctx := context.Background()
		visitor := &interfaces.Visitor{ID: "user1", Type: interfaces.RealName, Language: "zh-cn"}

		resourceTypeInfoMap := map[string]interfaces.ResourceType{
			"menu": {
				ID:          "menu",
				Name:        "菜单",
				Description: "菜单资源",
				InstanceURL: "/menu",
				DataStruct:  "{}",
				Operation: []interfaces.ResourceTypeOperation{
					{
						ID:          "read",
						Description: "读取",
						Scope:       []interfaces.OperationScopeType{interfaces.ScopeType},
						Name:        []interfaces.OperationName{{Language: "zh-cn", Value: "读取"}},
					},
				},
			},
			"submenu": {
				ID:          "submenu",
				Name:        "子菜单",
				Description: "子菜单资源",
				InstanceURL: "/submenu",
				DataStruct:  "{}",
				Operation:   []interfaces.ResourceTypeOperation{},
			},
		}

		Convey("GetAll失败", func() {
			resourceTypeHierarchy.EXPECT().GetAll(gomock.Any()).Return(nil, errors.New("db error"))

			result, err := r.getResourceTypeScopeInfoHierarchy(ctx, visitor, []string{"menu"}, resourceTypeInfoMap)
			assert.Error(t, err)
			assert.Nil(t, result)
		})

		Convey("空resourceTypeIDs", func() {
			resourceTypeHierarchy.EXPECT().GetAll(gomock.Any()).Return(map[string]interfaces.ResourceTypeHierarchy{}, nil)

			result, err := r.getResourceTypeScopeInfoHierarchy(ctx, visitor, []string{}, resourceTypeInfoMap)
			assert.NoError(t, err)
			assert.Empty(t, result)
		})

		Convey("无层级关系-扁平返回", func() {
			resourceTypeHierarchy.EXPECT().GetAll(gomock.Any()).Return(map[string]interfaces.ResourceTypeHierarchy{}, nil)

			result, err := r.getResourceTypeScopeInfoHierarchy(ctx, visitor, []string{"menu"}, resourceTypeInfoMap)
			assert.NoError(t, err)
			assert.Len(t, result, 1)
			assert.Equal(t, "menu", result[0].ID)
			assert.Equal(t, "菜单", result[0].Name)
			assert.Empty(t, result[0].Children)
		})

		Convey("有层级关系-递归填充子资源类型", func() {
			hierarchyMap := map[string]interfaces.ResourceTypeHierarchy{
				"menu": {
					ResourceTypeID: "menu",
					Children: []interfaces.ResourceTypeHierarchy{
						{ResourceTypeID: "submenu", Children: nil},
					},
				},
			}
			resourceTypeHierarchy.EXPECT().GetAll(gomock.Any()).Return(hierarchyMap, nil)

			result, err := r.getResourceTypeScopeInfoHierarchy(ctx, visitor, []string{"menu", "submenu"}, resourceTypeInfoMap)
			assert.NoError(t, err)
			assert.Len(t, result, 1)
			assert.Equal(t, "menu", result[0].ID)
			assert.Len(t, result[0].Children, 1)
			assert.Equal(t, "submenu", result[0].Children[0].ID)
			assert.Equal(t, "子菜单", result[0].Children[0].Name)
		})

		Convey("仅子层级资源类型时被跳过", func() {
			hierarchyMap := map[string]interfaces.ResourceTypeHierarchy{
				"menu": {
					ResourceTypeID: "menu",
					Children: []interfaces.ResourceTypeHierarchy{
						{ResourceTypeID: "submenu", Children: nil},
					},
				},
			}
			resourceTypeHierarchy.EXPECT().GetAll(gomock.Any()).Return(hierarchyMap, nil)

			// 仅传 submenu：submenu 在 childrenMap 中但不在 topMap 中，应被跳过
			result, err := r.getResourceTypeScopeInfoHierarchy(ctx, visitor, []string{"submenu"}, resourceTypeInfoMap)
			assert.NoError(t, err)
			assert.Empty(t, result)
		})

		Convey("嵌套children场景", func() {
			resourceTypeInfoMap["grandchild"] = interfaces.ResourceType{
				ID: "grandchild", Name: "孙级", Description: "", InstanceURL: "", DataStruct: "", Operation: nil,
			}
			hierarchyMap := map[string]interfaces.ResourceTypeHierarchy{
				"menu": {
					ResourceTypeID: "menu",
					Children: []interfaces.ResourceTypeHierarchy{
						{
							ResourceTypeID: "submenu",
							Children: []interfaces.ResourceTypeHierarchy{
								{ResourceTypeID: "grandchild", Children: nil},
							},
						},
					},
				},
			}
			resourceTypeHierarchy.EXPECT().GetAll(gomock.Any()).Return(hierarchyMap, nil)

			result, err := r.getResourceTypeScopeInfoHierarchy(ctx, visitor, []string{"menu", "submenu", "grandchild"}, resourceTypeInfoMap)
			assert.NoError(t, err)
			assert.Len(t, result, 1)
			assert.Equal(t, "menu", result[0].ID)
			assert.Len(t, result[0].Children, 1)
			assert.Equal(t, "submenu", result[0].Children[0].ID)
			assert.Len(t, result[0].Children[0].Children, 1)
			assert.Equal(t, "grandchild", result[0].Children[0].Children[0].ID)
			assert.Equal(t, "孙级", result[0].Children[0].Children[0].Name)
		})
	})
}

//nolint:funlen
func TestRole_GetResourceTypeRoles(t *testing.T) {
	Convey("TestRole_GetResourceTypeRoles", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		roleDB := mock.NewMockDBRole(ctrl)
		roleMemberDB := mock.NewMockDBRoleMember(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		logger := common.NewLogger()
		event := mock.NewMockLogicsEvent(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		r := newRole(roleDB, roleMemberDB, userMgnt, logger, event)
		r.resourceType = resourceType

		ctx := context.Background()
		visitor := &interfaces.Visitor{ID: "user1", Type: interfaces.RealName}
		searchInfo := interfaces.ResourceTypeRoleSearchInfo{
			ResourceTypeID: "test-resource-type",
			Offset:         0,
			Limit:          10,
			Keyword:        "test",
		}

		Convey("资源类型不存在", func() {
			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), []string{"test-resource-type"}).Return(map[string]interfaces.ResourceType{}, nil)
			count, roles, err := r.GetResourceTypeRoles(ctx, visitor, searchInfo)
			assert.Error(t, err)
			assert.Equal(t, count, 0)
			assert.Equal(t, len(roles), 0)
		})

		Convey("获取资源类型失败", func() {
			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), []string{"test-resource-type"}).Return(nil, errors.New("db error"))
			count, roles, err := r.GetResourceTypeRoles(ctx, visitor, searchInfo)
			assert.Error(t, err)
			assert.Equal(t, count, 0)
			assert.Equal(t, len(roles), 0)
		})

		Convey("获取所有角色失败", func() {
			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), []string{"test-resource-type"}).Return(map[string]interfaces.ResourceType{"test-resource-type": {ID: "test-resource-type"}}, nil)
			roleDB.EXPECT().GetAllUserRolesInternal(gomock.Any(), "test").Return(nil, errors.New("db error"))
			count, roles, err := r.GetResourceTypeRoles(ctx, visitor, searchInfo)
			assert.Error(t, err)
			assert.Equal(t, count, 0)
			assert.Equal(t, len(roles), 0)
		})

		Convey("成功获取角色 - 无匹配角色", func() {
			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), []string{"test-resource-type"}).Return(map[string]interfaces.ResourceType{"test-resource-type": {ID: "test-resource-type"}}, nil)
			roleDB.EXPECT().GetAllUserRolesInternal(gomock.Any(), "test").Return([]interfaces.RoleInfo{}, nil)
			count, roles, err := r.GetResourceTypeRoles(ctx, visitor, searchInfo)
			assert.NoError(t, err)
			assert.Equal(t, count, 0)
			assert.Equal(t, len(roles), 0)
		})

		Convey("成功获取角色 - 过滤内置角色", func() {
			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), []string{"test-resource-type"}).Return(map[string]interfaces.ResourceType{"test-resource-type": {ID: "test-resource-type"}}, nil)

			// 包含内置角色的测试数据
			testRoles := []interfaces.RoleInfo{
				{
					ID:          "00990824-4bf7-11f0-8fa7-865d5643e61f", // dataAdminRoleID
					Name:        "Data Admin",
					Description: "Data Admin Role",
					ModifyTime:  1000,
				},
				{
					ID:          "custom-role-1",
					Name:        "Custom Role 1",
					Description: "Custom Role 1",
					ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
						Unlimited: true,
					},
					ModifyTime: 2000,
				},
				{
					ID:          "custom-role-2",
					Name:        "Custom Role 2",
					Description: "Custom Role 2",
					ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
						Unlimited: false,
						Types: []interfaces.ResourceTypeScope{
							{ResourceTypeID: "test-resource-type"},
						},
					},
					ModifyTime: 1500,
				},
			}

			roleDB.EXPECT().GetAllUserRolesInternal(gomock.Any(), "test").Return(testRoles, nil)
			count, roles, err := r.GetResourceTypeRoles(ctx, visitor, searchInfo)
			assert.NoError(t, err)
			assert.Equal(t, count, 2) // 应该过滤掉内置角色
			assert.Equal(t, len(roles), 2)
			assert.Equal(t, roles[0].ID, "custom-role-1") // 按修改时间降序排序
			assert.Equal(t, roles[1].ID, "custom-role-2")
		})

		Convey("成功获取角色 - 分页处理", func() {
			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), []string{"test-resource-type"}).Return(map[string]interfaces.ResourceType{"test-resource-type": {ID: "test-resource-type"}}, nil)

			// 创建多个角色用于测试分页
			testRoles := make([]interfaces.RoleInfo, 0)
			for i := 1; i <= 15; i++ {
				testRoles = append(testRoles, interfaces.RoleInfo{
					ID:          fmt.Sprintf("role-%d", i),
					Name:        fmt.Sprintf("Role %d", i),
					Description: fmt.Sprintf("Role %d Description", i),
					ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
						Unlimited: true,
					},
					ModifyTime: int64(1000 + i),
				})
			}

			roleDB.EXPECT().GetAllUserRolesInternal(gomock.Any(), "test").Return(testRoles, nil)

			// 测试分页：offset=5, limit=5
			searchInfo.Offset = 5
			searchInfo.Limit = 5
			count, roles, err := r.GetResourceTypeRoles(ctx, visitor, searchInfo)
			assert.NoError(t, err)
			assert.Equal(t, count, 15)              // 总数
			assert.Equal(t, len(roles), 5)          // 分页后的数量
			assert.Equal(t, roles[0].ID, "role-10") // 按修改时间降序，offset=5后第一个应该是role-10
			assert.Equal(t, roles[4].ID, "role-6")  // 最后一个应该是role-6
		})

		Convey("成功获取角色 - 超出分页范围", func() {
			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), []string{"test-resource-type"}).Return(map[string]interfaces.ResourceType{"test-resource-type": {ID: "test-resource-type"}}, nil)

			testRoles := []interfaces.RoleInfo{
				{
					ID:          "role-1",
					Name:        "Role 1",
					Description: "Role 1 Description",
					ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
						Unlimited: true,
					},
					ModifyTime: 1000,
				},
			}

			roleDB.EXPECT().GetAllUserRolesInternal(gomock.Any(), "test").Return(testRoles, nil)

			// 测试超出范围的分页
			searchInfo.Offset = 10
			searchInfo.Limit = 5
			count, roles, err := r.GetResourceTypeRoles(ctx, visitor, searchInfo)
			assert.NoError(t, err)
			assert.Equal(t, count, 1)      // 总数
			assert.Equal(t, len(roles), 0) // 超出范围，返回空
		})

		Convey("成功获取角色 - 关键字过滤", func() {
			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), []string{"test-resource-type"}).Return(map[string]interfaces.ResourceType{"test-resource-type": {ID: "test-resource-type"}}, nil)

			testRoles := []interfaces.RoleInfo{
				{
					ID:          "admin-role",
					Name:        "Admin Role",
					Description: "Admin Role Description",
					ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
						Unlimited: true,
					},
					ModifyTime: 1000,
				},
			}

			roleDB.EXPECT().GetAllUserRolesInternal(gomock.Any(), "admin").Return(testRoles, nil)

			searchInfo.Keyword = "admin"
			count, roles, err := r.GetResourceTypeRoles(ctx, visitor, searchInfo)
			assert.NoError(t, err)
			assert.Equal(t, count, 1)
			assert.Equal(t, len(roles), 1)
			assert.Equal(t, roles[0].ID, "admin-role")
		})

		Convey("成功获取角色 - 资源类型范围匹配", func() {
			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), []string{"test-resource-type"}).Return(map[string]interfaces.ResourceType{"test-resource-type": {ID: "test-resource-type"}}, nil)

			testRoles := []interfaces.RoleInfo{
				{
					ID:          "unlimited-role",
					Name:        "Unlimited Role",
					Description: "Unlimited Role Description",
					ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
						Unlimited: true,
					},
					ModifyTime: 2000,
				},
				{
					ID:          "specific-role",
					Name:        "Specific Role",
					Description: "Specific Role Description",
					ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
						Unlimited: false,
						Types: []interfaces.ResourceTypeScope{
							{ResourceTypeID: "test-resource-type"},
						},
					},
					ModifyTime: 1000,
				},
				{
					ID:          "other-role",
					Name:        "Other Role",
					Description: "Other Role Description",
					ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
						Unlimited: false,
						Types: []interfaces.ResourceTypeScope{
							{ResourceTypeID: "other-resource-type"},
						},
					},
					ModifyTime: 1500,
				},
			}

			roleDB.EXPECT().GetAllUserRolesInternal(gomock.Any(), "test").Return(testRoles, nil)

			count, roles, err := r.GetResourceTypeRoles(ctx, visitor, searchInfo)
			assert.NoError(t, err)
			assert.Equal(t, count, 2) // unlimited-role 和 specific-role 应该被包含
			assert.Equal(t, len(roles), 2)
			assert.Equal(t, roles[0].ID, "unlimited-role") // 按修改时间降序
			assert.Equal(t, roles[1].ID, "specific-role")
		})

		Convey("成功获取角色 - 空关键字", func() {
			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), []string{"test-resource-type"}).Return(map[string]interfaces.ResourceType{"test-resource-type": {ID: "test-resource-type"}}, nil)

			testRoles := []interfaces.RoleInfo{
				{
					ID:          "role-1",
					Name:        "Role 1",
					Description: "Role 1 Description",
					ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
						Unlimited: true,
					},
					ModifyTime: 1000,
				},
			}

			roleDB.EXPECT().GetAllUserRolesInternal(gomock.Any(), "").Return(testRoles, nil)

			searchInfo.Keyword = ""
			count, roles, err := r.GetResourceTypeRoles(ctx, visitor, searchInfo)
			assert.NoError(t, err)
			assert.Equal(t, count, 1)
			assert.Equal(t, len(roles), 1)
			assert.Equal(t, roles[0].ID, "role-1")
		})
	})
}

//nolint:errcheck,funlen,staticcheck
func TestRole_GetAccessorRoles(t *testing.T) {
	Convey("测试GetAccessorRoles方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		roleDB := mock.NewMockDBRole(ctrl)
		roleMemberDB := mock.NewMockDBRoleMember(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		logger := common.NewLogger()
		event := mock.NewMockLogicsEvent(ctrl)
		r := newRole(roleDB, roleMemberDB, userMgnt, logger, event)
		r.initRoleOrder()
		ctx := context.Background()
		param := interfaces.AccessorRoleSearchInfo{
			AccessorID:   "user1",
			AccessorType: interfaces.AccessorUser,
			Offset:       0,
			Limit:        10,
		}

		Convey("GetAccessorIDsByUserID失败", func() {
			userMgnt.EXPECT().GetAccessorIDsByUserID(gomock.Any(), "user1").Return(nil, errors.New("获取访问令牌失败"))
			count, roles, err := r.GetAccessorRoles(ctx, param)
			assert.Error(t, err)
			assert.Equal(t, 0, count)
			assert.Nil(t, roles)
		})

		Convey("GetRoleByMembers失败", func() {
			userMgnt.EXPECT().GetAccessorIDsByUserID(gomock.Any(), "user1").Return([]string{"accessor1"}, nil)
			roleMemberDB.EXPECT().GetRoleByMembers(gomock.Any(), gomock.Any()).Return(nil, errors.New("获取角色失败"))
			count, roles, err := r.GetAccessorRoles(ctx, param)
			assert.Error(t, err)
			assert.Equal(t, 0, count)
			assert.Nil(t, roles)
		})

		Convey("成功获取角色 - 不包含系统角色", func() {
			userMgnt.EXPECT().GetAccessorIDsByUserID(gomock.Any(), "user1").Return([]string{"accessor1"}, nil)
			testRoles := []interfaces.RoleInfo{
				{
					ID:          "role1",
					Name:        "Role 1",
					Description: "Role 1 Description",
					RoleSource:  interfaces.RoleSourceBusiness,
					ModifyTime:  1000,
				},
				{
					ID:          "role2",
					Name:        "Role 2",
					Description: "Role 2 Description",
					RoleSource:  interfaces.RoleSourceUser,
					ModifyTime:  2000,
				},
			}
			roleMemberDB.EXPECT().GetRoleByMembers(gomock.Any(), gomock.Any()).Return(testRoles, nil)
			roleDB.EXPECT().GetRoleByIDs(gomock.Any(), []string{"role1", "role2"}).Return(map[string]interfaces.RoleInfo{
				"role1": testRoles[0],
				"role2": testRoles[1],
			}, nil)

			count, roles, err := r.GetAccessorRoles(ctx, param)
			assert.NoError(t, err)
			assert.Equal(t, 2, count)
			assert.Equal(t, 2, len(roles))
			// 按修改时间降序排序
			assert.Equal(t, "role2", roles[0].ID)
			assert.Equal(t, "role1", roles[1].ID)
		})

		Convey("成功获取角色 - 包含系统角色", func() {
			param.RoleSources = []interfaces.RoleSource{interfaces.RoleSourceSystem, interfaces.RoleSourceBusiness}
			userMgnt.EXPECT().GetAccessorIDsByUserID(gomock.Any(), "user1").Return([]string{"accessor1"}, nil)
			testRoles := []interfaces.RoleInfo{
				{
					ID:          "role1",
					Name:        "Role 1",
					Description: "Role 1 Description",
					RoleSource:  interfaces.RoleSourceBusiness,
					ModifyTime:  1000,
				},
			}
			roleMemberDB.EXPECT().GetRoleByMembers(gomock.Any(), gomock.Any()).Return(testRoles, nil)
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), "user1").Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			roleDB.EXPECT().GetRoleByIDs(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, roleIDs []string) (map[string]interfaces.RoleInfo, error) {
				result := make(map[string]interfaces.RoleInfo)
				for _, roleID := range roleIDs {
					if roleID == "role1" {
						result[roleID] = testRoles[0]
					} else if roleID == superAdminRoleID {
						result[roleID] = interfaces.RoleInfo{
							ID:         superAdminRoleID,
							Name:       "Super Admin",
							RoleSource: interfaces.RoleSourceSystem,
							ModifyTime: 3000,
						}
					}
				}
				return result, nil
			})

			count, roles, err := r.GetAccessorRoles(ctx, param)
			assert.NoError(t, err)
			assert.Equal(t, 2, count)
			assert.Equal(t, 2, len(roles))
		})

		Convey("成功获取角色 - 包含所有系统角色类型", func() {
			param.RoleSources = []interfaces.RoleSource{interfaces.RoleSourceSystem}
			userMgnt.EXPECT().GetAccessorIDsByUserID(gomock.Any(), "user1").Return([]string{"accessor1"}, nil)
			roleMemberDB.EXPECT().GetRoleByMembers(gomock.Any(), gomock.Any()).Return([]interfaces.RoleInfo{}, nil)
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), "user1").Return([]interfaces.SystemRoleType{
				interfaces.SuperAdmin,
				interfaces.SystemAdmin,
				interfaces.SecurityAdmin,
				interfaces.AuditAdmin,
				interfaces.OrganizationAdmin,
				interfaces.OrganizationAudit,
				interfaces.NormalUser, // NormalUser 不应该添加角色ID
			}, nil)
			roleDB.EXPECT().GetRoleByIDs(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, roleIDs []string) (map[string]interfaces.RoleInfo, error) {
				result := make(map[string]interfaces.RoleInfo)
				for _, roleID := range roleIDs {
					result[roleID] = interfaces.RoleInfo{
						ID:         roleID,
						Name:       "System Role",
						RoleSource: interfaces.RoleSourceSystem,
						ModifyTime: 1000,
					}
				}
				return result, nil
			})

			count, roles, err := r.GetAccessorRoles(ctx, param)
			assert.NoError(t, err)
			// 应该有6个系统角色（不包括NormalUser）
			assert.Equal(t, 6, count)
			assert.Equal(t, 6, len(roles))
		})

		Convey("GetUserRolesByUserID失败 - 包含系统角色", func() {
			param.RoleSources = []interfaces.RoleSource{interfaces.RoleSourceSystem}
			userMgnt.EXPECT().GetAccessorIDsByUserID(gomock.Any(), "user1").Return([]string{"accessor1"}, nil)
			roleMemberDB.EXPECT().GetRoleByMembers(gomock.Any(), gomock.Any()).Return([]interfaces.RoleInfo{}, nil)
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), "user1").Return(nil, errors.New("获取用户角色失败"))
			count, roles, err := r.GetAccessorRoles(ctx, param)
			assert.Error(t, err)
			assert.Equal(t, 0, count)
			assert.Nil(t, roles)
		})

		Convey("GetRoleByIDs失败", func() {
			userMgnt.EXPECT().GetAccessorIDsByUserID(gomock.Any(), "user1").Return([]string{"accessor1"}, nil)
			testRoles := []interfaces.RoleInfo{
				{
					ID:          "role1",
					Name:        "Role 1",
					Description: "Role 1 Description",
					RoleSource:  interfaces.RoleSourceBusiness,
					ModifyTime:  1000,
				},
			}
			roleMemberDB.EXPECT().GetRoleByMembers(gomock.Any(), gomock.Any()).Return(testRoles, nil)
			roleDB.EXPECT().GetRoleByIDs(gomock.Any(), []string{"role1"}).Return(nil, errors.New("获取角色信息失败"))
			count, roles, err := r.GetAccessorRoles(ctx, param)
			assert.Error(t, err)
			assert.Equal(t, 0, count)
			assert.Nil(t, roles)
		})

		Convey("成功获取角色 - 角色来源过滤", func() {
			param.RoleSources = []interfaces.RoleSource{interfaces.RoleSourceBusiness}
			userMgnt.EXPECT().GetAccessorIDsByUserID(gomock.Any(), "user1").Return([]string{"accessor1"}, nil)
			testRoles := []interfaces.RoleInfo{
				{
					ID:          "role1",
					Name:        "Role 1",
					Description: "Role 1 Description",
					RoleSource:  interfaces.RoleSourceBusiness,
					ModifyTime:  1000,
				},
				{
					ID:          "role2",
					Name:        "Role 2",
					Description: "Role 2 Description",
					RoleSource:  interfaces.RoleSourceUser,
					ModifyTime:  2000,
				},
				{
					ID:          "role3",
					Name:        "Role 3",
					Description: "Role 3 Description",
					RoleSource:  interfaces.RoleSourceSystem,
					ModifyTime:  3000,
				},
			}
			roleMemberDB.EXPECT().GetRoleByMembers(gomock.Any(), gomock.Any()).Return(testRoles, nil)
			roleDB.EXPECT().GetRoleByIDs(gomock.Any(), []string{"role1", "role2", "role3"}).Return(map[string]interfaces.RoleInfo{
				"role1": testRoles[0],
				"role2": testRoles[1],
				"role3": testRoles[2],
			}, nil)

			count, roles, err := r.GetAccessorRoles(ctx, param)
			assert.NoError(t, err)
			// 只应该返回业务角色
			assert.Equal(t, 1, count)
			assert.Equal(t, 1, len(roles))
			assert.Equal(t, "role1", roles[0].ID)
		})

		Convey("成功获取角色 - 空角色来源使用默认值", func() {
			param.RoleSources = []interfaces.RoleSource{}
			userMgnt.EXPECT().GetAccessorIDsByUserID(gomock.Any(), "user1").Return([]string{"accessor1"}, nil)
			testRoles := []interfaces.RoleInfo{
				{
					ID:          "role1",
					Name:        "Role 1",
					Description: "Role 1 Description",
					RoleSource:  interfaces.RoleSourceBusiness,
					ModifyTime:  1000,
				},
				{
					ID:          "role2",
					Name:        "Role 2",
					Description: "Role 2 Description",
					RoleSource:  interfaces.RoleSourceUser,
					ModifyTime:  2000,
				},
				{
					ID:          "role3",
					Name:        "Role 3",
					Description: "Role 3 Description",
					RoleSource:  interfaces.RoleSourceSystem,
					ModifyTime:  3000,
				},
			}
			roleMemberDB.EXPECT().GetRoleByMembers(gomock.Any(), gomock.Any()).Return(testRoles, nil)
			roleDB.EXPECT().GetRoleByIDs(gomock.Any(), []string{"role1", "role2", "role3"}).Return(map[string]interfaces.RoleInfo{
				"role1": testRoles[0],
				"role2": testRoles[1],
				"role3": testRoles[2],
			}, nil)

			count, roles, err := r.GetAccessorRoles(ctx, param)
			assert.NoError(t, err)
			// 默认应该返回业务角色和用户角色
			assert.Equal(t, 2, count)
			assert.Equal(t, 2, len(roles))
		})

		Convey("成功获取角色 - 分页", func() {
			param.Offset = 1
			param.Limit = 2
			userMgnt.EXPECT().GetAccessorIDsByUserID(gomock.Any(), "user1").Return([]string{"accessor1"}, nil)
			testRoles := make([]interfaces.RoleInfo, 0)
			for i := 1; i <= 5; i++ {
				testRoles = append(testRoles, interfaces.RoleInfo{
					ID:          fmt.Sprintf("role%d", i),
					Name:        fmt.Sprintf("Role %d", i),
					Description: fmt.Sprintf("Role %d Description", i),
					RoleSource:  interfaces.RoleSourceBusiness,
					ModifyTime:  int64(1000 + i),
				})
			}
			roleMemberDB.EXPECT().GetRoleByMembers(gomock.Any(), gomock.Any()).Return(testRoles, nil)
			roleMap := make(map[string]interfaces.RoleInfo)
			for _, role := range testRoles {
				roleMap[role.ID] = role
			}
			roleDB.EXPECT().GetRoleByIDs(gomock.Any(), gomock.Any()).Return(roleMap, nil)

			count, roles, err := r.GetAccessorRoles(ctx, param)
			assert.NoError(t, err)
			assert.Equal(t, 5, count)      // 总数
			assert.Equal(t, 2, len(roles)) // 分页后的数量
			// 按修改时间降序，offset=1, limit=2
			assert.Equal(t, "role4", roles[0].ID)
			assert.Equal(t, "role3", roles[1].ID)
		})

		Convey("成功获取角色 - 分页limit=-1返回所有", func() {
			param.Offset = 0
			param.Limit = -1
			userMgnt.EXPECT().GetAccessorIDsByUserID(gomock.Any(), "user1").Return([]string{"accessor1"}, nil)
			testRoles := make([]interfaces.RoleInfo, 0)
			for i := 1; i <= 3; i++ {
				testRoles = append(testRoles, interfaces.RoleInfo{
					ID:          fmt.Sprintf("role%d", i),
					Name:        fmt.Sprintf("Role %d", i),
					Description: fmt.Sprintf("Role %d Description", i),
					RoleSource:  interfaces.RoleSourceBusiness,
					ModifyTime:  int64(1000 + i),
				})
			}
			roleMemberDB.EXPECT().GetRoleByMembers(gomock.Any(), gomock.Any()).Return(testRoles, nil)
			roleMap := make(map[string]interfaces.RoleInfo)
			for _, role := range testRoles {
				roleMap[role.ID] = role
			}
			roleDB.EXPECT().GetRoleByIDs(gomock.Any(), gomock.Any()).Return(roleMap, nil)

			count, roles, err := r.GetAccessorRoles(ctx, param)
			assert.NoError(t, err)
			assert.Equal(t, 3, count)
			assert.Equal(t, 3, len(roles)) // limit=-1 时返回所有
		})

		Convey("成功获取角色 - 分页超出范围", func() {
			param.Offset = 10
			param.Limit = 5
			userMgnt.EXPECT().GetAccessorIDsByUserID(gomock.Any(), "user1").Return([]string{"accessor1"}, nil)
			testRoles := []interfaces.RoleInfo{
				{
					ID:          "role1",
					Name:        "Role 1",
					Description: "Role 1 Description",
					RoleSource:  interfaces.RoleSourceBusiness,
					ModifyTime:  1000,
				},
			}
			roleMemberDB.EXPECT().GetRoleByMembers(gomock.Any(), gomock.Any()).Return(testRoles, nil)
			roleDB.EXPECT().GetRoleByIDs(gomock.Any(), []string{"role1"}).Return(map[string]interfaces.RoleInfo{
				"role1": testRoles[0],
			}, nil)

			count, roles, err := r.GetAccessorRoles(ctx, param)
			assert.NoError(t, err)
			assert.Equal(t, 1, count)
			assert.Equal(t, 0, len(roles)) // 超出范围，返回空
		})

		Convey("成功获取角色 - 空结果", func() {
			userMgnt.EXPECT().GetAccessorIDsByUserID(gomock.Any(), "user1").Return([]string{"accessor1"}, nil)
			roleMemberDB.EXPECT().GetRoleByMembers(gomock.Any(), gomock.Any()).Return([]interfaces.RoleInfo{}, nil)
			// 不包含系统角色，所以不会调用 GetUserRolesByUserID

			count, roles, err := r.GetAccessorRoles(ctx, param)
			assert.NoError(t, err)
			assert.Equal(t, 0, count)
			assert.Equal(t, 0, len(roles))
		})
	})
}
