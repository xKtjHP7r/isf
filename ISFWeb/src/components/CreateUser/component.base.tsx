import * as React from 'react'
import { noop, trim } from 'lodash';
import { mailAndLenth, phoneNum, isUserLoginName, isNormalName, variousIdCard,  isNormalCode, isNormalPosition, isUserNormalName } from '@/util/validators';
import { formatTime } from '@/util/formatters';
import { Message2 } from '@/sweet-ui';
import session from '@/util/session';
import { SystemRoleType } from '@/core/role/role';
import { manageLog, Level, ManagementOps } from '@/core/log';
import { getErrorMessage } from '@/core/exception';
import { ValidateState, Dep, Type, UserInfoType } from '@/core/user';
import { ErrorCode } from '@/core/thrift/sharemgnt/errcode';
import { addUser, getUserInfo, getTriSystemStatus } from '@/core/thrift/sharemgnt';
import { getLevelConfig } from '@/core/apis/console/usermanagement';
import WebComponent from '../webcomponent';
import __ from './locale';

/**
 * 错误提示索引
 */
interface Validate {
    /**
     * 用户名错误提示
     */
    loginName: ValidateState;

    /**
     * 显示名错误提示
     */
    displayName: ValidateState;

    /**
     * 用户编码
     */
    code: ValidateState;

    /**
     * 岗位
     */
    position: ValidateState;

    /**
     * 备注错误提示
     */
    remark: ValidateState;

    /**
     * 手机号错误提示
     */
    telNumber: ValidateState;

    /**
     * 身份证号错误提示
     */
    idcardNumber: ValidateState;

    /**
     * 邮箱错误提示
     */
    email: ValidateState;
}

interface CreateUserProps extends React.Props<void> {
    /**
     * 选择的部门
     */
    dep: Dep;

    /**
     * 当前登录的用户
     */
    userid: string;

    /**
     * 取消新建用户
     */
    onRequestCancel: () => any;

    /**
     * 新建用户成功
     */
    onRequestCreateUserSuccess: (createData: Core.ShareMgnt.ncTUsrmUserInfo) => void;

    /**
     * 直属部门不存在，移除直属部门
     */
    onRequestRemoveDep: (dep: Core.ShareMgnt.ncTDepartmentInfo) => any;
}

interface CreateUserState {
    /**
     * 新建用户信息
     */
    userInfo: Core.ShareMgnt.ncTUsrmUserInfo;

    /**
     * 用户密级
     */
    csfOptions: Array<{name: string, value: number}>;

    /**
     * 用户密级2
     */
    csfOptions2: Array<{name: string, value: number}>;

    /**
     * 直属上级
     */
    managerInfo: UserInfoType[];

    /**
     * 错误提示信息索引
     */
    validateState: Validate;

    /**
     * 是否显示新建弹框
     */
    showAddDialog: boolean;

    /**
     * 是否显示选择直属上级弹框
     */
    showAddDirectSupervisorDialog: boolean;
}

export default class CreateUserBase extends WebComponent<CreateUserProps, CreateUserState> {
    static defaultProps: CreateUserProps = {
        dep: null,
        userid: '',
        onRequestCancel: noop,
        onRequestCreateUserSuccess: noop,
        onRequestRemoveDep: noop,
    }

    state: CreateUserState = {
        userInfo: {
            csfLevel: null,
            csfLevel2: null,
            show_csf_level2: false,
            loginName: '',
            displayName: '',
            code: '',
            position: '',
            remark: '',
            departments: '',
            email: '',
            telNumber: '',
            idcardNumber: '',
            expireTime: -1,
        },
        validateState: {
            loginName: ValidateState.Normal,
            displayName: ValidateState.Normal,
            code: ValidateState.Normal,
            position: ValidateState.Normal,
            remark: ValidateState.Normal,
            email: ValidateState.Normal,
            telNumber: ValidateState.Normal,
            idcardNumber: ValidateState.Normal,
        },
        csfOptions: [],
        csfOptions2: [],
        managerInfo: [],
        showAddDialog: true,
        showAddDirectSupervisorDialog: false,
    }

    triSystemStatus: boolean // 是否开启三权分立
    isAdmin: boolean // 是否系统管理员
    isRequest: boolean // 是否在请求中

    async componentDidMount() {
        try {
            this.triSystemStatus = await getTriSystemStatus();
            this.isAdmin = session.get('isf.userInfo').user.roles.some((role) => [SystemRoleType.Admin].includes(role.id))
            const { userInfo, validateState } = this.state;
            this.setState({
                userInfo: {
                    ...userInfo,
                },
                validateState: {
                    ...validateState,
                },
            })
            this.getUserCsfInfos()
        } catch ({ error }) {
            if (error) {
                await Message2.info({ message: error.errMsg })
            }
        }
    }

    /**
     * 获取用户密级枚举
     */
    private async getUserCsfInfos(): Promise<void> {
        const {csf_level_enum, csf_level2_enum, show_csf_level2} = await getLevelConfig({fields: 'csf_level_enum,csf_level2_enum,show_csf_level2'})
        this.setState({
            userInfo: {
                ...this.state.userInfo,
                csfLevel: csf_level_enum?.[0]?.value,
                csfLevel2: csf_level2_enum?.[0]?.value,
                show_csf_level2,
            },
            csfOptions: csf_level_enum,
            csfOptions2: csf_level2_enum,
        })
    }

    /**
     * 失焦事件
     */
    protected handleOnBlur(type: Type): void {
        const { userInfo: { loginName, displayName, code, position, remark, telNumber, idcardNumber, email }, validateState } = this.state
        const validateLoginName = isUserLoginName(loginName);
        const validateDisplayName = isUserNormalName(trim(displayName));
        const validateCode = isNormalCode(trim(code));
        const validatePosition = isNormalPosition(trim(position))
        const validateRemark = isNormalName(trim(remark));
        const validateTelNum = phoneNum(telNumber);
        const validateIdcardNum = variousIdCard(idcardNumber);
        const validateEmail = mailAndLenth(email, 4, 101);

        switch (type) {
            case Type.LoginName:
                this.setState({
                    validateState: {
                        ...validateState,
                        loginName: validateLoginName ? ValidateState.Normal : loginName ? ValidateState.NameInvalid : ValidateState.Empty,
                    },
                })
                break;

            case Type.Email:
                this.setState({
                    validateState: {
                        ...validateState,
                        email: validateEmail ? ValidateState.Normal : email ? ValidateState.EamilInvalid : ValidateState.Normal,
                    },
                })
                break;

            case Type.DisplayName:
                this.setState({
                    validateState: {
                        ...validateState,
                        displayName: validateDisplayName ? ValidateState.Normal : displayName ? ValidateState.DisplayNameInvalid : ValidateState.Empty,
                    },
                })
                break;

            case Type.Code:
                this.setState({
                    validateState: {
                        ...validateState,
                        code: validateCode ? ValidateState.Normal : code ? ValidateState.CodeInvalid : ValidateState.Normal,
                    },
                })
                break;

            case Type.Position:
                this.setState({
                    validateState: {
                        ...validateState,
                        position: validatePosition ? ValidateState.Normal : position ? ValidateState.PositionInvalid : ValidateState.Normal,
                    },
                })
                break;

            case Type.Remark:
                this.setState({
                    validateState: {
                        ...validateState,
                        remark: validateRemark ? ValidateState.Normal : remark ? ValidateState.RemarksInvalid : ValidateState.Normal,
                    },
                })
                break;

            case Type.IdcardNumber:
                this.setState({
                    validateState: {
                        ...validateState,
                        idcardNumber: validateIdcardNum ? ValidateState.Normal : idcardNumber ? ValidateState.IdCardInvalid : ValidateState.Normal,
                    },
                })
                break;

            case Type.TelNumber:
                this.setState({
                    validateState: {
                        ...validateState,
                        telNumber: validateTelNum ? ValidateState.Normal : telNumber ? ValidateState.PhoneInvalid : ValidateState.Normal,
                    },
                })
                break;
        }
    }

    /**
    * 检查表单合法性
    */
    private checkForm = (): boolean => {
        const { loginName, displayName, code, position, remark, telNumber, idcardNumber, email } = this.state.userInfo;
        const validateLoginName = isUserLoginName(loginName);
        const validateDisplayName = isUserNormalName(trim(displayName));
        const validateCode = isNormalCode(trim(code));
        const validatePosition = isNormalPosition(trim(position))
        const validateRemark = isNormalName(trim(remark));
        const validateTelNum = phoneNum(telNumber);
        const validateIdcardNum = variousIdCard(idcardNumber);
        const validateEmail = mailAndLenth(email, 4, 101);

        if (validateLoginName && validateDisplayName && (validateEmail || !email) && (validateCode || !code) && (validatePosition || !position) && (validateRemark || !remark) && (validateTelNum || !telNumber) && (validateIdcardNum || !idcardNumber)) {
            return true;
        } else {
            this.setState({
                validateState: {
                    ...this.state.validateState,
                    loginName: validateLoginName ? ValidateState.Normal : loginName ? ValidateState.NameInvalid : ValidateState.Empty,
                    email: validateEmail ? ValidateState.Normal : email ? ValidateState.EamilInvalid : ValidateState.Normal,
                    displayName: validateDisplayName ? ValidateState.Normal : displayName ? ValidateState.DisplayNameInvalid : ValidateState.Empty,
                    code: validateCode ? ValidateState.Normal : code ? ValidateState.CodeInvalid : ValidateState.Normal,
                    position: validatePosition ? ValidateState.Normal : position ? ValidateState.PositionInvalid : ValidateState.Normal,
                    remark: validateRemark ? ValidateState.Normal : remark ? ValidateState.RemarksInvalid : ValidateState.Normal,
                    telNumber: validateTelNum ? ValidateState.Normal : telNumber ? ValidateState.PhoneInvalid : ValidateState.Normal,
                    idcardNumber: validateIdcardNum ? ValidateState.Normal : idcardNumber ? ValidateState.IdCardInvalid : ValidateState.Normal,
                },
            })

            return false;
        }
    }

    /**
     * 设置文本框变更事件
     */
    protected handleValueChange(inputInfo: { loginName?: string; displayName?: string; code?: string; position?: string; remark?: string; email?: string; telNumber?: string; idcardNumber?: string }) {
        const { userInfo, validateState } = this.state;
        const { loginName, displayName, code, position, remark, email, telNumber, idcardNumber } = inputInfo;

        this.setState({
            userInfo: { ...userInfo, ...inputInfo },
            validateState: {
                ...validateState,
                loginName: loginName ? ValidateState.Normal : validateState.loginName,
                displayName: displayName ? ValidateState.Normal : validateState.displayName,
                code: code ? ValidateState.Normal : validateState.code,
                position: position ? ValidateState.Normal : validateState.position,
                remark: remark ? ValidateState.Normal : validateState.remark,
                telNumber: telNumber ? ValidateState.Normal : validateState.telNumber,
                idcardNumber: idcardNumber ? ValidateState.Normal : validateState.idcardNumber,
                email: email ? ValidateState.Normal : validateState.email,
            },
        })
    }

    /**
     * 保存新建用户
     */
    protected createUser = async () => {
        if (this.checkForm()) {
            const { managerInfo, userInfo: { loginName, displayName, code, position, remark, email, telNumber, idcardNumber, expireTime, csfLevel, csfLevel2 }, csfOptions, csfOptions2, validateState } = this.state;
            const { userid, dep, dep: { name } } = this.props;
            let { id } = dep;

            if (id === '-2') {
                id = -1;
            }

            const data = {
                ncTUsrmAddUserInfo: {
                    user: {
                        ncTUsrmUserInfo: {
                            loginName,
                            displayName: trim(displayName),
                            code: trim(code),
                            position: trim(position),
                            managerID: managerInfo.length ? managerInfo[0].id : null,
                            managerDisplayName: managerInfo.length ? managerInfo[0].name : null,
                            remark: trim(remark),
                            email,
                            telNumber,
                            idcardNumber,
                            departmentIds: [String(id)],
                            priority: 999,
                            csfLevel: csfLevel,
                            csfLevel2: csfLevel2,
                            pwdControl: false,
                            expireTime: expireTime === -1 ? -1 : expireTime / 1000 / 1000,
                        },
                    },
                },
            }
            if (!this.isRequest) {
                try {
                    this.isRequest = true;
                    const res = await addUser([data, userid]);
                    const userInfo = await getUserInfo([res]);
                    const createData = Object.assign(userInfo, {
                        directDeptInfo: {
                            departmentId: id,
                            departmentName: name,
                        },
                    });
                    const filteredCsfOptions = csfOptions.filter(({
                        value,
                    }) => value === csfLevel);
                    const csfLevelText = filteredCsfOptions.length > 0 ? filteredCsfOptions[0]?.name : '';
                    const filteredCsfOptions2 = csfOptions2.filter(({
                        value,
                    }) => value === csfLevel2);
                    const csfLevelText2 = filteredCsfOptions2.length > 0 ? filteredCsfOptions2[0]?.name : '';
                    const logMsg = __('新建用户“${displayName}(${loginName})”成功', {
                        displayName: trim(displayName),
                        loginName: loginName,
                    });
                    let baseParam = {
                            loginName,
                            displayName: trim(displayName),
                            idcardNumber: idcardNumber ? idcardNumber.replace(/^(.{3}).+(.{4})$/, '$1****$2') : '',
                            code,
                            position,
                            managerDisplayName: managerInfo.length ? managerInfo[0].name : '',
                            remark,
                            expireTime: expireTime !== -1 ? formatTime(expireTime / 1000, 'yyyy/MM/dd') : __('永久有效'),
                            email,
                            telNumber,
                            csfLevel: csfLevelText,
                    }
                    await manageLog(
                        ManagementOps.SET,
                        logMsg,
                        this.state.userInfo.show_csf_level2 ? __('用户名 “${loginName}”；显示名 “${displayName}”；用户编码 “${code}”；直属上级 “${managerDisplayName}”；岗位 “${position}”；备注 “${remark}”；邮箱地址 “${email}”；手机号 “${telNumber}”；用户密级 “${csfLevel}”；用户密级2 “${csfLevel2}”；有效期限 “${expireTime}”；身份证号 “${idcardNumber}”；', {
                            ...baseParam,
                            csfLevel2: csfLevelText2,
                        }): __('用户名 “${loginName}”；显示名 “${displayName}”；用户编码 “${code}”；直属上级 “${managerDisplayName}”；岗位 “${position}”；备注 “${remark}”；邮箱地址 “${email}”；手机号 “${telNumber}”；用户密级 “${csfLevel}”；有效期限 “${expireTime}”；身份证号 “${idcardNumber}”；', {
                            ...baseParam,
                        }),
                        Level.INFO,
                    )
                    this.setState({
                        showAddDialog: false,
                    }, async () => {
                        if (userInfo.user.status !== 0) {
                            await Message2.info({
                                message: __('新建用户 ${displayName} 成功。启用用户数已达用户许可总数的上限，当前账号已被禁用。', {
                                    displayName: userInfo.user.displayName,
                                }),
                            })
                            manageLog(
                                ManagementOps.SET,
                                __('启用账号“${displayName}(${loginName})”失败。', {
                                    displayName: trim(displayName),
                                    loginName: loginName,
                                }),
                                __('启用用户数已达用户许可总数的上限'),
                                Level.WARN,
                            )
                        }
                        this.props.onRequestCreateUserSuccess(createData);
                    })
                } catch ({ error }) {
                    this.isRequest = false;
                    if (error) {
                        switch (error.errID) {
                            case ErrorCode.InvalidUserCode:
                                this.setState({
                                    validateState: { ...validateState, code: ValidateState.CodeInvalid },
                                })
                                break;

                            case ErrorCode.UserCodeExit:
                                this.setState({
                                    validateState: { ...validateState, code: ValidateState.UserCodeExit },
                                })
                                break;

                            case ErrorCode.ParentDepartmentNotExist:
                                this.setState({
                                    showAddDialog: false,
                                }, async () => {
                                    this.props.onRequestCancel();
                                    if (await Message2.info({ message: __('新建失败，直属部门“${dep}”不存在，请重新选择。', { dep: name }) })) {
                                        this.props.onRequestRemoveDep(this.props.dep)
                                    }
                                })
                                break;

                            case ErrorCode.UserNameExist:
                                this.setState({
                                    validateState: { ...validateState, loginName: JSON.parse(error.errDetail).type === 'app' ? ValidateState.SameWithUserAccount : ValidateState.NameExist },
                                })
                                break;

                            case ErrorCode.InvalidDisplayName:
                                this.setState({
                                    validateState: { ...validateState, displayName: ValidateState.DisplayNameInvalid },
                                })
                                break;

                            case ErrorCode.DisplayNameExist:
                                this.setState({
                                    validateState: { ...validateState, displayName: ValidateState.DisplayNameExist },
                                })
                                break;

                            case ErrorCode.UserNameDisabled:
                                Message2.info({ message: __('该用户名不可用。') })
                                break;

                            case ErrorCode.UserByAdminExist:
                                Message2.info({ message: __('该用户名已被管理员占用。') })
                                break;

                            case ErrorCode.NameOccupiedByDoc:
                                this.setState({
                                    validateState: { ...validateState, displayName: ValidateState.DisplayNameUsed },
                                })
                                break;

                            case ErrorCode.EmailExist:
                                this.setState({
                                    validateState: {
                                        ...validateState,
                                        email: ValidateState.EmailExist,
                                    },
                                })
                                break;

                            case ErrorCode.DateExpired:
                            case ErrorCode.InvalidPhoneNub:
                            case ErrorCode.PhoneNubExist:
                            case ErrorCode.InvalidCardId:
                            case ErrorCode.InvalidRemarks:
                            case ErrorCode.CardIdExist:
                                Message2.info({ message: getErrorMessage(error.errID) })
                                break;

                            default:
                                Message2.info({ message: error.errMsg })
                                break;
                        }
                    }
                }
            }
        }
    }

    /**
     * 密级切换
     */
    protected updateCsfLevel(type: 'csfLevel' | 'csfLevel2', csfLevel: string): void {
        this.setState({
            userInfo: {
                ...this.state.userInfo,
                [type]: csfLevel,
            },
        })
    }

    /**
     * 更改日期组件的时间时触发事件
     * @param expireTime 从 1970-01-01 开始算，到截止日期之间的时间，单位：毫秒
     */
    protected changeExpireTime(expireTime: number): void {
        this.setState({
            userInfo: {
                ...this.state.userInfo,
                expireTime,
            },
        })
    }
}