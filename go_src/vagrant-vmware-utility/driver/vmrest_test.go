// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package driver

import (
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

type TestClient struct {
	Responses []*Response
	Results   []*http.Response
}

type Response struct {
	Body   string
	Status int
	Error  error
}

func (t *TestClient) Do(req *http.Request) (*http.Response, error) {
	if len(t.Responses) < 1 {
		panic("no responses configured")
	}
	r := t.Responses[0]
	t.Responses = t.Responses[1:]
	result := &http.Response{
		Status:        strconv.Itoa(r.Status),
		StatusCode:    r.Status,
		Body:          ioutil.NopCloser(strings.NewReader(r.Body)),
		ContentLength: int64(len(r.Body)),
		Request:       req}
	t.Results = append(t.Results, result)
	return result, r.Error
}

func TestSimpleSetup(t *testing.T) {
}
