package lookup

import (
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

const (
	goodHasIndex  = "[good"
	badHasIndex   = "bad"
	stringTest    = "test"
	zeroTest      = "0123490"
	goodIndexFull = "[1234]"
	halfIndex     = "[123"
	noIndex       = "123"
	badIndex      = "[b12]"
	name          = "name"
	panicError    = "unsuported kind for index"
)

type testStruct struct {
	Name string `yaml:"name"`
}

type badStruct struct {
	Name string `bad:"name"`
}

type jsonStruct struct {
	Name string `json:"test"`
}

var mapTest = map[string]string{
	name: "test",
}

var sliceTest = []string{"test"}

var jsonTest = jsonStruct{Name: "Test"}

var badStructTest = badStruct{
	Name: "name",
}

var emptyMap = map[string]string{}

var mapTest2 = map[string]string{
	"name":    "test",
	"attempt": "second",
}

func typeIsAggregable(v interface{}) bool {
	vType := reflect.ValueOf(v)

	return isAggregable(vType)
}

func typeIsMergeable(v interface{}) bool {
	vType := reflect.ValueOf(v)

	return isMergeable(vType)
}

func Test_Has_Index_Returns_True(t *testing.T) {
	assert.True(t, hasIndex(goodHasIndex))
}

func Test_Has_Index_Returns_False(t *testing.T) {
	assert.False(t, hasIndex(badHasIndex))
}

func Test_IsAggregable_Returns_True(t *testing.T) {
	assert.True(t, typeIsAggregable(mapTest))
	assert.True(t, typeIsAggregable(sliceTest))
}

func Test_IsAggregable_Returns_False(t *testing.T) {
	assert.False(t, typeIsAggregable(stringTest))
}

func Test_IsMergeable_Returns_True(t *testing.T) {
	assert.True(t, typeIsMergeable(sliceTest))
	assert.True(t, typeIsMergeable(mapTest))
}

func Test_IsMergeable_Returns_False(t *testing.T) {
	assert.False(t, typeIsMergeable(stringTest))
}

func Test_ParseIndex_Returns_No_Error(t *testing.T) {
	strValue, intVal, err := parseIndex(goodIndexFull)

	assert.NoError(t, err)
	assert.NotNil(t, intVal)
	assert.NotNil(t, strValue)
}

func Test_ParseIndex_Returns_Error(t *testing.T) {
	strValue, intVal, err := parseIndex(halfIndex)

	assert.Error(t, err)
	assert.Equal(t, intVal, -1)
	assert.Empty(t, strValue)
}

func Test_ParseIndex_Returns_Empty(t *testing.T) {
	strValue, intVal, err := parseIndex(noIndex)

	assert.NoError(t, err)
	assert.Nil(t, err)
	assert.Equal(t, intVal, -1)
	assert.Equal(t, strValue, noIndex)
}

func Test_ParseIndex_FailsToConverts(t *testing.T) {
	strValue, intVal, err := parseIndex(badIndex)

	assert.Error(t, err)
	assert.Equal(t, intVal, -1)
	assert.Empty(t, strValue)
}

func Test_IndexFunction_Good_WithSlice(t *testing.T) {
	vType := reflect.ValueOf(sliceTest)

	resp := indexFunction(vType)
	assert.Equal(t, resp(0).Interface(), sliceTest[0])
}

func Test_IndexFunction_Good_WithMap(t *testing.T) {
	vType := reflect.ValueOf(mapTest)

	resp := indexFunction(vType)
	assert.Equal(t, resp(0).Interface(), mapTest[name])
}

func Test_IndexFunction_Panics(t *testing.T) {
	vType := reflect.ValueOf(stringTest)

	defer func() {
		err := recover()
		assert.Equal(t, err, panicError)
	}()
	indexFunction(vType)
}

func Test_RemoveZeroValues(t *testing.T) {
	vType := reflect.ValueOf(zeroTest)

	resp := removeZeroValues([]reflect.Value{vType})
	assert.Equal(t, resp[0].Interface(), zeroTest)
}

func Test_FieldByTagName_Returns_Success(t *testing.T) {
	testType := testStruct{
		Name: "name",
	}
	vType := reflect.TypeOf(testType)
	resp, boolResult := FieldByTagName(vType, "name")

	assert.True(t, boolResult)
	assert.Equal(t, resp.Name, "Name")
}

func Test_FieldByTagName_Fails(t *testing.T) {
	vType := reflect.TypeOf(badStructTest)
	resp, boolResult := FieldByTagName(vType, "name")

	assert.False(t, boolResult)
	assert.Nil(t, resp)
}

func Test_FieldByName_GoodJson_Succeeds(t *testing.T) {
	vType := reflect.TypeOf(jsonTest)

	resp, boolResult := FieldByTagName(vType, "test")

	assert.True(t, boolResult)
	assert.Equal(t, resp.Name, "Name")
}

func Test_ValueByTagName_Returns_Success(t *testing.T) {
	vType := reflect.ValueOf(jsonTest)

	resp, boolResult := ValueByTagName(vType, "test")

	assert.True(t, boolResult)
	assert.Equal(t, resp.Interface(), "Test")
}

func Test_ValueByTagName_Fails(t *testing.T) {
	vType := reflect.ValueOf(badStructTest)

	resp, boolResult := ValueByTagName(vType, "test")

	assert.False(t, boolResult)
	assert.Empty(t, resp)
}

func Test_LookUpType_Returns_Success(t *testing.T) {
	vType := reflect.TypeOf(sliceTest)

	resp, boolResult := lookupType(vType, "[path")
	assert.True(t, boolResult)
	assert.IsType(t, reflect.String, resp.Kind())
}

func Test_LookUpType_Returns_Success_When_Length_Zero(t *testing.T) {
	vType := reflect.TypeOf(sliceTest)

	resp, boolResult := lookupType(vType)
	assert.True(t, boolResult)
	assert.Equal(t, resp, vType)
}

func Test_LookUpType_WithStruct(t *testing.T) {
	vType := reflect.TypeOf(jsonTest)

	resp, boolResult := lookupType(vType, "Name")
	assert.True(t, boolResult)
	assert.IsType(t, reflect.String, resp.Kind())
}

func Test_LookUpType_WithFieldByTagName(t *testing.T) {
	vType := reflect.TypeOf(jsonTest)

	resp, boolResult := lookupType(vType, "test")
	assert.True(t, boolResult)
	assert.IsType(t, reflect.String, resp.Kind())

}

func Test_LookUpType_WithFieldByTagName_WithIncorrectTag(t *testing.T) {
	vType := reflect.TypeOf(jsonTest)

	resp, boolResult := lookupType(vType, "none")
	assert.False(t, boolResult)
	assert.Nil(t, resp)
}

func Test_GetValueByName(t *testing.T) {
	vType := reflect.ValueOf(jsonTest)

	resp, err := getValueByName(vType, "Name")
	assert.NoError(t, err)
	assert.Equal(t, resp.Interface(), "Test")
}

func Test_GetValueByName_Returns_Error(t *testing.T) {
	vType := reflect.ValueOf(jsonTest)

	resp, err := getValueByName(vType, badIndex)

	assert.Error(t, err)
	assert.Empty(t, resp)
}

func Test_GetValueByName_Without_Tag(t *testing.T) {
	vType := reflect.ValueOf(jsonTest)

	resp, err := getValueByName(vType, "Name")
	assert.NoError(t, err)
	assert.Equal(t, resp.Interface(), jsonTest.Name)
}

func Test_GetValueByName_WithMap(t *testing.T) {
	vType := reflect.ValueOf(mapTest)

	resp, err := getValueByName(vType, "name")
	assert.NoError(t, err)
	assert.Equal(t, resp.Interface(), "test")
}

func Test_GetValueByName_WithMapError(t *testing.T) {
	vType := reflect.ValueOf(mapTest)

	resp, err := getValueByName(vType, "none")
	assert.Error(t, err)
	assert.Empty(t, resp)
}

func Test_Lookup(t *testing.T) {
	resp, err := Lookup(mapTest, "name")

	assert.NoError(t, err)
	assert.Equal(t, resp.Interface(), "test")
}

func Test_Lookup_Error(t *testing.T) {
	resp, err := Lookup(mapTest, "none")

	assert.Error(t, err)
	assert.Empty(t, resp)
}

func Test_MergeValue_Succeeds(t *testing.T) {
	vType := reflect.ValueOf(stringTest)
	resp := mergeValue([]reflect.Value{vType})

	assert.Equal(t, len(resp.Interface().([]string)), 1)
	assert.Equal(t, resp.Interface().([]string)[0], "test")
}

func Test_MergeValue_Returns_Empty(t *testing.T) {
	resp := mergeValue([]reflect.Value{})

	assert.Empty(t, resp)
}

func Test_MergeValue_Returns_PanicsWithMap(t *testing.T) {
	vType := reflect.ValueOf(mapTest)

	defer func() {
		err := recover()
		assert.Error(t, err.(error))
	}()
	mergeValue([]reflect.Value{vType})
}

func Test_AggreateAggregableValue_Panics_WithStruct(t *testing.T) {
	vType := reflect.ValueOf(jsonTest)

	defer func() {
		err := recover()
		assert.Error(t, err.(error))
	}()
	_, _ = aggreateAggregableValue(vType, []string{"Name"})

}

func Test_AggreateAggregableValue_Returns_EmptyWithEmptyMap(t *testing.T) {
	vType := reflect.ValueOf(emptyMap)

	resp, err := aggreateAggregableValue(vType, []string{""})

	assert.Error(t, err)
	assert.Empty(t, resp)
}

func Test_AggreateAggreagableValue_Returns_EmptySlice(t *testing.T) {
	vType := reflect.ValueOf(emptyMap)

	resp, err := aggreateAggregableValue(vType, []string{"[test"})
	assert.Nil(t, err)
	assert.NoError(t, err)
	assert.Equal(t, len(resp.Interface().([]string)), 0)
}

func Test_LookupString(t *testing.T) {
	resp, err := LookupString(mapTest, "name")

	assert.NoError(t, err)
	assert.Nil(t, err)
	assert.Equal(t, resp.Interface(), "test")
}

func Test_LookupString_Returns_Error(t *testing.T) {

	resp, err := LookupString(mapTest, "none")

	assert.Error(t, err)
	assert.NotNil(t, err)
	assert.Empty(t, resp)
}
