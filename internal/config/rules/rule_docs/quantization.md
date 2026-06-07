#### Weight Packing Correctness
Verify quantization scales and zero points are correctly applied.
Check for dtype consistency between packed weights and compute dtype.
Ensure dequantization happens at the right point in the forward pass.

#### Memory Accuracy
Claimed memory savings should be plausible.
Check for quantization config validation at load time.
Verify fallback to FP16 when quantization is unavailable.

#### Quality Impact
New quantization methods should document accuracy impact.
Check for calibration data references where applicable.
