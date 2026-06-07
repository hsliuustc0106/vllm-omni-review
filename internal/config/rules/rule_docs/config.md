#### Validation
All config fields must be validated at load time (fail fast).
Required fields must have explicit checks, not just defaults.
Invalid combinations (e.g., conflicting acceleration features) must error.

#### Defaults
Defaults must be safe for production use.
Check that default values don't mask missing configuration.
Breaking config changes must be documented.

#### Stage Configs
Stage YAML files must be parseable and complete.
Check that all referenced stages exist and are configured.
Verify pipeline topology makes sense (no missing connections).
