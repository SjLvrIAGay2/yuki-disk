param(
    [Parameter(Mandatory=$true)][string]$PrivateConfig,
    [Parameter(Mandatory=$true)][string]$Certificate,
    [Parameter(Mandatory=$true)][string]$PrivateKey,
    [string]$ServiceAccount = 'NT AUTHORITY\LOCAL SERVICE',
    [Parameter(Mandatory=$true)][string]$BinarySHA256,
    [Parameter(Mandatory=$true)][string]$InstallRoot,
    [Parameter(Mandatory=$true)][long]$CapacityBytes
)
$ErrorActionPreference = 'Stop'
if (-not ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) { throw 'Run in an elevated PowerShell window.' }
$root = [IO.Path]::GetFullPath($InstallRoot).TrimEnd('\')
if ($root -notmatch '^[A-Za-z]:\\.+\\[^\\]+$' -or $root.Contains('"') -or (Test-Path -LiteralPath $root)) { throw 'Choose a new absolute installation directory on a local drive.' }
$parent = Split-Path -Parent $root
if (-not (Test-Path -LiteralPath $parent -PathType Container)) { throw 'The parent directory must already exist.' }
$ancestor = $parent
while ($ancestor) {
    if ((Get-Item -LiteralPath $ancestor -Force).Attributes -band [IO.FileAttributes]::ReparsePoint) { throw 'Reparse ancestors are forbidden.' }
    $ancestor = Split-Path -Parent $ancestor
}
$volume = Get-Volume -DriveLetter $root.Substring(0,1)
if ($volume.FileSystem -ne 'NTFS' -or $CapacityBytes -lt 16777216 -or $CapacityBytes -gt 1099511627776 -or $volume.SizeRemaining -lt ($CapacityBytes + 2GB)) { throw 'A local NTFS volume with capacity plus 2 GiB headroom is required.' }
$account = New-Object Security.Principal.NTAccount($ServiceAccount)
$sid = $account.Translate([Security.Principal.SecurityIdentifier])
if ($sid.Value -in @('S-1-5-18','S-1-5-32-544')) { throw 'Use a dedicated ordinary service account.' }
foreach ($source in @($PrivateConfig,$Certificate,$PrivateKey)) {
    $item = Get-Item -LiteralPath $source -Force
    if ($item.PSIsContainer -or ($item.Attributes -band [IO.FileAttributes]::ReparsePoint)) { throw 'Private inputs must be regular files.' }
}
$config = Get-Content -LiteralPath $PrivateConfig -Raw | ConvertFrom-Json
if ($config.development -ne $false -or $config.storage_mode -ne 'container' -or $config.capacity_bytes -ne $CapacityBytes -or $config.listen -ne '127.0.0.1:18444') { throw 'Expected production container configuration with the exact capacity and loopback listener.' }
$keys = @($config.key,$config.capability_key,$config.maintenance_key,$config.monitor_key)
if (($keys | Select-Object -Unique).Count -ne 4) { throw 'Credentials must be independent.' }
foreach ($key in $keys) { if ($key -isnot [string] -or $key.Length -lt 32 -or $key -match 'development|REPLACE_|SET_') { throw 'Provide real private credentials outside the package.' } }
if ($BinarySHA256 -notmatch '^[a-fA-F0-9]{64}$' -or (Get-FileHash -LiteralPath (Join-Path $PSScriptRoot 'node.exe') -Algorithm SHA256).Hash -ne $BinarySHA256) { throw 'Binary digest mismatch.' }
# No existing installation, metadata or storage is overwritten or reformatted.
New-Item -ItemType Directory -Path $root | Out-Null
$acl = New-Object Security.AccessControl.DirectorySecurity
$acl.SetAccessRuleProtection($true,$false)
foreach ($adminSid in @('S-1-5-18','S-1-5-32-544')) {
    $identity = New-Object Security.Principal.SecurityIdentifier($adminSid)
    $acl.AddAccessRule((New-Object Security.AccessControl.FileSystemAccessRule($identity,'FullControl','ContainerInherit,ObjectInherit','None','Allow')))
}
$acl.AddAccessRule((New-Object Security.AccessControl.FileSystemAccessRule($sid,'ReadAndExecute','ContainerInherit,ObjectInherit','None','Allow')))
Set-Acl -LiteralPath $root -AclObject $acl
foreach ($directory in @('bin','private','data')) { New-Item -ItemType Directory -Path (Join-Path $root $directory) | Out-Null }
$data = Join-Path $root 'data'
$dataAcl = Get-Acl -LiteralPath $data
$dataAcl.AddAccessRule((New-Object Security.AccessControl.FileSystemAccessRule($sid,'Modify','ContainerInherit,ObjectInherit','None','Allow')))
Set-Acl -LiteralPath $data -AclObject $dataAcl
foreach ($name in @('node.exe','Run-Node.ps1','Run-Tunnel.ps1')) { Copy-Item -LiteralPath (Join-Path $PSScriptRoot $name) -Destination (Join-Path $root ('bin\'+$name)) }
Copy-Item -LiteralPath $Certificate -Destination (Join-Path $root 'private\node-cert.pem')
Copy-Item -LiteralPath $PrivateKey -Destination (Join-Path $root 'private\node-key.pem')
$config.data = $data
$config.tls_cert = Join-Path $root 'private\node-cert.pem'
$config.tls_key = Join-Path $root 'private\node-key.pem'
$config.container_file = 'blocks.pack'
[IO.File]::WriteAllText((Join-Path $root 'private\node.json'),($config | ConvertTo-Json -Depth 20),(New-Object Text.UTF8Encoding($false)))
Write-Output 'Installed private configuration and binary. No node, task, tunnel or container has been started.'
Write-Output 'Review the configuration, then register the task with the dedicated account. The first start performs actual allocation.'
