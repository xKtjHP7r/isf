// Package dbaccess 数据访问层
package dbaccess

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"

	"github.com/kweaver-ai/proton-rds-sdk-go/sqlx"

	"Authorization/common"
	"Authorization/interfaces"
)

type resourceType struct {
	db     *sqlx.DB
	logger common.Logger
}

var (
	resourceTypeOnce    sync.Once
	resourceTypeService *resourceType
)

// NewResource 创建数据库对象
func NewResource() *resourceType {
	resourceTypeOnce.Do(func() {
		resourceTypeService = &resourceType{
			db:     dbPool,
			logger: common.NewLogger(),
		}
	})
	return resourceTypeService
}

func (d *resourceType) GetPagination(ctx context.Context, params interfaces.ResourceTypePagination) (count int, resources []interfaces.ResourceType, err error) {
	countRows, err := d.db.Query("select count(1) from " + common.GetDBName(databaseName) + ".t_resource_type where f_hidden = 0")
	if err != nil {
		d.logger.Errorln(err)
		return 0, nil, err
	}
	for countRows.Next() {
		err = countRows.Scan(&count)
		if err != nil {
			d.logger.Errorln(err)
			return 0, nil, err
		}
	}

	if countRows != nil {
		if countRowsErr := countRows.Err(); countRowsErr != nil {
			d.logger.Errorln(countRowsErr)
		}
		if closeErr := countRows.Close(); closeErr != nil {
			d.logger.Errorln(closeErr)
		}
	}

	var rows *sql.Rows
	strSQL := "select f_id, f_name, f_description, f_instance_url, f_data_struct, f_operation, f_create_time, f_modify_time from " +
		common.GetDBName(databaseName) + ".t_resource_type where f_hidden = 0 order by f_primary_id asc limit ? offset ? "
	rows, err = d.db.Query(strSQL, params.Limit, params.Offset)
	if err != nil {
		d.logger.Errorln(err)
		return 0, nil, err
	}

	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				d.logger.Errorln(rowsErr)
			}

			if closeErr := rows.Close(); closeErr != nil {
				d.logger.Errorln(closeErr)
			}
		}
	}()

	for rows.Next() {
		var resource interfaces.ResourceType
		operationStr := ""
		err = rows.Scan(&resource.ID, &resource.Name, &resource.Description, &resource.InstanceURL, &resource.DataStruct, &operationStr, &resource.CreateTime, &resource.ModifyTime)
		if err != nil {
			d.logger.Errorln(err)
			return count, nil, err
		}
		err = json.Unmarshal([]byte(operationStr), &resource.Operation)
		if err != nil {
			d.logger.Errorln(err)
			return count, nil, err
		}
		resources = append(resources, resource)
	}
	return count, resources, nil
}

func (d *resourceType) Set(ctx context.Context, resource *interfaces.ResourceType) (err error) {
	curTime := common.GetCurrentMicrosecondTimestamp()
	// resource.Operation 转成string
	operationByte, err := json.Marshal(resource.Operation)
	if err != nil {
		d.logger.Errorln(err)
		return err
	}
	// 先判断是否存在
	strSQL := "select f_id from " + common.GetDBName(databaseName) + ".t_resource_type where f_id = ?"
	rows, err := d.db.Query(strSQL, resource.ID)
	if err != nil {
		d.logger.Errorln(err)
		return err
	}
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				d.logger.Errorln(rowsErr)
			}
		}
	}()

	hidden := 0
	if resource.Hidden {
		hidden = 1
	}
	if rows.Next() {
		strSQL = "update " + common.GetDBName(databaseName) +
			".t_resource_type set f_name = ?, f_description = ?, f_instance_url = ?, f_data_struct = ?, f_operation = ?, f_hidden = ?, f_modify_time = ? where f_id = ?"
		_, err = d.db.Exec(strSQL, resource.Name, resource.Description, resource.InstanceURL, resource.DataStruct, string(operationByte), hidden, curTime, resource.ID)
	} else {
		strSQL = "insert into " + common.GetDBName(databaseName) +
			".t_resource_type(f_id, f_name, f_description, f_instance_url, f_data_struct, f_operation, f_hidden, f_create_time, f_modify_time) values(?,?,?,?,?,?,?,?,?)"
		_, err = d.db.Exec(strSQL, resource.ID, resource.Name, resource.Description, resource.InstanceURL,
			resource.DataStruct, string(operationByte), hidden, curTime, curTime)
	}
	if err != nil {
		d.logger.Errorln(err)
		return err
	}
	return
}

func (d *resourceType) Delete(ctx context.Context, resourceID string) (err error) {
	strSQL := "delete from " + common.GetDBName(databaseName) + ".t_resource_type where f_id = ?"
	_, err = d.db.Exec(strSQL, resourceID)
	if err != nil {
		d.logger.Errorln(err)
		return err
	}
	return
}

func (d *resourceType) GetByIDs(ctx context.Context, resourceTypeIDs []string) (resourceMap map[string]interfaces.ResourceType, err error) {
	resourceMap = make(map[string]interfaces.ResourceType)
	if len(resourceTypeIDs) == 0 {
		return resourceMap, nil
	}
	IDsSet, IDsGroup := getFindInSetSQL(resourceTypeIDs)
	strSQL := "select f_id, f_name, f_description, f_instance_url, f_data_struct, f_operation from " + common.GetDBName(databaseName) + ".t_resource_type where f_id in (" + IDsSet + ")"
	rows, err := d.db.Query(strSQL, IDsGroup...)
	if err != nil {
		d.logger.Errorln(err)
		return nil, err
	}
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				d.logger.Errorln(rowsErr)
			}
			if closeErr := rows.Close(); closeErr != nil {
				d.logger.Errorln(closeErr)
			}
		}
	}()

	for rows.Next() {
		var resource interfaces.ResourceType
		operationStr := ""
		err = rows.Scan(&resource.ID, &resource.Name, &resource.Description, &resource.InstanceURL, &resource.DataStruct, &operationStr)
		if err != nil {
			d.logger.Errorln(err)
			return nil, err
		}
		err = json.Unmarshal([]byte(operationStr), &resource.Operation)
		if err != nil {
			d.logger.Errorln(err)
			return nil, err
		}
		resourceMap[resource.ID] = resource
	}
	return resourceMap, nil
}

// 获取所有资源类型, 不包含隐藏的资源类型
//
//nolint:dupl
func (d *resourceType) GetAllInternal(ctx context.Context) (resourceTypes []interfaces.ResourceType, err error) {
	strSQL := "select f_id, f_name, f_description, f_instance_url, f_data_struct, f_operation from " + common.GetDBName(databaseName) + ".t_resource_type where f_hidden = 0"
	rows, err := d.db.Query(strSQL)
	if err != nil {
		d.logger.Errorln(err)
		return nil, err
	}
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				d.logger.Errorln(rowsErr)
			}
			if closeErr := rows.Close(); closeErr != nil {
				d.logger.Errorln(closeErr)
			}
		}
	}()

	for rows.Next() {
		var resource interfaces.ResourceType
		operationStr := ""
		err = rows.Scan(&resource.ID, &resource.Name, &resource.Description, &resource.InstanceURL, &resource.DataStruct, &operationStr)
		if err != nil {
			d.logger.Errorln(err)
			return nil, err
		}
		err = json.Unmarshal([]byte(operationStr), &resource.Operation)
		if err != nil {
			d.logger.Errorln(err)
			return nil, err
		}
		resourceTypes = append(resourceTypes, resource)
	}
	return resourceTypes, nil
}

// 获取所有资源类型, 包含隐藏的资源类型
//
//nolint:dupl
func (d *resourceType) GetAllInternalWithHidden(ctx context.Context) (resourceTypes []interfaces.ResourceType, err error) {
	strSQL := "select f_id, f_name, f_description, f_instance_url, f_data_struct, f_operation from " + common.GetDBName(databaseName) + ".t_resource_type"
	rows, err := d.db.Query(strSQL)
	if err != nil {
		d.logger.Errorln(err)
		return nil, err
	}
	defer func() {
		if rows != nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				d.logger.Errorln(rowsErr)
			}
			if closeErr := rows.Close(); closeErr != nil {
				d.logger.Errorln(closeErr)
			}
		}
	}()

	for rows.Next() {
		var resource interfaces.ResourceType
		operationStr := ""
		err = rows.Scan(&resource.ID, &resource.Name, &resource.Description, &resource.InstanceURL, &resource.DataStruct, &operationStr)
		if err != nil {
			d.logger.Errorln(err)
			return nil, err
		}
		err = json.Unmarshal([]byte(operationStr), &resource.Operation)
		if err != nil {
			d.logger.Errorln(err)
			return nil, err
		}
		resourceTypes = append(resourceTypes, resource)
	}
	return resourceTypes, nil
}
