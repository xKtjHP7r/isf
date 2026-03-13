// Package main 主程序
//
//nolint:dupl
package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/go-lib/rest"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/assert"
	jsoniter "github.com/json-iterator/go"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"

	uerrors "devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/errors"
	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/interfaces"
)

var (
	struuid = "151bcb65-48ce-4b62-973f-0bb6685f9cb8"
)

//nolint:dupl
func TestGetOrgPermAPP(t *testing.T) {
	newTestUserManagement(t)
	Convey("获取应用账户权限", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		hydraMock, _ := initHTTPMock(ctrl)
		Convey("token验证失败,报错401000000", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, passiveUserToken)

			req := httptest.NewRequest("GET", "/api/user-management/v1/app-perms/"+struuid+"/user", http.NoBody)
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			err := jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, err, nil)
			assert.Equal(t, result.StatusCode, http.StatusUnauthorized)
			assert.Equal(t, resBody["code"].(float64), float64(rest.Unauthorized))
			assert.Equal(t, resBody["cause"].(string), "token expired")

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("普通用户token，验证失败,报错403000000", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)
			mock.ExpectQuery("select f_user_id, f_role_id from sharemgnt_db.t_user_role_relation where f_user_id in").
				WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}))

			req := httptest.NewRequest("GET", "/api/user-management/v1/app-perms/"+struuid+"/user", http.NoBody)
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			err := jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, err, nil)
			assert.Equal(t, result.StatusCode, http.StatusForbidden)
			assert.Equal(t, resBody["code"].(float64), float64(rest.Forbidden))
			assert.Equal(t, resBody["cause"].(string), "this user do not has the authority")

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("安全管理员token，验证失败,报错403000000", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)
			mock.ExpectQuery("select f_user_id, f_role_id from sharemgnt_db.t_user_role_relation where f_user_id in").
				WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}).AddRow(activeUserID, interfaces.SystemRoleSecAdmin))

			req := httptest.NewRequest("GET", "/api/user-management/v1/app-perms/"+struuid+"/user", http.NoBody)
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			err := jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, err, nil)
			assert.Equal(t, result.StatusCode, http.StatusForbidden)
			assert.Equal(t, resBody["code"].(float64), float64(rest.Forbidden))
			assert.Equal(t, resBody["cause"].(string), "this user do not has the authority")

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("审计管理员token，验证失败,报错403000000", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)
			mock.ExpectQuery("select f_user_id, f_role_id from sharemgnt_db.t_user_role_relation where f_user_id in").
				WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}).AddRow(activeUserID, interfaces.SystemRoleAuditAdmin))

			req := httptest.NewRequest("GET", "/api/user-management/v1/app-perms/"+struuid+"/user", http.NoBody)
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			err := jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, err, nil)
			assert.Equal(t, result.StatusCode, http.StatusForbidden)
			assert.Equal(t, resBody["code"].(float64), float64(rest.Forbidden))
			assert.Equal(t, resBody["cause"].(string), "this user do not has the authority")

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("应用账户不存在,报错404000000", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)
			mock.ExpectQuery("select f_user_id, f_role_id from sharemgnt_db.t_user_role_relation where f_user_id in").
				WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}).AddRow(activeUserID, interfaces.SystemRoleSuperAdmin))
			mock.ExpectQuery("select f_id, f_name, f_credential_type from user_management.t_app where f_id").
				WillReturnRows(sqlmock.NewRows([]string{"f_id", "f_name", "f_credential_type"}))

			req := httptest.NewRequest("GET", "/api/user-management/v1/app-perms/"+struuid+"/user", http.NoBody)
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			err := jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, err, nil)
			assert.Equal(t, result.StatusCode, http.StatusNotFound)
			assert.Equal(t, resBody["code"].(float64), float64(uerrors.NotFound))
			assert.Equal(t, resBody["cause"].(string), "id does not exist")

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("objects客体对象重复,报错400000000", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)

			req := httptest.NewRequest("GET", "/api/user-management/v1/app-perms/"+struuid+"/user,user,department", http.NoBody)
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			err := jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, err, nil)
			assert.Equal(t, result.StatusCode, http.StatusBadRequest)
			assert.Equal(t, resBody["code"].(float64), float64(rest.BadRequest))
			assert.Equal(t, resBody["cause"].(string), "org type is not uniqued")

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	})
}

func TestDeleteOrgPermAPP(t *testing.T) {
	newTestUserManagement(t)
	Convey("删除应用账户权限", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		hydraMock, _ := initHTTPMock(ctrl)
		Convey("token验证失败,报错401000000", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, passiveUserToken)

			req := httptest.NewRequest("DELETE", "/api/user-management/v1/app-perms/"+struuid+"/user", http.NoBody)
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			err := jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, err, nil)
			assert.Equal(t, result.StatusCode, http.StatusUnauthorized)
			assert.Equal(t, resBody["code"].(float64), float64(rest.Unauthorized))
			assert.Equal(t, resBody["cause"].(string), "token expired")

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("普通用户token，验证失败,报错403000000", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)
			mock.ExpectQuery("select f_user_id, f_role_id from sharemgnt_db.t_user_role_relation where f_user_id in").
				WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}))

			req := httptest.NewRequest("DELETE", "/api/user-management/v1/app-perms/"+struuid+"/user", http.NoBody)
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			err := jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, err, nil)
			assert.Equal(t, result.StatusCode, http.StatusForbidden)
			assert.Equal(t, resBody["code"].(float64), float64(rest.Forbidden))
			assert.Equal(t, resBody["cause"].(string), "this user do not has the authority")

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("审计管理员token，验证失败,报错403000000", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)
			mock.ExpectQuery("select f_user_id, f_role_id from sharemgnt_db.t_user_role_relation where f_user_id in").
				WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}).AddRow(activeUserID, interfaces.SystemRoleAuditAdmin))

			req := httptest.NewRequest("DELETE", "/api/user-management/v1/app-perms/"+struuid+"/user", http.NoBody)
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			err := jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, err, nil)
			assert.Equal(t, result.StatusCode, http.StatusForbidden)
			assert.Equal(t, resBody["code"].(float64), float64(rest.Forbidden))
			assert.Equal(t, resBody["cause"].(string), "this user do not has the authority")

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("安全管理员token，验证失败,报错403000000", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)
			mock.ExpectQuery("select f_user_id, f_role_id from sharemgnt_db.t_user_role_relation where f_user_id in").
				WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}).AddRow(activeUserID, interfaces.SystemRoleSecAdmin))

			req := httptest.NewRequest("DELETE", "/api/user-management/v1/app-perms/"+struuid+"/user", http.NoBody)
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			err := jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, err, nil)
			assert.Equal(t, result.StatusCode, http.StatusForbidden)
			assert.Equal(t, resBody["code"].(float64), float64(rest.Forbidden))
			assert.Equal(t, resBody["cause"].(string), "this user do not has the authority")

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("应用账户不存在,报错404000000", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)
			mock.ExpectQuery("select f_user_id, f_role_id from sharemgnt_db.t_user_role_relation where f_user_id in").
				WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}).AddRow(activeUserID, interfaces.SystemRoleSuperAdmin))
			mock.ExpectQuery("select f_id, f_name, f_credential_type from user_management.t_app where f_id").
				WillReturnRows(sqlmock.NewRows([]string{"f_id", "f_name", "f_credential_type"}))

			req := httptest.NewRequest("DELETE", "/api/user-management/v1/app-perms/"+struuid+"/user", http.NoBody)
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			err := jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, err, nil)
			assert.Equal(t, result.StatusCode, http.StatusNotFound)
			assert.Equal(t, resBody["code"].(float64), float64(uerrors.NotFound))
			assert.Equal(t, resBody["cause"].(string), "id does not exist")

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("objects客体对象重复,报错400000000", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)

			req := httptest.NewRequest("DELETE", "/api/user-management/v1/app-perms/"+struuid+"/user,user,department", http.NoBody)
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			err := jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, err, nil)
			assert.Equal(t, result.StatusCode, http.StatusBadRequest)
			assert.Equal(t, resBody["code"].(float64), float64(rest.BadRequest))
			assert.Equal(t, resBody["cause"].(string), "org type is not uniqued")

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	})
}

func TestUpdateOrgPermAPP(t *testing.T) {
	newTestUserManagement(t)
	Convey("更新应用账户权限", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		hydraMock, _ := initHTTPMock(ctrl)
		Convey("token验证失败,报错401000000", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, passiveUserToken)

			req := httptest.NewRequest("PUT", "/api/user-management/v1/app-perms/"+struuid+"/user", http.NoBody)
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			err := jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, err, nil)
			assert.Equal(t, result.StatusCode, http.StatusUnauthorized)
			assert.Equal(t, resBody["code"].(float64), float64(rest.Unauthorized))
			assert.Equal(t, resBody["cause"].(string), "token expired")

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		data1 := gin.H{
			"subject": struuid,
			"object":  "user",
			"perms": []string{
				"read",
			},
		}
		jsonData, _ := jsoniter.Marshal([]map[string]interface{}{data1})
		Convey("普通用户token，验证失败,报错403000000", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)
			mock.ExpectQuery("select f_user_id, f_role_id from sharemgnt_db.t_user_role_relation where f_user_id in").
				WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}))

			req := httptest.NewRequest("PUT", "/api/user-management/v1/app-perms/"+struuid+"/user", bytes.NewReader(jsonData))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			err := jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, err, nil)
			assert.Equal(t, result.StatusCode, http.StatusForbidden)
			assert.Equal(t, resBody["code"].(float64), float64(rest.Forbidden))
			assert.Equal(t, resBody["cause"].(string), "this user do not has the authority")

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("安全管理员token，验证失败,报错403000000", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)
			mock.ExpectQuery("select f_user_id, f_role_id from sharemgnt_db.t_user_role_relation where f_user_id in").
				WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}).AddRow(activeUserID, interfaces.SystemSecAdmin))

			req := httptest.NewRequest("PUT", "/api/user-management/v1/app-perms/"+struuid+"/user", bytes.NewReader(jsonData))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			err := jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, err, nil)
			assert.Equal(t, result.StatusCode, http.StatusForbidden)
			assert.Equal(t, resBody["code"].(float64), float64(rest.Forbidden))
			assert.Equal(t, resBody["cause"].(string), "this user do not has the authority")

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("审计管理员token，验证失败,报错403000000", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)
			mock.ExpectQuery("select f_user_id, f_role_id from sharemgnt_db.t_user_role_relation where f_user_id in").
				WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}).AddRow(activeUserID, interfaces.SystemAuditAdmin))

			req := httptest.NewRequest("PUT", "/api/user-management/v1/app-perms/"+struuid+"/user", bytes.NewReader(jsonData))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			err := jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, err, nil)
			assert.Equal(t, result.StatusCode, http.StatusForbidden)
			assert.Equal(t, resBody["code"].(float64), float64(rest.Forbidden))
			assert.Equal(t, resBody["cause"].(string), "this user do not has the authority")

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	})
}
