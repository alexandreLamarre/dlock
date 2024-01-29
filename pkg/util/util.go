package util

import (
	"context"
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

// WaitAll waits for all the given channels to be closed, under the
// following rules:
// 1. The lifetime of the task represented by each channel is directly tied to
// the provided context.
// 2. If a task exits with an error before the context is canceled, the
// context should be canceled.
// 3. If a task exits successfully, the context should not be canceled and
// other tasks should continue to run.
func WaitAll(ctx context.Context, ca context.CancelCauseFunc, channels ...<-chan error) error {
	cases := []reflect.SelectCase{
		{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(ctx.Done()),
		},
	}
	for _, ch := range channels {
		cases = append(cases, reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(ch),
		})
	}
	i, value, _ := reflect.Select(cases)
	if i == 0 {
		ca(ctx.Err())
		for _, c := range channels {
			<-c
		}
		return ctx.Err()
	}
	channelIdx := i - 1
	var err error
	if i := value.Interface(); i != nil {
		err = i.(error)
	}
	if err == nil {
		// run again, but skip the channel which exited successfully
		return WaitAll(ctx, ca, append(channels[:channelIdx], channels[channelIdx+1:]...)...)
	}
	ca(err)
	for i, c := range channels {
		if i == channelIdx {
			continue
		}
		<-c
	}
	return err
}
