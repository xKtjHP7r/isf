// Package logics obligation type logic
package logics

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"

	gerrors "github.com/kweaver-ai/go-lib/error"
	"github.com/xeipuuv/gojsonschema"

	"Authorization/common"
	"Authorization/interfaces"
)

var (
	obligationTypeOnce      sync.Once
	obligationTypeSingleton *obligationType
)

type obligationType struct {
	db           interfaces.DBObligationType
	userMgnt     interfaces.DrivenUserMgnt
	resourceType interfaces.LogicsResourceType
	logger       common.Logger
}

// NewObligationType 创建新的ObligationType对象
func NewObligationType() *obligationType {
	obligationTypeOnce.Do(func() {
		obligationTypeSingleton = &obligationType{
			db:           dbObligationType,
			userMgnt:     dnUserMgnt,
			resourceType: NewResourceType(),
			logger:       common.NewLogger(),
		}
	})
	return obligationTypeSingleton
}

func (o *obligationType) checkVisitorType(ctx context.Context, visitor *interfaces.Visitor) (err error) {
	// 获取访问者角色
	var roleTypes []interfaces.SystemRoleType

	// 实名用户获取对应角色信息
	if visitor.Type == interfaces.RealName {
		// 获取用户角色信息
		roleTypes, err = o.userMgnt.GetUserRolesByUserID(ctx, visitor.ID)
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

func (o *obligationType) checkResourceTypeAndOperation(ctx context.Context, info *interfaces.ObligationTypeInfo) (err error) {
	//  资源类型 不限制 直接返回
	if info.ResourceTypeScope.Unlimited {
		return
	}
	var resourceTypeInfos []interfaces.ResourceType
	resourceTypeInfos, err = o.resourceType.GetAllInternal(ctx)
	if err != nil {
		return err
	}
	resourceTypeInfoMap := make(map[string]interfaces.ResourceType, len(resourceTypeInfos))
	for _, resourceTypeInfo := range resourceTypeInfos {
		resourceTypeInfoMap[resourceTypeInfo.ID] = resourceTypeInfo
	}

	// 检查资源类型是否存在
	for _, resourceTypeScope := range info.ResourceTypeScope.Types {
		resourceTypeInfo, ok := resourceTypeInfoMap[resourceTypeScope.ResourceTypeID]
		// 不存在 直接返回错误
		if !ok {
			strTmp := fmt.Sprintf("resource type not found: %s", resourceTypeScope.ResourceTypeID)
			err = gerrors.NewError(gerrors.PublicBadRequest, strTmp)
			return
		} else {
			// 检查操作是否 限制
			if resourceTypeScope.OperationsScope.Unlimited {
				continue
			} else {
				operationMap := make(map[string]bool, len(resourceTypeInfo.Operation))
				for _, operation := range resourceTypeInfo.Operation {
					operationMap[operation.ID] = true
				}

				// 检查操作是否存在
				for _, operation := range resourceTypeScope.OperationsScope.Operations {
					if _, ok := operationMap[operation.ID]; !ok {
						strTmp := fmt.Sprintf("resource type %s operation not found: %s", resourceTypeScope.ResourceTypeID, operation.ID)
						err = gerrors.NewError(gerrors.PublicBadRequest, strTmp)
						return
					}
				}
			}
		}
	}
	return
}

// Set 设置义务类型
func (o *obligationType) Set(ctx context.Context, visitor *interfaces.Visitor, info *interfaces.ObligationTypeInfo) (err error) {
	// 角色检查 visitor
	err = o.checkVisitorType(ctx, visitor)
	if err != nil {
		return
	}

	// 校验 是不是合格的jsonSchema
	schema, err := gojsonschema.NewSchema(gojsonschema.NewGoLoader(info.Schema))
	if err != nil {
		err = gerrors.NewError(gerrors.PublicBadRequest, "schema is invalid")
		o.logger.Errorf("Set: %v", err)
		return
	}

	// 检查默认值 是否合法
	if info.DefaultValue != nil {
		var result *gojsonschema.Result
		result, err = schema.Validate(gojsonschema.NewGoLoader(info.DefaultValue))
		if err != nil {
			return gerrors.NewError(gerrors.PublicBadRequest, err.Error())
		}

		if !result.Valid() {
			msgList := make([]string, 0, len(result.Errors()))
			for _, err := range result.Errors() {
				msgList = append(msgList, err.String())
			}
			return gerrors.NewError(gerrors.PublicBadRequest, strings.Join(msgList, "; "))
		}
	}

	// 检查资源类型和操作是否合法
	err = o.checkResourceTypeAndOperation(ctx, info)
	if err != nil {
		return
	}
	err = o.db.Set(ctx, info)
	if err != nil {
		o.logger.Errorf("Set: %v", err)
		return
	}
	return
}

func (o *obligationType) Delete(ctx context.Context, visitor *interfaces.Visitor, obligationTypeID string) (err error) {
	// 角色检查 visitor
	err = o.checkVisitorType(ctx, visitor)
	if err != nil {
		return
	}
	err = o.db.Delete(ctx, obligationTypeID)
	if err != nil {
		o.logger.Errorf("Delete: %v", err)
		return
	}
	return
}

func (o *obligationType) GetByID(ctx context.Context, visitor *interfaces.Visitor, obligationTypeID string) (info interfaces.ObligationTypeInfo, err error) {
	// 角色检查 visitor
	err = o.checkVisitorType(ctx, visitor)
	if err != nil {
		return
	}

	info, err = o.db.GetByID(ctx, obligationTypeID)
	if err != nil {
		o.logger.Errorf("GetByID: %v", err)
		return
	}

	// 义务类型不存在
	if info.ID == "" {
		err = gerrors.NewError(gerrors.PublicNotFound, fmt.Sprintf("obligation_type_id %s not found ", obligationTypeID))
		return
	}

	// 不限制 直接返回
	if info.ResourceTypeScope.Unlimited {
		return
	}

	// 填充资源类型名称 和操作名称
	operationInfoMap, resourceTypeInfoMap, err := o.getResourceTypesOperation(ctx, visitor)
	if err != nil {
		return
	}

	for i := range info.ResourceTypeScope.Types {
		resourceTypeInfo, ok := resourceTypeInfoMap[info.ResourceTypeScope.Types[i].ResourceTypeID]
		// 资源类型不存在 直接跳过，考虑原因，一个资源类型被删除时，数据库里的历史数据不好删除，这里直接过滤
		if !ok {
			continue
		}

		// 资源类型存在
		info.ResourceTypeScope.Types[i].ResourceTypeName = resourceTypeInfo.Name

		// 操作不限制
		if info.ResourceTypeScope.Types[i].OperationsScope.Unlimited {
			continue
		}

		// 填充操作名称
		for j := range info.ResourceTypeScope.Types[i].OperationsScope.Operations {
			operation := info.ResourceTypeScope.Types[i].OperationsScope.Operations[j]
			operationNames := operationInfoMap[info.ResourceTypeScope.Types[i].ResourceTypeID]
			info.ResourceTypeScope.Types[i].OperationsScope.Operations[j].Name = operationNames[operation.ID]
		}
	}
	return
}

func (o *obligationType) getOperationNameByLanguage(operationName []interfaces.OperationName, language string) (name string) {
	for _, name := range operationName {
		if strings.EqualFold(name.Language, language) {
			return name.Value
		}
	}
	return
}

/*
获取资源类型和操作名称

@param ctx context.Context
@param visitor *interfaces.Visitor
@return operationInfoMap map[string]map[string]string 资源类型ID-操作ID-操作名称
@return resourceTypeInfoMap map[string]interfaces.ResourceType 资源类型ID-资源类型信息
@return err error
*/
func (o *obligationType) getResourceTypesOperation(ctx context.Context, visitor *interfaces.Visitor) (
	operationInfoMap map[string]map[string]string, resourceTypeInfoMap map[string]interfaces.ResourceType, err error,
) {
	// 获取资源类型
	resourceTypeInfos, err := o.resourceType.GetAllInternal(ctx)
	if err != nil {
		o.logger.Errorf("getResourceTypesOperation GetByIDsInternal err: %v", err)
		return
	}

	resourceTypeInfoMap = make(map[string]interfaces.ResourceType, len(resourceTypeInfos))
	for _, resourceTypeInfo := range resourceTypeInfos {
		resourceTypeInfoMap[resourceTypeInfo.ID] = resourceTypeInfo
	}

	operationInfoMap = make(map[string]map[string]string, len(resourceTypeInfoMap))
	for resourceType, resourceTypeInfo := range resourceTypeInfoMap {
		opeNames := make(map[string]string, len(resourceTypeInfo.Operation))
		for _, operation := range resourceTypeInfo.Operation {
			// 根据国际化获取名称
			name := o.getOperationNameByLanguage(operation.Name, visitor.Language)
			opeNames[operation.ID] = name
		}
		operationInfoMap[resourceType] = opeNames
	}
	return
}

func (o *obligationType) Get(ctx context.Context, visitor *interfaces.Visitor, info *interfaces.ObligationTypeSearchInfo) (count int, resultInfos []interfaces.ObligationTypeInfo, err error) {
	// 角色检查 visitor
	err = o.checkVisitorType(ctx, visitor)
	if err != nil {
		return
	}

	count, resultInfos, err = o.db.Get(ctx, info)
	if err != nil {
		o.logger.Errorf("Get: %v", err)
		return
	}

	if len(resultInfos) == 0 {
		return
	}

	// 填充资源类型名称 和操作名称
	operationInfoMap, resourceTypeInfoMap, err := o.getResourceTypesOperation(ctx, visitor)
	if err != nil {
		return
	}

	for i := range resultInfos {
		// 不限制 直接返回
		if resultInfos[i].ResourceTypeScope.Unlimited {
			return
		}

		for j := range resultInfos[i].ResourceTypeScope.Types {
			resourceTypeInfo, ok := resourceTypeInfoMap[resultInfos[i].ResourceTypeScope.Types[j].ResourceTypeID]
			// 资源类型不存在 直接跳过，考虑原因，一个资源类型被删除时，数据库里的历史数据不好删除，这里直接过滤
			if !ok {
				continue
			}

			// 资源类型存在
			resultInfos[i].ResourceTypeScope.Types[i].ResourceTypeName = resourceTypeInfo.Name

			// 操作不限制
			if resultInfos[i].ResourceTypeScope.Types[i].OperationsScope.Unlimited {
				continue
			}

			// 填充操作名称
			for k := range resultInfos[i].ResourceTypeScope.Types[j].OperationsScope.Operations {
				operation := resultInfos[i].ResourceTypeScope.Types[j].OperationsScope.Operations[k]
				operationNames := operationInfoMap[resultInfos[i].ResourceTypeScope.Types[j].ResourceTypeID]
				resultInfos[i].ResourceTypeScope.Types[j].OperationsScope.Operations[k].Name = operationNames[operation.ID]
			}
		}
	}
	return
}

// Query 查询义务类型
//
//	queryInfo 是查询条件
//	resultInfos 是查询结果 key是OperationID, value是ObligationTypeInfo列表
func (o *obligationType) Query(ctx context.Context, visitor *interfaces.Visitor,
	queryInfo *interfaces.QueryObligationTypeInfo,
) (resultInfos map[string][]interfaces.ObligationTypeInfo, err error) {
	// 暂时先不检查权限

	// 检查资源类型 是否存在，检查入参的操作是否存在
	resourceTypeInfoMap, err := o.resourceType.GetByIDsInternal(ctx, []string{queryInfo.ResourceType})
	if err != nil {
		o.logger.Errorf("Query GetByIDsInternal err: %v", err)
		return
	}

	if len(resourceTypeInfoMap) != 1 {
		err = gerrors.NewError(gerrors.PublicBadRequest, "resource type not found")
		return
	}
	resourceTypeInfo := resourceTypeInfoMap[queryInfo.ResourceType]

	opeMaps := make(map[string]bool, len(resourceTypeInfo.Operation))
	for _, operation := range resourceTypeInfo.Operation {
		opeMaps[operation.ID] = true
	}

	if len(queryInfo.Operation) == 0 {
		// 默认返回所有操作
		for _, operation := range resourceTypeInfo.Operation {
			queryInfo.Operation = append(queryInfo.Operation, operation.ID)
		}
	} else {
		for _, operation := range queryInfo.Operation {
			if !opeMaps[operation] {
				err = gerrors.NewError(gerrors.PublicBadRequest, "operation not found")
				return
			}
		}
	}

	// 获取数据库的 所有义务
	allObligationTypes, err := o.db.GetAll(ctx)
	if err != nil {
		o.logger.Errorf("Query GetAll err: %v", err)
		return
	}

	resultInfos = o.getOperationSet(queryInfo.ResourceType, queryInfo.Operation, allObligationTypes)
	return
}

func (o *obligationType) getOperationSet(resourceTypeID string, operationIDs []string,
	allObligationTypes []interfaces.ObligationTypeInfo) (resultInfos map[string][]interfaces.ObligationTypeInfo) {
	resultInfos = make(map[string][]interfaces.ObligationTypeInfo, len(operationIDs))
	for _, operation := range operationIDs {
		resultInfos[operation] = make([]interfaces.ObligationTypeInfo, 0)
	}
	// 遍历所有义务类型
	for i := range allObligationTypes {
		// 资源类型不限制, 则每个操作都有该义务
		if allObligationTypes[i].ResourceTypeScope.Unlimited {
			for _, operation := range operationIDs {
				resultInfos[operation] = append(resultInfos[operation], allObligationTypes[i])
			}
			continue
		}

		// 如果资源类型 不包含，直接跳过
		// 遍历资源类型
		resourceTypeFound := false
		resourceTypeTmp := interfaces.ObligationResourceTypeScope{}
		for _, resourceType := range allObligationTypes[i].ResourceTypeScope.Types {
			if resourceType.ResourceTypeID == resourceTypeID {
				resourceTypeFound = true
				resourceTypeTmp = resourceType
				break
			}
		}
		if !resourceTypeFound {
			continue
		}

		// 如果操作不限制 则该义务类型对所有操作都有
		if resourceTypeTmp.OperationsScope.Unlimited {
			for _, operation := range operationIDs {
				resultInfos[operation] = append(resultInfos[operation], allObligationTypes[i])
			}
			continue
		}

		// 如果操作有设置，则加入
		for _, operationTmp := range resourceTypeTmp.OperationsScope.Operations {
			for _, operation := range operationIDs {
				if operationTmp.ID == operation {
					resultInfos[operation] = append(resultInfos[operation], allObligationTypes[i])
				}
			}
		}
	}
	return
}

// QueryV2 查询义务类型V2
//
//	queryInfo 是查询条件，支持多个资源类型
//	resultInfos 是查询结果，第一层key是ResourceTypeID，第二层key是OperationID, value是ObligationTypeInfo列表
func (o *obligationType) QueryV2(ctx context.Context, visitor *interfaces.Visitor,
	queryInfo *interfaces.QueryObligationTypeInfoV2,
) (resultInfos map[string]map[string][]interfaces.ObligationTypeInfo, err error) {
	// 检查资源类型是否存在
	if len(queryInfo.ResourceTypeIDs) == 0 {
		err = gerrors.NewError(gerrors.PublicBadRequest, "resource type ids is empty")
		return
	}

	resourceTypeInfoMap, err := o.resourceType.GetByIDsInternal(ctx, queryInfo.ResourceTypeIDs)
	if err != nil {
		o.logger.Errorf("QueryV2 GetByIDsInternal err: %v", err)
		return
	}

	if len(resourceTypeInfoMap) != len(queryInfo.ResourceTypeIDs) {
		err = gerrors.NewError(gerrors.PublicBadRequest, "some resource types not found")
		return
	}

	// 获取数据库的所有义务类型
	allObligationTypes, err := o.db.GetAll(ctx)
	if err != nil {
		o.logger.Errorf("QueryV2 GetAll err: %v", err)
		return
	}

	resultInfos = make(map[string]map[string][]interfaces.ObligationTypeInfo, len(resourceTypeInfoMap))
	for resourceTypeID, resourceTypeInfo := range resourceTypeInfoMap {
		operations := make([]string, 0, len(resourceTypeInfo.Operation))
		for _, operation := range resourceTypeInfo.Operation {
			operations = append(operations, operation.ID)
		}
		resultInfos[resourceTypeID] = o.getOperationSet(resourceTypeID, operations, allObligationTypes)
	}
	return
}

// 指定ID批量获取义务类型
func (o *obligationType) GetByIDSInternal(ctx context.Context, obligationTypeIDs map[string]bool) (infos []interfaces.ObligationTypeInfo, err error) {
	o.logger.Debugf("GetByIDSInternal obligationTypeIDs: %v", obligationTypeIDs)
	IDs := make([]string, 0, len(obligationTypeIDs))
	for ID := range obligationTypeIDs {
		IDs = append(IDs, ID)
	}
	// 从数据库获取所有义务类型
	infos, err = o.db.GetByIDs(ctx, IDs)
	if err != nil {
		o.logger.Errorf("GetByIDSInternal GetByIDs err: %v", err)
		return
	}
	return
}

// 义务类型批量初始化
func (o *obligationType) InitObligationTypes(ctx context.Context, obligationTypes []interfaces.ObligationTypeInfo) error {
	if len(obligationTypes) == 0 {
		return nil
	}
	for _, obligationType := range obligationTypes {
		info, err := o.db.GetByID(ctx, obligationType.ID)
		if err != nil {
			o.logger.Errorf("InitObligationTypes GetByID: %v", err)
			return err
		}

		// 之前存在 且无变化
		if info.ID != "" && !o.checkObligationTypeChange(&info, &obligationType) {
			continue
		}
		err = o.db.Set(ctx, &obligationType)
		if err != nil {
			o.logger.Errorf("InitObligationTypes Set: %v", err)
			return err
		}
	}
	return nil
}

// checkObligationTypeChange 检查义务类型是否发生变化， 有变化返回true
func (o *obligationType) checkObligationTypeChange(old, newInfo *interfaces.ObligationTypeInfo) bool {
	if old.Name != newInfo.Name {
		return true
	}
	if old.Description != newInfo.Description {
		return true
	}
	if !reflect.DeepEqual(old.ResourceTypeScope, newInfo.ResourceTypeScope) {
		return true
	}
	if !reflect.DeepEqual(old.Schema, newInfo.Schema) {
		return true
	}
	if !reflect.DeepEqual(old.UiSchema, newInfo.UiSchema) {
		return true
	}
	if !reflect.DeepEqual(old.DefaultValue, newInfo.DefaultValue) {
		return true
	}
	return false
}

// Set 设置义务类型
func (o *obligationType) SetPrivate(ctx context.Context, _ *interfaces.Visitor, info *interfaces.ObligationTypeInfo) (err error) {
	// 校验 是不是合格的jsonSchema
	schema, err := gojsonschema.NewSchema(gojsonschema.NewGoLoader(info.Schema))
	if err != nil {
		err = gerrors.NewError(gerrors.PublicBadRequest, "schema is invalid")
		o.logger.Errorf("Set: %v", err)
		return
	}

	// 检查默认值 是否合法
	if info.DefaultValue != nil {
		var result *gojsonschema.Result
		result, err = schema.Validate(gojsonschema.NewGoLoader(info.DefaultValue))
		if err != nil {
			return gerrors.NewError(gerrors.PublicBadRequest, err.Error())
		}

		if !result.Valid() {
			msgList := make([]string, 0, len(result.Errors()))
			for _, err := range result.Errors() {
				msgList = append(msgList, err.String())
			}
			return gerrors.NewError(gerrors.PublicBadRequest, strings.Join(msgList, "; "))
		}
	}

	// 检查资源类型和操作是否合法
	err = o.checkResourceTypeAndOperation(ctx, info)
	if err != nil {
		return
	}
	err = o.db.Set(ctx, info)
	if err != nil {
		o.logger.Errorf("Set: %v", err)
		return
	}
	return
}
