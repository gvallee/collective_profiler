//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package errors

type InternalError struct {
	msg  string // message associated to the error
	code int    // error code
}

type ProfilerError struct {
	internal InternalError
	details  error
}

// ErrNone means success
var ErrNone = InternalError{"Success", 0}

// ErrNotFound means that the object/eentity requested could not be found
var ErrNotFound = InternalError{"Not found", -1}

// ErrInvalidHeader means we could not get the header
var ErrInvalidHeader = InternalError{"Invalid header", -2}

// ErrFatal means that a fatal error occured
var ErrFatal = InternalError{"Fatal error", -3}

func New(i InternalError, err error) *ProfilerError {
	e := new(ProfilerError)
	e.details = err
	e.internal = i
	return e
}

func (e *ProfilerError) Is(i InternalError) bool {
	if e.internal == i {
		return true
	}
	return false
}

func (e *ProfilerError) GetInternal() error {
	return e.details
}
