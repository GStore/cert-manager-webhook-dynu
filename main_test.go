package main

import (
	"io/ioutil"
	"log"
	"os"
	"testing"

	//"gitlab.com/smueller18/cert-manager-webhook-inwx/test"
	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"

	"github.com/jetstack/cert-manager/test/acme/dns"
)

var (
	zone               = os.Getenv("TEST_ZONE_NAME")
	kubeBuilderBinPath = "./_out/kubebuilder/bin"
	fqdn               string
	nodeName           = "cert-manager-dynu-webhook."
)

func TestRunsSuite(t *testing.T) {
	d, err := ioutil.ReadFile("testdata/config.json")
	if err != nil {
		log.Fatal(err)
	}

	fixture := dns.NewFixture(&dynuProviderSolver{},
		dns.SetResolvedZone(zone),
		dns.SetAllowAmbientCredentials(false),
		dns.SetBinariesPath(kubeBuilderBinPath),
		dns.SetStrict(true),
		dns.SetConfig(&extapi.JSON{
			Raw: d,
		}),
	)

	fixture.RunConformance(t)
}
func TestRunSuiteWithSecret(t *testing.T) {
	d, err := ioutil.ReadFile("testdata/config.secret.json")
	if err != nil {
		log.Fatal(err)
	}

	fixture := dns.NewFixture(&dynuProviderSolver{},
		dns.SetResolvedZone(zone),
		dns.SetAllowAmbientCredentials(false),
		dns.SetBinariesPath(kubeBuilderBinPath),
		dns.SetStrict(true),
		dns.SetManifestPath("testdata/secret-dynu-credentials.yaml"),
		dns.SetConfig(&extapi.JSON{
			Raw: d,
		}),
	)

	fixture.RunConformance(t)
}
