# Decisions — lemon-fork-port

## [2026-02-28T14:34:08Z] Session: ses_35b543197ffenZ6OIEINPrr7QL

### Integration Approach
- Direct source modification (not patch-based)
- No annotated patch comments (// +lemon-port)
- Lemon patches at: /Users/anilcan/Code/ai-help/repos/cliproxyforks/lemon/patches/

### Conflict Resolution
- Preserve our existing custom features alongside lemon's changes
- Do not overwrite — integrate/merge logic

### Commit Strategy
- T1: feat: add CI/CD sync workflow, config examples, and Dockerfile patch mechanism from lemon fork
- T2: feat(copilot): port Claude endpoint routing, unlimited headers, and vision detection from lemon fork
- T3: feat(antigravity): port anti-fingerprinting with dynamic version fetching from lemon fork
- T4: feat(antigravity): port thinking signature fix, assistant prefill, consecutive turn merge, and streaming tool deltas from lemon fork
- T5 (if needed): fix: resolve integration issues from lemon fork port
