package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

type GithubTags struct {
	TagName string         `json:"tag_name"`
	URL     string         `json:"url"`
	Assets  []GithubAssets `json:"assets"`
}

type GithubAssets struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	URL                string `json:"url"`
}

type ResourcesConfig struct {
	TasksConfig     []ResourceConfig `yaml:"tasks"`
	PipelinesConfig []ResourceConfig `yaml:"pipelines"`
}

type ResourceConfig struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

type CatalogConfig struct {
	Resources ResourcesConfig `yaml:"resources"`
}

type Catalog struct {
	Version string        `yaml:"version"`
	Catalog CatalogConfig `yaml:"catalog"`
}

type ResourceInfo struct {
	Source  string
	Version string
}

func testNameConflicts(m GitHubMatrixObject) error {
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
		err = json.Unmarshal(tagsBody, &releases)
		if err != nil {
			return err
		}

		for _, release := range releases {
			if !strings.Contains(githubObj.IgnoreVersions, release.TagName) {
				for _, asset := range release.Assets {
					if asset.Name == "catalog.yaml" {
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

func downloadAndParseFile(url, filepath, kind string, unique map[string]map[string][]ResourceInfo) error {
	err := downloadFile(url, filepath)
	if err != nil {
		return err
	}

	err = parseFile(filepath, kind, unique, url)
	if err != nil {
		return err
	}

	return nil
}

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

	_, err = io.Copy(file, res.Body)
	if err != nil {
		return err
	}

	return nil
}

func parseResources(resources []ResourceConfig, unique map[string][]ResourceInfo, source string) error {
	for _, res := range resources {
		_, exists := unique[res.Name]

		if exists {
			currResources := unique[res.Name]
			if currResources[0].Source != source {
				return fmt.Errorf("different source, same name, \nsource1: %s\nsource2: %s", currResources[0].Source, source)
			}
			for _, currResource := range currResources {
				if currResource.Version == res.Version {
					return fmt.Errorf("2 resources have same name from same source, %s", source)
				}
			}
			unique[res.Name] = append(unique[res.Name], ResourceInfo{Version: res.Version, Source: source})
		} else {
			unique[res.Name] = []ResourceInfo{{Version: res.Version, Source: source}}
		}
	}
	return nil
}

func parseFile(path, kind string, unique map[string]map[string][]ResourceInfo, source string) error {
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var catalog Catalog

	err = yaml.Unmarshal(yamlFile, &catalog)
	if err != nil {
		return err
	}

	var resources []ResourceConfig

	switch kind {
	case "tasks":
		resources = catalog.Catalog.Resources.TasksConfig
	case "pipelines":
		resources = catalog.Catalog.Resources.PipelinesConfig
	default:
		return fmt.Errorf("kind is neither tasks nor pipelines")
	}

	err = parseResources(resources, unique[kind], source)
	if err != nil {
		return err
	}

	return nil
}
