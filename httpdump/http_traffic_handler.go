package httpdump

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"fmt"
	"github.com/alonana/httshark/core"
	"github.com/google/gopacket/tcpassembly/tcpreader"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// ConnectionKey contains src and dst endpoint identify a connection
type ConnectionKey struct {
	src Endpoint
	dst Endpoint
}

func (ck *ConnectionKey) reverse() ConnectionKey {
	return ConnectionKey{ck.dst, ck.src}
}

// return the src ip and port
func (ck *ConnectionKey) srcString() string {
	return ck.src.String()
}

// return the dst ip and port
func (ck *ConnectionKey) dstString() string {
	return ck.dst.String()
}

// HTTPConnectionHandler impl ConnectionHandler
type HTTPConnectionHandler struct {
}

func (handler *HTTPConnectionHandler) handle(src Endpoint, dst Endpoint, connection *TCPConnection) {
	ck := ConnectionKey{src, dst}
	trafficHandler := &HTTPTrafficHandler{
		key:       ck,
		buffer:    new(bytes.Buffer),
		startTime: connection.lastTimestamp,
	}
	waitGroup.Add(1)
	go trafficHandler.handle(connection)
}

func (handler *HTTPConnectionHandler) finish() {
	//handler.printer.finish()
}

// HTTPTrafficHandler parse a http connection traffic and send to printer
type HTTPTrafficHandler struct {
	startTime time.Time
	endTime   time.Time
	key       ConnectionKey
	buffer    *bytes.Buffer
}

// read http request/response stream, and do output
func (h *HTTPTrafficHandler) handle(connection *TCPConnection) {
	defer waitGroup.Done()
	defer connection.upStream.Close()
	defer connection.downStream.Close()
	// filter by args setting

	requestReader := bufio.NewReader(connection.upStream)
	defer discardAll(requestReader)
	responseReader := bufio.NewReader(connection.downStream)
	defer discardAll(responseReader)

	for {
		h.buffer = new(bytes.Buffer)
		req, err := http.ReadRequest(requestReader)
		h.startTime = connection.lastTimestamp

		if err != nil {
			if err != io.EOF {
				logger.Warn("Error parsing HTTP requests:", err)
			}
			break
		}

		// if is websocket request,  by header: Upgrade: websocket
		expectContinue := req.Header.Get("Expect") == "100-continue"

		resp, err := http.ReadResponse(responseReader, nil)

		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				core.V5("Error parsing HTTP response: unexpected end, ", err, connection.clientID)
				break
			} else {
				core.Warn("Error parsing HTTP response:", err, connection.clientID)
			}
			h.report(req, nil)
			discardAll(req.Body)
			break
		}

		h.endTime = connection.lastTimestamp

		h.report(req, resp)
		discardAll(req.Body)

		if expectContinue {
			if resp.StatusCode == 100 {
				// read next response, the real response
				resp, err := http.ReadResponse(responseReader, nil)
				if err == io.EOF {
					logger.Warn("Error parsing HTTP requests: unexpected end, ", err)
					break
				}
				if err == io.ErrUnexpectedEOF {
					logger.Warn("Error parsing HTTP requests: unexpected end, ", err)
					// here return directly too, to avoid error when long polling connection is used
					break
				}
				if err != nil {
					logger.Warn("Error parsing HTTP response:", err, connection.clientID)
					break
				}
				h.report(req, resp)
			}
		}
	}
}

func (h *HTTPTrafficHandler) report(req *http.Request, res *http.Response) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		core.Warn("read request body failed: %v", body)
		return
	}

	transaction := core.HttpTransaction{
		Request: core.HttpRequest{
			HttpEntry: core.HttpEntry{
				Time:    &h.startTime,
				Stream:  0,
				Data:    string(body),
				Version: req.Proto,
				Headers: h.convertHeaders(req.Header),
			},
			Method: req.Method,
			Path:   req.URL.Path,
			Query:  req.URL.RawQuery,
		},
	}

	if res != nil {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			core.Warn("read response body failed: %v", body)
			return
		}
		transaction.Response = &core.HttpResponse{
			HttpEntry: core.HttpEntry{
				Time:    &h.endTime,
				Stream:  0,
				Data:    string(body),
				Version: res.Proto,
				Headers: h.convertHeaders(res.Header),
			},
			Code: res.StatusCode,
		}
	}

	processor(transaction)
}

func (h *HTTPTrafficHandler) convertHeaders(httpHeaders http.Header) []string {
	headers := make([]string, len(httpHeaders))
	i := 0
	for name, values := range httpHeaders {
		headers[i] = fmt.Sprintf("%v: %v", name, values[0])
		i++
	}
	return headers
}

func (h *HTTPTrafficHandler) tryDecompress(header http.Header, reader io.ReadCloser) (io.ReadCloser, bool) {
	contentEncoding := header.Get("Content-Encoding")
	var nr io.ReadCloser
	var err error
	if contentEncoding == "" {
		// do nothing
		return reader, false
	} else if strings.Contains(contentEncoding, "gzip") {
		nr, err = gzip.NewReader(reader)
		if err != nil {
			return reader, false
		}
		return nr, true
	} else if strings.Contains(contentEncoding, "deflate") {
		nr, err = zlib.NewReader(reader)
		if err != nil {
			return reader, false
		}
		return nr, true
	} else {
		return reader, false
	}
}

func discardAll(r io.Reader) (dicarded int) {
	return tcpreader.DiscardBytesToEOF(r)
}
