# Sample review snippet

- Severity: Medium
- Area: Verification
- Finding: The implementation adds server validation but the verify step only exercises the happy path.
- Evidence: No negative-case test or report entry covers malformed or duplicate emails.
- Recommendation: Add at least one server-side failure-path test and record it in the verify report.
