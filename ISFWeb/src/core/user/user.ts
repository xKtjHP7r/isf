import session from '@/util/session';
import __ from './locale'

/**
 * 用户认证类型
 */
export enum UserType {
    LocalUser = 1,
    DomainUser = 2,
    ThirdUser = 3,
}

export enum UserStringType {
    LocalUser = 'local',
    DomainUser = 'domain',
    ThirdUser = 'third',
}

/**
 * 用户信息枚举（getUserInfoCache）
 */
export enum UserInfo {
    /**
     * 显示名
     */
    displayName = 'displayName',

    /**
     * id
     */
    id = 'id',

    /**
     * 角色
     */
    roles = 'roles',
}

/**
 * 所选用户
 */
export interface UserInfoType {
    id: string;
    name: string;
    type: string;
}

/**
 * 从session获取获取当前登录用户信息{ displayName, id, roles }
 */
export const getUserInfoCache = (params: ReadonlyArray<UserInfo> | UserInfo): ReadonlyArray<string> | string | ReadonlyArray<{ id: string }> => {
    if (Array.isArray(params)) {
        return params.map((param) => getUserInfoCache(param))
    }

    const userinfo = session.get('isf.userInfo')

    switch (params) {
        case UserInfo.displayName:
            return userinfo && userinfo.user && userinfo.user.displayName

        case UserInfo.id:
            return userinfo && userinfo.id

        case UserInfo.roles:
            return userinfo && userinfo.user.roles

        default:
            return ''
    }
}

/**
 * 判断用户认证类型
 */
export function getUserType(usertype): string {
    switch (usertype) {
        case UserType.DomainUser:
            return __('域用户');
        case UserType.LocalUser:
            return __('本地用户');
        case UserType.ThirdUser:
            return __('外部用户');
        default:
            return __('未知');
    }
}

export function getUserStringType(usertype): string {
    switch (usertype) {
        case UserStringType.DomainUser:
            return __('域用户');
        case UserStringType.LocalUser:
            return __('本地用户');
        case UserStringType.ThirdUser:
            return __('外部用户');
        default:
            return __('未知');
    }
}

/**
 * 新建/编辑用户错误类型
 */
export enum ValidateState {
    /**
     * 正常无错误
     */
    Normal,

    /**
     * 输入不为空
     */
    Empty,

    /**
     * 用户名不合法
     */
    NameInvalid,

    /**
     * 用户名与应用账户同名
     */
    SameWithUserAccount,

    /**
     * 用户名已存在
     */
    NameExist,

    /**
     * 显示名不合法
     */
    DisplayNameInvalid,

    /**
     * 显示名已存在
     */
    DisplayNameExist,

    /**
     * 显示名已被文档库占用
     */
    DisplayNameUsed,

    /**
     * 用户编码不合法
     */
    CodeInvalid,

    /**
     * 岗位不合法
     */
    PositionInvalid,

    /**
     * 用户编码已存在
     */
    UserCodeExit,

    /**
     * 部门编码已存在
     */
    DpCodeExit,

    /**
     * 组织编码已存在
     */
    OrgCodeExit,

    /**
     * 备注不合法
     */
    RemarksInvalid,

    /**
     * 邮箱不合法
     */
    EamilInvalid,

    /**
     * 手机号不合法
     */
    PhoneInvalid,

    /**
     * 身份证不合法
     */
    IdCardInvalid,

    /**
     * 存储位置是否禁用
     */
    OssInfoDisabled,

    /**
     * 部门名称不合法
     */
    DepartmentInvalid,

    /**
     * 组织名称不合法
     */
    OrgInValid,

    /**
     * 部门名称已存在
     */
    DepartmentExist,

    /**
     * 组织名称已存在
     */
    OrgExist,

    /**
     * 邮箱已存在
     */
    EmailExist,
}

/**
 * 对象信息
 */
export interface OssInfo {
    /**
     *  对象存储ID
     */
    ossId: string;
    /**
     *  对象存储名称
     */
    ossName: string;

    /**
     * 对象存储状态
     */
    enable: boolean;
}

/**
* 密级信息
*/
export interface CsfOptions {
    /**
    * 密级等级
    */
    level: number;

    /**
    * 密级名称
    */
    text: string;
}

/**
 * 选择的部门信息
 */
export interface Dep {
    /**
     * 邮箱
     */
    email: string;

    /**
     * 选择的部门id
     */
    id: string;

    /**
     * 选择的部门名称
     */
    name: string;

    /**
     * 部门编码
     */
    code: string;

    /**
     * 部门负责人
     */
    managerInfo: UserInfoType[];

    /**
     * 备注
     */
    remark: string;

    /**
     * 状态
     */
    status: boolean;

    /**
     * 部门管理员
     */
    responsiblePersons: ReadonlyArray<any>;

    /**
     * 选择的部门归属站点
     */
    ossInfo: OssInfo;

    /**
     * 子部门数量
     */
    subDepartmentCount: number;

    /**
     * 部门用户数量
     */
    subUserCount: number;
}

/**
* 新建部门信息
*/
export interface DepartmentInfo {
    /**
     * 部门名称
     */
    departName: string;

    /**
     * 部门编码
     */
    code: string;

    /**
     * 备注
     */
    remark: string;

    /**
     * 状态
     */
    status: boolean;

    /**
     * 邮箱
     */
    email: string;

    /**
     * 存储位置
     */
    ossInfo: OssInfo;
    /**
     * 上级部门名称
     */
    parentName: string;

    /**
     * 上级部门id
     */
    parentId?: string;

    /**
     * 上级部门类型
     */
    parentType?: string;
}

/**
 * 新建组织信息
*/
export interface OrganizeInfo {
    /**
     * 组织名称
     */
    orgName: string;

    /**
     * 组织编码
     */
    code: ValidateState;

    /**
     * 备注
     */
    remark: ValidateState;

    /**
     * 状态
     */
    status: boolean;

    /**
     * 邮箱
     */
    email?: string;

    /**
     * 存储位置
     */
    ossInfo: OssInfo;
}

/**
 * 新建部门成功回调对象
*/
export interface NodeInfo {
    id: string;
    name: string;
    departmentId?: string;
    departmentName?: string;
    organizationId?: string;
    organizationName?: string;
    isOrganization?: boolean;
    responsiblePerson?: ReadonlyArray<any>;
    ossInfo: OssInfo;
    email: string;
}

/**
 * 用户组织错误信息
*/
export const ValidateMessages = {
    [ValidateState.Empty]: __('此项不允许为空。'),

    [ValidateState.NameInvalid]: __('用户名不能包含 空格 或 \\ / * ? " < > | 特殊字符，长度不能超过128个字符。'),

    [ValidateState.NameExist]: __('该用户名已被占用。'),

    [ValidateState.SameWithUserAccount]: __('用户名不能和应用帐户名重名。'),

    [ValidateState.DisplayNameInvalid]: __('显示名不能包含 \\ / * ? " < > | 特殊字符，长度不能超过128个字符。'),

    [ValidateState.DisplayNameExist]: __('该显示名已被用户占用。'),

    [ValidateState.DisplayNameUsed]: __('该显示名已被文档库占用。'),

    [ValidateState.CodeInvalid]:__('长度不能超过255字符，只支持大小写英文，数字，下划线和横线。'),

    [ValidateState.UserCodeExit]:__('用户编码已存在，请重新输入。'),

    [ValidateState.DpCodeExit]:__('部门编码已存在，请重新输入。'),

    [ValidateState.OrgCodeExit]:__('组织编码已存在，请重新输入。'),

    [ValidateState.PositionInvalid]: __('长度不能超过50个字符。'),

    [ValidateState.RemarksInvalid]: __('备注不能包含 \\ / : * ? " < > | 特殊字符，长度不能超过128个字符。'),

    [ValidateState.EamilInvalid]: __('邮箱地址只能包含 英文、数字 及 @-_. 字符，格式形如 XXX@XXX.XXX，长度范围 5~100 个字符。'),

    [ValidateState.PhoneInvalid]: __('手机号只能包含 数字，长度范围 1~20 个字符。'),

    [ValidateState.IdCardInvalid]: __('请输入正确的身份证号。'),

    [ValidateState.OssInfoDisabled]: __('所指定的存储位置已不可用，请更换。'),

    [ValidateState.DepartmentInvalid]: __('部门名称不能包含 \\ / : * ? " < > | 特殊字符，长度不能超过128个字符。'),

    [ValidateState.OrgInValid]: __('组织名称不能包含 \\ / : * ? " < > | 特殊字符，长度不能超过128个字符。'),

    [ValidateState.DepartmentExist]: __('该部门名称已被占用。'),

    [ValidateState.OrgExist]: __('该组织名称已被占用。'),

    [ValidateState.EmailExist]: __('该邮箱已被占用。'),

}

/**
 * 失焦弹框
*/
export enum Type {
    /**
     * 用户名
    */
    LoginName,

    /**
     * 邮箱
    */
    Email,

    /**
     * 显示名
    */
    DisplayName,

    /**
     * 用户编码
     */
    Code,

    /**
     * 岗位
     */
    Position,

    /**
     * 备注
    */
    Remark,

    /**
     * 手机号
    */
    TelNumber,

    /**
     * 身份证
    */
    IdcardNumber,

    /**
     * 配额空间
    */
    Space,
}
