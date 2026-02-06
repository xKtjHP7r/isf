// Package driveradapters AnyShare 入站适配器
package driveradapters

import (
	"context"
	_ "embed" // 标准用法
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/xeipuuv/gojsonschema"

	gerrors "github.com/kweaver-ai/go-lib/error"
	"github.com/kweaver-ai/go-lib/rest"

	"Authorization/interfaces"
	"Authorization/logics"
)

//go:embed jsonschema/resource/set.json
var setSchemaStr string

//go:embed jsonschema/resource/set_private.json
var setPrivateSchemaStr string

var (
	resourceTypeOnce    sync.Once
	resourceTypeHandler RestHandler
)

type resourceTypeRestHandler struct {
	resourceType        interfaces.LogicsResourceType
	hydra               interfaces.Hydra
	setSchemaStr        *gojsonschema.Schema
	setPrivateSchemaStr *gojsonschema.Schema
}

// NewResourceTypeRestHandler 权限适配器接口
func NewResourceTypeRestHandler() RestHandler {
	resourceTypeOnce.Do(func() {
		resourceTypeHandler = &resourceTypeRestHandler{
			resourceType:        logics.NewResourceType(),
			hydra:               newHydra(),
			setSchemaStr:        newJSONSchema(setSchemaStr),
			setPrivateSchemaStr: newJSONSchema(setPrivateSchemaStr),
		}
	})
	return resourceTypeHandler
}

// RegisterPrivate 注册内部API
func (s *resourceTypeRestHandler) RegisterPrivate(engine *gin.Engine) {
	engine.PUT("/api/authorization/v1/resource_type/:id", s.setPrivate)
}

// RegisterPublic 注册外部API
func (s *resourceTypeRestHandler) RegisterPublic(engine *gin.Engine) {
	engine.GET("/api/authorization/v1/resource_type", s.get)
	engine.PUT("/api/authorization/v1/resource_type/:id", s.set)
	engine.GET("/api/authorization/v1/resource_type/:id", s.getByID)
	engine.GET("/api/authorization/v1/resource_all_operation", s.getAllOperation)
	engine.GET("/api/authorization/v2/resource_all_operation", s.getAllOperationV2)
	engine.DELETE("/api/authorization/v1/resource_type/:id", s.delete)
}

//nolint:dupl
func (r *resourceTypeRestHandler) set(c *gin.Context) {
	visitor, err := verify(c, r.hydra)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	resourceTypeID := c.Param("id")

	var resourceDr map[string]any
	if err = validateAndBindGin(c, r.setSchemaStr, &resourceDr); err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	var resourceDescription string
	resourceDescriptionJson, ok := resourceDr["description"]
	if ok {
		resourceDescription = resourceDescriptionJson.(string)
	}

	var resourceInstanceURL string
	resourceInstanceURLJson, ok := resourceDr["instance_url"]
	if ok {
		resourceInstanceURL = resourceInstanceURLJson.(string)
	}

	resourceType := interfaces.ResourceType{
		ID:          resourceTypeID,
		Name:        resourceDr["name"].(string),
		Description: resourceDescription,
		InstanceURL: resourceInstanceURL,
		DataStruct:  resourceDr["data_struct"].(string),
	}

	operationsJson := resourceDr["operation"].([]any)
	for _, operationJson := range operationsJson {
		operationDr := operationJson.(map[string]any)
		operationID := operationDr["id"].(string)
		var operationDescription string
		opeDescriptionJson, ok := operationDr["description"]
		if ok {
			operationDescription = opeDescriptionJson.(string)
		}

		operationNameJson := operationDr["name"].([]any)
		operationNames := []interfaces.OperationName{}
		for _, name := range operationNameJson {
			nameDr := name.(map[string]any)
			operationName := interfaces.OperationName{
				Language: nameDr["language"].(string),
				Value:    nameDr["value"].(string),
			}
			operationNames = append(operationNames, operationName)
		}

		operationScope := []interfaces.OperationScopeType{}
		operationScopeJson := operationDr["scope"].([]any)
		for _, scope := range operationScopeJson {
			scopeStr := scope.(string)
			operationScope = append(operationScope, interfaces.OperationScopeType(scopeStr))
		}

		operation := interfaces.ResourceTypeOperation{
			ID:          operationID,
			Name:        operationNames,
			Description: operationDescription,
			Scope:       operationScope,
		}
		resourceType.Operation = append(resourceType.Operation, operation)
	}
	err = r.resourceType.Set(context.Background(), &visitor, &resourceType)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	rest.ReplyOK(c, http.StatusNoContent, nil)
}

func (r *resourceTypeRestHandler) delete(c *gin.Context) {
	visitor, err := verify(c, r.hydra)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	resourceTypeID := c.Param("id")

	err = r.resourceType.Delete(context.Background(), &visitor, resourceTypeID)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	rest.ReplyOK(c, http.StatusNoContent, nil)
}

func (r *resourceTypeRestHandler) getByID(c *gin.Context) {
	visitor, err := verify(c, r.hydra)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	resourceTypeID := c.Param("id")

	resourceType, err := r.resourceType.GetByID(context.Background(), &visitor, resourceTypeID)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	rest.ReplyOK(c, http.StatusOK, map[string]any{
		"id":           resourceType.ID,
		"name":         resourceType.Name,
		"description":  resourceType.Description,
		"instance_url": resourceType.InstanceURL,
		"data_struct":  resourceType.DataStruct,
		"operation":    resourceType.Operation,
	})
}

func (r *resourceTypeRestHandler) getAllOperation(c *gin.Context) {
	visitor, err := verify(c, r.hydra)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	resourceTypeID := c.Query("resource_type")
	if resourceTypeID == "" {
		err = gerrors.NewError(gerrors.PublicBadRequest, "resource_type is required")
		rest.ReplyErrorV2(c, err)
		return
	}
	scope := c.Query("scope")
	if scope == "" {
		err = gerrors.NewError(gerrors.PublicBadRequest, "scope is required")
		rest.ReplyErrorV2(c, err)
		return
	}

	var scopeType interfaces.OperationScopeType
	if scope == "type" {
		scopeType = interfaces.ScopeType
	} else {
		scopeType = interfaces.ScopeInstance
	}

	operationsInfo, _, err := r.resourceType.GetAllOperation(context.Background(), &visitor, resourceTypeID, scopeType)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}
	resp := make([]map[string]any, 0, len(operationsInfo))
	for i := range operationsInfo {
		resp = append(resp, map[string]any{
			"id":          operationsInfo[i].ID,
			"name":        operationsInfo[i].Name,
			"description": operationsInfo[i].Description,
		})
	}
	rest.ReplyOK(c, http.StatusOK, resp)
}

func (r *resourceTypeRestHandler) getAllOperationV2(c *gin.Context) {
	visitor, err := verify(c, r.hydra)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	resourceTypeID := c.Query("resource_type")
	if resourceTypeID == "" {
		err = gerrors.NewError(gerrors.PublicBadRequest, "resource_type is required")
		rest.ReplyErrorV2(c, err)
		return
	}
	scope := c.Query("scope")
	if scope == "" {
		err = gerrors.NewError(gerrors.PublicBadRequest, "scope is required")
		rest.ReplyErrorV2(c, err)
		return
	}

	var scopeType interfaces.OperationScopeType
	if scope == "type" {
		scopeType = interfaces.ScopeType
	} else {
		scopeType = interfaces.ScopeInstance
	}

	operationsInfo, childrenOperationsInfo, err := r.resourceType.GetAllOperation(context.Background(), &visitor, resourceTypeID, scopeType)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}
	rootOperationsResp := r.convertOperations(operationsInfo)

	resp := map[string]any{
		"operations": rootOperationsResp,
	}

	if len(childrenOperationsInfo) > 0 {
		childrenOperationsResp := r.convertChildrenOperations(childrenOperationsInfo)
		resp["children"] = childrenOperationsResp
	}

	rest.ReplyOK(c, http.StatusOK, resp)
}

// convertChildrenOperations 递归转换嵌套的childrenOperationsInfo为响应格式
func (r *resourceTypeRestHandler) convertChildrenOperations(childrenOperations []interfaces.ChildrenResourceTypeOperation) []map[string]any {
	result := make([]map[string]any, 0, len(childrenOperations))
	for i := range childrenOperations {
		child := childrenOperations[i]
		childResp := map[string]any{
			"id":         child.ResourceTypeID,
			"name":       child.Name,
			"operations": r.convertOperations(child.Operations),
		}
		if len(child.Children) > 0 {
			childResp["children"] = r.convertChildrenOperations(child.Children)
		}
		result = append(result, childResp)
	}
	return result
}

// convertOperations 转换operations为响应格式
func (r *resourceTypeRestHandler) convertOperations(operations []interfaces.ResourceTypeOperationResponse) []map[string]any {
	result := make([]map[string]any, 0, len(operations))
	for i := range operations {
		result = append(result, map[string]any{
			"id":          operations[i].ID,
			"name":        operations[i].Name,
			"description": operations[i].Description,
		})
	}
	return result
}

func (r *resourceTypeRestHandler) get(c *gin.Context) {
	visitor, err := verify(c, r.hydra)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	// 获取列举用户组信息
	queryInfo, err := getListQueryParam(c)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	parm := interfaces.ResourceTypePagination{
		Offset: queryInfo.offset,
		Limit:  queryInfo.limit,
	}

	count, resourceTypes, err := r.resourceType.GetPagination(context.Background(), &visitor, parm)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	resp := listResultInfo{
		TotalCount: count,
		Entries:    make([]any, 0, len(resourceTypes)),
	}
	for i := range resourceTypes {
		resp.Entries = append(resp.Entries, map[string]any{
			"id":           resourceTypes[i].ID,
			"name":         resourceTypes[i].Name,
			"description":  resourceTypes[i].Description,
			"instance_url": resourceTypes[i].InstanceURL,
			"data_struct":  resourceTypes[i].DataStruct,
			"operation":    resourceTypes[i].Operation,
		})
	}
	rest.ReplyOK(c, http.StatusOK, resp)
}

//nolint:dupl
func (r *resourceTypeRestHandler) setPrivate(c *gin.Context) {
	var err error
	resourceTypeID := c.Param("id")

	var resourceDr map[string]any
	if err = validateAndBindGin(c, r.setPrivateSchemaStr, &resourceDr); err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	var resourceDescription string
	resourceDescriptionJson, ok := resourceDr["description"]
	if ok {
		resourceDescription = resourceDescriptionJson.(string)
	}

	var resourceInstanceURL string
	resourceInstanceURLJson, ok := resourceDr["instance_url"]
	if ok {
		resourceInstanceURL = resourceInstanceURLJson.(string)
	}

	// 判断hidden是否存在，如果存在则设置为hidden
	var hidden bool
	hiddenJson, ok := resourceDr["hidden"]
	if ok {
		hidden = hiddenJson.(bool)
	}

	resourceType := interfaces.ResourceType{
		ID:          resourceTypeID,
		Name:        resourceDr["name"].(string),
		Description: resourceDescription,
		InstanceURL: resourceInstanceURL,
		DataStruct:  resourceDr["data_struct"].(string),
		Hidden:      hidden,
	}

	operationsJson := resourceDr["operation"].([]any)
	for _, operationJson := range operationsJson {
		operationDr := operationJson.(map[string]any)
		operationID := operationDr["id"].(string)
		var operationDescription string
		opeDescriptionJson, ok := operationDr["description"]
		if ok {
			operationDescription = opeDescriptionJson.(string)
		}

		operationNameJson := operationDr["name"].([]any)
		operationNames := []interfaces.OperationName{}
		for _, name := range operationNameJson {
			nameDr := name.(map[string]any)
			operationName := interfaces.OperationName{
				Language: nameDr["language"].(string),
				Value:    nameDr["value"].(string),
			}
			operationNames = append(operationNames, operationName)
		}

		operationScope := []interfaces.OperationScopeType{}
		operationScopeJson := operationDr["scope"].([]any)
		for _, scope := range operationScopeJson {
			scopeStr := scope.(string)
			operationScope = append(operationScope, interfaces.OperationScopeType(scopeStr))
		}

		operation := interfaces.ResourceTypeOperation{
			ID:          operationID,
			Name:        operationNames,
			Description: operationDescription,
			Scope:       operationScope,
		}
		resourceType.Operation = append(resourceType.Operation, operation)
	}
	err = r.resourceType.SetPrivate(context.Background(), &resourceType)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	rest.ReplyOK(c, http.StatusNoContent, nil)
}
