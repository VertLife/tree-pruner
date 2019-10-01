package main

import "fmt"

// PruneError is a map of error struct for the pruner
type PruneError struct {
	Detail string
	Code   int32
	JobID  string
	Err    error
}

func (e PruneError) Error() string {
	return fmt.Sprintf("Prune Error %v for job: %v: %v\n\t%v", e.Code, e.JobID, e.Detail, e.Err)
}

// NewError is a constructor
func NewError(code int32, jobid, detail string, err error) PruneError {
	return PruneError{
		Code:   code,
		JobID:  jobid,
		Detail: detail,
		Err:    err,
	}
}
