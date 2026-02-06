package dbaccess

import (
	"context"
	"errors"
	"testing"

	"github.com/kweaver-ai/proton-rds-sdk-go/sqlx"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-playground/assert"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"

	"UserManagement/common"
	"UserManagement/interfaces"
	mocks "UserManagement/interfaces/mock"
)

var (
	testDepartID = "testDepartID"
)

func newDepartmentDB(ptrDB *sqlx.DB) *department {
	return &department{
		db:     ptrDB,
		logger: common.NewLogger(),
	}
}

func TestNewDepartment(t *testing.T) {
	Convey("NewDepartment", t, func() {
		data := NewDepartment()
		assert.NotEqual(t, data, nil)
	})
}

func TestGetDepartmentName(t *testing.T) {
	Convey("GetDepartmentName, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		department := newDepartmentDB(db)

		deptIDs := make([]string, 0)

		Convey("department not exist", func() {
			fields := []string{
				"f_department_id",
				"f_name",
			}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields))
			infoMap, name, httpErr := department.GetDepartmentName(deptIDs)
			assert.Equal(t, len(name), 0)
			assert.Equal(t, httpErr, nil)
			assert.Equal(t, len(infoMap), 0)
		})

		Convey("success", func() {
			deptIDs = append(deptIDs, "test")

			fields := []string{
				"f_department_id",
				"f_name",
			}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow("test", "f_name"))
			infoMap, name, httpErr := department.GetDepartmentName(deptIDs)
			assert.Equal(t, httpErr, nil)
			tempNameInfo := interfaces.NameInfo{
				ID:   "test",
				Name: "f_name",
			}
			assert.Equal(t, infoMap, []interfaces.NameInfo{tempNameInfo})
			assert.Equal(t, name, []string{"test"})
		})
	})
}

func TestGetParentDepartmentID(t *testing.T) {
	Convey("GetParentDepartmentID, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		department := newDepartmentDB(db)

		deptIDs := make([]string, 0)

		deptIDs = append(deptIDs, "test")

		Convey("no parentDepartment", func() {
			fields := []string{"f_department_id"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields))
			directIDs, httpErr := department.GetParentDepartmentID(deptIDs)
			assert.Equal(t, len(directIDs), 0)
			assert.Equal(t, httpErr, nil)
		})

		Convey("success", func() {
			deptIDs = append(deptIDs, "test")

			fields := []string{
				"f_parent_department_id",
			}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow(""))
			_, httpErr := department.GetParentDepartmentID(deptIDs)
			assert.Equal(t, httpErr, nil)
		})
	})
}

func TestGetChildDepartmentIDs(t *testing.T) {
	Convey("GetChildDepartmentIDs, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		department := newDepartmentDB(db)

		var deptIDs []string

		Convey("no deptIDs", func() {
			directIDs, depMapInfo, httpErr := department.GetChildDepartmentIDs(deptIDs)
			assert.Equal(t, len(directIDs), 0)
			assert.Equal(t, httpErr, nil)
			assert.Equal(t, len(depMapInfo), 0)
		})

		deptIDs = append(deptIDs, "test")

		Convey("no data", func() {
			fields := []string{"f_department_id"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields))
			directIDs, depMapInfo, httpErr := department.GetChildDepartmentIDs(deptIDs)
			assert.Equal(t, len(directIDs), 0)
			assert.Equal(t, httpErr, nil)
			assert.Equal(t, len(depMapInfo), 0)
		})

		Convey("success", func() {
			fields := []string{
				"f_department_id",
				"f_parent_department_id",
			}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow(strID, strID1))
			directIDs, depMapInfo, httpErr := department.GetChildDepartmentIDs(deptIDs)
			assert.Equal(t, httpErr, nil)
			assert.Equal(t, len(depMapInfo), 1)
			assert.Equal(t, depMapInfo[strID1], []string{strID})
			assert.Equal(t, len(directIDs), 1)
			assert.Equal(t, directIDs[0], strID)
		})
	})
}

func TestGetChildDepartmentIDs2(t *testing.T) {
	Convey("GetChildDepartmentIDs2, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		trace := mocks.NewMockTraceClient(ctrl)
		ctx := context.Background()

		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		department := newDepartmentDB(db)
		department.dbTrace = db
		department.trace = trace

		var deptIDs []string
		Convey("no deptIDs", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			directIDs, depMapInfo, httpErr := department.GetChildDepartmentIDs2(ctx, deptIDs)
			assert.Equal(t, len(directIDs), 0)
			assert.Equal(t, httpErr, nil)
			assert.Equal(t, len(depMapInfo), 0)
		})

		deptIDs = append(deptIDs, "test")

		Convey("no data", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			fields := []string{"f_department_id"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields))
			directIDs, depMapInfo, httpErr := department.GetChildDepartmentIDs2(ctx, deptIDs)
			assert.Equal(t, len(directIDs), 0)
			assert.Equal(t, httpErr, nil)
			assert.Equal(t, len(depMapInfo), 0)
		})

		Convey("success", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			fields := []string{
				"f_department_id",
				"f_parent_department_id",
			}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow(strID, strID1))
			directIDs, depMapInfo, httpErr := department.GetChildDepartmentIDs2(ctx, deptIDs)
			assert.Equal(t, httpErr, nil)
			assert.Equal(t, len(depMapInfo), 1)
			assert.Equal(t, depMapInfo[strID1], []string{strID})
			assert.Equal(t, len(directIDs), 1)
			assert.Equal(t, directIDs[0], strID)
		})
	})
}

func TestGetChildUserIDs(t *testing.T) {
	Convey("GetChildUserIDs, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		department := newDepartmentDB(db)
		var depID string
		Convey("no data", func() {
			fields := []string{"f_department_id"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields))
			directIDs, userMaps, httpErr := department.GetChildUserIDs([]string{depID})
			assert.Equal(t, len(directIDs), 0)
			assert.Equal(t, len(userMaps), 0)
			assert.Equal(t, httpErr, nil)
		})

		Convey("success", func() {
			fields := []string{
				"f_user_id",
				"f_department_id",
			}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow(strID, strID1))
			directIDs, userMaps, httpErr := department.GetChildUserIDs([]string{depID})
			assert.Equal(t, httpErr, nil)
			assert.Equal(t, len(directIDs), 1)
			assert.Equal(t, directIDs[0], strID)
			assert.Equal(t, len(userMaps), 1)
			assert.Equal(t, userMaps[strID1], []string{strID})
		})
	})
}

func TestGetAllRootDeps(t *testing.T) {
	Convey("GetAllRootDeps, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		department := newDepartmentDB(db)
		Convey("no data", func() {
			fields := []string{"f_department_id", "f_name"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields))
			directIDs, httpErr := department.GetRootDeps(true, false, []string{}, 0, 10)
			assert.Equal(t, len(directIDs), 0)
			assert.Equal(t, httpErr, nil)
		})

		Convey("success", func() {
			fields := []string{
				"f_department_id",
				"f_name",
				"f_is_enterprise",
				"f_code",
				"f_remark",
				"f_manager_id",
				"f_status",
				"f_mail_address",
			}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow("", "", 1, "", "", "", 1, ""))
			_, httpErr := department.GetRootDeps(false, true, []string{"xxx"}, 0, 10)
			assert.Equal(t, httpErr, nil)
		})
	})
}

func TestGetSubDepartmentInfos(t *testing.T) {
	Convey("GetSubDepartmentInfos, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		depPath := ""

		department := newDepartmentDB(db)
		Convey("no data", func() {
			fields := []string{"f_department_id", "f_name", "f_is_enterprise"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields))
			directIDs, httpErr := department.GetSubDepartmentInfos(depPath, false, 0, 10)
			assert.Equal(t, len(directIDs), 0)
			assert.Equal(t, httpErr, nil)
		})

		Convey("success", func() {
			fields := []string{
				"f_department_id",
				"f_name",
				"f_is_enterprise",
				"f_code",
				"f_remark",
				"f_manager_id",
				"f_status",
				"f_mail_address",
			}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow("", "", 1, "", "", "", 1, ""))
			_, httpErr := department.GetSubDepartmentInfos(depPath, true, 0, 10)
			assert.Equal(t, httpErr, nil)
		})
	})
}

func TestGetSubUserInfos(t *testing.T) {
	Convey("GetSubDepartmentInfos, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		depID := ""

		department := newDepartmentDB(db)
		Convey("no data", func() {
			fields := []string{"f_user_id", "f_display_name", "f_status", "f_priority", "f_csf_level", "f_auto_disable_status"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields))
			directIDs, httpErr := department.GetSubUserInfos(depID, false, 0, 10)
			assert.Equal(t, len(directIDs), 0)
			assert.Equal(t, httpErr, nil)
		})

		Convey("success", func() {
			fields := []string{"f_user_id", "f_display_name", "f_status", "f_priority", "f_csf_level", "f_auto_disable_status"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow("", "", 1, 2, 3, 4))
			_, httpErr := department.GetSubUserInfos(depID, true, 0, 10)
			assert.Equal(t, httpErr, nil)
		})
	})
}

func TestGetDepartmentInfo(t *testing.T) {
	Convey("GetDepartmentInfo, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		depID := ""

		department := newDepartmentDB(db)
		fields := []string{"f_department_id", "f_name", "f_is_enterprise", "f_mail_address", "f_path", "f_manager_id", "f_code", "f_status", "f_third_party_id"}
		Convey("no data", func() {
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields))
			directIDs, httpErr := department.GetDepartmentInfo([]string{depID}, false, 0, 10)
			assert.Equal(t, len(directIDs), 0)
			assert.Equal(t, httpErr, nil)
		})

		Convey("success", func() {
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow("", "", 1, "", "", "", "", 1, ""))
			_, httpErr := department.GetDepartmentInfo([]string{depID}, true, 0, 10)
			assert.Equal(t, httpErr, nil)
		})
	})
}

func TestGetDepartmentInfo2(t *testing.T) {
	Convey("GetDepartmentInfo, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		trace := mocks.NewMockTraceClient(ctrl)
		ctx := context.Background()

		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		depID := ""
		department := newDepartmentDB(db)
		department.dbTrace = db
		department.trace = trace

		fields := []string{"f_department_id", "f_name", "f_is_enterprise", "f_mail_address", "f_path", "f_code"}
		Convey("no data", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields))
			directIDs, httpErr := department.GetDepartmentInfo2(ctx, []string{depID}, false, 0, 10)
			assert.Equal(t, len(directIDs), 0)
			assert.Equal(t, httpErr, nil)
		})

		Convey("success", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow("", "", 1, "", "", ""))
			_, httpErr := department.GetDepartmentInfo2(ctx, []string{depID}, true, 0, 10)
			assert.Equal(t, httpErr, nil)
		})
	})
}

func TestGetDepartmentInfoByIDs(t *testing.T) {
	Convey("GetDepartmentInfoByIDs, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		trace := mocks.NewMockTraceClient(ctrl)
		ctx := context.Background()

		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		depID := ""
		department := newDepartmentDB(db)
		department.dbTrace = db
		department.trace = trace

		fields := []string{"f_department_id", "f_name", "f_is_enterprise", "f_mail_address", "f_path", "f_code"}
		Convey("no data", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields))
			directIDs, httpErr := department.GetDepartmentInfoByIDs(ctx, []string{depID})
			assert.Equal(t, len(directIDs), 0)
			assert.Equal(t, httpErr, nil)
		})

		Convey("success", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow("", "", 1, "", "", ""))
			_, httpErr := department.GetDepartmentInfoByIDs(ctx, []string{depID})
			assert.Equal(t, httpErr, nil)
		})
	})
}

func TestSearchDepartsByKey(t *testing.T) {
	Convey("SearchDepartsByKey, db is available", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		trace := mocks.NewMockTraceClient(ctrl)
		ctx := context.Background()

		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		department := newDepartmentDB(db)
		department.trace = trace
		department.dbTrace = db
		Convey("no data", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			fields := []string{"f_department_id", "f_name", "f_path"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields))
			directIDs, httpErr := department.SearchDepartsByKey(ctx, true, false, nil, "xxx", 0, 10)
			assert.Equal(t, len(directIDs), 0)
			assert.Equal(t, httpErr, nil)
		})

		Convey("success", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			fields := []string{"f_department_id", "f_name", "f_path"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow("", "", ""))
			_, httpErr := department.SearchDepartsByKey(ctx, false, true, nil, "xxxxx", 0, 10)
			assert.Equal(t, httpErr, nil)
		})
	})
}

func TestGetScopeRootDeps(t *testing.T) {
	Convey("GetScopeRootDeps, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		department := newDepartmentDB(db)
		Convey("no data", func() {
			fields := []string{"f_department_id", "f_name", "f_is_enterprise", "f_code", "f_remark", "f_manager_id", "f_status", "f_mail_address"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields))
			directIDs, httpErr := department.GetRootDeps(true, false, []string{}, 0, 10)
			assert.Equal(t, len(directIDs), 0)
			assert.Equal(t, httpErr, nil)
		})

		Convey("success", func() {
			fields := []string{"f_department_id", "f_name", "f_is_enterprise", "f_code", "f_remark", "f_manager_id", "f_status", "f_mail_address"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow("", "", 1, "", "", "", 1, ""))
			_, httpErr := department.GetRootDeps(false, true, nil, 0, 10)
			assert.Equal(t, httpErr, nil)
		})
	})
}

func TestGetManagersOfDepartment(t *testing.T) {
	Convey("GetManagersOfDepartment, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		department := newDepartmentDB(db)

		Convey("success", func() {
			departmentIDs := []string{"a", "b"}
			fields := []string{"f_department_id", "f_user_id", "f_display_name"}

			tmpManager := []interfaces.NameInfo{
				{
					ID:   "cccc",
					Name: "name1",
				},
				{
					ID:   "dddd",
					Name: "name2",
				},
			}

			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow("1111", "cccc", "name1").AddRow("1111", "dddd", "name2"))
			result, err := department.GetManagersOfDepartment(departmentIDs)
			assert.Equal(t, err, nil)
			assert.Equal(t, result[0].Managers, tmpManager)
		})
	})
}

func TestGetDepartmentByPathLength(t *testing.T) {
	Convey("GetDepartmentByPathLength, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		department := newDepartmentDB(db)
		Convey("query failed", func() {
			mock.ExpectQuery("").WillReturnError(errors.New("xxx"))
			_, err := department.GetDepartmentByPathLength(0)
			assert.NotEqual(t, err, nil)
		})

		Convey("success", func() {
			fields := []string{"f_department_id", "f_name", "f_third_party_id"}

			tmpManager := []interfaces.DepartmentDBInfo{{ID: "cccc", Name: "name1", ThirdID: "third_id1"}, {ID: "dddd", Name: "name2", ThirdID: "third_id2"}}

			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow("cccc", "name1", "third_id1").AddRow("dddd", "name2", "third_id2"))
			result, err := department.GetDepartmentByPathLength(0)
			assert.Equal(t, err, nil)
			assert.Equal(t, result[0], tmpManager[0])
			assert.Equal(t, result[1], tmpManager[1])
		})
	})
}

func TestGetAllSubUserIDsByDepartPath(t *testing.T) {
	Convey("GetAllSubUserIDsByDepartPath, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		department := newDepartmentDB(db)
		Convey("query failed", func() {
			mock.ExpectQuery("").WillReturnError(errors.New("xxx"))
			_, err := department.GetAllSubUserIDsByDepartPath("")
			assert.NotEqual(t, err, nil)
		})

		Convey("success", func() {
			fields := []string{"f_user_id"}

			tmpManager := []string{"xx", "zzz"}

			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow(tmpManager[0]).AddRow(tmpManager[1]))
			result, err := department.GetAllSubUserIDsByDepartPath("")
			assert.Equal(t, err, nil)
			assert.Equal(t, result[0], tmpManager[0])
			assert.Equal(t, result[1], tmpManager[1])
		})
	})
}

func TestGetAllSubUserInfosByDepartPath(t *testing.T) {
	Convey("GetAllSubUserInfosByDepartPath, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		department := newDepartmentDB(db)
		Convey("query failed", func() {
			mock.ExpectQuery("").WillReturnError(errors.New("xxx"))
			_, err := department.GetAllSubUserInfosByDepartPath("")
			assert.NotEqual(t, err, nil)
		})

		Convey("success", func() {
			fields := []string{
				"f_user_id",
				"f_login_name",
				"f_display_name",
				"f_mail_address",
				"f_tel_number",
				"f_third_party_attr",
				"f_third_party_id",
			}

			outInfo := interfaces.UserBaseInfo{
				ID:        "id1",
				Account:   "account1",
				Name:      "name1",
				Email:     "emails",
				TelNumber: "telnums",
				ThirdAttr: "trhirdatrr",
				ThirdID:   "thirdID",
			}

			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow(outInfo.ID, outInfo.Account, outInfo.Name,
				outInfo.Email, outInfo.TelNumber, outInfo.ThirdAttr, outInfo.ThirdID))
			result, err := department.GetAllSubUserInfosByDepartPath("")
			assert.Equal(t, err, nil)
			assert.Equal(t, len(result), 1)
			assert.Equal(t, result[0].ID, outInfo.ID)
			assert.Equal(t, result[0].Name, outInfo.Name)
			assert.Equal(t, result[0].Account, outInfo.Account)
			assert.Equal(t, result[0].TelNumber, outInfo.TelNumber)
			assert.Equal(t, result[0].Email, outInfo.Email)
			assert.Equal(t, result[0].ThirdAttr, outInfo.ThirdAttr)
			assert.Equal(t, result[0].ThirdID, outInfo.ThirdID)
		})
	})
}

func TestDeleteOrgManagerRelationByDepartID(t *testing.T) {
	Convey("DeleteOrgManagerRelationByDepartID, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		department := newDepartmentDB(db)

		Convey("error", func() {
			mock.ExpectExec("").WillReturnError(errors.New(""))

			err := department.DeleteOrgManagerRelationByDepartID(testDepartID)
			assert.NotEqual(t, err, nil)
		})

		Convey("DeleteOrgManagerRelationByDepartID successfully", func() {
			mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(1, 1))

			err := department.DeleteOrgManagerRelationByDepartID(testDepartID)
			assert.Equal(t, err, nil)
		})
	})
}

func TestDeleteOrgAuditRelationByDepartID(t *testing.T) {
	Convey("DeleteOrgAuditRelationByDepartID, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		department := newDepartmentDB(db)

		Convey("error", func() {
			mock.ExpectExec("").WillReturnError(errors.New(""))

			err := department.DeleteOrgAuditRelationByDepartID(testDepartID)
			assert.NotEqual(t, err, nil)
		})

		Convey("DeleteOrgAuditRelationByDepartID successfully", func() {
			mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(1, 1))

			err := department.DeleteOrgAuditRelationByDepartID(testDepartID)
			assert.Equal(t, err, nil)
		})
	})
}

func TestDeleteUserDepartRelationByPath(t *testing.T) {
	Convey("DeleteUserDepartRelationByPath, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		department := newDepartmentDB(db)

		Convey("error", func() {
			mock.ExpectBegin()
			tx, err := db.Begin()
			assert.Equal(t, err, nil)

			mock.ExpectExec("").WillReturnError(errors.New(""))

			err = department.DeleteUserDepartRelationByPath(testDepartID, tx)

			assert.Equal(t, err, errors.New(""))
		})

		Convey("DeleteUserDepartRelationByPath successfully", func() {
			mock.ExpectBegin()
			tx, err := db.Begin()
			assert.Equal(t, err, nil)

			mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(1, 1))

			err = department.DeleteUserDepartRelationByPath(testDepartID, tx)

			assert.Equal(t, err, nil)
		})
	})
}

func TestAddUserToDepart(t *testing.T) {
	Convey("AddUserToDepart, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		department := newDepartmentDB(db)

		Convey("AddUserToDepart userid is nil", func() {
			err = department.AddUserToDepart(nil, testDepartID, "", nil)
			assert.Equal(t, err, nil)
		})

		testUserIDs := []string{userID}
		Convey("AddUserToDepart error", func() {
			mock.ExpectBegin()
			tx, err := db.Begin()
			assert.Equal(t, err, nil)

			mock.ExpectExec("").WillReturnError(errors.New(""))

			err = department.AddUserToDepart(testUserIDs, testDepartID, "", tx)
			assert.NotEqual(t, err, nil)
		})

		Convey("AddUserToDepart successfully", func() {
			mock.ExpectBegin()
			tx, err := db.Begin()
			assert.Equal(t, err, nil)

			mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(1, 1))

			err = department.AddUserToDepart(testUserIDs, testDepartID, "", tx)
			assert.Equal(t, err, nil)
		})
	})
}

func TestDeleteUserOURelation(t *testing.T) {
	Convey("DeleteUserOURelation, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		department := newDepartmentDB(db)

		Convey("DeleteUserOURelation userid is nil", func() {
			err = department.DeleteUserOURelation(nil, testDepartID, nil)

			assert.Equal(t, err, nil)
		})

		testUserIDs := []string{userID}

		Convey("DeleteUserOURelation error", func() {
			mock.ExpectBegin()
			tx, err := db.Begin()
			assert.Equal(t, err, nil)

			mock.ExpectExec("").WillReturnError(errors.New(""))

			err = department.DeleteUserOURelation(testUserIDs, testDepartID, tx)
			assert.Equal(t, err, errors.New(""))
		})

		Convey("DeleteUserOURelation successfully", func() {
			mock.ExpectBegin()
			tx, err := db.Begin()
			assert.Equal(t, err, nil)

			mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(1, 1))

			err = department.DeleteUserOURelation(testUserIDs, testDepartID, tx)
			assert.Equal(t, err, nil)
		})
	})
}

func TestDeleteDepartByPath(t *testing.T) {
	Convey("DeleteDepartByPath, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		department := newDepartmentDB(db)

		Convey("DeleteDepartByPath error", func() {
			mock.ExpectBegin()
			tx, err := db.Begin()
			assert.Equal(t, err, nil)

			mock.ExpectExec("").WillReturnError(errors.New(""))

			err = department.DeleteDepartByPath(testDepartID, tx)
			assert.NotEqual(t, err, nil)
		})

		Convey("DeleteDepartByPath successfully", func() {
			mock.ExpectBegin()
			tx, err := db.Begin()
			assert.Equal(t, err, nil)

			mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(1, 1))

			err = department.DeleteDepartByPath(testDepartID, tx)
			assert.Equal(t, err, nil)
		})
	})
}

func TestGetAllSubDepartIDsByPath(t *testing.T) {
	Convey("GetAllSubDepartIDsByPath, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		department := newDepartmentDB(db)
		Convey("query failed", func() {
			mock.ExpectQuery("").WillReturnError(errors.New("xxx"))
			_, err := department.GetAllSubDepartInfosByPath("")
			assert.NotEqual(t, err, nil)
		})

		Convey("successful", func() {
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"f_department_id", "f_path"}).AddRow(userID, userID).AddRow(testDepartID, testDepartID))
			out, err := department.GetAllSubDepartInfosByPath("")
			assert.Equal(t, err, nil)
			assert.Equal(t, len(out), 2)
			assert.Equal(t, out[0].ID, userID)
			assert.Equal(t, out[0].Path, userID)
			assert.Equal(t, out[1].ID, testDepartID)
			assert.Equal(t, out[1].Path, testDepartID)
		})
	})
}

func TestDeleteDepartRelations(t *testing.T) {
	Convey("DeleteDepartRelations, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		department := newDepartmentDB(db)

		Convey("DeleteDepartRelations userid is nil", func() {
			err = department.DeleteDepartRelations(nil, nil)

			assert.Equal(t, err, nil)
		})

		testUserIDs := []string{userID}
		Convey("DeleteDepartRelations error", func() {
			mock.ExpectBegin()
			tx, err := db.Begin()
			assert.Equal(t, err, nil)

			mock.ExpectExec("").WillReturnError(errors.New(userID))

			err = department.DeleteDepartRelations(testUserIDs, tx)
			assert.Equal(t, err, errors.New(userID))
		})

		Convey("DeleteDepartRelations successfully", func() {
			mock.ExpectBegin()
			tx, err := db.Begin()
			assert.Equal(t, err, nil)

			mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(1, 1))

			err = department.DeleteDepartRelations(testUserIDs, tx)
			assert.Equal(t, err, nil)
		})
	})
}

func TestDeleteDepartOURelations(t *testing.T) {
	Convey("DeleteDepartOURelations, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		department := newDepartmentDB(db)

		Convey("DeleteDepartOURelations userid is nil", func() {
			err = department.DeleteDepartOURelations(nil, nil)

			assert.Equal(t, err, nil)
		})

		testUserIDs := []string{userID}
		Convey("DeleteDepartOURelations error", func() {
			mock.ExpectBegin()
			tx, err := db.Begin()
			assert.Equal(t, err, nil)

			mock.ExpectExec("").WillReturnError(errors.New(""))
			err = department.DeleteDepartOURelations(testUserIDs, tx)
			assert.NotEqual(t, err, nil)
		})

		Convey("DeleteDepartOURelations successfully", func() {
			mock.ExpectBegin()
			tx, err := db.Begin()
			assert.Equal(t, err, nil)

			mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(1, 1))

			err = department.DeleteDepartOURelations(testUserIDs, tx)
			assert.Equal(t, err, nil)
		})
	})
}

func TestGetAllOrgManagerIDsByDepartIDs(t *testing.T) {
	Convey("GetAllOrgManagerIDsByDepartIDs, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		department := newDepartmentDB(db)

		Convey("GetAllOrgManagerIDsByDepartIDs departIds is nil", func() {
			_, err = department.GetAllOrgManagerIDsByDepartIDs(nil)

			assert.Equal(t, err, nil)
		})

		Convey("sql error", func() {
			mock.ExpectQuery("").WillReturnError(errors.New(""))
			_, err = department.GetAllOrgManagerIDsByDepartIDs([]string{userID})

			assert.NotEqual(t, err, nil)
		})

		Convey("success ", func() {
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"f_user_id"}).AddRow(userID).AddRow(userID))
			out, err := department.GetAllOrgManagerIDsByDepartIDs([]string{userID})

			assert.Equal(t, err, nil)
			assert.Equal(t, len(out), 2)
			assert.Equal(t, out[0], userID)
			assert.Equal(t, out[1], userID)
		})
	})
}

func TestGetAllOrgManagerIDs(t *testing.T) {
	Convey("GetAllOrgManagerIDs, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		department := newDepartmentDB(db)

		Convey("GetAllOrgManagerIDs sql error", func() {
			mock.ExpectQuery("").WillReturnError(errors.New(""))
			_, err = department.GetAllOrgManagerIDs()

			assert.NotEqual(t, err, nil)
		})

		Convey("success ", func() {
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"f_user_id"}).AddRow(userID))
			out, err := department.GetAllOrgManagerIDs()

			assert.Equal(t, err, nil)
			assert.Equal(t, len(out), 1)
			assert.Equal(t, out[0], userID)
		})
	})
}

func TestDeleteDocAutoCleanStrategy(t *testing.T) {
	Convey("DeleteDocAutoCleanStrategy, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		department := newDepartmentDB(db)

		Convey("sql error", func() {
			mock.ExpectExec("").WillReturnError(errors.New(""))
			err = department.DeleteDocAutoCleanStrategy(userID)

			assert.NotEqual(t, err, nil)
		})

		Convey("sql success", func() {
			mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(1, 1))
			err = department.DeleteDocAutoCleanStrategy(userID)

			assert.Equal(t, err, nil)
		})
	})
}

// 编写测试用例测试DeleteDepartManager
func TestDeleteDepartManager(t *testing.T) {
	Convey("DeleteDepartManager, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		department := newDepartmentDB(db)
		Convey("sql error", func() {
			mock.ExpectExec("").WillReturnError(errors.New(""))
			err = department.DeleteDepartManager(userID)

			assert.NotEqual(t, err, nil)
		})

		Convey("sql success", func() {
			mock.ExpectExec("").WillReturnResult(sqlmock.NewResult(1, 1))
			err = department.DeleteDepartManager(userID)

			assert.Equal(t, err, nil)
		})
	})
}

func TestSearchDeparts(t *testing.T) {
	Convey("SearchDeparts, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)
		ctrl := gomock.NewController(t)

		trace := mocks.NewMockTraceClient(ctrl)
		ctx := context.Background()

		department := newDepartmentDB(db)
		department.dbTrace = db
		department.trace = trace

		ks := &interfaces.DepartSearchKeyScope{BCode: true}
		k := &interfaces.DepartSearchKey{Code: "test", Offset: 0, Limit: 10}

		Convey("sql error", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			mock.ExpectQuery("").WillReturnError(errors.New(""))
			_, err = department.SearchDeparts(ctx, ks, k, nil)

			assert.NotEqual(t, err, nil)
		})

		Convey("sql success", func() {
			ks.BCode = false
			ks.BName = true
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			fields := []string{"f_department_id", "f_path", "f_name", "f_code", "f_manager_id", "f_remark", "f_status", "f_mail_address"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow("1", "path", "name", "code", "manager_id", "remark", 1, "email"))

			out, err := department.SearchDeparts(ctx, ks, k, []string{userID})
			assert.Equal(t, err, nil)
			assert.Equal(t, len(out), 1)
			assert.Equal(t, out[0].ID, "1")
			assert.Equal(t, out[0].Name, "name")
			assert.Equal(t, out[0].Code, "code")
			assert.Equal(t, out[0].ManagerID, "manager_id")
			assert.Equal(t, out[0].Remark, "remark")
			assert.Equal(t, out[0].Email, "email")
			assert.Equal(t, out[0].Status, true)
		})
	})
}

func TestSearchDepartsCount(t *testing.T) {
	Convey("SearchDepartsCount, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)
		ctrl := gomock.NewController(t)

		trace := mocks.NewMockTraceClient(ctrl)
		ctx := context.Background()

		department := newDepartmentDB(db)
		department.dbTrace = db
		department.trace = trace

		ks := &interfaces.DepartSearchKeyScope{BCode: true}
		k := &interfaces.DepartSearchKey{Code: "test", Offset: 0, Limit: 10}

		Convey("sql error", func() {
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			mock.ExpectQuery("").WillReturnError(errors.New(""))
			_, err = department.SearchDepartsCount(ctx, ks, k, nil)

			assert.NotEqual(t, err, nil)
		})

		Convey("sql success", func() {
			ks.BCode = false
			ks.BName = true
			trace.EXPECT().SetClientSpanName(gomock.Any()).AnyTimes()
			trace.EXPECT().AddClientTrace(gomock.Any()).AnyTimes().Return(ctx, nil)
			trace.EXPECT().TelemetrySpanEnd(gomock.Any(), gomock.Any()).AnyTimes()

			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

			out, err := department.SearchDepartsCount(ctx, ks, k, []string{userID})
			assert.Equal(t, err, nil)
			assert.Equal(t, out, 1)
		})
	})
}
