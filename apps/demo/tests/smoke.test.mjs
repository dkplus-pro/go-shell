import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const packageJson = JSON.parse(await readFile(new URL("../package.json", import.meta.url), "utf8"));
const pageSource = await readFile(new URL("../src/routes/page.tsx", import.meta.url), "utf8");
const layoutSource = await readFile(new URL("../src/routes/layout.tsx", import.meta.url), "utf8");
const configSource = await readFile(new URL("../modern.config.ts", import.meta.url), "utf8");

test("demo app exposes standard lifecycle scripts", () => {
  assert.equal(packageJson.scripts.dev, "modern dev");
  assert.equal(packageJson.scripts.build, "modern build");
  assert.equal(packageJson.scripts.typecheck, "tsc --noEmit");
  assert.ok(!("build:github-pages" in packageJson.scripts));
  assert.ok(!("deploy:github-pages" in packageJson.scripts));
});

test("demo app has visible hello-world content and Modern.js app tools configured", () => {
  assert.match(pageSource, /Hello from the demo app/);
  assert.match(layoutSource, /Outlet/);
  assert.match(configSource, /appTools\(\)/);
});

test("demo app config keeps dev server and flat output settings without pages logic", () => {
  assert.match(configSource, /server: { port: devServerPort }/);
  assert.match(configSource, /outputStructure: "flat"/);
  assert.match(configSource, /html: ""/);
  assert.ok(!/GITHUB_PAGES|assetPrefix/.test(configSource));
});
