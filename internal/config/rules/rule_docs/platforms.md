#### Device-Specific Code
Check for non-CUDA fallbacks (ROCm, NPU, XPU, MUSA).
Verify device placement is explicit, not assumed.
No hardcoded 'cuda' device strings without fallback.

#### Custom Ops
Custom CUDA/ROCm ops must have proper error checking.
Verify kernel launch configurations for correctness.
Check for stream synchronization after custom ops.

#### Tensor Parallelism
Bias tensors must be properly replicated across ranks.
Rank-specific logic must be isolated and tested.
Check for distributed initialization guards (dist.is_initialized()).
