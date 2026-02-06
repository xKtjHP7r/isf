import * as React from 'react';
import Dialog from '@/ui/Dialog2/ui.desktop';
import Panel from '@/ui/Panel/ui.desktop';
import { getErrorMessage } from '@/core/exception';
import { usrmGetThirdPartyRootNode } from '@/core/thrift/sharemgnt/sharemgnt';
import { Radio } from '@/sweet-ui';
import { MessageDialog, UIIcon, Text } from '@/ui/ui.desktop';
import { SelectType } from '@/ui/Tree2/ui.base'
import Tree2 from '@/ui/Tree2/ui.desktop';
import FlexBox from '@/ui/FlexBox/ui.desktop';
import ValidityBox2 from '../ValidityBox2/component.view';
import ImportOrganizationBase from './component.base';
import __ from './locale';
import styles from './styles.view';
import { Spin } from 'antd';
import { LoadingOutlined } from '@ant-design/icons';

export default class ImportOrganization extends ImportOrganizationBase {
    root = usrmGetThirdPartyRootNode([this.props.userid]).then((res) => (res.map((value) => ({ ...value, isRoot: true }))))
    render() {
        return (
            <div >
                {
                    this.state.progress === -1 && !this.state.failMessage && !this.state.errorStatus ?
                        (<Dialog
                            title={__('导入第三方用户组织')}
                            onClose={this.props.onCancel}
                        >
                            <Panel>
                                <Panel.Main>
                                    <FlexBox>
                                        <FlexBox.Item align="top">
                                            <div className={styles['select-org']}>
                                                <div>
                                                    {
                                                        __('请在列表中选择您要导入的用户或组织：')
                                                    }
                                                </div>
                                                <div className={styles['select-org-box']}>
                                                    <div className={styles['select-tree']}>
                                                        <Tree2
                                                            selectType={SelectType.CASCADE_MULTIPLE}
                                                            checkbox={true}
                                                            data={this.root}
                                                            isLeaf={this.getNodeIsLeaf}
                                                            renderNode={(node) => node ? this.getNodeTemplate(node) : ''}
                                                            getNodeChildren={this.getChildren}
                                                            onSelectionChange={this.getSelectedNode}
                                                        />
                                                    </div>

                                                </div>
                                            </div>
                                        </FlexBox.Item>
                                        <FlexBox.Item align="top">
                                       
                                            <div className={styles['import-config-title']}>
                                                {
                                                    __('在导入过程中，如果发现当前系统已存在同名的用户：')
                                                }
                                            </div>
                                            <div className={styles['import-config']}>
                                                <div>
                                                    <Radio
                                                        name="namerepeat"
                                                        value={true}
                                                        onChange={({ detail: { value } }) => this.setUserCover(value)}
                                                        checked={this.state.option.userCover}
                                                    >
                                                        {__('覆盖同名用户')}
                                                    </Radio>
                                                </div>
                                                <div className={styles['import-config-option']}>
                                                    <Radio
                                                        name="namerepeat"
                                                        value={false}
                                                        onChange={({ detail: { value } }) => this.setUserCover(value)}
                                                        checked={!this.state.option.userCover}
                                                    >
                                                        {__('跳过同名用户')}
                                                    </Radio>

                                                </div>
                                            </div>
                                            <div className={styles['import-config-title']}>
                                                {
                                                    __('对于每一个导入的新用户：')
                                                }
                                            </div>
                                            <div className={styles['import-config']}>
                                                <label>
                                                    {
                                                        __('用户有效期限：')
                                                    }
                                                </label>
                                                <div className={styles['expireTime-box']}>
                                                    <ValidityBox2
                                                        width={'100%'}
                                                        allowPermanent={true}
                                                        value={this.state.expireTime}
                                                        selectRange={[new Date()]}
                                                        onChange={(value) => { this.changeExpireTime(value) }}
                                                    />
                                                </div>
                                            </div>

                                        </FlexBox.Item>
                                    </FlexBox>
                                </Panel.Main>
                                <Panel.Footer>
                                    <Panel.Button
                                        theme='oem'
                                        disabled={!this.state.selectedData.length}
                                        onClick={this.importThirdUser}
                                    >
                                        {__('导入')}
                                    </Panel.Button>
                                    <Panel.Button onClick={this.props.onCancel} >{__('取消')}</Panel.Button>
                                </Panel.Footer>
                            </Panel>
                            {
                                this.state.invalidExpireTime ?
                                    (
                                        <MessageDialog onConfirm={this.closeInvalidExpireTimeTip.bind(this)}>
                                            {
                                                __('该日期已过期，请重新选择。')
                                            }
                                        </MessageDialog>
                                    ) :
                                    null
                            }
                        </Dialog>) :
                        null
                }
                {
                    this.state.progress !== -1 ?
                        (
                            <Spin size='large' tip={__('正在导入 ${progress}%...', { progress: this.state.progress })} fullscreen indicator={<LoadingOutlined style={{ color: '#fff' }} spin />}/>
                        ) :
                        null
                }
                {
                    this.state.failMessage ?
                        (
                            <MessageDialog onConfirm={this.closeFailInfo}>
                                {
                                    this.state.failMessage
                                }
                            </MessageDialog>
                        ) :
                        null
                }

                {
                    this.state.errorStatus ?
                        (
                            <MessageDialog onConfirm={this.closeErrorInfo}>
                                {
                                    getErrorMessage(this.state.errorStatus)
                                }
                            </MessageDialog>
                        ) :
                        null
                }
            </div >
        )
    }

    getNodeTemplate(node) {
        if (node.displayName) {
            return (
                <span className={styles['node-text']}>
                    <UIIcon code={'\uf007'} size={16} className={styles['node']} />
                    <Text>
                        {
                            node.displayName
                        }
                    </Text>
                </span>
            )
        }

        if (node.isRoot) {
            return (
                <span className={styles['node-text']}>
                    <UIIcon code={'\uf008'} size={16} className={styles['node']} />
                    <Text>
                        {
                            node.name
                        }
                    </Text>
                </span>
            )
        }

        return (
            <span className={styles['node-text']}>
                <UIIcon code={'\uf009'} size={16} className={styles['node']} />
                <Text>
                    {
                        node.name
                    }
                </Text>
            </span>
        )
    }
}