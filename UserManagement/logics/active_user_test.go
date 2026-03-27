package logics

import (
	"context"
	"testing"

	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/common"
	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/interfaces"
	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/interfaces/mock"
	gerrors "devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/go-lib/error"
	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/go-lib/rest"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/proton-rds-sdk-go/sqlx"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/assert"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

func TestSendActiveUserInfoExportedLog(t *testing.T) {
	Convey("发送活跃用户信息导出审计日志", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		auDB := mock.NewMockDBActiveUser(ctrl)
		userDB := mock.NewMockDBUser(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		ob := mock.NewMockLogicsOutbox(ctrl)
		eacpLog := mock.NewMockDrivenEacpLog(ctrl)

		auLogics := &activeUser{
			activeUserDB: auDB,
			userDB:       userDB,
			trace:        common.SvcARTrace,
			role:         role,
			i18n:         i18nActiveUserMap,
			logger:       common.NewLogger(),
			ob:           ob,
			eacpLog:      eacpLog,
		}

		testErr := rest.NewHTTPError("param groupid is illegal", rest.BadRequest, nil)
		data := map[string]interface{}{
			"visitor": make(map[string]interface{}),
			"is_year": false,
		}
		Convey("eerror", func() {
			eacpLog.EXPECT().OpActiveUserInfoExported(gomock.Any(), gomock.Any()).AnyTimes().Return(testErr)
			err := auLogics.sendActiveUserInfoExportedLog(data)

			assert.Equal(t, err, testErr)
		})
	})
}

func TestGetActiveUserInfo(t *testing.T) {
	Convey("GetActiveUserInfo, db is available", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		common.InitARTrace("test")

		dPool, txMock, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func() {
			if closeErr := dPool.Close(); closeErr != nil {
				assert.Equal(t, 1, 1)
			}
		}()

		auDB := mock.NewMockDBActiveUser(ctrl)
		userDB := mock.NewMockDBUser(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		ob := mock.NewMockLogicsOutbox(ctrl)

		auLogics := &activeUser{
			activeUserDB: auDB,
			userDB:       userDB,
			trace:        common.SvcARTrace,
			role:         role,
			i18n:         i18nActiveUserMap,
			logger:       common.NewLogger(),
			ob:           ob,
			pool:         dPool,
		}

		testErr := rest.NewHTTPError("error", 503000000, nil)
		visitor := interfaces.Visitor{
			ID: strID,
		}
		Convey("获取角色信息失败", func() {
			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).Return(nil, testErr)
			out, err := auLogics.GetActiveUserInfo(context.Background(), &visitor, false, 2026, 1)
			assert.Equal(t, err, testErr)
			assert.Equal(t, out, nil)
		})

		Convey("用户没有权限", func() {
			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).Return(map[string]map[interfaces.Role]bool{
				strID: {
					interfaces.SystemRoleNormalUser: true,
				},
			}, nil)
			out, err := auLogics.GetActiveUserInfo(context.Background(), &visitor, false, 2026, 1)
			assert.Equal(t, err, gerrors.NewError(gerrors.PublicForbidden, "this user do not has this role"))
			assert.Equal(t, out, nil)
		})

		Convey("获取年度报表，数据库获取失败", func() {
			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).Return(map[string]map[interfaces.Role]bool{
				strID: {
					interfaces.SystemRoleSuperAdmin: true,
				},
			}, nil)
			auDB.EXPECT().GetYearActiveUserInfo(gomock.Any(), 2026).Return(nil, testErr)
			out, err := auLogics.GetActiveUserInfo(context.Background(), &visitor, true, 2026, 0)
			assert.Equal(t, err, testErr)
			assert.Equal(t, out, nil)
		})

		Convey("获取月度报表，数据库获取失败", func() {
			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).Return(map[string]map[interfaces.Role]bool{
				strID: {
					interfaces.SystemRoleSuperAdmin: true,
				},
			}, nil)
			auDB.EXPECT().GetMonthActiveUserInfo(gomock.Any(), 2026, 1).Return(nil, testErr)
			out, err := auLogics.GetActiveUserInfo(context.Background(), &visitor, false, 2026, 1)
			assert.Equal(t, err, testErr)
			assert.Equal(t, out, nil)
		})

		activeUserInfos := []interfaces.ActiveUserInfo{
			{
				Year:          2026,
				Month:         1,
				ActiveCount:   100,
				ActivateCount: 100,
			},
		}

		Convey("tx begin error", func() {
			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).Return(map[string]map[interfaces.Role]bool{
				strID: {
					interfaces.SystemRoleSuperAdmin: true,
				},
			}, nil)
			auDB.EXPECT().GetMonthActiveUserInfo(gomock.Any(), 2026, 1).Return(activeUserInfos, nil)
			auDB.EXPECT().GetMonthTotalCount(gomock.Any(), 2026, 1).Return(100, 100, nil)
			txMock.ExpectBegin().WillReturnError(testErr)
			_, err := auLogics.GetActiveUserInfo(context.Background(), &visitor, false, 2026, 1)
			assert.Equal(t, err, testErr)
		})

		Convey("AddOutboxInfoerror", func() {
			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).Return(map[string]map[interfaces.Role]bool{
				strID: {
					interfaces.SystemRoleSuperAdmin: true,
				},
			}, nil)
			auDB.EXPECT().GetMonthActiveUserInfo(gomock.Any(), 2026, 1).Return(activeUserInfos, nil)
			auDB.EXPECT().GetMonthTotalCount(gomock.Any(), 2026, 1).Return(100, 100, nil)
			txMock.ExpectBegin()
			ob.EXPECT().AddOutboxInfo(gomock.Any(), gomock.Any(), gomock.Any()).Return(testErr)
			txMock.ExpectRollback()
			_, err := auLogics.GetActiveUserInfo(context.Background(), &visitor, false, 2026, 1)
			assert.Equal(t, err, testErr)
		})

		Convey("成功", func() {
			role.EXPECT().GetRolesByUserIDs2(gomock.Any(), gomock.Any()).Return(map[string]map[interfaces.Role]bool{
				strID: {
					interfaces.SystemRoleSuperAdmin: true,
				},
			}, nil)
			auDB.EXPECT().GetMonthActiveUserInfo(gomock.Any(), 2026, 1).Return(activeUserInfos, nil)
			auDB.EXPECT().GetMonthTotalCount(gomock.Any(), 2026, 1).Return(100, 100, nil)
			txMock.ExpectBegin()
			ob.EXPECT().AddOutboxInfo(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			txMock.ExpectCommit()
			ob.EXPECT().NotifyPushOutboxThread()
			out, err := auLogics.GetActiveUserInfo(context.Background(), &visitor, false, 2026, 1)
			assert.Equal(t, err, nil)
			assert.NotEqual(t, string(out), "")
		})
	})
}
