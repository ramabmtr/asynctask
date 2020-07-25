package asynctask

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResultStringSucceed(t *testing.T) {
	expected := "test"
	actual, err := ResultString(expected)

	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestResultStringError(t *testing.T) {
	actual, err := ResultString(nil)

	assert.Error(t, err)
	assert.Equal(t, "", actual)
}

func TestResultIntSucceed(t *testing.T) {
	expected := 1
	actual, err := ResultInt(expected)

	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestResultIntError(t *testing.T) {
	actual, err := ResultInt(nil)

	assert.Error(t, err)
	assert.Equal(t, 0, actual)
}

func TestResultBoolSucceed(t *testing.T) {
	expected := true
	actual, err := ResultBool(expected)

	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestResultBoolError(t *testing.T) {
	actual, err := ResultBool(nil)

	assert.Error(t, err)
	assert.Equal(t, false, actual)
}

func TestResultObjSucceed(t *testing.T) {
	type T struct {
		Key string `json:"key"`
		Val string `json:"val"`
	}

	key := "keyTest"
	val := "valTest"

	result := map[string]interface{}{
		"key": key,
		"val": val,
	}

	var actual T

	err := ResultObj(result, &actual)

	assert.NoError(t, err)
	assert.Equal(t, key, actual.Key)
	assert.Equal(t, val, actual.Val)
}

func TestResultObjMarshalError(t *testing.T) {
	err := ResultObj(make(chan int), "")

	assert.Error(t, err)
}

func TestResultObjUnmarshalError(t *testing.T) {
	result := map[string]interface{}{
		"key": "key",
		"val": "val",
	}

	err := ResultObj(result, "")

	assert.Error(t, err)
}
