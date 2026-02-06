import { noop, values } from 'lodash';
import { getRoleName } from '@/core/role/role';
import { NodeType } from '@/core/organization';
import WebComponent from '../../../webcomponent';
import __ from './locale';

export enum ValidateState {
    Normal,
    Empty,
    InvalidSpace,
}

export const ValidateMessages = {
    [ValidateState.Empty]: __('此项不允许为空。')
}

export default class SetOrgManagerBase extends WebComponent<Console.SetOrgManager.Props, Console.SetOrgManager.State> {
    static defaultProps = {
        userid: '',
        editRateInfo: null,
        roleInfo: null,
        userInfo: null,
        directDeptInfo: null,
        onConfirmSetRoleConfig: noop,
        onCancelSetRoleConfig: noop,
    }

    state = {
        validateState: {},
        selectDeps: [],
        selectState: false,
    }

    componentDidMount() {
        const { directDeptInfo } = this.props;

        // 如果是编辑，优先显示被编辑管理员信息
        if (this.props.editRateInfo) {
            const { editRateInfo: { manageDeptInfo } } = this.props;
            if (manageDeptInfo) {
                const { editRateInfo: { manageDeptInfo } } = this.props;

                this.setState({
                    selectDeps: manageDeptInfo.departmentIds.length ? manageDeptInfo.departmentIds.map((cur, index) => (
                        {
                            objectId: cur,
                            objectName: manageDeptInfo.departmentNames[index],
                            objType: NodeType.DEPARTMENT,
                        }
                    )) : []
                })
            }
        } else {
            if (directDeptInfo && directDeptInfo.departmentId !== '-1') {
                this.setState({
                    selectDeps: [
                        {
                            objectId: directDeptInfo.departmentId,
                            objectName: directDeptInfo.departmentName,
                            objType: NodeType.DEPARTMENT,
                        },
                    ],
                })
            }
        }
    }

    /**
     * 转入前先转换数据格式
     * @param data
     */
    protected convertData = (data) => {
        return {
            id: data.objectId,
            name: data.objectName,
            type: data.objType,
        }

    }

    /**
     * 转出数据时转换数据格式
     */
    protected convertDataOut(data) {
        return {
            objectId: data.id,
            objectName: data.name || data.displayName || data.departmentName || (data.user && data.user.displayName),
            objType: data.type,
        }
    }

    /**
     * 选择部门
     */
    protected selectDeparment(data) {
        this.setState({
            selectDeps: data,
        }, () => {
            if (this.state.selectDeps.length) {
                this.setState({
                    selectState: false,
                })
            }
        })
    }

    /**
     * 验证输入值以及是否选择部门
     */
    protected validateRole = () => {
        const { validateState,  selectDeps } = this.state;
        this.setState({
            validateState: {
                ...validateState,
            },
            selectState: !selectDeps.length ? true : false,
        }, () => {
            if (!(values(this.state.validateState).some((state) => state !== ValidateState.Normal)) && !this.state.selectState) {
                this.confirmRoleRateConfig()
            }
        })
    }

    /**
     * 将数据传出去
     */
    private confirmRoleRateConfig() {
        if (this.state.selectDeps.length) {
            let depInfo = this.state.selectDeps.reduce((pre, cur) => (
                {
                    depIds: [...pre.depIds, cur.objectId],
                    depNames: [...pre.depNames, cur.objectName],
                }
            ), { depIds: [], depNames: [] })
            let manageRange = {
                ncTManageDeptInfo: {
                    departmentIds: depInfo.depIds,
                    departmentNames: depInfo.depNames,
                },
            }
            this.props.onConfirmSetRoleConfig({
                name: getRoleName(this.props.roleInfo),
                id: this.props.roleInfo.id,
                manageRange,
            })
        }
    }

    /**
     * 取消本次操作
     */
    protected cancelSetRoleConfig = () => {
        this.setState({
            selectDeps: [],
        }, () => {
            this.props.onCancelSetRoleConfig();
        })
    }
}