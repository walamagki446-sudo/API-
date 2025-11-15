@echo off
REM Red Dead Redemption 2 Site Finder - Windows Launcher
REM ====================================================

echo.
echo ========================================================
echo   RED DEAD REDEMPTION 2 - SITE FINDER LAUNCHER
echo ========================================================
echo.

REM Check if binary exists
if not exist "rdr2_site_finder.exe" (
    echo ERROR: rdr2_site_finder.exe not found!
    echo.
    echo Please build the tool first:
    echo   go build -o rdr2_site_finder.exe rdr2_site_finder.go
    echo.
    pause
    exit /b 1
)

echo Starting site finder...
echo.
echo TIP: Results will be saved to files when complete:
echo   - rdr2_sites_YYYYMMDD_HHMMSS.json (structured data)
echo   - rdr2_sites_YYYYMMDD_HHMMSS.txt  (readable format)
echo.
echo Press Ctrl+C to stop the search at any time.
echo.
echo ========================================================
echo.

REM Run the site finder
rdr2_site_finder.exe

echo.
echo ========================================================
echo   SEARCH COMPLETED
echo ========================================================
echo.
echo Check the generated files in this directory.
echo.
pause
