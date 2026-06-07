#### Resource Management
Use context managers for all buffer/channel acquires.
Verify cleanup in ALL code paths (including error branches).
Check for try/finally or async context manager usage.

#### Error Propagation
Timeout errors must be handled gracefully.
Connector failures should surface as clear errors, not hangs.
Check for proper error propagation to the calling stage.

#### Configuration
Timeouts must be configurable, not hardcoded.
Buffer sizes should match expected payload sizes.
Check for sensible defaults with override capability.
