set GOOS=windows
set GOARCH=amd64
go build .
cd mediamtxlaunch
go build .
cd ..
signtool sign /tr http://timestamp.sectigo.com /td sha256 /fd sha256 /n "Thompson international Services" .\camerahls\camerahls.exe
copy .\mediamtxlaunch\camerahls.exe .
del winmediamtx.zip
"C:\Program Files\7-Zip\7z.exe" a -tzip -r winmediamtx mediamtx.exe camerahls.exe mediamtx.yml web -x!camerahls