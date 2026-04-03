package generatedruntime

import sharedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

type Operation = sharedruntime.Operation
type RequestField = sharedruntime.RequestField
type Hook = sharedruntime.Hook
type FollowUpSemantics = sharedruntime.FollowUpSemantics
type HookSet = sharedruntime.HookSet
type LifecycleSemantics = sharedruntime.LifecycleSemantics
type DeleteSemantics = sharedruntime.DeleteSemantics
type ListSemantics = sharedruntime.ListSemantics
type MutationSemantics = sharedruntime.MutationSemantics
type AuxiliaryOperation = sharedruntime.AuxiliaryOperation
type UnsupportedSemantic = sharedruntime.UnsupportedSemantic
type Semantics = sharedruntime.Semantics
type Config[T any] = sharedruntime.Config[T]
type ServiceClient[T any] = sharedruntime.ServiceClient[T]

func NewServiceClient[T any](cfg Config[T]) ServiceClient[T] {
	return sharedruntime.NewServiceClient(cfg)
}
