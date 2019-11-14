package api

import (
	"fmt"
	"github.com/programmfabrik/apitest/pkg/lib/util"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/programmfabrik/go-test-utils"
)

func TestRequestBuildHttp(t *testing.T) {
	request := Request{
		Endpoint: "endpoint",
		Method:   "DO!",
		QueryParams: map[string]interface{}{
			"query_param": "value",
		},
		ServerURL: "serverUrl",
	}
	request.buildPolicy = func(request Request) (ah map[string]string, b io.Reader, err error) {
		ah = make(map[string]string)
		ah["mock-header"] = "application/mock"
		b = strings.NewReader("mock_body")
		return ah, b, nil
	}
	httpRequest, err := request.buildHttpRequest()
	test_utils.CheckError(t, err, fmt.Sprintf("error building http-request: %s", err))
	test_utils.AssertStringEquals(t, httpRequest.Header.Get("mock-header"), "application/mock")

	assertBody, err := ioutil.ReadAll(httpRequest.Body)
	test_utils.CheckError(t, err, fmt.Sprintf("error reading http-request body: %s", err))
	test_utils.AssertStringEquals(t, string(assertBody), "mock_body")

	url := httpRequest.URL
	test_utils.AssertStringEquals(t, url.RawQuery, "query_param=value")
	test_utils.AssertStringEquals(t, url.Path, "serverUrl/endpoint")
}

func TestBuildCurl(t *testing.T) {
	request := Request{
		Endpoint: "endpoint",
		Method:   "GET",
		QueryParams: map[string]interface{}{
			"query_param": "value",
		},
		ServerURL: "https://serverUrl",
		Body: util.JsonObject{
			"hey": 1,
		},
	}

	t.Log(request.ToString(true))

	if request.ToString(true) != `curl -X 'GET' -d '{"hey":1}' -H 'Content-Type: application/json' 'https://serverUrl/endpoint?query_param=value'` {
		t.Fatalf("Did not match right curl command. Expected '%s' != '%s' GOT", `curl -X 'GET' -d '{"hey":1}' -H 'Content-Type: application/json' 'https://serverUrl/endpoint?query_param=value'`, request.ToString(true))
	}
}