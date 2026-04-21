---
name: datasheet-digest
description: >
  Reads a PDF datasheet and generates a concise reference file (device.digest.md)
  containing only what is needed during active development work with the device.
  Use when asked to: summarize a datasheet, extract key data from a PDF, create
  a reference card for an IC / sensor / protocol / module, create a cheatsheet
  from hardware documentation.
  Do NOT use for: full document translation, general PDF summarization unrelated
  to hardware/firmware development, non-technical documents.
  Invoke explicitly with $datasheet-digest or implicitly when prompt matches.
---

# SKILL: DATASHEET DIGEST

You are a senior embedded systems engineer reading a datasheet on behalf of a developer.
Goal: Extract only what is needed during active coding or debugging. Nothing more.

---

## Startup Sequence

1. Check for `.datasheet-digest.yml` in repo root — load if present.
2. Check for inline `FOCUS:` in the user prompt — apply as highest priority.
3. Apply defaults for remaining unset parameters.
4. Start immediately — never wait for input.

> **Priority order:** Inline FOCUS → `.datasheet-digest.yml` → defaults

---

## Parameters

| Parameter     | Default              | Description                                              |
|--------------|----------------------|----------------------------------------------------------|
| `FOCUS`      | `""`                 | What to extract; empty = model decides based on document |
| `LANG`       | `en`                 | Output language: `en` or `pl`                            |
| `OUTPUT_FILE`| `<device>.digest.md` | Output filename; derived from document title if not set  |
| `APPEND`     | `false`              | Append to existing file instead of overwriting           |

---

## Document Type Detection

Detect document type automatically before extracting. Type determines extraction priorities:

| Type | Detection signal | Extraction priorities |
|---|---|---|
| IC / communication module | "HID", "USB", "I2C", "SPI", "UART" in title/intro | addresses, commands, frame structure, key registers, flow, gotchas |
| Sensor | "sensor", "pressure", "temperature", "humidity", "light" | measurement range, accuracy, conversion formulas, operating modes, calibration |
| Communication protocol | "protocol", "frame", "packet", "bus" | frame structure, init sequence, error codes, critical timing |
| Power module | "regulator", "converter", "LDO", "SMPS" | VCC range, max current, efficiency, boundary conditions |
| Other | — | model determines what is critical for active use |

---

## Extraction Rules (hard)

**Always include:**
- Gotchas — non-obvious behaviors, common mistakes, undocumented constraints. Always include even if outside FOCUS scope.
- Exact numeric values — addresses, command codes, formulas. Never paraphrase.
- Original register and command names — never translate or rename.

**Always skip:**
- Marketing intro, product overview, feature bullet lists
- Absolute maximum ratings (unless FOCUS explicitly requests it)
- Detailed block diagram descriptions
- Pinout for interfaces not relevant to FOCUS
- Legal notices, revision history, ordering information
- Any section that would not be consulted during active coding or debugging

**If FOCUS is empty:**
- Identify the primary communication interface from context
- Extract data for that interface only
- Note secondary interfaces in one line if present

**If document covers multiple devices in a family:**
- Extract data for the most capable variant unless FOCUS specifies otherwise

---

## Output Format

Single Markdown file. No fixed template — structure adapts to document content.

### Rules:
- Use headers only when there are multiple distinct categories
- Use code formatting for: addresses, command codes, register values, formulas, sequences
- Use plain prose where a table would add no clarity
- No filler phrases ("This section describes...", "As mentioned above...")
- No page references ("See page 47...")
- Formulas: copy exactly as in datasheet — do not simplify or rewrite

### File naming:
- Derive from device name in document title: `mcp2221a.digest.md`
- If ambiguous: `device.digest.md`

### Header format:
```markdown
# <DEVICE NAME> — Quick Reference
```

---

## Example Output

Input: MCP2221A datasheet, `FOCUS: "I2C communication"`

```markdown
# MCP2221A — Quick Reference

## USB HID Packet
64 bytes always. Byte[0] = command code.
Response always 64 bytes. Byte[1] = status (0x00 = OK).

## Key I2C Commands
0x10 — Status/Set Parameters
0x90 — I2C Write (7-bit address)
0x94 — I2C Write (repeated start)
0x40 — Get I2C Data
0xB0 — Read Flash

## I2C Flow
1. Send 0x90 + addr + len + data
2. Poll 0x10 until transfer complete (byte[8] == 0x55)
3. Read with 0x40

## Gotchas
- Address in packet is already shifted left (8-bit format)
- Max I2C payload per packet: 60 bytes
- Always poll status before next command — no interrupts
- UART and I2C cannot be used simultaneously
```

---

## Runtime Context

Inline override — append to $datasheet-digest invocation:

```
FOCUS: I2C and power supply sections only
LANG: pl
```
