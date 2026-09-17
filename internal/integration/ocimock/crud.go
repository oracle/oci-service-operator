/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package ocimock

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
)

// Operation names a cloud behavior required by a mock integration scenario.
type Operation string

const (
	OperationCreate Operation = "create"
	OperationRead   Operation = "read"
	OperationUpdate Operation = "update"
	OperationDelete Operation = "delete"
)

// CRUDOptions describes one synchronous collection/item OCI API. Resource
// callbacks decode SDK requests, maintain typed state, and construct typed SDK
// response models; the shared responder owns routing and operation accounting.
type CRUDOptions[S any] struct {
	CollectionPath         string
	ItemPath               string
	CreatePath             string
	CreateMethod           string
	UpdatePath             string
	UpdateMethod           string
	DeletePath             string
	DeleteMethod           string
	ExpectedOperations     []Operation
	RequireCreateRead      bool
	RequireUpdateRead      bool
	RequireDeleteRead      bool
	RetainStateAfterDelete bool
	InitialState           *S
	List                   func(Request, bool, S) (Response, error)
	Create                 func(Request) (S, Response, error)
	Read                   func(Request, S) (Response, error)
	ReadTransition         func(Request, S) (S, Response, error)
	ReadByPhase            func(Request, ReadPhase, S) (S, Response, error)
	Update                 func(Request, S) (S, Response, error)
	Delete                 func(Request, S) (Response, error)
	DeleteTransition       func(Request, S) (S, Response, error)
	NotFound               func(Request) (Response, error)
	AdditionalRoutes       []Route
}

// Route declares one package-owned auxiliary SDK interaction.
type Route struct {
	Name         string
	Method       string
	Path         string
	MinimumCalls int
	Respond      func(Request) (Response, error)
}

// ReadPhase identifies why the service manager is reading the item route.
type ReadPhase string

const (
	ReadPhaseInitial ReadPhase = "initial"
	ReadPhaseCreated ReadPhase = "created"
	ReadPhaseUpdated ReadPhase = "updated"
	ReadPhaseDeleted ReadPhase = "deleted"
)

// CRUDResponder is a stateful synchronous OCI CRUD mock.
type CRUDResponder[S any] struct {
	mu          sync.Mutex
	options     CRUDOptions[S]
	state       S
	present     bool
	created     bool
	updated     bool
	deleted     bool
	createReads int
	updateReads int
	deleteReads int
	operations  map[Operation]int
	routeCalls  []int
}

// NewCRUDResponder validates and creates a stateful CRUD responder.
func NewCRUDResponder[S any](options CRUDOptions[S]) (*CRUDResponder[S], error) {
	options.CollectionPath = normalizePath(options.CollectionPath)
	options.ItemPath = normalizePath(options.ItemPath)
	options.CreatePath = normalizePath(options.CreatePath)
	options.UpdatePath = normalizePath(options.UpdatePath)
	options.DeletePath = normalizePath(options.DeletePath)
	if options.CreatePath == "" {
		options.CreatePath = options.CollectionPath
	}
	if options.CreateMethod == "" {
		options.CreateMethod = http.MethodPost
	}
	if options.UpdatePath == "" {
		options.UpdatePath = options.ItemPath
	}
	if options.DeletePath == "" {
		options.DeletePath = options.ItemPath
	}
	if options.DeleteMethod == "" {
		options.DeleteMethod = http.MethodDelete
	}
	if options.CollectionPath == "" || options.ItemPath == "" {
		return nil, errors.New("OCI mock CRUD collectionPath and itemPath are required")
	}
	if options.CollectionPath == options.ItemPath && options.List != nil {
		return nil, errors.New("OCI mock CRUD singleton path cannot also define a list handler")
	}
	readHandlers := 0
	for _, configured := range []bool{options.Read != nil, options.ReadTransition != nil, options.ReadByPhase != nil} {
		if configured {
			readHandlers++
		}
	}
	if readHandlers > 1 {
		return nil, errors.New("OCI mock CRUD accepts one of read, readTransition, or readByPhase")
	}
	if options.Delete != nil && options.DeleteTransition != nil {
		return nil, errors.New("OCI mock CRUD accepts delete or deleteTransition, not both")
	}
	if len(options.ExpectedOperations) == 0 {
		options.ExpectedOperations = expectedOperationsFromCallbacks(options)
	}
	seen := map[Operation]bool{}
	for _, operation := range options.ExpectedOperations {
		if !validOperation(operation) {
			return nil, fmt.Errorf("OCI mock CRUD operation %q is unsupported", operation)
		}
		if seen[operation] {
			return nil, fmt.Errorf("OCI mock CRUD operation %q is duplicated", operation)
		}
		seen[operation] = true
		if !operationHandlerConfigured(operation, options) {
			return nil, fmt.Errorf("OCI mock CRUD operation %q has no handler", operation)
		}
	}
	if options.RequireCreateRead && !seen[OperationCreate] {
		return nil, errors.New("OCI mock CRUD create readback requires create coverage")
	}
	if options.RequireUpdateRead && !seen[OperationUpdate] {
		return nil, errors.New("OCI mock CRUD update readback requires update coverage")
	}
	if options.RequireDeleteRead && !seen[OperationDelete] {
		return nil, errors.New("OCI mock CRUD delete readback requires delete coverage")
	}
	for index := range options.AdditionalRoutes {
		route := &options.AdditionalRoutes[index]
		route.Path = normalizePath(route.Path)
		if route.Name == "" || route.Method == "" || route.Path == "" || route.Respond == nil {
			return nil, fmt.Errorf("OCI mock CRUD auxiliary route %d is incomplete", index)
		}
		if route.MinimumCalls < 0 {
			return nil, fmt.Errorf("OCI mock CRUD auxiliary route %q minimumCalls is negative", route.Name)
		}
	}
	responder := &CRUDResponder[S]{options: options, operations: map[Operation]int{}, routeCalls: make([]int, len(options.AdditionalRoutes))}
	if options.InitialState != nil {
		responder.state = *options.InitialState
		responder.present = true
	}
	return responder, nil
}

// Respond routes an SDK-generated request and updates the typed mock state.
func (r *CRUDResponder[S]) Respond(request Request) (Response, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	requestPath := normalizePath(request.URL.Path)
	for index, route := range r.options.AdditionalRoutes {
		if request.Method == route.Method && requestPath == route.Path {
			r.routeCalls[index]++
			return route.Respond(request)
		}
	}
	switch {
	case request.Method == http.MethodGet && requestPath == r.options.CollectionPath && r.options.List != nil:
		r.operations[OperationRead]++
		if r.options.List == nil {
			return Response{}, fmt.Errorf("OCI mock CRUD list handler is not configured for %s", requestPath)
		}
		return r.options.List(request, r.present, r.state)
	case request.Method == r.options.CreateMethod && requestPath == r.options.CreatePath && !r.present:
		r.operations[OperationCreate]++
		if r.options.Create == nil {
			return Response{}, fmt.Errorf("OCI mock CRUD create handler is not configured for %s", requestPath)
		}
		if r.present {
			return Response{}, fmt.Errorf("OCI mock CRUD resource already exists at %s", r.options.ItemPath)
		}
		state, response, err := r.options.Create(request)
		if err == nil && successfulStatus(response.StatusCode) {
			r.state = state
			r.present = true
			r.created = true
			r.updated = false
			r.deleted = false
		}
		return response, err
	case request.Method == r.options.CreateMethod && requestPath == r.options.CreatePath &&
		!(requestPath == r.options.UpdatePath && updateMethodMatches(r.options.UpdateMethod, request.Method)):
		return Response{}, fmt.Errorf("OCI mock CRUD resource already exists at %s", r.options.ItemPath)
	case request.Method == http.MethodGet && requestPath == r.options.ItemPath:
		r.operations[OperationRead]++
		if !r.present {
			if r.deleted {
				r.deleteReads++
			}
			return r.notFound(request)
		}
		if r.deleted {
			r.deleteReads++
		} else if r.updated {
			r.updateReads++
		} else if r.created {
			r.createReads++
		}
		if r.options.ReadByPhase != nil {
			phase := r.readPhase()
			state, response, err := r.options.ReadByPhase(request, phase, r.state)
			if err == nil && successfulStatus(response.StatusCode) {
				r.state = state
			} else if err == nil && phase == ReadPhaseDeleted && response.StatusCode == http.StatusNotFound {
				r.present = false
			}
			return response, err
		}
		if r.options.ReadTransition != nil {
			state, response, err := r.options.ReadTransition(request, r.state)
			if err == nil && successfulStatus(response.StatusCode) {
				r.state = state
			}
			return response, err
		}
		if r.options.Read == nil {
			return Response{}, fmt.Errorf("OCI mock CRUD read handler is not configured for %s", requestPath)
		}
		return r.options.Read(request, r.state)
	case updateMethodMatches(r.options.UpdateMethod, request.Method) && requestPath == r.options.UpdatePath:
		r.operations[OperationUpdate]++
		if !r.present {
			return r.notFound(request)
		}
		if r.options.Update == nil {
			return Response{}, fmt.Errorf("OCI mock CRUD update handler is not configured for %s", requestPath)
		}
		state, response, err := r.options.Update(request, r.state)
		if err == nil && successfulStatus(response.StatusCode) {
			r.state = state
			r.updated = true
		}
		return response, err
	case request.Method == r.options.DeleteMethod && requestPath == r.options.DeletePath:
		r.operations[OperationDelete]++
		if !r.present {
			return r.notFound(request)
		}
		if r.options.Delete == nil && r.options.DeleteTransition == nil {
			return Response{}, fmt.Errorf("OCI mock CRUD delete handler is not configured for %s", requestPath)
		}
		var response Response
		var err error
		if r.options.DeleteTransition != nil {
			state, transitioned, transitionErr := r.options.DeleteTransition(request, r.state)
			response, err = transitioned, transitionErr
			if err == nil && successfulStatus(response.StatusCode) {
				r.state = state
			}
		} else {
			response, err = r.options.Delete(request, r.state)
		}
		if err == nil && successfulStatus(response.StatusCode) {
			r.present = r.options.RetainStateAfterDelete
			r.deleted = true
		}
		return response, err
	default:
		return Response{}, fmt.Errorf("OCI mock CRUD has no route for %s %s", request.Method, request.URL.String())
	}
}

func isUpdateMethod(method string) bool {
	return method == http.MethodPut || method == http.MethodPatch || method == http.MethodPost
}

func updateMethodMatches(configured, actual string) bool {
	if configured != "" {
		return actual == configured
	}
	return isUpdateMethod(actual)
}

func (r *CRUDResponder[S]) readPhase() ReadPhase {
	switch {
	case r.deleted:
		return ReadPhaseDeleted
	case r.updated:
		return ReadPhaseUpdated
	case r.created:
		return ReadPhaseCreated
	default:
		return ReadPhaseInitial
	}
}

// Verify requires every declared lifecycle operation to be exercised.
func (r *CRUDResponder[S]) Verify() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	var missing []string
	for _, operation := range r.options.ExpectedOperations {
		if r.operations[operation] == 0 {
			missing = append(missing, string(operation))
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("OCI mock CRUD did not exercise operation(s): %s", strings.Join(missing, ", "))
	}
	if r.options.RequireCreateRead && r.createReads == 0 {
		return errors.New("OCI mock CRUD did not read the resource after create")
	}
	if r.options.RequireUpdateRead && r.updateReads == 0 {
		return errors.New("OCI mock CRUD did not read the resource after update")
	}
	if r.options.RequireDeleteRead && r.deleteReads == 0 {
		return errors.New("OCI mock CRUD did not confirm deletion with a read")
	}
	for index, route := range r.options.AdditionalRoutes {
		if r.routeCalls[index] < route.MinimumCalls {
			return fmt.Errorf("OCI mock CRUD auxiliary route %q calls = %d, want at least %d", route.Name, r.routeCalls[index], route.MinimumCalls)
		}
	}
	return nil
}

// OperationCounts returns a defensive copy of observed lifecycle counts.
func (r *CRUDResponder[S]) OperationCounts() map[Operation]int {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make(map[Operation]int, len(r.operations))
	for operation, count := range r.operations {
		result[operation] = count
	}
	return result
}

// CurrentState returns the current typed state and whether it exists.
func (r *CRUDResponder[S]) CurrentState() (S, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.state, r.present
}

func (r *CRUDResponder[S]) notFound(request Request) (Response, error) {
	if r.options.NotFound != nil {
		return r.options.NotFound(request)
	}
	return JSONResponse(http.StatusNotFound, map[string]any{
		"code":    "NotAuthorizedOrNotFound",
		"message": "resource not found",
	})
}

func expectedOperationsFromCallbacks[S any](options CRUDOptions[S]) []Operation {
	var operations []Operation
	if options.Create != nil {
		operations = append(operations, OperationCreate)
	}
	if options.List != nil || options.Read != nil || options.ReadTransition != nil || options.ReadByPhase != nil {
		operations = append(operations, OperationRead)
	}
	if options.Update != nil {
		operations = append(operations, OperationUpdate)
	}
	if options.Delete != nil || options.DeleteTransition != nil {
		operations = append(operations, OperationDelete)
	}
	return operations
}

func validOperation(operation Operation) bool {
	switch operation {
	case OperationCreate, OperationRead, OperationUpdate, OperationDelete:
		return true
	default:
		return false
	}
}

func operationHandlerConfigured[S any](operation Operation, options CRUDOptions[S]) bool {
	switch operation {
	case OperationCreate:
		return options.Create != nil
	case OperationRead:
		return options.List != nil || options.Read != nil || options.ReadTransition != nil || options.ReadByPhase != nil
	case OperationUpdate:
		return options.Update != nil
	case OperationDelete:
		return options.Delete != nil || options.DeleteTransition != nil
	default:
		return false
	}
}

func successfulStatus(statusCode int) bool {
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	return statusCode >= 200 && statusCode < 300
}

func normalizePath(value string) string {
	value = strings.TrimSpace(value)
	if value == "/" {
		return value
	}
	return strings.TrimSuffix(value, "/")
}
