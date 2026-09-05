import { useParams } from "@tanstack/react-router";
import { Pencil } from "lucide-react";

import { EnumBadge } from "@components/enum-badge";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link, TextLink } from "@components/link";
import { QueryGate } from "@components/query-gate";
import { TokenList } from "@components/token-list";
import { Button } from "@components/ui/button";
import { useCan } from "@features/authz/access";
import { useGroup } from "@features/directory/groups/queries";
import { DIRECTORY_SOURCES } from "@features/directory/source";
import { parseRouteID } from "@lib/route-params";
import { countLabel, nonEmpty } from "@lib/utils";

export function GroupDetailPage() {
  const { id: rawID } = useParams({ from: "/_authenticated/directory/groups/$id" });
  const id = parseRouteID(rawID);
  const query = useGroup(id);
  const canEditGroups = useCan({ resource: "groups", access: "edit" });
  const canEditRoles = useCan({ resource: "authz.roles", access: "edit" });
  if (id === null)
    return <QueryGate title="Failed to Load Group" error={{ message: "Invalid group." }} />;
  if (query.error || !query.data) {
    return (
      <QueryGate
        title="Failed to Load Group"
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    );
  }
  const group = query.data;
  return (
    <PageShell>
      <PageHeader
        title={group.display_name}
        context={<EnumBadge value={group.source} metadata={DIRECTORY_SOURCES} />}
        actions={
          canEditGroups && canEditRoles ? (
            <Button
              size="sm"
              variant="outline"
              render={<Link to="/directory/groups/$id/edit" params={{ id: String(group.id) }} />}
              nativeButton={false}
            >
              <Pencil data-icon="inline-start" />
              Edit
            </Button>
          ) : null
        }
      />
      <KeyValueSection title="Group">
        <KeyValueRow label="Nickname" value={nonEmpty(group.mail_nickname) ?? "-"} />
        <KeyValueRow
          label="Members"
          value={
            <TextLink to="/directory/users" search={{ group_id: group.id }}>
              {countLabel(group.member_count, "member")}
            </TextLink>
          }
        />
        <KeyValueRow
          label="Roles"
          value={<TokenList values={group.roles.map((role) => role.name)} />}
        />
      </KeyValueSection>
    </PageShell>
  );
}
