# LogViewer Extension Settings

This document describes all available settings for the LogViewer VS Code extension.

## Configuration Settings

### `logviewer.configPath`

**Type**: `string`
**Default**: `""`
**Scope**: Window

Path to LogViewer configuration file.

**Behavior**:
- If **empty**: Uses default path `~/.logviewer/config.yaml` and auto-discovers files in `~/.logviewer/configs/*.yaml`
- If **set**: Uses only the specified file path
- **Environment variable**: Can be overridden by `LOGVIEWER_CONFIG` environment variable (supports colon-separated list)

**Example**:
```json
{
  "logviewer.configPath": "/path/to/my/custom-config.yaml"
}
```

**Multiple config files** (via environment):
```bash
export LOGVIEWER_CONFIG="/path/to/config1.yaml:/path/to/config2.yaml"
```

---

### `logviewer.autoStart`

**Type**: `boolean`
**Default**: `true`
**Scope**: Window

Automatically start the LogViewer server when opening the sidebar panel.

**Behavior**:
- `true`: Server starts automatically on first search
- `false`: User must manually start server via "LogViewer: Restart Server" command

**Example**:
```json
{
  "logviewer.autoStart": false
}
```

---

### `logviewer.watchConfigFiles`

**Type**: `boolean`
**Default**: `true`
**Scope**: Window

Watch configuration file(s) for changes and notify when restart is needed.

**Behavior**:
- `true`: Watches config files and shows notification when changed
- `false`: No file watching (manual restart required after config changes)

**What is watched**:
- If `logviewer.configPath` is set: watches that specific file
- If empty: watches `~/.logviewer/config.yaml` and `~/.logviewer/configs/*.yaml`

**Example**:
```json
{
  "logviewer.watchConfigFiles": false
}
```

---

### `logviewer.autoRestartOnConfigChange`

**Type**: `boolean`
**Default**: `false`
**Scope**: Window

Automatically restart the server when the configuration file changes.

**Behavior**:
- `true`: Server automatically restarts when config file is modified (no prompt)
- `false`: Shows notification with "Restart Now" button when config changes

**Requires**: `logviewer.watchConfigFiles` must be `true`

**Example**:
```json
{
  "logviewer.autoRestartOnConfigChange": true,
  "logviewer.watchConfigFiles": true
}
```

---

## Commands

These commands are available in the Command Palette (`Cmd+Shift+P` or `Ctrl+Shift+P`):

### `LogViewer: Open`
Opens the LogViewer sidebar panel.

### `LogViewer: Restart Server`
Manually restart the LogViewer backend server. Useful after:
- Changing configuration file
- Updating credentials
- Troubleshooting connection issues

### `LogViewer: Show Server Output`
Opens the LogViewer Server output channel to view server logs, errors, and debugging information.

---

## Configuration File Watching

The extension automatically watches configuration files for changes. Here's how it works:

### Watched Files

**When `logviewer.configPath` is empty** (default):
```
~/.logviewer/config.yaml
~/.logviewer/configs/*.yaml
~/.logviewer/configs/*.yml
```

**When `logviewer.configPath` is set**:
```
/path/to/your/custom-config.yaml
```

### Change Detection

When a watched config file is modified:

1. **With `autoRestartOnConfigChange: false`** (default):
   - Shows notification: "LogViewer configuration file changed. Restart server to apply changes?"
   - User can click "Restart Now" or "Later"
   - Message logged to "LogViewer Server" output channel

2. **With `autoRestartOnConfigChange: true`**:
   - Server automatically restarts in background
   - No user interaction required
   - Logs restart in "LogViewer Server" output channel

### File Deletion

If a watched config file is deleted:
- Warning notification shown
- Server continues running with last loaded config
- Server may fail on next restart

---

## Recommended Settings

### For Development/Testing
```json
{
  "logviewer.configPath": "",
  "logviewer.autoStart": true,
  "logviewer.watchConfigFiles": true,
  "logviewer.autoRestartOnConfigChange": true
}
```
**Why**: Automatically restarts when you modify config during development.

### For Production/Stability
```json
{
  "logviewer.configPath": "/etc/logviewer/config.yaml",
  "logviewer.autoStart": true,
  "logviewer.watchConfigFiles": true,
  "logviewer.autoRestartOnConfigChange": false
}
```
**Why**: Prompts before restarting, preventing unexpected disruptions.

### For Shared Configurations
```json
{
  "logviewer.configPath": "/shared/team/logviewer-config.yaml",
  "logviewer.autoStart": true,
  "logviewer.watchConfigFiles": false,
  "logviewer.autoRestartOnConfigChange": false
}
```
**Why**: Disables watching to avoid frequent restarts when multiple team members modify shared config.

---

## Troubleshooting

### Config Changes Not Detected

**Problem**: Modified config file but no notification appears.

**Solutions**:
1. Verify `logviewer.watchConfigFiles` is `true`
2. Check the file path in settings matches the actual file location
3. View "LogViewer Server" output channel for watch setup messages
4. Try manually restarting: Command Palette → "LogViewer: Restart Server"

### Auto-Restart Not Working

**Problem**: File changes detected but server doesn't restart automatically.

**Solutions**:
1. Verify `logviewer.autoRestartOnConfigChange` is `true`
2. Verify `logviewer.watchConfigFiles` is `true`
3. Check "LogViewer Server" output for error messages
4. Ensure server is actually running (try a search first)

### Server Uses Old Config After Change

**Problem**: Restarted server but still uses old configuration.

**Solutions**:
1. Verify you modified the correct config file (check `logviewer.configPath`)
2. Check for YAML syntax errors in the config file
3. View "LogViewer Server" output for config load errors
4. Try: Command Palette → "LogViewer: Restart Server"

### Multiple Config Files Conflicting

**Problem**: Behavior doesn't match expectations with multiple config files.

**Solutions**:
1. Set explicit `logviewer.configPath` to use only one file
2. Check `LOGVIEWER_CONFIG` environment variable (overrides setting)
3. Understand merge order:
   - `~/.logviewer/config.yaml` loaded first
   - `~/.logviewer/configs/*.yaml` loaded alphabetically
   - Later files override earlier ones

---

## Environment Variables

### `LOGVIEWER_CONFIG`

Overrides `logviewer.configPath` setting. Supports colon-separated list of config files.

**Example** (macOS/Linux):
```bash
export LOGVIEWER_CONFIG="/etc/logviewer.yaml:/home/user/custom.yaml"
code
```

**Example** (Windows):
```cmd
set LOGVIEWER_CONFIG=C:\config\logviewer.yaml;D:\custom.yaml
code
```

**Precedence**:
1. `LOGVIEWER_CONFIG` (highest)
2. `logviewer.configPath` setting
3. Default `~/.logviewer/` (lowest)

---

## Advanced Usage

### Conditional Configuration by Workspace

Use VS Code's workspace settings to have different configs per project:

**.vscode/settings.json** in Project A:
```json
{
  "logviewer.configPath": "${workspaceFolder}/.logviewer/config.yaml",
  "logviewer.autoRestartOnConfigChange": true
}
```

**.vscode/settings.json** in Project B:
```json
{
  "logviewer.configPath": "/shared/project-b-logs.yaml",
  "logviewer.autoRestartOnConfigChange": false
}
```

### Dynamic Config Reloading Workflow

For rapid config iteration:

1. Enable auto-restart:
   ```json
   {
     "logviewer.autoRestartOnConfigChange": true
   }
   ```

2. Open config file side-by-side with VS Code
3. Edit config, save
4. Server automatically restarts
5. Run new search to test changes

### Disable Watching for Performance

If you have many YAML files in `~/.logviewer/configs/`:

```json
{
  "logviewer.watchConfigFiles": false
}
```

Then manually restart after changes:
- Command Palette → "LogViewer: Restart Server"
- Or keyboard shortcut (configure in Keyboard Shortcuts)

---

## See Also

- [Extension README](./README.md) - General usage and features
- [Implementation Notes](./IMPLEMENTATION_NOTES.md) - Technical details
- Main LogViewer docs for configuration file format
