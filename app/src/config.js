import { parseBoolean, requireEnv } from "./utils.js";

function parseCsv(value, fallback = []) {
  if (!value || !String(value).trim()) return fallback;
  return String(value)
    .split(",")
    .map((v) => v.trim())
    .filter(Boolean);
}

export function getConfig() {
  return {
    syncIntervalSeconds: Number(process.env.SYNC_INTERVAL_SECONDS || 60),
    dnsSuffix: requireEnv("DNS_SUFFIX"),
    stateFile: requireEnv("STATE_FILE"),
    logLevel: process.env.LOG_LEVEL || "info",

    filters: {
      includeTypes: parseCsv(process.env.FILTER_INCLUDE_TYPES, ["qemu", "lxc"]),
      requireRunning: parseBoolean(process.env.FILTER_REQUIRE_RUNNING, false),
      includeTags: parseCsv(process.env.FILTER_INCLUDE_TAGS),
      excludeTags: parseCsv(process.env.FILTER_EXCLUDE_TAGS),
      includeNames: parseCsv(process.env.FILTER_INCLUDE_NAMES),
      excludeNames: parseCsv(process.env.FILTER_EXCLUDE_NAMES),
    },

    discovery: {
      vmOrder: parseCsv(process.env.DISCOVERY_VM_ORDER, ["guest-agent", "description", "cloudinit"]),
      lxcOrder: parseCsv(process.env.DISCOVERY_LXC_ORDER, ["config", "description"]),
      descriptionIpKeys: parseCsv(process.env.DESCRIPTION_IP_KEYS, ["dns_ip", "ip"]),
      descriptionNameKeys: parseCsv(process.env.DESCRIPTION_NAME_KEYS, ["dns_name", "name"]),
    },
  };
}