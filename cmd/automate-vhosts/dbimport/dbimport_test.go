package main

import (
	"os"
	"testing"

	"github.com/antuspenskiy/automate-vhosts/pkg/branch"
)

func TestArgs(t *testing.T) {
	expected := "1_test_branch"
	os.Args = []string{"1-test-branch"}

	actual := branch.PassArguments("1-test-branch")

	if actual != expected {
		t.Errorf("Test failed, expected: '%s', got:  '%s'", expected, actual)
	}
}
