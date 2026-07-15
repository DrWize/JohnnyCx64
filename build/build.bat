@echo off
setlocal
echo ============================================
echo  Johnny Castaway 2026 - Windows x64 Build
echo ============================================
echo.

set "BUILD_DIR=%~dp0"
for %%I in ("%BUILD_DIR%..") do set "PROJECT_ROOT=%%~fI"
if not defined MSYS2_ROOT set "MSYS2_ROOT=C:\msys64"
set "PATH=C:\Program Files\Go\bin;%MSYS2_ROOT%\mingw64\bin;%PATH%"
set "CGO_ENABLED=1"
set "GOOS=windows"
set "GOARCH=amd64"
set "CC=%MSYS2_ROOT%\mingw64\bin\gcc.exe"
set "RESOURCE_OBJECT=resource_windows_amd64.syso"
set "OUTPUT=%BUILD_DIR%JohnnyCastaway-x64.exe"
set "HISTORY_DIR=%BUILD_DIR%history"
for /f %%I in ('powershell -NoProfile -Command "Get-Date -Format yyyyMMdd-HHmmss-fff"') do set "BUILD_TIMESTAMP=%%I"
set "HISTORY_OUTPUT=%HISTORY_DIR%\JohnnyCastaway-x64-%BUILD_TIMESTAMP%.exe"

pushd "%PROJECT_ROOT%"

echo [1/5] Verifying user-supplied resource archive when available...
go test -run TestEmbeddedArchiveHashesAndDecompression -count=1 .
if errorlevel 1 (
    popd
    echo RESOURCE VERIFICATION FAILED
    exit /b 1
)

echo [2/5] Building Windows resources...
if exist "%RESOURCE_OBJECT%" del /q "%RESOURCE_OBJECT%"
pushd "%BUILD_DIR%windows"
windres -i JohnnyCastaway.rc -O coff -o "%PROJECT_ROOT%\%RESOURCE_OBJECT%"
if errorlevel 1 (
    popd
    popd
    echo RESOURCE BUILD FAILED
    exit /b 1
)
popd

echo [3/5] Building 64-bit Windows executable...
go build -trimpath -ldflags "-H windowsgui -s -w" -o "%OUTPUT%" .
if errorlevel 1 (
    if exist "%RESOURCE_OBJECT%" del /q "%RESOURCE_OBJECT%"
    popd
    echo BUILD FAILED
    exit /b 1
)
if exist "%RESOURCE_OBJECT%" del /q "%RESOURCE_OBJECT%"

echo [4/5] Archiving timestamped executable...
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
echo   %OUTPUT% - latest 64-bit Windows desktop app
echo   %HISTORY_OUTPUT% - timestamped test build
echo.
echo Source-only build: Sierra data is not embedded.
echo Use --data-dir to select RESOURCE.MAP, RESOURCE.001, and optional sounds.
echo.
