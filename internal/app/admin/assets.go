package admin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/google/uuid"

	"github.com/woodleighschool/woodgate/internal/domain"
)

func ensureMediaRoot(mediaRoot string) error {
	if strings.TrimSpace(mediaRoot) == "" {
		return errors.New("media root is required")
	}

	mkdirErr := os.MkdirAll(mediaRoot, 0o750)
	if mkdirErr != nil {
		return fmt.Errorf("create media root: %w", mkdirErr)
	}

	return nil
}

func (service *Service) CreateAsset(
	ctx context.Context,
	name *string,
	content []byte,
) (domain.Asset, error) {
	return service.createStoredAsset(ctx, name, domain.AssetTypeAsset, content)
}

func (service *Service) createStoredAsset(
	ctx context.Context,
	name *string,
	assetType domain.AssetType,
	content []byte,
) (domain.Asset, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return domain.Asset{}, fmt.Errorf("create asset id: %w", err)
	}

	contentType, fileExtension, validateErr := validateAssetContent(content)
	if validateErr != nil {
		return domain.Asset{}, validateErr
	}

	writeErr := writeAssetFile(assetAbsolutePath(service.mediaRoot, assetType, id, fileExtension), content)
	if writeErr != nil {
		return domain.Asset{}, writeErr
	}

	item, err := service.store.CreateAsset(ctx, id, name, assetType, contentType, fileExtension)
	if err != nil {
		removeAssetFile(assetAbsolutePath(service.mediaRoot, assetType, id, fileExtension))
		return domain.Asset{}, err
	}

	return item, nil
}

func (service *Service) GetAssetFile(ctx context.Context, id uuid.UUID) (domain.Asset, string, error) {
	item, err := service.store.GetAsset(ctx, id)
	if err != nil {
		return domain.Asset{}, "", err
	}

	return item, assetAbsolutePath(service.mediaRoot, item.Type, item.ID, item.FileExtension), nil
}

func (service *Service) UpdateAsset(
	ctx context.Context,
	id uuid.UUID,
	name *string,
	content []byte,
) (domain.Asset, error) {
	current, err := service.store.GetAsset(ctx, id)
	if err != nil {
		return domain.Asset{}, err
	}
	if current.Type != domain.AssetTypeAsset {
		return domain.Asset{}, &domain.ValidationError{
			Code:   "validation_error",
			Detail: "Photo assets are immutable.",
		}
	}

	contentType := current.ContentType
	fileExtension := current.FileExtension
	if len(content) > 0 {
		var validateErr error
		contentType, fileExtension, validateErr = validateAssetContent(content)
		if validateErr != nil {
			return domain.Asset{}, validateErr
		}

		writeErr := writeAssetFile(
			assetAbsolutePath(service.mediaRoot, current.Type, current.ID, fileExtension),
			content,
		)
		if writeErr != nil {
			return domain.Asset{}, writeErr
		}
		if fileExtension != current.FileExtension {
			removeAssetFile(assetAbsolutePath(service.mediaRoot, current.Type, current.ID, current.FileExtension))
		}
	}

	return service.store.UpdateAsset(ctx, id, name, contentType, fileExtension)
}

func (service *Service) DeleteAsset(ctx context.Context, id uuid.UUID) error {
	item, err := service.store.GetAsset(ctx, id)
	if err != nil {
		return err
	}

	deleteErr := service.store.DeleteAsset(ctx, id)
	if deleteErr != nil {
		return deleteErr
	}

	removeAssetFile(assetAbsolutePath(service.mediaRoot, item.Type, item.ID, item.FileExtension))
	return nil
}

func assetAbsolutePath(mediaRoot string, assetType domain.AssetType, id uuid.UUID, fileExtension string) string {
	return filepath.Join(mediaRoot, string(assetType), id.String()+fileExtension)
}

func writeAssetFile(absolutePath string, content []byte) error {
	mkdirErr := os.MkdirAll(filepath.Dir(absolutePath), 0o750)
	if mkdirErr != nil {
		return fmt.Errorf("create asset directory: %w", mkdirErr)
	}

	writeErr := os.WriteFile(absolutePath, content, 0o600)
	if writeErr != nil {
		return fmt.Errorf("write asset file: %w", writeErr)
	}

	return nil
}

func removeAssetFile(absolutePath string) {
	if removeErr := os.Remove(absolutePath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		return
	}
}

func validateAssetContent(content []byte) (string, string, error) {
	if len(content) == 0 {
		return "", "", &domain.ValidationError{
			Code:   "validation_error",
			Detail: "Asset is invalid.",
			FieldErrors: []domain.FieldError{{
				Field:   "file",
				Message: "is required",
				Code:    "required",
			}},
		}
	}

	detectedType := mimetype.Detect(content)
	switch {
	case detectedType.Is("image/png"):
		return "image/png", ".png", nil
	case detectedType.Is("image/jpeg"):
		return "image/jpeg", ".jpg", nil
	}

	return "", "", &domain.ValidationError{
		Code:   "validation_error",
		Detail: "Asset is invalid.",
		FieldErrors: []domain.FieldError{{
			Field:   "file",
			Message: "must be a PNG or JPEG image",
			Code:    "invalid",
		}},
	}
}
