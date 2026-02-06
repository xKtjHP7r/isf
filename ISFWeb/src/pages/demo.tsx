import React, {useRef, useState} from 'react'
import intl from 'react-intl-universal'
import { bootstrap, getMount, unmount } from '@/core/lifecycle'
import { Button } from "antd"
import { apis, components } from '@dip/components/dist/dip-components.min.js';
import { TextArea } from '@/sweet-ui';

const PermissionMgnt = () => {
    const accessPickerContainerRef = useRef<HTMLDivElement>(null)
    const batchAuthContainerRef = useRef<HTMLDivElement>(null)
    const defaultValue = 
        {
            resource: { id: "agent", name: "人力资源域", type: "agent", ancestors: []},
            scope: 'instance',
            pickerParams: {
                isAdmin: true,
                tabs: ['organization', 'group', 'app', 'role'],
                range: ['user', 'department', 'group', 'app', 'role'],
                role:'super_admin',
            }
        }

    const defaultBatchAuthValue = 
        {
            resources: [{ id: 'q1', name: '业务知识网络', type: 'agent', ancestors: []}, {id: 'q2', name: '智能体', type: 'agent', ancestors: []}],
            pickerParams: {
                isAdmin: true,
                tabs: ['organization', 'group', 'app', 'role'],
                range: ['user', 'department', 'group', 'app', 'role'],
                role:'super_admin',
            }
        }
    
    const [value, setValue] = useState(JSON.stringify(defaultValue))
    const [batchAuthValue, setBatchAuthValue] = useState(JSON.stringify(defaultBatchAuthValue))

    const showPerm = () => {
        const unmount = apis.mountComponent(
            components.PermConfig,
            {
                ...JSON.parse(value),
                onCancel: () => {
                    unmount();
                },
            },
            accessPickerContainerRef.current
        );
    }

    const showBatchAuth = () => {
        const unmount = apis.mountComponent(
            components.Authorization,
            {
                ...JSON.parse(batchAuthValue),
                onCancel: () => {
                    unmount();
                },
            },
            batchAuthContainerRef.current
        );
    }

    return (
        <div >
            <div style={{ marginBottom: 20 }}>
                <h1>权限配置组件测试</h1>
                <div>1.填写组件参数</div>
                <TextArea value={value} width={400} height={300} onValueChange={({detail}) => {
                    console.info({value: detail})
                    setValue(detail)
                }}/>
                <Button type='primary' style={{marginTop: 10}} onClick={showPerm}>{intl.get('ok')}</Button>
                <div ref={accessPickerContainerRef}></div>
            </div>
            <div>
                <h1>批量授权组件测试</h1>
                <div>1.填写组件参数</div>
                <TextArea value={batchAuthValue} width={400} height={300} onValueChange={({detail}) => {
                    console.info({value: detail})
                    setBatchAuthValue(detail)
                }}/>
                <Button type='primary' style={{marginTop: 10}} onClick={showBatchAuth}>{intl.get('ok')}</Button>
                <div ref={batchAuthContainerRef}></div>
            </div>
        </div>
    )
}

const mount = getMount(<PermissionMgnt />)

if (!window.__POWERED_BY_QIANKUN__) {
    mount({
        getToken: () => 'ory_at_meG9MJpP8eDdoD1-1HmnddAFmvK1EHLC2eASKasgnK4.dWxoWBreEnmAcmBwuby3AepV1TGAwMOYRfWTkAxm9K0',
        protocol: 'https:',
        host: location.hostname,
        port: '443',
        prefix: '',
        lang: 'zh-cn',
    })
}

export {
    bootstrap,
    mount,
    unmount,
}