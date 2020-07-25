# asynctask

[![Go Report Card](https://goreportcard.com/badge/github.com/ramabmtr/asynctask)](https://goreportcard.com/report/github.com/ramabmtr/asynctask)
[![Build Status](https://travis-ci.org/ramabmtr/asynctask.svg?branch=master)](https://travis-ci.org/ramabmtr/asynctask)
[![codecov](https://codecov.io/gh/ramabmtr/asynctask/branch/master/graph/badge.svg)](https://codecov.io/gh/ramabmtr/asynctask)

Golang handy goroutine runner

# Installation

Use go get

```bash
go get github.com/ramabmtr/asynctask
```

Then import it in your code

```go
import "github.com/ramabmtr/asynctask"
```

# Usage

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ramabmtr/asynctask"
)

func main() {
	runner := asynctask.NewAsyncTaskRunner(context.Background())

	// Run first task
	runner.SetFunc(func(p interface{}) (interface{}, error) {
		time.Sleep(3 * time.Second)
		return "test1", nil
	}).Do("taskID1")

	// Run second task
	runner.SetFunc(func(p interface{}) (interface{}, error) {
		time.Sleep(5 * time.Second)
		return true, nil
	}).Do("taskID2")

	// Wait all task to complete
	err := runner.Wait()
	if err != nil {
		fmt.Println(err)
		return
	}

	result1, err := asynctask.ResultString(runner.GetResult("taskID1"))
	if err != nil {
		fmt.Println(err)
		return
	}

	result2, err := asynctask.ResultBool(runner.GetResult("taskID2"))
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(result1) // test
	fmt.Println(result2) // true
}
```

`asynctask` will raise error immediately if one of `asynctask` return error

```go
// Run first task
runner.SetFunc(func(p interface{}) (interface{}, error) {
	return nil, fmt.Errorf("test error")
}).Do("taskID1")

// Run second task
runner.SetFunc(func(p interface{}) (interface{}, error) {
	time.Sleep(time.Second)
	return true, nil
}).Do("taskID2")

// this code will return error as soon as the first task return an error
// it wont wait for second task to complete
err := runner.Wait() // err == test error
```

If you want to run multiple `asynctask` with same ID, you can set with `SetMultiple()`.

This suit for calling `asynctask` inside the loop. The result will be slice of interface.

```go
runner := asynctask.NewAsyncTaskRunner(context.Background())
for i := 0; i < 10; i++ {
	runner.SetFunc(func(p interface{}) (interface{}, error) {
		return "test", nil
	}).SetMultiple().Do("taskID")
}

// Wait all task to complete
err := runner.Wait()
if err != nil {
    fmt.Println(err)
    return
}

result := runner.GetResult("taskID")
fmt.Println(result) // [test test ... test]
```

You can also pass param to `asynctask` runner with `SetParam(param interface{})`

```go
runner.SetFunc(func(p interface{}) (interface{}, error) {
    param, ok := p.(string)
    if !ok {
        return nil, fmt.Errorf("param is not string")
    }
    return param, nil
}).SetParam("param").Do("taskID")
```
