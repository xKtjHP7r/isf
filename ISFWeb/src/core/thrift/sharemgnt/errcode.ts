export const enum ErrorCode {
    /**
     * 日期已过期
     */
    DateExpired = 20005,

    /**
     * 导入组织信息内容错误
     */
    UserCreateError = 20304,

    /**
     * 用户名为空或者不合法
     */
    InvalidUserName = 20101,

    /**
     * 显示名为空或者不合法
     */
    InvalidDisplayName = 20102,

    /**
     * 邮箱地址格式不正确
     */
    InvalidEmail = 20103,

    /**
     * 用户名已被占用
     */
    UserNameExist = 20105,
    /**
     * 验证码过期
     */
    VcodeExpired = 20150,
    /**
     * 验证码输入错误
     */
    VcodeError = 20151,
    /**
     * 显示名已被占用
     */
    DisplayNameExist = 20123,
    /**
     * 密码失效了
     */
    PwdInValid = 20127,
    /**
     * 密码安全系数过低
     */
    PwdUnSafe = 20128,
    /**
     * 密码已过期
     */
    PwdExpired = 20137,
    /**
     * 第一次输入密码错误
     */
    PwdFirstError = 20131,
    /**
     * 第二次输入密码错误
     */
    PwdSecError = 20132,
    /**
     * 账号被锁定
     */
    AccountLocked = 20130,
    /**
     * 连续3次输入错误密码
     */
    ContinuousErrPwd3Times = 20135,
    /**
     * 用户名或密码不正确
     */
    AccountOrPwdInError = 20108,

    /**
     * 用户不存在
     */
    UserNotExist = 20110,

    /**
     * 输入错误次数超过限制
     */
    OverErrCount = 20143,
    OverErrCount2 = 20144,

    /**
     * 直属部门已不存在
     */
    ParentDepartmentNotExist = 20201,

    /**
     * 域不存在
     */
    DomainNotExists = 20402,

    /**
     * 域已存在
     */
    DomainAlreadyExists = 20403,

    /**
     * 域名不正确
     */
    DomainNameIncorrect = 20418,

    /**
     * 限速对象未设置
     */
    LimitRateObjectNotSet = 21901,
    /**
     * 该条限速配置不存在
     */
    LimitRateNotExist = 21902,
    /**
     * 限速值不合法
     */
    InvalidLimitRateValues = 21903,
    /**
     * 限速类型不合法
     */
    InvalidLimitRateType = 21904,
    /**
     * 只允许设置一个限速对象
     */
    OnlyOneLimitRateObject = 21905,
    /**
     * 用户已存在于列表中
     */
    LimitUserExist = 21906,
    /**
     * 部门已存在于列表中
     */
    LimitDepartExist = 21907,
    /**
     * 至少设置一种最大传输速度
     */
    AtLeastSetOneSpeed = 21908,

    /**
     * 普通用户登录控制台
     */
    LimitUserLogin = 20414,

    /**
     * 接收区名称不合法
     */
    InvalidRecvAreaName = 22801,

    /**
     * 接收区已存在
     */
    RecvAreaExist = 22802,

    /**
     * 接收区不存在
     */
    RecvAreaNotExist = 22803,

    /**
     * 创建发送目录失败
     */
    CreateSendDirError = 22804,

    /**
     * 接收区名称为空
     */
    RecvAreaNameIsEmpty = 22805,

    /**
     * 不允许为初始密码登录
     */
    CannotUseInitPwd = 20129,

    /**
     * 用户被禁用
     */
    UserDisabled = 20109,

    /**
     * 用户名不可用
     */
    UserNameDisabled = 20153,

    /**
     * 用户名已被管理员占用
     */
    UserByAdminExist = 20154,

    /**
     * 邮箱已被占用
     */
    EmailExist = 20106,

    /**
     * 手机号不正确
     */
    InvalidPhoneNub = 22603,

    /**
     * 手机号已被占用
     */
    PhoneNubExist = 22604,

    /**
     * 输入正确的身份证号
     */
    InvalidCardId = 20159,

    /**
     * 备注不合法
     */
    InvalidRemarks = 20158,

    /**
     * 身份证号已被占用
     */
    CardIdExist = 20160,

    /**
     * 配额空间不合法
     */
    InvalidUserSpace = 20169,

    /**
     * 组织名不合法
     */
    InvalidDepartName = 20206,

    /**
     * 组织名已存在
     */
    OrgNameExist = 20205,

    /**
     * 上级部门已不存在
     */
    DepNameNotExist = 20215,

    /**
    * 部门名已存在
    */
    DepNameExist = 20202,

    /**
     * 用户密级不合法
     */
    InvalidSecret = 20901,

    /*
    * 自动清理策略不存在
     * 连接LDAP服务器失败
     */
    DomainUnavailable = 20411,

    /**
     * 备用域与首选域不在同一个域内
     */
    DomainsNotInOneDomain = 20421,

    /**
     * 备用域地址不能和主域地址相同
     */
    SpareAddressDuplicateWithMainDomain = 20423,

    /**
     * 自动清理策略不存在
     */
    DocsAutoCleanStrategyNotExist = 23106,

    /**
     * 审核员无效
     */
    InvalidApprover = 21504,

    /**
     * 创建审核流程，适用范围无效
     */
    InvalidApproveRange = 21516,

    /**
     * 不能修改已存在用户的初始密码
     */
    InvalidChangePwd = 20305,

    /**
     * 正在导入用户
     */
    ImportTaskExist = 20302,

    /**
     * 正在导出用户
     */
    ExportTaskExist = 20301,

    /**
     * 存储位置不存在
     */
    OSSNotExist = 24404,

    /**
     * 存储位置已禁用
     */
    OSSDisabled = 24405,

    /**
    * 存储位置不可用
    */
    OSSInvalid = 22308,

    /**
     * 存储位置不可用
     */
    OSSUnabled = 20302,

    /**
     * 用户状态输入不合法
     */
    InvalidUserStatus = 20170,

    /**
     * 显示名被文档库占用
     */
    NameOccupiedByDoc = 20171,

    /**
     * 用户编码错误
     */
    InvalidUserCode = 20172,

    /**
     * 部门编码错误
     */
    InvalidDpCode = 20173,

    /**
     * 用户编码已存在
     */
    UserCodeExit = 20174,

    /**
     * 部门编码已存在
     */
    DpCodeExit = 20175,

    /**
     * 用户密级不能高于系统密级
     */
    InvalidCsfLevel = 20903,

    /**
     * 当前用户管理可分配空间已超出限制
     */
    LimitAssignUserSpace = 20138,

    /**
     * 无效的密码
     */
    InvalidPwd = 20125,

    /**
     * 组织管理员无法编辑自身
     */
    CannotEditUsers = 20308,

    /**
     * 个人文档不存在时，导入组织信息错误
     */
    UsersDocNotExist = 8199,

    /**
     * 实名共享策略信息不存在的时候，编辑策略错误
     */
    StrategyNotExist = 20804,

    /**
     * 移动部门时，部门不存在
     */
    DepNotExist = 20210,

    /**
     * 移动部门，目标部门不存在
     */
    TargetDepNotExist = 20211,

    /**
     * 目标部门包含同名的子部门
     */
    TargetDepIncludeSameNameDep = 20212,

    /**
     * 不能重复导入组织
     */
    ImportDomainAgain = 20409,

    /*
     * 添加用户至部门，部门或组织不存在
     */
    DepOrOrgNotExist = 20215,

    /**
     * 编辑组织，组织不存在
     */
    OrgNameNotExist = 20204,

    /**
     * 部门名不合法
     */
    InvalidDepName = 20203,

    /**
     * 文档流程不存在
     */
    ProcessNotExist = 21507,

    /**
     * 同步关键字设置主域不存在
     */
    SetDomainKeyConfigDomainInvalid = 20402,

    /**
    * 部门组织排序时目标部门不存在
    */
    SortTargetDepNotExist = 20223,

    /**
     * 启用用户数已达用户许可总数的上限
     */
    EnableUserCountOverproof = 20518,

    /**
     * 账户已过期
     */
    CountPassDue = 20157,

    /**
     * 导出正在处理
     */
    ExportProcessing = 23404,

    /**
     * 该类型的文档库不支持导出功能
     */
    ExportNotSupport = 23405,

    /**
     * 未设置SMTP邮箱服务器
     */
    ServerNotExistForSMTP = 20807,

    /*
     * 同步目标部门不存在
     */
    SyncTargetNotExist = 20415,

    /**
     * 备用域地址相同
     */
    SameSpareDomain = 20422,

    /**
     * 域用户或部门搜索请求数据过大
     */
    RequestDataLarge = 20224,

    /**
     * 未完成示名认证
     */
    NotrealNameAuth = 20208,

    /**
     * efast服务不可用
     */
    ServerError = 99,
}

/**
 * 需要弹出修改密码窗口的错误码
 */
export const ErrcodeNeedChangePassword = [
    // 密码已过期
    ErrorCode.PwdInValid,

    // 密码不安全
    ErrorCode.PwdUnSafe,
]