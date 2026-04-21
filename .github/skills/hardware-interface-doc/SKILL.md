---
name: hardware-interface-doc
description: >
  Reads source code (driver, HAL, communication module) and generates
  hardware interface documentation — register map, protocol, commands,
  init sequence, pin usage — as a Markdown file ready for README.
  Use when asked to: document a hardware interface, generate driver docs,
  extract protocol spec from code, document register map, create interface
  reference from source.
  Do NOT use for: PDF datasheets (use datasheet-digest), code review,
  or generating tests.
---

# SKILL: HARDWARE INTERFACE DOC

You are a senior embedded engineer documenting a hardware interface from source code.
Goal: Read code, extract interface spec, generate clean Markdown. Read-only.

---

## Startup Sequence

1. Check `.hw-interface-doc.yml` in repo root — load if present.
2. Check inline parameters in prompt — highest priority.
3. Apply defaults. Start immediately — no user input needed.

> **Priority order:** Inline → `.hw-interface-doc.yml` → defaults

---

## Parameters

| Parameter          | Default                   | Description                                              |
|-------------------|---------------------------|----------------------------------------------------------|
| `TARGET`          | `""`                      | File or directory to analyze; empty = model detects      |
| `INTERFACE`       | `auto`                    | `auto` \| `i2c` \| `spi` \| `uart` \| `gpio` \| `can`   |
| `OUTPUT_FILE`     | `<module>.interface.md`   | Derived from module/file name if not set                 |
| `INCLUDE_EXAMPLES`| `true`                    | Include usage examples copied from existing code         |
| `INCLUDE_GOTCHAS` | `false`                   | Include gotchas section from TODO/FIXME/HACK comments    |
| `LANG`            | `en`                      | Output language: `en` \| `pl`                            |

---

## Interface Detection

Auto-detect interface type from code signals:

| Interface | Detection signals |
|---|---|
| I2C | `i2c_write`, `i2c_read`, `HAL_I2C`, `Wire.begin`, `I2C_ADDRESS`, `0x` addresses |
| SPI | `spi_transfer`, `HAL_SPI`, `SPI.begin`, `CS_PIN`, `MOSI`/`MISO`/`SCK` |
| UART | `uart_write`, `HAL_UART`, `Serial.begin`, `baudrate`, `USART` |
| GPIO | `gpio_set`, `HAL_GPIO`, `pinMode`, `digitalWrite`, `PORT`/`PIN` defines |
| CAN | `can_send`, `HAL_CAN`, `CAN_ID`, `DLC`, `arbitration_id` |
| Multiple | Document primary interface first, note secondary in one line |

---

## What to Extract

### Always extract (if present in code):

**Identity**
- Module/device name from file name or comments
- Interface type and variant (e.g. I2C 7-bit, SPI mode 0)
- Device address / chip select / baud rate

**Register map** (if applicable)
- Register name, address, R/W/RW type, bit fields, description
- Extract from `#define`, `enum`, `const`, or struct with comments

**Command set** (if applicable)
- Command name, code, direction, parameters, description
- Extract from `#define`, `enum`, function names + docstrings

**Communication protocol**
- Init sequence (ordered steps)
- Read sequence
- Write sequence
- Error handling flow

**Pin usage / electrical**
- Required pins and their roles
- Pull-up/pull-down requirements if mentioned
- Voltage levels if mentioned

**Timing** (if present in code)
- Delays, timeouts, clock frequencies
- Extract from `HAL_Delay`, `_delay_ms`, timeout constants

### Extract if `INCLUDE_EXAMPLES=true`:
- Real usage examples copied verbatim from existing code
- Prefer complete, minimal, self-contained examples
- Max 2 examples per interface

### Extract if `INCLUDE_GOTCHAS=true`:
- TODO / FIXME / HACK / NOTE comments relevant to interface usage
- Format as bulleted "Gotchas" section

### Never include:
- Internal implementation details not relevant to using the interface
- Function bodies (only signatures + docstrings for examples)
- Test code
- Platform-specific boilerplate unrelated to interface

---

## Output Format

Single Markdown file. Structure adapts to what is found — no empty sections.

```markdown
# <Module Name> — Interface Reference

## Interface
<type, address/baud/mode, brief description>

## Register Map
| Register | Address | R/W | Description |
|---|---|---|---|
| REG_CONFIG | 0x1A | RW | Configuration register |

## Command Set
| Command | Code | Direction | Description |
|---|---|---|---|
| CMD_INIT | 0x10 | Host→Dev | Initialize device |

## Init Sequence
1. <step>
2. <step>

## Read Sequence
1. <step>

## Write Sequence
1. <step>

## Pin Usage
| Pin | Role | Notes |
|---|---|---|
| SDA | I2C data | 4.7kΩ pull-up required |

## Timing
- SCL frequency: 100–400 kHz
- Startup delay: 10 ms after VCC stable

## Usage Example
```c
// Initialize MCP2221A I2C at address 0x50
mcp2221_init(0x50, I2C_400KHZ);
uint8_t buf[2];
mcp2221_read(0x50, REG_STATUS, buf, 2);
```

## Gotchas
- <gotcha from TODO/FIXME>
```

---

## File Naming

- Derived from source file or module name: `mcp2221a.interface.md`
- If multiple files analyzed: `<directory>.interface.md`
- Override with `OUTPUT_FILE`

---

## Config File: `.hw-interface-doc.yml`

```yaml
# target: ""             # empty = model detects; e.g. "src/i2c/mcp2221.c"
interface: auto           # auto | i2c | spi | uart | gpio | can
output_file: ""           # empty = derived from module name
include_examples: true
include_gotchas: false
lang: en
```
