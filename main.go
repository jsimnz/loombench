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
	"io/ioutil"
	"math"
	"strings"
	// "net/http"
	// gourl "net/url"
	"os"
	"os/exec"
	"os/signal"
	// "regexp"
	"runtime"
	// "strings"
	"time"

	"github.com/jsimnz/loombench/requester"
	"github.com/jsimnz/loombench/types"
	"github.com/jsimnz/loombench/version"

	"github.com/Jeffail/gabs"
	"github.com/cheggaaa/pb"
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

	writeURL       = flag.String("w", "http://localhost:46658/rpc", "")
	readURL        = flag.String("r", "http://localhost:46658/query", "")
	chainID        = flag.String("i", "default", "")
	contractAddr   = flag.String("a", "SimpleStore", "")
	contractMethod = flag.String("m", "Set", "")
	privateKey     = flag.String("p", "genkey", "")
	directory      = flag.String("d", "", "")
	gitPath        = flag.String("g", "$GOPATH/src/github.com/jsimnz/loombench", "")

	updateGenesis = flag.Bool("update-genesis", false, "")

	//optimization
	rawRequest = flag.Bool("raw-request", false, "")
	fastJson   = flag.Bool("fast-json", false, "")
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
  -i  Chain ID for the Loom DAppChain. Default: default.
  -a  Address of the contract to execute on the Loom DAppChain. Default: SimpleStore
  -m  Method to invoke when calling the Loom Contract. Default: Set.
  -p  Private key file to read the signing private key from. Default: genkey
      A value of 'genkey' will generate a key on demand for the entire benchmark
	  session. Note a key will be generated for each batch of concurrent requests.
  -d  Directory containing a Loom DAppChain instance.
  -g  Path to loombench git source repo. Default: $GOPATH/src/github.com/jsimnz/loombench.

  Config
  ======
  -disable-keepalive    Disable keep-alive, prevents re-use of TCP
                        connections between different HTTP requests.
  -cpus                 Number of used cpu cores.
						(default for current machine is %d cores)
  -update-genesis		Update the genesis.json file when available (loombench install)

  Optimizations
  =============
  -raw-request	Craft a raw marshalled protobuf request ahead of time.
  -fast-json	Use a faster json encoder, requires -raw-request to be true. 		

  Advanced
  ========
  TODO - Advanced contract selection and execution.
`

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, fmt.Sprintf(usage, runtime.NumCPU()))
	}

	var cmd string
	if len(os.Args) > 1 {
		cmd = os.Args[1]
		os.Args = append(os.Args[0:1], os.Args[2:]...)
	}

	flag.Parse()

	if cmd == "" {
		usageAndExit("")
	} else if cmd == "run" {
		runCmd()
	} else if cmd == "install" {
		installCmd()
	} else if cmd == "help" {
		usageAndExit("")
	} else {
		usageAndExit(fmt.Sprintf("%s is not a command", cmd))
	}
}

func installCmd() {
	// build contract
	if *directory == "" {
		fmt.Println("Directory:", *directory)
		usageAndExit("Need to specify Loom DAppChain directory to install to")
	}

	var genesisFileBuf []byte
	var err error
	genesisPath := (*directory) + "/genesis.json"
	if *updateGenesis {
		genesisFileBuf, err = ioutil.ReadFile(genesisPath)
		if err != nil {
			panic(err)
		}
	}
	outputFile := (*directory) + "/contracts/simplestore." + version.ContractVersion

	if strings.Contains(*gitPath, "$GOPATH") {
		*gitPath = strings.Replace(*gitPath, "$GOPATH", os.Getenv("GOPATH"), -1)
	}
	contractSrcFile := (*gitPath) + "/contracts/simplestore.go"
	buildCmdArgs := []string{
		"build",
		"-o", outputFile,
		contractSrcFile,
	}
	cmd := exec.Command("go", buildCmdArgs...)
	err = cmd.Run()
	if err != nil {
		panic(err)
	}

	// update genesis file
	if *updateGenesis {
		genesisJson, err := gabs.ParseJSON(genesisFileBuf)
		if err != nil {
			panic(err)
		}

		contractEntry := map[string]interface{}{
			"vm":       "plugin",
			"format":   "plugin",
			"name":     "SimpleStore",
			"location": "simplestore:0.0.2",
			"init":     nil,
		}
		genesisJson.ArrayAppendP(contractEntry, "contracts")
		// TODO: Fix JSON generation value ordering
		err = ioutil.WriteFile(genesisPath, genesisJson.BytesIndent("", "\t"), 0777)
		if err != nil {
			panic(err)
		}
	}

	fmt.Println("Installed to", (*directory))
}

func runCmd() {
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

	// Craft transaction body
	body := &types.LoomBenchWriteTx{
		Key: []byte("hello"),
		Val: []byte("world"),
	}

	if *fastJson && !(*rawRequest) {
		usageAndExit("Fast JSON optimization requires the -raw-request flag")
	}

	w := &requester.Work{
		// Request:           req,
		RequestBody:       body,
		UseRawRequest:     *rawRequest,
		TransactionType:   *transactions,
		Ratio:             *ratio,
		N:                 num,
		C:                 conc,
		QPS:               q,
		Timeout:           *t,
		WriteURL:          *writeURL,
		ReadURL:           *readURL,
		ChainID:           *chainID,
		ContractAddress:   *contractAddr,
		ContractMethod:    *contractMethod,
		PrivateKey:        *privateKey,
		DisableKeepAlives: *disableKeepAlives,
		UseProgress:       true,
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

	// progress bar
	go func() {
		count := num
		bar := pb.StartNew(count)
		for _ = range w.Progress {
			bar.Increment()
			// time.Sleep(time.Millisecond)
			if cur := bar.Get(); int(cur) == count-1 {
				break
			}
		}
		bar.Increment()
		bar.FinishPrint("Done!")
	}()

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
