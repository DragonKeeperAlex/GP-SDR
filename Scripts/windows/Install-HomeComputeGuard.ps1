# Run from an elevated PowerShell session after copying the guard exe beside this file.
# The runtime task uses the logged-in user's session at limited privilege.
param(
 [Parameter(Mandatory=$true)][string]$PiAddress,
 [string]$InstallDirectory = 'D:\AI\GP-SDR-Deferred\Guard',
 [string]$User = "$env:COMPUTERNAME\$env:USERNAME"
)
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
$parsed = $null
if (-not [Net.IPAddress]::TryParse($PiAddress, [ref]$parsed)) { throw 'PiAddress must be a private LAN IP address.' }
$b = $parsed.GetAddressBytes()
if ($b.Length -ne 4 -or -not ($b[0] -eq 10 -or ($b[0] -eq 192 -and $b[1] -eq 168) -or ($b[0] -eq 172 -and $b[1] -ge 16 -and $b[1] -le 31))) { throw 'Only RFC1918 IPv4 addresses are supported.' }
New-Item -ItemType Directory -Force -Path $InstallDirectory | Out-Null
$InstallDirectory = (Get-Item -LiteralPath $InstallDirectory).FullName
$binary = Join-Path $InstallDirectory 'gp-sdr-home-guard.exe'
$source = Join-Path $PSScriptRoot 'gp-sdr-home-guard.exe'
if (-not (Test-Path $source)) { throw 'Build/copy gp-sdr-home-guard.exe beside the installer first.' }
Stop-ScheduledTask -TaskName 'GP-SDR-Home-Compute-Guard' -ErrorAction SilentlyContinue
Get-Process -Name gp-sdr-home-guard -ErrorAction SilentlyContinue | Where-Object { $_.Path -and [IO.Path]::GetFullPath($_.Path) -eq [IO.Path]::GetFullPath($binary) } | Stop-Process -Force
Start-Sleep -Seconds 1
if ([IO.Path]::GetFullPath($source) -ne [IO.Path]::GetFullPath($binary)) { Copy-Item -Force $source $binary }
Copy-Item -Force (Join-Path $PSScriptRoot 'HomeComputeControl.ps1') (Join-Path $InstallDirectory 'HomeComputeControl.ps1')
$action = New-ScheduledTaskAction -Execute $binary -Argument ('-idle 10m -listen 0.0.0.0:11435 -state-dir "' + $InstallDirectory + '"') -WorkingDirectory $InstallDirectory
$principal = New-ScheduledTaskPrincipal -UserId $User -LogonType Interactive -RunLevel Limited
$trigger = New-ScheduledTaskTrigger -AtLogOn -User $User
$settings = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 3 -RestartInterval (New-TimeSpan -Minutes 1) -MultipleInstances IgnoreNew -StartWhenAvailable -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
Register-ScheduledTask -TaskName 'GP-SDR-Home-Compute-Guard' -Action $action -Principal $principal -Trigger $trigger -Settings $settings -Description 'Pause GP-SDR background AI during desktop, fullscreen or controller use; resume after 10 idle minutes.' -Force | Out-Null
$rule = Get-NetFirewallRule -Name 'GP-SDR-Home-Compute-Guard' -ErrorAction SilentlyContinue
if ($rule) { $rule | Remove-NetFirewallRule }
New-NetFirewallRule -Name 'GP-SDR-Home-Compute-Guard' -DisplayName 'GP-SDR home compute from Pi only' -Direction Inbound -Action Allow -Protocol TCP -LocalPort 11435 -RemoteAddress $PiAddress -Profile Any -Program $binary | Out-Null
# Windows can create a broad "Query User" block after a first-run prompt.
# Only disable auto-generated blocks for this exact executable; the narrow
# source-IP allow rule above remains the sole inbound permission we add.
Get-NetFirewallApplicationFilter | Where-Object { $_.Program -ieq $binary } | Get-NetFirewallRule | Where-Object { $_.Name -like '*Query User*' -and $_.Action -eq 'Block' } | Disable-NetFirewallRule | Out-Null
$desktop = [Environment]::GetFolderPath('Desktop')
$shell = New-Object -ComObject WScript.Shell
foreach ($item in @(@('Pause GP-SDR AI','pause'),@('GP-SDR AI - Automatic','auto'),@('GP-SDR AI Status','status'))) {
 $shortcut = $shell.CreateShortcut((Join-Path $desktop ($item[0]+'.lnk')))
 $shortcut.TargetPath = "$env:SystemRoot\System32\WindowsPowerShell\v1.0\powershell.exe"
 $shortcut.Arguments = '-NoProfile -WindowStyle Hidden -ExecutionPolicy Bypass -File "' + (Join-Path $InstallDirectory 'HomeComputeControl.ps1') + '" -Mode ' + $item[1]
 $shortcut.WorkingDirectory = $InstallDirectory
 $shortcut.Description = 'Control GP-SDR background AI; your desktop and games always take priority.'
 $shortcut.Save()
}
Start-ScheduledTask -TaskName 'GP-SDR-Home-Compute-Guard'
Write-Output 'Guard installed. Pause, Automatic and Status shortcuts added to the user Desktop. No password stored.'
