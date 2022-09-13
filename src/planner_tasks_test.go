package main

import (
	"fmt"
	"testing"
	"time"
)

func TestExpirySeconds(t *testing.T) {
	x, y := time.ParseDuration(fmt.Sprintf("%ds", 3600))
	fmt.Printf("X %v, Y %v\n", x, y)
	t.Fatalf("Didn't get the access token expected")
}
