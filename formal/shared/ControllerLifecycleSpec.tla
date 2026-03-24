------------------------------ MODULE ControllerLifecycleSpec ------------------------------

RetryableConditions == {"Provisioning", "Updating", "Terminating"}

ShouldRequeue(condition) ==
  condition \in RetryableConditions

SuccessCondition == "Active"

DeleteInProgress(condition) ==
  condition = "Terminating"

=============================================================================
