$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
try {
    & (Join-Path $PSScriptRoot 'node.exe') -config (Join-Path $root 'private\node.json')
} finally {
    # Any unsolicited foreground exit asks Task Scheduler for a bounded restart.
    # An explicit Stop-ScheduledTask stops this wrapper and its child process.
    exit 1
}
