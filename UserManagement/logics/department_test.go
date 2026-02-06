package logics

import (
	"context"
	"testing"

	"github.com/kweaver-ai/go-lib/rest"
	"github.com/kweaver-ai/proton-rds-sdk-go/sqlx"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/assert"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"

	"UserManagement/common"
	"UserManagement/errors"
	"UserManagement/interfaces"
	"UserManagement/interfaces/mock"
)

const (
	strID1 = "xxxx"
)

func newDepartment(userDB interfaces.DBUser, db interfaces.DBDepartment, gm interfaces.DBGroupMember, role interfaces.LogicsRole) *department {
	return &department{
		db:            db,
		groupMemberDB: gm,
		userDB:        userDB,
		role:          role,
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
	}
}

func TestConvertDepartmentName(t *testing.T) {
	Convey("ConvertDepartmentName, db is available", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		departmentDB := mock.NewMockDBDepartment(ctrl)
		gmDB := mock.NewMockDBGroupMember(ctrl)
		user := mock.NewMockDBUser(ctrl)
		departmentLogics := newDepartment(user, departmentDB, gmDB, nil)
		deptIDs := make([]string, 0)
		visitor := interfaces.Visitor{}

		Convey("user id is empty", func() {
			outDeptIDs, err := departmentLogics.ConvertDepartmentName(&visitor, deptIDs, true)
			assert.Equal(t, len(outDeptIDs), 0)
			assert.Equal(t, err, nil)
		})

		deptIDs = append(deptIDs, "user_id")
		Convey("DB GetDepartmentName error", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			departmentDB.EXPECT().GetDepartmentName(gomock.Any()).AnyTimes().Return(nil, nil, testErr)
			outDeptIDs, err := departmentLogics.ConvertDepartmentName(&visitor, deptIDs, true)
			assert.Equal(t, outDeptIDs, nil)
			assert.Equal(t, err, testErr)
		})

		Convey("department is not exist", func() {
			testErr := rest.NewHTTPErrorV2(errors.DepartmentNotFound,
				departmentLogics.i18n.Load(i18nIDObjectsInDepartNotFound, visitor.Language),
				rest.SetDetail(map[string]interface{}{"ids": []string{"user_id"}}),
				rest.SetCodeStr(errors.StrBadRequestDepartmentNotFound))
			departmentDB.EXPECT().GetDepartmentName(gomock.Any()).AnyTimes().Return(nil, nil, nil)
			outDeptIDs, err := departmentLogics.ConvertDepartmentName(&visitor, deptIDs, true)
			assert.Equal(t, outDeptIDs, nil)
			assert.Equal(t, err, testErr)
		})

		Convey("success", func() {
			testNameInfo := make(map[string]interfaces.NameInfo)
			testLogics := interfaces.NameInfo{ID: "user_id", Name: "user_name"}
			testNameInfo[testLogics.ID] = testLogics
			testInfoMap := make([]interfaces.NameInfo, 0)
			testInfoMap = append(testInfoMap, testLogics)
			departmentDB.EXPECT().GetDepartmentName(gomock.Any()).AnyTimes().Return(testInfoMap, []string{testLogics.Name}, nil)
			outDeptIDs, err := departmentLogics.ConvertDepartmentName(&visitor, deptIDs, true)
			assert.Equal(t, len(outDeptIDs), 1)
			assert.Equal(t, outDeptIDs[0], testLogics)
			assert.Equal(t, err, nil)
		})

		Convey("success1", func() {
			testLogics := interfaces.NameInfo{ID: "user_id", Name: "user_name"}
			testInfoMap := make([]interfaces.NameInfo, 0)
			testInfoMap = append(testInfoMap, testLogics)
			departmentDB.EXPECT().GetDepartmentName(gomock.Any()).AnyTimes().Return(testInfoMap, []string{testLogics.Name}, nil)
			outDeptIDs, err := departmentLogics.ConvertDepartmentName(&visitor, []string{testLogics.ID, strID2}, false)
			assert.Equal(t, len(outDeptIDs), 1)
			assert.Equal(t, outDeptIDs[0], testLogics)
			assert.Equal(t, err, nil)
		})
	})
}

func TestGetDepartEmails(t *testing.T) {
	Convey("GetDepartEmails, db is available", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		departmentDB := mock.NewMockDBDepartment(ctrl)
		gmDB := mock.NewMockDBGroupMember(ctrl)
		user := mock.NewMockDBUser(ctrl)
		departmentLogics := newDepartment(user, departmentDB, gmDB, nil)
		deptIDs := make([]string, 0)
		visitor := interfaces.Visitor{}

		Convey("user id is empty", func() {
			outDeptIDs, err := departmentLogics.GetDepartEmails(&visitor, deptIDs)
			assert.Equal(t, len(outDeptIDs), 0)
			assert.Equal(t, err, nil)
		})

		deptIDs = append(deptIDs, "user_id")
		Convey("DB GetDepartmentInfo error", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, testErr)
			outDeptIDs, err := departmentLogics.GetDepartEmails(&visitor, deptIDs)
			assert.Equal(t, outDeptIDs, nil)
			assert.Equal(t, err, testErr)
		})

		Convey("department is not exist", func() {
			testErr := rest.NewHTTPErrorV2(errors.DepartmentNotFound,
				departmentLogics.i18n.Load(i18nIDObjectsInDepartNotFound, visitor.Language),
				rest.SetDetail(map[string]interface{}{"ids": []string{"user_id"}}),
				rest.SetCodeStr(errors.StrBadRequestDepartmentNotFound))
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
			outDeptIDs, err := departmentLogics.GetDepartEmails(&visitor, deptIDs)
			assert.Equal(t, outDeptIDs, nil)
			assert.Equal(t, err, testErr)
		})

		Convey("success", func() {
			testEmailInfo := make(map[string]interfaces.DepartmentDBInfo)
			testLogics := interfaces.DepartmentDBInfo{ID: "user_id", Email: "user_name"}
			testEmailInfo[testLogics.ID] = testLogics
			testInfoMap := make([]interfaces.DepartmentDBInfo, 0)
			testInfoMap = append(testInfoMap, testLogics)
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(testInfoMap, nil)
			outDeptIDs, err := departmentLogics.GetDepartEmails(&visitor, deptIDs)
			assert.Equal(t, len(outDeptIDs), 1)
			assert.Equal(t, outDeptIDs[0], interfaces.EmailInfo{ID: testLogics.ID, Email: testLogics.Email})
			assert.Equal(t, err, nil)
		})
	})
}

func TestGetAllChildDeparmentIDs(t *testing.T) {
	Convey("GetAllChildDeparmentIDs, db is available", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		departmentDB := mock.NewMockDBDepartment(ctrl)
		gmDB := mock.NewMockDBGroupMember(ctrl)
		user := mock.NewMockDBUser(ctrl)
		departmentLogics := newDepartment(user, departmentDB, gmDB, nil)
		deptIDs := make([]string, 0)

		Convey("deptIDs id is empty", func() {
			outDeptIDs, err := departmentLogics.GetAllChildDeparmentIDs(deptIDs)
			assert.Equal(t, len(outDeptIDs), 0)
			assert.Equal(t, err, nil)
		})

		deptIDs = append(deptIDs, "user_id")
		Convey("success", func() {
			tempData := []string{
				0: "xxxxx",
				1: "xxxxx",
			}
			departmentDB.EXPECT().GetChildDepartmentIDs(gomock.Any()).AnyTimes().Return(tempData, nil, nil).Times(1)
			departmentDB.EXPECT().GetChildDepartmentIDs(gomock.Any()).AnyTimes().Return(nil, nil, nil)
			outDeptIDs, err := departmentLogics.GetAllChildDeparmentIDs(deptIDs)
			assert.Equal(t, len(outDeptIDs), 1)
			assert.Equal(t, err, nil)
			assert.Equal(t, outDeptIDs[0], "xxxxx")
		})
	})
}

func TestGetManagersOfDepartment(t *testing.T) {
	Convey("GetOrgManagerIDsOfDepartment, db is available", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		departmentDB := mock.NewMockDBDepartment(ctrl)
		gmDB := mock.NewMockDBGroupMember(ctrl)
		user := mock.NewMockDBUser(ctrl)
		departmentLogics := newDepartment(user, departmentDB, gmDB, nil)
		deptIDs := make([]string, 0)

		Convey("deptIDs id is empty", func() {
			infoList, err := departmentLogics.GetDepartsInfo(deptIDs, interfaces.DepartInfoScope{BManagers: true}, true)
			assert.Equal(t, len(infoList), 0)
			assert.Equal(t, err, nil)
		})

		deptIDs = append(deptIDs, "user_id")
		Convey("success", func() {
			tempData := []interfaces.DepartmentManagerInfo{
				{
					DepartmentID: "user_id",
					Managers: []interfaces.NameInfo{
						{
							ID:   "xx1",
							Name: "name1",
						},
					},
				},
				{
					DepartmentID: "bb",
					Managers: []interfaces.NameInfo{
						{
							ID:   "xx2",
							Name: "name2",
						},
					},
				},
			}
			testNameInfo := make(map[string]interfaces.DepartmentDBInfo)
			testLogics := interfaces.DepartmentDBInfo{ID: "user_id", Name: "user_name", ManagerID: "xx1", ThirdID: "third_id1"}
			testNameInfo[testLogics.ID] = testLogics
			testInfoMap := make([]interfaces.DepartmentDBInfo, 0)
			testInfoMap = append(testInfoMap, testLogics)
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(testInfoMap, nil)
			departmentDB.EXPECT().GetManagersOfDepartment(gomock.Any()).AnyTimes().Return(tempData, nil)
			infoList, err := departmentLogics.GetDepartsInfo(deptIDs, interfaces.DepartInfoScope{BManagers: true, BManager: true, BThirdID: true}, true)
			assert.Equal(t, len(infoList), 1)
			assert.Equal(t, len(infoList[0].Managers), 1)
			assert.Equal(t, err, nil)
			assert.Equal(t, infoList[0].Manager.ID, "xx1")
			assert.Equal(t, infoList[0].ThirdID, "third_id1")
		})
	})
}

func TestGetAccessorIDsOfDepartment(t *testing.T) {
	Convey("GetAccessorIDsOfDepartment, db is available", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		departmentDB := mock.NewMockDBDepartment(ctrl)
		gmDB := mock.NewMockDBGroupMember(ctrl)
		user := mock.NewMockDBUser(ctrl)
		departmentLogics := newDepartment(user, departmentDB, gmDB, nil)

		Convey("GetDepartmentInfo error", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, testErr)

			_, err := departmentLogics.GetAccessorIDsOfDepartment(strID1)
			assert.Equal(t, err, testErr)
		})

		var strList []interfaces.DepartmentDBInfo
		Convey("department not exsit", func() {
			testErr1 := rest.NewHTTPErrorV2(errors.NotFound, "department does not exist")
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(strList, nil)

			_, err := departmentLogics.GetAccessorIDsOfDepartment(strID1)
			assert.Equal(t, err, testErr1)
		})

		info := interfaces.DepartmentDBInfo{
			Path: "depart1/depart2/depart3",
		}
		strList = append(strList, info)
		Convey("GetMembersBelongGroupIDs error", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(strList, nil)
			gmDB.EXPECT().GetMembersBelongGroupIDs(gomock.Any()).AnyTimes().Return(nil, nil, testErr)
			_, err := departmentLogics.GetAccessorIDsOfDepartment(strID1)
			assert.Equal(t, err, testErr)
		})

		Convey("success", func() {
			groupIDs := []string{
				"kkkkk",
				"tttttt",
			}
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(strList, nil)
			gmDB.EXPECT().GetMembersBelongGroupIDs(gomock.Any()).AnyTimes().Return(groupIDs, nil, nil)
			outDeptIDs, err := departmentLogics.GetAccessorIDsOfDepartment(strID1)

			assert.Equal(t, len(outDeptIDs), 5)
			assert.Equal(t, err, nil)
			assert.Equal(t, outDeptIDs[0], "depart1")
			assert.Equal(t, outDeptIDs[1], "depart2")
			assert.Equal(t, outDeptIDs[2], "depart3")
			assert.Equal(t, outDeptIDs[3], "kkkkk")
			assert.Equal(t, outDeptIDs[4], "tttttt")
		})
	})
}

func TestGetDepartMemberIDs(t *testing.T) {
	Convey("GetDepartMemberIDs, db is available", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		departmentDB := mock.NewMockDBDepartment(ctrl)
		gmDB := mock.NewMockDBGroupMember(ctrl)
		user := mock.NewMockDBUser(ctrl)
		departmentLogics := newDepartment(user, departmentDB, gmDB, nil)

		depID := strID1
		Convey("GetDepartmentName error", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			departmentDB.EXPECT().GetDepartmentName(gomock.Any()).AnyTimes().Return(nil, nil, testErr)

			_, err := departmentLogics.GetDepartMemberIDs(depID)
			assert.Equal(t, err, testErr)
		})

		var strList []string
		Convey("department not exsit", func() {
			testErr1 := rest.NewHTTPErrorV2(errors.NotFound, "department does not exist")
			departmentDB.EXPECT().GetDepartmentName(gomock.Any()).AnyTimes().Return(nil, strList, nil)

			_, err := departmentLogics.GetDepartMemberIDs(depID)
			assert.Equal(t, err, testErr1)
		})

		strList = append(strList, depID)
		Convey("GetChildDepartmentIDs error", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			departmentDB.EXPECT().GetDepartmentName(gomock.Any()).AnyTimes().Return(nil, strList, nil)
			departmentDB.EXPECT().GetChildDepartmentIDs(gomock.Any()).AnyTimes().Return(nil, nil, testErr)

			_, err := departmentLogics.GetDepartMemberIDs(depID)
			assert.Equal(t, err, testErr)
		})

		Convey("GetChildUserIDs error", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			departmentDB.EXPECT().GetDepartmentName(gomock.Any()).AnyTimes().Return(nil, strList, nil)
			departmentDB.EXPECT().GetChildDepartmentIDs(gomock.Any()).AnyTimes().Return(nil, nil, nil)
			departmentDB.EXPECT().GetChildUserIDs(gomock.Any()).AnyTimes().Return(nil, nil, testErr)
			_, err := departmentLogics.GetDepartMemberIDs(depID)
			assert.Equal(t, err, testErr)
		})

		Convey("success", func() {
			depIDs := []string{
				"xxxxx",
				"zzzzzz",
			}
			userIDs := []string{
				"kkkkk",
				"tttttt",
			}
			departmentDB.EXPECT().GetDepartmentName(gomock.Any()).AnyTimes().Return(nil, strList, nil)
			departmentDB.EXPECT().GetChildDepartmentIDs(gomock.Any()).AnyTimes().Return(depIDs, nil, nil)
			departmentDB.EXPECT().GetChildUserIDs(gomock.Any()).AnyTimes().Return(userIDs, nil, nil)
			outInfo, err := departmentLogics.GetDepartMemberIDs(depID)
			assert.Equal(t, len(outInfo.DepartIDs), 2)
			assert.Equal(t, len(outInfo.UserIDs), 2)
			assert.Equal(t, err, nil)
			assert.Equal(t, outInfo.DepartIDs[0], "xxxxx")
			assert.Equal(t, outInfo.DepartIDs[1], "zzzzzz")
			assert.Equal(t, outInfo.UserIDs[0], "kkkkk")
			assert.Equal(t, outInfo.UserIDs[1], "tttttt")
		})
	})
}

func TestGetAllDepartUserIDs(t *testing.T) {
	Convey("GetAllDepartUserIDs, db is available", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		departmentDB := mock.NewMockDBDepartment(ctrl)
		gmDB := mock.NewMockDBGroupMember(ctrl)
		user := mock.NewMockDBUser(ctrl)
		departmentLogics := newDepartment(user, departmentDB, gmDB, nil)

		depID := strID1
		Convey("GetDepartmentName error", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, testErr)

			_, err := departmentLogics.GetAllDepartUserIDs(depID, true)
			assert.Equal(t, err, testErr)
		})

		var strList []interfaces.DepartmentDBInfo
		Convey("department not exsit", func() {
			testErr1 := rest.NewHTTPErrorV2(errors.NotFound, "department does not exist")
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(strList, nil)

			_, err := departmentLogics.GetAllDepartUserIDs(depID, true)
			assert.Equal(t, err, testErr1)
		})

		strList = append(strList, interfaces.DepartmentDBInfo{})
		Convey("GetAllSubUserIDsByDepartPath error", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(strList, nil)
			departmentDB.EXPECT().GetAllSubUserIDsByDepartPath(gomock.Any()).AnyTimes().Return(nil, testErr)

			_, err := departmentLogics.GetAllDepartUserIDs(depID, true)
			assert.Equal(t, err, testErr)
		})

		Convey("show unbale user success", func() {
			userIDs := []string{
				"xxxxx",
				"zzzzzz",
			}
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(strList, nil)
			departmentDB.EXPECT().GetAllSubUserIDsByDepartPath(gomock.Any()).AnyTimes().Return(userIDs, nil)

			outInfo, err := departmentLogics.GetAllDepartUserIDs(depID, true)
			assert.Equal(t, len(outInfo), 2)
			assert.Equal(t, err, nil)
			assert.Equal(t, outInfo[0], "xxxxx")
			assert.Equal(t, outInfo[1], "zzzzzz")
		})

		Convey("GetUserDBInfo error", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			userIDs := []string{
				"xxxxx",
				"zzzzzz",
			}
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(strList, nil)
			departmentDB.EXPECT().GetAllSubUserIDsByDepartPath(gomock.Any()).AnyTimes().Return(userIDs, nil)
			user.EXPECT().GetUserDBInfo(gomock.Any()).AnyTimes().Return(nil, testErr)

			_, err := departmentLogics.GetAllDepartUserIDs(depID, false)
			assert.Equal(t, err, testErr)
		})

		Convey("no show unbale user success", func() {
			userIDs := []string{
				"xxxxx",
				"zzzzzz",
			}

			user1 := interfaces.UserDBInfo{
				ID:                userIDs[0],
				DisableStatus:     interfaces.Enabled,
				AutoDisableStatus: interfaces.ADisabled,
			}
			user2 := interfaces.UserDBInfo{
				ID:                userIDs[1],
				DisableStatus:     interfaces.Enabled,
				AutoDisableStatus: interfaces.AEnabled,
			}
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(strList, nil)
			departmentDB.EXPECT().GetAllSubUserIDsByDepartPath(gomock.Any()).AnyTimes().Return(userIDs, nil)
			user.EXPECT().GetUserDBInfo(userIDs).AnyTimes().Return([]interfaces.UserDBInfo{user1, user2}, nil)

			out, err := departmentLogics.GetAllDepartUserIDs(depID, false)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(out), 1)
			assert.Equal(t, out[0], user2.ID)
		})
	})
}

func TestGetDepartMemberInfo(t *testing.T) {
	Convey("获取部门成员信息-逻辑层", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		departmentDB := mock.NewMockDBDepartment(ctrl)
		gmDB := mock.NewMockDBGroupMember(ctrl)
		userDB := mock.NewMockDBUser(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		departmentLogics := newDepartment(userDB, departmentDB, gmDB, role)

		var temp interfaces.Role = strID1
		depID := string(temp)
		var info interfaces.OrgShowPageInfo
		var visitor interfaces.Visitor
		Convey("获取用户角色失败-报错", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(nil, testErr)

			_, _, _, _, err := departmentLogics.GetDepartMemberInfo(&visitor, depID, info)
			assert.Equal(t, err, testErr)
		})

		roleInfo := make(map[string]map[interfaces.Role]bool)
		mapRoles := make(map[interfaces.Role]bool)
		mapRoles[interfaces.SystemRoleAuditAdmin] = true
		roleInfo[visitor.ID] = mapRoles
		Convey("用户没有指定角色-报错", func() {
			info.Role = temp
			testErr := rest.NewHTTPError("this user do not has this role", rest.BadRequest, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfo, nil)

			_, _, _, _, err := departmentLogics.GetDepartMemberInfo(&visitor, depID, info)
			assert.Equal(t, err, testErr)
		})
	})
}

//nolint:funlen
func TestGetSupervisoryRootDeps(t *testing.T) {
	Convey("获取用户根部门信息-逻辑层", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		departmentDB := mock.NewMockDBDepartment(ctrl)
		gmDB := mock.NewMockDBGroupMember(ctrl)
		userDB := mock.NewMockDBUser(ctrl)
		org := mock.NewMockLogicsOrgPerm(ctrl)

		departs := department{
			db:            departmentDB,
			userDB:        userDB,
			groupMemberDB: gmDB,
			orgPerm:       org,
		}

		var info interfaces.OrgShowPageInfo
		Convey("type未传department，则返回空信息", func() {
			info.Role = interfaces.SystemRoleOrgAudit
			out, num, err := departs.getSupervisoryRootDeps("", info)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(out), 0)
			assert.Equal(t, num, 0)
		})

		info.BShowDeparts = true
		Convey("用户为组织审计员，获取审计范围失败-报错", func() {
			info.Role = interfaces.SystemRoleOrgAudit
			testErr := rest.NewHTTPError("error", 503000000, nil)
			userDB.EXPECT().GetOrgAduitDepartInfo(gomock.Any()).AnyTimes().Return(nil, testErr)

			_, _, err := departs.getSupervisoryRootDeps("", info)
			assert.Equal(t, err, testErr)
		})

		Convey("用户为组织管理员，获取管理范围失败-报错", func() {
			info.Role = interfaces.SystemRoleOrgManager
			testErr := rest.NewHTTPError("error", 503000000, nil)
			userDB.EXPECT().GetOrgManagerDepartInfo(gomock.Any()).AnyTimes().Return(nil, testErr)

			_, _, err := departs.getSupervisoryRootDeps("", info)
			assert.Equal(t, err, testErr)
		})

		Convey("用户为超级管理员，获取根部门失败-报错", func() {
			info.Role = interfaces.SystemRoleSuperAdmin
			testErr := rest.NewHTTPError("error", 503000000, nil)
			departmentDB.EXPECT().GetRootDeps(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, testErr)

			_, _, err := departs.getSupervisoryRootDeps("", info)
			assert.Equal(t, err, testErr)
		})

		temp := interfaces.DepartmentDBInfo{
			ID:     "ID1",
			Name:   "Name1",
			IsRoot: 1,
		}
		temp1 := interfaces.DepartmentDBInfo{
			ID:     "ID2",
			Name:   "Name2",
			IsRoot: 0,
		}
		allDeps := []interfaces.DepartmentDBInfo{temp}
		tempDeps := []interfaces.DepartmentDBInfo{temp, temp1}

		Convey("用户为超级管理员，获取根部门失败-报错1", func() {
			info.Role = interfaces.SystemRoleSuperAdmin
			testErr := rest.NewHTTPError("error", 503000000, nil)
			departmentDB.EXPECT().GetRootDeps(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(tempDeps, nil)
			departmentDB.EXPECT().GetRootDeps(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(nil, testErr)

			_, _, err := departs.getSupervisoryRootDeps("", info)
			assert.Equal(t, err, testErr)
		})

		Convey("用户为超级管理员，获取根部门成功", func() {
			info.Role = interfaces.SystemRoleSuperAdmin
			departmentDB.EXPECT().GetRootDeps(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(allDeps, nil)
			departmentDB.EXPECT().GetRootDeps(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(tempDeps, nil)

			test := interfaces.DepartInfo{
				ID:     temp.ID,
				Name:   temp.Name,
				IsRoot: true,
				Manager: interfaces.NameInfo{
					ID:   "",
					Name: "",
				},
				Code:    "",
				Enabled: false,
				Remark:  "",
				Email:   "",
			}
			out, num, err := departs.getSupervisoryRootDeps("", info)
			assert.Equal(t, err, nil)
			assert.Equal(t, num, 2)
			assert.Equal(t, len(out), 1)
			assert.Equal(t, out[0].Manager, test.Manager)
			assert.Equal(t, out[0].ID, test.ID)
			assert.Equal(t, out[0].Name, test.Name)
			assert.Equal(t, out[0].IsRoot, test.IsRoot)
		})

		Convey("用户为组织管理员，筛选最上层部门失败-报错", func() {
			info.Role = interfaces.SystemRoleOrgManager
			depIDs := []string{strID1}
			testErr := rest.NewHTTPError("error", 503000000, nil)
			userDB.EXPECT().GetOrgManagerDepartInfo(gomock.Any()).AnyTimes().Return(depIDs, nil)
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, testErr)

			_, _, err := departs.getSupervisoryRootDeps("", info)
			assert.Equal(t, err, testErr)
		})

		testErr := rest.NewHTTPError("error", 503000000, nil)
		Convey("用户为组织管理员，用户信息排序-报错", func() {
			info.Role = interfaces.SystemRoleOrgManager
			userDB.EXPECT().GetOrgManagerDepartInfo(gomock.Any()).AnyTimes().Return(nil, nil)
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, testErr)

			_, _, err := departs.getSupervisoryRootDeps("", info)
			assert.Equal(t, err, testErr)
		})

		data1 := interfaces.DepartmentDBInfo{
			ID:   strID,
			Name: "name11",
		}
		testDeps := []interfaces.DepartmentDBInfo{data1}
		Convey("用户为组织管理员，获取信息成功", func() {
			info.Role = interfaces.SystemRoleOrgManager
			depIDs := []string{strID1, "zzzz"}
			userDB.EXPECT().GetOrgManagerDepartInfo(gomock.Any()).AnyTimes().Return(depIDs, nil)
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(testDeps, nil)
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(testDeps, nil)

			out, num, err := departs.getSupervisoryRootDeps("", info)
			test := interfaces.DepartInfo{
				ID:   data1.ID,
				Name: data1.Name,
			}
			assert.Equal(t, err, nil)
			assert.Equal(t, num, 1)
			assert.Equal(t, len(out), 1)
			assert.Equal(t, test.ID, out[0].ID)
			assert.Equal(t, test.Name, out[0].Name)
		})

		subUsers := make(map[string][]string)
		Convey("用户为超级管理员，显示子部门的子部门和子用户是否存在信息，获取子用户失败", func() {
			info.Role = interfaces.SystemRoleSuperAdmin
			info.BShowSubDepart = true
			info.BShowSubUser = true
			departmentDB.EXPECT().GetRootDeps(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(allDeps, nil)
			departmentDB.EXPECT().GetRootDeps(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(tempDeps, nil)
			departmentDB.EXPECT().GetChildUserIDs(gomock.Any()).AnyTimes().Return(nil, subUsers, testErr)

			_, _, err := departs.getSupervisoryRootDeps("", info)
			assert.Equal(t, err, testErr)
		})

		subDeparts := make(map[string][]string)
		Convey("用户为超级管理员，显示子部门的子部门和子用户是否存在信息，获取子部门失败", func() {
			info.Role = interfaces.SystemRoleSuperAdmin
			info.BShowSubDepart = true
			info.BShowSubUser = true
			departmentDB.EXPECT().GetRootDeps(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(allDeps, nil)
			departmentDB.EXPECT().GetRootDeps(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(tempDeps, nil)
			departmentDB.EXPECT().GetChildUserIDs(gomock.Any()).AnyTimes().Return(nil, subUsers, nil)
			departmentDB.EXPECT().GetChildDepartmentIDs(gomock.Any()).AnyTimes().Return(nil, subDeparts, testErr)

			_, _, err := departs.getSupervisoryRootDeps("", info)
			assert.Equal(t, err, testErr)
		})

		Convey("用户为普通用户，显示子部门的子部门和子用户是否存在信息，获取子部门失败", func() {
			info.Role = interfaces.SystemRoleNormalUser
			info.BShowSubDepart = true
			info.BShowSubUser = true
			org.EXPECT().CheckPerms(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(false, nil)

			out, num, err := departs.getSupervisoryRootDeps("", info)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(out), 0)
			assert.Equal(t, num, 0)
		})

		temp3 := interfaces.DepartmentDBInfo{
			ID:     strID,
			Name:   strName,
			IsRoot: 1,
		}
		temp4 := interfaces.DepartmentDBInfo{
			ID:     strID1,
			Name:   strName1,
			IsRoot: 0,
		}
		temp5 := interfaces.DepartmentDBInfo{
			ID:     strID2,
			Name:   strName2,
			IsRoot: 0,
		}
		allDeps1 := []interfaces.DepartmentDBInfo{temp3, temp4}
		tempDeps1 := []interfaces.DepartmentDBInfo{temp3, temp4, temp5}

		subUsers[strID] = []string{}
		subDeparts[strID] = []string{strID1}
		subUsers[strID1] = []string{strID2}
		subDeparts[strID1] = []string{}
		Convey("用户为超级管理员，显示子部门的子部门和子用户是否存在信息，获取成功", func() {
			info.Role = interfaces.SystemRoleSuperAdmin
			info.BShowSubDepart = true
			info.BShowSubUser = true
			departmentDB.EXPECT().GetRootDeps(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(allDeps1, nil)
			departmentDB.EXPECT().GetRootDeps(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(tempDeps1, nil)
			departmentDB.EXPECT().GetChildUserIDs(gomock.Any()).AnyTimes().Return(nil, subUsers, nil)
			departmentDB.EXPECT().GetChildDepartmentIDs(gomock.Any()).AnyTimes().Return(nil, subDeparts, nil)

			out, num, err := departs.getSupervisoryRootDeps(strID, info)
			assert.Equal(t, err, nil)
			assert.Equal(t, num, 3)
			assert.Equal(t, len(out), 2)

			test1 := interfaces.DepartInfo{
				ID:            temp3.ID,
				Name:          temp3.Name,
				IsRoot:        true,
				BDepartExistd: true,
				BUserExistd:   false,
			}

			test2 := interfaces.DepartInfo{
				ID:            temp4.ID,
				Name:          temp4.Name,
				IsRoot:        false,
				BDepartExistd: false,
				BUserExistd:   true,
			}
			assert.Equal(t, test1.ID, out[0].ID)
			assert.Equal(t, test1.Name, out[0].Name)
			assert.Equal(t, test1.IsRoot, out[0].IsRoot)
			assert.Equal(t, test1.BDepartExistd, out[0].BDepartExistd)
			assert.Equal(t, test1.BUserExistd, out[0].BUserExistd)
			assert.Equal(t, test2.ID, out[1].ID)
			assert.Equal(t, test2.Name, out[1].Name)
			assert.Equal(t, test2.IsRoot, out[1].IsRoot)
			assert.Equal(t, test2.BDepartExistd, out[1].BDepartExistd)
			assert.Equal(t, test2.BUserExistd, out[1].BUserExistd)
		})

		Convey("用户为普通用户，显示子部门的子部门和子用户是否存在信息，获取子部门成功", func() {
			info.Role = interfaces.SystemRoleNormalUser
			info.BShowSubDepart = true
			info.BShowSubUser = true
			org.EXPECT().CheckPerms(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(true, nil)
			departmentDB.EXPECT().GetRootDeps(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(allDeps1, nil)
			departmentDB.EXPECT().GetRootDeps(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(tempDeps1, nil)
			departmentDB.EXPECT().GetChildUserIDs(gomock.Any()).AnyTimes().Return(nil, subUsers, nil)
			departmentDB.EXPECT().GetChildDepartmentIDs(gomock.Any()).AnyTimes().Return(nil, subDeparts, nil)

			out, num, err := departs.getSupervisoryRootDeps(strID, info)
			assert.Equal(t, err, nil)
			assert.Equal(t, num, 3)
			assert.Equal(t, len(out), 2)

			test1 := interfaces.DepartInfo{
				ID:            temp3.ID,
				Name:          temp3.Name,
				IsRoot:        true,
				BDepartExistd: true,
				BUserExistd:   false,
			}

			test2 := interfaces.DepartInfo{
				ID:            temp4.ID,
				Name:          temp4.Name,
				IsRoot:        false,
				BDepartExistd: false,
				BUserExistd:   true,
			}
			assert.Equal(t, test1.ID, out[0].ID)
			assert.Equal(t, test1.Name, out[0].Name)
			assert.Equal(t, test1.IsRoot, out[0].IsRoot)
			assert.Equal(t, test1.BDepartExistd, out[0].BDepartExistd)
			assert.Equal(t, test1.BUserExistd, out[0].BUserExistd)
			assert.Equal(t, test2.ID, out[1].ID)
			assert.Equal(t, test2.Name, out[1].Name)
			assert.Equal(t, test2.IsRoot, out[1].IsRoot)
			assert.Equal(t, test2.BDepartExistd, out[1].BDepartExistd)
			assert.Equal(t, test2.BUserExistd, out[1].BUserExistd)
		})

		Convey("用户为超级管理员，获取管理员信息失败", func() {
			info.Role = interfaces.SystemRoleSuperAdmin
			info.BShowDepartManager = true
			dep1 := interfaces.DepartmentDBInfo{
				ID:   "xxx",
				Name: "yyy",
			}
			dep2 := interfaces.DepartmentDBInfo{
				ID:   "xxx1",
				Name: "yyy1",
			}
			allDeps1 = []interfaces.DepartmentDBInfo{dep1, dep2}

			departmentDB.EXPECT().GetRootDeps(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(allDeps1, nil)
			departmentDB.EXPECT().GetRootDeps(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(allDeps1, nil)
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).Times(1).Return(nil, testErr)

			_, _, err := departs.getSupervisoryRootDeps("", info)
			assert.Equal(t, err, testErr)
		})

		Convey("用户为超级管理员，获取管理员信息成功", func() {
			info.Role = interfaces.SystemRoleSuperAdmin
			info.BShowDepartManager = true
			dep1 := interfaces.DepartmentDBInfo{
				ID:        "xxx",
				Name:      "yyy",
				ManagerID: "xxx1",
			}
			dep2 := interfaces.DepartmentDBInfo{
				ID:        "xxx1",
				Name:      "yyy1",
				ManagerID: "xxx2",
			}
			allDeps1 = []interfaces.DepartmentDBInfo{dep1, dep2}

			manager1 := interfaces.UserDBInfo{
				ID:   "xxx1",
				Name: "yyy1",
			}
			manager2 := interfaces.UserDBInfo{
				ID:   "xxx2",
				Name: "yyy2",
			}
			managerInfos := []interfaces.UserDBInfo{manager1, manager2}
			departmentDB.EXPECT().GetRootDeps(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(allDeps1, nil)
			departmentDB.EXPECT().GetRootDeps(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(allDeps1, nil)
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).Times(1).Return(managerInfos, nil)

			out, num, err := departs.getSupervisoryRootDeps("", info)
			assert.Equal(t, err, nil)
			assert.Equal(t, num, 2)
			assert.Equal(t, len(out), 2)
			assert.Equal(t, out[0].ID, dep1.ID)
			assert.Equal(t, out[0].Name, dep1.Name)
			assert.Equal(t, out[0].Manager.ID, manager1.ID)
			assert.Equal(t, out[0].Manager.Name, manager1.Name)
			assert.Equal(t, len(out[0].ParentDeps), 0)
			assert.Equal(t, out[1].ID, dep2.ID)
			assert.Equal(t, out[1].Name, dep2.Name)
			assert.Equal(t, out[1].Manager.ID, manager2.ID)
			assert.Equal(t, out[1].Manager.Name, manager2.Name)
			assert.Equal(t, len(out[1].ParentDeps), 0)
		})
	})
}

func TestGetDepartMemberInfoByID(t *testing.T) {
	Convey("获取部门子用户和子成员-逻辑层", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		departmentDB := mock.NewMockDBDepartment(ctrl)
		gmDB := mock.NewMockDBGroupMember(ctrl)
		userDB := mock.NewMockDBUser(ctrl)

		departs := department{
			db:            departmentDB,
			userDB:        userDB,
			groupMemberDB: gmDB,
		}
		var info interfaces.OrgShowPageInfo
		testErr := rest.NewHTTPError("error", 503000000, nil)
		Convey("部门获取失败-报错", func() {
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(nil, testErr)

			_, _, _, _, err := departs.getDepartMemberInfoByID("", "", info)
			assert.Equal(t, err, testErr)
		})

		Convey("部门不存在-报错", func() {
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(nil, nil)

			_, _, _, _, err := departs.getDepartMemberInfoByID("", "", info)
			assert.Equal(t, err, rest.NewHTTPErrorV2(errors.NotFound, "department does not exist"))
		})

		tempUser := interfaces.DepartmentDBInfo{ID: "xxx", Name: "zzz"}
		info.Role = "xxxx11"
		info.BShowDeparts = true
		Convey("部门在调用者管辖范围之外-直接返回", func() {
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return([]interfaces.DepartmentDBInfo{tempUser}, nil)

			out, num, outUser, numUser, err := departs.getDepartMemberInfoByID("", "", info)
			assert.Equal(t, err, nil)
			assert.Equal(t, num, 0)
			assert.Equal(t, len(out), 0)
			assert.Equal(t, numUser, 0)
			assert.Equal(t, len(outUser), 0)
		})

		info.Role = interfaces.SystemRoleAuditAdmin
		info.BShowDeparts = true
		Convey("获取部门子部门信息失败", func() {
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return([]interfaces.DepartmentDBInfo{tempUser}, nil)
			departmentDB.EXPECT().GetSubDepartmentInfos(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, testErr)

			_, _, _, _, err := departs.getDepartMemberInfoByID("", "", info)
			assert.Equal(t, err, testErr)
		})

		info.BShowDeparts = false
		info.BShowUsers = true
		Convey("获取部门子用户信息失败", func() {
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return([]interfaces.DepartmentDBInfo{tempUser}, nil)
			departmentDB.EXPECT().GetSubUserInfos(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, testErr)

			_, _, _, _, err := departs.getDepartMemberInfoByID("", "", info)
			assert.Equal(t, err, testErr)
		})

		info.BShowUsers = false
		Convey("不包含子部门/子用户信息，成功", func() {
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return([]interfaces.DepartmentDBInfo{tempUser}, nil)

			_, _, _, _, err := departs.getDepartMemberInfoByID("", "", info)
			assert.Equal(t, err, nil)
		})

		Convey("获取部门和用户数据成功", func() {
			info.BShowUsers = true
			info.BShowDeparts = true

			testDepart1 := interfaces.DepartmentDBInfo{
				ID:   "xxx",
				Name: "yyy",
			}
			testUser1 := interfaces.UserDBInfo{
				ID:   "xxx1",
				Name: "yyy1",
			}
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return([]interfaces.DepartmentDBInfo{testDepart1}, nil)
			departmentDB.EXPECT().GetSubDepartmentInfos(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return([]interfaces.DepartmentDBInfo{testDepart1}, nil)
			departmentDB.EXPECT().GetSubUserInfos(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return([]interfaces.UserDBInfo{testUser1}, nil)

			outDepart, numDepart, outUser, numUser, err := departs.getDepartMemberInfoByID("", "", info)
			assert.Equal(t, err, nil)
			assert.Equal(t, numDepart, 1)
			assert.Equal(t, len(outDepart), 1)
			assert.Equal(t, outDepart[0].ID, testDepart1.ID)
			assert.Equal(t, outDepart[0].Name, testDepart1.Name)
			assert.Equal(t, numUser, 1)
			assert.Equal(t, len(outUser), 1)
			assert.Equal(t, outUser[0].ID, testUser1.ID)
			assert.Equal(t, outUser[0].Name, testUser1.Name)
		})
	})
}

func TestGetSubDepartmentInfo(t *testing.T) {
	Convey("获取部门子部门信息-逻辑层", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		departmentDB := mock.NewMockDBDepartment(ctrl)
		gmDB := mock.NewMockDBGroupMember(ctrl)
		userDB := mock.NewMockDBUser(ctrl)

		departs := department{
			db:            departmentDB,
			userDB:        userDB,
			groupMemberDB: gmDB,
		}

		Convey("获取子部门信息失败-报错", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			departmentDB.EXPECT().GetSubDepartmentInfos(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(nil, testErr)

			_, _, err := departs.getSubDepartmentInfo("", interfaces.OrgShowPageInfo{Offset: 0, Limit: 100}, "")
			assert.Equal(t, err, testErr)
		})

		temp1 := interfaces.DepartmentDBInfo{
			ID:   "xxx",
			Name: "yyy",
		}
		temp2 := interfaces.DepartmentDBInfo{
			ID:   "xxx1",
			Name: "yyy1",
		}
		outDepIDs := []interfaces.DepartmentDBInfo{temp1}
		Convey("获取子部门信息失败2-报错", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			departmentDB.EXPECT().GetSubDepartmentInfos(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(outDepIDs, nil)
			departmentDB.EXPECT().GetSubDepartmentInfos(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(nil, testErr)

			_, _, err := departs.getSubDepartmentInfo("", interfaces.OrgShowPageInfo{Offset: 0, Limit: 100}, "")
			assert.Equal(t, err, testErr)
		})
		outDepIDs1 := []interfaces.DepartmentDBInfo{temp1, temp2}
		Convey("获取子部门信息成功", func() {
			departmentDB.EXPECT().GetSubDepartmentInfos(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(outDepIDs, nil)
			departmentDB.EXPECT().GetSubDepartmentInfos(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(outDepIDs1, nil)

			out, num, err := departs.getSubDepartmentInfo("", interfaces.OrgShowPageInfo{Offset: 0, Limit: 100}, "")
			assert.Equal(t, err, nil)
			assert.Equal(t, num, 2)
			assert.Equal(t, len(out), 1)
			assert.Equal(t, out[0].ID, temp1.ID)
		})

		testErr := rest.NewHTTPError("error", 503000000, nil)
		Convey("获取子部门是否存在子部门信息, GetChildDepartmentIDs报错", func() {
			departmentDB.EXPECT().GetSubDepartmentInfos(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(outDepIDs, nil)
			departmentDB.EXPECT().GetSubDepartmentInfos(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(outDepIDs1, nil)
			departmentDB.EXPECT().GetChildDepartmentIDs(gomock.Any()).AnyTimes().Times(1).Return(nil, nil, testErr)
			_, _, err := departs.getSubDepartmentInfo("", interfaces.OrgShowPageInfo{Offset: 0, Limit: 100, BShowSubDepart: true}, "")
			assert.Equal(t, err, testErr)
		})

		Convey("获取子部门是否存在子用户信息, GetChildUserIDs报错", func() {
			departmentDB.EXPECT().GetSubDepartmentInfos(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(outDepIDs1, nil)
			departmentDB.EXPECT().GetSubDepartmentInfos(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(outDepIDs1, nil)
			departmentDB.EXPECT().GetChildUserIDs(gomock.Any()).AnyTimes().Times(1).Return(nil, nil, testErr)

			_, _, err := departs.getSubDepartmentInfo("", interfaces.OrgShowPageInfo{Offset: 0, Limit: 100, BShowSubUser: true}, "")
			assert.Equal(t, err, testErr)
		})

		Convey("获取父部门信息， GetDepartmentInfo报错", func() {
			departmentDB.EXPECT().GetSubDepartmentInfos(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(outDepIDs1, nil)
			departmentDB.EXPECT().GetSubDepartmentInfos(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(outDepIDs1, nil)
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(nil, testErr)

			_, _, err := departs.getSubDepartmentInfo("", interfaces.OrgShowPageInfo{Offset: 0, Limit: 100, BShowDepartParentDeps: true}, "")
			assert.Equal(t, err, testErr)
		})

		Convey("获取管理员信息报错 GetUserDBInfo报错", func() {
			departmentDB.EXPECT().GetSubDepartmentInfos(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(outDepIDs1, nil)
			departmentDB.EXPECT().GetSubDepartmentInfos(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(outDepIDs1, nil)
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).AnyTimes().Times(1).Return(nil, testErr)

			_, _, err := departs.getSubDepartmentInfo("", interfaces.OrgShowPageInfo{Offset: 0, Limit: 100, BShowDepartManager: true}, "")
			assert.Equal(t, err, testErr)
		})

		Convey("获取管理员信息成功", func() {
			user1 := interfaces.UserDBInfo{
				ID:   "xxx",
				Name: "yyy",
			}
			user2 := interfaces.UserDBInfo{
				ID:   "xxx1",
				Name: "yyy1",
			}
			dep1 := interfaces.DepartmentDBInfo{
				ID:        "xxx",
				Name:      "yyy",
				IsRoot:    0,
				Code:      "xxx",
				Status:    true,
				Remark:    "xxx",
				Email:     "xxx",
				ManagerID: user1.ID,
			}
			dep2 := interfaces.DepartmentDBInfo{
				ID:        "xxx1",
				Name:      "yyy1",
				IsRoot:    1,
				Code:      "xxx1",
				Status:    false,
				Remark:    "xxx1",
				Email:     "xxx1",
				ManagerID: user2.ID,
			}
			paredep1 := interfaces.DepartmentDBInfo{
				ID:   "paredep1",
				Name: "paredep1",
				Code: "paredep1",
			}
			paredep2 := interfaces.DepartmentDBInfo{
				ID:   "paredep2",
				Name: "paredep2",
				Code: "paredep2",
			}
			departmentDB.EXPECT().GetSubDepartmentInfos(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return([]interfaces.DepartmentDBInfo{dep1, dep2}, nil)
			departmentDB.EXPECT().GetSubDepartmentInfos(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return([]interfaces.DepartmentDBInfo{dep1, dep2, dep1}, nil)
			departmentDB.EXPECT().GetChildDepartmentIDs(gomock.Any()).AnyTimes().Times(1).Return(nil, map[string][]string{dep1.ID: {dep1.ID}}, nil)
			departmentDB.EXPECT().GetChildUserIDs(gomock.Any()).AnyTimes().Times(1).Return(nil, map[string][]string{dep2.ID: {dep2.ID}}, nil)
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return([]interfaces.DepartmentDBInfo{paredep1, paredep2}, nil)
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).AnyTimes().Times(1).Return([]interfaces.UserDBInfo{user1, user2}, nil)
			dta, num, err := departs.getSubDepartmentInfo("", interfaces.OrgShowPageInfo{Offset: 0, Limit: 100, BShowSubDepart: true,
				BShowSubUser: true, BShowDepartParentDeps: true, BShowDepartManager: true}, "paredep1/paredep2")
			assert.Equal(t, err, nil)
			assert.Equal(t, num, 3)
			assert.Equal(t, len(dta), 2)
			assert.Equal(t, dta[0].ID, dep1.ID)
			assert.Equal(t, dta[0].Name, dep1.Name)
			assert.Equal(t, dta[0].IsRoot, false)
			assert.Equal(t, dta[0].BUserExistd, false)
			assert.Equal(t, dta[0].BDepartExistd, true)
			assert.Equal(t, dta[0].Code, dep1.Code)
			assert.Equal(t, dta[0].Enabled, dep1.Status)
			assert.Equal(t, dta[0].Remark, dep1.Remark)
			assert.Equal(t, dta[0].Email, dep1.Email)
			assert.Equal(t, len(dta[0].ParentDeps), 2)
			assert.Equal(t, dta[0].ParentDeps[0].ID, paredep1.ID)
			assert.Equal(t, dta[0].ParentDeps[0].Name, paredep1.Name)
			assert.Equal(t, dta[0].ParentDeps[0].Type, "department")
			assert.Equal(t, dta[0].ParentDeps[0].Code, paredep1.Code)
			assert.Equal(t, dta[0].ParentDeps[1].ID, paredep2.ID)
			assert.Equal(t, dta[0].ParentDeps[1].Name, paredep2.Name)
			assert.Equal(t, dta[0].ParentDeps[1].Type, "department")
			assert.Equal(t, dta[0].ParentDeps[1].Code, paredep2.Code)
			assert.Equal(t, dta[0].Manager.ID, user1.ID)
			assert.Equal(t, dta[0].Manager.Name, user1.Name)
			assert.Equal(t, dta[1].ID, dep2.ID)
			assert.Equal(t, dta[1].Name, dep2.Name)
			assert.Equal(t, dta[1].IsRoot, true)
			assert.Equal(t, dta[1].BUserExistd, true)
			assert.Equal(t, dta[1].BDepartExistd, false)
			assert.Equal(t, dta[1].Code, dep2.Code)
			assert.Equal(t, dta[1].Enabled, dep2.Status)
			assert.Equal(t, dta[1].Remark, dep2.Remark)
			assert.Equal(t, dta[1].Email, dep2.Email)
			assert.Equal(t, len(dta[1].ParentDeps), 2)
			assert.Equal(t, dta[1].ParentDeps[0].ID, paredep1.ID)
			assert.Equal(t, dta[1].ParentDeps[0].Name, paredep1.Name)
			assert.Equal(t, dta[1].ParentDeps[0].Type, "department")
			assert.Equal(t, dta[1].ParentDeps[0].Code, paredep1.Code)
			assert.Equal(t, dta[1].ParentDeps[1].ID, paredep2.ID)
			assert.Equal(t, dta[1].ParentDeps[1].Name, paredep2.Name)
			assert.Equal(t, dta[1].ParentDeps[1].Type, "department")
			assert.Equal(t, dta[1].ParentDeps[1].Code, paredep2.Code)
			assert.Equal(t, dta[1].Manager.ID, user2.ID)
			assert.Equal(t, dta[1].Manager.Name, user2.Name)
		})
	})
}

func TestGetSubUsersInfo(t *testing.T) {
	Convey("获取子用户信息-逻辑层", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		departmentDB := mock.NewMockDBDepartment(ctrl)
		gmDB := mock.NewMockDBGroupMember(ctrl)
		userDB := mock.NewMockDBUser(ctrl)

		departs := department{
			db:            departmentDB,
			userDB:        userDB,
			groupMemberDB: gmDB,
		}

		Convey("获取子用户信息失败-报错", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			departmentDB.EXPECT().GetSubUserInfos(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(nil, testErr)

			_, _, err := departs.getSubUsersInfo("", 0, 100)
			assert.Equal(t, err, testErr)
		})

		temp1 := interfaces.UserDBInfo{
			ID:   "xxx",
			Name: "yyy",
		}
		outUserIDs := []interfaces.UserDBInfo{temp1}
		Convey("获取子用户信息失败1-报错", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			departmentDB.EXPECT().GetSubUserInfos(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(outUserIDs, nil)
			departmentDB.EXPECT().GetSubUserInfos(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(nil, testErr)

			_, _, err := departs.getSubUsersInfo("", 0, 100)
			assert.Equal(t, err, testErr)
		})

		temp2 := interfaces.UserDBInfo{
			ID:   "xxx1",
			Name: "yyy1",
		}
		outUserIDs1 := []interfaces.UserDBInfo{temp1, temp2}
		Convey("获取子用户信息成功", func() {
			departmentDB.EXPECT().GetSubUserInfos(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(outUserIDs, nil)
			departmentDB.EXPECT().GetSubUserInfos(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(outUserIDs1, nil)

			out, num, err := departs.getSubUsersInfo("", 0, 100)
			assert.Equal(t, err, nil)
			assert.Equal(t, num, 2)
			assert.Equal(t, len(out), 1)
			assert.Equal(t, out[0].ID, temp1.ID)
			assert.Equal(t, out[0].Type, "user")
		})
	})
}

func TestSearchDepartsByKey(t *testing.T) {
	Convey("部门关键字搜索-逻辑层", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		departmentDB := mock.NewMockDBDepartment(ctrl)
		gmDB := mock.NewMockDBGroupMember(ctrl)
		userDB := mock.NewMockDBUser(ctrl)
		trace := mock.NewMockTraceClient(ctrl)
		org := mock.NewMockLogicsOrgPerm(ctrl)
		departs := department{
			db:            departmentDB,
			userDB:        userDB,
			groupMemberDB: gmDB,
			trace:         trace,
			orgPerm:       org,
		}

		var info interfaces.OrgShowPageInfo
		var visitor interfaces.Visitor
		var ctx context.Context
		Convey("调用者是组织审计员，获取审计范围失败-报错", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			info.Role = interfaces.SystemRoleOrgAudit
			testErr := rest.NewHTTPError("error", 503000000, nil)
			userDB.EXPECT().GetOrgAduitDepartInfo(gomock.Any()).AnyTimes().Times(1).Return(nil, testErr)

			_, _, err := departs.SearchDepartsByKey(ctx, &visitor, info)
			assert.Equal(t, err, testErr)
		})

		Convey("调用者是组织管理员，获取管理范围失败-报错", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			info.Role = interfaces.SystemRoleOrgManager
			testErr := rest.NewHTTPError("error", 503000000, nil)
			userDB.EXPECT().GetOrgManagerDepartInfo(gomock.Any()).AnyTimes().Times(1).Return(nil, testErr)

			_, _, err := departs.SearchDepartsByKey(ctx, &visitor, info)
			assert.Equal(t, err, testErr)
		})

		Convey("调用者没有特定角色", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			info.Role = "xxxx11"
			out, num, err := departs.SearchDepartsByKey(ctx, &visitor, info)
			assert.Equal(t, err, nil)
			assert.Equal(t, num, 0)
			assert.Equal(t, len(out), 0)
		})

		Convey("调用者为普通用户，没有权限", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			org.EXPECT().CheckPerms(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(false, nil)
			info.Role = interfaces.SystemRoleNormalUser
			out, num, err := departs.SearchDepartsByKey(ctx, &visitor, info)
			assert.Equal(t, err, nil)
			assert.Equal(t, num, 0)
			assert.Equal(t, len(out), 0)
		})

		tempIDs := []string{"xxx", strID1}
		Convey("获取子部门失败-报错", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			info.Role = interfaces.SystemRoleOrgManager
			testErr := rest.NewHTTPError("error", 503000000, nil)
			userDB.EXPECT().GetOrgManagerDepartInfo(gomock.Any()).AnyTimes().Times(1).Return(tempIDs, nil)
			departmentDB.EXPECT().GetDepartmentInfo2(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(nil, testErr)

			_, _, err := departs.SearchDepartsByKey(ctx, &visitor, info)
			assert.Equal(t, err, testErr)
		})

		temp1 := interfaces.DepartmentDBInfo{
			ID:   "xxx",
			Name: "asda",
			Path: "asda/xxxx",
		}

		temp2 := interfaces.DepartmentDBInfo{
			ID:   "xxx1",
			Name: "asda1",
		}

		out1 := []interfaces.DepartmentDBInfo{temp1}
		out2 := []interfaces.DepartmentDBInfo{temp1, temp2}

		Convey("子部门内搜索失败-报错", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			info.Role = interfaces.SystemRoleSecAdmin
			testErr := rest.NewHTTPError("error", 503000000, nil)
			departmentDB.EXPECT().GetDepartmentInfo2(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(nil, nil)
			departmentDB.EXPECT().SearchDepartsByKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(out1, testErr)

			_, _, err := departs.SearchDepartsByKey(ctx, &visitor, info)
			assert.Equal(t, err, testErr)
		})

		Convey("子部门内搜索失败1-报错", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			info.Role = interfaces.SystemRoleSecAdmin
			testErr := rest.NewHTTPError("error", 503000000, nil)
			departmentDB.EXPECT().GetDepartmentInfo2(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(nil, nil)
			departmentDB.EXPECT().SearchDepartsByKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(out1, nil)
			departmentDB.EXPECT().SearchDepartsByKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(out1, testErr)

			_, _, err := departs.SearchDepartsByKey(ctx, &visitor, info)
			assert.Equal(t, err, testErr)
		})

		Convey("处理部门父部门路径失败-报错", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			info.Role = interfaces.SystemRoleSecAdmin
			testErr := rest.NewHTTPError("error", 503000000, nil)
			departmentDB.EXPECT().GetDepartmentInfo2(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(nil, nil)
			departmentDB.EXPECT().SearchDepartsByKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(out1, nil)
			departmentDB.EXPECT().SearchDepartsByKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(out2, nil)
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(nil, testErr)

			_, _, err := departs.SearchDepartsByKey(ctx, &visitor, info)
			assert.Equal(t, err, testErr)
		})

		temp3 := interfaces.DepartmentDBInfo{
			ID:   "asda",
			Name: "asdad",
		}
		pathDeparts := []interfaces.DepartmentDBInfo{temp3}
		Convey("部门搜索成功", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			info.Role = interfaces.SystemRoleSecAdmin
			departmentDB.EXPECT().GetDepartmentInfo2(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(nil, nil)
			departmentDB.EXPECT().SearchDepartsByKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(out1, nil)
			departmentDB.EXPECT().SearchDepartsByKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(out2, nil)
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(pathDeparts, nil)

			out, num, err := departs.SearchDepartsByKey(ctx, &visitor, info)
			assert.Equal(t, err, nil)
			assert.Equal(t, num, 2)
			assert.Equal(t, len(out), 1)
			assert.Equal(t, out[0].Path, temp3.Name)
			assert.Equal(t, out[0].Name, temp1.Name)
			assert.Equal(t, out[0].Type, "department")
		})

		Convey("普通用户部门搜索成功", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			info.Role = interfaces.SystemRoleNormalUser
			org.EXPECT().CheckPerms(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(true, nil)
			departmentDB.EXPECT().GetDepartmentInfo2(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(nil, nil)
			departmentDB.EXPECT().SearchDepartsByKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(out1, nil)
			departmentDB.EXPECT().SearchDepartsByKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(out2, nil)
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(pathDeparts, nil)

			out, num, err := departs.SearchDepartsByKey(ctx, &visitor, info)
			assert.Equal(t, err, nil)
			assert.Equal(t, num, 2)
			assert.Equal(t, len(out), 1)
			assert.Equal(t, out[0].Path, temp3.Name)
			assert.Equal(t, out[0].Name, temp1.Name)
			assert.Equal(t, out[0].Type, "department")
		})
	})
}

func TestGetParentPath(t *testing.T) {
	Convey("获取父部门路径-逻辑层", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		departmentDB := mock.NewMockDBDepartment(ctrl)

		depInfo := interfaces.DepartmentDBInfo{ID: "xxx12", Name: "test", Path: "ID2/ID1/zzz"}
		var ctx context.Context
		Convey("获取部门信息失败-报错", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(nil, testErr)

			_, _, err := getParentDep(departmentDB, &depInfo, false, ctx)
			assert.Equal(t, err, testErr)
		})

		temp1 := interfaces.DepartmentDBInfo{
			ID:   "ID1",
			Name: "Name1",
		}
		temp2 := interfaces.DepartmentDBInfo{
			ID:   "ID2",
			Name: "Name2",
		}

		test1 := interfaces.ObjectBaseInfo{
			ID:   temp1.ID,
			Name: temp1.Name,
			Type: "department",
		}

		test2 := interfaces.ObjectBaseInfo{
			ID:   temp2.ID,
			Name: temp2.Name,
			Type: "department",
		}

		Convey("获取父部门信息失败-报错", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(nil, testErr)
			_, _, err := getParentDep(departmentDB, &depInfo, false, ctx)
			assert.Equal(t, err, testErr)
		})

		out1 := []interfaces.DepartmentDBInfo{temp1, temp2}
		Convey("获取父部门路径成功", func() {
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(out1, nil)

			out, path, err := getParentDep(departmentDB, &depInfo, false, ctx)
			assert.Equal(t, err, nil)
			assert.Equal(t, path, temp2.Name+"/"+temp1.Name)
			assert.Equal(t, out, []interfaces.ObjectBaseInfo{test2, test1})
		})
	})
}

func TestCheckDepInRange(t *testing.T) {
	Convey("部门在特定角色范围内检测", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		departmentDB := mock.NewMockDBDepartment(ctrl)
		userDB := mock.NewMockDBUser(ctrl)

		roleID := interfaces.SystemRoleOrgAudit
		Convey("获取组织审计员审计范围失败-报错", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			userDB.EXPECT().GetOrgAduitDepartInfo(gomock.Any()).AnyTimes().Times(1).Return([]string{"xxx"}, testErr)

			_, err := checkDepInRange(userDB, departmentDB, "", roleID, [][]string{})
			assert.Equal(t, err, testErr)
		})

		roleID = interfaces.SystemRoleOrgManager
		Convey("获取组织管理员管理范围失败-报错", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			userDB.EXPECT().GetOrgManagerDepartInfo(gomock.Any()).AnyTimes().Times(1).Return([]string{"xxx"}, testErr)

			_, err := checkDepInRange(userDB, departmentDB, "", roleID, [][]string{})
			assert.Equal(t, err, testErr)
		})

		Convey("获取子部门ID失败-报错", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			userDB.EXPECT().GetOrgManagerDepartInfo(gomock.Any()).AnyTimes().Times(1).Return([]string{"xxx"}, nil)
			departmentDB.EXPECT().GetChildDepartmentIDs(gomock.Any()).AnyTimes().Times(1).Return([]string{"xxx"}, nil, testErr)

			_, err := checkDepInRange(userDB, departmentDB, "", roleID, [][]string{})
			assert.Equal(t, err, testErr)
		})

		Convey("检测完成", func() {
			userDB.EXPECT().GetOrgManagerDepartInfo(gomock.Any()).AnyTimes().Times(1).Return([]string{"xxx"}, nil)
			departmentDB.EXPECT().GetChildDepartmentIDs(gomock.Any()).AnyTimes().Times(1).Return([]string{"xxx1"}, nil, nil)
			departmentDB.EXPECT().GetChildDepartmentIDs(gomock.Any()).AnyTimes().Times(1).Return([]string{}, nil, nil)

			temp := []string{"xxx1"}
			ret, err := checkDepInRange(userDB, departmentDB, "", roleID, [][]string{temp})
			assert.Equal(t, err, nil)
			assert.Equal(t, ret, []bool{true})
		})
	})
}

func TestGetAllChildDeparmentIDsAll(t *testing.T) {
	Convey("获取所有的子部门ID-逻辑层", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		departmentDB := mock.NewMockDBDepartment(ctrl)
		var ctx context.Context

		Convey("传入部门数组为空-返回空部门信息", func() {
			out, err := getAllChildDeparmentIDs(departmentDB, []string{}, ctx)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(out), 0)
		})

		Convey("获取子部门信息失败-报错", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			departmentDB.EXPECT().GetChildDepartmentIDs(gomock.Any()).AnyTimes().Times(1).Return(nil, nil, testErr)
			_, err := getAllChildDeparmentIDs(departmentDB, []string{"xx"}, ctx)
			assert.Equal(t, err, testErr)
		})

		Convey("获取所有子部门信息", func() {
			departmentDB.EXPECT().GetChildDepartmentIDs(gomock.Any()).AnyTimes().Times(1).Return([]string{"ID1"}, nil, nil)
			departmentDB.EXPECT().GetChildDepartmentIDs(gomock.Any()).AnyTimes().Times(1).Return([]string{}, nil, nil)
			out, err := getAllChildDeparmentIDs(departmentDB, []string{"xx"}, ctx)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(out), 1)
			assert.Equal(t, out[0], "ID1")
		})
	})
}

func TestGetDepartsInfo(t *testing.T) {
	Convey("获取部门信息-逻辑层", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		departmentDB := mock.NewMockDBDepartment(ctrl)
		gmDB := mock.NewMockDBGroupMember(ctrl)
		userDB := mock.NewMockDBUser(ctrl)
		roleDB := mock.NewMockLogicsRole(ctrl)
		departs := department{
			db:            departmentDB,
			userDB:        userDB,
			groupMemberDB: gmDB,
			role:          roleDB,
		}

		var depIDs []string
		var scopde interfaces.DepartInfoScope
		Convey("如果长度为0，则直接返回", func() {
			outs, err := departs.GetDepartsInfo(depIDs, scopde, true)
			assert.Equal(t, len(outs), 0)
			assert.Equal(t, err, nil)
		})

		depIDs = []string{"xxx", "xxx"}
		Convey("如果id重复，则报错", func() {
			_, err := departs.GetDepartsInfo(depIDs, scopde, true)
			assert.Equal(t, err, rest.NewHTTPError("departmentID is not unique", rest.BadRequest, nil))
		})

		testErr := rest.NewHTTPError("error", 503000000, nil)
		depIDs = []string{"xxx", "xxx1"}
		departInfos := make([]interfaces.DepartmentDBInfo, 0)
		Convey("获取部门信息失败，报错", func() {
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(departInfos, testErr)
			_, err := departs.GetDepartsInfo(depIDs, scopde, true)
			assert.Equal(t, err, testErr)
		})

		depIDs = []string{"depart1", "depart2"}
		departInfos = append(departInfos, interfaces.DepartmentDBInfo{ID: "depart1", Name: "name5"})
		Convey("获取部门信息错误，缺少部门，报错", func() {
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(departInfos, nil)
			_, err := departs.GetDepartsInfo(depIDs, scopde, true)
			testErr1 := rest.NewHTTPErrorV2(errors.NotFound, "department does not exist",
				rest.SetDetail(map[string]interface{}{"ids": []string{"depart2"}}))
			assert.Equal(t, err, testErr1)
		})

		departInfos = append(departInfos, interfaces.DepartmentDBInfo{ID: "depart2", Name: "name6"})
		depNameInfos := make([]interfaces.NameInfo, 0)
		scopde.BParentDeps = true
		Convey("如果获取部门父部门路径，获取部门名称失败，报错", func() {
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(departInfos, nil)
			departmentDB.EXPECT().GetDepartmentName(gomock.Any()).AnyTimes().Return(depNameInfos, []string{}, testErr)
			_, err := departs.GetDepartsInfo(depIDs, scopde, true)
			assert.Equal(t, err, testErr)
		})

		scopde.BParentDeps = false
		scopde.BManagers = true
		Convey("如果获取部门管理员，获取部门管理员失败，报错", func() {
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(departInfos, nil)
			departmentDB.EXPECT().GetManagersOfDepartment(gomock.Any()).AnyTimes().Return(nil, testErr)
			_, err := departs.GetDepartsInfo(depIDs, scopde, true)
			assert.Equal(t, err, testErr)
		})

		departInfos[0].Path = "sub1/sub2/sub3/depart1"
		departInfos[1].Path = "sub2/sub3/sub4/depart2"
		depNameInfos = []interfaces.NameInfo{{ID: "sub1", Name: "name1"}, {ID: "sub2", Name: "name2"}, {ID: "sub3", Name: "name3"},
			{ID: "sub4", Name: "name4"}, {ID: "depart1", Name: "name5"}, {ID: "depart2", Name: "name6"}}
		managers := make([]interfaces.DepartmentManagerInfo, 0)
		managers = append(managers, interfaces.DepartmentManagerInfo{DepartmentID: departInfos[0].ID, Managers: []interfaces.NameInfo{{ID: "manager1", Name: "manager1Name"}}},
			interfaces.DepartmentManagerInfo{DepartmentID: departInfos[1].ID, Managers: []interfaces.NameInfo{{ID: "manager2", Name: "manager2Name"}}})
		scopde.BShowName = true
		scopde.BParentDeps = true
		scopde.BManagers = true

		Convey("成功", func() {
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(departInfos, nil)
			departmentDB.EXPECT().GetDepartmentName(gomock.Any()).AnyTimes().Return(depNameInfos, []string{}, nil)
			departmentDB.EXPECT().GetManagersOfDepartment(gomock.Any()).AnyTimes().Return(managers, nil)
			outInfos, err := departs.GetDepartsInfo(depIDs, scopde, true)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(outInfos), 2)

			temp1 := []interfaces.ObjectBaseInfo{}
			temp1 = append(temp1, interfaces.ObjectBaseInfo{ID: "sub1", Name: "name1", Type: "department"},
				interfaces.ObjectBaseInfo{ID: "sub2", Name: "name2", Type: "department"},
				interfaces.ObjectBaseInfo{ID: "sub3", Name: "name3", Type: "department"})
			assert.Equal(t, outInfos[0], interfaces.DepartInfo{ID: "depart1", Name: "name5", ParentDeps: temp1, Managers: managers[0].Managers})

			temp2 := []interfaces.ObjectBaseInfo{}
			temp2 = append(temp2, interfaces.ObjectBaseInfo{ID: "sub2", Name: "name2", Type: "department"},
				interfaces.ObjectBaseInfo{ID: "sub3", Name: "name3", Type: "department"},
				interfaces.ObjectBaseInfo{ID: "sub4", Name: "name4", Type: "department"})
			assert.Equal(t, outInfos[1], interfaces.DepartInfo{ID: "depart2", Name: "name6", ParentDeps: temp2, Managers: managers[1].Managers})
		})
	})
}

func TestGetDepartsInfoByLevel(t *testing.T) {
	Convey("根据部门等级获取部门信息-逻辑层", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		departmentDB := mock.NewMockDBDepartment(ctrl)
		gmDB := mock.NewMockDBGroupMember(ctrl)
		userDB := mock.NewMockDBUser(ctrl)
		roleDB := mock.NewMockLogicsRole(ctrl)
		departs := department{
			db:            departmentDB,
			userDB:        userDB,
			groupMemberDB: gmDB,
			role:          roleDB,
		}

		Convey("如果层级小于0，则报错", func() {
			_, err := departs.GetDepartsInfoByLevel(-1)
			assert.Equal(t, err, rest.NewHTTPError("level is illegal", rest.BadRequest, nil))
		})

		infos := make([]interfaces.DepartmentDBInfo, 0)
		testErr := rest.NewHTTPError("error", 503000000, nil)
		Convey("获取层级信息错误，则报错", func() {
			departmentDB.EXPECT().GetDepartmentByPathLength(gomock.Any()).AnyTimes().Return(infos, testErr)
			_, err := departs.GetDepartsInfoByLevel(1)
			assert.Equal(t, err, testErr)
		})

		infos = append(infos, interfaces.DepartmentDBInfo{ID: "ID1", Name: "name1", ThirdID: "third_id1"}, interfaces.DepartmentDBInfo{ID: "ID2", Name: "name2", ThirdID: "third_id2"})
		Convey("接口调用成功", func() {
			departmentDB.EXPECT().GetDepartmentByPathLength(73).AnyTimes().Return(infos, nil)
			out, err := departs.GetDepartsInfoByLevel(1)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(out), 2)
			assert.Equal(t, out[0], interfaces.ObjectBaseInfo{ID: "ID1", Name: "name1", Type: "department", ThirdID: "third_id1"})
			assert.Equal(t, out[1], interfaces.ObjectBaseInfo{ID: "ID2", Name: "name2", Type: "department", ThirdID: "third_id2"})
		})
	})
}

func TestGetAllDepartUserInfos(t *testing.T) {
	Convey("GetAllDepartUserInfos, db is available", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		departmentDB := mock.NewMockDBDepartment(ctrl)
		gmDB := mock.NewMockDBGroupMember(ctrl)
		user := mock.NewMockDBUser(ctrl)
		departmentLogics := newDepartment(user, departmentDB, gmDB, nil)

		var depID string = strID1
		Convey("GetDepartmentInfo error", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, testErr)

			_, err := departmentLogics.GetAllDepartUserInfos(depID)
			assert.Equal(t, err, testErr)
		})

		var strList []interfaces.DepartmentDBInfo
		Convey("department not exsit", func() {
			testErr1 := rest.NewHTTPErrorV2(errors.NotFound, "department does not exist")
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(strList, nil)

			_, err := departmentLogics.GetAllDepartUserInfos(depID)
			assert.Equal(t, err, testErr1)
		})

		strList = append(strList, interfaces.DepartmentDBInfo{})
		Convey("GetAllSubUserIDsByDepartPath error", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(strList, nil)
			departmentDB.EXPECT().GetAllSubUserInfosByDepartPath(gomock.Any()).AnyTimes().Return(nil, testErr)

			_, err := departmentLogics.GetAllDepartUserInfos(depID)
			assert.Equal(t, err, testErr)
		})

		user1 := interfaces.UserBaseInfo{
			ID:        "ID1",
			Name:      "name1",
			Account:   "account1",
			Email:     "email1",
			TelNumber: "telephone1",
			ThirdAttr: "thridattr1",
			ThirdID:   "thirdid1",
		}
		user2 := interfaces.UserBaseInfo{
			ID:        "ID1",
			Name:      "name1",
			Account:   "account1",
			Email:     "email1",
			TelNumber: "telephone1",
			ThirdAttr: "thridattr1",
			ThirdID:   "thirdid1",
		}
		user3 := interfaces.UserBaseInfo{
			ID:        "ID3",
			Name:      "name3",
			Account:   "account3",
			Email:     "email3",
			TelNumber: "telephone3",
			ThirdAttr: "thridattr3",
			ThirdID:   "thirdid3",
		}
		userInfo := []interfaces.UserBaseInfo{user1, user2, user3}
		Convey("success", func() {
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(strList, nil)
			departmentDB.EXPECT().GetAllSubUserInfosByDepartPath(gomock.Any()).AnyTimes().Return(userInfo, nil)

			outInfo, err := departmentLogics.GetAllDepartUserInfos(depID)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(outInfo), 2)
			assert.Equal(t, outInfo[0], user2)
			assert.Equal(t, outInfo[1], user3)
		})
	})
}

func TestFilterTopDepartments(t *testing.T) {
	Convey("filterTopDepartments, db is available", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		departmentDB := mock.NewMockDBDepartment(ctrl)
		gmDB := mock.NewMockDBGroupMember(ctrl)
		user := mock.NewMockDBUser(ctrl)
		departmentLogics := newDepartment(user, departmentDB, gmDB, nil)

		Convey("GetDepartmentInfo error", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, testErr)

			_, err := departmentLogics.filterTopDepartments(nil)
			assert.Equal(t, err, testErr)
		})

		data1 := interfaces.DepartmentDBInfo{
			ID:   "data1",
			Path: "path1",
		}
		data2 := interfaces.DepartmentDBInfo{
			ID:   "data2",
			Path: "path2",
		}
		data3 := interfaces.DepartmentDBInfo{
			ID:   "data3",
			Path: "path2/path3",
		}
		Convey("success", func() {
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return([]interfaces.DepartmentDBInfo{data1, data2, data3}, nil)

			out, err := departmentLogics.filterTopDepartments(nil)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(out), 2)

			test1 := make(map[string]bool)
			for _, v := range out {
				test1[v] = true
			}
			assert.Equal(t, test1[data1.ID], true)
			assert.Equal(t, test1[data2.ID], true)
		})
	})
}

func TestDeleteOrgManagerRelationByDepartID(t *testing.T) {
	Convey("DeleteOrgManagerRelationByDepartID, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		departmentDB := mock.NewMockDBDepartment(ctrl)
		gmDB := mock.NewMockDBGroupMember(ctrl)
		user := mock.NewMockDBUser(ctrl)
		departmentLogics := newDepartment(user, departmentDB, gmDB, nil)

		Convey("DeleteOrgManagerRelationByDepartID error", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			departmentDB.EXPECT().DeleteOrgManagerRelationByDepartID(gomock.Any()).AnyTimes().Return(testErr)

			err := departmentLogics.DeleteOrgManagerRelationByDepartID("")
			assert.Equal(t, err, testErr)
		})
	})
}

func TestDeleteOrgAuditRelationByDepartID(t *testing.T) {
	Convey("DeleteOrgAuditRelationByDepartID, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		departmentDB := mock.NewMockDBDepartment(ctrl)
		gmDB := mock.NewMockDBGroupMember(ctrl)
		user := mock.NewMockDBUser(ctrl)
		departmentLogics := newDepartment(user, departmentDB, gmDB, nil)

		Convey("DeleteOrgManagerRelationByDepartID error", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			departmentDB.EXPECT().DeleteOrgAuditRelationByDepartID(gomock.Any()).AnyTimes().Return(testErr)

			err := departmentLogics.DeleteOrgAuditRelationByDepartID("")
			assert.Equal(t, err, testErr)
		})
	})
}

func TestDeleteDepartInfo(t *testing.T) {
	Convey("deleteDepartInfo, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		departmentDB := mock.NewMockDBDepartment(ctrl)
		gmDB := mock.NewMockDBGroupMember(ctrl)
		user := mock.NewMockDBUser(ctrl)
		departmentLogics := newDepartment(user, departmentDB, gmDB, nil)

		testErr := rest.NewHTTPError("error", 503000000, nil)
		Convey("DeleteUserDepartRelationByPath error", func() {
			departmentDB.EXPECT().DeleteUserDepartRelationByPath(gomock.Any(), gomock.Any()).AnyTimes().Return(testErr)

			err := departmentLogics.deleteDepartInfo("", nil, nil, nil, nil)
			assert.Equal(t, err, testErr)
		})

		Convey("AddUserToDepart error", func() {
			departmentDB.EXPECT().DeleteUserDepartRelationByPath(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departmentDB.EXPECT().AddUserToDepart(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(testErr)

			err := departmentLogics.deleteDepartInfo("", nil, nil, nil, nil)
			assert.Equal(t, err, testErr)
		})

		Convey("DeleteUserOURelation error", func() {
			departmentDB.EXPECT().DeleteUserDepartRelationByPath(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departmentDB.EXPECT().AddUserToDepart(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departmentDB.EXPECT().DeleteUserOURelation(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(testErr)

			err := departmentLogics.deleteDepartInfo("", nil, nil, nil, nil)
			assert.Equal(t, err, testErr)
		})

		Convey("DeleteDepartByPath error", func() {
			departmentDB.EXPECT().DeleteUserDepartRelationByPath(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departmentDB.EXPECT().AddUserToDepart(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departmentDB.EXPECT().DeleteUserOURelation(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departmentDB.EXPECT().DeleteDepartByPath(gomock.Any(), gomock.Any()).AnyTimes().Return(testErr)

			err := departmentLogics.deleteDepartInfo("", nil, nil, nil, nil)
			assert.Equal(t, err, testErr)
		})

		Convey("DeleteDepartRelations error", func() {
			departmentDB.EXPECT().DeleteUserDepartRelationByPath(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departmentDB.EXPECT().AddUserToDepart(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departmentDB.EXPECT().DeleteUserOURelation(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departmentDB.EXPECT().DeleteDepartByPath(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departmentDB.EXPECT().DeleteDepartRelations(gomock.Any(), gomock.Any()).AnyTimes().Return(testErr)

			err := departmentLogics.deleteDepartInfo("", nil, nil, nil, nil)
			assert.Equal(t, err, testErr)
		})

		Convey("DeleteDepartOURelations error", func() {
			departmentDB.EXPECT().DeleteUserDepartRelationByPath(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departmentDB.EXPECT().AddUserToDepart(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departmentDB.EXPECT().DeleteUserOURelation(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departmentDB.EXPECT().DeleteDepartByPath(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departmentDB.EXPECT().DeleteDepartRelations(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departmentDB.EXPECT().DeleteDepartOURelations(gomock.Any(), gomock.Any()).AnyTimes().Return(testErr)

			err := departmentLogics.deleteDepartInfo("", nil, nil, nil, nil)
			assert.Equal(t, err, testErr)
			assert.Equal(t, 1, 1)
		})

		Convey("success", func() {
			departmentDB.EXPECT().DeleteUserDepartRelationByPath(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departmentDB.EXPECT().AddUserToDepart(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departmentDB.EXPECT().DeleteUserOURelation(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departmentDB.EXPECT().DeleteDepartByPath(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departmentDB.EXPECT().DeleteDepartRelations(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departmentDB.EXPECT().DeleteDepartOURelations(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

			err := departmentLogics.deleteDepartInfo("", nil, nil, nil, nil)
			assert.Equal(t, err, nil)
		})
	})
}

func TestHandleDeletedDepartInfo(t *testing.T) {
	Convey("handleDeletedDepartInfo, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		departmentDB := mock.NewMockDBDepartment(ctrl)
		gmDB := mock.NewMockDBGroupMember(ctrl)
		user := mock.NewMockDBUser(ctrl)
		departmentLogics := newDepartment(user, departmentDB, gmDB, nil)

		testErr := rest.NewHTTPError("error", 503000000, nil)
		Convey("GetAllSubUserIDsByDepartPath error", func() {
			departmentDB.EXPECT().GetAllSubUserIDsByDepartPath(gomock.Any()).AnyTimes().Return(nil, testErr)

			_, _, _, _, err := departmentLogics.handleDeletedDepartInfo("")
			assert.Equal(t, err, testErr)
		})

		Convey("GetUsersPath error", func() {
			departmentDB.EXPECT().GetAllSubUserIDsByDepartPath(gomock.Any()).AnyTimes().Return(nil, nil)
			user.EXPECT().GetUsersPath(gomock.Any()).AnyTimes().Return(nil, testErr)

			_, _, _, _, err := departmentLogics.handleDeletedDepartInfo("")
			assert.Equal(t, err, testErr)
		})

		Convey("GetAllSubDepartIDsByPath error", func() {
			departmentDB.EXPECT().GetAllSubUserIDsByDepartPath(gomock.Any()).AnyTimes().Return(nil, nil)
			user.EXPECT().GetUsersPath(gomock.Any()).AnyTimes().Return(nil, nil)
			departmentDB.EXPECT().GetAllSubDepartInfosByPath(gomock.Any()).AnyTimes().Return(nil, testErr)

			_, _, _, _, err := departmentLogics.handleDeletedDepartInfo("")
			assert.Equal(t, err, testErr)
		})

		Convey("GetAllOrgManagerIDsByDepartIDs error", func() {
			departmentDB.EXPECT().GetAllSubUserIDsByDepartPath(gomock.Any()).AnyTimes().Return(nil, nil)
			user.EXPECT().GetUsersPath(gomock.Any()).AnyTimes().Return(nil, nil)
			departmentDB.EXPECT().GetAllSubDepartInfosByPath(gomock.Any()).AnyTimes().Return(nil, nil)
			departmentDB.EXPECT().GetAllOrgManagerIDsByDepartIDs(gomock.Any()).AnyTimes().Return(nil, testErr)

			_, _, _, _, err := departmentLogics.handleDeletedDepartInfo("")
			assert.Equal(t, err, testErr)
		})

		departPath := "D1/D12"
		allUserPaths := make(map[string][]string)
		allUserPaths["U1"] = []string{"D1/D12", "D1/D22"}
		allUserPaths["U2"] = []string{"D1/D12", "D1/D12/D13"}
		allUserPaths["U3"] = []string{"D1/D12", "D2"}

		departInfo1 := interfaces.DepartmentDBInfo{
			ID:   "D1",
			Path: "D1",
		}
		Convey("success", func() {
			departmentDB.EXPECT().GetAllSubUserIDsByDepartPath(gomock.Any()).AnyTimes().Return(nil, nil)
			user.EXPECT().GetUsersPath(gomock.Any()).AnyTimes().Return(allUserPaths, nil)
			departmentDB.EXPECT().GetAllSubDepartInfosByPath(gomock.Any()).AnyTimes().Return([]interfaces.DepartmentDBInfo{departInfo1}, nil)
			departmentDB.EXPECT().GetAllOrgManagerIDsByDepartIDs(gomock.Any()).AnyTimes().Return([]string{strID}, nil)

			needAddToUnDistributeIDs, needDeleteOUIDs, allDepIDs, allOrgIDs, err := departmentLogics.handleDeletedDepartInfo(departPath)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(needAddToUnDistributeIDs), 1)
			assert.Equal(t, needAddToUnDistributeIDs[0], "U2")
			assert.Equal(t, len(needDeleteOUIDs), 2)
			assert.Equal(t, len(allDepIDs), 1)
			assert.Equal(t, allDepIDs[0], "D1")

			needDeleteOUIDMap := make(map[string]bool)
			needDeleteOUIDMap[needDeleteOUIDs[0]] = true
			needDeleteOUIDMap[needDeleteOUIDs[1]] = true
			needDeleteOUIDMap["U2"] = true
			needDeleteOUIDMap["U3"] = true

			assert.Equal(t, len(allOrgIDs), 1)
			assert.Equal(t, allOrgIDs[0], strID)
		})
	})
}

func TestCheckDepartInScope(t *testing.T) {
	Convey("checkDepartInScope, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		departmentDB := mock.NewMockDBDepartment(ctrl)
		gmDB := mock.NewMockDBGroupMember(ctrl)
		user := mock.NewMockDBUser(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		departmentLogics := newDepartment(user, departmentDB, gmDB, role)

		testErr := rest.NewHTTPError("error", 503000000, nil)
		var departInfo interfaces.DepartmentDBInfo

		visitor := &interfaces.Visitor{ID: strID1}
		Convey("getRolesByUserID error", func() {
			roles := make(map[string]map[interfaces.Role]bool)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roles, testErr)

			err := departmentLogics.checkDepartInScope(visitor, &departInfo)
			assert.Equal(t, err, testErr)
		})

		Convey("超级管理员有权限", func() {
			roles := make(map[string]map[interfaces.Role]bool)
			roles[strID1] = map[interfaces.Role]bool{interfaces.SystemRoleSuperAdmin: true}
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roles, nil)

			err := departmentLogics.checkDepartInScope(visitor, &departInfo)
			assert.Equal(t, err, nil)
		})

		Convey("系统管理员有权限", func() {
			roles := make(map[string]map[interfaces.Role]bool)
			roles[strID1] = map[interfaces.Role]bool{interfaces.SystemRoleSysAdmin: true}
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roles, nil)

			err := departmentLogics.checkDepartInScope(visitor, &departInfo)
			assert.Equal(t, err, nil)
		})

		Convey("组织管理员时，GetUsersPath 报错", func() {
			roles := make(map[string]map[interfaces.Role]bool)
			roles[strID1] = map[interfaces.Role]bool{interfaces.SystemRoleOrgManager: true}
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roles, nil)
			user.EXPECT().GetUsersPath(gomock.Any()).AnyTimes().Return(nil, testErr)

			err := departmentLogics.checkDepartInScope(visitor, &departInfo)
			assert.Equal(t, err, testErr)
		})

		Convey("组织管理员时，GetOrgManagerDepartInfo 报错", func() {
			roles := make(map[string]map[interfaces.Role]bool)
			roles[strID1] = map[interfaces.Role]bool{interfaces.SystemRoleOrgManager: true}
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roles, nil)
			user.EXPECT().GetUsersPath(gomock.Any()).AnyTimes().Return(nil, nil)
			user.EXPECT().GetOrgManagerDepartInfo(gomock.Any()).AnyTimes().Return(nil, testErr)

			err := departmentLogics.checkDepartInScope(visitor, &departInfo)
			assert.Equal(t, err, testErr)
		})

		Convey("组织管理员时，GetDepartmentInfo报错", func() {
			roles := make(map[string]map[interfaces.Role]bool)
			roles[strID1] = map[interfaces.Role]bool{interfaces.SystemRoleOrgManager: true}
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roles, nil)
			user.EXPECT().GetUsersPath(gomock.Any()).AnyTimes().Return(nil, nil)
			user.EXPECT().GetOrgManagerDepartInfo(gomock.Any()).AnyTimes().Return(nil, nil)
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, testErr)
			err := departmentLogics.checkDepartInScope(visitor, &departInfo)
			assert.Equal(t, err, testErr)
		})

		Convey("组织管理员时，如果组织管理员在这个部门， 报错", func() {
			visitor.Language = interfaces.SimplifiedChinese
			departmentLogics.i18n = common.NewI18n(common.I18nMap{
				i18nIDObjectsInDepartDeleteNotContain: {
					interfaces.SimplifiedChinese:  "用户无法删除自己所在的部门",
					interfaces.TraditionalChinese: "使用者無法刪除自己所在的部門",
					interfaces.AmericanEnglish:    "The user can't delete his department.",
				}})
			departInfo.Path = strID1
			paths := make(map[string][]string)
			paths[strID1] = []string{strID1}
			roles := make(map[string]map[interfaces.Role]bool)
			roles[strID1] = map[interfaces.Role]bool{interfaces.SystemRoleOrgManager: true}
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roles, nil)
			user.EXPECT().GetUsersPath(gomock.Any()).AnyTimes().Return(paths, nil)

			err := departmentLogics.checkDepartInScope(visitor, &departInfo)
			assert.Equal(t, err, rest.NewHTTPErrorV2(rest.Forbidden, departmentLogics.i18n.Load(i18nIDObjectsInDepartDeleteNotContain, visitor.Language)))
		})

		test401Error := rest.NewHTTPError("this user do not has the authority", errors.Forbidden, nil)
		Convey("组织管理员时，如果不在范围内，则报错", func() {
			roles := make(map[string]map[interfaces.Role]bool)
			roles[strID1] = map[interfaces.Role]bool{interfaces.SystemRoleOrgManager: true}
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roles, nil)
			user.EXPECT().GetUsersPath(gomock.Any()).AnyTimes().Return(nil, nil)
			user.EXPECT().GetOrgManagerDepartInfo(gomock.Any()).AnyTimes().Return(nil, nil)
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
			err := departmentLogics.checkDepartInScope(visitor, &departInfo)
			assert.Equal(t, err, test401Error)
		})

		Convey("组织管理员时，如果在范围内，则正常返回", func() {
			departInfo.Path = "path1/path2"
			tempDepart := interfaces.DepartmentDBInfo{
				Path: "path1",
			}
			rangeDepartInfos := []interfaces.DepartmentDBInfo{tempDepart}
			roles := make(map[string]map[interfaces.Role]bool)
			roles[strID1] = map[interfaces.Role]bool{interfaces.SystemRoleOrgManager: true}
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roles, nil)
			user.EXPECT().GetUsersPath(gomock.Any()).AnyTimes().Return(nil, nil)
			user.EXPECT().GetOrgManagerDepartInfo(gomock.Any()).AnyTimes().Return(nil, nil)
			departmentDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(rangeDepartInfos, nil)
			err := departmentLogics.checkDepartInScope(visitor, &departInfo)
			assert.Equal(t, err, nil)
		})

		Convey("其他管理员报错", func() {
			roles := make(map[string]map[interfaces.Role]bool)
			roles[strID1] = map[interfaces.Role]bool{interfaces.SystemRoleSecAdmin: true}
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roles, nil)

			err := departmentLogics.checkDepartInScope(visitor, &departInfo)
			assert.Equal(t, err, test401Error)
		})
	})
}

func TestSendDepartDeleted(t *testing.T) {
	Convey("sendDepartDeleted, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mg := mock.NewMockDrivenMessageBroker(ctrl)
		departmentLogics := &department{
			messageBroker: mg,
		}
		testErr := rest.NewHTTPError("error", 503000000, nil)
		content := make(map[string]interface{})
		content["id"] = strID
		Convey("DepartDeleted error", func() {
			mg.EXPECT().DepartDeleted(gomock.Any()).AnyTimes().Return(testErr)
			err := departmentLogics.sendDepartDeleted(content)
			assert.Equal(t, err, testErr)
		})
	})
}

func TestSendQuotaUpdate(t *testing.T) {
	Convey("sendOrgManagerChanged, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mg := mock.NewMockDrivenMessageBroker(ctrl)
		departmentLogics := &department{
			messageBroker: mg,
		}
		testErr := rest.NewHTTPError("error", 503000000, nil)
		content := make(map[string]interface{})
		content["ids"] = []interface{}{strID}
		Convey("OrgManagerChanged error", func() {
			mg.EXPECT().OrgManagerChanged(gomock.Any()).AnyTimes().Return(testErr)
			err := departmentLogics.sendOrgManagerChanged(content)
			assert.Equal(t, err, testErr)
		})
	})
}

//nolint:funlen
func TestDeleteDepart(t *testing.T) {
	Convey("DeleteDepart, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		dPool, txMock, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func() {
			if closeErr := dPool.Close(); closeErr != nil {
				assert.Equal(t, 1, 1)
			}
		}()

		mg := mock.NewMockDrivenMessageBroker(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		departDB := mock.NewMockDBDepartment(ctrl)
		userDB := mock.NewMockDBUser(ctrl)
		ob := mock.NewMockLogicsOutbox(ctrl)
		eacpLog := mock.NewMockDrivenEacpLog(ctrl)
		departmentLogics := &department{
			messageBroker: mg,
			db:            departDB,
			role:          role,
			userDB:        userDB,
			pool:          dPool,
			ob:            ob,
			logger:        common.NewLogger(),
			eacpLog:       eacpLog,
		}

		testErr := rest.NewHTTPError("error", 503000000, nil)
		visitor := interfaces.Visitor{}
		Convey("GetDepartmentInfo error", func() {
			departDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, testErr)
			err := departmentLogics.DeleteDepart(&visitor, strID)
			assert.Equal(t, err, testErr)
		})

		test404Error := rest.NewHTTPError("this department do not exist", rest.URINotExist, nil)
		Convey("部门不存在，则报错", func() {
			departDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
			err := departmentLogics.DeleteDepart(&visitor, strID)
			assert.Equal(t, err, test404Error)
		})

		tempDepart := interfaces.DepartmentDBInfo{}
		deprtInfos := []interfaces.DepartmentDBInfo{tempDepart}
		test401Error := rest.NewHTTPError("this user do not has the authority", errors.Forbidden, nil)
		visitor.ID = strID1
		Convey("用户无权限，则报错", func() {
			roles := make(map[string]map[interfaces.Role]bool)
			roles[strID1] = map[interfaces.Role]bool{interfaces.SystemRoleOrgAudit: true}

			departDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(deprtInfos, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roles, nil)

			err := departmentLogics.DeleteDepart(&visitor, strID)
			assert.Equal(t, err, test401Error)
		})

		Convey("获取被删除部门信息失败", func() {
			roles := make(map[string]map[interfaces.Role]bool)
			roles[strID1] = map[interfaces.Role]bool{interfaces.SystemRoleSuperAdmin: true}

			departDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(deprtInfos, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roles, nil)
			departDB.EXPECT().GetAllSubUserIDsByDepartPath(gomock.Any()).AnyTimes().Return(nil, nil)
			userDB.EXPECT().GetUsersPath(gomock.Any()).AnyTimes().Return(nil, nil)
			departDB.EXPECT().GetAllSubDepartInfosByPath(gomock.Any()).AnyTimes().Return(nil, nil)
			departDB.EXPECT().GetAllOrgManagerIDsByDepartIDs(gomock.Any()).AnyTimes().Return(nil, testErr)

			err := departmentLogics.DeleteDepart(&visitor, strID)
			assert.Equal(t, err, testErr)
		})

		Convey("pool begin error", func() {
			roles := make(map[string]map[interfaces.Role]bool)
			roles[strID1] = map[interfaces.Role]bool{interfaces.SystemRoleSuperAdmin: true}

			departDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(deprtInfos, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roles, nil)
			departDB.EXPECT().GetAllSubUserIDsByDepartPath(gomock.Any()).AnyTimes().Return(nil, nil)
			userDB.EXPECT().GetUsersPath(gomock.Any()).AnyTimes().Return(nil, nil)
			departDB.EXPECT().GetAllSubDepartInfosByPath(gomock.Any()).AnyTimes().Return(nil, nil)
			departDB.EXPECT().GetAllOrgManagerIDsByDepartIDs(gomock.Any()).AnyTimes().Return(nil, nil)
			txMock.ExpectBegin().WillReturnError(testErr)

			err := departmentLogics.DeleteDepart(&visitor, strID)
			assert.Equal(t, err, testErr)
		})

		var tempDepart11 interfaces.DepartmentDBInfo
		tempDeparts := []interfaces.DepartmentDBInfo{tempDepart11}
		Convey("删除部门的信息库获取操作报错", func() {
			roles := make(map[string]map[interfaces.Role]bool)
			roles[strID1] = map[interfaces.Role]bool{interfaces.SystemRoleSuperAdmin: true}

			departDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(deprtInfos, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roles, nil)

			departDB.EXPECT().GetAllSubUserIDsByDepartPath(gomock.Any()).AnyTimes().Return(nil, nil)
			userDB.EXPECT().GetUsersPath(gomock.Any()).AnyTimes().Return(nil, nil)
			departDB.EXPECT().GetAllSubDepartInfosByPath(gomock.Any()).AnyTimes().Return(tempDeparts, nil)
			departDB.EXPECT().GetAllOrgManagerIDsByDepartIDs(gomock.Any()).AnyTimes().Return(nil, nil)

			txMock.ExpectBegin()

			departDB.EXPECT().DeleteUserDepartRelationByPath(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departDB.EXPECT().AddUserToDepart(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departDB.EXPECT().DeleteUserOURelation(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departDB.EXPECT().DeleteDepartByPath(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departDB.EXPECT().DeleteDepartRelations(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departDB.EXPECT().DeleteDepartOURelations(gomock.Any(), gomock.Any()).AnyTimes().Return(testErr)

			txMock.ExpectRollback()

			err := departmentLogics.DeleteDepart(&visitor, strID)
			assert.Equal(t, err, testErr)
		})

		Convey("发送组织管理员变更事件错误", func() {
			roles := make(map[string]map[interfaces.Role]bool)
			roles[strID1] = map[interfaces.Role]bool{interfaces.SystemRoleSuperAdmin: true}

			departDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(deprtInfos, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roles, nil)

			departDB.EXPECT().GetAllSubUserIDsByDepartPath(gomock.Any()).AnyTimes().Return(nil, nil)
			userDB.EXPECT().GetUsersPath(gomock.Any()).AnyTimes().Return(nil, nil)
			departDB.EXPECT().GetAllSubDepartInfosByPath(gomock.Any()).AnyTimes().Return(tempDeparts, nil)
			departDB.EXPECT().GetAllOrgManagerIDsByDepartIDs(gomock.Any()).AnyTimes().Return(nil, nil)

			txMock.ExpectBegin()

			departDB.EXPECT().DeleteUserDepartRelationByPath(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departDB.EXPECT().AddUserToDepart(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departDB.EXPECT().DeleteUserOURelation(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departDB.EXPECT().DeleteDepartByPath(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departDB.EXPECT().DeleteDepartRelations(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departDB.EXPECT().DeleteDepartOURelations(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

			ob.EXPECT().AddOutboxInfo(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(testErr)

			txMock.ExpectRollback()

			err := departmentLogics.DeleteDepart(&visitor, strID)
			assert.Equal(t, err, testErr)
		})

		Convey("记录被删除日志报错", func() {
			roles := make(map[string]map[interfaces.Role]bool)
			roles[strID1] = map[interfaces.Role]bool{interfaces.SystemRoleSuperAdmin: true}

			departDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(deprtInfos, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roles, nil)

			departDB.EXPECT().GetAllSubUserIDsByDepartPath(gomock.Any()).AnyTimes().Return(nil, nil)
			userDB.EXPECT().GetUsersPath(gomock.Any()).AnyTimes().Return(nil, nil)
			departDB.EXPECT().GetAllSubDepartInfosByPath(gomock.Any()).AnyTimes().Return(tempDeparts, nil)
			departDB.EXPECT().GetAllOrgManagerIDsByDepartIDs(gomock.Any()).AnyTimes().Return(nil, nil)

			txMock.ExpectBegin()

			departDB.EXPECT().DeleteUserDepartRelationByPath(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departDB.EXPECT().AddUserToDepart(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departDB.EXPECT().DeleteUserOURelation(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departDB.EXPECT().DeleteDepartByPath(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departDB.EXPECT().DeleteDepartRelations(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departDB.EXPECT().DeleteDepartOURelations(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

			ob.EXPECT().AddOutboxInfo(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil)
			ob.EXPECT().AddOutboxInfo(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(testErr)

			txMock.ExpectRollback()

			err := departmentLogics.DeleteDepart(&visitor, strID)
			assert.Equal(t, err, testErr)
		})

		Convey("成功", func() {
			roles := make(map[string]map[interfaces.Role]bool)
			roles[strID1] = map[interfaces.Role]bool{interfaces.SystemRoleSuperAdmin: true}

			departDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(deprtInfos, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roles, nil)

			departDB.EXPECT().GetAllSubUserIDsByDepartPath(gomock.Any()).AnyTimes().Return(nil, nil)
			userDB.EXPECT().GetUsersPath(gomock.Any()).AnyTimes().Return(nil, nil)
			departDB.EXPECT().GetAllSubDepartInfosByPath(gomock.Any()).AnyTimes().Return(tempDeparts, nil)
			departDB.EXPECT().GetAllOrgManagerIDsByDepartIDs(gomock.Any()).AnyTimes().Return(nil, nil)

			txMock.ExpectBegin()

			departDB.EXPECT().DeleteUserDepartRelationByPath(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departDB.EXPECT().AddUserToDepart(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departDB.EXPECT().DeleteUserOURelation(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departDB.EXPECT().DeleteDepartByPath(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departDB.EXPECT().DeleteDepartRelations(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			departDB.EXPECT().DeleteDepartOURelations(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

			ob.EXPECT().AddOutboxInfo(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			txMock.ExpectCommit()
			ob.EXPECT().NotifyPushOutboxThread()

			err := departmentLogics.DeleteDepart(&visitor, strID)
			assert.Equal(t, err, nil)
		})
	})
}

func TestCheckNormalUserAuth(t *testing.T) {
	Convey("检查普通用户权限", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		departmentDB := mock.NewMockDBDepartment(ctrl)
		gmDB := mock.NewMockDBGroupMember(ctrl)
		userDB := mock.NewMockDBUser(ctrl)
		org := mock.NewMockLogicsOrgPerm(ctrl)

		departs := department{
			db:            departmentDB,
			userDB:        userDB,
			groupMemberDB: gmDB,
			orgPerm:       org,
		}

		testErr := rest.NewHTTPError("error", 503000000, nil)
		Convey("CheckPerms User error", func() {
			org.EXPECT().CheckPerms(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(false, testErr)
			_, _, err := departs.checkNormalUserAuth(strID, interfaces.SystemRoleNormalUser, strID)
			assert.Equal(t, err, testErr)
		})

		Convey("CheckPerms Department error", func() {
			org.EXPECT().CheckPerms(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(true, nil)
			org.EXPECT().CheckPerms(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(false, testErr)
			_, _, err := departs.checkNormalUserAuth(strID, interfaces.SystemRoleNormalUser, strID)
			assert.Equal(t, err, testErr)
		})
	})
}

func TestSendDepartDeletedAuditLog(t *testing.T) {
	Convey("sendDepartDeletedAuditLog, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mg := mock.NewMockDrivenMessageBroker(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		departDB := mock.NewMockDBDepartment(ctrl)
		userDB := mock.NewMockDBUser(ctrl)
		ob := mock.NewMockLogicsOutbox(ctrl)
		eacpLog := mock.NewMockDrivenEacpLog(ctrl)
		departmentLogics := &department{
			messageBroker: mg,
			db:            departDB,
			role:          role,
			userDB:        userDB,
			ob:            ob,
			logger:        common.NewLogger(),
			eacpLog:       eacpLog,
		}

		testErr := rest.NewHTTPError("param groupid is illegal", rest.BadRequest, nil)
		data := map[string]interface{}{
			"visitor": make(map[string]interface{}),
			"name":    strENUS,
			"root":    true,
		}
		Convey("GetDepartmentInfo error", func() {
			eacpLog.EXPECT().OpDeleteDepart(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(testErr)
			err := departmentLogics.sendDepartDeletedAuditLog(data)
			assert.Equal(t, err, testErr)
		})
	})
}

// 编写DeleteDepartManager的测试用例
func TestDeleteDepartManager(t *testing.T) {
	Convey("DeleteDepartManager, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mg := mock.NewMockDrivenMessageBroker(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		departDB := mock.NewMockDBDepartment(ctrl)
		userDB := mock.NewMockDBUser(ctrl)
		ob := mock.NewMockLogicsOutbox(ctrl)
		eacpLog := mock.NewMockDrivenEacpLog(ctrl)
		departmentLogics := &department{
			messageBroker: mg,
			db:            departDB,
			role:          role,
			userDB:        userDB,
			ob:            ob,
			logger:        common.NewLogger(),
			eacpLog:       eacpLog,
		}

		testErr := rest.NewHTTPError("param groupid is illegal", rest.BadRequest, nil)
		Convey("sql error", func() {
			departDB.EXPECT().DeleteDepartManager(gomock.Any()).Return(testErr)
			err := departmentLogics.DeleteDepartManager(strID)
			assert.Equal(t, err, testErr)
		})
	})
}

func TestHandleSearchDepartOutInfo(t *testing.T) {
	Convey("handleSearchDepartOutInfo, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mg := mock.NewMockDrivenMessageBroker(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		departDB := mock.NewMockDBDepartment(ctrl)
		userDB := mock.NewMockDBUser(ctrl)
		ob := mock.NewMockLogicsOutbox(ctrl)
		eacpLog := mock.NewMockDrivenEacpLog(ctrl)
		trace := mock.NewMockTraceClient(ctrl)
		departmentLogics := &department{
			messageBroker: mg,
			db:            departDB,
			role:          role,
			userDB:        userDB,
			ob:            ob,
			logger:        common.NewLogger(),
			eacpLog:       eacpLog,
			trace:         trace,
		}

		departDBInfo := interfaces.DepartmentDBInfo{}
		departDBInfo.ID = strID
		departDBInfo.Name = strENUS
		departDBInfo.Code = strENUS
		departDBInfo.Status = true
		departDBInfo.Remark = strENUS
		departDBInfo.Email = strENUS
		departDBInfo.ManagerID = strName2
		departDBInfo.Path = strID2 + "/" + strID

		scope := interfaces.DepartInfoScope{}
		scope.BManager = true
		scope.BParentDeps = true

		mapManagerName := make(map[string]string)
		mapManagerName[strName2] = strName1

		mapParentDep := make(map[string]interfaces.ObjectBaseInfo)
		mapParentDep[strID2] = interfaces.ObjectBaseInfo{ID: strID2, Name: strName2, Type: "department", Code: strENUS}

		Convey("success", func() {
			temp := departmentLogics.handleSearchDepartOutInfo(&departDBInfo, &scope, mapManagerName, mapParentDep)
			assert.Equal(t, temp.ID, strID)
			assert.Equal(t, temp.Name, strENUS)
			assert.Equal(t, temp.Code, strENUS)
			assert.Equal(t, temp.Enabled, true)
			assert.Equal(t, temp.Remark, strENUS)
			assert.Equal(t, temp.Email, strENUS)
			assert.Equal(t, temp.Manager.ID, strName2)
			assert.Equal(t, temp.Manager.Name, strName1)
			assert.Equal(t, len(temp.ParentDeps), 1)
			assert.Equal(t, temp.ParentDeps[0].ID, strID2)
			assert.Equal(t, temp.ParentDeps[0].Name, strName2)
			assert.Equal(t, temp.ParentDeps[0].Type, "department")
			assert.Equal(t, temp.ParentDeps[0].Code, strENUS)
		})
	})
}

func TestGetDepartInfo(t *testing.T) {
	Convey("getDepartInfo, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		defer ctrl.Finish()

		mg := mock.NewMockDrivenMessageBroker(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		departDB := mock.NewMockDBDepartment(ctrl)
		userDB := mock.NewMockDBUser(ctrl)
		ob := mock.NewMockLogicsOutbox(ctrl)
		eacpLog := mock.NewMockDrivenEacpLog(ctrl)
		trace := mock.NewMockTraceClient(ctrl)
		departmentLogics := &department{
			messageBroker: mg,
			db:            departDB,
			role:          role,
			userDB:        userDB,
			ob:            ob,
			logger:        common.NewLogger(),
			eacpLog:       eacpLog,
			trace:         trace,
		}

		testErr := rest.NewHTTPError("error", 503000000, nil)
		Convey("如果是内置管理员，SearchDeparts报错", func() {
			departDB.EXPECT().SearchDeparts(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, testErr)
			_, _, err := departmentLogics.getDepartInfo(context.Background(), nil, nil, nil, interfaces.SystemRoleSuperAdmin)
			assert.Equal(t, err, testErr)
		})

		Convey("如果是内置管理员，SearchDepartsCount报错", func() {
			departDB.EXPECT().SearchDeparts(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
			departDB.EXPECT().SearchDepartsCount(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(0, testErr)
			_, _, err := departmentLogics.getDepartInfo(context.Background(), nil, nil, nil, interfaces.SystemRoleSuperAdmin)
			assert.Equal(t, err, testErr)
		})

		visitor := interfaces.Visitor{}
		visitor.ID = strID
		Convey("如果是组织管理员，GetOrgManagerDepartInfo 报错", func() {
			userDB.EXPECT().GetOrgManagerDepartInfo(gomock.Any()).Return(nil, testErr)
			_, _, err := departmentLogics.getDepartInfo(context.Background(), &visitor, nil, nil, interfaces.SystemRoleOrgManager)
			assert.Equal(t, err, testErr)
		})

		Convey("其他管理员，报错", func() {
			_, _, err := departmentLogics.getDepartInfo(context.Background(), nil, nil, nil, interfaces.SystemRoleOrgAudit)
			assert.Equal(t, err, rest.NewHTTPErrorV2(rest.Forbidden, "this user has no authority"))
		})

		Convey("success", func() {
			departDB.EXPECT().SearchDeparts(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
			departDB.EXPECT().SearchDepartsCount(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(0, nil)
			out, data, err := departmentLogics.getDepartInfo(context.Background(), nil, nil, nil, interfaces.SystemRoleSuperAdmin)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(out), 0)
			assert.Equal(t, data, 0)
		})
	})
}

func TestOrgManagerSearchDeparts(t *testing.T) {
	Convey("orgManagerSearchDeparts, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		defer ctrl.Finish()

		mg := mock.NewMockDrivenMessageBroker(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		departDB := mock.NewMockDBDepartment(ctrl)
		userDB := mock.NewMockDBUser(ctrl)
		ob := mock.NewMockLogicsOutbox(ctrl)
		eacpLog := mock.NewMockDrivenEacpLog(ctrl)
		trace := mock.NewMockTraceClient(ctrl)
		departmentLogics := &department{
			messageBroker: mg,
			db:            departDB,
			role:          role,
			userDB:        userDB,
			ob:            ob,
			logger:        common.NewLogger(),
			eacpLog:       eacpLog,
			trace:         trace,
		}

		testErr := rest.NewHTTPError("error", 503000000, nil)
		visitor := interfaces.Visitor{}
		visitor.ID = strID
		Convey("GetOrgManagerDepartInfo error", func() {
			userDB.EXPECT().GetOrgManagerDepartInfo(gomock.Any()).Return(nil, testErr)
			_, _, err := departmentLogics.orgManagerSearchDeparts(context.Background(), &visitor, nil, nil)
			assert.Equal(t, err, testErr)
		})

		Convey("如果没有负责的部门，直接返回", func() {
			userDB.EXPECT().GetOrgManagerDepartInfo(gomock.Any()).Return(nil, nil)
			_, _, err := departmentLogics.orgManagerSearchDeparts(context.Background(), &visitor, nil, nil)
			assert.Equal(t, err, nil)
		})

		Convey("GetDepartmentInfo2 error", func() {
			userDB.EXPECT().GetOrgManagerDepartInfo(gomock.Any()).Return([]string{strID}, nil)
			departDB.EXPECT().GetDepartmentInfo2(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, testErr)
			_, _, err := departmentLogics.orgManagerSearchDeparts(context.Background(), &visitor, nil, nil)
			assert.Equal(t, err, testErr)
		})

		dep1 := interfaces.DepartmentDBInfo{}
		dep1.ID = strID
		dep1.Name = strENUS
		dep1.Code = strENUS
		dep1.Status = true
		dep1.Remark = strENUS
		dep1.Email = strENUS
		dep1.ManagerID = strName2
		dep1.Path = strID2 + "/" + strID
		Convey("GetAllSubDepartInfosByPath报错", func() {
			userDB.EXPECT().GetOrgManagerDepartInfo(gomock.Any()).Return([]string{strID}, nil)
			departDB.EXPECT().GetDepartmentInfo2(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]interfaces.DepartmentDBInfo{dep1}, nil)
			departDB.EXPECT().GetAllSubDepartInfosByPath(gomock.Any()).Return(nil, testErr)
			_, _, err := departmentLogics.orgManagerSearchDeparts(context.Background(), &visitor, nil, nil)
			assert.Equal(t, err, testErr)
		})

		Convey("SearchDeparts error", func() {
			userDB.EXPECT().GetOrgManagerDepartInfo(gomock.Any()).Return([]string{strID}, nil)
			departDB.EXPECT().GetDepartmentInfo2(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]interfaces.DepartmentDBInfo{dep1}, nil)
			departDB.EXPECT().GetAllSubDepartInfosByPath(gomock.Any()).Return([]interfaces.DepartmentDBInfo{dep1}, nil)
			departDB.EXPECT().SearchDeparts(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, testErr)
			_, _, err := departmentLogics.orgManagerSearchDeparts(context.Background(), &visitor, nil, nil)
			assert.Equal(t, err, testErr)
		})

		Convey("SearchDepartsCount error", func() {
			userDB.EXPECT().GetOrgManagerDepartInfo(gomock.Any()).Return([]string{strID}, nil)
			departDB.EXPECT().GetDepartmentInfo2(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]interfaces.DepartmentDBInfo{dep1}, nil)
			departDB.EXPECT().GetAllSubDepartInfosByPath(gomock.Any()).Return([]interfaces.DepartmentDBInfo{dep1}, nil)
			departDB.EXPECT().SearchDeparts(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
			departDB.EXPECT().SearchDepartsCount(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(0, testErr)
			_, _, err := departmentLogics.orgManagerSearchDeparts(context.Background(), &visitor, nil, nil)
			assert.Equal(t, err, testErr)
		})

		Convey("success", func() {
			userDB.EXPECT().GetOrgManagerDepartInfo(gomock.Any()).Return([]string{strID}, nil)
			departDB.EXPECT().GetDepartmentInfo2(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]interfaces.DepartmentDBInfo{dep1}, nil)
			departDB.EXPECT().GetAllSubDepartInfosByPath(gomock.Any()).Return([]interfaces.DepartmentDBInfo{dep1}, nil)
			departDB.EXPECT().SearchDeparts(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]interfaces.DepartmentDBInfo{dep1}, nil)
			departDB.EXPECT().SearchDepartsCount(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(12, nil)
			out, num, err := departmentLogics.orgManagerSearchDeparts(context.Background(), &visitor, nil, nil)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(out), 1)
			assert.Equal(t, num, 12)
			assert.Equal(t, out[0].ID, strID)
			assert.Equal(t, out[0].Name, strENUS)
			assert.Equal(t, out[0].Code, strENUS)
			assert.Equal(t, out[0].Status, true)
			assert.Equal(t, out[0].Remark, strENUS)
			assert.Equal(t, out[0].Email, strENUS)
			assert.Equal(t, out[0].ManagerID, strName2)
			assert.Equal(t, out[0].Path, strID2+"/"+strID)
		})
	})
}

func TestSearchDeparts(t *testing.T) {
	Convey("searchDeparts, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		defer ctrl.Finish()

		mg := mock.NewMockDrivenMessageBroker(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		departDB := mock.NewMockDBDepartment(ctrl)
		userDB := mock.NewMockDBUser(ctrl)
		ob := mock.NewMockLogicsOutbox(ctrl)
		eacpLog := mock.NewMockDrivenEacpLog(ctrl)
		trace := mock.NewMockTraceClient(ctrl)
		departmentLogics := &department{
			messageBroker: mg,
			db:            departDB,
			role:          role,
			userDB:        userDB,
			ob:            ob,
			logger:        common.NewLogger(),
			eacpLog:       eacpLog,
			trace:         trace,
		}

		visitor := interfaces.Visitor{}
		visitor.ID = strID

		testErr := rest.NewHTTPError("error", 503000000, nil)
		ctx := context.Background()

		Convey("GetRolesByUserIDs2 error", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).Return(nil, testErr)
			_, _, err := departmentLogics.SearchDeparts(context.Background(), &visitor, nil, nil, nil, interfaces.SystemRoleSuperAdmin)
			assert.Equal(t, err, testErr)
		})

		Convey("调用者没有传入的角色", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).Return(nil, nil)
			_, _, err := departmentLogics.SearchDeparts(context.Background(), &visitor, nil, nil, nil, interfaces.SystemRoleSuperAdmin)
			assert.Equal(t, err, rest.NewHTTPErrorV2(rest.BadRequest, "this user do not has this role"))
		})

		tempRoles := make(map[string]map[interfaces.Role]bool)
		tempRoles[strID] = make(map[interfaces.Role]bool)
		tempRoles[strID][interfaces.SystemRoleSuperAdmin] = true
		Convey("获取部门信息SearchDeparts报错", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).Return(tempRoles, nil)
			departDB.EXPECT().SearchDeparts(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, testErr)
			_, _, err := departmentLogics.SearchDeparts(context.Background(), &visitor, nil, nil, nil, interfaces.SystemRoleSuperAdmin)
			assert.Equal(t, err, testErr)
		})

		parentDep1 := interfaces.DepartmentDBInfo{}
		parentDep1.ID = "parentDep1"
		parentDep1.Name = "parentName1"
		parentDep1.Code = "parentCode1"

		parentDep2 := interfaces.DepartmentDBInfo{}
		parentDep2.ID = "parentDep2"
		parentDep2.Name = "parentName2"
		parentDep2.Code = "parentCode2"

		parentDep3 := interfaces.DepartmentDBInfo{}
		parentDep3.ID = "parentDep3"
		parentDep3.Name = "parentName3"
		parentDep3.Code = "parentCode3"

		manager1 := interfaces.UserDBInfo{}
		manager1.ID = "manager1"
		manager1.Name = strName1

		manager2 := interfaces.UserDBInfo{}
		manager2.ID = "manager2"
		manager2.Name = strName2

		dep1 := interfaces.DepartmentDBInfo{}
		dep1.ID = strID
		dep1.Name = strENUS
		dep1.Code = strENUS
		dep1.Status = false
		dep1.Remark = strENUS
		dep1.Email = strENUS
		dep1.ManagerID = manager1.ID
		dep1.Path = parentDep1.ID + "/" + parentDep2.ID + "/" + dep1.ID

		dep2 := interfaces.DepartmentDBInfo{}
		dep2.ID = strID2
		dep2.Name = strName2
		dep2.Code = strName2
		dep2.Status = true
		dep2.Remark = strName2
		dep2.Email = strName2
		dep2.ManagerID = manager2.ID
		dep2.Path = parentDep2.ID + "/" + parentDep3.ID + "/" + dep2.ID

		infos := []interfaces.DepartmentDBInfo{dep1, dep2}
		Convey("GetUserDBInfo error", func() {
			scope := interfaces.DepartInfoScope{}
			scope.BManager = true
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).Return(tempRoles, nil)
			departDB.EXPECT().SearchDeparts(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(infos, nil)
			departDB.EXPECT().SearchDepartsCount(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(2, nil)
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).Return(nil, testErr)
			_, _, err := departmentLogics.SearchDeparts(context.Background(), &visitor, &scope, nil, nil, interfaces.SystemRoleSuperAdmin)
			assert.Equal(t, err, testErr)
		})

		Convey("GetDepartmentInfo error", func() {
			scope := interfaces.DepartInfoScope{}
			scope.BManager = true
			scope.BParentDeps = true
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).Return(tempRoles, nil)
			departDB.EXPECT().SearchDeparts(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(infos, nil)
			departDB.EXPECT().SearchDepartsCount(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(2, nil)
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).Return(nil, nil)
			departDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, testErr)
			_, _, err := departmentLogics.SearchDeparts(context.Background(), &visitor, &scope, nil, nil, interfaces.SystemRoleSuperAdmin)
			assert.Equal(t, err, testErr)
		})

		Convey("success", func() {
			scope := interfaces.DepartInfoScope{}
			scope.BManager = true
			scope.BParentDeps = true
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).Return(tempRoles, nil)
			departDB.EXPECT().SearchDeparts(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(infos, nil)
			departDB.EXPECT().SearchDepartsCount(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(12, nil)
			userDB.EXPECT().GetUserDBInfo(gomock.Any()).Return([]interfaces.UserDBInfo{manager1, manager2}, nil)
			departDB.EXPECT().GetDepartmentInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]interfaces.DepartmentDBInfo{parentDep1, parentDep2, parentDep3}, nil)
			out, num, err := departmentLogics.SearchDeparts(context.Background(), &visitor, &scope, nil, nil, interfaces.SystemRoleSuperAdmin)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(out), 2)
			assert.Equal(t, num, 12)

			assert.Equal(t, out[0].ID, dep1.ID)
			assert.Equal(t, out[0].Name, dep1.Name)
			assert.Equal(t, out[0].Code, dep1.Code)
			assert.Equal(t, out[0].Enabled, dep1.Status)
			assert.Equal(t, out[0].Remark, dep1.Remark)
			assert.Equal(t, out[0].Email, dep1.Email)
			assert.Equal(t, out[0].Manager.ID, manager1.ID)
			assert.Equal(t, out[0].Manager.Name, manager1.Name)
			assert.Equal(t, len(out[0].ParentDeps), 2)
			assert.Equal(t, out[0].ParentDeps[0].ID, parentDep1.ID)
			assert.Equal(t, out[0].ParentDeps[0].Name, parentDep1.Name)
			assert.Equal(t, out[0].ParentDeps[0].Code, parentDep1.Code)
			assert.Equal(t, out[0].ParentDeps[1].ID, parentDep2.ID)
			assert.Equal(t, out[0].ParentDeps[1].Name, parentDep2.Name)
			assert.Equal(t, out[0].ParentDeps[1].Code, parentDep2.Code)

			assert.Equal(t, out[1].ID, dep2.ID)
			assert.Equal(t, out[1].Name, dep2.Name)
			assert.Equal(t, out[1].Code, dep2.Code)
			assert.Equal(t, out[1].Enabled, dep2.Status)
			assert.Equal(t, out[1].Remark, dep2.Remark)
			assert.Equal(t, out[1].Email, dep2.Email)
			assert.Equal(t, out[1].Manager.ID, manager2.ID)
			assert.Equal(t, out[1].Manager.Name, manager2.Name)
			assert.Equal(t, len(out[1].ParentDeps), 2)
			assert.Equal(t, out[1].ParentDeps[0].ID, parentDep2.ID)
			assert.Equal(t, out[1].ParentDeps[0].Name, parentDep2.Name)
			assert.Equal(t, out[1].ParentDeps[0].Code, parentDep2.Code)
			assert.Equal(t, out[1].ParentDeps[1].ID, parentDep3.ID)
			assert.Equal(t, out[1].ParentDeps[1].Name, parentDep3.Name)
			assert.Equal(t, out[1].ParentDeps[1].Code, parentDep3.Code)
		})
	})
}
