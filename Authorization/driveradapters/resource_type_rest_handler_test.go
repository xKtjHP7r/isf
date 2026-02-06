//nolint:gocritic
package driveradapters

import (
	"bytes"
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

func TestResourceTypeRestHandler_Set(t *testing.T) {
	Convey("set", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockResourceType := mock.NewMockLogicsResourceType(ctrl)
		mockHydra := mock.NewMockHydra(ctrl)

		// 创建测试处理器
		handler := &resourceTypeRestHandler{
			resourceType: mockResourceType,
			hydra:        mockHydra,
			setSchemaStr: newJSONSchema(`{
				"type": "object",
				"required": ["name", "data_struct", "operation"],
				"properties": {
					"name": {"type": "string"},
					"description": {"type": "string"},
					"instance_url": {"type": "string"},
					"data_struct": {"type": "string"},
					"operation": {
						"type": "array",
						"items": {
							"type": "object",
							"required": ["id", "name"],
							"properties": {
								"id": {"type": "string"},
								"name": {
									"type": "array",
									"items": {
										"type": "object",
										"required": ["language", "value"],
										"properties": {
											"language": {"type": "string"},
											"value": {"type": "string"}
										}
									}
								},
								"description": {"type": "string"},
								"scope": {
									"type": "array",
									"items": {"type": "string"}
								}
							}
						}
					}
				}
			}`),
		}

		handler.RegisterPublic(r)

		Convey("成功设置资源类型", func() {
			// 准备请求数据
			reqBody := map[string]any{
				"name":         "文档",
				"description":  "文档资源类型",
				"instance_url": "https://example.com/doc",
				"data_struct":  "{}",
				"operation": []map[string]any{
					{
						"id":          "read",
						"name":        []map[string]any{{"language": "zh", "value": "读取"}},
						"description": "读取文档",
						"scope":       []string{"type", "instance"},
					},
					{
						"id":          "write",
						"name":        []map[string]any{{"language": "zh", "value": "写入"}},
						"description": "写入文档",
						"scope":       []string{"instance"},
					},
				},
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			mockResourceType.EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

			// 创建请求
			req := httptest.NewRequest("PUT", "/api/authorization/v1/resource_type/doc", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Authorization", "Bearer test-token")
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusNoContent)
		})

		Convey("验证失败 - 缺少必需字段", func() {
			// 准备请求数据 - 缺少 name 字段
			reqBody := map[string]any{
				"data_struct": "{}",
				"operation":   []map[string]any{},
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			// 创建请求
			req := httptest.NewRequest("PUT", "/api/authorization/v1/resource_type/doc", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Authorization", "Bearer test-token")
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("认证失败", func() {
			// 准备请求数据
			reqBody := map[string]any{
				"name":        "文档",
				"data_struct": "{}",
				"operation":   []map[string]any{},
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			// 设置 mock 期望 - 认证失败
			mockHydra.EXPECT().Introspect("invalid-token").Return(interfaces.TokenIntrospectInfo{
				Active: false,
			}, nil)

			// 创建请求
			req := httptest.NewRequest("PUT", "/api/authorization/v1/resource_type/doc", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Authorization", "Bearer invalid-token")
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusUnauthorized)
		})
	})
}

func TestResourceTypeRestHandler_SetPrivate(t *testing.T) {
	Convey("setPrivate", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockResourceType := mock.NewMockLogicsResourceType(ctrl)

		handler := &resourceTypeRestHandler{
			resourceType:        mockResourceType,
			setPrivateSchemaStr: newJSONSchema(setPrivateSchemaStr),
		}

		handler.RegisterPrivate(r)

		Convey("成功设置内部资源类型", func() {
			reqBody := map[string]any{
				"name":         "内部文档",
				"description":  "内部资源类型",
				"instance_url": "https://internal/doc",
				"data_struct":  "tree",
				"hidden":       true,
				"operation": []map[string]any{
					{
						"id":          "read",
						"name":        []map[string]any{{"language": "zh-cn", "value": "读取"}},
						"description": "读取操作",
						"scope":       []string{"type"},
					},
				},
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			mockResourceType.EXPECT().SetPrivate(gomock.Any(), gomock.Any()).Return(nil)

			req := httptest.NewRequest("PUT", "/api/authorization/v1/resource_type/internal_doc", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			So(w.Code, ShouldEqual, http.StatusNoContent)
		})

		Convey("验证失败 - 缺少必需字段", func() {
			reqBody := map[string]any{
				"data_struct": "tree",
				"operation":   []map[string]any{},
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("PUT", "/api/authorization/v1/resource_type/internal_doc", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("保存失败", func() {
			reqBody := map[string]any{
				"name":        "内部文档",
				"data_struct": "tree",
				"operation": []map[string]any{
					{
						"id":    "read",
						"name":  []map[string]any{{"language": "zh", "value": "读取"}},
						"scope": []string{"type"},
					},
				},
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			mockResourceType.EXPECT().SetPrivate(gomock.Any(), gomock.Any()).Return(gerrors.NewError(gerrors.PublicInternalServerError, "set private failed"))

			req := httptest.NewRequest("PUT", "/api/authorization/v1/resource_type/internal_doc", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			So(w.Code, ShouldEqual, http.StatusInternalServerError)
		})
	})
}

func TestResourceTypeRestHandler_Delete(t *testing.T) {
	Convey("delete", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockResourceType := mock.NewMockLogicsResourceType(ctrl)
		mockHydra := mock.NewMockHydra(ctrl)

		// 创建测试处理器
		handler := &resourceTypeRestHandler{
			resourceType: mockResourceType,
			hydra:        mockHydra,
		}

		handler.RegisterPublic(r)

		Convey("成功删除资源类型", func() {
			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			mockResourceType.EXPECT().Delete(gomock.Any(), gomock.Any(), "doc").Return(nil)

			// 创建请求
			req := httptest.NewRequest("DELETE", "/api/authorization/v1/resource_type/doc", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusNoContent)
		})

		Convey("删除失败 - 资源类型不存在", func() {
			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			mockResourceType.EXPECT().Delete(gomock.Any(), gomock.Any(), "nonexistent").Return(gerrors.NewError(gerrors.PublicNotFound, "resource type not found"))

			// 创建请求
			req := httptest.NewRequest("DELETE", "/api/authorization/v1/resource_type/nonexistent", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusNotFound)
		})
	})
}

func TestResourceTypeRestHandler_GetByID(t *testing.T) {
	Convey("getByID", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockResourceType := mock.NewMockLogicsResourceType(ctrl)
		mockHydra := mock.NewMockHydra(ctrl)

		// 创建测试处理器
		handler := &resourceTypeRestHandler{
			resourceType: mockResourceType,
			hydra:        mockHydra,
		}

		handler.RegisterPublic(r)

		Convey("成功获取资源类型", func() {
			// 准备返回数据
			resourceType := interfaces.ResourceType{
				ID:          "doc",
				Name:        "文档",
				Description: "文档资源类型",
				InstanceURL: "https://example.com/doc",
				DataStruct:  "{}",
				Operation: []interfaces.ResourceTypeOperation{
					{
						ID:          "read",
						Name:        []interfaces.OperationName{{Language: "zh", Value: "读取"}},
						Description: "读取文档",
						Scope:       []interfaces.OperationScopeType{interfaces.ScopeType, interfaces.ScopeInstance},
					},
				},
			}

			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			mockResourceType.EXPECT().GetByID(gomock.Any(), gomock.Any(), "doc").Return(resourceType, nil)

			// 创建请求
			req := httptest.NewRequest("GET", "/api/authorization/v1/resource_type/doc", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusOK)

			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(response["id"], ShouldEqual, "doc")
			So(response["name"], ShouldEqual, "文档")
			So(response["description"], ShouldEqual, "文档资源类型")
		})

		Convey("获取失败 - 资源类型不存在", func() {
			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			mockResourceType.EXPECT().GetByID(gomock.Any(), gomock.Any(), "nonexistent").Return(interfaces.ResourceType{}, gerrors.NewError(gerrors.PublicNotFound, "resource type not found"))

			// 创建请求
			req := httptest.NewRequest("GET", "/api/authorization/v1/resource_type/nonexistent", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusNotFound)
		})
	})
}

func TestResourceTypeRestHandler_GetAllOperationV2(t *testing.T) {
	Convey("getAllOperation", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockResourceType := mock.NewMockLogicsResourceType(ctrl)
		mockHydra := mock.NewMockHydra(ctrl)

		// 创建测试处理器
		handler := &resourceTypeRestHandler{
			resourceType: mockResourceType,
			hydra:        mockHydra,
		}

		handler.RegisterPublic(r)

		Convey("成功获取所有操作 - type 范围", func() {
			// 准备返回数据
			operations := []interfaces.ResourceTypeOperationResponse{
				{
					ID:          "read",
					Name:        "读取",
					Description: "读取文档",
				},
				{
					ID:          "write",
					Name:        "写入",
					Description: "写入文档",
				},
			}

			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			mockResourceType.EXPECT().GetAllOperation(gomock.Any(), gomock.Any(), "doc", interfaces.ScopeType).Return(operations, nil, nil)

			// 创建请求
			req := httptest.NewRequest("GET", "/api/authorization/v2/resource_all_operation?resource_type=doc&scope=type", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusOK)

			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			operationsList, ok := response["operations"].([]any)
			So(ok, ShouldBeTrue)
			So(len(operationsList), ShouldEqual, 2)
		})

		Convey("成功获取所有操作 - instance 范围", func() {
			// 准备返回数据
			operations := []interfaces.ResourceTypeOperationResponse{
				{
					ID:          "read",
					Name:        "读取",
					Description: "读取文档",
				},
			}

			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			mockResourceType.EXPECT().GetAllOperation(gomock.Any(), gomock.Any(), "doc", interfaces.ScopeInstance).Return(operations, nil, nil)

			// 创建请求
			req := httptest.NewRequest("GET", "/api/authorization/v2/resource_all_operation?resource_type=doc&scope=instance", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusOK)

			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			operationsList, ok := response["operations"].([]any)
			So(ok, ShouldBeTrue)
			So(len(operationsList), ShouldEqual, 1)
		})

		Convey("参数验证失败 - 缺少 resource_type", func() {
			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			// 创建请求 - 缺少 resource_type 参数
			req := httptest.NewRequest("GET", "/api/authorization/v2/resource_all_operation?scope=type", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("参数验证失败 - 缺少 scope", func() {
			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			// 创建请求 - 缺少 scope 参数
			req := httptest.NewRequest("GET", "/api/authorization/v2/resource_all_operation?resource_type=doc", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})
	})
}

func TestResourceTypeRestHandler_Get(t *testing.T) {
	Convey("get", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockResourceType := mock.NewMockLogicsResourceType(ctrl)
		mockHydra := mock.NewMockHydra(ctrl)

		// 创建测试处理器
		handler := &resourceTypeRestHandler{
			resourceType: mockResourceType,
			hydra:        mockHydra,
		}

		handler.RegisterPublic(r)

		Convey("成功获取资源类型列表", func() {
			// 准备返回数据
			resourceTypes := []interfaces.ResourceType{
				{
					ID:          "doc",
					Name:        "文档",
					Description: "文档资源类型",
					InstanceURL: "https://example.com/doc",
					DataStruct:  "{}",
					Operation: []interfaces.ResourceTypeOperation{
						{
							ID:          "read",
							Name:        []interfaces.OperationName{{Language: "zh", Value: "读取"}},
							Description: "读取文档",
						},
					},
				},
				{
					ID:          "folder",
					Name:        "文件夹",
					Description: "文件夹资源类型",
					InstanceURL: "https://example.com/folder",
					DataStruct:  "{}",
					Operation: []interfaces.ResourceTypeOperation{
						{
							ID:          "create",
							Name:        []interfaces.OperationName{{Language: "zh", Value: "创建"}},
							Description: "创建文件夹",
						},
					},
				},
			}

			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			mockResourceType.EXPECT().GetPagination(gomock.Any(), gomock.Any(), gomock.Any()).Return(2, resourceTypes, nil)

			// 创建请求
			req := httptest.NewRequest("GET", "/api/authorization/v1/resource_type?offset=0&limit=20", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusOK)

			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(response["total_count"], ShouldEqual, float64(2))

			entries := response["entries"].([]any)
			So(len(entries), ShouldEqual, 2)
		})

		Convey("获取资源类型列表 - 使用默认分页参数", func() {
			// 准备返回数据
			resourceTypes := []interfaces.ResourceType{}

			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			mockResourceType.EXPECT().GetPagination(gomock.Any(), gomock.Any(), gomock.Any()).Return(0, resourceTypes, nil)

			// 创建请求 - 不提供分页参数
			req := httptest.NewRequest("GET", "/api/authorization/v1/resource_type", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusOK)

			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(response["total_count"], ShouldEqual, float64(0))
		})

		Convey("分页参数验证失败 - 无效的 offset", func() {
			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			// 创建请求 - 无效的 offset
			req := httptest.NewRequest("GET", "/api/authorization/v1/resource_type?offset=-1&limit=20", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("分页参数验证失败 - 无效的 limit", func() {
			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			// 创建请求 - 无效的 limit
			req := httptest.NewRequest("GET", "/api/authorization/v1/resource_type?offset=0&limit=2000", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})
	})
}
