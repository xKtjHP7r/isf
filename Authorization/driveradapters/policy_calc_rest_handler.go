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

var (
	//go:embed jsonschema/policy_calc/check.json
	checkSchemaStr string
	//go:embed jsonschema/policy_calc/resource_list.json
	resourceListSchemaStr string
	//go:embed jsonschema/policy_calc/resource_filter.json
	resourceFilterSchemaStr string
	//go:embed jsonschema/policy_calc/resource_operation.json
	resourceOperationSchemaStr string
	//go:embed jsonschema/policy_calc/check_public.json
	checkPublicSchemaStr string
	//go:embed jsonschema/policy_calc/resource_operation_public.json
	resourceOperationPublicSchemaStr string
	//go:embed jsonschema/policy_calc/resource_type_operation_public.json
	resourceTypeOperationPublicSchemaStr string
)

var (
	policyCalcOnce    sync.Once
	policyCalcHandler RestHandler
)

type policyCalcRestHandler struct {
	policyCalc                        interfaces.LogicsPolicyCalc
	hydra                             interfaces.Hydra
	checkSchema                       *gojsonschema.Schema
	resourceListSchema                *gojsonschema.Schema
	resourceFilterSchema              *gojsonschema.Schema
	resourceOperationSchema           *gojsonschema.Schema
	checkPublicSchema                 *gojsonschema.Schema
	resourceOperationPublicSchema     *gojsonschema.Schema
	resourceTypeOperationPublicSchema *gojsonschema.Schema
	visitorStrToType                  map[string]interfaces.VisitorType
	includeStrToType                  map[string]interfaces.PolicCalcyIncludeType
}

// NewPolicyCalcRestHandler 策略计算适配器接口
func NewPolicyCalcRestHandler() RestHandler {
	policyCalcOnce.Do(func() {
		policyCalcHandler = &policyCalcRestHandler{
			policyCalc:                        logics.NewPolicyCalc(),
			hydra:                             newHydra(),
			checkSchema:                       newJSONSchema(checkSchemaStr),
			resourceListSchema:                newJSONSchema(resourceListSchemaStr),
			resourceFilterSchema:              newJSONSchema(resourceFilterSchemaStr),
			resourceOperationSchema:           newJSONSchema(resourceOperationSchemaStr),
			checkPublicSchema:                 newJSONSchema(checkPublicSchemaStr),
			resourceOperationPublicSchema:     newJSONSchema(resourceOperationPublicSchemaStr),
			resourceTypeOperationPublicSchema: newJSONSchema(resourceTypeOperationPublicSchemaStr),
			visitorStrToType: map[string]interfaces.VisitorType{
				"user": interfaces.RealName,
				"app":  interfaces.App,
			},
			includeStrToType: map[string]interfaces.PolicCalcyIncludeType{
				"operation_obligations": interfaces.PolicCalcyIncludeOperationObligations,
			},
		}
	})
	return policyCalcHandler
}

// RegisterPrivate 注册内部API
func (p *policyCalcRestHandler) RegisterPrivate(engine *gin.Engine) {
	engine.POST("/api/authorization/v1/operation-check", p.check)
	engine.POST("/api/authorization/v1/resource-list", p.resourceList)
	engine.POST("/api/authorization/v1/resource-filter", p.resourceFilter)
	engine.POST("/api/authorization/v1/resource-operation", p.resourceOperation)
}

// RegisterPublic 注册外部API
func (p *policyCalcRestHandler) RegisterPublic(engine *gin.Engine) {
	engine.POST("/api/authorization/v1/operation-check", p.checkPublic)
	engine.POST("/api/authorization/v1/resource-operation", p.resourceOperationPublic)
	engine.POST("/api/authorization/v1/resource-type-operation", p.resourceTypeOperationPublic)
}

func (p *policyCalcRestHandler) checkPublic(c *gin.Context) {
	visitor, err := verify(c, p.hydra)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	var jsonReq map[string]any
	if err = validateAndBindGin(c, p.checkPublicSchema, &jsonReq); err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	// 客体
	resourceJson := jsonReq["resource"].(map[string]any)
	resourceID := resourceJson["id"].(string)
	resourceType := resourceJson["type"].(string)
	ancestorsJson, ok := resourceJson["ancestors"]
	var ancestors []interfaces.Ancestor
	if ok {
		ancestors = p.ancestorsStrToInfo(ancestorsJson)
	}

	var resourceName string
	nameJson, ok := resourceJson["name"]
	if ok {
		resourceName = nameJson.(string)
	}

	accessor := interfaces.AccessorInfo{
		ID:   visitor.ID,
		Type: visitor.Type,
	}

	resource := interfaces.ResourceInfo{
		ID:        resourceID,
		Type:      resourceType,
		Name:      resourceName,
		Ancestors: ancestors,
	}

	operationsJson := jsonReq["operation"].([]any)
	operationsStr := make([]string, 0, len(operationsJson))
	for _, operation := range operationsJson {
		operationsStr = append(operationsStr, operation.(string))
	}

	includeParams := make([]interfaces.PolicCalcyIncludeType, 0)
	checkResult, err := p.policyCalc.Check(context.Background(), &resource, &accessor, operationsStr, includeParams)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	rest.ReplyOK(c, http.StatusOK, gin.H{
		"result": checkResult.Result,
	})
}

func (p *policyCalcRestHandler) check(c *gin.Context) {
	var err error
	var jsonReq map[string]any
	if err = validateAndBindGin(c, p.checkSchema, &jsonReq); err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	// 客体
	accessorJson := jsonReq["accessor"].(map[string]any)
	accessorID := accessorJson["id"].(string)
	accessorType := p.visitorStrToType[accessorJson["type"].(string)]
	resourceJson := jsonReq["resource"].(map[string]any)
	resourceID := resourceJson["id"].(string)
	resourceType := resourceJson["type"].(string)

	// 祖先信息 ,先判断是否 包含
	ancestorsJson, ok := resourceJson["ancestors"]
	var ancestors []interfaces.Ancestor
	if ok {
		ancestors = p.ancestorsStrToInfo(ancestorsJson)
	}

	var resourceName string
	nameJson, ok := resourceJson["name"]
	if ok {
		resourceName = nameJson.(string)
	}

	accessor := interfaces.AccessorInfo{
		ID:   accessorID,
		Type: accessorType,
	}

	resource := interfaces.ResourceInfo{
		ID:        resourceID,
		Type:      resourceType,
		Name:      resourceName,
		Ancestors: ancestors,
	}

	operationsJson := jsonReq["operation"].([]any)
	operationsStr := make([]string, 0, len(operationsJson))
	for _, operation := range operationsJson {
		operationsStr = append(operationsStr, operation.(string))
	}

	includeJson, ok := jsonReq["include"]
	includeParams := make([]interfaces.PolicCalcyIncludeType, 0)
	includeMap := map[interfaces.PolicCalcyIncludeType]bool{}
	if ok {
		includeInfo := includeJson.([]any)
		for _, v := range includeInfo {
			vStr := v.(string)
			if _, ok := p.includeStrToType[vStr]; !ok {
				err = gerrors.NewError(gerrors.PublicBadRequest, "include type not found :"+vStr)
				rest.ReplyErrorV2(c, err)
				return
			}
			includeParams = append(includeParams, p.includeStrToType[vStr])
			includeMap[p.includeStrToType[vStr]] = true
		}
	}

	checkResult, err := p.policyCalc.Check(context.Background(), &resource, &accessor, operationsStr, includeParams)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	oblistResult := make([]any, 0, len(checkResult.OperatrionOblist))
	resp := map[string]any{
		"result": checkResult.Result,
	}
	// 收集include数据
	inlcudeResp := map[string]any{}
	if includeMap[interfaces.PolicCalcyIncludeOperationObligations] {
		for k, list := range checkResult.OperatrionOblist {
			obligations := make([]any, 0, len(list))
			for _, item := range list {
				obligations = append(obligations, map[string]any{
					"type_id": item.TypeID,
					"value":   item.Value,
				})
			}
			if len(obligations) == 0 {
				continue
			}
			oblistResult = append(oblistResult, map[string]any{
				"operation":   k,
				"obligations": obligations,
			})
		}
		inlcudeResp["operation_obligations"] = oblistResult
	}

	if len(includeParams) > 0 {
		resp["include"] = inlcudeResp
	}

	rest.ReplyOK(c, http.StatusOK, resp)
}

func (p *policyCalcRestHandler) resourceFilter(c *gin.Context) {
	var err error
	var jsonReq map[string]any
	if err = validateAndBindGin(c, p.resourceFilterSchema, &jsonReq); err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	// 客体
	accessorJson := jsonReq["accessor"].(map[string]any)
	accessorID := accessorJson["id"].(string)
	accessorType := p.visitorStrToType[accessorJson["type"].(string)]
	resourcesJson := jsonReq["resources"].([]any)
	resources := make([]interfaces.ResourceInfo, 0, len(resourcesJson))
	for _, resource := range resourcesJson {
		resourceJson := resource.(map[string]any)
		resourceID := resourceJson["id"].(string)
		resourceType := resourceJson["type"].(string)
		ancestorsJson, ok := resourceJson["ancestors"]
		var ancestors []interfaces.Ancestor
		if ok {
			ancestors = p.ancestorsStrToInfo(ancestorsJson)
		}
		resources = append(resources, interfaces.ResourceInfo{
			ID:        resourceID,
			Type:      resourceType,
			Ancestors: ancestors,
		})
	}

	operationsJson := jsonReq["operation"].([]any)
	operationsStr := make([]string, 0, len(operationsJson))
	for _, operation := range operationsJson {
		operationsStr = append(operationsStr, operation.(string))
	}

	var allowOperation bool
	allowOperationJson, ok := jsonReq["allow_operation"]
	if ok {
		allowOperation = allowOperationJson.(bool)
	}

	accessor := interfaces.AccessorInfo{
		ID:   accessorID,
		Type: accessorType,
	}

	includeJson, ok := jsonReq["include"]
	includeParams := make([]interfaces.PolicCalcyIncludeType, 0)
	includeMap := map[interfaces.PolicCalcyIncludeType]bool{}
	if ok {
		includeInfo := includeJson.([]any)
		for _, v := range includeInfo {
			vStr := v.(string)
			if _, ok := p.includeStrToType[vStr]; !ok {
				err = gerrors.NewError(gerrors.PublicBadRequest, "include type not found :"+vStr)
				rest.ReplyErrorV2(c, err)
				return
			}
			includeParams = append(includeParams, p.includeStrToType[vStr])
			includeMap[p.includeStrToType[vStr]] = true
		}
	}
	// 资源过滤
	result, resourceOperationMap, resourceOperationObligationMap, err := p.policyCalc.ResourceFilter(context.Background(), resources, &accessor, operationsStr, includeParams)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	resp := []any{}
	for _, resource := range result {
		operationObligationMap := resourceOperationObligationMap[resource.ID]
		oblistResult := make([]any, 0, len(operationObligationMap))
		// 收集include数据
		inlcudeResp := map[string]any{}
		if includeMap[interfaces.PolicCalcyIncludeOperationObligations] {
			for k, list := range operationObligationMap {
				obligations := make([]any, 0, len(list))
				for _, item := range list {
					obligations = append(obligations, map[string]any{
						"type_id": item.TypeID,
						"value":   item.Value,
					})
				}
				if len(obligations) == 0 {
					continue
				}
				oblistResult = append(oblistResult, map[string]any{
					"operation":   k,
					"obligations": obligations,
				})
			}
			inlcudeResp["operation_obligations"] = oblistResult
		}

		respOne := map[string]any{
			"id": resource.ID,
		}
		if len(includeParams) > 0 {
			respOne["include"] = inlcudeResp
		}

		if allowOperation {
			respOne["allow_operation"] = resourceOperationMap[resource.ID]
		}
		resp = append(resp, respOne)
	}
	rest.ReplyOK(c, http.StatusOK, resp)
}

func (p *policyCalcRestHandler) resourceOperation(c *gin.Context) {
	var err error
	var jsonReq map[string]any
	if err = validateAndBindGin(c, p.resourceOperationSchema, &jsonReq); err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	// 客体
	accessorJson := jsonReq["accessor"].(map[string]any)
	accessorID := accessorJson["id"].(string)
	accessorType := p.visitorStrToType[accessorJson["type"].(string)]
	resourcesJson := jsonReq["resources"].([]any)
	resources := make([]interfaces.ResourceInfo, 0, len(resourcesJson))
	for _, resource := range resourcesJson {
		resourceJson := resource.(map[string]any)
		resourceID := resourceJson["id"].(string)
		resourceType := resourceJson["type"].(string)
		ancestorsJson, ok := resourceJson["ancestors"]
		var ancestors []interfaces.Ancestor
		if ok {
			ancestors = p.ancestorsStrToInfo(ancestorsJson)
		}

		resources = append(resources, interfaces.ResourceInfo{
			ID:        resourceID,
			Type:      resourceType,
			Ancestors: ancestors,
		})
	}

	accessor := interfaces.AccessorInfo{
		ID:   accessorID,
		Type: accessorType,
	}

	resourceOperationMap, resourceOperationObligationMap, err := p.policyCalc.GetResourceOperation(context.Background(), resources, &accessor)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	resp := []any{}
	for resourceID, operations := range resourceOperationMap {
		operationObligationMap := resourceOperationObligationMap[resourceID]
		oblistResult := make([]any, 0, len(operationObligationMap))
		for k, list := range operationObligationMap {
			obligations := make([]any, 0, len(list))
			for _, item := range list {
				obligations = append(obligations, map[string]any{
					"type_id": item.TypeID,
					"value":   item.Value,
				})
			}
			if len(obligations) == 0 {
				continue
			}
			oblistResult = append(oblistResult, map[string]any{
				"operation":   k,
				"obligations": obligations,
			})
		}
		resp = append(resp, map[string]any{
			"id":                    resourceID,
			"operation":             operations,
			"operation_obligations": oblistResult,
		})
	}
	rest.ReplyOK(c, http.StatusOK, resp)
}

func (p *policyCalcRestHandler) resourceOperationPublic(c *gin.Context) {
	visitor, err := verify(c, p.hydra)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	var jsonReq map[string]any
	if err = validateAndBindGin(c, p.resourceOperationPublicSchema, &jsonReq); err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	resourcesJson := jsonReq["resources"].([]any)
	resources := make([]interfaces.ResourceInfo, 0, len(resourcesJson))
	for _, resource := range resourcesJson {
		resourceJson := resource.(map[string]any)
		resourceID := resourceJson["id"].(string)
		resourceType := resourceJson["type"].(string)
		ancestorsJson, ok := resourceJson["ancestors"]
		var ancestors []interfaces.Ancestor
		if ok {
			ancestors = p.ancestorsStrToInfo(ancestorsJson)
		}

		resources = append(resources, interfaces.ResourceInfo{
			ID:        resourceID,
			Type:      resourceType,
			Ancestors: ancestors,
		})
	}

	accessor := interfaces.AccessorInfo{
		ID:   visitor.ID,
		Type: visitor.Type,
	}

	resourceOperationMap, _, err := p.policyCalc.GetResourceOperation(context.Background(), resources, &accessor)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	resp := []any{}
	for resourceID, operations := range resourceOperationMap {
		resp = append(resp, map[string]any{
			"id":        resourceID,
			"operation": operations,
		})
	}
	rest.ReplyOK(c, http.StatusOK, resp)
}

func (p *policyCalcRestHandler) resourceTypeOperationPublic(c *gin.Context) {
	visitor, err := verify(c, p.hydra)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	var jsonReq map[string]any
	if err = validateAndBindGin(c, p.resourceTypeOperationPublicSchema, &jsonReq); err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	resourceTypesJson := jsonReq["resource_types"].([]any)
	resourceTypes := make([]string, 0, len(resourceTypesJson))
	for _, resourceType := range resourceTypesJson {
		resourceTypes = append(resourceTypes, resourceType.(string))
	}

	accessor := interfaces.AccessorInfo{
		ID:   visitor.ID,
		Type: visitor.Type,
	}

	resourceOperationMap, err := p.policyCalc.GetResourceTypeOperation(context.Background(), resourceTypes, &accessor)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	resp := []any{}
	for resourceType, operations := range resourceOperationMap {
		resp = append(resp, map[string]any{
			"resource_type": resourceType,
			"operation":     operations,
		})
	}
	rest.ReplyOK(c, http.StatusOK, resp)
}

func (p *policyCalcRestHandler) resourceList(c *gin.Context) {
	var err error
	var jsonReq map[string]any
	if err = validateAndBindGin(c, p.resourceListSchema, &jsonReq); err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	// 客体
	accessorJson := jsonReq["accessor"].(map[string]any)
	accessorID := accessorJson["id"].(string)
	accessorType := p.visitorStrToType[accessorJson["type"].(string)]
	resourceJson := jsonReq["resource"].(map[string]any)
	resourceTypeID := resourceJson["type"].(string)

	accessor := interfaces.AccessorInfo{
		ID:   accessorID,
		Type: accessorType,
	}

	operationsJson := jsonReq["operation"].([]any)
	operationsStr := make([]string, 0, len(operationsJson))
	for _, operation := range operationsJson {
		operationsStr = append(operationsStr, operation.(string))
	}

	includeJson, ok := jsonReq["include"]
	includeParams := make([]interfaces.PolicCalcyIncludeType, 0)
	includeMap := map[interfaces.PolicCalcyIncludeType]bool{}
	if ok {
		includeInfo := includeJson.([]any)
		for _, v := range includeInfo {
			vStr := v.(string)
			if _, ok := p.includeStrToType[vStr]; !ok {
				err = gerrors.NewError(gerrors.PublicBadRequest, "include type not found :"+vStr)
				rest.ReplyErrorV2(c, err)
				return
			}
			includeParams = append(includeParams, p.includeStrToType[vStr])
			includeMap[p.includeStrToType[vStr]] = true
		}
	}
	// 资源过滤
	resourceList, resourceOperationObligationMap, err := p.policyCalc.GetResourceList(context.Background(), resourceTypeID, &accessor, operationsStr, includeParams)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	result := []map[string]any{}
	for _, resource := range resourceList {
		operationObligationMap := resourceOperationObligationMap[resource.ID]
		oblistResult := make([]any, 0, len(operationObligationMap))
		// 收集include数据
		inlcudeResp := map[string]any{}
		if includeMap[interfaces.PolicCalcyIncludeOperationObligations] {
			for k, list := range operationObligationMap {
				obligations := make([]any, 0, len(list))
				for _, item := range list {
					obligations = append(obligations, map[string]any{
						"type_id": item.TypeID,
						"value":   item.Value,
					})
				}
				if len(obligations) == 0 {
					continue
				}
				oblistResult = append(oblistResult, map[string]any{
					"operation":   k,
					"obligations": obligations,
				})
			}
			inlcudeResp["operation_obligations"] = oblistResult
		}

		respOne := map[string]any{
			"id": resource.ID,
		}
		if len(includeParams) > 0 {
			respOne["include"] = inlcudeResp
		}
		result = append(result, respOne)
	}
	rest.ReplyOK(c, http.StatusOK, result)
}

func (p *policyCalcRestHandler) ancestorsStrToInfo(ancestorsJson any) (result []interfaces.Ancestor) {
	ancestors := ancestorsJson.([]any)
	result = make([]interfaces.Ancestor, 0, len(ancestors))
	for _, v := range ancestors {
		ancestorMap := v.(map[string]any)
		ancestor := interfaces.Ancestor{
			ID:   ancestorMap["id"].(string),
			Type: ancestorMap["type"].(string),
		}
		result = append(result, ancestor)
	}
	return
}
