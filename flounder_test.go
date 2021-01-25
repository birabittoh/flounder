package main

import "testing"

func TestIsOKUsername(t *testing.T) {
	for _, u := range []string{"www", "proxy", "%", "", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"} {
		if isOkUsername(u) == nil {
			t.Errorf("Username " + u + " should be considered invalid, but wasn't")
		}
	}
	for _, u := range []string{"-", "alex", "1"} {
		if isOkUsername(u) != nil {
			t.Errorf("Username " + u + " should be considered valid, but wasn't")
		}
	}
}
