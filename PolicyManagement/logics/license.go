package logics

import (
	"context"
	"encoding/json"
	"fmt"
	"policy_mgnt/common"
	"policy_mgnt/drivenadapters"
	"policy_mgnt/interfaces"
	"strings"
	"sync"
	"time"

	gerrors "github.com/kweaver-ai/go-lib/error"
	"github.com/kweaver-ai/go-lib/observable"
	"github.com/kweaver-ai/proton-rds-sdk-go/sqlx"
	"github.com/mitchellh/mapstructure"
)

var (
	licenseOnce sync.Once
	lic         *license

	// 许可证缓存时间,初始值为1970-01-01 00:00:00
	licenseCacheGetTime = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	licenseCache        map[string]interfaces.License
)

var (
	strPrefix                    = "PolicyMgnt."
	StrProductAuthorizedNotValid = strPrefix + "BadRequest" + ".ProductAuthorizedNotValid"
)

type license struct {
	dnLicense        interfaces.DrivenLicense
	dnUserManagement interfaces.DrivenUserManagement
	trace            observable.Tracer
	log              common.Logger
	db               interfaces.DBLicense
	tracePool        *sqlx.DB
	event            interfaces.LogicsEvent
	dbConfig         interfaces.DBConfig
	outbox           interfaces.LogicsOutbox
	eacpLog          interfaces.DrivenEacpLog
	i18n             *common.I18n
}

func NewLicense() *license {
	licenseOnce.Do(func() {
		lic = &license{
			dnLicense:        drivenadapters.NewLicense(),
			dnUserManagement: drivenadapters.NewUserManagement(),
			trace:            common.SvcARTrace,
			log:              common.NewLogger(),
			db:               dbLicense,
			tracePool:        dbTracePool,
			event:            NewEvent(),
			dbConfig:         dbConfig,
			outbox:           NewOutbox(interfaces.OutboxProductAuthorizedUpdated),
			eacpLog:          drivenadapters.NewEacpLog(),
			i18n: common.NewI18n(common.I18nMap{
				i18nIDUserHasNoAuthUserProduct: {
					interfaces.SimplifiedChinese:  "您暂未获得此产品的使用授权，无法登录，请联系管理员",
					interfaces.TraditionalChinese: "您暫未獲得此產品的使用授權，無法登入，請聯繫管理員。",
					interfaces.AmericanEnglish:    "You do not currently have the authorization to use this product and cannot log in. Please contact the administrator.",
				},
				i18nIDHasNoLicense: {
					interfaces.SimplifiedChinese:  "产品无有效授权，无法登录，请联系管理员。",
					interfaces.TraditionalChinese: "產品無有效授權，無法登入，請聯繫管理員。",
					interfaces.AmericanEnglish:    "The product has no valid license. Login is unavailable. Please contact the administrator.",
				},
				i18nIDProductAuthorizedNotValid: {
					interfaces.SimplifiedChinese:  "产品“%s” 无有效授权。",
					interfaces.TraditionalChinese: "產品“%s” 無有效授權。",
					interfaces.AmericanEnglish:    "Product \"%s\" has no valid license.",
				},
				i18nIDProductAuthorizedOverQuota: {
					interfaces.SimplifiedChinese:  "产品“%s” 用户授权数已达上限。",
					interfaces.TraditionalChinese: "產品“%s” 用戶授權數已達上限。",
					interfaces.AmericanEnglish:    "The user license limit for products \"%s\" has been reached.",
				},
			}),
		}

		lic.event.RegisterUserCreated(lic.onUserCreated)
		lic.event.RegisterUserStatusChanged(lic.onUserStatusChanged)
		lic.event.RegisterUserDeleted(lic.onUserDeleted)

		lic.outbox.RegisterHandlers(outboxProductAuthorizedAddedLog, lic.onLogProductAuthorizedAdded)
		lic.outbox.RegisterHandlers(outboxProductAuthorizedUpdatedLog, lic.onLogProductAuthorizedUpdated)
		lic.outbox.RegisterHandlers(outboxProductAuthorizedDeletedLog, lic.onLogProductAuthorizedDeleted)

	})
	return lic
}

func (l *license) onLogProductAuthorizedUpdated(content interface{}) (err error) {
	msg := content.(map[string]interface{})
	name := msg["name"].(string)
	tempCurProducts := msg["currentProducts"].([]interface{})
	currentProducts := make([]string, 0)
	for _, v := range tempCurProducts {
		currentProducts = append(currentProducts, v.(string))
	}
	tempFutureProduct := msg["futureProducts"].([]interface{})
	futureProducts := make([]string, 0)
	for _, v := range tempFutureProduct {
		futureProducts = append(futureProducts, v.(string))
	}

	v := interfaces.Visitor{}
	err = mapstructure.Decode(msg["visitor"], &v)
	if err != nil {
		l.log.Errorf("license onLogProductAuthorizedUpdated decode visitor err: %v", err)
		return
	}

	err = l.eacpLog.OpUpdateAuthorizedProducts(&v, name, currentProducts, futureProducts)
	if err != nil {
		l.log.Errorf("license onLogProductAuthorizedUpdated op update authorized products err: %v", err)
		return
	}

	return nil
}

func (l *license) onLogProductAuthorizedDeleted(content interface{}) (err error) {
	msg := content.(map[string]interface{})
	name := msg["name"].(string)
	tempProducts := msg["products"].([]interface{})
	products := make([]string, 0)
	for _, v := range tempProducts {
		products = append(products, v.(string))
	}

	v := interfaces.Visitor{}
	err = mapstructure.Decode(msg["visitor"], &v)
	if err != nil {
		l.log.Errorf("license onLogProductAuthorizedDeleted decode visitor err: %v", err)
		return
	}

	err = l.eacpLog.OpDeleteAuthorizedProducts(&v, name, products)
	if err != nil {
		l.log.Errorf("license onLogProductAuthorizedDeleted op delete authorized products err: %v", err)
		return
	}

	return nil
}

func (l *license) onLogProductAuthorizedAdded(content interface{}) (err error) {
	msg := content.(map[string]interface{})
	name := msg["name"].(string)
	tempProducts := msg["products"].([]interface{})
	products := make([]string, 0)
	for _, v := range tempProducts {
		products = append(products, v.(string))
	}

	v := interfaces.Visitor{}
	err = mapstructure.Decode(msg["visitor"], &v)
	if err != nil {
		l.log.Errorf("license onLogProductAuthorizedAdded decode visitor err: %v", err)
		return
	}

	err = l.eacpLog.OpAddAuthorizedProducts(&v, name, products)
	if err != nil {
		l.log.Errorf("license onLogProductAuthorizedAdded op add authorized products err: %v", err)
		return
	}

	return nil
}

func (l *license) onUserDeleted(userID string) (err error) {
	// trace
	l.trace.SetInternalSpanName("业务逻辑-用户删除，处理用户产品授权")
	newCtx, span := l.trace.AddInternalTrace(context.Background())
	defer func() { l.trace.TelemetrySpanEnd(span, err) }()

	err = l.userDeleteAllProducts(newCtx, userID)
	if err != nil {
		l.log.Errorf("license onUserDeleted err: %v", err)
		return
	}

	return nil
}

func (l *license) onUserStatusChanged(userID string, status bool) (err error) {
	// trace
	l.trace.SetInternalSpanName("业务逻辑-用户状态改变，处理用户产品授权")
	newCtx, span := l.trace.AddInternalTrace(context.Background())
	defer func() { l.trace.TelemetrySpanEnd(span, err) }()

	if status {
		err = l.userAddAllProducts(newCtx, userID)
	} else {
		err = l.userDeleteAllProducts(newCtx, userID)
	}

	if err != nil {
		l.log.Errorf("license onUserStatusChanged err: %v", err)
		return
	}

	return nil
}

func (l *license) onUserCreated(userID string) (err error) {
	// trace
	l.trace.SetInternalSpanName("业务逻辑-用户创建")
	newCtx, span := l.trace.AddInternalTrace(context.Background())
	defer func() { l.trace.TelemetrySpanEnd(span, err) }()

	err = l.userAddAllProducts(newCtx, userID)
	if err != nil {
		l.log.Errorf("license onUserCreated err: %v", err)
		return
	}

	return nil
}

func (l *license) userDeleteAllProducts(ctx context.Context, userID string) (err error) {
	// 开始处理事务
	tx, err := l.tracePool.Begin()
	if err != nil {
		l.log.Errorf("license userDeleteAllProducts begin tx err: %v", err)
		return
	}

	// 异常时Rollback
	defer func() {
		switch err {
		case nil:
			// 提交事务
			if err = tx.Commit(); err != nil {
				l.log.Errorf("license userDeleteAllProducts Transaction Commit Error:%v", err)
				return
			}
		default:
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				l.log.Errorf("license userDeleteAllProducts Rollback err:%v", rollbackErr)
			}
		}
	}()

	err = l.db.DeleteUserAuthorizedProducts(ctx, userID, tx)
	if err != nil {
		l.log.Errorf("license userDeleteAllProducts delete authorized products err: %v", err)
		return
	}

	return nil
}

func (l *license) userAddAllProducts(ctx context.Context, userID string) (err error) {
	// 获取许可证信息
	infos, err := l.getLicensesFromDefaultConfig(ctx)
	if err != nil {
		l.log.Errorf("license userAddAllProducts get licenses err: %v", err)
		return
	}

	if len(infos) == 0 {
		return nil
	}

	// 获取用户已授权产品数量
	// 预防消息多次消费
	userAuthorizedProducts, err := l.db.GetAuthorizedProducts(ctx, []string{userID})
	if err != nil {
		l.log.Errorf("license userAddAllProducts get user authorized products err: %v", err)
		return
	}
	mapCurProducts := make(map[string]bool)
	if _, ok := userAuthorizedProducts[userID]; ok {
		for _, product := range userAuthorizedProducts[userID].Product {
			mapCurProducts[product] = true
		}
	}

	// 判断新增授权是否会导致超过授权量
	productes := make([]interfaces.ProductInfo, 0)
	for k := range infos {
		// 如果用户已授权产品中包含该产品，则跳过
		if _, ok := mapCurProducts[k]; ok {
			continue
		}

		var count int
		count, err = l.db.GetProductsAuthorizedCount(ctx, k)
		if err != nil {
			l.log.Errorf("license userAddAllProducts get products authorized count err: %v", err)
			return
		}

		// 如果许可证总授权量不为-1，则判断是否超过授权量
		if infos[k].TotalUserQuota != -1 && count+1 > infos[k].TotalUserQuota {
			l.log.Errorf("user authorized failed, license userAddAllProducts product %s is over the quota, userID: %s", k, userID)
			// 如果超过授权，则跳过此用户授权
			continue
		}

		productes = append(productes, interfaces.ProductInfo{
			AccountID: userID,
			Product:   k,
		})
	}

	if len(productes) == 0 {
		return nil
	}

	// 开始处理事务
	tx, err := l.tracePool.Begin()
	if err != nil {
		l.log.Errorf("license userAddAllProducts begin tx err: %v", err)
		return
	}

	// 异常时Rollback
	defer func() {
		switch err {
		case nil:
			// 提交事务
			if err = tx.Commit(); err != nil {
				l.log.Errorf("license userAddAllProducts Transaction Commit Error:%v", err)
				return
			}
		default:
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				l.log.Errorf("license userAddAllProducts Rollback err:%v", rollbackErr)
			}
		}
	}()

	// 再新增用户产品
	err = l.db.AddAuthorizedProducts(ctx, productes, tx)
	if err != nil {
		l.log.Errorf("license userAddAllProducts add authorized products err: %v", err)
		return
	}

	return nil
}

// GetLicenses 获取许可证
func (l *license) GetLicenses(ctx context.Context, visitor *interfaces.Visitor) (infos map[string]interfaces.LicenseInfo, err error) {
	// trace
	l.trace.SetInternalSpanName("业务逻辑-获取许可证")
	newCtx, span := l.trace.AddInternalTrace(ctx)
	defer func() { l.trace.TelemetrySpanEnd(span, err) }()
	// 应用账户不支持
	if visitor.Type != interfaces.RealName {
		err = gerrors.NewError(gerrors.PublicForbidden, "only real name user is supported")
		return
	}

	// 校验角色，必须是管理员
	userInfos, err := l.dnUserManagement.GetUserInfos(newCtx, []string{visitor.ID})
	if err != nil {
		l.log.Errorf("license GetLicenses get user infos err: %v", err)
		return
	}

	if !userInfos[visitor.ID].Roles[interfaces.SystemRoleSuperAdmin] &&
		!userInfos[visitor.ID].Roles[interfaces.SystemRoleSysAdmin] &&
		!userInfos[visitor.ID].Roles[interfaces.SystemRoleSecAdmin] &&
		!userInfos[visitor.ID].Roles[interfaces.SystemRoleAuditAdmin] &&
		!userInfos[visitor.ID].Roles[interfaces.SystemRoleOrgManager] &&
		!userInfos[visitor.ID].Roles[interfaces.SystemRoleOrgAudit] {
		err = gerrors.NewError(gerrors.PublicForbidden, "this user has no permission to access this resource")
		return
	}

	// 1、获取许可证
	licenses, err := l.getLicenses(newCtx)
	if err != nil {
		l.log.Errorf("license GetLicenses get licenses err: %v", err)
		return
	}

	// 获取当前许可证下已授权人数
	infos = make(map[string]interfaces.LicenseInfo)
	for k := range licenses {
		var authorizedUserCount int
		authorizedUserCount, err = l.db.GetProductsAuthorizedCount(newCtx, licenses[k].Product)
		if err != nil {
			l.log.Errorf("license GetLicenses get authorized user count err: %v", err)
			return
		}
		infos[k] = interfaces.LicenseInfo{
			Product:             k,
			TotalUserQuota:      licenses[k].TotalUserQuota,
			AuthorizedUserCount: authorizedUserCount,
		}
	}

	return
}

// GetAuthorizedProducts 获取已授权产品
// 宽松模式，不检查用户id是否存在或者重复
func (l *license) GetAuthorizedProducts(ctx context.Context, visitor *interfaces.Visitor, userIDs []string) (products map[string]interfaces.AuthorizedProduct, err error) {
	// trace
	l.trace.SetInternalSpanName("业务逻辑-获取已授权产品")
	newCtx, span := l.trace.AddInternalTrace(ctx)
	defer func() { l.trace.TelemetrySpanEnd(span, err) }()

	// 应用账户不支持
	if visitor.Type != interfaces.RealName {
		err = gerrors.NewError(gerrors.PublicForbidden, "only real name user is supported")
		return
	}

	// 校验角色，必须是管理员
	userInfos, err := l.dnUserManagement.GetUserInfos(newCtx, []string{visitor.ID})
	if err != nil {
		l.log.Errorf("license GetAuthorizedProducts get user infos err: %v", err)
		return
	}

	if !userInfos[visitor.ID].Roles[interfaces.SystemRoleSuperAdmin] &&
		!userInfos[visitor.ID].Roles[interfaces.SystemRoleSysAdmin] &&
		!userInfos[visitor.ID].Roles[interfaces.SystemRoleSecAdmin] &&
		!userInfos[visitor.ID].Roles[interfaces.SystemRoleAuditAdmin] &&
		!userInfos[visitor.ID].Roles[interfaces.SystemRoleOrgManager] &&
		!userInfos[visitor.ID].Roles[interfaces.SystemRoleOrgAudit] {
		err = gerrors.NewError(gerrors.PublicForbidden, "this user has no permission to access this resource")
		return
	}

	// userIDs去重
	userIDs = RemoveDuplicate(userIDs)

	// 获取已授权产品
	var tempProducts map[string]interfaces.AuthorizedProduct
	tempProducts, err = l.db.GetAuthorizedProducts(newCtx, userIDs)
	if err != nil {
		l.log.Errorf("license GetAuthorizedProducts get authorized products err: %v", err)
		return
	}

	// 用户信息处理
	products = make(map[string]interfaces.AuthorizedProduct)
	for k := range userIDs {
		temp := interfaces.AuthorizedProduct{
			ID:      userIDs[k],
			Type:    interfaces.ObjectTypeUser,
			Product: make([]string, 0),
		}

		if _, ok := tempProducts[userIDs[k]]; ok {
			temp.Product = RemoveDuplicate(tempProducts[userIDs[k]].Product)
		}
		products[userIDs[k]] = temp
	}

	return products, nil
}

// CheckProductAuthorized 检查产品是否已授权
func (l *license) CheckProductAuthorized(ctx context.Context, visitor *interfaces.Visitor, product string) (authorized bool, unauthorizedReason string, err error) {
	// trace
	l.trace.SetInternalSpanName("业务逻辑-检查产品是否已授权")
	newCtx, span := l.trace.AddInternalTrace(ctx)
	defer func() { l.trace.TelemetrySpanEnd(span, err) }()

	// 应用账户不支持
	if visitor.Type != interfaces.RealName {
		err = gerrors.NewError(gerrors.PublicForbidden, "only real name user is supported")
		return
	}

	// 获取许可证信息
	licenses, err := l.getLicensesClient(newCtx)
	if err != nil {
		l.log.Errorf("license CheckProductAuthorized get licenses err: %v", err)
		return
	}

	// 判断许可证是否有效
	if _, ok := licenses[product]; !ok {
		return false, l.i18n.Load(i18nIDHasNoLicense, visitor.Language), nil
	}

	// 获取用户授权信息
	authorizedProducts, err := l.db.GetAuthorizedProducts(newCtx, []string{visitor.ID})
	if err != nil {
		l.log.Errorf("license CheckProductAuthorized get authorized user count err: %v", err)
		return
	}

	// 判断用户是否已授权
	for _, p := range authorizedProducts[visitor.ID].Product {
		if p == product {
			return true, "", nil
		}
	}

	return false, l.i18n.Load(i18nIDUserHasNoAuthUserProduct, visitor.Language), nil
}

// UpdateAuthorizedProducts 更新已授权产品
func (l *license) UpdateAuthorizedProducts(ctx context.Context, visitor *interfaces.Visitor, products map[string]interfaces.AuthorizedProduct) (err error) {
	// trace
	l.trace.SetInternalSpanName("业务逻辑-更新已授权产品")
	newCtx, span := l.trace.AddInternalTrace(ctx)
	defer func() { l.trace.TelemetrySpanEnd(span, err) }()

	// 应用账户不支持
	if visitor.Type != interfaces.RealName {
		err = gerrors.NewError(gerrors.PublicForbidden, "only real name user is supported")
		return
	}

	// 获取用户信息，包括管理员和普通用户
	userIDs := make([]string, 0)
	allUserIDs := make([]string, 0)
	for k := range products {
		userIDs = append(userIDs, k)
		allUserIDs = append(allUserIDs, k)
	}

	allUserIDs = append(allUserIDs, visitor.ID)
	allUserIDs = RemoveDuplicate(allUserIDs)
	userInfos, err := l.dnUserManagement.GetUserInfos(newCtx, allUserIDs)
	if err != nil {
		l.log.Errorf("license UpdateAuthorizedProducts get user infos err: %v", err)
		return
	}

	if !userInfos[visitor.ID].Roles[interfaces.SystemRoleSuperAdmin] &&
		!userInfos[visitor.ID].Roles[interfaces.SystemRoleSysAdmin] &&
		!userInfos[visitor.ID].Roles[interfaces.SystemRoleSecAdmin] &&
		!userInfos[visitor.ID].Roles[interfaces.SystemRoleAuditAdmin] &&
		!userInfos[visitor.ID].Roles[interfaces.SystemRoleOrgManager] &&
		!userInfos[visitor.ID].Roles[interfaces.SystemRoleOrgAudit] {
		err = gerrors.NewError(gerrors.PublicForbidden, "this user has no permission to access this resource")
		return
	}

	// 获取当前所有的许可
	licenses, err := l.getLicenses(newCtx)
	if err != nil {
		l.log.Errorf("license UpdateAuthorizedProducts get licenses err: %v", err)
		return
	}

	currentLicenses := make(map[string]bool)
	for k := range licenses {
		currentLicenses[k] = true
	}

	// 检查需要修改的用户的产品是否在当前许可证中
	notExistProducts := make([]string, 0)
	for k := range products {
		for _, product := range products[k].Product {
			if _, ok := currentLicenses[product]; !ok {
				notExistProducts = append(notExistProducts, product)
			}
		}
	}
	notExistProducts = RemoveDuplicate(notExistProducts)

	// 获取当前用户有哪些授权产品
	curUserProducts, err := l.db.GetAuthorizedProducts(newCtx, userIDs)
	if err != nil {
		l.log.Errorf("license UpdateAuthorizedProducts get authorized products err: %v", err)
		return
	}

	// 获取哪些用户需要新增授权，哪些用户需要删除授权
	needAddProducts := make([]interfaces.ProductInfo, 0)
	needDeleteProducts := make([]interfaces.ProductInfo, 0)
	mapProductChange := make(map[string]int)

	addProductUserInfos := make(map[string][]string)
	deleteProductUserInfos := make(map[string][]string)
	updateProductUserInfos := make(map[string]interfaces.LogProductInfo)

	for _, userID := range userIDs {
		userCurProducts := curUserProducts[userID].Product
		userFutureProducts := products[userID].Product

		// 日志信息
		if len(userCurProducts) == 0 && len(userFutureProducts) > 0 {
			addProductUserInfos[userInfos[userID].Name] = userFutureProducts
		}

		if len(userCurProducts) > 0 && len(userFutureProducts) == 0 {
			deleteProductUserInfos[userInfos[userID].Name] = userCurProducts
		}

		if len(userCurProducts) > 0 && len(userFutureProducts) > 0 {
			updateProductUserInfos[userInfos[userID].Name] = interfaces.LogProductInfo{
				CurrentProducts: userCurProducts,
				FutureProducts:  userFutureProducts,
			}
		}

		// 产品列表map
		userMapFutureProducts := make(map[string]bool)
		userMapCurProducts := make(map[string]bool)
		for _, product := range userCurProducts {
			userMapCurProducts[product] = true
		}
		for _, product := range userFutureProducts {
			userMapFutureProducts[product] = true
		}

		// 获取哪些需要新增的授权
		for _, product := range userFutureProducts {
			if _, ok := userMapCurProducts[product]; !ok {
				needAddProducts = append(needAddProducts, interfaces.ProductInfo{
					AccountID: userID,
					Product:   product,
				})
				mapProductChange[product]++
			}
		}

		// 获取哪些需要删除的授权
		for _, product := range userCurProducts {
			if _, ok := userMapFutureProducts[product]; !ok {
				needDeleteProducts = append(needDeleteProducts, interfaces.ProductInfo{
					AccountID: userID,
					Product:   product,
				})
				mapProductChange[product]--
			}
		}
	}

	// 检查产品的数量是否超标
	overQuotaProducts := make([]string, 0)
	for product, change := range mapProductChange {
		// 如果没有授权产品，则跳过
		if _, ok := licenses[product]; !ok {
			continue
		}

		// 如果许可证总授权量不为-1，则判断是否超过授权量
		if change > 0 && licenses[product].TotalUserQuota != -1 {
			var count int
			count, err = l.db.GetProductsAuthorizedCount(newCtx, product)
			if err != nil {
				l.log.Errorf("license UpdateAuthorizedProducts get products authorized count err: %v", err)
				return
			}
			if count+change > licenses[product].TotalUserQuota {
				overQuotaProducts = append(overQuotaProducts, product)
			}
		}
	}

	// 当用户产品存在问题时，返回错误
	if len(notExistProducts) > 0 || len(overQuotaProducts) > 0 {
		msg := ""
		if len(overQuotaProducts) > 0 {
			tempMsg := l.i18n.Load(i18nIDProductAuthorizedOverQuota, visitor.Language)
			msg += fmt.Sprintf(tempMsg, strings.Join(overQuotaProducts, ","))
			if len(notExistProducts) > 0 {
				msg += "\u000A"
			}
		}

		if len(notExistProducts) > 0 {
			tempMsg := l.i18n.Load(i18nIDProductAuthorizedNotValid, visitor.Language)
			msg += fmt.Sprintf(tempMsg, strings.Join(notExistProducts, ","))
		}

		err = gerrors.NewError(StrProductAuthorizedNotValid, msg)
		return
	}

	// 开始处理事务
	tx, err := l.tracePool.Begin()
	if err != nil {
		l.log.Errorf("license UpdateAuthorizedProducts begin tx err: %v", err)
		return
	}

	// 异常时Rollback
	defer func() {
		switch err {
		case nil:
			// 提交事务
			if err = tx.Commit(); err != nil {
				l.log.Errorf("license UpdateAuthorizedProducts Transaction Commit Error:%v", err)
				return
			}

			l.outbox.NotifyPushOutboxThread()
		default:
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				l.log.Errorf("license UpdateAuthorizedProducts Rollback err:%v", rollbackErr)
			}
		}
	}()

	// 删除用户产品
	err = l.db.DeleteAuthorizedProducts(newCtx, needDeleteProducts, tx)
	if err != nil {
		l.log.Errorf("license UpdateAuthorizedProducts delete authorized products err: %v", err)
		return
	}

	// 新增用户产品
	err = l.db.AddAuthorizedProducts(newCtx, needAddProducts, tx)
	if err != nil {
		l.log.Errorf("license UpdateAuthorizedProducts add authorized products err: %v", err)
		return
	}

	// 记录日志
	for name, products := range addProductUserInfos {
		msg := make(map[string]interface{})
		msg["name"] = name
		msg["products"] = products
		msg["visitor"] = *visitor
		err = l.outbox.AddOutboxInfo(outboxProductAuthorizedAddedLog, msg, tx)
		if err != nil {
			l.log.Errorf("license UpdateAuthorizedProducts add outbox info err: %v", err)
			return
		}
	}

	for name, products := range deleteProductUserInfos {
		msg := make(map[string]interface{})
		msg["name"] = name
		msg["products"] = products
		msg["visitor"] = *visitor
		err = l.outbox.AddOutboxInfo(outboxProductAuthorizedDeletedLog, msg, tx)
		if err != nil {
			l.log.Errorf("license UpdateAuthorizedProducts add outbox info err: %v", err)
			return
		}
	}

	for name, products := range updateProductUserInfos {
		msg := make(map[string]interface{})
		msg["name"] = name
		msg["currentProducts"] = products.CurrentProducts
		msg["futureProducts"] = products.FutureProducts
		msg["visitor"] = *visitor
		err = l.outbox.AddOutboxInfo(outboxProductAuthorizedUpdatedLog, msg, tx)
		if err != nil {
			l.log.Errorf("license UpdateAuthorizedProducts add outbox info err: %v", err)
			return
		}
	}

	return nil
}

func (l *license) getLicenses(ctx context.Context) (infos map[string]interfaces.License, err error) {
	// 如果获取时间超过5秒，则重新获取
	now := time.Now()
	if now.Sub(licenseCacheGetTime) <= 5*time.Second {
		return licenseCache, nil
	}

	// 获取许可证信息，并缓存
	licenses, err := l.dnLicense.GetLicenses(ctx)
	if err != nil {
		l.log.Errorf("license getLicenses get licenses err: %v", err)
		return
	}
	licenseCache = licenses
	licenseCacheGetTime = now
	return licenses, nil
}

func (l *license) getLicensesClient(ctx context.Context) (infos map[string]interfaces.License, err error) {
	// 如果获取时间超过5分钟，则重新获取
	now := time.Now()
	if now.Sub(licenseCacheGetTime) <= 5*time.Minute {
		return licenseCache, nil
	}

	// 获取许可证信息，并缓存
	licenses, err := l.dnLicense.GetLicenses(ctx)
	if err != nil {
		l.log.Errorf("license getLicenses get licenses err: %v", err)
		return
	}
	licenseCache = licenses
	licenseCacheGetTime = now
	return licenses, nil
}

func (l *license) getLicensesFromDefaultConfig(ctx context.Context) (infos map[string]interfaces.License, err error) {
	// 如果获取时间超过5秒，则重新获取
	licenses := licenseCache
	now := time.Now()
	if now.Sub(licenseCacheGetTime) > 5*time.Second {
		licenses, err = l.dnLicense.GetLicenses(ctx)
		if err != nil {
			l.log.Errorf("license getLicenses get licenses err: %v", err)
			return
		}
		licenseCache = licenses
		licenseCacheGetTime = now
	}

	// 从数据库获取配置
	config, err := l.dbConfig.GetConfig(ctx, "default_user_license")
	if err != nil {
		l.log.Errorf("license getLicensesFromDefaultConfig get config err: %v", err)
		return
	}

	// 如果配置为空，则用当前所有的许可证
	if config == "" {
		return licenses, nil
	}

	//["", "product1", "product2"]json字符转换为[]string
	productList := make([]string, 0)
	err = json.Unmarshal([]byte(config), &productList)
	if err != nil {
		l.log.Errorf("license getLicensesFromDefaultConfig unmarshal config err: %v", err)
		return licenses, nil
	}
	mapDefaultProducts := make(map[string]bool)
	for _, product := range productList {
		mapDefaultProducts[product] = true
	}

	// 如果配置不为空，则用当前许可证中在配置中的许可证
	infos = make(map[string]interfaces.License)
	for _, license := range licenses {
		if _, ok := mapDefaultProducts[license.Product]; ok {
			infos[license.Product] = license
		}
	}

	return infos, nil
}
