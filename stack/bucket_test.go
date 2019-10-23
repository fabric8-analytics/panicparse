// Copyright 2018 Marc-Antoine Ruel. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package stack

import (
	"bytes"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"
)

func TestAggregateNotAggressive(t *testing.T) {
	// 2 goroutines with similar but not exact same signature.
	data := []string{
		"panic: runtime error: index out of range",
		"",
		"goroutine 6 [chan receive]:",
		"main.func·001(0x11000000, 2)",
		"	/gopath/src/github.com/maruel/panicparse/stack/stack.go:72 +0x49",
		"",
		"goroutine 7 [chan receive]:",
		"main.func·001(0x21000000, 2)",
		"	/gopath/src/github.com/maruel/panicparse/stack/stack.go:72 +0x49",
		"",
	}
	c, err := ParseDump(bytes.NewBufferString(strings.Join(data, "\n")), ioutil.Discard, false)
	if err != nil {
		t.Fatal(err)
	}
	actual := Aggregate(c.Goroutines, ExactLines)
	expected := []*Bucket{
		{
			Signature: Signature{
				State: "chan receive",
				Stack: Stack{
					Calls: []Call{
						{
							SrcPath: "/gopath/src/github.com/maruel/panicparse/stack/stack.go",
							Line:    72,
							Func:    Func{Raw: "main.func·001"},
							Args:    Args{Values: []Arg{{Value: 0x11000000, Name: ""}, {Value: 2}}},
						},
					},
				},
			},
			IDs:   []int{6},
			First: true,
		},
		{
			Signature: Signature{
				State: "chan receive",
				Stack: Stack{
					Calls: []Call{
						{
							SrcPath: "/gopath/src/github.com/maruel/panicparse/stack/stack.go",
							Line:    72,
							Func:    Func{Raw: "main.func·001"},
							Args:    Args{Values: []Arg{{Value: 0x21000000, Name: "#1"}, {Value: 2}}},
						},
					},
				},
			},
			IDs: []int{7},
		},
	}
	compareBuckets(t, expected, actual)
}

func TestAggregateExactMatching(t *testing.T) {
	// 2 goroutines with the exact same signature.
	data := []string{
		"panic: runtime error: index out of range",
		"",
		"goroutine 6 [chan receive]:",
		"main.func·001()",
		"	/gopath/src/github.com/maruel/panicparse/stack/stack.go:72 +0x49",
		"created by main.mainImpl",
		"	/gopath/src/github.com/maruel/panicparse/stack/stack.go:74 +0xeb",
		"",
		"goroutine 7 [chan receive]:",
		"main.func·001()",
		"	/gopath/src/github.com/maruel/panicparse/stack/stack.go:72 +0x49",
		"created by main.mainImpl",
		"	/gopath/src/github.com/maruel/panicparse/stack/stack.go:74 +0xeb",
		"",
	}
	c, err := ParseDump(bytes.NewBufferString(strings.Join(data, "\n")), &bytes.Buffer{}, false)
	if err != nil {
		t.Fatal(err)
	}
	actual := Aggregate(c.Goroutines, ExactLines)
	expected := []*Bucket{
		{
			Signature: Signature{
				State: "chan receive",
				Stack: Stack{
					Calls: []Call{
						{
							SrcPath: "/gopath/src/github.com/maruel/panicparse/stack/stack.go",
							Line:    72,
							Func:    Func{Raw: "main.func·001"},
						},
					},
				},
				CreatedBy: Call{
					SrcPath: "/gopath/src/github.com/maruel/panicparse/stack/stack.go",
					Line:    74,
					Func:    Func{Raw: "main.mainImpl"},
				},
			},
			IDs:   []int{6, 7},
			First: true,
		},
	}
	compareBuckets(t, expected, actual)
}

func TestAggregateAggressive(t *testing.T) {
	// 3 goroutines with similar signatures.
	data := []string{
		"panic: runtime error: index out of range",
		"",
		"goroutine 6 [chan receive, 10 minutes]:",
		"main.func·001(0x11000000, 2)",
		"	/gopath/src/github.com/maruel/panicparse/stack/stack.go:72 +0x49",
		"",
		"goroutine 7 [chan receive, 50 minutes]:",
		"main.func·001(0x21000000, 2)",
		"	/gopath/src/github.com/maruel/panicparse/stack/stack.go:72 +0x49",
		"",
		"goroutine 8 [chan receive, 100 minutes]:",
		"main.func·001(0x21000000, 2)",
		"	/gopath/src/github.com/maruel/panicparse/stack/stack.go:72 +0x49",
		"",
	}
	c, err := ParseDump(bytes.NewBufferString(strings.Join(data, "\n")), ioutil.Discard, false)
	if err != nil {
		t.Fatal(err)
	}
	actual := Aggregate(c.Goroutines, AnyPointer)
	expected := []*Bucket{
		{
			Signature: Signature{
				State:    "chan receive",
				SleepMin: 10,
				SleepMax: 100,
				Stack: Stack{
					Calls: []Call{
						{
							SrcPath: "/gopath/src/github.com/maruel/panicparse/stack/stack.go",
							Line:    72,
							Func:    Func{Raw: "main.func·001"},
							Args:    Args{Values: []Arg{{Value: 0x11000000, Name: "*"}, {Value: 2}}},
						},
					},
				},
			},
			IDs:   []int{6, 7, 8},
			First: true,
		},
	}
	compareBuckets(t, expected, actual)
}

func compareBuckets(t *testing.T, expected, actual []*Bucket) {
	if len(expected) != len(actual) {
		t.Fatalf("Different []Bucket length:\n- %v\n- %v", expected, actual)
	}
	for i := range expected {
		if !reflect.DeepEqual(expected[i], actual[i]) {
			t.Fatalf("Different Bucket:\n- %#v\n- %#v", expected[i], actual[i])
		}
	}
}

func Test_isSubset(t *testing.T) {
	type args struct {
		first  *callstack
		second *callstack
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test if first is subset of second.",
			args: args{
				first: &callstack{"a", "aa", "aaa"},
				second: &callstack{"a", "aa", "aaa", "aaaa"},
			},
			want: true,
		},
		{
			name: "Test equal.",
			args: args{
				first: &callstack{"a", "aa", "aaa"},
				second: &callstack{"a", "aa", "aaa"},
			},
			want: true,
		},
		{
			name: "Test if not a subset.",
			args: args{
				first: &callstack{"a"},
				second: &callstack{"aa", "aaa"},
			},
			want: false,
		},


	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSubset(tt.args.first, tt.args.second); got != tt.want {
				t.Errorf("isSubset() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_checkSubset(t *testing.T) {
	type args struct {
		fullStacks []*callstack
		curstack   callstack
	}
	tests := []struct {
		name string
		args args
		want []*callstack
	}{
		{
			name: "Test if same stack",
			args: args{
				fullStacks: []*callstack{&callstack{"a", "b"}, &callstack{"d", "f"}},
				curstack:callstack{"a", "b"},
			},
			want: []*callstack{&callstack{"a", "b"}, &callstack{"d", "f"}},
		},
		{
			name: "Test if incoming stack is already present in the fullstacks",
			args: args{
				fullStacks: []*callstack{&callstack{"a", "b"}, &callstack{"d", "f", "e"}},
				curstack:callstack{"d", "f"},
			},
			want: []*callstack{&callstack{"a", "b"}, &callstack{"d", "f", "e"}},
		},
		{
			name: "Test if incoming stack's subsets are present in the fullstacks",
			args: args{
				fullStacks: []*callstack{&callstack{"a", "b"}, &callstack{"d", "f"}},
				curstack:callstack{"a", "b", "c"},
			},
			want: []*callstack{&callstack{"d", "f"}, &callstack{"a", "b", "c"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkSubset(tt.args.fullStacks, tt.args.curstack)
			if len(got) != len(tt.want) {
				t.Errorf("Got: %v, Want: %v\n", got, tt.want)
			}
			for idx, result := range got {
				if !reflect.DeepEqual(*result, *tt.want[idx]) {
					t.Errorf("checkSubset() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestAggregateSubsets(t *testing.T) {
	type args struct {
		goroutines []*Goroutine
		allStacks  Callstacks
	}
	tests := []struct {
		name string
		args args
		want Callstacks
	}{
		{
			name: "Test empty set queue initialization.",
			args: args{
				goroutines: []*Goroutine{
					&Goroutine{
						Signature: Signature{
							State:     "",
							CreatedBy: Call{},
							SleepMin:  0,
							SleepMax:  0,
							Stack:     Stack{
								Calls:  []Call{
									{
										SrcPath:      "",
										LocalSrcPath: "",
										Line:         0,
										Func:         Func{
											Raw: "main.main",
										},
										Args:         Args{},
										IsStdlib:     false,
									},
									{
										SrcPath:      "",
										LocalSrcPath: "",
										Line:         0,
										Func:         Func{
											Raw: "init.init",
										},
										Args:         Args{},
										IsStdlib:     false,
									},
								},
								Elided: false,
							},
							Locked:    false,
						},
						ID:        0,
						First:     false,
					},
				},
				allStacks: nil,
			},
			want: Callstacks{
				&callstack{
					"main.main",
					"init.init",
				},
			},
		},
		{
			name: "Test multiple goroutines with different functions.",
			args: args{
				goroutines: []*Goroutine{
					&Goroutine{
						Signature: Signature{
							State:     "",
							CreatedBy: Call{},
							SleepMin:  0,
							SleepMax:  0,
							Stack:     Stack{
								Calls:  []Call{
									{
										SrcPath:      "",
										LocalSrcPath: "",
										Line:         0,
										Func:         Func{
											Raw: "main.main",
										},
										Args:         Args{},
										IsStdlib:     false,
									},
									{
										SrcPath:      "",
										LocalSrcPath: "",
										Line:         0,
										Func:         Func{
											Raw: "init.init",
										},
										Args:         Args{},
										IsStdlib:     false,
									},
								},
								Elided: false,
							},
							Locked:    false,
						},
						ID:        0,
						First:     false,
					},
					&Goroutine{
						Signature: Signature{
							State:     "",
							CreatedBy: Call{},
							SleepMin:  0,
							SleepMax:  0,
							Stack:     Stack{
								Calls:  []Call{
									{
										SrcPath:      "",
										LocalSrcPath: "",
										Line:         0,
										Func:         Func{
											Raw: "a.b",
										},
										Args:         Args{},
										IsStdlib:     false,
									},
									{
										SrcPath:      "",
										LocalSrcPath: "",
										Line:         0,
										Func:         Func{
											Raw: "c.d",
										},
										Args:         Args{},
										IsStdlib:     false,
									},
								},
								Elided: false,
							},
							Locked:    false,
						},
						ID:        0,
						First:     false,
					},
				},
				allStacks: nil,
			},
			want: Callstacks{
				&callstack{
					"main.main",
					"init.init",
				},
				&callstack{
					"a.b",
					"c.d",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AggregateSubsets(tt.args.goroutines, tt.args.allStacks); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AggregateSubsets() = %v, want %v", got, tt.want)
			}
		})
	}
}