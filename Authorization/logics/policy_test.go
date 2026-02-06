//nolint:staticcheck
package logics

import (
	"context"
	"errors"
	"testing"

	"github.com/kweaver-ai/proton-rds-sdk-go/sqlx"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"Authorization/common"
	"Authorization/interfaces"
	"Authorization/interfaces/mock"
)

func newPolicy(db interfaces.DBPolicy, userMgnt interfaces.DrivenUserMgnt, role interfaces.LogicsRole,
	resourceType interfaces.LogicsResourceType, policyCalc interfaces.LogicsPolicyCalc,
	resourceTypeHierarchy interfaces.LogicsResourceTypeHierarchy,
) *policy {
	return &policy{
		db:                    db,
		userMgmt:              userMgnt,
		logger:                common.NewLogger(),
		role:                  role,
		resourceType:          resourceType,
		resourceTypeHierarchy: resourceTypeHierarchy,
		policyCalc:            policyCalc,
		i18n: common.NewI18n(common.I18nMap{
			i18nAccessorRoleNotFound: {
				simplifiedChinese:  "角色不存在",
				traditionalChinese: "角色不存在",
				americanEnglish:    "The Role does not exist.",
			},
		}),
	}
}

func TestPolicyGetPagination(t *testing.T) {
	Convey("获取策略分页接口, 请求接口异常情况", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tmpDB := mock.NewMockDBPolicy(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		policyCalc := mock.NewMockLogicsPolicyCalc(ctrl)
		policy := newPolicy(tmpDB, userMgnt, role, resourceType, policyCalc, resourceTypeHierarchy)

		testErr := errors.New("some error")
		ctx := context.Background()
		visitor := interfaces.Visitor{
			ID:       accessorID,
			Type:     interfaces.RealName,
			Language: "zh-cn",
		}
		params := interfaces.PolicyPagination{
			ResourceID:   resourceID,
			ResourceType: resourceTypeDoc,
			Offset:       1,
			Limit:        10,
		}
		roleTypes := []interfaces.SystemRoleType{}
		calcIncludeParams := []interfaces.PolicCalcyIncludeType{}
		Convey("获取用户系统角色出错", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).Return(roleTypes, testErr)
			_, _, err := policy.GetPagination(ctx, &visitor, params)
			assert.Equal(t, err, testErr)
		})

		Convey("非管理员 检查权限接口出错", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).Return(roleTypes, nil)
			policyCalc.EXPECT().Check(gomock.Any(), gomock.Any(), gomock.Any(), []string{authorizeOperation}, calcIncludeParams).Return(interfaces.CheckResult{}, testErr)
			_, _, err := policy.GetPagination(ctx, &visitor, params)
			assert.Equal(t, err, testErr)
		})

		Convey("其他管理员 检查权限接口正常，没有权限", func() {
			roleTypesTmp := []interfaces.SystemRoleType{interfaces.OrganizationAudit}
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).Return(roleTypesTmp, nil)
			policyCalc.EXPECT().Check(gomock.Any(), gomock.Any(), gomock.Any(), []string{authorizeOperation}, calcIncludeParams).Return(interfaces.CheckResult{Result: false}, nil)
			_, _, err := policy.GetPagination(ctx, &visitor, params)
			assert.NotEqual(t, err, nil)
		})

		Convey("非管理员 检查权限接口正常，没有权限", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).Return(roleTypes, nil)
			policyCalc.EXPECT().Check(gomock.Any(), gomock.Any(), gomock.Any(), []string{authorizeOperation}, calcIncludeParams).Return(interfaces.CheckResult{Result: false}, nil)
			_, _, err := policy.GetPagination(ctx, &visitor, params)
			assert.NotEqual(t, err, nil)
		})

		Convey("超级管理员调用，获取策略接口出错", func() {
			roleTypes = []interfaces.SystemRoleType{interfaces.SuperAdmin}
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).Return(roleTypes, nil)
			tmpDB.EXPECT().GetPagination(gomock.Any(), gomock.Any()).Return(0, nil, testErr)
			_, _, err := policy.GetPagination(ctx, &visitor, params)
			assert.Equal(t, err, testErr)
		})

		Convey("超级管理员调用，获取策略接口正常，结果无策略", func() {
			roleTypes = []interfaces.SystemRoleType{interfaces.SuperAdmin}
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).Return(roleTypes, nil)
			tmpDB.EXPECT().GetPagination(gomock.Any(), gomock.Any()).Return(0, nil, nil)
			_, policies, err := policy.GetPagination(ctx, &visitor, params)
			assert.Equal(t, len(policies), 0)
			assert.Equal(t, err, nil)
		})

		Convey("系统管理员调用，获取策略接口正常，结果无策略", func() {
			roleTypes = []interfaces.SystemRoleType{interfaces.SystemAdmin}
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).Return(roleTypes, nil)
			tmpDB.EXPECT().GetPagination(gomock.Any(), gomock.Any()).Return(0, nil, nil)
			_, policies, err := policy.GetPagination(ctx, &visitor, params)
			assert.Equal(t, len(policies), 0)
			assert.Equal(t, err, nil)
		})

		Convey("安全管理员调用，获取策略接口正常，结果无策略", func() {
			roleTypes = []interfaces.SystemRoleType{interfaces.SecurityAdmin}
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).Return(roleTypes, nil)
			tmpDB.EXPECT().GetPagination(gomock.Any(), gomock.Any()).Return(0, nil, nil)
			_, policies, err := policy.GetPagination(ctx, &visitor, params)
			assert.Equal(t, len(policies), 0)
			assert.Equal(t, err, nil)
		})

		policies := []interfaces.PolicyInfo{
			{
				ResourceType: resourceTypeDoc,
				ResourceID:   resourceID,
				AccessorID:   accessorID,
				AccessorType: interfaces.AccessorDepartment,
				Operation: interfaces.PolicyOperation{
					Allow: []interfaces.PolicyOperationItem{
						{
							ID:   "1",
							Name: "1",
						},
					},
				},
			},
		}

		Convey("获取策略接口正常，结果有策略, 获取资源类型出错", func() {
			resourceTypeInfoMap := map[string]interfaces.ResourceType{
				resourceTypeDoc: {
					ID:   resourceTypeDoc,
					Name: resourceTypeDoc,
				},
			}
			roleTypes = []interfaces.SystemRoleType{interfaces.SuperAdmin}
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).Return(roleTypes, nil)
			tmpDB.EXPECT().GetPagination(gomock.Any(), gomock.Any()).Return(1, policies, nil)
			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), gomock.Any()).Return(resourceTypeInfoMap, testErr)
			_, _, err := policy.GetPagination(ctx, &visitor, params)
			assert.Equal(t, err, testErr)
		})

		Convey("获取策略接口正常，结果有策略, 获取部门父部门信息出错", func() {
			resourceTypeInfoMap := map[string]interfaces.ResourceType{
				resourceTypeDoc: {
					ID:   resourceTypeDoc,
					Name: resourceTypeDoc,
				},
			}
			roleTypes = []interfaces.SystemRoleType{interfaces.SuperAdmin}
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).Return(roleTypes, nil)
			tmpDB.EXPECT().GetPagination(gomock.Any(), gomock.Any()).Return(1, policies, nil)
			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), gomock.Any()).Return(resourceTypeInfoMap, nil)
			userMgnt.EXPECT().GetParentDepartmentsByDepartmentID(gomock.Any(), gomock.Any()).Return(nil, testErr)
			_, _, err := policy.GetPagination(ctx, &visitor, params)
			assert.Equal(t, err, testErr)
		})

		Convey("获取策略接口正常，结果有策略, 获取用户父部门信息出错", func() {
			resourceTypeInfoMap := map[string]interfaces.ResourceType{
				resourceTypeDoc: {
					ID:   resourceTypeDoc,
					Name: resourceTypeDoc,
				},
			}
			policies := []interfaces.PolicyInfo{
				{
					ResourceType: resourceTypeDoc,
					ResourceID:   resourceID,
					AccessorID:   accessorID,
					AccessorType: interfaces.AccessorUser,
					Operation: interfaces.PolicyOperation{
						Allow: []interfaces.PolicyOperationItem{
							{
								ID:   "1",
								Name: "1",
							},
						},
					},
				},
			}
			roleTypes = []interfaces.SystemRoleType{interfaces.SuperAdmin}
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).Return(roleTypes, nil)
			tmpDB.EXPECT().GetPagination(gomock.Any(), gomock.Any()).Return(1, policies, nil)
			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), gomock.Any()).Return(resourceTypeInfoMap, nil)
			userMgnt.EXPECT().BatchGetUserInfoByID(gomock.Any(), gomock.Any()).Return(nil, testErr)
			_, _, err := policy.GetPagination(ctx, &visitor, params)
			assert.Equal(t, err, testErr)
		})
	})
}

func TestPolicyGetPagination1(t *testing.T) {
	Convey("获取策略分页接口, 逻辑判断", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tmpDB := mock.NewMockDBPolicy(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		policyCalc := mock.NewMockLogicsPolicyCalc(ctrl)
		policy := newPolicy(tmpDB, userMgnt, role, resourceType, policyCalc, resourceTypeHierarchy)

		ctx := context.Background()
		// CreatePrivate -> checkResourceIDAndAncestors 会调用 GetParentResourceTypeID；默认返回无父级，避免要求必须传 ancestors
		resourceTypeHierarchy.EXPECT().HasHierarchy(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(false, nil)
		visitor := interfaces.Visitor{
			ID:       accessorID,
			Type:     interfaces.RealName,
			Language: "zh-cn",
		}
		params := interfaces.PolicyPagination{
			ResourceID:   resourceID,
			ResourceType: resourceTypeDoc,
			Offset:       1,
			Limit:        10,
		}
		roleTypes := []interfaces.SystemRoleType{interfaces.SuperAdmin}
		policies := []interfaces.PolicyInfo{
			{
				ResourceType: resourceTypeDoc,
				ResourceID:   resourceID,
				AccessorID:   accessorID,
				AccessorType: interfaces.AccessorUser,
				Operation: interfaces.PolicyOperation{
					Allow: []interfaces.PolicyOperationItem{
						{
							ID: "delete",
						},
						{
							ID: "display",
						},
					},
				},
			},
			{
				ResourceType: resourceTypeDoc,
				ResourceID:   resourceID,
				AccessorID:   accessorID2,
				AccessorType: interfaces.AccessorDepartment,
				Operation: interfaces.PolicyOperation{
					Deny: []interfaces.PolicyOperationItem{
						{
							ID: "display",
						},
						{
							ID: "delete",
						},
						{
							ID: "create",
						},
					},
				},
			},
		}
		resourceTypeInfoMap := map[string]interfaces.ResourceType{
			resourceTypeDoc: {
				ID:   resourceTypeDoc,
				Name: "doc",
				Operation: []interfaces.ResourceTypeOperation{
					{
						ID: "display",
						Name: []interfaces.OperationName{
							{
								Language: "zh-cn",
								Value:    "显示",
							},
							{
								Language: "en-us",
								Value:    "show",
							},
						},
					},
					{
						ID: "create",
						Name: []interfaces.OperationName{
							{
								Language: "zh-CN",
								Value:    "新建",
							},
							{
								Language: "en-US",
								Value:    "create",
							},
						},
					},
					{
						ID: "delete",
						Name: []interfaces.OperationName{
							{
								Language: "zh-CN",
								Value:    "删除",
							},
							{
								Language: "en-US",
								Value:    "delete",
							},
						},
					},
				},
			},
		}

		dep := []interfaces.Department{
			{
				ID:   "1",
				Name: "1",
			},
		}
		batchGetUserInfoByID := map[string]interfaces.UserInfo{
			accessorID: {
				ID:         accessorID,
				VisionName: "1",
				ParentDeps: [][]interfaces.Department{
					{
						{
							ID:   "2",
							Name: "2",
						},
					},
				},
			},
		}
		userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).AnyTimes().Return(roleTypes, nil)
		userMgnt.EXPECT().GetParentDepartmentsByDepartmentID(gomock.Any(), gomock.Any()).AnyTimes().Return(dep, nil)
		userMgnt.EXPECT().BatchGetUserInfoByID(gomock.Any(), gomock.Any()).Return(batchGetUserInfoByID, nil)
		tmpDB.EXPECT().GetPagination(gomock.Any(), gomock.Any()).Return(2, policies, nil)
		Convey("超级管理员调用接口, 检查用户来自，检查部门的来自，检查操作顺序", func() {
			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), gomock.Any()).Return(resourceTypeInfoMap, nil)
			count, policies, err := policy.GetPagination(ctx, &visitor, params)
			assert.Equal(t, err, nil)
			assert.Equal(t, count, 2)
			assert.Equal(t, len(policies), 2)
			assert.Equal(t, policies[0].ResourceType, resourceTypeDoc)
			assert.Equal(t, policies[0].ResourceID, resourceID)
			assert.Equal(t, policies[0].AccessorID, accessorID)
			assert.Equal(t, policies[0].AccessorType, interfaces.AccessorUser)
			assert.Equal(t, policies[0].Operation.Allow[0].ID, "display")
			assert.Equal(t, policies[0].Operation.Allow[0].Name, "显示")
			assert.Equal(t, policies[0].Operation.Allow[1].ID, "delete")
			assert.Equal(t, policies[0].Operation.Allow[1].Name, "删除")
			assert.Equal(t, policies[0].ParentDeps, batchGetUserInfoByID[accessorID].ParentDeps)
			assert.Equal(t, policies[1].ResourceType, resourceTypeDoc)
			assert.Equal(t, policies[1].ResourceID, resourceID)
			assert.Equal(t, policies[1].AccessorID, accessorID2)
			assert.Equal(t, policies[1].AccessorType, interfaces.AccessorDepartment)
			assert.Equal(t, policies[1].Operation.Deny[0].ID, "display")
			assert.Equal(t, policies[1].Operation.Deny[0].Name, "显示")
			assert.Equal(t, policies[1].Operation.Deny[1].ID, "create")
			assert.Equal(t, policies[1].Operation.Deny[1].Name, "新建")
			assert.Equal(t, policies[1].Operation.Deny[2].ID, "delete")
			assert.Equal(t, policies[1].Operation.Deny[2].Name, "删除")
			assert.Equal(t, policies[1].ParentDeps, [][]interfaces.Department{dep})
		})
	})
}

func TestPolicyDelete(t *testing.T) {
	Convey("删除策略接口, 请求接口异常情况", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tmpDB := mock.NewMockDBPolicy(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		policyCalc := mock.NewMockLogicsPolicyCalc(ctrl)
		policy := newPolicy(tmpDB, userMgnt, role, resourceType, policyCalc, resourceTypeHierarchy)

		testErr := errors.New("some error")
		var ctx context.Context
		ctx = context.Background()
		visitor := interfaces.Visitor{
			ID:   accessorID,
			Type: interfaces.RealName,
		}
		policyID := "1"
		policyInfo := interfaces.PolicyInfo{
			ID:           policyID,
			ResourceType: resourceTypeDoc,
			ResourceID:   resourceID,
			AccessorID:   accessorID,
			AccessorType: interfaces.AccessorUser,
		}
		roleTypes := []interfaces.SystemRoleType{}
		Convey("入参ids为空", func() {
			err := policy.Delete(ctx, &visitor, []string{})
			assert.Equal(t, err, nil)
		})
		policyMap := map[string]interfaces.PolicyInfo{
			policyID: policyInfo,
		}
		Convey("获取策略出错", func() {
			tmpDB.EXPECT().GetByPolicyIDs(gomock.Any(), gomock.Any()).Return(policyMap, testErr)
			err := policy.Delete(ctx, &visitor, []string{policyID})
			assert.Equal(t, err, testErr)
		})

		Convey("获取系统角色出错", func() {
			tmpDB.EXPECT().GetByPolicyIDs(gomock.Any(), gomock.Any()).Return(policyMap, nil)
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).Return(roleTypes, testErr)
			err := policy.Delete(ctx, &visitor, []string{policyID})
			assert.NotEqual(t, err, nil)
		})

		Convey("非管理员 检查权限接口调用出错", func() {
			tmpDB.EXPECT().GetByPolicyIDs(gomock.Any(), gomock.Any()).Return(policyMap, nil)
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).Return(roleTypes, nil)
			policyCalc.EXPECT().Check(gomock.Any(), gomock.Any(), gomock.Any(), []string{authorizeOperation}, gomock.Any()).Return(interfaces.CheckResult{}, testErr)
			err := policy.Delete(ctx, &visitor, []string{policyID})
			assert.NotEqual(t, err, nil)
		})

		Convey("非管理员 检查权限接口正常，没有权限", func() {
			tmpDB.EXPECT().GetByPolicyIDs(gomock.Any(), gomock.Any()).Return(policyMap, nil)
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).Return(roleTypes, nil)
			policyCalc.EXPECT().Check(gomock.Any(), gomock.Any(), gomock.Any(), []string{authorizeOperation}, gomock.Any()).Return(interfaces.CheckResult{Result: false}, nil)
			err := policy.Delete(ctx, &visitor, []string{policyID})
			assert.NotEqual(t, err, nil)
		})

		Convey("超级管理员调用，删除策略接口出错", func() {
			tmpDB.EXPECT().GetByPolicyIDs(gomock.Any(), gomock.Any()).Return(policyMap, nil)
			roleTypes = []interfaces.SystemRoleType{interfaces.SuperAdmin}
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).Return(roleTypes, nil)
			tmpDB.EXPECT().Delete(gomock.Any(), []string{"1"}).Return(testErr)
			err := policy.Delete(ctx, &visitor, []string{policyID})
			assert.Equal(t, err, testErr)
		})

		Convey("超级管理员调用，删除策略接口正常", func() {
			tmpDB.EXPECT().GetByPolicyIDs(gomock.Any(), gomock.Any()).Return(policyMap, nil)
			roleTypes = []interfaces.SystemRoleType{interfaces.SuperAdmin}
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).Return(roleTypes, nil)
			tmpDB.EXPECT().Delete(gomock.Any(), []string{"1"}).Return(nil)
			err := policy.Delete(ctx, &visitor, []string{policyID})
			assert.Equal(t, err, nil)
		})
	})
}

func TestPolicyUpdate(t *testing.T) {
	Convey("更新策略接口, 请求接口异常情况", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tmpDB := mock.NewMockDBPolicy(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		policyCalc := mock.NewMockLogicsPolicyCalc(ctrl)
		policy := newPolicy(tmpDB, userMgnt, role, resourceType, policyCalc, resourceTypeHierarchy)

		testErr := errors.New("some error")
		var ctx context.Context
		ctx = context.Background()
		visitor := interfaces.Visitor{
			ID:   accessorID,
			Type: interfaces.RealName,
		}
		policyID := "1"
		policyInfo := interfaces.PolicyInfo{
			ID:           policyID,
			ResourceType: resourceTypeDoc,
			ResourceID:   resourceID,
			AccessorID:   accessorID,
			AccessorType: interfaces.AccessorUser,
			EndTime:      -1,
		}
		roleTypes := []interfaces.SystemRoleType{}
		Convey("入参policys为空", func() {
			err := policy.Update(ctx, &visitor, []interfaces.PolicyInfo{})
			assert.Equal(t, err, nil)
		})

		Convey("入参policys中id重复", func() {
			policyInfo2 := policyInfo
			err := policy.Update(ctx, &visitor, []interfaces.PolicyInfo{policyInfo, policyInfo2})
			assert.NotEqual(t, err, nil)
		})

		Convey("获取历史策略出错", func() {
			tmpDB.EXPECT().GetByPolicyIDs(gomock.Any(), gomock.Any()).Return(nil, testErr)
			err := policy.Update(ctx, &visitor, []interfaces.PolicyInfo{policyInfo})
			assert.NotEqual(t, err, nil)
		})

		Convey("获取历史策略成功 但是无策略", func() {
			oldPoliciesMap := map[string]interfaces.PolicyInfo{}
			tmpDB.EXPECT().GetByPolicyIDs(gomock.Any(), gomock.Any()).Return(oldPoliciesMap, nil)
			err := policy.Update(ctx, &visitor, []interfaces.PolicyInfo{policyInfo})
			assert.Equal(t, err, nil)
		})

		Convey("获取历史策略成功 有策略，非管理员 且没权限", func() {
			oldPoliciesMap := map[string]interfaces.PolicyInfo{
				policyID: policyInfo,
			}
			tmpDB.EXPECT().GetByPolicyIDs(gomock.Any(), gomock.Any()).Return(oldPoliciesMap, nil)
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).Return(roleTypes, nil)
			policyCalc.EXPECT().Check(gomock.Any(), gomock.Any(), gomock.Any(), []string{authorizeOperation}, gomock.Any()).Return(interfaces.CheckResult{Result: false}, nil)
			err := policy.Update(ctx, &visitor, []interfaces.PolicyInfo{policyInfo})
			assert.NotEqual(t, err, nil)
		})
	})
}

func TestPolicyUpdate1(t *testing.T) {
	Convey("更新策略接口, 管理员调用，操作不合法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tmpDB := mock.NewMockDBPolicy(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		policyCalc := mock.NewMockLogicsPolicyCalc(ctrl)
		policy := newPolicy(tmpDB, userMgnt, role, resourceType, policyCalc, resourceTypeHierarchy)
		ctx := context.Background()
		visitor := interfaces.Visitor{
			ID:   accessorID,
			Type: interfaces.RealName,
		}
		policyID := "1"
		oldPolicyInfo := interfaces.PolicyInfo{
			ID:           policyID,
			ResourceType: resourceTypeDoc,
			ResourceID:   resourceID,
			AccessorID:   accessorID,
			AccessorType: interfaces.AccessorUser,
			Operation: interfaces.PolicyOperation{
				Allow: []interfaces.PolicyOperationItem{
					{
						ID: "create",
					},
				},
			},
			EndTime: -1,
		}

		roleTypes := []interfaces.SystemRoleType{interfaces.SuperAdmin}
		resourceTypeInfoMap := map[string]interfaces.ResourceType{
			resourceTypeDoc: {
				ID:   resourceTypeDoc,
				Name: "doc",
				Operation: []interfaces.ResourceTypeOperation{
					{
						ID: "display",
					},
					{
						ID: "create",
					},
				},
			},
		}
		oldPoliciesMap := map[string]interfaces.PolicyInfo{
			policyID: oldPolicyInfo,
		}
		resourceType.EXPECT().GetByIDsInternal(gomock.Any(), gomock.Any()).Return(resourceTypeInfoMap, nil)
		tmpDB.EXPECT().GetByPolicyIDs(gomock.Any(), gomock.Any()).Return(oldPoliciesMap, nil)
		userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).Return(roleTypes, nil)
		Convey("获取历史策略成功, 操作不合法. allow和deny同时为空", func() {
			policyInfoEmpty := interfaces.PolicyInfo{
				ID:           policyID,
				ResourceType: resourceTypeDoc,
				ResourceID:   resourceID,
				AccessorID:   accessorID,
				AccessorType: interfaces.AccessorUser,
				Operation: interfaces.PolicyOperation{
					Allow: []interfaces.PolicyOperationItem{},
					Deny:  []interfaces.PolicyOperationItem{},
				},
				EndTime: -1,
			}
			err := policy.Update(ctx, &visitor, []interfaces.PolicyInfo{policyInfoEmpty})
			assert.NotEqual(t, err, nil)
		})

		Convey("获取历史策略成功, 操作不合法. allow 出现不合法的操作", func() {
			policyInfoEmpty := interfaces.PolicyInfo{
				ID:           policyID,
				ResourceType: resourceTypeDoc,
				ResourceID:   resourceID,
				AccessorID:   accessorID,
				AccessorType: interfaces.AccessorUser,
				Operation: interfaces.PolicyOperation{
					Allow: []interfaces.PolicyOperationItem{
						{
							ID: "delete",
						},
					},
					Deny: []interfaces.PolicyOperationItem{},
				},
				EndTime: -1,
			}
			err := policy.Update(ctx, &visitor, []interfaces.PolicyInfo{policyInfoEmpty})
			assert.NotEqual(t, err, nil)
		})

		Convey("获取历史策略成功, 操作不合法. deny 出现不合法的操作", func() {
			policyInfoEmpty := interfaces.PolicyInfo{
				ID:           policyID,
				ResourceType: resourceTypeDoc,
				ResourceID:   resourceID,
				AccessorID:   accessorID,
				AccessorType: interfaces.AccessorUser,
				Operation: interfaces.PolicyOperation{
					Allow: []interfaces.PolicyOperationItem{
						{
							ID: "delete",
						},
					},
					Deny: []interfaces.PolicyOperationItem{},
				},
				EndTime: -1,
			}
			err := policy.Update(ctx, &visitor, []interfaces.PolicyInfo{policyInfoEmpty})
			assert.NotEqual(t, err, nil)
		})
	})
}

func TestPolicyUpdate2(t *testing.T) {
	Convey("更新策略接口, 管理员调用，操作合法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tmpDB := mock.NewMockDBPolicy(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		policyCalc := mock.NewMockLogicsPolicyCalc(ctrl)
		policy := newPolicy(tmpDB, userMgnt, role, resourceType, policyCalc, resourceTypeHierarchy)
		dbPool, txMock, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func() {
			if closeErr := dbPool.Close(); closeErr != nil {
				assert.Equal(t, 1, 1)
			}
		}()
		policy.pool = dbPool
		var ctx context.Context
		ctx = context.Background()
		visitor := interfaces.Visitor{
			ID:   accessorID,
			Type: interfaces.RealName,
		}
		policyID := "1"
		oldPolicyInfo := interfaces.PolicyInfo{
			ID:           policyID,
			ResourceType: resourceTypeDoc,
			ResourceID:   resourceID,
			AccessorID:   accessorID,
			AccessorType: interfaces.AccessorUser,
			Operation: interfaces.PolicyOperation{
				Allow: []interfaces.PolicyOperationItem{
					{
						ID: "create",
					},
				},
			},
			EndTime: -1,
		}

		roleTypes := []interfaces.SystemRoleType{interfaces.SuperAdmin}
		resourceTypeInfoMap := map[string]interfaces.ResourceType{
			resourceTypeDoc: {
				ID:   resourceTypeDoc,
				Name: "doc",
				Operation: []interfaces.ResourceTypeOperation{
					{
						ID: "display",
					},
					{
						ID: "create",
					},
				},
			},
		}
		oldPoliciesMap := map[string]interfaces.PolicyInfo{
			policyID: oldPolicyInfo,
		}
		resourceType.EXPECT().GetByIDsInternal(gomock.Any(), gomock.Any()).Return(resourceTypeInfoMap, nil)
		tmpDB.EXPECT().GetByPolicyIDs(gomock.Any(), gomock.Any()).Return(oldPoliciesMap, nil)
		userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).Return(roleTypes, nil)
		policyInfoEmpty := interfaces.PolicyInfo{
			ID:           policyID,
			ResourceType: resourceTypeDoc,
			ResourceID:   resourceID,
			AccessorID:   accessorID,
			AccessorType: interfaces.AccessorUser,
			Operation: interfaces.PolicyOperation{
				Allow: []interfaces.PolicyOperationItem{
					{
						ID: "display",
					},
				},
				Deny: []interfaces.PolicyOperationItem{},
			},
			EndTime: -1,
		}

		Convey("获取历史策略成功, 操作合法. 写入数据库失败", func() {
			txMock.ExpectBegin()
			txMock.ExpectRollback()
			testErr := errors.New("some error")
			tmpDB.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(testErr)
			err := policy.Update(ctx, &visitor, []interfaces.PolicyInfo{policyInfoEmpty})
			assert.Equal(t, err, testErr)
		})

		Convey("获取历史策略成功, 操作合法. 写入数据库成功", func() {
			txMock.ExpectBegin()
			txMock.ExpectCommit()
			tmpDB.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			err := policy.Update(ctx, &visitor, []interfaces.PolicyInfo{policyInfoEmpty})
			assert.Equal(t, err, nil)
		})
	})
}

func TestPolicyCreate(t *testing.T) {
	Convey("创建策略接口, 请求接口异常情况", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tmpDB := mock.NewMockDBPolicy(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		policyCalc := mock.NewMockLogicsPolicyCalc(ctrl)
		policy := newPolicy(tmpDB, userMgnt, role, resourceType, policyCalc, resourceTypeHierarchy)
		var ctx context.Context
		ctx = context.Background()
		visitor := interfaces.Visitor{
			ID:   accessorID,
			Type: interfaces.RealName,
		}
		Convey("入参policys为空", func() {
			_, err := policy.Create(ctx, &visitor, []interfaces.PolicyInfo{})
			assert.Equal(t, err, nil)
		})
	})
}

func TestPolicyCreate1(t *testing.T) {
	Convey("创建策略接口, 请求接口异常情况", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tmpDB := mock.NewMockDBPolicy(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		policyCalc := mock.NewMockLogicsPolicyCalc(ctrl)
		policy := newPolicy(tmpDB, userMgnt, role, resourceType, policyCalc, resourceTypeHierarchy)
		var ctx context.Context
		ctx = context.Background()
		visitor := interfaces.Visitor{
			ID:   accessorID,
			Type: interfaces.RealName,
		}
		roleTypes := []interfaces.SystemRoleType{interfaces.SuperAdmin}
		resourceTypeInfoMap := map[string]interfaces.ResourceType{
			resourceTypeDoc: {
				ID:   resourceTypeDoc,
				Name: "doc",
				Operation: []interfaces.ResourceTypeOperation{
					{
						ID: "display",
					},
					{
						ID: "create",
					},
				},
			},
		}

		userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).Return(roleTypes, nil)
		resourceType.EXPECT().GetByIDsInternal(gomock.Any(), gomock.Any()).Return(resourceTypeInfoMap, nil)
		resourceTypeHierarchy.EXPECT().HasHierarchy(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(false, nil)
		Convey("超级管理员调用， 到期时间异常", func() {
			policyInfo := interfaces.PolicyInfo{
				ResourceType: resourceTypeDoc,
				ResourceID:   resourceID,
				AccessorID:   accessorID,
				AccessorType: interfaces.AccessorUser,
				EndTime:      12345,
			}
			_, err := policy.Create(ctx, &visitor, []interfaces.PolicyInfo{policyInfo})
			assert.NotEqual(t, err, nil)
		})

		Convey("超级管理员调用， 操作不合法", func() {
			policyInfo := interfaces.PolicyInfo{
				ResourceType: resourceTypeDoc,
				ResourceID:   resourceID,
				AccessorID:   accessorID,
				AccessorType: interfaces.AccessorUser,
				EndTime:      -1,
				Operation: interfaces.PolicyOperation{
					Allow: []interfaces.PolicyOperationItem{
						{
							ID: "delete",
						},
					},
				},
			}
			_, err := policy.Create(ctx, &visitor, []interfaces.PolicyInfo{policyInfo})
			assert.NotEqual(t, err, nil)
		})

		Convey("超级管理员调用， 同一资源类型 两个相同的资源实例ID", func() {
			policyInfo := interfaces.PolicyInfo{
				ResourceType: resourceTypeDoc,
				ResourceID:   resourceID,
				AccessorID:   accessorID,
				AccessorType: interfaces.AccessorUser,
				EndTime:      -1,
				Operation: interfaces.PolicyOperation{
					Allow: []interfaces.PolicyOperationItem{
						{
							ID: "display",
						},
					},
				},
			}
			_, err := policy.Create(ctx, &visitor, []interfaces.PolicyInfo{policyInfo, policyInfo})
			assert.NotEqual(t, err, nil)
		})

		Convey("超级管理员调用，角色信息接口，角色不存在", func() {
			policyInfo := interfaces.PolicyInfo{
				ResourceType: resourceTypeDoc,
				ResourceID:   resourceID,
				AccessorID:   accessorID,
				AccessorType: interfaces.AccessorRole,
				EndTime:      -1,
				Operation: interfaces.PolicyOperation{
					Allow: []interfaces.PolicyOperationItem{
						{
							ID: "display",
						},
					},
				},
			}
			roleInfoMap := make(map[string]interfaces.RoleInfo)
			role.EXPECT().GetRolesByIDs(gomock.Any(), gomock.Any()).Return(roleInfoMap, nil)
			_, err := policy.Create(ctx, &visitor, []interfaces.PolicyInfo{policyInfo})
			assert.NotEqual(t, err, nil)
		})
	})
}

func TestPolicyCreate2(t *testing.T) {
	Convey("创建策略接口, 请求接口异常情况2", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tmpDB := mock.NewMockDBPolicy(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		policyCalc := mock.NewMockLogicsPolicyCalc(ctrl)
		policy := newPolicy(tmpDB, userMgnt, role, resourceType, policyCalc, resourceTypeHierarchy)
		dbPool, txMock, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func() {
			if closeErr := dbPool.Close(); closeErr != nil {
				assert.Equal(t, 1, 1)
			}
		}()
		policy.pool = dbPool
		var ctx context.Context
		ctx = context.Background()
		visitor := interfaces.Visitor{
			ID:   accessorID,
			Type: interfaces.RealName,
		}
		roleTypes := []interfaces.SystemRoleType{interfaces.SuperAdmin}
		resourceTypeInfoMap := map[string]interfaces.ResourceType{
			resourceTypeDoc: {
				ID:   resourceTypeDoc,
				Name: "doc",
				Operation: []interfaces.ResourceTypeOperation{
					{
						ID: "display",
					},
					{
						ID: "create",
					},
				},
			},
		}

		idNameMap := make(map[string]string)
		idNameMap[accessorID] = "test"
		policyInfo := interfaces.PolicyInfo{
			ResourceType: resourceTypeDoc,
			ResourceID:   resourceID,
			AccessorID:   accessorID,
			AccessorType: interfaces.AccessorRole,
			EndTime:      -1,
			Operation: interfaces.PolicyOperation{
				Allow: []interfaces.PolicyOperationItem{
					{
						ID: "display",
					},
				},
			},
		}
		roleInfoMap := make(map[string]interfaces.RoleInfo)
		roleInfoMap[accessorID] = interfaces.RoleInfo{
			ID:   accessorID,
			Name: "test",
		}
		testErr := errors.New("some error")
		oldPoliciesMap := make(map[string][]interfaces.PolicyInfo)
		userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).Return(roleTypes, nil)
		resourceType.EXPECT().GetByIDsInternal(gomock.Any(), gomock.Any()).Return(resourceTypeInfoMap, nil)
		resourceTypeHierarchy.EXPECT().HasHierarchy(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(false, nil)
		role.EXPECT().GetRolesByIDs(gomock.Any(), gomock.Any()).Return(roleInfoMap, nil)
		tmpDB.EXPECT().GetByResourceIDs(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(oldPoliciesMap, nil)
		Convey("超级管理员调用，获取历史数据为空, 写入数据库成功", func() {
			txMock.ExpectBegin()
			txMock.ExpectCommit()
			tmpDB.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			_, err = policy.Create(ctx, &visitor, []interfaces.PolicyInfo{policyInfo})
			assert.Equal(t, err, nil)
		})

		Convey("超级管理员调用，获取历史数据为空, 写入数据库失败", func() {
			txMock.ExpectBegin()
			txMock.ExpectRollback()
			tmpDB.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(testErr)
			_, err = policy.Create(ctx, &visitor, []interfaces.PolicyInfo{policyInfo})
			assert.Equal(t, err, testErr)
		})
	})
}

// TestCalcMinEndTime 测试计算最小过期时间
func TestCalcMinEndTime(t *testing.T) {
	Convey("测试计算最小过期时间", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		policy := newPolicy(nil, nil, nil, nil, nil, nil)

		Convey("当两个时间都是-1时，应该返回-1", func() {
			result := policy.calcMinEndTime(-1, -1)
			assert.Equal(t, int64(-1), result)
		})

		Convey("当第一个时间是-1时，应该返回第二个时间", func() {
			result := policy.calcMinEndTime(-1, 1000)
			assert.Equal(t, int64(1000), result)
		})

		Convey("当第二个时间是-1时，应该返回第一个时间", func() {
			result := policy.calcMinEndTime(1000, -1)
			assert.Equal(t, int64(1000), result)
		})

		Convey("当第一个时间小于第二个时间时，应该返回第一个时间", func() {
			result := policy.calcMinEndTime(1000, 2000)
			assert.Equal(t, int64(1000), result)
		})

		Convey("当第二个时间小于第一个时间时，应该返回第二个时间", func() {
			result := policy.calcMinEndTime(2000, 1000)
			assert.Equal(t, int64(1000), result)
		})

		Convey("当两个时间相等时，应该返回任意一个时间", func() {
			result := policy.calcMinEndTime(1000, 1000)
			assert.Equal(t, int64(1000), result)
		})
	})
}

// TestCheckPolicyOperationValid 测试检查策略操作是否合法
func TestCheckPolicyOperationValid(t *testing.T) {
	Convey("测试检查策略操作是否合法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		policy := newPolicy(nil, nil, nil, nil, nil, nil)

		operationIDMap := map[string]bool{
			"read":   true,
			"write":  true,
			"delete": true,
		}

		Convey("当允许和拒绝都为空时，应该返回错误", func() {
			policyInfo := &interfaces.PolicyInfo{
				Operation: interfaces.PolicyOperation{
					Allow: []interfaces.PolicyOperationItem{},
					Deny:  []interfaces.PolicyOperationItem{},
				},
			}
			err := policy.checkPolicyOperationValid(operationIDMap, policyInfo)
			assert.NotEqual(t, err, nil)
		})

		Convey("当允许操作不在定义的操作中时，应该返回错误", func() {
			policyInfo := &interfaces.PolicyInfo{
				Operation: interfaces.PolicyOperation{
					Allow: []interfaces.PolicyOperationItem{
						{ID: "invalid-operation"},
					},
					Deny: []interfaces.PolicyOperationItem{},
				},
			}
			err := policy.checkPolicyOperationValid(operationIDMap, policyInfo)
			assert.NotEqual(t, err, nil)
		})
		Convey("当拒绝操作不在定义的操作中时，应该返回错误", func() {
			policyInfo := &interfaces.PolicyInfo{
				Operation: interfaces.PolicyOperation{
					Allow: []interfaces.PolicyOperationItem{},
					Deny: []interfaces.PolicyOperationItem{
						{ID: "invalid-operation"},
					},
				},
			}
			err := policy.checkPolicyOperationValid(operationIDMap, policyInfo)
			assert.NotEqual(t, err, nil)
		})

		Convey("当允许和拒绝包含相同操作时，应该返回错误", func() {
			policyInfo := &interfaces.PolicyInfo{
				Operation: interfaces.PolicyOperation{
					Allow: []interfaces.PolicyOperationItem{
						{ID: "read"},
					},
					Deny: []interfaces.PolicyOperationItem{
						{ID: "read"},
					},
				},
			}
			err := policy.checkPolicyOperationValid(operationIDMap, policyInfo)
			assert.NotEqual(t, err, nil)
		})

		Convey("当操作都合法时，应该返回nil", func() {
			policyInfo := &interfaces.PolicyInfo{
				Operation: interfaces.PolicyOperation{
					Allow: []interfaces.PolicyOperationItem{
						{ID: "read"},
					},
					Deny: []interfaces.PolicyOperationItem{
						{ID: "write"},
					},
				},
			}
			err := policy.checkPolicyOperationValid(operationIDMap, policyInfo)
			assert.Equal(t, err, nil)
		})
	})
}

// TestCmpPolicy 测试比较策略是否有修改
func TestCmpPolicy(t *testing.T) {
	Convey("测试比较策略是否有修改", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		policy := newPolicy(nil, nil, nil, nil, nil, nil)

		oldPolicy := &interfaces.PolicyInfo{
			EndTime: 1000,
			Operation: interfaces.PolicyOperation{
				Allow: []interfaces.PolicyOperationItem{
					{ID: "read"},
					{ID: "write"},
				},
				Deny: []interfaces.PolicyOperationItem{
					{ID: "delete"},
				},
			},
		}

		Convey("当过期时间不同时，应该返回false", func() {
			newPolicy := &interfaces.PolicyInfo{
				EndTime: 2000,
				Operation: interfaces.PolicyOperation{
					Allow: []interfaces.PolicyOperationItem{
						{ID: "read"},
						{ID: "write"},
					},
					Deny: []interfaces.PolicyOperationItem{
						{ID: "delete"},
					},
				},
			}
			result := policy.cmpPolicy(oldPolicy, newPolicy)
			assert.False(t, result)
		})

		Convey("当新策略的操作是旧策略的子集时，应该返回true", func() {
			newPolicy := &interfaces.PolicyInfo{
				EndTime: 1000,
				Operation: interfaces.PolicyOperation{
					Allow: []interfaces.PolicyOperationItem{
						{ID: "read"},
					},
				},
			}
			result := policy.cmpPolicy(oldPolicy, newPolicy)
			assert.True(t, result)
		})

		Convey("当新策略包含旧策略没有的操作时，应该返回false", func() {
			newPolicy := &interfaces.PolicyInfo{
				EndTime: 1000,
				Operation: interfaces.PolicyOperation{
					Allow: []interfaces.PolicyOperationItem{
						{ID: "read"},
						{ID: "write"},
						{ID: "new-operation"},
					},
				},
			}
			result := policy.cmpPolicy(oldPolicy, newPolicy)
			assert.False(t, result)
		})

		Convey("当策略完全相同时，应该返回true", func() {
			newPolicy := &interfaces.PolicyInfo{
				EndTime: 1000,
				Operation: interfaces.PolicyOperation{
					Allow: []interfaces.PolicyOperationItem{
						{ID: "read"},
						{ID: "write"},
					},
					Deny: []interfaces.PolicyOperationItem{
						{ID: "delete"},
					},
				},
			}
			result := policy.cmpPolicy(oldPolicy, newPolicy)
			assert.True(t, result)
		})
	})
}

// TestMergeNewPolicy 测试合并策略
func TestMergeNewPolicy(t *testing.T) {
	Convey("测试合并策略", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		policy := newPolicy(nil, nil, nil, nil, nil, nil)

		oldPolicy := &interfaces.PolicyInfo{
			ID:           "old-id",
			ResourceID:   resourceID,
			ResourceType: resourceTypeDoc,
			AccessorID:   accessorID,
			AccessorType: interfaces.AccessorUser,
			EndTime:      1000,
			Operation: interfaces.PolicyOperation{
				Allow: []interfaces.PolicyOperationItem{
					{ID: "read"},
					{ID: "write"},
				},
				Deny: []interfaces.PolicyOperationItem{
					{ID: "delete"},
				},
			},
		}

		newPolicy := &interfaces.PolicyInfo{
			EndTime: 500,
			Operation: interfaces.PolicyOperation{
				Allow: []interfaces.PolicyOperationItem{
					{ID: "write"},
					{ID: "create"},
				},
				Deny: []interfaces.PolicyOperationItem{
					{ID: "update"},
				},
			},
		}

		Convey("合并策略应该正确合并操作和过期时间", func() {
			result := policy.mergeNewPolicy(oldPolicy, newPolicy)
			assert.Equal(t, "old-id", result.ID)
			assert.Equal(t, resourceID, result.ResourceID)
			assert.Equal(t, resourceTypeDoc, result.ResourceType)
			assert.Equal(t, accessorID, result.AccessorID)
			assert.Equal(t, interfaces.AccessorUser, result.AccessorType)
			assert.Equal(t, int64(500), result.EndTime) // 应该返回较小的过期时间

			// 检查允许操作
			allowIDs := make(map[string]bool)
			for _, op := range result.Operation.Allow {
				allowIDs[op.ID] = true
			}
			assert.True(t, allowIDs["read"])
			assert.True(t, allowIDs["write"])
			assert.True(t, allowIDs["create"])

			// 检查拒绝操作
			denyIDs := make(map[string]bool)
			for _, op := range result.Operation.Deny {
				denyIDs[op.ID] = true
			}
			assert.True(t, denyIDs["delete"])
			assert.True(t, denyIDs["update"])
		})

		Convey("当新策略的拒绝操作与旧策略的允许操作冲突时，拒绝应该优先", func() {
			newPolicy := &interfaces.PolicyInfo{
				EndTime: 1500,
				Operation: interfaces.PolicyOperation{
					Allow: []interfaces.PolicyOperationItem{},
					Deny: []interfaces.PolicyOperationItem{
						{ID: "read"}, // 拒绝旧策略中允许的操作
					},
				},
			}

			result := policy.mergeNewPolicy(oldPolicy, newPolicy)

			// 检查允许操作中不应该包含被拒绝的操作
			allowIDs := make(map[string]bool)
			for _, op := range result.Operation.Allow {
				allowIDs[op.ID] = true
			}
			assert.False(t, allowIDs["read"]) // read应该被拒绝
			assert.True(t, allowIDs["write"]) // write应该仍然被允许

			// 检查拒绝操作
			denyIDs := make(map[string]bool)
			for _, op := range result.Operation.Deny {
				denyIDs[op.ID] = true
			}
			assert.True(t, denyIDs["read"])
			assert.True(t, denyIDs["delete"])
		})
	})
}

// TestCreatePrivate 测试私有创建策略
func TestCreatePrivate(t *testing.T) {
	Convey("测试私有创建策略", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tmpDB := mock.NewMockDBPolicy(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		policyCalc := mock.NewMockLogicsPolicyCalc(ctrl)
		policy := newPolicy(tmpDB, userMgnt, role, resourceType, policyCalc, resourceTypeHierarchy)

		ctx := context.Background()
		resourceTypeHierarchy.EXPECT().HasHierarchy(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(false, nil)

		Convey("当策略列表为空时，应该直接返回", func() {
			err := policy.CreatePrivate(ctx, []interfaces.PolicyInfo{})
			assert.Nil(t, err)
		})

		Convey("当获取资源类型操作失败时，应该返回错误", func() {
			testErr := errors.New("resource type error")
			resourceTypeInfoMap := map[string]interfaces.ResourceType{}
			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), gomock.Any()).Return(resourceTypeInfoMap, testErr)

			policyInfo := interfaces.PolicyInfo{
				ResourceType: resourceTypeDoc,
				ResourceID:   resourceID,
				AccessorID:   accessorID,
				AccessorType: interfaces.AccessorUser,
				EndTime:      -1,
				Operation: interfaces.PolicyOperation{
					Allow: []interfaces.PolicyOperationItem{
						{ID: "read"},
					},
				},
			}

			err := policy.CreatePrivate(ctx, []interfaces.PolicyInfo{policyInfo})
			assert.Equal(t, testErr, err)
		})

		Convey("当检查策略操作不合法时，应该返回错误", func() {
			resourceTypeInfoMap := map[string]interfaces.ResourceType{
				resourceTypeDoc: {
					ID: resourceTypeDoc,
					Operation: []interfaces.ResourceTypeOperation{
						{ID: "read"},
					},
				},
			}
			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), gomock.Any()).Return(resourceTypeInfoMap, nil)

			policyInfo := interfaces.PolicyInfo{
				ResourceType: resourceTypeDoc,
				ResourceID:   resourceID,
				AccessorID:   accessorID,
				AccessorType: interfaces.AccessorUser,
				EndTime:      -1,
				Operation: interfaces.PolicyOperation{
					Allow: []interfaces.PolicyOperationItem{
						{ID: "invalid-operation"},
					},
				},
			}

			err := policy.CreatePrivate(ctx, []interfaces.PolicyInfo{policyInfo})
			assert.NotNil(t, err)
		})

		Convey("当策略重复时，应该返回错误", func() {
			resourceTypeInfoMap := map[string]interfaces.ResourceType{
				resourceTypeDoc: {
					ID: resourceTypeDoc,
					Operation: []interfaces.ResourceTypeOperation{
						{ID: "read"},
					},
				},
			}
			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), gomock.Any()).Return(resourceTypeInfoMap, nil)

			policyInfo := interfaces.PolicyInfo{
				ResourceType: resourceTypeDoc,
				ResourceID:   resourceID,
				AccessorID:   accessorID,
				AccessorType: interfaces.AccessorUser,
				EndTime:      -1,
				Operation: interfaces.PolicyOperation{
					Allow: []interfaces.PolicyOperationItem{
						{ID: "read"},
					},
				},
			}

			err := policy.CreatePrivate(ctx, []interfaces.PolicyInfo{policyInfo, policyInfo})
			assert.NotNil(t, err)
		})
	})
}

// TestInitPolicy 测试初始化策略
func TestInitPolicy(t *testing.T) {
	Convey("测试初始化策略", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tmpDB := mock.NewMockDBPolicy(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		policyCalc := mock.NewMockLogicsPolicyCalc(ctrl)
		policy := newPolicy(tmpDB, userMgnt, role, resourceType, policyCalc, resourceTypeHierarchy)
		dbPool, txMock, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func() {
			if closeErr := dbPool.Close(); closeErr != nil {
				assert.Equal(t, 1, 1)
			}
		}()
		policy.pool = dbPool

		ctx := context.Background()

		testErr := errors.New("test error")
		Convey("当策略列表为空时，应该直接返回", func() {
			err := policy.InitPolicy(ctx, []interfaces.PolicyInfo{})
			assert.Nil(t, err)
		})

		Convey("当获取现有策略失败时，应该返回错误", func() {
			policsMap := make(map[string][]interfaces.PolicyInfo)
			tmpDB.EXPECT().GetByResourceIDs(gomock.Any(), gomock.Any(), gomock.Any()).Return(policsMap, testErr)
			policyInfo := interfaces.PolicyInfo{
				ResourceType: resourceTypeDoc,
				ResourceID:   resourceID,
				AccessorID:   accessorID,
				AccessorType: interfaces.AccessorUser,
				EndTime:      -1,
				Operation: interfaces.PolicyOperation{
					Allow: []interfaces.PolicyOperationItem{
						{ID: "read"},
					},
				},
			}

			err := policy.InitPolicy(ctx, []interfaces.PolicyInfo{policyInfo})
			assert.Equal(t, testErr, err)
		})

		// 获取策略成功
		Convey("获取策略成功, 新建权限", func() {
			policyInfo := interfaces.PolicyInfo{
				ResourceType: resourceTypeDoc,
				ResourceID:   resourceID,
				AccessorID:   accessorID,
				AccessorType: interfaces.AccessorUser,
				Operation: interfaces.PolicyOperation{
					Allow: []interfaces.PolicyOperationItem{
						{ID: "read"},
					},
				},
			}
			txMock.ExpectBegin()
			txMock.ExpectCommit()
			policsMap := make(map[string][]interfaces.PolicyInfo)
			tmpDB.EXPECT().GetByResourceIDs(gomock.Any(), gomock.Any(), gomock.Any()).Return(policsMap, nil)
			tmpDB.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			err := policy.InitPolicy(ctx, []interfaces.PolicyInfo{policyInfo})
			assert.Nil(t, err)
		})

		// 获取策略成功
		Convey("获取策略成功, 新建权限失败", func() {
			policyInfo := interfaces.PolicyInfo{
				ResourceType: resourceTypeDoc,
				ResourceID:   resourceID,
				AccessorID:   accessorID,
				AccessorType: interfaces.AccessorUser,
				Operation: interfaces.PolicyOperation{
					Allow: []interfaces.PolicyOperationItem{
						{ID: "read"},
					},
				},
			}
			txMock.ExpectBegin()
			txMock.ExpectRollback()
			policsMap := make(map[string][]interfaces.PolicyInfo)
			tmpDB.EXPECT().GetByResourceIDs(gomock.Any(), gomock.Any(), gomock.Any()).Return(policsMap, nil)
			tmpDB.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(testErr)
			err := policy.InitPolicy(ctx, []interfaces.PolicyInfo{policyInfo})
			assert.Equal(t, testErr, err)
		})
	})
}

// TestInitPolicy1 测试初始化策略
func TestInitPolicy1(t *testing.T) {
	Convey("测试初始化策略 修改权限", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tmpDB := mock.NewMockDBPolicy(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		policyCalc := mock.NewMockLogicsPolicyCalc(ctrl)
		policy := newPolicy(tmpDB, userMgnt, role, resourceType, policyCalc, resourceTypeHierarchy)
		dbPool, txMock, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func() {
			if closeErr := dbPool.Close(); closeErr != nil {
				assert.Equal(t, 1, 1)
			}
		}()
		policy.pool = dbPool

		ctx := context.Background()

		testErr := errors.New("test error")
		// 获取策略成功
		Convey("获取策略成功, 修改权限", func() {
			policyInfo := interfaces.PolicyInfo{
				ResourceType: resourceTypeDoc,
				ResourceID:   resourceID,
				AccessorID:   accessorID,
				AccessorType: interfaces.AccessorUser,
				Operation: interfaces.PolicyOperation{
					Allow: []interfaces.PolicyOperationItem{
						{ID: "read"},
					},
				},
			}

			policyInfo2 := interfaces.PolicyInfo{
				ResourceType: resourceTypeDoc,
				ResourceID:   resourceID,
				AccessorID:   accessorID,
				AccessorType: interfaces.AccessorUser,
				Operation: interfaces.PolicyOperation{
					Allow: []interfaces.PolicyOperationItem{
						{ID: "delete"},
					},
				},
			}
			policsMap := make(map[string][]interfaces.PolicyInfo)
			policsMap[resourceID] = []interfaces.PolicyInfo{policyInfo}
			txMock.ExpectBegin()
			txMock.ExpectCommit()
			tmpDB.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			tmpDB.EXPECT().GetByResourceIDs(gomock.Any(), gomock.Any(), gomock.Any()).Return(policsMap, nil)
			err := policy.InitPolicy(ctx, []interfaces.PolicyInfo{policyInfo2})
			assert.Nil(t, err)
		})

		// 获取策略成功
		Convey("获取策略成功, 修改权限失败", func() {
			policyInfo := interfaces.PolicyInfo{
				ResourceType: resourceTypeDoc,
				ResourceID:   resourceID,
				AccessorID:   accessorID,
				AccessorType: interfaces.AccessorUser,
				Operation: interfaces.PolicyOperation{
					Allow: []interfaces.PolicyOperationItem{
						{ID: "read"},
					},
				},
			}

			policyInfo2 := interfaces.PolicyInfo{
				ResourceType: resourceTypeDoc,
				ResourceID:   resourceID,
				AccessorID:   accessorID,
				AccessorType: interfaces.AccessorUser,
				Operation: interfaces.PolicyOperation{
					Allow: []interfaces.PolicyOperationItem{
						{ID: "delete"},
					},
				},
			}
			policsMap := make(map[string][]interfaces.PolicyInfo)
			policsMap[resourceID] = []interfaces.PolicyInfo{policyInfo}
			txMock.ExpectBegin()
			txMock.ExpectRollback()
			tmpDB.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(testErr)
			tmpDB.EXPECT().GetByResourceIDs(gomock.Any(), gomock.Any(), gomock.Any()).Return(policsMap, nil)
			err := policy.InitPolicy(ctx, []interfaces.PolicyInfo{policyInfo2})
			assert.Equal(t, testErr, err)
		})
	})
}

// TestDeleteByResourceIDs 测试根据资源ID删除策略
func TestDeleteByResourceIDs(t *testing.T) {
	Convey("测试根据资源ID删除策略", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tmpDB := mock.NewMockDBPolicy(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		policyCalc := mock.NewMockLogicsPolicyCalc(ctrl)
		policy := newPolicy(tmpDB, userMgnt, role, resourceType, policyCalc, resourceTypeHierarchy)

		Convey("当资源列表为空时，应该直接返回", func() {
			err := policy.DeleteByResourceIDs(context.Background(), []interfaces.PolicyDeleteResourceInfo{})
			assert.Nil(t, err)
		})

		Convey("当删除成功时，应该返回nil", func() {
			resources := []interfaces.PolicyDeleteResourceInfo{
				{ID: resourceID, Type: resourceTypeDoc},
			}
			tmpDB.EXPECT().DeleteByResourceIDs(gomock.Any(), resources).Return(nil)

			err := policy.DeleteByResourceIDs(context.Background(), resources)
			assert.Nil(t, err)
		})

		Convey("当删除失败时，应该返回错误", func() {
			testErr := errors.New("delete error")
			resources := []interfaces.PolicyDeleteResourceInfo{
				{ID: resourceID, Type: resourceTypeDoc},
			}
			tmpDB.EXPECT().DeleteByResourceIDs(gomock.Any(), resources).Return(testErr)

			err := policy.DeleteByResourceIDs(context.Background(), resources)
			assert.Equal(t, testErr, err)
		})
	})
}

// TestDeleteByEndTime 测试根据过期时间删除策略
func TestDeleteByEndTime(t *testing.T) {
	Convey("测试根据过期时间删除策略", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tmpDB := mock.NewMockDBPolicy(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		policyCalc := mock.NewMockLogicsPolicyCalc(ctrl)
		policy := newPolicy(tmpDB, userMgnt, role, resourceType, policyCalc, resourceTypeHierarchy)

		Convey("当删除成功时，应该返回nil", func() {
			curTime := int64(1000)
			tmpDB.EXPECT().DeleteByEndTime(curTime).Return(nil)

			err := policy.DeleteByEndTime(curTime)
			assert.Nil(t, err)
		})

		Convey("当删除失败时，应该返回错误", func() {
			testErr := errors.New("delete error")
			curTime := int64(1000)
			tmpDB.EXPECT().DeleteByEndTime(curTime).Return(testErr)

			err := policy.DeleteByEndTime(curTime)
			assert.Equal(t, testErr, err)
		})
	})
}

// TestUpdateResourceName 测试更新资源名称
func TestUpdateResourceName(t *testing.T) {
	Convey("测试更新资源名称", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tmpDB := mock.NewMockDBPolicy(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceTypeSvc := mock.NewMockLogicsResourceType(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		policyCalc := mock.NewMockLogicsPolicyCalc(ctrl)
		policy := newPolicy(tmpDB, userMgnt, role, resourceTypeSvc, policyCalc, resourceTypeHierarchy)

		ctx := context.Background()
		resourceID := resourceID
		resourceType := resourceTypeDoc
		name := "new-name"

		Convey("当更新成功时，应该返回nil", func() {
			tmpDB.EXPECT().UpdateResourceName(ctx, resourceID, resourceType, name).Return(nil)

			err := policy.UpdateResourceName(ctx, resourceID, resourceType, name)
			assert.Nil(t, err)
		})

		Convey("当更新失败时，应该返回错误", func() {
			testErr := errors.New("update error")
			tmpDB.EXPECT().UpdateResourceName(ctx, resourceID, resourceType, name).Return(testErr)

			err := policy.UpdateResourceName(ctx, resourceID, resourceType, name)
			assert.Equal(t, testErr, err)
		})
	})
}

// TestDeletePolicyByAccessorID 测试根据访问者ID删除策略
func TestDeletePolicyByAccessorID(t *testing.T) {
	Convey("测试根据访问者ID删除策略", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tmpDB := mock.NewMockDBPolicy(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		policyCalc := mock.NewMockLogicsPolicyCalc(ctrl)
		policy := newPolicy(tmpDB, userMgnt, role, resourceType, policyCalc, resourceTypeHierarchy)

		accessorID := "accessor-id"

		Convey("当删除成功时，应该返回nil", func() {
			tmpDB.EXPECT().DeleteByAccessorIDs([]string{accessorID}).Return(nil)

			err := policy.deletePolicyByAccessorID(accessorID)
			assert.Nil(t, err)
		})

		Convey("当删除失败时，应该返回错误", func() {
			testErr := errors.New("delete error")
			tmpDB.EXPECT().DeleteByAccessorIDs([]string{accessorID}).Return(testErr)

			err := policy.deletePolicyByAccessorID(accessorID)
			assert.Equal(t, testErr, err)
		})
	})
}

// TestUpdatePolicyAccessorName 测试更新策略访问者名称
func TestUpdatePolicyAccessorName(t *testing.T) {
	Convey("测试更新策略访问者名称", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tmpDB := mock.NewMockDBPolicy(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		policyCalc := mock.NewMockLogicsPolicyCalc(ctrl)
		policy := newPolicy(tmpDB, userMgnt, role, resourceType, policyCalc, resourceTypeHierarchy)

		id := "accessor-id"
		name := "new-name"

		Convey("当更新成功时，应该返回nil", func() {
			tmpDB.EXPECT().UpdateAccessorName(id, name).Return(nil)

			err := policy.updatePolicyAccessorName(id, name)
			assert.Nil(t, err)
		})

		Convey("当更新失败时，应该返回错误", func() {
			testErr := errors.New("update error")
			tmpDB.EXPECT().UpdateAccessorName(id, name).Return(testErr)

			err := policy.updatePolicyAccessorName(id, name)
			assert.Equal(t, testErr, err)
		})
	})
}

// TestUpdateAppName 测试更新应用名称
func TestUpdateAppName(t *testing.T) {
	Convey("测试更新应用名称", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tmpDB := mock.NewMockDBPolicy(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		policyCalc := mock.NewMockLogicsPolicyCalc(ctrl)
		policy := newPolicy(tmpDB, userMgnt, role, resourceType, policyCalc, resourceTypeHierarchy)

		appInfo := &interfaces.AppInfo{
			ID:   "app-id",
			Name: "new-app-name",
		}

		Convey("当更新成功时，应该返回nil", func() {
			tmpDB.EXPECT().UpdateAccessorName(appInfo.ID, appInfo.Name).Return(nil)

			err := policy.updateAppName(appInfo)
			assert.Nil(t, err)
		})

		Convey("当更新失败时，应该返回错误", func() {
			testErr := errors.New("update error")
			tmpDB.EXPECT().UpdateAccessorName(appInfo.ID, appInfo.Name).Return(testErr)

			err := policy.updateAppName(appInfo)
			assert.Equal(t, testErr, err)
		})
	})
}

// TestGetOperationNameByLanguage 测试根据语言获取操作名称
func TestGetOperationNameByLanguage(t *testing.T) {
	Convey("测试根据语言获取操作名称", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		policy := newPolicy(nil, nil, nil, nil, nil, nil)

		operationNames := []interfaces.OperationName{
			{Language: "zh-cn", Value: "读取"},
			{Language: "en-us", Value: "read"},
			{Language: "zh-CN", Value: "读取"},
		}

		Convey("当找到匹配的语言时，应该返回对应的名称", func() {
			result := policy.getOperationNameByLanguage(operationNames, "zh-cn")
			assert.Equal(t, "读取", result)
		})

		Convey("当语言不区分大小写时，应该能找到匹配", func() {
			result := policy.getOperationNameByLanguage(operationNames, "ZH-CN")
			assert.Equal(t, "读取", result)
		})

		Convey("当找不到匹配的语言时，应该返回空字符串", func() {
			result := policy.getOperationNameByLanguage(operationNames, "fr-fr")
			assert.Equal(t, "", result)
		})

		Convey("当操作名称为空时，应该返回空字符串", func() {
			result := policy.getOperationNameByLanguage([]interfaces.OperationName{}, "zh-cn")
			assert.Equal(t, "", result)
		})
	})
}

// TestGetResourceTypeOperations 测试获取资源类型操作
func TestGetResourceTypeOperations(t *testing.T) {
	Convey("测试获取资源类型操作", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tmpDB := mock.NewMockDBPolicy(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		policyCalc := mock.NewMockLogicsPolicyCalc(ctrl)
		policy := newPolicy(tmpDB, userMgnt, role, resourceType, policyCalc, resourceTypeHierarchy)

		ctx := context.Background()
		resourceTypeMap := map[string][]string{
			"type1": {"resource1", "resource2"},
			"type2": {"resource3"},
		}

		Convey("当获取资源类型信息失败时，应该返回错误", func() {
			testErr := errors.New("get resource type error")
			resourceType.EXPECT().GetByIDsInternal(ctx, gomock.Any()).Return(nil, testErr)

			result, err := policy.getResourceTypeOperations(ctx, resourceTypeMap)
			assert.Nil(t, result)
			assert.Equal(t, testErr, err)
		})

		Convey("当资源类型不存在时，应该返回错误", func() {
			resourceTypeInfoMap := map[string]interfaces.ResourceType{}
			resourceType.EXPECT().GetByIDsInternal(ctx, gomock.Any()).Return(resourceTypeInfoMap, nil)

			_, err := policy.getResourceTypeOperations(ctx, resourceTypeMap)
			assert.NotNil(t, err)
		})

		Convey("当获取成功时，应该返回正确的操作映射", func() {
			resourceTypeInfoMap := map[string]interfaces.ResourceType{
				"type1": {
					ID: "type1",
					Operation: []interfaces.ResourceTypeOperation{
						{ID: "read"},
						{ID: "write"},
					},
				},
				"type2": {
					ID: "type2",
					Operation: []interfaces.ResourceTypeOperation{
						{ID: "delete"},
					},
				},
			}
			resourceType.EXPECT().GetByIDsInternal(ctx, gomock.Any()).Return(resourceTypeInfoMap, nil)

			result, err := policy.getResourceTypeOperations(ctx, resourceTypeMap)
			assert.Nil(t, err)
			assert.NotNil(t, result)
			assert.True(t, result["type1"]["read"])
			assert.True(t, result["type1"]["write"])
			assert.True(t, result["type2"]["delete"])
			assert.False(t, result["type1"]["delete"])
		})
	})
}

// TestGetAccessorPolicy 测试获取访问者策略
//
//nolint:funlen
func TestGetAccessorPolicy(t *testing.T) {
	Convey("测试获取访问者策略", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tmpDB := mock.NewMockDBPolicy(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		policyCalc := mock.NewMockLogicsPolicyCalc(ctrl)
		policy := newPolicy(tmpDB, userMgnt, role, resourceType, policyCalc, resourceTypeHierarchy)

		ctx := context.Background()
		visitor := interfaces.Visitor{
			ID:       accessorID,
			Type:     interfaces.RealName,
			Language: "zh-cn",
		}

		Convey("当访问者ID为空时，应该返回错误", func() {
			param := interfaces.AccessorPolicyParam{
				AccessorID:   "",
				AccessorType: interfaces.AccessorUser,
			}

			_, _, _, err := policy.GetAccessorPolicy(ctx, &visitor, param)
			assert.NotNil(t, err)
		})

		Convey("当资源ID不为空但资源类型为空时，应该返回错误", func() {
			param := interfaces.AccessorPolicyParam{
				AccessorID:   accessorID,
				AccessorType: interfaces.AccessorUser,
				ResourceID:   resourceID,
				ResourceType: "",
			}

			_, _, _, err := policy.GetAccessorPolicy(ctx, &visitor, param)
			assert.NotNil(t, err)
		})

		Convey("当数据库查询失败时，应该返回错误", func() {
			param := interfaces.AccessorPolicyParam{
				AccessorID:   accessorID,
				AccessorType: interfaces.AccessorUser,
			}

			testErr := errors.New("database error")
			tmpDB.EXPECT().GetAccessorPolicy(ctx, param).Return(0, nil, testErr)

			_, _, _, err := policy.GetAccessorPolicy(ctx, &visitor, param)
			assert.Equal(t, testErr, err)
		})

		Convey("当获取资源类型操作失败时，应该返回错误", func() {
			param := interfaces.AccessorPolicyParam{
				AccessorID:   accessorID,
				AccessorType: interfaces.AccessorUser,
			}

			policies := []interfaces.PolicyInfo{
				{
					ID:           "policy1",
					ResourceID:   resourceID,
					ResourceType: resourceTypeDoc,
					AccessorID:   accessorID,
					AccessorType: interfaces.AccessorUser,
					Operation: interfaces.PolicyOperation{
						Allow: []interfaces.PolicyOperationItem{{ID: "read"}},
						Deny:  []interfaces.PolicyOperationItem{{ID: "write"}},
					},
				},
			}

			tmpDB.EXPECT().GetAccessorPolicy(ctx, param).Return(0, policies, nil)
			testErr := errors.New("get resource type operations error")
			resourceType.EXPECT().GetByIDsInternal(ctx, []string{resourceTypeDoc}).Return(nil, testErr)

			_, _, _, err := policy.GetAccessorPolicy(ctx, &visitor, param)
			assert.Equal(t, testErr, err)
		})

		Convey("当成功获取策略时，应该返回正确的策略列表", func() {
			param := interfaces.AccessorPolicyParam{
				AccessorID:   accessorID,
				AccessorType: interfaces.AccessorUser,
			}

			policies := []interfaces.PolicyInfo{
				{
					ID:           "policy1",
					ResourceID:   resourceID,
					ResourceType: resourceTypeDoc,
					AccessorID:   accessorID,
					AccessorType: interfaces.AccessorUser,
					Operation: interfaces.PolicyOperation{
						Allow: []interfaces.PolicyOperationItem{{ID: "read"}},
						Deny:  []interfaces.PolicyOperationItem{{ID: "write"}},
					},
				},
				{
					ID:           "policy2",
					ResourceID:   resourceID2,
					ResourceType: resourceTypeDoc,
					AccessorID:   accessorID,
					AccessorType: interfaces.AccessorUser,
					Operation: interfaces.PolicyOperation{
						Allow: []interfaces.PolicyOperationItem{{ID: "create"}},
						Deny:  []interfaces.PolicyOperationItem{},
					},
				},
			}

			resourceTypeInfo := map[string]interfaces.ResourceType{
				resourceTypeDoc: {
					ID: resourceTypeDoc,
					Operation: []interfaces.ResourceTypeOperation{
						{
							ID: "read",
							Name: []interfaces.OperationName{
								{Language: "zh-cn", Value: "读取"},
								{Language: "en-us", Value: "read"},
							},
						},
						{
							ID: "write",
							Name: []interfaces.OperationName{
								{Language: "zh-cn", Value: "写入"},
								{Language: "en-us", Value: "write"},
							},
						},
						{
							ID: "create",
							Name: []interfaces.OperationName{
								{Language: "zh-cn", Value: "创建"},
								{Language: "en-us", Value: "create"},
							},
						},
					},
				},
			}

			tmpDB.EXPECT().GetAccessorPolicy(ctx, param).Return(1, policies, nil)
			resourceType.EXPECT().GetByIDsInternal(ctx, []string{resourceTypeDoc}).Return(resourceTypeInfo, nil)

			_, result, _, err := policy.GetAccessorPolicy(ctx, &visitor, param)
			assert.Nil(t, err)
			assert.Equal(t, 2, len(result))
			assert.Equal(t, "policy1", result[0].ID)
			assert.Equal(t, "policy2", result[1].ID)

			// 验证操作名称已正确填充
			assert.Equal(t, "读取", result[0].Operation.Allow[0].Name)
			assert.Equal(t, "写入", result[0].Operation.Deny[0].Name)
			assert.Equal(t, "创建", result[1].Operation.Allow[0].Name)
		})

		Convey("当策略列表为空时，应该返回空列表", func() {
			param := interfaces.AccessorPolicyParam{
				AccessorID:   accessorID,
				AccessorType: interfaces.AccessorUser,
			}

			tmpDB.EXPECT().GetAccessorPolicy(ctx, param).Return(0, []interfaces.PolicyInfo{}, nil)

			_, result, _, err := policy.GetAccessorPolicy(ctx, &visitor, param)
			assert.Nil(t, err)
			assert.Equal(t, 0, len(result))
		})

		Convey("当有多个资源类型时，应该正确处理", func() {
			param := interfaces.AccessorPolicyParam{
				AccessorID:   accessorID,
				AccessorType: interfaces.AccessorUser,
			}

			policies := []interfaces.PolicyInfo{
				{
					ID:           "policy1",
					ResourceID:   resourceID,
					ResourceType: resourceTypeDoc,
					AccessorID:   accessorID,
					AccessorType: interfaces.AccessorUser,
					Operation: interfaces.PolicyOperation{
						Allow: []interfaces.PolicyOperationItem{{ID: "read"}},
						Deny:  []interfaces.PolicyOperationItem{},
					},
				},
				{
					ID:           "policy2",
					ResourceID:   resourceID2,
					ResourceType: resourceTypeMcp,
					AccessorID:   accessorID,
					AccessorType: interfaces.AccessorUser,
					Operation: interfaces.PolicyOperation{
						Allow: []interfaces.PolicyOperationItem{{ID: "display"}},
						Deny:  []interfaces.PolicyOperationItem{},
					},
				},
			}

			resourceTypeInfo := map[string]interfaces.ResourceType{
				resourceTypeDoc: {
					ID: resourceTypeDoc,
					Operation: []interfaces.ResourceTypeOperation{
						{
							ID: "read",
							Name: []interfaces.OperationName{
								{Language: "zh-cn", Value: "读取"},
							},
						},
					},
				},
				resourceTypeMcp: {
					ID: resourceTypeMcp,
					Operation: []interfaces.ResourceTypeOperation{
						{
							ID: "display",
							Name: []interfaces.OperationName{
								{Language: "zh-cn", Value: "显示"},
							},
						},
					},
				},
			}

			tmpDB.EXPECT().GetAccessorPolicy(ctx, param).Return(0, policies, nil)
			resourceType.EXPECT().GetByIDsInternal(ctx, []string{resourceTypeDoc, resourceTypeMcp}).Return(resourceTypeInfo, nil)

			_, result, _, err := policy.GetAccessorPolicy(ctx, &visitor, param)
			assert.Nil(t, err)
			assert.Equal(t, 2, len(result))
			assert.Equal(t, "读取", result[0].Operation.Allow[0].Name)
			assert.Equal(t, "显示", result[1].Operation.Allow[0].Name)
		})
	})
}

// TestGetResourcePolicy 测试获取资源策略
//
//nolint:funlen
func TestGetResourcePolicy(t *testing.T) {
	Convey("测试获取资源策略", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tmpDB := mock.NewMockDBPolicy(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceTypeSvc := mock.NewMockLogicsResourceType(ctrl)
		policyCalc := mock.NewMockLogicsPolicyCalc(ctrl)
		obligationTypeSvc := mock.NewMockObligationType(ctrl)
		obligationSvc := mock.NewMockLogicsObligation(ctrl)

		policy := &policy{
			db:             tmpDB,
			userMgmt:       userMgnt,
			logger:         common.NewLogger(),
			role:           role,
			resourceType:   resourceTypeSvc,
			policyCalc:     policyCalc,
			obligationType: obligationTypeSvc,
			obligation:     obligationSvc,
			i18n: common.NewI18n(common.I18nMap{
				i18nAccessorRoleNotFound: {
					simplifiedChinese:  "角色不存在",
					traditionalChinese: "角色不存在",
					americanEnglish:    "The Role does not exist.",
				},
			}),
		}

		ctx := context.Background()
		visitor := &interfaces.Visitor{
			ID:       accessorID,
			Type:     interfaces.RealName,
			Language: "zh-cn",
		}

		params := interfaces.ResourcePolicyPagination{
			ResourceID:   resourceID,
			ResourceType: resourceTypeDoc,
			Offset:       0,
			Limit:        10,
		}

		roleTypes := []interfaces.SystemRoleType{interfaces.SuperAdmin}
		calcIncludeParams := []interfaces.PolicCalcyIncludeType{}

		Convey("权限检查失败 - 获取用户角色失败", func() {
			testErr := errors.New("get user roles error")
			userMgnt.EXPECT().GetUserRolesByUserID(ctx, accessorID).Return(nil, testErr)

			_, _, _, err := policy.GetResourcePolicy(ctx, visitor, params)
			assert.Equal(t, testErr, err)
		})

		Convey("权限检查失败 - 没有授权权限", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(ctx, accessorID).Return([]interfaces.SystemRoleType{}, nil)
			policyCalc.EXPECT().Check(gomock.Any(), gomock.Any(), gomock.Any(), []string{authorizeOperation}, calcIncludeParams).Return(interfaces.CheckResult{Result: false}, nil)

			_, _, _, err := policy.GetResourcePolicy(ctx, visitor, params)
			assert.NotNil(t, err)
		})

		Convey("数据库查询失败", func() {
			testErr := errors.New("database error")
			userMgnt.EXPECT().GetUserRolesByUserID(ctx, accessorID).Return(roleTypes, nil)
			tmpDB.EXPECT().GetPagination(ctx, gomock.Any()).Return(0, nil, testErr)

			_, _, _, err := policy.GetResourcePolicy(ctx, visitor, params)
			assert.Equal(t, testErr, err)
		})

		Convey("策略列表为空", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(ctx, accessorID).Return(roleTypes, nil)
			tmpDB.EXPECT().GetPagination(ctx, gomock.Any()).Return(0, []interfaces.PolicyInfo{}, nil)

			count, policies, _, err := policy.GetResourcePolicy(ctx, visitor, params)
			assert.Nil(t, err)
			assert.Equal(t, 0, count)
			assert.Equal(t, 0, len(policies))
		})

		Convey("获取资源类型操作失败", func() {
			testErr := errors.New("get resource type operations error")
			policies := []interfaces.PolicyInfo{
				{
					ResourceType: resourceTypeDoc,
					ResourceID:   resourceID,
					AccessorID:   accessorID,
					AccessorType: interfaces.AccessorUser,
					Operation: interfaces.PolicyOperation{
						Allow: []interfaces.PolicyOperationItem{{ID: "read"}},
					},
				},
			}

			userMgnt.EXPECT().GetUserRolesByUserID(ctx, accessorID).Return(roleTypes, nil)
			tmpDB.EXPECT().GetPagination(ctx, gomock.Any()).Return(1, policies, nil)
			resourceTypeSvc.EXPECT().GetByIDsInternal(ctx, []string{resourceTypeDoc}).Return(nil, testErr)

			_, _, _, err := policy.GetResourcePolicy(ctx, visitor, params)
			assert.Equal(t, testErr, err)
		})

		Convey("获取部门父部门信息失败", func() {
			testErr := errors.New("get parent department error")
			policies := []interfaces.PolicyInfo{
				{
					ResourceType: resourceTypeDoc,
					ResourceID:   resourceID,
					AccessorID:   "dep-id",
					AccessorType: interfaces.AccessorDepartment,
					Operation: interfaces.PolicyOperation{
						Allow: []interfaces.PolicyOperationItem{{ID: "read"}},
					},
				},
			}

			resourceTypeInfo := map[string]interfaces.ResourceType{
				resourceTypeDoc: {
					ID: resourceTypeDoc,
					Operation: []interfaces.ResourceTypeOperation{
						{
							ID: "read",
							Name: []interfaces.OperationName{
								{Language: "zh-cn", Value: "读取"},
							},
						},
					},
				},
			}

			userMgnt.EXPECT().GetUserRolesByUserID(ctx, accessorID).Return(roleTypes, nil)
			tmpDB.EXPECT().GetPagination(ctx, gomock.Any()).Return(1, policies, nil)
			resourceTypeSvc.EXPECT().GetByIDsInternal(ctx, []string{resourceTypeDoc}).Return(resourceTypeInfo, nil)
			userMgnt.EXPECT().GetParentDepartmentsByDepartmentID(ctx, "dep-id").Return(nil, testErr)

			_, _, _, err := policy.GetResourcePolicy(ctx, visitor, params)
			assert.Equal(t, testErr, err)
		})

		Convey("获取用户父部门信息失败", func() {
			testErr := errors.New("get user info error")
			policies := []interfaces.PolicyInfo{
				{
					ResourceType: resourceTypeDoc,
					ResourceID:   resourceID,
					AccessorID:   accessorID,
					AccessorType: interfaces.AccessorUser,
					Operation: interfaces.PolicyOperation{
						Allow: []interfaces.PolicyOperationItem{{ID: "read"}},
					},
				},
			}

			resourceTypeInfo := map[string]interfaces.ResourceType{
				resourceTypeDoc: {
					ID: resourceTypeDoc,
					Operation: []interfaces.ResourceTypeOperation{
						{
							ID: "read",
							Name: []interfaces.OperationName{
								{Language: "zh-cn", Value: "读取"},
							},
						},
					},
				},
			}

			userMgnt.EXPECT().GetUserRolesByUserID(ctx, accessorID).Return(roleTypes, nil)
			tmpDB.EXPECT().GetPagination(ctx, gomock.Any()).Return(1, policies, nil)
			resourceTypeSvc.EXPECT().GetByIDsInternal(ctx, []string{resourceTypeDoc}).Return(resourceTypeInfo, nil)
			userMgnt.EXPECT().BatchGetUserInfoByID(ctx, []string{accessorID}).Return(nil, testErr)

			_, _, _, err := policy.GetResourcePolicy(ctx, visitor, params)
			assert.Equal(t, testErr, err)
		})

		Convey("成功获取资源策略 - 基本情况", func() {
			policies := []interfaces.PolicyInfo{
				{
					ID:           "policy1",
					ResourceType: resourceTypeDoc,
					ResourceID:   resourceID,
					AccessorID:   accessorID,
					AccessorType: interfaces.AccessorUser,
					Operation: interfaces.PolicyOperation{
						Allow: []interfaces.PolicyOperationItem{
							{ID: "read"},
							{ID: "write"},
						},
						Deny: []interfaces.PolicyOperationItem{
							{ID: "delete"},
						},
					},
				},
			}

			resourceTypeInfo := map[string]interfaces.ResourceType{
				resourceTypeDoc: {
					ID: resourceTypeDoc,
					Operation: []interfaces.ResourceTypeOperation{
						{
							ID: "read",
							Name: []interfaces.OperationName{
								{Language: "zh-cn", Value: "读取"},
							},
						},
						{
							ID: "write",
							Name: []interfaces.OperationName{
								{Language: "zh-cn", Value: "写入"},
							},
						},
						{
							ID: "delete",
							Name: []interfaces.OperationName{
								{Language: "zh-cn", Value: "删除"},
							},
						},
					},
				},
			}

			userInfo := map[string]interfaces.UserInfo{
				accessorID: {
					ID:         accessorID,
					ParentDeps: [][]interfaces.Department{},
				},
			}

			userMgnt.EXPECT().GetUserRolesByUserID(ctx, accessorID).Return(roleTypes, nil)
			tmpDB.EXPECT().GetPagination(ctx, gomock.Any()).Return(1, policies, nil)
			resourceTypeSvc.EXPECT().GetByIDsInternal(ctx, []string{resourceTypeDoc}).Return(resourceTypeInfo, nil)
			userMgnt.EXPECT().BatchGetUserInfoByID(ctx, []string{accessorID}).Return(userInfo, nil)

			count, result, includeResp, err := policy.GetResourcePolicy(ctx, visitor, params)
			assert.Nil(t, err)
			assert.Equal(t, 1, count)
			assert.Equal(t, 1, len(result))
			assert.Equal(t, "policy1", result[0].ID)
			assert.Equal(t, "读取", result[0].Operation.Allow[0].Name)
			assert.Equal(t, "写入", result[0].Operation.Allow[1].Name)
			assert.Equal(t, "删除", result[0].Operation.Deny[0].Name)
			assert.Equal(t, 0, len(includeResp.ObligationTypes))
			assert.Equal(t, 0, len(includeResp.Obligations))
		})

		Convey("成功获取资源策略 - 带义务", func() {
			policies := []interfaces.PolicyInfo{
				{
					ID:           "policy1",
					ResourceType: resourceTypeDoc,
					ResourceID:   resourceID,
					AccessorID:   accessorID,
					AccessorType: interfaces.AccessorUser,
					Operation: interfaces.PolicyOperation{
						Allow: []interfaces.PolicyOperationItem{
							{
								ID: "read",
								Obligations: []interfaces.PolicyObligationItem{
									{TypeID: obligationTypeID1, ID: obligationID1},
									{TypeID: obligationTypeID2, Value: map[string]any{"key": "value"}},
								},
							},
						},
					},
				},
			}

			resourceTypeInfo := map[string]interfaces.ResourceType{
				resourceTypeDoc: {
					ID: resourceTypeDoc,
					Operation: []interfaces.ResourceTypeOperation{
						{
							ID: "read",
							Name: []interfaces.OperationName{
								{Language: "zh-cn", Value: "读取"},
							},
						},
					},
				},
			}

			userInfo := map[string]interfaces.UserInfo{
				accessorID: {
					ID:         accessorID,
					ParentDeps: [][]interfaces.Department{},
				},
			}

			obligationTypes := []interfaces.ObligationTypeInfo{
				{ID: obligationTypeID1, Name: "水印"},
				{ID: obligationTypeID2, Name: "审计"},
			}

			obligations := []interfaces.ObligationInfo{
				{ID: obligationID1, TypeID: obligationTypeID1},
			}

			userMgnt.EXPECT().GetUserRolesByUserID(ctx, accessorID).Return(roleTypes, nil)
			tmpDB.EXPECT().GetPagination(ctx, gomock.Any()).Return(1, policies, nil)
			resourceTypeSvc.EXPECT().GetByIDsInternal(ctx, []string{resourceTypeDoc}).Return(resourceTypeInfo, nil)
			userMgnt.EXPECT().BatchGetUserInfoByID(ctx, []string{accessorID}).Return(userInfo, nil)
			obligationTypeSvc.EXPECT().GetByIDSInternal(ctx, gomock.Any()).Return(obligationTypes, nil)
			obligationSvc.EXPECT().GetByIDSInternal(ctx, gomock.Any()).Return(obligations, nil)

			count, result, _, err := policy.GetResourcePolicy(ctx, visitor, params)
			assert.Nil(t, err)
			assert.Equal(t, 1, count)
			assert.Equal(t, 1, len(result))
			assert.Equal(t, 2, len(result[0].Operation.Allow[0].Obligations))
		})

		Convey("成功获取资源策略 - 带include参数", func() {
			paramsWithInclude := interfaces.ResourcePolicyPagination{
				ResourceID:   resourceID,
				ResourceType: resourceTypeDoc,
				Offset:       0,
				Limit:        10,
				Include:      []interfaces.PolicyIncludeType{interfaces.PolicyIncludeObligationType, interfaces.PolicyIncludeObligation},
			}

			policies := []interfaces.PolicyInfo{
				{
					ID:           "policy1",
					ResourceType: resourceTypeDoc,
					ResourceID:   resourceID,
					AccessorID:   accessorID,
					AccessorType: interfaces.AccessorUser,
					Operation: interfaces.PolicyOperation{
						Allow: []interfaces.PolicyOperationItem{
							{
								ID: "read",
								Obligations: []interfaces.PolicyObligationItem{
									{TypeID: obligationTypeID1, ID: obligationID1},
								},
							},
						},
					},
				},
			}

			resourceTypeInfo := map[string]interfaces.ResourceType{
				resourceTypeDoc: {
					ID: resourceTypeDoc,
					Operation: []interfaces.ResourceTypeOperation{
						{
							ID: "read",
							Name: []interfaces.OperationName{
								{Language: "zh-cn", Value: "读取"},
							},
						},
					},
				},
			}

			userInfo := map[string]interfaces.UserInfo{
				accessorID: {
					ID:         accessorID,
					ParentDeps: [][]interfaces.Department{},
				},
			}

			obligationTypes := []interfaces.ObligationTypeInfo{
				{ID: obligationTypeID1, Name: "水印"},
			}

			obligations := []interfaces.ObligationInfo{
				{ID: obligationID1, TypeID: obligationTypeID1, Name: "水印1"},
			}

			userMgnt.EXPECT().GetUserRolesByUserID(ctx, accessorID).Return(roleTypes, nil)
			tmpDB.EXPECT().GetPagination(ctx, gomock.Any()).Return(1, policies, nil)
			resourceTypeSvc.EXPECT().GetByIDsInternal(ctx, []string{resourceTypeDoc}).Return(resourceTypeInfo, nil)
			userMgnt.EXPECT().BatchGetUserInfoByID(ctx, []string{accessorID}).Return(userInfo, nil)
			obligationTypeSvc.EXPECT().GetByIDSInternal(ctx, gomock.Any()).Return(obligationTypes, nil)
			obligationSvc.EXPECT().GetByIDSInternal(ctx, gomock.Any()).Return(obligations, nil)

			count, result, includeResp, err := policy.GetResourcePolicy(ctx, visitor, paramsWithInclude)
			assert.Nil(t, err)
			assert.Equal(t, 1, count)
			assert.Equal(t, 1, len(result))
			assert.Equal(t, 1, len(includeResp.ObligationTypes))
			assert.Equal(t, 1, len(includeResp.Obligations))
			assert.Equal(t, obligationTypeID1, includeResp.ObligationTypes[0].ID)
			assert.Equal(t, obligationID1, includeResp.Obligations[0].ID)
		})

		Convey("过滤无效的义务类型", func() {
			policies := []interfaces.PolicyInfo{
				{
					ID:           "policy1",
					ResourceType: resourceTypeDoc,
					ResourceID:   resourceID,
					AccessorID:   accessorID,
					AccessorType: interfaces.AccessorUser,
					Operation: interfaces.PolicyOperation{
						Allow: []interfaces.PolicyOperationItem{
							{
								ID: "read",
								Obligations: []interfaces.PolicyObligationItem{
									{TypeID: obligationTypeID1, ID: obligationID1},
									{TypeID: "invalid-type-id", ID: "invalid-obligation-id"},
								},
							},
						},
					},
				},
			}

			resourceTypeInfo := map[string]interfaces.ResourceType{
				resourceTypeDoc: {
					ID: resourceTypeDoc,
					Operation: []interfaces.ResourceTypeOperation{
						{
							ID: "read",
							Name: []interfaces.OperationName{
								{Language: "zh-cn", Value: "读取"},
							},
						},
					},
				},
			}

			userInfo := map[string]interfaces.UserInfo{
				accessorID: {
					ID:         accessorID,
					ParentDeps: [][]interfaces.Department{},
				},
			}

			// 只返回有效的义务类型
			obligationTypes := []interfaces.ObligationTypeInfo{
				{ID: obligationTypeID1, Name: "水印"},
			}

			obligations := []interfaces.ObligationInfo{
				{ID: obligationID1, TypeID: obligationTypeID1},
			}

			userMgnt.EXPECT().GetUserRolesByUserID(ctx, accessorID).Return(roleTypes, nil)
			tmpDB.EXPECT().GetPagination(ctx, gomock.Any()).Return(1, policies, nil)
			resourceTypeSvc.EXPECT().GetByIDsInternal(ctx, []string{resourceTypeDoc}).Return(resourceTypeInfo, nil)
			userMgnt.EXPECT().BatchGetUserInfoByID(ctx, []string{accessorID}).Return(userInfo, nil)
			obligationTypeSvc.EXPECT().GetByIDSInternal(ctx, gomock.Any()).Return(obligationTypes, nil)
			obligationSvc.EXPECT().GetByIDSInternal(ctx, gomock.Any()).Return(obligations, nil)

			_, result, _, err := policy.GetResourcePolicy(ctx, visitor, params)
			assert.Nil(t, err)
			// 应该过滤掉无效的义务类型
			assert.Equal(t, 1, len(result[0].Operation.Allow[0].Obligations))
			assert.Equal(t, obligationTypeID1, result[0].Operation.Allow[0].Obligations[0].TypeID)
		})

		Convey("过滤无效的义务", func() {
			policies := []interfaces.PolicyInfo{
				{
					ID:           "policy1",
					ResourceType: resourceTypeDoc,
					ResourceID:   resourceID,
					AccessorID:   accessorID,
					AccessorType: interfaces.AccessorUser,
					Operation: interfaces.PolicyOperation{
						Allow: []interfaces.PolicyOperationItem{
							{
								ID: "read",
								Obligations: []interfaces.PolicyObligationItem{
									{TypeID: obligationTypeID1, ID: obligationID1},
									{TypeID: obligationTypeID1, ID: "invalid-obligation-id"},
								},
							},
						},
					},
				},
			}

			resourceTypeInfo := map[string]interfaces.ResourceType{
				resourceTypeDoc: {
					ID: resourceTypeDoc,
					Operation: []interfaces.ResourceTypeOperation{
						{
							ID: "read",
							Name: []interfaces.OperationName{
								{Language: "zh-cn", Value: "读取"},
							},
						},
					},
				},
			}

			userInfo := map[string]interfaces.UserInfo{
				accessorID: {
					ID:         accessorID,
					ParentDeps: [][]interfaces.Department{},
				},
			}

			obligationTypes := []interfaces.ObligationTypeInfo{
				{ID: obligationTypeID1, Name: "水印"},
			}

			// 只返回有效的义务
			obligations := []interfaces.ObligationInfo{
				{ID: obligationID1, TypeID: obligationTypeID1},
			}

			userMgnt.EXPECT().GetUserRolesByUserID(ctx, accessorID).Return(roleTypes, nil)
			tmpDB.EXPECT().GetPagination(ctx, gomock.Any()).Return(1, policies, nil)
			resourceTypeSvc.EXPECT().GetByIDsInternal(ctx, []string{resourceTypeDoc}).Return(resourceTypeInfo, nil)
			userMgnt.EXPECT().BatchGetUserInfoByID(ctx, []string{accessorID}).Return(userInfo, nil)
			obligationTypeSvc.EXPECT().GetByIDSInternal(ctx, gomock.Any()).Return(obligationTypes, nil)
			obligationSvc.EXPECT().GetByIDSInternal(ctx, gomock.Any()).Return(obligations, nil)

			_, result, _, err := policy.GetResourcePolicy(ctx, visitor, params)
			assert.Nil(t, err)
			// 应该过滤掉无效的义务
			assert.Equal(t, 1, len(result[0].Operation.Allow[0].Obligations))
			assert.Equal(t, obligationID1, result[0].Operation.Allow[0].Obligations[0].ID)
		})

		Convey("处理部门访问者的父部门信息", func() {
			depID := "dep-id"
			policies := []interfaces.PolicyInfo{
				{
					ID:           "policy1",
					ResourceType: resourceTypeDoc,
					ResourceID:   resourceID,
					AccessorID:   depID,
					AccessorType: interfaces.AccessorDepartment,
					Operation: interfaces.PolicyOperation{
						Allow: []interfaces.PolicyOperationItem{{ID: "read"}},
					},
				},
			}

			resourceTypeInfo := map[string]interfaces.ResourceType{
				resourceTypeDoc: {
					ID: resourceTypeDoc,
					Operation: []interfaces.ResourceTypeOperation{
						{
							ID: "read",
							Name: []interfaces.OperationName{
								{Language: "zh-cn", Value: "读取"},
							},
						},
					},
				},
			}

			parentDeps := []interfaces.Department{
				{ID: "parent-dep-1", Name: "父部门1"},
			}

			userMgnt.EXPECT().GetUserRolesByUserID(ctx, accessorID).Return(roleTypes, nil)
			tmpDB.EXPECT().GetPagination(ctx, gomock.Any()).Return(1, policies, nil)
			resourceTypeSvc.EXPECT().GetByIDsInternal(ctx, []string{resourceTypeDoc}).Return(resourceTypeInfo, nil)
			userMgnt.EXPECT().GetParentDepartmentsByDepartmentID(ctx, depID).Return(parentDeps, nil)

			_, result, _, err := policy.GetResourcePolicy(ctx, visitor, params)
			assert.Nil(t, err)
			assert.Equal(t, 1, len(result))
			assert.Equal(t, 1, len(result[0].ParentDeps))
			assert.Equal(t, 1, len(result[0].ParentDeps[0]))
			assert.Equal(t, "parent-dep-1", result[0].ParentDeps[0][0].ID)
		})

		Convey("处理根部门访问者", func() {
			policies := []interfaces.PolicyInfo{
				{
					ID:           "policy1",
					ResourceType: resourceTypeDoc,
					ResourceID:   resourceID,
					AccessorID:   rootDepID,
					AccessorType: interfaces.AccessorDepartment,
					Operation: interfaces.PolicyOperation{
						Allow: []interfaces.PolicyOperationItem{{ID: "read"}},
					},
				},
			}

			resourceTypeInfo := map[string]interfaces.ResourceType{
				resourceTypeDoc: {
					ID: resourceTypeDoc,
					Operation: []interfaces.ResourceTypeOperation{
						{
							ID: "read",
							Name: []interfaces.OperationName{
								{Language: "zh-cn", Value: "读取"},
							},
						},
					},
				},
			}

			userMgnt.EXPECT().GetUserRolesByUserID(ctx, accessorID).Return(roleTypes, nil)
			tmpDB.EXPECT().GetPagination(ctx, gomock.Any()).Return(1, policies, nil)
			resourceTypeSvc.EXPECT().GetByIDsInternal(ctx, []string{resourceTypeDoc}).Return(resourceTypeInfo, nil)

			_, result, _, err := policy.GetResourcePolicy(ctx, visitor, params)
			assert.Nil(t, err)
			assert.Equal(t, 1, len(result))
			// 根部门的父部门应该是空数组
			assert.Equal(t, 0, len(result[0].ParentDeps))
		})
	})
}
