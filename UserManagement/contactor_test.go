// Package main 主程序
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
)

//nolint:dupl
func TestDeleteContactor(t *testing.T) {
	newTestUserManagement(t)
	Convey("删除联系人组", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		hydraMock, _ := initHTTPMock(ctrl)
		Convey("token验证失败,报错401000000", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, passiveUserToken)

			errBytes := []byte{'a', 'z'}
			req := httptest.NewRequest("POST", "/api/eacp/v1/contactor/deletegroup", bytes.NewReader(errBytes))
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

		Convey("request body json格式错误报错400000000", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)

			reqBody, _ := jsoniter.Marshal(gin.H{"groupidxxxx": "380dbb6a-db17-11eb-a9f8-424a754a0131"})
			req := httptest.NewRequest("POST", "/api/eacp/v1/contactor/deletegroup", bytes.NewReader(reqBody))
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
			assert.Equal(t, resBody["cause"].(string), "body.groupid is required")

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("request body 不含有groupid 报错400000000", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)

			errBytes := []byte{'a', 'z'}
			req := httptest.NewRequest("POST", "/api/eacp/v1/contactor/deletegroup", bytes.NewReader(errBytes))
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

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("groupid格式错误，非uuid 报错400000000", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)

			reqBody1, _ := jsoniter.Marshal(gin.H{"groupid": "380dbb6asdadaddda-db17-11eb-a9f8-424a754a0131"})
			req := httptest.NewRequest("POST", "/api/eacp/v1/contactor/deletegroup", bytes.NewReader(reqBody1))
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
			assert.Equal(t, resBody["cause"].(string), "param groupid is illegal")

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("联系人组不属于当前用户 报错400000000", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)
			mock.ExpectQuery("select f_user_id from sharemgnt_db.t_person_group where f_group_id").WillReturnRows(sqlmock.NewRows([]string{"f_user_id"}))

			reqBody, _ := jsoniter.Marshal(gin.H{"groupid": "380dbb6a-db17-11eb-a9f8-424a754a0131"})
			req := httptest.NewRequest("POST", "/api/eacp/v1/contactor/deletegroup", bytes.NewReader(reqBody))
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
			assert.Equal(t, resBody["cause"].(string), "group is not exist")

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("删除联系人组成功", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)
			mock.ExpectQuery("select f_user_id from sharemgnt_db.t_person_group where f_group_id").WillReturnRows(sqlmock.NewRows([]string{"f_user_id"}).AddRow(activeUserID))
			mock.ExpectBegin()
			mock.ExpectExec("delete from sharemgnt_db.t_contact_person").WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec("delete from sharemgnt_db.t_person_group").WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec("insert into user_management.t_outbox").WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()

			reqBody, _ := jsoniter.Marshal(gin.H{"groupid": "380dbb6a-db17-11eb-a9f8-424a754a0131"})
			req := httptest.NewRequest("POST", "/api/eacp/v1/contactor/deletegroup", bytes.NewReader(reqBody))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			_ = jsoniter.Unmarshal(message, &resBody)
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
