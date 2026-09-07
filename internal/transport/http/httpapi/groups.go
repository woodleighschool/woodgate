package httpapi

import (
	"context"

	"github.com/woodleighschool/woodgate/internal/domain"
)

func (handler *Server) listGroups(ctx context.Context, input *ListGroupsParams) (*response[GroupListResponse], error) {
	params := input

	listOptions, err := parseListOptions(params.Limit.Value, params.Offset.Value, params.Search.Value, params.Sort.Value, params.Order.Value)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}

	items, total, err := handler.admin.ListGroups(ctx, domain.GroupListOptions{ListOptions: listOptions})
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}

	return &response[GroupListResponse]{Body: GroupListResponse{Rows: mapSliceValue(items, mapGroup), Total: total}}, nil
}

func (handler *Server) getGroup(ctx context.Context, input *ItemInput) (*response[Group], error) {
	id := input.ID

	item, err := handler.admin.GetGroup(ctx, id)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{NotFoundMessage: "group not found"})
	}

	return &response[Group]{Body: mapGroup(item)}, nil
}

func (handler *Server) listGroupMemberships(ctx context.Context, input *ListGroupMembershipsParams) (*response[GroupMembershipListResponse], error) {
	params := input

	listOptions, err := parseListOptions(params.Limit.Value, params.Offset.Value, params.Search.Value, params.Sort.Value, params.Order.Value)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}

	items, total, err := handler.admin.ListGroupMemberships(ctx, domain.GroupMembershipListOptions{
		ListOptions: listOptions,
		GroupID:     uuidPointer(params.GroupID.Value),
		UserID:      uuidPointer(params.UserID.Value),
	})
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}

	return &response[GroupMembershipListResponse]{Body: GroupMembershipListResponse{Rows: mapSliceValue(items, mapGroupMembership), Total: total}}, nil
}

func (handler *Server) getGroupMembership(ctx context.Context, input *ItemInput) (*response[GroupMembership], error) {
	id := input.ID

	item, err := handler.admin.GetGroupMembership(ctx, id)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{NotFoundMessage: "group membership not found"})
	}

	return &response[GroupMembership]{Body: mapGroupMembership(item)}, nil
}
