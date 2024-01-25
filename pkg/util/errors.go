package util

import (
	"errors"
	"sort"
	"sync"
)

// MultiErrGroup
// Best effort to run all tasks, then combines all errors
//
// Example:
//
// var eg util.MutliErrGroup{}

// tasks := []func() error{/* */}
//
//	for _, task := range tasks {
//		eg.Go(func() error {
//		return task()
//		}
//	}
//
// eg.Wait()
//
//	if err := eg.Error(); err != nil {
//		// handle error
//	}
type MultiErrGroup struct {
	errMu sync.Mutex
	errs  []error
	sync.WaitGroup
}

func (i *MultiErrGroup) Add(tasks int) {
	i.WaitGroup.Add(tasks)
}

func (i *MultiErrGroup) Done() {
	i.WaitGroup.Done()
}

func (i *MultiErrGroup) Wait() {
	i.WaitGroup.Wait()
}

func (i *MultiErrGroup) addError(err error) {
	i.errMu.Lock()
	defer i.errMu.Unlock()
	i.errs = append(i.errs, err)
}

func (i *MultiErrGroup) Errors() []error {
	i.errMu.Lock()
	defer i.errMu.Unlock()
	return i.errs
}

func (i *MultiErrGroup) Error() error {
	if len(i.errs) == 0 {
		return nil
	}
	duped := map[string]struct{}{}
	resErr := []error{}
	for _, err := range i.errs {
		if _, ok := duped[err.Error()]; !ok {
			duped[err.Error()] = struct{}{}
			resErr = append(resErr, err)
		}
	}
	sort.Slice(resErr, func(i, j int) bool {
		return resErr[i].Error() < resErr[j].Error()
	})
	return errors.Join(resErr...)
}

func (i *MultiErrGroup) Go(fn func() error) {
	i.Add(1)
	go func() {
		defer i.Done()
		if err := fn(); err != nil {
			i.addError(err)
		}
	}()
}
