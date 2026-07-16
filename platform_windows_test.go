//go:build windows

package main

import (
	"reflect"
	"testing"
)

func TestParseWindowsScreenSaverOptions(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    appOptions
		wantErr bool
	}{
		{name: "screensaver", args: []string{"/s"}, want: appOptions{monitor: 1, screenSaver: true}},
		{name: "preview separated handle", args: []string{"/p", "1234"}, want: appOptions{monitor: 1, screenSaver: true, previewParent: 1234}},
		{name: "preview attached handle", args: []string{"/P:0x4d2"}, want: appOptions{monitor: 1, screenSaver: true, previewParent: 1234}},
		{name: "configuration", args: []string{"/c:1234"}, want: appOptions{monitor: 1, windowed: true, menu: true, configuration: true}},
		{name: "missing preview handle", args: []string{"/p"}, wantErr: true},
		{name: "invalid preview handle", args: []string{"/p:not-a-window"}, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseOptions(test.args)
			if (err != nil) != test.wantErr {
				t.Fatalf("parseOptions() error = %v, wantErr %t", err, test.wantErr)
			}
			if !test.wantErr && !reflect.DeepEqual(got, test.want) {
				t.Fatalf("parseOptions() = %#v, want %#v", got, test.want)
			}
		})
	}
}
