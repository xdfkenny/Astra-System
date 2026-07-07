import type { Config } from "tailwindcss";
import { semantic, slate } from "./src/tokens/colors";
import { borderRadius, boxShadow } from "./src/tokens/elevation";
import { duration, easing } from "./src/tokens/motion";
import { spacing } from "./src/tokens/spacing";
import {
  fontFamily,
  fontSize,
  fontWeight,
  letterSpacing,
  lineHeight,
} from "./src/tokens/typography";
import { zIndex } from "./src/tokens/z-index";

const preset: Config = {
  darkMode: "class",
  content: [],
  theme: {
    extend: {
      colors: {
        ...slate,
        primary: semantic.primary,
        "primary-hover": semantic["primary-hover"],
        cta: semantic.cta,
        "cta-hover": semantic["cta-hover"],
        error: semantic.error,
        "error-hover": semantic["error-hover"],
        success: semantic.success,
        warning: semantic.warning,
        info: semantic.info,
        background: semantic.background,
        surface: semantic.surface,
        "surface-elevated": semantic["surface-elevated"],
        "text-primary": semantic["text-primary"],
        "text-secondary": semantic["text-secondary"],
        "text-disabled": semantic["text-disabled"],
        border: semantic.border,
      },
      spacing,
      fontFamily,
      fontSize,
      fontWeight,
      lineHeight,
      letterSpacing,
      borderRadius,
      boxShadow,
      transitionDuration: duration,
      transitionTimingFunction: easing,
      zIndex,
    },
  },
  plugins: [],
};

export default preset;
