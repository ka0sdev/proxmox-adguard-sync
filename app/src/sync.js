import { logger } from "./logger.js";
import {
  ensureStateFile,
  normalizeHostname,
  readJsonFile,
  writeJsonFile,
} from "./utils.js";

function buildDesiredMap(config, desiredGuests) {
  const desiredMap = new Map();

  for (const guest of desiredGuests) {
    const dnsName = guest.dnsName || guest.name;
    const domain = normalizeHostname(dnsName, config.dnsSuffix);

    desiredMap.set(domain, {
      domain,
      answer: guest.ip,
      vmid: guest.vmid,
      node: guest.node,
      sourceType: guest.sourceType,
      name: guest.name,
      dnsName,
      clientName: resolvePersistentClientName(
        config.persistentClients?.nameMode,
        guest,
        dnsName,
        domain
      ),
    });
  }

  return desiredMap;
}

function resolvePersistentClientName(nameMode, guest, dnsName, domain) {
  switch (nameMode) {
    case "fqdn":
      return domain;

    case "hostname":
      return dnsName;

    case "guest":
    default:
      return guest.name;
  }
}

function normalizeStringArray(values) {
  if (!Array.isArray(values)) return [];

  return [...new Set(values.map((value) => String(value).trim()).filter(Boolean))].sort(
    (a, b) => a.localeCompare(b)
  );
}

function sameStringArray(a, b) {
  const left = normalizeStringArray(a);
  const right = normalizeStringArray(b);

  if (left.length !== right.length) return false;

  return left.every((value, index) => value === right[index]);
}

function mergeTags(existingTags = [], managedTag = null) {
  const merged = normalizeStringArray(existingTags);

  if (!managedTag) return merged;

  if (!merged.includes(managedTag)) {
    merged.push(managedTag);
  }

  return normalizeStringArray(merged);
}

function buildManagedClientPayload(currentClient, desired, managedTag) {
  const payload = {
    name: desired.clientName,
    ids: [desired.answer],
  };

  const tags = mergeTags(currentClient?.tags, managedTag);

  if (tags.length) {
    payload.tags = tags;
  }

  return payload;
}

async function syncRewrites({ desiredMap, managedEntries, adguard }) {
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

  return {
    desired: desiredMap.size,
    adds,
    updates,
    deletes,
  };
}

async function syncPersistentClients({
  config,
  desiredMap,
  managedClients,
  adguard,
}) {
  if (!config.persistentClients?.enabled) {
    return {
      enabled: false,
      desired: 0,
      adds: 0,
      updates: 0,
      deletes: 0,
      skippedConflicts: 0,
    };
  }

  const currentClients = await adguard.listPersistentClients();
  const currentClientsByName = new Map();

  for (const client of currentClients) {
    const name = String(client?.name || "").trim();
    if (!name) continue;
    currentClientsByName.set(name, client);
  }

  let adds = 0;
  let updates = 0;
  let deletes = 0;
  let skippedConflicts = 0;

  for (const [domain, desired] of desiredMap.entries()) {
    const previousManaged = managedClients[domain];
    const currentByPreviousName = previousManaged?.name
      ? currentClientsByName.get(previousManaged.name)
      : null;
    const currentByDesiredName = currentClientsByName.get(desired.clientName);

    if (!previousManaged && currentByDesiredName) {
      skippedConflicts += 1;

      logger.warn("Skipping persistent client because a client with the same name already exists", {
        domain,
        clientName: desired.clientName,
        desiredId: desired.answer,
      });

      continue;
    }

    const currentClient = currentByPreviousName || currentByDesiredName || null;
    const desiredPayload = buildManagedClientPayload(
      currentClient,
      desired,
      config.persistentClients.managedTag
    );

    if (!currentClient) {
      await adguard.addPersistentClient(desiredPayload);

      managedClients[domain] = {
        name: desiredPayload.name,
        ids: desiredPayload.ids,
        vmid: desired.vmid,
        node: desired.node,
        sourceType: desired.sourceType,
        guestName: desired.name,
        dnsName: desired.dnsName,
        updatedAt: new Date().toISOString(),
      };

      adds += 1;

      logger.info("Added persistent client", {
        domain,
        clientName: desiredPayload.name,
        ids: desiredPayload.ids,
        sourceType: desired.sourceType,
        vmid: desired.vmid,
        node: desired.node,
      });

      continue;
    }

    const currentIds = normalizeStringArray(currentClient.ids);
    const desiredIds = normalizeStringArray(desiredPayload.ids);
    const currentTags = normalizeStringArray(currentClient.tags);
    const desiredTags = normalizeStringArray(desiredPayload.tags);

    const requiresUpdate =
      currentClient.name !== desiredPayload.name ||
      !sameStringArray(currentIds, desiredIds) ||
      !sameStringArray(currentTags, desiredTags);

    if (requiresUpdate) {
      await adguard.updatePersistentClient(currentClient.name, desiredPayload);

      managedClients[domain] = {
        name: desiredPayload.name,
        ids: desiredPayload.ids,
        vmid: desired.vmid,
        node: desired.node,
        sourceType: desired.sourceType,
        guestName: desired.name,
        dnsName: desired.dnsName,
        updatedAt: new Date().toISOString(),
      };

      updates += 1;

      logger.info("Updated persistent client", {
        domain,
        oldName: currentClient.name,
        newName: desiredPayload.name,
        oldIds: currentIds,
        newIds: desiredIds,
        sourceType: desired.sourceType,
        vmid: desired.vmid,
        node: desired.node,
      });

      continue;
    }

    managedClients[domain] = {
      name: desiredPayload.name,
      ids: desiredPayload.ids,
      vmid: desired.vmid,
      node: desired.node,
      sourceType: desired.sourceType,
      guestName: desired.name,
      dnsName: desired.dnsName,
      updatedAt: new Date().toISOString(),
    };
  }

  for (const [domain, previous] of Object.entries(managedClients)) {
    if (desiredMap.has(domain)) continue;

    const currentClient = previous?.name
      ? currentClientsByName.get(previous.name)
      : null;

    if (!currentClient) {
      delete managedClients[domain];
      continue;
    }

    await adguard.deletePersistentClient(previous.name);
    delete managedClients[domain];

    deletes += 1;

    logger.info("Deleted stale managed persistent client", {
      domain,
      clientName: previous.name,
      ids: currentClient.ids || previous.ids || [],
      previous,
    });
  }

  return {
    enabled: true,
    desired: desiredMap.size,
    adds,
    updates,
    deletes,
    skippedConflicts,
  };
}

export async function runSync({ config, proxmox, adguard }) {
  await ensureStateFile(config.stateFile);

  const state = await readJsonFile(config.stateFile, {
    managedEntries: {},
    managedClients: {},
  });

  const managedEntries = state.managedEntries || {};
  const managedClients = state.managedClients || {};

  const desiredGuests = await proxmox.buildDesiredEntries();
  const desiredMap = buildDesiredMap(config, desiredGuests);

  const rewriteStats = await syncRewrites({
    desiredMap,
    managedEntries,
    adguard,
  });

  const persistentClientStats = await syncPersistentClients({
    config,
    desiredMap,
    managedClients,
    adguard,
  });

  await writeJsonFile(config.stateFile, {
    managedEntries,
    managedClients,
  });

  logger.info("Sync completed", {
    rewrites: rewriteStats,
    persistentClients: persistentClientStats,
  });

  return {
    rewrites: rewriteStats,
    persistentClients: persistentClientStats,
  };
}