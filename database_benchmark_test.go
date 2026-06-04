package limen

import (
	"context"
	"fmt"
	"testing"
)

func newBenchmarkLimen(b *testing.B, userCount int) *Limen {
	b.Helper()
	l, err := New(&Config{
		BaseURL:  "http://localhost:8080",
		Database: newTestMemoryAdapter(b),
		Secret:   testSecret,
	})
	if err != nil {
		b.Fatalf("New: %v", err)
	}
	ctx := context.Background()
	for i := range userCount {
		if err := l.core.Create(ctx, l.core.Schema.User, &User{
			Email: fmt.Sprintf("user-%d@example.com", i),
		}, nil); err != nil {
			b.Fatalf("seed user: %v", err)
		}
	}
	return l
}

func BenchmarkDatabaseFindOneUser(b *testing.B) {
	l := newBenchmarkLimen(b, 100)
	ctx := context.Background()
	emailField := l.core.Schema.User.GetEmailField()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := l.core.FindOne(ctx, l.core.Schema.User, []Where{
			Eq(emailField, "user-42@example.com"),
		}, nil); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDatabaseFindManyUsers(b *testing.B) {
	l := newBenchmarkLimen(b, 100)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := l.core.FindMany(ctx, l.core.Schema.User, nil); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDatabaseCreateUser(b *testing.B) {
	l := newBenchmarkLimen(b, 0)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		if err := l.core.Create(ctx, l.core.Schema.User, &User{
			Email: fmt.Sprintf("bench-%d@example.com", i),
		}, nil); err != nil {
			b.Fatal(err)
		}
	}
}
