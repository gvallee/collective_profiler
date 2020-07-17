//
// Copyright (c) 2020, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package format

const (

	// ProfileSummaryFilePrefix is the prefix used for all generated profile summary files
	ProfileSummaryFilePrefix = "profile_alltoallv_rank"

	// MulticommHighlightFilePrefix is the prefix of the file used to store the highlights when data has multi-communicators patterns
	MulticommHighlightFilePrefix = "multicomm-highlights"

	// DefaultMsgSizeThreshold is the default threshold to differentiate message and large messages.
	DefaultMsgSizeThreshold = 200
)
