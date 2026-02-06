// Package main 主程序
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kweaver-ai/TelemetrySDK-Go/exporter/v2/ar_log"
	"github.com/kweaver-ai/go-lib/rest"
	"golang.org/x/sync/errgroup"

	"Authorization/common"
	"Authorization/dbaccess"
	"Authorization/drivenadapters"
	"Authorization/driveradapters"
	"Authorization/logics"
)

// Authorization 授权管理对象
type Authorization struct {
	timer                        driveradapters.Timer
	mqHandler                    driveradapters.MQHandler
	healthHandler                driveradapters.RestHandler
	systemConfigHandler          driveradapters.RestHandler
	initData                     driveradapters.InitData
	roleHandler                  driveradapters.RestHandler
	resourceTypeHandler          driveradapters.RestHandler
	policyHandler                driveradapters.RestHandler
	policyCalcHandler            driveradapters.RestHandler
	obligationTypeHandler        driveradapters.RestHandler
	obligationHandler            driveradapters.RestHandler
	resourceTypeHierarchyHandler driveradapters.RestHandler
}

// Start 开启服务
func (t *Authorization) Start() {
	svcLog := common.NewLogger()
	svcLog.Infoln("start authorization server")

	// 初始化数据
	t.initData.InitResourceType()
	t.initData.InitRole()
	t.initData.InitRoleMembers()
	t.initData.InitPolicy()
	t.initData.InitObligationType()

	t.timer.StartCleanThread()

	gin.SetMode(common.SvcConfig.GinMode)

	startLogger := common.NewLogger()

	g, ctx := errgroup.WithContext(context.Background())

	t.mqHandler.Subscribe(g, ctx)

	// authorization public server
	g.Go(func() error {
		engine := gin.New()
		engine.Use(gin.Recovery())
		engine.UseRawPath = true
		_ = engine.SetTrustedProxies([]string{"0.0.0.0/0", "::/0"})
		t.healthHandler.RegisterPublic(engine)
		t.resourceTypeHandler.RegisterPublic(engine)
		t.policyHandler.RegisterPublic(engine)
		t.policyCalcHandler.RegisterPublic(engine)
		t.roleHandler.RegisterPublic(engine)
		t.obligationTypeHandler.RegisterPublic(engine)
		t.obligationHandler.RegisterPublic(engine)
		s := &http.Server{
			Addr:    fmt.Sprintf("%s:%d", common.SvcConfig.SvcHost, common.SvcConfig.SvcPublicPort),
			Handler: engine.Handler(),
		}
		errChan := make(chan error)
		// defer close(errChan) // TODO
		go func() {
			errChan <- s.ListenAndServe()
		}()
		select {
		case err := <-errChan:
			startLogger.Error("authorization public server return an error and exit")
			return err
		case <-ctx.Done():
			// 上下文用于通知服务器它有 5 秒的时间来完成它当前正在处理的请求
			timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second) //nolint:mnd
			defer cancel()
			if err := s.Shutdown(timeoutCtx); err != nil {
				startLogger.Error("authorization public server forced to shutdown: ", err)
			}
			startLogger.Info("authorization public server exits gracefully")
			return ctx.Err()
		}
	})

	// authorization private server
	g.Go(func() error {
		engine := gin.New()
		engine.Use(gin.Recovery())
		engine.UseRawPath = true
		_ = engine.SetTrustedProxies([]string{"0.0.0.0/0", "::/0"})
		t.systemConfigHandler.RegisterPrivate(engine)
		t.resourceTypeHandler.RegisterPrivate(engine)
		t.healthHandler.RegisterPrivate(engine)
		t.policyHandler.RegisterPrivate(engine)
		t.policyCalcHandler.RegisterPrivate(engine)
		t.roleHandler.RegisterPrivate(engine)
		t.resourceTypeHierarchyHandler.RegisterPrivate(engine)
		t.obligationTypeHandler.RegisterPrivate(engine)
		docSharePrivateServer := &http.Server{
			Addr:    fmt.Sprintf("%s:%d", common.SvcConfig.SvcHost, common.SvcConfig.SvcPrivatePort),
			Handler: engine.Handler(),
		}
		errChan := make(chan error)
		// defer close(errChan) // TODO
		go func() {
			errChan <- docSharePrivateServer.ListenAndServe()
		}()
		select {
		case err := <-errChan:
			startLogger.Error("authorization private server return an error and exit")
			return err
		case <-ctx.Done():
			// 上下文用于通知服务器它有 5 秒的时间来完成它当前正在处理的请求
			timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second) //nolint:mnd
			defer cancel()
			if err := docSharePrivateServer.Shutdown(timeoutCtx); err != nil {
				startLogger.Error("authorization private server forced to shutdown: ", err)
			}
			startLogger.Info("authorization private server exits gracefully")
			return ctx.Err()
		}
	})

	// 当监听到优雅的关闭服务提供私有接口的服务和提供公开接口的服务
	g.Go(func() error {
		// 如果您使用了https://devops.aishu.cn/AISHUDevOps/ICT/_git/go-msq
		// 建议在您的程序中增加如下实现，因为此库已将syscall.SIGINT, syscall.SIGTERM两种信号的默认行为覆盖
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		select {
		case sig := <-quit:
			return fmt.Errorf("get os signal: %v", sig)
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	if err := g.Wait(); err != nil {
		startLogger.Fatal(err)
	}
}

func main() {
	// 配置注入
	common.InitConfig()

	// 初始化ARTrace实例
	common.InitARTrace("authorization")

	// 初始化AR logger
	ar_log.InitLogger("cm", "anyshare-telemetry-sdk", "authorization")
	defer ar_log.Logger.Close()

	// 获取logger实例用于下面记录日志
	svcLog := common.NewLogger()

	// 配置log等级
	if err := common.SetLogLevel(common.SvcConfig.LogLevel); err != nil {
		svcLog.Fatalln(err)
	}

	// 检查Lang值
	if !map[string]bool{"zh_CN": true, "zh_TW": true, "en_US": true}[common.SvcConfig.Lang] {
		svcLog.Fatalln("service language not set")
	}

	// 设置错误码语言
	rest.SetLang(common.SvcConfig.Lang)

	// dbPool依赖注入
	dbPool := common.NewDBPool()
	logics.SetDBPool(dbPool)
	dbaccess.SetDBPool(dbPool)

	// dbTracePool依赖注入
	dbTracePool := common.NewDBTracePool()
	logics.SetDBTracePool(dbTracePool)
	dbaccess.SetDBTracePool(dbTracePool)

	// drivenadapters 依赖注入
	mqClient, err := common.NewMQClient()
	if err != nil {
		svcLog.Panicln(err)
	}
	driveradapters.SetMQClient(mqClient)

	// 注入db实例到出栈

	// logics的dbaccess依赖注入
	logics.SetDBResourceType(dbaccess.NewResource())
	logics.SetDBPolicy(dbaccess.NewPolicy())
	logics.SetDBPolicyCalc(dbaccess.NewPolicyCalc())
	logics.SetDBRole(dbaccess.NewRole())
	logics.SetDBRoleMember(dbaccess.NewRoleMember())
	logics.SetDBObligationType(dbaccess.NewObligationType())
	logics.SetDBObligation(dbaccess.NewObligation())
	logics.SetDBResourceTypeHierarchy(dbaccess.NewResourceTypeHierarchy())
	// logics的drivenadapters依赖注入

	logics.SetDnUserMgnt(drivenadapters.NewUserMgnt())

	server := &Authorization{
		healthHandler:                driveradapters.NewHealthHandler(),
		systemConfigHandler:          driveradapters.NewServiceConfigDriver(),
		resourceTypeHandler:          driveradapters.NewResourceTypeRestHandler(),
		policyHandler:                driveradapters.NewPolicyRestHandler(),
		policyCalcHandler:            driveradapters.NewPolicyCalcRestHandler(),
		roleHandler:                  driveradapters.NewRoleRestHandler(),
		resourceTypeHierarchyHandler: driveradapters.NewResourceTypeHierarchyRestHandler(),
		initData:                     driveradapters.NewInitData(),
		mqHandler:                    driveradapters.NewMQHandler(),
		timer:                        driveradapters.NewTimer(),
		obligationTypeHandler:        driveradapters.NewObligationTypeRestHandler(),
		obligationHandler:            driveradapters.NewObligationRestHandler(),
	}

	server.Start()
}
