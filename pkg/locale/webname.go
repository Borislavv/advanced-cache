package locale

type LanguageCode string

const (
	WebnameAmharic           LanguageCode = "aa"
	WebnameAlbanian          LanguageCode = "al"
	WebnameKurdish           LanguageCode = "am" // Kurdish (Badînî)
	WebnameArabian           LanguageCode = "ar"
	WebnameEnglishAustralia  LanguageCode = "au" // English (Australia)
	WebnameAzerbaijanian     LanguageCode = "az"
	WebnameBulgarian         LanguageCode = "bg"
	WebnameBengali           LanguageCode = "bn"
	WebnamePortugueseBrazil  LanguageCode = "br" // Portuguese (Brazil)
	WebnameBosnian           LanguageCode = "bs"
	WebnameBelarusian        LanguageCode = "by"
	WebnameCanadian          LanguageCode = "ca" // Canadian English
	WebnameChinese           LanguageCode = "cn"
	WebnameCzech             LanguageCode = "cs"
	WebnameDanish            LanguageCode = "da"
	WebnameGerman            LanguageCode = "de"
	WebnameGreek             LanguageCode = "el"
	WebnameEnglish           LanguageCode = "en"
	WebnameEnglishBritain    LanguageCode = "gb"
	WebnameSpanish           LanguageCode = "es"
	WebnameSpanishPeruvian   LanguageCode = "pe"
	WebnameEstonian          LanguageCode = "et"
	WebnameIranian           LanguageCode = "fa"
	WebnameFinnish           LanguageCode = "fi"
	WebnameFrench            LanguageCode = "fr"
	WebnameHebrew            LanguageCode = "he"
	WebnameHindu             LanguageCode = "hi"
	WebnameCantoneseHongKong LanguageCode = "hk" // Cantonese (Hong Kong)
	WebnameCroatian          LanguageCode = "hr"
	WebnameHaitianCreole     LanguageCode = "ht" // Haitian Creole
	WebnameHungarian         LanguageCode = "hu"
	WebnameArmenian          LanguageCode = "hy"
	WebnameIndonesian        LanguageCode = "id"
	WebnameIndian            LanguageCode = "in" // Indian English
	WebnameIraqi             LanguageCode = "iq"
	WebnameIcelandic         LanguageCode = "is"
	WebnameItalian           LanguageCode = "it"
	WebnameJapanese          LanguageCode = "ja"
	WebnameGeorgian          LanguageCode = "ka"
	WebnameKhmer             LanguageCode = "km"
	WebnameKorean            LanguageCode = "ko"
	WebnameKurdishSorani     LanguageCode = "ku" // Kurdish (Soranî)
	WebnameKazakh            LanguageCode = "kz"
	WebnameLingala           LanguageCode = "ln"
	WebnameLao               LanguageCode = "lo"
	WebnameKyrgyz            LanguageCode = "ky"
	WebnameLithuanian        LanguageCode = "lt"
	WebnameLatvian           LanguageCode = "lv"
	WebnameMacedonian        LanguageCode = "mk"
	WebnameMongolian         LanguageCode = "mn"
	WebnameMalay             LanguageCode = "ms"
	WebnameMexican           LanguageCode = "mx" // Mexican Spanish
	WebnameBurmese           LanguageCode = "my"
	WebnameNorwegian         LanguageCode = "nb"
	WebnameKurdishKurmanci   LanguageCode = "ne" // Kurdish (Kurmancî)
	WebnameDutch             LanguageCode = "nl"
	WebnameNewZealand        LanguageCode = "nz" // New Zealand English
	WebnamePolish            LanguageCode = "pl"
	WebnamePortuguese        LanguageCode = "pt"
	WebnameRomanian          LanguageCode = "ro"
	WebnameRussian           LanguageCode = "ru"
	WebnameNepali            LanguageCode = "sd"
	WebnameSinhalese         LanguageCode = "si"
	WebnameSlovak            LanguageCode = "sk"
	WebnameSlovenian         LanguageCode = "sl"
	WebnameSerbian           LanguageCode = "sr"
	WebnameSerbianLatin      LanguageCode = "sp"
	WebnameSwedish           LanguageCode = "sv"
	WebnameSwahili           LanguageCode = "sw"
	WebnameThai              LanguageCode = "th"
	WebnameTelugu            LanguageCode = "te"
	WebnameTajik             LanguageCode = "tj"
	WebnameTagalog           LanguageCode = "tl"
	WebnameTurkish           LanguageCode = "tr"
	WebnameChineseTaiwan     LanguageCode = "tw" // Chinese (Taiwan)
	WebnameUkrainian         LanguageCode = "ua"
	WebnameUrdu              LanguageCode = "ur"
	WebnameEnglishUSA        LanguageCode = "us" // English (USA)
	WebnameUzbek             LanguageCode = "uz"
	WebnameVietnamese        LanguageCode = "vi"
	WebnameIran              LanguageCode = "ir"
	WebnameKurdishZaza       LanguageCode = "zu" // Kurdish (Zaza)
	WebnameKoreanKR          LanguageCode = "kr" // корейский язык в платформе crex
	WebnameKoreanZH          LanguageCode = "zh" // китайский язык в платформе blockchair
	WebnameTajikTG           LanguageCode = "tg"
	WebnameUkrainianUK       LanguageCode = "uk"
	WebnameAlbanianSQ        LanguageCode = "sq"
	WebnameSomali            LanguageCode = "so"
	WebnameCantonese         LanguageCode = "er"
	WebnameTamil             LanguageCode = "ta"
)

func (w LanguageCode) Country() (enum CountryEnum, ok bool) {
	switch w {
	case WebnameAmharic:
		return CountryEthiopia, true
	case WebnameArabian, WebnameIraqi:
		return CountryUAE, true
	case WebnameAzerbaijanian:
		return CountryAzerbaijan, true
	case WebnameBelarusian:
		return CountryBelarus, true
	case WebnameBulgarian:
		return CountryBulgaria, true
	case WebnameBengali:
		return CountryBangladesh, true
	case WebnameBosnian:
		return CountryBosnia, true
	case WebnameCzech:
		return CountryCzechRepublic, true
	case WebnameDanish:
		return CountryDenmark, true
	case WebnameGerman:
		return CountryGermany, true
	case WebnameGreek:
		return CountryGreece, true
	case WebnameCanadian:
		return CountryCanada, true
	case WebnameEnglishBritain, WebnameEnglishAustralia, WebnameEnglish, WebnameEnglishUSA:
		return CountryUnitedKingdom, true
	case WebnameIndian, WebnameHindu:
		return CountryIndia, true
	case WebnameNepali:
		return CountryNepal, true
	case WebnameNewZealand:
		return CountryNewZealand, true
	case WebnameSpanish:
		return CountrySpain, true
	case WebnameSpanishPeruvian:
		return CountryPeru, true
	case WebnameMexican:
		return CountryMexico, true
	case WebnameEstonian:
		return CountryEstonia, true
	case WebnameIranian, WebnameIran:
		return CountryIran, true
	case WebnameFinnish:
		return CountryFinland, true
	case WebnameFrench:
		return CountryFrance, true
	case WebnameHebrew:
		return CountryIsrael, true
	case WebnameCroatian:
		return CountryCroatia, true
	case WebnameHaitianCreole:
		return CountryHaiti, true
	case WebnameHungarian:
		return CountryHungary, true
	case WebnameArmenian:
		return CountryArmenia, true
	case WebnameIndonesian:
		return CountryIndonesia, true
	case WebnameIcelandic:
		return CountryIceland, true
	case WebnameItalian:
		return CountryItaly, true
	case WebnameJapanese:
		return CountryJapan, true
	case WebnameGeorgian:
		return CountryGeorgia, true
	case WebnameKazakh:
		return CountryKazakhstan, true
	case WebnameKhmer:
		return CountryCambodia, true
	case WebnameKorean, WebnameKoreanKR:
		return CountryKorea, true
	case WebnameKurdish, WebnameKurdishKurmanci, WebnameKurdishSorani, WebnameKurdishZaza, WebnameTurkish:
		return CountryTurkey, true
	case WebnameLingala:
		return CountryCongoKinshasa, true
	case WebnameKyrgyz:
		return CountryKyrgyzstan, true
	case WebnameLao:
		return CountryLaos, true
	case WebnameLithuanian:
		return CountryLithuania, true
	case WebnameLatvian:
		return CountryLatvia, true
	case WebnameMacedonian:
		return CountryMacedonia, true
	case WebnameMongolian:
		return CountryMongolia, true
	case WebnameMalay:
		return CountryMalaysia, true
	case WebnameBurmese:
		return CountryMyanmar, true
	case WebnameNorwegian:
		return CountryNorway, true
	case WebnameDutch:
		return CountryNetherlands, true
	case WebnamePolish:
		return CountryPoland, true
	case WebnamePortugueseBrazil:
		return CountryBrazil, true
	case WebnamePortuguese:
		return CountryPortugal, true
	case WebnameRomanian:
		return CountryRomania, true
	case WebnameRussian:
		return CountryRussian, true
	case WebnameSinhalese:
		return CountrySriLanka, true
	case WebnameSlovak:
		return CountrySlovakia, true
	case WebnameSlovenian:
		return CountrySlovenia, true
	case WebnameAlbanian, WebnameAlbanianSQ:
		return CountryAlbania, true
	case WebnameSerbian, WebnameSerbianLatin:
		return CountrySerbia, true
	case WebnameSwedish:
		return CountrySweden, true
	case WebnameSwahili:
		return CountryKenya, true
	case WebnameTajik, WebnameTajikTG:
		return CountryTajikistan, true
	case WebnameTamil:
		return CountrySriLanka, true
	case WebnameThai:
		return CountryThailand, true
	case WebnameTelugu:
		return CountryIndia, true
	case WebnameTagalog:
		return CountryPhilippines, true
	case WebnameUkrainian, WebnameUkrainianUK:
		return CountryUkraine, true
	case WebnameUrdu:
		return CountryPakistan, true
	case WebnameUzbek:
		return CountryUzbekistan, true
	case WebnameVietnamese:
		return CountryVietNam, true
	case WebnameChinese, WebnameKoreanZH, WebnameCantonese:
		return CountryChina, true
	case WebnameCantoneseHongKong:
		return CountryHongKong, true
	case WebnameChineseTaiwan:
		return CountryTaiwan, true
	case WebnameSomali:
		return CountrySomalia, true
	default:
		return "", false
	}
}

func (w LanguageCode) IsoLang() (enum IsoLang, ok bool) {
	switch w {
	case WebnameAmharic:
		return IsoLangAmharic, true
	case WebnameKurdish:
		return IsoLangKurdish, true
	case WebnameArabian, WebnameIraqi:
		return IsoLangArabic, true
	case WebnameAzerbaijanian:
		return IsoLangAzerbaijani, true
	case WebnameBelarusian:
		return IsoLangBelarusian, true
	case WebnameBulgarian:
		return IsoLangBulgarian, true
	case WebnameBengali:
		return IsoLangBengali, true
	case WebnameBosnian:
		return IsoLangBosnian, true
	case WebnameCzech:
		return IsoLangCzech, true
	case WebnameDanish:
		return IsoLangDanish, true
	case WebnameGerman:
		return IsoLangGerman, true
	case WebnameGreek:
		return IsoLangGreek, true
	case WebnameCanadian, WebnameNewZealand, WebnameIndian, WebnameEnglishUSA, WebnameEnglish, WebnameEnglishAustralia, WebnameEnglishBritain:
		return IsoLangEnglish, true
	case WebnameSpanish, WebnameMexican, WebnameSpanishPeruvian:
		return IsoLangSpanish, true
	case WebnameEstonian:
		return IsoLangEstonian, true
	case WebnameIranian, WebnameIran:
		return IsoLangPersian, true
	case WebnameFinnish:
		return IsoLangFinnish, true
	case WebnameFrench:
		return IsoLangFrench, true
	case WebnameHebrew:
		return IsoLangHebrew, true
	case WebnameHindu:
		return IsoLangHindi, true
	case WebnameCroatian:
		return IsoLangCroatian, true
	case WebnameHaitianCreole:
		return IsoLangHaitian, true
	case WebnameHungarian:
		return IsoLangHungarian, true
	case WebnameArmenian:
		return IsoLangArmenian, true
	case WebnameIndonesian:
		return IsoLangIndonesian, true
	case WebnameIcelandic:
		return IsoLangIcelandic, true
	case WebnameItalian:
		return IsoLangItalian, true
	case WebnameJapanese:
		return IsoLangJapanese, true
	case WebnameGeorgian:
		return IsoLangGeorgian, true
	case WebnameKazakh:
		return IsoLangKazakh, true
	case WebnameKhmer:
		return IsoLangCambodia, true
	case WebnameKorean, WebnameKoreanKR:
		return IsoLangKorean, true
	case WebnameKurdishSorani:
		return IsoLangKurdish, true
	case WebnameLingala:
		return IsoLangLingala, true
	case WebnameKyrgyz:
		return IsoLangKyrgyz, true
	case WebnameLao:
		return IsoLangLao, true
	case WebnameLithuanian:
		return IsoLangLithuanian, true
	case WebnameLatvian:
		return IsoLangLatvian, true
	case WebnameMacedonian:
		return IsoLangMacedonian, true
	case WebnameMongolian:
		return IsoLangMongolian, true
	case WebnameMalay:
		return IsoLangMalay, true
	case WebnameBurmese:
		return IsoLangBurmese, true
	case WebnameNorwegian:
		return IsoLangNorwegianBokmal, true
	case WebnameKurdishKurmanci:
		return IsoLangKurdish, true
	case WebnameDutch:
		return IsoLangDutch, true
	case WebnamePolish:
		return IsoLangPolish, true
	case WebnamePortugueseBrazil, WebnamePortuguese:
		return IsoLangPortuguese, true
	case WebnameRomanian:
		return IsoLangRomanian, true
	case WebnameRussian:
		return IsoLangRussian, true
	case WebnameNepali:
		return IsoLangNepali, true
	case WebnameSinhalese:
		return IsoLangSinhala, true
	case WebnameSlovak:
		return IsoLangSlovak, true
	case WebnameSlovenian:
		return IsoLangSlovenian, true
	case WebnameAlbanian, WebnameAlbanianSQ:
		return IsoLangAlbanian, true
	case WebnameSerbian:
		return IsoLangSerbian, true
	case WebnameSerbianLatin:
		return IsoLangSerbianLatin, true
	case WebnameSwedish:
		return IsoLangSwedish, true
	case WebnameSwahili:
		return IsoLangSwahili, true
	case WebnameTajik, WebnameTajikTG:
		return IsoLangTajik, true
	case WebnameThai:
		return IsoLangThai, true
	case WebnameTelugu:
		return IsoLangTelugu, true
	case WebnameTagalog:
		return IsoLangTagalog, true
	case WebnameTurkish:
		return IsoLangTurkish, true
	case WebnameUkrainian, WebnameUkrainianUK:
		return IsoLangUkrainian, true
	case WebnameUrdu:
		return IsoLangUrdu, true
	case WebnameUzbek:
		return IsoLangUzbek, true
	case WebnameVietnamese:
		return IsoLangVietnamese, true
	case WebnameChinese, WebnameChineseTaiwan, WebnameKoreanZH, WebnameCantoneseHongKong, WebnameCantonese:
		return IsoLangChinese, true
	case WebnameKurdishZaza:
		return IsoLangKurdish, true
	case WebnameSomali:
		return IsoLangSomali, true
	case WebnameTamil:
		return IsoLangSinhala, true
	default:
		return "", false
	}
}

func (w LanguageCode) Locale() (enum Locale, ok bool) {
	switch w {
	case WebnameAmharic:
		return LocaleAmharicEthiopia, true
	case WebnameKurdish:
		return LocaleKurdishBadini, true
	case WebnameArabian, WebnameIraqi:
		return LocaleArabicUAE, true
	case WebnameAzerbaijanian:
		return LocaleAzerbaijaniAzerbaijan, true
	case WebnameBelarusian:
		return LocaleBelarusianBelarus, true
	case WebnameBulgarian:
		return LocaleBulgarianBulgaria, true
	case WebnameBengali:
		return LocaleBengaliBangladesh, true
	case WebnameBosnian:
		return LocaleBosnianBosnia, true
	case WebnameCzech:
		return LocaleCzechCzechRepublic, true
	case WebnameDanish:
		return LocaleDanishDenmark, true
	case WebnameGerman:
		return LocaleGermanGermany, true
	case WebnameGreek:
		return LocaleGreekGreece, true
	case WebnameCanadian:
		return LocaleEnglishCanada, true
	case WebnameEnglishBritain, WebnameEnglishAustralia, WebnameEnglish, WebnameEnglishUSA:
		return LocaleEnglishUnitedKingdom, true
	case WebnameIndian:
		return LocaleEnglishIndia, true
	case WebnameNewZealand:
		return LocaleEnglishNewZealand, true
	case WebnameSpanish:
		return LocaleSpanishSpain, true
	case WebnameSpanishPeruvian:
		return LocalePeruvianSpanish, true
	case WebnameMexican:
		return LocaleSpanishMexico, true
	case WebnameEstonian:
		return LocaleEstonianEstonia, true
	case WebnameIranian, WebnameIran:
		return LocalePersianIran, true
	case WebnameFinnish:
		return LocaleFinnishFinland, true
	case WebnameFrench:
		return LocaleFrenchFrance, true
	case WebnameHebrew:
		return LocaleHebrewIsrael, true
	case WebnameHindu:
		return LocaleHindiIndia, true
	case WebnameCroatian:
		return LocaleCroatianCroatia, true
	case WebnameHaitianCreole:
		return LocaleHaitianHaiti, true
	case WebnameHungarian:
		return LocaleHungarianHungary, true
	case WebnameArmenian:
		return LocaleArmenianArmenia, true
	case WebnameIndonesian:
		return LocaleIndonesianIndonesia, true
	case WebnameIcelandic:
		return LocaleIcelandicIceland, true
	case WebnameItalian:
		return LocaleItalianItaly, true
	case WebnameJapanese:
		return LocaleJapaneseJapan, true
	case WebnameGeorgian:
		return LocaleGeorgianGeorgia, true
	case WebnameKazakh:
		return LocaleKazakhKazakhstan, true
	case WebnameKhmer:
		return LocaleCentralKhmer, true
	case WebnameKorean, WebnameKoreanKR:
		return LocaleKoreanSouthKorea, true
	case WebnameKurdishSorani:
		return LocaleKurdishSorani, true
	case WebnameLingala:
		return LocaleLingalaCongo, true
	case WebnameKyrgyz:
		return LocaleKyrgyz, true
	case WebnameLao:
		return LocaleLao, true
	case WebnameLithuanian:
		return LocaleLithuanianLithuania, true
	case WebnameLatvian:
		return LocaleLatvianLatvia, true
	case WebnameMacedonian:
		return LocaleMacedonianMacedonia, true
	case WebnameMongolian:
		return LocaleMongolianMongolia, true
	case WebnameMalay:
		return LocaleMalayMalaysia, true
	case WebnameBurmese:
		return LocaleBurmeseMyanmar, true
	case WebnameNorwegian:
		return LocaleNorwegianBokmalNorway, true
	case WebnameKurdishKurmanci:
		return LocaleKurdishTurkey, true
	case WebnameDutch:
		return LocaleDutchNetherlands, true
	case WebnamePolish:
		return LocalePolishPoland, true
	case WebnamePortugueseBrazil:
		return LocalePortugueseBrazil, true
	case WebnamePortuguese:
		return LocalePortuguesePortugal, true
	case WebnameRomanian:
		return LocaleRomanianRomania, true
	case WebnameRussian:
		return LocaleRussianRussia, true
	case WebnameNepali:
		return LocaleNepaliNepal, true
	case WebnameSinhalese:
		return LocaleSinhalaSrilanka, true
	case WebnameSlovak:
		return LocaleSlovakSlovakia, true
	case WebnameSlovenian:
		return LocaleSlovenianSlovenia, true
	case WebnameAlbanian, WebnameAlbanianSQ:
		return LocaleAlbanianAlbania, true
	case WebnameSerbian:
		return LocaleSerbianSerbia, true
	case WebnameSerbianLatin:
		return LocaleSerbianSerbiaLatin, true
	case WebnameSwedish:
		return LocaleSwedishSweden, true
	case WebnameSwahili:
		return LocaleSwahiliKenya, true
	case WebnameTajik, WebnameTajikTG:
		return LocaleTajikTajikistan, true
	case WebnameTamil:
		return LocaleTamilSrilanka, true
	case WebnameThai:
		return LocaleThaiThailand, true
	case WebnameTelugu:
		return LocaleTelugu, true
	case WebnameTagalog:
		return LocaleTagalogPhilippines, true
	case WebnameTurkish:
		return LocaleTurkishTurkey, true
	case WebnameUkrainian, WebnameUkrainianUK:
		return LocaleUkrainianUkraine, true
	case WebnameUrdu:
		return LocaleUrduPakistan, true
	case WebnameUzbek:
		return LocaleUzbekUzbekistan, true
	case WebnameVietnamese:
		return LocaleVietnameseVietnam, true
	case WebnameChinese, WebnameKoreanZH:
		return LocaleChineseChina, true
	case WebnameCantoneseHongKong:
		return LocaleChineseHongKong, true
	case WebnameChineseTaiwan:
		return LocaleChineseTaiwan, true
	case WebnameKurdishZaza:
		return LocaleKurdishZaza, true
	case WebnameSomali:
		return LocaleSomaliSomalia, true
	case WebnameCantonese:
		return LocaleAfarEritrea, true
	default:
		return "", false
	}
}

func TryLanguageCodeFromString(value string) (LanguageCode, bool) {
	switch LanguageCode(value) {
	case WebnameAmharic, WebnameAlbanian, WebnameArabian, WebnameArmenian,
		WebnameAzerbaijanian, WebnameBelarusian, WebnameBengali, WebnameBosnian,
		WebnameBulgarian, WebnameBurmese, WebnameKhmer, WebnameChinese,
		WebnameCantoneseHongKong, WebnameChineseTaiwan, WebnameCroatian, WebnameCzech,
		WebnameDanish, WebnameDutch, WebnameCanadian, WebnameIndian, WebnameNewZealand,
		WebnameEnglish, WebnameEstonian, WebnameFinnish, WebnameFrench, WebnameGeorgian,
		WebnameGerman, WebnameGreek, WebnameHaitianCreole, WebnameHebrew, WebnameHindu,
		WebnameHungarian, WebnameIcelandic, WebnameIndonesian, WebnameItalian,
		WebnameJapanese, WebnameKazakh, WebnameKorean, WebnameKurdishKurmanci,
		WebnameKurdishZaza, WebnameKurdish, WebnameKurdishSorani, WebnameKyrgyz,
		WebnameLao, WebnameLatvian, WebnameLingala, WebnameLithuanian, WebnameMacedonian,
		WebnameMalay, WebnameMongolian, WebnameNepali, WebnameNorwegian, WebnameIranian,
		WebnameSpanishPeruvian, WebnamePolish, WebnamePortugueseBrazil,
		WebnamePortuguese, WebnameRomanian, WebnameRussian, WebnameSerbian,
		WebnameSerbianLatin, WebnameSinhalese, WebnameSlovak, WebnameSlovenian,
		WebnameSomali, WebnameMexican, WebnameSpanish, WebnameSwahili, WebnameSwedish,
		WebnameTagalog, WebnameTajik, WebnameThai, WebnameTelugu, WebnameTurkish,
		WebnameUkrainian, WebnameUrdu, WebnameUzbek, WebnameVietnamese, WebnameTamil,
		WebnameCantonese:
		return LanguageCode(value), true
	default:
		return "", false
	}
}

func LanguageCodeList() []LanguageCode {
	return []LanguageCode{
		WebnameAmharic, WebnameAlbanian, WebnameArabian, WebnameArmenian,
		WebnameAzerbaijanian, WebnameBelarusian, WebnameBengali, WebnameBosnian,
		WebnameBulgarian, WebnameBurmese, WebnameKhmer, WebnameChinese,
		WebnameCantoneseHongKong, WebnameChineseTaiwan, WebnameCroatian, WebnameCzech,
		WebnameDanish, WebnameDutch, WebnameCanadian, WebnameIndian, WebnameNewZealand,
		WebnameEnglish, WebnameEstonian, WebnameFinnish, WebnameFrench, WebnameGeorgian,
		WebnameGerman, WebnameGreek, WebnameHaitianCreole, WebnameHebrew, WebnameHindu,
		WebnameHungarian, WebnameIcelandic, WebnameIndonesian, WebnameItalian,
		WebnameJapanese, WebnameKazakh, WebnameKorean, WebnameKurdishKurmanci,
		WebnameKurdishZaza, WebnameKurdish, WebnameKurdishSorani, WebnameKyrgyz,
		WebnameLao, WebnameLatvian, WebnameLingala, WebnameLithuanian, WebnameMacedonian,
		WebnameMalay, WebnameMongolian, WebnameNepali, WebnameNorwegian, WebnameIranian,
		WebnameSpanishPeruvian, WebnamePolish, WebnamePortugueseBrazil,
		WebnamePortuguese, WebnameRomanian, WebnameRussian, WebnameSerbian,
		WebnameSerbianLatin, WebnameSinhalese, WebnameSlovak, WebnameSlovenian,
		WebnameSomali, WebnameMexican, WebnameSpanish, WebnameSwahili, WebnameSwedish,
		WebnameTagalog, WebnameTajik, WebnameThai, WebnameTelugu, WebnameTurkish,
		WebnameUkrainian, WebnameUrdu, WebnameUzbek, WebnameVietnamese, WebnameTamil,
		WebnameCantonese,
	}
}
