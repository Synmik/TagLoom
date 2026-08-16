/**
 * Small shared logger so all console output has a consistent shape:
 *
 *   [TagLoom] <source> message…
 *
 * Usage:
 *   import { logger } from "../utils/logger";
 *   logger.error("files.loadFiles", e);
 *
 * `source` is a stable dot-separated identifier (store/action or
 * component name) so entries are easy to grep for in the DevTools console.
 */

const PREFIX = "[TagLoom]";

type Level = "log" | "info" | "warn" | "error";

function write(level: Level, source: string, ...args: unknown[]): void {
  const fn = console[level];
  fn.call(console, `${PREFIX} ${source}`, ...args);
}

export const logger = {
  /** Informational (default level — scan progress, pool stats, …). */
  log(source: string, ...args: unknown[]): void {
    write("log", source, ...args);
  },
  /** Informational, kept separate for future filtering. */
  info(source: string, ...args: unknown[]): void {
    write("info", source, ...args);
  },
  /** Non-fatal problems (retries, fallbacks, failed best-effort work). */
  warn(source: string, ...args: unknown[]): void {
    write("warn", source, ...args);
  },
  /** Failures worth investigating. */
  error(source: string, ...args: unknown[]): void {
    write("error", source, ...args);
  },
};
