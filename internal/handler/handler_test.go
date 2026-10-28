package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"pgx-practice/internal/domain"
	"pgx-practice/internal/service"
	"strings"
	"testing"
	"time"
)

type fakeService struct {
	getTasksResult []domain.Task
	getTasksErr    error

	getTasksByIDResult domain.Task
	getTasksByIDErr    error
	getTasksByIDArg    int
	getTaskByCalled    bool

	postTaskResult domain.Task
	postTaskErr    error
	postTaskCalled bool
	postTaskTitle  string
	postTaskStatus string

	waitCancel bool
}

func (f *fakeService) GetTasks(ctx context.Context) ([]domain.Task, error) {
	if f.waitCancel == true {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return f.getTasksResult, f.getTasksErr
}

func (f *fakeService) CreateTask(ctx context.Context, title, status string) (domain.Task, error) {
	f.postTaskCalled = true
	f.postTaskTitle = title
	f.postTaskStatus = status

	if f.waitCancel == true {
		<-ctx.Done()
		return domain.Task{}, ctx.Err()
	}

	return f.postTaskResult, f.postTaskErr
}

func (f *fakeService) GetTaskByID(ctx context.Context, id int) (domain.Task, error) {
	f.getTaskByCalled = true
	f.getTasksByIDArg = id

	if f.waitCancel == true {
		<-ctx.Done()
		return domain.Task{}, ctx.Err()
	}
	return f.getTasksByIDResult, f.getTasksByIDErr
}

func TestPostApi(t *testing.T) {
	tests := []struct {
		name        string
		service     fakeService
		title       string
		status      string
		body        string
		wantCode    int
		wantTask    domain.Task
		wantErrBody string
		wantCalled  bool
	}{
		{
			name: "success",
			service: fakeService{
				postTaskResult: domain.Task{
					ID:     1,
					Title:  "learn to Go",
					Status: "active",
				},
			},
			title:  "learn to Go",
			status: "active",
			body: `{
				"title": "learn to Go",
				"status": "active"
			}`,
			wantCode: http.StatusCreated,
			wantTask: domain.Task{
				ID:     1,
				Title:  "learn to Go",
				Status: "active",
			},
			wantCalled: true,
		},
		{
			name:    "bad json",
			service: fakeService{},
			body: `{
				"title": "learn to Go",
				"status": "active"
			`,
			wantCode:    http.StatusBadRequest,
			wantErrBody: "bad json",
			wantCalled:  false,
		},
		{
			name: "validation error",
			service: fakeService{
				postTaskErr: service.ErrInvalidTask,
			},
			title:  "",
			status: "active",
			body: `{
				"title": "",
				"status": "active"
			}`,
			wantCode:    http.StatusBadRequest,
			wantErrBody: "invalid task error",
			wantCalled:  true,
		},
		{
			name: "service error",
			service: fakeService{
				postTaskErr: errors.New("service error"),
			},
			title:  "learn to Go",
			status: "active",
			body: `{
				"title": "learn to Go",
				"status": "active"
			}`,
			wantCode:    http.StatusInternalServerError,
			wantErrBody: "invalid server error",
			wantCalled:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(&tt.service, 50*time.Millisecond)

			req := httptest.NewRequest(
				http.MethodPost,
				"/tasks",
				strings.NewReader(tt.body),
			)

			rec := httptest.NewRecorder()

			handler.PostApi(rec, req)

			if tt.wantCalled != tt.service.postTaskCalled {
				t.Errorf(
					"CreateTask() called = %v, want %v",
					tt.service.postTaskCalled,
					tt.wantCalled,
				)
			}

			if tt.wantCalled && tt.service.postTaskTitle != tt.title {
				t.Fatalf("error title = %s, want = %s", tt.service.postTaskTitle, tt.title)
			}

			if tt.wantCalled && tt.service.postTaskStatus != tt.status {
				t.Fatalf("error status = %s, want %s", tt.service.postTaskStatus, tt.status)
			}

			if rec.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d",
					rec.Code,
					tt.wantCode)
			}

			if tt.wantErrBody != "" {
				var got struct {
					Error string `json:"error"`
				}

				if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
					t.Fatalf("decode error response %v", err)
				}

				if got.Error != tt.wantErrBody {
					t.Errorf(
						"error = %q, want %q",
						got.Error,
						tt.wantErrBody,
					)
				}

				return
			}

			var u domain.Task
			if err := json.NewDecoder(rec.Body).Decode(&u); err != nil {
				t.Fatalf("decode task response %v", err)
			}

			if u != tt.wantTask {
				t.Errorf(
					"PostApi() task = %v, want %v",
					u,
					tt.wantTask,
				)
			}
		})
	}
}

func TestGetApiByID(t *testing.T) {
	tests := []struct {
		name        string
		service     fakeService
		id          string
		wantId      int
		wantCode    int
		wantTask    domain.Task
		wantErrBody string
		wantCalled  bool
	}{
		{
			name: "success",
			service: fakeService{
				getTasksByIDResult: domain.Task{
					ID:     1,
					Title:  "learn to Go 1",
					Status: "active",
				},
			},
			id:       "1",
			wantId:   1,
			wantCode: http.StatusOK,
			wantTask: domain.Task{
				ID:     1,
				Title:  "learn to Go 1",
				Status: "active",
			},
			wantCalled: true,
		},
		{
			name:        "invalid id",
			service:     fakeService{},
			id:          "0",
			wantCode:    http.StatusBadRequest,
			wantErrBody: "invalid id",
			wantCalled:  false,
		},
		{
			name:        "invalid path",
			service:     fakeService{},
			id:          "abc",
			wantCode:    http.StatusBadRequest,
			wantErrBody: "invalid URL",
			wantCalled:  false,
		},
		{
			name: "not found",
			service: fakeService{
				getTasksByIDErr: service.ErrTaskNotFound,
			},
			id:          "4",
			wantId:      4,
			wantCode:    http.StatusNotFound,
			wantErrBody: "task not found",
			wantCalled:  true,
		},
		{
			name: "service error",
			service: fakeService{
				getTasksByIDErr: errors.New("server error"),
			},
			id:          "5",
			wantId:      5,
			wantCode:    http.StatusInternalServerError,
			wantErrBody: "invalid server error",
			wantCalled:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(&tt.service, 50*time.Millisecond)

			req := httptest.NewRequest(
				http.MethodGet,
				"/tasks/"+tt.id,
				nil,
			)
			req.SetPathValue("id", tt.id)

			rec := httptest.NewRecorder()

			handler.GetApiByID(rec, req)

			if tt.wantCalled != tt.service.getTaskByCalled {
				t.Errorf(
					"GetTaskByID() called = %v, want %v",
					tt.service.getTaskByCalled,
					tt.wantCalled,
				)
			}

			if tt.wantCalled && tt.service.getTasksByIDArg != tt.wantId {
				t.Errorf(
					"GetTaskByID() id = %d, want %d",
					tt.service.getTasksByIDArg,
					tt.wantId,
				)
			}

			if tt.wantCode != rec.Code {
				t.Fatalf("status = %d, want %d",
					rec.Code,
					tt.wantCode)
			}

			if tt.wantErrBody != "" {
				var got struct {
					Error string `json:"error"`
				}

				if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
					t.Fatalf("decode error response: %v", err)
				}

				if got.Error != tt.wantErrBody {
					t.Errorf(
						"error = %q, want %q",
						got.Error,
						tt.wantErrBody,
					)
				}

				return
			}

			var u domain.Task
			if err := json.NewDecoder(rec.Body).Decode(&u); err != nil {
				t.Fatalf("decode task response: %v", err)
			}

			if u != tt.wantTask {
				t.Errorf("GetTaskByID %v, want = %v", u, tt.wantTask)
			}

		})
	}
}

func TestGetApi(t *testing.T) {
	tasks := []struct {
		name        string
		service     fakeService
		wantCode    int
		wantTasks   []domain.Task
		wantErrBody string
	}{
		{
			name: "success",
			service: fakeService{
				getTasksResult: []domain.Task{
					{
						ID:     1,
						Title:  "learn to Go 1",
						Status: "active",
					},
				},
			},
			wantCode: http.StatusOK,
			wantTasks: []domain.Task{
				{
					ID:     1,
					Title:  "learn to Go 1",
					Status: "active",
				},
			},
		},
		{
			name: "database error",
			service: fakeService{
				getTasksErr: errors.New("database error"),
			},
			wantCode:    http.StatusInternalServerError,
			wantErrBody: "invalid server error",
		},
	}

	for _, tt := range tasks {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(&tt.service, 50*time.Millisecond)

			req := httptest.NewRequest(
				http.MethodGet,
				"/tasks",
				nil,
			)

			rec := httptest.NewRecorder()

			handler.GetApi(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf(
					"status = %d, want %d",
					rec.Code,
					tt.wantCode,
				)
			}

			if tt.wantErrBody != "" {
				var got struct {
					Error string `json:"error"`
				}

				if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
					t.Fatalf("decode error response: %v", err)
				}

				if got.Error != tt.wantErrBody {
					t.Errorf(
						"error = %q, want %q",
						got.Error,
						tt.wantErrBody,
					)
				}

				return
			}

			var got []domain.Task

			if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
				t.Fatalf("decode tasks response: %v", err)
			}

			if len(got) != len(tt.wantTasks) {
				t.Fatalf(
					"len(tasks) = %d, want %d",
					len(got),
					len(tt.wantTasks),
				)
			}

			for i := range got {
				if got[i] != tt.wantTasks[i] {
					t.Errorf(
						"tasks[%d] = %+v, want %+v",
						i,
						got[i],
						tt.wantTasks[i],
					)
				}
			}

		})
	}
}

func TestGetApiDeadlineCancel(t *testing.T) {
	handler := NewHandler(&fakeService{
		waitCancel: true,
	},
		50*time.Millisecond,
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/tasks",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.GetApi(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Errorf(
			"status = %d, want %d",
			rec.Code,
			http.StatusGatewayTimeout,
		)
	}

	var got struct {
		Error string `json:"error"`
	}

	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("Decode error response %v", err)
	}

	if got.Error != "request timeout" {
		t.Errorf("error response body %s, want %s", got.Error, "request timeout")
	}
}

func TestGetApiByIDDeadlineCancel(t *testing.T) {
	handler := NewHandler(&fakeService{
		waitCancel: true,
	}, 50*time.Millisecond)

	req := httptest.NewRequest(
		http.MethodGet,
		"/tasks/1",
		nil,
	)

	req.SetPathValue("id", "1")

	rec := httptest.NewRecorder()

	handler.GetApiByID(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Errorf(
			"status = %d, want %d",
			rec.Code,
			http.StatusGatewayTimeout,
		)
	}

	var got struct {
		Error string `json:"error"`
	}

	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("Decode error response %v", err)
	}

	if got.Error != "request timeout" {
		t.Errorf("error response body %s, want %s", got.Error, "request timeout")
	}
}

func TestPostApiDeadlineCancel(t *testing.T) {
	handler := NewHandler(&fakeService{
		waitCancel: true,
	}, 50*time.Millisecond)

	body := `{
		"title": "learn to Go",
		"status": "active"
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/tasks",
		strings.NewReader(body),
	)

	rec := httptest.NewRecorder()

	handler.PostApi(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Errorf(
			"status = %d, want %d",
			rec.Code,
			http.StatusGatewayTimeout,
		)
	}

	var got struct {
		Error string `json:"error"`
	}

	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("Decode error response %v", err)
	}

	if got.Error != "request timeout" {
		t.Errorf("error response body %s, want %s", got.Error, "request timeout")
	}
}

func TestGetApiContextCanceled(t *testing.T) {
	handler := NewHandler(&fakeService{
		waitCancel: true,
	}, 50*time.Millisecond)

	req := httptest.NewRequest(
		http.MethodGet,
		"/tasks",
		nil,
	)

	ctx, cancel := context.WithCancel(req.Context())

	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	cancel()
	handler.GetApi(rec, req)

	if rec.Code != statusClientCloseRequest {
		t.Errorf(
			"status = %d, want %d",
			rec.Code,
			statusClientCloseRequest,
		)
	}

	var got struct {
		Error string `json:"error"`
	}

	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("Decode error response %v", err)
	}

	if got.Error != "client close request" {
		t.Errorf("error response body %s, want %s", got.Error, "client close request")
	}
}

func TestGetApiByIDContextCanceled(t *testing.T) {
	handler := NewHandler(&fakeService{
		waitCancel: true,
	}, 50*time.Millisecond)

	req := httptest.NewRequest(
		http.MethodGet,
		"/tasks/1",
		nil,
	)
	req.SetPathValue("id", "1")

	ctx, cancel := context.WithCancel(req.Context())

	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	cancel()
	handler.GetApiByID(rec, req)

	if rec.Code != statusClientCloseRequest {
		t.Errorf(
			"status = %d, want %d",
			rec.Code,
			statusClientCloseRequest,
		)
	}

	var got struct {
		Error string `json:"error"`
	}

	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("Decode error response %v", err)
	}

	if got.Error != "client close request" {
		t.Errorf("error response body %s, want %s", got.Error, "client close request")
	}
}

func TestPostApiContextCanceled(t *testing.T) {
	handler := NewHandler(&fakeService{
		waitCancel: true,
	}, 50*time.Millisecond)

	body := `{
		"title": "learn to Go",
		"status": "active"
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/tasks",
		strings.NewReader(body),
	)

	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	cancel()
	handler.PostApi(rec, req)

	if rec.Code != statusClientCloseRequest {
		t.Errorf(
			"status = %d, want %d",
			rec.Code,
			statusClientCloseRequest,
		)
	}

	var got struct {
		Error string `json:"error"`
	}

	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("Decode error response %v", err)
	}

	if got.Error != "client close request" {
		t.Errorf("error response body %s, want %s", got.Error, "client close request")
	}
}
