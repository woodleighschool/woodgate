import type { ThemeOptions } from "@mui/material/styles";
import { deepmerge } from "@mui/utils";
import { defaultDarkTheme, defaultLightTheme } from "react-admin";

const sharedOverrides: ThemeOptions = {
  shape: {
    borderRadius: 10,
  },
  typography: {
    fontFamily: [
      "Inter",
      "ui-sans-serif",
      "system-ui",
      "-apple-system",
      "BlinkMacSystemFont",
      '"Segoe UI"',
      "sans-serif",
    ].join(", "),
    button: {
      textTransform: "none",
      fontWeight: 600,
    },
  },
  components: {
    MuiButtonBase: {
      defaultProps: {
        disableRipple: true,
      },
    },
    MuiButton: {
      defaultProps: {
        variant: "contained",
      },
      styleOverrides: {
        root: {
          borderRadius: 999,
          boxShadow: "none",
        },
        contained: {
          boxShadow: "none",
        },
      },
    },
    MuiPaper: {
      styleOverrides: {
        root: {
          backgroundImage: "none",
        },
        rounded: {
          borderRadius: 14,
        },
      },
    },
    MuiCard: {
      styleOverrides: {
        root: {
          backgroundImage: "none",
        },
      },
    },
    MuiDrawer: {
      styleOverrides: {
        paper: {
          backgroundImage: "none",
        },
      },
    },
    MuiAppBar: {
      styleOverrides: {
        colorSecondary: {
          backgroundImage: "none",
          backdropFilter: "blur(8px)",
        },
      },
    },
    MuiFilledInput: {
      styleOverrides: {
        root: {
          borderRadius: 10,
        },
      },
    },
    MuiOutlinedInput: {
      styleOverrides: {
        root: {
          borderRadius: 10,
        },
      },
    },
    MuiTextField: {
      defaultProps: {
        variant: "outlined",
      },
    },
    MuiChip: {
      styleOverrides: {
        root: {
          borderRadius: 999,
          fontWeight: 600,
        },
      },
    },
    MuiTableCell: {
      styleOverrides: {
        head: {
          fontWeight: 700,
        },
      },
    },
  },
};

export const lightTheme = deepmerge(defaultLightTheme, {
  ...sharedOverrides,
  palette: {
    mode: "light",
    primary: {
      main: "#2F5D7E",
      light: "#4A789A",
      dark: "#21425A",
      contrastText: "#FFFFFF",
    },
    secondary: {
      main: "#B9843D",
      light: "#C99B5F",
      dark: "#8A622B",
      contrastText: "#1F1608",
    },
    success: {
      main: "#4A7F6A",
    },
    warning: {
      main: "#C18A3F",
    },
    error: {
      main: "#B35A4C",
    },
    info: {
      main: "#5C7FA6",
    },
    background: {
      default: "#F3F1EB",
      paper: "#FCFAF5",
    },
    divider: "#D6D0C3",
    text: {
      primary: "#1F2E38",
      secondary: "#5F6B72",
    },
  },
  components: {
    MuiAppBar: {
      styleOverrides: {
        colorSecondary: {
          backgroundColor: "rgba(252, 250, 245, 0.92)",
          color: "#1F2E38",
          borderBottom: "1px solid rgba(47, 93, 126, 0.14)",
        },
      },
    },
    MuiPaper: {
      styleOverrides: {
        root: {
          border: "1px solid rgba(47, 93, 126, 0.08)",
        },
      },
    },
    MuiOutlinedInput: {
      styleOverrides: {
        notchedOutline: {
          borderColor: "rgba(47, 93, 126, 0.18)",
        },
      },
    },
  },
} satisfies ThemeOptions);

export const darkTheme = deepmerge(defaultDarkTheme, {
  ...sharedOverrides,
  palette: {
    mode: "dark",
    primary: {
      main: "#7FA9C8",
      light: "#9ABDD6",
      dark: "#5D87A8",
      contrastText: "#0F1A22",
    },
    secondary: {
      main: "#D2A15F",
      light: "#DEB785",
      dark: "#AF7F41",
      contrastText: "#211708",
    },
    success: {
      main: "#7EAE99",
    },
    warning: {
      main: "#D1A15C",
    },
    error: {
      main: "#D07B6E",
    },
    info: {
      main: "#88ACD2",
    },
    background: {
      default: "#161C21",
      paper: "#1E262D",
    },
    divider: "#313C46",
    text: {
      primary: "#E7EDF1",
      secondary: "#AEB9C2",
    },
  },
  components: {
    MuiAppBar: {
      styleOverrides: {
        colorSecondary: {
          backgroundColor: "rgba(30, 38, 45, 0.9)",
          color: "#E7EDF1",
          borderBottom: "1px solid rgba(127, 169, 200, 0.14)",
        },
      },
    },
    MuiPaper: {
      styleOverrides: {
        root: {
          border: "1px solid rgba(127, 169, 200, 0.08)",
        },
      },
    },
    MuiOutlinedInput: {
      styleOverrides: {
        notchedOutline: {
          borderColor: "rgba(127, 169, 200, 0.2)",
        },
      },
    },
  },
} satisfies ThemeOptions);
