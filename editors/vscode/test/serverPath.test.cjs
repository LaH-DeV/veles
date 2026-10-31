const assert = require("node:assert/strict");
const { readFileSync } = require("node:fs");
const path = require("node:path");
const test = require("node:test");
const vm = require("node:vm");
const ts = require("typescript");

const source = ts.transpileModule(
  readFileSync(path.join(__dirname, "../src/extension.ts"), "utf8"),
  {
    compilerOptions: {
      module: ts.ModuleKind.CommonJS,
      target: ts.ScriptTarget.ES2020,
    },
  },
).outputText;

async function startup({
  roots = [],
  files = [],
  configured = "veles",
  fail = false,
  platform = "darwin",
} = {}) {
  let serverOptions;
  let errorMessage;
  const existing = new Set(files);
  const disposable = { dispose() {} };
  const vscode = {
    workspace: {
      workspaceFolders: roots.map((root) => ({ uri: { fsPath: root } })),
      getConfiguration: () => ({
        get: (key) => (key === "serverPath" ? configured : undefined),
      }),
      createFileSystemWatcher: () => disposable,
    },
    commands: { registerCommand: () => disposable },
    window: {
      showErrorMessage: (message) => {
        errorMessage = message;
      },
    },
  };
  class LanguageClient {
    constructor(id, name, options) {
      serverOptions = options;
    }
    async start() {
      if (fail) throw new Error("compiler not found");
    }
  }
  const sandbox = {
    exports: {},
    process: { platform },
    require: (name) => {
      if (name === "fs")
        return { existsSync: (candidate) => existing.has(candidate) };
      if (name === "path") return path;
      if (name === "vscode") return vscode;
      if (name === "vscode-languageclient/node") return { LanguageClient };
      throw new Error(`Unexpected dependency: ${name}`);
    },
  };
  vm.runInNewContext(source, sandbox, { filename: "extension.js" });
  await sandbox.exports.activate({ subscriptions: [] });
  return { command: serverOptions.run.command, errorMessage };
}

test("startup finds a workspace-local compiler without a global installation", async () => {
  const root = path.resolve("workspace with spaces");
  const compiler = path.join(root, "veles");
  assert.equal(
    (await startup({ roots: [root], files: [compiler] })).command,
    compiler,
  );
});

test("startup accepts the Windows compiler name at the workspace root", async () => {
  const root = path.resolve("workspace");
  const compiler = path.join(root, "veles.exe");
  assert.equal(
    (await startup({ roots: [root], files: [compiler], platform: "win32" }))
      .command,
    compiler,
  );
});

test("an explicit serverPath takes precedence over workspace discovery", async () => {
  const root = path.resolve("workspace");
  const configured = path.resolve("another build", "veles");
  assert.equal(
    (
      await startup({
        roots: [root],
        files: [path.join(root, "veles")],
        configured,
      })
    ).command,
    configured,
  );
});

test("missing compiler guidance explains local setup and restart rather than requiring PATH", async () => {
  const result = await startup({
    roots: [path.resolve("workspace")],
    fail: true,
  });
  assert.equal(result.command, "veles");
  assert.match(result.errorMessage, /go build -o veles \./);
  assert.match(result.errorMessage, /Restart Language Server/);
  assert.match(result.errorMessage, /no global installation is required/);
});
