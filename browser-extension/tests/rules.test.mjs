import { test } from "node:test";
import assert from "node:assert/strict";
import { compileRules, domainRegex, replaceRules } from "../rules.mjs";
import { Lease } from "../lease.mjs";
const matches = (d, sub, url) => new RegExp(domainRegex(d, sub), "i").test(url);
test("exact domains exclude suffix tricks, subdomains, credentials and nonstandard protocols", () => {
  for (const url of [
    "https://github.com/",
    "http://github.com:3000/path",
    "HTTPS://GITHUB.COM/",
  ])
    assert.ok(matches("github.com", false, url));
  for (const url of [
    "https://github.com.evil.test/",
    "https://evilgithub.com/",
    "https://www.github.com/",
    "https://github.com@evil.test/",
    "file://github.com/",
  ])
    assert.equal(matches("github.com", false, url), false);
});
test("subdomains and localhost are explicit", () => {
  assert.ok(matches("go.dev", true, "https://pkg.go.dev/"));
  assert.ok(matches("localhost", false, "http://localhost:5173/"));
  assert.equal(
    matches("localhost", false, "http://localhost.evil.test/"),
    false,
  );
  assert.equal(matches("go.dev", false, "https://pkg.go.dev/"), false);
});
test("strict policy explicitly allows main-frame navigation as well as subresources", () => {
  const r = compileRules({
    active: true,
    mode: "strict",
    websites: [{ domain: "go.dev", subdomains: false }],
  });
  assert.equal(r.length, 4);
  assert.equal(r[0].action.redirect.extensionPath, "/blocked.html");
  assert.equal(r[1].action.type, "block");
  assert.equal(r[2].action.type, "allow");
  assert.ok(r[2].priority > r[0].priority);
  assert.deepEqual(r[2].condition.resourceTypes, ["main_frame"]);
  assert.deepEqual(r[3].condition.excludedResourceTypes, ["main_frame"]);
});
test("blocklist only blocks listed domains", () => {
  const r = compileRules({
    active: true,
    mode: "blocklist",
    websites: [{ domain: "example.com", subdomains: true }],
  });
  assert.equal(r.length, 2);
  assert.ok(r.every((x) => x.condition.regexFilter.includes("example\\.com")));
});
test("idle and invalid policies cannot retain restrictions", () => {
  assert.deepEqual(compileRules({ active: false }), []);
  assert.throws(() =>
    compileRules({
      active: true,
      mode: "strict",
      websites: [{ domain: "example.com/path" }],
    }),
  );
  assert.throws(() =>
    compileRules({ active: true, mode: "invalid", websites: [] }),
  );
});
test("session rules atomically replace old rules and clear on fail-open", async () => {
  let options;
  const api = {
    getSessionRules: async () => [{ id: 1 }, { id: 10 }],
    updateSessionRules: async (o) => {
      options = o;
    },
  };
  await replaceRules(api, []);
  assert.deepEqual(options, { removeRuleIds: [1, 10], addRules: [] });
});
test("short lease renews, expires and cannot be extended beyond six seconds", () => {
  let now = 0,
    fired = 0,
    callback;
  const lease = new Lease(
    () => fired++,
    () => now,
    (fn) => {
      callback = fn;
      return 1;
    },
    () => {},
  );
  lease.renew(6000);
  now = 5999;
  lease.check();
  assert.equal(fired, 0);
  now = 6000;
  lease.check();
  assert.equal(fired, 1);
  lease.renew(2000);
  callback();
  assert.equal(fired, 2);
  assert.throws(() => lease.renew(60001));
});
