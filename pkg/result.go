package sift

type Result struct {
	conditionMatches Matches
	matches          Matches
	// if too many matches are found or input is read only from STDIN,
	// matches are streamed through a channel
	matchChan chan Matches
	streaming bool
	isBinary  bool
	target    string
}
