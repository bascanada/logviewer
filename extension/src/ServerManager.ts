import * as vscode from 'vscode';
import * as path from 'path';
import * as os from 'os';
import * as fs from 'fs';
import { ChildProcess, spawn } from 'child_process';

export class ServerManager {
  private process: ChildProcess | undefined;
  private port: number | undefined;
  private readonly extensionPath: string;
  private outputChannel: vscode.OutputChannel;
  private configWatcher: vscode.FileSystemWatcher | undefined;
  private isRestarting = false;

  constructor(private context: vscode.ExtensionContext) {
    this.extensionPath = context.extensionPath;
    this.outputChannel = vscode.window.createOutputChannel('LogViewer Server');
    this.setupConfigWatcher();

    // Watch for config path setting changes
    context.subscriptions.push(
      vscode.workspace.onDidChangeConfiguration((e) => {
        if (e.affectsConfiguration('logviewer.configPath') ||
            e.affectsConfiguration('logviewer.watchConfigFiles')) {
          this.outputChannel.appendLine('LogViewer settings changed, recreating config watcher...');
          this.disposeConfigWatcher();
          this.setupConfigWatcher();
        }
      })
    );
  }

  /**
   * Get the binary path for the current platform
   */
  private getBinaryPath(): string {
    const platform = os.platform();
    const arch = os.arch();

    let binaryName: string;

    switch (platform) {
      case 'darwin':
        binaryName =
          arch === 'arm64'
            ? 'logviewer-darwin-arm64'
            : 'logviewer-darwin-amd64';
        break;
      case 'linux':
        binaryName =
          arch === 'arm64'
            ? 'logviewer-linux-arm64'
            : 'logviewer-linux-amd64';
        break;
      case 'win32':
        binaryName =
          arch === 'arm64'
            ? 'logviewer-windows-arm64.exe'
            : 'logviewer-windows-amd64.exe';
        break;
      default:
        throw new Error(`Unsupported platform: ${platform}`);
    }

    return path.join(this.extensionPath, 'bin', binaryName);
  }

  /**
   * Start the LogViewer server process
   */
  async start(): Promise<number> {
    if (this.process && this.port) {
      return this.port;
    }

    const binaryPath = this.getBinaryPath();
    this.outputChannel.appendLine(`Starting server: ${binaryPath}`);

    return new Promise((resolve, reject) => {
      // Get config path from workspace settings or use default
      const configPath = vscode.workspace
        .getConfiguration('logviewer')
        .get<string>('configPath');

      const args = ['server', '--port', '0']; // Port 0 = random available port
      if (configPath) {
        args.push('--config', configPath);
      }

      try {
        this.process = spawn(binaryPath, args, {
          env: { ...process.env },
        });
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err);
        this.outputChannel.appendLine(`Failed to spawn process: ${message}`);
        reject(new Error(`Failed to start server: ${message}`));
        return;
      }

      let portDiscovered = false;

      this.process.stdout?.on('data', (data: Buffer) => {
        const output = data.toString();
        this.outputChannel.appendLine(output);

        // Parse port from server output
        // Expected format: "Server listening on :8080" or similar
        if (!portDiscovered) {
          const portMatch = output.match(
            /(?:listening|started|port)[^\d]*(\d+)/i
          );
          if (portMatch) {
            this.port = parseInt(portMatch[1], 10);
            portDiscovered = true;
            this.outputChannel.appendLine(`Discovered port: ${this.port}`);
            resolve(this.port);
          }
        }
      });

      this.process.stderr?.on('data', (data: Buffer) => {
        this.outputChannel.appendLine(`[stderr] ${data.toString()}`);
      });

      this.process.on('error', (err) => {
        this.outputChannel.appendLine(`Process error: ${err.message}`);
        reject(err);
      });

      this.process.on('exit', (code) => {
        this.outputChannel.appendLine(`Process exited with code: ${code}`);
        this.process = undefined;
        this.port = undefined;
      });

      // Timeout if port not discovered
      setTimeout(() => {
        if (!portDiscovered) {
          reject(new Error('Timeout waiting for server to start'));
        }
      }, 10000);
    });
  }

  /**
   * Stop the server process
   */
  stop(): void {
    if (this.process) {
      this.outputChannel.appendLine('Stopping server...');
      this.process.kill();
      this.process = undefined;
      this.port = undefined;
    }
  }

  /**
   * Get the current port (if server is running)
   */
  getPort(): number | undefined {
    return this.port;
  }

  /**
   * Check if server is running
   */
  isRunning(): boolean {
    return this.process !== undefined && this.port !== undefined;
  }

  /**
   * Restart the server
   */
  async restart(): Promise<number> {
    this.stop();
    return this.start();
  }

  /**
   * Show the output channel
   */
  showOutput(): void {
    this.outputChannel.show();
  }

  /**
   * Get configuration file paths to watch
   * Returns all potential config files that might affect the server
   */
  private getConfigPaths(): string[] {
    const config = vscode.workspace.getConfiguration('logviewer');
    const configPath = config.get<string>('configPath');
    const paths: string[] = [];

    // If explicit config path is set, watch it
    if (configPath && configPath.trim() !== '') {
      paths.push(configPath);
      return paths;
    }

    // Check LOGVIEWER_CONFIG environment variable
    const envConfig = process.env.LOGVIEWER_CONFIG;
    if (envConfig) {
      // Can be colon-separated list
      const envPaths = envConfig.split(path.delimiter);
      paths.push(...envPaths);
      return paths;
    }

    // Default paths: ~/.logviewer/config.yaml and ~/.logviewer/configs/*.yaml
    const homeDir = os.homedir();
    const logviewerDir = path.join(homeDir, '.logviewer');
    const mainConfig = path.join(logviewerDir, 'config.yaml');
    const configsDir = path.join(logviewerDir, 'configs');

    // Check if main config exists
    if (fs.existsSync(mainConfig)) {
      paths.push(mainConfig);
    }

    // Check if configs directory exists and add all .yaml/.yml files
    if (fs.existsSync(configsDir)) {
      try {
        const files = fs.readdirSync(configsDir);
        for (const file of files) {
          if (file.endsWith('.yaml') || file.endsWith('.yml')) {
            paths.push(path.join(configsDir, file));
          }
        }
      } catch (err) {
        // Ignore errors reading directory
      }
    }

    return paths;
  }

  /**
   * Set up file watcher for configuration files
   */
  private setupConfigWatcher(): void {
    const config = vscode.workspace.getConfiguration('logviewer');
    const watchEnabled = config.get<boolean>('watchConfigFiles', true);

    if (!watchEnabled) {
      return;
    }

    const configPath = config.get<string>('configPath');
    const homeDir = os.homedir();

    // Determine watch pattern based on config
    let watchPattern: string;

    if (configPath && configPath.trim() !== '') {
      // Watch explicit config file
      watchPattern = configPath;
    } else {
      // Watch default .logviewer directory
      watchPattern = path.join(homeDir, '.logviewer', '**/*.{yaml,yml}');
    }

    this.configWatcher = vscode.workspace.createFileSystemWatcher(
      watchPattern,
      false, // ignoreCreateEvents
      false, // ignoreChangeEvents
      false  // ignoreDeleteEvents
    );

    const handleConfigChange = async (uri: vscode.Uri) => {
      // Only handle if this is one of our watched config files
      const configPaths = this.getConfigPaths();
      const changedPath = uri.fsPath;

      if (!configPaths.some(p => p === changedPath)) {
        return;
      }

      if (this.isRestarting) {
        return; // Already restarting
      }

      this.outputChannel.appendLine(`Config file changed: ${changedPath}`);

      const autoRestart = config.get<boolean>('autoRestartOnConfigChange', false);

      if (autoRestart && this.isRunning()) {
        // Auto-restart
        this.outputChannel.appendLine('Auto-restarting server due to config change...');
        await this.handleConfigChangeRestart(true);
      } else if (this.isRunning()) {
        // Show notification
        const action = await vscode.window.showInformationMessage(
          'LogViewer configuration file changed. Restart server to apply changes?',
          'Restart Now',
          'Later'
        );

        if (action === 'Restart Now') {
          await this.handleConfigChangeRestart(false);
        }
      }
    };

    this.configWatcher.onDidChange(handleConfigChange);
    this.configWatcher.onDidCreate(handleConfigChange);
    this.configWatcher.onDidDelete((uri) => {
      this.outputChannel.appendLine(`Config file deleted: ${uri.fsPath}`);
      vscode.window.showWarningMessage(
        `LogViewer config file deleted: ${path.basename(uri.fsPath)}. Server may not work correctly.`
      );
    });

    this.context.subscriptions.push(this.configWatcher);
  }

  /**
   * Handle config change restart
   */
  private async handleConfigChangeRestart(isAuto: boolean): Promise<void> {
    if (this.isRestarting) {
      return;
    }

    this.isRestarting = true;

    try {
      const restartMsg = isAuto ? 'Auto-restarting' : 'Restarting';
      this.outputChannel.appendLine(`${restartMsg} server...`);

      await this.restart();

      this.outputChannel.appendLine('Server restarted successfully');

      if (!isAuto) {
        vscode.window.showInformationMessage('LogViewer server restarted successfully');
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      this.outputChannel.appendLine(`Failed to restart server: ${message}`);
      vscode.window.showErrorMessage(`Failed to restart LogViewer server: ${message}`);
    } finally {
      this.isRestarting = false;
    }
  }

  /**
   * Dispose of config watcher only
   */
  private disposeConfigWatcher(): void {
    if (this.configWatcher) {
      this.configWatcher.dispose();
      this.configWatcher = undefined;
    }
  }

  /**
   * Dispose of resources
   */
  dispose(): void {
    this.stop();
    this.disposeConfigWatcher();
  }
}
