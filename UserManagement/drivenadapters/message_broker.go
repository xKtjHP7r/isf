// Package drivenadapters 消息队列
package drivenadapters

import (
	"errors"
	"sync"

	jsoniter "github.com/json-iterator/go"

	"UserManagement/common"
	"UserManagement/interfaces"

	msqclient "github.com/kweaver-ai/proton-mq-sdk-go"
)

type messageBroker struct {
	log      common.Logger
	client   msqclient.ProtonMQClient
	handlers map[interfaces.MsgType]func(interface{}) (map[string]interface{}, string)
}

var (
	msgOnce sync.Once
	m       *messageBroker
)

// NewMessageBroker 创建消息发送对象
func NewMessageBroker() *messageBroker {
	msgOnce.Do(func() {
		client, err := msqclient.NewProtonMQClientFromFile("/sysvol/conf/mq_config.yaml")
		if err != nil {
			common.NewLogger().Fatalln(err)
		}
		m = &messageBroker{
			log:      common.NewLogger(),
			client:   client,
			handlers: make(map[interfaces.MsgType]func(interface{}) (map[string]interface{}, string)),
		}
		m.initHandlers()
	})
	return m
}

// Publish 消息发送
func (m *messageBroker) Publish(msgType interfaces.MsgType, msg interface{}) error {
	handler, ok := m.handlers[msgType]
	if !ok {
		err := errors.New("invalid msgType")
		return err
	}

	body, topic := handler(msg)

	// 发送消息
	message, err := jsoniter.Marshal(body)
	if err != nil {
		m.log.Errorln(err)
		return err
	}

	err = m.client.Pub(topic, message)
	if err != nil {
		m.log.Errorln(err)
		return err
	}

	return nil
}

func (m *messageBroker) initHandlers() {
	m.handlers[interfaces.DeleteGroup] = m.deleteGroupMsg
	m.handlers[interfaces.OrgNameChange] = m.orgNameChangeMsg
	m.handlers[interfaces.AppDeleted] = m.appDeletedMsg
	m.handlers[interfaces.AppNameChanged] = m.appNameChangedMsg
}

func (m *messageBroker) UserStatusChanged(id string, status bool) (err error) {
	topic := "user_management.user.status.changed"
	body := make(map[string]interface{})
	body["user_id"] = id
	body["status"] = status

	// 发送消息
	message, err := jsoniter.Marshal(body)
	if err != nil {
		m.log.Errorln(err)
		return err
	}

	err = m.client.Pub(topic, message)
	if err != nil {
		m.log.Errorln(err)
		return err
	}

	return nil
}

// OrgManagerChanged 发送组织管理员变更消息
func (m *messageBroker) OrgManagerChanged(ids []string) (err error) {
	topic := "user_management.org_manager.changed"
	body := make(map[string]interface{})
	body["ids"] = ids

	// 发送消息
	message, err := jsoniter.Marshal(body)
	if err != nil {
		m.log.Errorln(err)
		return err
	}

	err = m.client.Pub(topic, message)
	if err != nil {
		m.log.Errorln(err)
		return err
	}

	return nil
}

// DepartDeleted 部门被删除
func (m *messageBroker) DepartDeleted(id string) (err error) {
	topic := "core.dept.delete"
	body := make(map[string]interface{})
	body["id"] = id

	// 发送消息
	message, err := jsoniter.Marshal(body)
	if err != nil {
		m.log.Errorln(err)
		return err
	}

	err = m.client.Pub(topic, message)
	if err != nil {
		m.log.Errorln(err)
		return err
	}

	return nil
}

// InternalGroupDeleted 内部组被删除
func (m *messageBroker) InternalGroupDeleted(ids []string) (err error) {
	topic := "user_management.internal_group.deleted"
	body := make(map[string]interface{})
	body["ids"] = ids

	// 发送消息
	message, err := jsoniter.Marshal(body)
	if err != nil {
		m.log.Errorln(err)
		return err
	}

	err = m.client.Pub(topic, message)
	if err != nil {
		m.log.Errorln(err)
		return err
	}

	return nil
}

// ContactorDeleted 联系人组被删除
func (m *messageBroker) ContactorDeleted(ids []string) (err error) {
	topic := "core.user_management.contactor.deleted"
	body := make(map[string]interface{})
	body["ids"] = ids

	// 发送消息
	message, err := jsoniter.Marshal(body)
	if err != nil {
		m.log.Errorln(err)
		return err
	}

	err = m.client.Pub(topic, message)
	if err != nil {
		m.log.Errorln(err)
		return err
	}

	return nil
}

func (m *messageBroker) deleteGroupMsg(msg interface{}) (body map[string]interface{}, topic string) {
	body = make(map[string]interface{})
	body["id"] = msg.(string)

	return body, "core.group.delete"
}

func (m *messageBroker) orgNameChangeMsg(msg interface{}) (body map[string]interface{}, topic string) {
	temp := msg.(interfaces.NameChangeMsg)
	body = make(map[string]interface{})
	body["id"] = temp.ID
	body["new_name"] = temp.NewName
	body["type"] = temp.OType

	return body, "core.org.name.modify"
}

func (m *messageBroker) appDeletedMsg(msg interface{}) (body map[string]interface{}, topic string) {
	body = make(map[string]interface{})
	body["id"] = msg.(string)

	return body, "core.app.deleted"
}

func (m *messageBroker) appNameChangedMsg(msg interface{}) (body map[string]interface{}, topic string) {
	temp := msg.(interfaces.AppInfo)
	body = make(map[string]interface{})
	body["id"] = temp.ID
	body["new_name"] = temp.Name

	return body, "core.app.name.modified"
}

func (m *messageBroker) AnonymityAuth(msgType string, msg interface{}) error {
	body := map[string]any{
		"id":             msg.(map[string]any)["id"].(string),
		"accessed_times": msg.(map[string]any)["accessed_times"].(int32),
		"referrer":       msg.(map[string]any)["referrer"].(string),
	}

	message, err := jsoniter.Marshal(body)
	if err != nil {
		m.log.Errorln(err)
		return err
	}

	topic := "core.anonymity.auth." + msgType
	err = m.client.Pub(topic, message)
	if err != nil {
		m.log.Errorln(err)
		return err
	}

	return nil
}
