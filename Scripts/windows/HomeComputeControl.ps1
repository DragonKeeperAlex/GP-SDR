param([ValidateSet('pause','auto','status')][string]$Mode='status')
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName PresentationFramework
$marker = Join-Path $PSScriptRoot 'paused'
switch ($Mode) {
 'pause' { [IO.File]::WriteAllText($marker,'Paused by user'); $message='GP-SDR background AI is paused. Any running guarded request will be interrupted and retried later. It stays paused until you choose Automatic.' }
 'auto' { if (Test-Path $marker) { Remove-Item -LiteralPath $marker }; $message='Automatic mode enabled. GP-SDR AI runs after 10 idle minutes and pauses during desktop, fullscreen or game-controller use.' }
 'status' {
  $file=Join-Path $PSScriptRoot 'status.json'
  if (-not (Test-Path $file)) { $message='The GP-SDR AI guard is not running. Background work cannot reach the guarded endpoint.' }
  else {
   try {
    $s=Get-Content -Raw $file | ConvertFrom-Json
    if (((Get-Date).ToUniversalTime()-[DateTime]::Parse($s.updatedAt).ToUniversalTime()).TotalSeconds -gt 15) { $message='The guard status is stale. Background work stays queued if the guard is unavailable.' }
    elseif ($s.paused) { $message='Paused: '+$s.reason+'. Jobs remain saved for automatic retry.' }
    elseif ($s.active) { $message='Automatic mode: processing a background job while the PC is idle.' }
    else { $message='Automatic mode: PC idle and ready for background work.' }
   } catch { $message='Guard status unavailable. Background jobs are retained.' }
  }
 }
}
[System.Windows.MessageBox]::Show($message,'GP-SDR background AI') | Out-Null
