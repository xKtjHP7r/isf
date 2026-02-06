// Package driveradapters message quene 消息队列处理
package driveradapters

import (
	"context"
	_ "embed" // 标准用法
	"net/http"
	"reflect"
	"strings"
	"sync"

	jsoniter "github.com/json-iterator/go"
	gerrors "github.com/kweaver-ai/go-lib/error"
	"github.com/kweaver-ai/go-lib/rest"
	"github.com/xeipuuv/gojsonschema"
	"golang.org/x/sync/errgroup"

	"Authorization/common"
	"Authorization/interfaces"
	"Authorization/logics"
)

// MQHandler 消息队列处理接口
type MQHandler interface {
	// Subscribe 订阅mq消息
	Subscribe(errGroup *errgroup.Group, ctx context.Context)
}

var (
	pOnce sync.Once
	mq    MQHandler
)

type mqHandler struct {
	mqClient                      interfaces.MQClient
	pollIntervalMilliseconds      int64
	maxInFlight                   int
	log                           common.Logger
	resourceNameModifySchema      *gojsonschema.Schema
	resourceAncestorsModifySchema *gojsonschema.Schema
	event                         interfaces.LogicsEvent
	policy                        interfaces.LogicsPolicy
}

//go:embed jsonschema/policy/resource_name_modify.json
var resourceNameModifySchemaStr string

//go:embed jsonschema/policy/resource_ancestors_modify.json
var resourceAncestorsModifySchemaStr string

type jsonValueDesc = rest.JSONValueDesc

// NewMQHandler 创建MQHandler
func NewMQHandler() MQHandler {
	pOnce.Do(func() {
		handler := mqHandler{
			mqClient:                      mqClient,
			pollIntervalMilliseconds:      100,
			maxInFlight:                   200,
			event:                         logics.NewEvent(),
			log:                           common.NewLogger(),
			policy:                        logics.NewPolicy(),
			resourceNameModifySchema:      newJSONSchema(resourceNameModifySchemaStr),
			resourceAncestorsModifySchema: newJSONSchema(resourceAncestorsModifySchemaStr),
		}
		mq = &handler
	})

	return mq
}

// Subscribe 订阅mq消息
func (m *mqHandler) Subscribe(g *errgroup.Group, ctx context.Context) {
	channel := "Authorization"
	topicFuncMap := map[string]func([]byte) error{
		// 发布-订阅消息
		"core.user.delete":       m.userDeleted,
		"core.dept.delete":       m.deptDelete,
		"core.group.delete":      m.userGroupDeleted,
		"core.app.deleted":       m.appDeleted,
		"core.org.name.modify":   m.updateOrgName,
		"core.app.name.modified": m.appNameModified,

		// 点对点消息
		"authorization.resource.name.modify":      m.updateResourceName,
		"authorization.resource.ancestors.modify": m.updateResourceAncestors,
	}

	for t, f := range topicFuncMap {
		topic, fn := t, f
		g.Go(func() error {
			mqErrChan := make(chan error)
			defer close(mqErrChan)
			go func() {
				mqErrChan <- m.mqClient.Sub(topic, channel, fn, m.pollIntervalMilliseconds, m.maxInFlight)
			}()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case err := <-mqErrChan:
				m.log.Errorf("订阅消息失败, topic: %s, channel: %s, err: %v", topic, channel, err)
				return err
			}
		})
	}
}

// updateOrgName 更新组织架构显示名
func (m *mqHandler) updateOrgName(message []byte) error {
	id, name, orgType, err := m.getNameUpdateParamsInNsqMsg(message)
	if err != nil {
		m.log.Errorf("updateOrgName getNameUpdateParamsInNsqMsg error:%v", err)
		// 防止 NSQ 重发消息 返回nil
		return nil
	}
	// 处理显示名变更
	if err = m.event.OrgNameModified(id, name, orgType); err != nil {
		if m.needReTry(err) {
			return err
		}
	}

	return nil
}

func (m *mqHandler) userGroupDeleted(message []byte) error {
	id, err := m.getGroupDeleteParamsInNsqMsq(message)
	if err != nil {
		m.log.Errorf("userGroupDeleted getGroupDeleteParamsInNsqMsq error:%v", err)
		// 防止 NSQ 重发消息 返回nil
		return nil
	}

	if err = m.event.UserGroupDeleted(id); err != nil {
		if m.needReTry(err) {
			return err
		}
	}

	return nil
}

type deptDeleteMsg struct {
	ID string `json:"id"`
}

func (m *mqHandler) deptDelete(message []byte) (err error) {
	var msg deptDeleteMsg
	if err = jsoniter.Unmarshal(message, &msg); err != nil {
		m.log.Errorf("deptDelete error:%v", err)
		// 防止 NSQ 重发消息 返回nil
		return nil
	}

	if err = m.event.DepartmentDeleted(msg.ID); err != nil {
		if m.needReTry(err) {
			return err
		}
		return nil
	}
	return
}

// getNameUpdateParamsInNsqMsg 获取NSQ消息内显示名更新信息
func (m *mqHandler) getNameUpdateParamsInNsqMsg(message []byte) (id, name string, orgType interfaces.AccessorType, err error) {
	var jsonV any
	if err := jsoniter.Unmarshal(message, &jsonV); err != nil {
		return "", "", orgType, err
	}

	// 检测参数类型
	objDesc := make(map[string]*jsonValueDesc)
	objDesc["id"] = &jsonValueDesc{Kind: reflect.String, Required: true}
	objDesc["new_name"] = &jsonValueDesc{Kind: reflect.String, Required: true}
	objDesc["type"] = &jsonValueDesc{Kind: reflect.String, Required: true}
	reqParamsDesc := &jsonValueDesc{Kind: reflect.Map, Required: true, ValueDesc: objDesc}
	if cErr := rest.CheckJSONValue("body", jsonV, reqParamsDesc); cErr != nil {
		return "", "", orgType, cErr
	}

	// 检测type参数是否正常
	strType := jsonV.(map[string]any)["type"].(string)
	if strType != "user" && strType != "group" && strType != "department" && strType != "contactor" {
		return "", "", orgType, gerrors.NewError(gerrors.PublicBadRequest, "invalid org type")
	}

	orgType = common.StrToAccessorTypeMap[strType]

	id = jsonV.(map[string]any)["id"].(string)
	name = jsonV.(map[string]any)["new_name"].(string)

	return
}

// getNameUpdateParamsInNsqMsg 获取NSQ消息内需要删除的用户组ID
func (m *mqHandler) getGroupDeleteParamsInNsqMsq(message []byte) (string, error) {
	var jsonV any
	if err := jsoniter.Unmarshal(message, &jsonV); err != nil {
		return "", err
	}

	// 检测参数类型
	objDesc := make(map[string]*jsonValueDesc)
	objDesc["id"] = &jsonValueDesc{Kind: reflect.String, Required: true}
	reqParamsDesc := &jsonValueDesc{Kind: reflect.Map, Required: true, ValueDesc: objDesc}
	if cErr := rest.CheckJSONValue("body", jsonV, reqParamsDesc); cErr != nil {
		return "", cErr
	}

	return jsonV.(map[string]any)["id"].(string), nil
}

func (m *mqHandler) needReTry(err error) bool {
	// 错误模型说明：
	// 错误类型分为外部错误和内部错误。每种错误可用错误码对其进行标识，错误码分为通用错误码和指定错误码。
	// 外部错误：由于接口调用方传参或调用方式不对引起的错误。标识错误码使用400系列，即以"4"开头。
	// 内部错误：由于服务自身代码问题引起的错误。标识错误码使用500系列，即以"5"开头。
	// 通用错误码：所有服务公用的错误码，一般用于标识调用方无需特殊处理的错误。当前有如下通用错误码：
	//          - 400000000：客户端请求错误
	//          - 401000000：未授权或授权已过期
	//          - 500000000：服务端内部错误
	// 指定错误码：服务自定义的错误码，以某唯一的错误码对该错误进行标识（在errors模块中定义），一般用于标识调用方需特殊处理的错误，如UI界面提示错误信息。
	// rest.HTTPError：用于存储本服务的错误信息，必有标识错误码
	// errorv2.Error: 存储错误码是string类型的错误信息，必有标识错误码
	// errors.New("xxx")：用于表示本服务产生的内部错误，当作为RESTFul API错误返回时，会在返回处使用"500000000"通用错误码进行标识。如需用指定错误码标识，则使用rest.HTTPError来表示。
	if err == nil {
		return false
	}
	// 外部错误 不需要重试 (访问者不存在、文件不存在) 可能返回值404 考虑一些非标准的错误码
	// 内部错误 服务网络不通 、服务内部崩溃、数据库崩溃 500 503 等 出错 保留审核信息 需要重试
	switch e := err.(type) {
	case *rest.HTTPError:
		// 本服务 有标识的错误
		statusCode := e.Code
		if statusCode/100000000 == http.StatusInternalServerError/100 {
			// 内部错误
			return true
		}
		// 外部错误
		return false
	case *gerrors.Error:
		code := strings.Split(e.Code, ".")[1]
		if code == gerrors.InternalServerError || code == gerrors.ServiceUnavailable {
			// 内部错误
			return true
		}
		// 外部错误
		return false
	default:
		// 本服务 未标识的的内部错误。若作为RESTFul API错误返回时，会在返回处使用"500000000"通用错误码进行标识
		return true
	}
}

func (m *mqHandler) userDeleted(message []byte) (err error) {
	userID, err := m.getIDInNsqMsg(message)
	if err != nil {
		m.log.Errorf("userDeleted getIDInNsqMsg error:%v", err)
		return nil
	}

	if err = m.event.UserDeleted(userID); err != nil {
		if m.needReTry(err) {
			return err
		}
		return nil
	}
	return nil
}

func (m *mqHandler) getIDInNsqMsg(message []byte) (string, error) {
	var jsonV any
	if err := jsoniter.Unmarshal(message, &jsonV); err != nil {
		return "", err
	}

	objDesc := make(map[string]*jsonValueDesc)
	objDesc["id"] = &jsonValueDesc{Kind: reflect.String, Required: true}
	reqParamsDesc := &jsonValueDesc{Kind: reflect.Map, Required: true, ValueDesc: objDesc}
	if cErr := rest.CheckJSONValue("body", jsonV, reqParamsDesc); cErr != nil {
		return "", cErr
	}

	return jsonV.(map[string]any)["id"].(string), nil
}

// updateResourceName 资源名称修改
func (m *mqHandler) updateResourceName(message []byte) error {
	var jsonV map[string]any
	if err := validateAndBind(message, m.resourceNameModifySchema, &jsonV); err != nil {
		m.log.Errorf("updateResourceName validateAndBind error:%v", err)
		return nil
	}

	resourceID := jsonV["id"].(string)
	resourceType := jsonV["type"].(string)
	resourceName := jsonV["name"].(string)

	ctx := context.Background()
	m.log.Debugf("updateResourceName resourceID: %s, resourceType: %s, resourceName: %s", resourceID, resourceType, resourceName)
	err := m.policy.UpdateResourceName(ctx, resourceID, resourceType, resourceName)
	if err != nil {
		m.log.Errorf("updateResourceName UpdateResourceName error:%v", err)
		if m.needReTry(err) {
			return err
		}
	}
	return nil
}

// updateResourceAncestors 资源祖先信息修改
func (m *mqHandler) updateResourceAncestors(message []byte) error {
	var jsonV map[string]any
	if err := validateAndBind(message, m.resourceAncestorsModifySchema, &jsonV); err != nil {
		m.log.Errorf("updateResourceAncestors validateAndBind error:%v", err)
		return nil
	}

	resourceID := jsonV["id"].(string)
	resourceType := jsonV["type"].(string)

	ancestorsAny := jsonV["ancestors"].([]any)
	ancestors := make([]interfaces.Ancestor, 0, len(ancestorsAny))
	for _, a := range ancestorsAny {
		am := a.(map[string]any)
		ancestors = append(ancestors, interfaces.Ancestor{
			ID:   am["id"].(string),
			Type: am["type"].(string),
			Name: am["name"].(string),
		})
	}

	ctx := context.Background()
	m.log.Debugf("updateResourceAncestors resourceID: %s, resourceType: %s, ancestorsLen: %d", resourceID, resourceType, len(ancestors))
	if err := m.policy.UpdateResourceAncestors(ctx, resourceID, resourceType, ancestors); err != nil {
		m.log.Errorf("updateResourceAncestors UpdateResourceAncestors error:%v", err)
		if m.needReTry(err) {
			return err
		}
	}
	return nil
}

// AppDeleted 删除应用账户权限
func (m *mqHandler) appDeleted(message []byte) error {
	appID, err := m.getIDInNsqMsg(message)
	if err != nil {
		m.log.Errorf("AppDeleted getIDInNsqMsg error:%v", err)
		return nil
	}

	err = m.event.AppDeleted(appID)
	if err != nil {
		m.log.Errorf("AppDeleted error:%v", err)
		if m.needReTry(err) {
			return err
		}
	} else {
		m.log.Infof("Delete app perm successfully: appId: %s", appID)
	}

	return nil
}

func (m *mqHandler) getAppInfoInNsqMsg(message []byte) (info interfaces.AppInfo, err error) {
	var jsonV any
	if err = jsoniter.Unmarshal(message, &jsonV); err != nil {
		return
	}
	objDesc := make(map[string]*jsonValueDesc)
	objDesc["id"] = &jsonValueDesc{Kind: reflect.String, Required: true}
	objDesc["new_name"] = &jsonValueDesc{Kind: reflect.String, Required: true}
	reqParamsDesc := &jsonValueDesc{Kind: reflect.Map, Required: true, ValueDesc: objDesc}
	if err = rest.CheckJSONValue("body", jsonV, reqParamsDesc); err != nil {
		return
	}
	info.ID = jsonV.(map[string]any)["id"].(string)
	info.Name = jsonV.(map[string]any)["new_name"].(string)
	return
}

// AppNameModified 更新应用账户名称
func (m *mqHandler) appNameModified(message []byte) error {
	info, err := m.getAppInfoInNsqMsg(message)
	if err != nil {
		m.log.Errorf("AppNameModified getAppInfoInNsqMsg error:%v", err)
		return nil
	}

	err = m.event.AppNameModified(&info)
	if err != nil {
		m.log.Errorf("AppNameModified error:%v", err)
		if m.needReTry(err) {
			return err
		}
	} else {
		m.log.Infof("Update app name successfully: appId: %s, appName: %s", info.ID, info.Name)
	}

	return nil
}
