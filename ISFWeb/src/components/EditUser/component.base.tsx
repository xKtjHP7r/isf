import * as React from 'react'
import { noop, trim } from 'lodash';
import { mailAndLenth, phoneNum, isNormalName, isSpace, variousIdCard, isNormalCode, isNormalPosition, isUserNormalName } from '@/util/validators';
import { formatSize, formatTime, convertUnit } from '@/util/formatters';
import { Message2 } from '@/sweet-ui';
import session from '@/util/session';
import { SystemRoleType } from '@/core/role/role';
import { getErrorMessage } from '@/core/exception';
import { ValidateState, Dep, getUserType, Type, UserInfoType } from '@/core/user';
import { ErrorCode } from '@/core/thrift/sharemgnt/errcode';
import { editUser, getUserInfo, getTriSystemStatus } from '@/core/thrift/sharemgnt';
import { manageLog, Level, ManagementOps } from '@/core/log';
import WebComponent from '../webcomponent';
import __ from './locale';
import { getLevelConfig } from '@/core/apis/console/usermanagement';

/**
 * 错误提示索引
 */
interface Validate {
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
    telNum: ValidateState;

    /**
     * 邮箱错误提示
     */
    email: ValidateState;
}

interface EditUserProps extends React.Props<void> {
    /**
     * 选择的用户
     */
    selectUserId: string;

    /**
     * 选择的部门
     */
    dep: Dep | null;

    /**
     * 当前登录的用户
     */
    userid: string;

    /**
     * 取消编辑用户
     */
    onRequestCancel: () => any;

    /**
     * 编辑用户成功
     */
    onRequestEditUserSuccess: (user: Core.ShareMgnt.ncTUsrmUserInfo) => void;
}

interface EditUserState {
    /**
     * 重新编辑用户信息
     */
    userInfo: Core.ShareMgnt.ncTUsrmUserInfo;

    /**
     * 用户密级
     */
    csfOptions: Array<{name: string, value: string}>;

    /**
     * 用户密级2
     */
    csfOptions2: Array<{name: string, value: string}>;

    /**
     * 直属上级
     */
    managerInfo: UserInfoType[];

    /**
     * 身份证是否处于确定保存编辑状态
     */
    isIDNumEdit: boolean;

    /**
     * 错误提示信息索引
     */
    validateState: Validate;

    /**
     * 是否显示编辑弹框
     */
    showEditDialog: boolean;

    /**
     * 是否显示选择直属上级弹框
     */
    showAddDirectSupervisorDialog: boolean;

    /**
     * 是否修改了用户信息
     */
    isEditUserInfo: boolean;
}
export default class EditUserBase extends WebComponent<EditUserProps, EditUserState> {
    static defaultProps: EditUserProps = {
        selectUserId: '',
        dep: null,
        userid: '',
        onRequestCancel: noop,
        onRequestEditUserSuccess: noop,
    }

    state: EditUserState = {
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
            certification: '',
            email: '',
            telNum: '',
            idCard: '',
            expireTime: null,
        },
        csfOptions: [],
        csfOptions2: [],
        validateState: {
            displayName: ValidateState.Normal,
            code: ValidateState.Normal,
            position: ValidateState.Normal,
            remark: ValidateState.Normal,
            email: ValidateState.Normal,
            telNum: ValidateState.Normal,
        },
        managerInfo: [],
        isIDNumEdit: false,
        showEditDialog: false,
        showAddDirectSupervisorDialog: false,
        isEditUserInfo: false,
    }

    editInfo: { user: { displayName: string; idcardNumber: null; priority: null; pwdControl: null; loginName: ''; password: '' }; originalPwd: null };  // 当前编辑的用户信息
    triSystemStatus: boolean // 是否开启三权分立
    isAdmin: boolean // 是否系统管理员
    isSecurit: boolean // 是否安全管理员
    isRequest: boolean // 是否在请求中

    async componentDidMount() {
        try {
            const data = await getUserInfo([this.props.selectUserId]);
            this.triSystemStatus = await getTriSystemStatus();
            this.isAdmin = session.get('isf.userInfo').user.roles.some((role) => [SystemRoleType.Admin].includes(role.id))
            this.isSecurit = session.get('isf.userInfo').user.roles.some((role) => [SystemRoleType.Securit].includes(role.id))

            const { userInfo, validateState } = this.state;
            const { expireTime, loginName, displayName, code, managerID, managerDisplayName, position, remark, userType, email, telNumber, idcardNumber, csfLevel, csfLevel2 } = data.user;

            if (data.id === session.get('isf.userid')) {
                await Message2.info({ message: __('您无法编辑自身账号。') })
                this.props.onRequestCancel()
            } else if (data.user) {
                this.setState({
                    userInfo: {
                        ...userInfo,
                        loginName,
                        displayName,
                        code,
                        position,
                        remark,
                        certification: getUserType(userType),
                        email,
                        telNum: telNumber,
                        idCard: idcardNumber,
                        csfLevel,
                        csfLevel2,
                        expireTime: expireTime === -1 ? -1 : expireTime * 1000 * 1000,
                    },
                    managerInfo: managerID && managerDisplayName ? [{ id: managerID, name: managerDisplayName, type: 'user' }] : [],
                    showEditDialog: true,
                    validateState: {
                        ...validateState,
                    },
                    isEditUserInfo: false,
                })
                this.editInfo = data;
            }
            this.getUserCsfInfos()
        } catch ({ error }) {
            if (error) {
                switch (error.errID) {
                    case ErrorCode.UserNotExist:
                        await Message2.info({ message: getErrorMessage(error.errID) })
                        this.props.onRequestCancel();
                        break;
                    default:
                        await Message2.info({ message: error.errMsg })
                        this.props.onRequestCancel();
                }
            }
        }
    }

    /*
     * 获取用户密级枚举
     */
    private async getUserCsfInfos() {
        const {csf_level_enum, csf_level2_enum, show_csf_level2} = await getLevelConfig({fields: 'csf_level_enum,csf_level2_enum,show_csf_level2'})
        this.setState({
            userInfo: {
                ...this.state.userInfo,
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
        const { userInfo: { displayName, code, position, remark, telNum, email, idCard }, validateState } = this.state
        const validateDisplayName = isUserNormalName(trim(displayName));
        const validateCode = isNormalCode(trim(code));
        const validatePosition = isNormalPosition(trim(position))
        const validateRemark = isNormalName(trim(remark));
        const validateTelNum = phoneNum(telNum);
        const validateEmail = mailAndLenth(email, 4, 101);

        switch (type) {
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

            case Type.TelNumber:
                this.setState({
                    validateState: {
                        ...validateState,
                        telNum: validateTelNum ? ValidateState.Normal : telNum ? ValidateState.PhoneInvalid : ValidateState.Normal,
                    },
                })
                break;
        }
    }

    /**
    * 检查表单合法性
    */
    private checkForm = (): boolean => {
        const { displayName, code, position, remark, telNum, email, idCard } = this.state.userInfo;
        const validateDisplayName = isUserNormalName(trim(displayName));
        const validateCode = isNormalCode(trim(code));
        const validatePosition = isNormalPosition(trim(position));
        const validateRemark = isNormalName(trim(remark));
        const validateTelNum = phoneNum(telNum);
        const validateEmail = mailAndLenth(email, 4, 101);
        const validateIdcardNum = idCard !== this.editInfo.user.idcardNumber ? variousIdCard(idCard) : true;

        if (validateDisplayName && (validateIdcardNum || !idCard) && (validateEmail || !email) && (validateCode || !code) && (validateRemark || !remark) && (validateTelNum || !telNum)) {
            return true;
        } else {
            this.setState({
                validateState: {
                    ...this.state.validateState,
                    email: validateEmail ? ValidateState.Normal : email ? ValidateState.EamilInvalid : ValidateState.Normal,
                    displayName: validateDisplayName ? ValidateState.Normal : displayName ? ValidateState.DisplayNameInvalid : ValidateState.Empty,
                    position: validatePosition ? ValidateState.Normal : position ? ValidateState.PositionInvalid : ValidateState.Normal,
                    code: validateCode ? ValidateState.Normal : code ? ValidateState.CodeInvalid : ValidateState.Normal,
                    remark: validateRemark ? ValidateState.Normal : remark ? ValidateState.RemarksInvalid : ValidateState.Normal,
                    telNum: validateTelNum ? ValidateState.Normal : telNum ? ValidateState.PhoneInvalid : ValidateState.Normal,
                },
                isIDNumEdit: true,
            })
            return false;
        }
    }

    /**
     * 设置文本框变更事件
     */
    protected handleValueChange(inputInfo: { displayName?: string; code?: string; position?: string; remark?: string; telNum?: string; email?: string }) {
        const { userInfo, validateState } = this.state;
        const { displayName, code, position, remark, telNum,email } = inputInfo;

        this.setState({
            userInfo: { ...userInfo, ...inputInfo },
            validateState: {
                ...validateState,
                displayName: displayName ? ValidateState.Normal : validateState.displayName,
                code: code ? ValidateState.Normal : validateState.code,
                position: position ? ValidateState.Normal : validateState.position,
                remark: remark ? ValidateState.Normal : validateState.remark,
                telNum: telNum ? ValidateState.Normal : validateState.telNum,
                email: email ? ValidateState.Normal : validateState.email,
            },
            isEditUserInfo: true,
        })
    }

    /**
     * 保存编辑用户
     */
    protected editUser = async () => {
        if(!this.state.isEditUserInfo){
            this.props.onRequestCancel();
            return;
        }
        if (this.checkForm()) {
            this.setState({
                isEditUserInfo: false,
            })
            const { userInfo, managerInfo, userInfo: { displayName, code, position, remark, email, telNum, idCard, csfLevel, csfLevel2, expireTime }, csfOptions, csfOptions2, validateState } = this.state;
            const { userid, dep } = this.props;
            const { user: { idcardNumber, priority, loginName } } = this.editInfo;
            let { id } = dep;

            if (id === '-2') {
                id = -1;
            }
            let data = {
                ncTEditUserParam: {
                    id: this.editInfo.id,
                    displayName: trim(displayName),
                    code: trim(code),
                    position: trim(position),
                    managerID: managerInfo.length ? managerInfo[0].id : '',
                    remark: trim(remark),
                    idcardNumber: (idCard === idcardNumber) ? null : idCard,
                    priority,
                    csfLevel,
                    csfLevel2,
                    email,
                    telNumber: telNum,
                    expireTime: expireTime === -1 ? -1 : expireTime / 1000 / 1000,
                },
            }

            if (!this.isRequest) {
                try {
                    this.isRequest = true;
                    await editUser([data, userid])
                    let displayNameText = displayName;
                    if(displayName !== this.editInfo.user.displayName){
                        displayNameText = __('由 ${oldText} 改为 ${newText}', {
                            oldText: this.editInfo.user.displayName,
                            newText: displayName,
                        });
                    }
                    const user = {
                        ...this.editInfo,
                        user: Object.assign({}, this.editInfo.user, {
                            displayName: trim(displayName),
                            code: trim(code),
                            position: trim(position),
                            remark: trim(remark),
                            idcardNumber: (idCard === idcardNumber) ? null : idCard,
                            emailAddress: email,
                            csfLevel: parseInt(csfLevel),
                            csfLevel2: parseInt(csfLevel2),
                            telNumber: telNum,
                            expireTime: expireTime === -1 ? -1 : expireTime / 1000 / 1000,
                            managerID: managerInfo.length ? managerInfo[0].id : null,
                            managerDisplayName: managerInfo.length ? managerInfo[0].name : null,
                        }),
                    }
                    const expireTimeDate = expireTime !== -1 ? formatTime(expireTime / 1000, 'yyyy/MM/dd') : __('永久有效')
                    const filteredCsfOptions = csfOptions.filter(({
                        value,
                    }) => value === csfLevel);
                    const csfLevelText = filteredCsfOptions.length > 0 ? filteredCsfOptions[0]?.name : '';
                    const filteredCsfOptions2 = csfOptions2.filter(({
                        value,
                    }) => value === csfLevel2)
                    const csfLevel2Text = filteredCsfOptions2.length > 0 ? filteredCsfOptions2[0]?.name : '';
                    const logMsg = __('编辑用户“${displayName}(${loginName})”成功', {
                        displayName: trim(displayName),
                        loginName: loginName,
                    });

                    let textCsfLevel = csfLevelText;
                    if(csfLevel !== this.editInfo.user.csfLevel){
                        const oldCsfOption = csfOptions.find(({ value }) => value === parseInt(this.editInfo.user.csfLevel));
                        const oldText = oldCsfOption ? oldCsfOption.name : '';
                        textCsfLevel = __('由 ${oldText} 改为 ${newText}', {
                            oldText,
                            newText: csfLevelText,
                        })
                    }
                    let textCsfLevel2 = csfLevel2Text;
                    if(csfLevel2 !== this.editInfo.user.csfLevel2){
                        const oldCsfOption2 = csfOptions2.find(({ value }) => value === parseInt(this.editInfo.user.csfLevel2));
                        const oldText2 = oldCsfOption2 ? oldCsfOption2.name : '';
                        textCsfLevel2 = __('由 ${oldText} 改为 ${newText}', {
                            oldText: oldText2,
                            newText: csfLevel2Text,
                        })
                    }
                    let baseParam = {
                            loginName: loginName,
                            display: displayNameText,
                            code,
                            position,
                            managerDisplayName: managerInfo.length ? managerInfo[0].name : '',
                            remark: remark,
                            idcardNumber: idCard ? idCard.replace(/^(.{3}).+(.{4})$/, '$1****$2') : '',
                            userType: userInfo.certification,
                            expireTime: expireTimeDate,
                            email: email,
                            telNum: telNum,
                            csfLevel: textCsfLevel,
                    }
                    await manageLog(
                        ManagementOps.SET,
                        logMsg,
                        this.state.userInfo.show_csf_level2 ? __('用户名 “${loginName}”；显示名 “${display}”；用户编码 “${code}”；直属上级 “${managerDisplayName}”；岗位 “${position}”；备注 “${remark}”；认证类型 “${userType}”；邮箱地址 “${email}”；手机号 “${telNum}”；用户密级 “${csfLevel}”；用户密级2 “${csfLevel2}”；有效期限 “${expireTime}”；身份证号 “${idcardNumber}”；', {
                            ...baseParam,
                            csfLevel2: textCsfLevel2,
                        }): __('用户名 “${loginName}”；显示名 “${display}”；用户编码 “${code}”；直属上级 “${managerDisplayName}”；岗位 “${position}”；备注 “${remark}”；认证类型 “${userType}”；邮箱地址 “${email}”；手机号 “${telNum}”；用户密级 “${csfLevel}”；有效期限 “${expireTime}”；身份证号 “${idcardNumber}”；', {
                            ...baseParam,
                        }),
                        Level.INFO,
                    )
                    this.props.onRequestEditUserSuccess(user);
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
                                    showEditDialog: false,
                                }, async () => {
                                    await Message2.info({ message: __('编辑失败，直属部门“${dep}”不存在，请重新选择。', { dep: name }) })
                                    this.props.onRequestCancel();
                                })
                                break;

                            case ErrorCode.DisplayNameExist:
                                this.setState({
                                    validateState: { ...validateState, displayName: ValidateState.DisplayNameExist },
                                })
                                break;

                            case ErrorCode.InvalidDisplayName:
                                this.setState({
                                    validateState: { ...validateState, displayName: ValidateState.DisplayNameInvalid },
                                })
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
                            case ErrorCode.InvalidCsfLevel:
                            case ErrorCode.InvalidPhoneNub:
                            case ErrorCode.PhoneNubExist:
                            case ErrorCode.InvalidCardId:
                            case ErrorCode.InvalidRemarks:
                            case ErrorCode.CardIdExist:
                                Message2.info({ message: getErrorMessage(error.errID) })
                                break;
                            default:
                                await Message2.info({ message: error.errMsg })
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
        const { userInfo, isEditUserInfo } = this.state;
        this.setState({
            userInfo: {
                ...userInfo,
                [type]: csfLevel,
            },
            isEditUserInfo: isEditUserInfo || userInfo[type] !== csfLevel,
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
            isEditUserInfo: true,
        })
    }

}