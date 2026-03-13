// Package main 主程序
package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/go-lib/rest"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-playground/assert"
	jsoniter "github.com/json-iterator/go"
	. "github.com/smartystreets/goconvey/convey"

	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/errors"
)

var (
	departNameInfo          []string = []string{"f_department_id", "f_name"}
	departNameLevelInfo     []string = []string{"f_department_id", "f_name", "f_third_party_id"}
	managerInfo             []string = []string{"f_department_id", "f_user_id", "f_display_name"}
	departInfo              []string = []string{"f_department_id", "f_name", "f_is_enterprise", "f_mail_address", "f_path", "f_manager_id", "f_code", "f_status", "f_third_party_id"}
	strGetDepartmentInfoSQL          = "select f_department_id, f_name, f_is_enterprise, f_mail_address, f_path, f_manager_id, f_code, f_status, f_third_party_id from sharemgnt_db.t_department"
)

func TestGetDepartManagers(t *testing.T) {
	newTestUserManagement(t)
	Convey("批量获取部门文档库组织管理员", t, func() {
		// 允许以任意顺序匹配期望
		mock.MatchExpectationsInOrder(false)

		Convey("部门拥有一个或多个组织管理员(一个)，成功", func() {
			mock.ExpectQuery(strGetDepartmentInfoSQL).
				WillReturnRows(sqlmock.NewRows(departInfo).AddRow("this-is-departmenet-id", "department_name", 1, "", "", "", "", 1, "third_id1"))
			mock.ExpectQuery("SELECT a.f_department_id, a.f_user_id, b.f_display_name").WillReturnRows(sqlmock.NewRows(managerInfo).AddRow("this-is-departmenet-id", "user_id_1", "display_name_1"))

			req := httptest.NewRequest("GET", "/api/user-management/v1/departments/this-is-departmenet-id/managers", http.NoBody)
			w := httptest.NewRecorder()
			privateEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			var resBody []interface{}
			_ = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, len(resBody), 1)
			assert.Equal(t, len(resBody[0].(map[string]interface{})["managers"].([]interface{})), 1)
			assert.Equal(t, resBody[0].(map[string]interface{})["department_id"].(string), "this-is-departmenet-id")
			assert.Equal(t, resBody[0].(map[string]interface{})["managers"].([]interface{})[0].(map[string]interface{})["id"], "user_id_1")
			assert.Equal(t, resBody[0].(map[string]interface{})["managers"].([]interface{})[0].(map[string]interface{})["name"], "display_name_1")
			assert.Equal(t, result.StatusCode, http.StatusOK)
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("部门拥有一个或多个组织管理员（多个），成功", func() {
			mock.ExpectQuery(strGetDepartmentInfoSQL).
				WillReturnRows(sqlmock.NewRows(departInfo).AddRow("this-is-departmenet-id", "department_name", 1, "", "", "", "", 1, "third_id1"))
			mock.ExpectQuery("SELECT a.f_department_id, a.f_user_id, b.f_display_name").
				WillReturnRows(sqlmock.NewRows(managerInfo).AddRow("this-is-departmenet-id", "user_id_1", "display_name_1").AddRow("this-is-departmenet-id", "user_id_2", "display_name_2"))

			req := httptest.NewRequest("GET", "/api/user-management/v1/departments/this-is-departmenet-id/managers", http.NoBody)
			w := httptest.NewRecorder()
			privateEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			var resBody []interface{}
			_ = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, len(resBody), 1)
			assert.Equal(t, len(resBody[0].(map[string]interface{})["managers"].([]interface{})), 2)
			assert.Equal(t, resBody[0].(map[string]interface{})["department_id"].(string), "this-is-departmenet-id")
			assert.Equal(t, resBody[0].(map[string]interface{})["managers"].([]interface{})[0].(map[string]interface{})["id"], "user_id_1")
			assert.Equal(t, resBody[0].(map[string]interface{})["managers"].([]interface{})[0].(map[string]interface{})["name"], "display_name_1")
			assert.Equal(t, resBody[0].(map[string]interface{})["managers"].([]interface{})[1].(map[string]interface{})["id"], "user_id_2")
			assert.Equal(t, resBody[0].(map[string]interface{})["managers"].([]interface{})[1].(map[string]interface{})["name"], "display_name_2")
			assert.Equal(t, result.StatusCode, http.StatusOK)
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("department_ids中包含一个或多个部门id，成功", func() {
			mock.ExpectQuery(strGetDepartmentInfoSQL).
				WillReturnRows(sqlmock.NewRows(departInfo).AddRow("this-is-departmenet-id", "department_name", 1, "", "", "", "", 1, "third_id1").
					AddRow("this-is-department-id1", "department_name1", 1, "", "", "", "", 1, "third_id2"))
			mock.ExpectQuery("SELECT a.f_department_id, a.f_user_id, b.f_display_name").
				WillReturnRows(sqlmock.NewRows(managerInfo).AddRow("this-is-departmenet-id", "user_id_1", "display_name_1").AddRow("this-is-department-id1", "user_id_2", "display_name_2"))

			req := httptest.NewRequest("GET", "/api/user-management/v1/departments/this-is-departmenet-id,this-is-department-id1/managers", http.NoBody)
			w := httptest.NewRecorder()
			privateEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			var resBody []interface{}
			_ = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, len(resBody), 2)
			assert.Equal(t, result.StatusCode, http.StatusOK)
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("department_ids中包含不存在的部门id，抛错", func() {
			mock.ExpectQuery(strGetDepartmentInfoSQL).
				WillReturnRows(sqlmock.NewRows(departInfo).AddRow("this-is-departmenet-id1", "department_name", 1, "", "", "", "", 1, "third_id1").
					AddRow("this-is-department-id1", "department_name1", 1, "", "", "", "", 1, "third_id2"))

			req := httptest.NewRequest("GET", "/api/user-management/v1/departments/this-is-departmenet-id,this-is-department-id2/managers", http.NoBody)
			w := httptest.NewRecorder()
			privateEngine.ServeHTTP(w, req)
			result := w.Result()

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

func TestGetDepartInfo(t *testing.T) {
	newTestUserManagement(t)
	Convey("批量获取部门信息", t, func() {
		// 允许以任意顺序匹配期望
		mock.MatchExpectationsInOrder(false)

		Convey("传入的部门被删除或传入多个department_ids其中包含被删除的部门，获取部门信息，抛错404", func() {
			mock.ExpectQuery(strGetDepartmentInfoSQL).
				WillReturnRows(sqlmock.NewRows(departInfo).AddRow("departID1", "name1", 1, "", "sub1/sub2/sub3/departID1", "", "", 1, "third_id1"))

			req := httptest.NewRequest("GET", "/api/user-management/v1/departments/departID1,departID2/name,parent_deps,managers", http.NoBody)
			w := httptest.NewRecorder()
			privateEngine.ServeHTTP(w, req)
			result := w.Result()

			assert.Equal(t, result.StatusCode, http.StatusNotFound)

			tempErr := rest.NewHTTPError("", 503000000, nil)
			testErr1 := rest.NewHTTPErrorV2(errors.NotFound, "department does not exist",
				rest.SetDetail(map[string]interface{}{"ids": []string{"departID2"}}))
			message, _ := io.ReadAll(result.Body)
			_ = jsoniter.Unmarshal(message, &tempErr)
			assert.Equal(t, tempErr.Code, testErr1.Code)
			assert.Equal(t, tempErr.Message, testErr1.Message)
			assert.Equal(t, tempErr.Cause, testErr1.Cause)
			temp := tempErr.Detail["ids"]
			data, ok := temp.([]interface{})
			assert.Equal(t, ok, true)
			assert.Equal(t, data[0].(string), "departID2")

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("传入的department_ids格式不正确，获取部门信息，抛错404", func() {
			mock.ExpectQuery(strGetDepartmentInfoSQL).
				WillReturnRows(sqlmock.NewRows(departInfo).AddRow("departID1", "name1", 1, "", "sub1/sub2/sub3/departID1", "", "", 1, "third_id1"))

			req := httptest.NewRequest("GET", "/api/user-management/v1/departments/departID1,departID2/name,parent_deps,managers", http.NoBody)
			w := httptest.NewRecorder()
			privateEngine.ServeHTTP(w, req)
			result := w.Result()

			assert.Equal(t, result.StatusCode, http.StatusNotFound)

			tempErr := rest.NewHTTPError("", 503000000, nil)
			testErr1 := rest.NewHTTPErrorV2(errors.NotFound, "department does not exist",
				rest.SetDetail(map[string]interface{}{"ids": []string{"departID2"}}))
			message, _ := io.ReadAll(result.Body)
			_ = jsoniter.Unmarshal(message, &tempErr)
			assert.Equal(t, tempErr.Code, testErr1.Code)
			assert.Equal(t, tempErr.Message, testErr1.Message)
			assert.Equal(t, tempErr.Cause, testErr1.Cause)

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("传入的fields包含其他规定之外的字符串，获取部门信息，抛错400", func() {
			req := httptest.NewRequest("GET", "/api/user-management/v1/departments/departID1,departID2/name,parent_deps,managers,xxxx", http.NoBody)
			w := httptest.NewRecorder()
			privateEngine.ServeHTTP(w, req)
			result := w.Result()

			assert.Equal(t, result.StatusCode, http.StatusBadRequest)

			tempErr := rest.NewHTTPError("", 503000000, nil)
			message, _ := io.ReadAll(result.Body)
			_ = jsoniter.Unmarshal(message, &tempErr)
			assert.Equal(t, tempErr.Code, rest.BadRequest)
			assert.Equal(t, tempErr.Cause, "invalid params")

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("传入多个正确但重复的department_ids，获取部门信息，抛错400", func() {
			req := httptest.NewRequest("GET", "/api/user-management/v1/departments/departID1,departID1,departID2/name,parent_deps,managers", http.NoBody)
			w := httptest.NewRecorder()
			privateEngine.ServeHTTP(w, req)
			result := w.Result()

			assert.Equal(t, result.StatusCode, http.StatusBadRequest)

			resBody := rest.NewHTTPError("", 503000000, nil)
			message, _ := io.ReadAll(result.Body)
			_ = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, resBody.Cause, "departmentID is not unique")
			assert.Equal(t, resBody.Code, rest.BadRequest)

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("批量获取部门信息，成功", func() {
			mock.ExpectQuery(strGetDepartmentInfoSQL).
				WillReturnRows(sqlmock.NewRows(departInfo).AddRow("departID1", "name1", 1, "", "sub1/sub2/sub3/departID1", "", "", 1, "third_id1").
					AddRow("departID2", "name2", 1, "", "sub3/sub4/departID2", "", "", 1, "third_id2"))
			mock.ExpectQuery("select f_department_id, f_name from sharemgnt_db.t_department where f_department_id in").
				WillReturnRows(sqlmock.NewRows(departNameInfo).AddRow("sub1", "name11").AddRow("sub2", "name22").AddRow("sub3", "name33").AddRow("sub4", "name44"))
			mock.ExpectQuery("SELECT a.f_department_id, a.f_user_id, b.f_display_name").WillReturnRows(sqlmock.NewRows(managerInfo).AddRow("departID1", "user_id_1", "display_name_1"))

			req := httptest.NewRequest("GET", "/api/user-management/v1/departments/departID1,departID2/name,parent_deps,managers", http.NoBody)
			w := httptest.NewRecorder()
			privateEngine.ServeHTTP(w, req)
			result := w.Result()

			assert.Equal(t, result.StatusCode, http.StatusOK)

			message, _ := io.ReadAll(result.Body)
			var resBody []interface{}
			_ = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, len(resBody), 2)

			temp1 := resBody[0].(map[string]interface{})
			assert.Equal(t, temp1["department_id"].(string), "departID1")
			assert.Equal(t, temp1["name"].(string), "name1")
			tempParentDeps1 := temp1["parent_deps"].([]interface{})
			assert.Equal(t, len(tempParentDeps1), 3)
			deps1 := tempParentDeps1[0].(map[string]interface{})
			assert.Equal(t, deps1["id"], "sub1")
			assert.Equal(t, deps1["name"], "name11")
			assert.Equal(t, deps1["type"], "department")
			deps2 := tempParentDeps1[1].(map[string]interface{})
			assert.Equal(t, deps2["id"], "sub2")
			assert.Equal(t, deps2["name"], "name22")
			assert.Equal(t, deps2["type"], "department")
			deps3 := tempParentDeps1[2].(map[string]interface{})
			assert.Equal(t, deps3["id"], "sub3")
			assert.Equal(t, deps3["name"], "name33")
			assert.Equal(t, deps3["type"], "department")
			tempManagers := temp1["managers"].([]interface{})
			assert.Equal(t, len(tempManagers), 1)
			manager1 := tempManagers[0].(map[string]interface{})
			assert.Equal(t, manager1["id"], "user_id_1")
			assert.Equal(t, manager1["name"], "display_name_1")

			temp2 := resBody[1].(map[string]interface{})
			assert.Equal(t, temp2["department_id"].(string), "departID2")
			assert.Equal(t, temp2["name"].(string), "name2")
			tempParentDeps2 := temp2["parent_deps"].([]interface{})
			assert.Equal(t, len(tempParentDeps2), 2)
			deps11 := tempParentDeps2[0].(map[string]interface{})
			assert.Equal(t, deps11["id"], "sub3")
			assert.Equal(t, deps11["name"], "name33")
			assert.Equal(t, deps11["type"], "department")
			deps22 := tempParentDeps2[1].(map[string]interface{})
			assert.Equal(t, deps22["id"], "sub4")
			assert.Equal(t, deps22["name"], "name44")
			assert.Equal(t, deps22["type"], "department")
			tempManagers1 := temp2["managers"].([]interface{})
			assert.Equal(t, len(tempManagers1), 0)

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	})
}

func TestGetDepartByLevel(t *testing.T) {
	newTestUserManagement(t)
	Convey("根据部门层级获取部门信息", t, func() {
		// 允许以任意顺序匹配期望
		mock.MatchExpectationsInOrder(false)

		Convey("传入的level为负数，获取部门列举，抛错400", func() {
			req := httptest.NewRequest("GET", "/api/user-management/v1/departments?level=-1", http.NoBody)
			w := httptest.NewRecorder()
			privateEngine.ServeHTTP(w, req)
			result := w.Result()

			assert.Equal(t, result.StatusCode, http.StatusBadRequest)

			resBody := rest.NewHTTPError("", 503000000, nil)
			message, _ := io.ReadAll(result.Body)
			_ = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, resBody.Cause, "level is illegal")
			assert.Equal(t, resBody.Code, rest.BadRequest)

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("成功", func() {
			mock.ExpectQuery("SELECT f_department_id").
				WillReturnRows(sqlmock.NewRows(departNameLevelInfo).AddRow("departID1", "name1", "third_id1"))

			req := httptest.NewRequest("GET", "/api/user-management/v1/departments?level=1", http.NoBody)
			w := httptest.NewRecorder()
			privateEngine.ServeHTTP(w, req)
			result := w.Result()

			assert.Equal(t, result.StatusCode, http.StatusOK)

			resBody := make([]interface{}, 0)
			message, _ := io.ReadAll(result.Body)
			_ = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, len(resBody), 1)

			temp1 := resBody[0].(map[string]interface{})
			assert.Equal(t, temp1["id"].(string), "departID1")
			assert.Equal(t, temp1["name"].(string), "name1")
			assert.Equal(t, temp1["type"].(string), "department")
			assert.Equal(t, temp1["third_id"].(string), "third_id1")
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	})
}
