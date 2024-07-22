package cmd

import (
	"context"
	"path"
	"testing"

	gomega "github.com/onsi/gomega"
	"github.com/openshift-pipelines/catalog-cd/internal/config"
)

func TestScenarios(t *testing.T) {
	testDir := "../../testdata/resources/verify-name-conflicts"
	o := externalsOptions{config: "./externals.yaml"}
	args := []string{}
	cfg := config.NewConfig()
	g := gomega.NewWithT(t)

	t.Run("Pulling a single repo with no conflicts", func(_ *testing.T) {
		externalsFile := path.Join(testDir, "externals1.yaml")
		o.config = externalsFile
		err := runCatalogExternals(context.TODO(), cfg, args, o)

		g.Expect(err).ToNot(gomega.HaveOccurred())
	})

	t.Run("Pulling from multiple repos with conflicts", func(_ *testing.T) {
		externalsFile := path.Join(testDir, "externals2.yaml")
		o.config = externalsFile
		err := runCatalogExternals(context.TODO(), cfg, args, o)

		g.Expect(err).To(gomega.HaveOccurred())
		g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("different sources")))
	})

	t.Run("Pulling a single repo with conflicts", func(_ *testing.T) {
		externalsFile := path.Join(testDir, "externals3.yaml")
		o.config = externalsFile
		err := runCatalogExternals(context.TODO(), cfg, args, o)

		g.Expect(err).To(gomega.HaveOccurred())
		g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("same source")))
	})

	t.Run("Pulling from multiple repos with no conflicts", func(_ *testing.T) {
		externalsFile := path.Join(testDir, "externals4.yaml")
		o.config = externalsFile
		err := runCatalogExternals(context.TODO(), cfg, args, o)

		g.Expect(err).ToNot(gomega.HaveOccurred())
	})
}
