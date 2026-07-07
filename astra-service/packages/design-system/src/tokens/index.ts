export * from "./colors";
export * from "./spacing";
export * from "./typography";
export * from "./elevation";
export * from "./motion";
export * from "./z-index";

import { cssVariables as colorVariables } from "./colors";
import { cssVariables as spacingVariables } from "./spacing";
import { cssVariables as typographyVariables } from "./typography";
import { cssVariables as elevationVariables } from "./elevation";
import { cssVariables as motionVariables } from "./motion";
import { cssVariables as zIndexVariables } from "./z-index";

export const cssVariables: Record<string, string> = {
  ...colorVariables,
  ...spacingVariables,
  ...typographyVariables,
  ...elevationVariables,
  ...motionVariables,
  ...zIndexVariables,
};
