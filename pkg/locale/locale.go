package locale

type Locale string

const (
	LocaleAfarEthiopia          Locale = "aa_ET"
	LocaleAlbanianAlbania       Locale = "sq_AL"
	LocaleAmharicEthiopia       Locale = "am_ET"
	LocaleArabicUAE             Locale = "ar_AE"
	LocaleArmenianArmenia       Locale = "hy_AM"
	LocaleAzerbaijaniAzerbaijan Locale = "az_AZ"
	LocaleBelarusianBelarus     Locale = "be_BY"
	LocaleBengaliBangladesh     Locale = "bn_BD"
	LocaleBosnianBosnia         Locale = "bs_BA"
	LocaleBulgarianBulgaria     Locale = "bg_BG"
	LocaleBurmeseMyanmar        Locale = "my_MM"
	LocaleAfarEritrea           Locale = "aa_ER"
	LocaleCentralKhmer          Locale = "km_KH"
	LocaleChineseChina          Locale = "zh_CN"
	LocaleChineseHongKong       Locale = "zh_HK"
	LocaleChineseTaiwan         Locale = "zh_TW"
	LocaleCroatianCroatia       Locale = "hr_HR"
	LocaleCzechCzechRepublic    Locale = "cs_CZ"
	LocaleDanishDenmark         Locale = "da_DK"
	LocaleDutchNetherlands      Locale = "nl_NL"
	LocaleEnglishCanada         Locale = "en_CA"
	LocaleEnglishIndia          Locale = "en_IN"
	LocaleEnglishNewZealand     Locale = "en_NZ"
	LocaleEnglishUnitedKingdom  Locale = "en_GB"
	LocaleEstonianEstonia       Locale = "et_EE"
	LocaleFinnishFinland        Locale = "fi_FI"
	LocaleFrenchFrance          Locale = "fr_FR"
	LocaleGeorgianGeorgia       Locale = "ka_GE"
	LocaleGermanGermany         Locale = "de_DE"
	LocaleGreekGreece           Locale = "el_GR"
	LocaleHaitianHaiti          Locale = "ht_HT"
	LocaleHebrewIsrael          Locale = "he_IL"
	LocaleHindiIndia            Locale = "hi_IN"
	LocaleHungarianHungary      Locale = "hu_HU"
	LocaleIcelandicIceland      Locale = "is_IS"
	LocaleIndonesianIndonesia   Locale = "id_ID"
	LocaleItalianItaly          Locale = "it_IT"
	LocaleJapaneseJapan         Locale = "ja_JP"
	LocaleKazakhKazakhstan      Locale = "kk_KZ"
	LocaleKoreanSouthKorea      Locale = "ko_KR"
	LocaleKurdishTurkey         Locale = "ku_TR"
	LocaleKurdishZaza           Locale = "ku_GE"
	LocaleKurdishBadini         Locale = "ku_IQ"
	LocaleKurdishSorani         Locale = "ku_IR"
	LocaleKyrgyz                Locale = "ky_KG"
	LocaleLao                   Locale = "lo_LA"
	LocaleLatvianLatvia         Locale = "lv_LV"
	LocaleLingalaCongo          Locale = "ln_CD"
	LocaleLithuanianLithuania   Locale = "lt_LT"
	LocaleMacedonianMacedonia   Locale = "mk_MK"
	LocaleMalayMalaysia         Locale = "ms_MY"
	LocaleMongolianMongolia     Locale = "mn_MN"
	LocaleNepaliNepal           Locale = "ne_NP"
	LocaleNorwegianBokmalNorway Locale = "nb_NO"
	LocalePersianIran           Locale = "fa_IR"
	LocalePeruvianSpanish       Locale = "es_PE"
	LocalePolishPoland          Locale = "pl_PL"
	LocalePortugueseBrazil      Locale = "pt_BR"
	LocalePortuguesePortugal    Locale = "pt_PT"
	LocaleRomanianRomania       Locale = "ro_RO"
	LocaleRussianRussia         Locale = "ru_RU"
	LocaleSerbianSerbia         Locale = "sr_RS"
	LocaleSerbianSerbiaLatin    Locale = "sr_SP"
	LocaleSindhiIndia           Locale = "sd_IN"
	LocaleSinhalaSrilanka       Locale = "si_LK"
	LocaleSlovakSlovakia        Locale = "sk_SK"
	LocaleSlovenianSlovenia     Locale = "sl_SI"
	LocaleSomaliSomalia         Locale = "so_SO"
	LocaleSpanishMexico         Locale = "es_MX"
	LocaleSpanishSpain          Locale = "es_ES"
	LocaleSwahiliKenya          Locale = "sw_KE"
	LocaleSwedishSweden         Locale = "sv_SE"
	LocaleTagalogPhilippines    Locale = "tl_PH"
	LocaleTajikTajikistan       Locale = "tg_TJ"
	LocaleThaiThailand          Locale = "th_TH"
	LocaleTelugu                Locale = "te_TE"
	LocaleTurkishTurkey         Locale = "tr_TR"
	LocaleUkrainianUkraine      Locale = "uk_UA"
	LocaleUrduPakistan          Locale = "ur_PK"
	LocaleUzbekUzbekistan       Locale = "uz_UZ"
	LocaleVietnameseVietnam     Locale = "vi_VN"
	LocaleZuluSouthafrica       Locale = "zu_ZA"
	LocaleTamilSrilanka         Locale = "ta_LK"
)

func (l Locale) LanguageCode() (enum LanguageCode, ok bool) {
	switch l {
	case LocaleAfarEritrea:
		return WebnameCantonese, true
	case LocaleAfarEthiopia:
		return WebnameAmharic, true
	case LocaleAlbanianAlbania:
		return WebnameAlbanian, true
	case LocaleAmharicEthiopia:
		return WebnameAmharic, true
	case LocaleArabicUAE:
		return WebnameArabian, true
	case LocaleArmenianArmenia:
		return WebnameArmenian, true
	case LocaleAzerbaijaniAzerbaijan:
		return WebnameAzerbaijanian, true
	case LocaleBelarusianBelarus:
		return WebnameBelarusian, true
	case LocaleBengaliBangladesh:
		return WebnameBengali, true
	case LocaleBosnianBosnia:
		return WebnameBosnian, true
	case LocaleBulgarianBulgaria:
		return WebnameBulgarian, true
	case LocaleBurmeseMyanmar:
		return WebnameBurmese, true
	case LocaleCentralKhmer:
		return WebnameKhmer, true
	case LocaleChineseChina:
		return WebnameChinese, true
	case LocaleChineseHongKong:
		return WebnameCantoneseHongKong, true
	case LocaleChineseTaiwan:
		return WebnameChineseTaiwan, true
	case LocaleCroatianCroatia:
		return WebnameCroatian, true
	case LocaleCzechCzechRepublic:
		return WebnameCzech, true
	case LocaleDanishDenmark:
		return WebnameDanish, true
	case LocaleDutchNetherlands:
		return WebnameDutch, true
	case LocaleEnglishCanada:
		return WebnameCanadian, true
	case LocaleEnglishIndia:
		return WebnameIndian, true
	case LocaleEnglishNewZealand:
		return WebnameNewZealand, true
	case LocaleEnglishUnitedKingdom:
		return WebnameEnglish, true
	case LocaleEstonianEstonia:
		return WebnameEstonian, true
	case LocaleFinnishFinland:
		return WebnameFinnish, true
	case LocaleFrenchFrance:
		return WebnameFrench, true
	case LocaleGeorgianGeorgia:
		return WebnameGeorgian, true
	case LocaleGermanGermany:
		return WebnameGerman, true
	case LocaleGreekGreece:
		return WebnameGreek, true
	case LocaleHaitianHaiti:
		return WebnameHaitianCreole, true
	case LocaleHebrewIsrael:
		return WebnameHebrew, true
	case LocaleHindiIndia:
		return WebnameHindu, true
	case LocaleHungarianHungary:
		return WebnameHungarian, true
	case LocaleIcelandicIceland:
		return WebnameIcelandic, true
	case LocaleIndonesianIndonesia:
		return WebnameIndonesian, true
	case LocaleItalianItaly:
		return WebnameItalian, true
	case LocaleJapaneseJapan:
		return WebnameJapanese, true
	case LocaleKazakhKazakhstan:
		return WebnameKazakh, true
	case LocaleKoreanSouthKorea:
		return WebnameKorean, true
	case LocaleKurdishTurkey:
		return WebnameKurdishKurmanci, true
	case LocaleKurdishZaza:
		return WebnameKurdishZaza, true
	case LocaleKurdishBadini:
		return WebnameKurdish, true
	case LocaleKurdishSorani:
		return WebnameKurdishSorani, true
	case LocaleKyrgyz:
		return WebnameKyrgyz, true
	case LocaleLao:
		return WebnameLao, true
	case LocaleLatvianLatvia:
		return WebnameLatvian, true
	case LocaleLingalaCongo:
		return WebnameLingala, true
	case LocaleLithuanianLithuania:
		return WebnameLithuanian, true
	case LocaleMacedonianMacedonia:
		return WebnameMacedonian, true
	case LocaleMalayMalaysia:
		return WebnameMalay, true
	case LocaleMongolianMongolia:
		return WebnameMongolian, true
	case LocaleNepaliNepal:
		return WebnameNepali, true
	case LocaleNorwegianBokmalNorway:
		return WebnameNorwegian, true
	case LocalePersianIran:
		return WebnameIranian, true
	case LocalePeruvianSpanish:
		return WebnameSpanishPeruvian, true
	case LocalePolishPoland:
		return WebnamePolish, true
	case LocalePortugueseBrazil:
		return WebnamePortugueseBrazil, true
	case LocalePortuguesePortugal:
		return WebnamePortuguese, true
	case LocaleRomanianRomania:
		return WebnameRomanian, true
	case LocaleRussianRussia:
		return WebnameRussian, true
	case LocaleSerbianSerbia:
		return WebnameSerbian, true
	case LocaleSerbianSerbiaLatin:
		return WebnameSerbianLatin, true
	case LocaleSindhiIndia:
		return WebnameNepali, true
	case LocaleSinhalaSrilanka:
		return WebnameSinhalese, true
	case LocaleSlovakSlovakia:
		return WebnameSlovak, true
	case LocaleSlovenianSlovenia:
		return WebnameSlovenian, true
	case LocaleSomaliSomalia:
		return WebnameSomali, true
	case LocaleSpanishMexico:
		return WebnameMexican, true
	case LocaleSpanishSpain:
		return WebnameSpanish, true
	case LocaleSwahiliKenya:
		return WebnameSwahili, true
	case LocaleSwedishSweden:
		return WebnameSwedish, true
	case LocaleTagalogPhilippines:
		return WebnameTagalog, true
	case LocaleTajikTajikistan:
		return WebnameTajik, true
	case LocaleThaiThailand:
		return WebnameThai, true
	case LocaleTelugu:
		return WebnameTelugu, true
	case LocaleTurkishTurkey:
		return WebnameTurkish, true
	case LocaleUkrainianUkraine:
		return WebnameUkrainian, true
	case LocaleUrduPakistan:
		return WebnameUrdu, true
	case LocaleUzbekUzbekistan:
		return WebnameUzbek, true
	case LocaleVietnameseVietnam:
		return WebnameVietnamese, true
	case LocaleZuluSouthafrica:
		return WebnameKurdishZaza, true
	case LocaleTamilSrilanka:
		return WebnameTamil, true
	default:
		return "", false
	}
}

func (l Locale) IsoLang() (enum IsoLang, ok bool) {
	switch l {
	case LocaleAfarEritrea:
		return IsoLangChinese, true
	case LocaleAfarEthiopia:
		return IsoLangAfar, true
	case LocaleAlbanianAlbania:
		return IsoLangAlbanian, true
	case LocaleAmharicEthiopia:
		return IsoLangAmharic, true
	case LocaleArabicUAE:
		return IsoLangArabic, true
	case LocaleArmenianArmenia:
		return IsoLangArmenian, true
	case LocaleAzerbaijaniAzerbaijan:
		return IsoLangAzerbaijani, true
	case LocaleBelarusianBelarus:
		return IsoLangBelarusian, true
	case LocaleBengaliBangladesh:
		return IsoLangBengali, true
	case LocaleBosnianBosnia:
		return IsoLangBosnian, true
	case LocaleBulgarianBulgaria:
		return IsoLangBulgarian, true
	case LocaleBurmeseMyanmar:
		return IsoLangBurmese, true
	case LocaleCentralKhmer:
		return IsoLangCambodia, true
	case LocaleChineseChina, LocaleChineseHongKong, LocaleChineseTaiwan:
		return IsoLangChinese, true
	case LocaleCroatianCroatia:
		return IsoLangCroatian, true
	case LocaleCzechCzechRepublic:
		return IsoLangCzech, true
	case LocaleDanishDenmark:
		return IsoLangDanish, true
	case LocaleDutchNetherlands:
		return IsoLangDutch, true
	case LocaleEnglishCanada, LocaleEnglishIndia, LocaleEnglishNewZealand, LocaleEnglishUnitedKingdom:
		return IsoLangEnglish, true
	case LocaleEstonianEstonia:
		return IsoLangEstonian, true
	case LocaleFinnishFinland:
		return IsoLangFinnish, true
	case LocaleFrenchFrance:
		return IsoLangFrench, true
	case LocaleGeorgianGeorgia:
		return IsoLangGeorgian, true
	case LocaleGermanGermany:
		return IsoLangGerman, true
	case LocaleGreekGreece:
		return IsoLangGreek, true
	case LocaleHaitianHaiti:
		return IsoLangHaitian, true
	case LocaleHebrewIsrael:
		return IsoLangHebrew, true
	case LocaleHindiIndia:
		return IsoLangHindi, true
	case LocaleHungarianHungary:
		return IsoLangHungarian, true
	case LocaleIcelandicIceland:
		return IsoLangIcelandic, true
	case LocaleIndonesianIndonesia:
		return IsoLangIndonesian, true
	case LocaleItalianItaly:
		return IsoLangItalian, true
	case LocaleJapaneseJapan:
		return IsoLangJapanese, true
	case LocaleKazakhKazakhstan:
		return IsoLangKazakh, true
	case LocaleKoreanSouthKorea:
		return IsoLangKorean, true
	case LocaleKurdishTurkey, LocaleKurdishZaza, LocaleKurdishBadini, LocaleKurdishSorani:
		return IsoLangKurdish, true
	case LocaleKyrgyz:
		return IsoLangKyrgyz, true
	case LocaleLao:
		return IsoLangLao, true
	case LocaleLatvianLatvia:
		return IsoLangLatvian, true
	case LocaleLingalaCongo:
		return IsoLangLingala, true
	case LocaleLithuanianLithuania:
		return IsoLangLithuanian, true
	case LocaleMacedonianMacedonia:
		return IsoLangMacedonian, true
	case LocaleMalayMalaysia:
		return IsoLangMalay, true
	case LocaleMongolianMongolia:
		return IsoLangMongolian, true
	case LocaleNepaliNepal:
		return IsoLangNepali, true
	case LocaleNorwegianBokmalNorway:
		return IsoLangNorwegianBokmal, true
	case LocalePersianIran:
		return IsoLangPersian, true
	case LocalePeruvianSpanish, LocaleSpanishMexico, LocaleSpanishSpain:
		return IsoLangSpanish, true
	case LocalePolishPoland:
		return IsoLangPolish, true
	case LocalePortugueseBrazil, LocalePortuguesePortugal:
		return IsoLangPortuguese, true
	case LocaleRomanianRomania:
		return IsoLangRomanian, true
	case LocaleRussianRussia:
		return IsoLangRussian, true
	case LocaleSerbianSerbia:
		return IsoLangSerbian, true
	case LocaleSerbianSerbiaLatin:
		return IsoLangSerbianLatin, true
	case LocaleSindhiIndia:
		return IsoLangSindhi, true
	case LocaleSinhalaSrilanka:
		return IsoLangSinhala, true
	case LocaleSlovakSlovakia:
		return IsoLangSlovak, true
	case LocaleSlovenianSlovenia:
		return IsoLangSlovenian, true
	case LocaleSomaliSomalia:
		return IsoLangSomali, true
	case LocaleSwahiliKenya:
		return IsoLangSwahili, true
	case LocaleSwedishSweden:
		return IsoLangSwedish, true
	case LocaleTagalogPhilippines:
		return IsoLangTagalog, true
	case LocaleTajikTajikistan:
		return IsoLangTajik, true
	case LocaleThaiThailand:
		return IsoLangThai, true
	case LocaleTelugu:
		return IsoLangTelugu, true
	case LocaleTurkishTurkey:
		return IsoLangTurkish, true
	case LocaleUkrainianUkraine:
		return IsoLangUkrainian, true
	case LocaleUrduPakistan:
		return IsoLangUrdu, true
	case LocaleUzbekUzbekistan:
		return IsoLangUzbek, true
	case LocaleVietnameseVietnam:
		return IsoLangVietnamese, true
	case LocaleZuluSouthafrica:
		return IsoLangZulu, true
	case LocaleTamilSrilanka:
		return IsoLangSinhala, true
	default:
		return "", false
	}
}

func (l Locale) TranslatorsName() (enum TranslatorsName, ok bool) {
	switch l {
	case LocaleAfarEritrea:
		return TranslatorsCantonese, true
	case LocaleAfarEthiopia:
		return TranslatorsAmharic, true
	case LocaleAlbanianAlbania:
		return TranslatorsAlbanian, true
	case LocaleAmharicEthiopia:
		return TranslatorsAmharic, true
	case LocaleArabicUAE:
		return TranslatorsArabicUAE, true
	case LocaleArmenianArmenia:
		return TranslatorsArmenian, true
	case LocaleAzerbaijaniAzerbaijan:
		return TranslatorsAzerbaijani, true
	case LocaleBelarusianBelarus:
		return TranslatorsBelarusian, true
	case LocaleBengaliBangladesh:
		return TranslatorsBengali, true
	case LocaleBosnianBosnia:
		return TranslatorsBosnian, true
	case LocaleBulgarianBulgaria:
		return TranslatorsBulgarian, true
	case LocaleBurmeseMyanmar:
		return TranslatorsBurmese, true
	case LocaleCentralKhmer:
		return TranslatorsKhmer, true
	case LocaleChineseChina:
		return TranslatorsChinese, true
	case LocaleChineseHongKong:
		return TranslatorsCantoneseKaton, true
	case LocaleChineseTaiwan:
		return TranslatorsTraditionalChinese, true
	case LocaleCroatianCroatia:
		return TranslatorsCroatian, true
	case LocaleCzechCzechRepublic:
		return TranslatorsCzechCzechia, true
	case LocaleDanishDenmark:
		return TranslatorsDanish, true
	case LocaleDutchNetherlands:
		return TranslatorsDutch, true
	case LocaleEnglishCanada:
		return TranslatorsCanadianEnglish, true
	case LocaleEnglishIndia:
		return TranslatorsIndianEnglish, true
	case LocaleEnglishNewZealand:
		return TranslatorsNewZealandEnglish, true
	case LocaleEnglishUnitedKingdom:
		return TranslatorsEnglish, true
	case LocaleEstonianEstonia:
		return TranslatorsEstonian, true
	case LocaleFinnishFinland:
		return TranslatorsFinnish, true
	case LocaleFrenchFrance:
		return TranslatorsFrench, true
	case LocaleGeorgianGeorgia:
		return TranslatorsGeorgianGeorgia, true
	case LocaleGermanGermany:
		return TranslatorsGerman, true
	case LocaleGreekGreece:
		return TranslatorsGreek, true
	case LocaleHaitianHaiti:
		return TranslatorsHaitianCreole, true
	case LocaleHebrewIsrael:
		return TranslatorsHebrew, true
	case LocaleHindiIndia:
		return TranslatorsHindi, true
	case LocaleHungarianHungary:
		return TranslatorsHungarian, true
	case LocaleIcelandicIceland:
		return TranslatorsIcelandic, true
	case LocaleIndonesianIndonesia:
		return TranslatorsIndonesian, true
	case LocaleItalianItaly:
		return TranslatorsItalian, true
	case LocaleJapaneseJapan:
		return TranslatorsJapanese, true
	case LocaleKazakhKazakhstan:
		return TranslatorsKazakh, true
	case LocaleKoreanSouthKorea:
		return TranslatorsKorean, true
	case LocaleKurdishTurkey:
		return TranslatorsKurmanjiKurdish, true
	case LocaleKurdishZaza:
		return TranslatorsKurdishZaza, true
	case LocaleKurdishBadini:
		return TranslatorsKurdishBadini, true
	case LocaleKurdishSorani:
		return TranslatorsSoraniKurdish, true
	case LocaleKyrgyz:
		return TranslatorsKyrgyz, true
	case LocaleLao:
		return TranslatorsLao, true
	case LocaleLatvianLatvia:
		return TranslatorsLatvian, true
	case LocaleLingalaCongo:
		return TranslatorsLingala, true
	case LocaleLithuanianLithuania:
		return TranslatorsLithuanian, true
	case LocaleMacedonianMacedonia:
		return TranslatorsMacedonian, true
	case LocaleMalayMalaysia:
		return TranslatorsMalay, true
	case LocaleMongolianMongolia:
		return TranslatorsMongolian, true
	case LocaleNepaliNepal:
		return TranslatorsNepali, true
	case LocaleNorwegianBokmalNorway:
		return TranslatorsNorwegian, true
	case LocalePersianIran:
		return TranslatorsPersianIran, true
	case LocalePeruvianSpanish:
		return TranslatorsPeruvianSpanish, true
	case LocalePolishPoland:
		return TranslatorsPolish, true
	case LocalePortugueseBrazil:
		return TranslatorsPortugueseBrazil, true
	case LocalePortuguesePortugal:
		return TranslatorsPortuguese, true
	case LocaleRomanianRomania:
		return TranslatorsRomanian, true
	case LocaleRussianRussia:
		return TranslatorsRussian, true
	case LocaleSerbianSerbia:
		return TranslatorsSerbianSerbia, true
	case LocaleSerbianSerbiaLatin:
		return TranslatorsSerbianSerbiaLatin, true
	case LocaleSindhiIndia:
		return TranslatorsNepali, true
	case LocaleSinhalaSrilanka:
		return TranslatorsSinhala, true
	case LocaleSlovakSlovakia:
		return TranslatorsSlovak, true
	case LocaleSlovenianSlovenia:
		return TranslatorsSlovenian, true
	case LocaleSomaliSomalia:
		return TranslatorsSomali, true
	case LocaleSpanishMexico:
		return TranslatorsMexicanSpanish, true
	case LocaleSpanishSpain:
		return TranslatorsSpanish, true
	case LocaleSwahiliKenya:
		return TranslatorsSwahili, true
	case LocaleSwedishSweden:
		return TranslatorsSwedishSweden, true
	case LocaleTagalogPhilippines:
		return TranslatorsTagalog, true
	case LocaleTajikTajikistan:
		return TranslatorsTajik, true
	case LocaleThaiThailand:
		return TranslatorsThai, true
	case LocaleTelugu:
		return TranslatorsTelugu, true
	case LocaleTurkishTurkey:
		return TranslatorsTurkish, true
	case LocaleUkrainianUkraine:
		return TranslatorsUkrainian, true
	case LocaleUrduPakistan:
		return TranslatorsUrdu, true
	case LocaleUzbekUzbekistan:
		return TranslatorsUzbek, true
	case LocaleVietnameseVietnam:
		return TranslatorsVietnamese, true
	case LocaleZuluSouthafrica:
		return TranslatorsKurdishZaza, true
	case LocaleTamilSrilanka:
		return TranslatorsTamilSriLanka, true
	default:
		return "", false
	}
}

func TryLocaleFromString(value string) (Locale, bool) {
	switch Locale(value) {
	case LocaleAfarEthiopia, LocaleAlbanianAlbania, LocaleAmharicEthiopia, LocaleArabicUAE,
		LocaleArmenianArmenia, LocaleAzerbaijaniAzerbaijan, LocaleBelarusianBelarus,
		LocaleBengaliBangladesh, LocaleBosnianBosnia, LocaleBulgarianBulgaria,
		LocaleBurmeseMyanmar, LocaleAfarEritrea, LocaleCentralKhmer, LocaleChineseChina,
		LocaleChineseHongKong, LocaleChineseTaiwan, LocaleCroatianCroatia,
		LocaleCzechCzechRepublic, LocaleDanishDenmark, LocaleDutchNetherlands,
		LocaleEnglishCanada, LocaleEnglishIndia, LocaleEnglishNewZealand,
		LocaleEnglishUnitedKingdom, LocaleEstonianEstonia, LocaleFinnishFinland,
		LocaleFrenchFrance, LocaleGeorgianGeorgia, LocaleGermanGermany, LocaleGreekGreece,
		LocaleHaitianHaiti, LocaleHebrewIsrael, LocaleHindiIndia, LocaleHungarianHungary,
		LocaleIcelandicIceland, LocaleIndonesianIndonesia, LocaleItalianItaly,
		LocaleJapaneseJapan, LocaleKazakhKazakhstan, LocaleKoreanSouthKorea,
		LocaleKurdishTurkey, LocaleKurdishZaza, LocaleKurdishBadini, LocaleKurdishSorani,
		LocaleKyrgyz, LocaleLao, LocaleLatvianLatvia, LocaleLingalaCongo,
		LocaleLithuanianLithuania, LocaleMacedonianMacedonia, LocaleMalayMalaysia,
		LocaleMongolianMongolia, LocaleNepaliNepal, LocaleNorwegianBokmalNorway,
		LocalePersianIran, LocalePeruvianSpanish, LocalePolishPoland,
		LocalePortugueseBrazil, LocalePortuguesePortugal, LocaleRomanianRomania,
		LocaleRussianRussia, LocaleSerbianSerbia, LocaleSerbianSerbiaLatin,
		LocaleSindhiIndia, LocaleSinhalaSrilanka, LocaleSlovakSlovakia,
		LocaleSlovenianSlovenia, LocaleSomaliSomalia, LocaleSpanishMexico,
		LocaleSpanishSpain, LocaleSwahiliKenya, LocaleSwedishSweden,
		LocaleTagalogPhilippines, LocaleTajikTajikistan, LocaleThaiThailand,
		LocaleTelugu, LocaleTurkishTurkey, LocaleUkrainianUkraine, LocaleUrduPakistan,
		LocaleUzbekUzbekistan, LocaleVietnameseVietnam, LocaleZuluSouthafrica,
		LocaleTamilSrilanka:
		return Locale(value), true
	default:
		return "", false
	}
}

func LocalesList() []Locale {
	return []Locale{
		LocaleAfarEthiopia, LocaleAlbanianAlbania, LocaleAmharicEthiopia, LocaleArabicUAE,
		LocaleArmenianArmenia, LocaleAzerbaijaniAzerbaijan, LocaleBelarusianBelarus,
		LocaleBengaliBangladesh, LocaleBosnianBosnia, LocaleBulgarianBulgaria,
		LocaleBurmeseMyanmar, LocaleAfarEritrea, LocaleCentralKhmer, LocaleChineseChina,
		LocaleChineseHongKong, LocaleChineseTaiwan, LocaleCroatianCroatia,
		LocaleCzechCzechRepublic, LocaleDanishDenmark, LocaleDutchNetherlands,
		LocaleEnglishCanada, LocaleEnglishIndia, LocaleEnglishNewZealand,
		LocaleEnglishUnitedKingdom, LocaleEstonianEstonia, LocaleFinnishFinland,
		LocaleFrenchFrance, LocaleGeorgianGeorgia, LocaleGermanGermany, LocaleGreekGreece,
		LocaleHaitianHaiti, LocaleHebrewIsrael, LocaleHindiIndia, LocaleHungarianHungary,
		LocaleIcelandicIceland, LocaleIndonesianIndonesia, LocaleItalianItaly,
		LocaleJapaneseJapan, LocaleKazakhKazakhstan, LocaleKoreanSouthKorea,
		LocaleKurdishTurkey, LocaleKurdishZaza, LocaleKurdishBadini, LocaleKurdishSorani,
		LocaleKyrgyz, LocaleLao, LocaleLatvianLatvia, LocaleLingalaCongo,
		LocaleLithuanianLithuania, LocaleMacedonianMacedonia, LocaleMalayMalaysia,
		LocaleMongolianMongolia, LocaleNepaliNepal, LocaleNorwegianBokmalNorway,
		LocalePersianIran, LocalePeruvianSpanish, LocalePolishPoland,
		LocalePortugueseBrazil, LocalePortuguesePortugal, LocaleRomanianRomania,
		LocaleRussianRussia, LocaleSerbianSerbia, LocaleSerbianSerbiaLatin,
		LocaleSindhiIndia, LocaleSinhalaSrilanka, LocaleSlovakSlovakia,
		LocaleSlovenianSlovenia, LocaleSomaliSomalia, LocaleSpanishMexico,
		LocaleSpanishSpain, LocaleSwahiliKenya, LocaleSwedishSweden,
		LocaleTagalogPhilippines, LocaleTajikTajikistan, LocaleThaiThailand,
		LocaleTelugu, LocaleTurkishTurkey, LocaleUkrainianUkraine, LocaleUrduPakistan,
		LocaleUzbekUzbekistan, LocaleVietnameseVietnam, LocaleZuluSouthafrica,
		LocaleTamilSrilanka,
	}
}
