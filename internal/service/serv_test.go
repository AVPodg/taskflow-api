package service

import (
	"context"
	"errors"
	"pgx-practice/internal/domain"
	"pgx-practice/internal/storage"
	"testing"
	"time"
)

var testErr = errors.New("repository error")

type fakeStorage struct {
	saveResult domain.Task
	saveErr    error
	saveCalled bool
	savedTask  domain.Task

	readByIDResult domain.Task
	readByIDErr    error
	readByIDArg    int

	readResult []domain.Task
	readErr    error

	waitContext bool
}

func (f *fakeStorage) ReadByID(ctx context.Context, n int) (domain.Task, error) {
	f.readByIDArg = n
	return f.readByIDResult, f.readByIDErr
}

func (f *fakeStorage) Read(ctx context.Context) ([]domain.Task, error) {
	if f.waitContext {
		<-ctx.Done()
		return nil, ctx.Err()
	}

	return f.readResult, f.readErr
}

func (f *fakeStorage) Save(ctx context.Context, t domain.Task) (domain.Task, error) {
	f.saveCalled = true
	f.savedTask = t
	return f.saveResult, f.saveErr
}

func TestCreateTask(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		storage    fakeStorage
		title      string
		status     string
		wantID     int
		wantErr    error
		wantCalled bool
	}{
		{
			name: "success",
			storage: fakeStorage{
				saveResult: domain.Task{
					ID:     1,
					Title:  "learn Go",
					Status: "active",
				},
			},
			title:      "learn Go",
			status:     "active",
			wantID:     1,
			wantErr:    nil,
			wantCalled: true,
		},
		{
			name: "repository error",
			storage: fakeStorage{
				saveErr: testErr,
			},
			title:      "learn Go",
			status:     "active",
			wantErr:    testErr,
			wantCalled: true,
		},
		{
			name:       "empty_title",
			storage:    fakeStorage{},
			title:      "",
			status:     "active",
			wantErr:    ErrInvalidTask,
			wantCalled: false,
		},
		{
			name:       "empty_status",
			storage:    fakeStorage{},
			title:      "learn to Go",
			status:     "",
			wantErr:    ErrInvalidTask,
			wantCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(&tt.storage)

			got, err := service.CreateTask(ctx, tt.title, tt.status)

			if tt.storage.saveCalled != tt.wantCalled {
				t.Errorf(
					"Save() called = %v, want %v",
					tt.storage.saveCalled,
					tt.wantCalled,
				)
			}

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("CreateTask() error = nil, want %v", tt.wantErr)
				}

				if !errors.Is(err, tt.wantErr) {
					t.Errorf(
						"CreateTask() error = %v, want %v",
						err,
						tt.wantErr,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf("CreateTask() unexpected error: %v", err)
			}

			if tt.storage.savedTask.Title != tt.title {
				t.Errorf(
					"savedTask.Title = %q, want %q",
					tt.storage.savedTask.Title,
					tt.title,
				)
			}

			if tt.storage.savedTask.Status != tt.status {
				t.Errorf(
					"savedTask.Status = %q, want %q",
					tt.storage.savedTask.Status,
					tt.status,
				)
			}

			if tt.storage.savedTask.CreatedAt.IsZero() {
				t.Error("savedTask.CreatedAt is zero")
			}

			if got.ID != tt.wantID {
				t.Errorf(
					"CreateTask() ID = %d, want %d",
					got.ID,
					tt.wantID,
				)
			}
		})
	}
}

func TestGetTaskByID(t *testing.T) {
	ctx := context.Background()

	tasks := []struct {
		name    string
		storage fakeStorage
		id      int
		wantID  int
		wantErr error
	}{
		{
			name: "success",
			storage: fakeStorage{
				readByIDResult: domain.Task{
					ID:     1,
					Title:  "learn to Go",
					Status: "active",
				},
			},
			id:      1,
			wantID:  1,
			wantErr: nil,
		},
		{
			name: "not found",
			storage: fakeStorage{
				readByIDErr: storage.ErrNotFound,
			},
			id:      99,
			wantErr: ErrTaskNotFound,
		},
		{
			name: "repository_error",
			storage: fakeStorage{
				readByIDErr: testErr,
			},
			id:      5,
			wantErr: testErr,
		},
	}

	for _, tt := range tasks {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(&tt.storage)

			got, err := service.GetTaskByID(ctx, tt.id)

			if tt.storage.readByIDArg != tt.id {
				t.Errorf("readByIDArg = %d, want = %d", tt.storage.readByIDArg, tt.id)
			}

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("GetTaskByID() error = %v, want = %v", err, tt.wantErr)
				}

				if !errors.Is(err, tt.wantErr) {
					t.Errorf("GetTaskByID() error = %v, want = %v", err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Fatalf("GetTaskByID() error = %v, want = %v", err, tt.wantErr)
			}

			if got.ID != tt.wantID {
				t.Errorf(
					"GetTaskByID() ID = %d, want %d",
					got.ID,
					tt.wantID,
				)
			}
		})
	}
}

func TestGetTasks(t *testing.T) {
	ctx := context.Background()

	tasks := []struct {
		name    string
		storage fakeStorage
		wantErr error
	}{
		{
			name: "success",
			storage: fakeStorage{
				readResult: []domain.Task{
					{
						ID:     1,
						Title:  "learn to Go 1",
						Status: "active",
					},
					{
						ID:     2,
						Title:  "learn to Go 2",
						Status: "passive",
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "repository_error",
			storage: fakeStorage{
				readErr: testErr,
			},
			wantErr: testErr,
		},
	}

	for _, tt := range tasks {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(&tt.storage)

			got, err := service.GetTasks(ctx)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("GetTasks() error = %v, want %v", err, tt.wantErr)
				}

				if !errors.Is(err, tt.wantErr) {
					t.Errorf("GetTasks() error = %v, want %v", err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Fatalf("GetTasks() error = %v, want %v", err, tt.wantErr)
			}

			if len(got) != len(tt.storage.readResult) {
				t.Fatalf(
					"GetTasks() len = %d, want %d",
					len(got),
					len(tt.storage.readResult),
				)
			}

			for i := range got {
				if got[i] != tt.storage.readResult[i] {
					t.Errorf("index %d not compare: %v != %v", i, got[i], tt.storage.readResult[i])
				}
			}
		})
	}
}

func TestGetTasksContextCanceled(t *testing.T) {
	service := NewService(&fakeStorage{
		waitContext: true,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got, err := service.GetTasks(ctx)

	if got != nil {
		t.Errorf("GetTasks() result = %v, want nil", got)
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf(
			"GetTasks() error = %v, want context.Canceled",
			err,
		)
	}
}

func TestGetTasksContextTimeout(t *testing.T) {
	service := NewService(&fakeStorage{
		waitContext: true,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	got, err := service.GetTasks(ctx)
	if got != nil {
		t.Errorf("GetTasks() result = %v, want nil", got)
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf(
			"GetTasks() error = %v, want context.DeadlineExceeded",
			err,
		)
	}
}
