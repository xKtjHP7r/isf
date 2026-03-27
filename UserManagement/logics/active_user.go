package logics

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/common"
	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/interfaces"
	gerrors "devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/go-lib/error"
	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/go-lib/observable"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/proton-rds-sdk-go/sqlx"
	"github.com/mitchellh/mapstructure"
)

var (
	activeUserOnce   sync.Once
	activeUserLogics *activeUser

	i18nActiveUserMap = common.NewI18n(common.I18nMap{
		i18nIDObjectsInUnDistributeUserGroup: {
			interfaces.SimplifiedChinese:  "未分配组",
			interfaces.TraditionalChinese: "未分配組",
			interfaces.AmericanEnglish:    "Unassigned Group",
		},
		i18nIDObjectsMonthlyOverallIndex: {
			interfaces.SimplifiedChinese:  "月度总体指标",
			interfaces.TraditionalChinese: "月度總體指標",
			interfaces.AmericanEnglish:    "Monthly Overall Index",
		},
		i18nIDObjectsYearlyOverallIndex: {
			interfaces.SimplifiedChinese:  "年度总体指标",
			interfaces.TraditionalChinese: "年度總體指標",
			interfaces.AmericanEnglish:    "Annual Overall Index",
		},
		i18nIDObjectsIndex: {
			interfaces.SimplifiedChinese:  "指标",
			interfaces.TraditionalChinese: "指標",
			interfaces.AmericanEnglish:    "Index",
		},
		i18nIDObjectsValue: {
			interfaces.SimplifiedChinese:  "数值",
			interfaces.TraditionalChinese: "數值",
			interfaces.AmericanEnglish:    "Value",
		},
		i18nIDObjectsTotalUserCount: {
			interfaces.SimplifiedChinese:  "总用户数",
			interfaces.TraditionalChinese: "使用者總數",
			interfaces.AmericanEnglish:    "Total Users",
		},
		i18nIDObjectsActivateCount: {
			interfaces.SimplifiedChinese:  "激活用户数",
			interfaces.TraditionalChinese: "激活用戶數",
			interfaces.AmericanEnglish:    "Activate Users",
		},
		i18nIDObjectsAverageActiveUser: {
			interfaces.SimplifiedChinese:  "平均活跃用户数",
			interfaces.TraditionalChinese: "平均活躍用戶數",
			interfaces.AmericanEnglish:    "Average active user",
		},
		i18nIDObjectsAverageActiveDegree: {
			interfaces.SimplifiedChinese:  "平均活跃用户度",
			interfaces.TraditionalChinese: "平均活躍用戶度",
			interfaces.AmericanEnglish:    "Average active degree",
		},
		i18nIDObjectsLowestActiveUser: {
			interfaces.SimplifiedChinese:  "最低活跃用户数",
			interfaces.TraditionalChinese: "最低活躍用戶數",
			interfaces.AmericanEnglish:    "Lowest active user",
		},
		i18nIDObjectsLowestActiveDegree: {
			interfaces.SimplifiedChinese:  "最低活跃用户度",
			interfaces.TraditionalChinese: "最低活躍用戶度",
			interfaces.AmericanEnglish:    "Lowest active degree",
		},
		i18nIDObjectsHighestActiveUser: {
			interfaces.SimplifiedChinese:  "最高活跃用户数",
			interfaces.TraditionalChinese: "最高活躍用戶數",
			interfaces.AmericanEnglish:    "Highest active user",
		},
		i18nIDObjectsHighestActiveDegree: {
			interfaces.SimplifiedChinese:  "最高活跃用户度",
			interfaces.TraditionalChinese: "最高活躍用戶度",
			interfaces.AmericanEnglish:    "Highest active degree",
		},
		i18nIDObjectsMonthlyDetailedIndex: {
			interfaces.SimplifiedChinese:  "月度详细指标",
			interfaces.TraditionalChinese: "月度詳細指標",
			interfaces.AmericanEnglish:    "Monthly Detailed Index",
		},
		i18nIDObjectsYearlyDetailedIndex: {
			interfaces.SimplifiedChinese:  "年度详细指标",
			interfaces.TraditionalChinese: "年度詳細指標",
			interfaces.AmericanEnglish:    "Annual Detailed Index",
		},
		i18nIDObjectsDate: {
			interfaces.SimplifiedChinese:  "日期",
			interfaces.TraditionalChinese: "日期",
			interfaces.AmericanEnglish:    "Date",
		},
		i18nIDObjectsDailyActiveUser: {
			interfaces.SimplifiedChinese:  "当日活跃用户数",
			interfaces.TraditionalChinese: "當日活躍用戶數",
			interfaces.AmericanEnglish:    "Daily active user",
		},
		i18nIDObjectsDailyActiveDegree: {
			interfaces.SimplifiedChinese:  "日活跃度",
			interfaces.TraditionalChinese: "日活躍度",
			interfaces.AmericanEnglish:    "Daily active degree",
		},
		i18nIDObjectsMonth: {
			interfaces.SimplifiedChinese:  "月份",
			interfaces.TraditionalChinese: "月份",
			interfaces.AmericanEnglish:    "Month",
		},
		i18nIDObjectsMonthlyActiveUser: {
			interfaces.SimplifiedChinese:  "当月活跃用户数",
			interfaces.TraditionalChinese: "當月活躍用戶數",
			interfaces.AmericanEnglish:    "Monthly active user",
		},
		i18nIDObjectsMonthlyActiveDegree: {
			interfaces.SimplifiedChinese:  "月活跃度",
			interfaces.TraditionalChinese: "月活躍度",
			interfaces.AmericanEnglish:    "Monthly active degree",
		},
	})
)

type activeUser struct {
	activeUserDB interfaces.DBActiveUser
	trace        observable.Tracer
	userDB       interfaces.DBUser
	i18n         *common.I18n
	role         interfaces.LogicsRole
	pool         *sqlx.DB
	logger       common.Logger
	ob           interfaces.LogicsOutbox
	eacpLog      interfaces.DrivenEacpLog
}

func NewActiveUser() *activeUser {
	activeUserOnce.Do(func() {
		activeUserLogics = &activeUser{
			activeUserDB: dbActiveUser,
			userDB:       dbUser,
			trace:        common.SvcARTrace,
			role:         NewRole(),
			i18n:         i18nActiveUserMap,
			pool:         dbTracePool,
			logger:       common.NewLogger(),
			ob:           NewOutbox(OutboxBusinessActiveUser),
			eacpLog:      dnEacpLog,
		}

		activeUserLogics.ob.RegisterHandlers(outboxActiveUserInfoExportedLog, activeUserLogics.sendActiveUserInfoExportedLog)
	})
	return activeUserLogics
}

func (a *activeUser) sendActiveUserInfoExportedLog(content interface{}) (err error) {
	info := content.(map[string]interface{})
	bYear := info["is_year"].(bool)
	visitor := interfaces.Visitor{}
	err = mapstructure.Decode(info["visitor"], &visitor)
	if err != nil {
		a.logger.Errorf("sendActiveUserInfoExportedLog mapstructure.Decode err:%v", err)
		return err
	}

	err = a.eacpLog.OpActiveUserInfoExported(&visitor, bYear)
	if err != nil {
		a.logger.Errorf("sendActiveUserInfoExportedLog err:%v", err)
	}
	return err
}

// GetActiveUserInfo 获取活跃用户信息
func (a *activeUser) GetActiveUserInfo(ctx context.Context, v *interfaces.Visitor, bYear bool, year, month int) (out []byte, err error) {
	// trace
	a.trace.SetInternalSpanName("逻辑层-获取当前月度活跃用户信息")
	newCtx, span := a.trace.AddInternalTrace(ctx)
	defer func() { a.trace.TelemetrySpanEnd(span, err) }()

	// 检测调用者是否拥有指定角色
	var mapRoles map[interfaces.Role]bool
	mapRoles, err = getRolesByUserID2(newCtx, a.role, v.ID)
	if err != nil {
		return
	}

	if !mapRoles[interfaces.SystemRoleSuperAdmin] && !mapRoles[interfaces.SystemRoleSysAdmin] {
		err := gerrors.NewError(gerrors.PublicForbidden, "this user do not has this role")
		return nil, err
	}

	// 判断是获取月度活跃用户信息还是年度活跃用户信息
	var tempActiveUserInfos []interfaces.ActiveUserInfo
	if bYear {
		tempActiveUserInfos, err = a.activeUserDB.GetYearActiveUserInfo(newCtx, year)
	} else {
		tempActiveUserInfos, err = a.activeUserDB.GetMonthActiveUserInfo(newCtx, year, month)
	}

	if err != nil {
		return nil, err
	}

	// 最大活跃数和最小活跃数都是从数据库里面拿到的数据中找到的f_active_count字段的最大值和最小值
	// 最大活跃度和最小活跃度是计算数据， 活跃数/激活数 得到的结果中找到的最大值和最小值
	// 平均活跃度和活跃数是计算数据， 每天或者每月的活跃数和活跃度求平均值

	// 最大最小活跃数
	maxActiveCount := 0
	minActivaCount := math.MaxInt32

	// 最大最小活跃度
	maxActiveDegree := 0.0
	minActiveDegree := math.MaxFloat64

	// 平均活跃度和活跃数
	avgActiveCount := 0
	avgActiveDegree := 0.0

	tempAllInfos := make([]interfaces.ActiveUserInfo, 0)
	for _, v := range tempActiveUserInfos {
		tempInfo := interfaces.ActiveUserInfo{
			Year:        v.Year,
			Month:       v.Month,
			Day:         v.Day,
			ActiveCount: v.ActiveCount,
		}

		// 检查最大最小活跃数
		if tempInfo.ActiveCount > maxActiveCount {
			maxActiveCount = tempInfo.ActiveCount
		}
		if tempInfo.ActiveCount < minActivaCount {
			minActivaCount = tempInfo.ActiveCount
		}

		// 处理活跃度,保留4位小数
		if v.ActivateCount > 0 {
			tempInfo.ActiveDegree = float64(tempInfo.ActiveCount) / float64(v.ActivateCount)

			if tempInfo.ActiveDegree > 1.0 {
				tempInfo.ActiveDegree = 1.0
			}

			if tempInfo.ActiveDegree > maxActiveDegree {
				maxActiveDegree = tempInfo.ActiveDegree
			}
			if tempInfo.ActiveDegree < minActiveDegree {
				minActiveDegree = tempInfo.ActiveDegree
			}
		}

		avgActiveCount += tempInfo.ActiveCount
		avgActiveDegree += tempInfo.ActiveDegree
		tempAllInfos = append(tempAllInfos, tempInfo)
	}

	// 获取总数和总激活数
	totalCount, totalActiveCount := 0, 0
	if len(tempAllInfos) > 0 {
		totalCount, totalActiveCount, err = a.getTotalCount(newCtx, bYear, year, month)
		if err != nil {
			return nil, err
		}

		// 计算平均活跃数和平均活跃度 平均活跃数向上取整
		avgActiveCount = int(math.Ceil(float64(avgActiveCount) / float64(len(tempAllInfos))))
		avgActiveDegree /= float64(len(tempAllInfos))
	}

	minActivaCount, minActiveDegree = a.MaxMinCheck(minActivaCount, minActiveDegree)

	// 生成返回数据流
	out, err = a.generateFile(v, bYear, totalCount, totalActiveCount, avgActiveCount, avgActiveDegree,
		minActivaCount, minActiveDegree, maxActiveCount, maxActiveDegree, tempAllInfos)
	if err != nil {
		return nil, err
	}

	// 记录审计日志
	tx, err := a.pool.Begin()
	if err != nil {
		a.logger.Errorf("AddGroupMembers send audit log  pool begin error:%v", err)
		return
	}

	// 异常时Rollback
	defer func() {
		switch err {
		case nil:
			// 提交事务
			if err = tx.Commit(); err != nil {
				a.logger.Errorf("GetActiveUserInfo Transaction Commit Error:%v", err)
				return
			}

			a.ob.NotifyPushOutboxThread()

		default:
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				a.logger.Errorf("AddGroupMembers Rollback err:%v", rollbackErr)
			}
		}
	}()

	// 记录审计日志
	content := make(map[string]interface{})
	content["visitor"] = *v
	content["is_year"] = bYear
	err = a.ob.AddOutboxInfo(outboxActiveUserInfoExportedLog, content, tx)
	if err != nil {
		a.logger.Errorf("GetActiveUserInfo AddOutboxInfo err:%v", err)
		return nil, err
	}

	return out, nil
}

func (a *activeUser) MaxMinCheck(minActivaCount int, minActiveDegree float64) (outCount int, outDegree float64) {
	// 数据校验
	if minActivaCount == math.MaxInt32 {
		minActivaCount = 0
	}

	if minActiveDegree == math.MaxFloat64 {
		minActiveDegree = 0.0
	}
	return minActivaCount, minActiveDegree
}

// 获取总数和总激活数
func (a *activeUser) getTotalCount(ctx context.Context, bYear bool, year, month int) (totalCount, totalActiveCount int, err error) {
	// trace
	a.trace.SetInternalSpanName("逻辑层-获取当前月度活跃用户信息")
	newCtx, span := a.trace.AddInternalTrace(ctx)
	defer func() { a.trace.TelemetrySpanEnd(span, err) }()

	// 判断是获取月度活跃用户信息还是年度活跃用户信息
	nowYear := time.Now().Year()
	nowMonth := int(time.Now().Month())

	// 先获取数据库的数据
	if bYear {
		totalCount, totalActiveCount, err = a.activeUserDB.GetYearTotalCount(newCtx, year)
	} else {
		totalCount, totalActiveCount, err = a.activeUserDB.GetMonthTotalCount(newCtx, year, month)
	}

	if err != nil {
		return totalCount, totalActiveCount, err
	}

	// 如果当月或者当年的数据，需要加上所有数据
	// 当月和当年的数据 应该是激活后被删除的用户数
	if (bYear && nowYear == year) || (!bYear && nowYear == year && nowMonth == month) {
		temp1, err := a.userDB.GetAllUserCount(newCtx)
		if err != nil {
			return totalCount, totalActiveCount, err
		}
		temp2, err := a.activeUserDB.GetActivateUserCount(newCtx)
		if err != nil {
			return totalCount, totalActiveCount, err
		}

		totalCount += temp1
		totalActiveCount += temp2
	}

	return totalCount, totalActiveCount, nil
}

// generateFile 生成文件
func (a *activeUser) generateFile(visitor *interfaces.Visitor, bYear bool, totalCount,
	totalActiveCount, avgActiveCount int, avgActiveDegree float64, minActivaCount int, minActiveDegree float64, maxActiveCount int,
	maxActiveDegree float64, tempAllInfos []interfaces.ActiveUserInfo) (out []byte, err error) {
	// 生成一个csv文件
	buf := new(bytes.Buffer)

	// 文件写入
	csvWriter := csv.NewWriter(buf)
	buf.Write([]byte{0xEF, 0xBB, 0xBF})

	tempInsertStrings := make([][]string, 0)
	if bYear {
		tempInsertStrings = append(tempInsertStrings, []string{a.i18n.Load(i18nIDObjectsYearlyOverallIndex, visitor.Language)})
	} else {
		tempInsertStrings = append(tempInsertStrings, []string{a.i18n.Load(i18nIDObjectsMonthlyOverallIndex, visitor.Language)})
	}

	tempInsertStrings = append(tempInsertStrings,
		[]string{a.i18n.Load(i18nIDObjectsIndex, visitor.Language), a.i18n.Load(i18nIDObjectsValue, visitor.Language)},
		[]string{a.i18n.Load(i18nIDObjectsTotalUserCount, visitor.Language), strconv.Itoa(totalCount)},
		[]string{a.i18n.Load(i18nIDObjectsActivateCount, visitor.Language), strconv.Itoa(totalActiveCount)},
		[]string{a.i18n.Load(i18nIDObjectsAverageActiveUser, visitor.Language), strconv.Itoa(avgActiveCount)},
		[]string{a.i18n.Load(i18nIDObjectsAverageActiveDegree, visitor.Language), strconv.FormatFloat(avgActiveDegree, 'f', 4, 64)},
		[]string{a.i18n.Load(i18nIDObjectsLowestActiveUser, visitor.Language), strconv.Itoa(minActivaCount)},
		[]string{a.i18n.Load(i18nIDObjectsLowestActiveDegree, visitor.Language), strconv.FormatFloat(minActiveDegree, 'f', 4, 64)},
		[]string{a.i18n.Load(i18nIDObjectsHighestActiveUser, visitor.Language), strconv.Itoa(maxActiveCount)},
		[]string{a.i18n.Load(i18nIDObjectsHighestActiveDegree, visitor.Language), strconv.FormatFloat(maxActiveDegree, 'f', 4, 64)},
		[]string{""},
	)

	if bYear {
		tempInsertStrings = append(tempInsertStrings,
			[]string{a.i18n.Load(i18nIDObjectsYearlyDetailedIndex, visitor.Language)},
			[]string{a.i18n.Load(i18nIDObjectsMonth, visitor.Language),
				a.i18n.Load(i18nIDObjectsMonthlyActiveUser, visitor.Language), a.i18n.Load(i18nIDObjectsMonthlyActiveDegree, visitor.Language)})
	} else {
		tempInsertStrings = append(tempInsertStrings,
			[]string{a.i18n.Load(i18nIDObjectsMonthlyDetailedIndex, visitor.Language)},
			[]string{a.i18n.Load(i18nIDObjectsDate, visitor.Language),
				a.i18n.Load(i18nIDObjectsDailyActiveUser, visitor.Language), a.i18n.Load(i18nIDObjectsDailyActiveDegree, visitor.Language)})
	}

	for _, v := range tempAllInfos {
		strDate := ""
		if bYear {
			// "2026/03"
			strDate = fmt.Sprintf("%d/%02d", v.Year, v.Month)
		} else {
			// "2026/03/01"
			strDate = fmt.Sprintf("%d/%02d/%02d", v.Year, v.Month, v.Day)
		}
		tempInsertStrings = append(tempInsertStrings, []string{strDate, strconv.Itoa(v.ActiveCount), strconv.FormatFloat(v.ActiveDegree, 'f', 4, 64)})
	}

	for _, v := range tempInsertStrings {
		if err := csvWriter.Write(v); err != nil {
			return nil, err
		}
	}

	// 刷新缓冲区
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return nil, err
	}

	// 生成返回数据
	return buf.Bytes(), nil
}
