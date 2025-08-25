# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Architecture Overview

This is a Go library for parsing InnoDB database pages (16KB pages). The library provides low-level parsing of InnoDB's internal page format, focusing on the compact record format.

### Core Components

**Page Structure Hierarchy:**
1. `InnerPage` - Base 16KB page containing FIL header/trailer and raw data
2. `IndexPage` - Parsed INDEX-type page with header, FSEG header, system records (INFIMUM/SUPREMUM), and directory slots
3. Records are linked via relative offsets in their headers, forming a linked list

**Key Parsing Flow:**
1. `PageReader.ReadPage()` reads raw 16KB page from disk
2. `NewInnerPage()` validates FIL header/trailer and LSN consistency
3. `ParseIndexPage()` extracts index-specific structures for INDEX pages
4. `WalkRecords()` traverses the record linked list

### Data Layout

Pages are 16KB with this structure:
- FIL header (38 bytes) - page metadata, checksums, LSN
- Page header (56 bytes total with FSEG) - index-specific metadata
- Records area - linked list of records starting with INFIMUM/SUPREMUM
- Directory slots (from end, before trailer) - quick record access
- FIL trailer (8 bytes) - checksum and low 32 bits of LSN

All multi-byte values use big-endian encoding. The library only supports compact record format (not redundant format).

## Commands

```bash
# Run tests (when test files are added)
go test ./...

# Build as library
go build

# Format code
go fmt ./...
```