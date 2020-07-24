//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package timer

import "time"

// Handle is a structure gathering all the data necessary to implement timers
type Handle struct {
	t_start time.Time
}

// Start creates and start a timer
func Start() *Handle {
	h := new(Handle)
	h.t_start = time.Now()
	return h
}

// Stop ends a timer and returns the time in seconds in the form of a string
func (h *Handle) Stop() string {
	return time.Since(h.t_start).String()
}
