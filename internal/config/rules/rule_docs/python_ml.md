#### Correctness
Is the logic correct? Are there missing boundary conditions?
Are exceptions handled properly with specific exception types?
Is it thread-safe / async-safe in concurrent serving scenarios?

#### Multimodal Correctness
Are tensor shapes consistent across modality fusion points?
Are dtypes correct across encoder/decoder boundaries?
Are multi-stage pipeline transitions (e.g., Thinker → Talker) correct?
Is modality routing (text vs image vs audio path) correct?

#### Security
Are there hardcoded secrets or API keys?
Is user input validated before reaching the engine?
Are eval/exec/pickle.loads used on untrusted data?

#### Performance
Are there obvious performance issues (N+1 queries, unnecessary copies)?
Are resources properly released (GPU memory, file handles, connectors)?
Are async operations parallelized with asyncio.gather where possible?

#### Maintainability
Does the code follow vllm-omni's existing patterns and conventions?
Do names accurately express intent?
Are new public APIs documented?

#### Test Coverage
Do critical logic paths have tests?
Do tests cover edge cases (None, empty, boundary)?
Are mocks faithful to production class hierarchies?
