// Package main 主程序
//
//nolint:dupl, funlen, lll
package main

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"
	"time"

	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/go-lib/rest"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/assert"
	jsoniter "github.com/json-iterator/go"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"

	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/driveradapters"
	uerrorrs "devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/errors"
	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/interfaces"
)

var (
	// 密码为1
	noWeekPwd string = "B4o+3gqw/vK/UUkb4kBTXdXHvYGhSRmi08/OEl6f5v3dsMKsIPFuLFPEuBu2/ZedheoNfN6HZqbFo3em/735ZeIMWjGbhV0EI1JTBMmNsUj8hSwwiwc0Rji3i7xhLy5dxP8k5LF/5rLy655mQMTEnr24UZMQkryMW1XXIDTGlh8="
	// 密码为123123
	noStrongPwd string = "YC41A+XKCUVnjYKVHC2R1kpiaRDRraH3k0RhzJqRKa/wQVqOnNUsAVruwOPyJF5rsU595Qf/78FsNDbg0pc45hrR0CHem7jD6AyHdV1iEv6HrKugYfwUTXar0JKS9be1YszJAiQannHrHhBp7knWFzz6mu7Ydne3xPZMi/1w6GI="
	// 密码为123123aA
	strongPwd string = "IAy6p8bFGJ7XnsXWmg1DG3+v0mHDstz8lAONFkX1jBppGdPEcpGsReVDBkVept5WT/rIGlvQ4NAgMekFNvcU4ed5ttts0cBV3mjplc69RiRwDG82HOJwbbAven+99abr4tSUAg2YsT80FAlMwH0nF5GSwYzPAAynH5wwr4KPzWk="

	userDBColumns []string = []string{"f_user_id", " f_login_name", "f_display_name", "f_priority", "f_csf_level", "f_status",
		"f_auto_disable_status", "f_mail_address", "f_auth_type", "f_freeze_status", "f_real_name_auth_status", "f_tel_number", "f_third_party_attr", "f_third_party_id",
		"f_pwd_control", "f_pwd_timestamp", "f_password", "f_sha2_password", "f_oss_id", "f_manager_id", "f_create_time", "f_csf_level2"}
	passwordConfig []string = []string{"f_key", "f_value"}
)

func getLocalTimeStr() string {
	return time.Unix(time.Now().Unix(), 0).Format("2006-01-02 15:04:05")
}

func TestGetNorlmalUserInfo(t *testing.T) {
	newTestUserManagement(t)
	Convey("获取用户个人资料", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		hydraMock, ossMock := initHTTPMock(ctrl)
		Convey("获取用户个人资料，普通用户调用成功", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)
			mock.ExpectQuery("select f_user_id, f_login_name, f_display_name, f_priority, f_csf_level,").
				WillReturnRows(sqlmock.NewRows(userDBColumns).AddRow(activeUserID, "loginName1", "displayName1", "1", "1", "0", "0", "0@qq.com", "1", "1", "1", "xxx", "zzz", "kkk", 0, getLocalTimeStr(), "pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13))
			mock.ExpectQuery("select f_oss_id, f_key, f_type, f_time from user_management.t_avatar").
				WillReturnRows(sqlmock.NewRows([]string{"f_oss_id", "f_key", "f_type", "f_time"}).AddRow("ossID", "key1", "xxx", 123))
			ossMock.EXPECT().HandleRequest("GET", "/api/ossgateway/v1/download/ossID/key1", nil).AnyTimes().Return(http.StatusOK, ossGatewayGetDownURL)

			req := httptest.NewRequest("GET", "/api/user-management/v1/profile/avatar_url", http.NoBody)
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			err := jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, err, nil)
			assert.Equal(t, result.StatusCode, http.StatusOK)
			assert.Equal(t, resBody["avatar_url"].(string), "url1")

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("获取用户个人资料，token校验失败", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, passiveUserToken)

			req := httptest.NewRequest("GET", "/api/user-management/v1/profile/avatar_url", http.NoBody)
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
			assert.Equal(t, resBody["cause"].(string), string("token expired"))

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("获取用户个人资料，管理员调用接口，报错400", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, adminUserToken)

			req := httptest.NewRequest("GET", "/api/user-management/v1/profile/avatar_url", http.NoBody)
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
			assert.Equal(t, resBody["cause"].(string), string("only support normal user"))

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("获取用户个人资料，应用账户调用接口，报错400", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, businessUserToken)

			req := httptest.NewRequest("GET", "/api/user-management/v1/profile/avatar_url", http.NoBody)
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
			assert.Equal(t, resBody["cause"].(string), string("only support normal user"))

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("获取用户个人资料，匿名账户调用接口，报错400", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, anonymousUserToken)

			req := httptest.NewRequest("GET", "/api/user-management/v1/profile/avatar_url", http.NoBody)
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
			assert.Equal(t, resBody["cause"].(string), string("only support normal user"))

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("获取用户个人资料,url内field为空,报错404", func() {
			req := httptest.NewRequest("GET", "/api/user-management/v1/profile", http.NoBody)
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			assert.Equal(t, result.StatusCode, http.StatusNotFound)

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("获取用户个人资料,field为url,报错400 参数错误", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)

			req := httptest.NewRequest("GET", "/api/user-management/v1/profile/url", http.NoBody)
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
			assert.Equal(t, resBody["cause"].(string), "invalid params")

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("获取用户个人资料,field为1,报错400 参数错误", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)

			req := httptest.NewRequest("GET", "/api/user-management/v1/profile/1", http.NoBody)
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
			assert.Equal(t, resBody["cause"].(string), "invalid params")

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("获取用户个人资料,field为avatar_url,test,报错400 参数错误", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)

			req := httptest.NewRequest("GET", "/api/user-management/v1/profile/avatar_url,test", http.NoBody)
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
			assert.Equal(t, resBody["cause"].(string), "invalid params")

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	})
}

func TestUpdateAvatar(t *testing.T) {
	newTestUserManagement(t)
	Convey("更新个人头像", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		hydraMock, ossMock := initHTTPMock(ctrl)
		Convey("更新个人头像,token验证失败", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, passiveUserToken)

			req := httptest.NewRequest("POST", "/api/user-management/v1/profile/avatar", http.NoBody)
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
			assert.Equal(t, resBody["cause"].(string), string("token expired"))

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		// 创建表单发送
		ct, rd, err := createMultiPartRequest(0)
		assert.Equal(t, err, nil)
		Convey("更新个人头像，管理员调用返回400", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, adminUserToken)

			req := httptest.NewRequest("POST", "/api/user-management/v1/profile/avatar", rd)
			req.Header.Add("Authorization", "Bearer test-token")
			req.Header.Add("Content-Type", ct)
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			err = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, err, nil)
			assert.Equal(t, result.StatusCode, http.StatusBadRequest)
			assert.Equal(t, resBody["code"].(float64), float64(rest.BadRequest))
			assert.Equal(t, resBody["cause"].(string), string("only support normal user"))

			if err = result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err = mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("更新个人头像，匿名用户调用返回400", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, anonymousUserToken)

			req := httptest.NewRequest("POST", "/api/user-management/v1/profile/avatar", rd)
			req.Header.Add("Authorization", "Bearer test-token")
			req.Header.Add("Content-Type", ct)
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			err = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, err, nil)
			assert.Equal(t, result.StatusCode, http.StatusBadRequest)
			assert.Equal(t, resBody["code"].(float64), float64(rest.BadRequest))
			assert.Equal(t, resBody["cause"].(string), string("only support normal user"))

			if err = result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err = mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("更新个人头像，应用账户调用返回400", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, businessUserToken)

			req := httptest.NewRequest("POST", "/api/user-management/v1/profile/avatar", rd)
			req.Header.Add("Authorization", "Bearer test-token")
			req.Header.Add("Content-Type", ct)
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			err = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, err, nil)
			assert.Equal(t, result.StatusCode, http.StatusBadRequest)
			assert.Equal(t, resBody["code"].(float64), float64(rest.BadRequest))
			assert.Equal(t, resBody["cause"].(string), string("only support normal user"))

			if err = result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err = mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		ct, rd, err = createMultiPartRequest(1)
		assert.Equal(t, err, nil)
		Convey("更新个人头像，file为49k, 返回400", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)

			req := httptest.NewRequest("POST", "/api/user-management/v1/profile/avatar", rd)
			req.Header.Add("Authorization", "Bearer test-token")
			req.Header.Add("Content-Type", ct)
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			err = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, err, nil)
			assert.Equal(t, result.StatusCode, http.StatusBadRequest)
			assert.Equal(t, resBody["code"].(float64), float64(rest.BadRequest))
			assert.Equal(t, resBody["cause"].(string), string("invalid params, file is too big"))

			if err = result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err = mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		ct, rd, err = createMultiPartRequest(2)
		assert.Equal(t, err, nil)
		Convey("更新个人头像，file为1M, 返回400", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)

			req := httptest.NewRequest("POST", "/api/user-management/v1/profile/avatar", rd)
			req.Header.Add("Authorization", "Bearer test-token")
			req.Header.Add("Content-Type", ct)
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			err = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, err, nil)
			assert.Equal(t, result.StatusCode, http.StatusBadRequest)
			assert.Equal(t, resBody["code"].(float64), float64(rest.BadRequest))
			assert.Equal(t, resBody["cause"].(string), string("invalid params, file is too big"))

			if err = result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err = mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		ct, rd, err = createMultiPartRequest(0)
		assert.Equal(t, err, nil)
		Convey("更新个人头像，无默认存储，报错500", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)
			ossMock.EXPECT().HandleRequest("GET", "/api/ossgateway/v1/local-storages", nil).AnyTimes().Return(http.StatusOK, []byte("[]"))

			req := httptest.NewRequest("POST", "/api/user-management/v1/profile/avatar", rd)
			req.Header.Add("Authorization", "Bearer test-token")
			req.Header.Add("Content-Type", ct)
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			err = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, err, nil)
			assert.Equal(t, result.StatusCode, http.StatusInternalServerError)
			assert.Equal(t, resBody["code"].(float64), float64(rest.InternalServerError))
			assert.Equal(t, resBody["cause"].(string), string("no available oss"))

			if err = result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err = mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	})
}

// createMultiPartRequest 创建表单数据
func createMultiPartRequest(sizeType int) (ct string, w io.Reader, err error) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	// 在表单中创建一个文件字段
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="avatar"; filename="xx.png"`)
	h.Set("Content-Type", "image/png")

	formFile, err := writer.CreatePart(h)
	if err != nil {
		return "", nil, err
	}

	// 文件拷贝
	if sizeType == 0 {
		buffer := make([]byte, 100)
		_, err = io.Copy(formFile, bytes.NewReader(buffer))
		if err != nil {
			return "", nil, err
		}
	} else if sizeType == 1 {
		buffer := make([]byte, 49*1024)
		_, err = io.Copy(formFile, bytes.NewReader(buffer))
		if err != nil {
			return "", nil, err
		}
	} else if sizeType == 2 {
		buffer := make([]byte, 1024*1024)
		_, err = io.Copy(formFile, bytes.NewReader(buffer))
		if err != nil {
			return "", nil, err
		}
	}

	// 关闭
	if err := writer.Close(); err != nil {
		return "", nil, err
	}

	return writer.FormDataContentType(), body, nil
}

func TestGetUserBaseInfo(t *testing.T) {
	newTestUserManagement(t)
	Convey("获取用户基本信息内部接口", t, func() {
		Convey("获取用户基本信息,批量获取，请求内包含非规定枚举字段emailssss， 报错", func() {
			req := httptest.NewRequest("GET", "/api/user-management/v1/users/userID1,userID2/account,frozen,authenticated,roles,emailssss", http.NoBody)
			w := httptest.NewRecorder()
			privateEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			err := jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, err, nil)
			assert.Equal(t, result.StatusCode, http.StatusBadRequest)
			assert.Equal(t, resBody["code"].(float64), float64(rest.BadRequest))
			assert.Equal(t, resBody["cause"].(string), string("invalid type"))

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("获取用户基本信息,批量获取，用户不存在， 报错", func() {
			mock.ExpectQuery("select f_user_id, f_login_name, f_display_name, f_priority, f_csf_level,").
				WillReturnRows(sqlmock.NewRows(userDBColumns).AddRow("userID1", "loginName1", "displayName1", "1", "1", "0", "0", "0@qq.com",
					"1", "1", "1", "xxx", "zzz", "kkk", 0, getLocalTimeStr(), "pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13))

			req := httptest.NewRequest("GET", "/api/user-management/v1/users/userID1,userID2/account,frozen,authenticated,roles", http.NoBody)
			w := httptest.NewRecorder()
			privateEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			err := jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, err, nil)
			assert.Equal(t, result.StatusCode, http.StatusNotFound)
			assert.Equal(t, resBody["code"].(float64), float64(uerrorrs.NotFound))
			assert.Equal(t, resBody["cause"].(string), string("those users are not existing"))

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("获取用户基本信息,批量获取，获取用户账户名，冻结状态，实名认证信息等全部信息成功，返回用户信息数组，包含用户ID， 成功", func() {
			mock.ExpectQuery("select f_user_id, f_login_name, f_display_name, f_priority, f_csf_level,").
				WillReturnRows(sqlmock.NewRows(userDBColumns).AddRow("userID1", "loginName1", "displayName1", "1", "1", "0", "0", "0@qq.com", "1", "1",
					"1", "xxx", "zzz", "kkk", 0, getLocalTimeStr(), "pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13).
					AddRow("userID2", "loginName2", "displayName2", "2", "2", "1", "0", "2@qq.com", "1", "0", "0", "xxx1", "zzz1",
						"kkk", 0, getLocalTimeStr(), "pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13))
			mock.ExpectQuery("select f_user_id, f_role_id from sharemgnt_db.t_user_role_relation where f_user_id in").
				WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}).AddRow("userID1", interfaces.SystemRoleAuditAdmin))

			req := httptest.NewRequest("GET", "/api/user-management/v1/users/userID1,userID2/enabled,priority,csf_level,name,account,frozen,authenticated,roles,email,telephone,third_attr", http.NoBody)
			w := httptest.NewRecorder()
			privateEngine.ServeHTTP(w, req)
			result := w.Result()

			assert.Equal(t, result.StatusCode, http.StatusOK)

			message, _ := io.ReadAll(result.Body)
			sliceBody := make([]map[string]interface{}, 2)
			err1 := jsoniter.Unmarshal(message, &sliceBody)
			assert.Equal(t, err1, nil)

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			// 结果检查
			resBody := make(map[string]map[string]interface{})
			for _, v := range sliceBody {
				resBody[v["id"].(string)] = v
			}

			userInfo1 := resBody["userID1"]
			userInfo2 := resBody["userID2"]
			assert.Equal(t, userInfo1["id"].(string), "userID1")
			assert.Equal(t, userInfo1["name"].(string), "displayName1")
			assert.Equal(t, userInfo1["account"].(string), "loginName1")
			assert.Equal(t, userInfo1["frozen"].(bool), true)
			assert.Equal(t, userInfo1["authenticated"].(bool), true)
			assert.Equal(t, userInfo1["enabled"].(bool), true)
			assert.Equal(t, int(userInfo1["priority"].(float64)), 1)
			assert.Equal(t, int(userInfo1["csf_level"].(float64)), 1)
			assert.Equal(t, userInfo1["telephone"].(string), "xxx")
			assert.Equal(t, userInfo1["third_attr"].(string), "zzz")
			assert.Equal(t, userInfo1["email"].(string), "0@qq.com")

			assert.Equal(t, userInfo2["id"].(string), "userID2")
			assert.Equal(t, userInfo2["account"].(string), "loginName2")
			assert.Equal(t, userInfo2["frozen"].(bool), false)
			assert.Equal(t, userInfo2["authenticated"].(bool), false)
			assert.Equal(t, userInfo2["enabled"].(bool), false)
			assert.Equal(t, int(userInfo2["priority"].(float64)), 2)
			assert.Equal(t, int(userInfo2["csf_level"].(float64)), 2)
			assert.Equal(t, userInfo2["name"].(string), "displayName2")
			assert.Equal(t, userInfo2["telephone"].(string), "xxx1")
			assert.Equal(t, userInfo2["third_attr"].(string), "zzz1")
			assert.Equal(t, userInfo2["email"].(string), "2@qq.com")

			tempRoleInfo1 := make(map[string]bool)
			for _, v := range userInfo1["roles"].([]interface{}) {
				tempRoleInfo1[v.(string)] = true
			}
			assert.Equal(t, len(tempRoleInfo1), 2)
			assert.Equal(t, tempRoleInfo1[driveradapters.EnumNormaluser], true)
			assert.Equal(t, tempRoleInfo1[driveradapters.EnumAuditAdmin], true)

			tempRoleInfo2 := make(map[string]bool)
			for _, v := range userInfo2["roles"].([]interface{}) {
				tempRoleInfo2[v.(string)] = true
			}
			assert.Equal(t, len(tempRoleInfo2), 1)
			assert.Equal(t, tempRoleInfo2[driveradapters.EnumNormaluser], true)

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("获取用户基本信息,批量获取，获取用户状态和用户优先级部分信息成功，返回用户信息数组，包含用户ID， 成功", func() {
			mock.ExpectQuery("select f_user_id, f_login_name, f_display_name, f_priority, f_csf_level,").
				WillReturnRows(sqlmock.NewRows(userDBColumns).AddRow("userID1", "loginName1", "displayName1", "1", "1", "0",
					"0", "0@qq.com", "1", "1", "1", "xxx", "zzz", "kkk", 0, getLocalTimeStr(), "pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13).
					AddRow("userID2", "loginName2", "displayName2", "2", "2", "1", "0", "2@qq.com", "1", "0", "0", "xxx1", "zzz1", "kkk",
						0, getLocalTimeStr(), "pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13))

			req := httptest.NewRequest("GET", "/api/user-management/v1/users/userID1,userID2/enabled,priority", http.NoBody)
			w := httptest.NewRecorder()
			privateEngine.ServeHTTP(w, req)
			result := w.Result()

			assert.Equal(t, result.StatusCode, http.StatusOK)

			message, _ := io.ReadAll(result.Body)
			sliceBody := make([]map[string]interface{}, 2)
			err1 := jsoniter.Unmarshal(message, &sliceBody)
			assert.Equal(t, err1, nil)

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			// 结果检查
			resBody := make(map[string]map[string]interface{})
			for _, v := range sliceBody {
				resBody[v["id"].(string)] = v
			}

			userInfo1 := resBody["userID1"]
			userInfo2 := resBody["userID2"]
			assert.Equal(t, userInfo1["id"].(string), "userID1")
			assert.Equal(t, userInfo1["enabled"].(bool), true)
			assert.Equal(t, int(userInfo1["priority"].(float64)), 1)
			assert.Equal(t, userInfo2["enabled"].(bool), false)
			assert.Equal(t, int(userInfo2["priority"].(float64)), 2)

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("获取用户基本信息，批量获取，获取用户状态单个信息，返回用户信息数组，包含用户ID， 成功", func() {
			mock.ExpectQuery("select f_user_id, f_login_name, f_display_name, f_priority, f_csf_level,").
				WillReturnRows(sqlmock.NewRows(userDBColumns).AddRow("userID1", "loginName1", "displayName1", "1", "1", "0", "0", "0@qq.com", "1", "1",
					"1", "xxx", "zzz", "kkk", 0, getLocalTimeStr(), "pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13).
					AddRow("userID2", "loginName2", "displayName2", "2", "2", "1", "0", "2@qq.com", "1", "0", "0", "xxx1", "zzz1", "kkk", 0,
						getLocalTimeStr(), "pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13))

			req := httptest.NewRequest("GET", "/api/user-management/v1/users/userID1,userID2/enabled", http.NoBody)
			w := httptest.NewRecorder()
			privateEngine.ServeHTTP(w, req)
			result := w.Result()

			assert.Equal(t, result.StatusCode, http.StatusOK)

			message, _ := io.ReadAll(result.Body)
			sliceBody := make([]map[string]interface{}, 2)
			err1 := jsoniter.Unmarshal(message, &sliceBody)
			assert.Equal(t, err1, nil)

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			// 结果检查
			resBody := make(map[string]map[string]interface{})
			for _, v := range sliceBody {
				resBody[v["id"].(string)] = v
			}

			userInfo1 := resBody["userID1"]
			userInfo2 := resBody["userID2"]
			assert.Equal(t, userInfo1["id"].(string), "userID1")
			assert.Equal(t, userInfo1["enabled"].(bool), true)
			assert.Equal(t, userInfo2["enabled"].(bool), false)

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("获取用户基本信息,单个用户，获取用户账户名，冻结状态，实名认证信息，父部门信息成功，并且不包含ID", func() {
			mock.ExpectQuery("select f_user_id, f_login_name, f_display_name, f_priority, f_csf_level,").
				WillReturnRows(sqlmock.NewRows(userDBColumns).AddRow("userID1", "loginName1", "displayName1", "1", "1", "0", "0", "0@qq.com",
					"1", "1", "1", "xxx", "zzz", "kkk", 0, getLocalTimeStr(), "pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13))
			mock.ExpectQuery("select f_user_id, f_role_id from sharemgnt_db.t_user_role_relation where f_user_id").
				WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}).AddRow("userID1", interfaces.SystemRoleAuditAdmin))
			mock.ExpectQuery("select f_department_id, f_path from sharemgnt_db.t_user_department_relation where f_user_id").
				WillReturnRows(sqlmock.NewRows([]string{"f_department_id", "f_path"}).AddRow("departmentID1", "departmentID1/xxx"))
			mock.ExpectQuery("select f_department_id, f_name, f_is_enterprise, f_mail_address, f_path, f_manager_id, f_code, f_status, f_third_party_id from sharemgnt_db.t_department where f_department_id in").
				WillReturnRows(sqlmock.NewRows([]string{"f_department_id", "f_name", "f_is_enterprise", "f_mail_address", "f_path", "f_manager_id", "f_code", "f_status", "f_third_party_id"}).
					AddRow("departmentID1", "name1", "1", "d@qq.com", "xxxx", "", "", 1, "third_id1"))
			req := httptest.NewRequest("GET", "/api/user-management/v1/users/userID1/account,frozen,authenticated,roles,parent_deps", http.NoBody)
			w := httptest.NewRecorder()
			privateEngine.ServeHTTP(w, req)
			result := w.Result()

			assert.Equal(t, result.StatusCode, http.StatusOK)

			message, _ := io.ReadAll(result.Body)
			resBody := []map[string]interface{}{}
			err1 := jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, err1, nil)

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			// 结果检查
			_, ok := resBody[0]["id"]
			_, ok1 := resBody[0]["parent_deps"]
			assert.Equal(t, ok, true)
			assert.Equal(t, ok1, true)

			assert.Equal(t, resBody[0]["account"].(string), "loginName1")
			assert.Equal(t, resBody[0]["frozen"].(bool), true)
			assert.Equal(t, resBody[0]["authenticated"].(bool), true)

			tempRoleInfo1 := make(map[string]bool)
			for _, v := range resBody[0]["roles"].([]interface{}) {
				tempRoleInfo1[v.(string)] = true
			}
			assert.Equal(t, len(tempRoleInfo1), 2)
			assert.Equal(t, tempRoleInfo1[driveradapters.EnumNormaluser], true)
			assert.Equal(t, tempRoleInfo1[driveradapters.EnumAuditAdmin], true)

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	})
}

//nolint:lll
func TestModifyUserInfoPassowrdByRealName(t *testing.T) {
	newTestUserManagement(t)

	Convey("实名用户修改用户信息---修改密码", t, func() {
		reqBody, _ := jsoniter.Marshal(gin.H{"password": noStrongPwd})

		// 允许以任意顺序匹配期望
		mock.MatchExpectationsInOrder(false)

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		hydraMock, _ := initHTTPMock(ctrl)
		Convey("不具备超级管理员和安全管理员角色的实名用户修改用户密码失败", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)

			mock.ExpectQuery("select f_user_id, f_role_id from sharemgnt_db.t_user_role_relation where f_user_id").
				WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}).
					AddRow(activeUserID, interfaces.SystemRoleSysAdmin).
					AddRow(activeUserID, interfaces.SystemRoleAuditAdmin).
					AddRow(activeUserID, interfaces.SystemRoleOrgAudit).
					AddRow(activeUserID, interfaces.SystemRoleOrgManager))

			req := httptest.NewRequest("PUT", "/api/user-management/v1/management/users/xxx-xxx-xxx/password", bytes.NewReader(reqBody))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			_ = jsoniter.Unmarshal(message, &resBody)
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

		Convey("具有超级管理员角色的实名用户修改用户密码成功", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)
			mock.ExpectQuery("select f_user_id, f_role_id from sharemgnt_db.t_user_role_relation where f_user_id").
				WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}).AddRow(activeUserID, interfaces.SystemRoleSuperAdmin))
			mock.ExpectQuery("select `f_key`, `f_value` from `sharemgnt_db`.`t_sharemgnt_config` where `f_key` in ").
				WillReturnRows(sqlmock.NewRows(passwordConfig).AddRow("pwd_expire_time", "-1").AddRow("strong_pwd_status", "0").AddRow("strong_pwd_length", "10").
					AddRow("enable_pwd_lock", "0").AddRow("pwd_err_cnt", "0").AddRow("pwd_lock_time", "-1").AddRow("enable_des_password", "0"))
			mock.ExpectQuery("select f_user_id, f_login_name, f_display_name, f_priority, f_csf_level,").
				WillReturnRows(sqlmock.NewRows(userDBColumns).AddRow("userID1", "loginName1", "displayName1", "1", "1", "0", "0", "0@qq.com",
					"1", "1", "1", "xxx", "zzz", "kkk", 0, getLocalTimeStr(), "pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13))

			mock.ExpectBegin()
			mock.ExpectExec("update sharemgnt_db.t_user set").WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec("into user_management.t_outbox").WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()

			req := httptest.NewRequest("PUT", "/api/user-management/v1/management/users/xxx-xxx-xxx/password", bytes.NewReader(reqBody))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			_ = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, result.StatusCode, http.StatusNoContent)
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}

			time.Sleep(time.Microsecond)
		})
	})
}

func TestModifyUserInfoPassowrdByBuisness(t *testing.T) {
	newTestUserManagement(t)
	Convey("应用账户修改用户信息---修改密码", t, func() {
		// 允许以任意顺序匹配期望
		mock.MatchExpectationsInOrder(true)
		reqBody, _ := jsoniter.Marshal(gin.H{"password": strongPwd})

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		hydraMock, _ := initHTTPMock(ctrl)
		Convey("不具有权限的应用账户修改普通用户密码失败", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, businessUserToken)

			mock.ExpectQuery("select f_app_name, f_org_type, f_perm_value, f_end_time from user_management.t_org_perm_app where").
				WillReturnRows(sqlmock.NewRows([]string{"f_app_name", "f_org_type", "f_perm_value", "f_end_time"}).AddRow("xxx", 1, 2, -1))

			req := httptest.NewRequest("PUT", "/api/user-management/v1/management/users/xxx-xxx-xxx/password", bytes.NewReader(reqBody))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			_ = jsoniter.Unmarshal(message, &resBody)
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

		Convey("应用账户权限过期,修改失败", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, businessUserToken)

			mock.ExpectQuery("select f_app_name, f_org_type, f_perm_value, f_end_time from user_management.t_org_perm_app where").
				WillReturnRows(sqlmock.NewRows([]string{"f_app_name", "f_org_type", "f_perm_value", "f_end_time"}))

			req := httptest.NewRequest("PUT", "/api/user-management/v1/management/users/xxx-xxx-xxx/password", bytes.NewReader(reqBody))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			_ = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, result.StatusCode, http.StatusForbidden)
			assert.Equal(t, resBody["code"].(float64), float64(rest.Forbidden))
			assert.Equal(t, resBody["cause"].(string), "this user do not has the authority")
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
				assert.Equal(t, 1, 1)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("具有权限的应用账户修改普通用户密码成功", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, businessUserToken)
			mock.ExpectQuery("select f_app_name, f_org_type, f_perm_value, f_end_time from user_management.t_org_perm_app where").
				WillReturnRows(sqlmock.NewRows([]string{"f_app_name", "f_org_type", "f_perm_value", "f_end_time"}).AddRow("xxx", 1, 3, -1))
			mock.ExpectQuery("select `f_key`, `f_value` from `sharemgnt_db`.`t_sharemgnt_config` where `f_key` in").
				WillReturnRows(sqlmock.NewRows(passwordConfig).AddRow("pwd_expire_time", "-1").AddRow("strong_pwd_status", "0").AddRow("strong_pwd_length", "10").
					AddRow("enable_pwd_lock", "0").AddRow("pwd_err_cnt", "0").AddRow("pwd_lock_time", "-1").AddRow("enable_des_password", "0"))
			mock.ExpectQuery("select f_user_id, f_login_name, f_display_name, f_priority, f_csf_level,").
				WillReturnRows(sqlmock.NewRows(userDBColumns).AddRow("userID1", "loginName1", "displayName1", "1", "1", "0", "0", "0@qq.com",
					"1", "1", "1", "xxx", "zzz", "kkk", 0, getLocalTimeStr(), "pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13))

			mock.ExpectBegin()
			mock.ExpectExec("update sharemgnt_db.t_user set f_password =").WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec("into user_management.t_outbox").WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()

			req := httptest.NewRequest("PUT", "/api/user-management/v1/management/users/xxx-xxx-xxx/password", bytes.NewReader(reqBody))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			_ = jsoniter.Unmarshal(message, &resBody)
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

func TestModifyUserInfoPassowrdCommon(t *testing.T) {
	newTestUserManagement(t)
	Convey("修改用户信息---修改密码", t, func() {
		reqBody, _ := jsoniter.Marshal(gin.H{"password": noWeekPwd})

		// 允许以任意顺序匹配期望
		mock.MatchExpectationsInOrder(false)

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		hydraMock, _ := initHTTPMock(ctrl)
		Convey("未开启强密码策略,密码不符合弱密码配置,修改失败", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)
			mock.ExpectQuery("select f_user_id, f_role_id from sharemgnt_db.t_user_role_relation where f_user_id").
				WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}).AddRow(activeUserID, interfaces.SystemRoleSuperAdmin))
			mock.ExpectQuery("select `f_key`, `f_value` from `sharemgnt_db`.`t_sharemgnt_config` where `f_key` in").
				WillReturnRows(sqlmock.NewRows(passwordConfig).AddRow("pwd_expire_time", "-1").AddRow("strong_pwd_status", "0").AddRow("strong_pwd_length", "10").
					AddRow("enable_pwd_lock", "0").AddRow("pwd_err_cnt", "0").AddRow("pwd_lock_time", "-1").AddRow("enable_des_password", "0"))
			mock.ExpectQuery("select f_user_id, f_login_name, f_display_name, f_priority, f_csf_level,").
				WillReturnRows(sqlmock.NewRows(userDBColumns).AddRow("userID1", "loginName1", "displayName1", "1", "1", "0", "0", "0@qq.com",
					"1", "1", "1", "xxx", "zzz", "kkk", 0, getLocalTimeStr(), "pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13))

			req := httptest.NewRequest("PUT", "/api/user-management/v1/management/users/xxx-xxx-xxx/password", bytes.NewReader(reqBody))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			_ = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, result.StatusCode, http.StatusBadRequest)
			assert.Equal(t, resBody["code"].(float64), float64(rest.BadRequest))
			assert.Equal(t, resBody["cause"].(string), "invalid password")
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		reqBody, _ = jsoniter.Marshal(gin.H{"password": noStrongPwd})
		Convey("开启强密码策略,密码不符合强密码配置,修改失败", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)

			mock.ExpectQuery("select f_user_id, f_role_id from sharemgnt_db.t_user_role_relation where f_user_id").
				WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}).AddRow(activeUserID, interfaces.SystemRoleSuperAdmin))
			mock.ExpectQuery("select `f_key`, `f_value` from `sharemgnt_db`.`t_sharemgnt_config` where `f_key` in").
				WillReturnRows(sqlmock.NewRows(passwordConfig).AddRow("pwd_expire_time", "-1").AddRow("strong_pwd_status", "1").AddRow("strong_pwd_length", "10").
					AddRow("enable_pwd_lock", "0").AddRow("pwd_err_cnt", "0").AddRow("pwd_lock_time", "-1").AddRow("enable_des_password", "0"))
			mock.ExpectQuery("select f_user_id, f_login_name, f_display_name, f_priority, f_csf_level,").
				WillReturnRows(sqlmock.NewRows(userDBColumns).AddRow("userID1", "loginName1", "displayName1", "1", "1", "0", "0", "0@qq.com",
					"1", "1", "1", "xxx", "zzz", "kkk", 0, getLocalTimeStr(), "pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13))

			req := httptest.NewRequest("PUT", "/api/user-management/v1/management/users/xxx-xxx-xxx/password", bytes.NewReader(reqBody))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			_ = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, result.StatusCode, http.StatusBadRequest)
			assert.Equal(t, resBody["code"].(float64), float64(rest.BadRequest))
			assert.Equal(t, resBody["cause"].(string), "invalid password")
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("用户不存在,修改密码失败", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)

			mock.ExpectQuery("select f_user_id, f_role_id from sharemgnt_db.t_user_role_relation where f_user_id").
				WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}).AddRow(activeUserID, interfaces.SystemRoleSuperAdmin))
			mock.ExpectQuery("select `f_key`, `f_value` from `sharemgnt_db`.`t_sharemgnt_config` where `f_key` in").
				WillReturnRows(sqlmock.NewRows(passwordConfig).AddRow("pwd_expire_time", "-1").AddRow("strong_pwd_status", "0").AddRow("strong_pwd_length", "10").
					AddRow("enable_pwd_lock", "0").AddRow("pwd_err_cnt", "0").AddRow("pwd_lock_time", "-1").AddRow("enable_des_password", "0"))
			mock.ExpectQuery("select f_user_id, f_login_name, f_display_name, f_priority, f_csf_level,").
				WillReturnRows(sqlmock.NewRows(userDBColumns))

			req := httptest.NewRequest("PUT", "/api/user-management/v1/management/users/xxx-xxx-xxx/password", bytes.NewReader(reqBody))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			_ = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, result.StatusCode, http.StatusNotFound)
			assert.Equal(t, resBody["code"].(float64), float64(uerrorrs.NotFound))
			assert.Equal(t, resBody["cause"].(string), "user does not exist")
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("修改对象为域用户,修改失败", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)

			mock.ExpectQuery("select f_user_id, f_role_id from sharemgnt_db.t_user_role_relation where f_user_id").
				WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}).AddRow(activeUserID, interfaces.SystemRoleSuperAdmin))
			mock.ExpectQuery("select `f_key`, `f_value` from `sharemgnt_db`.`t_sharemgnt_config` where `f_key` in").
				WillReturnRows(sqlmock.NewRows(passwordConfig).AddRow("pwd_expire_time", "-1").AddRow("strong_pwd_status", "0").AddRow("strong_pwd_length", "10").
					AddRow("enable_pwd_lock", "0").AddRow("pwd_err_cnt", "0").AddRow("pwd_lock_time", "-1").AddRow("enable_des_password", "0"))
			mock.ExpectQuery("select f_user_id, f_login_name, f_display_name, f_priority, f_csf_level,").
				WillReturnRows(sqlmock.NewRows(userDBColumns).AddRow("userID1", "loginName1", "displayName1", "1", "1", "0", "0", "0@qq.com",
					"2", "1", "1", "xxx", "zzz", "kkk", 0, getLocalTimeStr(), "pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13))

			req := httptest.NewRequest("PUT", "/api/user-management/v1/management/users/xxx-xxx-xxx/password", bytes.NewReader(reqBody))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			_ = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, result.StatusCode, http.StatusBadRequest)
			assert.Equal(t, resBody["code"].(float64), float64(rest.BadRequest))
			assert.Equal(t, resBody["cause"].(string), "can not modify non local user password")
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("修改对象为第三方用户,修改失败", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)

			mock.ExpectQuery("select f_user_id, f_role_id from sharemgnt_db.t_user_role_relation where f_user_id").
				WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}).AddRow(activeUserID, interfaces.SystemRoleSuperAdmin))
			mock.ExpectQuery("select `f_key`, `f_value` from `sharemgnt_db`.`t_sharemgnt_config` where `f_key` in").
				WillReturnRows(sqlmock.NewRows(passwordConfig).AddRow("pwd_expire_time", "-1").AddRow("strong_pwd_status", "0").AddRow("strong_pwd_length", "10").
					AddRow("enable_pwd_lock", "0").AddRow("pwd_err_cnt", "0").AddRow("pwd_lock_time", "-1").AddRow("enable_des_password", "0"))
			mock.ExpectQuery("select f_user_id, f_login_name, f_display_name, f_priority, f_csf_level,").
				WillReturnRows(sqlmock.NewRows(userDBColumns).AddRow("userID1", "loginName1", "displayName1", "1", "1", "0", "0", "0@qq.com",
					"3", "1", "1", "xxx", "zzz", "kkk", 0, getLocalTimeStr(), "pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13))

			req := httptest.NewRequest("PUT", "/api/user-management/v1/management/users/xxx-xxx-xxx/password", bytes.NewReader(reqBody))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			_ = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, result.StatusCode, http.StatusBadRequest)
			assert.Equal(t, resBody["code"].(float64), float64(rest.BadRequest))
			assert.Equal(t, resBody["cause"].(string), "can not modify non local user password")
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("token失效,修改密码失败", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, passiveUserToken)
			req := httptest.NewRequest("PUT", "/api/user-management/v1/management/users/xxx-xxx-xxx/password", bytes.NewReader(reqBody))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			_ = jsoniter.Unmarshal(message, &resBody)
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

		Convey("必要参数不传,修改密码失败", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)

			reqBody1, _ := jsoniter.Marshal(gin.H{})
			req := httptest.NewRequest("PUT", "/api/user-management/v1/management/users/xxx-xxx-xxx/password", bytes.NewReader(reqBody1))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			_ = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, result.StatusCode, http.StatusBadRequest)
			assert.Equal(t, resBody["code"].(float64), float64(rest.BadRequest))
			assert.Equal(t, resBody["cause"].(string), "body.password is required")
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("必要参数传空,修改密码失败", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)

			mock.ExpectQuery("select f_user_id, f_role_id from sharemgnt_db.t_user_role_relation where f_user_id").
				WillReturnRows(sqlmock.NewRows([]string{"f_user_id", "f_role_id"}).AddRow(activeUserID, interfaces.SystemRoleSecAdmin))
			mock.ExpectQuery("select `f_key`, `f_value` from `sharemgnt_db`.`t_sharemgnt_config` where `f_key` in").
				WillReturnRows(sqlmock.NewRows(passwordConfig).AddRow("pwd_expire_time", "-1").AddRow("strong_pwd_status", "0").AddRow("strong_pwd_length", "10").
					AddRow("enable_pwd_lock", "0").AddRow("pwd_err_cnt", "0").AddRow("pwd_lock_time", "-1").AddRow("enable_des_password", "0"))
			mock.ExpectQuery("select f_user_id, f_login_name, f_display_name, f_priority, f_csf_level,").
				WillReturnRows(sqlmock.NewRows(userDBColumns).AddRow("userID1", "loginName1", "displayName1", "1", "1", "0", "0", "0@qq.com",
					"1", "1", "1", "xxx", "zzz", "kkk", 0, getLocalTimeStr(), "pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13))

			reqBody1, _ := jsoniter.Marshal(gin.H{"password": ""})
			req := httptest.NewRequest("PUT", "/api/user-management/v1/management/users/xxx-xxx-xxx/password", bytes.NewReader(reqBody1))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			_ = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, result.StatusCode, http.StatusBadRequest)
			assert.Equal(t, resBody["code"].(float64), float64(rest.BadRequest))
			assert.Equal(t, resBody["cause"].(string), "crypto/rsa: decryption error")
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("fields中包含无效字段,修改密码失败", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)

			reqBody, _ := jsoniter.Marshal(gin.H{"password": "xxxx"})
			req := httptest.NewRequest("PUT", "/api/user-management/v1/management/users/xxx-xxx-xxx/password,xxxx", bytes.NewReader(reqBody))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			_ = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, result.StatusCode, http.StatusBadRequest)
			assert.Equal(t, resBody["code"].(float64), float64(rest.BadRequest))
			assert.Equal(t, resBody["cause"].(string), "invalid params")
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	})
}

func TestGetAvatarByIDs(t *testing.T) {
	newTestUserManagement(t)
	Convey("根据ID批量获取用户头像", t, func() {
		reqBody, _ := jsoniter.Marshal(gin.H{"password": noWeekPwd})

		// 允许以任意顺序匹配期望
		mock.MatchExpectationsInOrder(false)

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		hydraMock, ossMock := initHTTPMock(ctrl)
		Convey("token过期，则报错401", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, passiveUserToken)

			req := httptest.NewRequest("GET", "/api/user-management/v1/avatars/xxx", bytes.NewReader(reqBody))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make(map[string]interface{})
			_ = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, result.StatusCode, http.StatusUnauthorized)
			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("token为管理员/普通用户/应用账户，获取用户头像成功", func() {
			Convey("token为管理员,获取用户头像成功", func() {
				hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, adminUserToken)
				mock.ExpectQuery("select f_user_id, f_login_name, f_display_name, f_priority, f_csf_level,").
					WillReturnRows(sqlmock.NewRows(userDBColumns).AddRow("xxx", "loginName1", "displayName1", "1", "1", "0", "0", "0@qq.com",
						"2", "1", "1", "xxx", "zzz", "kkk", 0, getLocalTimeStr(), "pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13).
						AddRow("xxx1", "loginName2", "displayName2", "1", "1", "0", "0", "0@qq.com", "2", "1", "1", "xxx", "zzz", "kkk", 0, getLocalTimeStr(), "pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13))
				mock.ExpectQuery("select f_oss_id, f_key, f_type, f_time from user_management.t_avatar").
					WillReturnRows(sqlmock.NewRows([]string{"f_oss_id", "f_key", "f_type", "f_time"}).AddRow("ossID", "key1", "xxx", 123))
				mock.ExpectQuery("select f_oss_id, f_key, f_type, f_time from user_management.t_avatar").
					WillReturnRows(sqlmock.NewRows([]string{"f_oss_id", "f_key", "f_type", "f_time"}).AddRow("ossID", "key1", "xxx", 123))
				ossMock.EXPECT().HandleRequest("GET", "/api/ossgateway/v1/download/ossID/key1", nil).AnyTimes().Return(http.StatusOK, ossGatewayGetDownURL)

				req := httptest.NewRequest("GET", "/api/user-management/v1/avatars/xxx,xxx1", bytes.NewReader(reqBody))
				req.Header.Add("Authorization", "Bearer test-token")
				w := httptest.NewRecorder()
				publicEngine.ServeHTTP(w, req)
				result := w.Result()

				message, _ := io.ReadAll(result.Body)
				resBody := make([]interface{}, 0)
				_ = jsoniter.Unmarshal(message, &resBody)
				assert.Equal(t, result.StatusCode, http.StatusOK)
				assert.Equal(t, len(resBody), 2)

				testData1, ok := resBody[0].(map[string]interface{})
				assert.Equal(t, ok, true)
				assert.Equal(t, testData1["id"], "xxx")
				assert.Equal(t, testData1["avatar_url"], "url1")

				testData2, ok := resBody[1].(map[string]interface{})
				assert.Equal(t, ok, true)
				assert.Equal(t, testData2["id"], "xxx1")
				assert.Equal(t, testData2["avatar_url"], "url1")

				if err := result.Body.Close(); err != nil {
					assert.Equal(t, err, nil)
				}

				if err := mock.ExpectationsWereMet(); err != nil {
					t.Errorf("there were unfulfilled expectations: %s", err)
				}
			})

			Convey("token为普通用户,获取用户头像成功", func() {
				hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, activeUserToken)
				mock.ExpectQuery("select f_user_id, f_login_name, f_display_name, f_priority, f_csf_level,").
					WillReturnRows(sqlmock.NewRows(userDBColumns).AddRow("xxx", "loginName1", "displayName1", "1", "1", "0", "0", "0@qq.com",
						"2", "1", "1", "xxx", "zzz", "kkk", 0, getLocalTimeStr(), "pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13).
						AddRow("xxx1", "loginName2", "displayName2", "1", "1", "0", "0", "0@qq.com", "2", "1", "1", "xxx", "zzz", "kkk", 0, getLocalTimeStr(),
							"pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13))
				mock.ExpectQuery("select f_oss_id, f_key, f_type, f_time from user_management.t_avatar").
					WillReturnRows(sqlmock.NewRows([]string{"f_oss_id", "f_key", "f_type", "f_time"}).AddRow("ossID", "key1", "xxx", 123))
				mock.ExpectQuery("select f_oss_id, f_key, f_type, f_time from user_management.t_avatar").
					WillReturnRows(sqlmock.NewRows([]string{"f_oss_id", "f_key", "f_type", "f_time"}).AddRow("ossID", "key1", "xxx", 123))
				ossMock.EXPECT().HandleRequest("GET", "/api/ossgateway/v1/download/ossID/key1", nil).AnyTimes().Return(http.StatusOK, ossGatewayGetDownURL)

				req := httptest.NewRequest("GET", "/api/user-management/v1/avatars/xxx,xxx1", bytes.NewReader(reqBody))
				req.Header.Add("Authorization", "Bearer test-token")
				w := httptest.NewRecorder()
				publicEngine.ServeHTTP(w, req)
				result := w.Result()

				message, _ := io.ReadAll(result.Body)
				resBody := make([]interface{}, 0)
				_ = jsoniter.Unmarshal(message, &resBody)
				assert.Equal(t, result.StatusCode, http.StatusOK)
				assert.Equal(t, len(resBody), 2)

				testData1, ok := resBody[0].(map[string]interface{})
				assert.Equal(t, ok, true)
				assert.Equal(t, testData1["id"], "xxx")
				assert.Equal(t, testData1["avatar_url"], "url1")

				testData2, ok := resBody[1].(map[string]interface{})
				assert.Equal(t, ok, true)
				assert.Equal(t, testData2["id"], "xxx1")
				assert.Equal(t, testData2["avatar_url"], "url1")

				if err := result.Body.Close(); err != nil {
					assert.Equal(t, err, nil)
				}

				if err := mock.ExpectationsWereMet(); err != nil {
					t.Errorf("there were unfulfilled expectations: %s", err)
				}
			})

			Convey("token为应用账户,获取用户头像成功", func() {
				hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, businessUserToken)
				mock.ExpectQuery("select f_user_id, f_login_name, f_display_name, f_priority, f_csf_level,").
					WillReturnRows(sqlmock.NewRows(userDBColumns).AddRow("xxx", "loginName1", "displayName1", "1", "1", "0", "0", "0@qq.com",
						"2", "1", "1", "xxx", "zzz", "kkk", 0, getLocalTimeStr(), "pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13).
						AddRow("xxx1", "loginName2", "displayName2", "1", "1", "0", "0", "0@qq.com", "2", "1", "1", "xxx", "zzz", "kkk", 0, getLocalTimeStr(),
							"pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13))
				mock.ExpectQuery("select f_oss_id, f_key, f_type, f_time from user_management.t_avatar").
					WillReturnRows(sqlmock.NewRows([]string{"f_oss_id", "f_key", "f_type", "f_time"}).AddRow("ossID", "key1", "xxx", 123))
				mock.ExpectQuery("select f_oss_id, f_key, f_type, f_time from user_management.t_avatar").
					WillReturnRows(sqlmock.NewRows([]string{"f_oss_id", "f_key", "f_type", "f_time"}).AddRow("ossID", "key1", "xxx", 123))
				ossMock.EXPECT().HandleRequest("GET", "/api/ossgateway/v1/download/ossID/key1", nil).AnyTimes().Return(http.StatusOK, ossGatewayGetDownURL)

				req := httptest.NewRequest("GET", "/api/user-management/v1/avatars/xxx,xxx1", bytes.NewReader(reqBody))
				req.Header.Add("Authorization", "Bearer test-token")
				w := httptest.NewRecorder()
				publicEngine.ServeHTTP(w, req)
				result := w.Result()

				message, _ := io.ReadAll(result.Body)
				resBody := make([]interface{}, 0)
				_ = jsoniter.Unmarshal(message, &resBody)
				assert.Equal(t, result.StatusCode, http.StatusOK)
				assert.Equal(t, len(resBody), 2)

				testData1, ok := resBody[0].(map[string]interface{})
				assert.Equal(t, ok, true)
				assert.Equal(t, testData1["id"], "xxx")
				assert.Equal(t, testData1["avatar_url"], "url1")

				testData2, ok := resBody[1].(map[string]interface{})
				assert.Equal(t, ok, true)
				assert.Equal(t, testData2["id"], "xxx1")
				assert.Equal(t, testData2["avatar_url"], "url1")

				if err := result.Body.Close(); err != nil {
					assert.Equal(t, err, nil)
				}

				if err := mock.ExpectationsWereMet(); err != nil {
					t.Errorf("there were unfulfilled expectations: %s", err)
				}
			})
		})

		Convey("用户格式错误或者用户不存在，报错404", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, adminUserToken)
			mock.ExpectQuery("select f_user_id, f_login_name, f_display_name, f_priority, f_csf_level,").
				WillReturnRows(sqlmock.NewRows(userDBColumns).AddRow("xxx", "loginName1", "displayName1", "1", "1", "0", "0", "0@qq.com",
					"2", "1", "1", "xxx", "zzz", "kkk", 0, getLocalTimeStr(), "pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13))
			ossMock.EXPECT().HandleRequest("GET", "/api/ossgateway/v1/download/ossID/key1", nil).AnyTimes().Return(http.StatusOK, ossGatewayGetDownURL)

			req := httptest.NewRequest("GET", "/api/user-management/v1/avatars/xxx,xxx1", bytes.NewReader(reqBody))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make([]interface{}, 0)
			_ = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, result.StatusCode, http.StatusNotFound)

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("用户重复，报错400", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, adminUserToken)

			req := httptest.NewRequest("GET", "/api/user-management/v1/avatars/xxx,xxx", bytes.NewReader(reqBody))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make([]interface{}, 0)
			_ = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, result.StatusCode, http.StatusBadRequest)

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("用户未上传头像，获取成功，avatar_url为空字符串", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, businessUserToken)
			mock.ExpectQuery("select f_user_id, f_login_name, f_display_name, f_priority, f_csf_level,").
				WillReturnRows(sqlmock.NewRows(userDBColumns).AddRow("xxx", "loginName1", "displayName1", "1", "1", "0", "0", "0@qq.com",
					"2", "1", "1", "xxx", "zzz", "kkk", 0, getLocalTimeStr(), "pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13))
			mock.ExpectQuery("select f_oss_id, f_key, f_type, f_time from user_management.t_avatar").
				WillReturnRows(sqlmock.NewRows([]string{"f_oss_id", "f_key", "f_type", "f_time"}))

			req := httptest.NewRequest("GET", "/api/user-management/v1/avatars/xxx", bytes.NewReader(reqBody))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make([]interface{}, 0)
			_ = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, result.StatusCode, http.StatusOK)
			assert.Equal(t, len(resBody), 1)

			testData1, ok := resBody[0].(map[string]interface{})
			assert.Equal(t, ok, true)
			assert.Equal(t, testData1["id"], "xxx")
			assert.Equal(t, testData1["avatar_url"], "")

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})

		Convey("批量获取10个普通用户头像,获取用户头像成功", func() {
			hydraMock.EXPECT().HandleRequest("POST", "/admin/oauth2/introspect", "token=test-token").AnyTimes().Return(http.StatusOK, businessUserToken)
			mock.ExpectQuery("select f_user_id, f_login_name, f_display_name, f_priority, f_csf_level,").
				WillReturnRows(sqlmock.NewRows(userDBColumns).AddRow("xxx1", "loginName1", "displayName1", "1", "1", "0", "0", "0@qq.com", "2", "1", "1", "xxx", "zzz", "kkk", 0, getLocalTimeStr(), "pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13).
					AddRow("xxx2", "loginName2", "displayName2", "1", "1", "0", "0", "0@qq.com", "2", "1", "1", "xxx", "zzz", "kkk", 0, getLocalTimeStr(), "pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13).
					AddRow("xxx3", "loginName3", "displayName3", "1", "1", "0", "0", "0@qq.com", "2", "1", "1", "xxx", "zzz", "kkk", 0, getLocalTimeStr(), "pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13).
					AddRow("xxx4", "loginName4", "displayName4", "1", "1", "0", "0", "0@qq.com", "2", "1", "1", "xxx", "zzz", "kkk", 0, getLocalTimeStr(), "pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13).
					AddRow("xxx5", "loginName5", "displayName5", "1", "1", "0", "0", "0@qq.com", "2", "1", "1", "xxx", "zzz", "kkk", 0, getLocalTimeStr(), "pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13).
					AddRow("xxx6", "loginName6", "displayName6", "1", "1", "0", "0", "0@qq.com", "2", "1", "1", "xxx", "zzz", "kkk", 0, getLocalTimeStr(), "pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13).
					AddRow("xxx7", "loginName7", "displayName7", "1", "1", "0", "0", "0@qq.com", "2", "1", "1", "xxx", "zzz", "kkk", 0, getLocalTimeStr(), "pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13).
					AddRow("xxx8", "loginName8", "displayName8", "1", "1", "0", "0", "0@qq.com", "2", "1", "1", "xxx", "zzz", "kkk", 0, getLocalTimeStr(), "pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13).
					AddRow("xxx9", "loginName9", "displayName9", "1", "1", "0", "0", "0@qq.com", "2", "1", "1", "xxx", "zzz", "kkk", 0, getLocalTimeStr(), "pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13).
					AddRow("xxx10", "loginName10", "displayName10", "1", "1", "0", "0", "0@qq.com", "2", "1", "1", "xxx", "zzz", "kkk", 0, getLocalTimeStr(), "pwd1", "pwd2", "oss_id", "manager_id", getLocalTimeStr(), 13))
			mock.ExpectQuery("select f_oss_id, f_key, f_type, f_time from user_management.t_avatar").
				WillReturnRows(sqlmock.NewRows([]string{"f_oss_id", "f_key", "f_type", "f_time"}))
			mock.ExpectQuery("select f_oss_id, f_key, f_type, f_time from user_management.t_avatar").
				WillReturnRows(sqlmock.NewRows([]string{"f_oss_id", "f_key", "f_type", "f_time"}).AddRow("ossID", "key1", "xxx", 123))
			mock.ExpectQuery("select f_oss_id, f_key, f_type, f_time from user_management.t_avatar").
				WillReturnRows(sqlmock.NewRows([]string{"f_oss_id", "f_key", "f_type", "f_time"}).AddRow("ossID", "key1", "xxx", 123))
			mock.ExpectQuery("select f_oss_id, f_key, f_type, f_time from user_management.t_avatar").
				WillReturnRows(sqlmock.NewRows([]string{"f_oss_id", "f_key", "f_type", "f_time"}).AddRow("ossID", "key1", "xxx", 123))
			mock.ExpectQuery("select f_oss_id, f_key, f_type, f_time from user_management.t_avatar").
				WillReturnRows(sqlmock.NewRows([]string{"f_oss_id", "f_key", "f_type", "f_time"}).AddRow("ossID", "key1", "xxx", 123))
			mock.ExpectQuery("select f_oss_id, f_key, f_type, f_time from user_management.t_avatar").
				WillReturnRows(sqlmock.NewRows([]string{"f_oss_id", "f_key", "f_type", "f_time"}).AddRow("ossID", "key1", "xxx", 123))
			mock.ExpectQuery("select f_oss_id, f_key, f_type, f_time from user_management.t_avatar").
				WillReturnRows(sqlmock.NewRows([]string{"f_oss_id", "f_key", "f_type", "f_time"}).AddRow("ossID", "key1", "xxx", 123))
			mock.ExpectQuery("select f_oss_id, f_key, f_type, f_time from user_management.t_avatar").
				WillReturnRows(sqlmock.NewRows([]string{"f_oss_id", "f_key", "f_type", "f_time"}).AddRow("ossID", "key1", "xxx", 123))
			mock.ExpectQuery("select f_oss_id, f_key, f_type, f_time from user_management.t_avatar").
				WillReturnRows(sqlmock.NewRows([]string{"f_oss_id", "f_key", "f_type", "f_time"}).AddRow("ossID", "key1", "xxx", 123))
			mock.ExpectQuery("select f_oss_id, f_key, f_type, f_time from user_management.t_avatar").
				WillReturnRows(sqlmock.NewRows([]string{"f_oss_id", "f_key", "f_type", "f_time"}).AddRow("ossID", "key1", "xxx", 123))
			ossMock.EXPECT().HandleRequest("GET", "/api/ossgateway/v1/download/ossID/key1", nil).AnyTimes().Return(http.StatusOK, ossGatewayGetDownURL)

			req := httptest.NewRequest("GET", "/api/user-management/v1/avatars/xxx1,xxx2,xxx3,xxx4,xxx5,xxx6,xxx7,xxx8,xxx9,xxx10", bytes.NewReader(reqBody))
			req.Header.Add("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			publicEngine.ServeHTTP(w, req)
			result := w.Result()

			message, _ := io.ReadAll(result.Body)
			resBody := make([]interface{}, 0)
			_ = jsoniter.Unmarshal(message, &resBody)
			assert.Equal(t, result.StatusCode, http.StatusOK)
			assert.Equal(t, len(resBody), 10)

			testData1, ok := resBody[0].(map[string]interface{})
			assert.Equal(t, ok, true)
			assert.Equal(t, testData1["id"], "xxx1")
			assert.Equal(t, testData1["avatar_url"], "")

			testData2, ok := resBody[1].(map[string]interface{})
			assert.Equal(t, ok, true)
			assert.Equal(t, testData2["id"], "xxx2")
			assert.Equal(t, testData2["avatar_url"], "url1")

			testData3, ok := resBody[2].(map[string]interface{})
			assert.Equal(t, ok, true)
			assert.Equal(t, testData3["id"], "xxx3")
			assert.Equal(t, testData3["avatar_url"], "url1")

			testData4, ok := resBody[3].(map[string]interface{})
			assert.Equal(t, ok, true)
			assert.Equal(t, testData4["id"], "xxx4")
			assert.Equal(t, testData4["avatar_url"], "url1")

			testData5, ok := resBody[4].(map[string]interface{})
			assert.Equal(t, ok, true)
			assert.Equal(t, testData5["id"], "xxx5")
			assert.Equal(t, testData5["avatar_url"], "url1")

			testData6, ok := resBody[5].(map[string]interface{})
			assert.Equal(t, ok, true)
			assert.Equal(t, testData6["id"], "xxx6")
			assert.Equal(t, testData6["avatar_url"], "url1")

			testData7, ok := resBody[6].(map[string]interface{})
			assert.Equal(t, ok, true)
			assert.Equal(t, testData7["id"], "xxx7")
			assert.Equal(t, testData7["avatar_url"], "url1")

			testData8, ok := resBody[7].(map[string]interface{})
			assert.Equal(t, ok, true)
			assert.Equal(t, testData8["id"], "xxx8")
			assert.Equal(t, testData8["avatar_url"], "url1")

			testData9, ok := resBody[8].(map[string]interface{})
			assert.Equal(t, ok, true)
			assert.Equal(t, testData9["id"], "xxx9")
			assert.Equal(t, testData9["avatar_url"], "url1")

			testData10, ok := resBody[9].(map[string]interface{})
			assert.Equal(t, ok, true)
			assert.Equal(t, testData10["id"], "xxx10")
			assert.Equal(t, testData10["avatar_url"], "url1")

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	})
}
