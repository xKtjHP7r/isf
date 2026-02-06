package driveradapters

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"

	"github.com/kweaver-ai/go-lib/httpclient"

	gerrors "github.com/kweaver-ai/go-lib/error"

	"Authorization/common"
	"Authorization/interfaces"
)

type hydra struct {
	adminAddress   string
	log            common.Logger
	client         *http.Client
	visitorTypeMap map[string]interfaces.VisitorType
	accountTypeMap map[string]interfaces.AccountType
}

var (
	hOnce sync.Once
	h     *hydra
)

var clientTypeMap = map[string]interfaces.ClientType{
	"unknown":       interfaces.Unknown,
	"ios":           interfaces.IOS,
	"android":       interfaces.Android,
	"windows_phone": interfaces.WindowsPhone,
	"windows":       interfaces.Windows,
	"mac_os":        interfaces.MacOS,
	"web":           interfaces.Web,
	"mobile_web":    interfaces.MobileWeb,
	"nas":           interfaces.Nas,
	"console_web":   interfaces.ConsoleWeb,
	"deploy_web":    interfaces.DeployWeb,
	"linux":         interfaces.Linux,
	"app":           interfaces.APP,
}

// newHydra 创建授权服务
func newHydra() *hydra {
	hOnce.Do(func() {
		config := common.SvcConfig
		visitorTypeMap := map[string]interfaces.VisitorType{
			"realname":  interfaces.RealName,
			"anonymous": interfaces.Anonymous,
			"business":  interfaces.App,
		}
		accountTypeMap := map[string]interfaces.AccountType{
			"other":   interfaces.Other,
			"id_card": interfaces.IDCard,
		}
		h = &hydra{
			adminAddress:   fmt.Sprintf("http://%s:%d", config.OAuthAdminHost, config.OAuthAdminPort),
			log:            common.NewLogger(),
			client:         httpclient.NewRawHTTPClient(),
			visitorTypeMap: visitorTypeMap,
			accountTypeMap: accountTypeMap,
		}
	})

	return h
}

// Introspect token内省
func (h *hydra) Introspect(token string) (info interfaces.TokenIntrospectInfo, err error) {
	target := fmt.Sprintf("%v/admin/oauth2/introspect", h.adminAddress)
	resp, err := h.client.Post(target, "application/x-www-form-urlencoded",
		bytes.NewReader([]byte(fmt.Sprintf("token=%v", token))))
	if err != nil {
		h.log.Errorln(err)
		return
	}

	defer func() {
		closeErr := resp.Body.Close()
		if closeErr != nil {
			common.NewLogger().Errorln(closeErr)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if (resp.StatusCode < http.StatusOK) || (resp.StatusCode >= http.StatusMultipleChoices) {
		err = errors.New(string(body))
		return
	}

	respParam := make(map[string]any)
	err = jsoniter.Unmarshal(body, &respParam)
	if err != nil {
		return
	}

	// 令牌状态
	info.Active = respParam["active"].(bool)
	if !info.Active {
		return
	}

	// 访问者ID
	info.VisitorID = respParam["sub"].(string)
	// Scope 权限范围
	info.Scope = respParam["scope"].(string)
	// 客户端ID
	info.ClientID = respParam["client_id"].(string)
	// 客户端凭据模式
	if info.VisitorID == info.ClientID {
		info.VisitorTyp = interfaces.App
		return
	}
	// 以下字段 只在非客户端凭据模式时才存在
	// 访问者类型
	info.VisitorTyp = h.visitorTypeMap[respParam["ext"].(map[string]any)["visitor_type"].(string)]

	// 匿名用户
	if info.VisitorTyp == interfaces.Anonymous {
		// 文档库访问规则接口考虑后续扩展性，clientType为必传。本身规则计算未使用clientType
		// 设备类型本身未解析,匿名时默认为web
		info.ClientTyp = interfaces.Web
		return
	}

	// 实名用户
	if info.VisitorTyp == interfaces.RealName {
		// 登陆IP
		info.LoginIP = respParam["ext"].(map[string]any)["login_ip"].(string)
		// 设备ID
		info.Udid = respParam["ext"].(map[string]any)["udid"].(string)
		// 登录账号类型
		info.AccountTyp = h.accountTypeMap[respParam["ext"].(map[string]any)["account_type"].(string)]
		// 设备类型
		info.ClientTyp = clientTypeMap[respParam["ext"].(map[string]any)["client_type"].(string)]
		return
	}

	return
}

func verify(c *gin.Context, hydra interfaces.Hydra) (visitor interfaces.Visitor, err error) {
	tokenID := c.GetHeader("Authorization")
	token := strings.TrimPrefix(tokenID, "Bearer ")
	info, err := hydra.Introspect(token)
	if err != nil {
		return
	}

	if !info.Active {
		err = gerrors.NewError(gerrors.PublicUnauthorized, "token expired")
		return
	}

	common.NewLogger().Debugf("verify info: %v", info.VisitorID)
	visitor = interfaces.Visitor{
		ID:         info.VisitorID,
		TokenID:    tokenID,
		IP:         c.ClientIP(),
		Mac:        c.GetHeader("X-Request-MAC"),
		UserAgent:  c.GetHeader("User-Agent"),
		Type:       info.VisitorTyp,
		ClientType: info.ClientTyp,
		ClientID:   info.ClientID,
		Language:   getXLang(c),
	}

	return
}
