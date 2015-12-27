// Load configuration object.
// key=value store with sections; so really
// section+key = values

// Retrieval is done on section+key; and if missing, on "default"+key

// This is used for general configuration;
// plus the zone.conf data; plus synthetically driven data

package main

import (
	"fmt"
	"testing"
)

func TestNewConfig(t *testing.T) {
	initGlobal("t/etc")

	c := NewConfig()

	// Go with the "blessed results" approach.
	want :=
		`&main.Config{FileInfo:main.FileInfoType{Name:"", Mtime:time.Time{sec:0, nsec:0, loc:(*time.Location)(nil)}}, Data:map[main.ConfigKey]main.ConfigVal{}, last:main.ConfigKey{Section:"default", Name:"unspecified"}}`
	have := fmt.Sprintf("%#v", c)
	if want == have {
		t.Logf("NewConfig() good")
	} else {
		t.Logf("wanted: %s", want)
		t.Fatalf("found: %s", have)
	}

}

func TestNewConfigFromFile(t *testing.T) {
	initGlobal("t/etc")

	filename := "t/test1.conf"
	test1, err := NewConfigFromFile(filename)
	if err != nil {
		t.Fatalf("error opening %s: %v", filename, err)
	}

	// Test several things about this file.
	a, _ := test1.GetSectionNameValueString("not-specific", "a")

	if a != "a" {
		t.Fatalf("expected value for 'a', got '%v'", a)
	}

	b, _ := test1.GetSectionNameValueString("not-specific", "b")
	if b != "b" {
		t.Fatalf("expected value for 'b', got '%v'", b)
	}

	b, _ = test1.GetSectionNameValueString("default", "b")
	if b != "b" {
		t.Fatalf("expected value for 'b', got '%v'", b)
	}

	empty, _ := test1.GetSectionNameValueString("not-specific", "not-specific")
	if empty != "" {
		t.Fatalf("unexpected value for 'empty', got '%v'", empty)
	}
	booltrue, _ := test1.GetSectionNameValueBool("special", "booltrue")
	if booltrue != true {
		t.Fatalf("unexpected value for 'booltrue', got '%v'", booltrue)
	}
	bool1, _ := test1.GetSectionNameValueBool("special", "bool1")
	if bool1 != true {
		t.Fatalf("unexpected value for 'booltrue', got '%v'", booltrue)
	}
	boolfalse, _ := test1.GetSectionNameValueBool("special", "boolfalse")
	if boolfalse != false {
		t.Fatalf("unexpected value for 'boolfalse', got '%v'", boolfalse)
	}

	testint, _ := test1.GetSectionNameValueInt("special", "int")
	if testint != 1 {
		t.Fatalf("unexpected value for 'int', expected 1, got '%v'", testint)
	}

	teststring, _ := test1.GetSectionNameValueString("special", "string")
	if teststring != "sample" {
		t.Fatalf("unexpected value for 'string', expected 'sample', got '%v'", teststring)
	}
	list1, _ := test1.GetSectionNameValueStrings("special", "list1")
	if len(list1) != 2 || list1[0] != "item1" || list1[1] != "item2" {
		t.Fatalf("unexpected value for 'list', expected %v, got %v", []string{"item1", "item2"}, list1)
	}
	list2, _ := test1.GetSectionNameValueStrings("special", "list2")
	if len(list2) != 2 || list2[0] != "item1" || list2[1] != "item2" {
		t.Fatalf("unexpected value for 'list2', expected %v, got %v", []string{"item1", "item2"}, list2)
	}

}

func TestConfigNeedReload(t *testing.T) {
	initGlobal("t/etc")

	c, _ := NewConfigFromFile("t/test1.conf")
	want := false
	have := c.NeedReload()
	if want != have {
		t.Logf("wanted: %v", want)
		t.Fatalf("found: %v", have)
	}
}
