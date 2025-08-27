package http

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/bascanada/logviewer/pkg/ty"
)

type Auth interface {
	Login(req *http.Request) error
}

type CookieAuth struct {
	Cookie string
}

func (c CookieAuth) Login(req *http.Request) error {

	req.Header.Set("Cookie", c.Cookie)

	return nil
}

// HeaderAuth sets fixed headers (like Authorization) on each request.
type HeaderAuth struct {
	Headers ty.MS
}

func (h HeaderAuth) Login(req *http.Request) error {
	for k, v := range h.Headers {
		req.Header.Set(k, v)
	}
	return nil
}

type HttpClient struct {
	client http.Client
	url    string
}

// Debug controls whether verbose HTTP-level debug logs are emitted. Tests and
// production code can toggle this to avoid leaking secrets into logs.
var Debug = false

// SetDebug sets the package debug flag.
func SetDebug(d bool) {
	Debug = d
}

// DebugEnabled returns whether HTTP debug logging is enabled.
func DebugEnabled() bool {
	return Debug
}

func (c HttpClient) post(path string, headers ty.MS, buf *bytes.Buffer, responseData interface{}, auth Auth) error {
	path = c.url + path

	//if Debug {
	log.Printf("[POST]%s %s"+ty.LB, path, buf.String())
	//}

	req, err := http.NewRequest("POST", path, buf)
	if err != nil {
		return err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Log headers but redact sensitive values (Authorization, Cookie, tokens)
	if Debug {
		log.Printf("[POST-HEADERS] %s\n", maskHeaderMap(req.Header))
	}

	if auth != nil {
		if err = auth.Login(req); err != nil {
			log.Printf("%s", err.Error())
		}
	}

	res, err := c.client.Do(req)
	if err != nil {
		return err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	//println(string(resBody))

	if res.StatusCode >= 400 {
		log.Printf("error %d  %s"+ty.LB, res.StatusCode, string(resBody))
		return errors.New(string(resBody))
	}

	return json.Unmarshal(resBody, &responseData)
}

func (c HttpClient) PostData(path string, headers ty.MS, body ty.MS, responseData interface{}, auth Auth) error {

	headers["Content-Type"] = "application/x-www-form-urlencoded"

	// Build form-encoded body using url.Values to ensure proper encoding of keys/values
	values := url.Values{}
	for k, v := range body {
		values.Add(k, v)
	}

	encoded := values.Encode()
	buf := bytes.NewBufferString(encoded)

	return c.post(path, headers, buf, responseData, auth)

}

func (c HttpClient) PostJson(path string, headers ty.MS, body interface{}, responseData interface{}, auth Auth) error {

	headers["Content-Type"] = "application/json"

	var buf bytes.Buffer
	encErr := json.NewEncoder(&buf).Encode(body)
	if encErr != nil {
		return encErr
	}

	return c.post(path, headers, &buf, responseData, auth)

}

func (c HttpClient) Get(path string, queryParams ty.MS, body interface{}, responseData interface{}, auth Auth) error {

	var buf bytes.Buffer

	if body != nil {
		encErr := json.NewEncoder(&buf).Encode(body)
		if encErr != nil {
			return encErr
		}

	}
	path = c.url + path

	q := url.Values{}

	for k, v := range queryParams {
		q.Add(k, v)
	}

	queryParamString := q.Encode()

	if queryParamString != "" {
		path += "?" + queryParamString
	}

	if Debug {
		log.Printf("[GET]%s %s\n", path, buf.String())
	}

	req, err := http.NewRequest("GET", path, &buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	if auth != nil {
		if err = auth.Login(req); err != nil {
			log.Printf("%s", err.Error())
		}
	}

	res, getErr := c.client.Do(req)
	if getErr != nil {
		return getErr
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	resBody, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		return readErr
	}

	// Log a truncated GET response body for debugging (avoid huge output)
	if Debug && len(resBody) > 0 {
		s := string(resBody)
		if len(s) > 2000 {
			s = s[:2000] + "...TRUNCATED"
		}
		log.Printf("[GET-RAW] %s", s)
	}

	jsonErr := json.Unmarshal(resBody, &responseData)
	if jsonErr != nil {
		return jsonErr
	}

	return nil
}

func GetClient(url string) HttpClient {
	// Normalize URL: if scheme is missing, default to https. Also remove
	// any trailing slash to avoid double slashes when appending paths.
	if url != "" {
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			url = "https://" + url
		}
		// remove trailing slash for consistent concatenation
		for strings.HasSuffix(url, "/") {
			url = strings.TrimSuffix(url, "/")
		}
	}

	spaceClient := getSpaceClient()

	return HttpClient{
		client: spaceClient,
		url:    url,
	}
}

func getSpaceClient() http.Client {
	switch v := http.DefaultTransport.(type) {
	case (*http.Transport):
		customTransport := v.Clone()
		customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		return http.Client{Transport: customTransport}
	default:
		return http.Client{}

	}

}

// maskHeaderMap returns a string representation of headers with sensitive
// values redacted (keeps first 4 chars for debugging). This avoids leaking
// secrets into logs while letting us verify headers are present.
func maskHeaderMap(h http.Header) string {
	redacted := []string{}
	for k, vals := range h {
		v := ""
		if len(vals) > 0 {
			val := vals[0]
			// Redact common sensitive headers
			switch strings.ToLower(k) {
			case "authorization", "cookie", "x-splunk-token", "x-auth-token":
				if len(val) > 4 {
					v = val[:4] + "...REDACTED"
				} else {
					v = "REDACTED"
				}
			default:
				v = val
			}
		}
		redacted = append(redacted, fmt.Sprintf("%s: %s", k, v))
	}
	return strings.Join(redacted, "; ")
}
