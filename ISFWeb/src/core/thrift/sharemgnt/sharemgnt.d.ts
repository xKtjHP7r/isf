declare namespace Core {
    namespace ShareMgnt {

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
            responsiblePerson: ncTUsrmGetUserInfo;

            /**
             * 对象存储信息
             */
            ossInfo: ncTUsrmOSSInfo;

            /**
             * 子部门id        
             */
            subDepartIds: Array<string>;
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
            directDeptInfo: ncTUsrmDirectDeptInfo;
        }

        /**
         *  对象存储信息
         */
        type ncTUsrmOSSInfo = {
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
            displayName: string;
            /**
             *  邮箱
             */
            email: string;
            /**
            * 用户类型
            * enum ncTUsrmUserType {
            *   NCT_USER_TYPE_LOCAL = 1,        // 本地用户
            *   NCT_USER_TYPE_DOMAIN = 2,       // 域用户
            *   NCT_USER_TYPE_THIRD = 3,        // 第三方验证用户
            *   }
             */
            userType: number;

            /**
             *  所属部门id，-1时表示为未分配用户          
             */
            departmentIds: Array<string>;

            /**
             *  所属部门名称
             */
            departmentNames: Array<string>;
            /**
             * // 用户状态
             * enum ncTUsrmUserStatus {
             *      NCT_STATUS_ENABLE = 0,          // 启用
             *      NCT_STATUS_DISABLE = 1,         // 禁用
             *      NCT_STATUS_DELETE = 2,          // 用户被第三方系统删除
             * }
             */
            status: number;

            /**
             *  排序优先级 
             */
            priority: number;

            /**
             *  用户密级
             */
            csfLevel: number;

            /**
             *  密码管控
             */
            pwdControl: boolean;

            /**
             *  归属站点信息   
             */
            ossInfo: ncTUsrmOSSInfo;

            /**
             *  用户创建时间
             */
            createTime: number;

            /**
             *  用户冻结状态，true:冻结 false:未冻结                    
             */
            freezeStatus: boolean;
        }

        /**
         * 限速配置管理信息
         */
        type ncTLimitRateInfo = {
            /**
             * 限速配置唯一标识
             */
            id: string;

            /**
             * 最大上传速度
             */
            uploadRate: number;

            /**
             * 最大下载速度
             */
            downloadRate: number;

            /**
             * 用户列表
             */
            userInfos: ReadonlyArray<ncTLimitRateObject>;

            /**
             * 部门列表
             */
            depInfos: ReadonlyArray<ncTLimitRateObject>;

            /**
             * 限速配置类型
             */
            limitType: number;
        }

        /**
         *  限速配置管理对象
         */

        type ncTLimitRateObject = {
            /**
             * 限速对象id
             */
            objectId: string;

            /**
             * 限速对象名称
             */
            objectName: string;
        }

        /**
         * 导入域用户
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
            users: ReadonlyArray<ncTUsrmDomainUser>;

            /**
             * 勾选的域组织
             */
            ous: ReadonlyArray<ncTUsrmDomainOU>;
        }

        /**
         * 域控信息
         */
        type ncTUsrmDomainInfo = {
            /**
             * 域id
             */
            id: number;

            /**
             * 1：主域，2：子域，3：信任域
             */
            type: ncTUsrmDomainType;

            /**
             * 域名
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
             * 状态,true-启用，false-禁用
             */
            status: boolean;

            /**
             * 同步状态, -1 关闭, 0 正向同步, 1 反向同步
             */
            syncStatus: number;

            /**
             * 域配置信息
             */
            config: ncTUsrmDomainConfig;

            /**
             * useSSL
             */
            useSSL?: boolean;
        }

        /*
        *域控类型
       */
        enum ncTUsrmDomainType {
            /**
             * 主域
             */
            NCT_DOMAIN_TYPE_PRIMARY = 1,

            /**
             * 子域
             */
            NCT_DOMAIN_TYPE_SUB = 2,

            /**
             * 信任域
             */
            NCT_DOMAIN_TYPE_TRUST = 3,
        }

        /*
        *域配置信息
       */
        type ncTUsrmDomainConfig = {
            /*
            *要导入到的目的组织或部门id
           */
            destDepartId: string;

            /**
             * 设置域配置信息的时候，设为None
             */
            desetDepartName: string;

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
        }

        /**
         * 同步设置
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
         * 导入域用户信息
         */
        type ncTUsrmDomainUser = {
            /**
             * 登录名
             */
            loginName: string;

            /**
             * 显示名
             */
            displayName: string;

            /**
             * 邮箱
             */
            email: string;

            /**
             * 所属部门的路径，格式：OU=研发部, DC=test2,
             */
            ouPath: string;

            /**
             * 对象的GUID，具有唯一性
             */
            objectGUID: string;

            /**
             * dnPath
             */
            dnPath: string;

            /**
             * 身份证号
             */
            idcardNumber: string;
        }

        /**
         * 导入域组织信息
         */
        type ncTUsrmDomainOU = {
            /**
             * 组织名称
             */
            name: string;

            /**
             * 组织负责人
             */
            rulerName: string;

            /**
             * 路径，格式：OU=研发部, DC=test2,DC=develop,DC=cn
             */
            pathName: string;

            /**
             * 父部门的路径，格式：OU=研发部, DC=test2,DC=develop,DC=cn
             */
            parentOUPath: string;

            /**
             * 对象的GUID，具有唯一性
             */
            objectGUID: string;

            /**
             * 是否导入组织下的所有子组织及用户，True--导入
             */
            importAll: boolean;
        }
    }
}