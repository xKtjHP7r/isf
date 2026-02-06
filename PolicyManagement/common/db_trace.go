// Package common dbTracePool
package common

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/kweaver-ai/go-lib/observable"
	protonRDS "github.com/kweaver-ai/proton-rds-sdk-go/driver"
	"github.com/kweaver-ai/proton-rds-sdk-go/sqlx"
	"github.com/qustavo/sqlhooks/v2"
)

var (
	dbTraceOnce sync.Once
	dbTracePool *sqlx.DB = nil
)

const (
	traceDriverName = "rds-trace"
)

func initTraceHook() {
	hook := &observable.RDSHook{System: "rds"}
	sql.Register(traceDriverName, sqlhooks.Wrap(new(protonRDS.RDSDriver), hook))
}

// NewDBTracePool 获取带有trace数据库连接池
func NewDBTracePool() *sqlx.DB {
	dbTraceOnce.Do(func() {
		initTraceHook()
		dbLog := NewLogger()

		// 获取DB_PORT
		port, err := strconv.Atoi(os.Getenv("DB_PORT"))
		if err != nil {
			panic(err)
		}

		connInfo := sqlx.DBConfig{
			Host:         os.Getenv("DB_HOST"),
			Port:         port,
			User:         os.Getenv("DB_USER"),
			Password:     os.Getenv("DB_PASSWORD"),
			CustomDriver: traceDriverName,
			Database:     GetDBName("policy_mgnt"),
			Charset:      "utf8",
			ParseTime:    "True",
			Loc:          "Local",
		}

		connInfo.CustomDriver = traceDriverName

		dbTracePool, err = sqlx.NewDB(&connInfo)
		if err != nil {
			// 判断err里
			if err.Error() == "driver must implement driver.ConnBeginTx" {
				connInfo.CustomDriver = "proton-rds"
				dbTracePool, err = sqlx.NewDB(&connInfo)
			}
			if err != nil {
				dbLog.Errorf("new db operator failed; error:%s, connInfo:%+v, configLoader.DB:%+v",
					err.Error(), connInfo, connInfo.Database)
				panic(err)
			}
		}
	})

	return dbTracePool
}

// GetDBName 获取数据库名
func GetDBName(dbName string) string {
	return fmt.Sprintf("%s%s", os.Getenv("SYSTEM_ID"), dbName)
}
