package parser

// Language code helpers (PTT's translate_languages surface).

// languageNames maps the ISO 639-1 codes the parser emits to English
// language names (PTT parse.py LANGUAGES_TRANSLATION_TABLE).
var languageNames = map[string]string{
	"en": "English", "ja": "Japanese", "zh": "Chinese", "ru": "Russian",
	"ar": "Arabic", "pt": "Portuguese", "es": "Spanish", "fr": "French",
	"de": "German", "it": "Italian", "ko": "Korean", "hi": "Hindi",
	"bn": "Bengali", "pa": "Punjabi", "mr": "Marathi", "gu": "Gujarati",
	"ta": "Tamil", "te": "Telugu", "kn": "Kannada", "ml": "Malayalam",
	"th": "Thai", "vi": "Vietnamese", "id": "Indonesian", "tr": "Turkish",
	"he": "Hebrew", "fa": "Persian", "uk": "Ukrainian", "el": "Greek",
	"lt": "Lithuanian", "lv": "Latvian", "et": "Estonian", "pl": "Polish",
	"cs": "Czech", "sk": "Slovak", "hu": "Hungarian", "ro": "Romanian",
	"bg": "Bulgarian", "sr": "Serbian", "hr": "Croatian", "sl": "Slovenian",
	"nl": "Dutch", "da": "Danish", "fi": "Finnish", "sv": "Swedish",
	"no": "Norwegian", "ms": "Malay", "la": "Latino",
}

// LanguageName returns the English name for a language code the parser
// emits, or "" when unknown.
func LanguageName(code string) string {
	return languageNames[code]
}

// TranslateLanguages converts language codes to English names, dropping
// codes with no translation (PTT translate_langs semantics).
func TranslateLanguages(codes []string) []string {
	out := make([]string, 0, len(codes))
	for _, c := range codes {
		if name, ok := languageNames[c]; ok {
			out = append(out, name)
		}
	}
	return out
}

// LanguageNames returns the result's languages as English names.
func (r *Result) LanguageNames() []string {
	return TranslateLanguages(r.Languages)
}
