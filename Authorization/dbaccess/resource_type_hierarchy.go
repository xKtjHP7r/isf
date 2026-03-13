// Package dbaccess resource type hierarchy Anyshare 数据访问层 - 资源类型层级关系数据库操作
package dbaccess

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/kweaver-ai/proton-rds-sdk-go/sqlx"

	"Authorization/common"
	"Authorization/interfaces"
)

type resourceTypeHierarchy struct {
	db     *sqlx.DB
	logger common.Logger
}

var (
	rthOnce sync.Once
	rthDB   *resourceTypeHierarchy
)

// NewResourceTypeHierarchy 创建数据库操作对象--和资源类型层级关系相关
func NewResourceTypeHierarchy() *resourceTypeHierarchy {
	rthOnce.Do(func() {
		rthDB = &resourceTypeHierarchy{
			db:     dbPool,
			logger: common.NewLogger(),
		}
	})

	return rthDB
}

// Set 设置资源类型层级关系
func (r *resourceTypeHierarchy) Set(ctx context.Context, hierarchy *interfaces.ResourceTypeHierarchy) error {
	curTime := common.GetCurrentMicrosecondTimestamp()

	// 将整个 hierarchy 转换为 JSON 字符串
	hierarchyStr, err := convertHierarchyChildrenToJSONString(hierarchy.Children)
	if err != nil {
		r.logger.Errorf("Set: json.Marshal hierarchy error: %v", err)
		return err
	}

	resourceTypeID := hierarchy.ResourceTypeID

	// 先判断是否存在
	strSQL := "select f_resource_type_id from " + common.GetDBName(databaseName) + ".t_resource_type_hierarchy where f_resource_type_id = ?"
	rows, err := r.db.Query(strSQL, resourceTypeID)
	if err != nil {
		r.logger.Errorf("Set: query error: %v", err)
		return err
	}
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				r.logger.Errorln(rowsErr)
			}
			if closeErr := rows.Close(); closeErr != nil {
				r.logger.Errorln(closeErr)
			}
		}
	}()

	if rows.Next() {
		// 更新
		strSQL = "update " + common.GetDBName(databaseName) +
			".t_resource_type_hierarchy set f_children = ?, f_modified_at = ? where f_resource_type_id = ?"
		_, err = r.db.Exec(strSQL, hierarchyStr, curTime, resourceTypeID)
	} else {
		// 插入
		strSQL = "insert into " + common.GetDBName(databaseName) +
			".t_resource_type_hierarchy(f_resource_type_id, f_children, f_created_at, f_modified_at) values(?,?,?,?)"
		_, err = r.db.Exec(strSQL, resourceTypeID, hierarchyStr, curTime, curTime)
	}
	if err != nil {
		r.logger.Errorf("Set: exec error: %v", err)
		return err
	}
	return nil
}

// Get 获取资源类型层级关系
func (r *resourceTypeHierarchy) GetAll(ctx context.Context) (hierarchyMap map[string]interfaces.ResourceTypeHierarchy, err error) {
	hierarchyMap = make(map[string]interfaces.ResourceTypeHierarchy)
	strSQL := "select f_resource_type_id, f_children from " + common.GetDBName(databaseName) + ".t_resource_type_hierarchy"
	rows, err := r.db.Query(strSQL)
	if err != nil {
		r.logger.Errorf("Get: query error: %v", err)
		return hierarchyMap, err
	}
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				r.logger.Errorln(rowsErr)
			}
			if closeErr := rows.Close(); closeErr != nil {
				r.logger.Errorln(closeErr)
			}
		}
	}()

	for rows.Next() {
		childStr := ""
		resourceTypeID := ""
		err = rows.Scan(&resourceTypeID, &childStr)
		if err != nil {
			r.logger.Errorf("Get: scan error: %v", err)
			return hierarchyMap, err
		}
		hierarchy := interfaces.ResourceTypeHierarchy{
			ResourceTypeID: resourceTypeID,
			Children:       nil,
		}
		hierarchy.Children, err = convertJSONStringToHierarchyChildren(childStr)
		if err != nil {
			r.logger.Errorf("GetAll: convertJSONStringToHierarchyChildren error: %v", err)
			return hierarchyMap, err
		}
		r.logger.Debugf("GetAll: resourceTypeID: %s, hierarchy: %v", resourceTypeID, hierarchy)
		hierarchyMap[resourceTypeID] = hierarchy
	}
	return hierarchyMap, nil
}

// convertHierarchyChildrenToJSONString 将 hierarchy.Children 转换为 JSON string，字段名使用下划线格式
func convertHierarchyChildrenToJSONString(children []interfaces.ResourceTypeHierarchy) (string, error) {
	jsonItems := convertHierarchyToJSONItems(children)
	jsonBytes, err := json.Marshal(jsonItems)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

// convertHierarchyToJSONItems 递归转换 ResourceTypeHierarchy 为 any
func convertHierarchyToJSONItems(children []interfaces.ResourceTypeHierarchy) []any {
	result := make([]any, 0, len(children))
	for _, child := range children {
		item := map[string]any{
			"resource_type_id": child.ResourceTypeID,
			"children":         convertHierarchyToJSONItems(child.Children),
		}
		result = append(result, item)
	}
	return result
}

// convertJSONStringToHierarchyChildren 将 JSON string 转换为 []interfaces.ResourceTypeHierarchy，字段名使用下划线格式
func convertJSONStringToHierarchyChildren(jsonStr string) ([]interfaces.ResourceTypeHierarchy, error) {
	var jsonItems []any
	err := json.Unmarshal([]byte(jsonStr), &jsonItems)
	if err != nil {
		return nil, err
	}
	return convertJSONItemsToHierarchy(jsonItems), nil
}

// convertJSONItemsToHierarchy 递归转换 any 为 []interfaces.ResourceTypeHierarchy
func convertJSONItemsToHierarchy(jsonItems []any) []interfaces.ResourceTypeHierarchy {
	if jsonItems == nil {
		return []interfaces.ResourceTypeHierarchy{}
	}
	result := make([]interfaces.ResourceTypeHierarchy, 0, len(jsonItems))
	for _, item := range jsonItems {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		resourceTypeID, _ := itemMap["resource_type_id"].(string)
		var children []interfaces.ResourceTypeHierarchy
		if childrenAny, ok := itemMap["children"]; ok {
			if childrenArray, ok := childrenAny.([]any); ok {
				children = convertJSONItemsToHierarchy(childrenArray)
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
