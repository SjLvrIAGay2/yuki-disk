$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$private = Join-Path $root 'private'
$config = Get-Content -LiteralPath (Join-Path $private 'tunnel.json') -Raw | ConvertFrom-Json
if (@($config.PSObject.Properties.Name | Where-Object { $_ -notin @('server','user','remotePort','localPort') }).Count -or $config.server -notmatch '^[A-Za-z0-9.-]+$' -or $config.user -notmatch '^[A-Za-z0-9_-]+$' -or $config.remotePort -lt 1024 -or $config.remotePort -gt 65535 -or $config.localPort -ne 18444) { throw 'Invalid fixed tunnel configuration.' }
$ssh = Join-Path $env:SystemRoot 'System32\OpenSSH\ssh.exe'
# Keep network reconnects inside the running task. Task Scheduler recovery alone
# did not restart an already-running task after a remote host outage.
while ($true) {
    try {
        & $ssh -N -T -o BatchMode=yes -o ConnectTimeout=10 -o ExitOnForwardFailure=yes -o ServerAliveInterval=15 -o ServerAliveCountMax=3 -o StrictHostKeyChecking=yes -o ('UserKnownHostsFile='+(Join-Path $private 'known_hosts')) -o IdentitiesOnly=yes -i (Join-Path $private 'tunnel-key') -R ('127.0.0.1:'+ $config.remotePort + ':127.0.0.1:' + $config.localPort) ($config.user+'@'+$config.server) 2>$null
    } catch {
        # Native stderr can be a terminating error in Windows PowerShell.
        # Do not publish raw SSH diagnostics or relax pinned host validation.
    }
    Start-Sleep -Seconds 15
}
