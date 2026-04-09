------------------------------ MODULE SecretSideEffectsContract ------------------------------

SecretWritePolicies == {"none", "ready-only"}

SecretWritesAllowed(policy, condition) ==
  /\ policy \in SecretWritePolicies
  /\ IF policy = "none" THEN FALSE ELSE condition = "Active"

=============================================================================
