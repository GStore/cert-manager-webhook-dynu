package dynuclient

import "net/http"

// DNSRecord ...
type DNSRecord struct {
	NodeName   string `json:"nodeName"`
	RecordType string `json:"recordType"`
	TextData   string `json:"textData"`
	TTL        string `json:"ttl"`
	DomainID   int    `json:"domainId,omitempty"`
	State      bool   `json:"state,omitempty"`
}

// DNSResponse ...
type DNSResponse struct {
	StatusCode int    `json:"statusCode"`
	ID         int    `json:"id"`
	DomainID   int    `json:"domainId"`
	DomainName string `json:"domainName"`
	NodeName   string `json:"nodeName"`
	Hostname   string `json:"hostname"`
	RecordType string `json:"recordType"`
	TTL        int16  `json:"ttl"`
	State      bool   `json:"state"`
	Content    string `json:"content"`
	UpdatedOn  string `json:"updatedOn"`
	TextData   string `json:"textData"`
}

// DynuClient ... options for DynuClient
type DynuClient struct {
	HTTPClient *http.Client
	HostName   string
	UserAgent  string
	APIKey     string
}

// DynuCreds - Details required to access API
type DynuCreds struct {
	APIKey string
}

// APIException ...
type APIException struct {
	StatusCode int
	Type       string
	Message    string
}

// Domain - The Root Domain
type Domain struct {
	StatusCode int          `json:"statusCode"`
	ID         int          `json:"id"`
	Hostname   string       `json:"hostname"`
	DomainName string       `json:"domainName"`
	Node       string       `json:"node"`
	Exception  APIException `json:"exception"`
}

// DNSRecords ...
type DNSRecords struct {
	StatusCode int           `json:"statusCode,omitempty"`
	DNSRecords []DNSResponse `json:"dnsRecords,omitempty"`
}
