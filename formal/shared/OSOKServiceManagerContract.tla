------------------------------ MODULE OSOKServiceManagerContract ------------------------------
EXTENDS Naturals

(*
Shared contract captured from pkg/servicemanager/interfaces.go and the generated
service-manager scaffolds:
- CreateOrUpdate returns an OSOKResponse with explicit success and requeue bits
- Delete reports confirmation separately from transport errors
- GetCrdStatus always projects an OSOKStatus surface back to the reconciler
*)

ResponseShape(response) ==
  response \in [IsSuccessful : BOOLEAN, ShouldRequeue : BOOLEAN, RequeueDuration : Nat]

SuccessfulResponsesAvoidFailedCondition(response, condition) ==
  response.IsSuccessful => condition # "Failed"

RequeueMatchesRetryableConditions(response, condition) ==
  response.ShouldRequeue => condition \in {"Provisioning", "Updating", "Terminating"}

DeleteRequiresExplicitConfirmation(deleteConfirmed, deleteErrored) ==
  deleteErrored => ~deleteConfirmed

StatusProjectionReturnsOSOKStatus(statusProjected) ==
  statusProjected

ContractInvariant(response, condition, deleteConfirmed, deleteErrored, statusProjected) ==
  /\ ResponseShape(response)
  /\ SuccessfulResponsesAvoidFailedCondition(response, condition)
  /\ RequeueMatchesRetryableConditions(response, condition)
  /\ DeleteRequiresExplicitConfirmation(deleteConfirmed, deleteErrored)
  /\ StatusProjectionReturnsOSOKStatus(statusProjected)

=============================================================================
