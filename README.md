# asynctask

[![Go Report Card](https://goreportcard.com/badge/github.com/ramabmtr/asynctask)](https://goreportcard.com/report/github.com/ramabmtr/asynctask)
[![Build Status](https://travis-ci.org/ramabmtr/asynctask.svg?branch=master)](https://travis-ci.org/ramabmtr/asynctask)
[![codecov](https://codecov.io/gh/ramabmtr/asynctask/branch/master/graph/badge.svg)](https://codecov.io/gh/ramabmtr/asynctask)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/ramabmtr/asynctask)](https://pkg.go.dev/github.com/ramabmtr/asynctask)
[![Release](https://img.shields.io/github/v/release/ramabmtr/asynctask?include_prereleases&sort=semver)](https://github.com/ramabmtr/asynctask/releases)

Golang handy goroutine runner.

This package aim to simplify goroutine management.

You can run goroutine and get result from it without make [channel](https://golang.org/doc/effective_go.html#channels)
or use [waitgroup](https://golang.org/pkg/sync/#WaitGroup). Just define the function and let this package
handle the rest for you.

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
	asyncTask := asynctask.NewAsyncTask(context.Background())

	// Run first task
	asyncTask.NewRunner().SetFunc(func(p interface{}) (interface{}, error) {
		time.Sleep(3 * time.Second)
		return "test1", nil
	}).Register("taskID1")

	// Run second task
	asyncTask.NewRunner().SetFunc(func(p interface{}) (interface{}, error) {
		time.Sleep(5 * time.Second)
		return true, nil
	}).Register("taskID2")

	// Wait all task to complete
	// if one of runner return error, asynctask will raise that error immediately
	err := asyncTask.StartAndWait()
	if err != nil {
		fmt.Println(err)
		return
	}

	result1, err := asynctask.ResultString(asyncTask.GetResult("taskID1"))
	if err != nil {
		fmt.Println(err)
		return
	}

	result2, err := asynctask.ResultBool(asyncTask.GetResult("taskID2"))
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(result1) // test
	fmt.Println(result2) // true
}
```

`asynctask` will raise error immediately if one of `runner` return error.

you can override it with `asyncTask.CancelOnError(false)`.

```go
asyncTask := asynctask.NewAsyncTask(context.Background())

// Run first task
asyncTask.NewRunner().SetFunc(func(p interface{}) (interface{}, error) {
	return nil, fmt.Errorf("test error")
}).Register("taskID1")

// Run second task
asyncTask.NewRunner().SetFunc(func(p interface{}) (interface{}, error) {
	time.Sleep(time.Second)
	return true, nil
}).Register("taskID2")

// this code will return error as soon as the first task return an error
// it wont wait for second task to complete by default
// uncomment code below to override this behaviour
// asyncTask.CancelOnError(false)
err := asyncTask.Wait() // err == test error
```

If you want to run multiple `runner` with same ID, you can set with `SetMultiple()`.

This suit for creating `runner` inside the loop. The result will be slice of interface.

```go
asyncTask := asynctask.NewAsyncTaskRunner(context.Background())
for i := 0; i < 10; i++ {
	asyncTask.NewRunner().SetFunc(func(p interface{}) (interface{}, error) {
		return "test", nil
	}).SetMultiple().Register("taskID")
}

// Wait all task to complete
err := asyncTask.Wait()
if err != nil {
    fmt.Println(err)
    return
}

result := asyncTask.GetResult("taskID")
fmt.Println(result) // [test test ... test]
```

You can also pass param to `runner` with `SetParam(param interface{})`

```go
asyncTask.NewRunner().SetFunc(func(p interface{}) (interface{}, error) {
    param, ok := p.(string)
    if !ok {
        return nil, fmt.Errorf("param is not string")
    }
    return param, nil
}).SetParam("param").Register("taskID")
```
