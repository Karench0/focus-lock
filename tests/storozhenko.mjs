import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { mkdir, mkdtemp, readFile } from "node:fs/promises";
import { createRequire } from "node:module";
import os from "node:os";
import path from "node:path";

const require = createRequire(import.meta.url);
const { chromium } = require(
  process.env.FOCUS_LOCK_PLAYWRIGHT_PATH || "playwright",
);
const root = path.resolve(import.meta.dirname, "..");
// Test-only source oracle. Answer keys never travel through the UI API.
const oraclePath =
  process.env.FOCUS_LOCK_QUIZ_ORACLE ||
  path.join(root, "tests/oracles/kotlin-quiz.json");
const oracle = JSON.parse(await readFile(oraclePath, "utf8"));
assert.equal(Object.keys(oracle).length, 20, "20-question private test oracle");
const scratch = await mkdtemp(
  path.join(
    process.env.FOCUS_LOCK_TEST_DIR || os.tmpdir(),
    "focuslock-kotlin-",
  ),
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
const delay = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
const poll = async (check, message, timeout = 15000) => {
  const deadline = Date.now() + timeout;
  while (Date.now() < deadline) {
    if (await check()) return;
    await delay(100);
  }
  assert.fail(message);
};
let browser;
try {
  let url;
  await poll(async () => {
    try {
      url = await readFile(urlFile, "utf8");
      return !!url;
    } catch {
      return false;
    }
  }, "isolated preview started");
  const base = new URL(url);
  const token = new URLSearchParams(base.hash.slice(1)).get("token");
  const api = async (command, payload = {}) => {
    const response = await fetch(base.origin + "/api", {
      method: "POST",
      headers: {
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ command, payload }),
    });
    if (!response.ok) throw new Error(await response.text());
    return response.json();
  };
  let state = await api("state");
  assert.equal(
    state.preview,
    true,
    "all tests use preview: no process/browser changes",
  );
  await api("saveSettings", {
    ...state.settings,
    language: "en",
    onboarded: true,
    exitDelaySeconds: 15,
  });
  const profile = {
    ...state.profiles[0],
    name: "My normal preferences",
    mode: "blocklist",
    websites: [{ domain: "example.com", subdomains: false }],
  };
  await api("saveProfile", profile);
  state = await api("state");
  const normalSettings = structuredClone(state.settings);
  const normalProfiles = structuredClone(state.profiles);

  browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({
    viewport: { width: 1320, height: 1000 },
    deviceScaleFactor: 1,
  });
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await page.goto(url);
  await page.getByRole("button", { name: "Settings", exact: true }).click();
  await page.getByRole("heading", { name: "Settings", exact: true }).waitFor();
  const dot = page.getByRole("button", {
    name: "Additional settings: activate three times in a row",
    exact: true,
  });
  const mode = page.getByRole("switch", {
    name: "Storozhenko mode",
    exact: true,
  });
  assert.equal(await mode.count(), 0, "advanced settings start hidden");
  await dot.hover();
  assert.equal(
    await dot.evaluate((node) => getComputedStyle(node).cursor),
    "default",
  );
  await dot.click();
  assert.equal(await mode.count(), 0, "one click does not reveal the panel");
  await delay(850);
  await dot.click();
  await delay(850);
  await dot.click();
  assert.equal(
    await mode.count(),
    0,
    "slow accidental clicks do not accumulate",
  );
  await delay(850);
  for (let click = 0; click < 3; click++) {
    await dot.click();
    if (click < 2) await delay(150);
  }
  await mode.waitFor();
  assert.equal(await mode.getAttribute("aria-checked"), "false");
  const screenshots = path.join(root, "docs/screenshots");
  const screenshot = async (name, target = page) => {
    if (process.env.FOCUS_LOCK_SCREENSHOTS === "1") {
      await mkdir(screenshots, { recursive: true });
      await delay(350);
      await target.screenshot({ path: path.join(screenshots, name + ".png") });
    }
  };
  await screenshot(
    "storozhenko-settings",
    page.locator(".storozhenko-settings"),
  );
  console.log(
    "PASS hidden settings: normal cursor, slow clicks rejected, timed triple activation",
  );

  await mode.click();
  let dialog = page.getByRole("dialog");
  await dialog
    .getByRole("heading", { name: "Are you Storozhenko?", exact: true })
    .waitFor();
  let enable = dialog.getByRole("button", { name: "Enable mode", exact: true });
  assert.equal(
    await enable.isDisabled(),
    true,
    "explicit self-attestation required",
  );
  assert.equal((await api("state")).session.exercise, false);
  await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
  await dialog.waitFor({ state: "detached" });
  state = await api("state");
  assert.equal(state.session.exercise, false);
  assert.deepEqual(state.settings, normalSettings);
  assert.deepEqual(state.profiles, normalProfiles);
  assert.equal(await mode.getAttribute("aria-checked"), "false");

  await mode.click();
  dialog = page.getByRole("dialog");
  // Force one start failure, then ensure the same dialog receives a new
  // preparation token and asks for consent again instead of replaying it.
  let firstAttempt = true;
  let nextQuestionDelay = 0;
  const startTokens = [];
  await page.route(base.origin + "/api", async (route) => {
    const payload = route.request().postDataJSON();
    if (payload?.command === "kotlinQuestion" && nextQuestionDelay) {
      const pause = nextQuestionDelay;
      nextQuestionDelay = 0;
      await delay(pause);
    }
    if (payload?.command === "startStorozhenko") {
      startTokens.push(payload.payload.token);
      if (firstAttempt) {
        firstAttempt = false;
        await route.fulfill({ status: 400, contentType: "text/plain", body: "Simulated first-start failure" });
        return;
      }
    }
    await route.continue();
  });
  await dialog.getByRole("checkbox", { name: "Yes, I am Storozhenko and I want to enable this mode", exact: true }).check();
  await dialog.getByRole("button", { name: "Enable mode", exact: true }).click();
  await dialog.getByRole("alert").filter({ hasText: "Simulated first-start failure" }).waitFor();
  assert.equal(await dialog.getByRole("checkbox").isChecked(), false, "fresh review resets explicit consent");
  await dialog
    .getByRole("checkbox", {
      name: "Yes, I am Storozhenko and I want to enable this mode",
      exact: true,
    })
    .check();
  enable = dialog.getByRole("button", { name: "Enable mode", exact: true });
  assert.equal(await enable.isEnabled(), true);
  await enable.click();
  await dialog.waitFor({ state: "detached" });
  assert.equal(startTokens.length, 2);
  assert.notEqual(startTokens[0], startTokens[1], "retry must use a fresh start token");
  await poll(
    async () => (await api("state")).session.exercise === true,
    "exercise started after consent",
  );
  state = await api("state");
  assert.equal(state.session.state, "ACTIVE");
  assert.equal(state.session.profile.name, "Storozhenko · Kotlin");
  assert.equal(state.session.profile.mode, "strict");
  assert.ok(
    state.session.profile.websites.some(
      (site) => site.domain === "kotlinlang.org",
    ),
  );
  assert.ok(
    !state.session.profile.websites.some((site) =>
      /youtube|tiktok|steam|example/.test(site.domain),
    ),
  );
  assert.deepEqual(
    state.settings,
    normalSettings,
    "exercise does not overwrite settings",
  );
  assert.deepEqual(
    state.profiles,
    normalProfiles,
    "exercise does not overwrite normal profiles",
  );
  await assert.rejects(
    api("requestExit"),
    /Kotlin|вопрос|quiz|exercise|заняти/i,
  );
  await assert.rejects(
    api("saveSettings", { ...normalSettings, exitDelaySeconds: 0 }),
    /locked/,
  );
  await assert.rejects(api("saveProfile", profile), /locked/);
  console.log(
    "PASS consent, Kotlin-only session, preserved preferences, ordinary exit blocked server-side",
  );

  await page.getByRole("button", { name: "Mini timer", exact: true }).click();
  await page.setViewportSize({ width: 420, height: 180 });
  await page.getByRole("region", { name: "Mini timer", exact: true }).waitFor();
  await page
    .getByRole("timer", { name: "Time studying Kotlin", exact: true })
    .waitFor();
  assert.equal(
    await page.getByText("End this session early?", { exact: true }).count(),
    0,
  );
  await screenshot("mini-kotlin");
  const compactCommands = [];
  const recordCommand = (request) => {
    if (request.url() === base.origin + "/api")
      compactCommands.push(request.postDataJSON()?.command);
  };
  page.on("request", recordCommand);
  await page.getByRole("button", { name: "Stop", exact: true }).click();
  await page.setViewportSize({ width: 1320, height: 1000 });
  dialog = page.getByRole("dialog");
  await dialog
    .getByRole("heading", {
      name: "One question before you finish",
      exact: true,
    })
    .waitFor();
  page.off("request", recordCommand);
  assert.equal(
    compactCommands.includes("requestExit"),
    false,
    "compact exercise stop bypasses the ordinary early-exit command",
  );
  assert.ok(compactCommands.includes("kotlinQuestion"));
  assert.equal((await api("state")).session.state, "ACTIVE");
  await dialog
    .getByRole("button", { name: "Keep studying", exact: true })
    .click();
  await dialog.waitFor({ state: "detached" });
  console.log(
    "PASS compact exercise timer shows elapsed time and Stop opens the Kotlin question directly",
  );

  await page.getByRole("button", { name: "Dashboard", exact: true }).click();
  await page
    .getByRole("button", { name: "End session early", exact: true })
    .click();
  dialog = page.getByRole("dialog");
  await dialog
    .getByRole("heading", {
      name: "One question before you finish",
      exact: true,
    })
    .waitFor();
  const firstQuestion = await api("kotlinQuestion");
  const codeOf = (question) => question.prompt.match(/```kotlin\n([\s\S]*?)```/)?.[1].trimEnd();
  await poll(async () => {
    const layout = await dialog.locator(".kotlin-code pre").evaluateAll((nodes) => {
      const node = nodes[0];
      if (!node?.isConnected) return null;
      const style = getComputedStyle(node);
      return { whiteSpace: style.whiteSpace, tabSize: style.getPropertyValue("tab-size"), text: node.textContent };
    });
    return layout?.whiteSpace === "pre" && layout.tabSize === "4" && layout.text === codeOf(firstQuestion);
  }, "current question's code block rendered with preserved indentation");
  assert.equal(await dialog.locator(".kotlin-code pre code").textContent(), codeOf(firstQuestion));
  assert.ok(await dialog.locator(".kotlin-code .kt-keyword, .kotlin-code .kt-function").count() > 0, "Kotlin tokens are highlighted");
  await screenshot("kotlin-code-question");
  await dialog
    .getByRole("button", { name: "Keep studying", exact: true })
    .click();
  await dialog.waitFor({ state: "detached" });
  assert.equal((await api("state")).session.state, "ACTIVE");
  await page.getByRole("button", { name: "Settings", exact: true }).click();
  await mode.waitFor();
  assert.equal(await mode.getAttribute("aria-checked"), "true");
  nextQuestionDelay = 700;
  await mode.click();
  dialog = page.getByRole("dialog");
  await dialog.getByRole("status").filter({ hasText: "Preparing your question" }).waitFor();
  assert.equal(await dialog.getByRole("radio").count(), 0, "loading does not flash the previous question");
  await poll(async () => (await dialog.getByRole("radio").count()) === 8, "all eight answers loaded after reopening");
  await dialog
    .getByRole("heading", {
      name: "One question before you finish",
      exact: true,
    })
    .waitFor();
  assert.equal(
    (await api("kotlinQuestion")).token,
    firstQuestion.token,
    "cancel and reopen preserves the unanswered question",
  );
  assert.deepEqual(Object.keys(firstQuestion).sort(), [
    "options",
    "prompt",
    "sourceUrl",
    "token",
  ]);
  assert.equal(firstQuestion.options.length, 8);
  assert.equal(new Set(firstQuestion.options).size, 8);
  assert.equal(new URL(firstQuestion.sourceUrl).hostname, "kotlinlang.org");
  const correctIndex = firstQuestion.options.indexOf(
    oracle[firstQuestion.prompt],
  );
  assert.ok(
    correctIndex >= 0,
    "question is covered by verified private oracle",
  );
  const wrongIndex = (correctIndex + 1) % 8;
  const submit = dialog.getByRole("button", {
    name: "Submit answer",
    exact: true,
  });
  assert.equal(await submit.isDisabled(), true);
  assert.equal(await dialog.getByRole("radio").count(), 8);
  for (const height of [1000, 730]) {
    await page.setViewportSize({ width: 1320, height });
    const layout = await submit.evaluate((button) => {
      const modal = button.closest(".modal");
      const options = modal.querySelector(".kotlin-options-scroll");
      return {
        buttonBottom: button.getBoundingClientRect().bottom,
        modalBottom: modal.getBoundingClientRect().bottom,
        optionsHeight: options.clientHeight,
        optionsContentHeight: options.scrollHeight,
        viewportHeight: window.innerHeight,
      };
    });
    assert.ok(
      layout.buttonBottom <=
        Math.min(layout.modalBottom, layout.viewportHeight) - 12,
      `submit remains visible without outer scrolling at ${height}px height`,
    );
    assert.ok(
      layout.optionsHeight >= 70,
      "answer list keeps a usable scroll area",
    );
    if (height === 730) {
      assert.ok(
        layout.optionsContentHeight > layout.optionsHeight,
        "long answer list scrolls independently",
      );
    }
  }
  const lastRadio = dialog.getByRole("radio").last();
  await lastRadio.focus();
  await page.keyboard.press("Space");
  assert.equal(
    await lastRadio.isChecked(),
    true,
    "last answer is keyboard reachable inside the scrolling list",
  );
  await page.keyboard.press("Tab");
  await page.keyboard.press("Tab");
  assert.equal(
    await submit.evaluate((button) => document.activeElement === button),
    true,
    "keyboard reaches the visible submit button",
  );
  await screenshot("kotlin-question-compact");
  await page.setViewportSize({ width: 1320, height: 1000 });
  await dialog.locator(".kotlin-option").nth(wrongIndex).click();
  assert.equal(await submit.isEnabled(), true);
  await screenshot("kotlin-question");
  const wrongResponse = page.waitForResponse(
    (response) =>
      response.url() === base.origin + "/api" &&
      response.request().postDataJSON()?.command === "answerKotlin",
  );
  await submit.click();
  assert.equal((await (await wrongResponse).json()).correct, false);
  assert.equal((await api("state")).session.exercise, true);
  assert.equal((await api("state")).session.state, "ACTIVE");
  const secondQuestion = await api("kotlinQuestion");
  assert.notEqual(secondQuestion.token, firstQuestion.token);
  assert.notEqual(
    secondQuestion.prompt,
    firstQuestion.prompt,
    "wrong answer gets a different question",
  );
  await assert.rejects(
    api("answerKotlin", { token: firstQuestion.token, option: correctIndex }),
    /устарел|вопрос|token|stale|challenge/i,
  );
  await poll(
    async () =>
      (await dialog.locator(".kotlin-code pre code").textContent()) ===
      codeOf(secondQuestion),
    "next question rendered",
  );
  assert.equal(
    await dialog.getByRole("radio", { checked: true }).count(),
    0,
    "new question resets selection",
  );
  assert.equal(await submit.isDisabled(), true);
  console.log(
    "PASS eight-choice question, wrong answer keeps restrictions, token replay rejected and next question differs",
  );

  const answerIndex = secondQuestion.options.indexOf(
    oracle[secondQuestion.prompt],
  );
  assert.ok(answerIndex >= 0);
  await dialog.locator(".kotlin-option").nth(answerIndex).click();
  const correctResponse = page.waitForResponse(
    (response) =>
      response.url() === base.origin + "/api" &&
      response.request().postDataJSON()?.command === "answerKotlin",
  );
  await submit.click();
  assert.equal((await (await correctResponse).json()).correct, true);
  await page
    .getByRole("heading", {
      name: "One question before you finish",
      exact: true,
    })
    .waitFor({ state: "detached" });
  await page
    .getByRole("heading", { name: "Session complete", exact: true })
    .waitFor();
  state = await api("state");
  assert.equal(state.session.state, "FINISHED");
  assert.deepEqual(state.settings, normalSettings);
  assert.deepEqual(state.profiles, normalProfiles);
  await assert.rejects(
    api("answerKotlin", { token: secondQuestion.token, option: answerIndex }),
  );
  await api("start", { id: profile.id, minutes: 1 });
  state = await api("state");
  assert.equal(state.session.state, "ACTIVE");
  assert.equal(
    state.session.exercise,
    false,
    "subsequent normal session uses normal exit flow",
  );
  assert.equal(state.session.profile.id, profile.id);
  await api("requestExit");
  assert.equal((await api("state")).session.state, "EXIT_PENDING");
  await api("cancelExit");
  assert.equal((await api("state")).session.state, "ACTIVE");
  assert.deepEqual(errors, []);
  console.log(
    "PASS correct answer finishes exercise, ordinary sessions remain usable and no JavaScript errors",
  );
} finally {
  await browser?.close();
  child.kill();
}
