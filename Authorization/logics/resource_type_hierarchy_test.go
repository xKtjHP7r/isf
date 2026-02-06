package logics

import (
	"context"
	"errors"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/Authorization/common"
	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/Authorization/interfaces"
	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/Authorization/interfaces/mock"
)

func newResourceTypeHierarchy(db interfaces.DBResourceTypeHierarchy, dbResourceType interfaces.DBResourceType) *resourceTypeHierarchy {
	return &resourceTypeHierarchy{
		db:             db,
		dbResourceType: dbResourceType,
		logger:         common.NewLogger(),
	}
}

func TestResourceTypeHierarchy_SetPrivate(t *testing.T) {
	Convey("测试SetPrivate方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBResourceTypeHierarchy(ctrl)
		dbResourceType := mock.NewMockDBResourceType(ctrl)
		r := newResourceTypeHierarchy(db, dbResourceType)

		ctx := context.Background()
		visitor := &interfaces.Visitor{
			ID:   testUserID,
			Type: interfaces.RealName,
		}

		hierarchy := &interfaces.ResourceTypeHierarchy{
			ResourceTypeID: testResourceTypeID,
			Children: []interfaces.ResourceTypeHierarchy{
				{
					ResourceTypeID: "child-type-1",
					Children:       []interfaces.ResourceTypeHierarchy{},
				},
			},
		}

		Convey("checkDuplicate失败-资源类型不存在", func() {
			dbResourceType.EXPECT().GetAllInternalWithHidden(gomock.Any()).Return([]interfaces.ResourceType{}, nil)

			err := r.SetPrivate(ctx, visitor, hierarchy)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "resource type not found")
		})

		Convey("checkDuplicate失败-子资源类型不存在", func() {
			resourceTypes := []interfaces.ResourceType{
				{
					ID:   testResourceTypeID,
					Name: "测试资源类型",
				},
			}
			dbResourceType.EXPECT().GetAllInternalWithHidden(gomock.Any()).Return(resourceTypes, nil)

			err := r.SetPrivate(ctx, visitor, hierarchy)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "resource type not found")
		})

		Convey("checkDuplicate失败-同一层级有重复", func() {
			resourceTypes := []interfaces.ResourceType{
				{
					ID:   testResourceTypeID,
					Name: "测试资源类型",
				},
				{
					ID:   "child-type-1",
					Name: "子资源类型1",
				},
			}
			duplicateHierarchy := &interfaces.ResourceTypeHierarchy{
				ResourceTypeID: testResourceTypeID,
				Children: []interfaces.ResourceTypeHierarchy{
					{
						ResourceTypeID: "child-type-1",
						Children:       []interfaces.ResourceTypeHierarchy{},
					},
					{
						ResourceTypeID: "child-type-1", // 重复
						Children:       []interfaces.ResourceTypeHierarchy{},
					},
				},
			}
			dbResourceType.EXPECT().GetAllInternalWithHidden(gomock.Any()).Return(resourceTypes, nil)

			err := r.SetPrivate(ctx, visitor, duplicateHierarchy)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "同一层级不能有重复的资源类型")
		})

		Convey("checkDuplicate失败-嵌套层级有重复", func() {
			resourceTypes := []interfaces.ResourceType{
				{
					ID:   testResourceTypeID,
					Name: "测试资源类型",
				},
				{
					ID:   "child-type-1",
					Name: "子资源类型1",
				},
				{
					ID:   "grandchild-type-1",
					Name: "孙资源类型1",
				},
			}
			nestedHierarchy := &interfaces.ResourceTypeHierarchy{
				ResourceTypeID: testResourceTypeID,
				Children: []interfaces.ResourceTypeHierarchy{
					{
						ResourceTypeID: "child-type-1",
						Children: []interfaces.ResourceTypeHierarchy{
							{
								ResourceTypeID: "grandchild-type-1",
								Children:       []interfaces.ResourceTypeHierarchy{},
							},
							{
								ResourceTypeID: "grandchild-type-1", // 重复
								Children:       []interfaces.ResourceTypeHierarchy{},
							},
						},
					},
				},
			}
			// checkDuplicate 会递归调用，根节点1次，子节点1次，共2次
			dbResourceType.EXPECT().GetAllInternalWithHidden(gomock.Any()).Return(resourceTypes, nil).Times(2)

			err := r.SetPrivate(ctx, visitor, nestedHierarchy)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "同一层级不能有重复的资源类型")
		})

		Convey("db.Set失败", func() {
			resourceTypes := []interfaces.ResourceType{
				{
					ID:   testResourceTypeID,
					Name: "测试资源类型",
				},
				{
					ID:   "child-type-1",
					Name: "子资源类型1",
				},
			}
			// checkDuplicate 会递归调用：根节点1次，子节点1次，共2次
			dbResourceType.EXPECT().GetAllInternalWithHidden(gomock.Any()).Return(resourceTypes, nil).Times(2)
			db.EXPECT().Set(gomock.Any(), hierarchy).Return(errors.New("数据库保存失败"))

			err := r.SetPrivate(ctx, visitor, hierarchy)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "数据库保存失败")
		})

		Convey("成功设置层级关系", func() {
			resourceTypes := []interfaces.ResourceType{
				{
					ID:   testResourceTypeID,
					Name: "测试资源类型",
				},
				{
					ID:   "child-type-1",
					Name: "子资源类型1",
				},
			}
			// checkDuplicate 会递归调用：根节点1次，子节点1次，共2次
			dbResourceType.EXPECT().GetAllInternalWithHidden(gomock.Any()).Return(resourceTypes, nil).Times(2)
			db.EXPECT().Set(gomock.Any(), hierarchy).Return(nil)

			err := r.SetPrivate(ctx, visitor, hierarchy)
			assert.NoError(t, err)
		})

		Convey("成功设置复杂嵌套层级关系", func() {
			complexHierarchy := &interfaces.ResourceTypeHierarchy{
				ResourceTypeID: testResourceTypeID,
				Children: []interfaces.ResourceTypeHierarchy{
					{
						ResourceTypeID: "child-type-1",
						Children: []interfaces.ResourceTypeHierarchy{
							{
								ResourceTypeID: "grandchild-type-1",
								Children:       []interfaces.ResourceTypeHierarchy{},
							},
						},
					},
					{
						ResourceTypeID: "child-type-2",
						Children:       []interfaces.ResourceTypeHierarchy{},
					},
				},
			}
			resourceTypes := []interfaces.ResourceType{
				{
					ID:   testResourceTypeID,
					Name: "测试资源类型",
				},
				{
					ID:   "child-type-1",
					Name: "子资源类型1",
				},
				{
					ID:   "child-type-2",
					Name: "子资源类型2",
				},
				{
					ID:   "grandchild-type-1",
					Name: "孙资源类型1",
				},
			}
			// checkDuplicate 会递归调用：根节点1次，child-type-1子节点1次，grandchild-type-1子节点1次，child-type-2子节点1次，共4次
			dbResourceType.EXPECT().GetAllInternalWithHidden(gomock.Any()).Return(resourceTypes, nil).Times(4)
			db.EXPECT().Set(gomock.Any(), complexHierarchy).Return(nil)

			err := r.SetPrivate(ctx, visitor, complexHierarchy)
			assert.NoError(t, err)
		})
	})
}

func TestResourceTypeHierarchy_Init(t *testing.T) {
	Convey("测试Init方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBResourceTypeHierarchy(ctrl)
		dbResourceType := mock.NewMockDBResourceType(ctrl)
		r := newResourceTypeHierarchy(db, dbResourceType)

		ctx := context.Background()

		Convey("成功初始化", func() {
			// init_data/resource_type_hierarchy.json 中 menu 有子节点 menu
			resourceTypes := []interfaces.ResourceType{
				{ID: "menu", Name: "菜单"},
			}
			// checkDuplicate 根节点1次、子节点1次，共2次
			dbResourceType.EXPECT().GetAllInternalWithHidden(gomock.Any()).Return(resourceTypes, nil).Times(2)
			db.EXPECT().Set(gomock.Any(), gomock.Any()).Return(nil)

			err := r.Init(ctx)
			assert.NoError(t, err)
		})

		Convey("SetPrivate失败-checkDuplicate资源类型不存在", func() {
			// init_data 中只有 menu，若 menu 不在 resourceTypes 中则 checkDuplicate 会失败
			dbResourceType.EXPECT().GetAllInternalWithHidden(gomock.Any()).Return([]interfaces.ResourceType{}, nil)

			err := r.Init(ctx)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "resource type not found")
		})

		Convey("SetPrivate失败-db.Set失败", func() {
			resourceTypes := []interfaces.ResourceType{
				{ID: "menu", Name: "菜单"},
			}
			dbResourceType.EXPECT().GetAllInternalWithHidden(gomock.Any()).Return(resourceTypes, nil).Times(2)
			db.EXPECT().Set(gomock.Any(), gomock.Any()).Return(errors.New("数据库保存失败"))

			err := r.Init(ctx)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "数据库保存失败")
		})
	})
}

func TestResourceTypeHierarchy_Get(t *testing.T) {
	Convey("测试Get方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBResourceTypeHierarchy(ctrl)
		dbResourceType := mock.NewMockDBResourceType(ctrl)
		r := newResourceTypeHierarchy(db, dbResourceType)

		ctx := context.Background()
		visitor := &interfaces.Visitor{
			ID:   testUserID,
			Type: interfaces.RealName,
		}

		Convey("db.GetAll失败", func() {
			db.EXPECT().GetAll(gomock.Any()).Return(nil, errors.New("数据库查询失败"))

			hierarchy, err := r.Get(ctx, visitor, testResourceTypeID)
			assert.Error(t, err)
			assert.Equal(t, hierarchy.ResourceTypeID, "")
		})

		Convey("找到根节点", func() {
			expectedHierarchy := interfaces.ResourceTypeHierarchy{
				ResourceTypeID: testResourceTypeID,
				Children: []interfaces.ResourceTypeHierarchy{
					{
						ResourceTypeID: "child-type-1",
						Children:       []interfaces.ResourceTypeHierarchy{},
					},
				},
			}
			hierarchyMap := map[string]interfaces.ResourceTypeHierarchy{
				testResourceTypeID: expectedHierarchy,
			}
			db.EXPECT().GetAll(gomock.Any()).Return(hierarchyMap, nil)

			hierarchy, err := r.Get(ctx, visitor, testResourceTypeID)
			assert.NoError(t, err)
			assert.Equal(t, hierarchy.ResourceTypeID, testResourceTypeID)
			assert.Equal(t, len(hierarchy.Children), 1)
		})

		Convey("在第一层子节点中找到", func() {
			rootHierarchy := interfaces.ResourceTypeHierarchy{
				ResourceTypeID: "root-type",
				Children: []interfaces.ResourceTypeHierarchy{
					{
						ResourceTypeID: testResourceTypeID,
						Children:       []interfaces.ResourceTypeHierarchy{},
					},
				},
			}
			hierarchyMap := map[string]interfaces.ResourceTypeHierarchy{
				"root-type": rootHierarchy,
			}
			db.EXPECT().GetAll(gomock.Any()).Return(hierarchyMap, nil)

			hierarchy, err := r.Get(ctx, visitor, testResourceTypeID)
			assert.NoError(t, err)
			assert.Equal(t, hierarchy.ResourceTypeID, testResourceTypeID)
			assert.Equal(t, len(hierarchy.Children), 0)
		})

		Convey("在深层嵌套子节点中找到", func() {
			rootHierarchy := interfaces.ResourceTypeHierarchy{
				ResourceTypeID: "root-type",
				Children: []interfaces.ResourceTypeHierarchy{
					{
						ResourceTypeID: "child-type-1",
						Children: []interfaces.ResourceTypeHierarchy{
							{
								ResourceTypeID: "grandchild-type-1",
								Children: []interfaces.ResourceTypeHierarchy{
									{
										ResourceTypeID: testResourceTypeID,
										Children:       []interfaces.ResourceTypeHierarchy{},
									},
								},
							},
						},
					},
				},
			}
			hierarchyMap := map[string]interfaces.ResourceTypeHierarchy{
				"root-type": rootHierarchy,
			}
			db.EXPECT().GetAll(gomock.Any()).Return(hierarchyMap, nil)

			hierarchy, err := r.Get(ctx, visitor, testResourceTypeID)
			assert.NoError(t, err)
			assert.Equal(t, hierarchy.ResourceTypeID, testResourceTypeID)
			assert.Equal(t, len(hierarchy.Children), 0)
		})

		Convey("在多个根节点中查找", func() {
			rootHierarchy1 := interfaces.ResourceTypeHierarchy{
				ResourceTypeID: "root-type-1",
				Children: []interfaces.ResourceTypeHierarchy{
					{
						ResourceTypeID: "child-type-1",
						Children:       []interfaces.ResourceTypeHierarchy{},
					},
				},
			}
			rootHierarchy2 := interfaces.ResourceTypeHierarchy{
				ResourceTypeID: "root-type-2",
				Children: []interfaces.ResourceTypeHierarchy{
					{
						ResourceTypeID: testResourceTypeID,
						Children:       []interfaces.ResourceTypeHierarchy{},
					},
				},
			}
			hierarchyMap := map[string]interfaces.ResourceTypeHierarchy{
				"root-type-1": rootHierarchy1,
				"root-type-2": rootHierarchy2,
			}
			db.EXPECT().GetAll(gomock.Any()).Return(hierarchyMap, nil)

			hierarchy, err := r.Get(ctx, visitor, testResourceTypeID)
			assert.NoError(t, err)
			assert.Equal(t, hierarchy.ResourceTypeID, testResourceTypeID)
		})

		Convey("未找到资源类型", func() {
			rootHierarchy := interfaces.ResourceTypeHierarchy{
				ResourceTypeID: "root-type",
				Children: []interfaces.ResourceTypeHierarchy{
					{
						ResourceTypeID: "child-type-1",
						Children:       []interfaces.ResourceTypeHierarchy{},
					},
				},
			}
			hierarchyMap := map[string]interfaces.ResourceTypeHierarchy{
				"root-type": rootHierarchy,
			}
			db.EXPECT().GetAll(gomock.Any()).Return(hierarchyMap, nil)

			hierarchy, err := r.Get(ctx, visitor, testResourceTypeID)
			assert.NoError(t, err)
			assert.Equal(t, hierarchy.ResourceTypeID, "")
		})

		Convey("空层级关系映射", func() {
			hierarchyMap := map[string]interfaces.ResourceTypeHierarchy{}
			db.EXPECT().GetAll(gomock.Any()).Return(hierarchyMap, nil)

			hierarchy, err := r.Get(ctx, visitor, testResourceTypeID)
			assert.NoError(t, err)
			assert.Equal(t, hierarchy.ResourceTypeID, "")
		})
	})
}

func TestResourceTypeHierarchy_FindInChildren(t *testing.T) {
	Convey("测试findInChildren方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBResourceTypeHierarchy(ctrl)
		dbResourceType := mock.NewMockDBResourceType(ctrl)
		r := newResourceTypeHierarchy(db, dbResourceType)

		Convey("空子节点列表", func() {
			children := []interfaces.ResourceTypeHierarchy{}
			found, ok := r.findInChildren(children, testResourceTypeID)
			assert.False(t, ok)
			assert.Equal(t, found.ResourceTypeID, "")
		})

		Convey("在第一层找到", func() {
			children := []interfaces.ResourceTypeHierarchy{
				{
					ResourceTypeID: testResourceTypeID,
					Children:       []interfaces.ResourceTypeHierarchy{},
				},
				{
					ResourceTypeID: "other-type",
					Children:       []interfaces.ResourceTypeHierarchy{},
				},
			}
			found, ok := r.findInChildren(children, testResourceTypeID)
			assert.True(t, ok)
			assert.Equal(t, found.ResourceTypeID, testResourceTypeID)
		})

		Convey("在深层嵌套中找到", func() {
			children := []interfaces.ResourceTypeHierarchy{
				{
					ResourceTypeID: "child-type-1",
					Children: []interfaces.ResourceTypeHierarchy{
						{
							ResourceTypeID: "grandchild-type-1",
							Children: []interfaces.ResourceTypeHierarchy{
								{
									ResourceTypeID: testResourceTypeID,
									Children:       []interfaces.ResourceTypeHierarchy{},
								},
							},
						},
					},
				},
			}
			found, ok := r.findInChildren(children, testResourceTypeID)
			assert.True(t, ok)
			assert.Equal(t, found.ResourceTypeID, testResourceTypeID)
		})

		Convey("未找到", func() {
			children := []interfaces.ResourceTypeHierarchy{
				{
					ResourceTypeID: "child-type-1",
					Children: []interfaces.ResourceTypeHierarchy{
						{
							ResourceTypeID: "grandchild-type-1",
							Children:       []interfaces.ResourceTypeHierarchy{},
						},
					},
				},
			}
			found, ok := r.findInChildren(children, testResourceTypeID)
			assert.False(t, ok)
			assert.Equal(t, found.ResourceTypeID, "")
		})

		Convey("在多个分支中查找，找到第一个匹配的", func() {
			children := []interfaces.ResourceTypeHierarchy{
				{
					ResourceTypeID: "child-type-1",
					Children: []interfaces.ResourceTypeHierarchy{
						{
							ResourceTypeID: testResourceTypeID,
							Children:       []interfaces.ResourceTypeHierarchy{},
						},
					},
				},
				{
					ResourceTypeID: "child-type-2",
					Children: []interfaces.ResourceTypeHierarchy{
						{
							ResourceTypeID: testResourceTypeID, // 第二个匹配，但不会返回这个
							Children:       []interfaces.ResourceTypeHierarchy{},
						},
					},
				},
			}
			found, ok := r.findInChildren(children, testResourceTypeID)
			assert.True(t, ok)
			assert.Equal(t, found.ResourceTypeID, testResourceTypeID)
			// 应该返回第一个找到的（在child-type-1下）
		})
	})
}

func TestResourceTypeHierarchy_HasHierarchy(t *testing.T) {
	Convey("测试HasHierarchy方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBResourceTypeHierarchy(ctrl)
		dbResourceType := mock.NewMockDBResourceType(ctrl)
		r := newResourceTypeHierarchy(db, dbResourceType)

		ctx := context.Background()
		visitor := &interfaces.Visitor{
			ID:   testUserID,
			Type: interfaces.RealName,
		}

		Convey("db.GetAll失败", func() {
			db.EXPECT().GetAll(gomock.Any()).Return(nil, errors.New("数据库查询失败"))

			hasHierarchy, err := r.HasHierarchy(ctx, visitor, testResourceTypeID)
			assert.Error(t, err)
			assert.False(t, hasHierarchy)
		})

		Convey("resourceTypeID是根节点-有层级关系", func() {
			hierarchyMap := map[string]interfaces.ResourceTypeHierarchy{
				testResourceTypeID: {
					ResourceTypeID: testResourceTypeID,
					Children: []interfaces.ResourceTypeHierarchy{
						{
							ResourceTypeID: "child-type-1",
							Children:       []interfaces.ResourceTypeHierarchy{},
						},
					},
				},
			}
			db.EXPECT().GetAll(gomock.Any()).Return(hierarchyMap, nil)

			hasHierarchy, err := r.HasHierarchy(ctx, visitor, testResourceTypeID)
			assert.NoError(t, err)
			assert.True(t, hasHierarchy)
		})

		Convey("resourceTypeID在第一层子节点中", func() {
			rootHierarchy := interfaces.ResourceTypeHierarchy{
				ResourceTypeID: "root-type",
				Children: []interfaces.ResourceTypeHierarchy{
					{
						ResourceTypeID: testResourceTypeID,
						Children:       []interfaces.ResourceTypeHierarchy{},
					},
				},
			}
			hierarchyMap := map[string]interfaces.ResourceTypeHierarchy{
				"root-type": rootHierarchy,
			}
			db.EXPECT().GetAll(gomock.Any()).Return(hierarchyMap, nil)

			hasHierarchy, err := r.HasHierarchy(ctx, visitor, testResourceTypeID)
			assert.NoError(t, err)
			assert.True(t, hasHierarchy)
		})

		Convey("resourceTypeID在深层嵌套子节点中", func() {
			rootHierarchy := interfaces.ResourceTypeHierarchy{
				ResourceTypeID: "root-type",
				Children: []interfaces.ResourceTypeHierarchy{
					{
						ResourceTypeID: "child-type-1",
						Children: []interfaces.ResourceTypeHierarchy{
							{
								ResourceTypeID: "grandchild-type-1",
								Children: []interfaces.ResourceTypeHierarchy{
									{
										ResourceTypeID: testResourceTypeID,
										Children:       []interfaces.ResourceTypeHierarchy{},
									},
								},
							},
						},
					},
				},
			}
			hierarchyMap := map[string]interfaces.ResourceTypeHierarchy{
				"root-type": rootHierarchy,
			}
			db.EXPECT().GetAll(gomock.Any()).Return(hierarchyMap, nil)

			hasHierarchy, err := r.HasHierarchy(ctx, visitor, testResourceTypeID)
			assert.NoError(t, err)
			assert.True(t, hasHierarchy)
		})

		Convey("resourceTypeID不存在于任何层级中", func() {
			rootHierarchy := interfaces.ResourceTypeHierarchy{
				ResourceTypeID: "root-type",
				Children: []interfaces.ResourceTypeHierarchy{
					{
						ResourceTypeID: "child-type-1",
						Children:       []interfaces.ResourceTypeHierarchy{},
					},
				},
			}
			hierarchyMap := map[string]interfaces.ResourceTypeHierarchy{
				"root-type": rootHierarchy,
			}
			db.EXPECT().GetAll(gomock.Any()).Return(hierarchyMap, nil)

			hasHierarchy, err := r.HasHierarchy(ctx, visitor, testResourceTypeID)
			assert.NoError(t, err)
			assert.False(t, hasHierarchy)
		})

		Convey("空层级关系映射", func() {
			hierarchyMap := map[string]interfaces.ResourceTypeHierarchy{}
			db.EXPECT().GetAll(gomock.Any()).Return(hierarchyMap, nil)

			hasHierarchy, err := r.HasHierarchy(ctx, visitor, testResourceTypeID)
			assert.NoError(t, err)
			assert.False(t, hasHierarchy)
		})

		Convey("在多个根节点中查找-在第二个根的子节点中找到", func() {
			rootHierarchy1 := interfaces.ResourceTypeHierarchy{
				ResourceTypeID: "root-type-1",
				Children: []interfaces.ResourceTypeHierarchy{
					{
						ResourceTypeID: "child-type-1",
						Children:       []interfaces.ResourceTypeHierarchy{},
					},
				},
			}
			rootHierarchy2 := interfaces.ResourceTypeHierarchy{
				ResourceTypeID: "root-type-2",
				Children: []interfaces.ResourceTypeHierarchy{
					{
						ResourceTypeID: testResourceTypeID,
						Children:       []interfaces.ResourceTypeHierarchy{},
					},
				},
			}
			hierarchyMap := map[string]interfaces.ResourceTypeHierarchy{
				"root-type-1": rootHierarchy1,
				"root-type-2": rootHierarchy2,
			}
			db.EXPECT().GetAll(gomock.Any()).Return(hierarchyMap, nil)

			hasHierarchy, err := r.HasHierarchy(ctx, visitor, testResourceTypeID)
			assert.NoError(t, err)
			assert.True(t, hasHierarchy)
		})
	})
}

func TestResourceTypeHierarchy_CheckDuplicate(t *testing.T) {
	Convey("测试checkDuplicate方法", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := mock.NewMockDBResourceTypeHierarchy(ctrl)
		dbResourceType := mock.NewMockDBResourceType(ctrl)
		r := newResourceTypeHierarchy(db, dbResourceType)

		ctx := context.Background()

		Convey("GetAllInternalWithHidden失败", func() {
			hierarchy := &interfaces.ResourceTypeHierarchy{
				ResourceTypeID: testResourceTypeID,
				Children:       []interfaces.ResourceTypeHierarchy{},
			}
			dbResourceType.EXPECT().GetAllInternalWithHidden(gomock.Any()).Return(nil, errors.New("查询失败"))

			err := r.checkDuplicate(ctx, hierarchy)
			assert.Error(t, err)
		})

		Convey("根节点资源类型不存在", func() {
			hierarchy := &interfaces.ResourceTypeHierarchy{
				ResourceTypeID: testResourceTypeID,
				Children:       []interfaces.ResourceTypeHierarchy{},
			}
			dbResourceType.EXPECT().GetAllInternalWithHidden(gomock.Any()).Return([]interfaces.ResourceType{}, nil)

			err := r.checkDuplicate(ctx, hierarchy)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "resource type not found")
		})

		Convey("子节点资源类型不存在", func() {
			hierarchy := &interfaces.ResourceTypeHierarchy{
				ResourceTypeID: testResourceTypeID,
				Children: []interfaces.ResourceTypeHierarchy{
					{
						ResourceTypeID: "child-type-1",
						Children:       []interfaces.ResourceTypeHierarchy{},
					},
				},
			}
			resourceTypes := []interfaces.ResourceType{
				{
					ID:   testResourceTypeID,
					Name: "测试资源类型",
				},
			}
			dbResourceType.EXPECT().GetAllInternalWithHidden(gomock.Any()).Return(resourceTypes, nil)

			err := r.checkDuplicate(ctx, hierarchy)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "resource type not found")
		})

		Convey("同一层级有重复", func() {
			hierarchy := &interfaces.ResourceTypeHierarchy{
				ResourceTypeID: testResourceTypeID,
				Children: []interfaces.ResourceTypeHierarchy{
					{
						ResourceTypeID: "child-type-1",
						Children:       []interfaces.ResourceTypeHierarchy{},
					},
					{
						ResourceTypeID: "child-type-1", // 重复
						Children:       []interfaces.ResourceTypeHierarchy{},
					},
				},
			}
			resourceTypes := []interfaces.ResourceType{
				{
					ID:   testResourceTypeID,
					Name: "测试资源类型",
				},
				{
					ID:   "child-type-1",
					Name: "子资源类型1",
				},
			}
			dbResourceType.EXPECT().GetAllInternalWithHidden(gomock.Any()).Return(resourceTypes, nil)

			err := r.checkDuplicate(ctx, hierarchy)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "同一层级不能有重复的资源类型")
		})

		Convey("成功-无重复", func() {
			hierarchy := &interfaces.ResourceTypeHierarchy{
				ResourceTypeID: testResourceTypeID,
				Children: []interfaces.ResourceTypeHierarchy{
					{
						ResourceTypeID: "child-type-1",
						Children:       []interfaces.ResourceTypeHierarchy{},
					},
					{
						ResourceTypeID: "child-type-2",
						Children:       []interfaces.ResourceTypeHierarchy{},
					},
				},
			}
			resourceTypes := []interfaces.ResourceType{
				{
					ID:   testResourceTypeID,
					Name: "测试资源类型",
				},
				{
					ID:   "child-type-1",
					Name: "子资源类型1",
				},
				{
					ID:   "child-type-2",
					Name: "子资源类型2",
				},
			}
			// checkDuplicate 会递归调用：根节点1次，child-type-1子节点1次，child-type-2子节点1次，共3次
			dbResourceType.EXPECT().GetAllInternalWithHidden(gomock.Any()).Return(resourceTypes, nil).Times(3)

			err := r.checkDuplicate(ctx, hierarchy)
			assert.NoError(t, err)
		})
	})
}
