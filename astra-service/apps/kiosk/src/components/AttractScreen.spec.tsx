import { describe, expect, it, vi } from "vitest";
import { screen, fireEvent } from "@testing-library/react";
import { renderWithMachine } from "../test-utils/renderWithMachine";
import { AttractScreen } from "../routes/AttractScreen";
import { useKioskMachine } from "../machines/KioskMachineProvider";

function StateReader(): React.JSX.Element {
  const { state } = useKioskMachine();
  return <div data-testid="stage">{state.value as string}</div>;
}

describe("AttractScreen", () => {
  it("starts a session when the tap-to-start button is pressed", () => {
    const mockCrypto = {
      randomUUID: () => "uuid-test",
      getRandomValues: (arr: Uint8Array) => arr,
    } as unknown as Crypto;
    const spy = vi.spyOn(window, "crypto", "get").mockReturnValue(mockCrypto);
    const { getByTestId } = renderWithMachine(
      <>
        <AttractScreen />
        <StateReader />
      </>,
    );

    const button = screen.getByLabelText("Tap to start shopping");
    fireEvent.click(button);

    expect(getByTestId("stage").textContent).toBe("MENU_BROWSE");
    spy.mockRestore();
  });
});
