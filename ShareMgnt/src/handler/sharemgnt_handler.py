from ShareMgnt.ttypes import (ncTAlarmConfig,
                              ncTSmtpSrvConf,
                              ncTUsrmAuthenType,
                              ncTUsrmImportResult,
                              ncTShareMgntError,
                              ncTAllConfig,
                              ncTVcodeType)
from src.common import global_info
from src.common.jsonconv_Ttype import (AlarmConfDec, AlarmConfEnc, SmtpConfDec,
                                       SmtpConfEnc)
from src.common.lib import (check_args, raise_exception, warp_exception)
from src.common.http import test_connection
from src.common.nc_senders import email_send, email_send_html_content
from src.modules.config_manage import ConfigManage
from src.modules.department_manage import DepartmentManage
from src.modules.doc_watermark_manage import DocWatermarkManage
from src.modules.domain_manage import DomainManage
from src.modules.find_share_manage import FindShareManage
from src.modules.group_manage import GroupManage
from src.modules.leak_proof_manage import LeakProofManage
from src.modules.link_share_manage import LinkShareManage
from src.modules.link_template_manage import LinkTemplateManage
from src.modules.login_access_control_manage import LoginAccessControlManage
from src.modules.login_manage import LoginManage
from src.modules.nc_thread import NotifiCenterThread
from src.modules.net_docs_limit_manage import NetDocsLimitManage
from src.modules.batch_users_manage import BatchUsersManage
from src.modules.oem_manage import OEMManage
from src.modules.online_manage import OnlineManage
from src.modules.openapi import OpenApi
from src.modules.perm_share_manage import PermShareManage
from src.modules.smtp_manage import (JsonConfManage,
                                     MailRecipient)
from src.modules.third_db_manage import ThirdDBManage
from src.modules.third_tool_manage import ThirdPartyToolManage
from src.modules.user_manage import UserManage
from src.modules.vcode_manage import VcodeManage
from src.third_party_auth.third_auth_manage import ThirdAuthManage
from src.third_party_auth.third_config_manage import ThirdConfigManage
from src.third_party_auth.third_party_manage import ThirdPartyManage
from src.third_party_auth.third_sync_manage import ThirdSyncManage
from src.modules.device_manage import DeviceManage
from src.modules.antivirus_manage import AntivirusManage
from src.modules.hide_ou_manage import HideOuManage
from src.modules.recycle_manage import RecycleManage
from src.modules.sms_manage import SmsManage
from src.modules.active_user_manage import ActiveUserManage
from src.modules.role_manage import RoleManage
from src.modules.file_crawl_manage import FileCrawlManage
from src.third_party_auth.third_import_manage import ThirdImportManage
from src.modules.doc_auto_archive_manage import DocAutoArchiveManage
from src.modules.doc_auto_clean_manage import DocAutoCleanManage
from src.modules.scan_virus_manage import ScanVirusManage
from src.modules.local_sync_manage import LocalSyncManage
from eisoo.tclients import TClient


class ShareMgntHandler(object):
    """
    ShareMgnt handler class
    """

    def __init__(self):
        """
        pass
        """
        self.batch_users_manage = BatchUsersManage()
        self.user_manage = UserManage()
        self.group_manage = GroupManage()
        self.login_manage = LoginManage()
        self.domain_manage = DomainManage()
        self.depart_manage = DepartmentManage()
        self.online_manage = OnlineManage()
        self.oem_manage = OEMManage()
        self.config_manage = ConfigManage()
        self.find_share_manage = FindShareManage()
        self.link_share_manage = LinkShareManage()
        self.perm_share_manage = PermShareManage()
        self.smtp_manage = JsonConfManage(
            "smtp_config", SmtpConfEnc, SmtpConfDec)
        self.alarm_manage = JsonConfManage(
            "alarm_config", AlarmConfEnc, AlarmConfDec)
        self.smtp_recipient_manage = MailRecipient("smtp_Recipient_config")
        self.leak_proof_manage = LeakProofManage()
        self.third_db_manage = ThirdDBManage()
        self.third_party_manage = ThirdPartyManage()
        self.third_config_manage = ThirdConfigManage()
        self.third_auth_manage = ThirdAuthManage()
        self.third_sync_manage = ThirdSyncManage()
        self.third_openapi = OpenApi()
        self.third_party_tool = ThirdPartyToolManage()
        self.login_access_control_manage = LoginAccessControlManage()
        self.doc_watermark_manage = DocWatermarkManage()
        self.link_template_manage = LinkTemplateManage()
        self.net_docs_limit_manage = NetDocsLimitManage()
        self.device_manage = DeviceManage()
        self.antivirus_manage = AntivirusManage()
        self.hide_ou_manage = HideOuManage()
        self.vcode_manage = VcodeManage()
        self.recycle_manage = RecycleManage()
        self.sms_manage = SmsManage()
        self.active_user_manage = ActiveUserManage()
        self.third_import_manage = ThirdImportManage()
        self.role_manage = RoleManage()
        self.file_crawl_manage = FileCrawlManage()
        self.doc_auto_archive_manage = DocAutoArchiveManage()
        self.doc_auto_clean_manage = DocAutoCleanManage()
        self.scan_virus_manage = ScanVirusManage()
        self.local_sync_manage = LocalSyncManage()

    @warp_exception
    @check_args
    def Usrm_GetSupervisoryRootOrg(self, userId):
        """
        获取用户管理的根组织
        """
        return self.depart_manage.get_supervisory_root_org(userId)

    @warp_exception
    @check_args
    def Usrm_GetRootOrgInfosByUserId(self, userId):
        """
        获取用户管理的根组织
        """
        return self.depart_manage.get_root_org_by_user_id(userId)

    @warp_exception
    @check_args
    def Usrm_CreateOrganization(self, addParam):
        """
        创建组织
        """
        return self.depart_manage.create_organization(addParam)

    @warp_exception
    @check_args
    def Usrm_EditOrganization(self, editParam):
        """
        编辑组织
        """
        return self.depart_manage.edit_organization(editParam)

    @warp_exception
    @check_args
    def Usrm_GetOrganizationById(self, organ_id):
        """
        编辑组织
        """
        return self.depart_manage.get_organization(organ_id, False)

    @warp_exception
    @check_args
    def Usrm_GetOrganizationByName(self, organ_name):
        """
        编辑组织
        """
        return self.depart_manage.get_organization_by_Name(organ_name, False)

    @warp_exception
    @check_args
    def Usrm_GetSubDepartments(self, department_id):
        """
        获取部门的子部门，只限一级子部门
        """
        return self.depart_manage.get_sub_departments(department_id)

    @warp_exception
    @check_args
    def Usrm_GetDepartmentOfUsersCount(self, department_id):
        """
        获取指定组织/部门下的用户数
        """
        return self.depart_manage.get_user_count_of_depart(department_id)

    @warp_exception
    @check_args
    def Usrm_GetDepartmentOfUsers(self, department_id, start, limit):
        """
        获取指定组织/部门下的用户
        """
        return self.depart_manage.get_users_of_depart(department_id, start, limit)

    @warp_exception
    @check_args
    def Usrm_AddDepartment(self, addParam):
        """
        新建部门
        """
        return self.depart_manage.add_department(addParam)

    @warp_exception
    @check_args
    def Usrm_EditDepartment(self, editParam):
        """
        编辑部门
        """
        self.depart_manage.edit_department(editParam)

    @warp_exception
    @check_args
    def Usrm_EditDepartOSS(self, departmentId, ossId):
        """
        编辑部门
        """
        self.depart_manage.edit_department_oss(departmentId, ossId)

    @warp_exception
    @check_args
    def Usrm_GetDepartmentById(self, departmentId):
        """
        根据部门id获取部门信息
        """
        return self.depart_manage.get_department_info(departmentId, False, True)

    @warp_exception
    @check_args
    def Usrm_GetDepartmentByThirdId(self, third_id):
        """
        根据部门第三方id获取部门信息
        """
        return self.depart_manage.get_department_info_by_third_id(third_id)

    @warp_exception
    @check_args
    def Usrm_GetDepartmentByName(self, name):
        """
        通过部门层级名获取部门信息
        """
        return self.depart_manage.get_department_info_by_name(name)

    @warp_exception
    @check_args
    def Usrm_MoveDepartment(self, srcDepartId, destDepartId):
        """
        移动部门
        """
        self.depart_manage.move_department(srcDepartId, destDepartId)

    @warp_exception
    @check_args
    def Usrm_SortDepartment(self, userId, srcDepartId, destDownDepartId):
        """
        对部门排序
        """
        return self.depart_manage.sort_department(userId, srcDepartId, destDownDepartId)

    @warp_exception
    @check_args
    def Usrm_GetDepartResponsiblePerson(self, depart_id):
        """
        获取指定部门下所有部门负责人信息
        """
        return self.depart_manage.get_depart_mgrs(depart_id)

    @warp_exception
    @check_args
    def Usrm_GetDepartmentParentPath(self, depart_ids):
        """
        批量根据部门ID(组织ID)获取部门（组织）父路经
        """
        return self.depart_manage.get_depart_parent_path_by_batch(depart_ids)

    @warp_exception
    @check_args
    def Usrm_AddUserToDepartment(self, user_ids, department_id):
        """
        添加用户到部门
        """
        return self.depart_manage.add_user_to_department(user_ids, department_id)

    @warp_exception
    @check_args
    def Usrm_MoveUserToDepartment(self, userIds, srcDepartId, destDepartId):
        """
        移动部门
        """
        return self.depart_manage.move_user_to_department(userIds,
                                                          srcDepartId,
                                                          destDepartId)

    @warp_exception
    @check_args
    def Usrm_RomoveUserFromDepartment(self, user_ids, department_id):
        """
        从部门移除用户
        """
        return self.depart_manage.remove_user_from_department(user_ids, department_id)

    @warp_exception
    @check_args
    def Usrm_SearchDepartments(self, user_id, search_key, start, limit):
        """
        搜索部门
        """
        return self.depart_manage.search_depart_by_key(user_id, search_key, start, limit)

    @warp_exception
    @check_args
    def Usrm_GetDeepestDeparts(self, department_id):
        """
        获取指定部门下最深层部门信息
        """
        return self.depart_manage.get_deepest_departs(department_id)

    @warp_exception
    @check_args
    def Usrm_GetSupervisoryDeparts(self, user_id):
        """
        获取用户所管辖的部门
        """
        return self.depart_manage.get_supervisory_departs(user_id)

    @warp_exception
    @check_args
    def Usrm_GetOrgDepartmentById(self, department_id):
        """
        根据部门id获取部门信息（包含组织）
        """
        return self.depart_manage.get_department_info(department_id, True)

    @warp_exception
    @check_args
    def Usrm_SearchDepartmentOfUsers(self, department_id, search_key, start, limit):
        """
        搜索用户
        """
        return self.depart_manage.search_department_of_users(department_id,
                                                             search_key,
                                                             start,
                                                             limit)

    @warp_exception
    @check_args
    def Usrm_CountSearchDepartmentOfUsers(self, department_id, search_key):
        """
        获取搜索到的用户总数
        """
        return self.depart_manage.count_serach_department_of_users(department_id,
                                                                   search_key)

    @warp_exception
    @check_args
    def Usrm_SearchSupervisoryUsers(self, managerid, key, start, limit):
        """
        Parameters:
         - managerid: 管理员id
         - key: 搜索关键字
        """
        return self.depart_manage.search_supervisory_users(managerid, key, start, limit)

    @warp_exception
    @check_args
    def Usrm_LocateUser(self, managerid, userid):
        """
        Parameters:
         - managerid
         - userid
        """
        return self.depart_manage.locate_user(managerid, userid)

    @warp_exception
    @check_args
    def Usrm_CheckUserInDepart(self, userId, departId):
        """
        检查用户是否属于某个部门及其子部门
        Parameters:
         - userId
         - departId
        """
        return self.depart_manage.check_user_in_depart_recur(userId, departId)

    @warp_exception
    def Usrm_GetAllUserCount(self):
        """
        获取用户总数

        @return int64

        @throw EThriftException.ncTException 1.获取用户总数失败.
        """
        return self.user_manage.get_all_user_count()

    @warp_exception
    @check_args
    def Usrm_GetAllUsers(self, start, limit):
        """
        获取所有用户

        @param start:从哪页开始显示
        @param limit:每页显示几条
        """
        return self.user_manage.get_all_users(start, limit)

    @warp_exception
    def Usrm_AddUser(self, user, responsible_person_id):
        """
        添加用户

        @param user: 添加的用户信息

        @throw EThriftException.ncTException 1.指定的用户不存在
        """
        return self.user_manage.add_user(user, responsible_person_id)

    @warp_exception
    def Usrm_EditUser(self, param, responsible_person_id):
        """
        修改用户

        @param user:用户信息
        """
        self.user_manage.edit_user(param, responsible_person_id)

    @warp_exception
    def Usrm_EditAdminAccount(self, adminId, account):
        """
        编辑内置管理员账号

        @param adminId: 管理员账号id
        @param account: 管理员账号名
        """
        self.user_manage.edit_admin_account(adminId, account)

    @warp_exception
    @check_args
    def Usrm_EditUserPriority(self, userId, priority):
        """
        编辑用户的排序权重
        @param user:userId, priority
        """
        self.user_manage.edit_user_priority(userId, priority)

    @warp_exception
    @check_args
    def Usrm_EditUserOSS(self, userId, ossId):
        """
        编辑用户的对象存储
        @param user:userId, priority
        """
        self.user_manage.edit_user_oss_id(userId, ossId)

    @warp_exception
    @check_args
    def Usrm_DelUser(self, user_id):
        """
        删除用户

        @param userId:用户的标识，具有唯一性
        """
        self.user_manage.delete_user(user_id)

    @warp_exception
    @check_args
    def Usrm_GetUserInfo(self, user_id):
        """
        获取指定用户的信息
        """
        return self.user_manage.get_user_by_id(user_id, origin_idcard=False)

    @warp_exception
    @check_args
    def Usrm_GetUserInfoByAccount(self, account):
        """
        获取指定用户的信息
        """
        return self.user_manage.get_user_by_loginname(account, throw_ex=True)

    @warp_exception
    @check_args
    def Usrm_GetUserInfoByThirdId(self, third_id):
        """
        根据用户第三方id获取用户信息
        """
        return self.user_manage.get_user_by_third_id(third_id)

    @warp_exception
    @check_args
    def Usrm_GetUserIdByAccount(self, account):
        """
        获取指定用户的id
        """
        return self.user_manage.get_userid_by_loginname(account)

    @warp_exception
    @check_args
    def Usrm_ModifyPassword(self, account, old_password, new_password, option):
        """
        修改用户密码
        """
        self.user_manage.modify_password(
            account, old_password, new_password, option)

    @warp_exception
    def Usrm_ResetPassword(self, user_id):
        """
        重置用户密码为初始密码：123456
        """
        self.user_manage.reset_password(user_id)

    @warp_exception
    def Usrm_ResetAllPassword(self, newPassword):
        """
        重置用户密码为初始密码：123456
        """
        self.user_manage.reset_all_password(newPassword)

    @warp_exception
    def Usrm_SetUserStatus(self, user_id, enable):
        """
        设置用户状态，启用、禁用
        """
        self.user_manage.set_user_status(user_id, enable)

    @warp_exception
    def Usrm_CheckUserStatus(self, user_id):
        """
        检查用户状态，是否启用/密码是否过期
        """
        self.user_manage.check_user_status(user_id)

    @warp_exception
    def Usrm_SelfRegistration(self, registerId, certID, realName, pwd):
        """
        用户自注册
        """
        return self.user_manage.self_registration(registerId, certID, realName, pwd)

    @warp_exception
    def Usrm_SetPasswordConfig(self, pwdConfig):
        """
        设置用户密码配置信息
        """
        self.user_manage.set_password_config(pwdConfig)

    @warp_exception
    def Usrm_GetPasswordConfig(self):
        """
        获取用户密码配置信息
        """
        return self.user_manage.get_password_config()

    @warp_exception
    def Usrm_GetSiteUsedUserNum(self, siteId):
        """
        获取指定站点启用用户数
        """
        return self.user_manage.get_site_used_user_num()

    @warp_exception
    def Usrm_SetUserExpireTime(self, userId, expireTime):
        """
        设置用户账号有效期
        """
        return self.user_manage.set_user_expire_time(userId, expireTime)

    @warp_exception
    def Usrm_GetTriSystemStatus(self):
        """
        检查三权分立是否开启
        """
        return self.user_manage.get_trisystem_status()

    @warp_exception
    def Usrm_SetTriSystemStatus(self, enable):
        """
        设置三权分立的状态
        """
        self.user_manage.set_trisystem_status(enable)

    @warp_exception
    def Usrm_SetVcodeConfig(self, vcodeConfig):
        """
        设置登录验证码配置信息
        """
        return self.vcode_manage.set_vcode_config(vcodeConfig)

    @warp_exception
    def Usrm_GetVcodeConfig(self):
        """
        获取登录验证码配置信息
        """
        return self.vcode_manage.get_vcode_config()

    @warp_exception
    def Usrm_CreateVcodeInfo(self, uuidIn, vcodeType):
        """
        生成登录验证码/忘记密码生成验证码
        """
        return self.vcode_manage.create_vcode_info(uuidIn, vcodeType)

    @warp_exception
    @check_args
    def Usrm_UserLogin(self, user_name, password, option):
        """
        用户登录验证

        @return string uuid

        @throw EThriftException.ncTException 1.用户验证失败.
        """
        return self.login_manage.login(user_name, password,
                                       ncTUsrmAuthenType.NCT_AUTHEN_TYPE_NORMAL,
                                       option)

    @warp_exception
    @check_args
    def Usrm_UserLoginByNTLMV1(self, user_name, challenge, password):
        """
        用户登录验证

        @return string uuid

        @throw EThriftException.ncTException 1.用户验证失败.
        """
        return self.login_manage.login_by_ntlmv1(user_name, challenge, password)

    @warp_exception
    @check_args
    def Usrm_UserLoginByNTLMV2(self, user_name, domain, challenge, password):
        """
        用户登录验证

        @return string uuid

        @throw EThriftException.ncTException 1.用户验证失败.
        """
        return self.login_manage.login_by_ntlmv2(user_name, domain, challenge, password)

    @warp_exception
    @check_args
    def Usrm_Login(self, user_name, password, authen_type, option, os_type):
        """
        用户登录WEB，如果登录控制台失败，记录日志
        """
        return self.login_manage.login_with_console_log(user_name, password, authen_type, option, os_type)

    @warp_exception
    def GetThirdPartyAppConfig(self, pluginType):
        """
        获取第三方配置
        """
        return self.third_config_manage.get_third_party_config(pluginType)

    @warp_exception
    def AddThirdPartyAppConfig(self, config):
        """
        新增第三方配置
        """
        return self.third_party_manage.add_third_party(config)

    @warp_exception
    def SetThirdPartyAppConfig(self, config):
        """
        设置第三方配置
        """
        self.third_party_manage.set_third_party(config)

    @warp_exception
    def DeleteThirdPartyAppConfig(self, indexId):
        """
        删除第三方配置
        """
        self.third_config_manage.delete_third_party_config(indexId)

    @warp_exception
    def AddGlobalThirdPartyPlugin(self, plugin_info):
        """
        向所有节点添加第三方认证插件
        """
        self.third_party_manage.add_global_third_party_plugin(plugin_info)

    @warp_exception
    def AddLocalThirdPartyPlugin(self, plugin_info):
        """
        向单个节点添加第三方认证插件
        """
        self.third_party_manage.add_local_third_party_plugin(plugin_info)

    @warp_exception
    def Usrm_GetThirdPartyAuth(self):
        """
        获取第三方认证系统配置
        """
        return self.third_config_manage.get_third_party_info_auth()

    @warp_exception
    def GetThirdAuthTypeStatus(self, authtype):
        """
        获取第三方认证插件是否开启
        """
        return self.third_auth_manage.get_third_auth_type_status(authtype)

    @warp_exception
    def Usrm_SendAuthVcode(self, userId, vcodeType, oldTelnum):
        """
        双因子认证短信验证码发送接口
        """
        return self.third_auth_manage.send_auth_vcode(userId, vcodeType, oldTelnum)

    @warp_exception
    def Usrm_ValidateThirdParty(self, params):
        """
        验证第三方认证
        """
        return self.third_auth_manage.validate(params)

    @warp_exception
    def Usrm_LoginConsoleByThirdParty(self, params):
        """
        第三方单点登录控制台
        """
        return self.login_manage.login_console_by_third_party(params)

    @warp_exception
    def Usrm_LoginConsoleByThirdPartyNew(self, params):
        """
        标准的第三方单点登录控制台
        """
        return self.login_manage.login_console_by_third_party_new(params)

    @warp_exception
    @check_args
    def Usrm_CheckConsoleUserPassword(self, user_name, password, authen_type, option):
        """
        验证控制台管理员的密码
        """
        return self.login_manage.login(user_name, password, authen_type, option)

    @warp_exception
    @check_args
    def Usrm_SMSValidate(self, userId, vcode):
        """
        双因子验证（短信验证码）
        """
        return self.third_auth_manage.sms_validate(userId, vcode, ncTVcodeType.DAUL_AUTH_VCODE, False)

    @warp_exception
    @check_args
    def Usrm_OTPValidate(self, userId, OTP):
        """
        双因子验证（动态密码）
        """
        return self.third_auth_manage.OTP_validate(OTP, userId)

    @warp_exception
    @check_args
    def Usrm_IMAGECodeValidate(self, uuid, vcode):
        """
        双因子验证（图形验证码）
        """
        return self.vcode_manage.verify_vcode_info(uuid, vcode)

    @warp_exception
    @check_args
    def Usrm_DomainAuth(self, loginName, ldapType, domainPath, password):
        """
        域认证
        """
        return self.login_manage.check_domain_password_func(loginName, ldapType, domainPath, password)

    @warp_exception
    @check_args
    def Usrm_ThirdAuth(self, loginName, password):
        """
        第三方认证
        """
        return self.login_manage.check_third_password_func(loginName, password)

    @warp_exception
    @check_args
    def Usrm_SendSMSVCode(self, telNumber, vcode):
        """
        发送短信验证码
        """
        return self.third_auth_manage.send_sms_vcode(telNumber, vcode)

    @warp_exception
    def Usrm_GetLoginClientInfo(self, userId):
        """
        获取用户登录webclient的信息
        """
        return self.login_manage.get_login_client_info(userId)

    @warp_exception
    def Usrm_ValidateSecurityDevice(self, params):
        """
        二次安全设备认证
        """
        return self.third_auth_manage.validate_security_device(params)

    @warp_exception
    def Usrm_GetAllDomains(self):
        """
        获取域控列表
        """
        return self.domain_manage.get_all_domains()

    @warp_exception
    @check_args
    def Usrm_AddDomain(self, domain):
        """
        添加域控

        @param domain:域控信息
        """
        return self.domain_manage.add_domain(domain)

    @warp_exception
    @check_args
    def Usrm_EditDomain(self, domain):
        """
        编辑域控

        @param domain:域控信息
        """
        self.domain_manage.edit_domain(domain)

    @warp_exception
    @check_args
    def Usrm_DeleteDomain(self, domain_id):
        """
        删除域控

        @param domain_id:域控id
        """
        self.domain_manage.delete_domain(domain_id)

    @warp_exception
    @check_args
    def Usrm_SetDomainStatus(self, domain_id, status):
        """
        启用/禁用 域控

        @param domain_id:域控id
        @param status: 状态
        """
        self.domain_manage.set_domain_status(domain_id, status)

    @warp_exception
    def Usrm_SetDomainSyncStatus(self, domainId, status):
        """
        Parameters:
         -status：
            -1:关闭域同步
            0：开启正向同步
        """
        return self.domain_manage.set_domain_sync_status(domainId, status)

    @warp_exception
    def Usrm_GetDomainById(self, domainId):
        """
        根据域控id获取域信息
        Parameters:
            domainId
        """
        return self.domain_manage.get_domain_by_id(domainId)

    @warp_exception
    def Usrm_GetDomainByName(self, domainName):
        """
        根据域名获取域控信息
        Parameters:
            domainName
        """
        return self.domain_manage.get_domain_by_name(domainName)

    @warp_exception
    def Usrm_GetDomainSyncStatus(self, domainId):
        """
        Parameters:
            domainId
        """
        return self.domain_manage.get_domain_sync_status(domainId)

    @warp_exception
    def Usrm_SetDomainConfig(self, domainId, domainConfig):
        """
        @param domainId:域控id
        @param domainConfig: 域控配置
        """
        self.domain_manage.set_domain_sync_config(domainId, domainConfig)

    @warp_exception
    def Usrm_GetDomainConfig(self, domainId):
        """
        @param domainId:域控id
        """
        return self.domain_manage.get_domain_sync_config(domainId)

    @warp_exception
    def Usrm_SetDomainKeyConfig(self, domainId, keyConfig):
        """
        @param domainId:域控id
        @param keyConfig: 域控关键字配置
        """
        self.domain_manage.set_domain_key_config(domainId, keyConfig)

    @warp_exception
    def Usrm_GetDomainKeyConfig(self, domainId):
        """
        @param domainId:域控id
        """
        return self.domain_manage.get_domain_key_config(domainId)

    @warp_exception
    def Usrm_SetADSSOStatus(self, status):
        """
        Parameters:
        """
        self.domain_manage.set_ad_sso_status(status)

    @warp_exception
    def Usrm_GetADSSOStatus(self):
        """
        Parameters:
        """
        return self.domain_manage.get_ad_sso_status()

    @warp_exception
    @check_args
    def Usrm_ExpandDomainNode(self, domain, node_path):
        """
        展开域控下的节点
        """
        return self.domain_manage.expand_domain_node(domain, node_path)

    @warp_exception
    @check_args
    def Usrm_ImportDomainUsers(self, content, option, responsible_person_id):
        """
        导入域用户
        """
        self.domain_manage.import_domain_users(
            content, option, responsible_person_id)

    @warp_exception
    @check_args
    def Usrm_ImportDomainOUs(self, content, option, responsible_person_id):
        """
        导入域用户以及组织结构
        """
        self.domain_manage.import_domain_ous(
            content, option, responsible_person_id)

    @warp_exception
    @check_args
    def Usrm_SearchDomainInfoByName(self, domain_id, name, start, limit):
        """
        搜索域用户组织
        """
        return self.domain_manage.search_info_by_name(domain_id, name, start, limit)

    @warp_exception
    @check_args
    def Usrm_CheckFailoverDomainAvailable(self, failover_domains):
        """
        检查备用域是否可用
        """
        self.domain_manage.check_failover_domain_available(failover_domains)

    @warp_exception
    @check_args
    def Usrm_EditFailoverDomains(self, failover_domains, parent_domain_id):
        """
        编辑备用域
        """
        self.domain_manage.edit_failover_domains(
            failover_domains, parent_domain_id)

    @warp_exception
    @check_args
    def Usrm_GetFailoverDomains(self, parent_domain_id):
        """
        获取备用域信息
        """
        return self.domain_manage.get_failover_domains(parent_domain_id)

    @warp_exception
    def Usrm_ImportBatchUsers(self, userinfo_file, user_cover, responsible_person_id):
        """
        批量导入用户
        """
        self.batch_users_manage.import_batch_users(
            userinfo_file, user_cover, responsible_person_id)

    @warp_exception
    def Usrm_ExportBatchUsers(self, department_ids, responsible_person_id):
        """
        批量导出用户
        """
        return self.batch_users_manage.export_batch_users(department_ids, responsible_person_id)

    @warp_exception
    def Usrm_DownloadBatchUsers(self, taskId):
        """
        下载带有用户信息的exel表
        """
        return self.batch_users_manage.download_batch_users_file(taskId)

    @warp_exception
    def Usrm_DownloadImportFailedUsers(self):
        """
        下载导入失败的用户信息exel表
        """
        return self.batch_users_manage.down_load_import_failed_users()

    @warp_exception
    def Usrm_GetProgress(self):
        """
        导入用户的进度
        """
        return self.batch_users_manage.get_import_user_progress()

    @warp_exception
    def Usrm_GetErrorInfos(self, start, limit):
        """
        获取错误信息
        """
        return self.batch_users_manage.get_import_user_errinfo(start, limit)

    @warp_exception
    def Usrm_GetExportBatchUsersTaskStatus(self, task_id):
        """
        获取导出任务的状态
        """
        return self.batch_users_manage.get_export_batch_users_task_status(task_id)

    @warp_exception
    def Usrm_GetImportProgress(self):
        """
        导入用户的进度
        """
        import_result = ncTUsrmImportResult()
        import_result.totalNum = global_info.IMPORT_TOTAL_NUM
        import_result.successNum = global_info.IMPORT_SUCCESS_NUM
        import_result.failNum = global_info.IMPORT_FAIL_NUM
        import_result.failInfos = global_info.IMPORT_FAIL_INFO

        if global_info.IMPORT_IS_STOP:
            import_result.totalNum = import_result.successNum + import_result.failNum
        return import_result

    @warp_exception
    def Usrm_ClearImportProgress(self):
        """
        清空导入进度
        """
        global_info.init_import_variable()

    @warp_exception
    def Usrm_GetThirdPartyRootNode(self, userId):
        """
        获取第三方根组织节点
        """
        return self.third_import_manage.get_root_node(userId)

    @warp_exception
    def Usrm_ExpandThirdPartyNode(self, third_id):
        """
        展开第三方节点
        """
        return self.third_import_manage.expand_node(third_id)

    @warp_exception
    def Usrm_ImportThirdPartyOUs(self, ous, users, option, responsiblePersonId):
        """
        导入第三方组织结构和用户
        """
        return self.third_import_manage.import_ous(ous, users, option, responsiblePersonId)

    @warp_exception
    @check_args
    def Usrm_CreatePersonGroup(self, cur_user_id, group_id):
        """
        创建联系人组
        """
        return self.group_manage.create_person_group(cur_user_id, group_id)

    @warp_exception
    @check_args
    def Usrm_EditPersonGroup(self, cur_user_id, group_id, new_name):
        """
        编辑联系人组
        """
        return self.group_manage.edit_person_group(cur_user_id, group_id, new_name)

    @warp_exception
    @check_args
    def Usrm_GetPersonGroups(self, cur_user_id):
        """
        获取联系人组
        """
        return self.group_manage.get_person_groups(cur_user_id)

    @warp_exception
    @check_args
    def Usrm_AddPersonByName(self, cur_user_id, name, group_id):
        """
        根据登录名、邮箱添加联系人
        """
        return self.group_manage.add_person_by_name(cur_user_id, name, group_id)

    @warp_exception
    @check_args
    def Usrm_AddPersonById(self, cur_user_id, users_id, group_id):
        """
        根据用户ID添加联系人
        """
        return self.group_manage.add_person_by_id(cur_user_id, users_id, group_id)

    @warp_exception
    @check_args
    def Usrm_DelPerson(self, cur_user_id, users_id, group_id):
        """
        删除联系人
        """
        return self.group_manage.del_person(cur_user_id, users_id, group_id)

    @warp_exception
    @check_args
    def Usrm_GetPersonFromGroup(self, cur_user_id, group_id, start, limit):
        """
        从联系人组获取联系人
        """
        return self.group_manage.get_person_from_group(cur_user_id, group_id,
                                                       start, limit)

    @warp_exception
    @check_args
    def Usrm_SearchPersonFromGroupByName(self, userId, searchKey):
        """
        从联系人组信息中根据用户显示名搜索用户
        """
        return self.group_manage.search_person_from_group(userId, searchKey)

    @warp_exception
    def Usrm_SetAutoDisable(self, config):
        """
        设置用户自动禁用配置
        """
        self.config_manage.set_auto_disable_config(config)

    @warp_exception
    def Usrm_GetAutoDisable(self):
        """
        获取用户自动禁用配置
        """
        return self.config_manage.get_auto_disable_config()

    @warp_exception
    def Usrm_SetUserFreezeStatus(self, userId, freezeStatus):
        """
        冻结|解冻用户
        """
        self.user_manage.set_user_freeze_status(userId, freezeStatus)

    @warp_exception
    def Usrm_SetFreezeStatus(self, status):
        """
        开启关闭冻结功能，True:开启，False:关闭
        """
        self.config_manage.set_freeze_status(status)

    @warp_exception
    def Usrm_GetFreezeStatus(self):
        """
        获取冻结状态，True:开启，False:关闭
        """
        return self.config_manage.get_freeze_status()

    @warp_exception
    def Usrm_SetRealNameAuthStatus(self, status):
        """
        设置实名认证开关状态
        """
        return self.config_manage.set_real_name_auth_status(status)

    @warp_exception
    def Usrm_GetRealNameAuthStatus(self):
        """
        获取实名认证开关状态（默认关闭）
        """
        return self.config_manage.get_real_name_auth_status()

    @warp_exception
    def Usrm_SetUserRealNameAuthStatus(self, userId, status):
        """
        设置用户实名状态（供ASC服务端调用）
        """
        return self.user_manage.set_user_real_name_status(userId, status)

    @warp_exception
    def Operm_GetCurrentOnlineUser(self):
        """
        获取当前在线用户总数
        """
        return self.online_manage.get_current_online_user()

    @warp_exception
    def Operm_GetMaxOnlineUserDay(self, date_month):
        """
        获取一个月内每天的最大在线用户数
        """
        return self.online_manage.get_max_online_user_day(date_month)

    @warp_exception
    def Operm_GetMaxOnlineUserMonth(self, start_month, end_month):
        """
        获取指定月份范围内的每月的最大在线用户数
        """
        return self.online_manage.get_max_online_user_month(start_month,
                                                            end_month)

    @warp_exception
    def Operm_GetEarliestTime(self):
        """
        获取有记录的最早时间
        """
        return self.online_manage.get_earliest_time()

    @warp_exception
    def OEM_SetConfig(self, oemInfo):
        """
        Parameters:
        - oemInfo
        """
        self.oem_manage.set_config(oemInfo)

    @warp_exception
    def OEM_GetConfigBySection(self, section):
        """
        Parameters:
         - section
        """
        return self.oem_manage.get_config_by_section(section)

    @warp_exception
    def OEM_GetConfigByOption(self, section, option):
        """
        Parameters:
         - section
         - option
        """
        return self.oem_manage.get_config_by_option(section, option)

    @warp_exception
    def OEM_SetUninstallPwd(self, pwd):
        """
        """
        return self.config_manage.set_uninstall_pwd(pwd)

    @warp_exception
    def OEM_GetUninstallPwd(self):
        """
        """
        return self.config_manage.get_uninstall_pwd()

    @warp_exception
    def OEM_GetUninstallPwdStatus(self):
        """n
        """
        return self.config_manage.get_uninstall_pwd_status()

    @warp_exception
    def OEM_CheckUninstallPwd(self, pwd):
        """
        """
        return self.config_manage.check_uninstall_pwd(pwd)

    @warp_exception
    def SetCustomConfigOfString(self, key, value):
        """
        设置自定义配置，String
        """
        self.config_manage.set_custom_config_of_string(key, value)

    @warp_exception
    def SetCustomConfigOfInt64(self, key, value):
        """
        设置自定义配置，Int64
        """
        self.config_manage.set_custom_config_of_int64(key, value)

    @warp_exception
    def SetCustomConfigOfBool(self, key, value):
        """
        设置自定义配置，Bool
        """
        self.config_manage.set_custom_config_of_bool(key, value)

    @warp_exception
    def GetCustomConfigOfString(self, key):
        """
        获取自定义配置，String
        """
        return self.config_manage.get_custom_config_of_string(key)

    @warp_exception
    def GetCustomConfigOfInt64(self, key):
        """
        获取自定义配置，Int64
        """
        return self.config_manage.get_custom_config_of_int64(key)

    @warp_exception
    def GetCustomConfigOfBool(self, key):
        """
        获取自定义配置，Bool
        """
        return self.config_manage.get_custom_config_of_bool(key)

    @warp_exception
    def SYNC_SyncToADOnce(self):
        """
        Parameters:
        """
        self.domain_resync_manage.sync_to_ad_once()

    @warp_exception
    def SYNC_StartSync(self, appId, autoSync):
        """
        Parameters:
        """
        return self.third_sync_manage.start_sync(appId, autoSync)

    @warp_exception
    def AddThirdSyncDBInfo(self, thirdDbInfo):
        """
        Parameters:
        """
        return self.third_db_manage.add_third_db_info(thirdDbInfo)

    @warp_exception
    def GetThirdSyncDBInfo(self, thirdDbId):
        """
        Parameters:
        """
        return self.third_db_manage.get_third_db_info(thirdDbId)

    @warp_exception
    def EditThirdSyncDBInfo(self, thirdDbInfo):
        """
        Parameters:
        """
        self.third_db_manage.edit_third_db_info(thirdDbInfo)

    @warp_exception
    def DeleteThirdSyncDBInfo(self, thirdDbId):
        """
        Parameters:
        """
        self.third_db_manage.delete_third_db_info(thirdDbId)

    @warp_exception
    def GetThirdDbTableInfos(self, thirdDbId):
        """
        Parameters:
        """
        return self.third_db_manage.get_third_db_table_infos(thirdDbId)

    @warp_exception
    def DeleteThirdTableInfo(self, thirdDbId):
        """
        Parameters:
        """
        return self.third_db_manage.delete_third_table(thirdDbId)

    @warp_exception
    def AddThirdDepartTableInfo(self, thirdTableInfo):
        """
        Parameters:
        """
        return self.third_db_manage.add_third_depart_table_info(thirdTableInfo)

    @warp_exception
    def EditThirdDepartTableInfo(self, thirdTableInfo):
        """
        Parameters:
        """
        self.third_db_manage.edit_third_depart_table_info(thirdTableInfo)

    @warp_exception
    def AddThirdDepartRelationTableInfo(self, thirdTableInfo):
        """
        Parameters:
        """
        return self.third_db_manage.add_third_depart_relation_table_info(thirdTableInfo)

    @warp_exception
    def EditThirdDepartRelationTableInfo(self, thirdTableInfo):
        """
        Parameters:
        """
        self.third_db_manage.edit_third_depart_relation_table_info(
            thirdTableInfo)

    @warp_exception
    def AddThirdUserTableInfo(self, thirdTableInfo):
        """
        Parameters:
        """
        return self.third_db_manage.add_third_user_table_info(thirdTableInfo)

    @warp_exception
    def EditThirdUserTableInfo(self, thirdTableInfo):
        """
        Parameters:
        """
        self.third_db_manage.edit_third_user_table_info(thirdTableInfo)

    @warp_exception
    def AddThirdUserRelationTableInfo(self, thirdTableInfo):
        """
        Parameters:
        """
        return self.third_db_manage.add_third_user_relation_table_info(thirdTableInfo)

    @warp_exception
    def EditThirdUserRelationTableInfo(self, thirdTableInfo):
        """
        Parameters:
        """
        self.third_db_manage.edit_third_user_relation_table_info(
            thirdTableInfo)

    @warp_exception
    def SetThirdDbSyncConfig(self, thirdDbId, syncConfig):
        """
        Parameters:
        """
        self.third_db_manage.add_third_db_sync_config(thirdDbId, syncConfig)

    @warp_exception
    def GetThirdDbSyncConfig(self, thirdDbId):
        """
        Parameters:
        """
        return self.third_db_manage.get_third_db_sync_config(thirdDbId)

    @warp_exception
    def SMTP_GetConfig(self):
        ret = self.smtp_manage.get_config()
        if not ret:
            return ncTSmtpSrvConf(safeMode=0, port=25, openRelay=False)
        return ret

    @warp_exception
    @check_args
    def SMTP_SetConfig(self, conf):
        self.smtp_manage.set_config(conf)
        NotifiCenterThread.instance().add(conf)

    @warp_exception
    @check_args
    def SMTP_ServerTest(self, conf):
        email_send(conf, [conf.email])

    @warp_exception
    def SMTP_ReceiverTest(self, toList):
        email_send_html_content(toList)

    @warp_exception
    def SMTP_SendEmail(self, toEmailList, subject, content):
        email_send_html_content(toEmailList, subject, content)

    @warp_exception
    def SMTP_SendEmailWithImage(self, toEmailList, subject, content, image):
        email_send_html_content(toEmailList, subject, content, image)

    @warp_exception
    def Alarm_GetConfig(self):
        ret = self.alarm_manage.get_config()
        if not ret:
            return ncTAlarmConfig(0, 0)
        return ret

    @warp_exception
    def SMTP_Alarm_GetConfig(self):
        return self.smtp_recipient_manage.get_config()

    @warp_exception
    def SMTP_Alarm_SetConfig(self, tomail):
        return self.smtp_recipient_manage.set_config(tomail)

    @warp_exception
    def SMTP_SetAdminMailList(self, adminId, mailList):
        return self.role_manage.set_user_role_mail(adminId, mailList)

    @warp_exception
    def SMTP_GetAdminMailList(self, adminId):
        return self.role_manage.get_user_role_mail(adminId)

    @warp_exception
    @check_args
    def Alarm_SetConfig(self, conf):
        self.alarm_manage.set_config(conf)
        NotifiCenterThread.instance().add(conf)

    @warp_exception
    def NC_SendNotify(self, notify, param):
        NotifiCenterThread.instance().add((notify, param))

    def _check_port(self, port):
        if type(port) != int:
            raise_exception(_('invalid port, must be interger'))

    @warp_exception
    def Usrm_SetUserDocStatus(self, status):
        self.config_manage.set_user_doc_status(status)

    @warp_exception
    def Usrm_GetUserDocStatus(self):
        return self.config_manage.get_user_doc_status()

    @warp_exception
    def Usrm_SetDefaulSpaceSize(self, spaceSize):
        self.config_manage.set_default_space_size(spaceSize)

    @warp_exception
    def Usrm_GetDefaulSpaceSize(self):
        return self.config_manage.get_default_space_size()

    @warp_exception
    def InitCSFLevels(self, csflevels):
        """
        初始化密级枚举
        """
        return self.config_manage.init_csf_levels(csflevels)

    @warp_exception
    def GetCSFLevels(self):
        """
        获取密级枚举
        """
        return self.config_manage.get_csf_levels()

    @warp_exception
    def GetMaxCSFLevel(self):
        """
        获取最大密级值
        """
        return self.config_manage.get_max_csf_level()

    @warp_exception
    def Usrm_GetSystemInitStatus(self):
        """
        获取系统初始化状态
        """
        return self.config_manage.get_system_init_status()

    @warp_exception
    def Usrm_InitSystem(self):
        """
        初始化系统
        """
        return self.config_manage.init_system()

    @warp_exception
    def Usrm_SetPwdControl(self, userId, param):
        """
        设置用户密码管控
        """
        return self.user_manage.set_password_control(userId, param)

    @warp_exception
    def Usrm_GetPwdControl(self, userId):
        """
        获取用户密码管控配置
        """
        return self.user_manage.get_password_control(userId)

    @warp_exception
    def Usrm_SetThirdPwdLock(self, status):
        """
        设置是否启用域认证或第三方认证密码锁策略
        """
        self.config_manage.set_third_pwd_lock(status)

    @warp_exception
    def Usrm_GetThirdPwdLock(self):
        """
        获取域认证或第三方认证是否启用密码锁策略状态
        """
        return self.config_manage.get_third_pwd_lock()

###################################################################################
#    发现共享管理相关接口
###################################################################################
    @warp_exception
    def Usrm_SetSystemFindShareStatus(self, status):
        self.find_share_manage.set_find_share_status(status)

    @warp_exception
    def Usrm_GetSystemFindShareStatus(self):
        return self.find_share_manage.get_find_share_status()

    @warp_exception
    def Usrm_AddFindShareInfo(self, shareInfo):
        return self.find_share_manage.add_find_share_info(shareInfo)

    @warp_exception
    def Usrm_DeleteFindShareInfo(self, sharerId):
        self.find_share_manage.delete_find_share_info(sharerId)

    @warp_exception
    def Usrm_GetFindShareInfoCnt(self):
        return self.find_share_manage.get_find_share_info_cnt()

    @warp_exception
    def Usrm_GetFindShareInfoByPage(self, start, limit):
        return self.find_share_manage.get_find_share_info_by_page(start, limit)

    @warp_exception
    def Usrm_SearchFindShareInfo(self, start, limit, searchKey):
        return self.find_share_manage.search_find_share_info(start, limit, searchKey)

###################################################################################
#    外链共享管理相关接口
###################################################################################
    @warp_exception
    def Usrm_SetSystemLinkShareStatus(self, status):
        self.link_share_manage.set_link_share_status(status)

    @warp_exception
    def Usrm_GetSystemLinkShareStatus(self):
        return self.link_share_manage.get_link_share_status()

    @warp_exception
    def Usrm_AddLinkShareInfo(self, shareInfo):
        return self.link_share_manage.add_link_share_info(shareInfo)

    @warp_exception
    def Usrm_DeleteLinkShareInfo(self, sharerId):
        self.link_share_manage.delete_link_share_info(sharerId)

    @warp_exception
    def Usrm_GetLinkShareInfoCnt(self):
        return self.link_share_manage.get_link_share_info_cnt()

    @warp_exception
    def Usrm_GetLinkShareInfoByPage(self, start, limit):
        return self.link_share_manage.get_link_share_info_by_page(start, limit)

    @warp_exception
    def Usrm_SearchLinkShareInfo(self, start, limit, searchKey):
        return self.link_share_manage.search_link_share_info(start, limit, searchKey)

###################################################################################
#    权限共享管理相关接口
###################################################################################
    @warp_exception
    def Usrm_SetSystemPermShareStatus(self, status):
        self.perm_share_manage.set_system_perm_share_status(status)

    @warp_exception
    def Usrm_GetSystemPermShareStatus(self):
        return self.perm_share_manage.get_system_perm_share_status()

    @warp_exception
    def Usrm_AddPermShareInfo(self, shareInfo):
        return self.perm_share_manage.add_perm_share_info(shareInfo)

    @warp_exception
    def Usrm_EditPermShareInfo(self, shareInfo):
        return self.perm_share_manage.edit_perm_share_info(shareInfo)

    @warp_exception
    def Usrm_DeletePermShareInfo(self, strategyId):
        self.perm_share_manage.delete_perm_share_info(strategyId)

    @warp_exception
    def Usrm_SetPermShareInfoStatus(self, strategyId, status):
        return self.perm_share_manage.set_perm_share_status(strategyId, status)

    @warp_exception
    def Usrm_GetPermShareInfoCnt(self):
        return self.perm_share_manage.get_perm_share_info_cnt()

    @warp_exception
    def Usrm_GetPermShareInfoByPage(self, start, limit):
        return self.perm_share_manage.get_perm_share_info_by_page(start, limit)

    @warp_exception
    def Usrm_SearchPermShareInfo(self, start, limit, searchKey):
        return self.perm_share_manage.search_perm_share_info(start, limit, searchKey)

    @warp_exception
    def Usrm_GetDefaulStrategySuperimStatus(self):
        return self.perm_share_manage.get_defaul_strategy_superim_status()

    @warp_exception
    def Usrm_SetDefaulStrategySuperimStatus(self, status):
        return self.perm_share_manage.set_defaul_strategy_superim_status(status)

###################################################################################
#    防泄密管理相关接口
###################################################################################

    @warp_exception
    def SetLeakProofStatus(self, status):
        """
        设置系统防泄密状态, true为开启，false为关闭
        """
        self.leak_proof_manage.set_leak_proof_status(status)

    @warp_exception
    def GetLeakProofStatus(self):
        """
        获取系统防泄密状态, true为开启，false为关闭
        """
        return self.leak_proof_manage.get_leak_proof_status()

    @warp_exception
    def AddLeakProofStrategy(self, param):
        """
        添加防泄密策略，返回策略id
        """
        return self.leak_proof_manage.add_strategy(param)

    @warp_exception
    def EditLeakProofStrategy(self, param):
        """
        编辑防泄密策略
        """
        self.leak_proof_manage.edit_strategy(param)

    @warp_exception
    def DeleteLeakProofStrategy(self, strategyId):
        """
        删除单条防泄密策略
        """
        self.leak_proof_manage.delete_strategy(strategyId)

    @warp_exception
    def GetLeakProofStrategyCount(self):
        """
        获取防泄密策略总条数
        """
        return self.leak_proof_manage.get_strategy_count()

    @warp_exception
    def GetLeakProofStrategyInfosByPage(self, start, limit):
        """
        分页获取防泄密策略信息
        """
        return self.leak_proof_manage.get_strategy_infos_by_page(start, limit)

    @warp_exception
    def SearchLeakProofStrategyCount(self, key):
        """
        获取防泄密策略总条数
        """
        return self.leak_proof_manage.search_strategy_count(key)

    @warp_exception
    def SearchLeakProofStrategyInfosByPage(self, key, start, limit):
        """
        分页获取防泄密策略信息
        """
        return self.leak_proof_manage.search_strategy_infos_by_page(key, start, limit)

    @warp_exception
    def GetClearCacheInterval(self):
        """
        获取清除缓存的时间间
        """
        return self.config_manage.get_clear_cache_interval()

    @warp_exception
    def SetClearCacheInterval(self, interval):
        """
        设置清除缓存的时间间隔
        """
        self.config_manage.set_clear_cache_interval(interval)

    @warp_exception
    def GetClearCacheQuota(self):
        """
        获取清除缓存的配额空间大小
        """
        return self.config_manage.get_clear_cache_size()

    @warp_exception
    def SetClearCacheQuota(self, size):
        """
        设置清除缓存的配额空间大小
        """
        self.config_manage.set_clear_cache_size(size)

    @warp_exception
    def GetForceClearCacheStatus(self):
        """
        获取客户端是否强制清除缓存状态
        """
        return self.config_manage.get_force_clear_cache_status()

    @warp_exception
    def SetForceClearCacheStatus(self, status):
        """
        设置客户端是否强制清除缓存
        """
        self.config_manage.set_force_clear_cache_status(status)

    @warp_exception
    def GetHideClientCacheSettingStatus(self):
        """
        获取客户端是否隐藏缓存设置的状态
        """
        return self.config_manage.get_hide_cache_setting_status()

    @warp_exception
    def SetHideClientCacheSettingStatus(self, status):
        """
        设置客户端是否隐藏缓存设置的状态
        """
        self.config_manage.set_hide_cache_setting_status(status)

    @warp_exception
    def SetLoginStrategyStatus(self, status):
        """
        设置登录策略状态，是否开启（禁用用户登录多个windows客户端）
        """
        self.config_manage.set_login_strategy_status(status)

    @warp_exception
    def GetLoginStrategyStatus(self):
        """
        获取登录策略状态
        """
        return self.config_manage.get_login_strategy_status()

###################################################################################
#    多租户模型相关接口
###################################################################################
    @warp_exception
    def SetMultiTenantStatus(self, status):
        """
        设置多租户开启状态
        """
        self.config_manage.set_multi_tenant_status(status)

    @warp_exception
    def GetMultiTenantStatus(self):
        """
        获取多租户开启状态
        """
        return self.config_manage.get_multi_tenant_status()

####################################################################################
#    用户管理openapi
####################################################################################

    @warp_exception
    def Usrm_AddThirdApp(self, appid):
        """
        添加第三方用户appid
        """
        return self.third_openapi.add_third_app(appid)

    @warp_exception
    def Usrm_SetThirdAppStatus(self, appid, status):
        """
        获取第三方应用状态
        """
        return self.third_openapi.set_third_app_status(appid, status)

    @warp_exception
    def Usrm_GenThirdAppSign(self, appid, appkey, method, body):
        """
        生成第三方应用签名
        """
        return self.third_openapi.gen_third_app_sign(appid, appkey, method, body)

    @warp_exception
    def Usrm_CheckThirdAppSign(self, appid, method, body, sign):
        """
        检查第三方应用签名
        """
        return self.third_openapi.check_third_app_sign(appid, method, body, sign)

####################################################################################
#    第三方预览工具配置管理
####################################################################################

    @warp_exception
    def GetThirdPartyToolConfig(self, thirdPartyToolId):
        """
        获取第三方预览工具配置信息
        """
        return self.third_party_tool.get_third_party_tool_config(thirdPartyToolId)

    @warp_exception
    def SetThirdPartyToolConfig(self, thirdPartyToolConfig):
        """
        设置第三方预览工具配置信息
        """
        self.third_party_tool.set_third_party_tool_config(thirdPartyToolConfig)

    @warp_exception
    def TestThirdPartyToolConfig(self, url):
        """
        测试第三方预览工具配置信息
        """
        return self.third_party_tool.test_third_party_tool_config(url)

    @warp_exception
    def GetAnyRobotURL(self, host, account):
        """
        获取AnyRobot跳转URL
        """

        return self.third_party_tool.get_anyrobot_url(host, account)

####################################################################################
#    涉密模块
####################################################################################

    @warp_exception
    def Secretm_GetStatus(self):
        """
        获取涉密模式总开关状态
        """
        return self.config_manage.get_secret_mode_status()

    @warp_exception
    def GetThirdCSFSysConfig(self):
        """
        获取设置第三方标密系统配置
        """
        return self.config_manage.get_third_csfsys_config()

    @warp_exception
    def GetDocWatermarkConfig(self):
        """
        获取文件水印策略配置
        """
        return self.doc_watermark_manage.get_doc_watermark_config()

    @warp_exception
    def SetDocWatermarkConfig(self, config):
        """
        设置文件水印策略配置
        """
        return self.doc_watermark_manage.set_doc_watermark_config(config)

    @warp_exception
    def AddWatermarkDoc(self, addId, watermarkType):
        """
        添加开启水印的文档库
        """
        return self.doc_watermark_manage.add_watermark_doc(addId, watermarkType)

    @warp_exception
    def UpdateWatermarkDoc(self, addId, watermarkType):
        """
        更新开启水印的文档库
        """
        return self.doc_watermark_manage.update_watermark_doc(addId, watermarkType)

    @warp_exception
    def GetWatermarkDocs(self):
        """
        获取所有开启水印的文档库
        """
        return self.doc_watermark_manage.get_watermark_docs()

    @warp_exception
    def DeleteWatermarkDoc(self, deleteId):
        """
        删除开启水印的文档库
        """
        return self.doc_watermark_manage.delete_watermark_doc(deleteId)

    @warp_exception
    def GetWatermarkDocCnt(self):
        """
        获取开启水印的文档库总数
        """
        return self.doc_watermark_manage.get_watermark_doc_cnt()

    @warp_exception
    def GetWatermarkDocByPage(self, start, limit):
        """
        分页获取开启水印的文档库信息
        """
        return self.doc_watermark_manage.get_watermark_doc_by_page(start, limit)

    @warp_exception
    def SearchWatermarkDocCnt(self, search_key):
        """
        搜索开启水印的文档库总数
        """
        return self.doc_watermark_manage.search_watermark_doc_cnt(search_key)

    @warp_exception
    def SearchWatermarkDocByPage(self, search_key, start, limit):
        """
        搜索开启水印的文档库信息
        """
        return self.doc_watermark_manage.search_watermark_doc_by_page(search_key, start, limit)

    @warp_exception
    def AddLinkTemplate(self, template_info):
        """
        添加模板
        """
        return self.link_template_manage.add_link_template(template_info)

    @warp_exception
    def DeleteLinkTemplate(self, template_id):
        """
        删除模板
        """
        return self.link_template_manage.delete_link_template_by_templateId(template_id)

    @warp_exception
    def EditLinkTemplate(self, templateInfo):
        """
        编辑模板
        """
        return self.link_template_manage.edit_link_template(templateInfo)

    @warp_exception
    def GetLinkTemplate(self, templateType):
        """
        获取模板
        """
        return self.link_template_manage.get_link_template(templateType)

    @warp_exception
    def SearchLinkTemplate(self, templateType, key):
        """
        搜索模板
        """
        return self.link_template_manage.search_link_template(templateType, key)

    @warp_exception
    def GetCalculatedLinkTemplateBySharerId(self, templateType, userId):
        """
        根据用户ID获取生效的模板
        """
        return self.link_template_manage.get_calculated_link_template_by_userId(templateType, userId)

    @warp_exception
    def CheckExternalLinkPerm(self, linkInfo):
        """
        检查外链共享权限是否符合模板
        """
        self.link_template_manage.check_external_link_perm(linkInfo)

    @warp_exception
    def Usrm_SetDDLEmailNotifyStatus(self, status):
        """
        设置下载量限制配置的邮件通知状态
        """
        self.config_manage.set_ddl_email_notify_mode_status(status)

    @warp_exception
    def Usrm_GetDDLEmailNotifyStatus(self):
        """
        获取下载量限制配置的邮件通知状态
        """
        return self.config_manage.get_ddl_email_notify_mode_status()

####################################################################################
#    文档共享开关配置
####################################################################################

    @warp_exception
    def GetShareDocStatus(self, docType, linkType):
        """
        获取共享文档开关配置
        """
        return self.config_manage.get_share_doc_status(docType, linkType)

    @warp_exception
    def SetShareDocStatus(self, docType, linkType, status):
        """
        设置共享文档开关状态
        """
        return self.config_manage.set_share_doc_status(docType, linkType, status)

####################################################################################
#    文件留底开关配置
####################################################################################

    @warp_exception
    def GetRetainFileStatus(self):
        """
        获取文件留底开关状态
        """
        return self.config_manage.get_retain_file_status()

####################################################################################
#    共享屏蔽组织架构信息管理
####################################################################################
    @warp_exception
    def GetHideUserInfoStatus(self):
        """
        获取屏蔽用户信息状态
        """
        return int(self.config_manage.get_config('hide_user_info'))

    @warp_exception
    def SetHideUserInfoStatus(self, status):
        """
        设置屏蔽用户信息状态
        """
        return self.config_manage.set_config('hide_user_info', status)

    @warp_exception
    def HideOum_GetStatus(self):
        """
        获取屏蔽组织架构显示状态
        """
        return int(self.config_manage.get_config('hide_ou_info'))

    @warp_exception
    def HideOum_SetStatus(self, status):
        """
        设置屏蔽组织架构显示状态
        """
        return self.config_manage.set_config('hide_ou_info', status)

    @warp_exception
    def HideOum_Add(self, departmentIds):
        """
        添加需要屏蔽组织架构的部门
        """
        return self.hide_ou_manage.add(departmentIds)

    @warp_exception
    def HideOum_Get(self):
        """
        获取屏蔽组织架构的部门信息
        """
        return self.hide_ou_manage.get()

    @warp_exception
    def HideOum_Search(self, searchKey):
        """
        根据部门名搜索屏蔽组织架构的部门信息
        """
        return self.hide_ou_manage.search(searchKey)

    @warp_exception
    def HideOum_Delete(self, departmentId):
        """
        根据id删除屏蔽组织架构中的部门
        """
        return self.hide_ou_manage.delete(departmentId)

    @warp_exception
    def HideOum_Check(self, userId):
        """
        检查用户是否需要屏蔽组织架构
        """
        return self.hide_ou_manage.check(userId)

    @warp_exception
    def SetSearchUserConfig(self, config):
        """
        设置用户共享时搜索配置
        """
        return self.config_manage.set_search_user_config(config)

    @warp_exception
    def GetSearchUserConfig(self):
        """
        获取用户共享时搜索配置
        """
        return self.config_manage.get_search_user_config()

####################################################################################
#    外链留底配置管理
####################################################################################
    @warp_exception
    def SetRetainOutLinkStatus(self, status):
        """
        设置外链留底开关状态
        """
        return self.config_manage.set_retain_out_link_status(status)

    @warp_exception
    def GetRetainOutLinkStatus(self):
        """
        获取外链留底开关状态
        """
        # 7.0暂不支持外链留底功能
        # return self.config_manage.get_retain_out_link_status()
        return False

####################################################################################
#    设备绑定信息管理
####################################################################################
    @warp_exception
    def Devicem_SearchUsersBindStatus(self, scope, searchKey, start, limit):
        """
        搜索用户设备绑定状态信息
        """
        return self.device_manage.search_users_bind_status(scope, searchKey,
                                                           start, limit,
                                                           cnt_only=False)

    @warp_exception
    def Devicem_SearchUsersBindStatusCnt(self, scope, searchKey):
        """
        搜索用户设备绑定状态信息数量
        """
        return self.device_manage.search_users_bind_status(scope, searchKey, 0, -1, cnt_only=True)

####################################################################################
#    防病毒管理
####################################################################################
    @warp_exception
    def GetAllAntivirusAdmin(self):
        """
        获取所有防病毒管理员
        """
        return self.antivirus_manage.get_all_antivirus_admin()

    @warp_exception
    def AddAntivirusAdmin(self, loginName):
        """
        通过登录名添加防病毒管理员
        """
        return self.antivirus_manage.add_antivirus_admin(loginName)

    @warp_exception
    def SetRecycleInfo(self, info, cid):
        """
        设置回收站配置
        """
        self.recycle_manage.set_info(info, cid)

    @warp_exception
    def GetRecycleInfo(self, cid):
        """
        获取回收站配置
        """
        return self.recycle_manage.get_info(cid)

    @warp_exception
    def DelRecycleInfo(self, cid):
        """
        删除回收站配置
        """
        self.recycle_manage.del_info(cid)

    @warp_exception
    def GetRecycleInfos(self):
        """
        获取所有回收站配置
        """
        return self.recycle_manage.get_all_info()

    @warp_exception
    def SMS_GetConfig(self):
        """
        获取短信服务器配置
        """
        return self.sms_manage.get_sms_config()

    @warp_exception
    def SMS_SetConfig(self, config):
        """
        设置短信服务器配置
        """
        return self.sms_manage.set_sms_config(config)

    @warp_exception
    def SMS_Test(self, config):
        """
        测试短信服务器
        """
        return self.sms_manage.check_sms_config(config)

    @warp_exception
    def SMS_SendVcode(self, account, passwd, telNumber):
        """
        发送短信验证码
        """
        return self.sms_manage.send_vcode(account, passwd, telNumber)

    @warp_exception
    def SMS_Activate(self, account, passwd, telNumber, mailAddress, verifyCode):
        """
        激活账号
        """
        return self.sms_manage.activate(account, passwd, telNumber, mailAddress, verifyCode)

    def GetActiveReportMonth(self, inquireDate):
        """
        获取月度活跃报表信息
        """
        return self.active_user_manage.get_active_report_month(inquireDate)

    @warp_exception
    def GetActiveReportYear(self, inquireDate):
        """
        获取年度活跃报表信息
        """
        return self.active_user_manage.get_active_report_year(inquireDate)

    @warp_exception
    def ExportActiveReportMonth(self, name, inquireDate):
        """
        创建导出月度活跃报表任务
        """
        return self.active_user_manage.export_active_report_month(name, inquireDate)

    @warp_exception
    def ExportActiveReportYear(self, name, inquireDate):
        """
        创建导出月度活跃报表任务
        """
        return self.active_user_manage.export_active_report_year(name, inquireDate)

    @warp_exception
    def GetGenActiveReportStatus(self, taskId):
        """
        获取生成活跃报表状态
        """
        return self.active_user_manage.get_gen_active_report_status(taskId)

    @warp_exception
    def GetActiveReportFileInfo(self, taskId):
        """
        获取活跃报表文件信息
        """
        return self.active_user_manage.get_active_report_file_info(taskId)

    @warp_exception
    def SetActiveReportNotifyStatus(self, status):
        """
        设置活跃报表邮件通知开关状态
        """
        self.active_user_manage.set_active_report_notify_status(status)

    @warp_exception
    def GetActiveReportNotifyStatus(self):
        """
        获取活跃报表邮件通知开关状态
        """
        return self.active_user_manage.get_active_report_notify_status()

    @warp_exception
    def SetEisooRecipientEmail(self, emailList):
        """
        设置通知到爱数的邮件接收地址
        """
        self.active_user_manage.set_eisoo_recipient_email(emailList)

    @warp_exception
    def GetEisooRecipientEmail(self):
        """
        获取通知到爱数的邮件接收地址
        """
        return self.active_user_manage.get_eisoo_recipient_email()

####################################################################################
#   用户角色管理
####################################################################################

    @warp_exception
    def UsrRolem_Add(self, roleInfo):
        """
        添加角色
        """
        return self.role_manage.add(roleInfo)

    @warp_exception
    def UsrRolem_Get(self, userId):
        """
        获取角色
        """
        return self.role_manage.get(userId)

    @warp_exception
    def UsrRolem_Edit(self, userId, roleInfo):
        """
        编辑角色
        """
        return self.role_manage.edit(userId, roleInfo)

    @warp_exception
    def UsrRolem_Delete(self, userId, roleId):
        """
        删除角色
        """
        return self.role_manage.delete(userId, roleId)

    @warp_exception
    def UsrRolem_SetMember(self, userId, roleId, memberInfo):
        """
        设置成员包含添加和编辑成员
        """
        return self.role_manage.set_member(userId, roleId, memberInfo)

    @warp_exception
    def UsrRolem_GetMember(self, userId, roleId):
        """
        获取角色成员
        """
        return self.role_manage.get_member(userId, roleId)

    @warp_exception
    def UsrRolem_SearchMember(self, userId, roleId, name):
        """
        搜索角色成员
        """
        return self.role_manage.search_member(userId, roleId, name)

    @warp_exception
    def UsrRolem_DeleteMember(self, userId, roleId, memberId):
        """
        删除成员
        """
        return self.role_manage.delete_member(userId, roleId, memberId)

    @warp_exception
    def UsrRolem_GetRole(self, userId):
        """
        获取用户角色信息
        """
        return self.role_manage.get_user_role(userId)

    @warp_exception
    def UsrRolem_GetMemberDetail(self, userId, roleId, memberId):
        """
        获取角色成员详细信息
        """
        return self.role_manage.get_member_detail(userId, roleId, memberId)

    @warp_exception
    @check_args
    def UsrRolem_GetSupervisoryRootOrg(self, userId, roleId):
        """
        根据用户角色获取用户所能看到的根组织
        """
        return self.depart_manage.get_supervisory_root_org(userId, roleId)

    @warp_exception
    @check_args
    def UsrRolem_SearchSupervisoryUsers(self, userId, roleId, key, start, limit):
        """
        在用户角色所管理的部门中搜索用户
        """
        return self.depart_manage.search_supervisory_users(userId, key, start, limit, roleId)

    @warp_exception
    @check_args
    def UsrRolem_SearchDepartments(self, userId, roleId, key, start, limit):
        """
        根据角色搜索部门
        """
        return self.depart_manage.search_depart_by_key(userId, key, start, limit, roleId)

    @warp_exception
    def UsrRolem_GetMailListByRoleId(self, roleId):
        """
        获取指定角色下所有邮箱列表
        """
        return list(self.role_manage.get_role_mails(roleId))

    @warp_exception
    def UsrRolem_CheckMemberExist(self, roleId, memberId):
        """
        检查成员是否存在
        """
        return self.role_manage.check_member_exist(roleId, memberId)

####################################################################################
#   文件抓取配置管理
####################################################################################

    @warp_exception
    def SetFileCrawlStatus(self, status):
        """
        设置文档抓取总开关
        """
        self.file_crawl_manage.set_file_crawl_status(status)

    @warp_exception
    def GetFileCrawlStatus(self):
        """
        设置文档抓取总开关
        """
        return self.file_crawl_manage.get_file_crawl_status()

    @warp_exception
    def AddFileCrawlConfig(self, fileCrawlConfig):
        """
        新建文档抓取配置
        """
        return self.file_crawl_manage.add_file_crawl_config(fileCrawlConfig)

    @warp_exception
    def SetFileCrawlConfig(self, fileCrawlConfig):
        """
        设置文档抓取配置
        """
        self.file_crawl_manage.set_file_crawl_config(fileCrawlConfig)

    @warp_exception
    def DeleteFileCrawlConfig(self, userId):
        """
        删除文档抓取配置
        """
        self.file_crawl_manage.delete_file_crawl_config(userId)

    @warp_exception
    def GetFileCrawlConfigCount(self):
        """
        获取文档抓取配置数
        """
        return self.file_crawl_manage.get_file_crawl_config_count()

    @warp_exception
    def GetFileCrawlConfig(self, start, limit):
        """
        分页获取文档抓取配置
        """
        return self.file_crawl_manage.get_file_crawl_config(start, limit)

    @warp_exception
    def GetSearchFileCrawlConfigCount(self, searchKey):
        """
        根据关键字分页搜索文档抓取配置数
        """
        return self.file_crawl_manage.get_search_file_crawl_config_count(searchKey)

    @warp_exception
    def SearchFileCrawlConfig(self, searchKey, start, limit):
        """
        根据关键字搜索文档抓取配置
        """
        return self.file_crawl_manage.search_file_crawl_config(searchKey, start, limit)

    @warp_exception
    def GetFileCrawlConfigByUserId(self, userId):
        """
        根据用户ID获取文件抓取配置
        """
        return self.file_crawl_manage.get_file_crawl_config_by_userid(userId)

    @warp_exception
    def SetFileCrawlShowStatus(self, status):
        """
        设置控制台是否显示抓取策略开关
        """
        self.file_crawl_manage.set_file_crawl_show_status(status)

    @warp_exception
    def GetFileCrawlShowStatus(self):
        """
        获取控制台是否显示抓取策略开关
        """
        return self.file_crawl_manage.get_file_crawl_show_status()

####################################################################################
#    个人文档自动归档策略管理
####################################################################################
    # @warp_exception
    # def SetDocAutoArchiveStatus(self, status):
    #     """
    #     开启/禁用自动归档策略
    #     """
    #     self.doc_auto_archive_manage.set_doc_auto_archive_status(status)

    # @warp_exception
    # def GetDocAutoArchiveStatus(self):
    #     """
    #     获取自动归档策略启用/禁用状态
    #     """
    #     return self.doc_auto_archive_manage.get_doc_auto_archive_status()

    # @warp_exception
    # def AddAutoArchiveConfig(self, config):
    #     """
    #     增加一条自动归档策略配置
    #     """
    #     return self.doc_auto_archive_manage.add_auto_archive_config(config)

    # @warp_exception
    # def EditAutoArchiveConfig(self, config):
    #     """
    #     编辑一条自动归档策略配置
    #     """
    #     return self.doc_auto_archive_manage.edit_auto_archive_config(config)

    # @warp_exception
    # def DeleteAutoArchiveConfig(self, strategyId):
    #     """
    #     删除一条自动归档策略配置
    #     """
    #     self.doc_auto_archive_manage.delete_auto_archive_config(strategyId)

    # @warp_exception
    # def GetAutoArchiveConfigCount(self, searchKey):
    #     """
    #     获取自动归档策略配置总数
    #     """
    #     return self.doc_auto_archive_manage.get_auto_archive_config_count(searchKey)

    # @warp_exception
    # def SearchAutoArchiveConfigByPage(self, start, limit, searchKey):
    #     """
    #     分页搜索自动归档策略配置
    #     """
    #     return self.doc_auto_archive_manage.search_auto_archive_config_by_page(start, limit, searchKey)

    # @warp_exception
    # def GetAllAutoArchiveUserId(self):
    #     """
    #     获取所有待归档用户Id
    #     """
    #     return self.doc_auto_archive_manage.get_all_auto_archive_userId()

    # @warp_exception
    # def GetAutoArchiveConfigByUserId(self, userId):
    #     """
    #     根据用户Id获取最后生效的策略
    #     """
    #     return self.doc_auto_archive_manage.get_auto_archive_config_by_userId(userId)

####################################################################################
#    个人文档自动清理策略管理
####################################################################################
    @warp_exception
    def SetDocAutoCleanStatus(self, status):
        """
        开启/禁用自动清理策略
        """
        self.doc_auto_clean_manage.set_doc_auto_clean_status(status)

    @warp_exception
    def GetDocAutoCleanStatus(self):
        """
        获取自动清理策略启用/禁用状态
        """
        return self.doc_auto_clean_manage.get_doc_auto_clean_status()

    @warp_exception
    def SetGlobalRecycleRetentionConfig(self, config):
        """
        设置管理员级别的回收站中数据保留时间
        """
        self.doc_auto_clean_manage.set_global_recycle_retention_config(config)

    @warp_exception
    def GetGlobalRecycleRetentionConfig(self):
        """
        获取管理员级别的回收站中数据保留时间
        """
        return self.doc_auto_clean_manage.get_global_recycle_retention_config()

    @warp_exception
    def AddAutoCleanConfig(self, config):
        """
        增加一条自动清理策略配置
        """
        return self.doc_auto_clean_manage.add_auto_clean_config(config)

    @warp_exception
    def EditAutoCleanConfig(self, config):
        """
        编辑一条自动清理策略配置
        """
        return self.doc_auto_clean_manage.edit_auto_clean_config(config)

    @warp_exception
    def DeleteAutoCleanConfig(self, strategyId):
        """
        删除一条自动归档策略配置
        """
        self.doc_auto_clean_manage.delete_auto_clean_config(strategyId)

    @warp_exception
    def SearchAutoCleanConfigByPage(self, start, limit, searchKey):
        """
        分页搜索自动清理策略配置
        """
        return self.doc_auto_clean_manage.search_auto_clean_config_by_page(start, limit, searchKey)

    @warp_exception
    def GetAutoCleanConfigCount(self, searchKey):
        """
        获取自动清理策略配置总数
        """
        return self.doc_auto_clean_manage.get_auto_clean_config_count(searchKey)

    @warp_exception
    def GetAllAutoCleanUserId(self):
        return self.doc_auto_clean_manage.get_all_auto_clean_userId()

    @warp_exception
    def GetAutoCleanConfigByUserId(self, userId):
        """
        根据用户id获取最后生效的策略
        """
        return self.doc_auto_clean_manage.get_auto_clean_config_by_userId(userId)

####################################################################################
#   指定文档库病毒扫描管理
####################################################################################

    # @warp_exception
    # def StartScanVirusTask(self, userIds, departIds, cids):
    #     """
    #     开始指定文档库病毒扫描任务
    #     """
    #     return self.scan_virus_manage.start_scan_virus_task(userIds, departIds, cids)

    @warp_exception
    def StopScanVirusTask(self):
        """
        暂停扫描任务
        """
        self.scan_virus_manage.stop_scan_virus_task()

    @warp_exception
    def ContinueScanVirusTask(self):
        """
        继续扫描任务
        """
        self.scan_virus_manage.continue_scan_virus_task()

    @warp_exception
    def CancelScanVirusTask(self):
        """
        取消扫描任务
        """
        self.scan_virus_manage.cancel_scan_virus_task()

    @warp_exception
    def DeleteInvalidSlaveSiteVirusTask(self, siteOSSIds):
        """
        删除移除主分站点关系后,分站点依旧保存在主站点的杀毒任务
        """
        self.scan_virus_manage.delete_invalid_slave_site_virus_task(siteOSSIds)

    @warp_exception
    def GetVirusInfoCnt(self):
        """
        获取本次扫描染毒文件数
        """
        return self.scan_virus_manage.get_scan_virus_task_result_cnt()

    @warp_exception
    def GetVirusInfoByPage(self, start, limit):
        """
        分页获取本次扫描染毒文件信息
        """
        return self.scan_virus_manage.get_virus_info_by_page(start, limit)

    @warp_exception
    def GetScanVirusTaskResult(self):
        """
        获取扫描结果
        """
        return self.scan_virus_manage.get_scan_virus_task_result()

    @warp_exception
    def SetGlobalVirusDB(self, virusDB):
        """
        向所有节点上传病毒库
        """
        self.scan_virus_manage.set_global_virus_db(virusDB)

    @warp_exception
    def GetVirusDB(self):
        """
        获取病毒库信息
        """
        return self.scan_virus_manage.get_virus_db()

    @warp_exception
    def GetAntivirusOptionAuthStatus(self):
        """
        杀毒选件是否授权过，存在已激活或已过期的选件都会返回True
        """
        return self.scan_virus_manage.check_enable_antivirus(raise_ex=False)

    @warp_exception
    def NotifyScanFinish(self):
        """
        杀毒完成发送邮件通知超级管理员或者安全管理员
        """
        self.scan_virus_manage.notify_scan_finish()

    @warp_exception
    def VirusFTPServerTest(self):
        """
        测试病毒服务器FTP是否正常
        """
        return self.scan_virus_manage.virus_ftp_server_test()

    @warp_exception
    def GetVirusDBDownloadUrl(self):
        """
        获取下载病毒库url
        """
        return self.scan_virus_manage.get_virus_db_download_url()

    @warp_exception
    def SetVirusDBDownloadUrl(self, virusDBDownloadUrl):
        """
        设置下载病毒库url
        """
        return self.scan_virus_manage.set_virus_db_download_url(virusDBDownloadUrl)


####################################################################################
#    本地同步配置管理
####################################################################################


    @warp_exception
    def AddLocalSyncConfig(self, config):
        """
        增加一条本地同步策略配置
        """
        return self.local_sync_manage.add_local_sync_config(config)

    @warp_exception
    def EditLocalSyncConfig(self, config):
        """
        编辑一条本地同步策略配置
        """
        self.local_sync_manage.edit_local_sync_config(config)

    @warp_exception
    def DeleteLocalSyncConfig(self, strategyId):
        """
        删除一条本地同步策略配置
        """
        self.local_sync_manage.delete_local_sync_config(strategyId)

    @warp_exception
    def GetLocalSyncConfigCount(self, searchKey):
        """
        获取本地同步策略配置总数
        """
        return self.local_sync_manage.get_local_sync_config_count(searchKey)

    @warp_exception
    def SearchLocalSyncConfigByPage(self, start, limit, searchKey):
        """
        分页搜索本地同步策略配置
        """
        return self.local_sync_manage.search_local_sync_config_by_page(start, limit, searchKey)

    @warp_exception
    def GetLocalSyncConfigByUserId(self, userId):
        """
        根据用户Id获取最后生效的策略
        """
        return self.local_sync_manage.get_local_sync_config_by_userId(userId)


####################################################################################
#    快速入门管理
####################################################################################
    @warp_exception
    def NeedQuickStart(self, user_id, os_type):
        """
        获取是否显示“快速入门”
        """
        return self.user_manage.need_quick_start(user_id, os_type)

    @warp_exception
    def SetQuickStartStatus(self, user_id, status, os_type):
        """
        更新用户"快速入门"状态
        """
        self.user_manage.set_quick_start_status(user_id, status, os_type)

####################################################################################
#    获取全部配置
####################################################################################
    @warp_exception
    def GetAllConfig(self):
        """
        获取全部配置
        """
        allconfig = ncTAllConfig()
        allconfig.thirdCSFSysConfig = self.config_manage.get_third_csfsys_config()
        allconfig.thirdPartyAuthConf = self.third_config_manage.get_third_party_info_auth()
        allconfig.toolOfficeConfig = self.third_party_tool.get_third_party_tool_config(
            "OFFICE")
        allconfig.toolWOPIConfig = self.third_party_tool.get_third_party_tool_config(
            "WOPI")
        allconfig.toolCADConfig = self.third_party_tool.get_third_party_tool_config(
            "CAD")
        allconfig.toolSurSenConfig = self.third_party_tool.get_third_party_tool_config(
            "SURSEN")
        allconfig.enableuseragreement = True if self.oem_manage.get_config_by_option(
            "anyshare", "userAgreement") == "true" else False
        allconfig.tag_max_num = self.config_manage.get_custom_config_of_int64(
            "tag_max_num")
        allconfig.smtpSrvConfig = self.smtp_manage.get_config()
        if not allconfig.smtpSrvConfig:
            allconfig.smtpSrvConfig = ncTSmtpSrvConf(
                safeMode=0, port=25, openRelay=False)
        return allconfig
