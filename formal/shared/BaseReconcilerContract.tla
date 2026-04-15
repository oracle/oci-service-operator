------------------------------ MODULE BaseReconcilerContract ------------------------------
EXTENDS ControllerLifecycleSpec

(*
Shared invariants captured from pkg/core/reconciler.go:
- keep the OSOK finalizer until delete has been confirmed
- delete-side status mutations persist before delete requeue or delete error returns
- successful active reconciles do not requeue
- provisioning, updating, and terminating states do requeue
- RequestedAt must be present on projected status
- secret side effects require an explicit policy
*)

VARIABLES
  deletionRequested,
  deleteConfirmed,
  finalizerPresent,
  deleteStatusMutated,
  deleteStatusPersisted,
  deleteReturnedError,
  lifecycleCondition,
  shouldRequeue,
  requestedAtStamped,
  secretWritePolicy

FinalizerRetention ==
  deletionRequested /\ ~deleteConfirmed => finalizerPresent

DeleteStatusPersistenceBeforeRetry ==
  deletionRequested /\ finalizerPresent /\ ~deleteConfirmed /\ deleteStatusMutated /\ (shouldRequeue \/ deleteReturnedError) => deleteStatusPersisted

SuccessNoImmediateRequeue ==
  lifecycleCondition = "Active" => ~shouldRequeue

RetryableConditionsRequeue ==
  lifecycleCondition \in {"Provisioning", "Updating", "Terminating"} => shouldRequeue

StatusProjectionStampsRequestedAt ==
  requestedAtStamped

SecretWritesNeedExplicitPolicy ==
  secretWritePolicy \in {"none", "ready-only"}

ContractInvariant ==
  /\ FinalizerRetention
  /\ DeleteStatusPersistenceBeforeRetry
  /\ SuccessNoImmediateRequeue
  /\ RetryableConditionsRequeue
  /\ StatusProjectionStampsRequestedAt
  /\ SecretWritesNeedExplicitPolicy

=============================================================================
