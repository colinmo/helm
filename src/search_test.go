package main

import (
	"testing"
)

func TestSearchDirectory(t *testing.T) {
	charles := searchFiles("../fixtures/search/a", "searchterm")
	if len(charles) != 2 {
		t.Fatalf("Wrong number of solutions found %d\n", len(charles))
	}
}
