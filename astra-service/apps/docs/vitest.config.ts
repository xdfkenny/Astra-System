import base from "@astra/config/vitest";
import { defineConfig, mergeConfig } from "vitest/config";

export default mergeConfig(base, defineConfig({
  test: {
    environment: "node",
  },
}));
