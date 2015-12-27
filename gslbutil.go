package main

// This file is full of crap that doesn't fit anywhere else.

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
	"unicode"
	// "fmt"
)

// fileInfo contains information about the file we loaded, in case we need to monitor it.
type FileInfoType struct {
	Name  string
	Mtime time.Time
}

// FileModifiedSince Check if a given file and timestamp suggest we should reload.
func FileModifiedSince(fileInfo FileInfoType) bool {
	//fn string, ModTime time.Time
	if fileInfo.Name == "" {
		//    fmt.Printf("FileModifiedSince(%v,%v) - no filename\n",fn,ModTime)
		return false // No filename, no change.
	}
	fi, err := os.Stat(fileInfo.Name)
	if err != nil {
		//    fmt.Printf("FileModifiedSince(%v,%v) - stat failed\n",fn,ModTime)
		return false // No file - don't you dare reload
	}
	t := fi.ModTime()
	//    fmt.Printf("FileModifiedSince(%v,%v) - t>modtime=%v\n",fn,ModTime,t.Unix() > ModTime.Unix())
	return t.Unix() > fileInfo.Mtime.Unix() // Newer?
}

// FileModifiedInfo returns info about the last modified time for a file
func FileModifiedInfo(name string) (fileInfo FileInfoType, ok bool) {
	if name == "" {
		return fileInfo, true
	}
	fi, err := os.Stat(name)
	if err != nil {
		return fileInfo, false
	}
	fileInfo.Name = name
	fileInfo.Mtime = fi.ModTime()
	return fileInfo, true
}

// Taken from a forum post by https://plus.google.com/u/0/+SoheilHassasYeganeh/posts
// Posted here https://groups.google.com/forum/#!topic/golang-nuts/pNwqLyfl2co
// Code posted here http://play.golang.org/p/ztqfYiPSlv
// This will take a string like  one two "three point one three point two" four
// and return 4 strings

// This KEEPS THE QUOTES with the words.
//  ie,    a b "C D"
// yields
//   a
//   b
//   "C D"

// QuotedStringToWords converts a string into roughly shellwords.
func QuotedStringToWords(s string) []string {
	// Maybe we have this already.  One can hope, at least.
	if v, ok := getLookupQWCache(s); ok {
		return v
	}

	lastQuote := rune(0)
	f := func(c rune) bool {
		switch {
		case c == lastQuote:
			lastQuote = rune(0)
			return false
		case lastQuote != rune(0):
			return false
		case unicode.In(c, unicode.Quotation_Mark):
			lastQuote = c
			return false
		default:
			return unicode.IsSpace(c)

		}
	}
	m := strings.FieldsFunc(s, f)

	// Save for next time.
	setLookupQWCache(s, m)

	return m
}

var stderr = log.New(os.Stderr, "", log.Ldate|log.Ltime|log.Lshortfile)

// Debugf prints to stdout (may later be a log file) but only if the -d flag is specified on the cli
func Debugf(format string, a ...interface{}) {
	if *debugFlag {
		stderr.Output(3, fmt.Sprintf(format, a...)) // This is a bit of peeking at how log.Printf() works.
	}
}

// Returns a slice of a string, length = number of callers on the function stack
// Used to indent messages in functions that recurse
func indentSpaces(c int) string {
	// lotsOfSpace is just a big string used by callerSpaces()
	const lotsOfSpace string = "                                                                                                                     "

	//pc := make([]uintptr, 100)
	//c := runtime.Callers(0, pc) // force stack walk
	c = c * 2                 // Make it visually line up easier
	if c > len(lotsOfSpace) { // But makes sure it isn't too big
		c = len(lotsOfSpace) // maxing out at whatever our constant holds
	}
	s := lotsOfSpace[0:c]
	return s
}
