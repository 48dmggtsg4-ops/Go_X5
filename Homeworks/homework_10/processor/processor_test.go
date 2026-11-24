package processor

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAnalyzeFile(t *testing.T) {
	path := filepath.Join("..", "testdata", "orders-a.txt")
	got := analyzeFile(path)
	if got.Err != nil || got.Path != path || got.Lines != 3 || got.Words != 5 {
		t.Fatalf("результат = %+v, ожидаем этот путь, 3 строки и 5 слов", got)
	}
}

func TestProcessFiles(t *testing.T) {
	paths := []string{
		filepath.Join("..", "testdata", "orders-a.txt"),
		filepath.Join("..", "testdata", "orders-b.txt"),
		filepath.Join("..", "testdata", "empty.txt"),
	}
	want := map[string][2]int{
		paths[0]: {3, 5},
		paths[1]: {2, 4},
		paths[2]: {0, 0},
	}

	for got := range ProcessFiles(paths, 2) {
		stats, found := want[got.Path]
		if !found {
			t.Fatalf("лишний результат для %q", got.Path)
		}
		if got.Err != nil || got.Lines != stats[0] || got.Words != stats[1] {
			t.Fatalf("для %q получили %+v, ожидаем строк и слов %v", got.Path, got, stats)
		}
		delete(want, got.Path)
	}
	if len(want) > 0 {
		t.Fatalf("не получены результаты: %v", want)
	}
}

func TestProcessFilesDuplicateAndMissing(t *testing.T) {
	existing := filepath.Join("..", "testdata", "orders-a.txt")
	missing := filepath.Join(t.TempDir(), "missing.txt")
	paths := []string{existing, missing, existing}
	counts := make(map[string]int)

	for got := range ProcessFiles(paths, 0) {
		counts[got.Path]++
		switch got.Path {
		case existing:
			if got.Err != nil || got.Lines != 3 || got.Words != 5 {
				t.Errorf("неверный результат для существующего файла: %+v", got)
			}
		case missing:
			if !errors.Is(got.Err, os.ErrNotExist) {
				t.Errorf("ожидали ошибку отсутствующего файла, получили %+v", got)
			}
		default:
			t.Errorf("лишний результат: %+v", got)
		}
	}

	if counts[existing] != 2 || counts[missing] != 1 {
		t.Errorf("число результатов по путям = %v, ожидали два и один", counts)
	}
}

func TestProcessFilesEmpty(t *testing.T) {
	select {
	case _, ok := <-ProcessFiles(nil, 2):
		if ok {
			t.Fatal("для пустого списка получен результат")
		}
	case <-time.After(time.Second):
		t.Fatal("канал результатов не закрылся")
	}
}
