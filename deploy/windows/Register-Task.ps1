param(
    [Parameter(Mandatory=$true)][string]$InstallRoot,
    [string]$TaskName = 'yukiDisk-Node',
    [switch]$Tunnel
)
$ErrorActionPreference = 'Stop'
$root = [IO.Path]::GetFullPath($InstallRoot).TrimEnd('\')
if ($root.Contains('"') -or $TaskName -notmatch '^yukiDisk-[A-Za-z0-9_-]+$') { throw 'Invalid installation or task name.' }
if (Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue) { throw 'Task already exists; inspect it instead of overwriting.' }
foreach ($name in @('bin\node.exe','bin\Run-Node.ps1','private\node.json')) { if (-not (Test-Path -LiteralPath (Join-Path $root $name) -PathType Leaf)) { throw 'Incomplete installation.' } }
$powershell = Join-Path $env:SystemRoot 'System32\WindowsPowerShell\v1.0\powershell.exe'
$scriptName = 'Run-Node.ps1'
if ($Tunnel) {
    $scriptName = 'Run-Tunnel.ps1'
    foreach ($name in @('tunnel.json','tunnel-key','known_hosts')) { if (-not (Test-Path -LiteralPath (Join-Path $root ('private\'+$name)) -PathType Leaf)) { throw 'Install the dedicated key, pinned host key and fixed tunnel configuration first.' } }
    if (-not (Test-Path -LiteralPath (Join-Path $env:SystemRoot 'System32\OpenSSH\ssh.exe'))) { throw 'Windows OpenSSH Client is required.' }
}
$action = New-ScheduledTaskAction -Execute $powershell -Argument ('-NoProfile -NonInteractive -ExecutionPolicy RemoteSigned -File "' + (Join-Path $root ('bin\'+$scriptName)) + '"') -WorkingDirectory $root
$settings = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1) -MultipleInstances IgnoreNew -StartWhenAvailable -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
$principal = New-ScheduledTaskPrincipal -UserId 'S-1-5-19' -LogonType ServiceAccount -RunLevel Limited
Register-ScheduledTask -TaskName $TaskName -Action $action -Trigger (New-ScheduledTaskTrigger -AtStartup) -Settings $settings -Principal $principal | Out-Null
Write-Output 'Registered without starting. Use Start-ScheduledTask after TLS, control and transport are ready.'
