package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	// cmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	"github.com/jetstack/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/jetstack/cert-manager/pkg/acme/webhook/cmd"
	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"

	// "github.com/jetstack/cert-manager/pkg/issuer/acme/dns/util"
	"github.com/gstore/cert-manager-webhook-dynu/dynuclient"
	certmgrv1 "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const acmeNode = "acme"

var (
	dnsRecordID int
)

// GroupName ...
var GroupName = os.Getenv("GROUP_NAME")

func main() {
	if GroupName == "" {
		panic("GROUP_NAME must be specified")
	}

	// This will register our custom DNS provider with the webhook serving
	// library, making it available as an API under the provided GroupName.
	// You can register multiple DNS provider implementations with a single
	// webhook, where the Name() method will be used to disambiguate between
	// the different implementations.
	cmd.RunWebhookServer(GroupName,
		&dynuProviderSolver{},
	)
}

// dynuProviderSolver implements the provider-specific logic needed to
// 'present' an ACME challenge TXT record for your own DNS provider.
// To do so, it must implement the `github.com/jetstack/cert-manager/pkg/acme/webhook.Solver`
// interface.
type dynuProviderSolver struct {
	// If a Kubernetes 'clientset' is needed, you must:
	// 1. uncomment the additional `client` field in this structure below
	// 2. uncomment the "k8s.io/client-go/kubernetes" import at the top of the file
	// 3. uncomment the relevant code in the Initialize method below
	// 4. ensure your webhook's service account has the required RBAC role
	//    assigned to it for interacting with the Kubernetes APIs you need.
	client     kubernetes.Clientset
	httpClient *http.Client
}

// dynuProviderConfig is a structure that is used to decode into when
// solving a DNS01 challenge.
// This information is provided by cert-manager, and may be a reference to
// additional configuration that's needed to solve the challenge for this
// particular certificate or issuer.
// This typically includes references to Secret resources containing DNS
// provider credentials, in cases where a 'multi-tenant' DNS solver is being
// created.
// If you do *not* require per-issuer or per-certificate configuration to be
// provided to your webhook, you can skip decoding altogether in favour of
// using CLI flags or similar to provide configuration.
// You should not include sensitive information here. If credentials need to
// be used by your provider here, you should reference a Kubernetes Secret
// resource and fetch these credentials using a Kubernetes clientset.
type dynuProviderConfig struct {
	// Change the two fields below according to the format of the configuration
	// to be decoded.
	// These fields will be set by users in the
	// `issuer.spec.acme.dns01.providers.webhook.config` field.

	//Email           string `json:"email"`
	// APIKeySecretRef v1alpha1.SecretKeySelector `json:"apiKeySecretRef"`
	APIKey             string                      `json:"apiKey"`
	TTL                int                         `json:"ttl"`
	APIKeySecretKeyRef certmgrv1.SecretKeySelector `json:"apikeySecretKeyRef"`
}

// Name is used as the name for this DNS solver when referencing it on the ACME
// Issuer resource.
// This should be unique **within the group name**, i.e. you can have two
// solvers configured with the same Name() **so long as they do not co-exist
// within a single webhook deployment**.
// For example, `cloudflare` may be used as the name of a solver.
func (c *dynuProviderSolver) Name() string {
	return "dynu"
}

// Present is responsible for actually presenting the DNS record with the
// DNS provider.
// This method should tolerate being called multiple times with the same value.
// cert-manager itself will later perform a self check to ensure that the
// solver has correctly configured the DNS provider.
func (c *dynuProviderSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	dynu, cfg, err := c.NewDynuClient(ch)
	if err != nil {
		klog.Error(fmt.Sprintf("\n\nUnable to create dynu client\nErr: %v\n", err))
		return err
	}
	nodeName := strings.TrimSuffix(strings.Replace(ch.ResolvedFQDN, ch.ResolvedZone, "", -1), ".")
	klog.Info("\n\nPresent DNSName ", ch.ResolvedFQDN, "\nzone ", ch.ResolvedZone, "\nnodeName: ", nodeName, "\nvalue ", ch.Key)

	rec := dynuclient.DNSRecord{
		NodeName:   nodeName,
		RecordType: "TXT",
		TextData:   ch.Key,
		TTL:        strconv.Itoa(cfg.TTL),
		State:      true,
	}

	dnsRecordID, err = dynu.CreateDNSRecord(rec)
	if err != nil {
		klog.Error(fmt.Sprintf("\n\nFailed to create DNS record\nErr: %v\n", err))
		return err
	}
	klog.Flush()
	return nil
}

// CleanUp should delete the relevant TXT record from the DNS provider console.
// If multiple TXT records exist with the same record name (e.g.
// _acme-challenge.example.com) then **only** the record with the same `key`
// value provided on the ChallengeRequest should be cleaned up.
// This is in order to facilitate multiple DNS validations for the same domain
// concurrently.
func (c *dynuProviderSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	dynu, _, err := c.NewDynuClient(ch)
	if err != nil {
		klog.Error(fmt.Sprintf("\n\nUnable to create dynu client\nErr: %v\n", err))
		return err
	}
	nodeName := strings.TrimSuffix(strings.Replace(ch.ResolvedFQDN, ch.ResolvedZone, "", -1), ".")
	klog.Info("\n\nCleanup DNSName ", ch.ResolvedFQDN, "\nzone ", ch.ResolvedZone, "\nnodeName: ", nodeName, "\nvalue ", ch.Key)

	err = dynu.RemoveDNSRecord(nodeName, ch.Key)
	if err != nil {
		klog.Error(fmt.Sprintf("\n\nFailed to remove DNS record\nErr: %v\n", err))
		return err
	}
	klog.Flush()
	return nil
}

// Initialize will be called when the webhook first starts.
// This method can be used to instantiate the webhook, i.e. initialising
// connections or warming up caches.
// Typically, the kubeClientConfig parameter is used to build a Kubernetes
// client that can be used to fetch resources from the Kubernetes API, e.g.
// Secret resources containing credentials used to authenticate with DNS
// provider accounts.
// The stopCh can be used to handle early termination of the webhook, in cases
// where a SIGTERM or similar signal is sent to the webhook process.
func (c *dynuProviderSolver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	klog.Info("Group Name:", GroupName)
	///// UNCOMMENT THE BELOW CODE TO MAKE A KUBERNETES CLIENTSET AVAILABLE TO
	///// YOUR CUSTOM DNS PROVIDER
	cl, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		klog.Error(fmt.Sprintf("\n\nFailed to Initialize\nErr: %v\n", err))
		return err
	}
	c.client = *cl
	klog.Flush()
	///// END OF CODE TO MAKE KUBERNETES CLIENTSET AVAILABLEuri := cfg.BaseURL + cfg.DomainId + "/" + cfg.EndPoint
	return nil
}

// loadConfig is a small helper function that decodes JSON configuration into
// the typed config struct.
func loadConfig(cfgJSON *extapi.JSON) (dynuProviderConfig, error) {
	cfg := dynuProviderConfig{}
	// handle the 'base case' where no configuration has been provided
	if cfgJSON == nil {
		return cfg, nil
	}
	if err := json.Unmarshal(cfgJSON.Raw, &cfg); err != nil {
		klog.Error(fmt.Sprintf("\nInit...Err: %v\n", err))
		return cfg, fmt.Errorf("error decoding solver config: %v", err)
	}

	return cfg, nil
}

func (c *dynuProviderSolver) getCredentials(config *dynuProviderConfig, ns string) (*dynuclient.DynuCreds, error) {

	creds := dynuclient.DynuCreds{}

	if config.APIKey != "" {
		creds.APIKey = config.APIKey
	} else {
		secret, err := c.client.CoreV1().Secrets(ns).Get(context.Background(), config.APIKeySecretKeyRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to load secret %q", ns+"/"+config.APIKeySecretKeyRef.Name)
		}
		if apikey, ok := secret.Data[config.APIKeySecretKeyRef.Key]; ok {
			creds.APIKey = strings.TrimSpace(string(apikey))
		} else {
			return nil, fmt.Errorf("no key %q in secret %q", config.APIKeySecretKeyRef, ns+"/"+config.APIKeySecretKeyRef.Name)
		}
	}

	// if config.HostName != "" {
	// 	creds.HostName = config.HostName
	// } else {
	// 	secret, err := c.client.CoreV1().Secrets(ns).Get(context.Background(), config.HostNameKeyRef.Name, metav1.GetOptions{})
	// 	if err != nil {
	// 		return nil, fmt.Errorf("failed to load secret %q", ns+"/"+config.HostNameKeyRef.Name)
	// 	}
	// 	if hostname, ok := secret.Data[config.HostNameKeyRef.Key]; ok {
	// 		creds.HostName = strings.TrimSpace(string(hostname))
	// 	} else {
	// 		return nil, fmt.Errorf("no key %q in secret %q", config.HostNameKeyRef, ns+"/"+config.HostNameKeyRef.Name)
	// 	}
	// }

	return &creds, nil
}

// NewDynuClient - Create a new DynuClient
func (c *dynuProviderSolver) NewDynuClient(ch *v1alpha1.ChallengeRequest) (*dynuclient.DynuClient, *dynuProviderConfig, error) {
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return nil, &cfg, err
	}

	creds, err := c.getCredentials(&cfg, ch.ResourceNamespace)
	if err != nil {
		return nil, &cfg, fmt.Errorf("error getting credentials: %v", err)
	}

	zone := strings.TrimSuffix(ch.ResolvedZone, ".")
	client := &dynuclient.DynuClient{HostName: zone, APIKey: creds.APIKey, HTTPClient: c.httpClient}

	return client, &cfg, nil
}
