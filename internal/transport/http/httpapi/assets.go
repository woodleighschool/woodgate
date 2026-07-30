package httpapi

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/woodleighschool/woodgate/internal/app/authz"
	"github.com/woodleighschool/woodgate/internal/domain"
)

func (handler *Server) ListAssets(writer http.ResponseWriter, request *http.Request, params ListAssetsParams) {
	listOptions, err := parseListOptions(params.Limit, params.Offset, params.Search, params.Sort, params.Order)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	allowedTypes, err := assetScope(request.Context(), handler.authorizer, domain.PermissionActionRead)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	types := assetTypesFilter(params.Type, allowedTypes)
	if !allowedTypes.All && len(types) == 0 {
		writeJSON(writer, http.StatusOK, AssetListResponse{Rows: []Asset{}, Total: 0})
		return
	}

	items, total, err := handler.admin.ListAssets(request.Context(), domain.AssetListOptions{
		ListOptions: listOptions,
		Types:       types,
	})
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	writeJSON(writer, http.StatusOK, AssetListResponse{Rows: mapSliceValue(items, mapAsset), Total: total})
}

func (handler *Server) CreateAsset(writer http.ResponseWriter, request *http.Request) {
	allowedTypes, err := assetScope(request.Context(), handler.authorizer, domain.PermissionActionCreate)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}
	if !allowedTypes.Contains(domain.AssetTypeAsset) {
		writeClassifiedError(writer, domain.ErrPermissionDenied, apiErrorOptions{})
		return
	}

	name, content, err := parseAssetUploadRequest(writer, request, true)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	item, err := handler.admin.CreateAsset(request.Context(), name, content)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	writeJSON(writer, http.StatusCreated, mapAsset(item))
}

func (handler *Server) GetAsset(writer http.ResponseWriter, request *http.Request, id Id) {
	item, err := handler.admin.GetAsset(request.Context(), id)
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
	writeJSON(writer, http.StatusOK, mapAsset(item))
}

func (handler *Server) GetAssetContent(writer http.ResponseWriter, request *http.Request, id Id) {
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

func (handler *Server) PatchAsset(writer http.ResponseWriter, request *http.Request, id Id) {
	current, err := handler.admin.GetAsset(request.Context(), id)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "asset not found"})
		return
	}
	allowed, allowErr := allowAssetType(
		request.Context(),
		handler.authorizer,
		domain.PermissionActionWrite,
		current.Type,
	)
	if allowErr != nil {
		writeClassifiedError(writer, allowErr, apiErrorOptions{})
		return
	}
	if !allowed {
		writeClassifiedError(writer, domain.ErrPermissionDenied, apiErrorOptions{})
		return
	}

	name, content, err := parseAssetUploadRequest(writer, request, false)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	item, err := handler.admin.UpdateAsset(request.Context(), id, name, content)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "asset not found"})
		return
	}
	writeJSON(writer, http.StatusOK, mapAsset(item))
}

func (handler *Server) DeleteAsset(writer http.ResponseWriter, request *http.Request, id Id) {
	current, err := handler.admin.GetAsset(request.Context(), id)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "asset not found"})
		return
	}
	allowed, allowErr := allowAssetType(
		request.Context(),
		handler.authorizer,
		domain.PermissionActionDelete,
		current.Type,
	)
	if allowErr != nil {
		writeClassifiedError(writer, allowErr, apiErrorOptions{})
		return
	}
	if !allowed {
		writeClassifiedError(writer, domain.ErrPermissionDenied, apiErrorOptions{})
		return
	}

	deleteErr := handler.admin.DeleteAsset(request.Context(), id)
	if deleteErr != nil {
		writeClassifiedError(writer, deleteErr, apiErrorOptions{NotFoundMessage: "asset not found"})
		return
	}
	writer.WriteHeader(http.StatusNoContent)
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
