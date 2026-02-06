// Package logics perm Anyshare 业务逻辑层 -文档权限
package logics

import (
	"context"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"sync"

	gerrors "github.com/kweaver-ai/go-lib/error"

	"Authorization/common"
	"Authorization/interfaces"
)

var (
	resourceOnce      sync.Once
	resourceSingleton *resourceType
)

const (
	nameMaxLength = 50
)

type resourceType struct {
	db                    interfaces.DBResourceType
	userMgmt              interfaces.DrivenUserMgnt
	resourceTypeHierarchy interfaces.LogicsResourceTypeHierarchy
	logger                common.Logger
}

// NewResourceType 创建新的NewResourceType对象
func NewResourceType() *resourceType {
	resourceOnce.Do(func() {
		resourceSingleton = &resourceType{
			db:                    dbResourceType,
			resourceTypeHierarchy: NewResourceTypeHierarchy(),
			logger:                common.NewLogger(),
			userMgmt:              dnUserMgnt,
		}
	})
	return resourceSingleton
}

func (r *resourceType) checkVisitorType(ctx context.Context, visitor *interfaces.Visitor) (err error) {
	// 获取访问者角色
	var roleTypes []interfaces.SystemRoleType

	// 实名用户获取对应角色信息
	if visitor.Type == interfaces.RealName {
		// 获取用户角色信息
		roleTypes, err = r.userMgmt.GetUserRolesByUserID(ctx, visitor.ID)
		if err != nil {
			return err
		}
	}

	return checkVisitorType(
		visitor,
		roleTypes,
		[]interfaces.VisitorType{interfaces.RealName},
		[]interfaces.SystemRoleType{interfaces.SuperAdmin, interfaces.SystemAdmin, interfaces.SecurityAdmin},
	)
}

// GetPagination 获取资源
func (r *resourceType) GetPagination(ctx context.Context, visitor *interfaces.Visitor, params interfaces.ResourceTypePagination) (count int, resources []interfaces.ResourceType, err error) {
	// 权限检查 visitor
	err = r.checkVisitorType(ctx, visitor)
	if err != nil {
		return
	}

	count, resources, err = r.db.GetPagination(ctx, params)
	if err != nil {
		r.logger.Errorf("GetPagination: %v", err)
		return 0, nil, err
	}
	return count, resources, nil
}

// Set 设置资源
func (r *resourceType) Set(ctx context.Context, visitor *interfaces.Visitor, resourceType *interfaces.ResourceType) (err error) {
	// 权限检查 visitor
	err = r.checkVisitorType(ctx, visitor)
	if err != nil {
		return
	}
	// id 长度 不超过50 ，只能是数字或者字母或者下划线
	if len(resourceType.ID) > nameMaxLength {
		err = gerrors.NewError(gerrors.PublicBadRequest, "id length must less than 50")
		return
	}
	// 数字或者字母或者下划线
	if !regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(resourceType.ID) {
		err = gerrors.NewError(gerrors.PublicBadRequest, "invalid id, only number or letter or underline")
		return
	}

	err = r.db.Set(ctx, resourceType)
	if err != nil {
		r.logger.Errorf("Set: %v", err)
		return
	}
	return
}

// checkResourceTypeChange 检查资源类型是否发生变化， 有变化返回true
func (r *resourceType) checkResourceTypeChange(old, newInfo *interfaces.ResourceType) bool {
	if old.Name != newInfo.Name {
		return true
	}
	if old.InstanceURL != newInfo.InstanceURL {
		return true
	}
	if old.DataStruct != newInfo.DataStruct {
		return true
	}
	if old.Description != newInfo.Description {
		return true
	}
	if old.Hidden != newInfo.Hidden {
		return true
	}
	if !reflect.DeepEqual(old.Operation, newInfo.Operation) {
		return true
	}
	return false
}

// InitResourceTypes 资源类型批量添加
func (r *resourceType) InitResourceTypes(ctx context.Context, resourceTypes []interfaces.ResourceType) (err error) {
	for i := range resourceTypes {
		var info interfaces.ResourceType
		// 如果之前存在，则不添加
		// 所有用户可以调用
		var infoMap map[string]interfaces.ResourceType
		infoMap, err = r.db.GetByIDs(ctx, []string{resourceTypes[i].ID})
		if err != nil {
			r.logger.Errorf("InitResourceTypes GetByIDs: %v", err)
			return
		}
		info, ok := infoMap[resourceTypes[i].ID]
		if ok && !r.checkResourceTypeChange(&info, &resourceTypes[i]) {
			continue
		}
		err = r.db.Set(ctx, &resourceTypes[i])
		if err != nil {
			r.logger.Errorf("InitResourceTypes: %v", err)
			return
		}
	}
	return nil
}

// Delete 删除资源
func (r *resourceType) Delete(ctx context.Context, visitor *interfaces.Visitor, resourceTypeID string) (err error) {
	// 权限检查 visitor
	err = r.checkVisitorType(ctx, visitor)
	if err != nil {
		return
	}
	err = r.db.Delete(ctx, resourceTypeID)
	if err != nil {
		r.logger.Errorf("Delete: %v", err)
		return err
	}
	return nil
}

// GetByID 获取资源
func (r *resourceType) GetByID(ctx context.Context, _ *interfaces.Visitor, resourceTypeID string) (resourceType interfaces.ResourceType, err error) {
	// 所有用户可以调用
	resourceTypeMap, err := r.db.GetByIDs(ctx, []string{resourceTypeID})
	if err != nil {
		r.logger.Errorf("GetByIDs: %v", err)
		return
	}
	resourceType, ok := resourceTypeMap[resourceTypeID]
	if !ok {
		return interfaces.ResourceType{}, nil
	}
	return
}

// GetAllOperation 获取资源类型所有操作
func (r *resourceType) GetAllOperation(ctx context.Context, visitor *interfaces.Visitor, resourceTypeID string, scope interfaces.OperationScopeType) (
	operations []interfaces.ResourceTypeOperationResponse, childrenOperations []interfaces.ChildrenResourceTypeOperation, err error,
) {
	// 所有用户可以调用
	resourceTypes, err := r.db.GetAllInternalWithHidden(ctx)
	if err != nil {
		r.logger.Errorf("GetAllOperation GetByIDs err: %v", err)
		return
	}

	resourceTypeMap := make(map[string]interfaces.ResourceType)
	for _, resourceTypeInfo := range resourceTypes {
		resourceTypeMap[resourceTypeInfo.ID] = resourceTypeInfo
	}

	resourceTypeInfo, ok := resourceTypeMap[resourceTypeID]
	if !ok {
		return
	}

	// 获取当前资源类型的操作信息
	operations = r.getOperations(visitor, &resourceTypeInfo, scope)

	// 请求类型上的操作信息，直接返回
	if scope == interfaces.ScopeType {
		return operations, nil, nil
	}

	// 获取资源类型层级关系
	resourceTypeHierarchy, err := r.resourceTypeHierarchy.Get(ctx, visitor, resourceTypeID)
	if err != nil {
		r.logger.Errorf("GetAllOperation resourceTypeHierarchy Get err: %v", err)
		return
	}

	// 如果当前资源类型没有子资源类型，则直接返回
	if len(resourceTypeHierarchy.Children) == 0 {
		return
	}

	// 获取子资源类型的操作信息
	scopeTmp := interfaces.ScopeType
	childrenOperations = r.getChildrenOperations(visitor, &resourceTypeHierarchy, scopeTmp, resourceTypeMap)
	return
}

func (r *resourceType) getOperationNameByLanguage(language string, operationName []interfaces.OperationName) (name string) {
	for _, name := range operationName {
		if strings.EqualFold(name.Language, language) {
			return name.Value
		}
	}
	return
}

// GetByIDsInternal 批量获取资源
func (r *resourceType) GetByIDsInternal(ctx context.Context, resourceTypeIDs []string) (resourceMap map[string]interfaces.ResourceType, err error) {
	resourceMap, err = r.db.GetByIDs(ctx, resourceTypeIDs)
	if err != nil {
		r.logger.Errorf("GetByIDsInternal: %v", err)
		return nil, err
	}
	return
}

func (r *resourceType) GetAllInternal(ctx context.Context) (resourceTypes []interfaces.ResourceType, err error) {
	resourceTypes, err = r.db.GetAllInternal(ctx)
	if err != nil {
		r.logger.Errorf("GetAllInternal: %v", err)
		return nil, err
	}
	return
}

// SetPrivate 设置资源
func (r *resourceType) SetPrivate(ctx context.Context, resourceType *interfaces.ResourceType) (err error) {
	// id 长度 不超过50 ，只能是数字或者字母或者下划线
	if len(resourceType.ID) > nameMaxLength {
		err = gerrors.NewError(gerrors.PublicBadRequest, "id length must less than 50")
		return
	}
	// 数字或者字母或者下划线
	if !regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(resourceType.ID) {
		err = gerrors.NewError(gerrors.PublicBadRequest, "invalid id, only number or letter or underline")
		return
	}

	// 如果之前存在，则不添加
	// 所有用户可以调用
	var infoMap map[string]interfaces.ResourceType
	infoMap, err = r.db.GetByIDs(ctx, []string{resourceType.ID})
	if err != nil {
		r.logger.Errorf("InitResourceTypes GetByIDs: %v", err)
		return
	}
	// 存在且没变化
	info, ok := infoMap[resourceType.ID]
	if ok && !r.checkResourceTypeChange(&info, resourceType) {
		return
	}

	err = r.db.Set(ctx, resourceType)
	if err != nil {
		r.logger.Errorf("Set: %v", err)
		return
	}
	return
}

// 根据国际化和scope(type, instance)获取操作信息
func (r *resourceType) getOperations(visitor *interfaces.Visitor, resourceTypeInfo *interfaces.ResourceType, scope interfaces.OperationScopeType) (
	operations []interfaces.ResourceTypeOperationResponse,
) {
	operations = make([]interfaces.ResourceTypeOperationResponse, 0, len(resourceTypeInfo.Operation))
	for _, operation := range resourceTypeInfo.Operation {
		opeInfo := interfaces.ResourceTypeOperationResponse{
			ID:          operation.ID,
			Description: operation.Description,
		}
		if slices.Contains(operation.Scope, scope) {
			// 根据国际化获取名称
			opeInfo.Name = r.getOperationNameByLanguage(visitor.Language, operation.Name)
			operations = append(operations, opeInfo)
		}
	}
	r.logger.Infof("visitor.Language: %s, getOperations: %v", visitor.Language, operations)
	return
}

// getChildrenOperations 获取子资源类型操作
// 只返回当前层级的子资源类型操作，不递归获取子节点的子资源类型操作
func (r *resourceType) getChildrenOperations(visitor *interfaces.Visitor, resourceTypeHierarchy *interfaces.ResourceTypeHierarchy,
	scope interfaces.OperationScopeType, resourceTypeMap map[string]interfaces.ResourceType) (
	childrenOperations []interfaces.ChildrenResourceTypeOperation,
) {
	childrenOperations = make([]interfaces.ChildrenResourceTypeOperation, 0, len(resourceTypeHierarchy.Children))
	operationsMap := make(map[string][]interfaces.ResourceTypeOperationResponse)
	for _, child := range resourceTypeHierarchy.Children {
		// 获取子资源类型信息
		childResourceType, ok := resourceTypeMap[child.ResourceTypeID]
		if !ok {
			// 如果资源类型不存在，跳过该子节点
			continue
		}

		// 使用map 暂时存储，避免重复获取操作信息
		operations, ok := operationsMap[child.ResourceTypeID]
		if !ok {
			operations = r.getOperations(visitor, &childResourceType, scope)
			operationsMap[child.ResourceTypeID] = operations
		}
		// 构建子资源类型操作对象
		childOperation := interfaces.ChildrenResourceTypeOperation{
			ResourceTypeID: child.ResourceTypeID,
			Name:           childResourceType.Name,
			Operations:     operations,
			Children:       make([]interfaces.ChildrenResourceTypeOperation, 0, len(child.Children)),
		}

		childrenOperations = append(childrenOperations, childOperation)
	}
	return
}
