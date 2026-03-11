package executor

import (
	"encoding/json"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/thinking"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/executor"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// applyPayloadConfigWithRoot behaves like applyPayloadConfig but treats all parameter
// paths as relative to the provided root path (for example, "request" for Gemini CLI)
// and restricts matches to the given protocol when supplied. Defaults are checked
// against the original payload when provided. requestedModel carries the client-visible
// model name before alias resolution so payload rules can target aliases precisely.
func applyPayloadConfigWithRoot(cfg *config.Config, model, protocol, root string, payload, original []byte, requestedModel string) []byte {
	if cfg == nil || len(payload) == 0 {
		return payload
	}
	rules := cfg.Payload
	if len(rules.Default) == 0 && len(rules.DefaultRaw) == 0 && len(rules.Override) == 0 && len(rules.OverrideRaw) == 0 && len(rules.Filter) == 0 {
		return payload
	}
	model = strings.TrimSpace(model)
	requestedModel = strings.TrimSpace(requestedModel)
	if model == "" && requestedModel == "" {
		return payload
	}
	candidates := payloadModelCandidates(model, requestedModel)
	out := payload
	source := original
	if len(source) == 0 {
		source = payload
	}
	appliedDefaults := make(map[string]struct{})
	// Apply default rules: first write wins per field across all matching rules.
	for i := range rules.Default {
		rule := &rules.Default[i]
		if !payloadModelRulesMatch(rule.Models, protocol, candidates) {
			continue
		}
		for path, value := range rule.Params {
			fullPath := buildPayloadPath(root, path)
			if fullPath == "" {
				continue
			}
			if gjson.GetBytes(source, fullPath).Exists() {
				continue
			}
			if _, ok := appliedDefaults[fullPath]; ok {
				continue
			}
			updated, errSet := sjson.SetBytes(out, fullPath, value)
			if errSet != nil {
				continue
			}
			out = updated
			appliedDefaults[fullPath] = struct{}{}
		}
	}
	// Apply default raw rules: first write wins per field across all matching rules.
	for i := range rules.DefaultRaw {
		rule := &rules.DefaultRaw[i]
		if !payloadModelRulesMatch(rule.Models, protocol, candidates) {
			continue
		}
		for path, value := range rule.Params {
			fullPath := buildPayloadPath(root, path)
			if fullPath == "" {
				continue
			}
			if gjson.GetBytes(source, fullPath).Exists() {
				continue
			}
			if _, ok := appliedDefaults[fullPath]; ok {
				continue
			}
			rawValue, ok := payloadRawValue(value)
			if !ok {
				continue
			}
			updated, errSet := sjson.SetRawBytes(out, fullPath, rawValue)
			if errSet != nil {
				continue
			}
			out = updated
			appliedDefaults[fullPath] = struct{}{}
		}
	}
	// Apply override rules: last write wins per field across all matching rules.
	for i := range rules.Override {
		rule := &rules.Override[i]
		if !payloadModelRulesMatch(rule.Models, protocol, candidates) {
			continue
		}
		for path, value := range rule.Params {
			fullPath := buildPayloadPath(root, path)
			if fullPath == "" {
				continue
			}
			updated, errSet := sjson.SetBytes(out, fullPath, value)
			if errSet != nil {
				continue
			}
			out = updated
		}
	}
	// Apply override raw rules: last write wins per field across all matching rules.
	for i := range rules.OverrideRaw {
		rule := &rules.OverrideRaw[i]
		if !payloadModelRulesMatch(rule.Models, protocol, candidates) {
			continue
		}
		for path, value := range rule.Params {
			fullPath := buildPayloadPath(root, path)
			if fullPath == "" {
				continue
			}
			rawValue, ok := payloadRawValue(value)
			if !ok {
				continue
			}
			updated, errSet := sjson.SetRawBytes(out, fullPath, rawValue)
			if errSet != nil {
				continue
			}
			out = updated
		}
	}
	// Apply filter rules: remove matching paths from payload.
	for i := range rules.Filter {
		rule := &rules.Filter[i]
		if !payloadModelRulesMatch(rule.Models, protocol, candidates) {
			continue
		}
		for _, path := range rule.Params {
			fullPath := buildPayloadPath(root, path)
			if fullPath == "" {
				continue
			}
			updated, errDel := sjson.DeleteBytes(out, fullPath)
			if errDel != nil {
				continue
			}
			out = updated
		}
	}
	return out
}

func payloadModelRulesMatch(rules []config.PayloadModelRule, protocol string, models []string) bool {
	if len(rules) == 0 || len(models) == 0 {
		return false
	}
	for _, model := range models {
		for _, entry := range rules {
			name := strings.TrimSpace(entry.Name)
			if name == "" {
				continue
			}
			if ep := strings.TrimSpace(entry.Protocol); ep != "" && protocol != "" && !strings.EqualFold(ep, protocol) {
				continue
			}
			if matchModelPattern(name, model) {
				return true
			}
		}
	}
	return false
}

func payloadModelCandidates(model, requestedModel string) []string {
	model = strings.TrimSpace(model)
	requestedModel = strings.TrimSpace(requestedModel)
	if model == "" && requestedModel == "" {
		return nil
	}
	candidates := make([]string, 0, 3)
	seen := make(map[string]struct{}, 3)
	addCandidate := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		candidates = append(candidates, value)
	}
	if model != "" {
		addCandidate(model)
	}
	if requestedModel != "" {
		parsed := thinking.ParseSuffix(requestedModel)
		base := strings.TrimSpace(parsed.ModelName)
		if base != "" {
			addCandidate(base)
		}
		if parsed.HasSuffix {
			addCandidate(requestedModel)
		}
	}
	return candidates
}

// buildPayloadPath combines an optional root path with a relative parameter path.
// When root is empty, the parameter path is used as-is. When root is non-empty,
// the parameter path is treated as relative to root.
func buildPayloadPath(root, path string) string {
	r := strings.TrimSpace(root)
	p := strings.TrimSpace(path)
	if r == "" {
		return p
	}
	if p == "" {
		return r
	}
	if strings.HasPrefix(p, ".") {
		p = p[1:]
	}
	return r + "." + p
}

func payloadRawValue(value any) ([]byte, bool) {
	if value == nil {
		return nil, false
	}
	switch typed := value.(type) {
	case string:
		return []byte(typed), true
	case []byte:
		return typed, true
	default:
		raw, errMarshal := json.Marshal(typed)
		if errMarshal != nil {
			return nil, false
		}
		return raw, true
	}
}

func payloadRequestedModel(opts cliproxyexecutor.Options, fallback string) string {
	fallback = strings.TrimSpace(fallback)
	if len(opts.Metadata) == 0 {
		return fallback
	}
	raw, ok := opts.Metadata[cliproxyexecutor.RequestedModelMetadataKey]
	if !ok || raw == nil {
		return fallback
	}
	switch v := raw.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return fallback
		}
		return strings.TrimSpace(v)
	case []byte:
		if len(v) == 0 {
			return fallback
		}
		trimmed := strings.TrimSpace(string(v))
		if trimmed == "" {
			return fallback
		}
		return trimmed
	default:
		return fallback
	}
}

// matchModelPattern performs simple wildcard matching where '*' matches zero or more characters.
// Examples:
//
//	"*-5" matches "gpt-5"
//	"gpt-*" matches "gpt-5" and "gpt-4"
//	"gemini-*-pro" matches "gemini-2.5-pro" and "gemini-3-pro".
func matchModelPattern(pattern, model string) bool {
	pattern = strings.TrimSpace(pattern)
	model = strings.TrimSpace(model)
	if pattern == "" {
		return false
	}
	if pattern == "*" {
		return true
	}
	// Iterative glob-style matcher supporting only '*' wildcard.
	pi, si := 0, 0
	starIdx := -1
	matchIdx := 0
	for si < len(model) {
		if pi < len(pattern) && (pattern[pi] == model[si]) {
			pi++
			si++
			continue
		}
		if pi < len(pattern) && pattern[pi] == '*' {
			starIdx = pi
			matchIdx = si
			pi++
			continue
		}
		if starIdx != -1 {
			pi = starIdx + 1
			matchIdx++
			si = matchIdx
			continue
		}
		return false
	}
	for pi < len(pattern) && pattern[pi] == '*' {
		pi++
	}
	return pi == len(pattern)
}

// applySystemPromptRules injects system prompts into model payloads based on config rules.
//
// The function evaluates SystemPrompts rules against the provided model and protocol,
// calling the appropriate protocol-specific injector when a match is found.
// If SystemPrompts is empty or the payload is empty, the payload is returned unchanged.
//
// Parameters:
// - cfg: global configuration containing SystemPrompts rules
// - model: the resolved model name (e.g., "claude-3-sonnet")
// - protocol: the translator protocol (e.g., "claude", "openai")
// - payload: the JSON request payload to modify
//
// Returns: the modified payload with injected system prompts, or original if no rules match.
func applySystemPromptRules(
	cfg *config.Config,
	model,
	protocol string,
	payload []byte,
	requestedModel string,
) []byte {
	// 1. Guard: empty config, payload, or no rules → return unchanged.
	if cfg == nil || len(payload) == 0 {
		return payload
	}
	if len(cfg.Payload.SystemPrompts) == 0 {
		return payload
	}

	// 2. Normalize and validate inputs.
	model = strings.TrimSpace(model)
	protocol = strings.TrimSpace(protocol)
	requestedModel = strings.TrimSpace(requestedModel)
	if model == "" && requestedModel == "" {
		return payload
	}

	// 3. Build model candidates for matching (includes both upstream and alias names).
	candidates := payloadModelCandidates(model, requestedModel)

	// 4. Iterate through rules and apply matching injectors.
	out := payload
	for i := range cfg.Payload.SystemPrompts {
		rule := &cfg.Payload.SystemPrompts[i]

		// Skip if models don't match.
		if !payloadModelRulesMatch(rule.Models, protocol, candidates) {
			continue
		}

		// Skip if prompt is empty.
		prompt := strings.TrimSpace(rule.EffectivePrompt())
		if prompt == "" {
			log.Warnf("[SystemPrompt] Rule %d: matched but prompt is empty", i)
			continue
		}

		// Normalize mode; default to "prepend".
		mode := strings.TrimSpace(rule.Mode)
		if mode == "" {
			mode = "prepend"
		}

		// Route to protocol-specific injector.
		switch protocol {
		case "claude":
			out = injectClaudeSystemPrompt(out, prompt, mode)
		case "openai":
			out = injectOpenAISystemPrompt(out, prompt, mode)
		}
	}

	return out
}

// injectClaudeSystemPrompt injects a system prompt into Claude-format payloads.
//
// Claude API uses a "system" field containing an array of content blocks.
// Each block has format: {"type":"text","text":"content"}.
//
// Mode behavior:
// - "prepend": Insert new block at beginning of system array
// - "append": Add new block at end of system array
// - "replace": Replace entire system array with single text block
//
// If the system field doesn't exist, it is created. If it's not an array,
// it is gracefully handled based on mode (replaced or wrapped).
//
// Parameters:
// - payload: the Claude-format JSON request payload
// - prompt: the system prompt text to inject
// - mode: injection mode ("prepend", "append", or "replace")
//
// Returns: modified payload with injected system prompt.
func injectClaudeSystemPrompt(payload []byte, prompt, mode string) []byte {
	if len(payload) == 0 || prompt == "" {
		return payload
	}

	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = "prepend"
	}

	// Get current system field (array of blocks).
	systemValue := gjson.GetBytes(payload, "system")

	switch mode {
	case "replace":
		// 1. Replace entire system with single text block.
		newBlock := map[string]string{
			"type": "text",
			"text": prompt,
		}
		updated, err := sjson.SetBytes(
			payload,
			"system",
			[]map[string]string{newBlock},
		)
		if err != nil {
			return payload
		}
		return updated

	case "append":
		// 1. Get existing system array or start empty.
		var blocks []interface{}
		if systemValue.IsArray() {
			// Parse existing blocks.
			raw := systemValue.Raw
			var parsed []interface{}
			if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
				blocks = parsed
			}
		}

		// 2. Append new block.
		newBlock := map[string]string{
			"type": "text",
			"text": prompt,
		}
		blocks = append(blocks, newBlock)

		// 3. Write back.
		updated, err := sjson.SetBytes(payload, "system", blocks)
		if err != nil {
			return payload
		}
		return updated

	case "prepend":
		fallthrough

	default:
		// 1. Get existing system array or start empty.
		var blocks []interface{}
		if systemValue.IsArray() {
			// Parse existing blocks.
			raw := systemValue.Raw
			var parsed []interface{}
			if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
				blocks = parsed
			}
		}

		// 2. Prepend new block.
		newBlock := map[string]string{
			"type": "text",
			"text": prompt,
		}
		blocks = append(
			[]interface{}{newBlock},
			blocks...,
		)

		// 3. Write back.
		updated, err := sjson.SetBytes(payload, "system", blocks)
		if err != nil {
			return payload
		}
		return updated
	}
}

// injectOpenAISystemPrompt injects a system prompt into OpenAI-format payloads.
//
// OpenAI API uses a "messages" array with role-based message objects.
// System prompts are messages with role: "system" and content: "text".
//
// Mode behavior:
// - "prepend": Insert new system message at beginning of messages array
// - "append": Add new system message at end of messages array
// - "replace": Remove all existing system messages, add one new system message
//
// If the messages field doesn't exist or is not an array, a new array is created.
// Existing system messages are preserved unless mode is "replace".
//
// Parameters:
// - payload: the OpenAI-format JSON request payload
// - prompt: the system prompt text to inject
// - mode: injection mode ("prepend", "append", or "replace")
//
// Returns: modified payload with injected system prompt.
func injectOpenAISystemPrompt(payload []byte, prompt, mode string) []byte {
	if len(payload) == 0 || prompt == "" {
		return payload
	}

	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = "prepend"
	}

	// Get current messages array.
	messagesValue := gjson.GetBytes(payload, "messages")

	switch mode {
	case "replace":
		// 1. Remove all existing system messages.
		var messages []interface{}
		if messagesValue.IsArray() {
			raw := messagesValue.Raw
			var parsed []interface{}
			if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
				// Filter out system messages.
				for _, msg := range parsed {
					msgMap, ok := msg.(map[string]interface{})
					if ok {
						if role, hasRole := msgMap["role"]; hasRole {
							if roleStr, isStr := role.(string); !isStr ||
								roleStr != "system" {
								messages = append(messages, msg)
							}
						}
					}
				}
			}
		}

		// 2. Prepend new system message.
		newMsg := map[string]string{
			"role":    "system",
			"content": prompt,
		}
		messages = append(
			[]interface{}{newMsg},
			messages...,
		)

		// 3. Write back.
		updated, err := sjson.SetBytes(payload, "messages", messages)
		if err != nil {
			return payload
		}
		return updated

	case "append":
		// 1. Get existing messages array or start empty.
		var messages []interface{}
		if messagesValue.IsArray() {
			raw := messagesValue.Raw
			var parsed []interface{}
			if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
				messages = parsed
			}
		}

		// 2. Append new system message.
		newMsg := map[string]string{
			"role":    "system",
			"content": prompt,
		}
		messages = append(messages, newMsg)

		// 3. Write back.
		updated, err := sjson.SetBytes(payload, "messages", messages)
		if err != nil {
			return payload
		}
		return updated

	case "prepend":
		fallthrough

	default:
		// 1. Get existing messages array or start empty.
		var messages []interface{}
		if messagesValue.IsArray() {
			raw := messagesValue.Raw
			var parsed []interface{}
			if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
				messages = parsed
			}
		}

		// 2. Prepend new system message.
		newMsg := map[string]string{
			"role":    "system",
			"content": prompt,
		}
		messages = append(
			[]interface{}{newMsg},
			messages...,
		)

		// 3. Write back.
		updated, err := sjson.SetBytes(payload, "messages", messages)
		if err != nil {
			return payload
		}
		return updated
	}
}
