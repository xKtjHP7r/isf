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
	"github.com/go-playground/assert"
	jsoniter "github.com/json-iterator/go"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

func TestGetMembersByID(t *testing.T) {
	newTestUserManagement(t)
	Convey("根据ID获取内部组成员", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		Convey("内部组ID不存在", func() {
			mock.ExpectQuery("select f_id from user_management.t_internal_group where f_id in").WillReturnRows(sqlmock.NewRows([]string{"f_id"}))

			req := httptest.NewRequest("GET", "/api/user-management/v1/internal-group-members/"+struuid, http.NoBody)
			w := httptest.NewRecorder()
			privateEngine.ServeHTTP(w, req)
			result := w.Result()

			assert.Equal(t, result.StatusCode, http.StatusNotFound)

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			err := jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, err, nil)
			assert.Equal(t, resBody["code"].(float64), float64(rest.URINotExist))
			assert.Equal(t, resBody["cause"].(string), "internal group do not exist")

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	})
}

func TestUpdateMembers(t *testing.T) {
	newTestUserManagement(t)
	Convey("更新内部组成员", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		Convey("内部组ID不存在", func() {
			data1 := map[string]interface{}{
				"id":   "client_1",
				"type": "user",
			}
			reqBody, _ := jsoniter.Marshal([]interface{}{data1})

			mock.ExpectQuery("select f_id from user_management.t_internal_group where f_id in").WillReturnRows(sqlmock.NewRows([]string{"f_id"}))

			req := httptest.NewRequest("PUT", "/api/user-management/v1/internal-group-members/"+struuid, bytes.NewReader(reqBody))
			w := httptest.NewRecorder()
			privateEngine.ServeHTTP(w, req)
			result := w.Result()

			assert.Equal(t, result.StatusCode, http.StatusNotFound)

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			err := jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, err, nil)
			assert.Equal(t, resBody["code"].(float64), float64(rest.URINotExist))
			assert.Equal(t, resBody["cause"].(string), "internal group do not exist")

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("组成员数组中存在重复成员", func() {
			data1 := map[string]interface{}{
				"id":   struuid,
				"type": "user",
			}
			data2 := map[string]interface{}{
				"id":   struuid,
				"type": "user",
			}
			reqBody, _ := jsoniter.Marshal([]interface{}{data1, data2})

			mock.ExpectQuery("select f_id from user_management.t_internal_group where f_id in").WillReturnRows(sqlmock.NewRows([]string{"f_id"}).AddRow(struuid))

			req := httptest.NewRequest("PUT", "/api/user-management/v1/internal-group-members/"+struuid, bytes.NewReader(reqBody))
			w := httptest.NewRecorder()
			privateEngine.ServeHTTP(w, req)
			result := w.Result()

			assert.Equal(t, result.StatusCode, http.StatusBadRequest)

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			err := jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, err, nil)
			assert.Equal(t, resBody["code"].(float64), float64(rest.BadRequest))
			assert.Equal(t, resBody["cause"].(string), "internal group member do not unique")

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("组成员数组中存在不存在、已删除成员", func() {
			data1 := map[string]interface{}{
				"id":   struuid,
				"type": "user",
			}
			reqBody, _ := jsoniter.Marshal([]interface{}{data1})

			mock.ExpectQuery("select f_id from user_management.t_internal_group where f_id in").WillReturnRows(sqlmock.NewRows([]string{"f_id"}).AddRow(struuid))
			mock.ExpectQuery("select f_user_id, f_display_name from sharemgnt_db.t_user where f_user_id in").WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_display_name"}))

			req := httptest.NewRequest("PUT", "/api/user-management/v1/internal-group-members/"+struuid, bytes.NewReader(reqBody))
			w := httptest.NewRecorder()
			privateEngine.ServeHTTP(w, req)
			result := w.Result()

			assert.Equal(t, result.StatusCode, http.StatusBadRequest)

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			err := jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, err, nil)
			assert.Equal(t, resBody["code"].(float64), float64(rest.BadRequest))
			assert.Equal(t, resBody["cause"].(string), "some members are not existing")
			details, ok := resBody["detail"].(map[string]interface{})
			assert.Equal(t, ok, true)
			assert.Equal(t, details["ids"], []interface{}{struuid})

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("组成员数组中存在用户类型不正确的成员", func() {
			data1 := map[string]interface{}{
				"id":   struuid,
				"type": "user1",
			}
			reqBody, _ := jsoniter.Marshal([]interface{}{data1})

			req := httptest.NewRequest("PUT", "/api/user-management/v1/internal-group-members/"+struuid, bytes.NewReader(reqBody))
			w := httptest.NewRecorder()
			privateEngine.ServeHTTP(w, req)
			result := w.Result()

			assert.Equal(t, result.StatusCode, http.StatusBadRequest)

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			err := jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, err, nil)
			assert.Equal(t, resBody["code"].(float64), float64(rest.BadRequest))
			assert.Equal(t, resBody["cause"].(string), "param member type is illegal")

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("组成员数组中存在userid为空", func() {
			data1 := map[string]interface{}{
				"id":   "",
				"type": "user",
			}
			reqBody, _ := jsoniter.Marshal([]interface{}{data1})

			req := httptest.NewRequest("PUT", "/api/user-management/v1/internal-group-members/"+struuid, bytes.NewReader(reqBody))
			w := httptest.NewRecorder()
			privateEngine.ServeHTTP(w, req)
			result := w.Result()

			assert.Equal(t, result.StatusCode, http.StatusBadRequest)

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			err := jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, err, nil)
			assert.Equal(t, resBody["code"].(float64), float64(rest.BadRequest))
			assert.Equal(t, resBody["cause"].(string), "param member id is illegal")

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	})
}
