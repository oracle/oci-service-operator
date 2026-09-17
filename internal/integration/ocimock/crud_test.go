/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package ocimock

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

type crudTestState struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func TestCRUDResponderMaintainsDynamicState(t *testing.T) {
	t.Parallel()

	responder, err := NewCRUDResponder(CRUDOptions[crudTestState]{
		CollectionPath:     "/v1/things",
		ItemPath:           "/v1/things/thing-1",
		ExpectedOperations: []Operation{OperationCreate, OperationRead, OperationUpdate, OperationDelete},
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		List: func(_ Request, present bool, state crudTestState) (Response, error) {
			if !present {
				return JSONResponse(http.StatusOK, []crudTestState{})
			}
			return JSONResponse(http.StatusOK, []crudTestState{state})
		},
		Create: func(request Request) (crudTestState, Response, error) {
			var body struct {
				Name string `json:"name"`
			}
			if err := json.Unmarshal(request.Body, &body); err != nil {
				return crudTestState{}, Response{}, err
			}
			state := crudTestState{ID: "thing-1", Name: body.Name}
			response, err := JSONResponse(http.StatusCreated, state)
			return state, response, err
		},
		Read: func(_ Request, state crudTestState) (Response, error) {
			return JSONResponse(http.StatusOK, state)
		},
		Update: func(request Request, state crudTestState) (crudTestState, Response, error) {
			var body struct {
				Name string `json:"name"`
			}
			if err := json.Unmarshal(request.Body, &body); err != nil {
				return crudTestState{}, Response{}, err
			}
			state.Name = body.Name
			response, err := JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Delete: func(_ Request, _ crudTestState) (Response, error) {
			return EmptyResponse(http.StatusNoContent), nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	requests := []Request{
		{Method: http.MethodGet, URL: mustTestURL(t, "https://mock.invalid/v1/things")},
		{Method: http.MethodPost, URL: mustTestURL(t, "https://mock.invalid/v1/things"), Body: []byte(`{"name":"created"}`)},
		{Method: http.MethodGet, URL: mustTestURL(t, "https://mock.invalid/v1/things/thing-1")},
		{Method: http.MethodPut, URL: mustTestURL(t, "https://mock.invalid/v1/things/thing-1"), Body: []byte(`{"name":"updated"}`)},
		{Method: http.MethodGet, URL: mustTestURL(t, "https://mock.invalid/v1/things/thing-1")},
		{Method: http.MethodDelete, URL: mustTestURL(t, "https://mock.invalid/v1/things/thing-1")},
		{Method: http.MethodGet, URL: mustTestURL(t, "https://mock.invalid/v1/things/thing-1")},
	}
	for index, request := range requests {
		response, err := responder.Respond(request)
		if err != nil {
			t.Fatalf("request %d: %v", index, err)
		}
		if index == len(requests)-1 && response.StatusCode != http.StatusNotFound {
			t.Fatalf("post-delete status = %d", response.StatusCode)
		}
	}
	if err := responder.Verify(); err != nil {
		t.Fatal(err)
	}
	state, present := responder.CurrentState()
	if present || state.Name != "updated" {
		t.Fatalf("state = %+v present=%t", state, present)
	}
}

func TestCRUDResponderVerificationFindsMissingOperations(t *testing.T) {
	t.Parallel()

	responder, err := NewCRUDResponder(CRUDOptions[crudTestState]{
		CollectionPath:     "/v1/things",
		ItemPath:           "/v1/things/thing-1",
		ExpectedOperations: []Operation{OperationCreate, OperationRead},
		List: func(_ Request, _ bool, _ crudTestState) (Response, error) {
			return JSONResponse(http.StatusOK, []crudTestState{})
		},
		Create: func(_ Request) (crudTestState, Response, error) {
			state := crudTestState{ID: "thing-1"}
			response, err := JSONResponse(http.StatusCreated, state)
			return state, response, err
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := responder.Respond(Request{Method: http.MethodGet, URL: mustTestURL(t, "https://mock.invalid/v1/things")}); err != nil {
		t.Fatal(err)
	}
	if err := responder.Verify(); err == nil {
		t.Fatal("Verify() error = nil, want missing create")
	}
}

func TestCRUDResponderRejectsExpectedOperationWithoutHandler(t *testing.T) {
	t.Parallel()

	_, err := NewCRUDResponder(CRUDOptions[crudTestState]{
		CollectionPath:     "/v1/things",
		ItemPath:           "/v1/things/thing-1",
		ExpectedOperations: []Operation{OperationUpdate},
	})
	if err == nil {
		t.Fatal("NewCRUDResponder() error = nil, want missing update handler")
	}
}

func TestCRUDResponderVerifiesExplicitAuxiliaryRoute(t *testing.T) {
	t.Parallel()

	responder, err := NewCRUDResponder(CRUDOptions[crudTestState]{
		CollectionPath: "/v1/things",
		ItemPath:       "/v1/things/thing-1",
		AdditionalRoutes: []Route{{
			Name:         "supporting lookup",
			Method:       http.MethodGet,
			Path:         "/v1/supporting",
			MinimumCalls: 1,
			Respond: func(Request) (Response, error) {
				return JSONResponse(http.StatusOK, map[string]any{"items": []any{}})
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := responder.Verify(); err == nil {
		t.Fatal("Verify() error = nil before required auxiliary call")
	}
	if _, err := responder.Respond(Request{Method: http.MethodGet, URL: mustTestURL(t, "https://mock.invalid/v1/supporting")}); err != nil {
		t.Fatal(err)
	}
	if err := responder.Verify(); err != nil {
		t.Fatal(err)
	}
}

func TestCRUDResponderAcceptsPostUpdateOnItemPath(t *testing.T) {
	t.Parallel()

	initial := crudTestState{ID: "thing-1", Name: "created"}
	responder, err := NewCRUDResponder(CRUDOptions[crudTestState]{
		CollectionPath:     "/v1/things",
		ItemPath:           "/v1/things/thing-1",
		InitialState:       &initial,
		ExpectedOperations: []Operation{OperationUpdate},
		Update: func(request Request, state crudTestState) (crudTestState, Response, error) {
			if request.Method != http.MethodPost {
				return crudTestState{}, Response{}, fmt.Errorf("update method = %s", request.Method)
			}
			state.Name = "updated"
			response, err := JSONResponse(http.StatusOK, state)
			return state, response, err
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := responder.Respond(Request{
		Method: http.MethodPost,
		URL:    mustTestURL(t, "https://mock.invalid/v1/things/thing-1"),
	}); err != nil {
		t.Fatal(err)
	}
	if err := responder.Verify(); err != nil {
		t.Fatal(err)
	}
}

func TestCRUDResponderSupportsPutCreateAndUpdateOnItemPath(t *testing.T) {
	t.Parallel()

	responder, err := NewCRUDResponder(CRUDOptions[crudTestState]{
		CollectionPath:     "/widgets",
		ItemPath:           "/widgets/widget-1",
		CreatePath:         "/widgets/widget-1",
		CreateMethod:       http.MethodPut,
		ExpectedOperations: []Operation{OperationCreate, OperationUpdate},
		Create: func(Request) (crudTestState, Response, error) {
			state := crudTestState{Name: "created"}
			response, err := JSONResponse(http.StatusCreated, state)
			return state, response, err
		},
		Update: func(Request, crudTestState) (crudTestState, Response, error) {
			state := crudTestState{Name: "updated"}
			response, err := JSONResponse(http.StatusOK, state)
			return state, response, err
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, request := range []Request{
		{Method: http.MethodPut, URL: mustTestURL(t, "https://example.test/widgets/widget-1")},
		{Method: http.MethodPut, URL: mustTestURL(t, "https://example.test/widgets/widget-1")},
	} {
		if _, err := responder.Respond(request); err != nil {
			t.Fatal(err)
		}
	}
	if err := responder.Verify(); err != nil {
		t.Fatal(err)
	}
}

func TestCRUDResponderSupportsActionDeletePath(t *testing.T) {
	t.Parallel()

	initial := crudTestState{ID: "thing-1", Name: "created"}
	responder, err := NewCRUDResponder(CRUDOptions[crudTestState]{
		CollectionPath:     "/v1/things",
		ItemPath:           "/v1/things/thing-1",
		DeletePath:         "/v1/things/thing-1/actions/cancel",
		DeleteMethod:       http.MethodPost,
		InitialState:       &initial,
		ExpectedOperations: []Operation{OperationDelete},
		Delete: func(request Request, _ crudTestState) (Response, error) {
			if request.Method != http.MethodPost {
				t.Fatalf("delete method = %s, want POST", request.Method)
			}
			return EmptyResponse(http.StatusAccepted), nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := responder.Respond(Request{
		Method: http.MethodPost,
		URL:    mustTestURL(t, "https://mock.invalid/v1/things/thing-1/actions/cancel"),
	}); err != nil {
		t.Fatal(err)
	}
	if err := responder.Verify(); err != nil {
		t.Fatal(err)
	}
}

func TestCRUDResponderSupportsLifecycleReadTransitions(t *testing.T) {
	t.Parallel()

	responder, err := NewCRUDResponder(CRUDOptions[crudTestState]{
		CollectionPath:         "/v1/things",
		ItemPath:               "/v1/things/thing-1",
		RequireCreateRead:      true,
		RequireDeleteRead:      true,
		RetainStateAfterDelete: true,
		Create: func(_ Request) (crudTestState, Response, error) {
			state := crudTestState{ID: "thing-1", Name: "CREATING"}
			response, err := JSONResponse(http.StatusAccepted, state)
			return state, response, err
		},
		ReadTransition: func(_ Request, state crudTestState) (crudTestState, Response, error) {
			switch state.Name {
			case "CREATING":
				state.Name = "ACTIVE"
			case "DELETING":
				state.Name = "DELETED"
			}
			response, err := JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ Request, state crudTestState) (crudTestState, Response, error) {
			state.Name = "DELETING"
			return state, EmptyResponse(http.StatusAccepted), nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := responder.Respond(Request{Method: http.MethodPost, URL: mustTestURL(t, "https://mock.invalid/v1/things")}); err != nil {
		t.Fatal(err)
	}
	if _, err := responder.Respond(Request{Method: http.MethodGet, URL: mustTestURL(t, "https://mock.invalid/v1/things/thing-1")}); err != nil {
		t.Fatal(err)
	}
	if state, _ := responder.CurrentState(); state.Name != "ACTIVE" {
		t.Fatalf("created state = %+v", state)
	}
	if _, err := responder.Respond(Request{Method: http.MethodDelete, URL: mustTestURL(t, "https://mock.invalid/v1/things/thing-1")}); err != nil {
		t.Fatal(err)
	}
	if _, err := responder.Respond(Request{Method: http.MethodGet, URL: mustTestURL(t, "https://mock.invalid/v1/things/thing-1")}); err != nil {
		t.Fatal(err)
	}
	if err := responder.Verify(); err != nil {
		t.Fatal(err)
	}
}

func TestCRUDResponderRejectsReadAndReadTransitionTogether(t *testing.T) {
	t.Parallel()
	_, err := NewCRUDResponder(CRUDOptions[crudTestState]{
		CollectionPath: "/v1/things",
		ItemPath:       "/v1/things/thing-1",
		Read: func(_ Request, _ crudTestState) (Response, error) {
			return EmptyResponse(http.StatusOK), nil
		},
		ReadTransition: func(_ Request, state crudTestState) (crudTestState, Response, error) {
			return state, EmptyResponse(http.StatusOK), nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "read, readTransition, or readByPhase") {
		t.Fatalf("NewCRUDResponder() error = %v", err)
	}
}

func TestCRUDResponderRejectsDeleteAndDeleteTransitionTogether(t *testing.T) {
	t.Parallel()
	_, err := NewCRUDResponder(CRUDOptions[crudTestState]{
		CollectionPath: "/v1/things",
		ItemPath:       "/v1/things/thing-1",
		Delete: func(_ Request, _ crudTestState) (Response, error) {
			return EmptyResponse(http.StatusNoContent), nil
		},
		DeleteTransition: func(_ Request, state crudTestState) (crudTestState, Response, error) {
			return state, EmptyResponse(http.StatusAccepted), nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "delete or deleteTransition") {
		t.Fatalf("NewCRUDResponder() error = %v", err)
	}
}

func TestCRUDResponderVerificationRequiresConfiguredReadbacks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		configure   func(*CRUDResponder[crudTestState])
		wantMessage string
	}{
		{name: "create", configure: func(current *CRUDResponder[crudTestState]) { current.options.RequireCreateRead = true }, wantMessage: "after create"},
		{name: "update", configure: func(current *CRUDResponder[crudTestState]) { current.options.RequireUpdateRead = true }, wantMessage: "after update"},
		{name: "delete", configure: func(current *CRUDResponder[crudTestState]) { current.options.RequireDeleteRead = true }, wantMessage: "confirm deletion"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			responder := &CRUDResponder[crudTestState]{
				options:    CRUDOptions[crudTestState]{ExpectedOperations: []Operation{OperationCreate, OperationRead, OperationUpdate, OperationDelete}},
				operations: map[Operation]int{OperationCreate: 1, OperationRead: 1, OperationUpdate: 1, OperationDelete: 1},
			}
			test.configure(responder)
			if err := responder.Verify(); err == nil || !strings.Contains(err.Error(), test.wantMessage) {
				t.Fatalf("Verify() error = %v", err)
			}
		})
	}
}

func TestCRUDResponderSupportsSingletonCRUDPathWithoutList(t *testing.T) {
	t.Parallel()
	created := crudTestState{ID: "singleton", Name: "created"}
	responder, err := NewCRUDResponder(CRUDOptions[crudTestState]{
		CollectionPath: "/v1/singleton", ItemPath: "/v1/singleton",
		ExpectedOperations: []Operation{OperationCreate, OperationRead, OperationDelete},
		Create: func(Request) (crudTestState, Response, error) {
			response, err := JSONResponse(http.StatusCreated, created)
			return created, response, err
		},
		Read:   func(_ Request, state crudTestState) (Response, error) { return JSONResponse(http.StatusOK, state) },
		Delete: func(Request, crudTestState) (Response, error) { return EmptyResponse(http.StatusNoContent), nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	requests := []Request{
		{Method: http.MethodPost, URL: mustTestURL(t, "https://mock.invalid/v1/singleton")},
		{Method: http.MethodGet, URL: mustTestURL(t, "https://mock.invalid/v1/singleton")},
		{Method: http.MethodDelete, URL: mustTestURL(t, "https://mock.invalid/v1/singleton")},
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
