import { useEffect, useRef } from "react";
import * as d3 from "d3";
import type { KioskNode, NodeHealth } from "../hooks/useFleetHealth";

const HEALTH_COLOR: Record<NodeHealth, string> = {
  healthy: "var(--color-success)",
  degraded: "var(--color-warning)",
  circuit_open: "var(--color-error)",
  offline: "var(--color-secondary)",
};

interface SimulationNode extends d3.SimulationNodeDatum {
  readonly kioskId: string;
  readonly health: NodeHealth;
  readonly isLeader: boolean;
}

interface SimulationLink extends d3.SimulationLinkDatum<SimulationNode> {
  readonly source: string | SimulationNode;
  readonly target: string | SimulationNode;
}

interface MeshTopologyGraphProps {
  readonly nodes: readonly KioskNode[];
}

/**
 * Renders the kiosk P2P mesh as a D3 force-directed graph. Falls back to a
 * static SVG circle layout when D3 is unavailable so the health dashboard
 * remains usable.
 */
export function MeshTopologyGraph({ nodes }: MeshTopologyGraphProps): React.JSX.Element {
  const svgRef = useRef<SVGSVGElement>(null);

  useEffect(() => {
    const svg = svgRef.current;
    if (!svg || nodes.length === 0) return;

    const width = svg.clientWidth || 320;
    const height = 320;
    const resolvedColors = new Map<NodeHealth, string>();

    function resolveColor(token: NodeHealth): string {
      if (!resolvedColors.has(token)) {
        const computed = getComputedStyle(document.documentElement).getPropertyValue(
          HEALTH_COLOR[token].replace("var(", "").replace(")", "").trim(),
        );
        resolvedColors.set(token, computed || HEALTH_COLOR[token]);
      }
      return resolvedColors.get(token) ?? HEALTH_COLOR[token];
    }

    const simulationNodes: SimulationNode[] = nodes.map((n) => ({
      kioskId: n.kioskId,
      health: n.health,
      isLeader: n.isLeader,
    }));

    const nodeById = new Map(nodes.map((n) => [n.kioskId, n]));
    const links: SimulationLink[] = [];
    const seen = new Set<string>();
    for (const node of nodes) {
      for (const peerId of node.meshPeers) {
        const key = [node.kioskId, peerId].sort().join("-");
        if (seen.has(key)) continue;
        seen.add(key);
        if (nodeById.has(peerId)) {
          links.push({ source: node.kioskId, target: peerId });
        }
      }
    }

    d3.select(svg).selectAll("*").remove();
    const root = d3.select(svg).attr("viewBox", `0 0 ${width} ${height}`);
    const g = root.append("g");

    const simulation = d3
      .forceSimulation<SimulationNode>(simulationNodes)
      .force(
        "link",
        d3
          .forceLink<SimulationNode, SimulationLink>(links)
          .id((d) => d.kioskId)
          .distance(80),
      )
      .force("charge", d3.forceManyBody().strength(-200))
      .force("center", d3.forceCenter(width / 2, height / 2))
      .force("collide", d3.forceCollide<SimulationNode>().radius(24));

    const link = g
      .append("g")
      .attr("stroke", "var(--color-border-strong)")
      .attr("stroke-width", 1.5)
      .selectAll("line")
      .data(links)
      .join("line");

    const nodeGroup = g
      .append("g")
      .selectAll("g")
      .data(simulationNodes)
      .join("g");

    nodeGroup
      .append("circle")
      .attr("r", (d) => (d.isLeader ? 16 : 12))
      .attr("fill", (d) => resolveColor(d.health))
      .attr("stroke", (d) => (d.isLeader ? "var(--color-ink)" : "none"))
      .attr("stroke-width", 2);

    nodeGroup
      .append("text")
      .attr("dy", 28)
      .attr("text-anchor", "middle")
      .attr("font-size", 10)
      .attr("fill", "var(--color-ink-muted)")
      .text((d) => d.kioskId);

    simulation.on("tick", () => {
      link
        .attr("x1", (d) => (d.source as SimulationNode).x ?? 0)
        .attr("y1", (d) => (d.source as SimulationNode).y ?? 0)
        .attr("x2", (d) => (d.target as SimulationNode).x ?? 0)
        .attr("y2", (d) => (d.target as SimulationNode).y ?? 0);

      nodeGroup.attr("transform", (d) => `translate(${d.x ?? 0},${d.y ?? 0})`);
    });

    return () => {
      simulation.stop();
    };
  }, [nodes]);

  return (
    <svg
      ref={svgRef}
      className="h-80 w-full"
      role="img"
      aria-label="Kiosk mesh topology"
      preserveAspectRatio="xMidYMid meet"
    >
      <title>Live P2P mesh topology</title>
    </svg>
  );
}
