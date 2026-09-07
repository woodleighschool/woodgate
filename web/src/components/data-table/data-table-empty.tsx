import type { ReactNode } from "react";

import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@components/ui/empty";

export function DataTableEmpty({
  icon,
  title,
  description,
  filtered = false,
  filteredTitle = "No Matches",
  filteredDescription,
}: {
  icon: ReactNode;
  title: string;
  description: string;
  filtered?: boolean;
  filteredTitle?: string;
  filteredDescription: string;
}) {
  return (
    <Empty className="min-h-72 border-0">
      <EmptyHeader>
        <EmptyMedia variant="icon">{icon}</EmptyMedia>
        <EmptyTitle>{filtered ? filteredTitle : title}</EmptyTitle>
        <EmptyDescription>{filtered ? filteredDescription : description}</EmptyDescription>
      </EmptyHeader>
    </Empty>
  );
}
