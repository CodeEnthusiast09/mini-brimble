import { useQuery } from "@tanstack/react-query";
import { getDeployments } from "../../api/deployments";
import { DeploymentCard } from "../DeploymentCard";

type Props = {
  selectedId: string | null;
  onSelect: (id: string) => void;
};

function SkeletonCard() {
  return (
    <div className="rounded-xl border border-white/7 bg-[#111111] p-4 animate-pulse">
      <div className="space-y-2.5">
        <div className="flex items-start justify-between gap-2">
          <div className="space-y-1.5">
            <div className="h-3.5 w-28 bg-white/7 rounded" />
            <div className="h-2.5 w-16 bg-white/5 rounded" />
          </div>
          <div className="h-4 w-14 bg-white/7 rounded" />
        </div>
        <div className="h-2.5 w-40 bg-white/5 rounded" />
      </div>
    </div>
  );
}

export function DeploymentList({ selectedId, onSelect }: Props) {
  const {
    data: deployments,
    isLoading,
    isError,
    error,
  } = useQuery({
    queryKey: ["deployments"],
    queryFn: getDeployments,
    refetchInterval: 5000,
  });

  if (isLoading) {
    return (
      <div className="space-y-2.5">
        <SkeletonCard />
        <SkeletonCard />
      </div>
    );
  }

  if (isError) {
    return <p className="text-red-400 text-sm">{error.message}</p>;
  }

  if (!deployments?.length) {
    return (
      <div className="rounded-xl border border-dashed border-white/6 py-12 flex items-center justify-center">
        <p className="text-gray-600 text-sm">No deployments yet</p>
      </div>
    );
  }

  return (
    <div className="space-y-2.5">
      {deployments.map((d) => (
        <DeploymentCard
          key={d.id}
          deployment={d}
          isSelected={selectedId === d.id}
          onSelect={onSelect}
        />
      ))}
    </div>
  );
}
