![loombench](https://github.com/jsimnz/loombench/raw/master/assets/LoomBench-logo.png)
---
A benchmarking tool for a Loom DAppChain powered blockchain. Based on [rakyll/hey](https://github.com/rakyll/hey)

### Requirements

Go version 1.7 or greater.

You must also have a deployed Loom DAppChain. You can see detailed instructions on setting one up at [Loom SDK](https://loomx.io/developers/docs/en/prereqs-ubuntu.html).


### Installation

To install this utility to you system use:
```
go get github.com/jsimnz/loombench
```
After you have run the above `go get` command, you'll also need to install the associated Go Contract that is used by default. 
```
loombench install -p /path/to/loom/directory -update-genesis
```
Alternatively you may use your own Contract, but isn't fully documented yet.

### Usage
Loombench runs a series of requests (transactions) against a running DAppChain. Afterwhich prints a summary report.
```
Usage: loombench [options...] 

Commands:
  install	Add the loombench contract to an existing Loom DAppChain.
  run		Run the benchmarking utility against a running DAppChain.

Flags:
  Basic
  =====
  -x  Type of transactions to submit to the DAppChain. 
      Available values: read, write, mixed.
  -o  Ratio to use of transaction types between read and write calls.
      Example: -o 0.75 means 75% of the transactions are reads and
      25% are writes.

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

  Config
  ======
  -disable-keepalive    Disable keep-alive, prevents re-use of TCP
                        connections between different HTTP requests.
  -cpus                 Number of used cpu cores.
                        (default for current machine is 4 cores)

  Advanced
  ========
  TODO - Advanced contract selection and execution.
  ```
  
 To run a simple benchmark, you may just use 
 ```
 loombench run
 ```
 Which runs using default settings.
 
 

### TODO
- Optimize request creation to reduce overhead
- More seemless contract install process
- Document alternative contract usage.
- Add support for calling EVM contracts

### Credits
Written by John-Alan Simmons. Based heavily on the http benchmark utility [rakyll/hey](https://github.com/rakyll/hey).

