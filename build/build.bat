@echo off
setlocal
echo ============================================
echo  Johnny Castaway 2026 - Windows 11 x64 Build
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
set "APP_OUTPUT=%BUILD_DIR%JohnnyCastaway.exe"
set "SCR_OUTPUT=%BUILD_DIR%JohnnyCastaway.scr"
set "HISTORY_DIR=%BUILD_DIR%history"
for /f %%I in ('powershell -NoProfile -Command "Get-Date -Format yyyyMMdd-HHmmss-fff"') do set "BUILD_TIMESTAMP=%%I"
set "APP_HISTORY=%HISTORY_DIR%\JohnnyCastaway-%BUILD_TIMESTAMP%.exe"
set "SCR_HISTORY=%HISTORY_DIR%\JohnnyCastaway-%BUILD_TIMESTAMP%.scr"

pushd "%PROJECT_ROOT%"

echo [1/8] Running native x64 regression suite...
go test ./...
if errorlevel 1 (
    popd
    echo X64 TESTS FAILED
    exit /b 1
)

echo [2/8] Building x64 application resources...
if exist "%RESOURCE_OBJECT%" del /q "%RESOURCE_OBJECT%"
pushd "%BUILD_DIR%windows"
windres -i JohnnyCastaway.rc -O coff -o "%PROJECT_ROOT%\%RESOURCE_OBJECT%"
if errorlevel 1 (
    popd
    popd
    echo APPLICATION RESOURCE BUILD FAILED
    exit /b 1
)
popd

echo [3/8] Building native x64 application...
go build -trimpath -ldflags "-H windowsgui -s -w" -o "%APP_OUTPUT%" .
if errorlevel 1 (
    if exist "%RESOURCE_OBJECT%" del /q "%RESOURCE_OBJECT%"
    popd
    echo APPLICATION BUILD FAILED
    exit /b 1
)
if exist "%RESOURCE_OBJECT%" del /q "%RESOURCE_OBJECT%"

echo [4/8] Building x64 screensaver resources...
pushd "%BUILD_DIR%windows"
windres -i JohnnyCastaway-screensaver.rc -O coff -o "%PROJECT_ROOT%\%RESOURCE_OBJECT%"
if errorlevel 1 (
    popd
    popd
    echo SCREENSAVER RESOURCE BUILD FAILED
    exit /b 1
)
popd

echo [5/8] Building native x64 screensaver...
go build -trimpath -ldflags "-H windowsgui -s -w" -o "%SCR_OUTPUT%" .
if errorlevel 1 (
    if exist "%RESOURCE_OBJECT%" del /q "%RESOURCE_OBJECT%"
    popd
    echo SCREENSAVER BUILD FAILED
    exit /b 1
)
if exist "%RESOURCE_OBJECT%" del /q "%RESOURCE_OBJECT%"

echo [6/8] Verifying application architecture...
go version -m "%APP_OUTPUT%" | findstr /c:"GOARCH=amd64" >nul
if errorlevel 1 (
    popd
    echo APPLICATION IS NOT AMD64
    exit /b 1
)

echo [7/8] Verifying screensaver architecture...
go version -m "%SCR_OUTPUT%" | findstr /c:"GOARCH=amd64" >nul
if errorlevel 1 (
    popd
    echo SCREENSAVER IS NOT AMD64
    exit /b 1
)

echo [8/8] Archiving timestamped artifacts...
if not exist "%HISTORY_DIR%" mkdir "%HISTORY_DIR%"
copy /y "%APP_OUTPUT%" "%APP_HISTORY%" >nul
if errorlevel 1 (
    popd
    echo TIMESTAMPED APPLICATION ARCHIVE FAILED
    exit /b 1
)
copy /y "%SCR_OUTPUT%" "%SCR_HISTORY%" >nul
if errorlevel 1 (
    popd
    echo TIMESTAMPED SCREENSAVER ARCHIVE FAILED
    exit /b 1
)
popd

echo Done!
echo.
echo Output:
echo   %APP_OUTPUT% - native Windows 11 x64 desktop app
echo   %SCR_OUTPUT% - native Windows 11 x64 screensaver
echo   %APP_HISTORY% - timestamped application build
echo   %SCR_HISTORY% - timestamped screensaver build
echo.
echo Source-only build: Sierra data is not embedded.
echo Uses a nearby verified scrantic folder by default; --data-dir overrides it.
echo.
