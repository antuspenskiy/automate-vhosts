package db

import (
	"fmt"
	"log"
	"regexp"
)

// ParseBranchName parse branch name symbols and lenght
func ParseBranchName(name string) string {

	// Remove all Non-Alphanumeric Characters from a NameBranch
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		log.Fatalf("Error: %s", err)
	}

	branchString := reg.ReplaceAllString(name, "_")

	// User name (should be no longer than 32) for Percona Server
	if len(branchString) > 32 {
		branchCut := branchString[0:32]
		fmt.Printf("A string of %s becomes %s \n", name, branchCut)
		return branchCut
	}
	fmt.Printf("A string of %s becomes %s \n", name, branchString)
	return branchString
}
