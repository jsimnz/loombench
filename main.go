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

// Command hey is an HTTP load generator.
package main

import (
	"flag"
	"fmt"
	// "io/ioutil"
	"math"
	// "net/http"
	// gourl "net/url"
	"os"
	"os/signal"
	// "regexp"
	"runtime"
	// "strings"
	"time"

	"github.com/jsimnz/loombench/requester"
)

const (
	headerRegexp = `^([\w-]+):\s*(.+)`
	authRegexp   = `^(.+):([^\s].+)`
	heyUA        = "hey/0.0.1"
)

var (
	transactions = flag.String("x", "", "")
	ratio        = flag.Float64("o", 0.5, "")

	c = flag.Int("c", 50, "")
	n = flag.Int("n", 200, "")
	q = flag.Float64("q", 0, "")
	t = flag.Int("t", 20, "")
	z = flag.Duration("z", 0, "")

	cpus              = flag.Int("cpus", runtime.GOMAXPROCS(-1), "")
	disableKeepAlives = flag.Bool("disable-keepalive", false, "")

	writeURL   = flag.String("w", "http://localhost:46658/rpc", "")
	readURL    = flag.String("r", "http://localhost:46658/query", "")
	privateKey = flag.String("p", "genkey", "")
)

var usage = `Usage: loombench [options...] 

Commands:
  install	Add the loombench contract to an existing Loom DAppChain.
  run		Run the benchmarking utility against a running DAppChain.

Flags:
  Basic
  =====
  -x  Type of transactions to submit to the DAppChain. 
      Available values: read, write, mixed.
  -o  Ratio to use of transaction types between read and write calls.
      Example: -o 0.75 means 75%% of the transactions are reads and
      25%% are writes.

  -n  Number of requests to run. Default is 200.
  -c  Number of requests to run concurrently. Total number of requests cannot
      be smaller than the concurrency level. Default is 50.
  -q  Rate limit, in queries per second (QPS). Default is no rate limit.
  -z  Duration of application to send requests. When duration is reached,
      application stops and exits. If duration is specified, n is ignored.
      Examples: -z 10s -z 3m..
  -t  Timeout for each request in seconds. Default is 20, use 0 for infinite.
  
  
  Loom
  ====
  -w  Write URL for submitting transactions to a Loom DAppChain 
      Default: http://localhost:46658/rpc.
  -r  Read URL for retrieving transactions from a Loom DAppChain 
      Default: http://localhost:46658/query.
  -p  Private key file to read the signing private key from.
      A value of 'genkey' will generate a key on demand for the entire benchmark
      session.

  Config
  ======
  -disable-keepalive    Disable keep-alive, prevents re-use of TCP
                        connections between different HTTP requests.
  -cpus                 Number of used cpu cores.
                        (default for current machine is %d cores)

  Advanced
  ========
  TODO - Advanced contract selection and execution.
`

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, fmt.Sprintf(usage, runtime.NumCPU()))
	}

	flag.Parse()
	var tt string = *transactions
	if len(tt) <= 0 {
		usageAndExit("")
	}

	runtime.GOMAXPROCS(*cpus)
	num := *n
	conc := *c
	q := *q
	dur := *z

	if dur > 0 {
		num = math.MaxInt32
		if conc <= 0 {
			usageAndExit("-c cannot be smaller than 1.")
		}
	} else {
		if num <= 0 || conc <= 0 {
			usageAndExit("-n and -c cannot be smaller than 1.")
		}

		if num < conc {
			usageAndExit("-n cannot be less than -c.")
		}
	}

	w := &requester.Work{
		// Request:           req,
		// RequestBody:       bodyAll,
		TransactionType:   *transactions,
		Ratio:             *ratio,
		N:                 num,
		C:                 conc,
		QPS:               q,
		Timeout:           *t,
		WriteURL:          *writeURL,
		ReadURL:           *readURL,
		PrivateKey:        *privateKey,
		DisableKeepAlives: *disableKeepAlives,
	}
	w.Init()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		w.Stop()
	}()
	if dur > 0 {
		go func() {
			time.Sleep(dur)
			w.Stop()
		}()
	}
	w.Run()
}

func errAndExit(msg string) {
	fmt.Fprintf(os.Stderr, msg)
	fmt.Fprintf(os.Stderr, "\n")
	os.Exit(1)
}

func usageAndExit(msg string) {
	if msg != "" {
		fmt.Fprintf(os.Stderr, msg)
		fmt.Fprintf(os.Stderr, "\n\n")
	}
	flag.Usage()
	fmt.Fprintf(os.Stderr, "\n")
	os.Exit(1)
}

// func parseInputWithRegexp(input, regx string) ([]string, error) {
// 	re := regexp.MustCompile(regx)
// 	matches := re.FindStringSubmatch(input)
// 	if len(matches) < 1 {
// 		return nil, fmt.Errorf("could not parse the provided input; input = %v", input)
// 	}
// 	return matches, nil
// }

// type headerSlice []string

// func (h *headerSlice) String() string {
// 	return fmt.Sprintf("%s", *h)
// }

// func (h *headerSlice) Set(value string) error {
// 	*h = append(*h, value)
// 	return nil
// }
