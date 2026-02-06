package dbaccess

import (
	"context"
	"fmt"
	"policy_mgnt/common"
	"sync"

	"github.com/kweaver-ai/proton-rds-sdk-go/sqlx"

	"github.com/kweaver-ai/go-lib/observable"
)

var (
	configOnce sync.Once
	confi      *config
)

type config struct {
	db    *sqlx.DB
	log   common.Logger
	trace observable.Tracer
}

func NewDBConfig() *config {
	configOnce.Do(func() {
		confi = &config{
			db:    dbTracePool,
			log:   common.NewLogger(),
			trace: common.SvcARTrace,
		}
	})
	return confi
}

func (d *config) GetConfig(ctx context.Context, key string) (value string, err error) {
	// trace
	d.trace.SetClientSpanName("数据访问层-获取配置")
	newCtx, span := d.trace.AddClientTrace(ctx)
	defer func() { d.trace.TelemetrySpanEnd(span, err) }()

	// 查询已授权用户数量
	query := "SELECT value from %s.option where `key` = ?"

	dbName := common.GetDBName("user_management")
	query = fmt.Sprintf(query, dbName)
	rows, err := d.db.QueryContext(newCtx, query, key)
	if err != nil {
		d.log.Errorf("license GetConfig query err: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&value)
		if err != nil {
			d.log.Errorf("license GetConfig scan err: %v", err)
			return
		}
	}

	return value, nil
}
