//go:build gdrive

package main

import "embed"

// Вшиваем и ASCII, и ключи (так как мы в корне, доступ к src/ есть)
//
//go:embed src/install_gdrive.ascii src/credentials.json src/EngineFixes.dll src/steam_fix.tar.gz src/mod-organizer.ico src/rfad-tui-launcher.ico src/tweaks.json src/SettingsUser.json src/innoextract
var bundledAssets embed.FS

// Функция-помощник для main.go
func getCreds() []byte {
	creds, _ := bundledAssets.ReadFile("src/credentials.json")
	return creds
}

func getAscii() []byte {
	asciiBytes, _ := bundledAssets.ReadFile("src/install_gdrive.ascii")
	return asciiBytes
}
