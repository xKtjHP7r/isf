//nolint:gocritic
package driveradapters

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"

	gerrors "github.com/kweaver-ai/go-lib/error"

	"Authorization/interfaces"
	"Authorization/interfaces/mock"
)

//nolint:funlen
func TestResourceTypeHierarchyRestHandler_SetPrivate(t *testing.T) {
	Convey("setPrivate", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockResourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)

		// 创建测试处理器
		handler := &resourceTypeHierarchyRestHandler{
			resourceTypeHierarchy: mockResourceTypeHierarchy,
			setHierarchySchema: newJSONSchema(`{
				"$schema": "http://json-schema.org/draft-07/schema#",
				"definitions": {
					"resourceTypeHierarchy": {
						"type": "object",
						"required": [
							"resource_type_id",
							"children"
						],
						"properties": {
							"resource_type_id": {
								"type": "string"
							},
							"children": {
								"type": "array",
								"items": {
									"$ref": "#/definitions/resourceTypeHierarchy"
								}
							}
						}
					}
				},
				"type": "array",
				"items": {
					"$ref": "#/definitions/resourceTypeHierarchy"
				}
			}`),
		}

		handler.RegisterPrivate(r)

		Convey("成功设置资源类型层级关系", func() {
			reqBody := []map[string]any{
				{
					"resource_type_id": "child-type-1",
					"children":         []any{},
				},
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			mockResourceTypeHierarchy.EXPECT().
				SetPrivate(gomock.Any(), gomock.Any(), gomock.Any()).
				Do(func(ctx context.Context, visitor *interfaces.Visitor, hierarchy *interfaces.ResourceTypeHierarchy) {
					So(hierarchy.ResourceTypeID, ShouldEqual, "parent-type")
					So(len(hierarchy.Children), ShouldEqual, 1)
					So(hierarchy.Children[0].ResourceTypeID, ShouldEqual, "child-type-1")
				}).
				Return(nil)

			req := httptest.NewRequest("PUT", "/api/authorization/v1/resource_type_hierarchy/parent-type", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			So(w.Code, ShouldEqual, http.StatusNoContent)
		})

		Convey("成功设置嵌套层级关系", func() {
			reqBody := []map[string]any{
				{
					"resource_type_id": "child-type-1",
					"children": []any{
						map[string]any{
							"resource_type_id": "grandchild-type-1",
							"children":         []any{},
						},
					},
				},
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			mockResourceTypeHierarchy.EXPECT().
				SetPrivate(gomock.Any(), gomock.Any(), gomock.Any()).
				Do(func(ctx context.Context, visitor *interfaces.Visitor, hierarchy *interfaces.ResourceTypeHierarchy) {
					So(hierarchy.ResourceTypeID, ShouldEqual, "parent-type")
					So(len(hierarchy.Children), ShouldEqual, 1)
					So(hierarchy.Children[0].ResourceTypeID, ShouldEqual, "child-type-1")
					So(len(hierarchy.Children[0].Children), ShouldEqual, 1)
					So(hierarchy.Children[0].Children[0].ResourceTypeID, ShouldEqual, "grandchild-type-1")
				}).
				Return(nil)

			req := httptest.NewRequest("PUT", "/api/authorization/v1/resource_type_hierarchy/parent-type", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			So(w.Code, ShouldEqual, http.StatusNoContent)
		})

		Convey("验证失败 - JSON schema 验证失败", func() {
			// 缺少必需字段
			reqBody := []map[string]any{
				{
					"resource_type_id": "child-type-1",
					// 缺少 children 字段
				},
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("PUT", "/api/authorization/v1/resource_type_hierarchy/parent-type", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("验证失败 - 无效的 JSON 格式", func() {
			reqBodyBytes := []byte("invalid json")

			req := httptest.NewRequest("PUT", "/api/authorization/v1/resource_type_hierarchy/parent-type", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("验证失败 - 不是数组", func() {
			reqBody := map[string]any{
				"resource_type_id": "child-type-1",
				"children":         []any{},
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("PUT", "/api/authorization/v1/resource_type_hierarchy/parent-type", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("解析失败 - resource_type_id 不是字符串", func() {
			reqBody := []map[string]any{
				{
					"resource_type_id": 123, // 应该是字符串
					"children":         []any{},
				},
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("PUT", "/api/authorization/v1/resource_type_hierarchy/parent-type", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("解析失败 - children 不是数组", func() {
			reqBody := []map[string]any{
				{
					"resource_type_id": "child-type-1",
					"children":         "not-an-array", // 应该是数组
				},
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("PUT", "/api/authorization/v1/resource_type_hierarchy/parent-type", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("SetPrivate 失败", func() {
			reqBody := []map[string]any{
				{
					"resource_type_id": "child-type-1",
					"children":         []any{},
				},
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			mockResourceTypeHierarchy.EXPECT().
				SetPrivate(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(gerrors.NewError(gerrors.PublicInternalServerError, "设置层级关系失败"))

			req := httptest.NewRequest("PUT", "/api/authorization/v1/resource_type_hierarchy/parent-type", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			So(w.Code, ShouldEqual, http.StatusInternalServerError)
		})
	})
}

func TestResourceTypeHierarchyRestHandler_ParseHierarchy(t *testing.T) {
	Convey("parseHierarchy", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		handler := &resourceTypeHierarchyRestHandler{}

		Convey("成功解析简单层级", func() {
			jsonReq := map[string]any{
				"resource_type_id": "child-type-1",
				"children":         []any{},
			}

			hierarchy, err := handler.parseHierarchy(jsonReq)
			So(err, ShouldBeNil)
			So(hierarchy.ResourceTypeID, ShouldEqual, "child-type-1")
			So(len(hierarchy.Children), ShouldEqual, 0)
		})

		Convey("成功解析嵌套层级", func() {
			jsonReq := map[string]any{
				"resource_type_id": "child-type-1",
				"children": []any{
					map[string]any{
						"resource_type_id": "grandchild-type-1",
						"children":         []any{},
					},
					map[string]any{
						"resource_type_id": "grandchild-type-2",
						"children":         []any{},
					},
				},
			}

			hierarchy, err := handler.parseHierarchy(jsonReq)
			So(err, ShouldBeNil)
			So(hierarchy.ResourceTypeID, ShouldEqual, "child-type-1")
			So(len(hierarchy.Children), ShouldEqual, 2)
			So(hierarchy.Children[0].ResourceTypeID, ShouldEqual, "grandchild-type-1")
			So(hierarchy.Children[1].ResourceTypeID, ShouldEqual, "grandchild-type-2")
		})

		Convey("解析失败 - 不是对象", func() {
			jsonReq := "not-an-object"

			_, err := handler.parseHierarchy(jsonReq)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "invalid hierarchy format: expected object")
		})

		Convey("解析失败 - 缺少 resource_type_id", func() {
			jsonReq := map[string]any{
				"children": []any{},
			}

			_, err := handler.parseHierarchy(jsonReq)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "invalid resource_type_id: expected string")
		})

		Convey("解析失败 - resource_type_id 不是字符串", func() {
			jsonReq := map[string]any{
				"resource_type_id": 123,
				"children":         []any{},
			}

			_, err := handler.parseHierarchy(jsonReq)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "invalid resource_type_id: expected string")
		})

		Convey("解析失败 - 缺少 children", func() {
			jsonReq := map[string]any{
				"resource_type_id": "child-type-1",
			}

			_, err := handler.parseHierarchy(jsonReq)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "invalid children: expected array")
		})

		Convey("解析失败 - children 不是数组", func() {
			jsonReq := map[string]any{
				"resource_type_id": "child-type-1",
				"children":         "not-an-array",
			}

			_, err := handler.parseHierarchy(jsonReq)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "invalid children: expected array")
		})

		Convey("解析失败 - 嵌套层级中 children 格式错误", func() {
			jsonReq := map[string]any{
				"resource_type_id": "child-type-1",
				"children": []any{
					"not-an-object", // 应该是对象
				},
			}

			_, err := handler.parseHierarchy(jsonReq)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "invalid hierarchy format: expected object")
		})
	})
}

func TestResourceTypeHierarchyRestHandler_RegisterPrivate(t *testing.T) {
	Convey("RegisterPrivate", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockResourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		mockResourceTypeHierarchy.EXPECT().SetPrivate(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		handler := &resourceTypeHierarchyRestHandler{
			resourceTypeHierarchy: mockResourceTypeHierarchy,
			hydra:                 newHydra(),
			setHierarchySchema:    newJSONSchema(setHierarchySchemaStr),
		}
		handler.RegisterPrivate(r)

		Convey("应该注册 PUT /api/authorization/v1/resource_type_hierarchy/:resource_type_id 路由", func() {
			// 创建一个简单的请求来验证路由已注册
			reqBody := []map[string]any{
				{
					"resource_type_id": "child-type-1",
					"children":         []any{},
				},
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("PUT", "/api/authorization/v1/resource_type_hierarchy/test-type", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			// 即使返回错误，也说明路由已注册（否则会返回 404）
			So(w.Code, ShouldNotEqual, http.StatusNotFound)
		})
	})
}

func TestResourceTypeHierarchyRestHandler_RegisterPublic(t *testing.T) {
	Convey("RegisterPublic", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockResourceTypeHierarchy := mock.NewMockLogicsResourceTypeHierarchy(ctrl)
		handler := &resourceTypeHierarchyRestHandler{
			resourceTypeHierarchy: mockResourceTypeHierarchy,
			hydra:                 newHydra(),
			setHierarchySchema:    newJSONSchema(setHierarchySchemaStr),
		}
		handler.RegisterPublic(r)

		Convey("目前没有注册任何公共路由", func() {
			// 测试一个不存在的路由，应该返回 404
			req := httptest.NewRequest("GET", "/api/authorization/v1/resource_type_hierarchy/test", nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			// 由于没有注册任何路由，应该返回 404
			So(w.Code, ShouldEqual, http.StatusNotFound)
		})
	})
}
