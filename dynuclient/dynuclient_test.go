package dynuclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"

	guntest "github.com/gstore/cert-manager-webhook-dynu/test"
	"github.com/stretchr/testify/assert"
)

var (
	hostname = os.Getenv("DYNU_HOST_NAME")
	apikey   = os.Getenv("DYNU_APIKEY")
	nodeName = "txt"
	txtData  = "123=="
)

func TestGetDomainID(t *testing.T) {
	hostName := "example.com"
	expectedMethod := "GET"
	expectedURL := fmt.Sprintf("/v2/dns/getroot/%s", hostName)
	testHandlerFunc := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, expectedURL, req.URL.String(), "Should call %s but called %s", expectedURL, req.URL.String())
		assert.Equal(t, expectedMethod, req.Method, "Should be %s but got %s", expectedMethod, req.Method)

		w.Write([]byte(`{"statusCode": 200,"id": 12345,"domainName": "example.com","hostname": "example.com","node": ""}`))
	})
	client := &guntest.Testclient{}
	httpClient, teardown := client.TestingHTTPClient(testHandlerFunc)
	defer teardown()

	dynu := DynuClient{HTTPClient: httpClient, HostName: hostName}
	domainID, err := dynu.GetDomainID()
	assert.Equal(t, 12345, domainID)
	assert.Nil(t, err, "error returned")
}

func TestRemoveDNSRecord(t *testing.T) {
	expectedMethod := "DELETE"
	hostname := "example.com"
	domainID := 98765
	expectedURL := fmt.Sprintf("/v2/dns/%d/record/12345", domainID)
	i := 0
	testHandlerFunc := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch i {
		case 0:
			w.Write([]byte(`{}`))
		case 1:
			w.Write([]byte(fmt.Sprintf(`{"statusCode": 200,"id": %d,"domainName": "example.com","hostname": "example.com","node": ""}`, domainID)))
		default:
			assert.Equal(t, expectedURL, req.URL.String(), "Should call %s but called %s", expectedURL, req.URL.String())
			assert.Equal(t, expectedMethod, req.Method, "Should be %s but got %s", expectedMethod, req.Method)
			w.Write([]byte("ok"))
		}
		// if i == 0 {
		// 	w.Write([]byte(fmt.Sprintf(`{"statusCode": 200,"id": %d,"domainName": "example.com","hostname": "example.com","node": ""}`, domainID)))
		// } else {
		// 	assert.Equal(t, expectedURL, req.URL.String(), "Should call %s but called %s", expectedURL, req.URL.String())
		// 	assert.Equal(t, expectedMethod, req.Method, "Should be %s but got %s", expectedMethod, req.Method)
		// 	w.Write([]byte("ok"))
		// }
		i++
	})
	client := &guntest.Testclient{}
	httpClient, teardown := client.TestingHTTPClient(testHandlerFunc)
	defer teardown()

	dynu := DynuClient{HTTPClient: httpClient, HostName: hostname}
	err := dynu.RemoveDNSRecord(nodeName, txtData)
	assert.Nil(t, err, "error returned")
}

func TestCreateDNSRecord(t *testing.T) {

	expectedMethod := "POST"
	expectedRecordID := 987654
	hostname := "example.com"
	domainID := 98765
	expectedURL := fmt.Sprintf("/v2/dns/%d/record", domainID)
	i := 0
	rec := DNSRecord{
		NodeName:   nodeName,
		RecordType: "TXT",
		TextData:   txtData,
		TTL:        "90",
	}
	dnsResp := DNSResponse{
		StatusCode: 200,
		ID:         expectedRecordID,
		DomainName: "domainName",
		NodeName:   nodeName,
		Hostname:   "hostName",
		RecordType: "TXT",
		TTL:        90,
		State:      true,
		Content:    "content",
		UpdatedOn:  "2020-10-29T23:00",
		TextData:   txtData,
	}
	dnsResponse, _ := json.Marshal(dnsResp)
	testHandlerFunc := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch i {
		case 0:
			w.Write([]byte(fmt.Sprintf(`{"statusCode": 200,"id": %d,"domainName": "example.com","hostname": "example.com","node": ""}`, domainID)))
		case 1:
			w.Write([]byte(`{}`))
		default:
			assert.Equal(t, expectedURL, req.URL.String(), "Should call %s but called %s", expectedURL, req.URL.String())
			assert.Equal(t, expectedMethod, req.Method, "Should be %s but got %s", expectedMethod, req.Method)
			w.Write([]byte(dnsResponse))
		}

		// if i == 0 {
		// 	w.Write([]byte(fmt.Sprintf(`{"statusCode": 200,"id": %d,"domainName": "example.com","hostname": "example.com","node": ""}`, domainID)))
		// } else {
		// 	assert.Equal(t, expectedURL, req.URL.String(), "Should call %s but called %s", expectedURL, req.URL.String())
		// 	assert.Equal(t, expectedMethod, req.Method, "Should be %s but got %s", expectedMethod, req.Method)
		// 	w.Write([]byte(dnsResponse))
		// }
		i++
	})

	client := &guntest.Testclient{}
	httpClient, teardown := client.TestingHTTPClient(testHandlerFunc)
	defer teardown()
	dynu := DynuClient{HTTPClient: httpClient, HostName: hostname}
	recordID, err := dynu.CreateDNSRecord(rec)
	if err != nil {
		fmt.Println("an error occured: ", err.Error())
		return
	}

	assert.Equal(t, expectedRecordID, recordID, "RecordID expected %d got %d", expectedRecordID, recordID)
}

func TestAddAndRemoveRecord(t *testing.T) {
	if hostname == "" || apikey == "" {
		t.Skip("This has been intentionally skipped as it runs a test against the Live API.")
	}

	d := &DynuClient{HostName: hostname, APIKey: apikey}
	domainID, err := d.GetDomainID()
	assert.Nil(t, err)
	assert.NotEqual(t, -1, domainID)
	fmt.Printf("DomainID: %d", domainID)

	rec := DNSRecord{
		NodeName:   nodeName,
		RecordType: "TXT",
		TextData:   txtData,
		TTL:        "300",
		State:      true,
	}

	dnsrecordid, err := d.CreateDNSRecord(rec)

	assert.NoError(t, err)
	assert.NotNil(t, dnsrecordid, "DNSRecordID", dnsrecordid)
	t.Logf("CREATED DNSRecordID: %d", dnsrecordid)
	err = d.RemoveDNSRecord(nodeName, txtData)
	assert.NoError(t, err)
	t.Logf("Removed DNSRecordID: %d", dnsrecordid)
}
