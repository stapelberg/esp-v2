// Copyright 2018 Google Cloud Platform Proxy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package integration

import (
	"strings"
	"testing"
	"time"

	"cloudesf.googlesource.com/gcpproxy/tests/endpoints/bookstore-grpc/client"
	"cloudesf.googlesource.com/gcpproxy/tests/env"
	"cloudesf.googlesource.com/gcpproxy/tests/env/testdata"
)

const (
	addr = "127.0.0.1:8080"
)

func TestGrpc(t *testing.T) {
	serviceName := "bookstore-service"
	configId := "test-config-id"

	args := []string{"--service_name=" + serviceName, "--config_id=" + configId,
		"--skip_service_control_filter=true", "--backend_protocol=grpc"}

	s := env.NewTestEnv(true, true, true)

	if err := s.Setup("bookstore", args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()
	time.Sleep(time.Duration(5 * time.Second))

	tests := []struct {
		desc           string
		clientProtocol string
		method         string
		wantResp       string
	}{
		{
			desc:           "gRPC client calling gRPC backend",
			clientProtocol: "grpc",
			method:         "GetShelf",
			wantResp:       `{"theme":"Unknown Shelf"}`,
		},
		{
			desc:           "Http client calling gRPC backend",
			clientProtocol: "http",
			method:         "/v1/shelves/125",
			wantResp:       `{"id":"125","theme":"Unknown Shelf"}`,
		},
	}

	for _, tc := range tests {
		resp, err := client.MakeCall(tc.clientProtocol, addr, "GET", tc.method, "")
		if err != nil {
			t.Errorf("failed to run test: %s", err)
		}

		if !strings.Contains(resp, tc.wantResp) {
			t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, resp)
		}
	}
}

func TestGrpcJwt(t *testing.T) {
	serviceName := "bookstore-service"
	configId := "test-config-id"

	args := []string{"--service_name=" + serviceName, "--config_id=" + configId,
		"--skip_service_control_filter=true", "--backend_protocol=grpc"}

	s := env.NewTestEnv(true, true, true)

	if err := s.Setup("bookstore", args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()
	time.Sleep(time.Duration(5 * time.Second))

	tests := []struct {
		desc           string
		clientProtocol string
		httpMethod     string
		method         string
		token          string
		wantResp       string
		wantedError    string
	}{
		// Testing JWT is required or not.
		{
			desc:           "Fail for gPRC client, without valid JWT token",
			clientProtocol: "grpc",
			method:         "ListShelves",
			wantedError:    "code = Unauthenticated desc = Jwt is missing",
		},
		{
			desc:           "Fail for Http client, without valid JWT token",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves",
			wantedError:    "401 Unauthorized",
		},
		{
			desc:           "Fail for gPRC client, with bad JWT token",
			clientProtocol: "grpc",
			method:         "ListShelves",
			token:          testdata.FakeBadToken,
			wantedError:    "code = Unauthenticated desc = Jwt issuer is not configured",
		},
		{
			desc:           "Fail for Http client, with bad JWT token",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves",
			token:          testdata.FakeBadToken,
			wantedError:    "401 Unauthorized",
		},
		{
			desc:           "Succeed for gPRC client, with valid JWT token",
			clientProtocol: "grpc",
			method:         "CreateShelf",
			token:          testdata.FakeGoodToken,
			wantResp:       `{"id":"1","theme":"New Shelf"}`,
		},

		// Testing JWT RouteMatcher matches by HttpHeader and parameters in "{}", for Http Client only.
		{
			desc:           "Succeed for Http client, Jwt RouteMatcher matches by HttpHeader method",
			clientProtocol: "http",
			httpMethod:     "POST",
			method:         "/v1/shelves",
			token:          testdata.FakeGoodToken,
			wantResp:       `{"id":"1","theme":"New Shelf"}`,
		},
		{
			desc:           "Fail for Http client, Jwt RouteMatcher matches by HttpHeader method",
			clientProtocol: "http",
			httpMethod:     "POST",
			method:         "/v1/shelves",
			wantedError:    "401 Unauthorized",
		},
		{
			desc:           "Succeed for Http client, Jwt RouteMatcher works for multi query parameters",
			clientProtocol: "http",
			httpMethod:     "DELETE",
			method:         "/v1/shelves/125/books/001",
			token:          testdata.FakeGoodToken,
			wantResp:       "{}",
		},
		{
			desc:           "Fail for Http client, Jwt RouteMatcher works for multi query parameters",
			clientProtocol: "http",
			httpMethod:     "DELETE",
			method:         "/v1/shelves/125/books/001",
			wantedError:    "401 Unauthorized",
		},
		{
			desc:           "Succeed for Http client, Jwt RouteMatcher works for multi query parameters and HttpHeader, no audience",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves/125/books/001",
			wantResp:       `{"id":"125","title":"Unknown Book"}`,
		},

		// Test JWT with audiences.
		{
			desc:           "Succeed for gPRC client, with valid JWT token, with single audience",
			clientProtocol: "grpc",
			method:         "ListShelves",
			token:          testdata.FakeGoodTokenSingleAud,
			wantResp:       `{"shelves":[{"id":"123","theme":"Shakspeare"},{"id":"124","theme":"Hamlet"}]}`,
		},
		{
			desc:           "Succeed for Http client, with valid JWT token, with single audience",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves",
			token:          testdata.FakeGoodTokenSingleAud,
			wantResp:       `{"shelves":[{"id":"123","theme":"Shakspeare"},{"id":"124","theme":"Hamlet"}]}`,
		},
		{
			desc:           "Fail for gPRC client, with JWT token but not expected audience",
			clientProtocol: "grpc",
			method:         "ListShelves",
			token:          testdata.FakeGoodToken,
			wantedError:    "code = Unauthenticated desc = Audiences in Jwt are not allowed",
		},
		{
			desc:           "Fail for Http client, with JWT token but not expected audience",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves",
			token:          testdata.FakeGoodToken,
			wantedError:    "401 Unauthorized",
		},
		{
			desc:           "Fail for gPRC client, with JWT token but wrong audience",
			clientProtocol: "grpc",
			method:         "ListShelves",
			token:          testdata.FakeGoodAdminToken,
			wantedError:    "code = Unauthenticated desc = Audiences in Jwt are not allowed",
		},
		{
			desc:           "Succeed for gPRC client, with JWT token with one audience while multi audiences are allowed",
			clientProtocol: "grpc",
			method:         "CreateBook",
			token:          testdata.FakeGoodAdminToken,
			wantResp:       `{"title":"New Book"}`,
		},
		{
			desc:           "Succeed for Http client, with JWT token with multi audience while multi audiences are allowed",
			clientProtocol: "http",
			httpMethod:     "POST",
			method:         "/v1/shelves/12345/books",
			token:          testdata.FakeGoodTokenMultiAud,
			wantResp:       `{"id":"12345","title":"New Book"}`,
		},
	}

	for _, tc := range tests {
		resp, err := client.MakeCall(tc.clientProtocol, addr, tc.httpMethod, tc.method, tc.token)

		if tc.wantedError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantedError)) {
			t.Errorf("Test (%s): failed, expected err: %v, got: %v", tc.desc, tc.wantedError, err)
		} else {
			if !strings.Contains(resp, tc.wantResp) {
				t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, resp)
			}
		}
	}
}
