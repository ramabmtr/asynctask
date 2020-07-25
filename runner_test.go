package asynctask

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAsyncTaskSucceed(t *testing.T) {
	testID1 := "testID1"
	testID2 := "testID2"

	expectedResult1 := "result1"
	runner := NewAsyncTaskRunner(context.Background())

	runner.SetFunc(func(param interface{}) (interface{}, error) {
		return expectedResult1, nil
	}).Do(testID1)

	runner.SetFunc(func(param interface{}) (interface{}, error) {
		return nil, nil
	}).Do(testID2)

	err := runner.Wait()

	assert.NoError(t, err)

	actualResult1 := runner.GetResult(testID1)

	assert.Equal(t, expectedResult1, actualResult1)

	actualResult2 := runner.GetResult(testID2)

	assert.Nil(t, actualResult2)
}

func TestAsyncTaskWithParamSucceed(t *testing.T) {
	testID1 := "testID1"
	testID2 := "testID2"

	expectedResult1 := "result1"
	expectedResult2 := "result2"
	runner := NewAsyncTaskRunner(context.Background())

	runner.SetFunc(func(param interface{}) (interface{}, error) {
		return expectedResult1, nil
	}).SetParam(expectedResult1).Do(testID1)

	runner.SetFunc(func(param interface{}) (interface{}, error) {
		return expectedResult2, nil
	}).SetParam(expectedResult2).Do(testID2)

	err := runner.Wait()

	assert.NoError(t, err)

	actualResult1 := runner.GetResult(testID1)

	assert.Equal(t, expectedResult1, actualResult1)

	actualResult2 := runner.GetResult(testID2)

	assert.Equal(t, expectedResult2, actualResult2)
}

func TestAsyncTaskMultipleSucceed(t *testing.T) {
	testID := "testID"
	runner := NewAsyncTaskRunner(context.Background())

	expectedResult := make([]interface{}, 0)
	testResultPrefix := "testResult"
	iteration := 3

	for i := 0; i < iteration; i++ {
		param := fmt.Sprintf("%s%v", testResultPrefix, i)
		expectedResult = append(expectedResult, param)
		runner.SetFunc(func(param interface{}) (interface{}, error) {
			return param, nil
		}).SetParam(param).SetMultiple().Do(testID)
	}

	err := runner.Wait()

	assert.NoError(t, err)

	actualResult := runner.GetResult(testID)

	assert.Len(t, actualResult, iteration)
	for i := 0; i < iteration; i++ {
		assert.Contains(t, actualResult, expectedResult[i])
	}
}

func TestAsyncTaskErrorDoubleID(t *testing.T) {
	testID := "testID"

	runner := NewAsyncTaskRunner(context.Background())

	runner.SetFunc(func(param interface{}) (interface{}, error) {
		return "test", nil
	}).Do(testID)

	runner.SetFunc(func(param interface{}) (interface{}, error) {
		return "test", nil
	}).Do(testID)

	err := runner.Wait()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), testID)
}

func TestAsyncTaskErrorProcessResultMultiple(t *testing.T) {
	testID := "testID"

	runner := NewAsyncTaskRunner(context.Background())

	runner.SetFunc(func(param interface{}) (interface{}, error) {
		return "test", nil
	}).Do(testID)

	runner.SetFunc(func(param interface{}) (interface{}, error) {
		time.Sleep(100 * time.Millisecond)
		return "test", nil
	}).SetMultiple().Do(testID)

	err := runner.Wait()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot append result")
}

func TestAsyncTaskPanic(t *testing.T) {
	runner := NewAsyncTaskRunner(context.Background())

	runner.SetFunc(func(param interface{}) (interface{}, error) {
		time.Sleep(100 * time.Millisecond)
		panic("test")
	}).Do("id1")

	runner.SetFunc(func(param interface{}) (interface{}, error) {
		// add delay to simulate context cancel
		time.Sleep(200 * time.Millisecond)
		panic("test")
	}).Do("id2")

	runner.SetFunc(func(param interface{}) (interface{}, error) {
		// add delay to simulate context cancel
		time.Sleep(250 * time.Millisecond)
		panic("test")
	}).Do("id3")

	err := runner.Wait()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "panic")

	// add delay to simulate context cancel
	time.Sleep(300 * time.Millisecond)
}

func TestAsyncTaskErrorAndCancelContext(t *testing.T) {
	runner := NewAsyncTaskRunner(context.Background())

	testErr := fmt.Errorf("test error")

	runner.SetFunc(func(param interface{}) (interface{}, error) {
		return nil, testErr
	}).Do("id1")

	runner.SetFunc(func(param interface{}) (interface{}, error) {
		// add delay to simulate context cancel
		time.Sleep(100 * time.Millisecond)
		return param, nil
	}).SetParam("result").Do("id2")

	runner.SetFunc(func(param interface{}) (interface{}, error) {
		// add delay to simulate context cancel
		time.Sleep(150 * time.Millisecond)
		return param, nil
	}).SetParam("result").Do("id3")

	err := runner.Wait()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), testErr.Error())

	// add delay to simulate context cancel
	time.Sleep(200 * time.Millisecond)
}
