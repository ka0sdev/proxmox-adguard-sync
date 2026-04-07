import { logger } from "./logger.js";
import {
  ensureStateFile,
  normalizeHostname,
  readJsonFile,
  writeJsonFile,
} from "./utils.js";

export async function runSync({ config, proxmox, adguard }) {
  await ensureStateFile(config.stateFile);

  const state = await readJsonFile(config.stateFile, { managedEntries: {} });
  const managedEntries = state.managedEntries || {};

  const desiredGuests = await proxmox.buildDesiredEntries();
  const desiredMap = new Map();

  for (const guest of desiredGuests) {
    const domain = normalizeHostname(guest.dnsName || guest.name, config.dnsSuffix);

    desiredMap.set(domain, {
      domain,
      answer: guest.ip,
      vmid: guest.vmid,
      node: guest.node,
      sourceType: guest.sourceType,
      name: guest.name,
      dnsName: guest.dnsName || guest.name,
    });
  }

  const currentRewrites = await adguard.listRewrites();
  const currentMap = new Map(
    currentRewrites.map((entry) => [entry.domain, entry.answer])
  );

  let adds = 0;
  let updates = 0;
  let deletes = 0;

  for (const [domain, desired] of desiredMap.entries()) {
    const currentAnswer = currentMap.get(domain);

    if (!currentAnswer) {
      await adguard.addRewrite(domain, desired.answer);
      managedEntries[domain] = {
        answer: desired.answer,
        vmid: desired.vmid,
        node: desired.node,
        sourceType: desired.sourceType,
        name: desired.name,
        dnsName: desired.dnsName,
        updatedAt: new Date().toISOString(),
      };
      adds += 1;

      logger.info("Added rewrite", {
        domain,
        answer: desired.answer,
        sourceType: desired.sourceType,
        vmid: desired.vmid,
        node: desired.node,
      });
      continue;
    }

    if (currentAnswer !== desired.answer) {
      await adguard.deleteRewrite(domain, currentAnswer);
      await adguard.addRewrite(domain, desired.answer);

      managedEntries[domain] = {
        answer: desired.answer,
        vmid: desired.vmid,
        node: desired.node,
        sourceType: desired.sourceType,
        name: desired.name,
        dnsName: desired.dnsName,
        updatedAt: new Date().toISOString(),
      };
      updates += 1;

      logger.info("Updated rewrite", {
        domain,
        oldAnswer: currentAnswer,
        newAnswer: desired.answer,
        sourceType: desired.sourceType,
        vmid: desired.vmid,
        node: desired.node,
      });
      continue;
    }

    managedEntries[domain] = {
      answer: desired.answer,
      vmid: desired.vmid,
      node: desired.node,
      sourceType: desired.sourceType,
      name: desired.name,
      dnsName: desired.dnsName,
      updatedAt: new Date().toISOString(),
    };
  }

  for (const [domain, previous] of Object.entries(managedEntries)) {
    if (desiredMap.has(domain)) continue;

    const currentAnswer = currentMap.get(domain);
    if (!currentAnswer) {
      delete managedEntries[domain];
      continue;
    }

    await adguard.deleteRewrite(domain, currentAnswer);
    delete managedEntries[domain];
    deletes += 1;

    logger.info("Deleted stale managed rewrite", {
      domain,
      answer: currentAnswer,
      previous,
    });
  }

  await writeJsonFile(config.stateFile, { managedEntries });

  logger.info("Sync completed", {
    desired: desiredMap.size,
    adds,
    updates,
    deletes,
  });

  return {
    desired: desiredMap.size,
    adds,
    updates,
    deletes,
  };
}