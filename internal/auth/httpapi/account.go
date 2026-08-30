package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodgate/internal/api"
	"github.com/woodleighschool/woodgate/internal/api/ctxkeys"
	"github.com/woodleighschool/woodgate/internal/auth"
	"github.com/woodleighschool/woodgate/internal/authorization"
	"github.com/woodleighschool/woodgate/internal/directory"
	"github.com/woodleighschool/woodgate/internal/fault"
)

type accountOutput struct {
	Body accountBody
}

type accountBody struct {
	User                 directory.User                                  `json:"user"`
	EffectivePermissions map[authorization.Resource]authorization.Access `json:"effective_permissions"`
}

type accountPutInput struct {
	Body directory.AccountMutation
}

func registerGetAccount(
	humaAPI huma.API,
	authService *auth.Service,
	permissions PermissionDirectory,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "get-account",
		Method:      http.MethodGet,
		Path:        "/api/account",
		Tags:        []string{api.TagAccount},
		Summary:     "Get account",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, _ *struct{}) (*accountOutput, error) {
		user, err := ctxkeys.RequireUser(ctx)
		if err != nil {
			return nil, err
		}
		account, err := authService.Account(ctx, user.ID)
		if err != nil {
			return nil, api.HandlerError(ctx, logger, "get-account", err, "user_id", user.ID)
		}
		output, err := newAccountOutput(ctx, account, permissions)
		if err != nil {
			return nil, api.HandlerError(ctx, logger, "get-account", err, "user_id", user.ID)
		}
		return output, nil
	})
}

func registerPutAccount(
	humaAPI huma.API,
	userService *directory.UserService,
	permissions PermissionDirectory,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "update-account",
		Method:      http.MethodPut,
		Path:        "/api/account",
		Tags:        []string{api.TagAccount},
		Summary:     "Update account",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusConflict,
			http.StatusNotFound,
		},
	}, func(ctx context.Context, input *accountPutInput) (*accountOutput, error) {
		user, err := ctxkeys.RequireUser(ctx)
		if err != nil {
			return nil, err
		}
		account, err := userService.UpdateAccount(ctx, user.ID, input.Body)
		if err != nil {
			return nil, api.HandlerError(ctx, logger, "update-account", accountMutationError(err), "user_id", user.ID)
		}
		output, err := newAccountOutput(ctx, account, permissions)
		if err != nil {
			return nil, api.HandlerError(ctx, logger, "update-account", err, "user_id", user.ID)
		}
		return output, nil
	})
}

func newAccountOutput(
	ctx context.Context,
	account *directory.Account,
	permissions PermissionDirectory,
) (*accountOutput, error) {
	effective, err := permissions.EffectivePermissions(ctx, account.User.ID)
	if err != nil {
		return nil, fmt.Errorf("get effective permissions: %w", err)
	}
	return &accountOutput{Body: accountBody{
		User:                 account.User,
		EffectivePermissions: effective,
	}}, nil
}

func accountMutationError(err error) error {
	switch {
	case errors.Is(err, fault.ErrAlreadyExists):
		return huma.Error409Conflict("email already in use")
	case errors.Is(err, directory.ErrWeakPassword):
		return huma.Error400BadRequest(directory.ErrWeakPassword.Error())
	default:
		return api.ResourceMutationError("user", err)
	}
}
