package sift

type Match struct {
	// offset of the start of the match
	start int64
	// offset of the end of the match
	end int64
	// offset of the beginning of the first line of the match
	lineStart int64
	// offset of the end of the last line of the match
	lineEnd int64
	// the match
	match string
	// the match including the non-matched text on the first and last line
	line string
	// the line number of the beginning of the match
	lineno int64
	// the index to global.conditions (if this match belongs to a condition)
	conditionID int
	// the context before the match
	contextBefore *string
	// the context after the match
	contextAfter *string
}

type Matches []Match

func (e Matches) Len() int           { return len(e) }
func (e Matches) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }
func (e Matches) Less(i, j int) bool { return e[i].start < e[j].start }
