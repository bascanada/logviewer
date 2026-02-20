import * as vscode from 'vscode';
import * as path from 'path';
import * as os from 'os';
import { ChildProcess, spawn } from 'child_process';

export class ServerManager {
  private process: ChildProcess | undefined;
  private port: number | undefined;
  private readonly extensionPath: string;
  private outputChannel: vscode.OutputChannel;

  constructor(private context: vscode.ExtensionContext) {
    this.extensionPath = context.extensionPath;
    this.outputChannel = vscode.window.createOutputChannel('LogViewer Server');
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
}
