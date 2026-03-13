package logics

import (
	"testing"

	"github.com/kweaver-ai/proton-rds-sdk-go/sqlx"
	"github.com/stretchr/testify/assert"

	"Authorization/interfaces/mock"
)

func TestSetDBPool(t *testing.T) {
	// 创建 mock DB 实例
	mockDB := &sqlx.DB{}

	// 测试设置 DB Pool
	SetDBPool(mockDB)

	// 验证全局变量是否被正确设置
	assert.Equal(t, mockDB, dbPool)
}

func TestSetDBTracePool(t *testing.T) {
	// 创建 mock DB 实例
	mockDB := &sqlx.DB{}

	// 测试设置 DB Trace Pool
	SetDBTracePool(mockDB)

	// 验证全局变量是否被正确设置
	assert.Equal(t, mockDB, dbTracePool)
}

func TestSetDBResourceType(t *testing.T) {
	// 创建 mock DBResourceType 实例
	mockDBResourceType := &mock.MockDBResourceType{}

	// 测试设置 DB Resource Type
	SetDBResourceType(mockDBResourceType)

	// 验证全局变量是否被正确设置
	assert.Equal(t, mockDBResourceType, dbResourceType)
}

func TestSetDBPolicy(t *testing.T) {
	// 创建 mock DBPolicy 实例
	mockDBPolicy := &mock.MockDBPolicy{}

	// 测试设置 DB Policy
	SetDBPolicy(mockDBPolicy)

	// 验证全局变量是否被正确设置
	assert.Equal(t, mockDBPolicy, dbPolicy)
}

func TestSetDBPolicyCalc(t *testing.T) {
	// 创建 mock DBPolicyCalc 实例
	mockDBPolicyCalc := &mock.MockDBPolicyCalc{}

	// 测试设置 DB Policy Calc
	SetDBPolicyCalc(mockDBPolicyCalc)

	// 验证全局变量是否被正确设置
	assert.Equal(t, mockDBPolicyCalc, dbPolicyCalc)
}

func TestSetDBRole(t *testing.T) {
	// 创建 mock DBRole 实例
	mockDBRole := &mock.MockDBRole{}

	// 测试设置 DB Role
	SetDBRole(mockDBRole)

	// 验证全局变量是否被正确设置
	assert.Equal(t, mockDBRole, dbRole)
}

func TestSetDBRoleMember(t *testing.T) {
	// 创建 mock DBRoleMember 实例
	mockDBRoleMember := &mock.MockDBRoleMember{}

	// 测试设置 DB Role Member
	SetDBRoleMember(mockDBRoleMember)

	// 验证全局变量是否被正确设置
	assert.Equal(t, mockDBRoleMember, dbRoleMember)
}

func TestSetFunctionsWithNilValues(t *testing.T) {
	// 测试设置 nil 值
	SetDBPool(nil)
	assert.Nil(t, dbPool)

	SetDBTracePool(nil)
	assert.Nil(t, dbTracePool)

	SetDBResourceType(nil)
	assert.Nil(t, dbResourceType)

	SetDBPolicy(nil)
	assert.Nil(t, dbPolicy)

	SetDBPolicyCalc(nil)
	assert.Nil(t, dbPolicyCalc)

	SetDBRole(nil)
	assert.Nil(t, dbRole)

	SetDBRoleMember(nil)
	assert.Nil(t, dbRoleMember)
}

func TestSetFunctionsWithMultipleCalls(t *testing.T) {
	// 创建多个不同的 mock 实例
	mockDB1 := &sqlx.DB{}
	mockDB2 := &sqlx.DB{}
	mockDBResourceType1 := &mock.MockDBResourceType{}
	mockDBResourceType2 := &mock.MockDBResourceType{}

	// 第一次设置
	SetDBPool(mockDB1)
	SetDBResourceType(mockDBResourceType1)

	// 验证第一次设置
	assert.Equal(t, mockDB1, dbPool)
	assert.Equal(t, mockDBResourceType1, dbResourceType)

	// 第二次设置
	SetDBPool(mockDB2)
	SetDBResourceType(mockDBResourceType2)
}

func TestGlobalVariablesInitialState(t *testing.T) {
	// 测试全局变量的初始状态
	// 注意：这些测试依赖于包的初始化状态
	// 在实际运行中，这些变量可能已经被其他测试设置过

	// 重置全局变量为 nil 进行测试
	originalDBPool := dbPool
	originalDBTracePool := dbTracePool
	originalDBResourceType := dbResourceType
	originalDBPolicy := dbPolicy
	originalDBPolicyCalc := dbPolicyCalc
	originalDBRole := dbRole
	originalDBRoleMember := dbRoleMember

	// 设置测试值
	SetDBPool(nil)
	SetDBTracePool(nil)
	SetDBResourceType(nil)
	SetDBPolicy(nil)
	SetDBPolicyCalc(nil)
	SetDBRole(nil)
	SetDBRoleMember(nil)

	// 验证所有全局变量都被正确设置
	assert.Nil(t, dbPool)
	assert.Nil(t, dbTracePool)
	assert.Nil(t, dbResourceType)
	assert.Nil(t, dbPolicy)
	assert.Nil(t, dbPolicyCalc)
	assert.Nil(t, dbRole)
	assert.Nil(t, dbRoleMember)

	// 恢复原始值
	SetDBPool(originalDBPool)
	SetDBTracePool(originalDBTracePool)
	SetDBResourceType(originalDBResourceType)
	SetDBPolicy(originalDBPolicy)
	SetDBPolicyCalc(originalDBPolicyCalc)
	SetDBRole(originalDBRole)
	SetDBRoleMember(originalDBRoleMember)
}

// Benchmark 测试
func BenchmarkSetDBPool(b *testing.B) {
	mockDB := &sqlx.DB{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SetDBPool(mockDB)
	}
}

func BenchmarkSetDBResourceType(b *testing.B) {
	mockDBResourceType := &mock.MockDBResourceType{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SetDBResourceType(mockDBResourceType)
	}
}

func BenchmarkSetDBPolicy(b *testing.B) {
	mockDBPolicy := &mock.MockDBPolicy{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SetDBPolicy(mockDBPolicy)
	}
}
