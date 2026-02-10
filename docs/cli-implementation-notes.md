# CLI Implementation Guide: Building Rich Interactive Command-Line Interfaces

This document provides architectural patterns, technical approaches, and implementation guidance for building sophisticated command-line interfaces with rich interactivity, based on analysis of a production-grade TypeScript CLI application.

## Table of Contents

1. [Core Libraries & Dependencies](#core-libraries--dependencies)
2. [Architecture & Layering](#architecture--layering)
3. [Interactive Prompts & User Input](#interactive-prompts--user-input)
4. [List Handling & Selection](#list-handling--selection)
5. [Progress Indicators & Spinners](#progress-indicators--spinners)
6. [Tables & Formatted Output](#tables--formatted-output)
7. [Colors & Styling](#colors--styling)
8. [Terminal Utilities](#terminal-utilities)
9. [Error Handling & Validation](#error-handling--validation)
10. [Multi-Step Wizards](#multi-step-wizards)
11. [Best Practices & Design Principles](#best-practices--design-principles)

---

## Core Libraries & Dependencies

### Recommended Library Stack

**Essential Dependencies:**
- **`@clack/prompts`** - Modern prompt library for interactive user input
  - Features: intro/outro messages, select, multiselect, text input, confirm dialogs, spinners
  - Benefits: Beautiful UI, cancellation handling, built-in styling
- **`chalk`** - ANSI terminal color styling
  - Features: RGB/hex color support, nesting, auto-detection
  - Benefits: Wide compatibility, environment-aware color support
- **`commander`** - Command-line argument parsing and command structure
  - Features: Subcommands, options, help generation, version management
  - Benefits: Type-safe, well-maintained, flexible
- **`osc-progress`** - OSC 9001 terminal progress protocol
  - Features: Native terminal progress bars (VS Code, iTerm2, etc.)
  - Benefits: Non-intrusive, works in integrated terminals

### Why This Stack?

1. **`@clack/prompts`** provides excellent UX with minimal code
2. **`chalk`** handles cross-platform color complexity
3. **`commander`** scales from simple to complex CLI structures
4. **`osc-progress`** enables modern terminal features with graceful fallback

---

## Architecture & Layering

### Four-Layer Architecture

```
┌─────────────────────────────────────┐
│   Command Layer                     │  ← Business logic, high-level flows
│   (commands/)                       │
├─────────────────────────────────────┤
│   Wizard Layer                      │  ← Abstract prompt interfaces
│   (wizard/)                         │     Multi-step flows
├─────────────────────────────────────┤
│   CLI Layer                         │  ← Command registration
│   (cli/)                            │     Prompt styling wrappers
├─────────────────────────────────────┤
│   Terminal Layer                    │  ← Low-level utilities
│   (terminal/)                       │     Colors, tables, ANSI, progress
└─────────────────────────────────────┘
```

### Layer Responsibilities

**1. Terminal Layer** - Reusable utilities:
- ANSI code handling (stripping, width calculation)
- Color theme and palette
- Table rendering with intelligent wrapping
- Progress line management
- Terminal capability detection

**2. CLI Layer** - Command infrastructure:
- Command registration with Commander
- Styled prompt wrappers
- Progress factory functions
- CLI-specific utilities (error handling, lifecycle)

**3. Wizard Layer** - Abstract interfaces:
- `WizardPrompter` interface decouples UI from business logic
- Implementations for different backends (Clack, session-based, testing)
- Reusable multi-step flow patterns

**4. Command Layer** - Business logic:
- Specific command implementations
- Uses wizards for user interaction
- Domain-specific validation and processing

### Benefits of Layering

- **Testability:** Mock `WizardPrompter` for unit tests
- **Flexibility:** Swap prompt implementations (CLI vs. web vs. test)
- **Reusability:** Terminal utilities work across all commands
- **Maintainability:** Clear separation of concerns

---

## Interactive Prompts & User Input

### Prompt Abstraction Pattern

Define an abstract interface for all prompt operations:

```typescript
export interface WizardPrompter {
  intro: (title: string) => Promise<void>;
  outro: (message: string) => Promise<void>;
  note: (message: string, title?: string) => Promise<void>;

  select: <T>(params: {
    message: string;
    options: Array<{ value: T; label: string; hint?: string }>;
    initialValue?: T;
  }) => Promise<T>;

  multiselect: <T>(params: {
    message: string;
    options: Array<{ value: T; label: string; hint?: string }>;
    required?: boolean;
  }) => Promise<T[]>;

  text: (params: {
    message: string;
    initialValue?: string;
    placeholder?: string;
    validate?: (value: string) => string | undefined;
  }) => Promise<string>;

  confirm: (params: {
    message: string;
    initialValue?: boolean;
  }) => Promise<boolean>;

  progress: (label: string) => WizardProgress;
}
```

### Styled Wrapper Pattern

Create styled wrappers around library functions to centralize styling:

```typescript
// Styling utilities
const stylePromptMessage = (msg: string) =>
  isColorSupported() ? theme.accent(msg) : msg;

const stylePromptHint = (hint: string) =>
  isColorSupported() ? theme.muted(hint) : hint;

const stylePromptTitle = (title: string) =>
  isColorSupported() ? theme.heading(title) : title;

// Styled wrapper for select
const selectStyled = <T>(params: Parameters<typeof select<T>>[0]) =>
  select({
    ...params,
    message: stylePromptMessage(params.message),
    options: params.options.map((opt) =>
      opt.hint === undefined
        ? opt
        : { ...opt, hint: stylePromptHint(opt.hint) }
    ),
  });
```

**Benefits:**
- Consistent styling across all prompts
- Easy to update theme globally
- Respects color capability and environment variables

### Cancellation Handling

Always guard prompt results with cancellation checks:

```typescript
import { isCancel } from '@clack/prompts';

const result = await selectStyled({
  message: "Choose an option",
  options: [...],
});

if (isCancel(result)) {
  cancel("Operation cancelled.");
  process.exit(0);
}

// Safe to use result here
```

**Custom error for cancellation:**

```typescript
class WizardCancelledError extends Error {
  constructor() {
    super("Wizard cancelled by user");
    this.name = "WizardCancelledError";
  }
}

function guardCancel<T>(value: T | symbol): T {
  if (isCancel(value)) {
    throw new WizardCancelledError();
  }
  return value;
}
```

### TTY Detection Pattern

Check for TTY before showing interactive prompts:

```typescript
if (!process.stdin.isTTY) {
  if (!options.yes) {
    console.error("Interactive mode requires a terminal (TTY).");
    console.error("Use --yes flag for non-interactive execution.");
    process.exit(1);
  }
  // Use defaults/skip prompts in non-interactive mode
  return;
}
```

---

## List Handling & Selection

### Single Selection with Hints

```typescript
const choice = await selectStyled({
  message: "Select environment",
  options: [
    {
      value: "prod",
      label: "Production",
      hint: "Live environment"
    },
    {
      value: "staging",
      label: "Staging",
      hint: "Pre-production testing"
    },
    {
      value: "dev",
      label: "Development",
      hint: "Local development"
    },
  ],
  initialValue: "dev",
});
```

**Key features:**
- Hints provide context without cluttering labels
- Initial value pre-selects default
- Scrollable list (handled by @clack/prompts)

### Multiple Selection

```typescript
const multiselectStyled = <T>(params: Parameters<typeof multiselect<T>>[0]) =>
  multiselect({
    ...params,
    message: stylePromptMessage(params.message),
    options: params.options.map((opt) =>
      opt.hint === undefined
        ? opt
        : { ...opt, hint: stylePromptHint(opt.hint) }
    ),
  });

const selected = await multiselectStyled({
  message: "Select features to enable",
  options: [
    { value: "auth", label: "Authentication" },
    { value: "cache", label: "Caching" },
    { value: "logging", label: "Logging" },
  ],
  required: true, // Prevents empty selection
});
```

### Long List Handling

**@clack/prompts automatically handles:**
- Scrolling through long lists
- Search/filter functionality (type to filter)
- Pagination indicators
- Keyboard navigation (arrow keys, page up/down)

**No additional code needed** - the library manages viewport and scrolling.

### Dynamic Option Generation

```typescript
// Load options dynamically
const items = await fetchAvailableItems();

const choice = await selectStyled({
  message: "Select item",
  options: items.map(item => ({
    value: item.id,
    label: item.name,
    hint: item.description,
  })),
});
```

---

## Progress Indicators & Spinners

### Multi-Mode Progress System

Implement a progress system with multiple fallback modes:

```typescript
type ProgressMode = "osc" | "spinner" | "line" | "log" | "none";

interface ProgressConfig {
  label: string;
  indeterminate?: boolean;
  total?: number;
  enabled?: boolean;
  delayMs?: number;
  stream?: NodeJS.WriteStream;
  fallback?: "spinner" | "line" | "log" | "none";
}

interface ProgressReporter {
  setLabel(label: string): void;
  setPercent(percent: number): void;
  tick(delta?: number): void;
  done(): void;
}
```

### Mode Selection Algorithm

```typescript
function selectProgressMode(config: ProgressConfig): ProgressMode {
  const { stream = process.stdout, fallback } = config;
  const isTty = stream.isTTY ?? false;

  // Priority 1: OSC progress (modern terminals)
  const canOsc = isTty && supportsOscProgress();
  if (canOsc) return "osc";

  // Priority 2: Spinner (TTY with spinner fallback)
  const allowSpinner = isTty &&
    (fallback === undefined || fallback === "spinner");
  if (allowSpinner) return "spinner";

  // Priority 3: Line-based progress (TTY with line fallback)
  const allowLine = isTty && fallback === "line";
  if (allowLine) return "line";

  // Priority 4: Throttled logging (non-TTY with log fallback)
  const allowLog = !isTty && fallback === "log";
  if (allowLog) return "log";

  // Priority 5: No output
  return "none";
}
```

### OSC Progress Implementation

OSC 9001 protocol for native terminal progress bars:

```typescript
class OscProgress implements ProgressReporter {
  constructor(
    private label: string,
    private total: number,
    private stream: NodeJS.WriteStream
  ) {}

  setLabel(label: string): void {
    this.label = label;
    this.render();
  }

  setPercent(percent: number): void {
    const value = Math.round((percent / 100) * this.total);
    this.stream.write(`\x1b]9001;SetProgress;${value};${this.total}\x07`);
  }

  tick(delta = 1): void {
    // Calculate new percent and call setPercent
  }

  done(): void {
    this.stream.write(`\x1b]9001;SetProgress;${this.total};${this.total}\x07`);
  }

  private render(): void {
    this.stream.write(`\x1b]9001;SetLabel;${this.label}\x07`);
  }
}
```

### Spinner Progress

```typescript
import { spinner } from '@clack/prompts';

class SpinnerProgress implements ProgressReporter {
  private s = spinner();

  constructor(label: string) {
    this.s.start(label);
  }

  setLabel(label: string): void {
    this.s.message(label);
  }

  setPercent(percent: number): void {
    // Spinners don't show percent, just update message
  }

  done(): void {
    this.s.stop();
  }
}
```

### Line-Based Progress

Simple percent display with line clearing:

```typescript
class LineProgress implements ProgressReporter {
  private current = 0;

  constructor(
    private label: string,
    private total: number,
    private stream: NodeJS.WriteStream
  ) {}

  setPercent(percent: number): void {
    this.current = percent;
    this.render();
  }

  done(): void {
    this.clear();
  }

  private render(): void {
    const bar = this.createBar(this.current);
    this.stream.write(`\r\x1b[2K${this.label}: ${bar} ${this.current}%`);
  }

  private clear(): void {
    this.stream.write('\r\x1b[2K'); // Clear line
  }

  private createBar(percent: number): string {
    const width = 20;
    const filled = Math.round((percent / 100) * width);
    return '█'.repeat(filled) + '░'.repeat(width - filled);
  }
}
```

### Progress Line Management

Ensure only one active progress line at a time:

```typescript
let activeProgressLine: ProgressReporter | undefined;

export function setActiveProgressLine(line: ProgressReporter): void {
  clearActiveProgressLine();
  activeProgressLine = line;
}

export function clearActiveProgressLine(): void {
  if (activeProgressLine) {
    activeProgressLine.done();
    activeProgressLine = undefined;
  }
}
```

### Wrapper for Async Operations

```typescript
async function withProgress<T>(
  config: ProgressConfig,
  fn: (progress: ProgressReporter) => Promise<T>
): Promise<T> {
  const progress = createProgress(config);
  try {
    return await fn(progress);
  } finally {
    progress.done();
  }
}

// Usage
await withProgress(
  { label: "Installing dependencies", total: 100 },
  async (progress) => {
    for (let i = 0; i <= 100; i += 10) {
      await doWork();
      progress.setPercent(i);
    }
  }
);
```

---

## Tables & Formatted Output

### Table Rendering Architecture

**Core requirements:**
1. ANSI-aware text wrapping (preserve colors across line breaks)
2. Responsive column sizing (flex columns)
3. Alignment options (left, right, center)
4. Word wrapping with intelligent break points
5. Unicode or ASCII border styles

### Table Configuration

```typescript
interface TableColumn<T = Record<string, unknown>> {
  key: keyof T;
  header: string;
  align?: "left" | "right" | "center";
  minWidth?: number;
  maxWidth?: number;
  flex?: boolean; // Expand to fill available width
}

interface TableConfig<T> {
  width?: number; // Total table width (defaults to terminal width)
  columns: TableColumn<T>[];
  rows: T[];
  padding?: number; // Cell padding (default: 1)
  borderStyle?: "unicode" | "ascii";
}
```

### Responsive Width Calculation

```typescript
function calculateColumnWidths<T>(
  config: TableConfig<T>,
  terminalWidth: number
): Map<keyof T, number> {
  const totalWidth = config.width ?? Math.max(60, terminalWidth - 1);
  const padding = config.padding ?? 1;
  const borderChars = 3; // Borders between columns

  // Step 1: Calculate content widths
  const contentWidths = new Map<keyof T, number>();
  for (const col of config.columns) {
    const headerWidth = visibleWidth(col.header);
    const maxContentWidth = Math.max(
      ...config.rows.map(row => visibleWidth(String(row[col.key])))
    );
    contentWidths.set(col.key, Math.max(headerWidth, maxContentWidth));
  }

  // Step 2: Apply constraints (minWidth, maxWidth)
  const constrainedWidths = new Map<keyof T, number>();
  for (const col of config.columns) {
    let width = contentWidths.get(col.key) ?? 0;
    if (col.minWidth) width = Math.max(width, col.minWidth);
    if (col.maxWidth) width = Math.min(width, col.maxWidth);
    constrainedWidths.set(col.key, width);
  }

  // Step 3: Distribute remaining width to flex columns
  const nonFlexWidth = config.columns
    .filter(col => !col.flex)
    .reduce((sum, col) => sum + (constrainedWidths.get(col.key) ?? 0), 0);

  const overhead = (config.columns.length + 1) * borderChars +
                   config.columns.length * 2 * padding;
  const availableForFlex = totalWidth - nonFlexWidth - overhead;

  const flexCols = config.columns.filter(col => col.flex);
  const flexWidth = Math.max(10, Math.floor(availableForFlex / flexCols.length));

  const finalWidths = new Map<keyof T, number>();
  for (const col of config.columns) {
    if (col.flex) {
      finalWidths.set(col.key, flexWidth);
    } else {
      finalWidths.set(col.key, constrainedWidths.get(col.key) ?? 0);
    }
  }

  return finalWidths;
}
```

### ANSI-Aware Text Wrapping

Preserve ANSI codes and hyperlinks when wrapping:

```typescript
const ANSI_REGEX = /\x1b\[[0-9;]*m/g;
const OSC8_REGEX = /\x1b\]8;;[^\x07]*\x07/g;

function stripAnsi(text: string): string {
  return text
    .replace(OSC8_REGEX, "")
    .replace(ANSI_REGEX, "");
}

function visibleWidth(text: string): number {
  return Array.from(stripAnsi(text)).length;
}

interface Token {
  type: "text" | "ansi" | "hyperlink";
  value: string;
  width: number; // Visible width (0 for ANSI/hyperlinks)
}

function tokenize(text: string): Token[] {
  const tokens: Token[] = [];
  let pos = 0;

  // Match ANSI codes, hyperlinks, and text
  const pattern = /(\x1b\[[0-9;]*m|\x1b\]8;;[^\x07]*\x07)/g;
  let match: RegExpExecArray | null;

  while ((match = pattern.exec(text)) !== null) {
    // Add text before match
    if (match.index > pos) {
      const textValue = text.slice(pos, match.index);
      tokens.push({
        type: "text",
        value: textValue,
        width: Array.from(textValue).length,
      });
    }

    // Add ANSI/hyperlink token
    const isHyperlink = match[0].startsWith('\x1b]8');
    tokens.push({
      type: isHyperlink ? "hyperlink" : "ansi",
      value: match[0],
      width: 0,
    });

    pos = match.index + match[0].length;
  }

  // Add remaining text
  if (pos < text.length) {
    const textValue = text.slice(pos);
    tokens.push({
      type: "text",
      value: textValue,
      width: Array.from(textValue).length,
    });
  }

  return tokens;
}

function wrapText(text: string, maxWidth: number): string[] {
  const tokens = tokenize(text);
  const lines: string[] = [];
  let currentLine: Token[] = [];
  let currentWidth = 0;
  let activeAnsi = ""; // Track active ANSI codes

  for (const token of tokens) {
    if (token.type === "ansi") {
      currentLine.push(token);
      activeAnsi = token.value; // Update active style
      continue;
    }

    if (token.type === "hyperlink") {
      currentLine.push(token);
      continue;
    }

    // Text token - may need wrapping
    const words = token.value.split(/(\s+)/);
    for (const word of words) {
      const wordWidth = Array.from(word).length;

      if (currentWidth + wordWidth > maxWidth) {
        // Finish current line
        lines.push(tokensToString(currentLine));

        // Start new line with active ANSI code
        currentLine = activeAnsi ? [{ type: "ansi", value: activeAnsi, width: 0 }] : [];
        currentWidth = 0;
      }

      currentLine.push({ type: "text", value: word, width: wordWidth });
      currentWidth += wordWidth;
    }
  }

  if (currentLine.length > 0) {
    lines.push(tokensToString(currentLine));
  }

  return lines;
}

function tokensToString(tokens: Token[]): string {
  return tokens.map(t => t.value).join("");
}
```

### Table Rendering Example

```typescript
function renderTable<T>(config: TableConfig<T>): string {
  const terminalWidth = process.stdout.columns ?? 80;
  const widths = calculateColumnWidths(config, terminalWidth);
  const lines: string[] = [];

  // Header
  const headerCells = config.columns.map(col => {
    const width = widths.get(col.key) ?? 10;
    return padCell(col.header, width, col.align ?? "left");
  });
  lines.push(`│ ${headerCells.join(" │ ")} │`);

  // Separator
  const separatorCells = config.columns.map(col => {
    const width = widths.get(col.key) ?? 10;
    return "─".repeat(width);
  });
  lines.push(`├─${separatorCells.join("─┼─")}─┤`);

  // Rows
  for (const row of config.rows) {
    const rowCells = config.columns.map(col => {
      const width = widths.get(col.key) ?? 10;
      const value = String(row[col.key]);
      const wrapped = wrapText(value, width);
      return wrapped.map(line => padCell(line, width, col.align ?? "left"));
    });

    // Handle multi-line cells
    const maxLines = Math.max(...rowCells.map(c => c.length));
    for (let i = 0; i < maxLines; i++) {
      const lineCells = rowCells.map(cells => cells[i] ?? " ".repeat(widths.get(config.columns[0].key) ?? 0));
      lines.push(`│ ${lineCells.join(" │ ")} │`);
    }
  }

  return lines.join("\n");
}

function padCell(text: string, width: number, align: "left" | "right" | "center"): string {
  const visible = visibleWidth(text);
  const padding = Math.max(0, width - visible);

  if (align === "right") {
    return " ".repeat(padding) + text;
  } else if (align === "center") {
    const leftPad = Math.floor(padding / 2);
    const rightPad = padding - leftPad;
    return " ".repeat(leftPad) + text + " ".repeat(rightPad);
  } else {
    return text + " ".repeat(padding);
  }
}
```

### Usage Example

```typescript
const tableWidth = Math.max(60, (process.stdout.columns ?? 120) - 1);

console.log(renderTable({
  width: tableWidth,
  columns: [
    { key: "name", header: "Name", minWidth: 14, flex: true },
    { key: "status", header: "Status", minWidth: 10, align: "center" },
    { key: "port", header: "Port", minWidth: 6, align: "right" },
  ],
  rows: [
    { name: "Web Server", status: "Running", port: 8080 },
    { name: "Database", status: "Running", port: 5432 },
    { name: "Cache", status: "Stopped", port: 6379 },
  ],
}));
```

---

## Colors & Styling

### Theme System

Define a centralized color palette:

```typescript
// palette.ts
export const palette = {
  // Primary colors
  accent: "#FF5A2D",        // Orange-red
  accentBright: "#FF7A3D",  // Lighter orange
  accentDim: "#D14A22",     // Darker orange

  // Semantic colors
  info: "#FF8A5B",          // Info orange
  success: "#2FBF71",       // Green
  warn: "#FFB020",          // Yellow
  error: "#E23D2D",         // Red
  muted: "#8B7F77",         // Gray

  // Backgrounds (optional)
  bgAccent: "#2D1810",
  bgInfo: "#1A2030",
  bgSuccess: "#0F2419",
  bgWarn: "#2A1F0F",
  bgError: "#2A0F0F",
};
```

### Theme Factory

```typescript
// theme.ts
import chalk from 'chalk';
import { palette } from './palette.js';

function hex(color: string) {
  return chalk.hex(color);
}

const bold = chalk.bold;

export const theme = {
  // Base colors
  accent: hex(palette.accent),
  accentBright: hex(palette.accentBright),
  accentDim: hex(palette.accentDim),
  info: hex(palette.info),
  success: hex(palette.success),
  warn: hex(palette.warn),
  error: hex(palette.error),
  muted: hex(palette.muted),

  // Composite styles
  heading: bold.hex(palette.accent),
  command: hex(palette.accentBright),
  option: hex(palette.warn),
  link: chalk.underline.hex(palette.info),

  // Semantic helpers
  successText: (text: string) => `${hex(palette.success)("✓")} ${text}`,
  errorText: (text: string) => `${hex(palette.error)("✗")} ${text}`,
  warnText: (text: string) => `${hex(palette.warn)("⚠")} ${text}`,
  infoText: (text: string) => `${hex(palette.info)("ℹ")} ${text}`,
};
```

### Color Capability Detection

Respect environment variables and terminal capabilities:

```typescript
// colors.ts
export function isColorSupported(): boolean {
  // NO_COLOR environment variable (universal standard)
  if (process.env.NO_COLOR !== undefined) {
    return false;
  }

  // FORCE_COLOR environment variable
  if (process.env.FORCE_COLOR !== undefined) {
    return true;
  }

  // Check chalk's auto-detection
  return chalk.level > 0;
}

export function isRichTerminal(): boolean {
  return (process.stdout.isTTY ?? false) && isColorSupported();
}
```

### Styled Utility Functions

```typescript
// Use theme with capability detection
export function styledAccent(text: string): string {
  return isColorSupported() ? theme.accent(text) : text;
}

export function styledSuccess(text: string): string {
  return isColorSupported() ? theme.successText(text) : `✓ ${text}`;
}

export function styledError(text: string): string {
  return isColorSupported() ? theme.errorText(text) : `✗ ${text}`;
}

export function styledMuted(text: string): string {
  return isColorSupported() ? theme.muted(text) : text;
}
```

### Hyperlink Support

OSC-8 terminal hyperlinks (supported in modern terminals):

```typescript
// links.ts
export function terminalLink(text: string, url: string): string {
  if (!isRichTerminal()) {
    return text; // Fallback: just show text
  }

  // OSC-8 format: \x1b]8;;URL\x07TEXT\x1b]8;;\x07
  return `\x1b]8;;${url}\x07${text}\x1b]8;;\x07`;
}

// Usage
console.log(terminalLink("Documentation", "https://example.com/docs"));
```

---

## Terminal Utilities

### Width Detection

```typescript
export function getTerminalWidth(): number {
  return process.stdout.columns ?? 80;
}

export function getMaxContentWidth(): number {
  const columns = getTerminalWidth();
  return Math.max(40, Math.min(88, columns - 10));
}
```

### TTY Checks

```typescript
export function isInteractive(): boolean {
  return process.stdin.isTTY ?? false;
}

export function canShowProgress(): boolean {
  return process.stdout.isTTY ?? false;
}
```

### Terminal State Restoration

Always restore terminal state on exit:

```typescript
// restore.ts
export function createTerminalRestorer() {
  const RESET_SEQUENCE =
    "\x1b[0m" +        // Reset colors
    "\x1b[?25h" +      // Show cursor
    "\x1b[?1000l" +    // Disable mouse tracking
    "\x1b[?1002l" +    // Disable button event tracking
    "\x1b[?1003l" +    // Disable any event tracking
    "\x1b[?1006l" +    // Disable SGR ext mode
    "\x1b[?2004l";     // Disable bracketed paste

  return () => {
    // Clear any active progress line
    clearActiveProgressLine();

    // Exit raw mode if enabled
    if (process.stdin.isTTY && process.stdin.isRaw) {
      process.stdin.setRawMode(false);
    }

    // Resume stdin if paused
    if (process.stdin.isPaused()) {
      process.stdin.resume();
    }

    // Write reset sequence
    process.stdout.write(RESET_SEQUENCE);
  };
}

// Register cleanup handlers
const restoreTerminal = createTerminalRestorer();
process.on("exit", restoreTerminal);
process.on("SIGINT", () => {
  restoreTerminal();
  process.exit(130);
});
process.on("SIGTERM", () => {
  restoreTerminal();
  process.exit(143);
});
```

### Safe Stream Writing

Handle broken pipe errors gracefully:

```typescript
// stream-writer.ts
export function safeWrite(
  stream: NodeJS.WriteStream,
  data: string
): boolean {
  try {
    return stream.write(data);
  } catch (err) {
    if (isErrnoException(err) && err.code === "EPIPE") {
      // Broken pipe - output was closed, suppress error
      return false;
    }
    throw err;
  }
}

function isErrnoException(err: unknown): err is NodeJS.ErrnoException {
  return (
    err instanceof Error &&
    "code" in err &&
    typeof (err as { code?: unknown }).code === "string"
  );
}
```

### Note Formatting with Wrapping

Wrap long messages with bullet point preservation:

```typescript
// note.ts
export function formatNote(
  message: string,
  options: {
    columns?: number;
    maxWidth?: number;
  } = {}
): string {
  const columns = options.columns ?? getTerminalWidth();
  const maxWidth = options.maxWidth ?? getMaxContentWidth();

  const lines = message.split("\n");
  const wrapped: string[] = [];

  for (const line of lines) {
    // Check for bullet points
    const bulletMatch = line.match(/^(\s*[-*•]\s+)/);
    if (bulletMatch) {
      const bullet = bulletMatch[1];
      const text = line.slice(bullet.length);
      const bulletWidth = visibleWidth(bullet);

      // First line includes bullet
      const firstLine = wrapText(text, maxWidth - bulletWidth);
      wrapped.push(bullet + firstLine[0]);

      // Continuation lines indented
      const indent = " ".repeat(bulletWidth);
      for (let i = 1; i < firstLine.length; i++) {
        wrapped.push(indent + firstLine[i]);
      }
    } else {
      // Regular line
      wrapped.push(...wrapText(line, maxWidth));
    }
  }

  return wrapped.join("\n");
}
```

---

## Error Handling & Validation

### Input Validation Pattern

```typescript
async function text(params: {
  message: string;
  validate?: (value: string) => string | undefined;
}): Promise<string> {
  const result = await clackText({
    message: stylePromptMessage(params.message),
    validate: params.validate
      ? (value: string) => {
          const error = params.validate!(value);
          return error ? styleError(error) : undefined;
        }
      : undefined,
  });

  return guardCancel(result);
}
```

### Common Validators

```typescript
// validators.ts
export const validators = {
  required: (value: string) =>
    value.trim() ? undefined : "This field is required",

  email: (value: string) =>
    /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)
      ? undefined
      : "Invalid email address",

  port: (value: string) => {
    const num = parseInt(value, 10);
    return num > 0 && num < 65536
      ? undefined
      : "Port must be between 1 and 65535";
  },

  url: (value: string) => {
    try {
      new URL(value);
      return undefined;
    } catch {
      return "Invalid URL";
    }
  },

  minLength: (min: number) => (value: string) =>
    value.length >= min
      ? undefined
      : `Must be at least ${min} characters`,

  maxLength: (max: number) => (value: string) =>
    value.length <= max
      ? undefined
      : `Must be at most ${max} characters`,
};
```

### Manager Lifecycle Pattern

For operations requiring resource management:

```typescript
export async function withManager<T, R>(params: {
  getManager: () => Promise<{ manager?: T; error?: string }>;
  onMissing: (error?: string) => void;
  run: (manager: T) => Promise<R>;
  close: (manager: T) => Promise<void>;
  onCloseError?: (err: unknown) => void;
}): Promise<R | undefined> {
  const { manager, error } = await params.getManager();

  if (!manager) {
    params.onMissing(error);
    return undefined;
  }

  try {
    return await params.run(manager);
  } finally {
    try {
      await params.close(manager);
    } catch (err) {
      params.onCloseError?.(err);
    }
  }
}

// Usage
await withManager({
  getManager: async () => {
    try {
      const mgr = await connectToService();
      return { manager: mgr };
    } catch (err) {
      return { error: String(err) };
    }
  },
  onMissing: (error) => {
    console.error(theme.errorText(`Failed to connect: ${error}`));
    process.exit(1);
  },
  run: async (manager) => {
    await manager.doWork();
  },
  close: async (manager) => {
    await manager.disconnect();
  },
  onCloseError: (err) => {
    console.warn(theme.warnText(`Cleanup failed: ${err}`));
  },
});
```

### Runtime Abstraction

Decouple from concrete `console` for testability:

```typescript
export interface RuntimeEnv {
  log: (message: string) => void;
  error: (message: string) => void;
  exit: (code: number) => void;
}

export const defaultRuntime: RuntimeEnv = {
  log: (msg) => console.log(msg),
  error: (msg) => console.error(msg),
  exit: (code) => process.exit(code),
};

export const testRuntime = (): RuntimeEnv & {
  logs: string[];
  errors: string[];
  exitCode?: number;
} => {
  const logs: string[] = [];
  const errors: string[] = [];
  let exitCode: number | undefined;

  return {
    logs,
    errors,
    exitCode,
    log: (msg) => logs.push(msg),
    error: (msg) => errors.push(msg),
    exit: (code) => { exitCode = code; },
  };
};
```

---

## Multi-Step Wizards

### Wizard Flow Pattern

```typescript
type WizardStep =
  | { type: "intro"; title: string }
  | { type: "select"; message: string; options: SelectOption[]; key: string }
  | { type: "text"; message: string; key: string; validate?: Validator }
  | { type: "confirm"; message: string; key: string }
  | { type: "note"; message: string; title?: string }
  | { type: "outro"; message: string };

interface WizardResult {
  [key: string]: unknown;
}

async function runWizard(
  steps: WizardStep[],
  prompter: WizardPrompter
): Promise<WizardResult> {
  const result: WizardResult = {};

  for (const step of steps) {
    switch (step.type) {
      case "intro":
        await prompter.intro(step.title);
        break;

      case "select":
        result[step.key] = await prompter.select({
          message: step.message,
          options: step.options,
        });
        break;

      case "text":
        result[step.key] = await prompter.text({
          message: step.message,
          validate: step.validate,
        });
        break;

      case "confirm":
        result[step.key] = await prompter.confirm({
          message: step.message,
        });
        break;

      case "note":
        await prompter.note(step.message, step.title);
        break;

      case "outro":
        await prompter.outro(step.message);
        break;
    }
  }

  return result;
}
```

### Conditional Branching

```typescript
async function runConditionalWizard(
  prompter: WizardPrompter
): Promise<WizardResult> {
  const result: WizardResult = {};

  await prompter.intro("Setup Wizard");

  // Step 1: Choose mode
  const mode = await prompter.select({
    message: "Setup mode",
    options: [
      { value: "quick", label: "Quick Start", hint: "Recommended" },
      { value: "advanced", label: "Advanced", hint: "Full customization" },
    ],
  });
  result.mode = mode;

  // Step 2: Conditional branching
  if (mode === "quick") {
    // Quick mode: minimal questions
    result.useDefaults = true;
    await prompter.note("Using recommended defaults");
  } else {
    // Advanced mode: detailed configuration
    result.port = await prompter.text({
      message: "Port number",
      validate: validators.port,
    });

    result.enableAuth = await prompter.confirm({
      message: "Enable authentication?",
      initialValue: true,
    });

    if (result.enableAuth) {
      result.authMethod = await prompter.select({
        message: "Authentication method",
        options: [
          { value: "oauth", label: "OAuth 2.0" },
          { value: "apikey", label: "API Key" },
        ],
      });
    }
  }

  // Step 3: Confirmation
  const proceed = await prompter.confirm({
    message: "Proceed with setup?",
    initialValue: true,
  });

  if (!proceed) {
    throw new WizardCancelledError();
  }

  await prompter.outro("Setup complete!");

  return result;
}
```

### Progress Integration in Wizards

```typescript
async function wizardWithProgress(
  prompter: WizardPrompter
): Promise<void> {
  await prompter.intro("Installation Wizard");

  const features = await prompter.multiselect({
    message: "Select features to install",
    options: [
      { value: "db", label: "Database" },
      { value: "cache", label: "Cache" },
      { value: "logging", label: "Logging" },
    ],
  });

  // Show progress during installation
  const progress = prompter.progress("Installing features...");

  const total = features.length;
  for (let i = 0; i < features.length; i++) {
    progress.setLabel(`Installing ${features[i]}...`);
    await installFeature(features[i]);
    progress.setPercent(((i + 1) / total) * 100);
  }

  progress.done();

  await prompter.outro("Installation complete!");
}
```

### Session-Based Wizard (Client-Server)

For remote/detached wizard execution:

```typescript
interface WizardSession {
  id: string;
  steps: WizardStep[];
  results: WizardResult;
  currentStep: number;
  completed: boolean;
}

class SessionWizardPrompter implements WizardPrompter {
  constructor(
    private session: WizardSession,
    private sendToClient: (step: WizardStep) => Promise<unknown>
  ) {}

  async select<T>(params: SelectParams<T>): Promise<T> {
    const step: WizardStep = {
      type: "select",
      message: params.message,
      options: params.options,
      key: `step_${this.session.currentStep}`,
    };

    const result = await this.sendToClient(step);
    this.session.results[step.key] = result;
    this.session.currentStep++;

    return result as T;
  }

  // Similar implementations for other methods...
}
```

---

## Best Practices & Design Principles

### 1. Layered Architecture

**Principle:** Separate concerns into distinct layers.

- **Terminal Layer:** Pure utilities, no business logic
- **CLI Layer:** Command structure, prompt wrappers
- **Wizard Layer:** Abstract interfaces, reusable flows
- **Command Layer:** Business logic, domain operations

**Benefits:**
- Easy testing (mock interfaces)
- Flexible implementations (swap backends)
- Clear dependencies (bottom-up)

### 2. Graceful Degradation

**Principle:** Always provide fallbacks for limited environments.

```typescript
// Example: Progress fallback chain
OSC Progress → Spinner → Line-based → Logging → None

// Example: Color fallback
Colors → Plain text (NO_COLOR respected)

// Example: Interactive → Non-interactive
Prompts → CLI flags or defaults
```

### 3. Environment Awareness

**Principle:** Detect and respect environment capabilities.

```typescript
// Check TTY before interactive prompts
if (!process.stdin.isTTY) { /* fallback */ }

// Respect NO_COLOR
if (process.env.NO_COLOR) { /* disable colors */ }

// Detect terminal width
const width = process.stdout.columns ?? 80;
```

### 4. Error Recovery

**Principle:** Always clean up resources and restore state.

```typescript
// Terminal state restoration on exit
process.on("exit", restoreTerminal);
process.on("SIGINT", cleanup);

// Manager lifecycle pattern
try {
  await run(manager);
} finally {
  await close(manager);
}
```

### 5. User Experience

**Principle:** Prioritize clarity and feedback.

- **Clear messages:** Use styled headings and hints
- **Progress feedback:** Show progress for long operations
- **Cancellation:** Allow Ctrl+C at any time
- **Validation:** Provide immediate feedback on errors
- **Confirmation:** Ask before destructive operations

### 6. Abstraction Over Concretion

**Principle:** Code to interfaces, not implementations.

```typescript
// Good: Interface-based
function runCommand(prompter: WizardPrompter) { }

// Bad: Implementation-coupled
function runCommand() {
  const result = clackSelect(...); // Tightly coupled
}
```

**Benefits:**
- Testable (mock prompter)
- Flexible (swap implementations)
- Reusable (works with any prompter)

### 7. Responsive Design

**Principle:** Adapt to available terminal space.

```typescript
// Flex columns expand/contract
{ key: "name", flex: true }

// Constrain widths
{ key: "id", minWidth: 6, maxWidth: 12 }

// Responsive table width
const width = Math.max(60, process.stdout.columns - 1);
```

### 8. ANSI Awareness

**Principle:** Preserve formatting across operations.

- Strip ANSI for width calculation
- Preserve ANSI codes when wrapping
- Track active styles across line breaks
- Handle hyperlinks (OSC-8) correctly

### 9. Centralized Styling

**Principle:** Single source of truth for colors and themes.

```typescript
// Define once
const theme = createTheme(palette);

// Use everywhere
theme.accent(text);
theme.successText(text);
```

**Benefits:**
- Consistent styling
- Easy theme changes
- Respects color capabilities

### 10. Safe Defaults

**Principle:** Provide sensible defaults, allow overrides.

```typescript
const columns = options.columns ?? process.stdout.columns ?? 80;
const maxWidth = options.maxWidth ?? 88;
const padding = options.padding ?? 1;
```

---

## Implementation Checklist

When implementing a new CLI:

### Core Setup
- [ ] Install dependencies: `@clack/prompts`, `chalk`, `commander`, `osc-progress`
- [ ] Create directory structure: `terminal/`, `cli/`, `wizard/`, `commands/`
- [ ] Define color palette and theme
- [ ] Implement terminal utilities (width, TTY detection, ANSI handling)

### Prompting System
- [ ] Define `WizardPrompter` interface
- [ ] Implement Clack-based prompter with styling
- [ ] Create styled wrapper functions
- [ ] Implement cancellation handling

### Progress System
- [ ] Implement multi-mode progress (OSC, spinner, line, log)
- [ ] Create progress line manager (single active line)
- [ ] Build `withProgress` wrapper utility

### Output Formatting
- [ ] Implement ANSI-aware text wrapping
- [ ] Build table renderer with responsive columns
- [ ] Create note formatter with bullet preservation
- [ ] Add hyperlink support (OSC-8)

### Error Handling
- [ ] Create validation helpers
- [ ] Implement manager lifecycle pattern
- [ ] Build runtime abstraction for testing
- [ ] Add terminal state restoration

### Command Structure
- [ ] Set up Commander with subcommands
- [ ] Register signal handlers (SIGINT, SIGTERM)
- [ ] Add `--yes` flag for non-interactive mode
- [ ] Implement help and version commands

### Testing
- [ ] Create mock `WizardPrompter` for tests
- [ ] Test with `NO_COLOR=1`
- [ ] Test non-TTY mode (pipe output)
- [ ] Test terminal state restoration

---

## Additional Resources

### Terminal Escape Sequences

**ANSI SGR (colors/styles):**
- Format: `\x1b[<params>m`
- Reset: `\x1b[0m`
- Bold: `\x1b[1m`
- 256-color: `\x1b[38;5;<n>m` (foreground), `\x1b[48;5;<n>m` (background)
- RGB: `\x1b[38;2;<r>;<g>;<b>m`

**OSC-8 (hyperlinks):**
- Format: `\x1b]8;;URL\x07TEXT\x1b]8;;\x07`
- Support: VS Code, iTerm2, terminals

**OSC 9001 (progress):**
- Set progress: `\x1b]9001;SetProgress;value;total\x07`
- Set label: `\x1b]9001;SetLabel;text\x07`
- Support: VS Code integrated terminal

**Cursor control:**
- Clear line: `\x1b[2K`
- Carriage return: `\r`
- Show cursor: `\x1b[?25h`
- Hide cursor: `\x1b[?25l`

### Environment Variables

- `NO_COLOR`: Disable all color output (standard)
- `FORCE_COLOR`: Force color output
- `CI=true`: Continuous integration environment
- `TERM`: Terminal type (check for capabilities)

---

## Conclusion

This guide provides a comprehensive foundation for building rich, interactive command-line interfaces. Key takeaways:

1. **Layer your architecture** - Separate terminal utilities, CLI infrastructure, wizards, and commands
2. **Abstract prompting** - Use interfaces to decouple UI from logic
3. **Support all environments** - TTY, non-TTY, color, no-color
4. **Handle ANSI properly** - Strip for width, preserve for wrapping
5. **Provide fallbacks** - Progress, colors, interactivity
6. **Clean up resources** - Restore terminal state, close managers
7. **Centralize styling** - Single theme, consistent application
8. **Be responsive** - Adapt to terminal width
9. **Give feedback** - Progress, validation, confirmation
10. **Test thoroughly** - Mock interfaces, check edge cases

By following these patterns, you'll build CLIs that are robust, user-friendly, and maintainable.
