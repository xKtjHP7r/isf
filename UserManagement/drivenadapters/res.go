// Package drivenadapters 翻译文件
package drivenadapters

import (
	"sync"

	"UserManagement/common"
)

var (
	resOnce sync.Once
	resMap  = make(map[string]string)
)

// loadString 获取字符串型语言项
func loadString(key string) string {
	resOnce.Do(initResMap)

	return resMap[key]
}

//nolint:funlen,dupl
func initResMap() {
	switch common.SvcConfig.Lang {
	case "zh_CN":
		resMap["IDS_GROUP_CREATED_SUCCESS"] = "新建 用户组 成功"
		resMap["IDS_GROUP_NAME"] = "用户组名称：%s"
		resMap["IDS_GROUP_NOTES"] = "备注：%s"
		resMap["IDS_GROUP_DELETED_SUCCESS"] = "删除 用户组 成功"
		resMap["IDS_GROUP_MODIFIED_SUCCESS"] = "编辑 用户组 成功"
		resMap["IDS_GROUP_MEMEBERS_ADDED_SUCCESS"] = "添加 用户组成员 成功"
		resMap["IDS_GROUP_MEMBERS"] = "用户组成员：%s"
		resMap["IDS_GROUP_MEMEBERS_DELETED_SUCCESS"] = "删除 用户组成员 成功"
		resMap["IDS_APP_REGISTER_SUCCESS"] = "新建应用账户“%s” 成功"
		resMap["IDS_DELETE_APP_SUCCESS"] = "删除应用账户“%s” 成功"
		resMap["IDS_UPDATE_APP_SUCCESS"] = "设置应用账户“%s” 成功"
		resMap["IDS_SET_ORG_PERM_APP_SUCCESS"] = "设置应用账户“%s”组织架构管理权限 成功"
		resMap["IDS_MODIFY_ORG_PERM_APP_SUCCESS"] = "编辑应用账户“%s”组织架构管理权限 成功"
		resMap["IDS_DELETE_ORG_PERM_APP_SUCCESS"] = "删除应用账户“%s”组织架构管理权限 成功"
		resMap["IDS_ORG_TYPE"] = "对象类型：%v"
		resMap["IDS_ORG_TYPE_USER"] = "用户"
		resMap["IDS_ORG_TYPE_DEPART"] = "部门"
		resMap["IDS_ORG_TYPE_GROUP"] = "用户组"
		resMap["IDS_ORG_PERM"] = "权限：%v"
		resMap["IDS_ORG_PERM_MODIFY"] = "编辑"
		resMap["IDS_ORG_PERM_READ"] = "显示"
		resMap["IDS_SET_DEFAULT_PWD"] = "重置 初始密码 成功"
		resMap["IDS_ORG_DEPART"] = "删除 组织 “%s” 及其子部门 成功"
		resMap["IDS_DEPART_DEPART"] = "删除 部门 “%s” 及其子部门 成功"
		resMap["IDS_SYSTEM"] = "系统"
		resMap["IDS_APP_TOKEN_GENERATED_SUCCESS"] = "重新生成应用账户“%s”的token成功"
		resMap["IDS_SET_CSF_LEVEL_ENUM_SUCCESS"] = "将用户密级从低到高自定义为“%s”成功"
		resMap["IDS_SET_CSF_LEVEL2_ENUM_SUCCESS"] = "将用户密级2从低到高自定义为“%s”成功"
		resMap["IDS_DISABLE_EXPIRED_USER"] = "禁用 用户“%s(%s)”成功"
		resMap["IDS_DISABLE_EXPIRED_USER_EXMSG"] = "禁用原因：用户账号过期"
		resMap["IDS_DISABLE_NOT_LOGIN_USER"] = "禁用 用户“%s(%s)”成功"
		resMap["IDS_DISABLE_NOT_LOGIN_USER_EXMSG"] = "禁用原因：用户长时间未登录"

	case "zh_TW":
		resMap["IDS_GROUP_CREATED_SUCCESS"] = "新建 用戶組 成功"
		resMap["IDS_GROUP_NAME"] = "用戶組名稱：%s"
		resMap["IDS_GROUP_NOTES"] = "備註：%s"
		resMap["IDS_GROUP_DELETED_SUCCESS"] = "刪除 用戶組 成功"
		resMap["IDS_GROUP_MODIFIED_SUCCESS"] = "編輯 用戶組 成功"
		resMap["IDS_GROUP_MEMEBERS_ADDED_SUCCESS"] = "添加 用戶組成員 成功"
		resMap["IDS_GROUP_MEMBERS"] = "用戶組成員：%s"
		resMap["IDS_GROUP_MEMEBERS_DELETED_SUCCESS"] = "刪除 用戶組成員 成功"
		resMap["IDS_APP_REGISTER_SUCCESS"] = "建立應用帳戶“%s” 成功"
		resMap["IDS_DELETE_APP_SUCCESS"] = "刪除應用帳戶“%s” 成功"
		resMap["IDS_UPDATE_APP_SUCCESS"] = "設定應用帳戶“%s” 成功"
		resMap["IDS_SET_ORG_PERM_APP_SUCCESS"] = "設定應用帳戶“%s”組織架構管理權限 成功"
		resMap["IDS_MODIFY_ORG_PERM_APP_SUCCESS"] = "編輯應用帳戶“%s”組織架構管理權限 成功"
		resMap["IDS_DELETE_ORG_PERM_APP_SUCCESS"] = "刪除應用帳戶“%s”組織架構管理權限 成功"
		resMap["IDS_ORG_TYPE"] = "物件類型：%v"
		resMap["IDS_ORG_TYPE_USER"] = "使用者"
		resMap["IDS_ORG_TYPE_DEPART"] = "部門"
		resMap["IDS_ORG_TYPE_GROUP"] = "使用者組"
		resMap["IDS_ORG_PERM"] = "權限：%v"
		resMap["IDS_ORG_PERM_MODIFY"] = "編輯"
		resMap["IDS_ORG_PERM_READ"] = "顯示"
		resMap["IDS_SET_DEFAULT_PWD"] = "重設 初始密碼 成功"
		resMap["IDS_ORG_DEPART"] = "刪除 組織 “%s” 及其子部門 成功"
		resMap["IDS_DEPART_DEPART"] = "刪除 部門 “%s” 及其 子部門 成功"
		resMap["IDS_SYSTEM"] = "系統"
		resMap["IDS_APP_TOKEN_GENERATED_SUCCESS"] = "重新生成應用帳戶“%s”的token成功"
		resMap["IDS_SET_CSF_LEVEL_ENUM_SUCCESS"] = "將用戶密級從低到高自訂為“%s”成功"
		resMap["IDS_SET_CSF_LEVEL2_ENUM_SUCCESS"] = "將用戶密級2從低到高自訂為“%s”成功"
		resMap["IDS_DISABLE_EXPIRED_USER"] = "停用 使用者“%s(%s)”成功"
		resMap["IDS_DISABLE_EXPIRED_USER_EXMSG"] = "停用原因：使用者帳戶已過期"
		resMap["IDS_DISABLE_NOT_LOGIN_USER"] = "停用 使用者“%s(%s)”成功"
		resMap["IDS_DISABLE_NOT_LOGIN_USER_EXMSG"] = "停用原因：使用者長時間未登入"

	case "en_US":
		resMap["IDS_GROUP_CREATED_SUCCESS"] = "create group successfully"
		resMap["IDS_GROUP_NAME"] = "group name：%s"
		resMap["IDS_GROUP_NOTES"] = "group notes：%s"
		resMap["IDS_GROUP_DELETED_SUCCESS"] = "delete group successfully"
		resMap["IDS_GROUP_MODIFIED_SUCCESS"] = "modify group successfully"
		resMap["IDS_GROUP_MEMEBERS_ADDED_SUCCESS"] = "add members to the group successfully"
		resMap["IDS_GROUP_MEMBERS"] = "group members：%s"
		resMap["IDS_GROUP_MEMEBERS_DELETED_SUCCESS"] = "delete members from the group successfully"
		resMap["IDS_APP_REGISTER_SUCCESS"] = "app Account \"%s\" is created successfully"
		resMap["IDS_DELETE_APP_SUCCESS"] = "App Account \"%s\" is deleted successfully"
		resMap["IDS_UPDATE_APP_SUCCESS"] = "App Account \"%s\" is set successfully"
		resMap["IDS_SET_ORG_PERM_APP_SUCCESS"] = "%s's Permission for Org Management is set successfully"
		resMap["IDS_MODIFY_ORG_PERM_APP_SUCCESS"] = "%s's Permission for Org Management is edited successfully"
		resMap["IDS_DELETE_ORG_PERM_APP_SUCCESS"] = "%s's Permission for Org Management is deleted successfully"
		resMap["IDS_ORG_TYPE"] = "Object: %v"
		resMap["IDS_ORG_TYPE_USER"] = "User"
		resMap["IDS_ORG_TYPE_DEPART"] = "Department"
		resMap["IDS_ORG_TYPE_GROUP"] = "User Group"
		resMap["IDS_ORG_PERM"] = "Permission: %v"
		resMap["IDS_ORG_PERM_MODIFY"] = "Edit"
		resMap["IDS_ORG_PERM_READ"] = "Display"
		resMap["IDS_SET_DEFAULT_PWD"] = "The initial password has been reset successfully"
		resMap["IDS_ORG_DEPART"] = "Succeeded to delete organization \"%s\" and its sub-departments"
		resMap["IDS_DEPART_DEPART"] = "Succeeded to delete department \"%s\" and its sub-departments"
		resMap["IDS_SYSTEM"] = "System"
		resMap["IDS_APP_TOKEN_GENERATED_SUCCESS"] = "The token for application account \"%s\" has been regenerated successfully."
		resMap["IDS_SET_CSF_LEVEL_ENUM_SUCCESS"] = "Customizing user security levels from lowest to highest \"%s\" successful."
		resMap["IDS_SET_CSF_LEVEL2_ENUM_SUCCESS"] = "Customizing user security levels 2 from lowest to highest \"%s\" successful."
		resMap["IDS_DISABLE_EXPIRED_USER"] = "Disable User\"%s(%s)\"Successfully."
		resMap["IDS_DISABLE_EXPIRED_USER_EXMSG"] = "Disable reason:The user account has expired."
		resMap["IDS_DISABLE_NOT_LOGIN_USER"] = "Disable User\"%s(%s)\"Successfully."
		resMap["IDS_DISABLE_NOT_LOGIN_USER_EXMSG"] = "Disable reason:The user has not logged in for a long time."

	default:
		common.NewLogger().Fatalln("service language not set")
	}
}
