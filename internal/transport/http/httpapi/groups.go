package httpapi

import (
	"net/http"

	"github.com/woodleighschool/woodgate/internal/domain"
)

func (handler *Server) ListGroups(writer http.ResponseWriter, request *http.Request, params ListGroupsParams) {
	listOptions, err := parseListOptions(params.Limit, params.Offset, params.Search, params.Sort, params.Order)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	items, total, err := handler.admin.ListGroups(request.Context(), domain.GroupListOptions{ListOptions: listOptions})
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	writeJSON(writer, http.StatusOK, GroupListResponse{Rows: mapSliceValue(items, mapGroup), Total: total})
}

func (handler *Server) GetGroup(writer http.ResponseWriter, request *http.Request, id Id) {
	item, err := handler.admin.GetGroup(request.Context(), id)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "group not found"})
		return
	}

	writeJSON(writer, http.StatusOK, mapGroup(item))
}

func (handler *Server) ListGroupMemberships(
	writer http.ResponseWriter,
	request *http.Request,
	params ListGroupMembershipsParams,
) {
	listOptions, err := parseListOptions(params.Limit, params.Offset, params.Search, params.Sort, params.Order)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	items, total, err := handler.admin.ListGroupMemberships(request.Context(), domain.GroupMembershipListOptions{
		ListOptions: listOptions,
		GroupID:     uuidPointer(params.GroupId),
		UserID:      uuidPointer(params.UserId),
	})
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	writeJSON(
		writer,
		http.StatusOK,
		GroupMembershipListResponse{Rows: mapSliceValue(items, mapGroupMembership), Total: total},
	)
}

func (handler *Server) GetGroupMembership(writer http.ResponseWriter, request *http.Request, id Id) {
	item, err := handler.admin.GetGroupMembership(request.Context(), id)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "group membership not found"})
		return
	}

	writeJSON(writer, http.StatusOK, mapGroupMembership(item))
}
