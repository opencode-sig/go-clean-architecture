package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kun/zhisuo-server/internal/port"
)

func TestMemoryGetMiss(t *testing.T) {
	c := NewMemory(time.Minute)

	if _, err := c.Get(context.Background(), "missing"); !errors.Is(err, port.ErrCacheMiss) {
		t.Fatalf("expected ErrCacheMiss, got %v", err)
	}
}

func TestMemorySetGetDelete(t *testing.T) {
	c := NewMemory(time.Minute)
	ctx := context.Background()

	if err := c.Set(ctx, "k", []byte("v"), time.Minute); err != nil {
		t.Fatal(err)
	}

	got, err := c.Get(ctx, "k")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if string(got) != "v" {
		t.Fatalf("got %q, want %q", got, "v")
	}

	if err := c.Del(ctx, "k"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Get(ctx, "k"); !errors.Is(err, port.ErrCacheMiss) {
		t.Fatalf("expected miss after delete, got %v", err)
	}
}

func TestMemoryExpiry(t *testing.T) {
	c := NewMemory(time.Minute)
	ctx := context.Background()

	if err := c.Set(ctx, "k", []byte("v"), 10*time.Millisecond); err != nil {
		t.Fatal(err)
	}

	got, err := c.Get(ctx, "k")
	if err != nil || string(got) != "v" {
		t.Fatalf("expected value within TTL: got %q err %v", got, err)
	}

	time.Sleep(15 * time.Millisecond)
	if _, err := c.Get(ctx, "k"); !errors.Is(err, port.ErrCacheMiss) {
		t.Fatalf("expected miss after expiry, got %v", err)
	}
}

func TestMemoryDefaultTTL(t *testing.T) {
	c := NewMemory(5 * time.Millisecond)
	ctx := context.Background()

	if err := c.Set(ctx, "k", []byte("v"), 0); err != nil {
		t.Fatal(err)
	}

	time.Sleep(10 * time.Millisecond)
	if _, err := c.Get(ctx, "k"); !errors.Is(err, port.ErrCacheMiss) {
		t.Fatalf("expected default TTL applied, got %v", err)
	}
}

func TestMemoryNoTTLneverExpires(t *testing.T) {
	c := NewMemory(0) // default TTL disabled
	ctx := context.Background()

	if err := c.Set(ctx, "k", []byte("v"), 0); err != nil {
		t.Fatal(err)
	}

	time.Sleep(10 * time.Millisecond)
	if _, err := c.Get(ctx, "k"); err != nil {
		t.Fatalf("expected value to persist, got %v", err)
	}
}

func TestMemoryDeleteByPrefix(t *testing.T) {
	c := NewMemory(time.Minute)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		if err := c.Set(ctx, "article:list:"+string(rune(i)), []byte("x"), time.Minute); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.Set(ctx, "other:key", []byte("y"), time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteByPrefix(ctx, "article:list:"); err != nil {
		t.Fatal(err)
	}
	if len(c.items) != 1 {
		t.Fatalf("expected only non-matching key to remain, got %d items", len(c.items))
	}
}
