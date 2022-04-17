@pushd %~dp0

@call _mainsrcnames.bat

go run "%_mainsrc%" %*

@popd