import { requireEnv } from "./utils.js";

export class AdGuardClient {
  constructor() {
    this.baseUrl = requireEnv("ADGUARD_BASE_URL").replace(/\/+$/, "");
    this.username = requireEnv("ADGUARD_USERNAME");
    this.password = requireEnv("ADGUARD_PASSWORD");
  }

  get authHeader() {
    const token = Buffer.from(
      `${this.username}:${this.password}`,
      "utf8"
    ).toString("base64");

    return `Basic ${token}`;
  }

  async request(path, options = {}) {
    const response = await fetch(`${this.baseUrl}${path}`, {
      ...options,
      headers: {
        Authorization: this.authHeader,
        Accept: "application/json",
        "Content-Type": "application/json",
        ...(options.headers || {}),
      },
    });

    if (!response.ok) {
      const body = await response.text().catch(() => "");
      throw new Error(
        `AdGuard request failed ${response.status} ${response.statusText}: ${body}`
      );
    }

    if (response.status === 204) return null;

    const text = await response.text();

    if (!text) return null;

    try {
      return JSON.parse(text);
    } catch {
      return text;
    }
  }

  async listRewrites() {
    const data = await this.request("/control/rewrite/list", {
      method: "GET",
    });

    return Array.isArray(data) ? data : [];
  }

  async addRewrite(domain, answer) {
    return this.request("/control/rewrite/add", {
      method: "POST",
      body: JSON.stringify({ domain, answer }),
    });
  }

  async deleteRewrite(domain, answer) {
    return this.request("/control/rewrite/delete", {
      method: "POST",
      body: JSON.stringify({ domain, answer }),
    });
  }

  async listPersistentClients() {
    const data = await this.request("/control/clients", {
      method: "GET",
    });

    return this.extractPersistentClients(data);
  }

  async addPersistentClient(client) {
    return this.request("/control/clients/add", {
      method: "POST",
      body: JSON.stringify(client),
    });
  }

  async updatePersistentClient(name, client) {
    return this.request("/control/clients/update", {
      method: "POST",
      body: JSON.stringify({
        name,
        data: client,
      }),
    });
  }

  async deletePersistentClient(name) {
    return this.request("/control/clients/delete", {
      method: "POST",
      body: JSON.stringify({ name }),
    });
  }

  extractPersistentClients(data) {
    if (Array.isArray(data)) return data;

    if (Array.isArray(data?.clients)) return data.clients;

    if (data?.clients && typeof data.clients === "object") {
      return Object.values(data.clients);
    }

    if (Array.isArray(data?.data)) return data.data;

    if (Array.isArray(data?.data?.clients)) return data.data.clients;

    return [];
  }
}