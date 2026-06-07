#### Scheduler Correctness
Verify no deadlocks in concurrent request paths.
Check that state transitions are explicit and validated.
Ensure cleanup happens on cancellation and error paths.

#### KV Cache Management
Allocation, update, and cleanup must be clearly separated.
Error paths must clean up allocated cache blocks.
Use type system / enums to enforce valid cache states.

#### Async Safety
No blocking calls in async code paths.
Use asyncio.to_thread() for blocking operations.
Check for race conditions in shared state.

#### Worker Lifecycle
Workers must have proper init/start/stop with state guards.
Resource cleanup in stop() must handle partial initialization.
Check for dangling worker processes on error.
