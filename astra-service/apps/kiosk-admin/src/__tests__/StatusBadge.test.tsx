import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { StatusBadge } from "../components/StatusBadge";

describe("StatusBadge", () => {
  it("renders the provided status label", () => {
    render(<StatusBadge status="healthy" />);
    expect(screen.getByText("healthy")).toBeInTheDocument();
  });

  it("uses the custom children label when provided", () => {
    render(<StatusBadge status="open">Circuit Open</StatusBadge>);
    expect(screen.getByText("Circuit Open")).toBeInTheDocument();
  });

  it("capitalizes the default status text", () => {
    render(<StatusBadge status="circuit_open" />);
    expect(screen.getByText("circuit open")).toBeInTheDocument();
  });
});
