//go:build cdn

package main

import "embed"

//go:embed src/install_cdn.ascii src/EngineFixes.dll src/steam_fix.tar.gz src/mod-organizer.ico src/rfad-tui-launcher.ico src/tweaks.json
var bundledAssets embed.FS

// Заглушка, возвращает nil
func getCreds() []byte {
	return nil
}

func getAscii() []byte {
	asciiBytes, _ := bundledAssets.ReadFile("src/install_cdn.ascii")
	return asciiBytes
}
