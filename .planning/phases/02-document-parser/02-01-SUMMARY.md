# Summary: Phase 02-01 Core Parser & Text/MD

Implemented the core `Parser` interface and the implementations for TXT and Markdown files.

## Changes
- Created `rag.Parser` interface.
- Implemented `TXTParser` with UTF-8 cleaning.
- Implemented `MDParser` using `goldmark` with custom AST traversal to preserve tables and links.
- Implemented `Validator` using `whatlanggo` for language detection (VI/EN) and 100-char minimum check.

## Verification Results
- `TestTXTParser`: PASS
- `TestMDParser`: PASS (Verified structured rendering of tables and links)
- `TestValidator`: PASS (Successfully detects Russian as invalid, accepts VI/EN)
