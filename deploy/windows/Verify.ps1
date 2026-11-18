param([Parameter(Mandatory=$true)][string]$InstallRoot, [Parameter(Mandatory=$true)][string]$PublicOrigin)
$ErrorActionPreference = 'Stop'
$config = Get-Content -LiteralPath (Join-Path $InstallRoot 'private\node.json') -Raw | ConvertFrom-Json
$uri = [Uri]$PublicOrigin
if ($uri.Scheme -ne 'https' -or $PublicOrigin -ne $uri.GetLeftPart([UriPartial]::Authority)) { throw 'Provide the exact HTTPS node origin.' }
Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
using Microsoft.Win32.SafeHandles;
public static class YukiDiskAllocation {
 [StructLayout(LayoutKind.Sequential)] public struct Standard {
  public long AllocationSize, EndOfFile; public uint NumberOfLinks;
  public byte DeletePending, Directory;
 }
 [DllImport("kernel32.dll",SetLastError=true)] public static extern bool GetFileInformationByHandleEx(SafeFileHandle h, int kind, out Standard value, uint size);
}
'@
$container = Join-Path $config.data $config.container_file
$attributes = (Get-Item -LiteralPath $container -Force).Attributes
if ($attributes -band ([IO.FileAttributes]::SparseFile -bor [IO.FileAttributes]::Compressed -bor [IO.FileAttributes]::ReparsePoint)) { throw 'Container attributes violate the storage contract.' }
$stream = [IO.File]::Open($container,[IO.FileMode]::Open,[IO.FileAccess]::Read,([IO.FileShare]::ReadWrite -bor [IO.FileShare]::Delete))
try {
    $standard = New-Object YukiDiskAllocation+Standard
    if (-not [YukiDiskAllocation]::GetFileInformationByHandleEx($stream.SafeFileHandle,1,[ref]$standard,[Runtime.InteropServices.Marshal]::SizeOf($standard))) { throw 'Cannot inspect real allocation.' }
    if ($standard.AllocationSize -lt $config.capacity_bytes -or $standard.EndOfFile -ne $config.capacity_bytes -or $standard.NumberOfLinks -ne 1 -or $standard.Directory -ne 0) { throw 'Container is not fully allocated with the expected single-link size.' }
    Write-Output ('Allocated bytes: '+$standard.AllocationSize+'; configured bytes: '+$config.capacity_bytes)
} finally { $stream.Dispose() }
try {
    $response = Invoke-WebRequest -UseBasicParsing -Uri ($PublicOrigin+'/statusz') -Headers @{'X-Yuki-Monitor'=$config.monitor_key} -TimeoutSec 10 -MaximumRedirection 0
    $status = $response.Content | ConvertFrom-Json
    if ($status.service -ne 'storage' -or $status.ready -ne $true -or $status.storage.configured_bytes -ne $config.capacity_bytes) { throw 'Not ready.' }
} catch { throw 'Authenticated HTTPS readiness failed. Check the task, TLS, tunnel and control reconciliation without printing credentials.' }
Write-Output 'PASS actual allocation and authenticated HTTPS readiness. This is not a full replica/upload/download acceptance test.'
