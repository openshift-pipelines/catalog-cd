package cmd

import (
	"context"
	"path"
	"testing"

	gomega "github.com/onsi/gomega"
	"github.com/openshift-pipelines/catalog-cd/internal/config"
)

type TestConfig struct {
	Name                   string
	ExternalsFile          string
	ExpectError            bool
	ExpectedErrorSubstring string
}

func testCatalogExternals(t *testing.T, testConfig TestConfig) {
	t.Helper()

	o := externalsOptions{config: testConfig.ExternalsFile}
	args := []string{}
	cfg := config.NewConfig()
	g := gomega.NewWithT(t)

	err := runCatalogExternals(context.TODO(), cfg, args, o)

	if testConfig.ExpectError {
		g.Expect(err).To(gomega.HaveOccurred())
		g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring(testConfig.ExpectedErrorSubstring)))
	} else {
		g.Expect(err).ToNot(gomega.HaveOccurred())
	}
}

func TestScenarios(t *testing.T) {
	testDir := "../../testdata/resources/externals"
	tests := []TestConfig{
		{
			Name:          "Pulling a single repo with no conflicts",
			ExpectError:   false,
			ExternalsFile: path.Join(testDir, "externals1.yaml"),
		},
		{
			Name:                   "Pulling from multiple repos with conflicts",
			ExternalsFile:          path.Join(testDir, "externals2.yaml"),
			ExpectError:            true,
			ExpectedErrorSubstring: "different sources",
		},
		{
			Name:                   "Pulling a single repo with conflicts",
			ExternalsFile:          path.Join(testDir, "externals3.yaml"),
			ExpectError:            true,
			ExpectedErrorSubstring: "same source",
		},
		{
			Name:          "Pulling from multiple repos with no conflicts",
			ExternalsFile: path.Join(testDir, "externals4.yaml"),
			ExpectError:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			testCatalogExternals(t, tc)
		})
	}
}
