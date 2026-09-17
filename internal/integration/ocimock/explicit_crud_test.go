/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package ocimock

import (
	"net/http"
	"testing"
)

func TestExplicitCRUDResponderUsesOnlyDeclaredTypedContract(t *testing.T) {
	t.Parallel()

	create := explicitFixture{Name: "created", Count: 1}
	created := crudTestState{ID: "thing-1", Name: "created"}
	update := explicitFixture{Name: "updated", Count: 2}
	updated := crudTestState{ID: "thing-1", Name: "updated"}
	responder, err := NewExplicitCRUDResponder(ExplicitCRUDOptions[crudTestState, explicitFixture, explicitFixture]{
		CollectionPath:    "/v1/things",
		ItemPath:          "/v1/things/thing-1",
		Operations:        []Operation{OperationCreate, OperationRead, OperationUpdate, OperationDelete},
		CreateRequest:     &create,
		CreatedState:      &created,
		UpdateRequest:     &update,
		UpdatedState:      &updated,
		ListShape:         ListShapeItems,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	requests := []Request{
		{Method: http.MethodPost, URL: mustTestURL(t, "https://mock.invalid/v1/things"), Body: []byte("{\"name\":\"created\",\"count\":1}")},
		{Method: http.MethodGet, URL: mustTestURL(t, "https://mock.invalid/v1/things/thing-1")},
		{Method: http.MethodPut, URL: mustTestURL(t, "https://mock.invalid/v1/things/thing-1"), Body: []byte("{\"name\":\"updated\",\"count\":2}")},
		{Method: http.MethodGet, URL: mustTestURL(t, "https://mock.invalid/v1/things/thing-1")},
		{Method: http.MethodDelete, URL: mustTestURL(t, "https://mock.invalid/v1/things/thing-1")},
		{Method: http.MethodGet, URL: mustTestURL(t, "https://mock.invalid/v1/things/thing-1")},
	}
	for _, request := range requests {
		if _, err := responder.Respond(request); err != nil {
			t.Fatal(err)
		}
	}
	if err := responder.Verify(); err != nil {
		t.Fatal(err)
	}
}

func TestExplicitCRUDResponderAdvancesDeclaredDeleteReadStates(t *testing.T) {
	t.Parallel()

	initial := crudTestState{ID: "thing-1", Name: "active"}
	deleting := crudTestState{ID: "thing-1", Name: "deleting"}
	responder, err := NewExplicitCRUDResponder(ExplicitCRUDOptions[crudTestState, struct{}, struct{}]{
		CollectionPath:     "/v1/things",
		ItemPath:           "/v1/things/thing-1",
		Operations:         []Operation{OperationRead, OperationDelete},
		InitialState:       &initial,
		ListShape:          ListShapeItems,
		DeletedReadStates:  []crudTestState{deleting},
		DeleteEndsNotFound: true,
		RequireDeleteRead:  true,
		NotFoundCode:       "NotFound",
	})
	if err != nil {
		t.Fatal(err)
	}
	deleteRequest := Request{Method: http.MethodDelete, URL: mustTestURL(t, "https://mock.invalid/v1/things/thing-1")}
	if _, err := responder.Respond(deleteRequest); err != nil {
		t.Fatal(err)
	}
	readRequest := Request{Method: http.MethodGet, URL: mustTestURL(t, "https://mock.invalid/v1/things/thing-1")}
	if response, err := responder.Respond(readRequest); err != nil || response.StatusCode != http.StatusOK {
		t.Fatalf("first delete read = status %d, error %v", response.StatusCode, err)
	}
	if response, err := responder.Respond(readRequest); err != nil || response.StatusCode != http.StatusNotFound {
		t.Fatalf("final delete read = status %d, error %v", response.StatusCode, err)
	}
	listRequest := Request{Method: http.MethodGet, URL: mustTestURL(t, "https://mock.invalid/v1/things")}
	if response, err := responder.Respond(listRequest); err != nil || string(response.Body) != "{\"items\":[]}" {
		t.Fatalf("post-delete list = body %s, error %v", response.Body, err)
	}
	if err := responder.Verify(); err != nil {
		t.Fatal(err)
	}
}

func TestExplicitCRUDResponderAdvancesDeclaredReadStatuses(t *testing.T) {
	t.Parallel()

	created := crudTestState{ID: "thing-1", Name: "created"}
	responder, err := NewExplicitCRUDResponder(ExplicitCRUDOptions[crudTestState, struct{}, struct{}]{
		CollectionPath:      "/v1/things",
		ItemPath:            "/v1/things/thing-1",
		Operations:          []Operation{OperationCreate, OperationRead},
		CreatedState:        &created,
		ValidateCreateRaw:   func(Request) error { return nil },
		CreatedReadStatuses: []int{http.StatusNotFound, http.StatusOK},
		NotFoundCode:        "NotFound",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := responder.Respond(Request{Method: http.MethodPost, URL: mustTestURL(t, "https://mock.invalid/v1/things")}); err != nil {
		t.Fatal(err)
	}
	read := Request{Method: http.MethodGet, URL: mustTestURL(t, "https://mock.invalid/v1/things/thing-1")}
	if response, err := responder.Respond(read); err != nil || response.StatusCode != http.StatusNotFound {
		t.Fatalf("first created read = status %d, error %v", response.StatusCode, err)
	}
	if response, err := responder.Respond(read); err != nil || response.StatusCode != http.StatusOK {
		t.Fatalf("second created read = status %d, error %v", response.StatusCode, err)
	}
}

func TestExplicitCRUDResponderAdvancesDeclaredDeleteStatuses(t *testing.T) {
	t.Parallel()

	initial := crudTestState{ID: "thing-1", Name: "active"}
	responder, err := NewExplicitCRUDResponder(ExplicitCRUDOptions[crudTestState, struct{}, struct{}]{
		CollectionPath: "/v1/things",
		ItemPath:       "/v1/things/thing-1",
		Operations:     []Operation{OperationRead, OperationDelete},
		InitialState:   &initial,
		DeleteStatuses: []int{http.StatusNoContent, http.StatusNotFound},
		NotFoundCode:   "NotFound",
	})
	if err != nil {
		t.Fatal(err)
	}
	request := Request{Method: http.MethodDelete, URL: mustTestURL(t, "https://mock.invalid/v1/things/thing-1")}
	if response, err := responder.Respond(request); err != nil || response.StatusCode != http.StatusNoContent {
		t.Fatalf("first delete = status %d, error %v", response.StatusCode, err)
	}
	if response, err := responder.Respond(request); err != nil || response.StatusCode != http.StatusNotFound {
		t.Fatalf("second delete = status %d, error %v", response.StatusCode, err)
	}
}

func TestExplicitCRUDResponderAddsDeclaredOperationHeaders(t *testing.T) {
	t.Parallel()
	created := crudTestState{ID: "thing-1", Name: "created"}
	updated := crudTestState{ID: "thing-1", Name: "updated"}
	responder, err := NewExplicitCRUDResponder(ExplicitCRUDOptions[crudTestState, struct{}, struct{}]{
		CollectionPath: "/v1/things", ItemPath: "/v1/things/thing-1",
		Operations:   []Operation{OperationCreate, OperationUpdate, OperationDelete},
		CreatedState: &created, UpdatedState: &updated,
		ValidateCreateRaw: func(Request) error { return nil }, ValidateUpdateRaw: func(Request) error { return nil },
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"create-wr"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"update-wr"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"delete-wr"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	requests := []struct {
		request Request
		want    string
	}{
		{request: Request{Method: http.MethodPost, URL: mustTestURL(t, "https://mock.invalid/v1/things")}, want: "create-wr"},
		{request: Request{Method: http.MethodPut, URL: mustTestURL(t, "https://mock.invalid/v1/things/thing-1")}, want: "update-wr"},
		{request: Request{Method: http.MethodDelete, URL: mustTestURL(t, "https://mock.invalid/v1/things/thing-1")}, want: "delete-wr"},
	}
	for _, test := range requests {
		response, err := responder.Respond(test.request)
		if err != nil {
			t.Fatal(err)
		}
		if got := response.Header.Get("Opc-Work-Request-Id"); got != test.want {
			t.Fatalf("operation header = %q, want %q", got, test.want)
		}
	}
}
