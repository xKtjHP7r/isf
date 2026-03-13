// Package logics resource type hierarchy Anyshare 业务逻辑层 -资源类型层级关系
package logics

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"

	gerrors "github.com/kweaver-ai/go-lib/error"

	"Authorization/common"
	"Authorization/interfaces"
)

var (
	resourceTypeHierarchyOnce      sync.Once
	resourceTypeHierarchySingleton *resourceTypeHierarchy
)

var (
	//go:embed init_data/resource_type_hierarchy.json
	resourceTypeHierarchyInitDataStr string
)

type resourceTypeHierarchy struct {
	db             interfaces.DBResourceTypeHierarchy
	dbResourceType interfaces.DBResourceType
	logger         common.Logger
}

// LoadResourceTypeHierarchyFromInitData 从 resourceTypeHierarchyInitDataStr 加载 []interfaces.ResourceTypeHierarchy
// 支持 children 的嵌套场景
func (r *resourceTypeHierarchy) LoadResourceTypeHierarchyFromInitData(initDataStr string) ([]interfaces.ResourceTypeHierarchy, error) {
	var jsonItems []any
	err := json.Unmarshal([]byte(initDataStr), &jsonItems)
	if err != nil {
		r.logger.Errorf("LoadResourceTypeHierarchyFromInitData: json.Unmarshal error: %v", err)
		return nil, err
	}
	return r.convertJSONItemsToResourceTypeHierarchy(jsonItems), nil
}

// convertJSONItemsToResourceTypeHierarchy 递归转换 any 为 []interfaces.ResourceTypeHierarchy
func (r *resourceTypeHierarchy) convertJSONItemsToResourceTypeHierarchy(jsonItems []any) []interfaces.ResourceTypeHierarchy {
	if jsonItems == nil {
		r.logger.Errorf("LoadResourceTypeHierarchyFromInitData: convertJSONItemsToResourceTypeHierarchy: jsonItems is nil")
		return []interfaces.ResourceTypeHierarchy{}
	}
	result := make([]interfaces.ResourceTypeHierarchy, 0, len(jsonItems))
	for _, item := range jsonItems {
		itemMap, ok := item.(map[string]any)
		if !ok {
			r.logger.Errorf("LoadResourceTypeHierarchyFromInitData: convertJSONItemsToResourceTypeHierarchy: item is not map[string]any")
			continue
		}
		resourceTypeID, _ := itemMap["resource_type_id"].(string)
		r.logger.Errorf("LoadResourceTypeHierarchyFromInitData: convertJSONItemsToResourceTypeHierarchy: resourceTypeID: %s", resourceTypeID)
		var children []interfaces.ResourceTypeHierarchy
		if childrenAny, ok := itemMap["children"]; ok {
			if childrenArray, ok := childrenAny.([]any); ok {
				children = r.convertJSONItemsToResourceTypeHierarchy(childrenArray)
			}
		}
		hierarchy := interfaces.ResourceTypeHierarchy{
			ResourceTypeID: resourceTypeID,
			Children:       children,
		}
		result = append(result, hierarchy)
	}
	return result
}

// NewResourceTypeHierarchy 创建新的NewResourceTypeHierarchy对象
func NewResourceTypeHierarchy() *resourceTypeHierarchy {
	resourceTypeHierarchyOnce.Do(func() {
		resourceTypeHierarchySingleton = &resourceTypeHierarchy{
			db:             dbResourceTypeHierarchy,
			dbResourceType: dbResourceType,
			logger:         common.NewLogger(),
		}
		err := resourceTypeHierarchySingleton.Init(context.Background())
		if err != nil {
			resourceTypeHierarchySingleton.logger.Errorf("NewResourceTypeHierarchy: Init error: %v", err)
		}
	})
	return resourceTypeHierarchySingleton
}

// Set 设置资源类型层级关系
func (r *resourceTypeHierarchy) SetPrivate(ctx context.Context, _ *interfaces.Visitor, hierarchie *interfaces.ResourceTypeHierarchy) error {
	// 递归检查, 同一层级是否有重复 ,重复报错 统一处理
	err := r.checkDuplicate(ctx, hierarchie)
	if err != nil {
		return err
	}
	err = r.db.Set(ctx, hierarchie)
	if err != nil {
		r.logger.Errorf("SetPrivate: db.Set error: %v", err)
		return err
	}
	return nil
}

func (r *resourceTypeHierarchy) checkDuplicate(ctx context.Context, hierarchie *interfaces.ResourceTypeHierarchy) error {
	// 检查资源类型 是否存在
	resourceTypes, err := r.dbResourceType.GetAllInternalWithHidden(ctx)
	if err != nil {
		return err
	}

	resourceTypeInfoMap := make(map[string]interfaces.ResourceType, len(resourceTypes))
	for _, resourceTypeInfo := range resourceTypes {
		resourceTypeInfoMap[resourceTypeInfo.ID] = resourceTypeInfo
	}

	var ok bool
	_, ok = resourceTypeInfoMap[hierarchie.ResourceTypeID]
	if !ok {
		des := fmt.Sprintf("resource type not found: %s", hierarchie.ResourceTypeID)
		return gerrors.NewError(gerrors.PublicBadRequest, des)
	}

	// 检查当前层级的重复
	checkDupMap := make(map[string]bool)
	for _, child := range hierarchie.Children {
		_, ok = resourceTypeInfoMap[child.ResourceTypeID]
		if !ok {
			des := fmt.Sprintf("resource type not found: %s", child.ResourceTypeID)
			return gerrors.NewError(gerrors.PublicBadRequest, des)
		}
		if _, ok := checkDupMap[child.ResourceTypeID]; ok {
			return gerrors.NewError(gerrors.PublicBadRequest, "同一层级不能有重复的资源类型")
		}
		checkDupMap[child.ResourceTypeID] = true
	}

	// 递归检查每个子节点
	for _, child := range hierarchie.Children {
		if err := r.checkDuplicate(ctx, &child); err != nil {
			return err
		}
	}
	return nil
}

// findInChildren 递归在子节点中查找指定的 resourceTypeID，找到第一个匹配的即返回
func (r *resourceTypeHierarchy) findInChildren(children []interfaces.ResourceTypeHierarchy, resourceTypeID string) (interfaces.ResourceTypeHierarchy, bool) {
	for _, child := range children {
		// 如果当前节点匹配，直接返回
		if child.ResourceTypeID == resourceTypeID {
			return child, true
		}
		// 递归查找子节点
		if found, ok := r.findInChildren(child.Children, resourceTypeID); ok {
			return found, true
		}
	}
	return interfaces.ResourceTypeHierarchy{}, false
}

// Get 获取资源类型层级关系
func (r *resourceTypeHierarchy) Get(ctx context.Context, _ *interfaces.Visitor, resourceTypeID string) (hierarchy interfaces.ResourceTypeHierarchy, err error) {
	// 获取数据库中 所有的层级关系，有可能不是根节点
	r.logger.Debugf("Get: resourceTypeID: %s", resourceTypeID)
	resourceTypeHierarchyMap, err := r.db.GetAll(ctx)
	if err != nil {
		r.logger.Errorf("Get: db.GetAll error: %v", err)
		return interfaces.ResourceTypeHierarchy{}, err
	}

	// 先检查是否为根节点
	resourceTypeHierarchy, ok := resourceTypeHierarchyMap[resourceTypeID]
	if ok {
		r.logger.Debugf("Get: resourceTypeID found in root: %s", resourceTypeID)
		return resourceTypeHierarchy, nil
	}

	// 如果不是根节点，递归在所有子节点中查找
	for _, hierarchy := range resourceTypeHierarchyMap {
		if found, ok := r.findInChildren(hierarchy.Children, resourceTypeID); ok {
			r.logger.Debugf("Get: resourceTypeID found in children: %s, found: %v", resourceTypeID, found)
			return found, nil
		}
	}

	// 如果都没找到，返回空的 ResourceTypeHierarchy
	r.logger.Debugf("Get: resourceTypeID not found: %s", resourceTypeID)
	return interfaces.ResourceTypeHierarchy{}, nil
}

// HasHierarchy 判断一个资源类型是否有设置过层级关系
func (r *resourceTypeHierarchy) HasHierarchy(ctx context.Context, _ *interfaces.Visitor, resourceTypeID string) (hasHierarchy bool, err error) {
	// 获取所有资源类型层级关系
	resourceTypeHierarchyMap, err := r.db.GetAll(ctx)
	if err != nil {
		r.logger.Errorf("HasHierarchy GetAll: %v", err)
		return false, err
	}

	// 遍历所有根节点，递归查找是否存在层级关系
	for parentID, hierarchy := range resourceTypeHierarchyMap {
		if parentID == resourceTypeID {
			return true, nil
		}
		if _, hasHierarchy = r.findInChildren(hierarchy.Children, resourceTypeID); hasHierarchy {
			return true, nil
		}
	}
	return false, nil
}

// GetAll 获取所有资源类型层级关系
func (r *resourceTypeHierarchy) GetAll(ctx context.Context) (resourceTypeHierarchyMap map[string]interfaces.ResourceTypeHierarchy, err error) {
	return r.db.GetAll(ctx)
}

// Init 初始化资源类型层级关系
func (r *resourceTypeHierarchy) Init(ctx context.Context) error {
	resourceTypeHierarchys, err := r.LoadResourceTypeHierarchyFromInitData(resourceTypeHierarchyInitDataStr)
	if err != nil {
		r.logger.Errorf("Init: LoadResourceTypeHierarchyFromInitData error: %v", err)
		return err
	}

	for _, resourceTypeHierarchy := range resourceTypeHierarchys {
		err = r.SetPrivate(ctx, nil, &resourceTypeHierarchy)
		if err != nil {
			r.logger.Errorf("Init: SetPrivate error: %v", err)
			return err
		}
	}
	return nil
}
