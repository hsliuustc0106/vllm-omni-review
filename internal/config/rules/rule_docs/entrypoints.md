#### Input Validation
Validate all user inputs at the API layer, before reaching the engine.
Return clean HTTP 400 errors with actionable messages.
Never pass unvalidated user input to model.generate().

#### API Contract
Responses must match the declared schema.
Check for proper error response formatting.
Verify OpenAI API compatibility where claimed.

#### Security
No user input in eval/exec/format strings.
Check for proper authentication/authorization if applicable.
Sensitive data must not leak into log messages or error responses.
