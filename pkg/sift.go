// sift
// Copyright (C) 2014-2016 Sven Taute
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, version 3 of the License.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package sift

import (
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	// internal
	gitignore "github.com/sniperkit/sift/plugin/gitignore"
	nbreader "github.com/sniperkit/sift/plugin/nbreader"
)

var (
	InputBlockSize int = 256 * 1024
)

// type Sift struct {}

// processDirectories reads global.directoryChan and processes
// directories via processDirectory.
func processDirectories() {
	n := options.Cores
	if n > MaxDirRecursionRoutines {
		n = MaxDirRecursionRoutines
	}
	for i := 0; i < n; i++ {
		go func() {
			for dirname := range global.directoryChan {
				processDirectory(dirname)
			}
		}()
	}
}

// enqueueDirectory enqueues directories on global.directoryChan.
// If the channel blocks, the directory is processed directly.
func enqueueDirectory(dirname string) {
	global.recurseWaitGroup.Add(1)
	select {
	case global.directoryChan <- dirname:
	default:
		processDirectory(dirname)
	}
}

// processDirectory recurses into a directory and sends all files
// fulfilling the selected options on global.filesChan
func processDirectory(dirname string) {
	defer global.recurseWaitGroup.Done()
	var gic *gitignore.Checker
	if options.Git {
		gic = gitignore.NewCheckerWithCache(global.gitignoreCache)
		err := gic.LoadBasePath(dirname)
		if err != nil {
			logger.Printf("cannot load gitignore files for path '%s': %s", dirname, err)
		}
	}
	dir, err := os.Open(dirname)
	if err != nil {
		logger.Printf("cannot open directory '%s': %s\n", dirname, err)
		return
	}
	defer dir.Close()
	for {
		entries, err := dir.Readdir(256)
		if err == io.EOF {
			return
		}
		if err != nil {
			logger.Printf("cannot read directory '%s': %s\n", dirname, err)
			return
		}

	nextEntry:
		for _, fi := range entries {
			fullpath := filepath.Join(dirname, fi.Name())

			// check directory include/exclude options
			if fi.IsDir() {
				if !options.Recursive {
					continue nextEntry
				}
				for _, dirPattern := range options.ExcludeDirs {
					matched, err := filepath.Match(dirPattern, fi.Name())
					if err != nil {
						logger.Fatalf("cannot match malformed pattern '%s' against directory name: %s\n", dirPattern, err)
					}
					if matched {
						continue nextEntry
					}
				}
				if len(options.IncludeDirs) > 0 {
					for _, dirPattern := range options.IncludeDirs {
						matched, err := filepath.Match(dirPattern, fi.Name())
						if err != nil {
							logger.Fatalf("cannot match malformed pattern '%s' against directory name: %s\n", dirPattern, err)
						}
						if matched {
							goto includeDirMatchFound
						}
					}
					continue nextEntry
				includeDirMatchFound:
				}
				if options.Git {
					if fi.Name() == gitignore.GitFoldername || gic.Check(fullpath, fi) {
						continue nextEntry
					}
				}
				enqueueDirectory(fullpath)
				continue nextEntry
			}

			// check whether this is a regular file
			if fi.Mode()&os.ModeType != 0 {
				if options.FollowSymlinks && fi.Mode()&os.ModeType == os.ModeSymlink {
					realPath, err := filepath.EvalSymlinks(fullpath)
					if err != nil {
						logger.Printf("cannot follow symlink '%s': %s\n", fullpath, err)
					} else {
						realFi, err := os.Stat(realPath)
						if err != nil {
							logger.Printf("cannot follow symlink '%s': %s\n", fullpath, err)
						}
						if realFi.IsDir() {
							enqueueDirectory(realPath)
							continue nextEntry
						} else {
							if realFi.Mode()&os.ModeType != 0 {
								continue nextEntry
							}
						}
					}
				} else {
					continue nextEntry
				}
			}

			// check file path options
			if global.excludeFilepathRegex != nil {
				if global.excludeFilepathRegex.MatchString(fullpath) {
					continue nextEntry
				}
			}
			if global.includeFilepathRegex != nil {
				if !global.includeFilepathRegex.MatchString(fullpath) {
					continue nextEntry
				}
			}

			// check file extension options
			if len(options.ExcludeExtensions) > 0 {
				for _, e := range strings.Split(options.ExcludeExtensions, ",") {
					if filepath.Ext(fi.Name()) == "."+e {
						continue nextEntry
					}
				}
			}
			if len(options.IncludeExtensions) > 0 {
				for _, e := range strings.Split(options.IncludeExtensions, ",") {
					if filepath.Ext(fi.Name()) == "."+e {
						goto includeExtensionFound
					}
				}
				continue nextEntry
			includeExtensionFound:
			}

			// check file include/exclude options
			for _, filePattern := range options.ExcludeFiles {
				matched, err := filepath.Match(filePattern, fi.Name())
				if err != nil {
					logger.Fatalf("cannot match malformed pattern '%s' against file name: %s\n", filePattern, err)
				}
				if matched {
					continue nextEntry
				}
			}
			if len(options.IncludeFiles) > 0 {
				for _, filePattern := range options.IncludeFiles {
					matched, err := filepath.Match(filePattern, fi.Name())
					if err != nil {
						logger.Fatalf("cannot match malformed pattern '%s' against file name: %s\n", filePattern, err)
					}
					if matched {
						goto includeFileMatchFound
					}
				}
				continue nextEntry
			includeFileMatchFound:
			}

			// check file type options
			if len(options.ExcludeTypes) > 0 {
				for _, t := range strings.Split(options.ExcludeTypes, ",") {
					for _, filePattern := range global.FileTypesMap[t].Patterns {
						if matched, _ := filepath.Match(filePattern, fi.Name()); matched {
							continue nextEntry
						}
					}
					sr := global.FileTypesMap[t].ShebangRegex
					if sr != nil {
						if m, err := checkShebang(global.FileTypesMap[t].ShebangRegex, fullpath); m && err == nil {
							continue nextEntry
						}
					}
				}
			}
			if len(options.IncludeTypes) > 0 {
				for _, t := range strings.Split(options.IncludeTypes, ",") {
					for _, filePattern := range global.FileTypesMap[t].Patterns {
						if matched, _ := filepath.Match(filePattern, fi.Name()); matched {
							goto includeTypeFound
						}
					}
					sr := global.FileTypesMap[t].ShebangRegex
					if sr != nil {
						if m, err := checkShebang(global.FileTypesMap[t].ShebangRegex, fullpath); err != nil || m {
							goto includeTypeFound
						}
					}
				}
				continue nextEntry
			includeTypeFound:
			}

			if options.Git {
				if fi.Name() == gitignore.GitIgnoreFilename || gic.Check(fullpath, fi) {
					continue
				}
			}

			global.filesChan <- fullpath
		}
	}
}

// checkShebang checks whether the first line of file matches the given regex
func checkShebang(regex *regexp.Regexp, filepath string) (bool, error) {
	f, err := os.Open(filepath)
	defer f.Close()
	if err != nil {
		return false, err
	}
	b, err := bufio.NewReader(f).ReadBytes('\n')
	return regex.Match(b), nil
}

// processFileTargets reads filesChan, builds an io.Reader for the target and calls processReader
func processFileTargets() {
	defer global.targetsWaitGroup.Done()
	dataBuffer := make([]byte, InputBlockSize)
	testBuffer := make([]byte, InputBlockSize)
	matchRegexes := make([]*regexp.Regexp, len(global.MatchPatterns))
	for i := range global.MatchPatterns {
		matchRegexes[i] = regexp.MustCompile(global.MatchPatterns[i])
	}

	for filepath := range global.filesChan {
		var err error
		var infile *os.File
		var reader io.Reader

		if options.TargetsOnly {
			global.resultsChan <- &Result{target: filepath}
			continue
		}

		if filepath == "-" {
			infile = os.Stdin
		} else {
			infile, err = os.Open(filepath)
			if err != nil {
				logger.Printf("cannot open file '%s': %s\n", filepath, err)
				continue
			}
		}

		if options.Zip && strings.HasSuffix(filepath, ".gz") {
			rawReader := infile
			reader, err = gzip.NewReader(rawReader)
			if err != nil {
				logger.Printf("error decompressing file '%s', opening as normal file\n", infile.Name())
				infile.Seek(0, 0)
				reader = infile
			}
		} else if infile == os.Stdin && options.Multiline {
			reader = nbreader.NewNBReader(infile, InputBlockSize,
				nbreader.ChunkTimeout(MultilinePipeChunkTimeout), nbreader.Timeout(MultilinePipeTimeout))
		} else {
			reader = infile
		}

		if options.InvertMatch {
			err = processReaderInvertMatch(reader, matchRegexes, filepath)
		} else {
			err = processReader(reader, matchRegexes, dataBuffer, testBuffer, filepath)
		}
		if err != nil {
			if err == errLineTooLong {
				global.totalLineLengthErrors += 1
				if options.ErrShowLineLength {
					errmsg := fmt.Sprintf("file contains very long lines (>= %d bytes). See options --blocksize and --err-skip-line-length.", InputBlockSize)
					logger.Printf("cannot process data from file '%s': %s\n", filepath, errmsg)
				}
			} else {
				logger.Printf("cannot process data from file '%s': %s\n", filepath, err)
			}
		}
		infile.Close()
	}
}

// processNetworkTarget starts a listening TCP socket and calls processReader
func processNetworkTarget(target string) {
	matchRegexes := make([]*regexp.Regexp, len(global.MatchPatterns))
	for i := range global.MatchPatterns {
		matchRegexes[i] = regexp.MustCompile(global.MatchPatterns[i])
	}
	defer global.targetsWaitGroup.Done()

	var reader io.Reader
	netParams := global.netTcpRegex.FindStringSubmatch(target)
	proto := netParams[1]
	addr := netParams[2]

	listener, err := net.Listen(proto, addr)
	if err != nil {
		logger.Fatalf("could not listen on '%s'\n", target)
	}

	conn, err := listener.Accept()
	if err != nil {
		logger.Fatalf("could not accept connections on '%s'\n", target)
	}

	if options.Multiline {
		reader = nbreader.NewNBReader(conn, InputBlockSize, nbreader.ChunkTimeout(MultilinePipeChunkTimeout),
			nbreader.Timeout(MultilinePipeTimeout))
	} else {
		reader = conn
	}

	dataBuffer := make([]byte, InputBlockSize)
	testBuffer := make([]byte, InputBlockSize)
	err = processReader(reader, matchRegexes, dataBuffer, testBuffer, target)
	if err != nil {
		logger.Printf("error processing data from '%s'\n", target)
		return
	}
}

func ExecuteSearch(targets []string) (ret int, err error) {
	defer func() {
		if r := recover(); r != nil {
			ret = 2
			err = errors.New(r.(string))
		}
	}()
	tstart := time.Now()
	global.filesChan = make(chan string, 256)
	global.directoryChan = make(chan string, 128)
	global.resultsChan = make(chan *Result, 128)
	global.resultsDoneChan = make(chan struct{})
	global.gitignoreCache = gitignore.NewGitIgnoreCache()
	global.totalTargetCount = 0
	global.totalLineLengthErrors = 0
	global.totalMatchCount = 0
	global.totalResultCount = 0

	go resultHandler()

	for i := 0; i < options.Cores; i++ {
		global.targetsWaitGroup.Add(1)
		go processFileTargets()
	}

	go processDirectories()

	for _, target := range targets {
		switch {
		case target == "-":
			global.filesChan <- "-"
		case global.netTcpRegex.MatchString(target):
			global.targetsWaitGroup.Add(1)
			go processNetworkTarget(target)
		default:
			fileinfo, err := os.Stat(target)
			if err != nil {
				if os.IsNotExist(err) {
					logger.Fatalf("no such file or directory: %s\n", target)
				} else {
					logger.Fatalf("cannot open file or directory: %s\n", target)
				}
			}
			if fileinfo.IsDir() {
				global.recurseWaitGroup.Add(1)
				global.directoryChan <- target
			} else {
				global.filesChan <- target
			}
		}
	}

	global.recurseWaitGroup.Wait()
	close(global.directoryChan)

	close(global.filesChan)
	global.targetsWaitGroup.Wait()

	close(global.resultsChan)
	<-global.resultsDoneChan

	var retVal int
	if global.totalResultCount > 0 {
		retVal = 0
	} else {
		retVal = 1
	}

	if !options.ErrSkipLineLength && !options.ErrShowLineLength && global.totalLineLengthErrors > 0 {
		logger.Printf("%d files skipped due to very long lines (>= %d bytes). See options --blocksize, --err-show-line-length and --err-skip-line-length.", global.totalLineLengthErrors, InputBlockSize)
	}

	if options.Stats {
		tend := time.Now()
		fmt.Fprintln(os.Stderr, global.totalTargetCount, "files processed")
		fmt.Fprintln(os.Stderr, global.totalResultCount, "files match")
		fmt.Fprintln(os.Stderr, global.totalMatchCount, "matches found")
		fmt.Fprintf(os.Stderr, "in %v\n", tend.Sub(tstart))
	}

	return retVal, nil
}
