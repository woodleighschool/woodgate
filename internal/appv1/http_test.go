package appv1

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func multipartRequest(t *testing.T, fields [][2]string, photo []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	for _, field := range fields {
		if err := form.WriteField(field[0], field[1]); err != nil {
			t.Fatal(err)
		}
	}
	if photo != nil {
		file, err := form.CreateFormFile("photo", "selfie.jpg")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write(photo); err != nil {
			t.Fatal(err)
		}
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/checkins", &body)
	request.Header.Set("Content-Type", form.FormDataContentType())
	return request
}

func TestMultipartContract(t *testing.T) {
	valid := [][2]string{{"user_id", "10000000-0000-0000-0000-000000000001"}, {"location_id", "20000000-0000-0000-0000-000000000001"}, {"direction", " check_in "}, {"notes", "  front desk  "}}
	t.Run("values and photo", func(t *testing.T) {
		body, problem := parseSubmission(httptest.NewRecorder(), multipartRequest(t, valid, []byte("photo bytes")))
		if problem != nil {
			t.Fatalf("problem = %+v", problem)
		}
		if body.Direction != "check_in" || body.Notes != "front desk" || string(body.Photo) != "photo bytes" {
			t.Fatalf("body = %+v", body)
		}
	})
	t.Run("empty file", func(t *testing.T) {
		_, problem := parseSubmission(httptest.NewRecorder(), multipartRequest(t, valid, []byte{}))
		want := validationProblem("Checkin is invalid.", []fieldError{{Field: "photo", Message: "must not be empty", Code: "required"}})
		if problem == nil || !reflect.DeepEqual(*problem, want) {
			t.Fatalf("problem = %+v, want %+v", problem, want)
		}
	})
	t.Run("query values are not form values", func(t *testing.T) {
		request := multipartRequest(t, nil, nil)
		request.URL.RawQuery = "user_id=" + valid[0][1] + "&location_id=" + valid[1][1] + "&direction=check_in"
		_, problem := parseSubmission(httptest.NewRecorder(), request)
		if problem == nil || problem.Status != 422 || len(problem.FieldErrors) != 3 {
			t.Fatalf("problem = %+v", problem)
		}
	})
	t.Run("body too large", func(t *testing.T) {
		_, problem := parseSubmission(httptest.NewRecorder(), multipartRequest(t, valid, bytes.Repeat([]byte("x"), maxMultipartBytes)))
		if problem == nil || problem.Status != 400 || problem.Detail != "request body is invalid" {
			t.Fatalf("problem = %+v", problem)
		}
	})
	t.Run("wrong content type", func(t *testing.T) {
		request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/checkins", strings.NewReader("{}"))
		request.Header.Set("Content-Type", "application/json")
		_, problem := parseSubmission(httptest.NewRecorder(), request)
		if problem == nil || problem.Status != 400 || problem.Detail != "request body must be multipart/form-data" {
			t.Fatalf("problem = %+v", problem)
		}
	})
}

func TestProblemWireFormat(t *testing.T) {
	writer := httptest.NewRecorder()
	writeProblem(writer, validationProblem("Checkin is invalid.", []fieldError{{Field: "photo", Message: "is required when the location requires a photo", Code: "required"}}))
	if writer.Code != 422 || writer.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("status=%d content-type=%q", writer.Code, writer.Header().Get("Content-Type"))
	}
	var payload map[string]any
	if err := json.Unmarshal(writer.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["type"] != "urn:woodgate:problem:validation-error" || payload["code"] != "validation_error" || payload["status"] != float64(422) {
		t.Fatalf("payload = %v", payload)
	}
	fields, ok := payload["field_errors"].([]any)
	if !ok || len(fields) != 1 {
		t.Fatalf("fields = %v", payload["field_errors"])
	}
}
