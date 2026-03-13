// Package main 主程序
package main

import (
	"bytes"
	"database/sql"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/go-lib/rest"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-playground/assert"
	jsoniter "github.com/json-iterator/go"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"

	uerrors "devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/errors"
	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/interfaces"
)

var (
	cRSA2048 = "OnPfN16wF+kw6dck2HyR/CLzftN80LLext5ao/PPkQy3qLlOFc6WAzuDOYT+wrp/IMNfJ8QVR8G2yfnT/MpBKgpcYrLSzez2Sybf7Gul820zv2h4w4zbPt7" +
		"zwdEzAZofw7ZBcys1TwO3TzZV/5e00Wkp3w0rPfL9Mq34Lozz7yGjIQtUtdBtMsQPMkEGKhnw907SEfvFxU6HSKV3GAHPS3iEulMPs0A/XG7S4/2i69W30GU17dV9OaRvAxYmWttv1EuW/O1" +
		"AjboahH1UJP/XQyCFnwZH0Aepoz4kXm7UpxjiGOZZgcWuuJy+k3Rri7sl1w4GKdF0nj7PjxnoG7y3nQ=="
)

func TestGeneralAppRegisterWithoutOutbox1(t *testing.T) {
	newTestUserManagement(t)

	Convey("通用账户注册", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		tmpBody := map[string]interface{}{
			"name":     "client_1",
			"password": cRSA2048,
		}
		reqBody, _ := jsoniter.Marshal(tmpBody)

		hydraMock, _ := initHTTPMock(ctrl)

		// 允许以任意顺序匹配期望
		mock.MatchExpectationsInOrder(false)

		Convey("令牌失效", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, passiveUserToken)
			req := httptest.NewRequest("POST", "/api/user-management/v1/apps", bytes.NewReader(reqBody))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			_ = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, resBody["code"].(float64), float64(rest.Unauthorized))
			assert.Equal(t, result.StatusCode, http.StatusUnauthorized)
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("普通用户注册通用应用账户注册，抛错", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)
			// 用户角色期望
			mock.ExpectQuery("^select f_user_id, f_role_id").WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}).AddRow("not-admin", "xxx-xxx-xxx-xxx"))

			req := httptest.NewRequest("POST", "/api/user-management/v1/apps", bytes.NewReader(reqBody))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			_ = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, resBody["code"].(float64), float64(uerrors.Forbidden))
			assert.Equal(t, result.StatusCode, http.StatusForbidden)
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("密码未加密，报错", func() {
			tmpBody1 := map[string]interface{}{
				"name":     "client_1",
				"password": "test11test",
			}
			reqBody1, _ := jsoniter.Marshal(tmpBody1)
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)

			req := httptest.NewRequest("POST", "/api/user-management/v1/apps", bytes.NewReader(reqBody1))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			_ = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, resBody["cause"], "illegal base64 data at input byte 8")
			assert.Equal(t, result.StatusCode, http.StatusBadRequest)
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	})
}

func mockRequest(needBody bool, method, target string, body io.Reader, r http.Handler) (result *http.Response, resBody map[string]interface{}) {
	req := httptest.NewRequest(method, target, body)
	req.Header.Add("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	result = w.Result()

	if needBody {
		message, _ := io.ReadAll(result.Body)
		_ = jsoniter.Unmarshal(message, &resBody)
	}

	return
}

func nameExist(duplicateUsername bool, reqBody io.Reader, r http.Handler, t *testing.T) {
	mock.MatchExpectationsInOrder(false)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	hydraMock, _ := initHTTPMock(ctrl)

	hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, adminUserToken)
	if r == publicEngine {
		mock.ExpectQuery("^select f_user_id, f_role_id").WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}).AddRow(interfaces.SystemSysAdmin, "7dcfcc9c-ad02-11e8-aa06-000c29358ad6"))
	}

	// 与其他应用账户重名期望
	if duplicateUsername {
		mock.ExpectQuery("where f_name").WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery("^select 1").WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow("1"))
	} else {
		mock.ExpectQuery("where f_name").WillReturnRows(sqlmock.NewRows([]string{"f_id", "f_name"}).AddRow("client_1", "tmp-name"))
	}

	result, resBody := mockRequest(true, "POST", "/api/user-management/v1/apps", reqBody, r)

	assert.Equal(t, resBody["code"].(float64), float64(uerrors.Conflict))

	assert.Equal(t, result.StatusCode, http.StatusConflict)
	if err := result.Body.Close(); err != nil {
		assert.Equal(t, err, nil)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGeneralAppRegisterWithoutOutbox2(t *testing.T) {
	newTestUserManagement(t)

	Convey("通用账户注册", t, func() {
		tmpBody := map[string]interface{}{
			"name":     "client_1",
			"password": cRSA2048,
		}

		// 允许以任意顺序匹配期望
		mock.MatchExpectationsInOrder(false)

		Convey("name已存在（重复注册），抛错", func() {
			reqBody, _ := jsoniter.Marshal(tmpBody)
			nameExist(false, bytes.NewReader(reqBody), publicEngine, t)
		})

		Convey("name与普通用户登录名相同/name为admin/system/security/audit", func() {
			tmpBody["name"] = "client_3"
			reqBody, _ := jsoniter.Marshal(tmpBody)
			nameExist(true, bytes.NewReader(reqBody), publicEngine, t)
		})
	})
}

func TestSpecifiedAppRegisterWithoutOutbox(t *testing.T) {
	newTestUserManagement(t)

	Convey("专用账户注册", t, func() {
		tmpBody := map[string]interface{}{
			"name":     "client_1",
			"password": "some-secret",
			"type":     "specified",
		}

		// 允许以任意顺序匹配期望
		mock.MatchExpectationsInOrder(false)

		Convey("name已存在（重复注册），抛错", func() {
			reqBody, _ := jsoniter.Marshal(tmpBody)
			nameExist(false, bytes.NewReader(reqBody), privateEngine, t)
		})

		Convey("name与普通用户登录名相同/name为admin/system/security/audit", func() {
			tmpBody["name"] = "client_3"
			reqBody, _ := jsoniter.Marshal(tmpBody)
			nameExist(true, bytes.NewReader(reqBody), privateEngine, t)
		})
	})
}

func TestDeleteApp1(t *testing.T) {
	newTestUserManagement(t)

	Convey("删除应用账户", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		hydraMock, _ := initHTTPMock(ctrl)

		// 允许以任意顺序匹配期望
		mock.MatchExpectationsInOrder(false)

		Convey("不传id，抛错/id传空字符串/None，抛错", func() {
			result, _ := mockRequest(false, "DELETE", "/api/user-management/v1/apps/", nil, publicEngine)

			assert.Equal(t, result.StatusCode, http.StatusNotFound)
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("db delete app failed", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, adminUserToken)
			mock.ExpectQuery("^select f_user_id, f_role_id").WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}).AddRow(interfaces.SystemSysAdmin, "7dcfcc9c-ad02-11e8-aa06-000c29358ad6"))
			mock.ExpectQuery("select f_id, f_name, f_credential_type from user_management.t_app where f_id").WillReturnError(sql.ErrNoRows)

			req := httptest.NewRequest("DELETE", "/api/user-management/v1/apps/xxxxxxxx", http.NoBody)
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			assert.Equal(t, result.StatusCode, http.StatusNoContent)
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("应用账户id不存在", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, adminUserToken)
			mock.ExpectQuery("^select f_user_id, f_role_id").WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}).AddRow(interfaces.SystemSysAdmin, "7dcfcc9c-ad02-11e8-aa06-000c29358ad6"))
			mock.ExpectQuery("select f_id, f_name, f_credential_type from user_management.t_app where f_id").WillReturnError(sql.ErrNoRows)

			req := httptest.NewRequest("DELETE", "/api/user-management/v1/apps/xxxxxxxx", http.NoBody)
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			assert.Equal(t, result.StatusCode, http.StatusNoContent)
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			start := time.Now()
			for {
				err := mock.ExpectationsWereMet()
				if err == nil {
					break
				}
				if time.Until(start).Seconds() < -1 {
					t.Errorf("timeout and there were unfulfilled expectations: %v", err)
					break
				}
			}
		})
	})
}

func TestDeleteApp2(t *testing.T) {
	newTestUserManagement(t)

	Convey("Test Delete App", t, func() {
		contentJSON := map[string]interface{}{
			"id": "xxx-xxx-xxx-xxx",
		}
		outboxJSON := map[string]interface{}{
			"type":    3,
			"content": contentJSON,
		}
		outboxMsg, _ := jsoniter.MarshalToString(outboxJSON)
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		hydraMock, _ := initHTTPMock(ctrl)

		// 允许以任意顺序匹配期望
		mock.MatchExpectationsInOrder(false)

		Convey("hydra delete app failed", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, adminUserToken)
			hydraMock.EXPECT().HandleRequest("DELETE", "/admin/clients/xxx-xxx-xxx-xxx", nil).AnyTimes().Return(http.StatusInternalServerError, nil)
			// 设置由逻辑层触发的outbox开启事务期望
			mock.ExpectBegin()
			mock.ExpectQuery("select f_business_type from user_management.t_outbox_lock").WillReturnRows(sqlmock.NewRows([]string{"f_business_type"}).AddRow(123))
			mock.ExpectQuery("select f_id, f_message from user_management.t_outbox where f_business_type =").WillReturnRows(sqlmock.NewRows([]string{"f_id", "f_message"}).AddRow(123, outboxMsg))
			mock.ExpectRollback()

			mock.ExpectQuery("^select f_user_id, f_role_id").WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}).AddRow(interfaces.SystemSysAdmin, "7dcfcc9c-ad02-11e8-aa06-000c29358ad6"))
			mock.ExpectQuery("select f_id, f_name, f_credential_type from user_management.t_app where f_id").
				WillReturnRows(sqlmock.NewRows([]string{"f_id", "f_name", "f_credential_type"}).AddRow("xxx-xxx-xxx", "client_1", interfaces.CredentialTypePassword))
			mock.ExpectBegin()
			mock.ExpectExec("into user_management.t_outbox").WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec("into user_management.t_outbox").WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec("delete from user_management.t_app where").WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec("into user_management.t_outbox").WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()

			req := httptest.NewRequest("DELETE", "/api/user-management/v1/apps/xxxxxxxx", http.NoBody)
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			assert.Equal(t, result.StatusCode, http.StatusNoContent)
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			start := time.Now()
			for {
				err := mock.ExpectationsWereMet()
				if err == nil {
					break
				}
				if time.Until(start).Seconds() < -1 {
					t.Errorf("timeout and there were unfulfilled expectations: %v", err)
					break
				}
			}
		})
	})
}

func TestUpdateAppWithoutOutbox(t *testing.T) {
	newTestUserManagement(t)

	Convey("更新应用账户", t, func() {
		tmpBody := map[string]interface{}{
			"name": "client_2",
		}
		reqBody, _ := jsoniter.Marshal(tmpBody)

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		hydraMock, _ := initHTTPMock(ctrl)
		// 允许以任意顺序匹配期望
		mock.MatchExpectationsInOrder(false)

		Convey("id does not exist", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, adminUserToken)
			mock.ExpectQuery("^select f_user_id, f_role_id").WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}).AddRow(interfaces.SystemSysAdmin, "7dcfcc9c-ad02-11e8-aa06-000c29358ad6"))
			mock.ExpectQuery("select f_id, f_name, f_credential_type from user_management.t_app where f_id").WillReturnError(sql.ErrNoRows)

			req := httptest.NewRequest("PUT", "/api/user-management/v1/apps/xxx-xxx-xxx/name", bytes.NewReader(reqBody))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			_ = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, resBody["code"].(float64), float64(uerrors.NotFound))
			assert.Equal(t, result.StatusCode, http.StatusNotFound)
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	})
}

func TestUpdateAppWithoutOutbox1(t *testing.T) {
	newTestUserManagement(t)

	Convey("更新应用账户", t, func() {
		tmpBody := map[string]interface{}{
			"name": "client_2",
		}
		reqBody, _ := jsoniter.Marshal(tmpBody)
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		hydraMock, _ := initHTTPMock(ctrl)

		// 允许以任意顺序匹配期望
		mock.MatchExpectationsInOrder(false)

		Convey("待更新的账户名已存在（与其他应用账户重名）", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, adminUserToken)
			mock.ExpectQuery("^select f_user_id, f_role_id").WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}).AddRow(interfaces.SystemSysAdmin, "7dcfcc9c-ad02-11e8-aa06-000c29358ad6"))
			mock.ExpectQuery("select f_id, f_name, f_credential_type from user_management.t_app where f_id").
				WillReturnRows(sqlmock.NewRows([]string{"f_id", "f_name", "f_credential_type"}).AddRow("xxx-xxx-xxx", "client_1", interfaces.CredentialTypePassword))
			mock.ExpectQuery("select f_id, f_name from user_management.t_app where f_name").WillReturnRows(sqlmock.NewRows([]string{"f_id", "f_name"}).AddRow("xxx-xxx-xxx1", "client_2"))

			req := httptest.NewRequest("PUT", "/api/user-management/v1/apps/xxx-xxx-xxx/name", bytes.NewReader(reqBody))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			_ = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, resBody["code"].(float64), float64(uerrors.Conflict))
			assert.Equal(t, result.StatusCode, http.StatusConflict)
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("待更新的name与AS系统中账户名相同，抛错", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, adminUserToken)
			mock.ExpectQuery("^select f_user_id, f_role_id").WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}).AddRow(interfaces.SystemSysAdmin, "7dcfcc9c-ad02-11e8-aa06-000c29358ad6"))
			mock.ExpectQuery("select f_id, f_name, f_credential_type from user_management.t_app where f_id").
				WillReturnRows(sqlmock.NewRows([]string{"f_id", "f_name", "f_credential_type"}).AddRow("xxx-xxx-xxx", "client_1", interfaces.CredentialTypePassword))
			mock.ExpectQuery("select f_id, f_name from user_management.t_app where f_name").WillReturnError(sql.ErrNoRows)
			mock.ExpectQuery("^select 1").WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow("1"))

			req := httptest.NewRequest("PUT", "/api/user-management/v1/apps/xxx-xxx-xxx/name", bytes.NewReader(reqBody))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			_ = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, resBody["code"].(float64), float64(uerrors.Conflict))
			assert.Equal(t, result.StatusCode, http.StatusConflict)
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	})
}

//nolint:dupl
func TestUpdateAppOutbox1(t *testing.T) {
	newTestUserManagement(t)

	Convey("更新应用账户", t, func() {
		tmpBody := map[string]interface{}{
			"name":     "client_2",
			"password": cRSA2048,
		}
		reqBody, _ := jsoniter.Marshal(tmpBody)
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		hydraMock, _ := initHTTPMock(ctrl)
		testErr := errors.New("some err")

		// 允许以任意顺序匹配期望
		mock.MatchExpectationsInOrder(true)

		Convey("db update app failed (update name)", func() {
			err := mock.ExpectationsWereMet()
			assert.Equal(t, err, nil)

			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, adminUserToken)
			mock.ExpectQuery("^select f_user_id, f_role_id").WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}).AddRow(interfaces.SystemSysAdmin, "7dcfcc9c-ad02-11e8-aa06-000c29358ad6"))
			mock.ExpectQuery("select f_id, f_name, f_credential_type from user_management.t_app where f_id").
				WillReturnRows(sqlmock.NewRows([]string{"f_id", "f_name", "f_credential_type"}).AddRow("xxx-xxx-xxx", "client_1", interfaces.CredentialTypePassword))
			mock.ExpectQuery("select f_id, f_name from user_management.t_app where f_name").WillReturnError(sql.ErrNoRows)
			mock.ExpectQuery("^select 1").WillReturnError(sql.ErrNoRows)
			mock.ExpectBegin()
			mock.ExpectExec("insert into user_management.t_outbox").WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec("insert into user_management.t_outbox").WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec("update user_management.t_app").WillReturnError(testErr)
			mock.ExpectRollback()

			req := httptest.NewRequest("PUT", "/api/user-management/v1/apps/xxx-xxx-xxx/name", bytes.NewReader(reqBody))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			assert.Equal(t, result.StatusCode, http.StatusInternalServerError)
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("db update app failed (update password)", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, adminUserToken)
			mock.ExpectQuery("^select f_user_id, f_role_id").WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}).AddRow(interfaces.SystemSysAdmin, "7dcfcc9c-ad02-11e8-aa06-000c29358ad6"))
			mock.ExpectQuery("select f_id, f_name, f_credential_type from user_management.t_app where f_id").
				WillReturnRows(sqlmock.NewRows([]string{"f_id", "f_name", "f_credential_type"}).AddRow("xxx-xxx-xxx", "client_2", interfaces.CredentialTypePassword))
			mock.ExpectBegin()
			mock.ExpectExec("into user_management.t_outbox").WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec("update user_management.t_app").WillReturnError(testErr)
			mock.ExpectRollback()

			req := httptest.NewRequest("PUT", "/api/user-management/v1/apps/xxx-xxx-xxx/password", bytes.NewReader(reqBody))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			assert.Equal(t, result.StatusCode, http.StatusInternalServerError)
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("密码未加密，报错", func() {
			tmpBody1 := map[string]interface{}{
				"name":     "client_2",
				"password": "test111tets",
			}
			reqBody1, _ := jsoniter.Marshal(tmpBody1)

			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, adminUserToken)

			req := httptest.NewRequest("PUT", "/api/user-management/v1/apps/xxx-xxx-xxx/password", bytes.NewReader(reqBody1))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			assert.Equal(t, result.StatusCode, http.StatusBadRequest)
			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			_ = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, resBody["cause"], "illegal base64 data at input byte 8")
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("更新aaa应用账户，只更新密码", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, adminUserToken)
			mock.ExpectQuery("^select f_user_id, f_role_id").WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}).AddRow(interfaces.SystemSysAdmin, "7dcfcc9c-ad02-11e8-aa06-000c29358ad6"))
			mock.ExpectQuery("select f_id, f_name, f_credential_type from user_management.t_app where f_id").
				WillReturnRows(sqlmock.NewRows([]string{"f_id", "f_name", "f_credential_type"}).AddRow("xxx-xxx-xxx", "client_3", interfaces.CredentialTypePassword))
			mock.ExpectBegin()
			mock.ExpectExec("insert into user_management.t_outbox").WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec("update user_management.t_app").WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec("insert into user_management.t_outbox").WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()

			req := httptest.NewRequest("PUT", "/api/user-management/v1/apps/xxx-xxx-xxx/password", bytes.NewReader(reqBody))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			assert.Equal(t, result.StatusCode, http.StatusNoContent)
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("更新aaa应用账户，只更新账户名", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, adminUserToken)
			mock.ExpectQuery("^select f_user_id, f_role_id").WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}).AddRow(interfaces.SystemSysAdmin, "7dcfcc9c-ad02-11e8-aa06-000c29358ad6"))
			mock.ExpectQuery("select f_id, f_name, f_credential_type from user_management.t_app where f_id").
				WillReturnRows(sqlmock.NewRows([]string{"f_id", "f_name", "f_credential_type"}).AddRow("xxx-xxx-xxx", "client_3", interfaces.CredentialTypePassword))
			mock.ExpectQuery("select f_id, f_name from user_management.t_app where f_name").WillReturnRows(sqlmock.NewRows([]string{"f_id", "f_name"}))
			mock.ExpectQuery("^select 1").WillReturnError(sql.ErrNoRows)
			mock.ExpectBegin()
			mock.ExpectExec("insert into user_management.t_outbox").WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec("insert into user_management.t_outbox").WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec("update user_management.t_app").WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec("insert into user_management.t_outbox").WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()

			req := httptest.NewRequest("PUT", "/api/user-management/v1/apps/xxx-xxx-xxx/name", bytes.NewReader(reqBody))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			assert.Equal(t, result.StatusCode, http.StatusNoContent)
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("更新aaa应用账户，同时更新账户名和密码/name与普通用户显示名相同", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, adminUserToken)
			mock.ExpectQuery("^select f_user_id, f_role_id").WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}).AddRow(interfaces.SystemSysAdmin, "7dcfcc9c-ad02-11e8-aa06-000c29358ad6"))
			mock.ExpectQuery("select f_id, f_name, f_credential_type from user_management.t_app where f_id").
				WillReturnRows(sqlmock.NewRows([]string{"f_id", "f_name", "f_credential_type"}).AddRow("xxx-xxx-xxx", "client_3", interfaces.CredentialTypePassword))
			mock.ExpectQuery("select f_id, f_name from user_management.t_app where f_name").WillReturnRows(sqlmock.NewRows([]string{"f_id", "f_name"}))
			mock.ExpectQuery("^select 1").WillReturnError(sql.ErrNoRows)
			mock.ExpectBegin()
			mock.ExpectExec("insert into user_management.t_outbox").WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec("insert into user_management.t_outbox").WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec("update user_management.t_app").WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec("insert into user_management.t_outbox").WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()

			req := httptest.NewRequest("PUT", "/api/user-management/v1/apps/xxx-xxx-xxx/name,password", bytes.NewReader(reqBody))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			assert.Equal(t, result.StatusCode, http.StatusNoContent)
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	})
}

func TestAppList(t *testing.T) {
	newTestUserManagement(t)

	Convey("Test App List", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		hydraMock, _ := initHTTPMock(ctrl)
		// 允许以任意顺序匹配期望
		mock.MatchExpectationsInOrder(false)

		Convey("get app list success", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, adminUserToken)
			mock.ExpectQuery("^select f_user_id, f_role_id").WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}).AddRow(interfaces.SystemSysAdmin, "7dcfcc9c-ad02-11e8-aa06-000c29358ad6"))
			mock.ExpectQuery("select count\\(\\*\\) from user_management.t_app where f_type").WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(20))
			mock.ExpectQuery("select f_id, f_name, f_credential_type from user_management.t_app where f_type").
				WillReturnRows(sqlmock.NewRows([]string{"f_id", "f_name", "f_credential_type"}).AddRow("xxx-xxx-xxx", "client_1", interfaces.CredentialTypePassword))

			req := httptest.NewRequest("GET", "/api/user-management/v1/apps", http.NoBody)
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			_ = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, resBody["total_count"].(float64), float64(20))
			assert.Equal(t, resBody["entries"].([]interface{})[0].(map[string]interface{})["name"].(string), "client_1")
			assert.Equal(t, resBody["entries"].([]interface{})[0].(map[string]interface{})["credential_type"].(string), "password")
			assert.Equal(t, result.StatusCode, http.StatusOK)
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	})
}

func TestGetApp(t *testing.T) {
	newTestUserManagement(t)

	Convey("获取应用账户信息", t, func() {
		// 允许以任意顺序匹配期望
		mock.MatchExpectationsInOrder(false)

		Convey("id不是应用账户，获取失败", func() {
			mock.ExpectQuery("select f_id, f_name, f_credential_type from user_management.t_app where f_id").WillReturnError(sql.ErrNoRows)

			req := httptest.NewRequest("GET", "/api/user-management/v1/apps/this-is-id", http.NoBody)
			w := httptest.NewRecorder()
			privateEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			_ = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, resBody["code"].(float64), float64(uerrors.NotFound))
			assert.Equal(t, result.StatusCode, http.StatusNotFound)
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("不传id，抛错/id传空字符串/None，抛错", func() {
			result, _ := mockRequest(false, "GET", "/api/user-management/v1/apps/", nil, privateEngine)

			assert.Equal(t, result.StatusCode, http.StatusNotFound)
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("get app success", func() {
			mock.ExpectQuery("select f_id, f_name, f_credential_type from user_management.t_app where f_id").
				WillReturnRows(sqlmock.NewRows([]string{"f_id", "f_name", "f_credential_type"}).AddRow("this-is-id", "client_1", interfaces.CredentialTypePassword))

			req := httptest.NewRequest("GET", "/api/user-management/v1/apps/this-is-id", http.NoBody)
			w := httptest.NewRecorder()
			privateEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			_ = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, resBody["name"].(string), "client_1")
			assert.Equal(t, result.StatusCode, http.StatusOK)
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	})
}
