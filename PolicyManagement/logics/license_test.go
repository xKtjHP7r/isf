package logics

import (
	"context"
	"errors"
	"policy_mgnt/common"
	"policy_mgnt/interfaces"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"policy_mgnt/interfaces/mock"

	gerrors "github.com/kweaver-ai/go-lib/error"
	"github.com/kweaver-ai/proton-rds-sdk-go/sqlx"
)

var (
	strUserID1   = "userID1"
	strVisitorID = "visitorID"
)

func TestOnUserStatusChanged(t *testing.T) {
	Convey("onUserStatusChanged,", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db, mockDB, err := sqlx.New()
		assert.Equal(t, err, nil)

		dnLicense := mock.NewMockDrivenLicense(ctrl)
		dnUserManagement := mock.NewMockDrivenUserManagement(ctrl)
		dbLicense := mock.NewMockDBLicense(ctrl)
		trace := mock.NewMockTraceClient(ctrl)
		eve := mock.NewMockLogicsEvent(ctrl)
		config := mock.NewMockDBConfig(ctrl)

		lic := &license{
			dnLicense:        dnLicense,
			dnUserManagement: dnUserManagement,
			db:               dbLicense,
			trace:            trace,
			tracePool:        db,
			event:            eve,
			dbConfig:         config,
			log:              common.NewLogger(),
		}

		Convey("用户被启用，接口调用失败，返回错误", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).Return()
			trace.EXPECT().AddInternalTrace(gomock.Any()).Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).Return()

			dnLicense.EXPECT().GetLicenses(gomock.Any()).Return(nil, errors.New("test"))
			err := lic.onUserStatusChanged(strUserID1, true)
			assert.Equal(t, err, errors.New("test"))
		})

		Convey("用户被禁用，接口调用失败，返回错误", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).Return()
			trace.EXPECT().AddInternalTrace(gomock.Any()).Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).Return()

			mockDB.ExpectBegin().WillReturnError(errors.New("test"))
			err := lic.onUserStatusChanged(strUserID1, false)
			assert.Equal(t, err, errors.New("test"))
		})
	})
}

func TestOnUserCreated(t *testing.T) {
	Convey("onUserCreated,", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db, _, err := sqlx.New()
		assert.Equal(t, err, nil)

		dnLicense := mock.NewMockDrivenLicense(ctrl)
		dnUserManagement := mock.NewMockDrivenUserManagement(ctrl)
		dbLicense := mock.NewMockDBLicense(ctrl)
		trace := mock.NewMockTraceClient(ctrl)
		eve := mock.NewMockLogicsEvent(ctrl)
		config := mock.NewMockDBConfig(ctrl)

		lic := &license{
			dnLicense:        dnLicense,
			dnUserManagement: dnUserManagement,
			db:               dbLicense,
			trace:            trace,
			tracePool:        db,
			event:            eve,
			dbConfig:         config,
			log:              common.NewLogger(),
		}

		Convey("接口调用失败，返回错误", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).Return()
			trace.EXPECT().AddInternalTrace(gomock.Any()).Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).Return()

			dnLicense.EXPECT().GetLicenses(gomock.Any()).Return(nil, errors.New("test"))
			err := lic.onUserCreated(strUserID1)
			assert.Equal(t, err, errors.New("test"))
		})
	})
}

func TestUserDeleteAllProducts(t *testing.T) {
	Convey("userDeleteAllProducts,", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db, mockDB, err := sqlx.New()
		assert.Equal(t, err, nil)

		dnLicense := mock.NewMockDrivenLicense(ctrl)
		dnUserManagement := mock.NewMockDrivenUserManagement(ctrl)
		dbLicense := mock.NewMockDBLicense(ctrl)
		trace := mock.NewMockTraceClient(ctrl)
		eve := mock.NewMockLogicsEvent(ctrl)
		config := mock.NewMockDBConfig(ctrl)

		lic := &license{
			dnLicense:        dnLicense,
			dnUserManagement: dnUserManagement,
			db:               dbLicense,
			trace:            trace,
			tracePool:        db,
			event:            eve,
			dbConfig:         config,
			log:              common.NewLogger(),
		}

		Convey("接口调用失败，返回错误", func() {
			mockDB.ExpectBegin().WillReturnError(errors.New("test"))
			err := lic.userDeleteAllProducts(context.Background(), strUserID1)
			assert.Equal(t, err, errors.New("test"))
		})

		Convey("DeleteUserAuthorizedProducts接口调用失败，返回错误", func() {
			mockDB.ExpectBegin()
			dbLicense.EXPECT().DeleteUserAuthorizedProducts(gomock.Any(), strUserID1, gomock.Any()).Return(errors.New("test"))
			mockDB.ExpectRollback()
			err := lic.userDeleteAllProducts(context.Background(), strUserID1)
			assert.Equal(t, err, errors.New("test"))
		})

		Convey("DeleteUserAuthorizedProducts接口调用成功，返回nil", func() {
			mockDB.ExpectBegin()
			dbLicense.EXPECT().DeleteUserAuthorizedProducts(gomock.Any(), strUserID1, gomock.Any()).Return(nil)
			mockDB.ExpectCommit()
			err := lic.userDeleteAllProducts(context.Background(), strUserID1)
			assert.Equal(t, err, nil)
		})
	})
}

func TestUserAddAllProducts(t *testing.T) {
	Convey("userAddAllProducts,", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db, mockDB, err := sqlx.New()
		assert.Equal(t, err, nil)

		dnLicense := mock.NewMockDrivenLicense(ctrl)
		dnUserManagement := mock.NewMockDrivenUserManagement(ctrl)
		dbLicense := mock.NewMockDBLicense(ctrl)
		trace := mock.NewMockTraceClient(ctrl)
		eve := mock.NewMockLogicsEvent(ctrl)
		config := mock.NewMockDBConfig(ctrl)

		lic := &license{
			dnLicense:        dnLicense,
			dnUserManagement: dnUserManagement,
			db:               dbLicense,
			trace:            trace,
			tracePool:        db,
			event:            eve,
			dbConfig:         config,
			log:              common.NewLogger(),
		}

		Convey("GetLicenses 接口调用失败，返回错误", func() {
			licenseCache = make(map[string]interfaces.License)
			licenseCacheGetTime = time.Now().Add(-25 * time.Hour)

			dnLicense.EXPECT().GetLicenses(gomock.Any()).Return(nil, errors.New("test"))
			err := lic.userAddAllProducts(context.Background(), strUserID1)
			assert.Equal(t, err, errors.New("test"))
		})

		Convey("GetConfig 接口调用失败，返回错误", func() {
			licenseCache = make(map[string]interfaces.License)
			licenseCacheGetTime = time.Now()

			config.EXPECT().GetConfig(gomock.Any(), gomock.Any()).Return("", errors.New("test"))
			err := lic.userAddAllProducts(context.Background(), strUserID1)
			assert.Equal(t, err, errors.New("test"))
		})

		Convey("GetConfig 接口调用成功 无许可证，直接返回", func() {
			licenseCache = make(map[string]interfaces.License)
			licenseCacheGetTime = time.Now()

			config.EXPECT().GetConfig(gomock.Any(), gomock.Any()).Return("", nil)

			err := lic.userAddAllProducts(context.Background(), strUserID1)
			assert.Equal(t, err, nil)
		})

		Convey("有许可证，GetAuthorizedProducts报错，直接返回", func() {
			licenseCache = make(map[string]interfaces.License)
			licenseCache = map[string]interfaces.License{
				"product1": {
					Product:        "product1",
					TotalUserQuota: 100,
				},
			}
			licenseCacheGetTime = time.Now()

			config.EXPECT().GetConfig(gomock.Any(), gomock.Any()).Return("", nil)
			dbLicense.EXPECT().GetAuthorizedProducts(gomock.Any(), gomock.Any()).Return(nil, errors.New("test"))
			err := lic.userAddAllProducts(context.Background(), strUserID1)
			assert.Equal(t, err, errors.New("test"))
		})

		Convey("有许可证，GetProductsAuthorizedCount 报错", func() {
			licenseCache = make(map[string]interfaces.License)
			licenseCache = map[string]interfaces.License{
				"product1": {
					Product:        "product1",
					TotalUserQuota: 100,
				},
			}
			licenseCacheGetTime = time.Now()

			config.EXPECT().GetConfig(gomock.Any(), gomock.Any()).Return("", nil)
			dbLicense.EXPECT().GetAuthorizedProducts(gomock.Any(), gomock.Any()).Return(map[string]interfaces.AuthorizedProduct{}, nil)
			dbLicense.EXPECT().GetProductsAuthorizedCount(gomock.Any(), gomock.Any()).Return(0, errors.New("test"))
			err := lic.userAddAllProducts(context.Background(), strUserID1)
			assert.Equal(t, err, errors.New("test"))
		})

		Convey("有许可证，GetProductsAuthorizedCount 成功，但是超过了数值", func() {
			licenseCache = make(map[string]interfaces.License)
			licenseCache = map[string]interfaces.License{
				"product1": {
					Product:        "product1",
					TotalUserQuota: 100,
				},
			}
			licenseCacheGetTime = time.Now()

			config.EXPECT().GetConfig(gomock.Any(), gomock.Any()).Return("", nil)
			dbLicense.EXPECT().GetAuthorizedProducts(gomock.Any(), gomock.Any()).Return(map[string]interfaces.AuthorizedProduct{}, nil)
			dbLicense.EXPECT().GetProductsAuthorizedCount(gomock.Any(), gomock.Any()).Return(100, nil)
			err := lic.userAddAllProducts(context.Background(), strUserID1)
			assert.Equal(t, err, nil)
		})

		Convey("有许可证，未超过授权，tx begin报错", func() {
			licenseCache = make(map[string]interfaces.License)
			licenseCache = map[string]interfaces.License{
				"product1": {
					Product:        "product1",
					TotalUserQuota: 100,
				},
			}
			licenseCacheGetTime = time.Now()

			config.EXPECT().GetConfig(gomock.Any(), gomock.Any()).Return("", nil)
			dbLicense.EXPECT().GetAuthorizedProducts(gomock.Any(), gomock.Any()).Return(map[string]interfaces.AuthorizedProduct{}, nil)
			dbLicense.EXPECT().GetProductsAuthorizedCount(gomock.Any(), gomock.Any()).Return(99, nil)
			mockDB.ExpectBegin().WillReturnError(errors.New("test"))
			err := lic.userAddAllProducts(context.Background(), strUserID1)
			assert.Equal(t, err, errors.New("test"))
		})

		Convey("有许可证，未超过授权，tx begin成功，AddAuthorizedProducts报错", func() {
			licenseCache = make(map[string]interfaces.License)
			licenseCache = map[string]interfaces.License{
				"product1": {
					Product:        "product1",
					TotalUserQuota: 100,
				},
			}
			licenseCacheGetTime = time.Now()

			config.EXPECT().GetConfig(gomock.Any(), gomock.Any()).Return("", nil)
			dbLicense.EXPECT().GetAuthorizedProducts(gomock.Any(), gomock.Any()).Return(map[string]interfaces.AuthorizedProduct{}, nil)
			dbLicense.EXPECT().GetProductsAuthorizedCount(gomock.Any(), gomock.Any()).Return(99, nil)
			mockDB.ExpectBegin()
			dbLicense.EXPECT().AddAuthorizedProducts(gomock.Any(), []interfaces.ProductInfo{
				{
					AccountID: strUserID1,
					Product:   "product1",
				},
			}, gomock.Any()).Return(errors.New("test"))
			mockDB.ExpectRollback()
			err := lic.userAddAllProducts(context.Background(), strUserID1)
			assert.Equal(t, err, errors.New("test"))
		})

		Convey("success", func() {
			licenseCache = make(map[string]interfaces.License)
			licenseCache = map[string]interfaces.License{
				"product1": {
					Product:        "product1",
					TotalUserQuota: 100,
				},
			}
			licenseCacheGetTime = time.Now()

			config.EXPECT().GetConfig(gomock.Any(), gomock.Any()).Return("", nil)
			dbLicense.EXPECT().GetAuthorizedProducts(gomock.Any(), gomock.Any()).Return(map[string]interfaces.AuthorizedProduct{}, nil)
			dbLicense.EXPECT().GetProductsAuthorizedCount(gomock.Any(), gomock.Any()).Return(99, nil)
			mockDB.ExpectBegin()
			dbLicense.EXPECT().AddAuthorizedProducts(gomock.Any(), []interfaces.ProductInfo{
				{
					AccountID: strUserID1,
					Product:   "product1",
				},
			}, gomock.Any()).Return(nil)
			mockDB.ExpectCommit()
			err := lic.userAddAllProducts(context.Background(), strUserID1)
			assert.Equal(t, err, nil)
		})

		Convey("success1", func() {
			licenseCache = make(map[string]interfaces.License)
			licenseCache = map[string]interfaces.License{
				"product1": {
					Product:        "product1",
					TotalUserQuota: 100,
				},
				"product2": {
					Product:        "product2",
					TotalUserQuota: 100,
				},
			}
			licenseCacheGetTime = time.Now()

			config.EXPECT().GetConfig(gomock.Any(), gomock.Any()).Return("", nil)
			dbLicense.EXPECT().GetAuthorizedProducts(gomock.Any(), gomock.Any()).Return(map[string]interfaces.AuthorizedProduct{
				strUserID1: {
					ID:      strUserID1,
					Type:    interfaces.ObjectTypeUser,
					Product: []string{"product1"},
				},
			}, nil)
			dbLicense.EXPECT().GetProductsAuthorizedCount(gomock.Any(), gomock.Any()).Return(99, nil)
			mockDB.ExpectBegin()
			dbLicense.EXPECT().AddAuthorizedProducts(gomock.Any(), []interfaces.ProductInfo{
				{
					AccountID: strUserID1,
					Product:   "product2",
				},
			}, gomock.Any()).Return(nil)
			mockDB.ExpectCommit()
			err := lic.userAddAllProducts(context.Background(), strUserID1)
			assert.Equal(t, err, nil)
		})
	})
}

func TestGetLicenses(t *testing.T) {
	Convey("getLicenses,", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db, _, err := sqlx.New()
		assert.Equal(t, err, nil)

		dnLicense := mock.NewMockDrivenLicense(ctrl)
		dnUserManagement := mock.NewMockDrivenUserManagement(ctrl)
		dbLicense := mock.NewMockDBLicense(ctrl)
		trace := mock.NewMockTraceClient(ctrl)
		eve := mock.NewMockLogicsEvent(ctrl)
		config := mock.NewMockDBConfig(ctrl)

		lic := &license{
			dnLicense:        dnLicense,
			dnUserManagement: dnUserManagement,
			db:               dbLicense,
			trace:            trace,
			tracePool:        db,
			event:            eve,
			dbConfig:         config,
			log:              common.NewLogger(),
		}

		visitor := &interfaces.Visitor{ID: strUserID1, Language: interfaces.SimplifiedChinese, Type: interfaces.RealName}

		Convey("GetUserInfos 接口调用失败，返回错误", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).Return()
			trace.EXPECT().AddInternalTrace(gomock.Any()).Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).Return()

			dnUserManagement.EXPECT().GetUserInfos(gomock.Any(), gomock.Any()).Return(nil, errors.New("test"))
			_, err := lic.GetLicenses(context.Background(), visitor)
			assert.Equal(t, err, errors.New("test"))
		})

		Convey("用户不是管理员，无权限", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).Return()
			trace.EXPECT().AddInternalTrace(gomock.Any()).Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).Return()

			dnUserManagement.EXPECT().GetUserInfos(gomock.Any(), gomock.Any()).Return(map[string]interfaces.UserInfo{
				strUserID1: {
					ID:    strUserID1,
					Roles: map[interfaces.Role]bool{interfaces.SystemRoleNormalUser: true},
				},
			}, nil)
			_, err := lic.GetLicenses(context.Background(), visitor)
			assert.Equal(t, err, gerrors.NewError(gerrors.PublicForbidden, "this user has no permission to access this resource"))
		})

		Convey("GetLicenses 接口调用报错，返回许可证信息", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).Return()
			trace.EXPECT().AddInternalTrace(gomock.Any()).Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).Return()

			licenseCache = make(map[string]interfaces.License)
			licenseCacheGetTime = time.Now().Add(-25 * time.Hour)

			dnUserManagement.EXPECT().GetUserInfos(gomock.Any(), gomock.Any()).Return(map[string]interfaces.UserInfo{
				strUserID1: {
					ID:    strUserID1,
					Roles: map[interfaces.Role]bool{interfaces.SystemRoleSuperAdmin: true},
				},
			}, nil)
			dnLicense.EXPECT().GetLicenses(gomock.Any()).Return(nil, errors.New("test"))
			_, err := lic.GetLicenses(context.Background(), visitor)
			assert.Equal(t, err, errors.New("test"))
		})

		Convey("GetProductsAuthorizedCount 接口调用失败", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).Return()
			trace.EXPECT().AddInternalTrace(gomock.Any()).Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).Return()

			licenseCache = make(map[string]interfaces.License)
			licenseCache = map[string]interfaces.License{
				"product1": {
					Product:        "product1",
					TotalUserQuota: 100,
				},
			}
			licenseCacheGetTime = time.Now().Add(-25 * time.Hour)

			dnUserManagement.EXPECT().GetUserInfos(gomock.Any(), gomock.Any()).Return(map[string]interfaces.UserInfo{
				strUserID1: {
					ID:    strUserID1,
					Roles: map[interfaces.Role]bool{interfaces.SystemRoleSuperAdmin: true},
				},
			}, nil)
			dnLicense.EXPECT().GetLicenses(gomock.Any()).Return(map[string]interfaces.License{
				"product1": {
					Product:        "product1",
					TotalUserQuota: 100,
				},
			}, nil)
			dbLicense.EXPECT().GetProductsAuthorizedCount(gomock.Any(), gomock.Any()).Return(0, errors.New("test"))
			_, err := lic.GetLicenses(context.Background(), visitor)
			assert.Equal(t, err, errors.New("test"))
		})

		Convey("success", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).Return()
			trace.EXPECT().AddInternalTrace(gomock.Any()).Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).Return()

			licenseCache = make(map[string]interfaces.License)
			licenseCacheGetTime = time.Now().Add(-25 * time.Hour)

			dnUserManagement.EXPECT().GetUserInfos(gomock.Any(), gomock.Any()).Return(map[string]interfaces.UserInfo{
				strUserID1: {
					ID:    strUserID1,
					Roles: map[interfaces.Role]bool{interfaces.SystemRoleSuperAdmin: true},
				},
			}, nil)
			dnLicense.EXPECT().GetLicenses(gomock.Any()).Return(map[string]interfaces.License{
				"product1": {
					Product:        "product1",
					TotalUserQuota: 100,
				},
			}, nil)
			dbLicense.EXPECT().GetProductsAuthorizedCount(gomock.Any(), gomock.Any()).Return(101, nil)
			out, err := lic.GetLicenses(context.Background(), visitor)
			assert.Equal(t, err, nil)
			assert.Equal(t, out, map[string]interfaces.LicenseInfo{
				"product1": {
					Product:             "product1",
					TotalUserQuota:      100,
					AuthorizedUserCount: 101,
				},
			})
		})
	})
}

func TestGetAuthorizedProducts(t *testing.T) {
	Convey("getAuthorizedProducts,", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db, _, err := sqlx.New()
		assert.Equal(t, err, nil)

		dnLicense := mock.NewMockDrivenLicense(ctrl)
		dnUserManagement := mock.NewMockDrivenUserManagement(ctrl)
		dbLicense := mock.NewMockDBLicense(ctrl)
		trace := mock.NewMockTraceClient(ctrl)
		eve := mock.NewMockLogicsEvent(ctrl)
		config := mock.NewMockDBConfig(ctrl)

		lic := &license{
			dnLicense:        dnLicense,
			dnUserManagement: dnUserManagement,
			db:               dbLicense,
			trace:            trace,
			tracePool:        db,
			event:            eve,
			dbConfig:         config,
			log:              common.NewLogger(),
		}

		visitor := &interfaces.Visitor{ID: strUserID1, Language: interfaces.SimplifiedChinese, Type: interfaces.RealName}

		Convey("GetUserInfos 接口调用失败，返回错误", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).Return()
			trace.EXPECT().AddInternalTrace(gomock.Any()).Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).Return()

			dnUserManagement.EXPECT().GetUserInfos(gomock.Any(), gomock.Any()).Return(nil, errors.New("test"))
			_, err := lic.GetAuthorizedProducts(context.Background(), visitor, []string{strUserID1})
			assert.Equal(t, err, errors.New("test"))
		})

		Convey("用户不是管理员，无权限", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).Return()
			trace.EXPECT().AddInternalTrace(gomock.Any()).Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).Return()

			dnUserManagement.EXPECT().GetUserInfos(gomock.Any(), gomock.Any()).Return(map[string]interfaces.UserInfo{
				strUserID1: {
					ID:    strUserID1,
					Roles: map[interfaces.Role]bool{interfaces.SystemRoleNormalUser: true},
				},
			}, nil)
			_, err := lic.GetAuthorizedProducts(context.Background(), visitor, []string{strUserID1})
			assert.Equal(t, err, gerrors.NewError(gerrors.PublicForbidden, "this user has no permission to access this resource"))
		})

		Convey("GetAuthorizedProducts 接口调用报错，返回错误信息", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).Return()
			trace.EXPECT().AddInternalTrace(gomock.Any()).Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).Return()

			licenseCache = make(map[string]interfaces.License)
			licenseCacheGetTime = time.Now().Add(-25 * time.Hour)

			dnUserManagement.EXPECT().GetUserInfos(gomock.Any(), gomock.Any()).Return(map[string]interfaces.UserInfo{
				strUserID1: {
					ID:    strUserID1,
					Roles: map[interfaces.Role]bool{interfaces.SystemRoleSuperAdmin: true},
				},
			}, nil)
			dbLicense.EXPECT().GetAuthorizedProducts(gomock.Any(), gomock.Any()).Return(nil, errors.New("test"))
			_, err := lic.GetAuthorizedProducts(context.Background(), visitor, []string{strUserID1})
			assert.Equal(t, err, errors.New("test"))
		})

		Convey("GetAuthorizedProducts 接口调用成功，返回已授权产品", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).Return()
			trace.EXPECT().AddInternalTrace(gomock.Any()).Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).Return()

			licenseCache = make(map[string]interfaces.License)
			licenseCacheGetTime = time.Now().Add(-25 * time.Hour)

			dnUserManagement.EXPECT().GetUserInfos(gomock.Any(), gomock.Any()).Return(map[string]interfaces.UserInfo{
				strUserID1: {
					ID:    strUserID1,
					Roles: map[interfaces.Role]bool{interfaces.SystemRoleSuperAdmin: true},
				},
			}, nil)
			dbLicense.EXPECT().GetAuthorizedProducts(gomock.Any(), gomock.Any()).Return(map[string]interfaces.AuthorizedProduct{
				strUserID1: {
					ID:      strUserID1,
					Type:    interfaces.ObjectTypeUser,
					Product: []string{"product1", "product2"},
				},
			}, nil)
			out, err := lic.GetAuthorizedProducts(context.Background(), visitor, []string{strUserID1})
			assert.Equal(t, err, nil)
			assert.Equal(t, out, map[string]interfaces.AuthorizedProduct{
				strUserID1: {
					ID:      strUserID1,
					Type:    interfaces.ObjectTypeUser,
					Product: []string{"product1", "product2"},
				},
			})
		})
	})
}

func TestCheckProductAuthorized(t *testing.T) {
	Convey("checkProductAuthorized,", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db, _, err := sqlx.New()
		assert.Equal(t, err, nil)

		dnLicense := mock.NewMockDrivenLicense(ctrl)
		dnUserManagement := mock.NewMockDrivenUserManagement(ctrl)
		dbLicense := mock.NewMockDBLicense(ctrl)
		trace := mock.NewMockTraceClient(ctrl)
		eve := mock.NewMockLogicsEvent(ctrl)
		config := mock.NewMockDBConfig(ctrl)

		lic := &license{
			dnLicense:        dnLicense,
			dnUserManagement: dnUserManagement,
			db:               dbLicense,
			trace:            trace,
			tracePool:        db,
			event:            eve,
			dbConfig:         config,
			log:              common.NewLogger(),
			i18n: common.NewI18n(common.I18nMap{
				i18nIDHasNoLicense: {
					interfaces.SimplifiedChinese:  "产品无有效授权，无法登录，请联系管理员。",
					interfaces.TraditionalChinese: "產品無有效授權，無法登入，請聯繫管理員。",
					interfaces.AmericanEnglish:    "The product has no valid license. Login is unavailable. Please contact the administrator.",
				},
				i18nIDUserHasNoAuthUserProduct: {
					interfaces.SimplifiedChinese:  "您暂未获得此产品的使用授权，无法登录，请联系管理员。",
					interfaces.TraditionalChinese: "您暫未獲得此產品的使用授權，無法登入，請聯繫管理員。",
					interfaces.AmericanEnglish:    "You do not currently have the authorization to use this product and cannot log in. Please contact the administrator.",
				},
			}),
		}

		visitor := &interfaces.Visitor{ID: strUserID1, Language: interfaces.SimplifiedChinese, Type: interfaces.RealName}

		Convey("GetLicenses 接口调用失败，返回错误", func() {
			licenseCache = make(map[string]interfaces.License)
			licenseCacheGetTime = time.Now().Add(-25 * time.Hour)

			trace.EXPECT().SetInternalSpanName(gomock.Any()).Return()
			trace.EXPECT().AddInternalTrace(gomock.Any()).Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).Return()

			dnLicense.EXPECT().GetLicenses(gomock.Any()).Return(nil, errors.New("test"))
			_, _, err := lic.CheckProductAuthorized(context.Background(), visitor, "product1")
			assert.Equal(t, err, errors.New("test"))
		})

		Convey("许可证失效", func() {
			licenseCache = make(map[string]interfaces.License)
			licenseCacheGetTime = time.Now()

			trace.EXPECT().SetInternalSpanName(gomock.Any()).Return()
			trace.EXPECT().AddInternalTrace(gomock.Any()).Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).Return()

			result, unauthorizedReason, err := lic.CheckProductAuthorized(context.Background(), visitor, "product1")
			assert.Equal(t, err, nil)
			assert.Equal(t, result, false)
			assert.Equal(t, unauthorizedReason, lic.i18n.Load(i18nIDHasNoLicense, interfaces.SimplifiedChinese))
		})

		Convey("许可证有效，GetAuthorizedProducts 接口调用失败", func() {
			licenseCache = make(map[string]interfaces.License)
			licenseCache = map[string]interfaces.License{
				"product1": {
					Product:        "product1",
					TotalUserQuota: 100,
				},
			}
			licenseCacheGetTime = time.Now()

			trace.EXPECT().SetInternalSpanName(gomock.Any()).Return()
			trace.EXPECT().AddInternalTrace(gomock.Any()).Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).Return()

			dbLicense.EXPECT().GetAuthorizedProducts(gomock.Any(), gomock.Any()).Return(nil, errors.New("test"))
			_, _, err := lic.CheckProductAuthorized(context.Background(), visitor, "product1")
			assert.Equal(t, err, errors.New("test"))
		})

		Convey("许可证有效，GetAuthorizedProducts 接口调用成功", func() {
			licenseCache = make(map[string]interfaces.License)
			licenseCache = map[string]interfaces.License{
				"product1": {
					Product:        "product1",
					TotalUserQuota: 100,
				},
			}
			licenseCacheGetTime = time.Now()

			trace.EXPECT().SetInternalSpanName(gomock.Any()).Return()
			trace.EXPECT().AddInternalTrace(gomock.Any()).Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).Return()

			dbLicense.EXPECT().GetAuthorizedProducts(gomock.Any(), gomock.Any()).Return(map[string]interfaces.AuthorizedProduct{
				strUserID1: {
					ID:      strUserID1,
					Type:    interfaces.ObjectTypeUser,
					Product: []string{"product1"},
				},
			}, nil)
			result, unauthorizedReason, err := lic.CheckProductAuthorized(context.Background(), visitor, "product1")
			assert.Equal(t, err, nil)
			assert.Equal(t, result, true)
			assert.Equal(t, unauthorizedReason, "")
		})

		Convey("许可证有效，但是此用户无授权", func() {
			licenseCache = make(map[string]interfaces.License)
			licenseCache = map[string]interfaces.License{
				"product1": {
					Product:        "product1",
					TotalUserQuota: 100,
				},
			}
			licenseCacheGetTime = time.Now()

			trace.EXPECT().SetInternalSpanName(gomock.Any()).Return()
			trace.EXPECT().AddInternalTrace(gomock.Any()).Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).Return()

			dbLicense.EXPECT().GetAuthorizedProducts(gomock.Any(), gomock.Any()).Return(map[string]interfaces.AuthorizedProduct{
				strUserID1: {
					ID:      strUserID1,
					Type:    interfaces.ObjectTypeUser,
					Product: []string{"product2"},
				},
			}, nil)
			result, unauthorizedReason, err := lic.CheckProductAuthorized(context.Background(), visitor, "product1")
			assert.Equal(t, err, nil)
			assert.Equal(t, result, false)
			assert.Equal(t, unauthorizedReason, lic.i18n.Load(i18nIDUserHasNoAuthUserProduct, interfaces.SimplifiedChinese))
		})
	})
}

func TestUpdateAuthorizedProducts(t *testing.T) {
	Convey("updateAuthorizedProducts,", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db, mockDB, err := sqlx.New()
		assert.Equal(t, err, nil)

		dnLicense := mock.NewMockDrivenLicense(ctrl)
		dnUserManagement := mock.NewMockDrivenUserManagement(ctrl)
		dbLicense := mock.NewMockDBLicense(ctrl)
		trace := mock.NewMockTraceClient(ctrl)
		eve := mock.NewMockLogicsEvent(ctrl)
		config := mock.NewMockDBConfig(ctrl)
		eacpLog := mock.NewMockDrivenEacpLog(ctrl)
		outbox := mock.NewMockLogicsOutbox(ctrl)

		lic := &license{
			dnLicense:        dnLicense,
			dnUserManagement: dnUserManagement,
			db:               dbLicense,
			trace:            trace,
			tracePool:        db,
			event:            eve,
			dbConfig:         config,
			eacpLog:          eacpLog,
			log:              common.NewLogger(),
			outbox:           outbox,
			i18n: common.NewI18n(common.I18nMap{
				i18nIDProductAuthorizedNotValid: {
					interfaces.SimplifiedChinese:  "产品“%s” 无有效授权。",
					interfaces.TraditionalChinese: "產品“%s” 無有效授權。",
					interfaces.AmericanEnglish:    "Product \"%s\" has no valid license.",
				},
				i18nIDProductAuthorizedOverQuota: {
					interfaces.SimplifiedChinese:  "产品“%s” 用户授权数已达上限。",
					interfaces.TraditionalChinese: "產品“%s” 用戶授權數已達上限。",
					interfaces.AmericanEnglish:    "The user license limit for products \"%s\" has been reached.",
				},
			}),
		}

		visitor := &interfaces.Visitor{ID: strVisitorID, Language: interfaces.SimplifiedChinese, Type: interfaces.RealName}

		Convey("GetUserInfos 接口调用失败，返回错误", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).Return()
			trace.EXPECT().AddInternalTrace(gomock.Any()).Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).Return()

			dnUserManagement.EXPECT().GetUserInfos(gomock.Any(), gomock.Any()).Return(nil, errors.New("test"))
			err := lic.UpdateAuthorizedProducts(context.Background(), visitor, map[string]interfaces.AuthorizedProduct{
				strUserID1: {
					ID:      strUserID1,
					Type:    interfaces.ObjectTypeUser,
					Product: []string{"product1"},
				},
			})
			assert.Equal(t, err, errors.New("test"))
		})

		Convey("用户不是管理员，无权限", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).Return()
			trace.EXPECT().AddInternalTrace(gomock.Any()).Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).Return()

			dnUserManagement.EXPECT().GetUserInfos(gomock.Any(), []string{strUserID1, visitor.ID}).Return(map[string]interfaces.UserInfo{
				strUserID1: {
					ID:    strUserID1,
					Roles: map[interfaces.Role]bool{interfaces.SystemRoleNormalUser: true},
				},
				visitor.ID: {
					ID:    visitor.ID,
					Roles: map[interfaces.Role]bool{interfaces.SystemRoleNormalUser: true},
				},
			}, nil)
			err := lic.UpdateAuthorizedProducts(context.Background(), visitor, map[string]interfaces.AuthorizedProduct{
				strUserID1: {
					ID:      strUserID1,
					Type:    interfaces.ObjectTypeUser,
					Product: []string{"product1"},
				},
			})
			assert.Equal(t, err, gerrors.NewError(gerrors.PublicForbidden, "this user has no permission to access this resource"))
		})

		Convey("获取当前所有的许可证失败", func() {
			licenseCache = make(map[string]interfaces.License)
			licenseCacheGetTime = time.Now().Add(-25 * time.Hour)

			trace.EXPECT().SetInternalSpanName(gomock.Any()).Return()
			trace.EXPECT().AddInternalTrace(gomock.Any()).Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).Return()

			dnUserManagement.EXPECT().GetUserInfos(gomock.Any(), []string{strUserID1, visitor.ID}).Return(map[string]interfaces.UserInfo{
				strUserID1: {
					ID:    strUserID1,
					Roles: map[interfaces.Role]bool{interfaces.SystemRoleNormalUser: true},
				},
				visitor.ID: {
					ID:    visitor.ID,
					Roles: map[interfaces.Role]bool{interfaces.SystemRoleSuperAdmin: true},
				},
			}, nil)
			dnLicense.EXPECT().GetLicenses(gomock.Any()).Return(nil, errors.New("test"))
			err := lic.UpdateAuthorizedProducts(context.Background(), visitor, map[string]interfaces.AuthorizedProduct{
				strUserID1: {
					ID:      strUserID1,
					Type:    interfaces.ObjectTypeUser,
					Product: []string{"product1"},
				},
			})
			assert.Equal(t, err, errors.New("test"))
		})

		Convey("GetAuthorizedProducts 接口调用报错，返回错误", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).Return()
			trace.EXPECT().AddInternalTrace(gomock.Any()).Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).Return()

			dnLicense.EXPECT().GetLicenses(gomock.Any()).Return(map[string]interfaces.License{
				"product1": {
					Product:        "product1",
					TotalUserQuota: 100,
				},
			}, nil)

			dnUserManagement.EXPECT().GetUserInfos(gomock.Any(), []string{strUserID1, visitor.ID}).Return(map[string]interfaces.UserInfo{
				strUserID1: {
					ID:    strUserID1,
					Roles: map[interfaces.Role]bool{interfaces.SystemRoleNormalUser: true},
				},
				visitor.ID: {
					ID:    visitor.ID,
					Roles: map[interfaces.Role]bool{interfaces.SystemRoleSuperAdmin: true},
				},
			}, nil)
			dbLicense.EXPECT().GetAuthorizedProducts(gomock.Any(), gomock.Any()).Return(nil, errors.New("test"))

			err := lic.UpdateAuthorizedProducts(context.Background(), visitor, map[string]interfaces.AuthorizedProduct{
				strUserID1: {
					ID:      strUserID1,
					Type:    interfaces.ObjectTypeUser,
					Product: []string{"product1"},
				},
			})
			assert.Equal(t, err, errors.New("test"))
		})

		//
		Convey("GetProductsAuthorizedCount 接口调用报错，返回错误", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).Return()
			trace.EXPECT().AddInternalTrace(gomock.Any()).Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).Return()

			licenseCacheGetTime = time.Now().Add(-25 * time.Hour)
			dnUserManagement.EXPECT().GetUserInfos(gomock.Any(), []string{strUserID1, visitor.ID}).Return(map[string]interfaces.UserInfo{
				strUserID1: {
					ID:    strUserID1,
					Roles: map[interfaces.Role]bool{interfaces.SystemRoleNormalUser: true},
				},
				visitor.ID: {
					ID:    visitor.ID,
					Roles: map[interfaces.Role]bool{interfaces.SystemRoleSuperAdmin: true},
				},
			}, nil)
			dnLicense.EXPECT().GetLicenses(gomock.Any()).Return(map[string]interfaces.License{
				"product1": {
					Product:        "product1",
					TotalUserQuota: 100,
				},
				"product2": {
					Product:        "product2",
					TotalUserQuota: 100,
				},
				"product3": {
					Product:        "product3",
					TotalUserQuota: 100,
				},
			}, nil)
			dbLicense.EXPECT().GetAuthorizedProducts(gomock.Any(), gomock.Any()).Return(map[string]interfaces.AuthorizedProduct{
				strUserID1: {
					ID:      strUserID1,
					Type:    interfaces.ObjectTypeUser,
					Product: []string{"product1", "product3"},
				},
			}, nil)

			dbLicense.EXPECT().GetProductsAuthorizedCount(gomock.Any(), gomock.Any()).Return(0, errors.New("test"))

			err := lic.UpdateAuthorizedProducts(context.Background(), visitor, map[string]interfaces.AuthorizedProduct{
				strUserID1: {
					ID:      strUserID1,
					Type:    interfaces.ObjectTypeUser,
					Product: []string{"product1", "product2"},
				},
			})
			assert.Equal(t, err, errors.New("test"))
		})

		Convey("授权产品数量超过限制", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).Return()
			trace.EXPECT().AddInternalTrace(gomock.Any()).Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).Return()

			licenseCacheGetTime = time.Now().Add(-25 * time.Hour)
			dnUserManagement.EXPECT().GetUserInfos(gomock.Any(), []string{strUserID1, visitor.ID}).Return(map[string]interfaces.UserInfo{
				strUserID1: {
					ID:    strUserID1,
					Roles: map[interfaces.Role]bool{interfaces.SystemRoleNormalUser: true},
				},
				visitor.ID: {
					ID:    visitor.ID,
					Roles: map[interfaces.Role]bool{interfaces.SystemRoleSuperAdmin: true},
				},
			}, nil)
			dnLicense.EXPECT().GetLicenses(gomock.Any()).Return(map[string]interfaces.License{
				"product1": {
					Product:        "product1",
					TotalUserQuota: 100,
				},
				"product2": {
					Product:        "product2",
					TotalUserQuota: 100,
				},
				"product3": {
					Product:        "product3",
					TotalUserQuota: 100,
				},
			}, nil)
			dbLicense.EXPECT().GetAuthorizedProducts(gomock.Any(), gomock.Any()).Return(map[string]interfaces.AuthorizedProduct{
				strUserID1: {
					ID:      strUserID1,
					Type:    interfaces.ObjectTypeUser,
					Product: []string{"product3"},
				},
			}, nil)

			dbLicense.EXPECT().GetProductsAuthorizedCount(gomock.Any(), gomock.Any()).AnyTimes().Return(100, nil)

			err := lic.UpdateAuthorizedProducts(context.Background(), visitor, map[string]interfaces.AuthorizedProduct{
				strUserID1: {
					ID:      strUserID1,
					Type:    interfaces.ObjectTypeUser,
					Product: []string{"product2"},
				},
			})
			assert.Equal(t, err, gerrors.NewError(StrProductAuthorizedNotValid, "产品“product2” 用户授权数已达上限。"))
		})

		Convey("产品未授权", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).Return()
			trace.EXPECT().AddInternalTrace(gomock.Any()).Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).Return()

			licenseCacheGetTime = time.Now().Add(-25 * time.Hour)
			dnUserManagement.EXPECT().GetUserInfos(gomock.Any(), []string{strUserID1, visitor.ID}).Return(map[string]interfaces.UserInfo{
				strUserID1: {
					ID:    strUserID1,
					Roles: map[interfaces.Role]bool{interfaces.SystemRoleNormalUser: true},
				},
				visitor.ID: {
					ID:    visitor.ID,
					Roles: map[interfaces.Role]bool{interfaces.SystemRoleSuperAdmin: true},
				},
			}, nil)
			dnLicense.EXPECT().GetLicenses(gomock.Any()).Return(map[string]interfaces.License{
				"product3": {
					Product:        "product3",
					TotalUserQuota: 100,
				},
			}, nil)
			dbLicense.EXPECT().GetAuthorizedProducts(gomock.Any(), gomock.Any()).Return(map[string]interfaces.AuthorizedProduct{
				strUserID1: {
					ID:      strUserID1,
					Type:    interfaces.ObjectTypeUser,
					Product: []string{},
				},
			}, nil)

			dbLicense.EXPECT().GetProductsAuthorizedCount(gomock.Any(), gomock.Any()).AnyTimes().Return(100, nil)

			err := lic.UpdateAuthorizedProducts(context.Background(), visitor, map[string]interfaces.AuthorizedProduct{
				strUserID1: {
					ID:      strUserID1,
					Type:    interfaces.ObjectTypeUser,
					Product: []string{"product1", "product3"},
				},
			})
			assert.Equal(t, err, gerrors.NewError(StrProductAuthorizedNotValid, "产品“product3” 用户授权数已达上限。\n产品“product1” 无有效授权。"))
		})

		Convey("tx begin error", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).Return()
			trace.EXPECT().AddInternalTrace(gomock.Any()).Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).Return()

			licenseCacheGetTime = time.Now().Add(-25 * time.Hour)
			dnUserManagement.EXPECT().GetUserInfos(gomock.Any(), []string{strUserID1, visitor.ID}).Return(map[string]interfaces.UserInfo{
				strUserID1: {
					ID:    strUserID1,
					Roles: map[interfaces.Role]bool{interfaces.SystemRoleNormalUser: true},
				},
				visitor.ID: {
					ID:    visitor.ID,
					Roles: map[interfaces.Role]bool{interfaces.SystemRoleSuperAdmin: true},
				},
			}, nil)
			dnLicense.EXPECT().GetLicenses(gomock.Any()).Return(map[string]interfaces.License{
				"product1": {
					Product:        "product1",
					TotalUserQuota: 100,
				},
				"product2": {
					Product:        "product2",
					TotalUserQuota: 100,
				},
				"product3": {
					Product:        "product3",
					TotalUserQuota: 100,
				},
			}, nil)
			dbLicense.EXPECT().GetAuthorizedProducts(gomock.Any(), gomock.Any()).Return(map[string]interfaces.AuthorizedProduct{
				strUserID1: {
					ID:      strUserID1,
					Type:    interfaces.ObjectTypeUser,
					Product: []string{"product1", "product3"},
				},
			}, nil)

			dbLicense.EXPECT().GetProductsAuthorizedCount(gomock.Any(), gomock.Any()).Return(99, nil)
			mockDB.ExpectBegin().WillReturnError(errors.New("test"))

			err := lic.UpdateAuthorizedProducts(context.Background(), visitor, map[string]interfaces.AuthorizedProduct{
				strUserID1: {
					ID:      strUserID1,
					Type:    interfaces.ObjectTypeUser,
					Product: []string{"product1", "product2"},
				},
			})
			assert.Equal(t, err, errors.New("test"))
		})

		Convey("DeleteAuthorizedProducts 接口调用报错，返回错误", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).Return()
			trace.EXPECT().AddInternalTrace(gomock.Any()).Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).Return()

			licenseCacheGetTime = time.Now().Add(-25 * time.Hour)
			dnUserManagement.EXPECT().GetUserInfos(gomock.Any(), []string{strUserID1, visitor.ID}).Return(map[string]interfaces.UserInfo{
				strUserID1: {
					ID:    strUserID1,
					Roles: map[interfaces.Role]bool{interfaces.SystemRoleNormalUser: true},
				},
				visitor.ID: {
					ID:    visitor.ID,
					Roles: map[interfaces.Role]bool{interfaces.SystemRoleSuperAdmin: true},
				},
			}, nil)
			dnLicense.EXPECT().GetLicenses(gomock.Any()).Return(map[string]interfaces.License{
				"product1": {
					Product:        "product1",
					TotalUserQuota: 100,
				},
				"product2": {
					Product:        "product2",
					TotalUserQuota: 100,
				},
				"product3": {
					Product:        "product3",
					TotalUserQuota: 100,
				},
			}, nil)
			dbLicense.EXPECT().GetAuthorizedProducts(gomock.Any(), gomock.Any()).Return(map[string]interfaces.AuthorizedProduct{
				strUserID1: {
					ID:      strUserID1,
					Type:    interfaces.ObjectTypeUser,
					Product: []string{"product1", "product3"},
				},
			}, nil)

			dbLicense.EXPECT().GetProductsAuthorizedCount(gomock.Any(), gomock.Any()).Return(99, nil)
			mockDB.ExpectBegin()
			dbLicense.EXPECT().DeleteAuthorizedProducts(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("test"))
			mockDB.ExpectRollback()

			err := lic.UpdateAuthorizedProducts(context.Background(), visitor, map[string]interfaces.AuthorizedProduct{
				strUserID1: {
					ID:      strUserID1,
					Type:    interfaces.ObjectTypeUser,
					Product: []string{"product1", "product2"},
				},
			})
			assert.Equal(t, err, errors.New("test"))
		})

		Convey("AddAuthorizedProducts 接口调用报错，返回错误", func() {

			trace.EXPECT().SetInternalSpanName(gomock.Any()).Return()
			trace.EXPECT().AddInternalTrace(gomock.Any()).Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).Return()

			licenseCacheGetTime = time.Now().Add(-25 * time.Hour)
			dnUserManagement.EXPECT().GetUserInfos(gomock.Any(), []string{strUserID1, visitor.ID}).Return(map[string]interfaces.UserInfo{
				strUserID1: {
					ID:    strUserID1,
					Roles: map[interfaces.Role]bool{interfaces.SystemRoleNormalUser: true},
				},
				visitor.ID: {
					ID:    visitor.ID,
					Roles: map[interfaces.Role]bool{interfaces.SystemRoleSuperAdmin: true},
				},
			}, nil)
			dnLicense.EXPECT().GetLicenses(gomock.Any()).Return(map[string]interfaces.License{
				"product1": {
					Product:        "product1",
					TotalUserQuota: 100,
				},
				"product2": {
					Product:        "product2",
					TotalUserQuota: 100,
				},
				"product3": {
					Product:        "product3",
					TotalUserQuota: 100,
				},
			}, nil)
			dbLicense.EXPECT().GetAuthorizedProducts(gomock.Any(), gomock.Any()).Return(map[string]interfaces.AuthorizedProduct{
				strUserID1: {
					ID:      strUserID1,
					Type:    interfaces.ObjectTypeUser,
					Product: []string{"product1", "product3"},
				},
			}, nil)

			dbLicense.EXPECT().GetProductsAuthorizedCount(gomock.Any(), gomock.Any()).Return(99, nil)
			mockDB.ExpectBegin()
			dbLicense.EXPECT().DeleteAuthorizedProducts(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			dbLicense.EXPECT().AddAuthorizedProducts(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("test"))
			mockDB.ExpectRollback()

			err := lic.UpdateAuthorizedProducts(context.Background(), visitor, map[string]interfaces.AuthorizedProduct{
				strUserID1: {
					ID:      strUserID1,
					Type:    interfaces.ObjectTypeUser,
					Product: []string{"product1", "product2"},
				},
			})
			assert.Equal(t, err, errors.New("test"))
		})

		Convey("success", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).Return()
			trace.EXPECT().AddInternalTrace(gomock.Any()).Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).Return()

			licenseCacheGetTime = time.Now().Add(-25 * time.Hour)
			dnUserManagement.EXPECT().GetUserInfos(gomock.Any(), []string{strUserID1, visitor.ID}).Return(map[string]interfaces.UserInfo{
				strUserID1: {
					ID:    strUserID1,
					Roles: map[interfaces.Role]bool{interfaces.SystemRoleNormalUser: true},
				},
				visitor.ID: {
					ID:    visitor.ID,
					Roles: map[interfaces.Role]bool{interfaces.SystemRoleSuperAdmin: true},
				},
			}, nil)
			dnLicense.EXPECT().GetLicenses(gomock.Any()).Return(map[string]interfaces.License{
				"product1": {
					Product:        "product1",
					TotalUserQuota: 100,
				},
				"product2": {
					Product:        "product2",
					TotalUserQuota: 100,
				},
				"product3": {
					Product:        "product3",
					TotalUserQuota: 100,
				},
			}, nil)
			dbLicense.EXPECT().GetAuthorizedProducts(gomock.Any(), gomock.Any()).Return(map[string]interfaces.AuthorizedProduct{
				strUserID1: {
					ID:      strUserID1,
					Type:    interfaces.ObjectTypeUser,
					Product: []string{"product1", "product3"},
				},
			}, nil)

			dbLicense.EXPECT().GetProductsAuthorizedCount(gomock.Any(), gomock.Any()).Return(99, nil)
			mockDB.ExpectBegin()
			dbLicense.EXPECT().DeleteAuthorizedProducts(gomock.Any(), []interfaces.ProductInfo{
				{
					AccountID: strUserID1,
					Product:   "product3",
				},
			}, gomock.Any()).Return(nil)
			dbLicense.EXPECT().AddAuthorizedProducts(gomock.Any(), []interfaces.ProductInfo{
				{
					AccountID: strUserID1,
					Product:   "product2",
				},
			}, gomock.Any()).Return(nil)
			outbox.EXPECT().AddOutboxInfo(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			mockDB.ExpectCommit()
			outbox.EXPECT().NotifyPushOutboxThread().AnyTimes().Return()

			err := lic.UpdateAuthorizedProducts(context.Background(), visitor, map[string]interfaces.AuthorizedProduct{
				strUserID1: {
					ID:      strUserID1,
					Type:    interfaces.ObjectTypeUser,
					Product: []string{"product1", "product2"},
				},
			})
			assert.Equal(t, err, nil)
		})
	})
}

func TestOnUserDeleted(t *testing.T) {
	Convey("用户删除", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db, mockDB, err := sqlx.New()
		assert.Equal(t, err, nil)

		dnLicense := mock.NewMockDrivenLicense(ctrl)
		dnUserManagement := mock.NewMockDrivenUserManagement(ctrl)
		dbLicense := mock.NewMockDBLicense(ctrl)
		trace := mock.NewMockTraceClient(ctrl)
		eve := mock.NewMockLogicsEvent(ctrl)
		config := mock.NewMockDBConfig(ctrl)

		lic := &license{
			dnLicense:        dnLicense,
			dnUserManagement: dnUserManagement,
			db:               dbLicense,
			trace:            trace,
			tracePool:        db,
			event:            eve,
			dbConfig:         config,
			log:              common.NewLogger(),
		}

		Convey("tx begin error", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).Return()
			trace.EXPECT().AddInternalTrace(gomock.Any()).Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).Return()

			mockDB.ExpectBegin().WillReturnError(errors.New("test"))
			mockDB.ExpectRollback()

			err := lic.onUserDeleted(strUserID1)
			assert.Equal(t, err, errors.New("test"))
		})

		Convey("DeleteUserAuthorizedProducts 接口调用报错，返回错误", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).Return()
			trace.EXPECT().AddInternalTrace(gomock.Any()).Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).Return()

			mockDB.ExpectBegin()
			dbLicense.EXPECT().DeleteUserAuthorizedProducts(gomock.Any(), strUserID1, gomock.Any()).Return(errors.New("test"))
			mockDB.ExpectRollback()

			err := lic.onUserDeleted(strUserID1)
			assert.Equal(t, err, errors.New("test"))
		})

		Convey("success", func() {
			trace.EXPECT().SetInternalSpanName(gomock.Any()).Return()
			trace.EXPECT().AddInternalTrace(gomock.Any()).Return(context.Background(), nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).Return()

			mockDB.ExpectBegin()
			dbLicense.EXPECT().DeleteUserAuthorizedProducts(gomock.Any(), strUserID1, gomock.Any()).Return(nil)
			mockDB.ExpectCommit()

			err := lic.onUserDeleted(strUserID1)
			assert.Equal(t, err, nil)
		})
	})
}
