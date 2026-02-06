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

func TestRoleRestHandler_CreateRole(t *testing.T) {
	Convey("createRole", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockRole := mock.NewMockLogicsRole(ctrl)
		mockHydra := mock.NewMockHydra(ctrl)

		// 创建测试处理器
		handler := &roleRestHandler{
			role:             mockRole,
			hydra:            mockHydra,
			createRoleSchema: newJSONSchema(createRoleSchemaStr),
		}

		handler.RegisterPublic(r)

		Convey("成功创建角色", func() {
			// 准备请求数据
			reqBody := map[string]any{
				"name":        "测试角色",
				"description": "这是一个测试角色",
				"resource_type_scope": map[string]any{
					"unlimited": false,
					"types": []map[string]any{
						{"id": "doc", "name": "文档"},
						{"id": "folder", "name": "文件夹"},
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

			mockRole.EXPECT().AddRole(gomock.Any(), gomock.Any(), gomock.Any()).Return("role1", nil)

			// 创建请求
			req := httptest.NewRequest("POST", "/api/authorization/v1/roles", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Authorization", "Bearer test-token")
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusCreated)
			So(w.Header().Get("Location"), ShouldEqual, "/api/authorization/v1/roles/role1")

			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(response["id"], ShouldEqual, "role1")
		})
	})
}

func TestRoleRestHandler_DeleteRole(t *testing.T) {
	Convey("deleteRole", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockRole := mock.NewMockLogicsRole(ctrl)
		mockHydra := mock.NewMockHydra(ctrl)

		// 创建测试处理器
		handler := &roleRestHandler{
			role:  mockRole,
			hydra: mockHydra,
		}

		handler.RegisterPublic(r)

		Convey("成功删除角色", func() {
			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			mockRole.EXPECT().DeleteRole(gomock.Any(), gomock.Any(), "role1").Return(nil)

			// 创建请求
			req := httptest.NewRequest("DELETE", "/api/authorization/v1/roles/role1", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusNoContent)
		})
	})
}

func TestRoleRestHandler_ModifyRole(t *testing.T) {
	Convey("modifyRole", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockRole := mock.NewMockLogicsRole(ctrl)
		mockHydra := mock.NewMockHydra(ctrl)

		// 创建测试处理器
		handler := &roleRestHandler{
			role:             mockRole,
			hydra:            mockHydra,
			modifyRoleSchema: newJSONSchema(modifyRoleSchemaStr),
		}

		handler.RegisterPublic(r)

		Convey("成功修改角色名称", func() {
			// 准备请求数据
			reqBody := map[string]any{
				"name": "修改后的角色名称",
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			mockRole.EXPECT().ModifyRole(gomock.Any(), gomock.Any(), "role1", "修改后的角色名称", true, "", false, gomock.Any(), false).Return(nil)

			// 创建请求
			req := httptest.NewRequest("PUT", "/api/authorization/v1/roles/role1/name", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Authorization", "Bearer test-token")
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusNoContent)
		})

		Convey("成功修改角色描述", func() {
			// 准备请求数据
			reqBody := map[string]any{
				"description": "修改后的角色描述",
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			mockRole.EXPECT().ModifyRole(gomock.Any(), gomock.Any(), "role1", "", false, "修改后的角色描述", true, gomock.Any(), false).Return(nil)

			// 创建请求
			req := httptest.NewRequest("PUT", "/api/authorization/v1/roles/role1/description", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Authorization", "Bearer test-token")
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusNoContent)
		})

		Convey("成功修改角色资源类型范围", func() {
			// 准备请求数据
			reqBody := map[string]any{
				"resource_type_scope": map[string]any{
					"unlimited": false,
					"types": []map[string]any{
						{"id": "doc"},
						{"id": "folder"},
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

			expectedScopes := interfaces.ResourceTypeScopeInfo{
				Unlimited: false,
				Types: []interfaces.ResourceTypeScope{
					{ResourceTypeID: "doc"},
					{ResourceTypeID: "folder"},
				},
			}

			mockRole.EXPECT().ModifyRole(gomock.Any(), gomock.Any(), "role1", "", false, "", false, expectedScopes, true).Return(nil)

			// 创建请求
			req := httptest.NewRequest("PUT", "/api/authorization/v1/roles/role1/resource_type_scope", bytes.NewBuffer(reqBodyBytes))
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

func TestRoleRestHandler_GetRole(t *testing.T) {
	Convey("getRole", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockRole := mock.NewMockLogicsRole(ctrl)
		mockHydra := mock.NewMockHydra(ctrl)

		// 创建测试处理器
		handler := &roleRestHandler{
			role:  mockRole,
			hydra: mockHydra,
		}

		handler.RegisterPublic(r)

		Convey("成功获取角色列表", func() {
			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			expectedRoles := []interfaces.RoleInfo{
				{
					ID:          "role1",
					Name:        "测试角色1",
					Description: "这是测试角色1",
					ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
						Unlimited: false,
						Types: []interfaces.ResourceTypeScope{
							{ResourceTypeID: "doc", ResourceTypeName: "文档"},
						},
					},
				},
				{
					ID:          "role2",
					Name:        "测试角色2",
					Description: "这是测试角色2",
					ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
						Unlimited: false,
						Types: []interfaces.ResourceTypeScope{
							{ResourceTypeID: "doc", ResourceTypeName: "文档"},
						},
					},
				},
			}

			mockRole.EXPECT().GetRoles(gomock.Any(), gomock.Any(), gomock.Any()).Return(2, expectedRoles, nil)

			// 创建请求
			req := httptest.NewRequest("GET", "/api/authorization/v1/roles?offset=0&limit=10", nil)
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
		})
	})
}

func TestRoleRestHandler_GetRoleByID(t *testing.T) {
	Convey("getRoleByID", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockRole := mock.NewMockLogicsRole(ctrl)
		mockHydra := mock.NewMockHydra(ctrl)

		// 创建测试处理器
		handler := &roleRestHandler{
			role:  mockRole,
			hydra: mockHydra,
		}

		handler.RegisterPublic(r)

		Convey("成功根据ID获取角色", func() {
			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			expectedRole := interfaces.RoleInfoWithResourceTypeOperation{
				ID:          "role1",
				Name:        "测试角色",
				Description: "这是测试角色",
				ResourceTypeScopesInfo: interfaces.ResourceTypeScopeInfoWithOperation{
					Unlimited: false,
					Types: []interfaces.ResourceTypeScopeWithOperation{
						{
							ID:                "doc",
							Name:              "文档",
							Description:       "这是文档",
							TypeOperation:     []interfaces.ResourceTypeOperationResponse{},
							InstanceOperation: []interfaces.ResourceTypeOperationResponse{},
						},
					},
				},
				CreateTime: 1716393600,
				ModifyTime: 1716393600,
			}

			mockRole.EXPECT().GetRoleByID(gomock.Any(), gomock.Any(), "role1", gomock.Any()).Return(expectedRole, nil)

			// 创建请求
			req := httptest.NewRequest("GET", "/api/authorization/v1/roles/role1", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusOK)

			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(response["id"], ShouldEqual, "role1")
			So(response["name"], ShouldEqual, "测试角色")
		})
	})
}

func TestRoleRestHandler_GetRoleMembers(t *testing.T) {
	Convey("getRoleMembers", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockRole := mock.NewMockLogicsRole(ctrl)
		mockHydra := mock.NewMockHydra(ctrl)

		// 创建测试处理器
		handler := &roleRestHandler{
			role:  mockRole,
			hydra: mockHydra,
			memberIntTypes: map[interfaces.AccessorType]string{
				interfaces.AccessorUser:       "user",
				interfaces.AccessorDepartment: "department",
				interfaces.AccessorGroup:      "group",
				interfaces.AccessorApp:        "app",
			},
		}

		handler.RegisterPublic(r)

		Convey("成功获取角色成员列表", func() {
			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			expectedMembers := []interfaces.RoleMemberInfo{
				{
					ID:         "user1",
					MemberType: interfaces.AccessorUser,
					Name:       "用户1",
					ParentDeps: [][]interfaces.Department{},
				},
				{
					ID:         "dept1",
					MemberType: interfaces.AccessorDepartment,
					Name:       "部门1",
					ParentDeps: [][]interfaces.Department{},
				},
			}

			mockRole.EXPECT().GetRoleMembers(gomock.Any(), gomock.Any(), "role1", gomock.Any()).Return(2, expectedMembers, nil)

			// 创建请求
			req := httptest.NewRequest("GET", "/api/authorization/v1/role-members/role1?offset=0&limit=10", nil)
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
		})
	})
}

func TestRoleRestHandler_AddOrDeleteRoleMembers(t *testing.T) {
	Convey("addOrDeleteRoleMembers", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockRole := mock.NewMockLogicsRole(ctrl)
		mockHydra := mock.NewMockHydra(ctrl)

		// 创建测试处理器
		handler := &roleRestHandler{
			role:  mockRole,
			hydra: mockHydra,
			addDeleteMembersSchema: newJSONSchema(`{
				"required": ["method", "members"],
				"type": "object",
				"properties": {
					"method": {
						"type": "string",
						"enum": ["POST", "DELETE"]
					},
					"members": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"id": {"type": "string"},
								"type": {
									"type": "string",
									"enum": ["user", "department", "group", "app"]
								}
							}
						}
					}
				}
			}`),
			memberStringTypes: map[string]interfaces.AccessorType{
				"user":       interfaces.AccessorUser,
				"department": interfaces.AccessorDepartment,
				"group":      interfaces.AccessorGroup,
				"app":        interfaces.AccessorApp,
			},
		}

		handler.RegisterPublic(r)

		Convey("成功添加角色成员", func() {
			// 准备请求数据
			reqBody := map[string]any{
				"method": "POST",
				"members": []map[string]any{
					{"id": "user1", "type": "user"},
					{"id": "dept1", "type": "department"},
				},
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			mockRole.EXPECT().AddOrDeleteRoleMemebers(gomock.Any(), gomock.Any(), "POST", "role1", gomock.Any()).Return(nil)

			// 创建请求
			req := httptest.NewRequest("POST", "/api/authorization/v1/role-members/role1", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Authorization", "Bearer test-token")
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusNoContent)
		})

		Convey("成功删除角色成员", func() {
			// 准备请求数据
			reqBody := map[string]any{
				"method": "DELETE",
				"members": []map[string]any{
					{"id": "user1", "type": "user"},
				},
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			mockRole.EXPECT().AddOrDeleteRoleMemebers(gomock.Any(), gomock.Any(), "DELETE", "role1", gomock.Any()).Return(nil)

			// 创建请求
			req := httptest.NewRequest("POST", "/api/authorization/v1/role-members/role1", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Authorization", "Bearer test-token")
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusNoContent)
		})

		Convey("无效的成员类型", func() {
			// 准备请求数据
			reqBody := map[string]any{
				"method": "POST",
				"members": []map[string]any{
					{"id": "user1", "type": "invalid_type"},
				},
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			// 创建请求
			req := httptest.NewRequest("POST", "/api/authorization/v1/role-members/role1", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Authorization", "Bearer test-token")
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("无效的方法", func() {
			// 准备请求数据
			reqBody := map[string]any{
				"method": "PUT",
				"members": []map[string]any{
					{"id": "user1", "type": "user"},
				},
			}
			reqBodyBytes, _ := json.Marshal(reqBody)

			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			// 创建请求
			req := httptest.NewRequest("POST", "/api/authorization/v1/role-members/role1", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Authorization", "Bearer test-token")
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})
	})
}

//nolint:funlen
func TestRoleRestHandler_GetRoleByResourceTypeID(t *testing.T) {
	Convey("getRoleByResourceTypeID", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockRole := mock.NewMockLogicsRole(ctrl)
		mockHydra := mock.NewMockHydra(ctrl)

		// 创建测试处理器
		handler := &roleRestHandler{
			role:  mockRole,
			hydra: mockHydra,
		}

		handler.RegisterPublic(r)

		Convey("成功根据资源类型ID获取角色列表", func() {
			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			expectedRoles := []interfaces.RoleInfo{
				{
					ID:          "role1",
					Name:        "文档管理员",
					Description: "管理文档相关权限",
					ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
						Unlimited: false,
						Types: []interfaces.ResourceTypeScope{
							{ResourceTypeID: "doc", ResourceTypeName: "文档"},
						},
					},
				},
				{
					ID:          "role2",
					Name:        "文件夹管理员",
					Description: "管理文件夹相关权限",
					ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
						Unlimited: false,
						Types: []interfaces.ResourceTypeScope{
							{ResourceTypeID: "folder", ResourceTypeName: "文件夹"},
						},
					},
				},
			}

			mockRole.EXPECT().GetResourceTypeRoles(gomock.Any(), gomock.Any(), gomock.Any()).Return(2, expectedRoles, nil)

			// 创建请求
			req := httptest.NewRequest("GET", "/api/authorization/v1/resource_type_roles?resource_type_id=doc&offset=0&limit=10", nil)
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

			// 验证第一个角色
			role1 := entries[0].(map[string]any)
			So(role1["id"], ShouldEqual, "role1")
			So(role1["name"], ShouldEqual, "文档管理员")
			So(role1["description"], ShouldEqual, "管理文档相关权限")
		})

		Convey("缺少资源类型ID参数", func() {
			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			// 创建请求（没有 resource_type_id 参数）
			req := httptest.NewRequest("GET", "/api/authorization/v1/resource_type_roles?offset=0&limit=10", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果 返回200
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("空资源类型ID参数", func() {
			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			// 创建请求（空的 resource_type_id 参数）
			req := httptest.NewRequest("GET", "/api/authorization/v1/resource_type_roles?resource_type_id=&offset=0&limit=10", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("带关键词搜索", func() {
			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			expectedRoles := []interfaces.RoleInfo{
				{
					ID:          "role1",
					Name:        "文档管理员",
					Description: "管理文档相关权限",
					ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
						Unlimited: false,
						Types: []interfaces.ResourceTypeScope{
							{ResourceTypeID: "doc", ResourceTypeName: "文档"},
						},
					},
				},
			}

			mockRole.EXPECT().GetResourceTypeRoles(gomock.Any(), gomock.Any(), gomock.Any()).Return(1, expectedRoles, nil)

			// 创建请求（带关键词搜索）
			req := httptest.NewRequest("GET", "/api/authorization/v1/resource_type_roles?resource_type_id=doc&keyword=管理员&offset=0&limit=10", nil)
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

		Convey("业务逻辑层返回错误", func() {
			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			mockRole.EXPECT().GetResourceTypeRoles(gomock.Any(), gomock.Any(), gomock.Any()).Return(0, nil, gerrors.NewError(gerrors.PublicInternalServerError, "internal error"))

			// 创建请求
			req := httptest.NewRequest("GET", "/api/authorization/v1/resource_type_roles?resource_type_id=doc&offset=0&limit=10", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusInternalServerError)
		})

		Convey("无限制资源类型范围的角色", func() {
			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			expectedRoles := []interfaces.RoleInfo{
				{
					ID:          "role1",
					Name:        "超级管理员",
					Description: "拥有所有权限",
					ResourceTypeScopeInfo: interfaces.ResourceTypeScopeInfo{
						Unlimited: true,
						Types:     []interfaces.ResourceTypeScope{},
					},
				},
			}

			mockRole.EXPECT().GetResourceTypeRoles(gomock.Any(), gomock.Any(), gomock.Any()).Return(1, expectedRoles, nil)

			// 创建请求
			req := httptest.NewRequest("GET", "/api/authorization/v1/resource_type_roles?resource_type_id=doc&offset=0&limit=10", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusOK)

			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)

			entries := response["entries"].([]any)
			So(len(entries), ShouldEqual, 1)
		})

		Convey("分页参数测试", func() {
			// 设置 mock 期望
			mockHydra.EXPECT().Introspect("test-token").Return(interfaces.TokenIntrospectInfo{
				Active:     true,
				VisitorID:  "admin1",
				VisitorTyp: interfaces.RealName,
			}, nil)

			expectedRoles := []interfaces.RoleInfo{}

			mockRole.EXPECT().GetResourceTypeRoles(gomock.Any(), gomock.Any(), gomock.Any()).Return(0, expectedRoles, nil)

			// 创建请求（自定义分页参数）
			req := httptest.NewRequest("GET", "/api/authorization/v1/resource_type_roles?resource_type_id=doc&offset=10&limit=5", nil)
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
			So(len(response["entries"].([]any)), ShouldEqual, 0)
		})
	})
}

//nolint:funlen
func TestRoleRestHandler_GetAccessorRoles(t *testing.T) {
	Convey("getAccessorRoles", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// 创建 mock 对象
		mockRole := mock.NewMockLogicsRole(ctrl)

		// 创建测试处理器
		handler := &roleRestHandler{
			role: mockRole,
			accessorRoleStrToIntMap: map[string]interfaces.AccessorType{
				"user": interfaces.AccessorUser,
			},
			strToRoleSourceMap: map[string]interfaces.RoleSource{
				"system":   interfaces.RoleSourceSystem,
				"business": interfaces.RoleSourceBusiness,
				"user":     interfaces.RoleSourceUser,
			},
		}

		handler.RegisterPrivate(r)

		Convey("成功获取角色列表", func() {
			// 设置 mock 期望
			expectedRoles := []interfaces.RoleInfo{
				{
					ID:          "role1",
					Name:        "测试角色1",
					Description: "这是测试角色1",
				},
				{
					ID:          "role2",
					Name:        "测试角色2",
					Description: "这是测试角色2",
				},
			}

			mockRole.EXPECT().GetAccessorRoles(gomock.Any(), gomock.Any()).Return(2, expectedRoles, nil)

			// 创建请求
			req := httptest.NewRequest("GET", "/api/authorization/v1/accessor_roles?accessor_id=user1&accessor_type=user&offset=0&limit=10", nil)
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

			// 验证第一个角色
			role1 := entries[0].(map[string]any)
			So(role1["id"], ShouldEqual, "role1")
			So(role1["name"], ShouldEqual, "测试角色1")
			So(role1["description"], ShouldEqual, "这是测试角色1")
		})

		Convey("缺少accessor_id参数", func() {
			// 创建请求（没有 accessor_id 参数）
			req := httptest.NewRequest("GET", "/api/authorization/v1/accessor_roles?accessor_type=user", nil)
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("空的accessor_id参数", func() {
			// 创建请求（空的 accessor_id 参数）
			req := httptest.NewRequest("GET", "/api/authorization/v1/accessor_roles?accessor_id=&accessor_type=user", nil)
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("缺少accessor_type参数", func() {
			// 创建请求（没有 accessor_type 参数）
			req := httptest.NewRequest("GET", "/api/authorization/v1/accessor_roles?accessor_id=user1", nil)
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("非法的accessor_type参数", func() {
			// 创建请求（非法的 accessor_type 参数）
			req := httptest.NewRequest("GET", "/api/authorization/v1/accessor_roles?accessor_id=user1&accessor_type=invalid", nil)
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("无效的offset参数（负数）", func() {
			// 创建请求（负数的 offset）
			req := httptest.NewRequest("GET", "/api/authorization/v1/accessor_roles?accessor_id=user1&accessor_type=user&offset=-1", nil)
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("无效的limit参数（小于1）", func() {
			// 创建请求（limit 小于 1）
			req := httptest.NewRequest("GET", "/api/authorization/v1/accessor_roles?accessor_id=user1&accessor_type=user&limit=0", nil)
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("无效的limit参数（大于1000）", func() {
			// 创建请求（limit 大于 1000）
			req := httptest.NewRequest("GET", "/api/authorization/v1/accessor_roles?accessor_id=user1&accessor_type=user&limit=1001", nil)
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("limit为-1（不限制）", func() {
			// 设置 mock 期望
			expectedRoles := []interfaces.RoleInfo{
				{
					ID:          "role1",
					Name:        "测试角色1",
					Description: "这是测试角色1",
				},
			}

			mockRole.EXPECT().GetAccessorRoles(gomock.Any(), gomock.Any()).Return(1, expectedRoles, nil)

			// 创建请求（limit 为 -1）
			req := httptest.NewRequest("GET", "/api/authorization/v1/accessor_roles?accessor_id=user1&accessor_type=user&limit=-1", nil)
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

		Convey("非法的source参数", func() {
			// 创建请求（非法的 source 参数）
			req := httptest.NewRequest("GET", "/api/authorization/v1/accessor_roles?accessor_id=user1&accessor_type=user&source=invalid", nil)
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("带source过滤", func() {
			// 设置 mock 期望
			expectedRoles := []interfaces.RoleInfo{
				{
					ID:          "role1",
					Name:        "系统角色",
					Description: "系统角色描述",
				},
			}

			mockRole.EXPECT().GetAccessorRoles(gomock.Any(), gomock.Any()).Return(1, expectedRoles, nil)

			// 创建请求（带 source 过滤）
			req := httptest.NewRequest("GET", "/api/authorization/v1/accessor_roles?accessor_id=user1&accessor_type=user&source=system", nil)
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

		Convey("多个source参数", func() {
			// 设置 mock 期望
			expectedRoles := []interfaces.RoleInfo{
				{
					ID:          "role1",
					Name:        "业务角色",
					Description: "业务角色描述",
				},
				{
					ID:          "role2",
					Name:        "用户角色",
					Description: "用户角色描述",
				},
			}

			mockRole.EXPECT().GetAccessorRoles(gomock.Any(), gomock.Any()).Return(2, expectedRoles, nil)

			// 创建请求（多个 source 参数）
			req := httptest.NewRequest("GET", "/api/authorization/v1/accessor_roles?accessor_id=user1&accessor_type=user&source=business&source=user", nil)
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusOK)

			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(response["total_count"], ShouldEqual, float64(2))
		})

		Convey("不带source参数（默认返回business和user）", func() {
			// 设置 mock 期望
			expectedRoles := []interfaces.RoleInfo{
				{
					ID:          "role1",
					Name:        "默认角色",
					Description: "默认角色描述",
				},
			}

			mockRole.EXPECT().GetAccessorRoles(gomock.Any(), gomock.Any()).Return(1, expectedRoles, nil)

			// 创建请求（不带 source 参数，应该默认使用 business 和 user）
			req := httptest.NewRequest("GET", "/api/authorization/v1/accessor_roles?accessor_id=user1&accessor_type=user", nil)
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

		Convey("业务逻辑层返回错误", func() {
			// 设置 mock 期望
			mockRole.EXPECT().GetAccessorRoles(gomock.Any(), gomock.Any()).Return(0, nil, gerrors.NewError(gerrors.PublicInternalServerError, "internal error"))

			// 创建请求
			req := httptest.NewRequest("GET", "/api/authorization/v1/accessor_roles?accessor_id=user1&accessor_type=user", nil)
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusInternalServerError)
		})

		Convey("分页参数测试", func() {
			// 设置 mock 期望
			expectedRoles := []interfaces.RoleInfo{}

			mockRole.EXPECT().GetAccessorRoles(gomock.Any(), gomock.Any()).Return(0, expectedRoles, nil)

			// 创建请求（自定义分页参数）
			req := httptest.NewRequest("GET", "/api/authorization/v1/accessor_roles?accessor_id=user1&accessor_type=user&offset=10&limit=5", nil)
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证结果
			So(w.Code, ShouldEqual, http.StatusOK)

			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(response["total_count"], ShouldEqual, float64(0))
			So(len(response["entries"].([]any)), ShouldEqual, 0)
		})

		Convey("默认分页参数测试", func() {
			// 设置 mock 期望
			expectedRoles := []interfaces.RoleInfo{}

			mockRole.EXPECT().GetAccessorRoles(gomock.Any(), gomock.Any()).Return(0, expectedRoles, nil)

			// 创建请求（不传 offset 和 limit，应该使用默认值）
			req := httptest.NewRequest("GET", "/api/authorization/v1/accessor_roles?accessor_id=user1&accessor_type=user", nil)
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
	})
}

func TestRoleRestHandler_ConvertChildrenResourceTypeScope(t *testing.T) {
	Convey("convertChildrenResourceTypeScope", t, func() {
		handler := &roleRestHandler{}

		Convey("空children返回空切片", func() {
			result := handler.convertChildrenResourceTypeScope(nil)
			So(result, ShouldBeEmpty)
			result = handler.convertChildrenResourceTypeScope([]interfaces.ResourceTypeScopeWithOperation{})
			So(result, ShouldBeEmpty)
		})

		Convey("单个子节点无嵌套", func() {
			children := []interfaces.ResourceTypeScopeWithOperation{
				{
					ID:          "menu",
					Name:        "菜单",
					Description: "菜单资源",
					InstanceURL: "/menu",
					DataStruct:  "{}",
					TypeOperation: []interfaces.ResourceTypeOperationResponse{
						{ID: "read", Name: "读", Description: "读取"},
					},
					InstanceOperation: []interfaces.ResourceTypeOperationResponse{
						{ID: "write", Name: "写", Description: "写入"},
					},
					Children: nil,
				},
			}
			result := handler.convertChildrenResourceTypeScope(children)
			So(len(result), ShouldEqual, 1)
			childInfo := result[0].(map[string]any)
			So(childInfo["id"], ShouldEqual, "menu")
			So(childInfo["name"], ShouldEqual, "菜单")
			So(childInfo["description"], ShouldEqual, "菜单资源")
			So(childInfo["instance_url"], ShouldEqual, "/menu")
			So(childInfo["data_struct"], ShouldEqual, "{}")
			operation := childInfo["operation"].(map[string]any)
			typeOps := operation["type"].([]any)
			So(len(typeOps), ShouldEqual, 1)
			So(typeOps[0].(map[string]any)["id"], ShouldEqual, "read")
			instanceOps := operation["instance"].([]any)
			So(len(instanceOps), ShouldEqual, 1)
			So(instanceOps[0].(map[string]any)["id"], ShouldEqual, "write")
		})

		Convey("递归嵌套children", func() {
			children := []interfaces.ResourceTypeScopeWithOperation{
				{
					ID:          "parent",
					Name:        "父级",
					Description: "",
					InstanceURL: "",
					DataStruct:  "",
					TypeOperation: []interfaces.ResourceTypeOperationResponse{
						{ID: "read", Name: "读", Description: ""},
					},
					InstanceOperation: nil,
					Children: []interfaces.ResourceTypeScopeWithOperation{
						{
							ID:                "child",
							Name:              "子级",
							Description:       "",
							InstanceURL:       "",
							DataStruct:        "",
							TypeOperation:     nil,
							InstanceOperation: nil,
							Children:          nil,
						},
					},
				},
			}
			result := handler.convertChildrenResourceTypeScope(children)
			So(len(result), ShouldEqual, 1)
			parentInfo := result[0].(map[string]any)
			So(parentInfo["id"], ShouldEqual, "parent")
			So(parentInfo["children"], ShouldNotBeNil)
			nestedChildren := parentInfo["children"].([]any)
			So(len(nestedChildren), ShouldEqual, 1)
			childInfo := nestedChildren[0].(map[string]any)
			So(childInfo["id"], ShouldEqual, "child")
			So(childInfo["name"], ShouldEqual, "子级")
		})
	})
}
