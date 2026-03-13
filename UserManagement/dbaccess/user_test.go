package dbaccess

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kweaver-ai/proton-rds-sdk-go/sqlx"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-playground/assert"
	jsoniter "github.com/json-iterator/go"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"

	"UserManagement/common"
	"UserManagement/interfaces"
	mocks "UserManagement/interfaces/mock"
)

const (
	mockUserID = "userID1"
)

func newUserDB(ptrDB *sqlx.DB) *user {
	return &user{
		db:     ptrDB,
		logger: common.NewLogger(),
		lockKeyMap: map[interfaces.DistributedLockType]string{
			interfaces.UserExpiredDisableLock:  "user_expired_disable_lock",
			interfaces.UserNotLoginDisableLock: "user_not_login_disable_lock",
		},
	}
}

func TestGetDirectBelongDepartmentIDs(t *testing.T) {
	Convey("GetDirectBelongDepartmentIDs, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		user := newUserDB(db)

		userID := "user_id"
		Convey("unassigned users", func() {
			fields := []string{"f_department_id", "f_path"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow("-1", "test"))
			directIDs, _, httpErr := user.GetDirectBelongDepartmentIDs(userID)
			assert.Equal(t, len(directIDs), 0)
			assert.Equal(t, httpErr, nil)
		})

		Convey("success", func() {
			fields := []string{"f_department_id", "f_path"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow("direct_id", "direct_path"))
			_, _, httpErr := user.GetDirectBelongDepartmentIDs(userID)
			assert.Equal(t, httpErr, nil)
		})
	})
}

func TestGetDirectBelongDepartmentIDs2(t *testing.T) {
	Convey("GetDirectBelongDepartmentIDs2, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		trace := mocks.NewMockTraceClient(ctrl)
		ctx := context.Background()

		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		user := newUserDB(db)
		userID := "user_id"
		user.dbTrace = db
		user.trace = trace
		Convey("unassigned users", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			fields := []string{"f_department_id", "f_path"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow("-1", "test"))
			directIDs, _, httpErr := user.GetDirectBelongDepartmentIDs2(ctx, userID)
			assert.Equal(t, len(directIDs), 0)
			assert.Equal(t, httpErr, nil)
		})

		Convey("success", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			fields := []string{"f_department_id", "f_path"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow("direct_id", "direct_path"))
			_, _, httpErr := user.GetDirectBelongDepartmentIDs2(ctx, userID)
			assert.Equal(t, httpErr, nil)
		})
	})
}

func TestGetUserName(t *testing.T) {
	Convey("GetUserName, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		assert.NotEqual(t, db, nil)
		user := newUserDB(db)

		userIDs := make([]string, 0)

		Convey("user not exist", func() {
			fields := []string{
				"f_user_id",
				"f_display_name",
			}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields))
			tempInfoMap, userNames, httpErr := user.GetUserName(userIDs)
			assert.Equal(t, len(userNames), 0)
			assert.Equal(t, httpErr, nil)
			assert.Equal(t, len(tempInfoMap), 0)
		})

		Convey("Success", func() {
			userIDs = append(userIDs, "test")

			fields := []string{
				"f_user_id",
				"f_display_name",
			}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow("user_id", "f_display_name"))
			tempInfoMap, name, httpErr := user.GetUserName(userIDs)
			assert.Equal(t, httpErr, nil)
			tempNameInfo := interfaces.UserDBInfo{
				ID:   "user_id",
				Name: "f_display_name",
			}
			assert.Equal(t, tempInfoMap, []interfaces.UserDBInfo{tempNameInfo})
			assert.Equal(t, name, []string{"user_id"})
		})
	})
}

func TestGetUsersInDepartments(t *testing.T) {
	Convey("GetUsersInDepartments, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		assert.NotEqual(t, db, nil)
		user := newUserDB(db)

		var userIDs []string
		var depIDs []string

		Convey("userIDs empty", func() {
			roleIDs, httpErr := user.GetUsersInDepartments(userIDs, depIDs)
			assert.Equal(t, len(roleIDs), 0)
			assert.Equal(t, httpErr, nil)
		})

		userIDs = append(userIDs, "xxx")
		Convey("depIDs empty", func() {
			roleIDs, httpErr := user.GetUsersInDepartments(userIDs, depIDs)
			assert.Equal(t, len(roleIDs), 0)
			assert.Equal(t, httpErr, nil)
		})

		depIDs = append(depIDs, "yyy")
		Convey("user id not exist", func() {
			fields := []string{
				"f_user_id",
			}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields))
			roleIDs, httpErr := user.GetUsersInDepartments(userIDs, depIDs)
			assert.Equal(t, len(roleIDs), 0)
			assert.Equal(t, httpErr, nil)
		})

		Convey("Success1", func() {
			fields := []string{
				"f_user_id",
			}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow("user_id"))
			roleIDs, httpErr := user.GetUsersInDepartments(userIDs, depIDs)
			assert.Equal(t, len(roleIDs), 1)
			assert.Equal(t, httpErr, nil)
			assert.Equal(t, roleIDs[0], "user_id")
		})

		Convey("f_user_id", func() {
			fields := []string{
				"f_role_id",
			}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow("user_id").AddRow("user_id1"))
			roleIDs, httpErr := user.GetUsersInDepartments(userIDs, depIDs)
			assert.Equal(t, len(roleIDs), 2)
			assert.Equal(t, httpErr, nil)
			assert.Equal(t, roleIDs[0], "user_id")
			assert.Equal(t, roleIDs[1], "user_id1")
		})
	})
}

func TestGetUserDBInfo(t *testing.T) {
	Convey("GetUserDBInfo, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		assert.NotEqual(t, db, nil)
		user := newUserDB(db)

		Convey("Success1", func() {
			fields := []string{
				"f_user_id",
				"f_login_name",
				"f_display_name",
				"f_priority",
				"f_csf_level",
				"f_status",
				"f_auto_disable_status",
				"f_mail_address",
				"f_auth_type",
				"f_freeze_status",
				"f_real_name_auth_status",
				"f_tel_number",
				"f_third_party_attr",
				"f_third_party_id",
				"f_pwd_control",
				"f_pwd_timestamp",
				"f_password",
				"f_sha2_password",
				"f_oss_id",
				"f_manager_id",
				"f_create_time",
				"f_csf_level2",
			}

			time1 := time.Now()
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow("asdasasasds", "xx", "yy", 1, 2, 2, 2, "", 1, 1, 1, "xxx",
				"zzz", "asd", 0, time.Now().Format("2006-01-02 15:04:05"), "password", "sha2_password", "oss_id", "manager_id",
				time1.Format("2006-01-02 15:04:05"), 13))
			info, httpErr := user.GetUserDBInfo([]string{mockUserID})
			assert.Equal(t, httpErr, nil)
			assert.Equal(t, len(info), 1)
			assert.Equal(t, info[0].ID, "asdasasasds")
			assert.Equal(t, info[0].Priority, 1)
			assert.Equal(t, info[0].CSFLevel, 2)
			assert.Equal(t, info[0].DisableStatus, interfaces.Deleted)
			assert.Equal(t, info[0].AutoDisableStatus, interfaces.ExpireDisabled)
			assert.Equal(t, info[0].Account, "xx")
			assert.Equal(t, info[0].Name, "yy")
			assert.Equal(t, info[0].AuthType, interfaces.Local)
			assert.Equal(t, info[0].Frozen, true)
			assert.Equal(t, info[0].Authenticated, true)
			assert.Equal(t, info[0].TelNumber, "xxx")
			assert.Equal(t, info[0].ThirdAttr, "zzz")
			assert.Equal(t, info[0].ThirdID, "asd")
			assert.Equal(t, info[0].PWDControl, false)
			assert.Equal(t, info[0].PWDTimeStamp, time.Now().Unix())
			assert.Equal(t, info[0].Password, "password")
			assert.Equal(t, info[0].Sha2Password, "sha2_password")
			assert.Equal(t, info[0].OssID, "oss_id")
			assert.Equal(t, info[0].ManagerID, "manager_id")
			assert.Equal(t, info[0].CreatedAtTimeStamp, time1.Unix())
			assert.Equal(t, info[0].CSFLevel2, 13)
		})
	})
}

//nolint:lll
func TestGetUserDBInfo2(t *testing.T) {
	Convey("GetUserDBInfo2, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		trace := mocks.NewMockTraceClient(ctrl)
		ctx := context.Background()
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		assert.NotEqual(t, db, nil)
		user := newUserDB(db)
		user.dbTrace = db
		user.trace = trace

		Convey("Success1", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()
			fields := []string{
				"f_user_id",
				"f_login_name",
				"f_display_name",
				"f_priority",
				"f_csf_level",
				"f_status",
				"f_auto_disable_status",
				"f_mail_address",
				"f_auth_type",
				"f_freeze_status",
				"f_real_name_auth_status",
				"f_tel_number",
				"f_third_party_attr",
				"f_third_party_id",
				"f_pwd_control",
				"f_pwd_timestamp",
				"f_password",
				"f_sha2_password",
				"f_oss_id",
			}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow("asdasasasds", "xx", "yy", 1, 2, 2, 2, "", 1, 1, 1, "xxx", "zzz", "asd", 0, time.Now().Format("2006-01-02 15:04:05"), "password", "sha2_password", "oss_id"))
			info, httpErr := user.GetUserDBInfo2(ctx, []string{mockUserID})
			assert.Equal(t, httpErr, nil)
			assert.Equal(t, len(info), 1)
			assert.Equal(t, info[0].ID, "asdasasasds")
			assert.Equal(t, info[0].Priority, 1)
			assert.Equal(t, info[0].CSFLevel, 2)
			assert.Equal(t, info[0].DisableStatus, interfaces.Deleted)
			assert.Equal(t, info[0].AutoDisableStatus, interfaces.ExpireDisabled)
			assert.Equal(t, info[0].Account, "xx")
			assert.Equal(t, info[0].Name, "yy")
			assert.Equal(t, info[0].AuthType, interfaces.Local)
			assert.Equal(t, info[0].Frozen, true)
			assert.Equal(t, info[0].Authenticated, true)
			assert.Equal(t, info[0].TelNumber, "xxx")
			assert.Equal(t, info[0].ThirdAttr, "zzz")
			assert.Equal(t, info[0].ThirdID, "asd")
			assert.Equal(t, info[0].PWDControl, false)
			assert.Equal(t, info[0].PWDTimeStamp, time.Now().Unix())
			assert.Equal(t, info[0].Password, "password")
			assert.Equal(t, info[0].Sha2Password, "sha2_password")
			assert.Equal(t, info[0].OssID, "oss_id")
		})
	})
}

func TestGetUserInfoByAccount(t *testing.T) {
	Convey("GetUserInfoByAccount, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		assert.NotEqual(t, db, nil)
		user := newUserDB(db)

		Convey("id is empty", func() {
			info, httpErr := user.GetUserInfoByAccount("")
			assert.Equal(t, httpErr, nil)
			assert.Equal(t, info.ID, "")
		})

		Convey("Success1", func() {
			fields := []string{
				"f_user_id",
				"f_status",
				"f_auto_disable_status",
				"f_mail_address",
				"f_auth_type",
				"f_tel_number",
				"f_pwd_control",
				"f_login_name",
				"f_pwd_error_latest_timestamp",
				"f_pwd_error_cnt",
				"f_ldap_server_type",
				"f_domain_path",
			}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow("asdasasasds", 2, 2, "1@qq.com", 1, "185", 1, "login_name_001", time.Now().Format(time.DateTime), 0, 0, "xx"))
			info, httpErr := user.GetUserInfoByAccount(mockUserID)
			assert.Equal(t, httpErr, nil)
			assert.Equal(t, info.AuthType, interfaces.Local)
			assert.Equal(t, info.Email, "1@qq.com")
			assert.Equal(t, info.TelNumber, "185")
			assert.Equal(t, info.PWDControl, true)
			assert.Equal(t, info.ID, "asdasasasds")
			assert.Equal(t, info.DisableStatus, interfaces.Deleted)
			assert.Equal(t, info.AutoDisableStatus, interfaces.ExpireDisabled)
			assert.Equal(t, info.Account, "login_name_001")
			assert.Equal(t, info.PWDErrLatestTime, time.Now().Unix())
			assert.Equal(t, info.PWDErrCnt, 0)
			assert.Equal(t, info.LDAPType, interfaces.OtherLDAP)
			assert.Equal(t, info.DomainPath, "xx")
		})
	})
}

func TestGetUserInfoByIDCard(t *testing.T) {
	Convey("GetUserInfoByIDCard, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		assert.NotEqual(t, db, nil)
		user := newUserDB(db)
		Convey("id is empty", func() {
			info, httpErr := user.GetUserInfoByIDCard("")
			assert.Equal(t, httpErr, nil)
			assert.Equal(t, info.ID, "")
		})

		Convey("Success1", func() {
			fields := []string{
				"f_user_id",
				"f_status",
				"f_auto_disable_status",
				"f_mail_address",
				"f_auth_type",
				"f_tel_number",
				"f_pwd_control",
				"f_login_name",
				"f_pwd_error_latest_timestamp",
				"f_pwd_error_cnt",
				"f_ldap_server_type",
				"f_domain_path",
			}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow("asdasasasds", 2, 2, "1@qq.com", 1, "185", 1, "login_name_001", time.Now().Format(time.DateTime), 0, 0, "xx"))
			info, httpErr := user.GetUserInfoByIDCard(mockUserID)
			assert.Equal(t, httpErr, nil)
			assert.Equal(t, info.ID, "asdasasasds")
			assert.Equal(t, info.DisableStatus, interfaces.Deleted)
			assert.Equal(t, info.AutoDisableStatus, interfaces.ExpireDisabled)
			assert.Equal(t, info.AuthType, interfaces.Local)
			assert.Equal(t, info.Email, "1@qq.com")
			assert.Equal(t, info.TelNumber, "185")
			assert.Equal(t, info.PWDControl, true)
			assert.Equal(t, info.Account, "login_name_001")
			assert.Equal(t, info.PWDErrLatestTime, time.Now().Unix())
			assert.Equal(t, info.PWDErrCnt, 0)
			assert.Equal(t, info.LDAPType, interfaces.OtherLDAP)
			assert.Equal(t, info.DomainPath, "xx")
		})
	})
}

func TestGetOrgAduitDepartInfo(t *testing.T) {
	Convey("GetOrgAduitDepartInfo, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		userDB := newUserDB(db)
		Convey("no data", func() {
			fields := []string{"f_department_id"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields))
			out, httpErr := userDB.GetOrgAduitDepartInfo("xx")
			assert.Equal(t, len(out), 0)
			assert.Equal(t, httpErr, nil)
		})

		Convey("success", func() {
			fields := []string{"f_department_id"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow("xx"))
			_, httpErr := userDB.GetOrgAduitDepartInfo("xx")
			assert.Equal(t, httpErr, nil)
		})
	})
}

func TestGetOrgAduitDepartInfo2(t *testing.T) {
	Convey("GetOrgAduitDepartInfo2, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		trace := mocks.NewMockTraceClient(ctrl)
		ctx := context.Background()

		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		userDB := newUserDB(db)
		userDB.dbTrace = db
		userDB.trace = trace
		Convey("no data", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			fields := []string{"f_department_id"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields))
			out, httpErr := userDB.GetOrgAduitDepartInfo2(ctx, "xx")
			assert.Equal(t, len(out), 0)
			assert.Equal(t, httpErr, nil)
		})

		Convey("success", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			fields := []string{"f_department_id"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow("xx"))
			_, httpErr := userDB.GetOrgAduitDepartInfo2(ctx, "xx")
			assert.Equal(t, httpErr, nil)
		})
	})
}

func TestGetOrgManagerDepartInfo(t *testing.T) {
	Convey("GetOrgManagerDepartInfo, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		userDB := newUserDB(db)
		Convey("no data", func() {
			fields := []string{"f_department_id"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields))
			out, httpErr := userDB.GetOrgManagerDepartInfo("xx")
			assert.Equal(t, len(out), 0)
			assert.Equal(t, httpErr, nil)
		})

		Convey("success", func() {
			fields := []string{"f_department_id"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow("xx"))
			out, httpErr := userDB.GetOrgManagerDepartInfo("xx")
			assert.Equal(t, httpErr, nil)
			assert.Equal(t, len(out), 1)
			assert.Equal(t, out[0], "xx")
		})
	})
}

func TestGetOrgManagerDepartInfo2(t *testing.T) {
	Convey("GetOrgManagerDepartInfo2, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		trace := mocks.NewMockTraceClient(ctrl)
		ctx := context.Background()

		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		userDB := newUserDB(db)
		userDB.dbTrace = db
		userDB.trace = trace

		Convey("no data", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			fields := []string{"f_department_id"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields))
			out, httpErr := userDB.GetOrgManagerDepartInfo2(ctx, "xx")
			assert.Equal(t, len(out), 0)
			assert.Equal(t, httpErr, nil)
		})

		Convey("success", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			fields := []string{"f_department_id"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow("xx"))
			out, httpErr := userDB.GetOrgManagerDepartInfo2(ctx, "xx")
			assert.Equal(t, httpErr, nil)
			assert.Equal(t, len(out), 1)
			assert.Equal(t, out[0], "xx")
		})
	})
}

func TestSearchUsersByKeywordInDeparts(t *testing.T) {
	Convey("SearchUsersByKeyInScope, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		trace := mocks.NewMockTraceClient(ctrl)

		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		userDB := newUserDB(db)
		userDB.trace = trace
		userDB.dbTrace = db
		ctx := context.Background()
		Convey("no deps", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			fields := []string{"f_user_id", "f_display_name", "f_login_name", "f_priority"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields))
			out, httpErr := userDB.SearchOrgUsersByKey(ctx, false, false, "xx", 0, 100, true, false, []string{"xx", "zz", "asd", "aasd"})
			assert.Equal(t, len(out), 0)
			assert.Equal(t, httpErr, nil)
		})

		Convey("no data", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			fields := []string{"f_user_id", "f_display_name", "f_login_name", "f_priority"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields))
			out, httpErr := userDB.SearchOrgUsersByKey(ctx, false, false, "xx", 0, 100, true, false, []string{"xx", "zz", "asd", "aasd"})
			assert.Equal(t, len(out), 0)
			assert.Equal(t, httpErr, nil)
		})

		Convey("success", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			fields := []string{"f_user_id", "f_display_name", "f_login_name", "f_priority"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow("xx", "", "", 11))
			_, httpErr := userDB.SearchOrgUsersByKey(ctx, true, true, "xx", 0, 100, true, true, []string{"xx", "zz", "asd", "aasd"})
			assert.Equal(t, httpErr, nil)
		})
	})
}

func TestModifyUserInfo(t *testing.T) {
	Convey("ModifyPassword, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		userDB := newUserDB(db)
		var bRange interfaces.UserUpdateRange
		var userInfo interfaces.UserDBInfo
		Convey("no update success", func() {
			mock.ExpectBegin()
			mock.ExpectCommit()
			tx, _ := db.Begin()
			err := userDB.ModifyUserInfo(bRange, &userInfo, tx)
			assert.Equal(t, err, nil)
		})

		bRange.UpdatePWD = true
		Convey("execute error", func() {
			mock.ExpectBegin()
			mock.ExpectExec("").WillReturnError(errors.New("unknown"))
			mock.ExpectCommit()
			tx, _ := db.Begin()
			err := userDB.ModifyUserInfo(bRange, &userInfo, tx)
			assert.NotEqual(t, err, nil)
		})

		Convey("success", func() {
			mock.ExpectBegin()
			mock.ExpectExec("").WillReturnError(nil)
			mock.ExpectCommit()
			tx, _ := db.Begin()
			err := userDB.ModifyUserInfo(bRange, &userInfo, tx)
			assert.NotEqual(t, err, nil)
		})
	})
}

func TestGetUserPath(t *testing.T) {
	Convey("GetUserPath, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		user := newUserDB(db)

		Convey("userid is nil", func() {
			paths, httpErr := user.GetUsersPath(nil)
			assert.Equal(t, len(paths), 0)
			assert.Equal(t, httpErr, nil)
		})

		testErr := errors.New("this is error")
		Convey("query error", func() {
			mock.ExpectQuery("").WillReturnError(testErr)
			paths, httpErr := user.GetUsersPath([]string{userID})
			assert.Equal(t, len(paths), 0)
			assert.Equal(t, httpErr, testErr)
		})

		testPath := make(map[string][]string)
		testPath["userID"] = []string{"f_path"}
		Convey("success", func() {
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_path"}).AddRow("userID", "f_path"))
			paths, httpErr := user.GetUsersPath([]string{userID})
			assert.Equal(t, len(paths), 1)
			assert.Equal(t, httpErr, nil)

			assert.Equal(t, paths, testPath)
		})
	})
}

func TestGetUserPath2(t *testing.T) {
	Convey("GetUserPath2, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		trace := mocks.NewMockTraceClient(ctrl)
		ctx := context.Background()

		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		user := newUserDB(db)
		user.dbTrace = db
		user.trace = trace

		Convey("userid is nil", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			paths, httpErr := user.GetUsersPath2(ctx, nil)
			assert.Equal(t, len(paths), 0)
			assert.Equal(t, httpErr, nil)
		})

		testErr := errors.New("this is error")
		Convey("query error", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			mock.ExpectQuery("").WillReturnError(testErr)
			paths, httpErr := user.GetUsersPath2(ctx, []string{userID})
			assert.Equal(t, len(paths), 0)
			assert.Equal(t, httpErr, testErr)
		})

		testPath := make(map[string][]string)
		testPath["userID"] = []string{"f_path"}
		Convey("success", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_path"}).AddRow("userID", "f_path"))
			paths, httpErr := user.GetUsersPath2(ctx, []string{userID})
			assert.Equal(t, len(paths), 1)
			assert.Equal(t, httpErr, nil)

			assert.Equal(t, paths, testPath)
		})
	})
}

func TestUpdatePwdErrInfo(t *testing.T) {
	Convey("UpdatePwdErrInfo, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		userDB := newUserDB(db)
		Convey("Success", func() {
			mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(1, 1))

			err := userDB.UpdatePwdErrInfo("id", 1, time.Now().Unix())

			assert.Equal(t, err, nil)
		})
	})
}

func TestGetOrgManagersDepartInfo(t *testing.T) {
	Convey("GetOrgManagersDepartInfo, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		userDB := newUserDB(db)
		Convey("id is nil", func() {
			out, httpErr := userDB.GetOrgManagersDepartInfo(nil)
			assert.Equal(t, len(out), 0)
			assert.Equal(t, httpErr, nil)
		})

		Convey("sql error", func() {
			mock.ExpectQuery("").WillReturnError(errors.New(""))
			out, httpErr := userDB.GetOrgManagersDepartInfo([]string{strID})
			assert.Equal(t, len(out), 0)
			assert.NotEqual(t, httpErr, nil)
		})

		Convey("success", func() {
			fields := []string{"f_user_id", "f_department_id"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow(strID, strID))
			out, httpErr := userDB.GetOrgManagersDepartInfo([]string{strID})
			assert.Equal(t, httpErr, nil)
			assert.Equal(t, len(out), 1)
			assert.Equal(t, out[strID], []string{strID})
		})
	})
}

func TestGetUserInfoByName(t *testing.T) {
	Convey("GetUserInfoByName, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		trace := mocks.NewMockTraceClient(ctrl)
		ctx := context.Background()

		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		user := newUserDB(db)
		user.dbTrace = db
		user.trace = trace

		Convey("QueryContext error", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			mock.ExpectQuery("").WillReturnError(errors.New(""))
			_, httpErr := user.GetUserInfoByName(ctx, "")
			assert.Equal(t, httpErr, errors.New(""))
		})

		Convey("success", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"f_user_id"}).AddRow(strID))
			out, httpErr := user.GetUserInfoByName(ctx, "")
			assert.Equal(t, httpErr, nil)
			assert.Equal(t, out.ID, strID)
		})
	})
}

func TestSearchUserInfoByName(t *testing.T) {
	Convey("SearchUserInfoByName, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		trace := mocks.NewMockTraceClient(ctrl)
		ctx := context.Background()

		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		user := newUserDB(db)
		user.dbTrace = db
		user.trace = trace

		Convey("QueryContext error", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			mock.ExpectQuery("").WillReturnError(errors.New(""))
			_, httpErr := user.SearchUserInfoByName(ctx, "")
			assert.Equal(t, httpErr, errors.New(""))
		})

		Convey("success", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_display_name"}).AddRow(strID, strID1).AddRow(strAsc, strAsc))
			out, httpErr := user.SearchUserInfoByName(ctx, "")
			assert.Equal(t, httpErr, nil)
			assert.Equal(t, len(out), 2)
			assert.Equal(t, out[0].ID, strID)
			assert.Equal(t, out[0].Name, strID1)
			assert.Equal(t, out[1].ID, strAsc)
			assert.Equal(t, out[1].Name, strAsc)
		})
	})
}

func TestGetUserCustomAttr(t *testing.T) {
	Convey("GetUserCustomAttr, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		assert.NotEqual(t, db, nil)
		user := newUserDB(db)

		Convey("id is nil", func() {
			out, httpErr := user.GetUserCustomAttr("")
			assert.Equal(t, len(out), 0)
			assert.Equal(t, httpErr, nil)
		})

		Convey("sql error", func() {
			mock.ExpectQuery("").WillReturnError(errors.New(""))
			out, httpErr := user.GetUserCustomAttr(mockUserID)
			assert.Equal(t, len(out), 0)
			assert.NotEqual(t, httpErr, nil)
		})

		Convey("Success1", func() {
			fields := []string{
				"f_custom_attr",
			}
			out := make(map[string]interface{}, 0)
			out["as"] = "as"
			customAttrbyte, _ := jsoniter.Marshal(out)
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow(customAttrbyte))
			info, httpErr := user.GetUserCustomAttr(mockUserID)
			assert.Equal(t, httpErr, nil)
			assert.Equal(t, len(info), 1)
		})
	})
}

//nolint:dupl
func TestUpdateUserCustomAttr(t *testing.T) {
	Convey("UpdateUserCustomAttr, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		trace := mocks.NewMockTraceClient(ctrl)
		ctx := context.Background()
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		user := newUserDB(db)
		user.dbTrace = db
		user.trace = trace
		out := make(map[string]interface{}, 0)
		out["test2"] = "test2"

		Convey("execute error", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()
			mock.ExpectBegin()
			mock.ExpectExec("").WillReturnError(errors.New("unknown"))
			mock.ExpectCommit()
			tx, _ := db.Begin()
			err := user.UpdateUserCustomAttr(ctx, mockUserID, out, tx)
			assert.NotEqual(t, err, nil)
		})

		Convey("success", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()
			mock.ExpectBegin()
			mock.ExpectExec("").WillReturnError(nil)
			mock.ExpectCommit()
			tx, _ := db.Begin()
			err := user.UpdateUserCustomAttr(ctx, mockUserID, out, tx)
			assert.NotEqual(t, err, nil)
		})
	})
}

//nolint:dupl
func TestAddUserCustomAttr(t *testing.T) {
	Convey("AddUserCustomAttr, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		trace := mocks.NewMockTraceClient(ctrl)
		ctx := context.Background()
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		user := newUserDB(db)
		user.dbTrace = db
		user.trace = trace
		out := make(map[string]interface{}, 0)
		out["test3"] = "test3"

		Convey("execute error", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()
			mock.ExpectBegin()
			mock.ExpectExec("").WillReturnError(errors.New("unknown"))
			mock.ExpectCommit()
			tx, _ := db.Begin()
			err := user.AddUserCustomAttr(ctx, mockUserID, out, tx)
			assert.NotEqual(t, err, nil)
		})

		Convey("success", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()
			mock.ExpectBegin()
			mock.ExpectExec("").WillReturnError(nil)
			mock.ExpectCommit()
			tx, _ := db.Begin()
			err := user.AddUserCustomAttr(ctx, mockUserID, out, tx)
			assert.NotEqual(t, err, nil)
		})
	})
}

func TestGetUserDBInfoByTels(t *testing.T) {
	Convey("GetUserDBInfoByTels, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		trace := mocks.NewMockTraceClient(ctrl)
		ctx := context.Background()

		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		user := newUserDB(db)
		user.dbTrace = db
		user.trace = trace

		Convey("QueryContext error", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			mock.ExpectQuery("").WillReturnError(errors.New(""))
			_, httpErr := user.GetUserDBInfoByTels(ctx, []string{strID})
			assert.Equal(t, httpErr, errors.New(""))
		})

		Convey("success", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_login_name", "f_display_name", "f_mail_address", "f_tel_number", "f_third_party_id"}).
				AddRow(strID, strID1, strID, strID1, strID, strID1).AddRow(strAsc, strAsc, strAsc, strAsc, strAsc, strAsc))
			out, httpErr := user.GetUserDBInfoByTels(ctx, []string{strID})
			assert.Equal(t, httpErr, nil)
			assert.Equal(t, len(out), 2)
			assert.Equal(t, out[0].ID, strID)
			assert.Equal(t, out[0].Name, strID)
			assert.Equal(t, out[0].Account, strID1)
			assert.Equal(t, out[0].Email, strID1)
			assert.Equal(t, out[0].TelNumber, strID)
			assert.Equal(t, out[0].ThirdID, strID1)
			assert.Equal(t, out[1].ID, strAsc)
			assert.Equal(t, out[1].Name, strAsc)
			assert.Equal(t, out[1].Account, strAsc)
			assert.Equal(t, out[1].Email, strAsc)
			assert.Equal(t, out[1].TelNumber, strAsc)
			assert.Equal(t, out[1].ThirdID, strAsc)
		})
	})
}

func TestDeleteUserManagerID(t *testing.T) {
	Convey("DeleteUserManagerID, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		trace := mocks.NewMockTraceClient(ctrl)

		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		user := newUserDB(db)
		user.dbTrace = db
		user.trace = trace

		Convey("execute error", func() {
			mock.ExpectExec("").WillReturnError(errors.New(""))
			httpErr := user.DeleteUserManagerID(strID)
			assert.NotEqual(t, httpErr, nil)
		})

		Convey("success", func() {
			mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(1, 1))
			httpErr := user.DeleteUserManagerID(strID)
			assert.Equal(t, httpErr, nil)
		})
	})
}

func TestSearchUsers(t *testing.T) {
	Convey("SearchUsers, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		trace := mocks.NewMockTraceClient(ctrl)

		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		common.InitARTrace("test")

		user := newUserDB(db)
		user.dbTrace = db
		user.trace = trace

		ctx := context.Background()
		ks := interfaces.UserSearchInDepartKeyScope{}
		k := interfaces.UserSearchInDepartKey{}

		Convey("execute error", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			mock.ExpectQuery("").WillReturnError(errors.New(""))
			_, httpErr := user.SearchUsers(ctx, &ks, &k)
			assert.NotEqual(t, httpErr, nil)
		})

		ks.BDepartmentID = true
		ks.BCode = true
		fields := []string{"f_user_id", " f_login_name", "f_display_name", "f_remark", "f_csf_level", "f_auth_type", "f_priority", "f_create_time",
			"f_status", "f_auto_disable_status", "f_code", "f_manager_id", "f_position", "f_freeze_status", "f_csf_level2"}
		Convey("success", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow(strID, strID1, strAsc, strAsc, 11, 1, 12, 1234, 0, 1, strDesc, strID1, strAsc, 0, 13))
			out, httpErr := user.SearchUsers(ctx, &ks, &k)
			assert.Equal(t, httpErr, nil)
			assert.Equal(t, len(out), 1)

			assert.Equal(t, out[0].ID, strID)
			assert.Equal(t, out[0].Account, strID1)
			assert.Equal(t, out[0].Name, strAsc)
			assert.Equal(t, out[0].Remark, strAsc)
			assert.Equal(t, out[0].CSFLevel, 11)
			assert.Equal(t, out[0].AuthType, interfaces.Local)
			assert.Equal(t, out[0].Priority, 12)
			assert.Equal(t, out[0].DisableStatus, interfaces.Enabled)
			assert.Equal(t, out[0].AutoDisableStatus, interfaces.ADisabled)
			assert.Equal(t, out[0].Code, strDesc)
			assert.Equal(t, out[0].ManagerID, strID1)
			assert.Equal(t, out[0].Position, strAsc)
			assert.Equal(t, out[0].Frozen, false)
			assert.Equal(t, out[0].CSFLevel2, 13)
		})
	})
}

func TestSearchUsersCount(t *testing.T) {
	Convey("SearchUsersCount, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		trace := mocks.NewMockTraceClient(ctrl)

		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		common.InitARTrace("test")

		user := newUserDB(db)
		user.dbTrace = db
		user.trace = trace

		ctx := context.Background()
		ks := interfaces.UserSearchInDepartKeyScope{}
		k := interfaces.UserSearchInDepartKey{}

		Convey("execute error", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			mock.ExpectQuery("").WillReturnError(errors.New(""))
			_, httpErr := user.SearchUsersCount(ctx, &ks, &k)
			assert.NotEqual(t, httpErr, nil)
		})

		ks.BDepartmentID = true
		ks.BCode = true
		fields := []string{"count(*)"}
		Convey("success", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow(2))
			out, httpErr := user.SearchUsersCount(ctx, &ks, &k)
			assert.Equal(t, httpErr, nil)
			assert.Equal(t, out, 2)
		})
	})
}

func TestGetUserList(t *testing.T) {
	Convey("GetUserList, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		trace := mocks.NewMockTraceClient(ctrl)

		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		common.InitARTrace("test")

		user := newUserDB(db)
		user.dbTrace = db
		user.trace = trace

		ctx := context.Background()
		direction := interfaces.Desc
		createdStamp := int64(0)
		userID := ""
		limit := 0

		Convey("execute error", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			mock.ExpectQuery("").WillReturnError(errors.New(""))
			_, httpErr := user.GetUserList(ctx, direction, false, createdStamp, userID, limit)
			assert.NotEqual(t, httpErr, nil)
		})

		Convey("success", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_login_name", "f_display_name", "f_status",
				"f_auto_disable_status", "f_mail_address", "IFNULL(f_tel_number,'')", "f_create_time", "f_freeze_status"}).
				AddRow(strID, strID1, strAsc, 0, 1, strID1, strID, "2025-01-01 00:00:00", 1))
			out, httpErr := user.GetUserList(ctx, direction, true, createdStamp, userID, limit)
			assert.Equal(t, httpErr, nil)
			assert.Equal(t, len(out), 1)
			assert.Equal(t, out[0].ID, strID)
			assert.Equal(t, out[0].Account, strID1)
			assert.Equal(t, out[0].Name, strAsc)
			assert.Equal(t, out[0].DisableStatus, interfaces.Enabled)
			assert.Equal(t, out[0].AutoDisableStatus, interfaces.ADisabled)
			assert.Equal(t, out[0].Email, strID1)
			assert.Equal(t, out[0].TelNumber, strID)
			assert.Equal(t, out[0].CreatedAtTimeStamp, time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local).Unix())
			assert.Equal(t, out[0].Frozen, true)
		})
	})
}

func TestGetAllUserCount(t *testing.T) {
	Convey("GetAllUserCount, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		trace := mocks.NewMockTraceClient(ctrl)

		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		common.InitARTrace("test")

		user := newUserDB(db)
		user.dbTrace = db
		user.trace = trace

		ctx := context.Background()

		Convey("execute error", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			mock.ExpectQuery("").WillReturnError(errors.New(""))
			_, httpErr := user.GetAllUserCount(ctx)
			assert.NotEqual(t, httpErr, nil)
		})

		Convey("success", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).
				AddRow(1))
			out, httpErr := user.GetAllUserCount(ctx)
			assert.Equal(t, httpErr, nil)
			assert.Equal(t, out, 1)
		})
	})
}

func TestUserGetLock(t *testing.T) {
	Convey("GetLockConfig, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		trace := mocks.NewMockTraceClient(ctrl)

		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		common.InitARTrace("test")

		user := newUserDB(db)
		user.dbTrace = db
		user.trace = trace

		ctx := context.Background()

		Convey("execute error", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			mock.ExpectBegin()
			mock.ExpectQuery("").WillReturnError(errors.New("unknown"))
			mock.ExpectCommit()
			tx, err := db.Begin()
			assert.Equal(t, err, nil)

			err = user.GetLock(ctx, interfaces.UserExpiredDisableLock, tx)
			assert.NotEqual(t, err, nil)
		})

		Convey("success", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			mock.ExpectBegin()
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"f_value"}).AddRow("1"))
			mock.ExpectCommit()
			tx, err := db.Begin()
			assert.Equal(t, err, nil)

			err = user.GetLock(ctx, interfaces.UserExpiredDisableLock, tx)
			assert.Equal(t, err, nil)
		})
	})
}

func TestGetExpiredNeedDisableUserInfos(t *testing.T) {
	Convey("GetExpiredNeedDisableUserInfos, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		trace := mocks.NewMockTraceClient(ctrl)

		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		common.InitARTrace("test")

		user := newUserDB(db)
		user.dbTrace = db
		user.trace = trace

		ctx := context.Background()

		Convey("execute error", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			mock.ExpectBegin()
			mock.ExpectQuery("").WillReturnError(errors.New(""))
			mock.ExpectCommit()
			tx, err := db.Begin()
			assert.Equal(t, err, nil)

			_, httpErr := user.GetExpiredNeedDisableUserInfos(ctx, tx)
			assert.NotEqual(t, httpErr, nil)
		})

		Convey("success", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			mock.ExpectBegin()
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_display_name", "f_login_name"}).AddRow(strID, strAsc, strID1))
			mock.ExpectCommit()
			tx, err := db.Begin()
			assert.Equal(t, err, nil)

			userInfos, httpErr := user.GetExpiredNeedDisableUserInfos(ctx, tx)
			assert.Equal(t, httpErr, nil)
			assert.Equal(t, len(userInfos), 1)
			assert.Equal(t, userInfos[0].ID, strID)
			assert.Equal(t, userInfos[0].Name, strAsc)
			assert.Equal(t, userInfos[0].Account, strID1)
		})
	})
}

func TestGetNotLoginNeedDisableUserInfos(t *testing.T) {
	Convey("GetNotLoginNeedDisableUserInfos, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		trace := mocks.NewMockTraceClient(ctrl)

		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		common.InitARTrace("test")

		user := newUserDB(db)
		user.dbTrace = db
		user.trace = trace

		ctx := context.Background()

		Convey("execute error", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			mock.ExpectBegin()
			mock.ExpectQuery("").WillReturnError(errors.New(""))
			mock.ExpectCommit()
			tx, err := db.Begin()
			assert.Equal(t, err, nil)

			_, httpErr := user.GetNotLoginNeedDisableUserInfos(ctx, 0, tx)
			assert.NotEqual(t, httpErr, nil)
		})

		Convey("success", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			mock.ExpectBegin()
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_display_name", "f_login_name"}).AddRow(strID, strAsc, strID1))
			mock.ExpectCommit()
			tx, err := db.Begin()
			assert.Equal(t, err, nil)

			userInfos, httpErr := user.GetNotLoginNeedDisableUserInfos(ctx, 0, tx)
			assert.Equal(t, httpErr, nil)
			assert.Equal(t, len(userInfos), 1)
			assert.Equal(t, userInfos[0].ID, strID)
			assert.Equal(t, userInfos[0].Name, strAsc)
			assert.Equal(t, userInfos[0].Account, strID1)
		})
	})
}

func TestSetAutoDisableStatus(t *testing.T) {
	Convey("SetAutoDisableStatus, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		trace := mocks.NewMockTraceClient(ctrl)

		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		common.InitARTrace("test")

		user := newUserDB(db)
		user.dbTrace = db
		user.trace = trace

		ctx := context.Background()

		Convey("execute error", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			mock.ExpectBegin()
			mock.ExpectExec("").WillReturnError(errors.New(""))
			mock.ExpectCommit()
			tx, err := db.Begin()
			assert.Equal(t, err, nil)

			err = user.SetAutoDisableStatus(ctx, []string{strID}, interfaces.ExpiredDisabled, tx)
			assert.NotEqual(t, err, nil)
		})

		Convey("success", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			mock.ExpectBegin()
			mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()
			tx, err := db.Begin()
			assert.Equal(t, err, nil)

			err = user.SetAutoDisableStatus(ctx, []string{strID}, interfaces.ExpiredDisabled, tx)
			assert.Equal(t, err, nil)
		})
	})
}
