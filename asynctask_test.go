package asynctask

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSafeWriteChan(t *testing.T) {
	safeCh := newSafeResultChan()

	go func() {
		safeCh.write(result{"res", nil})
	}()

	select {
	case res := <-safeCh.read():
		assert.NoError(t, res.err)
		assert.Equal(t, "res", res.resp)
	}

	safeCh.close()

	// write after close should be not panic
	go func() {
		safeCh.write(result{"res", nil})
	}()

	select {
	case res := <-safeCh.read():
		assert.NoError(t, res.err)
		assert.Nil(t, res.resp)
	}
}

func TestAsyncTaskSucceed(t *testing.T) {
	testID1 := "testID1"
	testID2 := "testID2"

	expectedResult1 := "result1"

	asyncTask := NewAsyncTask(context.Background()).SetRunnerPoolSize(10)

	asyncTask.NewRunner().SetFunc(func(param interface{}) (interface{}, error) {
		return expectedResult1, nil
	}).Register(testID1)

	asyncTask.NewRunner().SetFunc(func(param interface{}) (interface{}, error) {
		return nil, nil
	}).Register(testID2)

	err := asyncTask.StartAndWait()

	assert.NoError(t, err)

	actualResult1 := asyncTask.GetResult(testID1)

	assert.Equal(t, expectedResult1, actualResult1)

	actualResult2 := asyncTask.GetResult(testID2)

	assert.Nil(t, actualResult2)
}

func TestAsyncTaskNoCancelWhenError(t *testing.T) {
	testID1 := "testID1"
	testID2 := "testID2"

	expectedResult2 := "result2"

	asyncTask := NewAsyncTask(context.Background()).SetRunnerPoolSize(10)
	asyncTask.CancelOnError(false)

	asyncTask.NewRunner().SetFunc(func(param interface{}) (interface{}, error) {
		return nil, fmt.Errorf("test")
	}).Register(testID1)

	asyncTask.NewRunner().SetFunc(func(param interface{}) (interface{}, error) {
		time.Sleep(100 * time.Millisecond)
		return expectedResult2, nil
	}).Register(testID2)

	err := asyncTask.StartAndWait()

	assert.Error(t, err)

	actualResult2 := asyncTask.GetResult(testID2)

	assert.Equal(t, expectedResult2, actualResult2)
}

func TestAsyncTaskWithParamSucceed(t *testing.T) {
	testID1 := "testID1"
	testID2 := "testID2"

	expectedResult1 := "result1"
	expectedResult2 := "result2"

	asyncTask := NewAsyncTask(context.Background())

	asyncTask.NewRunner().SetFunc(func(param interface{}) (interface{}, error) {
		return expectedResult1, nil
	}).SetParam(expectedResult1).Register(testID1)

	asyncTask.NewRunner().SetFunc(func(param interface{}) (interface{}, error) {
		return expectedResult2, nil
	}).SetParam(expectedResult2).Register(testID2)

	err := asyncTask.StartAndWait()

	assert.NoError(t, err)

	actualResult1 := asyncTask.GetResult(testID1)

	assert.Equal(t, expectedResult1, actualResult1)

	actualResult2 := asyncTask.GetResult(testID2)

	assert.Equal(t, expectedResult2, actualResult2)
}

func TestAsyncTaskMultipleSucceed(t *testing.T) {
	testID := "testID"
	asyncTask := NewAsyncTask(context.Background())

	expectedResult := make([]interface{}, 0)
	testResultPrefix := "testResult"
	iteration := 3

	for i := 0; i < iteration; i++ {
		param := fmt.Sprintf("%s%v", testResultPrefix, i)
		expectedResult = append(expectedResult, param)
		asyncTask.NewRunner().SetFunc(func(param interface{}) (interface{}, error) {
			return param, nil
		}).SetParam(param).SetMultiple().Register(testID)
	}

	err := asyncTask.StartAndWait()

	assert.NoError(t, err)

	actualResult := asyncTask.GetResult(testID)

	assert.Len(t, actualResult, iteration)
	for i := 0; i < iteration; i++ {
		assert.Contains(t, actualResult, expectedResult[i])
	}
}

func TestAsyncTaskErrorDoubleID(t *testing.T) {
	testID := "testID"

	asyncTask := NewAsyncTask(context.Background())

	asyncTask.NewRunner().SetFunc(func(param interface{}) (interface{}, error) {
		return "test", nil
	}).Register(testID)

	asyncTask.NewRunner().SetFunc(func(param interface{}) (interface{}, error) {
		return "test", nil
	}).Register(testID)

	err := asyncTask.StartAndWait()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), testID)
}

func TestAsyncTaskErrorProcessResultMultipleError(t *testing.T) {
	testID := "testID"

	asyncTask := NewAsyncTask(context.Background())

	asyncTask.NewRunner().SetFunc(func(param interface{}) (interface{}, error) {
		return "test", nil
	}).Register(testID)

	asyncTask.NewRunner().SetFunc(func(param interface{}) (interface{}, error) {
		time.Sleep(100 * time.Millisecond)
		return "test", nil
	}).SetMultiple().Register(testID)

	err := asyncTask.StartAndWait()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot append result")
}

func TestAsyncTaskError(t *testing.T) {
	asyncTask := NewAsyncTask(context.Background())
	// set pool to 1 to simulate context cancel
	asyncTask.SetRunnerPoolSize(1)

	testErr := fmt.Errorf("test error")

	asyncTask.NewRunner().SetFunc(func(param interface{}) (interface{}, error) {
		return nil, testErr
	}).Register("id1")

	asyncTask.NewRunner().SetFunc(func(param interface{}) (interface{}, error) {
		return param, nil
	}).SetParam("result").Register("id2")

	asyncTask.NewRunner().SetFunc(func(param interface{}) (interface{}, error) {
		return param, nil
	}).SetParam("result").Register("id3")

	err := asyncTask.StartAndWait()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), testErr.Error())
}

func TestAsyncTaskPanic(t *testing.T) {
	asyncTask := NewAsyncTask(context.Background())

	asyncTask.NewRunner().SetFunc(func(param interface{}) (interface{}, error) {
		time.Sleep(100 * time.Millisecond)
		panic("test")
	}).Register("id1")

	err := asyncTask.StartAndWait()

	assert.Error(t, err)
	if err != nil {
		assert.Contains(t, err.Error(), "panic")
	}
}

func TestAsyncTaskErrorAndCancelContext(t *testing.T) {
	asyncTask := NewAsyncTask(context.Background())

	testErr := fmt.Errorf("test error")

	asyncTask.NewRunner().SetFunc(func(param interface{}) (interface{}, error) {
		return "result", testErr
	}).Register("id1")

	asyncTask.NewRunner().SetFunc(func(param interface{}) (interface{}, error) {
		time.Sleep(100 * time.Millisecond)
		return param, testErr
	}).SetParam("result").Register("id2")

	err := asyncTask.StartAndWait()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), testErr.Error())

	// add delay to simulate context cancel
	time.Sleep(200 * time.Millisecond)
}

func TestAsyncTaskWithLimitedPoolErrorAndCancelContext(t *testing.T) {
	asyncTask := NewAsyncTask(context.Background())
	asyncTask.SetRunnerPoolSize(1)

	testErr := fmt.Errorf("test error")

	asyncTask.NewRunner().SetFunc(func(param interface{}) (interface{}, error) {
		return nil, testErr
	}).Register("id1")

	asyncTask.NewRunner().SetFunc(func(param interface{}) (interface{}, error) {
		return param, nil
	}).SetParam("result").Register("id2")

	asyncTask.NewRunner().SetFunc(func(param interface{}) (interface{}, error) {
		return param, nil
	}).SetParam("result").Register("id3")

	err := asyncTask.StartAndWait()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), testErr.Error())
}
