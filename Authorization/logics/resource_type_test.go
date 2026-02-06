//nolint:gocritic
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

func newResourceType(db interfaces.DBResourceType, userMgmt interfaces.DrivenUserMgnt, resourceTypeHierarchy interfaces.LogicsResourceTypeHierarchy) *resourceType {
	return &resourceType{
		db:                    db,
		userMgmt:              userMgmt,
		resourceTypeHierarchy: resourceTypeHierarchy,
		logger:                common.NewLogger(),
	}
}

const (
	testResourceTypeID  = "test_resource_type"
	testResourceTypeID2 = "test_resource_type_2"
	testUserID          = "test_user_id"
)

func TestResourceType_GetPagination(t *testing.T) {
	Convey("测试GetPagination方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBResourceType(ctrl)
		userMgmt := mock.NewMockDrivenUserMgnt(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		rt := newResourceType(db, userMgmt, resourceTypeHierarchy)

		ctx := context.Background()
		visitor := &interfaces.Visitor{
			ID:   testUserID,
			Type: interfaces.RealName,
		}
		params := interfaces.ResourceTypePagination{
			Offset: 0,
			Limit:  10,
		}

		Convey("权限检查失败", func() {
			userMgmt.EXPECT().GetUserRolesByUserID(gomock.Any(), testUserID).Return(nil, errors.New("权限检查失败"))

			count, resources, err := rt.GetPagination(ctx, visitor, params)

			assert.Error(t, err)
			assert.Equal(t, 0, count)
			assert.Nil(t, resources)
		})

		Convey("数据库查询失败", func() {
			userMgmt.EXPECT().GetUserRolesByUserID(gomock.Any(), testUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			db.EXPECT().GetPagination(gomock.Any(), params).Return(0, nil, errors.New("数据库错误"))

			count, resources, err := rt.GetPagination(ctx, visitor, params)

			assert.Error(t, err)
			assert.Equal(t, 0, count)
			assert.Nil(t, resources)
		})

		Convey("成功获取资源类型列表", func() {
			expectedResources := []interfaces.ResourceType{
				{
					ID:          testResourceTypeID,
					Name:        "测试资源类型",
					Description: "测试描述",
				},
			}

			userMgmt.EXPECT().GetUserRolesByUserID(gomock.Any(), testUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			db.EXPECT().GetPagination(gomock.Any(), params).Return(1, expectedResources, nil)

			count, resources, err := rt.GetPagination(ctx, visitor, params)

			assert.NoError(t, err)
			assert.Equal(t, 1, count)
			assert.Equal(t, expectedResources, resources)
		})
	})
}

func TestResourceType_Set(t *testing.T) {
	Convey("测试Set方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBResourceType(ctrl)
		userMgmt := mock.NewMockDrivenUserMgnt(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		rt := newResourceType(db, userMgmt, resourceTypeHierarchy)

		ctx := context.Background()
		visitor := &interfaces.Visitor{
			ID:   testUserID,
			Type: interfaces.RealName,
		}

		Convey("权限检查失败", func() {
			userMgmt.EXPECT().GetUserRolesByUserID(gomock.Any(), testUserID).Return(nil, errors.New("权限检查失败"))

			resourceType := interfaces.ResourceType{ID: testResourceTypeID}
			err := rt.Set(ctx, visitor, &resourceType)

			assert.Error(t, err)
		})

		Convey("ID长度超过50", func() {
			userMgmt.EXPECT().GetUserRolesByUserID(gomock.Any(), testUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)

			resourceType := interfaces.ResourceType{ID: "this_is_a_very_long_resource_type_id_that_exceeds_fifty_characters"}
			err := rt.Set(ctx, visitor, &resourceType)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "id length must less than 50")
		})

		Convey("ID包含非法字符", func() {
			userMgmt.EXPECT().GetUserRolesByUserID(gomock.Any(), testUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)

			resourceType := interfaces.ResourceType{ID: "test-resource-type"}
			err := rt.Set(ctx, visitor, &resourceType)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid id, only number or letter or underline")
		})

		Convey("数据库保存失败", func() {
			userMgmt.EXPECT().GetUserRolesByUserID(gomock.Any(), testUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			db.EXPECT().Set(gomock.Any(), gomock.Any()).Return(errors.New("数据库保存失败"))

			resourceType := interfaces.ResourceType{ID: testResourceTypeID}
			err := rt.Set(ctx, visitor, &resourceType)

			assert.Error(t, err)
		})

		Convey("成功保存资源类型", func() {
			userMgmt.EXPECT().GetUserRolesByUserID(gomock.Any(), testUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			db.EXPECT().Set(gomock.Any(), gomock.Any()).Return(nil)

			resourceType := interfaces.ResourceType{ID: testResourceTypeID}
			err := rt.Set(ctx, visitor, &resourceType)

			assert.NoError(t, err)
		})
	})
}

//nolint:dupl
func TestResourceType_Delete(t *testing.T) {
	Convey("测试Delete方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBResourceType(ctrl)
		userMgmt := mock.NewMockDrivenUserMgnt(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		rt := newResourceType(db, userMgmt, resourceTypeHierarchy)

		ctx := context.Background()
		visitor := &interfaces.Visitor{
			ID:   testUserID,
			Type: interfaces.RealName,
		}

		Convey("权限检查失败", func() {
			userMgmt.EXPECT().GetUserRolesByUserID(gomock.Any(), testUserID).Return(nil, errors.New("权限检查失败"))

			err := rt.Delete(ctx, visitor, testResourceTypeID)

			assert.Error(t, err)
		})

		Convey("数据库删除失败", func() {
			userMgmt.EXPECT().GetUserRolesByUserID(gomock.Any(), testUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			db.EXPECT().Delete(gomock.Any(), testResourceTypeID).Return(errors.New("数据库删除失败"))

			err := rt.Delete(ctx, visitor, testResourceTypeID)

			assert.Error(t, err)
		})

		Convey("成功删除资源类型", func() {
			userMgmt.EXPECT().GetUserRolesByUserID(gomock.Any(), testUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			db.EXPECT().Delete(gomock.Any(), testResourceTypeID).Return(nil)

			err := rt.Delete(ctx, visitor, testResourceTypeID)

			assert.NoError(t, err)
		})
	})
}

func TestResourceType_GetByID(t *testing.T) {
	Convey("测试GetByID方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBResourceType(ctrl)
		userMgmt := mock.NewMockDrivenUserMgnt(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		rt := newResourceType(db, userMgmt, resourceTypeHierarchy)

		ctx := context.Background()
		visitor := &interfaces.Visitor{
			ID:   testUserID,
			Type: interfaces.RealName,
		}

		Convey("数据库查询失败", func() {
			db.EXPECT().GetByIDs(gomock.Any(), []string{testResourceTypeID}).Return(nil, errors.New("数据库查询失败"))

			resourceType, err := rt.GetByID(ctx, visitor, testResourceTypeID)

			assert.Error(t, err)
			assert.Equal(t, interfaces.ResourceType{}, resourceType)
		})

		Convey("资源类型不存在", func() {
			db.EXPECT().GetByIDs(gomock.Any(), []string{testResourceTypeID}).Return(map[string]interfaces.ResourceType{}, nil)

			resourceType, err := rt.GetByID(ctx, visitor, testResourceTypeID)

			assert.NoError(t, err)
			assert.Equal(t, interfaces.ResourceType{}, resourceType)
		})

		Convey("成功获取资源类型", func() {
			expectedResourceType := interfaces.ResourceType{
				ID:          testResourceTypeID,
				Name:        "测试资源类型",
				Description: "测试描述",
			}

			db.EXPECT().GetByIDs(gomock.Any(), []string{testResourceTypeID}).Return(map[string]interfaces.ResourceType{
				testResourceTypeID: expectedResourceType,
			}, nil)

			resourceType, err := rt.GetByID(ctx, visitor, testResourceTypeID)

			assert.NoError(t, err)
			assert.Equal(t, expectedResourceType, resourceType)
		})
	})
}

//nolint:funlen
func TestResourceType_GetAllOperation(t *testing.T) {
	Convey("测试GetAllOperation方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBResourceType(ctrl)
		userMgmt := mock.NewMockDrivenUserMgnt(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		rt := newResourceType(db, userMgmt, resourceTypeHierarchy)

		ctx := context.Background()
		visitor := &interfaces.Visitor{
			ID:       testUserID,
			Type:     interfaces.RealName,
			Language: "zh-cn",
		}

		Convey("数据库查询失败", func() {
			db.EXPECT().GetAllInternalWithHidden(gomock.Any()).Return(nil, errors.New("数据库查询失败"))

			operations, childrenOperations, err := rt.GetAllOperation(ctx, visitor, testResourceTypeID, interfaces.ScopeType)

			assert.Error(t, err)
			assert.Nil(t, operations)
			assert.Nil(t, childrenOperations)
		})

		Convey("资源类型不存在", func() {
			db.EXPECT().GetAllInternalWithHidden(gomock.Any()).Return([]interfaces.ResourceType{}, nil)

			operations, childrenOperations, err := rt.GetAllOperation(ctx, visitor, testResourceTypeID, interfaces.ScopeType)

			assert.NoError(t, err)
			assert.Nil(t, operations)
			assert.Nil(t, childrenOperations)
		})

		Convey("scope为ScopeType时直接返回操作列表", func() {
			resourceType := interfaces.ResourceType{
				ID:   testResourceTypeID,
				Name: "测试资源类型",
				Operation: []interfaces.ResourceTypeOperation{
					{
						ID:          "display",
						Description: "显示操作",
						Scope:       []interfaces.OperationScopeType{interfaces.ScopeType},
						Name: []interfaces.OperationName{
							{Language: "zh-cn", Value: "显示"},
							{Language: "en-us", Value: "Display"},
						},
					},
					{
						ID:          "create",
						Description: "创建操作",
						Scope:       []interfaces.OperationScopeType{interfaces.ScopeInstance},
						Name: []interfaces.OperationName{
							{Language: "zh-cn", Value: "创建"},
							{Language: "en-us", Value: "Create"},
						},
					},
				},
			}

			db.EXPECT().GetAllInternalWithHidden(gomock.Any()).Return([]interfaces.ResourceType{resourceType}, nil)

			operations, childrenOperations, err := rt.GetAllOperation(ctx, visitor, testResourceTypeID, interfaces.ScopeType)

			assert.NoError(t, err)
			assert.Len(t, operations, 1)
			assert.Equal(t, "display", operations[0].ID)
			assert.Equal(t, "显示", operations[0].Name)
			assert.Nil(t, childrenOperations)
		})

		Convey("scope为ScopeInstance且没有子资源类型", func() {
			resourceType := interfaces.ResourceType{
				ID:   testResourceTypeID,
				Name: "测试资源类型",
				Operation: []interfaces.ResourceTypeOperation{
					{
						ID:          "create",
						Description: "创建操作",
						Scope:       []interfaces.OperationScopeType{interfaces.ScopeInstance},
						Name: []interfaces.OperationName{
							{Language: "zh-cn", Value: "创建"},
						},
					},
				},
			}

			emptyHierarchy := interfaces.ResourceTypeHierarchy{
				ResourceTypeID: testResourceTypeID,
				Children:       []interfaces.ResourceTypeHierarchy{},
			}

			db.EXPECT().GetAllInternalWithHidden(gomock.Any()).Return([]interfaces.ResourceType{resourceType}, nil)
			resourceTypeHierarchy.EXPECT().Get(gomock.Any(), visitor, testResourceTypeID).Return(emptyHierarchy, nil)

			operations, childrenOperations, err := rt.GetAllOperation(ctx, visitor, testResourceTypeID, interfaces.ScopeInstance)

			assert.NoError(t, err)
			assert.Len(t, operations, 1)
			assert.Equal(t, "create", operations[0].ID)
			assert.Nil(t, childrenOperations)
		})

		Convey("scope为ScopeInstance且有子资源类型", func() {
			childResourceTypeID := "child_resource_type"
			resourceType := interfaces.ResourceType{
				ID:   testResourceTypeID,
				Name: "测试资源类型",
				Operation: []interfaces.ResourceTypeOperation{
					{
						ID:          "create",
						Description: "创建操作",
						Scope:       []interfaces.OperationScopeType{interfaces.ScopeInstance},
						Name: []interfaces.OperationName{
							{Language: "zh-cn", Value: "创建"},
						},
					},
				},
			}

			childResourceType := interfaces.ResourceType{
				ID:   childResourceTypeID,
				Name: "子资源类型",
				Operation: []interfaces.ResourceTypeOperation{
					{
						ID:          "view",
						Description: "查看操作",
						Scope:       []interfaces.OperationScopeType{interfaces.ScopeType},
						Name: []interfaces.OperationName{
							{Language: "zh-cn", Value: "查看"},
						},
					},
				},
			}

			hierarchy := interfaces.ResourceTypeHierarchy{
				ResourceTypeID: testResourceTypeID,
				Children: []interfaces.ResourceTypeHierarchy{
					{
						ResourceTypeID: childResourceTypeID,
						Children:       []interfaces.ResourceTypeHierarchy{},
					},
				},
			}

			db.EXPECT().GetAllInternalWithHidden(gomock.Any()).Return([]interfaces.ResourceType{resourceType, childResourceType}, nil)
			resourceTypeHierarchy.EXPECT().Get(gomock.Any(), visitor, testResourceTypeID).Return(hierarchy, nil)

			operations, childrenOperations, err := rt.GetAllOperation(ctx, visitor, testResourceTypeID, interfaces.ScopeInstance)

			assert.NoError(t, err)
			assert.Len(t, operations, 1)
			assert.Equal(t, "create", operations[0].ID)
			assert.Len(t, childrenOperations, 1)
			assert.Equal(t, childResourceTypeID, childrenOperations[0].ResourceTypeID)
			assert.Equal(t, "子资源类型", childrenOperations[0].Name)
			assert.Len(t, childrenOperations[0].Operations, 1)
			assert.Equal(t, "view", childrenOperations[0].Operations[0].ID)
		})

		Convey("获取层级关系失败", func() {
			resourceType := interfaces.ResourceType{
				ID:   testResourceTypeID,
				Name: "测试资源类型",
				Operation: []interfaces.ResourceTypeOperation{
					{
						ID:          "create",
						Description: "创建操作",
						Scope:       []interfaces.OperationScopeType{interfaces.ScopeInstance},
						Name: []interfaces.OperationName{
							{Language: "zh-cn", Value: "创建"},
						},
					},
				},
			}

			db.EXPECT().GetAllInternalWithHidden(gomock.Any()).Return([]interfaces.ResourceType{resourceType}, nil)
			resourceTypeHierarchy.EXPECT().Get(gomock.Any(), visitor, testResourceTypeID).Return(interfaces.ResourceTypeHierarchy{}, errors.New("获取层级关系失败"))

			operations, childrenOperations, err := rt.GetAllOperation(ctx, visitor, testResourceTypeID, interfaces.ScopeInstance)

			assert.Error(t, err)
			// 即使获取层级关系失败，当前资源类型的操作信息已经获取到了
			assert.NotNil(t, operations)
			assert.Len(t, operations, 1)
			assert.Equal(t, "create", operations[0].ID)
			assert.Nil(t, childrenOperations)
		})

		Convey("子资源类型不存在时跳过", func() {
			resourceType := interfaces.ResourceType{
				ID:   testResourceTypeID,
				Name: "测试资源类型",
				Operation: []interfaces.ResourceTypeOperation{
					{
						ID:          "create",
						Description: "创建操作",
						Scope:       []interfaces.OperationScopeType{interfaces.ScopeInstance},
						Name: []interfaces.OperationName{
							{Language: "zh-cn", Value: "创建"},
						},
					},
				},
			}

			nonExistentChildID := "non_existent_child"
			hierarchy := interfaces.ResourceTypeHierarchy{
				ResourceTypeID: testResourceTypeID,
				Children: []interfaces.ResourceTypeHierarchy{
					{
						ResourceTypeID: nonExistentChildID,
						Children:       []interfaces.ResourceTypeHierarchy{},
					},
				},
			}

			db.EXPECT().GetAllInternalWithHidden(gomock.Any()).Return([]interfaces.ResourceType{resourceType}, nil)
			resourceTypeHierarchy.EXPECT().Get(gomock.Any(), visitor, testResourceTypeID).Return(hierarchy, nil)

			operations, childrenOperations, err := rt.GetAllOperation(ctx, visitor, testResourceTypeID, interfaces.ScopeInstance)

			assert.NoError(t, err)
			assert.Len(t, operations, 1)
			assert.Equal(t, "create", operations[0].ID)
			assert.Empty(t, childrenOperations)
		})
	})
}

func TestResourceType_GetByIDsInternal(t *testing.T) {
	Convey("测试GetByIDsInternal方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBResourceType(ctrl)
		userMgmt := mock.NewMockDrivenUserMgnt(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		rt := newResourceType(db, userMgmt, resourceTypeHierarchy)

		ctx := context.Background()
		resourceTypeIDs := []string{testResourceTypeID, testResourceTypeID2}

		Convey("数据库查询失败", func() {
			db.EXPECT().GetByIDs(gomock.Any(), resourceTypeIDs).Return(nil, errors.New("数据库查询失败"))

			resourceMap, err := rt.GetByIDsInternal(ctx, resourceTypeIDs)

			assert.Error(t, err)
			assert.Nil(t, resourceMap)
		})

		Convey("成功获取资源类型映射", func() {
			expectedResourceMap := map[string]interfaces.ResourceType{
				testResourceTypeID: {
					ID:          testResourceTypeID,
					Name:        "测试资源类型1",
					Description: "测试描述1",
				},
				testResourceTypeID2: {
					ID:          testResourceTypeID2,
					Name:        "测试资源类型2",
					Description: "测试描述2",
				},
			}

			db.EXPECT().GetByIDs(gomock.Any(), resourceTypeIDs).Return(expectedResourceMap, nil)

			resourceMap, err := rt.GetByIDsInternal(ctx, resourceTypeIDs)

			assert.NoError(t, err)
			assert.Equal(t, expectedResourceMap, resourceMap)
		})
	})
}

func TestResourceType_InitResourceTypes(t *testing.T) {
	Convey("测试InitResourceTypes方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBResourceType(ctrl)
		userMgmt := mock.NewMockDrivenUserMgnt(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		rt := newResourceType(db, userMgmt, resourceTypeHierarchy)

		ctx := context.Background()
		resourceTypes := []interfaces.ResourceType{
			{
				ID:          testResourceTypeID,
				Name:        "测试资源类型1",
				Description: "测试描述1",
			},
			{
				ID:          testResourceTypeID2,
				Name:        "测试资源类型2",
				Description: "测试描述2",
			},
		}

		Convey("获取现有资源类型失败", func() {
			db.EXPECT().GetByIDs(gomock.Any(), []string{testResourceTypeID}).Return(nil, errors.New("数据库查询失败"))

			err := rt.InitResourceTypes(ctx, resourceTypes)

			assert.Error(t, err)
		})

		Convey("保存新资源类型失败", func() {
			db.EXPECT().GetByIDs(gomock.Any(), []string{testResourceTypeID}).Return(map[string]interfaces.ResourceType{}, nil)
			db.EXPECT().Set(gomock.Any(), gomock.Any()).Return(errors.New("保存失败"))

			err := rt.InitResourceTypes(ctx, resourceTypes)

			assert.Error(t, err)
		})

		Convey("资源类型已存在且无变化", func() {
			existingResourceType := interfaces.ResourceType{
				ID:          testResourceTypeID,
				Name:        "测试资源类型1",
				Description: "测试描述1",
			}

			db.EXPECT().GetByIDs(gomock.Any(), []string{testResourceTypeID}).Return(map[string]interfaces.ResourceType{
				testResourceTypeID: existingResourceType,
			}, nil)
			// 由于无变化，不会调用Set方法

			err := rt.InitResourceTypes(ctx, resourceTypes[:1])

			assert.NoError(t, err)
		})

		Convey("资源类型已存在但有变化", func() {
			existingResourceType := interfaces.ResourceType{
				ID:          testResourceTypeID,
				Name:        "旧名称",
				Description: "旧描述",
			}

			db.EXPECT().GetByIDs(gomock.Any(), []string{testResourceTypeID}).Return(map[string]interfaces.ResourceType{
				testResourceTypeID: existingResourceType,
			}, nil)
			db.EXPECT().Set(gomock.Any(), gomock.Any()).Return(nil)

			err := rt.InitResourceTypes(ctx, resourceTypes[:1])

			assert.NoError(t, err)
		})

		Convey("成功初始化资源类型", func() {
			db.EXPECT().GetByIDs(gomock.Any(), []string{testResourceTypeID}).Return(map[string]interfaces.ResourceType{}, nil)
			db.EXPECT().Set(gomock.Any(), gomock.Any()).Return(nil)
			db.EXPECT().GetByIDs(gomock.Any(), []string{testResourceTypeID2}).Return(map[string]interfaces.ResourceType{}, nil)
			db.EXPECT().Set(gomock.Any(), gomock.Any()).Return(nil)

			err := rt.InitResourceTypes(ctx, resourceTypes)

			assert.NoError(t, err)
		})
	})
}

func TestResourceType_checkResourceTypeChange(t *testing.T) {
	Convey("测试checkResourceTypeChange方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBResourceType(ctrl)
		userMgmt := mock.NewMockDrivenUserMgnt(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		rt := newResourceType(db, userMgmt, resourceTypeHierarchy)

		Convey("名称发生变化", func() {
			old := &interfaces.ResourceType{Name: "旧名称"}
			new := &interfaces.ResourceType{Name: "新名称"}

			result := rt.checkResourceTypeChange(old, new)

			assert.True(t, result)
		})

		Convey("InstanceURL发生变化", func() {
			old := &interfaces.ResourceType{InstanceURL: "旧URL"}
			new := &interfaces.ResourceType{InstanceURL: "新URL"}

			result := rt.checkResourceTypeChange(old, new)

			assert.True(t, result)
		})

		Convey("DataStruct发生变化", func() {
			old := &interfaces.ResourceType{DataStruct: "旧结构"}
			new := &interfaces.ResourceType{DataStruct: "新结构"}

			result := rt.checkResourceTypeChange(old, new)

			assert.True(t, result)
		})

		Convey("Description发生变化", func() {
			old := &interfaces.ResourceType{Description: "旧描述"}
			new := &interfaces.ResourceType{Description: "新描述"}

			result := rt.checkResourceTypeChange(old, new)

			assert.True(t, result)
		})

		Convey("Operation发生变化", func() {
			old := &interfaces.ResourceType{
				Operation: []interfaces.ResourceTypeOperation{
					{ID: "old_operation"},
				},
			}
			new := &interfaces.ResourceType{
				Operation: []interfaces.ResourceTypeOperation{
					{ID: "new_operation"},
				},
			}

			result := rt.checkResourceTypeChange(old, new)

			assert.True(t, result)
		})

		Convey("无变化", func() {
			resourceType := &interfaces.ResourceType{
				Name:        "相同名称",
				InstanceURL: "相同URL",
				DataStruct:  "相同结构",
				Description: "相同描述",
				Operation:   []interfaces.ResourceTypeOperation{{ID: "相同操作"}},
			}

			result := rt.checkResourceTypeChange(resourceType, resourceType)

			assert.False(t, result)
		})
	})
}

func TestResourceType_getOperationNameByLanguage(t *testing.T) {
	Convey("测试getOperationNameByLanguage方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBResourceType(ctrl)
		userMgmt := mock.NewMockDrivenUserMgnt(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		rt := newResourceType(db, userMgmt, resourceTypeHierarchy)

		operationNames := []interfaces.OperationName{
			{Language: "zh-cn", Value: "显示"},
			{Language: "en-us", Value: "Display"},
		}

		Convey("找到匹配的语言", func() {
			name := rt.getOperationNameByLanguage("zh-cn", operationNames)
			assert.Equal(t, "显示", name)
		})

		Convey("找到匹配的语言（大小写不敏感）", func() {
			name := rt.getOperationNameByLanguage("ZH-CN", operationNames)
			assert.Equal(t, "显示", name)
		})

		Convey("未找到匹配的语言", func() {
			name := rt.getOperationNameByLanguage("fr-fr", operationNames)
			assert.Empty(t, name)
		})

		Convey("空操作名称列表", func() {
			name := rt.getOperationNameByLanguage("zh-cn", []interfaces.OperationName{})
			assert.Empty(t, name)
		})
	})
}

func TestResourceType_checkVisitorType(t *testing.T) {
	Convey("测试checkVisitorType方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBResourceType(ctrl)
		userMgmt := mock.NewMockDrivenUserMgnt(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		rt := newResourceType(db, userMgmt, resourceTypeHierarchy)

		ctx := context.Background()

		Convey("匿名用户访问", func() {
			visitor := &interfaces.Visitor{
				ID:   testUserID,
				Type: interfaces.Anonymous,
			}

			err := rt.checkVisitorType(ctx, visitor)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "Unsupported user type")
		})

		Convey("实名用户获取角色失败", func() {
			visitor := &interfaces.Visitor{
				ID:   testUserID,
				Type: interfaces.RealName,
			}

			userMgmt.EXPECT().GetUserRolesByUserID(gomock.Any(), testUserID).Return(nil, errors.New("获取角色失败"))

			err := rt.checkVisitorType(ctx, visitor)

			assert.Error(t, err)
		})

		Convey("实名用户无权限角色", func() {
			visitor := &interfaces.Visitor{
				ID:   testUserID,
				Type: interfaces.RealName,
			}

			userMgmt.EXPECT().GetUserRolesByUserID(gomock.Any(), testUserID).Return([]interfaces.SystemRoleType{interfaces.NormalUser}, nil)

			err := rt.checkVisitorType(ctx, visitor)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "Unsupported user role type")
		})

		Convey("实名用户有权限角色", func() {
			visitor := &interfaces.Visitor{
				ID:   testUserID,
				Type: interfaces.RealName,
			}

			userMgmt.EXPECT().GetUserRolesByUserID(gomock.Any(), testUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)

			err := rt.checkVisitorType(ctx, visitor)

			assert.NoError(t, err)
		})
	})
}

func TestResourceType_SetPrivate(t *testing.T) {
	Convey("测试SetPrivate方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBResourceType(ctrl)
		userMgmt := mock.NewMockDrivenUserMgnt(ctrl)
		resourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		rt := newResourceType(db, userMgmt, resourceTypeHierarchy)

		ctx := context.Background()

		Convey("ID长度超过50", func() {
			resourceType := interfaces.ResourceType{ID: "this_is_a_very_long_resource_type_id_that_exceeds_fifty_characters"}

			err := rt.SetPrivate(ctx, &resourceType)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "id length must less than 50")
		})

		Convey("ID包含非法字符", func() {
			resourceType := interfaces.ResourceType{ID: "invalid-id"}

			err := rt.SetPrivate(ctx, &resourceType)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid id, only number or letter or underline")
		})

		Convey("获取现有资源类型失败", func() {
			resourceType := interfaces.ResourceType{ID: testResourceTypeID}

			db.EXPECT().GetByIDs(gomock.Any(), []string{testResourceTypeID}).Return(nil, errors.New("数据库查询失败"))

			err := rt.SetPrivate(ctx, &resourceType)

			assert.Error(t, err)
		})

		Convey("资源类型已存在且无变化", func() {
			existing := interfaces.ResourceType{
				ID:          testResourceTypeID,
				Name:        "名称",
				Description: "描述",
			}
			resourceType := existing

			db.EXPECT().GetByIDs(gomock.Any(), []string{testResourceTypeID}).Return(map[string]interfaces.ResourceType{
				testResourceTypeID: existing,
			}, nil)

			err := rt.SetPrivate(ctx, &resourceType)

			assert.NoError(t, err)
		})

		Convey("保存资源类型失败", func() {
			resourceType := interfaces.ResourceType{ID: testResourceTypeID}

			db.EXPECT().GetByIDs(gomock.Any(), []string{testResourceTypeID}).Return(map[string]interfaces.ResourceType{}, nil)
			db.EXPECT().Set(gomock.Any(), &resourceType).Return(errors.New("保存失败"))

			err := rt.SetPrivate(ctx, &resourceType)

			assert.Error(t, err)
		})

		Convey("资源类型存在但有变化需要更新", func() {
			existing := interfaces.ResourceType{
				ID:          testResourceTypeID,
				Name:        "旧名称",
				Description: "旧描述",
			}
			resourceType := interfaces.ResourceType{
				ID:          testResourceTypeID,
				Name:        "新名称",
				Description: "新描述",
			}

			db.EXPECT().GetByIDs(gomock.Any(), []string{testResourceTypeID}).Return(map[string]interfaces.ResourceType{
				testResourceTypeID: existing,
			}, nil)
			db.EXPECT().Set(gomock.Any(), &resourceType).Return(nil)

			err := rt.SetPrivate(ctx, &resourceType)

			assert.NoError(t, err)
		})

		Convey("成功保存新的资源类型", func() {
			resourceType := interfaces.ResourceType{ID: testResourceTypeID2}

			db.EXPECT().GetByIDs(gomock.Any(), []string{testResourceTypeID2}).Return(map[string]interfaces.ResourceType{}, nil)
			db.EXPECT().Set(gomock.Any(), &resourceType).Return(nil)

			err := rt.SetPrivate(ctx, &resourceType)

			assert.NoError(t, err)
		})
	})
}
