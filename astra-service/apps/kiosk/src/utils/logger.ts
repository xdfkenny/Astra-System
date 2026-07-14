type LogLevel = "debug" | "info" | "warn" | "error";

const LOG_LEVELS: Record<LogLevel, number> = {
  debug: 0,
  info: 1,
  warn: 2,
  error: 3,
};

const ENV = import.meta.env as Record<string, string | undefined>;

function currentLevel(): number {
  const raw = ENV["VITE_LOG_LEVEL"] ?? "warn";
  const mapped = (LOG_LEVELS as Record<string, number>)[raw];
  return mapped ?? LOG_LEVELS.warn;
}

function formatTimestamp(): string {
  return new Date().toISOString();
}

function serializeError(error: unknown): Record<string, unknown> {
  if (error instanceof Error) {
    return {
      name: error.name,
      message: error.message,
      stack: error.stack,
    };
  }
  return { message: String(error) };
}

class Logger {
  private name: string;
  private level: number;

  constructor(name: string) {
    this.name = name;
    this.level = currentLevel();
  }

  shouldLog(level: LogLevel): boolean {
    return LOG_LEVELS[level] >= this.level;
  }

  private log(
    level: LogLevel,
    message: string,
    context?: Record<string, unknown>,
  ): void {
    if (!this.shouldLog(level)) return;

    const entry: Record<string, unknown> = {
      timestamp: formatTimestamp(),
      level,
      logger: this.name,
      message,
    };

    if (context) {
      entry["context"] = context;
    }

    const consoleFn = level === "error" ? console.error : console.warn;

    consoleFn(JSON.stringify(entry));
  }

  debug(message: string, context?: Record<string, unknown>): void {
    this.log("debug", message, context);
  }

  info(message: string, context?: Record<string, unknown>): void {
    this.log("info", message, context);
  }

  warn(message: string, context?: Record<string, unknown>): void {
    this.log("warn", message, context);
  }

  error(message: string, error?: unknown, context?: Record<string, unknown>): void {
    const merged: Record<string, unknown> = { ...context };
    if (error) {
      merged["error"] = serializeError(error);
    }
    this.log("error", message, merged);
  }

  child(name: string): Logger {
    return new Logger(`${this.name}:${name}`);
  }
}

interface LoggerFactory {
  (name: string): Logger;
  default: Logger;
  isDev: boolean;
}

function createLogger(name: string): Logger {
  return new Logger(name);
}

const defaultLogger = new Logger("astra");

const loggerFactory = createLogger as LoggerFactory;
loggerFactory.default = defaultLogger;
loggerFactory.isDev = ENV["VITE_LOG_LEVEL"] === "debug" || ENV["VITE_LOG_LEVEL"] === "info";

export { Logger, createLogger, defaultLogger };
export type { LogLevel };

