import * as vscode from 'vscode';
import { ServerManager } from './ServerManager';
import { LogViewerSidebarProvider } from './LogViewerSidebar';
import { LogViewerResultsPanel } from './LogViewerResults';

let serverManager: ServerManager | undefined;

export async function activate(context: vscode.ExtensionContext) {
  console.log('LogViewer extension activating...');

  // Initialize server manager
  serverManager = new ServerManager(context);

  // Register sidebar provider
  const sidebarProvider = new LogViewerSidebarProvider(context, serverManager);

  // Handle search requests from sidebar - open results in new editor tab
  sidebarProvider.onSearch(async (params) => {
    console.log('[Extension] onSearch called with params:', params);
    try {
      console.log('[Extension] Starting server...');
      const port = await serverManager!.start();
      console.log('[Extension] Server started on port:', port);
      console.log('[Extension] Creating/showing results panel...');
      LogViewerResultsPanel.createOrShow(context, port, params);
      console.log('[Extension] Results panel created/shown');
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      console.error('[Extension] Failed to open results:', message, err);
      vscode.window.showErrorMessage(`Failed to open results: ${message}`);
    }
  });

  context.subscriptions.push(
    vscode.window.registerWebviewViewProvider('logviewer.sidebar', sidebarProvider)
  );

  // Register command to open sidebar
  context.subscriptions.push(
    vscode.commands.registerCommand('logviewer.open', () => {
      vscode.commands.executeCommand('workbench.view.extension.logviewer');
    })
  );

  // Register command to restart server
  context.subscriptions.push(
    vscode.commands.registerCommand('logviewer.restart', async () => {
      if (serverManager) {
        try {
          await serverManager.restart();
          vscode.window.showInformationMessage('LogViewer server restarted');
        } catch (err) {
          const message = err instanceof Error ? err.message : String(err);
          vscode.window.showErrorMessage(`Failed to restart server: ${message}`);
        }
      }
    })
  );

  // Register command to show output
  context.subscriptions.push(
    vscode.commands.registerCommand('logviewer.showOutput', () => {
      serverManager?.showOutput();
    })
  );

  console.log('LogViewer extension activated');
}

export function deactivate() {
  console.log('LogViewer extension deactivating...');
  serverManager?.stop();
}
