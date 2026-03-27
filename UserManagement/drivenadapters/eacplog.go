// Package drivenadapters 日志处理接口
package drivenadapters

import (
	"fmt"
	"strings"
	"sync"

	msqclient "github.com/kweaver-ai/proton-mq-sdk-go"
	jsoniter "github.com/json-iterator/go"
	"github.com/satori/uuid"

	"UserManagement/common"
	"UserManagement/interfaces"
)

var (
	logOnce sync.Once
	l       *eacplogSvc
)

type msgInfo struct {
	msg      string
	exMsg    string
	logType  logType
	logLevel logLevel
	opType   int32
}

type eacplogSvc struct {
	log           common.Logger
	handlers      map[interfaces.Operation]func(interface{}) msgInfo
	mapOrgTypeStr map[interfaces.OrgType]string
	mapOrgPermStr map[interfaces.AppOrgPermValue]string
	userTypeMap   map[interfaces.VisitorType]string
	logTypeMap    map[logType]string
	client        msqclient.ProtonMQClient
}

// logType 日志类型
type logType int

const (
	// ltLogin 登录日志
	ltLogin logType = 10

	// ltManage 管理日志
	ltManage logType = 11

	// ltOperation 操作日志
	ltOperation logType = 12
)

// logLevel 消息级别
type logLevel int

const (
	// llInfo 信息
	llInfo logLevel = 1

	// llWarn 警告
	llWarn logLevel = 2
)

// manageOpType 管理日志操作类型
type manageOpType int

const (
	// mtCreate 创建
	mtCreate manageOpType = 1

	// mtAdd 新增
	mtAdd manageOpType = 2

	// mtSet 设置
	mtSet manageOpType = 3

	// mtDelete 删除
	mtDelete manageOpType = 4

	// mtExport 导出
	mtExport manageOpType = 9

	// mtUpdate 更新
	mtUpdate manageOpType = 21
)

// NewEacpLog 创建日志处理对象
func NewEacpLog() *eacplogSvc {
	logOnce.Do(func() {
		client, err := msqclient.NewProtonMQClientFromFile("/sysvol/conf/mq_config.yaml")
		if err != nil {
			common.NewLogger().Fatalln(err)
		}

		l = &eacplogSvc{
			log:      common.NewLogger(),
			handlers: make(map[interfaces.Operation]func(interface{}) msgInfo),
			mapOrgTypeStr: map[interfaces.OrgType]string{
				interfaces.User:       "IDS_ORG_TYPE_USER",
				interfaces.Department: "IDS_ORG_TYPE_DEPART",
				interfaces.Group:      "IDS_ORG_TYPE_GROUP",
			},
			mapOrgPermStr: map[interfaces.AppOrgPermValue]string{
				interfaces.Modify: "IDS_ORG_PERM_MODIFY",
				interfaces.Read:   "IDS_ORG_PERM_READ",
			},
			userTypeMap: map[interfaces.VisitorType]string{
				interfaces.RealName:  "authenticated_user",
				interfaces.Anonymous: "anonymous_user",
				interfaces.App:       "app",
			},
			logTypeMap: map[logType]string{
				ltLogin:     "as.audit_log.log_login",
				ltManage:    "as.audit_log.log_management",
				ltOperation: "as.audit_log.log_operation",
			},
			client: client,
		}
		l.initHandlers()
	})

	return l
}

// 记录日志
func (e *eacplogSvc) EacpLog(visitor *interfaces.Visitor, op interfaces.Operation, logInfo interface{}) error {
	handler, ok := e.handlers[op]
	if !ok {
		err := fmt.Errorf("invalid operation type: %v", op)
		e.log.Errorln(err)
		return nil
	}
	info := handler(logInfo)

	return e.writeLog(visitor, &info)
}

// publish 消息发送
func (e *eacplogSvc) publish(topic string, msg interface{}) error {
	// 发送消息
	message, err := jsoniter.Marshal(msg)
	if err != nil {
		e.log.Errorln(err)
		return err
	}

	err = e.client.Pub(topic, message)
	if err != nil {
		e.log.Errorln(err)
		return err
	}
	return nil
}

func (e *eacplogSvc) writeLog(visitor *interfaces.Visitor, info *msgInfo) (err error) {
	//
	strUUID := uuid.Must(uuid.NewV4(), err).String()
	if err != nil {
		e.log.Errorln("eacplog write log uuid err :%v, topic:%v", err)
		return
	}

	// 消息整合
	body := make(map[string]interface{})
	body["user_id"] = visitor.ID
	body["user_type"] = e.userTypeMap[visitor.Type]
	body["level"] = info.logLevel
	body["date"] = common.Now().UnixNano() / 1e3
	body["ip"] = visitor.IP
	body["mac"] = visitor.Mac
	body["msg"] = info.msg
	body["ex_msg"] = info.exMsg
	body["user_agent"] = visitor.UserAgent
	body["op_type"] = info.opType
	body["out_biz_id"] = strUUID

	if visitor.ID == common.EacpLogSystemID {
		body["user_type"] = "internal_service"
		body["user_name"] = loadString("IDS_SYSTEM")

		tempAddInfo := map[string]interface{}{
			"user_account": loadString("IDS_SYSTEM"),
		}

		// 发送消息
		var strAdditionInfo []byte
		strAdditionInfo, err = jsoniter.Marshal(tempAddInfo)
		if err != nil {
			e.log.Errorln("eacplog write log additional info marshal err :%v", err)
			return
		}

		body["additional_info"] = string(strAdditionInfo)
	}

	err = e.publish(e.logTypeMap[info.logType], body)
	if err != nil {
		e.log.Errorln("eacplog write log err :%v, topic:%v", err, e.logTypeMap[info.logType])
	}
	return
}

func (e *eacplogSvc) initHandlers() {
	e.handlers[interfaces.OpAddGroup] = e.addGroupLog
	e.handlers[interfaces.OpDeleteGroup] = e.deleteGroupLog
	e.handlers[interfaces.OpModifyGroup] = e.modifyGroupLog
	e.handlers[interfaces.OpDeleteGroupMembers] = e.deleteGroupMembersLog
	e.handlers[interfaces.OpAddGroupMembers] = e.addGroupMembersLog
	e.handlers[interfaces.OpAppRegister] = e.appRegisterLog
	e.handlers[interfaces.OpDeleteApp] = e.deleteAppLog
	e.handlers[interfaces.OpUpdateApp] = e.updateAppLog
	e.handlers[interfaces.OpAppTokenGenerated] = e.appTokenGeneratedLog
}

func (e *eacplogSvc) appTokenGeneratedLog(logInfo interface{}) msgInfo {
	name := logInfo.(string)
	msg := fmt.Sprintf(loadString("IDS_APP_TOKEN_GENERATED_SUCCESS"), name)
	return msgInfo{
		msg:      msg,
		logType:  ltManage,
		logLevel: llInfo,
		opType:   int32(mtUpdate),
	}
}

func (e *eacplogSvc) updateAppLog(logInfo interface{}) msgInfo {
	name := logInfo.(string)
	msg := fmt.Sprintf(loadString("IDS_UPDATE_APP_SUCCESS"), name)
	return msgInfo{
		msg:      msg,
		logType:  ltManage,
		logLevel: llInfo,
		opType:   int32(mtSet),
	}
}

func (e *eacplogSvc) deleteAppLog(logInfo interface{}) msgInfo {
	name := logInfo.(string)
	msg := fmt.Sprintf(loadString("IDS_DELETE_APP_SUCCESS"), name)
	return msgInfo{
		msg:      msg,
		logType:  ltManage,
		logLevel: llWarn,
		opType:   int32(mtDelete),
	}
}

func (e *eacplogSvc) appRegisterLog(logInfo interface{}) msgInfo {
	name := logInfo.(string)
	msg := fmt.Sprintf(loadString("IDS_APP_REGISTER_SUCCESS"), name)
	return msgInfo{
		msg:      msg,
		logType:  ltManage,
		logLevel: llInfo,
		opType:   int32(mtCreate),
	}
}

func (e *eacplogSvc) addGroupMembersLog(logInfo interface{}) msgInfo {
	msg := loadString("IDS_GROUP_MEMEBERS_ADDED_SUCCESS")
	tempMsg := logInfo.(interfaces.GroupMemberNames)
	memberNames := strings.Join(tempMsg.MemberNames, ",")
	exMsg := fmt.Sprintf("%v; %v", fmt.Sprintf(loadString("IDS_GROUP_NAME"), tempMsg.GroupName),
		fmt.Sprintf(loadString("IDS_GROUP_MEMBERS"), memberNames))
	return msgInfo{
		msg:      msg,
		exMsg:    exMsg,
		logType:  ltManage,
		logLevel: llInfo,
		opType:   int32(mtAdd),
	}
}

func (e *eacplogSvc) deleteGroupMembersLog(logInfo interface{}) msgInfo {
	msg := loadString("IDS_GROUP_MEMEBERS_DELETED_SUCCESS")
	tempMsg := logInfo.(interfaces.GroupMemberNames)
	memberNames := strings.Join(tempMsg.MemberNames, ",")
	exMsg := fmt.Sprintf("%v; %v", fmt.Sprintf(loadString("IDS_GROUP_NAME"), tempMsg.GroupName),
		fmt.Sprintf(loadString("IDS_GROUP_MEMBERS"), memberNames))
	return msgInfo{
		msg:      msg,
		exMsg:    exMsg,
		logType:  ltManage,
		logLevel: llInfo,
		opType:   int32(mtDelete),
	}
}

func (e *eacplogSvc) modifyGroupLog(logInfo interface{}) msgInfo {
	msg := loadString("IDS_GROUP_MODIFIED_SUCCESS")
	tempMsg := logInfo.(interfaces.GroupInfo)
	exMsg := fmt.Sprintf("%v; %v", fmt.Sprintf(loadString("IDS_GROUP_NAME"), tempMsg.Name),
		fmt.Sprintf(loadString("IDS_GROUP_NOTES"), tempMsg.Notes))
	return msgInfo{
		msg:      msg,
		exMsg:    exMsg,
		logType:  ltManage,
		logLevel: llInfo,
		opType:   int32(mtSet),
	}
}

func (e *eacplogSvc) deleteGroupLog(logInfo interface{}) msgInfo {
	msg := loadString("IDS_GROUP_DELETED_SUCCESS")
	tempMsg := logInfo.(interfaces.GroupInfo)
	exMsg := fmt.Sprintf("%v; %v", fmt.Sprintf(loadString("IDS_GROUP_NAME"), tempMsg.Name),
		fmt.Sprintf(loadString("IDS_GROUP_NOTES"), tempMsg.Notes))
	return msgInfo{
		msg:      msg,
		exMsg:    exMsg,
		logType:  ltManage,
		logLevel: llInfo,
		opType:   int32(mtDelete),
	}
}

func (e *eacplogSvc) addGroupLog(logInfo interface{}) msgInfo {
	msg := loadString("IDS_GROUP_CREATED_SUCCESS")
	tempMsg := logInfo.(interfaces.GroupInfo)
	exMsg := fmt.Sprintf("%v; %v", fmt.Sprintf(loadString("IDS_GROUP_NAME"), tempMsg.Name),
		fmt.Sprintf(loadString("IDS_GROUP_NOTES"), tempMsg.Notes))
	return msgInfo{
		msg:      msg,
		exMsg:    exMsg,
		logType:  ltManage,
		logLevel: llInfo,
		opType:   int32(mtCreate),
	}
}

// OpDeleteDepart 删除部门
func (e *eacplogSvc) OpDeleteDepart(visitor *interfaces.Visitor, departName string, isRoot bool) (err error) {
	msg := loadString("IDS_DEPART_DEPART")
	if isRoot {
		msg = loadString("IDS_ORG_DEPART")
	}
	msg = fmt.Sprintf(msg, departName)

	info := &msgInfo{
		msg:      msg,
		logType:  ltManage,
		logLevel: llWarn,
		opType:   int32(mtDelete),
	}

	return e.writeLog(visitor, info)
}

// OpSetDefaultPWDLog 更新用户初始密码
func (e *eacplogSvc) OpSetDefaultPWDLog(visitor *interfaces.Visitor) error {
	msg := loadString("IDS_SET_DEFAULT_PWD")
	info := &msgInfo{
		msg:      msg,
		logType:  ltManage,
		logLevel: llInfo,
		opType:   int32(mtSet),
	}

	return e.writeLog(visitor, info)
}

// OpAddOrgPermAppLog 增加应用账户组织架构管理权限
func (e *eacplogSvc) OpAddOrgPermAppLog(visitor *interfaces.Visitor, perm *interfaces.AppOrgPerm) error {
	msg := fmt.Sprintf(loadString("IDS_SET_ORG_PERM_APP_SUCCESS"), perm.Name)
	exMsg := fmt.Sprintf("%v; %v", fmt.Sprintf(loadString("IDS_ORG_TYPE"), loadString(e.mapOrgTypeStr[perm.Object])),
		fmt.Sprintf(loadString("IDS_ORG_PERM"), e.formatOrgPermAppStr(perm.Value)))

	info := &msgInfo{
		msg:      msg,
		exMsg:    exMsg,
		logType:  ltManage,
		logLevel: llInfo,
		opType:   int32(mtCreate),
	}

	return e.writeLog(visitor, info)
}

// OpDeleteOrgPermAppLog 删除应用账户组织架构管理权限
func (e *eacplogSvc) OpDeleteOrgPermAppLog(visitor *interfaces.Visitor, perm *interfaces.AppOrgPerm) error {
	msg := fmt.Sprintf(loadString("IDS_DELETE_ORG_PERM_APP_SUCCESS"), perm.Name)
	exMsg := fmt.Sprintf("%v; %v", fmt.Sprintf(loadString("IDS_ORG_TYPE"), loadString(e.mapOrgTypeStr[perm.Object])),
		fmt.Sprintf(loadString("IDS_ORG_PERM"), e.formatOrgPermAppStr(perm.Value)))

	info := &msgInfo{
		msg:      msg,
		exMsg:    exMsg,
		logType:  ltManage,
		logLevel: llWarn,
		opType:   int32(mtDelete),
	}

	return e.writeLog(visitor, info)
}

// OpUpdateOrgPermAppLog 更新应用账户组织架构管理权限
func (e *eacplogSvc) OpUpdateOrgPermAppLog(visitor *interfaces.Visitor, perm *interfaces.AppOrgPerm) error {
	msg := fmt.Sprintf(loadString("IDS_MODIFY_ORG_PERM_APP_SUCCESS"), perm.Name)
	exMsg := fmt.Sprintf("%v; %v", fmt.Sprintf(loadString("IDS_ORG_TYPE"), loadString(e.mapOrgTypeStr[perm.Object])),
		fmt.Sprintf(loadString("IDS_ORG_PERM"), e.formatOrgPermAppStr(perm.Value)))

	info := &msgInfo{
		msg:      msg,
		exMsg:    exMsg,
		logType:  ltManage,
		logLevel: llInfo,
		opType:   int32(mtSet),
	}

	return e.writeLog(visitor, info)
}

// OpSetCSFLevelEnumLog 更新密级枚举
func (e *eacplogSvc) OpSetCSFLevelEnumLog(visitor *interfaces.Visitor, csfLevelEnum []string) error {
	msg := fmt.Sprintf(loadString("IDS_SET_CSF_LEVEL_ENUM_SUCCESS"), strings.Join(csfLevelEnum, ","))
	info := &msgInfo{
		msg:      msg,
		logType:  ltManage,
		logLevel: llInfo,
		opType:   int32(mtSet),
	}
	return e.writeLog(visitor, info)
}

// OpSetCSFLevel2EnumLog 更新密级2枚举
func (e *eacplogSvc) OpSetCSFLevel2EnumLog(visitor *interfaces.Visitor, csfLevel2Enum []string) error {
	msg := fmt.Sprintf(loadString("IDS_SET_CSF_LEVEL2_ENUM_SUCCESS"), strings.Join(csfLevel2Enum, ","))
	info := &msgInfo{
		msg:      msg,
		logType:  ltManage,
		logLevel: llInfo,
		opType:   int32(mtSet),
	}
	return e.writeLog(visitor, info)
}

func (e *eacplogSvc) formatOrgPermAppStr(values interfaces.AppOrgPermValue) (str string) {
	var builder strings.Builder
	for k, v := range e.mapOrgPermStr {
		if values&k != 0 {
			if builder.Len() != 0 {
				fmt.Fprint(&builder, "/")
			}
			fmt.Fprint(&builder, loadString(v))
		}
	}
	return builder.String()
}

// OpUserExpiredDisabled 用户状态变更
func (e *eacplogSvc) OpUserExpiredDisabled(displayName, loginName string) (err error) {
	msg := fmt.Sprintf(loadString("IDS_DISABLE_EXPIRED_USER"), displayName, loginName)
	exMsg := loadString("IDS_DISABLE_EXPIRED_USER_EXMSG")
	info := &msgInfo{
		msg:      msg,
		exMsg:    exMsg,
		logType:  ltManage,
		logLevel: llWarn,
		opType:   int32(mtSet),
	}

	visitor := &interfaces.Visitor{
		ID: common.EacpLogSystemID,
	}
	return e.writeLog(visitor, info)
}

// OpUserNotLoginDisabled 用户长时间未登录自动禁用
func (e *eacplogSvc) OpUserNotLoginDisabled(displayName, loginName string) (err error) {
	msg := fmt.Sprintf(loadString("IDS_DISABLE_NOT_LOGIN_USER"), displayName, loginName)
	exMsg := loadString("IDS_DISABLE_NOT_LOGIN_USER_EXMSG")
	info := &msgInfo{
		msg:      msg,
		exMsg:    exMsg,
		logType:  ltManage,
		logLevel: llWarn,
		opType:   int32(mtSet),
	}

	visitor := &interfaces.Visitor{
		ID: common.EacpLogSystemID,
	}
	return e.writeLog(visitor, info)
}

// OpActiveUserInfoExported 活跃用户信息导出
func (e *eacplogSvc) OpActiveUserInfoExported(visitor *interfaces.Visitor, bYear bool) error {
	var msg string
	if bYear {
		msg = loadString("IDS_YEAR_ACTIVE_USER_INFO_EXPORTED_SUCCESS")
	} else {
		msg = loadString("IDS_MONTH_ACTIVE_USER_INFO_EXPORTED_SUCCESS")
	}
	info := &msgInfo{
		msg:      msg,
		logType:  ltManage,
		logLevel: llInfo,
		opType:   int32(mtExport),
	}
	return e.writeLog(visitor, info)
}
