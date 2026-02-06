declare namespace Core {
    namespace ShareMgnt {

        /**
         * 角色信息
         */
        type ncTRoleInfo = {
            /**
             * 创建者id
             */
            creatorId: string;

            /**
             * 描述
             */
            description: string;

            /**
             * 显示名
             */
            displayName: string;

            /**
             * 角色id标识
             */
            id: string;

            /**
             * 名称
             */
            name: string;
        }

        /**
         * 部门信息
         */
        type ncTUsrmDepartmentInfo = {

            /**
             * 部门id
             */
            departmentId: string;

            /**
             * 部门名称
             */
            departmentName: string

            /**
             * 父部门id
             */
            parentDepartId: string;

            /**
             * 父部门名称
             */
            parentDepartName: string;

            /**
             * 部门管理员
             */
            responsiblePerson: ReadonlyArray<ncTUsrmGetUserInfo>;

            /**
             * 归属站点信息
             */
            ossInfo: ncTUsrmOSSInfo;

            /**
             * 子部门id        
             */
            subDepartIds?: ReadonlyArray<string>;

            /**
             * 部门邮箱
             */
            email: string;

            /**
             * 父目录路径
             */
            parentPath: string;
        }

        /**
         * 用户节点信息
         */
        type ncTUsrmGetUserInfo = {
            /**
             * 用户id
             */
            id: string;
            /**
             * 用户基本信息
             */
            user: ncTUsrmUserInfo;
            /**
             * 是否为初始密码      
             */
            originalPwd: boolean;
            /**
             * 用户密码   
             */
            password: string;
            /**
             * 直属部门信息
             */
            directDeptInfo: any;
        }

        /**
         * 部门（组织）节点信息
         */
        type ncTDepartmentInfo = {
            /**
             * 是否是组织节点
             */
            isOrganization?: boolean;

            /**
             * 部门id
             */
            id: string;

            /**
             * 部门名称
             */
            name: string;

            /**
             * 部门管理员
             */
            responsiblePersons: ReadonlyArray<ncTUsrmGetUserInfo>;

            /**
             * 子部门数量
             */
            subDepartmentCount: number;

            /**
             * 子用户数量
             */
            subUserCount: number;

            /**
             * 存储位置信息
             */
            ossInfo: ncTUsrmOSSInfo;

            /**
             * 部门邮箱
             */
            email: string;
        }

        /**
         *  归属站点信息
         */
        type ncTUsrmOSSInfo = {
            /**
            * 对象存储ID
            */
            ossId: string;

            /**
             * 对象存储名称
             */
            ossName: string;

            /**
             * 对象存储状态
             */
            enabled: boolean;
        }
        /**
         *  用户基本信息
         */
        type ncTUsrmUserInfo = {
            /**
             *  用户名称
             */
            loginName: string;
            /**
             *  显示名
             */
            displayName?: string;
            /**
             *  备注
             */
            remark?: string;
            /**
             *  邮箱
             */
            email?: string;
            /**
             *  身份证号
             */
            idcardNumber?: string;
            /**
            * 用户类型
            * enum ncTUsrmUserType {
            *   NCT_USER_TYPE_LOCAL = 1,        // 本地用户
            *   NCT_USER_TYPE_DOMAIN = 2,       // 域用户
            *   NCT_USER_TYPE_THIRD = 3,        // 第三方验证用户
            *   }
             */
            userType?: number;

            /**
             *  所属部门id，-1时表示为未分配用户          
             */
            departmentIds: Array<string>;

            /**
             *  所属部门名称
             */
            departmentNames?: Array<string>;
            /**
             * // 用户状态
             * enum ncTUsrmUserStatus {
             *      NCT_STATUS_ENABLE = 0,          // 启用
             *      NCT_STATUS_DISABLE = 1,         // 禁用
             *      NCT_STATUS_DELETE = 2,          // 用户被第三方系统删除
             * }
             */
            status?: number;

            /**
             *  排序优先级 
             */
            priority?: number;

            /**
             *  用户密级
             */
            csfLevel?: number;

            /**
             *  密码管控
             */
            pwdControl: boolean;

            /**
             *  归属站点信息   
             */
            ossInfo?: ncTUsrmOSSInfo;

            /**
             *  用户创建时间
             */
            createTime?: number;

            /**
             *  用户冻结状态，true:冻结 false:未冻结                    
             */
            freezeStatus?: boolean;

            /**
             *  手机号
             */
            telNumber?: string;

            /**
             * 用户所有角色
             */
            roles?: ncTRoleInfo;
            /**
             * 有效期限
             */
            expireTime?: number;
        }

        /**
         * 管辖部门信息
         */

        type ncTManageDeptInfo = {
            /**
             * 所属部门id，-1时表示为未分配用户
             */
            departmentIds: Array<string>;

            /**
             * 所属部门名称
             */
            departmentNames: Array<string>;
        }

        /**
         * 角色成员信息
         */
        type ncTRoleMemberInfo = {
            /**
             * 用户id
             */
            userId: string;

            /**
             * 显示名
             */
            displayName: string;

            /**
             * 所属部门id，-1时表示为未分配用户
             */
            departmentIds: Array<string>;

            /**
             * 所属部门名称
             */
            departmentNames: Array<string>;

            /**
             * 管辖部门信息
             */
            manageDeptInfo: ncTManageDeptInfo;

            /**
             * 审核对象
             */
            auditObj: ncTAuditObject;

        }

        /**
         * 审核员审核对象
         */
        type ncTAuditObject = {
            /**
             * 审核员审核对象类型
             * 1:用户， 2：部门， 3：文档库， 4：归档库
             */
            objType: number;

            /**
             * 审核对象id
             */
            objId: string;

            /**
             * 审核对象名称
             */
            objName: string;
        }

        /**
         * 组织节点信息
         */
        type ncTRootOrgInfo = {
            /**
             * 是否是组织，true表示组织，false表示部门
             */
            isOrganization: boolean;

            /**
             * id
             */
            id: string;

            /**
             * 名称
             */
            name: string;

            /**
             * 管理员
             */
            responsiblePersons: Array<ncTUsrmGetUserInfo>;

            /**
             * 子部门数量
             */
            subDepartmentCount: number;

            /**
             * 子用户数量
             */
            subUserCount: number;

            /**
             * 归属站点信息
             */
            ossInfo: ncTUsrmOSSInfo;

            /**
             * 组织邮箱地址
             */
            email: string;
        }

        /**
         * 搜索时返回的用户信息
         */
        type ncTSearchUserInfo = {
            /**
             * 用户id
             */
            id: string;

            /**
             * 用户登录名
             */
            loginName: string;

            /**
             * 用户显示名
             */
            displayName: string;

            /**
             * 用户密级
             */
            csfLevel: number;

            /**
             * 所属部门id，-1时表示为未分配用户
             */
            departmentIds: Array<string>;

            /**
             * 所属部门名称
             */
            departmentNames: Array<string>;
        }

        /**
         * 登录验证码配置参数
         */
        type ncTVcodeConfig = {

            /**
             * 开启关闭登录验证码功能。 true - 开启，false - 关闭。
             */
            isEnable: boolean;

            /**
             *  达到开启登录验证码的用户密码出错次数
             */
            passwdErrCnt: number;
        }

        /**
         * 生成的验证码信息
         */
        type ncTVcodeCreateInfo = {
            /**
             * 经过 base64 编码后的验证码图片字符串
             */
            vcode: string;

            /**
             * 验证码唯一标识
             */
            uuid: string;
        }

        /**
         * 用户登录附带选项信息
         */
        interface ncTUserLoginOption {
            /**
             * 用户登录
             */
            loginIp?: string;

            /**
             * 验证码唯一标识
             */
            uuid?: string

            /**
             * 验证码唯一标识
             */
            vcode?: string;
        }

        /**
         * 定义简单的用户信息
         */
        type ncTSimpleUserInfo = {
            id: string
            displayName: string
            loginName: string
            status: ncTUsrmUserStatus
        }

        /**
         * 用户状态
         */
        enum ncTUsrmUserStatus {
            /**
             * 启用
             */
            NCT_STATUS_ENABLE = 0,

            /**
             * 禁用
             */
            NCT_STATUS_DISABLE = 1,

            /**
             * 用户被第三方系统删除
             */
            NCT_STATUS_DELETE = 2,
        }

        /**
         * 文档审核模式
         */
        enum ncTDocAuditType {
            /**
             * 同级审核，一个人审核通过即可
             */
            NCT_DAT_ONE = 1,
            /**
             * 汇签审核，全部通过才算通过
             */

            NCT_DAT_ALL = 2,
            /**
             * 逐级审核，一级一级通过
             */

            NCT_DAT_LEVEL = 3,

            /**
             * 免审核
             */
            NCT_DAT_FREE = 4,
        }

        /**
         * 创建流程参数
         */
        type ncTDocAuditInfo = {
            /**
             * 流程id
             */
            processId?: string;

            /**
             * 流程名称
             */
            name: string;

            /**
             * 审核模式
             */
            auditType: ncTDocAuditType;

            /**
             * 如果是同级/汇签审核，表示所有的审核员；如果是逐级审核，auditorIds[0]表示第一级审核，依次类推
             */
            auditorIds: Array<string>;

            /**
             * 审核通过后，存放的gns路径
             */
            destDocId: string;

            /**
             * 创建者id
             */
            creatorId?: string;

            /**
             * 有效状态
             */
            status?: boolean;

            /**
             * 审核员名称
             */
            auditorNames?: Array<string>;

            /**
             * 创建者名称
             */
            creatorName?: string;

            /**
             * 文档路径名称
             */
            destDocName?: string;

            /**
             * 流程适用的范围，限定哪些部门，哪些人可以看到该流程
             */
            accessorInfos?: Array<{ 'ncTAccessorInfo': ncTAccessorInfo }>
        }

        /**
         * 访问者信息
         */
        type ncTAccessorInfo = {
            /**
             * 访问者id
             */
            id: string;

            /**
             * 访问者类型 1:用户, 2:部门
             */
            type: number;

            /**
             * 访问者名称
             */
            name: string
        }

        /**
         * 活跃报表信息
         */
        type ActiveReportInfo = {
            /**
             * 用户总数
             */
            totalCount: number;

            /**
             * 平均用户数
             */
            avgCount: number;

            /**
             * 平均活跃度
             */
            avgActivity: number;

            /**
             * 活跃用户数信息
             */
            userInfos: ReadonlyArray<ncTActiveUserInfo>;
        };

        /**
         * 活跃用户信息
         */
        type ncTActiveUserInfo = {
            /**
             * 时间
             */
            time: string;

            /**
             * 用户活跃数
             */
            activeCount: number;

            /**
             * 用户活跃度
             */
            userActivity: number;
        }

        /**
         * 第三方根节点信息
         */
        type ncTThirdPartyRootNodeInfo = {
            /**
             * 跟组织名称
             */
            name: string;

            /**
             * 根组织第三方id
             */
            thirdId: string;
        }

        /**
         * 第三方节点信息
         */
        type ncTUsrmThirdPartyNode = {
            /**
             * 组织
             */
            ncTUsrmThirdPartyOUs: ReadonlyArray<ncTUsrmThirdPartyOU>

            /**
             * 用户
             */
            ncTUsrmThirdPartyUsers: ReadonlyArray<ncTUsrmThirdPartyUser>
        }

        /**
         * 第三方用户信息
         */
        type ncTUsrmThirdPartyUser = {
            /**
             * 登录名
             */
            loginName: string;

            /**
             * 显示名
             */
            displayName: string;

            /**
             * 第三方用户id
             */
            thirdId: string;

            /**
             * 第三方父部门id
             */
            deptThirdId: string;
        }

        /**
         * 第三方组织单位信息
         */
        type ncTUsrmThirdPartyOU = {
            /**
             * 组织名称
             */
            name: string,

            /**
             * 第三方部门id
             */
            thirdId: string;

            /**
             * 第三方父部门id
             */
            parentThirdId: string;

            /**
             * 是否导入组织下的所有子组织及用户，True--导入，False--只导入此组织
             */
            importAll: boolean;
        }

        /**
         * 导入的内容
         */
        type ncTUsrmImportContent = {
            /**
             * 选择导入目标所属的域控信息
             */
            domain: ncTUsrmDomainInfo;

            /**
             * 勾选的域控，当domainName不为None时，代表导入整个域用户，此时users和ous为None
             */
            domainName: string;

            /**
             * 勾选的域用户
             */
            users: ReadonlyArray<ncTUsrmDomainUser>,

            /**
             * 勾选的域组织
             */
            ous: ReadonlyArray<ncTUsrmDomainOU>
        }

        /**
         * 用户导入的配置选项
         */
        type ncTUsrmImportOption = {
            /**
             * 是否导入用户邮箱
             */
            userEmail: boolean;

            /**
             * 是否导入用户显示名
             */
            userDisplayName: boolean;

            /**
             * 是否覆盖已有用户
             */
            userCover: boolean;

            /**
             * 导入目的地
             */
            departmentId: string;

            /**
             * 用户密级
             */
            csfLevel: number;
        }

        type ncTUsrmImportResult = {

            /**
             * 需要导入的总数
             */
            totalNum: number;

            /**
             * 已导入的总数
             */
            successNum: number;

            /**
             * 出错总数
             */
            failNum: number;
            /**
             * 出错内容
             */
            failInfos: ReadonlyArray<string>
        }

        /**
         * 插件类型
         */
        enum ncTPluginType {
            /**
             * 认证插件
             */
            AUTHENTICATION = 0,

            /**
             * 消息推送插件
             */
            MESSAGE = 1
        }

        type pluginType = {
            /**
             * 插件类型
             */
            type: ncTPluginType;
        }

        /**
         * 第三方应用配置
         */
        type ncTThirdPartyConfig = {

            /**
             * 唯一索引
             */
            indexId?: number;

            /**
             * 唯一标识第三方认证系统
             */
            thirdPartyId: string;

            /**
             * 第三方认证系统名称
             */
            thirdPartyName: string;

            /**
             * 需要单独配置的信息，采用json string来保存
             */
            config: string;

            /**
             * 内部配置，不开放
             */
            internalConfig: string;

            /**
             * 开启状态
             */
            enabled: boolean;

            /**
             * 插件信息
             */
            plugin: pluginType;
        }

        /**
         * 第三方预览工具配置信息
         */
        type ncTThirdPartyToolConfig = {
            /**
             * 第三方工具标识
             */
            thirdPartyToolId: string;

            /**
             * 开启状态
             */
            enabled: boolean;

            /**
             * url
             */
            url: string;

            /**
             * 第三方工具名称
             */
            thirdPartyToolName: string;
        }

        /**
         * 导入用户信息失败记录错误信息
         */
        type ncTImportFailInfo = {
            /**
             * 错误码
             */
            errorID: number;

            /**
             * 错误信息
             */
            errorMessage: ReadonlyArray<string>;

            /**
             * 错误信息的索引
             */
            index: number;

            /**
             * 导入组织相关信息
             */
            userInfo: ReadonlyArray<ncTUsrmUserInfo>
        }

        /** 
        * 备用域控信息
        */
        type ncTUsrmFailoverDomainInfo = {
            /**
             * 域id
             */
            id: number;

            /**
             * 首选域id
             */
            parentId: number;

            /**
             * 域控制器地址
             */
            address: string;

            /**
             * 域端口
             */
            port: number;

            /**
             * 管理员账号
             */
            adminName: string;

            /**
             * 管理员密码（密文）
             */
            password: string;

            /**
             * 是否使用ssl连接，true-使用，false-不使用,默认不启用
             */
            useSSL: boolean;
        }

        /**
         * 安全连接选项
         */
        type SafeMode = {
            /**
             * 默认值，无
             */
            Default,

            /**
             * SSL/TSL
             */
            SslOrTsl,

            /**
             * STARTTLS
             */
            Starttls,
        }

        /*
         * SMTP配置信息
         */
        type ncTSmtpSrvConf = {
            /**
             * 邮件服务器
             */
            server: string,

            /**
             * 安全连接
             */
            safeMode: SafeMode,

            /**
             * 端口
             */
            port: number,

            /**
             * open relay
             */
            openRelay: boolean,

            /**
             * 邮件地址
             */
            email: string,

            /**
             * 邮件密码
             */
            password: string,
        }

        /**
         * 获取密码管控信息
         */
        type ncTUsrmGetPwdControl = {
            /**
             * 锁定状态
             */
            lockStatus: boolean;

            /**
             * 密码
             */
            password: string;

            /**
             * 是否密码管控
             */
            pwdControl: boolean;
        }

        type ncTUsrmGetPwdConfig = {
            /**
             * 密码状态
             */
            strongStatus: boolean;

            /**
             * 密码有效期(单位:天), -1: 永不失效
             */
            expireTime: number;

            /**
             * 锁定状态
             */
            lockStatus: boolean;

            /**
             * 密码错误次数
             */
            passwdErrCnt: number;

            /**
             * 密码锁定时间(以分钟为单位)
             */
            passwdLockTime?: number;

            /**
             * 强密码的最小长度
             */
            strongPwdLength: number;
        }

        /**
         * 设置密码管控
         */
        type ncTUsrmSetPwdControlConfig = {
            /**
             * 用户锁定状态
             */
            lockStatus: boolean,

            /**
             * 密码
             */
            password: string,

            /**
             * 是否启用密码管控
             */
            pwdControl: boolean
        }

        /**
         * 新建用户信息
         */
        type ncTUsrmAddUserInfo = {
            /**
             * 用户基本信息
             */
            user: ReadonlyArray<ncTUsrmUserInfo>;

            /**
             * 密码
             */
            password: string;
        }

        /**
         * 共享范围添加策略用户信息
         */
        type ncTShareObjInfo = {
            /**
             * 用户id
             */
            id: string;

            /**
             * 用户显示名
             */
            name: string;

            /**
             * 用户部门id
             */
            parentId: string;

            /**
             * 用户部门名称
             */
            parentName: string;
        }

        /**
         * 共享范围策略信息
         */
        type ncTPermShareInfo = {
            /**
             * 策略ID
             */
            strategyId: string;

            /**
             * 共享者用户选项
             */
            sharerUsers: ReadonlyArray<ncTShareObjInfo>;

            /**
             * 共享者部门选项
             */
            sharerDepts: ReadonlyArray<ncTShareObjInfo>;

            /**
             * 共享范围用户选项
             */
            scopeUsers: ReadonlyArray<ncTShareObjInfo>

            /**
             * 共享范围部门选项
             */
            scopeDepts: ReadonlyArray<ncTShareObjInfo>

            /**
             * 策略开启的状态
             */
            status: boolean;
        }

        /**
         * 匿名共享限制信息
         */
        type ShareInfos = {

            /**
             * 共享者id
             */
            sharerId: string,

            /**
             * 共享者类型  
             */
            sharerType: number,

            /**
             * 共享者名称
             */
            sharerName: string,
        }

        /**
         * 获取域信息
         */
        type ncTUsrmDomainInfo = {
            /**
             * id
             */
            id: number;

            /**
             * 类型   1：主域，2：子域，3：信任域
             */
            type: number;

            /**
             * 父级id  主域时，为 -1，子域或者信任域时，为主域的id值
             */
            parentId: number;

            /**
             * 名称
             */
            name: string;

            /**
             * 域ip
             */
            ipAddress: string;

            /**
             * 域端口
             */
            port: number;

            /**
             * 管理员账号
             */
            adminName: string;

            /**
             * 管理员密码
             */
            password: string;

            /**
             * 状态 true-启用  false-禁用
             */
            status: boolean;

            /**
             * 同步状态, -1 关闭, 0 正向同步, 1 反向同步
             */
            syncStatus?: number;

            /**
             * 域配置信息
             */
            config?: ncTUsrmDomainConfig;

            /**
             * 是否使用ssl连接，true-使用，false-不使用,默认不启用
             */
            useSSL?: boolean;
        }

        /**
         * 域配置信息
         */
        type ncTUsrmDomainConfig = {
            /**
             * 要导入到的目的组织或部门id
             */
            destDepartId: string;

            /**
             * 设置域配置信息的时候，设为None
             */
            destDepartName: string;

            /**
             * 要导入的域组织路径，空为导入整个域
             */
            ouPath: ReadonlyArray<string>;

            /**
             * 同步时间间隔
             */
            syncInterval: number;

            /**
             * 同步方式
             */
            syncMode: ncTUsrmDomainSyncMode;

            /**
             * 用户默认创建状态，true:启用 false:禁用
             */
            userEnableStatus: boolean;

            /**
             * 强制同步方式
             */
            forcedSync: boolean;

            /**
             * 用户账号有效期(单位：天), 默认为 -1, 永久有效
             */
            validPeriod: number;

            /**
             * 用户密级
             */
            csfLevel: number;
        }

        /**
         * 域同步方式
         */
        enum ncTUsrmDomainSyncMode {
            /**
             * 同步对象包括上层组织结构
             */
            NCT_SYNC_UPPER_OU = 0,

            /**
             * 不同步上层组织结构
             */
            NCT_NOT_SYNC_UPPER_OU = 1,

            /**
             * 仅同步用户
             */
            NCT_SYNC_USERS_ONLY = 2,
        }

        /**
         * 域节点展开部门信息
         */
        type ncTUsrmDomainOU = {
            /**
             * 是否导入组织下的所有子组织及用户，True--导入，False--只导入此组织
             */
            importAll: boolean;

            /**
             * 组织名称
             */
            name: string;

            /**
             * 对象的GUID，具有唯一性
             */
            objectGUID: string;

            /**
             * 父部门的路径，格式：OU=研发部, DC=test2,DC=develop,DC=cn
             */
            parentOUPath: string;

            /**
             * 路径，格式：OU=研发部, DC=test2,DC=develop,DC=cn
             */
            pathName: string;

            /**
             * 组织负责人
             */
            rulerName: string;
        }

        /**
         * 域用户信息
         */
        type ncTUsrmDomainUser = {
            /**
             * 显示名
             */
            displayName: string;

            /**
             * dnPath
             */
            dnPath: string;
        }

        /*        
         * 在线用户数据类型
         */
        export interface OnlineDataType {
            /**
             * 当前时间用户数
             */
            count: string,

            /**
             * 当前时间
             */
            time: string
        }

        /**
         * 新建部门信息
         */
        type ncTAddDepartParam = {
            /**
             * 父节点id
             */
            parentId: string;

            /**
            * 部门名
            */
            departName: string;

            /**
             * 邮箱
             */
            email?: string;

            /**
             * 对象存储
             */
            ossId: string;
        }

        /**
         * 编辑部门信息
         */
        type ncTEditDepartParam = {
            /**
             * 部门id
             */
            departId: string;

            /**
            * 部门名
            */
            departName: string;

            /**
             * 邮箱
             */
            email: string;

            /**
             * 身份证号
             */
            idcardNumber: string;

            /**
             * 登录名
             */
            loginName: string;

            /**
             * objectGUID
             */
            objectGUID: string;

            /**
             * 所属部门的路径，格式：OU=研发部, DC=test2,DC=develop,DC=cn
             */
            ouPath: string;
        }
    }

    /**
     * 新建组织信息
     */
    type ncTAddOrgParam = {
        /**
         * 组织名
         */
        orgName: string;

        /**
         * 邮箱
         */
        email: string;

        /**
         * 对象存储
         */
        ossId: string;
    }

    /**
         * 编辑组织信息
         */
    type ncTEditOrgParam = {
        /**
         * 组织id
         */
        departId: string;

        /**
        * 组织名
        */
        departName: string;

        /**
         * 邮箱
         */
        email: string;

        /**
         * 对象存储
         */
        ossId: string;
    }

    /**
     * 获取免审核部门信息
     */
    type ncTGetInfosParam = {
        /**
         * 部门名
         */
        departName: string;

        /**
         * 部门id
         */
        departmentId: string;

        /**
         * 是否可用
         */
        isEnable: boolean;
    }

    /**
     * 流程信息
     */
    type Process = {
        /**
         * 适用范围
         */
        accessorInfos: ReadonlyArray<{ type: number, id: string, name: string }>

        /**
         * 审核模式
         */
        auditType: number

        /**
         * 审核员id
         */
        auditorIds: ReadonlyArray<string>

        /**
         * 审核员
         */
        auditorNames: ReadonlyArray<string>

        /**
         * 流程创建者id
         */
        creatorId: string

        /**
         * 流程创建者
         */
        creatorName: string

        /**
         * 文档最终保存位置id
         */
        destDocId: string

        /**
         * 文档最终保存位置
         */
        destDocName: string

        /**
         * 流程名称
         */
        name: string

        /**
         * 流程id
         */
        processId: string

        /**
         * 流程是否显示
         */
        status: boolean
    }

    /**
     * 域关键字字段
     */
    type ncTUsrmDomainKeyConfig = {
        /**
         * 部门名对应的域key字段
         */
        departNameKeys: Array<string>;
        /**
         * 部门ID对应的域key字段
         */
        departThirdIdKeys: Array<string>;
        /**
         * 登录名对应的域key字段
         */
        loginNameKeys: Array<string>;
        /**
         * 显示名对应的域key字段
         */
        displayNameKeys: Array<string>;
        /**
         * 用户邮箱对应的域key字段
         */
        emailKeys: Array<string>;
        /**
         * 用户Id对应的域key字段
         */
        userThirdIdKeys: Array<string>;
        /**
         * 安全组信息的key字段
         */
        groupKeys: Array<string>;
        /**
         * 搜索子部门的Filter
         */
        subOuFilter: string;
        /**
         * 搜索子用户的Filter
         */
        subUserFilter: string;
        /**
         * 具体某个部门或用户信息的filter
         */
        baseFilter: string;
        /**
         * 用户状态对应的域key字段
         */
        statusKeys: Array<string>;
        /**
         * 用户身份证号对应的域key字段
         */
        idcardNumberKeys: Array<string>;
    }

    /**
     * 第三方认证配置
    */
    type ncTThirdPartyAuthConf = {
        /**
         * 唯一标识第三方认证系统
         */
        thirdPartyId: string;

        /**
         * 第三方认证系统名称
         */
        thirdPartyName: string;

        /**
         * 开启状态
         */
        enabled: boolean;

        /**
         * 配置信息
         */
        config: string;
    }

    /********************************** 函数声明*****************************/

    /**
     * 获取登录验证码配置信息
     */
    type GetVcodeConfig = Core.APIs.ThriftAPI<
        void,
        ncTVcodeConfig
    >

    /**
     * 设置登录验证码配置
     */
    type SetVcodeConfig = Core.APIs.ThriftAPI<
        [{ 'ncTVcodeConfig': ncTVcodeConfig }],
        void
    >

    /**
     * 生成验证码
     */
    type CreateVcodeInfo = Core.APIs.ThriftAPI<
        [string],
        ncTVcodeCreateInfo
    >

    /**
     * 获取日志导出加密开关状态
     */
    type GetExportWithPassWordStatus = Core.APIs.ThriftAPI<
        void,
        boolean
    >

    /**
     * 添加导出历史日志文件任务
     */
    type ExportHistoryLog = Core.APIs.ThriftAPI<
        [string, number, string],
        string
    >

    /**
     * 获取日志文件信息
     */
    type GetCompressFileInfo = Core.APIs.ThriftAPI<
        [string],
        string
    >

    type GetGenCompressFileStatus = Core.APIs.ThriftAPI<
        [string],
        boolean
    >

    /**
     * 登录认证
     */
    type Login = Core.APIs.ThriftAPI<
        [string, string, number, { 'ncTUserLoginOption': ncTUserLoginOption }], // ncTUserLoginOption
        string
    >

    /**
     * 验证控制台密码
     */
    type CheckConsoleUserPassword = Core.APIs.ThriftAPI<
        [string, string, number, { 'ncTUserLoginOption': ncTUserLoginOption }], // ncTUserLoginOption
        string
    >

    /**
     * 编辑内置管理员账号
     */
    type EditAdminAccount = Core.APIs.ThriftAPI<
        [string, string],
        void
    >

    /**
     * 设置管理员邮箱
     */
    type SetAdminMailList = Core.APIs.ThriftAPI<
        [string, Array<string>],
        void
    >

    /**
     * 获取组织列表
     */
    type GetDepartmentUser = Core.APIs.ThriftAPI<
        [string, number, number],
        ReadonlyArray<any>
    >

    /**
     * 获取部门用户数量
     */
    type GetDepartmentOfUsersCount = Core.APIs.ThriftAPI<
        [string],
        number
    >

    /**
     * 获取所有用户
     */
    type GetALlUser = Core.APIs.ThriftAPI<
        [number, number],
        ReadonlyArray<any>
    >

    /**
     * 获取子部门列表
     */
    type GetSubDepartments = Core.APIs.ThriftAPI<
        [string],
        ReadonlyArray<ncTDepartmentInfo>
    >

    /**
     * 设置短信配置
     */
    type SMSSetConfig = Core.APIs.ThriftAPI<
        [string],
        void
    >

    /**
     * 短信配置测试
     */
    type SMSTest = Core.APIs.ThriftAPI<
        [string],
        void
    >

    /**
     * 获取短信配置
     */
    type SMSGetConfig = Core.APIs.ThriftAPI<
        void,
        string
    >

    /**
     * 获取月度活跃报表
     */
    type GetActiveReportMonth = Core.APIs.ThriftAPI<
        /**
         * 形如：2018-03
         */
        [string],
        ActiveReportInfo
    >
    /**
     * 获取年度活跃报表
     */
    type GetActiveReportYear = Core.APIs.ThriftAPI<
        /**
         * 形如：2018
         */
        [string],
        ActiveReportInfo
    >

    /**
     * 导出月度活跃报表
     */
    type ExportActiveReportMonth = Core.APIs.ThriftAPI<
        [string, string],
        string
    >

    /**
     * 导出年度活跃报表
     */
    type ExportActiveReportYear = Core.APIs.ThriftAPI<
        [string, string],
        string
    >

    /**
     * 获取最早统计时间
     */
    type OpermGetEarliestTime = Core.APIs.ThriftAPI<
        void,
        string
    >

    /**
     * 获取生成活跃报表状态
     */
    type GetGenActiveReportStatus = Core.APIs.ThriftAPI<
        [string],
        boolean
    >

    /**
     * 获取运维助手开关状态
     */
    type GetActiveReportNotifyStatus = Core.APIs.ThriftAPI<
        void,
        boolean
    >
    /**
     * 设置运维助手开关状态
     */
    type SetActiveReportNotifyStatus = Core.APIs.ThriftAPI<
        [boolean],
        void
    >

    /**
     * 获取部门管理员
     */
    type UsrmGetDepartResponsiblePerson = Core.APIs.ThriftAPI<
        [string],
        ReadonlyArray<ncTUsrmGetUserInfo>
    >

    /**
     * 获取个人文档状态
     */
    type UsrmGetUserDocStatus = Core.APIs.ThriftAPI<
        void,
        boolean
    >

    /**
     * 获取所有域
     */
    type UsrmGetAllDomains = Core.APIs.ThriftAPI<
        void,
        ReadonlyArray<ncTUsrmDomainInfo>
    >

    /**
     * 展开域控节点
     */
    type UsrmExpandDomainNode = Core.APIs.ThriftAPI<
        [ncTUsrmDomainInfo, string],
        {
            ous: ReadonlyArray<ncTUsrmDomainOU>,
            users: ReadonlyArray<ncTUsrmDomainUser>
        }
    >

    /**
     * 搜索域用户或部门
     */
    type UsrmSearchDomainInfoByName = Core.APIs.ThriftAPI<
        [number, string, number, number],
        {
            ous: ReadonlyArray<ncTUsrmDomainOU>,
            users: ReadonlyArray<ncTUsrmDomainUser>,
        }
    >

    /**
     * 获取第三方根组织节点
     */
    type UsrmGetThirdPartyRootNode = Core.APIs.ThriftAPI<
        [string],
        ReadonlyArray<ncTThirdPartyRootNodeInfo>
    >

    /**
     * 展开第三方节点
     */
    type UsrmExpandThirdPartyNode = Core.APIs.ThriftAPI<
        void,
        ncTUsrmThirdPartyNode
    >

    /**
     * 导入第三方组织结构和用户
     */
    type UsrmImportThirdPartyOUs = Core.APIs.ThriftAPI<
        [ReadonlyArray<ncTUsrmThirdPartyOU>, ReadonlyArray<ncTUsrmThirdPartyUser>, ncTUsrmImportOption, string],
        void
    >

    /**
     * 清除导入进度
     */
    type UsrmClearImportProgress = Core.APIs.ThriftAPI<
        void,
        void
    >

    /**
     * 获取导入进度
     */
    type UsrmGetImportProgress = Core.APIs.ThriftAPI<
        void,
        ncTUsrmImportResult
    >

    /**
     * 导入用户
     */
    type UsrmImportDomainUsers = Core.APIs.ThriftAPI<
        [ncTUsrmImportContent, ncTUsrmImportOption, string],
        void
    >

    /**
     * 导入用户
     */
    type UsrmImportDomainOUs = Core.APIs.ThriftAPI<
        [ncTUsrmImportContent, ncTUsrmImportOption, string],
        void
    >

    /**
     * 获取第三方应用配置
     */
    type GetThirdPartyAppConfig = Core.APIs.ThriftAPI<
        ncTPluginType,
        ReadonlyArray<ncTThirdPartyConfig>
    >

    /**
     * 新增第三方应用配置
     */
    type AddThirdPartyAppConfig = Core.APIs.ThriftAPI<
        ncTThirdPartyConfig,
        number
    >

    /**
     * 设置第三方应用配置
     */
    type SetThirdPartyAppConfig = Core.APIs.ThriftAPI<
        ncTThirdPartyConfig,
        void
    >

    /**
     * 删除第三方应用配置
     */
    type DeleteThirdPartyAppConfig = Core.APIs.ThriftAPI<
        number,
        void
    >

    /*   
    * 删除角色
    */
    type DeleteUserRolem = Core.APIs.ThriftAPI<
        [string, string],
        void
    >

    /*
     * 添加角色
     */
    type AddUserRolem = Core.APIs.ThriftAPI<
        [{ 'ncTRoleInfo': ncTRoleInfo }],
        string
    >

    /**
     * 获取角色
     */
    type GetUserRolem = Core.APIs.ThriftAPI<
        string,
        ReadonlyArray<ncTRoleInfo>
    >

    /**
     * 编辑角色
     */
    type EditUserRolem = Core.APIs.ThriftAPI<
        [string, { 'ncTRoleInfo': ncTRoleInfo }],
        void
    >

    /*
     * 设置成员包含添加和编辑成员
     */
    type SetUserRolemMember = Core.APIs.ThriftAPI<
        [string, string, { 'ncTRoleMemberInfo': ncTRoleMemberInfo }],
        void
    >

    /**
     * 获取成员列表
     */
    type GetUserRolemMember = Core.APIs.ThriftAPI<
        [string, string],
        ReadonlyArray<ncTRoleMemberInfo>
    >

    /**
     * 在角色成员列表中根据用户名搜索用户
     */
    type SearchUserRolemMember = Core.APIs.ThriftAPI<
        [string, string, string],
        ReadonlyArray<ncTRoleMemberInfo>
    >

    /**
     * 删除成员
     */
    type DeleteUserRolemMember = Core.APIs.ThriftAPI<
        [string, string, string],
        void
    >

    /**
     * 获取用户角色信息
     */
    type GetUserRole = Core.APIs.ThriftAPI<
        string,
        ReadonlyArray<ncTRoleInfo>
    >

    /**
     * 在所选角色中根据成员id获取详细信息
     */
    type GetRoleMemberDetail = Core.APIs.ThriftAPI<
        [string, string, string],
        ncTRoleMemberInfo
    >

    /**
     * 根据用户角色获取用户所能看到的根组织
     */
    type GetRoleSupervisoryRootOrg = Core.APIs.ThriftAPI<
        [string, string],
        ReadonlyArray<ncTRootOrgInfo>
    >

    /**
     * 在用户角色所管理的部门中搜索用户
     */
    type SearchRoleSupervisoryUsers = Core.APIs.ThriftAPI<
        [string, string, string],
        ReadonlyArray<ncTSearchUserInfo>
    >

    /**
     * 检查先添加的成员是否已经有该角色
     */
    type CheckMemberExist = Core.APIs.ThriftAPI<
        [string, string],
        void
    >

    type GetCustomConfigOfString = Core.APIs.ThriftAPI<
        [string],
        object
    >

    /**
     * 设置双因子认证
     */
    type SetCustomConfigOfString = Core.APIs.ThriftAPI<
        [string, object],
        void
    >

    /**
     * 获取第三方预览工具配置信息
     */
    type GetThirdPartyToolConfig = Core.APIs.ThriftAPI<
        [string],
        ncTThirdPartyToolConfig
    >

    /**
     * 设置第三方预览工具配置信息
     */
    type SetThirdPartyToolConfig = Core.APIs.ThriftAPI<
        [ncTThirdPartyToolConfig],
        void
    >

    /**
     * 测试第三方预览工具配置信息
     */
    type TestThirdPartyToolConfig = Core.APIs.ThriftAPI<
        [string],
        boolean
    >

    /**
     * 获取备用域信息
     */
    type UsrmGetFailoverDomains = Core.APIs.ThriftAPI<
        [string],
        ReadonlyArray<ncTUsrmFailoverDomainInfo>
    >

    /**
     * 检查备用域是否可用（不保存到数据库）
     */
    type UsrmCheckFailoverDomainAvailable = Core.APIs.ThriftAPI<
        [ReadonlyArray<ncTUsrmFailoverDomainInfo>],
        void
    >

    /**
     * 编辑备用域（使用参数覆盖的方式，包括增、删、改；parentDomainId为首选域的id
     */
    type UsrmEditFailoverDomains = Core.APIs.ThriftAPI<
        [ReadonlyArray<ncTUsrmFailoverDomainInfo>, number],
        void
    >

    /**
     * 导出excel组织信息列表
     */
    type UsrmExportBatchUsers = Core.APIs.ThriftAPI<
        Array<string>,
        string
    >

    /**
     * 导出组织信息的excel表是否可以下载 
     */
    type UsrmGetExportBatchUsersTaskStatus = Core.APIs.ThriftAPI<
        [string],
        void
    >

    /**
     * 下载导出组织信息的excel表 
     */
    type UsrmDownloadBatchUsers = Core.APIs.ThriftAPI<
        [string],
        string
    >

    /**
     * 下载导出组织信息的excel表 
     */
    type UsrmGetErrorInfos = Core.APIs.ThriftAPI<
        [number, number],
        ReadonlyArray<ncTImportFailInfo>
    >

    /**
     * 下载导入失败的记录 
     */
    type UsrmDownloadImportFailedUsers = Core.APIs.ThriftAPI<
        void,
        string
    >

    /**
    * 获取组织信息导入进度 
    */
    type UsrmGetProgress = Core.APIs.ThriftAPI<
        void,
        ncTUsrmImportResult
    >

    /*
     * 获取SMTP 配置信息
     */
    type GetSMTPConfig = Core.APIs.ThriftAPI<
        void,
        ncTSmtpSrvConf
    >

    /**
     * 测试SMTP服务器
     */
    type TestSMTPServer = Core.APIs.ThriftAPI<
        [ncTSmtpSrvConf],
        void
    >

    /**
     * 获取三权分立状态
     */
    type GetTriSystemStatus = Core.APIs.ThriftAPI<
        void,
        boolean
    >
    /*
     * 设置SMTP服务器
     */
    type SetSMTPConfig = Core.APIs.ThriftAPI<
        [ncTSmtpSrvConf],
        void
    >

    /**
     * 获取密码管控信息
     */
    type GetPwdControl = Core.APIs.ThriftAPI<
        [string],
        ncTUsrmGetPwdControl
    >

    /**
     * 获取密码管控配置
     */
    type GetPwdConfig = Core.APIs.ThriftAPI<
        void,
        ncTUsrmGetPwdConfig
    >

    /**
     * 设置密码管控配置
     */
    type SetPwdControl = Core.APIs.ThriftAPI<
        [string, ncTUsrmSetPwdControlConfig],
        void
    >
    /**
     * 新建用户 
     */
    type AddUser = Core.APIs.ThriftAPI<
        [{ 'ncTUsrmAddUserInfo': ncTUsrmAddUserInfo }, string],
        string
    >

    /**
     * 编辑用户 
     */
    type EditUser = Core.APIs.ThriftAPI<
        [{ 'ncTEditUserParam': ncTUsrmUserInfo }, string],
        string
    >

    /**
     * 获取当前密级
     */
    type GetSysCsfLevel = Core.APIs.ThriftAPI<
        void,
        number
    >

    /**
     * 获取匿名共享的状态
     */
    type GetSystemLinkShareStatus = Core.APIs.ThriftAPI<
        void,
        boolean
    >

    /**
     * 分页获取匿名共享的策略信息
     */
    type GetLinkShareInfoByPage = Core.APIs.ThriftAPI<
        [number, number],
        ReadonlyArray<ShareInfos>
    >

    /**
     * 获取匿名共享的策略信息总数
     */
    type GetLinkShareInfoCnt = Core.APIs.ThriftAPI<
        void,
        number
    >

    /**
     * 搜索匿名共享的策略信息
     */
    type SearchLinkShareInfo = Core.APIs.ThriftAPI<
        [number, number, string],
        ReadonlyArray<ShareInfos>
    >
    /**
     * 设置匿名共享限制开启关闭状态
     */
    type SetSystemLinkShareStatus = Core.APIs.ThriftAPI<
        [boolean],
        void
    >

    /**
     * 添加匿名共享的策略信息
     */
    type AddLinkShareInfo = Core.APIs.ThriftAPI<
        [ShareInfos],
        void
    >

    /**
     * 删除匿名共享的策略信息
     */
    type DeleteLinkShareInfo = Core.APIs.ThriftAPI<
        [string],
        void
    >

    /**
     * 编辑实名共享的策略信息
     */
    type EditPermShareInfo = Core.APIs.ThriftAPI<
        [{ 'ncTPermShareInfo': ncTPermShareInfo }],
        void
    >

    /**
     * 添加实名共享的策略信息
     */
    type AddPermShareInfo = Core.APIs.ThriftAPI<
        [{ 'ncTPermShareInfo': ncTPermShareInfo }],
        string
    >

    /**
     * 获取实名用户共享限制开启禁用状态
     */
    type GetSystemPermShareStatus = Core.APIs.ThriftAPI<
        void,
        boolean
    >

    /**
     * 分页获取实名共享的策略信息
     */
    type GetPermShareInfoByPage = Core.APIs.ThriftAPI<
        [number, number],
        ReadonlyArray<ncTPermShareInfo>
    >

    /**
     * 获取实名共享的策略信息的总数
     */
    type GetPermShareInfoCnt = Core.APIs.ThriftAPI<
        void,
        number
    >

    /**
     * 搜索实名共享的策略信息
     */
    type SearchPermShareInfo = Core.APIs.ThriftAPI<
        [number, number, string],
        ReadonlyArray<ncTPermShareInfo>
    >

    /**
     * 设置所有用户对其直属部门/其直属组织的共享状态
     */
    type SetPermShareInfoStatus = Core.APIs.ThriftAPI<
        [string, boolean],
        void
    >

    /**
     * 设置实名用户共享限制开启禁用状态
     */
    type SetSystemPermShareStatus = Core.APIs.ThriftAPI<
        [boolean],
        void
    >

    /**
     * 删除实名共享的策略信息
     */
    type DeletePermShareInfo = Core.APIs.ThriftAPI<
        [string],
        void
    >

    /**
     * 设置开启/关闭实名或者匿名共享个人文档库状态
     */
    type SetShareDocStatus = Core.APIs.ThriftAPI<
        [number, number, boolean],
        void
    >

    /**
     * 获取实名或者匿名共享个人文档库状态
     */
    type GetShareDocStatus = Core.APIs.ThriftAPI<
        [number, number],
        void
    >

    /**
     * 批量根据部门ID(组织ID)获取部门（组织）父路经
     */
    type GetDepParentPathById = Core.APIs.ThriftAPI<
        [ReadonlyArray<string>],
        ReadonlyArray<ncTUsrmDepartmentInfo>
    >

    /**
     * 移动部门
     */
    type MoveDepartment = Core.APIs.ThriftAPI<
        [string, string],
        void
    >

    /**
     * 编辑部门的存储位置
     */
    type EditDepartOSS = Core.APIs.ThriftAPI<
        [string, string],
        void
    >

    /**
     * 添加用户至部门
     */
    type AddUsersToDep = Core.APIs.ThriftAPI<
        [ReadonlyArray<string>, string],
        void
    >

    /**
     * 获取当前实时在线用户数
     */
    type OpermGetCurrentOnlineUser = Core.APIs.ThriftAPI<
        void,
        ReadonlyArray<OnlineDataType>
    >

    /**
     * 获取当日最高上线用户数
     */
    type OpermGetMaxOnlineUserDay = Core.APIs.ThriftAPI<
        [string],
        ReadonlyArray<OnlineDataType>
    >

    /**
         * 设置文档标签策略
         */
    type SetCustomConfigOfInt64 = Core.APIs.ThriftAPI<
        [string, number],
        void
    >

    /**
     * 获取文档标签策略
     */
    type GetCustomConfigOfInt64 = Core.APIs.ThriftAPI<
        [string],
        number
    >

    /**
     * 新建部门
     */
    type Usrm_AddDepartment = Core.APIs.ThriftAPI<
        [ncTAddDepartParam],
        string
    >

    /**
     * 编辑部门
     */
    type Usrm_EditDepartment = Core.APIs.ThriftAPI<
        [ncTEditDepartParam],
        void
    >

    /* 
    *新建组织 
    */
    type UsrmCreateOrganization = Core.APIs.ThriftAPI<
        [ncTAddOrgParam],
        string
    >

    /**
     * 编辑组织
     */
    type Usrm_EditOrganization = Core.APIs.ThriftAPI<
        [ncTEditOrgParam],
        void
    >

    /**
     * 获取是否开启匿名共享审核机制
     */
    type GetCustomConfigOfBool = Core.APIs.ThriftAPI<
        [string],
        boolean
    >

    /**
     * 启用或禁用匿名共享审核机制
     */
    type SetCustomConfigOfBool = Core.APIs.ThriftAPI<
        [string, boolean],
        void
    >

    /**
     * 获取第三方标密系统配置
     */
    type GetThirdCSFSysConfig = Core.APIs.ThriftAPI<
        void,
        ncTThirdCSFSysConfig
    >

    /**
     * 第三方标密系统配置
     */
    type ncTThirdCSFSysConfig = {
        /**
         * 是否使用第三方标密系统
         */
        isEnable: boolean;

        /**
         * 第三方标密系统id
         */
        id: string;

        /**
         * 仅上传已标密文件（启用文件上传定密勾选项参数）
         */
        only_upload_classified: boolean;

        /**
         * 仅共享已标密文件
         */
        only_share_classified: boolean;

        /**
         * 上传自动识别密级（启用自动识别文件密级勾选项参数）
         */
        auto_match_doc_classfication: boolean;
    }

    /**
     * 设置第三方标密系统配置
     */
    type SetThirdCSFSysConfig = Core.APIs.ThriftAPI<
        [ncTThirdCSFSysConfig],
        void
    >

    /**
     * 设置域同步状态,-1:域同步关闭,0：域正向同步开启，1：域反向同步开启
     */
    type SetDomainSyncStatus = Core.APIs.ThriftAPI<
        [number, number],
        void
    >

    /**
     * 第三方同步(如果为域同步，则appId为域id; autoSync: True-定期同步， False-单次同步)
     */
    type StartSync = Core.APIs.ThriftAPI<
        [string, number],
        void
    >

    /**
     * 获取域配置信息
     */
    type GetDomainConfig = Core.APIs.ThriftAPI<
        [string],
        void
    >

    /**
     * 开启或者关闭域控
     */
    type SetDomainStatus = Core.APIs.ThriftAPI<
        [number, number],
        void
    >

    /**
     * 增加域控
     */
    type AddDomain = Core.APIs.ThriftAPI<
        [ncTUsrmDomainInfo],
        void
    >

    /**
     * 
     */
    type EditDomain = Core.APIs.ThriftAPI<
        [ncTUsrmDomainInfo],
        void
    >

    /**
     * 获取第三方认证管理信息
     */
    type GetThirdPartyAuth = Core.APIs.ThriftAPI<
        void,
        ncTThirdPartyAuthConf
    >

    /**
     * 获取冻结状态
     */
    type GetFreezeStatus = Core.APIs.ThriftAPI<
        void,
        boolean
    >

    /**
     * 在部门中根据key搜索用户，并返回搜索的用户总数
     */
    type CountSearchDepartmentOfUsers = APIs.ThriftAPI<
        [string, string],
        number
    >

    /**
     * 删除域
     */
    type DeleteDomain = Core.APIs.ThriftAPI<
        [number],
        void
    >

    /**
     * 根据id获取域信息
     */
    type GetDomainById = Core.APIs.ThriftAPI<
        [number],
        ReadonlyArray<any>
    >

    /**
     * 设置域配置信息
     */
    type SetDomainConfig = Core.APIs.ThriftAPI<
        [number, ncTUsrmDomainConfig],
        void
    >

    /**
     * 设置域关键字配置信息
     */
    type SetDomainKeyConfig = Core.APIs.ThriftAPI<
        [number, ncTUsrmDomainKeyConfig],
        void
    >

    /**
     * 获取域关键字配置信息
     */
    type GetDomainKeyConfig = Core.APIs.ThriftAPI<
        [number],
        ncTUsrmDomainKeyConfig
    >

    /*
     * 在部门中根据key搜索用户，并返回分页数据
     */
    type SearchDepartmentOfUsers = Core.APIs.ThriftAPI<
        [string, string, number, number],
        ReadonlyArray<ncTUsrmGetUserInfo>
    >

    /**
     * 分页获取用户信息
     */
    type GetDepartmentOfUsers = Core.APIs.ThriftAPI<
        [string, number, number],
        ReadonlyArray<ncTUsrmGetUserInfo>
    >

    /**
     * 获取所有用户下的用户数
     */
    type GetAllUserCount = Core.APIs.ThriftAPI<
        void,
        number
    >

    /**
     * 分页获取所有用户下的用户
     */
    type GetAllUsers = Core.APIs.ThriftAPI<
        [number, number],
        ReadonlyArray<ncTUsrmGetUserInfo>
    >

    /**
     * 对同级部门进行排序
     */
    type SortDepartment = Core.APIs.ThriftAPI<
        [string, string, string],
        void
    >

    /**
     * 获取密级枚举
     */
    type GetCSFLevels = Core.APIs.ThriftAPI<
        void,
        any
    >

    /**
     * 检查用户是否属于某个部门及其子部门
     */
    type CheckUserInDepart = Core.APIs.ThriftAPI<
        [string, string],
        boolean
    >

    /**
     * 设置用户权重
     */
    type EditUserPriority = Core.APIs.ThriftAPI<
        [string, number],
        void
    >

    /**
     * 获取月份间每天的最大在线数
     */
    type GetMaxOnlineUserMonth = Core.APIs.ThriftAPI<
        [string, string],
        ReadonlyArray<OnlineDataType>
    >

    /**
     * 获取清除缓存的时间间隔
     */
    type GetClearCacheInterval = Core.APIs.ThriftAPI<
        void,
        number
    >

    /**
     * 设置清除缓存的时间间隔
     */
    type SetClearCacheInterval = Core.APIs.ThriftAPI<
        number,
        void
    >

    /**
     * 获取清除缓存的空间限额
     */
    type GetClearCacheQuota = Core.APIs.ThriftAPI<
        void,
        number
    >

    /**
     * 设置清除缓存的空间限额
     */
    type SetClearCacheQuota = Core.APIs.ThriftAPI<
        number,
        void
    >

    /**
     * 获取客户端是否强制清除缓存状态
     */
    type GetForceClearCacheStatus = Core.APIs.ThriftAPI<
        void,
        boolean
    >

    /**
     * 设置客户端是否强制清除缓存
     */
    type SetForceClearCacheStatus = Core.APIs.ThriftAPI<
        boolean,
        void
    >

    /**
     * 设置客户端是否隐藏缓存设置的状态
     */
    type GetHideClientCacheSettingStatus = Core.APIs.ThriftAPI<
        void,
        boolean
    >

    /**
     * 获取客户端是否隐藏缓存设置的状态
     */
    type SetHideClientCacheSettingStatus = Core.APIs.ThriftAPI<
        boolean,
        void
    >
}