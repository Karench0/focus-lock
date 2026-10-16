import { test } from "node:test";
import assert from "node:assert/strict";
import { enforceStudyTabs, restrictedStudyTab } from "../study-tabs.mjs";

const websites = [
  {
    domain: "kotlinlang.org",
    subdomains: false,
    resources: ["resources.jetbrains.com"],
  },
  { domain: "play.kotlinlang.org", subdomains: true },
];
const policy = { active: true, exercise: true, mode: "strict", websites };
const blockedURL = "chrome-extension://test-extension/blocked.html";

test("study tabs use exact navigation rules and never treat resources as websites", () => {
  for (const url of [
    "https://kotlinlang.org/docs/home.html",
    "HTTP://KOTLINLANG.ORG:80/docs/",
    "https://play.kotlinlang.org/",
    "https://sub.play.kotlinlang.org/",
  ])
    assert.equal(restrictedStudyTab({ url }, websites), false, url);
  for (const url of [
    "https://www.youtube.com/watch?v=fixture",
    "https://www.tiktok.com/",
    "https://resources.jetbrains.com/",
    "https://kotlinlang.org.evil.test/",
    "https://www.kotlinlang.org/",
    "https://kotlinlang.org@evil.test/",
    "http://127.0.0.1:1234/",
  ])
    assert.equal(restrictedStudyTab({ url }, websites), true, url);
});

test("browser, extension, local and other non-web tabs are preserved", () => {
  for (const url of [
    "chrome://newtab/",
    "edge://extensions/",
    "about:blank",
    "file:///C:/notes.txt",
    blockedURL,
    "data:text/html,example",
    "blob:https://example.test/id",
    "devtools://devtools/",
  ])
    assert.equal(restrictedStudyTab({ url }, websites), false, url);
  assert.equal(restrictedStudyTab({}, websites), false);
  assert.equal(
    restrictedStudyTab({ pendingUrl: "https://youtube.com/" }, websites),
    true,
  );
});

test("ordinary or inactive sessions never enumerate or change tabs", async () => {
  const noTabs = {
    query: () => {
      throw new Error("must not inspect ordinary tabs");
    },
  };
  for (const state of [
    { ...policy, active: false },
    { ...policy, exercise: false },
    { ...policy, exercise: undefined },
  ])
    await enforceStudyTabs(noTabs, state, blockedURL, () => true);
  await enforceStudyTabs(noTabs, policy, blockedURL, () => false);
});

test("the sweep rechecks current tabs and replaces only disallowed web documents", async () => {
  const updates = [];
  const latest = new Map([
    [1, { id: 1, url: "https://youtube.com/" }],
    [2, { id: 2, url: "https://kotlinlang.org/docs/home.html" }],
    [3, { id: 3, url: "chrome://extensions/" }],
    // This tab navigated to the documentation after the initial snapshot.
    [4, { id: 4, url: "https://kotlinlang.org/docs/home.html" }],
  ]);
  const tabs = {
    query: async () => [
      ...[...latest.values()].map((tab) =>
        tab.id === 4 ? { ...tab, url: "https://youtube.com/" } : tab,
      ),
      { id: 5, url: "https://youtube.com/" },
      { url: "https://youtube.com/" },
      { id: -1, url: "https://youtube.com/" },
    ],
    get: async (id) => {
      if (!latest.has(id)) throw new Error("closed tab");
      return latest.get(id);
    },
    update: async (id, change) => updates.push({ id, change }),
  };
  await enforceStudyTabs(tabs, policy, blockedURL, () => true);
  assert.deepEqual(updates, [{ id: 1, change: { url: blockedURL } }]);
});

test("a stale connection stops navigation after an asynchronous snapshot or tab lookup", async () => {
  for (const staleAt of ["query", "get"]) {
    let current = true;
    const tabs = {
      query: async () => {
        if (staleAt === "query") current = false;
        return [{ id: 1, url: "https://youtube.com/" }];
      },
      get: async () => {
        if (staleAt === "get") current = false;
        return { id: 1, url: "https://youtube.com/" };
      },
      update: async () => assert.fail("stale policy navigated a tab"),
    };
    await enforceStudyTabs(tabs, policy, blockedURL, () => current);
  }
});

test("disconnecting during one replacement prevents subsequent replacements", async () => {
  let current = true;
  const updates = [];
  const tabs = {
    query: async () => [{ id: 1 }, { id: 2 }],
    get: async (id) => ({ id, url: "https://youtube.com/" }),
    update: async (id) => {
      updates.push(id);
      current = false;
    },
  };
  await enforceStudyTabs(tabs, policy, blockedURL, () => current);
  assert.deepEqual(updates, [1]);
});

test("failed replacements cancel activation without including the original site URL", async () => {
  const sensitiveURL = "https://example.test/private-fixture";
  await assert.rejects(
    enforceStudyTabs(
      {
        query: async () => [{ id: 1 }],
        get: async () => ({ id: 1, url: sensitiveURL }),
        update: async () => {
          throw new Error(sensitiveURL);
        },
      },
      policy,
      blockedURL,
      () => true,
    ),
    (error) =>
      error.message === "Could not replace a restricted study tab" &&
      !error.message.includes(sensitiveURL),
  );
  await assert.rejects(
    enforceStudyTabs(
      {
        query: async () => {
          throw new Error(sensitiveURL);
        },
      },
      policy,
      blockedURL,
      () => true,
    ),
    { message: "Could not inspect study tabs" },
  );
});

test("closing a tab during its replacement does not interrupt the sweep", async () => {
  let firstLookup = true;
  const updates = [];
  const tabs = {
    query: async () => [{ id: 1 }, { id: 2 }],
    get: async (id) => {
      if (id === 1 && !firstLookup) throw new Error("closed tab");
      firstLookup = false;
      return { id, url: "https://youtube.com/" };
    },
    update: async (id) => {
      if (id === 1) throw new Error("closed tab");
      updates.push(id);
    },
  };
  await enforceStudyTabs(tabs, policy, blockedURL, () => true);
  assert.deepEqual(updates, [2]);
});
