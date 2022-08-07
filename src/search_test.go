package main

import (
	"testing"
)

func TestSearchDirectory(t *testing.T) {
	charles, err := searchFiles(`f:\dropbox\swap\golang\helm\fixtures\search\a`, "search term")
	if err != nil {
		t.Fatalf("Failed to execute at all %s\n", err)
	}
	if len(charles) != 2 {
		t.Fatalf("Wrong number of solutions found %d\n", len(charles))
	}
}
