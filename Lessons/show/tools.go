//go:build tools

package lesson9

// Сохраняет зависимости CLI в go.mod/go.sum. Файл исключён из обычной сборки.
import _ "github.com/swaggo/swag/cmd/swag"
