#include "eachttpserver.h"
#include "ncEACThirdDepHandler.h"
#include "ncEACHttpServerUtil.h"
#include "../ncEACThirdUtil.h"
#include "../user/ncEACThirdUserHandler.h"
#include "../org/ncEACThirdOrgHandler.h"
#include <ehttpserver/ncEHttpUtil.h>
#include <ethriftutil/ncThriftClient.h>
#include "eacServiceAccessConfig.h"

ncEACThirdDepHandler::ncEACThirdDepHandler ()
{
    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p"), this);
    _methodFuncs.insert (pair<String, ncMethodFunc>(_T("createdep"), &ncEACThirdDepHandler::CreateDep));
    _methodFuncs.insert (pair<String, ncMethodFunc>(_T("editdep"), &ncEACThirdDepHandler::EditDep));
    _methodFuncs.insert (pair<String, ncMethodFunc>(_T("deletedep"), &ncEACThirdDepHandler::DeleteDep));
    _methodFuncs.insert (pair<String, ncMethodFunc>(_T("getdepbyid"), &ncEACThirdDepHandler::GetDepById));
    _methodFuncs.insert (pair<String, ncMethodFunc>(_T("getdepbythirdid"), &ncEACThirdDepHandler::GetDepByThirdId));
    _methodFuncs.insert (pair<String, ncMethodFunc>(_T("getdepbyname"), &ncEACThirdDepHandler::GetDepByName));
    _methodFuncs.insert (pair<String, ncMethodFunc>(_T("movedep"), &ncEACThirdDepHandler::MoveDep));
    _methodFuncs.insert (pair<String, ncMethodFunc>(_T("adduserstodep"), &ncEACThirdDepHandler::AddUsersToDep));
    _methodFuncs.insert (pair<String, ncMethodFunc>(_T("moveuserstodep"), &ncEACThirdDepHandler::MoveUsersToDep));
    _methodFuncs.insert (pair<String, ncMethodFunc>(_T("removeusersfromdep"), &ncEACThirdDepHandler::RemoveUsersFromDep));
    _methodFuncs.insert (pair<String, ncMethodFunc>(_T("getsubdepsbydepid"), &ncEACThirdDepHandler::GetSubDepsByDepId));
    _methodFuncs.insert (pair<String, ncMethodFunc>(_T("getsubusersbydepid"), &ncEACThirdDepHandler::GetSubUsersByDepId));
    _methodFuncs.insert (pair<String, ncMethodFunc>(_T("setmanager"), &ncEACThirdDepHandler::SetManager));
    _methodFuncs.insert (pair<String, ncMethodFunc>(_T("cancelmanager"), &ncEACThirdDepHandler::CancelManager));
}

ncEACThirdDepHandler::~ncEACThirdDepHandler (void)
{
    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p"), this);
}

void
ncEACThirdDepHandler::CreateDep (brpc::Controller* cntl, ncIntrospectInfo &info)
{
    // 创建部门信息
    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, begin"), this, cntl);

    // 检查是否为应用账户，且有权限
    if (info.visitorType != ncTokenVisitorType::BUSINESS
        || !ncEACThirdUtil::CheckAppOrgPerm (info.userId, ncAppPermOrgType::DEPARTMENT, ncAppOrgPermValue::MODIFY)) {
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
    ncTAddDepartParam addDepInfo;
    addDepInfo.parentId = requestJson["parentId"].s();
    addDepInfo.departName = requestJson["depName"].s();

    if (requestJson["priority"].type() != JSON::NIL)
        addDepInfo.__set_priority(requestJson["priority"].i ());
    if (requestJson["thirdId"].type() != JSON::NIL)
        addDepInfo.__set_thirdId(requestJson["thirdId"].s ());
    if (requestJson["manager"].type() != JSON::NIL)
    {
        JSON::Object& managerInfo = requestJson["manager"].o ();
        if (managerInfo["type"].s () != "user")
        {
            THROW_E (EAC_HTTP_SERVER, EACHTTP_INVALID_PARAM_VALUE,
                LOAD_STRING (_T("IDS_EACHTTP_ERR_MSG_AGR_ERR")));
        }

        addDepInfo.__set_managerID(managerInfo["id"].s ());
    }

    //调用sharemgnt服务
    string retDepId;
    ncTUsrmDepartmentInfo depInfo;
    try {
        ncThriftClient<ncTShareMgntClient> shareMgntClient (EacServiceAccessConfig::getInstance()->sharemgntHost, EacServiceAccessConfig::getInstance()->sharemgntPort);
        shareMgntClient->Usrm_AddDepartment (retDepId, addDepInfo);

        shareMgntClient->Usrm_GetDepartmentById (depInfo, retDepId);
    }
    catch (ncTException & e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_CREATER_DEP_ERROR, e.expMsg.c_str ());
    }
    catch (TException & e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_UNKNOWN_ERR, e.what ());
    }

    //返回结果
    JSON::Value replyJson;
    replyJson["depId"] = retDepId.c_str();
    string body;
    JSON::Writer::write (replyJson.o (), body);
    ncHttpSendReply (cntl, brpc::HTTP_STATUS_OK, "ok", body);

    // 记录审计日志
    String msg,exmsg;
    // 创建 部门 %s 成功
    msg.format (ncEACHttpServerLoader, _T("IDS_EACHTTP_APP_CREATER_DEP_SUCCESS"), depInfo.departmentName.c_str());
    exmsg.format (ncEACHttpServerLoader, _T("IDS_EACHTTP_APP_CREATER_DEP_SUCCESS_EXMSG"), depInfo.ossInfo.ossName.c_str());

    ncEACHttpServerUtil::Log (cntl, info.userId, info.visitorType, ncTLogType::NCT_LT_MANAGEMENT, ncTLogLevel::NCT_LL_INFO,
                             ncTManagementType::NCT_MNT_CREATE, msg.getCStr(), exmsg.getCStr());

    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, end"), this, cntl);
}

void
ncEACThirdDepHandler::EditDep (brpc::Controller* cntl, ncIntrospectInfo &info)
{
    // 编辑部门信息
    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, begin"), this, cntl);

    // 检查是否为应用账户，且有权限
    if (info.visitorType != ncTokenVisitorType::BUSINESS
        || !ncEACThirdUtil::CheckAppOrgPerm (info.userId, ncAppPermOrgType::DEPARTMENT, ncAppOrgPermValue::MODIFY)) {
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
    ncTEditDepartParam editDepInfo;
    editDepInfo.departId = requestJson["depId"].s();

    if (requestJson["depName"].type() != JSON::NIL)
        editDepInfo.__set_departName(requestJson["depName"].s ());
    if (requestJson["priority"].type() != JSON::NIL)
        editDepInfo.__set_priority(requestJson["priority"].i ());
    if (requestJson["manager"].type() != JSON::NIL)
    {
        JSON::Object& managerInfo = requestJson["manager"].o ();
        if (managerInfo["type"].s () != "user")
        {
            THROW_E (EAC_HTTP_SERVER, EACHTTP_INVALID_PARAM_VALUE,
                LOAD_STRING (_T("IDS_EACHTTP_ERR_MSG_AGR_ERR")));
        }

        editDepInfo.__set_managerID(managerInfo["id"].s ());
    }

    //调用sharemgnt服务
    string retOrgId;
    ncTUsrmDepartmentInfo oldDepInfo;
    ncTUsrmDepartmentInfo newDepInfo;
    try {
        ncThriftClient<ncTShareMgntClient> shareMgntClient (EacServiceAccessConfig::getInstance()->sharemgntHost, EacServiceAccessConfig::getInstance()->sharemgntPort);

        shareMgntClient->Usrm_GetDepartmentById (oldDepInfo, editDepInfo.departId);
        shareMgntClient->Usrm_EditDepartment (editDepInfo);
        shareMgntClient->Usrm_GetDepartmentById (newDepInfo, editDepInfo.departId);
    }
    catch (ncTException & e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_EDIT_DEP_ERROR, e.expMsg.c_str ());
    }
    catch (TException & e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_UNKNOWN_ERR, e.what ());
    }

    // 返回结果
    string body = "";
    ncHttpSendReply (cntl, brpc::HTTP_STATUS_OK, "ok", body);

    // 记录审计日志
    String msg,exmsg;
    // 编辑 组织 %s 成功
    msg.format (ncEACHttpServerLoader, _T("IDS_EACHTTP_APP_EDIT_DEP_SUCCESS"), newDepInfo.departmentName.c_str());
    exmsg.format (ncEACHttpServerLoader, _T("IDS_EACHTTP_APP_EDIT_ORG_SUCCESS_EXMSG"), oldDepInfo.departmentName.c_str(), newDepInfo.ossInfo.ossName.c_str());

    ncEACHttpServerUtil::Log (cntl, info.userId, info.visitorType, ncTLogType::NCT_LT_MANAGEMENT, ncTLogLevel::NCT_LL_INFO,
                             ncTManagementType::NCT_MNT_SET, msg.getCStr(), exmsg.getCStr());

    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, end"), this, cntl);
}

void
ncEACThirdDepHandler::DeleteDep (brpc::Controller* cntl, ncIntrospectInfo &info)
{
    // 删除部门信息
    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, begin"), this, cntl);

    // 检查是否为应用账户，且有权限
    if (info.visitorType != ncTokenVisitorType::BUSINESS
        || !ncEACThirdUtil::CheckAppOrgPerm (info.userId, ncAppPermOrgType::DEPARTMENT, ncAppOrgPermValue::MODIFY)) {
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

    string depId = requestJson["depId"].s();

    //调用sharemgnt服务
    ncTUsrmDepartmentInfo depInfo;
    try {
        ncThriftClient<ncTShareMgntClient> shareMgntClient (EacServiceAccessConfig::getInstance()->sharemgntHost, EacServiceAccessConfig::getInstance()->sharemgntPort);
        shareMgntClient->Usrm_GetDepartmentById (depInfo, depId);

        nsresult ret;
        nsCOMPtr<userManagementInterface> userManager = do_CreateInstance (USER_MANAGEMENT_CONTRACTID, &ret);
        if (NS_FAILED (ret)) {
            THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_INIT_USER_MANAGEMENT_ERR,
                _T("Failed to create usermanagement instance: 0x%x"), ret);
        }
        userManager->DeleteDepart (toCFLString(depId));
    }
    catch (ncTException & e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_DELETE_DEP_ERROR, e.expMsg.c_str ());
    }
    catch (TException & e) {
        printf(e.what ());
        THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_UNKNOWN_ERR, e.what ());
    }

    // 返回结果
    string body = "";
    ncHttpSendReply (cntl, brpc::HTTP_STATUS_OK, "ok", body);

    // 记录审计日志
    String msg,exmsg;
    // 删除 部门 %s 成功
    msg.format (ncEACHttpServerLoader, _T("IDS_EACHTTP_APP_DELETE_DEP_SUCCESS"), depInfo.departmentName.c_str());

    ncEACHttpServerUtil::Log (cntl, info.userId, info.visitorType, ncTLogType::NCT_LT_MANAGEMENT, ncTLogLevel::NCT_LL_INFO,
                             ncTManagementType::NCT_MNT_DELETE, msg.getCStr(), exmsg.getCStr());

    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, end"), this, cntl);
}

void
ncEACThirdDepHandler::GetDepById (brpc::Controller* cntl, ncIntrospectInfo &info)
{
    // 通过id获取部门信息
    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, begin"), this, cntl);

    // 检查是否为应用账户，且有权限
    if (info.visitorType != ncTokenVisitorType::BUSINESS
        || !ncEACThirdUtil::CheckAppOrgPerm (info.userId, ncAppPermOrgType::DEPARTMENT, ncAppOrgPermValue::READ)) {
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

    string depId = requestJson["depId"].s();

    //调用sharemgnt服务
    ncTUsrmDepartmentInfo depInfo;
    try {
        ncThriftClient<ncTShareMgntClient> shareMgntClient (EacServiceAccessConfig::getInstance()->sharemgntHost, EacServiceAccessConfig::getInstance()->sharemgntPort);
        shareMgntClient->Usrm_GetDepartmentById (depInfo, depId);
    }
    catch (ncTException & e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_GET_DEP_ERROR, e.expMsg.c_str ());
    }
    catch (TException & e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_UNKNOWN_ERR, e.what ());
    }

    //返回结果
    JSON::Value replyJson = ConvertDepartmentInfo(depInfo, true, true);
    string body;
    JSON::Writer::write (replyJson.o (), body);
    ncHttpSendReply (cntl, brpc::HTTP_STATUS_OK, "ok", body);


    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, end"), this, cntl);
}

void
ncEACThirdDepHandler::GetDepByThirdId (brpc::Controller* cntl, ncIntrospectInfo &info)
{
    // 通过第三方id获取部门信息
    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, begin"), this, cntl);

    // 检查是否为应用账户，且有权限
    if (info.visitorType != ncTokenVisitorType::BUSINESS
        || !ncEACThirdUtil::CheckAppOrgPerm (info.userId, ncAppPermOrgType::DEPARTMENT, ncAppOrgPermValue::READ)) {
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

    string thirdId = "";
    if (requestJson["thirdId"].type() != JSON::NIL)
        thirdId = requestJson["thirdId"].s();;
    if (thirdId.empty()) {
        THROW_E (EAC_HTTP_SERVER, INVALID_PARAM_THIRDID, LOAD_STRING (_T("IDS_INVALID_PARAM_THIRDID")));
    }

    //调用sharemgnt服务
    ncTUsrmDepartmentInfo depInfo;
    try {
        ncThriftClient<ncTShareMgntClient> shareMgntClient (EacServiceAccessConfig::getInstance()->sharemgntHost, EacServiceAccessConfig::getInstance()->sharemgntPort);
        shareMgntClient->Usrm_GetDepartmentByThirdId (depInfo, thirdId);
    }
    catch (ncTException & e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_GET_DEP_ERROR, e.expMsg.c_str ());
    }
    catch (TException & e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_UNKNOWN_ERR, e.what ());
    }

    //返回结果
    JSON::Value replyJson = ConvertDepartmentInfo(depInfo, false, false);
    string body;
    JSON::Writer::write (replyJson.o (), body);
    ncHttpSendReply (cntl, brpc::HTTP_STATUS_OK, "ok", body);

    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, end"), this, cntl);
}

void
ncEACThirdDepHandler::GetDepByName (brpc::Controller* cntl, ncIntrospectInfo &info)
{
    // 通过name获取部门信息
    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, begin"), this, cntl);

    // 检查是否为应用账户，且有权限
    if (info.visitorType != ncTokenVisitorType::BUSINESS
        || !ncEACThirdUtil::CheckAppOrgPerm (info.userId, ncAppPermOrgType::DEPARTMENT, ncAppOrgPermValue::READ)) {
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

    string name = requestJson["name"].s();

    //调用sharemgnt服务
    ncTUsrmDepartmentInfo depInfo;
    try {
        ncThriftClient<ncTShareMgntClient> shareMgntClient (EacServiceAccessConfig::getInstance()->sharemgntHost, EacServiceAccessConfig::getInstance()->sharemgntPort);
        shareMgntClient->Usrm_GetDepartmentByName (depInfo, name);
    }
    catch (ncTException & e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_GET_DEP_ERROR, e.expMsg.c_str ());
    }
    catch (TException & e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_UNKNOWN_ERR, e.what ());
    }

    //返回结果
    JSON::Value replyJson = ConvertDepartmentInfo(depInfo, false, false);
    string body;
    JSON::Writer::write (replyJson.o (), body);
    ncHttpSendReply (cntl, brpc::HTTP_STATUS_OK, "ok", body);

    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, end"), this, cntl);
}

JSON::Value
ncEACThirdDepHandler::ConvertDepartmentInfo (ncTUsrmDepartmentInfo & depInfo, bool needThirdId, bool showDepartPath)
{
    //将部门信息转为json格式返回
    JSON::Value replyJson;
    replyJson["depId"] = depInfo.departmentId.c_str();
    replyJson["depName"] = depInfo.departmentName.c_str();
    replyJson["parentId"] = depInfo.parentDepartId.c_str();
    replyJson["parentName"] = depInfo.parentDepartName.c_str();
    if (needThirdId) {
        replyJson["thirdId"] = depInfo.thirdId.c_str();
    }

    // 仅用于向上兼容，返回任意一个管理员信息
    if (!depInfo.responsiblePersons.empty ()) {
        replyJson["managerId"] = depInfo.responsiblePersons[0].id.c_str();
        replyJson["managerName"] = depInfo.responsiblePersons[0].user.displayName.c_str();
    }
    else {
        replyJson["managerId"] = "";
        replyJson["managerName"] = "";
    }

    JSON::Array& managers = replyJson["managerInfos"].a ();
    for(int i = 0; i < depInfo.responsiblePersons.size(); ++i) {
        managers.push_back (JSON::OBJECT);
        JSON::Object& tmpObj = managers.back ().o ();
        tmpObj["managerId"] = depInfo.responsiblePersons[i].id.c_str();
        tmpObj["managerName"] = depInfo.responsiblePersons[i].user.displayName.c_str();
    }

    if (showDepartPath) {
        String strPath = depInfo.parentPath.c_str();
        strPath.append ("/");
        strPath.append (depInfo.departmentName.c_str());
        replyJson["path"] = strPath.getCStr ();
    }
    return replyJson;
}

void ncEACThirdDepHandler::ConvertOrganizationName (ncTUsrmDepartmentInfo & depInfo, ncTUsrmOrganizationInfo &orgInfo)
{
    //将组织名转为部门名
    depInfo.departmentName = orgInfo.organizationName;
}

void
ncEACThirdDepHandler::MoveDep (brpc::Controller* cntl, ncIntrospectInfo &info)
{
    // 移动部门信息
    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, begin"), this, cntl);

    // 检查是否为应用账户，且有权限
    if (info.visitorType != ncTokenVisitorType::BUSINESS
        || !ncEACThirdUtil::CheckAppOrgPerm (info.userId, ncAppPermOrgType::DEPARTMENT, ncAppOrgPermValue::MODIFY)) {
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

    string depId = requestJson["depId"].s();
    string newParentDepId = requestJson["newParentDepId"].s();

    //调用sharemgnt服务
    ncTUsrmDepartmentInfo newDepInfo;
    ncTUsrmDepartmentInfo oldDepInfo;
    try {
        ncThriftClient<ncTShareMgntClient> shareMgntClient (EacServiceAccessConfig::getInstance()->sharemgntHost, EacServiceAccessConfig::getInstance()->sharemgntPort);

        shareMgntClient->Usrm_GetDepartmentById (oldDepInfo, depId);
        shareMgntClient->Usrm_MoveDepartment (depId, newParentDepId);

        shareMgntClient->Usrm_GetDepartmentById (newDepInfo, depId);

    }
    catch (ncTException & e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_MOVE_DEP_ERROR, e.expMsg.c_str ());
    }
    catch (TException & e) {
        printf(e.what ());
        THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_UNKNOWN_ERR, e.what ());
    }

    //返回结果
    string body = "";
    ncHttpSendReply (cntl, brpc::HTTP_STATUS_OK, "ok", body);

    // 记录审计日志
    String msg,exmsg;
    // 移动部门 %s 至 部门/组织 %s 成功
    msg.format (ncEACHttpServerLoader, _T("IDS_EACHTTP_APP_MOVE_DEP_SUCCESS"), oldDepInfo.departmentName.c_str(), newDepInfo.departmentName.c_str());

    ncEACHttpServerUtil::Log (cntl, info.userId, info.visitorType, ncTLogType::NCT_LT_MANAGEMENT, ncTLogLevel::NCT_LL_INFO,
                             ncTManagementType::NCT_MNT_MOVE, msg.getCStr(), exmsg.getCStr());

    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, end"), this, cntl);
}

void
ncEACThirdDepHandler::AddUsersToDep (brpc::Controller* cntl, ncIntrospectInfo &info)
{
    // 批量添加用户到部门
    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, begin"), this, cntl);

    // 检查是否为应用账户，且有权限
    if (info.visitorType != ncTokenVisitorType::BUSINESS
        || !ncEACThirdUtil::CheckAppOrgPerm (info.userId, ncAppPermOrgType::DEPARTMENT, ncAppOrgPermValue::MODIFY)) {
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

    string depId = requestJson["depId"].s();

    vector<string> userIds;
    JSON::Array& jsonConfigs = requestJson["userIds"].a ();
    for (size_t i = 0; i < jsonConfigs.size (); ++i) {
        string tmp = jsonConfigs[i].s ().c_str ();
        userIds.push_back(tmp);
    }

    //调用sharemgnt服务
    ncTUsrmDepartmentInfo depInfo;
    ncTUsrmOrganizationInfo orgInfo;
    try {
        ncThriftClient<ncTShareMgntClient> shareMgntClient (EacServiceAccessConfig::getInstance()->sharemgntHost, EacServiceAccessConfig::getInstance()->sharemgntPort);
        try{
            shareMgntClient->Usrm_GetDepartmentById (depInfo, depId);
        }
        catch (ncTException & e)
        {
            if (e.errID == ncTShareMgntError::NCT_DEPARTMENT_NOT_EXIST) {
                try{
                    shareMgntClient->Usrm_GetOrganizationById (orgInfo, depId);
                    ConvertOrganizationName(depInfo, orgInfo);
                }
                catch(...){
                    throw;
                }
            }
            else {
                throw;
            }
        }
        std::vector<string> responsibleUserID;
        shareMgntClient->Usrm_AddUserToDepartment (responsibleUserID, userIds, depId);
    }
    catch (ncTException & e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_ADD_USER_TO_DEP_ERROR, e.expMsg.c_str ());
    }
    catch (TException & e) {
        printf(e.what ());
        THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_UNKNOWN_ERR, e.what ());
    }

    //返回结果
    string body = "";
    ncHttpSendReply (cntl, brpc::HTTP_STATUS_OK, "ok", body);

    //记录审计日志
    try {
        string userId;
        String msg,exmsg;
        ncTUsrmGetUserInfo retUserInfo;
        for (int i = 0; i < userIds.size(); ++i)
        {
            userId = userIds[i];
            ncThriftClient<ncTShareMgntClient> shareMgntClient (EacServiceAccessConfig::getInstance()->sharemgntHost, EacServiceAccessConfig::getInstance()->sharemgntPort);
            shareMgntClient->Usrm_GetUserInfo (retUserInfo, userId);
            msg.format (ncEACHttpServerLoader, _T("IDS_EACHTTP_APP_ADD_USER_TO_DEP_SUCCESS"),
                        retUserInfo.user.displayName.c_str(),
                        depInfo.departmentName.c_str());

            ncEACHttpServerUtil::Log (cntl, info.userId, info.visitorType, ncTLogType::NCT_LT_MANAGEMENT, ncTLogLevel::NCT_LL_INFO,
                                     ncTManagementType::NCT_MNT_COPY, msg.getCStr(), exmsg.getCStr());
        }
    }
    catch (...){
    }
    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, end"), this, cntl);
}

void
ncEACThirdDepHandler::RemoveUsersFromDep (brpc::Controller* cntl, ncIntrospectInfo &info)
{
    // 批量从部门移除用户
    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, begin"), this, cntl);

    // 检查是否为应用账户，且有权限
    if (info.visitorType != ncTokenVisitorType::BUSINESS
        || !ncEACThirdUtil::CheckAppOrgPerm (info.userId, ncAppPermOrgType::DEPARTMENT, ncAppOrgPermValue::MODIFY)) {
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

    string depId = requestJson["depId"].s();

    vector<string> userIds;
    JSON::Array& jsonConfigs = requestJson["userIds"].a ();
    for (size_t i = 0; i < jsonConfigs.size (); ++i) {
        string tmp = jsonConfigs[i].s ().c_str ();
        userIds.push_back(tmp);
    }

    //调用sharemgnt服务
    ncTUsrmDepartmentInfo depInfo;
    ncTUsrmOrganizationInfo orgInfo;
    try {
        ncThriftClient<ncTShareMgntClient> shareMgntClient (EacServiceAccessConfig::getInstance()->sharemgntHost, EacServiceAccessConfig::getInstance()->sharemgntPort);
        try{
            shareMgntClient->Usrm_GetDepartmentById (depInfo, depId);
        }
        catch (ncTException & e)
        {
            if (e.errID == ncTShareMgntError::NCT_DEPARTMENT_NOT_EXIST) {
                try{
                    shareMgntClient->Usrm_GetOrganizationById (orgInfo, depId);
                    ConvertOrganizationName(depInfo, orgInfo);
                }
                catch(...){
                    throw;
                }
            }
            else {
                throw;
            }
        }
        vector<string> retUserIds;
        shareMgntClient->Usrm_RomoveUserFromDepartment (retUserIds, userIds, depId);
    }
    catch (ncTException & e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_REMOVE_USER_FROM_DEP_ERROR, e.expMsg.c_str ());
    }
    catch (TException & e) {
        printf(e.what ());
        THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_UNKNOWN_ERR, e.what ());
    }

    //返回结果
    string body = "";
    ncHttpSendReply (cntl, brpc::HTTP_STATUS_OK, "ok", body);

    //记录审计日志
    try {
        string userId;
        String msg,exmsg;
        ncTUsrmGetUserInfo retUserInfo;
        for (int i = 0; i < userIds.size(); ++i)
        {
            userId = userIds[i];
            ncThriftClient<ncTShareMgntClient> shareMgntClient (EacServiceAccessConfig::getInstance()->sharemgntHost, EacServiceAccessConfig::getInstance()->sharemgntPort);
            shareMgntClient->Usrm_GetUserInfo (retUserInfo, userId);
            msg.format (ncEACHttpServerLoader, _T("IDS_EACHTTP_APP_REMOVE_USER_FROM_DEP_SUCCESS"),
                        depInfo.departmentName.c_str(),
                        retUserInfo.user.displayName.c_str(),
                        retUserInfo.user.loginName.c_str());

            ncEACHttpServerUtil::Log (cntl, info.userId, info.visitorType, ncTLogType::NCT_LT_MANAGEMENT, ncTLogLevel::NCT_LL_INFO,
                                     ncTManagementType::NCT_MNT_DELETE, msg.getCStr(), exmsg.getCStr());
        }
    }
    catch (...){
    }

    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, end"), this, cntl);
}

void
ncEACThirdDepHandler::MoveUsersToDep (brpc::Controller* cntl, ncIntrospectInfo &info)
{
    // 批量移动用户到部门
    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, begin"), this, cntl);

    // 检查是否为应用账户，且有权限
    if (info.visitorType != ncTokenVisitorType::BUSINESS
        || !ncEACThirdUtil::CheckAppOrgPerm (info.userId, ncAppPermOrgType::DEPARTMENT, ncAppOrgPermValue::MODIFY)) {
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

    string srcDepId = requestJson["srcDepId"].s();
    string destDepId = requestJson["destDepId"].s();

    vector<string> userIds;
    JSON::Array& jsonConfigs = requestJson["userIds"].a ();
    for (size_t i = 0; i < jsonConfigs.size (); ++i) {
        string tmp = jsonConfigs[i].s ().c_str ();
        userIds.push_back(tmp);
    }

    //调用sharemgnt服务
    ncTUsrmDepartmentInfo depInfo;
    ncTUsrmOrganizationInfo orgInfo;
    vector<string> retUserIds;
    try {
        ncThriftClient<ncTShareMgntClient> shareMgntClient (EacServiceAccessConfig::getInstance()->sharemgntHost, EacServiceAccessConfig::getInstance()->sharemgntPort);
        try{
            shareMgntClient->Usrm_GetDepartmentById (depInfo, destDepId);
        }
        catch (ncTException & e)
        {
            if (e.errID == ncTShareMgntError::NCT_DEPARTMENT_NOT_EXIST) {
                try{
                    shareMgntClient->Usrm_GetOrganizationById (orgInfo, destDepId);
                    ConvertOrganizationName(depInfo, orgInfo);
                }
                catch(...){
                    throw;
                }
            }
            else {
                throw;
            }
        }
        shareMgntClient->Usrm_MoveUserToDepartment (retUserIds, userIds, srcDepId, destDepId);
    }
    catch (ncTException & e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_MOVE_USER_TO_DEP_ERROR, e.expMsg.c_str ());
    }
    catch (TException & e) {
        printf(e.what ());
        THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_UNKNOWN_ERR, e.what ());
    }

    //返回结果
    JSON::Value replyJson;
    JSON::Array& userIdsJson = replyJson["userIds"].a ();
    for (int i = 0; i < retUserIds.size(); ++i) {
        userIdsJson.push_back (retUserIds[i]);
    }
    string body;
    JSON::Writer::write (replyJson.o (), body);
    ncHttpSendReply (cntl, brpc::HTTP_STATUS_OK, "ok", body);

    // 记录审计日志
    // 移动部门 %s 至 部门/组织 %s 成功
    // 获取成功移动的用户id集合
    try {
        String msg,exmsg;
        string successUserId;
        bool findStr = false;
        ncTUsrmGetUserInfo retUserInfo;
        for(vector<string>::iterator iter = userIds.begin(); iter != userIds.end(); ++iter)
        {

            for(vector<string>::iterator iter1 = retUserIds.begin(); iter1 != retUserIds.end(); ++iter1)
            {
                if (*iter1 == *iter) {
                    findStr = true;
                }
            }
            if (!findStr) {
                successUserId = *iter;
                ncThriftClient<ncTShareMgntClient> shareMgntClient (EacServiceAccessConfig::getInstance()->sharemgntHost, EacServiceAccessConfig::getInstance()->sharemgntPort);
                shareMgntClient->Usrm_GetUserInfo (retUserInfo, successUserId);
                msg.format (ncEACHttpServerLoader, _T("IDS_EACHTTP_APP_MOVE_USER_TO_DEP_SUCCESS"),
                            retUserInfo.user.displayName.c_str(),
                            retUserInfo.user.loginName.c_str(),
                            depInfo.departmentName.c_str());

                ncEACHttpServerUtil::Log (cntl, info.userId, info.visitorType, ncTLogType::NCT_LT_MANAGEMENT, ncTLogLevel::NCT_LL_INFO,
                                         ncTManagementType::NCT_MNT_MOVE, msg.getCStr(), exmsg.getCStr());

            }
        }
    }
    catch (...){
    }

    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, end"), this, cntl);
}


void
ncEACThirdDepHandler::GetSubDepsByDepId (brpc::Controller* cntl, ncIntrospectInfo &info)
{
    // 通过id获取子部门信息
    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, begin"), this, cntl);

    string bodyBuffer = cntl->request_attachment ().to_string ();

    // 检查是否为应用账户，且有权限
    if (info.visitorType != ncTokenVisitorType::BUSINESS
        || !ncEACThirdUtil::CheckAppOrgPerm (info.userId, ncAppPermOrgType::DEPARTMENT, ncAppOrgPermValue::READ)) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_USER_AUTH_ERROR,
                    LOAD_STRING (_T("IDS_EACHTTP_ERR_MSG_NOT_AUTHORIZED")));
    }

    // 获取参数
    JSON::Value requestJson;
    try {
        JSON::Reader::read (requestJson, bodyBuffer.c_str (), bodyBuffer.size ());
    }
    catch (Exception& e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_INVALID_REQUEST_JSON_FORMAT,
            LOAD_STRING (_T("IDS_EACHTTP_INVALID_JSON")));
    }

    string depId = requestJson["depId"].s();

    //调用sharemgnt服务
    vector<ncTDepartmentInfo> retDepInfos;
    try {
        ncThriftClient<ncTShareMgntClient> shareMgntClient (EacServiceAccessConfig::getInstance()->sharemgntHost, EacServiceAccessConfig::getInstance()->sharemgntPort);
        shareMgntClient->Usrm_GetSubDepartments (retDepInfos, depId);
    }
    catch (ncTException & e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_GET_SUB_DEP_ERROR, e.expMsg.c_str ());
    }
    catch (TException & e) {
        printf(e.what ());
        THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_UNKNOWN_ERR, e.what ());
    }

    //结果处理
    JSON::Value replyJson;
    JSON::Array& depJson = replyJson["subDepInfos"].a ();
    for (int i = 0; i < retDepInfos.size(); ++i) {
        depJson.push_back (JSON::OBJECT);
        JSON::Object& tmpObj = depJson.back ().o ();
        tmpObj = ncEACThirdOrgHandler::ConvertDepInfo(retDepInfos[i]);
    }
    string body;
    JSON::Writer::write (replyJson.o (), body);
    ncHttpSendReply (cntl, brpc::HTTP_STATUS_OK, "ok", body);

    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, end"), this, cntl);
}

void
ncEACThirdDepHandler::GetSubUsersByDepId (brpc::Controller* cntl, ncIntrospectInfo &info)
{
    // 通过id获取子用户信息
    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, begin"), this, cntl);

    // 检查是否为应用账户，且有权限
    if (info.visitorType != ncTokenVisitorType::BUSINESS
        || !ncEACThirdUtil::CheckAppOrgPerm (info.userId, ncAppPermOrgType::DEPARTMENT, ncAppOrgPermValue::READ)) {
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

    string depId = requestJson["depId"].s();
    int start = requestJson["start"].i();
    int limit = requestJson["limit"].i();

    //调用sharemgnt服务
    vector<ncTUsrmGetUserInfo> retUserInfos;
    try {
        ncThriftClient<ncTShareMgntClient> shareMgntClient (EacServiceAccessConfig::getInstance()->sharemgntHost, EacServiceAccessConfig::getInstance()->sharemgntPort);
        shareMgntClient->Usrm_GetDepartmentOfUsers (retUserInfos, depId, start, limit);
    }
    catch (ncTException & e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_GET_SUB_USER_ERROR, e.expMsg.c_str ());
    }
    catch (TException & e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_UNKNOWN_ERR, e.what ());
    }

    //结果处理
    JSON::Value replyJson;
    JSON::Array& depJson = replyJson["subUserInfos"].a ();
    for (int i = 0; i < retUserInfos.size(); ++i) {
        depJson.push_back (JSON::OBJECT);
        JSON::Object& tmpObj = depJson.back ().o ();
        tmpObj = ncEACThirdUserHandler::ConvertUserInfo(retUserInfos[i], false);
    }
    string body;
    JSON::Writer::write (replyJson.o (), body);
    ncHttpSendReply (cntl, brpc::HTTP_STATUS_OK, "ok", body);

    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, end"), this, cntl);
}

void
ncEACThirdDepHandler::SetManager (brpc::Controller* cntl, ncIntrospectInfo &info)
{
    // 设置部门负责人
    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, begin"), this, cntl);

    // 检查是否为应用账户，且有权限
    if (info.visitorType != ncTokenVisitorType::BUSINESS
        || !ncEACThirdUtil::CheckAppOrgPerm (info.userId, ncAppPermOrgType::DEPARTMENT, ncAppOrgPermValue::MODIFY)) {
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

    // 获取所有部门ID
    vector<string> depIds;
    JSON::Array jsonDepIds = requestJson["depIds"].a();
    if (jsonDepIds.size() == 0) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_INVALID_PARAMETER,                                                                         \
                     LOAD_STRING (_T("IDS_EACHTTP_INVALID_ARG_VALUE")), "depIds");
    }

    for(auto tempId:jsonDepIds){
        depIds.push_back(tempId.s());
    }
    string userId = requestJson["userId"].s();

    //调用sharemgnt服务
    ncTUsrmOrganizationInfo orgInfo;
    ncTUsrmGetUserInfo retUserInfo;
    string strDepNames;
    try {
        ncThriftClient<ncTShareMgntClient> shareMgntClient (EacServiceAccessConfig::getInstance()->sharemgntHost, EacServiceAccessConfig::getInstance()->sharemgntPort);
        // 获取所有部门的名称，用于日志
        for (auto iter = depIds.begin(); iter != depIds.end(); iter++) {
            ncTUsrmDepartmentInfo depInfo;
            shareMgntClient->Usrm_GetOrgDepartmentById (depInfo, *iter);

            if (strDepNames != "") {
                strDepNames += ",";
            }

            strDepNames += depInfo.departmentName;
        }
        shareMgntClient->Usrm_GetUserInfo (retUserInfo, userId);

        // 获取当前三权分立是否开启，若开启则用安全管理员账号设置，若未开启则用超级管理员账号
        bool bStatus = shareMgntClient->Usrm_GetTriSystemStatus();
        string userID = g_ShareMgnt_constants.NCT_USER_ADMIN;
        if (bStatus) {
            userID = g_ShareMgnt_constants.NCT_USER_SECURIT;
        }

        ncTRoleMemberInfo memberInfo;
        memberInfo.userId = userId;
        memberInfo.manageDeptInfo = ncTManageDeptInfo();
        memberInfo.manageDeptInfo.departmentIds = std::move(depIds);
        shareMgntClient->UsrRolem_SetMember (userID,
                                             g_ShareMgnt_constants.NCT_SYSTEM_ROLE_ORG_MANAGER,
                                             memberInfo);

    }
    catch (ncTException& e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_SET_MANAGER_ERROR, e.expMsg.c_str ());
    }
    catch (TException& e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_UNKNOWN_ERR, e.what ());
    }

    //返回结果
    string body = "";
    ncHttpSendReply (cntl, brpc::HTTP_STATUS_OK, "ok", body);

    // 记录审计日志
    String msg,exmsg;
    msg.format (ncEACHttpServerLoader, _T("IDS_EACHTTP_APP_SET_MANAGER_SUCCESS"), strDepNames.c_str(), retUserInfo.user.displayName.c_str());

    ncEACHttpServerUtil::Log (cntl, info.userId, info.visitorType, ncTLogType::NCT_LT_MANAGEMENT, ncTLogLevel::NCT_LL_INFO,
                             ncTManagementType::NCT_MNT_SET, msg.getCStr(), exmsg.getCStr());

    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, end"), this, cntl);
}


void
ncEACThirdDepHandler::CancelManager (brpc::Controller* cntl, ncIntrospectInfo &info)
{
    // 设置部门负责人
    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, begin"), this, cntl);

    // 检查是否为应用账户，且有权限
    if (info.visitorType != ncTokenVisitorType::BUSINESS
        || !ncEACThirdUtil::CheckAppOrgPerm (info.userId, ncAppPermOrgType::DEPARTMENT, ncAppOrgPermValue::MODIFY)) {
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

    //调用sharemgnt服务
    ncTUsrmOrganizationInfo orgInfo;
    // 记录审计日志
    ncTRoleMemberInfo userInfo;
    string strDepNames;
    try {
        // 用于记录日志
        ncThriftClient<ncTShareMgntClient> shareMgntClient (EacServiceAccessConfig::getInstance()->sharemgntHost, EacServiceAccessConfig::getInstance()->sharemgntPort);

        // 获取当前三权分立是否开启，若开启则用安全管理员账号设置，若未开启则用超级管理员账号
        bool bStatus = shareMgntClient->Usrm_GetTriSystemStatus();
        string handlerID = g_ShareMgnt_constants.NCT_USER_ADMIN;
        if (bStatus) {
            handlerID = g_ShareMgnt_constants.NCT_USER_SECURIT;
        }

        // 获取用户管理的所有部门
        vector<ncTRoleMemberInfo> roleInfos;
        shareMgntClient->UsrRolem_GetMember(roleInfos, handlerID, g_ShareMgnt_constants.NCT_SYSTEM_ROLE_ORG_MANAGER);

        // 判断用户是否为组织管理员，且管理所有的部门
        bool bUserIsManager = false;
        for(auto iter = roleInfos.begin(); iter != roleInfos.end(); iter++) {
            if (iter->userId == userId) {
                    userInfo = *iter;
                    bUserIsManager = true;
                    break;
            }
        }

        if (!bUserIsManager) {
            THROW_E (EAC_HTTP_SERVER, EACHTTP_CANCEL_MANAGER_ERROR, "The user is not the responsible person for the department");
        }

        // 获取所有的部门信息
        auto manageDepsNames = userInfo.manageDeptInfo.departmentNames;
        for (auto iter = manageDepsNames.begin(); iter != manageDepsNames.end(); iter++) {
            if (strDepNames != "") {
                strDepNames += ",";
            }
            strDepNames += *iter;
        }

        shareMgntClient->UsrRolem_DeleteMember (handlerID,
                                                g_ShareMgnt_constants.NCT_SYSTEM_ROLE_ORG_MANAGER,
                                                userId);
    }
    catch (ncTException & e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_CANCEL_MANAGER_ERROR, e.expMsg.c_str ());
    }
    catch (TException & e) {
        THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_UNKNOWN_ERR, e.what ());
    }

    //返回结果
    string body = "";
    ncHttpSendReply (cntl, brpc::HTTP_STATUS_OK, "ok", body);

    String msg,exmsg;
    msg.format (ncEACHttpServerLoader, _T("IDS_EACHTTP_APP_CANCEL_MANAGER_SUCCESS"), strDepNames.c_str(), userInfo.displayName.c_str());

    ncEACHttpServerUtil::Log (cntl, info.userId, info.visitorType, ncTLogType::NCT_LT_MANAGEMENT, ncTLogLevel::NCT_LL_INFO,
                             ncTManagementType::NCT_MNT_SET, msg.getCStr(), exmsg.getCStr());

    NC_EAC_HTTP_SERVER_TRACE (_T("this: %p, cntl: %p, end"), this, cntl);
}
