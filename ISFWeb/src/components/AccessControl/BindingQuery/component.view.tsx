import * as React from 'react'
import * as classnames from 'classnames'
import BindingQueryBase, { SearchField, deviceType, PageSize } from './component.base'
import ToolBar from '@/ui/ToolBar/ui.desktop'
import AutoComplete from '@/ui/AutoComplete/ui.desktop'
import Title from '@/ui/Title/ui.desktop'
import Text from '@/ui/Text/ui.desktop'
import { isBrowser, Browser } from '@/util/browser';
import AutoCompleteList from '@/ui/AutoCompleteList/ui.desktop'
import { DataGrid, Select } from '@/sweet-ui';
import FlexBox from '@/ui/FlexBox/ui.desktop'
import ListTipComponent from '../../ListTipComponent/component.view';
import { ListTipStatus } from '../../ListTipComponent/helper';
import { NetType } from '../VisitorNetBind/helper';
import __ from './locale'
import styles from './styles.view.css'

// 判断是否为Safari浏览器，是时，添加空的伪元素，解决Safari浏览器下显示双tooltip
const isSafari = isBrowser({ app: Browser.Safari });

export default class BindingQuery extends BindingQueryBase {

    // 部门用户路径信息
    getDepName(value) {
        if (value.parentPath) {
            return (
                <div className={styles['title']}>
                    <div>
                        {`${value.departmentName}(${__('部门')})`}
                    </div>
                    <div>
                        {`${value.parentPath}`}
                    </div>
                </div>
            )
        } else if (value.departmentPaths) {
            return (
                <div className={styles['title']}>
                    <div>
                        {`${value.displayName}(${value.loginName})`}
                    </div>
                    <div>
                        {`${value.departmentPaths[0]}`}
                    </div>
                </div>
            )
        } else {
            return value.departmentName
        }
    }

    render() {
        let { visitorNetInfos, deviceInfos, deviceIdUsers, searchResults, searchField, searchKey, listTipStatus, deviceListStatus } = this.state
        return (
            <div className={styles['container']}>
                <div className={styles['head-wrapper']}>
                    <div>
                        <label className={styles['tip']}>{__('输入访问者或设备识别码，查询最终绑定信息')}</label>
                    </div>
                    <div className={styles['info-header']}>
                        <ToolBar>
                            <FlexBox>
                                <FlexBox.Item width={230}>
                                    <Select onChange={this.handleChangeSearchField.bind(this)} value={searchField}>
                                        <Select.Option selected={searchField === SearchField.User} value={SearchField.User}>
                                            {__('访问者')}
                                        </Select.Option>
                                        <Select.Option selected={searchField === SearchField.DeviceId} value={SearchField.DeviceId}>
                                            {__('设备识别码')}
                                        </Select.Option>
                                    </Select>
                                </FlexBox.Item>
                                <FlexBox.Item>
                                    <AutoComplete
                                        className={styles['search-input']}
                                        ref="auto-complete"
                                        value={searchKey}
                                        width="100%"
                                        placeholder={__('请输入访问者或设备识别码')}
                                        missingMessage={__('没有找到符合条件的结果')}
                                        onChange={this.handleSearchChange.bind(this)}
                                        loader={this.loaders[searchField]}
                                        onLoad={this.handleSearchLoaded.bind(this)}
                                        onEnter={this.handleEnter.bind(this)}
                                        lazyLoader={
                                            {
                                                limit: PageSize,
                                                trigger: 0.75,
                                                onChange: this.lazyLoade,
                                            }
                                        }
                                    >
                                        {
                                            searchResults && searchResults.length ? (
                                                <AutoCompleteList>
                                                    {
                                                        searchResults.map((item, index) => {
                                                            switch (searchField) {
                                                                case SearchField.User:
                                                                    return (
                                                                        <AutoCompleteList.Item key={index}>
                                                                            <span href="javascript:void(0);" className={styles['search-item']} onClick={() => this.queryUser(item)}>

                                                                                <Title content={this.getDepName(item)}>
                                                                                    <div className={classnames(
                                                                                        styles['allname'],
                                                                                        {
                                                                                            [styles['safari']]: isSafari,
                                                                                        },
                                                                                    )}>
                                                                                        <span className={styles['dename']}>{item.name || item.displayName || item.departmentName}</span>
                                                                                        <div className={styles['depaths']}>
                                                                                            {item.parentPath ? item.parentPath : item.departmentPaths ? item.departmentPaths[0] : ''}
                                                                                        </div>
                                                                                    </div>
                                                                                </Title>
                                                                            </span>
                                                                        </AutoCompleteList.Item>
                                                                    )
                                                                case SearchField.DeviceId:
                                                                    return (
                                                                        <AutoCompleteList.Item key={index}>
                                                                            <span href="javascript:void(0);" className={styles['search-item']} onClick={() => this.renderDeviceId(item)}>
                                                                                {item}
                                                                            </span>
                                                                        </AutoCompleteList.Item>
                                                                    )
                                                            }
                                                        })
                                                    }
                                                </AutoCompleteList>
                                            ) : null
                                        }
                                    </AutoComplete>
                                </FlexBox.Item>
                            </FlexBox>
                        </ToolBar>
                    </div >
                </div>
                {
                    searchField === SearchField.DeviceId ? (
                        <div className={classnames(styles['data-wrapper'], styles['udid-wrapper'])}>
                            <DataGrid
                                data={deviceIdUsers}
                                height="100%"
                                showBorder={true}
                                start={deviceIdUsers.length}
                                limit={this.defaultLimit}
                                refreshing={listTipStatus !== ListTipStatus.None}
                                RefreshingComponent={
                                    <ListTipComponent
                                        listTipStatus={listTipStatus}
                                    />
                                }
                                onRequestLazyLoad={this.lazyLoadDeviceUdidUsers.bind(this)}
                                columns={
                                    [
                                        {
                                            title: __('用户'),
                                            key: 'name',
                                            renderCell: (name, record) => (
                                                <Text className={styles['udid-user']}>{record}</Text>
                                            ),
                                        },
                                    ]
                                }
                            />
                        </div>
                    ) : (
                        <div className={styles['data-wrapper']}>
                            <div className={classnames(styles['fl'], styles['net-container'])}>
                                <div className={classnames(styles['net-wrapper'], styles['user-net-wrapper'])}>
                                    <DataGrid
                                        data={visitorNetInfos}
                                        height="100%"
                                        showBorder={true}
                                        limit={PageSize}
                                        refreshing={listTipStatus !== ListTipStatus.None}
                                        RefreshingComponent={
                                            <ListTipComponent
                                                listTipStatus={listTipStatus}
                                            />
                                        }
                                        onRequestLazyLoad={this.handleLazyLoadNetList}
                                        columns={
                                            [
                                                {
                                                    title: __('网段名称'),
                                                    key: 'name',
                                                    renderCell: (name, record) => (
                                                        <Text>{record.id === 'public-net' ? __('所有外网网段') : (name ? name : '---')}</Text>
                                                    ),
                                                },
                                                {
                                                    title: __('网段'),
                                                    key: 'net',
                                                    renderCell: (net, record) => (
                                                        <Text>
                                                            {
                                                                record.id === 'public-net' ? null
                                                                    : (record.netType === NetType.Range ?
                                                                        (record.originIP + '-' + record.endIP)
                                                                        : (record.ip + '/' + record.mask)
                                                                    )

                                                            }
                                                        </Text>
                                                    ),
                                                },
                                            ]
                                        }
                                    />
                                </div>
                            </div>
                            <div className={classnames(styles['fr'], styles['device-container'])}>
                                <div className={styles['device-wrapper']}>
                                    <DataGrid
                                        data={deviceInfos}
                                        height="100%"
                                        showBorder={true}
                                        refreshing={deviceListStatus !== ListTipStatus.None}
                                        RefreshingComponent={
                                            <ListTipComponent
                                                listTipStatus={deviceListStatus}
                                            />
                                        }
                                        limit={this.defaultLimit}
                                        onRequestLazyLoad={this.handleRequestLoadDevice}
                                        columns={
                                            [
                                                {
                                                    title: __('设备识别码'),
                                                    key: 'udid',
                                                    renderCell: (value, { baseInfo: { udid } }) => (
                                                        <Text>{udid}</Text>
                                                    ),
                                                },
                                                {
                                                    title: __('设备类型'),
                                                    key: 'osType',
                                                    renderCell: (value, { baseInfo: { osType } }) => (
                                                        <Text>{deviceType[osType]}</Text>
                                                    ),
                                                },
                                                {
                                                    title: __('状态'),
                                                    key: 'bindFlag',
                                                    renderCell: (value, { bindFlag }) => (
                                                        <Text>{bindFlag ? __('绑定') : __('解绑')}</Text>
                                                    ),
                                                },
                                            ]
                                        }
                                    />
                                </div>
                            </div>
                        </div>
                    )
                }
            </div>
        )
    }
}