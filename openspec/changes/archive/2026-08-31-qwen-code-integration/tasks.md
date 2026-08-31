# Tasks:

> **Checkbox reconciliation 2026-08-31:** all 30 boxes were left unchecked after the original verification reported them complete (drift). Reconciled on the basis of the original verify-report verdict (PASS, all tasks complete, suites green) plus fresh evidence: qwen adapter/assets/model enum present on main, scoped suites exit 0 today. Qwen Code Agent Integration

## Phase 1: Foundation — Types, Constants, Catalog

- [x] T-01 Add `AgentQwenCode AgentID = "qwen-code"` constant to `internal/model/types.go`
- [x] T-02 Add Qwen Code catalog entry to `internal/catalog/agents.go`: `{ID: AgentQwenCode, Name: "Qwen Code", Tier: TierFull, ConfigPath: "~/.qwen"}`
- [x] T-03 Add `all:qwen` to `//go:embed` directive in `internal/assets/assets.go`
- [x] T-04 Create `internal/assets/qwen/` directory for Qwen-specific assets

## Phase 2: Adapter Implementation

- [x] T-05 Create `internal/agents/qwen/adapter.go`:
  - `Adapter` struct with `lookPath` and `statPath` dependency injection
  - `NewAdapter()` constructor
  - `Agent()` returns `model.AgentQwenCode`
  - `Tier()` returns `model.TierFull`
  - `Detect()` — lookPath for `qwen`, stat for `~/.qwen/`
  - `SupportsAutoInstall()` returns `true`
  - `InstallCommand()` — npm with sudo on Linux when `NpmWritable=false`
  - Config path methods: `GlobalConfigDir`, `SystemPromptDir`, `SystemPromptFile` (`QWEN.md`), `SkillsDir`, `SettingsPath`
  - Strategies: `StrategyFileReplace` (system prompt), `StrategyMergeIntoSettings` (MCP)
  - `MCPConfigPath()` returns `settings.json`
  - Capabilities: `SupportsOutputStyles()=false`, `SupportsSlashCommands()=true`, `CommandsDir()` returns `~/.qwen/commands`, `SupportsSkills()=true`, `SupportsSystemPrompt()=true`, `SupportsMCP()=true`
- [x] T-06 Create `internal/assets/qwen/sdd-orchestrator.md`:
  - SDD orchestrator prompt referencing `~/.qwen/skills/` paths
  - Based on Gemini CLI orchestrator with Qwen-specific path substitutions
  - Includes slash command awareness (`SupportsSlashCommands()=true`)

## Phase 3: Registration & Wiring

- [x] T-07 Modify `internal/agents/factory.go`:
  - Import `github.com/gentleman-programming/gentle-ai/internal/agents/qwen`
  - Add `case model.AgentQwenCode: return qwen.NewAdapter(), nil` in `NewAdapter()`
  - Add `model.AgentQwenCode` to `NewDefaultRegistry()` agent list
  - Update registry capacity from 8 to 9
- [x] T-08 Modify `internal/components/sdd/inject.go`:
  - Add `case model.AgentQwenCode: return "qwen/sdd-orchestrator.md"` in `sddOrchestratorAsset()`
- [x] T-09 Modify `internal/components/permissions/inject.go`:
  - Add `qwenCodeOverlayJSON` variable: `{"permissions": {"defaultMode": "auto_edit"}}`
  - Add `case model.AgentQwenCode: return qwenCodeOverlayJSON` in `agentOverlay()`
- [x] T-10 Modify `internal/components/engram/setup.go`:
  - Add `case model.AgentQwenCode: return "qwen-code", true` in `SetupAgentSlug()`
- [x] T-11 Modify `internal/system/config_scan.go`:
  - Add `{Agent: "qwen-code", Path: filepath.Join(homeDir, ".qwen")}` in `knownAgentConfigDirs()`
- [x] T-12 Modify `internal/cli/validate.go`:
  - Add `case string(model.AgentQwenCode)` in agent validation switch
- [x] T-13 Modify `internal/tui/model.go`:
  - Add `case string(model.AgentQwenCode)` in `loadSelection()` switch

## Phase 4: Testing

- [x] T-14 Create `internal/agents/qwen/adapter_test.go`:
  - `TestDetect` — table-driven: binary+config found, both missing, stat error
  - `TestInstallCommand` — table-driven: darwin, linux+sudo, linux+nvm, windows
  - `TestConfigPathsCrossPlatform` — table-driven: all path methods verified
  - `TestCapabilities` — table-driven: all boolean flags + strategies
- [x] T-15 Add `TestInjectQwenCodeWritesSDDOrchestratorAndSkills` to `internal/components/sdd/inject_test.go`:
  - Fresh home directory, call `Inject()`, verify `~/.qwen/QWEN.md` contains SDD content
  - Verify `~/.qwen/skills/sdd-init/SKILL.md` exists
  - Verify `~/.qwen/skills/` path reference in orchestrator
- [x] T-16 Add `AgentQwenCode` case to `TestSDDOrchestratorAssetSelection` in `internal/components/sdd/inject_test.go`:
  - `{agent: model.AgentQwenCode, want: "qwen/sdd-orchestrator.md"}`
- [x] T-17 Add `AgentQwenCode` test case to `TestSetupAgentSlug` in `internal/components/engram/setup_test.go`:
  - `{model.AgentQwenCode, "qwen-code", true}`
- [x] T-18 Update `TestNormalizeInstallFlagsDefaults` in `internal/cli/install_test.go`:
  - Add `model.AgentQwenCode` to expected agents list
  - Update registry capacity from 8 to 9
- [x] T-19 Update `TestDefaultAgentsFromDetection_AllAgentsMappedCorrectly` in `internal/cli/install_test.go`:
  - Add `{"qwen-code", model.AgentQwenCode}` test case
  - Update `makeDetectionWithAgents()` known agents to include `"qwen-code"`
- [x] T-20 Update default registry test in `internal/agents/registry_test.go`:
  - Add `model.AgentQwenCode` to expected agents list
- [x] T-21 Update `makeDetectionWithAgents()` in `internal/tui/model_test.go`:
  - Add `"qwen-code"` to known agents slice

## Phase 5: Build & Verification

- [x] T-22 Run `go build ./...` — must pass with zero errors
- [x] T-23 Run `go vet ./...` — must pass with zero issues
- [x] T-24 Run `go test ./internal/agents/qwen/...` — all adapter tests pass
- [x] T-25 Run `go test ./internal/components/sdd/...` — SDD injection test passes
- [x] T-26 Run `go test ./internal/components/engram/...` — engram setup test passes
- [x] T-27 Run `go test ./internal/cli/...` — install validation tests pass
- [x] T-28 Run `go test ./internal/agents/...` — registry test passes
- [x] T-29 Run `go test ./internal/tui/...` — TUI tests pass
- [x] T-30 Verify `gentle-ai install --agent qwen-code --dry-run` shows correct plan
