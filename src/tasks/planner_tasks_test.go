package tasks

import (
	"fmt"
	"testing"
	"time"
)

func TestExpirySeconds(t *testing.T) {
	x, y := time.ParseDuration(fmt.Sprintf("%ds", 3600))
	t.Fatalf("Didn't get the access token expected")
}
