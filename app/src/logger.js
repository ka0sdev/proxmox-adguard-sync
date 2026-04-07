const LEVELS = {
  debug: 10,
  info: 20,
  warn: 30,
  error: 40,
};

const levelName = (process.env.LOG_LEVEL || "info").toLowerCase();
const currentLevel = LEVELS[levelName] ?? LEVELS.info;

const SERVICE = process.env.SERVICE_NAME || "proxmox-adguard-sync";
const JSON_LOGS = ["1", "true"].includes(String(process.env.LOG_JSON).toLowerCase());

function shouldLog(level) {
  return LEVELS[level] >= currentLevel;
}

function basePayload(level, message, meta) {
  return {
    ts: new Date().toISOString(),
    level,
    service: SERVICE,
    msg: message,
    ...(meta && Object.keys(meta).length ? { meta } : {}),
  };
}

function format(level, message, meta) {
  const payload = basePayload(level, message, meta);

  if (JSON_LOGS) {
    return JSON.stringify(payload);
  }

  const metaString =
    payload.meta && Object.keys(payload.meta).length
      ? ` ${JSON.stringify(payload.meta)}`
      : "";

  return `[${payload.ts}] [${level.toUpperCase()}] [${SERVICE}] ${message}${metaString}`;
}

export const logger = {
  debug(message, meta = {}) {
    if (shouldLog("debug")) console.log(format("debug", message, meta));
  },
  info(message, meta = {}) {
    if (shouldLog("info")) console.log(format("info", message, meta));
  },
  warn(message, meta = {}) {
    if (shouldLog("warn")) console.warn(format("warn", message, meta));
  },
  error(message, meta = {}) {
    if (shouldLog("error")) console.error(format("error", message, meta));
  },
};