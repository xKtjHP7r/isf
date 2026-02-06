import { noop } from 'lodash'
import { Message2 as Message, Toast } from '@/sweet-ui'
import {
    getTriSystemStatus,
    getThirdPartyAuth,
    getFreezeStatus,
} from '@/core/thrift/sharemgnt/sharemgnt'
import { getConfidentialConfig } from '@/core/apis/eachttp/config/config'
import { SystemRoleType } from '@/core/role/role'
import { ErrorCode } from '@/core/thrift/sharemgnt/errcode'
import session from '@/util/session'
import { DataNode } from '@/sweet-ui/components/DragTree/helper'
import { ImportStyle } from '../ImportDomainUser/component.base'
import WebComponent from '../webcomponent'
import {
    MenuGroup,
    getMenuGroupsByRole,
    changeActionStatusByDep,
    changeActionStatusBySelectedUsers,
    Action,
    getIsShowManagePwd,
    getIsShowSetRole,
    getIsShowEnableAndDisableUser,
} from './helper'
import __ from './locale'
import AppConfigContext from '@/core/context/AppConfigContext'

interface UserOrgMgntState {
    /**
     * 菜单栏
     */
    menus: ReadonlyArray<MenuGroup>;

    /**
     * 操作类型
     */
    action: Action;

    /**
     * 选中的部门或组织
     */
    selectedDep: DataNode | null;

    /**
     * 选中的用户
     */
    selectedUsers: ReadonlyArray<Core.ShareMgnt.ncTUsrmGetUserInfo>;

    /**
     * 是否显示初始化弹窗
     */
    isShowInit: boolean;

    /**
     * 是否显示用户角色
     */
    isShowSetRole: boolean | SystemRoleType;

    /**
     * 
     */
    urlParams: any
}

export default class UserOrgMgnt extends WebComponent<any, UserOrgMgntState> {
    static contextType = AppConfigContext
    static defaultProps = {
        doRedirectDomain: noop,
    }

    state = {
        menus: [],
        action: Action.None,
        selectedDep: null,
        selectedUsers: [],
        isShowInit: false,
        isShowSetRole: false,
        urlParams: null,
    }

    /**
     * 当前登录用户的信息
     */
    userInfo = session.get('isf.userInfo')

    /**
     * 用户id
     */
    userid = session.get('isf.userid')

    /**
     * OrgTree的ref
     */
    orgTree

    /**
     * Grid的ref
     */
    userGrid

    /**
     * 是否开启冻结用户功能
     */
    freezeStatus: boolean

    /**
     * 是否显示启用/禁用用户
     */
    isShowEnableAndDisableUser: boolean

    /**
     * 是否禁用空间站
     */
    isKjzDisabled: boolean = true

    /**
     * 是否显示交接工作前置弹窗
     */
    isShowPrePosition = false

    async componentDidMount() {
        try {
            const params = new URLSearchParams(location.search)
            this.setState({
                urlParams: params
            })
            // 获取是否开启冻结用户功能
            this.freezeStatus = await getFreezeStatus()

            // 获取三权分离的状态
            const triSystemStatus = await getTriSystemStatus()

            // 获取是否显示管控密码
            const isShowManagePwd = await getIsShowManagePwd()

            // 获取是否显示用户角色
            const isShowSetRole = await getIsShowSetRole(triSystemStatus)

            // 获取是否显示启用/禁用用户
            this.isShowEnableAndDisableUser = await getIsShowEnableAndDisableUser()

            // 获取空间站是否禁用
            const res = await getConfidentialConfig('kjz_disabled')
            this.isKjzDisabled = (typeof res === 'undefined' || res)

            // 获取是否开启了第三方认证集成
            const { enabled: enableThirdImport } = await getThirdPartyAuth()

            const menus = getMenuGroupsByRole({ roles: this.userInfo.user.roles, triSystemStatus, freezeStatus: this.freezeStatus, enableThirdImport, isShowManagePwd, isShowSetRole, isShowEnableAndDisableUser: this.isShowEnableAndDisableUser })

            // 超级管理员、系统管理员、安全管理员、审计管理员可以看到初始化密级界面
            const isManager = this.userInfo.user.roles.some((role) => [SystemRoleType.Supper, SystemRoleType.Admin, SystemRoleType.Securit, SystemRoleType.Audit].includes(role.id))

            this.setState({
                menus: changeActionStatusByDep(menus, this.state.selectedDep),
                isShowSetRole,
                isShowInit: isManager,
            })
        } catch (ex) {
            if (ex && ex.error && ex.error.errID) {
                Message.info({ message: this.getErrorMsg(ex) })
            }
        }
    }

    /**
     * 在组件销毁后设置state，防止内存泄漏
     */
    componentWillUnmount() {
        this.setState = (state, callback) => {
            return
        }
    }

    /**
     * 改变操作
     */
    protected changeAction = async (action: Action): Promise<void> => {
        this.setState({
            action,
        })
        this.isShowPrePosition = false
    }

    /**
     * 新建组织/部门成功
     */
    protected addTreeNode = (depInfo: Core.ShareMgnt.ncTDepartmentInfo, action: Action): void => {
        if (action === Action.CreateOrg) {
            this.orgTree.addNode(depInfo)
        } else {
            this.orgTree.addNode(depInfo, this.state.selectedDep.data)
        }

        this.changeAction(Action.None)
    }

    /**
     * 编辑组织/部门成功
     */
    protected updateTreeNode = (depInfo: Core.ShareMgnt.ncTDepartmentInfo): void => {
        this.changeAction(Action.None)

        this.orgTree.updateNode(depInfo, { ossInfo: depInfo.ossInfo }, depInfo.changeChild)
    }

    /**
     * 删除组织/部门成功
     */
    protected deleteTreeNode = (depInfo: Core.ShareMgnt.ncTDepartmentInfo): void => {
        this.changeAction(Action.None)

        this.orgTree.deleteNode(depInfo)
    }

    /**
     * 移动部门
     */
    protected moveDep = (srcDep: Core.ShareMgnt.ncTDepartmentInfo, targetDep: Core.ShareMgnt.ncTDepartmentInfo, ossInfo: { ossInfo: Core.ShareMgnt.ncTUsrmOSSInfo } | null): void => {
        this.changeAction(Action.None)

        this.orgTree.moveNode(srcDep, targetDep, ossInfo)
    }

    /**
     * 更新用户列表
     */
    protected updateGrid = (isBackToFirst: boolean = false): void => {
        this.changeAction(Action.None)

        if (isBackToFirst) {
            this.userGrid.backToFirst()
        } else {
            this.userGrid.updateCurrentPage(this.state.selectedUsers.length)
        }
    }

    /**
     * 启用/禁用用户
     */
    protected enableUser = (isUpdateAll: boolean, param: any): void => {
        this.changeAction(Action.None)

        if (isUpdateAll) {
            this.userGrid.updateCurrentPage()
        } else {
            this.state.selectedUsers.map(async (userInfo) => {
                await this.userGrid.updateUserInfo({ ...userInfo, user: { ...userInfo.user, ...param } }, true)
            })
        }
    }

    /**
     * 导入域用户成功
     */
    protected importDomainUser = (importStyle: ImportStyle): void => {
        if (importStyle === ImportStyle.Users) {
            this.userGrid.backToFirst()
        } else {
            this.orgTree.initOrgTree()
        }
    }

    /**
     * 选中组织或部门
     */
    protected selectDep = (selectedDep: DataNode): void => {
        this.setState({
            selectedDep,
            menus: changeActionStatusByDep(this.state.menus, selectedDep),
        })
    }

    /**
     * 选中用户
     */
    protected selectUsers = (selectedUsers: ReadonlyArray<Core.ShareMgnt.ncTUsrmGetUserInfo>): void => {
        this.setState({
            selectedUsers,
            menus: changeActionStatusBySelectedUsers(this.state.menus, selectedUsers, this.state.selectedDep),
        })
    }

    /**
     * 关闭初始化界面
     */
    protected closeInit = () => {
        this.setState({
            isShowInit: false,
        })

        this.userGrid.updateCurrentPage()
    }

    /**
     * 获取错误信息
     */
    private getErrorMsg = (ex: any): string | undefined => {
        if (ex && ex.error && ex.error.errID) {
            switch (ex.error.errID) {
                case ErrorCode.ExportProcessing:
                    return __('正在处理中，请稍候...')

                case ErrorCode.ExportNotSupport:
                    return __('该类型的文档库不支持导出功能')

                default:
                    return ex.error.errMsg
            }
        }
    }

    /**
     * 三权分立下，系统管理员没有设置组织管理员的入口
     */
    protected isAdmin = (): boolean => {
        const { user: { roles } } = session.get('isf.userInfo')

        return roles &&
            roles.some(({ id }) => id === SystemRoleType.Admin)
    }

    protected doRedirectDomain = async (): void => {
        if(this.userInfo.user.roles.some((role) => role.id === SystemRoleType.Supper || role.id === SystemRoleType.Admin)) {
            if(this.context.history && this.context.history.navigateToMicroWidget) {
                this.context.history.navigateToMicroWidget({ name: 'cert-manage', path: "?tab=domain-auth" })
                this.setState({action: Action.None})
            }
        }else {
            Toast.open(__('您暂无权限进入此页面'))
        }
    }

    protected isSuperOrAdmin = () => {
        return this.userInfo.user.roles.some((role) => role.id === SystemRoleType.Supper || role.id === SystemRoleType.Admin)
    }

    protected onChangeTab = (tabId): void => {
        const url = new URL(window.location.href);
        url.searchParams.set('tab', tabId);
        window.history.pushState({}, '', url)
        this.setState({
            urlParams: url.searchParams
        })
    }
}
