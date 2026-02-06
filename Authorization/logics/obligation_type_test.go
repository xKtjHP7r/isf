package logics

import (
	"context"
	"errors"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	gerrors "github.com/kweaver-ai/go-lib/error"

	"Authorization/common"
	"Authorization/interfaces"
	"Authorization/interfaces/mock"
)

func newObligationType(db interfaces.DBObligationType, userMgnt interfaces.DrivenUserMgnt,
	resourceType interfaces.LogicsResourceType,
) *obligationType {
	return &obligationType{
		db:           db,
		userMgnt:     userMgnt,
		resourceType: resourceType,
		logger:       common.NewLogger(),
	}
}

const (
	testObligationTypeID  = "test_obligation_type"
	testObligationTypeID2 = "test_obligation_type_2"
	testObligationUserID  = "test_user_id"
)

func TestObligationType_checkResourceTypeAndOperation(t *testing.T) {
	Convey("测试checkResourceTypeAndOperation方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBObligationType(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		ot := newObligationType(db, userMgnt, resourceType)

		ctx := context.Background()

		Convey("资源类型不限制", func() {
			info := &interfaces.ObligationTypeInfo{
				ResourceTypeScope: interfaces.ObligationResourceTypeScopeInfo{
					Unlimited: true,
				},
			}

			err := ot.checkResourceTypeAndOperation(ctx, info)

			assert.NoError(t, err)
		})

		Convey("获取资源类型失败", func() {
			info := &interfaces.ObligationTypeInfo{
				ResourceTypeScope: interfaces.ObligationResourceTypeScopeInfo{
					Unlimited: false,
					Types: []interfaces.ObligationResourceTypeScope{
						{
							ResourceTypeID: testResourceTypeID,
						},
					},
				},
			}

			resourceType.EXPECT().GetAllInternal(gomock.Any()).Return(nil, errors.New("数据库错误"))

			err := ot.checkResourceTypeAndOperation(ctx, info)

			assert.Error(t, err)
		})

		Convey("资源类型不存在", func() {
			info := &interfaces.ObligationTypeInfo{
				ResourceTypeScope: interfaces.ObligationResourceTypeScopeInfo{
					Unlimited: false,
					Types: []interfaces.ObligationResourceTypeScope{
						{
							ResourceTypeID: "non_existent_type",
						},
					},
				},
			}

			resourceTypes := []interfaces.ResourceType{
				{
					ID:   testResourceTypeID,
					Name: "测试资源类型",
				},
			}

			resourceType.EXPECT().GetAllInternal(gomock.Any()).Return(resourceTypes, nil)

			err := ot.checkResourceTypeAndOperation(ctx, info)

			assert.Error(t, err)
			assert.Equal(t, gerrors.PublicBadRequest, err.(*gerrors.Error).Code)
		})

		Convey("操作不限制", func() {
			info := &interfaces.ObligationTypeInfo{
				ResourceTypeScope: interfaces.ObligationResourceTypeScopeInfo{
					Unlimited: false,
					Types: []interfaces.ObligationResourceTypeScope{
						{
							ResourceTypeID: testResourceTypeID,
							OperationsScope: interfaces.ObligationOperationsScopeInfo{
								Unlimited: true,
							},
						},
					},
				},
			}

			resourceTypes := []interfaces.ResourceType{
				{
					ID:   testResourceTypeID,
					Name: "测试资源类型",
					Operation: []interfaces.ResourceTypeOperation{
						{ID: "read"},
						{ID: "write"},
					},
				},
			}

			resourceType.EXPECT().GetAllInternal(gomock.Any()).Return(resourceTypes, nil)

			err := ot.checkResourceTypeAndOperation(ctx, info)

			assert.NoError(t, err)
		})

		Convey("操作不存在", func() {
			info := &interfaces.ObligationTypeInfo{
				ResourceTypeScope: interfaces.ObligationResourceTypeScopeInfo{
					Unlimited: false,
					Types: []interfaces.ObligationResourceTypeScope{
						{
							ResourceTypeID: testResourceTypeID,
							OperationsScope: interfaces.ObligationOperationsScopeInfo{
								Unlimited: false,
								Operations: []interfaces.ObligationOperation{
									{ID: "non_existent_operation"},
								},
							},
						},
					},
				},
			}

			resourceTypes := []interfaces.ResourceType{
				{
					ID:   testResourceTypeID,
					Name: "测试资源类型",
					Operation: []interfaces.ResourceTypeOperation{
						{ID: "read"},
						{ID: "write"},
					},
				},
			}

			resourceType.EXPECT().GetAllInternal(gomock.Any()).Return(resourceTypes, nil)

			err := ot.checkResourceTypeAndOperation(ctx, info)

			assert.Error(t, err)
			assert.Equal(t, gerrors.PublicBadRequest, err.(*gerrors.Error).Code)
		})

		Convey("资源类型和操作都合法", func() {
			info := &interfaces.ObligationTypeInfo{
				ResourceTypeScope: interfaces.ObligationResourceTypeScopeInfo{
					Unlimited: false,
					Types: []interfaces.ObligationResourceTypeScope{
						{
							ResourceTypeID: testResourceTypeID,
							OperationsScope: interfaces.ObligationOperationsScopeInfo{
								Unlimited: false,
								Operations: []interfaces.ObligationOperation{
									{ID: "read"},
									{ID: "write"},
								},
							},
						},
					},
				},
			}

			resourceTypes := []interfaces.ResourceType{
				{
					ID:   testResourceTypeID,
					Name: "测试资源类型",
					Operation: []interfaces.ResourceTypeOperation{
						{ID: "read"},
						{ID: "write"},
						{ID: "delete"},
					},
				},
			}

			resourceType.EXPECT().GetAllInternal(gomock.Any()).Return(resourceTypes, nil)

			err := ot.checkResourceTypeAndOperation(ctx, info)

			assert.NoError(t, err)
		})
	})
}

func TestObligationType_Set(t *testing.T) {
	Convey("测试Set方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBObligationType(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		ot := newObligationType(db, userMgnt, resourceType)

		ctx := context.Background()
		visitor := &interfaces.Visitor{
			ID:   testObligationUserID,
			Type: interfaces.RealName,
		}

		Convey("权限检查失败", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return(nil, errors.New("权限检查失败"))

			info := &interfaces.ObligationTypeInfo{
				ID: testObligationTypeID,
			}
			err := ot.Set(ctx, visitor, info)

			assert.Error(t, err)
		})

		Convey("Schema不合法", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)

			info := &interfaces.ObligationTypeInfo{
				ID:     testObligationTypeID,
				Schema: "invalid schema",
			}
			err := ot.Set(ctx, visitor, info)

			assert.Error(t, err)
			assert.Equal(t, gerrors.PublicBadRequest, err.(*gerrors.Error).Code)
		})

		Convey("默认值不符合Schema", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)

			info := &interfaces.ObligationTypeInfo{
				ID: testObligationTypeID,
				Schema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{
							"type": "string",
						},
					},
					"required": []any{"name"},
				},
				DefaultValue: map[string]any{
					"age": 18, // 缺少required字段name
				},
			}
			err := ot.Set(ctx, visitor, info)

			assert.Error(t, err)
		})

		Convey("资源类型和操作检查失败", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			resourceType.EXPECT().GetAllInternal(gomock.Any()).Return(nil, errors.New("数据库错误"))

			info := &interfaces.ObligationTypeInfo{
				ID: testObligationTypeID,
				Schema: map[string]any{
					"type": "string",
				},
				ResourceTypeScope: interfaces.ObligationResourceTypeScopeInfo{
					Unlimited: false,
					Types: []interfaces.ObligationResourceTypeScope{
						{ResourceTypeID: testResourceTypeID},
					},
				},
			}
			err := ot.Set(ctx, visitor, info)

			assert.Error(t, err)
		})

		Convey("数据库保存失败", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			db.EXPECT().Set(gomock.Any(), gomock.Any()).Return(errors.New("数据库保存失败"))
			info := &interfaces.ObligationTypeInfo{
				ID: testObligationTypeID,
				Schema: map[string]any{
					"type": "string",
				},
				ResourceTypeScope: interfaces.ObligationResourceTypeScopeInfo{
					Unlimited: true,
				},
			}
			err := ot.Set(ctx, visitor, info)

			assert.Error(t, err)
		})

		Convey("成功保存义务类型", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			resourceType.EXPECT().GetAllInternal(gomock.Any()).Return([]interfaces.ResourceType{
				{
					ID: testResourceTypeID,
					Operation: []interfaces.ResourceTypeOperation{
						{ID: "read"},
					},
				},
			}, nil)
			db.EXPECT().Set(gomock.Any(), gomock.Any()).Return(nil)

			info := &interfaces.ObligationTypeInfo{
				ID: testObligationTypeID,
				Schema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{
							"type": "string",
						},
					},
				},
				DefaultValue: map[string]any{
					"name": "test",
				},
				ResourceTypeScope: interfaces.ObligationResourceTypeScopeInfo{
					Unlimited: false,
					Types: []interfaces.ObligationResourceTypeScope{
						{
							ResourceTypeID: testResourceTypeID,
							OperationsScope: interfaces.ObligationOperationsScopeInfo{
								Unlimited: false,
								Operations: []interfaces.ObligationOperation{
									{ID: "read"},
								},
							},
						},
					},
				},
			}
			err := ot.Set(ctx, visitor, info)

			assert.NoError(t, err)
		})
	})
}

//nolint:dupl
func TestObligationType_Delete(t *testing.T) {
	Convey("测试Delete方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBObligationType(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		ot := newObligationType(db, userMgnt, resourceType)

		ctx := context.Background()
		visitor := &interfaces.Visitor{
			ID:   testObligationUserID,
			Type: interfaces.RealName,
		}

		Convey("权限检查失败", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return(nil, errors.New("权限检查失败"))

			err := ot.Delete(ctx, visitor, testObligationTypeID)

			assert.Error(t, err)
		})

		Convey("数据库删除失败", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			db.EXPECT().Delete(gomock.Any(), testObligationTypeID).Return(errors.New("数据库删除失败"))

			err := ot.Delete(ctx, visitor, testObligationTypeID)

			assert.Error(t, err)
		})

		Convey("成功删除义务类型", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			db.EXPECT().Delete(gomock.Any(), testObligationTypeID).Return(nil)

			err := ot.Delete(ctx, visitor, testObligationTypeID)

			assert.NoError(t, err)
		})
	})
}

func TestObligationType_GetByID(t *testing.T) {
	Convey("测试GetByID方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBObligationType(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		ot := newObligationType(db, userMgnt, resourceType)

		ctx := context.Background()
		visitor := &interfaces.Visitor{
			ID:       testObligationUserID,
			Type:     interfaces.RealName,
			Language: "zh-cn",
		}

		Convey("权限检查失败", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return(nil, errors.New("权限检查失败"))

			info, err := ot.GetByID(ctx, visitor, testObligationTypeID)

			assert.Error(t, err)
			assert.Equal(t, interfaces.ObligationTypeInfo{}, info)
		})

		Convey("数据库查询失败", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			db.EXPECT().GetByID(gomock.Any(), testObligationTypeID).Return(interfaces.ObligationTypeInfo{}, errors.New("数据库查询失败"))

			info, err := ot.GetByID(ctx, visitor, testObligationTypeID)

			assert.Error(t, err)
			assert.Equal(t, interfaces.ObligationTypeInfo{}, info)
		})

		Convey("义务类型不存在", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			db.EXPECT().GetByID(gomock.Any(), testObligationTypeID).Return(interfaces.ObligationTypeInfo{}, nil)

			_, err := ot.GetByID(ctx, visitor, testObligationTypeID)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "not found")
		})

		Convey("资源类型不限制", func() {
			expectedInfo := interfaces.ObligationTypeInfo{
				ID:   testObligationTypeID,
				Name: "测试义务类型",
				ResourceTypeScope: interfaces.ObligationResourceTypeScopeInfo{
					Unlimited: true,
				},
			}

			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			db.EXPECT().GetByID(gomock.Any(), testObligationTypeID).Return(expectedInfo, nil)

			info, err := ot.GetByID(ctx, visitor, testObligationTypeID)

			assert.NoError(t, err)
			assert.Equal(t, expectedInfo, info)
		})

		Convey("获取资源类型信息失败", func() {
			expectedInfo := interfaces.ObligationTypeInfo{
				ID:   testObligationTypeID,
				Name: "测试义务类型",
				ResourceTypeScope: interfaces.ObligationResourceTypeScopeInfo{
					Unlimited: false,
					Types: []interfaces.ObligationResourceTypeScope{
						{ResourceTypeID: testResourceTypeID},
					},
				},
			}

			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			db.EXPECT().GetByID(gomock.Any(), testObligationTypeID).Return(expectedInfo, nil)
			resourceType.EXPECT().GetAllInternal(gomock.Any()).Return(nil, errors.New("数据库错误"))

			_, err := ot.GetByID(ctx, visitor, testObligationTypeID)

			assert.Error(t, err)
		})

		Convey("成功获取义务类型并填充资源类型名称", func() {
			expectedInfo := interfaces.ObligationTypeInfo{
				ID:   testObligationTypeID,
				Name: "测试义务类型",
				ResourceTypeScope: interfaces.ObligationResourceTypeScopeInfo{
					Unlimited: false,
					Types: []interfaces.ObligationResourceTypeScope{
						{
							ResourceTypeID: testResourceTypeID,
							OperationsScope: interfaces.ObligationOperationsScopeInfo{
								Unlimited: false,
								Operations: []interfaces.ObligationOperation{
									{ID: "read"},
								},
							},
						},
					},
				},
			}

			resourceTypes := []interfaces.ResourceType{
				{
					ID:   testResourceTypeID,
					Name: "测试资源类型",
					Operation: []interfaces.ResourceTypeOperation{
						{
							ID: "read",
							Name: []interfaces.OperationName{
								{Language: "zh-cn", Value: "读取"},
								{Language: "en-us", Value: "read"},
							},
						},
					},
				},
			}

			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			db.EXPECT().GetByID(gomock.Any(), testObligationTypeID).Return(expectedInfo, nil)
			resourceType.EXPECT().GetAllInternal(gomock.Any()).Return(resourceTypes, nil)

			info, err := ot.GetByID(ctx, visitor, testObligationTypeID)

			assert.NoError(t, err)
			assert.Equal(t, testObligationTypeID, info.ID)
			assert.Equal(t, "测试资源类型", info.ResourceTypeScope.Types[0].ResourceTypeName)
			assert.Equal(t, "读取", info.ResourceTypeScope.Types[0].OperationsScope.Operations[0].Name)
		})
	})
}

func TestObligationType_getOperationNameByLanguage(t *testing.T) {
	Convey("测试getOperationNameByLanguage方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBObligationType(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		ot := newObligationType(db, userMgnt, resourceType)

		operationNames := []interfaces.OperationName{
			{Language: "zh-cn", Value: "读取"},
			{Language: "en-us", Value: "read"},
		}

		Convey("找到匹配的语言", func() {
			name := ot.getOperationNameByLanguage(operationNames, "zh-cn")
			assert.Equal(t, "读取", name)
		})

		Convey("找到匹配的语言（大小写不敏感）", func() {
			name := ot.getOperationNameByLanguage(operationNames, "ZH-CN")
			assert.Equal(t, "读取", name)
		})

		Convey("未找到匹配的语言", func() {
			name := ot.getOperationNameByLanguage(operationNames, "fr-fr")
			assert.Empty(t, name)
		})

		Convey("空操作名称列表", func() {
			name := ot.getOperationNameByLanguage([]interfaces.OperationName{}, "zh-cn")
			assert.Empty(t, name)
		})
	})
}

func TestObligationType_Get(t *testing.T) {
	Convey("测试Get方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBObligationType(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		ot := newObligationType(db, userMgnt, resourceType)

		ctx := context.Background()
		visitor := &interfaces.Visitor{
			ID:       testObligationUserID,
			Type:     interfaces.RealName,
			Language: "zh-cn",
		}

		Convey("权限检查失败", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return(nil, errors.New("权限检查失败"))

			searchInfo := &interfaces.ObligationTypeSearchInfo{}
			count, infos, err := ot.Get(ctx, visitor, searchInfo)

			assert.Error(t, err)
			assert.Equal(t, 0, count)
			assert.Nil(t, infos)
		})

		Convey("数据库查询失败", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			db.EXPECT().Get(gomock.Any(), gomock.Any()).Return(0, nil, errors.New("数据库查询失败"))

			searchInfo := &interfaces.ObligationTypeSearchInfo{}
			count, infos, err := ot.Get(ctx, visitor, searchInfo)

			assert.Error(t, err)
			assert.Equal(t, 0, count)
			assert.Nil(t, infos)
		})

		Convey("成功获取义务类型列表（无记录）", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			db.EXPECT().Get(gomock.Any(), gomock.Any()).Return(0, []interfaces.ObligationTypeInfo{}, nil)

			searchInfo := &interfaces.ObligationTypeSearchInfo{}
			count, infos, err := ot.Get(ctx, visitor, searchInfo)

			assert.NoError(t, err)
			assert.Equal(t, 0, count)
			assert.Empty(t, infos)
		})
	})
}

//nolint:funlen
func TestObligationType_Query(t *testing.T) {
	Convey("测试Query方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBObligationType(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		ot := newObligationType(db, userMgnt, resourceType)

		ctx := context.Background()
		visitor := &interfaces.Visitor{
			ID:       testObligationUserID,
			Type:     interfaces.RealName,
			Language: "zh-cn",
		}

		Convey("资源类型不存在", func() {
			queryInfo := &interfaces.QueryObligationTypeInfo{
				ResourceType: "non_existent_type",
			}

			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), []string{"non_existent_type"}).Return(map[string]interfaces.ResourceType{}, nil)

			infos, err := ot.Query(ctx, visitor, queryInfo)

			assert.Error(t, err)
			assert.Nil(t, infos)
		})

		Convey("操作不存在", func() {
			queryInfo := &interfaces.QueryObligationTypeInfo{
				ResourceType: testResourceTypeID,
				Operation:    []string{"non_existent_operation"},
			}

			resourceTypeMap := map[string]interfaces.ResourceType{
				testResourceTypeID: {
					ID: testResourceTypeID,
					Operation: []interfaces.ResourceTypeOperation{
						{ID: "read"},
					},
				},
			}

			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), []string{testResourceTypeID}).Return(resourceTypeMap, nil)

			infos, err := ot.Query(ctx, visitor, queryInfo)

			assert.Error(t, err)
			assert.Nil(t, infos)
		})

		Convey("获取所有义务类型失败", func() {
			queryInfo := &interfaces.QueryObligationTypeInfo{
				ResourceType: testResourceTypeID,
			}

			resourceTypeMap := map[string]interfaces.ResourceType{
				testResourceTypeID: {
					ID: testResourceTypeID,
					Operation: []interfaces.ResourceTypeOperation{
						{ID: "read"},
					},
				},
			}

			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), []string{testResourceTypeID}).Return(resourceTypeMap, nil)
			db.EXPECT().GetAll(gomock.Any()).Return(nil, errors.New("数据库错误"))

			infos, err := ot.Query(ctx, visitor, queryInfo)

			assert.Error(t, err)
			assert.Nil(t, infos)
		})

		Convey("成功查询义务类型（资源类型不限制）", func() {
			queryInfo := &interfaces.QueryObligationTypeInfo{
				ResourceType: testResourceTypeID,
				Operation:    []string{"read"},
			}

			resourceTypeMap := map[string]interfaces.ResourceType{
				testResourceTypeID: {
					ID: testResourceTypeID,
					Operation: []interfaces.ResourceTypeOperation{
						{ID: "read"},
						{ID: "write"},
					},
				},
			}

			allObligations := []interfaces.ObligationTypeInfo{
				{
					ID:   testObligationTypeID,
					Name: "测试义务1",
					ResourceTypeScope: interfaces.ObligationResourceTypeScopeInfo{
						Unlimited: true,
					},
				},
			}

			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), []string{testResourceTypeID}).Return(resourceTypeMap, nil)
			db.EXPECT().GetAll(gomock.Any()).Return(allObligations, nil)

			infos, err := ot.Query(ctx, visitor, queryInfo)

			assert.NoError(t, err)
			assert.NotNil(t, infos)
			assert.Len(t, infos["read"], 1)
			assert.Equal(t, testObligationTypeID, infos["read"][0].ID)
		})

		Convey("成功查询义务类型（操作不限制）", func() {
			queryInfo := &interfaces.QueryObligationTypeInfo{
				ResourceType: testResourceTypeID,
				Operation:    []string{"read", "write"},
			}

			resourceTypeMap := map[string]interfaces.ResourceType{
				testResourceTypeID: {
					ID: testResourceTypeID,
					Operation: []interfaces.ResourceTypeOperation{
						{ID: "read"},
						{ID: "write"},
					},
				},
			}

			allObligations := []interfaces.ObligationTypeInfo{
				{
					ID:   testObligationTypeID,
					Name: "测试义务1",
					ResourceTypeScope: interfaces.ObligationResourceTypeScopeInfo{
						Unlimited: false,
						Types: []interfaces.ObligationResourceTypeScope{
							{
								ResourceTypeID: testResourceTypeID,
								OperationsScope: interfaces.ObligationOperationsScopeInfo{
									Unlimited: true,
								},
							},
						},
					},
				},
			}

			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), []string{testResourceTypeID}).Return(resourceTypeMap, nil)
			db.EXPECT().GetAll(gomock.Any()).Return(allObligations, nil)

			infos, err := ot.Query(ctx, visitor, queryInfo)

			assert.NoError(t, err)
			assert.NotNil(t, infos)
			assert.Len(t, infos["read"], 1)
			assert.Len(t, infos["write"], 1)
		})

		Convey("成功查询义务类型（指定操作）", func() {
			queryInfo := &interfaces.QueryObligationTypeInfo{
				ResourceType: testResourceTypeID,
				Operation:    []string{"read"},
			}

			resourceTypeMap := map[string]interfaces.ResourceType{
				testResourceTypeID: {
					ID: testResourceTypeID,
					Operation: []interfaces.ResourceTypeOperation{
						{ID: "read"},
						{ID: "write"},
					},
				},
			}

			allObligations := []interfaces.ObligationTypeInfo{
				{
					ID:   testObligationTypeID,
					Name: "测试义务1",
					ResourceTypeScope: interfaces.ObligationResourceTypeScopeInfo{
						Unlimited: false,
						Types: []interfaces.ObligationResourceTypeScope{
							{
								ResourceTypeID: testResourceTypeID,
								OperationsScope: interfaces.ObligationOperationsScopeInfo{
									Unlimited: false,
									Operations: []interfaces.ObligationOperation{
										{ID: "read"},
									},
								},
							},
						},
					},
				},
			}

			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), []string{testResourceTypeID}).Return(resourceTypeMap, nil)
			db.EXPECT().GetAll(gomock.Any()).Return(allObligations, nil)

			infos, err := ot.Query(ctx, visitor, queryInfo)

			assert.NoError(t, err)
			assert.NotNil(t, infos)
			assert.Len(t, infos["read"], 1)
			assert.Equal(t, testObligationTypeID, infos["read"][0].ID)
		})

		Convey("默认返回所有操作", func() {
			queryInfo := &interfaces.QueryObligationTypeInfo{
				ResourceType: testResourceTypeID,
				Operation:    []string{},
			}

			resourceTypeMap := map[string]interfaces.ResourceType{
				testResourceTypeID: {
					ID: testResourceTypeID,
					Operation: []interfaces.ResourceTypeOperation{
						{ID: "read"},
						{ID: "write"},
					},
				},
			}

			allObligations := []interfaces.ObligationTypeInfo{}

			resourceType.EXPECT().GetByIDsInternal(gomock.Any(), []string{testResourceTypeID}).Return(resourceTypeMap, nil)
			db.EXPECT().GetAll(gomock.Any()).Return(allObligations, nil)

			infos, err := ot.Query(ctx, visitor, queryInfo)

			assert.NoError(t, err)
			assert.NotNil(t, infos)
			assert.Len(t, infos, 2) // read和write两个操作
		})
	})
}

func TestObligationType_GetByIDSInternal(t *testing.T) {
	Convey("测试GetByIDSInternal方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBObligationType(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		ot := newObligationType(db, userMgnt, resourceType)

		ctx := context.Background()

		Convey("数据库查询失败", func() {
			obligationTypeIDs := map[string]bool{
				testObligationTypeID: true,
			}

			db.EXPECT().GetByIDs(gomock.Any(), []string{testObligationTypeID}).Return(nil, errors.New("数据库查询失败"))

			infos, err := ot.GetByIDSInternal(ctx, obligationTypeIDs)

			assert.Error(t, err)
			assert.Nil(t, infos)
		})

		Convey("成功获取义务类型列表", func() {
			obligationTypeIDs := map[string]bool{
				testObligationTypeID:  true,
				testObligationTypeID2: true,
			}

			expectedInfos := []interfaces.ObligationTypeInfo{
				{
					ID:   testObligationTypeID,
					Name: "测试义务类型1",
				},
				{
					ID:   testObligationTypeID2,
					Name: "测试义务类型2",
				},
			}

			db.EXPECT().GetByIDs(gomock.Any(), gomock.Any()).Return(expectedInfos, nil)

			infos, err := ot.GetByIDSInternal(ctx, obligationTypeIDs)

			assert.NoError(t, err)
			assert.Len(t, infos, 2)
		})

		Convey("空的ID映射", func() {
			obligationTypeIDs := map[string]bool{}

			db.EXPECT().GetByIDs(gomock.Any(), []string{}).Return([]interfaces.ObligationTypeInfo{}, nil)

			infos, err := ot.GetByIDSInternal(ctx, obligationTypeIDs)

			assert.NoError(t, err)
			assert.Empty(t, infos)
		})
	})
}

func TestObligationType_SetPrivate(t *testing.T) {
	Convey("测试SetPrivate方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBObligationType(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		resourceType := mock.NewMockLogicsResourceType(ctrl)
		ot := newObligationType(db, userMgnt, resourceType)

		ctx := context.Background()
		visitor := &interfaces.Visitor{}

		Convey("Schema不合法", func() {
			info := &interfaces.ObligationTypeInfo{
				ID:     testObligationTypeID,
				Schema: "",
			}
			err := ot.SetPrivate(ctx, visitor, info)

			assert.Error(t, err)
			assert.Equal(t, gerrors.PublicBadRequest, err.(*gerrors.Error).Code)
		})

		Convey("默认值不符合Schema", func() {
			info := &interfaces.ObligationTypeInfo{
				ID: testObligationTypeID,
				Schema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{
							"type": "string",
						},
					},
					"required": []any{"name"},
				},
				DefaultValue: map[string]any{
					"age": 18, // 缺少required字段name
				},
			}
			err := ot.SetPrivate(ctx, visitor, info)

			assert.Error(t, err)
		})

		Convey("资源类型和操作检查失败", func() {
			resourceType.EXPECT().GetAllInternal(gomock.Any()).Return(nil, errors.New("数据库错误"))

			info := &interfaces.ObligationTypeInfo{
				ID: testObligationTypeID,
				Schema: map[string]any{
					"type": "string",
				},
				ResourceTypeScope: interfaces.ObligationResourceTypeScopeInfo{
					Unlimited: false,
					Types: []interfaces.ObligationResourceTypeScope{
						{ResourceTypeID: testResourceTypeID},
					},
				},
			}
			err := ot.SetPrivate(ctx, visitor, info)

			assert.Error(t, err)
		})

		Convey("数据库保存失败", func() {
			resourceType.EXPECT().GetAllInternal(gomock.Any()).Return([]interfaces.ResourceType{
				{ID: testResourceTypeID, Operation: []interfaces.ResourceTypeOperation{{ID: "read"}}},
			}, nil)
			db.EXPECT().Set(gomock.Any(), gomock.Any()).Return(errors.New("数据库保存失败"))
			info := &interfaces.ObligationTypeInfo{
				ID: testObligationTypeID,
				Schema: map[string]any{
					"type": "string",
				},
				ResourceTypeScope: interfaces.ObligationResourceTypeScopeInfo{
					Unlimited: false,
					Types: []interfaces.ObligationResourceTypeScope{
						{ResourceTypeID: testResourceTypeID, OperationsScope: interfaces.ObligationOperationsScopeInfo{Unlimited: false, Operations: []interfaces.ObligationOperation{{ID: "read"}}}},
					},
				},
			}
			err := ot.SetPrivate(ctx, visitor, info)

			assert.Error(t, err)
		})

		Convey("成功保存义务类型", func() {
			resourceType.EXPECT().GetAllInternal(gomock.Any()).Return([]interfaces.ResourceType{
				{
					ID: testResourceTypeID,
					Operation: []interfaces.ResourceTypeOperation{
						{ID: "read"},
					},
				},
			}, nil)
			db.EXPECT().Set(gomock.Any(), gomock.Any()).Return(nil)

			info := &interfaces.ObligationTypeInfo{
				ID: testObligationTypeID,
				Schema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{
							"type": "string",
						},
					},
				},
				DefaultValue: map[string]any{
					"name": "test",
				},
				ResourceTypeScope: interfaces.ObligationResourceTypeScopeInfo{
					Unlimited: false,
					Types: []interfaces.ObligationResourceTypeScope{
						{
							ResourceTypeID: testResourceTypeID,
							OperationsScope: interfaces.ObligationOperationsScopeInfo{
								Unlimited: false,
								Operations: []interfaces.ObligationOperation{
									{ID: "read"},
								},
							},
						},
					},
				},
			}
			err := ot.SetPrivate(ctx, visitor, info)

			assert.NoError(t, err)
		})
	})
}
