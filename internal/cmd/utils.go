package cmd

import (
	"context"
	"net/http"

	"github.com/openshift-pipelines/catalog-cd/internal/contract"
)

func LoadContractFromArgs(args []string) (*contract.Contract, error) {
	var location string
	if len(args) == 0 {
		location = "."
	} else {
		location = args[0]
	}
	return contract.NewContractFromFile(location)
}

func MakeGetRequest(uri string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return res, nil
}
