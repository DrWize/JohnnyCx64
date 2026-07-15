@echo off
setlocal

if not defined MSYS2_ROOT set "MSYS2_ROOT=C:\msys64"
set "PATH=C:\Program Files\Go\bin;%MSYS2_ROOT%\mingw64\bin;%PATH%"
set "CGO_ENABLED=1"
set "GOOS=windows"
set "GOARCH=amd64"
set "CC=%MSYS2_ROOT%\mingw64\bin\gcc.exe"

pushd "%~dp0.."
go test ./...
set "RESULT=%ERRORLEVEL%"
popd
exit /b %RESULT%
