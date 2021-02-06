package dynuclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"runtime"
	"strings"
	"time"

	"k8s.io/klog"
)

const dynuAPI string = "https://api.dynu.com/v2"

// Conformance tests fail
const dynuRateLimit int = 5

var httpClient *http.Client

// CreateDNSRecord ... Create a DNS Record and return it's ID
//   POST https://api.dynu.com/v2/dns/{DNSID}/record
func (c *DynuClient) CreateDNSRecord(record DNSRecord) (int, error) {
	klog.Info("\n\nCreating DNS Record for: ", record.NodeName, " hostname: ", c.HostName, " textdata: ", record.TextData, "\n\n")
	domainID, err := c.GetDomainID()
	if err != nil {
		klog.Error(fmt.Sprintf("\n\nCreateDNSRecord...Err: %v\n", err))
		return -1, err
	}
	dnsRecord, err := c.GetDNSRecord(domainID, record.NodeName, record.TextData)
	if err == nil {
		return dnsRecord.ID, nil
	}
	dnsURL := fmt.Sprintf("%s/dns/%d/record", dynuAPI, domainID)
	body, err := json.Marshal(record)
	if err != nil {
		klog.Error(fmt.Sprintf("\n\nCreateDNSRecord...Err: %v\n", err))
		return -1, err
	}

	var resp *http.Response

	resp, err = c.makeRequest(dnsURL, "POST", bytes.NewReader(body))
	if err != nil {
		klog.Error(fmt.Sprintf("\n\nCreateDNSRecord...Err: %v\n", err))
		return -1, err
	}

	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			klog.Error(fmt.Sprintf("\n\nCreateDNSRecord...Err: %v\n", err))
			return -1, err
		}
		c.logResponseBody(bodyBytes)
		var dnsBody DNSResponse
		err = json.Unmarshal(bodyBytes, &dnsBody)
		if err != nil {
			klog.Error(fmt.Sprintf("\n\nCreateDNSRecord...Err: %v\n", err))
			return -1, err
		}
		klog.Info("\n\nDNS Record created for: ", record.NodeName, " hostname: ", c.HostName, "\n\n")
		return dnsBody.ID, nil
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		klog.Error(fmt.Sprintf("\n\nCreateDNSRecord...Err: %v\n", err))
		return -1, err
	}

	c.logResponseBody(bodyBytes)

	return -1, fmt.Errorf("%s received for %s", resp.Status, dnsURL)
}

// RemoveDNSRecord ... Removes a DNS record based on dnsRecordID
//   DELETE https://api.dynu.com/v2/dns/{DNSID}/record/{DNSRecordID}
func (c *DynuClient) RemoveDNSRecord(nodeName, textData string) error {
	klog.Info("\n\nRemoving DNS Record for: ", nodeName, " hostname: ", c.HostName, " with text: ", textData, "\n\n")
	var err error
	domainID, err := c.GetDomainID()
	if err != nil {
		return err
	}
	klog.Info(fmt.Sprintf("\n\nRemoveDNSRecord: \nDomainId: %d\n\n", domainID))
	dnsRecord, err := c.GetDNSRecord(domainID, nodeName, textData)
	if err != nil {
		if strings.Contains(err.Error(), "Unable to find DNS Records") {
			klog.Info(fmt.Sprintf("Couldn't find record: %v", err))
			return nil
		}
		return err
	}

	dnsURL := fmt.Sprintf("%s/dns/%d/record/%d", dynuAPI, domainID, dnsRecord.ID)
	var resp *http.Response

	resp, err = c.makeRequest(dnsURL, "DELETE", nil)
	if err != nil {
		return nil
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(resp.Status)
	}
	klog.Info("\n\nDNS Record removed for: ", nodeName, " hostname: ", c.HostName, " with text: ", textData, "\n\n")
	return nil
}

func (c *DynuClient) makeRequest(URL string, method string, body io.Reader) (*http.Response, error) {
	time.Sleep(time.Duration(dynuRateLimit) * time.Second)
	req, err := http.NewRequest(method, URL, body)
	if err != nil {
		return nil, err
	}

	klog.Info("\n\nAPI Key: ", c.APIKey, "\n\n")
	req.Header["accept"] = []string{"application/json"}
	req.Header["User-Agent"] = []string{c.UserAgent}
	req.Header["Content-Type"] = []string{"application/json"}
	req.Header["API-Key"] = []string{c.APIKey}

	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{}
	}

	c.HTTPClient.Timeout = 30 * time.Second

	return c.HTTPClient.Do(req)
}

// DecodeBytes ..
func (c *DynuClient) decodeBytes(input []byte) (string, error) {

	buf := new(strings.Builder)
	_, err := io.Copy(buf, bytes.NewBuffer(input))
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// GetDomainID ...
func (c *DynuClient) GetDomainID() (int, error) {
	dnsURL := fmt.Sprintf("%s/dns/getroot/%s", dynuAPI, c.HostName)

	klog.Info("\ndnsURL: \n", dnsURL, "\n\n")
	resp, err := c.makeRequest(dnsURL, "GET", nil)
	if err != nil {
		return -1, err
	}

	defer resp.Body.Close()

	var domain Domain

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return -1, err
		}
		err = json.Unmarshal(bodyBytes, &domain)
		if err != nil {
			return -1, err
		}
		return domain.ID, nil
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	respbody, err := c.decodeBytes(bodyBytes)
	if err == nil {
		klog.Info("\nrespbody: \n", respbody, "\n\n")
	}

	return -1, fmt.Errorf("Unable to find Domain ID \nError Type:%s\nError: %s", domain.Exception.Type, domain.Exception.Message)
}

// GetDNSRecord ...
func (c *DynuClient) GetDNSRecord(domainID int, nodeName, textData string) (*DNSResponse, error) {
	var dnsRecords DNSRecords
	dnsURL := fmt.Sprintf("%s/dns/%d/record", dynuAPI, domainID)
	var resp *http.Response

	resp, err := c.makeRequest(dnsURL, "GET", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(bodyBytes, &dnsRecords)
		if err != nil {
			return nil, err
		}
		for _, rec := range dnsRecords.DNSRecords {
			if rec.NodeName == nodeName && rec.TextData == textData {
				return &rec, nil
			}
		}
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	c.logResponseBody(bodyBytes)
	return nil, fmt.Errorf("Unable to find DNS Records for Domain ID: %d", domainID)
}

func (c *DynuClient) logResponseBody(body []byte) {
	_, file, no, ok := runtime.Caller(1)
	if ok {
		klog.Info(fmt.Sprintf("\n\ncalled from %s#%d\n\n", file, no))
	}
	respbody, err := c.decodeBytes(body)
	if err != nil {
		klog.Error(fmt.Sprintf("Couldn't decode resonse body: %v", err))
	}
	klog.Info(fmt.Sprintf("\nResponseBody: %v\n", respbody))
}
