set GOARCH=amd64
set GOOS=linux
go build -o D:\Web\go\bin\VeTerran\VeTerran -ldflags="-s -w"
C:\Progs\System\upx-3.95-win64\upx.exe D:\Web\go\bin\VeTerran\VeTerran
C:\Progs\System\7-Zip\7z a D:\Web\go\bin\VeTerran\VeTerran-linux.zip D:\Web\go\bin\VeTerran\VeTerran
