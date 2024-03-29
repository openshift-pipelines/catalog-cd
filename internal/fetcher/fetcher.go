package fetcher

import (
	"fmt"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/openshift-pipelines/catalog-cd/internal/contract"
	"github.com/openshift-pipelines/catalog-cd/internal/fetcher/config"
)

// FetchContractsFromRepository fetches contracts from a repository.
func FetchContractsFromRepository(r config.Repository, client *api.RESTClient) (map[string]*contract.Contract, error) {
	m := map[string]*contract.Contract{}

	if !strings.HasPrefix(r.URL, "https://github.com") {
		return m, fmt.Errorf("non-github repository not supported: %s", r.URL)
	}
	repo := strings.TrimPrefix(r.URL, "https://github.com/")
	versions, err := fetchVersions(repo, client)
	if err != nil {
		return m, fmt.Errorf("failed to fetch versions from %s: %w", r.URL, err)
	}
	for _, v := range versions {
		if v.PreRelease || v.Draft {
			// Ignore drafts or pre-releases
			continue
		}
		var contractAsset Asset
		contractFound := false
		for _, a := range v.Assets {
			// catalog.yml is there for backward-compatibility
			if a.Name == r.CatalogName || a.Name == "catalog.yml" {
				contractFound = true
				contractAsset = a
				break
			}
		}
		if !contractFound {
			// FIXME(vdemeester) should we ignore or error out ?
			continue
		}
		// Load contract from asset
		contract, err := contract.NewContractFromURL(contractAsset.DownloadURL)
		if err != nil {
			return m, fmt.Errorf("failed to load asset %s from %s: %w", contractAsset.Name, v.TagName, err)
		}
		m[v.TagName] = contract
	}
	return m, nil
}

func fetchVersions(github string, client *api.RESTClient) ([]Version, error) {
	versions := []Version{}
	err := client.Get(fmt.Sprintf("repos/%s/releases", github), &versions)
	if err != nil {
		return nil, err
	}
	return versions, nil
}

type Version struct {
	Name       string
	TagName    string `json:"tag_name"`
	ID         int    `json:"id"`
	Draft      bool
	PreRelease bool
	Assets     []Asset
	URL        string `json:"url"`
	TarballURL string `json:"tarball_url"`
}

type Asset struct {
	ID          int    `json:"id"`
	URL         string `json:"url"`
	Name        string
	Label       string
	ContentType string `json:"content_type"`
	State       string
	DownloadURL string `json:"browser_download_url"`
}
