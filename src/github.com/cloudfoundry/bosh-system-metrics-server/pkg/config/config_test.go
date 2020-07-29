package config_test

import (
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/cloudfoundry/bosh-system-metrics-server/pkg/config"
	. "github.com/onsi/gomega"
)

func TestConfigRead(t *testing.T) {
	RegisterTestingT(t)

	configFilePath := writeConfigFile(configContents)
	defer os.Remove(configFilePath)

	expected := config.Config{UaaClientPassword: "value1"}

	c, err := config.Read(configFilePath)
	Expect(err).ToNot(HaveOccurred())
	Expect(c).To(Equal(expected))
}

func writeConfigFile(config string) string {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		log.Fatal(err)
	}

	f, err := ioutil.TempFile(dir, "test-config.yml")
	if err != nil {
		log.Fatal(err)
	}

	_, err = f.Write([]byte(config))
	if err != nil {
		log.Fatal(err)
	}

	return f.Name()
}

const (
	configContents = `
uaa-client-password: value1
`
)
