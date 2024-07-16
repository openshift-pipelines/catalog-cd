package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/openshift-pipelines/catalog-cd/internal/contract"
	"gopkg.in/yaml.v2"
)

// GithubTags represents the response JSON when fetching various tags/releases of a github repo
type GithubTags struct {
	TagName string         `json:"tag_name"`
	URL     string         `json:"url"`
	Assets  []GithubAssets `json:"assets"`
}

// GithubAssets represents the Assets field of the GithubTags
type GithubAssets struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	URL                string `json:"url"`
}

// ResourceInfo represents the value field of our map whose key would be the Name of the resource
type ResourceInfo struct {
	Source  string
	Version string
}

// verifyNameConflicts function handles the logic to fetch the various releases from the repos & check whether they have any conflicts in their Name,
// either from the same repo. i.e. same source or from different repo. i.e. different source
func verifyNameConflicts(m GitHubMatrixObject) error {
	tempDirPath, err := os.MkdirTemp("", "example")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDirPath)

	kindSourceMap := make(map[string]map[string][]ResourceInfo)
	kindSourceMap["tasks"] = make(map[string][]ResourceInfo)
	kindSourceMap["pipelines"] = make(map[string][]ResourceInfo)

	for _, githubObj := range m.Include {
		var orgURL string
		partsOfURL := strings.Split(githubObj.URL, "github.com/")

		if len(partsOfURL) > 1 {
			orgURL = partsOfURL[1]
		} else {
			return fmt.Errorf("incorrect url i.e. url doesn't contain github.com")
		}

		tagsURL := "https://api.github.com/repos/" + orgURL + "/releases"
		tagsResp, err := MakeGetRequest(tagsURL)
		if err != nil {
			return err
		}
		defer tagsResp.Body.Close()

		tagsBody, err := io.ReadAll(tagsResp.Body)
		if err != nil {
			return err
		}

		var releases []GithubTags
		if err = json.Unmarshal(tagsBody, &releases); err != nil {
			return err
		}

		for _, release := range releases {
			if !strings.Contains(githubObj.IgnoreVersions, release.TagName) {
				for _, asset := range release.Assets {
					if asset.Name == contract.Filename {
						filePath := tempDirPath + "/" + githubObj.Type + "-" + release.TagName + "-" + githubObj.Name + ".yaml"
						err := downloadAndParseFile(asset.BrowserDownloadURL, filePath, githubObj.Type, kindSourceMap)
						if err != nil {
							return err
						}
					}
				}
			}
		}
	}

	return nil
}

// downloadAndParseFile function calls the downloadFile & parseFile functions which downloads the various catalog.yaml files of each release & then parses them
func downloadAndParseFile(url, filepath, kind string, unique map[string]map[string][]ResourceInfo) error {
	if err := downloadFile(url, filepath); err != nil {
		return err
	}

	if err := parseFile(filepath, kind, unique, url); err != nil {
		return err
	}

	return nil
}

// downloadFile function downloads the file mentioned in the url & stores it in the mentioned filepath
func downloadFile(url, filepath string) error {
	file, err := os.OpenFile(filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	res, err := MakeGetRequest(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned non-200 status code: %v", res.Status)
	}

	if _, err = io.Copy(file, res.Body); err != nil {
		return err
	}

	return nil
}

// parseResources function parses the resources & checks for uniqueness
func parseResources(resources []*contract.TektonResource, unique map[string][]ResourceInfo, source string, kind string) error {
	for _, res := range resources {
		name := res.Name
		version := res.Version
		_, exists := unique[name]

		if exists {
			currResources := unique[name]
			//Checks whether the sources are different
			if currResources[0].Source != source {
				return fmt.Errorf("2 resources of kind '%s', have same name '%s', from different sources, \nsource1: %s\nsource2: %s", kind, name, currResources[0].Source, source)
			}
			//Checks whether the versions are same or not, if its from same source
			for _, currResource := range currResources {
				if currResource.Version == version {
					return fmt.Errorf("2 resources of kind '%s', have same name '%s', from same source '%s'", kind, name, source)
				}
			}
			//If none of the above then the resource is still unique so append
			unique[name] = append(unique[name], ResourceInfo{Version: version, Source: source})
		} else {
			//If no name conflict then resource is unique so append
			unique[name] = []ResourceInfo{{Version: version, Source: source}}
		}
	}
	return nil
}

// parseFile function parses the file present in the path & then calls the parseResources function to check for uniqueness
func parseFile(path, kind string, unique map[string]map[string][]ResourceInfo, source string) error {
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var catalog contract.Contract

	err = yaml.Unmarshal(yamlFile, &catalog)
	if err != nil {
		return err
	}

	var resources []*contract.TektonResource

	switch kind {
	case "tasks":
		resources = catalog.Catalog.Resources.Tasks
	case "pipelines":
		resources = catalog.Catalog.Resources.Pipelines
	default:
		return fmt.Errorf("kind is neither tasks nor pipelines")
	}

	err = parseResources(resources, unique[kind], source, kind)
	if err != nil {
		return err
	}

	return nil
}
