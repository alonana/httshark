package httpdump

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/alonana/httshark/core"
	"github.com/alonana/httshark/core/aggregated"
	"github.com/google/gopacket/tcpassembly/tcpreader"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

func newHttpTrafficHandler(originalKey string, src Endpoint, dst Endpoint, connection *TCPConnection) {
	ck := ConnectionKey{src, dst}
	trafficHandler := &HTTPTrafficHandler{
		originalKey: originalKey,
		key:         ck,
		buffer:      new(bytes.Buffer),
		startTime:   connection.lastTimestamp,
	}
	go trafficHandler.handle(connection)
}

type HTTPTrafficHandler struct {
	startTime   time.Time
	endTime     time.Time
	key         ConnectionKey
	buffer      *bytes.Buffer
	originalKey string
}

// read http request/response stream, and do output
func (h *HTTPTrafficHandler) handle(connection *TCPConnection) {
	core.V2("%v http traffic - starting", h.originalKey)
	defer func() { _ = connection.upStream.Close() }()
	defer func() { _ = connection.downStream.Close() }()

	requestReader := bufio.NewReader(connection.upStream)
	defer discardAll(requestReader)
	responseReader := bufio.NewReader(connection.downStream)
	defer discardAll(responseReader)

	for {
		core.V2("%v http traffic - lopping", h.originalKey)

		h.buffer = new(bytes.Buffer)
		req, err := http.ReadRequest(requestReader)
		h.startTime = connection.lastTimestamp

		if err != nil {
			if err == io.EOF {
				core.V2("%v http traffic - break on EOF", h.originalKey)
			} else {
				core.V2("%v http traffic - break on error: %v", h.originalKey, limitedError(err))
				aggregated.Warn("Parsing HTTP request failed: %v", limitedError(err))
			}
			break
		}

		// if is websocket request,  by header: Upgrade: websocket
		expectContinue := req.Header.Get("Expect") == "100-continue"

		core.V2("%v http traffic - reading response starting", h.originalKey)
		resp, err := http.ReadResponse(responseReader, nil)
		core.V2("%v http traffic - reading response done", h.originalKey)

		if err != nil {
			if err != io.EOF && err != io.ErrUnexpectedEOF {
				aggregated.Warn("parsing HTTP response failed: %v", limitedError(err))
			}
			h.report(req, nil)
			discardAll(req.Body)
			core.V2("%v http traffic - read response error", h.originalKey)
			break
		}

		h.endTime = connection.lastTimestamp

		core.V2("%v http traffic - reporting", h.originalKey)
		h.report(req, resp)
		discardAll(req.Body)

		if expectContinue {
			core.V2("%v http traffic - expect continue", h.originalKey)
			if resp.StatusCode == 100 {
				// read next response, the real response
				resp, err := http.ReadResponse(responseReader, nil)
				if err != nil {
					if err != io.EOF && err != io.ErrUnexpectedEOF {
						aggregated.Warn("parsing HTTP continue response failed: %v", limitedError(err))
					}
					h.report(req, nil)
					discardAll(req.Body)
					core.V2("%v http traffic - expect continue read response error", h.originalKey)
					break
				}
				h.report(req, resp)
			}
		}
	}

	core.V2("%v http traffic - terminating", h.originalKey)
}

func (h *HTTPTrafficHandler) report(req *http.Request, res *http.Response) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		aggregated.Warn("read request body failed: %v", limitedError(err))
		return
	}

	fullUrl := "http://" + req.Host + req.URL.Path
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
			Path:   fullUrl,
			Query:  req.URL.RawQuery,
		},
	}

	if res != nil {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			aggregated.Warn("read response body failed: %v", limitedError(err))
			body = []byte("UNKNOWN")
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

func discardAll(r io.Reader) int {
	discarded, err := tcpreader.DiscardBytesToFirstError(r)
	if err != nil {
		if err != io.EOF {
			aggregated.Warn("discard bytes failed: %v", limitedError(err))
		}
	}
	return discarded
}

func limitedError(err error) string {
	s := err.Error()
	if len(s) > core.Config.LimitedErrorLength {
		return s[:core.Config.LimitedErrorLength]
	}
	return s
}
