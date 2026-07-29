# Testing Configuration File Watching

This document provides manual test cases for verifying the configuration file watching feature.

## Prerequisites

1. Install the extension (or run in Extension Development Host via F5)
2. Have a LogViewer configuration file at `~/.logviewer/config.yaml`
3. Ensure the server can start successfully

---

## Test Suite

### Test 1: Default Config Path - Manual Restart

**Setup**:
```json
{
  "logviewer.configPath": "",
  "logviewer.watchConfigFiles": true,
  "logviewer.autoRestartOnConfigChange": false
}
```

**Steps**:
1. Open LogViewer sidebar
2. Run a search (starts the server)
3. Open `~/.logviewer/config.yaml` in another editor
4. Make a change (add comment or modify context)
5. Save the file

**Expected Result**:
- Notification appears: "LogViewer configuration file changed. Restart server to apply changes?"
- Two buttons shown: "Restart Now" and "Later"
- LogViewer Server output shows: "Config file changed: ~/.logviewer/config.yaml"

**Verify**:
- [ ] Notification appears within 1-2 seconds of save
- [ ] Clicking "Restart Now" restarts server
- [ ] Clicking "Later" dismisses notification
- [ ] Output channel logs the change

---

### Test 2: Default Config Path - Auto Restart

**Setup**:
```json
{
  "logviewer.configPath": "",
  "logviewer.watchConfigFiles": true,
  "logviewer.autoRestartOnConfigChange": true
}
```

**Steps**:
1. Open LogViewer sidebar
2. Run a search (starts the server)
3. Open `~/.logviewer/config.yaml`
4. Make a change
5. Save the file

**Expected Result**:
- No notification shown
- Server automatically restarts
- LogViewer Server output shows: "Auto-restarting server due to config change..."
- Output shows: "Server restarted successfully"

**Verify**:
- [ ] No user prompt appears
- [ ] Server restarts within 1-2 seconds
- [ ] New search uses updated config

---

### Test 3: Custom Config Path

**Setup**:
```json
{
  "logviewer.configPath": "/tmp/test-logviewer.yaml",
  "logviewer.watchConfigFiles": true,
  "logviewer.autoRestartOnConfigChange": false
}
```

**Steps**:
1. Create `/tmp/test-logviewer.yaml` with valid config
2. Open LogViewer sidebar
3. Run a search
4. Modify `/tmp/test-logviewer.yaml`
5. Save the file

**Expected Result**:
- Notification appears
- Correctly identifies custom config path in output
- Does NOT trigger on changes to `~/.logviewer/config.yaml`

**Verify**:
- [ ] Custom path is watched
- [ ] Default path is NOT watched
- [ ] Notification references correct file path

---

### Test 4: Multiple Config Files (Drop-in Directory)

**Setup**:
```json
{
  "logviewer.configPath": "",
  "logviewer.watchConfigFiles": true,
  "logviewer.autoRestartOnConfigChange": false
}
```

**Steps**:
1. Ensure `~/.logviewer/configs/` directory exists
2. Create `~/.logviewer/configs/custom.yaml` with additional contexts
3. Start server
4. Modify `~/.logviewer/configs/custom.yaml`
5. Save

**Expected Result**:
- Notification appears
- Output shows: "Config file changed: ~/.logviewer/configs/custom.yaml"

**Verify**:
- [ ] Drop-in configs are watched
- [ ] Changes detected correctly
- [ ] Server uses merged config on restart

---

### Test 5: Config File Deleted

**Setup**:
```json
{
  "logviewer.configPath": "/tmp/test-logviewer.yaml",
  "logviewer.watchConfigFiles": true,
  "logviewer.autoRestartOnConfigChange": false
}
```

**Steps**:
1. Create config file
2. Start server
3. Delete the config file
4. Observe

**Expected Result**:
- Warning notification: "LogViewer config file deleted: test-logviewer.yaml. Server may not work correctly."
- Server continues running (uses cached config)
- Next restart attempt will fail

**Verify**:
- [ ] Warning notification appears
- [ ] Server doesn't crash immediately
- [ ] Appropriate warning message shown

---

### Test 6: Watch Disabled

**Setup**:
```json
{
  "logviewer.configPath": "",
  "logviewer.watchConfigFiles": false,
  "logviewer.autoRestartOnConfigChange": false
}
```

**Steps**:
1. Start server
2. Modify `~/.logviewer/config.yaml`
3. Save
4. Wait 5 seconds

**Expected Result**:
- No notification appears
- No output in LogViewer Server channel
- Config changes not detected

**Verify**:
- [ ] File changes ignored
- [ ] No notifications
- [ ] Manual restart still works via Command Palette

---

### Test 7: Config Path Changed via Settings

**Setup**:
```json
{
  "logviewer.configPath": "",
  "logviewer.watchConfigFiles": true
}
```

**Steps**:
1. Start server
2. Open VS Code Settings
3. Change `logviewer.configPath` to `/tmp/new-config.yaml`
4. Check LogViewer Server output

**Expected Result**:
- Output shows: "LogViewer settings changed, recreating config watcher..."
- File watcher now watches `/tmp/new-config.yaml` instead of default

**Verify**:
- [ ] Watcher recreated on setting change
- [ ] New path is watched
- [ ] Old path is no longer watched

---

### Test 8: YAML Syntax Error

**Setup**:
```json
{
  "logviewer.configPath": "",
  "logviewer.watchConfigFiles": true,
  "logviewer.autoRestartOnConfigChange": true
}
```

**Steps**:
1. Start server
2. Modify config with syntax error (e.g., invalid indentation)
3. Save
4. Observe

**Expected Result**:
- Server attempts restart
- Restart fails
- Error notification: "Failed to restart LogViewer server: [error message]"
- Output shows config parse error

**Verify**:
- [ ] Restart attempted
- [ ] Parse error caught and logged
- [ ] User notified of failure
- [ ] Server state remains stopped

---

### Test 9: Rapid Successive Changes

**Setup**:
```json
{
  "logviewer.configPath": "",
  "logviewer.watchConfigFiles": true,
  "logviewer.autoRestartOnConfigChange": true
}
```

**Steps**:
1. Start server
2. Rapidly edit and save config file 3 times within 5 seconds
3. Observe behavior

**Expected Result**:
- Only one restart triggered (debounced)
- `isRestarting` flag prevents concurrent restarts
- Output shows single restart sequence

**Verify**:
- [ ] No multiple simultaneous restarts
- [ ] Server ends up in stable state
- [ ] No race conditions observed

---

### Test 10: Server Not Running

**Setup**:
```json
{
  "logviewer.configPath": "",
  "logviewer.watchConfigFiles": true,
  "logviewer.autoRestartOnConfigChange": false
}
```

**Steps**:
1. Do NOT start server (don't run any search)
2. Modify config file
3. Save

**Expected Result**:
- No notification (server not running)
- File change logged in output: "Config file changed: ..."
- Next server start will use new config

**Verify**:
- [ ] No unnecessary notifications
- [ ] Change logged for awareness
- [ ] Config picked up on next start

---

## Edge Cases

### Edge Case 1: Permission Denied

**Scenario**: Config file becomes unreadable

**Steps**:
1. Start server
2. `chmod 000 ~/.logviewer/config.yaml`
3. Trigger restart

**Expected**: Error notification about permission denied

---

### Edge Case 2: Symlinked Config

**Scenario**: Config path is a symlink

**Steps**:
1. `ln -s /path/to/real-config.yaml ~/.logviewer/config.yaml`
2. Start server
3. Modify `/path/to/real-config.yaml`

**Expected**: Change detected (FileSystemWatcher follows symlinks)

---

### Edge Case 3: Network Drive Config

**Scenario**: Config on network/SMB share

**Steps**:
1. Set `logviewer.configPath` to network path
2. Start server
3. Modify config from another machine

**Expected**: Change detection may be delayed or unreliable (OS-dependent)

---

## Debugging

If tests fail, check these:

### View Output Channel
1. Command Palette → "LogViewer: Show Server Output"
2. Look for:
   - "Setting up config watcher..."
   - "Config file changed: ..."
   - "Auto-restarting..." or notification messages
   - Any error messages

### Check File Watcher
```typescript
// In extension, log watcher setup:
console.log('Watching pattern:', watchPattern);
console.log('Config paths:', this.getConfigPaths());
```

### Verify File System Events
- Use `fswatch` or `inotifywait` to confirm OS-level events
- macOS: `fswatch ~/.logviewer/config.yaml`
- Linux: `inotifywait -m ~/.logviewer/config.yaml`

### Check Settings
- Verify settings in User vs Workspace scope
- Check for conflicting settings
- Verify `LOGVIEWER_CONFIG` environment variable

---

## Performance Testing

### Test: Many Config Files

**Setup**: 50+ YAML files in `~/.logviewer/configs/`

**Measure**:
- Extension activation time
- File change detection latency
- Memory usage

**Expected**: No significant performance degradation

---

### Test: Large Config File

**Setup**: Config file with 100+ contexts

**Measure**:
- File change detection time
- Server restart time
- UI responsiveness during restart

**Expected**: Restart completes within 5 seconds

---

## Automated Testing (Future)

Consider adding:
1. Unit tests for `getConfigPaths()`
2. Integration tests with mock FileSystemWatcher
3. E2E tests with temporary config files

Example structure:
```typescript
describe('ServerManager', () => {
  describe('getConfigPaths', () => {
    it('returns explicit path when set', () => {});
    it('returns default paths when empty', () => {});
    it('returns env var paths when set', () => {});
  });
});
```
