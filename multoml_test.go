/*
 * BSD 3-Clause License
 * Copyright (c) 2018, Psiphon Inc.
 * All rights reserved.
 */

package multoml

import (
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestNewFromFiles(t *testing.T) {
	type args struct {
		filenames    []string
		searchPaths  []string
		envOverrides map[string]string
	}
	tests := []struct {
		name              string
		args              args
		environmentValues map[string]string
		wantConfigFname   string
		wantFilesUsed     []string
		wantErr           bool
	}{
		{
			name: "Success: simple, no override",
			args: args{
				filenames:    []string{"t1.toml"},
				searchPaths:  []string{"testdata"},
				envOverrides: nil,
			},
			wantConfigFname: "testdata/t1-want.toml",
			wantFilesUsed:   []string{"testdata/t1.toml"},
			wantErr:         false,
		},
		{
			name: "Success: has override",
			args: args{
				filenames:    []string{"t2.toml", "t2_override.toml"},
				searchPaths:  []string{"invalid-asdfljjl", "testdata"},
				envOverrides: nil,
			},
			wantConfigFname: "testdata/t2-want.toml",
			wantFilesUsed:   []string{"testdata/t2.toml", "testdata/t2_override.toml"},
			wantErr:         false,
		},
		{
			name: "Success: has override, some nonexisting",
			args: args{
				filenames:    []string{"t2.toml", "asdfijfdij.toml", "t2_override.toml", "fdhvlja.toml"},
				searchPaths:  []string{"invalid-asdfljjl", "testdata"},
				envOverrides: nil,
			},
			wantConfigFname: "testdata/t2-want.toml",
			wantFilesUsed:   []string{"testdata/t2.toml", "", "testdata/t2_override.toml", ""},
			wantErr:         false,
		},
		{
			name: "Error: no filenames provided",
			args: args{
				filenames:    nil,
				searchPaths:  nil,
				envOverrides: nil,
			},
			wantConfigFname: "",
			wantFilesUsed:   nil,
			wantErr:         true,
		},
		{
			name: "Error: file doesn't exist",
			args: args{
				filenames:    []string{"afdsljkfads.toml"},
				searchPaths:  []string{"testdata"},
				envOverrides: nil,
			},
			wantConfigFname: "",
			wantFilesUsed:   nil,
			wantErr:         true,
		},
		{
			name: "Error: invalid TOML",
			args: args{
				filenames:    []string{"invalid.toml"},
				searchPaths:  []string{"testdata"},
				envOverrides: nil,
			},
			wantConfigFname: "",
			wantFilesUsed:   nil,
			wantErr:         true,
		},
		{
			name: "Success: nested, has override",
			args: args{
				filenames:    []string{"t3.toml", "t3_override.toml"},
				searchPaths:  []string{"testdata", "invalid-asdfljjl"},
				envOverrides: nil,
			},
			wantConfigFname: "testdata/t3-want.toml",
			wantFilesUsed:   []string{"testdata/t3.toml", "testdata/t3_override.toml"},
			wantErr:         false,
		},
		{
			name: "Success: multiple overrides",
			args: args{
				filenames:    []string{"t4.toml", "t4_override.toml", "t4_override_again.toml"},
				searchPaths:  []string{"invalid-asdfljjl", "testdata"},
				envOverrides: nil,
			},
			wantConfigFname: "testdata/t4-want.toml",
			wantFilesUsed:   []string{"testdata/t4.toml", "testdata/t4_override.toml", "testdata/t4_override_again.toml"},
			wantErr:         false,
		},
		{
			name: "Success: override from environment variables",
			args: args{
				filenames:    []string{"t5.toml"},
				searchPaths:  []string{"testdata"},
				envOverrides: map[string]string{"B": "b", "D": "section.d"},
			},
			environmentValues: map[string]string{"B": "environment-b", "D": "environment-d"},
			wantConfigFname:   "testdata/t5-want.toml",
			wantFilesUsed:     []string{"testdata/t5.toml"},
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: Will setting these interfere with subsequent tests? Probably. Need to reset them.
			if len(tt.environmentValues) > 0 {
				for k, v := range tt.environmentValues {
					os.Setenv(k, v)
				}
			}

			gotConf, gotFilesUsed, err := NewFromFiles(tt.args.filenames, tt.args.searchPaths, tt.args.envOverrides)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFromFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			wantConfig, err := ioutil.ReadFile(tt.wantConfigFname)
			gotConfString, err := gotConf.ToTomlString()
			if err != nil {
				t.Fatalf("gotConf.ToTomlString failed: %v", err)
			}
			if string(wantConfig) != gotConfString {
				t.Errorf("gotConf = {%v}, want {%v}", gotConfString, string(wantConfig))
			}

			if !filePathSlicesEqual(gotFilesUsed, tt.wantFilesUsed) {
				t.Errorf("gotFilesUsed = %v, want %v", gotFilesUsed, tt.wantFilesUsed)
			}
		})
	}
}

func filePathSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if strings.Replace(a[i], "\\", "/", -1) != strings.Replace(b[i], "\\", "/", -1) {
			return false
		}
	}

	return true
}

func TestNewFromReaders(t *testing.T) {
	type args struct {
		filenames    []string
		envOverrides map[string]string
	}
	tests := []struct {
		name              string
		args              args
		environmentValues map[string]string
		wantConfigFname   string
		wantErr           bool
	}{
		{
			name: "Success: simple, no override",
			args: args{
				filenames:    []string{"testdata/t1.toml"},
				envOverrides: nil,
			},
			wantConfigFname: "testdata/t1-want.toml",
			wantErr:         false,
		},
		{
			name: "Error: no readers provided",
			args: args{
				filenames:    nil,
				envOverrides: nil,
			},
			wantConfigFname: "",
			wantErr:         true,
		},
		{
			name: "Error: nil reader",
			args: args{
				filenames:    []string{""},
				envOverrides: nil,
			},
			wantConfigFname: "",
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: Will setting these interfere with subsequent tests? Probably. Need to reset them.
			if len(tt.environmentValues) > 0 {
				for k, v := range tt.environmentValues {
					os.Setenv(k, v)
				}
			}

			readers := make([]io.Reader, len(tt.args.filenames))
			for i, fname := range tt.args.filenames {
				if fname == "" {
					// special value to set nil reader
					readers[i] = nil
					continue
				}

				f, err := os.Open(fname)
				if err != nil {
					t.Fatalf("unable to open input file: %s", fname)
				}
				defer f.Close()
				readers[i] = f
			}

			gotConf, err := NewFromReaders(readers, tt.args.envOverrides)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFromReaders() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			wantConfig, err := ioutil.ReadFile(tt.wantConfigFname)
			gotConfString, err := gotConf.ToTomlString()
			if err != nil {
				t.Fatalf("gotConf.ToTomlString failed: %v", err)
			}
			if string(wantConfig) != gotConfString {
				t.Errorf("gotConf = {%v}, want {%v}", gotConfString, string(wantConfig))
			}
		})
	}
}
