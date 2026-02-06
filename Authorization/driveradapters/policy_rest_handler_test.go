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

func TestPolicyRestHandler_Create(t *testing.T) {
	Convey("create", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockPolicy := mock.NewMockLogicsPolicy(ctrl)
		mockHydra := mock.NewMockHydra(ctrl)

		// 创建测试处理器
		handler := &policyRestHandler{
			policy: mockPolicy,
			hydra:  mockHydra,
			policySchema: newJSONSchema(`{
				"type": "array",
				"items": {
					"type": "object",
					"required": ["accessor", "resource", "operation"],
					"properties": {
						"accessor": {
							"type": "object",
							"required": ["id", "type"],
							"properties": {
								"id": {"type": "string"},
								"type": {"type": "string", "enum": ["user", "department", "group", "role", "app"]}
							}
						},
						"resource": {
							"type": "object",
							"required": ["id", "name", "type"],
							"properties": {
								"id": {"type": "string"},
								"name": {"type": "string"},
								"type": {"type": "string"}
							}
						},
						"operation": {
							"type": "object",
							"required": ["allow", "deny"],
							"properties": {
								"allow": {
									"type": "array",
									"items": {
										"type": "object",
										"properties": {
											"id": {"type": "string"}
										}
									}
								},
								"deny": {
									"type": "array",
									"items": {
										"type": "object",
										"properties": {
											"id": {"type": "string"}
										}
									}
								}
							}
						},
						"condition": {"type": "string"},
						"expires_at": {"type": "string"}
					}
				}
			}`),
			accessorStrToType: map[string]interfaces.AccessorType{
				"user":       interfaces.AccessorUser,
				"department": interfaces.AccessorDepartment,
				"group":      interfaces.AccessorGroup,
				"app":        interfaces.AccessorApp,
				"role":       interfaces.AccessorRole,
			},
		}

		handler.RegisterPublic(r)

		Convey("成功创建策略", func() {
			// 准备请求数据
			reqBody := []map[string]any{
				{
					"accessor": map[string]any{
						"id":   "user1",
						"type": "user",
					},
					"resource": map[string]any{
						"id":   "resource1",
						"name": "测试文档",
						"type": "doc",
					},
					"operation": map[string]any{
						"allow": []map[string]any{
							{"id": "read"},
							{"id": "write"},
						},
						"deny": []map[string]any{
							{"id": "delete"},
						},
					},
					"condition":  "test condition",
					"expires_at": "2024-12-31T23:59:59Z",
				},
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			mockPolicy.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return([]string{"policy1", "policy2"}, nil)

			// 创建请求
			req := httptest.NewRequest("POST", "/api/authorization/v1/policy", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Authorization", "Bearer test-token")
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusCreated)
		})
	})
}

func TestPolicyRestHandler_CreatePrivate(t *testing.T) {
	Convey("createPrivate", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockPolicy := mock.NewMockLogicsPolicy(ctrl)

		// 创建测试处理器
		handler := &policyRestHandler{
			policy: mockPolicy,
			policySchema: newJSONSchema(`{
				"type": "array",
				"items": {
					"type": "object",
					"required": ["accessor", "resource", "operation"],
					"properties": {
						"accessor": {
							"type": "object",
							"required": ["id", "type"],
							"properties": {
								"id": {"type": "string"},
								"type": {"type": "string", "enum": ["user", "department", "group", "role", "app"]}
							}
						},
						"resource": {
							"type": "object",
							"required": ["id", "name", "type"],
							"properties": {
								"id": {"type": "string"},
								"name": {"type": "string"},
								"type": {"type": "string"}
							}
						},
						"operation": {
							"type": "object",
							"required": ["allow", "deny"],
							"properties": {
								"allow": {
									"type": "array",
									"items": {
										"type": "object",
										"properties": {
											"id": {"type": "string"}
										}
									}
								},
								"deny": {
									"type": "array",
									"items": {
										"type": "object",
										"properties": {
											"id": {"type": "string"}
										}
									}
								}
							}
						},
						"condition": {"type": "string"},
						"expires_at": {"type": "string"}
					}
				}
			}`),
			accessorStrToType: map[string]interfaces.AccessorType{
				"user":       interfaces.AccessorUser,
				"department": interfaces.AccessorDepartment,
				"group":      interfaces.AccessorGroup,
				"app":        interfaces.AccessorApp,
				"role":       interfaces.AccessorRole,
			},
		}

		handler.RegisterPrivate(r)

		Convey("成功创建私有策略", func() {
			// 准备请求数据
			reqBody := []map[string]any{
				{
					"accessor": map[string]any{
						"id":   "user1",
						"type": "user",
					},
					"resource": map[string]any{
						"id":   "resource1",
						"name": "测试文档",
						"type": "doc",
					},
					"operation": map[string]any{
						"allow": []map[string]any{
							{"id": "read"},
						},
						"deny": []map[string]any{},
					},
				},
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			// 设置 mock 期望
			mockPolicy.EXPECT().CreatePrivate(gomock.Any(), gomock.Any()).Return(nil)

			// 创建请求
			req := httptest.NewRequest("POST", "/api/authorization/v1/policy", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusNoContent)
		})
	})
}

func TestPolicyRestHandler_DeletePrivate(t *testing.T) {
	Convey("deletePrivate", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockPolicy := mock.NewMockLogicsPolicy(ctrl)

		// 创建测试处理器
		handler := &policyRestHandler{
			policy: mockPolicy,
			policyDeleteSchema: newJSONSchema(`{
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

		handler.RegisterPrivate(r)

		Convey("成功删除私有策略", func() {
			// 准备请求数据
			reqBody := map[string]any{
				"resources": []map[string]any{
					{"id": "resource1", "type": "doc"},
					{"id": "resource2", "type": "folder"},
				},
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			// 设置 mock 期望
			mockPolicy.EXPECT().DeleteByResourceIDs(gomock.Any(), gomock.Any()).Return(nil)

			// 创建请求
			req := httptest.NewRequest("POST", "/api/authorization/v1/policy-delete", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusNoContent)
		})
	})
}

func TestPolicyRestHandler_Set(t *testing.T) {
	Convey("set", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockPolicy := mock.NewMockLogicsPolicy(ctrl)
		mockHydra := mock.NewMockHydra(ctrl)

		// 创建测试处理器
		handler := &policyRestHandler{
			policy: mockPolicy,
			hydra:  mockHydra,
			modifyPolicySchema: newJSONSchema(`{
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"operation": {
							"type": "object",
							"required": ["allow", "deny"],
							"properties": {
								"allow": {
									"type": "array",
									"items": {
										"type": "object",
										"properties": {
											"id": {"type": "string"}
										}
									}
								},
								"deny": {
									"type": "array",
									"items": {
										"type": "object",
										"properties": {
											"id": {"type": "string"}
										}
									}
								}
							}
						},
						"condition": {"type": "string"},
						"expires_at": {"type": "string"}
					}
				}
			}`),
		}

		handler.RegisterPublic(r)

		Convey("成功更新策略", func() {
			// 准备请求数据
			reqBody := []map[string]any{
				{
					"operation": map[string]any{
						"allow": []map[string]any{
							{"id": "read"},
							{"id": "write"},
						},
						"deny": []map[string]any{
							{"id": "delete"},
						},
					},
					"condition":  "updated condition",
					"expires_at": "2024-12-31T23:59:59Z",
				},
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			mockPolicy.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

			// 创建请求
			req := httptest.NewRequest("PUT", "/api/authorization/v1/policy/policy1,policy2", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Authorization", "Bearer test-token")
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusNoContent)
		})
	})
}

func TestPolicyRestHandler_Delete(t *testing.T) {
	Convey("delete", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockPolicy := mock.NewMockLogicsPolicy(ctrl)
		mockHydra := mock.NewMockHydra(ctrl)

		// 创建测试处理器
		handler := &policyRestHandler{
			policy: mockPolicy,
			hydra:  mockHydra,
		}

		handler.RegisterPublic(r)

		Convey("成功删除策略", func() {
			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			mockPolicy.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

			// 创建请求
			req := httptest.NewRequest("DELETE", "/api/authorization/v1/policy/policy1,policy2", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusNoContent)
		})
	})
}

func TestPolicyRestHandler_Get(t *testing.T) {
	Convey("get", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockPolicy := mock.NewMockLogicsPolicy(ctrl)
		mockHydra := mock.NewMockHydra(ctrl)

		// 创建测试处理器
		handler := &policyRestHandler{
			policy: mockPolicy,
			hydra:  mockHydra,
			accessorTypeToStr: map[interfaces.AccessorType]string{
				interfaces.AccessorUser:       "user",
				interfaces.AccessorDepartment: "department",
				interfaces.AccessorGroup:      "group",
				interfaces.AccessorApp:        "app",
				interfaces.AccessorRole:       "role",
			},
		}

		handler.RegisterPublic(r)

		Convey("成功获取策略列表", func() {
			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			expectedPolicies := []interfaces.PolicyInfo{
				{
					ID:           "policy1",
					ResourceID:   "resource1",
					ResourceType: "doc",
					ResourceName: "测试文档",
					AccessorID:   "user1",
					AccessorType: interfaces.AccessorUser,
					AccessorName: "用户1",
					Operation: interfaces.PolicyOperation{
						Allow: []interfaces.PolicyOperationItem{{ID: "read"}},
						Deny:  []interfaces.PolicyOperationItem{},
					},
					Condition: "test condition",
					EndTime:   -1,
				},
			}

			mockPolicy.EXPECT().GetPagination(gomock.Any(), gomock.Any(), gomock.Any()).Return(1, expectedPolicies, nil)

			// 创建请求
			req := httptest.NewRequest("GET", "/api/authorization/v1/policy?resource_id=resource1&resource_type=doc&offset=0&limit=10", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusOK)

			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(response["total_count"], ShouldEqual, float64(1))
		})

		Convey("缺少必传参数 resource_id", func() {
			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			// 创建请求
			req := httptest.NewRequest("GET", "/api/authorization/v1/policy?resource_type=doc", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("缺少必传参数 resource_type", func() {
			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			// 创建请求
			req := httptest.NewRequest("GET", "/api/authorization/v1/policy?resource_id=resource1", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})
	})
}

//nolint:funlen
func TestPolicyRestHandler_GetAccessorPolicy(t *testing.T) {
	Convey("getAccessorPolicy", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockPolicy := mock.NewMockLogicsPolicy(ctrl)
		mockHydra := mock.NewMockHydra(ctrl)

		// 创建测试处理器
		handler := &policyRestHandler{
			policy: mockPolicy,
			hydra:  mockHydra,
			accessorStrToType: map[string]interfaces.AccessorType{
				"user":       interfaces.AccessorUser,
				"department": interfaces.AccessorDepartment,
				"group":      interfaces.AccessorGroup,
				"app":        interfaces.AccessorApp,
				"role":       interfaces.AccessorRole,
			},
			includeStrToType: map[string]interfaces.PolicyIncludeType{
				"obligation_types": interfaces.PolicyIncludeObligationType,
				"obligations":      interfaces.PolicyIncludeObligation,
			},
		}

		handler.RegisterPublic(r)

		tmp := interfaces.PolicyIncludeResp{}
		Convey("成功获取访问者策略", func() {
			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			expectedPolicies := []interfaces.PolicyInfo{
				{
					ID:           "policy1",
					ResourceID:   "resource1",
					ResourceType: "doc",
					ResourceName: "测试文档",
					AccessorID:   "user1",
					AccessorType: interfaces.AccessorUser,
					AccessorName: "用户1",
					Operation: interfaces.PolicyOperation{
						Allow: []interfaces.PolicyOperationItem{{ID: "read"}},
						Deny:  []interfaces.PolicyOperationItem{},
					},
					Condition: "test condition",
					EndTime:   -1,
				},
				{
					ID:           "policy2",
					ResourceID:   "resource2",
					ResourceType: "doc",
					ResourceName: "测试文档2",
					AccessorID:   "user1",
					AccessorType: interfaces.AccessorUser,
					AccessorName: "用户1",
					Operation: interfaces.PolicyOperation{
						Allow: []interfaces.PolicyOperationItem{{ID: "write"}},
						Deny:  []interfaces.PolicyOperationItem{{ID: "delete"}},
					},
					Condition: "",
					EndTime:   1640995200, // 2022-01-01 00:00:00
				},
			}

			mockPolicy.EXPECT().GetAccessorPolicy(gomock.Any(), gomock.Any(), gomock.Any()).Return(2, expectedPolicies, tmp, nil)

			// 创建请求
			req := httptest.NewRequest("GET", "/api/authorization/v1/accessor-policy?accessor_id=user1&accessor_type=user&resource_type=doc&resource_id=resource1", nil)
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

			// 验证 entries
			entries := response["entries"].([]any)
			So(len(entries), ShouldEqual, 2)

			// 验证第一个策略
			policy1 := entries[0].(map[string]any)
			So(policy1["id"], ShouldEqual, "policy1")
			So(policy1["condition"], ShouldEqual, "test condition")

			// 验证资源信息
			resource1 := policy1["resource"].(map[string]any)
			So(resource1["id"], ShouldEqual, "resource1")
			So(resource1["type"], ShouldEqual, "doc")
			So(resource1["name"], ShouldEqual, "测试文档")

			// 验证操作信息
			operation1 := policy1["operation"].(map[string]any)
			allow1 := operation1["allow"].([]any)
			So(len(allow1), ShouldEqual, 1)
			So(allow1[0].(map[string]any)["id"], ShouldEqual, "read")

			// 验证第二个策略
			policy2 := entries[1].(map[string]any)
			So(policy2["id"], ShouldEqual, "policy2")
			So(policy2["condition"], ShouldEqual, "")
		})

		Convey("无效的访问者类型", func() {
			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			// 创建请求
			req := httptest.NewRequest("GET", "/api/authorization/v1/accessor-policy?accessor_id=user1&accessor_type=invalid&resource_type=doc", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("策略服务返回错误", func() {
			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			mockPolicy.EXPECT().GetAccessorPolicy(gomock.Any(), gomock.Any(), gomock.Any()).Return(0, nil, tmp, gerrors.NewError(gerrors.PublicInternalServerError, "internal error"))

			// 创建请求
			req := httptest.NewRequest("GET", "/api/authorization/v1/accessor-policy?accessor_id=user1&accessor_type=user&resource_type=doc", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusInternalServerError)
		})

		Convey("返回空策略列表", func() {
			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			mockPolicy.EXPECT().GetAccessorPolicy(gomock.Any(), gomock.Any(), gomock.Any()).Return(0, []interfaces.PolicyInfo{}, tmp, nil)

			// 创建请求
			req := httptest.NewRequest("GET", "/api/authorization/v1/accessor-policy?accessor_id=user1&accessor_type=user&resource_type=doc", nil)
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
			entries := response["entries"].([]any)
			So(len(entries), ShouldEqual, 0)
		})

		Convey("测试不同的访问者类型", func() {
			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			expectedPolicies := []interfaces.PolicyInfo{
				{
					ID:           "policy1",
					ResourceID:   "resource1",
					ResourceType: "doc",
					ResourceName: "测试文档",
					AccessorID:   "dept1",
					AccessorType: interfaces.AccessorDepartment,
					AccessorName: "部门1",
					Operation: interfaces.PolicyOperation{
						Allow: []interfaces.PolicyOperationItem{{ID: "read"}},
						Deny:  []interfaces.PolicyOperationItem{},
					},
					Condition: "",
					EndTime:   -1,
				},
			}

			mockPolicy.EXPECT().GetAccessorPolicy(gomock.Any(), gomock.Any(), gomock.Any()).Return(1, expectedPolicies, tmp, nil)

			// 测试部门类型
			req := httptest.NewRequest("GET", "/api/authorization/v1/accessor-policy?accessor_id=dept1&accessor_type=user&resource_type=doc", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusOK)

			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(response["total_count"], ShouldEqual, float64(1))
			entries := response["entries"].([]any)
			So(len(entries), ShouldEqual, 1)
		})

		Convey("测试带条件的策略", func() {
			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			expectedPolicies := []interfaces.PolicyInfo{
				{
					ID:           "policy1",
					ResourceID:   "resource1",
					ResourceType: "doc",
					ResourceName: "测试文档",
					AccessorID:   "user1",
					AccessorType: interfaces.AccessorUser,
					AccessorName: "用户1",
					Operation: interfaces.PolicyOperation{
						Allow: []interfaces.PolicyOperationItem{{ID: "read"}},
						Deny:  []interfaces.PolicyOperationItem{},
					},
					Condition: `{"time": {"start": "09:00", "end": "18:00"}}`,
					EndTime:   -1,
				},
			}

			mockPolicy.EXPECT().GetAccessorPolicy(gomock.Any(), gomock.Any(), gomock.Any()).Return(1, expectedPolicies, tmp, nil)

			// 创建请求
			req := httptest.NewRequest("GET", "/api/authorization/v1/accessor-policy?accessor_id=user1&accessor_type=user&resource_type=doc", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusOK)
		})
	})
}

func TestPolicyRestHandler_GetResourcePolicy(t *testing.T) {
	Convey("getResourcePolicy", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockPolicy := mock.NewMockLogicsPolicy(ctrl)
		mockHydra := mock.NewMockHydra(ctrl)

		// 创建测试处理器
		handler := &policyRestHandler{
			policy: mockPolicy,
			hydra:  mockHydra,
			accessorTypeToStr: map[interfaces.AccessorType]string{
				interfaces.AccessorUser:       "user",
				interfaces.AccessorDepartment: "department",
				interfaces.AccessorGroup:      "group",
				interfaces.AccessorApp:        "app",
				interfaces.AccessorRole:       "role",
			},
			includeStrToType: map[string]interfaces.PolicyIncludeType{
				"obligation_types": interfaces.PolicyIncludeObligationType,
				"obligations":      interfaces.PolicyIncludeObligation,
			},
		}

		handler.RegisterPublic(r)

		tmp := interfaces.PolicyIncludeResp{}
		Convey("成功获取资源策略", func() {
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			expectedPolicies := []interfaces.PolicyInfo{
				{
					ID:           "policy1",
					ResourceID:   "resource1",
					ResourceType: "doc",
					ResourceName: "测试文档",
					AccessorID:   "user1",
					AccessorType: interfaces.AccessorUser,
					AccessorName: "用户1",
					Operation: interfaces.PolicyOperation{
						Allow: []interfaces.PolicyOperationItem{
							{
								ID:   "read",
								Name: "读取",
							},
						},
						Deny: []interfaces.PolicyOperationItem{},
					},
					Condition: "test condition",
					EndTime:   -1,
				},
			}

			mockPolicy.EXPECT().GetResourcePolicy(gomock.Any(), gomock.Any(), gomock.Any()).Return(1, expectedPolicies, tmp, nil)

			req := httptest.NewRequest("GET", "/api/authorization/v1/resource-policy?resource_id=resource1&resource_type=doc", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			So(w.Code, ShouldEqual, http.StatusOK)

			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(response["total_count"], ShouldEqual, float64(1))

			entries := response["entries"].([]any)
			So(len(entries), ShouldEqual, 1)

			policy1 := entries[0].(map[string]any)
			So(policy1["id"], ShouldEqual, "policy1")

			accessor := policy1["accessor"].(map[string]any)
			So(accessor["id"], ShouldEqual, "user1")
			So(accessor["type"], ShouldEqual, "user")
		})

		Convey("缺少必填参数 resource_id", func() {
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			req := httptest.NewRequest("GET", "/api/authorization/v1/resource-policy?resource_type=doc", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("缺少必填参数 resource_type", func() {
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			req := httptest.NewRequest("GET", "/api/authorization/v1/resource-policy?resource_id=resource1", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("成功获取资源策略 - 带义务类型 include", func() {
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			expectedPolicies := []interfaces.PolicyInfo{
				{
					ID:           "policy1",
					ResourceID:   "resource1",
					ResourceType: "doc",
					ResourceName: "测试文档",
					AccessorID:   "user1",
					AccessorType: interfaces.AccessorUser,
					AccessorName: "用户1",
					Operation: interfaces.PolicyOperation{
						Allow: []interfaces.PolicyOperationItem{
							{
								ID:   "read",
								Name: "读取",
								Obligations: []interfaces.PolicyObligationItem{
									{
										TypeID: "watermark",
										ID:     "obligation1",
									},
								},
							},
						},
						Deny: []interfaces.PolicyOperationItem{},
					},
					EndTime: -1,
				},
			}

			includeResp := interfaces.PolicyIncludeResp{
				ObligationTypes: []interfaces.ObligationTypeInfo{
					{
						ID:          "watermark",
						Name:        "水印",
						Description: "添加水印",
						Schema:      map[string]any{"type": "string"},
					},
				},
			}

			mockPolicy.EXPECT().GetResourcePolicy(gomock.Any(), gomock.Any(), gomock.Any()).Return(1, expectedPolicies, includeResp, nil)

			req := httptest.NewRequest("GET", "/api/authorization/v1/resource-policy?resource_id=resource1&resource_type=doc&include=obligation_types", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			So(w.Code, ShouldEqual, http.StatusOK)

			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)

			include := response["include"].(map[string]any)
			obligationTypes := include["obligation_types"].([]any)
			So(len(obligationTypes), ShouldEqual, 1)
		})

		Convey("无效的 include 参数", func() {
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			req := httptest.NewRequest("GET", "/api/authorization/v1/resource-policy?resource_id=resource1&resource_type=doc&include=invalid", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})
	})
}

func TestPolicyRestHandler_OperationArrayToJson(t *testing.T) {
	Convey("operationArrayToJson", t, func() {
		handler := &policyRestHandler{}

		Convey("空操作列表", func() {
			operations := []interfaces.PolicyOperationItem{}
			result := handler.operationArrayToJson(operations)

			So(len(result), ShouldEqual, 0)
		})

		Convey("单个操作", func() {
			operations := []interfaces.PolicyOperationItem{
				{
					ID:   "read",
					Name: "读取",
				},
			}
			result := handler.operationArrayToJson(operations)

			So(len(result), ShouldEqual, 1)
			operation := result[0].(map[string]any)
			So(operation["id"], ShouldEqual, "read")
			So(operation["name"], ShouldEqual, "读取")
		})

		Convey("多个操作", func() {
			operations := []interfaces.PolicyOperationItem{
				{ID: "read", Name: "读取"},
				{ID: "write", Name: "写入"},
				{ID: "delete", Name: "删除"},
			}
			result := handler.operationArrayToJson(operations)

			So(len(result), ShouldEqual, 3)
			So(result[0].(map[string]any)["id"], ShouldEqual, "read")
			So(result[1].(map[string]any)["id"], ShouldEqual, "write")
			So(result[2].(map[string]any)["id"], ShouldEqual, "delete")
		})
	})
}

func TestPolicyRestHandler_OperationArrayToJsonWithObligations(t *testing.T) {
	Convey("operationArrayToJsonWithObligations", t, func() {
		handler := &policyRestHandler{}

		Convey("空操作列表", func() {
			operations := []interfaces.PolicyOperationItem{}
			result := handler.operationArrayToJsonWithObligations(operations)

			So(len(result), ShouldEqual, 0)
		})

		Convey("无义务的操作", func() {
			operations := []interfaces.PolicyOperationItem{
				{
					ID:          "read",
					Name:        "读取",
					Obligations: []interfaces.PolicyObligationItem{},
				},
			}
			result := handler.operationArrayToJsonWithObligations(operations)

			So(len(result), ShouldEqual, 1)
			operation := result[0].(map[string]any)
			So(operation["id"], ShouldEqual, "read")
			So(operation["name"], ShouldEqual, "读取")
			_, hasObligations := operation["obligations"]
			So(hasObligations, ShouldBeFalse)
		})

		Convey("带义务ID的操作", func() {
			operations := []interfaces.PolicyOperationItem{
				{
					ID:   "read",
					Name: "读取",
					Obligations: []interfaces.PolicyObligationItem{
						{
							TypeID: "watermark",
							ID:     "obligation1",
						},
					},
				},
			}
			result := handler.operationArrayToJsonWithObligations(operations)

			So(len(result), ShouldEqual, 1)
			operation := result[0].(map[string]any)
			So(operation["id"], ShouldEqual, "read")

			obligations := operation["obligations"].([]map[string]any)
			So(len(obligations), ShouldEqual, 1)
			So(obligations[0]["type_id"], ShouldEqual, "watermark")
			So(obligations[0]["id"], ShouldEqual, "obligation1")
			_, hasValue := obligations[0]["value"]
			So(hasValue, ShouldBeFalse)
		})

		Convey("带义务值的操作", func() {
			operations := []interfaces.PolicyOperationItem{
				{
					ID:   "read",
					Name: "读取",
					Obligations: []interfaces.PolicyObligationItem{
						{
							TypeID: "watermark",
							ID:     "",
							Value:  "test watermark text",
						},
					},
				},
			}
			result := handler.operationArrayToJsonWithObligations(operations)

			So(len(result), ShouldEqual, 1)
			operation := result[0].(map[string]any)

			obligations := operation["obligations"].([]map[string]any)
			So(len(obligations), ShouldEqual, 1)
			So(obligations[0]["type_id"], ShouldEqual, "watermark")
			So(obligations[0]["value"], ShouldEqual, "test watermark text")
			_, hasID := obligations[0]["id"]
			So(hasID, ShouldBeFalse)
		})

		Convey("多个操作多个义务", func() {
			operations := []interfaces.PolicyOperationItem{
				{
					ID:   "read",
					Name: "读取",
					Obligations: []interfaces.PolicyObligationItem{
						{TypeID: "watermark", ID: "obligation1"},
						{TypeID: "log", Value: "access log"},
					},
				},
				{
					ID:   "write",
					Name: "写入",
					Obligations: []interfaces.PolicyObligationItem{
						{TypeID: "approval", ID: "obligation2"},
					},
				},
			}
			result := handler.operationArrayToJsonWithObligations(operations)

			So(len(result), ShouldEqual, 2)

			operation1 := result[0].(map[string]any)
			obligations1 := operation1["obligations"].([]map[string]any)
			So(len(obligations1), ShouldEqual, 2)

			operation2 := result[1].(map[string]any)
			obligations2 := operation2["obligations"].([]map[string]any)
			So(len(obligations2), ShouldEqual, 1)
		})
	})
}

func TestPolicyRestHandler_GetObligations(t *testing.T) {
	Convey("getObligations", t, func() {
		handler := &policyRestHandler{}

		Convey("空义务列表", func() {
			obligationsJson := []any{}
			result, err := handler.getObligations(obligationsJson)

			So(err, ShouldBeNil)
			So(len(result), ShouldEqual, 0)
		})

		Convey("仅包含 type_id", func() {
			obligationsJson := []any{
				map[string]any{
					"type_id": "watermark",
				},
			}
			result, err := handler.getObligations(obligationsJson)

			So(err, ShouldBeNil)
			So(len(result), ShouldEqual, 1)
			So(result[0].TypeID, ShouldEqual, "watermark")
			So(result[0].ID, ShouldEqual, "")
			So(result[0].Value, ShouldBeNil)
		})

		Convey("包含 type_id 和 id", func() {
			obligationsJson := []any{
				map[string]any{
					"type_id": "watermark",
					"id":      "obligation1",
				},
			}
			result, err := handler.getObligations(obligationsJson)

			So(err, ShouldBeNil)
			So(len(result), ShouldEqual, 1)
			So(result[0].TypeID, ShouldEqual, "watermark")
			So(result[0].ID, ShouldEqual, "obligation1")
			So(result[0].Value, ShouldBeNil)
		})

		Convey("包含 type_id 和 value", func() {
			obligationsJson := []any{
				map[string]any{
					"type_id": "watermark",
					"value":   "test watermark",
				},
			}
			result, err := handler.getObligations(obligationsJson)

			So(err, ShouldBeNil)
			So(len(result), ShouldEqual, 1)
			So(result[0].TypeID, ShouldEqual, "watermark")
			So(result[0].ID, ShouldEqual, "")
			So(result[0].Value, ShouldEqual, "test watermark")
		})

		Convey("同时包含 id 和 value - 应该返回错误", func() {
			obligationsJson := []any{
				map[string]any{
					"type_id": "watermark",
					"id":      "obligation1",
					"value":   "test watermark",
				},
			}
			result, err := handler.getObligations(obligationsJson)

			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "id and value cannot be both set")
			So(result, ShouldBeNil)
		})

		Convey("多个义务", func() {
			obligationsJson := []any{
				map[string]any{
					"type_id": "watermark",
					"id":      "obligation1",
				},
				map[string]any{
					"type_id": "log",
					"value":   "access log",
				},
				map[string]any{
					"type_id": "approval",
				},
			}
			result, err := handler.getObligations(obligationsJson)

			So(err, ShouldBeNil)
			So(len(result), ShouldEqual, 3)
			So(result[0].TypeID, ShouldEqual, "watermark")
			So(result[0].ID, ShouldEqual, "obligation1")
			So(result[1].TypeID, ShouldEqual, "log")
			So(result[1].Value, ShouldEqual, "access log")
			So(result[2].TypeID, ShouldEqual, "approval")
		})

		Convey("value 为复杂对象", func() {
			obligationsJson := []any{
				map[string]any{
					"type_id": "watermark",
					"value": map[string]any{
						"text":  "Confidential",
						"color": "red",
						"size":  12,
					},
				},
			}
			result, err := handler.getObligations(obligationsJson)

			So(err, ShouldBeNil)
			So(len(result), ShouldEqual, 1)
			So(result[0].TypeID, ShouldEqual, "watermark")

			value := result[0].Value.(map[string]any)
			So(value["text"], ShouldEqual, "Confidential")
			So(value["color"], ShouldEqual, "red")
			So(value["size"], ShouldEqual, 12)
		})
	})
}
