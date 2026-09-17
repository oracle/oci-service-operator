/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

// Package ocimock provides a credential-free HTTP boundary for service-manager
// integration tests. The production OCI SDK still builds and decodes every
// request and response; only the network dispatcher is replaced.
package ocimock

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"

	"github.com/oracle/oci-go-sdk/v65/common"
)

const defaultMaxBodyBytes int64 = 4 << 20

// Request is the immutable HTTP request observed after OCI SDK serialization.
type Request struct {
	Method string
	URL    *url.URL
	Header http.Header
	Body   []byte
}

// Response is an OCI-compatible HTTP response returned to the real OCI SDK.
type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

// Responder owns the mock cloud behavior and verifies its expected operations.
type Responder interface {
	Respond(Request) (Response, error)
	Verify() error
}

// Options configures one OCI SDK mock session.
type Options struct {
	Host         string
	BasePath     string
	Responder    Responder
	MaxBodyBytes int64
}

// Session owns a credential-free OCI SDK base client and its mock transport.
type Session struct {
	baseClient common.BaseClient
	transport  *transport
}

type unsignedSigner struct{}

func (unsignedSigner) Sign(*http.Request) error { return nil }

type transport struct {
	mu           sync.Mutex
	responder    Responder
	maxBodyBytes int64
	requests     []Request
	closed       bool
}

// Open creates a mock session that cannot perform network I/O.
func Open(options Options) (*Session, error) {
	host, err := normalizeHost(options.Host)
	if err != nil {
		return nil, err
	}
	if options.Responder == nil {
		return nil, errors.New("OCI mock responder is required")
	}
	maxBodyBytes := options.MaxBodyBytes
	if maxBodyBytes <= 0 {
		maxBodyBytes = defaultMaxBodyBytes
	}
	dispatcher := &transport{responder: options.Responder, maxBodyBytes: maxBodyBytes}
	baseClient := common.DefaultBaseClientWithSigner(unsignedSigner{})
	baseClient.Host = host
	baseClient.BasePath = strings.Trim(options.BasePath, "/")
	noRetry := common.NoRetryPolicy()
	baseClient.Configuration.RetryPolicy = &noRetry
	baseClient.HTTPClient = dispatcher
	return &Session{baseClient: baseClient, transport: dispatcher}, nil
}

// BaseClient returns a copy of the OCI SDK base client attached to the mock.
func (s *Session) BaseClient() common.BaseClient {
	if s == nil {
		return common.BaseClient{}
	}
	return s.baseClient
}

// BaseClientFor attaches another OCI endpoint to the same mock state.
func (s *Session) BaseClientFor(host string, basePath string) (common.BaseClient, error) {
	if s == nil || s.transport == nil {
		return common.BaseClient{}, errors.New("OCI mock session is not initialized")
	}
	normalized, err := normalizeHost(host)
	if err != nil {
		return common.BaseClient{}, err
	}
	baseClient := s.baseClient
	baseClient.Host = normalized
	baseClient.BasePath = strings.Trim(basePath, "/")
	baseClient.HTTPClient = s.transport
	return baseClient, nil
}

// Requests returns defensive copies of all HTTP requests observed so far.
func (s *Session) Requests() []Request {
	if s == nil || s.transport == nil {
		return nil
	}
	return s.transport.Requests()
}

// Close verifies the responder contract. It is safe to call more than once.
func (s *Session) Close() error {
	if s == nil || s.transport == nil {
		return nil
	}
	return s.transport.Close()
}

func (t *transport) Do(request *http.Request) (*http.Response, error) {
	if request == nil {
		return nil, errors.New("OCI mock received a nil request")
	}
	body, err := readAndRestoreBody(&request.Body, t.maxBodyBytes)
	if err != nil {
		return nil, fmt.Errorf("read OCI mock request body: %w", err)
	}
	observed := Request{
		Method: request.Method,
		URL:    cloneURL(request.URL),
		Header: request.Header.Clone(),
		Body:   append([]byte(nil), body...),
	}

	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil, errors.New("OCI mock transport is closed")
	}
	t.requests = append(t.requests, observed)
	t.mu.Unlock()

	configured, err := t.responder.Respond(observed)
	if err != nil {
		return nil, err
	}
	statusCode := configured.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	headers := configured.Header.Clone()
	if headers == nil {
		headers = make(http.Header)
	}
	if len(configured.Body) > 0 && headers.Get("Content-Type") == "" {
		headers.Set("Content-Type", "application/json")
	}
	return &http.Response{
		StatusCode: statusCode,
		Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
		Header:     headers,
		Body:       io.NopCloser(bytes.NewReader(configured.Body)),
		Request:    request,
	}, nil
}

func (t *transport) Requests() []Request {
	t.mu.Lock()
	defer t.mu.Unlock()
	result := make([]Request, len(t.requests))
	for index, request := range t.requests {
		result[index] = cloneRequest(request)
	}
	return result
}

func (t *transport) Close() error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil
	}
	t.closed = true
	t.mu.Unlock()
	return t.responder.Verify()
}

// JSONResponse serializes an OCI SDK model as an HTTP JSON response.
func JSONResponse(statusCode int, value any) (Response, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return Response{}, fmt.Errorf("marshal OCI mock response: %w", err)
	}
	return Response{StatusCode: statusCode, Body: body}, nil
}

// EmptyResponse creates an OCI response without a body.
func EmptyResponse(statusCode int) Response {
	return Response{StatusCode: statusCode}
}

// DecodeJSONRequest decodes an SDK-generated request body into its typed SDK
// details model and rejects trailing or unknown fields.
func DecodeJSONRequest(request Request, target any) error {
	if target == nil {
		return errors.New("OCI mock request target is nil")
	}
	decoder := json.NewDecoder(bytes.NewReader(request.Body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode %s %s request body: %w", request.Method, request.URL.Path, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("decode %s %s request body: trailing JSON value", request.Method, request.URL.Path)
		}
		return fmt.Errorf("decode %s %s request body trailer: %w", request.Method, request.URL.Path, err)
	}
	return nil
}

// DecodeDiscriminatedJSONRequest validates and removes one SDK polymorphic
// discriminator before strictly decoding the remaining fields into a concrete
// SDK details model.
func DecodeDiscriminatedJSONRequest(request Request, target any, field, want string) error {
	field = strings.TrimSpace(field)
	if field == "" {
		return errors.New("OCI mock discriminator field is empty")
	}
	decoder := json.NewDecoder(bytes.NewReader(request.Body))
	var values map[string]json.RawMessage
	if err := decoder.Decode(&values); err != nil {
		return fmt.Errorf("decode %s %s discriminated request body: %w", request.Method, request.URL.Path, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("decode %s %s discriminated request body: trailing JSON value", request.Method, request.URL.Path)
		}
		return fmt.Errorf("decode %s %s discriminated request body trailer: %w", request.Method, request.URL.Path, err)
	}
	raw, exists := values[field]
	if !exists {
		return fmt.Errorf("decode %s %s discriminated request body: missing %s", request.Method, request.URL.Path, field)
	}
	var got string
	if err := json.Unmarshal(raw, &got); err != nil {
		return fmt.Errorf("decode %s %s discriminator %s: %w", request.Method, request.URL.Path, field, err)
	}
	if got != want {
		return fmt.Errorf("decode %s %s discriminator %s = %q, want %q", request.Method, request.URL.Path, field, got, want)
	}
	delete(values, field)
	body, err := json.Marshal(values)
	if err != nil {
		return fmt.Errorf("normalize %s %s discriminated request body: %w", request.Method, request.URL.Path, err)
	}
	request.Body = body
	return DecodeJSONRequest(request, target)
}

// ValidateMandatoryFields checks the mandatory tags on an OCI SDK details
// struct. Cross-field service rules remain explicit resource-test assertions.
func ValidateMandatoryFields(value any) error {
	current := reflect.ValueOf(value)
	for current.IsValid() && (current.Kind() == reflect.Pointer || current.Kind() == reflect.Interface) {
		if current.IsNil() {
			return errors.New("OCI SDK request details are nil")
		}
		current = current.Elem()
	}
	if !current.IsValid() || current.Kind() != reflect.Struct {
		return fmt.Errorf("OCI SDK request details must be a struct, got %T", value)
	}
	typeInfo := current.Type()
	var missing []string
	for index := 0; index < current.NumField(); index++ {
		fieldInfo := typeInfo.Field(index)
		if fieldInfo.Tag.Get("mandatory") != "true" {
			continue
		}
		if mandatoryValueMissing(current.Field(index)) {
			name := strings.Split(fieldInfo.Tag.Get("json"), ",")[0]
			if name == "" || name == "-" {
				name = fieldInfo.Name
			}
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("OCI SDK request is missing mandatory field(s): %s", strings.Join(missing, ", "))
	}
	return nil
}

func mandatoryValueMissing(value reflect.Value) bool {
	if !value.IsValid() {
		return true
	}
	switch value.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice:
		return value.IsNil()
	case reflect.String:
		return strings.TrimSpace(value.String()) == ""
	default:
		return false
	}
}

func readAndRestoreBody(body *io.ReadCloser, maxBytes int64) ([]byte, error) {
	if body == nil || *body == nil {
		return nil, nil
	}
	limited := io.LimitReader(*body, maxBytes+1)
	content, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > maxBytes {
		return nil, fmt.Errorf("request body exceeds %d bytes", maxBytes)
	}
	if err := (*body).Close(); err != nil {
		return nil, err
	}
	*body = io.NopCloser(bytes.NewReader(content))
	return content, nil
}

func normalizeHost(value string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("OCI mock host %q must be an absolute HTTP(S) URL", value)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("OCI mock host %q must use HTTP or HTTPS", value)
	}
	if (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("OCI mock host %q must not contain a path, query, or fragment", value)
	}
	return parsed.Scheme + "://" + parsed.Host, nil
}

func cloneURL(source *url.URL) *url.URL {
	if source == nil {
		return &url.URL{}
	}
	result := *source
	return &result
}

func cloneRequest(source Request) Request {
	return Request{
		Method: source.Method,
		URL:    cloneURL(source.URL),
		Header: source.Header.Clone(),
		Body:   append([]byte(nil), source.Body...),
	}
}
