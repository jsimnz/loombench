package loomclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
)

//easyjson:json
type RPCRequest struct {
	Version string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"` // map[string]interface{} or []interface{}
	ID      string          `json:"id"`
}

//easyjson:json
type RPCResponse struct {
	Version string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

//easyjson:json
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

func (err RPCError) Error() string {
	if err.Data != "" {
		return fmt.Sprintf("RPC error %v - %s: %s", err.Code, err.Message, err.Data)
	}
	return fmt.Sprintf("RPC error %v - %s", err.Code, err.Message)
}

func NewRPCRequest(method string, params json.RawMessage, id string) RPCRequest {
	return RPCRequest{
		Version: "2.0",
		Method:  method,
		Params:  params,
		ID:      id,
	}
}

type JSONRPCClient struct {
	host     string
	client   *http.Client
	reqMaker func(*http.Request) *http.Request
}

type TracedJSONRPCClient struct {
	JSONRPCClient
}

func newHTTPDialer(host string) func(string, string) (net.Conn, error) {
	u, err := url.Parse(host)
	// default to tcp if nothing specified
	protocol := u.Scheme
	if err != nil {
		return func(_ string, _ string) (net.Conn, error) {
			return nil, fmt.Errorf("Invalid host: %s", host)
		}
	}
	if protocol == "http" {
		protocol = "tcp"
	}
	return func(p, a string) (net.Conn, error) {
		return net.Dial(protocol, u.Host)
	}
}

func DefaultHTTPClient(host string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Dial: newHTTPDialer(host),
		},
	}
}

func NewJSONRPCClient(client *http.Client, host string) *JSONRPCClient {
	if client.Transport == nil {
		tr := &http.Transport{
			Dial: newHTTPDialer(host),
		}
		client.Transport = tr
	} else if tr := client.Transport.(*http.Transport); tr.Dial == nil {
		// tr := client.Transport.(*http.Transport)
		tr.Dial = newHTTPDialer(host)
		client.Transport = tr
	}

	return &JSONRPCClient{
		host:   host,
		client: client,
	}
}

// func NewTracedJSONRPCClient(host string) *JSONRPCClient {
// 	return &JSONRPCClient{
// 		host: host,
// 		client: &http.Client{
// 			Transport: &http.Transport{
// 				Dial: newHTTPDialer(host),
// 			},
// 		},
// 	}
// }

func (c *JSONRPCClient) UseTrace(trace *httptrace.ClientTrace) {
	c.reqMaker = func(req *http.Request) *http.Request {
		return req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	}
}

func (c *JSONRPCClient) Call(method string, params map[string]interface{}, id string, result interface{}) error {
	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return err
	}
	rpcReq := NewRPCRequest(method, paramsBytes, id)
	reqBytes, err := json.Marshal(rpcReq)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.host, bytes.NewBuffer(reqBytes))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "text/json")
	resp, err := c.doReq(req)
	// resp, err := c.client.Post(c.host, "text/json", )

	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var rpcResp RPCResponse
	if err := json.Unmarshal(respBytes, &rpcResp); err != nil {
		return fmt.Errorf("error unmarshalling rpc response: %v", err)
	}
	if rpcResp.Error != nil {
		return fmt.Errorf("Response error: %v", rpcResp.Error)
	}
	if result != nil {
		if err := json.Unmarshal(rpcResp.Result, result); err != nil {
			return fmt.Errorf("error unmarshalling rpc response result: %v", err)
		}
	}
	return nil
}

func (c *JSONRPCClient) CallRaw(reqBytes []byte, result *BroadcastTxCommitResult) error {
	req, err := http.NewRequest("POST", c.host, bytes.NewBuffer(reqBytes))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "text/json")
	resp, err := c.doReq(req)
	// resp, err := c.client.Post(c.host, "text/json", )

	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var rpcResp RPCResponse
	if err := json.Unmarshal(respBytes, &rpcResp); err != nil {
		return fmt.Errorf("error unmarshalling rpc response: %v", err)
	}
	if rpcResp.Error != nil {
		return fmt.Errorf("Response error: %v", rpcResp.Error)
	}
	if result != nil {
		if err := json.Unmarshal(rpcResp.Result, result); err != nil {
			return fmt.Errorf("error unmarshalling rpc response result: %v", err)
		}
	}
	return nil
}

func (c *JSONRPCClient) doReq(req *http.Request) (*http.Response, error) {
	if c.reqMaker != nil {
		req = c.reqMaker(req)
	}
	return c.client.Do(req)
}
