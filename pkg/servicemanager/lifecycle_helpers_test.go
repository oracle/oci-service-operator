package servicemanager

import (
	"errors"
	"fmt"
	"testing"

	shared "github.com/oracle/oci-service-operator/pkg/shared"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestResolveResourceID(t *testing.T) {
	t.Parallel()

	statusID := shared.OCID("ocid1.status.oc1..tracked")
	specID := shared.OCID("ocid1.spec.oc1..desired")

	got, err := ResolveResourceID(statusID, specID)
	if err != nil {
		t.Fatalf("ResolveResourceID() error = %v", err)
	}
	if got != statusID {
		t.Fatalf("ResolveResourceID() = %q, want tracked status id", got)
	}

	got, err = ResolveResourceID("", specID)
	if err != nil {
		t.Fatalf("ResolveResourceID() error = %v, want nil", err)
	}
	if got != specID {
		t.Fatalf("ResolveResourceID() = %q, want spec id", got)
	}

	if _, err := ResolveResourceID("", ""); err == nil {
		t.Fatal("ResolveResourceID() error = nil, want failure when both ids are empty")
	}
}

func TestSetCreatedAtIfUnset(t *testing.T) {
	t.Parallel()

	status := &shared.OSOKStatus{}
	SetCreatedAtIfUnset(status)
	if status.CreatedAt == nil {
		t.Fatal("SetCreatedAtIfUnset() should initialize CreatedAt")
	}

	existing := metav1.Now()
	status.CreatedAt = &existing
	SetCreatedAtIfUnset(status)
	if !status.CreatedAt.Equal(&existing) {
		t.Fatalf("CreatedAt = %v, want %v", status.CreatedAt, existing)
	}
}

func TestIsNotFoundServiceErrorRecognizes404Only(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "namespace not found", err: newLifecycleTestServiceError(404, "NamespaceNotFound", "namespace missing"), want: true},
		{name: "auth-shaped not found", err: newLifecycleTestServiceError(404, "NotAuthorizedOrNotFound", "missing"), want: true},
		{name: "forbidden", err: newLifecycleTestServiceError(403, "NotAuthorized", "forbidden"), want: false},
		{name: "generic", err: errors.New("boom"), want: false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := IsNotFoundServiceError(tc.err); got != tc.want {
				t.Fatalf("IsNotFoundServiceError() = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestIsNotFoundErrorStringRecognizesCaseInsensitiveMatches(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "http 404", err: errors.New("HTTP STATUS CODE: 404"), want: true},
		{name: "not found token", err: errors.New("NamespaceNotFound"), want: true},
		{name: "spaced not found", err: errors.New("resource not found"), want: true},
		{name: "generic", err: errors.New("boom"), want: false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := IsNotFoundErrorString(tc.err); got != tc.want {
				t.Fatalf("IsNotFoundErrorString() = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestIsSecretNotFoundErrorRecognizesKubernetesAndStringSignals(t *testing.T) {
	t.Parallel()

	notFound := apierrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, "sample")
	if !IsSecretNotFoundError(notFound) {
		t.Fatal("IsSecretNotFoundError() should accept Kubernetes NotFound")
	}
	if !IsSecretNotFoundError(errors.New("secret not found")) {
		t.Fatal("IsSecretNotFoundError() should accept matching error strings")
	}
	if IsSecretNotFoundError(errors.New("permission denied")) {
		t.Fatal("IsSecretNotFoundError() should reject unrelated errors")
	}
}

type lifecycleTestServiceError struct {
	statusCode int
	code       string
	message    string
}

func newLifecycleTestServiceError(statusCode int, code string, message string) lifecycleTestServiceError {
	if message == "" {
		message = fmt.Sprintf("%d %s", statusCode, code)
	}
	return lifecycleTestServiceError{
		statusCode: statusCode,
		code:       code,
		message:    message,
	}
}

func (e lifecycleTestServiceError) Error() string {
	return e.message
}

func (e lifecycleTestServiceError) GetHTTPStatusCode() int {
	return e.statusCode
}

func (e lifecycleTestServiceError) GetMessage() string {
	return e.message
}

func (e lifecycleTestServiceError) GetCode() string {
	return e.code
}

func (e lifecycleTestServiceError) GetOpcRequestID() string {
	return "opc-request-id"
}
