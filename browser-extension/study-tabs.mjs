import { domainRegex } from "./rules.mjs";

export function restrictedStudyTab(tab, websites) {
  const url = tab.url || tab.pendingUrl;
  if (typeof url !== "string" || !/^https?:\/\//i.test(url)) return false;
  return !websites.some((site) =>
    new RegExp(domainRegex(site.domain, site.subdomains), "i").test(url),
  );
}

// Existing documents can keep playing cached media after network rules change.
// Replace only restricted web tabs, never retain or report their addresses.
export async function enforceStudyTabs(tabs, state, blockedURL, current) {
  if (!state.active || state.exercise !== true || !current()) return;
  if (state.mode !== "strict") throw new Error("Invalid study policy");
  let existing;
  try {
    existing = await tabs.query({});
  } catch {
    throw new Error("Could not inspect study tabs");
  }
  for (const candidate of existing) {
    if (!current()) return;
    if (!Number.isInteger(candidate.id) || candidate.id < 0) continue;
    let tab;
    try {
      // Respect navigation or closure that happened after the initial snapshot.
      tab = await tabs.get(candidate.id);
    } catch {
      continue;
    }
    if (!current()) return;
    if (!restrictedStudyTab(tab, state.websites)) continue;
    try {
      await tabs.update(candidate.id, { url: blockedURL });
    } catch {
      if (!current()) return;
      try {
        await tabs.get(candidate.id);
      } catch {
        // Closing a tab while the sweep is running is already sufficient.
        continue;
      }
      throw new Error("Could not replace a restricted study tab");
    }
  }
}
