import { getErrorMessage, getErrorTemplate } from '@/core/errcodeconsole/errcode';
import __ from './locale';

export enum DocLibType {
    /**
     * 全部文档库
     */
    All = 'all',

    /**
     * 个人文档库
     */
    UserDocLib = 'user_doc_lib',

    /**
     * 部门文档库
     */
    DepDocLib = 'department_doc_lib',

    /**
     * 自定义文档库
     */
    CustomDocLib = 'custom_doc_lib',

    /**
     * 知识库
     */
    KnowledgeDocLib = 'knowledge_doc_lib',
    /**
     * 知识仓库
     */
    KnowledgeRepo = 'knowledge_repo',
}

/**
 * 文档库
 */
export declare namespace Doclibs {
    /**
     * 文档库信息
     */
    interface DocInfo {
        /**
         * 文档库id
         */
        id?: string;

        /**
         * 文档库名称
         */
        name: string;

        /**
         * 配额空间
         */
        quota: {
            /**
             * 分配空间
             */
            allocated: number;

            /**
             * 已使用空间
             */
            used: number;
        };

        /**
         * 存储位置
         */
        storage?: StorageInfo;

        /**
         * 存储位置（个人文档库）
         */
        siteInfo?: StorageInfo;

        /**
         * 库所有者
         */
        owned_by?: ReadonlyArray<UserInfo>;

        /**
         * 库创建者
         */
        created_by?: UserInfo;

        /**
         * 创建时间
         */
        created_at?: string;

        /**
         * 库类型
         */
        type?: string;

        /**
         * 管理部门名称
         */
        department?: {
            /**
             * 部门id
             */
            id: string;

            /**
             * 部门名称
             */
            name: string;
        };

        /**
        * 库分类
        */
        subtype?: {
            /**
            * 库分类id
            */
            id?: string;

            /**
            * 库分类名称
            */
            name?: string;
        };
    }

    /**
     * 存储位置信息
     */
    interface StorageInfo {
        /**
         * 存储位置id
         */
        id: string;

        /**
         * 存储位置名称
         */
        name: string;

        /**
         * 存储位置显示名称
         */
        displayName?: string;
    }

    /**
     * 用户信息
     */
    interface UserInfo {
        /**
         * 用户id
         */
        id: string;

        /**
         * 用户显示名
         */
        name: string;

        /**
         * 类型
         */
        type: string;
    }

    /**
     * 选择部门或者自定义文档库信息
     */
    interface DocLibInfo {
        /**
         * 文档库id
         */
        id: string;

        /**
         * 文档库名称
         */
        name: string;

        /**
         * 文档库类型
         */
        type?: string;

        /**
         * 部门文档库对应部门id，如果是自定义文档库，则为''
         */
        depId?: string;
    }

    /**
     * 错误信息
     */
    interface ErrorInfo {
        /**
         * 错误码
         */
        code?: number;

        /**
         * 错误信息
         */
        message?: string;

        /**
         * 错误详情
         */
        detail?: any;
    }

    interface ValidateInfo {
        /**
         * 校验码
         */
        code: number;

        /**
         * 气泡提示
         */
        tipMsg: { [key: number]: string };
    }
}

/**
 * 校验状态
 */
export const enum ValidateStatus {
    /**
     * 正常
     */
    Normal,

    /**
     * 不允许为空
     */
    Empty,

    /**
     * 库名称不合法
     */
    InvalidDocName,

    /**
     * 库类型名称不合法
     */
    InvalidDocTypeName,

    /**
     * 库类型名称已存在
     */
    ExitDocTypeName,

    /**
     * 库类型不存在
     */
    NotExitDocTypeName,
}

/**
 * 提示语
 */
export const ValidateMessages = {
    [ValidateStatus.Empty]: __('此项不允许为空。'),
    [ValidateStatus.InvalidDocName]: __('库名称不能包含 \\ / : * ? " < > | 特殊字符，长度不能超过128个字符。'),
    [ValidateStatus.InvalidDocTypeName]: __('库分类显示不能包含 \\ / : * ? " < > | 特殊字符，长度不能超过128个字符。'),
    [ValidateStatus.ExitDocTypeName]: __('该分类已被其他类型的显示占用，请重新输入。'),
    [ValidateStatus.NotExitDocTypeName]: __('您选中的库分类已不存在，请重新选择。'),
}

/**
 * 文档库类型
 */
export enum Type {
    /**
     * 空
     */
    None = '',

    /**
     * 自定义文档库
     */
    Custom = 'custom',

    /**
     * 部门文档库
     */
    Department = 'department',

    /**
     * 个人文档库
     */
    User = 'user',

    /**
     * 知识库
     */
    Knowledge = 'knowledge',

    /**
     * 知识仓库
     */
    KnowledgeRepo = 'knowledge_repo',
}

/**
 * 特殊库的名称
 */
export const SpecialLibName = {
    [Type.Custom]: __('所有自定义文档库'),
    [Type.Department]: __('所有部门文档库'),
    [Type.User]: __('所有个人文档库'),
    [Type.Knowledge]: __('所有知识库'),
    [Type.KnowledgeRepo]:__('所有知识仓库'),
}

/**
 * 特殊库
 */
export const specialUserLib = { id: '', name: SpecialLibName[Type.User] };

/**
 * 特殊策略
 */
export enum SpecialPolicy {
    /**
     * 特殊策略所有个人文档库id
     */
    AllUserDocLib = 'user_doc_lib',

    /**
     * 特殊策略所有部门文档库id
     */
    AllDepDocLib = 'department_doc_lib',

    /**
     * 特殊策略所有自定义文档库id
     */
    AllCustomDocLib = 'custom_doc_lib',

    /**
     * 特殊策略所有知识库id
     */
    AllKnowledgeDocLib = 'knowledge_doc_lib',

    /**
     * 特殊策略所有所有知识仓库id
     */
    AllKnowledgeRepo = 'knowledge_repo'
}

/**
 * 特殊策略的名称
 */
export const SpecialPolicyLabels = {
    [SpecialPolicy.AllUserDocLib]: __('所有个人文档库'),
    [SpecialPolicy.AllDepDocLib]: __('所有部门文档库'),
    [SpecialPolicy.AllCustomDocLib]: __('所有自定义文档库'),
    [SpecialPolicy.AllKnowledgeDocLib]: __('所有知识库'),
    [SpecialPolicy.AllKnowledgeRepo]:__('所有知识仓库'),
}

export interface ScopeInfos {
    /**
     * 首页数据
     */
    firstPageData: ReadonlyArray<Doclibs.DocInfo>;

    /**
     * 搜索个人文档库的范围
     */
    scopeParam: any;

    /**
     * 个人文档库总数
     */
    total: number;
}

/**
 * 获取校验提示语（气泡提示：静态检查 | 接口抛错）
 * @param statusCode ValidateStatus | ErrorCode 错误码 | 状态提示
 * @param params 提示语是函数，需要根据参数给出提示
 * @param args 提示语中的参数信息
 * @param validateMsg 提示语对象
 */
export const getValidateInfo = (statusCode: number, exargs: { params?: any; args?: object; validateMsg?: object } = {}): Doclibs.ValidateInfo => {
    if (statusCode === ValidateStatus.Normal) {
        return {
            code: statusCode,
            tipMsg: {},
        }
    } else {
        const { params, args, validateMsg } = exargs

        const match = validateMsg ? validateMsg[statusCode] : ValidateMessages[statusCode]

        let message = ''

        if (!match) {
            message = args ? getErrorTemplate(statusCode, params)(args) : getErrorMessage(statusCode, params)
        } else {
            message = args && typeof match === 'function' ? match(args) : match
        }

        return {
            code: statusCode,
            tipMsg: { [statusCode]: message },
        }
    }
}

/**
 * 存储位置
 */
export declare namespace OSSSite {
    /**
     * 对象存储信息
     */
    interface OSSInfo {
        /**
         * 存储位置id
         */
        ossId: string;

        /**
         * 存储位置名称
         */
        ossName: string;

        /**
         * 对象存储状态
         */
        enabled: boolean;

        /**
         * 存储位置显示名称
         */
        displayName?: string;
    }

    /**
     * 文档库类别管理中的存储位置信息
     */
    interface StorageInfo {
        /**
         * 对象存储id
         */
        id: string;

        /**
         * 对象存储name
         */
        name: string;
    }
}

/**
 * 获取存储位置显示名称
 */
export function getStorageDisplayName(storage: OSSSite.StorageInfo): string {
    const { id, name } = storage

    return !id ? __('未指定（跟随文件上传者的指定存储位置）') : name
}

/**
 * 存储类型
 */
export enum StorageType {
    /**
     * 未指定（跟随文件上传者的指定存储位置）
     */
    Unspecified = 'unspecified',

    /**
     * 指定对象存储id
     */
    Specified = 'specified',

    /**
     * 使用对应用户的存储位置
     */
    SameAsUser = 'same_as_user',
}

/**
 * 访问资源类型
 */
export enum ResourceType {
    /**
     * 所有（个人/部门/自定义/知识库）文档库类型
     */
    Abstract = 'abstract',

    /**
     * 指定文档库
     */
    Specific = 'specific',
}
export interface KcRepoInfoType {
    id: string;
    name: string;
}