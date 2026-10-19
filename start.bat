@echo off
rem  UTF-8  .      GBK  .
chcp 65001 >nul 2>&1
rem ARTEX   (Windows)
rem
rem :
rem   start.bat                   (Ctrl-C  )
rem   start.bat -addr :9000        artex   
rem
rem     . artex.exe  ,        .
rem
rem   0          ->  
rem   75        ->   ( " "  "" )
rem              ->    (1->2->4… 60)
rem
rem ·SHA256  ·    ,  artex     
rem (selfupdate ).   .   start.sh   .

setlocal enabledelayedexpansion
cd /d "%~dp0"

set "BIN=artex.exe"
if not exist "%BIN%" (
	echo [artex]     : %BIN% 1>&2
	exit /b 1
)

set "RESTART_CODE=75"
set "MAX_DELAY=60"
set /a delay=1

:loop
"%BIN%" %*
set "code=!ERRORLEVEL!"

if "!code!"=="0" (
	echo [artex]  
	exit /b 0
)

if "!code!"=="%RESTART_CODE%" (
	rem / .   artex      .
	echo [artex]  :   …
	set /a delay=1
	goto loop
)

echo [artex]   ^(code=!code!^), !delay!s   1>&2
rem timeout     ping  (N  N+1  ).
set /a pings=!delay!+1
ping -n !pings! 127.0.0.1 >nul 2>&1
set /a delay=!delay!*2
if !delay! gtr %MAX_DELAY% set /a delay=%MAX_DELAY%
goto loop
