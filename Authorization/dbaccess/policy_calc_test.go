//nolint:lll
package dbaccess

import (
	"context"
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
	policyCalcTestResourceTypeID = "type-1"
)

func TestPolicyCalcGetPoliciesByResourceTypeAndAccessToken(t *testing.T) {
	Convey("TestPolicyCalcGetPoliciesByResourceTypeAndAccessToken", t, func() {
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
		b := &policy{db: db, logger: common.NewLogger()}
		d := &policyCalc{
			db:     db,
			logger: common.NewLogger(),
		}

		resourceTypeID := policyCalcTestResourceTypeID
		accessToken := []string{"token1", "token2"}

		Convey("query error", func() {
			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnError(mockErr)
			policies, err := d.GetPoliciesByResourceTypeAndAccessToken(ctx, resourceTypeID, accessToken)
			assert.Equal(t, err, mockErr)
			assert.Equal(t, len(policies), 0)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("scan error", func() {
			rows := sqlmock.NewRows([]string{
				"f_id", "f_resource_id", "f_resource_type",
				"f_resource_name", "f_accessor_id", "f_accessor_type", "f_accessor_name", "f_operation",
				"f_condition", "f_end_time", "f_create_time", "f_modify_time",
			}).
				AddRow("policy-1", "resource-1", policyCalcTestResourceTypeID, "name-1", "accessor-1",
					interfaces.AccessorUser, "accessor-name-1", "invalid-json", "condition-1", int64(-1), 1234567890, 1234567890)
			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnRows(rows)
			policies, err := d.GetPoliciesByResourceTypeAndAccessToken(ctx, resourceTypeID, accessToken)
			assert.NotEqual(t, err, nil)
			assert.Equal(t, len(policies), 0)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("json unmarshal error", func() {
			operationJSON := invalidJSON
			rows := sqlmock.NewRows([]string{
				"f_id", "f_resource_id", "f_resource_type",
				"f_resource_name", "f_accessor_id", "f_accessor_type", "f_accessor_name",
				"f_operation", "f_condition", "f_end_time", "f_create_time", "f_modify_time",
			}).
				AddRow("policy-1", "resource-1", policyCalcTestResourceTypeID, "name-1", "accessor-1",
					interfaces.AccessorUser, "accessor-name-1", operationJSON, "condition-1", int64(-1), 1234567890, 1234567890)
			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnRows(rows)
			policies, err := d.GetPoliciesByResourceTypeAndAccessToken(ctx, resourceTypeID, accessToken)
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

			rows := sqlmock.NewRows([]string{
				"f_id", "f_resource_id",
				"f_resource_type", "f_resource_name", "f_accessor_id", "f_accessor_type", "f_accessor_name",
				"f_operation", "f_condition", "f_end_time", "f_create_time", "f_modify_time",
			}).
				AddRow("policy-1", "resource-1", policyCalcTestResourceTypeID, "name-1", "accessor-1",
					interfaces.AccessorUser, "accessor-name-1", operationJSON, "condition-1", int64(-1), 1234567890, 1234567890)
			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnRows(rows)
			policies, err := d.GetPoliciesByResourceTypeAndAccessToken(ctx, resourceTypeID, accessToken)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(policies), 1)
			assert.Equal(t, policies[0].ID, "policy-1")
			assert.Equal(t, policies[0].ResourceID, "resource-1")
			assert.Equal(t, policies[0].ResourceType, policyCalcTestResourceTypeID)
			assert.Equal(t, policies[0].ResourceName, "name-1")
			assert.Equal(t, policies[0].AccessorID, "accessor-1")
			assert.Equal(t, policies[0].AccessorType, interfaces.AccessorUser)
			assert.Equal(t, policies[0].AccessorName, "accessor-name-1")
			assert.Equal(t, policies[0].Condition, "condition-1")
			assert.Equal(t, policies[0].CreateTime, int64(1234567890))
			assert.Equal(t, policies[0].ModifyTime, int64(1234567890))
			assert.Equal(t, len(policies[0].Rules.Allow), 1)
			assert.Equal(t, len(policies[0].Rules.Deny), 1)
		})

		//nolint:dupl
		Convey("filter expired policies", func() {
			rules := makeRulesForTest([]interfaces.PolicyOperationItem{{ID: "op1", Name: "operation1"}}, nil)
			operationJSON, _ := b.rulesInfoToString(rules)
			curTime := common.GetCurrentMicrosecondTimestamp()
			expiredTime := curTime - 1000000 // 过期时间（1秒前）
			futureTime := curTime + 1000000  // 未来时间（1秒后）

			rows := sqlmock.NewRows([]string{
				"f_id", "f_resource_id",
				"f_resource_type", "f_resource_name", "f_accessor_id", "f_accessor_type", "f_accessor_name",
				"f_operation", "f_condition", "f_end_time", "f_create_time", "f_modify_time",
			}).
				AddRow("policy-1", "resource-1", policyCalcTestResourceTypeID, "name-1", "accessor-1",
					interfaces.AccessorUser, "accessor-name-1", operationJSON, "condition-1", int64(-1), 1234567890, 1234567890). // 永不过期
				AddRow("policy-2", "resource-1", policyCalcTestResourceTypeID, "name-2", "accessor-2",
					interfaces.AccessorUser, "accessor-name-2", operationJSON, "condition-2", expiredTime, 1234567890, 1234567890). // 已过期
				AddRow("policy-3", "resource-1", policyCalcTestResourceTypeID, "name-3", "accessor-3",
					interfaces.AccessorUser, "accessor-name-3", operationJSON, "condition-3", futureTime, 1234567890, 1234567890) // 未过期
			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnRows(rows)
			policies, err := d.GetPoliciesByResourceTypeAndAccessToken(ctx, resourceTypeID, accessToken)
			assert.Equal(t, err, nil)
			// 应该只返回2条：policy-1（永不过期）和 policy-3（未过期），policy-2（已过期）被过滤
			assert.Equal(t, len(policies), 2)
			assert.Equal(t, policies[0].ID, "policy-1")
			assert.Equal(t, policies[1].ID, "policy-3")
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})
	})
}

func TestPolicyCalcGetPoliciesByResourcesAndAccessToken(t *testing.T) {
	Convey("TestPolicyCalcGetPoliciesByResourcesAndAccessToken", t, func() {
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
		b := &policy{db: db, logger: common.NewLogger()}
		d := &policyCalc{
			db:     db,
			logger: common.NewLogger(),
		}

		resourceInfo := []interfaces.ResourceInfo{
			{
				ID:   "resource-1",
				Type: policyCalcTestResourceTypeID,
				Name: "resource-name-1",
			},
			{
				ID:   "resource-2",
				Type: policyCalcTestResourceTypeID,
				Name: "resource-name-2",
			},
		}
		accessToken := []string{"token1", "token2"}

		Convey("empty resource info", func() {
			policies, err := d.GetPoliciesByResourcesAndAccessToken(ctx, []interfaces.ResourceInfo{}, accessToken)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(policies), 0)
		})

		Convey("query error", func() {
			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnError(mockErr)
			policies, err := d.GetPoliciesByResourcesAndAccessToken(ctx, resourceInfo, accessToken)
			assert.Equal(t, err, mockErr)
			assert.Equal(t, len(policies), 0)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("scan error", func() {
			rows := sqlmock.NewRows([]string{"f_id", "f_resource_id", "f_resource_type", "f_resource_name", "f_accessor_id", "f_accessor_type", "f_accessor_name", "f_operation", "f_condition", "f_end_time", "f_create_time", "f_modify_time"}).
				AddRow("policy-1", "resource-1", policyCalcTestResourceTypeID, "name-1", "accessor-1", interfaces.AccessorUser, "accessor-name-1", "invalid-json", "condition-1", int64(-1), 1234567890, 1234567890)
			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnRows(rows)
			policies, err := d.GetPoliciesByResourcesAndAccessToken(ctx, resourceInfo, accessToken)
			assert.NotEqual(t, err, nil)
			assert.Equal(t, len(policies), 0)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("json unmarshal error", func() {
			operationJSON := invalidJSON
			rows := sqlmock.NewRows([]string{"f_id", "f_resource_id", "f_resource_type", "f_resource_name", "f_accessor_id", "f_accessor_type", "f_accessor_name", "f_operation", "f_condition", "f_end_time", "f_create_time", "f_modify_time"}).
				AddRow("policy-1", "resource-1", policyCalcTestResourceTypeID, "name-1", "accessor-1", interfaces.AccessorUser, "accessor-name-1", operationJSON, "condition-1", int64(-1), 1234567890, 1234567890)
			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnRows(rows)
			policies, err := d.GetPoliciesByResourcesAndAccessToken(ctx, resourceInfo, accessToken)
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

			rows := sqlmock.NewRows([]string{"f_id", "f_resource_id", "f_resource_type", "f_resource_name", "f_accessor_id", "f_accessor_type", "f_accessor_name", "f_operation", "f_condition", "f_end_time", "f_create_time", "f_modify_time"}).
				AddRow("policy-1", "resource-1", policyCalcTestResourceTypeID, "name-1", "accessor-1", interfaces.AccessorUser, "accessor-name-1", operationJSON, "condition-1", int64(-1), 1234567890, 1234567890).
				AddRow("policy-2", "resource-2", policyCalcTestResourceTypeID, "name-2", "accessor-2", interfaces.AccessorUser, "accessor-name-2", operationJSON, "condition-2", int64(-1), 1234567890, 1234567890)
			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnRows(rows)
			policies, err := d.GetPoliciesByResourcesAndAccessToken(ctx, resourceInfo, accessToken)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(policies), 2)
			assert.Equal(t, policies[0].ID, "policy-1")
			assert.Equal(t, policies[0].ResourceID, "resource-1")
			assert.Equal(t, policies[1].ID, "policy-2")
			assert.Equal(t, policies[1].ResourceID, "resource-2")
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		//nolint:dupl
		Convey("filter expired policies", func() {
			rules := makeRulesForTest([]interfaces.PolicyOperationItem{{ID: "op1", Name: "operation1"}}, nil)
			operationJSON, _ := b.rulesInfoToString(rules)
			curTime := common.GetCurrentMicrosecondTimestamp()
			expiredTime := curTime - 1000000 // 过期时间（1秒前）
			futureTime := curTime + 1000000  // 未来时间（1秒后）

			rows := sqlmock.NewRows([]string{"f_id", "f_resource_id", "f_resource_type", "f_resource_name", "f_accessor_id", "f_accessor_type", "f_accessor_name", "f_operation", "f_condition", "f_end_time", "f_create_time", "f_modify_time"}).
				AddRow("policy-1", "resource-1", policyCalcTestResourceTypeID, "name-1", "accessor-1", interfaces.AccessorUser, "accessor-name-1", operationJSON, "condition-1", int64(-1), 1234567890, 1234567890).   // 永不过期
				AddRow("policy-2", "resource-2", policyCalcTestResourceTypeID, "name-2", "accessor-2", interfaces.AccessorUser, "accessor-name-2", operationJSON, "condition-2", expiredTime, 1234567890, 1234567890). // 已过期
				AddRow("policy-3", "resource-1", policyCalcTestResourceTypeID, "name-3", "accessor-3", interfaces.AccessorUser, "accessor-name-3", operationJSON, "condition-3", futureTime, 1234567890, 1234567890)   // 未过期
			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnRows(rows)
			policies, err := d.GetPoliciesByResourcesAndAccessToken(ctx, resourceInfo, accessToken)
			assert.Equal(t, err, nil)
			// 应该只返回2条：policy-1（永不过期）和 policy-3（未过期），policy-2（已过期）被过滤
			assert.Equal(t, len(policies), 2)
			assert.Equal(t, policies[0].ID, "policy-1")
			assert.Equal(t, policies[1].ID, "policy-3")
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})
	})
}

func TestPolicyCalcGetPoliciesByResourceTypes(t *testing.T) {
	Convey("TestPolicyCalcGetPoliciesByResourceTypes", t, func() {
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
		b := &policy{db: db, logger: common.NewLogger()}
		d := &policyCalc{
			db:     db,
			logger: common.NewLogger(),
		}

		resourceTypes := []string{policyCalcTestResourceTypeID, "type-2"}
		accessToken := []string{"token1", "token2"}

		Convey("query error", func() {
			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnError(mockErr)
			policies, err := d.GetPoliciesByResourceTypes(ctx, resourceTypes, accessToken)
			assert.Equal(t, err, mockErr)
			assert.Equal(t, len(policies), 0)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("scan error", func() {
			rows := sqlmock.NewRows([]string{"f_id", "f_resource_id", "f_resource_type", "f_resource_name", "f_accessor_id", "f_accessor_type", "f_accessor_name", "f_operation", "f_condition", "f_end_time", "f_create_time", "f_modify_time"}).
				AddRow("policy-1", "resource-1", policyCalcTestResourceTypeID, "name-1", "accessor-1", interfaces.AccessorUser, "accessor-name-1", "invalid-json", "condition-1", int64(-1), 1234567890, 1234567890)
			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnRows(rows)
			policies, err := d.GetPoliciesByResourceTypes(ctx, resourceTypes, accessToken)
			assert.NotEqual(t, err, nil)
			assert.Equal(t, len(policies), 0)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("json unmarshal error", func() {
			operationJSON := invalidJSON
			rows := sqlmock.NewRows([]string{"f_id", "f_resource_id", "f_resource_type", "f_resource_name", "f_accessor_id", "f_accessor_type", "f_accessor_name", "f_operation", "f_condition", "f_end_time", "f_create_time", "f_modify_time"}).
				AddRow("policy-1", "resource-1", policyCalcTestResourceTypeID, "name-1", "accessor-1", interfaces.AccessorUser, "accessor-name-1", operationJSON, "condition-1", int64(-1), 1234567890, 1234567890)
			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnRows(rows)
			policies, err := d.GetPoliciesByResourceTypes(ctx, resourceTypes, accessToken)
			assert.NotEqual(t, err, nil)
			assert.Equal(t, len(policies), 0)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("Success", func() {
			rules := makeRulesForTest([]interfaces.PolicyOperationItem{{ID: "op1", Name: "operation1"}}, []interfaces.PolicyOperationItem{{ID: "op2", Name: "operation2"}})
			operationJSON, _ := b.rulesInfoToString(rules)

			rows := sqlmock.NewRows([]string{"f_id", "f_resource_id", "f_resource_type", "f_resource_name", "f_accessor_id", "f_accessor_type", "f_accessor_name", "f_operation", "f_condition", "f_end_time", "f_create_time", "f_modify_time"}).
				AddRow("policy-1", "*", policyCalcTestResourceTypeID, "name-1", "accessor-1", interfaces.AccessorUser, "accessor-name-1", operationJSON, "condition-1", int64(-1), 1234567890, 1234567890).
				AddRow("policy-2", "*", "type-2", "name-2", "accessor-2", interfaces.AccessorUser, "accessor-name-2", operationJSON, "condition-2", int64(-1), 1234567890, 1234567890)
			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnRows(rows)
			policies, err := d.GetPoliciesByResourceTypes(ctx, resourceTypes, accessToken)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(policies), 2)
			assert.Equal(t, policies[0].ID, "policy-1")
			assert.Equal(t, policies[0].ResourceID, "*")
			assert.Equal(t, policies[0].ResourceType, policyCalcTestResourceTypeID)
			assert.Equal(t, policies[1].ID, "policy-2")
			assert.Equal(t, policies[1].ResourceID, "*")
			assert.Equal(t, policies[1].ResourceType, "type-2")
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("filter expired policies", func() {
			rules := makeRulesForTest([]interfaces.PolicyOperationItem{{ID: "op1", Name: "operation1"}}, nil)
			operationJSON, _ := b.rulesInfoToString(rules)
			curTime := common.GetCurrentMicrosecondTimestamp()
			expiredTime := curTime - 1000000 // 过期时间（1秒前）
			futureTime := curTime + 1000000  // 未来时间（1秒后）

			rows := sqlmock.NewRows([]string{"f_id", "f_resource_id", "f_resource_type", "f_resource_name", "f_accessor_id", "f_accessor_type", "f_accessor_name", "f_operation", "f_condition", "f_end_time", "f_create_time", "f_modify_time"}).
				AddRow("policy-1", "*", policyCalcTestResourceTypeID, "name-1", "accessor-1", interfaces.AccessorUser, "accessor-name-1", operationJSON, "condition-1", int64(-1), 1234567890, 1234567890).   // 永不过期
				AddRow("policy-2", "*", policyCalcTestResourceTypeID, "name-2", "accessor-2", interfaces.AccessorUser, "accessor-name-2", operationJSON, "condition-2", expiredTime, 1234567890, 1234567890). // 已过期
				AddRow("policy-3", "*", "type-2", "name-3", "accessor-3", interfaces.AccessorUser, "accessor-name-3", operationJSON, "condition-3", futureTime, 1234567890, 1234567890)                       // 未过期
			mock.ExpectQuery("^select.*f_id.*f_resource_id").WillReturnRows(rows)
			policies, err := d.GetPoliciesByResourceTypes(ctx, resourceTypes, accessToken)
			assert.Equal(t, err, nil)
			// 应该只返回2条：policy-1（永不过期）和 policy-3（未过期），policy-2（已过期）被过滤
			assert.Equal(t, len(policies), 2)
			assert.Equal(t, policies[0].ID, "policy-1")
			assert.Equal(t, policies[1].ID, "policy-3")
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})
	})
}

func TestGetFindInSetSQL(t *testing.T) {
	Convey("TestGetFindInSetSQL", t, func() {
		Convey("empty slice", func() {
			setStr, args := getFindInSetSQL([]string{})
			assert.Equal(t, setStr, "")
			assert.Equal(t, len(args), 0)
		})

		Convey("single item", func() {
			setStr, args := getFindInSetSQL([]string{"item1"})
			assert.Equal(t, setStr, "?")
			assert.Equal(t, len(args), 1)
			assert.Equal(t, args[0], "item1")
		})

		Convey("multiple items", func() {
			setStr, args := getFindInSetSQL([]string{"item1", "item2", "item3"})
			assert.Equal(t, setStr, "?,?,?")
			assert.Equal(t, len(args), 3)
			assert.Equal(t, args[0], "item1")
			assert.Equal(t, args[1], "item2")
			assert.Equal(t, args[2], "item3")
		})
	})
}

func TestNewPolicyCalc(t *testing.T) {
	Convey("TestNewPolicyCalc", t, func() {
		Convey("singleton pattern", func() {
			instance1 := NewPolicyCalc()
			instance2 := NewPolicyCalc()
			assert.Equal(t, instance1, instance2)
		})
	})
}

//nolint:dupl
func TestOperationStrToInfo(t *testing.T) {
	Convey("TestOperationStrToInfo", t, func() {
		d := &policyCalc{
			logger: common.NewLogger(),
		}

		Convey("json unmarshal error", func() {
			operationStr := invalidJSON
			_, err := d.rulesStrToInfo(operationStr)
			assert.NotEqual(t, err, nil)
		})

		Convey("allow and deny are empty", func() {
			operationStr := `{"allow":[],"deny":[]}`
			operation, err := d.rulesStrToInfo(operationStr)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(operation.Allow), 0)
			assert.Equal(t, len(operation.Deny), 0)
		})

		Convey("without obligations", func() {
			operationStr := `{"allow":[{"condition":{},"operations":[{"id":"op1","obligations":[]}]}],"deny":[{"condition":{},"operations":[{"id":"op2","obligations":[]}]}]}`
			operation, err := d.rulesStrToInfo(operationStr)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(operation.Allow), 1)
			assert.Equal(t, operation.Allow[0].Operations[0].ID, "op1")
			assert.Equal(t, len(operation.Allow[0].Operations[0].Obligations), 0)
			assert.Equal(t, len(operation.Deny), 1)
			assert.Equal(t, operation.Deny[0].Operations[0].ID, "op2")
		})

		Convey("with obligations", func() {
			operationStr := `{"allow":[{"condition":{},"operations":[{"id":"op1","obligations":[{"type_id":"type1","id":"obl1","value":"value1"}]}]}],"deny":[{"condition":{},"operations":[{"id":"op2","obligations":[]}]}]}`
			operation, err := d.rulesStrToInfo(operationStr)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(operation.Allow), 1)
			assert.Equal(t, operation.Allow[0].Operations[0].ID, "op1")
			assert.Equal(t, len(operation.Allow[0].Operations[0].Obligations), 1)
			assert.Equal(t, operation.Allow[0].Operations[0].Obligations[0].TypeID, "type1")
			assert.Equal(t, operation.Allow[0].Operations[0].Obligations[0].ID, "obl1")
			assert.Equal(t, operation.Allow[0].Operations[0].Obligations[0].Value, "value1")
			assert.Equal(t, len(operation.Deny), 1)
			assert.Equal(t, operation.Deny[0].Operations[0].ID, "op2")
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
			operation, err := d.rulesStrToInfo(operationStr)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(operation.Allow), 2)
			assert.Equal(t, operation.Allow[0].Operations[0].ID, "op1")
			assert.Equal(t, len(operation.Allow[0].Operations[0].Obligations), 1)
			assert.Equal(t, operation.Allow[0].Operations[0].Obligations[0].TypeID, "type1")
			assert.Equal(t, operation.Allow[1].Operations[0].ID, "op3")
			assert.Equal(t, len(operation.Allow[1].Operations[0].Obligations), 1)
			assert.Equal(t, operation.Allow[1].Operations[0].Obligations[0].TypeID, "type3")
			assert.Equal(t, len(operation.Deny), 1)
		})
	})
}

func TestGetObligations(t *testing.T) {
	Convey("TestGetObligations", t, func() {
		d := &policyCalc{
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
