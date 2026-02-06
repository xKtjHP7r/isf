import { noop } from 'lodash';
import { NodeType } from '@/core/organization';
import { getRoleName } from '@/core/role/role';
import WebComponent from '../../../webcomponent';

export default class SetOrgAuditBase extends WebComponent<Console.SetOrgAudit.Props, Console.SetOrgAudit.State> {
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
        selectDeps: [],
    }

    componentDidMount() {
        const { directDeptInfo } = this.props;
        if (this.props.editRateInfo) {
            const { editRateInfo: { manageDeptInfo } } = this.props;
            if (manageDeptInfo) {
                this.setState({
                    selectDeps: manageDeptInfo.departmentIds.length ? manageDeptInfo.departmentIds.map((cur, index) => (
                        {
                            objectId: cur,
                            objectName: manageDeptInfo.departmentNames[index],
                            objType: 2,
                        }
                    )) : [],
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
    protected convertDataOut = (data) => {
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
        })
    }

    /**
     * 将数据传出去
     */
    protected confirmSetRoleConfig() {
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