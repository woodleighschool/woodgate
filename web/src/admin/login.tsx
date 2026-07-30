import { listAuthProviders, type AuthProviders } from "@/api/auth";
import MicrosoftIcon from "@mui/icons-material/Microsoft";
import { Stack, Typography } from "@mui/material";
import { useEffect, useState, type ReactElement } from "react";
import { Button, Login, LoginForm } from "react-admin";

export const LoginPage = (): ReactElement => {
  const [providers, setProviders] = useState<AuthProviders | undefined>();

  useEffect(() => {
    const controller = new AbortController();

    listAuthProviders(controller.signal)
      .then((result: AuthProviders): void => {
        setProviders(result);
      })
      .catch((error: unknown): void => {
        if ((error as { name?: string }).name === "AbortError") {
          return;
        }
        setProviders(undefined);
      });

    return (): void => {
      controller.abort();
    };
  }, []);

  const origin = globalThis.location.origin;
  const parameters = new URLSearchParams({ site: origin, from: origin });
  const oauthLoginHref = `/auth/microsoft/login?${parameters.toString()}`;

  const showMicrosoft = providers ? providers.microsoft : true;
  const showLocal = providers ? providers.local : true;

  return (
    <Login>
      <Stack spacing={2} sx={{ px: 3, py: 3 }}>
        <Stack spacing={0.5}>
          <Typography variant="h5" fontWeight={600}>
            Sign In
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Use a Microsoft account or a local account.
          </Typography>
        </Stack>

        {showMicrosoft ? (
          <Button
            component="a"
            href={oauthLoginHref}
            variant="contained"
            size="large"
            fullWidth
            label="Continue With Microsoft"
            startIcon={<MicrosoftIcon />}
          />
        ) : undefined}

        {showLocal ? <LoginForm /> : undefined}
      </Stack>
    </Login>
  );
};
