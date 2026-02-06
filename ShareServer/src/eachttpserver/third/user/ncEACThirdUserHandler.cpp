#include "eachttpserver.h"
#include "ncEACThirdUserHandler.h"
#include "ncEACHttpServerUtil.h"
#include "../ncEACThirdUtil.h"
#include <ehttpserver/ncEHttpUtil.h>
#include <ethriftutil/ncThriftClient.h>
#include "eacServiceAccessConfig.h"

ncEACThirdUserHandler::ncEACThirdUserHandler ()
{
    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p"), this);
    _methodFuncs.insert (pair<String, ncMethodFunc>(_T("createuser"), &ncEACThirdUserHandler::CreateUser));
    _methodFuncs.insert (pair<String, ncMethodFunc>(_T("edituser"), &ncEACThirdUserHandler::EditUser));
    _methodFuncs.insert (pair<String, ncMethodFunc>(_T("deleteuser"), &ncEACThirdUserHandler::DeleteUser));
    _methodFuncs.insert (pair<String, ncMethodFunc>(_T("getuserbyid"), &ncEACThirdUserHandler::GetUserById));
    _methodFuncs.insert (pair<String, ncMethodFunc>(_T("getuserbythirdid"), &ncEACThirdUserHandler::GetUserByThirdId));
    _methodFuncs.insert (pair<String, ncMethodFunc>(_T("getuserbyname"), &ncEACThirdUserHandler::GetUserByName));
    _methodFuncs.insert (pair<String, ncMethodFunc>(_T("getalluser"), &ncEACThirdUserHandler::GetAllUser));
    _methodFuncs.insert (pair<String, ncMethodFunc>(_T("getallusercount"), &ncEACThirdUserHandler::GetAllUserCount));
}

ncEACThirdUserHandler::~ncEACThirdUserHandler (void)
{
    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p"), this);
}

void
ncEACThirdUserHandler::CreateUser (brpc::Controller* cntl, ncIntrospectInfo &info)
{
    // 添加用户信息
    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, begin"), this, cntl);

    // 检查是否为应用账户，且有权限
    if (info.visitorType != ncTokenVisitorType::BUSINESS
        || !ncEACThirdUtil::CheckAppOrgPerm (info.userId, ncAppPermOrgType::USER, ncAppOrgPermValue::MODIFY)) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_USER_AUTH_ERROR,
                    LOAD_STRING (_T("IDS_EACHTTP_ERR_MSG_NOT_AUTHORIZED")));
    }

    string bodyBuffer = cntl->request_attachment ().to_string ();

    // 获取添加用户的参数
    JSON::Value requestJson;
    try {
        JSON::Reader::read (requestJson, bodyBuffer.c_str (), bodyBuffer.size ());
    }
    catch (Exception& e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_INVALID_REQUEST_JSON_FORMAT,
            LOAD_STRING (_T("IDS_EACHTTP_INVALID_JSON")));
    }
    ncTUsrmAddUserInfo addUserInfo;
    addUserInfo.user.loginName = requestJson["loginName"].s();

    if (requestJson["displayName"].type() != JSON::NIL)
        addUserInfo.user.__set_displayName(requestJson["displayName"].s ());
    if (requestJson["email_address"].type() != JSON::NIL){
        string emailAddress;
        string decodeEmailAddress(ncEACHttpServerUtil::Base64Decode(requestJson["email_address"].s().c_str()));
        emailAddress = ncEACHttpServerUtil::RSADecrypt2048(decodeEmailAddress);
        addUserInfo.user.__set_email(emailAddress);
    }
    else if (requestJson["email"].type() != JSON::NIL)
        addUserInfo.user.__set_email(requestJson["email"].s ());
    if (requestJson["type"].type() != JSON::NIL)
        addUserInfo.user.__set_userType(static_cast<ncTUsrmUserType::type>(requestJson["type"].i ()));
    if (requestJson["pwdControl"].type() != JSON::NIL)
        addUserInfo.user.__set_pwdControl(requestJson.get<bool> ("pwdControl", false));
    if (requestJson["password"].type() != JSON::NIL)
        addUserInfo.__set_password(requestJson["password"].s ());
    if (requestJson["csfLevel"].type() != JSON::NIL)
        addUserInfo.user.__set_csfLevel(requestJson["csfLevel"].i());
    if (requestJson["priority"].type() != JSON::NIL)
        addUserInfo.user.__set_priority(requestJson["priority"].i());
    if (requestJson["thirdId"].type() != JSON::NIL)
        addUserInfo.user.__set_thirdId(requestJson["thirdId"].s ());
    if (requestJson["status"].type() != JSON::NIL) {
        bool bStatus = requestJson["status"].b ();
        auto status = ncTUsrmUserStatus::NCT_STATUS_ENABLE;
        if (!bStatus) {
            status = ncTUsrmUserStatus::NCT_STATUS_DISABLE;
        }
        addUserInfo.user.__set_status(status);
    }
    if (requestJson["server_type"].type() != JSON::NIL) {
        addUserInfo.user.__set_server_type(requestJson["server_type"].i());
    }
    if (requestJson["domain_path"].type() != JSON::NIL) {
        addUserInfo.user.__set_dnPath(requestJson["domain_path"].s());
    }
    if (requestJson["manager"].type() != JSON::NIL)
    {
        JSON::Object& managerInfo = requestJson["manager"].o ();
        if (managerInfo["type"].s () != "user")
        {
            THROW_E (EAC_HTTP_SERVER, EACHTTP_INVALID_PARAM_VALUE,
                LOAD_STRING (_T("IDS_EACHTTP_ERR_MSG_AGR_ERR")));
        }

        addUserInfo.user.__set_managerID(managerInfo["id"].s ());
    }

    JSON::Array& jsonConfigs = requestJson["depIds"].a ();
    for (size_t i = 0; i < jsonConfigs.size (); ++i) {
        string tmp = jsonConfigs[i].s ().c_str ();
        addUserInfo.user.departmentIds.push_back(tmp);
    }

    jsonConfigs = requestJson["depNames"].a ();
    for (size_t i = 0; i < jsonConfigs.size (); ++i) {
        string tmp = jsonConfigs[i].s ().c_str ();
        addUserInfo.user.departmentNames.push_back(tmp);
    }

    // 调用接口创建个人文档库
    // openapi需兼容以前的功能，支持创建用户时自动创建个人文档库
    // 在用户管理还在AS的时候暂时无法解耦，因为一个接口需要实现两个功能
    // 当AS对接部署的用户管理时进行解耦
    string retUserId;
    ncTUsrmGetUserInfo retUserInfo;
    try {
        ncThriftClient<ncTShareMgntClient> shareMgntClient (EacServiceAccessConfig::getInstance()->sharemgntHost, EacServiceAccessConfig::getInstance()->sharemgntPort);
        shareMgntClient->Usrm_AddUser (retUserId, addUserInfo, g_ShareMgnt_constants.NCT_USER_ADMIN);

        // 根据id获取用户详细信息
        shareMgntClient->Usrm_GetUserInfo (retUserInfo, retUserId);

        // feature 614165 不中断升级解耦，添加用户后 document会监听用户创建的消息为用户创建个文档库
    }
    catch (ncTException & e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_USER_CREATER_ERROR, e.expMsg.c_str ());
    }
    catch (TException & e) {
        printf(e.what ());
        THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_UNKNOWN_ERR, e.what ());
    }

    //返回结果
    JSON::Value replyJson;
    replyJson["userId"] = retUserId.c_str();
    string body;
    JSON::Writer::write (replyJson.o (), body);
    ncHttpSendReply (cntl, brpc::HTTP_STATUS_OK, "ok", body);

    // 记录审计日志
    try {
        String userType = ConvertUserType(retUserInfo);
        String csfLevel = ConvertCsfLevel(retUserInfo);
        String pwdControl = ConvertPwdControl(retUserInfo);

        String msg,exmsg;
        // "新建用户账号%s(%s)成功，密级设置为 %s"
        msg.format (ncEACHttpServerLoader, _T("IDS_EACHTTP_APP_CREATER_SUCCESS"), retUserInfo.user.displayName.c_str(),
                                           retUserInfo.user.loginName.c_str(),
                                           csfLevel.getCStr(),
                                           pwdControl.getCStr());

        exmsg.format (ncEACHttpServerLoader, _T("IDS_EACHTTP_APP_CREATER_SUCCESS_EXMSG"), retUserInfo.user.loginName.c_str(),
                      retUserInfo.user.displayName.c_str(), userType.getCStr(),
                      retUserInfo.user.ossInfo.ossName.c_str(), retUserInfo.user.email.c_str());


        ncEACHttpServerUtil::Log (cntl, info.userId, info.visitorType, ncTLogType::NCT_LT_MANAGEMENT, ncTLogLevel::NCT_LL_INFO,
                                 ncTManagementType::NCT_MNT_CREATE, msg.getCStr(), exmsg.getCStr());
    }
    catch (...) {
    }

    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, end"), this, cntl);
}

void
ncEACThirdUserHandler::EditUser (brpc::Controller* cntl, ncIntrospectInfo &info)
{
    // 编辑用户
    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, begin"), this, cntl);

    // 检查是否为应用账户，且有权限
    if (info.visitorType != ncTokenVisitorType::BUSINESS
        || !ncEACThirdUtil::CheckAppOrgPerm (info.userId, ncAppPermOrgType::USER, ncAppOrgPermValue::MODIFY)) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_USER_AUTH_ERROR,
                    LOAD_STRING (_T("IDS_EACHTTP_ERR_MSG_NOT_AUTHORIZED")));
    }

    string bodyBuffer = cntl->request_attachment ().to_string ();

    // 获取编辑用户的参数
    JSON::Value requestJson;
    try {
        JSON::Reader::read (requestJson, bodyBuffer.c_str (), bodyBuffer.size ());
    }
    catch (Exception& e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_INVALID_REQUEST_JSON_FORMAT,
            LOAD_STRING (_T("IDS_EACHTTP_INVALID_JSON")));
    }
    ncTEditUserParam editUserInfo;
    bool bUpdateStatus = false, bStatus = false;
    editUserInfo.id = requestJson["userId"].s();
    if (requestJson["displayName"].type() != JSON::NIL)
        editUserInfo.__set_displayName(requestJson["displayName"].s ());
    if (requestJson["email_address"].type() != JSON::NIL){
        string emailAddress;
        string decodeEmailAddress(ncEACHttpServerUtil::Base64Decode(requestJson["email_address"].s().c_str()));
        emailAddress = ncEACHttpServerUtil::RSADecrypt2048(decodeEmailAddress);
        editUserInfo.__set_email(emailAddress);
    }
    else if (requestJson["email"].type() != JSON::NIL)
        editUserInfo.__set_email(requestJson["email"].s ());
    if (requestJson["pwdControl"].type() != JSON::NIL)
        editUserInfo.__set_pwdControl(requestJson.get<bool> ("pwdControl", false));
    if (requestJson["password"].type() != JSON::NIL)
        editUserInfo.__set_pwd(requestJson["password"].s ());
    if (requestJson["csfLevel"].type() != JSON::NIL)
        editUserInfo.__set_csfLevel(requestJson["csfLevel"].i ());
    if (requestJson["priority"].type() != JSON::NIL)
        editUserInfo.__set_priority(requestJson["priority"].i ());
    if (requestJson["status"].type() != JSON::NIL)
    {
        bUpdateStatus = true;
        bStatus = requestJson["status"].b ();
    }
    if (requestJson["manager"].type() != JSON::NIL)
    {
        JSON::Object& managerInfo = requestJson["manager"].o ();
        if (managerInfo["type"].s () != "user")
        {
            THROW_E (EAC_HTTP_SERVER, EACHTTP_INVALID_PARAM_VALUE,
                LOAD_STRING (_T("IDS_EACHTTP_ERR_MSG_AGR_ERR")));
        }

        editUserInfo.__set_managerID(managerInfo["id"].s ());
    }
    if (requestJson["account"].type() != JSON::NIL)
        editUserInfo.__set_account(requestJson["account"].s ());

    // 调用sharemgnt服务处理
    ncTUsrmGetUserInfo retUserInfo;
    try {
        ncThriftClient<ncTShareMgntClient> shareMgntClient (EacServiceAccessConfig::getInstance()->sharemgntHost, EacServiceAccessConfig::getInstance()->sharemgntPort);

        // 编辑用户
        shareMgntClient->Usrm_EditUser (editUserInfo, g_ShareMgnt_constants.NCT_USER_ADMIN);

        // 设置用户状态
        if (bUpdateStatus)
        {
            shareMgntClient->Usrm_SetUserStatus (editUserInfo.id, bStatus);
        }

        // 根据id获取用户详细信息
        shareMgntClient->Usrm_GetUserInfo (retUserInfo, editUserInfo.id);
    }
    catch (ncTException & e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_USER_EDIT_ERROR, e.expMsg.c_str ());
    }
    catch (TException & e) {
        printf(e.what ());
        THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_UNKNOWN_ERR, e.what ());
    }

    // 返回结果
    string body = "";
    ncHttpSendReply (cntl, brpc::HTTP_STATUS_OK, "ok", body);

    // 记录审计日志
    try {
        String userType = ConvertUserType(retUserInfo);
        String csfLevel = ConvertCsfLevel(retUserInfo);
        String pwdControl = ConvertPwdControl(retUserInfo);

        String msg,exmsg;
        // "编辑用户账号“%s(%s)”成功，密级设置为 “非密”，允许用户自主修改密码"
        msg.format (ncEACHttpServerLoader, _T("IDS_EACHTTP_APP_EDIT_SUCCESS"), retUserInfo.user.displayName.c_str(),
                                           retUserInfo.user.loginName.c_str(),
                                           csfLevel.getCStr(),
                                           pwdControl.getCStr());

        exmsg.format (ncEACHttpServerLoader, _T("IDS_EACHTTP_APP_CREATER_SUCCESS_EXMSG"), retUserInfo.user.loginName.c_str(),
                      retUserInfo.user.displayName.c_str(), userType.getCStr(),
                      retUserInfo.user.ossInfo.ossName.c_str(), retUserInfo.user.email.c_str());

        ncEACHttpServerUtil::Log (cntl, info.userId, info.visitorType, ncTLogType::NCT_LT_MANAGEMENT, ncTLogLevel::NCT_LL_INFO,
                                 ncTManagementType::NCT_MNT_SET, msg.getCStr(), exmsg.getCStr());
    }
    catch (...){
    }

    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, end"), this, cntl);
}

void
ncEACThirdUserHandler::DeleteUser (brpc::Controller* cntl, ncIntrospectInfo &info)
{
    // 删除用户
    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, begin"), this, cntl);

    // 检查是否为应用账户，且有权限
    if (info.visitorType != ncTokenVisitorType::BUSINESS
        || !ncEACThirdUtil::CheckAppOrgPerm (info.userId, ncAppPermOrgType::USER, ncAppOrgPermValue::MODIFY)) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_USER_AUTH_ERROR,
                    LOAD_STRING (_T("IDS_EACHTTP_ERR_MSG_NOT_AUTHORIZED")));
    }

    string bodyBuffer = cntl->request_attachment ().to_string ();

    // 获取参数
    JSON::Value requestJson;
    try {
        JSON::Reader::read (requestJson, bodyBuffer.c_str (), bodyBuffer.size ());
    }
    catch (Exception& e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_INVALID_REQUEST_JSON_FORMAT,
            LOAD_STRING (_T("IDS_EACHTTP_INVALID_JSON")));
    }
    string userId = requestJson["userId"].s();

    //调用sharemgnt服务处理请求
    ncTUsrmGetUserInfo retUserInfo;
    try {
        ncThriftClient<ncTShareMgntClient> shareMgntClient (EacServiceAccessConfig::getInstance()->sharemgntHost, EacServiceAccessConfig::getInstance()->sharemgntPort);

        // 根据id获取用户详细信息
        shareMgntClient->Usrm_GetUserInfo (retUserInfo, userId);
        shareMgntClient->Usrm_DelUser (userId);
    }
    catch (ncTException & e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_USER_DELETE_ERROR, e.expMsg.c_str ());
    }
    catch (TException & e) {
        printf(e.what ());
        THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_UNKNOWN_ERR, e.what ());
    }

    //返回结果
    string body = "";
    ncHttpSendReply (cntl, brpc::HTTP_STATUS_OK, "ok", body);

    // 记录审计日志
    try {
        String msg,exmsg;
        // 删除用户%s(%s)成功
        msg.format (ncEACHttpServerLoader, _T("IDS_EACHTTP_APP_DELETE_SUCCESS"), retUserInfo.user.loginName.c_str(),
                      retUserInfo.user.displayName.c_str());

        ncEACHttpServerUtil::Log (cntl, info.userId, info.visitorType, ncTLogType::NCT_LT_MANAGEMENT, ncTLogLevel::NCT_LL_WARN,
                                 ncTManagementType::NCT_MNT_DELETE, msg.getCStr(), exmsg.getCStr());
    }
    catch (...){
    }

    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, end"), this, cntl);
}

JSON::Value
ncEACThirdUserHandler::ConvertUserInfo (ncTUsrmGetUserInfo & userinfo, bool needThirdId)
{
    //将用户信息转为jason格式返回
    JSON::Value replyJson;
    replyJson["userId"] = userinfo.id.c_str();
    replyJson["loginName"] = userinfo.user.loginName.c_str();
    replyJson["displayName"] = userinfo.user.displayName.c_str();
    replyJson["email"] = userinfo.user.email.c_str();
    replyJson["type"] = userinfo.user.userType;
    replyJson["status"] = userinfo.user.status;
    JSON::Array& depIdsJson = replyJson["depIds"].a ();
    for(int i = 0; i < userinfo.user.departmentIds.size(); ++i) {
        depIdsJson.push_back(userinfo.user.departmentIds[i]);
    }

    JSON::Array& depNamesJson = replyJson["depNames"].a ();
    for(int i = 0; i < userinfo.user.departmentNames.size(); ++i) {
        depNamesJson.push_back(userinfo.user.departmentNames[i]);
    }
    replyJson["pwdControl"] = userinfo.user.pwdControl;
    replyJson["csfLevel"] = userinfo.user.csfLevel;
    replyJson["priority"] = userinfo.user.priority;
    if (needThirdId) {
        replyJson["thirdId"] = userinfo.user.thirdId;
    }
    return replyJson;
}

String ncEACThirdUserHandler::ConvertCsfLevel (ncTUsrmGetUserInfo & userinfo)
{
    // 调用sharemgnt服务
    map<string, int32_t> csflevels;
    try {
        ncThriftClient<ncTShareMgntClient> shareMgntClient (EacServiceAccessConfig::getInstance()->sharemgntHost, EacServiceAccessConfig::getInstance()->sharemgntPort);
        shareMgntClient->GetCSFLevels(csflevels);
    }
    catch (ncTException & e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_GET_CSF_LEVELS_ERROR, e.expMsg.c_str ());
    }
    catch (TException & e) {
        printf(e.what ());
        THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_UNKNOWN_ERR, e.what ());
    }

    String name;
    for (auto iter = csflevels.begin ();iter != csflevels.end ();iter++)
    {
        if (userinfo.user.csfLevel == iter->second)
            name = toCFLString (iter->first);
    }
    return name;
}

String ncEACThirdUserHandler::ConvertUserType (ncTUsrmGetUserInfo & userinfo)
{
    String userType;
    if (userinfo.user.userType == ncTUsrmUserType::NCT_USER_TYPE_LOCAL) {
        userType.format(ncEACHttpServerLoader, _T("IDS_LOCAL_USER"));
    }
    else if (userinfo.user.userType == ncTUsrmUserType::NCT_USER_TYPE_DOMAIN) {
        userType.format(ncEACHttpServerLoader, _T("IDS_DOMAIN_USER"));
    }
    else if (userinfo.user.userType == ncTUsrmUserType::NCT_USER_TYPE_THIRD) {
        userType.format(ncEACHttpServerLoader, _T("IDS_THIRD_USER"));
    }

    return userType;
}

String ncEACThirdUserHandler::ConvertPwdControl (ncTUsrmGetUserInfo & userinfo)
{
    String pwdControl;
    if (userinfo.user.pwdControl) {
        pwdControl.format(ncEACHttpServerLoader, _T("IDS_PWD_CONTROL"));
    }
    else {
        pwdControl.format(ncEACHttpServerLoader, _T("IDS_NOT_PWD_CONTROL"));
    }

    return pwdControl;
}

void
ncEACThirdUserHandler::GetUserById (brpc::Controller* cntl, ncIntrospectInfo &info)
{
    // 根据用户id获取用户信息
    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, begin"), this, cntl);

    // 检查是否为应用账户，且有权限
    if (info.visitorType != ncTokenVisitorType::BUSINESS
        || !ncEACThirdUtil::CheckAppOrgPerm (info.userId, ncAppPermOrgType::USER, ncAppOrgPermValue::READ)) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_USER_AUTH_ERROR,
                    LOAD_STRING (_T("IDS_EACHTTP_ERR_MSG_NOT_AUTHORIZED")));
    }

    string bodyBuffer = cntl->request_attachment ().to_string ();

    JSON::Value requestJson;
    try {
        JSON::Reader::read (requestJson, bodyBuffer.c_str (), bodyBuffer.size ());
    }
    catch (Exception& e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_INVALID_REQUEST_JSON_FORMAT,
            LOAD_STRING (_T("IDS_EACHTTP_INVALID_JSON")));
    }
    string userId = requestJson["userId"].s();

    // 调用sharemgnt服务处理请求
    ncTUsrmGetUserInfo retUserInfo;
    try {
        ncThriftClient<ncTShareMgntClient> shareMgntClient (EacServiceAccessConfig::getInstance()->sharemgntHost, EacServiceAccessConfig::getInstance()->sharemgntPort);
        shareMgntClient->Usrm_GetUserInfo (retUserInfo, userId);
    }
    catch (ncTException & e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_USER_GETUSER_ERROR, e.expMsg.c_str ());
    }
    catch (TException & e) {
        printf(e.what ());
        THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_UNKNOWN_ERR, e.what ());
    }

    //返回结果
    JSON::Value replyJson = ConvertUserInfo(retUserInfo, true);
    string body;
    JSON::Writer::write (replyJson.o (), body);
    ncHttpSendReply (cntl, brpc::HTTP_STATUS_OK, "ok", body);
    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, end"), this, cntl);
}

void
ncEACThirdUserHandler::GetUserByThirdId (brpc::Controller* cntl, ncIntrospectInfo &info)
{
    // 根据用户第三方id获取用户信息
    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, begin"), this, cntl);

    // 检查是否为应用账户，且有权限
    if (info.visitorType != ncTokenVisitorType::BUSINESS
        || !ncEACThirdUtil::CheckAppOrgPerm (info.userId, ncAppPermOrgType::USER, ncAppOrgPermValue::READ)) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_USER_AUTH_ERROR,
                    LOAD_STRING (_T("IDS_EACHTTP_ERR_MSG_NOT_AUTHORIZED")));
    }

    string bodyBuffer = cntl->request_attachment ().to_string ();
    JSON::Value requestJson;
    try {
        JSON::Reader::read (requestJson, bodyBuffer.c_str (), bodyBuffer.size ());
    }
    catch (Exception& e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_INVALID_REQUEST_JSON_FORMAT,
            LOAD_STRING (_T("IDS_EACHTTP_INVALID_JSON")));
    }
    string thirdId = "";
    if (requestJson["thirdId"].type() != JSON::NIL)
        thirdId = requestJson["thirdId"].s();;
    if (thirdId.empty()) {
        THROW_E (EAC_HTTP_SERVER, INVALID_PARAM_THIRDID, LOAD_STRING (_T("IDS_INVALID_PARAM_THIRDID")));
    }
    // 调用sharemgnt服务处理请求
    ncTUsrmGetUserInfo retUserInfo;
    try {
        ncThriftClient<ncTShareMgntClient> shareMgntClient (EacServiceAccessConfig::getInstance()->sharemgntHost, EacServiceAccessConfig::getInstance()->sharemgntPort);
        shareMgntClient->Usrm_GetUserInfoByThirdId (retUserInfo, thirdId);
    }
    catch (ncTException & e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_USER_GETUSER_ERROR, e.expMsg.c_str ());
    }
    catch (TException & e) {
        printf(e.what ());
        THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_UNKNOWN_ERR, e.what ());
    }

    //返回结果
    JSON::Value replyJson = ConvertUserInfo(retUserInfo, false);
    string body;
    JSON::Writer::write (replyJson.o (), body);
    ncHttpSendReply (cntl, brpc::HTTP_STATUS_OK, "ok", body);

    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, end"), this, cntl);
}

void
ncEACThirdUserHandler::GetUserByName (brpc::Controller* cntl, ncIntrospectInfo &info)
{
    // 通过用户名获取用户信息
    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, begin"), this, cntl);

    // 检查是否为应用账户，且有权限
    if (info.visitorType != ncTokenVisitorType::BUSINESS
        || !ncEACThirdUtil::CheckAppOrgPerm (info.userId, ncAppPermOrgType::USER, ncAppOrgPermValue::READ)) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_USER_AUTH_ERROR,
                    LOAD_STRING (_T("IDS_EACHTTP_ERR_MSG_NOT_AUTHORIZED")));
    }

    string bodyBuffer = cntl->request_attachment ().to_string ();

    // 通过登录名获取用户信息
    JSON::Value requestJson;
    try {
        JSON::Reader::read (requestJson, bodyBuffer.c_str (), bodyBuffer.size ());
    }
    catch (Exception& e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_INVALID_REQUEST_JSON_FORMAT,
            LOAD_STRING (_T("IDS_EACHTTP_INVALID_JSON")));
    }
    string loginName = requestJson["loginName"].s();

    // 调用sharemgnt服务处理请求
    ncTUsrmGetUserInfo retUserInfo;
    try {
        ncThriftClient<ncTShareMgntClient> shareMgntClient (EacServiceAccessConfig::getInstance()->sharemgntHost, EacServiceAccessConfig::getInstance()->sharemgntPort);
        shareMgntClient->Usrm_GetUserInfoByAccount (retUserInfo, loginName);
    }
    catch (ncTException & e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_USER_GETUSER_ERROR, e.expMsg.c_str ());
    }
    catch (TException & e) {
        printf(e.what ());
        THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_UNKNOWN_ERR, e.what ());
    }

    // 返回结果
    JSON::Value replyJson = ConvertUserInfo(retUserInfo, false);
    string body;
    JSON::Writer::write (replyJson.o (), body);
    ncHttpSendReply (cntl, brpc::HTTP_STATUS_OK, "ok", body);
    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, end"), this, cntl);
}

void
ncEACThirdUserHandler::GetAllUser (brpc::Controller* cntl, ncIntrospectInfo &info)
{
    // 分页获取所有用户信息
    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, begin"), this, cntl);

    // 检查是否为应用账户，且有权限
    if (info.visitorType != ncTokenVisitorType::BUSINESS
        || !ncEACThirdUtil::CheckAppOrgPerm (info.userId, ncAppPermOrgType::USER, ncAppOrgPermValue::READ)) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_USER_AUTH_ERROR,
                    LOAD_STRING (_T("IDS_EACHTTP_ERR_MSG_NOT_AUTHORIZED")));
    }

    string bodyBuffer = cntl->request_attachment ().to_string ();

    // 获取参数
    JSON::Value requestJson;
    try {
        JSON::Reader::read (requestJson, bodyBuffer.c_str (), bodyBuffer.size ());
    }
    catch (Exception& e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_INVALID_REQUEST_JSON_FORMAT,
            LOAD_STRING (_T("IDS_EACHTTP_INVALID_JSON")));
    }
    int start = requestJson["start"].i();
    int limit = requestJson["limit"].i();

    // 调用sharemgnt服务处理请求
    vector<ncTUsrmGetUserInfo> retUserInfos;
    try {
        ncThriftClient<ncTShareMgntClient> shareMgntClient (EacServiceAccessConfig::getInstance()->sharemgntHost, EacServiceAccessConfig::getInstance()->sharemgntPort);
        shareMgntClient->Usrm_GetAllUsers (retUserInfos, start, limit);
    }
    catch (ncTException & e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_USER_GETUSER_ERROR, e.expMsg.c_str ());
    }
    catch (TException & e) {
        printf(e.what ());
        THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_UNKNOWN_ERR, e.what ());
    }

    //结果处理
    JSON::Value replyJson;
    JSON::Array& usersJson = replyJson["userinfos"].a ();
    for (int i = 0; i < retUserInfos.size(); ++i) {
        usersJson.push_back (JSON::OBJECT);
        JSON::Object& tmpObj = usersJson.back ().o ();
        tmpObj = ConvertUserInfo(retUserInfos[i], false);
    }
    string body;
    JSON::Writer::write (replyJson.o (), body);
    ncHttpSendReply (cntl, brpc::HTTP_STATUS_OK, "ok", body);
    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, end"), this, cntl);
}

void
ncEACThirdUserHandler::GetAllUserCount (brpc::Controller* cntl, ncIntrospectInfo &info)
{
    // 获取所有用户数量

    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, begin"), this, cntl);

    // 检查是否为应用账户，且有权限
    if (info.visitorType != ncTokenVisitorType::BUSINESS
        || !ncEACThirdUtil::CheckAppOrgPerm (info.userId, ncAppPermOrgType::USER, ncAppOrgPermValue::READ)) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_USER_AUTH_ERROR,
                    LOAD_STRING (_T("IDS_EACHTTP_ERR_MSG_NOT_AUTHORIZED")));
    }

    string bodyBuffer = cntl->request_attachment ().to_string ();

    int count;
    try {
        ncThriftClient<ncTShareMgntClient> shareMgntClient (EacServiceAccessConfig::getInstance()->sharemgntHost, EacServiceAccessConfig::getInstance()->sharemgntPort);
        count = shareMgntClient->Usrm_GetAllUserCount ();
    }
    catch (ncTException & e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_USER_GETUSER_ERROR, e.expMsg.c_str ());
    }
    catch (TException & e) {
        printf(e.what ());
        THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_UNKNOWN_ERR, e.what ());
    }

    JSON::Value replyJson;
    replyJson["count"] = count;
    string body;
    JSON::Writer::write (replyJson.o (), body);
    ncHttpSendReply (cntl, brpc::HTTP_STATUS_OK, "ok", body);

    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, end"), this, cntl);
}
