import * as React from 'react'
import intl from 'react-intl-universal';
import classnames from 'classnames'
import { includes } from 'lodash';
import { Table, Checkbox } from 'antd';
import { NumberBox, Switch, Select } from '@/sweet-ui'
import { Text, SearchBox, ToolBar, Title, UIIcon } from '@/ui/ui.desktop'
import { getNameByRole, getRoleName, SystemRoleType, UserRole } from '@/core/role/role'
import { formatTime } from '@/util/formatters'
import { getUserStringType, getUserType } from '@/core/user/'
import { ListTipStatus } from '../../ListTipComponent/helper'
import SetUserExpireTime from '../../SetUserExpireTime/component.view'
import * as userClosedImg from '../assets/images/userClosed.png'
import PathTitle from './PathTitle/component.view'
import UserGridBase, { Limit } from './component.base'
import __ from './locale'
import styles from './styles.view';
import { formatCode } from '../helper';
import loadFailedIcon from '../../../icons/loadFailed.png'
import SearchEmptyIcon from '../../../icons/searchEmpty.png'
import EmptyIcon from '../../../icons/empty.png'

export default class UserGrid extends UserGridBase {
    render() {
        const {
            data = { users: [], page: 1, pageSize: Limit, total: 0 },
            listTipStatus,
            selections,
            searchKey,
            serarchField,
            isShowSetExpireTime,
        } = this.state;
        const { users, page, pageSize, total } = data;

        const searchFilter = [{ key:'name', label: __('显示名称') }, { key:'account', label: __('用户名称') }, { key:'code', label: __('用户编码') }, { key:'manager_name', label: __('直属上级') }, { key:'direct_department_code', label: __('直属部门编码') }, { key:'position', label: __('岗位') }]

        return (
            <div className={styles['grid']}>
                <div className={styles['header']}>
                    <ToolBar>
                        <div style={{ display: 'flex' }}>
                            <Select
                                maxMenuHeight={190}
                                className={styles['search-filter']}
                                value={serarchField}
                                onChange={({ detail }) => this.changeFilter(detail)}
                            >
                                {
                                    searchFilter.map(({ key, label }) => {
                                        return <Select.Option key={key} value={key}>{label}</Select.Option>
                                    })
                                }
                            </Select>
                            <SearchBox
                                role={'ui-searchbox'}
                                ref={(searchBox) => this.searchBox = searchBox}
                                className={styles['search-box']}
                                width={220}
                                placeholder={__('请输入搜索内容')}
                                value={searchKey}
                                onChange={this.changeSearchKey}
                            />
                        </div>
                    </ToolBar>
                </div>
                <div className={styles['grid-wrapper']}>
                    <div className={styles['grid-content']}>
                        <Table
                            size="small"
                            tableLayout="fixed"
                            loading={listTipStatus === ListTipStatus.Loading}
                            dataSource={users}
                            columns={this.filterVisibleColumns(this.getcolumns().map(column => ({
                                ...column,
                                onHeaderCell: (col) => ({
                                    onContextMenu: (e) => this.handleContextMenu(e, col),
                                }),
                            })))}
                            rowSelection={{
                                selectedRowKeys: selections.map(user => user.id),
                                onChange: (selectedRowKeys: React.Key[], selectedRows: Core.ShareMgnt.ncTUsrmGetUserInfo[]) => {
                                    this.changeSelection(selectedRows);
                                },
                                columnWidth: 40, 
                                fixed: 'left',
                            }}
                            rowKey={(record) => record.id}
                            onRow={(record) => ({
                                onClick: () => {
                                    this.changeSelection([record]);
                                },
                            })}
                            scroll={{ x: 'max-content', y: 'calc(100vh - 330px)' }}
                            pagination={{
                                current: page,
                                pageSize,
                                total: total,
                                showSizeChanger: true,
                                showQuickJumper: false,
                                showTotal: (total) => {
                                    return intl.get("list.total.tip", { total });
                                },
                                onChange: (curPage, curPageSize) =>{
                                    if(curPage === page && curPageSize === pageSize) return 
                                    const newPage = pageSize !== curPageSize ? 1 : curPage
                                    const offset = ((pageSize !== curPageSize ? 1 : curPage) - 1) * curPageSize
                                    this.handlePageChange(newPage, curPageSize, offset)
                                }, 
                            }}
                            locale={{
                                emptyText: (
                                    <div className={styles["empty"]}>
                                        <img
                                            src={listTipStatus === ListTipStatus.LoadFailed ? loadFailedIcon : this.state.searchKey ? SearchEmptyIcon : EmptyIcon}
                                            alt=""
                                            width={128}
                                            height={128}
                                        />
                                        <span>{intl.get(listTipStatus === ListTipStatus.LoadFailed ? "loadFailed" : this.state.searchKey ? "no.search.result" : "no.user")}</span>
                                    </div>
                                ),
                            }}
                        />
                        {this.state.contextMenuVisible && (
                            <div
                                className={styles['table-context-menu']}
                                style={{
                                    left: this.state.contextMenuPosition.x,
                                    top: this.state.contextMenuPosition.y,
                                }}
                                onClick={(e) => e.stopPropagation()}
                            >
                                <div
                                   className={styles['context-list-title']}
                                >
                                    <Text>{__('筛选列')}</Text>
                                </div>
                                <div className={styles['context-list-all']}>
                                    <div
                                        className={styles['all-checkbox']}
                                    >
                                        <Checkbox
                                            checked={this.isAllColumnsSelected(this.getcolumns())}
                                            onChange={() => {
                                                const columns = this.getcolumns();
                                                const isAllSelected = this.isAllColumnsSelected(columns);
                                                if (isAllSelected) {
                                                    this.unselectAllColumns(columns);
                                                } else {
                                                    this.selectAllColumns(columns);
                                                }
                                            }}
                                            style={{ marginRight: '12px', verticalAlign: 'middle', cursor: 'pointer' }}
                                        >
                                            <span title={__('全选')}>{__('全选')}</span>
                                        </Checkbox>
                                    </div>
                                    <div className={styles['context-list-divider']}></div>
                                    <div className={styles['column-title-list']}>
                                        {this.getcolumns().map((col) => {
                                            const isNonHideable = col.key && this.nonHideableColumns.includes(col.key);
                                            
                                            return (
                                                <div
                                                    key={col.key || 'default'}
                                                    className={styles['column-title-item']}
                                                    style={{ opacity: isNonHideable ? 0.6 : 1 }}
                                                >
                                                    <Checkbox
                                                        checked={isNonHideable || this.state.columnVisibility[col.key] !== false}
                                                        disabled={isNonHideable}
                                                        onChange={() => {
                                                            if (col.key && !isNonHideable) {
                                                                this.toggleColumnVisibility(col.key);
                                                            }
                                                        }}
                                                        style={{ marginRight: '12px', verticalAlign: 'middle' }}
                                                    >
                                                        <span className={styles['column-title']} title={col.title}>{col.title}</span>
                                                    </Checkbox>
                                                </div>
                                            );
                                        })}
                                    </div>
                                </div>
                            </div>
                        )}
                    </div>
                </div>
                {
                    isShowSetExpireTime ?
                        <SetUserExpireTime
                            dep={
                                this.expiredUserInfo.directDeptInfo ?
                                    {
                                        name: this.expiredUserInfo.directDeptInfo.departmentName,
                                        id: this.expiredUserInfo.directDeptInfo.departmentId,
                                    }
                                    : {}
                            }
                            users={[this.expiredUserInfo]}
                            shouldEnableUsers={true}
                            onCancel={() => this.setState({ isShowSetExpireTime: false })}
                            onSuccess={() => { this.setState({ isShowSetExpireTime: false }); this.updateCurrentPage() }}
                        />
                        : null
                }
            </div>
        )
    }

    /**
     * 获取列表项
     */
    protected getcolumns = () => {
        const {
            selectedDep,
            isShowSetRole,
            isShowEnableAndDisableUser,
            freezeStatus,
            isKjzDisabled,
        } = this.props

        const allColumns = [
            {
                title: __('显示名称'),
                key: 'displayName',
                fixed: 'left',
                className: styles['column'],
                render: (displayName, record) => (
                    <div className={classnames(styles['user-displayName'], { [styles['gray-text']]: record.user.status })}>
                        {this.getUserIcon(record)}
                        <Text className={classnames(styles['displayName-text'], styles['ellipsis'])} role={'ui-text'} title={record.user.displayName}>
                            {record.user.displayName}
                        </Text>
                    </div>
                ),
            },
            {
                title: __('用户名称'),
                key: 'loginName',
                className: styles['column'],
                render: (loginName, record) => (
                    <Text className={classnames(styles['column-item-text'], { [styles['gray-text']]: record.user.status })} role={'ui-text'}>
                        {record.user.loginName}
                    </Text>
                ),
            },
            {
                title: __('用户编码'),
                key: 'code',
                className: styles['column'],
                render: (code, record) => (
                    <Text className={classnames(styles['column-item-text'], { [styles['gray-text']]: record.user.status })} role={'ui-text'}>
                        {record.user.code || '---'}
                    </Text>
                ),
            },
            {
                title: __('直属上级'),
                key: 'directMangager',
                className: styles['column-middle'],
                render: (managerDisplayName, record) => (
                    <Text className={classnames(styles['column-item-text'], { [styles['gray-text']]: record.user.status })} role={'ui-text'}>
                        {record.user.managerDisplayName || '---'}
                    </Text>
                ),
            },
            {
                title: __('直属部门编码'),
                key: 'departmentCodes',
                className: styles['column-large'],
                render: (departmentCodes, record) => (
                    <Text className={classnames(styles['column-item-text'], { [styles['gray-text']]: record.user.status })} role={'ui-text'}>
                        {record.user.departmentCodes && formatCode(record.user.departmentCodes).length ? formatCode(record.user.departmentCodes).join(',') : '---'}
                    </Text>
                ),
            },
            {
                title: __('岗位'),
                key: 'position',
                className: styles['column'],
                render: (position, record) => (
                    <Text className={classnames(styles['column-item-text'], { [styles['gray-text']]: record.user.status })} role={'ui-text'}>
                        {record.user.position || '---'}
                    </Text>
                ),
            },
            {
                title: __('备注'),
                key: 'remark',
                className: styles['column'],
                render: (remark, record) => (
                    <Text className={classnames(styles['column-item-text'], { [styles['gray-text']]: record.user.status })} role={'ui-text'}>
                        {record.user.remark || '---'}
                    </Text>
                ),
            },
            {
                title: __('直属部门'),
                key: 'directDeptInfo',
                className: styles['column'],
                render: (directDeptInfo, record) => (
                    <PathTitle
                        selectedDep={selectedDep}
                        record={record}
                        onRequestUpdatePath={(depPath) => this.updateUserInfo({ ...record, depPath })}
                    />
                ),
            },
            ...(
                isShowSetRole ?
                    [{
                        title: __('系统角色'),
                        key: 'roles',
                        className: styles['column'],
                        render: (roles, record) => (
                            <Text className={classnames(styles['column-item-text'], { [styles['gray-text']]: record.user.status })} role={'ui-text'}>
                                {
                                    this.state.searchKey ?  [...this.getAllRoleNameByRole(record.user.roles)].join('/') : [...this.getAllRoleName(record.user.roles)].join('/') || '---'
                                }
                            </Text>
                        ),
                    }] : []
            ),
            {
                title: __('用户密级'),
                key: 'csfLevel',
                className: styles['column'],
                render: (csfLevel, record) => {
                    const csfLevel1 = this.csfLevels?.csf_level_enum?.find((item) => item.value === record.user.csfLevel)?.name || '---'
                    const csfLevel2 = this.csfLevels?.csf_level2_enum?.find((item) => item.value === record.user.csfLevel2)?.name || '---'

                    let csfLevelStr = []
                    let titleStr = []
                    csfLevelStr.push(csfLevel1)
                    titleStr.push(this.csfLevels?.show_csf_level2 ? `${__('用户密级：')}${csfLevel1}` : csfLevel1)
                    if(this.csfLevels?.show_csf_level2) {
                        csfLevelStr.push(csfLevel2)
                        titleStr.push(`${__('用户密级2：')}${csfLevel2}`)
                    }
                    return (
                        <div title={titleStr.join('\n')} className={classnames(styles['column-item-text'], { [styles['gray-text']]: record.user.status })}>
                            {csfLevelStr.join('，')}
                        </div>
                )
                },
            },
            {
                title: __('认证类型'),
                key: 'userType',
                className: styles['column'],
                render: (userType, record) => (
                    <Text className={classnames(styles['column-item-text'], { [styles['gray-text']]: record.user.status })} role={'ui-text'}>
                        {typeof(record.user.userType) === 'number' ? getUserType(record.user.userType)  : getUserStringType(record.user.userType)}
                    </Text>
                ),
            },
            ,
            {
                title: __('产品授权'),
                key: 'productLicense',
                className: styles['column'],
                render: (productLicense, record) => (
                    <Text 
                        role={'ui-text'}
                        className={classnames(styles['column-item-text'], { [styles['gray-text']]: record.user.status || !record.user.products?.length})}
                     >
                       {record.user.products?.join('，') || __('未授权')}
                    </Text>
                ),
            },
            ...(
                isKjzDisabled ?
                    [{
                        title: __('权重排序'),
                        key: 'priority',
                        className: styles['column'],
                        render: (priority, record) => (
                            <NumberBox
                                role={'sweetui-numberbox'}
                                className={classnames(styles['column-item-text'], { [styles['gray-text']]: record.user.status })}
                                width={50}
                                precision={0}
                                defaultNumber={record.user.priority}
                                onBlur={(e, value) => this.setPriority(record, value)}
                                {...(this.state.invalidPriorityID === record.id ? { value: 999 } : {})}
                            />
                        ),
                    }]
                    : []
            ),
            {
                title: __('创建时间'),
                key: 'createTime',
                className: styles['column'],
                render: (user, record) => {
                    const { user: { createTime } } = record

                    return (
                        <Text role={'sweetui-text'} className={classnames(styles['column-item-text'])}>
                            {formatTime(typeof(createTime) === 'number' ? createTime * 1000 : createTime, 'yyyy-MM-dd HH:mm:ss')}
                        </Text>
                    )
                },
            },
            ...(
                (isShowEnableAndDisableUser && isKjzDisabled) ?
                    [{            
                        title: __('状态'),
                        key: 'status',
                        className: styles['column'],
                        render: (status, record) => (
                            <Title content={__(`点此${!record.user.status ? '禁用' : '开启'}用户`)} role={'sweetui-title'} className={classnames(styles['column-item-text'])}>
                                <Switch
                                    role={'sweetui-switch'}
                                    checked={!record.user.status}
                                    onChange={({ detail }) => { this.changeStatus(record, detail) }}
                                />
                            </Title>
                        ),
                    }]
                    : []
            ),
            ...(
                (freezeStatus && isKjzDisabled) ?
                    [{            
                        title: __('冻结状态'),
                        key: 'freezeStatus',
                        className: styles['column'],
                        render: (freezeStatus, record) => (
                            <Title content={__(`点此${record.user.freezeStatus ? '解冻' : '冻结'}用户`)} role={'sweetui-title'} className={classnames(styles['column-item-text'])}>
                                <Switch
                                    role={'sweetui-switch'}
                                    checked={record.user.freezeStatus}
                                    onChange={({ detail }) => { this.changeFreezeStatus(record, detail) }}
                                />
                            </Title>
                        ),
                    }]
                    : []
            ),
        ]

        return allColumns
    }

    /**
     * 获取所有角色名称
     */
    private getAllRoleName = (roles: ReadonlyArray<any>): ReadonlyArray<string> => {
        // 升级时，去除原有数据数组中包含的共享、定密、文档审核员
        const newRoles = roles.filter((role) => {
            return !includes([
                SystemRoleType.SharedApprove,
                SystemRoleType.CsfApprove,
                SystemRoleType.DocApprove,
            ], role.id)
        })
        const role = !newRoles.includes(UserRole.NormalUser) ? newRoles.map(getRoleName) : []
        return role
    }

    private getAllRoleNameByRole = (roles: UserRole[]): ReadonlyArray<string> => {
        return roles.map(getNameByRole)
    }

    /**
     * 获取用户图标
     */
    private getUserIcon(userInfo) {
        const { user: { status } } = userInfo

        return (
            <Title role={'ui-title'}>
                <UIIcon
                    className={classnames(styles['user-icon'], { [styles['icon-gray']]: status })}
                    role={'ui-uiicon'}
                    fallback={userClosedImg}
                    code={'\u0000'}
                    size={16}
                />
            </Title>
        )
    }
}