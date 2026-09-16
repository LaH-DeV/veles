import * as fs from "fs";
import * as path from "path";
import * as vscode from "vscode";
import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
} from "vscode-languageclient/node";

let client: LanguageClient | undefined;

function serverCommand(): string {
  const configured = vscode.workspace.getConfiguration("veles").get<string>("serverPath") || "veles";
  if (configured !== "veles") {
    return configured;
  }
  // Developer convenience: a compiler built at the workspace root wins over PATH.
  for (const folder of vscode.workspace.workspaceFolders ?? []) {
    for (const name of ["veles.exe", "veles"]) {
      const candidate = path.join(folder.uri.fsPath, name);
      if (fs.existsSync(candidate)) {
        return candidate;
      }
    }
  }
  return configured;
}

async function startClient(context: vscode.ExtensionContext): Promise<void> {
  const command = serverCommand();
  const serverOptions: ServerOptions = {
    run: { command, args: ["lsp"] },
    debug: { command, args: ["lsp"] },
  };
  const clientOptions: LanguageClientOptions = {
    documentSelector: [{ scheme: "file", language: "veles" }],
    synchronize: {
      fileEvents: vscode.workspace.createFileSystemWatcher("**/{*.vs,veles.toml}"),
    },
  };
  client = new LanguageClient("veles", "Veles Language Server", serverOptions, clientOptions);
  try {
    await client.start();
  } catch (err) {
    void vscode.window.showErrorMessage(
      `Veles: could not start "${command} lsp". Build the compiler (go build -o veles .) and put it on PATH, or set "veles.serverPath". (${err})`
    );
    client = undefined;
  }
}

function runInTerminal(subcommand: string): void {
  const editor = vscode.window.activeTextEditor;
  if (!editor || editor.document.languageId !== "veles") {
    void vscode.window.showInformationMessage("Open a .vs file first.");
    return;
  }
  const dir = path.dirname(editor.document.uri.fsPath);
  const term = vscode.window.terminals.find((t) => t.name === "veles") ?? vscode.window.createTerminal("veles");
  term.show(true);
  term.sendText(`${serverCommand()} ${subcommand} "${dir}"`);
}

export async function activate(context: vscode.ExtensionContext): Promise<void> {
  context.subscriptions.push(
    vscode.commands.registerCommand("veles.restartServer", async () => {
      if (client) {
        await client.stop();
        client = undefined;
      }
      await startClient(context);
    }),
    vscode.commands.registerCommand("veles.run", () => runInTerminal("run")),
    vscode.commands.registerCommand("veles.test", () => runInTerminal("test"))
  );
  await startClient(context);
}

export async function deactivate(): Promise<void> {
  if (client) {
    await client.stop();
    client = undefined;
  }
}
