import React, { useState, useContext, useEffect } from "react";
import intl from "react-intl-universal";
import { Button, Form, Modal, Tag, message } from "antd";
import AppConfigContext from "@/core/context/AppConfigContext";
import { Level, ManagementOps, manageLog } from "@/core/log";
import { SystemRoleType, getRoleName } from "@/core/role/role";
import { checkMemberExist, setUserRolemMember } from "@/core/thrift/sharemgnt";
import { apis, components } from "@dip/components/dist/dip-components.min.js"
import { fromatItem, getRoleType } from "../../util";
import { VisitorType } from "../../types";
import { defaultModalParams } from "@/util/modal"
import { getErrorMessage } from "@/core/exception";
import styles from "./styles.css";
import classnames from "classnames";
import { getIcon } from "../../index";
import { PickerRangeEnum } from "@/core/apis/console/authorization/type";

const { info } = Modal

export const SystemRoleMember = ({type = 'create', title, currentRole, selectmembersRef, selectRangesRef, selections, onCancel, onSuccess}) => {
    const { config: { userInfo } } = useContext(AppConfigContext)
    const [showVisitorTip, setShowVisitorTip] = useState(false);
    const [showRangeTip, setShowRangeTip] = useState(false);
    const [visitors, setVisitors] = useState([]);
    const [ranges, setRanges] = useState([]);   
    const [btnLoading, setBtnLoading] = useState(false);

    const handleUnique = (origin, selections) => {
        const unique = [...origin, ...selections].reduce((acc, current) => {
            const existingIndex = acc.findIndex((item: any) => item.id === current.id);
            if (existingIndex === -1) {
                acc.push(current);
            }
            return acc;
        }, []);
        return unique
    }

    const handleSystemError = (e) => {
        if(e?.error?.errID) {
            info({
                ...defaultModalParams,
                content: getErrorMessage(e.error.errID),
                getContainer: document.getElementById('isf-web-plugins')
            })
        }
    }

    const addMembers = () => {
        const unmount = apis.mountComponent(
            components.AccessorPicker,
            {
                title: intl.get("select.member"),
                tabs: ["organization"],
                range: ["user"],
                isAdmin: true,
                role: getRoleType(userInfo?.user?.roles),
                onSelect: (data) => {
                    const newSelections = data.map(item => {
                        return fromatItem(item, true);
                    });
                    const  uniqueVisitors = handleUnique(visitors, newSelections)
                    setVisitors(uniqueVisitors);
                    setShowVisitorTip(false);
                    unmount()
                   
                },
                onCancel: () => {
                    unmount();
                },
            },
            selectmembersRef.current,
        );
    }

    const addRanges = () => {
        const unmount = apis.mountComponent(
            components.AccessorPicker,
            {
                title: intl.get("select.range"),
                tabs: ["organization"],
                range: ["department"],
                isAdmin: true,
                role: getRoleType(userInfo?.user?.roles),
                onSelect: (data) => {
                    const newSelections = data.map(item => {
                        return fromatItem(item, true);
                    });
                    const  uniqueRanges = handleUnique(ranges, newSelections)
                    setRanges(uniqueRanges);
                    setShowRangeTip(false);
                    unmount()
                   
                },
                onCancel: () => {
                    unmount();
                },
            },
            selectRangesRef.current,
        );
    }

    const preCheck = ({visitors, ranges}) => {
        let result = true;
        if (!visitors.length) {
            setShowVisitorTip(true);
            result = false;
        }
        if (!ranges.length) {
            setShowRangeTip(true);
            result = false;
        }
        return result;
    }

    const getLogMessage = (memberInfo) => {

        switch (currentRole.id) {
            case SystemRoleType.OrgAudit:
                return intl.get("set.member.orgaudit.log.message", {
                    name: memberInfo.displayName,
                    memberRange: memberInfo.manageDeptInfo.departmentNames.join(intl.get("quota")),
                })

            default:
                return ''
        }
    }

    const setRoleMember = async() => {
        try {
            let existMember = []
            if(type === 'create') {
                for (let member of visitors) {
                    const existResult = await checkMemberExist([currentRole.id, member.id])
                    if (existResult) {
                        existMember = [...existMember, member]
                    }
                }
            }

            if(existMember.length) {
                info({
                    ...defaultModalParams,
                    title: intl.get("member.existed"),
                    content: (
                        <div className={styles["user-list-tip"]}>
                            {
                                existMember.map((cur) =>(
                                    <div key={cur.userId} className={styles["item"]} title={cur.name}>{cur.name}</div>
                                ))
                            }
                        </div>
                    ),
                    getContainer: document.getElementById('isf-web-plugins')
                })
                setBtnLoading(false);
                return
            }

            for (let member of visitors) {
                const departmentIds = ranges.map(cur => cur.id)
                const departmentNames = ranges.map(cur => cur.name)
                await setUserRolemMember([userInfo.id, currentRole.id, {
                    userId: member.id,
                    displayName: member.name,
                    manageDeptInfo: {
                        ncTManageDeptInfo: {
                            departmentIds,
                            departmentNames,
                        }
                    }
                }])

                const memberInfo = {
                    userId: member.id,
                    displayName: member.name,
                    manageDeptInfo: {
                        departmentIds,
                        departmentNames,
                    }
                }
                if(type === "create") {
                    if (currentRole.id === SystemRoleType.OrgManager) {
                        manageLog(
                            ManagementOps.SET,
                            intl.get("set.member.org.log", { userName: memberInfo.displayName, departmentName: memberInfo.manageDeptInfo.departmentNames.join(intl.get(("quota"))) }),
                            getLogMessage(memberInfo),
                            Level.INFO,
                        )
                    } else {
                        manageLog(
                            ManagementOps.SET,
                            intl.get("set.member.log", { roleName: getRoleName(currentRole), userName: memberInfo.displayName }),
                            getLogMessage(memberInfo),
                            Level.INFO,
                        )
                    }
                } else {
                    manageLog(
                        ManagementOps.SET,
                        intl.get('edit.member.log', { userName: memberInfo.displayName, departmentName: memberInfo.manageDeptInfo.departmentNames.join(intl.get("quota")) }),
                        '',
                        Level.INFO,
                    )
                }
            }
            message.success(intl.get(type === "create" ? "add.success" : "edit.success"));
            onSuccess()
            onCancel();
        }catch(e) {
            setBtnLoading(false)
            handleSystemError(e)
        }
    }

    const handleSubmit = async() => {
        const result = preCheck({visitors, ranges});
        
        if(result) {
            setBtnLoading(true);
            await setRoleMember()
        }
    }

    useEffect(() => {
        if(selections.length) {
            const visitors = [{id: selections[0]?.userId, name: selections[0]?.displayName}]
            setVisitors(visitors)
            if(selections[0]?.manageDeptInfo && selections[0].manageDeptInfo?.departmentNames.length ===  selections[0].manageDeptInfo?.departmentIds.length) {
                const ids = selections[0].manageDeptInfo?.departmentIds
                const names = selections[0].manageDeptInfo?.departmentNames
                const ranges = ids.map((cur, index) => {
                    return {id: cur, name: names[index], type: PickerRangeEnum.Dept}
                })
                setRanges(ranges)
            }
        }
    }, [currentRole.id, selections])

    return (
        <Modal 
            centered
            maskClosable={false}
            open={true}
            width={500}
            title={title} 
            onCancel={onCancel} 
            footer={[
                <Button key="submit" type="primary" htmlType="submit" loading={btnLoading} onClick={handleSubmit}>
                    {intl.get('ok')}
                </Button>,
                <Button
                    key="back"
                    onClick={onCancel}
                >
                    {intl.get('cancel')}
                </Button>,
            ]}
            getContainer={document.getElementById("isf-web-plugins") as HTMLElement}
        > 
            <div className={styles["add-system-role"]}>
                <Form>
                    {type === 'create' ? (
                        <Form.Item
                            className={styles['required-label']}
                            label={intl.get('members')}
                            required
                            validateStatus={showVisitorTip ? 'error' : ''}
                            help={showVisitorTip ? <div>{intl.get('select.member.tip')}</div> : ''}
                        >
                            <div className={styles['add-visitor']}>
                                <div className={classnames(styles['visitors'], showVisitorTip && styles['visitor-error'])}>
                                    {!visitors.length && <span className={styles['visitors-tip']}>{intl.get("select.member.placeholder")}</span>}
                                    {visitors.map((item: VisitorType) => {
                                        return (
                                            <Tag 
                                                className={styles['tag']} 
                                                closable={true} 
                                                key={item.id} 
                                                title={item.name} 
                                                onClose={() => {
                                                    const newVisitors = visitors.filter((cur: VisitorType) => cur.id !== item.id)
                                                    setVisitors(newVisitors)
                                                }}
                                            >
                                                <div className={styles['tag-content']}>
                                                    <div className={styles['icon']}>{getIcon(item.type)}</div>
                                                    <span className={styles['text']}>{item.name}</span>
                                                </div>
                                            </Tag>
                                        );
                                    })}
                                </div>
                                <Button
                                    className={styles['add-btn']}
                                    onClick={addMembers}
                                >
                                    {intl.get('add')}
                                </Button>
                            </div>
                        </Form.Item>
                    ): (
                        <Form.Item label={intl.get('members')} className={styles['label']}>
                            <div className={styles['visitor']}>
                                <span className={styles['icon']}>{getIcon(PickerRangeEnum.User)}</span>
                                <span title={selections[0]?.displayName}>{selections[0]?.displayName}</span>
                            </div>
                        </Form.Item>
                    )}
                    <Form.Item 
                        className={styles['required-label']} 
                        label={intl.get("manage.range")} 
                        required 
                        validateStatus={showRangeTip ? 'error' : ''}
                        help={showRangeTip ? <div>{intl.get("select.range.tip")}</div> : ''}
                    > 
                        <div className={styles['add-visitor']}>
                            <div className={classnames(styles['visitors'], showRangeTip && styles['visitor-error'])}>
                                {!ranges.length && <span className={styles['visitors-tip']}>{intl.get("select.range.placeholder")}</span>}
                                {ranges.map((item: VisitorType) => {
                                    return (
                                        <Tag 
                                            className={styles['tag']} 
                                            closable={true} 
                                            key={item.id} 
                                            title={item.name} 
                                            onClose={() => {
                                                const newRanges = ranges.filter((cur: VisitorType) => cur.id !== item.id)
                                                setRanges(newRanges)
                                            }}
                                        >
                                            <div className={styles['tag-content']}>
                                                <div className={styles['icon']}>{getIcon(item.type)}</div>
                                                <span className={styles['text']}>{item.name}</span>
                                            </div>
                                        </Tag>
                                    );
                                })}
                            </div>
                            <Button
                                className={styles['add-btn']}
                                onClick={addRanges}
                            >
                                {intl.get('add')}
                            </Button>
                        </div>
                    </Form.Item>
                </Form>
            </div>
        </Modal>
    )
}