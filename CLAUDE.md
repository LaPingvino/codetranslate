# codetranslate

LLM-agnostic framework for systematic codebase translation between programming languages.

## Concept

The core problem: LLM sessions translate a few functions, declare victory, and stop. There's no framework to ensure an entire codebase is systematically translated with verification gates.

codetranslate solves this by separating concerns:
- **The framework** manages the inventory, scheduling, and verification
- **The LLM** (any LLM) does the actual translation
- **The compiler** is the judge — if it doesn't compile, it's not done

## Architecture

### Translation Ledger (Dolt)
A versioned database tracking every translatable unit:
- Source function/type/file → target location
- Status: todo / wip / translated / compiles / tested / done / failed
- Which model translated it, how many attempts, last error
- Dependency tier (translate dependencies first)

### Pipeline Loop
```
1. Pick next untranslated unit (respecting dependency order)
2. Gather context (already-translated dependencies, type definitions)
3. Send to LLM with source + context + target language conventions
4. Write output to target file
5. Compile → if fail, retry with error context (up to N times)
6. Run tests if available
7. Update ledger
8. Go to 1 — never stop until ledger is empty
```

### LLM Backends
Simple interface — anything that takes a prompt and returns code:
- Claude API (Haiku for bulk, Sonnet/Opus for hard cases)
- OpenAI API (GPT-4o-mini for bulk)
- Ollama (local models)
- Manual mode (human translates, framework just tracks)

### CLI Commands
```
translate init --source ./path --from c++ --target ./path --to go
translate scan                    # build/update function inventory
translate run                     # main loop: translate, compile, gate, repeat
translate run --model haiku --concurrency 4
translate status                  # ledger summary
translate retry --failed          # re-run failures with different model/context
translate verify                  # compile all, run tests, update ledger
translate show <function>         # show source, translation, status
translate diff                    # dolt diff on the ledger
translate export                  # export ledger to JSON
```

## Key Design Principles

1. **The agent never decides when to stop** — the ledger decides
2. **Compilation is the minimum gate** — not "looks right", actually compiles
3. **LLM-agnostic** — swap models freely, use cheap models for bulk
4. **Dependency-aware** — translate in tier order so context is available
5. **Resumable** — ledger persists across sessions, machines, models
6. **Diffable** — Dolt tracks every change to the ledger and translations
7. **Parallel-safe** — multiple agents can work on different functions concurrently

## Context for the Builder

This tool was conceived to solve real problems in two projects:
- **goloco**: C++ (decompiled OpenLoco) → Go, game engine, ~100+ critical functions
- **The Puzzle Pits**: DOS C (Borland) → modern C with SDL2 compatibility layer

The approach should generalize to any source→target language pair.

### AST Pseudocode Step
For complex codebases (especially decompiled C++), an intermediate AST-to-pseudocode step dramatically improves LLM translation quality. The pipeline should support an optional intermediate representation step before translation.

## Tech Stack
- Go (CLI and framework)
- Dolt (translation ledger — versioned, diffable, syncable)
- Any LLM API for the actual translation work

## Getting Started
```bash
go build -o translate .
./translate init --source ../openloco/src --from c++ --target ../goloco --to go
./translate scan
./translate run --model haiku
```
