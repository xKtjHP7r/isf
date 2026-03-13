package driveradapters

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"sync"

	"github.com/kweaver-ai/go-lib/rest"

	"Authorization/common"
	"Authorization/interfaces"
	"Authorization/logics"
)

// InitData 初始化数据接口
type InitData interface {
	// InitResourceType 初始化资源类型
	InitResourceType()
	// InitRole 初始化角色
	InitRole()
	// InitRoleMembers 初始化角色成员
	InitRoleMembers()
	// InitPolicy 初始化策略
	InitPolicy()
	// InitObligationType 初始化义务类型
	InitObligationType()
}

var (
	initDataOnce sync.Once
	i            InitData
)

var (
	//go:embed init_data/resource_type.json
	resourceTypeDataStr string
	//go:embed init_data/role.json
	roleDataStr string
	//go:embed init_data/role_members.json
	roleMembersStr string
	//go:embed init_data/policy/*.json
	policyDataDirectory embed.FS
	//go:embed init_data/obligation_type/*.json
	obligationTypeDataDirectory embed.FS
)

type initData struct {
	log               common.Logger
	resourceType      interfaces.LogicsResourceType
	role              interfaces.LogicsRole
	policy            interfaces.LogicsPolicy
	obligationType    interfaces.ObligationType
	memberStringTypes map[string]interfaces.AccessorType
	accessorStrToType map[string]interfaces.AccessorType
}

func NewInitData() InitData {
	initDataOnce.Do(func() {
		i = &initData{
			log:            common.NewLogger(),
			resourceType:   logics.NewResourceType(),
			role:           logics.NewLogicsRole(),
			policy:         logics.NewPolicy(),
			obligationType: logics.NewObligationType(),
			memberStringTypes: map[string]interfaces.AccessorType{
				"user":       interfaces.AccessorUser,
				"department": interfaces.AccessorDepartment,
				"group":      interfaces.AccessorGroup,
				"app":        interfaces.AccessorApp,
			},
			accessorStrToType: map[string]interfaces.AccessorType{
				// 用户、部门、用户组、角色、应用账户
				"user":       interfaces.AccessorUser,
				"department": interfaces.AccessorDepartment,
				"group":      interfaces.AccessorGroup,
				"role":       interfaces.AccessorRole,
				"app":        interfaces.AccessorApp,
			},
		}
	})

	return i
}

//nolint:dupl
func (i *initData) InitResourceType() {
	resourceTypeData := []interfaces.ResourceType{}
	var resourceTypesJson []any
	err := json.Unmarshal([]byte(resourceTypeDataStr), &resourceTypesJson)
	if err != nil {
		i.log.Errorf("unmarshal resourceTypeDataStr failed, err: %v", err)
		return
	}
	for _, resourceTypeJson := range resourceTypesJson {
		resourceTypeDr := resourceTypeJson.(map[string]any)
		resourceType := interfaces.ResourceType{}
		resourceType.ID = resourceTypeDr["id"].(string)
		resourceType.Name = resourceTypeDr["name"].(string)
		resourceType.Description = resourceTypeDr["description"].(string)
		resourceType.InstanceURL = resourceTypeDr["instance_url"].(string)
		resourceType.DataStruct = resourceTypeDr["data_struct"].(string)
		// 判断hidden是否存在，如果存在则设置为hidden
		hiddenJson, ok := resourceTypeDr["hidden"]
		if ok {
			resourceType.Hidden = hiddenJson.(bool)
		}
		operationsJson := resourceTypeDr["operation"].([]any)
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
		resourceTypeData = append(resourceTypeData, resourceType)
	}
	err = i.resourceType.InitResourceTypes(context.Background(), resourceTypeData)
	if err != nil {
		i.log.Errorf("InitResourceTypes  failed, err: %v", err)
		return
	}
}

func (i *initData) InitRole() {
	roleData := []interfaces.RoleInfo{}
	var rolesJson []any
	err := json.Unmarshal([]byte(roleDataStr), &rolesJson)
	if err != nil {
		i.log.Errorf("unmarshal roleDataStr failed, err: %v", err)
		return
	}
	roleSourceMap := map[string]interfaces.RoleSource{
		"system":   interfaces.RoleSourceSystem,
		"business": interfaces.RoleSourceBusiness,
		"user":     interfaces.RoleSourceUser,
	}
	for _, roleJson := range rolesJson {
		roleDr := roleJson.(map[string]any)
		var roleID string
		roleIDJson, ok := roleDr["id"]
		if ok {
			roleID = roleIDJson.(string)
		}
		roleName := roleDr["name"].(string)
		roleDescription := roleDr["description"].(string)
		roleSource := roleSourceMap[roleDr["source"].(string)]

		roleResourceTypeScopesInfoJson := roleDr["resource_type_scope"].(map[string]any)
		roleResourceTypeScopesInfo := interfaces.ResourceTypeScopeInfo{}
		roleResourceTypeScopesInfo.Unlimited = roleResourceTypeScopesInfoJson["unlimited"].(bool)
		typesJson, ok := roleResourceTypeScopesInfoJson["types"]
		if ok {
			roleResourceTypeScopesInfo.Types = make([]interfaces.ResourceTypeScope, 0, len(typesJson.([]any)))
			for _, roleResourceTypeScopeInfoJson := range typesJson.([]any) {
				roleResourceTypeScopeInfoDr := roleResourceTypeScopeInfoJson.(map[string]any)
				roleResourceTypeScopesInfo.Types = append(roleResourceTypeScopesInfo.Types, interfaces.ResourceTypeScope{
					ResourceTypeID:   roleResourceTypeScopeInfoDr["id"].(string),
					ResourceTypeName: roleResourceTypeScopeInfoDr["name"].(string),
				})
			}
		}

		roleData = append(roleData, interfaces.RoleInfo{
			ID:                    roleID,
			Name:                  roleName,
			Description:           roleDescription,
			RoleSource:            roleSource,
			ResourceTypeScopeInfo: roleResourceTypeScopesInfo,
		})
	}
	err = i.role.InitRoles(context.Background(), roleData)
	if err != nil {
		i.log.Errorf("InitRoles failed, err: %v", err)
		return
	}
}

func (i *initData) InitRoleMembers() {
	var rolesJson []any
	err := json.Unmarshal([]byte(roleMembersStr), &rolesJson)
	if err != nil {
		i.log.Errorf("unmarshal roleMembersStr failed, err: %v", err)
		return
	}
	roleMap := make(map[string][]interfaces.RoleMemberInfo)
	for _, roleJson := range rolesJson {
		roleDr := roleJson.(map[string]any)
		roleID := roleDr["role_id"].(string)

		membersJson := roleDr["members"].([]any)
		for _, memberJson := range membersJson {
			tmp := interfaces.RoleMemberInfo{}
			memberDn := memberJson.(map[string]any)
			tmp.ID = memberDn["id"].(string)
			tmp.MemberType = i.memberStringTypes[memberDn["type"].(string)]
			tmp.Name = memberDn["name"].(string)
			roleMap[roleID] = append(roleMap[roleID], tmp)
		}
	}
	err = i.role.InitRoleMemebers(context.Background(), roleMap)
	if err != nil {
		i.log.Errorf("InitRoleMembers failed, err: %v", err)
		return
	}
}

func (i *initData) InitPolicy() {
	policyData := []interfaces.PolicyInfo{}
	var policiesJson []any

	err := fs.WalkDir(policyDataDirectory, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".json" {
			policyData, err := policyDataDirectory.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read policy file %s failed, err: %w", path, err)
			}
			var policyJson []any
			err = json.Unmarshal(policyData, &policyJson)
			if err != nil {
				return fmt.Errorf("unmarshal policy file %s failed, err: %w", path, err)
			}
			policiesJson = append(policiesJson, policyJson...)
		}
		return nil
	})
	if err != nil {
		i.log.Errorf("walk policy directory failed, err: %v", err)
		return
	}

	for _, policyJson := range policiesJson {
		policyDr := policyJson.(map[string]any)
		policy := interfaces.PolicyInfo{}
		expiresAtJson, ok := policyDr["expires_at"]
		if !ok {
			policy.EndTime = -1
		} else {
			// 权限到期时间
			var timeStamp int64
			timeStamp, err = rest.StringToTimeStamp(expiresAtJson.(string))
			if err != nil {
				i.log.Errorf("StringToTimeStamp failed, err: %v", err)
				continue
			}
			// 数据库中 -1 表示永久 单位使用微妙
			if timeStamp == 0 {
				policy.EndTime = -1
			} else {
				policy.EndTime = timeStamp / 1000
			}
		}

		resourceJson := policyDr["resource"].(map[string]any)
		accessorJson := policyDr["accessor"].(map[string]any)

		policy.ResourceID = resourceJson["id"].(string)
		policy.ResourceType = resourceJson["type"].(string)
		policy.ResourceName = resourceJson["name"].(string)
		policy.AccessorID = accessorJson["id"].(string)
		policy.AccessorType = i.accessorStrToType[accessorJson["type"].(string)]
		policy.AccessorName = accessorJson["name"].(string)

		rulesJson := policyDr["rules"].(map[string]any)
		allowJson := rulesJson["allow"].([]any)
		denyJson := rulesJson["deny"].([]any)
		allow := []interfaces.PolicyRuleItem{}
		deny := []interfaces.PolicyRuleItem{}
		for _, v := range allowJson {
			item := v.(map[string]any)
			operationsJson := item["operations"].([]any)
			operations := make([]interfaces.PolicyOperationItem, 0, len(operationsJson))
			for _, operationJson := range operationsJson {
				operation := operationJson.(map[string]any)
				operations = append(operations, interfaces.PolicyOperationItem{
					ID: operation["id"].(string),
				})
			}
			allow = append(allow, interfaces.PolicyRuleItem{
				Condition:  item["condition"].(map[string]any),
				Operations: operations,
			})
		}
		for _, v := range denyJson {
			item := v.(map[string]any)
			operationsJson := item["operations"].([]any)
			operations := make([]interfaces.PolicyOperationItem, 0, len(operationsJson))
			for _, operationJson := range operationsJson {
				operation := operationJson.(map[string]any)
				operations = append(operations, interfaces.PolicyOperationItem{
					ID: operation["id"].(string),
				})
			}
			deny = append(deny, interfaces.PolicyRuleItem{
				Condition:  item["condition"].(map[string]any),
				Operations: operations,
			})
		}
		policy.Rules = interfaces.PolicyRules{
			Allow: allow,
			Deny:  deny,
		}
		policyData = append(policyData, policy)
	}
	err = i.policy.InitPolicy(context.Background(), policyData)
	if err != nil {
		i.log.Errorf("InitPolicy failed, err: %v", err)
		return
	}
}

func (i *initData) InitObligationType() {
	obligationTypeData := []interfaces.ObligationTypeInfo{}
	var obligationTypesJson []any

	err := fs.WalkDir(obligationTypeDataDirectory, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".json" {
			obligationTypeData, err := obligationTypeDataDirectory.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read  file %s failed, err: %w", path, err)
			}
			var tmpJson []any
			err = json.Unmarshal(obligationTypeData, &tmpJson)
			if err != nil {
				return fmt.Errorf("unmarshal file %s failed, err: %w", path, err)
			}
			obligationTypesJson = append(obligationTypesJson, tmpJson...)
		}
		return nil
	})
	if err != nil {
		i.log.Errorf("walk obligationType directory failed, err: %v", err)
		return
	}

	for _, obligationTypeJson := range obligationTypesJson {
		obligationTypeDr := obligationTypeJson.(map[string]any)
		obligationType := interfaces.ObligationTypeInfo{
			ID:     obligationTypeDr["id"].(string),
			Name:   obligationTypeDr["name"].(string),
			Schema: obligationTypeDr["schema"],
		}

		defJson, ok := obligationTypeDr["default_value"]
		if ok {
			obligationType.DefaultValue = defJson
		}

		descriptionJson, ok := obligationTypeDr["description"]
		if ok {
			obligationType.Description = descriptionJson.(string)
		}

		uiSchemaJson, ok := obligationTypeDr["ui_schema"]
		if ok {
			obligationType.UiSchema = uiSchemaJson
		}

		resourceTypeScopesJson := obligationTypeDr["applicable_resource_types"].(map[string]any)
		obligationType.ResourceTypeScope.Unlimited = resourceTypeScopesJson["unlimited"].(bool)

		// 如果资源类型有范围限制，则需要设置资源类型范围
		if !obligationType.ResourceTypeScope.Unlimited {
			_, resourceTypesExist := resourceTypeScopesJson["resource_types"]
			if !resourceTypesExist {
				// 错误数据直接跳过
				i.log.Errorf("InitObligationType InitObligationType ID %s resource_types is required", obligationType.ID)
				continue
			}
			resourceTypesJson := resourceTypeScopesJson["resource_types"].([]any)
			for _, resourceTypeJson := range resourceTypesJson {
				// 遍历资源类型，每个资源类型信息 放入 resourceTypeScope
				var resourceTypeScope interfaces.ObligationResourceTypeScope
				resourceTypeJsonMap := resourceTypeJson.(map[string]any)
				resourceTypeID := resourceTypeJsonMap["id"].(string)
				operationsScopeJson := resourceTypeJsonMap["applicable_operations"].(map[string]any)
				var operationsScopeInfo interfaces.ObligationOperationsScopeInfo
				operationsScopeInfo.Unlimited = operationsScopeJson["unlimited"].(bool)
				if !operationsScopeInfo.Unlimited {
					_, operationsExist := operationsScopeJson["operations"]
					if !operationsExist {
						// 错误数据直接跳过
						i.log.Errorf("InitObligationType InitObligationType ID %s resource_types %s operations is required", obligationType.ID, resourceTypeID)
						continue
					}
					// 获取资源类型上的操作
					operationsJson := operationsScopeJson["operations"].([]any)
					for _, operationJson := range operationsJson {
						operationJsonMap := operationJson.(map[string]any)
						operationID := operationJsonMap["id"].(string)
						var operation interfaces.ObligationOperation
						operation.ID = operationID
						operationsScopeInfo.Operations = append(operationsScopeInfo.Operations, operation)
					}
				}
				resourceTypeScope.ResourceTypeID = resourceTypeID
				resourceTypeScope.OperationsScope = operationsScopeInfo
				// 将资源类型信息放入资源类型范围
				obligationType.ResourceTypeScope.Types = append(obligationType.ResourceTypeScope.Types, resourceTypeScope)
			}
		}
		obligationTypeData = append(obligationTypeData, obligationType)
	}

	err = i.obligationType.InitObligationTypes(context.Background(), obligationTypeData)
	if err != nil {
		i.log.Errorf("InitObligationTypes failed, err: %v", err)
		return
	}
}
