package dbaccess

import (
	"context"
	"errors"
	"testing"

	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/common"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/proton-rds-sdk-go/sqlx"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-playground/assert"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetMonthActiveUserInfo(t *testing.T) {
	Convey("GetMonthActiveUserInfo, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		common.InitARTrace("test")

		user := &activeUser{
			dbTrace: db,
			logger:  common.NewLogger(),
			trace:   common.SvcARTrace,
		}

		Convey("error", func() {
			testErr := errors.New("error")
			mock.ExpectQuery("").WillReturnError(testErr)
			activeUserInfos, err := user.GetMonthActiveUserInfo(context.Background(), 2026, 1)
			assert.Equal(t, err, testErr)
			assert.Equal(t, len(activeUserInfos), 0)
		})

		Convey("success", func() {
			fields := []string{"f_active_count", "f_activate_count", "f_time"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow(100, 100, "2026-01-01"))
			activeUserInfos, err := user.GetMonthActiveUserInfo(context.Background(), 2026, 1)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(activeUserInfos), 1)
			assert.Equal(t, activeUserInfos[0].ActiveCount, 100)
			assert.Equal(t, activeUserInfos[0].ActivateCount, 100)
			assert.Equal(t, activeUserInfos[0].Year, 2026)
			assert.Equal(t, activeUserInfos[0].Month, 1)
			assert.Equal(t, activeUserInfos[0].Day, 1)
		})
	})
}

func TestGetYearActiveUserInfo(t *testing.T) {
	Convey("GetYearActiveUserInfo, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		common.InitARTrace("test")

		user := &activeUser{
			dbTrace: db,
			logger:  common.NewLogger(),
			trace:   common.SvcARTrace,
		}

		Convey("error", func() {
			testErr := errors.New("error")
			mock.ExpectQuery("").WillReturnError(testErr)
			activeUserInfos, err := user.GetYearActiveUserInfo(context.Background(), 2026)
			assert.Equal(t, err, testErr)
			assert.Equal(t, len(activeUserInfos), 0)
		})

		Convey("success", func() {
			fields := []string{"f_active_count", "f_activate_count", "f_time"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow(100, 100, "2026-01"))
			activeUserInfos, err := user.GetYearActiveUserInfo(context.Background(), 2026)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(activeUserInfos), 1)
			assert.Equal(t, activeUserInfos[0].ActiveCount, 100)
			assert.Equal(t, activeUserInfos[0].ActivateCount, 100)
			assert.Equal(t, activeUserInfos[0].Year, 2026)
			assert.Equal(t, activeUserInfos[0].Month, 1)
		})
	})
}

func TestGetMonthTotalCount(t *testing.T) {
	Convey("GetMonthTotalCount, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		common.InitARTrace("test")

		user := &activeUser{
			dbTrace: db,
			logger:  common.NewLogger(),
			trace:   common.SvcARTrace,
		}

		Convey("error", func() {
			testErr := errors.New("error")
			mock.ExpectQuery("").WillReturnError(testErr)
			count, activeteCount, err := user.GetMonthTotalCount(context.Background(), 2026, 1)
			assert.Equal(t, err, testErr)
			assert.Equal(t, count, 0)
			assert.Equal(t, activeteCount, 0)
		})

		Convey("success", func() {
			fields := []string{"f_total_count", "f_activate_count"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow(100, 100))
			count, activeteCount, err := user.GetMonthTotalCount(context.Background(), 2026, 1)
			assert.Equal(t, err, nil)
			assert.Equal(t, count, 100)
			assert.Equal(t, activeteCount, 100)
		})
	})
}

func TestGetYearTotalCount(t *testing.T) {
	Convey("GetYearTotalCount, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		common.InitARTrace("test")

		user := &activeUser{
			dbTrace: db,
			logger:  common.NewLogger(),
			trace:   common.SvcARTrace,
		}

		Convey("error", func() {
			testErr := errors.New("error")
			mock.ExpectQuery("").WillReturnError(testErr)
			count, activeteCount, err := user.GetYearTotalCount(context.Background(), 2026)
			assert.Equal(t, err, testErr)
			assert.Equal(t, count, 0)
			assert.Equal(t, activeteCount, 0)
		})

		Convey("success", func() {
			fields := []string{"f_total_count", "f_activate_count"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow(100, 100))
			count, activeteCount, err := user.GetYearTotalCount(context.Background(), 2026)
			assert.Equal(t, err, nil)
			assert.Equal(t, count, 100)
			assert.Equal(t, activeteCount, 100)
		})
	})
}

func TestGetActivateUserCount(t *testing.T) {
	Convey("GetActivateUserCount, db is available", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)

		common.InitARTrace("test")

		user := &activeUser{
			dbTrace: db,
			logger:  common.NewLogger(),
			trace:   common.SvcARTrace,
		}

		Convey("error", func() {
			testErr := errors.New("error")
			mock.ExpectQuery("").WillReturnError(testErr)
			count, err := user.GetActivateUserCount(context.Background())
			assert.Equal(t, err, testErr)
			assert.Equal(t, count, 0)
		})

		Convey("success", func() {
			fields := []string{"count(*)"}
			mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows(fields).AddRow(100))
			count, err := user.GetActivateUserCount(context.Background())
			assert.Equal(t, err, nil)
			assert.Equal(t, count, 100)
		})
	})
}
