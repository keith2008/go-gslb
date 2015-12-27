package main

/*
Server configs, zone data, and derived data use a common format,
with a standard set of getters/setters.  Lookup keys have
both a "section" and a "name", that combine to be the real key.
Lookups will first look for the section specified; and then try
again (if needed) to look for section="default", with the specified name.

This lets us have zones with overrides.

This also lets us have server configs with hostname based
overrides.

The maxmind cache that is built (after loading zone data) also
uses this cache; in that case, the section name is always "default".
*/

/*
IMPORTANT NOTES FOR COPROCESSING

Routines that generate a configuration object are threadsafe.

For RO: Finished objects can be used as-is.
For RW: Finished objects should be wrapped with a lock.
Locking is not provided here, as it is not needed.

General strategy for this product:
  - New objects: no lock
  - Finished objects: copy *Config to global variable with RW lock, with appropriate Set function
  - Using objects: routines get "latest" copy of global variable with RO lock

 You can see this in action by looking at global.go:
 getter GlobalConfig() and setter SetGlobalConfig


Reminders with the function comments will be listed as:
Threadsafe: Yes     -- Function does not create or interact with other goroutines
Threadsafe: for RO  -- If all users of the object are RO, you're safe.
Threadsafe: NO      -- Must be single threaded, or mutex locked.
*/

import (
	"bufio"
	//	"fmt"
	"errors"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

//
// config.filename
// config.time

// ConfigKey is a map key for generic configuration data.
// The key contains both a Section and Name, where you might see
// the config file look like this:
// [Section]
// Name: value
type ConfigKey struct {
	Section string
	Name    string
}

// ConfigVal stores one or more strings in the cache.
// When multiple values are added to the cache, the first
// one passed will always be available as "First" (as a string).
// The entire array, including the First, is also in Values.
// [Section]
// Name: [First, Value2, Value3]
type ConfigVal struct {
	First  string
	Values []string
}

// Config is a generic container for section/name=values
type Config struct {
	FileInfo FileInfoType
	Data     map[ConfigKey]ConfigVal
	last     ConfigKey
}

// NewConfig simply creates an empty *Config .
// Threadsafe: Yes
func NewConfig() *Config {
	c := new(Config)
	c.Data = make(map[ConfigKey]ConfigVal, 1000)
	c.last = ConfigKey{"default", "unspecified"}
	return c
}

// NewConfigFromFile generates a new Config object from a
// file name.  The line is read line by line.
// Returns a *Config; sets "error" with the first error found
// if there is one.  Even if there are multiple errors,
// only the first is returned; and as much of the file is
// parsed as possible.
// Threadsafe: Yes until function returns
func NewConfigFromFile(name string) (*Config, error) {

	var firsterror error
	c := NewConfig() // *Config
	c.FileInfo, _ = FileModifiedInfo(name)

	// Open the file, read lines parse.
	file, err := os.Open(name)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		//		fmt.Println(scanner.Text())
		err = c.AddLine(scanner.Text())
		if firsterror == nil {
			firsterror = err
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error scanning config %s: %v", name, err)
		return c, err
	}

	return c, firsterror
}

// NewConfigFromString generates a new Config object from a
// multiline string.  The line is broken on newline and then
// parsed line by line, just as if it were a file.
// Returns a *Config; sets "error" with the first error found
// if there is one.  Even if there are multiple errors,
// only the first is returned; and as much of the file is
// parsed as possible.
// Threadsafe: Yes until function returns
func NewConfigFromString(s string) (*Config, error) {
	var firsterror error
	c := NewConfig() // *Config

	for _, line := range strings.Split(string(s), "\n") {
		err := c.AddLine(line)
		if firsterror == nil {
			firsterror = err
		}
	}
	return c, firsterror
}

// NeedReload returns true only if the *Config
// was loaded from a file, and only if the modify
// time on disk is newer than when we loaded.
// Threadsafe: for RO
func (c *Config) NeedReload() bool {
	return FileModifiedSince(c.FileInfo)
}

// GetSectionNameData gets the entire ConfigVal for a given section and name.
// Use only if "ok".
// Threadsafe: for RO
func (c *Config) GetSectionNameData(section string, name string) (val ConfigVal, ok bool) {
	val, ok = c.Data[ConfigKey{section, name}]
	if ok {
		return val, ok
	}
	val, ok = c.Data[ConfigKey{"default", name}]
	return val, ok

}

// GetSectionNameValueStrings gets the []strings slice for a given section and name.
// Use only if "ok".
// Threadsafe: for RO
func (c *Config) GetSectionNameValueStrings(section string, name string) (values []string, ok bool) {
	val, ok := c.GetSectionNameData(section, name)
	if ok {
		return val.Values, ok

	}
	return nil, false
}

// GetSectionNameValueString gets the first defined value for a given section and name.
// Use only if "ok".   Principally used for reading the configuration data (vs zone data).
// Threadsafe: for RO
func (c *Config) GetSectionNameValueString(section string, name string) (value string, ok bool) {
	val, ok := c.GetSectionNameData(section, name)
	if ok {
		return val.First, ok
	}
	return "", false
}

// GetSectionNameValueInt gets the first defined value for a given section and name,
// parsed into an int . Use only if "ok."
// Threadsafe: for RO
func (c *Config) GetSectionNameValueInt(section string, name string) (value int, ok bool) {
	val, ok := c.GetSectionNameData(section, name)
	if ok {
		if i, err := strconv.Atoi(val.First); err == nil {
			return i, ok
		}
	}
	return 0, false
}

// GetSectionNameValueBool gets the first defined value for a given section and name,
// parsed into true or false.  Any string starting with a non-0 digit, or any of
// the common true names (yes,Yes,true,True) will parse as true.  Any others
// will be false.  Use only if "ok".
// Threadsafe: for RO
func (c *Config) GetSectionNameValueBool(section string, name string) (value bool, ok bool) {

	ret := false

	val, ok := c.GetSectionNameData(section, name)
	if ok {
		switch val.First[0:1] {
		case "y", "Y", "t", "T", "1", "2", "3", "4", "5", "6", "7", "8", "9":
			ret = true
		case "n", "N", "f", "F", "0":
			ret = false
		default:
			ret = false
		}
		return ret, true
	}
	return false, false
}

// AddValue adds another value to the last used section/name.
// Threadsafe: NO
func (c *Config) AddValue(value string) {
	c.AddKeyValue(c.last, value)
}

// AddKeyValue adds another value, for a specified key.
// Keys can be given on the fly:   ConfigKey{section,name}
// Threadsafe: NO
func (c *Config) AddKeyValue(key ConfigKey, value string) {
	if key.Section == "" {
		return // Do nothing, if the section is blank (unwanted)
	}

	// If this looks like an array collapsed into a single line, expand and recurse
	// into our own function.  This allows you to send
	// a value such as string{}"[one, two, three"} and have it
	// stored as []string{"one", "two", "three"}
	matches := reArray.FindStringSubmatch(value)
	if matches != nil {
		inside := matches[1]
		values := strings.Split(inside, ", ")
		for _, t := range values {
			c.AddKeyValue(key, t)
		}
		return
	}

	// Do we already have somethingin the cache?
	var newVal ConfigVal
	_, ok := c.Data[key]

	// Clean up leading, trailing, and redundant middle whitespace
	value = strings.Join(QuotedStringToWords(value), " ") // Hmm.

	if ok == true {
		newVal.First = c.Data[key].First
		newVal.Values = append(c.Data[key].Values, value)
	} else {
		newVal.First = value
		newVal.Values = make([]string, 1)
		newVal.Values[0] = value
	}
	c.Data[key] = newVal
}

// SetKeyValue will, for a specified Configkey,
// store the new value.  Values will be updated;
// and (if this is the first time) First will be set.
// Threadsafe: NO
func (c *Config) SetKeyValue(key ConfigKey, value string) {
	if key.Section == "" {
		return // Do nothing, if the section is blank (unwanted)
	}
	var newVal ConfigVal
	newVal.First = value
	newVal.Values = make([]string, 1)
	newVal.Values[0] = value
	c.Data[key] = newVal
}

// The section name may be "default" or "mumble"
// or it may be hostname specific; ie "default/Jasons-MacBook.local" or "mumble/Jasons-MacBook.local".
// If a slash is found, it will look for a hostname on the right hand side, and (effectively) discard
// the data if hostname is not ours.
// Threadsafe: NO
func (c *Config) setSection(s string) {
	sp := strings.SplitN(s, "/", 2)
	if len(sp) < 2 {
		c.last.Section = s // No funny business.
	} else {
		if sp[1] == ourHostname() {
			c.last.Section = sp[0] + "" // I don't want the old slice; I want a new string.  I think.
		} else {
			c.last.Section = "discarded" // Effectively, I don't want this section.
		}
	}
}

// setName sets the current variable name (to be used with AddValue).
// Threadsafe: NO
func (c *Config) setName(s string) { c.last.Name = s }

// Regex. "And now you have two problems."

var reComment = regexp.MustCompile(`#.*$`)               // #
var reSection = regexp.MustCompile(`^\[(.*)\]$`)         // [foo]
var reKey = regexp.MustCompile(`^(\S+):$`)               // key:
var reKeyValue = regexp.MustCompile(`^(\S+):\s+(\S.*)$`) // key: value
var reValue = regexp.MustCompile(`^\s*[:-]\s*(\S.*)$`)   // - value   or  //  : value
var reArray = regexp.MustCompile(`^\[(.*)\]$`)           // [value], as in key: [value, value, value]

// AddLine will parse a single line destined for the *Config .
// The line is examined for [section] names, and for
// name: value pairs.  A name can have more than one
// value by using YAML-ish syntax:
//
// [section]
// name: value
// name2: [value1, value2]
// name3:
//    - value1
//    - value2
//    - Value3
//
// Additionally, section names may by "default" or a given name.
// Section names may also identify a hostname requirement.
// [default/Jasons-MacBook.local]
// debug: 1
//
// Threadsafe: NO
func (c *Config) AddLine(s string) (err error) {
	// Identify what kind of line is this
	// Identify if there is a section name, a key name, and/or a value
	// and do the needful.

	s = reComment.ReplaceAllLiteralString(s, "") // Remove comments
	s = strings.TrimSpace(s)                     // Remove leading and trailing whitespace
	if s == "" {
		return nil
	}

	// Check for [section]
	var matches []string
	matches = reSection.FindStringSubmatch(s)
	if matches != nil {
		c.setSection(matches[1])
		return nil
	}
	matches = reKey.FindStringSubmatch(s)
	if matches != nil {
		c.setName(matches[1])
		return nil
	}
	matches = reKeyValue.FindStringSubmatch(s)
	if matches != nil {
		c.setName(matches[1])
		c.AddValue(matches[2])
		return nil
	}
	matches = reValue.FindStringSubmatch(s)
	if matches != nil {
		c.AddValue(matches[1])
		return nil
	}
	return errors.New("Unexpected text parsing config line: " + s)
}

// ourHostname returns the current host we are on.
// Used for config parsing to identify host-specific configuration.
func ourHostname() string {
	// Cache the hostname we are running on.
	// We will use this as part of our key lookup strategy.
	h, e := os.Hostname()
	if e == nil {
		return h
	}
	return "bad-hostname"
}
