package asynctask

import (
	"errors"

	jsoniter "github.com/json-iterator/go"
)

// ResultString parse asynctask runner result to string
// return error if actual result is not string
func ResultString(i interface{}) (string, error) {
	x, ok := i.(string)
	if !ok {
		return "", errors.New("interface is not string")
	}

	return x, nil
}

// ResultInt parse asynctask runner result to int
// return error if actual result is not int
func ResultInt(i interface{}) (int, error) {
	x, ok := i.(int)
	if !ok {
		return 0, errors.New("interface is not int")
	}

	return x, nil
}

// ResultBool parse asynctask runner result to bool
// return error if actual result is not bool
func ResultBool(i interface{}) (bool, error) {
	x, ok := i.(bool)
	if !ok {
		return false, errors.New("interface is not bool")
	}

	return x, nil
}

// ResultObj parse asynctask runner result to destination interface
// return error if actual result schema and destination schema is not match
func ResultObj(i interface{}, o interface{}) error {
	b, err := jsoniter.Marshal(i)
	if err != nil {
		return err
	}

	err = jsoniter.Unmarshal(b, o)
	if err != nil {
		return err
	}

	return nil
}
