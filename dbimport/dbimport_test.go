package main

import (
	"os"
	"testing"

	"github.com/antuspenskiy/automate-vhosts/branch"
)

func TestArgs(t *testing.T) {
	expected := "1testbranch"
	os.Args = []string{"-branch=1-test-branch"}

	actual := branch.PassArguments()

	if actual != expected {
		t.Errorf("Test failed, expected: '%s', got:  '%s'", expected, actual)
	}
}
