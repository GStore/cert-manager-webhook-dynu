package main

import (
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/gstore/cert-manager-webhook-dynu/test"
	logf "github.com/jetstack/cert-manager/pkg/logs"
	"github.com/jetstack/cert-manager/test/acme/dns/server"
	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"

	"github.com/jetstack/cert-manager/test/acme/dns"
)

var (
	zone               string // should have . at the end
	subdomain          string // should have . at the end
	kubeBuilderBinPath = "./_out/kubebuilder/bin"
	fqdn               string
	nodeName           = "cert-manager-dynu-webhook."
)

func setZoneAndSubdomain() {
	if os.Getenv("TEST_ZONE_NAME") != "" {
		zone = os.Getenv("TEST_ZONE_NAME")
	}

	if os.Getenv("TEST_SUBDOMAIN") != "" {
		subdomain = os.Getenv("TEST_SUBDOMAIN")
	}
}

func TestRunsSuite(t *testing.T) {
	setZoneAndSubdomain()
	fqdn = nodeName + subdomain + zone
	log.Printf("\n\nfqdn: %v\n\n", fqdn)
	d, err := ioutil.ReadFile("testdata/config.json")
	if err != nil {
		log.Fatal(err)
	}

	ctx := logf.NewContext(nil, nil, t.Name())
	srv := &server.BasicServer{
		Handler: &test.DNSHandler{
			Log: logf.FromContext(ctx, "dnsBasicServer"),
			TxtRecords: map[string][][]string{
				fqdn: {
					{},
					{},
					{"123d=="},
					{"123d=="},
				},
			},
			Zones: []string{zone},
		},
	}

	fixture := dns.NewFixture(&dynuProviderSolver{},
		dns.SetResolvedZone(zone),
		dns.SetResolvedFQDN(fqdn),
		dns.SetDNSServer(srv.ListenAddr()),
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
	setZoneAndSubdomain()
	fqdn = nodeName + subdomain + zone
	d, err := ioutil.ReadFile("testdata/config.secret.json")
	if err != nil {
		log.Fatal(err)
	}

	fixture := dns.NewFixture(&dynuProviderSolver{},
		dns.SetResolvedZone(zone),
		dns.SetResolvedFQDN(fqdn),
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
