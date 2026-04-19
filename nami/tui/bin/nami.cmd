@echo off
setlocal
set "SCRIPT_DIR=%~dp0"
set "LOCAL_NODE_PATH=%SCRIPT_DIR%..\runtime\node\node.exe"
set "LAUNCHER_PATH=%~dp0nami.js"

if exist "%LOCAL_NODE_PATH%" (
	"%LOCAL_NODE_PATH%" "%LAUNCHER_PATH%" %*
	exit /b %ERRORLEVEL%
)

where node >nul 2>nul
if not errorlevel 1 (
	node "%LAUNCHER_PATH%" %*
	exit /b %ERRORLEVEL%
)

where bun >nul 2>nul
if not errorlevel 1 (
	bun "%LAUNCHER_PATH%" %*
	exit /b %ERRORLEVEL%
)

where deno >nul 2>nul
if not errorlevel 1 (
	deno run --allow-env --allow-read --allow-run "%LAUNCHER_PATH%" %*
	exit /b %ERRORLEVEL%
)

>&2 echo nami requires one of these runtimes on PATH: node, bun, deno.
exit /b 1