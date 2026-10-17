import { compileRules, replaceRules } from "./rules.mjs";
import { Lease } from "./lease.mjs";
import { enforceStudyTabs } from "./study-tabs.mjs";

export function startBridge(chrome, clock = {
  now: () => Date.now(),
  setTimeout: (fn, ms) => setTimeout(fn, ms),
  clearTimeout: (id) => clearTimeout(id),
  setInterval: (fn, ms) => setInterval(fn, ms),
  clearInterval: (id) => clearInterval(id),
}) {
  let port = null,
    applied = "",
    active = false,
    blocks = 0,
    ready = false,
    rulesDirty = false,
    wanted = "",
    heartbeat = null,
    retry = null,
    retryDelay = 500,
    cancelSweep = null,
    epoch = 0,
    chain = Promise.resolve();
  const lease = new Lease(() => disconnect(), () => clock.now(), clock.setTimeout, clock.clearTimeout);
  function queue(fn) {
    chain = chain.then(fn, fn);
    return chain;
  }
  async function clear() {
    active = false;
    ready = false;
    applied = "";
    blocks = 0;
    await replaceRules(chrome.declarativeNetRequest, []);
    rulesDirty = false;
    await chrome.storage.session.set({ active: false, applied: "" });
    await chrome.action.setBadgeText({ text: "" });
  }
  function disconnect() {
    epoch++;
    const old = port;
    port = null;
    ready = false;
    cancelSweep?.();
    cancelSweep = null;
    if (heartbeat) clock.clearInterval(heartbeat);
    heartbeat = null;
    lease.clear();
    try {
      old?.disconnect();
    } catch {
      /* Already disconnected. */
    }
    void queue(clear).catch(() => chrome.action.setBadgeText({ text: "!" }));
    if (!retry) {
      retry = clock.setTimeout(() => {
        retry = null;
        connect();
      }, retryDelay);
      retryDelay = Math.min(5000, retryDelay * 2);
    }
  }
  function send(ok = ready) {
    if (!port) return;
    try {
      port.postMessage({ ok, applied, blocks, studyCapable: true, resourceCapable: true });
      blocks = 0;
    } catch {
      disconnect();
    }
  }
  function connect() {
    if (port) return;
    if (retry) clock.clearTimeout(retry);
    retry = null;
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
      if (generation !== epoch || !port) return;
      // Heartbeats are independent of DNR installation and the asynchronous tab
      // sweep. Only a real response from the app renews the fail-open lease.
      try {
        if (typeof state.active !== "boolean") throw new Error("Invalid policy");
        const id = state.active ? state.sessionId : "";
        if (state.active && (typeof id !== "string" || !/^[0-9a-f]{32}$/.test(id)))
          throw new Error("Invalid session");
        lease.renew(state.leaseMs);
        if (wanted !== id) cancelSweep?.();
        wanted = id;
        if (applied !== wanted || active !== state.active) ready = false;
        retryDelay = 500;
      } catch {
        disconnect();
        return;
      }
      void queue(async () => {
        if (generation !== epoch || !port) return;
        try {
          const id = state.active ? state.sessionId : "";
          if (
            state.active &&
            (typeof id !== "string" || !/^[0-9a-f]{32}$/.test(id))
          )
            throw new Error("Invalid session");
          if (id !== wanted) return;
          const wasReady = ready;
          const changed = rulesDirty || applied !== id || active !== state.active;
          if (changed) {
            rulesDirty = true;
            await replaceRules(chrome.declarativeNetRequest, compileRules(state));
            if (generation !== epoch) return;
            let cancel;
            const cancelled = new Promise((resolve) => { cancel = resolve; });
            cancelSweep = cancel;
            try {
              await Promise.race([
                enforceStudyTabs(
                  chrome.tabs,
                  state,
                  chrome.runtime.getURL("blocked.html"),
                  () => generation === epoch && !!port && wanted === id,
                ),
                cancelled,
              ]);
            } finally {
              if (cancelSweep === cancel) cancelSweep = null;
            }
            if (generation !== epoch || !port || wanted !== id) return;
            applied = id;
            active = state.active;
            rulesDirty = false;
          }
          ready = true;
          await chrome.storage.session.set({
            active,
            applied,
            language: state.language || "ru",
          });
          await chrome.action.setBadgeText({ text: active ? "ON" : "" });
          await chrome.action.setBadgeBackgroundColor({ color: "#3b82f6" });
          if (changed || !wasReady) send();
        } catch (error) {
          if (generation !== epoch || !port) return;
          void chrome.storage.session.set({ lastError: String(error) });
          disconnect();
        }
      });
    });
    send(false);
    heartbeat = clock.setInterval(() => send(), 2000);
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

  return { disconnect };
}
