@echo off
setlocal
echo ============================================
echo  Johnny Castaway 2026 - Windows x86 SCR Build
echo ============================================
echo.

set "BUILD_DIR=%~dp0"
for %%I in ("%BUILD_DIR%..") do set "PROJECT_ROOT=%%~fI"
if not defined MSYS2_ROOT set "MSYS2_ROOT=C:\msys64"
set "PATH=C:\Program Files\Go\bin;%MSYS2_ROOT%\mingw32\bin;%PATH%"
set "CGO_ENABLED=1"
set "GOOS=windows"
set "GOARCH=386"
set "CC=%MSYS2_ROOT%\mingw32\bin\gcc.exe"
set "RESOURCE_OBJECT=resource_windows_386.syso"
set "OUTPUT=%BUILD_DIR%JohnnyCastaway-x86.scr"
set "HISTORY_DIR=%BUILD_DIR%history"
for /f %%I in ('powershell -NoProfile -Command "Get-Date -Format yyyyMMdd-HHmmss-fff"') do set "BUILD_TIMESTAMP=%%I"
set "HISTORY_OUTPUT=%HISTORY_DIR%\JohnnyCastaway-x86-%BUILD_TIMESTAMP%.scr"

pushd "%PROJECT_ROOT%"

echo [1/5] Running x86 regression suite...
go test ./...
if errorlevel 1 (
    popd
    echo X86 TESTS FAILED
    exit /b 1
)

echo [2/5] Building x86 Windows resources...
if exist "%RESOURCE_OBJECT%" del /q "%RESOURCE_OBJECT%"
pushd "%BUILD_DIR%windows"
windres -i JohnnyCastaway-x86.rc -O coff -o "%PROJECT_ROOT%\%RESOURCE_OBJECT%"
if errorlevel 1 (
    popd
    popd
    echo X86 RESOURCE BUILD FAILED
    exit /b 1
)
popd

echo [3/5] Building 32-bit Windows screensaver...
go build -trimpath -ldflags "-H windowsgui -s -w" -o "%OUTPUT%" .
if errorlevel 1 (
    if exist "%RESOURCE_OBJECT%" del /q "%RESOURCE_OBJECT%"
    popd
    echo X86 BUILD FAILED
    exit /b 1
)
if exist "%RESOURCE_OBJECT%" del /q "%RESOURCE_OBJECT%"

echo [4/5] Archiving timestamped screensaver...
if not exist "%HISTORY_DIR%" mkdir "%HISTORY_DIR%"
copy /y "%OUTPUT%" "%HISTORY_OUTPUT%" >nul
if errorlevel 1 (
    popd
    echo TIMESTAMPED ARCHIVE FAILED
    exit /b 1
)
popd

echo [5/5] Done!
echo.
echo Output:
echo   %OUTPUT% - latest 32-bit Windows screensaver
echo   %HISTORY_OUTPUT% - timestamped test build
echo.
echo Supported Windows screensaver modes: /s, /p HWND, and /c.
echo Source-only build: Sierra data is not embedded.
echo.
