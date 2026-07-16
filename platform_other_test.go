//go:build !windows

package main

import (
	"reflect"
	"testing"
)

func TestNonWindowsArgumentsPassThrough(t *testing.T) {
	args := []string{"/s", "--mute"}
	got, parent, configuration, err := normalizeWindowsScreenSaverArgs(args)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, args) || parent != 0 || configuration {
		t.Fatalf("normalize result = %#v, %#x, %t", got, parent, configuration)
	}
}
