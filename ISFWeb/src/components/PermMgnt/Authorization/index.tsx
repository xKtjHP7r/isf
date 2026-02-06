import React, { useEffect, useState, useContext, useRef } from "react"
import intl from "react-intl-universal";
import { getRoleInfo, getAccessorPolicy, getResourceInstance, addPolicyConfig, updatePolicyConfig, deletePolicyConfig } from "@/core/apis/console/authorization";
import { Button, Empty, Input, message, Modal, Popconfirm, Table } from "antd"
import { SearchOutlined } from "@ant-design/icons";
import AppConfigContext from "@/core/context/AppConfigContext";
import { trim, debounce } from "lodash"
import AuthIcon from "../../../icons/authorization.svg"
import DeleteIcon from "../../../icons/delete.svg"
import OperationPolicyIcon from "../../../icons/operation-policy.svg"
import EmptyIcon from "../../../icons/empty.png";
import loadFailedIcon from "../../../icons/loadFailed.png";
import SearchEmptyIcon from "../../../icons/searchEmpty.png";
import { foreverExpire, formatExpires, formatPerm, formatRequest } from "../util";
import { PermConfig } from "../PermConfig";
import { PermOperationEnum } from "../types";
import styles from "./styles.css";
import { defaultModalParams } from "@/util/modal";
import dayjs from "dayjs";
import { AuthorizationErrorCodeEnum, ResourceType } from "@/core/apis/console/authorization/type";

const { info } = Modal

const limit = 50;
export const Authorization = ({curRole, topMargin, updateRoleList}) => {
    const { oemColor } = useContext(AppConfigContext);
    const [roleInfo, setRoleInfo] = useState(null)
    const [types, setTypes] = useState([])
    const [curType, setCurType]= useState(null)
    const [hoverIndex, setHoverIndex] = useState(0)
    const [resourceSearchKey, setResourceSearchKey] = useState("")
    const [curResourceTypeInfo, setResourceTypeInfo] = useState(null)
    const [allPermAndExpires, setAllPermAndExpires] = useState([])
    const [isLoading, setIsLoading] = useState(false);  
    const [isError, setIsError] = useState(false);
    const [data, setData] = useState([])
    const [total, setTotal] = useState(0)
    const [curPage, setPage] = useState(1)
    const [curPageSize, setPageSize] = useState<number>(limit)
    const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([]);
    const [selections, setSelections] = useState([]);
    const [expandedRowKeys, setExpandedRowKeys] = useState([]);
    const [hoverId, setHoverId] = useState("")
    const [permInfo, setPermInfo] = useState(null)
    const [permTip, setPermTip] = useState("")
    const [expireTip, setExpireTip] = useState("")
    const [operationType, setOperationType] = useState(PermOperationEnum.SetAllResource)
    const [showAuthorization, setShowAuthorization]  = useState(false)
    const [curInfo, setCurInfo] = useState([])
    const [curConfig, setCurConfig] = useState(null)
    const setPermPolicyRef = useRef<HTMLDivElement>(null)
    const [resourceSearchValue, setResourceSearchValue] = useState("")
    // 用于跟踪请求序列号，确保只有最新的请求结果会被处理
    const requestIdRef = useRef(0)

    const handleError = (error) => {
        if(error?.code === AuthorizationErrorCodeEnum.RoleNotFound) {
            info({ 
                ...defaultModalParams, 
                closable: false,
                content: intl.get("role.not.exist"), 
                getContainer: document.getElementById('isf-web-plugins'), 
                onOk: () => {
                    updateRoleList?.()
                }
            })
        }else {
            const msg = error?.description || ""
            msg && info({ ...defaultModalParams, content: msg, getContainer: document.getElementById('isf-web-plugins')})
        }
    }

    // 获取当前角色信息
    const getCurrentRoleInfo = async() => {
        try {
            const data = await getRoleInfo({id: curRole?.id, resource_type_view_mode: 'hierarchy'})
            const { resource_type_scopes } = data
            setRoleInfo(data)
            setTypes(resource_type_scopes?.types)
            if(resource_type_scopes?.types.length) {
                setCurType(resource_type_scopes?.types[0])
                getResourceInfo(resource_type_scopes?.types[0])
                getPermAndExpiresConfig(resource_type_scopes?.types[0]?.id)
            }
        }catch(e) {
            handleError(e)
        }
    }

    const getTreeChildrenPlaceholder = (item, dataStruct) => {
        if (dataStruct !== "tree") {
            return undefined;
        }
        const hasChildren = typeof item?.has_children === "boolean" ? item.has_children : true;
        return hasChildren ? [] : undefined;
    };

    // 获取资源信息
    const getResourceInfo = async(resource, offset = 0, limit= 50, keyword = "" ) => {
        const currentRequestId = ++requestIdRef.current;
        setIsLoading(true)
        setIsError(false)
        
        try {
            const { instance_url, data_struct } = resource
            const { method, urlParams } = formatRequest(instance_url)
            setResourceTypeInfo({instance_url: instance_url, method, urlParams, data_struct})
            
            if(method && urlParams?.length) {
                const baseQuery = {
                    method, 
                    urlParams, 
                    offset, 
                    limit
                }
                const{ entries, total_count } = await getResourceInstance(trim(keyword) ? {...baseQuery, keyword: trim(keyword)} : baseQuery)
                
                if (currentRequestId !== requestIdRef.current) {
                    return;
                }
                
                let data = entries.map((cur) => {
                    return !trim(keyword) && (data_struct === "tree") ? {
                        ...cur,
                        key: cur.id,
                        children: getTreeChildrenPlaceholder(cur, data_struct),
                        ancestors: data_struct === "tree" ? [] : undefined
                    } : cur
                })
                
                if (!trim(keyword) && (data_struct === "tree") && offset + entries.length < total_count) {
                    data = [
                        ...data,
                        {
                            id: `loadmore-${resource.id}`,
                            key: `loadmore-${resource.id}`,
                            name: intl.get("load.more"),
                            isLoadMore: true
                        }
                    ]
                }
                setData(data)
                setTotal(total_count)
            } else {
                setData([])
                setTotal(0)
            }
        } catch(e) {
            if (currentRequestId !== requestIdRef.current) {
                return;
            }
            setIsError(true)
            setData([])
            setTotal(0)
            handleError(e)
        } finally {
            // 确保请求结束时设置loading为false
            if (currentRequestId === requestIdRef.current) {
                setIsLoading(false)
            }
        }
    }

    // 资源类型搜索处理
    const handleResourceTypeSearch = (e) => {
        const value = trim(e.target.value);
        setResourceSearchKey(value)
        setResourceSearchValue("")
        const types = roleInfo?.resource_type_scopes?.types
        if (value) {
            const filtered = types?.filter(({name}) => 
                (name).toLowerCase().includes(value.toLowerCase())
            );
            setTypes(filtered);
            if(filtered.length) {
                setCurType(filtered[0])
                setPage(1)
                getResourceInfo(filtered[0], 0, curPageSize)
            }
        } else {
            setTypes(types);
            if(types?.length) {
                setCurType(types[0])
                setPage(1)
                getResourceInfo(types[0], 0, curPageSize)
            }
        }
    }

    // 切换资源类型
    const changeSelect = (resource) => {
        if(resource?.id === curType?.id) return 
        setCurType(resource)
        getResourceInfo(resource, 0, curPageSize )
        setPage(1)
        setData([])
        setSelections([])
        setSelectedRowKeys([])
        setExpandedRowKeys([])
        getPermAndExpiresConfig(resource?.id)
        setResourceSearchValue("")
    }

    // 获取权限和有效期
    const getPermAndExpiresConfig = async (resource_type: string) => {
        if(resource_type) {
            try {
                const { entries } = await getAccessorPolicy({accessor_id: curRole?.id, accessor_type: "role", resource_type, limit: -1})
                setAllPermAndExpires(entries)
            }catch(e) {
                handleError(e)
            }
        }
    }

    useEffect(() => {
        if(curRole?.id) {
            getCurrentRoleInfo()
        }
    }, [])

    // 获取当前权限和有效期
    const getCurrentPermAndExpires = (item) => {
        const permAndExpires = allPermAndExpires?.find((cur) => cur.resource.id === item.id ||  cur.resource.type === item.id && cur.resource.id === "*")
        return permAndExpires
    }

    // 获取已设置策略的权限
    const getCurPermConfig = (cur) => { 
        const permPolicy = getCurrentPermAndExpires({ id: cur?.id})?.operation?.allow
            .filter(item => item.obligations && item.obligations?.length)
            .map(item => item.name);
        return permPolicy
    }

    // 获取权限配置组件信息
    const getValue = ({ operationType, operation, expires_at }) => {
        setPermInfo({ operationType, operation, expires_at })
    }

    // 获取策略id
    const getExistedPolicy = (curs) => {
        const curIds = curs.map(cur => cur.id).join(',');
        const matchedPermAndExpires = allPermAndExpires.filter(item => 
            curIds.split(',').includes(item.resource.id) || curIds === item.resource.type && item.resource.id === "*"
        );
        return matchedPermAndExpires
    }

    //权限配置校验
    const preCheck = () => {
        let result = true;
        if(!permInfo?.operation?.allow?.length && !permInfo?.operation?.deny?.length) {
            setPermTip(intl.get("select.perm.tip"))
            result = false
        }

        if (permInfo?.expires_at !== foreverExpire && dayjs(dayjs(permInfo?.expires_at).valueOf()).isBefore(dayjs())) {
            setExpireTip(intl.get("select.expires.tip"))
            result = false;
        }
        return result
    }

    // 设置或编辑权限
    const setPolicy = async (curs = []) => {
        if(!preCheck()) {
            return false
        } 
        try {
            const existPolicy = getExistedPolicy(curs)
            const notExistPolicy = curs.filter(cur => !existPolicy.some(item => item.resource.id === cur.id || item.resource.type === cur.id && item.resource.id === "*"))
            const policyIds = existPolicy.map(item => item.id);
            if(existPolicy.length) {
                const ids = policyIds.join(','); 
                const body = existPolicy.map(() => {
                    return {
                        operation: permInfo?.operation,
                        expires_at: permInfo?.expires_at,
                    }
                })
                await updatePolicyConfig({ids, body})
            }

            if(notExistPolicy.length || !curs.length) {
                const body = notExistPolicy.length ? notExistPolicy.map((item) => {
                    return {
                        accessor: { id: curRole?.id, type: "role" },
                        resource: { id: item.id, name: item.name, type: item.type, ancestors: item.ancestors},
                        operation: permInfo?.operation,
                        expires_at: permInfo?.expires_at,
                    }
                }) : [{
                    accessor: { id: curRole?.id, type: "role" },
                    resource: { id: "*", name: curType.name, type: curType.id, ancestors: curType.data_struct === 'tree' ? [] : undefined},
                    operation: permInfo?.operation,
                    expires_at: permInfo?.expires_at,
                }]
                await addPolicyConfig(body)
            }
            getPermAndExpiresConfig(curType?.id)
            message.success(intl.get("set.perm.success"))
            return true
        }catch(error) {
            handleError(error)
            return false
        }
    }

    // 删除权限
    const deletePolicy = async(curs) => {
        try {
            const existPolicy = getExistedPolicy(curs)
            if(!existPolicy.length) {
                message.info(intl.get("without.perm.cancel.tip"))
                return true
            }
            const ids = existPolicy.map(item => item.id).join(','); 
            await deletePolicyConfig(ids)
            message.success(intl.get("cancel.perm.success"))
            getPermAndExpiresConfig(curType?.id)
            return true
        }catch(error) {
            handleError(error)
            return false
        }
    }

    // 使用useRef保存防抖函数实例，避免每次渲染创建新的函数实例
    const debouncedSearch = useRef(
        debounce(({ value, type, pageSize }) => {
            getResourceInfo(type, 0, pageSize, value);
        }, 300)
    )

    const handleResourceSearch = (e, curType) => {
        setResourceSearchValue(e.target.value)
        setPage(1)
        debouncedSearch.current?.({ value: e.target.value, type: curType, pageSize: curPageSize })
    }

    // 表配置
    const columns = [
        {
            key: "name",
            dataIndex: "name",
            title: intl.get("resource.name"),
            width: "100%",
            render: (name, cur) => {
                if (cur.isLoadMore) {
                    return (
                        <a onClick={() => handleLoadMore(cur)}>
                            {cur.name}
                        </a>
                    );
                }
                
                const permAndExpires = getCurrentPermAndExpires({ id: cur?.id });
                const hasOperation = permAndExpires?.operation;
                const policyConfig = hasOperation ? getCurPermConfig(cur) : null;
                
                return (
                    <div className={styles["item"]}>
                        <div title={name} className={styles["name"]} >{name}</div>
                        <div className={styles["info"]}>
                            {hasOperation && (
                                <div className={styles["perm"]}>
                                    <div
                                        className={styles["perm-string"]}
                                        title={`${formatPerm({operation: permAndExpires?.operation}, curType?.operation?.instance)}(${formatExpires(permAndExpires?.expires_at)})`}
                                    >
                                        {`${formatPerm({operation: permAndExpires?.operation}, curType?.operation?.instance)}(${formatExpires(permAndExpires?.expires_at)})`}
                                    </div>
                                    {
                                        policyConfig?.length ? (
                                            <div 
                                                className={styles['policy-icon']} 
                                                title={intl.get("perm.policy.tip", {perm: policyConfig?.join("、")})}
                                            >
                                                <OperationPolicyIcon style={{ width: 14, height: 14, marginLeft: 2 }} />
                                            </div>
                                        ): null
                                    }
                                </div>
                            )}
                            {(selectedRowKeys.includes(cur.id) || hoverId === cur.id) && (
                                <div className={styles["operation"]}>
                                    {!hasOperation ? (
                                        <a onClick={() => {
                                            setCurInfo([cur])
                                            setCurConfig(null)
                                            setOperationType(PermOperationEnum.SetSpecifiedResource)
                                            setShowAuthorization(true)
                                        }}>
                                            {intl.get("set.perm")}
                                        </a>
                                    ) : (
                                        <div>
                                            <a onClick={() => {
                                                setCurInfo([cur])
                                                setCurConfig({operation: permAndExpires?.operation, expires_at: permAndExpires?.expires_at})
                                                setOperationType(PermOperationEnum.EditSpecifiedResource)
                                                setShowAuthorization(true)
                                            }}>
                                                {intl.get("edit")}
                                            </a>
                                            <Popconfirm
                                                placement="bottomRight"
                                                destroyOnHidden={true}
                                                title={intl.get("cancel.perm.tip")}
                                                getPopupContainer={() => document.getElementById("role-authorization")}
                                                onConfirm={async() => {
                                                    const result = await deletePolicy([cur])
                                                    if (!result) {
                                                        return Promise.reject();
                                                    }
                                                    return Promise.resolve();
                                                }}
                                            >
                                                <a className={styles["text"]}>
                                                    {intl.get("cancel.perm")}
                                                </a>
                                            </Popconfirm>
                                        </div>
                                    )}
                                </div>
                            )}
                        </div>
                    </div>
                );
            }
        }
    ]

    // 获取选中项
    const getSelections = (recordKeys: string[]) => {
        const flattenData = (items: ResourceType[]) => {
            let result: ResourceType[] = [];
            for (const item of items) {
                result.push(item);
                if (item.children && item.children.length) {
                    result = result.concat(flattenData(item.children));
                }
            }
            return result;
        };

        const flatData = flattenData(data);
        const selections = flatData.filter((item: ResourceType) => recordKeys.includes(item.id));
        setSelections(selections);
    };

    useEffect(() => {
        setSelectedRowKeys([]);
        setSelections([]);
    }, [resourceSearchValue, curPage, curPageSize]);
  
    // 行事件处理
    const onRow = (record) => ({
        onClick: () => {
            if (record.isLoadMore) {
                return;
            }
            const id = record.id || record.userId
            setSelectedRowKeys([id]);
            getSelections([id]);
        },
        onMouseEnter: () => {
            setHoverId(record.id)
        },
        onMouseLeave: () => {
            setHoverId("")
        }
    });

    const rowSelection = {
        selectedRowKeys,
        onChange: (selectedKeys: string[]) => {
            setSelectedRowKeys(selectedKeys);
            getSelections(selectedKeys);
        },
        getCheckboxProps: (record) => ({
            disabled: record.isLoadMore,
            key: record.id,
        }),
    };

    // 行展开处理
    const handleExpand = async(expanded, record) => {
        if (expanded) {
            // 先清除该节点原有的 children 数据
            setData(prevData => updateDataWithChildren(prevData, record.id, [], false));
            // 从展开列表中移除该节点，之后再重新添加，避免重复
            let filteredKeys = expandedRowKeys.filter(key => key !== record.key);
            // 定义递归函数获取点击节点的所有子节点 key
            const getChildrenKeys = (items: any[], keys: string[] = []) => {
                for (const item of items) {
                    keys.push(item.key);
                    if (item.children && item.children.length) {
                        getChildrenKeys(item.children, keys);
                    }
                }
                return keys;
            };

            // 查找点击节点
            const findNode = (dataList: ResourceType[]) => {
                for (const item of dataList) {
                    if (item.id === record.id) {
                        return item;
                    }
                    if (item.children && item.children.length) {
                        const found = findNode(item.children);
                        if (found) {
                            return found;
                        }
                    }
                }
                return null;
            };

            const targetNode = findNode(data);
            if (targetNode && targetNode.children) {
                const childrenKeys = getChildrenKeys(targetNode.children);
                // 移除点击节点子节点的展开状态
                filteredKeys = filteredKeys.filter(key => !childrenKeys.includes(key));
            }

            setExpandedRowKeys(filteredKeys);

            try {
                const { entries, total_count } = await getResourceInstance({ method: curResourceTypeInfo.method, urlParams: curResourceTypeInfo.urlParams, id: record.id, offset: 0, limit })
                const hasMore = entries.length < total_count;
                if (entries.length > 0) {
                    const keys = [...filteredKeys, record.key];
                    setExpandedRowKeys(keys);
                    setData(prevData => updateDataWithChildren(prevData, record.id, entries, hasMore));
                } else {
                    // 无数据时已在前面清除 children，这里只需保持展开状态处理
                    setExpandedRowKeys(filteredKeys);
                }
            } catch (e) {
                console.info('展开节点获取数据失败:', e);
            }
        } else {
            const keys = expandedRowKeys.filter(key => key !== record.key);
            setExpandedRowKeys(keys);
        }
    }

    // 定义加载更多的处理函数
    const handleLoadMore = async (record) => {
        const targetId = record.id.replace('loadmore-', '');

        // 第一层加载更多的情况
        if (record.key.startsWith('loadmore-') && !data.some(item => item.children?.some(child => child.key === record.key))) {
            try {
                const { method, urlParams, data_struct } = curResourceTypeInfo;
                const offset = data.filter(item => !item.isLoadMore).length; 
                const { entries, total_count } = await getResourceInstance({ method, urlParams, id: targetId, offset, limit })
                const hasMore = offset + entries.length < total_count;
                const newItems = entries.map(item => ({
                    ...item,
                    key: item.id,
                    children: getTreeChildrenPlaceholder(item, data_struct),
                    ancestors: data_struct === "tree" ? [] : undefined
                }));

                let updatedData = data.filter(item => !item.isLoadMore);
                updatedData = [...updatedData, ...newItems];

                if (hasMore) {
                    updatedData = [
                        ...updatedData,
                        {
                            id: `loadmore-${targetId}`,
                            key: `loadmore-${targetId}`,
                            name: intl.get("load.more"),
                            isLoadMore: true
                        }
                    ];
                }

                setData(updatedData);
                setTotal(total_count);
            } catch (e) {
                console.info('第一层加载更多失败:', e);
            }
            return;
        }

        const targetItem = findItemInData(data, targetId);
        if (!targetItem) return;

        // 假设从当前子项数量开始加载新数据
        const offset = targetItem.children ? targetItem.children.filter(child => !child.isLoadMore).length : 0; 
        try {
            const { method, urlParams, data_struct } = curResourceTypeInfo;
            
            const { entries, total_count } = await getResourceInstance({ method, urlParams, id: targetId, offset, limit })
            const hasMore = offset + entries.length < total_count;
            const newChildren = entries.map(child => ({
                ...child,
                key: child.id,
                children: getTreeChildrenPlaceholder(child, data_struct),
                ancestors: [...(targetItem.ancestors || []), {id: targetItem.id, name: targetItem.name, type: targetItem.type}]
            }));
            setData(prevData => {
                // 移除旧的加载更多项
                const updatedData = removeOldLoadMore(prevData, targetId); 
                return updateDataWithChildren(updatedData, targetId, newChildren, hasMore);
            });
        } catch (e) {
            console.info('加载更多失败:', e);
        }
    };

    // 在数据中查找指定 id 的项
    const findItemInData = (data, targetId) => {
        for (const item of data) {
            if (item.id === targetId) return item;
            if (item.children) {
                const found = findItemInData(item.children, targetId);
                if (found) return found;
            }
        }
        return null;
    };

    // 移除旧的加载更多项
    const removeOldLoadMore = (data, targetId) => {
        return data.map(item => {
            if (item.id === targetId && item.children) {
                return {
                    ...item,
                    children: item.children.filter(child => !child.isLoadMore)
                };
            }
            if (item.children) {
                return {
                    ...item,
                    children: removeOldLoadMore(item.children, targetId)
                };
            }
            return item;
        });
    };

    // 递归更新数据的函数
    const updateDataWithChildren = (prevData, targetId, newChildren, hasMore = false) => {
        return prevData.map(item => {
            if (item.id === targetId) {
                if (newChildren.length === 0) {
                    // 当 newChildren 为空时，删除 children 属性
                    const { children, ...rest } = item;
                    return rest;
                }
                // 获取原有子项，过滤掉加载更多项
                const originalChildren = item.children ? item.children.filter(child => !child.isLoadMore) : [];
                // 合并原有子项和新获取的子项
                const mergedChildren = [...originalChildren, ...newChildren];
                let processedChildren = mergedChildren.map(child => ({
                    ...child,
                    key: child.id,
                    children: getTreeChildrenPlaceholder(child, curResourceTypeInfo.data_struct),
                    ancestors: [...(item.ancestors || []), {id: item.id, name: item.name, type: item.type}]
                }));
                if (hasMore) {
                    processedChildren = [
                        ...processedChildren,
                        {
                            id: `loadmore-${targetId}`,
                            key: `loadmore-${targetId}`,
                            name: intl.get("load.more"),
                            isLoadMore: true
                        }
                    ];
                }
                return {
                    ...item,
                    children: processedChildren
                };
            } else if (item.children && item.children.length) {
                return {
                    ...item,
                    children: updateDataWithChildren(item.children, targetId, newChildren, hasMore)
                };
            }
            return item;
        });
    };

    return (
        <div className={styles["role-authorization"]} id={"role-authorization"}>
            <div className={styles["nav"]}>
                <Input 
                    className={styles["search"]}
                    placeholder={intl.get("search.name")}
                    prefix={<SearchOutlined />}
                    value={resourceSearchKey}
                    onChange={handleResourceTypeSearch}
                    allowClear
                />
                <div className={styles["resource"]}>
                    {
                        types.length ? 
                            types.map((cur, index) => {
                                return (
                                    <div 
                                        className={styles["item"]} 
                                        key={cur.id} 
                                        style={{
                                            backgroundColor:
                                            curType?.id === cur.id || index === hoverIndex
                                                ? oemColor.colorPrimaryBg
                                                : "transparent",
                                        }}
                                        title={cur.name}
                                        onClick={() => changeSelect(cur)}
                                        onMouseEnter={() => {
                                            setHoverIndex(index)
                                        }}
                                        onMouseLeave={() => {
                                            setHoverIndex(undefined)
                                        }}
                                    >
                                        {cur.name}
                                    </div>
                                )
                            }) 
                        :
                        <div className={styles["empty"]}>
                            <Empty image={resourceSearchKey ? SearchEmptyIcon : EmptyIcon} description={intl.get(resourceSearchKey ? "no.match.search.result" : "list.empty")}/>
                        </div>
                    }
                </div>
            </div>
            {
                types.length ?
                 <div className={styles["content"]}>
                    <div className={styles["top"]}>
                        <div className={styles["all-text"]}>{intl.get("all.resources")}</div>
                        {
                            getCurrentPermAndExpires({ id: curType?.id}) && (
                                <div className={styles["all-resource-config"]}>
                                    <div 
                                        className={styles["perm"]} 
                                        title={`${formatPerm({operation: getCurrentPermAndExpires({ id: curType?.id})?.operation}, curType?.operation?.type)}(${formatExpires(getCurrentPermAndExpires({ id: curType?.id})?.expires_at)})`}
                                    >
                                        {`${formatPerm({operation: getCurrentPermAndExpires({ id: curType?.id})?.operation}, curType?.operation?.type) }(${formatExpires(getCurrentPermAndExpires({ id: curType?.id})?.expires_at)})`}
                                    </div>
                                    {
                                        getCurPermConfig(curType)?.length ? 
                                            <div className={styles['policy-icon']} title={intl.get("perm.policy.tip", {perm: getCurPermConfig(curType).join("、")})}>
                                                <OperationPolicyIcon style={{ width: 14, height: 14 }} />
                                            </div>
                                            : null
                                    }
                                </div>
                            )
                        }
                        {
                            !getCurrentPermAndExpires({ id: curType?.id}) &&
                            <a 
                                className={styles["set-text"]}
                                onClick={() => {
                                    setCurInfo()
                                    setCurConfig(null)
                                    setOperationType(PermOperationEnum.SetAllResource)
                                    setShowAuthorization(true)
                                }}
                            >
                                {intl.get("set.perm")}
                            </a>
                        }
                        {
                            getCurrentPermAndExpires({ id: curType?.id}) &&
                                <div className={styles["operation"]}>
                                    <a 
                                        onClick={() => {
                                            setCurInfo([{id: curType.id}])
                                            setCurConfig(getCurrentPermAndExpires({ id: curType?.id}))
                                            setOperationType(PermOperationEnum.EditAllResource)
                                            setShowAuthorization(true)
                                        }}
                                        className={styles["text"]}
                                    >
                                        {intl.get("edit")}
                                    </a>
                                    <Popconfirm
                                        placement="bottomRight"
                                        destroyOnHidden={true}
                                        title={intl.get("cancel.perm.tip")}
                                        getPopupContainer={trigger => trigger.parentNode as HTMLDivElement}
                                        onConfirm={async() =>{
                                            const result = await deletePolicy([{id: curType.id}])
                                            if (!result) {
                                                return Promise.reject(); 
                                            }
                                            return Promise.resolve();
                                        }}
                                    >
                                        <a className={styles["text"]}>{intl.get("cancel.perm")}</a>
                                    </Popconfirm>
                                </div>
                        }
                    </div>
                    <div className={styles["main"]}>
                        <div className={styles["header"]} style={{ display: curResourceTypeInfo?.instance_url  ? "flex": "inline-flex" }}>
                            <div className={styles["sepcified-text"]} style={{ width: curResourceTypeInfo?.instance_url ? "160px": "fit-content" }}>
                                {intl.get("specified.resources")}
                            </div>
                            {
                                curResourceTypeInfo?.instance_url ? 
                                    <div className={styles["header-right"]}>
                                        <Button
                                            icon={<AuthIcon style={{width: "14px", height: "14px"}}/>}
                                            className={styles["btn"]}
                                            disabled={!selections.length}
                                            onClick={() => {
                                                if (selections.length) {
                                                    const firstType = selections?.[0]?.type;
                                                    const sameType = selections.every(item => item.type === firstType);
                                                    if (!sameType) {
                                                        message.info(intl.get("resource.type.must.be.same"));
                                                        return;
                                                    }
                                                    setCurInfo(selections)
                                                    setCurConfig(selections.length === 1 ? {operation: getCurrentPermAndExpires({ id: selections[0]?.id})?.operation, expires_at: getCurrentPermAndExpires({ id: selections[0]?.id})?.expires_at } : null)
                                                    setOperationType(PermOperationEnum.SetSpecifiedResource)
                                                    setShowAuthorization(true)
                                                }
                                            }}
                                        >
                                            {intl.get("set.perm")}
                                        </Button>
                                        <Popconfirm
                                            placement="bottomRight"
                                            destroyOnHidden={true}
                                            title={selections.length > 1 ? intl.get("cancel.batch.perm.tip", {count: selections.length}): intl.get("cancel.perm.tip")}
                                            onConfirm={async() => {
                                                const result = await deletePolicy(selections)
                                                if (!result) {
                                                    return Promise.reject(); 
                                                }
                                                return Promise.resolve();
                                            }}
                                            getPopupContainer={trigger => trigger.parentNode as HTMLDivElement}
                                        >
                                            <Button
                                                icon={<DeleteIcon style={{width: "14px", height: "14px"}}/>}
                                                className={styles["btn"]}
                                                disabled={!selections.length}
                                            >
                                                {intl.get("cancel.perm")}
                                            </Button>
                                        </Popconfirm>
                                        {
                                            curResourceTypeInfo?.instance_url?.includes("keyword") ?
                                                <Input 
                                                    allowClear
                                                    placeholder={intl.get("search.name")}
                                                    className={styles["resource-search"]}
                                                    prefix={<SearchOutlined />}
                                                    value={resourceSearchValue}
                                                    onChange={(e) => handleResourceSearch(e, curType)}
                                                /> : null
                                        }
                                    </div> : 
                                    <div className={styles["no-instance-tip"]}>{intl.get("no.instance.tip")}</div>
                            }
                        </div>
                        {
                            curResourceTypeInfo?.instance_url ?
                                <div className={styles["table"]}>
                                    <Table
                                        size="small"
                                        tableLayout="fixed"
                                        loading={isLoading}
                                        columns={columns}
                                        dataSource={data}
                                        rowSelection={rowSelection}
                                        onRow={onRow}
                                        rowKey={record => record.id}
                                        scroll={{x: "100%", y: curType?.data_struct === "string" ? `calc(100vh - ${topMargin} - 40px)`: `calc(100vh - ${topMargin})`}}
                                        pagination={
                                            curType?.data_struct === "string" ? {
                                                current: curPage,
                                                pageSize: curPageSize,
                                                total,
                                                showSizeChanger: true,
                                                showQuickJumper: false,
                                                showTotal: (total) => {
                                                    return intl.get("list.total.tip", { total });
                                                },
                                                onChange: (page, pageSize) => {
                                                    setPageSize(pageSize);
                                                    setPage(pageSize !== curPageSize ? 1 : page)
                                                    getResourceInfo(curType, ((pageSize !== curPageSize ? 1 : page) - 1) * pageSize, pageSize, resourceSearchValue)
                                                },
                                            } : false
                                        }
                                        expandable={{
                                            expandedRowKeys,
                                            onExpand: handleExpand,
                                        }}
                                        locale={{
                                            emptyText: (
                                                <div className={styles["empty"]}>
                                                    <img
                                                        src={isError ? loadFailedIcon : resourceSearchValue ? SearchEmptyIcon : EmptyIcon}
                                                        alt=""
                                                        width={128}
                                                        height={128}
                                                    />
                                                    <span>{intl.get(isError ? "loadFailed" : resourceSearchValue ? "no.search.result" : "no.instance.tip")}</span>
                                                </div>
                                            ),
                                        }}
                                    />
                                </div> : null
                        }
                    </div>
                </div> : null
            }
            {
                showAuthorization && (
                    <Modal
                        centered
                        maskClosable={false}
                        open={true}
                        width={540}
                        title={intl.get("perm.config")}
                        onCancel={() => setShowAuthorization(false)}
                        footer={[
                            <Button key="confirm" type="primary" onClick={async() => {
                                const result = await setPolicy(curInfo)
                                if (result) {
                                    setShowAuthorization(false)
                                }
                            }}>
                                {intl.get('ok')}
                            </Button>,
                            <Button key="cancel" onClick={() => setShowAuthorization(false)}>
                                {intl.get('cancel')}
                            </Button>,
                        ]}
                        getContainer={document.getElementById("role-authorization") as HTMLElement}
                    >
                        <PermConfig 
                            setPermPolicyRef={setPermPolicyRef}
                            resourceType={curType}
                            operationType={operationType} 
                            operationConfig={operationType === PermOperationEnum.SetAllResource || operationType === PermOperationEnum.EditAllResource ? curType?.operation?.type : curType?.operation?.instance}
                            getValue={getValue}
                            permTip={permTip}
                            setPermTip={setPermTip}
                            expireTip={expireTip}
                            setExpireTip={setExpireTip}
                            curConfig={curConfig}
                        />
                    </Modal>
                    
                )
            }
            <div ref={setPermPolicyRef}></div>
        </div>
    )
}