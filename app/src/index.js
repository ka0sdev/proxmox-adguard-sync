import { logger } from "./logger.js";
import { sleep } from "./utils.js";
import { ProxmoxClient } from "./proxmox.js";
import { AdGuardClient } from "./adguard.js";
import { runSync } from "./sync.js";
import { getConfig } from "./config.js";

const isHealthcheck = process.argv.includes("--healthcheck");

async function main() {
  const config = getConfig();
  const proxmox = new ProxmoxClient(config);
  const adguard = new AdGuardClient();

  if (isHealthcheck) {
    await adguard.listRewrites();
    process.exit(0);
  }

  let shuttingDown = false;

  process.on("SIGTERM", () => {
    shuttingDown = true;
    logger.info("Received SIGTERM, shutting down");
  });

  process.on("SIGINT", () => {
    shuttingDown = true;
    logger.info("Received SIGINT, shutting down");
  });

  logger.info("Starting proxmox-adguard-sync", {
    intervalSeconds: config.syncIntervalSeconds,
    dnsSuffix: config.dnsSuffix,
    vmDiscoveryOrder: config.discovery.vmOrder,
    lxcDiscoveryOrder: config.discovery.lxcOrder,
  });

  while (!shuttingDown) {
    try {
      await runSync({ config, proxmox, adguard });
    } catch (error) {
      logger.error("Sync failed", {
        error: error.message,
        stack: error.stack,
      });
    }

    if (shuttingDown) break;
    await sleep(config.syncIntervalSeconds * 1000);
  }

  logger.info("Stopped proxmox-adguard-sync");
}

main().catch((error) => {
  logger.error("Fatal startup error", {
    error: error.message,
    stack: error.stack,
  });
  process.exit(1);
});