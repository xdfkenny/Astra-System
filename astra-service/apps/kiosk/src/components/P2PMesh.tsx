/*P2P mesh visualization for showing network topology and sync status.
Bottom sheet revealing connected nodes and data flow.
*/
import { motion, AnimatePresence } from "framer-motion";
import { motion as motionTokens } from "@astra/design-tokens";
import { cn } from "@/utils/cn";

export interface P2PMeshProps {
  open: boolean;
  onClose: () => void;
  "aria-label"?: string;
  className?: string;
}

// Mock data - replace with real P2P API data
const mockNodes = [
  { id: "node-1", x: 20, y: 50, status: "healthy", label: "Kiosk-1" },
  { id: "node-2", x: 60, y: 20, status: "syncing", label: "Kiosk-2" },
  { id: "node-3", x: 100, y: 60, status: "healthy", label: "Kiosk-3" },
  { id: "node-4", x: 140, y: 30, status: "offline", label: "Kiosk-4" },
  { id: "node-5", x: 180, y: 50, status: "connecting", label: "Kiosk-5" },
];

const connectionLines = [
  { from: "node-1", to: "node-2", active: true },
  { from: "node-1", to: "node-3", active: true },
  { from: "node-2", to: "node-4", active: false },
  { from: "node-3", to: "node-5", active: true },
  { from: "node-5", to: "node-4", active: true },
];

function getStatusColor(status: string): string {
  switch (status) {
    case "healthy": return "#5A7A5C"; // moss
    case "syncing": return "#B87E6B"; // amber
    case "connecting": return "#B87E6B"; // amber
    case "offline": return "#6B6862"; // stone
    default: return "#6B6862";
  }
}

export function P2PMesh({ open, onClose, "aria-label": ariaLabel = "P2P mesh topology", className }: P2PMeshProps) {
  const width = 260;
  const height = 120;

  return (
    <AnimatePresence>
      {open && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.2 }}
          className={cn(
            "fixed inset-0 z-50 flex items-center justify-center",
            "bg-charcoal/20 backdrop-blur-[4px]"
          )}
          onClick={onClose}
          role="dialog"
          aria-modal="true"
          aria-label={ariaLabel}
        >
          <motion.div
            initial={{ scale: 0.95, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            exit={{ scale: 0.95, opacity: 0 }}
            transition={{ duration: 0.25, ease: motionTokens.easeOutExpo }}
            className={cn(
              "w-full max-w-md rounded-[24px] bg-white p-6",
              "shadow-[0_8px_32px_rgba(45,42,38,0.15)]",
              "border border-taupe/20",
              className
            )}
            onClick={(e) => { e.stopPropagation(); }}
          >
            <div className="flex items-center justify-between mb-4">
              <h2 className="font-mono text-[18px] font-semibold text-charcoal">
                Mesh Topology
              </h2>
              <button
                type="button"
                onClick={onClose}
                className="h-10 w-10 rounded-full bg-linen border border-taupe"
                aria-label="Close mesh view"
              >
                <svg viewBox="0 0 20 20" className="h-5 w-5 text-charcoal" fill="none" stroke="currentColor" strokeWidth={2}>
                  <path d="M6 6l12 12M12 6l-12 12" strokeLinecap="round" />
                </svg>
              </button>
            </div>

            {/* Network visualization */}
            <div className="relative mx-auto" style={{ width, height }}>
              {/* SVG connections */}
              <svg
                width={width}
                height={height}
                className="absolute inset-0"
                aria-hidden="true"
              >
                {connectionLines.map((line, index) => {
                  const fromNode = mockNodes.find((n) => n.id === line.from);
                  const toNode = mockNodes.find((n) => n.id === line.to);
                  if (!fromNode || !toNode) return null;

                  return (
                    <motion.path
                      key={index}
                      d={`M ${fromNode.x} ${fromNode.y} L ${toNode.x} ${toNode.y}`}
                      stroke={line.active ? getStatusColor(fromNode.status) : "#E5E5E0"}
                      strokeWidth={line.active ? 2 : 1}
                      fill="none"
                      strokeDasharray={line.active ? "none" : "4 4"}
                      initial={{ pathLength: 0, opacity: 0 }}
                      animate={{ pathLength: 1, opacity: 1 }}
                      transition={{ delay: index * 0.1, duration: 0.5 }}
                    />
                  );
                })}
              </svg>

              {/* Nodes */}
              {mockNodes.map((node, index) => (
                <motion.div
                  key={node.id}
                  className="absolute flex flex-col items-center"
                  style={{ left: node.x - 20, top: node.y - 20 }}
                  initial={{ scale: 0, opacity: 0 }}
                  animate={{ scale: 1, opacity: 1 }}
                  transition={{ delay: index * 0.1, type: "spring", stiffness: 200 }}
                >
                  {/* Node dot */}
                  <motion.div
                    className="h-5 w-5 rounded-full border-2 border-white"
                    style={{ backgroundColor: getStatusColor(node.status) }}
                    whileHover={{ scale: 1.2 }}
                    whileTap={{ scale: 0.9 }}
                  />

                  {/* Label */}
                  <div className="mt-1 font-mono text-[10px] text-stone">
                    {node.label}
                  </div>

                  {/* Status dot */}
                  <div
                    className="mt-0.5 h-1.5 w-1.5 rounded-full"
                    style={{ backgroundColor: getStatusColor(node.status) }}
                  />
                </motion.div>
              ))}
            </div>

            {/* Legend */}
            <div className="mt-6 space-y-2">
              <h3 className="font-sans text-[14px] font-medium text-charcoal mb-2">
                Connection Status
              </h3>
              <div className="flex items-center gap-4">
                <div className="flex items-center gap-1">
                  <div className="h-3 w-3 rounded-full bg-moss" />
                  <span className="font-sans text-[12px] text-stone">Healthy</span>
                </div>
                <div className="flex items-center gap-1">
                  <div className="h-3 w-3 rounded-full bg-amber" />
                  <span className="font-sans text-[12px] text-stone">Syncing</span>
                </div>
                <div className="flex items-center gap-1">
                  <div className="h-3 w-3 rounded-full bg-stone" />
                  <span className="font-sans text-[12px] text-stone">Offline</span>
                </div>
              </div>
            </div>

            {/* Live data toggle */}
            <div className="mt-6 flex items-center justify-between">
              <span className="font-sans text-[12px] text-stone">
                Data refreshed 2s ago
              </span>
              <button
                type="button"
                className="rounded-full bg-moss/10 px-3 py-1 font-sans text-[12px] font-medium text-moss hover:bg-moss/20"
              >
                Refresh
              </button>
            </div>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}
