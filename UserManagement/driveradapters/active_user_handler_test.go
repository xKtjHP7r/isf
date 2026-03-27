package driveradapters

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/common"
	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/interfaces"
	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/UserManagement/interfaces/mock"
	gerrors "devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/go-lib/error"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/assert"
	jsoniter "github.com/json-iterator/go"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

func TestGetActiveUserInfo(t *testing.T) {
	Convey("getActiveUserInfo", t, func() {
		test := setGinMode()
		defer test()
		r := gin.New()
		r.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		h := mock.NewMockHydra(ctrl)
		activeUserLogics := mock.NewMockLogicsActiveUser(ctrl)

		testActiveUserRH := &activeUserRestHandler{
			activeUser: activeUserLogics,
			hydra:      h,
			i18n: common.NewI18n(common.I18nMap{
				i18nIDObjectsMonthlyActiveUserFileName: {
					interfaces.SimplifiedChinese:  "%d-%02d月度活跃报表.csv",
					interfaces.TraditionalChinese: "%d-%02d月度活躍報表.csv",
					interfaces.AmericanEnglish:    "%d-%02d Monthly Active Report.csv",
				},
				i18nIDObjectsYearlyActiveUserFileName: {
					interfaces.SimplifiedChinese:  "%d年度活跃报表.csv",
					interfaces.TraditionalChinese: "%d年度活躍報表.csv",
					interfaces.AmericanEnglish:    "%d Annual Active Report.csv",
				},
			}),
		}

		testActiveUserRH.RegisterPublic(r)
		target := "/api/user-management/v1/active-user-report"

		Convey("getActiveUserInfo hydra验证失败", func() {
			h.EXPECT().Introspect(gomock.Any()).Return(interfaces.TokenIntrospectInfo{
				Active:    false,
				VisitorID: "department_id",
			}, nil)

			req := httptest.NewRequest("GET", target, http.NoBody)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			result := w.Result()

			assert.Equal(t, result.StatusCode, http.StatusUnauthorized)
		})

		Convey("getActiveUserInfo hydra验证成功, 但是query不包含year", func() {
			h.EXPECT().Introspect(gomock.Any()).Return(interfaces.TokenIntrospectInfo{
				Active:    true,
				VisitorID: "user_id",
			}, nil)

			req := httptest.NewRequest("GET", target, http.NoBody)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			result := w.Result()
			respBody, _ := io.ReadAll(result.Body)

			respParam := gerrors.NewError(gerrors.PublicInternalServerError, "")
			err := jsoniter.Unmarshal(respBody, &respParam)
			assert.Equal(t, err, nil)
			assert.Equal(t, gerrors.PublicBadRequest, respParam.Code)
			assert.Equal(t, "year is required", respParam.Description)

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			assert.Equal(t, result.StatusCode, http.StatusBadRequest)
		})

		Convey("getActiveUserInfo hydra验证成功, 但是query内year不是数字", func() {
			h.EXPECT().Introspect(gomock.Any()).Return(interfaces.TokenIntrospectInfo{
				Active:    true,
				VisitorID: "user_id",
			}, nil)

			req := httptest.NewRequest("GET", target+"?year=abc", http.NoBody)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			result := w.Result()
			respBody, _ := io.ReadAll(result.Body)
			respParam := gerrors.NewError(gerrors.PublicInternalServerError, "")
			err := jsoniter.Unmarshal(respBody, &respParam)
			assert.Equal(t, err, nil)
			assert.Equal(t, gerrors.PublicBadRequest, respParam.Code)
			assert.Equal(t, "year is invalid: strconv.Atoi: parsing \"abc\": invalid syntax", respParam.Description)
		})

		Convey("getActiveUserInfo hydra验证成功, 但是query内month不是数字", func() {
			h.EXPECT().Introspect(gomock.Any()).Return(interfaces.TokenIntrospectInfo{
				Active:    true,
				VisitorID: "user_id",
			}, nil)

			req := httptest.NewRequest("GET", target+"?year=2026&month=abc", http.NoBody)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			result := w.Result()
			respBody, _ := io.ReadAll(result.Body)
			respParam := gerrors.NewError(gerrors.PublicInternalServerError, "")
			err := jsoniter.Unmarshal(respBody, &respParam)
			assert.Equal(t, err, nil)
			assert.Equal(t, gerrors.PublicBadRequest, respParam.Code)
			assert.Equal(t, "month is invalid: strconv.Atoi: parsing \"abc\": invalid syntax", respParam.Description)

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			assert.Equal(t, result.StatusCode, http.StatusBadRequest)
		})

		Convey("getActiveUserInfo hydra验证成功, 但是query内month不在1-12之间", func() {
			h.EXPECT().Introspect(gomock.Any()).Return(interfaces.TokenIntrospectInfo{
				Active:    true,
				VisitorID: "user_id",
			}, nil)

			req := httptest.NewRequest("GET", target+"?year=2026&month=13", http.NoBody)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			result := w.Result()
			respBody, _ := io.ReadAll(result.Body)
			respParam := gerrors.NewError(gerrors.PublicInternalServerError, "")
			err := jsoniter.Unmarshal(respBody, &respParam)
			assert.Equal(t, err, nil)
			assert.Equal(t, gerrors.PublicBadRequest, respParam.Code)
			assert.Equal(t, "month is invalid", respParam.Description)
		})

		Convey("getActiveUserInfo hydra验证成功, GetActiveUserInfo报错", func() {
			h.EXPECT().Introspect(gomock.Any()).Return(interfaces.TokenIntrospectInfo{
				Active:    true,
				VisitorID: "user_id",
			}, nil)

			testErr := gerrors.NewError(gerrors.PublicInternalServerError, "GetActiveUserInfo报错")
			activeUserLogics.EXPECT().GetActiveUserInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, testErr)

			req := httptest.NewRequest("GET", target+"?year=2026&month=12", http.NoBody)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			result := w.Result()
			respBody, _ := io.ReadAll(result.Body)
			respParam := gerrors.NewError(gerrors.PublicInternalServerError, "")
			err := jsoniter.Unmarshal(respBody, &respParam)
			assert.Equal(t, err, nil)
			assert.Equal(t, gerrors.PublicInternalServerError, respParam.Code)
			assert.Equal(t, "GetActiveUserInfo报错", respParam.Description)

			if err := result.Body.Close(); err != nil {
				assert.Equal(t, err, nil)
			}

			assert.Equal(t, result.StatusCode, http.StatusInternalServerError)
		})

		Convey("getActiveUserInfo hydra验证成功, GetActiveUserInfo成功", func() {
			h.EXPECT().Introspect(gomock.Any()).Return(interfaces.TokenIntrospectInfo{
				Active:    true,
				VisitorID: "user_id",
			}, nil)

			tempOut := []byte("test")
			activeUserLogics.EXPECT().GetActiveUserInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(tempOut, nil)

			req := httptest.NewRequest("GET", target+"?year=2026&month=12", http.NoBody)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			result := w.Result()
			respBody, _ := io.ReadAll(result.Body)
			assert.Equal(t, result.StatusCode, http.StatusOK)
			assert.Equal(t, string(tempOut), string(respBody))
		})
	})
}
