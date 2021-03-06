// Copyright 2014 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package requester provides commands to run load tests and display results.
package requester

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/jsimnz/loombench/loomclient"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/auth"
	// "github.com/mailru/easyjson"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/net/http2"
)

// Max size of the buffer of result channel.
const maxResult = 1000000
const maxIdleConn = 500

type result struct {
	err           error
	statusCode    int
	duration      time.Duration
	connDuration  time.Duration // connection setup(DNS lookup + Dial up) duration
	dnsDuration   time.Duration // dns lookup duration
	reqDuration   time.Duration // request "write" duration
	resDuration   time.Duration // response "read" duration
	delayDuration time.Duration // delay between response and request
	contentLength int64
}

type Work struct {
	// Type of transactions to be used in the benchmark.
	TransactionType string

	// Ratio of transaction types.
	Ratio float64

	// Request is the request to be made.
	Request *http.Request

	RequestBody proto.Message

	// Raw byte array of a crafted request body
	RequestBodyRaw []byte

	// UseRawRequest is an option to craft the raw binary marshalled protobuf ahead of time
	UseRawRequest bool

	// UseFastJSON is an option to use an alternate JSON encoder to increase performace.
	UseFastJSON bool

	// N is the total number of requests to make.
	N int

	// C is the concurrency level, the number of concurrent workers to run.
	C int

	// H2 is an option to make HTTP/2 requests
	H2 bool

	// Timeout in seconds.
	Timeout int

	// Qps is the rate limit in queries per second.
	QPS float64

	// DisableCompression is an option to disable compression in response
	DisableCompression bool

	// DisableKeepAlives is an option to prevents re-use of TCP connections between different HTTP requests
	DisableKeepAlives bool

	// DisableRedirects is an option to prevent the following of HTTP redirects
	DisableRedirects bool

	// Output represents the output type. If "csv" is provided, the
	// output will be dumped as a csv stream.
	Output string

	// ProxyAddr is the address of HTTP proxy server in the format on "host:port".
	// Optional.
	ProxyAddr *url.URL

	// Contract Address
	ContractAddress string

	// Method to call on the Loom Contract
	ContractMethod string

	// Loom Chain ID
	ChainID string

	// Loom Write URL
	WriteURL string

	// Loom Read URL
	ReadURL string

	// Priate Key to transaction signing
	PrivateKey string

	// Writer is where results will be written. If nil, results are written to stdout.
	Writer io.Writer

	initOnce sync.Once
	results  chan *result
	stopCh   chan struct{}
	start    time.Duration

	// Progress tracking
	UseProgress bool
	Progress    chan struct{}

	report *report
}

func (b *Work) writer() io.Writer {
	if b.Writer == nil {
		return os.Stdout
	}
	return b.Writer
}

// Init initializes internal data-structures
func (b *Work) Init() {
	b.initOnce.Do(func() {
		b.results = make(chan *result, min(b.C*1000, maxResult))
		b.stopCh = make(chan struct{}, b.C)
		if b.UseProgress {
			b.Progress = make(chan struct{}, b.C*2)
		}
	})
}

// Run makes all the requests, prints the summary. It blocks until
// all work is done.
func (b *Work) Run() {
	b.Init()
	b.start = now()
	b.report = newReport(b.writer(), b.results, b.Output, b.N)
	// Run the reporter first, it polls the result channel until it is closed.
	go func() {
		runReporter(b.report)
	}()
	b.runWorkers()
	b.Finish()
}

func (b *Work) Stop() {
	// Send stop signal so that workers can stop gracefully.
	for i := 0; i < b.C; i++ {
		b.stopCh <- struct{}{}
	}
}

func (b *Work) Finish() {
	close(b.results)
	total := now() - b.start
	// Wait until the reporter is done.
	<-b.report.done
	b.report.finalize(total)
}

func (b *Work) makeRequest(lc *loomclient.ContractClient, rpc *loomclient.DAppChainRPCClient, nonce uint64) {
	s := now()
	// var size int64
	// var code int
	var connStart, resStart, reqStart, delayStart time.Duration
	var connDuration, resDuration, reqDuration, delayDuration time.Duration
	// req := cloneRequest(b.Request, b.RequestBody)
	trace := &httptrace.ClientTrace{
		GetConn: func(h string) {
			connStart = now()
		},
		GotConn: func(connInfo httptrace.GotConnInfo) {
			if !connInfo.Reused {
				connDuration = now() - connStart
			}
			reqStart = now()
		},
		WroteRequest: func(w httptrace.WroteRequestInfo) {
			reqDuration = now() - reqStart
			delayStart = now()
		},
		GotFirstResponseByte: func() {
			delayDuration = now() - delayStart
			resStart = now()
		},
	}

	// TODO: Save request body from contract params and clone request on each call
	// 		 to save time from client overhead.
	// add traceclient to DAppChainRPCClient
	rpc.UseTrace(trace)
	// make Loom Call
	var err error
	if b.UseRawRequest {
		var signedTxBytes, rpcReqBytes []byte
		contract := lc.GetContract()
		signedTxBytes, err = contract.SignTxBytes(b.RequestBodyRaw, nonce, lc.GetSigner())
		if err != nil {
			panic(err)
		}
		rpcReqBytes, err = contract.CraftRPCReqBytes("broadcast_tx_commit", signedTxBytes)
		if err != nil {
			panic(err)
		}
		err = contract.CallRaw(rpcReqBytes)
	} else {
		err = lc.Call(b.ContractMethod, b.RequestBody, nil)
	}

	// req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	// resp, err := c.Do(req)
	// if err == nil {
	// 	size = resp.ContentLength
	// 	code = resp.StatusCode
	// 	io.Copy(ioutil.Discard, resp.Body)
	// 	resp.Body.Close()
	// }
	t := now()
	resDuration = t - resStart
	finish := t - s
	b.results <- &result{
		// statusCode:    code,
		statusCode:    200, // TODO: Get stausCoec from Loom Call
		duration:      finish,
		err:           err,
		contentLength: 0, // TODO: Get ContentLength from Loom Call
		connDuration:  connDuration,
		// dnsDuration:   dnsDuration,
		reqDuration:   reqDuration,
		resDuration:   resDuration,
		delayDuration: delayDuration,
	}

	if b.UseProgress {
		b.Progress <- struct{}{}
	}

	// return err
}

func (b *Work) runWorker(client *http.Client, n int) {
	var throttle <-chan time.Time
	if b.QPS > 0 {
		throttle = time.Tick(time.Duration(1e6/(b.QPS)) * time.Microsecond)
	}

	if b.DisableRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	lc, rpc, err := b.createWorkerClients(client) // Create Loom Client
	if err != nil {
		panic(err)
	}

	nonce := uint64(0)
	// var err error
	if b.UseRawRequest {
		signer := lc.GetSigner()
		b.RequestBodyRaw, err = lc.GetContract().CraftCallTx(b.ContractMethod, b.RequestBody, signer)
		if err != nil {
			panic(err)
		}
		nonce, err = rpc.GetNonce(signer)
		if err != nil {
			panic(err)
		}
	}

	for i := 0; i < n; i++ {
		// Check if application is stopped. Do not send into a closed channel.
		select {
		case <-b.stopCh:
			return
		default:
			if b.QPS > 0 {
				<-throttle
			}
			b.makeRequest(lc, rpc, nonce)
			nonce++
		}
	}
}

func (b *Work) runWorkers() {
	var wg sync.WaitGroup
	wg.Add(b.C)

	tr := &http.Transport{
		// TLSClientConfig: &tls.Config{
		// 	InsecureSkipVerify: true,
		// 	ServerName:         b.Request.Host,
		// },
		MaxIdleConnsPerHost: min(b.C, maxIdleConn),
		DisableCompression:  b.DisableCompression,
		DisableKeepAlives:   b.DisableKeepAlives,
		Proxy:               http.ProxyURL(b.ProxyAddr),
	}
	if b.H2 {
		http2.ConfigureTransport(tr)
	} else {
		tr.TLSNextProto = make(map[string]func(string, *tls.Conn) http.RoundTripper)
	}
	httpclient := &http.Client{Transport: tr, Timeout: time.Duration(b.Timeout) * time.Second}

	// Ignore the case where b.N % b.C != 0.
	for i := 0; i < b.C; i++ {
		go func() {
			b.runWorker(httpclient, b.N/b.C)
			wg.Done()
		}()
	}
	wg.Wait()
}

func (b *Work) createWorkerClients(httpclient *http.Client) (*loomclient.ContractClient, *loomclient.DAppChainRPCClient, error) {
	// create signer
	// var signer *auth.Signer
	var privKey []byte
	var err error
	if b.PrivateKey == "genkey" {
		_, privKey, err = ed25519.GenerateKey(nil)
	} else {
		privKeyB64, err := ioutil.ReadFile(b.PrivateKey)
		if err != nil {
			return nil, nil, err
		}

		privKey, err = base64.StdEncoding.DecodeString(string(privKeyB64))
		if err != nil {
			return nil, nil, err
		}
	}
	if err != nil {
		return nil, nil, err
	}
	signer := auth.NewEd25519Signer(privKey)

	rpcClient := loomclient.NewDAppChainRPCClient(httpclient, b.ChainID, b.WriteURL, b.ReadURL)
	client, err := loomclient.NewContractClient(b.ContractAddress, b.ChainID, signer, rpcClient)

	return client, rpcClient, err
}

// cloneRequest returns a clone of the provided *http.Request.
// The clone is a shallow copy of the struct and its Header map.
func cloneRequest(r *http.Request, body []byte) *http.Request {
	// shallow copy of the struct
	r2 := new(http.Request)
	*r2 = *r
	// deep copy of the Header
	r2.Header = make(http.Header, len(r.Header))
	for k, s := range r.Header {
		r2.Header[k] = append([]string(nil), s...)
	}
	if len(body) > 0 {
		r2.Body = ioutil.NopCloser(bytes.NewReader(body))
	}
	return r2
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
