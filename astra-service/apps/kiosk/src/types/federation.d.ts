/**
 * Ambient module declarations for Native Federation remotes consumed by the
 * unified kiosk host. Each remote publishes its default React component so the
 * shell can lazy-load it at runtime.
 */

declare module "astra_menu/MenuApp" {
  import type React from "react";
  import type { MenuItem } from "@astra/shared-types";

  export interface MenuAppProps {
    readonly laneMode: "express" | "full";
    readonly silentAssistArmed: boolean;
    readonly onSelectItem: (item: MenuItem) => void;
  }

  const MenuApp: (props: MenuAppProps) => React.JSX.Element;
  export default MenuApp;
}

declare module "astra_cart/CartApp" {
  import type React from "react";

  export interface CartAppProps {
    readonly onBackToMenu: () => void;
    readonly onProceedToPayment: () => void;
  }

  const CartApp: (props: CartAppProps) => React.JSX.Element;
  export default CartApp;
}

declare module "astra_payment/PaymentApp" {
  import type React from "react";
  import type { PaymentAuthorizationResult } from "@astra/shared-types";

  export interface PaymentAppProps {
    readonly onResult: (result: PaymentAuthorizationResult) => void;
    readonly onCancel: () => void;
  }

  const PaymentApp: (props: PaymentAppProps) => React.JSX.Element;
  export default PaymentApp;
}

declare module "astra_kiosk/Shell" {
  import type React from "react";

  const Shell: () => React.JSX.Element;
  export default Shell;
}
