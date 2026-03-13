//nolint:govet
package logics

import (
	"context"
	"errors"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"Authorization/common"
	"Authorization/interfaces"
	"Authorization/interfaces/mock"
)

func newPolicyCalc(pdb interfaces.DBPolicyCalc, userMgnt interfaces.DrivenUserMgnt, role interfaces.LogicsRole, resourceType interfaces.LogicsResourceType) *policyCalc {
	return &policyCalc{
		db:           pdb,
		userMgnt:     userMgnt,
		logger:       common.NewLogger(),
		role:         role,
		resourceType: resourceType,
		obligationPriority: map[interfaces.AccessorType]int{
			interfaces.AccessorUser:       1,
			interfaces.AccessorApp:        1,
			interfaces.AccessorRole:       2,
			interfaces.AccessorDepartment: 3,
			interfaces.AccessorGroup:      3,
		},
	}
}

const (
	allResourceID     = "*"
	resourceID        = "resourceID1"
	resourceID2       = "resourceID2"
	resourceID3       = "resourceID3"
	resourceTypeDoc   = "doc"
	resourceTypeMcp   = "mcp"
	accessorID        = "accessorID1"
	accessorID2       = "accessorID2"
	tmpOperation1     = "display"
	tmpOperation2     = "preview"
	tmpOperation3     = "create"
	tmpOperation4     = "authorize"
	obligationTypeID1 = "obligationTypeID1"
	obligationTypeID2 = "obligationTypeID2"
	obligationID1     = "obligationID1"
	obligationID11    = "obligationID11"
	obligationID2     = "obligationID2"
)

func TestPolicyCalcCheck(t *testing.T) {
	Convey("单个检查接口, 请求接口异常情况", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pdb := mock.NewMockDBPolicyCalc(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		pc := newPolicyCalc(pdb, userMgnt, role, resourceType)

		testErr := errors.New("some error")
		ctx := context.Background()
		resource := interfaces.ResourceInfo{
			ID:   resourceID,
			Type: resourceTypeDoc,
		}
		accessor := interfaces.AccessorInfo{
			ID:   accessorID,
			Type: interfaces.RealName,
		}

		policys := []interfaces.PolicyInfo{}
		var outInfo []interfaces.RoleInfo
		includeParams := []interfaces.PolicCalcyIncludeType{}
		Convey("获取用户访问令牌出错", func() {
			userMgnt.EXPECT().GetAccessorIDsByUserID(gomock.Any(), gomock.Any()).Return(nil, testErr)
			operation := []string{tmpOperation1, tmpOperation2}
			_, err := pc.Check(ctx, &resource, &accessor, operation, includeParams)
			assert.Equal(t, err, testErr)
		})

		Convey("获取角色出错", func() {
			userMgnt.EXPECT().GetAccessorIDsByUserID(gomock.Any(), gomock.Any()).Return([]string{accessorID}, nil)
			role.EXPECT().GetRoleByMembers(gomock.Any(), gomock.Any()).Return(outInfo, testErr)
			operation := []string{tmpOperation1, tmpOperation2}
			_, err := pc.Check(ctx, &resource, &accessor, operation, includeParams)
			assert.Equal(t, err, testErr)
		})

		Convey("获取系统角色出错", func() {
			userMgnt.EXPECT().GetAccessorIDsByUserID(gomock.Any(), gomock.Any()).Return([]string{accessorID}, nil)
			role.EXPECT().GetRoleByMembers(gomock.Any(), gomock.Any()).Return(outInfo, nil)
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, testErr)
			operation := []string{tmpOperation1, tmpOperation2}
			_, err := pc.Check(ctx, &resource, &accessor, operation, includeParams)
			assert.Equal(t, err, testErr)
		})

		Convey("获取策略配置出错", func() {
			userMgnt.EXPECT().GetAccessorIDsByUserID(gomock.Any(), gomock.Any()).Return([]string{accessorID}, nil)
			role.EXPECT().GetRoleByMembers(gomock.Any(), gomock.Any()).Return(outInfo, nil)
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			pdb.EXPECT().GetPoliciesByResourcesAndAccessToken(gomock.Any(), gomock.Any(), gomock.Any()).Return(policys, testErr)
			operation := []string{tmpOperation1, tmpOperation2}
			_, err := pc.Check(ctx, &resource, &accessor, operation, includeParams)
			assert.Equal(t, err, testErr)
		})
	})
}

func TestPolicyCalcCheck1(t *testing.T) {
	Convey("单个检查接口, 逻辑计算", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pdb := mock.NewMockDBPolicyCalc(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		pc := newPolicyCalc(pdb, userMgnt, role, resourceType)

		ctx := context.Background()
		resource := interfaces.ResourceInfo{
			ID:   resourceID,
			Type: resourceTypeDoc,
		}
		accessor := interfaces.AccessorInfo{
			ID:   accessorID,
			Type: interfaces.RealName,
		}

		includeParams := []interfaces.PolicCalcyIncludeType{}
		policys := []interfaces.PolicyInfo{}
		var outInfo []interfaces.RoleInfo
		userMgnt.EXPECT().GetAccessorIDsByUserID(gomock.Any(), gomock.Any()).Return([]string{accessorID}, nil)
		role.EXPECT().GetRoleByMembers(gomock.Any(), gomock.Any()).Return(outInfo, nil)
		userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)

		Convey("无策略，check 结果为false", func() {
			pdb.EXPECT().GetPoliciesByResourcesAndAccessToken(gomock.Any(), gomock.Any(), gomock.Any()).Return(policys, nil)
			operation := []string{tmpOperation1, tmpOperation2}
			checkResult, err := pc.Check(ctx, &resource, &accessor, operation, includeParams)
			assert.Equal(t, err, nil)
			assert.Equal(t, checkResult.Result, false)
		})

		Convey("直接配置在本层 允许，check 结果为true", func() {
			policys = []interfaces.PolicyInfo{
				{
					ResourceID: resourceID,
					Rules:      interfaces.PolicyRules{Allow: []interfaces.PolicyRuleItem{{Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation1}}}}},
				},
			}
			pdb.EXPECT().GetPoliciesByResourcesAndAccessToken(gomock.Any(), gomock.Any(), gomock.Any()).Return(policys, nil)
			operation := []string{tmpOperation1}
			checkResult, err := pc.Check(ctx, &resource, &accessor, operation, includeParams)
			assert.Equal(t, err, nil)
			assert.Equal(t, checkResult.Result, true)
		})

		Convey("直接配置在本层 拒绝，check 结果为false", func() {
			policys = []interfaces.PolicyInfo{
				{
					ResourceID: resourceID,
					Rules:      interfaces.PolicyRules{Deny: []interfaces.PolicyRuleItem{{Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation1}}}}},
				},
			}
			pdb.EXPECT().GetPoliciesByResourcesAndAccessToken(gomock.Any(), gomock.Any(), gomock.Any()).Return(policys, nil)
			operation := []string{tmpOperation1}
			checkResult, err := pc.Check(ctx, &resource, &accessor, operation, includeParams)
			assert.Equal(t, err, nil)
			assert.Equal(t, checkResult.Result, false)
		})

		Convey("直接配置在本层 只允许一个操作，check 结果为false", func() {
			policys = []interfaces.PolicyInfo{
				{
					ResourceID: resourceID,
					Rules:      interfaces.PolicyRules{Allow: []interfaces.PolicyRuleItem{{Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation1}}}}},
				},
			}
			pdb.EXPECT().GetPoliciesByResourcesAndAccessToken(gomock.Any(), gomock.Any(), gomock.Any()).Return(policys, nil)
			operation := []string{tmpOperation1, tmpOperation2}
			checkResult, err := pc.Check(ctx, &resource, &accessor, operation, includeParams)
			assert.Equal(t, err, nil)
			assert.Equal(t, checkResult.Result, false)
		})

		Convey("直接配置在上层 允许，check 结果为true", func() {
			policys = []interfaces.PolicyInfo{
				{
					ResourceID: allResourceID,
					Rules:      interfaces.PolicyRules{Allow: []interfaces.PolicyRuleItem{{Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation1}}}}},
				},
			}
			pdb.EXPECT().GetPoliciesByResourcesAndAccessToken(gomock.Any(), gomock.Any(), gomock.Any()).Return(policys, nil)
			operation := []string{tmpOperation1}
			checkResult, err := pc.Check(ctx, &resource, &accessor, operation, includeParams)
			assert.Equal(t, err, nil)
			assert.Equal(t, checkResult.Result, true)
		})

		Convey("直接配置在上一层 拒绝，check 结果为false", func() {
			policys = []interfaces.PolicyInfo{
				{
					ResourceID: allResourceID,
					Rules:      interfaces.PolicyRules{Deny: []interfaces.PolicyRuleItem{{Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation1}}}}},
				},
			}
			pdb.EXPECT().GetPoliciesByResourcesAndAccessToken(gomock.Any(), gomock.Any(), gomock.Any()).Return(policys, nil)
			operation := []string{tmpOperation1}
			checkResult, err := pc.Check(ctx, &resource, &accessor, operation, includeParams)
			assert.Equal(t, err, nil)
			assert.Equal(t, checkResult.Result, false)
		})
	})
}

func TestPolicyCalcCheck2(t *testing.T) {
	Convey("单个检查接口, 继承规则和拒绝优先", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pdb := mock.NewMockDBPolicyCalc(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		pc := newPolicyCalc(pdb, userMgnt, role, resourceType)

		ctx := context.Background()
		resource := interfaces.ResourceInfo{
			ID:   resourceID,
			Type: resourceTypeDoc,
		}
		accessor := interfaces.AccessorInfo{
			ID:   accessorID,
			Type: interfaces.RealName,
		}

		policys := []interfaces.PolicyInfo{}
		var outInfo []interfaces.RoleInfo
		userMgnt.EXPECT().GetAccessorIDsByUserID(gomock.Any(), gomock.Any()).AnyTimes().Return([]string{accessorID}, nil)
		role.EXPECT().GetRoleByMembers(gomock.Any(), gomock.Any()).AnyTimes().Return(outInfo, nil)
		userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).AnyTimes().Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
		// Check 方法内部会调用 createAncestorsResource，需要 mock GetByIDsInternal
		resourceType.EXPECT().GetByIDsInternal(gomock.Any(), []string{resourceTypeDoc}).AnyTimes().Return(map[string]interfaces.ResourceType{
			resourceTypeDoc: {
				ID:         resourceTypeDoc,
				DataStruct: "string",
			},
		}, nil)
		includeParams := []interfaces.PolicCalcyIncludeType{}

		Convey("拒绝优先。直接配置在本层 允许，check 结果为false", func() {
			policys = []interfaces.PolicyInfo{
				{
					ResourceID: resourceID,
					Rules:      interfaces.PolicyRules{Allow: []interfaces.PolicyRuleItem{{Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation1}}}}},
				},
				{
					ResourceID: resourceID,
					Rules:      interfaces.PolicyRules{Deny: []interfaces.PolicyRuleItem{{Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation1}}}}},
				},
			}
			pdb.EXPECT().GetPoliciesByResourcesAndAccessToken(gomock.Any(), gomock.Any(), gomock.Any()).Return(policys, nil)
			operation := []string{tmpOperation1}
			checkResult, err := pc.Check(ctx, &resource, &accessor, operation, includeParams)
			assert.Equal(t, err, nil)
			assert.Equal(t, checkResult.Result, false)
		})

		Convey("就近原则 允许 。不同层级 本层允许显示、上层 拒绝显示，check 结果为true", func() {
			policys = []interfaces.PolicyInfo{
				{
					ResourceID: resourceID,
					Rules:      interfaces.PolicyRules{Allow: []interfaces.PolicyRuleItem{{Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation1}}}}},
				},
				{
					ResourceID: allResourceID,
					Rules:      interfaces.PolicyRules{Deny: []interfaces.PolicyRuleItem{{Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation1}}}}},
				},
			}
			pdb.EXPECT().GetPoliciesByResourcesAndAccessToken(gomock.Any(), gomock.Any(), gomock.Any()).Return(policys, nil)
			operation := []string{tmpOperation1}
			checkResult, err := pc.Check(ctx, &resource, &accessor, operation, includeParams)
			assert.Equal(t, err, nil)
			assert.Equal(t, checkResult.Result, true)
		})

		Convey("就近原则 拒绝 。不同层级 本层拒绝显示、上层 允许显示，check 结果为false", func() {
			policys = []interfaces.PolicyInfo{
				{
					ResourceID: allResourceID,
					Rules:      interfaces.PolicyRules{Allow: []interfaces.PolicyRuleItem{{Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation1}}}}},
				},
				{
					ResourceID: resourceID,
					Rules:      interfaces.PolicyRules{Deny: []interfaces.PolicyRuleItem{{Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation1}}}}},
				},
			}
			pdb.EXPECT().GetPoliciesByResourcesAndAccessToken(gomock.Any(), gomock.Any(), gomock.Any()).Return(policys, nil)
			operation := []string{tmpOperation1}
			checkResult, err := pc.Check(ctx, &resource, &accessor, operation, includeParams)
			assert.Equal(t, err, nil)
			assert.Equal(t, checkResult.Result, false)
		})

		Convey("就近原则 和 拒绝优先 结合 3条策略。本层 一条 允许显示 预览， 一条拒绝预览。上层 允许新建，拒绝显示", func() {
			policys = []interfaces.PolicyInfo{
				{
					ResourceID: allResourceID,
					Rules: interfaces.PolicyRules{
						Allow: []interfaces.PolicyRuleItem{{Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation3}}}},
						Deny:  []interfaces.PolicyRuleItem{{Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation1}}}},
					},
				},
				{
					ResourceID: resourceID,
					Rules: interfaces.PolicyRules{
						Allow: []interfaces.PolicyRuleItem{{Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation1}, {ID: tmpOperation2}}}},
					},
				},
				{
					ResourceID: resourceID,
					Rules: interfaces.PolicyRules{
						Deny: []interfaces.PolicyRuleItem{{Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation2}}}},
					},
				},
			}
			pdb.EXPECT().GetPoliciesByResourcesAndAccessToken(gomock.Any(), gomock.Any(), gomock.Any()).Return(policys, nil)
			// 检查显示 true
			operation := []string{tmpOperation1}
			checkResult, err := pc.Check(ctx, &resource, &accessor, operation, includeParams)
			assert.Equal(t, err, nil)
			assert.Equal(t, checkResult.Result, true)
			// 检查新建 true
			pdb.EXPECT().GetPoliciesByResourcesAndAccessToken(gomock.Any(), gomock.Any(), gomock.Any()).Return(policys, nil)
			operation = []string{tmpOperation3}
			checkResult, err = pc.Check(ctx, &resource, &accessor, operation, includeParams)
			assert.Equal(t, err, nil)
			assert.Equal(t, checkResult.Result, true)
			// 检查预览 false
			pdb.EXPECT().GetPoliciesByResourcesAndAccessToken(gomock.Any(), gomock.Any(), gomock.Any()).Return(policys, nil)
			operation = []string{tmpOperation2}
			checkResult, err = pc.Check(ctx, &resource, &accessor, operation, includeParams)
			assert.Equal(t, err, nil)
			assert.Equal(t, checkResult.Result, false)
		})
	})
}

// jsonlogic 条件用例：data 中为 accessor.id、resource.created_by.id 等，条件为 jsonlogic 规则
func TestPolicyCalcCheckWithJsonLogicCondition(t *testing.T) {
	Convey("单个检查接口, 带 jsonlogic 条件", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pdb := mock.NewMockDBPolicyCalc(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		pc := newPolicyCalc(pdb, userMgnt, role, resourceType)

		ctx := context.Background()
		resource := interfaces.ResourceInfo{
			ID:   resourceID,
			Type: resourceTypeDoc,
		}
		accessor := interfaces.AccessorInfo{
			ID:   accessorID,
			Type: interfaces.RealName,
		}
		var outInfo []interfaces.RoleInfo
		includeParams := []interfaces.PolicCalcyIncludeType{}
		userMgnt.EXPECT().GetAccessorIDsByUserID(gomock.Any(), gomock.Any()).AnyTimes().Return([]string{accessorID}, nil)
		role.EXPECT().GetRoleByMembers(gomock.Any(), gomock.Any()).AnyTimes().Return(outInfo, nil)
		userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).AnyTimes().Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
		resourceType.EXPECT().GetByIDsInternal(gomock.Any(), []string{resourceTypeDoc}).AnyTimes().Return(map[string]interfaces.ResourceType{
			resourceTypeDoc: {ID: resourceTypeDoc, DataStruct: "string"},
		}, nil)

		// jsonlogic: accessor.id == accessorID 时允许 display
		condAccessorEq := map[string]any{
			"==": []any{
				map[string]any{"var": "accessor.id"},
				accessorID,
			},
		}

		Convey("Allow 规则带条件 accessor.id == accessorID，访问者匹配则 check 为 true", func() {
			policys := []interfaces.PolicyInfo{
				{
					ResourceID: resourceID,
					Rules: interfaces.PolicyRules{
						Allow: []interfaces.PolicyRuleItem{{
							Condition:  condAccessorEq,
							Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation1}},
						}},
					},
				},
			}
			pdb.EXPECT().GetPoliciesByResourcesAndAccessToken(gomock.Any(), gomock.Any(), gomock.Any()).Return(policys, nil)
			checkResult, err := pc.Check(ctx, &resource, &accessor, []string{tmpOperation1}, includeParams)
			assert.NoError(t, err)
			assert.True(t, checkResult.Result)
		})

		Convey("Allow 规则带条件 accessor.id == otherUser，当前访问者为 accessorID 故条件不满足，check 为 false", func() {
			condOtherUser := map[string]any{
				"==": []any{map[string]any{"var": "accessor.id"}, "otherUser"},
			}
			policys := []interfaces.PolicyInfo{
				{
					ResourceID: resourceID,
					Rules: interfaces.PolicyRules{
						Allow: []interfaces.PolicyRuleItem{{
							Condition:  condOtherUser,
							Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation1}},
						}},
					},
				},
			}
			pdb.EXPECT().GetPoliciesByResourcesAndAccessToken(gomock.Any(), gomock.Any(), gomock.Any()).Return(policys, nil)
			checkResult, err := pc.Check(ctx, &resource, &accessor, []string{tmpOperation1}, includeParams)
			assert.NoError(t, err)
			assert.False(t, checkResult.Result)
		})

		Convey("Deny 规则带条件 accessor.id == accessorID，访问者匹配则 check 为 false", func() {
			policys := []interfaces.PolicyInfo{
				{
					ResourceID: resourceID,
					Rules: interfaces.PolicyRules{
						Deny: []interfaces.PolicyRuleItem{{
							Condition:  condAccessorEq,
							Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation1}},
						}},
					},
				},
			}
			pdb.EXPECT().GetPoliciesByResourcesAndAccessToken(gomock.Any(), gomock.Any(), gomock.Any()).Return(policys, nil)
			checkResult, err := pc.Check(ctx, &resource, &accessor, []string{tmpOperation1}, includeParams)
			assert.NoError(t, err)
			assert.False(t, checkResult.Result)
		})

		Convey("Allow 条件为 and(accessor.id==x, 常量true)，访问者匹配则 check 为 true", func() {
			condAnd := map[string]any{
				"and": []any{
					map[string]any{"==": []any{map[string]any{"var": "accessor.id"}, accessorID}},
					true,
				},
			}
			policys := []interfaces.PolicyInfo{
				{
					ResourceID: resourceID,
					Rules: interfaces.PolicyRules{
						Allow: []interfaces.PolicyRuleItem{{
							Condition:  condAnd,
							Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation1}},
						}},
					},
				},
			}
			pdb.EXPECT().GetPoliciesByResourcesAndAccessToken(gomock.Any(), gomock.Any(), gomock.Any()).Return(policys, nil)
			checkResult, err := pc.Check(ctx, &resource, &accessor, []string{tmpOperation1}, includeParams)
			assert.NoError(t, err)
			assert.True(t, checkResult.Result)
		})

		Convey("Allow 条件为 resource.created_by.id == creatorID，资源创建者匹配则 check 为 true", func() {
			creatorID := "creator-1"
			resourceWithCreator := interfaces.ResourceInfo{
				ID:        resourceID,
				Type:      resourceTypeDoc,
				CreatedBy: interfaces.CreatedByInfo{ID: creatorID},
			}
			condCreatedBy := map[string]any{
				"==": []any{
					map[string]any{"var": "resource.created_by.id"},
					creatorID,
				},
			}
			policys := []interfaces.PolicyInfo{
				{
					ResourceID: resourceID,
					Rules: interfaces.PolicyRules{
						Allow: []interfaces.PolicyRuleItem{{
							Condition:  condCreatedBy,
							Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation1}},
						}},
					},
				},
			}
			pdb.EXPECT().GetPoliciesByResourcesAndAccessToken(gomock.Any(), gomock.Any(), gomock.Any()).Return(policys, nil)
			checkResult, err := pc.Check(ctx, &resourceWithCreator, &accessor, []string{tmpOperation1}, includeParams)
			assert.NoError(t, err)
			assert.True(t, checkResult.Result)
		})

		Convey("Allow 条件为 resource.created_by.id == creatorID，资源创建者不匹配则 check 为 false", func() {
			resourceWithCreator := interfaces.ResourceInfo{
				ID:        resourceID,
				Type:      resourceTypeDoc,
				CreatedBy: interfaces.CreatedByInfo{ID: "other-creator"},
			}
			condCreatedBy := map[string]any{
				"==": []any{
					map[string]any{"var": "resource.created_by.id"},
					"creator-1",
				},
			}
			policys := []interfaces.PolicyInfo{
				{
					ResourceID: resourceID,
					Rules: interfaces.PolicyRules{
						Allow: []interfaces.PolicyRuleItem{{
							Condition:  condCreatedBy,
							Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation1}},
						}},
					},
				},
			}
			pdb.EXPECT().GetPoliciesByResourcesAndAccessToken(gomock.Any(), gomock.Any(), gomock.Any()).Return(policys, nil)
			checkResult, err := pc.Check(ctx, &resourceWithCreator, &accessor, []string{tmpOperation1}, includeParams)
			assert.NoError(t, err)
			assert.False(t, checkResult.Result)
		})
	})
}

func TestPolicyResourceList(t *testing.T) {
	Convey("资源列表接口, 逻辑计算", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pdb := mock.NewMockDBPolicyCalc(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		pc := newPolicyCalc(pdb, userMgnt, role, resourceType)

		ctx := context.Background()
		accessor := interfaces.AccessorInfo{
			ID:   accessorID,
			Type: interfaces.RealName,
		}

		includeParams := []interfaces.PolicCalcyIncludeType{}
		policys := []interfaces.PolicyInfo{}
		var outInfo []interfaces.RoleInfo
		userMgnt.EXPECT().GetAccessorIDsByUserID(gomock.Any(), gomock.Any()).AnyTimes().Return([]string{accessorID}, nil)
		role.EXPECT().GetRoleByMembers(gomock.Any(), gomock.Any()).AnyTimes().Return(outInfo, nil)
		userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).AnyTimes().Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)

		Convey("无策略，返回空列表", func() {
			pdb.EXPECT().GetPoliciesByResourceTypeAndAccessToken(gomock.Any(), gomock.Any(), gomock.Any()).Return(policys, nil)
			// GetResourceList 内部会调用 createAncestorsResource，需要 mock GetByIDsInternal
			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), []string{resourceTypeDoc}).AnyTimes().Return(map[string]interfaces.ResourceType{
				resourceTypeDoc: {
					ID:         resourceTypeDoc,
					DataStruct: "list",
				},
			}, nil)
			resources, _, err := pc.GetResourceList(ctx, resourceTypeDoc, &accessor, []string{tmpOperation1}, includeParams)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(resources), 0)
		})

		Convey("三个资源ID，只有两个有权限，返回两个", func() {
			policys = []interfaces.PolicyInfo{
				{
					ResourceID: resourceID,
					Rules:      interfaces.PolicyRules{Allow: []interfaces.PolicyRuleItem{{Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation1}}}}},
				},
				{
					ResourceID: resourceID,
					Rules:      interfaces.PolicyRules{Allow: []interfaces.PolicyRuleItem{{Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation2}}}}},
				},
				{
					ResourceID: resourceID2,
					Rules:      interfaces.PolicyRules{Allow: []interfaces.PolicyRuleItem{{Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation1}, {ID: tmpOperation2}}}}},
				},
				{
					ResourceID: resourceID3,
					Rules:      interfaces.PolicyRules{Allow: []interfaces.PolicyRuleItem{{Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation1}}}}},
				},
			}
			pdb.EXPECT().GetPoliciesByResourceTypeAndAccessToken(gomock.Any(), gomock.Any(), gomock.Any()).Return(policys, nil)
			// GetResourceList 内部会调用 createAncestorsResource，需要 mock GetByIDsInternal
			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), []string{resourceTypeDoc}).AnyTimes().Return(map[string]interfaces.ResourceType{
				resourceTypeDoc: {
					ID:         resourceTypeDoc,
					DataStruct: "list",
				},
			}, nil)
			operation := []string{tmpOperation1, tmpOperation2}
			resources, _, err := pc.GetResourceList(ctx, resourceTypeDoc, &accessor, operation, includeParams)
			resultMap := make(map[string]bool)
			for _, resource := range resources {
				resultMap[resource.ID] = true
			}
			assert.Equal(t, err, nil)
			assert.Equal(t, len(resultMap), 2)
			assert.Equal(t, resultMap[resourceID], true)
			assert.Equal(t, resultMap[resourceID2], true)
		})
	})
}

func TestPolicyResourceFliter(t *testing.T) {
	Convey("资源过滤接口, 逻辑计算", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pdb := mock.NewMockDBPolicyCalc(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		pc := newPolicyCalc(pdb, userMgnt, role, resourceType)

		ctx := context.Background()
		accessor := interfaces.AccessorInfo{
			ID:   accessorID,
			Type: interfaces.RealName,
		}

		policys := []interfaces.PolicyInfo{}
		resources := []interfaces.ResourceInfo{
			{
				ID:   resourceID,
				Type: resourceTypeDoc,
			},
			{
				ID:   resourceID2,
				Type: resourceTypeDoc,
			},
			{
				ID:   resourceID3,
				Type: resourceTypeDoc,
			},
		}
		includeParams := []interfaces.PolicCalcyIncludeType{}
		var outInfo []interfaces.RoleInfo
		userMgnt.EXPECT().GetAccessorIDsByUserID(gomock.Any(), gomock.Any()).AnyTimes().Return([]string{accessorID}, nil)
		role.EXPECT().GetRoleByMembers(gomock.Any(), gomock.Any()).AnyTimes().Return(outInfo, nil)
		userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).AnyTimes().Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)

		Convey("无策略，返回空列表", func() {
			pdb.EXPECT().GetPoliciesByResourcesAndAccessToken(gomock.Any(), gomock.Any(), gomock.Any()).Return(policys, nil)
			resources, tmpMap, _, err := pc.ResourceFilter(ctx, resources, &accessor, []string{tmpOperation1}, includeParams)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(resources), 0)
			assert.Equal(t, len(tmpMap), 0)
		})

		policys1 := []interfaces.PolicyInfo{
			{
				ResourceID: resourceID,
				Rules:      interfaces.PolicyRules{Allow: []interfaces.PolicyRuleItem{{Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation1}}}}},
			},
			{
				ResourceID: resourceID,
				Rules:      interfaces.PolicyRules{Allow: []interfaces.PolicyRuleItem{{Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation2}}}}},
			},
			{
				ResourceID: resourceID3,
				Rules:      interfaces.PolicyRules{Allow: []interfaces.PolicyRuleItem{{Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation2}}}}},
			},
			{
				ResourceID: resourceID2,
				Rules:      interfaces.PolicyRules{Allow: []interfaces.PolicyRuleItem{{Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation1}}}}},
			},
			{
				ResourceID: resourceID3,
				Rules:      interfaces.PolicyRules{Allow: []interfaces.PolicyRuleItem{{Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation1}}}}},
			},
		}

		Convey("三个资源ID，只有两个有权限，返回两个", func() {
			pdb.EXPECT().GetPoliciesByResourcesAndAccessToken(gomock.Any(), gomock.Any(), gomock.Any()).Return(policys1, nil)
			resources, tmpMap, _, err := pc.ResourceFilter(ctx, resources, &accessor, []string{tmpOperation1, tmpOperation2}, includeParams)
			resultMap := make(map[string]bool)
			resultMap[resourceID] = true
			resultMap[resourceID3] = true
			assert.Equal(t, err, nil)
			assert.Equal(t, len(resources), 2)
			assert.Equal(t, len(tmpMap), 2)
			assert.Equal(t, resultMap[resources[0].ID], true)
			assert.Equal(t, resultMap[resources[1].ID], true)
		})

		Convey("三个资源ID，三个都有权限，返回三个", func() {
			pdb.EXPECT().GetPoliciesByResourcesAndAccessToken(gomock.Any(), gomock.Any(), gomock.Any()).Return(policys1, nil)
			resources, tmpMap, _, err := pc.ResourceFilter(ctx, resources, &accessor, []string{tmpOperation1}, includeParams)
			resultMap := make(map[string]bool)
			resultMap[resourceID] = true
			resultMap[resourceID2] = true
			resultMap[resourceID3] = true
			assert.Equal(t, err, nil)
			assert.Equal(t, len(resources), 3)
			assert.Equal(t, len(tmpMap), 3)
			assert.Equal(t, resultMap[resources[0].ID], true)
			assert.Equal(t, resultMap[resources[1].ID], true)
			assert.Equal(t, resultMap[resources[2].ID], true)
		})
	})
}

func TestPolicyResourceTypeOperation(t *testing.T) {
	Convey("资源类型操作接口, 接口出错", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pdb := mock.NewMockDBPolicyCalc(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		pc := newPolicyCalc(pdb, userMgnt, role, resourceType)

		ctx := context.Background()
		accessor := interfaces.AccessorInfo{
			ID:   accessorID,
			Type: interfaces.RealName,
		}

		policys := []interfaces.PolicyInfo{}
		var outInfo []interfaces.RoleInfo
		userMgnt.EXPECT().GetAccessorIDsByUserID(gomock.Any(), gomock.Any()).AnyTimes().Return([]string{accessorID}, nil)
		role.EXPECT().GetRoleByMembers(gomock.Any(), gomock.Any()).AnyTimes().Return(outInfo, nil)
		userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).AnyTimes().Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
		testErr := errors.New("some error")
		Convey("获取策略出错", func() {
			pdb.EXPECT().GetPoliciesByResourceTypes(gomock.Any(), gomock.Any(), gomock.Any()).Return(policys, testErr)
			_, err := pc.GetResourceTypeOperation(ctx, []string{resourceTypeDoc}, &accessor)
			assert.Equal(t, err, testErr)
		})

		Convey("获取资源类型信息出错", func() {
			pdb.EXPECT().GetPoliciesByResourceTypes(gomock.Any(), gomock.Any(), gomock.Any()).Return(policys, nil)
			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), gomock.Any()).Return(nil, testErr)
			_, err := pc.GetResourceTypeOperation(ctx, []string{resourceTypeDoc}, &accessor)
			assert.Equal(t, err, testErr)
		})
	})
}

func TestPolicyResourceTypeOperation1(t *testing.T) {
	Convey("资源类型操作接口, 逻辑计算", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pdb := mock.NewMockDBPolicyCalc(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		pc := newPolicyCalc(pdb, userMgnt, role, resourceType)

		ctx := context.Background()
		accessor := interfaces.AccessorInfo{
			ID:   accessorID,
			Type: interfaces.RealName,
		}

		policys := []interfaces.PolicyInfo{}
		resourceTypeMap := map[string]interfaces.ResourceType{
			resourceTypeDoc: {
				ID: resourceTypeDoc,
				Operation: []interfaces.ResourceTypeOperation{
					{
						ID: tmpOperation1,
						Scope: []interfaces.OperationScopeType{
							interfaces.ScopeType,
							interfaces.ScopeInstance,
						},
					},
					{
						ID: tmpOperation2,
						Scope: []interfaces.OperationScopeType{
							interfaces.ScopeType,
						},
					},
					{
						ID: tmpOperation3,
						Scope: []interfaces.OperationScopeType{
							interfaces.ScopeInstance,
						},
					},
				},
			},
			resourceTypeMcp: {
				ID: resourceTypeMcp,
				Operation: []interfaces.ResourceTypeOperation{
					{
						ID: tmpOperation1,
						Scope: []interfaces.OperationScopeType{
							interfaces.ScopeInstance,
						},
					},
					{
						ID: tmpOperation4,
						Scope: []interfaces.OperationScopeType{
							interfaces.ScopeType,
						},
					},
				},
			},
		}
		var outInfo []interfaces.RoleInfo
		userMgnt.EXPECT().GetAccessorIDsByUserID(gomock.Any(), gomock.Any()).AnyTimes().Return([]string{accessorID}, nil)
		role.EXPECT().GetRoleByMembers(gomock.Any(), gomock.Any()).AnyTimes().Return(outInfo, nil)
		userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).AnyTimes().Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
		resourceType.EXPECT().GetByIDsInternal(gomock.Any(), gomock.Any()).AnyTimes().Return(resourceTypeMap, nil)
		Convey("无策略，返回空", func() {
			pdb.EXPECT().GetPoliciesByResourceTypes(gomock.Any(), gomock.Any(), gomock.Any()).Return(policys, nil)
			resourceTypeMap, err := pc.GetResourceTypeOperation(ctx, []string{resourceTypeDoc, resourceTypeMcp}, &accessor)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(resourceTypeMap), 2)
			assert.Equal(t, len(resourceTypeMap[resourceTypeDoc]), 0)
			assert.Equal(t, len(resourceTypeMap[resourceTypeMcp]), 0)
		})

		Convey("有策略，返回资源类型操作", func() {
			policy := []interfaces.PolicyInfo{
				{
					ResourceType: resourceTypeDoc,
					Rules:        interfaces.PolicyRules{Allow: []interfaces.PolicyRuleItem{{Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation1}}}}},
				},
				{
					ResourceType: resourceTypeDoc,
					Rules:        interfaces.PolicyRules{Allow: []interfaces.PolicyRuleItem{{Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation2}}}}},
				},
				{
					ResourceType: resourceTypeMcp,
					Rules:        interfaces.PolicyRules{Allow: []interfaces.PolicyRuleItem{{Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation1}}}}},
				},
				{
					ResourceType: resourceTypeMcp,
					Rules:        interfaces.PolicyRules{Allow: []interfaces.PolicyRuleItem{{Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation4}}}}},
				},
			}
			pdb.EXPECT().GetPoliciesByResourceTypes(gomock.Any(), gomock.Any(), gomock.Any()).Return(policy, nil)
			resourceTypeMap, err := pc.GetResourceTypeOperation(ctx, []string{resourceTypeDoc, resourceTypeMcp}, &accessor)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(resourceTypeMap), 2)
			assert.Equal(t, len(resourceTypeMap[resourceTypeDoc]), 2)
			assert.Equal(t, len(resourceTypeMap[resourceTypeMcp]), 1)
			assert.Equal(t, resourceTypeMap[resourceTypeMcp][0], tmpOperation4)
		})
	})
}

func TestPolicyResourceOperation(t *testing.T) {
	Convey("资源操作接口, 接口出错", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pdb := mock.NewMockDBPolicyCalc(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		pc := newPolicyCalc(pdb, userMgnt, role, resourceType)

		ctx := context.Background()
		accessor := interfaces.AccessorInfo{
			ID:   accessorID,
			Type: interfaces.RealName,
		}

		policys := []interfaces.PolicyInfo{}
		var outInfo []interfaces.RoleInfo
		userMgnt.EXPECT().GetAccessorIDsByUserID(gomock.Any(), gomock.Any()).AnyTimes().Return([]string{accessorID}, nil)
		role.EXPECT().GetRoleByMembers(gomock.Any(), gomock.Any()).AnyTimes().Return(outInfo, nil)
		userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).AnyTimes().Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
		testErr := errors.New("some error")
		resources := []interfaces.ResourceInfo{
			{
				ID:   resourceID,
				Type: resourceTypeDoc,
			},
		}
		Convey("获取策略出错", func() {
			pdb.EXPECT().GetPoliciesByResourcesAndAccessToken(gomock.Any(), gomock.Any(), gomock.Any()).Return(policys, testErr)
			_, _, err := pc.GetResourceOperation(ctx, resources, &accessor)
			assert.Equal(t, err, testErr)
		})

		Convey("获取资源类型信息出错", func() {
			pdb.EXPECT().GetPoliciesByResourcesAndAccessToken(gomock.Any(), gomock.Any(), gomock.Any()).Return(policys, nil)
			// createAncestorsResource 不再调用 GetByIDsInternal，这里只会调用一次 GetByIDsInternal（获取资源类型信息）
			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), []string{resourceTypeDoc}).Return(nil, testErr)
			_, _, err := pc.GetResourceOperation(ctx, resources, &accessor)
			assert.Equal(t, err, testErr)
		})
	})
}

func TestPolicyResourceOperation1(t *testing.T) {
	Convey("资源操作接口, 逻辑计算", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pdb := mock.NewMockDBPolicyCalc(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		pc := newPolicyCalc(pdb, userMgnt, role, resourceType)

		ctx := context.Background()
		accessor := interfaces.AccessorInfo{
			ID:   accessorID,
			Type: interfaces.RealName,
		}

		policys := []interfaces.PolicyInfo{}
		resourceTypeMap := map[string]interfaces.ResourceType{
			resourceTypeDoc: {
				ID: resourceTypeDoc,
				Operation: []interfaces.ResourceTypeOperation{
					{
						ID: tmpOperation1,
						Scope: []interfaces.OperationScopeType{
							interfaces.ScopeType,
							interfaces.ScopeInstance,
						},
					},
					{
						ID: tmpOperation2,
						Scope: []interfaces.OperationScopeType{
							interfaces.ScopeType,
						},
					},
					{
						ID: tmpOperation3,
						Scope: []interfaces.OperationScopeType{
							interfaces.ScopeInstance,
						},
					},
				},
			},
		}
		var outInfo []interfaces.RoleInfo
		userMgnt.EXPECT().GetAccessorIDsByUserID(gomock.Any(), gomock.Any()).AnyTimes().Return([]string{accessorID}, nil)
		role.EXPECT().GetRoleByMembers(gomock.Any(), gomock.Any()).AnyTimes().Return(outInfo, nil)
		userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), gomock.Any()).AnyTimes().Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
		resourceType.EXPECT().GetByIDsInternal(gomock.Any(), gomock.Any()).AnyTimes().Return(resourceTypeMap, nil)
		resources := []interfaces.ResourceInfo{
			{
				ID:   resourceID,
				Type: resourceTypeDoc,
			},
			{
				ID:   allResourceID,
				Type: resourceTypeDoc,
			},
		}
		Convey("无策略，返回空", func() {
			pdb.EXPECT().GetPoliciesByResourcesAndAccessToken(gomock.Any(), gomock.Any(), gomock.Any()).Return(policys, nil)
			// GetResourceOperation 内部会调用 createAncestorsResource，需要 mock GetByIDsInternal
			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), []string{resourceTypeDoc}).AnyTimes().Return(map[string]interfaces.ResourceType{
				resourceTypeDoc: {
					ID:         resourceTypeDoc,
					DataStruct: "list",
				},
			}, nil)
			resourceTypeMap, _, err := pc.GetResourceOperation(ctx, resources, &accessor)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(resourceTypeMap), 2)
			assert.Equal(t, len(resourceTypeMap[resourceID]), 0)
			assert.Equal(t, len(resourceTypeMap[allResourceID]), 0)
		})

		Convey("有策略，返回资源类型操作", func() {
			policy := []interfaces.PolicyInfo{
				{
					ResourceID:   allResourceID,
					ResourceType: resourceTypeDoc,
					Rules:        interfaces.PolicyRules{Allow: []interfaces.PolicyRuleItem{{Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation1}}}}},
				},
				{
					ResourceID:   allResourceID,
					ResourceType: resourceTypeDoc,
					Rules:        interfaces.PolicyRules{Allow: []interfaces.PolicyRuleItem{{Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation2}}}}},
				},
				{
					ResourceID:   allResourceID,
					ResourceType: resourceTypeDoc,
					Rules:        interfaces.PolicyRules{Allow: []interfaces.PolicyRuleItem{{Operations: []interfaces.PolicyOperationItem{{ID: tmpOperation3}}}}},
				},
			}
			pdb.EXPECT().GetPoliciesByResourcesAndAccessToken(gomock.Any(), gomock.Any(), gomock.Any()).Return(policy, nil)
			// GetResourceOperation 内部会调用 createAncestorsResource，需要 mock GetByIDsInternal
			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), []string{resourceTypeDoc}).AnyTimes().Return(map[string]interfaces.ResourceType{
				resourceTypeDoc: {
					ID:         resourceTypeDoc,
					DataStruct: "list",
				},
			}, nil)
			resourceTypeMap, _, err := pc.GetResourceOperation(ctx, resources, &accessor)
			typeMap := make(map[string]bool)
			typeMap[tmpOperation1] = true
			typeMap[tmpOperation2] = true
			assert.Equal(t, err, nil)
			assert.Equal(t, len(resourceTypeMap), 2)
			assert.Equal(t, len(resourceTypeMap[resourceID]), 2)
			assert.Equal(t, len(resourceTypeMap[allResourceID]), 2)
			assert.Equal(t, typeMap[resourceTypeMap[allResourceID][0]], true)
			assert.Equal(t, typeMap[resourceTypeMap[allResourceID][1]], true)
		})
	})
}

func TestCalcObligationWithPriority(t *testing.T) {
	Convey("义务优先级计算接口", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pdb := mock.NewMockDBPolicyCalc(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		obligation := mock.NewMockLogicsObligation(ctrl)
		pc := newPolicyCalc(pdb, userMgnt, role, resourceType)
		pc.obligation = obligation

		ope1 := []policyObligationCalcItem{
			{
				priority: 2,
				PolicyObligationItem: interfaces.PolicyObligationItem{
					TypeID: obligationTypeID1,
					ID:     obligationID1,
				},
			},
			{
				priority: 1,
				PolicyObligationItem: interfaces.PolicyObligationItem{
					TypeID: obligationTypeID1,
					ID:     obligationID2,
				},
			},
			{
				priority: 4,
				PolicyObligationItem: interfaces.PolicyObligationItem{
					TypeID: obligationTypeID1,
					ID:     obligationID11,
				},
			},
		}

		ope2 := []policyObligationCalcItem{
			{
				priority: 1,
				PolicyObligationItem: interfaces.PolicyObligationItem{
					TypeID: obligationTypeID1,
					ID:     obligationID1,
				},
			},
			{
				priority: 2,
				PolicyObligationItem: interfaces.PolicyObligationItem{
					TypeID: obligationTypeID2,
					ID:     obligationID2,
				},
			},
		}

		ope3 := []policyObligationCalcItem{
			{
				priority: 1,
				PolicyObligationItem: interfaces.PolicyObligationItem{
					TypeID: obligationTypeID1,
					ID:     obligationID2,
				},
			},
			{
				priority: 1,
				PolicyObligationItem: interfaces.PolicyObligationItem{
					TypeID: obligationTypeID1,
					ID:     obligationID1,
				},
			},
		}

		ctx := context.Background()
		obligationResps := []interfaces.ObligationInfo{
			{
				ID:     obligationID1,
				TypeID: obligationTypeID1,
			},
			{
				ID:     obligationID2,
				TypeID: obligationTypeID1,
			},
			{
				ID:     obligationID11,
				TypeID: obligationTypeID1,
			},
		}
		obligation.EXPECT().GetByIDSInternal(gomock.Any(), gomock.Any()).AnyTimes().Return(obligationResps, nil)
		Convey("同一操作，一个操作有3个义务，且义务类型相同, 按优先级只返回一个", func() {
			allowMapWithObligation := map[string][]policyObligationCalcItem{
				tmpOperation1: ope1,
			}
			obligationResult := pc.calcObligationWithPriority(ctx, allowMapWithObligation)
			assert.Equal(t, len(obligationResult), 1)
			assert.Equal(t, obligationResult[tmpOperation1][0].TypeID, obligationTypeID1)
			assert.Equal(t, obligationResult[tmpOperation1][0].ID, obligationID2)
		})

		Convey("同一操作，类型义务不同, 返回多个", func() {
			allowMapWithObligation := map[string][]policyObligationCalcItem{
				tmpOperation1: ope1,
				tmpOperation2: ope2,
			}
			obligationResult := pc.calcObligationWithPriority(ctx, allowMapWithObligation)
			assert.Equal(t, len(obligationResult), 2)
			assert.Equal(t, obligationResult[tmpOperation1][0].TypeID, obligationTypeID1)
			assert.Equal(t, obligationResult[tmpOperation1][0].ID, obligationID2)

			assert.Equal(t, len(obligationResult[tmpOperation2]), 2)
			// 多个义务类型时，第一个义务类型是随机的，检查数组中是否包含期望的元素
			obligation2Result := obligationResult[tmpOperation2]
			foundObligation1 := false
			foundObligation2 := false
			for _, item := range obligation2Result {
				if item.TypeID == obligationTypeID1 && item.ID == obligationID1 {
					foundObligation1 = true
				}
				if item.TypeID == obligationTypeID2 && item.ID == obligationID2 {
					foundObligation2 = true
				}
			}
			assert.Equal(t, foundObligation1, true, "应该包含 obligationTypeID1/obligationID1")
			assert.Equal(t, foundObligation2, true, "应该包含 obligationTypeID2/obligationID2")
		})

		Convey("同一操作，类型义务不同, 相同优先级返回出现在最前面的，用于有层级结构的义务", func() {
			allowMapWithObligation := map[string][]policyObligationCalcItem{
				tmpOperation1: ope3,
			}
			obligationResult := pc.calcObligationWithPriority(ctx, allowMapWithObligation)
			assert.Equal(t, len(obligationResult), 1)
			assert.Equal(t, obligationResult[tmpOperation1][0].TypeID, obligationTypeID1)
			assert.Equal(t, obligationResult[tmpOperation1][0].ID, obligationID2)
		})
	})
}

func TestCalcOneResourcePermWithObligation(t *testing.T) {
	Convey("策略配置转换义务信息测试", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pdb := mock.NewMockDBPolicyCalc(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		pc := newPolicyCalc(pdb, userMgnt, role, resourceType)
		policys := []interfaces.PolicyInfo{
			{
				// 所有用户的义务
				AccessorID:   accessorID,
				AccessorType: interfaces.AccessorUser,
				Rules: interfaces.PolicyRules{
					Allow: []interfaces.PolicyRuleItem{
						{
							Operations: []interfaces.PolicyOperationItem{
								{
									ID: tmpOperation1,
									Obligations: []interfaces.PolicyObligationItem{
										{
											TypeID: obligationTypeID1,
											ID:     obligationID11,
										},
									},
								},
							},
						},
					},
				},
			},
			{
				// 所有用户的义务
				AccessorID:   accessorID,
				AccessorType: interfaces.AccessorDepartment,
				Rules: interfaces.PolicyRules{
					Allow: []interfaces.PolicyRuleItem{
						{
							Operations: []interfaces.PolicyOperationItem{
								{
									ID: tmpOperation1,
									Obligations: []interfaces.PolicyObligationItem{
										{
											TypeID: obligationTypeID1,
											ID:     obligationID11,
										},
									},
								},
							},
						},
					},
				},
			},
			{
				// 所有用户的义务
				AccessorID:   rootDepID,
				AccessorType: interfaces.AccessorUser,
				Rules: interfaces.PolicyRules{
					Allow: []interfaces.PolicyRuleItem{
						{
							Operations: []interfaces.PolicyOperationItem{
								{
									ID: tmpOperation1,
									Obligations: []interfaces.PolicyObligationItem{
										{
											TypeID: obligationTypeID1,
											ID:     obligationID1,
										},
									},
								},
							},
						},
					},
				},
			},
		}

		Convey("一个操作3条配置都有义务，用户、部门、所有用户都有义务，检查义务信息的数组", func() {
			permResult := pc.calcOneResourcePerm(resourceID, nil, policys)
			assert.Equal(t, len(permResult.allow), 1)
			obligations := permResult.allow[tmpOperation1]
			assert.Equal(t, len(obligations), 3)
			assert.Equal(t, obligations[0].TypeID, obligationTypeID1)
			assert.Equal(t, obligations[0].ID, obligationID11)
			assert.Equal(t, obligations[0].priority, 1)
			assert.Equal(t, obligations[1].TypeID, obligationTypeID1)
			assert.Equal(t, obligations[1].ID, obligationID11)
			assert.Equal(t, obligations[1].priority, 3)
			assert.Equal(t, obligations[2].TypeID, obligationTypeID1)
			assert.Equal(t, obligations[2].ID, obligationID1)
			assert.Equal(t, obligations[2].priority, 5)
		})
	})
}

//nolint:funlen
func TestCalcOneResourcePermWithObligation1(t *testing.T) {
	Convey("策略配置转换义务信息测试-简单场景", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pdb := mock.NewMockDBPolicyCalc(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		pc := newPolicyCalc(pdb, userMgnt, role, resourceType)

		Convey("一个实例3条策略，只有用户策略配置有义务", func() {
			policys := []interfaces.PolicyInfo{
				{
					// 用户的配置
					AccessorID:   accessorID,
					AccessorType: interfaces.AccessorUser,
					Rules: interfaces.PolicyRules{
						Allow: []interfaces.PolicyRuleItem{
							{
								Operations: []interfaces.PolicyOperationItem{
									{
										ID: tmpOperation1,
										Obligations: []interfaces.PolicyObligationItem{
											{
												TypeID: obligationTypeID1,
												ID:     obligationID11,
											},
										},
									},
								},
							},
						},
					},
				},
				{
					// 部门的配置
					AccessorID:   accessorID,
					AccessorType: interfaces.AccessorDepartment,
					Rules: interfaces.PolicyRules{
						Allow: []interfaces.PolicyRuleItem{
							{
								Operations: []interfaces.PolicyOperationItem{
									{
										ID:          tmpOperation1,
										Obligations: []interfaces.PolicyObligationItem{},
									},
								},
							},
						},
					},
				},
				{
					// 所有用户
					AccessorID:   rootDepID,
					AccessorType: interfaces.AccessorUser,
					Rules: interfaces.PolicyRules{
						Allow: []interfaces.PolicyRuleItem{
							{
								Operations: []interfaces.PolicyOperationItem{
									{
										ID:          tmpOperation1,
										Obligations: []interfaces.PolicyObligationItem{},
									},
								},
							},
						},
					},
				},
			}
			permResult := pc.calcOneResourcePerm(resourceID, nil, policys)
			assert.Equal(t, len(permResult.allow), 1)
			obligations := permResult.allow[tmpOperation1]
			assert.Equal(t, len(obligations), 1)
			assert.Equal(t, obligations[0].TypeID, obligationTypeID1)
			assert.Equal(t, obligations[0].ID, obligationID11)
			assert.Equal(t, obligations[0].priority, 1)
		})

		Convey("一个实例3条策略，只有部门策略配置有义务", func() {
			policys := []interfaces.PolicyInfo{
				{
					// 用户的配置
					AccessorID:   accessorID,
					AccessorType: interfaces.AccessorUser,
					Rules: interfaces.PolicyRules{
						Allow: []interfaces.PolicyRuleItem{
							{
								Operations: []interfaces.PolicyOperationItem{
									{
										ID:          tmpOperation1,
										Obligations: []interfaces.PolicyObligationItem{},
									},
								},
							},
						},
					},
				},
				{
					// 部门的配置
					AccessorID:   accessorID,
					AccessorType: interfaces.AccessorDepartment,
					Rules: interfaces.PolicyRules{
						Allow: []interfaces.PolicyRuleItem{
							{
								Operations: []interfaces.PolicyOperationItem{
									{
										ID: tmpOperation1,
										Obligations: []interfaces.PolicyObligationItem{
											{
												TypeID: obligationTypeID1,
												ID:     obligationID11,
											},
										},
									},
								},
							},
						},
					},
				},
				{
					// 所有用户
					AccessorID:   rootDepID,
					AccessorType: interfaces.AccessorUser,
					Rules: interfaces.PolicyRules{
						Allow: []interfaces.PolicyRuleItem{
							{
								Operations: []interfaces.PolicyOperationItem{
									{
										ID:          tmpOperation1,
										Obligations: []interfaces.PolicyObligationItem{},
									},
								},
							},
						},
					},
				},
			}
			permResult := pc.calcOneResourcePerm(resourceID, nil, policys)
			assert.Equal(t, len(permResult.allow), 1)
			obligations := permResult.allow[tmpOperation1]
			assert.Equal(t, len(obligations), 1)
			assert.Equal(t, obligations[0].TypeID, obligationTypeID1)
			assert.Equal(t, obligations[0].ID, obligationID11)
			assert.Equal(t, obligations[0].priority, pc.obligationPriority[interfaces.AccessorDepartment])
		})
		Convey("一个实例3条策略，只有所有用户策略配置有义务", func() {
			policys := []interfaces.PolicyInfo{
				{
					// 用户的配置
					AccessorID:   accessorID,
					AccessorType: interfaces.AccessorUser,
					Rules: interfaces.PolicyRules{
						Allow: []interfaces.PolicyRuleItem{
							{
								Operations: []interfaces.PolicyOperationItem{
									{
										ID:          tmpOperation1,
										Obligations: []interfaces.PolicyObligationItem{},
									},
								},
							},
						},
					},
				},
				{
					// 部门的配置
					AccessorID:   accessorID,
					AccessorType: interfaces.AccessorDepartment,
					Rules: interfaces.PolicyRules{
						Allow: []interfaces.PolicyRuleItem{
							{
								Operations: []interfaces.PolicyOperationItem{
									{
										ID:          tmpOperation1,
										Obligations: []interfaces.PolicyObligationItem{},
									},
								},
							},
						},
					},
				},
				{
					// 所有用户
					AccessorID:   rootDepID,
					AccessorType: interfaces.AccessorUser,
					Rules: interfaces.PolicyRules{
						Allow: []interfaces.PolicyRuleItem{
							{
								Operations: []interfaces.PolicyOperationItem{
									{
										ID: tmpOperation1,
										Obligations: []interfaces.PolicyObligationItem{
											{
												TypeID: obligationTypeID1,
												ID:     obligationID11,
											},
										},
									},
								},
							},
						},
					},
				},
			}
			permResult := pc.calcOneResourcePerm(resourceID, nil, policys)
			assert.Equal(t, len(permResult.allow), 1)
			obligations := permResult.allow[tmpOperation1]
			assert.Equal(t, len(obligations), 1)
			assert.Equal(t, obligations[0].TypeID, obligationTypeID1)
			assert.Equal(t, obligations[0].ID, obligationID11)
			assert.Equal(t, obligations[0].priority, rootDepObligationPriority)
		})
	})
}

func TestCalcOneResourcePermWithObligation2(t *testing.T) {
	Convey("策略配置转换义务信息测试-简单场景2", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pdb := mock.NewMockDBPolicyCalc(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		pc := newPolicyCalc(pdb, userMgnt, role, resourceType)

		Convey("一个实例3条策略，只有部门策略配置有义务，义务有两条，一条有ID，一条没有ID", func() {
			policys := []interfaces.PolicyInfo{
				{
					// 用户的配置
					AccessorID:   accessorID,
					AccessorType: interfaces.AccessorUser,
					Rules: interfaces.PolicyRules{
						Allow: []interfaces.PolicyRuleItem{
							{
								Operations: []interfaces.PolicyOperationItem{
									{
										ID:          tmpOperation1,
										Obligations: []interfaces.PolicyObligationItem{},
									},
								},
							},
						},
					},
				},
				{
					// 部门的配置
					AccessorID:   accessorID,
					AccessorType: interfaces.AccessorDepartment,
					Rules: interfaces.PolicyRules{
						Allow: []interfaces.PolicyRuleItem{
							{
								Operations: []interfaces.PolicyOperationItem{
									{
										ID: tmpOperation1,
										Obligations: []interfaces.PolicyObligationItem{
											{
												TypeID: obligationTypeID1,
												ID:     obligationID11,
											},
											{
												TypeID: obligationTypeID2,
											},
										},
									},
								},
							},
						},
					},
				},
				{
					// 所有用户
					AccessorID:   rootDepID,
					AccessorType: interfaces.AccessorUser,
					Rules: interfaces.PolicyRules{
						Allow: []interfaces.PolicyRuleItem{
							{
								Operations: []interfaces.PolicyOperationItem{
									{
										ID:          tmpOperation1,
										Obligations: []interfaces.PolicyObligationItem{},
									},
								},
							},
						},
					},
				},
			}
			permResult := pc.calcOneResourcePerm(resourceID, nil, policys)
			assert.Equal(t, len(permResult.allow), 1)
			obligations := permResult.allow[tmpOperation1]
			assert.Equal(t, len(obligations), 2)
			assert.Equal(t, obligations[0].TypeID, obligationTypeID1)
			assert.Equal(t, obligations[0].ID, obligationID11)
			assert.Equal(t, obligations[0].priority, pc.obligationPriority[interfaces.AccessorDepartment])
			assert.Equal(t, obligations[1].TypeID, obligationTypeID2)
			assert.Equal(t, obligations[1].ID, "")
			assert.Equal(t, obligations[1].priority, pc.obligationPriority[interfaces.AccessorDepartment])
		})
	})
}

func TestCalcResourceInheritedOperation(t *testing.T) {
	Convey("带有层级的策略配置转换义务信息测试", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		pdb := mock.NewMockDBPolicyCalc(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		role := mock.NewMockLogicsRole(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		pc := newPolicyCalc(pdb, userMgnt, role, resourceType)

		ope1 := []policyObligationCalcItem{
			{
				priority: 2,
				PolicyObligationItem: interfaces.PolicyObligationItem{
					TypeID: obligationTypeID1,
					ID:     obligationID1,
				},
			},
			{
				priority: 1,
				PolicyObligationItem: interfaces.PolicyObligationItem{
					TypeID: obligationTypeID2,
					ID:     obligationID1,
				},
			},
			{
				priority: 4,
				PolicyObligationItem: interfaces.PolicyObligationItem{
					TypeID: obligationTypeID1,
					ID:     obligationID11,
				},
			},
		}

		ope2 := []policyObligationCalcItem{
			{
				priority: 1,
				PolicyObligationItem: interfaces.PolicyObligationItem{
					TypeID: obligationTypeID1,
					ID:     obligationID2,
				},
			},
		}

		resourcePermMap := map[string]resourcePerm{
			resourceID: {
				allow: map[string][]policyObligationCalcItem{
					tmpOperation1: ope1,
				},
			},
			"*": {
				allow: map[string][]policyObligationCalcItem{
					tmpOperation1: ope2,
				},
			},
		}

		resource := interfaces.ResourceInfo{
			ID: resourceID,
		}
		Convey("一个操作3条配置都有义务，都有义务，相同义务类型按照就近原则顺序收集", func() {
			_, _, OpaObligationsMap := pc.calcResourceInheritedOperation([]interfaces.ResourceInfo{resource}, resourcePermMap)
			assert.Equal(t, len(OpaObligationsMap[tmpOperation1]), 4)
			assert.Equal(t, OpaObligationsMap[tmpOperation1][0].TypeID, obligationTypeID1)
			assert.Equal(t, OpaObligationsMap[tmpOperation1][0].ID, obligationID1)
			assert.Equal(t, OpaObligationsMap[tmpOperation1][3].TypeID, obligationTypeID1)
			assert.Equal(t, OpaObligationsMap[tmpOperation1][3].ID, obligationID2)
		})
	})
}
