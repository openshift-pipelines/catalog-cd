package contract

import (
	"path"
	"testing"

	o "github.com/onsi/gomega"
)

func TestNewContractEmpty(t *testing.T) {
	t.Skip("Skipping, need to be rewritten")
	g := o.NewWithT(t)

	testDir := "../../testdata/resources"

	c := NewContractEmpty()

	t.Run("AddResourceFile", func(_ *testing.T) {
		taskFile := path.Join(testDir, "task.yaml")
		version := "0.0.1"

		err := c.AddResourceFile(taskFile, version)
		g.Expect(err).ToNot(o.HaveOccurred())
		g.Expect(c.Catalog.Resources).ToNot(o.BeNil())
		g.Expect(c.Catalog.Resources.Tasks).To(o.HaveLen(1))

		resource := c.Catalog.Resources.Tasks[0]
		g.Expect(resource).ToNot(o.BeNil())
		g.Expect(resource.Name).To(o.Equal("task"))
		g.Expect(resource.Version).To(o.Equal(version))
		g.Expect(resource.Filename).To(o.Equal(taskFile))
		g.Expect(resource.Checksum).NotTo(o.BeEmpty())
	})

	t.Run("Print", func(_ *testing.T) {
		contractBytes, err := c.Print()
		g.Expect(err).ToNot(o.HaveOccurred())
		g.Expect(contractBytes).NotTo(o.BeEmpty())
	})

	t.Run("Output", func(_ *testing.T) {
		contractFile := path.Join(testDir, Filename)
		err := c.SaveAs(contractFile)
		g.Expect(err).ToNot(o.HaveOccurred())
		g.Expect(contractFile).To(o.BeAnExistingFile())
	})
}

func TestNewContractFromFile(t *testing.T) {
	g := o.NewWithT(t)

	t.Run("Filename", func(_ *testing.T) {
		c, err := NewContractFromFile("../../testdata/resources/.catalog.yaml")
		g.Expect(err).ToNot(o.HaveOccurred())
		g.Expect(c.Catalog.Resources).ToNot(o.BeNil())
	})

	t.Run("Directory", func(_ *testing.T) {
		c, err := NewContractFromFile("../../testdata/resources")
		g.Expect(err).ToNot(o.HaveOccurred())
		g.Expect(c.Catalog.Resources).ToNot(o.BeNil())
	})
}
