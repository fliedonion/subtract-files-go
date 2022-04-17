@pushd %~dp0

@if not exist "%~dp0\dest" (
    mkdir "%~dp0\dest"
)

@call _mainsrcnames.bat

go build -o dest/ "%_mainsrc%"

@popd
