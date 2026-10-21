package tgbot

import _ "embed"

//go:embed assets/main-menu-banner.jpg
var mainMenuBanner []byte

func MainMenuBanner() []byte {
	return mainMenuBanner
}
