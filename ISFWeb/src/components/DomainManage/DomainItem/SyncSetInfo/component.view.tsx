import * as React from 'react'
import { Form, Text } from '@/ui/ui.desktop'
import { Switch, ComboArea, Button, ValidateNumber, Select, Radio } from '@/sweet-ui'
import { NodeType } from '@/core/organization';
import CsfWarnning from '../../../CsfWarnning/component.view'
import OrganizationPicker from '../../../OrganizationPicker/component.view'
import SyncObjectPicker from '../../SyncObjectPicker/component.view'
import { ValidateMessages, ActionType } from '../../helper'
import SyncSetInfoBase, { VerifyType, SyncUnit, ExpireTime, SyncMode } from './component.base'
import styles from './styles.view';
import __ from './locale'

export default class SyncSetInfo extends SyncSetInfoBase {
    /**
     * 格式化同步对象
     */
    private formatter = (item): string => {
        return item.name
    }

    /**
    * 转出数据时转换数据格式
    */
    private convererOutSyncTarget = ({ id, name }): { id: string; name: string } => {
        return {
            id,
            name,
        }
    }

    /**
    * 转出数据时转换数据格式
    */
    private convererOutSyncObject = (detail) => {
        return detail
    }

    render() {
        const { selection, actionType, domainInfo: { id, name } } = this.props;
        const {
            syncSettingInfo: {
                periodicSyncStatus, syncObject, syncInterval, syncIntervalPlaceholder, syncIntervalUnit, expireTime, syncTarget, userStatus, syncMode, csfLevel, csfOptions,
            },
            validateStatus: {
                syncIntervalValidateStatus,
            },
            isSyncSettingEditStatus,
            isShowsyncObjectDialog,
            isShowSyncTargetDialog,
        } = this.state;

        return (
            <div>
                <Form role={'ui-form'}>
                    <Form.Row role={'ui-form.row'}>
                        <Form.Label role={'ui-form.label'}>{__('定期同步：')}</Form.Label>
                        <Form.Field role={'ui-form.field'}>
                            <Switch
                                role={'sweetui-switch'}
                                checked={periodicSyncStatus}
                                onChange={this.changeSyncStatus}
                            />
                        </Form.Field>
                    </Form.Row>
                    <Form.Row role={'ui-form.row'}>
                        <Form.Label role={'ui-form.label'}>{__('域名：')}</Form.Label>
                        <Form.Field role={'ui-form.field'}>
                            <div className={styles['wrapper']}>
                                <Text role={'ui-text'} className={styles['name']}>{name}</Text>
                            </div>
                        </Form.Field>
                    </Form.Row>
                    <Form.Row role={'ui-form.row'}>
                        <Form.Label role={'ui-form.label'} align={'top'}>
                            <span className={styles['form-label']}>{__('同步源：')}</span>
                        </Form.Label>
                        <Form.Field role={'ui-form.field'}>
                            <div className={styles['wrapper']}>
                                <ComboArea
                                    role={'sweetui-comboarea'}
                                    width={368}
                                    height={100}
                                    uneditable={true}
                                    disabled={!periodicSyncStatus}
                                    placeholder={__('默认以当前域为同步源')}
                                    value={syncObject}
                                    formatter={this.formatter}
                                    onChange={this.changeSyncObject}
                                />
                            </div>
                            <Button
                                role={'sweetui-button'}
                                size={'auto'}
                                className={styles['block']}
                                disabled={!periodicSyncStatus}
                                onClick={() => this.setState({ isShowsyncObjectDialog: true })}
                            >
                                {__('选择')}
                            </Button>
                        </Form.Field>
                    </Form.Row>
                    <Form.Row role={'ui-form.row'}>
                        <Form.Label role={'ui-form.label'} align={'top'}>
                            <span className={styles['form-label']}>{__('同步目标：')}</span>
                        </Form.Label>
                        <Form.Field role={'ui-form.field'}>
                            <div className={styles['one-line']}>
                                <ComboArea
                                    role={'sweetui-comboarea'}
                                    width={280}
                                    height={32}
                                    disabled={!periodicSyncStatus}
                                    placeholder={__('默认同步到以域控命名的新组织')}
                                    uneditable={true}
                                    value={syncTarget}
                                    formatter={this.formatter}
                                    onChange={this.changeSyncTarget}
                                />
                                <Button
                                    role={'sweetui-button'}
                                    width={80}
                                    className={styles['chose']}
                                    disabled={!periodicSyncStatus}
                                    onClick={() => this.setState({ isShowSyncTargetDialog: true })}
                                >
                                    {__('选择')}
                                </Button>
                            </div>
                        </Form.Field>
                    </Form.Row>
                    <Form.Row role={'ui-form.row'}>
                        <Form.Label role={'ui-form.label'}>{__('同步周期：')}</Form.Label>
                        <Form.Field role={'ui-form.field'}>
                            <div className={styles['one-line']}>
                                <ValidateNumber
                                    role={'sweetui-validatenumber'}
                                    width={280}
                                    disabled={!periodicSyncStatus}
                                    min={1}
                                    max={999999}
                                    maxLength={6}
                                    precision={0}
                                    step={1}
                                    placeholder={syncIntervalPlaceholder}
                                    value={syncInterval}
                                    onValueChange={this.changeSyncInterval}
                                    onBlur={() => this.verifySyncSettingInfo(VerifyType.SyncInterval)}
                                    validateState={syncIntervalValidateStatus}
                                    validateMessages={ValidateMessages}
                                />
                                <Select
                                    role={'sweetui-select'}
                                    className={styles['chose']}
                                    width={80}
                                    disabled={!periodicSyncStatus}
                                    value={syncIntervalUnit}
                                    onChange={this.changeSyncIntervalUnit}
                                >
                                    <Select.Option
                                        value={SyncUnit.Minutes}
                                        selected={syncIntervalUnit === SyncUnit.Minutes}
                                    >
                                        {__('分钟')}
                                    </Select.Option>
                                    <Select.Option
                                        value={SyncUnit.Hour}
                                        selected={syncIntervalUnit === SyncUnit.Hour}
                                    >
                                        {__('小时')}
                                    </Select.Option>
                                    <Select.Option
                                        value={SyncUnit.Day}
                                        selected={syncIntervalUnit === SyncUnit.Day}
                                    >
                                        {__('天')}
                                    </Select.Option>
                                </Select>
                            </div>
                        </Form.Field>
                    </Form.Row>
                    <Form.Row role={'ui-form.row'}>
                        <Form.Label role={'ui-form.label'}>{__('新建用户密级：')}</Form.Label>
                        <Form.Field role={'ui-form.field'}>
                            <div className={styles['wrapper']}>
                                <Select
                                    role={'sweetui-select'}
                                    width={368}
                                    value={csfLevel}
                                    disabled={!periodicSyncStatus}
                                    onChange={({ detail }) => this.updateCsfLevel(detail)}
                                >
                                    {
                                        csfOptions.map((secret) => (
                                            <Select.Option
                                                value={secret.value}
                                                key={secret.value}
                                                selected={csfLevel === secret.value}
                                            >
                                                {secret.name}
                                            </Select.Option>
                                        ))
                                    }
                                </Select>
                                <CsfWarnning />
                            </div>
                        </Form.Field>
                    </Form.Row>
                    <Form.Row role={'ui-form.row'}>
                        <Form.Label role={'ui-form.label'}>{__('用户有效期限：')}</Form.Label>
                        <Form.Field role={'ui-form.field'}>
                            <Select
                                role={'sweetui-select'}
                                width={368}
                                value={expireTime}
                                disabled={!periodicSyncStatus}
                                onChange={this.changeExpireTime}
                            >
                                <Select.Option
                                    value={ExpireTime.OneMonth}
                                    selected={expireTime === ExpireTime.OneMonth}
                                >
                                    {__('一个月')}
                                </Select.Option>
                                <Select.Option
                                    value={ExpireTime.ThreeMonths}
                                    selected={expireTime === ExpireTime.ThreeMonths}
                                >
                                    {__('三个月')}
                                </Select.Option>
                                <Select.Option
                                    value={ExpireTime.HalfYear}
                                    selected={expireTime === ExpireTime.HalfYear}
                                >
                                    {__('半年')}
                                </Select.Option>
                                <Select.Option
                                    value={ExpireTime.OneYear}
                                    selected={expireTime === ExpireTime.OneYear}
                                >
                                    {__('一年')}
                                </Select.Option>
                                <Select.Option
                                    value={ExpireTime.TwoYears}
                                    selected={expireTime === ExpireTime.TwoYears}
                                >
                                    {__('两年')}
                                </Select.Option>
                                <Select.Option
                                    value={ExpireTime.ThreeYears}
                                    selected={expireTime === ExpireTime.ThreeYears}
                                >
                                    {__('三年')}
                                </Select.Option>
                                <Select.Option
                                    value={ExpireTime.FourYears}
                                    selected={expireTime === ExpireTime.FourYears}
                                >
                                    {__('四年')}
                                </Select.Option>
                                <Select.Option
                                    value={ExpireTime.Forever}
                                    selected={expireTime === ExpireTime.Forever}
                                >
                                    {__('永久有效')}
                                </Select.Option>
                            </Select>
                        </Form.Field>
                    </Form.Row>
                    <Form.Row role={'ui-form.row'}>
                        <Form.Label role={'ui-form.label'}>{__('用户默认状态：')}</Form.Label>
                        <Form.Field role={'ui-form.field'}>
                            <Radio
                                role={'sweetui-radio'}
                                disabled={!periodicSyncStatus}
                                value={true}
                                checked={userStatus}
                                onChange={() => this.changeUserStatus(true)}
                            >
                                {__('启用')}
                            </Radio>
                            <Radio
                                role={'sweetui-radio'}
                                disabled={!periodicSyncStatus}
                                value={false}
                                checked={!userStatus}
                                onChange={() => this.changeUserStatus(false)}
                            >
                                {__('禁用')}
                            </Radio>
                        </Form.Field>
                    </Form.Row>
                    <Form.Row role={'ui-form.row'}>
                        <Form.Label role={'ui-form.label'} align={'top'}>{__('同步方式：')}</Form.Label>
                        <Form.Field role={'ui-form.field'}>
                            <div className={styles['wrapper']}>
                                <div className={styles['radio']}>
                                    <Radio
                                        role={'sweetui-radio'}
                                        disabled={!periodicSyncStatus}
                                        value={SyncMode.ALL}
                                        checked={syncMode === SyncMode.ALL}
                                        onChange={() => this.changeSyncMode(SyncMode.ALL)}
                                    >
                                        {__('同步选中的对象及其成员（包括上层的组织结构）')}
                                    </Radio>
                                </div>
                                <div className={styles['radio']}>
                                    <Radio
                                        role={'sweetui-radio'}
                                        disabled={!periodicSyncStatus}
                                        value={SyncMode.PART}
                                        checked={syncMode === SyncMode.PART}
                                        onChange={() => this.changeSyncMode(SyncMode.PART)}
                                    >
                                        {__('同步选中的对象及其成员（不包括上层的组织结构）')}
                                    </Radio>
                                </div>
                                <div className={styles['radio']}>
                                    <Radio
                                        role={'sweetui-radio'}
                                        disabled={!periodicSyncStatus}
                                        value={SyncMode.USERS}
                                        checked={syncMode === SyncMode.USERS}
                                        onChange={() => this.changeSyncMode(SyncMode.USERS)}
                                    >
                                        {__('仅同步用户账号（不包括组织结构）')}
                                    </Radio>
                                </div>
                            </div>
                        </Form.Field>
                    </Form.Row>
                    <Form.Row role={'ui-form.row'}>
                        <Form.Label role={'ui-form.label'}></Form.Label>
                        <Form.Field role={'ui-form.field'}>
                            {
                                actionType === ActionType.Add && selection.id ?
                                    null :
                                    isSyncSettingEditStatus ? <div className={styles['btns']}>
                                        <Button role={'sweetui-button'} onClick={this.saveDomainConfig}>{__('保存')}</Button>
                                        <Button
                                            role={'sweetui-button'}
                                            className={styles['cancel']}
                                            onClick={this.cancelDomainConfig}
                                        >
                                            {__('取消')}
                                        </Button>
                                    </div> : null
                            }

                        </Form.Field>
                    </Form.Row>
                </Form>
                {
                    isShowsyncObjectDialog ?
                        <SyncObjectPicker
                            userid={this.userId}
                            zIndex={20}
                            selectType={[NodeType.ORGANIZATION, NodeType.DEPARTMENT]}
                            domainId={id}
                            data={!syncObject.length ? [] : syncObject[0].pathName ? syncObject : []}
                            convererOut={this.convererOutSyncObject}
                            onConfirm={this.updateSyncObject}
                            onCancel={() => this.setState({ isShowsyncObjectDialog: false })}
                        />
                        :
                        null
                }
                {
                    isShowSyncTargetDialog ?
                        <OrganizationPicker
                            title={__('选择同步目标')}
                            tip={__('注：修改同步位置后，需将原同步位置的域组织结构从用户组织结构中删除，否则无法同步到新的同步位置。')}
                            isSingleChoice={true}
                            userid={this.userId}
                            selectType={[NodeType.ORGANIZATION, NodeType.DEPARTMENT]}
                            data={syncTarget}
                            convererOut={this.convererOutSyncTarget}
                            onConfirm={this.updateSyncTarget}
                            onCancel={() => this.setState({ isShowSyncTargetDialog: false })}
                        /> : null
                }
            </div>
        )
    }
}