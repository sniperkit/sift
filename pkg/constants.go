package sift

import (
	"time"
)

const (
	// InputMultilineWindow is the size of the sliding window for multiline matching
	InputMultilineWindow = 32 * 1024
	// MultilinePipeTimeout is the timeout for reading and matching input
	// from STDIN/network in multiline mode
	MultilinePipeTimeout = 1000 * time.Millisecond
	// MultilinePipeChunkTimeout is the timeout to consider last input from STDIN/network
	// as a complete chunk for multiline matching
	MultilinePipeChunkTimeout = 150 * time.Millisecond
	// MaxDirRecursionRoutines is the maximum number of parallel routines used
	// to recurse into directories
	MaxDirRecursionRoutines = 3
	SiftConfigFile          = ".sift.conf"
	SiftVersion             = "0.9.0"
)

type ConditionType int

const (
	ConditionPreceded ConditionType = iota
	ConditionFollowed
	ConditionSurrounded
	ConditionFileMatches
	ConditionLineMatches
	ConditionRangeMatches
)
