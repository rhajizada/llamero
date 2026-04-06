package proxy

import (
	"bytes"
	"net/http"
)

type Forwarder interface {
	Forward(r *http.Request, method, baseAddress, targetPath string, body []byte) (*http.Response, error)
	ForwardGET(r *http.Request, baseAddress, targetPath string) (*http.Response, error)
	ForwardLLM(r *http.Request, baseAddress string, body []byte) (*http.Response, error)
}

type Client struct {
	httpClient *http.Client
}

func New(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{httpClient: httpClient}
}

func (c *Client) Forward(r *http.Request, method, baseAddress, targetPath string, body []byte) (*http.Response, error) {
	target, err := buildBackendURL(baseAddress, targetPath, r.URL.RawQuery)
	if err != nil {
		return nil, err
	}

	//nolint:gosec // buildBackendURL validates the backend target before constructing the request.
	req, err := http.NewRequestWithContext(r.Context(), method, target, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	copyHeaders(req.Header, r.Header)
	stripProxyHeaders(req.Header)
	applyForwardHeaders(req, r)
	//nolint:gosec // request target was validated by buildBackendURL.
	return c.httpClient.Do(req)
}

func (c *Client) ForwardGET(r *http.Request, baseAddress, targetPath string) (*http.Response, error) {
	target, err := buildBackendURL(baseAddress, targetPath, r.URL.RawQuery)
	if err != nil {
		return nil, err
	}

	//nolint:gosec // buildBackendURL validates the backend target before constructing the request.
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	copyHeaders(req.Header, r.Header)
	stripProxyHeaders(req.Header)
	applyForwardHeaders(req, r)
	//nolint:gosec // request target was validated by buildBackendURL.
	return c.httpClient.Do(req)
}

func (c *Client) ForwardLLM(r *http.Request, baseAddress string, body []byte) (*http.Response, error) {
	return c.Forward(r, r.Method, baseAddress, normalizeLLMPath(r.URL.Path), body)
}
