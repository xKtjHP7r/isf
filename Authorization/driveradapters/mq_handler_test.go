package driveradapters

import (
	"context"
	"errors"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/assert"
	jsoniter "github.com/json-iterator/go"
	gerrors "github.com/kweaver-ai/go-lib/error"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
	"golang.org/x/sync/errgroup"

	"Authorization/common"
	"Authorization/interfaces/mock"
)

func TestSubscribe(t *testing.T) {
	Convey("订阅", t, func() {
		test := setGinMode()
		defer test()
		engine := gin.New()
		engine.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		h := mock.NewMockMQClient(ctrl)

		handler := mqHandler{
			mqClient: h,
			log:      common.NewLogger(),
		}
		Convey("Subscribe ok", func() {
			g, ctx := errgroup.WithContext(context.Background())
			h.EXPECT().Sub(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			handler.Subscribe(g, ctx)
		})
	})
}

func TestUserDeleted(t *testing.T) {
	Convey("用户删除", t, func() {
		test := setGinMode()
		defer test()
		engine := gin.New()
		engine.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		e := mock.NewMockLogicsEvent(ctrl)

		handler := mqHandler{
			event: e,
			log:   common.NewLogger(),
		}

		Convey("userDeleted params is error", func() {
			errMsg, _ := jsoniter.Marshal(map[string]any{"isd": "xxxx"})
			err := handler.userDeleted(errMsg)
			assert.Equal(t, err, nil)
		})

		Convey("userDeleted params id  int", func() {
			errMsg, _ := jsoniter.Marshal(map[string]any{"id": 1})
			e.EXPECT().UserDeleted(gomock.Any()).AnyTimes().Return(nil)
			err := handler.userDeleted(errMsg)
			assert.Equal(t, err, nil)
		})

		Convey("userDeleted params is ok", func() {
			errMsg, _ := jsoniter.Marshal(map[string]any{"id": "xxxx"})
			e.EXPECT().UserDeleted(gomock.Any()).AnyTimes().Return(nil)
			err := handler.userDeleted(errMsg)
			assert.Equal(t, err, nil)
		})
	})
}

func TestDeptDelete(t *testing.T) {
	Convey("TestDeptDelete", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		event := mock.NewMockLogicsEvent(ctrl)
		handler := mqHandler{
			event: event,
			log:   common.NewLogger(),
		}

		Convey("Unmarshal error", func() {
			err := handler.deptDelete([]byte("invalid json"))
			assert.Equal(t, nil, err)
		})

		Convey("DepartmentDeleted error", func() {
			msg := []byte(`{"ID": "123"}`)
			event.EXPECT().DepartmentDeleted("123").Return(errors.New("some error"))
			err := handler.deptDelete(msg)
			assert.NotEqual(t, nil, err)
		})

		Convey("DepartmentDeleted success", func() {
			msg := []byte(`{"ID": "123"}`)
			event.EXPECT().DepartmentDeleted("123").Return(nil)
			err := handler.deptDelete(msg)
			assert.Equal(t, nil, err)
		})
	})
}

func TestDeleteGroup(t *testing.T) {
	Convey("DeleteGroup", t, func() {
		test := setGinMode()
		defer test()
		engine := gin.New()
		engine.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		e := mock.NewMockLogicsEvent(ctrl)
		handler := mqHandler{
			event: e,
			log:   common.NewLogger(),
		}

		Convey("params  is xxx", func() {
			e.EXPECT().UserGroupDeleted(gomock.Any()).AnyTimes().Return(nil)
			err := handler.userGroupDeleted([]byte("xxxx"))
			assert.Equal(t, err, nil)
		})

		Convey("params  is ok", func() {
			errMsg, _ := jsoniter.Marshal(map[string]any{"apply_id": "xxxx", "result": true})
			e.EXPECT().UserGroupDeleted(gomock.Any()).AnyTimes().Return(nil)
			err := handler.userGroupDeleted(errMsg)
			assert.Equal(t, err, nil)
		})
	})
}

func TestDeleteApp(t *testing.T) {
	Convey("根据 NSQ 删除应用账户", t, func() {
		test := setGinMode()
		defer test()
		engine := gin.New()
		engine.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		e := mock.NewMockLogicsEvent(ctrl)
		handler := mqHandler{
			event: e,
			log:   common.NewLogger(),
		}

		Convey("AppDeleted 参数是 string is string", func() {
			errMsg, _ := jsoniter.Marshal("i am string")
			err := handler.appDeleted(errMsg)
			assert.Equal(t, err, nil)
		})

		Convey("AppDeleted 成功", func() {
			nsqValue := map[string]any{
				"id": "d22f7ec5-231f-35f5-a495-9194b66193e4",
			}
			var err error
			nsqMsg, _ := jsoniter.Marshal(nsqValue)
			e.EXPECT().AppDeleted(gomock.Any()).Times(1).Return(nil)
			err = handler.appDeleted(nsqMsg)
			assert.Equal(t, err, nil)
		})

		Convey("AppDeleted 失败", func() {
			nsqValue := map[string]any{
				"id": "d22f7ec5-231f-35f5-a495-9194b66193e4",
			}
			var err error
			nsqMsg, _ := jsoniter.Marshal(nsqValue)
			e.EXPECT().AppDeleted(gomock.Any()).Times(1).Return(errors.New("logic AppDeleted error"))
			err = handler.appDeleted(nsqMsg)
			assert.NotEqual(t, err, nil)
		})
	})
}

func TestUpdateOrgName(t *testing.T) {
	Convey("updateOrgName", t, func() {
		test := setGinMode()
		defer test()
		engine := gin.New()
		engine.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		e := mock.NewMockLogicsEvent(ctrl)

		handler := mqHandler{
			event: e,
			log:   common.NewLogger(),
		}

		Convey("params no id", func() {
			errMsg, _ := jsoniter.Marshal(map[string]any{"new_name": "xxxx", "type": "xxx"})

			err := handler.updateOrgName(errMsg)
			assert.Equal(t, err, nil)
		})

		Convey("params no name", func() {
			errMsg, _ := jsoniter.Marshal(map[string]any{"type": "xxxx", "id": "xxx"})

			err := handler.updateOrgName(errMsg)
			assert.Equal(t, err, nil)
		})

		Convey("params no type", func() {
			errMsg, _ := jsoniter.Marshal(map[string]any{"new_name": "xxxx", "id": "xxx"})

			err := handler.updateOrgName(errMsg)
			assert.Equal(t, err, nil)
		})

		Convey("type is ileagal", func() {
			errMsg, _ := jsoniter.Marshal(map[string]any{"type": "xxx", "new_name": "xxxx", "id": "xxx"})

			err := handler.updateOrgName(errMsg)
			assert.Equal(t, err, nil)
		})

		Convey("UpdateOrgName error", func() {
			testErr := gerrors.NewError(gerrors.PublicBadRequest, "param expires_at is invalid")
			msg, _ := jsoniter.Marshal(map[string]any{"type": "user", "new_name": "xxxx", "id": "xxx"})
			e.EXPECT().OrgNameModified(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(testErr)

			err := handler.updateOrgName(msg)
			assert.Equal(t, err, nil)
		})

		Convey("Success", func() {
			msg, _ := jsoniter.Marshal(map[string]any{"type": "user", "new_name": "xxxx", "id": "xxx"})
			e.EXPECT().OrgNameModified(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

			err := handler.updateOrgName(msg)
			assert.Equal(t, err, nil)
		})
	})
}

func TestUpdateAppName(t *testing.T) {
	Convey("根据 NSQ 更新应用账户名称", t, func() {
		test := setGinMode()
		defer test()
		engine := gin.New()
		engine.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		e := mock.NewMockLogicsEvent(ctrl)

		handler := mqHandler{
			event: e,
			log:   common.NewLogger(),
		}

		Convey("AppNameModified 参数是 string is string", func() {
			errMsg, _ := jsoniter.Marshal("i am string")
			err := handler.appNameModified(errMsg)
			assert.Equal(t, err, nil)
		})

		Convey("AppNameModified 成功", func() {
			nsqValue := map[string]any{
				"id":       "d22f7ec5-231f-35f5-a495-9194b66193e4",
				"new_name": "app_new1",
			}
			var err error
			nsqMsg, _ := jsoniter.Marshal(nsqValue)
			e.EXPECT().AppNameModified(gomock.Any()).Times(1).Return(nil)
			err = handler.appNameModified(nsqMsg)
			assert.Equal(t, err, nil)
		})

		Convey("AppNameModified 失败", func() {
			nsqValue := map[string]any{
				"id":       "d22f7ec5-231f-35f5-a495-9194b66193e4",
				"new_name": "app_new1",
			}
			var err error
			nsqMsg, _ := jsoniter.Marshal(nsqValue)
			e.EXPECT().AppNameModified(gomock.Any()).Times(1).Return(errors.New("logic AppNameModified error"))
			err = handler.appNameModified(nsqMsg)
			assert.NotEqual(t, err, nil)
		})
	})
}

func TestUpdateResourceName(t *testing.T) {
	Convey("根据 NSQ 更新资源名称", t, func() {
		test := setGinMode()
		defer test()
		engine := gin.New()
		engine.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		policy := mock.NewMockLogicsPolicy(ctrl)

		handler := mqHandler{
			policy:                   policy,
			log:                      common.NewLogger(),
			resourceNameModifySchema: newJSONSchema(resourceNameModifySchemaStr),
		}

		Convey("updateResourceName 参数是 string is string", func() {
			errMsg, _ := jsoniter.Marshal("i am string")
			err := handler.updateResourceName(errMsg)
			assert.Equal(t, err, nil)
		})

		Convey("updateResourceName 成功", func() {
			nsqValue := map[string]any{
				"id":   "d22f7ec5-231f-35f5-a495-9194b66193e4",
				"type": "vega_logic_view",
				"name": "元数据视图",
			}
			var err error
			nsqMsg, _ := jsoniter.Marshal(nsqValue)
			policy.EXPECT().UpdateResourceName(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil)
			err = handler.updateResourceName(nsqMsg)
			assert.Equal(t, err, nil)
		})
	})
}

func TestUpdateResourceAncestors(t *testing.T) {
	Convey("根据 NSQ 更新资源祖先信息", t, func() {
		test := setGinMode()
		defer test()
		engine := gin.New()
		engine.Use(gin.Recovery())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		policy := mock.NewMockLogicsPolicy(ctrl)

		handler := mqHandler{
			policy:                        policy,
			log:                           common.NewLogger(),
			resourceAncestorsModifySchema: newJSONSchema(resourceAncestorsModifySchemaStr),
		}

		Convey("updateResourceAncestors 参数是 string", func() {
			errMsg, _ := jsoniter.Marshal("i am string")
			err := handler.updateResourceAncestors(errMsg)
			assert.Equal(t, err, nil)
		})

		Convey("updateResourceAncestors 成功", func() {
			nsqValue := map[string]any{
				"id":   "resource-1",
				"type": "type-1",
				"ancestors": []any{
					map[string]any{
						"id":   "A",
						"type": "view",
						"name": "数据视图A",
					},
				},
			}
			nsqMsg, _ := jsoniter.Marshal(nsqValue)
			policy.EXPECT().UpdateResourceAncestors(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil)
			err := handler.updateResourceAncestors(nsqMsg)
			assert.Equal(t, err, nil)
		})
	})
}
