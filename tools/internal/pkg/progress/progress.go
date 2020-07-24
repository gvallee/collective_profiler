//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package progress

import (
	"fmt"
)

type Bar struct {
	label   string
	enabled bool
	current int
	max     int
}

func (b *Bar) display() {
	label := b.label
	if label == "" {
		label = "Progress"
	}
	fmt.Printf("\r%s: %d/%d", b.label, b.current, b.max)
}

func NewBar(max int, label string) *Bar {
	b := new(Bar)
	b.max = max
	b.current = 0
	b.enabled = true
	b.label = label
	//fmt.Printf("\n\n")
	b.display()
	return b
}

func (b *Bar) Increment(val int) {
	b.current += val
	b.display()
}

func EndBar(b *Bar) {
	b.enabled = false
	fmt.Printf("\r%s: %d/%d", b.label, b.current, b.max)
	fmt.Printf("\n")
}
