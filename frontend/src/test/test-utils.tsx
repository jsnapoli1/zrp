import React from "react";
import { render, type RenderOptions } from "@testing-library/react";
import { BrowserRouter } from "react-router-dom";
import { WebSocketProvider } from "../contexts/WebSocketContext";

function AllProviders({ children }: { children: React.ReactNode }) {
  return (
    <BrowserRouter>
      <WebSocketProvider>{children}</WebSocketProvider>
    </BrowserRouter>
  );
}

const customRender = (
  ui: React.ReactElement,
  options?: Omit<RenderOptions, "wrapper">
) => render(ui, { wrapper: AllProviders, ...options });

export * from "@testing-library/react";
export { customRender as render };
