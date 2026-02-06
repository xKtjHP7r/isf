// Package driveradapters AnyShare 入站适配器
package driveradapters

import (
	"context"
	_ "embed" // 标准用法
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/xeipuuv/gojsonschema"

	"github.com/kweaver-ai/go-lib/rest"

	gerrors "github.com/kweaver-ai/go-lib/error"

	"Authorization/common"
	"Authorization/interfaces"
	"Authorization/logics"
)

var (
	//go:embed jsonschema/policy/policy.json
	policySchemaStr string
	//go:embed jsonschema/policy/modify_policy.json
	modifyPolicySchemaStr string
	//go:embed jsonschema/policy/policy_delete.json
	policyDeleteSchemaStr string
)

var (
	policyOnce    sync.Once
	policyHandler RestHandler
)

type policyRestHandler struct {
	policy             interfaces.LogicsPolicy
	hydra              interfaces.Hydra
	policySchema       *gojsonschema.Schema
	modifyPolicySchema *gojsonschema.Schema
	policyDeleteSchema *gojsonschema.Schema
	accessorStrToType  map[string]interfaces.AccessorType
	accessorTypeToStr  map[interfaces.AccessorType]string
	includeStrToType   map[string]interfaces.PolicyIncludeType
	logger             common.Logger
}

// NewPolicyRestHandler 策略适配器接口
func NewPolicyRestHandler() RestHandler {
	policyOnce.Do(func() {
		policyHandler = &policyRestHandler{
			policy:             logics.NewPolicy(),
			hydra:              newHydra(),
			policySchema:       newJSONSchema(policySchemaStr),
			modifyPolicySchema: newJSONSchema(modifyPolicySchemaStr),
			policyDeleteSchema: newJSONSchema(policyDeleteSchemaStr),
			logger:             common.NewLogger(),
			accessorStrToType: map[string]interfaces.AccessorType{
				// 用户、部门、用户组
				"user":       interfaces.AccessorUser,
				"department": interfaces.AccessorDepartment,
				"group":      interfaces.AccessorGroup,
				"app":        interfaces.AccessorApp,
				"role":       interfaces.AccessorRole,
			},
			accessorTypeToStr: map[interfaces.AccessorType]string{
				interfaces.AccessorUser:       "user",
				interfaces.AccessorDepartment: "department",
				interfaces.AccessorGroup:      "group",
				interfaces.AccessorApp:        "app",
				interfaces.AccessorRole:       "role",
			},
			includeStrToType: map[string]interfaces.PolicyIncludeType{
				"obligation_types": interfaces.PolicyIncludeObligationType,
				"obligations":      interfaces.PolicyIncludeObligation,
			},
		}
	})
	return policyHandler
}

// RegisterPrivate 注册内部API
func (p *policyRestHandler) RegisterPrivate(engine *gin.Engine) {
	engine.POST("/api/authorization/v1/policy", p.createPrivate)
	engine.POST("/api/authorization/v1/policy-delete", p.deletePrivate)
}

// RegisterPublic 注册外部API
func (p *policyRestHandler) RegisterPublic(engine *gin.Engine) {
	engine.GET("/api/authorization/v1/policy", p.get)
	engine.POST("/api/authorization/v1/policy", p.create)
	engine.PUT("/api/authorization/v1/policy/:ids", p.set)
	engine.DELETE("/api/authorization/v1/policy/:ids", p.delete)
	engine.GET("/api/authorization/v1/resource-policy", p.getResourcePolicy)
	engine.GET("/api/authorization/v1/accessor-policy", p.getAccessorPolicy)
}

func (p *policyRestHandler) create(c *gin.Context) {
	visitor, err := verify(c, p.hydra)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	var jsonReqs []any
	if err = validateAndBindGin(c, p.policySchema, &jsonReqs); err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	policys := []interfaces.PolicyInfo{}
	for _, jsonReqJson := range jsonReqs {
		jsonReq := jsonReqJson.(map[string]any)
		// 客体
		accessor := jsonReq["accessor"].(map[string]any)
		accessorID := accessor["id"].(string)
		accessorType := p.accessorStrToType[accessor["type"].(string)]
		resource := jsonReq["resource"].(map[string]any)
		resourceID := resource["id"].(string)
		resourceType := resource["type"].(string)
		resourceName := resource["name"].(string)
		var ancestors []interfaces.Ancestor
		ancestorsJson, hasAncestors := resource["ancestors"]
		if hasAncestors {
			ancestors = p.ancestorsStrToInfo(ancestorsJson)
		}

		operationJson := jsonReq["operation"].(map[string]any)
		allowJson := operationJson["allow"].([]any)
		denyJson := operationJson["deny"].([]any)
		allow := []interfaces.PolicyOperationItem{}
		deny := []interfaces.PolicyOperationItem{}
		for _, v := range allowJson {
			item := v.(map[string]any)
			allowItem := interfaces.PolicyOperationItem{
				ID: item["id"].(string),
			}
			// 解析义务
			obligationsJson, exist := item["obligations"]
			if exist {
				allowItem.Obligations, err = p.getObligations(obligationsJson)
				if err != nil {
					rest.ReplyErrorV2(c, err)
					return
				}
			}
			allow = append(allow, allowItem)
		}
		for _, v := range denyJson {
			item := v.(map[string]any)
			denyItem := interfaces.PolicyOperationItem{
				ID: item["id"].(string),
			}
			deny = append(deny, denyItem)
		}

		operation := interfaces.PolicyOperation{
			Allow: allow,
			Deny:  deny,
		}

		var condition string
		conditionJson, exist := jsonReq["condition"]
		if exist {
			condition = conditionJson.(string)
		} else {
			condition = ""
		}

		var timeStamp int64
		var endTime int64
		expiresAtJson, exist := jsonReq["expires_at"]
		if exist {
			// 权限到期时间
			timeStamp, err = rest.StringToTimeStamp(expiresAtJson.(string))
			if err != nil {
				err = gerrors.NewError(gerrors.PublicBadRequest, "param expires_at is invalid")
				rest.ReplyErrorV2(c, err)
				return
			}
			// 数据库中 -1 表示永久 单位使用微妙
			if timeStamp == 0 {
				endTime = -1
			} else {
				endTime = timeStamp / 1000
			}
		} else {
			endTime = -1
		}

		policy := interfaces.PolicyInfo{
			AccessorID:   accessorID,
			AccessorType: accessorType,
			ResourceID:   resourceID,
			ResourceType: resourceType,
			ResourceName: resourceName,
			Operation:    operation,
			Condition:    condition,
			EndTime:      endTime,
			Ancestors:    ancestors,
			HasAncestors: hasAncestors,
		}
		policys = append(policys, policy)
	}
	policyIDs, err := p.policy.Create(context.Background(), &visitor, policys)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}
	if len(policyIDs) > 0 {
		strIDs := strings.Join(policyIDs, ",")
		c.Writer.Header().Set("Location", fmt.Sprintf("/api/authorization/v1/policy/%s", strIDs))
	}

	rest.ReplyOK(c, http.StatusCreated, gin.H{"ids": policyIDs})
}

func (p *policyRestHandler) createPrivate(c *gin.Context) {
	var err error
	var jsonReqs []any
	if err = validateAndBindGin(c, p.policySchema, &jsonReqs); err != nil {
		p.logger.Errorf("createPrivate: validateAndBindGin: %v", err)
		rest.ReplyErrorV2(c, err)
		return
	}

	policys := []interfaces.PolicyInfo{}
	for _, jsonReqJson := range jsonReqs {
		jsonReq := jsonReqJson.(map[string]any)
		// 客体
		accessor := jsonReq["accessor"].(map[string]any)
		accessorID := accessor["id"].(string)
		accessorType := p.accessorStrToType[accessor["type"].(string)]
		resource := jsonReq["resource"].(map[string]any)
		resourceID := resource["id"].(string)
		resourceType := resource["type"].(string)
		resourceName := resource["name"].(string)
		var ancestors []interfaces.Ancestor
		ancestorsJson, hasAncestors := resource["ancestors"]
		if hasAncestors {
			ancestors = p.ancestorsStrToInfo(ancestorsJson)
		}

		operationJson := jsonReq["operation"].(map[string]any)
		allowJson := operationJson["allow"].([]any)
		denyJson := operationJson["deny"].([]any)
		allow := []interfaces.PolicyOperationItem{}
		deny := []interfaces.PolicyOperationItem{}
		for _, v := range allowJson {
			item := v.(map[string]any)
			allowItem := interfaces.PolicyOperationItem{
				ID: item["id"].(string),
			}
			// 解析义务
			obligationsJson, exist := item["obligations"]
			if exist {
				allowItem.Obligations, err = p.getObligations(obligationsJson)
				if err != nil {
					rest.ReplyErrorV2(c, err)
					return
				}
			}
			allow = append(allow, allowItem)
		}
		for _, v := range denyJson {
			item := v.(map[string]any)
			deny = append(deny, interfaces.PolicyOperationItem{
				ID: item["id"].(string),
			})
		}

		operation := interfaces.PolicyOperation{
			Allow: allow,
			Deny:  deny,
		}

		var condition string
		conditionJson, exist := jsonReq["condition"]
		if exist {
			condition = conditionJson.(string)
		} else {
			condition = ""
		}

		var timeStamp int64
		var endTime int64
		expiresAtJson, exist := jsonReq["expires_at"]
		if exist {
			// 权限到期时间
			timeStamp, err = rest.StringToTimeStamp(expiresAtJson.(string))
			if err != nil {
				err = gerrors.NewError(gerrors.PublicBadRequest, "param expires_at is invalid")
				rest.ReplyErrorV2(c, err)
				return
			}
			// 数据库中 -1 表示永久 单位使用微妙
			if timeStamp == 0 {
				endTime = -1
			} else {
				endTime = timeStamp / 1000
			}
		} else {
			endTime = -1
		}

		policy := interfaces.PolicyInfo{
			AccessorID:   accessorID,
			AccessorType: accessorType,
			ResourceID:   resourceID,
			ResourceType: resourceType,
			ResourceName: resourceName,
			Operation:    operation,
			Condition:    condition,
			EndTime:      endTime,
			Ancestors:    ancestors,
			HasAncestors: hasAncestors,
		}
		policys = append(policys, policy)
	}

	err = p.policy.CreatePrivate(context.Background(), policys)
	if err != nil {
		p.logger.Errorf("createPrivate call logic err: %v", err)
		rest.ReplyErrorV2(c, err)
		return
	}

	rest.ReplyOK(c, http.StatusNoContent, gin.H{})
}

func (p *policyRestHandler) deletePrivate(c *gin.Context) {
	var err error
	var jsonReqs map[string]any
	if err = validateAndBindGin(c, p.policyDeleteSchema, &jsonReqs); err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}
	var resources []interfaces.PolicyDeleteResourceInfo
	resourcesJson := jsonReqs["resources"].([]any)

	for _, resourceJson := range resourcesJson {
		resource := resourceJson.(map[string]any)
		resourceID := resource["id"].(string)
		resourceType := resource["type"].(string)
		resources = append(resources, interfaces.PolicyDeleteResourceInfo{
			ID:   resourceID,
			Type: resourceType,
		})
	}

	err = p.policy.DeleteByResourceIDs(context.Background(), resources)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	rest.ReplyOK(c, http.StatusNoContent, gin.H{})
}

func (p *policyRestHandler) set(c *gin.Context) {
	visitor, err := verify(c, p.hydra)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	var jsonReqs []any
	if err = validateAndBindGin(c, p.modifyPolicySchema, &jsonReqs); err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	policyIDsStr := c.Param("ids")
	policyIDs := strings.Split(policyIDsStr, ",")
	policys := []interfaces.PolicyInfo{}
	for i, jsonReqJson := range jsonReqs {
		// 客体
		jsonReq := jsonReqJson.(map[string]any)
		operationJson := jsonReq["operation"].(map[string]any)
		allowJson := operationJson["allow"].([]any)
		denyJson := operationJson["deny"].([]any)
		allow := []interfaces.PolicyOperationItem{}
		deny := []interfaces.PolicyOperationItem{}
		for _, v := range allowJson {
			item := v.(map[string]any)
			allowItem := interfaces.PolicyOperationItem{
				ID: item["id"].(string),
			}
			obligationsJson, exist := item["obligations"]
			if exist {
				allowItem.Obligations, err = p.getObligations(obligationsJson)
				if err != nil {
					rest.ReplyErrorV2(c, err)
					return
				}
			}
			allow = append(allow, allowItem)
		}
		for _, v := range denyJson {
			item := v.(map[string]any)
			denyItem := interfaces.PolicyOperationItem{
				ID: item["id"].(string),
			}
			deny = append(deny, denyItem)
		}
		operation := interfaces.PolicyOperation{
			Allow: allow,
			Deny:  deny,
		}

		var condition string
		conditionJson, exist := jsonReq["condition"]
		if exist {
			condition = conditionJson.(string)
		} else {
			condition = ""
		}

		var timeStamp int64
		var endTime int64
		expiresAtJson, exist := jsonReq["expires_at"]
		if exist {
			// 权限到期时间
			timeStamp, err = rest.StringToTimeStamp(expiresAtJson.(string))
			if err != nil {
				err = gerrors.NewError(gerrors.PublicBadRequest, "param expires_at is invalid")
				rest.ReplyErrorV2(c, err)
				return
			}
			// 数据库中 -1 表示永久 单位使用微妙
			if timeStamp == 0 {
				endTime = -1
			} else {
				endTime = timeStamp / 1000
			}
		} else {
			endTime = -1
		}

		policy := interfaces.PolicyInfo{
			ID:        policyIDs[i],
			Operation: operation,
			Condition: condition,
			EndTime:   endTime,
		}
		policys = append(policys, policy)
	}

	err = p.policy.Update(context.Background(), &visitor, policys)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	rest.ReplyOK(c, http.StatusNoContent, gin.H{})
}

func (p *policyRestHandler) delete(c *gin.Context) {
	visitor, err := verify(c, p.hydra)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	policyIDsStr := c.Param("ids")
	policyIDs := strings.Split(policyIDsStr, ",")
	err = p.policy.Delete(context.Background(), &visitor, policyIDs)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	rest.ReplyOK(c, http.StatusNoContent, gin.H{})
}

func (p *policyRestHandler) get(c *gin.Context) {
	visitor, err := verify(c, p.hydra)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}
	parm := interfaces.PolicyPagination{
		ResourceID:   c.Query("resource_id"),
		ResourceType: c.Query("resource_type"),
	}
	// 必传参数校验
	if parm.ResourceID == "" {
		err = gerrors.NewError(gerrors.PublicBadRequest, "resource_id is required")
		rest.ReplyErrorV2(c, err)
		return
	}
	if parm.ResourceType == "" {
		err = gerrors.NewError(gerrors.PublicBadRequest, "resource_type is required")
		rest.ReplyErrorV2(c, err)
		return
	}

	info, err := getListQueryParam(c)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}
	parm.Offset = info.offset
	parm.Limit = info.limit

	count, policies, err := p.policy.GetPagination(context.Background(), &visitor, parm)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	resp := listResultInfo{
		TotalCount: count,
		Entries:    make([]any, 0, len(policies)),
	}

	for i := range policies {
		expiresAt := policies[i].EndTime
		if expiresAt == -1 {
			expiresAt = 0
		} else {
			expiresAt *= 1000
		}
		resp.Entries = append(resp.Entries, map[string]any{
			"id": policies[i].ID,
			"resource": map[string]any{
				"id":   policies[i].ResourceID,
				"type": policies[i].ResourceType,
				"name": policies[i].ResourceName,
			},
			"accessor": map[string]any{
				"id":          policies[i].AccessorID,
				"type":        p.accessorTypeToStr[policies[i].AccessorType],
				"name":        policies[i].AccessorName,
				"parent_deps": policies[i].ParentDeps,
			},
			"operation": map[string]any{
				"allow": p.operationArrayToJson(policies[i].Operation.Allow),
				"deny":  p.operationArrayToJson(policies[i].Operation.Deny),
			},
			"condition":  policies[i].Condition,
			"expires_at": rest.TimeStampToString(expiresAt),
		})
	}

	rest.ReplyOK(c, http.StatusOK, resp)
}

//nolint:dupl
func (p *policyRestHandler) getAccessorPolicy(c *gin.Context) {
	visitor, err := verify(c, p.hydra)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}
	accessorID := c.Query("accessor_id")
	accessorTypeStr := c.Query("accessor_type")
	resourceType := c.Query("resource_type")
	resourceID := c.Query("resource_id")

	accessorType, ok := p.accessorStrToType[accessorTypeStr]
	if !ok {
		err = gerrors.NewError(gerrors.PublicBadRequest, "accessor_type is invalid")
		rest.ReplyErrorV2(c, err)
		return
	}

	param := interfaces.AccessorPolicyParam{
		AccessorID:   accessorID,
		AccessorType: accessorType,
		ResourceType: resourceType,
		ResourceID:   resourceID,
	}

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

	// 获取include参数
	includeStr := c.QueryArray("include")
	for _, v := range includeStr {
		includeType, ok := p.includeStrToType[v]
		if !ok {
			err = gerrors.NewError(gerrors.PublicBadRequest, "include is invalid")
			rest.ReplyErrorV2(c, err)
			return
		}
		param.Include = append(param.Include, includeType)
	}

	count, policies, includeResp, err := p.policy.GetAccessorPolicy(context.Background(), &visitor, param)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	entriesTmp := make([]any, 0, len(policies))
	for i := range policies {
		expiresAt := policies[i].EndTime
		if expiresAt == -1 {
			expiresAt = 0
		} else {
			expiresAt *= 1000
		}
		entriesTmp = append(entriesTmp, map[string]any{
			"id": policies[i].ID,
			"resource": map[string]any{
				"id":   policies[i].ResourceID,
				"type": policies[i].ResourceType,
				"name": policies[i].ResourceName,
			},
			"operation": map[string]any{
				"allow": p.operationArrayToJsonWithObligations(policies[i].Operation.Allow),
				"deny":  p.operationArrayToJson(policies[i].Operation.Deny),
			},
			"condition":  policies[i].Condition,
			"expires_at": rest.TimeStampToString(expiresAt),
		})
	}

	resp := map[string]any{
		"total_count": count,
		"entries":     entriesTmp,
	}

	if len(param.Include) > 0 {
		includeRespJson := map[string]any{}
		for _, includeType := range param.Include {
			switch includeType {
			case interfaces.PolicyIncludeObligationType:
				tmp := []any{}
				for i := range includeResp.ObligationTypes {
					tmp = append(tmp, map[string]any{
						"id":          includeResp.ObligationTypes[i].ID,
						"name":        includeResp.ObligationTypes[i].Name,
						"description": includeResp.ObligationTypes[i].Description,
						"schema":      includeResp.ObligationTypes[i].Schema,
					})
				}
				includeRespJson["obligation_types"] = tmp
			case interfaces.PolicyIncludeObligation:
				tmp := []any{}
				for i := range includeResp.Obligations {
					tmp = append(tmp, map[string]any{
						"id":          includeResp.Obligations[i].ID,
						"type_id":     includeResp.Obligations[i].TypeID,
						"name":        includeResp.Obligations[i].Name,
						"description": includeResp.Obligations[i].Description,
						"value":       includeResp.Obligations[i].Value,
					})
				}
				includeRespJson["obligations"] = tmp
			}
		}
		resp["include"] = includeRespJson
	}

	rest.ReplyOK(c, http.StatusOK, resp)
}

func (p *policyRestHandler) getObligations(obligationsJson any) (result []interfaces.PolicyObligationItem, err error) {
	obligations := obligationsJson.([]any)
	for _, v := range obligations {
		obligationMap := v.(map[string]any)
		obligationID := ""
		obligationIDJson, idExist := obligationMap["id"]
		if idExist {
			obligationID = obligationIDJson.(string)
		}
		var obligationValue any
		obligationValueJson, valueExist := obligationMap["value"]
		if valueExist {
			obligationValue = obligationValueJson
		}
		if idExist && valueExist {
			// 不可以同时存在
			err = gerrors.NewError(gerrors.PublicBadRequest, "obligation id and value cannot be both set")
			return
		}
		obligation := interfaces.PolicyObligationItem{
			TypeID: obligationMap["type_id"].(string),
			ID:     obligationID,
			Value:  obligationValue,
		}
		result = append(result, obligation)
	}
	return
}

func (p *policyRestHandler) operationArrayToJson(operations []interfaces.PolicyOperationItem) (resp []any) {
	resp = make([]any, 0, len(operations))
	for i := range operations {
		operationItem := make(map[string]any)
		operationItem["id"] = operations[i].ID
		operationItem["name"] = operations[i].Name
		resp = append(resp, operationItem)
	}
	return
}

func (p *policyRestHandler) operationArrayToJsonWithObligations(operations []interfaces.PolicyOperationItem) (resp []any) {
	resp = make([]any, 0, len(operations))
	for i := range operations {
		operationItem := make(map[string]any)
		operationItem["id"] = operations[i].ID
		operationItem["name"] = operations[i].Name
		obligations := make([]map[string]any, 0, len(operations[i].Obligations))
		for j := range operations[i].Obligations {
			obligationItem := make(map[string]any)
			obligationItem["type_id"] = operations[i].Obligations[j].TypeID
			if operations[i].Obligations[j].ID != "" {
				obligationItem["id"] = operations[i].Obligations[j].ID
			} else {
				obligationItem["value"] = operations[i].Obligations[j].Value
			}
			obligations = append(obligations, obligationItem)
		}
		if len(obligations) > 0 {
			operationItem["obligations"] = obligations
		}
		resp = append(resp, operationItem)
	}
	return
}

func (p *policyRestHandler) ancestorsStrToInfo(ancestorsJson any) (result []interfaces.Ancestor) {
	ancestors := ancestorsJson.([]any)
	result = make([]interfaces.Ancestor, 0, len(ancestors))
	for _, v := range ancestors {
		ancestorMap := v.(map[string]any)
		ancestor := interfaces.Ancestor{
			ID:   ancestorMap["id"].(string),
			Type: ancestorMap["type"].(string),
			Name: ancestorMap["name"].(string),
		}
		result = append(result, ancestor)
	}
	return
}

//nolint:dupl
func (p *policyRestHandler) getResourcePolicy(c *gin.Context) {
	visitor, err := verify(c, p.hydra)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}
	parm := interfaces.ResourcePolicyPagination{
		ResourceID:   c.Query("resource_id"),
		ResourceType: c.Query("resource_type"),
	}
	// 必传参数校验
	if parm.ResourceID == "" {
		err = gerrors.NewError(gerrors.PublicBadRequest, "resource_id is required")
		rest.ReplyErrorV2(c, err)
		return
	}
	if parm.ResourceType == "" {
		err = gerrors.NewError(gerrors.PublicBadRequest, "resource_type is required")
		rest.ReplyErrorV2(c, err)
		return
	}

	info, err := getListQueryParam(c)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}
	parm.Offset = info.offset
	parm.Limit = info.limit

	// 获取include参数
	includeStr := c.QueryArray("include")
	for _, v := range includeStr {
		includeType, ok := p.includeStrToType[v]
		if !ok {
			err = gerrors.NewError(gerrors.PublicBadRequest, "include is invalid")
			rest.ReplyErrorV2(c, err)
			return
		}
		parm.Include = append(parm.Include, includeType)
	}

	count, policies, includeResp, err := p.policy.GetResourcePolicy(context.Background(), &visitor, parm)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	entriesTmp := make([]any, 0, len(policies))
	for i := range policies {
		expiresAt := policies[i].EndTime
		if expiresAt == -1 {
			expiresAt = 0
		} else {
			expiresAt *= 1000
		}
		entriesTmp = append(entriesTmp, map[string]any{
			"id": policies[i].ID,
			"accessor": map[string]any{
				"id":          policies[i].AccessorID,
				"type":        p.accessorTypeToStr[policies[i].AccessorType],
				"name":        policies[i].AccessorName,
				"parent_deps": policies[i].ParentDeps,
			},
			"operation": map[string]any{
				"allow": p.operationArrayToJsonWithObligations(policies[i].Operation.Allow),
				"deny":  p.operationArrayToJson(policies[i].Operation.Deny),
			},
			"condition":  policies[i].Condition,
			"expires_at": rest.TimeStampToString(expiresAt),
		})
	}

	resp := map[string]any{
		"total_count": count,
		"entries":     entriesTmp,
	}

	if len(parm.Include) > 0 {
		includeRespJson := map[string]any{}
		for _, includeType := range parm.Include {
			switch includeType {
			case interfaces.PolicyIncludeObligationType:
				tmp := []any{}
				for i := range includeResp.ObligationTypes {
					tmp = append(tmp, map[string]any{
						"id":          includeResp.ObligationTypes[i].ID,
						"name":        includeResp.ObligationTypes[i].Name,
						"description": includeResp.ObligationTypes[i].Description,
						"schema":      includeResp.ObligationTypes[i].Schema,
					})
				}
				includeRespJson["obligation_types"] = tmp
			case interfaces.PolicyIncludeObligation:
				tmp := []any{}
				for i := range includeResp.Obligations {
					tmp = append(tmp, map[string]any{
						"id":          includeResp.Obligations[i].ID,
						"type_id":     includeResp.Obligations[i].TypeID,
						"name":        includeResp.Obligations[i].Name,
						"description": includeResp.Obligations[i].Description,
						"value":       includeResp.Obligations[i].Value,
					})
				}
				includeRespJson["obligations"] = tmp
			}
		}
		resp["include"] = includeRespJson
	}

	rest.ReplyOK(c, http.StatusOK, resp)
}
