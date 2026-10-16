import assert from "node:assert/strict";
import { spawn, spawnSync } from "node:child_process";
import { randomBytes } from "node:crypto";
import { mkdtemp, readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import os from "node:os";
import { createRequire } from "node:module";
import http from "node:http";
const require = createRequire(import.meta.url);
const { chromium } = require(
  process.env.FOCUS_LOCK_PLAYWRIGHT_PATH || "playwright",
);
const root = path.resolve(import.meta.dirname, "..");
const scratch = await mkdtemp(
  path.join(
    process.env.FOCUS_LOCK_TEST_DIR || os.tmpdir(),
    "focuslock-extension-",
  ),
);
const host = path.join(root, "build/bin/FocusLock-host.exe");
const urlFile = path.join(scratch, "url");
const token = randomBytes(24).toString("hex");
const env = {
  ...process.env,
  FOCUS_LOCK_PIPE_SUFFIX: `-e2e-${process.pid}`,
  FOCUS_LOCK_TEST_TOKEN: token,
  FOCUS_LOCK_TEST_HOST: host,
  FOCUS_LOCK_TEST_URL_FILE: urlFile,
};
const manifest = JSON.parse(
  await readFile(path.join(root, "browser-extension/manifest.json"), "utf8"),
);
const extensionID = "ilmpmofmjbomibcnecekibjejgbpgcdb";
const nativeManifest = path.join(scratch, "native-host.json");
await writeFile(
  nativeManifest,
  JSON.stringify({
    name: "com.focuslock.bridge",
    description: "Focus Lock integration test",
    type: "stdio",
    path: host,
    allowed_origins: [`chrome-extension://${extensionID}/`],
  }),
);
const delay = (ms) => new Promise((r) => setTimeout(r, ms));
const saved = [];
let context;
let child;
const destination = http.createServer((req, res) => {
  res.setHeader("Cache-Control", "no-store");
  res.setHeader("Content-Type", "text/html");
  res.setHeader("Access-Control-Allow-Origin", "*");
  if (req.url === "/favicon.ico") {
    res.writeHead(204);
    res.end();
    return;
  }
  if (req.url === "/style.css") {
    res.setHeader("Content-Type", "text/css");
    res.end("body { background-color: rgb(17, 34, 51); }");
    return;
  }
  if (req.url === "/app.js") {
    res.setHeader("Content-Type", "application/javascript");
    res.end("window.resourceScriptLoaded = true;");
    return;
  }
  if (req.url === "/image.svg") {
    res.setHeader("Content-Type", "image/svg+xml");
    res.end(
      '<svg xmlns="http://www.w3.org/2000/svg" width="2" height="2"><rect width="2" height="2" fill="blue"/></svg>',
    );
    return;
  }
  if (req.url === "/resource-page") {
    const cdn = `http://127.0.0.1:${destination.address().port}`;
    res.end(
      `<!doctype html><title>Resource dependency fixture</title><link rel="stylesheet" href="${cdn}/style.css"><script src="${cdn}/app.js"></script><img id="cdn-image" src="${cdn}/image.svg"><h1>Resource test</h1>`,
    );
    return;
  }
  res.end(
    "<!doctype html><title>Focus Lock test destination</title><h1>Allowed test destination</h1>",
  );
});
await new Promise((resolve) => destination.listen(0, "127.0.0.1", resolve));
const reg = (args) =>
  spawnSync("reg.exe", args, { windowsHide: true, stdio: "pipe" });
async function until(fn, timeout = 15000) {
  const deadline = Date.now() + timeout;
  while (Date.now() < deadline) {
    const result = await fn();
    if (result) return result;
    await delay(100);
  }
  throw new Error("Timed out waiting for browser state");
}
try {
  assert.ok(manifest.key, "stable public extension identity");
  for (const vendor of ["Google\\Chrome", "Chromium", "Microsoft\\Edge"]) {
    const key = `HKCU\\Software\\${vendor}\\NativeMessagingHosts\\com.focuslock.bridge`;
    const backup = path.join(scratch, `${saved.length}.reg`);
    const existed = reg(["query", key]).status === 0;
    if (existed) assert.equal(reg(["export", key, backup, "/y"]).status, 0);
    saved.push({ key, backup, existed });
    assert.equal(
      reg(["add", key, "/ve", "/t", "REG_SZ", "/d", nativeManifest, "/f"])
        .status,
      0,
    );
  }
  child = spawn(
    process.env.FOCUS_LOCK_BRIDGE_FIXTURE ||
      path.join(root, "build/bin/bridge-fixture.exe"),
    [],
    { env, windowsHide: true, stdio: ["ignore", "pipe", "pipe"] },
  );
  let fixtureLog = "";
  child.stderr.on("data", (b) => {
    fixtureLog += b;
  });
  let url;
  await until(async () => {
    try {
      url = await readFile(urlFile, "utf8");
      return true;
    } catch {
      if (child.exitCode !== null) throw new Error(fixtureLog);
      return false;
    }
  });
  const api = async (rules) => {
    const response = await fetch(url + "/rules", {
      method: rules ? "POST" : "GET",
      headers: {
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json",
      },
      body: rules ? JSON.stringify(rules) : undefined,
    });
    assert.equal(response.status, 200);
    return response.json();
  };
  const args = [
    `--disable-extensions-except=${path.join(root, "browser-extension")}`,
    `--load-extension=${path.join(root, "browser-extension")}`,
  ];
  const launch = () =>
    chromium.launchPersistentContext(path.join(scratch, "profile"), {
      channel: process.env.FOCUS_LOCK_TEST_BROWSER || "chromium",
      headless: true,
      env,
      args,
    });
  context = await launch();
  context.on("console", (message) => {
    if (message.type() === "error")
      console.error("Browser error:", message.text());
  });
  context.on("weberror", (error) =>
    console.error("Worker error:", error.error().message),
  );
  let [worker] = context.serviceWorkers();
  if (!worker) worker = await context.waitForEvent("serviceworker");
  assert.equal(new URL(worker.url()).host, extensionID);
  try {
    await until(async () => (await api()).some((s) => s.connected));
  } catch (e) {
    console.error(
      "Native diagnostics:",
      await worker.evaluate(async () => ({
        storage: await chrome.storage.session.get(null),
        alarms: await chrome.alarms.getAll(),
      })),
      fixtureLog,
    );
    console.error(
      "Direct probe:",
      await worker.evaluate(
        () =>
          new Promise((resolve) => {
            const p = chrome.runtime.connectNative("com.focuslock.bridge");
            p.onDisconnect.addListener(() =>
              resolve(chrome.runtime.lastError?.message || "disconnected"),
            );
            p.onMessage.addListener((m) => {
              resolve(m);
              p.disconnect();
            });
            p.postMessage({ ok: false, applied: "", blocks: 0 });
            setTimeout(() => {
              p.disconnect();
              resolve("timeout");
            }, 5000);
          }),
      ),
    );
    throw e;
  }
  console.log(
    "PASS authenticated Native Messaging and real current-user named pipe",
  );
  const page = await context.newPage();
  const port = destination.address().port;
  const allowed = `http://localhost:${port}/page`;
  const blocked = `http://127.0.0.1:${port}/page`;
  const rules = {
    sessionId: "a".repeat(32),
    active: true,
    mode: "strict",
    websites: [{ domain: "localhost", subdomains: false }],
    language: "en",
    leaseMs: 6000,
  };
  await api(rules);
  await until(async () =>
    (await api()).some((s) => s.applied === rules.sessionId && s.connected),
  );
  await page.goto(allowed);
  if ((await page.title()) !== "Focus Lock test destination") {
    console.error(
      "Rule diagnostics:",
      page.url(),
      JSON.stringify(
        await worker.evaluate(async () => ({
          rules: await chrome.declarativeNetRequest.getSessionRules(),
        })),
      ),
    );
    console.error(
      "Match diagnostics:",
      await worker.evaluate(async (url) => {
        try {
          return await chrome.declarativeNetRequest.testMatchOutcome({
            url,
            type: "main_frame",
          });
        } catch (e) {
          return String(e);
        }
      }, allowed),
    );
  }
  assert.equal(await page.title(), "Focus Lock test destination");
  await page.goto(blocked);
  assert.equal(page.url(), `chrome-extension://${extensionID}/blocked.html`);
  assert.match(await page.locator("h1").innerText(), /unavailable|недоступен/);
  console.log(
    "PASS real Chrome DNR: allowlisted page opens, unlisted page redirects",
  );
  await page.goto(allowed);
  let failed = "";
  page.on("requestfailed", (r) => {
    if (r.url() === blocked) failed = r.failure()?.errorText || "";
  });
  await page.evaluate(async (url) => {
    try {
      await fetch(url);
    } catch {
      /* Expected block. */
    }
  }, blocked);
  assert.match(failed, /BLOCKED_BY_CLIENT/);
  console.log(
    "PASS strict mode also blocks non-allowlisted network subresources",
  );
  // Reproduce a GitHub-like page whose assets are on a separate domain.
  await page.goto(`http://localhost:${port}/resource-page`);
  assert.equal(await page.evaluate(() => !!window.resourceScriptLoaded), false);
  rules.sessionId = "c".repeat(32);
  rules.websites = [
    { domain: "localhost", subdomains: false, resources: ["127.0.0.1"] },
  ];
  await api(rules);
  await until(async () =>
    (await api()).some((s) => s.applied === rules.sessionId),
  );
  await page.reload();
  assert.equal(
    await page.evaluate(() => getComputedStyle(document.body).backgroundColor),
    "rgb(17, 34, 51)",
  );
  assert.equal(await page.evaluate(() => window.resourceScriptLoaded), true);
  assert.equal(
    await page
      .locator("#cdn-image")
      .evaluate((img) => img.complete && img.naturalWidth === 2),
    true,
  );
  const match = async (type, initiator) =>
    worker.evaluate(
      async ({ url, type, initiator }) =>
        chrome.declarativeNetRequest.testMatchOutcome({ url, type, initiator }),
      { url: blocked, type, initiator },
    );
  assert.ok(
    (await match("stylesheet", `http://localhost:${port}`)).matchedRules.some(
      (r) => r.ruleId === 1001,
    ),
  );
  assert.ok(
    (await match("stylesheet", "http://other.test")).matchedRules.some(
      (r) => r.ruleId === 2,
    ),
  );
  assert.ok(
    (await match("sub_frame", `http://localhost:${port}`)).matchedRules.some(
      (r) => r.ruleId === 2,
    ),
  );
  await page.goto(blocked);
  assert.match(page.url(), /blocked.html$/);
  console.log(
    "PASS cross-domain CSS, JavaScript and images load only after explicit approval; unrelated origins, frames and direct CDN navigation remain blocked",
  );
  if (process.env.FOCUS_LOCK_LIVE_GITHUB === "1") {
    rules.sessionId = "d".repeat(32);
    rules.websites = [
      {
        domain: "github.com",
        subdomains: false,
        resources: [
          "github.githubassets.com",
          "avatars.githubusercontent.com",
          "user-images.githubusercontent.com",
          "private-user-images.githubusercontent.com",
        ],
      },
    ];
    await api(rules);
    await until(async () =>
      (await api()).some((s) => s.applied === rules.sessionId),
    );
    const styles = [];
    page.on("response", (r) => {
      if (
        r.request().resourceType() === "stylesheet" &&
        new URL(r.url()).hostname === "github.githubassets.com"
      )
        styles.push(r.status());
    });
    await page.goto("https://github.com", {
      waitUntil: "load",
      timeout: 60000,
    });
    assert.ok(
      styles.some((status) => status === 200),
      "GitHub CDN stylesheet must actually load",
    );
    assert.ok(await page.evaluate(() => document.styleSheets.length > 0));
    console.log(
      `PASS live GitHub: ${styles.filter((status) => status === 200).length} CDN stylesheets loaded`,
    );
  }
  rules.sessionId = "b".repeat(32);
  rules.mode = "blocklist";
  rules.websites = [{ domain: "127.0.0.1", subdomains: false }];
  await api(rules);
  await until(async () =>
    (await api()).some((s) => s.applied === rules.sessionId),
  );
  await page.goto(allowed);
  assert.equal(await page.title(), "Focus Lock test destination");
  await page.goto(blocked);
  assert.match(page.url(), /blocked.html$/);
  console.log("PASS blocklist mode in real Chromium");
  const crashed = Date.now();
  child.kill();
  await until(
    async () =>
      await worker.evaluate(async () => {
        return (
          (await chrome.declarativeNetRequest.getSessionRules()).length === 0
        );
      }),
    12000,
  );
  await page.goto(blocked);
  assert.equal(await page.title(), "Focus Lock test destination");
  console.log(
    `PASS bridge process killed: access restored in ${Date.now() - crashed} ms`,
  );
  await context.close();
  context = await launch();
  [worker] = context.serviceWorkers();
  if (!worker) worker = await context.waitForEvent("serviceworker");
  await until(
    async () =>
      await worker.evaluate(
        async () =>
          (await chrome.declarativeNetRequest.getSessionRules()).length === 0,
      ),
  );
  console.log(
    "PASS browser restart without desktop app leaves no restrictions",
  );
} finally {
  await context?.close();
  child?.kill();
  destination.close();
  for (const { key, backup, existed } of saved.reverse()) {
    const r = existed ? reg(["import", backup]) : reg(["delete", key, "/f"]);
    if (r.status !== 0)
      console.error(`Could not restore test native-host registration: ${key}`);
  }
}
