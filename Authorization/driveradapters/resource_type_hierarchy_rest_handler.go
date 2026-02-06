// Package driveradapters AnyShare 入站适配器
package driveradapters

import (
	"context"
	_ "embed" // 标准用法
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/xeipuuv/gojsonschema"

	gerrors "devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/go-lib/error"
	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/go-lib/rest"

	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/Authorization/interfaces"
	"devops.aishu.cn/AISHUDevOps/AnyShareFamily/_git/Authorization/logics"
)

//go:embed jsonschema/resource_type_hierarchy/set.json
var setHierarchySchemaStr string

var (
	resourceTypeHierarchyOnce    sync.Once
	resourceTypeHierarchyHandler RestHandler
)

type resourceTypeHierarchyRestHandler struct {
	resourceTypeHierarchy interfaces.LogicsResourceTypeHierarchy
	hydra                 interfaces.Hydra
	setHierarchySchema    *gojsonschema.Schema
}

// NewResourceTypeHierarchyRestHandler 资源类型层级关系适配器接口
func NewResourceTypeHierarchyRestHandler() RestHandler {
	resourceTypeHierarchyOnce.Do(func() {
		resourceTypeHierarchyHandler = &resourceTypeHierarchyRestHandler{
			resourceTypeHierarchy: logics.NewResourceTypeHierarchy(),
			hydra:                 newHydra(),
			setHierarchySchema:    newJSONSchema(setHierarchySchemaStr),
		}
	})
	return resourceTypeHierarchyHandler
}

// RegisterPrivate 注册内部API
func (r *resourceTypeHierarchyRestHandler) RegisterPrivate(engine *gin.Engine) {
	// 暂无内部API
	engine.PUT("/api/authorization/v1/resource_type_hierarchy/:resource_type_id", r.setPrivate)
}

// RegisterPublic 注册外部API
func (r *resourceTypeHierarchyRestHandler) RegisterPublic(_ *gin.Engine) {
}

// set 设置资源类型层级关系
func (r *resourceTypeHierarchyRestHandler) setPrivate(c *gin.Context) {
	resourceTypeID := c.Param("resource_type_id")
	var jsonReqs []any
	if err := validateAndBindGin(c, r.setHierarchySchema, &jsonReqs); err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}

	hierarchies := make([]interfaces.ResourceTypeHierarchy, 0, len(jsonReqs))
	for _, jsonReqJson := range jsonReqs {
		hierarchie, err := r.parseHierarchy(jsonReqJson)
		if err != nil {
			rest.ReplyErrorV2(c, err)
			return
		}
		hierarchies = append(hierarchies, hierarchie)
	}
	hierarchie := &interfaces.ResourceTypeHierarchy{
		ResourceTypeID: resourceTypeID,
		Children:       hierarchies,
	}
	err := r.resourceTypeHierarchy.SetPrivate(context.Background(), nil, hierarchie)
	if err != nil {
		rest.ReplyErrorV2(c, err)
		return
	}
	rest.ReplyOK(c, http.StatusNoContent, gin.H{})
}

// parseHierarchy 递归解析层级关系
func (r *resourceTypeHierarchyRestHandler) parseHierarchy(jsonReq any) (interfaces.ResourceTypeHierarchy, error) {
	jsonReqMap, ok := jsonReq.(map[string]any)
	if !ok {
		return interfaces.ResourceTypeHierarchy{}, gerrors.NewError(gerrors.PublicBadRequest, "invalid hierarchy format: expected object")
	}

	resourceTypeID, ok := jsonReqMap["resource_type_id"].(string)
	if !ok {
		return interfaces.ResourceTypeHierarchy{}, gerrors.NewError(gerrors.PublicBadRequest, "invalid resource_type_id: expected string")
	}

	childrenJson, ok := jsonReqMap["children"].([]any)
	if !ok {
		return interfaces.ResourceTypeHierarchy{}, gerrors.NewError(gerrors.PublicBadRequest, "invalid children: expected array")
	}

	children := make([]interfaces.ResourceTypeHierarchy, 0, len(childrenJson))
	for _, childJson := range childrenJson {
		child, err := r.parseHierarchy(childJson)
		if err != nil {
			return interfaces.ResourceTypeHierarchy{}, err
		}
		children = append(children, child)
	}

	return interfaces.ResourceTypeHierarchy{
		ResourceTypeID: resourceTypeID,
		Children:       children,
	}, nil
}
