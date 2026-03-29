# SECURITY Stage Guidance

- This is a verification gate. Do not edit or commit code here.
- Check for hardcoded secrets, auth or credential handling mistakes, unsafe file, shell, or network behavior, risky RBAC or manifest changes, and new dependency risk.
- If `govulncheck` is installed locally, run `govulncheck ./...`; otherwise note that it was unavailable and continue with manual review.
- Reject changes that introduce credential leakage, unsafe defaults, or overly broad permissions without explicit justification.
