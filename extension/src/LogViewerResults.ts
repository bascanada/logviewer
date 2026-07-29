import * as vscode from 'vscode';
import * as path from 'path';
import * as fs from 'fs';
import { SearchParams } from './LogViewerSidebar';

export class LogViewerResultsPanel {
  public static currentPanels: Map<string, LogViewerResultsPanel> = new Map();

  private readonly panel: vscode.WebviewPanel;
  private readonly extensionPath: string;
  private readonly port: number;
  private readonly panelHash: string;
  private disposables: vscode.Disposable[] = [];

  /**
   * Create a hash from search parameters
   */
  private static createHash(searchParams: SearchParams): string {
    const parts = [
      searchParams.contextId,
      searchParams.query || '',
      searchParams.timeRange || '1h',
      searchParams.level || '',
      searchParams.size?.toString() || '100'
    ];
    return parts.join('|');
  }

  public static createOrShow(
    context: vscode.ExtensionContext,
    port: number,
    searchParams: SearchParams
  ) {
    console.log('[LogViewerResults] createOrShow called with params:', searchParams);

    // Create hash based on search parameters
    const panelHash = LogViewerResultsPanel.createHash(searchParams);
    console.log('[LogViewerResults] Panel hash:', panelHash);
    console.log('[LogViewerResults] Current panels count:', LogViewerResultsPanel.currentPanels.size);

    // Check if panel with this hash already exists
    const existingPanel = LogViewerResultsPanel.currentPanels.get(panelHash);
    if (existingPanel) {
      // Panel exists, just reveal it
      console.log('[LogViewerResults] Found existing panel, revealing it');
      existingPanel.panel.reveal(vscode.ViewColumn.One);
      return;
    }

    console.log('[LogViewerResults] No existing panel found, creating new one');

    // Create title from search params
    const title = searchParams.query
      ? `Logs: ${searchParams.query.substring(0, 30)}`
      : `Logs: ${searchParams.contextId}`;
    console.log('[LogViewerResults] Panel title:', title);

    // Create new panel
    console.log('[LogViewerResults] Creating webview panel...');
    const panel = vscode.window.createWebviewPanel(
      'logviewer.results',
      title,
      vscode.ViewColumn.One,
      {
        enableScripts: true,
        retainContextWhenHidden: true,
        localResourceRoots: [
          vscode.Uri.joinPath(context.extensionUri, 'media'),
        ],
      }
    );
    console.log('[LogViewerResults] Webview panel created');

    console.log('[LogViewerResults] Creating LogViewerResultsPanel instance...');
    const resultsPanel = new LogViewerResultsPanel(
      panel,
      context.extensionPath,
      port,
      searchParams,
      panelHash
    );
    console.log('[LogViewerResults] LogViewerResultsPanel instance created');

    LogViewerResultsPanel.currentPanels.set(panelHash, resultsPanel);
    console.log('[LogViewerResults] Panel added to map, total panels:', LogViewerResultsPanel.currentPanels.size);
  }

  private constructor(
    panel: vscode.WebviewPanel,
    extensionPath: string,
    port: number,
    searchParams: SearchParams,
    panelHash: string
  ) {
    console.log('[LogViewerResults] Constructor called');
    this.panel = panel;
    this.extensionPath = extensionPath;
    this.port = port;
    this.panelHash = panelHash;

    console.log('[LogViewerResults] Getting HTML content...');
    this.panel.webview.html = this.getHtmlContent(searchParams);
    console.log('[LogViewerResults] HTML content set');

    this.panel.onDidDispose(() => this.dispose(), null, this.disposables);
    console.log('[LogViewerResults] Constructor completed');
  }

  public dispose() {
    console.log('[LogViewerResults] Disposing panel with hash:', this.panelHash);

    // Remove from the map
    LogViewerResultsPanel.currentPanels.delete(this.panelHash);
    console.log('[LogViewerResults] Removed from map, remaining panels:', LogViewerResultsPanel.currentPanels.size);

    // Dispose the panel
    this.panel.dispose();

    // Dispose all disposables
    while (this.disposables.length) {
      const x = this.disposables.pop();
      if (x) {
        x.dispose();
      }
    }
    console.log('[LogViewerResults] Dispose completed');
  }

  private getHtmlContent(searchParams: SearchParams): string {
    const mediaPath = path.join(this.extensionPath, 'media');
    const resultsHtmlPath = path.join(mediaPath, 'results.html');

    if (!fs.existsSync(resultsHtmlPath)) {
      return this.getErrorHtml(new Error('Results panel not built. Run: make web/build'));
    }

    let html = fs.readFileSync(resultsHtmlPath, 'utf8');

    const assetsUri = this.panel.webview.asWebviewUri(
      vscode.Uri.file(path.join(this.extensionPath, 'media', 'assets'))
    );

    html = html.replace(/href="\.\/assets\//g, `href="${assetsUri}/`);
    html = html.replace(/src="\.\/assets\//g, `src="${assetsUri}/`);

    const nonce = this.getNonce();

    const csp = `
      default-src 'none';
      style-src ${this.panel.webview.cspSource} 'unsafe-inline';
      script-src ${this.panel.webview.cspSource} 'nonce-${nonce}';
      connect-src http://localhost:${this.port} http://127.0.0.1:${this.port};
      img-src ${this.panel.webview.cspSource} data:;
      font-src ${this.panel.webview.cspSource};
    `.replace(/\s+/g, ' ').trim();

    const searchScript = `
      <script nonce="${nonce}">
        window.LOGVIEWER_PORT = ${this.port};
        window.LOGVIEWER_SEARCH = ${JSON.stringify(searchParams)};
      </script>
    `;
    const cspMeta = `<meta http-equiv="Content-Security-Policy" content="${csp}">`;

    html = html.replace('</head>', `${cspMeta}\n${searchScript}\n</head>`);

    return html;
  }

  private getErrorHtml(error: unknown): string {
    const message = error instanceof Error ? error.message : String(error);
    return `
      <!DOCTYPE html>
      <html>
      <head><meta charset="UTF-8"></head>
      <body style="padding: 20px; color: var(--vscode-errorForeground);">
        <h2>Error</h2>
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
