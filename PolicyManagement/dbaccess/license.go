package dbaccess

import (
	"context"
	"database/sql"
	"fmt"
	"policy_mgnt/common"
	"policy_mgnt/interfaces"
	"sync"

	"github.com/kweaver-ai/proton-rds-sdk-go/sqlx"

	"github.com/kweaver-ai/go-lib/observable"
)

var (
	licenseOnce sync.Once
	lice        *license
)

type license struct {
	db    *sqlx.DB
	log   common.Logger
	trace observable.Tracer
}

func NewDBLicense() *license {
	licenseOnce.Do(func() {
		lice = &license{
			db:    dbTracePool,
			log:   common.NewLogger(),
			trace: common.SvcARTrace,
		}
	})
	return lice
}

func (d *license) GetAuthorizedProducts(ctx context.Context, userIDs []string) (products map[string]interfaces.AuthorizedProduct, err error) {
	// trace
	d.trace.SetClientSpanName("数据访问层-获取已授权产品")
	newCtx, span := d.trace.AddClientTrace(ctx)
	defer func() { d.trace.TelemetrySpanEnd(span, err) }()

	// 分批获取 每次1000条
	products = make(map[string]interfaces.AuthorizedProduct)
	for i := 0; i < len(userIDs); i += 1000 {
		end := i + 1000
		if end > len(userIDs) {
			end = len(userIDs)
		}

		var tempProducts map[string]interfaces.AuthorizedProduct
		tempProducts, err = d.getAuthorizedProducts(newCtx, userIDs[i:end])
		if err != nil {
			d.log.Errorf("license GetAuthorizedProducts get authorized products err: %v", err)
			return
		}

		for k, v := range tempProducts {
			products[k] = v
		}
	}

	return products, nil
}

func (d *license) getAuthorizedProducts(ctx context.Context, userIDs []string) (products map[string]interfaces.AuthorizedProduct, err error) {
	// 查询已授权产品
	products = make(map[string]interfaces.AuthorizedProduct)
	if len(userIDs) == 0 {
		return
	}

	userSet, userArgIDs := GetFindInSetSQL(userIDs)
	query := "SELECT f_account_id, f_product FROM %s.t_product_relation WHERE f_account_id IN ( "
	query += userSet
	query += " )"

	dbName := common.GetDBName("policy_mgnt")
	query = fmt.Sprintf(query, dbName)
	rows, err := d.db.QueryContext(ctx, query, userArgIDs...)
	if err != nil {
		d.log.Errorf("license GetAuthorizedProducts query err: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var accountID string
		var product string
		err = rows.Scan(&accountID, &product)
		if err != nil {
			d.log.Errorf("license GetAuthorizedProducts scan err: %v", err)
			return
		}

		if data, ok := products[accountID]; !ok {
			products[accountID] = interfaces.AuthorizedProduct{
				ID:      accountID,
				Type:    interfaces.ObjectTypeUser,
				Product: []string{product},
			}
		} else {
			data.Product = append(data.Product, product)
			products[accountID] = data
		}
	}

	return products, nil
}

func (d *license) DeleteAuthorizedProducts(ctx context.Context, products []interfaces.ProductInfo, tx *sql.Tx) (err error) {
	// 删除已授权产品
	if len(products) == 0 {
		return
	}

	// 根据产品分组，key为product，value为accountID列表
	productInfos := make(map[string][]string)
	for k := range products {
		if productInfos[products[k].Product] == nil {
			productInfos[products[k].Product] = make([]string, 0)
		}
		productInfos[products[k].Product] = append(productInfos[products[k].Product], products[k].AccountID)
	}

	// 根据产品进行批量删除
	for k := range productInfos {
		err = d.deleteAuthorizedProducts(ctx, k, productInfos[k], tx)
		if err != nil {
			d.log.Errorf("license DeleteAuthorizedProducts delete authorized products err: %v", err)
			return
		}
	}

	return nil
}

func (d *license) deleteAuthorizedProducts(ctx context.Context, product string, userIDs []string, tx *sql.Tx) (err error) {
	// 分批删除 每次1000条
	for i := 0; i < len(userIDs); i += 1000 {
		end := i + 1000
		if end > len(userIDs) {
			end = len(userIDs)
		}
		err = d.deleteAuthorizedProductsSingle(ctx, product, userIDs[i:end], tx)
		if err != nil {
			d.log.Errorf("license DeleteAuthorizedProducts delete authorized products err: %v", err)
			return
		}
	}
	return nil
}

func (d *license) deleteAuthorizedProductsSingle(ctx context.Context, product string, userIDs []string, tx *sql.Tx) (err error) {
	// 删除已授权产品
	if len(userIDs) == 0 {
		return
	}

	// 拼接用户id
	userSet, userArgIDs := GetFindInSetSQL(userIDs)
	query := "DELETE FROM %s.t_product_relation WHERE f_account_id IN ( "
	query += userSet
	query += " ) AND f_product = ?"
	userArgIDs = append(userArgIDs, product)

	dbName := common.GetDBName("policy_mgnt")
	query = fmt.Sprintf(query, dbName)
	_, err = tx.ExecContext(ctx, query, userArgIDs...)
	if err != nil {
		d.log.Errorf("license DeleteAuthorizedProducts delete authorized products err: %v", err)
		return
	}
	return nil
}

// AddAuthorizedProducts 新增已授权产品
func (d *license) AddAuthorizedProducts(ctx context.Context, products []interfaces.ProductInfo, tx *sql.Tx) (err error) {
	// 每1000条分批新增
	for i := 0; i < len(products); i += 1000 {
		end := i + 1000
		if end > len(products) {
			end = len(products)
		}
		err = d.addAuthorizedProductsSingle(ctx, products[i:end], tx)
		if err != nil {
			d.log.Errorf("license AddAuthorizedProducts add authorized products err: %v", err)
			return
		}
	}
	return nil
}

func (d *license) addAuthorizedProductsSingle(ctx context.Context, products []interfaces.ProductInfo, tx *sql.Tx) (err error) {
	// 新增已授权产品
	if len(products) == 0 {
		return
	}

	// 拼接用户id
	args := make([]interface{}, 0)
	query := "INSERT INTO %s.t_product_relation (f_account_id, f_product, f_account_type) VALUES "

	dbName := common.GetDBName("policy_mgnt")
	query = fmt.Sprintf(query, dbName)
	for i := 0; i < len(products); i++ {
		query += "( ?, ?, ? ),"
		args = append(args, products[i].AccountID, products[i].Product, 1)
	}
	query = query[:len(query)-1]
	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		d.log.Errorf("license AddAuthorizedProducts add authorized products err: %v", err)
	}
	return
}

func (d *license) DeleteUserAuthorizedProducts(ctx context.Context, userID string, tx *sql.Tx) (err error) {
	// 删除用户已授权产品
	query := "DELETE FROM %s.t_product_relation WHERE f_account_id = ?"
	dbName := common.GetDBName("policy_mgnt")
	query = fmt.Sprintf(query, dbName)
	_, err = tx.ExecContext(ctx, query, userID)
	if err != nil {
		d.log.Errorf("license DeleteUserAuthorizedProducts delete user authorized products err: %v", err)
		return
	}
	return
}

func (d *license) GetProductsAuthorizedCount(ctx context.Context, product string) (count int, err error) {
	// 查询已授权用户数量
	query := "SELECT COUNT(f_account_id) FROM %s.t_product_relation WHERE f_product = ?"
	dbName := common.GetDBName("policy_mgnt")
	query = fmt.Sprintf(query, dbName)
	rows, err := d.db.QueryContext(ctx, query, product)
	if err != nil {
		d.log.Errorf("license GetProductsAuthorizedCount query err: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&count)
		if err != nil {
			d.log.Errorf("license GetProductsAuthorizedCount scan err: %v", err)
			return
		}
	}

	return count, nil
}
