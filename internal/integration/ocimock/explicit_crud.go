/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package ocimock

import (
	"fmt"
	"net/http"
)

// ListShape declares the SDK response envelope used by a collection read.
type ListShape string

const (
	ListShapeNone  ListShape = ""
	ListShapeArray ListShape = "array"
	ListShapeItems ListShape = "items"
)

// ExplicitCRUDOptions contains only package-owned, typed test inputs. It does
// not inspect formal metadata, CR fields, or SDK types.
type ExplicitCRUDOptions[S, C, U any] struct {
	CollectionPath string
	ItemPath       string
	CreatePath     string
	CreateMethod   string
	UpdatePath     string
	UpdateMethod   string
	DeletePath     string
	DeleteMethod   string
	Operations     []Operation

	CreateRequest       *C
	CreatedState        *S
	UpdateRequest       *U
	UpdatedState        *S
	DeletedState        *S
	InitialState        *S
	CreatedReadStates   []S
	UpdatedReadStates   []S
	DeletedReadStates   []S
	CreatedReadStatuses []int
	UpdatedReadStatuses []int
	DeletedReadStatuses []int
	DeleteEndsNotFound  bool

	ListShape              ListShape
	RequireCreateRead      bool
	RequireUpdateRead      bool
	RequireDeleteRead      bool
	RetainStateAfterDelete bool

	CreateStatus   int
	ReadStatus     int
	UpdateStatus   int
	DeleteStatus   int
	DeleteStatuses []int
	NotFoundCode   string
	CreateHeaders  http.Header
	UpdateHeaders  http.Header
	DeleteHeaders  http.Header

	ValidateCreate    func(Request, C) error
	CompareCreate     func(C, C) error
	ValidateCreateRaw func(Request) error
	ValidateRead      func(Request, S) error
	ValidateList      func(Request, bool, S) error
	ValidateUpdate    func(Request, U) error
	CompareUpdate     func(U, U) error
	ValidateUpdateRaw func(Request) error
	ValidateDelete    func(Request, S) error
	AdditionalRoutes  []Route
}

type explicitItemsResponse[S any] struct {
	Items []S `json:"items"`
}

// NewExplicitCRUDResponder builds a stateful responder from the typed contract
// declared in one resource's package-local integration test.
func NewExplicitCRUDResponder[S, C, U any](options ExplicitCRUDOptions[S, C, U]) (*CRUDResponder[S], error) {
	crudOptions := CRUDOptions[S]{
		CollectionPath:         options.CollectionPath,
		ItemPath:               options.ItemPath,
		CreatePath:             options.CreatePath,
		CreateMethod:           options.CreateMethod,
		UpdatePath:             options.UpdatePath,
		UpdateMethod:           options.UpdateMethod,
		DeletePath:             options.DeletePath,
		DeleteMethod:           options.DeleteMethod,
		ExpectedOperations:     append([]Operation(nil), options.Operations...),
		RequireCreateRead:      options.RequireCreateRead,
		RequireUpdateRead:      options.RequireUpdateRead,
		RequireDeleteRead:      options.RequireDeleteRead,
		RetainStateAfterDelete: options.RetainStateAfterDelete,
		InitialState:           options.InitialState,
		AdditionalRoutes:       append([]Route(nil), options.AdditionalRoutes...),
	}

	if options.ListShape != ListShapeNone {
		crudOptions.List = func(request Request, present bool, state S) (Response, error) {
			if options.ValidateList != nil {
				if err := options.ValidateList(request, present, state); err != nil {
					return Response{}, err
				}
			}
			switch options.ListShape {
			case ListShapeArray:
				if !present {
					return JSONResponse(statusOrDefault(options.ReadStatus, http.StatusOK), []S{})
				}
				return JSONResponse(statusOrDefault(options.ReadStatus, http.StatusOK), []S{state})
			case ListShapeItems:
				items := []S{}
				if present {
					items = append(items, state)
				}
				return JSONResponse(statusOrDefault(options.ReadStatus, http.StatusOK), explicitItemsResponse[S]{Items: items})
			default:
				return Response{}, fmt.Errorf("explicit OCI mock list shape %q is unsupported", options.ListShape)
			}
		}
	}
	if options.CreateRequest != nil || options.ValidateCreateRaw != nil {
		crudOptions.Create = func(request Request) (S, Response, error) {
			var zero S
			if options.ValidateCreateRaw != nil {
				if err := options.ValidateCreateRaw(request); err != nil {
					return zero, Response{}, err
				}
			} else {
				actual, err := explicitRequestDetails(request, *options.CreateRequest, options.CompareCreate)
				if err != nil {
					return zero, Response{}, err
				}
				if options.ValidateCreate != nil {
					if err := options.ValidateCreate(request, actual); err != nil {
						return zero, Response{}, err
					}
				}
			}
			if options.CreatedState == nil {
				return zero, Response{}, fmt.Errorf("explicit OCI mock create state is required")
			}
			state := *options.CreatedState
			response, err := JSONResponse(statusOrDefault(options.CreateStatus, http.StatusOK), state)
			mergeResponseHeaders(&response, options.CreateHeaders)
			return state, response, err
		}
	}
	if len(options.CreatedReadStates)+len(options.UpdatedReadStates)+len(options.DeletedReadStates)+
		len(options.CreatedReadStatuses)+len(options.UpdatedReadStatuses)+len(options.DeletedReadStatuses) > 0 || options.DeleteEndsNotFound {
		readIndexes := map[ReadPhase]int{}
		crudOptions.ReadByPhase = func(request Request, phase ReadPhase, state S) (S, Response, error) {
			if options.ValidateRead != nil {
				if err := options.ValidateRead(request, state); err != nil {
					var zero S
					return zero, Response{}, err
				}
			}
			states := explicitReadStates(options, phase)
			index := readIndexes[phase]
			readIndexes[phase]++
			if phase == ReadPhaseDeleted && index >= len(states) && options.DeleteEndsNotFound {
				response, err := explicitNotFoundResponse(options.NotFoundCode)
				return state, response, err
			}
			status := explicitReadStatus(options, phase, index)
			if status == http.StatusNotFound {
				response, err := explicitNotFoundResponse(options.NotFoundCode)
				return state, response, err
			}
			if len(states) > 0 {
				if index >= len(states) {
					index = len(states) - 1
				}
				state = states[index]
			}
			response, err := JSONResponse(status, state)
			return state, response, err
		}
	} else {
		crudOptions.Read = func(request Request, state S) (Response, error) {
			if options.ValidateRead != nil {
				if err := options.ValidateRead(request, state); err != nil {
					return Response{}, err
				}
			}
			return JSONResponse(statusOrDefault(options.ReadStatus, http.StatusOK), state)
		}
	}
	if options.UpdateRequest != nil || options.ValidateUpdateRaw != nil {
		crudOptions.Update = func(request Request, _ S) (S, Response, error) {
			var zero S
			if options.ValidateUpdateRaw != nil {
				if err := options.ValidateUpdateRaw(request); err != nil {
					return zero, Response{}, err
				}
			} else {
				actual, err := explicitRequestDetails(request, *options.UpdateRequest, options.CompareUpdate)
				if err != nil {
					return zero, Response{}, err
				}
				if options.ValidateUpdate != nil {
					if err := options.ValidateUpdate(request, actual); err != nil {
						return zero, Response{}, err
					}
				}
			}
			if options.UpdatedState == nil {
				return zero, Response{}, fmt.Errorf("explicit OCI mock updated state is required")
			}
			state := *options.UpdatedState
			response, err := JSONResponse(statusOrDefault(options.UpdateStatus, http.StatusOK), state)
			mergeResponseHeaders(&response, options.UpdateHeaders)
			return state, response, err
		}
	}
	if operationDeclared(options.Operations, OperationDelete) {
		deleteIndex := 0
		deleteHandler := func(request Request, state S) (Response, error) {
			if options.ValidateDelete != nil {
				if err := options.ValidateDelete(request, state); err != nil {
					return Response{}, err
				}
			}
			status := statusOrDefault(options.DeleteStatus, http.StatusNoContent)
			if len(options.DeleteStatuses) > 0 {
				index := deleteIndex
				if index >= len(options.DeleteStatuses) {
					index = len(options.DeleteStatuses) - 1
				}
				status = options.DeleteStatuses[index]
				deleteIndex++
			}
			if status == http.StatusNotFound {
				return explicitNotFoundResponse(options.NotFoundCode)
			}
			response := EmptyResponse(status)
			mergeResponseHeaders(&response, options.DeleteHeaders)
			return response, nil
		}
		if options.DeletedState == nil && len(options.DeletedReadStates) == 0 {
			crudOptions.Delete = deleteHandler
		} else {
			crudOptions.RetainStateAfterDelete = true
			crudOptions.DeleteTransition = func(request Request, state S) (S, Response, error) {
				response, err := deleteHandler(request, state)
				if err != nil {
					var zero S
					return zero, Response{}, err
				}
				if options.DeletedState != nil {
					state = *options.DeletedState
				}
				return state, response, nil
			}
		}
	}
	if options.NotFoundCode != "" {
		crudOptions.NotFound = func(Request) (Response, error) {
			return explicitNotFoundResponse(options.NotFoundCode)
		}
	}
	return NewCRUDResponder(crudOptions)
}

func explicitReadStatus[S, C, U any](options ExplicitCRUDOptions[S, C, U], phase ReadPhase, index int) int {
	var statuses []int
	switch phase {
	case ReadPhaseCreated:
		statuses = options.CreatedReadStatuses
	case ReadPhaseUpdated:
		statuses = options.UpdatedReadStatuses
	case ReadPhaseDeleted:
		statuses = options.DeletedReadStatuses
	}
	if len(statuses) == 0 {
		return statusOrDefault(options.ReadStatus, http.StatusOK)
	}
	if index >= len(statuses) {
		index = len(statuses) - 1
	}
	return statusOrDefault(statuses[index], http.StatusOK)
}

func mergeResponseHeaders(response *Response, headers http.Header) {
	if response == nil || len(headers) == 0 {
		return
	}
	if response.Header == nil {
		response.Header = make(http.Header)
	}
	for name, values := range headers {
		response.Header[name] = append([]string(nil), values...)
	}
}

func explicitRequestDetails[T any](request Request, expected T, compare func(T, T) error) (T, error) {
	var actual T
	if err := DecodeJSONRequest(request, &actual); err != nil {
		return actual, err
	}
	if compare != nil {
		if err := compare(actual, expected); err != nil {
			return actual, err
		}
		return actual, nil
	}
	if err := ValidateJSONRequest(request, expected); err != nil {
		return actual, err
	}
	return actual, nil
}

func explicitReadStates[S, C, U any](options ExplicitCRUDOptions[S, C, U], phase ReadPhase) []S {
	switch phase {
	case ReadPhaseCreated:
		return options.CreatedReadStates
	case ReadPhaseUpdated:
		return options.UpdatedReadStates
	case ReadPhaseDeleted:
		if len(options.DeletedReadStates) > 0 {
			return options.DeletedReadStates
		}
		if options.DeletedState != nil {
			return []S{*options.DeletedState}
		}
	}
	return nil
}

func explicitNotFoundResponse(code string) (Response, error) {
	if code == "" {
		code = "NotAuthorizedOrNotFound"
	}
	return JSONResponse(http.StatusNotFound, map[string]string{
		"code":    code,
		"message": "resource not found",
	})
}

func statusOrDefault(value, fallback int) int {
	if value == 0 {
		return fallback
	}
	return value
}

func operationDeclared(operations []Operation, target Operation) bool {
	for _, operation := range operations {
		if operation == target {
			return true
		}
	}
	return false
}
