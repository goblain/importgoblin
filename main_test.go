package main

import (
	"strconv"
	"testing"
)

type TestCaseExcluded struct {
	Exclude   string
	ResultMap map[string]bool
}

func TestExcluded(t *testing.T) {
	tests := &[]TestCaseExcluded{
		{Exclude: "", ResultMap: map[string]bool{"some/path/one": false, "some/excluded/path": false}},
		{Exclude: "exclude", ResultMap: map[string]bool{"some/path/one": false, "some/excluded/path": true}},
	}
	for _, testcase := range *tests {
		ic.exclude = testcase.Exclude
		for k, v := range testcase.ResultMap {
			if excluded(k) != v {
				t.Fatalf("Path '%s' returned excluded == %s where %s expected", k, strconv.FormatBool(!v), strconv.FormatBool(v))
			}
		}
	}
	//, "some/path/two", "some/excluded/path", "/yet/another/path"}
}
