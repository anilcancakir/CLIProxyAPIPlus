# Port Lemon Fork Improvements to Anilcan Fork

## TL;DR

> **Quick Summary**: Port all 8 patches + direct source changes from the lemon07r/CLIProxyAPIPlus fork into our anilcancakir fork using direct source modification, while preserving existing custom features and upstream sync capability.
> 
> **Deliverables**:
> - Copilot unlimited headers + Claude endpoint routing + assistant prefill fix
> - Antigravity anti-fingerprinting with dynamic version fetching
> - Thinking signature fix, consecutive turn merge, vision detection, streaming tool call deltas
> - Adapted CI/CD workflow (GitHub Actions upstream sync + Docker build)
> - Adapted config examples (config.example.custom.yaml, example.opencode.json)
> 
> **Estimated Effort**: Large
> **Parallel Execution**: YES - 2 waves
> **Critical Path**: Wave 1 tasks → Integration verification → Final QA

---

## Context

### Original Request
Port improvements from the lemon07r/CLIProxyAPIPlus fork into our anilcancakir fork, preserving upstream sync with router-for-me/CLIProxyAPIPlus and our existing custom changes (KooshaPari port, SDK features, TUI, cliproxyctl).

### Interview Summary
**Key Discussions**:
- **Integration approach**: Direct source modification (not patch-based like lemon)
- **Scope**: All 8 patches + direct source changes + CI/CD + config examples
- **CI/CD**: Adapted GitHub Actions workflow for our repo
- **Config examples**: Adapted with our branding/URLs

**Research Findings**:
- Our fork: ~85 files changed vs upstream (+7958/-701)
- Lemon fork: ~19 files changed vs upstream (+2384/-20)
- Significant overlap in: stabilizeResponsesIDs, filterResponsesReasoningItems, tool ID injection, thinking signature handling
- Lemon's biggest patches: 002 (Claude via Copilot, 670 lines) and 006 (anti-fingerprinting, 305 lines)

### Metis Review
**Identified Gaps** (addressed):
- **File contention risk**: Tasks grouped by TARGET FILES, not by patch number, to prevent parallel agent conflicts
- **Conflict resolution strategy**: Preserve our existing custom features; integrate lemon's alongside, don't overwrite
- **Upstream sync preservation**: Changes are direct source mods; future upstream merges may need manual conflict resolution on touched files — this is acceptable given our existing approach
- **Scope creep prevention**: Agents must NOT refactor unrelated code while applying patches

---

## Work Objectives

### Core Objective
Apply all functional improvements from the lemon fork (8 patches + direct source changes) into our codebase using direct source modification, with adapted CI/CD and config examples.

### Concrete Deliverables
- Modified `internal/runtime/executor/github_copilot_executor.go` (patches 001, 002, part of 007)
- Modified `internal/runtime/executor/github_copilot_executor_test.go` (patch 002 tests)
- Modified `internal/translator/claude/openai/chat-completions/claude_openai_response.go` (patch 002 + 008)
- Modified `sdk/api/handlers/handlers.go` (patch 002 copilot- prefix handling)
- Modified `sdk/api/handlers/openai/endpoint_compat.go` (patch 002 copilot- prefix)
- Modified `sdk/api/handlers/openai/openai_handlers.go` (patch 002 copilot- prefix)
- Modified `internal/translator/antigravity/claude/antigravity_claude_request.go` (patches 003, 004 + enableThoughtTranslate removal)
- Modified `internal/translator/antigravity/claude/antigravity_claude_response.go` (non-stream thinking signature cache)
- Modified `internal/translator/antigravity/gemini/antigravity_gemini_request.go` (patches 003, 004, 005)
- Modified `internal/runtime/executor/antigravity_executor.go` (patch 006)
- New `.github/workflows/sync-and-build.yml`
- New `config.example.custom.yaml`
- New `example.opencode.json`

### Definition of Done
- [ ] `go build ./...` succeeds with zero errors
- [ ] `go vet ./...` reports no issues
- [ ] `go test ./...` passes all tests (existing + new)
- [ ] All 8 patch functionalities are present in the source code
- [ ] Existing custom features (stabilizeResponsesIDs, filterResponsesReasoningItems, tool ID injection) remain intact
- [ ] CI/CD workflow file exists and is syntactically valid YAML

### Must Have
- All Copilot header updates (patch 001)
- Claude endpoint routing via Copilot with native format (patch 002)
- Thinking signature fix for Claude via Antigravity (patch 003)
- Assistant prefill fix for Claude via Antigravity (patch 004)
- Consecutive same-role turn merging for Gemini API (patch 005)
- Anti-fingerprinting with dynamic version fetching (patch 006)
- Vision detection for Copilot Responses API (patch 007)
- Streaming tool call deltas in Claude-to-OpenAI translator (patch 008)
- Adapted CI/CD workflow
- Adapted config examples

### Must NOT Have (Guardrails)
- Do NOT overwrite our existing `stabilizeResponsesStreamIDs` / `stabilizeResponsesNonStreamIDs` implementations (already working)
- Do NOT overwrite our existing `filterResponsesReasoningItems` implementation (already working)
- Do NOT refactor or touch any Go files outside the scope of the target patches
- Do NOT change our `docker-compose.yml` image to lemon07r (keep our own or upstream default)
- Do NOT remove or modify our existing TUI, cliproxyctl, SDK features
- Do NOT add `// +lemon-port` demarcation comments — our approach is direct integration, not annotated patches
- Over-engineer abstractions: if the patch adds a simple `if` statement, add the `if` statement

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed. No exceptions.

### Test Decision
- **Infrastructure exists**: YES (Go test framework)
- **Automated tests**: YES (Tests-after — update existing tests for modified signatures, add new tests for new helpers)
- **Framework**: `go test`
- **Approach**: Modify existing tests to match new function signatures, add tests for new helper functions (isCopilotClaudeModel, isGPT5Model, normalizeCopilotClaudeThinking, etc.)

### QA Policy
Every task MUST include agent-executed QA scenarios.
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **Go compilation**: Use Bash — `go build ./...`
- **Go tests**: Use Bash — `go test ./... -v -count=1`
- **Go vet**: Use Bash — `go vet ./...`
- **YAML validation**: Use Bash — validate workflow YAML structure

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Start Immediately — domain-isolated, MAX PARALLEL):
├── Task 1: Scaffolding — CI/CD workflow + config examples [quick]
├── Task 2: Port Copilot Features — Patches 001, 002, 007 [deep]
│   (github_copilot_executor.go, executor_test.go, claude_openai_response.go,
│    sdk/handlers.go, endpoint_compat.go, openai_handlers.go)
├── Task 3: Port Anti-Fingerprinting — Patch 006 [unspecified-high]
│   (antigravity_executor.go)
├── Task 4: Port Antigravity/Translator Features — Patches 003, 004, 005, 008 + Direct [deep]
│   (antigravity_claude_request.go, antigravity_claude_response.go,
│    antigravity_gemini_request.go)

Wave 2 (After Wave 1 — integration & QA):
├── Task 5: Integration Build + Test Fix [unspecified-high]
└── Task 6: Final Verification [deep]

Wave FINAL (After ALL tasks — independent review):
├── Task F1: Plan compliance audit [oracle]
├── Task F2: Code quality review [unspecified-high]
├── Task F3: Real build + test QA [unspecified-high]
└── Task F4: Scope fidelity check [deep]

Critical Path: Tasks 2-4 → Task 5 → Task 6 → F1-F4
Parallel Speedup: ~60% faster than sequential
Max Concurrent: 4 (Wave 1)
```

### Dependency Matrix

| Task | Depends On | Blocks | Wave |
|------|-----------|--------|------|
| 1    | —         | 5      | 1    |
| 2    | —         | 5      | 1    |
| 3    | —         | 5      | 1    |
| 4    | —         | 5      | 1    |
| 5    | 1,2,3,4   | 6      | 2    |
| 6    | 5         | F1-F4  | 2    |
| F1-F4| 6         | —      | FINAL|

### Agent Dispatch Summary

- **Wave 1**: **4 tasks** — T1 → `quick`, T2 → `deep`, T3 → `unspecified-high`, T4 → `deep`
- **Wave 2**: **2 tasks** — T5 → `unspecified-high`, T6 → `deep`
- **FINAL**: **4 tasks** — F1 → `oracle`, F2 → `unspecified-high`, F3 → `unspecified-high`, F4 → `deep`

---

## TODOs

> Implementation + Test = ONE Task. Never separate.
> EVERY task MUST have: Recommended Agent Profile + Parallelization info + QA Scenarios.

- [ ] 1. Scaffolding — CI/CD Workflow + Config Examples

  **What to do**:
  - Create `.github/workflows/sync-and-build.yml` adapted from lemon's `build-and-push.yml`:
    - Change `DOCKERHUB_REPO` references to use our repo (`anilcancakir/cli-proxy-api-plus`)
    - Keep the upstream sync logic (fetch upstream, merge, push)
    - Keep the Docker build with patch application mechanism in Dockerfile
    - Update `runs-on` to `ubuntu-latest` (not ARM-specific unless we need ARM)
    - Remove lemon-specific secrets references, use generic `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN`
  - Create `config.example.custom.yaml` adapted from lemon's version:
    - Keep the full copilot alias examples (copilot-claude-opus-4.6, copilot-claude-sonnet-4, etc.)
    - Keep the antigravity alias examples
    - Keep the codex alias examples
    - Replace any lemon-specific URLs/references
  - Create `example.opencode.json` adapted from lemon's version:
    - Keep all provider definitions (cliproxy-codex, cliproxy-copilot, cliproxy-copilot-gpt, cliproxy-qwen, cliproxy-kimi, cliproxy-antigravity)
    - Replace `your.cliproxy.ip.or.url` placeholder (keep as-is, it's already a placeholder)
  - Update `Dockerfile` to include patch application mechanism (git init + git apply loop for patches/ directory)

  **Must NOT do**:
  - Do NOT change docker-compose.yml default image to lemon07r
  - Do NOT modify any Go source files
  - Do NOT add lemon-specific branding

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: [`anilcan-coding`]

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2, 3, 4)
  - **Blocks**: [5]
  - **Blocked By**: None

  **References**:
  - Lemon's workflow: `/Users/anilcan/Code/ai-help/repos/cliproxyforks/lemon/.github/workflows/build-and-push.yml`
  - Lemon's Dockerfile: `/Users/anilcan/Code/ai-help/repos/cliproxyforks/lemon/Dockerfile`
  - Lemon's custom config: `/Users/anilcan/Code/ai-help/repos/cliproxyforks/lemon/config.example.custom.yaml`
  - Lemon's opencode config: `/Users/anilcan/Code/ai-help/repos/cliproxyforks/lemon/example.opencode.json`
  - Our existing Dockerfile: `Dockerfile` — Extend with patch mechanism
  - Our existing docker-compose.yml: `docker-compose.yml` — DO NOT change default image

  **Acceptance Criteria**:
  - [ ] `.github/workflows/sync-and-build.yml` exists and is valid YAML
  - [ ] `config.example.custom.yaml` exists with copilot alias examples
  - [ ] `example.opencode.json` exists with all provider definitions
  - [ ] `Dockerfile` includes patch application step
  - [ ] `docker-compose.yml` default image is NOT changed

  **QA Scenarios (MANDATORY):**
  ```
  Scenario: CI/CD workflow YAML is valid
    Tool: Bash
    Steps:
      1. Run: python3 -c "import yaml; yaml.safe_load(open('.github/workflows/sync-and-build.yml'))"
      2. Assert: exit code 0
    Expected Result: Valid YAML
    Evidence: .sisyphus/evidence/task-1-yaml-valid.txt

  Scenario: Config example contains copilot aliases
    Tool: Bash
    Steps:
      1. Run: grep -c 'copilot-claude' config.example.custom.yaml
      2. Assert: count > 0
    Expected Result: Copilot aliases present
    Evidence: .sisyphus/evidence/task-1-config-aliases.txt

  Scenario: Dockerfile has patch application
    Tool: Bash
    Steps:
      1. Run: grep -c 'git apply' Dockerfile
      2. Assert: count > 0
    Expected Result: Patch mechanism present
    Evidence: .sisyphus/evidence/task-1-dockerfile-patches.txt
  ```

  **Commit**: YES
  - Message: `feat: add CI/CD sync workflow, config examples, and Dockerfile patch mechanism from lemon fork`
  - Files: `.github/workflows/sync-and-build.yml`, `config.example.custom.yaml`, `example.opencode.json`, `Dockerfile`

---

- [ ] 2. Port Copilot Features — Patches 001, 002, 007

  **What to do**:
  Apply changes to the Copilot executor and related SDK handler files. This is the LARGEST task.

  **From Patch 001 (Unlimited Copilot Headers):**
  - In `internal/runtime/executor/github_copilot_executor.go`, update header constants:
    - `copilotUserAgent` → `"GithubCopilot/1.0"`
    - `copilotEditorVersion` → `"vscode/1.109.0-20260124"`
    - `copilotPluginVersion` → `"copilot-chat/0.37.2026013101"`
    - `copilotOpenAIIntent` → `"conversation-edits"`
    - `copilotGitHubAPIVer` → `"2025-10-01"`
    - Add: `copilotThinkingBeta = "interleaved-thinking-2025-05-14,context-management-2025-06-27"`
  - In `applyHeaders`: always set `X-Initiator` to `"agent"` (remove dynamic logic), add `VScode-SessionId` and `VScode-MachineId` with `uuid.NewString()`

  **From Patch 002 (Copilot Claude Endpoint — 670 lines):**
  - Add constant: `githubCopilotMessagesPath = "/v1/messages"`
  - Add helpers: `isCopilotClaudeModel(model)`, `isGPT5Model(model)`, `getEndpointPath(model, format)`, `isCopilotClaudeFormat(format)`, `normalizeCopilotClaudeThinking(model, body)`
  - Change `applyHeaders` signature: add `format sdktranslator.Format` param, add `anthropic-beta` header for Claude
  - Modify `Execute` method: tri-state format (`claude`/`codex`/`openai`), skip flattenAssistantContent for Claude, skip OpenAI thinking for Claude, add normalizeCopilotClaudeThinking, use getEndpointPath, add parseClaudeUsage fallback
  - Modify `ExecuteStream`: same tri-state format, SSE passthrough via `HasResponseTransformer`, parseClaudeStreamUsage
  - Update `normalizeModel`: strip `copilot-` prefix before thinking suffix parse
  - Replace `useGitHubCopilotResponsesEndpoint` with `isGPT5Model`
  - SDK changes:
    - `sdk/api/handlers/handlers.go`: fallback `copilot-` prefix stripping in provider resolution
    - `sdk/api/handlers/openai/endpoint_compat.go`: strip `copilot-` prefix in registry lookup
    - `sdk/api/handlers/openai/openai_handlers.go`: strip `copilot-` prefix before responses routing
  - Claude translator: add `convertPlainClaudeResponseToOpenAI` in `claude_openai_response.go`

  **From Patch 007 (Vision Detection):**
  - Verify vision detection works for Copilot Responses API; add if missing

  **Tests:**
  - Update `TestApplyHeadersUserInitiator` → expect `"agent"`
  - Update all `applyHeaders` call signatures in tests (add format param)
  - Replace `TestUseGitHubCopilotResponsesEndpoint_*` with `TestIsGPT5Model` and `TestIsCopilotClaudeModel`
  - Update `TestApplyHeaders_GitHubAPIVersion` → expect `"2025-10-01"`

  **Must NOT do**:
  - Do NOT modify existing `stabilizeResponsesStreamIDs`/`stabilizeResponsesNonStreamIDs`
  - Do NOT modify existing `filterResponsesReasoningItems`
  - Do NOT touch antigravity executor (Task 3) or translator files (Task 4)

  **Recommended Agent Profile**:
  - **Category**: `deep` — 670 lines of complex endpoint routing
  - **Skills**: [`anilcan-coding`]

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 3, 4)
  - **Blocks**: [5]
  - **Blocked By**: None

  **References**:
  - Lemon Patch 001: `/Users/anilcan/Code/ai-help/repos/cliproxyforks/lemon/patches/001-unlimited-copilot-headers.patch`
  - Lemon Patch 002: `/Users/anilcan/Code/ai-help/repos/cliproxyforks/lemon/patches/002-copilot-claude-endpoint.patch`
  - Lemon Patch 007: `/Users/anilcan/Code/ai-help/repos/cliproxyforks/lemon/patches/007-copilot-responses-vision-detection.patch`
  - Our copilot executor: `internal/runtime/executor/github_copilot_executor.go`
  - Our copilot tests: `internal/runtime/executor/github_copilot_executor_test.go`
  - Claude response translator: `internal/translator/claude/openai/chat-completions/claude_openai_response.go`
  - SDK handlers: `sdk/api/handlers/handlers.go`
  - Endpoint compat: `sdk/api/handlers/openai/endpoint_compat.go`
  - OpenAI handlers: `sdk/api/handlers/openai/openai_handlers.go`

  **Acceptance Criteria**:
  - [ ] `go build ./...` succeeds
  - [ ] `go test ./internal/runtime/executor/... -v -count=1` passes
  - [ ] Helper functions `isCopilotClaudeModel`, `isGPT5Model`, `getEndpointPath` exist and are correct
  - [ ] `normalizeModel` strips `copilot-` prefix
  - [ ] `applyHeaders` always sets `X-Initiator` to `"agent"` and includes format param
  - [ ] `convertPlainClaudeResponseToOpenAI` exists in claude_openai_response.go
  - [ ] Existing `stabilizeResponsesStreamIDs` and `filterResponsesReasoningItems` are untouched

  **QA Scenarios (MANDATORY):**
  ```
  Scenario: Go build succeeds after copilot changes
    Tool: Bash
    Steps:
      1. Run: go build ./...
      2. Assert: exit code 0
    Expected Result: Clean compilation
    Evidence: .sisyphus/evidence/task-2-build.txt

  Scenario: Copilot executor tests pass
    Tool: Bash
    Steps:
      1. Run: go test ./internal/runtime/executor/... -v -run 'TestIsGPT5|TestIsCopilotClaude|TestApplyHeaders|TestNormalize' -count=1
      2. Assert: all PASS
    Expected Result: All copilot tests pass
    Evidence: .sisyphus/evidence/task-2-copilot-tests.txt

  Scenario: Existing custom functions preserved
    Tool: Bash
    Steps:
      1. Run: grep -n 'func.*stabilizeResponsesNonStreamIDs' internal/runtime/executor/github_copilot_executor.go
      2. Run: grep -n 'func.*filterResponsesReasoningItems' internal/runtime/executor/github_copilot_executor.go
      3. Assert: both found
    Expected Result: Custom functions intact
    Evidence: .sisyphus/evidence/task-2-preserved.txt
  ```

  **Commit**: YES
  - Message: `feat(copilot): port Claude endpoint routing, unlimited headers, and vision detection from lemon fork`
  - Files: copilot executor, tests, claude response translator, sdk handlers
  - Pre-commit: `go build ./... && go test ./internal/runtime/executor/... -count=1`

---

- [ ] 3. Port Anti-Fingerprinting — Patch 006

  **What to do**:
  Apply the 305-line anti-fingerprinting changes to `internal/runtime/executor/antigravity_executor.go`:

  - Fix `defaultAntigravityAgent` version: `"antigravity/1.18.3 darwin/arm64"` (was incorrectly showing old version)
  - Add dynamic version fetching:
    - Constants: `antigravityVersionURL`, `antigravityVersionRefresh` (12h)
    - Variables: `antigravityUACache`, `antigravityAccountVersions`, `antigravityUAVersions`, `antigravityUAPlatforms`, `antigravityDynVersion`, `antigravityVersionRe`
    - Function: `fetchAntigravityVersion()` — HTTP GET to auto-updater API with 5s timeout, regex parse version, cache 12h
    - Function: `compareVersions(a, b)` — semver comparison
  - Per-account User-Agent generation:
    - In `getAntigravityUserAgent()`: check cache first, try dynamic version, fall back to deterministic static pick, prevent version downgrades per account, cache result
  - Session ID fix:
    - In `geminiToAntigravity()`: add `authID` parameter, salt session ID with auth ID for cross-account correlation prevention, fix format to match real traffic pattern `(-{uuid}:{model}:{project}:seed-{hex16})`
  - Project ID word pool expansion: ~30 adjectives/nouns instead of ~10
  - Update `geminiToAntigravity` call site to pass `authID`

  **Must NOT do**:
  - Do NOT touch copilot executor files (Task 2's domain)
  - Do NOT touch translator files (Task 4's domain)
  - Do NOT refactor unrelated antigravity code

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high` — 305 lines of HTTP/TLS modifications
  - **Skills**: [`anilcan-coding`]

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2, 4)
  - **Blocks**: [5]
  - **Blocked By**: None

  **References**:
  - Lemon Patch 006: `/Users/anilcan/Code/ai-help/repos/cliproxyforks/lemon/patches/006-antigravity-anti-fingerprinting.patch`
  - Our antigravity executor: `internal/runtime/executor/antigravity_executor.go`
  - Lemon's antigravity executor (final state): Read lemon's file directly at `/Users/anilcan/Code/ai-help/repos/cliproxyforks/lemon/internal/runtime/executor/antigravity_executor.go` for reference

  **Acceptance Criteria**:
  - [ ] `go build ./...` succeeds
  - [ ] `fetchAntigravityVersion()` function exists
  - [ ] `compareVersions()` function exists
  - [ ] `antigravityUACache` and `antigravityAccountVersions` maps exist
  - [ ] `geminiToAntigravity` accepts `authID` parameter
  - [ ] `defaultAntigravityAgent` is `"antigravity/1.18.3 darwin/arm64"`

  **QA Scenarios (MANDATORY):**
  ```
  Scenario: Go build succeeds after anti-fingerprinting changes
    Tool: Bash
    Steps:
      1. Run: go build ./...
      2. Assert: exit code 0
    Expected Result: Clean compilation
    Evidence: .sisyphus/evidence/task-3-build.txt

  Scenario: New functions exist
    Tool: Bash
    Steps:
      1. Run: grep -n 'func fetchAntigravityVersion' internal/runtime/executor/antigravity_executor.go
      2. Run: grep -n 'func compareVersions' internal/runtime/executor/antigravity_executor.go
      3. Assert: both found
    Expected Result: Functions present
    Evidence: .sisyphus/evidence/task-3-functions.txt

  Scenario: Default agent version updated
    Tool: Bash
    Steps:
      1. Run: grep 'defaultAntigravityAgent' internal/runtime/executor/antigravity_executor.go
      2. Assert: contains "antigravity/1.18.3"
    Expected Result: Version updated
    Evidence: .sisyphus/evidence/task-3-version.txt
  ```

  **Commit**: YES
  - Message: `feat(antigravity): port anti-fingerprinting with dynamic version fetching from lemon fork`
  - Files: `internal/runtime/executor/antigravity_executor.go`
  - Pre-commit: `go build ./...`

---

- [ ] 4. Port Antigravity/Translator Features — Patches 003, 004, 005, 008 + Direct Changes

  **What to do**:
  Apply changes to the Antigravity translator files. Handle overlaps with our existing code carefully.

  **From Patch 003 (Thinking Signature Fix):**
  - In `internal/translator/antigravity/gemini/antigravity_gemini_request.go`:
    - Replace the Gemini-only thinking signature handling with model-aware logic:
      - For Gemini (non-Claude): keep `skip_thought_signature_validator` sentinel on thinking + functionCall parts
      - For Claude: strip thinking blocks entirely from previous assistant turns, clear stale signatures from functionCall parts
      - Clean up both `thoughtSignature` and `thought_signature` (snake_case) fields
    - IMPORTANT: Our fork already has tool ID injection in this file. DO NOT remove it. The thinking signature changes go in a different section of the `ForEach` loop.

  **From Patch 004 (Assistant Prefill Fix):**
  - In `internal/translator/antigravity/claude/antigravity_claude_request.go`:
    - After contents processing, detect trailing "model" messages without functionCall parts
    - Extract text as prefill, remove trailing model message, inject synthetic user message with "Continue from: " prefix
  - In `internal/translator/antigravity/gemini/antigravity_gemini_request.go`:
    - Same prefill logic for Claude models only (check `strings.Contains(modelName, "claude")`)
    - Native Gemini models unaffected

  **From Patch 005 (Merge Consecutive Turns):**
  - In `internal/translator/antigravity/gemini/antigravity_gemini_request.go`:
    - Add consecutive same-role turn merging BEFORE the assistant prefill handling
    - Track `prevRole`, when same role appears consecutively, merge parts arrays

  **From Patch 008 (Streaming Tool Call Deltas):**
  - In `internal/translator/claude/openai/chat-completions/claude_openai_response.go`:
    - Add streaming support for tool call argument deltas in Claude-to-OpenAI translation
    - Read lemon's patch 008 for exact implementation

  **Direct Source Changes (not in patches):**
  - In `antigravity_claude_request.go`: Remove `enableThoughtTranslate` flag entirely. Currently when unsigned thinking block is found, it sets `enableThoughtTranslate = false` which disables thinking for the entire request. Instead, just drop the unsigned block silently and continue translating subsequent messages normally.
  - In `antigravity_claude_response.go`: In `ConvertAntigravityResponseToClaudeNonStream`, add `cache.CacheSignature(modelName, thinkingText, thinkingSignature)` call before constructing the signature string (mirrors stream translator behavior).

  **Must NOT do**:
  - Do NOT remove our existing tool ID injection in `antigravity_gemini_request.go`
  - Do NOT touch copilot executor files (Task 2)
  - Do NOT touch antigravity executor files (Task 3)

  **Recommended Agent Profile**:
  - **Category**: `deep` — Highly sensitive, overlaps with existing reasoning filters and tool ID injection
  - **Skills**: [`anilcan-coding`]

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2, 3)
  - **Blocks**: [5]
  - **Blocked By**: None

  **References**:
  - Lemon Patch 003: `/Users/anilcan/Code/ai-help/repos/cliproxyforks/lemon/patches/003-antigravity-claude-thinking-signature-fix.patch`
  - Lemon Patch 004: `/Users/anilcan/Code/ai-help/repos/cliproxyforks/lemon/patches/004-antigravity-assistant-prefill-fix.patch`
  - Lemon Patch 005: `/Users/anilcan/Code/ai-help/repos/cliproxyforks/lemon/patches/005-antigravity-merge-consecutive-turns.patch`
  - Lemon Patch 008: `/Users/anilcan/Code/ai-help/repos/cliproxyforks/lemon/patches/008-streaming-tool-call-deltas.patch`
  - Our antigravity claude request: `internal/translator/antigravity/claude/antigravity_claude_request.go`
  - Our antigravity claude response: `internal/translator/antigravity/claude/antigravity_claude_response.go`
  - Our antigravity gemini request: `internal/translator/antigravity/gemini/antigravity_gemini_request.go`
  - Our claude openai response: `internal/translator/claude/openai/chat-completions/claude_openai_response.go`
  - Lemon's final source files for comparison:
    - `/Users/anilcan/Code/ai-help/repos/cliproxyforks/lemon/internal/translator/antigravity/claude/antigravity_claude_request.go`
    - `/Users/anilcan/Code/ai-help/repos/cliproxyforks/lemon/internal/translator/antigravity/claude/antigravity_claude_response.go`
    - `/Users/anilcan/Code/ai-help/repos/cliproxyforks/lemon/internal/translator/antigravity/gemini/antigravity_gemini_request.go`

  **Acceptance Criteria**:
  - [ ] `go build ./...` succeeds
  - [ ] `enableThoughtTranslate` variable no longer exists in `antigravity_claude_request.go`
  - [ ] Thinking signature caching exists in `antigravity_claude_response.go` non-stream path
  - [ ] Consecutive turn merging logic exists in `antigravity_gemini_request.go`
  - [ ] Assistant prefill handling exists in both claude and gemini request translators
  - [ ] Existing tool ID injection in `antigravity_gemini_request.go` is preserved

  **QA Scenarios (MANDATORY):**
  ```
  Scenario: Go build succeeds after translator changes
    Tool: Bash
    Steps:
      1. Run: go build ./...
      2. Assert: exit code 0
    Expected Result: Clean compilation
    Evidence: .sisyphus/evidence/task-4-build.txt

  Scenario: enableThoughtTranslate removed
    Tool: Bash
    Steps:
      1. Run: grep -c 'enableThoughtTranslate' internal/translator/antigravity/claude/antigravity_claude_request.go
      2. Assert: count = 0
    Expected Result: Flag completely removed
    Evidence: .sisyphus/evidence/task-4-thought-flag.txt

  Scenario: Tool ID injection preserved
    Tool: Bash
    Steps:
      1. Run: grep -c 'generateToolID' internal/translator/antigravity/gemini/antigravity_gemini_request.go
      2. Assert: count > 0
    Expected Result: Tool ID injection still present
    Evidence: .sisyphus/evidence/task-4-tool-id.txt

  Scenario: Consecutive turn merge exists
    Tool: Bash
    Steps:
      1. Run: grep -c 'prevRole' internal/translator/antigravity/gemini/antigravity_gemini_request.go
      2. Assert: count > 0
    Expected Result: Turn merging logic present
    Evidence: .sisyphus/evidence/task-4-merge-turns.txt

  Scenario: Assistant prefill handling exists
    Tool: Bash
    Steps:
      1. Run: grep -c 'Continue from:' internal/translator/antigravity/claude/antigravity_claude_request.go
      2. Run: grep -c 'Continue from:' internal/translator/antigravity/gemini/antigravity_gemini_request.go
      3. Assert: both counts > 0
    Expected Result: Prefill handling in both translators
    Evidence: .sisyphus/evidence/task-4-prefill.txt
  ```

  **Commit**: YES
  - Message: `feat(antigravity): port thinking signature fix, assistant prefill, consecutive turn merge, and streaming tool deltas from lemon fork`
  - Files: `antigravity_claude_request.go`, `antigravity_claude_response.go`, `antigravity_gemini_request.go`, `claude_openai_response.go`
  - Pre-commit: `go build ./...`

---

- [ ] 5. Integration Build + Test Fix

  **What to do**:
  After all Wave 1 tasks complete, resolve any compilation or test failures:
  - Run `go build ./...` — fix any import errors, type mismatches, or missing references
  - Run `go vet ./...` — fix any vet warnings
  - Run `go test ./... -v -count=1` — fix any test failures caused by the parallel changes
  - Common issues to expect:
    - Import conflicts (same package imported differently)
    - Function signature changes not propagated to all call sites
    - Test expectations needing updates for changed behavior

  **Must NOT do**:
  - Do NOT add new features; only fix integration issues
  - Do NOT refactor or reorganize code

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: [`anilcan-coding`]

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 2 (sequential)
  - **Blocks**: [6]
  - **Blocked By**: [1, 2, 3, 4]

  **References**:
  - All files modified by Tasks 1-4
  - Test output from failed builds/tests

  **Acceptance Criteria**:
  - [ ] `go build ./...` succeeds with zero errors
  - [ ] `go vet ./...` reports no issues
  - [ ] `go test ./... -count=1` all tests pass

  **QA Scenarios (MANDATORY):**
  ```
  Scenario: Full build succeeds
    Tool: Bash
    Steps:
      1. Run: go build ./...
      2. Assert: exit code 0
    Expected Result: Zero compilation errors
    Evidence: .sisyphus/evidence/task-5-build.txt

  Scenario: Go vet clean
    Tool: Bash
    Steps:
      1. Run: go vet ./...
      2. Assert: exit code 0
    Expected Result: No vet warnings
    Evidence: .sisyphus/evidence/task-5-vet.txt

  Scenario: All tests pass
    Tool: Bash
    Steps:
      1. Run: go test ./... -count=1 2>&1
      2. Assert: no FAIL lines
    Expected Result: All tests pass
    Evidence: .sisyphus/evidence/task-5-tests.txt
  ```

  **Commit**: YES (only if fixes were needed)
  - Message: `fix: resolve integration issues from lemon fork port`
  - Pre-commit: `go build ./... && go test ./... -count=1`

---

- [ ] 6. Final Verification

  **What to do**:
  Comprehensive verification that everything works:
  - Run full test suite with verbose output
  - Verify all Must Have items from the plan
  - Verify no Must NOT Have violations
  - Check git diff to confirm scope

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: [`anilcan-coding`]

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 2 (after Task 5)
  - **Blocks**: [F1-F4]
  - **Blocked By**: [5]

  **Acceptance Criteria**:
  - [ ] `go build ./...` + `go vet ./...` + `go test ./... -v -count=1` all pass
  - [ ] All Must Have features verified present
  - [ ] All Must NOT Have restrictions verified absent
  - [ ] Git diff shows only expected file changes

  **QA Scenarios (MANDATORY):**
  ```
  Scenario: Complete verification
    Tool: Bash
    Steps:
      1. Run: go build ./... && go vet ./... && go test ./... -v -count=1
      2. Assert: all pass
    Expected Result: Clean build, vet, and test
    Evidence: .sisyphus/evidence/task-6-full-verify.txt

  Scenario: Must Have features present
    Tool: Bash
    Steps:
      1. Run: grep -l 'isCopilotClaudeModel' internal/runtime/executor/github_copilot_executor.go
      2. Run: grep -l 'fetchAntigravityVersion' internal/runtime/executor/antigravity_executor.go
      3. Run: grep -l 'prevRole' internal/translator/antigravity/gemini/antigravity_gemini_request.go
      4. Run: grep -l 'convertPlainClaudeResponseToOpenAI' internal/translator/claude/openai/chat-completions/claude_openai_response.go
      5. Run: test -f .github/workflows/sync-and-build.yml
      6. Run: test -f config.example.custom.yaml
      7. Assert: all found
    Expected Result: All features present
    Evidence: .sisyphus/evidence/task-6-must-have.txt

  Scenario: Must NOT Have violations absent
    Tool: Bash
    Steps:
      1. Run: grep -c 'lemon07r' docker-compose.yml
      2. Assert: count = 0
    Expected Result: No lemon branding in docker-compose
    Evidence: .sisyphus/evidence/task-6-must-not-have.txt
  ```

  **Commit**: NO (verification only)

---

## Final Verification Wave (MANDATORY — after ALL implementation tasks)

> 4 review agents run in PARALLEL. ALL must APPROVE. Rejection → fix → re-run.

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, grep for function names, run `go build`). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in .sisyphus/evidence/. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [ ] F2. **Code Quality Review** — `unspecified-high`
  Run `go vet ./...` + `go build ./...` + `go test ./...`. Review all changed files for: empty error handling, unused imports, commented-out code. Check for AI slop: excessive comments, over-abstraction, generic variable names.
  Output: `Build [PASS/FAIL] | Vet [PASS/FAIL] | Tests [N pass/N fail] | Files [N clean/N issues] | VERDICT`

- [ ] F3. **Real Build + Test QA** — `unspecified-high`
  Start from clean state. Run `go build ./...` and `go test ./... -v -count=1`. Capture full output. Verify no panics, no data races, no compilation errors. Test that new helper functions exist and are callable.
  Output: `Build [PASS/FAIL] | Tests [N/N pass] | VERDICT`

- [ ] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual diff (`git diff`). Verify 1:1 — everything in spec was built (no missing), nothing beyond spec was built (no creep). Check "Must NOT do" compliance — verify existing features (stabilizeResponsesIDs, filterResponsesReasoningItems) are untouched. Flag unaccounted changes.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

- **Wave 1**: Each task gets its own commit
  - T1: `feat: add CI/CD sync workflow and config examples from lemon fork`
  - T2: `feat(copilot): port Claude endpoint routing, unlimited headers, and vision detection from lemon fork`
  - T3: `feat(antigravity): port anti-fingerprinting with dynamic version fetching from lemon fork`
  - T4: `feat(antigravity): port thinking signature fix, assistant prefill, consecutive turn merge, and streaming tool call deltas from lemon fork`
- **Wave 2**:
  - T5: `fix: resolve integration issues from lemon fork port`
  - T6: No commit (verification only)

---

## Success Criteria

### Verification Commands
```bash
go build ./...          # Expected: success, no errors
go vet ./...            # Expected: no issues
go test ./... -v -count=1  # Expected: all tests pass
```

### Final Checklist
- [ ] All "Must Have" present
- [ ] All "Must NOT Have" absent
- [ ] All tests pass
- [ ] CI/CD workflow valid YAML
- [ ] Config examples present and syntactically valid
