import * as React from 'react';
import { isFunction, noop } from 'lodash';
import session from '@/util/session';
import { syncDelay, timer } from '@/util/timer';
import { Message2 as Message } from '@/sweet-ui';
import { usrmImportDomainUsers, usrmImportDomainOUs, usrmClearImportProgress, usrmGetImportProgress } from '@/core/thrift/sharemgnt/sharemgnt';
import { ErrorCode } from '@/core/thrift/sharemgnt/errcode';
import WebComponent from '../webcomponent';
import __ from './locale';
import { getLevelConfig } from '@/core/apis/console/usermanagement';

/**
 * 导入方式
 */
export enum ImportStyle {
    /**
     * 导入选中的对象及其成员（包括上层的组织结构）
     */
    All,

    /**
     * 导入选中的对象及其成员（不包括上层的组织结构）
     */
    SelectedDepAndUser,

    /**
     * 仅导入用户账号（不包括组织结构
     */
    Users,
}

/**
 * 渲染类型
 */
export enum RenderType {
    /**
     * 渲染设置界面
     */
    View,

    /**
     * 渲染进度
     */
    Progress,

    /**
     * 默认
     */
    Null,
}

interface ImportDomainUserProps extends React.Props<void> {
    /**
     * 部门id
     */
    departmentId: string;

    /**
     * 关闭弹窗
     */
    onRequestCancel: () => void;

    /**
     * 成功后回调
     */
    onRequestSuccess: (importStyle: ImportStyle) => void;

    /**
     * 跳转域认证集成
     */
    doRedirectDomain: () => void;
}

interface ImportDomainUserState {
    /**
     * 渲染类型
     */
    renderType: RenderType;

    /**
     * 导入方式
     */
    importStyle: ImportStyle;

    /**
     * 覆盖同名用户
     */
    userCover: boolean;

    /**
     * 用户默认装态
     */
    userStatus: boolean;

    /**
     * 用户有效期限
     */
    expireTime: number;

    /**
     * 导入进度
     */
    progress: number;

    /**
     * 选中项
     */
    selected: ReadonlyArray<any>;

    /**
     * 密级
     */
    csfLevel: number;

    /**
     * 密级集合
     */
    csfOptions: Array<{ value: number, name: string }>;
}

export default class ImportDomainUserBase extends WebComponent<ImportDomainUserProps, ImportDomainUserState> {
    static defaultProps = {
        onRequestCancel: noop,
        onRequestSuccess: noop,
        doRedirectDomain: noop,
    }

    state: ImportDomainUserState = {
        renderType: RenderType.View,
        importStyle: ImportStyle.All,
        userCover: true,
        userStatus: true,
        expireTime: -1,
        progress: 0,
        selected: [],
        csfLevel: null,
        csfOptions: [],
    }

    /**
     * 树结构选中的节点
     */
    selectTree: ReadonlyArray<any> = []

    /**
     * 导入初始化是否完成
     */
    importProgressInit: boolean = false

    /**
     * 错误信息
     */
    failInfos: ReadonlyArray<any> = []

    async componentDidMount() {
        try {
            const { csf_level_enum } = await getLevelConfig({ fields: 'csf_level_enum' })

            this.setState({
                csfOptions: csf_level_enum,
                csfLevel: csf_level_enum?.[0]?.value,
            })
        } catch (ex) {
            if (ex.error.errMsg) {
                Message.info({ message: ex.error.errMsg })
            }
        }
    }

    /**
     * 点击选中
     */
    protected handleSelect = (selected: ReadonlyArray<any>, search?: boolean): void => {
        if (search) {
            this.setState({
                selected,
            })
        } else {
            this.selectTree = selected
        }
    }

    /**
     * 密级切换
     */
    protected updateCsfLevel(csfLevel: number): void {
        this.setState({
            csfLevel,
        })
    }

    /**
     * 添加
     */
    protected addTreeData = (): void => {
        if (this.selectTree.length) {
            this.setState({
                selected: this.selectTree,
            })
            this.ref.cancelSelections()
            this.selectTree = []
        }
    }

    /**
    * 删除已选
    * @param detail 部门/成员
    */
    protected deleteSelected = (detail: any): void => {
        this.setState({
            selected: this.state.selected.filter((value) => value.name ? value.name !== detail.name : value.displayName !== detail.displayName),
        })
    }

    /**
     * 节点获取上层组织结构
     */
    private getSupOrg(data, arr = []) {
        if (!data.parentNode) {
            return arr
        }

        if (data.parentNode.parentNode) {
            arr = [...arr, {
                ncTUsrmDomainOU: {
                    name: data.parentNode.name,
                    parentOUPath: data.parentNode.parentOUPath,
                    objectGUID: data.parentNode.objectGUID,
                    pathName: data.parentNode.pathName,
                    rulerName: data.parentNode.rulerName || null,
                    importAll: false,
                },
            }]
        }

        return this.getSupOrg(data.parentNode, arr)
    }

    /**
     * 处理接口参数
     */
    private getImportData = (item): { ous: ReadonlyArray<any>; users: ReadonlyArray<any> } => {
        const { importStyle } = this.state;

        let data = {
            ous: [],
            users: [],
        }

        if (item.pathName) {
            data = {
                ...data,
                ous: importStyle === ImportStyle.All && item.parentNode.domainInfo ?
                    [
                        ...(data.ous || []),
                        ...(this.getSupOrg(item)),
                        {
                            ncTUsrmDomainOU: {
                                name: item.name,
                                parentOUPath: item.parentOUPath,
                                objectGUID: item.objectGUID,
                                pathName: item.pathName,
                                rulerName: item.rulerName || null,
                                importAll: true,
                            },
                        },
                    ] : [
                        ...(data.ous || []),
                        {
                            ncTUsrmDomainOU: {
                                name: item.name,
                                parentOUPath: item.parentOUPath,
                                objectGUID: item.objectGUID,
                                pathName: item.pathName,
                                rulerName: item.rulerName || null,
                                importAll: true,
                            },
                        },
                    ],
            }
        } else {
            data = {
                ...data,
                ous: importStyle === ImportStyle.All ?
                    [
                        ...(data.ous || []),
                        ...(this.getSupOrg(item)),
                    ] : [],
                users: [
                    ...(data.users || []),
                    {
                        ncTUsrmDomainUser: {
                            ouPath: item.ouPath,
                            displayName: item.displayName,
                            objectGUID: item.objectGUID,
                            loginName: item.loginName,
                            dnPath: item.dnPath,
                            email: item.email,
                            idcardNumber: item.idcardNumber,
                        },
                    },
                ],
            }
        }

        return data
    }

    /**
     * 点击导入按钮
     */
    protected confirmImport = async () => {

        const { userCover, userStatus, expireTime, selected, csfLevel } = this.state;

        const selectedDomains = selected.reduce((prev, item) => {
            /**
             * 此项为部门或者用户
             */
            if (item.domainInfo) {
                const { ous, users } = this.getImportData(item)

                const prevDomain = prev[item.domainInfo.ncTUsrmDomainInfo.name]

                prev = {
                    ...prev,
                    [item.domainInfo.ncTUsrmDomainInfo.name]: {
                        ...prevDomain,
                        domain: item.domainInfo,
                        domainName: null,
                        ous: [
                            ...(prevDomain && prevDomain.ous || []),
                            ...ous,
                        ],
                        users: [
                            ...(prevDomain && prevDomain.users || []),
                            ...users,
                        ],
                    },
                }
            }
            /**
             * 此项为组织
             */
            else {
                prev[item.name] = {
                    domain: item,
                    domainName: item.name,
                    ous: [],
                    users: [],
                }
            }
            return prev
        }, {})

        for (let key in selectedDomains) {
            const { domain, domainName, ous, users } = selectedDomains[key];

            this.failInfos = [];

            /**
             * 接口所需域参数
             */
            const importContent = {
                ncTUsrmImportContent: {
                    domain: domain.ncTUsrmDomainInfo ? domain : {
                        ncTUsrmDomainInfo: {
                            ...domain,
                            config: {
                                ncTUsrmDomainConfig: {
                                    ...domain.config,
                                },
                            },
                        },
                    },
                    domainName,
                    users,
                    ous,
                },
            }

            /**
             * 接口所需配置参数
             */
            const importOption = {
                ncTUsrmImportOption: {
                    userEmail: true,
                    userDisplayName: true,
                    userCover,
                    userStatus: userStatus ? 0 : 1,
                    expireTime: expireTime === -1 ? -1 : expireTime / 1000000,
                    departmentId: this.props.departmentId,
                    csfLevel,
                },
            }

            await this.getProgress(importContent, importOption)

            if (this.failInfos && this.failInfos.length) {
                await Message.info({ message: this.failInfos[0].match(/error:(.*)/)[1] })
            }
        }
    }

    /**
     * 获取导入进度
     */
    private getProgress = async (importContent: Core.ShareMgnt.ncTUsrmImportContent, importOption: Core.ShareMgnt.ncTUsrmImportOption) => {

        return new Promise(async (resolve, reject) => {

            const { importStyle } = this.state;

            await usrmClearImportProgress()

            if (importStyle === ImportStyle.Users) {
                usrmImportDomainUsers([importContent, importOption, session.get('isf.userid')]).then(() => {
                    this.importProgressInit = true
                }).catch(() => {
                    this.importProgressInit = true
                })
            } else {
                usrmImportDomainOUs([importContent, importOption, session.get('isf.userid')]).then(() => {
                    this.importProgressInit = true
                }).catch(async (ex) => {
                    this.importProgressInit = true

                    switch (ex.error.errID) {
                        case ErrorCode.ImportDomainAgain:
                            isFunction(stopTimer) && stopTimer()
                            this.props.onRequestCancel()
                            Message.error({
                                message: __('导入失败，不能重复导入组织。'),
                            })

                            reject()
                            break;

                        default:
                            isFunction(stopTimer) && stopTimer()
                            this.props.onRequestCancel()
                            Message.error({
                                message: ex.error.errMsg,
                            })
                            reject()
                    }
                    reject()
                    return
                })
            }

            this.setState({
                renderType: RenderType.Progress,
                progress: 0,
            })

            const stopTimer = timer(async () => {

                let { successNum, failNum, totalNum, failInfos, disableNum } = await usrmGetImportProgress()

                // 避免上次导入还在后台继续，请求数据后 successNum 或 failNum 不为 0 但是 totalNum 为 0 导致的死循环
                if (successNum + failNum > 0 && totalNum === 0) {
                    stopTimer()
                    reject()
                }

                // 如果 usrmGetImportProgress 接口后端未初始化完成，等待 3s 后再次请求，直到成功
                if (!this.importProgressInit && totalNum === 0) {
                    await syncDelay(3000);

                    ({ successNum, failNum, totalNum, failInfos, disableNum } = await usrmGetImportProgress())

                    if (totalNum !== 0) {
                        this.importProgressInit = true
                    } else {
                        return
                    }
                }

                if (failInfos.length > 0) {
                    this.failInfos = [...this.failInfos, ...failInfos]

                    this.setState({
                        progress: 1,
                    })
                    stopTimer()
                    this.props.onRequestSuccess(this.state.importStyle)
                    resolve()
                } else {
                    this.setState({
                        progress: totalNum === 0 ? 1 : Number(((successNum + failNum) / totalNum).toFixed(2)),
                    })
                }

                if (successNum + failNum === totalNum && totalNum >= 0) {
                    stopTimer()
                    if (this.state.userStatus && disableNum > 0) {
                        if (await Message.info({ message: __('全部导入成功，启用的用户数已达到用户许可总数上限，超出的部分账号已被禁用。') })) {
                            this.props.onRequestSuccess(this.state.importStyle)
                            this.props.onRequestCancel()
                        }
                        this.props.onRequestSuccess(this.state.importStyle)
                    } else {
                        this.props.onRequestSuccess(this.state.importStyle)
                    }
                    resolve()
                }
            }, 100)
        })
    }

    /**
     * 设置有效期限
     */
    protected changeExpireTime = (expireTime: number): void => {
        this.setState({
            expireTime,
        })
    }

    /**
     * 清空已选择部门或用户
     */
    protected clearSelected = () => {
        this.setState({
            selected: [],
        })

        this.ref.cancelSelections()
    }
}