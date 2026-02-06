package dbaccess

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/proton-rds-sdk-go/sqlx"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-playground/assert"
	. "github.com/smartystreets/goconvey/convey"

	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/Authorization/common"
	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/Authorization/interfaces"
)

func TestNewResourceTypeHierarchy(t *testing.T) {
	Convey("TestNewResourceTypeHierarchy", t, func() {
		Convey("singleton pattern", func() {
			// 注意：由于使用了单例模式，需要重置才能测试
			// 在实际测试中，可能需要重置 rthOnce 和 rthDB
			instance1 := NewResourceTypeHierarchy()
			instance2 := NewResourceTypeHierarchy()
			assert.Equal(t, instance1, instance2)
		})
	})
}

const (
	resourceTypeHierarchyTestResourceTypeID = "type-1"
)

func TestResourceTypeHierarchy_Set(t *testing.T) {
	Convey("TestResourceTypeHierarchy_Set", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func(db *sqlx.DB) {
			err := db.Close()
			if err != nil {
				return
			}
		}(db)

		ctx := context.Background()
		r := &resourceTypeHierarchy{
			db:     db,
			logger: common.NewLogger(),
		}

		resourceTypeID := resourceTypeHierarchyTestResourceTypeID
		hierarchy := &interfaces.ResourceTypeHierarchy{
			ResourceTypeID: resourceTypeID,
			Children: []interfaces.ResourceTypeHierarchy{
				{
					ResourceTypeID: "child-type-1",
					Children:       []interfaces.ResourceTypeHierarchy{},
				},
				{
					ResourceTypeID: "child-type-2",
					Children: []interfaces.ResourceTypeHierarchy{
						{
							ResourceTypeID: "grandchild-type-1",
							Children:       []interfaces.ResourceTypeHierarchy{},
						},
					},
				},
			},
		}

		Convey("insert new hierarchy", func() {
			// 查询不存在
			rows := sqlmock.NewRows([]string{"f_resource_type_id"})
			mock.ExpectQuery("^select f_resource_type_id").WithArgs(resourceTypeID).WillReturnRows(rows)
			// 插入
			mock.ExpectExec("^insert into").WithArgs(
				resourceTypeID,
				sqlmock.AnyArg(), // hierarchyStr (JSON)
				sqlmock.AnyArg(), // curTime
				sqlmock.AnyArg(), // curTime
			).WillReturnResult(sqlmock.NewResult(1, 1))

			err := r.Set(ctx, hierarchy)
			assert.Equal(t, err, nil)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("update existing hierarchy", func() {
			// 查询存在
			rows := sqlmock.NewRows([]string{"f_resource_type_id"}).
				AddRow(resourceTypeID)
			mock.ExpectQuery("^select f_resource_type_id").WithArgs(resourceTypeID).WillReturnRows(rows)
			// 更新
			mock.ExpectExec("^update").WithArgs(
				sqlmock.AnyArg(), // hierarchyStr (JSON)
				sqlmock.AnyArg(), // curTime
				resourceTypeID,
			).WillReturnResult(sqlmock.NewResult(0, 1))

			err := r.Set(ctx, hierarchy)
			assert.Equal(t, err, nil)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("query error", func() {
			mockErr := errors.New("query error")
			mock.ExpectQuery("^select f_resource_type_id").WithArgs(resourceTypeID).WillReturnError(mockErr)

			err := r.Set(ctx, hierarchy)
			assert.Equal(t, err, mockErr)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("insert error", func() {
			rows := sqlmock.NewRows([]string{"f_resource_type_id"})
			mock.ExpectQuery("^select f_resource_type_id").WithArgs(resourceTypeID).WillReturnRows(rows)
			mockErr := errors.New("insert error")
			mock.ExpectExec("^insert into").WillReturnError(mockErr)

			err := r.Set(ctx, hierarchy)
			assert.Equal(t, err, mockErr)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("update error", func() {
			rows := sqlmock.NewRows([]string{"f_resource_type_id"}).
				AddRow(resourceTypeID)
			mock.ExpectQuery("^select f_resource_type_id").WithArgs(resourceTypeID).WillReturnRows(rows)
			mockErr := errors.New("update error")
			mock.ExpectExec("^update").WillReturnError(mockErr)

			err := r.Set(ctx, hierarchy)
			assert.Equal(t, err, mockErr)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("empty children", func() {
			emptyHierarchy := &interfaces.ResourceTypeHierarchy{
				ResourceTypeID: resourceTypeID,
				Children:       []interfaces.ResourceTypeHierarchy{},
			}

			rows := sqlmock.NewRows([]string{"f_resource_type_id"})
			mock.ExpectQuery("^select f_resource_type_id").WithArgs(resourceTypeID).WillReturnRows(rows)
			mock.ExpectExec("^insert into").WillReturnResult(sqlmock.NewResult(1, 1))

			err := r.Set(ctx, emptyHierarchy)
			assert.Equal(t, err, nil)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})
	})
}

func TestResourceTypeHierarchy_GetAll(t *testing.T) {
	Convey("TestResourceTypeHierarchy_GetAll", t, func() {
		db, mock, err := sqlx.New()
		assert.Equal(t, err, nil)
		defer func(db *sqlx.DB) {
			err := db.Close()
			if err != nil {
				return
			}
		}(db)

		ctx := context.Background()
		r := &resourceTypeHierarchy{
			db:     db,
			logger: common.NewLogger(),
		}

		Convey("query error", func() {
			mockErr := errors.New("query error")
			mock.ExpectQuery("^select f_resource_type_id, f_children").WillReturnError(mockErr)

			hierarchyMap, err := r.GetAll(ctx)
			assert.Equal(t, err, mockErr)
			assert.Equal(t, len(hierarchyMap), 0)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("empty result", func() {
			rows := sqlmock.NewRows([]string{"f_resource_type_id", "f_children"})
			mock.ExpectQuery("^select f_resource_type_id, f_children").WillReturnRows(rows)

			hierarchyMap, err := r.GetAll(ctx)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(hierarchyMap), 0)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("single hierarchy without children", func() {
			resourceTypeID := resourceTypeHierarchyTestResourceTypeID
			childrenJSON := "[]"

			rows := sqlmock.NewRows([]string{"f_resource_type_id", "f_children"}).
				AddRow(resourceTypeID, childrenJSON)
			mock.ExpectQuery("^select f_resource_type_id, f_children").WillReturnRows(rows)

			hierarchyMap, err := r.GetAll(ctx)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(hierarchyMap), 1)
			assert.Equal(t, hierarchyMap[resourceTypeID].ResourceTypeID, resourceTypeID)
			assert.Equal(t, len(hierarchyMap[resourceTypeID].Children), 0)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("single hierarchy with children", func() {
			resourceTypeID := resourceTypeHierarchyTestResourceTypeID
			childrenJSON := `[{"resource_type_id":"child-type-1","children":[]},{"resource_type_id":"child-type-2","children":[{"resource_type_id":"grandchild-type-1","children":[]}]}]`

			rows := sqlmock.NewRows([]string{"f_resource_type_id", "f_children"}).
				AddRow(resourceTypeID, childrenJSON)
			mock.ExpectQuery("^select f_resource_type_id, f_children").WillReturnRows(rows)

			hierarchyMap, err := r.GetAll(ctx)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(hierarchyMap), 1)
			assert.Equal(t, hierarchyMap[resourceTypeID].ResourceTypeID, resourceTypeID)
			assert.Equal(t, len(hierarchyMap[resourceTypeID].Children), 2)
			assert.Equal(t, hierarchyMap[resourceTypeID].Children[0].ResourceTypeID, "child-type-1")
			assert.Equal(t, hierarchyMap[resourceTypeID].Children[1].ResourceTypeID, "child-type-2")
			assert.Equal(t, len(hierarchyMap[resourceTypeID].Children[1].Children), 1)
			assert.Equal(t, hierarchyMap[resourceTypeID].Children[1].Children[0].ResourceTypeID, "grandchild-type-1")
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("multiple hierarchies", func() {
			resourceTypeID1 := "type-1"
			childrenJSON1 := `[{"resource_type_id":"child-type-1","children":[]}]`
			resourceTypeID2 := "type-2"
			childrenJSON2 := `[]`

			rows := sqlmock.NewRows([]string{"f_resource_type_id", "f_children"}).
				AddRow(resourceTypeID1, childrenJSON1).
				AddRow(resourceTypeID2, childrenJSON2)
			mock.ExpectQuery("^select f_resource_type_id, f_children").WillReturnRows(rows)

			hierarchyMap, err := r.GetAll(ctx)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(hierarchyMap), 2)
			assert.Equal(t, hierarchyMap[resourceTypeID1].ResourceTypeID, resourceTypeID1)
			assert.Equal(t, len(hierarchyMap[resourceTypeID1].Children), 1)
			assert.Equal(t, hierarchyMap[resourceTypeID2].ResourceTypeID, resourceTypeID2)
			assert.Equal(t, len(hierarchyMap[resourceTypeID2].Children), 0)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("scan error", func() {
			rows := sqlmock.NewRows([]string{"f_resource_type_id", "f_children"}).
				AddRow("type-1", nil) // nil 会导致 scan 错误
			mock.ExpectQuery("^select f_resource_type_id, f_children").WillReturnRows(rows)

			hierarchyMap, err := r.GetAll(ctx)
			assert.NotEqual(t, err, nil)
			assert.Equal(t, len(hierarchyMap), 0)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})

		Convey("invalid JSON", func() {
			resourceTypeID := resourceTypeHierarchyTestResourceTypeID
			invalidJSON := "invalid json"

			rows := sqlmock.NewRows([]string{"f_resource_type_id", "f_children"}).
				AddRow(resourceTypeID, invalidJSON)
			mock.ExpectQuery("^select f_resource_type_id, f_children").WillReturnRows(rows)

			hierarchyMap, err := r.GetAll(ctx)
			assert.NotEqual(t, err, nil)
			assert.Equal(t, len(hierarchyMap), 0)
			assert.Equal(t, mock.ExpectationsWereMet(), nil)
		})
	})
}

func TestConvertHierarchyChildrenToJSONString(t *testing.T) {
	Convey("TestConvertHierarchyChildrenToJSONString", t, func() {
		Convey("empty children", func() {
			children := []interfaces.ResourceTypeHierarchy{}
			jsonStr, err := convertHierarchyChildrenToJSONString(children)
			assert.Equal(t, err, nil)
			assert.Equal(t, jsonStr, "[]")
		})

		Convey("single child without grandchildren", func() {
			children := []interfaces.ResourceTypeHierarchy{
				{
					ResourceTypeID: "child-type-1",
					Children:       []interfaces.ResourceTypeHierarchy{},
				},
			}
			jsonStr, err := convertHierarchyChildrenToJSONString(children)
			assert.Equal(t, err, nil)
			var result []any
			err = json.Unmarshal([]byte(jsonStr), &result)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(result), 1)
			item := result[0].(map[string]any)
			assert.Equal(t, item["resource_type_id"], "child-type-1")
		})

		Convey("multiple children with nested structure", func() {
			children := []interfaces.ResourceTypeHierarchy{
				{
					ResourceTypeID: "child-type-1",
					Children:       []interfaces.ResourceTypeHierarchy{},
				},
				{
					ResourceTypeID: "child-type-2",
					Children: []interfaces.ResourceTypeHierarchy{
						{
							ResourceTypeID: "grandchild-type-1",
							Children:       []interfaces.ResourceTypeHierarchy{},
						},
					},
				},
			}
			jsonStr, err := convertHierarchyChildrenToJSONString(children)
			assert.Equal(t, err, nil)
			var result []any
			err = json.Unmarshal([]byte(jsonStr), &result)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(result), 2)
		})
	})
}

func TestConvertJSONStringToHierarchyChildren(t *testing.T) {
	Convey("TestConvertJSONStringToHierarchyChildren", t, func() {
		Convey("empty JSON array", func() {
			jsonStr := "[]"
			children, err := convertJSONStringToHierarchyChildren(jsonStr)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(children), 0)
		})

		Convey("invalid JSON", func() {
			jsonStr := "invalid json"
			children, err := convertJSONStringToHierarchyChildren(jsonStr)
			assert.NotEqual(t, err, nil)
			assert.Equal(t, children, nil)
		})

		Convey("single child without grandchildren", func() {
			jsonStr := `[{"resource_type_id":"child-type-1","children":[]}]`
			children, err := convertJSONStringToHierarchyChildren(jsonStr)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(children), 1)
			assert.Equal(t, children[0].ResourceTypeID, "child-type-1")
			assert.Equal(t, len(children[0].Children), 0)
		})

		Convey("multiple children with nested structure", func() {
			jsonStr := `[{"resource_type_id":"child-type-1","children":[]},{"resource_type_id":"child-type-2","children":[{"resource_type_id":"grandchild-type-1","children":[]}]}]`
			children, err := convertJSONStringToHierarchyChildren(jsonStr)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(children), 2)
			assert.Equal(t, children[0].ResourceTypeID, "child-type-1")
			assert.Equal(t, len(children[0].Children), 0)
			assert.Equal(t, children[1].ResourceTypeID, "child-type-2")
			assert.Equal(t, len(children[1].Children), 1)
			assert.Equal(t, children[1].Children[0].ResourceTypeID, "grandchild-type-1")
		})

		Convey("null children field", func() {
			jsonStr := `[{"resource_type_id":"child-type-1","children":null}]`
			children, err := convertJSONStringToHierarchyChildren(jsonStr)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(children), 1)
			assert.Equal(t, children[0].ResourceTypeID, "child-type-1")
			assert.Equal(t, len(children[0].Children), 0)
		})

		Convey("missing children field", func() {
			jsonStr := `[{"resource_type_id":"child-type-1"}]`
			children, err := convertJSONStringToHierarchyChildren(jsonStr)
			assert.Equal(t, err, nil)
			assert.Equal(t, len(children), 1)
			assert.Equal(t, children[0].ResourceTypeID, "child-type-1")
			assert.Equal(t, len(children[0].Children), 0)
		})
	})
}

func TestConvertJSONItemsToHierarchy(t *testing.T) {
	Convey("TestConvertJSONItemsToHierarchy", t, func() {
		Convey("nil input", func() {
			children := convertJSONItemsToHierarchy(nil)
			assert.Equal(t, len(children), 0)
		})

		Convey("empty array", func() {
			children := convertJSONItemsToHierarchy([]any{})
			assert.Equal(t, len(children), 0)
		})

		Convey("invalid item type", func() {
			items := []any{
				"invalid",
				map[string]any{
					"resource_type_id": "valid-type",
					"children":         []any{},
				},
			}
			children := convertJSONItemsToHierarchy(items)
			// 无效项被跳过，只返回有效项
			assert.Equal(t, len(children), 1)
			assert.Equal(t, children[0].ResourceTypeID, "valid-type")
		})

		Convey("single item", func() {
			items := []any{
				map[string]any{
					"resource_type_id": "child-type-1",
					"children":         []any{},
				},
			}
			children := convertJSONItemsToHierarchy(items)
			assert.Equal(t, len(children), 1)
			assert.Equal(t, children[0].ResourceTypeID, "child-type-1")
			assert.Equal(t, len(children[0].Children), 0)
		})

		Convey("nested structure", func() {
			items := []any{
				map[string]any{
					"resource_type_id": "child-type-1",
					"children": []any{
						map[string]any{
							"resource_type_id": "grandchild-type-1",
							"children":         []any{},
						},
					},
				},
			}
			children := convertJSONItemsToHierarchy(items)
			assert.Equal(t, len(children), 1)
			assert.Equal(t, children[0].ResourceTypeID, "child-type-1")
			assert.Equal(t, len(children[0].Children), 1)
			assert.Equal(t, children[0].Children[0].ResourceTypeID, "grandchild-type-1")
		})

		Convey("missing resource_type_id", func() {
			items := []any{
				map[string]any{
					"children": []any{},
				},
			}
			children := convertJSONItemsToHierarchy(items)
			assert.Equal(t, len(children), 1)
			assert.Equal(t, children[0].ResourceTypeID, "")
		})

		Convey("children is not array", func() {
			items := []any{
				map[string]any{
					"resource_type_id": "child-type-1",
					"children":         "not an array",
				},
			}
			children := convertJSONItemsToHierarchy(items)
			assert.Equal(t, len(children), 1)
			assert.Equal(t, children[0].ResourceTypeID, "child-type-1")
			assert.Equal(t, len(children[0].Children), 0)
		})
	})
}
