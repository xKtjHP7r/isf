import { apiUpdateActivityStatus } from '../../timeout'
import { ShareMgnt, ShareMgntSingle } from '../thrift';

/**
 * 移动用户到部门
 * @param userIds
 * @param srcDepartId
 * @param destDepartId
 */
export function moveUserToDepartment(userIds: Array<string>, srcDepartId: string, destDepartId: string): Promise<Array<string>> {
    return ShareMgnt('Usrm_MoveUserToDepartment', [userIds, srcDepartId, destDepartId]);
}

/**
 * 编辑用户对象存储
 * @param userId
 * @param ossId
 */
export function editUserOss(userId: string, ossId: string): Promise<void> {
    return ShareMgnt('Usrm_EditUserOSS', [userId, ossId]);
}

/**
 *  移除用户
 * @param userIds
 * @param departmentId
 */
export function removeUserFromDepartment(userIds: Array<string>, departmentId: string): Promise<Array<string>> {
    return ShareMgnt('Usrm_RomoveUserFromDepartment', [userIds, departmentId]);
}

/**
 * 启用/者禁用用户
 * @param userId 用户id
 * @param status  true ： 启用， false：禁用
 */
export function setUserStatus(userId: string, status: boolean): Promise<void> {
    return ShareMgnt('Usrm_SetUserStatus', [userId, status])
}

/**
 * 获取部门的父部门
 * @param depart_id 当前部门id
 */
export function getDepartmentById(depart_id: string): Promise<Core.ShareMgnt.ncTUsrmDepartmentInfo> {
    return ShareMgnt('Usrm_GetDepartmentById', [depart_id])
}

/**
 * 获取部门/组织信息
 * @param depart_id 当前部门id
 */
export function getOrgDepartmentById(depart_id: string): Promise<Core.ShareMgnt.ncTUsrmDepartmentInfo> {
    return ShareMgnt('Usrm_GetOrgDepartmentById', [depart_id])
}

/**
 * 设置用户冻结状态
 * @param userid 用户id
 * @param freezeStatus true: 冻结 false: 解冻
 */
export function setUserFreezeStatus(userid: string, freezeStatus: boolean): Promise<void> {
    return ShareMgnt('Usrm_SetUserFreezeStatus', [userid, freezeStatus])
}

/**
 * 获取登录验证码配置信息
 */
export const getVcodeConfig: Core.ShareMgnt.GetVcodeConfig = function () {
    return ShareMgnt('Usrm_GetVcodeConfig')
}

/**
 * 设置登录验证码配置
 */
export const setVcodeConfig: Core.ShareMgnt.SetVcodeConfig = function ([vcodeconfig]) {
    return ShareMgnt('Usrm_SetVcodeConfig', [vcodeconfig])
}

/**
 * 验证码生成函数接口
 */
export const createVcodeInfo: Core.ShareMgnt.CreateVcodeInfo = function ([uuid, vCodeType = 1]: [string, 1 | 2]) {
    return ShareMgnt('Usrm_CreateVcodeInfo', [uuid, vCodeType])
}

/**
 * 获取日志导出加密开关状态
 */
export const getExportWithPassWordStatus: Core.ShareMgnt.GetExportWithPassWordStatus = function () {
    return ShareMgnt('GetExportWithPassWordStatus')
}

/**
 * 添加导出历史日志文件任务
 */
export const exportHistoryLog: Core.ShareMgnt.ExportHistoryLog = function ([id, validSeconds, pwd]) {
    return ShareMgnt('ExportHistoryLog', [id, validSeconds, pwd])
}

/**
 * 获取日志文件信息
 */
export const getCompressFileInfo: Core.ShareMgnt.GetCompressFileInfo = function ([taskId]) {
    return ShareMgnt('GetCompressFileInfo', [taskId])
}

/**
 * 获取日志打包进度
 */
export const getGenCompressFileStatus: Core.ShareMgnt.GetCompressFileInfo = function ([taskId]) {
    apiUpdateActivityStatus()
    return ShareMgnt('GetGenCompressFileStatus', [taskId])
}

/*
 * 获取管理员邮箱列表设置
 */
export const SMTPGetAdminMailList: Core.ShareMgnt.SMTPGetAdminMailList = function ([adminId]) {
    return ShareMgnt('SMTP_GetAdminMailList', [adminId])
}

/**
 * 测试邮箱是否正确
 */
export const SMTPReceiverTest: Core.ShareMgnt.SMTPReceiverTest = function ([mails]) {
    return ShareMgnt('SMTP_ReceiverTest', [mails])
}

/**
 * 验证控制台密码
 */
export const checkConsoleUserPassword: Core.ShareMgnt.CheckConsoleUserPassword = function ([userName, password, authenType, ncTUserLoginOption]) {
    return ShareMgnt('Usrm_CheckConsoleUserPassword', [userName, password, authenType, { ncTUserLoginOption: ncTUserLoginOption }])
}

/**
 * 编辑内置管理员账号
 */
export const editAdminAccount: Core.ShareMgnt.EditAdminAccount = function ([adminId, Account]) {
    return ShareMgnt('Usrm_EditAdminAccount', [adminId, Account])
}

/**
 * 设置管理员邮箱
 */
export const setAdminMailList: Core.ShareMgnt.SetAdminMailList = function ([adminId, mailList]) {
    return ShareMgnt('SMTP_SetAdminMailList', [adminId, mailList])
}

/**
 * 获取组织用户列表
 */
export const getDepartmentUser: Core.ShareMgnt.GetDepartmentUser = function ([userid, start, end]) {
    apiUpdateActivityStatus()
    return ShareMgnt('Usrm_GetDepartmentOfUsers', [userid, start, end])
}

/**
 * 获取部门用户数量
 */
export const getDepartmentOfUsersCount: Core.ShareMgnt.GetDepartmentOfUsersCount = function ([depId]) {
    return ShareMgnt('Usrm_GetDepartmentOfUsersCount', [depId])
}

/**
 * 获取部门列表
 */
export const getSubDepartments: Core.ShareMgnt.GetSubDepartments = function ([usid]) {
    return ShareMgnt('Usrm_GetSubDepartments', [usid])
}

/**
 *
 * 检查用户是否属于某个部门及其子部门
 */
export const checkUserInDepart: Core.ShareMgnt.CheckUserInDepart = function ([userId, departId]) {
    return ShareMgnt('Usrm_CheckUserInDepart', [userId, departId])
}

/**
 * 设置用户权重
 */
export const editUserPriority: Core.ShareMgnt.EditUserPriority = function ([userid, priority]) {
    return ShareMgnt('Usrm_EditUserPriority', [userid, priority])
}

/**
 * 获取短信服务器配置
 */
export const getConfig: Core.ShareMgnt.SMSGetConfig = function ([]) { // eslint-disable-line
    return ShareMgnt('SMS_GetConfig')
}

/**
 * 设置短信服务器配置
 */
export const setConfig: Core.ShareMgnt.SMSSetConfig = function ([config]) {
    return ShareMgnt('SMS_SetConfig', [config])
}

/**
 * 测试短信服务器
 */
export const test: Core.ShareMgnt.SMSTest = function ([config]) {
    return ShareMgnt('SMS_Test', [config])
}

/**
 * 获取月度活跃报表信息
 */
export const getActiveReportMonth: Core.ShareMgnt.GetActiveReportMonth = function ([inquireDate]) {
    return ShareMgnt('GetActiveReportMonth', [inquireDate]);
}

/**
 * 获取年度活跃报表信息
 */
export const getActiveReportYear: Core.ShareMgnt.GetActiveReportYear = function ([inquireDate]) {
    return ShareMgnt('GetActiveReportYear', [inquireDate])
}

/**
 * 创建月度活跃报表导出任务
 */
export const exportActiveReportMonth: Core.ShareMgnt.ExportActiveReportMonth = function ([name, inquireDate]) {
    return ShareMgnt('ExportActiveReportMonth', [name, inquireDate]);
}

/**
 * 创建年度活跃报表导出任务
 */
export const exportActiveReportYear: Core.ShareMgnt.ExportActiveReportYear = function ([name, inquireDate]) {
    return ShareMgnt('ExportActiveReportYear', [name, inquireDate])
}

/**
 * 获取存在活跃统计的最早时间
 */
export const opermGetEarliestTime: Core.ShareMgnt.OpermGetEarliestTime = function () {
    return ShareMgnt('Operm_GetEarliestTime');
}

/**
 *  获取生成活跃报表状态
 */
export const getGenActiveReportStatus: Core.ShareMgnt.GetGenActiveReportStatus = function ([taskId]) {
    apiUpdateActivityStatus()
    return ShareMgnt('GetGenActiveReportStatus', [taskId])
}

/**
 * 获取指定部门下所有部门负责人信息
 */
export const usrmGetDepartResponsiblePerson: Core.ShareMgnt.UsrmGetDepartResponsiblePerson = function ([departId]) {
    return ShareMgnt('Usrm_GetDepartResponsiblePerson', [departId])
}

/**
 * 获取所有域
 */
export const usrmGetAllDomains: Core.ShareMgnt.UsrmGetAllDomains = function () {
    return ShareMgnt('Usrm_GetAllDomains')
}

/**
 * 展开域中节点
 */
export const usrmExpandDomainNode: Core.ShareMgnt.UsrmExpandDomainNode = function ([domain, pathName]) {
    return ShareMgnt('Usrm_ExpandDomainNode', [domain, pathName])
}

/**
 * 搜索域用户或部门
 */
export const usrmSearchDomainInfoByName: Core.ShareMgnt.UsrmSearchDomainInfoByName = function ([domainId, name, start, limit]) {
    return ShareMgnt('Usrm_SearchDomainInfoByName', [domainId, name, start, limit])
}

/**
 * 获取第三方组织根节点
 */
export const usrmGetThirdPartyRootNode: Core.ShareMgnt.UsrmGetThirdPartyRootNode = function ([userid]) {
    return ShareMgntSingle('Usrm_GetThirdPartyRootNode', [userid])
}

/**
 * 展开第三方节点
 */
export const usrmExpandThirdPartyNode: Core.ShareMgnt.UsrmExpandThirdPartyNode = function ([thirdId]) {
    return ShareMgntSingle('Usrm_ExpandThirdPartyNode', [thirdId])
}

/**
 * 导入第三方组织结构和用户
 */
export const usrmImportThirdPartyOUs: Core.ShareMgnt.UsrmImportThirdPartyOUs = function ([ous, users, option, responsiblePersonId]) {
    return ShareMgntSingle('Usrm_ImportThirdPartyOUs', [ous, users, option, responsiblePersonId], { timeout: 60 * 5000 })
}

/**
 * 清除第三方导入进度
 */
export const usrmClearThirdImportProgress: Core.ShareMgnt.UsrmClearImportProgress = function () {
    return ShareMgntSingle('Usrm_ClearImportProgress');
}

/**
 * 获取第三方导入进度
 */
export const usrmGetThirdImportProgress: Core.ShareMgnt.UsrmGetImportProgress = function () {
    apiUpdateActivityStatus()
    return ShareMgntSingle('Usrm_GetImportProgress')
}

/**
 * 清除导入进度
 */
export const usrmClearImportProgress: Core.ShareMgnt.UsrmClearImportProgress = function () {
    return ShareMgnt('Usrm_ClearImportProgress');
}

/**
 * 获取导入进度
 */
export const usrmGetImportProgress: Core.ShareMgnt.UsrmGetImportProgress = function () {
    apiUpdateActivityStatus()
    return ShareMgnt('Usrm_GetImportProgress')
}

/**
 * 导入域用户
 */
export const usrmImportDomainUsers: Core.ShareMgnt.UsrmImportDomainUsers = function ([ncTUsrmImportContent, ncTUsrmImportOption, responsiblePersonId]) {
    return ShareMgnt('Usrm_ImportDomainUsers', [ncTUsrmImportContent, ncTUsrmImportOption, responsiblePersonId])
}

/**
 * 导入域部门
 */
export const usrmImportDomainOUs: Core.ShareMgnt.UsrmImportDomainOUs = function ([ncTUsrmImportContent, ncTUsrmImportOption, responsiblePersonId]) {
    return ShareMgnt('Usrm_ImportDomainOUs', [ncTUsrmImportContent, ncTUsrmImportOption, responsiblePersonId])
}

/**
 * 获取第三方应用配置
 */
export const getThirdPartyAppConfig: Core.ShareMgnt.GetThirdPartyAppConfig = function (type) {
    return ShareMgnt('GetThirdPartyAppConfig', [type])
}

/**
 * 获取导入进度
 */
export const addThirdPartyAppConfig: Core.ShareMgnt.AddThirdPartyAppConfig = function (config) {
    apiUpdateActivityStatus()
    return ShareMgntSingle('AddThirdPartyAppConfig', [config])
}

/**
 * 设置第三方应用配置
 */
export const setThirdPartyAppConfig: Core.ShareMgnt.SetThirdPartyAppConfig = function (config) {
    return ShareMgntSingle('SetThirdPartyAppConfig', [config])
}

/**
 * 删除第三方应用配置
 */
export const deleteThirdPartyAppConfig: Core.ShareMgnt.DeleteThirdPartyAppConfig = function (indexId) {
    return ShareMgntSingle('DeleteThirdPartyAppConfig', [indexId])
}

/*
 * 添加角色
 */
export const addUserRolem: Core.ShareMgnt.AddUserRolem = function ([ncTRoleInfo]) {
    return ShareMgnt('UsrRolem_Add', [{ ncTRoleInfo: ncTRoleInfo }])
}

/**
 * 获取角色
 */
export const getUserRolem: Core.ShareMgnt.GetUserRolem = function ([userId]) {
    return ShareMgnt('UsrRolem_Get', [userId])
}

/**
 * 编辑角色
 */
export const editUserRolem: Core.ShareMgnt.EditUserRolem = function ([userId, ncTRoleInfo]) {
    return ShareMgnt('UsrRolem_Edit', [userId, { ncTRoleInfo: ncTRoleInfo }])
}

/**
 *
 * 删除角色
 */
export const deleteUserRolem: Core.ShareMgnt.DeleteUserRolem = function ([userId, roleId]) {
    return ShareMgnt('UsrRolem_Delete', [userId, roleId])
}

/**
 * 设置成员包含添加和编辑成员
 */
export const setUserRolemMember: Core.ShareMgnt.SetUserRolemMember = function ([userId, roleId, ncTRoleMemberInfo]) {
    return ShareMgnt('UsrRolem_SetMember', [userId, roleId, { ncTRoleMemberInfo: ncTRoleMemberInfo }])
}

/**
 * 获取成员列表
 */
export const getUserRolemMember: Core.ShareMgnt.GetUserRolemMember = function ([userId, roleId]) {
    return ShareMgnt('UsrRolem_GetMember', [userId, roleId])
}

/**
 * 删除成员
 */
export const deleteUserRolemMember: Core.ShareMgnt.DeleteUserRolemMember = function ([userId, roleId, memberId]) {
    return ShareMgnt('UsrRolem_DeleteMember', [userId, roleId, memberId])
}

/**
 * 在所选角色中根据成员id获取详细信息
 */
export const getRoleMemberDetail: Core.ShareMgnt.GetUserRole = function ([userId, roleId, memberId]) {
    return ShareMgnt('UsrRolem_GetMemberDetail', [userId, roleId, memberId])
}

/**
 * 检查当前选择成员时候已存在
 */
export const checkMemberExist: Core.ShareMgnt.CheckMemberExist = function ([roleId, memberId]) {
    return ShareMgnt('UsrRolem_CheckMemberExist', [roleId, memberId])
}

/**
 * 根据用户id检查用户状态，是否启用/密码是否过期
 */
export const checkUserStatus: Core.ShareMgnt.CheckMemberExist = function ([userId]) {
    return ShareMgnt('Usrm_CheckUserStatus', [userId])
}

/**
 * 获取杀毒选件状态
 */
export const getAntivirusOptionAuthStatus = function () {
    return ShareMgnt('GetAntivirusOptionAuthStatus', [])
}

/**
 * 获取用户信息
 */
export const getUserInfo: Core.ShareMgnt.ncTUsrmGetUserInfo = function ([userId]) {
    return ShareMgnt('Usrm_GetUserInfo', [userId])
}

/**
 * 设置用户有效期
 */
export const setUserExpireTime: Core.ShareMgnt.SetUserExpireTime = function ([userId, expireTime]) {
    return ShareMgnt('Usrm_SetUserExpireTime', [userId, expireTime])
}

/**
 * 获取双因子验证类型的状态
 */
export const getCustomConfigOfString: Core.ShareMgnt.GetCustomConfigOfString = function ([configName]) {
    return ShareMgnt('GetCustomConfigOfString', [configName])
}

/**
 * 设置双因子验证类型的状态
 */
export const setCustomConfigOfString: Core.ShareMgnt.SetCustomConfigOfString = function ([configName, configValue]) {
    return ShareMgnt('SetCustomConfigOfString', [configName, configValue])
}

/**
* 获取第三方预览工具配置信息
*/
export const getThirdPartyToolConfig: Core.ShareMgnt.GetThirdPartyToolConfig = function ([thirdPartyToolId]) {
    return ShareMgnt('GetThirdPartyToolConfig', [thirdPartyToolId])
}

/**
* 设置第三方预览工具配置信息
*/
export const setThirdPartyToolConfig: Core.ShareMgnt.SetThirdPartyToolConfig = function ([thirdPartyToolConfig]) {
    return ShareMgnt('SetThirdPartyToolConfig', [{ ncTThirdPartyToolConfig: thirdPartyToolConfig }])
}

/**
* 测试第三方预览工具配置信息
*/
export const testThirdPartyToolConfig: Core.ShareMgnt.TestThirdPartyToolConfig = function ([url]) {
    return ShareMgnt('TestThirdPartyToolConfig', [url])
}

/**
 * 获取备用域信息
 */
export const usrmGetFailoverDomains: Core.ShareMgnt.UsrmGetFailoverDomains = function ([parentDomainId]) {
    return ShareMgnt('Usrm_GetFailoverDomains', [parentDomainId])
}

/**
 * 检查备用域是否可用（不保存到数据库）
 */
export const usrmCheckFailoverDomainAvailable: Core.ShareMgnt.UsrmCheckFailoverDomainAvailable = function ([failoverDomainInfos]) {
    return ShareMgnt('Usrm_CheckFailoverDomainAvailable', [failoverDomainInfos])
}

/**
 * 编辑备用域（使用参数覆盖的方式，包括增、删、改；parentDomainId为首选域的id
 */
export const usrmEditFailoverDomains: Core.ShareMgnt.UsrmEditFailoverDomains = function ([failoverDomainInfos, parentDomainId]) {
    return ShareMgnt('Usrm_EditFailoverDomains', [failoverDomainInfos, parentDomainId])
}

/**
 * 导出excel组织信息
 */
export const usrmExportBatchUsers: Core.ShareMgnt.UsrmExportBatchUsers = function ([departmentIds, userid]) {
    return ShareMgnt('Usrm_ExportBatchUsers', [departmentIds, userid])
}

/**
 * 导出组织信息的excel表是否可以下载
 */
export const usrmGetExportBatchUsersTaskStatus: Core.ShareMgnt.UsrmGetExportBatchUsersTaskStatus = function ([taskid]) {
    return ShareMgnt('Usrm_GetExportBatchUsersTaskStatus', [taskid])
}

/**
 * 下载导出组织信息的excel表
 */
export const usrmDownloadBatchUsers: Core.ShareMgnt.UsrmDownloadBatchUsers = function ([taskid]) {
    return ShareMgnt('Usrm_DownloadBatchUsers', [taskid])
}

/**
 * 导入组织信息错误数据获取
 */
export const usrmGetErrorInfos: Core.ShareMgnt.UsrmGetErrorInfos = function ([start, limit]) {
    return ShareMgnt('Usrm_GetErrorInfos', [start, limit])
}

/**
 * 下载导入失败的记录
 */
export const downloadImportFailedUsers: Core.ShareMgnt.UsrmDownloadImportFailedUsers = function () {
    return ShareMgnt('Usrm_DownloadImportFailedUsers')
}

/**
 * 获取组织信息导入进度
 */
export const getProgress: Core.ShareMgnt.UsrmGetProgress = function () {
    apiUpdateActivityStatus()
    return ShareMgnt('Usrm_GetProgress')
}

/**
 * 获取三权分立状态
 */
export const getTriSystemStatus: Core.ShareMgnt.GetTriSystemStatus = function () {
    return ShareMgnt('Usrm_GetTriSystemStatus')
}

/*
 * 获取SMTP配置信息
 */
export const getSMTPConfig: Core.ShareMgnt.GetSMTPConfig = function () {
    return ShareMgnt('SMTP_GetConfig')
}

/**
 * 测试SMTP服务器
 */
export const testSMTPServer: Core.ShareMgnt.TestSMTPServer = function ([SMTPConfigInfo]) {
    return ShareMgnt('SMTP_ServerTest', [{ ncTSmtpSrvConf: SMTPConfigInfo }], { timeout: 0 })
}

/**
 * 设置SMTP服务器
 */
export const setSMTPConfig: Core.ShareMgnt.SetSMTPConfig = function ([SMTPConfigInfo]) {
    return ShareMgnt('SMTP_SetConfig', [{ ncTSmtpSrvConf: SMTPConfigInfo }])
}

/**
 * 获取密码管控信息
 * @param userid
 */
export const getPwdControl: Core.ShareMgnt.GetPwdControl = function ([userId]) {
    return ShareMgnt('Usrm_GetPwdControl', [userId])
}

/**
 * 获取密码管控配置
 */
export const getPwdConfig: Core.ShareMgnt.GetPwdConfig = function () {
    return ShareMgnt('Usrm_GetPasswordConfig')
}

/**
 * 设置密码管控
 */
export const setPwdControl: Core.ShareMgnt.SetPwdControl = function ([userid, pwdControlConfig]) {
    return ShareMgnt('Usrm_SetPwdControl', [userid, { ncTUsrmPwdControlConfig: pwdControlConfig }])
}
/**
 * 新建用户
 */
export const addUser: Core.ShareMgnt.AddUser = function ([data, userid]) {
    return ShareMgnt('Usrm_AddUser', [data, userid])
}

/**
 * 编辑用户
 */
export const editUser: Core.ShareMgnt.EditUser = function ([data, userid]) {
    return ShareMgnt('Usrm_EditUser', [data, userid])
}

/**
 * 获取当前密级
 */
export const getSysCsfLevel: Core.ShareMgnt.GetSysCsfLevel = function () {
    return ShareMgnt('GetSysCSFLevel')
}

/**
 * 编辑实名共享的策略信息
 */
export const editPermShareInfo: Core.ShareMgnt.EditPermShareInfo = function ([sharerData]) {
    return ShareMgnt('Usrm_EditPermShareInfo', [sharerData])
}

/**
 * 添加实名共享的策略信息
 */
export const addPermShareInfo: Core.ShareMgnt.AddPermShareInfo = function ([sharerData]) {
    return ShareMgnt('Usrm_AddPermShareInfo', [sharerData])
}

/**
 * 设置实名用户共享限制开启禁用状态
 */
export const getSystemPermShareStatus: Core.ShareMgnt.GetSystemPermShareStatus = function () {
    return ShareMgnt('Usrm_GetSystemPermShareStatus')
}

/**
 * 分页获取实名共享的策略信息
 */
export const getPermShareInfoByPage: Core.ShareMgnt.GetPermShareInfoByPage = function ([start, defaultPageSize]) {
    return ShareMgnt('Usrm_GetPermShareInfoByPage', [start, defaultPageSize])
}

/**
 * 获取实名共享的策略信息的总数
 */
export const getPermShareInfoCnt: Core.ShareMgnt.GetPermShareInfoCnt = function () {
    return ShareMgnt('Usrm_GetPermShareInfoCnt')
}

/**
 * 搜索实名共享的策略信息
 */
export const searchPermShareInfo: Core.ShareMgnt.SearchPermShareInfo = function ([start, limit, searchKey]) {
    return ShareMgnt('Usrm_SearchPermShareInfo', [start, limit, searchKey])
}

/**
 * 设置所有用户对其直属部门/其直属组织的共享状态
 */
export const setPermShareInfoStatus: Core.ShareMgnt.SetPermShareInfoStatus = function ([strategyId, checked]) {
    return ShareMgnt('Usrm_SetPermShareInfoStatus', [strategyId, checked])
}

/**
 * 设置实名用户共享限制开启禁用状态
 */
export const setSystemPermShareStatus: Core.ShareMgnt.SetSystemPermShareStatus = function ([isShareStrategyEabled]) {
    return ShareMgnt('Usrm_SetSystemPermShareStatus', [isShareStrategyEabled])
}

/**
 * 删除实名共享的策略信息
 */
export const deletePermShareInfo: Core.ShareMgnt.DeletePermShareInfo = function ([id]) {
    return ShareMgnt('Usrm_DeletePermShareInfo', [id])
}

/**
 * 获取匿名共享的状态
 */
export const getSystemLinkShareStatus: Core.ShareMgnt.GetSystemLinkShareStatus = function () {
    return ShareMgnt('Usrm_GetSystemLinkShareStatus')
}

/**
 * 分页获取匿名共享的策略信息
 */
export const getLinkShareInfoByPage: Core.ShareMgnt.GetLinkShareInfoByPage = function ([start, defaultPageSize]) {
    return ShareMgnt('Usrm_GetLinkShareInfoByPage', [start, defaultPageSize])
}

/**
 * 获取匿名共享的策略信息总数
 */
export const getLinkShareInfoCnt: Core.ShareMgnt.GetLinkShareInfoCnt = function () {
    return ShareMgnt('Usrm_GetLinkShareInfoCnt')
}

/**
 * 搜索匿名共享的策略信息
 */
export const searchLinkShareInfo: Core.ShareMgnt.SearchLinkShareInfo = function ([start, limit, searchKey]) {
    return ShareMgnt('Usrm_SearchLinkShareInfo', [start, limit, searchKey])
}

/**
 * 设置匿名共享限制开启关闭状态
 */
export const setSystemLinkShareStatus: Core.ShareMgnt.SetSystemLinkShareStatus = function ([isScopeEabled]) {
    return ShareMgnt('Usrm_SetSystemLinkShareStatus', [isScopeEabled])
}

/**
 * 添加匿名共享的策略信息
 */
export const addLinkShareInfo: Core.ShareMgnt.AddLinkShareInfo = function ([sharer]) {
    return ShareMgnt('Usrm_AddLinkShareInfo', [sharer])
}

/**
 * 删除匿名共享的策略信息
 */
export const deleteLinkShareInfo: Core.ShareMgnt.DeleteLinkShareInfo = function ([sharerId]) {
    return ShareMgnt('Usrm_DeleteLinkShareInfo', [sharerId])
}

/**
 * 设置开启/关闭实名或者匿名共享个人文档库状态
 */
export const setShareDocStatus: Core.ShareMgnt.SetShareDocStatus = function ([userType, shareType, detail]) {
    return ShareMgnt('SetShareDocStatus', [userType, shareType, detail])
}

/**
 * 获取实名或者匿名共享个人文档库状态
 */
export const getShareDocStatus: Core.ShareMgnt.GetShareDocStatus = function ([userType, shareType]) {
    return ShareMgnt('GetShareDocStatus', [userType, shareType])
}

/**
 * 批量根据部门ID(组织ID)获取部门（组织）父路经
 */
export const getDepParentPathById: Core.ShareMgnt.GetDepParentPathById = function (departIds) {
    return ShareMgnt('Usrm_GetDepartmentParentPath', [departIds])
}

/**
 * 移动部门
 */
export const moveDepartment: Core.ShareMgnt.MoveDepartment = function ([srcDepId, destDepId]) {
    return ShareMgnt('Usrm_MoveDepartment', [srcDepId, destDepId])
}

/**
 * 编辑部门的对象存储
 */
export const editDepartOSS: Core.ShareMgnt.EditDepartOSS = function ([depId, ossId]) {
    return ShareMgnt('Usrm_EditDepartOSS', [depId, ossId])
}

/**
 * 添加用户至部门
 */
export const addUsersToDep: Core.ShareMgnt.AddUsersToDep = function ([userIds, departmentId]) {
    return ShareMgnt('Usrm_AddUserToDepartment', [userIds, departmentId])
}

/**
 * 获取当前实时在线用户数
 */
export const opermGetCurrentOnlineUser: Core.ShareMgnt.OpermGetCurrentOnlineUser = function () {
    return ShareMgnt('Operm_GetCurrentOnlineUser')
}

/**
 * 获取当日最高上线用户数
 */
export const opermGetMaxOnlineUserDay: Core.ShareMgnt.OpermGetMaxOnlineUserDay = function ([date]) {
    return ShareMgnt('Operm_GetMaxOnlineUserDay', [date])
}

/**
 * 设置文档标签策略
 */
export const setCustomConfigOfInt64: Core.ShareMgnt.SetCustomConfigOfInt64 = function ([key, value]) {
    return ShareMgnt('SetCustomConfigOfInt64', [key, value])
}

/**
 * 获取文档标签策略
 */
export const getCustomConfigOfInt64: Core.ShareMgnt.GetCustomConfigOfInt64 = function ([key]) {
    return ShareMgnt('GetCustomConfigOfInt64', [key])
}

/**
* 新建部门
 */
export const addDepartment: Core.ShareMgnt.Usrm_AddDepartment = function ([addParmas]) {
    return ShareMgnt('Usrm_AddDepartment', [addParmas])
}

/**
 * 编辑部门
 */
export const editDepartment: Core.ShareMgnt.Usrm_EditDepartment = function ([editParma]) {
    return ShareMgnt('Usrm_EditDepartment', [editParma])
}

/**
 * 新建组织
 */
export const createOrganization: Core.ShareMgnt.UsrmCreateOrganization = function ([data]) {
    return ShareMgnt('Usrm_CreateOrganization', [data])
}

/**
 * 编辑组织
 */
export const editOrganization: Core.ShareMgnt.Usrm_EditOrganization = function ([editParma]) {
    return ShareMgnt('Usrm_EditOrganization', [editParma])
}

/**
 * 获取是否开启匿名共享审核机制
 */
export const getCustomConfigOfBool: Core.ShareMgnt.GetCustomConfigOfBool = function ([key]) {
    return ShareMgnt('GetCustomConfigOfBool', [key])
}

/**
 * 启用或禁用匿名共享审核机制
 */
export const setCustomConfigOfBool: Core.ShareMgnt.SetCustomConfigOfBool = function ([key, enable]) {
    return ShareMgnt('SetCustomConfigOfBool', [key, enable])
}

/**
 * 获取第三方标密系统配置
 */
export const getThirdCSFSysConfig: Core.ShareMgnt.GetThirdCSFSysConfig = function () {
    return ShareMgnt('GetThirdCSFSysConfig')
}

/**
 * 设置域同步状态,-1:域同步关闭,0：域正向同步开启，1：域反向同步开启
 */
export const setDomainSyncStatus: Core.ShareMgnt.SetDomainSyncStatus = function ([id, status]) {
    return ShareMgnt('Usrm_SetDomainSyncStatus', [id, status])
}

/**
 * 第三方同步(如果为域同步，则appId为域id; autoSync: True-定期同步， False-单次同步)
 */
export const startSync: Core.ShareMgnt.StartSync = function ([id, status]) {
    return ShareMgntSingle('SYNC_StartSync', [id, status])
}

/**
 * 获取 域配置信息
 */
export const getDomainConfig: Core.ShareMgnt.GetDomainConfig = function ([id]) {
    return ShareMgnt('Usrm_GetDomainConfig', [id])
}

/**
 * 开启或者关闭域控
 */
export const setDomainStatus: Core.ShareMgnt.SetDomainStatus = function ([id, status]) {
    return ShareMgnt('Usrm_SetDomainStatus', [id, status])
}

/**
 * 增加域控
 */
export const addDomain: Core.ShareMgnt.AddDomain = function ([ncTUsrmDomainInfo]) {
    return ShareMgnt('Usrm_AddDomain', [ncTUsrmDomainInfo])
}

/**
 * 编辑域控
 */
export const editDomain: Core.ShareMgnt.EditDomain = function ([ncTUsrmDomainInfo]) {
    return ShareMgnt('Usrm_EditDomain', [ncTUsrmDomainInfo])
}

/**
 * 删除域
 */
export const deleteDomain: Core.ShareMgnt.DeleteDomain = function ([id]) {
    return ShareMgntSingle('Usrm_DeleteDomain', [id])
}

/**
 *  根据域id获取域控信息
 */
export const getDomainById: Core.ShareMgnt.GetDomainById = function ([id]) {
    return ShareMgnt('Usrm_GetDomainById', [id])
}

/**
 * 设置域配置信息
 */
export const setDomainConfig: Core.ShareMgnt.SetDomainConfig = function ([id, ncTUsrmDomainConfig]) {
    return ShareMgntSingle('Usrm_SetDomainConfig', [id, ncTUsrmDomainConfig])
}

/**
 * 获取域关键字配置信息
 */
export const getDomainKeyConfig: Core.ShareMgnt.GetDomainKeyConfig = function ([domainId]) {
    return ShareMgnt('Usrm_GetDomainKeyConfig', [domainId])
}

/**
* 设置域关键字配置信息
*/
export const setDomainKeyConfig: Core.ShareMgnt.SetDomainKeyConfig = function ([domainId, keyConfig]) {
    return ShareMgntSingle('Usrm_SetDomainKeyConfig', [domainId, keyConfig])
}
/**
* 获取第三方认证管理信息
*/
export const getThirdPartyAuth: Core.ShareMgnt.GetThirdPartyAuth = function () {
    return ShareMgnt('Usrm_GetThirdPartyAuth')
}

/**
 * 获取冻结状态
 */
export const getFreezeStatus: Core.ShareMgnt.GetFreezeStatus = function () {
    return ShareMgnt('Usrm_GetFreezeStatus')
}

/**
 * 在部门中根据key搜索用户，并返回搜索的用户总数
 */
export const countSearchDepartmentOfUsers: Core.ShareMgnt.CountSearchDepartmentOfUsers = function ([departmentId, searchKey]) {
    return ShareMgnt('Usrm_CountSearchDepartmentOfUsers', [departmentId, searchKey])
}

/**
 * 在部门中根据key搜索用户，并返回分页数据
 */
export const searchDepartmentOfUsers: Core.ShareMgnt.SearchDepartmentOfUsers = function ([departmentId, searchKey, start, limit]) {
    return ShareMgnt('Usrm_SearchDepartmentOfUsers', [departmentId, searchKey, start, limit])
}

/**
 * 分页获取用户信息
 */
export const getDepartmentOfUsers: Core.ShareMgnt.GetDepartmentOfUsers = function ([departmentId, start, limit]) {
    return ShareMgnt('Usrm_GetDepartmentOfUsers', [departmentId, start, limit])
}

/**
 * 获取所有用户下的用户数
 */
export const getAllUserCount: Core.ShareMgnt.GetAllUserCount = function () {
    return ShareMgnt('Usrm_GetAllUserCount')
}

/**
 * 分页获取所有用户下的用户
 */
export const getAllUsers: Core.ShareMgnt.GetAllUsers = function ([start, limit]) {
    return ShareMgnt('Usrm_GetAllUsers', [start, limit])
}

/**
 * 对同级部门进行排序
 */
export const sortDepartment: Core.ShareMgnt.SortDepartment = function ([userId, srcDepartId, destUpDepartId]) {
    return ShareMgnt('Usrm_SortDepartment', [userId, srcDepartId, destUpDepartId])
}

/**
 * 获取密级枚举
 */
export const getCSFLevels: Core.ShareMgnt.GetCSFLevels = function () {
    return ShareMgnt('GetCSFLevels')
}

/**
 * 获取月份间每天的最大在线数
 */
export const getMaxOnlineUserMonth: Core.ShareMgnt.GetMaxOnlineUserMonth = function ([startMonth, endMonth]) {
    return ShareMgnt('Operm_GetMaxOnlineUserMonth', [startMonth, endMonth])
}

/**
 * 获取清除缓存的时间间隔
 */
export const getClearCacheInterval: Core.ShareMgnt.GetClearCacheInterval = function () {
    return ShareMgnt('GetClearCacheInterval')
}

/**
 * 设置清除缓存的时间间隔
 */
export const setClearCacheInterval: Core.ShareMgnt.SetClearCacheInterval = function (interval) {
    return ShareMgnt('SetClearCacheInterval', [interval])
}

/**
 * 获取清除缓存的空间限额
 */
export const getClearCacheQuota: Core.ShareMgnt.GetClearCacheQuota = function () {
    return ShareMgnt('GetClearCacheQuota')
}

/**
 * 设置清除缓存的空间限额
 */
export const setClearCacheQuota: Core.ShareMgnt.SetClearCacheQuota = function (quota) {
    return ShareMgnt('SetClearCacheQuota', [quota])
}

/**
 * 获取客户端是否强制清除缓存状态
 */
export const getForceClearCacheStatus: Core.ShareMgnt.GetForceClearCacheStatus = function () {
    return ShareMgnt('GetForceClearCacheStatus')
}

/**
 * 设置客户端是否强制清除缓存
 */
export const setForceClearCacheStatus: Core.ShareMgnt.SetForceClearCacheStatus = function (status) {
    return ShareMgnt('SetForceClearCacheStatus', [status])
}

/**
 * 获取客户端是否隐藏缓存设置的状态
 */
export const getHideClientCacheSettingStatus: Core.ShareMgnt.GetHideClientCacheSettingStatus = function () {
    return ShareMgnt('GetHideClientCacheSettingStatus')
}

/**
 * 设置客户端是否隐藏缓存设置的状态
 */
export const setHideClientCacheSettingStatus: Core.ShareMgnt.SetHideClientCacheSettingStatus = function (status) {
    return ShareMgnt('SetHideClientCacheSettingStatus', [status])
}