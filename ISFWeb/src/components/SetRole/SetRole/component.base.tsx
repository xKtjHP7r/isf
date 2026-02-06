import * as React from 'react';
import { noop, includes } from 'lodash';
import * as PropTypes from 'prop-types';
import WebComponent from '../../webcomponent';
import { getConfidentialConfig } from '@/core/apis/eachttp/config/config';
import { ShareMgnt } from '@/core/thrift';
import { manageLog, Level, ManagementOps } from '@/core/log';
import { SystemRoleType, getRoleName } from '@/core/role/role';
import { getErrorMessage } from '@/core/exception';
import { transformAdmins } from '@/core/confidential/confidential'
import { Message } from '@/sweet-ui';
import styles from './styles.desktop';
import __ from './locale';

/**
 * 可编辑的角色
 */
export const EditableRoles = [
    // 组织管理员
    SystemRoleType.OrgManager,
    // 组织审计员
    SystemRoleType.OrgAudit,
]

/**
 * 可登录控制台的角色
 */
export const LoginConsoleRoles = [
    // 超级管理员
    SystemRoleType.Supper,
    // 系统管理员
    SystemRoleType.Admin,
    // 安全管理员
    SystemRoleType.Securit,
    // 审计管理员
    SystemRoleType.Audit,
    // 组织管理员
    SystemRoleType.OrgManager,
    // 组织审计员
    SystemRoleType.OrgAudit,
]

export default class SetRoleComponentBase extends WebComponent<Console.SetRoleComponent.Props, Console.SetRoleComponent.State> {
    static defaultProps = {
        users: [],
        dep: null,
        userid: '',
        onComplete: noop,
    }

    static contextTypes = {
        toast: PropTypes.func,
    }

    state = {
        // 搜索框输入值
        value: '',
        // 搜索结果
        results: [],
        // 选择的用户
        userInfo: this.props.users[0],
        // 所选用户已拥有的角色
        ownRole: [],
        // 登录用户所有可操作的角色
        allSelectableRoles: [],
        // 点选给用户配置的角色
        selectRoleInfo: null,
        // 显示可配置角色的配置界面
        showRoleEditDialog: false,
        /**
         * 所有可配置的角色，用于实现ASE产品用户已经拥有的共享审核员和定密审核员可编辑和删除，
         * 从老版本升级到ASE，用户已经拥有的共享审核员和定密审核员不屏蔽
         */
        allRoles: [],
        // 当前登录用户角色信息
        roles: [],
    }
    /**
     * 当前正在编辑的一条角色配置
     */
    roleRateInEdit = null;

    /**
     * 是否支持定密审核，不支持的产品屏蔽定密审核员
     */
    supportSecurityApprove = true

    async componentDidMount() {
        // this.supportSecurityApprove = await hasFeature(Features.SecurityApprove)
        await Promise.all([
            ShareMgnt('Usrm_GetUserInfo', [this.state.userInfo.id]),
            ShareMgnt('UsrRolem_Get', [this.props.userid]),
            ShareMgnt('Usrm_GetUserInfo', [this.props.userid]),
            getConfidentialConfig('disabled_roles'),
        ]).then(([selectUserInfo, allRoles, loginUserInfo, disabledRoles]) => {

            // 存储当前登录的限额信息、角色信息
            if ( loginUserInfo.user.roles.some((item) => item.id === SystemRoleType.OrgManager) ) {
                const { user: { roles } } = loginUserInfo
                this.setState({
                    roles: roles,
                })
            } else {
                const { user: { roles } } = loginUserInfo
                this.setState({
                    roles: roles,
                })
            }

            const allSelectableRoles = this.filterRoles(allRoles)
            const disabledRoleIds = transformAdmins(disabledRoles)

            if (
                loginUserInfo.user.roles &&
                loginUserInfo.user.roles.some((role) => [SystemRoleType.Supper, SystemRoleType.Securit].includes(role.id))
            ) {
                this.setState({
                    ownRole: [...this.filterRoles(selectUserInfo.user.roles)],
                    allSelectableRoles: [...allSelectableRoles].filter(({ id }) => !disabledRoleIds.includes(id)),
                    // 对于已有的用户角色，不管什么产品都显示
                    allRoles: [...allSelectableRoles],
                })
            } else {
                const selectableRolesWithOutOrgAudit = allSelectableRoles.filter((role) => role.id !== SystemRoleType.OrgAudit)
                this.setState({
                    ownRole: [...this.filterRoles(selectUserInfo.user.roles)],
                    allSelectableRoles: this.supportSecurityApprove ?
                        selectableRolesWithOutOrgAudit.filter(({ id }) => !disabledRoleIds.includes(id))
                        :
                        allSelectableRoles.filter((role) => ![
                            SystemRoleType.OrgAudit, ...disabledRoleIds,
                        ].includes(role.id)),
                    allRoles: selectableRolesWithOutOrgAudit,
                })
            }
        }).catch((ex) => {
            if (ex.error && ex.error.errID) {
                this.popErrMessage(ex.error.errID)
            } else if (ex.code) {
                this.context.toast(ex.message)
            }
        })
    }

    /**
     * 升级时，去除原有角色数据数组中包含的共享、定密、文档审核员
     * @param info 信息
     */
    protected filterRoles = (roles: any): any => {

        return roles.filter((role) => {
            return !includes([
                SystemRoleType.SharedApprove,
                SystemRoleType.CsfApprove,
                SystemRoleType.DocApprove,
            ], role.id)
        })
    }

    /**
     * 输入变化
     * @param key 输入值
     */
    protected handleChange = (key) => {
        this.setState({
            value: key,
        })
    }

    /**
     * 根据输入值搜索角色
     */
    protected searchRoles = (key) => {
        const results = key !== '' ? this.state.allSelectableRoles.filter((item) => getRoleName(item).includes(key)) : this.state.allSelectableRoles
        return results
    }

    /**
     * 加载搜索结果
     * @param results
     */
    protected getSearchData = (results) => {
        this.setState({
            results,
        })
    }

    /**
     * 聚焦
     */
    protected handleFocus = () => {
        if (this.state.value === '') {
            this.refs.autocomplete.toggleActive(true);
            this.setState({
                results: this.state.allSelectableRoles,
            })
        }
    }

    /**
     * 按下enter
     */
    protected handleEnter = (e, selectIndex: number) => {
        if (selectIndex >= 0) {
            this.handleChooseRole(this.state.results[selectIndex])
        }
    }

    /**
     * 选择配置的角色
     */
    protected handleChooseRole = async (data) => {
        const { userInfo } = this.state
        if (this.state.ownRole.some((item) => item.id === data.id)) {
            this.context.toast(__('当前用户已拥有该角色。'));
        } else {
            if (!EditableRoles.includes(data.id)) {
                let memberInfo = {
                    ncTRoleMemberInfo: {
                        userId: userInfo.id,
                        displayName: userInfo.user.displayName,
                        departmentIds: userInfo.user.departmentIds,
                        departmentNames: userInfo.user.departmentNames,
                        manageDeptInfo: null,
                    },
                }
                try {
                    await ShareMgnt('UsrRolem_SetMember', [this.props.userid, data.id, memberInfo]);
                    manageLog(
                        ManagementOps.SET,
                        __('为用户“${displayName}” 添加 “${roleName}”角色 成功', {
                            displayName: userInfo.user.displayName,
                            roleName: getRoleName(data),
                        }),
                        '',
                        Level.INFO,
                    )
                    this.context.toast(__('添加成功'));
                    this.setState({
                        ownRole: [{ ...data }, ...this.state.ownRole],
                    })
                } catch (ex) {
                    if (ex.error && ex.error.errID) {
                        this.popErrMessage(ex.error.errID)
                    } else if (ex.code) {
                        this.context.toast(ex.message);
                    }
                }
            } else {
                this.setState({
                    showRoleEditDialog: true,
                    selectRoleInfo: data,
                })
            }
        }
        this.refs.autocomplete.toggleActive(false);
        this.refs.autocomplete.blur();
    }

    /**
     * 编辑角色
     * @param data 正在编辑的角色
     */
    protected editRoleInfo = async (data) => {
        try {
            this.roleRateInEdit = await ShareMgnt('UsrRolem_GetMemberDetail', [this.props.userid, data.id, this.state.userInfo.id]);
            this.setState({
                showRoleEditDialog: true,
                selectRoleInfo: data,
            });
        } catch (ex) {
            this.popErrMessage(ex.error.errID)
        }
    }

    /**
     * 删除角色
     */
    protected deleteRoleInfo = async (data) => {
        if (await Message.confirm({
            message: <div key={'deleteRoleInfo'} className={styles['del-text']}>{__('您确定要移除 “${displayName}” 的 “${roleName}” 角色吗？', {
                displayName: this.state.userInfo.user.displayName,
                roleName: getRoleName(data),
            })}</div>,
        })) {
            this.setState({
                selectRoleInfo: data,
            }, () => {
                this.confirmDeleteUsers()
            })
        }
    }

    /**
     * 确认删除角色
     */
    protected confirmDeleteUsers = async () => {
        const { ownRole, userInfo, selectRoleInfo } = this.state;
        let auditRange = [];
        if (EditableRoles.includes(selectRoleInfo.id)) {
            try {
                let roleMemberInfo = await ShareMgnt('UsrRolem_GetMemberDetail', [this.props.userid, selectRoleInfo.id, userInfo.id]);
                auditRange = roleMemberInfo.manageDeptInfo ?
                    roleMemberInfo.manageDeptInfo.departmentNames : []
                try {
                    await ShareMgnt('UsrRolem_DeleteMember', [this.props.userid, selectRoleInfo.id, userInfo.id]);
                    this.setState({
                        ownRole: ownRole.filter((item) => item.id !== selectRoleInfo.id),
                    })

                    if (selectRoleInfo.id === SystemRoleType.OrgManager) {
                        manageLog(
                            ManagementOps.DELETE,
                            __('取消 “${userName}” 的组织管理员身份，管辖部门：“${departmentName}”', { userName: userInfo.user.displayName, departmentName: auditRange.join(__('”，“')) }),
                            null,
                            Level.INFO,
                        )
                    } else {
                        manageLog(
                            ManagementOps.SET,
                            __('为用户“${displayName}” 删除 “${roleName}”角色 成功', {
                                displayName: userInfo.user.displayName,
                                roleName: getRoleName(selectRoleInfo),
                            }),
                            EditableRoles.includes(selectRoleInfo.id) && auditRange.length ?
                                selectRoleInfo.id === SystemRoleType.OrgManager || selectRoleInfo.id === SystemRoleType.OrgAudit ?
                                    __('“${displayName}”的管辖范围为：“${rangeName}”', {
                                        displayName: userInfo.user.displayName,
                                        rangeName: auditRange.join(__('”，“')),
                                    }) :
                                    __('“${displayName}”的审核范围为：“${rangeName}”', {
                                        displayName: userInfo.user.displayName,
                                        rangeName: auditRange.join(__('”，“')),
                                    })
                                : '',
                            Level.INFO,
                        )
                    }
                } catch (ex) {
                    this.popErrMessage(ex.error.errID)
                }
            } catch (ex) {
                this.popErrMessage(ex.error.errID)
            }
        } else {
            try {
                await ShareMgnt('UsrRolem_DeleteMember', [this.props.userid, selectRoleInfo.id, userInfo.id]);

                if (selectRoleInfo.id === SystemRoleType.OrgManager) {
                    manageLog(
                        ManagementOps.DELETE,
                        __('取消 “${userName}” 的组织管理员身份，管辖部门：“${departmentName}”', { userName: userInfo.user.displayName, departmentName: auditRange.join(__('”，“')) }),
                        null,
                        Level.INFO,
                    )
                } else {
                    manageLog(
                        ManagementOps.SET,
                        __('为用户“${displayName}” 删除 “${roleName}”角色 成功', {
                            displayName: userInfo.user.displayName,
                            roleName: getRoleName(selectRoleInfo),
                        }),
                        '',
                        Level.INFO,
                    )
                }

                this.setState({
                    ownRole: ownRole.filter((item) => item.id !== selectRoleInfo.id),
                })
            } catch (ex) {
                if (ex.error && ex.error.errID) {
                    this.popErrMessage(ex.error.errID)
                } else if (ex.code) {
                    switch (ex.code) {
                        default:
                            this.context.toast(ex.message)
                    }
                } else {
                    this.context.toast(ex)
                }
            }
        }
    }

    /**
     * 取消操作
     */
    protected handleCancelSetRoleConfig = () => {
        this.setState({
            showRoleEditDialog: false,
        });
        this.roleRateInEdit = null;
    }

    /**
     * 确定或添加角色
     */
    protected handleConfirmSetRoleConfig = async (roleConfig) => {
        const { ownRole, userInfo } = this.state;
        let memberInfo = {
            ncTRoleMemberInfo: {
                userId: userInfo.id,
                displayName: userInfo.user.displayName,
                departmentIds: userInfo.user.departmentIds,
                departmentNames: userInfo.user.departmentNames,
                manageDeptInfo: roleConfig.manageRange ? roleConfig.manageRange : null,
            },
        }
        let auditRange = roleConfig.manageRange.ncTManageDeptInfo.departmentNames

        try {
            await ShareMgnt('UsrRolem_SetMember', [this.props.userid, roleConfig.id, memberInfo])
            this.setState({
                showRoleEditDialog: false,
            });
            if (!this.roleRateInEdit) {
                this.context.toast(__('添加成功'));
                this.setState({
                    ownRole: [{ ...roleConfig }, ...ownRole],
                })
                if (roleConfig.id === SystemRoleType.OrgManager) {
                    manageLog(
                        ManagementOps.SET,
                        __('将 “${userName}” 设为组织管理员，管辖部门：“${departmentName}”', { userName: userInfo.user.displayName, departmentName: auditRange.join(__('”，“')) }),
                        '',
                        Level.INFO,
                    )
                } else {
                    manageLog(
                        ManagementOps.SET,
                        __('为用户“${displayName}” 添加 “${roleName}”角色 成功', {
                            displayName: userInfo.user.displayName,
                            roleName: roleConfig.name,
                        }),
                        EditableRoles.includes(roleConfig.id) ?
                            roleConfig.id === SystemRoleType.OrgManager || roleConfig.id === SystemRoleType.OrgAudit ?
                                __('“${displayName}”的管辖范围为：“${rangeName}”', {
                                    displayName: userInfo.user.displayName,
                                    rangeName: auditRange.join(__('”，“')),
                                }) :
                                __('“${displayName}”的审核范围为：“${rangeName}”', {
                                    displayName: userInfo.user.displayName,
                                    rangeName: auditRange.join(__('”，“')),
                                })
                            : '',
                        Level.INFO,
                    )
                }
            } else {
                if (roleConfig.id === SystemRoleType.OrgManager) {
                    manageLog(
                        ManagementOps.SET,
                        __('编辑 “${userName}” 为组织管理员，管辖部门：“${departmentName}”', { userName: userInfo.user.displayName, departmentName: auditRange.join(__('”，“')) }),
                        '',
                        Level.INFO,
                    )
                } else {
                    manageLog(
                        ManagementOps.SET,
                        __('为用户“${displayName}” 编辑 “${roleName}”角色 成功', {
                            displayName: userInfo.user.displayName,
                            roleName: roleConfig.name,
                        }),
                        EditableRoles.includes(roleConfig.id) ?
                            roleConfig.id === SystemRoleType.OrgManager || roleConfig.id === SystemRoleType.OrgAudit ?
                                __('“${displayName}”的管辖范围为：“${rangeName}”', {
                                    displayName: userInfo.user.displayName,
                                    rangeName: auditRange.join(__('”，“')),
                                }) :
                                __('“${displayName}”的审核范围为：“${rangeName}”', {
                                    displayName: userInfo.user.displayName,
                                    rangeName: auditRange.join(__('”，“')),
                                })
                            : '',
                        Level.INFO,
                    )
                }
            }
        } catch (ex) {
            this.popErrMessage(ex.error.errID)
        } finally {
            this.roleRateInEdit = null;
        }
    }

    /**
     * 弹出错误信息
     */
    private popErrMessage(errId) {
        if (errId === 22909) {
            this.context.toast(__('角色不存在。'));
        } else {
            this.context.toast(getErrorMessage(errId));
        }
    }
}