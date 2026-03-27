package dbaccess // active_user Anyshare 数据访问层 - 活跃用户数据库操作

import (
	"context"
	"fmt"
	"sync"
	"time"

	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/common"
	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/interfaces"
	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/go-lib/observable"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/proton-rds-sdk-go/sqlx"
)

var (
	auOnce sync.Once
	au     *activeUser
)

type activeUser struct {
	logger  common.Logger
	trace   observable.Tracer
	dbTrace *sqlx.DB
}

// NewActiveUser 创建数据库操作对象
func NewActiveUser() *activeUser {
	auOnce.Do(func() {
		au = &activeUser{
			logger:  common.NewLogger(),
			trace:   common.SvcARTrace,
			dbTrace: dbTracePool,
		}
	})
	return au
}

// GetMonthActiveUserInfo 获取当前月度活跃用户信息
func (a *activeUser) GetMonthActiveUserInfo(ctx context.Context, year, month int) (out []interfaces.ActiveUserInfo, err error) {
	// trace
	a.trace.SetClientSpanName("数据库操作-获取当前月度活跃用户信息")
	newCtx, span := a.trace.AddClientTrace(ctx)
	defer func() { a.trace.TelemetrySpanEnd(span, err) }()

	// 初始化
	out = make([]interfaces.ActiveUserInfo, 0)

	// 获取当月最大时间和最小时间
	// f_time为字符串，格式为"2025-09-19"
	minTime := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	maxTime := minTime.AddDate(0, 1, 0).Add(-time.Second)
	maxTimeStr := fmt.Sprintf("%d-%02d-%02d", year, month, maxTime.Day())
	minTimeStr := fmt.Sprintf("%d-%02d-%02d", year, month, minTime.Day())

	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "select f_active_count, f_activate_count, f_time from %s.t_active_user_day where f_time >= ? and f_time <= ? order by f_time asc"
	strSQL = fmt.Sprintf(strSQL, dbName)
	rows, sqlErr := a.dbTrace.QueryContext(newCtx, strSQL, minTimeStr, maxTimeStr)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				a.logger.Errorln(rowsErr)
			}
		}
	}()

	if sqlErr != nil {
		a.logger.Errorln(sqlErr, strSQL)
		return out, sqlErr
	}

	for rows.Next() {
		var activeCount int
		var activateCount int
		var strTime string
		if err := rows.Scan(&activeCount, &activateCount, &strTime); err != nil {
			a.logger.Errorln(err, strSQL)
			return out, err
		}

		// 解析时间字符串
		time, err := time.Parse("2006-01-02", strTime)
		if err != nil {
			a.logger.Errorln(err, strSQL)
			return out, err
		}

		out = append(out, interfaces.ActiveUserInfo{
			ActiveCount:   activeCount,
			ActivateCount: activateCount,
			Year:          time.Year(),
			Month:         int(time.Month()),
			Day:           time.Day(),
		})
	}

	return out, nil
}

// GetYearActiveUserInfo 获取当前年度活跃用户信息
func (a *activeUser) GetYearActiveUserInfo(ctx context.Context, year int) (out []interfaces.ActiveUserInfo, err error) {
	// trace
	a.trace.SetClientSpanName("数据库操作-获取当前年度活跃用户信息")
	newCtx, span := a.trace.AddClientTrace(ctx)
	defer func() { a.trace.TelemetrySpanEnd(span, err) }()

	// 初始化
	out = make([]interfaces.ActiveUserInfo, 0)

	// 获取当月最大时间和最小时间
	// f_time为字符串，格式为"2025-01"
	maxTimeStr := fmt.Sprintf("%d-12", year)
	minTimeStr := fmt.Sprintf("%d-01", year)

	dbName := common.GetDBName("sharemgnt_db")
	strSQL := "select f_active_count, f_activate_count, f_time from %s.t_active_user_month where f_time >= ? and f_time <= ? order by f_time asc"
	strSQL = fmt.Sprintf(strSQL, dbName)
	rows, sqlErr := a.dbTrace.QueryContext(newCtx, strSQL, minTimeStr, maxTimeStr)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				a.logger.Errorln(rowsErr)
			}
		}
	}()

	if sqlErr != nil {
		a.logger.Errorln(sqlErr, strSQL)
		return out, sqlErr
	}

	for rows.Next() {
		var activeCount int
		var activateCount int
		var strTime string
		if err := rows.Scan(&activeCount, &activateCount, &strTime); err != nil {
			a.logger.Errorln(err, strSQL)
			return out, err
		}

		// 解析时间字符串
		time, err := time.Parse("2006-01", strTime)
		if err != nil {
			a.logger.Errorln(err, strSQL)
			return out, err
		}

		out = append(out, interfaces.ActiveUserInfo{
			ActiveCount:   activeCount,
			ActivateCount: activateCount,
			Year:          time.Year(),
			Month:         int(time.Month()),
		})
	}

	return out, nil
}

// GetMonthTotalCount 获取当前月度活跃用户总数和激活数
func (a *activeUser) GetMonthTotalCount(ctx context.Context, year, month int) (count, activeteCount int, err error) {
	// trace
	a.trace.SetClientSpanName("数据库操作-获取当前月度活跃用户信息")
	newCtx, span := a.trace.AddClientTrace(ctx)
	defer func() { a.trace.TelemetrySpanEnd(span, err) }()

	// 获取当月最大时间和最小时间
	// f_time为字符串，格式为"2025-09"
	currentTime := fmt.Sprintf("%d-%02d", year, month)
	sql := "select f_total_count, f_activate_count from %s.t_active_user_month where f_time = ?"
	sql = fmt.Sprintf(sql, common.GetDBName("sharemgnt_db"))
	rows, err := a.dbTrace.QueryContext(newCtx, sql, currentTime)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				a.logger.Errorln(rowsErr)
			}
		}
	}()

	if err != nil {
		a.logger.Errorln(err, sql)
		return count, activeteCount, err
	}

	for rows.Next() {
		if err := rows.Scan(&count, &activeteCount); err != nil {
			a.logger.Errorln(err, sql)
			return count, activeteCount, err
		}
	}

	return count, activeteCount, nil
}

// GetYearTotalCount 获取当前年度活跃用户总数和激活数
func (a *activeUser) GetYearTotalCount(ctx context.Context, year int) (count, activeteCount int, err error) {
	// trace
	a.trace.SetClientSpanName("数据库操作-获取当前年度活跃用户信息")
	newCtx, span := a.trace.AddClientTrace(ctx)
	defer func() { a.trace.TelemetrySpanEnd(span, err) }()

	// 获取当年最大时间和最小时间
	// f_time为字符串，格式为"2025-01"
	currentTime := fmt.Sprintf("%d", year)
	sql := "select f_total_count, f_activate_count from %s.t_active_user_year where f_time = ?"
	sql = fmt.Sprintf(sql, common.GetDBName("sharemgnt_db"))
	rows, err := a.dbTrace.QueryContext(newCtx, sql, currentTime)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				a.logger.Errorln(rowsErr)
			}
		}
	}()

	if err != nil {
		a.logger.Errorln(err, sql)
		return count, activeteCount, err
	}

	for rows.Next() {
		if err := rows.Scan(&count, &activeteCount); err != nil {
			a.logger.Errorln(err, sql)
			return count, activeteCount, err
		}
	}

	return count, activeteCount, nil
}

// GetActivateUserCount 获取用户激活数
func (a *activeUser) GetActivateUserCount(ctx context.Context) (count int, err error) {
	// trace
	a.trace.SetClientSpanName("数据库操作-获取用户激活数")
	newCtx, span := a.trace.AddClientTrace(ctx)
	defer func() { a.trace.TelemetrySpanEnd(span, err) }()

	// 获取用户激活数
	sql := `select count(*) from %s.t_user 
			where f_activate_status = 1 
			and f_user_id <> '266c6a42-6131-4d62-8f39-853e7093701c' 
			and f_user_id <> '94752844-BDD0-4B9E-8927-1CA8D427E699' 
			and f_user_id <> '234562BE-88FF-4440-9BFF-447F139871A2' 
			and f_user_id <> '4bb41612-a040-11e6-887d-005056920bea'`
	sql = fmt.Sprintf(sql, common.GetDBName("sharemgnt_db"))
	rows, err := a.dbTrace.QueryContext(newCtx, sql)
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				a.logger.Errorln(rowsErr)
			}
		}
	}()

	if err != nil {
		a.logger.Errorln(err, sql)
		return count, err
	}

	for rows.Next() {
		if err := rows.Scan(&count); err != nil {
			a.logger.Errorln(err, sql)
			return count, err
		}
	}

	return count, nil
}
