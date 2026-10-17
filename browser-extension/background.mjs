import { compileRules, replaceRules } from "./rules.mjs";
import { Lease } from "./lease.mjs";
import { enforceStudyTabs } from "./study-tabs.mjs";

let port = null,
  applied = "",
  active = false,
  blocks = 0,
  next = null,
  epoch = 0,
  chain = Promise.resolve();
const lease = new Lease(() => disconnect());
function queue(fn) {
  chain = chain.then(fn, fn);
  return chain;
}
async function clear() {
  active = false;
  applied = "";
  blocks = 0;
  await replaceRules(chrome.declarativeNetRequest, []);
  await chrome.storage.session.set({ active: false, applied: "" });
  await chrome.action.setBadgeText({ text: "" });
}
function disconnect() {
  epoch++;
  const old = port;
  port = null;
  if (next) clearTimeout(next);
  next = null;
  lease.clear();
  try {
    old?.disconnect();
  } catch {
    /* Already disconnected. */
  }
  void queue(clear).catch(() => chrome.action.setBadgeText({ text: "!" }));
}
function send(ok = true) {
  if (!port) return;
  try {
    port.postMessage({ ok, applied, blocks, studyCapable: true });
    blocks = 0;
  } catch {
    disconnect();
  }
}
function connect() {
  if (port) return;
  const generation = ++epoch;
  try {
    port = chrome.runtime.connectNative("com.focuslock.bridge");
  } catch {
    disconnect();
    return;
  }
  port.onDisconnect.addListener(() => {
    const reason = chrome.runtime.lastError?.message || "";
    void chrome.storage.session.set({ lastError: reason });
    if (generation === epoch) disconnect();
  });
  port.onMessage.addListener((state) => {
    void queue(async () => {
      if (generation !== epoch || !port) return;
      try {
        if (next) clearTimeout(next);
        next = null;
        const id = state.active ? state.sessionId : "";
        if (
          state.active &&
          (typeof id !== "string" || !/^[0-9a-f]{32}$/.test(id))
        )
          throw new Error("Invalid session");
        const changed = applied !== id || active !== state.active;
        if (changed) {
          await replaceRules(chrome.declarativeNetRequest, compileRules(state));
          if (generation !== epoch) return;
          await enforceStudyTabs(
            chrome.tabs,
            state,
            chrome.runtime.getURL("blocked.html"),
            () => generation === epoch && !!port,
          );
          if (generation !== epoch || !port) return;
          applied = id;
          active = state.active;
        }
        lease.renew(state.leaseMs);
        await chrome.storage.session.set({
          active,
          applied,
          language: state.language || "ru",
        });
        await chrome.action.setBadgeText({ text: active ? "ON" : "" });
        await chrome.action.setBadgeBackgroundColor({ color: "#3b82f6" });
        if (changed) send();
        else next = setTimeout(() => send(), 2000);
      } catch (error) {
        if (generation !== epoch || !port) return;
        void chrome.storage.session.set({ lastError: String(error) });
        disconnect();
      }
    });
  });
  send(false);
}
chrome.runtime.onMessage.addListener((message, sender, respond) => {
  if (
    message?.kind === "blocked" &&
    sender.url === chrome.runtime.getURL("blocked.html") &&
    sender.tab &&
    active
  ) {
    blocks++;
    respond({ ok: true });
  }
  if (
    message?.kind === "status" &&
    sender.url === chrome.runtime.getURL("popup.html")
  )
    respond({ connected: !!port, active });
  return false;
});
chrome.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name === "focuslock-watchdog") {
    lease.check();
    if (!port) connect();
  }
});
chrome.runtime.onStartup.addListener(() => {
  disconnect();
  void queue(async () => {
    await clear();
    connect();
  });
});
chrome.runtime.onInstalled.addListener(() => {
  disconnect();
  void queue(async () => {
    await clear();
    connect();
  });
});
// A worker restart always begins fail-open. A native connection keeps the
// worker alive during a session; onDisconnect is the primary crash recovery.
void queue(async () => {
  await clear();
  await chrome.alarms.create("focuslock-watchdog", { periodInMinutes: 0.5 });
  connect();
});
