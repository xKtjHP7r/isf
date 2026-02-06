import * as React from 'react';
import __ from './locale';
import { SystemRoleType } from '@/core/role/role';
import { getRoleName } from '@/core/role/role';
import { Text } from '@/ui/ui.desktop';
import AutoComplete from '@/ui/AutoComplete/ui.desktop';
import AutoCompleteList from '@/ui/AutoCompleteList/ui.desktop';
import InlineButton from '@/ui/InlineButton/ui.desktop';
import { DataGrid } from '@/sweet-ui';
import SetRoleComponentBase, { EditableRoles, LoginConsoleRoles } from './component.base';
import Dialog from '@/ui/Dialog2/ui.desktop';
import Panel from '@/ui/Panel/ui.desktop';
import SetOrgManager from './SetRole.OrgManager/component.view';
import SetOrgAudit from './SetRole.OrgAudit/component.view';
import styles from './styles.desktop';

export default class SetRoleComponent extends SetRoleComponentBase {
    render() {
        const {
            value,
            results,
            userInfo,
            ownRole,
            selectRoleInfo,
            showRoleEditDialog,
            allRoles,
            roles,
        } = this.state;

        return (
            <div>
                <Dialog
                    role={'ui-dialog'}
                    width={'600px'}
                    title={__('设置系统角色')}
                    onClose={this.props.onComplete}
                >
                    <Panel role={'ui-panel'}>
                        <Panel.Main role={'ui-panel.main'}>
                            <div className={styles['layout']}>
                                <div className={styles['setRole-head']}>
                                    <div className={styles['setRole-tit']}>
                                        <span>{__('为用户 “')}</span>
                                        <span className={styles['user-name']}><Text role={'ui-text'}>{userInfo.user.displayName}</Text></span>
                                        <span>{__('” 添加角色：')}</span>
                                    </div>
                                    <div className={styles['setRole-search']}>
                                        <AutoComplete
                                            role={'ui-autocomplete'}
                                            ref="autocomplete"
                                            width={285}
                                            icon=""
                                            value={value}
                                            onChange={this.handleChange.bind(this)}
                                            onFocus={this.handleFocus.bind(this)}
                                            loader={this.searchRoles.bind(this)}
                                            onLoad={(data) => { this.getSearchData(data) }}
                                            onEnter={this.handleEnter.bind(this)}
                                            placeholder={__('点击选择或输入系统角色名称搜索')}
                                            missingMessage={__('未找到匹配的结果')}
                                        >
                                            {
                                                results && results.length ?
                                                    <AutoCompleteList role={'ui-autocompletelist'}>
                                                        {
                                                            results.map((value) => (
                                                                <AutoCompleteList.Item
                                                                    role={'ui-autocompletelist.item'}
                                                                    key={value.id}>
                                                                    <span className={styles['search-item']} onClick={() => { this.handleChooseRole(value) }}>
                                                                        <span className={styles['seleted-data']}>
                                                                            <Text role={'ui-text'} fontSize={13}>{getRoleName(value)}</Text>
                                                                        </span>
                                                                    </span>
                                                                </AutoCompleteList.Item>
                                                            ))
                                                        }
                                                    </AutoCompleteList>
                                                    : null
                                            }
                                        </AutoComplete>
                                    </div>
                                </div>
                                <div className={styles['list']}>
                                    <DataGrid
                                        role={'sweetui-datagrid'}
                                        height={400}
                                        data={ownRole}
                                        headless={true}
                                        columns={[
                                            {
                                                key: 'name',
                                                width: '70%',
                                                renderCell: (name, record) => (
                                                    <Text
                                                        role={'ui-text'}
                                                        className={styles['text']}
                                                    >
                                                        {getRoleName(record)}
                                                    </Text>
                                                ),
                                            },
                                            {
                                                key: 'eidt',
                                                width: '5%',
                                                renderCell: (eidt, record) => (
                                                    EditableRoles.includes(record.id) &&
                                                        allRoles.some((role) => role.id === record.id) &&
                                                        (ownRole.some((role) =>
                                                            role.id === SystemRoleType.Supper) ||
                                                            this.props.userid !== userInfo.id ||
                                                            !LoginConsoleRoles.includes(record.id)
                                                        ) ?
                                                        <InlineButton
                                                            role={'ui-inlinebutton'}
                                                            className={styles['handle-icon']}
                                                            title={__('编辑')}
                                                            size={24}
                                                            iconSize={16}
                                                            code={'\uf05c'}
                                                            onClick={() => this.editRoleInfo(record)}
                                                        />
                                                        : null
                                                ),
                                            },
                                            {
                                                key: 'del',
                                                width: '7%',
                                                renderCell: (del, record) => (
                                                    allRoles.some((role) => role.id === record.id) &&
                                                        (ownRole.some((role) =>
                                                            role.id === SystemRoleType.Supper) ||
                                                            this.props.userid !== userInfo.id ||
                                                            !LoginConsoleRoles.includes(record.id)
                                                        ) ?
                                                        <InlineButton
                                                            role={'ui-inlinebutton'}
                                                            className={styles['handle-icon']}
                                                            title={__('移除')}
                                                            size={24}
                                                            iconSize={16}
                                                            code={'\uf046'}
                                                            onClick={() => this.deleteRoleInfo(record)}
                                                        />
                                                        : null
                                                ),
                                            },
                                        ]}
                                    />
                                </div>
                            </div>
                        </Panel.Main>
                    </Panel>
                </Dialog>
                {
                    showRoleEditDialog ?
                        selectRoleInfo.id === SystemRoleType.OrgManager ?
                            <SetOrgManager
                                onConfirmSetRoleConfig={this.handleConfirmSetRoleConfig}
                                onCancelSetRoleConfig={this.handleCancelSetRoleConfig}
                                editRateInfo={this.roleRateInEdit}
                                roleInfo={selectRoleInfo}
                                userInfo={userInfo}
                                userid={this.props.userid}
                                directDeptInfo={
                                    userInfo.directDeptInfo
                                        ? userInfo.directDeptInfo
                                        : (userInfo.user && Array.isArray(userInfo.user.departmentIds))
                                            ? { departmentId: userInfo.user.departmentIds[0], departmentName: userInfo.user.departmentNames[0] }
                                            : null
                                }
                                roles={roles}
                            /> :
                            selectRoleInfo.id === SystemRoleType.OrgAudit ?
                                <SetOrgAudit
                                    onConfirmSetRoleConfig={this.handleConfirmSetRoleConfig}
                                    onCancelSetRoleConfig={this.handleCancelSetRoleConfig}
                                    editRateInfo={this.roleRateInEdit}
                                    roleInfo={selectRoleInfo}
                                    userInfo={userInfo}
                                    userid={this.props.userid}
                                    directDeptInfo={
                                        userInfo.directDeptInfo
                                            ? userInfo.directDeptInfo
                                            : (userInfo.user && Array.isArray(userInfo.user.departmentIds))
                                                ? { departmentId: userInfo.user.departmentIds[0], departmentName: userInfo.user.departmentNames[0] }
                                                : null
                                    }
                                /> : null
                        : null
                }
            </div>
        )
    }
}