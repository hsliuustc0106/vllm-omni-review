#### Latent Cache Lifecycle
Caches must be explicitly cleared after generation.
Check for unbounded growth in generation loops.
Verify cleanup in error paths (try/finally or context manager).

#### Diffusion Pipeline
Check timestep scheduling correctness.
Verify VAE encode/decode pairs are balanced.
Ensure attention backends are correctly configured.

#### Memory Management
Large latent tensors must be explicitly freed.
Check for memory-pressure handling.
Verify offloading behavior when VRAM is tight.

#### Acceleration Features
Verify CUDA graph compatibility (static shapes only).
Check that caching (TeaCache, MagCache) doesn't affect quality.
Ensure parallel features (CFG, VAE) don't conflict.
