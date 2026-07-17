import { useCallback, useMemo, useRef, useState } from "react";
import { motion } from "framer-motion";
import { motion as motionTokens } from "@astra/design-tokens";
import { useKioskMachine } from "../machines/KioskMachineProvider";
import { MAIN_LANGUAGES, FEATURE_LANGUAGES, useTranslation } from "../i18n";
import type { LocaleCode } from "../i18n";

const VISIBLE_FEATURE_COUNT = 6;

export function LanguageSelectScreen(): React.JSX.Element {
  const { send } = useKioskMachine();
  const { t } = useTranslation();
  const [showAll, setShowAll] = useState(false);

  const handleSelect = useCallback(
    (code: LocaleCode) => {
      send({ type: "SET_LANGUAGE", locale: code });
    },
    [send],
  );

  const visibleFeatureLanguages = useMemo(
    () => (showAll ? FEATURE_LANGUAGES : FEATURE_LANGUAGES.slice(0, VISIBLE_FEATURE_COUNT)),
    [showAll],
  );

  return (
    <div className="flex h-full w-full flex-1 flex-col bg-linen overflow-hidden">
      {/* Animated background blobs */}
      <div className="absolute inset-0 pointer-events-none" aria-hidden="true">
        <motion.div
          className="absolute -left-[10%] -top-[10%] h-[50vh] w-[50vh] rounded-full bg-moss opacity-[0.04]"
          animate={{
            borderRadius: [
              "60% 40% 30% 70% / 60% 30% 70% 40%",
              "30% 60% 70% 40% / 50% 60% 30% 60%",
              "50% 60% 30% 60% / 30% 40% 70% 50%",
              "60% 40% 30% 70% / 60% 30% 70% 40%",
            ],
          }}
          transition={{ duration: 12, repeat: Infinity, ease: "easeInOut" }}
        />
        <motion.div
          className="absolute -bottom-[10%] -right-[10%] h-[45vh] w-[45vh] rounded-full bg-amber opacity-[0.03]"
          animate={{
            borderRadius: [
              "40% 60% 60% 40% / 50% 40% 60% 50%",
              "60% 30% 40% 70% / 40% 60% 40% 60%",
              "30% 70% 50% 50% / 60% 40% 50% 40%",
              "40% 60% 60% 40% / 50% 40% 60% 50%",
            ],
          }}
          transition={{ duration: 12, repeat: Infinity, ease: "easeInOut" }}
        />
      </div>

      {/* Content */}
      <div className="relative z-10 flex flex-1 flex-col px-6 py-8">
        {/* Header */}
        <div className="text-center mb-6">
          <h1 className="font-heading text-[42px] font-semibold tracking-tight text-charcoal">
            Astra
          </h1>
          <p className="mt-2 font-sans text-[16px] text-stone">
            {t("language.select")}
          </p>
        </div>

        {/* Language list */}
        <div className="flex-1 overflow-y-auto px-2">
          {/* Main languages section */}
          <div className="mb-2">
            <p className="font-sans text-[12px] font-medium uppercase tracking-wider text-stone/60 mb-2 px-1">
              {t("language.mainLanguages")}
            </p>
            <div className="flex flex-col gap-2">
              {MAIN_LANGUAGES.map((lang) => (
                <LanguageButton
                  key={lang.code}
                  name={lang.name}
                  nameEn={lang.nameEn}
                  dir={lang.dir}
                  onClick={() => { handleSelect(lang.code); }}
                />
              ))}
            </div>
          </div>

          {/* More languages section */}
          <div className="mt-4 mb-2">
            <p className="font-sans text-[12px] font-medium uppercase tracking-wider text-stone/60 mb-2 px-1">
              {t("language.moreLanguages")}
            </p>
            <div className="flex flex-col gap-2">
              {visibleFeatureLanguages.map((lang) => (
                <LanguageButton
                  key={lang.code}
                  name={lang.name}
                  nameEn={lang.nameEn}
                  dir={lang.dir}
                  onClick={() => { handleSelect(lang.code); }}
                />
              ))}
            </div>

            {!showAll && FEATURE_LANGUAGES.length > VISIBLE_FEATURE_COUNT && (
              <motion.button
                type="button"
                initial={{ opacity: 0, y: 4 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: 0.3 }}
                onClick={() => { setShowAll(true); }}
                className="mt-3 mx-auto flex items-center justify-center gap-1.5 h-11 w-full rounded-[14px] border border-taupe/40 bg-white/40 font-sans text-[14px] text-stone active:bg-white/60 transition-colors duration-100"
              >
                <span>{t("language.showAll", { count: FEATURE_LANGUAGES.length })}</span>
                <svg
                  viewBox="0 0 20 20"
                  className="h-4 w-4"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth={2}
                  strokeLinecap="round"
                >
                  <path d="M6 8l4 4 4-4" />
                </svg>
              </motion.button>
            )}
          </div>

          {/* Bottom spacer for safe area */}
          <div className="h-4" />
        </div>
      </div>
    </div>
  );
}

function LanguageButton({
  name,
  nameEn,
  dir,
  onClick,
}: {
  name: string;
  nameEn: string;
  dir: "ltr" | "rtl";
  onClick: () => void;
}): React.JSX.Element {
  const [pressed, setPressed] = useState(false);
  const btnRef = useRef<HTMLButtonElement>(null);

  const handlePointerDown = useCallback(() => { setPressed(true); }, []);
  const handlePointerUp = useCallback(() => { setPressed(false); }, []);
  const handlePointerLeave = useCallback(() => { setPressed(false); }, []);

  return (
    <motion.button
      ref={btnRef}
      type="button"
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.25, ease: motionTokens.easeOutExpo }}
      onPointerDown={handlePointerDown}
      onPointerUp={handlePointerUp}
      onPointerLeave={handlePointerLeave}
      onClick={onClick}
      className={`relative flex items-center justify-between h-14 w-full rounded-[16px] border border-taupe/30 bg-white/70 px-5 font-sans text-left transition-all duration-100 ${
        pressed ? "scale-[0.98] bg-warm-cream/80 shadow-sm" : "shadow-[0_1px_4px_rgba(45,42,38,0.04)]"
      }`}
      style={{ direction: dir }}
    >
      <span className="text-[18px] font-medium text-charcoal">{name}</span>
      <span className="text-[13px] text-stone/60">{nameEn}</span>
    </motion.button>
  );
}
