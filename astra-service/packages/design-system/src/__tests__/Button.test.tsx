import { describe, it, expect, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { Button } from "../components/Button";

describe("Button", () => {
  it("renders a primary button", () => {
    render(<Button>Pay</Button>);
    expect(screen.getByRole("button", { name: "Pay" })).toBeTruthy();
  });

  it("forwards ref", () => {
    let node: HTMLButtonElement | null = null;
    render(<Button ref={(el) => { node = el; }}>Test</Button>);
    expect(node).toBeInstanceOf(HTMLButtonElement);
  });

  it("triggers haptic feedback and click handler", () => {
    const vibrate = vi.fn();
    Object.assign(navigator, { vibrate });
    const onClick = vi.fn();

    render(<Button onClick={onClick}>Tap</Button>);
    fireEvent.click(screen.getByRole("button", { name: "Tap" }));

    expect(vibrate).toHaveBeenCalledWith([20]);
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it("is disabled when loading", () => {
    render(<Button loading>Busy</Button>);
    const button = screen.getByRole("button", { name: "Busy" });
    expect(button.hasAttribute("disabled")).toBe(true);
  });
});
