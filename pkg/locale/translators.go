package locale

type TranslatorsName string

const (
	TranslatorsAlbanian             TranslatorsName = "sq_AL"
	TranslatorsAmharic              TranslatorsName = "aa_ET"
	TranslatorsArabicUAE            TranslatorsName = "ar_AE"
	TranslatorsArmenian             TranslatorsName = "hy_AM"
	TranslatorsAzerbaijani          TranslatorsName = "az_AZ"
	TranslatorsBelarusian           TranslatorsName = "be_BY"
	TranslatorsBengali              TranslatorsName = "bn_BD"
	TranslatorsBosnian              TranslatorsName = "bs_BA"
	TranslatorsBulgarian            TranslatorsName = "bg_BG"
	TranslatorsBurmese              TranslatorsName = "my_MM"
	TranslatorsCanadianEnglish      TranslatorsName = "en_CA"
	TranslatorsCantonese            TranslatorsName = "aa_ER"
	TranslatorsCantoneseKaton       TranslatorsName = "zh_HK"
	TranslatorsChinese              TranslatorsName = "zh_CN"
	TranslatorsCroatian             TranslatorsName = "hr_HR"
	TranslatorsCzechCzechia         TranslatorsName = "cs_CZ"
	TranslatorsDanish               TranslatorsName = "da_DK"
	TranslatorsDutch                TranslatorsName = "nl_NL"
	TranslatorsEnglish              TranslatorsName = "en_GB"
	TranslatorsEstonian             TranslatorsName = "et_EE"
	TranslatorsFinnish              TranslatorsName = "fi_FI"
	TranslatorsFrench               TranslatorsName = "fr_FR"
	TranslatorsGeorgianGeorgia      TranslatorsName = "ka_GE"
	TranslatorsGerman               TranslatorsName = "de_DE"
	TranslatorsGreek                TranslatorsName = "el_GR"
	TranslatorsHaitianCreole        TranslatorsName = "ht_HT"
	TranslatorsHebrew               TranslatorsName = "he_IL"
	TranslatorsHindi                TranslatorsName = "hi_IN"
	TranslatorsHungarian            TranslatorsName = "hu_HU"
	TranslatorsIcelandic            TranslatorsName = "is_IS"
	TranslatorsIndianEnglish        TranslatorsName = "en_IN"
	TranslatorsIndonesian           TranslatorsName = "id_ID"
	TranslatorsItalian              TranslatorsName = "it_IT"
	TranslatorsJapanese             TranslatorsName = "ja_JP"
	TranslatorsKazakh               TranslatorsName = "kk_KZ"
	TranslatorsKhmer                TranslatorsName = "km_KH"
	TranslatorsKorean               TranslatorsName = "ko_KR"
	TranslatorsKurdishZaza          TranslatorsName = "zu_ZA"
	TranslatorsKurdishBadini        TranslatorsName = "am_ET"
	TranslatorsKurmanjiKurdish      TranslatorsName = "ne_NP"
	TranslatorsKyrgyz               TranslatorsName = "ky_KG"
	TranslatorsLao                  TranslatorsName = "lo_LA"
	TranslatorsLatvian              TranslatorsName = "lv_LV"
	TranslatorsLingala              TranslatorsName = "ln_CD"
	TranslatorsLithuanian           TranslatorsName = "lt_LT"
	TranslatorsMacedonian           TranslatorsName = "mk_MK"
	TranslatorsMalay                TranslatorsName = "ms_MY"
	TranslatorsMexicanSpanish       TranslatorsName = "es_MX"
	TranslatorsMongolian            TranslatorsName = "mn_MN"
	TranslatorsNepali               TranslatorsName = "sd_IN"
	TranslatorsNewZealandEnglish    TranslatorsName = "en_NZ"
	TranslatorsNorwegian            TranslatorsName = "nb_NO"
	TranslatorsPersianIran          TranslatorsName = "fa_IR"
	TranslatorsPeruvianSpanish      TranslatorsName = "es_PE"
	TranslatorsPolish               TranslatorsName = "pl_PL"
	TranslatorsPortuguese           TranslatorsName = "pt_PT"
	TranslatorsPortugueseBrazil     TranslatorsName = "pt_BR"
	TranslatorsRomanian             TranslatorsName = "ro_RO"
	TranslatorsRussian              TranslatorsName = "ru_RU"
	TranslatorsSerbianSerbia        TranslatorsName = "sr_RS"
	TranslatorsSerbianSerbiaLatin   TranslatorsName = "sr_SP"
	TranslatorsSinhala              TranslatorsName = "si_LK"
	TranslatorsSlovak               TranslatorsName = "sk_SK"
	TranslatorsSlovenian            TranslatorsName = "sl_SI"
	TranslatorsSomali               TranslatorsName = "so_SO"
	TranslatorsSoraniKurdish        TranslatorsName = "ku_TR"
	TranslatorsSpanish              TranslatorsName = "es_ES"
	TranslatorsSwahili              TranslatorsName = "sw_KE"
	TranslatorsSwedishSweden        TranslatorsName = "sv_SE"
	TranslatorsTagalog              TranslatorsName = "tl_PH"
	TranslatorsTajik                TranslatorsName = "tg_TJ"
	TranslatorsTajikTajikistan      TranslatorsName = "tj_TJ"
	TranslatorsTamilSriLanka        TranslatorsName = "ta_LK"
	TranslatorsThai                 TranslatorsName = "th_TH"
	TranslatorsTelugu               TranslatorsName = "te_TE"
	TranslatorsTraditionalChinese   TranslatorsName = "zh_TW"
	TranslatorsTurkish              TranslatorsName = "tr_TR"
	TranslatorsUkrainian            TranslatorsName = "uk_UA"
	TranslatorsUrdu                 TranslatorsName = "ur_PK"
	TranslatorsUzbek                TranslatorsName = "uz_UZ"
	TranslatorsUzbekLatin           TranslatorsName = "uz_Latn"
	TranslatorsUzbekLatinUzbekistan TranslatorsName = "uz_Latn_UZ"
	TranslatorsVietnamese           TranslatorsName = "vi_VN"
)

func (t TranslatorsName) Locale() (enum Locale, ok bool) {
	switch t {
	case TranslatorsCantonese:
		return LocaleAfarEritrea, true
	case TranslatorsAmharic:
		return LocaleAmharicEthiopia, true
	case TranslatorsAlbanian:
		return LocaleAlbanianAlbania, true
	case TranslatorsKurdishBadini:
		return LocaleKurdishBadini, true
	case TranslatorsArabicUAE:
		return LocaleArabicUAE, true
	case TranslatorsArmenian:
		return LocaleArmenianArmenia, true
	case TranslatorsAzerbaijani:
		return LocaleAzerbaijaniAzerbaijan, true
	case TranslatorsBelarusian:
		return LocaleBelarusianBelarus, true
	case TranslatorsBengali:
		return LocaleBengaliBangladesh, true
	case TranslatorsBosnian:
		return LocaleBosnianBosnia, true
	case TranslatorsBulgarian:
		return LocaleBulgarianBulgaria, true
	case TranslatorsBurmese:
		return LocaleBurmeseMyanmar, true
	case TranslatorsKhmer:
		return LocaleCentralKhmer, true
	case TranslatorsChinese:
		return LocaleChineseChina, true
	case TranslatorsCantoneseKaton:
		return LocaleChineseHongKong, true
	case TranslatorsTraditionalChinese:
		return LocaleChineseTaiwan, true
	case TranslatorsCroatian:
		return LocaleCroatianCroatia, true
	case TranslatorsCzechCzechia:
		return LocaleCzechCzechRepublic, true
	case TranslatorsDanish:
		return LocaleDanishDenmark, true
	case TranslatorsDutch:
		return LocaleDutchNetherlands, true
	case TranslatorsCanadianEnglish:
		return LocaleEnglishCanada, true
	case TranslatorsIndianEnglish:
		return LocaleEnglishIndia, true
	case TranslatorsNewZealandEnglish:
		return LocaleEnglishNewZealand, true
	case TranslatorsEnglish:
		return LocaleEnglishUnitedKingdom, true
	case TranslatorsEstonian:
		return LocaleEstonianEstonia, true
	case TranslatorsFinnish:
		return LocaleFinnishFinland, true
	case TranslatorsFrench:
		return LocaleFrenchFrance, true
	case TranslatorsGeorgianGeorgia:
		return LocaleGeorgianGeorgia, true
	case TranslatorsGerman:
		return LocaleGermanGermany, true
	case TranslatorsGreek:
		return LocaleGreekGreece, true
	case TranslatorsHaitianCreole:
		return LocaleHaitianHaiti, true
	case TranslatorsHebrew:
		return LocaleHebrewIsrael, true
	case TranslatorsHindi:
		return LocaleHindiIndia, true
	case TranslatorsHungarian:
		return LocaleHungarianHungary, true
	case TranslatorsIcelandic:
		return LocaleIcelandicIceland, true
	case TranslatorsIndonesian:
		return LocaleIndonesianIndonesia, true
	case TranslatorsItalian:
		return LocaleItalianItaly, true
	case TranslatorsJapanese:
		return LocaleJapaneseJapan, true
	case TranslatorsKazakh:
		return LocaleKazakhKazakhstan, true
	case TranslatorsKorean:
		return LocaleKoreanSouthKorea, true
	case TranslatorsSoraniKurdish:
		return LocaleKurdishSorani, true
	case TranslatorsKyrgyz:
		return LocaleKyrgyz, true
	case TranslatorsLao:
		return LocaleLao, true
	case TranslatorsLatvian:
		return LocaleLatvianLatvia, true
	case TranslatorsLingala:
		return LocaleLingalaCongo, true
	case TranslatorsLithuanian:
		return LocaleLithuanianLithuania, true
	case TranslatorsMacedonian:
		return LocaleMacedonianMacedonia, true
	case TranslatorsMalay:
		return LocaleMalayMalaysia, true
	case TranslatorsMongolian:
		return LocaleMongolianMongolia, true
	case TranslatorsKurmanjiKurdish:
		return LocaleKurdishTurkey, true
	case TranslatorsNorwegian:
		return LocaleNorwegianBokmalNorway, true
	case TranslatorsPersianIran:
		return LocalePersianIran, true
	case TranslatorsPeruvianSpanish:
		return LocalePeruvianSpanish, true
	case TranslatorsPolish:
		return LocalePolishPoland, true
	case TranslatorsPortugueseBrazil:
		return LocalePortugueseBrazil, true
	case TranslatorsPortuguese:
		return LocalePortuguesePortugal, true
	case TranslatorsRomanian:
		return LocaleRomanianRomania, true
	case TranslatorsRussian:
		return LocaleRussianRussia, true
	case TranslatorsSerbianSerbia:
		return LocaleSerbianSerbia, true
	case TranslatorsSerbianSerbiaLatin:
		return LocaleSerbianSerbiaLatin, true
	case TranslatorsNepali:
		return LocaleNepaliNepal, true
	case TranslatorsSinhala:
		return LocaleSinhalaSrilanka, true
	case TranslatorsSlovak:
		return LocaleSlovakSlovakia, true
	case TranslatorsSlovenian:
		return LocaleSlovenianSlovenia, true
	case TranslatorsSomali:
		return LocaleSomaliSomalia, true
	case TranslatorsMexicanSpanish:
		return LocaleSpanishMexico, true
	case TranslatorsSpanish:
		return LocaleSpanishSpain, true
	case TranslatorsSwahili:
		return LocaleSwahiliKenya, true
	case TranslatorsSwedishSweden:
		return LocaleSwedishSweden, true
	case TranslatorsTagalog:
		return LocaleTagalogPhilippines, true
	case TranslatorsTajik:
		return LocaleTajikTajikistan, true
	case TranslatorsThai:
		return LocaleThaiThailand, true
	case TranslatorsTelugu:
		return LocaleTelugu, true
	case TranslatorsTurkish:
		return LocaleTurkishTurkey, true
	case TranslatorsUkrainian:
		return LocaleUkrainianUkraine, true
	case TranslatorsUrdu:
		return LocaleUrduPakistan, true
	case TranslatorsUzbek:
		return LocaleUzbekUzbekistan, true
	case TranslatorsVietnamese:
		return LocaleVietnameseVietnam, true
	case TranslatorsKurdishZaza:
		return LocaleKurdishZaza, true
	case TranslatorsTajikTajikistan:
		return LocaleTajikTajikistan, true
	case TranslatorsTamilSriLanka:
		return LocaleTamilSrilanka, true
	case TranslatorsUzbekLatin:
		return LocaleUzbekUzbekistan, true
	case TranslatorsUzbekLatinUzbekistan:
		return LocaleUzbekUzbekistan, true
	default:
		return "", false
	}
}

func TryTranslatorsNameFromString(value string) (TranslatorsName, bool) {
	switch TranslatorsName(value) {
	case TranslatorsCantonese, TranslatorsAmharic, TranslatorsAlbanian, TranslatorsArabicUAE,
		TranslatorsArmenian, TranslatorsAzerbaijani, TranslatorsBelarusian, TranslatorsBengali,
		TranslatorsBosnian, TranslatorsBulgarian, TranslatorsBurmese, TranslatorsKhmer,
		TranslatorsChinese, TranslatorsCantoneseKaton, TranslatorsTraditionalChinese,
		TranslatorsCroatian, TranslatorsCzechCzechia, TranslatorsDanish, TranslatorsDutch,
		TranslatorsCanadianEnglish, TranslatorsIndianEnglish, TranslatorsNewZealandEnglish,
		TranslatorsEnglish, TranslatorsEstonian, TranslatorsFinnish, TranslatorsFrench,
		TranslatorsGeorgianGeorgia, TranslatorsGerman, TranslatorsGreek,
		TranslatorsHaitianCreole, TranslatorsHebrew, TranslatorsHindi, TranslatorsHungarian,
		TranslatorsIcelandic, TranslatorsIndonesian, TranslatorsItalian, TranslatorsJapanese,
		TranslatorsKazakh, TranslatorsKorean, TranslatorsKurmanjiKurdish,
		TranslatorsKurdishZaza, TranslatorsKurdishBadini, TranslatorsSoraniKurdish,
		TranslatorsKyrgyz, TranslatorsLao, TranslatorsLatvian, TranslatorsLingala,
		TranslatorsLithuanian, TranslatorsMacedonian, TranslatorsMalay, TranslatorsMongolian,
		TranslatorsNepali, TranslatorsNorwegian, TranslatorsPersianIran,
		TranslatorsPeruvianSpanish, TranslatorsPolish, TranslatorsPortugueseBrazil,
		TranslatorsPortuguese, TranslatorsRomanian, TranslatorsRussian,
		TranslatorsSerbianSerbia, TranslatorsSerbianSerbiaLatin, TranslatorsSinhala,
		TranslatorsSlovak, TranslatorsSlovenian, TranslatorsSomali,
		TranslatorsMexicanSpanish, TranslatorsSpanish, TranslatorsSwahili,
		TranslatorsSwedishSweden, TranslatorsTagalog, TranslatorsTajik, TranslatorsThai,
		TranslatorsTelugu, TranslatorsTurkish, TranslatorsUkrainian, TranslatorsUrdu,
		TranslatorsUzbek, TranslatorsVietnamese, TranslatorsTamilSriLanka:
		return TranslatorsName(value), true
	default:
		return "", false
	}
}

func TranslatorsList() []TranslatorsName {
	return []TranslatorsName{
		TranslatorsCantonese, TranslatorsAmharic, TranslatorsAlbanian, TranslatorsArabicUAE,
		TranslatorsArmenian, TranslatorsAzerbaijani, TranslatorsBelarusian, TranslatorsBengali,
		TranslatorsBosnian, TranslatorsBulgarian, TranslatorsBurmese, TranslatorsKhmer,
		TranslatorsChinese, TranslatorsCantoneseKaton, TranslatorsTraditionalChinese,
		TranslatorsCroatian, TranslatorsCzechCzechia, TranslatorsDanish, TranslatorsDutch,
		TranslatorsCanadianEnglish, TranslatorsIndianEnglish, TranslatorsNewZealandEnglish,
		TranslatorsEnglish, TranslatorsEstonian, TranslatorsFinnish, TranslatorsFrench,
		TranslatorsGeorgianGeorgia, TranslatorsGerman, TranslatorsGreek,
		TranslatorsHaitianCreole, TranslatorsHebrew, TranslatorsHindi, TranslatorsHungarian,
		TranslatorsIcelandic, TranslatorsIndonesian, TranslatorsItalian, TranslatorsJapanese,
		TranslatorsKazakh, TranslatorsKorean, TranslatorsKurmanjiKurdish,
		TranslatorsKurdishZaza, TranslatorsKurdishBadini, TranslatorsSoraniKurdish,
		TranslatorsKyrgyz, TranslatorsLao, TranslatorsLatvian, TranslatorsLingala,
		TranslatorsLithuanian, TranslatorsMacedonian, TranslatorsMalay, TranslatorsMongolian,
		TranslatorsNepali, TranslatorsNorwegian, TranslatorsPersianIran,
		TranslatorsPeruvianSpanish, TranslatorsPolish, TranslatorsPortugueseBrazil,
		TranslatorsPortuguese, TranslatorsRomanian, TranslatorsRussian,
		TranslatorsSerbianSerbia, TranslatorsSerbianSerbiaLatin, TranslatorsSinhala,
		TranslatorsSlovak, TranslatorsSlovenian, TranslatorsSomali,
		TranslatorsMexicanSpanish, TranslatorsSpanish, TranslatorsSwahili,
		TranslatorsSwedishSweden, TranslatorsTagalog, TranslatorsTajik, TranslatorsThai,
		TranslatorsTelugu, TranslatorsTurkish, TranslatorsUkrainian, TranslatorsUrdu,
		TranslatorsUzbek, TranslatorsVietnamese, TranslatorsTamilSriLanka,
	}
}
