package locator

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
)

type (
	db     struct{ name string }
	users  struct{ db *db }
	mailer struct{ users *users }
)

func TestGetBuildsOnceResolvesDependencies(t *testing.T) {
	l := New()
	var built atomic.Int32
	Provide(l, func(*Locator) (*db, error) { built.Add(1); return &db{"main"}, nil })
	Provide(l, func(l *Locator) (*users, error) { return &users{db: Get[*db](l)}, nil })
	Provide(l, func(l *Locator) (*mailer, error) { return &mailer{users: Get[*users](l)}, nil })

	m := Get[*mailer](l)
	if m.users.db.name != "main" {
		t.Fatalf("dependency not wired: %+v", m)
	}
	if Get[*users](l) != m.users || Get[*db](l) != m.users.db {
		t.Error("Get must return the one instance")
	}
	if built.Load() != 1 {
		t.Errorf("db built %d times", built.Load())
	}
}

func TestSetIsAReadyValue(t *testing.T) {
	l := New()
	Set(l, &db{"fixed"})
	if got := Get[*db](l); got.name != "fixed" {
		t.Errorf("got %+v", got)
	}
}

func TestSetReplacesProviderBeforeItResolved(t *testing.T) {
	l := New()
	Provide(l, func(*Locator) (*db, error) { return &db{"real"}, nil })
	Set(l, &db{"fake"})
	if got := Get[*db](l); got.name != "fake" {
		t.Errorf("a test must be able to replace a service: %+v", got)
	}
}

func TestMissingServiceNamesTheType(t *testing.T) {
	l := New()
	_, err := TryGet[*db](l)
	if !errors.Is(err, ErrMissing) || !strings.Contains(err.Error(), "locator.db") {
		t.Fatalf("err %v", err)
	}
}

func TestGetPanicsWhenServiceCannotBuilt(t *testing.T) {
	l := New()
	defer func() {
		if recover() == nil {
			t.Error("Get of a missing service must panic: it is a wiring bug")
		}
	}()
	Get[*db](l)
}

func TestBuildErrorNamesRequestingPath(t *testing.T) {
	l := New()
	boom := errors.New("boom")
	Provide(l, func(*Locator) (*db, error) { return nil, boom })
	Provide(l, func(l *Locator) (*users, error) {
		d, err := TryGet[*db](l)
		return &users{db: d}, err
	})
	_, err := TryGet[*users](l)
	if !errors.Is(err, boom) || !strings.Contains(err.Error(), "users") || !strings.Contains(err.Error(), "db") {
		t.Fatalf("err %v", err)
	}
	// A failed build is not retried into a half-made service.
	if _, err := TryGet[*db](l); !errors.Is(err, boom) {
		t.Errorf("second try %v", err)
	}
}

func TestCycleIsErrorNotAHang(t *testing.T) {
	l := New()
	Provide(l, func(l *Locator) (*users, error) { _, err := TryGet[*mailer](l); return nil, err })
	Provide(l, func(l *Locator) (*mailer, error) { _, err := TryGet[*users](l); return nil, err })
	if _, err := TryGet[*users](l); !errors.Is(err, ErrCycle) {
		t.Fatalf("err %v", err)
	}
}

func TestBuildResolvesEverythingReportsFirstFailure(t *testing.T) {
	l := New()
	var built atomic.Int32
	Provide(l, func(*Locator) (*db, error) { built.Add(1); return &db{}, nil })
	if err := l.Build(); err != nil || built.Load() != 1 {
		t.Fatalf("Build: %v, built %d", err, built.Load())
	}

	bad := New()
	Provide(bad, func(*Locator) (*db, error) { return nil, errors.New("no file") })
	if err := bad.Build(); err == nil || !strings.Contains(err.Error(), "no file") {
		t.Fatalf("Build must surface a broken provider at start: %v", err)
	}
}

func TestCloseRunsReverseOrderOfConstruction(t *testing.T) {
	l := New()
	var order []string
	Provide(l, func(l *Locator) (*db, error) {
		OnClose(l, func(context.Context) error { order = append(order, "db"); return nil })
		return &db{}, nil
	})
	Provide(l, func(l *Locator) (*users, error) {
		d := Get[*db](l)
		OnClose(l, func(context.Context) error { order = append(order, "users"); return errors.New("late") })
		return &users{db: d}, nil
	})
	Get[*users](l)

	err := l.Close(context.Background())
	if len(order) != 2 || order[0] != "users" || order[1] != "db" {
		t.Errorf("order %v: what is built last uses what was built first, so it closes first", order)
	}
	if err == nil || !strings.Contains(err.Error(), "late") {
		t.Errorf("Close must still return the error: %v", err)
	}
	if l.Close(context.Background()) != nil || len(order) != 2 {
		t.Error("a second Close must do nothing")
	}
}

func TestProvideTwiceIsAnError(t *testing.T) {
	l := New()
	Provide(l, func(*Locator) (*db, error) { return &db{}, nil })
	defer func() {
		if recover() == nil {
			t.Error("two providers for one type is a wiring bug")
		}
	}()
	Provide(l, func(*Locator) (*db, error) { return &db{}, nil })
}
