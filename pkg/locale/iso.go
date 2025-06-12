package locale

type IsoLang string

const (
	IsoLangAfar            IsoLang = "aa"
	IsoLangAlbanian        IsoLang = "sq"
	IsoLangAmharic         IsoLang = "am"
	IsoLangArabic          IsoLang = "ar"
	IsoLangArmenian        IsoLang = "hy"
	IsoLangAzerbaijani     IsoLang = "az"
	IsoLangBelarusian      IsoLang = "be"
	IsoLangBengali         IsoLang = "bn"
	IsoLangBosnian         IsoLang = "bs"
	IsoLangBulgarian       IsoLang = "bg"
	IsoLangBurmese         IsoLang = "my"
	IsoLangChinese         IsoLang = "zh"
	IsoLangCroatian        IsoLang = "hr"
	IsoLangCzech           IsoLang = "cs"
	IsoLangDanish          IsoLang = "da"
	IsoLangDutch           IsoLang = "nl" // Dutch(, Flemish
	IsoLangEnglish         IsoLang = "en"
	IsoLangEstonian        IsoLang = "et"
	IsoLangFinnish         IsoLang = "fi"
	IsoLangFrench          IsoLang = "fr"
	IsoLangGeorgian        IsoLang = "ka"
	IsoLangGerman          IsoLang = "de"
	IsoLangGreek           IsoLang = "el" // Greek(, Mod (1453–)
	IsoLangHaitian         IsoLang = "ht" // Haitian(, Haitian Creole
	IsoLangHebrew          IsoLang = "he"
	IsoLangHindi           IsoLang = "hi"
	IsoLangHungarian       IsoLang = "hu"
	IsoLangIcelandic       IsoLang = "is"
	IsoLangIndonesian      IsoLang = "id"
	IsoLangItalian         IsoLang = "it"
	IsoLangJapanese        IsoLang = "ja"
	IsoLangKazakh          IsoLang = "kk"
	IsoLangCambodia        IsoLang = "km"
	IsoLangKorean          IsoLang = "ko"
	IsoLangKurdish         IsoLang = "ku"
	IsoLangKyrgyz          IsoLang = "ky"
	IsoLangLao             IsoLang = "lo"
	IsoLangLatvian         IsoLang = "lv"
	IsoLangLingala         IsoLang = "ln"
	IsoLangLithuanian      IsoLang = "lt"
	IsoLangMacedonian      IsoLang = "mk"
	IsoLangMalay           IsoLang = "ms"
	IsoLangMongolian       IsoLang = "mn"
	IsoLangNepali          IsoLang = "ne"
	IsoLangNorwegianBokmal IsoLang = "nb" // Norwegian Bokmål
	IsoLangPersian         IsoLang = "fa"
	IsoLangPolish          IsoLang = "pl"
	IsoLangPortuguese      IsoLang = "pt"
	IsoLangRomanian        IsoLang = "ro" // Romanian(, Moldavian(, Moldovan
	IsoLangRussian         IsoLang = "ru"
	IsoLangSerbian         IsoLang = "sr"
	IsoLangSerbianLatin    IsoLang = "sp"
	IsoLangSindhi          IsoLang = "sd"
	IsoLangSinhala         IsoLang = "si" // Sinhala(, Sinhalese
	IsoLangSlovak          IsoLang = "sk"
	IsoLangSlovenian       IsoLang = "sl"
	IsoLangSomali          IsoLang = "so"
	IsoLangSpanish         IsoLang = "es" // Spanish(, Castilian
	IsoLangSwahili         IsoLang = "sw"
	IsoLangSwedish         IsoLang = "sv"
	IsoLangTagalog         IsoLang = "tl"
	IsoLangTajik           IsoLang = "tg"
	IsoLangThai            IsoLang = "th"
	IsoLangTelugu          IsoLang = "te"
	IsoLangTurkish         IsoLang = "tr"
	IsoLangUkrainian       IsoLang = "uk"
	IsoLangUrdu            IsoLang = "ur"
	IsoLangUzbek           IsoLang = "uz"
	IsoLangVietnamese      IsoLang = "vi"
	IsoLangZulu            IsoLang = "zu"
)

func (lang IsoLang) Locale() (enums []Locale, ok bool) {
	switch lang {
	case IsoLangAfar:
		return []Locale{LocaleAfarEthiopia}, true
	case IsoLangAmharic:
		return []Locale{LocaleAmharicEthiopia}, true
	case IsoLangArabic:
		return []Locale{LocaleArabicUAE}, true
	case IsoLangAzerbaijani:
		return []Locale{LocaleAzerbaijaniAzerbaijan}, true
	case IsoLangBelarusian:
		return []Locale{LocaleBelarusianBelarus}, true
	case IsoLangBulgarian:
		return []Locale{LocaleBulgarianBulgaria}, true
	case IsoLangBengali:
		return []Locale{LocaleBengaliBangladesh}, true
	case IsoLangBosnian:
		return []Locale{LocaleBosnianBosnia}, true
	case IsoLangCzech:
		return []Locale{LocaleCzechCzechRepublic}, true
	case IsoLangDanish:
		return []Locale{LocaleDanishDenmark}, true
	case IsoLangGerman:
		return []Locale{LocaleGermanGermany}, true
	case IsoLangGreek:
		return []Locale{LocaleGreekGreece}, true
	case IsoLangEnglish:
		return []Locale{
			LocaleEnglishUnitedKingdom,
			LocaleEnglishCanada,
			LocaleEnglishIndia,
			LocaleEnglishNewZealand,
		}, true
	case IsoLangSpanish:
		return []Locale{
			LocaleSpanishSpain,
			LocaleSpanishMexico,
		}, true
	case IsoLangEstonian:
		return []Locale{LocaleEstonianEstonia}, true
	case IsoLangPersian:
		return []Locale{LocalePersianIran}, true
	case IsoLangFinnish:
		return []Locale{LocaleFinnishFinland}, true
	case IsoLangFrench:
		return []Locale{LocaleFrenchFrance}, true
	case IsoLangHebrew:
		return []Locale{LocaleHebrewIsrael}, true
	case IsoLangHindi:
		return []Locale{LocaleHindiIndia}, true
	case IsoLangCroatian:
		return []Locale{LocaleCroatianCroatia}, true
	case IsoLangHaitian:
		return []Locale{LocaleHaitianHaiti}, true
	case IsoLangHungarian:
		return []Locale{LocaleHungarianHungary}, true
	case IsoLangArmenian:
		return []Locale{LocaleArmenianArmenia}, true
	case IsoLangIndonesian:
		return []Locale{LocaleIndonesianIndonesia}, true
	case IsoLangIcelandic:
		return []Locale{LocaleIcelandicIceland}, true
	case IsoLangItalian:
		return []Locale{LocaleItalianItaly}, true
	case IsoLangJapanese:
		return []Locale{LocaleJapaneseJapan}, true
	case IsoLangGeorgian:
		return []Locale{LocaleGeorgianGeorgia}, true
	case IsoLangKazakh:
		return []Locale{LocaleKazakhKazakhstan}, true
	case IsoLangCambodia:
		return []Locale{LocaleCentralKhmer}, true
	case IsoLangKorean:
		return []Locale{LocaleKoreanSouthKorea}, true
	case IsoLangKurdish:
		return []Locale{LocaleKurdishTurkey}, true
	case IsoLangLingala:
		return []Locale{LocaleLingalaCongo}, true
	case IsoLangKyrgyz:
		return []Locale{LocaleKyrgyz}, true
	case IsoLangLao:
		return []Locale{LocaleLao}, true
	case IsoLangLithuanian:
		return []Locale{LocaleLithuanianLithuania}, true
	case IsoLangLatvian:
		return []Locale{LocaleLatvianLatvia}, true
	case IsoLangMacedonian:
		return []Locale{LocaleMacedonianMacedonia}, true
	case IsoLangMongolian:
		return []Locale{LocaleMongolianMongolia}, true
	case IsoLangMalay:
		return []Locale{LocaleMalayMalaysia}, true
	case IsoLangBurmese:
		return []Locale{LocaleBurmeseMyanmar}, true
	case IsoLangNorwegianBokmal:
		return []Locale{LocaleNorwegianBokmalNorway}, true
	case IsoLangNepali:
		return []Locale{LocaleNepaliNepal}, true
	case IsoLangDutch:
		return []Locale{LocaleDutchNetherlands}, true
	case IsoLangPolish:
		return []Locale{LocalePolishPoland}, true
	case IsoLangPortuguese:
		return []Locale{
			LocalePortugueseBrazil,
			LocalePortuguesePortugal,
		}, true
	case IsoLangRomanian:
		return []Locale{LocaleRomanianRomania}, true
	case IsoLangRussian:
		return []Locale{LocaleRussianRussia}, true
	case IsoLangSindhi:
		return []Locale{LocaleSindhiIndia}, true
	case IsoLangSinhala:
		return []Locale{LocaleSinhalaSrilanka}, true
	case IsoLangSlovak:
		return []Locale{LocaleSlovakSlovakia}, true
	case IsoLangSlovenian:
		return []Locale{LocaleSlovenianSlovenia}, true
	case IsoLangSomali:
		return []Locale{LocaleSomaliSomalia}, true
	case IsoLangAlbanian:
		return []Locale{LocaleAlbanianAlbania}, true
	case IsoLangSerbian:
		return []Locale{LocaleSerbianSerbia}, true
	case IsoLangSerbianLatin:
		return []Locale{LocaleSerbianSerbiaLatin}, true
	case IsoLangSwedish:
		return []Locale{LocaleSwedishSweden}, true
	case IsoLangSwahili:
		return []Locale{LocaleSwahiliKenya}, true
	case IsoLangTajik:
		return []Locale{LocaleTajikTajikistan}, true
	case IsoLangThai:
		return []Locale{LocaleThaiThailand}, true
	case IsoLangTelugu:
		return []Locale{LocaleTelugu}, true
	case IsoLangTagalog:
		return []Locale{LocaleTagalogPhilippines}, true
	case IsoLangTurkish:
		return []Locale{LocaleTurkishTurkey}, true
	case IsoLangUkrainian:
		return []Locale{LocaleUkrainianUkraine}, true
	case IsoLangUrdu:
		return []Locale{LocaleUrduPakistan}, true
	case IsoLangUzbek:
		return []Locale{
			LocaleUzbekUzbekistan,
			LocaleUzbekUzbekistan,
		}, true
	case IsoLangVietnamese:
		return []Locale{LocaleVietnameseVietnam}, true
	case IsoLangChinese:
		return []Locale{
			LocaleChineseChina,
			LocaleChineseHongKong,
			LocaleChineseTaiwan,
		}, true
	case IsoLangZulu:
		return []Locale{LocaleZuluSouthafrica}, true
	default:
		return []Locale{}, false
	}
}

func TryIsoLangFromString(value string) (IsoLang, bool) {
	switch IsoLang(value) {
	case IsoLangAfar, IsoLangAlbanian, IsoLangAmharic, IsoLangArabic, IsoLangArmenian,
		IsoLangAzerbaijani, IsoLangBelarusian, IsoLangBengali, IsoLangBosnian,
		IsoLangBulgarian, IsoLangBurmese, IsoLangCambodia, IsoLangChinese,
		IsoLangCroatian, IsoLangCzech, IsoLangDanish, IsoLangDutch, IsoLangEnglish,
		IsoLangEstonian, IsoLangFinnish, IsoLangFrench, IsoLangGeorgian, IsoLangGerman,
		IsoLangGreek, IsoLangHaitian, IsoLangHebrew, IsoLangHindi, IsoLangHungarian,
		IsoLangIcelandic, IsoLangIndonesian, IsoLangItalian, IsoLangJapanese,
		IsoLangKazakh, IsoLangKorean, IsoLangKurdish, IsoLangKyrgyz, IsoLangLao,
		IsoLangLatvian, IsoLangLingala, IsoLangLithuanian, IsoLangMacedonian,
		IsoLangMalay, IsoLangMongolian, IsoLangNepali, IsoLangNorwegianBokmal,
		IsoLangPersian, IsoLangSpanish, IsoLangPolish, IsoLangPortuguese,
		IsoLangRomanian, IsoLangRussian, IsoLangSerbian, IsoLangSerbianLatin,
		IsoLangSindhi, IsoLangSinhala, IsoLangSlovak, IsoLangSlovenian, IsoLangSomali,
		IsoLangSwahili, IsoLangSwedish, IsoLangTagalog, IsoLangTajik, IsoLangThai,
		IsoLangTelugu, IsoLangTurkish, IsoLangUkrainian, IsoLangUrdu, IsoLangUzbek,
		IsoLangVietnamese, IsoLangZulu:
		return IsoLang(value), true
	default:
		return "", false
	}
}

func IsoList() []IsoLang {
	return []IsoLang{
		IsoLangAfar, IsoLangAlbanian, IsoLangAmharic, IsoLangArabic, IsoLangArmenian,
		IsoLangAzerbaijani, IsoLangBelarusian, IsoLangBengali, IsoLangBosnian,
		IsoLangBulgarian, IsoLangBurmese, IsoLangCambodia, IsoLangChinese,
		IsoLangCroatian, IsoLangCzech, IsoLangDanish, IsoLangDutch, IsoLangEnglish,
		IsoLangEstonian, IsoLangFinnish, IsoLangFrench, IsoLangGeorgian, IsoLangGerman,
		IsoLangGreek, IsoLangHaitian, IsoLangHebrew, IsoLangHindi, IsoLangHungarian,
		IsoLangIcelandic, IsoLangIndonesian, IsoLangItalian, IsoLangJapanese,
		IsoLangKazakh, IsoLangKorean, IsoLangKurdish, IsoLangKyrgyz, IsoLangLao,
		IsoLangLatvian, IsoLangLingala, IsoLangLithuanian, IsoLangMacedonian,
		IsoLangMalay, IsoLangMongolian, IsoLangNepali, IsoLangNorwegianBokmal,
		IsoLangPersian, IsoLangSpanish, IsoLangPolish, IsoLangPortuguese,
		IsoLangRomanian, IsoLangRussian, IsoLangSerbian, IsoLangSerbianLatin,
		IsoLangSindhi, IsoLangSinhala, IsoLangSlovak, IsoLangSlovenian, IsoLangSomali,
		IsoLangSwahili, IsoLangSwedish, IsoLangTagalog, IsoLangTajik, IsoLangThai,
		IsoLangTelugu, IsoLangTurkish, IsoLangUkrainian, IsoLangUrdu, IsoLangUzbek,
		IsoLangVietnamese, IsoLangZulu,
	}
}
