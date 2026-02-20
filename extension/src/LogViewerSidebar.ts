import * as vscode from 'vscode';
import * as path from 'path';
import * as fs from 'fs';
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
}

export interface SearchParams {
  contextId: string;
  query?: string;
  timeRange?: string;
  level?: string;
  size?: number;
}
