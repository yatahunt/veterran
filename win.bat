set GOARCH=amd64
set GOOS=windows
go build -o D:\Go\bin\VeTerran\VeTerran.exe -ldflags="-s -w"
C:\Progs\System\upx-3.95-win64\upx.exe D:\Go\bin\VeTerran\VeTerran.exe
C:\Progs\System\7-Zip\7z a D:\Go\bin\VeTerran\VeTerran.zip D:\Go\bin\VeTerran\VeTerran.exe D:\Go\bin\VeTerran\ladderbots.json
