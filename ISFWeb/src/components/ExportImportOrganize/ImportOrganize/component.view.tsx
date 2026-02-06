import * as React from 'react';
import { Dialog2 as Dialog, Panel, Button, Text, UIIcon } from '@/ui/ui.desktop';
import { ErrorCode } from '@/core/thrift/sharemgnt/errcode';
import { DataGrid } from '@/sweet-ui';
import ImportOrganizeBase from './component.base';
import __ from './locale';
import styles from './styles.view';

export default class ImportOrganize extends ImportOrganizeBase {

    /*
     *错误信息处理
    */
    errorMg(record: Record<string, any>) {
        const {
            InvalidUserName, InvalidDisplayName, InvalidEmail, InvalidSecret, UserNameExist, DisplayNameExist, OSSDisabled,
            UserByAdminExist, EmailExist, InvalidPhoneNub, PhoneNubExist, InvalidCardId, InvalidRemarks, CardIdExist, InvalidDepartName,
            InvalidChangePwd, UserCreateError, DateExpired, OSSNotExist, InvalidUserStatus, InvalidPwd, LimitAssignUserSpace, CannotEditUsers,
        } = ErrorCode;

        switch (record.errorID) {
            case InvalidUserName:
                if (record.userInfo.loginName === '') {
                    return __('用户名不能为空')
                } else {
                    return __('用户名不合法，可能字符过长或包含 \\ / * ? " < > | 特殊字符')
                }

            case InvalidDisplayName:
                if (record.userInfo.displayName === '') {
                    return __('显示名不能为空')
                } else {
                    return __('显示名不合法，可能字符过长或包含 \\ / * ? " < > | 特殊字符')
                }

            case InvalidDepartName:
                if (record.userInfo.departmentNames[0] === '') {
                    return __('部门不能为空')
                } else {
                    return __('请填写正确的部门格式（详情见填写表格须知）')
                }

            case InvalidRemarks:
                return __('备注不合法，可能字符过长或包含 \\ / : * ? " < > | 特殊字符')

            case InvalidPhoneNub:
                return __('手机号码只能包含 数字，长度范围 1~20 个字符，请重新输入')

            case PhoneNubExist:
                return __('手机号已被占用')

            case InvalidEmail:
                return __('邮箱地址只能包含 英文、数字 及 @-_. 字符，格式形如 XXX@XXX，长度范围 3~100 个字符')

            case EmailExist:
                return __('邮箱地址已被占用')

            case InvalidCardId:
                return __('请输入正确的身份证号')

            case CardIdExist:
                return __('身份证号已被占用')

            case InvalidSecret:
                return __('您输入的用户密级不合法')

            case UserNameExist:
                return __('用户名已被占用')

            case DisplayNameExist:
                return __('显示名已被占用')

            case OSSDisabled:
            case OSSNotExist:
                return __('所指定的存储位置已不可用，请更换')

            case UserByAdminExist:
                return __('用户名已被管理员占用')

            case InvalidChangePwd:
                return __('不能修改已存在用户的初始密码')

            case DateExpired:
                return __('您输入的有效期不合法')

            case UserCreateError:
                return __('文件内容格式错误，用户组织列表不能新建、删除、修改')

            case InvalidUserStatus:
                return __('您输入的用户状态不合法')

            case InvalidPwd:
                return __('无效的密码')

            case LimitAssignUserSpace:
                return __('当前用户管理可分配空间已超出限制')

            case CannotEditUsers:
                return __('组织管理员不能编辑自身')

            default:
                return record.errorMessage
        }
    }

    /*
     *错误列表
    */
    getErrorTemplate() {
        const { page, defaultList, count } = this.state;
        const list = [
            {
                title: __('序号'),
                key: 'index',
                width: '10%',
                renderCell: (index, record) => (
                    <Text>
                        {index}
                    </Text>
                ),
            },
            {
                title: __('用户名'),
                key: 'username',
                width: '10%',
                renderCell: (username, record) => (
                    <Text>
                        {record.userInfo.loginName}
                    </Text>
                ),
            },
            {
                title: __('显示名'),
                key: 'showname',
                width: '10%',
                renderCell: (showname, record) => (
                    <Text>
                        {record.userInfo.displayName}
                    </Text>
                ),
            },
            {
                title: __('部门'),
                key: 'department',
                width: '10%',
                renderCell: (department, record) => (
                    <Text>
                        {record.userInfo.departmentNames[0]}
                    </Text>
                ),
            },
            {
                title: __('邮箱地址'),
                key: 'email',
                width: '10%',
                renderCell: (email, record) => (
                    <Text>
                        {record.userInfo.email}
                    </Text>
                ),
            },
            {
                title: __('手机号'),
                key: 'phoneNumer',
                width: '10%',
                renderCell: (phoneNumer, record) => (
                    <Text>
                        {record.userInfo.telNumber}
                    </Text>
                ),
            },
            {
                title: __('身份证号'),
                key: 'idNumber',
                width: '15%',
                renderCell: (idNumber, record) => (
                    <Text>
                        {record.userInfo.idcardNumber}
                    </Text>
                ),
            },
            {
                title: __('错误信息'),
                key: 'errorMessage',
                width: '25%',
                renderCell: (errorMessage, record) => (
                    <Text className={styles['error-message']}>
                        {this.errorMg(record)}
                    </Text>
                ),
            },
        ]

        return (
            <div className={styles['grid']}>
                <DataGrid
                    ref={(dataGrid) => this.dataGrid = dataGrid}
                    data={defaultList}
                    height={280}
                    enableMultiSelect={false}
                    enableSelect={false}
                    showBorder={false}
                    DataGridHeader={{ enableSelectAll: true }}
                    DataGridPager={{
                        total: count,
                        page: page + 1,
                        size: this.PageSize,
                        onPageChange: ({ detail }) => { this.handlePageChange(detail.page) },
                    }}
                    columns={list}
                />
            </div>
        )
    }

    render() {
        const { failNum, successNum, onCancel, onContinue, onImportSuccess } = this.props;

        return (
            <Dialog
                title={__('导入用户组织')}
                onClose={onCancel}
            >
                <Panel>
                    <Panel.Main>
                        {failNum === 0 ?
                            <div className={styles['import-organize-success']}>
                                <UIIcon
                                    code={'\uf063'}
                                    color="#81c884"
                                    size={35}
                                />
                                <span className={styles['import-success-number']}>
                                    {__('已成功导入${successNum}个用户', { successNum: successNum })}
                                </span>
                            </div>
                            :
                            <div className={styles['import-organize-error']}>
                                <div className={styles['import-success-number']}>
                                    {__('已成功导入${successNum}个用户', { successNum: successNum })}
                                </div>
                                <span className={styles['import-error-number']}>
                                    {__('${failNum}条记录错误，未导入成功。', { failNum: failNum })}
                                </span>
                                {__('您可以下载未导入成功的用户，修改正确后再试')}
                                <div className={styles['operation']}>
                                    <Button
                                        className={styles['download-btn']}
                                        onClick={this.downloadErrorList.bind(this)}
                                    >
                                        {__('下载导入失败的记录')}
                                    </Button>
                                    <Button
                                        className={styles['continue-btn']}
                                        onClick={onContinue}
                                        style={{ color: '#3c8eff' }}
                                    >
                                        {__('继续导入')}
                                    </Button>
                                </div>
                                {
                                    this.getErrorTemplate()
                                }
                            </div>

                        }
                    </Panel.Main>
                    {
                        failNum === 0 ?
                            <div>
                                <Panel.Footer>
                                    <Panel.Button
                                        onClick={onImportSuccess}
                                    >
                                        {__('确定')}
                                    </Panel.Button>
                                    <Panel.Button
                                        onClick={onContinue}
                                        style={{ color: '#3c8eff', border: 'none', background: '#f9f9f9' }}
                                    >
                                        {__('继续导入')}
                                    </Panel.Button>
                                </Panel.Footer>
                            </div>
                            : null
                    }
                </Panel>
            </Dialog>

        )
    }
}