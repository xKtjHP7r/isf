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

	"Authorization/interfaces"
	"Authorization/interfaces/mock"
)

func TestPolicyCalcRestHandler_CheckPublic(t *testing.T) {
	Convey("checkPublic", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockPolicyCalc := mock.NewMockLogicsPolicyCalc(ctrl)
		mockHydra := mock.NewMockHydra(ctrl)

		// 创建测试处理器
		handler := &policyCalcRestHandler{
			policyCalc: mockPolicyCalc,
			hydra:      mockHydra,
			checkPublicSchema: newJSONSchema(`{
				"type": "object",
				"required": ["resource", "operation", "method"],
				"properties": {
					"resource": {
						"type": "object",
						"required": ["id", "type"],
						"properties": {
							"id": {"type": "string"},
							"name": {"type": "string"},
							"type": {"type": "string"}
						}
					},
					"operation": {
						"type": "array",
						"items": {"type": "string"}
					},
					"method": {
						"type": "string",
						"enum": ["GET"]
					}
				}
			}`),
		}

		handler.RegisterPublic(r)

		Convey("成功检查权限", func() {
			// 准备请求数据
			reqBody := map[string]any{
				"resource": map[string]any{
					"id":   "resource1",
					"type": "doc",
					"name": "测试文档",
				},
				"operation": []string{"read", "write"},
				"method":    "GET",
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "user1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			mockPolicyCalc.EXPECT().Check(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(interfaces.CheckResult{
				Result: true,
			}, nil)

			// 创建请求
			req := httptest.NewRequest("POST", "/api/authorization/v1/operation-check", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Authorization", "Bearer test-token")
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusOK)

			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(response["result"], ShouldEqual, true)
		})

		Convey("权限检查失败", func() {
			// 准备请求数据
			reqBody := map[string]any{
				"resource": map[string]any{
					"id":   "resource1",
					"type": "doc",
				},
				"operation": []string{"read"},
				"method":    "GET",
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "user1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			mockPolicyCalc.EXPECT().Check(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(interfaces.CheckResult{
				Result: false,
			}, nil)

			// 创建请求
			req := httptest.NewRequest("POST", "/api/authorization/v1/operation-check", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Authorization", "Bearer test-token")
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusOK)

			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(response["result"], ShouldEqual, false)
		})
	})
}

func TestPolicyCalcRestHandler_Check(t *testing.T) {
	Convey("check", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockPolicyCalc := mock.NewMockLogicsPolicyCalc(ctrl)

		// 创建测试处理器
		handler := &policyCalcRestHandler{
			policyCalc: mockPolicyCalc,
			checkSchema: newJSONSchema(`{
				"type": "object",
				"required": ["accessor", "resource", "operation", "method"],
				"properties": {
					"accessor": {
						"type": "object",
						"required": ["id", "type"],
						"properties": {
							"id": {"type": "string"},
							"type": {"type": "string", "enum": ["user", "app"]}
						}
					},
					"resource": {
						"type": "object",
						"required": ["id", "type"],
						"properties": {
							"id": {"type": "string"},
							"name": {"type": "string"},
							"type": {"type": "string"}
						}
					},
					"operation": {
						"type": "array",
						"items": {"type": "string"}
					},
					"method": {
						"type": "string",
						"enum": ["GET"]
					},
					"include": {
						"type": "array",
						"items": {"type": "string"}
					}
				}
			}`),
			visitorStrToType: map[string]interfaces.VisitorType{
				"user": interfaces.RealName,
				"app":  interfaces.App,
			},
			includeStrToType: map[string]interfaces.PolicCalcyIncludeType{
				"operation_obligations": interfaces.PolicCalcyIncludeOperationObligations,
			},
		}

		handler.RegisterPrivate(r)

		Convey("成功检查权限", func() {
			// 准备请求数据
			reqBody := map[string]any{
				"accessor": map[string]any{
					"id":   "user1",
					"type": "user",
				},
				"resource": map[string]any{
					"id":   "resource1",
					"type": "doc",
					"name": "测试文档",
				},
				"operation": []string{"read", "write"},
				"method":    "GET",
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			// 设置 mock 期望
			mockPolicyCalc.EXPECT().Check(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(interfaces.CheckResult{
				Result: true,
			}, nil)

			// 创建请求
			req := httptest.NewRequest("POST", "/api/authorization/v1/operation-check", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusOK)

			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(response["result"], ShouldEqual, true)
		})
	})
}

func TestPolicyCalcRestHandler_ResourceFilter(t *testing.T) {
	Convey("resourceFilter", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockPolicyCalc := mock.NewMockLogicsPolicyCalc(ctrl)

		// 创建测试处理器
		handler := &policyCalcRestHandler{
			policyCalc: mockPolicyCalc,
			resourceFilterSchema: newJSONSchema(`{
				"type": "object",
				"required": ["accessor", "resources", "operation"],
				"properties": {
					"accessor": {
						"type": "object",
						"required": ["id", "type"],
						"properties": {
							"id": {"type": "string"},
							"type": {"type": "string", "enum": ["user", "app"]}
						}
					},
					"resources": {
						"type": "array",
						"items": {
							"type": "object",
							"required": ["id", "type"],
							"properties": {
								"id": {"type": "string"},
								"type": {"type": "string"}
							}
						}
					},
					"operation": {
						"type": "array",
						"items": {"type": "string"}
					},
					"allow_operation": {"type": "boolean"},
					"include": {
						"type": "array",
						"items": {"type": "string"}
					}
				}
			}`),
			visitorStrToType: map[string]interfaces.VisitorType{
				"user": interfaces.RealName,
				"app":  interfaces.App,
			},
			includeStrToType: map[string]interfaces.PolicCalcyIncludeType{
				"operation_obligations": interfaces.PolicCalcyIncludeOperationObligations,
			},
		}

		handler.RegisterPrivate(r)

		Convey("成功过滤资源", func() {
			// 准备请求数据
			reqBody := map[string]any{
				"accessor": map[string]any{
					"id":   "user1",
					"type": "user",
				},
				"resources": []map[string]any{
					{"id": "resource1", "type": "doc"},
					{"id": "resource2", "type": "doc"},
				},
				"operation":       []string{"read"},
				"allow_operation": true,
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			// 设置 mock 期望
			expectedResources := []interfaces.ResourceInfo{
				{ID: "resource1", Type: "doc"},
			}
			expectedOperationMap := map[string][]string{
				"resource1": {"read"},
			}
			expectedObligationMap := map[string]map[string][]interfaces.PolicyObligationItem{
				"resource1": {},
			}

			mockPolicyCalc.EXPECT().ResourceFilter(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
				expectedResources, expectedOperationMap, expectedObligationMap, nil)

			// 创建请求
			req := httptest.NewRequest("POST", "/api/authorization/v1/resource-filter", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusOK)

			var response []map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(len(response), ShouldEqual, 1)
			So(response[0]["id"], ShouldEqual, "resource1")
			So(response[0]["allow_operation"], ShouldResemble, []any{"read"})
		})
	})
}

func TestPolicyCalcRestHandler_ResourceOperation(t *testing.T) {
	Convey("resourceOperation", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockPolicyCalc := mock.NewMockLogicsPolicyCalc(ctrl)

		// 创建测试处理器
		handler := &policyCalcRestHandler{
			policyCalc: mockPolicyCalc,
			resourceOperationSchema: newJSONSchema(`{
				"type": "object",
				"required": ["accessor", "resources"],
				"properties": {
					"accessor": {
						"type": "object",
						"required": ["id", "type"],
						"properties": {
							"id": {"type": "string"},
							"type": {"type": "string", "enum": ["user", "app"]}
						}
					},
					"resources": {
						"type": "array",
						"items": {
							"type": "object",
							"required": ["id", "type"],
							"properties": {
								"id": {"type": "string"},
								"type": {"type": "string"}
							}
						}
					}
				}
			}`),
			visitorStrToType: map[string]interfaces.VisitorType{
				"user": interfaces.RealName,
				"app":  interfaces.App,
			},
		}

		handler.RegisterPrivate(r)

		Convey("成功获取资源操作", func() {
			// 准备请求数据
			reqBody := map[string]any{
				"accessor": map[string]any{
					"id":   "user1",
					"type": "user",
				},
				"resources": []map[string]any{
					{"id": "resource1", "type": "doc"},
					{"id": "resource2", "type": "doc"},
				},
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			// 设置 mock 期望
			expectedOperationMap := map[string][]string{
				"resource1": {"read", "write"},
				"resource2": {"read"},
			}
			expectedObligationMap := map[string]map[string][]interfaces.PolicyObligationItem{
				"resource1": {
					"read": {{TypeID: "type1", Value: map[string]any{"key": "value"}}},
				},
				"resource2": {},
			}

			mockPolicyCalc.EXPECT().GetResourceOperation(gomock.Any(), gomock.Any(), gomock.Any()).Return(
				expectedOperationMap, expectedObligationMap, nil)

			// 创建请求
			req := httptest.NewRequest("POST", "/api/authorization/v1/resource-operation", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusOK)

			var response []map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(len(response), ShouldEqual, 2)
		})
	})
}

func TestPolicyCalcRestHandler_ResourceOperationPublic(t *testing.T) {
	Convey("resourceOperationPublic", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockPolicyCalc := mock.NewMockLogicsPolicyCalc(ctrl)
		mockHydra := mock.NewMockHydra(ctrl)

		// 创建测试处理器
		handler := &policyCalcRestHandler{
			policyCalc: mockPolicyCalc,
			hydra:      mockHydra,
			resourceOperationPublicSchema: newJSONSchema(`{
				"type": "object",
				"required": ["resources"],
				"properties": {
					"resources": {
						"type": "array",
						"items": {
							"type": "object",
							"required": ["id", "type"],
							"properties": {
								"id": {"type": "string"},
								"type": {"type": "string"}
							}
						}
					}
				}
			}`),
		}

		handler.RegisterPublic(r)

		Convey("成功获取资源操作", func() {
			// 准备请求数据
			reqBody := map[string]any{
				"resources": []map[string]any{
					{"id": "resource1", "type": "doc"},
					{"id": "resource2", "type": "doc"},
				},
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "user1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			expectedOperationMap := map[string][]string{
				"resource1": {"read", "write"},
				"resource2": {"read"},
			}
			expectedObligationMap := map[string]map[string][]interfaces.PolicyObligationItem{}

			mockPolicyCalc.EXPECT().GetResourceOperation(gomock.Any(), gomock.Any(), gomock.Any()).Return(
				expectedOperationMap, expectedObligationMap, nil)

			// 创建请求
			req := httptest.NewRequest("POST", "/api/authorization/v1/resource-operation", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Authorization", "Bearer test-token")
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusOK)

			var response []map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(len(response), ShouldEqual, 2)
		})
	})
}

func TestPolicyCalcRestHandler_ResourceTypeOperationPublic(t *testing.T) {
	Convey("resourceTypeOperationPublic", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockPolicyCalc := mock.NewMockLogicsPolicyCalc(ctrl)
		mockHydra := mock.NewMockHydra(ctrl)

		// 创建测试处理器
		handler := &policyCalcRestHandler{
			policyCalc: mockPolicyCalc,
			hydra:      mockHydra,
			resourceTypeOperationPublicSchema: newJSONSchema(`{
				"type": "object",
				"required": ["resource_types"],
				"properties": {
					"resource_types": {
						"type": "array",
						"items": {"type": "string"}
					}
				}
			}`),
		}

		handler.RegisterPublic(r)

		Convey("成功获取资源类型操作", func() {
			// 准备请求数据
			reqBody := map[string]any{
				"resource_types": []string{"doc", "folder"},
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "user1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			expectedOperationMap := map[string][]string{
				"doc":    {"read", "write"},
				"folder": {"read"},
			}

			mockPolicyCalc.EXPECT().GetResourceTypeOperation(gomock.Any(), gomock.Any(), gomock.Any()).Return(
				expectedOperationMap, nil)

			// 创建请求
			req := httptest.NewRequest("POST", "/api/authorization/v1/resource-type-operation", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Authorization", "Bearer test-token")
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusOK)

			var response []map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(len(response), ShouldEqual, 2)
		})
	})
}

func TestPolicyCalcRestHandler_ResourceList(t *testing.T) {
	Convey("resourceList", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockPolicyCalc := mock.NewMockLogicsPolicyCalc(ctrl)

		// 创建测试处理器
		handler := &policyCalcRestHandler{
			policyCalc: mockPolicyCalc,
			resourceListSchema: newJSONSchema(`{
				"type": "object",
				"required": ["accessor", "resource", "operation"],
				"properties": {
					"accessor": {
						"type": "object",
						"required": ["id", "type"],
						"properties": {
							"id": {"type": "string"},
							"type": {"type": "string", "enum": ["user", "app"]}
						}
					},
					"resource": {
						"type": "object",
						"required": ["type"],
						"properties": {
							"type": {"type": "string"}
						}
					},
					"operation": {
						"type": "array",
						"items": {"type": "string"}
					},
					"include": {
						"type": "array",
						"items": {"type": "string"}
					}
				}
			}`),
			visitorStrToType: map[string]interfaces.VisitorType{
				"user": interfaces.RealName,
				"app":  interfaces.App,
			},
			includeStrToType: map[string]interfaces.PolicCalcyIncludeType{
				"operation_obligations": interfaces.PolicCalcyIncludeOperationObligations,
			},
		}

		handler.RegisterPrivate(r)

		Convey("成功获取资源列表", func() {
			// 准备请求数据
			reqBody := map[string]any{
				"accessor": map[string]any{
					"id":   "user1",
					"type": "user",
				},
				"resource": map[string]any{
					"type": "doc",
				},
				"operation": []string{"read"},
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			// 设置 mock 期望
			expectedResources := []interfaces.ResourceInfo{
				{ID: "resource1", Type: "doc"},
				{ID: "resource2", Type: "doc"},
			}
			expectedObligationMap := map[string]map[string][]interfaces.PolicyObligationItem{}

			mockPolicyCalc.EXPECT().GetResourceList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
				expectedResources, expectedObligationMap, nil)

			// 创建请求
			req := httptest.NewRequest("POST", "/api/authorization/v1/resource-list", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusOK)

			var response []map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(len(response), ShouldEqual, 2)
		})
	})
}

func TestPolicyCalcRestHandler_CheckWithInclude(t *testing.T) {
	Convey("check with include", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockPolicyCalc := mock.NewMockLogicsPolicyCalc(ctrl)

		// 创建测试处理器
		handler := &policyCalcRestHandler{
			policyCalc: mockPolicyCalc,
			checkSchema: newJSONSchema(`{
				"type": "object",
				"required": ["accessor", "resource", "operation", "method"],
				"properties": {
					"accessor": {
						"type": "object",
						"required": ["id", "type"],
						"properties": {
							"id": {"type": "string"},
							"type": {"type": "string", "enum": ["user", "app"]}
						}
					},
					"resource": {
						"type": "object",
						"required": ["id", "type"],
						"properties": {
							"id": {"type": "string"},
							"name": {"type": "string"},
							"type": {"type": "string"}
						}
					},
					"operation": {
						"type": "array",
						"items": {"type": "string"}
					},
					"method": {
						"type": "string",
						"enum": ["GET"]
					},
					"include": {
						"type": "array",
						"items": {"type": "string"}
					}
				}
			}`),
			visitorStrToType: map[string]interfaces.VisitorType{
				"user": interfaces.RealName,
				"app":  interfaces.App,
			},
			includeStrToType: map[string]interfaces.PolicCalcyIncludeType{
				"operation_obligations": interfaces.PolicCalcyIncludeOperationObligations,
			},
		}

		handler.RegisterPrivate(r)

		Convey("成功检查权限 - 带义务", func() {
			// 准备请求数据
			reqBody := map[string]any{
				"accessor": map[string]any{
					"id":   "user1",
					"type": "user",
				},
				"resource": map[string]any{
					"id":   "resource1",
					"type": "doc",
					"name": "测试文档",
				},
				"operation": []string{"read", "write"},
				"method":    "GET",
				"include":   []string{"operation_obligations"},
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			// 设置 mock 期望
			mockPolicyCalc.EXPECT().Check(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(interfaces.CheckResult{
				Result: true,
				OperatrionOblist: map[string][]interfaces.PolicyObligationItem{
					"read": {
						{TypeID: "type1", Value: map[string]any{"key": "value"}},
					},
					"write": {
						{TypeID: "type2", Value: map[string]any{"key2": "value2"}},
					},
				},
			}, nil)

			// 创建请求
			req := httptest.NewRequest("POST", "/api/authorization/v1/operation-check", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusOK)

			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(response["result"], ShouldEqual, true)
			So(response["include"], ShouldNotBeNil)

			include := response["include"].(map[string]any)
			So(include["operation_obligations"], ShouldNotBeNil)
		})

		Convey("无效的 include 类型", func() {
			// 准备请求数据
			reqBody := map[string]any{
				"accessor": map[string]any{
					"id":   "user1",
					"type": "user",
				},
				"resource": map[string]any{
					"id":   "resource1",
					"type": "doc",
				},
				"operation": []string{"read"},
				"method":    "GET",
				"include":   []string{"invalid_type"},
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			// 创建请求
			req := httptest.NewRequest("POST", "/api/authorization/v1/operation-check", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果 - 应该返回错误
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})
	})
}

func TestPolicyCalcRestHandler_ResourceFilterWithInclude(t *testing.T) {
	Convey("resourceFilter with include", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockPolicyCalc := mock.NewMockLogicsPolicyCalc(ctrl)

		// 创建测试处理器
		handler := &policyCalcRestHandler{
			policyCalc: mockPolicyCalc,
			resourceFilterSchema: newJSONSchema(`{
				"type": "object",
				"required": ["accessor", "resources", "operation"],
				"properties": {
					"accessor": {
						"type": "object",
						"required": ["id", "type"],
						"properties": {
							"id": {"type": "string"},
							"type": {"type": "string", "enum": ["user", "app"]}
						}
					},
					"resources": {
						"type": "array",
						"items": {
							"type": "object",
							"required": ["id", "type"],
							"properties": {
								"id": {"type": "string"},
								"type": {"type": "string"}
							}
						}
					},
					"operation": {
						"type": "array",
						"items": {"type": "string"}
					},
					"allow_operation": {"type": "boolean"},
					"include": {
						"type": "array",
						"items": {"type": "string"}
					}
				}
			}`),
			visitorStrToType: map[string]interfaces.VisitorType{
				"user": interfaces.RealName,
				"app":  interfaces.App,
			},
			includeStrToType: map[string]interfaces.PolicCalcyIncludeType{
				"operation_obligations": interfaces.PolicCalcyIncludeOperationObligations,
			},
		}

		handler.RegisterPrivate(r)

		Convey("成功过滤资源 - 带义务", func() {
			// 准备请求数据
			reqBody := map[string]any{
				"accessor": map[string]any{
					"id":   "user1",
					"type": "user",
				},
				"resources": []map[string]any{
					{"id": "resource1", "type": "doc"},
					{"id": "resource2", "type": "doc"},
				},
				"operation": []string{"read"},
				"include":   []string{"operation_obligations"},
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			// 设置 mock 期望
			expectedResources := []interfaces.ResourceInfo{
				{ID: "resource1", Type: "doc"},
			}
			expectedOperationMap := map[string][]string{
				"resource1": {"read"},
			}
			expectedObligationMap := map[string]map[string][]interfaces.PolicyObligationItem{
				"resource1": {
					"read": {{TypeID: "type1", Value: map[string]any{"key": "value"}}},
				},
			}

			mockPolicyCalc.EXPECT().ResourceFilter(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
				expectedResources, expectedOperationMap, expectedObligationMap, nil)

			// 创建请求
			req := httptest.NewRequest("POST", "/api/authorization/v1/resource-filter", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusOK)

			var response []map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(len(response), ShouldEqual, 1)
			So(response[0]["id"], ShouldEqual, "resource1")
			So(response[0]["include"], ShouldNotBeNil)
		})
	})
}

func TestPolicyCalcRestHandler_ResourceListWithInclude(t *testing.T) {
	Convey("resourceList with include", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockPolicyCalc := mock.NewMockLogicsPolicyCalc(ctrl)

		// 创建测试处理器
		handler := &policyCalcRestHandler{
			policyCalc: mockPolicyCalc,
			resourceListSchema: newJSONSchema(`{
				"type": "object",
				"required": ["accessor", "resource", "operation"],
				"properties": {
					"accessor": {
						"type": "object",
						"required": ["id", "type"],
						"properties": {
							"id": {"type": "string"},
							"type": {"type": "string", "enum": ["user", "app"]}
						}
					},
					"resource": {
						"type": "object",
						"required": ["type"],
						"properties": {
							"type": {"type": "string"}
						}
					},
					"operation": {
						"type": "array",
						"items": {"type": "string"}
					},
					"include": {
						"type": "array",
						"items": {"type": "string"}
					}
				}
			}`),
			visitorStrToType: map[string]interfaces.VisitorType{
				"user": interfaces.RealName,
				"app":  interfaces.App,
			},
			includeStrToType: map[string]interfaces.PolicCalcyIncludeType{
				"operation_obligations": interfaces.PolicCalcyIncludeOperationObligations,
			},
		}

		handler.RegisterPrivate(r)

		Convey("成功获取资源列表 - 带义务", func() {
			// 准备请求数据
			reqBody := map[string]any{
				"accessor": map[string]any{
					"id":   "user1",
					"type": "user",
				},
				"resource": map[string]any{
					"type": "doc",
				},
				"operation": []string{"read"},
				"include":   []string{"operation_obligations"},
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			// 设置 mock 期望
			expectedResources := []interfaces.ResourceInfo{
				{ID: "resource1", Type: "doc"},
				{ID: "resource2", Type: "doc"},
			}
			expectedObligationMap := map[string]map[string][]interfaces.PolicyObligationItem{
				"resource1": {
					"read": {{TypeID: "type1", Value: map[string]any{"key": "value"}}},
				},
				"resource2": {},
			}

			mockPolicyCalc.EXPECT().GetResourceList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
				expectedResources, expectedObligationMap, nil)

			// 创建请求
			req := httptest.NewRequest("POST", "/api/authorization/v1/resource-list", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusOK)

			var response []map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(len(response), ShouldEqual, 2)
			So(response[0]["id"], ShouldNotBeEmpty)
		})
	})
}
