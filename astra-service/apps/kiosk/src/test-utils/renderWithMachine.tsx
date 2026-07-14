import { render } from "@testing-library/react";
import { KioskMachineProvider } from "../machines/KioskMachineProvider";

export function renderWithMachine(element: React.ReactElement): ReturnType<typeof render> {
  return render(
    <KioskMachineProvider>{element}</KioskMachineProvider>,
  );
}

