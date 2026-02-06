import * as React from 'react'
import { trim, noop } from 'lodash'
import session from '@/util/session'
import { ErrorCode } from '@/core/thrift/sharemgnt/errcode'
import { manageLog, Level, ManagementOps } from '@/core/log'
import { setDomainConfig, setDomainSyncStatus, startSync, usrmGetAllDomains } from '@/core/thrift/sharemgnt/sharemgnt'
import { Message2 } from '@/sweet-ui'
import WebComponent from '../../../webcomponent'
import { ActionType, ValidateStatus } from '../../helper'
import __ from './locale'
import { getLevelConfig } from '@/core/apis/console/usermanagement'

/**
 * 同步方式
 */
export enum SyncMode {
    ALL,
    PART,
    USERS,
}

/**
 * 有效期
 */
export enum ExpireTime {
    /**
     * 永久有效
     */
    Forever = -1,

    /**
     * 一个月
     */
    OneMonth = 30,

    /**
     * 三个月
     */
    ThreeMonths = 90,

    /**
     * 半年
     */
    HalfYear = 180,

    /**
     * 一年
     */
    OneYear = 365,

    /**
     * 两年
     */
    TwoYears = 730,

    /**
     * 三年
     */
    ThreeYears = 1095,

    /**
     * 四年
     */
    FourYears = 1460,
}

/**
 * 同步周期单位
 */
export enum SyncUnit {
    /**
     * 分钟
     */
    Minutes,

    /**
     * 小时
     */
    Hour,

    /**
     * 天
     */
    Day,
}

/**
 * 验证类型
 */
export enum VerifyType {
    /**
     * 同步周期
     */
    SyncInterval,
}

/**
 * 用户有效期
 */
const ExpireTimes = {
    [ExpireTime.OneMonth]: __('一个月'),
    [ExpireTime.ThreeMonths]: __('三个月'),
    [ExpireTime.HalfYear]: __('半年'),
    [ExpireTime.OneYear]: __('一年'),
    [ExpireTime.TwoYears]: __('两年'),
    [ExpireTime.ThreeYears]: __('三年'),
    [ExpireTime.FourYears]: __('四年'),
    [ExpireTime.Forever]: __('永久有效'),
}

interface SyncSetInfoProps {
    /**
     * 域信息
     */
    domainInfo: Core.ShareMgnt.ncTUsrmDomainInfo;

    /**
     * 选中项
     */
    selection: Core.ShareMgnt.ncTUsrmDomainInfo;

    /**
     * 渲染类型
     */
    actionType: ActionType;

    /**
     * 向父组件传递编辑状态
     */
    onRequestEditStatus: (status: boolean) => void;

    /**
     * 域错误时回调
     */
    onRequestDomainInvalid: (id) => void;

    /**
     * 编辑完成回调
     */
    onSetSyncSetInfoSuccess: () => void;
}

interface SyncSetInfoState {
    /**
         * 定期同步设置信息
         */
    syncSettingInfo: {
        periodicSyncStatus: boolean;
        syncObject: ReadonlyArray<any>;
        syncTarget: ReadonlyArray<any>;
        syncInterval: number | string;
        syncIntervalPlaceholder: string;
        syncIntervalUnit: SyncUnit;
        expireTime: ExpireTime;
        userStatus: boolean;
        syncMode: SyncMode;
        csfLevel: number;
        csfOptions: Array<{ value: number, name: string }>;
    };

    /**
     * 定期同步设置保存取消是否显示
     */
    isSyncSettingEditStatus: boolean;

    /**
     * 同步对象窗口显示
     */
    isShowsyncObjectDialog: boolean;

    /**
     * 同步目标窗口显示
     */
    isShowSyncTargetDialog: boolean;

    /**
     * 验证状态
     */
    validateStatus: {
        syncIntervalValidateStatus: ValidateStatus;
    };
}

export default class SyncSetInfoBase extends WebComponent<SyncSetInfoProps, SyncSetInfoState> {
    static defaultProps = {
        actionType: ActionType.Add,
        selection: {},
        editDomain: {},
        onRequestEditStatus: noop,
        onSetSyncSetInfoSuccess: noop,
    }

    state: SyncSetInfoState = {
        /**
         * 定期同步设置信息
         */
        syncSettingInfo: {
            periodicSyncStatus: false,
            syncObject: [],
            syncTarget: [],
            syncInterval: 5,
            syncIntervalPlaceholder: __('请输入1-60的数值'),
            syncIntervalUnit: SyncUnit.Minutes,
            expireTime: ExpireTime.Forever,
            userStatus: true,
            syncMode: SyncMode.ALL,
            csfLevel: null,
            csfOptions: [],
        },

        /**
         * 定期同步设置保存取消是否显示
         */
        isSyncSettingEditStatus: false,

        /**
         * 同步对象窗口显示
         */
        isShowsyncObjectDialog: false,

        /**
         * 同步目标窗口显示
         */
        isShowSyncTargetDialog: false,

        /**
         * 验证状态
         */
        validateStatus: {
            syncIntervalValidateStatus: ValidateStatus.Normal,
        },
    }

    /**
     * 用户 id
     */
    userId = session.get('isf.userid')

    /**
     * 存储原始信息
     */
    originSyncSettingInfo = this.state.syncSettingInfo

    async componentDidMount() {
        const { csf_level_enum } = await getLevelConfig({ fields: 'csf_level_enum' })
        const domains = await usrmGetAllDomains()
        const { name, syncStatus, config: { ouPath, syncMode, destDepartId, desetDepartName, syncInterval, validPeriod, userEnableStatus, csfLevel } } = domains.find((item) => item.id === this.props.domainInfo.id)

        this.setState({
            syncSettingInfo: {
                periodicSyncStatus: syncStatus === 0 ? true : false,
                syncObject: ouPath.length ? ouPath.map((item) => {
                    return { name: item.split(',')[0].split('=')[1], pathName: item }
                }) : [{ name }],
                syncTarget: destDepartId && desetDepartName ? [{ id: destDepartId, name: desetDepartName }] : [],
                syncInterval: syncInterval < 60 ? syncInterval : syncInterval < 1440 ? syncInterval / 60 : syncInterval / 1440,
                syncIntervalPlaceholder: syncInterval < 60 ? __('请输入1-60的数值') : syncInterval < 1440 ? __('请输入1-24的数值') : __('请输入正整数'),
                syncIntervalUnit: syncInterval < 60 ? SyncUnit.Minutes : syncInterval < 1440 ? SyncUnit.Hour : SyncUnit.Day,
                expireTime: validPeriod,
                userStatus: userEnableStatus,
                syncMode: syncMode,
                csfLevel: csfLevel || csf_level_enum?.[0]?.value,
                csfOptions: csf_level_enum,
            },
        }, () => {
            this.originSyncSettingInfo = this.state.syncSettingInfo
        })
    }

    /**
     * 定期同步开关
     */
    protected changeSyncStatus = ({ detail }: { detail: boolean }): void => {
        detail ?
            this.setState({
                syncSettingInfo: {
                    ...this.state.syncSettingInfo,
                    periodicSyncStatus: true,
                },
                validateStatus: {
                    syncIntervalValidateStatus: ValidateStatus.Normal,
                },
                isSyncSettingEditStatus: true,
            }) :
            this.setState({
                syncSettingInfo: {
                    ...this.originSyncSettingInfo,
                    periodicSyncStatus: false,
                },
                validateStatus: {
                    syncIntervalValidateStatus: ValidateStatus.Normal,
                },
                isSyncSettingEditStatus: true,
            })
        this.props.onRequestEditStatus(true)
    }

    /**
     * 改变同步对象
     */
    protected changeSyncObject = (syncObject: any): void => {
        this.setState({
            syncSettingInfo: {
                ...this.state.syncSettingInfo,
                syncObject,
            },
            isSyncSettingEditStatus: true,
        })
        this.props.onRequestEditStatus(true)
    }

    /**
     * 改变同步目标
     */
    protected changeSyncTarget = (syncTarget: any): void => {
        this.setState({
            syncSettingInfo: {
                ...this.state.syncSettingInfo,
                syncTarget,
            },
            isSyncSettingEditStatus: true,
        })
        this.props.onRequestEditStatus(true)
    }

    /**
     * 更新同步对象
     */
    protected updateSyncObject = (syncObject: ReadonlyArray<any>): void => {
        this.setState({
            syncSettingInfo: {
                ...this.state.syncSettingInfo,
                syncObject,
            },
            isSyncSettingEditStatus: true,
            isShowsyncObjectDialog: false,
        })
        this.props.onRequestEditStatus(true)
    }

    /**
     * 更新同步目标
     */
    protected updateSyncTarget = (syncTarget: ReadonlyArray<any>): void => {
        this.setState({
            syncSettingInfo: {
                ...this.state.syncSettingInfo,
                syncTarget,
            },
            isSyncSettingEditStatus: true,
            isShowSyncTargetDialog: false,
        })
        this.props.onRequestEditStatus(true)
    }

    /**
     * 改变同步周期
     */
    protected changeSyncInterval = ({ detail }: { detail: number }) => {
        const { syncIntervalUnit } = this.state.syncSettingInfo;
        this.setState({
            syncSettingInfo: {
                ...this.state.syncSettingInfo,
                syncInterval: detail === 0 ? '' : syncIntervalUnit === SyncUnit.Minutes ? detail > 60 ? 60 : detail : syncIntervalUnit === SyncUnit.Hour ? detail > 24 ? 24 : detail : detail,
            },
            isSyncSettingEditStatus: true,
            validateStatus: {
                ...this.state.validateStatus,
                syncIntervalValidateStatus: ValidateStatus.Normal,
            },
        })
        this.props.onRequestEditStatus(true)
    }

    /**
     * 改变同步周期单位
     */
    protected changeSyncIntervalUnit = ({ detail }: { detail: SyncUnit }) => {
        this.setState({
            syncSettingInfo: {
                ...this.state.syncSettingInfo,
                syncIntervalUnit: detail,
                syncInterval: '',
                syncIntervalPlaceholder: detail === SyncUnit.Minutes ? __('请输入1-60的数值') : detail === SyncUnit.Hour ? __('请输入1-24的数值') : __('请输入正整数'),
            },
            isSyncSettingEditStatus: true,
        })
        this.props.onRequestEditStatus(true)
    }

    /**
    * 密级切换
    */
    protected updateCsfLevel(csfLevel: number): void {
        this.setState({
            syncSettingInfo: {
                ...this.state.syncSettingInfo,
                csfLevel,
            },
            isSyncSettingEditStatus: true,
        })
    }

    /**
     * 改变用户有效期限
     */
    protected changeExpireTime = ({ detail }: { detail: ExpireTime }): void => {
        this.setState({
            syncSettingInfo: {
                ...this.state.syncSettingInfo,
                expireTime: detail,
            },
            isSyncSettingEditStatus: true,
        })
        this.props.onRequestEditStatus(true)
    }

    /**
     * 改变用户默认状态
     */
    protected changeUserStatus = (status: boolean): void => {
        this.setState({
            syncSettingInfo: {
                ...this.state.syncSettingInfo,
                userStatus: status,
            },
            isSyncSettingEditStatus: true,
        })
        this.props.onRequestEditStatus(true)
    }

    /**
     * 改变同步方式
     */
    protected changeSyncMode = (syncMode: SyncMode): void => {
        this.setState({
            syncSettingInfo: {
                ...this.state.syncSettingInfo,
                syncMode,
            },
            isSyncSettingEditStatus: true,
        })
        this.props.onRequestEditStatus(true)
    }

    /**
     * 转换同步周期
     */
    private getSyncTime = (): number => {
        const { syncSettingInfo: { syncInterval, syncIntervalUnit } } = this.state;

        switch (syncIntervalUnit) {
            case SyncUnit.Minutes:
                return Number(syncInterval)

            case SyncUnit.Hour:
                return Number(syncInterval) * 60

            case SyncUnit.Day:
                return Number(syncInterval) * 1440
        }
    }

    /**
     * 校验定期同步设置
     */
    protected verifySyncSettingInfo = (verifyType?: VerifyType): boolean => {
        const { syncSettingInfo: { syncInterval } } = this.state,
            syncIntrevalResult = trim(syncInterval) !== '';

        switch (verifyType) {
            case VerifyType.SyncInterval:
                if (syncIntrevalResult) {
                    return true
                } else {
                    this.setState({
                        validateStatus: {
                            ...this.state.validateStatus,
                            syncIntervalValidateStatus: ValidateStatus.Empty,
                        },
                    })
                    return false
                }

            default:
                if (syncIntrevalResult) {
                    return true
                } else {
                    this.setState({
                        validateStatus: {
                            ...this.state.validateStatus,
                            syncIntervalValidateStatus: syncIntrevalResult ? ValidateStatus.Normal : ValidateStatus.Empty,
                        },
                    })
                    return false
                }
        }
    }

    /**
     * 保存定期同步设置
     */
    protected saveDomainConfig = async (): Promise<void> => {
        const { id, name } = this.props.domainInfo;
        const { syncSettingInfo: { periodicSyncStatus, syncObject, syncInterval, syncTarget, expireTime, syncMode, userStatus, csfLevel, csfOptions, syncIntervalUnit } } = this.state;

        if (!periodicSyncStatus) {
            try {
                await setDomainSyncStatus([id, -1]);

                manageLog(
                    ManagementOps.SET,
                    __('关闭 域控 “${name}” 定期同步 成功', { name: name }),
                    '',
                    Level.INFO,
                )

                this.setState({
                    isSyncSettingEditStatus: false,
                })

                this.originSyncSettingInfo = this.state.syncSettingInfo

                this.props.onRequestEditStatus(false)
                this.props.onSetSyncSetInfoSuccess()

            } catch (ex) {
                this.handleError(ex)
            }
        } else {
            if (this.verifySyncSettingInfo()) {
                const ncTUsrmDomainConfig = {
                    destDepartId: syncTarget.length ? syncTarget[0].id : '',
                    desetDepartName: syncTarget.length ? syncTarget[0].name : '',
                    ouPath: syncObject.length ? syncObject.some((item) => item.ipAddress || !item.pathName) ? [] : syncObject.map((item) => item.pathName) : [],
                    syncInterval: this.getSyncTime(),
                    userEnableStatus: userStatus,
                    syncMode,
                    validPeriod: expireTime,
                    csfLevel,
                }

                try {
                    await setDomainSyncStatus([id, 0])
                    manageLog(
                        ManagementOps.SET,
                        __('开启 域控 “${name}” 定期同步 成功', { name }),
                        '',
                        Level.INFO,
                    )
                    await setDomainConfig([id, { ncTUsrmDomainConfig }])

                    this.setState({
                        isSyncSettingEditStatus: false,
                    })

                    const originSyncSettingInfo = this.originSyncSettingInfo
                    this.originSyncSettingInfo = this.state.syncSettingInfo

                    let syncIntervalText = syncInterval + (syncIntervalUnit === SyncUnit.Minutes ? __('分钟') : syncIntervalUnit === SyncUnit.Hour ? __('小时') : __('天'))
                    if(originSyncSettingInfo.syncIntervalUnit !== syncIntervalUnit||originSyncSettingInfo.syncInterval !== syncInterval){
                        const newSyncInterval = syncIntervalText
                        const oldSyncInterval = originSyncSettingInfo.syncInterval + (originSyncSettingInfo.syncIntervalUnit === SyncUnit.Minutes ? __('分钟') : originSyncSettingInfo.syncIntervalUnit === SyncUnit.Hour ? __('小时') : __('天'))
                        syncIntervalText = __('由 ${oldText} 改为 ${newText}', { oldText: oldSyncInterval, newText: newSyncInterval })
                    }

                    let csfLevelText = csfOptions.filter((item) => item.value === csfLevel)[0].name
                    if(originSyncSettingInfo.csfLevel !== csfLevel){
                        const oldCsfLevelText = csfOptions.filter((item) => item.value === originSyncSettingInfo.csfLevel)[0].name
                        csfLevelText = __('由 ${oldText} 改为 ${newText}', { oldText: oldCsfLevelText, newText: csfLevelText })
                    }

                    this.props.onRequestEditStatus(false)
                    this.props.onSetSyncSetInfoSuccess()

                    manageLog(
                        ManagementOps.SET,
                        __('设置 域 “${name}” 同步目标为 “${ouPath}”', {
                            name: name,
                            ouPath: ncTUsrmDomainConfig.desetDepartName || name,
                        }),
                        __('同步周期 “${syncInterval}”；新建用户密级 “${csfLevel}”；用户有效期限 “${validPeriod}”；用户默认状态 “${syncStatus}”；同步方式 “${syncMode}”', {
                            syncInterval: syncIntervalText,
                            csfLevel: csfLevelText,
                            validPeriod: ExpireTimes[expireTime],
                            syncStatus: userStatus ? __('启用') : __('禁用'),
                            syncMode: syncMode === 0 ? __('同步选中的对象及其成员（包括上层的组织结构）') : syncMode === 1 ? __('同步选中的对象及其成员（不包括上层的组织结构）') : __('仅同步用户账号（不包括组织结构）'),
                        }),
                        Level.INFO,
                    )

                    startSync([id.toString(), 1])

                } catch (ex) {
                    this.handleError(ex)
                }
            }
        }
    }

    /**
    * 取消定期同步设置
    */
    protected cancelDomainConfig = (): void => {
        this.setState({
            syncSettingInfo: {
                ...this.originSyncSettingInfo,
            },
            validateStatus: {
                syncIntervalValidateStatus: ValidateStatus.Normal,
            },
            isSyncSettingEditStatus: false,
        })
        this.props.onRequestEditStatus(false)
    }

    /**
     * 错误处理函数
     */
    private handleError = async (ex): Promise<void> => {
        const { syncSettingInfo: { syncTarget } } = this.state;

        switch (ex.error.errID) {
            case ErrorCode.DomainUnavailable:
                Message2.info({ message: __('连接LDAP服务器失败，请检查域控制器地址是否正确，或者域控制器是否开启。'), zIndex: 10000 })
                break

            case ErrorCode.DomainAlreadyExists:
                Message2.info({ message: __('已存在相同的域名。') })
                break

            case ErrorCode.DomainNotExists:
                this.props.onRequestDomainInvalid(this.props.domainInfo.name)
                break

            case ErrorCode.SyncTargetNotExist:
                Message2.info({ message: __('同步目标的部门“${name}”不存在。', { name: syncTarget[0].name }) })
                break
        }
    }
}