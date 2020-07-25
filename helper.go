package asynctask

import (
	"errors"

	jsoniter "github.com/json-iterator/go"
)

func ResultString(i interface{}) (string, error) {
	x, ok := i.(string)
	if !ok {
		return "", errors.New("interface is not string")
	}

	return x, nil
}

func ResultInt(i interface{}) (int, error) {
	x, ok := i.(int)
	if !ok {
		return 0, errors.New("interface is not int")
	}

	return x, nil
}

func ResultBool(i interface{}) (bool, error) {
	x, ok := i.(bool)
	if !ok {
		return false, errors.New("interface is not bool")
	}

	return x, nil
}

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
