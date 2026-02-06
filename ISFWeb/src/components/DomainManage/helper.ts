import __ from './locale'

/**
 * 域类型
 */
export enum DomainType {
    /**
     * 主域
     */
    Primary = 1,

    /**
     * 子域
     */
    Sub = 2,

    /**
     * 信任域
     */
    Trust = 3,
}

/**
 * 操作类型
 */
export enum ActionType {
    /**
     * 无
     */
    None,

    /**
     * 新建
     */
    Add,

    /**
     * 编辑
     */
    Edit,
}

/**
 * 验证参数
 */
export const enum ValidateStatus {
    /**
     * 正常
     */
    Normal,

    /**
     * 此项不允许为空
     */
    Empty,

    /**
     * 域名只能包含 英文、数字 及 -. 字符，每一级不能以“-”字符开头或结尾，每一级长度必需 1~63 个字符，且总长不能超过253个字符。
     */
    InvalidDomainName,

    /**
     * IP不合法
     */
    InvalidDomainIP,

    /**
     * 端口不合法
     */
    InvalidDomainPort,

    /**
     * 域备用域域名相同
     */
    DuplicateWithSpareDomain,

    /**
     * 备用域与主域不在一个域
     */
    DomainsNotInOneDomain,

    /**
     * 当前域与主域相同
     */
    SpareAddressDuplicateWithMainDomain,

    /**
     * 备用域与存在
     */
    SpareDomainExist,

    /**
     * 域名已存在
     */
    DomainNameExist,

    /**
     * 域名与ip不匹配
     */
    AddressNotMatchIP,
}

/**
 * 错误信息 不需函数
 */
export const ValidateMessages = {
    [ValidateStatus.Empty]: __('此项不允许为空。'),
    [ValidateStatus.InvalidDomainName]: __('域名只能包含 英文、数字 及 -. 字符，每一级不能以“-”字符开头或结尾，每一级长度必需 1~63 个字符，且总长不能超过253个字符。'),
    [ValidateStatus.InvalidDomainIP]: __('IP地址输入不合法，请检查您输入的内容是否有误。IPv4地址格式形如 XXX.XXX.XXX.XXX，每段必须是 0~255 之间的整数。 IPv6地址格式形如 XXXX:XXXX:XXXX:XXXX:XXXX:XXXX:XXXX:XXXX，其中每个X都为十六进制数。'),
    [ValidateStatus.InvalidDomainPort]: __('端口号必须是1~65535之间的整数。'),
    [ValidateStatus.DuplicateWithSpareDomain]: __('当前域控地址与备用域地址相同。'),
    [ValidateStatus.DomainsNotInOneDomain]: __('当前域控地址与主域不在同一个域内。'),
    [ValidateStatus.SpareAddressDuplicateWithMainDomain]: __('当前域控地址与主域地址相同。'),
    [ValidateStatus.SpareDomainExist]: __('当前域控地址已存在。'),
    [ValidateStatus.DomainNameExist]: __('当前域名已存在。'),
    [ValidateStatus.AddressNotMatchIP]: __('当前域名与域控地址不匹配。'),
}