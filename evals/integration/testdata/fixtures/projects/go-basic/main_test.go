package main

import "testing"

func TestGreet(t *testing.T) {
	got := greet("world")
	want := "Hello, world!"
	if got != want {
		t.Errorf("greet() = %q, want %q", got, want)
	}
}
