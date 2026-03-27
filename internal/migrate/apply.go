package migrate

import (
    "fmt"
    "os"
    "path/filepath"
    "sort"
    "strings"

    "github.com/jmoiron/sqlx"
)

// Apply runs all *.up.sql files in the given directory against the provided DB.
// Files are applied in lexical order, which works with numeric prefixes
// like 001_, 002_, etc.
func Apply(db *sqlx.DB, dir string) error {
    entries, err := os.ReadDir(dir)
    if err != nil {
        return fmt.Errorf("read migrations dir: %w", err)
    }

    var files []string
    for _, e := range entries {
        if e.IsDir() {
            continue
        }
        name := e.Name()
        if strings.HasSuffix(name, ".up.sql") {
            files = append(files, name)
        }
    }

    sort.Strings(files)

    for _, name := range files {
        path := filepath.Join(dir, name)
        sqlBytes, err := os.ReadFile(path)
        if err != nil {
            return fmt.Errorf("read %s: %w", path, err)
        }
        if len(strings.TrimSpace(string(sqlBytes))) == 0 {
            continue
        }
        if _, err := db.Exec(string(sqlBytes)); err != nil {
            return fmt.Errorf("apply %s: %w", path, err)
        }
    }

    return nil
}
