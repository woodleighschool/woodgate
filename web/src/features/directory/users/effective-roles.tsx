import { EnumBadge } from "@components/enum-badge";
import { TokenList } from "@components/token-list";
import type { RoleSummary } from "@lib/api";
import type { EnumMetadataMap } from "@lib/enum-metadata";

const NO_ACCESS = {
  none: {
    name: "No Access",
    description: "Cannot sign in until a role is assigned.",
  },
} satisfies EnumMetadataMap<"none">;

export function EffectiveRoles({ roles }: { roles: readonly RoleSummary[] }) {
  return (
    <TokenList
      values={roles.map((role) => role.name)}
      empty={<EnumBadge value="none" metadata={NO_ACCESS} />}
    />
  );
}
