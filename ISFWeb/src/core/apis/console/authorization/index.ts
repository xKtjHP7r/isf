import { consolehttp } from "../../../openapiconsole";
import { addPolicyConfigBodyType, updatePolicyConfigBodyType } from "./type";

/**
 * 获取角色
 */
export const getRoles = ({
    offset,
    limit,
    sort,
    direction,
    keyword,
    source
}: {
  offset: number;
  limit: number;
  sort?: string;
  direction?: string;
  keyword?: string;
  source?: string[];
}) => {
    return consolehttp("get", ["authorization", "v1", "roles"], null, {
        offset,
        limit,
        sort,
        direction,
        keyword,
        source
    });
};

/**
 * 
 * @param param0 新建角色
 * @returns 
 */
export const createRoles = (roleInfo) => {
    return consolehttp("post", ["authorization", "v1", "roles"], roleInfo, null) 
}

/**
 * 编辑角色信息
 */
export const EditRole = ({id, fields = 'name,description,resource_type_scope', name, description, resource_type_scope}) => {
    return consolehttp("put", ["authorization", "v1", "roles", id, fields], {name, description, resource_type_scope}, null) 
}

/**
 * 删除角色
 */
export const deleteRole = (id) => {
    return consolehttp("delete", ["authorization", "v1", "roles", id], null, null)
}

/**
 * 获取成员信息
 */
export const getMembers = ({
    id,
    offset,
    limit,
    sort,
    direction,
    type,
    keyword,
}: {
  id: string;
  offset: number;
  limit: number;
  sort?: string;
  direction?: string;
  keyword?: string;
  type?: string[];
}) => {
    return consolehttp("get", ["authorization", "v1", "role-members", id], null, {
        offset,
        limit,
        sort,
        direction,
        keyword,
        type
    });
};

/**
 * 添加/删除成员
 */
export const updateMembers = ({
    method,
    id,
    members,
}: {
  method: "POST" | "DELETE";
  id: string;
  members: {id: string; type: string}[];
}) => {
    return consolehttp(
        "post",
        ["authorization", "v1", "role-members", id],
        { method, members },
        null
    );
};

/**
 * 获取所有资源类型
 */
export const getResources = ({
    offset,
    limit
}:{
    offset: number;
    limit: number;
}) => {
    return consolehttp(
        "get",
        ["authorization", "v1", "resource_type"],
        null,
        {offset, limit}
    )
}

/**
 * 获取角色信息
 */
export const getRoleInfo = ({id, resource_type_view_mode = 'flat' }) => {
    return consolehttp("get", ["authorization", "v1", "roles", id], null, { resource_type_view_mode });
}

/**
 * 获取角色权限和有效期配置
 */
export const getAccessorPolicy = (data:{ accessor_id: string; accessor_type: string; resource_type?: string; resource_id?: string; offset?: number; limit?: number; include?: string[]}) => {
    return consolehttp("get", ["authorization", "v1", "accessor-policy"], null, data); 
}

/**
 * 获取资源类型信息
 */
export const getResourceInfo = (id) => {
    return consolehttp("get", ["authorization", "v1", "resource_type", id], null, null);
}

/**
 * 获取资源实例
 */
export const getResourceInstance = ({method, urlParams, offset, limit, id, keyword}:{method: "post" | "get" | "put" | "patch" | "delete"; urlParams: string[]; offset: number; limit: number; id?: string; keyword?: string}) => {
    return consolehttp(method, [...urlParams], null, {offset, limit, id, keyword}); 
}

/**
 * 新增实例配置
 */
export const addPolicyConfig = (body: addPolicyConfigBodyType[]) => {
    return consolehttp("post", ["authorization", "v1", "policy"], body, null);
}

/**
 * 修改实例配置
 */
export const updatePolicyConfig = ({ ids, body }: { ids: string; body: updatePolicyConfigBodyType[] }) => {
    return consolehttp("put", ["authorization", "v1", "policy", ids], body, null)
}

/**
 * 删除实例配置
 */
export const deletePolicyConfig = (ids: string) => {
    return consolehttp("delete", ["authorization", "v1", "policy", ids], null, null)
}

/**
 * 查询义务类型
 */
export const getObligationsType = ({ resource_type_id, operation_ids }: { resource_type_id: string; operation_ids?: string[] }) => {
    return consolehttp("get", ["authorization", "v1", "query-obligation-types"], { resource_type_id, operation_ids }, null)
}