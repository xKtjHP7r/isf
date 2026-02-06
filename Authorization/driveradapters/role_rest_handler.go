// Package driveradapters group AnyShare  用户组逻辑接口处理层
package driveradapters

import (
	_ "embed" // 标准用法
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	gerrors "github.com/kweaver-ai/go-lib/error"
	"github.com/kweaver-ai/go-lib/rest"
	"github.com/xeipuuv/gojsonschema"

	"Authorization/interfaces"
	"Authorization/logics"
)

type roleRestHandler struct {
	hydra                   interfaces.Hydra
	role                    interfaces.LogicsRole
	memberIntTypes          map[interfaces.AccessorType]string
	memberStringTypes       map[string]interfaces.AccessorType
	createRoleSchema        *gojsonschema.Schema
	addDeleteMembersSchema  *gojsonschema.Schema
	modifyRoleSchema        *gojsonschema.Schema
	roleSourceToStrMap      map[interfaces.RoleSource]string
	strToRoleSourceMap      map[string]interfaces.RoleSource
	accessorRoleStrToIntMap map[string]interfaces.AccessorType
}

var (
	roleOnce    sync.Once
	roleHandler RestHandler
)

var (
	//go:embed jsonschema/role/create_role.json
	createRoleSchemaStr string
	//go:embed jsonschema/role/add_delete_members.json
	addDeleteMembersSchemaStr string
	//go:embed jsonschema/role/modify_role.json
	modifyRoleSchemaStr string
)

// NewRoleRestHandler 角色 restful api handler 对象
func NewRoleRestHandler() RestHandler {
	roleOnce.Do(func() {
		roleHandler = &roleRestHandler{
			hydra: newHydra(),
			role:  logics.NewLogicsRole(),
			memberStringTypes: map[string]interfaces.AccessorType{
				"user":       interfaces.AccessorUser,
				"department": interfaces.AccessorDepartment,
				"group":      interfaces.AccessorGroup,
				"app":        interfaces.AccessorApp,
			},
			memberIntTypes: map[interfaces.AccessorType]string{
				interfaces.AccessorUser:       "user",
				interfaces.AccessorDepartment: "department",
				interfaces.AccessorGroup:      "group",
				interfaces.AccessorApp:        "app",
			},
			createRoleSchema:       newJSONSchema(createRoleSchemaStr),
			addDeleteMembersSchema: newJSONSchema(addDeleteMembersSchemaStr),
			modifyRoleSchema:       newJSONSchema(modifyRoleSchemaStr),
			roleSourceToStrMap: map[interfaces.RoleSource]string{
				interfaces.RoleSourceSystem:   "system",
				interfaces.RoleSourceBusiness: "business",
				interfaces.RoleSourceUser:     "user",
			},
			strToRoleSourceMap: map[string]interfaces.RoleSource{
				"system":   interfaces.RoleSourceSystem,
				"business": interfaces.RoleSourceBusiness,
				"user":     interfaces.RoleSourceUser,
			},
			accessorRoleStrToIntMap: map[string]interfaces.AccessorType{
				"user": interfaces.AccessorUser,
			},
		}
	})

	return roleHandler
}

// RegisterPrivate 注册内部API
func (r *roleRestHandler) RegisterPrivate(engine *gin.Engine) {
	// 访问者角色查询
	engine.GET("/api/authorization/v1/accessor_roles", r.getAccessorRoles)
}

// RegisterPublic 注册外部API
func (r *roleRestHandler) RegisterPublic(engine *gin.Engine) {
	// 角色管理
	engine.POST("/api/authorization/v1/roles", r.createRole)
	engine.DELETE("/api/authorization/v1/roles/:id", r.deleteRole)
	engine.PUT("/api/authorization/v1/roles/:id/:fields", r.modifyRole)
	engine.GET("/api/authorization/v1/roles", r.getRole)
	engine.GET("/api/authorization/v1/roles/:id", r.getRoleByID)

	// 角色查询
	engine.GET("/api/authorization/v1/resource_type_roles", r.getRoleByResourceTypeID)

	// 角色成员管理
	engine.GET("/api/authorization/v1/role-members/:id", r.getRoleMembers)
	engine.POST("/api/authorization/v1/role-members/:id", r.addOrDeleteRoleMembers)
}

// createRole 创建组
func (r *roleRestHandler) createRole(c *gin.Context) {
	// token验证和userID获取
	visitor, vErr := verify(c, r.hydra)
	if vErr != nil {
		rest.ReplyErrorV2(c, vErr)
		return
	}

	var jsonReq map[string]any
	if err := validateAndBindGin(c, r.createRoleSchema, &jsonReq); err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	// 获取具体请求参数
	name := jsonReq["name"].(string)
	description := ""
	if descriptionJson, ok := jsonReq["description"]; ok {
		description = descriptionJson.(string)
	}

	strResourceTypeScopeInfo := jsonReq["resource_type_scope"].(map[string]any)
	unlimited := strResourceTypeScopeInfo["unlimited"].(bool)
	resourceTypeScopesInfo := interfaces.ResourceTypeScopeInfo{
		Unlimited: unlimited,
	}
	// 如果资源范围有限制，则需要获取资源类型范围
	if !unlimited {
		resourceTypeScopes := []interfaces.ResourceTypeScope{}
		if strTypes, ok := strResourceTypeScopeInfo["types"]; ok {
			for _, v := range strTypes.([]any) {
				scopeJson := v.(map[string]any)
				tmp := interfaces.ResourceTypeScope{
					ResourceTypeID: scopeJson["id"].(string),
				}
				resourceTypeScopes = append(resourceTypeScopes, tmp)
			}
		}
		resourceTypeScopesInfo.Types = resourceTypeScopes
	}

	roleInfo := &interfaces.RoleInfo{
		Name:                  name,
		Description:           description,
		ResourceTypeScopeInfo: resourceTypeScopesInfo,
	}

	strID, err := r.role.AddRole(c, &visitor, roleInfo)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	c.Writer.Header().Set("Location", fmt.Sprintf("/api/authorization/v1/roles/%s", strID))
	rest.ReplyOK(c, http.StatusCreated, gin.H{"id": strID})
}

// deleteRole 删除组
func (r *roleRestHandler) deleteRole(c *gin.Context) {
	// token验证和userID获取
	visitor, vErr := verify(c, r.hydra)
	if vErr != nil {
		rest.ReplyErrorV2(c, vErr)
		return
	}

	// 获取删除用户组ID
	roleID := c.Param("id")

	// 删除用户组
	err := r.role.DeleteRole(c, &visitor, roleID)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}
	rest.ReplyOK(c, http.StatusNoContent, nil)
}

// modifyRole 修改组
func (r *roleRestHandler) modifyRole(c *gin.Context) {
	// token验证和userID获取
	visitor, vErr := verify(c, r.hydra)
	if vErr != nil {
		rest.ReplyErrorV2(c, vErr)
		return
	}

	// 获取修改用户组 存在哪些参数
	groupID := c.Param("id")
	fields := strings.Split(c.Param("fields"), ",")
	var nameExist, descriptionExist, resourceTypeScopesExist bool
	for _, v := range fields {
		if v == "name" {
			nameExist = true
		}
		if v == "description" {
			descriptionExist = true
		}
		if v == "resource_type_scope" {
			resourceTypeScopesExist = true
		}
	}

	var jsonReq map[string]any
	if err := validateAndBindGin(c, r.modifyRoleSchema, &jsonReq); err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	name := ""
	if nameExist {
		nameJson, ok := jsonReq["name"]
		if ok {
			name = nameJson.(string)
		} else {
			err := gerrors.NewError(gerrors.PublicBadRequest, "param name is illegal")
			rest.ReplyErrorV2(c, err)
			return
		}
	}

	description := ""
	if descriptionExist {
		descriptionJson, ok := jsonReq["description"]
		if ok {
			description = descriptionJson.(string)
		} else {
			err := gerrors.NewError(gerrors.PublicBadRequest, "param description is illegal")
			rest.ReplyErrorV2(c, err)
			return
		}
	}

	resourceTypeScopesInfo := interfaces.ResourceTypeScopeInfo{}
	if resourceTypeScopesExist {
		strResourceTypeScopesInfo, ok := jsonReq["resource_type_scope"]
		if ok {
			strResourceTypeScopesInfoJson := strResourceTypeScopesInfo.(map[string]any)
			resourceTypeScopesInfo.Unlimited = strResourceTypeScopesInfoJson["unlimited"].(bool)
			resourceTypeScopesInfo.Types = make([]interfaces.ResourceTypeScope, 0, len(strResourceTypeScopesInfoJson["types"].([]any)))
			for _, v := range strResourceTypeScopesInfoJson["types"].([]any) {
				scopeJson := v.(map[string]any)
				resourceTypeScopesInfo.Types = append(resourceTypeScopesInfo.Types, interfaces.ResourceTypeScope{
					ResourceTypeID: scopeJson["id"].(string),
				})
			}
		} else {
			err := gerrors.NewError(gerrors.PublicBadRequest, "param resource_type_scope is illegal")
			rest.ReplyErrorV2(c, err)
			return
		}
	}

	// 修改用户组
	err := r.role.ModifyRole(c, &visitor, groupID, name, nameExist, description, descriptionExist, resourceTypeScopesInfo, resourceTypeScopesExist)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	rest.ReplyOK(c, http.StatusNoContent, nil)
}

// getRole 列举组
//
//nolint:staticcheck
func (r *roleRestHandler) getRole(c *gin.Context) {
	// token验证和userID获取
	visitor, vErr := verify(c, r.hydra)
	if vErr != nil {
		rest.ReplyErrorV2(c, vErr)
		return
	}

	// 获取列举用户组信息
	queryInfo, err := getListQueryParam(c)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	roleSources := c.QueryArray("source")
	roleSourcesTmp := make([]interfaces.RoleSource, 0, len(roleSources))
	for _, v := range roleSources {
		// 不存在直接抛错
		if _, ok := r.strToRoleSourceMap[v]; !ok {
			err := gerrors.NewError(gerrors.PublicBadRequest, "param source is illegal")
			rest.ReplyErrorV2(c, err)
			return
		}
		roleSourcesTmp = append(roleSourcesTmp, r.strToRoleSourceMap[v])
	}

	// 不传时 默认返回业务内置和用户自定义
	if len(roleSourcesTmp) == 0 {
		roleSourcesTmp = []interfaces.RoleSource{interfaces.RoleSourceBusiness, interfaces.RoleSourceUser}
	}

	keyword := c.DefaultQuery("keyword", "")
	searchInfo := interfaces.RoleSearchInfo{
		Offset:      queryInfo.offset,
		Limit:       queryInfo.limit,
		Keyword:     keyword,
		RoleSources: roleSourcesTmp,
	}

	// 列举用户组
	num, groupInfos, err := r.role.GetRoles(c, &visitor, searchInfo)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	// 设置返回参数
	resp := listResultInfo{
		TotalCount: num,
		Entries:    make([]any, 0, len(groupInfos)),
	}

	for _, v := range groupInfos {
		data := make(map[string]any)
		data["id"] = v.ID
		data["name"] = v.Name
		data["description"] = v.Description
		data["source"] = r.roleSourceToStrMap[v.RoleSource]
		resourceTypeScopeInfoResp := make(map[string]any)
		resourceTypeScopeInfoResp["unlimited"] = v.ResourceTypeScopeInfo.Unlimited
		typesResp := make([]any, 0, len(v.ResourceTypeScopeInfo.Types))
		for _, v := range v.ResourceTypeScopeInfo.Types {
			typesResp = append(typesResp, map[string]any{
				"id":   v.ResourceTypeID,
				"name": v.ResourceTypeName,
			})
		}
		resourceTypeScopeInfoResp["types"] = typesResp
		data["resource_type_scopes"] = resourceTypeScopeInfoResp
		resp.Entries = append(resp.Entries, data)
	}

	rest.ReplyOK(c, http.StatusOK, resp)
}

// getRoleByID 根据组ID获取组信息
func (r *roleRestHandler) getRoleByID(c *gin.Context) {
	// token验证和userID获取
	visitor, vErr := verify(c, r.hydra)
	if vErr != nil {
		rest.ReplyErrorV2(c, vErr)
		return
	}

	// 获取用户组ID
	groupID := c.Param("id")

	param := interfaces.RoleInfoParam{}
	// 新增 resource_type_view_mode
	resourceTypeViewMode := c.Query("resource_type_view_mode")
	if resourceTypeViewMode == "" {
		param.ResourceTypeViewMode = interfaces.ResourceTypeViewModeFlat
	} else {
		if resourceTypeViewMode != string(interfaces.ResourceTypeViewModeFlat) &&
			resourceTypeViewMode != string(interfaces.ResourceTypeViewModeHierarchy) {
			err := gerrors.NewError(gerrors.PublicBadRequest, "param resource_type_view_mode is illegal")
			rest.ReplyErrorV2(c, err)
			return
		}
		param.ResourceTypeViewMode = interfaces.ResourceTypeViewMode(resourceTypeViewMode)
	}

	// 获取用户组信息
	roleInfo, err := r.role.GetRoleByID(c, &visitor, groupID, param)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}
	resourcInfoTypeResp := make([]any, 0, len(roleInfo.ResourceTypeScopesInfo.Types))
	typesTmp := roleInfo.ResourceTypeScopesInfo.Types
	for i := range typesTmp {
		resourceInfo := make(map[string]any)
		resourceInfo["id"] = typesTmp[i].ID
		resourceInfo["name"] = typesTmp[i].Name
		resourceInfo["instance_url"] = typesTmp[i].InstanceURL
		resourceInfo["data_struct"] = typesTmp[i].DataStruct
		resourceInfo["description"] = typesTmp[i].Description
		typeOperationsResp := make([]any, 0, len(typesTmp[i].TypeOperation))
		for _, operation := range typesTmp[i].TypeOperation {
			operationResp := make(map[string]any)
			operationResp["id"] = operation.ID
			operationResp["description"] = operation.Description
			operationResp["name"] = operation.Name
			typeOperationsResp = append(typeOperationsResp, operationResp)
		}
		instanceOperationsResp := make([]any, 0, len(typesTmp[i].InstanceOperation))
		for _, operation := range typesTmp[i].InstanceOperation {
			operationResp := make(map[string]any)
			operationResp["id"] = operation.ID
			operationResp["description"] = operation.Description
			operationResp["name"] = operation.Name
			instanceOperationsResp = append(instanceOperationsResp, operationResp)
		}
		operationResp := make(map[string]any)
		operationResp["type"] = typeOperationsResp
		operationResp["instance"] = instanceOperationsResp
		resourceInfo["operation"] = operationResp
		if len(typesTmp[i].Children) > 0 {
			// 递归填充子资源类型
			resourceInfo["children"] = r.convertChildrenResourceTypeScope(typesTmp[i].Children)
		}
		resourcInfoTypeResp = append(resourcInfoTypeResp, resourceInfo)
	}
	resourcInforesp := make(map[string]any)
	resourcInforesp["types"] = resourcInfoTypeResp
	resourcInforesp["unlimited"] = roleInfo.ResourceTypeScopesInfo.Unlimited
	out := make(map[string]any)
	out["id"] = roleInfo.ID
	out["name"] = roleInfo.Name
	out["description"] = roleInfo.Description
	out["resource_type_scopes"] = resourcInforesp
	out["source"] = r.roleSourceToStrMap[roleInfo.RoleSource]
	rest.ReplyOK(c, http.StatusOK, out)
}

// convertChildrenResourceTypeScope 递归转换子资源类型
func (r *roleRestHandler) convertChildrenResourceTypeScope(children []interfaces.ResourceTypeScopeWithOperation) []any {
	result := make([]any, 0, len(children))
	for i := range children {
		childInfo := make(map[string]any)
		childInfo["id"] = children[i].ID
		childInfo["name"] = children[i].Name
		childInfo["description"] = children[i].Description
		childInfo["instance_url"] = children[i].InstanceURL
		childInfo["data_struct"] = children[i].DataStruct
		typeOperationsResp := make([]any, 0, len(children[i].TypeOperation))
		for _, operation := range children[i].TypeOperation {
			operationResp := map[string]any{
				"id":          operation.ID,
				"description": operation.Description,
				"name":        operation.Name,
			}
			typeOperationsResp = append(typeOperationsResp, operationResp)
		}
		instanceOperationsResp := make([]any, 0, len(children[i].InstanceOperation))
		for _, operation := range children[i].InstanceOperation {
			operationResp := map[string]any{
				"id":          operation.ID,
				"description": operation.Description,
				"name":        operation.Name,
			}
			instanceOperationsResp = append(instanceOperationsResp, operationResp)
		}
		childInfo["operation"] = map[string]any{
			"type":     typeOperationsResp,
			"instance": instanceOperationsResp,
		}
		if len(children[i].Children) > 0 {
			childInfo["children"] = r.convertChildrenResourceTypeScope(children[i].Children)
		}
		result = append(result, childInfo)
	}
	return result
}

// getRoleMembers 列举组成员
func (r *roleRestHandler) getRoleMembers(c *gin.Context) {
	// token验证和userID获取
	visitor, vErr := verify(c, r.hydra)
	if vErr != nil {
		rest.ReplyErrorV2(c, vErr)
		return
	}

	// 获取请求参数
	roleID := c.Param("id")
	queryInfo, err := getListQueryParam(c)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	// type 是一个数组 需要转换为字符串
	memberType := c.QueryArray("type")
	memberTypes := make([]interfaces.AccessorType, 0, len(memberType))
	for _, v := range memberType {
		if _, ok := r.memberStringTypes[v]; ok {
			memberTypes = append(memberTypes, r.memberStringTypes[v])
		} else {
			err := gerrors.NewError(gerrors.PublicBadRequest, "param type is illegal")
			rest.ReplyErrorV2(c, err)
			return
		}
	}

	keyword := c.DefaultQuery("keyword", "")
	searchInfo := interfaces.RoleMemberSearchInfo{
		Offset:      queryInfo.offset,
		Limit:       queryInfo.limit,
		MemberTypes: memberTypes,
		Keyword:     keyword,
	}

	// 列举组成员
	num, infos, err := r.role.GetRoleMembers(c, &visitor, roleID, searchInfo)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	// 设置返回参数
	outInfo := make(map[string]any)
	outInfo["total_count"] = num

	entriesData := make([]any, 0, len(infos))
	for i := range infos {
		entriesData = append(entriesData, r.handleGroupMemberInfo(&infos[i]))
	}
	outInfo["entries"] = entriesData

	rest.ReplyOK(c, http.StatusOK, outInfo)
}

// addOrDeleteRoleMembers 添加和删除组成员
func (r *roleRestHandler) addOrDeleteRoleMembers(c *gin.Context) {
	// token验证和userID获取
	visitor, vErr := verify(c, r.hydra)
	if vErr != nil {
		rest.ReplyErrorV2(c, vErr)
		return
	}

	// 获取请求参数
	roleID := c.Param("id")

	var jsonReq map[string]any
	if err := validateAndBindGin(c, r.addDeleteMembersSchema, &jsonReq); err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	method := jsonReq["method"].(string)
	members := jsonReq["members"].([]any)
	memberInfos := make(map[string]interfaces.RoleMemberInfo)
	for _, v := range members {
		member := interfaces.RoleMemberInfo{}
		temp := v.(map[string]any)
		member.ID = temp["id"].(string)
		strType := temp["type"].(string)
		var ok bool
		member.MemberType, ok = r.memberStringTypes[strType]
		if !ok {
			err := gerrors.NewError(gerrors.PublicBadRequest, "param member type is illegal")
			rest.ReplyErrorV2(c, err)
			return
		}
		if member.ID == "" {
			err := gerrors.NewError(gerrors.PublicBadRequest, "param member id is illegal")
			rest.ReplyErrorV2(c, err)
			return
		}

		memberInfos[member.ID] = member
	}
	if method != "POST" && method != "DELETE" {
		err := gerrors.NewError(gerrors.PublicBadRequest, "param method is illegal")
		rest.ReplyErrorV2(c, err)
		return
	}

	// 批量添加或者删除用户组成员
	cErr := r.role.AddOrDeleteRoleMemebers(c, &visitor, method, roleID, memberInfos)
	if cErr != nil {
		rest.ReplyErrorV2(c, cErr)
		return
	}
	rest.ReplyOK(c, http.StatusNoContent, nil)
}

// handleGroupMemberInfos 处理用户组成员信息
func (r *roleRestHandler) handleGroupMemberInfo(input *interfaces.RoleMemberInfo) any {
	tempMemberInfo := make(map[string]any)
	tempMemberInfo["id"] = input.ID
	tempMemberInfo["type"] = r.memberIntTypes[input.MemberType]
	tempMemberInfo["name"] = input.Name
	tempMemberInfo["parent_deps"] = input.ParentDeps
	return tempMemberInfo
}

// getRoleByResourceTypeID 根据资源类型ID列举角色
func (r *roleRestHandler) getRoleByResourceTypeID(c *gin.Context) {
	// token验证和userID获取
	visitor, vErr := verify(c, r.hydra)
	if vErr != nil {
		rest.ReplyErrorV2(c, vErr)
		return
	}

	// 获取列举用户组信息
	queryInfo, err := getListQueryParam(c)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	resourceTypeID := c.DefaultQuery("resource_type_id", "")
	if resourceTypeID == "" {
		err := gerrors.NewError(gerrors.PublicBadRequest, "param resource_type_id is required")
		rest.ReplyErrorV2(c, err)
		return
	}

	keyword := c.DefaultQuery("keyword", "")
	searchInfo := interfaces.ResourceTypeRoleSearchInfo{
		Offset:         queryInfo.offset,
		Limit:          queryInfo.limit,
		Keyword:        keyword,
		ResourceTypeID: resourceTypeID,
	}

	// 列举资源类型角色
	num, roleInfos, err := r.role.GetResourceTypeRoles(c, &visitor, searchInfo)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	// 设置返回参数
	resp := listResultInfo{
		TotalCount: num,
		Entries:    make([]any, 0, len(roleInfos)),
	}

	for _, v := range roleInfos {
		data := make(map[string]any)
		data["id"] = v.ID
		data["name"] = v.Name
		data["description"] = v.Description
		resp.Entries = append(resp.Entries, data)
	}

	rest.ReplyOK(c, http.StatusOK, resp)
}

// getAccessorRoles 根据访问者类型列举角色
func (r *roleRestHandler) getAccessorRoles(c *gin.Context) {
	var err error
	param := interfaces.AccessorRoleSearchInfo{}
	// 获取访问者ID
	param.AccessorID = c.Query("accessor_id")
	if param.AccessorID == "" {
		err = gerrors.NewError(gerrors.PublicBadRequest, "param accessor_id is required")
		rest.ReplyErrorV2(c, err)
		return
	}
	// 获取访问者类型
	accessorTypeStr := c.Query("accessor_type")
	accessorType, ok := r.accessorRoleStrToIntMap[accessorTypeStr]
	if !ok {
		err = gerrors.NewError(gerrors.PublicBadRequest, "param accessor_type is illegal")
		rest.ReplyErrorV2(c, err)
		return
	}
	param.AccessorType = accessorType

	// 获取数据下标
	param.Offset, _ = strconv.Atoi(c.DefaultQuery("offset", "0"))
	if param.Offset < 0 {
		err = gerrors.NewError(gerrors.PublicBadRequest, "invalid offset(>=0)")
		rest.ReplyErrorV2(c, err)
		return
	}

	// 获取分页数据
	param.Limit, _ = strconv.Atoi(c.DefaultQuery("limit", "20"))
	if (param.Limit != -1) && (param.Limit < 1 || param.Limit > 1000) {
		err = gerrors.NewError(gerrors.PublicBadRequest, "invalid limit([ 1 .. 1000 ] or -1)")
		rest.ReplyErrorV2(c, err)
		return
	}

	// 获取角色来源过滤条件
	roleSources := c.QueryArray("source")
	roleSourcesTmp := make([]interfaces.RoleSource, 0, len(roleSources))
	for _, v := range roleSources {
		// 不存在直接抛错
		if _, ok := r.strToRoleSourceMap[v]; !ok {
			err := gerrors.NewError(gerrors.PublicBadRequest, "param source is illegal")
			rest.ReplyErrorV2(c, err)
			return
		}
		roleSourcesTmp = append(roleSourcesTmp, r.strToRoleSourceMap[v])
	}

	// 不传时 默认返回业务内置和用户自定义
	if len(roleSourcesTmp) == 0 {
		roleSourcesTmp = []interfaces.RoleSource{interfaces.RoleSourceBusiness, interfaces.RoleSourceUser}
	}

	param.RoleSources = roleSourcesTmp
	// 根据访问者列举角色
	num, roleInfos, err := r.role.GetAccessorRoles(c, param)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	// 设置返回参数
	resp := listResultInfo{
		TotalCount: num,
		Entries:    make([]any, 0, len(roleInfos)),
	}

	for _, v := range roleInfos {
		data := make(map[string]any)
		data["id"] = v.ID
		data["name"] = v.Name
		data["description"] = v.Description
		resp.Entries = append(resp.Entries, data)
	}

	rest.ReplyOK(c, http.StatusOK, resp)
}
