import { test } from "node:test";
import assert from "node:assert/strict";
import { startBridge } from "../connection.mjs";

const flush = async () => { for (let i = 0; i < 40; i++) await Promise.resolve(); };
const deferred = () => {
  let resolve;
  const promise = new Promise((done) => { resolve = done; });
  return { promise, resolve };
};
const event = () => {
  const listeners = [];
  return { addListener: (fn) => listeners.push(fn), emit: (...args) => listeners.forEach((fn) => fn(...args)) };
};

async function fixture() {
  let now = 0, serial = 0, policy = { active: false, leaseMs: 6000 }, responding = true;
  let rules = [], installDelay, tabDelay;
  const timers = new Map(), ports = [], sent = [];
  const schedule = (fn, ms, period = 0) => {
    const id = ++serial;
    timers.set(id, { fn, at: now + ms, period });
    return id;
  };
  const clock = {
    now: () => now,
    setTimeout: (fn, ms) => schedule(fn, ms),
    clearTimeout: (id) => timers.delete(id),
    setInterval: (fn, ms) => schedule(fn, ms, ms),
    clearInterval: (id) => timers.delete(id),
  };
  async function advance(ms) {
    const end = now + ms;
    for (;;) {
      const next = [...timers].sort((a, b) => a[1].at - b[1].at).find(([, t]) => t.at <= end);
      if (!next) break;
      const [id, timer] = next;
      now = timer.at;
      if (timer.period) timer.at += timer.period;
      else timers.delete(id);
      timer.fn();
      await flush();
    }
    now = end;
    await flush();
  }
  const chrome = {
    runtime: {
      onMessage: event(), onStartup: event(), onInstalled: event(),
      getURL: (path) => `chrome-extension://fixture/${path}`,
      connectNative: () => {
        const port = {
          onMessage: event(), onDisconnect: event(), disconnected: false,
          postMessage: (ack) => {
            sent.push({ ...ack, at: now });
            if (responding) queueMicrotask(() => port.onMessage.emit(structuredClone(policy)));
          },
          disconnect: () => { port.disconnected = true; port.onDisconnect.emit(); },
        };
        ports.push(port);
        return port;
      },
    },
    declarativeNetRequest: {
      getSessionRules: async () => rules,
      updateSessionRules: async ({ addRules }) => {
        if (addRules.length && installDelay) await installDelay.promise;
        rules = addRules;
      },
    },
    tabs: {
      query: async () => [{ id: 1 }],
      get: async () => ({ id: 1, url: "https://youtube.com/" }),
      update: async () => { if (tabDelay) await tabDelay.promise; },
    },
    storage: { session: { set: async () => {} } },
    action: { setBadgeText: async () => {}, setBadgeBackgroundColor: async () => {} },
    alarms: { onAlarm: event(), create: async () => {} },
  };
  startBridge(chrome, clock);
  await flush();
  await advance(2000);
  return {
    ports, sent, advance,
    get rules() { return rules; },
    set policy(value) { policy = value; },
    set responding(value) { responding = value; },
    holdInstall: () => (installDelay = deferred()),
    holdTabs: () => (tabDelay = deferred()),
  };
}

const study = { active: true, exercise: true, sessionId: "a".repeat(32), mode: "strict", leaseMs: 6000, websites: [{ domain: "kotlinlang.org" }] };
for (const operation of ["holdInstall", "holdTabs"]) {
  test(`heartbeats survive a slow ${operation} without premature acknowledgement`, async () => {
    const f = await fixture();
    const hold = f[operation]();
    f.policy = study;
    await f.advance(2000);
    const before = f.sent.length;
    await f.advance(10000);
    assert.equal(f.ports[0].disconnected, false);
    assert.ok(f.sent.length >= before + 5, "native channel kept alive during async work");
    assert.equal(f.sent.some((ack) => ack.ok && ack.applied === study.sessionId), false);
    hold.resolve();
    await flush();
    assert.ok(f.sent.some((ack) => ack.ok && ack.applied === study.sessionId));
    assert.ok(f.rules.length > 0);
    f.responding = false;
    await f.advance(6000);
    assert.equal(f.ports[0].disconnected, true, "only real app responses renew the lease");
    assert.deepEqual(f.rules, [], "app silence still clears browser restrictions");
  });
}

test("stop during a slow tab sweep cannot acknowledge the cancelled exercise", async () => {
  const f = await fixture();
  const hold = f.holdTabs();
  f.policy = study;
  await f.advance(2000);
  f.policy = { active: false, leaseMs: 6000 };
  await f.advance(2000);
  assert.equal(f.sent.some((ack) => ack.ok && ack.applied === study.sessionId), false);
  assert.deepEqual(f.rules, []);
  await f.advance(2000);
  assert.ok(f.sent.at(-1).ok);
  hold.resolve();
  await flush();
  assert.deepEqual(f.rules, [], "late tab update cannot restore cancelled rules");
});

test("disconnect clears rules without waiting for a stuck tab API", async () => {
  const f = await fixture();
  const hold = f.holdTabs();
  f.policy = study;
  await f.advance(2000);
  assert.ok(f.rules.length > 0);
  f.responding = false;
  await f.advance(6000);
  assert.equal(f.ports[0].disconnected, true);
  assert.deepEqual(f.rules, []);
  hold.resolve();
  await flush();
  assert.deepEqual(f.rules, []);
});

test("a transient native disconnect reconnects before the 30-second alarm", async () => {
  const f = await fixture();
  f.ports[0].disconnect();
  await flush();
  await f.advance(499);
  assert.equal(f.ports.length, 1);
  await f.advance(1);
  assert.equal(f.ports.length, 2);
  await f.advance(2000);
  assert.equal(f.ports[1].disconnected, false);
  assert.ok(f.sent.at(-1).resourceCapable && f.sent.at(-1).ok);
});
