@echo off
setlocal enabledelayedexpansion
cd /d "%~dp0"
set "BIN=blacksand.exe"
if not exist "%BIN%" (
  echo [Blacksand] Executable not found: %BIN% 1>&2
  exit /b 1
)
set "RESTART_CODE=75"
set "MAX_DELAY=60"
set /a delay=1
:loop
"%BIN%" %*
set "code=!ERRORLEVEL!"
if "!code!"=="0" (
  echo [Blacksand] Exited normally
  exit /b 0
)
if "!code!"=="%RESTART_CODE%" (
  echo [Blacksand] Restart requested
  set /a delay=1
  goto loop
)
echo [Blacksand] Exited with code !code!; restarting in !delay!s 1>&2
set /a pings=!delay!+1
ping -n !pings! 127.0.0.1 >nul 2>&1
set /a delay=!delay!*2
if !delay! gtr %MAX_DELAY% set /a delay=%MAX_DELAY%
goto loop
