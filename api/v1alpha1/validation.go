package v1alpha1

import "errors"

func (in *LockRequest) Validate() error {
	if in.Key == "" {
		return errors.New("key is required")
	}
	return nil
}
