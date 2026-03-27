// Package main 主程序
package main

import (
	"fmt"

	"github.com/kweaver-ai/go-lib/rest"

	"UserManagement/common"
	"UserManagement/dbaccess"
	"UserManagement/drivenadapters"
	"UserManagement/driveradapters"
	"UserManagement/logics"

	"github.com/gin-gonic/gin"
)

// userManagement 用户管理对象
type userManagement struct {
	uRESTHandler        driveradapters.UserRestHandler
	cRESTHandler        driveradapters.RestHandler
	gRestHandler        driveradapters.GroupRestHandler
	dRestHandler        driveradapters.DepartRestHandler
	aRestHandler        driveradapters.AnonymousRestHandler
	pRestHandler        driveradapters.AppRestHandler
	conRestHandler      driveradapters.ContactorRestHandler
	oaRestHandler       driveradapters.OrgPermAppHandler
	igRestHandler       driveradapters.InternalGroupRestHandler
	confRestHandler     driveradapters.ConfigRestHandler
	mqHandler           driveradapters.MQHandler
	timer               driveradapters.Timer
	opRestHandler       driveradapters.OrgPermHandler
	roleHandler         driveradapters.RoleRestHandler
	reservedNameHandler driveradapters.ReservedNameHandler
	activeUserHandler   driveradapters.ActiveUserRestHandler
}

// Start 开启服务
func (t *userManagement) Start() {
	svcLog := common.NewLogger()
	svcLog.Infoln("start user-management server")

	gin.SetMode(gin.ReleaseMode)

	// 启动清理线程
	t.timer.StartCleanThread()

	// mq订阅
	t.mqHandler.Subscribe()

	go func() {
		engine := gin.New()
		engine.Use(gin.Recovery())
		engine.UseRawPath = true
		_ = engine.SetTrustedProxies([]string{"0.0.0.0/0", "::/0"})

		// 注册公共外部API
		t.cRESTHandler.RegisterPublic(engine)

		// 注册用户组API
		t.gRestHandler.RegisterPublic(engine)

		// 注册userAPI
		t.uRESTHandler.RegisterPublic(engine)

		// 注册departmentAPI
		t.dRestHandler.RegisterPublic(engine)

		// 注册应用账户管理API
		t.pRestHandler.RegisterPublic(engine)

		// 注册联系人组API
		t.conRestHandler.RegisterPublic(engine)

		// 注册应用账户组织架构管理权限API
		t.oaRestHandler.RegisterPublic(engine)

		// 注册配置管理API
		t.confRestHandler.RegisterPublic(engine)

		// 注册活跃用户API
		t.activeUserHandler.RegisterPublic(engine)

		if err := engine.Run(fmt.Sprintf("%s:%d", common.SvcConfig.SvcHost, common.SvcConfig.SvcPublicPort)); err != nil {
			svcLog.Errorln(err)
		}
	}()

	go func() {
		engine := gin.New()
		engine.Use(gin.Recovery())
		engine.UseRawPath = true
		_ = engine.SetTrustedProxies([]string{"0.0.0.0/0", "::/0"})

		// 注册公共内部API
		t.cRESTHandler.RegisterPrivate(engine)

		// 注册user内部API
		t.uRESTHandler.RegisterPrivate(engine)

		// 注册group内部API
		t.gRestHandler.RegisterPrivate(engine)

		// 注册department内部API
		t.dRestHandler.RegisterPrivate(engine)

		// 注册anonymous内部API
		t.aRestHandler.RegisterPrivate(engine)

		// 注册应用账户管理API
		t.pRestHandler.RegisterPrivate(engine)

		// 注册内部组管理API
		t.igRestHandler.RegisterPrivate(engine)

		// 注册配置管理API
		t.confRestHandler.RegisterPrivate(engine)

		// 注册配置权限API
		t.opRestHandler.RegisterPrivate(engine)

		// 注册联系人组API
		t.conRestHandler.RegisterPrivate(engine)

		// 注册角色API
		t.roleHandler.RegisterPrivate(engine)

		// 注册保留名称API
		t.reservedNameHandler.RegisterPrivate(engine)

		if err := engine.Run(fmt.Sprintf("%s:%d", common.SvcConfig.SvcHost, common.SvcConfig.SvcPrivatePort)); err != nil {
			svcLog.Errorln(err)
		}
	}()
}

func main() {
	// 配置注入
	common.InitConfig()

	// 初始化ARTrace实例
	common.InitARTrace("user-management")

	// 配置log等级
	svcLog := common.NewLogger()
	svcLog.SetLevel(common.SvcConfig.LogLevel)

	// 设置错误码语言
	rest.SetLang(common.SvcConfig.Lang)

	// dbPool注入
	dbPool := common.NewDBPool()
	logics.SetDBPool(dbPool)
	dbaccess.SetDBPool(dbPool)

	// dbTracePool依赖注入
	dbTracePool := common.NewDBTracePool()
	dbaccess.SetDBTracePool(dbTracePool)
	logics.SetDBTracePool(dbTracePool)

	// dbaccess 依赖注入
	logics.SetDBUser(dbaccess.NewUser())
	logics.SetDBGroup(dbaccess.NewGroup())
	logics.SetDBGroupMembers(dbaccess.NewGroupMember())
	logics.SetDBDepartment(dbaccess.NewDepartment())
	logics.SetDBContactor(dbaccess.NewContactor())
	logics.SetDBAnonymous(dbaccess.NewAnonymous())
	logics.SetDBApp(dbaccess.NewApp())
	logics.SetDBOutbox(dbaccess.NewOutbox())
	logics.SetDBConfig(dbaccess.NewConfig())
	logics.SetDBOrgPermApp(dbaccess.NewOrgPermApp())
	logics.SetDBRole(dbaccess.NewRole())
	logics.SetDBAvatar(dbaccess.NewAvatar())
	logics.SetDBInternalGroup(dbaccess.NewInternalGroup())
	logics.SetDBInternalGroupMember(dbaccess.NewInternalGroupMember())
	logics.SetDBOrgPerm(dbaccess.NewOrgPerm())
	logics.SetDBReservedName(dbaccess.NewReservedName())
	logics.SetDBActiveUser(dbaccess.NewActiveUser())

	// drivenadapters 依赖注入
	logics.SetDnEacpLog(drivenadapters.NewEacpLog())
	logics.SetDnHydra(drivenadapters.NewHydra())
	logics.SetDnMessageBroker(drivenadapters.NewMessageBroker())
	logics.SetDnOSSGateWay(drivenadapters.NewOSSGateWay())

	server := &userManagement{
		uRESTHandler:        driveradapters.NewUserRESTHandler(),
		cRESTHandler:        driveradapters.NewRESTHandler(),
		gRestHandler:        driveradapters.NewGroupRESTHandler(),
		dRestHandler:        driveradapters.NewDepartRESTHandler(),
		aRestHandler:        driveradapters.NewAnonymousRestHandler(),
		pRestHandler:        driveradapters.NewAppRESTHandler(),
		conRestHandler:      driveradapters.NewContactorRESTHandler(),
		oaRestHandler:       driveradapters.NewOrgPermAppHandler(),
		igRestHandler:       driveradapters.NewInternalGroupRESTHandler(),
		confRestHandler:     driveradapters.NewConfigRESTHandler(),
		mqHandler:           driveradapters.NewMQHandler(),
		timer:               driveradapters.NewTimer(),
		opRestHandler:       driveradapters.NewOrgPermApHandler(),
		roleHandler:         driveradapters.NewRoleRestHandler(),
		reservedNameHandler: driveradapters.NewReservedNameHander(),
		activeUserHandler:   driveradapters.NewActiveUserRestHandler(),
	}
	server.Start()

	select {}
}
