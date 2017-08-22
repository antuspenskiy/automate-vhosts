package main

import (
	"os"
	"testing"

	"github.com/antuspenskiy/automate-vhosts/branch"
)

func TestArgs(t *testing.T) {
	expected := "66-chuck-norris"
	os.Args = []string{"-branch=66-chuck-norris"}

	actual := branch.PassArguments()

	if actual != expected {
		t.Errorf("Test failed, expected: '%s', got:  '%s'", expected, actual)
	}
}
