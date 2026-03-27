package driveradapters

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"

	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/common"
	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/interfaces"
	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/logics"
	gerrors "devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/go-lib/error"
	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/go-lib/rest"
)

// ActiveUserRestHandler 活跃用户RESTfual API Handler 接口
type ActiveUserRestHandler interface {
	// RegisterPublic 注册外部API
	RegisterPublic(engine *gin.Engine)
}

type activeUserRestHandler struct {
	activeUser interfaces.LogicsActiveUser
	hydra      interfaces.Hydra
	i18n       *common.I18n
}

var (
	activeUserOnce    sync.Once
	activeUserHandler ActiveUserRestHandler
)

// NewActiveUserRestHandler handler 对象
func NewActiveUserRestHandler() ActiveUserRestHandler {
	activeUserOnce.Do(func() {
		activeUserHandler = &activeUserRestHandler{
			activeUser: logics.NewActiveUser(),
			hydra:      newHydra(),
			i18n: common.NewI18n(common.I18nMap{
				i18nIDObjectsMonthlyActiveUserFileName: {
					interfaces.SimplifiedChinese:  "%d-%02d月度活跃报表.csv",
					interfaces.TraditionalChinese: "%d-%02d月度活躍報表.csv",
					interfaces.AmericanEnglish:    "%d-%02d Monthly Active Report.csv",
				},
				i18nIDObjectsYearlyActiveUserFileName: {
					interfaces.SimplifiedChinese:  "%d年度活跃报表.csv",
					interfaces.TraditionalChinese: "%d年度活躍報表.csv",
					interfaces.AmericanEnglish:    "%d Annual Active Report.csv",
				},
			}),
		}
	})

	return activeUserHandler
}

// RegisterPublic 注册外部API
func (h *activeUserRestHandler) RegisterPublic(engine *gin.Engine) {
	engine.GET("/api/user-management/v1/active-user-report", h.getActiveUserInfo)
}

// getActiveUserInfo 获取活跃用户信息
func (h *activeUserRestHandler) getActiveUserInfo(c *gin.Context) {
	// token验证
	visitor, vErr := verifyNewError(c, h.hydra)
	if vErr != nil {
		rest.ReplyErrorV2(c, vErr)
		return
	}

	// 获取参数
	// 年份不限制
	strYear, ok := c.GetQuery("year")
	if !ok {
		err := gerrors.NewError(gerrors.PublicBadRequest, "year is required")
		rest.ReplyErrorV2(c, err)
		return
	}
	year, err := strconv.Atoi(strYear)
	if err != nil {
		err := gerrors.NewError(gerrors.PublicBadRequest, fmt.Sprintf("year is invalid: %s", err.Error()))
		rest.ReplyErrorV2(c, err)
		return
	}

	// 月份限制在1-12之间
	var month int
	strMonth, ok := c.GetQuery("month")
	if ok {
		month, err = strconv.Atoi(strMonth)
		if err != nil {
			err := gerrors.NewError(gerrors.PublicBadRequest, fmt.Sprintf("month is invalid: %s", err.Error()))
			rest.ReplyErrorV2(c, err)
			return
		}

		if month < 1 || month > 12 {
			err := gerrors.NewError(gerrors.PublicBadRequest, "month is invalid")
			rest.ReplyErrorV2(c, err)
			return
		}
	}

	// 获取活跃用户信息
	out, err := h.activeUser.GetActiveUserInfo(c.Request.Context(), &visitor, !ok, year, month)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	// 生成文件名
	fileName := h.i18n.Load(i18nIDObjectsYearlyActiveUserFileName, visitor.Language, year)
	if ok {
		fileName = h.i18n.Load(i18nIDObjectsMonthlyActiveUserFileName, visitor.Language, year, month)
	}

	// 对文件名进行编码
	encodedFileName := url.PathEscape(fileName)

	// 使用 RFC 6266 规范：
	// 1. filename= 用于兼容旧版浏览器（通常会显示为编码后的字符串或下划线）
	// 2. filename*=utf-8'' 用于现代浏览器，能完美显示中文
	headerValue := fmt.Sprintf("attachment; filename=%q; filename*=utf-8''%s", fileName, encodedFileName)

	// 响应200
	// out 直接以文件stream的形势返回
	c.Header("Content-Disposition", headerValue)
	c.Header("Access-Control-Expose-Headers", "Content-Disposition")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", out)
}
