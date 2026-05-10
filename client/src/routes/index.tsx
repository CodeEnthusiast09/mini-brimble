import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { getDeployments } from "../api/deployments";
import { DeploymentList } from "../components/DeploymentList";
import { LogViewer } from "../components/LogViewer";
import { DeployModal } from "../components/DeployModal";

export const Route = createFileRoute("/")({
  component: Index,
});

function Index() {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [modalOpen, setModalOpen] = useState(false);

  const { data: deployments } = useQuery({
    queryKey: ["deployments"],
    queryFn: getDeployments,
    refetchInterval: 5000,
  });

  const selectedDeployment =
    deployments?.find((d) => d.id === selectedId) ?? null;

  const handleSelect = (id: string) => {
    setSelectedId((prev) => (prev === id ? null : id));
  };

  return (
    <div className="max-w-7xl mx-auto px-6 py-8 space-y-8">
      {/* Page header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">
            mini-brimble{" "}
            <span className="font-light text-gray-500">Pipeline</span>
          </h1>
          <p className="text-sm text-gray-600 mt-0.5">
            Deploy your apps with zero configuration.
          </p>
        </div>
        <button
          onClick={() => setModalOpen(true)}
          className="flex items-center gap-2 rounded-lg bg-white text-black px-4 py-2 text-sm font-medium hover:bg-gray-100 transition-colors"
        >
          <svg
            width="11"
            height="11"
            viewBox="0 0 11 11"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
          >
            <path d="M5.5 1v9M1 5.5h9" />
          </svg>
          New Deployment
        </button>
      </div>

      {/* Main grid */}
      <div className="grid grid-cols-1 lg:grid-cols-[2fr_3fr] gap-6 items-start">
        <section>
          <h2 className="flex items-center gap-2 text-[11px] font-semibold text-gray-600 uppercase tracking-widest mb-3">
            <svg
              width="11"
              height="11"
              viewBox="0 0 11 11"
              fill="currentColor"
              className="text-blue-400 shrink-0"
            >
              <path d="M5.5 0L10 8H1L5.5 0Z" />
            </svg>
            Deployments
          </h2>
          <DeploymentList selectedId={selectedId} onSelect={handleSelect} />
        </section>

        <section>
          <h2 className="flex items-center gap-2 text-[11px] font-semibold text-gray-600 uppercase tracking-widest mb-3">
            <span className="text-blue-400 font-mono text-xs leading-none">
              &gt;_
            </span>
            Live Logs
          </h2>
          {selectedDeployment ? (
            <LogViewer deployment={selectedDeployment} />
          ) : (
            <div className="rounded-xl border border-dashed border-white/6 min-h-64 flex items-center justify-center">
              <p className="text-gray-600 text-sm">
                Select a deployment to view logs
              </p>
            </div>
          )}
        </section>
      </div>

      {modalOpen && <DeployModal onClose={() => setModalOpen(false)} />}
    </div>
  );
}
