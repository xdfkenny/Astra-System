import { create } from "zustand";
import { localeRegistry } from "./locales";

export type LocaleCode =
  | "en" | "es" | "zh-CN" | "fr"
  | "ja" | "ko" | "hi" | "ar" | "pt" | "ru" | "bn" | "de" | "ur" | "tr"
  | "zh-TW" | "vi" | "th";

export interface LanguageInfo {
  code: LocaleCode;
  name: string;
  nameEn: string;
  dir: "ltr" | "rtl";
  isMain: boolean;
}

export const LANGUAGES: LanguageInfo[] = [
  { code: "en", name: "English", nameEn: "English", dir: "ltr", isMain: true },
  { code: "es", name: "Español", nameEn: "Spanish", dir: "ltr", isMain: true },
  { code: "zh-CN", name: "简体中文", nameEn: "Chinese (Simplified)", dir: "ltr", isMain: true },
  { code: "fr", name: "Français", nameEn: "French", dir: "ltr", isMain: true },
  { code: "ja", name: "日本語", nameEn: "Japanese", dir: "ltr", isMain: false },
  { code: "ko", name: "한국어", nameEn: "Korean", dir: "ltr", isMain: false },
  { code: "hi", name: "हिन्दी", nameEn: "Hindi", dir: "ltr", isMain: false },
  { code: "ar", name: "العربية", nameEn: "Arabic", dir: "rtl", isMain: false },
  { code: "pt", name: "Português", nameEn: "Portuguese", dir: "ltr", isMain: false },
  { code: "ru", name: "Русский", nameEn: "Russian", dir: "ltr", isMain: false },
  { code: "bn", name: "বাংলা", nameEn: "Bengali", dir: "ltr", isMain: false },
  { code: "de", name: "Deutsch", nameEn: "German", dir: "ltr", isMain: false },
  { code: "ur", name: "اردو", nameEn: "Urdu", dir: "rtl", isMain: false },
  { code: "tr", name: "Türkçe", nameEn: "Turkish", dir: "ltr", isMain: false },
  { code: "zh-TW", name: "繁體中文", nameEn: "Chinese (Traditional)", dir: "ltr", isMain: false },
  { code: "vi", name: "Tiếng Việt", nameEn: "Vietnamese", dir: "ltr", isMain: false },
  { code: "th", name: "ไทย", nameEn: "Thai", dir: "ltr", isMain: false },
];

export const MAIN_LANGUAGES = LANGUAGES.filter((l) => l.isMain);
export const FEATURE_LANGUAGES = LANGUAGES.filter((l) => !l.isMain);

export interface LocaleState {
  locale: LocaleCode;
  dir: "ltr" | "rtl";
  setLocale: (locale: LocaleCode) => void;
}

export const useLocaleStore = create<LocaleState>((set) => ({
  locale: "en",
  dir: "ltr",
  setLocale: (locale) => {
    const lang = LANGUAGES.find((l) => l.code === locale);
    set({ locale, dir: lang?.dir ?? "ltr" });
  },
}));

export function useTranslation(): {
  t: (key: string, params?: Record<string, string | number>) => string;
  locale: LocaleCode;
  dir: "ltr" | "rtl";
} {
  const locale = useLocaleStore((s) => s.locale);
  const dir = useLocaleStore((s) => s.dir);
  const translations = localeRegistry[locale] ?? localeRegistry.en;

  function t(key: string, params?: Record<string, string | number>): string {
    let value = translations?.[key] ?? localeRegistry.en?.[key] ?? key;
    if (params) {
      for (const [k, v] of Object.entries(params)) {
        value = value.replace(`{${k}}`, String(v));
      }
    }
    return value;
  }

  return { t, locale, dir };
}
