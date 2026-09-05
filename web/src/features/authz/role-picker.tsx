import { Button } from "@components/ui/button";
import { Checkbox } from "@components/ui/checkbox";
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldError,
  FieldLabel,
  FieldLegend,
  FieldSet,
  FieldTitle,
} from "@components/ui/field";
import { Spinner } from "@components/ui/spinner";
import { useAuthzRoles } from "@features/resources/queries";

export function RolePicker({
  value,
  onChange,
  description = "A user with no effective role has No Access.",
}: {
  value: number[];
  onChange: (value: number[]) => void;
  description?: string;
}) {
  const roles = useAuthzRoles();
  return (
    <FieldSet>
      <FieldLegend variant="label">Roles</FieldLegend>
      <FieldDescription>{description}</FieldDescription>
      {roles.isLoading ? (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Spinner /> Loading roles
        </div>
      ) : roles.error ? (
        <FieldError>
          <span>Could not load roles.</span>{" "}
          <Button
            type="button"
            variant="link"
            className="h-auto p-0"
            onClick={() => void roles.refetch()}
          >
            Try again
          </Button>
        </FieldError>
      ) : (
        <div data-slot="checkbox-group" className="grid gap-2 md:grid-cols-2">
          {roles.data?.map((role) => {
            const checked = value.includes(role.id);
            return (
              <FieldLabel key={role.id} htmlFor={`role-${role.id}`}>
                <Field orientation="horizontal">
                  <Checkbox
                    id={`role-${role.id}`}
                    checked={checked}
                    onCheckedChange={(next) =>
                      onChange(
                        next
                          ? [...value, role.id]
                          : value.filter((candidate) => candidate !== role.id),
                      )
                    }
                  />
                  <FieldContent>
                    <FieldTitle>{role.name}</FieldTitle>
                    {role.description ? (
                      <FieldDescription>{role.description}</FieldDescription>
                    ) : null}
                  </FieldContent>
                </Field>
              </FieldLabel>
            );
          })}
        </div>
      )}
    </FieldSet>
  );
}
