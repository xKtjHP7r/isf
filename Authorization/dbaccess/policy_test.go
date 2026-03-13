package dbaccess

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-playground/assert"
	"github.com/kweaver-ai/proton-rds-sdk-go/sqlx"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"

	"Authorization/common"
	"Authorization/interfaces"
)

const (
	invalidJSON = `{"invalid": "json"`
)

// makeRulesForTest 构造测试用 PolicyOperation（新格式：Allow/Deny 为 []PolicyConditionItem）
func makeRulesForTest(allowOps, denyOps []interfaces.PolicyOperationItem) interfaces.PolicyRules {
	rules := interfaces.PolicyRules{}
	if len(allowOps) > 0 {
		rules.Allow = []interfaces.PolicyRuleItem{{Condition: map[string]any{}, Operations: allowOps}}
	}
	if len(denyOps) > 0 {
		rules.Deny = []interfaces.PolicyRuleItem{{Condition: map[string]any{}, Operations: denyOps}}
	}
	return rules
}

//nolint:lll
func TestDBGetPagination(t *testing.T) {
	Convey("TestDBGetPagination", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func(db *sqlx.DB) {
			err := db.Close()
			if err != nil {
				return
			}
		}(db)

		mockErr := errors.New("test error")
		ctx := context.Background()

		b := &policy{
			db:     db,
			logger: common.NewLogger(),
		}

		params := interfaces.PolicyPagination{
			ResourceID:   "test-resource-id",
			ResourceType: "test-resource-type",
			Offset:       0,
			Limit:        10,
		}

		Convey("data query error", func() {
			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnError(mockErr)
			policies, err := b.GetPagination(ctx, params)
			assert.Equal(t, err, mockErr)
			assert.Equal(t, len(policies), 0)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("json unmarshal error", func() {
			operationJSON := invalidJSON
			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnRows(
				sqlmock.NewRows([]string{"f_id", "f_resource_id", "f_resource_type", "f_resource_name", "f_accessor_id", "f_accessor_type", "f_accessor_name", "f_operation", "f_condition", "f_end_time", "f_create_time", "f_modify_time"}).
					AddRow("policy-1", "resource-1", "type-1", "name-1", "accessor-1", interfaces.AccessorUser, "accessor-name-1", operationJSON, "condition-1", 1234567890, 1234567890, 1234567890),
			)
			policies, err := b.GetPagination(ctx, params)
			assert.NotEqual(t, err, nil)
			assert.Equal(t, len(policies), 0)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("Success", func() {
			rules := makeRulesForTest(
				[]interfaces.PolicyOperationItem{{ID: "op1", Name: "operation1"}},
				[]interfaces.PolicyOperationItem{{ID: "op2", Name: "operation2"}},
			)
			operationJSON, _ := b.rulesInfoToString(rules)

			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnRows(
				sqlmock.NewRows([]string{"f_id", "f_resource_id", "f_resource_type", "f_resource_name", "f_accessor_id", "f_accessor_type", "f_accessor_name", "f_operation", "f_condition", "f_end_time", "f_create_time", "f_modify_time"}).
					AddRow("policy-1", "resource-1", "type-1", "name-1", "accessor-1", interfaces.AccessorUser, "accessor-name-1", operationJSON, "condition-1", 1234567890, 1234567890, 1234567890),
			)
			policies, err := b.GetPagination(ctx, params)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(policies), 1)
			assert.Equal(t, policies[0].ID, "policy-1")
			assert.Equal(t, policies[0].ResourceID, "resource-1")
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})
	})
}

func TestDBCreate(t *testing.T) {
	Convey("TestDBCreate", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func(db *sqlx.DB) {
			err = db.Close()
			if err != nil {
				return
			}
		}(db)

		mockErr := errors.New("test error")
		ctx := context.Background()

		b := &policy{
			db:     db,
			logger: common.NewLogger(),
		}

		mock.ExpectBegin()
		var tx *sql.Tx
		tx, err = db.Begin()
		assert.Equal(t, err, nil)

		policies := []interfaces.PolicyInfo{
			{
				ID:           "policy-1",
				ResourceID:   "resource-1",
				ResourceType: "type-1",
				ResourceName: "name-1",
				AccessorID:   "accessor-1",
				AccessorType: interfaces.AccessorUser,
				AccessorName: "accessor-name-1",
				Rules:        makeRulesForTest([]interfaces.PolicyOperationItem{{ID: "op1", Name: "operation1"}}, nil),
				Condition:    "condition-1",
				EndTime:      1234567890,
			},
		}

		Convey("empty policies", func() {
			mock.ExpectCommit()
			err := b.Create(ctx, []interfaces.PolicyInfo{}, tx)
			assert.Equal(t, err, nil)
			_ = tx.Commit()
		})

		Convey("insert error", func() {
			mock.ExpectRollback()
			mock.ExpectExec("^insert").WillReturnError(mockErr)
			err := b.Create(ctx, policies, tx)
			assert.NotEqual(t, err, nil)
			_ = tx.Rollback()
		})

		Convey("Success", func() {
			mock.ExpectCommit()
			mock.ExpectExec("^insert").WillReturnResult(sqlmock.NewResult(1, 1))
			err := b.Create(ctx, policies, tx)
			assert.NotEqual(t, err, nil)
			_ = tx.Commit()
		})
	})
}

func TestDBUpdate(t *testing.T) {
	Convey("TestDBUpdate", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func(db *sqlx.DB) {
			err = db.Close()
			if err != nil {
				return
			}
		}(db)

		mockErr := errors.New("test error")
		ctx := context.Background()

		b := &policy{
			db:     db,
			logger: common.NewLogger(),
		}

		policies := []interfaces.PolicyInfo{
			{
				ID:           "policy-1",
				ResourceID:   "resource-1",
				ResourceType: "type-1",
				ResourceName: "name-1",
				AccessorID:   "accessor-1",
				AccessorType: interfaces.AccessorUser,
				AccessorName: "accessor-name-1",
				Rules:        makeRulesForTest([]interfaces.PolicyOperationItem{{ID: "op1", Name: "operation1"}}, nil),
				Condition:    "condition-1",
				EndTime:      1234567890,
			},
		}

		mock.ExpectBegin()
		var tx *sql.Tx
		tx, err = db.Begin()
		assert.Equal(t, err, nil)

		Convey("update error", func() {
			mock.ExpectRollback()
			mock.ExpectExec("^update").WillReturnError(mockErr)
			err := b.Update(ctx, policies, tx)
			assert.NotEqual(t, err, nil)
			_ = tx.Rollback()
		})

		Convey("Success", func() {
			mock.ExpectCommit()
			mock.ExpectExec("^update").WillReturnResult(sqlmock.NewResult(0, 1))
			err := b.Update(ctx, policies, tx)
			assert.NotEqual(t, err, nil)
			_ = tx.Commit()
		})
	})
}

func TestDBDelete(t *testing.T) {
	Convey("TestDBDelete", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func(db *sqlx.DB) {
			err = db.Close()
			if err != nil {
				return
			}
		}(db)

		mockErr := errors.New("test error")
		ctx := context.Background()
		mock.ExpectBegin()
		var tx *sql.Tx
		tx, err = db.Begin()
		assert.Equal(t, err, nil)

		b := &policy{
			db:     db,
			logger: common.NewLogger(),
		}

		ids := []string{"policy-1", "policy-2"}

		Convey("empty ids", func() {
			err := b.Delete(ctx, []string{}, tx)
			assert.Equal(t, err, nil)
		})

		Convey("delete error", func() {
			mock.ExpectExec("^delete").WillReturnError(mockErr)
			err := b.Delete(ctx, ids, tx)
			assert.Equal(t, err, mockErr)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("Success", func() {
			mock.ExpectExec("^delete").WillReturnResult(sqlmock.NewResult(0, 2))
			err := b.Delete(ctx, ids, tx)
			assert.Equal(t, err, nil)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})
	})
}

func TestDBGetByResourceIDs(t *testing.T) {
	Convey("TestDBGetByResourceIDs", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func(db *sqlx.DB) {
			err := db.Close()
			if err != nil {
				return
			}
		}(db)

		mockErr := errors.New("test error")
		ctx := context.Background()

		b := &policy{
			db:     db,
			logger: common.NewLogger(),
		}

		resourceType := "test-resource-type"
		resourceIDs := []string{"resource-1", "resource-2"}

		Convey("query error", func() {
			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnError(mockErr)
			policiesMap, err := b.GetByResourceIDs(ctx, resourceType, resourceIDs)
			assert.Equal(t, err, mockErr)
			assert.Equal(t, len(policiesMap), 0)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("json unmarshal error", func() {
			operationJSON := invalidJSON
			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnRows(
				sqlmock.NewRows([]string{
					"f_id", "f_resource_id", "f_resource_type", "f_resource_name", "f_ancestors",
					"f_accessor_id", "f_accessor_type", "f_accessor_name", "f_operation", "f_condition", "f_end_time", "f_create_time",
				}).
					AddRow("policy-1", "resource-1", "type-1", "name-1", "[]", "accessor-1", interfaces.AccessorUser,
						"accessor-name-1", operationJSON, "condition-1", 1234567890, 1234567890),
			)
			policiesMap, err := b.GetByResourceIDs(ctx, resourceType, resourceIDs)
			assert.NotEqual(t, err, nil)
			assert.Equal(t, len(policiesMap), 0)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("Success", func() {
			rules := makeRulesForTest([]interfaces.PolicyOperationItem{{ID: "op1", Name: "operation1"}}, nil)
			operationJSON, _ := b.rulesInfoToString(rules)

			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnRows(
				sqlmock.NewRows([]string{
					"f_id", "f_resource_id", "f_resource_type",
					"f_resource_name", "f_ancestors", "f_accessor_id", "f_accessor_type", "f_accessor_name", "f_operation",
					"f_condition", "f_end_time", "f_create_time",
				}).
					AddRow("policy-1", "resource-1", "type-1", "name-1", "[]", "accessor-1",
						interfaces.AccessorUser, "accessor-name-1", operationJSON, "condition-1", 1234567890, 1234567890),
			)
			policiesMap, err := b.GetByResourceIDs(ctx, resourceType, resourceIDs)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(policiesMap), 1)
			assert.Equal(t, len(policiesMap["resource-1"]), 1)
			assert.Equal(t, policiesMap["resource-1"][0].ID, "policy-1")
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})
	})
}

func TestDBGetByPolicyIDs(t *testing.T) {
	Convey("TestDBGetByPolicyIDs", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func(db *sqlx.DB) {
			err := db.Close()
			if err != nil {
				return
			}
		}(db)

		mockErr := errors.New("test error")
		ctx := context.Background()

		b := &policy{
			db:     db,
			logger: common.NewLogger(),
		}

		policyIDs := []string{"policy-1", "policy-2"}

		Convey("empty policy ids", func() {
			policies, err := b.GetByPolicyIDs(ctx, []string{})
			assert.Equal(t, err, nil)
			assert.Equal(t, len(policies), 0)
		})

		Convey("query error", func() {
			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnError(mockErr)
			policies, err := b.GetByPolicyIDs(ctx, policyIDs)
			assert.Equal(t, err, mockErr)
			assert.Equal(t, len(policies), 0)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("json unmarshal error", func() {
			operationJSON := invalidJSON
			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnRows(
				sqlmock.NewRows([]string{
					"f_id", "f_resource_id", "f_resource_type", "f_resource_name", "f_ancestors", "f_accessor_id",
					"f_accessor_type", "f_accessor_name", "f_operation", "f_condition", "f_end_time", "f_create_time",
				}).
					AddRow("policy-1", "resource-1", "type-1", "name-1", "[]", "accessor-1", interfaces.AccessorUser, "accessor-name-1",
						operationJSON, "condition-1", 1234567890, 1234567890),
			)
			policies, err := b.GetByPolicyIDs(ctx, policyIDs)
			assert.NotEqual(t, err, nil)
			assert.Equal(t, len(policies), 0)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("Success", func() {
			rules := makeRulesForTest([]interfaces.PolicyOperationItem{{ID: "op1", Name: "operation1"}}, nil)
			operationJSON, _ := b.rulesInfoToString(rules)

			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnRows(
				sqlmock.NewRows([]string{
					"f_id", "f_resource_id", "f_resource_type", "f_resource_name", "f_ancestors", "f_accessor_id",
					"f_accessor_type", "f_accessor_name", "f_operation", "f_condition", "f_end_time", "f_create_time",
				}).
					AddRow("policy-1", "resource-1", "type-1", "name-1", "[]", "accessor-1", interfaces.AccessorUser,
						"accessor-name-1", operationJSON, "condition-1", 1234567890, 1234567890),
			)
			policies, err := b.GetByPolicyIDs(ctx, policyIDs)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(policies), 1)
			assert.Equal(t, policies["policy-1"].ID, "policy-1")
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})
	})
}

func TestDBDeleteByResourceIDs(t *testing.T) {
	Convey("TestDBDeleteByResourceIDs", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func(db *sqlx.DB) {
			err := db.Close()
			if err != nil {
				return
			}
		}(db)

		mockErr := errors.New("test error")
		ctx := context.Background()

		b := &policy{
			db:     db,
			logger: common.NewLogger(),
		}

		resources := []interfaces.PolicyDeleteResourceInfo{
			{ID: "resource-1", Type: "type-1"},
			{ID: "resource-2", Type: "type-2"},
		}

		Convey("empty resources", func() {
			err := b.DeleteByResourceIDs(ctx, []interfaces.PolicyDeleteResourceInfo{})
			assert.Equal(t, err, nil)
		})

		Convey("delete error", func() {
			mock.ExpectExec("^delete").WillReturnError(mockErr)
			err := b.DeleteByResourceIDs(ctx, resources)
			assert.Equal(t, err, mockErr)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("Success", func() {
			mock.ExpectExec("^delete").WillReturnResult(sqlmock.NewResult(0, 2))
			err := b.DeleteByResourceIDs(ctx, resources)
			assert.Equal(t, err, nil)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})
	})
}

//nolint:dupl
func TestDBDeleteByAccessorIDs(t *testing.T) {
	Convey("TestDBDeleteByAccessorIDs", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func(db *sqlx.DB) {
			err := db.Close()
			if err != nil {
				return
			}
		}(db)

		mockErr := errors.New("test error")

		b := &policy{
			db:     db,
			logger: common.NewLogger(),
		}

		accessorIDs := []string{"accessor-1", "accessor-2"}

		Convey("empty accessor ids", func() {
			err := b.DeleteByAccessorIDs([]string{})
			assert.Equal(t, err, nil)
		})

		Convey("delete error", func() {
			mock.ExpectExec("^delete").WillReturnError(mockErr)
			err := b.DeleteByAccessorIDs(accessorIDs)
			assert.Equal(t, err, mockErr)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("Success", func() {
			mock.ExpectExec("^delete").WillReturnResult(sqlmock.NewResult(0, 2))
			err := b.DeleteByAccessorIDs(accessorIDs)
			assert.Equal(t, err, nil)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})
	})
}

func TestDBUpdateAccessorName(t *testing.T) {
	Convey("TestDBUpdateAccessorName", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func(db *sqlx.DB) {
			err := db.Close()
			if err != nil {
				return
			}
		}(db)

		mockErr := errors.New("test error")

		b := &policy{
			db:     db,
			logger: common.NewLogger(),
		}

		accessorID := "accessor-1"
		name := "new-name"

		Convey("update error", func() {
			mock.ExpectExec("^update").WillReturnError(mockErr)
			err := b.UpdateAccessorName(accessorID, name)
			assert.Equal(t, err, mockErr)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("Success", func() {
			mock.ExpectExec("^update").WillReturnResult(sqlmock.NewResult(0, 1))
			err := b.UpdateAccessorName(accessorID, name)
			assert.Equal(t, err, nil)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})
	})
}

func TestDBUpdateResourceName(t *testing.T) {
	Convey("TestDBUpdateResourceName", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func(db *sqlx.DB) {
			err := db.Close()
			if err != nil {
				return
			}
		}(db)

		mockErr := errors.New("test error")
		ctx := context.Background()

		b := &policy{
			db:     db,
			logger: common.NewLogger(),
		}

		resourceID := "resource-1"
		resourceType := "type-1"
		name := "new-name"

		Convey("update error", func() {
			mock.ExpectExec("^update").WillReturnError(mockErr)
			err := b.UpdateResourceName(ctx, resourceID, resourceType, name)
			assert.Equal(t, err, mockErr)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("Success", func() {
			mock.ExpectExec("^update").WillReturnResult(sqlmock.NewResult(0, 1))
			err := b.UpdateResourceName(ctx, resourceID, resourceType, name)
			assert.Equal(t, err, nil)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})
	})
}

func TestDBDeleteByEndTime(t *testing.T) {
	Convey("TestDBDeleteByEndTime", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func(db *sqlx.DB) {
			err := db.Close()
			if err != nil {
				return
			}
		}(db)

		mockErr := errors.New("test error")

		b := &policy{
			db:     db,
			logger: common.NewLogger(),
		}

		curTime := int64(1234567890)

		Convey("delete error", func() {
			mock.ExpectExec("^delete").WillReturnError(mockErr)
			err := b.DeleteByEndTime(curTime)
			assert.Equal(t, err, mockErr)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("Success", func() {
			mock.ExpectExec("^delete").WillReturnResult(sqlmock.NewResult(0, 5))
			err := b.DeleteByEndTime(curTime)
			assert.Equal(t, err, nil)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})
	})
}

func TestDBGetAccessorPolicy(t *testing.T) {
	Convey("TestDBGetAccessorPolicy", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func(db *sqlx.DB) {
			err := db.Close()
			if err != nil {
				return
			}
		}(db)

		mockErr := errors.New("test error")
		ctx := context.Background()

		b := &policy{
			db:     db,
			logger: common.NewLogger(),
		}

		param := interfaces.AccessorPolicyParam{
			AccessorID:   "accessor-1",
			AccessorType: interfaces.AccessorUser,
			Limit:        10,
			Offset:       0,
		}

		Convey("query error", func() {
			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnError(mockErr)
			policies, err := b.GetAccessorPolicy(ctx, param)
			assert.Equal(t, err, mockErr)
			assert.Equal(t, len(policies), 0)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("Success - basic query", func() {
			rules := makeRulesForTest([]interfaces.PolicyOperationItem{{ID: "op1"}}, nil)
			operationJSON, _ := b.rulesInfoToString(rules)

			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnRows(
				sqlmock.NewRows([]string{
					"f_id", "f_resource_id", "f_resource_type", "f_resource_name",
					"f_operation", "f_condition", "f_ancestors", "f_end_time", "f_create_time",
				}).
					AddRow("policy-1", "resource-1", "type-1", "name-1",
						operationJSON, "condition-1", "[]", 1234567890, 1234567890),
			)
			policies, err := b.GetAccessorPolicy(ctx, param)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(policies), 1)
			assert.Equal(t, policies[0].ID, "policy-1")
			assert.Equal(t, policies[0].ResourceID, "resource-1")
			assert.Equal(t, policies[0].ResourceType, "type-1")
			assert.Equal(t, policies[0].ResourceName, "name-1")
			assert.Equal(t, policies[0].Condition, "condition-1")
			assert.Equal(t, policies[0].EndTime, int64(1234567890))
			assert.Equal(t, policies[0].CreateTime, int64(1234567890))
			assert.Equal(t, len(policies[0].Rules.Allow), 1)
			assert.Equal(t, len(policies[0].Rules.Allow[0].Operations), 1)
			assert.Equal(t, policies[0].Rules.Allow[0].Operations[0].ID, "op1")
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("Success - with resource type filter", func() {
			param.ResourceType = "test-type"
			rules := makeRulesForTest([]interfaces.PolicyOperationItem{{ID: "op1"}}, []interfaces.PolicyOperationItem{{ID: "op2"}})
			operationJSON, _ := b.rulesInfoToString(rules)

			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnRows(
				sqlmock.NewRows([]string{
					"f_id", "f_resource_id", "f_resource_type", "f_resource_name",
					"f_operation", "f_condition", "f_ancestors", "f_end_time", "f_create_time",
				}).
					AddRow("policy-1", "resource-1", "test-type", "name-1",
						operationJSON, "condition-1", "[]", 1234567890, 1234567890),
			)
			policies, err := b.GetAccessorPolicy(ctx, param)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(policies), 1)
			assert.Equal(t, policies[0].ResourceType, "test-type")
			assert.Equal(t, len(policies[0].Rules.Allow), 1)
			assert.Equal(t, len(policies[0].Rules.Deny), 1)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("Success - with resource ID filter", func() {
			param.ResourceType = ""
			param.ResourceID = "test-resource-id"
			rules := makeRulesForTest([]interfaces.PolicyOperationItem{{ID: "op1"}}, nil)
			operationJSON, _ := b.rulesInfoToString(rules)

			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnRows(
				sqlmock.NewRows([]string{
					"f_id", "f_resource_id", "f_resource_type", "f_resource_name",
					"f_operation", "f_condition", "f_ancestors", "f_end_time", "f_create_time",
				}).
					AddRow("policy-1", "test-resource-id", "type-1", "name-1",
						operationJSON, "condition-1", "[]", 1234567890, 1234567890),
			)
			policies, err := b.GetAccessorPolicy(ctx, param)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(policies), 1)
			assert.Equal(t, policies[0].ResourceID, "test-resource-id")
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("Success - with both resource type and ID filter", func() {
			param.ResourceType = "test-type"
			param.ResourceID = "test-resource-id"
			rules := makeRulesForTest([]interfaces.PolicyOperationItem{{ID: "op1"}}, nil)
			operationJSON, _ := b.rulesInfoToString(rules)

			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnRows(
				sqlmock.NewRows([]string{
					"f_id", "f_resource_id", "f_resource_type", "f_resource_name",
					"f_operation", "f_condition", "f_ancestors", "f_end_time", "f_create_time",
				}).
					AddRow("policy-1", "test-resource-id", "test-type", "name-1",
						operationJSON, "condition-1", "[]", 1234567890, 1234567890),
			)
			policies, err := b.GetAccessorPolicy(ctx, param)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(policies), 1)
			assert.Equal(t, policies[0].ResourceID, "test-resource-id")
			assert.Equal(t, policies[0].ResourceType, "test-type")
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("Success - empty result", func() {
			param.ResourceType = ""
			param.ResourceID = ""
			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnRows(
				sqlmock.NewRows([]string{
					"f_id", "f_resource_id", "f_resource_type", "f_resource_name",
					"f_operation", "f_condition", "f_ancestors", "f_end_time", "f_create_time",
				}),
			)
			policies, err := b.GetAccessorPolicy(ctx, param)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(policies), 0)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("Success - multiple policies", func() {
			param.ResourceType = ""
			param.ResourceID = ""
			rules1 := makeRulesForTest([]interfaces.PolicyOperationItem{{ID: "op1"}}, nil)
			rules2 := makeRulesForTest(nil, []interfaces.PolicyOperationItem{{ID: "op2"}})
			operationJSON1, _ := b.rulesInfoToString(rules1)
			operationJSON2, _ := b.rulesInfoToString(rules2)

			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnRows(
				sqlmock.NewRows([]string{
					"f_id", "f_resource_id", "f_resource_type", "f_resource_name",
					"f_operation", "f_condition", "f_ancestors", "f_end_time", "f_create_time",
				}).
					AddRow("policy-1", "resource-1", "type-1", "name-1",
						operationJSON1, "condition-1", "[]", 1234567890, 1234567890).
					AddRow("policy-2", "resource-2", "type-2", "name-2",
						operationJSON2, "condition-2", "[]", 1234567891, 1234567891),
			)
			policies, err := b.GetAccessorPolicy(ctx, param)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(policies), 2)
			assert.Equal(t, policies[0].ID, "policy-1")
			assert.Equal(t, policies[1].ID, "policy-2")
			assert.Equal(t, len(policies[0].Rules.Allow), 1)
			assert.Equal(t, len(policies[1].Rules.Deny), 1)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})
	})
}

func TestDBGetResourcePolicies(t *testing.T) {
	Convey("TestDBGetResourcePolicies", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func(db *sqlx.DB) {
			err := db.Close()
			if err != nil {
				return
			}
		}(db)

		mockErr := errors.New("test error")
		ctx := context.Background()

		b := &policy{
			db:     db,
			logger: common.NewLogger(),
		}

		params := interfaces.ResourcePolicyPagination{
			ResourceID:   "test-resource-id",
			ResourceType: "test-resource-type",
			Offset:       0,
			Limit:        10,
		}

		Convey("count query error", func() {
			mock.ExpectQuery("^select count").WillReturnError(mockErr)
			count, policies, err := b.GetResourcePolicies(ctx, params)
			assert.Equal(t, err, mockErr)
			assert.Equal(t, count, 0)
			assert.Equal(t, len(policies), 0)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("data query error", func() {
			mock.ExpectQuery("^select count").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnError(mockErr)
			count, policies, err := b.GetResourcePolicies(ctx, params)
			assert.Equal(t, err, mockErr)
			assert.Equal(t, count, 0)
			assert.Equal(t, len(policies), 0)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("json unmarshal error", func() {
			operationJSON := invalidJSON
			mock.ExpectQuery("^select count").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnRows(
				sqlmock.NewRows([]string{
					"f_id", "f_resource_id", "f_resource_type", "f_resource_name",
					"f_accessor_id", "f_accessor_type", "f_accessor_name", "f_operation",
					"f_condition", "f_end_time", "f_create_time", "f_modify_time",
				}).
					AddRow("policy-1", "resource-1", "type-1", "name-1", "accessor-1",
						interfaces.AccessorUser, "accessor-name-1", operationJSON, "condition-1", 1234567890, 1234567890, 1234567890),
			)
			count, policies, err := b.GetResourcePolicies(ctx, params)
			assert.NotEqual(t, err, nil)
			assert.Equal(t, count, 0)
			assert.Equal(t, len(policies), 0)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("Success", func() {
			rules := makeRulesForTest([]interfaces.PolicyOperationItem{{ID: "op1"}}, []interfaces.PolicyOperationItem{{ID: "op2"}})
			operationJSON, _ := b.rulesInfoToString(rules)

			mock.ExpectQuery("^select count").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnRows(
				sqlmock.NewRows([]string{
					"f_id", "f_resource_id", "f_resource_type", "f_resource_name",
					"f_accessor_id", "f_accessor_type", "f_accessor_name", "f_operation",
					"f_condition", "f_end_time", "f_create_time", "f_modify_time",
				}).
					AddRow("policy-1", "resource-1", "type-1", "name-1", "accessor-1",
						interfaces.AccessorUser, "accessor-name-1", operationJSON, "condition-1", 1234567890, 1234567890, 1234567890),
			)
			count, policies, err := b.GetResourcePolicies(ctx, params)
			assert.Equal(t, err, nil)
			assert.Equal(t, count, 1)
			assert.Equal(t, len(policies), 1)
			assert.Equal(t, policies[0].ID, "policy-1")
			assert.Equal(t, policies[0].ResourceID, "resource-1")
			assert.Equal(t, policies[0].ResourceType, "type-1")
			assert.Equal(t, policies[0].ResourceName, "name-1")
			assert.Equal(t, len(policies[0].Rules.Allow), 1)
			assert.Equal(t, len(policies[0].Rules.Deny), 1)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})
	})
}

//nolint:lll,dupl
func TestPolicyRulesStrToInfo(t *testing.T) {
	Convey("TestPolicyRulesStrToInfo", t, func() {
		d := &policy{
			logger: common.NewLogger(),
		}

		Convey("json unmarshal error", func() {
			operationStr := invalidJSON
			_, err := d.rulesStrToInfo(operationStr)
			assert.NotEqual(t, err, nil)
		})

		Convey("allow and deny are empty", func() {
			operationStr := `{"allow":[],"deny":[]}`
			rules, err := d.rulesStrToInfo(operationStr)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(rules.Allow), 0)
			assert.Equal(t, len(rules.Deny), 0)
		})

		Convey("without obligations", func() {
			operationStr := `{"allow":[{"condition":{},"operations":[{"id":"op1","obligations":[]}]}],"deny":[{"condition":{},"operations":[{"id":"op2","obligations":[]}]}]}`
			rules, err := d.rulesStrToInfo(operationStr)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(rules.Allow), 1)
			assert.Equal(t, len(rules.Allow[0].Operations), 1)
			assert.Equal(t, rules.Allow[0].Operations[0].ID, "op1")
			assert.Equal(t, len(rules.Allow[0].Operations[0].Obligations), 0)
			assert.Equal(t, len(rules.Deny), 1)
			assert.Equal(t, len(rules.Deny[0].Operations), 1)
			assert.Equal(t, rules.Deny[0].Operations[0].ID, "op2")
		})

		Convey("with obligations", func() {
			operationStr := `{"allow":[{"condition":{},"operations":[{"id":"op1","obligations":[{"type_id":"type1","id":"obl1","value":"value1"}]}]}],"deny":[{"condition":{},"operations":[{"id":"op2","obligations":[]}]}]}`
			rules, err := d.rulesStrToInfo(operationStr)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(rules.Allow), 1)
			assert.Equal(t, rules.Allow[0].Operations[0].ID, "op1")
			assert.Equal(t, len(rules.Allow[0].Operations[0].Obligations), 1)
			assert.Equal(t, rules.Allow[0].Operations[0].Obligations[0].TypeID, "type1")
			assert.Equal(t, rules.Allow[0].Operations[0].Obligations[0].ID, "obl1")
			assert.Equal(t, rules.Allow[0].Operations[0].Obligations[0].Value, "value1")
			assert.Equal(t, len(rules.Deny), 1)
			assert.Equal(t, rules.Deny[0].Operations[0].ID, "op2")
		})

		Convey("with multiple obligations", func() {
			operationStr := `{"allow":[{"condition":{},"operations":[{"id":"op1","obligations":[{"type_id":"type1","id":"obl1","value":"value1"},{"type_id":"type2","id":"obl2","value":123}]}]}],"deny":[]}`
			rules, err := d.rulesStrToInfo(operationStr)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(rules.Allow), 1)
			assert.Equal(t, len(rules.Allow[0].Operations[0].Obligations), 2)
			assert.Equal(t, rules.Allow[0].Operations[0].Obligations[0].TypeID, "type1")
			assert.Equal(t, rules.Allow[0].Operations[0].Obligations[0].ID, "obl1")
			assert.Equal(t, rules.Allow[0].Operations[0].Obligations[0].Value, "value1")
			assert.Equal(t, rules.Allow[0].Operations[0].Obligations[1].TypeID, "type2")
			assert.Equal(t, rules.Allow[0].Operations[0].Obligations[1].ID, "obl2")
			assert.Equal(t, rules.Allow[0].Operations[0].Obligations[1].Value, float64(123))
		})

		Convey("multiple allow items with obligations", func() {
			operationStr := `{"allow":[{"condition":{},"operations":[{"id":"op1","obligations":[{"type_id":"type1","id":"obl1","value":"value1"}]}]},{"condition":{},"operations":[{"id":"op3","obligations":[{"type_id":"type3","id":"obl3","value":"value3"}]}]}],"deny":[{"condition":{},"operations":[{"id":"op2","obligations":[]}]}]}`
			rules, err := d.rulesStrToInfo(operationStr)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(rules.Allow), 2)
			assert.Equal(t, rules.Allow[0].Operations[0].ID, "op1")
			assert.Equal(t, len(rules.Allow[0].Operations[0].Obligations), 1)
			assert.Equal(t, rules.Allow[0].Operations[0].Obligations[0].TypeID, "type1")
			assert.Equal(t, rules.Allow[1].Operations[0].ID, "op3")
			assert.Equal(t, len(rules.Allow[1].Operations[0].Obligations), 1)
			assert.Equal(t, rules.Allow[1].Operations[0].Obligations[0].TypeID, "type3")
			assert.Equal(t, len(rules.Deny), 1)
		})
	})
}

func TestPolicyGetObligations(t *testing.T) {
	Convey("TestPolicyGetObligations", t, func() {
		d := &policy{
			logger: common.NewLogger(),
		}

		Convey("empty obligations", func() {
			obligationsJson := []any{}
			result := d.getObligations(obligationsJson)
			assert.Equal(t, len(result), 0)
		})

		Convey("single obligation with string value", func() {
			obligationsJson := []any{
				map[string]any{
					"type_id": "type1",
					"id":      "obl1",
					"value":   "value1",
				},
			}
			result := d.getObligations(obligationsJson)
			assert.Equal(t, len(result), 1)
			assert.Equal(t, result[0].TypeID, "type1")
			assert.Equal(t, result[0].ID, "obl1")
			assert.Equal(t, result[0].Value, "value1")
		})

		Convey("single obligation with int value", func() {
			obligationsJson := []any{
				map[string]any{
					"type_id": "type1",
					"id":      "obl1",
					"value":   123,
				},
			}
			result := d.getObligations(obligationsJson)
			assert.Equal(t, len(result), 1)
			assert.Equal(t, result[0].TypeID, "type1")
			assert.Equal(t, result[0].ID, "obl1")
			assert.Equal(t, result[0].Value, 123)
		})

		Convey("single obligation with map value", func() {
			obligationsJson := []any{
				map[string]any{
					"type_id": "type1",
					"id":      "obl1",
					"value": map[string]any{
						"key1": "value1",
						"key2": 123,
					},
				},
			}
			result := d.getObligations(obligationsJson)
			assert.Equal(t, len(result), 1)
			assert.Equal(t, result[0].TypeID, "type1")
			assert.Equal(t, result[0].ID, "obl1")
			valueMap := result[0].Value.(map[string]any)
			assert.Equal(t, valueMap["key1"], "value1")
			assert.Equal(t, valueMap["key2"], 123)
		})

		Convey("multiple obligations", func() {
			obligationsJson := []any{
				map[string]any{
					"type_id": "type1",
					"id":      "obl1",
					"value":   "value1",
				},
				map[string]any{
					"type_id": "type2",
					"id":      "obl2",
					"value":   456,
				},
				map[string]any{
					"type_id": "type3",
					"id":      "obl3",
					"value": map[string]any{
						"nested": "data",
					},
				},
			}
			result := d.getObligations(obligationsJson)
			assert.Equal(t, len(result), 3)
			assert.Equal(t, result[0].TypeID, "type1")
			assert.Equal(t, result[0].ID, "obl1")
			assert.Equal(t, result[0].Value, "value1")
			assert.Equal(t, result[1].TypeID, "type2")
			assert.Equal(t, result[1].ID, "obl2")
			assert.Equal(t, result[1].Value, 456)
			assert.Equal(t, result[2].TypeID, "type3")
			assert.Equal(t, result[2].ID, "obl3")
			valueMap := result[2].Value.(map[string]any)
			assert.Equal(t, valueMap["nested"], "data")
		})

		Convey("obligation with nil value", func() {
			obligationsJson := []any{
				map[string]any{
					"type_id": "type1",
					"id":      "obl1",
					"value":   nil,
				},
			}
			result := d.getObligations(obligationsJson)
			assert.Equal(t, len(result), 1)
			assert.Equal(t, result[0].TypeID, "type1")
			assert.Equal(t, result[0].ID, "obl1")
			assert.Equal(t, result[0].Value, nil)
		})
	})
}
