#### Model Registration
New models must be properly registered in the model registry.
Config classes must match between encoder and decoder components.
Check that processor chains are correctly wired.

#### Mixin + nn.Module MRO
Verify mixin is listed BEFORE nn.Module in inheritance, OR uses lazy init.
Check that test mocks match production class hierarchy.
If mixin defines __init__, verify it's actually called.

#### Weight Loading
Check for safe tensor loading with proper device placement.
Verify memory budgeting for large model weights.
Ensure fallback paths for missing optional weights.

#### Stage Input Processors
Processors must handle all modalities declared by the model.
Check for proper None handling in processor chains.
Verify processor output shapes match stage input expectations.
