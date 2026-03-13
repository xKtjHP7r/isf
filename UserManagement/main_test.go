package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/assert"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/mock/gomock"

	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/proton-rds-sdk-go/sqlx"

	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/common"
	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/dbaccess"
	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/drivenadapters"
	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/driveradapters"
	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/interfaces"
	imock "devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/interfaces/mock"
	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/logics"
)

var (
	mockTestOnce sync.Once

	// handler test
	ahandler   driveradapters.AppRestHandler
	dhandler   driveradapters.DepartRestHandler
	uhandler   driveradapters.UserRestHandler
	conHanlder driveradapters.ContactorRestHandler
	oaHandler  driveradapters.OrgPermAppHandler
	igHandler  driveradapters.InternalGroupRestHandler

	// dephttpsvc
	hydraHandler interfaces.DepHTTPSvc
	ossHandler   interfaces.DepHTTPSvc

	// engine
	publicEngine  *gin.Engine
	privateEngine *gin.Engine

	// response
	activeUserToken      []byte
	passiveUserToken     []byte
	adminUserToken       []byte
	businessUserToken    []byte
	anonymousUserToken   []byte
	ossGatewayGetDownURL []byte
	updateClientRes      []byte
	registClientRes      []byte

	// data
	mock         sqlmock.Sqlmock
	activeUserID = "a5d47ec5-231f-35f5-1111-9194b66134a5"
)

const (
	POST = "POST"
)

type testMsgBroker struct {
}

// Publish 消息发送
func (t *testMsgBroker) Publish(msgType interfaces.MsgType, msg interface{}) error {
	return nil
}

// AnonymityAuth 匿名认证消息发送
func (t *testMsgBroker) AnonymityAuth(msgType string, msg interface{}) error {
	return nil
}

// ContactorDeleted 联系人组被删除
func (t *testMsgBroker) ContactorDeleted(ids []string) (err error) {
	return nil
}

// InternalGroupDeleted 内部组被删除
func (t *testMsgBroker) InternalGroupDeleted(ids []string) (err error) {
	return nil
}

// DepartDeleted 部门被删除
func (t *testMsgBroker) DepartDeleted(id string) (err error) {
	return nil
}

// OrgManagerChanged 更新配额
func (t *testMsgBroker) OrgManagerChanged(ids []string) (err error) {
	return nil
}

// UserStatusChanged 用户状态变更
func (t *testMsgBroker) UserStatusChanged(id string, status bool) (err error) {
	return nil
}

type testEACPLog struct {
}

func (t *testEACPLog) OpSetCSFLevelEnumLog(visitor *interfaces.Visitor, csfLevelEnum []string) error {
	return nil
}

func (t *testEACPLog) OpSetCSFLevel2EnumLog(visitor *interfaces.Visitor, csfLevel2Enum []string) error {
	return nil
}

// 记录日志
func (t *testEACPLog) EacpLog(visitor *interfaces.Visitor, op interfaces.Operation, logInfo interface{}) error {
	return nil
}

// OpAddOrgPermAppLog 增加应用账户组织架构管理权限
func (t *testEACPLog) OpAddOrgPermAppLog(visitor *interfaces.Visitor, perm *interfaces.AppOrgPerm) error {
	return nil
}

// OpDeleteOrgPermAppLog 删除应用账户组织架构管理权限
func (t *testEACPLog) OpDeleteOrgPermAppLog(visitor *interfaces.Visitor, perm *interfaces.AppOrgPerm) error {
	return nil
}

// OpUpdateOrgPermAppLog 更新应用账户组织架构管理权限
func (t *testEACPLog) OpUpdateOrgPermAppLog(visitor *interfaces.Visitor, perm *interfaces.AppOrgPerm) error {
	return nil
}

// OpSetDefaultPWDLog 更新用户初始密码
func (t *testEACPLog) OpSetDefaultPWDLog(visitor *interfaces.Visitor) error {
	return nil
}

// OpDeleteDepart 删除部门
func (t *testEACPLog) OpDeleteDepart(visitor *interfaces.Visitor, departName string, isRoot bool) error {
	return nil
}

// OpUserExpiredDisabled 用户过期自动禁用
func (t *testEACPLog) OpUserExpiredDisabled(displayName, loginName string) error {
	return nil
}

// OpUserNotLoginDisabled 用户长时间未登录自动禁用
func (t *testEACPLog) OpUserNotLoginDisabled(displayName, loginName string) error {
	return nil
}

func newTestUserManagement(t *testing.T) {
	mockTestOnce.Do(func() {
		// 设置
		common.SvcConfig.CleanAvatarOffsetTime = 3600
		common.SvcConfig.Lang = "zh_CN"

		// 初始化http服务
		TestInitHydraService(t)
		TestInitOSSGatewayService(t)

		common.InitARTrace("test")

		// 依赖注入
		var db *sqlx.DB
		db, mock, _ = sqlx.New()

		dbaccess.SetDBPool(db)
		logics.SetDBPool(db)
		logics.SetDBUser(dbaccess.NewUser())
		logics.SetDBGroup(dbaccess.NewGroup())
		logics.SetDBGroupMembers(dbaccess.NewGroupMember())
		logics.SetDBDepartment(dbaccess.NewDepartment())
		logics.SetDBContactor(dbaccess.NewContactor())
		logics.SetDBAnonymous(dbaccess.NewAnonymous())
		logics.SetDBApp(dbaccess.NewApp())
		logics.SetDBConfig(dbaccess.NewConfig())
		logics.SetDBOrgPermApp(dbaccess.NewOrgPermApp())
		logics.SetDBOutbox(dbaccess.NewOutbox())
		logics.SetDBRole(dbaccess.NewRole())
		logics.SetDBAvatar(dbaccess.NewAvatar())
		logics.SetDBInternalGroup(dbaccess.NewInternalGroup())
		logics.SetDBInternalGroupMember(dbaccess.NewInternalGroupMember())
		tempLog := testEACPLog{}
		logics.SetDnEacpLog(&tempLog)
		logics.SetDnHydra(drivenadapters.NewHydra())
		tempBR := testMsgBroker{}
		logics.SetDnMessageBroker(&tempBR)
		logics.SetDnOSSGateWay(drivenadapters.NewOSSGateWay())

		ahandler = driveradapters.NewAppRESTHandler()
		dhandler = driveradapters.NewDepartRESTHandler()
		uhandler = driveradapters.NewUserRESTHandler()
		conHanlder = driveradapters.NewContactorRESTHandler()
		oaHandler = driveradapters.NewOrgPermAppHandler()
		igHandler = driveradapters.NewInternalGroupRESTHandler()

		// 接口注册
		gin.SetMode(gin.TestMode)
		publicEngine = gin.New()
		publicEngine.Use(gin.Recovery())
		ahandler.RegisterPublic(publicEngine)
		uhandler.RegisterPublic(publicEngine)
		conHanlder.RegisterPublic(publicEngine)
		oaHandler.RegisterPublic(publicEngine)

		gin.SetMode(gin.TestMode)
		privateEngine = gin.New()
		privateEngine.Use(gin.Recovery())
		ahandler.RegisterPrivate(privateEngine)
		dhandler.RegisterPrivate(privateEngine)
		uhandler.RegisterPrivate(privateEngine)
		igHandler.RegisterPrivate(privateEngine)

		// 初始化response
		initResponseData()
	})
}

func handle(t *testing.T, w http.ResponseWriter, r *http.Request, handler interfaces.DepHTTPSvc) {
	target := r.URL.EscapedPath()
	method := r.Method
	body, err := io.ReadAll(r.Body)
	assert.Equal(t, err, nil)
	var reqBody map[string]interface{}
	if method == POST {
		reqBody = make(map[string]interface{})
		err = jsoniter.Unmarshal(body, &reqBody)
		assert.Equal(t, err, nil)
	}
	resCode, resBody := handler.HandleRequest(method, target, reqBody)
	w.WriteHeader(resCode)
	_, err = w.Write(resBody)
	assert.Equal(t, err, nil)
}

func TestInitHydraService(t *testing.T) {
	// http mock,模拟hydra服务
	tsHydra1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		url := r.URL.EscapedPath()
		method := r.Method
		body, _ := io.ReadAll(r.Body)
		var resCode int
		var resBody []byte
		var reqBody interface{}
		if url == "/admin/oauth2/introspect" {
			reqBody = string(body)
		} else if url == "/admin/clients" && method == POST {
			reqBody = make(map[string]interface{})
			err := jsoniter.Unmarshal(body, &reqBody)
			assert.Equal(t, err, nil)
		} else if method == POST || method == "PUT" {
			reqBody = make(map[string]interface{})
			err := jsoniter.Unmarshal(body, &reqBody)
			assert.Equal(t, err, nil)
		} else if url == "/admin/clients" && method == "PATCH" {
			reqBody = make([]interface{}, 0)
			err := jsoniter.Unmarshal(body, &reqBody)
			assert.Equal(t, err, nil)
		}
		resCode, resBody = hydraHandler.HandleRequest(method, url, reqBody)
		w.WriteHeader(resCode)
		_, err := w.Write(resBody)
		assert.Equal(t, err, nil)
	}))

	// 解析服务地址
	h, err := url.Parse(tsHydra1.URL)
	if err != nil {
		t.Fatalf("%v", err)
	}
	// 配置注入
	port, _ := strconv.Atoi(h.Port())
	common.SvcConfig.OAuthAdminHost = h.Hostname()
	common.SvcConfig.OAuthAdminPort = port
}

func TestInitOSSGatewayService(t *testing.T) {
	// http mock,模拟ossgateway服务
	tsOSS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handle(t, w, r, ossHandler)
	}))

	// 解析服务地址
	h, err := url.Parse(tsOSS.URL)
	if err != nil {
		t.Fatalf("%v", err)
	}
	// 配置注入
	port, _ := strconv.Atoi(h.Port())
	common.SvcConfig.OSSGateWayPrivateHost = h.Hostname()
	common.SvcConfig.OSSGateWayPrivatePort = port
}

func initHTTPMock(ctrl *gomock.Controller) (hydraMockObj, ossMockObj *imock.MockDepHTTPSvc) {
	hydraMockObj = imock.NewMockDepHTTPSvc(ctrl)
	hydraHandler = hydraMockObj

	ossMockObj = imock.NewMockDepHTTPSvc(ctrl)
	ossHandler = ossMockObj
	return
}

func initResponseData() {
	activeUserToken, _ = jsoniter.Marshal(gin.H{
		"active":    true,
		"client_id": "xxx",
		"sub":       activeUserID,
		"scope":     "xxx",
		"ext": gin.H{
			"visitor_type": "realname",
			"login_ip":     "xx.xx.xx.xx",
			"udid":         "xxx",
			"account_type": "other",
			"client_type":  "web",
		},
	})

	passiveUserToken, _ = jsoniter.Marshal(gin.H{
		"active":    false,
		"client_id": "xxx",
		"sub":       activeUserID,
		"scope":     "xxx",
		"ext": gin.H{
			"visitor_type": "realname",
			"login_ip":     "xx.xx.xx.xx",
			"udid":         "xxx",
			"account_type": "other",
			"client_type":  "web",
		},
	})

	adminUserToken, _ = jsoniter.Marshal(gin.H{
		"active":    true,
		"client_id": "xxx",
		"sub":       interfaces.SystemSysAdmin,
		"scope":     "xxx",
		"ext": gin.H{
			"visitor_type": "realname",
			"login_ip":     "xx.xx.xx.xx",
			"udid":         "xxx",
			"account_type": "other",
			"client_type":  "web",
		},
	})

	businessUserToken, _ = jsoniter.Marshal(gin.H{
		"active":    true,
		"client_id": activeUserID,
		"sub":       activeUserID,
		"scope":     "xxx",
		"ext": gin.H{
			"visitor_type": "realname",
			"login_ip":     "xx.xx.xx.xx",
			"udid":         "xxx",
			"account_type": "other",
			"client_type":  "web",
		},
	})

	anonymousUserToken, _ = jsoniter.Marshal(gin.H{
		"active":    true,
		"client_id": "xxx",
		"sub":       activeUserID,
		"scope":     "xxx",
		"ext": gin.H{
			"visitor_type": "anonymous",
			"login_ip":     "xx.xx.xx.xx",
			"udid":         "xxx",
			"account_type": "other",
			"client_type":  "web",
		},
	})

	ossGatewayGetDownURL, _ = jsoniter.Marshal(gin.H{
		"url":     "url1",
		"method":  "GET",
		"headers": make(map[string]interface{}),
	})

	updateClientRes, _ = jsoniter.Marshal(gin.H{
		"client_id":     "xxx-xxx-xxx-xxx",
		"client_name":   "client_1",
		"client_secret": "some-secret",
	})

	registClientRes, _ = jsoniter.Marshal(gin.H{
		"id": "client_id",
	})
}
