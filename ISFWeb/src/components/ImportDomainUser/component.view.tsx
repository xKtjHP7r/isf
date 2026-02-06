import * as React from 'react'
import classnames from 'classnames'
import { isBrowser, Browser } from '@/util/browser'
import { Text, Radio, ModalDialog2 as Dialog, SweetIcon, Button, Select } from '@/sweet-ui'
import { ProgressBar, Title, UIIcon } from '@/ui/ui.desktop'
import ValidityBox2 from '../ValidityBox2/component.view'
import DomainTree from '../DomainTree/component.view'
import { convertPath } from '../DomainTree/helper'
import ImportDomainUserBase, { ImportStyle, RenderType } from './component.base'
import * as deleteIcon from './assets/delete.png'
import styles from './styles.view';
import __ from './locale'

// 判断是否为Safari浏览器时，添加空的伪元素，解决Safari浏览器下显示双tooltip
const isSafari = isBrowser({ app: Browser.Safari });

export default class ImportDomainUser extends ImportDomainUserBase {

    render() {
        const { importStyle, userCover, userStatus, expireTime, renderType, progress, selected, csfLevel, csfOptions } = this.state;

        switch (renderType) {
            case RenderType.View:
                return (
                    <div>
                        <Dialog
                            title={__('导入域用户组织')}
                            onClose={this.props.onRequestCancel}
                            icons={[{
                                icon: <SweetIcon name="x" size={16} />,
                                onClick: this.props.onRequestCancel,
                            }]}
                            buttons={[{
                                text: __('导入'),
                                theme: 'oem',
                                disabled: selected.length === 0,
                                onClick: this.confirmImport,
                            },
                            {
                                text: __('取消'),
                                theme: 'regular',
                                onClick: this.props.onRequestCancel,
                            }]}
                        >
                            <div className={styles['container']}>
                                <div className={styles['left']}>
                                    <Text>
                                        {__('请选择您要导入的域用户和部门：')}
                                    </Text>
                                    <div className={styles['selected-header']}>
                                        {__('已选：')}
                                        <Button
                                            theme={'text'}
                                            onClick={this.clearSelected}
                                            className={styles['clear-btn']}
                                            disabled={selected.length === 0} >
                                            {__('清空')}
                                        </Button>
                                    </div>
                                    <div className={styles['list']}>
                                        <div className={styles['list-main']}>
                                            <DomainTree
                                                ref={(ref) => this.ref = ref}
                                                selection={selected}
                                                onSelectionChange={this.handleSelect}
                                                doRedirectDomain={this.props.doRedirectDomain}
                                            />
                                        </div>
                                    </div>
                                    <div className={styles['org-picker-arrow']}>
                                        <UIIcon
                                            size={28}
                                            code={'\uf0f5'}
                                            color={'#757575'}
                                            onClick={this.addTreeData}
                                        />
                                    </div>
                                    <div className={styles['org-picker-selections']}>
                                        <ul>
                                            {
                                                selected.map((sharer, index) => (
                                                    <li
                                                        key={index}
                                                        style={{ position: 'relative' }}
                                                        className={styles['selection']}
                                                    >
                                                        <Title content={sharer.ipAddress ? sharer.name : convertPath(sharer.pathName || sharer.dnPath)}>
                                                            <div className={classnames(
                                                                styles['dep-name'],
                                                                {
                                                                    [styles['safari']]: isSafari,
                                                                },
                                                            )}>
                                                                <span>{sharer.name || sharer.displayName}</span>
                                                            </div>
                                                        </Title>
                                                        <UIIcon
                                                            className={styles['icon-del']}
                                                            size={13}
                                                            code={'\uf014'}
                                                            fallback={deleteIcon}
                                                            onClick={() => { this.deleteSelected(sharer) }}
                                                        />
                                                    </li>
                                                ))
                                            }
                                        </ul>
                                    </div>
                                </div>
                                <div className={styles['right']}>
                                    <div className={styles['option']}>
                                        <Text textStyle={{ fontWeight: 'bold' }}>{__('请设置导入的方式：')}</Text>
                                        <div>
                                            <div className={styles['line']}>
                                                <Radio
                                                    checked={importStyle === ImportStyle.All}
                                                    value={ImportStyle.All}
                                                    onChange={({ detail: { value } }) => this.setState({ importStyle: value })}
                                                >
                                                    {__('导入选中的对象及其成员（包括上层的组织结构）')}
                                                </Radio>
                                            </div>
                                            <div className={styles['line']}>
                                                <Radio
                                                    checked={importStyle === ImportStyle.SelectedDepAndUser}
                                                    value={ImportStyle.SelectedDepAndUser}
                                                    onChange={({ detail: { value } }) => this.setState({ importStyle: value })}
                                                >
                                                    {__('导入选中的对象及其成员（不包括上层的组织结构）')}
                                                </Radio>
                                            </div>
                                            <div className={styles['line']}>
                                                <Radio
                                                    checked={importStyle === ImportStyle.Users}
                                                    value={ImportStyle.Users}
                                                    onChange={({ detail: { value } }) => this.setState({ importStyle: value })}
                                                >
                                                    {__('仅导入用户账号（不包括组织结构）')}
                                                </Radio>
                                            </div>
                                        </div>
                                    </div>
                                    <div className={styles['option']}>
                                        <Text textStyle={{ fontWeight: 'bold' }}>{__('在导入过程中，如果发现当前系统已存在同名的用户：')}</Text>
                                        <div>
                                            <div className={styles['line']}>
                                                <Radio
                                                    checked={userCover}
                                                    value={true}
                                                    onChange={() => this.setState({ userCover: true })}
                                                >
                                                    {__('覆盖同名用户')}
                                                </Radio>
                                            </div>
                                            <div className={styles['line']}>
                                                <Radio
                                                    checked={!userCover}
                                                    value={false}
                                                    onChange={() => this.setState({ userCover: false })}
                                                >
                                                    {__('跳过同名用户')}
                                                </Radio>
                                            </div>
                                        </div>
                                    </div>
                                    <div className={styles['option']}>
                                        <Text textStyle={{ fontWeight: 'bold' }}>{__('对于每一个导入的新用户：')}</Text>
                                        <div className={styles['line']}>
                                            <Text inline={true}>{__('用户密级：')}</Text>
                                            <span className={styles['tip']}>
                                                <Select
                                                    width={200}
                                                    value={csfLevel}
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
                                            </span>
                                        </div>
                                        <div className={styles['line']}>
                                            <Text inline={true}>{__('用户有效期限：')}</Text>
                                            <span className={styles['tip']}>
                                                <ValidityBox2
                                                    width={200}
                                                    value={expireTime}
                                                    allowPermanent={true}
                                                    onChange={this.changeExpireTime}
                                                />
                                            </span>
                                        </div>
                                        <div className={styles['line']}>
                                            <Text inline={true}>{__('用户状态默认为：')}</Text>
                                            <span className={styles['tip']}>
                                                <Radio
                                                    checked={userStatus}
                                                    value={true}
                                                    onChange={() => this.setState({ userStatus: true })}
                                                >
                                                    {__('启用')}
                                                </Radio>
                                            </span>
                                            <Radio
                                                checked={!userStatus}
                                                value={false}
                                                onChange={() => this.setState({ userStatus: false })}
                                            >
                                                {__('禁用')}
                                            </Radio>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </Dialog>
                    </div >
                )

            case RenderType.Progress:
                return (
                    <div>
                        <Dialog
                            title={__('正在导入')}
                            buttons={[
                                {
                                    text: __('关闭'),
                                    theme: 'oem',
                                    onClick: this.props.onRequestCancel,
                                },
                            ]}
                        >
                            <div className={styles['content']}>
                                <div className={styles['status']}>
                                    <Text inline={true}>{__('执行状态：')}</Text>
                                    <Text inline={true}>{progress === 1 ? __('导入完成') : __('正在导入')}</Text>
                                </div>
                                <ProgressBar
                                    value={progress}
                                    width={350}
                                    height={20}
                                    progressBackground={'#9abbef'}
                                />
                            </div>
                        </Dialog>
                    </div>
                )

            default:
                return null
        }
    }
}