package logics

import (
	"context"
	"testing"

	gerrors "github.com/kweaver-ai/go-lib/error"
	"github.com/kweaver-ai/go-lib/rest"
	"github.com/kweaver-ai/proton-rds-sdk-go/sqlx"
	"github.com/gin-gonic/gin"
	"github.com/pborman/uuid"

	"UserManagement/common"
	uerrors "UserManagement/errors"
	"UserManagement/interfaces"
	"UserManagement/interfaces/mock"

	"github.com/go-playground/assert"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

var (
	cRSA2048 = "Zu4VRKomAqM1f82V/N8XTjFQWpxvp3ObKllfFKGNql6CTYOxgRAlosxEjFBVCMl1ArrDZJjqiebwky288LfjcpqFN" +
		"RoDrUGbcWseDpB5QJK25dxqZE/PqlOh5ZAOXqeuHODPDKikJZP4hR5bllZtP6f+jKwnUTIKsJT8erL5iwP31eiFEcJZTKPME4kg2/sAqKNn/yI" +
		"8hH4y9lGSY46Hs9rJI2c855mCg6IL7B26QMIFoJUXgHVcu2bVpUxIgpy3DhRig4TVsQFQy7FADlpFjpw4x+4B7NKscB/gEmoAzzZG3hsDYMautAx" +
		"phXHkn/3Fxkf5ft8eArIN89ZlzIobgQ=="
)

func newApp(db interfaces.DBApp, dbUser interfaces.DBUser, ob interfaces.LogicsOutbox, dbPool *sqlx.DB, h interfaces.DrivenHydra, dnEacpLog interfaces.DrivenEacpLog, role interfaces.LogicsRole) *app {
	return &app{
		db:      db,
		userDB:  dbUser,
		ob:      ob,
		hydra:   h,
		eacpLog: dnEacpLog,
		logger:  common.NewLogger(),
		pool:    dbPool,
		role:    role,
		i18n: common.NewI18n(common.I18nMap{
			i18nIDObjectsInAppNotFound: {
				interfaces.SimplifiedChinese:  "此应用账户已不存在。",
				interfaces.TraditionalChinese: "此應用賬號已不存在。",
				interfaces.AmericanEnglish:    "This application account no longer exists.",
			},
		}),
	}
}

func TestNewApp(t *testing.T) {
	Convey("NewApp", t, func() {
		sqlDB, _, err := sqlx.New()
		assert.Equal(t, err, nil)
		dbPool = sqlDB

		data := NewApp()
		assert.NotEqual(t, data, nil)
	})
}

func TestRegisterAppAuthority(t *testing.T) {
	Convey("注册用户权限相关检查", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		dbApp := mock.NewMockDBApp(ctrl)
		loOutbox := mock.NewMockLogicsOutbox(ctrl)
		userDB := mock.NewMockDBUser(ctrl)
		hydra := mock.NewMockDrivenHydra(ctrl)
		eacplog := mock.NewMockDrivenEacpLog(ctrl)
		role := mock.NewMockLogicsRole(ctrl)

		var err error
		db, txMock, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func() {
			if closeErr := db.Close(); closeErr != nil {
				assert.Equal(t, 1, 1)
			}
		}()

		app := newApp(dbApp, userDB, loOutbox, db, hydra, eacplog, role)

		common.SvcConfig.Lang = strENUS

		name := "Test"
		password := cRSA2048
		apptype := interfaces.General

		visitorInfo := &interfaces.Visitor{
			ID:        "xxx-xxx-xxx-xxx",
			IP:        "1.1.1.1",
			Mac:       "X-Request-MAC",
			UserAgent: "User-Agent",
		}

		Convey("获取用户角色报错，注册失败", func() {
			testErr := rest.NewHTTPError("xxxxx", uerrors.Forbidden, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(nil, testErr)
			_, err := app.RegisterApp(visitorInfo, name, password, apptype)

			assert.Equal(t, err, testErr)
		})

		testErr := rest.NewHTTPError("this user do not has the authority", uerrors.Forbidden, nil)
		Convey("普通用户注册通用应用账户失败", func() {
			roleInfos := make(map[string]map[interfaces.Role]bool)
			userRoles := make(map[interfaces.Role]bool)
			userRoles[interfaces.SystemRoleNormalUser] = true
			roleInfos[visitorInfo.ID] = userRoles

			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			_, err := app.RegisterApp(visitorInfo, name, password, apptype)

			assert.Equal(t, err, testErr)
		})

		Convey("三权分立下，安全管理员注册通用应用账户失败", func() {
			roleInfos := make(map[string]map[interfaces.Role]bool)
			userRoles := make(map[interfaces.Role]bool)
			userRoles[interfaces.SystemRoleSecAdmin] = true
			roleInfos[visitorInfo.ID] = userRoles

			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			_, err := app.RegisterApp(visitorInfo, name, password, apptype)

			assert.Equal(t, err, testErr)
		})

		Convey("三权分立下，审计管理员注册通用应用账户失败", func() {
			roleInfos := make(map[string]map[interfaces.Role]bool)
			userRoles := make(map[interfaces.Role]bool)
			userRoles[interfaces.SystemRoleAuditAdmin] = true
			roleInfos[visitorInfo.ID] = userRoles

			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			_, err := app.RegisterApp(visitorInfo, name, password, apptype)

			assert.Equal(t, err, testErr)
		})

		Convey("三权分立下，系统管理员注册通用应用账户成功", func() {
			roleInfos := make(map[string]map[interfaces.Role]bool)
			userRoles := make(map[interfaces.Role]bool)
			userRoles[interfaces.SystemRoleSysAdmin] = true
			roleInfos[visitorInfo.ID] = userRoles

			txMock.ExpectBegin()
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			dbApp.EXPECT().GetAppByName(gomock.Any()).AnyTimes().Return(nil, nil)
			userDB.EXPECT().CheckNameExist(gomock.Any()).AnyTimes().Return(false, nil)
			hydra.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any()).Return("client_id", nil)
			dbApp.EXPECT().RegisterApp(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			loOutbox.EXPECT().AddOutboxInfo(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			txMock.ExpectCommit()
			loOutbox.EXPECT().NotifyPushOutboxThread()
			_, err := app.RegisterApp(visitorInfo, name, password, apptype)

			assert.Equal(t, err, nil)
			assert.Equal(t, 1, 1)
		})

		Convey("超级管理员注册通用应用账户成功1", func() {
			roleInfos := make(map[string]map[interfaces.Role]bool)
			userRoles := make(map[interfaces.Role]bool)
			userRoles[interfaces.SystemRoleSuperAdmin] = true
			roleInfos[visitorInfo.ID] = userRoles

			txMock.ExpectBegin()
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			dbApp.EXPECT().GetAppByName(gomock.Any()).AnyTimes().Return(nil, nil)
			userDB.EXPECT().CheckNameExist(gomock.Any()).AnyTimes().Return(false, nil)
			hydra.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any()).Return("client_id", nil)
			dbApp.EXPECT().RegisterApp(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			loOutbox.EXPECT().AddOutboxInfo(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			txMock.ExpectCommit()
			loOutbox.EXPECT().NotifyPushOutboxThread()
			_, err := app.RegisterApp(visitorInfo, name, password, apptype)

			assert.Equal(t, err, nil)
		})

		Convey("超级管理员注册通用应用账户成功", func() {
			roleInfos := make(map[string]map[interfaces.Role]bool)
			userRoles := make(map[interfaces.Role]bool)
			userRoles[interfaces.SystemRoleSuperAdmin] = true
			roleInfos[visitorInfo.ID] = userRoles

			txMock.ExpectBegin()
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			dbApp.EXPECT().GetAppByName(gomock.Any()).AnyTimes().Return(nil, nil)
			userDB.EXPECT().CheckNameExist(gomock.Any()).AnyTimes().Return(false, nil)
			hydra.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any()).Return("client_id", nil)
			dbApp.EXPECT().RegisterApp(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			loOutbox.EXPECT().AddOutboxInfo(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			txMock.ExpectCommit()
			loOutbox.EXPECT().NotifyPushOutboxThread()
			_, err := app.RegisterApp(visitorInfo, name, "", apptype)

			assert.Equal(t, err, nil)
		})
	})
}

func TestRegisterApp(t *testing.T) {
	Convey("RegisterApp, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		dbApp := mock.NewMockDBApp(ctrl)
		userDB := mock.NewMockDBUser(ctrl)
		hydra := mock.NewMockDrivenHydra(ctrl)
		eacplog := mock.NewMockDrivenEacpLog(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		loOutbox := mock.NewMockLogicsOutbox(ctrl)

		var err error
		db, txMock, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func() {
			if closeErr := db.Close(); closeErr != nil {
				assert.Equal(t, 1, 1)
			}
		}()

		app := newApp(dbApp, userDB, loOutbox, db, hydra, eacplog, role)

		common.SvcConfig.Lang = strENUS

		name := "Test"
		password := cRSA2048
		apptype := interfaces.General

		visitorInfo := &interfaces.Visitor{
			ID:        "xxx-xxx-xxx-xxx",
			IP:        "1.1.1.1",
			Mac:       "X-Request-MAC",
			UserAgent: "User-Agent",
		}

		txMock.ExpectBegin()

		roleInfos := make(map[string]map[interfaces.Role]bool)
		userRoles := make(map[interfaces.Role]bool)
		userRoles[interfaces.SystemRoleSuperAdmin] = true
		roleInfos[visitorInfo.ID] = userRoles

		Convey("外部接口密码未加密，报错", func() {
			_, err := app.RegisterApp(visitorInfo, name, "test1111", apptype)

			assert.Equal(t, err, rest.NewHTTPError("crypto/rsa: decryption error", rest.BadRequest, nil))
		})

		Convey("GetApp, db is available", func() {
			tmp := &interfaces.AppInfo{}
			testErr := rest.NewHTTPError("error", 503000000, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			dbApp.EXPECT().GetAppByName(gomock.Any()).AnyTimes().Return(tmp, testErr)
			_, err := app.RegisterApp(visitorInfo, name, password, apptype)

			assert.Equal(t, err, testErr)
		})

		Convey("GetApp, name already exists", func() {
			tmp := &interfaces.AppInfo{
				ID:   "test2",
				Name: "test3",
			}

			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			dbApp.EXPECT().GetAppByName(gomock.Any()).AnyTimes().Return(tmp, nil)
			_, err := app.RegisterApp(visitorInfo, name, password, apptype)

			testErr1 := rest.NewHTTPErrorV2(uerrors.Conflict, "name already exists",
				rest.SetDetail(map[string]interface{}{"type": "app", "id": "test2"}),
				rest.SetCodeStr(uerrors.StrConflictApp))

			assert.Equal(t, err, testErr1)
		})

		Convey("register app failed", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)

			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			dbApp.EXPECT().GetAppByName(gomock.Any()).AnyTimes().Return(nil, nil)
			userDB.EXPECT().CheckNameExist(gomock.Any()).AnyTimes().Return(false, nil)
			hydra.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return("client_id", nil)
			dbApp.EXPECT().RegisterApp(gomock.Any(), gomock.Any()).AnyTimes().Return(testErr)
			txMock.ExpectRollback()
			_, err := app.RegisterApp(visitorInfo, name, password, apptype)

			assert.Equal(t, err, testErr)
		})

		Convey("AddOutboxInfo failed", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)

			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			dbApp.EXPECT().GetAppByName(gomock.Any()).AnyTimes().Return(nil, nil)
			userDB.EXPECT().CheckNameExist(gomock.Any()).AnyTimes().Return(false, nil)
			hydra.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return("client_id", nil)
			dbApp.EXPECT().RegisterApp(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			loOutbox.EXPECT().AddOutboxInfo(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(testErr)
			txMock.ExpectRollback()
			_, err := app.RegisterApp(visitorInfo, name, password, apptype)

			assert.Equal(t, err, testErr)
		})

		Convey("register success", func() {
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			dbApp.EXPECT().GetAppByName(gomock.Any()).AnyTimes().Return(nil, nil)
			userDB.EXPECT().CheckNameExist(gomock.Any()).AnyTimes().Return(false, nil)
			hydra.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return("client_id", nil)
			dbApp.EXPECT().RegisterApp(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			loOutbox.EXPECT().AddOutboxInfo(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			txMock.ExpectCommit()
			loOutbox.EXPECT().NotifyPushOutboxThread()
			_, err := app.RegisterApp(visitorInfo, name, password, apptype)

			assert.Equal(t, err, nil)
		})
	})
}

func TestDeleteAppAuthority(t *testing.T) {
	Convey("删除应用账户权限相关", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		dbApp := mock.NewMockDBApp(ctrl)
		loOutbox := mock.NewMockLogicsOutbox(ctrl)
		userDB := mock.NewMockDBUser(ctrl)
		eacplog := mock.NewMockDrivenEacpLog(ctrl)
		role := mock.NewMockLogicsRole(ctrl)

		var err error
		db, txMock, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func() {
			if closeErr := db.Close(); closeErr != nil {
				assert.Equal(t, 1, 1)
			}
		}()

		app := newApp(dbApp, userDB, loOutbox, db, nil, eacplog, role)

		id := uuid.New()

		visitorInfo := &interfaces.Visitor{
			ID:        "xxx-xxx-xxx-xxx",
			IP:        "1.1.1.1",
			Mac:       "X-Request-MAC",
			UserAgent: "User-Agent",
		}

		appInfo := &interfaces.AppInfo{
			ID:   "xxx-xxx-xxx",
			Name: "test-name",
		}

		Convey("获取用户角色，删除失败", func() {
			testErr := rest.NewHTTPError("xxx", uerrors.Forbidden, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(nil, testErr)

			err := app.DeleteApp(visitorInfo, id)

			assert.Equal(t, err, testErr)
		})

		testErr := rest.NewHTTPError("this user do not has the authority", uerrors.Forbidden, nil)
		Convey("普通用户删除应用账户失败", func() {
			roleInfo := make(map[string]map[interfaces.Role]bool)
			userRoles := make(map[interfaces.Role]bool)
			userRoles[interfaces.SystemRoleNormalUser] = true
			roleInfo[visitorInfo.ID] = userRoles
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfo, nil)

			err := app.DeleteApp(visitorInfo, id)

			assert.Equal(t, err, testErr)
		})

		Convey("三权分立下，安全管理员删除应用账户失败", func() {
			roleInfos := make(map[string]map[interfaces.Role]bool)
			userRoles := make(map[interfaces.Role]bool)
			userRoles[interfaces.SystemRoleSecAdmin] = true
			roleInfos[visitorInfo.ID] = userRoles
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)

			err := app.DeleteApp(visitorInfo, id)

			assert.Equal(t, err, testErr)
		})

		Convey("三权分立下，审计管理员删除应用账户失败", func() {
			roleInfos := make(map[string]map[interfaces.Role]bool)
			userRoles := make(map[interfaces.Role]bool)
			userRoles[interfaces.SystemRoleAuditAdmin] = true
			roleInfos[visitorInfo.ID] = userRoles
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)

			err := app.DeleteApp(visitorInfo, id)

			assert.Equal(t, err, testErr)
		})

		Convey("三权分离下，系统管理员删除应用账户成功", func() {
			roleInfos := make(map[string]map[interfaces.Role]bool)
			userRoles := make(map[interfaces.Role]bool)
			userRoles[interfaces.SystemRoleSysAdmin] = true
			roleInfos[visitorInfo.ID] = userRoles

			txMock.ExpectBegin()
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			dbApp.EXPECT().GetAppByID(gomock.Any()).AnyTimes().Return(appInfo, nil)
			loOutbox.EXPECT().AddOutboxInfo(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			dbApp.EXPECT().DeleteApp(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			txMock.ExpectCommit()
			loOutbox.EXPECT().NotifyPushOutboxThread()
			eacplog.EXPECT().EacpLog(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			err := app.DeleteApp(visitorInfo, id)

			assert.Equal(t, err, nil)
			assert.Equal(t, 1, 1)
		})

		Convey("超级管理员删除应用账户成功", func() {
			roleInfos := make(map[string]map[interfaces.Role]bool)
			userRoles := make(map[interfaces.Role]bool)
			userRoles[interfaces.SystemRoleSuperAdmin] = true
			roleInfos[visitorInfo.ID] = userRoles

			txMock.ExpectBegin()
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			dbApp.EXPECT().GetAppByID(gomock.Any()).AnyTimes().Return(appInfo, nil)
			loOutbox.EXPECT().AddOutboxInfo(gomock.Any(), gomock.Any(), gomock.Any()).Times(2).Return(nil)
			dbApp.EXPECT().DeleteApp(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			loOutbox.EXPECT().AddOutboxInfo(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil)
			txMock.ExpectCommit()
			loOutbox.EXPECT().NotifyPushOutboxThread()
			err := app.DeleteApp(visitorInfo, id)

			assert.Equal(t, err, nil)
		})
	})
}

func TestDeleteApp(t *testing.T) {
	Convey("DeleteApp, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		dbApp := mock.NewMockDBApp(ctrl)
		loOutbox := mock.NewMockLogicsOutbox(ctrl)
		userDB := mock.NewMockDBUser(ctrl)
		eacplog := mock.NewMockDrivenEacpLog(ctrl)
		role := mock.NewMockLogicsRole(ctrl)

		var err error
		db, txMock, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func() {
			if closeErr := db.Close(); closeErr != nil {
				assert.Equal(t, 1, 1)
			}
		}()

		app := newApp(dbApp, userDB, loOutbox, db, nil, eacplog, role)

		id := uuid.New()

		visitorInfo := &interfaces.Visitor{
			ID:        "xxx-xxx-xxx-xxx",
			IP:        "1.1.1.1",
			Mac:       "X-Request-MAC",
			UserAgent: "User-Agent",
		}

		appInfo := &interfaces.AppInfo{
			ID:   "xxx-xxx-xxx",
			Name: "test-name",
		}

		roleInfos := make(map[string]map[interfaces.Role]bool)
		userRoles := make(map[interfaces.Role]bool)
		userRoles[interfaces.SystemRoleSuperAdmin] = true
		roleInfos[visitorInfo.ID] = userRoles

		txMock.ExpectBegin()

		Convey("db delete failed", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)

			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			dbApp.EXPECT().GetAppByID(gomock.Any()).AnyTimes().Return(appInfo, nil)
			loOutbox.EXPECT().AddOutboxInfo(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			dbApp.EXPECT().DeleteApp(gomock.Any(), gomock.Any()).AnyTimes().Return(testErr)
			eacplog.EXPECT().EacpLog(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			err := app.DeleteApp(visitorInfo, id)

			assert.Equal(t, err, testErr)
		})

		Convey("delete success", func() {
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			dbApp.EXPECT().GetAppByID(gomock.Any()).AnyTimes().Return(appInfo, nil)
			loOutbox.EXPECT().AddOutboxInfo(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			dbApp.EXPECT().DeleteApp(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			txMock.ExpectCommit()
			loOutbox.EXPECT().NotifyPushOutboxThread()
			eacplog.EXPECT().EacpLog(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			err := app.DeleteApp(visitorInfo, id)

			assert.Equal(t, err, nil)
		})
	})
}

func TestUpdateAppAuthority(t *testing.T) {
	Convey("更新应用账户权限相关", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		dbApp := mock.NewMockDBApp(ctrl)
		loOutbox := mock.NewMockLogicsOutbox(ctrl)
		userDB := mock.NewMockDBUser(ctrl)
		eacplog := mock.NewMockDrivenEacpLog(ctrl)
		role := mock.NewMockLogicsRole(ctrl)

		var err error
		db, txMock, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func() {
			if closeErr := db.Close(); closeErr != nil {
				assert.Equal(t, 1, 1)
			}
		}()

		app := newApp(dbApp, userDB, loOutbox, db, nil, eacplog, role)

		id := uuid.New()
		pwd := cRSA2048

		appInfo := &interfaces.AppInfo{
			ID:             "xxx-xxx-xxx-xxx",
			Name:           "ceshi",
			CredentialType: interfaces.CredentialTypePassword,
		}

		visitorInfo := &interfaces.Visitor{
			ID:        "xxx-xxx-xxx-xxx",
			IP:        "1.1.1.1",
			Mac:       "X-Request-MAC",
			UserAgent: "User-Agent",
		}

		testErr := rest.NewHTTPError("ccccc", uerrors.Forbidden, nil)
		Convey("获取用户角色失败，更新账户失败", func() {
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(nil, testErr)
			err := app.UpdateApp(visitorInfo, id, true, "test1", true, pwd)

			assert.Equal(t, err, testErr)
		})

		testErr = rest.NewHTTPError("this user do not has the authority", uerrors.Forbidden, nil)
		Convey("三权分立下，普通用户更新应用账户失败", func() {
			roleInfos := make(map[string]map[interfaces.Role]bool)
			userRole := make(map[interfaces.Role]bool)
			userRole[interfaces.SystemRoleNormalUser] = true
			roleInfos[id] = userRole
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)

			err := app.UpdateApp(visitorInfo, id, true, "test1", true, pwd)

			assert.Equal(t, err, testErr)
		})

		Convey("三权分立下，安全管理员更新应用账户失败", func() {
			roleInfos := make(map[string]map[interfaces.Role]bool)
			userRole := make(map[interfaces.Role]bool)
			userRole[interfaces.SystemRoleSecAdmin] = true
			roleInfos[id] = userRole
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)

			err := app.UpdateApp(visitorInfo, id, true, "test1", true, pwd)

			assert.Equal(t, err, testErr)
		})

		Convey("三权分立下，审计管理员更新应用账户失败", func() {
			roleInfos := make(map[string]map[interfaces.Role]bool)
			userRole := make(map[interfaces.Role]bool)
			userRole[interfaces.SystemRoleAuditAdmin] = true
			roleInfos[id] = userRole
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)

			err := app.UpdateApp(visitorInfo, id, true, "test1", true, pwd)

			assert.Equal(t, err, testErr)
		})

		Convey("三权分立下，系统管理员更新应用账户成功", func() {
			roleInfos := make(map[string]map[interfaces.Role]bool)
			userRole := make(map[interfaces.Role]bool)
			userRole[interfaces.SystemRoleSysAdmin] = true
			roleInfos[visitorInfo.ID] = userRole

			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			txMock.ExpectBegin()
			dbApp.EXPECT().GetAppByID(gomock.Any()).AnyTimes().Return(appInfo, nil)
			dbApp.EXPECT().GetAppByName(gomock.Any()).AnyTimes().Return(nil, nil)
			userDB.EXPECT().CheckNameExist(gomock.Any()).AnyTimes().Return(false, nil)
			loOutbox.EXPECT().AddOutboxInfo(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			dbApp.EXPECT().UpdateApp(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			txMock.ExpectCommit()
			loOutbox.EXPECT().NotifyPushOutboxThread()
			eacplog.EXPECT().EacpLog(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			err := app.UpdateApp(visitorInfo, id, true, "test1", true, pwd)

			assert.Equal(t, err, nil)
			assert.Equal(t, 1, 1)
		})

		Convey("超级管理员更新应用账户成功", func() {
			roleInfos := make(map[string]map[interfaces.Role]bool)
			userRole := make(map[interfaces.Role]bool)
			userRole[interfaces.SystemRoleSuperAdmin] = true
			roleInfos[visitorInfo.ID] = userRole

			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			txMock.ExpectBegin()
			dbApp.EXPECT().GetAppByID(gomock.Any()).AnyTimes().Return(appInfo, nil)
			dbApp.EXPECT().GetAppByName(gomock.Any()).AnyTimes().Return(nil, nil)
			userDB.EXPECT().CheckNameExist(gomock.Any()).AnyTimes().Return(false, nil)
			loOutbox.EXPECT().AddOutboxInfo(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			dbApp.EXPECT().UpdateApp(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			txMock.ExpectCommit()
			loOutbox.EXPECT().NotifyPushOutboxThread()
			eacplog.EXPECT().EacpLog(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			err := app.UpdateApp(visitorInfo, id, true, "test1", true, pwd)

			assert.Equal(t, err, nil)
		})
	})
}

func TestUpdateApp(t *testing.T) {
	Convey("UpdateApp, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		dbApp := mock.NewMockDBApp(ctrl)
		loOutbox := mock.NewMockLogicsOutbox(ctrl)
		userDB := mock.NewMockDBUser(ctrl)
		eacplog := mock.NewMockDrivenEacpLog(ctrl)
		role := mock.NewMockLogicsRole(ctrl)

		var err error
		db, txMock, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func() {
			if closeErr := db.Close(); closeErr != nil {
				assert.Equal(t, 1, 1)
			}
		}()

		app := newApp(dbApp, userDB, loOutbox, db, nil, eacplog, role)

		id := uuid.New()
		name := "ceshi"
		pwd := cRSA2048

		appInfo := &interfaces.AppInfo{
			ID:             "xxx-xxx-xxx-xxx",
			Name:           "ceshi",
			CredentialType: interfaces.CredentialTypePassword,
		}

		visitorInfo := &interfaces.Visitor{
			ID:        "xxx-xxx-xxx-xxx",
			IP:        "1.1.1.1",
			Mac:       "X-Request-MAC",
			UserAgent: "User-Agent",
		}

		roleInfos := make(map[string]map[interfaces.Role]bool)
		userRole := make(map[interfaces.Role]bool)
		userRole[interfaces.SystemRoleSuperAdmin] = true
		roleInfos[visitorInfo.ID] = userRole

		txMock.ExpectBegin()

		Convey("密码未加密，报错", func() {
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			dbApp.EXPECT().GetAppByID(gomock.Any()).AnyTimes().Return(nil, nil)
			err := app.UpdateApp(visitorInfo, id, true, name, true, "xxxxxx")

			assert.Equal(t, err, rest.NewHTTPError("illegal base64 data at input byte 4", rest.BadRequest, nil))
		})

		Convey("id does not exist", func() {
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			dbApp.EXPECT().GetAppByID(gomock.Any()).AnyTimes().Return(nil, nil)
			err := app.UpdateApp(visitorInfo, id, true, name, true, pwd)

			assert.Equal(t, err, rest.NewHTTPErrorV2(uerrors.NotFound, "id does not exist", rest.SetCodeStr(uerrors.StrNotFoundAppNotFound)))
		})

		Convey("name already exists", func() {
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			dbApp.EXPECT().GetAppByID(gomock.Any()).AnyTimes().Return(appInfo, nil)
			dbApp.EXPECT().GetAppByName(gomock.Any()).AnyTimes().Return(appInfo, nil)
			err := app.UpdateApp(visitorInfo, id, true, "test1", true, pwd)

			testErr := rest.NewHTTPErrorV2(uerrors.Conflict, "name already exists",
				rest.SetDetail(map[string]interface{}{"type": "app", "id": appInfo.ID}),
				rest.SetCodeStr(uerrors.StrConflictApp))
			assert.Equal(t, err, testErr)
		})

		Convey("db update failed", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			dbApp.EXPECT().GetAppByID(gomock.Any()).AnyTimes().Return(appInfo, nil)
			dbApp.EXPECT().GetAppByName(gomock.Any()).AnyTimes().Return(nil, nil)
			userDB.EXPECT().CheckNameExist(gomock.Any()).AnyTimes().Return(false, nil)
			loOutbox.EXPECT().AddOutboxInfo(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			dbApp.EXPECT().UpdateApp(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(testErr)
			txMock.ExpectRollback()
			err := app.UpdateApp(visitorInfo, id, true, "test1", true, pwd)

			assert.Equal(t, err, testErr)
		})

		Convey("name包含数字、中英文、符号，创建成功", func() {
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			dbApp.EXPECT().GetAppByID(gomock.Any()).AnyTimes().Return(appInfo, nil)
			dbApp.EXPECT().GetAppByName(gomock.Any()).AnyTimes().Return(nil, nil)
			userDB.EXPECT().CheckNameExist(gomock.Any()).AnyTimes().Return(false, nil)
			loOutbox.EXPECT().AddOutboxInfo(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			dbApp.EXPECT().UpdateApp(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			txMock.ExpectCommit()
			loOutbox.EXPECT().NotifyPushOutboxThread()
			eacplog.EXPECT().EacpLog(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

			testName := "(σﾟ∀ﾟ)σ..☆哎哟不错哦❤haha666"
			err := app.UpdateApp(visitorInfo, id, true, testName, true, pwd)

			assert.Equal(t, err, nil)
		})

		Convey("令牌类应用账户禁止处理password", func() {
			appInfo.CredentialType = interfaces.CredentialTypeToken
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			dbApp.EXPECT().GetAppByID(gomock.Any()).AnyTimes().Return(appInfo, nil)

			testName := "(σﾟ∀ﾟ)σ..☆哎哟不错哦❤haha666"
			err := app.UpdateApp(visitorInfo, id, true, testName, true, pwd)

			assert.Equal(t, err, rest.NewHTTPErrorV2(rest.BadRequest, "app credential type is token, password cannot be updated"))
		})
	})
}

func TestCheckPassword(t *testing.T) {
	Convey("密码相关检查", t, func() {
		app := newApp(dbApp, nil, nil, nil, nil, nil, nil)
		Convey("password长度小于6，抛错", func() {
			testPwd := "xx"
			err := app.checkPassword(testPwd)

			assert.Equal(t, err, rest.NewHTTPError("param password is illegal", rest.BadRequest, nil))
		})

		Convey("password长度大于100，抛错", func() {
			testPwd := "123333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333"
			err := app.checkPassword(testPwd)

			assert.Equal(t, err, rest.NewHTTPError("param password is illegal", rest.BadRequest, nil))
		})

		Convey("password中包含中文，抛错", func() {
			testPwd := "包含中文的密码"
			err := app.checkPassword(testPwd)

			assert.Equal(t, err, rest.NewHTTPError("param password is illegal", rest.BadRequest, nil))
		})

		Convey("password为空字符串/None，抛错", func() {
			testPwd := ""
			err := app.checkPassword(testPwd)

			assert.Equal(t, err, rest.NewHTTPError("param password is illegal", rest.BadRequest, nil))
		})
	})
}

func TestCheckName(t *testing.T) {
	Convey("名称相关检查", t, func() {
		app := newApp(dbApp, nil, nil, nil, nil, nil, nil)
		Convey("name长度超过128个字符，抛错", func() {
			testName := "test1111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111"
			err := app.checkName(testName)

			assert.Equal(t, err, rest.NewHTTPError("param name is illegal", rest.BadRequest, nil))
		})

		Convey("name包含特殊字符，\\ / : * ? \" < > |，抛错", func() {
			testName := "name:*?"
			err := app.checkName(testName)

			assert.Equal(t, err, rest.NewHTTPError("param name is illegal", rest.BadRequest, nil))
		})

		Convey("name前包含空格，抛错", func() {
			testName := " namexxxx"
			err := app.checkName(testName)

			assert.Equal(t, err, rest.NewHTTPError("param name is illegal", rest.BadRequest, nil))
		})

		Convey("name中包含空格，抛错", func() {
			testName := "xxx name"
			err := app.checkName(testName)

			assert.Equal(t, err, rest.NewHTTPError("param name is illegal", rest.BadRequest, nil))
		})

		Convey("name后包含空格，抛错", func() {
			testName := "xxxname "
			err := app.checkName(testName)

			assert.Equal(t, err, rest.NewHTTPError("param name is illegal", rest.BadRequest, nil))
		})

		Convey("name为空字符串/None，抛错", func() {
			testName := ""
			err := app.checkName(testName)

			assert.Equal(t, err, rest.NewHTTPError("param name is illegal", rest.BadRequest, nil))
		})
	})
}

func TestAppListAuthority(t *testing.T) {
	Convey("获取应用账户列表权限相关", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		dbApp := mock.NewMockDBApp(ctrl)
		loOutbox := mock.NewMockLogicsOutbox(ctrl)
		userDB := mock.NewMockDBUser(ctrl)
		eacplog := mock.NewMockDrivenEacpLog(ctrl)
		role := mock.NewMockLogicsRole(ctrl)

		var err error
		db, _, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func() {
			if closeErr := db.Close(); closeErr != nil {
				assert.Equal(t, 1, 1)
			}
		}()

		app := newApp(dbApp, userDB, loOutbox, db, nil, eacplog, role)

		searchInfo := &interfaces.SearchInfo{
			Direction: interfaces.Asc,
			Sort:      interfaces.DateCreated,
			Offset:    0,
			Limit:     10,
		}

		info := &[]interfaces.AppInfo{
			{
				ID:   "test1",
				Name: "test1",
			},
			{
				ID:   "test2",
				Name: "test2",
			},
		}

		var visitor interfaces.Visitor
		Convey("超级管理员获取应用账户列表成功", func() {
			roleInfos := make(map[string]map[interfaces.Role]bool)
			userRole := make(map[interfaces.Role]bool)
			userRole[interfaces.SystemRoleSuperAdmin] = true
			roleInfos[visitor.ID] = userRole

			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			dbApp.EXPECT().AppListCount(gomock.Any()).AnyTimes().Return(10, nil)
			dbApp.EXPECT().AppList(gomock.Any()).AnyTimes().Return(info, nil)
			_, _, err := app.AppList(&visitor, searchInfo)

			assert.Equal(t, err, nil)
		})

		Convey("三权分离下，系统管理员获取应用账户列表成功", func() {
			roleInfos := make(map[string]map[interfaces.Role]bool)
			userRole := make(map[interfaces.Role]bool)
			userRole[interfaces.SystemRoleSysAdmin] = true
			roleInfos[visitor.ID] = userRole

			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			dbApp.EXPECT().AppListCount(gomock.Any()).AnyTimes().Return(10, nil)
			dbApp.EXPECT().AppList(gomock.Any()).AnyTimes().Return(info, nil)
			_, _, err := app.AppList(&visitor, searchInfo)

			assert.Equal(t, err, nil)
		})

		Convey("三权分立下，安全管理员获取应用账户列表成功", func() {
			roleInfos := make(map[string]map[interfaces.Role]bool)
			userRole := make(map[interfaces.Role]bool)
			userRole[interfaces.SystemRoleSecAdmin] = true
			roleInfos[visitor.ID] = userRole

			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			dbApp.EXPECT().AppListCount(gomock.Any()).AnyTimes().Return(10, nil)
			dbApp.EXPECT().AppList(gomock.Any()).AnyTimes().Return(info, nil)
			_, _, err := app.AppList(&visitor, searchInfo)

			assert.Equal(t, err, nil)
		})

		Convey("三权分立下，审计管理员获取应用账户列表成功", func() {
			roleInfos := make(map[string]map[interfaces.Role]bool)
			userRole := make(map[interfaces.Role]bool)
			userRole[interfaces.SystemRoleAuditAdmin] = true
			roleInfos[visitor.ID] = userRole

			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			dbApp.EXPECT().AppListCount(gomock.Any()).AnyTimes().Return(10, nil)
			dbApp.EXPECT().AppList(gomock.Any()).AnyTimes().Return(info, nil)
			_, _, err := app.AppList(&visitor, searchInfo)

			assert.Equal(t, err, nil)
		})

		testErr := rest.NewHTTPError("this user do not has the authority", uerrors.Forbidden, nil)
		Convey("普通管理员获取应用账户列表失败", func() {
			roleInfos := make(map[string]map[interfaces.Role]bool)
			userRole := make(map[interfaces.Role]bool)
			userRole[interfaces.SystemRoleNormalUser] = true
			roleInfos[visitor.ID] = userRole

			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			_, _, err := app.AppList(&visitor, searchInfo)

			assert.Equal(t, err, testErr)
		})

		testErr = rest.NewHTTPError("xxxx", uerrors.Forbidden, nil)
		Convey("获取用户角色信息失败，报错", func() {
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(nil, testErr)
			_, _, err := app.AppList(&visitor, searchInfo)

			assert.Equal(t, err, testErr)
		})
	})
}

func TestAppList(t *testing.T) {
	Convey("AppList, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		dbApp := mock.NewMockDBApp(ctrl)
		loOutbox := mock.NewMockLogicsOutbox(ctrl)
		userDB := mock.NewMockDBUser(ctrl)
		eacplog := mock.NewMockDrivenEacpLog(ctrl)
		role := mock.NewMockLogicsRole(ctrl)

		var err error
		db, _, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func() {
			if closeErr := db.Close(); closeErr != nil {
				assert.Equal(t, 1, 1)
			}
		}()

		app := newApp(dbApp, userDB, loOutbox, db, nil, eacplog, role)

		searchInfo := &interfaces.SearchInfo{
			Direction: interfaces.Asc,
			Sort:      interfaces.DateCreated,
			Offset:    0,
			Limit:     10,
		}

		visitor := interfaces.Visitor{
			ID: "zzz",
		}

		roleInfos := make(map[string]map[interfaces.Role]bool)
		userRole := make(map[interfaces.Role]bool)
		userRole[interfaces.SystemRoleSuperAdmin] = true
		roleInfos[visitor.ID] = userRole

		Convey("appListCount failed", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			dbApp.EXPECT().AppListCount(gomock.Any()).AnyTimes().Return(0, testErr)
			_, _, err := app.AppList(&visitor, searchInfo)

			assert.Equal(t, err, testErr)
		})

		Convey("applist failed", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			dbApp.EXPECT().AppListCount(gomock.Any()).AnyTimes().Return(10, nil)
			dbApp.EXPECT().AppList(gomock.Any()).AnyTimes().Return(nil, testErr)
			_, _, err := app.AppList(&visitor, searchInfo)

			assert.Equal(t, err, testErr)
		})

		Convey("applist success", func() {
			info := &[]interfaces.AppInfo{
				{
					ID:   "test1",
					Name: "test1",
				},
				{
					ID:   "test2",
					Name: "test2",
				},
			}

			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(roleInfos, nil)
			dbApp.EXPECT().AppListCount(gomock.Any()).AnyTimes().Return(10, nil)
			dbApp.EXPECT().AppList(gomock.Any()).AnyTimes().Return(info, nil)
			_, _, err := app.AppList(&visitor, searchInfo)

			assert.Equal(t, err, nil)
		})
	})
}

func TestGetApp(t *testing.T) {
	Convey("GetApp, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		dbApp := mock.NewMockDBApp(ctrl)
		loOutbox := mock.NewMockLogicsOutbox(ctrl)
		userDB := mock.NewMockDBUser(ctrl)
		eacplog := mock.NewMockDrivenEacpLog(ctrl)

		var err error
		db, _, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func() {
			if closeErr := db.Close(); closeErr != nil {
				assert.Equal(t, 1, 1)
			}
		}()

		app := newApp(dbApp, userDB, loOutbox, db, nil, eacplog, nil)

		id := "xxx-xxx-xxx-xxx"
		Convey("id does not exist", func() {
			dbApp.EXPECT().GetAppByID(gomock.Any()).AnyTimes().Return(nil, nil)
			_, err := app.GetApp(id)

			assert.Equal(t, err, rest.NewHTTPErrorV2(uerrors.NotFound, "id does not exist", rest.SetCodeStr(uerrors.StrNotFoundAppNotFound)))
		})

		Convey("get app success", func() {
			appInfo := &interfaces.AppInfo{
				ID:   "test",
				Name: "test",
			}
			dbApp.EXPECT().GetAppByID(gomock.Any()).AnyTimes().Return(appInfo, nil)
			_, err := app.GetApp(id)

			assert.Equal(t, err, nil)
		})
	})
}

func TestConvertAppName(t *testing.T) {
	Convey("ConvertAppName, db is available", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		dbApp := mock.NewMockDBApp(ctrl)
		loOutbox := mock.NewMockLogicsOutbox(ctrl)
		userDB := mock.NewMockDBUser(ctrl)
		eacplog := mock.NewMockDrivenEacpLog(ctrl)

		var err error
		db, _, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func() {
			if closeErr := db.Close(); closeErr != nil {
				assert.Equal(t, 1, 1)
			}
		}()

		app := newApp(dbApp, userDB, loOutbox, db, nil, eacplog, nil)

		Convey(" ids empty", func() {
			outInfo, err := app.ConvertAppName(make([]string, 0), false, true)

			assert.Equal(t, err, nil)
			assert.Equal(t, len(outInfo), 0)
		})

		Convey(" ConvertAppName Error", func() {
			testErr := rest.NewHTTPError("error", 503000000, nil)
			dbApp.EXPECT().GetAppName(gomock.Any()).AnyTimes().Return(nil, nil, testErr)
			_, err := app.ConvertAppName([]string{"xxx"}, false, true)

			assert.Equal(t, err, testErr)
		})

		Convey(" ids not exsit", func() {
			testInfo := make([]interfaces.NameInfo, 0)
			exsitIDs := []string{"xxx"}
			dbApp.EXPECT().GetAppName(gomock.Any()).AnyTimes().Return(testInfo, exsitIDs, nil)
			_, err := app.ConvertAppName([]string{"xxx", "yyy"}, false, true)

			tmpIds := []string{"yyy"}
			testErr := rest.NewHTTPErrorV2(uerrors.NotFound, "app does not exist",
				rest.SetDetail(map[string]interface{}{"ids": tmpIds}))
			assert.Equal(t, err, testErr)
		})

		Convey(" ids not exsit1", func() {
			testInfo := make([]interfaces.NameInfo, 0)
			exsitIDs := []string{"xxx"}
			dbApp.EXPECT().GetAppName(gomock.Any()).AnyTimes().Return(testInfo, exsitIDs, nil)
			_, err := app.ConvertAppName([]string{"xxx", "yyy"}, true, true)

			tmpIds := []string{"yyy"}
			testErr := rest.NewHTTPErrorV2(uerrors.AppNotFound, "app does not exist",
				rest.SetDetail(map[string]interface{}{"ids": tmpIds}),
				rest.SetCodeStr(uerrors.StrBadRequestAppNotFound))
			assert.Equal(t, err, testErr)
		})

		Convey("Success", func() {
			tmp1 := interfaces.NameInfo{
				ID:   "xxx",
				Name: "zzzzz",
			}
			tmp2 := interfaces.NameInfo{
				ID:   "yyy",
				Name: "zzzzz1",
			}
			testInfo := []interfaces.NameInfo{
				tmp1,
				tmp2,
			}
			exsitIDs := []string{"xxx", "yyy"}
			dbApp.EXPECT().GetAppName(gomock.Any()).AnyTimes().Return(testInfo, exsitIDs, nil)
			outInfo, err := app.ConvertAppName([]string{"xxx", "yyy"}, false, true)

			assert.Equal(t, err, nil)
			assert.Equal(t, len(outInfo), 2)
		})

		Convey("Success1", func() {
			tmp1 := interfaces.NameInfo{
				ID:   "xxx",
				Name: "zzzzz",
			}
			tmp2 := interfaces.NameInfo{
				ID:   "yyy",
				Name: "zzzzz1",
			}
			testInfo := []interfaces.NameInfo{
				tmp1,
				tmp2,
			}
			exsitIDs := []string{"xxx", "yyy"}
			dbApp.EXPECT().GetAppName(gomock.Any()).AnyTimes().Return(testInfo, exsitIDs, nil)
			outInfo, err := app.ConvertAppName([]string{"xxx", "yyy", "zzz"}, false, false)

			assert.Equal(t, err, nil)
			assert.Equal(t, len(outInfo), 2)
		})
	})
}

//nolint:dupl
func TestSendAppRegisterAuditLog(t *testing.T) {
	Convey("发送app注册审计日志", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		dbApp := mock.NewMockDBApp(ctrl)
		loOutbox := mock.NewMockLogicsOutbox(ctrl)
		userDB := mock.NewMockDBUser(ctrl)
		eacplog := mock.NewMockDrivenEacpLog(ctrl)
		role := mock.NewMockLogicsRole(ctrl)

		app := newApp(dbApp, userDB, loOutbox, nil, nil, eacplog, role)

		testErr := rest.NewHTTPError("param groupid is illegal", rest.BadRequest, nil)
		data := map[string]interface{}{
			"visitor": make(map[string]interface{}),
			"name":    strENUS,
		}
		Convey("eerror", func() {
			eacplog.EXPECT().EacpLog(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(testErr)
			err := app.sendAppRegisterAuditLog(data)

			assert.Equal(t, err, testErr)
		})
	})
}

//nolint:dupl
func TestSendAppDeletedAuditLog(t *testing.T) {
	Convey("发送app删除审计日志", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		dbApp := mock.NewMockDBApp(ctrl)
		loOutbox := mock.NewMockLogicsOutbox(ctrl)
		userDB := mock.NewMockDBUser(ctrl)
		eacplog := mock.NewMockDrivenEacpLog(ctrl)
		role := mock.NewMockLogicsRole(ctrl)

		app := newApp(dbApp, userDB, loOutbox, nil, nil, eacplog, role)

		testErr := rest.NewHTTPError("param groupid is illegal", rest.BadRequest, nil)
		data := map[string]interface{}{
			"visitor": make(map[string]interface{}),
			"name":    strENUS,
		}
		Convey("eerror", func() {
			eacplog.EXPECT().EacpLog(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(testErr)
			err := app.sendAppDeletedAuditLog(data)

			assert.Equal(t, err, testErr)
		})
	})
}

//nolint:dupl
func TestSendAppModifiedAuditLog(t *testing.T) {
	Convey("发送app修改审计日志", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		dbApp := mock.NewMockDBApp(ctrl)
		loOutbox := mock.NewMockLogicsOutbox(ctrl)
		userDB := mock.NewMockDBUser(ctrl)
		eacplog := mock.NewMockDrivenEacpLog(ctrl)
		role := mock.NewMockLogicsRole(ctrl)

		app := newApp(dbApp, userDB, loOutbox, nil, nil, eacplog, role)

		testErr := rest.NewHTTPError("param groupid is illegal", rest.BadRequest, nil)
		data := map[string]interface{}{
			"visitor": make(map[string]interface{}),
			"name":    strENUS,
		}
		Convey("eerror", func() {
			eacplog.EXPECT().EacpLog(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(testErr)
			err := app.sendAppModifiedAuditLog(data)

			assert.Equal(t, err, testErr)
		})
	})
}

//nolint:funlen
func TestGenerateAppToken(t *testing.T) {
	Convey("GenerateAppToken, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		var err error
		db, txMock, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func() {
			if closeErr := db.Close(); closeErr != nil {
				assert.Equal(t, 1, 1)
			}
		}()

		dbApp := mock.NewMockDBApp(ctrl)
		loOutbox := mock.NewMockLogicsOutbox(ctrl)
		userDB := mock.NewMockDBUser(ctrl)
		eacplog := mock.NewMockDrivenEacpLog(ctrl)
		trace := mock.NewMockTraceClient(ctrl)
		hydra := mock.NewMockDrivenHydra(ctrl)
		role := mock.NewMockLogicsRole(ctrl)

		app := newApp(dbApp, userDB, loOutbox, db, hydra, eacplog, role)
		app.trace = trace

		visitor := interfaces.Visitor{
			ID:       "zzz",
			Language: interfaces.AmericanEnglish,
		}
		ctx := context.Background()

		testErr := rest.NewHTTPError("error", 503000000, nil)
		Convey("user has not role", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(nil, testErr)
			_, err := app.GenerateAppToken(ctx, &visitor, "test-id")

			assert.Equal(t, err, testErr)
		})

		Convey("app get error", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			dbApp.EXPECT().GetAppByID(gomock.Any()).AnyTimes().Return(nil, testErr)
			_, err := app.GenerateAppToken(ctx, nil, "test-id")

			assert.Equal(t, err, testErr)
		})

		Convey("app not found", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(map[string]map[interfaces.Role]bool{
				visitor.ID: {
					interfaces.SystemRoleSuperAdmin: true,
				},
			}, nil)
			dbApp.EXPECT().GetAppByID(gomock.Any()).AnyTimes().Return(nil, nil)
			_, err := app.GenerateAppToken(ctx, &visitor, "test-id")

			assert.Equal(t, err, gerrors.NewError(uerrors.StrBadRequestAppNotFound, app.i18n.Load(i18nIDObjectsInAppNotFound, visitor.Language),
				gerrors.SetDetail(map[string]interface{}{"id": "test-id"})))
		})

		appInfo := &interfaces.AppInfo{
			ID:             "test-id",
			CredentialType: interfaces.CredentialTypePassword,
		}

		Convey("app credential type is password", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(map[string]map[interfaces.Role]bool{
				visitor.ID: {
					interfaces.SystemRoleSuperAdmin: true,
				},
			}, nil)
			dbApp.EXPECT().GetAppByID(gomock.Any()).AnyTimes().Return(appInfo, nil)
			_, err := app.GenerateAppToken(ctx, &visitor, "test-id")

			assert.Equal(t, err, gerrors.NewError(gerrors.PublicBadRequest, "app credential type is password, token cannot be generated"))
		})

		appInfo.CredentialType = interfaces.CredentialTypeToken
		Convey("hydra.Update error", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(map[string]map[interfaces.Role]bool{
				visitor.ID: {
					interfaces.SystemRoleSuperAdmin: true,
				},
			}, nil)
			dbApp.EXPECT().GetAppByID(gomock.Any()).AnyTimes().Return(appInfo, nil)
			hydra.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(testErr)
			_, err := app.GenerateAppToken(ctx, &visitor, "test-id")

			assert.Equal(t, err, testErr)
		})

		Convey("hydra.DeleteClientToken error", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(map[string]map[interfaces.Role]bool{
				visitor.ID: {
					interfaces.SystemRoleSuperAdmin: true,
				},
			}, nil)
			dbApp.EXPECT().GetAppByID(gomock.Any()).AnyTimes().Return(appInfo, nil)
			hydra.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			hydra.EXPECT().DeleteClientToken(gomock.Any()).AnyTimes().Return(testErr)
			_, err := app.GenerateAppToken(ctx, &visitor, "test-id")

			assert.Equal(t, err, testErr)
		})

		Convey("hydra.GenerateToken error", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(map[string]map[interfaces.Role]bool{
				visitor.ID: {
					interfaces.SystemRoleSuperAdmin: true,
				},
			}, nil)
			dbApp.EXPECT().GetAppByID(gomock.Any()).AnyTimes().Return(appInfo, nil)
			hydra.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			hydra.EXPECT().DeleteClientToken(gomock.Any()).AnyTimes().Return(nil)
			hydra.EXPECT().GenerateToken(gomock.Any(), gomock.Any()).AnyTimes().Return("", testErr)
			_, err := app.GenerateAppToken(ctx, &visitor, "test-id")

			assert.Equal(t, err, testErr)
		})

		Convey("tx.Begin error", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(map[string]map[interfaces.Role]bool{
				visitor.ID: {
					interfaces.SystemRoleSuperAdmin: true,
				},
			}, nil)
			dbApp.EXPECT().GetAppByID(gomock.Any()).AnyTimes().Return(appInfo, nil)
			hydra.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			hydra.EXPECT().DeleteClientToken(gomock.Any()).AnyTimes().Return(nil)
			hydra.EXPECT().GenerateToken(gomock.Any(), gomock.Any()).AnyTimes().Return("token1", nil)
			txMock.ExpectBegin().WillReturnError(testErr)
			_, err := app.GenerateAppToken(ctx, &visitor, "test-id")

			assert.Equal(t, err, testErr)
		})

		Convey("eacpLog.EacpLog error", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(map[string]map[interfaces.Role]bool{
				visitor.ID: {
					interfaces.SystemRoleSuperAdmin: true,
				},
			}, nil)
			dbApp.EXPECT().GetAppByID(gomock.Any()).AnyTimes().Return(appInfo, nil)
			hydra.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			hydra.EXPECT().DeleteClientToken(gomock.Any()).AnyTimes().Return(nil)
			hydra.EXPECT().GenerateToken(gomock.Any(), gomock.Any()).AnyTimes().Return("token1", nil)
			txMock.ExpectBegin()
			loOutbox.EXPECT().AddOutboxInfo(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(testErr)
			txMock.ExpectRollback()
			_, err := app.GenerateAppToken(ctx, &visitor, "test-id")

			assert.Equal(t, err, testErr)
		})

		Convey("success", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddInternalTrace(gomock.Any()).AnyTimes().Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			role.EXPECT().GetRolesByUserIDs(gomock.Any()).AnyTimes().Return(map[string]map[interfaces.Role]bool{
				visitor.ID: {
					interfaces.SystemRoleSuperAdmin: true,
				},
			}, nil)
			dbApp.EXPECT().GetAppByID(gomock.Any()).AnyTimes().Return(appInfo, nil)
			hydra.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			hydra.EXPECT().DeleteClientToken(gomock.Any()).AnyTimes().Return(nil)
			hydra.EXPECT().GenerateToken(gomock.Any(), gomock.Any()).AnyTimes().Return("token1", nil)
			txMock.ExpectBegin()
			loOutbox.EXPECT().AddOutboxInfo(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			txMock.ExpectCommit()
			loOutbox.EXPECT().NotifyPushOutboxThread()
			data, err := app.GenerateAppToken(ctx, &visitor, "test-id")

			assert.Equal(t, err, nil)
			assert.Equal(t, data, "token1")
		})
	})
}
