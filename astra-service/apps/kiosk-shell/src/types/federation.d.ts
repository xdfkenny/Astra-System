// Ambient module declarations for Module Federation remotes. Each remote
// publishes its own richer types via a generated .d.ts synced through the
// `packages/shared-types` build step in CI; this file is the fallback
// contract consumed at typecheck time in this repo.
declare module "astra_menu/MenuApp" {
  export interface MenuAppProps {
    laneMode: "express" | "full";
    silentAssistArmed: boolean;
  }
  const MenuApp: (props: MenuAppProps) => React.JSX.Element;
  export default MenuApp;
}

declare module "astra_cart/CartApp" {
  export interface CartAppProps {
    onBackToMenu: () => void;
    onProceedToPayment: () => void;
  }
  const CartApp: (props: CartAppProps) => React.JSX.Element;
  export default CartApp;
}

declare module "astra_payment/PaymentApp" {
  import type { PaymentAuthorizationResult } from "@astra/shared-types";
  export interface PaymentAppProps {
    onResult: (result: PaymentAuthorizationResult) => void;
    onCancel: () => void;
  }
  const PaymentApp: (props: PaymentAppProps) => React.JSX.Element;
  export default PaymentApp;
}

declare module "astra_admin/AdminApp" {
  const AdminApp: () => React.JSX.Element;
  export default AdminApp;
}
