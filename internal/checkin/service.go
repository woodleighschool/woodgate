package checkin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/woodleighschool/woodgate/internal/fault"
	"github.com/woodleighschool/woodgate/internal/listing"
	"github.com/woodleighschool/woodgate/internal/station"
	"github.com/woodleighschool/woodgate/internal/storage"
)

type attachmentIngestor interface {
	BeginDirect(context.Context, string, string) (*storage.Object, storage.UploadTarget, error)
	Finalize(context.Context, int64, string) (*storage.Object, error)
	Write(context.Context, string, string, string, []byte) (*storage.Object, error)
	Delete(context.Context, int64, string) error
}

type objectDeliverer interface {
	Deliver(http.ResponseWriter, *http.Request, storage.Object, storage.DeliveryOptions) error
}

type locationNotifier interface {
	LocationChanged(int64)
}

// Service owns check-in workflow rules and storage lifecycle.
type Service struct {
	store    *Store
	objects  *storage.ObjectStore
	ingestor attachmentIngestor
	delivery objectDeliverer
	notifier locationNotifier
}

// NewService returns the check-in application service.
func NewService(store *Store, objects *storage.ObjectStore, ingestor *storage.Ingestor, delivery *storage.Delivery) *Service {
	return &Service{store: store, objects: objects, ingestor: ingestor, delivery: delivery}
}

// SetLocationNotifier connects location mutations to Station refreshes.
func (s *Service) SetLocationNotifier(notifier locationNotifier) { s.notifier = notifier }

// ListLocations returns paginated locations.
func (s *Service) ListLocations(ctx context.Context, params LocationListParams) ([]Location, int, error) {
	params.ListParams = listingNormalize(params.ListParams)
	if err := listing.Validate(params.ListParams); err != nil {
		return nil, 0, err
	}
	return s.store.ListLocations(ctx, params)
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
	return s.store.CreateLocation(ctx, mutation)
}

// UpdateLocation replaces a location and tells its Stations to refresh.
func (s *Service) UpdateLocation(ctx context.Context, id int64, mutation LocationMutation) (*Location, error) {
	mutation.normalize()
	if err := mutation.validate(); err != nil {
		return nil, err
	}
	location, err := s.store.UpdateLocation(ctx, id, mutation)
	if err == nil && s.notifier != nil {
		s.notifier.LocationChanged(id)
	}
	return location, err
}

// DeleteLocation deletes an unreferenced location.
func (s *Service) DeleteLocation(ctx context.Context, id int64) error {
	return s.store.DeleteLocation(ctx, id)
}

// GetStationLocation returns the location projection used by Station v1.
func (s *Service) GetStationLocation(ctx context.Context, id int64) (*station.LocationConfiguration, error) {
	location, err := s.store.GetLocation(ctx, id)
	if err != nil {
		return nil, err
	}
	return &station.LocationConfiguration{ID: location.ID, Name: location.Name, Enabled: location.Enabled, Notes: location.Notes,
		Photo: location.Photo, BackgroundObjectID: location.BackgroundObjectID, LogoObjectID: location.LogoObjectID, UpdatedAt: location.UpdatedAt}, nil
}

// ListStationPeople returns only people eligible for a location's configured groups.
func (s *Service) ListStationPeople(ctx context.Context, locationID int64) ([]station.Person, error) {
	if _, err := s.store.GetLocation(ctx, locationID); err != nil {
		return nil, err
	}
	rows, err := s.store.listPeople(ctx, locationID)
	if err != nil {
		return nil, err
	}
	people := make([]station.Person, len(rows))
	for i, row := range rows {
		people[i] = station.Person{ID: row.ID, Name: row.Name, Email: row.Email}
	}
	return people, nil
}

// ListCheckins returns paginated check-in history.
func (s *Service) ListCheckins(ctx context.Context, params CheckinListParams) ([]Checkin, int, error) {
	params.normalize()
	if err := params.validate(); err != nil {
		return nil, 0, err
	}
	return s.store.ListCheckins(ctx, params)
}

// GetCheckin returns one check-in event.
func (s *Service) GetCheckin(ctx context.Context, id int64) (*Checkin, error) {
	return s.store.GetCheckin(ctx, id)
}

// CreateCheckin records a human-authenticated event.
func (s *Service) CreateCheckin(ctx context.Context, create CheckinCreate, actorUserID int64) (*Checkin, error) {
	create.normalize()
	if err := create.validate(); err != nil {
		return nil, err
	}
	location, err := s.store.GetLocation(ctx, create.LocationID)
	if err != nil {
		return nil, err
	}
	if !location.Enabled {
		return nil, fmt.Errorf("%w: location is disabled", fault.ErrInvalidInput)
	}
	if location.Photo {
		return nil, fmt.Errorf("%w: use a Station when this location requires a photo", fault.ErrInvalidInput)
	}
	if !location.Notes && create.Notes != "" {
		return nil, fmt.Errorf("%w: notes are disabled for this location", fault.ErrInvalidInput)
	}
	eligible, err := s.store.PersonEligible(ctx, create.LocationID, create.UserID)
	if err != nil {
		return nil, err
	}
	if !eligible {
		return nil, fmt.Errorf("%w: user is not eligible for this location", fault.ErrInvalidInput)
	}
	return s.store.CreateCheckin(ctx, create, nil, &actorUserID, nil)
}

// SubmitStationCheckin validates and records an event from a bound Station.
func (s *Service) SubmitStationCheckin(ctx context.Context, submission station.CheckinSubmission) (*station.CheckinReceipt, error) {
	create := CheckinCreate{UserID: submission.PersonID, LocationID: submission.LocationID, Direction: Direction(submission.Direction), Notes: strings.TrimSpace(submission.Notes)}
	if err := create.validate(); err != nil {
		return nil, err
	}
	location, err := s.store.GetLocation(ctx, create.LocationID)
	if err != nil {
		return nil, err
	}
	if !location.Enabled {
		return nil, fmt.Errorf("%w: location is disabled", fault.ErrConflict)
	}
	if !location.Notes && create.Notes != "" {
		return nil, fmt.Errorf("%w: notes are disabled for this location", fault.ErrInvalidInput)
	}
	if location.Photo && len(submission.Photo) == 0 {
		return nil, fmt.Errorf("%w: photo is required", fault.ErrInvalidInput)
	}
	eligible, err := s.store.PersonEligible(ctx, create.LocationID, create.UserID)
	if err != nil {
		return nil, err
	}
	if !eligible {
		return nil, fmt.Errorf("%w: person is not eligible for this location", fault.ErrInvalidInput)
	}

	var photoObjectID *int64
	var object *storage.Object
	if len(submission.Photo) > 0 {
		object, err = s.ingestor.Write(ctx, PhotoObjectPrefix, "photo.jpg", submission.ContentType, submission.Photo)
		if err != nil {
			return nil, err
		}
		photoObjectID = &object.ID
	}
	item, err := s.store.CreateCheckin(ctx, create, &submission.StationID, nil, photoObjectID)
	if err != nil && object != nil {
		return nil, errors.Join(err, s.cleanupObject(ctx, object.ID, PhotoObjectPrefix))
	}
	if err != nil {
		return nil, err
	}
	return &station.CheckinReceipt{ID: item.ID, PersonID: item.UserID, LocationID: item.LocationID, Direction: station.Direction(item.Direction)}, nil
}

// BeginLocationBackgroundUpload reserves an object in the background gallery.
func (s *Service) BeginLocationBackgroundUpload(ctx context.Context, filename string) (*storage.Object, storage.UploadTarget, error) {
	return s.ingestor.BeginDirect(ctx, BackgroundObjectPrefix, filename)
}

// BeginLocationLogoUpload reserves an object in the logo gallery.
func (s *Service) BeginLocationLogoUpload(ctx context.Context, filename string) (*storage.Object, storage.UploadTarget, error) {
	return s.ingestor.BeginDirect(ctx, LogoObjectPrefix, filename)
}

// ListLocationBackgrounds returns available objects in the background gallery.
func (s *Service) ListLocationBackgrounds(ctx context.Context, params listing.Params) ([]storage.Object, int, error) {
	params = listingNormalize(params)
	if err := listing.Validate(params); err != nil {
		return nil, 0, err
	}
	return s.objects.ListByPrefix(ctx, BackgroundObjectPrefix, params)
}

// ListLocationLogos returns available objects in the logo gallery.
func (s *Service) ListLocationLogos(ctx context.Context, params listing.Params) ([]storage.Object, int, error) {
	params = listingNormalize(params)
	if err := listing.Validate(params); err != nil {
		return nil, 0, err
	}
	return s.objects.ListByPrefix(ctx, LogoObjectPrefix, params)
}

// SetLocationBackground finalizes and attaches an uploaded background.
func (s *Service) SetLocationBackground(ctx context.Context, locationID, objectID int64) (*storage.Object, error) {
	return s.setLocationAttachment(ctx, locationID, objectID, BackgroundObjectPrefix, s.store.SetLocationBackground)
}

// SetLocationLogo finalizes and attaches an uploaded logo.
func (s *Service) SetLocationLogo(ctx context.Context, locationID, objectID int64) (*storage.Object, error) {
	return s.setLocationAttachment(ctx, locationID, objectID, LogoObjectPrefix, s.store.SetLocationLogo)
}

func (s *Service) setLocationAttachment(
	ctx context.Context,
	locationID, objectID int64,
	prefix string,
	set func(context.Context, int64, int64) error,
) (*storage.Object, error) {
	object, err := s.ingestor.Finalize(ctx, objectID, prefix)
	if err != nil {
		return nil, err
	}
	if err := set(ctx, locationID, object.ID); err != nil {
		return nil, errors.Join(err, s.cleanupObject(ctx, object.ID, prefix))
	}
	if s.notifier != nil {
		s.notifier.LocationChanged(locationID)
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

// DeliverStationBackground sends the background owned by a Station's location.
func (s *Service) DeliverStationBackground(w http.ResponseWriter, r *http.Request, locationID int64) error {
	location, err := s.store.GetLocation(r.Context(), locationID)
	if err != nil {
		return err
	}
	if location.BackgroundObjectID == nil {
		return fault.ErrNotFound
	}
	return s.deliverObject(w, r, *location.BackgroundObjectID, BackgroundObjectPrefix)
}

// DeliverStationLogo sends the logo owned by a Station's location.
func (s *Service) DeliverStationLogo(w http.ResponseWriter, r *http.Request, locationID int64) error {
	location, err := s.store.GetLocation(r.Context(), locationID)
	if err != nil {
		return err
	}
	if location.LogoObjectID == nil {
		return fault.ErrNotFound
	}
	return s.deliverObject(w, r, *location.LogoObjectID, LogoObjectPrefix)
}

func (s *Service) deliverObject(w http.ResponseWriter, r *http.Request, objectID int64, prefix string) error {
	object, err := s.objects.GetByID(r.Context(), objectID)
	if err != nil {
		return err
	}
	if object.Prefix != prefix || !object.Available() {
		return fault.ErrNotFound
	}
	return s.delivery.Deliver(w, r, *object, storage.DeliveryOptions{CacheControl: "private, max-age=3600"})
}

func (s *Service) cleanupObject(ctx context.Context, id int64, prefix string) error {
	err := s.ingestor.Delete(ctx, id, prefix)
	if errors.Is(err, fault.ErrNotFound) || errors.Is(err, fault.ErrConflict) {
		return nil
	}
	return err
}
