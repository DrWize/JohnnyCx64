param(
    [string]$Artifact = (Join-Path $PSScriptRoot 'JohnnyCastaway.scr'),
    [string]$DataDirectory = ''
)

$ErrorActionPreference = 'Stop'
$screensaverPath = (Resolve-Path -LiteralPath $Artifact).Path
$projectRoot = Split-Path -Parent $PSScriptRoot
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

Add-Type -AssemblyName System.Windows.Forms
Add-Type @'
using System;
using System.Runtime.InteropServices;

public static class JohnnyScreensaverQA {
    public delegate bool EnumChildProc(IntPtr hwnd, IntPtr parameter);

    [DllImport("user32.dll")]
    public static extern bool EnumChildWindows(IntPtr parent, EnumChildProc callback, IntPtr parameter);

    [DllImport("user32.dll")]
    public static extern uint GetWindowThreadProcessId(IntPtr hwnd, out uint processId);
}
'@

function Start-JohnnyMode {
    param([string]$Arguments)

    $startInfo = New-Object System.Diagnostics.ProcessStartInfo
    $startInfo.FileName = $screensaverPath
    $startInfo.Arguments = $Arguments
    $startInfo.UseShellExecute = $false
    $process = [System.Diagnostics.Process]::Start($startInfo)
    $deadline = [DateTime]::UtcNow.AddSeconds(8)
    while ([DateTime]::UtcNow -lt $deadline -and !$process.HasExited) {
        [System.Windows.Forms.Application]::DoEvents()
        Start-Sleep -Milliseconds 100
        $process.Refresh()
        if ($process.MainWindowHandle -ne [IntPtr]::Zero) {
            return $process
        }
    }
    if ($process.HasExited) {
        throw "Mode '$Arguments' exited early with code $($process.ExitCode)"
    }
    return $process
}

function Stop-JohnnyProcess {
    param([System.Diagnostics.Process]$Process)

    if (!$Process.HasExited) {
        $Process.Kill()
        $Process.WaitForExit(5000) | Out-Null
    }
    $Process.Dispose()
}

if (![Environment]::Is64BitOperatingSystem -or ![Environment]::Is64BitProcess) {
    throw 'Native screensaver QA requires a 64-bit Windows host and PowerShell process'
}

$common = "--mute --data-dir `"$dataPath`""

$configuration = Start-JohnnyMode "/c $common"
Stop-JohnnyProcess $configuration
Write-Output 'PASS /c configuration window'

$fullScreenProcess = Start-JohnnyMode "/s $common"
Stop-JohnnyProcess $fullScreenProcess
Write-Output 'PASS /s full-screen mode'

$hostForm = New-Object System.Windows.Forms.Form
$hostForm.Text = 'Johnny Castaway preview QA host'
$hostForm.ClientSize = New-Object System.Drawing.Size(320, 240)
$hostForm.Show()
[System.Windows.Forms.Application]::DoEvents()

$previewInfo = New-Object System.Diagnostics.ProcessStartInfo
$previewInfo.FileName = $screensaverPath
$previewInfo.Arguments = "/p $($hostForm.Handle) $common"
$previewInfo.UseShellExecute = $false
$preview = [System.Diagnostics.Process]::Start($previewInfo)
try {
    $script:child = [IntPtr]::Zero
    $deadline = [DateTime]::UtcNow.AddSeconds(8)
    while ([DateTime]::UtcNow -lt $deadline -and $script:child -eq [IntPtr]::Zero -and !$preview.HasExited) {
        [System.Windows.Forms.Application]::DoEvents()
        [JohnnyScreensaverQA]::EnumChildWindows($hostForm.Handle, {
            param($hwnd, $parameter)
            [uint32]$processId = 0
            [JohnnyScreensaverQA]::GetWindowThreadProcessId($hwnd, [ref]$processId) | Out-Null
            if ($processId -eq $preview.Id) {
                $script:child = $hwnd
                return $false
            }
            return $true
        }, [IntPtr]::Zero) | Out-Null
        Start-Sleep -Milliseconds 100
        $preview.Refresh()
    }
    if ($preview.HasExited) {
        throw "Preview exited early with code $($preview.ExitCode)"
    }
    if ($script:child -eq [IntPtr]::Zero) {
        throw 'Preview did not create a child window in the supplied HWND'
    }
    Write-Output "PASS /p preview child HWND $script:child"

    $hostForm.Close()
    $hostForm.Dispose()
    [System.Windows.Forms.Application]::DoEvents()
    if (!$preview.WaitForExit(5000)) {
        throw 'Preview did not exit after its host window closed'
    }
    Write-Output 'PASS /p host-close shutdown'
} finally {
    Stop-JohnnyProcess $preview
    if (!$hostForm.IsDisposed) {
        $hostForm.Close()
    }
    $hostForm.Dispose()
}
