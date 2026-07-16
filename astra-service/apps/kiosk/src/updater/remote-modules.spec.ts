import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  RemoteModuleManager,
  versionedRemoteUrl,
  type RemoteDefinition,
} from "./remote-modules";

describe("RemoteModuleManager", () => {
  let manager: RemoteModuleManager;
  const onVersionChange = vi.fn();

  beforeEach(() => {
    manager = new RemoteModuleManager({ onVersionChange });
  });

  it("registers and retrieves a remote", () => {
    const def: RemoteDefinition = {
      name: "astra_menu",
      baseUrl: "https://cdn.example.com/menu/v1.0.0",
      version: "1.0.0",
      timeoutMs: 5000,
      fallbackUrl: "https://cdn.example.com/menu/v1.0.0",
    };
    manager.register(def);
    const state = manager.get("astra_menu");
    expect(state).toBeDefined();
    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
    expect(state!.definition.version).toBe("1.0.0");
    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
    expect(state!.status).toBe("mounting");
  });

  it("lists all registered remotes", () => {
    manager.register({
      name: "menu",
      baseUrl: "https://cdn.example.com/menu/v1",
      version: "1.0.0",
      timeoutMs: 5000,
      fallbackUrl: "https://cdn.example.com/menu/v1",
    });
    manager.register({
      name: "cart",
      baseUrl: "https://cdn.example.com/cart/v1",
      version: "1.0.0",
      timeoutMs: 5000,
      fallbackUrl: "https://cdn.example.com/cart/v1",
    });
    expect(manager.list()).toHaveLength(2);
  });

  it("returns already-on-version when versions match", async () => {
    const def: RemoteDefinition = {
      name: "menu",
      baseUrl: "https://cdn.example.com/menu/v1",
      version: "1.0.0",
      timeoutMs: 5000,
      fallbackUrl: "https://cdn.example.com/menu/v1",
    };
    manager.register(def);
    const result = await manager.swapVersion("menu", "https://cdn.example.com/menu/v1", "1.0.0");
    expect(result).toBe(true);
  });

  it("throws for unknown remote", async () => {
    await expect(
      manager.swapVersion("nonexistent", "https://example.com", "2.0.0"),
    ).rejects.toThrow("unknown remote: nonexistent");
  });

  it("rollbackAll resets all remotes to fallback", () => {
    manager.register({
      name: "menu",
      baseUrl: "https://cdn.example.com/menu/v2",
      version: "2.0.0",
      timeoutMs: 5000,
      fallbackUrl: "https://cdn.example.com/menu/v1",
    });
    manager.register({
      name: "cart",
      baseUrl: "https://cdn.example.com/cart/v2",
      version: "2.0.0",
      timeoutMs: 5000,
      fallbackUrl: "https://cdn.example.com/cart/v1",
    });
    manager.rollbackAll();
    const states = manager.list();
    for (const s of states) {
      expect(s.status).toBe("rolled_back");
      expect(s.definition.baseUrl).toBe(s.definition.fallbackUrl);
    }
  });
});

describe("versionedRemoteUrl", () => {
  it("appends version path to base URL", () => {
    expect(versionedRemoteUrl("https://cdn.example.com/menu", "1.2.3")).toBe(
      "https://cdn.example.com/menu/v1.2.3",
    );
  });

  it("strips trailing slash from base URL", () => {
    expect(versionedRemoteUrl("https://cdn.example.com/menu/", "2.0.0")).toBe(
      "https://cdn.example.com/menu/v2.0.0",
    );
  });

  it("strips leading v from version", () => {
    expect(versionedRemoteUrl("https://cdn.example.com/cart", "v3.0.0")).toBe(
      "https://cdn.example.com/cart/v3.0.0",
    );
  });
});
