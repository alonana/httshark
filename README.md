# HttShark

## Overview

HttShark is an application that captures packets from a network interface, 
collects and correlates HTTP requests and HTTP responses, 
and produces HAR files containing the HTTP transactions.

## Architecture

The application architecture is as follows:

![](./httshark.png)

The application can be configured to use one of the following capture engines:
* tshark 
* httpdump

### tshark capture engine
Using tshark as the capture engine executes the tshark command line to capture http packets.
For example, the following command line is executed:
```bash
sudo tshark -i eth0 \
    -f 'tcp port 80' \
    -d 'tcp.port==80,http' \
    -Y http \
    -T json \
    -e frame.time_epoch \
    -e tcp.stream \
    -e http.request \
    -e http.request.method \
    -e http.request.version \  
    -e http.request.uri.path \
    -e http.request.uri.query \
    -e http.request.line \
    -e http.file_data \
    -e http.response \
    -e http.response.version \
    -e http.response.code \
    -e http.response.line
```

The tshark command line arguments configure it to capture HTTP packets on a specific network interface,
and to output specific list of flags in a JSON format.

An example of such output is:

```json
  {
    "_index": "packets-2020-04-06",
    "_type": "pcap_file",
    "_score": null,
    "_source": {
      "layers": {
        "frame.time_epoch": ["1586165861.751442868"],
        "tcp.stream": ["0"],
        "http.request": ["1"],
        "http.request.method": ["GET"],
        "http.request.version": ["HTTP\/1.1"],
        "http.request.line": ["Host: example.com\r\n","User-Agent: curl\/7.58.0\r\n","Accept: *\/*\r\n"]
      }
    }
  }
```

A sequence of processors is run to produce har entries from the tshark STDOUT.
* ***Line Processor*** - 
collects line by line from the STDOUT, and detects JSON entry start and stop.
Once a JSON entry is located, it is sent to the next processor.
* ***Bulk Processor*** - 
parses a JSON entry, and converts it to a proprietary 
HTTP request and HTTP response structures.
These structures are sent to the next processor.
* ***Correlator Processor*** -
keeps in memory map of TCP stream ID to HTTP request.
Once a related TCP stream ID response is received,
it creates an proprietary HTTP transaction structure, 
and sends it to the next processor.
In case an HTTP request did not encounter a matching HTTP response within a certain timeout,
it is sent as a transaction without a response. 

### httpdump capture engine

The httpdump capture engine is an alternative for the HTTP dump.
It uses a mechanism based on the https://github.com/hsiafan/httpdump project.
This is based on google's gopacket project,
to capture TCP packets, and create HTTP transactions from it.

## Command Line Flags

*  -capture="tshark": capture engine to use, one of tshark,httpdump
*  -channel-buffer=1: channel buffer size.
 It configures the GO channel between the processors. 
 Using a higher value would allow one processor to provide several entries,
 while the next processor is still working on previous entries.
*  -device="": interface to use sniffing for
*  -drop-content-type="image,audio,video": comma separated list of content type whose body should be removed (case insensitive, using include for match)
*  -export-interval=10s: export HAL to file interval
*  -har-processer="file": processor of the har file. one of file,memory
*  -hosts="": comma separated list of IPs to sample. Empty list to sample all hosts
*  -output-folder=".": hal files output folder
*  -port=80: filter packets for this port
*  -response-check-interval=10s: check timed out responses interval
*  -response-timeout=5m0s: timeout for waiting for response
*  -verbose=0: print verbose information 0=nothing 5=all
