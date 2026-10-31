"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || (function () {
    var ownKeys = function(o) {
        ownKeys = Object.getOwnPropertyNames || function (o) {
            var ar = [];
            for (var k in o) if (Object.prototype.hasOwnProperty.call(o, k)) ar[ar.length] = k;
            return ar;
        };
        return ownKeys(o);
    };
    return function (mod) {
        if (mod && mod.__esModule) return mod;
        var result = {};
        if (mod != null) for (var k = ownKeys(mod), i = 0; i < k.length; i++) if (k[i] !== "default") __createBinding(result, mod, k[i]);
        __setModuleDefault(result, mod);
        return result;
    };
})();
Object.defineProperty(exports, "__esModule", { value: true });
exports.activate = activate;
exports.deactivate = deactivate;
const fs = __importStar(require("fs"));
const path = __importStar(require("path"));
const vscode = __importStar(require("vscode"));
const node_1 = require("vscode-languageclient/node");
let client;
function serverCommand() {
    const configured = vscode.workspace.getConfiguration("veles").get("serverPath") || "veles";
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
async function startClient(context) {
    const command = serverCommand();
    const serverOptions = {
        run: { command, args: ["lsp"] },
        debug: { command, args: ["lsp"] },
    };
    const clientOptions = {
        documentSelector: [{ scheme: "file", language: "veles" }],
        // the server reads `allocationHints` at start; later changes to the
        // `veles` settings reach it as workspace/didChangeConfiguration
        initializationOptions: {
            allocationHints: vscode.workspace.getConfiguration("veles").get("inlayHints.allocations") ?? false,
        },
        synchronize: {
            configurationSection: "veles",
            fileEvents: vscode.workspace.createFileSystemWatcher("**/{*.vs,*.vss,veles.toml}"),
        },
    };
    client = new node_1.LanguageClient("veles", "Veles Language Server", serverOptions, clientOptions);
    try {
        await client.start();
    }
    catch (err) {
        const localName = process.platform === "win32" ? "veles.exe" : "veles";
        void vscode.window.showErrorMessage(`Veles: could not start "${command} lsp". In the compiler repository, run "go build -o ${localName} ." at the workspace root, then run "Veles: Restart Language Server". Other workspaces can set "veles.serverPath" to that binary's absolute path; no global installation is required. (${err})`);
        client = undefined;
    }
}
function runInTerminal(subcommand) {
    const editor = vscode.window.activeTextEditor;
    if (!editor || editor.document.languageId !== "veles") {
        void vscode.window.showInformationMessage("Open a .vs or .vss file first.");
        return;
    }
    const dir = path.dirname(editor.document.uri.fsPath);
    const term = vscode.window.terminals.find((t) => t.name === "veles") ?? vscode.window.createTerminal("veles");
    term.show(true);
    term.sendText(`${serverCommand()} ${subcommand} "${dir}"`);
}
async function activate(context) {
    context.subscriptions.push(vscode.commands.registerCommand("veles.restartServer", async () => {
        if (client) {
            await client.stop();
            client = undefined;
        }
        await startClient(context);
    }), vscode.commands.registerCommand("veles.run", () => runInTerminal("run")), vscode.commands.registerCommand("veles.test", () => runInTerminal("test")));
    await startClient(context);
}
async function deactivate() {
    if (client) {
        await client.stop();
        client = undefined;
    }
}
//# sourceMappingURL=extension.js.map