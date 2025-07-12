package requests

import (
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"slices"

	"github.com/sirupsen/logrus"
)

const (
	APPLICATION_JSON = "application/json"
	APPLICATION_XML  = "application/xml"
	H_CONTENT_TYPE   = "Content-Type"
	MAX_RETRIES      = 5
	MAX_RETRY_DELAY  = 10
)

var (
	RETRY_ERROR_CODES = []int{
		502,
		503,
		504,
	}
)

type ApiRequest struct {
	TestEnvironment    bool
	Sensitive          bool
	TestSimulationPath string
}

var Request *ApiRequest

func init() {
	Request = &ApiRequest{
		TestEnvironment: false,
	}
}

func ReadFile(file string) ([]byte, error) {
	openedFile, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer openedFile.Close()

	byteValue, _ := io.ReadAll(openedFile)

	return byteValue, nil

}

func (apir *ApiRequest) Request(request string, headers map[string][]string, urlPath string, method string, output any) (err error) {

	reqURL, _ := url.Parse(urlPath)

	reqBody := io.NopCloser(strings.NewReader(request))

	req := &http.Request{
		Method: method,
		URL:    reqURL,
		Header: headers,
		Body:   reqBody,
	}

	if apir.TestEnvironment {

		err = Simulation(output, apir.TestSimulationPath, fmt.Sprintf("%s|%s", method, urlPath))

		if err != nil {
			return err
		}
		return
	}

	res, err := ExternalRequestWithRetry(req, MAX_RETRIES, time.Duration(MAX_RETRY_DELAY)*time.Second)
	if err != nil {
		logrus.Errorf("SEND REQUEST | URL : %s | METHOD : %s | BODY : %s | ERROR : %v", urlPath, method, request, err)
		return err
	}

	data, _ := io.ReadAll(res.Body)
	defer res.Body.Close()
	resbody := string(data)

	if len(data) > 0 && output != nil {
		switch headers[H_CONTENT_TYPE][0] {
		case APPLICATION_XML:
			if err = xml.Unmarshal(data, output); err != nil {
				logrus.Errorf("Failed to parse XML for SEND REQUEST | URL : %s | METHOD : %s | BODY : %s | STATUS : %s | HTTP_CODE : %d", urlPath, method, request, res.Status, res.StatusCode)
				err = fmt.Errorf("%d | RESPONSE : %s", res.StatusCode, resbody)
				return
			}
		default:
			if err = json.Unmarshal(data, output); err != nil {
				logrus.Errorf("Failed to parse JSON for SEND REQUEST | URL : %s | METHOD : %s | BODY : %s | STATUS : %s | HTTP_CODE : %d", urlPath, method, request, res.Status, res.StatusCode)
				err = fmt.Errorf("%d | RESPONSE : %s", res.StatusCode, resbody)
				return
			}
		}
	}

	if res.StatusCode > 299 || res.StatusCode <= 199 {
		logrus.Errorf("SEND REQUEST | URL : %s | METHOD : %s | BODY : %s | STATUS : %s | HTTP_CODE : %d", urlPath, method, request, res.Status, res.StatusCode)
		err = fmt.Errorf("%d | RESPONSE : %s", res.StatusCode, resbody)
		return
	}
	if !apir.Sensitive {
		logrus.Infof("SEND REQUEST | URL : %s | METHOD : %s | BODY : %s | STATUS : %s | HTTP_CODE : %d | RESPONSE : %s", urlPath, method, request, res.Status, res.StatusCode, resbody)
	}

	return
}

func ExternalRequestWithRetry(req *http.Request, maxRetries int, baseDelay time.Duration) (*http.Response, error) {
	var res *http.Response
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
		res, err = ExternalRequestTimer(req)
		if err == nil && res.StatusCode >= 200 && res.StatusCode < 400 {
			// Success: Return the response
			return res, nil
		}
		// Retry on network-level errors (e.g., EOF, temporary network issues)
		if err != nil {
			if isTemporaryNetworkError(err) {
				logrus.Warnf("Retrying due to network error (attempt %d/%d): %v", attempt, maxRetries, err)
				time.Sleep(baseDelay*time.Duration(attempt) + jitter)
				continue
			}
			// If it's a non-retryable error, exit the loop
			break
		}
		// Check if the error is recoverable
		if res != nil {
			if slices.Contains(RETRY_ERROR_CODES, res.StatusCode) {
				// Retry on HTTP status codes greater than 500
				logrus.Warnf("Retrying due to server error: %d %s", res.StatusCode, res.Status)
				time.Sleep(baseDelay*time.Duration(attempt) + jitter)
				continue
			} else if res.StatusCode == 429 {
				// Retry on 429 Too Many Requests
				var delay time.Duration
				if retryAfter := res.Header.Get("Retry-After"); retryAfter != "" {
					parsedDelay, parseErr := time.ParseDuration(retryAfter + "s")
					if parseErr == nil {
						delay = parsedDelay
					} else {
						delay = baseDelay * time.Duration(attempt)
					}
				} else {
					delay = baseDelay * time.Duration(attempt)
				}
				time.Sleep(delay + jitter)
				continue
			}
		}
		// If the error is not recoverable, break the loop
		break
	}

	// Return final err
	return res, err
}

func ExternalRequestTimer(req *http.Request) (*http.Response, error) {

	var start, connect, dns, tlsHandshake time.Time

	trace := &httptrace.ClientTrace{
		DNSStart: func(dsi httptrace.DNSStartInfo) {
			dns = time.Now()
			logrus.Debug(dsi)
		},
		DNSDone: func(ddi httptrace.DNSDoneInfo) {
			logrus.Debug(ddi)
			logrus.Infof("DNS Done: %v", time.Since(dns))
		},

		TLSHandshakeStart: func() { tlsHandshake = time.Now() },
		TLSHandshakeDone: func(cs tls.ConnectionState, err error) {
			// logrus.Debug(cs, err)
			logrus.Infof("TLS Handshake: %v", time.Since(tlsHandshake))
		},

		ConnectStart: func(network, addr string) {
			connect = time.Now()
			logrus.Debug(network, addr)
		},
		ConnectDone: func(network, addr string, err error) {
			logrus.Debug(network, addr, err)
			logrus.Infof("Connect time: %v", time.Since(connect))
		},

		GotFirstResponseByte: func() {
			logrus.Warnf("TAT : %v", time.Since(start))
		},
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	start = time.Now()

	// NOTE: Below line is to ignore ssl certificate
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	res, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return res, err
	}
	return res, nil
}

func Simulation(output any, simpath, uristr string) (err error) {
	const CALLER = 2
	pc, _, _, _ := runtime.Caller(CALLER)
	caller := runtime.FuncForPC(pc).Name()

	simulations := map[string][]struct {
		ExpectedOutput interface{} `json:"expected_output" binding:"required"`
		TargetURI      string      `json:"target_uri" binding:"required"`
	}{}
	bytt, err := ReadFile(simpath)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(bytt, &simulations); err != nil {
		return err
	}

	uris, ok := simulations[caller]

	if !ok {
		return fmt.Errorf("caller function simualtion not setup, %s ", caller)
	}
	for _, uri := range uris {
		if strings.EqualFold(uri.TargetURI, uristr) {

			out, err := json.Marshal(uri.ExpectedOutput)
			if err != nil {
				return err
			}
			err = json.Unmarshal(out, output)
			if err != nil {
				return err
			}
			return nil
		}
	}
	return fmt.Errorf("uri for function test not set, %s", uristr)
}

func isTemporaryNetworkError(err error) bool {
	var netErr net.Error

	if errors.Is(err, io.EOF) {
		return true
	}
	return errors.As(err, &netErr) && netErr.Timeout()
}
