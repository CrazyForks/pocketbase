package dbutils

import (
	"regexp"
	"strings"

	"github.com/pocketbase/pocketbase/tools/tokenizer"
)

var (
	indexRegex       = regexp.MustCompile(`(?im)create\s+(unique\s+)?\s*index\s*(if\s+not\s+exists\s+)?([\w\"\'\[\]\.]*)\s+on\s+([\w\"\'\[\]\.]*)\s+\(([\s\S]*)\)(?:\s*where\s+([\s\S]*))?`)
	indexColumnRegex = regexp.MustCompile(`(?im)^([\s\S]+?)(?:\s+collate\s+([\w]+))?(?:\s+(asc|desc))?$`)
)

// IndexColumn represents a single parsed SQL index column.
type IndexColumn struct {
	Name    string `json:"name"` // identifier or expression
	Collate string `json:"collate"`
	Sort    string `json:"sort"`
}

// Index represents a single parsed SQL CREATE INDEX expression.
type Index struct {
	Unique     bool          `json:"unique"`
	Optional   bool          `json:"optional"`
	SchemaName string        `json:"schemaName"`
	IndexName  string        `json:"indexName"`
	TableName  string        `json:"tableName"`
	Columns    []IndexColumn `json:"columns"`
	Where      string        `json:"where"`
}

// IsValid checks if the current Index contains the minimum required fields to be considered valid.
func (idx Index) IsValid() bool {
	return idx.IndexName != "" && idx.TableName != "" && len(idx.Columns) > 0
}

// ParseIndex parses the provided `CREATE INDEX` SQL string into Index struct.
func ParseIndex(createIndexExpr string) Index {
	result := Index{}

	matches := indexRegex.FindStringSubmatch(createIndexExpr)
	if len(matches) != 7 {
		return result
	}

	trimChars := "`\"'[]\r\n\t\f\v "

	// Unique
	// ---
	result.Unique = strings.TrimSpace(matches[1]) != ""

	// Optional (aka. "IF NOT EXISTS")
	// ---
	result.Optional = strings.TrimSpace(matches[2]) != ""

	// SchemaName and IndexName
	// ---
	nameTk := tokenizer.NewFromString(matches[3])
	nameTk.Separators('.')

	nameParts, _ := nameTk.ScanAll()
	if len(nameParts) == 2 {
		result.SchemaName = strings.Trim(nameParts[0], trimChars)
		result.IndexName = strings.Trim(nameParts[1], trimChars)
	} else {
		result.IndexName = strings.Trim(nameParts[0], trimChars)
	}

	// TableName
	// ---
	result.TableName = strings.Trim(matches[4], trimChars)

	// Columns
	// ---
	columnsTk := tokenizer.NewFromString(matches[5])
	columnsTk.Separators(',')

	rawColumns, _ := columnsTk.ScanAll()

	result.Columns = make([]IndexColumn, 0, len(rawColumns))

	for _, col := range rawColumns {
		colMatches := indexColumnRegex.FindStringSubmatch(col)
		if len(colMatches) != 4 {
			continue
		}

		trimmedName := strings.Trim(colMatches[1], trimChars)
		if trimmedName == "" {
			continue
		}

		result.Columns = append(result.Columns, IndexColumn{
			Name:    trimmedName,
			Collate: strings.TrimSpace(colMatches[2]),
			Sort:    strings.ToUpper(colMatches[3]),
		})
	}

	// WHERE expression
	// ---
	result.Where = strings.TrimSpace(matches[6])

	return result
}