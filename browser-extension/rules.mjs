// A session has immutable rules. Only in-memory session rules are used.
const hostname =
  /^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)*[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/;
export function domainRegex(domain, subdomains) {
  if (
    typeof domain !== "string" ||
    domain.length > 253 ||
    !hostname.test(domain)
  )
    throw new Error("Invalid domain");
  const escaped = domain.replaceAll(".", "\\.");
  return `^(https?|wss?)://${subdomains ? "([a-z0-9-]+\\.)*" : ""}${escaped}(:[0-9]+)?/`;
}
export function compileRules(state) {
  if (!state.active) return [];
  if (
    !["strict", "blocklist"].includes(state.mode) ||
    !Array.isArray(state.websites) ||
    state.websites.length > 200
  )
    throw new Error("Invalid policy");
  const rules = [];
  if (state.mode === "strict") {
    rules.push({
      id: 1,
      priority: 1,
      action: {
        type: "redirect",
        redirect: { extensionPath: "/blocked.html" },
      },
      condition: { regexFilter: "^https?://", resourceTypes: ["main_frame"] },
    });
    rules.push({
      id: 2,
      priority: 1,
      action: { type: "block" },
      condition: {
        regexFilter: "^(https?|wss?)://",
        excludedResourceTypes: ["main_frame"],
      },
    });
  }
  for (const [i, site] of state.websites.entries()) {
    const condition = {
      regexFilter: domainRegex(site.domain, site.subdomains),
      isUrlFilterCaseSensitive: false,
    };
    if (state.mode === "strict") {
      // DNR's implicit resource types omit main_frame. Explicitly allow
      // navigation as well as all other network resource types.
      rules.push({
        id: 10 + i * 2,
        priority: 10,
        action: { type: "allow" },
        condition: { ...condition, resourceTypes: ["main_frame"] },
      });
      rules.push({
        id: 11 + i * 2,
        priority: 10,
        action: { type: "allow" },
        condition: { ...condition, excludedResourceTypes: ["main_frame"] },
      });
    } else {
      rules.push({
        id: 10 + i * 2,
        priority: 1,
        action: {
          type: "redirect",
          redirect: { extensionPath: "/blocked.html" },
        },
        condition: { ...condition, resourceTypes: ["main_frame"] },
      });
      rules.push({
        id: 11 + i * 2,
        priority: 1,
        action: { type: "block" },
        condition: { ...condition, excludedResourceTypes: ["main_frame"] },
      });
    }
  }
  return rules;
}
export async function replaceRules(api, rules) {
  const old = await api.getSessionRules();
  await api.updateSessionRules({
    removeRuleIds: old.map((r) => r.id),
    addRules: rules,
  });
}
