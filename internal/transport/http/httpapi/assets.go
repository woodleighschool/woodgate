package httpapi

import (
	"context"
	"github.com/google/uuid"
	"net/http"
	"os"
	"strings"

	"github.com/woodleighschool/woodgate/internal/app/authz"
	"github.com/woodleighschool/woodgate/internal/domain"
)

func (handler *Server) listAssets(ctx context.Context, input *ListAssetsParams) (*response[AssetListResponse], error) {
	params := input

	listOptions, err := parseListOptions(params.Limit.Value, params.Offset.Value, params.Search.Value, params.Sort.Value, params.Order.Value)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}

	allowedTypes, err := assetScope(ctx, handler.authorizer, domain.PermissionActionRead)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}

	types := assetTypesFilter(params.Type.Value, allowedTypes)
	if !allowedTypes.All && len(types) == 0 {
		return &response[AssetListResponse]{Body: AssetListResponse{Rows: []Asset{}, Total: 0}}, nil
	}

	items, total, err := handler.admin.ListAssets(ctx, domain.AssetListOptions{
		ListOptions: listOptions,
		Types:       types,
	})
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}

	return &response[AssetListResponse]{Body: AssetListResponse{Rows: mapSliceValue(items, mapAsset), Total: total}}, nil
}

func (handler *Server) createAsset(ctx context.Context, input *RequestInput) (*response[Asset], error) {
	writer, request := input.writer, input.request

	allowedTypes, err := assetScope(ctx, handler.authorizer, domain.PermissionActionCreate)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}
	if !allowedTypes.Contains(domain.AssetTypeAsset) {
		return nil, classifiedError(domain.ErrPermissionDenied, apiErrorOptions{})
	}

	name, content, err := parseAssetUploadRequest(writer, request, true)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}

	item, err := handler.admin.CreateAsset(ctx, name, content)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}

	return &response[Asset]{Body: mapAsset(item)}, nil
}

func (handler *Server) getAsset(ctx context.Context, input *ItemInput) (*response[Asset], error) {
	id := input.ID

	item, err := handler.admin.GetAsset(ctx, id)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{NotFoundMessage: "asset not found"})
	}
	allowed, allowErr := allowAssetType(ctx, handler.authorizer, domain.PermissionActionRead, item.Type)
	if allowErr != nil {
		return nil, classifiedError(allowErr, apiErrorOptions{})
	}
	if !allowed {
		return nil, classifiedError(domain.ErrPermissionDenied, apiErrorOptions{})
	}
	return &response[Asset]{Body: mapAsset(item)}, nil
}

func (handler *Server) getAssetContent(writer http.ResponseWriter, request *http.Request, id uuid.UUID) {
	item, path, err := handler.admin.GetAssetFile(request.Context(), id)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "asset not found"})
		return
	}
	allowed, allowErr := allowAssetType(request.Context(), handler.authorizer, domain.PermissionActionRead, item.Type)
	if allowErr != nil {
		writeClassifiedError(writer, allowErr, apiErrorOptions{})
		return
	}
	if !allowed {
		writeClassifiedError(writer, domain.ErrPermissionDenied, apiErrorOptions{})
		return
	}

	_, statErr := os.Stat(path)
	if statErr != nil {
		writeClassifiedError(writer, statErr, apiErrorOptions{})
		return
	}

	writer.Header().Set("Cache-Control", "private, max-age=3600")
	writer.Header().Set("Content-Type", item.ContentType)
	http.ServeFile(writer, request, path)
}

func (handler *Server) patchAsset(ctx context.Context, input *itemRequestInput) (*response[Asset], error) {
	writer, request := input.writer, input.request

	id := input.ID

	current, err := handler.admin.GetAsset(ctx, id)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{NotFoundMessage: "asset not found"})
	}
	allowed, allowErr := allowAssetType(
		ctx,
		handler.authorizer,
		domain.PermissionActionWrite,
		current.Type,
	)
	if allowErr != nil {
		return nil, classifiedError(allowErr, apiErrorOptions{})
	}
	if !allowed {
		return nil, classifiedError(domain.ErrPermissionDenied, apiErrorOptions{})
	}

	name, content, err := parseAssetUploadRequest(writer, request, false)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}

	item, err := handler.admin.UpdateAsset(ctx, id, name, content)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{NotFoundMessage: "asset not found"})
	}
	return &response[Asset]{Body: mapAsset(item)}, nil
}

func (handler *Server) deleteAsset(ctx context.Context, input *ItemInput) (*struct{}, error) {
	id := input.ID

	current, err := handler.admin.GetAsset(ctx, id)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{NotFoundMessage: "asset not found"})
	}
	allowed, allowErr := allowAssetType(
		ctx,
		handler.authorizer,
		domain.PermissionActionDelete,
		current.Type,
	)
	if allowErr != nil {
		return nil, classifiedError(allowErr, apiErrorOptions{})
	}
	if !allowed {
		return nil, classifiedError(domain.ErrPermissionDenied, apiErrorOptions{})
	}

	deleteErr := handler.admin.DeleteAsset(ctx, id)
	if deleteErr != nil {
		return nil, classifiedError(deleteErr, apiErrorOptions{NotFoundMessage: "asset not found"})
	}
	return nil, nil
}

func assetTypesFilter(value *AssetType, allowed authz.Scope[domain.AssetType]) []domain.AssetType {
	if allowed.All {
		if value == nil {
			return nil
		}
		return []domain.AssetType{domain.AssetType(*value)}
	}

	if value != nil {
		requested := domain.AssetType(*value)
		if allowed.Contains(requested) {
			return []domain.AssetType{requested}
		}
		return []domain.AssetType{}
	}

	return allowed.Values
}

func allowAssetType(
	ctx context.Context,
	authorizer authz.Authorizer,
	required domain.PermissionAction,
	assetType domain.AssetType,
) (bool, error) {
	allowedTypes, err := assetScope(ctx, authorizer, required)
	if err != nil {
		return false, err
	}
	return allowedTypes.Contains(assetType), nil
}

func parseAssetUploadRequest(
	writer http.ResponseWriter,
	request *http.Request,
	requireFile bool,
) (*string, []byte, error) {
	parseErr := parseMultipartForm(writer, request)
	if parseErr != nil {
		return nil, nil, parseErr
	}

	validationErr := &domain.ValidationError{Code: "validation_error", Detail: "Asset is invalid."}
	name := optionalStringPointer(multipartValue(request, "name"))
	content, fileErr := readMultipartFile(request, "file", requireFile, validationErr)
	if fileErr != nil {
		return nil, nil, fileErr
	}

	if validationErr.HasFieldErrors() {
		return nil, nil, validationErr
	}

	return name, content, nil
}

func optionalStringPointer(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
