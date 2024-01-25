package cmd

import "github.com/openshift-pipelines/catalog-cd/internal/contract"

func LoadContractFromArgs(args []string) (*contract.Contract, error) {
	var location string
	if len(args) == 0 {
		location = "."
	} else {
		location = args[0]
	}
	return contract.NewContractFromFile(location)
}
