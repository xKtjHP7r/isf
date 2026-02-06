import * as React from 'react';
import { getErrorMessage } from '@/core/exception';
import { Dialog2 as Dialog, FlexBox, ComboArea, Button, Panel, MessageDialog, Text } from '@/ui/ui.desktop';
import { NodeType } from '@/core/organization';
import SetManagerByDepBase from './component.base';
import SearchDep from '../SearchDep/component.desktop';
import OrganizationTree from '../OrganizationTree/component.view';
import __ from './locale';
import styles from './styles.view';
import { Spin } from 'antd';
import { LoadingOutlined } from '@ant-design/icons';

export default class SetManagerByDep extends SetManagerByDepBase {
    render() {
        return (
            <div>
                {
                    this.state.isConfigManager ?
                        (
                            <Dialog
                                role={'ui-dialog2'}
                                title={__('设置组织管理员')}
                                onClose={this.props.onCancel}
                            >
                                <Panel role={'ui-panel'}>
                                    <Panel.Main role={'ui-panel.main'}>
                                        <FlexBox role={'ui-flexbox'}>
                                            <FlexBox.Item align="left top" role={'ui-flexbox.item'}>
                                                <div className={styles['dep-box']}>
                                                    <ComboArea
                                                        role={'ui-comboarea'}
                                                        minHeight={80}
                                                        width={380}
                                                        uneditable={true}
                                                        value={this.state.managers}
                                                        formatter={this.userDataFormatter}
                                                        onChange={(data) => { this.deleteManager(data) }}
                                                    />
                                                </div>
                                            </FlexBox.Item>
                                            <FlexBox.Item align="right top" role={'ui-flexbox.item'}>
                                                <div className={styles['dep-btn']}>
                                                    <Button onClick={this.openAddManager} role={'ui-button'}>
                                                        {__('添加')}
                                                    </Button>
                                                </div>
                                            </FlexBox.Item>
                                        </FlexBox>
                                    </Panel.Main>
                                    <Panel.Footer role={'ui-panel.footer'}>
                                        <Panel.Button
                                            theme='oem'
                                            onClick={this.onConfirmManager} >
                                            {
                                                __('确定')
                                            }
                                        </Panel.Button>
                                        <Panel.Button
                                            role={'ui-panel.button'}
                                            onClick={this.props.onCancel}
                                        >
                                            {
                                                __('取消')
                                            }
                                        </Panel.Button>
                                    </Panel.Footer>
                                </Panel>

                            </Dialog>
                        ) :
                        null
                }

                {
                    this.state.isAddingManager ?
                        (
                            <Dialog
                                role={'ui-dialog2'}
                                title={__('添加组织管理员')}
                                onClose={this.cancelAddManager}
                            >
                                <Panel role={'ui-panel'}>
                                    <Panel.Main role={'ui-panel.main'}>
                                        <FlexBox role={'ui-flexbox'}>
                                            <FlexBox.Item role={'ui-flexbox.item'}>
                                                <div>
                                                    <div className={styles['selected-user']}>
                                                        <div className={styles['add-manager-user']}>
                                                            <div className={styles['add-manager-label']}>
                                                                {
                                                                    __('已选：')
                                                                }
                                                            </div>
                                                            <div className={styles['add-manager-name']}>
                                                                <Text className={styles['name']} role={'ui-text'}>
                                                                    {
                                                                        this.state.currentUser && this.state.currentUser.user ?
                                                                            this.state.currentUser.user.displayName :
                                                                            '---'
                                                                    }
                                                                </Text>
                                                            </div>
                                                        </div>
                                                    </div>
                                                    <div className={styles['search-box']}>
                                                        <SearchDep
                                                            onSelectDep={this.selectUser}
                                                            selectType={[NodeType.USER]}
                                                            userid={this.props.userid}
                                                            width={320}
                                                        />
                                                    </div>
                                                    <div className={styles['organization-tree']}>
                                                        <OrganizationTree
                                                            userid={this.props.userid}
                                                            selectType={[NodeType.USER]}
                                                            onSelectionChange={this.selectUser}
                                                        />
                                                    </div>
                                                </div>
                                            </FlexBox.Item>
                                        </FlexBox>

                                    </Panel.Main>

                                    <Panel.Footer role={'ui-panel.footer'}>
                                        <Panel.Button
                                            theme='oem'
                                            role={'ui-panel.button'}
                                            onClick={this.onConfirmAddManager}
                                            disabled={!this.state.currentUser}
                                        >
                                            {
                                                __('确定')
                                            }
                                        </Panel.Button>
                                        <Panel.Button onClick={this.cancelAddManager} role={'ui-panel.button'}>
                                            {
                                                __('取消')
                                            }
                                        </Panel.Button>
                                    </Panel.Footer>
                                </Panel>
                            </Dialog>
                        ) :
                        null
                }

                {
                    this.state.isSetting ?
                        (<Spin size='large' tip={__('正在配置组织管理员...')} fullscreen indicator={<LoadingOutlined style={{ color: '#fff' }} spin />}/>) :
                        null
                }
                {
                    this.state.errorStatus && this.state.errorStatus.error && this.state.errorStatus.error.errID ?
                        (
                            <MessageDialog onConfirm={this.closeError} role={'ui-messagedialog'}>
                                {
                                    getErrorMessage(this.state.errorStatus.error.errID)
                                }
                            </MessageDialog>
                        ) :
                        null
                }
            </div>
        )
    }

}