import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { mkdtemp, readFile, mkdir } from "node:fs/promises";
import path from "node:path";
import os from "node:os";
import { createRequire } from "node:module";
const require = createRequire(import.meta.url);
const { chromium } = require(
  process.env.FOCUS_LOCK_PLAYWRIGHT_PATH || "playwright",
);
const root = path.resolve(import.meta.dirname, "..");
const scratch = await mkdtemp(
  path.join(process.env.FOCUS_LOCK_TEST_DIR || os.tmpdir(), "focuslock-ui-"),
);
const urlFile = path.join(scratch, "url.txt");
const exe =
  process.env.FOCUS_LOCK_TEST_EXE || path.join(root, "build/bin/FocusLock.exe");
const child = spawn(exe, ["--preview"], {
  env: {
    ...process.env,
    FOCUS_LOCK_DATA_DIR: scratch,
    FOCUS_LOCK_PREVIEW_URL_FILE: urlFile,
  },
  windowsHide: true,
  stdio: "ignore",
});
const delay = (ms) => new Promise((r) => setTimeout(r, ms));
let browser;
try {
  let url;
  for (let i = 0; i < 150; i++) {
    try {
      url = await readFile(urlFile, "utf8");
      break;
    } catch {
      await delay(100);
    }
  }
  assert.ok(url, "preview started");
  const u = new URL(url);
  const token = new URLSearchParams(u.hash.slice(1)).get("token");
  const api = async (command, payload = {}) => {
    const r = await fetch(u.origin + "/api", {
      method: "POST",
      headers: {
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ command, payload }),
    });
    if (!r.ok) throw new Error(await r.text());
    return r.json();
  };
  assert.equal(
    (await fetch(u.origin + "/api", { method: "POST" })).status,
    401,
  );
  let state = await api("state");
  await api("saveSettings", { ...state.settings, language: "en" });
  browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({
    viewport: { width: 1320, height: 900 },
    deviceScaleFactor: 1,
  });
  const errors = [];
  page.on("pageerror", (e) => errors.push(e.message));
  await page.goto(url);
  await page
    .getByRole("heading", { name: "Make room for what matters." })
    .waitFor();
  for (let step = 0; step < 4; step++) {
    await page.getByRole("button", { name: "Continue", exact: true }).click();
    await delay(300);
  }
  await page
    .getByRole("button", { name: "Go to Dashboard", exact: true })
    .click();
  await page.getByRole("dialog").waitFor({ state: "detached" });
  console.log("PASS onboarding, authenticated preview and live backend");
  state = await api("state");
  const profile = {
    ...state.profiles[0],
    name: "Development",
    websites: [
      { domain: "go.dev", subdomains: true },
      { domain: "github.com", subdomains: false },
      { domain: "localhost", subdomains: false },
    ],
  };
  const discovered = await api("discover");
  const code =
    discovered.find((a) =>
      /visual studio code|^code$|obsidian/i.test(a.name),
    ) || discovered[0];
  if (code) profile.applications = [code];
  await api("saveProfile", profile);
  await api("saveSettings", {
    ...state.settings,
    language: "en",
    onboarded: true,
    exitDelaySeconds: 15,
  });
  await delay(1500);
  const screenDir = path.join(root, "docs/screenshots");
  await mkdir(screenDir, { recursive: true });
  const screenshot = async (name) => {
    if (process.env.FOCUS_LOCK_SCREENSHOTS === "1") {
      await delay(350);
      await page.screenshot({ path: path.join(screenDir, name + ".png") });
    }
  };
  await screenshot("dashboard");
  await page.getByRole("button", { name: "Applications", exact: true }).click();
  await page
    .getByRole("heading", { name: "Applications", exact: true })
    .waitFor();
  await page.getByRole("textbox", { name: "Search applications…" }).fill("");
  await page
    .getByRole("heading", { name: "Applications", exact: true })
    .click();
  await screenshot("applications");
  await page.getByRole("button", { name: "Websites", exact: true }).click();
  await page.getByRole("heading", { name: "Websites", exact: true }).waitFor();
  await screenshot("websites");
  await page.getByRole("button", { name: "Profiles", exact: true }).click();
  await page.getByRole("button", { name: "New profile", exact: true }).click();
  await page
    .getByRole("dialog")
    .getByRole("textbox")
    .fill("Integration test profile");
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await page.getByRole("dialog").waitFor({ state: "detached" });
  assert.ok(
    (await api("state")).profiles.some(
      (p) => p.name === "Integration test profile",
    ),
  );
  console.log("PASS website/profile changes through SQLite and UI");
  await page.getByRole("button", { name: "Dashboard", exact: true }).click();
  await page.getByRole("button", { name: "Start Focus", exact: true }).click();
  await page
    .getByRole("button", { name: "End session early", exact: true })
    .waitFor();
  await screenshot("active-session");
  await assert.rejects(api("saveProfile", profile), /locked/);
  await assert.rejects(
    api("saveSettings", {
      language: "en",
      exitDelaySeconds: 0,
      onboarded: true,
    }),
    /locked/,
  );
  await page
    .getByRole("button", { name: "End session early", exact: true })
    .click();
  await page
    .getByRole("heading", {
      name: "Are you sure you want to leave your focus session?",
    })
    .waitFor();
  await screenshot("early-exit");
  await page.getByRole("button", { name: "End anyway", exact: true }).click();
  await page
    .getByRole("button", { name: "I changed my mind — continue", exact: true })
    .click();
  await page.getByRole("dialog").waitFor({ state: "detached" });
  assert.equal((await api("state")).session.state, "ACTIVE");
  await page
    .getByRole("button", { name: "End session early", exact: true })
    .click();
  await page.getByRole("button", { name: "End anyway", exact: true }).click();
  await page
    .getByRole("heading", { name: "Session complete", exact: true })
    .waitFor({ timeout: 25000 });
  assert.equal((await api("state")).session.outcome, "interrupted");
  console.log("PASS early exit, cancellation, delay expiry and mutation lock");
  await page.getByRole("button", { name: "New session", exact: true }).click();
  await api("start", { id: profile.id, minutes: 1 });
  console.log("Waiting for a real 60-second natural completion…");
  for (let i = 0; i < 70; i++) {
    if ((await api("state")).session.state === "FINISHED") break;
    await delay(1000);
  }
  state = await api("state");
  assert.equal(state.session.outcome, "completed");
  assert.equal(state.statistics.completed, 1);
  assert.equal(state.statistics.interrupted, 1);
  await page
    .getByRole("button", { name: "View statistics", exact: true })
    .click();
  await page
    .getByRole("heading", { name: "Statistics", exact: true })
    .waitFor();
  await screenshot("statistics");
  assert.deepEqual(errors, []);
  console.log(
    "PASS natural completion, statistics, six screenshots and no JavaScript errors",
  );
} finally {
  await browser?.close();
  child.kill();
}
