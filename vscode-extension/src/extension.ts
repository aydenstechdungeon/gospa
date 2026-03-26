import * as vscode from 'vscode';
import { LanguageClient, type LanguageClientOptions, type ServerOptions } from 'vscode-languageclient/node';

let client: LanguageClient;

export function activate(context: vscode.ExtensionContext) {
    const lspPath = vscode.workspace.getConfiguration('gospa').get<string>('lsp.path') || 'gospa-lsp';

    const serverOptions: ServerOptions = {
        run: { command: lspPath },
        debug: { command: lspPath }
    };

    const clientOptions: LanguageClientOptions = {
        documentSelector: [{ scheme: 'file', language: 'gospa' }],
        synchronize: {
            fileEvents: vscode.workspace.createFileSystemWatcher('**/*.gospa')
        }
    };

    client = new LanguageClient(
        'gospaLsp',
        'GoSPA Language Server',
        serverOptions,
        clientOptions
    );

    client.start();
}

export function deactivate(): Thenable<void> | undefined {
    if (!client) {
        return undefined;
    }
    return client.stop();
}
