import { createContext, useContext } from "react";
import { useMachine } from "@xstate/react";
import type { ActorRefFrom, SnapshotFrom } from "xstate";
import { kioskMachine, type KioskEvent } from "./kioskMachine";

export type KioskActorRef = ActorRefFrom<typeof kioskMachine>;
export type KioskSnapshot = SnapshotFrom<typeof kioskMachine>;

interface KioskMachineContextValue {
  readonly state: KioskSnapshot;
  readonly send: KioskActorRef["send"];
  readonly actorRef: KioskActorRef;
}

const KioskMachineContext = createContext<KioskMachineContextValue | null>(null);

export function KioskMachineProvider({
  children,
}: {
  readonly children: React.ReactNode;
}): React.JSX.Element {
  const [state, send, actorRef] = useMachine(kioskMachine);

  return (
    <KioskMachineContext.Provider value={{ state, send, actorRef }}>
      {children}
    </KioskMachineContext.Provider>
  );
}

// eslint-disable-next-line react-refresh/only-export-components -- hook is colocated with its provider; fast-refresh is not critical for this state root
export function useKioskMachine(): KioskMachineContextValue {
  const ctx = useContext(KioskMachineContext);
  if (!ctx) {
    throw new Error("useKioskMachine must be used within a KioskMachineProvider");
  }
  return ctx;
}

export type { KioskEvent };

