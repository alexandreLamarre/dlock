package util

import (
	"errors"
	"reflect"
)

func Must[T any](t T, err ...error) T {
	if len(err) > 0 {
		if err[0] != nil {
			panic(errors.Join(err...))
		}
	} else if tv := reflect.ValueOf(t); (tv != reflect.Value{}) {
		if verr := tv.Interface().(error); verr != nil {
			panic(verr)
		}
	}
	return t
}
