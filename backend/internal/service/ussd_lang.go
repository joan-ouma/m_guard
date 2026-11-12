package service

import "strings"

const (
	// langEN is the default menu and alert language.
	langEN = "en"
	// langSW is Kiswahili, chosen by a rider over SMS.
	langSW = "sw"
)

// smsLanguageWords are the texts a rider sends to pick a language, for example "SW" or "ENGLISH".
var smsLanguageWords = map[string]string{
	"en":        langEN,
	"english":   langEN,
	"sw":        langSW,
	"kiswahili": langSW,
	"swahili":   langSW,
}

// smsLanguageCommand reports whether a rider's text is a language choice, and which language.
func smsLanguageCommand(input string) (lang string, ok bool) {
	lang, ok = smsLanguageWords[strings.ToLower(strings.TrimSpace(input))]
	return lang, ok
}

// normalizeLang returns a supported language code, falling back to English.
func normalizeLang(lang string) string {
	if lang == langSW {
		return langSW
	}
	return langEN
}

// languageChangedText confirms a language choice, in the language just chosen.
func languageChangedText(lang string) string {
	return ussdText(lang,
		"Language set to English. Text MENU to open the menu.",
		"Lugha imewekwa kuwa Kiswahili. Tuma MENU kufungua menyu.")
}

// languageHint tells a rider on the welcome menu how to switch to the other language.
func languageHint(lang string) string {
	return ussdText(lang, "Text SW for Kiswahili", "Tuma EN kwa Kiingereza")
}

// ussdText returns the Kiswahili text when lang is Swahili, otherwise the English text.
func ussdText(lang, en, sw string) string {
	if lang == langSW {
		return sw
	}
	return en
}
