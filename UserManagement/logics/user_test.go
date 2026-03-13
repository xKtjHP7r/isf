package logics

import (
	"context"
	"fmt"
	"testing"
	"time"

	stdErr "errors"

	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/go-lib/rest"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/proton-rds-sdk-go/sqlx"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/assert"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"

	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/common"
	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/errors"
	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/interfaces"
	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/interfaces/mock"
)

const (
	cRSAPWD = "h/bpbcxyhcqGoXju+RdhdmK0ZMxCJU1xfZQzHSR41Lvn8OwFrikfMB2sqz/FBbI+kwXpLmDZKCmx8loIwsHJxm3v1B61BTDhss9m4i+YBcw1V7BJHCR4DFEzzVg4luujh13YhlYeCBfoQL0di1GbiWJcCXMAL3cpwmLoogoOYxE="
	strTest = "test"
)

func newUser(userDB interfaces.DBUser, departmentDB interfaces.DBDepartment, contactorDB interfaces.DBContactor,
	groupMemberDB interfaces.DBGroupMember, role interfaces.LogicsRole) *user {
	return &user{
		userDB:        userDB,
		departmentDB:  departmentDB,
		contactorDB:   contactorDB,
		groupMemberDB: groupMemberDB,
		role:          role,
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
	}
}

func TestConvertUserName(t *testing.T) {
	Convey("ConvertUserName, db is available", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		departmentDB := mock.NewMockDBDepartment(ctrl)
		contactorDB := mock.NewMockDBContactor(ctrl)
		groupMemberDB := mock.NewMockDBGroupMember(ctrl)
		userLogics := newUser(userDB, departmentDB, contactorDB, groupMemberDB, nil)
		userIDs := make([]string, 0)
		visitor := interfaces.Visitor{}

		Convey("user id is empty", func() {
			nameInfos, err := userLogics.ConvertUserName(&visitor, userIDs, true)
			assert.Equal(t, len(nameInfos), 0)
			assert.Equal(t, err, nil)
		})

		userIDs = append(userIDs, "user_id")
		Convey("DB GetUserName error", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			userDB.EXPECT().GetUserName(gomock.Any()).AnyTimes().Return(nil, nil, testErr)
			nameInfos, err := userLogics.ConvertUserName(&visitor, userIDs, true)
			assert.Equal(t, nameInfos, nil)
			assert.Equal(t, err, testErr)
		})

		Convey("user is not exist", func() {
			testErr := rest.NewHTTPErrorV2(errors.UserNotFound, userLogics.i18n.Load(i18nIDObjectsInUserNotFound, visitor.Language),
				rest.SetCodeStr(errors.StrBadRequestUserNotFound),
				rest.SetDetail(map[string]interface{}{"ids": []string{"user_id"}}))
			userDB.EXPECT().GetUserName(gomock.Any()).AnyTimes().Return(nil, nil, nil)
			nameInfos, err := userLogics.ConvertUserName(&visitor, userIDs, true)
			assert.Equal(t, nameInfos, nil)
			assert.Equal(t, err, testErr)
		})

		Convey("success", func() {
			testNameInfo := make(map[string]interfaces.UserDBInfo)
			testLogics := interfaces.UserDBInfo{ID: "user_id", Name: "user_name"}
			tmpName := interfaces.NameInfo{ID: "user_id", Name: "user_name"}
			testNameInfo[testLogics.ID] = testLogics
			testInfoMap := make([]interfaces.UserDBInfo, 0)
			testInfoMap = append(testInfoMap, testLogics)
			userDB.EXPECT().GetUserName(gomock.Any()).AnyTimes().Return(testInfoMap, []string{testLogics.Name}, nil).Times(1)
			nameInfos, err := userLogics.ConvertUserName(&visitor, userIDs, true)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(nameInfos), 1)
			assert.Equal(t, nameInfos[0], tmpName)
		})

		Convey("success1", func() {
			testLogics := interfaces.UserDBInfo{ID: "user_id", Name: "user_name"}
			tmpName := interfaces.NameInfo{ID: "user_id", Name: "user_name"}
			testInfoMap := make([]interfaces.UserDBInfo, 0)
			testInfoMap = append(testInfoMap, testLogics)
			userDB.EXPECT().GetUserName(gomock.Any()).AnyTimes().Return(testInfoMap, []string{testLogics.Name}, nil).Times(1)
			nameInfos, err := userLogics.ConvertUserName(&visitor, []string{strID, strID2}, false)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(nameInfos), 1)
			assert.Equal(t, nameInfos[0], tmpName)
		})
	})
}

func TestGetUserEmails(t *testing.T) {
	Convey("GetUserEmails, db is available", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		departmentDB := mock.NewMockDBDepartment(ctrl)
		contactorDB := mock.NewMockDBContactor(ctrl)
		groupMemberDB := mock.NewMockDBGroupMember(ctrl)
		userLogics := newUser(userDB, departmentDB, contactorDB, groupMemberDB, nil)
		userIDs := make([]string, 0)
		visitor := interfaces.Visitor{
			Language: interfaces.AmericanEnglish,
		}

		Convey("user id is empty", func() {
			nameInfos, err := userLogics.GetUserEmails(&visitor, userIDs)
			assert.Equal(t, len(nameInfos), 0)
			assert.Equal(t, err, nil)
		})

		userIDs = append(userIDs, "user_id")
		Convey("DB GetUserDBInfo error", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).AnyTimes().Return(nil, testErr)
			nameInfos, err := userLogics.GetUserEmails(&visitor, userIDs)
			assert.Equal(t, nameInfos, nil)
			assert.Equal(t, err, testErr)
		})

		Convey("user is not exist", func() {
			testErr := rest.NewHTTPErrorV2(errors.UserNotFound, userLogics.i18n.Load(i18nIDObjectsInUserNotFound, visitor.Language),
				rest.SetCodeStr(errors.StrBadRequestUserNotFound),
				rest.SetDetail(map[string]interface{}{"ids": []string{"user_id"}}))
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).AnyTimes().Return(nil, nil)
			nameInfos, err := userLogics.GetUserEmails(&visitor, userIDs)
			assert.Equal(t, nameInfos, nil)
			assert.Equal(t, err, testErr)
		})

		Convey("success", func() {
			testNameInfo := make(map[string]interfaces.UserDBInfo)
			testLogics := interfaces.UserDBInfo{ID: "user_id", Email: "user_name"}
			tmpName := interfaces.EmailInfo{ID: "user_id", Email: "user_name"}
			testNameInfo[testLogics.ID] = testLogics
			testInfoMap := make([]interfaces.UserDBInfo, 0)
			testInfoMap = append(testInfoMap, testLogics)
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).AnyTimes().Return(testInfoMap, nil).Times(1)
			nameInfos, err := userLogics.GetUserEmails(&visitor, userIDs)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(nameInfos), 1)
			assert.Equal(t, nameInfos[0], tmpName)
		})
	})
}

func TestGetAllBelongDepartmentIDs(t *testing.T) {
	Convey("GetAllBelongDepartmentIDs, db is available", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		departmentDB := mock.NewMockDBDepartment(ctrl)
		contactorDB := mock.NewMockDBContactor(ctrl)
		groupMemberDB := mock.NewMockDBGroupMember(ctrl)
		userLogics := newUser(userDB, departmentDB, contactorDB, groupMemberDB, nil)
		userID := strTest

		Convey("GetUserPath is error", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			userDB.EXPECT().GetUsersPath(gomock.Any()).AnyTimes().Return(nil, testErr)
			deptIDS, err := userLogics.GetAllBelongDepartmentIDs(userID)
			assert.Equal(t, deptIDS, nil)
			assert.Equal(t, err, testErr)
		})

		Convey("如果用户在未分配组，则跳过", func() {
			tempPaths := []string{"-1"}
			userDB.EXPECT().GetUsersPath(gomock.Any()).AnyTimes().Return(map[string][]string{userID: tempPaths}, nil)
			deptIDS, err := userLogics.GetAllBelongDepartmentIDs(userID)
			assert.Equal(t, len(deptIDS), 0)
			assert.Equal(t, err, nil)
		})

		Convey("success", func() {
			path1 := "depart1/depart2/depart3"
			path2 := "depart2/depart3"
			testPaths := []string{path1, path2}
			userDB.EXPECT().GetUsersPath(gomock.Any()).AnyTimes().Return(map[string][]string{userID: testPaths}, nil)
			deptIDS, err := userLogics.GetAllBelongDepartmentIDs(userID)
			assert.Equal(t, len(deptIDS), 3)
			assert.Equal(t, err, nil)

			test1 := make(map[string]bool)
			for _, v := range deptIDS {
				test1[v] = true
			}
			assert.Equal(t, test1["depart1"], true)
			assert.Equal(t, test1["depart2"], true)
			assert.Equal(t, test1["depart3"], true)
		})
	})
}

func TestGetUsersInDepartments(t *testing.T) {
	Convey("GetUsersInDepartments, db is available", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		departmentDB := mock.NewMockDBDepartment(ctrl)
		contactorDB := mock.NewMockDBContactor(ctrl)
		groupMemberDB := mock.NewMockDBGroupMember(ctrl)
		userLogics := newUser(userDB, departmentDB, contactorDB, groupMemberDB, nil)

		var userIDs []string
		var depIDs []string

		Convey("userIDs is empty", func() {
			outInfo, err := userLogics.GetUsersInDepartments(userIDs, depIDs)
			assert.Equal(t, len(outInfo), 0)
			assert.Equal(t, err, nil)
		})

		userIDs = append(userIDs, "xxxx")
		Convey("depIDs is empty", func() {
			outInfo, err := userLogics.GetUsersInDepartments(userIDs, depIDs)
			assert.Equal(t, len(outInfo), 0)
			assert.Equal(t, err, nil)
		})

		depIDs = append(depIDs, "GetUsersInDepartments")
		Convey("GetParentDepartmentID is error", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			userDB.EXPECT().GetUsersInDepartments(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, testErr)
			outInfo, err := userLogics.GetUsersInDepartments(userIDs, depIDs)
			assert.Equal(t, len(outInfo), 0)
			assert.Equal(t, err, testErr)
		})

		Convey("success", func() {
			tmpOutInfo := []string{
				0: "zzzzz",
			}
			userDB.EXPECT().GetUsersInDepartments(gomock.Any(), gomock.Any()).AnyTimes().Return(tmpOutInfo, nil)
			outInfo, err := userLogics.GetUsersInDepartments(userIDs, depIDs)
			assert.Equal(t, len(outInfo), 1)
			assert.Equal(t, outInfo[0], "zzzzz")
			assert.Equal(t, err, nil)
		})
	})
}

func TestGetAccessorIDsOfUser(t *testing.T) {
	Convey("GetAccessorIDsOfUser, db is available", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		departmentDB := mock.NewMockDBDepartment(ctrl)
		contactorDB := mock.NewMockDBContactor(ctrl)
		groupMemberDB := mock.NewMockDBGroupMember(ctrl)
		internalGroup := mock.NewMockLogicsInternalGroup(ctrl)
		userLogics := &user{
			userDB:        userDB,
			departmentDB:  departmentDB,
			contactorDB:   contactorDB,
			groupMemberDB: groupMemberDB,
			role:          nil,
			internalGroup: internalGroup,
		}

		userID := "xxxx1"
		Convey("GetUserName is error", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			userDB.EXPECT().GetUserName(gomock.Any()).AnyTimes().Return(nil, nil, testErr)
			outInfo, err := userLogics.GetAccessorIDsOfUser(userID)
			assert.Equal(t, len(outInfo), 0)
			assert.Equal(t, err, testErr)
		})

		Convey("userID is not exsit", func() {
			userDB.EXPECT().GetUserName(gomock.Any()).AnyTimes().Return(nil, nil, nil)
			outInfo, err := userLogics.GetAccessorIDsOfUser(userID)
			assert.Equal(t, len(outInfo), 0)
			assert.Equal(t, err, rest.NewHTTPErrorV2(errors.NotFound, "user does not exist"))
		})

		userNameInfo := interfaces.UserDBInfo{
			ID:   userID,
			Name: "zzzxxxff",
		}
		userInfos := []interfaces.UserDBInfo{
			userNameInfo,
		}
		path := "xxxx"
		paths := []string{path}
		Convey("GetAllBelongDepartmentIDs is error", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			userDB.EXPECT().GetUserName(gomock.Any()).AnyTimes().Return(userInfos, []string{userID}, nil)
			userDB.EXPECT().GetUsersPath(gomock.Any()).AnyTimes().Return(map[string][]string{userID: paths}, testErr)
			outInfo, err := userLogics.GetAccessorIDsOfUser(userID)
			assert.Equal(t, len(outInfo), 0)
			assert.Equal(t, err, testErr)
		})

		Convey("GetUserAllBelongContactorIDs is error", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			userDB.EXPECT().GetUserName(gomock.Any()).AnyTimes().Return(userInfos, []string{userID}, nil)
			testDeptsID := []string{0: strTest}
			userDB.EXPECT().GetDirectBelongDepartmentIDs(gomock.Any()).AnyTimes().Return(testDeptsID, nil, nil)
			userDB.EXPECT().GetUsersPath(gomock.Any()).AnyTimes().Return(map[string][]string{userID: paths}, nil)
			contactorDB.EXPECT().GetUserAllBelongContactorIDs(gomock.Any()).AnyTimes().Return(nil, testErr)
			outInfo, err := userLogics.GetAccessorIDsOfUser(userID)
			assert.Equal(t, len(outInfo), 0)
			assert.Equal(t, err, testErr)
		})

		Convey("GetMembersBelongGroupIDs is error", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			userDB.EXPECT().GetUserName(gomock.Any()).AnyTimes().Return(userInfos, []string{userID}, nil)
			testDeptsID := []string{0: strTest}
			userDB.EXPECT().GetDirectBelongDepartmentIDs(gomock.Any()).AnyTimes().Return(testDeptsID, nil, nil)
			userDB.EXPECT().GetUsersPath(gomock.Any()).AnyTimes().Return(map[string][]string{userID: paths}, nil)
			contactorDB.EXPECT().GetUserAllBelongContactorIDs(gomock.Any()).AnyTimes().Return([]string{"zzzz"}, nil)
			groupMemberDB.EXPECT().GetMembersBelongGroupIDs(gomock.Any()).AnyTimes().Return(nil, nil, testErr)
			outInfo, err := userLogics.GetAccessorIDsOfUser(userID)
			assert.Equal(t, len(outInfo), 0)
			assert.Equal(t, err, testErr)
		})

		testErr := rest.NewHTTPError("error", 503000000, nil)
		Convey("GetMembersBelongGroupIDs error", func() {
			userDB.EXPECT().GetUserName(gomock.Any()).AnyTimes().Return(userInfos, []string{userID}, nil)
			testDeptsID := []string{0: strTest}
			userDB.EXPECT().GetDirectBelongDepartmentIDs(gomock.Any()).AnyTimes().Return(testDeptsID, nil, nil)
			userDB.EXPECT().GetUsersPath(gomock.Any()).AnyTimes().Return(map[string][]string{userID: paths}, nil)
			departmentDB.EXPECT().GetParentDepartmentID(gomock.Any()).AnyTimes().Return(nil, nil)
			contactorDB.EXPECT().GetUserAllBelongContactorIDs(gomock.Any()).AnyTimes().Return([]string{"zzzz"}, nil)
			groupMemberDB.EXPECT().GetMembersBelongGroupIDs(gomock.Any()).AnyTimes().Return([]string{"kkkk"}, nil, nil)
			internalGroup.EXPECT().GetBelongGroups(gomock.Any()).AnyTimes().Return(nil, testErr)

			_, err := userLogics.GetAccessorIDsOfUser(userID)
			assert.Equal(t, err, testErr)
		})

		Convey("success", func() {
			userDB.EXPECT().GetUserName(gomock.Any()).AnyTimes().Return(userInfos, []string{userID}, nil)
			testDeptsID := []string{0: strTest}
			userDB.EXPECT().GetDirectBelongDepartmentIDs(gomock.Any()).AnyTimes().Return(testDeptsID, nil, nil)
			userDB.EXPECT().GetUsersPath(gomock.Any()).AnyTimes().Return(map[string][]string{userID: paths}, nil)
			contactorDB.EXPECT().GetUserAllBelongContactorIDs(gomock.Any()).AnyTimes().Return([]string{"zzzz"}, nil)
			groupMemberDB.EXPECT().GetMembersBelongGroupIDs(gomock.Any()).AnyTimes().Return([]string{"kkkk"}, nil, nil)
			internalGroup.EXPECT().GetBelongGroups(gomock.Any()).AnyTimes().Return([]string{"mmmm"}, nil)

			outInfo, err := userLogics.GetAccessorIDsOfUser(userID)

			testInfo := []string{
				userID,
				path,
				"zzzz",
				"kkkk",
				"mmmm",
			}
			assert.Equal(t, outInfo, testInfo)
			assert.Equal(t, err, nil)
		})
	})
}

func TestGetUserBaseInfoInScope(t *testing.T) {
	Convey("获取范围内用户信息-逻辑层", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		departmentDB := mock.NewMockDBDepartment(ctrl)
		contactorDB := mock.NewMockDBContactor(ctrl)
		groupMemberDB := mock.NewMockDBGroupMember(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		userLogics := newUser(userDB, departmentDB, contactorDB, groupMemberDB, role)

		userID := "xxxx"
		roleID := interfaces.SystemRoleSuperAdmin
		var rangeInfo interfaces.UserBaseInfoRange
		var userInfo interfaces.UserDBInfo
		mapRoles := make(map[interfaces.Role]bool)
		var visitor interfaces.Visitor
		visitor.ID = "zz13"

		roleInfo := make(map[string]map[interfaces.Role]bool)
		roleInfo[visitor.ID] = mapRoles
		Convey("获取用户角色失败-报错", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfo, testErr)
			_, err := userLogics.GetUserBaseInfoInScope(&visitor, roleID, []string{userID}, rangeInfo)
			assert.Equal(t, err, testErr)
		})

		Convey("调用者没有指定角色-报错", func() {
			testErr := rest.NewHTTPError("this user do not has this role", rest.BadRequest, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfo, nil)
			_, err := userLogics.GetUserBaseInfoInScope(&visitor, roleID, []string{userID}, rangeInfo)
			assert.Equal(t, err, testErr)
		})

		mapRoles[interfaces.SystemRoleSuperAdmin] = true
		Convey("获取用户信息失败-报错", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfo, nil)
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).AnyTimes().Return([]interfaces.UserDBInfo{}, testErr)
			_, err := userLogics.GetUserBaseInfoInScope(&visitor, roleID, []string{userID}, rangeInfo)
			assert.Equal(t, err, testErr)
		})

		Convey("用户不存在-报错", func() {
			testErr := rest.NewHTTPErrorV2(errors.NotFound, "user does not exist")
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfo, nil)
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).AnyTimes().Return([]interfaces.UserDBInfo{}, nil)
			_, err := userLogics.GetUserBaseInfoInScope(&visitor, roleID, []string{userID}, rangeInfo)
			assert.Equal(t, err, testErr)
		})

		Convey("用户不在指定范围之内-报错", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfo, nil)
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).AnyTimes().Return([]interfaces.UserDBInfo{userInfo}, nil)
			userDB.EXPECT().GetDirectBelongDepartmentIDs(gomock.Any()).AnyTimes().Return(nil, nil, testErr)
			_, err := userLogics.GetUserBaseInfoInScope(&visitor, roleID, []string{userID}, rangeInfo)
			assert.Equal(t, err, testErr)
		})

		temp1 := interfaces.UserDBInfo{
			ID:   "ID1",
			Name: "Name1",
		}
		temp2 := interfaces.UserDBInfo{
			ID:   "ID2",
			Name: "Name2",
		}
		Convey("获取用户父部门信息失败-报错", func() {
			rangeInfo.ShowParentDepPaths = true
			rangeInfo.ShowRoles = true
			roleID = interfaces.SystemRoleSuperAdmin
			testErr := rest.NewHTTPError("error", 503000000, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfo, nil)
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).AnyTimes().Return([]interfaces.UserDBInfo{userInfo}, nil)
			userDB.EXPECT().GetDirectBelongDepartmentIDs(gomock.Any()).AnyTimes().Times(1).Return([]string{"xxxx"}, nil, nil)
			userDB.EXPECT().GetDirectBelongDepartmentIDs(gomock.Any()).AnyTimes().Times(1).Return([]string{}, nil, testErr)
			_, err := userLogics.GetUserBaseInfoInScope(&visitor, roleID, []string{userID}, rangeInfo)
			assert.Equal(t, err, testErr)
		})

		userID1 := "zzzz"
		Convey("接口调用成功", func() {
			rangeInfo.ShowParentDepPaths = true
			rangeInfo.ShowRoles = true
			rangeInfo.ShowName = true
			roleID = interfaces.SystemRoleSuperAdmin
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfo, nil)
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).AnyTimes().Return([]interfaces.UserDBInfo{temp1, temp2}, nil)
			userDB.EXPECT().GetDirectBelongDepartmentIDs(gomock.Any()).AnyTimes().Return([]string{}, nil, nil)
			out, err := userLogics.GetUserBaseInfoInScope(&visitor, roleID, []string{userID, userID1}, rangeInfo)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(out), 2)
			assert.Equal(t, out[0].ID, temp1.ID)
			assert.Equal(t, out[1].ID, temp2.ID)
			assert.Equal(t, out[0].Name, temp1.Name)
			assert.Equal(t, out[1].Name, temp2.Name)
		})
	})
}

func TestSearchUsersByKey(t *testing.T) {
	Convey("用户搜索-逻辑层", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		departmentDB := mock.NewMockDBDepartment(ctrl)
		contactorDB := mock.NewMockDBContactor(ctrl)
		groupMemberDB := mock.NewMockDBGroupMember(ctrl)
		orgPerm := mock.NewMockLogicsOrgPerm(ctrl)
		trace := mock.NewMockTraceClient(ctrl)
		userLogics := newUser(userDB, departmentDB, contactorDB, groupMemberDB, nil)
		userLogics.trace = trace
		userLogics.orgPerm = orgPerm

		var info interfaces.OrgShowPageInfo
		info.Role = interfaces.SystemRoleOrgAudit
		var visitor interfaces.Visitor
		ctx := context.Background()
		Convey("获取组织审计员审计范围失败-报错", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			testErr := rest.NewHTTPError("error", 503000000, nil)
			userDB.EXPECT().GetOrgAduitDepartInfo2(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, testErr)

			_, _, err := userLogics.SearchUsersByKeyInScope(ctx, &visitor, info)
			assert.Equal(t, err, testErr)
		})

		info.Role = interfaces.SystemRoleOrgManager
		Convey("获取组织管理员管理范围失败-报错", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			testErr := rest.NewHTTPError("error", 503000000, nil)
			userDB.EXPECT().GetOrgManagerDepartInfo2(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, testErr)

			_, _, err := userLogics.SearchUsersByKeyInScope(ctx, &visitor, info)
			assert.Equal(t, err, testErr)
		})

		info.Role = interfaces.SystemRoleNormalUser
		Convey("获取普通用户管理权限失败-报错", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			testErr := rest.NewHTTPError("error", 503000000, nil)
			orgPerm.EXPECT().CheckPerms(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(true, testErr)

			_, _, err := userLogics.SearchUsersByKeyInScope(ctx, &visitor, info)
			assert.Equal(t, err, testErr)
		})

		Convey("获取普通用户管理权限,无权限-报错", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			orgPerm.EXPECT().CheckPerms(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(false, nil)

			out, num, err := userLogics.SearchUsersByKeyInScope(ctx, &visitor, info)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(out), 0)
			assert.Equal(t, num, 0)
		})

		info.Role = interfaces.SystemRoleOrgManager
		Convey("获取子部门ID失败-报错", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			testErr := rest.NewHTTPError("error", 503000000, nil)
			userDB.EXPECT().GetOrgManagerDepartInfo2(gomock.Any(), gomock.Any()).AnyTimes().Return([]string{"xxx"}, nil)
			departmentDB.EXPECT().GetDepartmentInfo2(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, testErr)

			_, _, err := userLogics.SearchUsersByKeyInScope(ctx, &visitor, info)
			assert.Equal(t, err, testErr)
		})

		info.Role = interfaces.SystemRoleSuperAdmin
		temp1 := interfaces.UserDBInfo{
			ID:   "ID1",
			Name: "Name1",
		}
		userInfo := []interfaces.UserDBInfo{temp1}
		Convey("获取用户父部门信息失败-报错", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			testErr := rest.NewHTTPError("error", 503000000, nil)
			userDB.EXPECT().GetOrgManagerDepartInfo2(gomock.Any(), gomock.Any()).AnyTimes().Return([]string{"xxx"}, nil)
			departmentDB.EXPECT().GetDepartmentInfo2(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
			userDB.EXPECT().SearchOrgUsersByKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(userInfo, testErr)

			_, _, err := userLogics.SearchUsersByKeyInScope(ctx, &visitor, info)
			assert.Equal(t, err, testErr)
		})

		Convey("用户搜索完成", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			userDB.EXPECT().GetOrgManagerDepartInfo2(gomock.Any(), gomock.Any()).AnyTimes().Return([]string{"xxx"}, nil)
			departmentDB.EXPECT().GetDepartmentInfo2(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)

			userDB.EXPECT().SearchOrgUsersByKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(userInfo, nil)
			userDB.EXPECT().SearchOrgUsersByKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(userInfo, nil)
			userDB.EXPECT().GetDirectBelongDepartmentIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return([]string{}, nil, nil)

			out, num, err := userLogics.SearchUsersByKeyInScope(ctx, &visitor, info)
			assert.Equal(t, err, nil)
			assert.Equal(t, num, 1)
			assert.Equal(t, len(out), 1)
			assert.Equal(t, out[0].ID, temp1.ID)
		})

		info.Role = interfaces.SystemRoleNormalUser
		Convey("普通用户，用户搜索完成", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			orgPerm.EXPECT().CheckPerms(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(true, nil)
			userDB.EXPECT().GetOrgManagerDepartInfo2(gomock.Any(), gomock.Any()).AnyTimes().Return([]string{"xxx"}, nil)
			departmentDB.EXPECT().GetDepartmentInfo2(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)

			userDB.EXPECT().SearchOrgUsersByKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(userInfo, nil)
			userDB.EXPECT().SearchOrgUsersByKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(userInfo, nil)
			userDB.EXPECT().GetDirectBelongDepartmentIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return([]string{}, nil, nil)

			out, num, err := userLogics.SearchUsersByKeyInScope(ctx, &visitor, info)
			assert.Equal(t, err, nil)
			assert.Equal(t, num, 1)
			assert.Equal(t, len(out), 1)
			assert.Equal(t, out[0].ID, temp1.ID)
		})
	})
}

func TestHandleUserBaseInfo(t *testing.T) {
	Convey("handleUserBaseInfo 用户基础信息整理", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		u := &user{}

		info := interfaces.UserBaseInfoRange{
			ShowCSFLevel:      true,
			ShowEnable:        true,
			ShowPriority:      true,
			ShowName:          true,
			ShowAccount:       true,
			ShowFrozen:        true,
			ShowAuthenticated: true,
			ShowEmail:         true,
			ShowTelNumber:     true,
			ShowThirdAttr:     true,
			ShowThirdID:       true,
			ShowAuthType:      true,
		}
		userID := "xxxxx"
		userDBInfo := interfaces.UserDBInfo{
			ID:                userID,
			Name:              "Name11",
			Account:           "Account1",
			CSFLevel:          12,
			Priority:          123,
			DisableStatus:     interfaces.Disabled,
			AutoDisableStatus: interfaces.ADisabled,
			Frozen:            true,
			Authenticated:     true,
			Email:             "xxx@qq.com",
			TelNumber:         "123456",
			ThirdAttr:         "zzzz",
			ThirdID:           "thirdID",
			AuthType:          interfaces.Local,
		}

		Convey("success", func() {
			out := u.handleUserBaseInfo(info, &userDBInfo)
			assert.Equal(t, out.ID, userID)
			assert.Equal(t, out.CSFLevel, userDBInfo.CSFLevel)
			assert.Equal(t, out.Enabled, false)
			assert.Equal(t, out.Priority, userDBInfo.Priority)
			assert.Equal(t, out.Name, userDBInfo.Name)
			assert.Equal(t, out.Account, userDBInfo.Account)
			assert.Equal(t, out.Frozen, true)
			assert.Equal(t, out.Authenticated, true)
			assert.Equal(t, out.Email, "xxx@qq.com")
			assert.Equal(t, out.TelNumber, "123456")
			assert.Equal(t, out.ThirdAttr, "zzzz")
			assert.Equal(t, out.ThirdID, "thirdID")
			assert.Equal(t, out.AuthType, userDBInfo.AuthType)
		})
	})
}

func TestHandleUserJSON(t *testing.T) {
	Convey("handleUserInfo, db is available", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		departmentDB := mock.NewMockDBDepartment(ctrl)
		contactorDB := mock.NewMockDBContactor(ctrl)
		groupMemberDB := mock.NewMockDBGroupMember(ctrl)

		u := &user{
			userDB:        userDB,
			departmentDB:  departmentDB,
			contactorDB:   contactorDB,
			groupMemberDB: groupMemberDB,
		}

		info := interfaces.UserBaseInfoRange{
			ShowCSFLevel:       true,
			ShowEnable:         true,
			ShowRoles:          true,
			ShowPriority:       true,
			ShowName:           true,
			ShowParentDeps:     true,
			ShowParentDepPaths: true,
			ShowAccount:        true,
			ShowFrozen:         true,
			ShowAuthenticated:  true,
			ShowEmail:          true,
			ShowTelNumber:      true,
			ShowThirdAttr:      true,
			ShowThirdID:        true,
			ShowAvatar:         true,
			ShowAuthType:       true,
			ShowGroups:         true,
			ShowCustomAttr:     true,
			ShowManager:        true,
			ShowCSFLevel2:      true,
		}
		userID := "xxxxx"
		userDBInfo := interfaces.UserDBInfo{
			ID:                userID,
			Name:              "Name11",
			Account:           "Account1",
			CSFLevel:          12,
			Priority:          123,
			DisableStatus:     interfaces.Disabled,
			AutoDisableStatus: interfaces.ADisabled,
			Frozen:            true,
			Authenticated:     true,
			Email:             "xxx@qq.com",
			TelNumber:         "123456",
			ThirdAttr:         "zzzz",
			ThirdID:           "thirdID",
			AuthType:          interfaces.Local,
			ManagerID:         strID,
			CSFLevel2:         13,
		}
		userRoleInfo := make(map[interfaces.Role]bool)
		userRoleInfo["sdadada"] = true
		temp := []interfaces.ObjectBaseInfo{}
		depInfos := [][]interfaces.ObjectBaseInfo{temp}
		paths := []string{"asdasd"}
		avatars := "112331"
		groups := []interfaces.GroupInfo{}
		customs := make(map[string]interface{}, 0)
		tempxx := interfaces.NameInfo{
			ID:   strID,
			Name: strName,
		}
		managers := map[string]interfaces.NameInfo{
			strID: tempxx,
		}

		Convey("success", func() {
			out := u.handleUserInfo(info, &userDBInfo, userRoleInfo, depInfos, paths, avatars, groups, customs, managers)
			assert.Equal(t, out.CSFLevel, userDBInfo.CSFLevel)
			assert.Equal(t, out.Enabled, false)
			assert.Equal(t, out.VecRoles, []interfaces.Role{"sdadada"})
			assert.Equal(t, out.Priority, userDBInfo.Priority)
			assert.Equal(t, out.Name, userDBInfo.Name)
			assert.Equal(t, out.Account, userDBInfo.Account)
			assert.Equal(t, out.ParentDeps, depInfos)
			assert.Equal(t, out.ParentDepPaths, paths)
			assert.Equal(t, out.Frozen, true)
			assert.Equal(t, out.Authenticated, true)
			assert.Equal(t, out.Email, "xxx@qq.com")
			assert.Equal(t, out.TelNumber, "123456")
			assert.Equal(t, out.ThirdAttr, "zzzz")
			assert.Equal(t, out.ThirdID, "thirdID")
			assert.Equal(t, out.Avatar, avatars)
			assert.Equal(t, out.AuthType, userDBInfo.AuthType)
			assert.Equal(t, out.Groups, groups)
			assert.Equal(t, out.CustomAttr, customs)
			assert.Equal(t, out.Manager.ID, strID)
			assert.Equal(t, out.Manager.Name, strName)
			assert.Equal(t, out.CSFLevel2, userDBInfo.CSFLevel2)
		})
	})
}

func TestGetUserParentDepsInfo(t *testing.T) {
	Convey("获取用户父部门信息", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		departmentDB := mock.NewMockDBDepartment(ctrl)
		contactorDB := mock.NewMockDBContactor(ctrl)
		groupMemberDB := mock.NewMockDBGroupMember(ctrl)
		trace := mock.NewMockTraceClient(ctrl)

		u := &user{
			userDB:        userDB,
			departmentDB:  departmentDB,
			contactorDB:   contactorDB,
			groupMemberDB: groupMemberDB,
			trace:         trace,
		}

		userID := "xxxxxxx"
		ctx := context.Background()
		Convey("获取用户直属部门信息失败-报错", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			userDB.EXPECT().GetDirectBelongDepartmentIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil, testErr)

			_, _, err := u.getUserParentDepsInfo(userID, ctx)
			assert.Equal(t, err, testErr)
		})

		temp1 := interfaces.DepartmentDBInfo{
			ID:   "xxx",
			Name: "zzz",
		}

		Convey("获取部门父部门信息失败-报错", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			userDB.EXPECT().GetDirectBelongDepartmentIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, []interfaces.DepartmentDBInfo{temp1}, nil)
			departmentDB.EXPECT().GetDepartmentInfo2(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, testErr)

			_, _, err := u.getUserParentDepsInfo(userID, ctx)
			assert.Equal(t, err, testErr)
		})

		temp2 := interfaces.DepartmentDBInfo{
			ID:   "xxx1",
			Name: "zzz1",
		}

		test1 := interfaces.DepartmentDBInfo{
			Path: temp2.ID + "/" + temp1.ID,
		}
		Convey("获取用户父部门信息", func() {
			userDB.EXPECT().GetDirectBelongDepartmentIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return([]string{"xxx"}, []interfaces.DepartmentDBInfo{test1}, nil)
			departmentDB.EXPECT().GetDepartmentInfo2(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return([]interfaces.DepartmentDBInfo{temp1, temp2}, nil)

			out1, paths, err := u.getUserParentDepsInfo(userID, ctx)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(paths), 1)
			assert.Equal(t, paths[0], temp2.Name+"/"+temp1.Name)
			assert.Equal(t, len(out1), 1)
			assert.Equal(t, len(out1[0]), 2)
			assert.Equal(t, out1[0][0].ID, temp2.ID)
			assert.Equal(t, out1[0][1].ID, temp1.ID)
		})
	})
}

func TestCheckUserInRange(t *testing.T) {
	Convey("判断用户是否在范围内", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		departmentDB := mock.NewMockDBDepartment(ctrl)
		contactorDB := mock.NewMockDBContactor(ctrl)
		groupMemberDB := mock.NewMockDBGroupMember(ctrl)

		u := &user{
			userDB:        userDB,
			departmentDB:  departmentDB,
			contactorDB:   contactorDB,
			groupMemberDB: groupMemberDB,
		}

		Convey("获取用户直属部门失败", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			userDB.EXPECT().GetDirectBelongDepartmentIDs(gomock.Any()).AnyTimes().Return(nil, nil, testErr)

			_, err := u.checkUserInRange("xxxx", "sasd", []string{"xxx"})
			assert.Equal(t, err, testErr)
		})
	})
}

func TestModifyUserInfo(t *testing.T) {
	Convey("修改用户信息", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		departmentDB := mock.NewMockDBDepartment(ctrl)
		contactorDB := mock.NewMockDBContactor(ctrl)
		groupMemberDB := mock.NewMockDBGroupMember(ctrl)
		configDB := mock.NewMockDBConfig(ctrl)
		hydra := mock.NewMockDrivenHydra(ctrl)
		ob := mock.NewMockLogicsOutbox(ctrl)
		role := mock.NewMockLogicsRole(ctrl)

		dPool, txMock, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func() {
			if closeErr := dPool.Close(); closeErr != nil {
				assert.Equal(t, 1, 1)
			}
		}()

		u := &user{
			userDB:        userDB,
			departmentDB:  departmentDB,
			contactorDB:   contactorDB,
			groupMemberDB: groupMemberDB,
			pool:          dPool,
			configDB:      configDB,
			hydra:         hydra,
			ob:            ob,
			logger:        common.NewLogger(),
			role:          role,
		}

		var data interfaces.UserUpdateRange
		data.UpdatePWD = true
		var data2 interfaces.UserBaseInfo
		var visitor interfaces.Visitor

		visitor.Type = interfaces.RealName
		visitor.ID = "zz1"
		Convey("获取用户角色信息失败", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(nil, testErr)

			err := u.ModifyUserInfo(&visitor, data, &data2)
			assert.Equal(t, err, testErr)
		})

		roleInfo := make(map[string]map[interfaces.Role]bool)
		mapRole := make(map[interfaces.Role]bool)
		roleInfo[visitor.ID] = mapRole
		Convey("用户无权限", func() {
			testErr1 := rest.NewHTTPError("this user do not has the authority", rest.Forbidden, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfo, nil)

			err := u.ModifyUserInfo(&visitor, data, &data2)
			assert.Equal(t, err, testErr1)
		})

		mapRole[interfaces.SystemRoleSuperAdmin] = true
		tempConfig := interfaces.Config{
			PwdExpireTime:   -1,
			StrongPwdStatus: false,
			StrongPwdLength: 80,
			EnablePwdLock:   false,
			PwdLockTime:     -1,
			PwdErrCnt:       2,
		}
		Convey("获取配置失败", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfo, nil)
			configDB.EXPECT().GetConfig(gomock.Any()).AnyTimes().Return(tempConfig, testErr)

			err := u.ModifyUserInfo(&visitor, data, &data2)
			assert.Equal(t, err, testErr)
		})

		Convey("参数检查失败", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).AnyTimes().Return(nil, testErr)
			configDB.EXPECT().GetConfig(gomock.Any()).AnyTimes().Return(tempConfig, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(nil, testErr)

			err := u.ModifyUserInfo(&visitor, data, &data2)
			assert.Equal(t, err, testErr)
		})

		tempUser := interfaces.UserDBInfo{}
		tempUser.AuthType = interfaces.Local
		userInfo := []interfaces.UserDBInfo{tempUser}
		data.UpdatePWD = true
		data2.Password = cRSAPWD

		Convey("db pool 初始化失败", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfo, nil)
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).AnyTimes().Return(userInfo, nil)
			configDB.EXPECT().GetConfig(gomock.Any()).AnyTimes().Return(tempConfig, nil)
			txMock.ExpectBegin().WillReturnError(testErr)

			err := u.ModifyUserInfo(&visitor, data, &data2)
			assert.NotEqual(t, err, nil)
		})

		Convey("修改密码报错", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			txMock.ExpectBegin()
			txMock.ExpectRollback()
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfo, nil)
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).AnyTimes().Return(userInfo, nil)
			configDB.EXPECT().GetConfig(gomock.Any()).AnyTimes().Return(tempConfig, nil)
			userDB.EXPECT().ModifyUserInfo(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(testErr)

			err := u.ModifyUserInfo(&visitor, data, &data2)
			assert.Equal(t, err, testErr)
		})

		Convey("修改密码成功,outbox添加失败", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			txMock.ExpectBegin()
			txMock.ExpectRollback()
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfo, nil)
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).AnyTimes().Return(userInfo, nil)
			configDB.EXPECT().GetConfig(gomock.Any()).AnyTimes().Return(tempConfig, nil)
			userDB.EXPECT().ModifyUserInfo(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

			ob.EXPECT().AddOutboxInfo(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(testErr)

			err := u.ModifyUserInfo(&visitor, data, &data2)
			assert.Equal(t, err, testErr)
		})

		Convey("修改密码成功", func() {
			txMock.ExpectBegin()
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfo, nil)
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).AnyTimes().Return(userInfo, nil)
			configDB.EXPECT().GetConfig(gomock.Any()).AnyTimes().Return(tempConfig, nil)
			userDB.EXPECT().ModifyUserInfo(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

			ob.EXPECT().AddOutboxInfo(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			txMock.ExpectCommit()
			ob.EXPECT().NotifyPushOutboxThread().AnyTimes().Return()

			err := u.ModifyUserInfo(&visitor, data, &data2)
			assert.Equal(t, err, nil)
		})
	})
}

func TestCheckModifyPwdParam(t *testing.T) {
	Convey("修改用户密码,参数检测", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		departmentDB := mock.NewMockDBDepartment(ctrl)
		contactorDB := mock.NewMockDBContactor(ctrl)
		groupMemberDB := mock.NewMockDBGroupMember(ctrl)
		configDB := mock.NewMockDBConfig(ctrl)
		hydra := mock.NewMockDrivenHydra(ctrl)

		dPool, _, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func() {
			if closeErr := dPool.Close(); closeErr != nil {
				assert.Equal(t, 1, 1)
			}
		}()

		u := &user{
			userDB:        userDB,
			departmentDB:  departmentDB,
			contactorDB:   contactorDB,
			groupMemberDB: groupMemberDB,
			configDB:      configDB,
			hydra:         hydra,
			pool:          dPool,
		}

		testConfig := interfaces.Config{
			PwdExpireTime:   -1,
			StrongPwdStatus: false,
			StrongPwdLength: 80,
			EnablePwdLock:   false,
			PwdLockTime:     -1,
			PwdErrCnt:       2,
		}
		userInfo := interfaces.UserBaseInfo{}
		Convey("获取用户信息失败", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).AnyTimes().Return(nil, testErr)

			_, err := u.checkModifyPwdParam(&userInfo, &testConfig)
			assert.Equal(t, err, testErr)
		})

		Convey("用户不存在", func() {
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).AnyTimes().Return(nil, nil)

			testErr1 := rest.NewHTTPErrorV2(errors.NotFound, "user does not exist")
			_, err := u.checkModifyPwdParam(&userInfo, &testConfig)
			assert.Equal(t, err, testErr1)
		})

		tempUser := interfaces.UserDBInfo{
			AuthType: 2,
		}
		userDBInfo := []interfaces.UserDBInfo{tempUser}
		Convey("非本地认证用户不能修改密码", func() {
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).AnyTimes().Return(userDBInfo, nil)

			testErr1 := rest.NewHTTPError("can not modify non local user password", rest.BadRequest, nil)
			_, err := u.checkModifyPwdParam(&userInfo, &testConfig)
			assert.Equal(t, err, testErr1)
		})

		tempUser.AuthType = 1
		userDBInfo = []interfaces.UserDBInfo{tempUser}
		Convey("rsa密码解密失败", func() {
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).AnyTimes().Return(userDBInfo, nil)

			_, err := u.checkModifyPwdParam(&userInfo, &testConfig)
			assert.NotEqual(t, err, nil)
		})

		userInfo.Password = cRSAPWD
		Convey("密码参数检测通过", func() {
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).AnyTimes().Return(userDBInfo, nil)
			configDB.EXPECT().GetConfig(gomock.Any()).AnyTimes().Return(testConfig, nil)

			_, err := u.checkModifyPwdParam(&userInfo, &testConfig)
			assert.Equal(t, err, nil)
		})
	})
}

func TestGeneratePWD(t *testing.T) {
	Convey("获取密码测试", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		departmentDB := mock.NewMockDBDepartment(ctrl)
		contactorDB := mock.NewMockDBContactor(ctrl)
		groupMemberDB := mock.NewMockDBGroupMember(ctrl)
		configDB := mock.NewMockDBConfig(ctrl)
		hydra := mock.NewMockDrivenHydra(ctrl)

		dPool, _, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func() {
			if closeErr := dPool.Close(); closeErr != nil {
				assert.Equal(t, 1, 1)
			}
		}()

		u := &user{
			userDB:        userDB,
			departmentDB:  departmentDB,
			contactorDB:   contactorDB,
			groupMemberDB: groupMemberDB,
			configDB:      configDB,
			hydra:         hydra,
			pool:          dPool,
		}

		config := interfaces.Config{
			EnableDesPwd: true,
		}
		Convey("更新成功，需要返回DES密码", func() {
			md5, des, ntlm, err := u.generatePWD("xxxx", &config)
			assert.Equal(t, err, nil)
			assert.NotEqual(t, md5, "")
			assert.NotEqual(t, des, "")
			assert.NotEqual(t, ntlm, "")
		})

		config.EnableDesPwd = false
		Convey("更新成功，不需要返回DES密码", func() {
			md5, des, ntlm, err := u.generatePWD("xxxx", &config)
			assert.Equal(t, err, nil)
			assert.NotEqual(t, md5, "")
			assert.Equal(t, des, "")
			assert.NotEqual(t, ntlm, "")
		})
	})
}

func TestCheckPasswordValid(t *testing.T) {
	Convey("检查密码是否合法", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		departmentDB := mock.NewMockDBDepartment(ctrl)
		contactorDB := mock.NewMockDBContactor(ctrl)
		groupMemberDB := mock.NewMockDBGroupMember(ctrl)
		configDB := mock.NewMockDBConfig(ctrl)

		u := &user{
			userDB:        userDB,
			departmentDB:  departmentDB,
			contactorDB:   contactorDB,
			groupMemberDB: groupMemberDB,
			configDB:      configDB,
		}

		testConfig := interfaces.Config{
			PwdExpireTime:   -1,
			StrongPwdStatus: false,
			StrongPwdLength: 80,
			EnablePwdLock:   false,
			PwdLockTime:     -1,
			PwdErrCnt:       2,
		}
		Convey("按照弱密码检查成功", func() {
			configDB.EXPECT().GetConfig(gomock.Any()).AnyTimes().Return(testConfig, nil)

			out := u.checkPasswordValid("123123123", &testConfig)
			assert.Equal(t, out, true)
		})

		testConfig.StrongPwdStatus = true
		Convey("按照强密码检查失败", func() {
			configDB.EXPECT().GetConfig(gomock.Any()).AnyTimes().AnyTimes().Return(testConfig, nil)

			out := u.checkPasswordValid("123123123", &testConfig)
			assert.Equal(t, out, false)
		})
	})
}

func TestCheckModifyUserPWDAuthority(t *testing.T) {
	Convey("检查是否拥有修改用户密码权限", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		departmentDB := mock.NewMockDBDepartment(ctrl)
		contactorDB := mock.NewMockDBContactor(ctrl)
		groupMemberDB := mock.NewMockDBGroupMember(ctrl)
		configDB := mock.NewMockDBConfig(ctrl)
		orgPermAppDB := mock.NewMockDBOrgPermApp(ctrl)
		role := mock.NewMockLogicsRole(ctrl)

		u := &user{
			userDB:        userDB,
			departmentDB:  departmentDB,
			contactorDB:   contactorDB,
			groupMemberDB: groupMemberDB,
			configDB:      configDB,
			orgPermAppDB:  orgPermAppDB,
			role:          role,
		}

		visitor := interfaces.Visitor{
			Type: interfaces.Anonymous,
			ID:   "2313",
		}

		Convey("visitor类型为匿名用户，不允许修改应用账户", func() {
			tempErr1 := rest.NewHTTPError("Unsupported user type", rest.Forbidden, nil)
			out := u.checkModifyUserPWDAuthority(&visitor)
			assert.Equal(t, out, tempErr1)
		})

		visitor.Type = interfaces.RealName
		Convey("visitor类型为实名账户，获取用户信息失败", func() {
			tempErr1 := rest.NewHTTPError("xxx", errors.Forbidden, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(nil, tempErr1)
			out := u.checkModifyUserPWDAuthority(&visitor)
			assert.Equal(t, out, tempErr1)
		})

		Convey("visitor类型为实名账户，但是不是超级管理员或者安全管理员", func() {
			tempErr1 := rest.NewHTTPError("this user do not has the authority", rest.Forbidden, nil)
			userRoles := make(map[interfaces.Role]bool)
			roleInfos := make(map[string]map[interfaces.Role]bool)
			roleInfos[visitor.ID] = userRoles
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			out := u.checkModifyUserPWDAuthority(&visitor)
			assert.Equal(t, out, tempErr1)
		})

		Convey("visitor类型为实名账户，超级管理员角色， 检查成功", func() {
			userRoles := make(map[interfaces.Role]bool)
			userRoles[interfaces.SystemRoleSuperAdmin] = true
			roleInfos := make(map[string]map[interfaces.Role]bool)
			roleInfos[visitor.ID] = userRoles
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			out := u.checkModifyUserPWDAuthority(&visitor)
			assert.Equal(t, out, nil)
		})

		Convey("visitor类型为实名账户，安全管理员角色， 检查成功", func() {
			userRoles := make(map[interfaces.Role]bool)
			userRoles[interfaces.SystemRoleSecAdmin] = true
			roleInfos := make(map[string]map[interfaces.Role]bool)
			roleInfos[visitor.ID] = userRoles
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			out := u.checkModifyUserPWDAuthority(&visitor)
			assert.Equal(t, out, nil)
		})

		visitor.Type = interfaces.App
		var info interfaces.AppOrgPerm
		allPerm := make(map[interfaces.OrgType]interfaces.AppOrgPerm)
		info.Value = 1
		Convey("visitor类型为应用账户，获取应用账户权限失败", func() {
			tempErr1 := rest.NewHTTPError("xxx", errors.Forbidden, nil)
			orgPermAppDB.EXPECT().GetAppPermByID(gomock.Any()).AnyTimes().Return(allPerm, tempErr1)
			out := u.checkModifyUserPWDAuthority(&visitor)
			assert.Equal(t, out, tempErr1)
		})

		Convey("visitor类型为应用账户，没有修改用户密码权限", func() {
			tempErr1 := rest.NewHTTPError("this user do not has the authority", rest.Forbidden, nil)
			orgPermAppDB.EXPECT().GetAppPermByID(gomock.Any()).AnyTimes().Return(allPerm, nil)
			out := u.checkModifyUserPWDAuthority(&visitor)
			assert.Equal(t, out, tempErr1)
		})

		allPerm[interfaces.User] = info
		Convey("visitor类型为应用账户，具有修改用户密码权限", func() {
			orgPermAppDB.EXPECT().GetAppPermByID(gomock.Any()).AnyTimes().Return(allPerm, nil)
			out := u.checkModifyUserPWDAuthority(&visitor)
			assert.Equal(t, out, nil)
		})
	})
}

func TestCheckIncrementModifyUserInfoAuthority(t *testing.T) {
	Convey("checkIncrementModifyUserInfoAuthority", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		departmentDB := mock.NewMockDBDepartment(ctrl)
		contactorDB := mock.NewMockDBContactor(ctrl)
		groupMemberDB := mock.NewMockDBGroupMember(ctrl)
		configDB := mock.NewMockDBConfig(ctrl)
		orgPermAppDB := mock.NewMockDBOrgPermApp(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		trace := mock.NewMockTraceClient(ctrl)

		u := &user{
			userDB:        userDB,
			departmentDB:  departmentDB,
			contactorDB:   contactorDB,
			groupMemberDB: groupMemberDB,
			configDB:      configDB,
			orgPermAppDB:  orgPermAppDB,
			role:          role,
			trace:         trace,
		}
		ctx := context.Background()

		visitor := interfaces.Visitor{
			Type: interfaces.Anonymous,
			ID:   "2313",
		}

		Convey("visitor类型为匿名用户，不允许修改应用账户", func() {
			tempErr1 := rest.NewHTTPErrorV2(rest.Forbidden, "Unsupported user type")
			out := u.checkIncrementModifyUserInfoAuthority(ctx, &visitor)
			assert.Equal(t, out, tempErr1)
		})

		visitor.Type = interfaces.RealName
		Convey("visitor类型为实名账户，获取用户信息失败", func() {
			tempErr1 := rest.NewHTTPError("xxx", errors.Forbidden, nil)
			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, tempErr1)
			out := u.checkIncrementModifyUserInfoAuthority(ctx, &visitor)
			assert.Equal(t, out, tempErr1)
		})

		Convey("visitor类型为实名账户，但是不是超级管理员或者安全管理员", func() {
			tempErr1 := rest.NewHTTPErrorV2(rest.Forbidden, "this user do not has the authority")
			userRoles := make(map[interfaces.Role]bool)
			roleInfos := make(map[string]map[interfaces.Role]bool)
			roleInfos[visitor.ID] = userRoles
			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return(roleInfos, nil)
			out := u.checkIncrementModifyUserInfoAuthority(ctx, &visitor)
			assert.Equal(t, out, tempErr1)
		})

		Convey("visitor类型为实名账户，超级管理员角色， 检查成功", func() {
			userRoles := make(map[interfaces.Role]bool)
			userRoles[interfaces.SystemRoleSuperAdmin] = true
			roleInfos := make(map[string]map[interfaces.Role]bool)
			roleInfos[visitor.ID] = userRoles
			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return(roleInfos, nil)
			out := u.checkIncrementModifyUserInfoAuthority(ctx, &visitor)
			assert.Equal(t, out, nil)
		})

		Convey("visitor类型为实名账户，安全管理员角色， 检查成功", func() {
			userRoles := make(map[interfaces.Role]bool)
			userRoles[interfaces.SystemRoleSecAdmin] = true
			roleInfos := make(map[string]map[interfaces.Role]bool)
			roleInfos[visitor.ID] = userRoles
			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return(roleInfos, nil)
			out := u.checkIncrementModifyUserInfoAuthority(ctx, &visitor)
			assert.Equal(t, out, nil)
		})

		Convey("visitor类型为实名账户，系统管理员角色， 检查成功", func() {
			userRoles := make(map[interfaces.Role]bool)
			userRoles[interfaces.SystemRoleSysAdmin] = true
			roleInfos := make(map[string]map[interfaces.Role]bool)
			roleInfos[visitor.ID] = userRoles
			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return(roleInfos, nil)
			out := u.checkIncrementModifyUserInfoAuthority(ctx, &visitor)
			assert.Equal(t, out, nil)
		})

		visitor.Type = interfaces.App
		var info interfaces.AppOrgPerm
		allPerm := make(map[interfaces.OrgType]interfaces.AppOrgPerm)
		info.Value = 1
		Convey("visitor类型为应用账户，获取应用账户权限失败", func() {
			tempErr1 := rest.NewHTTPError("xxx", errors.Forbidden, nil)
			orgPermAppDB.EXPECT().GetAppPermByID2(gomock.Any(), gomock.Any()).AnyTimes().Return(allPerm, tempErr1)
			out := u.checkIncrementModifyUserInfoAuthority(ctx, &visitor)
			assert.Equal(t, out, tempErr1)
		})

		Convey("visitor类型为应用账户，没有修改用户密码权限", func() {
			tempErr1 := rest.NewHTTPErrorV2(rest.Forbidden, "this user do not has the authority")
			orgPermAppDB.EXPECT().GetAppPermByID2(gomock.Any(), gomock.Any()).AnyTimes().Return(allPerm, nil)
			out := u.checkIncrementModifyUserInfoAuthority(ctx, &visitor)
			assert.Equal(t, out, tempErr1)
		})

		allPerm[interfaces.User] = info
		Convey("visitor类型为应用账户，具有修改用户密码权限", func() {
			orgPermAppDB.EXPECT().GetAppPermByID2(gomock.Any(), gomock.Any()).AnyTimes().Return(allPerm, nil)
			out := u.checkIncrementModifyUserInfoAuthority(ctx, &visitor)
			assert.Equal(t, out, nil)
		})
	})
}

func TestCheckIsValidPassword(t *testing.T) {
	u := &user{}
	Convey("弱密码测试", t, func() {
		Convey("弱密码测试正常", func() {
			pwd := testPWD
			out := u.checkIsValidPassword(pwd)
			assert.Equal(t, out, true)
		})

		Convey("弱密码测试,密码为空", func() {
			out := u.checkIsValidPassword("")
			assert.Equal(t, out, false)
		})

		Convey("密码测试，长度小于6 无效", func() {
			pwd := "12312"
			out := u.checkIsValidPassword(pwd)
			assert.Equal(t, out, false)
		})

		Convey("密码测试，长度等于6 有效", func() {
			pwd := "111egh"
			out := u.checkIsValidPassword(pwd)
			assert.Equal(t, out, true)
		})

		Convey("密码测试，长度大于100  无效", func() {
			pwd := ""
			temp := "1234567890"
			for i := 0; i < 10; i++ {
				pwd += temp
			}
			pwd += "1"

			out := u.checkIsValidPassword(pwd)
			assert.Equal(t, out, false)
		})

		Convey("密码测试，长度等于100  有效", func() {
			pwd := ""
			temp := "1234567890"
			for i := 0; i < 10; i++ {
				pwd += temp
			}

			out := u.checkIsValidPassword(pwd)
			assert.Equal(t, out, true)
		})

		Convey("密码测试，包含!@#$%-_,.  正常", func() {
			pwd := "12312!@#$%-_,."
			out := u.checkIsValidPassword(pwd)
			assert.Equal(t, out, true)
		})

		Convey("密码测试，包含~`^&*()+=|<>?  有效", func() {
			pwd := "12312123123&&sd~`^&*()+=|<>?"
			out := u.checkIsValidPassword(pwd)
			assert.Equal(t, out, true)
		})

		Convey("密码测试，包含0x20-0x7E  有效", func() {
			pwd := " !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~"
			out := u.checkIsValidPassword(pwd)
			assert.Equal(t, out, true)
		})

		Convey("密码测试范围外 \t为ascii的9 无效", func() {
			pwd := "111111111\t"
			out := u.checkIsValidPassword(pwd)
			assert.Equal(t, out, false)
		})
	})
}

func TestCheckIsStrongPassword(t *testing.T) {
	u := &user{}
	Convey("强密码测试正常", t, func() {
		Convey("密码测试正常", func() {
			pwd := "12312312312aA~"
			out := u.checkIsStrongPassword(pwd, 10)
			assert.Equal(t, out, true)
		})

		Convey("密码测试长度不足，报错", func() {
			pwd := "123123aA"
			out := u.checkIsStrongPassword(pwd, 10)
			assert.Equal(t, out, false)
		})

		Convey("密码测试，不包含小写字母，报错", func() {
			pwd := "1231231232112A"
			out := u.checkIsStrongPassword(pwd, 10)
			assert.Equal(t, out, false)
		})

		Convey("密码测试，不包含大写字母，报错", func() {
			pwd := "123123asd12a"
			out := u.checkIsStrongPassword(pwd, 10)
			assert.Equal(t, out, false)
		})

		Convey("密码测试，不包含数字，报错", func() {
			pwd := "aaaaaaassdsdsAAA"
			out := u.checkIsStrongPassword(pwd, 10)
			assert.Equal(t, out, false)
		})

		Convey("密码测试，不符合弱密码要求，包含\t，报错", func() {
			pwd := "1234567aAA\t\t"
			out := u.checkIsStrongPassword(pwd, 10)
			assert.Equal(t, out, false)
		})

		Convey("密码测试，长度等于101  报错", func() {
			pwd := ""
			temp := "123456aA01"
			for i := 0; i < 10; i++ {
				pwd += temp
			}
			pwd += "1"

			out := u.checkIsStrongPassword(pwd, 10)
			assert.Equal(t, out, false)
		})
	})
}

//nolint:dupl
func TestGetUsersBaseInfo(t *testing.T) {
	Convey("批量获取用户信息", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		departmentDB := mock.NewMockDBDepartment(ctrl)
		contactorDB := mock.NewMockDBContactor(ctrl)
		groupMemberDB := mock.NewMockDBGroupMember(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		ava := mock.NewMockLogicsAvatar(ctrl)
		trace := mock.NewMockTraceClient(ctrl)
		u := &user{
			userDB:        userDB,
			departmentDB:  departmentDB,
			contactorDB:   contactorDB,
			groupMemberDB: groupMemberDB,
			role:          role,
			avatar:        ava,
			trace:         trace,
		}

		var rangeInfo interfaces.UserBaseInfoRange

		ctx := context.Background()
		visitor := &interfaces.Visitor{
			ID:   interfaces.SystemAuditAdmin,
			Type: interfaces.App,
		}

		trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
		trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
		trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

		Convey("获取用户信息数据失败", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).AnyTimes().Return(nil, testErr)

			_, err := u.GetUsersBaseInfo(ctx, visitor, []string{"xxx"}, rangeInfo, true)
			assert.Equal(t, err, testErr)
		})

		userInfo1 := interfaces.UserDBInfo{
			ID:            "xxx",
			Frozen:        true,
			Authenticated: true,
			Account:       "xxxx1",
			ManagerID:     "manager_id",
		}
		Convey("存在相同用户，报错", func() {
			testErr := rest.NewHTTPError("there are same users", rest.BadRequest, nil)

			_, err := u.GetUsersBaseInfo(ctx, visitor, []string{"xxx", "xxx", "yyy"}, rangeInfo, true)
			assert.Equal(t, err, testErr)
		})

		userDBInfo := []interfaces.UserDBInfo{userInfo1}
		Convey("用户不存在，报错", func() {
			testErr := rest.NewHTTPErrorV2(errors.NotFound, "those users are not existing",
				rest.SetDetail(map[string]interface{}{"ids": []string{"yyy"}}))

			userDB.EXPECT().GetUserDBInfo(gomock.Any()).AnyTimes().Return(userDBInfo, nil)

			_, err := u.GetUsersBaseInfo(ctx, visitor, []string{"xxx", "yyy"}, rangeInfo, true)
			assert.Equal(t, err, testErr)
		})

		rangeInfo.ShowRoles = true
		Convey("获取用户角色失败，报错", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).AnyTimes().Return(userDBInfo, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(nil, testErr)

			_, err := u.GetUsersBaseInfo(ctx, visitor, []string{"xxx"}, rangeInfo, true)
			assert.Equal(t, err, testErr)
		})

		rangeInfo.ShowAuthenticated = true
		rangeInfo.ShowFrozen = true
		rangeInfo.ShowAccount = true
		rangeInfo.ShowAvatar = true
		rangeInfo.ShowCustomAttr = true
		rangeInfo.ShowManager = true
		Convey("获取用户角色自定义属性，报错", func() {
			tmpErr := fmt.Errorf("err")
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).AnyTimes().Return(userDBInfo, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(nil, nil)
			ava.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return("url1", nil)
			userDB.EXPECT().GetUserCustomAttr(gomock.Any()).Return(nil, tmpErr)

			_, err := u.GetUsersBaseInfo(ctx, visitor, []string{"xxx"}, rangeInfo, true)
			assert.Equal(t, err, tmpErr)
		})

		manager1 := interfaces.UserDBInfo{
			ID:   "manager_id",
			Name: "manager_name",
		}
		managerInfos := []interfaces.UserDBInfo{manager1}
		Convey("成功", func() {
			customAttr := make(map[string]interface{}, 0)
			customAttr[strTest] = strTest
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).AnyTimes().Return(userDBInfo, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(nil, nil)
			ava.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return("url1", nil)
			userDB.EXPECT().GetUserCustomAttr(gomock.Any()).Return(customAttr, nil)
			userDB.EXPECT().GetUserName(gomock.Any()).AnyTimes().Return(managerInfos, []string{manager1.ID}, nil)

			out, err := u.GetUsersBaseInfo(ctx, visitor, []string{"xxx"}, rangeInfo, true)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(out), 1)
			assert.Equal(t, out[0].Account, userInfo1.Account)
			assert.Equal(t, out[0].Frozen, true)
			assert.Equal(t, out[0].Authenticated, true)
			assert.Equal(t, out[0].Avatar, "url1")
			assert.Equal(t, out[0].CustomAttr, customAttr)
			assert.Equal(t, out[0].Manager.ID, manager1.ID)
			assert.Equal(t, out[0].Manager.Name, manager1.Name)
		})

		Convey("不严格模式， 用户重复，成功", func() {
			customAttr := make(map[string]interface{}, 0)
			customAttr[strTest] = strTest
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).AnyTimes().Return(userDBInfo, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(nil, nil)
			ava.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return("url1", nil)
			userDB.EXPECT().GetUserCustomAttr(gomock.Any()).Return(customAttr, nil)
			userDB.EXPECT().GetUserName(gomock.Any()).AnyTimes().Return(managerInfos, []string{manager1.ID}, nil)

			out, err := u.GetUsersBaseInfo(ctx, visitor, []string{"xxx", "xxx"}, rangeInfo, false)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(out), 1)
			assert.Equal(t, out[0].Account, userInfo1.Account)
			assert.Equal(t, out[0].Frozen, true)
			assert.Equal(t, out[0].Authenticated, true)
			assert.Equal(t, out[0].Avatar, "url1")
			assert.Equal(t, out[0].CustomAttr, customAttr)
			assert.Equal(t, out[0].Manager.ID, manager1.ID)
			assert.Equal(t, out[0].Manager.Name, manager1.Name)
		})

		Convey("不严格模式，用户不存在, 成功", func() {
			customAttr := make(map[string]interface{}, 0)
			customAttr[strTest] = strTest
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).AnyTimes().Return(userDBInfo, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(nil, nil)
			ava.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return("url1", nil)
			userDB.EXPECT().GetUserCustomAttr(gomock.Any()).Return(customAttr, nil)
			userDB.EXPECT().GetUserName(gomock.Any()).AnyTimes().Return(managerInfos, []string{manager1.ID}, nil)

			out, err := u.GetUsersBaseInfo(ctx, visitor, []string{"xxx", "yyyy"}, rangeInfo, false)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(out), 1)
			assert.Equal(t, out[0].Account, userInfo1.Account)
			assert.Equal(t, out[0].Frozen, true)
			assert.Equal(t, out[0].Authenticated, true)
			assert.Equal(t, out[0].Avatar, "url1")
			assert.Equal(t, out[0].CustomAttr, customAttr)
			assert.Equal(t, out[0].Manager.ID, manager1.ID)
			assert.Equal(t, out[0].Manager.Name, manager1.Name)
		})
	})
}

func TestGetUserInfoByAccount(t *testing.T) {
	Convey("通过账户名匹配账户信息", t, func() {
		test := setGinMode()
		defer test()

		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		cfgLogics := mock.NewMockLogicsConfig(ctrl)
		u := &user{
			userDB: userDB,
			config: cfgLogics,
		}

		id := "f35dcd71-7dda-45e5-b114-4446c7f43ea3"
		account := "account1"
		dbUser := interfaces.UserDBInfo{
			ID:                id,
			PWDErrCnt:         0,
			PWDErrLatestTime:  time.Now().Unix(),
			DisableStatus:     interfaces.Enabled,
			AutoDisableStatus: interfaces.AEnabled,
			LDAPType:          0,
			DomainPath:        "xx",
		}
		config := interfaces.Config{
			EnablePwdLock: false,
		}
		Convey("数据库异常", func() {
			tmpErr := fmt.Errorf("err")
			userDB.EXPECT().GetUserInfoByAccount(gomock.Any()).Return(dbUser, tmpErr)

			_, _, err := u.GetUserInfoByAccount(account, true, true)

			assert.Equal(t, err, tmpErr)
		})
		Convey("根据账户名精确匹配", func() {
			userDB.EXPECT().GetUserInfoByAccount(gomock.Any()).Return(dbUser, nil)
			cfgLogics.EXPECT().GetConfig(gomock.Any()).Return(config, nil)

			result, userInfo, err := u.GetUserInfoByAccount(account, true, true)

			assert.Equal(t, result, true)
			assert.Equal(t, userInfo.ID, id)
			assert.Equal(t, userInfo.PwdErrCnt, 0)
			assert.Equal(t, userInfo.PwdErrLastTime, time.Now().Unix())
			assert.Equal(t, userInfo.Enabled, true)
			assert.Equal(t, userInfo.LDAPType, interfaces.LDAPServerType(0))
			assert.Equal(t, userInfo.DomainPath, "xx")
			assert.Equal(t, err, nil)
		})
		Convey("根据身份证号精确匹配", func() {
			userDB.EXPECT().GetUserInfoByAccount(gomock.Any()).Return(interfaces.UserDBInfo{}, nil)
			userDB.EXPECT().GetUserInfoByIDCard(gomock.Any()).Return(dbUser, nil)
			cfgLogics.EXPECT().GetConfig(gomock.Any()).Return(config, nil)

			result, userInfo, err := u.GetUserInfoByAccount(account, true, true)

			assert.Equal(t, result, true)
			assert.Equal(t, userInfo.ID, id)
			assert.Equal(t, userInfo.PwdErrCnt, 0)
			assert.Equal(t, userInfo.PwdErrLastTime, time.Now().Unix())
			assert.Equal(t, userInfo.Enabled, true)
			assert.Equal(t, userInfo.LDAPType, interfaces.LDAPServerType(0))
			assert.Equal(t, userInfo.DomainPath, "xx")
			assert.Equal(t, err, nil)
		})
		Convey("前缀匹配", func() {
			account = "1111"
			dbUser.Account = "1111@qq.com"
			userDB.EXPECT().GetUserInfoByAccount(gomock.Any()).Return(interfaces.UserDBInfo{}, nil)
			userDB.EXPECT().GetUserInfoByIDCard(gomock.Any()).Return(interfaces.UserDBInfo{}, nil)
			userDB.EXPECT().GetDomainUserInfoByAccount(gomock.Any()).Return(dbUser, nil)
			cfgLogics.EXPECT().GetConfig(gomock.Any()).Return(config, nil)

			result, userInfo, err := u.GetUserInfoByAccount(account, true, true)

			assert.Equal(t, result, true)
			assert.Equal(t, userInfo.ID, id)
			assert.Equal(t, userInfo.PwdErrCnt, 0)
			assert.Equal(t, userInfo.PwdErrLastTime, time.Now().Unix())
			assert.Equal(t, userInfo.Enabled, true)
			assert.Equal(t, userInfo.LDAPType, interfaces.LDAPServerType(0))
			assert.Equal(t, userInfo.DomainPath, "xx")
			assert.Equal(t, err, nil)
		})
		Convey("前缀匹配失败", func() {
			account = "1111"
			dbUser.Account = "1111@qq.com@gmail.com"
			userDB.EXPECT().GetUserInfoByAccount(gomock.Any()).Return(interfaces.UserDBInfo{}, nil)
			userDB.EXPECT().GetUserInfoByIDCard(gomock.Any()).Return(interfaces.UserDBInfo{}, nil)
			userDB.EXPECT().GetDomainUserInfoByAccount(gomock.Any()).Return(dbUser, nil)

			result, _, _ := u.GetUserInfoByAccount(account, true, true)

			assert.Equal(t, result, false)
		})
		Convey("匹配失败", func() {
			userDB.EXPECT().GetUserInfoByAccount(gomock.Any()).Return(interfaces.UserDBInfo{}, nil)
			userDB.EXPECT().GetUserInfoByIDCard(gomock.Any()).Return(interfaces.UserDBInfo{}, nil)
			userDB.EXPECT().GetDomainUserInfoByAccount(gomock.Any()).Return(interfaces.UserDBInfo{}, nil)

			result, userAuthInfo, err := u.GetUserInfoByAccount(account, true, true)

			assert.Equal(t, result, false)
			assert.Equal(t, userAuthInfo.ID, "")
			assert.Equal(t, err, nil)
		})
		Convey("匹配成功，自动解锁账户1", func() {
			config.EnablePwdLock = true
			config.PwdErrCnt = 3
			config.PwdLockTime = 10

			dbUser.AuthType = interfaces.Local
			dbUser.PWDErrCnt = 3
			dbUser.PWDErrLatestTime = time.Now().Unix() - 600

			userDB.EXPECT().GetUserInfoByAccount(gomock.Any()).Return(dbUser, nil)
			cfgLogics.EXPECT().GetConfig(gomock.Any()).Return(config, nil)
			userDB.EXPECT().UpdatePwdErrInfo(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

			result, userInfo, err := u.GetUserInfoByAccount(account, true, true)

			assert.Equal(t, result, true)
			assert.Equal(t, userInfo.ID, id)
			assert.Equal(t, userInfo.PwdErrCnt, 0)
			assert.Equal(t, userInfo.PwdErrLastTime, time.Now().Unix())
			assert.Equal(t, userInfo.Enabled, true)
			assert.Equal(t, userInfo.LDAPType, interfaces.LDAPServerType(0))
			assert.Equal(t, userInfo.DomainPath, "xx")
			assert.Equal(t, err, nil)
		})
		Convey("匹配成功，自动解锁账户2", func() {
			config.EnablePwdLock = true
			config.EnableThirdPwdLock = true
			config.PwdErrCnt = 3
			config.PwdLockTime = 10

			dbUser.AuthType = interfaces.Third
			dbUser.PWDErrCnt = 3
			dbUser.PWDErrLatestTime = time.Now().Unix() - 600

			userDB.EXPECT().GetUserInfoByAccount(gomock.Any()).Return(dbUser, nil)
			cfgLogics.EXPECT().GetConfig(gomock.Any()).Return(config, nil)
			userDB.EXPECT().UpdatePwdErrInfo(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

			result, userInfo, err := u.GetUserInfoByAccount(account, true, true)

			assert.Equal(t, result, true)
			assert.Equal(t, userInfo.ID, id)
			assert.Equal(t, userInfo.PwdErrCnt, 0)
			assert.Equal(t, userInfo.PwdErrLastTime, time.Now().Unix())
			assert.Equal(t, userInfo.Enabled, true)
			assert.Equal(t, userInfo.LDAPType, interfaces.LDAPServerType(0))
			assert.Equal(t, userInfo.DomainPath, "xx")
			assert.Equal(t, err, nil)
		})
	})
}

func TestGetNorlmalUserInfo(t *testing.T) {
	Convey("获取普通用户自身信息", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		departmentDB := mock.NewMockDBDepartment(ctrl)
		contactorDB := mock.NewMockDBContactor(ctrl)
		groupMemberDB := mock.NewMockDBGroupMember(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		ava := mock.NewMockLogicsAvatar(ctrl)
		trace := mock.NewMockTraceClient(ctrl)

		u := &user{
			userDB:        userDB,
			departmentDB:  departmentDB,
			contactorDB:   contactorDB,
			groupMemberDB: groupMemberDB,
			role:          role,
			avatar:        ava,
			trace:         trace,
		}

		var rangeInfo interfaces.UserBaseInfoRange
		visitor := interfaces.Visitor{
			ID:   interfaces.SystemAuditAdmin,
			Type: interfaces.App,
		}
		testErr := rest.NewHTTPError("error", 503000000, nil)

		ctx := context.Background()

		trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
		trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
		trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()
		Convey("应用账户，用户错误", func() {
			_, err := u.GetNorlmalUserInfo(context.Background(), &visitor, rangeInfo)
			assert.Equal(t, err, rest.NewHTTPError("only support normal user", rest.BadRequest, nil))
		})

		visitor.Type = interfaces.RealName
		Convey("用户角色错误，非普通用户", func() {
			_, err := u.GetNorlmalUserInfo(context.Background(), &visitor, rangeInfo)
			assert.Equal(t, err, rest.NewHTTPError("only support normal user", rest.BadRequest, nil))
		})

		visitor.ID = "xxx"
		Convey("获取用户数据库信息失败", func() {
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).AnyTimes().Return(nil, testErr)

			_, err := u.GetNorlmalUserInfo(context.Background(), &visitor, rangeInfo)
			assert.Equal(t, err, testErr)
		})

		temp1 := interfaces.UserDBInfo{
			ID:   visitor.ID,
			Name: "Name1",
		}

		rangeInfo.ShowAvatar = true
		Convey("成功", func() {
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).AnyTimes().Return([]interfaces.UserDBInfo{temp1}, nil)
			ava.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return("xxx", nil)

			info, err := u.GetNorlmalUserInfo(context.Background(), &visitor, rangeInfo)
			assert.Equal(t, err, nil)
			assert.Equal(t, info.Avatar, "xxx")
		})
	})
}

func TestGetPWDRetrievalMethodByAccount(t *testing.T) {
	Convey("根据账户获取密码找回信息", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		departmentDB := mock.NewMockDBDepartment(ctrl)
		contactorDB := mock.NewMockDBContactor(ctrl)
		groupMemberDB := mock.NewMockDBGroupMember(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		ava := mock.NewMockLogicsAvatar(ctrl)
		config := mock.NewMockLogicsConfig(ctrl)

		u := &user{
			userDB:        userDB,
			departmentDB:  departmentDB,
			contactorDB:   contactorDB,
			groupMemberDB: groupMemberDB,
			role:          role,
			avatar:        ava,
			config:        config,
		}

		testErr := rest.NewHTTPError("error", 503000000, nil)
		var userInfo interfaces.UserDBInfo
		Convey("GetUserInfoByAccount 报错", func() {
			userDB.EXPECT().GetUserInfoByAccount(gomock.Any()).AnyTimes().Return(userInfo, testErr)
			_, err := u.GetPWDRetrievalMethodByAccount("")
			assert.Equal(t, err, testErr)
		})

		var curConfig interfaces.Config
		Convey("GetConfig 报错", func() {
			userDB.EXPECT().GetUserInfoByAccount(gomock.Any()).AnyTimes().Return(userInfo, nil)
			config.EXPECT().GetConfig(gomock.Any()).AnyTimes().Return(curConfig, testErr)
			_, err := u.GetPWDRetrievalMethodByAccount("")
			assert.Equal(t, err, testErr)
		})

		userInfo.ID = ""
		curConfig.IDCardLogin = true
		Convey("GetUserInfoByIDCard 报错", func() {
			userDB.EXPECT().GetUserInfoByAccount(gomock.Any()).AnyTimes().Return(userInfo, nil)
			config.EXPECT().GetConfig(gomock.Any()).AnyTimes().Return(curConfig, nil)
			userDB.EXPECT().GetUserInfoByIDCard(gomock.Any()).AnyTimes().Return(userInfo, testErr)
			_, err := u.GetPWDRetrievalMethodByAccount("")
			assert.Equal(t, err, testErr)
		})

		Convey("开启身份证登录，无效账户", func() {
			userDB.EXPECT().GetUserInfoByAccount(gomock.Any()).AnyTimes().Return(userInfo, nil)
			config.EXPECT().GetConfig(gomock.Any()).AnyTimes().Return(curConfig, nil)
			userDB.EXPECT().GetUserInfoByIDCard(gomock.Any()).AnyTimes().Return(userInfo, nil)
			data, err := u.GetPWDRetrievalMethodByAccount("")
			assert.Equal(t, err, nil)
			assert.Equal(t, data.Status, interfaces.PRSInvalidAccount)
		})

		curConfig.IDCardLogin = false
		Convey("未开启身份证登录，无效账户", func() {
			userDB.EXPECT().GetUserInfoByAccount(gomock.Any()).AnyTimes().Return(userInfo, nil)
			config.EXPECT().GetConfig(gomock.Any()).AnyTimes().Return(curConfig, nil)
			data, err := u.GetPWDRetrievalMethodByAccount("")
			assert.Equal(t, err, nil)
			assert.Equal(t, data.Status, interfaces.PRSInvalidAccount)
		})

		userInfo.ID = "userID1"
		userInfo.DisableStatus = interfaces.Disabled
		Convey("用户被禁用", func() {
			userDB.EXPECT().GetUserInfoByAccount(gomock.Any()).AnyTimes().Return(userInfo, nil)
			config.EXPECT().GetConfig(gomock.Any()).AnyTimes().Return(curConfig, nil)
			data, err := u.GetPWDRetrievalMethodByAccount("")
			assert.Equal(t, err, nil)
			assert.Equal(t, data.Status, interfaces.PRSDisableUser)
		})

		userInfo.DisableStatus = interfaces.Enabled
		curConfig.EmailPwdRetrieval = false
		curConfig.TelPwdRetrieval = false
		Convey("密码找回功能未开启", func() {
			userDB.EXPECT().GetUserInfoByAccount(gomock.Any()).AnyTimes().Return(userInfo, nil)
			config.EXPECT().GetConfig(gomock.Any()).AnyTimes().Return(curConfig, nil)
			data, err := u.GetPWDRetrievalMethodByAccount("")
			assert.Equal(t, err, nil)
			assert.Equal(t, data.Status, interfaces.PRSUnablePWDRetrieval)
		})

		curConfig.EmailPwdRetrieval = true
		curConfig.TelPwdRetrieval = true
		userInfo.AuthType = interfaces.Domain
		Convey("非本地账户", func() {
			userDB.EXPECT().GetUserInfoByAccount(gomock.Any()).AnyTimes().Return(userInfo, nil)
			config.EXPECT().GetConfig(gomock.Any()).AnyTimes().Return(curConfig, nil)
			data, err := u.GetPWDRetrievalMethodByAccount("")
			assert.Equal(t, err, nil)
			assert.Equal(t, data.Status, interfaces.PRSNonLocalUser)
		})

		userInfo.AuthType = interfaces.Local
		userInfo.PWDControl = true
		Convey("管控密码开启", func() {
			userDB.EXPECT().GetUserInfoByAccount(gomock.Any()).AnyTimes().Return(userInfo, nil)
			config.EXPECT().GetConfig(gomock.Any()).AnyTimes().Return(curConfig, nil)
			data, err := u.GetPWDRetrievalMethodByAccount("")
			assert.Equal(t, err, nil)
			assert.Equal(t, data.Status, interfaces.PRSEnablePwdControl)
		})

		userInfo.PWDControl = false
		userInfo.Email = "email"
		userInfo.TelNumber = "telnumber"
		Convey("成功", func() {
			userDB.EXPECT().GetUserInfoByAccount(gomock.Any()).AnyTimes().Return(userInfo, nil)
			config.EXPECT().GetConfig(gomock.Any()).AnyTimes().Return(curConfig, nil)
			data, err := u.GetPWDRetrievalMethodByAccount("")
			assert.Equal(t, err, nil)
			assert.Equal(t, data.Status, interfaces.PRSAvaliable)
			assert.Equal(t, data.BEmail, true)
			assert.Equal(t, data.Telephone, userInfo.TelNumber)
			assert.Equal(t, data.BTelephone, true)
			assert.Equal(t, data.Email, userInfo.Email)
		})
	})
}

func TestUserAuth(t *testing.T) {
	Convey("本地认证", t, func() {
		test := setGinMode()
		defer test()

		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		configLogics := mock.NewMockLogicsConfig(ctrl)
		u := &user{
			userDB: userDB,
			config: configLogics,
		}

		id := "f35dcd71-7dda-45e5-b114-4446c7f43ea3"
		dbUser := interfaces.UserDBInfo{}
		config := interfaces.Config{}
		config2 := interfaces.Config{}
		Convey("用户不存在", func() {
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).Return([]interfaces.UserDBInfo{}, nil)

			result, _, err := u.UserAuth(id, "password")

			assert.Equal(t, result, false)
			assert.Equal(t, err, nil)
		})
		Convey("密码错误", func() {
			dbUser.Sha2Password = encodeSha2("password")
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).Return([]interfaces.UserDBInfo{dbUser}, nil)

			result, reason, err := u.UserAuth(id, "wrongpassword")

			assert.Equal(t, result, false)
			assert.Equal(t, reason, interfaces.InvalidPassword)
			assert.Equal(t, err, nil)
		})
		Convey("初始密码错误1", func() {
			dbUser.Password, _ = encodeMD5("123456")
			config2.UserDefaultMd5PWD = "e10adc3949ba59abbe56e057f20f883e"
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).Return([]interfaces.UserDBInfo{dbUser}, nil)
			configLogics.EXPECT().GetConfig(gomock.Any()).Return(config, nil)
			configLogics.EXPECT().GetConfigFromOption(gomock.Any()).Return(config2, nil)

			result, reason, err := u.UserAuth(id, "123456")

			assert.Equal(t, result, false)
			assert.Equal(t, reason, interfaces.InitialPassword)
			assert.Equal(t, err, nil)
		})
		Convey("初始密码错误2", func() {
			dbUser.Sha2Password = encodeSha2("123456")
			config2.UserDefaultSha2PWD = "8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92"
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).Return([]interfaces.UserDBInfo{dbUser}, nil)
			configLogics.EXPECT().GetConfig(gomock.Any()).Return(config, nil)
			configLogics.EXPECT().GetConfigFromOption(gomock.Any()).Return(config2, nil)

			result, reason, err := u.UserAuth(id, "123456")

			assert.Equal(t, result, false)
			assert.Equal(t, reason, interfaces.InitialPassword)
			assert.Equal(t, err, nil)
		})
		Convey("不符合强密码策略", func() {
			dbUser.Sha2Password = encodeSha2("cccccc")
			config.StrongPwdStatus = true
			config.StrongPwdLength = 8
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).Return([]interfaces.UserDBInfo{dbUser}, nil)
			configLogics.EXPECT().GetConfig(gomock.Any()).Return(config, nil)
			configLogics.EXPECT().GetConfigFromOption(gomock.Any()).Return(config2, nil)

			result, reason, err := u.UserAuth(id, "cccccc")

			assert.Equal(t, result, false)
			assert.Equal(t, reason, interfaces.PasswordNotSafe)
			assert.Equal(t, err, nil)
		})
		Convey("管控状态下密码过期", func() {
			config.PwdExpireTime = 1
			dbUser.PWDControl = true
			dbUser.Sha2Password = encodeSha2("cccccc")
			dbUser.PWDTimeStamp = time.Now().Add(-25 * time.Hour).Unix()
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).Return([]interfaces.UserDBInfo{dbUser}, nil)
			configLogics.EXPECT().GetConfig(gomock.Any()).Return(config, nil)
			configLogics.EXPECT().GetConfigFromOption(gomock.Any()).Return(config2, nil)

			result, reason, err := u.UserAuth(id, "cccccc")

			assert.Equal(t, result, false)
			assert.Equal(t, reason, interfaces.UnderControlPasswordExpire)
			assert.Equal(t, err, nil)
		})
		Convey("密码过期", func() {
			config.PwdExpireTime = 1
			dbUser.Sha2Password = encodeSha2("cccccc")
			dbUser.PWDTimeStamp = time.Now().Add(-25 * time.Hour).Unix()
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).Return([]interfaces.UserDBInfo{dbUser}, nil)
			configLogics.EXPECT().GetConfig(gomock.Any()).Return(config, nil)
			configLogics.EXPECT().GetConfigFromOption(gomock.Any()).Return(config2, nil)

			result, reason, err := u.UserAuth(id, "cccccc")

			assert.Equal(t, result, false)
			assert.Equal(t, reason, interfaces.PasswordExpire)
			assert.Equal(t, err, nil)
		})
		Convey("认证成功", func() {
			config.PwdExpireTime = -1
			dbUser.Sha2Password = encodeSha2("cccccc")
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).Return([]interfaces.UserDBInfo{dbUser}, nil)
			configLogics.EXPECT().GetConfig(gomock.Any()).Return(config, nil)
			configLogics.EXPECT().GetConfigFromOption(gomock.Any()).Return(config2, nil)

			result, reason, err := u.UserAuth(id, "cccccc")

			assert.Equal(t, result, true)
			assert.Equal(t, reason, interfaces.AuthFailedReason(0))
			assert.Equal(t, err, nil)
		})
	})
}

func TestUpdatePwdErrInfo(t *testing.T) {
	Convey("更新密码错误信息", t, func() {
		test := setGinMode()
		defer test()

		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		configLogics := mock.NewMockLogicsConfig(ctrl)
		u := &user{
			userDB: userDB,
			config: configLogics,
		}

		config := interfaces.Config{}
		dbUser := interfaces.UserDBInfo{}
		tmpErr1 := rest.NewHTTPError("invalid type", rest.URINotExist, map[string]interface{}{"params": "user_id"})
		tmpErr2 := rest.NewHTTPError("invalid type", rest.BadRequest, map[string]interface{}{"params": "pwd_err_cnt"})
		tmpErr3 := rest.NewHTTPError("invalid type", rest.BadRequest, map[string]interface{}{"params": "pwd_err_last_time, time in the future?"})
		Convey("用户不存在", func() {
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).Return([]interfaces.UserDBInfo{}, nil)

			err := u.UpdatePwdErrInfo("id", -1, time.Now().Unix())

			assert.Equal(t, err, tmpErr1)
		})
		Convey("参数不合法--pwd_err_cnt小于0", func() {
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).Return([]interfaces.UserDBInfo{dbUser}, nil)

			err := u.UpdatePwdErrInfo("id", -1, time.Now().Unix())

			assert.Equal(t, err, tmpErr2)
		})
		Convey("参数不合法--pwd_err_last_time小于0", func() {
			config.PwdErrCnt = 3
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).Return([]interfaces.UserDBInfo{dbUser}, nil)

			err := u.UpdatePwdErrInfo("id", 1, -1)

			assert.Equal(t, err, tmpErr3)
		})
		Convey("参数不合法--pwd_err_last_time大于当前时间", func() {
			config.PwdErrCnt = 3
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).Return([]interfaces.UserDBInfo{dbUser}, nil)

			err := u.UpdatePwdErrInfo("id", 1, time.Now().Add(time.Second*6).Unix())

			assert.Equal(t, err, tmpErr3)
		})
		Convey("更新成功", func() {
			config.PwdErrCnt = 3
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).Return([]interfaces.UserDBInfo{dbUser}, nil)
			userDB.EXPECT().UpdatePwdErrInfo(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

			err := u.UpdatePwdErrInfo("id", 1, time.Now().Add(time.Second*4).Unix())

			assert.Equal(t, err, nil)
		})
	})
}

//nolint:dupl,funlen
func TestIncrementModifyUserInfo(t *testing.T) {
	Convey("增量修改用户信息", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		departmentDB := mock.NewMockDBDepartment(ctrl)
		contactorDB := mock.NewMockDBContactor(ctrl)
		groupMemberDB := mock.NewMockDBGroupMember(ctrl)
		configDB := mock.NewMockDBConfig(ctrl)
		hydra := mock.NewMockDrivenHydra(ctrl)
		ob := mock.NewMockLogicsOutbox(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		trace := mock.NewMockTraceClient(ctrl)

		dPool, txMock, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func() {
			if closeErr := dPool.Close(); closeErr != nil {
				assert.Equal(t, 1, 1)
			}
		}()

		u := &user{
			userDB:        userDB,
			departmentDB:  departmentDB,
			contactorDB:   contactorDB,
			groupMemberDB: groupMemberDB,
			pool:          dPool,
			tracePool:     dPool,
			configDB:      configDB,
			hydra:         hydra,
			ob:            ob,
			logger:        common.NewLogger(),
			role:          role,
			trace:         trace,
		}

		var data interfaces.UserUpdateRange
		data.CustomAttr = true
		var data2 interfaces.UserBaseInfo
		var visitor interfaces.Visitor
		ctx := context.Background()

		visitor.Type = interfaces.RealName
		visitor.ID = "zz1"
		Convey("获取用户角色信息失败", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()
			testErr := rest.NewHTTPError("error", 503000000, nil)
			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, testErr)

			err := u.IncrementModifyUserInfo(ctx, &visitor, data, &data2)
			assert.Equal(t, err, testErr)
		})

		roleInfo := make(map[string]map[interfaces.Role]bool)
		mapRole := make(map[interfaces.Role]bool)
		roleInfo[visitor.ID] = mapRole
		Convey("用户无权限", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()
			testErr1 := rest.NewHTTPErrorV2(rest.Forbidden, "this user do not has the authority")
			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return(roleInfo, testErr1)

			err := u.IncrementModifyUserInfo(ctx, &visitor, data, &data2)
			assert.Equal(t, err, testErr1)
		})

		mapRole[interfaces.SystemRoleSuperAdmin] = true
		Convey("参数检查失败", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()
			testErr := rest.NewHTTPError("error", 503000000, nil)
			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return(roleInfo, nil)
			userDB.EXPECT().GetUserDBInfo2(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, testErr)

			err := u.IncrementModifyUserInfo(ctx, &visitor, data, &data2)
			assert.Equal(t, err, testErr)
		})

		tempUser := interfaces.UserDBInfo{}
		tempUser.AuthType = interfaces.Local
		userInfo := []interfaces.UserDBInfo{tempUser}
		map1 := make(map[string]interface{}, 0)
		map1["test1"] = "test1"
		data2.CustomAttr = map1

		Convey("db pool 初始化失败", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()
			testErr := rest.NewHTTPError("error", 503000000, nil)
			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return(roleInfo, nil)
			userDB.EXPECT().GetUserDBInfo2(gomock.Any(), gomock.Any()).AnyTimes().Return(userInfo, nil)
			txMock.ExpectBegin().WillReturnError(testErr)

			err := u.IncrementModifyUserInfo(ctx, &visitor, data, &data2)
			assert.NotEqual(t, err, nil)
		})

		Convey("获取自定义配置报错", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()
			testErr := rest.NewHTTPError("error", 503000000, nil)
			txMock.ExpectBegin()
			txMock.ExpectRollback()
			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return(roleInfo, nil)
			userDB.EXPECT().GetUserDBInfo2(gomock.Any(), gomock.Any()).AnyTimes().Return(userInfo, nil)
			userDB.EXPECT().GetUserCustomAttr(gomock.Any()).AnyTimes().Return(nil, testErr)

			err := u.IncrementModifyUserInfo(ctx, &visitor, data, &data2)
			assert.Equal(t, err, testErr)
		})

		customAttr := make(map[string]interface{}, 0)
		Convey("添加用户自定义属性失败", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()
			testErr := rest.NewHTTPError("error", 503000000, nil)
			txMock.ExpectBegin()
			txMock.ExpectRollback()
			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return(roleInfo, nil)
			userDB.EXPECT().GetUserDBInfo2(gomock.Any(), gomock.Any()).AnyTimes().Return(userInfo, nil)
			userDB.EXPECT().GetUserCustomAttr(gomock.Any()).AnyTimes().Return(customAttr, nil)
			userDB.EXPECT().AddUserCustomAttr(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(testErr)

			err := u.IncrementModifyUserInfo(ctx, &visitor, data, &data2)
			assert.Equal(t, err, testErr)
		})

		Convey("添加用户自定义属性成功", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()
			txMock.ExpectBegin()
			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return(roleInfo, nil)
			userDB.EXPECT().GetUserDBInfo2(gomock.Any(), gomock.Any()).AnyTimes().Return(userInfo, nil)
			userDB.EXPECT().GetUserCustomAttr(gomock.Any()).AnyTimes().Return(customAttr, nil)
			userDB.EXPECT().AddUserCustomAttr(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

			txMock.ExpectCommit()

			err := u.IncrementModifyUserInfo(ctx, &visitor, data, &data2)
			assert.Equal(t, err, nil)
		})

		customAttr["test2"] = "test2"
		Convey("更新用户自定义属性失败", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()
			testErr := rest.NewHTTPError("error", 503000000, nil)
			txMock.ExpectBegin()
			txMock.ExpectRollback()
			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return(roleInfo, nil)
			userDB.EXPECT().GetUserDBInfo2(gomock.Any(), gomock.Any()).AnyTimes().Return(userInfo, nil)
			userDB.EXPECT().GetUserCustomAttr(gomock.Any()).AnyTimes().Return(customAttr, nil)
			userDB.EXPECT().UpdateUserCustomAttr(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(testErr)

			err := u.IncrementModifyUserInfo(ctx, &visitor, data, &data2)
			assert.Equal(t, err, testErr)
		})

		Convey("更新用户自定义属性成功", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()
			txMock.ExpectBegin()
			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return(roleInfo, nil)
			userDB.EXPECT().GetUserDBInfo2(gomock.Any(), gomock.Any()).AnyTimes().Return(userInfo, nil)
			userDB.EXPECT().GetUserCustomAttr(gomock.Any()).AnyTimes().Return(customAttr, nil)
			userDB.EXPECT().UpdateUserCustomAttr(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

			txMock.ExpectCommit()

			err := u.IncrementModifyUserInfo(ctx, &visitor, data, &data2)
			assert.Equal(t, err, nil)
		})

		Convey("更新用户自定义属性成功1", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()
			txMock.ExpectBegin()
			userDB.EXPECT().GetUserDBInfo2(gomock.Any(), gomock.Any()).AnyTimes().Return(userInfo, nil)
			userDB.EXPECT().GetUserCustomAttr(gomock.Any()).AnyTimes().Return(customAttr, nil)
			userDB.EXPECT().UpdateUserCustomAttr(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

			txMock.ExpectCommit()

			err := u.IncrementModifyUserInfo(ctx, nil, data, &data2)
			assert.Equal(t, err, nil)
		})
	})
}

func TestGetUsersBaseInfoByTelephones(t *testing.T) {
	Convey("根据电话批量获取用户信息", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		departmentDB := mock.NewMockDBDepartment(ctrl)
		contactorDB := mock.NewMockDBContactor(ctrl)
		groupMemberDB := mock.NewMockDBGroupMember(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		ava := mock.NewMockLogicsAvatar(ctrl)
		trace := mock.NewMockTraceClient(ctrl)
		orgPerm := mock.NewMockDBOrgPermApp(ctrl)

		u := &user{
			userDB:        userDB,
			departmentDB:  departmentDB,
			contactorDB:   contactorDB,
			groupMemberDB: groupMemberDB,
			role:          role,
			avatar:        ava,
			trace:         trace,
			orgPermAppDB:  orgPerm,
		}

		var rangeInfo interfaces.UserBaseInfoRange
		ctx := context.Background()
		visitor := interfaces.Visitor{}

		Convey("非应用账户", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			_, _, err := u.GetUserBaseInfoByTelephone(ctx, &visitor, strName, rangeInfo)
			assert.Equal(t, err, rest.NewHTTPErrorV2(rest.Forbidden, "this user has no authority"))
		})

		testPerm := make(map[interfaces.OrgType]interfaces.AppOrgPerm)
		visitor.Type = interfaces.App
		Convey("应用账户无权限", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			orgPerm.EXPECT().GetAppPermByID(gomock.Any()).AnyTimes().Return(testPerm, nil)

			_, _, err := u.GetUserBaseInfoByTelephone(ctx, &visitor, strName, rangeInfo)
			assert.Equal(t, err, rest.NewHTTPError("this user do not has the authority", rest.Forbidden, nil))
		})

		testPerm[interfaces.User] = interfaces.AppOrgPerm{
			Value: interfaces.Read,
		}

		Convey("密码解密失败", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			orgPerm.EXPECT().GetAppPermByID(gomock.Any()).AnyTimes().Return(testPerm, nil)

			_, _, err := u.GetUserBaseInfoByTelephone(ctx, &visitor, strName, rangeInfo)
			tempErr, ok := err.(*rest.HTTPError)
			assert.Equal(t, ok, true)
			assert.Equal(t, tempErr.Code, rest.BadRequest)
		})

		testErr := rest.NewHTTPErrorV2(rest.BadRequest, strID)
		Convey("GetUserDBInfoByTels，报错", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			orgPerm.EXPECT().GetAppPermByID(gomock.Any()).AnyTimes().Return(testPerm, nil)
			userDB.EXPECT().GetUserDBInfoByTels(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, testErr)

			_, _, err := u.GetUserBaseInfoByTelephone(ctx, &visitor, cRSA2048, rangeInfo)
			assert.Equal(t, err, testErr)
		})

		rangeInfo.ShowAccount = true
		rangeInfo.ShowName = true
		rangeInfo.ShowEmail = true
		rangeInfo.ShowTelNumber = true
		rangeInfo.ShowThirdID = true
		userInfo1 := interfaces.UserDBInfo{
			ID:        strID,
			Account:   strID2,
			Name:      strName,
			Email:     strName1,
			TelNumber: strName2,
			ThirdID:   strID1,
		}
		Convey("success", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			orgPerm.EXPECT().GetAppPermByID(gomock.Any()).AnyTimes().Return(testPerm, nil)
			userDB.EXPECT().GetUserDBInfoByTels(gomock.Any(), gomock.Any()).AnyTimes().Return([]interfaces.UserDBInfo{userInfo1}, nil)

			result, out, err := u.GetUserBaseInfoByTelephone(ctx, &visitor, cRSA2048, rangeInfo)
			assert.Equal(t, err, nil)
			assert.Equal(t, result, true)
			assert.Equal(t, out.ID, userInfo1.ID)
			assert.Equal(t, out.Account, userInfo1.Account)
			assert.Equal(t, out.Name, userInfo1.Name)
			assert.Equal(t, out.Email, userInfo1.Email)
			assert.Equal(t, out.TelNumber, userInfo1.TelNumber)
			assert.Equal(t, out.ThirdID, userInfo1.ThirdID)
		})

		Convey("success 1", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			orgPerm.EXPECT().GetAppPermByID(gomock.Any()).AnyTimes().Return(testPerm, nil)
			userDB.EXPECT().GetUserDBInfoByTels(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)

			result, _, err := u.GetUserBaseInfoByTelephone(ctx, &visitor, cRSA2048, rangeInfo)
			assert.Equal(t, err, nil)
			assert.Equal(t, result, false)
		})
	})
}

func TestUser_onUserDeleted(t *testing.T) {
	Convey("Test onUserDeleted", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserDB := mock.NewMockDBUser(ctrl)

		u := &user{
			userDB: mockUserDB,
			logger: common.NewLogger(),
		}

		Convey("When DeleteUserManagerID succeeds", func() {
			mockUserDB.EXPECT().DeleteUserManagerID(strID).Return(nil)

			err := u.onUserDeleted(strID)
			assert.Equal(t, err, nil)
		})
	})
}

//nolint:funlen
func TestSearchUsers(t *testing.T) {
	Convey("用户搜索-逻辑层", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		departmentDB := mock.NewMockDBDepartment(ctrl)
		contactorDB := mock.NewMockDBContactor(ctrl)
		groupMemberDB := mock.NewMockDBGroupMember(ctrl)
		orgPerm := mock.NewMockLogicsOrgPerm(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		trace := mock.NewMockTraceClient(ctrl)
		userLogics := newUser(userDB, departmentDB, contactorDB, groupMemberDB, role)
		userLogics.trace = trace
		userLogics.orgPerm = orgPerm

		var info interfaces.OrgShowPageInfo
		info.Role = interfaces.SystemRoleOrgAudit
		var visitor interfaces.Visitor
		visitor.ID = strID
		visitor.Language = interfaces.SimplifiedChinese
		ctx := context.Background()

		ks := interfaces.UserSearchInDepartKeyScope{}
		k := interfaces.UserSearchInDepartKey{}
		f := interfaces.UserBaseInfoRange{}

		userLogics.i18n = common.NewI18n(common.I18nMap{
			i18nIDObjectsInUnDistributeUserGroup: {
				interfaces.SimplifiedChinese:  "未分配组",
				interfaces.TraditionalChinese: "未分配組",
				interfaces.AmericanEnglish:    "Unassigned Group",
			},
		})

		testErr := rest.NewHTTPError("error", 503000000, nil)
		Convey("检查是否有指定角色-报错", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, testErr)

			_, _, err := userLogics.SearchUsers(ctx, &visitor, &ks, &k, f, interfaces.SystemRoleOrgAudit)
			assert.Equal(t, err, testErr)
		})

		Convey("检查是否有指定角色-没有指定角色", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)

			_, _, err := userLogics.SearchUsers(ctx, &visitor, &ks, &k, f, interfaces.SystemRoleOrgAudit)
			assert.Equal(t, err, rest.NewHTTPErrorV2(rest.BadRequest, "this user do not has this role"))
		})

		tempxxx := map[interfaces.Role]bool{
			interfaces.SystemRoleOrgAudit: true,
		}
		testRoles := map[string]map[interfaces.Role]bool{
			visitor.ID: tempxxx,
		}
		ks.BDepartmentID = true
		k.DepartmentID = strID2
		Convey("部门下搜索，报错", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return(testRoles, nil)
			departmentDB.EXPECT().GetDepartmentInfoByIDs(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, testErr)

			_, _, err := userLogics.SearchUsers(ctx, &visitor, &ks, &k, f, interfaces.SystemRoleOrgAudit)
			assert.Equal(t, err, testErr)
		})

		Convey("部门下搜索，部门不存在", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return(testRoles, nil)
			departmentDB.EXPECT().GetDepartmentInfoByIDs(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)

			_, _, err := userLogics.SearchUsers(ctx, &visitor, &ks, &k, f, interfaces.SystemRoleOrgAudit)
			assert.Equal(t, err, rest.NewHTTPErrorV2(rest.BadRequest, "this department not exist"))
		})

		tempDepartInfo := interfaces.DepartmentDBInfo{}
		Convey("部门下搜索，部门存在，检查权限报错", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return(testRoles, nil)
			departmentDB.EXPECT().GetDepartmentInfoByIDs(gomock.Any(), gomock.Any()).AnyTimes().Return([]interfaces.DepartmentDBInfo{tempDepartInfo}, nil)
			userDB.EXPECT().GetOrgAduitDepartInfo2(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, testErr)

			_, _, err := userLogics.SearchUsers(ctx, &visitor, &ks, &k, f, interfaces.SystemRoleOrgAudit)
			assert.Equal(t, err, testErr)
		})

		Convey("部门下搜索，部门存在，检查权限没有权限", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return(testRoles, nil)
			departmentDB.EXPECT().GetDepartmentInfoByIDs(gomock.Any(), gomock.Any()).AnyTimes().Return([]interfaces.DepartmentDBInfo{tempDepartInfo}, nil)
			userDB.EXPECT().GetOrgAduitDepartInfo2(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)

			_, _, err := userLogics.SearchUsers(ctx, &visitor, &ks, &k, f, interfaces.SystemRoleOrgAudit)
			assert.Equal(t, err, rest.NewHTTPErrorV2(rest.Forbidden, "this user do not has this authority"))
		})

		ks.BDepartmentID = false
		Convey("所有用户中搜索，无权限", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return(testRoles, nil)

			_, _, err := userLogics.SearchUsers(ctx, &visitor, &ks, &k, f, interfaces.SystemRoleOrgAudit)
			assert.Equal(t, err, rest.NewHTTPErrorV2(rest.Forbidden, "this user has no authority"))
		})

		testRoles[visitor.ID][interfaces.SystemRoleSuperAdmin] = true
		Convey("SearchUsers搜索报错", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return(testRoles, nil)
			userDB.EXPECT().SearchUsers(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, testErr)

			_, _, err := userLogics.SearchUsers(ctx, &visitor, &ks, &k, f, interfaces.SystemRoleSuperAdmin)
			assert.Equal(t, err, testErr)
		})

		Convey("SearchUsersCount搜索报错", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return(testRoles, nil)
			userDB.EXPECT().SearchUsers(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
			userDB.EXPECT().SearchUsersCount(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(0, testErr)

			_, _, err := userLogics.SearchUsers(ctx, &visitor, &ks, &k, f, interfaces.SystemRoleSuperAdmin)
			assert.Equal(t, err, testErr)
		})

		user1 := interfaces.UserDBInfo{}
		userDBInfos := []interfaces.UserDBInfo{user1}
		f.ShowParentDeps = true
		Convey("GetDirectBelongDepartmentIDs2搜索报错", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return(testRoles, nil)
			userDB.EXPECT().SearchUsers(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(userDBInfos, nil)
			userDB.EXPECT().SearchUsersCount(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(0, nil)
			userDB.EXPECT().GetDirectBelongDepartmentIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil, testErr)

			_, _, err := userLogics.SearchUsers(ctx, &visitor, &ks, &k, f, interfaces.SystemRoleSuperAdmin)
			assert.Equal(t, err, testErr)
		})

		Convey("GetRolesByUserIDs搜索报错", func() {
			f.ShowRoles = true
			f.ShowParentDeps = false
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return(testRoles, nil)
			userDB.EXPECT().SearchUsers(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(userDBInfos, nil)
			userDB.EXPECT().SearchUsersCount(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(0, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(nil, testErr)

			_, _, err := userLogics.SearchUsers(ctx, &visitor, &ks, &k, f, interfaces.SystemRoleSuperAdmin)
			assert.Equal(t, err, testErr)
		})

		Convey("GetUserName搜索报错", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			f.ShowParentDeps = false
			f.ShowManager = true
			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return(testRoles, nil)
			userDB.EXPECT().SearchUsers(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(userDBInfos, nil)
			userDB.EXPECT().SearchUsersCount(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(0, nil)
			userDB.EXPECT().GetUserName(gomock.Any()).AnyTimes().Return(nil, nil, testErr)

			_, _, err := userLogics.SearchUsers(ctx, &visitor, &ks, &k, f, interfaces.SystemRoleSuperAdmin)
			assert.Equal(t, err, testErr)
		})

		user1 = interfaces.UserDBInfo{
			ID:                 strID,
			Name:               strName,
			Account:            strName1,
			CSFLevel:           11,
			AuthType:           interfaces.Domain,
			Priority:           12,
			CreatedAtTimeStamp: 111111,
			AutoDisableStatus:  interfaces.AEnabled,
			DisableStatus:      interfaces.Enabled,
			Code:               strName2,
			Position:           strRSA1024PrivateKey,
			Frozen:             true,
		}

		tempxxx1 := map[interfaces.Role]bool{
			interfaces.SystemRoleSuperAdmin: true,
			interfaces.SystemRoleNormalUser: true,
		}
		testRoles1 := map[string]map[interfaces.Role]bool{
			user1.ID: tempxxx1,
		}

		Convey("success", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			f.ShowParentDeps = false
			f.ShowManager = false
			f.ShowRoles = true
			f.ShowName = true
			f.ShowAccount = true
			f.ShowCSFLevel = true
			f.ShowAuthType = true
			f.ShowPriority = true
			f.ShowCreatedAt = true
			f.ShowEnable = true
			f.ShowCode = true
			f.ShowPosition = true
			f.ShowFrozen = true
			userDBInfos = []interfaces.UserDBInfo{user1}
			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return(testRoles, nil)
			userDB.EXPECT().SearchUsers(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(userDBInfos, nil)
			userDB.EXPECT().SearchUsersCount(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(10, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(testRoles1, nil)

			outs, num, err := userLogics.SearchUsers(ctx, &visitor, &ks, &k, f, interfaces.SystemRoleSuperAdmin)
			assert.Equal(t, err, nil)
			assert.Equal(t, num, 10)
			assert.Equal(t, len(outs), 1)
			assert.Equal(t, outs[0].Account, user1.Account)
			assert.Equal(t, outs[0].Name, user1.Name)
			assert.Equal(t, outs[0].Account, user1.Account)
			assert.Equal(t, outs[0].CSFLevel, user1.CSFLevel)
			assert.Equal(t, outs[0].AuthType, user1.AuthType)
			assert.Equal(t, outs[0].Priority, user1.Priority)
			assert.Equal(t, outs[0].CreatedAt, user1.CreatedAtTimeStamp)
			assert.Equal(t, outs[0].Enabled, true)
			assert.Equal(t, outs[0].Code, user1.Code)
			assert.Equal(t, outs[0].Position, user1.Position)
			assert.Equal(t, outs[0].Frozen, user1.Frozen)

			assert.Equal(t, len(outs[0].VecRoles), 2)
			data1 := map[interfaces.Role]bool{
				outs[0].VecRoles[0]: true,
				outs[0].VecRoles[1]: true,
			}
			assert.Equal(t, data1[interfaces.SystemRoleSuperAdmin], true)
			assert.Equal(t, data1[interfaces.SystemRoleNormalUser], true)
		})

		Convey("success 未分配组", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			f.ShowParentDeps = true
			f.ShowManager = false
			f.ShowRoles = false
			f.ShowName = false
			f.ShowAccount = false
			f.ShowCSFLevel = false
			f.ShowAuthType = false
			f.ShowPriority = false
			f.ShowCreatedAt = false
			f.ShowEnable = false
			f.ShowCode = false
			f.ShowPosition = false
			f.ShowFrozen = false
			userDBInfos = []interfaces.UserDBInfo{user1}
			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return(testRoles, nil)
			userDB.EXPECT().SearchUsers(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(userDBInfos, nil)
			userDB.EXPECT().SearchUsersCount(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(10, nil)
			userDB.EXPECT().GetDirectBelongDepartmentIDs2(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil, nil)
			outs, num, err := userLogics.SearchUsers(ctx, &visitor, &ks, &k, f, interfaces.SystemRoleSuperAdmin)
			assert.Equal(t, err, nil)
			assert.Equal(t, num, 10)
			assert.Equal(t, len(outs), 1)
			assert.Equal(t, len(outs[0].ParentDeps), 1)
			assert.Equal(t, outs[0].ParentDeps[0][0].ID, "-1")
			assert.Equal(t, outs[0].ParentDeps[0][0].Name, "未分配组")
			assert.Equal(t, outs[0].ParentDeps[0][0].Type, "department")
			assert.Equal(t, outs[0].ParentDeps[0][0].Code, "")
		})
	})
}

func TestGetManagerInfos(t *testing.T) {
	Convey("获取直属上级-逻辑层", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		departmentDB := mock.NewMockDBDepartment(ctrl)
		contactorDB := mock.NewMockDBContactor(ctrl)
		groupMemberDB := mock.NewMockDBGroupMember(ctrl)
		orgPerm := mock.NewMockLogicsOrgPerm(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		trace := mock.NewMockTraceClient(ctrl)
		userLogics := newUser(userDB, departmentDB, contactorDB, groupMemberDB, role)
		userLogics.trace = trace
		userLogics.orgPerm = orgPerm
		userLogics.logger = common.NewLogger()

		var info interfaces.OrgShowPageInfo
		info.Role = interfaces.SystemRoleOrgAudit
		var visitor interfaces.Visitor
		visitor.ID = strID

		user1 := interfaces.UserDBInfo{
			ID:        strID,
			ManagerID: strName,
		}
		testErr := rest.NewHTTPError("error", 503000000, nil)
		Convey("GetUserName-报错", func() {
			userDB.EXPECT().GetUserName(gomock.Any()).AnyTimes().Return(nil, nil, testErr)

			_, err := userLogics.getManagerInfos([]interfaces.UserDBInfo{user1})
			assert.Equal(t, err, testErr)
		})

		manager1 := interfaces.UserDBInfo{
			ID:   user1.ManagerID,
			Name: strID2,
		}
		Convey("success", func() {
			userDB.EXPECT().GetUserName(gomock.Any()).AnyTimes().Return([]interfaces.UserDBInfo{manager1}, []string{}, nil)

			out, err := userLogics.getManagerInfos([]interfaces.UserDBInfo{user1})
			assert.Equal(t, err, nil)
			assert.Equal(t, len(out), 1)
			assert.Equal(t, out[manager1.ID].ID, manager1.ID)
			assert.Equal(t, out[manager1.ID].Name, manager1.Name)
		})
	})
}

func TestCheckDepartmentInUserScope(t *testing.T) {
	Convey("检查部门是否在用户管理范围内-逻辑层", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		departmentDB := mock.NewMockDBDepartment(ctrl)
		contactorDB := mock.NewMockDBContactor(ctrl)
		groupMemberDB := mock.NewMockDBGroupMember(ctrl)
		orgPerm := mock.NewMockLogicsOrgPerm(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		trace := mock.NewMockTraceClient(ctrl)
		userLogics := newUser(userDB, departmentDB, contactorDB, groupMemberDB, role)
		userLogics.trace = trace
		userLogics.orgPerm = orgPerm

		ctx := context.Background()
		visitor := interfaces.Visitor{}

		testErr := rest.NewHTTPError("error", 503000000, nil)
		Convey("如果是内置管理员，则返回true", func() {
			result, err := userLogics.checkDepartmentInUserScope(ctx, &visitor, interfaces.SystemRoleSuperAdmin, strID)
			assert.Equal(t, err, nil)
			assert.Equal(t, result, true)
		})

		Convey("如果是组织管理员，GetOrgManagerDepartInfo2报错", func() {
			userDB.EXPECT().GetOrgManagerDepartInfo2(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, testErr)
			_, err := userLogics.checkDepartmentInUserScope(ctx, &visitor, interfaces.SystemRoleOrgManager, strID)
			assert.Equal(t, err, testErr)
		})

		Convey("如果是组织审计员，GetOrgAduitDepartInfo2报错", func() {
			userDB.EXPECT().GetOrgAduitDepartInfo2(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, testErr)
			_, err := userLogics.checkDepartmentInUserScope(ctx, &visitor, interfaces.SystemRoleOrgAudit, strID)
			assert.Equal(t, err, testErr)
		})

		Convey("如果是组织审计员，不在范围内", func() {
			userDB.EXPECT().GetOrgAduitDepartInfo2(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
			result, err := userLogics.checkDepartmentInUserScope(ctx, &visitor, interfaces.SystemRoleOrgAudit, strID)
			assert.Equal(t, err, nil)
			assert.Equal(t, result, false)
		})

		Convey("如果是组织审计员，在范围内", func() {
			userDB.EXPECT().GetOrgAduitDepartInfo2(gomock.Any(), gomock.Any()).AnyTimes().Return([]string{strID}, nil)
			result, err := userLogics.checkDepartmentInUserScope(ctx, &visitor, interfaces.SystemRoleOrgAudit, strID)
			assert.Equal(t, err, nil)
			assert.Equal(t, result, true)
		})
	})
}

func TestCheckUserNameExistd(t *testing.T) {
	Convey("检查用户名是否存在", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		reservedName := mock.NewMockLogicsReservedName(ctrl)

		trace := mock.NewMockTraceClient(ctrl)
		ctx := context.Background()

		trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
		trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
		trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

		u := &user{
			userDB:       userDB,
			reservedName: reservedName,
			logger:       common.NewLogger(),
			trace:        trace,
		}

		userInfo := interfaces.UserDBInfo{
			ID: "123",
		}
		rName := interfaces.ReservedNameInfo{
			ID: "123",
		}

		name := "name"

		Convey("name为空", func() {
			name = ""
			result, err := u.CheckUserNameExistd(ctx, name)
			assert.Equal(t, err, rest.NewHTTPErrorV2(rest.BadRequest, "invalid name"))
			assert.Equal(t, result, false)
		})

		Convey("查询用户信息失败", func() {
			userDB.EXPECT().GetUserInfoByName(gomock.Any(), gomock.Any()).Return(userInfo, stdErr.New(strTest))
			result, err := u.CheckUserNameExistd(ctx, name)
			assert.Equal(t, err, stdErr.New(strTest))
			assert.Equal(t, result, false)
		})

		Convey("用户存在", func() {
			userDB.EXPECT().GetUserInfoByName(gomock.Any(), gomock.Any()).Return(userInfo, nil)
			result, err := u.CheckUserNameExistd(ctx, name)
			assert.Equal(t, err, nil)
			assert.Equal(t, result, true)
		})

		Convey("用户不存在，获取保留名称失败", func() {
			userInfo.ID = ""
			userDB.EXPECT().GetUserInfoByName(gomock.Any(), gomock.Any()).Return(userInfo, nil)
			reservedName.EXPECT().GetReservedName(gomock.Any()).Return(rName, stdErr.New(strTest))
			result, err := u.CheckUserNameExistd(ctx, name)
			assert.Equal(t, err, stdErr.New(strTest))
			assert.Equal(t, result, false)
		})

		Convey("用户不存在，获取保留名称成功", func() {
			userInfo.ID = ""
			userDB.EXPECT().GetUserInfoByName(gomock.Any(), gomock.Any()).AnyTimes().Return(userInfo, nil)
			Convey("保留名称存在", func() {
				rName.ID = "123"
				reservedName.EXPECT().GetReservedName(gomock.Any()).Return(rName, nil)
				result, err := u.CheckUserNameExistd(ctx, name)
				assert.Equal(t, err, nil)
				assert.Equal(t, result, true)
			})

			Convey("保留名称不存在", func() {
				rName.ID = ""
				reservedName.EXPECT().GetReservedName(gomock.Any()).Return(rName, nil)
				result, err := u.CheckUserNameExistd(ctx, name)
				assert.Equal(t, err, nil)
				assert.Equal(t, result, false)
			})
		})
	})
}

func TestGetUserList(t *testing.T) {
	Convey("获取用户列表", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		reservedName := mock.NewMockLogicsReservedName(ctrl)

		trace := mock.NewMockTraceClient(ctrl)
		ctx := context.Background()

		trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
		trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
		trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

		u := &user{
			userDB:       userDB,
			reservedName: reservedName,
			logger:       common.NewLogger(),
			trace:        trace,
		}

		userInfoRange := interfaces.UserBaseInfoRange{
			ShowName:      true,
			ShowAccount:   true,
			ShowTelNumber: true,
			ShowCreatedAt: true,
			ShowEmail:     true,
			ShowEnable:    true,
			ShowFrozen:    true,
		}
		direction := interfaces.Desc
		createdStamp := int64(0)
		userID := ""
		limit := 0

		userDBInfos := []interfaces.UserDBInfo{
			{
				ID:                 strID,
				Account:            strID2,
				Name:               strID2,
				DisableStatus:      interfaces.Enabled,
				AutoDisableStatus:  interfaces.AEnabled,
				Email:              strID1,
				TelNumber:          strID,
				CreatedAtTimeStamp: time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local).Unix(),
				Frozen:             true,
			},
			{
				ID:                 strID1,
				Account:            strID2,
				Name:               strID2,
				DisableStatus:      interfaces.Enabled,
				AutoDisableStatus:  interfaces.ADisabled,
				Email:              strID2,
				TelNumber:          strID1,
				CreatedAtTimeStamp: time.Date(2025, 2, 1, 0, 0, 0, 0, time.Local).Unix(),
				Frozen:             false,
			},
		}

		Convey("GetUserList-报错", func() {
			userDB.EXPECT().GetUserList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, stdErr.New(strTest))
			_, _, _, err := u.GetUserList(ctx, userInfoRange, direction, true, createdStamp, userID, limit)
			assert.Equal(t, err, stdErr.New(strTest))
		})

		Convey("GetAllUserCount 报错", func() {
			userDB.EXPECT().GetUserList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
			userDB.EXPECT().GetAllUserCount(gomock.Any()).AnyTimes().Return(0, stdErr.New(strTest))
			_, _, _, err := u.GetUserList(ctx, userInfoRange, direction, true, createdStamp, userID, limit)
			assert.Equal(t, err, stdErr.New(strTest))
		})

		Convey("success1, 如果只获取1个记录", func() {
			userDB.EXPECT().GetUserList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(userDBInfos, nil)
			userDB.EXPECT().GetAllUserCount(gomock.Any()).AnyTimes().Return(1, nil)
			out, num, hasNext, err := u.GetUserList(ctx, userInfoRange, direction, true, createdStamp, userID, 1)
			assert.Equal(t, err, nil)
			assert.Equal(t, num, 1)
			assert.Equal(t, hasNext, true)
			assert.Equal(t, len(out), 1)
			assert.Equal(t, out[0].ID, strID)
			assert.Equal(t, out[0].Account, strID2)
			assert.Equal(t, out[0].Name, strID2)
			assert.Equal(t, out[0].Enabled, true)
			assert.Equal(t, out[0].Email, strID1)
			assert.Equal(t, out[0].TelNumber, strID)
			assert.Equal(t, out[0].CreatedAt, time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local).Unix())
			assert.Equal(t, out[0].Frozen, true)
		})

		Convey("success1, 如果只获取2个记录", func() {
			userDB.EXPECT().GetUserList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(userDBInfos, nil)
			userDB.EXPECT().GetAllUserCount(gomock.Any()).AnyTimes().Return(1, nil)
			out, num, hasNext, err := u.GetUserList(ctx, userInfoRange, direction, true, createdStamp, userID, 2)
			assert.Equal(t, err, nil)
			assert.Equal(t, num, 1)
			assert.Equal(t, hasNext, false)
			assert.Equal(t, len(out), 2)
			assert.Equal(t, out[0].ID, strID)
			assert.Equal(t, out[0].Account, strID2)
			assert.Equal(t, out[0].Name, strID2)
			assert.Equal(t, out[0].Enabled, true)
			assert.Equal(t, out[0].Email, strID1)
			assert.Equal(t, out[0].TelNumber, strID)
			assert.Equal(t, out[0].CreatedAt, time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local).Unix())
			assert.Equal(t, out[0].Frozen, true)
			assert.Equal(t, out[1].ID, strID1)
			assert.Equal(t, out[1].Account, strID2)
			assert.Equal(t, out[1].Name, strID2)
			assert.Equal(t, out[1].Enabled, false)
			assert.Equal(t, out[1].Email, strID2)
			assert.Equal(t, out[1].TelNumber, strID1)
			assert.Equal(t, out[1].CreatedAt, time.Date(2025, 2, 1, 0, 0, 0, 0, time.Local).Unix())
			assert.Equal(t, out[1].Frozen, false)
		})
	})
}

//nolint:dupl
func TestUserNotLoginAutoDisabled(t *testing.T) {
	Convey("用户长时间未登录自动禁用 处理", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		reservedName := mock.NewMockLogicsReservedName(ctrl)
		eacpLog := mock.NewMockDrivenEacpLog(ctrl)
		msgBroker := mock.NewMockDrivenMessageBroker(ctrl)

		trace := mock.NewMockTraceClient(ctrl)
		ctx := context.Background()

		trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
		trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
		trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

		u := &user{
			userDB:        userDB,
			reservedName:  reservedName,
			logger:        common.NewLogger(),
			trace:         trace,
			eacpLog:       eacpLog,
			messageBroker: msgBroker,
		}

		msg := make(map[string]interface{})
		msg["id"] = strID
		msg["display_name"] = strID2
		msg["login_name"] = strID1

		Convey("messageBroker UserStatusChanged-报错", func() {
			msgBroker.EXPECT().UserStatusChanged(gomock.Any(), gomock.Any()).Return(stdErr.New(strTest))
			err := u.userNotLoginAutoDisabled(msg)
			assert.Equal(t, err, stdErr.New(strTest))
		})

		Convey("eacp  OpUserNotLoginDisabled- 报错", func() {
			msgBroker.EXPECT().UserStatusChanged(gomock.Any(), gomock.Any()).Return(nil)
			eacpLog.EXPECT().OpUserNotLoginDisabled(gomock.Any(), gomock.Any()).Return(stdErr.New(strTest))
			err := u.userNotLoginAutoDisabled(msg)
			assert.Equal(t, err, stdErr.New(strTest))
		})

		Convey("success", func() {
			msgBroker.EXPECT().UserStatusChanged(gomock.Any(), gomock.Any()).Return(nil)
			eacpLog.EXPECT().OpUserNotLoginDisabled(gomock.Any(), gomock.Any()).Return(nil)
			err := u.userNotLoginAutoDisabled(msg)
			assert.Equal(t, err, nil)
		})
	})
}

//nolint:dupl
func TestUserExpiredAutoDisabled(t *testing.T) {
	Convey("用户过期自动禁用 处理", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		reservedName := mock.NewMockLogicsReservedName(ctrl)
		eacpLog := mock.NewMockDrivenEacpLog(ctrl)
		msgBroker := mock.NewMockDrivenMessageBroker(ctrl)

		trace := mock.NewMockTraceClient(ctrl)
		ctx := context.Background()

		trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
		trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
		trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

		u := &user{
			userDB:        userDB,
			reservedName:  reservedName,
			logger:        common.NewLogger(),
			trace:         trace,
			eacpLog:       eacpLog,
			messageBroker: msgBroker,
		}

		msg := make(map[string]interface{})
		msg["id"] = strID
		msg["display_name"] = strID2
		msg["login_name"] = strID1

		Convey("messageBroker UserStatusChanged-报错", func() {
			msgBroker.EXPECT().UserStatusChanged(gomock.Any(), gomock.Any()).Return(stdErr.New(strTest))
			err := u.userExpiredAutoDisabled(msg)
			assert.Equal(t, err, stdErr.New(strTest))
		})

		Convey("eacp  OpUserExpiredDisabled- 报错", func() {
			msgBroker.EXPECT().UserStatusChanged(gomock.Any(), gomock.Any()).Return(nil)
			eacpLog.EXPECT().OpUserExpiredDisabled(gomock.Any(), gomock.Any()).Return(stdErr.New(strTest))
			err := u.userExpiredAutoDisabled(msg)
			assert.Equal(t, err, stdErr.New(strTest))
		})

		Convey("success", func() {
			msgBroker.EXPECT().UserStatusChanged(gomock.Any(), gomock.Any()).Return(nil)
			eacpLog.EXPECT().OpUserExpiredDisabled(gomock.Any(), gomock.Any()).Return(nil)
			err := u.userExpiredAutoDisabled(msg)
			assert.Equal(t, err, nil)
		})
	})
}

func TestUserExpiredAutoDisable(t *testing.T) {
	Convey("用户过期自动禁用", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		reservedName := mock.NewMockLogicsReservedName(ctrl)
		eacpLog := mock.NewMockDrivenEacpLog(ctrl)
		msgBroker := mock.NewMockDrivenMessageBroker(ctrl)
		ob := mock.NewMockLogicsOutbox(ctrl)

		trace := mock.NewMockTraceClient(ctrl)
		ctx := context.Background()

		trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
		trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
		trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		u := &user{
			userDB:        userDB,
			reservedName:  reservedName,
			logger:        common.NewLogger(),
			trace:         trace,
			eacpLog:       eacpLog,
			messageBroker: msgBroker,
			pool:          db,
			ob:            ob,
		}

		userInfos := []interfaces.UserDBInfo{
			{
				ID:      strID,
				Name:    strID2,
				Account: strID1,
			},
		}

		Convey("begin-报错", func() {
			mock.ExpectBegin().WillReturnError(stdErr.New(strTest))
			err := u.userExpiredAutoDisable(ctx)
			assert.Equal(t, err, stdErr.New(strTest))
		})

		Convey("GetLock-报错", func() {
			mock.ExpectBegin()
			userDB.EXPECT().GetLock(gomock.Any(), gomock.Any(), gomock.Any()).Return(stdErr.New(strTest))
			mock.ExpectRollback()
			err := u.userExpiredAutoDisable(ctx)
			assert.Equal(t, err, stdErr.New(strTest))
		})

		Convey("GetExpiredNeedDisableUserInfos-报错", func() {
			mock.ExpectBegin()
			userDB.EXPECT().GetLock(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			userDB.EXPECT().GetExpiredNeedDisableUserInfos(gomock.Any(), gomock.Any()).Return(nil, stdErr.New(strTest))
			mock.ExpectRollback()
			err := u.userExpiredAutoDisable(ctx)
			assert.Equal(t, err, stdErr.New(strTest))
		})

		Convey("SetAutoDisableStatus-报错", func() {
			mock.ExpectBegin()
			userDB.EXPECT().GetLock(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			userDB.EXPECT().GetExpiredNeedDisableUserInfos(gomock.Any(), gomock.Any()).Return(nil, nil)
			userDB.EXPECT().SetAutoDisableStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(stdErr.New(strTest))
			mock.ExpectRollback()
			err := u.userExpiredAutoDisable(ctx)
			assert.Equal(t, err, stdErr.New(strTest))
		})

		Convey("AddOutboxInfo-报错", func() {
			mock.ExpectBegin()
			userDB.EXPECT().GetLock(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			userDB.EXPECT().GetExpiredNeedDisableUserInfos(gomock.Any(), gomock.Any()).Return(userInfos, nil)
			userDB.EXPECT().SetAutoDisableStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			ob.EXPECT().AddOutboxInfo(gomock.Any(), gomock.Any(), gomock.Any()).Return(stdErr.New(strTest))
			mock.ExpectRollback()
			err := u.userExpiredAutoDisable(ctx)
			assert.Equal(t, err, stdErr.New(strTest))
		})

		Convey("success", func() {
			mock.ExpectBegin()
			userDB.EXPECT().GetLock(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			userDB.EXPECT().GetExpiredNeedDisableUserInfos(gomock.Any(), gomock.Any()).Return(userInfos, nil)
			userDB.EXPECT().SetAutoDisableStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			ob.EXPECT().AddOutboxInfo(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			mock.ExpectCommit()
			ob.EXPECT().NotifyPushOutboxThread().AnyTimes()
			err := u.userExpiredAutoDisable(ctx)
			assert.Equal(t, err, nil)
		})
	})
}

func TestUserNotLoginAutoDisable(t *testing.T) {
	Convey("用户长时间未登录自动禁用", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userDB := mock.NewMockDBUser(ctrl)
		reservedName := mock.NewMockLogicsReservedName(ctrl)
		eacpLog := mock.NewMockDrivenEacpLog(ctrl)
		msgBroker := mock.NewMockDrivenMessageBroker(ctrl)
		ob := mock.NewMockLogicsOutbox(ctrl)
		conf := mock.NewMockLogicsConfig(ctrl)

		trace := mock.NewMockTraceClient(ctrl)
		ctx := context.Background()

		trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
		trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
		trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		u := &user{
			userDB:        userDB,
			reservedName:  reservedName,
			logger:        common.NewLogger(),
			trace:         trace,
			eacpLog:       eacpLog,
			messageBroker: msgBroker,
			pool:          db,
			ob:            ob,
			config:        conf,
		}

		userInfos := []interfaces.UserDBInfo{
			{
				ID:      strID,
				Name:    strID2,
				Account: strID1,
			},
		}

		Convey("GetConfig-报错", func() {
			conf.EXPECT().GetConfig(gomock.Any()).Return(interfaces.Config{AutoDisableEnabled: true}, stdErr.New(strTest))
			err := u.userNotLoginAutoDisable(ctx)
			assert.Equal(t, err, stdErr.New(strTest))
		})

		Convey("begin-报错", func() {
			conf.EXPECT().GetConfig(gomock.Any()).Return(interfaces.Config{AutoDisableEnabled: true}, nil)
			mock.ExpectBegin().WillReturnError(stdErr.New(strTest))
			err := u.userNotLoginAutoDisable(ctx)
			assert.Equal(t, err, stdErr.New(strTest))
		})

		Convey("GetLock-报错", func() {
			conf.EXPECT().GetConfig(gomock.Any()).Return(interfaces.Config{AutoDisableEnabled: true}, nil)
			mock.ExpectBegin()
			userDB.EXPECT().GetLock(gomock.Any(), gomock.Any(), gomock.Any()).Return(stdErr.New(strTest))
			mock.ExpectRollback()
			err := u.userNotLoginAutoDisable(ctx)
			assert.Equal(t, err, stdErr.New(strTest))
		})

		Convey("GetNotLoginNeedDisableUserInfos-报错", func() {
			conf.EXPECT().GetConfig(gomock.Any()).Return(interfaces.Config{AutoDisableEnabled: true}, nil)
			mock.ExpectBegin()
			userDB.EXPECT().GetLock(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			userDB.EXPECT().GetNotLoginNeedDisableUserInfos(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, stdErr.New(strTest))
			mock.ExpectRollback()
			err := u.userNotLoginAutoDisable(ctx)
			assert.Equal(t, err, stdErr.New(strTest))
		})

		Convey("SetAutoDisableStatus-报错", func() {
			conf.EXPECT().GetConfig(gomock.Any()).Return(interfaces.Config{AutoDisableEnabled: true}, nil)
			mock.ExpectBegin()
			userDB.EXPECT().GetLock(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			userDB.EXPECT().GetNotLoginNeedDisableUserInfos(gomock.Any(), gomock.Any(), gomock.Any()).Return(userInfos, nil)
			userDB.EXPECT().SetAutoDisableStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(stdErr.New(strTest))
			mock.ExpectRollback()
			err := u.userNotLoginAutoDisable(ctx)
			assert.Equal(t, err, stdErr.New(strTest))
		})

		Convey("AddOutboxInfo-报错", func() {
			conf.EXPECT().GetConfig(gomock.Any()).Return(interfaces.Config{AutoDisableEnabled: true}, nil)
			mock.ExpectBegin()
			userDB.EXPECT().GetLock(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			userDB.EXPECT().GetNotLoginNeedDisableUserInfos(gomock.Any(), gomock.Any(), gomock.Any()).Return(userInfos, nil)
			userDB.EXPECT().SetAutoDisableStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			ob.EXPECT().AddOutboxInfo(gomock.Any(), gomock.Any(), gomock.Any()).Return(stdErr.New(strTest))
			mock.ExpectRollback()
			err := u.userNotLoginAutoDisable(ctx)
			assert.Equal(t, err, stdErr.New(strTest))
		})

		Convey("success", func() {
			conf.EXPECT().GetConfig(gomock.Any()).Return(interfaces.Config{AutoDisableEnabled: true}, nil)
			mock.ExpectBegin()
			userDB.EXPECT().GetLock(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			userDB.EXPECT().GetNotLoginNeedDisableUserInfos(gomock.Any(), gomock.Any(), gomock.Any()).Return(userInfos, nil)
			userDB.EXPECT().SetAutoDisableStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			ob.EXPECT().AddOutboxInfo(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			mock.ExpectCommit()
			ob.EXPECT().NotifyPushOutboxThread()
			err := u.userNotLoginAutoDisable(ctx)
			assert.Equal(t, err, nil)
		})
	})
}
