---
name: hardware-interface-doc
description: >
  Reads source code (driver, HAL, communication module) and generates
  hardware interface documentation — register map, protocol, commands,
  init sequence — as a Markdown file ready for README.
  Use when asked to: document a hardware interface, generate driver docs,
  extract protocol spec from code, document register map, create interface
  reference from source.
  Do NOT use for: PDF datasheets (use datasheet-digest), code review, tests.
  Invoke explicitly with $hardware-interface-doc or implicitly when prompt matches.
---

# SKILL: HARDWARE INTERFACE DOC

Senior embedded engineer documenting hardware interface from source. Read-only.

---

## Parameters

| Parameter           | Default                 | Description                                          |
|--------------------|-------------------------|------------------------------------------------------|
| `TARGET`           | `""`                    | File/dir to analyze; empty = model detects           |
| `INTERFACE`        | `auto`                  | `auto` \| `i2c` \| `spi` \| `uart` \| `gpio` \| `can` |
| `OUTPUT_FILE`      | `<module>.interface.md` | Derived from module name                             |
| `INCLUDE_EXAMPLES` | `true`                  | Usage examples from existing code                    |
| `INCLUDE_GOTCHAS`  | `false`                 | Gotchas from TODO/FIXME/HACK comments                |
| `LANG`             | `en`                    | `en` \| `pl`                                         |

---

## Interface Detection

| Interface | Signals |
|---|---|
| I2C | `i2c_*`, `HAL_I2C`, `Wire.*`, `I2C_ADDRESS`, `0x` addresses |
| SPI | `spi_*`, `HAL_SPI`, `CS_PIN`, `MOSI/MISO/SCK` |
| UART | `uart_*`, `HAL_UART`, `baudrate`, `USART` |
| GPIO | `gpio_*`, `HAL_GPIO`, `pinMode`, `PORT/PIN` defines |
| CAN | `can_*`, `HAL_CAN`, `CAN_ID`, `DLC` |

---

## Extract Always (if present)

- Device identity: name, interface type, address/baud/mode
- Register map: name, address, R/W, bit fields, description
- Command set: name, code, direction, description
- Init / read / write sequences
- Pin usage and electrical requirements
- Timing constants

## Extract if INCLUDE_EXAMPLES=true
Real usage examples from existing code — verbatim, minimal, max 2.

## Extract if INCLUDE_GOTCHAS=true
TODO/FIXME/HACK comments relevant to interface usage.

## Never include
Internal implementation, function bodies, test code, unrelated boilerplate.

---

## Output

```markdown
# <Module> — Interface Reference
## Interface
## Register Map
## Command Set
## Init Sequence
## Read / Write Sequence
## Pin Usage
## Timing
## Usage Example
## Gotchas  (only if INCLUDE_GOTCHAS=true)
```

Omit empty sections. Structure adapts to what is found in code.
