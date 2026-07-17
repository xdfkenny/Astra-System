import type { TranslationMap } from "./types";
import { en } from "./en";
import { es } from "./es";
import { zhCN } from "./zh-CN";
import { fr } from "./fr";
import { ja } from "./ja";
import { ko } from "./ko";
import { hi } from "./hi";
import { ar } from "./ar";
import { pt } from "./pt";
import { ru } from "./ru";
import { bn } from "./bn";
import { de } from "./de";
import { ur } from "./ur";
import { tr } from "./tr";
import { zhTW } from "./zh-TW";
import { vi } from "./vi";
import { th } from "./th";
import type { LocaleCode } from "../index";

export const localeRegistry: Partial<Record<LocaleCode, TranslationMap>> = {
  "en": en,
  "es": es,
  "zh-CN": zhCN,
  "fr": fr,
  "ja": ja,
  "ko": ko,
  "hi": hi,
  "ar": ar,
  "pt": pt,
  "ru": ru,
  "bn": bn,
  "de": de,
  "ur": ur,
  "tr": tr,
  "zh-TW": zhTW,
  "vi": vi,
  "th": th,
};
