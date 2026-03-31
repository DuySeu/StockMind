# Phase 1: Dependency Injection Assembly - Discussion Log

**Date:** 2026-03-31

**Question 1:**
- **Area:** Tool Configuration
- **Question:** Currently, `DefaultOpenAIConfig` and `DefaultAnthropicConfig` are global variables in the codebase. How should we refactor them to support Dependency Injection?
- **Presented Options:**
  - Inject into Agent Service (Recommended) — main.go loads the config struct and explicitly passes it into agent.NewService(..., config) on startup.
  - Attach to Request Context — Middleware injects the config per HTTP request via Context.
  - Make Immutable Getters — Convert them to functions that return safe copies instead of mutable global maps.
- **User Selection:** Inject into Agent Service
