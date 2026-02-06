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

func newObligation(db interfaces.DBObligation, userMgnt interfaces.DrivenUserMgnt,
	obligationType interfaces.ObligationType,
) *obligation {
	return &obligation{
		db:             db,
		userMgnt:       userMgnt,
		obligationType: obligationType,
		logger:         common.NewLogger(),
	}
}

const (
	testObligationID   = "test_obligation_id"
	testObligationID2  = "test_obligation_id_2"
	testObligationName = "test_obligation_name"
	testObligationDesc = "test_obligation_description"
	testResourceType   = "test_resource_type"
	testOperationID    = "test_operation_id"
)

func TestObligation_checkJsonValueValid(t *testing.T) {
	Convey("测试checkJsonValueValid方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBObligation(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		obligationType := mock.NewMockObligationType(ctrl)
		ob := newObligation(db, userMgnt, obligationType)

		ctx := context.Background()

		Convey("schema无效", func() {
			schema := map[string]any{
				"type":       "invalid_type",
				"properties": "invalid",
			}
			value := map[string]any{
				"test": "value",
			}

			err := ob.checkJsonValueValid(ctx, schema, value)

			assert.Error(t, err)
		})

		Convey("value不符合schema", func() {
			schema := map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type": "string",
					},
					"age": map[string]any{
						"type": "integer",
					},
				},
				"required": []string{"name", "age"},
			}
			value := map[string]any{
				"name": "test",
				// 缺少必需的 age 字段
			}

			err := ob.checkJsonValueValid(ctx, schema, value)

			assert.Error(t, err)
		})

		Convey("value符合schema", func() {
			schema := map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type": "string",
					},
					"age": map[string]any{
						"type": "integer",
					},
				},
				"required": []string{"name", "age"},
			}
			value := map[string]any{
				"name": "test",
				"age":  25,
			}

			err := ob.checkJsonValueValid(ctx, schema, value)

			assert.NoError(t, err)
		})
	})
}

func TestObligation_Add(t *testing.T) {
	Convey("测试Add方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBObligation(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		obligationType := mock.NewMockObligationType(ctrl)
		ob := newObligation(db, userMgnt, obligationType)

		ctx := context.Background()
		visitor := &interfaces.Visitor{
			ID:   testObligationUserID,
			Type: interfaces.RealName,
		}

		obligationInfo := &interfaces.ObligationInfo{
			TypeID:      testObligationTypeID,
			Name:        testObligationName,
			Description: testObligationDesc,
			Value: map[string]any{
				"field": "value",
			},
		}

		Convey("用户权限检查失败", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.NormalUser}, nil)

			id, err := ob.Add(ctx, visitor, obligationInfo)

			assert.Error(t, err)
			assert.Empty(t, id)
		})

		Convey("义务类型不存在", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			obligationType.EXPECT().GetByIDSInternal(gomock.Any(), gomock.Any()).Return([]interfaces.ObligationTypeInfo{}, nil)

			id, err := ob.Add(ctx, visitor, obligationInfo)

			assert.Error(t, err)
			assert.Equal(t, gerrors.PublicBadRequest, err.(*gerrors.Error).Code)
			assert.Empty(t, id)
		})

		Convey("获取义务类型失败", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			obligationType.EXPECT().GetByIDSInternal(gomock.Any(), gomock.Any()).Return(nil, errors.New("数据库错误"))

			id, err := ob.Add(ctx, visitor, obligationInfo)

			assert.Error(t, err)
			assert.Empty(t, id)
		})

		Convey("value不符合schema", func() {
			schema := map[string]any{
				"type": "object",
				"properties": map[string]any{
					"required_field": map[string]any{
						"type": "string",
					},
				},
				"required": []string{"required_field"},
			}
			obligationTypeInfo := []interfaces.ObligationTypeInfo{
				{
					ID:     testObligationTypeID,
					Schema: schema,
				},
			}

			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			obligationType.EXPECT().GetByIDSInternal(gomock.Any(), gomock.Any()).Return(obligationTypeInfo, nil)

			_, err := ob.Add(ctx, visitor, obligationInfo)

			assert.Error(t, err)
			assert.Equal(t, gerrors.PublicBadRequest, err.(*gerrors.Error).Code)
		})

		Convey("数据库添加失败", func() {
			schema := map[string]any{
				"type": "object",
				"properties": map[string]any{
					"field": map[string]any{
						"type": "string",
					},
				},
			}
			obligationTypeInfo := []interfaces.ObligationTypeInfo{
				{
					ID:     testObligationTypeID,
					Schema: schema,
				},
			}

			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			obligationType.EXPECT().GetByIDSInternal(gomock.Any(), gomock.Any()).Return(obligationTypeInfo, nil)
			db.EXPECT().Add(gomock.Any(), gomock.Any()).Return(errors.New("数据库错误"))

			id, err := ob.Add(ctx, visitor, obligationInfo)

			assert.Error(t, err)
			assert.NotEmpty(t, id) // ID已经生成了
		})

		Convey("成功添加义务", func() {
			schema := map[string]any{
				"type": "object",
				"properties": map[string]any{
					"field": map[string]any{
						"type": "string",
					},
				},
			}
			obligationTypeInfo := []interfaces.ObligationTypeInfo{
				{
					ID:     testObligationTypeID,
					Schema: schema,
				},
			}

			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			obligationType.EXPECT().GetByIDSInternal(gomock.Any(), gomock.Any()).Return(obligationTypeInfo, nil)
			db.EXPECT().Add(gomock.Any(), gomock.Any()).Return(nil)

			id, err := ob.Add(ctx, visitor, obligationInfo)

			assert.NoError(t, err)
			assert.NotEmpty(t, id)
		})
	})
}

func TestObligation_Delete(t *testing.T) {
	Convey("测试Delete方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBObligation(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		obligationType := mock.NewMockObligationType(ctrl)
		ob := newObligation(db, userMgnt, obligationType)

		ctx := context.Background()
		visitor := &interfaces.Visitor{
			ID:   testObligationUserID,
			Type: interfaces.RealName,
		}

		Convey("用户权限检查失败", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.NormalUser}, nil)

			err := ob.Delete(ctx, visitor, testObligationID)

			assert.Error(t, err)
		})

		Convey("数据库删除失败", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			db.EXPECT().Delete(gomock.Any(), testObligationID).Return(errors.New("数据库错误"))

			err := ob.Delete(ctx, visitor, testObligationID)

			assert.Error(t, err)
		})

		Convey("成功删除义务", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			db.EXPECT().Delete(gomock.Any(), testObligationID).Return(nil)

			err := ob.Delete(ctx, visitor, testObligationID)

			assert.NoError(t, err)
		})
	})
}

func TestObligation_Update(t *testing.T) {
	Convey("测试Update方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBObligation(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		obligationType := mock.NewMockObligationType(ctrl)
		ob := newObligation(db, userMgnt, obligationType)

		ctx := context.Background()
		visitor := &interfaces.Visitor{
			ID:   testObligationUserID,
			Type: interfaces.RealName,
		}

		newName := "new_name"
		newDesc := "new_description"
		newValue := map[string]any{
			"field": "new_value",
		}

		Convey("用户权限检查失败", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.NormalUser}, nil)

			err := ob.Update(ctx, visitor, testObligationID, newName, true, newDesc, true, newValue, true)

			assert.Error(t, err)
		})

		Convey("义务不存在", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			db.EXPECT().GetByID(gomock.Any(), testObligationID).Return(interfaces.ObligationInfo{}, nil)

			err := ob.Update(ctx, visitor, testObligationID, newName, true, newDesc, true, newValue, true)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "not found")
		})

		Convey("获取义务失败", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			db.EXPECT().GetByID(gomock.Any(), testObligationID).Return(interfaces.ObligationInfo{}, errors.New("数据库错误"))

			err := ob.Update(ctx, visitor, testObligationID, newName, true, newDesc, true, newValue, true)

			assert.Error(t, err)
		})

		Convey("更新value时获取义务类型失败", func() {
			existingObligation := interfaces.ObligationInfo{
				ID:     testObligationID,
				TypeID: testObligationTypeID,
			}

			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			db.EXPECT().GetByID(gomock.Any(), testObligationID).Return(existingObligation, nil)
			obligationType.EXPECT().GetByIDSInternal(gomock.Any(), gomock.Any()).Return(nil, errors.New("数据库错误"))

			err := ob.Update(ctx, visitor, testObligationID, newName, true, newDesc, true, newValue, true)

			assert.Error(t, err)
		})

		Convey("更新value时义务类型不存在", func() {
			existingObligation := interfaces.ObligationInfo{
				ID:     testObligationID,
				TypeID: testObligationTypeID,
			}

			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			db.EXPECT().GetByID(gomock.Any(), testObligationID).Return(existingObligation, nil)
			obligationType.EXPECT().GetByIDSInternal(gomock.Any(), gomock.Any()).Return([]interfaces.ObligationTypeInfo{}, nil)

			err := ob.Update(ctx, visitor, testObligationID, newName, true, newDesc, true, newValue, true)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "not found")
		})

		Convey("更新value时校验失败", func() {
			existingObligation := interfaces.ObligationInfo{
				ID:     testObligationID,
				TypeID: testObligationTypeID,
			}
			schema := map[string]any{
				"type": "object",
				"properties": map[string]any{
					"required_field": map[string]any{
						"type": "string",
					},
				},
				"required": []string{"required_field"},
			}
			obligationTypeInfo := []interfaces.ObligationTypeInfo{
				{
					ID:     testObligationTypeID,
					Schema: schema,
				},
			}

			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			db.EXPECT().GetByID(gomock.Any(), testObligationID).Return(existingObligation, nil)
			obligationType.EXPECT().GetByIDSInternal(gomock.Any(), gomock.Any()).Return(obligationTypeInfo, nil)

			err := ob.Update(ctx, visitor, testObligationID, newName, true, newDesc, true, newValue, true)

			assert.Error(t, err)
		})

		Convey("数据库更新失败", func() {
			existingObligation := interfaces.ObligationInfo{
				ID:     testObligationID,
				TypeID: testObligationTypeID,
			}
			schema := map[string]any{
				"type": "object",
				"properties": map[string]any{
					"field": map[string]any{
						"type": "string",
					},
				},
			}
			obligationTypeInfo := []interfaces.ObligationTypeInfo{
				{
					ID:     testObligationTypeID,
					Schema: schema,
				},
			}

			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			db.EXPECT().GetByID(gomock.Any(), testObligationID).Return(existingObligation, nil)
			obligationType.EXPECT().GetByIDSInternal(gomock.Any(), gomock.Any()).Return(obligationTypeInfo, nil)
			db.EXPECT().Update(gomock.Any(), testObligationID, newName, true, newDesc, true, newValue, true).Return(errors.New("数据库错误"))

			err := ob.Update(ctx, visitor, testObligationID, newName, true, newDesc, true, newValue, true)

			assert.Error(t, err)
		})

		Convey("成功更新义务（包括value）", func() {
			existingObligation := interfaces.ObligationInfo{
				ID:     testObligationID,
				TypeID: testObligationTypeID,
			}
			schema := map[string]any{
				"type": "object",
				"properties": map[string]any{
					"field": map[string]any{
						"type": "string",
					},
				},
			}
			obligationTypeInfo := []interfaces.ObligationTypeInfo{
				{
					ID:     testObligationTypeID,
					Schema: schema,
				},
			}

			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			db.EXPECT().GetByID(gomock.Any(), testObligationID).Return(existingObligation, nil)
			obligationType.EXPECT().GetByIDSInternal(gomock.Any(), gomock.Any()).Return(obligationTypeInfo, nil)
			db.EXPECT().Update(gomock.Any(), testObligationID, newName, true, newDesc, true, newValue, true).Return(nil)

			err := ob.Update(ctx, visitor, testObligationID, newName, true, newDesc, true, newValue, true)

			assert.NoError(t, err)
		})

		Convey("成功更新义务（不更新value）", func() {
			existingObligation := interfaces.ObligationInfo{
				ID:     testObligationID,
				TypeID: testObligationTypeID,
			}

			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			db.EXPECT().GetByID(gomock.Any(), testObligationID).Return(existingObligation, nil)
			db.EXPECT().Update(gomock.Any(), testObligationID, newName, true, newDesc, true, nil, false).Return(nil)

			err := ob.Update(ctx, visitor, testObligationID, newName, true, newDesc, true, nil, false)

			assert.NoError(t, err)
		})
	})
}

func TestObligation_GetByID(t *testing.T) {
	Convey("测试GetByID方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBObligation(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		obligationType := mock.NewMockObligationType(ctrl)
		ob := newObligation(db, userMgnt, obligationType)

		ctx := context.Background()
		visitor := &interfaces.Visitor{
			ID:   testObligationUserID,
			Type: interfaces.RealName,
		}

		Convey("用户权限检查失败", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.NormalUser}, nil)

			info, err := ob.GetByID(ctx, visitor, testObligationID)

			assert.Error(t, err)
			assert.Empty(t, info.ID)
		})

		Convey("数据库查询失败", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			db.EXPECT().GetByID(gomock.Any(), testObligationID).Return(interfaces.ObligationInfo{}, errors.New("数据库错误"))

			info, err := ob.GetByID(ctx, visitor, testObligationID)

			assert.Error(t, err)
			assert.Empty(t, info.ID)
		})

		Convey("义务不存在", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			db.EXPECT().GetByID(gomock.Any(), testObligationID).Return(interfaces.ObligationInfo{}, nil)

			info, err := ob.GetByID(ctx, visitor, testObligationID)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "not found")
			assert.Empty(t, info.ID)
		})

		Convey("成功获取义务", func() {
			expectedInfo := interfaces.ObligationInfo{
				ID:          testObligationID,
				TypeID:      testObligationTypeID,
				Name:        testObligationName,
				Description: testObligationDesc,
			}

			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			db.EXPECT().GetByID(gomock.Any(), testObligationID).Return(expectedInfo, nil)

			info, err := ob.GetByID(ctx, visitor, testObligationID)

			assert.NoError(t, err)
			assert.Equal(t, expectedInfo.ID, info.ID)
			assert.Equal(t, expectedInfo.TypeID, info.TypeID)
			assert.Equal(t, expectedInfo.Name, info.Name)
		})
	})
}

func TestObligation_Get(t *testing.T) {
	Convey("测试Get方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBObligation(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		obligationType := mock.NewMockObligationType(ctrl)
		ob := newObligation(db, userMgnt, obligationType)

		ctx := context.Background()
		visitor := &interfaces.Visitor{
			ID:   testObligationUserID,
			Type: interfaces.RealName,
		}

		searchInfo := &interfaces.ObligationSearchInfo{
			Offset: 0,
			Limit:  10,
		}

		Convey("用户权限检查失败", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.NormalUser}, nil)

			count, infos, err := ob.Get(ctx, visitor, searchInfo)

			assert.Error(t, err)
			assert.Equal(t, 0, count)
			assert.Nil(t, infos)
		})

		Convey("数据库查询失败", func() {
			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			db.EXPECT().Get(gomock.Any(), searchInfo).Return(0, nil, errors.New("数据库错误"))

			count, infos, err := ob.Get(ctx, visitor, searchInfo)

			assert.Error(t, err)
			assert.Equal(t, 0, count)
			assert.Nil(t, infos)
		})

		Convey("成功获取义务列表", func() {
			expectedInfos := []interfaces.ObligationInfo{
				{
					ID:          testObligationID,
					TypeID:      testObligationTypeID,
					Name:        testObligationName,
					Description: testObligationDesc,
				},
				{
					ID:          testObligationID2,
					TypeID:      testObligationTypeID,
					Name:        "name2",
					Description: "desc2",
				},
			}

			userMgnt.EXPECT().GetUserRolesByUserID(gomock.Any(), testObligationUserID).Return([]interfaces.SystemRoleType{interfaces.SuperAdmin}, nil)
			db.EXPECT().Get(gomock.Any(), searchInfo).Return(2, expectedInfos, nil)

			count, infos, err := ob.Get(ctx, visitor, searchInfo)

			assert.NoError(t, err)
			assert.Equal(t, 2, count)
			assert.Equal(t, 2, len(infos))
			assert.Equal(t, testObligationID, infos[0].ID)
			assert.Equal(t, testObligationID2, infos[1].ID)
		})
	})
}

//nolint:funlen
func TestObligation_Query(t *testing.T) {
	Convey("测试Query方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBObligation(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		obligationType := mock.NewMockObligationType(ctrl)
		ob := newObligation(db, userMgnt, obligationType)

		ctx := context.Background()
		visitor := &interfaces.Visitor{
			ID:   testObligationUserID,
			Type: interfaces.RealName,
		}

		queryInfo := &interfaces.QueryObligationInfo{
			ResourceType: testResourceType,
			Operation:    []string{testOperationID},
		}

		Convey("查询义务类型失败", func() {
			obligationType.EXPECT().Query(gomock.Any(), visitor, gomock.Any()).Return(nil, errors.New("查询失败"))

			result, err := ob.Query(ctx, visitor, queryInfo)

			assert.Error(t, err)
			assert.Nil(t, result)
		})

		Convey("无匹配的义务类型", func() {
			obligationType.EXPECT().Query(gomock.Any(), visitor, gomock.Any()).Return(map[string][]interfaces.ObligationTypeInfo{}, nil)

			result, err := ob.Query(ctx, visitor, queryInfo)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, 0, len(result))
		})

		Convey("根据义务类型ID过滤且无匹配", func() {
			queryInfoWithTypeIDs := &interfaces.QueryObligationInfo{
				ResourceType:      testResourceType,
				Operation:         []string{testOperationID},
				ObligationTypeIDs: []string{"non_existing_type_id"},
			}
			operationAndTypes := map[string][]interfaces.ObligationTypeInfo{
				testOperationID: {
					{
						ID:   testObligationTypeID,
						Name: "type1",
					},
				},
			}

			obligationType.EXPECT().Query(gomock.Any(), visitor, gomock.Any()).Return(operationAndTypes, nil)

			result, err := ob.Query(ctx, visitor, queryInfoWithTypeIDs)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, 0, len(result[testOperationID]))
		})

		Convey("获取义务失败", func() {
			operationAndTypes := map[string][]interfaces.ObligationTypeInfo{
				testOperationID: {
					{
						ID:   testObligationTypeID,
						Name: "type1",
					},
				},
			}

			obligationType.EXPECT().Query(gomock.Any(), visitor, gomock.Any()).Return(operationAndTypes, nil)
			db.EXPECT().GetByObligationTypeIDs(gomock.Any(), gomock.Any()).Return(nil, errors.New("数据库错误"))

			_, err := ob.Query(ctx, visitor, queryInfo)

			assert.Error(t, err)
		})

		Convey("成功查询义务", func() {
			operationAndTypes := map[string][]interfaces.ObligationTypeInfo{
				testOperationID: {
					{
						ID:   testObligationTypeID,
						Name: "type1",
					},
				},
			}
			typeAndObligations := map[string][]interfaces.ObligationInfo{
				testObligationTypeID: {
					{
						ID:          testObligationID,
						TypeID:      testObligationTypeID,
						Name:        testObligationName,
						Description: testObligationDesc,
					},
				},
			}

			obligationType.EXPECT().Query(gomock.Any(), visitor, gomock.Any()).Return(operationAndTypes, nil)
			db.EXPECT().GetByObligationTypeIDs(gomock.Any(), gomock.Any()).Return(typeAndObligations, nil)

			result, err := ob.Query(ctx, visitor, queryInfo)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Contains(t, result, testOperationID)
			assert.Equal(t, 1, len(result[testOperationID]))
			assert.Equal(t, testObligationID, result[testOperationID][0].ID)
		})

		Convey("成功查询义务（根据义务类型ID过滤）", func() {
			queryInfoWithTypeIDs := &interfaces.QueryObligationInfo{
				ResourceType:      testResourceType,
				Operation:         []string{testOperationID},
				ObligationTypeIDs: []string{testObligationTypeID},
			}
			operationAndTypes := map[string][]interfaces.ObligationTypeInfo{
				testOperationID: {
					{
						ID:   testObligationTypeID,
						Name: "type1",
					},
					{
						ID:   "other_type_id",
						Name: "type2",
					},
				},
			}
			typeAndObligations := map[string][]interfaces.ObligationInfo{
				testObligationTypeID: {
					{
						ID:          testObligationID,
						TypeID:      testObligationTypeID,
						Name:        testObligationName,
						Description: testObligationDesc,
					},
				},
			}

			obligationType.EXPECT().Query(gomock.Any(), visitor, gomock.Any()).Return(operationAndTypes, nil)
			db.EXPECT().GetByObligationTypeIDs(gomock.Any(), gomock.Any()).Return(typeAndObligations, nil)

			result, err := ob.Query(ctx, visitor, queryInfoWithTypeIDs)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Contains(t, result, testOperationID)
			assert.Equal(t, 1, len(result[testOperationID]))
			assert.Equal(t, testObligationID, result[testOperationID][0].ID)
		})

		Convey("成功查询义务（多个操作和义务类型）", func() {
			operationID2 := "operation_id_2"
			obligationTypeID2 := "obligation_type_id_2"

			operationAndTypes := map[string][]interfaces.ObligationTypeInfo{
				testOperationID: {
					{
						ID:   testObligationTypeID,
						Name: "type1",
					},
				},
				operationID2: {
					{
						ID:   obligationTypeID2,
						Name: "type2",
					},
				},
			}
			typeAndObligations := map[string][]interfaces.ObligationInfo{
				testObligationTypeID: {
					{
						ID:     testObligationID,
						TypeID: testObligationTypeID,
						Name:   testObligationName,
					},
				},
				obligationTypeID2: {
					{
						ID:     testObligationID2,
						TypeID: obligationTypeID2,
						Name:   "name2",
					},
				},
			}

			obligationType.EXPECT().Query(gomock.Any(), visitor, gomock.Any()).Return(operationAndTypes, nil)
			db.EXPECT().GetByObligationTypeIDs(gomock.Any(), gomock.Any()).Return(typeAndObligations, nil)

			result, err := ob.Query(ctx, visitor, queryInfo)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Contains(t, result, testOperationID)
			assert.Contains(t, result, operationID2)
			assert.Equal(t, 1, len(result[testOperationID]))
			assert.Equal(t, 1, len(result[operationID2]))
			assert.Equal(t, testObligationID, result[testOperationID][0].ID)
			assert.Equal(t, testObligationID2, result[operationID2][0].ID)
		})
	})
}

func TestObligation_GetByIDSInternal(t *testing.T) {
	Convey("测试GetByIDSInternal方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBObligation(ctrl)
		userMgnt := mock.NewMockDrivenUserMgnt(ctrl)
		obligationType := mock.NewMockObligationType(ctrl)
		ob := newObligation(db, userMgnt, obligationType)

		ctx := context.Background()

		Convey("数据库查询失败", func() {
			obligationIDs := map[string]bool{
				testObligationID: true,
			}

			db.EXPECT().GetByIDs(gomock.Any(), gomock.Any()).Return(nil, errors.New("数据库错误"))

			infos, err := ob.GetByIDSInternal(ctx, obligationIDs)

			assert.Error(t, err)
			assert.Nil(t, infos)
		})

		Convey("成功获取义务列表", func() {
			obligationIDs := map[string]bool{
				testObligationID:  true,
				testObligationID2: true,
			}
			expectedInfos := []interfaces.ObligationInfo{
				{
					ID:          testObligationID,
					TypeID:      testObligationTypeID,
					Name:        testObligationName,
					Description: testObligationDesc,
				},
				{
					ID:          testObligationID2,
					TypeID:      testObligationTypeID,
					Name:        "name2",
					Description: "desc2",
				},
			}

			db.EXPECT().GetByIDs(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, ids []string) ([]interfaces.ObligationInfo, error) {
				// 验证传入的IDs
				assert.Equal(t, 2, len(ids))
				return expectedInfos, nil
			})

			infos, err := ob.GetByIDSInternal(ctx, obligationIDs)

			assert.NoError(t, err)
			assert.Equal(t, 2, len(infos))
			assert.Equal(t, testObligationID, infos[0].ID)
			assert.Equal(t, testObligationID2, infos[1].ID)
		})

		Convey("空ID映射", func() {
			obligationIDs := map[string]bool{}
			expectedInfos := []interfaces.ObligationInfo{}

			db.EXPECT().GetByIDs(gomock.Any(), gomock.Any()).Return(expectedInfos, nil)

			infos, err := ob.GetByIDSInternal(ctx, obligationIDs)

			assert.NoError(t, err)
			assert.Equal(t, 0, len(infos))
		})
	})
}
