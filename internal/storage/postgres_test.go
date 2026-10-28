package storage

import (
	"context"
	"errors"
	"pgx-practice/internal/domain"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSave(t *testing.T) {
	ctx := context.Background()

	tasks := NewDB()
	ids := make(chan int)

	var readWg sync.WaitGroup
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			task, err := tasks.Save(ctx, domain.Task{
				Title:  "learn to Go",
				Status: "active",
			})
			if err != nil {
				t.Errorf("Save() unexpected error: %v", err)
				return
			}

			ids <- task.ID
		}()
	}

	for i := 0; i < 150; i++ {
		readWg.Add(1)
		go func() {
			defer readWg.Done()
			if _, err := tasks.Read(ctx); err != nil {
				t.Errorf("Read() unexpected error: %v", err)
			}
		}()
	}

	for i := 0; i < 100; i++ {
		readWg.Add(1)
		go func() {
			defer readWg.Done()

			id := <-ids

			if _, err := tasks.ReadByID(ctx, id); err != nil {
				t.Errorf("ReadByID() unexpected error: %v", err)
			}
		}()
	}

	wg.Wait()
	readWg.Wait()

	got, err := tasks.Read(ctx)
	if err != nil {
		t.Fatalf("Read() unexpected error: %v", err)
	}

	if len(got) != 100 {
		t.Errorf(
			"len(tasks) = %d, want 100",
			len(got),
		)
	}

	unique_ids := make(map[int]int)
	for i := range got {
		unique_ids[got[i].ID]++
	}

	if len(unique_ids) != 100 {
		t.Errorf("len(unique_ids) != len(tasks)")
	}
}

func SetupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper() // говорит компилятору, что это вспомогательная функция

	ctx := context.Background()

	test_pool, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/tasks_test")

	if err != nil {
		t.Fatalf("connect to postgres: %v", err)
	}

	if err = test_pool.Ping(ctx); err != nil {
		t.Fatalf("ping postgres: %v", err)
	}

	t.Cleanup(func() {
		defer test_pool.Close()
	})

	return test_pool
}

func TestRepository_Save(t *testing.T) {
	ctx := context.Background()

	test_pool := SetupTestDB(t)

	repo := NewRepository(test_pool)

	_, err := repo.db.Exec(ctx, `TRUNCATE tasks, users RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatal("error delete table")
	}

	user, err := repo.SaveUser(ctx, "vanyalox")
	if err != nil {
		t.Fatal("error create user")
	}

	task := domain.Task{UserID: user.ID, Title: "learn to Go", Status: "active"}

	tt, err := repo.Save(ctx, task)
	if err != nil {
		t.Fatal("error")
	}

	if tt.ID <= 0 {
		t.Errorf("error id")
	}

	if tt.Title != task.Title {
		t.Errorf("error title")
	}

	if tt.Status != task.Status {
		t.Errorf("error status")
	}

	if tt.UserID != task.UserID {
		t.Errorf("error user id")
	}

	if tt.CreatedAt.IsZero() {
		t.Errorf("error created time")
	}
}

func TestRepository_ReadByID(t *testing.T) {
	ctx := context.Background()

	test_pool := SetupTestDB(t)

	repo := NewRepository(test_pool)

	_, err := repo.db.Exec(ctx, `TRUNCATE tasks, users RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatal("error")
	}

	user, err := repo.SaveUser(ctx, "vanyalox")
	if err != nil {
		t.Fatal("error create user")
	}

	task := domain.Task{UserID: user.ID, Title: "learn to Go", Status: "active"}

	tt, err := repo.Save(ctx, task)
	if err != nil {
		t.Fatal("error")
	}

	ts, err := repo.ReadByID(ctx, tt.ID, tt.UserID)
	if err != nil {
		t.Fatal("error")
	}

	if errors.Is(err, ErrNotFound) {
		t.Errorf("error not found")
	}

	if tt.Title != ts.Title {
		t.Errorf("error title")
	}

	if tt.Status != ts.Status {
		t.Errorf("error status")
	}

	if ts.UserID != tt.UserID {
		t.Errorf("wrong user id")
	}

	if ts.CreatedAt.IsZero() {
		t.Errorf("created_at is zero")
	}

	if tt.ID != ts.ID {
		t.Errorf("error id")
	}
}

func TestRepository_Read(t *testing.T) {
	ctx := context.Background()

	test_pool := SetupTestDB(t)

	repo := NewRepository(test_pool)

	_, err := repo.db.Exec(ctx, `TRUNCATE tasks, users RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatal("error")
	}

	user, err := repo.SaveUser(ctx, "vanyalox")
	if err != nil {
		t.Fatal("error create user")
	}

	users, err := repo.Read(ctx, user.ID)
	if err != nil {
		t.Fatal("error read")
	}

	if users == nil {
		t.Errorf("error read")
	}

}
