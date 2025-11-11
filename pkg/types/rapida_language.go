/*
 *  Copyright (c) 2024. Rapida
 *
 *  Permission is hereby granted, free of charge, to any person obtaining a copy
 *  of this software and associated documentation files (the "Software"), to deal
 *  in the Software without restriction, including without limitation the rights
 *  to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 *  copies of the Software, and to permit persons to whom the Software is
 *  furnished to do so, subject to the following conditions:
 *
 *  The above copyright notice and this permission notice shall be included in
 *  all copies or substantial portions of the Software.
 *
 *  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 *  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 *  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 *  AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 *  LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 *  OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 *  THE SOFTWARE.
 *
 *  Author: Prashant <prashant@rapida.ai>
 *
 */
package types

import (
	"strings"
)

// Language holds language name and ISO codes
type Language struct {
	Name     string
	ISO639_1 string
	ISO639_2 string
}

var languages = map[string]Language{
	// 🌍 International Languages
	"en": {"English", "en", "eng"},
	"fr": {"French", "fr", "fra"},
	"es": {"Spanish", "es", "spa"},
	"de": {"German", "de", "deu"},
	"it": {"Italian", "it", "ita"},
	"pt": {"Portuguese", "pt", "por"},
	"ru": {"Russian", "ru", "rus"},
	"zh": {"Chinese", "zh", "zho"},
	"ja": {"Japanese", "ja", "jpn"},
	"ko": {"Korean", "ko", "kor"},
	"ar": {"Arabic", "ar", "ara"},
	"tr": {"Turkish", "tr", "tur"},
	"nl": {"Dutch", "nl", "nld"},
	"pl": {"Polish", "pl", "pol"},
	"sv": {"Swedish", "sv", "swe"},
	"no": {"Norwegian", "no", "nor"},
	"da": {"Danish", "da", "dan"},
	"fi": {"Finnish", "fi", "fin"},
	"he": {"Hebrew", "he", "heb"},
	"el": {"Greek", "el", "ell"},

	// 🇮🇳 Indian Languages
	"hi": {"Hindi", "hi", "hin"},
	"bn": {"Bengali", "bn", "ben"},
	"te": {"Telugu", "te", "tel"},
	"mr": {"Marathi", "mr", "mar"},
	"ta": {"Tamil", "ta", "tam"},
	"ur": {"Urdu", "ur", "urd"},
	"gu": {"Gujarati", "gu", "guj"},
	"kn": {"Kannada", "kn", "kan"},
	"ml": {"Malayalam", "ml", "mal"},
	"or": {"Odia", "or", "ori"},
	"pa": {"Punjabi", "pa", "pan"},
	"as": {"Assamese", "as", "asm"},
	"sa": {"Sanskrit", "sa", "san"},
	"sd": {"Sindhi", "sd", "snd"},
	"ks": {"Kashmiri", "ks", "kas"},
}

// Lookup function by long name
func GetLanguageByName(name string) Language {
	lang, ok := languages[strings.ToLower(name)]
	if !ok {
		lang = languages["en"]
	}
	return lang

}
