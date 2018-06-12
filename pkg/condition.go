package sift

import (
	"regexp"
)

type Condition struct {
	regex          *regexp.Regexp
	conditionType  ConditionType
	within         int64
	lineRangeStart int64
	lineRangeEnd   int64
	negated        bool
}
