package checkin

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gabriel-vasile/mimetype"

	"github.com/woodleighschool/goodies/bloby"
	"github.com/woodleighschool/woodgate/internal/directory"
	"github.com/woodleighschool/woodgate/internal/fault"
	"github.com/woodleighschool/woodgate/internal/listing"
)

// Service owns check-in workflow rules and resource attachments.
type Service struct {
	store   *Store
	objects *bloby.Service
}

// NewService returns the check-in application service.
func NewService(store *Store, objects *bloby.Service) *Service {
	return &Service{store: store, objects: objects}
}

// ListLocations returns paginated locations.
func (s *Service) ListLocations(ctx context.Context, params LocationListParams) ([]Location, int, error) {
	params.ListParams = listingNormalize(params.ListParams)
	if err := listing.Validate(params.ListParams); err != nil {
		return nil, 0, err
	}
	return s.store.ListLocations(ctx, params)
}

// ListLocationGroupChoices returns group identities assignable to a location.
func (s *Service) ListLocationGroupChoices(
	ctx context.Context,
	params listing.Params,
) ([]directory.GroupSummary, int, error) {
	params = listing.Normalize(params)
	if err := listing.Validate(params); err != nil {
		return nil, 0, err
	}
	return s.store.ListLocationGroupChoices(ctx, params)
}

// GetLocation returns one location.
func (s *Service) GetLocation(ctx context.Context, id int64) (*Location, error) {
	return s.store.GetLocation(ctx, id)
}

// CreateLocation creates a location.
func (s *Service) CreateLocation(ctx context.Context, mutation LocationMutation) (*Location, error) {
	mutation.normalize()
	if err := mutation.validate(); err != nil {
		return nil, err
	}
	ids, err := s.finalizeLocationAttachments(ctx, mutation)
	if err != nil {
		return nil, err
	}
	item, err := s.store.CreateLocation(ctx, mutation)
	if err != nil {
		s.objects.DeleteUnreferenced(ctx, ids...)
	}
	return item, err
}

// UpdateLocation replaces a location.
func (s *Service) UpdateLocation(ctx context.Context, id int64, mutation LocationMutation) (*Location, error) {
	mutation.normalize()
	if err := mutation.validate(); err != nil {
		return nil, err
	}
	ids, err := s.finalizeLocationAttachments(ctx, mutation)
	if err != nil {
		return nil, err
	}
	item, err := s.store.UpdateLocation(ctx, id, mutation)
	if err != nil {
		s.objects.DeleteUnreferenced(ctx, ids...)
	}
	return item, err
}

// DeleteLocation deletes an unreferenced location.
func (s *Service) DeleteLocation(ctx context.Context, id int64) error {
	return s.store.DeleteLocation(ctx, id)
}

// ListCheckins returns paginated check-in history.
func (s *Service) ListCheckins(ctx context.Context, params CheckinListParams) ([]Checkin, int, error) {
	params.normalize()
	if err := params.validate(); err != nil {
		return nil, 0, err
	}
	return s.store.ListCheckins(ctx, params)
}

// ListCheckinDepartments returns departments represented in check-in history.
func (s *Service) ListCheckinDepartments(ctx context.Context) ([]string, error) {
	return s.store.ListCheckinDepartments(ctx)
}

// ListCheckinLocations includes disabled locations with existing history.
func (s *Service) ListCheckinLocations(ctx context.Context) ([]LocationSummary, error) {
	return s.store.ListCheckinLocations(ctx)
}

// GetCheckinUser resolves the person used to scope check-in history.
func (s *Service) GetCheckinUser(ctx context.Context, id int64) (*PersonSummary, error) {
	return s.store.GetCheckinUser(ctx, id)
}

// GetCheckin returns one check-in event.
func (s *Service) GetCheckin(ctx context.Context, id int64) (*Checkin, error) {
	return s.store.GetCheckin(ctx, id)
}

// CreateCheckin records an event submitted by an authenticated user.
func (s *Service) CreateCheckin(ctx context.Context, create CheckinCreate, actorUserID int64) (*Checkin, error) {
	actor, err := s.store.UserActor(ctx, actorUserID)
	if err != nil {
		return nil, err
	}
	return s.Submit(ctx, create, actor, nil)
}

// Submit records an event and any photo as a single owner workflow.
func (s *Service) Submit(ctx context.Context, create CheckinCreate, actor Actor, photo []byte) (*Checkin, error) {
	create.normalize()
	if err := create.validate(); err != nil {
		return nil, err
	}
	location, err := s.store.GetLocation(ctx, create.LocationID)
	if err != nil {
		return nil, err
	}
	if err := ValidateSubmission(*location, create.Notes, len(photo) > 0); err != nil {
		return nil, err
	}
	var photoObjectID *int64
	if len(photo) > 0 {
		contentType := mimetype.Detect(photo)
		if !contentType.Is("image/jpeg") && !contentType.Is("image/png") {
			return nil, fmt.Errorf("%w: photo must be a PNG or JPEG image", fault.ErrInvalidInput)
		}
		filename := "photo.jpg"
		if contentType.Is("image/png") {
			filename = "photo.png"
		}
		object, err := s.objects.Write(ctx, PhotoObjectPrefix, filename, contentType.String(), photo)
		if err != nil {
			return nil, err
		}
		photoObjectID = &object.ID
	}
	item, err := s.store.CreateCheckin(ctx, create, actor, photoObjectID)
	if err != nil && photoObjectID != nil {
		s.objects.DeleteUnreferenced(ctx, *photoObjectID)
	}
	return item, err
}

// ValidateSubmission applies the workflow rules shared by both API surfaces.
func ValidateSubmission(location Location, notes string, hasPhoto bool) error {
	switch {
	case !location.Enabled:
		return fmt.Errorf("%w: location is disabled", fault.ErrInvalidInput)
	case location.Photo && !hasPhoto:
		return fmt.Errorf("%w: photo is required", fault.ErrInvalidInput)
	case !location.Notes && notes != "":
		return fmt.Errorf("%w: notes are disabled for this location", fault.ErrInvalidInput)
	default:
		return nil
	}
}

// BeginLocationBackgroundUpload reserves an object in the background gallery.
func (s *Service) BeginLocationBackgroundUpload(ctx context.Context, filename string) (*bloby.Object, bloby.UploadAction, error) {
	return s.objects.BeginDirect(ctx, BackgroundObjectPrefix, filename)
}

// BeginLocationLogoUpload reserves an object in the logo gallery.
func (s *Service) BeginLocationLogoUpload(ctx context.Context, filename string) (*bloby.Object, bloby.UploadAction, error) {
	return s.objects.BeginDirect(ctx, LogoObjectPrefix, filename)
}

// ListLocationBackgrounds returns available objects in the background gallery.
func (s *Service) ListLocationBackgrounds(ctx context.Context, params listing.Params) ([]bloby.Object, int, error) {
	params = listingNormalize(params)
	if err := listing.Validate(params); err != nil {
		return nil, 0, err
	}
	return s.objects.ListByPrefix(ctx, BackgroundObjectPrefix, blobListOptions(params))
}

// ListLocationLogos returns available objects in the logo gallery.
func (s *Service) ListLocationLogos(ctx context.Context, params listing.Params) ([]bloby.Object, int, error) {
	params = listingNormalize(params)
	if err := listing.Validate(params); err != nil {
		return nil, 0, err
	}
	return s.objects.ListByPrefix(ctx, LogoObjectPrefix, blobListOptions(params))
}

func blobListOptions(params listing.Params) bloby.ListOptions {
	return bloby.ListOptions{
		Limit:  int(params.PageSize),
		Offset: int(params.PageIndex) * int(params.PageSize),
	}
}

// SetLocationBackground finalizes and attaches an uploaded background.
func (s *Service) SetLocationBackground(ctx context.Context, locationID, objectID int64) (*bloby.Object, error) {
	return s.setLocationAttachment(ctx, locationID, objectID, BackgroundObjectPrefix, s.store.SetLocationBackground)
}

// SetLocationLogo finalizes and attaches an uploaded logo.
func (s *Service) SetLocationLogo(ctx context.Context, locationID, objectID int64) (*bloby.Object, error) {
	return s.setLocationAttachment(ctx, locationID, objectID, LogoObjectPrefix, s.store.SetLocationLogo)
}

func (s *Service) setLocationAttachment(
	ctx context.Context,
	locationID, objectID int64,
	prefix string,
	set func(context.Context, int64, int64) error,
) (*bloby.Object, error) {
	object, err := s.objects.Finalize(ctx, objectID, prefix)
	if err != nil {
		return nil, err
	}
	if err := set(ctx, locationID, object.ID); err != nil {
		return nil, errors.Join(err, s.cleanupObject(ctx, object.ID, prefix))
	}
	return object, nil
}

// DeliverLocationBackground sends one object from the background gallery.
func (s *Service) DeliverLocationBackground(w http.ResponseWriter, r *http.Request, objectID int64) error {
	return s.deliverObject(w, r, objectID, BackgroundObjectPrefix)
}

// DeliverLocationLogo sends one object from the logo gallery.
func (s *Service) DeliverLocationLogo(w http.ResponseWriter, r *http.Request, objectID int64) error {
	return s.deliverObject(w, r, objectID, LogoObjectPrefix)
}

// DeliverCheckinPhoto sends the photo owned by one check-in.
func (s *Service) DeliverCheckinPhoto(w http.ResponseWriter, r *http.Request, checkinID int64) error {
	item, err := s.store.GetCheckin(r.Context(), checkinID)
	if err != nil {
		return err
	}
	if item.PhotoObjectID == nil {
		return fault.ErrNotFound
	}
	return s.deliverObject(w, r, *item.PhotoObjectID, PhotoObjectPrefix)
}

func (s *Service) deliverObject(w http.ResponseWriter, r *http.Request, objectID int64, prefix string) error {
	object, err := s.objects.GetByID(r.Context(), objectID)
	if err != nil {
		return err
	}
	if object.Prefix != prefix || !object.Available() {
		return fault.ErrNotFound
	}
	return s.objects.Deliver(w, r, *object, bloby.DeliveryOptions{CacheControl: "private, max-age=3600"})
}

func (s *Service) cleanupObject(ctx context.Context, id int64, prefix string) error {
	err := s.objects.Delete(ctx, id, prefix)
	if errors.Is(err, bloby.ErrNotFound) || errors.Is(err, bloby.ErrConflict) {
		return nil
	}
	return err
}

func (s *Service) finalizeLocationAttachments(ctx context.Context, mutation LocationMutation) ([]int64, error) {
	ids := []int64{}
	for _, attachment := range []struct {
		id     *int64
		prefix string
	}{{mutation.BackgroundObjectID, BackgroundObjectPrefix}, {mutation.LogoObjectID, LogoObjectPrefix}} {
		if attachment.id == nil {
			continue
		}
		object, err := s.objects.Finalize(ctx, *attachment.id, attachment.prefix)
		if err != nil {
			s.objects.DeleteUnreferenced(ctx, ids...)
			return nil, err
		}
		ids = append(ids, object.ID)
	}
	return ids, nil
}
