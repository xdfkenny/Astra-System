import { describe, expect, it, vi } from "vitest";
import { screen, fireEvent, waitFor } from "@testing-library/react";
import { renderWithMachine } from "../test-utils/renderWithMachine";
import { AttractScreen } from "../routes/AttractScreen";
import { useKioskMachine } from "../machines/KioskMachineProvider";
import { useEffect } from "react";

function StateReader(): React.JSX.Element {
  const { state } = useKioskMachine();
  return <div data-testid="stage">{state.value as string}</div>;
}

function LanguageSelector(): null {
  const { send } = useKioskMachine();
  useEffect(() => {
    send({ type: "SET_LANGUAGE", locale: "en" });
  }, [send]);
  return null;
}

describe("AttractScreen", () => {
  it("starts a session when the tap-to-start button is pressed", async () => {
    const mockCrypto = {
      randomUUID: () => "uuid-test",
      getRandomValues: (arr: Uint8Array) => arr,
    } as unknown as Crypto;
    const spy = vi.spyOn(window, "crypto", "get").mockReturnValue(mockCrypto);
    const { getByTestId } = renderWithMachine(
      <>
        <LanguageSelector />
        <AttractScreen />
        <StateReader />
      </>,
    );

    await waitFor(
      () => {
        expect(getByTestId("stage").textContent).toBe("ATTRACT");
      },
      { timeout: 1000 },
    );

    const button = screen.getByRole("button", { name: "Attract screen. Touch to begin shopping." });
    fireEvent.click(button);

    await waitFor(
      () => {
        expect(getByTestId("stage").textContent).toBe("MENU");
      },
      { timeout: 2000 },
    );
    spy.mockRestore();
  });
});
