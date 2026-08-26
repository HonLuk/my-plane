// Package output renders Plane responses as JSON or terminal-friendly tables.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// Renderer owns output destinations so command handlers do not depend on
// process-global stdout/stderr state. Tests can therefore capture output
// without changing the behavior of the executable.
type Renderer struct {
	Out   io.Writer
	Err   io.Writer
	color bool
}

// NewRenderer enables ANSI colors only for a terminal when NO_COLOR and TERM
// do not explicitly disable them.
func NewRenderer(out, errWriter io.Writer) *Renderer {
	return &Renderer{
		Out:   out,
		Err:   errWriter,
		color: colorEnabled(out),
	}
}

// NewPlainRenderer is useful for deterministic tests and non-terminal tools.
func NewPlainRenderer(out, errWriter io.Writer) *Renderer {
	return &Renderer{Out: out, Err: errWriter}
}

func colorEnabled(writer io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	file, ok := writer.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func (r *Renderer) colorize(code, text string) string {
	if !r.color {
		return text
	}
	return fmt.Sprintf("\033[%sm%s\033[0m", code, text)
}

func (r *Renderer) Dim(text string) string    { return r.colorize("2", text) }
func (r *Renderer) Bold(text string) string   { return r.colorize("1", text) }
func (r *Renderer) Red(text string) string    { return r.colorize("31", text) }
func (r *Renderer) Green(text string) string  { return r.colorize("32", text) }
func (r *Renderer) Yellow(text string) string { return r.colorize("33", text) }
func (r *Renderer) Cyan(text string) string   { return r.colorize("36", text) }

func (r *Renderer) Println(args ...any) { _, _ = fmt.Fprintln(r.Out, args...) }
func (r *Renderer) Errorln(args ...any) { _, _ = fmt.Fprintln(r.Err, args...) }
func (r *Renderer) Errorf(format string, args ...any) {
	_, _ = fmt.Fprintf(r.Err, format, args...)
}

// Format routes a decoded API value to JSON or the same heuristic table views
// used by the original CLI.
func (r *Renderer) Format(data any, format string, showPagination bool) {
	if format == "json" {
		encoded, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			r.Errorln(r.Red(fmt.Sprintf("Could not encode JSON output: %v", err)))
			return
		}
		r.Println(string(encoded))
		return
	}

	if object, ok := data.(map[string]any); ok {
		pagination := paginationInfo(object)
		if results, exists := object["results"]; exists {
			if showPagination && pagination != nil {
				r.printPagination(pagination)
			}
			data = results
		}
	}

	switch typed := data.(type) {
	case []any:
		r.printList(typed)
	case map[string]any:
		r.printObject(typed)
	case nil:
		return
	default:
		r.Println(resolve(typed))
	}
}

type pagination struct {
	nextCursor   any
	prevCursor   any
	totalResults any
	totalPages   any
	count        any
}

func paginationInfo(object map[string]any) *pagination {
	if !truthy(object["next_cursor"]) && !truthy(object["prev_cursor"]) {
		return nil
	}
	return &pagination{
		nextCursor:   object["next_cursor"],
		prevCursor:   object["prev_cursor"],
		totalResults: object["total_results"],
		totalPages:   object["total_pages"],
		count:        object["count"],
	}
}

func (r *Renderer) printPagination(info *pagination) {
	parts := make([]string, 0, 3)
	if truthy(info.totalResults) {
		parts = append(parts, fmt.Sprintf("total: %s", resolve(info.totalResults)))
	}
	if truthy(info.totalPages) {
		parts = append(parts, fmt.Sprintf("pages: %s", resolve(info.totalPages)))
	}
	if truthy(info.count) {
		parts = append(parts, fmt.Sprintf("showing: %s", resolve(info.count)))
	}
	if len(parts) > 0 {
		r.Println(r.Dim("Pagination: " + strings.Join(parts, " | ")))
	}
	if truthy(info.nextCursor) {
		r.Println(r.Dim(fmt.Sprintf("Next page: --cursor %s", resolve(info.nextCursor))))
	}
	if truthy(info.prevCursor) {
		r.Println(r.Dim(fmt.Sprintf("Prev page: --cursor %s", resolve(info.prevCursor))))
	}
	r.Println()
}

func (r *Renderer) printList(rows []any) {
	if len(rows) == 0 {
		r.Println(r.Dim("No results."))
		return
	}
	sample, _ := rows[0].(map[string]any)
	switch {
	case has(sample, "name") && has(sample, "identifier"):
		r.printTable(rows, []column{{"identifier", "ID", 10}, {"name", "NAME", 40}, {"id", "UUID", 36}})
	case has(sample, "name") && has(sample, "priority"):
		r.printTable(rows, []column{{"sequence_id", "SEQ", 8}, {"name", "NAME", 50}, {"priority", "PRI", 8}, {"state", "STATE", 20}, {"id", "UUID", 36}})
	case has(sample, "name") && has(sample, "start_date"):
		r.printTable(rows, []column{{"name", "NAME", 40}, {"start_date", "START", 12}, {"end_date", "END", 12}, {"id", "UUID", 36}})
	case has(sample, "display_name"):
		r.printTable(rows, []column{{"display_name", "NAME", 30}, {"email", "EMAIL", 40}, {"role", "ROLE", 10}, {"id", "UUID", 36}})
	default:
		keys := []string{"id", "identifier", "name", "title", "state", "priority", "sequence_id"}
		columns := make([]column, 0, len(keys))
		for _, key := range keys {
			if has(sample, key) {
				columns = append(columns, column{key, strings.ToUpper(key), 40})
			}
		}
		if len(columns) > 0 {
			r.printTable(rows, columns)
			return
		}
		for _, item := range rows {
			encoded, _ := json.MarshalIndent(item, "", "  ")
			r.Println(string(encoded))
		}
	}
}

type column struct {
	key    string
	header string
	max    int
}

func (r *Renderer) printTable(rows []any, columns []column) {
	widths := make([]int, len(columns))
	for index, item := range columns {
		widths[index] = runeLen(item.header)
	}
	cells := make([][]string, len(rows))
	rawCells := make([][]string, len(rows))
	for rowIndex, rawRow := range rows {
		row, _ := rawRow.(map[string]any)
		cells[rowIndex] = make([]string, len(columns))
		rawCells[rowIndex] = make([]string, len(columns))
		for columnIndex, item := range columns {
			raw := resolve(row[item.key])
			value := raw
			if item.key == "priority" {
				value = priorityString(row[item.key], r)
				raw = priorityLabel(row[item.key])
			} else {
				value = truncate(raw, item.max)
				raw = value
			}
			cells[rowIndex][columnIndex] = value
			rawCells[rowIndex][columnIndex] = raw
			if runeLen(raw) > widths[columnIndex] {
				widths[columnIndex] = runeLen(raw)
			}
		}
	}
	for index, item := range columns {
		if widths[index] > item.max {
			widths[index] = item.max
		}
	}

	var header strings.Builder
	for index, item := range columns {
		if index > 0 {
			header.WriteString("  ")
		}
		header.WriteString(r.Bold(padRight(item.header, widths[index])))
	}
	r.Println(header.String())
	r.Println(r.Dim(strings.Repeat("─", sum(widths)+2*(len(widths)-1))))
	for rowIndex := range cells {
		parts := make([]string, len(columns))
		for columnIndex := range columns {
			padding := widths[columnIndex] - runeLen(rawCells[rowIndex][columnIndex])
			if padding < 0 {
				padding = 0
			}
			parts[columnIndex] = cells[rowIndex][columnIndex] + strings.Repeat(" ", padding)
		}
		r.Println(strings.Join(parts, "  "))
	}
}

func (r *Renderer) printObject(object map[string]any) {
	maxKey := 0
	for key := range object {
		if !strings.HasPrefix(key, "_") && runeLen(key) > maxKey {
			maxKey = runeLen(key)
		}
	}
	keys := make([]string, 0, len(object))
	for key := range object {
		if !strings.HasPrefix(key, "_") {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		if strings.HasPrefix(key, "_") {
			continue
		}
		r.Println(fmt.Sprintf("  %s  %s", r.Bold(padRight(key, maxKey)), resolve(object[key])))
	}
}

func has(object map[string]any, key string) bool {
	_, ok := object[key]
	return ok
}

func truthy(value any) bool {
	if value == nil {
		return false
	}
	if number, ok := value.(json.Number); ok {
		if integer, err := number.Int64(); err == nil {
			return integer != 0
		}
		if decimal, err := strconv.ParseFloat(number.String(), 64); err == nil {
			return decimal != 0
		}
		return number.String() != ""
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String, reflect.Array, reflect.Slice, reflect.Map:
		return v.Len() > 0
	case reflect.Bool:
		return v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() != 0
	case reflect.Float32, reflect.Float64:
		return v.Float() != 0
	default:
		return true
	}
}

// Resolve pulls a human-readable value out of a nested Plane object.
func Resolve(value any) string { return resolve(value) }

func resolve(value any) string {
	if object, ok := value.(map[string]any); ok {
		for _, key := range []string{"name", "display_name", "id"} {
			if nested, exists := object[key]; exists {
				return resolve(nested)
			}
		}
		encoded, _ := json.Marshal(object)
		return string(encoded)
	}
	if value == nil {
		return "—"
	}
	if number, ok := value.(json.Number); ok {
		return number.String()
	}
	return fmt.Sprint(value)
}

func priorityLabel(value any) string {
	var number int
	switch typed := value.(type) {
	case json.Number:
		parsed, _ := typed.Int64()
		number = int(parsed)
	case int:
		number = typed
	case float64:
		number = int(typed)
	default:
		if value == nil || fmt.Sprint(value) == "" {
			return "none"
		}
		return fmt.Sprint(value)
	}
	if label, ok := map[int]string{0: "none", 1: "urgent", 2: "high", 3: "medium", 4: "low"}[number]; ok {
		return label
	}
	return fmt.Sprint(number)
}

func priorityString(value any, r *Renderer) string {
	label := priorityLabel(value)
	switch label {
	case "urgent":
		return r.colorize("1;31", label)
	case "high":
		return r.colorize("31", label)
	case "medium":
		return r.colorize("33", label)
	case "low":
		return r.colorize("34", label)
	case "none":
		return r.Dim(label)
	default:
		return label
	}
}

func truncate(value string, width int) string {
	if runeLen(value) <= width {
		return value
	}
	runes := []rune(value)
	if width <= 1 {
		return string(runes[:width])
	}
	return string(runes[:width-1]) + "…"
}

func runeLen(value string) int { return len([]rune(value)) }

func padRight(value string, width int) string {
	return value + strings.Repeat(" ", max(0, width-runeLen(value)))
}

func sum(values []int) int {
	total := 0
	for _, value := range values {
		total += value
	}
	return total
}

func max(first, second int) int {
	if first > second {
		return first
	}
	return second
}
