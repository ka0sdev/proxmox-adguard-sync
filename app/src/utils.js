import fs from "node:fs/promises";
import path from "node:path";

export function requireEnv(name, fallback = undefined) {
  const value = process.env[name] ?? fallback;
  if (value === undefined || value === null || value === "") {
    throw new Error(`Missing required environment variable: ${name}`);
  }
  return value;
}

export function parseBoolean(value, defaultValue = false) {
  if (value === undefined || value === null || value === "") return defaultValue;
  return ["1", "true", "yes", "on"].includes(String(value).toLowerCase());
}

export function normalizeHostname(name, suffix) {
  const sanitized = String(name)
    .trim()
    .toLowerCase()
    .replace(/\s+/g, "-")
    .replace(/[^a-z0-9.-]/g, "-")
    .replace(/-+/g, "-")
    .replace(/^\-+|\-+$/g, "");

  if (!sanitized) {
    throw new Error(`Unable to normalize hostname from name: ${name}`);
  }

  return `${sanitized}.${suffix}`;
}

export function isValidIPv4(ip) {
  if (!ip || typeof ip !== "string") return false;
  const parts = ip.trim().split(".");
  if (parts.length !== 4) return false;

  return parts.every((part) => {
    if (!/^\d+$/.test(part)) return false;
    const n = Number(part);
    return n >= 0 && n <= 255;
  });
}

export function extractIPv4FromCidr(value) {
  if (!value || typeof value !== "string") return null;
  const ip = value.split("/")[0]?.trim();
  return isValidIPv4(ip) ? ip : null;
}

export function parseLxcIpFromNetConfig(configValue) {
  if (!configValue || typeof configValue !== "string") return null;

  const match = configValue.match(/(?:^|,)ip=([^,]+)/i);
  if (!match) return null;

  const raw = match[1].trim();
  if (raw.toLowerCase() === "dhcp" || raw.toLowerCase() === "auto") {
    return null;
  }

  return extractIPv4FromCidr(raw);
}

export function parseCloudInitIp(configValue) {
  if (!configValue || typeof configValue !== "string") return null;

  const match = configValue.match(/(?:^|,)ip=([^,]+)/i);
  if (!match) return null;

  const raw = match[1].trim();
  if (raw.toLowerCase() === "dhcp" || raw.toLowerCase() === "auto") {
    return null;
  }

  return extractIPv4FromCidr(raw);
}

export function parseKeyValueFromText(text, keys = []) {
  if (!text || typeof text !== "string") return null;

  for (const key of keys) {
    const escapedKey = key.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    const match = text.match(new RegExp(`^\\s*${escapedKey}\\s*=\\s*(.+?)\\s*$`, "im"));
    if (match?.[1]) {
      return match[1].trim();
    }
  }

  return null;
}

export function parseDnsIpFromDescription(description, keys = ["dns_ip", "ip"]) {
  const value = parseKeyValueFromText(description, keys);
  return isValidIPv4(value) ? value : null;
}

export function parseDnsNameFromDescription(description, keys = ["dns_name", "name"]) {
  const value = parseKeyValueFromText(description, keys);
  return value || null;
}

export async function ensureStateFile(stateFile) {
  const dir = path.dirname(stateFile);
  await fs.mkdir(dir, { recursive: true });

  try {
    await fs.access(stateFile);
  } catch {
    await fs.writeFile(
      stateFile,
      JSON.stringify({ managedEntries: {} }, null, 2),
      "utf8"
    );
  }
}

export async function readJsonFile(filePath, fallback) {
  try {
    const raw = await fs.readFile(filePath, "utf8");
    return JSON.parse(raw);
  } catch {
    return fallback;
  }
}

export async function writeJsonFile(filePath, value) {
  await fs.writeFile(filePath, JSON.stringify(value, null, 2), "utf8");
}

export function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

export function splitTags(tags) {
  if (!tags || typeof tags !== "string") return [];
  return tags
    .split(/[;,]/)
    .map((t) => t.trim())
    .filter(Boolean);
}

export function matchesAny(value, patterns = []) {
  if (!patterns.length) return true;
  const normalized = String(value || "").toLowerCase();
  return patterns.some((pattern) => normalized.includes(String(pattern).toLowerCase()));
}