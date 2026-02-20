import * as vscode from 'vscode';
import * as path from 'path';
import * as fs from 'fs';
import * as os from 'os';
import { ServerManager } from './ServerManager';

export class LogViewerSidebarProvider implements vscode.WebviewViewProvider {
  public static readonly viewType = 'logviewer.sidebar';

  private view?: vscode.WebviewView;
  private onSearchCallback?: (params: SearchParams) => void;

  constructor(
    private readonly context: vscode.ExtensionContext,
    private readonly serverManager: ServerManager
  ) {}

  public onSearch(callback: (params: SearchParams) => void) {
    this.onSearchCallback = callback;
  }

  public async resolveWebviewView(
    webviewView: vscode.WebviewView,
    _context: vscode.WebviewViewResolveContext,
    _token: vscode.CancellationToken
  ): Promise<void> {
    this.view = webviewView;

    webviewView.webview.options = {
      enableScripts: true,
      localResourceRoots: [
        vscode.Uri.joinPath(this.context.extensionUri, 'media'),
      ],
    };

    // Start server
    let port: number;
    try {
      port = await this.serverManager.start();
    } catch (err) {
      webviewView.webview.html = this.getErrorHtml(err);
      return;
    }

    webviewView.webview.html = this.getHtmlContent(webviewView.webview, port);

    // Handle messages from sidebar
    webviewView.webview.onDidReceiveMessage((message) => {
      console.log('[LogViewerSidebar] Received message:', message);

      if (message.type === 'search') {
        if (this.onSearchCallback) {
          console.log('[LogViewerSidebar] Calling onSearchCallback with params:', {
            contextId: message.contextId,
            query: message.query,
            timeRange: message.timeRange,
            level: message.level,
            size: message.size,
          });
          this.onSearchCallback({
            contextId: message.contextId,
            query: message.query,
            timeRange: message.timeRange,
            level: message.level,
            size: message.size,
          });
        } else {
          console.error('[LogViewerSidebar] onSearchCallback is not set!');
        }
      } else if (message.type === 'openConfig') {
        this.handleOpenConfig();
      }
    });
  }

  private getHtmlContent(webview: vscode.Webview, port: number): string {
    const mediaPath = path.join(this.context.extensionPath, 'media');
    const sidebarHtmlPath = path.join(mediaPath, 'sidebar.html');

    if (!fs.existsSync(sidebarHtmlPath)) {
      return this.getErrorHtml(new Error('Sidebar not built. Run: make web/build'));
    }

    let html = fs.readFileSync(sidebarHtmlPath, 'utf8');

    const assetsUri = webview.asWebviewUri(
      vscode.Uri.joinPath(this.context.extensionUri, 'media', 'assets')
    );

    html = html.replace(/href="\.\/assets\//g, `href="${assetsUri}/`);
    html = html.replace(/src="\.\/assets\//g, `src="${assetsUri}/`);

    const nonce = this.getNonce();

    const csp = `
      default-src 'none';
      style-src ${webview.cspSource} 'unsafe-inline';
      script-src ${webview.cspSource} 'nonce-${nonce}';
      connect-src http://localhost:${port} http://127.0.0.1:${port};
      img-src ${webview.cspSource} data:;
      font-src ${webview.cspSource};
    `.replace(/\s+/g, ' ').trim();

    const portScript = `<script nonce="${nonce}">window.LOGVIEWER_PORT = ${port};</script>`;
    const cspMeta = `<meta http-equiv="Content-Security-Policy" content="${csp}">`;

    html = html.replace('</head>', `${cspMeta}\n${portScript}\n</head>`);

    return html;
  }

  private getErrorHtml(error: unknown): string {
    const message = error instanceof Error ? error.message : String(error);
    return `
      <!DOCTYPE html>
      <html>
      <head><meta charset="UTF-8"></head>
      <body style="padding: 10px; color: var(--vscode-errorForeground);">
        <p>${this.escapeHtml(message)}</p>
      </body>
      </html>
    `;
  }

  private getNonce(): string {
    let text = '';
    const possible = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
    for (let i = 0; i < 32; i++) {
      text += possible.charAt(Math.floor(Math.random() * possible.length));
    }
    return text;
  }

  private escapeHtml(text: string): string {
    return text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
  }

  /**
   * Get the config file path to open
   * Returns the first config file found based on settings/environment
   */
  private getConfigPath(): string | undefined {
    const config = vscode.workspace.getConfiguration('logviewer');
    const configPath = config.get<string>('configPath');

    // If explicit config path is set, use it
    if (configPath && configPath.trim() !== '') {
      return configPath;
    }

    // Check LOGVIEWER_CONFIG environment variable
    const envConfig = process.env.LOGVIEWER_CONFIG;
    if (envConfig) {
      // Return first file from colon-separated list
      const envPaths = envConfig.split(path.delimiter);
      if (envPaths.length > 0 && envPaths[0].trim() !== '') {
        return envPaths[0];
      }
    }

    // Default: ~/.logviewer/config.yaml
    const homeDir = os.homedir();
    const defaultConfig = path.join(homeDir, '.logviewer', 'config.yaml');

    // Check if default config exists
    if (fs.existsSync(defaultConfig)) {
      return defaultConfig;
    }

    // If default doesn't exist, return path anyway (user can create it)
    return defaultConfig;
  }

  /**
   * Handle opening the config file in VS Code
   */
  private async handleOpenConfig(): Promise<void> {
    const configPath = this.getConfigPath();

    if (!configPath) {
      vscode.window.showErrorMessage('No LogViewer configuration file found');
      return;
    }

    try {
      const uri = vscode.Uri.file(configPath);

      // Check if file exists
      if (!fs.existsSync(configPath)) {
        // Ask user if they want to create it
        const create = await vscode.window.showInformationMessage(
          `Config file does not exist: ${configPath}. Create it?`,
          'Create',
          'Cancel'
        );

        if (create === 'Create') {
          // Create directory if needed
          const dir = path.dirname(configPath);
          if (!fs.existsSync(dir)) {
            fs.mkdirSync(dir, { recursive: true });
          }

          // Create empty file with basic structure
          const template = `# LogViewer Configuration
# See documentation for full configuration options

clients: {}

contexts: {}
`;
          fs.writeFileSync(configPath, template, 'utf8');
          vscode.window.showInformationMessage(`Created config file: ${configPath}`);
        } else {
          return;
        }
      }

      // Open the file
      const document = await vscode.workspace.openTextDocument(uri);
      await vscode.window.showTextDocument(document, {
        preview: false,
        viewColumn: vscode.ViewColumn.One
      });
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      vscode.window.showErrorMessage(`Failed to open config file: ${message}`);
    }
  }
}

export interface SearchParams {
  contextId: string;
  query?: string;
  timeRange?: string;
  level?: string;
  size?: number;
}
