import { useNavigate, useParams } from "@tanstack/react-router";
import { Pencil, Trash2 } from "lucide-react";
import { useState } from "react";

import { EnumBadge } from "@components/enum-badge";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { QueryGate } from "@components/query-gate";
import { TokenList } from "@components/token-list";
import { Button } from "@components/ui/button";
import { useAccount } from "@features/account/queries";
import { useCan } from "@features/authz/access";
import { DIRECTORY_SOURCES } from "@features/directory/source";
import { UserDeleteDialog } from "@features/directory/users/delete-dialog";
import { EffectiveRoles } from "@features/directory/users/effective-roles";
import { useUser } from "@features/directory/users/queries";
import { parseRouteID } from "@lib/route-params";
import { nonEmpty } from "@lib/utils";

export function UserDetailPage() {
  const { id: userID } = useParams({
    from: "/_authenticated/directory/users/$id",
  });
  const navigate = useNavigate();
  const account = useAccount();
  const canEditUsers = useCan({ resource: "users", access: "edit" });
  const canEditRoles = useCan({ resource: "authz.roles", access: "edit" });
  const currentUser = account.data?.user;
  const id = parseRouteID(userID);
  const query = useUser(id);
  const [deleteOpen, setDeleteOpen] = useState(false);

  if (id === null) {
    return <QueryGate title="Failed to Load User" error={{ message: "User route is invalid." }} />;
  }
  if (query.error || !query.data) {
    return (
      <QueryGate
        title="Failed to Load User"
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    );
  }

  const user = query.data;
  const isSelf = currentUser?.id === user.id;
  const canManageUsers = canEditUsers && canEditRoles;
  const editLink = isSelf
    ? ({ to: "/account" } as const)
    : ({
        to: "/directory/users/$id/edit",
        params: { id: String(user.id) },
      } as const);

  return (
    <>
      <PageShell>
        <PageHeader
          title="User Details"
          actions={
            isSelf || canManageUsers ? (
              <>
                <Button size="sm" render={<Link {...editLink} />} nativeButton={false}>
                  <Pencil data-icon="inline-start" />
                  Edit
                </Button>
                {canManageUsers && !isSelf ? (
                  <Button
                    type="button"
                    variant="destructive"
                    size="sm"
                    onClick={() => setDeleteOpen(true)}
                  >
                    <Trash2 data-icon="inline-start" />
                    Delete
                  </Button>
                ) : null}
              </>
            ) : null
          }
        />

        <KeyValueSection title="Overview">
          <KeyValueRow label="Name" value={nonEmpty(user.name) ?? "-"} />
          <KeyValueRow label="Email" value={user.email} />
          <KeyValueRow
            label="Source"
            value={<EnumBadge value={user.source} metadata={DIRECTORY_SOURCES} />}
          />
          <KeyValueRow
            label="Direct Roles"
            value={<TokenList values={user.roles.map((role) => role.name)} />}
          />
          <KeyValueRow
            label="Effective Access"
            value={<EffectiveRoles roles={user.effective_roles} />}
          />
          <KeyValueRow label="Department" value={nonEmpty(user.department) ?? "-"} />
          <KeyValueRow label="Given Name" value={nonEmpty(user.given_name) ?? "-"} />
          <KeyValueRow label="Family Name" value={nonEmpty(user.family_name) ?? "-"} />
          <KeyValueRow
            label="User Principal Name"
            value={nonEmpty(user.user_principal_name) ?? "-"}
          />
          <KeyValueRow label="External ID" value={nonEmpty(user.external_id) ?? "-"} />
        </KeyValueSection>
      </PageShell>

      <UserDeleteDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        user={user}
        onDeleted={() => void navigate({ to: "/directory/users" })}
      />
    </>
  );
}
