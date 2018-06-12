package sift

import (
	"io"
	"regexp"
	"sync"

	// internal
	gitignore "github.com/sniperkit/sift/plugin/gitignore"
)

var global *Global

// Global specifies
type Global struct {
	// public
	Conditions         []Condition
	StreamingAllowed   bool
	StreamingThreshold int
	FileTypesMap       map[string]FileType
	OutputFilePath     string
	MatchPatterns      []string

	// private
	filesChan             chan string
	directoryChan         chan string
	includeFilepathRegex  *regexp.Regexp
	excludeFilepathRegex  *regexp.Regexp
	netTcpRegex           *regexp.Regexp
	outputFile            io.Writer
	matchRegexes          []*regexp.Regexp
	gitignoreCache        *gitignore.GitIgnoreCache
	resultsChan           chan *Result
	resultsDoneChan       chan struct{}
	targetsWaitGroup      sync.WaitGroup
	recurseWaitGroup      sync.WaitGroup
	termHighlightFilename string
	termHighlightLineno   string
	termHighlightMatch    string
	termHighlightReset    string
	totalLineLengthErrors int64
	totalMatchCount       int64
	totalResultCount      int64
	totalTargetCount      int64
}

/*
func (g *Global) WithGlobals(g *Global) *Global {
	if g.OutputFilePath == "" {
		g.OutputFilePath = "sift.log"
	}
	g.netTcpRegex = regexp.MustCompile(`^(tcp[46]?)://(.*:\d+)$`)
	if g.StreamingThreshold <= 0 {
		g.StreamingThreshold = 1 << 16
	}
	return g
}

func (g *Global) WithOptions(opts *Options) *Global {
	options = opts
}

func (g *Global) WithMatchPatterns(patterns []string) *Global {
	var err error
	matchRegexes := make([]*regexp.Regexp, len(patterns))
	for i := range patterns {
		matchRegexes[i], err = regexp.Compile(patterns[i])
		if err != nil {
			logger.Fatalf("cannot parse pattern: %s\n", err)
		}
	}
	g.matchRegexes = matchRegexes
	return g
}
*/

func SetGlobals(g *Global) {
	if g.OutputFilePath == "" {
		g.OutputFilePath = "sift.log"
	}
	g.netTcpRegex = regexp.MustCompile(`^(tcp[46]?)://(.*:\d+)$`)
	if g.StreamingThreshold <= 0 {
		g.StreamingThreshold = 1 << 16
	}
	global = g
}

func SetMatchRegexes(patterns []string) []*regexp.Regexp {
	var err error
	matchRegexes := make([]*regexp.Regexp, len(patterns))
	for i := range patterns {
		matchRegexes[i], err = regexp.Compile(patterns[i])
		if err != nil {
			logger.Fatalf("cannot parse pattern: %s\n", err)
			return nil
		}
	}
	return matchRegexes
}
