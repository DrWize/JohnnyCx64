param(
    [int]$SceneAdvances = 60,
    [int]$StorySeconds = 25,
    [ValidateSet('', 'off', 'lightweight', 'fast', 'lottes')]
    [string]$CrtMode = '',
    [string]$DataDirectory = ''
)

$ErrorActionPreference = 'Stop'

Add-Type @'
using System;
using System.Runtime.InteropServices;
public static class JohnnySweepWindow {
    [DllImport("user32.dll")]
    public static extern bool PostMessage(IntPtr window, uint message, IntPtr key, IntPtr data);
    [DllImport("user32.dll")]
    public static extern uint MapVirtualKey(uint code, uint mapType);
}
'@

function Send-Key([Diagnostics.Process]$Process, [int]$Key) {
    $window = [IntPtr]::Zero
    for ($attempt = 0; $attempt -lt 50; $attempt++) {
        $Process.Refresh()
        if ($Process.HasExited) {
            throw "Johnny Castaway exited with code $($Process.ExitCode)"
        }
        $window = $Process.MainWindowHandle
        if ($window -ne [IntPtr]::Zero) {
            break
        }
        Start-Sleep -Milliseconds 100
    }
    if ($window -eq [IntPtr]::Zero) {
        throw 'Johnny Castaway did not recreate its window within 5 seconds'
    }

    $scanCode = [JohnnySweepWindow]::MapVirtualKey($Key, 0)
    $keyDownData = [IntPtr](1 -bor ($scanCode -shl 16))
    $keyUpData = [IntPtr]([long](1 -bor ($scanCode -shl 16)) -bor 0xC0000000L)
    [JohnnySweepWindow]::PostMessage($window, 0x100, [IntPtr]$Key, $keyDownData) | Out-Null
    Start-Sleep -Milliseconds 50
    [JohnnySweepWindow]::PostMessage($window, 0x101, [IntPtr]$Key, $keyUpData) | Out-Null
}

function Get-TtmNames([string]$MapPath, [string]$ResourcePath) {
    $map = [IO.File]::ReadAllBytes($MapPath)
    $resources = [IO.File]::ReadAllBytes($ResourcePath)
    $entryCount = [BitConverter]::ToUInt16($map, 19)

    $names = for ($index = 0; $index -lt $entryCount; $index++) {
        $offset = [BitConverter]::ToUInt32($map, 25 + 8 * $index)
        if ($offset + 13 -gt $resources.Length) {
            continue
        }

        $header = [Text.Encoding]::ASCII.GetString($resources, $offset, 13)
        $dot = $header.IndexOf('.')
        if ($dot -lt 0) {
            continue
        }

        $end = [Math]::Min($dot + 4, 13)
        $name = $header.Substring(0, $end).TrimEnd([char]0)
        if ($name.Substring($dot) -eq '.TTM') {
            $name
        }
    }

    return @($names | Sort-Object -Unique)
}

function Read-NewLog([string]$Path, [long]$Offset) {
    $stream = [IO.File]::Open($Path, 'Open', 'Read', 'ReadWrite')
    try {
        $null = $stream.Seek($Offset, 'Begin')
        $reader = [IO.StreamReader]::new($stream)
        try { return $reader.ReadToEnd() } finally { $reader.Dispose() }
    } finally {
        $stream.Dispose()
    }
}

$projectRoot = Split-Path -Parent $PSScriptRoot
$exe = Join-Path $PSScriptRoot 'JohnnyCastaway.exe'
if (!$DataDirectory) {
    $DataDirectory = @(
        (Join-Path $PSScriptRoot 'scrantic'),
        (Join-Path $projectRoot 'scrantic'),
        (Join-Path $projectRoot '..\scrantic')
    ) | Where-Object {
        (Test-Path -LiteralPath (Join-Path $_ 'RESOURCE.MAP') -PathType Leaf) -and
        (Test-Path -LiteralPath (Join-Path $_ 'RESOURCE.001') -PathType Leaf)
    } | Select-Object -First 1
}
if (!$DataDirectory) {
    throw 'No scrantic data folder was found; pass -DataDirectory explicitly'
}
$dataPath = (Resolve-Path -LiteralPath $DataDirectory).Path
$map = Join-Path $dataPath 'RESOURCE.MAP'
$resources = Join-Path $dataPath 'RESOURCE.001'
$log = Join-Path $env:LOCALAPPDATA 'JohnnyCastaway\JohnnyCastaway.log'
$ttms = Get-TtmNames $map $resources

$existing = @(Get-CimInstance Win32_Process | Where-Object {
    $_.Name -eq 'JohnnyCastaway.exe' -and $_.ExecutablePath -eq $exe
})
if ($existing.Count -ne 0) {
    $ids = ($existing.ProcessId -join ', ')
    throw "Close the existing Johnny Castaway test process(es) before running the sweep: $ids"
}

if ($ttms.Count -ne 41) {
    throw "Expected 41 indexed TTMs, found $($ttms.Count)"
}

$logOffset = (Get-Item $log -ErrorAction SilentlyContinue).Length
$ttmArguments = @('--windowed', '--mute', '--no-save-settings', '--data-dir', $dataPath, '--ttm', $ttms[0])
if ($CrtMode) { $ttmArguments += @('--crt', $CrtMode) }
$process = Start-Process -FilePath $exe -ArgumentList $ttmArguments -WindowStyle Hidden -PassThru
$completed = 0
$failure = ''

try {
    Start-Sleep -Seconds 4
    foreach ($ttm in $ttms) {
        for ($scene = 0; $scene -lt $SceneAdvances; $scene++) {
            Send-Key $process 0x54 # T
            Start-Sleep -Milliseconds 70
        }
        Write-Output "PASS $ttm ($SceneAdvances scene advances)"
        $completed++
        Send-Key $process 0x4E # N
        Start-Sleep -Milliseconds 1600
    }

    Send-Key $process 0x70 # F1 settings
    Start-Sleep -Milliseconds 500
    Send-Key $process 0x1B # Escape
    Send-Key $process 0x74 # F5 runtime log
    Start-Sleep -Milliseconds 500
    Send-Key $process 0x1B # Escape
} catch {
    $failure = $_.Exception.Message
} finally {
    $process.Refresh()
    if (!$process.HasExited) {
        Stop-Process -Id $process.Id
        $process.WaitForExit()
    }
}

$ttmLog = Read-NewLog $log $logOffset
$fatal = $ttmLog -match 'fatal error'
$advanceMatches = [regex]::Matches($ttmLog, 'advanced ([A-Z0-9_]+\.TTM) to scene (\d+)')
$wrappedTtms = @($advanceMatches | Group-Object { $_.Groups[1].Value } | Where-Object {
    $uniqueScenes = @($_.Group | ForEach-Object { $_.Groups[2].Value } | Sort-Object -Unique)
    $_.Count -gt $uniqueScenes.Count
}).Count

$storyOffset = (Get-Item $log).Length
$storyArguments = @('--windowed', '--mute', '--no-save-settings', '--data-dir', $dataPath)
if ($CrtMode) { $storyArguments += @('--crt', $CrtMode) }
$story = Start-Process -FilePath $exe -ArgumentList $storyArguments -WindowStyle Hidden -PassThru
Start-Sleep -Seconds $StorySeconds
$story.Refresh()
$storyAlive = !$story.HasExited
if ($storyAlive) {
    Stop-Process -Id $story.Id
    $story.WaitForExit()
}
$storyLog = Read-NewLog $log $storyOffset

[pscustomobject]@{
    IndexedTTMs = $ttms.Count
    CompletedTTMs = $completed
    SceneAdvances = $advanceMatches.Count
    WrappedTTMs = $wrappedTtms
    ContentSwitches = ([regex]::Matches($ttmLog, 'switching content').Count)
    TtmFatal = $fatal
    TtmFailure = $failure
    StoryAliveAfterSeconds = $storyAlive
    StoryFatal = $storyLog -match 'fatal error'
} | Format-List

if ($completed -ne $ttms.Count -or $wrappedTtms -ne $ttms.Count -or $fatal -or $failure -or !$storyAlive -or $storyLog -match 'fatal error') {
    exit 1
}
