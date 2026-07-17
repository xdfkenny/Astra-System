import { describe, expect, it } from "vitest";
import { screen, fireEvent, waitFor } from "@testing-library/react";
import { renderWithMachine } from "../test-utils/renderWithMachine";
import { LanguageSelectScreen } from "./LanguageSelectScreen";
import { useKioskMachine } from "../machines/KioskMachineProvider";

function StateReader(): React.JSX.Element {
  const { state } = useKioskMachine();
  return <div data-testid="stage">{state.value as string}</div>;
}

describe("LanguageSelectScreen", () => {
  it("renders all main languages", () => {
    renderWithMachine(
      <>
        <LanguageSelectScreen />
        <StateReader />
      </>,
    );

    expect(screen.getAllByText("English").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("Español")).toBeTruthy();
    expect(screen.getByText("简体中文")).toBeTruthy();
    expect(screen.getByText("Français")).toBeTruthy();
  });

  it("transitions to ATTRACT with Spanish locale when Español is selected", async () => {
    const { getByTestId } = renderWithMachine(
      <>
        <LanguageSelectScreen />
        <StateReader />
      </>,
    );

    expect(getByTestId("stage").textContent).toBe("LANGUAGE_SELECT");

    const spanishBtn = screen.getByText("Español");
    fireEvent.click(spanishBtn);

    await waitFor(() => {
      expect(getByTestId("stage").textContent).toBe("ATTRACT");
    });
  });
});
