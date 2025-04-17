package httpclient

import (
	"io"
	"net"
	"net/http"
	"time"
)

type HttpClient struct {
	client *http.Client
}

func New() *HttpClient {
	return &HttpClient{
		client: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:          100,              // Max idle connections across all hosts
				MaxIdleConnsPerHost:   10,               // Max idle connections per host
				IdleConnTimeout:       90 * time.Second, // Keep idle connections for 90s
				TLSHandshakeTimeout:   10 * time.Second, // Timeout for TLS handshake
				ExpectContinueTimeout: 1 * time.Second,  // Wait time for 100-Continue responses
				DialContext: (&net.Dialer{
					Timeout:   5 * time.Second,  // Connection timeout
					KeepAlive: 30 * time.Second, // TCP keep-alive time
				}).DialContext,
			},
			Timeout: 15 * time.Second, // Overall request timeout
		},
	}
}

func NewWithConfig(config *http.Client) *HttpClient {
	return &HttpClient{
		client: config,
	}
}

func (h *HttpClient) ReadResponse(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (h *HttpClient) Get(url string, header http.Header) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header = header
	return h.client.Do(req)
}

func (h *HttpClient) Post(url string, header http.Header, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	if header != nil {
		req.Header = header
	}
	return h.client.Do(req)
}

func (h *HttpClient) Put(url string, header http.Header, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("PUT", url, body)
	if err != nil {
		return nil, err
	}

	req.Header = header
	return h.client.Do(req)
}

func (h *HttpClient) Delete(url string, header http.Header) (*http.Response, error) {
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header = header
	return h.client.Do(req)
}

func (h *HttpClient) Patch(url string, header http.Header, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("PATCH", url, body)
	if err != nil {
		return nil, err
	}

	req.Header = header
	return h.client.Do(req)
}
