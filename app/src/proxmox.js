import { logger } from "./logger.js";
import {
  isValidIPv4,
  parseBoolean,
  parseCloudInitIp,
  parseDnsIpFromDescription,
  parseDnsNameFromDescription,
  parseLxcIpFromNetConfig,
  requireEnv,
  splitTags,
  matchesAny,
} from "./utils.js";

export class ProxmoxClient {
  constructor(config) {
    this.config = config;
    this.baseUrl = requireEnv("PROXMOX_BASE_URL").replace(/\/+$/, "");
    this.tokenId = requireEnv("PROXMOX_TOKEN_ID");
    this.tokenSecret = requireEnv("PROXMOX_TOKEN_SECRET");
    this.verifyTls = parseBoolean(process.env.PROXMOX_VERIFY_TLS, true);

    if (!this.verifyTls) {
      process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0";
      logger.warn("Proxmox TLS verification is disabled");
    }
  }

  get authHeader() {
    return `PVEAPIToken=${this.tokenId}=${this.tokenSecret}`;
  }

  async get(path) {
    const url = `${this.baseUrl}${path}`;

    const response = await fetch(url, {
      method: "GET",
      headers: {
        Authorization: this.authHeader,
        Accept: "application/json",
      },
    });

    if (!response.ok) {
      const body = await response.text().catch(() => "");
      throw new Error(`Proxmox request failed ${response.status} ${response.statusText}: ${body}`);
    }

    const json = await response.json();
    return json.data;
  }

  async listClusterResources() {
    return this.get("/cluster/resources?type=vm");
  }

  async getLxcConfig(node, vmid) {
    return this.get(`/nodes/${encodeURIComponent(node)}/lxc/${vmid}/config`);
  }

  async getVmConfig(node, vmid) {
    return this.get(`/nodes/${encodeURIComponent(node)}/qemu/${vmid}/config`);
  }

  async getVmAgentNetworkInterfaces(node, vmid) {
    return this.get(`/nodes/${encodeURIComponent(node)}/qemu/${vmid}/agent/network-get-interfaces`);
  }

  shouldIncludeResource(resource) {
    const { filters } = this.config;
    const type = String(resource.type || "").toLowerCase();
    const name = String(resource.name || "");
    const tags = splitTags(resource.tags);

    if (!filters.includeTypes.includes(type)) return false;
    if (resource.template === 1) return false;
    if (!resource.name || !resource.node || !resource.vmid) return false;
    if (filters.requireRunning && resource.status !== "running") return false;

    if (filters.includeNames.length && !matchesAny(name, filters.includeNames)) {
      return false;
    }

    if (filters.excludeNames.length && matchesAny(name, filters.excludeNames)) {
      return false;
    }

    if (filters.includeTags.length) {
      const hasIncludedTag = tags.some((tag) =>
        filters.includeTags.some((wanted) => tag.toLowerCase() === wanted.toLowerCase())
      );
      if (!hasIncludedTag) return false;
    }

    if (filters.excludeTags.length) {
      const hasExcludedTag = tags.some((tag) =>
        filters.excludeTags.some((blocked) => tag.toLowerCase() === blocked.toLowerCase())
      );
      if (hasExcludedTag) return false;
    }

    return true;
  }

  async buildDesiredEntries() {
    const resources = await this.listClusterResources();
    const desiredEntries = [];

    for (const resource of resources) {
      if (!this.shouldIncludeResource(resource)) continue;

      const { type, vmid, node, name, status } = resource;

      try {
        if (type === "lxc") {
          const entry = await this.buildLxcEntry({ node, vmid, name, status });
          if (entry) desiredEntries.push(entry);
          continue;
        }

        if (type === "qemu") {
          const entry = await this.buildVmEntry({ node, vmid, name, status });
          if (entry) desiredEntries.push(entry);
        }
      } catch (error) {
        logger.warn("Failed to build desired entry for guest", {
          type,
          vmid,
          node,
          name,
          error: error.message,
        });
      }
    }

    return desiredEntries;
  }

  async buildLxcEntry({ node, vmid, name, status }) {
    const config = await this.getLxcConfig(node, vmid);

    let ip = null;
    let customName = null;

    for (const strategy of this.config.discovery.lxcOrder) {
      if (strategy === "config") {
        ip = this.extractLxcConfigIp(config);
      }

      if (strategy === "description") {
        customName ||= parseDnsNameFromDescription(
          config.description,
          this.config.discovery.descriptionNameKeys
        );
        ip ||= parseDnsIpFromDescription(
          config.description,
          this.config.discovery.descriptionIpKeys
        );
      }

      if (ip) break;
    }

    if (!ip) {
      logger.warn("Skipping LXC without discoverable IPv4", {
        node,
        vmid,
        name,
        status,
      });
      return null;
    }

    return {
      sourceType: "lxc",
      vmid: String(vmid),
      node,
      name,
      dnsName: customName || name,
      ip,
    };
  }

  async buildVmEntry({ node, vmid, name, status }) {
    const config = await this.getVmConfig(node, vmid);

    let ip = null;
    let customName = null;

    for (const strategy of this.config.discovery.vmOrder) {
      if (strategy === "guest-agent") {
        try {
          const interfacesPayload = await this.getVmAgentNetworkInterfaces(node, vmid);
          ip = this.extractUsefulVmIpv4(interfacesPayload);
        } catch (error) {
          logger.debug("VM guest agent query failed", {
            node,
            vmid,
            name,
            error: error.message,
          });
        }
      }

      if (strategy === "description") {
        customName ||= parseDnsNameFromDescription(
          config.description,
          this.config.discovery.descriptionNameKeys
        );
        ip ||= parseDnsIpFromDescription(
          config.description,
          this.config.discovery.descriptionIpKeys
        );
      }

      if (strategy === "cloudinit") {
        ip ||= this.extractVmCloudInitIp(config);
      }

      if (ip) break;
    }

    if (!ip || !isValidIPv4(ip)) {
      logger.warn("Skipping VM without discoverable IPv4", {
        node,
        vmid,
        name,
        status,
      });
      return null;
    }

    return {
      sourceType: "qemu",
      vmid: String(vmid),
      node,
      name,
      dnsName: customName || name,
      ip,
    };
  }

  extractLxcConfigIp(config) {
    for (const [key, value] of Object.entries(config || {})) {
      if (/^net\d+$/i.test(key)) {
        const ip = parseLxcIpFromNetConfig(value);
        if (ip) return ip;
      }
    }
    return null;
  }

  extractVmCloudInitIp(config) {
    for (const [key, value] of Object.entries(config || {})) {
      if (/^ipconfig\d+$/i.test(key)) {
        const ip = parseCloudInitIp(value);
        if (ip) return ip;
      }
    }
    return null;
  }

  extractUsefulVmIpv4(payload) {
    const interfaces = this.normalizeAgentInterfaces(payload);

    if (!Array.isArray(interfaces) || interfaces.length === 0) {
      return null;
    }

    for (const iface of interfaces) {
      const ifaceName = String(iface?.name || iface?.["interface-name"] || "").toLowerCase();

      if (
        ifaceName === "lo" ||
        ifaceName.startsWith("docker") ||
        ifaceName.startsWith("br-") ||
        ifaceName.startsWith("veth") ||
        ifaceName.startsWith("virbr") ||
        ifaceName.startsWith("tailscale") ||
        ifaceName.startsWith("zt")
      ) {
        continue;
      }

      const addresses =
        iface?.["ip-addresses"] ||
        iface?.ipAddresses ||
        iface?.ip_addresses ||
        [];

      if (!Array.isArray(addresses)) continue;

      for (const addr of addresses) {
        const ipType = String(
          addr?.["ip-address-type"] ||
          addr?.ipAddressType ||
          addr?.ip_address_type ||
          ""
        ).toLowerCase();

        const raw =
          addr?.["ip-address"] ||
          addr?.ipAddress ||
          addr?.ip_address ||
          addr?.address ||
          null;

        if (!raw || typeof raw !== "string") continue;
        if (ipType && ipType !== "ipv4") continue;
        if (!isValidIPv4(raw)) continue;

        if (raw.startsWith("127.") || raw.startsWith("169.254.")) {
          continue;
        }

        return raw;
      }
    }

    return null;
  }

  normalizeAgentInterfaces(payload) {
    if (Array.isArray(payload)) return payload;
    if (!payload || typeof payload !== "object") return [];
    if (Array.isArray(payload.result)) return payload.result;
    if (Array.isArray(payload.interfaces)) return payload.interfaces;
    if (payload.data && Array.isArray(payload.data)) return payload.data;
    if (payload.data && Array.isArray(payload.data.result)) return payload.data.result;
    return [];
  }
}