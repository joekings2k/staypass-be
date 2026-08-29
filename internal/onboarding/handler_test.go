package onboarding

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/joekings2k/staypass-be/internal/estate"
	"github.com/joekings2k/staypass-be/internal/user"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// errorResponse mirrors the JSON error envelope produced by the web package.
type errorResponse struct {
	Error struct {
		Code    string            `json:"code"`
		Message string            `json:"message"`
		Fields  map[string]string `json:"fields"`
	} `json:"error"`
}

func TestRegisterEstate_Validation(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		wantStatus     int
		wantCode       string
		wantFieldKeys  []string
		wantFieldMsgs  map[string]string
		wantFieldCount int
	}{
		{
			name:           "missing required fields",
			body:           `{}`,
			wantStatus:     http.StatusUnprocessableEntity,
			wantCode:       "validation",
			wantFieldKeys:  []string{"fullName", "email", "phone", "password", "estateName"},
			wantFieldCount: 5, // optional fields (address/city/state) must not appear
		},
		{
			name:           "invalid email",
			body:           `{"fullName":"Ada","email":"not-an-email","phone":"08012345678","password":"supersecret","estateName":"Green Park"}`,
			wantStatus:     http.StatusUnprocessableEntity,
			wantCode:       "validation",
			wantFieldMsgs:  map[string]string{"email": "must be a valid email address"},
			wantFieldCount: 1,
		},
		{
			name:           "password too short",
			body:           `{"fullName":"Ada","email":"ada@example.com","phone":"08012345678","password":"short","estateName":"Green Park"}`,
			wantStatus:     http.StatusUnprocessableEntity,
			wantCode:       "validation",
			wantFieldMsgs:  map[string]string{"password": "must be at least 8 characters"},
			wantFieldCount: 1,
		},
		{
			name:           "malformed json falls back to 400",
			body:           `{"email":`,
			wantStatus:     http.StatusBadRequest,
			wantCode:       "bad_request",
			wantFieldCount: 0,
		},
		{
			name:           "unknown field is rejected",
			body:           `{"fullName":"Ada","email":"ada@example.com","phone":"08012345678","password":"supersecret","estateName":"Green Park","slug":"green-park","role":"super_admin"}`,
			wantStatus:     http.StatusBadRequest,
			wantCode:       "bad_request",
			wantFieldCount: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHandler(nil)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/estate-onboarding", strings.NewReader(tc.body))
			rec := httptest.NewRecorder()

			h.RegisterEstate(rec, req)

			require.Equal(t, tc.wantStatus, rec.Code)

			var resp errorResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
			require.Equal(t, tc.wantCode, resp.Error.Code)

			for _, key := range tc.wantFieldKeys {
				require.Contains(t, resp.Error.Fields, key, "expected field %q to be reported", key)
			}
			for key, msg := range tc.wantFieldMsgs {
				require.Equal(t, msg, resp.Error.Fields[key], "unexpected message for field %q", key)
			}
			if tc.wantFieldCount >= 0 {
				require.Len(t, resp.Error.Fields, tc.wantFieldCount)
			}
		})
	}
}

func TestRegisterEstate_ServiceErrors(t *testing.T) {
	// A well-formed body so the request always reaches the service.
	const validBody = `{
		"fullName":   "Ada Lovelace",
		"email":      "ada@example.com",
		"phone":      "08012345678",
		"password":   "supersecret",
		"estateName": "Green Park Estate"
	}`

	tests := []struct {
		name       string
		serviceErr error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "email already taken maps to 409",
			serviceErr: user.ErrEmailTaken,
			wantStatus: http.StatusConflict,
			wantCode:   "conflict",
		},
		{
			name:       "estate already exists maps to 409",
			serviceErr: estate.ErrEstateAlreadyExists,
			wantStatus: http.StatusConflict,
			wantCode:   "conflict",
		},
		{
			name:       "unexpected error maps to 500",
			serviceErr: errors.New("db exploded"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			service := NewMockOnboardingServicer(ctrl)
			service.EXPECT().
				Register(gomock.Any(), gomock.Any()).
				Return(nil, nil, nil, tc.serviceErr)

			h := NewHandler(service)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/estate-onboarding", strings.NewReader(validBody))
			rec := httptest.NewRecorder()

			h.RegisterEstate(rec, req)

			require.Equal(t, tc.wantStatus, rec.Code)

			var resp errorResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
			require.Equal(t, tc.wantCode, resp.Error.Code)
		})
	}
}

func TestRegisterEstate_success(t *testing.T) {
	tests := []struct {
		name          string
		body          string
		buildStubs    func(s *MockOnboardingServicer)
		wantStatus    int
		checkResponse func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name: "ok",
			body: `{
			"fullName": "Ada Lovelace",
				"email": "ada@example.com",
				"phone": "08012345678",
				"password": "supersecret",
				"estateName": "Green Park Estate"
			}`,
			buildStubs: func(s *MockOnboardingServicer) {
				expectedEstate := &estate.Estate{
					ID: uuid.UUID(uuid.New()),
				}

				expectedUser := &user.User{
					ID: uuid.UUID(uuid.New()),
				}
				s.EXPECT().Register(gomock.Any(), gomock.Any()).Return(expectedEstate, expectedUser, nil, nil)

			},
			wantStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, rec.Code)

				var response registerEstateResponse
				require.NoError(
					t,
					json.NewDecoder(rec.Body).Decode(&response),
				)

				require.NotEmpty(t, response.EstateID)
				require.NotEmpty(t, response.UserID)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			service := NewMockOnboardingServicer(ctrl)

			tc.buildStubs(service)

			h := NewHandler(service)

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/estate-onboarding",
				strings.NewReader(tc.body),
			)

			rec := httptest.NewRecorder()

			h.RegisterEstate(rec, req)

			require.Equal(t, tc.wantStatus, rec.Code)

			tc.checkResponse(t, rec)
		})
	}
}
